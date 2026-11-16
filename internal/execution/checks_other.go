//go:build !unix

package execution

import (
	"os"
	"os/exec"
)

// configureCheckProcess falls back to cancelling the single process on hosts
// without Unix process groups. Descendants are not covered.
func configureCheckProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return cmd.Process.Kill()
	}
}

// checkProcessSignal is Unix-only; other hosts report no signal observation.
func checkProcessSignal(*os.ProcessState) string {
	return ""
}

// interruptCheckProcessGroup cannot address a process group on this platform.
// It forwards the signal to the single process; the bounded grace period still
// escalates through the configured cancellation.
func interruptCheckProcessGroup(cmd *exec.Cmd, sig os.Signal) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(sig)
}
