//go:build unix

package execution

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureCheckProcess starts the check in its own process group and cancels
// the whole group. Ordinary descendants are covered; a deliberately detached
// process is not, and this is not a sandbox.
func configureCheckProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killCheckProcessGroup(cmd) }
}

// killCheckProcessGroup signals only the check's own process group. An already
// finished process reports os.ErrProcessDone so the wait is not treated as a
// cancellation failure.
func killCheckProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return os.ErrProcessDone
	}
	err := syscall.Kill(-pid, syscall.SIGKILL)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, syscall.ESRCH):
		return os.ErrProcessDone
	default:
		return err
	}
}

// checkProcessSignal reports the signal that terminated the process, if any. A
// signalled process has no exit code, only this observation.
func checkProcessSignal(state *os.ProcessState) string {
	if state == nil {
		return ""
	}
	status, ok := state.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return ""
	}
	return status.Signal().String()
}
