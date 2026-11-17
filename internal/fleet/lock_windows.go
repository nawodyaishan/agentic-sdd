//go:build windows

package fleet

import (
	"fmt"
	"os"
)

// fleetLockingSupported is false: fleet mutation paths fail closed on Windows.
// No reviewed equivalent of the POSIX leaf checks exists here, so the previous
// LockFileEx path is not used. Read-only commands never take the lock.
const fleetLockingSupported = false

func openLockLeaf(string) (*os.File, error) {
	return nil, fmt.Errorf("%w: fleet apply and restore refuse to write on Windows", ErrLockUnsupported)
}

func tryLockFile(*os.File) (bool, error) {
	return false, ErrLockUnsupported
}

func unlockFile(*os.File) error {
	return nil
}

func lockLeafMatches(*os.File, string) (bool, error) {
	return false, ErrLockUnsupported
}
