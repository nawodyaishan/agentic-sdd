//go:build linux || darwin

package skillsync

import (
	"os"
	"syscall"
)

// openBoundedNoFollow opens path read-only without following a symbolic link and
// without blocking on a FIFO. The caller validates the descriptor's type.
func openBoundedNoFollow(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
