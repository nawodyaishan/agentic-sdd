package skillsync

import (
	"path/filepath"
	"strings"
)

// OperationKind identifies which engine operation produced an
// OperationResult. It lets a caller route a result without inferring the
// operation from display text.
type OperationKind string

const (
	// OperationApply is a preview or apply of the portable source.
	OperationApply OperationKind = "apply"
	// OperationRestore is a preview or apply of one recorded backup.
	OperationRestore OperationKind = "restore"
)

// Outcome is the definite state one engine operation reached. The zero value
// is not a success: callers must compare against the explicit constants and
// treat a returned error as a failed operation.
type Outcome string

const (
	// OutcomePreview means the read-only plan was produced; nothing was
	// written and no backup was created.
	OutcomePreview Outcome = "preview"
	// OutcomeNoChange means an apply or restore ran but every planned
	// destination was already up to date, so no backup was created.
	OutcomeNoChange Outcome = "no-change"
	// OutcomeApplied means the planned changes were written and a new,
	// complete backup was recorded.
	OutcomeApplied Outcome = "applied"
	// OutcomeFailed means the operation did not complete. When the failure
	// happened after a complete backup was written, BackupID and BackupPath
	// identify it and Rollback reports what was attempted; otherwise both
	// are empty and no destination was intentionally changed.
	OutcomeFailed Outcome = "failed"
)

// ChangeAction is the honest action planned or performed for one destination.
// The tokens match the engine's transcript verbs, so a structured result can
// be reconciled with SummarizePlan without parsing prose.
type ChangeAction string

const (
	// ChangeInstall creates a destination that did not exist before.
	ChangeInstall ChangeAction = "install"
	// ChangeReplace overwrites an existing destination after saving its
	// previous content to the operation's backup.
	ChangeReplace ChangeAction = "replace"
	// ChangeRestore puts a missing destination back from a backup.
	ChangeRestore ChangeAction = "restore"
	// ChangeRemove deletes a destination that the backup says did not exist
	// before the operation, after saving its current content.
	ChangeRemove ChangeAction = "remove"
	// ChangeUpToDate means the destination already matches and is left
	// untouched.
	ChangeUpToDate ChangeAction = "up-to-date"
)

// ResultChange describes one managed destination in a plan or result. ID,
// Kind and Client use the same values as InventoryEntry: skill IDs are
// directory names, profile IDs omit ".md", and Kind is InventoryKindSkill or
// InventoryKindProfile.
type ResultChange struct {
	Client string
	ID     string
	Kind   string
	Path   string
	Action ChangeAction

	// Backup and SHA256 describe the saved previous content. They are set
	// only on applied replace/remove changes: Backup is relative to the
	// operation's backup directory and SHA256 is the digest of that saved
	// content. Both are empty when nothing needed saving.
	Backup string
	SHA256 string
}

// RollbackInfo reports what the engine did after a write failure partway
// through committing changes. It is the zero value when rollback was not
// required or not attempted.
type RollbackInfo struct {
	// Attempted reports whether the engine tried to undo already committed
	// changes.
	Attempted bool
	// Complete reports whether every attempted undo succeeded. False with
	// Attempted true means the destinations may be in a partial state.
	Complete bool
	// Error carries the rollback failure detail, if any. It never includes
	// destination content.
	Error string
}

// OperationResult is the structured outcome of SyncWithOptionsResult or
// RestoreResult. It is a concrete value with no interface hierarchy: callers
// can inspect the outcome, the affected destinations and backup identity
// directly instead of parsing the operation transcript.
//
// Planned holds the complete plan, including up-to-date entries, in the same
// order as the transcript. Applied holds the changes that were committed and
// is populated only for OutcomeApplied; a failed result never claims applied
// changes. BackupID and BackupPath identify the backup this operation wrote
// (for a restore, the pre-restore backup, not the backup being restored).
// Use them and Rollback to describe a failure: Attempted with Complete false
// means destinations may be in a partial state.
//
// SourceDigest is a stable identity of exactly the source payload the plan
// used: for apply, the selected portable skill trees plus any OpenCode pack
// skill and profile sources; for restore, the backup ID, manifest bytes and
// the payload digests the plan reads. It never includes destination content,
// so a preview and the apply it authorizes for the same inputs agree. It is
// empty when the operation failed before a plan was produced.
type OperationResult struct {
	Operation    OperationKind
	Outcome      Outcome
	BackupID     string
	BackupPath   string
	Planned      []ResultChange
	Applied      []ResultChange
	Rollback     RollbackInfo
	SourceDigest string
}

// plannedChange builds the structured form of one transcript item. isFile
// selects the profile identity, whose ID omits the ".md" suffix.
func plannedChange(t target, id string, isFile bool, action ChangeAction) ResultChange {
	resultID := id
	kind := InventoryKindSkill
	if isFile {
		resultID = strings.TrimSuffix(id, ".md")
		kind = InventoryKindProfile
	}
	return ResultChange{
		Client: t.name,
		ID:     resultID,
		Kind:   kind,
		Path:   filepath.Join(t.path, id),
		Action: action,
	}
}

// changeAction returns the planned action recorded on an internal change,
// falling back to the flags the direct installer callers already set.
func changeAction(c change) ChangeAction {
	if c.action != "" {
		return c.action
	}
	switch {
	case c.remove:
		return ChangeRemove
	case c.old:
		return ChangeReplace
	default:
		return ChangeInstall
	}
}

// appliedChange turns one committed internal change and its manifest entry
// into a structured applied change.
func appliedChange(c change, e entry) ResultChange {
	applied := plannedChange(c.target, c.skill, c.isFile, changeAction(c))
	applied.Backup = e.Backup
	applied.SHA256 = e.SHA256
	return applied
}
