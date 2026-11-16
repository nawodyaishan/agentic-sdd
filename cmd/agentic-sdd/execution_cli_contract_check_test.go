package main

// Feature 021 B1 contract tests for the public `asdx-core run check` boundary
// and the CLI service edges that were previously reachable only through the
// typed service: timeout (G1), failure (G2), output cap (G3), interruption (G4)
// and the not_started / worktree-drift / freshness / blocker outcomes (G12).
//
// Everything is asserted structurally at the CLI boundary: exit code, the
// versioned JSON envelope, the observed record and the reported run state. No
// golden JSON, no built binary and no real os.Exit. Every fixture is a
// disposable temporary repository; no test reads or writes a real home.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"agentic-sdd/internal/execution"
)

// TestExecutionContractCheckTimeout covers G1 (spec FR-02/FR-05, AC-02/AC-05):
// a check that exceeds its wall-clock limit exits 124 with a timed_out
// observation, partial run state and the private log paths, and never looks
// like a pass.
func TestExecutionContractCheckTimeout(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable on this host: check timeout case is not_run")
	}
	if runtime.GOOS == "windows" {
		t.Skip("not_run: Windows sh exit-code semantics are outside the supported test matrix")
	}
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	// 300ms against a 5s sleep keeps a deliberate margin: the check timer starts
	// before cmd.Start, so a tighter deadline can yield not_started under load.
	code, out, errOut := runExecutionCLI(t, dir, "run", "check",
		"--run", identity.RunID, "--task", "T01",
		"--timeout", "300ms", "--format", "json",
		"--", shPath, "-c", "sleep 5")
	if code != 124 {
		t.Fatalf("timed-out check exit %d, want 124: stdout %q stderr %q", code, out, errOut)
	}
	var envelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode timed-out check: %v\n%s", err, out)
	}
	if envelope.SchemaVersion != execution.SchemaVersion || envelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", envelope)
	}
	record := envelope.Check
	if record.Result != execution.CheckTimedOut || !record.TimedOut {
		t.Fatalf("record = %+v, want a timed_out observation", record)
	}
	if record.Result == execution.CheckPassed {
		t.Fatal("a timed-out check reported a pass")
	}
	if record.ExitCode != nil {
		t.Fatalf("timed-out check reported exit code %d, but a killed process has none", *record.ExitCode)
	}
	if record.StdoutPath == "" || record.StderrPath == "" {
		t.Fatalf("timed-out check lost its log paths: %+v", record)
	}
	for _, logPath := range []string{record.StdoutPath, record.StderrPath} {
		info, statErr := os.Stat(logPath)
		if statErr != nil || !info.Mode().IsRegular() {
			t.Fatalf("check log %q is not a regular file: %v", logPath, statErr)
		}
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedPartial {
		t.Fatalf("run state = %q, want observed-partial (reasons %v)", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a timed-out check produced verified evidence")
	}
	if len(status.Checks) != 1 || status.Checks[0].Result != execution.CheckTimedOut {
		t.Fatalf("status checks = %+v", status.Checks)
	}
}

// TestExecutionContractCheckFailure covers G2: a non-zero command records the
// command's own exit code (not the wrapper's) as a failed observation.
func TestExecutionContractCheckFailure(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable on this host: check failure case is not_run")
	}
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	code, out, errOut := runExecutionCLI(t, dir, "run", "check",
		"--run", identity.RunID, "--task", "T01", "--format", "json",
		"--", shPath, "-c", "exit 7")
	if code != 1 {
		t.Fatalf("failed check exit %d, want 1: stdout %q stderr %q", code, out, errOut)
	}
	var envelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode failed check: %v\n%s", err, out)
	}
	if envelope.SchemaVersion != execution.SchemaVersion || envelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", envelope)
	}
	record := envelope.Check
	if record.Result != execution.CheckFailed {
		t.Fatalf("record = %+v, want a failed observation", record)
	}
	if record.ExitCode == nil || *record.ExitCode != 7 {
		t.Fatalf("record exit code = %v, want the command's own exit 7", record.ExitCode)
	}
	if record.TimedOut || record.OutputLimit {
		t.Fatalf("failed check reported a timeout or output limit: %+v", record)
	}
	if record.StdoutPath == "" || record.StderrPath == "" {
		t.Fatalf("failed check lost its log paths: %+v", record)
	}
	for _, logPath := range []string{record.StdoutPath, record.StderrPath} {
		if _, statErr := os.Stat(logPath); statErr != nil {
			t.Fatalf("check log %q is missing: %v", logPath, statErr)
		}
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedFailed {
		t.Fatalf("run state = %q, want observed-failed (reasons %v)", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a failed check produced verified evidence")
	}
	if status.NextPermittedInvestigation != execution.InvestigateRerunCheck {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateRerunCheck)
	}
}

// TestExecutionContractCheckOutputLimit covers G3: a stream over the configured
// cap is truncated, terminated and reported as an output-limit observation in a
// partial run state.
func TestExecutionContractCheckOutputLimit(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable on this host: output-limit case is not_run")
	}
	if runtime.GOOS == "windows" {
		t.Skip("not_run: Windows sh exit-code semantics are outside the supported test matrix")
	}
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	// POSIX sh builtins only: 400 lines of 40 bytes overflow the 1 KiB cap
	// without depending on seq, head or any other external program.
	const loop = `i=0; while [ $i -lt 400 ]; do echo 0123456789012345678901234567890123456789; i=$((i+1)); done`
	code, out, errOut := runExecutionCLI(t, dir, "run", "check",
		"--run", identity.RunID, "--task", "T01",
		"--max-output-bytes", "1024", "--format", "json",
		"--", shPath, "-c", loop)
	if code != 1 {
		t.Fatalf("output-limit check exit %d, want 1: stdout %q stderr %q", code, out, errOut)
	}
	var envelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode output-limit check: %v\n%s", err, out)
	}
	if envelope.SchemaVersion != execution.SchemaVersion || envelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", envelope)
	}
	record := envelope.Check
	if record.Result != execution.CheckOutputLimit {
		t.Fatalf("record = %+v, want an output_limit observation", record)
	}
	if !record.OutputLimit {
		t.Fatalf("record did not set output_limit: %+v", record)
	}
	if !record.StdoutTruncated {
		t.Fatalf("record did not set stdout_truncated: %+v", record)
	}
	if record.StdoutBytes <= 0 || record.StdoutBytes > 1024 {
		t.Fatalf("stdout bytes = %d, want 1..1024", record.StdoutBytes)
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedPartial {
		t.Fatalf("run state = %q, want observed-partial (reasons %v)", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("an over-limit check produced verified evidence")
	}
}

// TestExecutionContractCheckInterrupted covers G4: a command killed by a signal
// is an interrupted observation, stays partial and releases the run lock.
func TestExecutionContractCheckInterrupted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal semantics are not portable to windows: interrupted case is not_run")
	}
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable on this host: interrupted case is not_run")
	}
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	code, out, errOut := runExecutionCLI(t, dir, "run", "check",
		"--run", identity.RunID, "--task", "T01", "--format", "json",
		"--", shPath, "-c", "kill -TERM $$")
	if code != 1 {
		t.Fatalf("interrupted check exit %d, want 1: stdout %q stderr %q", code, out, errOut)
	}
	var envelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode interrupted check: %v\n%s", err, out)
	}
	if envelope.SchemaVersion != execution.SchemaVersion || envelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", envelope)
	}
	record := envelope.Check
	if record.Result != execution.CheckInterrupted {
		t.Fatalf("record = %+v, want an interrupted observation", record)
	}
	if record.Signal == "" {
		t.Fatalf("interrupted check did not report the terminating signal: %+v", record)
	}
	if record.ExitCode != nil {
		t.Fatalf("interrupted check reported exit code %d, but a signalled process has none", *record.ExitCode)
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedPartial {
		t.Fatalf("run state = %q, want observed-partial (reasons %v)", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("an interrupted check produced verified evidence")
	}
	if _, statErr := os.Stat(filepath.Join(identity.RunDir, ".lock")); !os.IsNotExist(statErr) {
		t.Fatalf("run lock survived the interrupted check: %v", statErr)
	}
}

// TestExecutionContractServiceEdges covers G12: the CLI service edges that the
// typed service already proves — not_started, a worktree change during a check,
// spec drift versus progress-only drift, and a recorded blocker.
func TestExecutionContractServiceEdges(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh unavailable on this host: worktree-mutation edge is not_run")
	}
	dir := executionTestRepo(t)

	// A missing program is an observed not_started failure, not a usage error:
	// the attempt is recorded and the wrapper exits 1.
	missingRun := initExecutionIntegrityRun(t, dir)
	code, out, errOut := runExecutionCLI(t, dir, "run", "check",
		"--run", missingRun.RunID, "--task", "T01", "--format", "json",
		"--", "asdx-contract-missing-program")
	if code != 1 {
		t.Fatalf("missing-program check exit %d, want 1: stdout %q stderr %q", code, out, errOut)
	}
	var checkEnvelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &checkEnvelope); err != nil {
		t.Fatalf("decode not_started check: %v\n%s", err, out)
	}
	if checkEnvelope.SchemaVersion != execution.SchemaVersion || checkEnvelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", checkEnvelope)
	}
	record := checkEnvelope.Check
	if record.Result != execution.CheckNotStarted {
		t.Fatalf("record = %+v, want not_started", record)
	}
	if record.ExitCode != nil {
		t.Fatalf("not_started check reported exit code %d", *record.ExitCode)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", missingRun.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedFailed {
		t.Fatalf("run state = %q, want observed-failed (reasons %v)", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a missing program produced verified evidence")
	}

	// A passing command that mutates the worktree while it runs stays observable
	// but can never support verified evidence.
	mutatingRun := initExecutionIntegrityRun(t, dir)
	code, out, errOut = runExecutionCLI(t, dir, "run", "check",
		"--run", mutatingRun.RunID, "--task", "T01", "--format", "json",
		"--", shPath, "-c", "echo mutation > mutation.txt")
	if code != 0 {
		t.Fatalf("mutating check exit %d, want 0 for a passing command: %s", code, errOut)
	}
	checkEnvelope.Check = execution.CheckRecord{}
	if err := json.Unmarshal([]byte(out), &checkEnvelope); err != nil {
		t.Fatalf("decode mutating check: %v\n%s", err, out)
	}
	record = checkEnvelope.Check
	if record.Result != execution.CheckPassed {
		t.Fatalf("mutating check result = %q, want passed", record.Result)
	}
	if !record.WorktreeChangedDuringCheck {
		t.Fatalf("mutating check did not report the worktree change: %+v", record)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "mutation.txt")); statErr != nil {
		t.Fatalf("the fixture mutation did not reach the worktree: %v", statErr)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", mutatingRun.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status = decodeExecutionStatus(t, out)
	if status.State != execution.StateObservedPartial {
		t.Fatalf("run state = %q, want observed-partial after a check-time worktree mutation", status.State)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a worktree mutation during the check produced verified evidence")
	}

	// A spec edit after run init is blocking drift that requires reconciliation.
	specRun := initExecutionIntegrityRun(t, dir)
	specPath := filepath.Join(dir, "feature", "spec.md")
	specData, readErr := os.ReadFile(specPath)
	if readErr != nil {
		t.Fatalf("read fixture spec: %v", readErr)
	}
	appended := append(append([]byte(nil), specData...), []byte("\n### FR-02 — Appended requirement\n\nAdded after run init.\n")...)
	if writeErr := os.WriteFile(specPath, appended, 0o600); writeErr != nil {
		t.Fatalf("edit fixture spec: %v", writeErr)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", specRun.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("stale run status exit %d: %s", code, errOut)
	}
	status = decodeExecutionStatus(t, out)
	if status.State != execution.StateStale {
		t.Fatalf("run state = %q, want stale after a spec edit (reasons %v)", status.State, status.StaleReasons)
	}
	if !slices.Contains(status.StaleReasons, execution.ReasonSpecChanged) {
		t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonSpecChanged)
	}
	if status.ProgressUpdated {
		t.Fatal("a spec change was reported as progress-only")
	}
	if status.NextPermittedInvestigation != execution.InvestigateReconcileSource {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateReconcileSource)
	}

	// A tasks status-line edit is progress-only: it is reported, it does not
	// stale the run and it never counts as a changed batch scope.
	progressRun := initExecutionIntegrityRun(t, dir)
	tasksPath := filepath.Join(dir, "feature", "tasks.md")
	tasksData, readErr := os.ReadFile(tasksPath)
	if readErr != nil {
		t.Fatalf("read fixture tasks: %v", readErr)
	}
	// Progress-only lines are dropped from the normalized section, but blank
	// lines are kept: insert the status line without adding a second blank line
	// so only the ignored line changes.
	const outcomeLine = "**Outcome:** One fixture task."
	progressTasks := strings.Replace(string(tasksData), outcomeLine, outcomeLine+"\n**Status:** in progress", 1)
	if progressTasks == string(tasksData) {
		t.Fatalf("fixture tasks.md no longer carries %q; the progress edge cannot be constructed", outcomeLine)
	}
	if writeErr := os.WriteFile(tasksPath, []byte(progressTasks), 0o600); writeErr != nil {
		t.Fatalf("edit fixture tasks: %v", writeErr)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", progressRun.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("progress run status exit %d: %s", code, errOut)
	}
	status = decodeExecutionStatus(t, out)
	if !status.ProgressUpdated {
		t.Fatalf("status did not report progress_updated: state %q reasons %v", status.State, status.StaleReasons)
	}
	if !slices.Contains(status.StaleReasons, execution.ReasonProgressUpdated) {
		t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonProgressUpdated)
	}
	for _, blocking := range []string{execution.ReasonSpecChanged, execution.ReasonTasksChanged, execution.ReasonBatchChanged} {
		if slices.Contains(status.StaleReasons, blocking) {
			t.Fatalf("progress-only edit reported blocking reason %q: %v", blocking, status.StaleReasons)
		}
	}
	if status.State != execution.StatePrepared {
		t.Fatalf("run state = %q, want prepared for progress-only drift", status.State)
	}
	if status.NextPermittedInvestigation != execution.InvestigateNone {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateNone)
	}

	// A recorded blocker is the strongest observed state and never a pass.
	blockerRun := initExecutionIntegrityRun(t, dir)
	recordPath := filepath.Join(t.TempDir(), "blocked.json")
	const claimDocument = `{"schemaVersion":1,"taskId":"T01","result":"blocked","summary":"dependency unavailable","changedPaths":[],"checks":[],"blockers":[{"kind":"test","detail":"fixture dependency unavailable"}],"unknowns":[]}`
	if writeErr := os.WriteFile(recordPath, []byte(claimDocument), 0o600); writeErr != nil {
		t.Fatalf("write blocker claim: %v", writeErr)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "record",
		"--run", blockerRun.RunID, "--task", "T01", "--result", recordPath, "--format", "json")
	if code != 0 {
		t.Fatalf("blocker record exit %d: %s", code, errOut)
	}
	var claimEnvelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Record        execution.ClaimRecord `json:"record"`
	}
	if err := json.Unmarshal([]byte(out), &claimEnvelope); err != nil {
		t.Fatalf("decode blocker claim: %v\n%s", err, out)
	}
	if claimEnvelope.SchemaVersion != execution.SchemaVersion || claimEnvelope.Kind != "execution.run.record" {
		t.Fatalf("record envelope = %+v", claimEnvelope)
	}
	if claimEnvelope.Record.Result != execution.ClaimBlocked || len(claimEnvelope.Record.Blockers) != 1 {
		t.Fatalf("claim = %+v, want one recorded blocker", claimEnvelope.Record)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", blockerRun.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("blocked run status exit %d: %s", code, errOut)
	}
	status = decodeExecutionStatus(t, out)
	if status.State != execution.StateBlocked {
		t.Fatalf("run state = %q, want blocked (reasons %v)", status.State, status.StaleReasons)
	}
	if len(status.Blockers) != 1 || status.Blockers[0].Kind != "test" || status.Blockers[0].Detail == "" {
		t.Fatalf("status blockers = %+v, want the recorded test blocker", status.Blockers)
	}
	if status.NextPermittedInvestigation != execution.InvestigateResolveBlocker {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateResolveBlocker)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a blocker claim produced verified evidence")
	}
}
