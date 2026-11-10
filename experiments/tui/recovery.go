package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agentic-sdd/internal/skillsync"
)

// unknownCount marks a recorded-change count this build cannot prove from the
// manifest format. It renders as "unknown", never as zero.
const unknownCount = -1

// BackupBreakdown holds truthful, verified facts derived from a backup entry.
// Fresh-install entries (recorded to allow clean removal on restore) are
// distinguished from saved replacement snapshots (prior file content preserved
// on disk). A count is unknownCount when the manifest format does not record
// it.
//
// AnalyzeBackup is pure: it reads only the already-validated skillsync.Backup
// value. It never touches the filesystem, so rendering and key handling stay
// free of I/O.
type BackupBreakdown struct {
	ID                    string
	ReadableDate          string
	RawDate               string
	Operation             string
	Source                string
	Format                string
	IsLegacy              bool
	IsUnusable            bool
	UnusableReason        string
	IsEmpty               bool
	RecordedFreshInstalls int
	SavedReplacements     int
	AffectedClients       []string
	// ClientsKnown reports whether AffectedClients is authoritative for this
	// manifest. It is false for legacy, unusable and unrecorded metadata.
	ClientsKnown bool
}

// AnalyzeBackup inspects a validated backup and produces an honest breakdown
// of recorded changes versus preserved file snapshots. It performs no I/O.
func AnalyzeBackup(backup skillsync.Backup) BackupBreakdown {
	breakdown := BackupBreakdown{
		ID:                    backup.ID,
		RawDate:               restoreWhen(backup),
		ReadableDate:          formatReadableDate(restoreWhen(backup)),
		Operation:             restoreOperation(backup),
		Source:                restoreSource(backup),
		Format:                restoreFormat(backup),
		IsLegacy:              backup.Legacy,
		IsUnusable:            backup.Unusable != "",
		UnusableReason:        backup.Unusable,
		RecordedFreshInstalls: unknownCount,
		SavedReplacements:     unknownCount,
	}

	if breakdown.IsUnusable {
		// A failed entry has zero-value metadata. Nothing about its recorded
		// changes can be claimed, so every count stays unknown.
		return breakdown
	}

	if backup.Legacy {
		// Format 1 recorded the skill set and destination paths, not the saved
		// per-client snapshots the restore engine discovers on disk. The
		// validated list summary carries those disk-derived facts; this screen
		// must not reinterpret the manifest fields as snapshot counts.
		return breakdown
	}
	if backup.Manifest.Format <= 0 {
		// Format not recorded: no per-entry metadata exists to count.
		return breakdown
	}

	// Format 2 & 3 manifests contain per-entry records.
	clientSet := make(map[string]bool)
	replaced, installed := 0, 0
	for _, entry := range backup.Manifest.Entries {
		if entry.Client != "" {
			clientSet[entry.Client] = true
		}
		switch strings.ToLower(entry.Action) {
		case "replaced":
			replaced++
		case "installed":
			installed++
		}
	}
	breakdown.SavedReplacements = replaced
	breakdown.RecordedFreshInstalls = installed

	clients := make([]string, 0, len(clientSet))
	for c := range clientSet {
		clients = append(clients, c)
	}
	sort.Strings(clients)
	breakdown.AffectedClients = clients
	breakdown.ClientsKnown = true

	if len(backup.Manifest.Entries) == 0 {
		// No per-destination record: the backup provably covers nothing, which
		// is different from a fresh install of recorded destinations.
		breakdown.IsEmpty = true
	}

	return breakdown
}

func formatReadableDate(raw string) string {
	if raw == "" || raw == "time not recorded" {
		return "time not recorded"
	}
	// Try parsing RFC3339Nano then RFC3339
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC().Format("Jan 02, 2006 15:04:05 UTC")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC().Format("Jan 02, 2006 15:04:05 UTC")
	}
	// Try parsing backup ID timestamp format: 20060102T150405.000000000Z
	if t, err := time.Parse("20060102T150405.000000000Z", raw); err == nil {
		return t.UTC().Format("Jan 02, 2006 15:04:05 UTC")
	}
	return raw
}

// ReadBackupManifestContent reads the bounded raw content of a backup's
// manifest.json. Max bytes is capped to DefaultMaxLogBytes (64 KiB).
//
// It validates the backup root and ID, refuses a backup directory that is not
// a real directory, and reads only a regular, non-symlink manifest. The read is
// scoped to an os.Root for the validated backup root, so a path swapped between
// the validation stat and the open cannot make this view follow a symlink
// outside that root; an in-root replacement is rejected by identity. It never
// performs any write.
func ReadBackupManifestContent(backupRoot, backupID string, maxBytes int64) (string, error) {
	return readBackupManifestContent(backupRoot, backupID, maxBytes, nil)
}

// readBackupManifestContent is ReadBackupManifestContent with an optional hook
// that runs after path validation and before the open. Production passes nil;
// a test uses it to swap the path deterministically and prove the race guard.
func readBackupManifestContent(backupRoot, backupID string, maxBytes int64, beforeOpen func()) (string, error) {
	if backupRoot == "" || backupID == "" {
		return "", errors.New("missing backup root or backup ID")
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		return "", err
	}
	rootInfo, err := os.Lstat(backupRoot)
	if err != nil {
		return "", fmt.Errorf("unable to inspect backup root: %w", err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("refusing non-directory backup root %s", backupRoot)
	}

	// os.Root confines every open below the already-validated backup root, so a
	// symlink that appears after validation cannot escape it.
	root, err := os.OpenRoot(backupRoot)
	if err != nil {
		return "", fmt.Errorf("unable to inspect backup root: %w", err)
	}
	defer root.Close()

	backupDir := filepath.Join(backupRoot, backupID)
	dirInfo, err := root.Lstat(backupID)
	if err != nil {
		return "", fmt.Errorf("unable to inspect backup directory: %w", err)
	}
	if !dirInfo.IsDir() {
		return "", fmt.Errorf("refusing non-directory backup %s", backupDir)
	}

	rel := filepath.Join(backupID, "manifest.json")
	info, err := root.Lstat(rel)
	if err != nil {
		return "", fmt.Errorf("unable to open manifest.json: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("refusing non-regular manifest.json in backup %s", backupID)
	}

	if beforeOpen != nil {
		beforeOpen()
	}

	file, err := root.Open(rel)
	if err != nil {
		return "", fmt.Errorf("refusing unreadable manifest.json in backup %s: %w", backupID, err)
	}
	defer file.Close()
	// Re-check the opened file: a path swapped after Lstat must not turn this
	// read-only view into a read of a pipe, device, symlink target or another
	// file. os.SameFile rejects a replacement even inside the root.
	openedInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to inspect manifest.json: %w", err)
	}
	if !openedInfo.Mode().IsRegular() {
		return "", fmt.Errorf("refusing non-regular manifest.json in backup %s", backupID)
	}
	if !os.SameFile(info, openedInfo) {
		return "", fmt.Errorf("refusing replaced manifest.json in backup %s", backupID)
	}

	if maxBytes <= 0 {
		maxBytes = DefaultMaxLogBytes
	}

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("error reading manifest: %w", err)
	}

	truncated := false
	if int64(len(data)) > maxBytes {
		data = data[:maxBytes]
		truncated = true
	}

	content := string(data)
	if truncated {
		content = strings.TrimRight(content, "\r\n") + "\n... [truncated: manifest exceeds display limit]"
	}
	return content, nil
}
