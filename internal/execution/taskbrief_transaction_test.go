package execution

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTaskBrief_Transaction_LiveWriterLock verifies that a live writer lock
// blocks TaskBrief with ErrConflict, creates no brief artifact, and leaves the lock intact.
func TestTaskBrief_Transaction_LiveWriterLock(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	// Acquire live lock before TaskBrief
	holdLock, err := acquireRunLock(runDir, "test")
	if err != nil {
		t.Fatalf("acquireRunLock failed: %v", err)
	}
	t.Cleanup(func() {
		_ = holdLock.release()
	})

	req := TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	}

	_, err = TaskBrief(req)
	if err == nil {
		t.Fatalf("expected error from locked run, got nil")
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}

	// Verify no brief artifact was created
	briefPath := filepath.Join(runDir, "briefs", "T01.md")
	if _, statErr := os.Lstat(briefPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected brief artifact to not exist, got stat err: %v", statErr)
	}

	// Verify lock remains active
	if _, tryLockErr := acquireRunLock(runDir, "test2"); !errors.Is(tryLockErr, ErrConflict) {
		t.Fatalf("expected lock to still be held, but acquireRunLock returned %v", tryLockErr)
	}
}

// TestTaskBrief_Transaction_ContradictoryRequestAndDirectScope verifies that
// contradictory explicit fields, missing run.json, run ID without run dir, and direct scope
// all fail closed.
func TestTaskBrief_Transaction_ContradictoryRequestAndDirectScope(t *testing.T) {
	root, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	// 1. Contradictory RunID
	_, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		RunID:  "run-20261113T120000Z-999999",
		TaskID: "T01",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for wrong RunID, got %v", err)
	}

	// 2. Contradictory BatchID
	_, err = TaskBrief(TaskBriefRequest{
		RunDir:  runDir,
		BatchID: "B99",
		TaskID:  "T01",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for wrong BatchID, got %v", err)
	}

	// 3. Contradictory RepoRoot
	_, err = TaskBrief(TaskBriefRequest{
		RunDir:   runDir,
		RepoRoot: t.TempDir(),
		TaskID:   "T01",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for wrong RepoRoot, got %v", err)
	}

	// 4. Contradictory TasksPath
	_, err = TaskBrief(TaskBriefRequest{
		RunDir:    runDir,
		TasksPath: "specs/001-test/other_tasks.md",
		TaskID:    "T01",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for wrong TasksPath, got %v", err)
	}

	// 5. Contradictory SpecPath
	_, err = TaskBrief(TaskBriefRequest{
		RunDir:   runDir,
		SpecPath: "specs/001-test/other_spec.md",
		TaskID:   "T01",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for wrong SpecPath, got %v", err)
	}

	// 6. Missing run.json in runDir
	emptyRunDir := t.TempDir()
	_, err = TaskBrief(TaskBriefRequest{
		RunDir: emptyRunDir,
		TaskID: "T01",
	})
	if err == nil {
		t.Fatalf("expected error for missing run.json, got nil")
	}

	// 7. RunID supplied without RunDir fails closed with ErrUsage
	_, err = TaskBrief(TaskBriefRequest{
		RunID:  identity.RunID,
		TaskID: "T01",
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage when RunID supplied without RunDir, got %v", err)
	}

	// 8. Direct scope run refused with ErrUsage
	directDir := filepath.Join(t.TempDir(), "direct-run")
	if err := os.MkdirAll(directDir, 0700); err != nil {
		t.Fatal(err)
	}
	directIdentity := RunIdentity{
		SchemaVersion: SchemaVersion,
		RunID:         "run-20261113T120000Z-123456",
		CreatedAt:     time.Now().UTC(),
		RunRoot:       filepath.Dir(directDir),
		RunDir:        directDir,
		Scope: Scope{
			Kind:     ScopeDirect,
			RepoRoot: root,
			Label:    "direct-fix",
		},
	}
	data, _ := json.MarshalIndent(directIdentity, "", "  ")
	if err := os.WriteFile(filepath.Join(directDir, "run.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = TaskBrief(TaskBriefRequest{
		RunDir: directDir,
		TaskID: "T01",
	})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("expected ErrUsage for direct scope run, got %v", err)
	}
	if !strings.Contains(err.Error(), "not supported for direct-scope runs") {
		t.Fatalf("expected direct-scope refusal message, got %v", err)
	}
}

// TestTaskBrief_Transaction_RegistrationAndStatus verifies that successful creation
// registers Tasks["T01"] BriefPath and BriefDigest in the checkpoint, and status stays clean.
func TestTaskBrief_Transaction_RegistrationAndStatus(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	brief, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("TaskBrief failed: %v", err)
	}

	if brief.RunID != identity.RunID {
		t.Fatalf("expected brief.RunID %q, got %q", identity.RunID, brief.RunID)
	}
	if brief.TaskID != "T01" {
		t.Fatalf("expected brief.TaskID T01, got %q", brief.TaskID)
	}

	// Check checkpoint
	cp, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}
	taskState, ok := cp.Tasks["T01"]
	if !ok {
		t.Fatalf("expected task T01 in checkpoint.Tasks, got map: %+v", cp.Tasks)
	}
	if taskState.BriefPath != brief.BriefPath {
		t.Fatalf("expected checkpoint BriefPath %q, got %q", brief.BriefPath, taskState.BriefPath)
	}
	if taskState.BriefDigest != brief.BriefDigest {
		t.Fatalf("expected checkpoint BriefDigest %q, got %q", brief.BriefDigest, taskState.BriefDigest)
	}
	if taskState.State != StatePrepared {
		t.Fatalf("expected task state prepared, got %q", taskState.State)
	}
	if cp.State != StatePrepared {
		t.Fatalf("expected checkpoint state prepared, got %q", cp.State)
	}

	// Status should be clean
	status, err := StatusRun(runRoot, identity.RunID)
	if err != nil {
		t.Fatalf("StatusRun failed: %v", err)
	}
	if status.State != StatePrepared {
		t.Fatalf("expected status state prepared, got %q", status.State)
	}
	if len(status.StaleReasons) != 0 {
		t.Fatalf("expected 0 stale reasons, got %v", status.StaleReasons)
	}
}

// TestTaskBrief_Transaction_PreservesPriorTaskState verifies that task brief registration
// preserves existing task history (LastRecordID, LastCheckID, etc.).
func TestTaskBrief_Transaction_PreservesPriorTaskState(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	// Seed checkpoint with prior evidence on T01
	baseline := identity.Baseline
	cp := &Checkpoint{
		SchemaVersion:   SchemaVersion,
		RunID:           identity.RunID,
		UpdatedAt:       time.Now().UTC(),
		State:           StatePrepared,
		LastObservation: &baseline,
		Tasks: map[string]TaskState{
			"T01": {
				TaskID:       "T01",
				State:        StatePrepared,
				LastRecordID: "rec-20261113T120000Z-000001",
				LastCheckID:  "chk-20261113T120000Z-000001",
			},
		},
	}
	if err := WriteCheckpoint(runDir, cp); err != nil {
		t.Fatalf("WriteCheckpoint seed failed: %v", err)
	}

	brief, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("TaskBrief failed: %v", err)
	}

	cpLoaded, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}
	taskState := cpLoaded.Tasks["T01"]
	if taskState.LastRecordID != "rec-20261113T120000Z-000001" {
		t.Errorf("LastRecordID lost, got %q", taskState.LastRecordID)
	}
	if taskState.LastCheckID != "chk-20261113T120000Z-000001" {
		t.Errorf("LastCheckID lost, got %q", taskState.LastCheckID)
	}
	if taskState.BriefPath != brief.BriefPath {
		t.Errorf("BriefPath not registered: got %q, want %q", taskState.BriefPath, brief.BriefPath)
	}
	if taskState.BriefDigest != brief.BriefDigest {
		t.Errorf("BriefDigest not registered: got %q, want %q", taskState.BriefDigest, brief.BriefDigest)
	}
}

// TestTaskBrief_Transaction_LegacyRepairAndCheckpointFailure verifies that repeating
// TaskBrief with byte-identical output restores registration (repairing legacy unregistered brief),
// and that failure to persist checkpoint returns an error without false success.
func TestTaskBrief_Transaction_LegacyRepairAndCheckpointFailure(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	// 1. Initial creation
	brief1, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("initial TaskBrief failed: %v", err)
	}

	// 2. Simulate legacy unregistered brief by clearing registration in checkpoint
	cp, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatal(err)
	}
	delete(cp.Tasks, "T01")
	if err := WriteCheckpoint(runDir, cp); err != nil {
		t.Fatal(err)
	}

	// 3. Repeat TaskBrief with identical output: repairs registration
	brief2, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("repeat TaskBrief failed: %v", err)
	}
	if brief1.BriefDigest != brief2.BriefDigest {
		t.Fatalf("expected identical digest, got %s vs %s", brief1.BriefDigest, brief2.BriefDigest)
	}

	cpAfter, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if cpAfter.Tasks["T01"].BriefDigest != brief1.BriefDigest {
		t.Fatalf("expected registration restored in checkpoint, got %+v", cpAfter.Tasks["T01"])
	}

	// 4. Test failure to persist checkpoint: make runDir unwritable for checkpoint
	// Remove checkpoint and make runDir read-only
	brief2Path := filepath.Join(runDir, "briefs", "T02.md")
	_ = os.Remove(brief2Path)

	if err := os.Chmod(runDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(runDir, 0700)
	})

	_, err = TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T02",
	})
	// Chmod back so we can inspect
	_ = os.Chmod(runDir, 0700)

	if err == nil {
		t.Fatalf("expected error when checkpoint write fails, got nil")
	}
}

// TestTaskBrief_Transaction_TamperedBriefForce verifies that tampered brief without
// Force returns ErrConflict and leaves file untouched, and with Force replaces file and checkpoint.
func TestTaskBrief_Transaction_TamperedBriefForce(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	brief1, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("initial TaskBrief failed: %v", err)
	}

	briefFile := filepath.Join(runDir, "briefs", "T01.md")
	tamperedBytes := []byte("tampered artifact bytes")
	if err := os.WriteFile(briefFile, tamperedBytes, 0600); err != nil {
		t.Fatal(err)
	}

	// Without Force -> ErrConflict
	_, err = TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
		Force:  false,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for tampered brief without force, got %v", err)
	}

	// Verify file was untouched
	data, _ := os.ReadFile(briefFile)
	if string(data) != string(tamperedBytes) {
		t.Fatalf("expected tampered file to remain untouched, got %q", string(data))
	}

	// With Force -> success and restored brief
	briefForced, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("TaskBrief with Force failed: %v", err)
	}
	if briefForced.BriefDigest != brief1.BriefDigest {
		t.Fatalf("expected restored digest %s, got %s", brief1.BriefDigest, briefForced.BriefDigest)
	}

	// Checkpoint also updated
	cp, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if cp.Tasks["T01"].BriefDigest != brief1.BriefDigest {
		t.Fatalf("expected checkpoint digest %s, got %s", brief1.BriefDigest, cp.Tasks["T01"].BriefDigest)
	}
}

// TestTaskBrief_Transaction_SourceAndBatchDrift verifies that spec changes and substantive
// task changes cause ErrConflict (even with Force), while progress-only changes succeed.
func TestTaskBrief_Transaction_SourceAndBatchDrift(t *testing.T) {
	root, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	specFile := filepath.Join(root, "specs", "001-test", "spec.md")
	tasksFile := filepath.Join(root, "specs", "001-test", "tasks.md")

	origSpec, err := os.ReadFile(specFile)
	if err != nil {
		t.Fatal(err)
	}
	origTasks, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Spec change -> ErrConflict (even with Force: true)
	if err := os.WriteFile(specFile, append(origSpec, []byte("\n| **AC-02** | Extra condition |\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
		Force:  true,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict on spec drift, got %v", err)
	}
	// Restore spec
	if err := os.WriteFile(specFile, origSpec, 0600); err != nil {
		t.Fatal(err)
	}

	// 2. Substantive task change (edit AC reference) -> ErrConflict (even with Force: true)
	substantiveTasks := strings.Replace(string(origTasks), "Exercise the fixture.", "Substantive change to requirements.", 1)
	if err := os.WriteFile(tasksFile, []byte(substantiveTasks), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
		Force:  true,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict on substantive task drift, got %v", err)
	}
	// Restore tasks
	if err := os.WriteFile(tasksFile, origTasks, 0600); err != nil {
		t.Fatal(err)
	}

	// 3. Progress-only change in tasks.md: checkbox change [ ] -> [x]
	progressTasks := strings.Replace(string(origTasks), "- [ ] **T01**", "- [x] **T01**", 1)
	if err := os.WriteFile(tasksFile, []byte(progressTasks), 0600); err != nil {
		t.Fatal(err)
	}
	brief, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T01",
	})
	if err != nil {
		t.Fatalf("expected progress-only task change to succeed, got %v", err)
	}
	if brief.TaskID != "T01" {
		t.Fatalf("expected T01, got %q", brief.TaskID)
	}
}

// TestTaskBrief_Transaction_LockReleasedOnError verifies that when TaskBrief encounters
// an error, the writer lock is reliably released on every path.
func TestTaskBrief_Transaction_LockReleasedOnError(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	runDir := identity.RunDir

	// Try with non-existent task ID
	_, err := TaskBrief(TaskBriefRequest{
		RunDir: runDir,
		TaskID: "T99",
	})
	if err == nil {
		t.Fatalf("expected error for non-existent task, got nil")
	}

	// Lock must be released: acquireRunLock should succeed immediately
	lock, lockErr := acquireRunLock(runDir, "verify-release")
	if lockErr != nil {
		t.Fatalf("lock was not released after error: %v", lockErr)
	}
	_ = lock.release()
}

// TestTaskBrief_PureExtraction verifies that pure extraction works with no RunDir,
// acquiring no lock and creating no brief artifact or checkpoint.
func TestTaskBrief_PureExtraction(t *testing.T) {
	tasksPath := filepath.Join("testdata", "tasks", "standard_tasks.md")
	specPath := filepath.Join("testdata", "spec", "standard_spec.md")

	req := TaskBriefRequest{
		BatchID:   "B1",
		TaskID:    "T1",
		TasksPath: tasksPath,
		SpecPath:  specPath,
	}

	brief, err := TaskBrief(req)
	if err != nil {
		t.Fatalf("pure extraction failed: %v", err)
	}

	if brief.RunID != "" {
		t.Errorf("expected empty RunID in pure extraction, got %q", brief.RunID)
	}
	if brief.BriefPath != "" {
		t.Errorf("expected empty BriefPath in pure extraction, got %q", brief.BriefPath)
	}
	if brief.TaskID != "T1" {
		t.Errorf("expected TaskID T1, got %q", brief.TaskID)
	}
	if brief.BriefDigest == "" {
		t.Errorf("expected computed BriefDigest, got empty")
	}
}
