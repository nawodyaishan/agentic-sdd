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
