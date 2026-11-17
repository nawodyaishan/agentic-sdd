//go:build !windows

package fleet

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// fleetLockingSupported reports whether fleet mutation paths may take the lock.
const fleetLockingSupported = true

// lockLeafFlags opens the legacy lock leaf without following a symlink, without
// blocking on a FIFO, without truncating, and without leaking the descriptor
// into children. O_CREAT is the only creation path; nothing is ever unlinked.
const lockLeafFlags = syscall.O_RDWR | syscall.O_CREAT | syscall.O_NOFOLLOW |
	syscall.O_NONBLOCK | syscall.O_CLOEXEC | syscall.O_NOCTTY

// openLockLeaf opens path as a private regular file owned by the current user.
// It refuses, before and after open, anything that is not exactly that: a
// symlink, FIFO, directory, device, foreign-owned file or a mode other than 0600.
// Refusals wrap ErrLockUnsafe and never modify the referent.
func openLockLeaf(path string) (*os.File, error) {
	if fi, err := os.Lstat(path); err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: %s is not a regular file", ErrLockUnsafe, path)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: inspect %s: %v", ErrLockUnsafe, path, err)
	}
	f, err := os.OpenFile(path, lockLeafFlags, 0600)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.EISDIR) {
			return nil, fmt.Errorf("%w: %s: %v", ErrLockUnsafe, path, err)
		}
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	if err := checkOpenedLockLeaf(f, path); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// checkOpenedLockLeaf validates the object actually opened, not an earlier path lookup.
func checkOpenedLockLeaf(f *os.File, path string) error {
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("%w: stat opened lock file %s: %v", ErrLockUnsafe, path, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", ErrLockUnsafe, path)
	}
	if fi.Mode().Perm() != 0600 {
		return fmt.Errorf("%w: %s must be mode 0600, found %v", ErrLockUnsafe, path, fi.Mode().Perm())
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("%w: %s is not owned by the current user", ErrLockUnsafe, path)
	}
	return nil
}

// tryLockFile takes a non-blocking exclusive lock on f. It reports false with
// no error when another open file description already holds the lock. The
// kernel releases the lock when the descriptor closes, so a crashed process
// can never leave a lock that blocks later work. A filesystem that cannot
// lock is a typed refusal, never a success.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK), errors.Is(err, syscall.EAGAIN):
		return false, nil
	case errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP), errors.Is(err, syscall.ENOLCK):
		return false, fmt.Errorf("%w: flock: %v", ErrLockUnsupported, err)
	default:
		return false, err
	}
}

func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// lockLeafMatches reports whether path, not followed, still names the object f
// holds. A missing or replaced path is a mismatch, never an error to ignore.
func lockLeafMatches(f *os.File, path string) (bool, error) {
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	li, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	a, okA := fi.Sys().(*syscall.Stat_t)
	b, okB := li.Sys().(*syscall.Stat_t)
	return okA && okB && uint64(a.Dev) == uint64(b.Dev) && uint64(a.Ino) == uint64(b.Ino), nil
}
