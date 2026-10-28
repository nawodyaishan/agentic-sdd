//go:build linux

package harnesscheck

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCancellationStopsOwnedGroupNotUnrelatedProcess(t *testing.T) {
	unrelated := exec.Command("sleep", "30")
	if err := unrelated.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = unrelated.Process.Kill(); _ = unrelated.Wait() }()
	c := fixtureContract()
	c.Checks[0].Steps = []Step{{Argv: []string{"/bin/sh", "-c", "trap '' TERM; sleep 30 & echo $! > child.pid; wait"}, Env: map[string]string{}}}
	o := optionsFor(t, c)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results := make(chan Result, 1)
	errors := make(chan error, 1)
	go func() { r, err := Run(ctx, o); results <- r; errors <- err }()
	var pid int
	for until := time.Now().Add(3 * time.Second); time.Now().Before(until); {
		data, err := os.ReadFile(filepath.Join(o.Workspace, "child.pid"))
		if err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
			if pid > 0 {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("fixture did not create its ordinary child")
	}
	cancel()
	select {
	case result := <-results:
		if err := <-errors; err != nil || result.Execution != "cancelled" {
			t.Fatalf("%+v %v", result, err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("owned group did not stop promptly")
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err == nil {
		// A killed orphan can remain a zombie until PID 1 reaps it; that is
		// retained membership, not running activity. The runner reports uncertainty.
		parts := strings.Fields(string(data)[strings.LastIndex(string(data), ")")+1:])
		if len(parts) == 0 || parts[0] != "Z" {
			t.Fatalf("child still active: %d", pid)
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := syscall.Kill(unrelated.Process.Pid, 0); err != nil {
		t.Fatalf("unrelated process affected: %v", err)
	}
}

func TestLeaderCannotPassWithLiveDescendant(t *testing.T) {
	c := fixtureContract()
	c.Checks[0].Steps = []Step{{Argv: []string{"/bin/sh", "-c", "sleep 30 &"}, Env: map[string]string{}}}
	o := optionsFor(t, c)
	r, err := Run(context.Background(), o)
	if err != nil || r.Execution != "incomplete" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestTerminationWaitIsBoundedAndReportsUnconfirmed(t *testing.T) {
	done := make(chan error)
	start := time.Now()
	if _, confirmed := waitProcess(done, 20*time.Millisecond); confirmed {
		t.Fatal("invented process completion")
	}
	if time.Since(start) > time.Second {
		t.Fatal("termination wait was not bounded")
	}
}
