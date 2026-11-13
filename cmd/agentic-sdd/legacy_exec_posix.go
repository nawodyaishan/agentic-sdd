//go:build !windows

package main

import (
	"os"
	"syscall"
)

// execLegacyTarget replaces the compatibility process with the canonical
// sibling. On POSIX there is no return on success: the target inherits the
// streams and terminal, receives signals directly and supplies the exit code.
func execLegacyTarget(path string, argv []string) error {
	return syscall.Exec(path, argv, os.Environ())
}
