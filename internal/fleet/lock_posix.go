//go:build !windows

package fleet

import (
	"errors"
	"os"
	"syscall"
)

// tryLockFile takes a non-blocking exclusive lock on f. It reports false with
// no error when another open file description already holds the lock. The
// kernel releases the lock when the descriptor closes, so a crashed process
// can never leave a lock that blocks later work.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK), errors.Is(err, syscall.EAGAIN):
		return false, nil
	default:
		return false, err
	}
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
