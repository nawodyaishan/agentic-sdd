package skillsync

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// B6 (T12) before-state coverage for the agent-file writer. The engine re-reads
// each destination under the held guard; the caller's Old flag and the backup
// it asks for are not authority over what gets saved or replaced.

func b6AgentFixture(t *testing.T) (home, agentDir, backupRoot string) {
	t.Helper()
	home = t.TempDir()
	agentDir = filepath.Join(home, ".config", "opencode", "agents")
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		t.Fatal(err)
	}
	return home, agentDir, filepath.Join(home, ".agentic-sdd", "backups")
}

// TestB6ApplyAgentFilesRefusesCallerAbsentButPresentTarget: the caller says the
// profile is absent, but a newer target exists. It is refused, never replaced.
func TestB6ApplyAgentFilesRefusesCallerAbsentButPresentTarget(t *testing.T) {
	home, agentDir, backupRoot := b6AgentFixture(t)
	dst := filepath.Join(agentDir, "sdd-b6.md")
	if err := os.WriteFile(dst, []byte("newer target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	changes := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("fleet\n"), Old: false}}
	_, err := ApplyAgentFiles(backupRoot, home, "fleet", changes, nil, io.Discard)
	if !errors.Is(err, ErrBeforeStateChanged) {
		t.Fatalf("ApplyAgentFiles error = %v, want ErrBeforeStateChanged", err)
	}
	if got := b6ReadFile(t, dst); string(got) != "newer target\n" {
		t.Fatal("refused apply overwrote the newer target")
	}
	if _, statErr := os.Lstat(backupRoot); !os.IsNotExist(statErr) {
		t.Fatalf("refused apply created a backup root: %v", statErr)
	}
}

// TestB6ApplyAgentFilesRefusesCallerPresentButAbsentTarget: the caller says the
// profile exists, but it is gone. Nothing is created, and nothing is backed up.
func TestB6ApplyAgentFilesRefusesCallerPresentButAbsentTarget(t *testing.T) {
	home, agentDir, backupRoot := b6AgentFixture(t)
	changes := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("fleet\n"), Old: true}}
	_, err := ApplyAgentFiles(backupRoot, home, "fleet", changes, nil, io.Discard)
	if !errors.Is(err, ErrBeforeStateChanged) {
		t.Fatalf("ApplyAgentFiles error = %v, want ErrBeforeStateChanged", err)
	}
	if _, statErr := os.Lstat(filepath.Join(agentDir, "sdd-b6.md")); !os.IsNotExist(statErr) {
		t.Fatalf("refused apply created the profile: %v", statErr)
	}
}

// TestB6ApplyAgentFilesRefusesDirectoryAtFileTargetBeforeBackup: a directory at
// a planned file destination is refused by the before-state re-read, before any
// backup or staging. The directory and its contents are left untouched.
func TestB6ApplyAgentFilesRefusesDirectoryAtFileTargetBeforeBackup(t *testing.T) {
	home, agentDir, backupRoot := b6AgentFixture(t)
	blocked := filepath.Join(agentDir, "sdd-b6.md")
	if err := os.MkdirAll(blocked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "marker"), []byte("dir"), 0o600); err != nil {
		t.Fatal(err)
	}
	changes := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("fleet\n"), Old: false}}
	if _, err := ApplyAgentFiles(backupRoot, home, "fleet", changes, nil, io.Discard); err == nil {
		t.Fatal("ApplyAgentFiles accepted a directory at a file target")
	}
	if got := b6ReadFile(t, filepath.Join(blocked, "marker")); string(got) != "dir" {
		t.Fatal("refused apply modified the directory")
	}
	if _, statErr := os.Lstat(backupRoot); !os.IsNotExist(statErr) {
		t.Fatalf("refused apply created a backup root: %v", statErr)
	}
}

// TestB6ApplyAgentFilesChecksExpectedDigestAndBacksUpDisk: with an expected
// digest, a mismatch is refused untouched; a match backs up the bytes on disk.
func TestB6ApplyAgentFilesChecksExpectedDigestAndBacksUpDisk(t *testing.T) {
	home, agentDir, backupRoot := b6AgentFixture(t)
	dst := filepath.Join(agentDir, "sdd-b6.md")
	onDisk := []byte("on disk\n")
	if err := os.WriteFile(dst, onDisk, 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := FileDigest(os.DirFS(agentDir), "sdd-b6.md")
	if err != nil {
		t.Fatal(err)
	}

	stale := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("fleet\n"), Old: true, ExpectSHA256: "0000"}}
	if _, err := ApplyAgentFiles(backupRoot, home, "fleet", stale, nil, io.Discard); !errors.Is(err, ErrBeforeStateChanged) {
		t.Fatalf("mismatched expected digest error = %v, want ErrBeforeStateChanged", err)
	}
	if got := b6ReadFile(t, dst); string(got) != string(onDisk) {
		t.Fatal("digest mismatch changed the profile")
	}

	match := []AgentFileChange{{File: "sdd-b6.md", Data: []byte("fleet\n"), Old: true, ExpectSHA256: current}}
	result, err := ApplyAgentFiles(backupRoot, home, "fleet", match, nil, io.Discard)
	if err != nil {
		t.Fatalf("matching expected digest: %v", err)
	}
	saved, err := os.ReadFile(filepath.Join(result.BackupDir, "opencode-agents", "sdd-b6.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(onDisk) {
		t.Fatalf("backup holds %q, want the bytes read from disk under the guard", saved)
	}
}
