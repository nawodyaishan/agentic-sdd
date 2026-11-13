package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ReviewPackageRequest configures a review package build.
type ReviewPackageRequest struct {
	RunDir        string
	RunID         string
	RepoRoot      string
	Base          string
	Head          string
	MaxFileBytes  int64
	MaxTotalBytes int64
}

// defaultSensitivePatterns lists the default screening patterns for sensitive files.
var defaultSensitivePatterns = []string{
	".env",
	".env.*",
	"*.pem",
	"*.key",
	"*.p12",
	"*.jks",
	"*.keystore",
	"id_rsa",
	"id_rsa.*",
	"id_ed25519",
	"id_ed25519.*",
	".netrc",
	".npmrc",
	"*.pypirc",
	"credentials*",
	"secrets*",
}

// isSensitivePath reports whether a repository-relative path matches default sensitive patterns.
func isSensitivePath(p string) bool {
	clean := path.Clean(filepath.ToSlash(p))
	base := path.Base(clean)
	baseLower := strings.ToLower(base)
	cleanLower := strings.ToLower(clean)

	for _, pat := range defaultSensitivePatterns {
		patLower := strings.ToLower(pat)
		if m, _ := path.Match(patLower, baseLower); m {
			return true
		}
		if m, _ := path.Match(patLower, cleanLower); m {
			return true
		}
	}
	if baseLower == ".env" || strings.HasPrefix(baseLower, ".env.") {
		return true
	}
	if strings.HasPrefix(baseLower, "credentials") || strings.HasPrefix(baseLower, "secrets") {
		return true
	}
	if baseLower == "id_rsa" || strings.HasPrefix(baseLower, "id_rsa.") || baseLower == "id_ed25519" || strings.HasPrefix(baseLower, "id_ed25519.") {
		return true
	}
	return false
}

// rawDiffRecord is one record from git diff --raw -z.
type rawDiffRecord struct {
	oldMode string
	newMode string
	oldSha  string
	newSha  string
	status  string
	path    string
	oldPath string
}

// parseRawDiff parses NUL-delimited git diff --raw -z output.
func parseRawDiff(data []byte) ([]rawDiffRecord, error) {
	if len(data) == 0 {
		return nil, nil
	}
	tokens := bytes.Split(data, []byte{0})
	var records []rawDiffRecord
	for i := 0; i < len(tokens); {
		token := string(tokens[i])
		if strings.TrimSpace(token) == "" {
			i++
			continue
		}
		if !strings.HasPrefix(token, ":") {
			i++
			continue
		}
		fields := strings.Fields(token)
		if len(fields) < 5 {
			i++
			continue
		}
		oldMode := strings.TrimPrefix(fields[0], ":")
		newMode := fields[1]
		oldSha := fields[2]
		newSha := fields[3]
		statusStr := fields[4]
		statusChar := statusStr[0]

		if statusChar == 'R' || statusChar == 'C' {
			if i+2 >= len(tokens) {
				break
			}
			oldPath := string(tokens[i+1])
			newPath := string(tokens[i+2])
			records = append(records, rawDiffRecord{
				oldMode: oldMode,
				newMode: newMode,
				oldSha:  oldSha,
				newSha:  newSha,
				status:  string(statusChar),
				path:    newPath,
				oldPath: oldPath,
			})
			i += 3
		} else {
			if i+1 >= len(tokens) {
				break
			}
			filePath := string(tokens[i+1])
			records = append(records, rawDiffRecord{
				oldMode: oldMode,
				newMode: newMode,
				oldSha:  oldSha,
				newSha:  newSha,
				status:  string(statusChar),
				path:    filePath,
			})
			i += 2
		}
	}
	return records, nil
}

// statusRecord is one entry parsed from git status --porcelain=v1 -z.
type statusRecord struct {
	x        byte
	y        byte
	path     string
	origPath string
}

// parseStatusPorcelain parses NUL-delimited git status --porcelain=v1 -z output.
func parseStatusPorcelain(data []byte) ([]statusRecord, error) {
	if len(data) == 0 {
		return nil, nil
	}
	tokens := bytes.Split(data, []byte{0})
	var records []statusRecord
	for i := 0; i < len(tokens); {
		if len(tokens[i]) == 0 {
			i++
			continue
		}
		entry := string(tokens[i])
		if len(entry) < 3 {
			i++
			continue
		}
		x := entry[0]
		y := entry[1]
		filePath := entry[3:]

		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			origPath := ""
			if i+1 < len(tokens) && len(tokens[i+1]) > 0 {
				origPath = string(tokens[i+1])
			}
			records = append(records, statusRecord{
				x:        x,
				y:        y,
				path:     filePath,
				origPath: origPath,
			})
			i += 2
		} else {
			records = append(records, statusRecord{
				x:    x,
				y:    y,
				path: filePath,
			})
			i++
		}
	}
	return records, nil
}

// checkIndexConflict checks if git has unmerged files in the index.
func checkIndexConflict(repo *Repository) (bool, []string, error) {
	out, err := repo.git.run("ls-files", "-u", "-z")
	if err != nil {
		return false, nil, err
	}
	if len(out) == 0 {
		return false, nil, nil
	}
	tokens := bytes.Split(out, []byte{0})
	seen := make(map[string]bool)
	var paths []string
	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
		idx := bytes.IndexByte(tok, '\t')
		if idx >= 0 && idx+1 < len(tok) {
			p := string(tok[idx+1:])
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	return len(paths) > 0, paths, nil
}

// getBlobSizeDigest retrieves object size and SHA-256 digest from Git.
func getBlobSizeDigest(repo *Repository, sha string, maxBytes int64) (int64, string, error) {
	if sha == "" || sha == "0000000" || strings.HasPrefix(sha, "0000000") {
		return 0, "", nil
	}
	sizeOut, err := repo.git.run("cat-file", "-s", sha)
	if err != nil {
		return 0, "", err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(sizeOut)), 10, 64)
	if err != nil {
		return 0, "", err
	}
	if size > maxBytes {
		return size, "", nil
	}
	data, err := repo.git.run("cat-file", "-p", sha)
	if err != nil {
		return size, "", err
	}
	return size, DigestBytes(data), nil
}

// getStagedFileInfo inspects an index entry via git ls-files --stage.
func getStagedFileInfo(repo *Repository, filePath string, maxBytes int64) (string, int64, string) {
	out, err := repo.git.run("ls-files", "--stage", "-z", "--", filePath)
	if err != nil || len(out) == 0 {
		return "", 0, ""
	}
	first := bytes.Split(out, []byte{0})[0]
	tabIdx := bytes.IndexByte(first, '\t')
	if tabIdx < 0 {
		return "", 0, ""
	}
	meta := string(first[:tabIdx])
	fields := strings.Fields(meta)
	if len(fields) < 2 {
		return "", 0, ""
	}
	mode := fields[0]
	sha := fields[1]
	if mode == "160000" {
		return mode, 0, ""
	}
	size, digest, _ := getBlobSizeDigest(repo, sha, maxBytes)
	return mode, size, digest
}

// addPartialReason adds reason to list if not already present.
func addPartialReason(list *[]string, reason string) {
	for _, r := range *list {
		if r == reason {
			return
		}
	}
	*list = append(*list, reason)
}

// BuildReviewPackage creates a complete or partial review change package.
func BuildReviewPackage(req ReviewPackageRequest) (ReviewManifest, error) {
	maxFileBytes := req.MaxFileBytes
	if maxFileBytes <= 0 {
		maxFileBytes = DefaultMaxFileBytes
	}
	maxTotalBytes := req.MaxTotalBytes
	if maxTotalBytes <= 0 {
		maxTotalBytes = DefaultMaxTotalBytes
	}

	base := strings.TrimSpace(req.Base)
	if base == "" {
		return ReviewManifest{}, fmt.Errorf("%w: base commit is required", ErrUsage)
	}

	runDir := req.RunDir
	if runDir == "" {
		if req.RunID == "" {
			return ReviewManifest{}, fmt.Errorf("%w: run ID or run directory is required", ErrUsage)
		}
		repoRoot := req.RepoRoot
		if repoRoot == "" {
			repoRoot = "."
		}
		runRoot, err := ResolveRunRoot(repoRoot, "")
		if err != nil {
			return ReviewManifest{}, err
		}
		runDir = filepath.Join(runRoot, req.RunID)
	}

	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ReviewManifest{}, fmt.Errorf("%w: run is not present under the selected run root", ErrNotFound)
		}
		return ReviewManifest{}, err
	}
	if req.RunID != "" && identity.RunID != req.RunID {
		return ReviewManifest{}, fmt.Errorf("%w: run ID does not match run identity", ErrConflict)
	}
	if req.RepoRoot != "" {
		reqAbs, err := filepath.Abs(req.RepoRoot)
		if err == nil {
			idAbs, err := filepath.Abs(identity.Scope.RepoRoot)
			if err == nil && filepath.Clean(reqAbs) != filepath.Clean(idAbs) {
				return ReviewManifest{}, fmt.Errorf("%w: repository root does not match run identity", ErrConflict)
			}
		}
	}

	repo, err := OpenRepository(identity.Scope.RepoRoot)
	if err != nil {
		return ReviewManifest{}, err
	}

	lock, err := acquireRunLock(runDir, "review")
	if err != nil {
		return ReviewManifest{}, err
	}
	defer lock.release()

	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return ReviewManifest{}, err
		}
		baseline := identity.Baseline
		checkpoint = &Checkpoint{
			SchemaVersion:   SchemaVersion,
			RunID:           identity.RunID,
			UpdatedAt:       time.Now().UTC(),
			State:           StatePrepared,
			LastObservation: &baseline,
		}
	}

	baseOut, err := repo.git.run("rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return ReviewManifest{}, fmt.Errorf("%w: cannot resolve base commit %q: %v", ErrUsage, base, err)
	}
	baseCommit := strings.TrimSpace(string(baseOut))

	currentHeadOut, err := repo.git.run("rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return ReviewManifest{}, fmt.Errorf("%w: cannot resolve current HEAD: %v", ErrUsage, err)
	}
	currentHead := strings.TrimSpace(string(currentHeadOut))

	headArg := strings.TrimSpace(req.Head)
	resolvedHead := currentHead
	if headArg != "" {
		headOut, err := repo.git.run("rev-parse", "--verify", headArg+"^{commit}")
		if err != nil {
			return ReviewManifest{}, fmt.Errorf("%w: cannot resolve head commit %q: %v", ErrUsage, headArg, err)
		}
		resolvedHead = strings.TrimSpace(string(headOut))
	}

	headIsCurrent := (resolvedHead == currentHead)

	_, ancestorCode, _ := repo.git.runCode("merge-base", "--is-ancestor", baseCommit, resolvedHead)
	isAncestor := (ancestorCode == 0)

	now := time.Now().UTC()
	packageID, err := newArtifactID("pkg", now)
	if err != nil {
		return ReviewManifest{}, err
	}

	if err := ensureArtifactDir(runDir, "review"); err != nil {
		return ReviewManifest{}, err
	}
	pkgDir := filepath.Join(runDir, "review", packageID)
	if err := os.Mkdir(pkgDir, 0o700); err != nil {
		return ReviewManifest{}, fmt.Errorf("create review package directory: %w", err)
	}
	if err := os.Chmod(pkgDir, 0o700); err != nil {
		return ReviewManifest{}, fmt.Errorf("restrict review package directory permissions: %w", err)
	}

	var partialReasons []string
	var omissions []ReviewOmission
	if !isAncestor {
		addPartialReason(&partialReasons, "base_not_ancestor")
	}

	filesMap := make(map[string]*ReviewFile)
	var totalBytes int64

	var committedPatchBuf bytes.Buffer
	var stagedPatchBuf bytes.Buffer
	var unstagedPatchBuf bytes.Buffer

	var committedCount, stagedCount, unstagedCount, untrackedCount int

	// 1. Committed diffs (baseCommit..resolvedHead)
	rawOut, err := repo.git.run("diff", "--raw", "-z", "--find-renames", "--find-copies", baseCommit, resolvedHead)
	if err != nil {
		return ReviewManifest{}, fmt.Errorf("git diff --raw failed: %w", err)
	}
	rawRecords, err := parseRawDiff(rawOut)
	if err != nil {
		return ReviewManifest{}, err
	}

	sort.Slice(rawRecords, func(i, j int) bool {
		return rawRecords[i].path < rawRecords[j].path
	})

	for _, rec := range rawRecords {
		committedCount++
		p := rec.path
		oldP := rec.oldPath

		isSubmodule := (rec.newMode == "160000" || rec.oldMode == "160000")
		mode := rec.newMode
		if rec.status == "D" {
			mode = rec.oldMode
		}

		var fileSize int64
		var fileDigest string
		if rec.status != "D" && !isSubmodule {
			fileSize, fileDigest, _ = getBlobSizeDigest(repo, rec.newSha, maxFileBytes)
		}

		if isSensitivePath(p) || (oldP != "" && isSensitivePath(oldP)) {
			addPartialReason(&partialReasons, "sensitive_path")
			omissions = append(omissions, ReviewOmission{Path: p, Reason: "sensitive_path"})
			filesMap[p] = &ReviewFile{
				Path:      p,
				OldPath:   oldP,
				Status:    rec.status,
				Mode:      mode,
				Size:      fileSize,
				Digest:    fileDigest,
				Submodule: isSubmodule,
				Included:  false,
				Reason:    "sensitive_path",
			}
			continue
		}

		diffArgs := []string{"diff", "--binary", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--find-copies", baseCommit + ".." + resolvedHead, "--"}
		if oldP != "" {
			diffArgs = append(diffArgs, oldP, p)
		} else {
			diffArgs = append(diffArgs, p)
		}
		diffBytes, err := repo.git.run(diffArgs...)
		if err != nil {
			addPartialReason(&partialReasons, "unreadable_path")
			omissions = append(omissions, ReviewOmission{Path: p, Reason: err.Error()})
			filesMap[p] = &ReviewFile{
				Path:      p,
				OldPath:   oldP,
				Status:    rec.status,
				Mode:      mode,
				Size:      fileSize,
				Digest:    fileDigest,
				Submodule: isSubmodule,
				Included:  false,
				Reason:    err.Error(),
			}
			continue
		}

		isBinary := bytes.Contains(diffBytes, []byte("GIT binary patch")) || bytes.Contains(diffBytes, []byte("Binary files"))

		if int64(len(diffBytes)) > maxFileBytes {
			addPartialReason(&partialReasons, "max_file_bytes_exceeded")
			omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_file_bytes_exceeded"})
			filesMap[p] = &ReviewFile{
				Path:      p,
				OldPath:   oldP,
				Status:    rec.status,
				Binary:    isBinary,
				Submodule: isSubmodule,
				Mode:      mode,
				Size:      fileSize,
				Digest:    fileDigest,
				Included:  false,
				Reason:    "max_file_bytes_exceeded",
			}
			continue
		}

		if totalBytes+int64(len(diffBytes)) > maxTotalBytes {
			addPartialReason(&partialReasons, "max_total_bytes_exceeded")
			omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_total_bytes_exceeded"})
			filesMap[p] = &ReviewFile{
				Path:      p,
				OldPath:   oldP,
				Status:    rec.status,
				Binary:    isBinary,
				Submodule: isSubmodule,
				Mode:      mode,
				Size:      fileSize,
				Digest:    fileDigest,
				Included:  false,
				Reason:    "max_total_bytes_exceeded",
			}
			continue
		}

		totalBytes += int64(len(diffBytes))
		committedPatchBuf.Write(diffBytes)
		filesMap[p] = &ReviewFile{
			Path:      p,
			OldPath:   oldP,
			Status:    rec.status,
			Binary:    isBinary,
			Submodule: isSubmodule,
			Mode:      mode,
			Size:      fileSize,
			Digest:    fileDigest,
			Included:  true,
		}
	}

	// 2. Head check & Worktree / Index Collection
	hasConflict := false
	if !headIsCurrent {
		addPartialReason(&partialReasons, "head_not_current")
		omissions = append(omissions, ReviewOmission{Path: "(staged)", Reason: "head_not_current"})
		omissions = append(omissions, ReviewOmission{Path: "(unstaged)", Reason: "head_not_current"})
		omissions = append(omissions, ReviewOmission{Path: "(untracked)", Reason: "head_not_current"})
	} else {
		statusOut, err := repo.git.run("status", "--porcelain=v1", "-z", "--untracked-files=normal")
		if err != nil {
			return ReviewManifest{}, fmt.Errorf("git status failed: %w", err)
		}
		statusRecords, err := parseStatusPorcelain(statusOut)
		if err != nil {
			return ReviewManifest{}, err
		}

		conflictPresent, unmergedPaths, _ := checkIndexConflict(repo)
		hasConflict = conflictPresent
		for _, rec := range statusRecords {
			if rec.x == 'U' || rec.y == 'U' || (rec.x == 'A' && rec.y == 'A') || (rec.x == 'D' && rec.y == 'D') {
				hasConflict = true
			}
		}

		if hasConflict {
			addPartialReason(&partialReasons, "index_conflict")
			omissions = append(omissions, ReviewOmission{Path: "(index)", Reason: "index_conflict"})
			for _, u := range unmergedPaths {
				f, ok := filesMap[u]
				if !ok {
					f = &ReviewFile{Path: u}
					filesMap[u] = f
				}
				f.Status = "U"
				f.Included = false
				f.Reason = "index_conflict"
				omissions = append(omissions, ReviewOmission{Path: u, Reason: "index_conflict"})
			}
		}

		// Staged changes (head..index)
		if !hasConflict {
			var stagedRecs []statusRecord
			for _, rec := range statusRecords {
				if rec.x != ' ' && rec.x != '?' && rec.x != '!' && rec.x != 'U' {
					stagedRecs = append(stagedRecs, rec)
				}
			}
			sort.Slice(stagedRecs, func(i, j int) bool {
				return stagedRecs[i].path < stagedRecs[j].path
			})

			for _, rec := range stagedRecs {
				stagedCount++
				p := rec.path
				oldP := rec.origPath
				f, ok := filesMap[p]
				if !ok {
					f = &ReviewFile{Path: p, OldPath: oldP, Status: string(rec.x)}
					filesMap[p] = f
				}
				f.Staged = true
				if f.Status == "" {
					f.Status = string(rec.x)
				}

				mode, size, digest := getStagedFileInfo(repo, p, maxFileBytes)
				if mode != "" {
					f.Mode = mode
				}
				if f.Size == 0 {
					f.Size = size
				}
				if f.Digest == "" {
					f.Digest = digest
				}
				if f.Mode == "160000" {
					f.Submodule = true
				}

				if isSensitivePath(p) || (oldP != "" && isSensitivePath(oldP)) {
					addPartialReason(&partialReasons, "sensitive_path")
					omissions = append(omissions, ReviewOmission{Path: p, Reason: "sensitive_path"})
					f.Included = false
					f.Reason = "sensitive_path"
					continue
				}

				diffArgs := []string{"diff", "--binary", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--find-copies", "--cached", resolvedHead, "--"}
				if oldP != "" {
					diffArgs = append(diffArgs, oldP, p)
				} else {
					diffArgs = append(diffArgs, p)
				}
				diffBytes, err := repo.git.run(diffArgs...)
				if err != nil {
					addPartialReason(&partialReasons, "unreadable_path")
					omissions = append(omissions, ReviewOmission{Path: p, Reason: err.Error()})
					f.Included = false
					f.Reason = err.Error()
					continue
				}

				isBinary := bytes.Contains(diffBytes, []byte("GIT binary patch")) || bytes.Contains(diffBytes, []byte("Binary files"))
				f.Binary = f.Binary || isBinary

				if int64(len(diffBytes)) > maxFileBytes {
					addPartialReason(&partialReasons, "max_file_bytes_exceeded")
					omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_file_bytes_exceeded"})
					f.Included = false
					f.Reason = "max_file_bytes_exceeded"
					continue
				}
				if totalBytes+int64(len(diffBytes)) > maxTotalBytes {
					addPartialReason(&partialReasons, "max_total_bytes_exceeded")
					omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_total_bytes_exceeded"})
					f.Included = false
					f.Reason = "max_total_bytes_exceeded"
					continue
				}

				totalBytes += int64(len(diffBytes))
				stagedPatchBuf.Write(diffBytes)
				if f.Reason == "" {
					f.Included = true
				}
			}
		}

		// Unstaged changes (index..worktree)
		var unstagedRecs []statusRecord
		for _, rec := range statusRecords {
			if rec.y != ' ' && rec.y != '?' && rec.y != '!' && rec.y != 'U' {
				unstagedRecs = append(unstagedRecs, rec)
			}
		}
		sort.Slice(unstagedRecs, func(i, j int) bool {
			return unstagedRecs[i].path < unstagedRecs[j].path
		})

		for _, rec := range unstagedRecs {
			unstagedCount++
			p := rec.path
			f, ok := filesMap[p]
			if !ok {
				f = &ReviewFile{Path: p, Status: string(rec.y)}
				filesMap[p] = f
			}
			f.Unstaged = true
			if f.Status == "" {
				f.Status = string(rec.y)
			}

			full := filepath.Join(repo.Root, filepath.FromSlash(p))
			info, statErr := os.Lstat(full)
			if statErr == nil {
				if info.Mode()&os.ModeSymlink != 0 {
					f.Mode = "120000"
					if target, err := os.Readlink(full); err == nil {
						f.Size = int64(len(target))
						f.Digest = DigestBytes([]byte(target))
					}
				} else if info.Mode().IsRegular() {
					f.Mode = fmt.Sprintf("%06o", info.Mode().Perm())
					f.Size = info.Size()
					if info.Size() <= maxFileBytes {
						if data, err := os.ReadFile(full); err == nil {
							f.Digest = DigestBytes(data)
						}
					}
				}
			} else if errors.Is(statErr, fs.ErrNotExist) {
				f.Status = "D"
			}

			if isSensitivePath(p) {
				addPartialReason(&partialReasons, "sensitive_path")
				omissions = append(omissions, ReviewOmission{Path: p, Reason: "sensitive_path"})
				f.Included = false
				f.Reason = "sensitive_path"
				continue
			}

			diffArgs := []string{"diff", "--binary", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--find-copies", "--", p}
			diffBytes, err := repo.git.run(diffArgs...)
			if err != nil {
				addPartialReason(&partialReasons, "unreadable_path")
				omissions = append(omissions, ReviewOmission{Path: p, Reason: err.Error()})
				f.Included = false
				f.Reason = err.Error()
				continue
			}

			isBinary := bytes.Contains(diffBytes, []byte("GIT binary patch")) || bytes.Contains(diffBytes, []byte("Binary files"))
			f.Binary = f.Binary || isBinary

			if int64(len(diffBytes)) > maxFileBytes {
				addPartialReason(&partialReasons, "max_file_bytes_exceeded")
				omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_file_bytes_exceeded"})
				f.Included = false
				f.Reason = "max_file_bytes_exceeded"
				continue
			}
			if totalBytes+int64(len(diffBytes)) > maxTotalBytes {
				addPartialReason(&partialReasons, "max_total_bytes_exceeded")
				omissions = append(omissions, ReviewOmission{Path: p, Reason: "max_total_bytes_exceeded"})
				f.Included = false
				f.Reason = "max_total_bytes_exceeded"
				continue
			}

			totalBytes += int64(len(diffBytes))
			unstagedPatchBuf.Write(diffBytes)
			if f.Reason == "" {
				f.Included = true
			}
		}

		// Untracked changes
		untrackedOut, err := repo.git.run("ls-files", "-z", "--others", "--exclude-standard")
		if err == nil {
			var untrackedNames []string
			for _, raw := range bytes.Split(untrackedOut, []byte{0}) {
				if len(raw) == 0 {
					continue
				}
				name := string(raw)
				if withinPath(filepath.Join(repo.Root, ".agentic-sdd", "runs"), filepath.Join(repo.Root, filepath.FromSlash(name))) {
					continue
				}
				if withinPath(runDir, filepath.Join(repo.Root, filepath.FromSlash(name))) {
					continue
				}
				untrackedNames = append(untrackedNames, name)
			}
			sort.Strings(untrackedNames)

			untrackedDir := filepath.Join(pkgDir, "untracked")

			for _, name := range untrackedNames {
				untrackedCount++
				f, ok := filesMap[name]
				if !ok {
					f = &ReviewFile{Path: name, Status: "??"}
					filesMap[name] = f
				}
				f.Untracked = true

				if isSensitivePath(name) {
					addPartialReason(&partialReasons, "sensitive_path")
					omissions = append(omissions, ReviewOmission{Path: name, Reason: "sensitive_path"})
					f.Included = false
					f.Reason = "sensitive_path"
					continue
				}

				full := filepath.Join(repo.Root, filepath.FromSlash(name))
				info, statErr := os.Lstat(full)
				if statErr != nil {
					addPartialReason(&partialReasons, "unreadable_path")
					omissions = append(omissions, ReviewOmission{Path: name, Reason: statErr.Error()})
					f.Included = false
					f.Reason = statErr.Error()
					continue
				}

				dest := filepath.Join(untrackedDir, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
					addPartialReason(&partialReasons, "unreadable_path")
					omissions = append(omissions, ReviewOmission{Path: name, Reason: err.Error()})
					f.Included = false
					f.Reason = err.Error()
					continue
				}

				switch {
				case info.Mode()&os.ModeSymlink != 0:
					target, readErr := os.Readlink(full)
					if readErr != nil {
						addPartialReason(&partialReasons, "unreadable_path")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: readErr.Error()})
						f.Included = false
						f.Reason = readErr.Error()
						continue
					}
					f.Mode = "120000"
					f.Size = int64(len(target))
					f.Digest = DigestBytes([]byte(target))
					symlinkMeta := []byte("symlink -> " + target + "\n")
					if err := writeFileAtomic(dest, symlinkMeta, 0o600); err != nil {
						addPartialReason(&partialReasons, "unreadable_path")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: err.Error()})
						f.Included = false
						f.Reason = err.Error()
						continue
					}
					totalBytes += int64(len(symlinkMeta))
					f.Included = true

				case info.Mode().IsRegular():
					f.Mode = fmt.Sprintf("%06o", info.Mode().Perm())
					f.Size = info.Size()

					if info.Size() > maxFileBytes {
						addPartialReason(&partialReasons, "max_file_bytes_exceeded")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: "max_file_bytes_exceeded"})
						f.Included = false
						f.Reason = "max_file_bytes_exceeded"
						continue
					}
					if totalBytes+info.Size() > maxTotalBytes {
						addPartialReason(&partialReasons, "max_total_bytes_exceeded")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: "max_total_bytes_exceeded"})
						f.Included = false
						f.Reason = "max_total_bytes_exceeded"
						continue
					}

					data, readErr := os.ReadFile(full)
					if readErr != nil {
						addPartialReason(&partialReasons, "unreadable_path")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: readErr.Error()})
						f.Included = false
						f.Reason = readErr.Error()
						continue
					}

					f.Digest = DigestBytes(data)
					if bytes.IndexByte(data, 0) >= 0 {
						f.Binary = true
					}

					if err := writeFileAtomic(dest, data, 0o600); err != nil {
						addPartialReason(&partialReasons, "unreadable_path")
						omissions = append(omissions, ReviewOmission{Path: name, Reason: err.Error()})
						f.Included = false
						f.Reason = err.Error()
						continue
					}
					totalBytes += int64(len(data))
					f.Included = true

				default:
					addPartialReason(&partialReasons, "unreadable_path")
					omissions = append(omissions, ReviewOmission{Path: name, Reason: "not a regular file or symlink"})
					f.Included = false
					f.Reason = "not a regular file or symlink"
				}
			}
		}
	}

	// 3. Write review.patch
	var patchBuf bytes.Buffer
	patchBuf.WriteString("# === committed ===\n")
	patchBuf.Write(committedPatchBuf.Bytes())
	patchBuf.WriteString("\n# === staged ===\n")
	if !headIsCurrent {
		patchBuf.WriteString("# omitted: head_not_current\n")
	} else if hasConflict {
		patchBuf.WriteString("# omitted: index_conflict\n")
	} else {
		patchBuf.Write(stagedPatchBuf.Bytes())
	}
	patchBuf.WriteString("\n# === unstaged ===\n")
	if !headIsCurrent {
		patchBuf.WriteString("# omitted: head_not_current\n")
	} else {
		patchBuf.Write(unstagedPatchBuf.Bytes())
	}

	patchPath := filepath.Join(pkgDir, "review.patch")
	if err := writeFileAtomic(patchPath, patchBuf.Bytes(), 0o600); err != nil {
		return ReviewManifest{}, fmt.Errorf("write review.patch: %w", err)
	}

	// 4. Build manifest files
	files := make([]ReviewFile, 0, len(filesMap))
	for _, f := range filesMap {
		files = append(files, *f)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	complete := (len(omissions) == 0 && len(partialReasons) == 0)

	fresh := evaluateFreshness(identity, checkpoint, runDir)
	runState := effectiveState(checkpoint.State, fresh.reasons, fresh.lock)

	untrackedDir := ""
	if untrackedCount > 0 {
		untrackedDir = filepath.Join(pkgDir, "untracked")
	}

	manifest := ReviewManifest{
		SchemaVersion:  SchemaVersion,
		Kind:           "execution.review.package",
		PackageID:      packageID,
		RunID:          identity.RunID,
		CreatedAt:      now,
		RepoRoot:       identity.Scope.RepoRoot,
		Base:           baseCommit,
		Head:           resolvedHead,
		HeadRequested:  req.Head,
		Branch:         repo.Branch,
		GitVersion:     repo.GitVersion,
		Scope:          identity.Scope,
		Complete:       complete,
		PartialReasons: partialReasons,
		Files:          files,
		Omissions:      omissions,
		Counts: ReviewCounts{
			Committed: committedCount,
			Staged:    stagedCount,
			Unstaged:  unstagedCount,
			Untracked: untrackedCount,
			Omitted:   len(omissions),
		},
		PatchPath:    patchPath,
		UntrackedDir: untrackedDir,
		RunState:     runState,
		StaleReasons: fresh.reasons,
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ReviewManifest{}, fmt.Errorf("encode review.json: %w", err)
	}
	manifestPath := filepath.Join(pkgDir, "review.json")
	if err := writeFileAtomic(manifestPath, append(manifestJSON, '\n'), 0o600); err != nil {
		return ReviewManifest{}, fmt.Errorf("write review.json: %w", err)
	}

	// 5. Update checkpoint
	obs, err := ObserveRun(repo, identity.Scope, runDir)
	if err == nil {
		checkpoint.LastObservation = &obs
	}
	checkpoint.ReviewPackages = append(checkpoint.ReviewPackages, ReviewPackageRecord{
		PackageID:      packageID,
		ManifestPath:   manifestPath,
		PatchPath:      patchPath,
		Complete:       complete,
		PartialReasons: partialReasons,
		CreatedAt:      now,
	})
	if err := WriteCheckpoint(runDir, checkpoint); err != nil {
		return ReviewManifest{}, fmt.Errorf("update checkpoint: %w", err)
	}

	if !complete {
		return manifest, ErrPartial
	}
	return manifest, nil
}
