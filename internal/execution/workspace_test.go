package execution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// workspaceTestRun creates one repository-backed run and returns the repository
// root, its run root and the run identity.
func workspaceTestRun(t *testing.T) (string, string, RunIdentity) {
	t.Helper()
	root := repoTestInit(t)
	featureDir := repoTestFeature(t, root)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	scope, err := FeatureScope(repo, featureDir, "B1")
	if err != nil {
		t.Fatalf("feature scope: %v", err)
	}
	identity, err := InitRun(InitRequest{Scope: scope, ToolVersion: "test"})
	if err != nil {
		t.Fatalf("init run: %v", err)
	}
	return root, identity.RunRoot, identity
}

func TestRunIDFormat(t *testing.T) {
	now := time.Date(2026, time.November, 13, 11, 45, 0, 0, time.UTC)
	id, err := NewRunID(now)
	if err != nil {
		t.Fatalf("new run ID: %v", err)
	}
	if !strings.HasPrefix(id, "run-20261113T114500Z-") {
		t.Fatalf("run ID = %q", id)
	}
	if err := ValidateRunID(id); err != nil {
		t.Fatalf("generated run ID is invalid: %v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, err := NewRunID(now)
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate run ID %q", id)
		}
		seen[id] = true
	}
}

func TestValidateRunID(t *testing.T) {
	valid := []string{"run-20261113T114500Z-a1b2c3", "run-00010101T000000Z-000000"}
	for _, id := range valid {
		if err := ValidateRunID(id); err != nil {
			t.Errorf("ValidateRunID(%q) = %v, want nil", id, err)
		}
	}
	invalid := []string{
		"",
		"run-20261113T114500Z-A1B2C3",
		"run-20261113T114500Z-a1b2c",
		"run-20261113T114500-a1b2c3",
		"../run-20261113T114500Z-a1b2c3",
		"run-20261113T114500Z-a1b2c3/../x",
		"chk-20261113T114500Z-a1b2c3",
	}
	for _, id := range invalid {
		if err := ValidateRunID(id); !errors.Is(err, ErrUsage) {
			t.Errorf("ValidateRunID(%q) = %v, want ErrUsage", id, err)
		}
	}
}

func TestWorkspaceInitCreatesPrivateIgnoredRun(t *testing.T) {
	root, runRoot, identity := workspaceTestRun(t)
	runDir := identity.RunDir
	if want := filepath.Join(root, ".agentic-sdd", "runs"); runRoot != want {
		t.Fatalf("run root = %q, want %q", runRoot, want)
	}
	if identity.RunRoot != runRoot {
		t.Fatalf("identity run root = %q", identity.RunRoot)
	}
	if identity.WorkspaceGitVisible {
		t.Fatal("repository-local run root must be reported Git-invisible")
	}
	if identity.ApprovalReference != "specs/001-test/spec.md §Approval" {
		t.Fatalf("approval reference = %q", identity.ApprovalReference)
	}
	info, err := os.Stat(runDir)
	if err != nil {
		t.Fatalf("stat run dir: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("run dir mode = %o, want 700", info.Mode().Perm())
	}
	for _, name := range []string{"run.json", "checkpoint.json"} {
		info, err := os.Stat(filepath.Join(runDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", name, info.Mode().Perm())
		}
	}
	ignorePath := filepath.Join(runRoot, ".gitignore")
	if data := repoTestRead(t, ignorePath); data != "*" {
		t.Fatalf("run-root ignore file = %q, want %q", data, "*")
	}
	status := repoTestGit(t, root, "status", "--porcelain", "--untracked-files=all")
	if status != "" {
		t.Fatalf("worktree is not clean after run creation:\n%s", status)
	}
	loaded, err := LoadRunIdentity(runDir)
	if err != nil {
		t.Fatalf("load run identity: %v", err)
	}
	if loaded.RunID != identity.RunID || loaded.Scope.BatchID != "B1" {
		t.Fatalf("loaded identity = %+v", loaded)
	}
	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StatePrepared || checkpoint.RunID != identity.RunID {
		t.Fatalf("initial checkpoint = %+v", checkpoint)
	}
}

func TestWorkspaceInitFailClosedOnVisibleIgnore(t *testing.T) {
	root := repoTestInit(t)
	featureDir := repoTestFeature(t, root)
	runsRoot := filepath.Join(root, ".agentic-sdd", "runs")
	repoTestWrite(t, filepath.Join(runsRoot, ".gitignore"), "keep.txt\n")
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := FeatureScope(repo, featureDir, "B1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = InitRun(InitRequest{Scope: scope, ToolVersion: "test"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("InitRun with a non-ignoring run root error = %v, want ErrUsage", err)
	}
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "run-") {
			t.Fatalf("failing init left a run directory %q", entry.Name())
		}
	}
}

func TestWorkspaceInitRequiresGitRepository(t *testing.T) {
	dir := t.TempDir()
	scope := Scope{Kind: ScopeDirect, RepoRoot: dir}
	if _, err := InitRun(InitRequest{Scope: scope}); !errors.Is(err, ErrUsage) {
		t.Fatalf("InitRun(non-repository) error = %v, want ErrUsage", err)
	}
}

func TestWorkspaceInitRejectsUnbornHead(t *testing.T) {
	root := repoTestInitEmpty(t)
	scope := Scope{Kind: ScopeDirect, RepoRoot: root}
	if _, err := InitRun(InitRequest{Scope: scope}); !errors.Is(err, ErrUsage) {
		t.Fatalf("InitRun(unborn HEAD) error = %v, want ErrUsage", err)
	}
}

func TestWorkspaceInitRejectsScopeRootSubdirectory(t *testing.T) {
	root := repoTestInit(t)
	repoTestFeature(t, root)
	scope := Scope{Kind: ScopeDirect, RepoRoot: filepath.Join(root, "specs")}
	if _, err := InitRun(InitRequest{Scope: scope}); !errors.Is(err, ErrConflict) {
		t.Fatalf("InitRun(subdirectory scope root) error = %v, want ErrConflict", err)
	}
}

func TestWorkspaceInitExplicitRootOutsideRepository(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := DirectScope(repo, "direct fix")
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runRoot := filepath.Join(outside, "private-runs")
	identity, err := InitRun(InitRequest{Scope: scope, RunRoot: runRoot, ToolVersion: "test"})
	if err != nil {
		t.Fatalf("init run with explicit root: %v", err)
	}
	if filepath.Clean(identity.RunRoot) != filepath.Clean(runRoot) {
		t.Fatalf("identity run root = %q, want %q", identity.RunRoot, runRoot)
	}
	if identity.WorkspaceGitVisible {
		t.Fatal("an explicit run root outside the repository cannot be Git-visible")
	}
	if _, err := os.Stat(filepath.Join(root, ".agentic-sdd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("explicit root must not create repository scratch: %v", err)
	}
	if _, err := os.Stat(identity.RunDir); err != nil {
		t.Fatalf("run dir missing: %v", err)
	}
}

func TestWorkspaceInitExplicitRootInsideRepositoryReportsVisibility(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := DirectScope(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	runRoot := filepath.Join(root, "scratch-runs")
	identity, err := InitRun(InitRequest{Scope: scope, RunRoot: runRoot, ToolVersion: "test"})
	if err != nil {
		t.Fatalf("init run with visible root: %v", err)
	}
	if !identity.WorkspaceGitVisible {
		t.Fatal("an unignored run root inside the repository must be reported Git-visible")
	}
	status := repoTestGit(t, root, "status", "--porcelain")
	if !strings.Contains(status, "?? scratch-runs/") {
		t.Fatalf("expected visible scratch in Git status, got:\n%s", status)
	}
	observed, err := StatusRun(runRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if observed.State != StatePrepared || len(observed.StaleReasons) != 0 {
		t.Fatalf("visible scratch made its own run stale: state=%q reasons=%v", observed.State, observed.StaleReasons)
	}
}

func TestWorkspaceInitRefusesSymlinkedScratchParent(t *testing.T) {
	root := repoTestInit(t)
	repoTestFeature(t, root)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	featureDir := filepath.Join(root, "specs", "001-test")
	scope, err := FeatureScope(repo, featureDir, "B1")
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".agentic-sdd")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := InitRun(InitRequest{Scope: scope}); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("InitRun through a symlinked scratch parent error = %v, want ErrUnsafePath", err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("scratch was created through the symlink: %v", entries)
	}
}

func TestWorkspaceInitRefusesSymlinkRunRoot(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := DirectScope(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "linked-runs")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := InitRun(InitRequest{Scope: scope, RunRoot: link}); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("InitRun(symlink run root) error = %v, want ErrUnsafePath", err)
	}
}

func TestResolveRunRoot(t *testing.T) {
	root := repoTestInit(t)
	def, err := ResolveRunRoot(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if def != filepath.Join(root, ".agentic-sdd", "runs") {
		t.Fatalf("default run root = %q", def)
	}
	explicit := filepath.Join(root, "custom")
	got, err := ResolveRunRoot(root, explicit)
	if err != nil {
		t.Fatal(err)
	}
	if got != explicit {
		t.Fatalf("explicit run root = %q, want %q", got, explicit)
	}
	if _, err := ResolveRunRoot("", ""); !errors.Is(err, ErrUsage) {
		t.Fatalf("ResolveRunRoot(no repository) error = %v, want ErrUsage", err)
	}
}

func TestLockConflictAndRelease(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir
	lock, err := acquireRunLock(runDir, "test")
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if observed := inspectLock(runDir); !observed.exists || !observed.live {
		t.Fatalf("lock observation = %+v, want live", observed)
	}
	if _, err := acquireRunLock(runDir, "test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second acquire error = %v, want ErrConflict", err)
	}
	if err := lock.release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
	if observed := inspectLock(runDir); observed.exists {
		t.Fatalf("lock still exists after release: %+v", observed)
	}
	again, err := acquireRunLock(runDir, "test")
	if err != nil {
		t.Fatalf("reacquire lock: %v", err)
	}
	if err := again.release(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestLockOwnershipCheckedRelease(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir
	lockPath := filepath.Join(runDir, lockFileName)
	payload := fmt.Sprintf(`{"schemaVersion":1,"token":"other-writer","pid":%d,"hostname":%q,"started_at":%q,"operation":"check"}`,
		os.Getpid(), hostName(), time.Now().UTC().Format(time.RFC3339Nano))
	if err := writeFileAtomic(lockPath, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	handle := &runLockHandle{path: lockPath, token: "this-writer"}
	if err := handle.release(); !errors.Is(err, ErrConflict) {
		t.Fatalf("release of a foreign lock error = %v, want ErrConflict", err)
	}
	if _, err := os.Lstat(lockPath); err != nil {
		t.Fatalf("foreign lock was removed: %v", err)
	}
}

func TestLockStaleReclaim(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir
	lockPath := filepath.Join(runDir, lockFileName)
	t.Run("dead pid on this host", func(t *testing.T) {
		payload := fmt.Sprintf(`{"schemaVersion":1,"token":"dead","pid":2147483647,"hostname":%q,"started_at":%q,"operation":"check"}`,
			hostName(), time.Now().UTC().Format(time.RFC3339Nano))
		if err := writeFileAtomic(lockPath, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		if observed := inspectLock(runDir); !observed.stale {
			t.Fatalf("dead-pid lock observation = %+v, want stale", observed)
		}
		lock, err := acquireRunLock(runDir, "test")
		if err != nil {
			t.Fatalf("reclaim dead-pid lock: %v", err)
		}
		if err := lock.release(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("aged foreign-host lock", func(t *testing.T) {
		started := time.Now().UTC().Add(-31 * time.Minute)
		payload := fmt.Sprintf(`{"schemaVersion":1,"token":"old","pid":%d,"hostname":"another-host","started_at":%q,"operation":"check"}`,
			os.Getpid(), started.Format(time.RFC3339Nano))
		if err := writeFileAtomic(lockPath, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		lock, err := acquireRunLock(runDir, "test")
		if err != nil {
			t.Fatalf("reclaim aged lock: %v", err)
		}
		if err := lock.release(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("live foreign-host lock is a conflict", func(t *testing.T) {
		payload := fmt.Sprintf(`{"schemaVersion":1,"token":"live","pid":%d,"hostname":"another-host","started_at":%q,"operation":"check"}`,
			os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
		if err := writeFileAtomic(lockPath, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := acquireRunLock(runDir, "test"); !errors.Is(err, ErrConflict) {
			t.Fatalf("acquire over a live foreign lock error = %v, want ErrConflict", err)
		}
		os.Remove(lockPath)
	})
}

func TestCleanPreviewApplyAndAbsent(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	preview, err := CleanRun(runRoot, identity.RunID, false)
	if err != nil {
		t.Fatalf("clean preview: %v", err)
	}
	if preview.Applied || preview.Removed || preview.AlreadyAbsent {
		t.Fatalf("preview result = %+v", preview)
	}
	if preview.FileCount < 2 || preview.TotalBytes == 0 {
		t.Fatalf("preview counts = %d/%d", preview.FileCount, preview.TotalBytes)
	}
	if !repoTestContains(preview.Entries, "run.json") {
		t.Fatalf("preview entries = %v", preview.Entries)
	}
	if _, err := os.Stat(identity.RunDir); err != nil {
		t.Fatalf("preview removed the run: %v", err)
	}
	applied, err := CleanRun(runRoot, identity.RunID, true)
	if err != nil {
		t.Fatalf("clean apply: %v", err)
	}
	if !applied.Applied || !applied.Removed {
		t.Fatalf("apply result = %+v", applied)
	}
	if _, err := os.Stat(identity.RunDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("run directory still exists: %v", err)
	}
	if _, err := os.Stat(runRoot); err != nil {
		t.Fatalf("run root was removed: %v", err)
	}
	if repoTestRead(t, filepath.Join(runRoot, ".gitignore")) != "*" {
		t.Fatal("run-root ignore file was removed or rewritten")
	}
	absent, err := CleanRun(runRoot, identity.RunID, true)
	if err != nil {
		t.Fatalf("clean absent: %v", err)
	}
	if !absent.AlreadyAbsent || absent.Removed {
		t.Fatalf("absent result = %+v", absent)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-runs")
	absentRoot, err := CleanRun(missingRoot, identity.RunID, true)
	if err != nil {
		t.Fatalf("clean with a missing run root: %v", err)
	}
	if !absentRoot.AlreadyAbsent || absentRoot.Removed {
		t.Fatalf("missing-root result = %+v", absentRoot)
	}
	if _, err := os.Stat(missingRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("clean created the missing run root: %v", err)
	}
}

func TestCleanRefusesLiveLock(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	lock, err := acquireRunLock(identity.RunDir, "check")
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if _, err := CleanRun(runRoot, identity.RunID, true); !errors.Is(err, ErrConflict) {
		t.Fatalf("clean under a live lock error = %v, want ErrConflict", err)
	}
	if _, err := os.Stat(identity.RunDir); err != nil {
		t.Fatalf("run directory was removed under a live lock: %v", err)
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	if _, err := CleanRun(runRoot, identity.RunID, true); err != nil {
		t.Fatalf("clean after release: %v", err)
	}
}

func TestCleanRefusesUnownedAndSymlinkedRuns(t *testing.T) {
	runRoot := t.TempDir()
	runID := "run-20261113T114500Z-abcdef"
	unowned := filepath.Join(runRoot, runID)
	if err := os.Mkdir(unowned, 0o700); err != nil {
		t.Fatal(err)
	}
	repoTestWrite(t, filepath.Join(unowned, "notes.txt"), "not a run\n")
	if _, err := CleanRun(runRoot, runID, true); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("clean of an unowned directory error = %v, want ErrUnsafePath", err)
	}
	if _, err := os.Stat(unowned); err != nil {
		t.Fatalf("unowned directory was removed: %v", err)
	}

	linkRunRoot := t.TempDir()
	target := t.TempDir()
	linkID := "run-20261113T114501Z-abcdef"
	if err := os.Symlink(target, filepath.Join(linkRunRoot, linkID)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := CleanRun(linkRunRoot, linkID, true); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("clean of a symlinked run error = %v, want ErrUnsafePath", err)
	}
}

func TestCleanRefusesMovedRun(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	otherRoot := t.TempDir()
	moved := filepath.Join(otherRoot, identity.RunID)
	if err := os.Rename(identity.RunDir, moved); err != nil {
		t.Fatal(err)
	}
	if _, err := CleanRun(otherRoot, identity.RunID, true); err == nil {
		t.Fatal("clean of a moved run must be refused")
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("moved run directory was removed: %v", err)
	}
}

func repoTestContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
