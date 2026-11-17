package skillsync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// AgentFileChange is one managed OpenCode agent-profile write. File must be a
// bare file name (no separators); Data is the exact content to stage when
// Remove is false. Old reports whether the destination exists before the
// transaction, so the engine backs it up before replacing or removing it.
type AgentFileChange struct {
	File   string
	Data   []byte
	Old    bool
	Remove bool
	// ExpectSHA256, when non-empty, is the raw SHA-256 the destination must have
	// under the guard. A mismatch fails with ErrBeforeStateChanged and nothing is
	// written. Old is checked against the destination but is never backup authority.
	ExpectSHA256 string
}

// AgentFileResult identifies the backup a fleet transaction wrote and what the
// engine's rollback attempt did when the transaction failed.
type AgentFileResult struct {
	BackupID  string
	BackupDir string
	Applied   []ResultChange
	Rollback  RollbackInfo
}

// ApplyAgentFiles writes managed OpenCode agent profiles through the same
// backup, staging, rename and rollback engine Sync and Restore use. It does
// not decide fleet policy: callers must validate file names and provenance
// first. The engine holds the agent-target guard for the whole transaction,
// re-reads each destination under it and refuses a mismatch with Old or
// ExpectSHA256 (ErrBeforeStateChanged) before any backup or write.
//
// afterBackup, when non-nil, runs once the backup directory is complete and
// restorable but before any destination is staged or changed. Its bytes are
// written to <backupDir>/fleet-manifest.json. An error there aborts the
// transaction with no destination modified.
func ApplyAgentFiles(backupRoot, home, sourceLabel string, changes []AgentFileChange, afterBackup func(backupDir string) ([]byte, error), out io.Writer) (AgentFileResult, error) {
	var result AgentFileResult
	if len(changes) == 0 {
		return result, nil
	}
	home, err := filepath.Abs(home)
	if err != nil {
		return result, err
	}
	key, err := agentTargetGuardKey(home)
	if err != nil {
		return result, err
	}
	guard, err := AcquireTargetMutation(key, DomainAgentTarget)
	if err != nil {
		return result, err
	}
	defer func() { _ = guard.Close() }()
	return ApplyAgentFilesGuarded(guard, backupRoot, home, sourceLabel, changes, afterBackup, out)
}

// ApplyAgentFilesGuarded is ApplyAgentFiles for a caller that already holds the
// agent-target guard for home, such as the fleet writer that takes it before
// its own plan compare-and-swap. The guard is checked against home and the
// agent-target domain, then used; it is never reacquired.
func ApplyAgentFilesGuarded(guard *MutationGuard, backupRoot, home, sourceLabel string, changes []AgentFileChange, afterBackup func(backupDir string) ([]byte, error), out io.Writer) (AgentFileResult, error) {
	var result AgentFileResult
	if len(changes) == 0 {
		return result, nil
	}
	home, err := filepath.Abs(home)
	if err != nil {
		return result, err
	}
	key, err := agentTargetGuardKey(home)
	if err != nil {
		return result, err
	}
	if err := guard.Require(key, DomainAgentTarget); err != nil {
		return result, err
	}
	agentTarget := openCodeAgentsTarget(home)
	if err := checkParents(agentTarget.path, home); err != nil {
		return result, err
	}
	if info, err := os.Lstat(agentTarget.path); err == nil {
		if !info.IsDir() {
			return result, fmt.Errorf("refusing non-directory agent target %s", agentTarget.path)
		}
	} else if !os.IsNotExist(err) {
		return result, err
	}

	seen := make(map[string]bool, len(changes))
	engineChanges := make([]change, 0, len(changes))
	var names []string
	for _, c := range changes {
		if c.File == "" || c.File != filepath.Base(c.File) || c.File == "." || c.File == ".." {
			return result, fmt.Errorf("invalid agent file name %q", c.File)
		}
		if seen[c.File] {
			return result, fmt.Errorf("duplicate agent file %q in one transaction", c.File)
		}
		seen[c.File] = true
		if c.Remove && c.Data != nil {
			return result, fmt.Errorf("agent file %q has both data and remove", c.File)
		}
		if !c.Remove && c.Data == nil {
			return result, fmt.Errorf("agent file %q has no staged data", c.File)
		}
		engineChanges = append(engineChanges, change{
			target: agentTarget,
			skill:  c.File,
			isFile: true,
			old:    c.Old,
			remove: c.Remove,
			data:   c.Data,
			expect: c.ExpectSHA256,
		})
		names = append(names, c.File)
	}
	sort.Strings(names)

	hook := afterBackup
	var engineHook afterBackupHook
	if hook != nil {
		engineHook = func(backupDir string) error {
			data, err := hook(backupDir)
			if err != nil {
				return err
			}
			if data == nil {
				return nil
			}
			return os.WriteFile(filepath.Join(backupDir, "fleet-manifest.json"), data, 0600)
		}
	}

	written, err := installChangesWithHook(guard, backupRoot, home, sourceLabel, nil, names, engineChanges, operation{kind: "fleet-apply"}, engineHook, out)
	result.BackupID = written.backupID
	result.BackupDir = written.backupPath
	result.Applied = written.applied
	result.Rollback = written.rollback
	return result, err
}
