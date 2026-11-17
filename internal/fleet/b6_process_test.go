//go:build !windows

package fleet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// B6 (T13) two-process exclusion. A child process holds one kernel-held writer
// domain so the parent proves the other writer is refused across processes.

const (
	b6HoldEnv = "AGENTIC_SDD_B6_HOLD"
	// b6HoldAgent is the agent-target guard of a canonical home (what installer
	// and fleet writers share). b6HoldLegacy is the legacy fleet backup-root lock.
	b6HoldAgent  = "agent"
	b6HoldLegacy = "legacy"
)

// TestB6HoldHelperProcess is not a test: it is the child that holds one writer
// domain until its stdin closes. It runs only when the environment names one.
func TestB6HoldHelperProcess(t *testing.T) {
	spec := os.Getenv(b6HoldEnv)
	if spec == "" {
		return
	}
	kind, path, ok := strings.Cut(spec, "|")
	if !ok {
		fmt.Fprintln(os.Stdout, "B6-ERROR bad hold spec")
		os.Exit(3)
	}
	var release func() error
	switch kind {
	case b6HoldAgent:
		guard, err := skillsync.AcquireTargetMutation(path, skillsync.DomainAgentTarget)
		if err != nil {
			fmt.Fprintf(os.Stdout, "B6-ERROR %v\n", err)
			os.Exit(3)
		}
		release = guard.Close
	case b6HoldLegacy:
		lock, err := AcquireLock(path)
		if err != nil {
			fmt.Fprintf(os.Stdout, "B6-ERROR %v\n", err)
			os.Exit(3)
		}
		release = lock.Release
	default:
		fmt.Fprintln(os.Stdout, "B6-ERROR unknown kind")
		os.Exit(3)
	}
	fmt.Fprintln(os.Stdout, "B6-HELD")
	_, _ = io.Copy(io.Discard, os.Stdin)
	_ = release()
	os.Exit(0)
}

type b6Holder struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func b6StartHolder(t *testing.T, kind, path string) *b6Holder {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestB6HoldHelperProcess$", "-test.count=1")
	cmd.Env = append(os.Environ(), b6HoldEnv+"="+kind+"|"+path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "B6-HELD" {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatalf("holder %s did not acquire: line=%q err=%v", kind, line, err)
	}
	return &b6Holder{cmd: cmd, stdin: stdin}
}

func (h *b6Holder) stop(t *testing.T) {
	t.Helper()
	_ = h.stdin.Close()
	if err := h.cmd.Wait(); err != nil {
		t.Fatalf("holder exited with error: %v", err)
	}
}

func TestB6InstallerProcessBlocksFleetApply(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	holder := b6StartHolder(t, b6HoldAgent, b6CanonicalHome(t, home))
	defer holder.stop(t)

	if _, err := ApplyPlan(plan, home, io.Discard); !errors.Is(err, skillsync.ErrMutationConflict) {
		t.Fatalf("ApplyPlan while another process holds the agent guard = %v, want ErrMutationConflict", err)
	}
	if n := len(backupDirs(t, home)); n != 0 {
		t.Fatalf("refused apply created %d backup(s)", n)
	}
	if _, err := os.Stat(filepath.Join(TargetRootForHome(home), b6Target)); !os.IsNotExist(err) {
		t.Fatalf("refused apply wrote the target: %v", err)
	}
}

func TestB6InstallerProcessBlocksFleetRestore(t *testing.T) {
	home := newFleetHome(t)
	b6Create(t, home)
	applyPlan(t, b6Update(t, home), home)
	backup := filepath.Base(latestBackup(t, home))
	target := filepath.Join(TargetRootForHome(home), b6Target)
	before := readString(t, target)

	holder := b6StartHolder(t, b6HoldAgent, b6CanonicalHome(t, home))
	defer holder.stop(t)
	if _, err := RestoreBackup(home, backup, io.Discard); !errors.Is(err, skillsync.ErrMutationConflict) {
		t.Fatalf("RestoreBackup while another process holds the agent guard = %v, want ErrMutationConflict", err)
	}
	if got := readString(t, target); got != before {
		t.Fatalf("refused restore modified the target: %q", got)
	}
}

func TestB6FleetProcessBlocksInstallerApply(t *testing.T) {
	home := newFleetHome(t)
	holder := b6StartHolder(t, b6HoldAgent, b6CanonicalHome(t, home))
	defer holder.stop(t)

	changes := []skillsync.AgentFileChange{{File: b6Target, Data: []byte("installer\n")}}
	_, err := skillsync.ApplyAgentFiles(skillsync.BackupRootFor("", home), home, "installer", changes, nil, io.Discard)
	if !errors.Is(err, skillsync.ErrMutationConflict) {
		t.Fatalf("installer apply while fleet holds the guard = %v, want ErrMutationConflict", err)
	}
	if _, err := os.Stat(filepath.Join(TargetRootForHome(home), b6Target)); !os.IsNotExist(err) {
		t.Fatalf("refused installer wrote the target: %v", err)
	}
}

func TestB6OldFleetLegacyLockBlocksNewApply(t *testing.T) {
	home := newFleetHome(t)
	plan := buildPlanFor(t, b6ConfigFor(t, b6Model1), home, false)
	holder := b6StartHolder(t, b6HoldLegacy, skillsync.BackupRootFor("", home))
	defer holder.stop(t)

	if _, err := ApplyPlan(plan, home, io.Discard); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("ApplyPlan while another fleet process holds the legacy lock = %v, want ErrLockHeld", err)
	}
}
