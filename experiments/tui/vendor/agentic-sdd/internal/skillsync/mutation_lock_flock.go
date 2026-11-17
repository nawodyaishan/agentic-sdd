//go:build linux || darwin

package skillsync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// mutationLockingSupported reports whether this platform has the leaf checks and
// kernel-held flock this guard depends on.
const mutationLockingSupported = true

// mutationLeafFlags open the lock leaf without following a symlink, without
// blocking on a FIFO, without truncating, and without inheriting the descriptor.
const mutationLeafFlags = syscall.O_RDWR | syscall.O_CREAT | syscall.O_NOFOLLOW |
	syscall.O_NONBLOCK | syscall.O_CLOEXEC | syscall.O_NOCTTY

// openMutationLeaf opens a private regular lock file owned by the current user.
// It refuses, before and after open, a symlink, FIFO, directory, device, foreign
// file or any mode other than 0600. A refusal never modifies the referent.
func openMutationLeaf(path string) (*os.File, error) {
	if fi, err := os.Lstat(path); err == nil {
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: %s is not a regular file", ErrMutationUnsafe, path)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: inspect %s: %v", ErrMutationUnsafe, path, err)
	}
	f, err := os.OpenFile(path, mutationLeafFlags, 0600)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.EISDIR) {
			return nil, fmt.Errorf("%w: %s: %v", ErrMutationUnsafe, path, err)
		}
		return nil, fmt.Errorf("open target mutation lock %s: %w", path, err)
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("%w: stat opened lock %s: %v", ErrMutationUnsafe, path, err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm() != 0600 || !ownedByCurrentUser(fi) {
		_ = f.Close()
		return nil, fmt.Errorf("%w: %s must be a private regular file owned by the current user with mode 0600", ErrMutationUnsafe, path)
	}
	return f, nil
}

// tryLockMutation takes a non-blocking exclusive flock. It reports false with no
// error when another open file description holds it. A filesystem that cannot
// lock is a typed refusal, never a success.
func tryLockMutation(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK), errors.Is(err, syscall.EAGAIN):
		return false, nil
	case errors.Is(err, syscall.ENOTSUP), errors.Is(err, syscall.EOPNOTSUPP), errors.Is(err, syscall.ENOLCK):
		return false, fmt.Errorf("%w: flock refused by this filesystem: %v", ErrMutationUnsupported, err)
	default:
		return false, err
	}
}

func unlockMutation(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// sameMutationFile reports whether path, not followed, still names the object f
// holds. A missing or replaced path is a mismatch rather than a success.
func sameMutationFile(f *os.File, path string) (bool, error) {
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

func ownedByCurrentUser(fi os.FileInfo) bool {
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Getuid())
}
