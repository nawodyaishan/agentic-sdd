package skillsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// B6 (T12) manifest-reader coverage: bounded, regular-file-only reads.

const b6BackupID = "20260101T000000.000000000Z"

// b6BareBackupRoot returns a home and a backup root holding one empty backup
// directory named like a real backup, so only manifest.json varies per case.
func b6BareBackupRoot(t *testing.T) (root, id, home string) {
	t.Helper()
	home = t.TempDir()
	root = filepath.Join(home, ".agentic-sdd", "backups")
	if err := os.MkdirAll(filepath.Join(root, b6BackupID), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, b6BackupID, home
}

// TestB6ManifestLimitIsFourMiBPerFile: a manifest of exactly the limit is
// read; one byte more is refused as oversize before parsing.
func TestB6ManifestLimitIsFourMiBPerFile(t *testing.T) {
	const limit = 4 << 20
	for _, tc := range []struct {
		name     string
		size     int
		oversize bool
	}{
		{"at limit", limit, false},
		{"limit plus one", limit + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, id, home := b6BareBackupRoot(t)
			data := []byte(strings.Repeat(" ", tc.size))
			copy(data, `{"created":"b6"}`)
			if err := os.WriteFile(filepath.Join(root, id, "manifest.json"), data, 0o600); err != nil {
				t.Fatal(err)
			}
			_, _, err := loadBackup(root, id, home)
			if err == nil {
				t.Fatal("loadBackup accepted a manifest that cannot match this home")
			}
			if got := strings.Contains(err.Error(), "exceeds"); got != tc.oversize {
				t.Fatalf("oversize refusal = %v for %d bytes, want %v: %v", got, tc.size, tc.oversize, err)
			}
		})
	}
}

// TestB6SymlinkManifestRefused: a manifest that is a symlink is never followed.
func TestB6SymlinkManifestRefused(t *testing.T) {
	root, id, home := b6BareBackupRoot(t)
	target := filepath.Join(home, "elsewhere.json")
	if err := os.WriteFile(target, []byte(`{"created":"b6"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, id, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadBackup(root, id, home); err == nil {
		t.Fatal("loadBackup followed a symlinked manifest")
	}
}

// TestB6DirectoryManifestRefused: a directory where manifest.json belongs is
// refused, not read.
func TestB6DirectoryManifestRefused(t *testing.T) {
	root, id, home := b6BareBackupRoot(t)
	if err := os.Mkdir(filepath.Join(root, id, "manifest.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadBackup(root, id, home); err == nil {
		t.Fatal("loadBackup accepted a directory as manifest.json")
	}
}
