//go:build !linux && !darwin

package skillsync

import "os"

// openBoundedNoFollow has no portable no-follow, non-blocking open here. Read-only
// manifest reads stay usable: readBoundedRegularFile lstats before the open and
// compares the opened object with that lstat afterwards, so a swap is refused.
// Residual risk: a symbolic-link swap between the lstat and the open is detected
// only by that comparison, not prevented. Mutation paths fail closed on this
// platform (see mutationLockingSupported).
func openBoundedNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
