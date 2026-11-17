//go:build linux || darwin

package skillsync

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// B1 (T01) coverage for the per-home target mutation guard. Kernel-held
// exclusion is proven across processes with helper children; the path-leaf
// cases prove that a symlink, FIFO, directory or replaced leaf never lets the
// guard modify a referent, block, or clear a successor.

const b1MutationHoldEnv = "AGENTIC_SDD_B1_MUTATION_HOLD"

// TestB1MutationHelperProcess is the child that holds one guard until its stdin
// closes (or it is killed). It runs only when the environment names a home.
func TestB1MutationHelperProcess(t *testing.T) {
	spec := os.Getenv(b1MutationHoldEnv)
	if spec == "" {
		return
	}
	home, domain, _ := strings.Cut(spec, "|")
	g, err := AcquireTargetMutation(home, MutationDomain(domain))
	if err != nil {
		fmt.Fprintf(os.Stdout, "B1-ERROR %v\n", err)
		os.Exit(3)
	}
	fmt.Fprintln(os.Stdout, "B1-HELD")
	_, _ = io.Copy(io.Discard, os.Stdin)
	_ = g.Close()
	os.Exit(0)
}

type b1MutationHolder struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func startB1MutationHolder(t *testing.T, home string, domain MutationDomain) *b1MutationHolder {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestB1MutationHelperProcess$", "-test.count=1")
	cmd.Env = append(os.Environ(), b1MutationHoldEnv+"="+home+"|"+string(domain))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			line := sc.Text()
			if line == "B1-HELD" {
				ready <- nil
				return
			}
			if strings.HasPrefix(line, "B1-ERROR") {
				ready <- errors.New(line)
				return
			}
		}
		ready <- errors.New("helper exited before acquiring the guard")
	}()
	select {
	case err := <-ready:
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("helper did not hold the guard: %v", err)
		}
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("helper did not acquire the guard within 60s")
	}
	return &b1MutationHolder{cmd: cmd, stdin: stdin}
}

func (h *b1MutationHolder) stop(t *testing.T) {
	t.Helper()
	_ = h.stdin.Close()
	if err := h.cmd.Wait(); err != nil {
		t.Fatalf("helper exit: %v", err)
	}
}

// b1CanonicalHome returns a resolved home so the test names the canonical key.
func b1CanonicalHome(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return home
}

// b1LockLeaf is the layout the guard must use: a private per-home state
// directory with one fixed file per closed domain.
func b1LockLeaf(home string, domain MutationDomain) string {
	name := map[MutationDomain]string{
		DomainAgentTarget: "agent-target.lock",
		DomainMCPTarget:   "mcp-target.lock",
	}[domain]
	return filepath.Join(home, ".agentic-sdd", "locks", name)
}

func b1MkLockDir(t *testing.T, home string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".agentic-sdd", "locks"), 0700); err != nil {
		t.Fatal(err)
	}
}

func b1Sentinel(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "victim-sentinel")
	content := []byte("B1 SENTINEL: must never change\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path, content
}

func b1AssertSentinel(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sentinel unreadable: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("sentinel referent was modified: %q", got)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Fatalf("sentinel mode changed to %v", fi.Mode().Perm())
	}
}

func b1SetMutationHook(t *testing.T, hook func(path string)) {
	t.Helper()
	mutationLeafHook = hook
	t.Cleanup(func() { mutationLeafHook = nil })
}

func b1MustAcquire(t *testing.T, home string, domain MutationDomain) *MutationGuard {
	t.Helper()
	g, err := AcquireTargetMutation(home, domain)
	if err != nil {
		t.Fatalf("acquire %s in %s: %v", domain, home, err)
	}
	return g
}

func TestB1MutationGuardConflictsWithinOneHomeAndDomain(t *testing.T) {
	home := b1CanonicalHome(t)
	g := b1MustAcquire(t, home, DomainAgentTarget)
	defer g.Close()

	second, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = second.Close()
		t.Fatal("second acquisition of the same home and domain succeeded")
	}
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("expected ErrMutationConflict, got %v", err)
	}

	if err := g.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	again := b1MustAcquire(t, home, DomainAgentTarget)
	_ = again.Close()
}

func TestB1MutationGuardDomainsAreIndependentWithinOneHome(t *testing.T) {
	home := b1CanonicalHome(t)
	agent := b1MustAcquire(t, home, DomainAgentTarget)
	defer agent.Close()
	mcp := b1MustAcquire(t, home, DomainMCPTarget)
	defer mcp.Close()
}

func TestB1MutationGuardHomesAreIndependentUnderOneBackupRoot(t *testing.T) {
	shared := b1CanonicalHome(t)
	homeA := filepath.Join(shared, "home-a")
	homeB := filepath.Join(shared, "home-b")
	for _, h := range []string{homeA, homeB} {
		if err := os.Mkdir(h, 0700); err != nil {
			t.Fatal(err)
		}
	}
	a := b1MustAcquire(t, homeA, DomainAgentTarget)
	defer a.Close()
	b := b1MustAcquire(t, homeB, DomainAgentTarget)
	defer b.Close()
}

func TestB1MutationGuardRejectsUnknownDomainAndRelativeHome(t *testing.T) {
	home := b1CanonicalHome(t)
	if _, err := AcquireTargetMutation(home, MutationDomain("../escape")); !errors.Is(err, ErrMutationUnsupported) {
		t.Fatalf("unknown domain: expected ErrMutationUnsupported, got %v", err)
	}
	if _, err := AcquireTargetMutation("relative/home", DomainAgentTarget); !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("relative home: expected ErrMutationUnsafe, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agentic-sdd")); !os.IsNotExist(err) {
		t.Fatalf("a refused domain created state: %v", err)
	}
}

func TestB1MutationGuardRequireBindsHomeDomainAndHeldState(t *testing.T) {
	home := b1CanonicalHome(t)
	other := b1CanonicalHome(t)
	g := b1MustAcquire(t, home, DomainAgentTarget)
	if err := g.Require(home, DomainAgentTarget); err != nil {
		t.Fatalf("Require on the held guard: %v", err)
	}
	if err := g.Require(other, DomainAgentTarget); !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("Require for another home: expected ErrMutationUnsafe, got %v", err)
	}
	if err := g.Require(home, DomainMCPTarget); !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("Require for another domain: expected ErrMutationUnsafe, got %v", err)
	}

	var zero MutationGuard
	if err := zero.Require(home, DomainAgentTarget); !errors.Is(err, ErrMutationNotHeld) {
		t.Fatalf("zero guard: expected ErrMutationNotHeld, got %v", err)
	}
	var nilGuard *MutationGuard
	if err := nilGuard.Require(home, DomainAgentTarget); !errors.Is(err, ErrMutationNotHeld) {
		t.Fatalf("nil guard: expected ErrMutationNotHeld, got %v", err)
	}

	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if err := g.Require(home, DomainAgentTarget); !errors.Is(err, ErrMutationNotHeld) {
		t.Fatalf("closed guard: expected ErrMutationNotHeld, got %v", err)
	}
}

func TestB1MutationGuardSymlinkLeafLeavesSentinelUntouched(t *testing.T) {
	home := b1CanonicalHome(t)
	b1MkLockDir(t, home)
	victim, want := b1Sentinel(t)
	if err := os.Symlink(victim, b1LockLeaf(home, DomainAgentTarget)); err != nil {
		t.Fatal(err)
	}

	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = g.Close()
		t.Fatal("guard followed a symlink leaf")
	}
	if !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("expected ErrMutationUnsafe, got %v", err)
	}
	b1AssertSentinel(t, victim, want)
}

func TestB1MutationGuardSymlinkStateDirectoryIsRefused(t *testing.T) {
	home := b1CanonicalHome(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, ".agentic-sdd")); err != nil {
		t.Fatal(err)
	}

	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = g.Close()
		t.Fatal("guard created state through a symlinked state directory")
	}
	if !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("expected ErrMutationUnsafe, got %v", err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("symlink target was written: %d entries", len(entries))
	}
}

func TestB1MutationGuardFIFOLeafDoesNotBlock(t *testing.T) {
	home := b1CanonicalHome(t)
	b1MkLockDir(t, home)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	if err := syscall.Mkfifo(leaf, 0600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	type result struct {
		g   *MutationGuard
		err error
	}
	done := make(chan result, 1)
	go func() {
		g, err := AcquireTargetMutation(home, DomainAgentTarget)
		done <- result{g, err}
	}()
	var res result
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("AcquireTargetMutation blocked on a FIFO leaf")
	}
	if res.g != nil {
		_ = res.g.Close()
		t.Fatal("guard accepted a FIFO leaf")
	}
	if !errors.Is(res.err, ErrMutationUnsafe) {
		t.Fatalf("expected ErrMutationUnsafe, got %v", res.err)
	}
	if fi, err := os.Lstat(leaf); err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("FIFO leaf was replaced: %v %v", fi, err)
	}
}

func TestB1MutationGuardDirectoryLeafIsRefused(t *testing.T) {
	home := b1CanonicalHome(t)
	b1MkLockDir(t, home)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	if err := os.Mkdir(leaf, 0700); err != nil {
		t.Fatal(err)
	}
	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = g.Close()
		t.Fatal("guard accepted a directory leaf")
	}
	if !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("expected ErrMutationUnsafe, got %v", err)
	}
	if fi, err := os.Lstat(leaf); err != nil || !fi.IsDir() {
		t.Fatalf("directory leaf was altered: %v %v", fi, err)
	}
}

func TestB1MutationGuardReplacedLeafRetriesThenConflicts(t *testing.T) {
	home := b1CanonicalHome(t)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	calls := 0
	b1SetMutationHook(t, func(path string) {
		calls++
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("replacement\n"), 0600); err != nil {
			t.Fatal(err)
		}
	})

	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = g.Close()
		t.Fatal("guard accepted a leaf replaced after locking")
	}
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("expected ErrMutationConflict after one retry, got %v", err)
	}
	if calls != mutationAcquireAttempts {
		t.Fatalf("expected %d attempts, hook ran %d times", mutationAcquireAttempts, calls)
	}
	if got, _ := os.ReadFile(leaf); string(got) != "replacement\n" {
		t.Fatalf("replacement was modified: %q", got)
	}
}

func TestB1MutationGuardReplacedOnceRecoversOnRetry(t *testing.T) {
	home := b1CanonicalHome(t)
	calls := 0
	b1SetMutationHook(t, func(path string) {
		calls++
		if calls > 1 {
			return
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("transient\n"), 0600); err != nil {
			t.Fatal(err)
		}
	})

	g := b1MustAcquire(t, home, DomainAgentTarget)
	if err := g.Require(home, DomainAgentTarget); err != nil {
		t.Fatalf("Require after retry: %v", err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected one retry, hook ran %d times", calls)
	}
}

func TestB1MutationGuardReplacedBySymlinkIsUnsafe(t *testing.T) {
	home := b1CanonicalHome(t)
	victim, want := b1Sentinel(t)
	b1SetMutationHook(t, func(path string) {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(victim, path); err != nil {
			t.Fatal(err)
		}
	})

	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = g.Close()
		t.Fatal("guard accepted a leaf swapped for a symlink")
	}
	if !errors.Is(err, ErrMutationUnsafe) {
		t.Fatalf("expected ErrMutationUnsafe, got %v", err)
	}
	b1AssertSentinel(t, victim, want)
}

func TestB1MutationGuardCloseNeverTouchesSuccessor(t *testing.T) {
	home := b1CanonicalHome(t)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	g := b1MustAcquire(t, home, DomainAgentTarget)

	successor := "successor-owner\n"
	if err := os.Remove(leaf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leaf, []byte(successor), 0600); err != nil {
		t.Fatal(err)
	}
	if err := g.Require(home, DomainAgentTarget); !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("Require after the path was replaced: expected ErrMutationConflict, got %v", err)
	}
	if err := g.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got, _ := os.ReadFile(leaf); string(got) != successor {
		t.Fatalf("closing an old guard changed a successor: %q", got)
	}
}

func TestB1MutationGuardLeafIsPrivateRegularAndNeverUnlinked(t *testing.T) {
	home := b1CanonicalHome(t)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	g := b1MustAcquire(t, home, DomainAgentTarget)
	fi, err := os.Lstat(leaf)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm() != 0600 {
		t.Fatalf("lock leaf must be a private regular file, got %v", fi.Mode())
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(leaf); err != nil {
		t.Fatalf("Close unlinked the guard file: %v", err)
	}
}

func TestB1MutationGuardHelperProcessExclusion(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainAgentTarget)

	if g, err := AcquireTargetMutation(home, DomainAgentTarget); err == nil {
		_ = g.Close()
		holder.stop(t)
		t.Fatal("a second process acquired a guard held by another process")
	} else if !errors.Is(err, ErrMutationConflict) {
		holder.stop(t)
		t.Fatalf("expected ErrMutationConflict, got %v", err)
	}

	// The same home's other domain is independent of the held one.
	mcp := b1MustAcquire(t, home, DomainMCPTarget)
	_ = mcp.Close()

	holder.stop(t)
	g := b1MustAcquire(t, home, DomainAgentTarget)
	_ = g.Close()
}

func TestB1MutationGuardDeadHolderIsRecovered(t *testing.T) {
	home := b1CanonicalHome(t)
	holder := startB1MutationHolder(t, home, DomainAgentTarget)
	if err := holder.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = holder.cmd.Wait()

	g, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err != nil {
		t.Fatalf("killed holder kept the guard: %v", err)
	}
	_ = g.Close()
}

func TestB1MutationGuardLiveHolderIsNotEvictedByAge(t *testing.T) {
	home := b1CanonicalHome(t)
	leaf := b1LockLeaf(home, DomainAgentTarget)
	g := b1MustAcquire(t, home, DomainAgentTarget)
	defer g.Close()

	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(leaf, old, old); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireTargetMutation(home, DomainAgentTarget)
	if err == nil {
		_ = second.Close()
		t.Fatal("an aged but live lock was evicted")
	}
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("expected ErrMutationConflict, got %v", err)
	}
	if err := g.Require(home, DomainAgentTarget); err != nil {
		t.Fatalf("held guard lost ownership by age: %v", err)
	}
}
