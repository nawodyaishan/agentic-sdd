package execution

// Feature 022 B2 (T03, O-06): owned process-group completion. These fixtures use
// real helper processes. The post-exit grace and termination waits are shortened
// through package variables; no fixture waits out a production window.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// b2Argv re-executes the test binary as a helper in the given mode.
func b2Argv(t *testing.T, mode string) []string {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary path: %v", err)
	}
	t.Setenv("EXECUTION_B2_HELPER", "1")
	t.Setenv("EXECUTION_B2_MODE", mode)
	return []string{binary, "-test.run=^TestB2GroupHelperProcess$"}
}

// TestB2GroupHelperProcess is the fixture command. It only runs when re-executed
// by a b2 fixture with EXECUTION_B2_HELPER=1.
func TestB2GroupHelperProcess(t *testing.T) {
	if os.Getenv("EXECUTION_B2_HELPER") != "1" {
		return
	}
	switch os.Getenv("EXECUTION_B2_MODE") {
	case "none":
		fmt.Fprintln(os.Stdout, "b2 leader without children")
		os.Exit(0)
	case "survivor":
		// The leader starts a same-group child whose stdio is /dev/null, so no
		// pipe from the check is held open, then exits 0 while the child runs.
		child := exec.Command(os.Args[0], "-test.run=^TestB2GroupHelperProcess$")
		child.Env = append(os.Environ(),
			"EXECUTION_B2_HELPER=1",
			"EXECUTION_B2_MODE=sleep",
		)
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "cannot start the survivor")
			os.Exit(99)
		}
		if pidFile := os.Getenv("EXECUTION_B2_PID_FILE"); pidFile != "" {
			if err := os.WriteFile(pidFile, []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o600); err != nil {
				fmt.Fprintln(os.Stderr, "cannot write pid file")
				os.Exit(99)
			}
		}
		fmt.Fprintln(os.Stdout, "b2 leader exiting with a survivor")
		os.Exit(0)
	case "sleep":
		milliseconds := 30000
		if raw := os.Getenv("EXECUTION_B2_CHILD_MS"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				os.Exit(98)
			}
			milliseconds = parsed
		}
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		os.Exit(0)
	}
	os.Exit(97)
}

// restoreB2GroupTiming shortens the post-exit grace and termination waits for one
// test, and restores the production hooks when the test ends.
func restoreB2GroupTiming(t *testing.T, grace, termWait time.Duration) {
	t.Helper()
	previousGrace, previousTerm := checkPostExitGrace, checkTermWait
	checkPostExitGrace, checkTermWait = grace, termWait
	t.Cleanup(func() { checkPostExitGrace, checkTermWait = previousGrace, previousTerm })
}

// b2RecordSignals replaces the group signaler with a recorder that never sends a
// real signal. It returns the slice of recorded signals.
func b2RecordSignals(t *testing.T) *[]os.Signal {
	t.Helper()
	sent := &[]os.Signal{}
	previous := checkGroupSignal
	checkGroupSignal = func(pgid int, sig os.Signal) error {
		*sent = append(*sent, sig)
		return nil
	}
	t.Cleanup(func() { checkGroupSignal = previous })
	return sent
}

func b2RunSurvivorCheck(t *testing.T, identity RunIdentity, mode string) (CheckRecord, error, []int) {
	t.Helper()
	pidFile := filepath.Join(t.TempDir(), "survivor.pid")
	t.Setenv("EXECUTION_B2_PID_FILE", pidFile)
	argv := b2Argv(t, mode)
	record, err := RunCheck(CheckRequest{
		RunDir:  identity.RunDir,
		TaskID:  "T01",
		Argv:    argv,
		Timeout: 30 * time.Second,
	})
	var pids []int
	if data, readErr := os.ReadFile(pidFile); readErr == nil {
		for _, field := range strings.Fields(string(data)) {
			if pid, convErr := strconv.Atoi(field); convErr == nil && pid > 0 {
				pids = append(pids, pid)
			}
		}
	}
	return record, err, pids
}

// TestB2NoChildPassesGroupGate: a leader that exits 0 with no owned children keeps
// its passing result after the group gate.
func TestB2NoChildPassesGroupGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("owned process groups are not available on this platform")
	}
	restoreB2GroupTiming(t, 500*time.Millisecond, 200*time.Millisecond)
	identity := checkTestRun(t)
	record, err, _ := b2RunSurvivorCheck(t, identity, "none")
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want passed (summary %v)", record.Result, record.Summary)
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", record.ExitCode)
	}
}

// TestB2RedirectedSurvivorFailsGroupGate is the AC-01 regression: the leader exits
// 0 while a redirected same-group child keeps running. The check must not pass,
// the leader exit must be preserved, timed_out must stay false, and the survivor
// must be terminated by the bounded escalation.
func TestB2RedirectedSurvivorFailsGroupGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("owned process groups are not available on this platform")
	}
	if runtime.GOOS != "linux" {
		t.Skip("the survivor escalation is Linux-pinned; other hosts classify only")
	}
	restoreB2GroupTiming(t, 300*time.Millisecond, 200*time.Millisecond)
	t.Setenv("EXECUTION_B2_CHILD_MS", "30000")
	identity := checkTestRun(t)
	record, err, pids := b2RunSurvivorCheck(t, identity, "survivor")
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckFailed {
		t.Fatalf("result = %q, want failed (summary %v)", record.Result, record.Summary)
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("leader exit = %v, want the observed 0 preserved", record.ExitCode)
	}
	if record.TimedOut {
		t.Fatal("a survivor-only failure must not report timed_out")
	}
	if !strings.Contains(strings.Join(record.Summary, "\n"), "process group") {
		t.Fatalf("summary does not name the surviving group: %v", record.Summary)
	}
	if len(pids) != 1 {
		t.Fatalf("survivor pid file = %v, want one pid", pids)
	}
	assertProcessesGone(t, pids)
	checkpoint, loadErr := LoadCheckpoint(identity.RunDir)
	if loadErr != nil {
		t.Fatalf("load checkpoint: %v", loadErr)
	}
	if checkpoint.State == StateVerifiedEvidence {
		t.Fatal("a survivor-failed check reached verified-evidence")
	}
}

// TestB2SurvivorExitingWithinGracePasses: a same-group child that exits on its own
// inside the post-exit grace leaves the leader's passing result intact.
func TestB2SurvivorExitingWithinGracePasses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("owned process groups are not available on this platform")
	}
	if runtime.GOOS != "linux" {
		t.Skip("the pinned grace observation is Linux-only")
	}
	restoreB2GroupTiming(t, 2*time.Second, 200*time.Millisecond)
	t.Setenv("EXECUTION_B2_CHILD_MS", "50")
	identity := checkTestRun(t)
	record, err, pids := b2RunSurvivorCheck(t, identity, "survivor")
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckPassed {
		t.Fatalf("result = %q, want passed (summary %v)", record.Result, record.Summary)
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("exit code = %v, want 0", record.ExitCode)
	}
	assertProcessesGone(t, pids)
}

// TestB2UncertainGroupSendsNoSignal: when the group cannot be observed after the
// leader exits, the check fails with a bounded cause, the leader exit is kept,
// and no group signal is sent. The survivor is cleaned up by the test itself.
func TestB2UncertainGroupSendsNoSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("owned process groups are not available on this platform")
	}
	if runtime.GOOS != "linux" {
		t.Skip("the injected uncertain probe targets the Linux pinned observer")
	}
	restoreB2GroupTiming(t, 300*time.Millisecond, 200*time.Millisecond)
	t.Setenv("EXECUTION_B2_CHILD_MS", "30000")
	sent := b2RecordSignals(t)
	previousMembers := checkGroupMembers
	checkGroupMembers = func(pgid, leader int) (checkGroupMembership, error) {
		return checkGroupMembership{}, errors.New("injected uncertain process group")
	}
	t.Cleanup(func() { checkGroupMembers = previousMembers })

	identity := checkTestRun(t)
	record, err, pids := b2RunSurvivorCheck(t, identity, "survivor")
	t.Cleanup(func() {
		for _, pid := range pids {
			if process, err := os.FindProcess(pid); err == nil {
				_ = process.Kill()
			}
		}
	})
	if err != nil {
		t.Fatalf("run check: %v", err)
	}
	if record.Result != CheckFailed {
		t.Fatalf("result = %q, want failed for an uncertain group", record.Result)
	}
	if record.TimedOut {
		t.Fatal("an uncertain group must not report timed_out")
	}
	if record.ExitCode == nil || *record.ExitCode != 0 {
		t.Fatalf("leader exit = %v, want the observed 0 preserved", record.ExitCode)
	}
	if !strings.Contains(strings.Join(record.Summary, "\n"), "could not be observed") {
		t.Fatalf("summary does not carry the uncertainty cause: %v", record.Summary)
	}
	if len(*sent) != 0 {
		t.Fatalf("signals sent for an uncertain group: %v", *sent)
	}
	if len(pids) == 1 && !pidAlive(pids[0]) {
		t.Fatal("the uncertain survivor was signalled and exited")
	}
}
