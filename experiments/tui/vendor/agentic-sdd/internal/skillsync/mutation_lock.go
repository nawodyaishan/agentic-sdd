package skillsync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Typed refusals for the per-home target mutation guard. Match with errors.Is.
var (
	// ErrMutationConflict means another live holder owns the same home and domain,
	// or the lock path was replaced while it was being acquired.
	ErrMutationConflict = errors.New("target mutation is held by another process")
	// ErrMutationUnsafe means the home, state directory or lock leaf is not a
	// private, owned, regular object inside the home. Nothing was modified.
	ErrMutationUnsafe = errors.New("target mutation lock path is unsafe")
	// ErrMutationUnsupported means this platform, or the requested domain, cannot
	// provide exclusion. Mutation fails closed; it never silently proceeds.
	ErrMutationUnsupported = errors.New("target mutation locking is unsupported")
	// ErrMutationNotHeld means the guard is nil, zero, closed, or not the handle
	// for the home and domain being checked.
	ErrMutationNotHeld = errors.New("target mutation guard is not held")
)

// MutationDomain names one closed mutation domain. Only the constants below are
// accepted; a lock file name is never supplied by a caller.
type MutationDomain string

const (
	// DomainAgentTarget covers global agent profiles. Installer writes, fleet
	// apply/restore and provenance finalization acquire this one domain.
	DomainAgentTarget MutationDomain = "agent-target"
	// DomainMCPTarget covers MCP configuration writes. It is deliberately a
	// separate domain so MCP writes do not contend with agent profiles.
	DomainMCPTarget MutationDomain = "mcp-target"
)

const (
	mutationStateDirName = ".agentic-sdd"
	mutationLockDirName  = "locks"
	// mutationAcquireAttempts bounds how often a leaf replaced after locking is
	// re-opened before the acquisition reports a conflict.
	mutationAcquireAttempts = 2
)

// mutationLeafHook is a test seam. It runs after a leaf is locked and before its
// identity is compared with the path. It is nil in production.
var mutationLeafHook func(path string)

// leafName maps a closed domain to its fixed lock file name. The name is new,
// so no legacy lock file is ever reused.
func (d MutationDomain) leafName() (string, error) {
	switch d {
	case DomainAgentTarget:
		return "agent-target.lock", nil
	case DomainMCPTarget:
		return "mcp-target.lock", nil
	}
	return "", fmt.Errorf("%w: unknown mutation domain %q", ErrMutationUnsupported, string(d))
}

// MutationGuard is an opaque, held exclusion for one canonical home and one
// closed domain. Its fields are private, so a guard can only come from
// AcquireTargetMutation. Close releases it. A guard is not safe for concurrent
// use, and it must be released before its holder exits.
//
// Lock order for composed writers is fixed: agent target, then MCP, then any
// legacy lock. Nested work passes the checked guard and never reacquires.
type MutationGuard struct {
	home   string
	domain MutationDomain
	path   string
	file   *os.File
}

// AcquireTargetMutation takes the non-blocking exclusive guard for one home and
// domain. home must be a canonical absolute path. The lock is kernel-held, so a
// dead holder releases it and no elapsed time can evict a live one. Acquisition
// never waits: a held lock returns ErrMutationConflict immediately.
func AcquireTargetMutation(home string, domain MutationDomain) (*MutationGuard, error) {
	name, err := domain.leafName()
	if err != nil {
		return nil, err
	}
	if !mutationLockingSupported {
		return nil, fmt.Errorf("%w: agent and MCP writes refuse on this platform", ErrMutationUnsupported)
	}
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, fmt.Errorf("%w: home must be a canonical absolute path", ErrMutationUnsafe)
	}
	for attempt := 0; attempt < mutationAcquireAttempts; attempt++ {
		lockDir, err := ensureMutationLockDir(home)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(lockDir, name)
		f, err := openMutationLeaf(path)
		if err != nil {
			return nil, err
		}
		held, err := tryLockMutation(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if !held {
			_ = f.Close()
			return nil, fmt.Errorf("%w: %s lock held at %s", ErrMutationConflict, domain, path)
		}
		if mutationLeafHook != nil {
			mutationLeafHook(path)
		}
		same, err := sameMutationFile(f, path)
		if err != nil {
			_ = unlockMutation(f)
			_ = f.Close()
			return nil, err
		}
		if same {
			return &MutationGuard{home: home, domain: domain, path: path, file: f}, nil
		}
		// The path now names another object, so the held inode is not the one
		// other writers will open. Release it and try the current leaf once more.
		_ = unlockMutation(f)
		_ = f.Close()
	}
	return nil, fmt.Errorf("%w: %s lock path changed while acquiring", ErrMutationConflict, domain)
}

// Require reports whether g still holds the lock for home and domain. It checks
// the handle, the binding and the live path identity, so a path replaced after
// locking is detected on every nested entry.
func (g *MutationGuard) Require(home string, domain MutationDomain) error {
	if g == nil || g.file == nil {
		return fmt.Errorf("%w", ErrMutationNotHeld)
	}
	if g.home != home || g.domain != domain {
		return fmt.Errorf("%w: guard is for %s in another home or domain", ErrMutationUnsafe, g.domain)
	}
	same, err := sameMutationFile(g.file, g.path)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("%w: %s no longer names the held lock", ErrMutationConflict, g.path)
	}
	return nil
}

// Close releases the lock. It unlocks and closes only the descriptor it holds;
// it never unlinks the file, truncates it or touches a successor at the path.
func (g *MutationGuard) Close() error {
	if g == nil || g.file == nil {
		return nil
	}
	unlockErr := unlockMutation(g.file)
	closeErr := g.file.Close()
	g.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// ensureMutationLockDir validates or creates <home>/.agentic-sdd/locks, lstat-ing
// each component inside the home. A symlink, a non-directory or a component owned
// by another user is refused.
func ensureMutationLockDir(home string) (string, error) {
	if err := checkMutationDir(home, false); err != nil {
		return "", err
	}
	state := filepath.Join(home, mutationStateDirName)
	if err := checkMutationDir(state, true); err != nil {
		return "", err
	}
	lockDir := filepath.Join(state, mutationLockDirName)
	if err := checkMutationDir(lockDir, true); err != nil {
		return "", err
	}
	return lockDir, nil
}

func checkMutationDir(dir string, create bool) error {
	fi, err := os.Lstat(dir)
	if err != nil && create && errors.Is(err, fs.ErrNotExist) {
		if mkErr := os.Mkdir(dir, 0700); mkErr != nil && !errors.Is(mkErr, fs.ErrExist) {
			return fmt.Errorf("%w: create %s: %v", ErrMutationUnsafe, dir, mkErr)
		}
		fi, err = os.Lstat(dir)
	}
	if err != nil {
		return fmt.Errorf("%w: inspect %s: %v", ErrMutationUnsafe, dir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%w: %s is not a real directory", ErrMutationUnsafe, dir)
	}
	if !ownedByCurrentUser(fi) {
		return fmt.Errorf("%w: %s is not owned by the current user", ErrMutationUnsafe, dir)
	}
	return nil
}
