package execution

// Task-brief provenance and freshness regressions for Feature 017. A brief is
// trusted only through a registered path and digest; discovery is bounded and
// contained, and any unsafe or unreadable layout blocks a clean status. Fixtures
// are disposable temporary repositories.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func briefTestRun(t *testing.T) RunIdentity {
	t.Helper()
	_, _, identity := workspaceTestRun(t)
	return identity
}

func briefTestLoadCheckpoint(t *testing.T, runDir string) *Checkpoint {
	t.Helper()
	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	return checkpoint
}

func briefTestWriteArtifact(t *testing.T, runDir, name, content string) string {
	t.Helper()
	if err := ensureArtifactDir(runDir, briefArtifactDirName); err != nil {
		t.Fatalf("create brief directory: %v", err)
	}
	path := filepath.Join(runDir, briefArtifactDirName, name)
	if err := writeFileAtomic(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write brief artifact: %v", err)
	}
	return path
}

func briefTestBrief(runID, taskID, path, content string) Brief {
	return Brief{
		SchemaVersion: SchemaVersion,
		Kind:          "execution.task.brief",
		RunID:         runID,
		TaskID:        taskID,
		BriefPath:     path,
		BriefDigest:   DigestBytes([]byte(content)),
		GeneratedAt:   time.Now().UTC(),
	}
}

// briefTestRegister writes one designated artifact and registers it in a fresh
// checkpoint, mirroring what a successful run-backed brief operation persists.
func briefTestRegister(t *testing.T, identity RunIdentity, taskID, content string) string {
	t.Helper()
	path := briefTestWriteArtifact(t, identity.RunDir, taskID+".md", content)
	checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
	if err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, taskID, path, content)); err != nil {
		t.Fatalf("register brief: %v", err)
	}
	if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
		t.Fatalf("persist checkpoint: %v", err)
	}
	return path
}

func briefTestInspect(t *testing.T, runDir string, checkpoint *Checkpoint) []string {
	t.Helper()
	reasons, err := inspectBriefArtifacts(runDir, checkpoint)
	if err != nil {
		t.Fatalf("inspect brief artifacts: %v", err)
	}
	return reasons
}

func briefTestReasons(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("reasons = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("reasons = %v, want %v", got, want)
		}
	}
}

func TestBriefRegisterContract(t *testing.T) {
	t.Run("preserves existing task state and history", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T01\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T01.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StateAwaitingVerification, LastCheckID: "chk-1", LastRecordID: "rec-1"},
		}
		if err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, "T01", path, content)); err != nil {
			t.Fatalf("register brief: %v", err)
		}
		task := checkpoint.Tasks["T01"]
		if task.State != StateAwaitingVerification || task.LastCheckID != "chk-1" || task.LastRecordID != "rec-1" {
			t.Fatalf("registration changed task state or history: %+v", task)
		}
		if task.BriefPath != path || task.BriefDigest != DigestBytes([]byte(content)) {
			t.Fatalf("registration recorded %+v", task)
		}
		checkpoint.State = StateAwaitingVerification
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		reloaded := briefTestLoadCheckpoint(t, identity.RunDir)
		persisted := reloaded.Tasks["T01"]
		if persisted.State != StateAwaitingVerification || persisted.LastCheckID != "chk-1" || persisted.LastRecordID != "rec-1" {
			t.Fatalf("persisted registration lost task state or history: %+v", persisted)
		}
		if persisted.BriefPath != path || persisted.BriefDigest != DigestBytes([]byte(content)) {
			t.Fatalf("persisted brief reference = %+v", persisted)
		}
	})

	t.Run("initializes an absent task as prepared", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T02\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T02.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		if err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, "T02", path, content)); err != nil {
			t.Fatalf("register brief: %v", err)
		}
		task := checkpoint.Tasks["T02"]
		if task.TaskID != "T02" || task.State != StatePrepared {
			t.Fatalf("initialized task = %+v, want T02 prepared", task)
		}
	})

	t.Run("accepts the designated path in relative form", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T03\n"
		briefTestWriteArtifact(t, identity.RunDir, "T03.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		relative := filepath.Join(briefArtifactDirName, "T03.md")
		if err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, "T03", relative, content)); err != nil {
			t.Fatalf("register relative brief path: %v", err)
		}
		want := filepath.Join(identity.RunDir, briefArtifactDirName, "T03.md")
		if got := checkpoint.Tasks["T03"].BriefPath; got != want {
			t.Fatalf("recorded path = %q, want %q", got, want)
		}
	})

	t.Run("refuses a brief from another run", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T01\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T01.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(foreignCheckpointRunID, "T01", path, content))
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("foreign brief error = %v, want ErrConflict", err)
		}
	})

	t.Run("refuses a path that is not the designated artifact", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T02\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T02.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, "T01", path, content))
		if !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("mismatched designated path error = %v, want ErrUnsafePath", err)
		}
	})

	t.Run("refuses a digest that does not match the artifact", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Brief T01\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T01.md", content)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		brief := briefTestBrief(identity.RunID, "T01", path, content)
		brief.BriefDigest = DigestBytes([]byte("different bytes"))
		if err := registerTaskBrief(checkpoint, identity.RunDir, brief); !errors.Is(err, ErrConflict) {
			t.Fatalf("mismatched digest error = %v, want ErrConflict", err)
		}
	})

	t.Run("refuses a missing or incomplete artifact", func(t *testing.T) {
		identity := briefTestRun(t)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		missing := filepath.Join(identity.RunDir, briefArtifactDirName, "T01.md")
		if err := registerTaskBrief(checkpoint, identity.RunDir, briefTestBrief(identity.RunID, "T01", missing, "content")); !errors.Is(err, ErrConflict) {
			t.Fatalf("missing artifact error = %v, want ErrConflict", err)
		}
		path := briefTestWriteArtifact(t, identity.RunDir, "T02.md", "content")
		brief := briefTestBrief(identity.RunID, "T02", path, "content")
		brief.BriefDigest = ""
		if err := registerTaskBrief(checkpoint, identity.RunDir, brief); !errors.Is(err, ErrUsage) {
			t.Fatalf("incomplete brief error = %v, want ErrUsage", err)
		}
	})
}

func TestBriefInspectArtifactsDiagnosesDrift(t *testing.T) {
	t.Run("matching registered brief is clean", func(t *testing.T) {
		identity := briefTestRun(t)
		briefTestRegister(t, identity, "T01", "# Brief T01\n")
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons)
	})

	t.Run("deleted registered brief reports missing artifact", func(t *testing.T) {
		identity := briefTestRun(t)
		path := briefTestRegister(t, identity, "T01", "# Brief T01\n")
		if err := os.Remove(path); err != nil {
			t.Fatalf("delete brief: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonMissingArtifact)
	})

	t.Run("changed bytes report a digest mismatch", func(t *testing.T) {
		identity := briefTestRun(t)
		path := briefTestRegister(t, identity, "T01", "# Brief T01\n")
		if err := os.WriteFile(path, []byte("# Tampered\n"), 0o600); err != nil {
			t.Fatalf("tamper brief: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefDigestMismatch)
	})

	t.Run("unregistered regular artifact is not trusted", func(t *testing.T) {
		identity := briefTestRun(t)
		briefTestWriteArtifact(t, identity.RunDir, "legacy.md", "# Legacy\n")
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonUnregisteredBrief)
	})

	t.Run("registered symlink is unsafe", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Target\n"
		target := filepath.Join(identity.RunDir, "target.md")
		if err := writeFileAtomic(target, []byte(content), 0o600); err != nil {
			t.Fatalf("write symlink target: %v", err)
		}
		registered := filepath.Join(identity.RunDir, briefArtifactDirName, "T01.md")
		if err := ensureArtifactDir(identity.RunDir, briefArtifactDirName); err != nil {
			t.Fatalf("create brief directory: %v", err)
		}
		if err := os.Symlink(target, registered); err != nil {
			t.Fatalf("create brief symlink: %v", err)
		}
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StatePrepared, BriefPath: registered, BriefDigest: DigestBytes([]byte(content))},
		}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefInspectionFailed)
	})

	t.Run("registered non-regular artifact is unsafe", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := ensureArtifactDir(identity.RunDir, briefArtifactDirName); err != nil {
			t.Fatalf("create brief directory: %v", err)
		}
		registered := filepath.Join(identity.RunDir, briefArtifactDirName, "T01.md")
		if err := os.Mkdir(registered, 0o700); err != nil {
			t.Fatalf("create brief directory artifact: %v", err)
		}
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StatePrepared, BriefPath: registered, BriefDigest: DigestBytes([]byte("x"))},
		}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefInspectionFailed)
	})

	t.Run("unreadable registered artifact is unsafe", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission failure is not observable as root")
		}
		identity := briefTestRun(t)
		path := briefTestRegister(t, identity, "T01", "# Brief T01\n")
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatalf("restrict brief permissions: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefInspectionFailed)
	})

	t.Run("digest without a registered path is unsafe", func(t *testing.T) {
		identity := briefTestRun(t)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StatePrepared, BriefDigest: DigestBytes([]byte("x"))},
		}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefInspectionFailed)
	})

	t.Run("registered path outside briefs is an unsafe layout error", func(t *testing.T) {
		identity := briefTestRun(t)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StatePrepared, BriefPath: filepath.Join(identity.RunDir, "T01.md"), BriefDigest: DigestBytes([]byte("x"))},
		}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		reasons, err := inspectBriefArtifacts(identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		if !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("outside-briefs error = %v, want ErrUnsafePath", err)
		}
		if len(reasons) != 0 {
			t.Fatalf("unsafe layout returned reasons %v with an error", reasons)
		}
	})

	t.Run("brief directory symlink is an unsafe layout error", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := os.Remove(filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("remove brief directory: %v", err)
		}
		target := filepath.Join(identity.RunDir, "briefs-target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatalf("create brief target: %v", err)
		}
		if err := os.Symlink("briefs-target", filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("create brief directory symlink: %v", err)
		}
		if _, err := inspectBriefArtifacts(identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir)); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("brief directory symlink error = %v, want ErrUnsafePath", err)
		}
	})

	t.Run("unregistered symlink entry is unsafe", func(t *testing.T) {
		identity := briefTestRun(t)
		target := filepath.Join(identity.RunDir, "target.md")
		if err := writeFileAtomic(target, []byte("# Target\n"), 0o600); err != nil {
			t.Fatalf("write symlink target: %v", err)
		}
		if err := ensureArtifactDir(identity.RunDir, briefArtifactDirName); err != nil {
			t.Fatalf("create brief directory: %v", err)
		}
		if err := os.Symlink(target, filepath.Join(identity.RunDir, briefArtifactDirName, "legacy.md")); err != nil {
			t.Fatalf("create legacy symlink: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons, ReasonBriefInspectionFailed)
	})

	t.Run("legacy run without a briefs directory is clean", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := os.Remove(filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("remove brief directory: %v", err)
		}
		reasons := briefTestInspect(t, identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir))
		briefTestReasons(t, reasons)
	})

	t.Run("entry budget exhaustion fails closed", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := ensureArtifactDir(identity.RunDir, briefArtifactDirName); err != nil {
			t.Fatalf("create brief directory: %v", err)
		}
		dir := filepath.Join(identity.RunDir, briefArtifactDirName)
		for i := 0; i < maxBriefArtifacts+1; i++ {
			name := filepath.Join(dir, fmt.Sprintf("artifact-%04d.md", i))
			if err := os.WriteFile(name, nil, 0o600); err != nil {
				t.Fatalf("create budget artifact %d: %v", i, err)
			}
		}
		if _, err := inspectBriefArtifacts(identity.RunDir, briefTestLoadCheckpoint(t, identity.RunDir)); !errors.Is(err, ErrUnsafePath) {
			t.Fatalf("entry budget error = %v, want ErrUnsafePath", err)
		}
	})
}

func TestStatusBriefFreshness(t *testing.T) {
	t.Run("registered matching brief keeps the planned state", func(t *testing.T) {
		identity := briefTestRun(t)
		briefTestRegister(t, identity, "T01", "# Brief T01\n")
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.State = StateAwaitingVerification
		task := checkpoint.Tasks["T01"]
		task.State = StateAwaitingVerification
		checkpoint.Tasks["T01"] = task
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StateAwaitingVerification || len(status.StaleReasons) != 0 {
			t.Fatalf("registered brief = %q with reasons %v, want awaiting-verification with none", status.State, status.StaleReasons)
		}
	})

	t.Run("legacy run with no brief artifacts stays prepared", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := os.Remove(filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("remove brief directory: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StatePrepared || len(status.StaleReasons) != 0 {
			t.Fatalf("legacy run = %q with reasons %v, want prepared with none", status.State, status.StaleReasons)
		}
	})

	t.Run("legacy unregistered brief is stale reconciliation evidence", func(t *testing.T) {
		identity := briefTestRun(t)
		content := "# Legacy brief\n"
		path := briefTestWriteArtifact(t, identity.RunDir, "T01.md", content)
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StateStale {
			t.Fatalf("state = %q, want stale; reasons = %v", status.State, status.StaleReasons)
		}
		if !statusHasReason(status, ReasonUnregisteredBrief) {
			t.Fatalf("reasons = %v, want unregistered_brief", status.StaleReasons)
		}
		if status.NextPermittedInvestigation != InvestigateReconcileSource {
			t.Fatalf("next investigation = %q, want %q", status.NextPermittedInvestigation, InvestigateReconcileSource)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read legacy brief: %v", err)
		}
		if string(data) != content {
			t.Fatal("status rewrote the unregistered legacy brief")
		}
	})

	t.Run("deleted registered brief is stale", func(t *testing.T) {
		identity := briefTestRun(t)
		path := briefTestRegister(t, identity, "T01", "# Brief T01\n")
		if err := os.Remove(path); err != nil {
			t.Fatalf("delete brief: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StateStale || !statusHasReason(status, ReasonMissingArtifact) {
			t.Fatalf("state/reasons = %q/%v, want stale with missing_artifact", status.State, status.StaleReasons)
		}
		if status.NextPermittedInvestigation != InvestigateReconcileSource {
			t.Fatalf("next investigation = %q, want %q", status.NextPermittedInvestigation, InvestigateReconcileSource)
		}
	})

	t.Run("changed registered brief is stale", func(t *testing.T) {
		identity := briefTestRun(t)
		path := briefTestRegister(t, identity, "T01", "# Brief T01\n")
		if err := os.WriteFile(path, []byte("# Tampered\n"), 0o600); err != nil {
			t.Fatalf("tamper brief: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StateStale || !statusHasReason(status, ReasonBriefDigestMismatch) {
			t.Fatalf("state/reasons = %q/%v, want stale with brief_digest_mismatch", status.State, status.StaleReasons)
		}
	})

	t.Run("unsafe registered path never yields a clean status", func(t *testing.T) {
		identity := briefTestRun(t)
		checkpoint := briefTestLoadCheckpoint(t, identity.RunDir)
		checkpoint.Tasks = map[string]TaskState{
			"T01": {TaskID: "T01", State: StatePrepared, BriefPath: filepath.Join(identity.RunDir, "escaped.md"), BriefDigest: DigestBytes([]byte("x"))},
		}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatalf("persist checkpoint: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State == StatePrepared || len(status.StaleReasons) == 0 {
			t.Fatalf("unsafe layout produced a clean status: %+v", status)
		}
		if !statusHasReason(status, ReasonBriefInspectionFailed) {
			t.Fatalf("reasons = %v, want brief_inspection_failed", status.StaleReasons)
		}
		if status.NextPermittedInvestigation != InvestigateReconcileSource {
			t.Fatalf("next investigation = %q, want %q", status.NextPermittedInvestigation, InvestigateReconcileSource)
		}
	})

	t.Run("brief directory symlink is stale, not an error", func(t *testing.T) {
		identity := briefTestRun(t)
		if err := os.Remove(filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("remove brief directory: %v", err)
		}
		target := filepath.Join(identity.RunDir, "briefs-target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatalf("create brief target: %v", err)
		}
		if err := os.Symlink("briefs-target", filepath.Join(identity.RunDir, briefArtifactDirName)); err != nil {
			t.Fatalf("create brief directory symlink: %v", err)
		}
		status := statusForTest(t, identity.RunRoot, identity.RunID)
		if status.State != StateStale || !statusHasReason(status, ReasonBriefInspectionFailed) {
			t.Fatalf("state/reasons = %q/%v, want stale with brief_inspection_failed", status.State, status.StaleReasons)
		}
	})
}

func TestBriefNextInvestigationReasons(t *testing.T) {
	reasons := []string{
		ReasonMissingArtifact,
		ReasonBriefDigestMismatch,
		ReasonUnregisteredBrief,
		ReasonBriefInspectionFailed,
	}
	for _, reason := range reasons {
		if got := nextInvestigation(StateStale, []string{reason}); got != InvestigateReconcileSource {
			t.Errorf("nextInvestigation(stale, %q) = %q, want %q", reason, got, InvestigateReconcileSource)
		}
		if got := effectiveState(StatePrepared, []string{reason}, lockObservation{}); got != StateStale {
			t.Errorf("effectiveState(prepared, %q) = %q, want stale", reason, got)
		}
	}
}
