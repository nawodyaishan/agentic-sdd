package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCheckHelperProcess is the fixture process for check execution tests. It
// only acts when the parent sets EXECUTION_CHECK_MODE, so an ordinary package run
// is a no-op.
func TestCheckHelperProcess(t *testing.T) {
	if os.Getenv("EXECUTION_CHECK_HELPER") != "1" {
		return
	}
	switch mode := os.Getenv("EXECUTION_CHECK_MODE"); mode {
	case "pass":
		fmt.Fprintln(os.Stdout, "helper checks passed")
		os.Exit(0)
	case "fail-early":
		fmt.Fprintln(os.Stdout, "line before the failure")
		fmt.Fprintln(os.Stdout, "FAIL: cannot continue: fixture dependency missing")
		for index := 0; index < 1500; index++ {
			fmt.Fprintf(os.Stdout, "tail noise %d\n", index)
		}
		os.Exit(3)
	case "exit-fail":
		fmt.Fprintln(os.Stderr, "error: synthetic failure")
		os.Exit(1)
	case "markers":
		for index := 0; index < 60; index++ {
			fmt.Fprintf(os.Stdout, "error %02d: synthetic marker %d\n", index, index)
		}
		os.Exit(1)
	case "plain":
		for index := 1; index <= 30; index++ {
			fmt.Fprintf(os.Stdout, "plain line %d\n", index)
		}
		os.Exit(0)
	case "sleep":
		milliseconds := 30000
		if raw := os.Getenv("EXECUTION_CHECK_SLEEP_MS"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid sleep duration")
				os.Exit(98)
			}
			milliseconds = parsed
		}
		fmt.Fprintln(os.Stdout, "helper sleeping")
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		os.Exit(0)
	case "flood":
		block := bytes.Repeat([]byte("flood-output-line\n"), 256)
		for {
			if _, err := os.Stdout.Write(block); err != nil {
				os.Exit(1)
			}
		}
	case "dirty":
		if err := os.WriteFile("dirty-marker.txt", []byte("mutated by the check\n"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "cannot write marker: "+err.Error())
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "wrote the marker")
		os.Exit(0)
	case "run-check":
		// The runner is killed while its own RunCheck is in flight, so the
		// checkpoint must keep a running record and no final observation.
		os.Setenv("EXECUTION_CHECK_MODE", "sleep")
		os.Setenv("EXECUTION_CHECK_SLEEP_MS", "8000")
		_, err := RunCheck(CheckRequest{
			RunDir:  os.Getenv("EXECUTION_CHECK_RUN_DIR"),
			Argv:    []string{os.Args[0], "-test.run=^TestCheckHelperProcess$"},
			Timeout: time.Minute,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "run check: "+err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(97)
	}
}

// checkTestArgv returns the fixture command for one helper mode and exports the
// marker environment the child process needs.
func checkTestArgv(t *testing.T, mode string) []string {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary path: %v", err)
	}
	t.Setenv("EXECUTION_CHECK_HELPER", "1")
	t.Setenv("EXECUTION_CHECK_MODE", mode)
	return []string{binary, "-test.run=^TestCheckHelperProcess$"}
}

func checkTestRun(t *testing.T) RunIdentity {
	t.Helper()
	_, _, identity := workspaceTestRun(t)
	return identity
}

// waitForCheckpointState polls until the checkpoint reaches want. The check
// writer publishes the running state before starting the command.
func waitForCheckpointState(t *testing.T, runDir string, want CheckpointState) *Checkpoint {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		checkpoint, err := LoadCheckpoint(runDir)
		if err == nil && checkpoint.State == want {
			return checkpoint
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("checkpoint did not reach state %q", want)
	return nil
}

func TestCheckPassReachesVerifiedEvidence(t *testing.T) {
	identity := checkTestRun(t)
	argv := checkTestArgv(t, "pass")
	record, err := RunCheck(CheckRequest{
		RunDir:  identity.RunDir,
		RunID:   identity.RunID,
		TaskID:  "T01",
		Argv:    argv,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want passed (summary %v)", record.Result, record.Summary)
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", record.ExitCode)
	}
	if record.Signal != "" || record.TimedOut || record.OutputLimit {
		t.Fatalf("unexpected failure flags: %+v", record)
	}
	if record.WorktreeChangedDuringCheck {
		t.Fatal("a check that only writes inside the run reported a worktree change")
	}
	if record.EndedAt == nil || record.EndedAt.Before(record.StartedAt) {
		t.Fatalf("timestamps = %v..%v", record.StartedAt, record.EndedAt)
	}
	if want := "helper checks passed"; len(record.Summary) == 0 || record.Summary[len(record.Summary)-1] != want {
		t.Fatalf("summary = %v, want the last line %q", record.Summary, want)
	}
	if data := repoTestRead(t, record.StdoutPath); !strings.Contains(data, "helper checks passed") {
		t.Fatalf("stdout log = %q", data)
	}

	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateVerifiedEvidence {
		t.Fatalf("checkpoint state = %q, want verified-evidence", checkpoint.State)
	}
	task, present := checkpoint.Tasks["T01"]
	if !present || task.State != StateVerifiedEvidence || task.LastCheckID != record.CheckID {
		t.Fatalf("task state = %+v", task)
	}
	if len(checkpoint.Checks) != 1 || checkpoint.Checks[0].Result != CheckPassed {
		t.Fatalf("checkpoint checks = %+v", checkpoint.Checks)
	}
	if checkpoint.LastObservation == nil {
		t.Fatal("checkpoint has no end observation")
	}

	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateVerifiedEvidence {
		t.Fatalf("status state = %q, want verified-evidence (reasons %v)", status.State, status.StaleReasons)
	}
	if len(status.StaleReasons) != 0 {
		t.Fatalf("status stale reasons = %v", status.StaleReasons)
	}
}

func TestCheckArtifactsAndPrivateProvenance(t *testing.T) {
	identity := checkTestRun(t)
	argv := checkTestArgv(t, "pass")
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: argv})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	checkDir := filepath.Join(identity.RunDir, "checks", record.CheckID)
	info, err := os.Stat(checkDir)
	if err != nil {
		t.Fatalf("stat check directory: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("check directory mode = %o, want 700", info.Mode().Perm())
	}
	for _, name := range []string{"meta.json", "argv.json", "stdout.log", "stderr.log"} {
		info, err := os.Stat(filepath.Join(checkDir, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %o, want 600", name, info.Mode().Perm())
		}
	}
	var document checkArgvDocument
	if err := json.Unmarshal([]byte(repoTestRead(t, record.ArgvPath)), &document); err != nil {
		t.Fatalf("decode argv.json: %v", err)
	}
	if document.Digest != record.ArgvDigest {
		t.Fatalf("argv digest = %q, want %q", document.Digest, record.ArgvDigest)
	}
	if len(document.Argv) != len(argv) || document.Argv[1] != argv[1] {
		t.Fatalf("argv.json = %v", document.Argv)
	}
	var meta checkMeta
	if err := json.Unmarshal([]byte(repoTestRead(t, filepath.Join(checkDir, "meta.json"))), &meta); err != nil {
		t.Fatalf("decode meta.json: %v", err)
	}
	if !meta.Finished || meta.Kind != kindCheckMeta || meta.Check.CheckID != record.CheckID {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.MaxOutputBytes != DefaultMaxOutputBytes {
		t.Fatalf("meta output limit = %d, want %d", meta.MaxOutputBytes, DefaultMaxOutputBytes)
	}
	if meta.StartObservation == nil || meta.EndObservation == nil {
		t.Fatalf("meta observations = %+v", meta)
	}
	if record.WorkingDir != identity.Scope.RepoRoot {
		t.Fatalf("working dir = %q, want %q", record.WorkingDir, identity.Scope.RepoRoot)
	}
	if record.CommandProgram != filepath.Base(argv[0]) {
		t.Fatalf("program = %q", record.CommandProgram)
	}
	if record.CommandArgCount != len(argv)-1 {
		t.Fatalf("arg count = %d, want %d", record.CommandArgCount, len(argv)-1)
	}
	if record.ArgvDigest == "" || !strings.HasPrefix(record.ArgvDigest, "sha256:") {
		t.Fatalf("argv digest = %q", record.ArgvDigest)
	}
}

func TestCheckFailureFarFromTail(t *testing.T) {
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: checkTestArgv(t, "fail-early")})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckFailed {
		t.Fatalf("result = %q, want failed", record.Result)
	}
	if record.ExitCode == nil || *record.ExitCode != 3 {
		t.Fatalf("exit code = %v, want 3", record.ExitCode)
	}
	joined := strings.Join(record.Summary, "\n")
	if !strings.Contains(joined, "FAIL: cannot continue") {
		t.Fatalf("summary missed the early failure:\n%s", joined)
	}
	for _, line := range record.Summary {
		if strings.Contains(line, "tail noise") {
			t.Fatalf("summary contains tail noise only: %v", record.Summary)
		}
	}
	log := repoTestRead(t, record.StdoutPath)
	if !strings.Contains(log, "FAIL: cannot continue") || !strings.Contains(log, "tail noise 1499") {
		t.Fatalf("full log is not complete: %d bytes", len(log))
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedFailed {
		t.Fatalf("checkpoint state = %q, want observed-failed", checkpoint.State)
	}
	if checkpoint.Tasks["T01"].State != StateObservedFailed {
		t.Fatalf("task state = %q", checkpoint.Tasks["T01"].State)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateObservedFailed {
		t.Fatalf("status state = %q, want observed-failed", status.State)
	}
}

func TestCheckTimeout(t *testing.T) {
	identity := checkTestRun(t)
	t.Setenv("EXECUTION_CHECK_SLEEP_MS", "20000")
	argv := checkTestArgv(t, "sleep")
	started := time.Now()
	record, err := RunCheck(CheckRequest{
		RunDir:  identity.RunDir,
		TaskID:  "T01",
		Argv:    argv,
		Timeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("timeout did not terminate the command promptly: %s", elapsed)
	}
	if record.Result != CheckTimedOut {
		t.Fatalf("result = %q, want timed_out", record.Result)
	}
	if !record.TimedOut {
		t.Fatal("record does not report the timeout")
	}
	if record.ExitCode != nil {
		t.Fatalf("exit code = %v, want nil for a terminated process", *record.ExitCode)
	}
	if runtime.GOOS != "windows" && record.Signal == "" {
		t.Fatal("record does not report the terminating signal")
	}
	if record.StdoutBytes == 0 {
		t.Fatal("output before the timeout was not preserved")
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedPartial {
		t.Fatalf("checkpoint state = %q, want observed-partial", checkpoint.State)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateObservedPartial {
		t.Fatalf("status state = %q, want observed-partial", status.State)
	}
}

func TestCheckSignalIsInterrupted(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh is unavailable: %v", err)
	}
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{
		RunDir:  identity.RunDir,
		TaskID:  "T01",
		Argv:    []string{shell, "-c", "kill -TERM $$"},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckInterrupted {
		t.Fatalf("result = %q, want interrupted", record.Result)
	}
	if record.Signal == "" {
		t.Fatal("record does not report the signal")
	}
	if record.ExitCode != nil {
		t.Fatalf("exit code = %v, want nil for a signalled process", *record.ExitCode)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedPartial {
		t.Fatalf("checkpoint state = %q, want observed-partial", checkpoint.State)
	}
}

func TestCheckOutputLimitTerminatesAndTruncates(t *testing.T) {
	identity := checkTestRun(t)
	started := time.Now()
	record, err := RunCheck(CheckRequest{
		RunDir:         identity.RunDir,
		TaskID:         "T01",
		Argv:           checkTestArgv(t, "flood"),
		Timeout:        30 * time.Second,
		MaxOutputBytes: 1024,
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("output cap did not terminate the command promptly: %s", elapsed)
	}
	if record.Result != CheckOutputLimit {
		t.Fatalf("result = %q, want output_limit", record.Result)
	}
	if !record.OutputLimit || !record.StdoutTruncated {
		t.Fatalf("record does not report a truncated stdout: %+v", record)
	}
	if record.StdoutBytes != 1024 {
		t.Fatalf("stdout bytes = %d, want the 1024-byte cap", record.StdoutBytes)
	}
	if info, statErr := os.Stat(record.StdoutPath); statErr != nil || info.Size() != 1024 {
		t.Fatalf("stdout log = %v (%v), want 1024 bytes", info, statErr)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedPartial {
		t.Fatalf("checkpoint state = %q, want observed-partial", checkpoint.State)
	}
}

func TestCheckSummaryTruncation(t *testing.T) {
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, Argv: checkTestArgv(t, "markers")})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckFailed {
		t.Fatalf("result = %q, want failed", record.Result)
	}
	if len(record.Summary) != checkSummaryMatchLimit {
		t.Fatalf("summary lines = %d, want %d", len(record.Summary), checkSummaryMatchLimit)
	}
	if !record.SummaryTruncated {
		t.Fatal("summary is truncated but the flag is not set")
	}
}

func TestCheckSummaryTailFallback(t *testing.T) {
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, Argv: checkTestArgv(t, "plain")})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want passed", record.Result)
	}
	if len(record.Summary) != checkSummaryTailLines {
		t.Fatalf("summary lines = %d, want %d", len(record.Summary), checkSummaryTailLines)
	}
	if record.Summary[0] != "plain line 11" || record.Summary[len(record.Summary)-1] != "plain line 30" {
		t.Fatalf("summary tail = %v", record.Summary)
	}
	if record.SummaryTruncated {
		t.Fatal("a tail excerpt is not a truncated marker list")
	}
}

func TestCheckNoSecretEcho(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh is unavailable: %v", err)
	}
	identity := checkTestRun(t)
	secret := "secret-token-ABC123-do-not-echo"
	t.Setenv("EXECUTION_TEST_SECRET", "env-secret-XYZ-do-not-echo")
	record, err := RunCheck(CheckRequest{
		RunDir:  identity.RunDir,
		TaskID:  "T01",
		Argv:    []string{shell, "-c", "printf 'done\\n'", secret},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want passed", record.Result)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	meta := repoTestRead(t, filepath.Join(identity.RunDir, "checks", record.CheckID, "meta.json"))
	for name, surface := range map[string]string{"record": string(encoded), "meta": meta} {
		if strings.Contains(surface, secret) {
			t.Fatalf("%s echoed an argument the caller supplied", name)
		}
		if strings.Contains(surface, "env-secret-XYZ") {
			t.Fatalf("%s echoed the environment", name)
		}
	}
	argvDocument := repoTestRead(t, record.ArgvPath)
	if !strings.Contains(argvDocument, secret) {
		t.Fatal("private argv.json lost the exact argument vector")
	}
}

func TestCheckMissingProgramIsNotStarted(t *testing.T) {
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{
		RunDir: identity.RunDir,
		TaskID: "T01",
		Argv:   []string{"/nonexistent/execution-check-fixture"},
	})
	if err != nil {
		t.Fatalf("run check returned a wrapper error for an observed start failure: %v", err)
	}
	if record.Result != CheckNotStarted {
		t.Fatalf("result = %q, want not_started", record.Result)
	}
	if record.ExitCode != nil || record.Signal != "" {
		t.Fatalf("a command that never ran reported exit/signal: %+v", record)
	}
	joined := strings.Join(record.Summary, "\n")
	if joined == "" {
		t.Fatal("summary does not carry the exact start error")
	}
	if !strings.Contains(joined, "/nonexistent/execution-check-fixture") {
		t.Fatalf("summary = %q", joined)
	}
	meta := repoTestRead(t, filepath.Join(identity.RunDir, "checks", record.CheckID, "meta.json"))
	if !strings.Contains(meta, "no such file") && !strings.Contains(meta, "executable file not found") {
		t.Fatalf("meta.json does not carry the exact error: %s", meta)
	}
	for _, name := range []string{record.StdoutPath, record.StderrPath} {
		info, statErr := os.Stat(name)
		if statErr != nil {
			t.Fatalf("stat log: %v", statErr)
		}
		if info.Size() != 0 {
			t.Fatalf("log %s has %d bytes for a command that never ran", name, info.Size())
		}
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedFailed {
		t.Fatalf("checkpoint state = %q, want observed-failed", checkpoint.State)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateObservedFailed {
		t.Fatalf("status state = %q, want observed-failed", status.State)
	}
}

func TestCheckRequestValidation(t *testing.T) {
	identity := checkTestRun(t)
	argv := checkTestArgv(t, "pass")
	cases := []struct {
		name string
		req  CheckRequest
		want error
	}{
		{"empty vector", CheckRequest{RunDir: identity.RunDir}, ErrUsage},
		{"empty program", CheckRequest{RunDir: identity.RunDir, Argv: []string{"  "}}, ErrUsage},
		{"NUL argument", CheckRequest{RunDir: identity.RunDir, Argv: []string{argv[0], "a\x00b"}}, ErrUsage},
		{"negative timeout", CheckRequest{RunDir: identity.RunDir, Argv: argv, Timeout: -time.Second}, ErrUsage},
		{"missing run directory", CheckRequest{Argv: argv}, ErrUsage},
		{"unknown run", CheckRequest{RunDir: filepath.Join(t.TempDir(), "absent"), Argv: argv}, ErrNotFound},
		{"invalid run ID", CheckRequest{RunDir: identity.RunDir, RunID: "../run", Argv: argv}, ErrUsage},
		{"mismatched run ID", CheckRequest{RunDir: identity.RunDir, RunID: "run-20260101T000000Z-abcdef", Argv: argv}, ErrConflict},
		{"unsafe task ID", CheckRequest{RunDir: identity.RunDir, TaskID: "../T01", Argv: argv}, ErrUsage},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := RunCheck(testCase.req); !errors.Is(err, testCase.want) {
				t.Fatalf("RunCheck error = %v, want %v", err, testCase.want)
			}
		})
	}
	entries, err := os.ReadDir(filepath.Join(identity.RunDir, "checks"))
	if err != nil {
		t.Fatalf("read checks: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("a rejected request created check artifacts: %v", entries)
	}
}

func TestCheckConcurrentLockAndRunningStatus(t *testing.T) {
	identity := checkTestRun(t)
	t.Setenv("EXECUTION_CHECK_SLEEP_MS", "1500")
	argv := checkTestArgv(t, "sleep")
	type outcome struct {
		record CheckRecord
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: argv, Timeout: 30 * time.Second})
		done <- outcome{record: record, err: err}
	}()
	running := waitForCheckpointState(t, identity.RunDir, StateRunning)
	if len(running.Checks) != 1 || running.Checks[0].Result != CheckRunning {
		t.Fatalf("in-progress checkpoint = %+v", running.Checks)
	}
	if running.Tasks["T01"].State != StateRunning {
		t.Fatalf("in-progress task state = %+v", running.Tasks["T01"])
	}
	if running.Checks[0].ArgvDigest == "" || running.Checks[0].StartedAt.IsZero() {
		t.Fatalf("in-progress record lacks start provenance: %+v", running.Checks[0])
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status while running: %v", err)
	}
	if status.State != StateRunning {
		t.Fatalf("status while running = %q, want running", status.State)
	}
	if _, err := RunCheck(CheckRequest{RunDir: identity.RunDir, Argv: argv, Timeout: 30 * time.Second}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second concurrent check error = %v, want ErrConflict", err)
	}
	result := <-done
	if result.err != nil {
		t.Fatalf("first check: %v", result.err)
	}
	if result.record.Result != CheckPassed {
		t.Fatalf("first check result = %q", result.record.Result)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if len(checkpoint.Checks) != 1 {
		t.Fatalf("concurrent attempt interleaved the checkpoint: %+v", checkpoint.Checks)
	}
	if checkpoint.State != StateVerifiedEvidence {
		t.Fatalf("checkpoint state = %q, want verified-evidence", checkpoint.State)
	}
}

func TestCheckReleasesLockOnEveryPath(t *testing.T) {
	identity := checkTestRun(t)
	paths := map[string][]string{
		"pass":        checkTestArgv(t, "pass"),
		"not started": {"/nonexistent/execution-check-fixture"},
	}
	for name, argv := range paths {
		t.Run(name, func(t *testing.T) {
			if _, err := RunCheck(CheckRequest{RunDir: identity.RunDir, Argv: argv}); err != nil {
				t.Fatalf("run check: %v", err)
			}
			lock, err := acquireRunLock(identity.RunDir, "test")
			if err != nil {
				t.Fatalf("run lock was not released: %v", err)
			}
			if err := lock.release(); err != nil {
				t.Fatalf("release test lock: %v", err)
			}
		})
	}
}

func TestCheckWorktreeChangedDuringCheck(t *testing.T) {
	identity := checkTestRun(t)
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: checkTestArgv(t, "dirty")})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want the command's own exit status", record.Result)
	}
	if !record.WorktreeChangedDuringCheck {
		t.Fatal("a check that mutated the worktree was reported as unchanged")
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateObservedPartial {
		t.Fatalf("checkpoint state = %q, want observed-partial", checkpoint.State)
	}
	if checkpoint.Tasks["T01"].State != StateObservedPartial {
		t.Fatalf("task state = %q, want observed-partial", checkpoint.Tasks["T01"].State)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateObservedPartial {
		t.Fatalf("status state = %q, want observed-partial", status.State)
	}
}

func TestCheckLaterPassReachesVerifiedEvidence(t *testing.T) {
	identity := checkTestRun(t)
	failed, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: checkTestArgv(t, "exit-fail")})
	if err != nil {
		t.Fatalf("failing check: %v", err)
	}
	if failed.Result != CheckFailed {
		t.Fatalf("first result = %q", failed.Result)
	}
	passed, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: checkTestArgv(t, "pass")})
	if err != nil {
		t.Fatalf("passing check: %v", err)
	}
	if passed.Result != CheckPassed {
		t.Fatalf("second result = %q", passed.Result)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if len(checkpoint.Checks) != 2 {
		t.Fatalf("checks = %d, want 2", len(checkpoint.Checks))
	}
	if checkpoint.State != StateVerifiedEvidence {
		t.Fatalf("checkpoint state = %q, want verified-evidence", checkpoint.State)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateVerifiedEvidence {
		t.Fatalf("status state = %q, want verified-evidence (reasons %v)", status.State, status.StaleReasons)
	}
}

func TestCheckKilledWriterLeavesInterruptedPartialState(t *testing.T) {
	identity := checkTestRun(t)
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary path: %v", err)
	}
	t.Setenv("EXECUTION_CHECK_HELPER", "1")
	t.Setenv("EXECUTION_CHECK_MODE", "run-check")
	t.Setenv("EXECUTION_CHECK_RUN_DIR", identity.RunDir)
	cmd := exec.Command(binary, "-test.run=^TestCheckHelperProcess$")
	cmd.Env = os.Environ()
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the check writer: %v", err)
	}
	running := waitForCheckpointState(t, identity.RunDir, StateRunning)
	if len(running.Checks) != 1 || running.Checks[0].Result != CheckRunning {
		t.Fatalf("in-progress checkpoint = %+v", running.Checks)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill the check writer: %v", err)
	}
	_ = cmd.Wait()

	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.State != StateRunning {
		t.Fatalf("checkpoint state = %q, want the interrupted running record", checkpoint.State)
	}
	if len(checkpoint.Checks) != 1 || checkpoint.Checks[0].Result != CheckRunning {
		t.Fatalf("a killed writer produced a final record: %+v", checkpoint.Checks)
	}
	status, err := StatusRun(identity.RunRoot, identity.RunID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.State != StateObservedPartial {
		t.Fatalf("status state = %q, want observed-partial", status.State)
	}
	if !statusHasReason(status, ReasonInterruptedCheck) {
		t.Fatalf("status reasons = %v, want interrupted_check", status.StaleReasons)
	}
	if status.NextPermittedInvestigation != InvestigateRerunCheck {
		t.Fatalf("next investigation = %q, want %q", status.NextPermittedInvestigation, InvestigateRerunCheck)
	}
}

func TestCheckEvidenceWriteFailureIsNotAPass(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	identity := checkTestRun(t)
	checksDir := filepath.Join(identity.RunDir, "checks")
	if err := os.Chmod(checksDir, 0o500); err != nil {
		t.Fatalf("restrict checks directory: %v", err)
	}
	defer os.Chmod(checksDir, 0o700)
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, TaskID: "T01", Argv: checkTestArgv(t, "pass")})
	if err == nil {
		t.Fatalf("a check that could not persist evidence reported success: %+v", record)
	}
	if record.Result == CheckPassed {
		t.Fatal("a check that could not persist evidence was reported as passed")
	}
	checkpoint, loadErr := LoadCheckpoint(identity.RunDir)
	if loadErr != nil {
		t.Fatalf("load checkpoint: %v", loadErr)
	}
	if checkpoint.State != StatePrepared || len(checkpoint.Checks) != 0 {
		t.Fatalf("failed evidence write left a claim: state=%q checks=%+v", checkpoint.State, checkpoint.Checks)
	}
	lock, lockErr := acquireRunLock(identity.RunDir, "test")
	if lockErr != nil {
		t.Fatalf("run lock was not released after an evidence write failure: %v", lockErr)
	}
	if err := lock.release(); err != nil {
		t.Fatalf("release test lock: %v", err)
	}
}

func TestCheckDirectScopeDefaultsTask(t *testing.T) {
	root := repoTestInit(t)
	repo, err := OpenRepository(root)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	scope, err := DirectScope(repo, "direct fixture")
	if err != nil {
		t.Fatalf("direct scope: %v", err)
	}
	identity, err := InitRun(InitRequest{Scope: scope, ToolVersion: "test"})
	if err != nil {
		t.Fatalf("init run: %v", err)
	}
	record, err := RunCheck(CheckRequest{RunDir: identity.RunDir, Argv: checkTestArgv(t, "pass")})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.TaskID != "direct" {
		t.Fatalf("task ID = %q, want direct", record.TaskID)
	}
	checkpoint, err := LoadCheckpoint(identity.RunDir)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.Tasks["direct"].State != StateVerifiedEvidence {
		t.Fatalf("direct task state = %+v", checkpoint.Tasks["direct"])
	}
}
