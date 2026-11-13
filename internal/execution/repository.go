package execution

// Repository observation and path containment. Every Git read uses a minimal,
// inherited-config-free environment, disables external diff drivers and
// optional locks, and is bounded. Nothing here mutates the repository. Digests
// are observations, not security boundaries.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxGitOutput      = 16 << 20
	maxGitStderr      = 64 << 10
	gitCommandTimeout = 60 * time.Second

	maxRelativePathBytes = 4096
	maxUntrackedFiles    = 20000
	maxUntrackedList     = 16 << 20
	untrackedFileLimit   = 2 << 20
	untrackedTotalLimit  = 32 << 20
)

// Repository is an opened Git worktree. It caches the canonical root, HEAD and
// branch observed at open time; Observe always re-reads current state.
type Repository struct {
	Root       string
	Head       string
	Branch     string
	GitVersion string

	git *gitRunner
}

// OpenRepository resolves dir to its canonical Git worktree root and verifies
// that HEAD resolves. A non-repository, a repository without commits, or an
// unreadable worktree fails closed.
func OpenRepository(dir string) (*Repository, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: repository path is invalid", ErrUsage)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: repository path is not a readable directory", ErrUsage)
	}
	probe := &gitRunner{root: abs}
	out, err := probe.run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("%w: not a Git worktree", ErrUsage)
	}
	root := strings.TrimSpace(string(out))
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("%w: Git worktree root is unreadable", ErrUsage)
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	resolvedAbs, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("%w: repository path is unreadable", ErrUsage)
	}
	if !withinPath(resolvedRoot, filepath.Clean(resolvedAbs)) {
		return nil, fmt.Errorf("%w: path is outside the Git worktree", ErrUsage)
	}
	git := &gitRunner{root: resolvedRoot}
	versionOut, err := git.run("--version")
	if err != nil {
		return nil, fmt.Errorf("Git is unavailable")
	}
	headOut, err := git.run("rev-parse", "--verify", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("%w: repository has no commit (unborn HEAD)", ErrUsage)
	}
	repository := &Repository{
		Root:       resolvedRoot,
		Head:       strings.TrimSpace(string(headOut)),
		GitVersion: strings.TrimSpace(string(versionOut)),
		git:        git,
	}
	if branchOut, branchErr := git.run("symbolic-ref", "--quiet", "--short", "HEAD"); branchErr == nil {
		repository.Branch = strings.TrimSpace(string(branchOut))
	}
	return repository, nil
}

// Observe records the current HEAD, branch, status, diff and untracked
// fingerprints. Ignored files are excluded. It performs no write to the
// repository.
func (r *Repository) Observe() (GitObservation, error) {
	return r.observe(nil)
}

// ObserveExcluding observes the worktree while excluding repository-relative
// paths from the status, diff and untracked fingerprints. It is used so that
// canonical source edits are judged by their own digests rather than as generic
// worktree drift.
func (r *Repository) ObserveExcluding(exclude []string) (GitObservation, error) {
	for _, relative := range exclude {
		if err := ValidateRelativePath(relative); err != nil {
			return GitObservation{}, err
		}
	}
	return r.observe(exclude)
}

func (r *Repository) observe(exclude []string) (GitObservation, error) {
	if r == nil || r.git == nil {
		return GitObservation{}, fmt.Errorf("%w: repository is not open", ErrUsage)
	}
	headOut, err := r.git.run("rev-parse", "--verify", "HEAD")
	if err != nil {
		return GitObservation{}, fmt.Errorf("%w: cannot resolve HEAD", ErrUsage)
	}
	head := strings.TrimSpace(string(headOut))
	branch := ""
	if branchOut, branchErr := r.git.run("symbolic-ref", "--quiet", "--short", "HEAD"); branchErr == nil {
		branch = strings.TrimSpace(string(branchOut))
	}
	pathspec := excludePathspec(exclude)
	statusArgs := append([]string{"status", "--porcelain=v1", "-z", "--untracked-files=normal"}, pathspec...)
	statusDigest, err := r.git.digest(statusArgs...)
	if err != nil {
		return GitObservation{}, fmt.Errorf("%w: cannot observe Git status", ErrUsage)
	}
	diffArgs := append([]string{"diff", "HEAD", "--binary", "--no-ext-diff", "--no-textconv", "--no-color"}, pathspec...)
	diffDigest, err := r.git.digest(diffArgs...)
	if err != nil {
		return GitObservation{}, fmt.Errorf("%w: cannot observe Git diff", ErrUsage)
	}
	untrackedDigest, err := r.untrackedDigest(exclude)
	if err != nil {
		return GitObservation{}, err
	}
	h := sha256.New()
	hashField(h, "worktree-v1")
	for _, relative := range exclude {
		hashField(h, "exclude")
		hashField(h, relative)
	}
	hashField(h, head)
	hashField(h, statusDigest)
	hashField(h, diffDigest)
	hashField(h, untrackedDigest)
	return GitObservation{
		Head:            head,
		Branch:          branch,
		StatusDigest:    statusDigest,
		DiffDigest:      diffDigest,
		UntrackedDigest: untrackedDigest,
		WorktreeDigest:  digestPrefix(h),
		GitVersion:      r.GitVersion,
		ObservedAt:      time.Now().UTC(),
	}, nil
}

// excludePathspec builds an exclusion-only pathspec. Git treats a pathspec with
// only exclusions as the whole tree minus the excluded paths.
func excludePathspec(exclude []string) []string {
	if len(exclude) == 0 {
		return nil
	}
	pathspec := make([]string, 0, len(exclude)+1)
	pathspec = append(pathspec, "--")
	for _, relative := range exclude {
		pathspec = append(pathspec, ":(exclude,literal)"+relative)
	}
	return pathspec
}

// ObserveScope observes the worktree with the scope's canonical sources
// excluded from the generic fingerprint. Their edits are judged by their own
// digests instead.
func ObserveScope(repo *Repository, scope Scope) (GitObservation, error) {
	if repo == nil {
		return GitObservation{}, fmt.Errorf("%w: repository is required", ErrUsage)
	}
	return repo.ObserveExcluding(scopeExclusions(scope))
}

// ObserveRun observes the worktree for a run, excluding the canonical sources
// and the run's own (possibly Git-visible) scratch directory so a run never
// reports itself as project drift.
func ObserveRun(repo *Repository, scope Scope, runDir string) (GitObservation, error) {
	if repo == nil {
		return GitObservation{}, fmt.Errorf("%w: repository is required", ErrUsage)
	}
	return repo.ObserveExcluding(observationExclusions(scope, runDir))
}

// untrackedDigest hashes sorted untracked paths with bounded content: regular
// files up to the per-file and total budgets contribute bytes, larger files
// contribute path, size and modification time, and symlinks are recorded by
// target without being followed.
func (r *Repository) untrackedDigest(exclude []string) (string, error) {
	listArgs := append([]string{"ls-files", "-z", "--others", "--exclude-standard"}, excludePathspec(exclude)...)
	out, err := r.git.run(listArgs...)
	if err != nil {
		return "", fmt.Errorf("%w: cannot list untracked files", ErrUsage)
	}
	if len(out) > maxUntrackedList {
		return "", fmt.Errorf("%w: untracked file list exceeds bound", ErrUsage)
	}
	names := make([]string, 0, 64)
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		names = append(names, string(raw))
	}
	if len(names) > maxUntrackedFiles {
		return "", fmt.Errorf("%w: untracked file count exceeds bound", ErrUsage)
	}
	sort.Strings(names)
	h := sha256.New()
	hashField(h, "untracked-v1")
	budget := int64(untrackedTotalLimit)
	for _, name := range names {
		hashField(h, name)
		full := filepath.Join(r.Root, filepath.FromSlash(name))
		info, statErr := os.Lstat(full)
		if statErr != nil {
			hashField(h, "missing")
			continue
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, readErr := os.Readlink(full)
			if readErr != nil {
				hashField(h, "symlink-unreadable")
				continue
			}
			hashField(h, "symlink")
			hashField(h, target)
		case info.Mode().IsRegular():
			if info.Size() <= untrackedFileLimit && info.Size() <= budget {
				data, readErr := os.ReadFile(full)
				if readErr != nil {
					hashField(h, "unreadable")
					continue
				}
				budget -= int64(len(data))
				hashField(h, "content")
				hashField(h, DigestBytes(data))
				continue
			}
			hashField(h, "metadata")
			hashField(h, strconv.FormatInt(info.Size(), 10))
			hashField(h, strconv.FormatInt(info.ModTime().UnixNano(), 10))
		default:
			hashField(h, "other")
		}
	}
	return digestPrefix(h), nil
}

// FeatureScope resolves a feature directory and both canonical source files,
// then records their full-file digests and the normalized selected-batch scope
// digest. Symlinks, paths outside the repository and missing sources fail
// closed. The result is a reference, never an approval.
func FeatureScope(repo *Repository, featureDir, batchID string) (Scope, error) {
	if repo == nil || repo.git == nil {
		return Scope{}, fmt.Errorf("%w: repository is required", ErrUsage)
	}
	if err := validateToken("batch ID", batchID); err != nil {
		return Scope{}, err
	}
	dir, rel, err := resolveInside(repo.Root, featureDir)
	if err != nil {
		return Scope{}, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Scope{}, fmt.Errorf("%w: feature directory is not a regular directory", ErrUsage)
	}
	specPath := filepath.Join(dir, "spec.md")
	tasksPath := filepath.Join(dir, "tasks.md")
	specData, err := readBoundedRegular(specPath, MaxSpecFileBytes)
	if err != nil {
		return Scope{}, fmt.Errorf("%w: feature spec.md is unavailable", ErrUsage)
	}
	tasksData, err := readBoundedRegular(tasksPath, MaxTasksFileBytes)
	if err != nil {
		return Scope{}, fmt.Errorf("%w: feature tasks.md is unavailable", ErrUsage)
	}
	if !utf8.Valid(specData) || !utf8.Valid(tasksData) {
		return Scope{}, fmt.Errorf("%w: feature sources must be valid UTF-8", ErrUsage)
	}
	batchDigest, err := NormalizedBatchScopeDigest(tasksData, batchID)
	if err != nil {
		return Scope{}, err
	}
	return Scope{
		Kind:             ScopeFeature,
		RepoRoot:         repo.Root,
		FeatureDir:       rel,
		SpecPath:         path.Join(rel, "spec.md"),
		SpecDigest:       DigestBytes(specData),
		TasksPath:        path.Join(rel, "tasks.md"),
		TasksDigest:      DigestBytes(tasksData),
		BatchID:          batchID,
		BatchScopeDigest: batchDigest,
	}, nil
}

// DirectScope builds a clearly identified direct-scope identity. It does not
// fabricate feature files or a batch.
func DirectScope(repo *Repository, label string) (Scope, error) {
	if repo == nil || repo.git == nil {
		return Scope{}, fmt.Errorf("%w: repository is required", ErrUsage)
	}
	if err := validateLabel(label); err != nil {
		return Scope{}, err
	}
	return Scope{Kind: ScopeDirect, RepoRoot: repo.Root, Label: label}, nil
}

// DigestBytes returns the sha256 digest of data in the package's stable form.
func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ValidateRelativePath rejects absolute paths, traversal, NUL, control
// characters, backslashes and non-canonical forms. Paths use forward slashes
// and are repository-relative.
func ValidateRelativePath(p string) error {
	if p == "" || len(p) > maxRelativePathBytes {
		return fmt.Errorf("%w: relative path is empty or too long", ErrUnsafePath)
	}
	if strings.ContainsRune(p, 0) || strings.ContainsAny(p, "\\") {
		return fmt.Errorf("%w: relative path contains a forbidden character", ErrUnsafePath)
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: relative path contains a control character", ErrUnsafePath)
		}
	}
	if path.IsAbs(p) || strings.HasPrefix(p, "/") {
		return fmt.Errorf("%w: absolute paths are refused", ErrUnsafePath)
	}
	if path.Clean(p) != p {
		return fmt.Errorf("%w: relative path is not canonical", ErrUnsafePath)
	}
	if p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return fmt.Errorf("%w: path traversal is refused", ErrUnsafePath)
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("%w: path traversal is refused", ErrUnsafePath)
		}
	}
	return nil
}

// withinPath reports whether target is root or inside root.
func withinPath(root, target string) bool {
	if root == "" || target == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// resolveInside resolves candidate (absolute or root-relative) to a clean
// absolute path inside root and refuses symlinked components. It returns the
// absolute path and the slash-relative form.
func resolveInside(root, candidate string) (string, string, error) {
	if strings.TrimSpace(candidate) == "" {
		return "", "", fmt.Errorf("%w: path is required", ErrUsage)
	}
	abs := candidate
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, abs)
	}
	abs = filepath.Clean(abs)
	if abs == filepath.Clean(root) || !withinPath(root, abs) {
		return "", "", fmt.Errorf("%w: path is outside the repository", ErrUnsafePath)
	}
	rel, err := filepath.Rel(filepath.Clean(root), abs)
	if err != nil {
		return "", "", fmt.Errorf("%w: path is outside the repository", ErrUnsafePath)
	}
	if err := ensureNoSymlink(root, rel); err != nil {
		return "", "", err
	}
	return abs, filepath.ToSlash(rel), nil
}

// ensureNoSymlink refuses any symlink component of rel below root. Missing
// components are allowed; they cannot yet be symlinks.
func ensureNoSymlink(root, rel string) error {
	current := filepath.Clean(root)
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w: path is unreadable", ErrUnsafePath)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlinked paths are refused", ErrUnsafePath)
		}
	}
	return nil
}

// readBoundedRegular reads a regular, non-symlink file no larger than limit.
// It re-checks the opened file identity so a swapped path is not read.
func readBoundedRegular(name string, limit int64) ([]byte, error) {
	before, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: not a regular file", ErrUnsafePath)
	}
	if before.Size() > limit {
		return nil, fmt.Errorf("%w: file exceeds the size bound", ErrUsage)
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%w: file changed while opening", ErrConflict)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: file exceeds the size bound", ErrUsage)
	}
	return data, nil
}

// validateToken requires a short, path-safe identifier.
func validateToken(name, value string) error {
	if value == "" || len(value) > 64 {
		return fmt.Errorf("%w: %s is required and must be at most 64 characters", ErrUsage, name)
	}
	for i, r := range value {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.'
		if !ok || i == 0 && (r == '-' || r == '_' || r == '.') {
			return fmt.Errorf("%w: %s is not a safe identifier", ErrUsage, name)
		}
	}
	return nil
}

// validateLabel bounds an optional direct-scope label and refuses control
// characters so it cannot distort terminal output.
func validateLabel(label string) error {
	if len(label) > 200 {
		return fmt.Errorf("%w: label is too long", ErrUsage)
	}
	if !utf8.ValidString(label) {
		return fmt.Errorf("%w: label must be valid UTF-8", ErrUsage)
	}
	for _, r := range label {
		if r == 0 || r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: label contains a control character", ErrUsage)
		}
	}
	return nil
}

// hashField writes a length-prefixed field so concatenation is unambiguous.
func hashField(h hash.Hash, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	h.Write(length[:])
	io.WriteString(h, value)
}

func digestPrefix(h hash.Hash) string {
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// gitRunner runs bounded, read-only Git commands with an isolated environment.
type gitRunner struct {
	root string
}

func (g *gitRunner) command(ctx context.Context, args ...string) *exec.Cmd {
	argv := append([]string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-C", g.root}, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Env = []string{
		"PATH=" + gitPath(),
		"HOME=" + os.DevNull,
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"LC_ALL=C",
	}
	return cmd
}

func (g *gitRunner) stream(w io.Writer, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	cmd := g.command(ctx, args...)
	cmd.Stdout = w
	stderr := &cappedBuffer{limit: maxGitStderr}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return gitCommandError(args, err)
	}
	return nil
}

func (g *gitRunner) run(args ...string) ([]byte, error) {
	out := &cappedBuffer{limit: maxGitOutput}
	if err := g.stream(out, args...); err != nil {
		return nil, err
	}
	if out.over {
		return nil, fmt.Errorf("Git output exceeds %d bytes", maxGitOutput)
	}
	return out.buf.Bytes(), nil
}

func (g *gitRunner) digest(args ...string) (string, error) {
	h := sha256.New()
	if err := g.stream(h, args...); err != nil {
		return "", err
	}
	return digestPrefix(h), nil
}

// runCode runs a Git command and returns its exit code without treating a
// ordinary non-zero status as an error.
func (g *gitRunner) runCode(args ...string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	cmd := g.command(ctx, args...)
	out := &cappedBuffer{limit: maxGitOutput}
	stderr := &cappedBuffer{limit: maxGitStderr}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return out.buf.Bytes(), 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return out.buf.Bytes(), exitErr.ExitCode(), nil
	}
	return nil, 0, gitCommandError(args, err)
}

func gitCommandError(args []string, err error) error {
	code := -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	name := "git"
	if len(args) > 0 {
		name = args[0]
	}
	return fmt.Errorf("Git command %s failed (exit %d)", name, code)
}

func gitPath() string {
	if p := os.Getenv("PATH"); p != "" {
		return p
	}
	return "/usr/bin:/bin"
}

// cappedBuffer keeps at most limit bytes and records whether more arrived.
type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.limit > 0 && c.buf.Len()+len(p) > c.limit {
		c.over = true
		remaining := c.limit - c.buf.Len()
		if remaining > 0 {
			c.buf.Write(p[:remaining])
		}
		return len(p), nil
	}
	return c.buf.Write(p)
}
