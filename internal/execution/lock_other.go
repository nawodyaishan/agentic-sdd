//go:build !unix

package execution

import (
	"fmt"
	"os"
)

// pidAlive cannot verify process liveness on this platform. It conservatively
// reports the owner as alive, so a legacy lock is never reclaimed here.
func pidAlive(pid int) bool {
	return pid > 0
}

// lockRunGuard fails closed on this platform: there is no kernel guard, so a run
// mutation must not pretend to be exclusive.
func lockRunGuard(path string) (*os.File, error) {
	return nil, fmt.Errorf("%w: run locking is unavailable on this platform; run mutation fails closed", ErrUnsupported)
}
