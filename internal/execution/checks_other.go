//go:build !unix

package execution

import (
	"errors"
	"os"
	"os/exec"
)

// errGroupUnsupported reports that owned process groups do not exist on this
// host. Completion cannot be proven, so the owned-group gate fails closed.
var errGroupUnsupported = errors.New("owned process groups are not available on this platform")

// configureCheckProcess has no process-group setup on this platform.
func configureCheckProcess(cmd *exec.Cmd) {}

// sendCheckGroupSignal cannot address a group on this platform.
func sendCheckGroupSignal(int, os.Signal) error {
	return errGroupUnsupported
}

// probeCheckGroup cannot observe a group on this platform.
func probeCheckGroup(int) (bool, error) {
	return false, errGroupUnsupported
}

// signalCheckTree forwards a pre-exit trigger to the single process. Descendants
// are not covered on this platform.
func signalCheckTree(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	if sig == os.Kill {
		return cmd.Process.Kill()
	}
	return cmd.Process.Signal(sig)
}

// checkProcessSignal is Unix-only; other hosts report no signal observation.
func checkProcessSignal(*os.ProcessState) string {
	return ""
}
