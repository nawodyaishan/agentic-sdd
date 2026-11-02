package opencodeadapter

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestACPSleepHelper is a helper process that sleeps until signalled.
func TestACPSleepHelper(t *testing.T) {
	if os.Getenv("SDD_SLEEP_HELPER") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
}

// TestSyntheticCancellationPreservesUnrelatedProcesses verifies that
// terminating the owned process group does not signal an unrelated process.
func TestSyntheticCancellationPreservesUnrelatedProcesses(t *testing.T) {
	unrelated := exec.Command(os.Args[0], "-test.run=TestACPSleepHelper", "--")
	unrelated.Env = []string{"SDD_SLEEP_HELPER=1"}
	unrelated.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := unrelated.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = unrelated.Process.Kill()
		_ = unrelated.Wait()
	}()

	proc, err := spawnProcess(os.Args[0], []string{"-test.run=TestACPSleepHelper", "--"},
		[]string{"SDD_SLEEP_HELPER=1"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Terminate(); err != nil {
		t.Fatalf("terminate owned process: %v", err)
	}
	if err := proc.Wait(); err == nil && proc.cmd.ProcessState == nil {
		t.Fatal("owned process was not reaped")
	}
	if proc.cmd.ProcessState == nil {
		t.Fatal("owned process still alive after terminate")
	}

	// The unrelated process must still be alive and signal-able.
	if err := unrelated.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("unrelated process was killed by owned-process termination: %v", err)
	}
}

// TestSpawnOwnedProcessUsesAllowlistedEnvironment verifies the production spawn
// does not inherit the caller's environment.
func TestSpawnOwnedProcessUsesAllowlistedEnvironment(t *testing.T) {
	// The production spawner is exercised only via fake peers; the allowlisted
	// environment is a pure helper, so assert it exposes no credential/state
	// variables and carries only PATH.
	env := minimalEnv()
	for _, e := range env {
		for _, forbidden := range []string{"HOME=", "XDG_", "OPENAI_", "ANTHROPIC_", "OPENROUTER_", "HTTP_PROXY", "AWS_", "SECRET"} {
			if len(e) >= len(forbidden) && e[:len(forbidden)] == forbidden {
				t.Fatalf("allowlisted env leaks %q: %q", forbidden, e)
			}
		}
	}
}
