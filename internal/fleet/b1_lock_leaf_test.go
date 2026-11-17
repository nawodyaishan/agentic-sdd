//go:build !windows

package fleet

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

// These tests cover the B1 (T02) hardening of the legacy fleet lock leaf
// `${backupRoot}/.fleet.lock`: a symlink, FIFO, directory or replaced leaf must
// never let acquisition or release modify a referent, block, or clear another
// owner's lock. Helper processes prove kernel-held exclusion across processes.

const b1FleetHoldEnv = "AGENTIC_SDD_B1_FLEET_HOLD_ROOT"

// TestB1FleetLockHelperProcess is not a test: it is the child that holds the
// fleet lock until its stdin closes (or it is killed). It runs only when the
// environment variable names a backup root.
func TestB1FleetLockHelperProcess(t *testing.T) {
	root := os.Getenv(b1FleetHoldEnv)
	if root == "" {
		return
	}
	lock, err := AcquireLock(root)
	if err != nil {
		fmt.Fprintf(os.Stdout, "B1-ERROR %v\n", err)
		os.Exit(3)
	}
	fmt.Fprintln(os.Stdout, "B1-HELD")
	_, _ = io.Copy(io.Discard, os.Stdin)
	_ = lock.Release()
	os.Exit(0)
}

type b1FleetHolder struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func startB1FleetHolder(t *testing.T, root string) *b1FleetHolder {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestB1FleetLockHelperProcess$", "-test.count=1")
	cmd.Env = append(os.Environ(), b1FleetHoldEnv+"="+root)
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
		ready <- errors.New("helper exited before acquiring the lock")
	}()
	select {
	case err := <-ready:
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("helper did not hold the lock: %v", err)
		}
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("helper did not acquire the lock within 60s")
	}
	return &b1FleetHolder{cmd: cmd, stdin: stdin}
}

func (h *b1FleetHolder) stop(t *testing.T) {
	t.Helper()
	_ = h.stdin.Close()
	if err := h.cmd.Wait(); err != nil {
		t.Fatalf("helper exit: %v", err)
	}
}

func b1WriteSentinel(t *testing.T, dir string) (string, []byte) {
	t.Helper()
	path := filepath.Join(dir, "victim-sentinel")
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

func TestB1FleetLockSymlinkLeafIsRefusedAndSentinelUntouched(t *testing.T) {
	root := t.TempDir()
	victim, want := b1WriteSentinel(t, t.TempDir())
	if err := os.Symlink(victim, filepath.Join(root, LockFileName)); err != nil {
		t.Fatal(err)
	}

	lock, err := AcquireLock(root)
	if err == nil {
		_ = lock.Release()
		t.Fatal("AcquireLock followed a symlink leaf and succeeded")
	}
	if !errors.Is(err, ErrLockUnsafe) {
		t.Fatalf("expected ErrLockUnsafe, got %v", err)
	}
	b1AssertSentinel(t, victim, want)
}

func TestB1FleetLockFIFOLeafIsRefusedWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	if err := syscall.Mkfifo(leaf, 0600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	type result struct {
		lock *Lock
		err  error
	}
	done := make(chan result, 1)
	go func() {
		l, err := AcquireLock(root)
		done <- result{l, err}
	}()
	var res result
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("AcquireLock blocked on a FIFO leaf")
	}
	if res.lock != nil {
		_ = res.lock.Release()
		t.Fatal("AcquireLock accepted a FIFO leaf")
	}
	if !errors.Is(res.err, ErrLockUnsafe) {
		t.Fatalf("expected ErrLockUnsafe, got %v", res.err)
	}
	fi, err := os.Lstat(leaf)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("FIFO leaf was replaced: mode %v", fi.Mode())
	}
}

func TestB1FleetLockDirectoryLeafIsRefused(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	if err := os.Mkdir(leaf, 0700); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireLock(root)
	if err == nil {
		_ = lock.Release()
		t.Fatal("AcquireLock accepted a directory leaf")
	}
	if !errors.Is(err, ErrLockUnsafe) {
		t.Fatalf("expected ErrLockUnsafe, got %v", err)
	}
	if fi, err := os.Lstat(leaf); err != nil || !fi.IsDir() {
		t.Fatalf("directory leaf was altered: %v %v", fi, err)
	}
}

func TestB1FleetLockRequiresPrivateMode(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	if err := os.WriteFile(leaf, []byte("shared\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(leaf, 0644); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireLock(root)
	if err == nil {
		_ = lock.Release()
		t.Fatal("AcquireLock accepted a lock leaf that is not mode 0600")
	}
	if !errors.Is(err, ErrLockUnsafe) {
		t.Fatalf("expected ErrLockUnsafe, got %v", err)
	}
	if fi, err := os.Stat(leaf); err != nil || fi.Mode().Perm() != 0644 {
		t.Fatalf("refused leaf was modified: %v %v", fi, err)
	}
}

func TestB1FleetLockReleaseNeverTouchesSwappedReferent(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	victim, want := b1WriteSentinel(t, t.TempDir())
	if err := os.Remove(leaf); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, leaf); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	b1AssertSentinel(t, victim, want)
}

func TestB1FleetLockOldHandleDoesNotClearSuccessor(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	successor := "successor-owner\n"
	if err := os.Remove(leaf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leaf, []byte(successor), 0600); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got := readString(t, leaf); got != successor {
		t.Fatalf("releasing an old handle cleared a successor: %q", got)
	}
}

func TestB1FleetLockHeldByAnotherProcessIsExclusive(t *testing.T) {
	root := t.TempDir()
	holder := startB1FleetHolder(t, root)

	if lock, err := AcquireLock(root); err == nil {
		_ = lock.Release()
		holder.stop(t)
		t.Fatal("second process acquired a lock held by another process")
	} else if !errors.Is(err, ErrLockHeld) {
		holder.stop(t)
		t.Fatalf("expected ErrLockHeld, got %v", err)
	}

	holder.stop(t)
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatalf("lock not available after holder exited: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

// b1SetLockLeafHook installs a race injector for the duration of one test.
func b1SetLockLeafHook(t *testing.T, hook func(path string)) {
	t.Helper()
	lockLeafHook = hook
	t.Cleanup(func() { lockLeafHook = nil })
}

func TestB1FleetLockReplacedLeafRetriesThenConflicts(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	calls := 0
	b1SetLockLeafHook(t, func(path string) {
		calls++
		// Another actor replaces the leaf after it is locked, every time.
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("replacement\n"), 0600); err != nil {
			t.Fatal(err)
		}
	})

	lock, err := AcquireLock(root)
	if err == nil {
		_ = lock.Release()
		t.Fatal("AcquireLock accepted a leaf replaced after locking")
	}
	if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("expected a conflict after one retry, got %v", err)
	}
	if calls != lockAcquireAttempts {
		t.Fatalf("expected %d attempts, hook ran %d times", lockAcquireAttempts, calls)
	}
	if got := readString(t, leaf); got != "replacement\n" {
		t.Fatalf("replacement was modified by the refused acquisition: %q", got)
	}
}

func TestB1FleetLockReplacedOnceRecoversOnRetry(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	calls := 0
	b1SetLockLeafHook(t, func(path string) {
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

	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatalf("expected the retry to acquire the current leaf: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected a retry, hook ran %d times", calls)
	}
	if fi, err := os.Lstat(leaf); err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("lock leaf not a regular file after release: %v %v", fi, err)
	}
}

func TestB1FleetLockReplacedBySymlinkAfterLockIsUnsafe(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, LockFileName)
	victim, want := b1WriteSentinel(t, t.TempDir())
	b1SetLockLeafHook(t, func(path string) {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(victim, path); err != nil {
			t.Fatal(err)
		}
	})

	lock, err := AcquireLock(root)
	if err == nil {
		_ = lock.Release()
		t.Fatal("AcquireLock accepted a leaf swapped for a symlink")
	}
	if !errors.Is(err, ErrLockUnsafe) {
		t.Fatalf("expected ErrLockUnsafe, got %v", err)
	}
	b1AssertSentinel(t, victim, want)
	if fi, err := os.Lstat(leaf); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("swapped leaf was altered: %v %v", fi, err)
	}
}

func TestB1FleetLockDeadHolderIsRecovered(t *testing.T) {
	root := t.TempDir()
	holder := startB1FleetHolder(t, root)
	if err := holder.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = holder.cmd.Wait()

	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatalf("killed holder kept the lock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}
