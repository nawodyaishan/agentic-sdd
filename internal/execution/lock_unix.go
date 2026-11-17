//go:build unix

package execution

import (
	"errors"
	"fmt"
	"os"
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

// lockRunGuard takes the exclusive, nonblocking kernel lock on one private guard
// file. The file is opened without following links and without blocking on a
// FIFO, must be a regular file owned by this user with mode 0600, and must still
// be the path's object after the lock is taken; a replaced leaf is retried once.
// The descriptor is close-on-exec, so no child inherits it.
func lockRunGuard(path string) (*os.File, error) {
	for attempt := 0; attempt < 2; attempt++ {
		fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0o600)
		if err != nil {
			if errors.Is(err, syscall.ELOOP) {
				return nil, fmt.Errorf("%w: run guard is a symlink", ErrUnsafePath)
			}
			return nil, fmt.Errorf("open run guard: %w", err)
		}
		file := os.NewFile(uintptr(fd), path)
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("stat run guard: %w", err)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ok || int(stat.Uid) != os.Geteuid() {
			file.Close()
			return nil, fmt.Errorf("%w: run guard is not a private regular file owned by this user", ErrUnsafePath)
		}
		if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			file.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
				return nil, fmt.Errorf("%w: run is locked by another writer", ErrConflict)
			}
			return nil, fmt.Errorf("lock run guard: %w", err)
		}
		current, err := os.Lstat(path)
		if err == nil && os.SameFile(info, current) {
			return file, nil
		}
		// The leaf was replaced or removed while this descriptor waited: the
		// locked object is not the run's guard, so retry on the current one.
		file.Close()
	}
	return nil, fmt.Errorf("%w: run guard was replaced during acquisition", ErrConflict)
}
