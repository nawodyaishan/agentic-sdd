package fleet

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"agentic-sdd/internal/skillsync"
)

// ErrLegacyProvenance reports a backup or record whose last-applied desired
// state is unknown. Fleet never fabricates a managed/clean state for it.
var ErrLegacyProvenance = errors.New("legacy or unknown fleet provenance")

// RestoreResult describes the transaction a restore performed.
type RestoreResult struct {
	BackupID     string
	RestoredFrom string
	BackupPath   string
	LivePath     string
	Restored     int
	Removed      int
	NoOp         bool
	Rollback     skillsync.RollbackInfo
}

// BackupEntry describes an inspected backup under the backup root that has a
// valid fleet-manifest.json sidecar.
type BackupEntry struct {
	ID       string
	Path     string
	Manifest *Manifest
}

// ListFleetBackups scans the backup root for home and returns all backups
// that contain a format-4 fleet manifest, sorted newest first. Non-fleet backups
// or malformed manifests are safely skipped.
func ListFleetBackups(home string) ([]BackupEntry, error) {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	backupRoot := skillsync.BackupRootFor("", absHome)
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if err := skillsync.ValidateBackupID(id); err != nil {
			continue
		}
		backupDir := filepath.Join(backupRoot, id)
		sidecarPath := filepath.Join(backupDir, ManifestFileName)
		m, err := ReadManifestFile(sidecarPath)
		if err != nil || m == nil {
			continue // not a valid fleet backup
		}
		backups = append(backups, BackupEntry{
			ID:       id,
			Path:     backupDir,
			Manifest: m,
		})
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ID > backups[j].ID
	})
	return backups, nil
}

type restoreChange struct {
	file         string
	target       string
	data         []byte
	old          bool
	remove       bool
	backupDigest string
	record       FileRecord
}

// RestoreBackup validates the recorded transaction for backupID against the
// current installed hashes and reverses exactly the managed files that
// transaction touched. Unrelated files are never read or written, a modified
// target is refused rather than overwritten, and a legacy backup with no
// fleet manifest is refused instead of guessed. The restore itself is backed
// up through the same engine, so a failed rollback leaves recoverable state.
func RestoreBackup(home, backupID string, out io.Writer) (RestoreResult, error) {
	result := RestoreResult{RestoredFrom: backupID}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return result, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return result, err
	}
	homeInfo, err := os.Lstat(absHome)
	if err != nil {
		return result, fmt.Errorf("fleet restore: %w", err)
	}
	if !homeInfo.IsDir() {
		return result, fmt.Errorf("%w: home %s is not a directory", ErrUnsafeTarget, absHome)
	}
	backupRoot := skillsync.BackupRootFor("", absHome)
	lock, err := AcquireLock(backupRoot)
	if err != nil {
		return result, err
	}
	defer func() { _ = lock.Release() }()

	backupDir := filepath.Join(backupRoot, backupID)
	backupInfo, err := os.Lstat(backupDir)
	if err != nil {
		return result, fmt.Errorf("fleet restore: %w", err)
	}
	if !backupInfo.IsDir() {
		return result, fmt.Errorf("%w: backup %s is not a directory", ErrUnsafeTarget, backupDir)
	}
	manifestPath := filepath.Join(backupDir, ManifestFileName)
	manifest, err := ReadManifestFile(manifestPath)
	if err != nil {
		return result, err
	}
	if manifest == nil {
		return result, fmt.Errorf("%w: backup %s has no fleet manifest; its last-applied state is unknown and is not fabricated", ErrLegacyProvenance, backupID)
	}
	targetRoot := TargetRootForHome(absHome)
	if err := manifest.Validate(backupID, absHome, backupRoot, targetRoot); err != nil {
		return result, err
	}
	if manifest.State != ManifestStateApplied {
		return result, fmt.Errorf("%w: backup %s transaction state is %q, not applied; restore a completed transaction or recover the backup manually", ErrLegacyProvenance, backupID, manifest.State)
	}
	if err := checkTargetParents(targetRoot, absHome); err != nil {
		return result, err
	}

	live, err := ReadLiveManifest(absHome)
	if err != nil {
		return result, err
	}
	if live == nil {
		live, err = RebuildLiveManifest(absHome)
		if err != nil {
			return result, err
		}
	}
	if err := validateLiveManifest(live, absHome, targetRoot); err != nil {
		return result, err
	}

	records := append([]FileRecord(nil), manifest.Files...)
	sort.Slice(records, func(i, j int) bool { return records[i].Target < records[j].Target })

	var changes []restoreChange
	for _, r := range records {
		target, err := targetPathFor(targetRoot, filepath.Base(r.Target))
		if err != nil {
			return result, err
		}
		if filepath.Clean(r.Target) != filepath.Clean(target) {
			return result, fmt.Errorf("%w: backup target %q is not directly under %s", ErrUnsafeTarget, r.Target, targetRoot)
		}
		current, exists, err := onDiskDigest(target)
		if err != nil {
			return result, err
		}

		switch r.Provenance {
		case ProvenanceManaged, ProvenanceAdopted, ProvenanceRestored, ProvenanceUnknown:
			if r.InstalledDigest == "" {
				return result, fmt.Errorf("%w: backup %s has no recorded installed digest for %s", ErrLegacyProvenance, backupID, r.Target)
			}
			if !exists || current != r.InstalledDigest {
				return result, fmt.Errorf("%w: %s was modified after the recorded transaction; review it before restoring", ErrUnsafeTarget, target)
			}
		case ProvenanceRemoved:
			if exists {
				return result, fmt.Errorf("%w: %s was recreated after the recorded removal; review it before restoring", ErrUnsafeTarget, target)
			}
		default:
			return result, fmt.Errorf("%w: backup %s record %s has provenance %q", ErrLegacyProvenance, backupID, r.Target, r.Provenance)
		}

		file := filepath.Base(target)
		if r.BackupRef == "" {
			if r.Provenance == ProvenanceRemoved {
				return result, fmt.Errorf("%w: backup %s removal record %s has no saved content", ErrLegacyProvenance, backupID, target)
			}
			// The transaction created this file; reversing it removes the
			// file, with the current bytes saved in the restore backup.
			changes = append(changes, restoreChange{
				file:         file,
				target:       target,
				old:          true,
				remove:       true,
				backupDigest: current,
				record:       r,
			})
			continue
		}

		backupFile, data, err := readBackupRef(backupDir, r.BackupRef, file)
		if err != nil {
			return result, err
		}
		if r.BackupDigest == "" || digestBytes(data) != r.BackupDigest {
			return result, fmt.Errorf("%w: backup %s saved content for %s failed digest verification", ErrLegacyProvenance, backupID, backupFile)
		}
		changes = append(changes, restoreChange{
			file:         file,
			target:       target,
			data:         data,
			old:          exists,
			backupDigest: current,
			record:       r,
		})
	}

	if len(changes) == 0 {
		result.NoOp = true
		fmt.Fprintln(out, "No changes: the recorded transaction is already reversed.")
		return result, nil
	}

	now := time.Now().UTC()
	restoreManifest := &Manifest{
		Format:          ManifestFormat,
		Operation:       "restore",
		RestoredFrom:    backupID,
		State:           ManifestStateApplying,
		CreatedAt:       now.Format(time.RFC3339Nano),
		Home:            absHome,
		BackupRoot:      backupRoot,
		TargetRoot:      targetRoot,
		Scope:           ScopeGlobal,
		SourceRev:       manifest.SourceRev,
		ConfigDigest:    manifest.ConfigDigest,
		RendererVersion: manifest.RendererVersion,
		Files:           restoreRecords(changes, manifest),
	}
	hook := func(dir string) ([]byte, error) {
		applying := *restoreManifest
		applying.ID = filepath.Base(dir)
		applying.BackupPath = dir
		if err := verifyBackupCopies(dir, applying.Files); err != nil {
			return nil, err
		}
		return marshalManifest(&applying)
	}

	engineChanges := make([]skillsync.AgentFileChange, 0, len(changes))
	for _, c := range changes {
		engineChanges = append(engineChanges, skillsync.AgentFileChange{
			File:   c.file,
			Data:   c.data,
			Old:    c.old,
			Remove: c.remove,
		})
	}
	written, restoreErr := skillsync.ApplyAgentFiles(backupRoot, absHome, "fleet-restore", engineChanges, hook, out)
	result.BackupID = written.BackupID
	result.BackupPath = written.BackupDir
	result.Rollback = written.Rollback
	if restoreErr != nil {
		if written.BackupDir != "" {
			failed := *restoreManifest
			failed.ID = written.BackupID
			failed.BackupPath = written.BackupDir
			failed.State = ManifestStateFailed
			failed.Rollback = &RollbackState{
				Attempted: written.Rollback.Attempted,
				Complete:  written.Rollback.Complete,
				Error:     written.Rollback.Error,
			}
			_ = writeManifestFile(written.BackupDir, &failed)
			return result, fmt.Errorf("fleet restore: %w (recoverable backup: %s; rollback attempted=%t complete=%t)", restoreErr, written.BackupDir, written.Rollback.Attempted, written.Rollback.Complete)
		}
		return result, fmt.Errorf("fleet restore: %w", restoreErr)
	}

	applied := *restoreManifest
	applied.ID = written.BackupID
	applied.BackupPath = written.BackupDir
	applied.State = ManifestStateApplied
	for i := range applied.Files {
		if applied.Files[i].Provenance == ProvenanceRemoved {
			continue
		}
		applied.Files[i].InstalledDigest = applied.Files[i].DesiredDigest
	}
	if err := verifyInstalled(applied.Files); err != nil {
		failed := applied
		failed.State = ManifestStateFailed
		_ = writeManifestFile(written.BackupDir, &failed)
		return result, fmt.Errorf("%w (backup is recoverable at %s)", err, written.BackupDir)
	}
	if err := writeManifestFile(written.BackupDir, &applied); err != nil {
		return result, fmt.Errorf("fleet restore: write fleet manifest: %w", err)
	}
	merged := MergeLive(live, &applied)
	if err := WriteLiveManifest(absHome, merged); err != nil {
		return result, fmt.Errorf("fleet restore: write live manifest: %w", err)
	}
	result.LivePath = LivePointerPath(absHome)

	for _, c := range changes {
		if c.remove {
			result.Removed++
		} else {
			result.Restored++
		}
	}
	fmt.Fprintf(out, "Restored %d file(s) from backup %s; removed %d created file(s).\n", result.Restored, backupID, result.Removed)
	fmt.Fprintf(out, "Backup: %s\n", written.BackupDir)
	fmt.Fprintf(out, "Live fleet manifest: %s\n", result.LivePath)
	return result, nil
}

// readBackupRef reads one saved previous file and refuses any path other than
// opencode-agents/<bare-name> inside the backup directory.
func readBackupRef(backupDir, ref, file string) (string, []byte, error) {
	want := filepath.Join("opencode-agents", file)
	if filepath.Clean(ref) != filepath.Clean(want) {
		return "", nil, fmt.Errorf("%w: unexpected backup reference %q for %s", ErrUnsafeTarget, ref, file)
	}
	path := filepath.Join(backupDir, want)
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: refusing special file or symlink %s", ErrUnsafeTarget, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return path, data, nil
}

// restoreRecords describes the restore transaction in the same sidecar shape
// as an apply, so the restore can itself be validated or reversed later.
// Restored bytes that were fleet-owned before the reversed transaction keep a
// managed-style provenance so the next plan compares them against the current
// render instead of forcing an unrelated --adopt. Content that was never
// fleet-owned stays unknown: the restore does not fabricate ownership.
func restoreRecords(changes []restoreChange, source *Manifest) []FileRecord {
	records := make([]FileRecord, 0, len(changes))
	for _, c := range changes {
		rec := FileRecord{
			InstanceID:      c.record.InstanceID,
			SourceRef:       c.record.SourceRef,
			SourceDigest:    c.record.SourceDigest,
			ConfigDigest:    source.ConfigDigest,
			RendererVersion: source.RendererVersion,
			Scope:           ScopeGlobal,
			Target:          c.target,
			Provenance:      ProvenanceUnknown,
		}
		if c.remove {
			rec.Provenance = ProvenanceRemoved
		} else {
			rec.DesiredDigest = digestBytes(c.data)
			switch c.record.Provenance {
			case ProvenanceManaged, ProvenanceRestored, ProvenanceRemoved:
				rec.Provenance = ProvenanceRestored
			}
		}
		if c.old {
			rec.BackupRef = filepath.Join("opencode-agents", c.file)
			rec.BackupDigest = c.backupDigest
		}
		records = append(records, rec)
	}
	return records
}
