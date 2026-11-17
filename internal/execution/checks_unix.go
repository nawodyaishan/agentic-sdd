//go:build unix

package execution

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// configureCheckProcess starts the check in its own process group. Ordinary
// descendants are covered; a deliberately detached process is not, and this is
// not a sandbox. The command is not bound to a context: every group signal is
// sent through the owned-group gate in checks_group.go.
func configureCheckProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// sendCheckGroupSignal signals the owned group only. The group id must be
// positive: kill(0, ...) and kill(-0, ...) would address the caller's own group.
// A group that is already gone reports nil.
func sendCheckGroupSignal(pgid int, sig os.Signal) error {
	if pgid <= 0 {
		return errors.New("owned process group id is invalid")
	}
	s, ok := sig.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported group signal %v", sig)
	}
	err := syscall.Kill(-pgid, s)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// probeCheckGroup reports whether any process still carries the group id. Signal
// zero delivers nothing. A permission error means the group cannot be decided.
func probeCheckGroup(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, errors.New("owned process group id is invalid")
	}
	err := syscall.Kill(-pgid, 0)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}

// signalCheckTree delivers a pre-exit trigger (timeout, output limit or
// interrupt) to the owned group. The leader is unreaped here, so the group id is
// pinned. If the group signal fails, the unreaped leader is killed directly.
func signalCheckTree(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	err := checkGroupSignal(cmd.Process.Pid, sig)
	if err != nil {
		signalLeaderFallback(cmd)
	}
	return err
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
