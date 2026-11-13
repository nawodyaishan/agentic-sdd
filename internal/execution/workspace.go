package execution

// Run workspace identity, layout, permissions, atomic writes, per-run locking
// and scoped cleanup. A run is private scratch under a canonical root; it is
// Git-invisible by default but never a sandbox and never an approval record.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	runIDPrefix      = "run-"
	artifactRuneSize = 3
	runIDTypeRunes   = 6

	runRootParentDir = ".agentic-sdd"
	runRootDirName   = "runs"
	runIdentityName  = "run.json"
	checkpointName   = "checkpoint.json"
	lockFileName     = ".lock"
	ignoreFileName   = ".gitignore"
	ignoreFileData   = "*"

	maxRunIDAttempts    = 5
	maxArtifactAttempts = 5
	maxRunIdentityBytes = 4 << 20
	maxCheckpointBytes  = 8 << 20
	maxLockBytes        = 64 << 10
	maxIgnoreBytes      = 64 << 10

	lockSchemaVersion = 1

	// LockStateNone, LockStateStale and LockStateLive describe an inspected
	// per-run writer lock.
	LockStateNone  = "none"
	LockStateStale = "stale"
	LockStateLive  = "live"
)

var runIDPattern = regexp.MustCompile(`^run-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$`)

// NewRunID returns a path-safe, collision-resistant run identifier in the form
// run-<UTC compact>-<6 hex>.
func NewRunID(now time.Time) (string, error) {
	return newArtifactID("run", now)
}

// newArtifactID builds <prefix>-<UTC compact>-<6 hex> identifiers for run and
// record artifacts.
func newArtifactID(prefix string, now time.Time) (string, error) {
	raw := make([]byte, artifactRuneSize)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("random identifier unavailable")
	}
	return prefix + "-" + now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(raw), nil
}

// ValidateRunID refuses anything but the canonical run identifier shape so a
// caller-supplied identifier can never escape the run root.
func ValidateRunID(id string) error {
	if !runIDPattern.MatchString(id) {
		return fmt.Errorf("%w: run ID is invalid (expected run-YYYYMMDDTHHMMSSZ-xxxxxx)", ErrUsage)
	}
	return nil
}

// ResolveRunRoot returns the run root path without creating it. An empty
// explicit root selects <repo>/.agentic-sdd/runs.
func ResolveRunRoot(repoRoot, explicit string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("%w: repository root is required", ErrUsage)
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("%w: repository root is invalid", ErrUsage)
	}
	root = filepath.Clean(root)
	if strings.TrimSpace(explicit) == "" {
		return filepath.Join(root, runRootParentDir, runRootDirName), nil
	}
	abs, err := filepath.Abs(explicit)
	if err != nil {
		return "", fmt.Errorf("%w: run root is invalid", ErrUsage)
	}
	return filepath.Clean(abs), nil
}

// InitRun creates one private run workspace: a fresh run directory, run.json,
// and a prepared checkpoint. It writes only private run artifacts and never
// modifies tracked project files.
func InitRun(req InitRequest) (RunIdentity, error) {
	if err := validateScope(req.Scope); err != nil {
		return RunIdentity{}, err
	}
	repoRoot := strings.TrimSpace(req.Scope.RepoRoot)
	repo, err := OpenRepository(repoRoot)
	if err != nil {
		return RunIdentity{}, err
	}
	if err := sameRoot(repo.Root, repoRoot); err != nil {
		return RunIdentity{}, err
	}
	runRoot, explicit, err := initRunRoot(repo.Root, req.RunRoot)
	if err != nil {
		return RunIdentity{}, err
	}
	gitVisible, err := prepareRunRoot(repo, runRoot, explicit)
	if err != nil {
		return RunIdentity{}, err
	}
	now := time.Now().UTC()
	runID, runDir, err := createRunDir(runRoot, now)
	if err != nil {
		return RunIdentity{}, err
	}
	cleanup := func() { os.RemoveAll(runDir) }
	lock, err := acquireRunLock(runDir, "init")
	if err != nil {
		cleanup()
		return RunIdentity{}, err
	}
	baseline, err := ObserveRun(repo, req.Scope, runDir)
	if err != nil {
		lock.release()
		cleanup()
		return RunIdentity{}, err
	}
	approvalReference := req.ApprovalReference
	if approvalReference == "" && req.Scope.Kind == ScopeFeature {
		approvalReference = req.Scope.SpecPath + " §Approval"
	}
	identity := RunIdentity{
		SchemaVersion:       SchemaVersion,
		Kind:                "execution.run.identity",
		RunID:               runID,
		CreatedAt:           now,
		ToolVersion:         req.ToolVersion,
		RunRoot:             runRoot,
		RunDir:              runDir,
		Scope:               req.Scope,
		Baseline:            baseline,
		ApprovalReference:   approvalReference,
		WorkspaceGitVisible: gitVisible,
	}
	writeErr := func() error {
		if err := writeRunIdentity(runDir, identity); err != nil {
			return err
		}
		checkpoint := &Checkpoint{
			SchemaVersion:   SchemaVersion,
			RunID:           runID,
			UpdatedAt:       now,
			State:           StatePrepared,
			LastObservation: &baseline,
		}
		return WriteCheckpoint(runDir, checkpoint)
	}()
	releaseErr := lock.release()
	if writeErr != nil {
		cleanup()
		return RunIdentity{}, writeErr
	}
	if releaseErr != nil {
		cleanup()
		return RunIdentity{}, releaseErr
	}
	return identity, nil
}

// CleanRun previews (apply=false) or removes (apply=true) exactly one owned
// run directory. An already-absent run is a success. A live lock, a symlinked
// run, or a directory without matching ownership evidence is refused.
func CleanRun(runRoot, runID string, apply bool) (CleanResult, error) {
	if err := ValidateRunID(runID); err != nil {
		return CleanResult{}, err
	}
	root, err := canonicalRunRoot(runRoot)
	if err != nil {
		return CleanResult{}, err
	}
	runDir := filepath.Join(root, runID)
	result := CleanResult{
		SchemaVersion: SchemaVersion,
		Kind:          kindRunClean,
		RunID:         runID,
		RunRoot:       root,
		RunDir:        runDir,
		Applied:       apply,
		LockState:     LockStateNone,
	}
	info, err := os.Lstat(runDir)
	if errors.Is(err, fs.ErrNotExist) {
		result.AlreadyAbsent = true
		return result, nil
	}
	if err != nil {
		return CleanResult{}, fmt.Errorf("run directory is unreadable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return CleanResult{}, fmt.Errorf("%w: run directory is a symlink or not a directory", ErrUnsafePath)
	}
	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return CleanResult{}, err
		}
		return CleanResult{}, fmt.Errorf("%w: run directory has no valid ownership record", ErrUnsafePath)
	}
	if identity.RunID != runID {
		return CleanResult{}, fmt.Errorf("%w: run identity does not match the requested run", ErrConflict)
	}
	if filepath.Clean(identity.RunRoot) != root || filepath.Clean(identity.RunDir) != runDir {
		return CleanResult{}, fmt.Errorf("%w: run identity points outside the requested run directory", ErrConflict)
	}
	lock := inspectLock(runDir)
	result.LockState = lockStateName(lock)
	if lock.live {
		return CleanResult{}, fmt.Errorf("%w: run is locked by a live writer", ErrConflict)
	}
	count, total, entries, err := runDirUsage(runDir)
	if err != nil {
		return CleanResult{}, fmt.Errorf("run directory is unreadable: %w", err)
	}
	result.FileCount = count
	result.TotalBytes = total
	result.Entries = entries
	if !apply {
		return result, nil
	}
	if runDir == root || filepath.Dir(runDir) != root {
		return CleanResult{}, fmt.Errorf("%w: refusing to remove a path outside the owned run root", ErrUnsafePath)
	}
	if err := os.RemoveAll(runDir); err != nil {
		return CleanResult{}, fmt.Errorf("remove run directory: %w", err)
	}
	if _, err := os.Lstat(runDir); !errors.Is(err, fs.ErrNotExist) {
		return CleanResult{}, fmt.Errorf("run directory still exists after removal")
	}
	result.Removed = true
	return result, nil
}

// LoadRunIdentity reads and validates one run.json from runDir.
func LoadRunIdentity(runDir string) (RunIdentity, error) {
	var identity RunIdentity
	info, err := os.Lstat(runDir)
	if err != nil {
		return identity, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return identity, fmt.Errorf("%w: run directory is a symlink or not a directory", ErrUnsafePath)
	}
	data, err := readBoundedRegular(filepath.Join(runDir, runIdentityName), maxRunIdentityBytes)
	if err != nil {
		return identity, err
	}
	if err := json.Unmarshal(data, &identity); err != nil {
		return identity, fmt.Errorf("%w: run identity is not valid JSON", ErrUsage)
	}
	if identity.SchemaVersion != SchemaVersion {
		return identity, fmt.Errorf("%w: unsupported run identity schema", ErrUnsupported)
	}
	if err := ValidateRunID(identity.RunID); err != nil {
		return identity, err
	}
	if filepath.Clean(identity.RunDir) != filepath.Clean(runDir) {
		return identity, fmt.Errorf("%w: run identity directory does not match its location", ErrConflict)
	}
	if err := validateScope(identity.Scope); err != nil {
		return identity, err
	}
	return identity, nil
}

// writeRunIdentity validates and atomically writes run.json.
func writeRunIdentity(runDir string, identity RunIdentity) error {
	if identity.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: unsupported run identity schema", ErrUnsupported)
	}
	if err := ValidateRunID(identity.RunID); err != nil {
		return err
	}
	if filepath.Clean(identity.RunDir) != filepath.Clean(runDir) {
		return fmt.Errorf("%w: run identity directory does not match its location", ErrConflict)
	}
	if err := validateScope(identity.Scope); err != nil {
		return err
	}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("run identity encoding unavailable")
	}
	return writeFileAtomic(filepath.Join(runDir, runIdentityName), append(data, '\n'), 0o600)
}

// initRunRoot resolves the requested run root and reports whether it was
// explicit.
func initRunRoot(repoRoot, explicit string) (string, bool, error) {
	if strings.TrimSpace(explicit) != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", false, fmt.Errorf("%w: run root is invalid", ErrUsage)
		}
		canonical, err := canonicalExistingPrefix(filepath.Clean(abs))
		if err != nil {
			return "", false, err
		}
		if withinPath(filepath.Join(repoRoot, ".git"), canonical) {
			return "", false, fmt.Errorf("%w: run root cannot be inside .git", ErrUnsafePath)
		}
		return canonical, true, nil
	}
	return filepath.Join(repoRoot, runRootParentDir, runRootDirName), false, nil
}

// prepareRunRoot creates the run root and, for the repository-local default,
// guarantees that the whole run tree is Git-ignored without editing tracked
// files. An explicit root is created as a private directory and its visibility
// is reported honestly.
func prepareRunRoot(repo *Repository, runRoot string, explicit bool) (bool, error) {
	if withinPath(repo.Root, runRoot) {
		rel, err := filepath.Rel(repo.Root, runRoot)
		if err != nil {
			return false, fmt.Errorf("%w: run root is outside the repository", ErrUnsafePath)
		}
		if err := ensureNoSymlink(repo.Root, rel); err != nil {
			return false, err
		}
	}
	existed := false
	if info, err := os.Lstat(runRoot); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, fmt.Errorf("%w: run root is a symlink or not a directory", ErrUnsafePath)
		}
		existed = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("run root is unreadable: %w", err)
	}
	if !existed {
		if err := os.MkdirAll(runRoot, 0o700); err != nil {
			return false, fmt.Errorf("create run root: %w", err)
		}
		// Restrictive permissions even under a permissive umask. An existing
		// user-supplied root keeps its own mode.
		if err := os.Chmod(runRoot, 0o700); err != nil {
			return false, fmt.Errorf("restrict run root permissions: %w", err)
		}
	}
	info, err := os.Lstat(runRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("%w: run root is not a regular directory", ErrUnsafePath)
	}
	if explicit {
		if !withinPath(repo.Root, runRoot) {
			return false, nil
		}
		ignored, err := repo.isIgnored(runRoot)
		if err != nil {
			return false, err
		}
		return !ignored, nil
	}
	if err := ensureIgnoreFile(runRoot); err != nil {
		return false, err
	}
	rel, err := filepath.Rel(repo.Root, runRoot)
	if err != nil {
		return false, fmt.Errorf("%w: run root is outside the repository", ErrUnsafePath)
	}
	probe := filepath.ToSlash(filepath.Join(rel, "probe-run", runIdentityName))
	ignored, err := repo.isIgnoredPath(probe)
	if err != nil {
		return false, err
	}
	if !ignored {
		return false, fmt.Errorf("%w: run root is not Git-ignored; pass an explicit --run-root", ErrUsage)
	}
	tracked, err := repo.git.run("ls-files", "-z", "--", filepath.ToSlash(rel))
	if err != nil {
		return false, fmt.Errorf("cannot verify run root tracking state: %w", err)
	}
	if len(strings.Trim(string(tracked), "\x00")) != 0 {
		return false, fmt.Errorf("%w: run root contains tracked files", ErrUsage)
	}
	return false, nil
}

// isIgnored reports whether an absolute path is ignored by the repository's
// ignore rules.
func (r *Repository) isIgnored(target string) (bool, error) {
	rel, err := filepath.Rel(r.Root, target)
	if err != nil || !withinPath(r.Root, target) {
		return false, fmt.Errorf("%w: path is outside the repository", ErrUnsafePath)
	}
	return r.isIgnoredPath(filepath.ToSlash(rel))
}

func (r *Repository) isIgnoredPath(rel string) (bool, error) {
	_, code, err := r.git.runCode("check-ignore", "-q", "--", rel)
	if err != nil {
		return false, fmt.Errorf("cannot verify ignore rules: %w", err)
	}
	switch code {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, fmt.Errorf("cannot verify ignore rules (Git exit %d)", code)
	}
}

// ensureIgnoreFile creates a self-ignoring runs/.gitignore when absent. An
// existing ignore file is never rewritten; the caller verifies its effect.
func ensureIgnoreFile(runRoot string) error {
	ignPath := filepath.Join(runRoot, ignoreFileName)
	info, err := os.Lstat(ignPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: run-root ignore file is a symlink or not a regular file", ErrUnsafePath)
		}
		if _, err := readBoundedRegular(ignPath, maxIgnoreBytes); err != nil {
			return fmt.Errorf("%w: run-root ignore file is unreadable", ErrUnsafePath)
		}
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("run-root ignore file is unreadable: %w", err)
	}
	return writeFileAtomic(ignPath, []byte(ignoreFileData), 0o600)
}

// createRunDir creates the run directory and its standard children, retrying
// on an identifier collision and cleaning up on failure.
func createRunDir(runRoot string, now time.Time) (string, string, error) {
	for attempt := 0; attempt < maxRunIDAttempts; attempt++ {
		runID, err := NewRunID(now)
		if err != nil {
			return "", "", err
		}
		runDir := filepath.Join(runRoot, runID)
		if err := os.Mkdir(runDir, 0o700); err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", "", fmt.Errorf("create run directory: %w", err)
		}
		if err := os.Chmod(runDir, 0o700); err != nil {
			os.RemoveAll(runDir)
			return "", "", fmt.Errorf("restrict run directory permissions: %w", err)
		}
		for _, sub := range []string{"briefs", "checks", "records", "review"} {
			if err := os.Mkdir(filepath.Join(runDir, sub), 0o700); err != nil {
				os.RemoveAll(runDir)
				return "", "", fmt.Errorf("create run layout: %w", err)
			}
		}
		return runID, runDir, nil
	}
	return "", "", fmt.Errorf("%w: could not allocate a unique run ID", ErrConflict)
}

// canonicalRunRoot resolves an existing prefix of the run root while refusing
// a final symlink component.
func canonicalRunRoot(runRoot string) (string, error) {
	if strings.TrimSpace(runRoot) == "" {
		return "", fmt.Errorf("%w: run root is required", ErrUsage)
	}
	abs, err := filepath.Abs(runRoot)
	if err != nil {
		return "", fmt.Errorf("%w: run root is invalid", ErrUsage)
	}
	return canonicalExistingPrefix(filepath.Clean(abs))
}

// canonicalExistingPrefix resolves symlinks in the longest existing prefix and
// appends the missing remainder. A final existing symlink is refused.
func canonicalExistingPrefix(target string) (string, error) {
	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: symlinked run root paths are refused", ErrUnsafePath)
		}
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil {
			return "", fmt.Errorf("%w: run root is unreadable", ErrUnsafePath)
		}
		return filepath.Clean(resolved), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%w: run root is unreadable", ErrUnsafePath)
	}
	parent := filepath.Dir(target)
	if parent == target {
		return target, nil
	}
	resolvedParent, err := canonicalExistingPrefix(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(target)), nil
}

// ensureArtifactDir creates or verifies one artifact directory under the run
// directory. Existing symlinks and non-directories are refused.
func ensureArtifactDir(runDir, name string) error {
	dir := filepath.Join(runDir, name)
	info, err := os.Lstat(dir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%w: artifact directory is a symlink or not a directory", ErrUnsafePath)
		}
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("artifact directory is unreadable: %w", err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("restrict artifact directory permissions: %w", err)
	}
	return nil
}

// writeFileAtomic writes data to a same-directory temporary file, syncs it,
// renames it over the target and syncs the directory.
func writeFileAtomic(target string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(target)
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if dirInfo.Mode()&os.ModeSymlink != 0 || !dirInfo.IsDir() {
		return fmt.Errorf("%w: artifact directory is a symlink or not a directory", ErrUnsafePath)
	}
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: refusing to replace a symlinked artifact", ErrUnsafePath)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	remove := true
	defer func() {
		if remove {
			os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return err
	}
	remove = false
	return syncDir(dir)
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// runDirUsage counts regular-file bytes and lists top-level entries without
// following symlinks.
func runDirUsage(runDir string) (int, int64, []string, error) {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return 0, 0, nil, err
	}
	names := make([]string, 0, len(entries))
	count := 0
	var total int64
	for _, entry := range entries {
		names = append(names, entry.Name())
		walkErr := filepath.WalkDir(filepath.Join(runDir, entry.Name()), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			count++
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			if info.Mode().IsRegular() {
				total += info.Size()
			}
			return nil
		})
		if walkErr != nil {
			return 0, 0, nil, walkErr
		}
	}
	sort.Strings(names)
	return count, total, names, nil
}

// validateScope enforces the recorded identity invariants for both scope kinds.
func validateScope(scope Scope) error {
	if strings.TrimSpace(scope.RepoRoot) == "" {
		return fmt.Errorf("%w: scope repository root is required", ErrUsage)
	}
	switch scope.Kind {
	case ScopeFeature:
		if err := validateToken("batch ID", scope.BatchID); err != nil {
			return err
		}
		for name, value := range map[string]string{
			"feature directory": scope.FeatureDir,
			"spec.md path":      scope.SpecPath,
			"tasks.md path":     scope.TasksPath,
		} {
			if err := ValidateRelativePath(value); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		if scope.SpecDigest == "" || scope.TasksDigest == "" || scope.BatchScopeDigest == "" {
			return fmt.Errorf("%w: feature scope requires source digests", ErrUsage)
		}
	case ScopeDirect:
		if err := validateLabel(scope.Label); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: scope kind must be feature or direct", ErrUsage)
	}
	return nil
}

// sameRoot verifies that a requested repository root resolves to the opened
// canonical root.
func sameRoot(canonical, requested string) error {
	abs, err := filepath.Abs(requested)
	if err != nil {
		return fmt.Errorf("%w: repository root is invalid", ErrUsage)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("%w: repository root is unreadable", ErrUsage)
	}
	if filepath.Clean(resolved) != filepath.Clean(canonical) {
		return fmt.Errorf("%w: scope repository root does not match the opened worktree", ErrConflict)
	}
	return nil
}

// runLock is the private per-run writer lock payload.
type runLock struct {
	SchemaVersion int       `json:"schemaVersion"`
	Token         string    `json:"token"`
	PID           int       `json:"pid"`
	Hostname      string    `json:"hostname"`
	StartedAt     time.Time `json:"started_at"`
	Operation     string    `json:"operation"`
}

// runLockHandle owns one acquired lock for as long as release is not called.
type runLockHandle struct {
	path     string
	token    string
	released bool
}

// lockObservation is the inspected lock state.
type lockObservation struct {
	exists bool
	live   bool
	stale  bool
	age    time.Duration
	lock   runLock
}

// acquireRunLock creates the per-run lock with O_CREATE|O_EXCL. A stale lock is
// reclaimed only when its PID is dead on this host or it is older than
// LockStaleAfter; any other existing lock is a conflict.
func acquireRunLock(runDir, operation string) (*runLockHandle, error) {
	info, err := os.Lstat(runDir)
	if err != nil {
		return nil, fmt.Errorf("run directory is unreadable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("%w: run directory is a symlink or not a directory", ErrUnsafePath)
	}
	lockPath := filepath.Join(runDir, lockFileName)
	for attempt := 0; attempt < 2; attempt++ {
		token, err := randomToken()
		if err != nil {
			return nil, err
		}
		payload, err := json.Marshal(runLock{
			SchemaVersion: lockSchemaVersion,
			Token:         token,
			PID:           os.Getpid(),
			Hostname:      hostName(),
			StartedAt:     time.Now().UTC(),
			Operation:     operation,
		})
		if err != nil {
			return nil, fmt.Errorf("lock encoding unavailable")
		}
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			if _, err := f.Write(payload); err != nil {
				f.Close()
				os.Remove(lockPath)
				return nil, fmt.Errorf("write run lock: %w", err)
			}
			if err := f.Sync(); err != nil {
				f.Close()
				os.Remove(lockPath)
				return nil, fmt.Errorf("sync run lock: %w", err)
			}
			if err := f.Close(); err != nil {
				os.Remove(lockPath)
				return nil, fmt.Errorf("close run lock: %w", err)
			}
			if err := syncDir(runDir); err != nil {
				os.Remove(lockPath)
				return nil, fmt.Errorf("sync run directory: %w", err)
			}
			return &runLockHandle{path: lockPath, token: token}, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("create run lock: %w", err)
		}
		observed := inspectLock(runDir)
		if !observed.stale {
			return nil, fmt.Errorf("%w: run is locked by another writer", ErrConflict)
		}
		if err := os.Remove(lockPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reclaim stale run lock: %w", err)
		}
	}
	return nil, fmt.Errorf("%w: run lock contention", ErrConflict)
}

// release removes the lock only when it still carries this handle's token.
func (h *runLockHandle) release() error {
	if h == nil || h.released {
		return nil
	}
	h.released = true
	data, err := readBoundedRegular(h.path, maxLockBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: run lock is unreadable", ErrConflict)
	}
	var lock runLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("%w: run lock is invalid", ErrConflict)
	}
	if lock.Token != h.token {
		return fmt.Errorf("%w: run lock is owned by another writer", ErrConflict)
	}
	if err := os.Remove(h.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove run lock: %w", err)
	}
	return syncDir(filepath.Dir(h.path))
}

// inspectLock reports whether the run has a live or reclaimable lock. A
// malformed lock is judged by its file age.
func inspectLock(runDir string) lockObservation {
	lockPath := filepath.Join(runDir, lockFileName)
	info, err := os.Lstat(lockPath)
	if errors.Is(err, fs.ErrNotExist) {
		return lockObservation{}
	}
	if err != nil {
		return lockObservation{exists: true, live: true}
	}
	observed := lockObservation{exists: true}
	now := time.Now().UTC()
	data, readErr := readBoundedRegular(lockPath, maxLockBytes)
	if readErr == nil {
		var lock runLock
		if json.Unmarshal(data, &lock) == nil && lock.Token != "" && !lock.StartedAt.IsZero() {
			observed.lock = lock
			observed.age = now.Sub(lock.StartedAt)
			if observed.age < 0 {
				observed.age = 0
			}
			sameHost := lock.Hostname != "" && lock.Hostname == hostName()
			deadOnHost := sameHost && lock.PID > 0 && !pidAlive(lock.PID)
			observed.stale = observed.age > LockStaleAfter || deadOnHost
			observed.live = !observed.stale
			return observed
		}
	}
	observed.age = now.Sub(info.ModTime())
	if observed.age < 0 {
		observed.age = 0
	}
	observed.stale = observed.age > LockStaleAfter
	observed.live = !observed.stale
	return observed
}

// lockStateName renders the inspected lock state.
func lockStateName(observed lockObservation) string {
	switch {
	case !observed.exists:
		return LockStateNone
	case observed.stale:
		return LockStateStale
	default:
		return LockStateLive
	}
}

func randomToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("random lock token unavailable")
	}
	return hex.EncodeToString(raw), nil
}

func hostName() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}
