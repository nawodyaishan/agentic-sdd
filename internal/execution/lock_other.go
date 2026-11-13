//go:build !unix

package execution

// pidAlive cannot verify process liveness on this platform, so it conservatively
// reports the owner as alive and relies on the lock age for reclaim.
func pidAlive(pid int) bool {
	return pid > 0
}
