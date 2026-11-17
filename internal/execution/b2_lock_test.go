package execution

// Feature 022 B2 (T04/T05, plan §3.2 and §12.1 execution part): the per-run guard
// lives under <runRoot>/.locks/<runID>, is taken before the run directory is
// trusted, and the legacy <runDir>/.lock is reclaimed only by identity (a dead
// holder on this host), never by age.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// b2GuardPath is the per-run guard path under the run root.
func b2GuardPath(identity RunIdentity) string {
	return filepath.Join(identity.RunRoot, ".locks", identity.RunID)
}

// b2WriteLegacyLock writes a legacy-format run lock for one test scenario.
func b2WriteLegacyLock(t *testing.T, runDir string, pid int, host string, started time.Time, raw string) {
	t.Helper()
	payload := raw
	if payload == "" {
		payload = fmt.Sprintf(`{"schemaVersion":1,"token":"b2-foreign","pid":%d,"hostname":%q,"started_at":%q,"operation":"check"}`,
			pid, host, started.UTC().Format(time.RFC3339Nano))
	}
	if err := writeFileAtomic(filepath.Join(runDir, lockFileName), []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestB2GuardLivesUnderRunRoot: the held guard is <runRoot>/.locks/<runID>, not a
// path inside the run directory, and a second writer is refused.
func TestB2GuardLivesUnderRunRoot(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	lock, err := acquireRunLock(identity.RunDir, "test")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	info, err := os.Lstat(b2GuardPath(identity))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("guard under run root: info=%v err=%v", info, err)
	}
	if _, err := os.Lstat(filepath.Join(identity.RunDir, ".locks")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("guard directory inside the run directory: %v", err)
	}
	if _, err := acquireRunLock(identity.RunDir, "test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("second writer error = %v, want ErrConflict", err)
	}
	if err := lock.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
}

// TestB2RunIDRejectsGuardDirectoryName: the guard directory name can never be a
// run identifier, so enumeration and clean cannot mistake it for a run.
func TestB2RunIDRejectsGuardDirectoryName(t *testing.T) {
	if err := ValidateRunID(".locks"); !errors.Is(err, ErrUsage) {
		t.Fatalf("ValidateRunID(.locks) = %v, want ErrUsage", err)
	}
}

// TestB2AgedLiveHolderIsNotReclaimed: a legacy lock whose holder is alive on this
// host stays a conflict however old its timestamp is.
func TestB2AgedLiveHolderIsNotReclaimed(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	b2WriteLegacyLock(t, identity.RunDir, os.Getpid(), hostName(), time.Now().UTC().Add(-2*time.Hour), "")
	if observed := inspectLock(identity.RunDir); observed.stale {
		t.Fatalf("an aged lock with a live same-host holder is stale: %+v", observed)
	}
	if _, err := acquireRunLock(identity.RunDir, "test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("acquire over an aged live holder error = %v, want ErrConflict", err)
	}
	if _, err := os.Lstat(filepath.Join(identity.RunDir, lockFileName)); err != nil {
		t.Fatalf("the aged live holder's lock was removed: %v", err)
	}
	_ = os.Remove(filepath.Join(identity.RunDir, lockFileName))
}

// TestB2AgedForeignHostLockIsConflict: a foreign-host lock cannot be proven dead
// from this host, so age alone never reclaims it.
func TestB2AgedForeignHostLockIsConflict(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	b2WriteLegacyLock(t, identity.RunDir, 2147483647, "another-host", time.Now().UTC().Add(-2*time.Hour), "")
	if _, err := acquireRunLock(identity.RunDir, "test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("acquire over an aged foreign-host lock error = %v, want ErrConflict", err)
	}
	_ = os.Remove(filepath.Join(identity.RunDir, lockFileName))
}

// TestB2UnknownLegacyLockBlocks: an unparseable legacy lock is unknown ownership
// and blocks; it is neither reclaimed nor deleted.
func TestB2UnknownLegacyLockBlocks(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	b2WriteLegacyLock(t, identity.RunDir, 0, "", time.Time{}, "not a lock document")
	if _, err := acquireRunLock(identity.RunDir, "test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("acquire over an unknown legacy lock error = %v, want ErrConflict", err)
	}
	if _, err := os.Lstat(filepath.Join(identity.RunDir, lockFileName)); err != nil {
		t.Fatalf("the unknown legacy lock was removed: %v", err)
	}
	_ = os.Remove(filepath.Join(identity.RunDir, lockFileName))
}

// TestB2CleanApplyRetiresGuard: a completed clean removes the run and its guard,
// while holding the guard through deletion.
func TestB2CleanApplyRetiresGuard(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	result, err := CleanRun(identity.RunRoot, identity.RunID, true)
	if err != nil || !result.Removed {
		t.Fatalf("clean apply = %+v, %v; want removed", result, err)
	}
	if _, err := os.Lstat(b2GuardPath(identity)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("guard survived clean: %v", err)
	}
}

// TestB2GuardRefusesSymlinkLeaf: a symlink planted at the guard leaf is never
// followed or truncated, and acquisition fails closed.
func TestB2GuardRefusesSymlinkLeaf(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	// InitRun already created this run's guard file; replace it with the link.
	if err := os.Remove(b2GuardPath(identity)); err != nil {
		t.Fatalf("remove init-created guard: %v", err)
	}
	if err := os.Symlink(victim, b2GuardPath(identity)); err != nil {
		t.Fatalf("plant guard symlink: %v", err)
	}
	if _, err := acquireRunLock(identity.RunDir, "test"); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("acquire through a symlinked guard error = %v, want ErrUnsafePath", err)
	}
	data, err := os.ReadFile(victim)
	if err != nil || string(data) != "keep" {
		t.Fatalf("the symlink target was modified: %q, %v", data, err)
	}
}

// TestB2MissingRunIsNotFoundWithoutGuard: a missing run is reported as not found
// and leaves no guard behind for that identifier.
func TestB2MissingRunIsNotFoundWithoutGuard(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	ghost := filepath.Join(identity.RunRoot, "run-20000101T000000Z-aaaaaa")
	if _, err := acquireRunLock(ghost, "test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("acquire on a missing run error = %v, want ErrNotFound", err)
	}
	if _, err := os.Lstat(filepath.Join(identity.RunRoot, ".locks", filepath.Base(ghost))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a guard was created for a missing run: %v", err)
	}
}

// TestB2WriteCheckpointSelfAcquires: the public checkpoint write takes the run
// guard itself, so it refuses to write while another writer holds the run.
func TestB2WriteCheckpointSelfAcquires(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	lock, err := acquireRunLock(identity.RunDir, "test")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer func() { _ = lock.release() }()
	checkpoint := &Checkpoint{SchemaVersion: SchemaVersion, RunID: identity.RunID, State: StatePrepared}
	if err := WriteCheckpoint(identity.RunDir, checkpoint); !errors.Is(err, ErrConflict) {
		t.Fatalf("WriteCheckpoint under a held run error = %v, want ErrConflict", err)
	}
}

// TestB2LockedCheckpointRejectsStaleHandle: a nested write rejects a released
// handle and a handle for another run directory.
func TestB2LockedCheckpointRejectsStaleHandle(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	checkpoint := &Checkpoint{SchemaVersion: SchemaVersion, RunID: identity.RunID, State: StatePrepared}

	lock, err := acquireRunLock(identity.RunDir, "test")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := writeCheckpointLocked(lock, identity.RunDir, checkpoint); err != nil {
		t.Fatalf("held handle write: %v", err)
	}
	if err := writeCheckpointLocked(lock, filepath.Join(identity.RunRoot, "elsewhere"), checkpoint); !errors.Is(err, ErrConflict) {
		t.Fatalf("handle for another directory error = %v, want ErrConflict", err)
	}
	if err := lock.release(); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := writeCheckpointLocked(lock, identity.RunDir, checkpoint); !errors.Is(err, ErrConflict) {
		t.Fatalf("released handle write error = %v, want ErrConflict", err)
	}
}
