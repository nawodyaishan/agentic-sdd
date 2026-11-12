package fleet

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"agentic-sdd/internal/skillsync"
)

// RestorePreview is the read-only projection of reversing one managed backup.
// It never carries an approval: RestoreBackup remains the only writer, and it
// revalidates the same state before touching a target.
type RestorePreview struct {
	BackupID   string
	BackupPath string
	Files      []FilePreview
	Counts     map[string]int
}

// PreviewRestoreBackup reports exactly what RestoreBackup would do for
// backupID against the current target bytes, without acquiring the engine
// lock, creating a backup, writing a file or touching the live manifest.
//
// It mirrors RestoreBackup's validation order deliberately: malformed IDs,
// home and backup-directory checks, a missing or corrupt fleet manifest, a
// non-applied transaction, manifest identity/target checks, the symlink path
// check, the live-pointer check and every per-record installed-digest,
// provenance, recreated-removal, backup-reference and saved-content digest
// check. A refusal is therefore the same refusal, with the same sentinel
// errors, as the writer.
func PreviewRestoreBackup(home, backupID string) (*RestorePreview, error) {
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return nil, err
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	homeInfo, err := os.Lstat(absHome)
	if err != nil {
		return nil, fmt.Errorf("fleet restore preview: %w", err)
	}
	if !homeInfo.IsDir() {
		return nil, fmt.Errorf("%w: home %s is not a directory", ErrUnsafeTarget, absHome)
	}
	backupRoot := skillsync.BackupRootFor("", absHome)
	backupDir := filepath.Join(backupRoot, backupID)
	backupInfo, err := os.Lstat(backupDir)
	if err != nil {
		return nil, fmt.Errorf("fleet restore preview: %w", err)
	}
	if !backupInfo.IsDir() {
		return nil, fmt.Errorf("%w: backup %s is not a directory", ErrUnsafeTarget, backupDir)
	}
	manifest, err := ReadManifestFile(filepath.Join(backupDir, ManifestFileName))
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("%w: backup %s has no fleet manifest; its last-applied state is unknown and is not fabricated", ErrLegacyProvenance, backupID)
	}
	targetRoot := TargetRootForHome(absHome)
	if err := manifest.Validate(backupID, absHome, backupRoot, targetRoot); err != nil {
		return nil, err
	}
	if manifest.State != ManifestStateApplied {
		return nil, fmt.Errorf("%w: backup %s transaction state is %q, not applied; restore a completed transaction or recover the backup manually", ErrLegacyProvenance, backupID, manifest.State)
	}
	if err := checkTargetParents(targetRoot, absHome); err != nil {
		return nil, err
	}
	live, err := ReadLiveManifest(absHome)
	if err != nil {
		return nil, err
	}
	if live == nil {
		live, err = RebuildLiveManifest(absHome)
		if err != nil {
			return nil, err
		}
	}
	if err := validateLiveManifest(live, absHome, targetRoot); err != nil {
		return nil, err
	}

	records := append([]FileRecord(nil), manifest.Files...)
	sort.Slice(records, func(i, j int) bool { return records[i].Target < records[j].Target })

	preview := &RestorePreview{
		BackupID:   backupID,
		BackupPath: backupDir,
		Files:      make([]FilePreview, 0, len(records)),
		Counts: map[string]int{
			"total":      0,
			FileOpCreate: 0,
			FileOpUpdate: 0,
			FileOpRemove: 0,
			FileOpNoOp:   0,
		},
	}
	for _, r := range records {
		target, err := targetPathFor(targetRoot, filepath.Base(r.Target))
		if err != nil {
			return nil, err
		}
		if filepath.Clean(r.Target) != filepath.Clean(target) {
			return nil, fmt.Errorf("%w: backup target %q is not directly under %s", ErrUnsafeTarget, r.Target, targetRoot)
		}
		current, exists, err := onDiskDigest(target)
		if err != nil {
			return nil, err
		}
		switch r.Provenance {
		case ProvenanceManaged, ProvenanceAdopted, ProvenanceRestored, ProvenanceUnknown:
			if r.InstalledDigest == "" {
				return nil, fmt.Errorf("%w: backup %s has no recorded installed digest for %s", ErrLegacyProvenance, backupID, r.Target)
			}
			if !exists || current != r.InstalledDigest {
				return nil, fmt.Errorf("%w: %s was modified after the recorded transaction; review it before restoring", ErrUnsafeTarget, target)
			}
		case ProvenanceRemoved:
			if exists {
				return nil, fmt.Errorf("%w: %s was recreated after the recorded removal; review it before restoring", ErrUnsafeTarget, target)
			}
		default:
			return nil, fmt.Errorf("%w: backup %s record %s has provenance %q", ErrLegacyProvenance, backupID, r.Target, r.Provenance)
		}

		var oldBytes []byte
		if exists {
			oldBytes, err = os.ReadFile(target)
			if err != nil {
				return nil, err
			}
		}
		file := filepath.Base(target)
		filePreview := FilePreview{
			InstanceID: r.InstanceID,
			Path:       target,
			OldBytes:   oldBytes,
			OldDigest:  current,
		}
		if r.BackupRef == "" {
			if r.Provenance == ProvenanceRemoved {
				return nil, fmt.Errorf("%w: backup %s removal record %s has no saved content", ErrLegacyProvenance, backupID, target)
			}
			// The transaction created this file; reversing it removes it.
			filePreview.Op = FileOpRemove
			filePreview.Removed = true
			preview.Files = append(preview.Files, filePreview)
			continue
		}
		backupFile, data, err := readBackupRef(backupDir, r.BackupRef, file)
		if err != nil {
			return nil, err
		}
		if r.BackupDigest == "" || digestBytes(data) != r.BackupDigest {
			return nil, fmt.Errorf("%w: backup %s saved content for %s failed digest verification", ErrLegacyProvenance, backupID, backupFile)
		}
		filePreview.NewBytes = data
		filePreview.NewDigest = digestBytes(data)
		switch {
		case !exists:
			filePreview.Op = FileOpCreate
		case filePreview.NewDigest == filePreview.OldDigest:
			filePreview.Op = FileOpNoOp
		default:
			filePreview.Op = FileOpUpdate
		}
		preview.Files = append(preview.Files, filePreview)
	}

	preview.Counts["total"] = len(preview.Files)
	for _, f := range preview.Files {
		preview.Counts[f.Op]++
	}
	return preview, nil
}
