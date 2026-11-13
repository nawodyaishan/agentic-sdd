package execution

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHighestStatePrecedence(t *testing.T) {
	order := []CheckpointState{
		StateBlocked,
		StateRunning,
		StateStale,
		StateObservedFailed,
		StateObservedPartial,
		StateAwaitingVerification,
		StateVerifiedEvidence,
		StatePrepared,
	}
	for i := 0; i < len(order); i++ {
		for j := i + 1; j < len(order); j++ {
			if got := HighestState(order[i], order[j]); got != order[i] {
				t.Errorf("HighestState(%q, %q) = %q, want %q", order[i], order[j], got, order[i])
			}
			if got := HighestState(order[j], order[i]); got != order[i] {
				t.Errorf("HighestState(%q, %q) = %q, want %q", order[j], order[i], got, order[i])
			}
		}
	}
	if got := HighestState(); got != StatePrepared {
		t.Fatalf("HighestState() = %q, want prepared", got)
	}
	if got := HighestState("unknown"); got != StatePrepared {
		t.Fatalf("HighestState(unknown) = %q, want prepared", got)
	}
	if got := HighestState(StateBlocked, StateStale, StateRunning); got != StateBlocked {
		t.Fatalf("HighestState(blocked, stale, running) = %q", got)
	}
}

const digestTasks = `# Tasks

## B1 — First batch

**Batch ID:** B1
**State:** idle
**Next action:** start

### T01 — One

- [ ] **T01** Do a thing.
  - **Objective:** Keep it exact.

### T02 — Two

- [x] **T02** Already done.

## B2 — Second batch

**Batch ID:** B2

### T01 — Elsewhere

- [ ] **T01** Other work.
`

func TestNormalizedBatchScopeDigest(t *testing.T) {
	baseline, err := NormalizedBatchScopeDigest([]byte(digestTasks), "B1")
	if err != nil {
		t.Fatalf("baseline digest: %v", err)
	}
	progress := strings.Replace(digestTasks, "- [ ] **T01** Do a thing.", "- [x] **T01** Do a thing.", 1)
	progress = strings.Replace(progress, "**State:** idle", "**State:** done", 1)
	progress = strings.Replace(progress, "**Next action:** start", "**Next action:** review", 1)
	progressDigest, err := NormalizedBatchScopeDigest([]byte(progress), "B1")
	if err != nil {
		t.Fatalf("progress digest: %v", err)
	}
	if progressDigest != baseline {
		t.Fatal("checkbox and status-only progress changed the batch scope digest")
	}
	crlf, err := NormalizedBatchScopeDigest([]byte(strings.ReplaceAll(digestTasks, "\n", "\r\n")), "B1")
	if err != nil {
		t.Fatalf("CRLF digest: %v", err)
	}
	if crlf != baseline {
		t.Fatal("CRLF conversion changed the batch scope digest")
	}
	requirement := strings.Replace(digestTasks, "Do a thing.", "Do a different thing.", 1)
	requirementDigest, err := NormalizedBatchScopeDigest([]byte(requirement), "B1")
	if err != nil {
		t.Fatalf("requirement digest: %v", err)
	}
	if requirementDigest == baseline {
		t.Fatal("changing a task requirement did not change the batch scope digest")
	}
	boundary := strings.Replace(digestTasks, "  - **Objective:** Keep it exact.", "  - **Objective:** Keep it exact.\n  - **Files:** `a.go`", 1)
	boundaryDigest, err := NormalizedBatchScopeDigest([]byte(boundary), "B1")
	if err != nil {
		t.Fatalf("boundary digest: %v", err)
	}
	if boundaryDigest == baseline {
		t.Fatal("changing a task boundary did not change the batch scope digest")
	}
	otherBatch := strings.Replace(digestTasks, "Other work.", "Changed other work.", 1)
	otherDigest, err := NormalizedBatchScopeDigest([]byte(otherBatch), "B1")
	if err != nil {
		t.Fatalf("other batch digest: %v", err)
	}
	if otherDigest != baseline {
		t.Fatal("another batch's change altered the selected batch scope digest")
	}
	alternativeHeading := strings.Replace(digestTasks, "## B1 — First batch", "## Batch B1 — First batch", 1)
	if _, err := NormalizedBatchScopeDigest([]byte(alternativeHeading), "B1"); err != nil {
		t.Fatalf("alternative batch heading: %v", err)
	}
	fenced := "```md\n## B1 — Fenced duplicate\n```\n\n" + digestTasks
	if _, err := NormalizedBatchScopeDigest([]byte(fenced), "B1"); err != nil {
		t.Fatalf("fenced heading must not count as a batch: %v", err)
	}
	if _, err := NormalizedBatchScopeDigest([]byte(digestTasks), "B9"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing batch error = %v, want ErrNotFound", err)
	}
	duplicate := digestTasks + "\n## B1 — Duplicate batch\n"
	if _, err := NormalizedBatchScopeDigest([]byte(duplicate), "B1"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate batch error = %v, want ErrConflict", err)
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	checkpoint.Tasks = map[string]TaskState{
		"T01": {TaskID: "T01", State: StateAwaitingVerification, LastRecordID: "rec-x", UpdatedAt: time.Now().UTC()},
	}
	checkpoint.State = StateAwaitingVerification
	if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	reloaded, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("reload checkpoint: %v", err)
	}
	if reloaded.State != StateAwaitingVerification || reloaded.Tasks["T01"].State != StateAwaitingVerification {
		t.Fatalf("reloaded checkpoint = %+v", reloaded)
	}
	info, err := os.Stat(filepath.Join(identity.RunDir, checkpointName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("checkpoint mode = %o, want 600", info.Mode().Perm())
	}
}

func TestStatusPrepared(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	status, err := StatusRun(runRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.SchemaVersion != SchemaVersion || status.Kind != "execution.run.status" {
		t.Fatalf("status identity = %d/%q", status.SchemaVersion, status.Kind)
	}
	if status.State != StatePrepared {
		t.Fatalf("state = %q, want prepared", status.State)
	}
	if len(status.StaleReasons) != 0 {
		t.Fatalf("stale reasons = %v, want none", status.StaleReasons)
	}
	if status.ProgressUpdated {
		t.Fatal("fresh run reports progress_updated")
	}
	if status.NextPermittedInvestigation != InvestigateNone {
		t.Fatalf("next investigation = %q, want none", status.NextPermittedInvestigation)
	}
	if status.CurrentObservation == nil || status.LastObservation == nil {
		t.Fatal("status is missing observations")
	}
	if status.CurrentObservation.Head != status.LastObservation.Head {
		t.Fatal("fresh run observations disagree")
	}
	if status.RunDir != identity.RunDir {
		t.Fatalf("status run dir = %q", status.RunDir)
	}
}

func TestStatusSourceAndWorktreeFreshness(t *testing.T) {
	t.Run("spec changed", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		specPath := filepath.Join(root, "specs", "001-test", "spec.md")
		repoTestWrite(t, specPath, repoTestRead(t, specPath)+"\nAdditional requirement.\n")
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonSpecChanged) {
			t.Fatalf("reasons = %v, want spec_changed", status.StaleReasons)
		}
		if status.State != StateStale {
			t.Fatalf("state = %q, want stale", status.State)
		}
		if status.NextPermittedInvestigation != InvestigateReconcileSource {
			t.Fatalf("next investigation = %q", status.NextPermittedInvestigation)
		}
	})
	t.Run("checkbox progress is informational", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		tasksPath := filepath.Join(root, "specs", "001-test", "tasks.md")
		updated := strings.Replace(repoTestRead(t, tasksPath), "- [ ] **T01** Do a thing.", "- [x] **T01** Do a thing.", 1)
		updated = strings.Replace(updated, "**State:** `in progress`", "**State:** `done working`", 1)
		repoTestWrite(t, tasksPath, updated)
		status := statusForTest(t, runRoot, identity.RunID)
		if !status.ProgressUpdated || !statusHasReason(status, ReasonProgressUpdated) {
			t.Fatalf("reasons = %v, progress = %v, want progress_updated", status.StaleReasons, status.ProgressUpdated)
		}
		if status.State != StatePrepared {
			t.Fatalf("state = %q, want prepared", status.State)
		}
		if statusHasReason(status, ReasonTasksChanged) || statusHasReason(status, ReasonBatchChanged) {
			t.Fatalf("progress-only change reported batch drift: %v", status.StaleReasons)
		}
	})
	t.Run("task requirement changed", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		tasksPath := filepath.Join(root, "specs", "001-test", "tasks.md")
		updated := strings.Replace(repoTestRead(t, tasksPath), "Do a thing.", "Do a different thing.", 1)
		repoTestWrite(t, tasksPath, updated)
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonBatchChanged) || !statusHasReason(status, ReasonTasksChanged) {
			t.Fatalf("reasons = %v, want batch_changed and tasks_changed", status.StaleReasons)
		}
		if status.State != StateStale {
			t.Fatalf("state = %q, want stale", status.State)
		}
		if status.NextPermittedInvestigation != InvestigateReconcileSource {
			t.Fatalf("next investigation = %q", status.NextPermittedInvestigation)
		}
	})
	t.Run("another batch changed", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		tasksPath := filepath.Join(root, "specs", "001-test", "tasks.md")
		updated := strings.Replace(repoTestRead(t, tasksPath), "Appears only in another batch.", "Changed elsewhere.", 1)
		repoTestWrite(t, tasksPath, updated)
		status := statusForTest(t, runRoot, identity.RunID)
		if status.State != StatePrepared {
			t.Fatalf("state = %q, want prepared for an unrelated batch edit", status.State)
		}
	})
	t.Run("head changed", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		repoTestWrite(t, filepath.Join(root, "later.txt"), "later\n")
		repoTestCommit(t, root, "later")
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonHeadChanged) {
			t.Fatalf("reasons = %v, want head_changed", status.StaleReasons)
		}
		if statusHasReason(status, ReasonWorktreeChanged) {
			t.Fatalf("clean commit must not report worktree_changed: %v", status.StaleReasons)
		}
		if status.State != StateStale || status.NextPermittedInvestigation != InvestigateChangedPaths {
			t.Fatalf("state/investigation = %q/%q", status.State, status.NextPermittedInvestigation)
		}
	})
	t.Run("worktree changed", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		repoTestWrite(t, filepath.Join(root, "README.md"), "baseline\nedited\n")
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonWorktreeChanged) {
			t.Fatalf("reasons = %v, want worktree_changed", status.StaleReasons)
		}
		if status.State != StateStale || status.NextPermittedInvestigation != InvestigateChangedPaths {
			t.Fatalf("state/investigation = %q/%q", status.State, status.NextPermittedInvestigation)
		}
	})
	t.Run("untracked file added", func(t *testing.T) {
		root, runRoot, identity := workspaceTestRun(t)
		repoTestWrite(t, filepath.Join(root, "untracked.txt"), "new\n")
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonWorktreeChanged) {
			t.Fatalf("reasons = %v, want worktree_changed", status.StaleReasons)
		}
	})
}

func TestStatusMissingArtifact(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint.Tasks = map[string]TaskState{
		"T01": {TaskID: "T01", State: StateAwaitingVerification, BriefPath: filepath.Join(identity.RunDir, "briefs", "T01.md")},
	}
	if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
		t.Fatal(err)
	}
	status := statusForTest(t, runRoot, identity.RunID)
	if !statusHasReason(status, ReasonMissingArtifact) {
		t.Fatalf("reasons = %v, want missing_artifact", status.StaleReasons)
	}
	if status.State != StateStale || status.NextPermittedInvestigation != InvestigateReconcileSource {
		t.Fatalf("state/investigation = %q/%q", status.State, status.NextPermittedInvestigation)
	}
}

func TestStatusInterruptedCheck(t *testing.T) {
	t.Run("missing lock", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatal(err)
		}
		checkpoint.State = StateRunning
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatal(err)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonInterruptedCheck) {
			t.Fatalf("reasons = %v, want interrupted_check", status.StaleReasons)
		}
		if status.State != StateObservedPartial {
			t.Fatalf("state = %q, want observed-partial", status.State)
		}
		if status.NextPermittedInvestigation != InvestigateRerunCheck {
			t.Fatalf("next investigation = %q", status.NextPermittedInvestigation)
		}
	})
	t.Run("stale lock", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatal(err)
		}
		checkpoint.State = StateRunning
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatal(err)
		}
		payload := fmt.Sprintf(`{"schemaVersion":1,"token":"dead","pid":2147483647,"hostname":%q,"started_at":%q,"operation":"check"}`,
			hostName(), time.Now().UTC().Format(time.RFC3339Nano))
		if err := writeFileAtomic(filepath.Join(identity.RunDir, lockFileName), []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonInterruptedCheck) || !statusHasReason(status, ReasonStaleLock) {
			t.Fatalf("reasons = %v, want interrupted_check and stale_lock", status.StaleReasons)
		}
		if status.State != StateObservedPartial {
			t.Fatalf("state = %q, want observed-partial", status.State)
		}
	})
}

func TestStatusBlocked(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint.State = StateBlocked
	checkpoint.Blockers = []Blocker{{Kind: "permission", Detail: "write denied"}}
	if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
		t.Fatal(err)
	}
	status := statusForTest(t, runRoot, identity.RunID)
	if status.State != StateBlocked {
		t.Fatalf("state = %q, want blocked", status.State)
	}
	if status.NextPermittedInvestigation != InvestigateResolveBlocker {
		t.Fatalf("next investigation = %q", status.NextPermittedInvestigation)
	}
}

func TestStatusContradictoryEvidence(t *testing.T) {
	t.Run("verified without a passing check", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatal(err)
		}
		checkpoint.State = StateVerifiedEvidence
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatal(err)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if !statusHasReason(status, ReasonContradictoryEvidence) {
			t.Fatalf("reasons = %v, want contradictory_evidence", status.StaleReasons)
		}
		if status.State != StateStale {
			t.Fatalf("state = %q, want stale", status.State)
		}
	})
	t.Run("verified with a passing check", func(t *testing.T) {
		_, runRoot, identity := workspaceTestRun(t)
		checkpoint, err := LoadCheckpoint(identity.RunDir)
		if err != nil {
			t.Fatal(err)
		}
		checkpoint.State = StateVerifiedEvidence
		checkpoint.Checks = []CheckRecord{{CheckID: "chk-test", Result: CheckPassed}}
		if err := WriteCheckpoint(identity.RunDir, checkpoint); err != nil {
			t.Fatal(err)
		}
		status := statusForTest(t, runRoot, identity.RunID)
		if statusHasReason(status, ReasonContradictoryEvidence) {
			t.Fatalf("passing evidence was called contradictory: %v", status.StaleReasons)
		}
		if status.State != StateVerifiedEvidence {
			t.Fatalf("state = %q, want verified-evidence", status.State)
		}
		if status.NextPermittedInvestigation != InvestigateNone {
			t.Fatalf("next investigation = %q, want none", status.NextPermittedInvestigation)
		}
	})
}

func TestStatusScopeMismatch(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	identity.Scope.RepoRoot = t.TempDir()
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(filepath.Join(identity.RunDir, runIdentityName), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	status := statusForTest(t, runRoot, identity.RunID)
	if !statusHasReason(status, ReasonScopeMismatch) {
		t.Fatalf("reasons = %v, want scope_mismatch", status.StaleReasons)
	}
	if status.State != StateStale || status.NextPermittedInvestigation != InvestigateReconcileSource {
		t.Fatalf("state/investigation = %q/%q", status.State, status.NextPermittedInvestigation)
	}
}

func TestStatusNeverUsesApprovalLanguage(t *testing.T) {
	_, runRoot, identity := workspaceTestRun(t)
	status := statusForTest(t, runRoot, identity.RunID)
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(data))
	for _, forbidden := range []string{"approved", "approval granted", "resume", "next_batch", "start_next", "completed"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("status JSON contains forbidden claim %q: %s", forbidden, text)
		}
	}
	allowed := map[string]bool{
		InvestigateNone:            true,
		InvestigateChangedPaths:    true,
		InvestigateRerunCheck:      true,
		InvestigateReconcileSource: true,
		InvestigateResolveBlocker:  true,
	}
	if !allowed[status.NextPermittedInvestigation] {
		t.Fatalf("next investigation = %q is outside the permitted values", status.NextPermittedInvestigation)
	}
}

func TestStatusStateMachineTable(t *testing.T) {
	live := lockObservation{exists: true, live: true}
	stale := lockObservation{exists: true, stale: true}
	none := lockObservation{}
	cases := []struct {
		name      string
		base      CheckpointState
		reasons   []string
		lock      lockObservation
		wantState CheckpointState
		wantNext  string
	}{
		{"blocked wins over drift", StateBlocked, []string{ReasonWorktreeChanged}, live, StateBlocked, InvestigateResolveBlocker},
		{"live running check", StateRunning, nil, live, StateRunning, InvestigateNone},
		{"interrupted running check", StateRunning, []string{ReasonInterruptedCheck}, none, StateObservedPartial, InvestigateRerunCheck},
		{"failed check", StateObservedFailed, nil, none, StateObservedFailed, InvestigateRerunCheck},
		{"partial check", StateObservedPartial, nil, none, StateObservedPartial, InvestigateRerunCheck},
		{"awaiting verification", StateAwaitingVerification, nil, none, StateAwaitingVerification, InvestigateChangedPaths},
		{"verified evidence", StateVerifiedEvidence, nil, none, StateVerifiedEvidence, InvestigateNone},
		{"prepared", StatePrepared, nil, none, StatePrepared, InvestigateNone},
		{"source drift", StateAwaitingVerification, []string{ReasonBatchChanged}, none, StateStale, InvestigateReconcileSource},
		{"head drift", StateAwaitingVerification, []string{ReasonHeadChanged}, none, StateStale, InvestigateChangedPaths},
		{"worktree drift", StateAwaitingVerification, []string{ReasonWorktreeChanged}, none, StateStale, InvestigateChangedPaths},
		{"missing artifact", StateVerifiedEvidence, []string{ReasonMissingArtifact}, none, StateStale, InvestigateReconcileSource},
		{"contradictory evidence", StateVerifiedEvidence, []string{ReasonContradictoryEvidence}, none, StateStale, InvestigateReconcileSource},
		{"progress only", StateAwaitingVerification, []string{ReasonProgressUpdated}, none, StateAwaitingVerification, InvestigateChangedPaths},
		{"stale lock only", StateObservedPartial, []string{ReasonInterruptedCheck, ReasonStaleLock}, stale, StateObservedPartial, InvestigateRerunCheck},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			state := effectiveState(testCase.base, testCase.reasons, testCase.lock)
			if state != testCase.wantState {
				t.Fatalf("state = %q, want %q", state, testCase.wantState)
			}
			if next := nextInvestigation(state, testCase.reasons); next != testCase.wantNext {
				t.Fatalf("next investigation = %q, want %q", next, testCase.wantNext)
			}
		})
	}
}

func TestLoadCheckpointRejectsUnknownState(t *testing.T) {
	_, _, identity := workspaceTestRun(t)
	for _, invalid := range []string{"completed", "approved", ""} {
		data := []byte(`{"schemaVersion":1,"run_id":"` + identity.RunID + `","state":"` + invalid + `"}`)
		if err := writeFileAtomic(filepath.Join(identity.RunDir, checkpointName), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadCheckpoint(identity.RunDir); !errors.Is(err, ErrUsage) {
			t.Fatalf("LoadCheckpoint(state=%q) error = %v, want ErrUsage", invalid, err)
		}
	}
}

func TestStatusUnknownRun(t *testing.T) {
	_, runRoot, _ := workspaceTestRun(t)
	if _, err := StatusRun(runRoot, "run-20261113T114500Z-abcdef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("StatusRun(unknown) error = %v, want ErrNotFound", err)
	}
}

func statusForTest(t *testing.T, runRoot, runID string) Status {
	t.Helper()
	status, err := StatusRun(runRoot, runID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return status
}

func statusHasReason(status Status, reason string) bool {
	for _, existing := range status.StaleReasons {
		if existing == reason {
			return true
		}
	}
	return false
}
