//go:build unix

package execution

import (
	"errors"
	"syscall"
)

// pidAlive reports whether pid is an existing process this user can signal. A
// permission denial still means the process exists, so it is treated as alive.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return !errors.Is(err, syscall.ESRCH)
}
