//go:build !linux && !darwin

package skillsync

import (
	"fmt"
	"os"
)

// mutationLockingSupported is false off Linux and Darwin. Every changed mutation
// path refuses here. Read-only paths never call the guard, so they stay usable.
const mutationLockingSupported = false

func unsupportedMutation() error {
	return fmt.Errorf("%w: no reviewed leaf check on this platform", ErrMutationUnsupported)
}

func openMutationLeaf(string) (*os.File, error) {
	return nil, unsupportedMutation()
}

func tryLockMutation(*os.File) (bool, error) {
	return false, unsupportedMutation()
}

func unlockMutation(*os.File) error {
	return nil
}

func sameMutationFile(*os.File, string) (bool, error) {
	return false, unsupportedMutation()
}

func ownedByCurrentUser(os.FileInfo) bool {
	return false
}
