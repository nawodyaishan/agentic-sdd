//go:build linux || darwin

package harnesscheck

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Darwin's implementation is cross-compiled, but execution stays fail-closed
// until native cancellation/descendant tests have been observed there.
const processSupported = runtime.GOOS == "linux"

// A new process group contains this command and ordinary descendants only.
// Deliberately detached/escaped processes require OS isolation; this is not a
// sandbox or a universal process-tree tracker.
func process(ctx context.Context, argv, env []string, dir string, log io.Writer) (string, *int, string, bool) {
	if err := ctx.Err(); err != nil {
		return contextStatus(ctx), nil, err.Error(), false
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, env, log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 250 * time.Millisecond
	if err := cmd.Start(); err != nil {
		return "blocked", nil, "cannot start executable: " + err.Error(), false
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	joined := false
	status, reason := "passed", "process exited zero"
	select {
	case err = <-done:
		joined = true
		if ctx.Err() != nil {
			status, reason = contextStatus(ctx), ctx.Err().Error()
		}
	case <-ctx.Done():
		status, reason = contextStatus(ctx), ctx.Err().Error()
		_ = syscall.Kill(-pid, syscall.SIGTERM)
		err, joined = waitProcess(done, 150*time.Millisecond)
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		if !joined {
			err, joined = waitProcess(done, time.Second)
		}
		if !joined {
			return status, nil, reason + "; termination unconfirmed after bounded KILL wait; remaining activity/effects uncertain", true
		}
	}
	// A leader may exit without its children. Never accept that as a pass.
	if syscall.Kill(-pid, 0) != syscall.ESRCH {
		_ = syscall.Kill(-pid, syscall.SIGTERM)
		time.Sleep(50 * time.Millisecond)
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		if status == "passed" {
			status = "incomplete"
			reason = "leader exited with remaining process-group activity; termination requested"
		}
		if syscall.Kill(-pid, 0) != syscall.ESRCH {
			reason += "; remaining group membership/side effects uncertain (may include zombies)"
		}
	}
	if cmd.ProcessState == nil {
		return "incomplete", nil, "process wait returned no exit state; effects uncertain", true
	}
	code := cmd.ProcessState.ExitCode()
	if status == "passed" {
		if errors.Is(err, exec.ErrWaitDelay) {
			status, reason = "incomplete", "output pipes outlived process; descendant effects uncertain"
		} else if err != nil {
			status, reason = "failed", "process failed: "+err.Error()
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				status = "incomplete"
			}
		}
	}
	return status, &code, reason, true
}

func waitProcess(done <-chan error, limit time.Duration) (error, bool) {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case err := <-done:
		return err, true
	case <-timer.C:
		return nil, false
	}
}

// Executable lookup uses the explicitly constructed environment, not a PATH
// override silently different from the one exec.Command used during creation.
func executable(name, dir string, env []string) (string, error) {
	if strings.ContainsRune(name, '/') {
		if name[0] != '/' {
			name = dir + "/" + name
		}
		return executableFile(name)
	}
	for _, entry := range env {
		if len(entry) >= 5 && entry[:5] == "PATH=" {
			for _, base := range filepath.SplitList(entry[5:]) {
				if base == "" || base[0] != '/' {
					continue
				} // Refuse implicit current-directory PATH lookup.
				if p, err := executableFile(base + "/" + name); err == nil {
					return p, nil
				}
			}
		}
	}
	return "", exec.ErrNotFound
}

func executableFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return "", os.ErrPermission
	}
	return path, nil
}
