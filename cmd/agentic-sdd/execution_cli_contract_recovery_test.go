package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"agentic-sdd/internal/execution"
)

var executionCheckIDPattern = regexp.MustCompile(`^chk-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$`)

type executionCLICallResult struct {
	code   int
	out    string
	errOut string
}

func setExecutionGitIsolationEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_OPTIONAL_LOCKS", "0")
}

// runExecutionCLIDirect invokes runNonInteractive without calling t.Setenv so
// worker goroutines and concurrent callers never mutate process environment.
func runExecutionCLIDirect(dir string, args ...string) executionCLICallResult {
	var stdout, stderr bytes.Buffer
	fullArgs := append([]string(nil), args...)
	insertAt := len(fullArgs)
	for index, arg := range fullArgs {
		if arg == "--" {
			insertAt = index
			break
		}
	}
	fullArgs = append(fullArgs[:insertAt], append([]string{"--repo", dir}, fullArgs[insertAt:]...)...)
	code := runNonInteractive(fullArgs, &stdout, &stderr)
	return executionCLICallResult{
		code:   code,
		out:    stdout.String(),
		errOut: stderr.String(),
	}
}

// startSleepingCheckHolder starts a deterministic sleeping check in a worker
// goroutine, polls until .lock exists and checkpoint.json reports state
// "running", and returns the observed checkpoint bytes along with a join func.
// Callers should defer join() immediately so the worker is always joined.
func startSleepingCheckHolder(t *testing.T, dir, runID, runDir, shPath string) ([]byte, func() executionCLICallResult) {
	t.Helper()
	done := make(chan executionCLICallResult, 1)
	joined := false
	var joinedRes executionCLICallResult
	join := func() executionCLICallResult {
		if !joined {
			joinedRes = <-done
			joined = true
		}
		return joinedRes
	}

	go func() {
		done <- runExecutionCLIDirect(
			dir,
			"run", "check",
			"--run", runID,
			"--task", "T01",
			"--format", "json",
			"--", shPath, "-c", "sleep 3",
		)
	}()

	lockPath := filepath.Join(runDir, ".lock")
	checkpointPath := filepath.Join(runDir, "checkpoint.json")
	deadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case res := <-done:
			joined = true
			joinedRes = res
			t.Fatalf("holder check exited before lock/running observation (exit %d): stdout=%q stderr=%q", res.code, res.out, res.errOut)
		default:
		}
		if _, err := os.Lstat(lockPath); err == nil {
			if data, readErr := os.ReadFile(checkpointPath); readErr == nil {
				var cp execution.Checkpoint
				if json.Unmarshal(data, &cp) == nil && cp.State == execution.StateRunning {
					return data, join
				}
			}
		}
		if time.Now().After(deadline) {
			_ = join()
			t.Fatalf("timed out waiting for .lock and running checkpoint in %s", runDir)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestExecutionContractWriterLock(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("not_run: sh unavailable: %v", err)
	}
	setExecutionGitIsolationEnv(t)
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	runningCheckpoint, join := startSleepingCheckHolder(t, dir, identity.RunID, identity.RunDir, shPath)
	defer func() { _ = join() }()

	checksDir := filepath.Join(identity.RunDir, "checks")
	briefsDir := filepath.Join(identity.RunDir, "briefs")
	checksBefore, err := os.ReadDir(checksDir)
	if err != nil {
		t.Fatalf("read checks dir: %v", err)
	}
	if len(checksBefore) != 1 {
		t.Fatalf("checks count while holder running = %d, want 1", len(checksBefore))
	}
	briefsBefore, err := os.ReadDir(briefsDir)
	if err != nil {
		t.Fatalf("read briefs dir: %v", err)
	}
	if len(briefsBefore) != 0 {
		t.Fatalf("briefs count before competing call = %d, want 0", len(briefsBefore))
	}

	competingCheck := runExecutionCLIDirect(
		dir,
		"run", "check",
		"--run", identity.RunID,
		"--task", "T01",
		"--format", "json",
		"--", shPath, "-c", "exit 0",
	)
	if competingCheck.code != 2 {
		t.Fatalf("competing run check exit = %d, want 2: stdout=%q stderr=%q", competingCheck.code, competingCheck.out, competingCheck.errOut)
	}
	if strings.TrimSpace(competingCheck.out) != "" {
		t.Fatalf("competing run check wrote unexpected stdout: %q", competingCheck.out)
	}
	if !strings.Contains(strings.ToLower(competingCheck.errOut), "locked") {
		t.Fatalf("competing run check stderr missing lock diagnostic: %q", competingCheck.errOut)
	}

	competingBrief := runExecutionCLIDirect(
		dir,
		"task", "brief",
		"--run", identity.RunID,
		"--task", "T01",
		"--format", "json",
	)
	if competingBrief.code != 2 {
		t.Fatalf("competing task brief exit = %d, want 2: stdout=%q stderr=%q", competingBrief.code, competingBrief.out, competingBrief.errOut)
	}
	if strings.TrimSpace(competingBrief.out) != "" {
		t.Fatalf("competing task brief wrote unexpected stdout: %q", competingBrief.out)
	}
	if !strings.Contains(strings.ToLower(competingBrief.errOut), "locked") {
		t.Fatalf("competing task brief stderr missing lock diagnostic: %q", competingBrief.errOut)
	}

	checksAfterCompeting, err := os.ReadDir(checksDir)
	if err != nil {
		t.Fatalf("read checks dir after competing calls: %v", err)
	}
	if len(checksAfterCompeting) != 1 || checksAfterCompeting[0].Name() != checksBefore[0].Name() {
		t.Fatalf("competing writer altered checks dir: got %v, want %v", checksAfterCompeting, checksBefore)
	}
	briefsAfterCompeting, err := os.ReadDir(briefsDir)
	if err != nil {
		t.Fatalf("read briefs dir after competing calls: %v", err)
	}
	if len(briefsAfterCompeting) != 0 {
		t.Fatalf("competing task brief created artifact: %v", briefsAfterCompeting)
	}
	checkpointDuring, err := os.ReadFile(filepath.Join(identity.RunDir, "checkpoint.json"))
	if err != nil {
		t.Fatalf("read checkpoint during holder sleep: %v", err)
	}
	if !bytes.Equal(checkpointDuring, runningCheckpoint) {
		t.Fatal("competing writer changed checkpoint.json while lock was held")
	}

	holder := join()
	if holder.code != 0 {
		t.Fatalf("holder check exit = %d, want 0: stdout=%q stderr=%q", holder.code, holder.out, holder.errOut)
	}
	var checkEnvelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(holder.out), &checkEnvelope); err != nil {
		t.Fatalf("decode holder check JSON: %v\n%s", err, holder.out)
	}
	if checkEnvelope.SchemaVersion != execution.SchemaVersion || checkEnvelope.Kind != "execution.run.check" {
		t.Fatalf("holder check envelope = %+v", checkEnvelope)
	}
	if !executionCheckIDPattern.MatchString(checkEnvelope.Check.CheckID) {
		t.Fatalf("holder check ID %q does not match chk-* grammar", checkEnvelope.Check.CheckID)
	}
	if checkEnvelope.Check.Result != execution.CheckPassed {
		t.Fatalf("holder check result = %q, want passed", checkEnvelope.Check.Result)
	}

	if _, err := os.Lstat(filepath.Join(identity.RunDir, ".lock")); !os.IsNotExist(err) {
		t.Fatalf("lock still present after holder completed: %v", err)
	}
	finalChecks, err := os.ReadDir(checksDir)
	if err != nil {
		t.Fatalf("read checks dir after join: %v", err)
	}
	if len(finalChecks) != 1 {
		t.Fatalf("final checks count = %d, want 1", len(finalChecks))
	}
	finalBriefs, err := os.ReadDir(briefsDir)
	if err != nil {
		t.Fatalf("read briefs dir after join: %v", err)
	}
	if len(finalBriefs) != 0 {
		t.Fatalf("final briefs count = %d, want 0", len(finalBriefs))
	}
}

func TestExecutionContractCleanRefusals(t *testing.T) {
	t.Run("unowned run directory", func(t *testing.T) {
		dir := executionTestRepo(t)
		identity := initExecutionIntegrityRun(t, dir)
		unownedID := "run-20261113T114500Z-abcdef"
		if identity.RunID == unownedID {
			unownedID = "run-20261113T114500Z-fedcba"
		}
		unownedDir := filepath.Join(filepath.Dir(identity.RunDir), unownedID)
		if err := os.Mkdir(unownedDir, 0o700); err != nil {
			t.Fatalf("mkdir unowned run: %v", err)
		}
		notePath := filepath.Join(unownedDir, "notes.txt")
		if err := os.WriteFile(notePath, []byte("not a run\n"), 0o600); err != nil {
			t.Fatalf("write note: %v", err)
		}

		code, out, errOut := runExecutionCLI(t, dir, "run", "clean", "--run", unownedID, "--yes", "--format", "json")
		if code != 2 {
			t.Fatalf("unowned clean exit = %d, want 2: stdout=%q stderr=%q", code, out, errOut)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("unowned clean wrote unexpected stdout: %q", out)
		}
		if !strings.Contains(errOut, "error:") {
			t.Fatalf("unowned clean stderr missing error prefix: %q", errOut)
		}
		if _, err := os.ReadFile(notePath); err != nil {
			t.Fatalf("unowned run contents were deleted or altered: %v", err)
		}
	})

	t.Run("symlinked run directory", func(t *testing.T) {
		dir := executionTestRepo(t)
		identity := initExecutionIntegrityRun(t, dir)
		target := t.TempDir()
		keepPath := filepath.Join(target, "keep.txt")
		if err := os.WriteFile(keepPath, []byte("keep\n"), 0o600); err != nil {
			t.Fatalf("write keep file: %v", err)
		}
		symlinkID := "run-20261113T114501Z-abcdef"
		if identity.RunID == symlinkID {
			symlinkID = "run-20261113T114501Z-fedcba"
		}
		symlinkPath := filepath.Join(filepath.Dir(identity.RunDir), symlinkID)
		if err := os.Symlink(target, symlinkPath); err != nil {
			t.Skipf("not_run: symlinks unavailable: %v", err)
		}

		code, out, errOut := runExecutionCLI(t, dir, "run", "clean", "--run", symlinkID, "--yes", "--format", "json")
		if code != 2 {
			t.Fatalf("symlinked clean exit = %d, want 2: stdout=%q stderr=%q", code, out, errOut)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("symlinked clean wrote unexpected stdout: %q", out)
		}
		if !strings.Contains(strings.ToLower(errOut), "symlink") {
			t.Fatalf("symlinked clean stderr missing symlink diagnostic: %q", errOut)
		}
		if info, err := os.Lstat(symlinkPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("symlinked entry was modified or deleted: info=%v err=%v", info, err)
		}
		if _, err := os.ReadFile(keepPath); err != nil {
			t.Fatalf("symlink target contents were deleted: %v", err)
		}
	})

	t.Run("moved run directory", func(t *testing.T) {
		dir := executionTestRepo(t)
		srcRoot := t.TempDir()
		code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--run-root", srcRoot, "--format", "json")
		if code != 0 {
			t.Fatalf("run init exit %d: %s", code, errOut)
		}
		identity := decodeRunIdentity(t, out)

		dstRoot := t.TempDir()
		movedDir := filepath.Join(dstRoot, identity.RunID)
		if err := os.Rename(identity.RunDir, movedDir); err != nil {
			t.Fatalf("move run dir: %v", err)
		}

		code, out, errOut = runExecutionCLI(t, dir, "run", "clean", "--run-root", dstRoot, "--run", identity.RunID, "--yes", "--format", "json")
		if code != 2 {
			t.Fatalf("moved clean exit = %d, want 2: stdout=%q stderr=%q", code, out, errOut)
		}
		if strings.TrimSpace(out) != "" {
			t.Fatalf("moved clean wrote unexpected stdout: %q", out)
		}
		if !strings.Contains(errOut, "error:") {
			t.Fatalf("moved clean stderr missing error prefix: %q", errOut)
		}
		if _, err := os.Stat(filepath.Join(movedDir, "run.json")); err != nil {
			t.Fatalf("moved run was deleted or altered: %v", err)
		}
	})

	t.Run("live writer lock", func(t *testing.T) {
		shPath, err := exec.LookPath("sh")
		if err != nil {
			t.Skipf("not_run: sh unavailable: %v", err)
		}
		setExecutionGitIsolationEnv(t)
		dir := executionTestRepo(t)
		identity := initExecutionIntegrityRun(t, dir)

		_, join := startSleepingCheckHolder(t, dir, identity.RunID, identity.RunDir, shPath)
		defer func() { _ = join() }()

		cleanRes := runExecutionCLIDirect(dir, "run", "clean", "--run", identity.RunID, "--yes", "--format", "json")
		if cleanRes.code != 2 {
			t.Fatalf("live-locked clean exit = %d, want 2: stdout=%q stderr=%q", cleanRes.code, cleanRes.out, cleanRes.errOut)
		}
		if strings.TrimSpace(cleanRes.out) != "" {
			t.Fatalf("live-locked clean wrote unexpected stdout: %q", cleanRes.out)
		}
		if !strings.Contains(strings.ToLower(cleanRes.errOut), "locked") {
			t.Fatalf("live-locked clean stderr missing lock diagnostic: %q", cleanRes.errOut)
		}
		if _, err := os.Stat(filepath.Join(identity.RunDir, "run.json")); err != nil {
			t.Fatalf("run.json missing after refused live-lock clean: %v", err)
		}
		if _, err := os.Stat(filepath.Join(identity.RunDir, "checkpoint.json")); err != nil {
			t.Fatalf("checkpoint.json missing after refused live-lock clean: %v", err)
		}

		holder := join()
		if holder.code != 0 {
			t.Fatalf("holder check exit = %d, want 0: stdout=%q stderr=%q", holder.code, holder.out, holder.errOut)
		}
	})

	t.Run("absent run", func(t *testing.T) {
		dir := executionTestRepo(t)
		absentID := "run-20261113T114502Z-abcdef"

		code, out, errOut := runExecutionCLI(t, dir, "run", "clean", "--run", absentID, "--yes", "--format", "json")
		if code != 0 {
			t.Fatalf("absent clean JSON exit = %d, want 0: stderr=%q", code, errOut)
		}
		if strings.TrimSpace(errOut) != "" {
			t.Fatalf("absent clean JSON wrote unexpected stderr: %q", errOut)
		}
		var result execution.CleanResult
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("decode clean JSON: %v\n%s", err, out)
		}
		if result.SchemaVersion != execution.SchemaVersion || result.Kind != "execution.run.clean" {
			t.Fatalf("clean envelope = %+v", result)
		}
		if result.RunID != absentID || !result.AlreadyAbsent || result.Removed {
			t.Fatalf("absent clean result = %+v", result)
		}

		code, out, errOut = runExecutionCLI(t, dir, "run", "clean", "--run", absentID, "--yes")
		if code != 0 {
			t.Fatalf("absent clean text exit = %d, want 0: stderr=%q", code, errOut)
		}
		if !strings.Contains(strings.ToLower(out), "already absent") {
			t.Fatalf("absent clean text output missing 'already absent': %q", out)
		}
	})
}

func TestExecutionContractStatusStale(t *testing.T) {
	shPath, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("not_run: sh unavailable: %v", err)
	}

	t.Run("head changed after passing check", func(t *testing.T) {
		dir := executionTestRepo(t)
		identity := initExecutionIntegrityRun(t, dir)

		code, _, errOut := runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--format", "json", "--", shPath, "-c", "exit 0")
		if code != 0 {
			t.Fatalf("passing check exit = %d: %s", code, errOut)
		}

		code, out, errOut := runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
		if code != 0 {
			t.Fatalf("baseline status exit = %d: %s", code, errOut)
		}
		baselineStatus := decodeExecutionStatus(t, out)
		if baselineStatus.State != execution.StateVerifiedEvidence {
			t.Fatalf("baseline state = %q, want %q", baselineStatus.State, execution.StateVerifiedEvidence)
		}

		if err := os.WriteFile(filepath.Join(dir, "later.txt"), []byte("later\n"), 0o600); err != nil {
			t.Fatalf("write committed file: %v", err)
		}
		executionGitOutput(t, dir, "add", "later.txt")
		executionGitOutput(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-qm", "later")

		code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
		if code != 0 {
			t.Fatalf("stale head status exit = %d, want 0: stderr=%q", code, errOut)
		}
		if strings.TrimSpace(errOut) != "" {
			t.Fatalf("stale head status wrote unexpected stderr: %q", errOut)
		}
		status := decodeExecutionStatus(t, out)
		if err := execution.ValidateRunID(status.RunID); err != nil || status.RunID != identity.RunID {
			t.Fatalf("status run_id = %q, want %q (err=%v)", status.RunID, identity.RunID, err)
		}
		if status.State != execution.StateStale {
			t.Fatalf("state = %q, want %q (reasons=%v)", status.State, execution.StateStale, status.StaleReasons)
		}
		if !slices.Contains(status.StaleReasons, execution.ReasonHeadChanged) {
			t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonHeadChanged)
		}
		if slices.Contains(status.StaleReasons, execution.ReasonWorktreeChanged) {
			t.Fatalf("clean commit unexpectedly reported %q: %v", execution.ReasonWorktreeChanged, status.StaleReasons)
		}
		if status.NextPermittedInvestigation != execution.InvestigateChangedPaths {
			t.Fatalf("next_permitted_investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateChangedPaths)
		}
		if status.LastObservation == nil || status.CurrentObservation == nil || status.LastObservation.Head == status.CurrentObservation.Head {
			t.Fatalf("expected different observed HEADs: last=%+v current=%+v", status.LastObservation, status.CurrentObservation)
		}
	})

	t.Run("worktree changed after passing check", func(t *testing.T) {
		dir := executionTestRepo(t)
		identity := initExecutionIntegrityRun(t, dir)

		code, _, errOut := runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--format", "json", "--", shPath, "-c", "exit 0")
		if code != 0 {
			t.Fatalf("passing check exit = %d: %s", code, errOut)
		}

		code, out, errOut := runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
		if code != 0 {
			t.Fatalf("baseline status exit = %d: %s", code, errOut)
		}
		baselineStatus := decodeExecutionStatus(t, out)
		if baselineStatus.State != execution.StateVerifiedEvidence {
			t.Fatalf("baseline state = %q, want %q", baselineStatus.State, execution.StateVerifiedEvidence)
		}

		if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty\n"), 0o600); err != nil {
			t.Fatalf("write dirty file: %v", err)
		}

		code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
		if code != 0 {
			t.Fatalf("stale worktree status exit = %d, want 0: stderr=%q", code, errOut)
		}
		if strings.TrimSpace(errOut) != "" {
			t.Fatalf("stale worktree status wrote unexpected stderr: %q", errOut)
		}
		status := decodeExecutionStatus(t, out)
		if err := execution.ValidateRunID(status.RunID); err != nil || status.RunID != identity.RunID {
			t.Fatalf("status run_id = %q, want %q (err=%v)", status.RunID, identity.RunID, err)
		}
		if status.State != execution.StateStale {
			t.Fatalf("state = %q, want %q (reasons=%v)", status.State, execution.StateStale, status.StaleReasons)
		}
		if !slices.Contains(status.StaleReasons, execution.ReasonWorktreeChanged) {
			t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonWorktreeChanged)
		}
		if slices.Contains(status.StaleReasons, execution.ReasonHeadChanged) {
			t.Fatalf("dirty worktree unexpectedly reported %q: %v", execution.ReasonHeadChanged, status.StaleReasons)
		}
		if status.NextPermittedInvestigation != execution.InvestigateChangedPaths {
			t.Fatalf("next_permitted_investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateChangedPaths)
		}
		if status.LastObservation == nil || status.CurrentObservation == nil || status.LastObservation.Head != status.CurrentObservation.Head {
			t.Fatalf("expected matching HEADs for dirty worktree: last=%+v current=%+v", status.LastObservation, status.CurrentObservation)
		}
	})
}
