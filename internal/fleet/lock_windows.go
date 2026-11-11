//go:build windows

package fleet

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	// errorLockViolation is ERROR_LOCK_VIOLATION, returned when another
	// handle holds the requested region.
	errorLockViolation syscall.Errno = 33
)

var procLockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

// tryLockFile takes a non-blocking exclusive lock on f and reports false with
// no error when another handle already holds it. The lock is released when the
// handle closes, so an abandoned lock cannot outlive its process.
func tryLockFile(f *os.File) (bool, error) {
	var overlapped syscall.Overlapped
	ok, _, err := procLockFileEx.Call(
		f.Fd(),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ok != 0 {
		return true, nil
	}
	if errno, isErrno := err.(syscall.Errno); isErrno && errno == errorLockViolation {
		return false, nil
	}
	return false, err
}

// unlockFile is a no-op: closing the handle releases a LockFileEx lock.
func unlockFile(*os.File) error {
	return nil
}
