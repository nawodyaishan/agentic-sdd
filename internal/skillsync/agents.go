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
// not decide fleet policy: callers must validate file names, provenance and
// current hashes first.
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

	written, err := installChangesWithHook(backupRoot, home, sourceLabel, nil, names, engineChanges, operation{kind: "fleet-apply"}, engineHook, out)
	result.BackupID = written.backupID
	result.BackupDir = written.backupPath
	result.Applied = written.applied
	result.Rollback = written.rollback
	return result, err
}
