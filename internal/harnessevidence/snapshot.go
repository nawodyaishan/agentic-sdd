package harnessevidence

// Scoped workspace identity, compatible with the receipt v1 sha256-v1
// semantics: explicit unique scope paths; scoped HEAD, staged-index and
// untracked observations; worktree kind/mode/content records. The canonical
// record encoding reproduces the reference consumer's byte layout exactly, so
// a digest computed here equals the reference digest for the same workspace.
// Symlinks, submodules, unmerged indexes, special files and non-UTF-8 paths
// fail closed. Git observations use a minimal environment with no
// user-global/system config, no inherited index and no credential variables.
// This is not a concurrent-writer sandbox; content and manifests are never
// exported, only the digest.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxIdentityFiles = 10000
	maxIdentityBytes = 128 << 20
	maxFileBytes     = 32 << 20
	maxGitOutput     = 10 << 20
	gitTimeout       = 15 * time.Second
)

// ScopeIdentity is the observed identity of the selected scopes.
type ScopeIdentity struct {
	BaseRevision string `json:"base_revision"`
	Diff         string `json:"diff"`
}

// Snapshot measures the scoped identity of a Git workspace. Paths are exact,
// literal, repository-relative scopes; "." selects the whole repository.
// Ignored files are excluded unless explicitly named. Missing explicitly
// selected paths are recorded as missing, never silently omitted.
func Snapshot(workspace string, paths []string) (ScopeIdentity, error) {
	if err := validateScopes(paths); err != nil {
		return ScopeIdentity{}, err
	}
	root, err := resolveExisting(workspace)
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("workspace unavailable")
	}
	for _, p := range paths {
		if err := noSymlinkScope(root, p); err != nil {
			return ScopeIdentity{}, err
		}
	}
	git := func(args ...string) ([]byte, error) {
		return gitObserve(root, true, args...)
	}
	toplevel, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("identity Git observation unavailable")
	}
	topRoot, err := resolveExisting(strings.TrimRight(string(toplevel), "\n"))
	if err != nil || topRoot != root {
		return ScopeIdentity{}, fmt.Errorf("workspace must be the repository root")
	}
	base, err := git("rev-parse", "--verify", "HEAD")
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("identity Git observation unavailable")
	}
	pathspecs := append([]string{"--"}, paths...)
	head, err := git(append([]string{"ls-tree", "-r", "-z", "HEAD"}, pathspecs...)...)
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("identity Git observation unavailable")
	}
	index, err := git(append([]string{"ls-files", "--stage", "-z"}, pathspecs...)...)
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("identity Git observation unavailable")
	}
	others, err := git(append([]string{"ls-files", "--others", "--exclude-standard", "-z"}, pathspecs...)...)
	if err != nil {
		return ScopeIdentity{}, fmt.Errorf("identity Git observation unavailable")
	}

	names := map[string]bool{}
	records := [][]any{}
	for _, p := range paths {
		records = append(records, []any{"scope", p})
	}
	for _, entry := range []struct {
		kind   string
		raw    []byte
		staged bool
	}{
		{"HEAD", head, false},
		{"index", index, true},
	} {
		for _, item := range bytes.Split(entry.raw, []byte{0}) {
			if len(item) == 0 {
				continue
			}
			meta, name, ok := bytes.Cut(item, []byte("\t"))
			if !ok || !utf8.Valid(name) {
				return ScopeIdentity{}, fmt.Errorf("identity contains non-UTF-8 or malformed path")
			}
			fields := bytes.Fields(meta)
			if len(fields) == 0 || !bytes.Equal(fields[0], []byte("100644")) && !bytes.Equal(fields[0], []byte("100755")) {
				return ScopeIdentity{}, fmt.Errorf("identity contains unsupported Git file kind")
			}
			if entry.staged && !bytes.Equal(fields[len(fields)-1], []byte("0")) {
				return ScopeIdentity{}, fmt.Errorf("unmerged index cannot establish identity")
			}
			path := string(name)
			names[path] = true
			records = append(records, []any{entry.kind, path, string(meta)})
		}
	}
	for _, name := range bytes.Split(others, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		if !utf8.Valid(name) {
			return ScopeIdentity{}, fmt.Errorf("identity contains non-UTF-8 path")
		}
		names[string(name)] = true
	}
	if err := walkScopes(root, paths); err != nil {
		return ScopeIdentity{}, err
	}
	// Explicitly selected missing or non-directory paths stay part of the scope.
	for _, p := range paths {
		if p == "." {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(p))
		info, err := os.Stat(target)
		if err != nil || !info.IsDir() {
			names[p] = true
		}
	}
	if len(names) > maxIdentityFiles {
		return ScopeIdentity{}, fmt.Errorf("identity file count exceeds bound")
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	total := 0
	for _, name := range sorted {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := noSymlinkAncestors(root, target); err != nil {
			return ScopeIdentity{}, err
		}
		info, err := os.Lstat(target)
		if os.IsNotExist(err) {
			records = append(records, []any{"worktree", name, "missing"})
			continue
		}
		if err != nil {
			return ScopeIdentity{}, fmt.Errorf("identity observation unavailable")
		}
		if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
			return ScopeIdentity{}, fmt.Errorf("identity requires bounded regular files")
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return ScopeIdentity{}, fmt.Errorf("identity observation unavailable")
		}
		total += len(data)
		if total > maxIdentityBytes {
			return ScopeIdentity{}, fmt.Errorf("identity bytes exceed bound")
		}
		sum := sha256.Sum256(data)
		records = append(records, []any{"worktree", name, "file", fileMode(info), hex.EncodeToString(sum[:])})
	}
	payload, err := canonicalRecords(records)
	if err != nil {
		return ScopeIdentity{}, err
	}
	digest := sha256.Sum256(payload)
	return ScopeIdentity{
		BaseRevision: strings.TrimRight(string(base), "\n"),
		Diff:         "sha256-v1:" + hex.EncodeToString(digest[:]),
	}, nil
}

// fileMode is the portable permission/mode integer, including setuid, setgid
// and sticky bits, matching the reference mode observation.
func fileMode(info os.FileInfo) uint32 {
	mode := uint32(info.Mode().Perm())
	if info.Mode()&os.ModeSetuid != 0 {
		mode |= 0o4000
	}
	if info.Mode()&os.ModeSetgid != 0 {
		mode |= 0o2000
	}
	if info.Mode()&os.ModeSticky != 0 {
		mode |= 0o1000
	}
	return mode
}

// validateScopes checks the explicit unique literal scope list.
func validateScopes(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("explicit unique identity scope required")
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if p == "" || seen[p] {
			return fmt.Errorf("explicit unique identity scope required")
		}
		seen[p] = true
		if strings.ContainsAny(p, "\\*?[]:\x00\n\r") || strings.HasPrefix(p, "/") {
			return fmt.Errorf("unsafe identity scope")
		}
		if p != normalizeScope(p) {
			return fmt.Errorf("unsafe identity scope")
		}
		for _, part := range strings.Split(p, "/") {
			if part == ".." || part == ".git" {
				return fmt.Errorf("unsafe identity scope")
			}
		}
	}
	return nil
}

// normalizeScope mirrors PurePosixPath normalization: empty and "." parts are
// dropped, ".." parts are preserved so traversal stays detectable.
func normalizeScope(p string) string {
	parts := strings.Split(p, "/")
	kept := parts[:0]
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		kept = append(kept, part)
	}
	if len(kept) == 0 {
		return "."
	}
	return strings.Join(kept, "/")
}

func noSymlinkScope(root, scope string) error {
	current := root
	for _, part := range strings.Split(scope, "/") {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("identity scope symlink refused")
		}
	}
	return nil
}

func noSymlinkAncestors(root, target string) error {
	current := target
	for {
		if current == root {
			return nil
		}
		if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("identity symlink refused")
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

// walkScopes verifies every non-pruned directory entry under the scopes is a
// bounded regular file, refusing non-ignored special files and symlinks.
func walkScopes(root string, paths []string) error {
	seen := 0
	for _, scope := range paths {
		target := filepath.Join(root, filepath.FromSlash(scope))
		info, err := os.Lstat(target)
		if err != nil || !info.IsDir() {
			continue
		}
		walker := func(current string, entry os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("identity observation unavailable")
			}
			rel, relErr := filepath.Rel(root, current)
			if relErr != nil {
				return fmt.Errorf("identity observation unavailable")
			}
			rel = filepath.ToSlash(rel)
			if entry.IsDir() {
				if entry.Name() == ".git" {
					return filepath.SkipDir
				}
				isIgnored, ignoreErr := ignored(root, rel)
				if ignoreErr != nil {
					return ignoreErr
				}
				if isIgnored {
					return filepath.SkipDir
				}
				if entry.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("identity symlink refused")
				}
				return nil
			}
			seen++
			if seen > maxIdentityFiles {
				return fmt.Errorf("identity entry count exceeds bound")
			}
			if entry.Type().IsRegular() {
				return nil
			}
			isIgnored, ignoreErr := ignored(root, rel)
			if ignoreErr != nil {
				return ignoreErr
			}
			if isIgnored {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if info, err := os.Stat(current); err == nil && info.IsDir() {
					return fmt.Errorf("identity symlink refused")
				}
			}
			return fmt.Errorf("identity special file refused")
		}
		if err := filepath.WalkDir(target, walker); err != nil {
			return err
		}
	}
	return nil
}

// ignored reports Git's exclude-standard verdict for one literal path. Git's
// check-ignore does not support literal pathspec magic, so that flag is
// omitted here and pathspec-magic names fail closed. An unavailable
// observation is an error, never a silent include/exclude.
func ignored(root, rel string) (bool, error) {
	if strings.HasPrefix(rel, ":") || strings.ContainsAny(rel, "*?[]\\") {
		return false, fmt.Errorf("ignore observation unavailable")
	}
	out, err := gitObserve(root, false, "check-ignore", "-q", "--", rel)
	if err == nil {
		if len(out) != 0 {
			return false, fmt.Errorf("ignore observation unavailable")
		}
		return true, nil
	}
	var observation *gitObservationError
	if errors.As(err, &observation) && observation.code == 1 {
		return false, nil
	}
	return false, fmt.Errorf("ignore observation unavailable")
}

// gitObservationError carries a Git exit status, so callers can distinguish
// the documented "not ignored" status of check-ignore from a real failure.
type gitObservationError struct {
	code int
}

func (e *gitObservationError) Error() string {
	return "identity Git observation unavailable"
}

// gitObserve runs one bounded Git observation with a minimal environment.
// literal controls the global --literal-pathspecs flag: every pathspec-driven
// command uses it except check-ignore, which rejects literal pathspec magic.
func gitObserve(root string, literal bool, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	argv := []string{"git", "--no-optional-locks"}
	if literal {
		argv = append(argv, "--literal-pathspecs")
	}
	argv = append(argv, "-C", root)
	argv = append(argv, args...)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = []string{
		"PATH=" + gitPath(),
		"HOME=" + os.DevNull,
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		code := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
		return nil, &gitObservationError{code: code}
	}
	if len(out) > maxGitOutput {
		return nil, fmt.Errorf("identity Git observation unavailable")
	}
	return out, nil
}

func gitPath() string {
	if p := os.Getenv("PATH"); p != "" {
		return p
	}
	return "/usr/bin:/bin"
}

// canonicalRecords encodes identity records in the reference consumer's
// canonical form: compact ensure-ASCII JSON of the sorted records, where
// records compare element-wise (strings by code point, then integers).
func canonicalRecords(records [][]any) ([]byte, error) {
	sort.Slice(records, func(i, j int) bool { return lessRecord(records[i], records[j]) })
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, record := range records {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('[')
		for j, value := range record {
			if j > 0 {
				buf.WriteByte(',')
			}
			switch v := value.(type) {
			case string:
				buf.WriteString(asciiJSON(v))
			case uint32:
				buf.WriteString(strconv.FormatUint(uint64(v), 10))
			default:
				return nil, fmt.Errorf("identity record encoding unavailable")
			}
		}
		buf.WriteByte(']')
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func lessRecord(a, b []any) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		av, aok := a[i].(string)
		bv, bok := b[i].(string)
		if aok && bok {
			if av != bv {
				return av < bv
			}
			continue
		}
		ai, aok := a[i].(uint32)
		bi, bok := b[i].(uint32)
		if aok && bok {
			if ai != bi {
				return ai < bi
			}
			continue
		}
		// Mixed types at one position cannot occur in well-formed identity
		// records; fall back to type order to stay deterministic.
		return aok
	}
	return len(a) < len(b)
}

// asciiJSON quotes one string exactly like the reference ensure-ASCII JSON:
// short escapes for common controls, \uXXXX (lowercase) for everything else
// outside printable ASCII, including surrogate pairs for astral characters.
func asciiJSON(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		default:
			switch {
			case r >= 0x20 && r < 0x7f:
				buf.WriteRune(r)
			case r < 0x20 || (r >= 0x7f && r <= 0x9f):
				fmt.Fprintf(&buf, `\u%04x`, r)
			case r < 0x10000:
				fmt.Fprintf(&buf, `\u%04x`, r)
			default:
				r -= 0x10000
				hi := 0xd800 + (r >> 10)
				lo := 0xdc00 + (r & 0x3ff)
				fmt.Fprintf(&buf, `\u%04x\u%04x`, hi, lo)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
