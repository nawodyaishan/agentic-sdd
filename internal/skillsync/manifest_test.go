package skillsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agentic-sdd/internal/version"
)

func TestApplyWritesFormat2Manifest(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(old, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC()
	if err := Sync(repo, home, true, new(discardWriter)); err != nil {
		t.Fatal(err)
	}

	backupDir := onlyBackup(t, filepath.Join(repo, "backups"))
	m, err := readManifest(filepath.Join(repo, "backups", backupDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.isLegacy() {
		t.Fatalf("manifest reports legacy: %+v", m)
	}
	if m.Format != 2 {
		t.Fatalf("format = %d, want 2", m.Format)
	}
	if m.ID != backupDir {
		t.Fatalf("id = %q, want %q", m.ID, backupDir)
	}
	if m.Created != backupDir {
		t.Fatalf("legacy created field changed: %q != %q", m.Created, backupDir)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, m.CreatedAt)
	if err != nil {
		t.Fatalf("created_at %q: %v", m.CreatedAt, err)
	}
	if createdAt.Before(before.Add(-time.Second)) || createdAt.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("created_at %v not close to now (before=%v)", createdAt, before)
	}
	if m.Operation != "apply" {
		t.Fatalf("operation = %q, want apply", m.Operation)
	}
	if m.RestoredFrom != "" {
		t.Fatalf("restored_from = %q, want empty", m.RestoredFrom)
	}
	if m.Tool == nil {
		t.Fatal("tool is nil")
	}
	if m.Tool.Version != version.Version || m.Tool.Commit != version.Commit ||
		m.Tool.Date != version.Date || m.Tool.GoVersion != version.GoVersion {
		t.Fatalf("tool = %+v, want current internal/version values", m.Tool)
	}
	wantHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}
	if m.Home != wantHome {
		t.Fatalf("home = %q, want %q", m.Home, wantHome)
	}
	wantBackupRoot := filepath.Join(repo, "backups")
	if m.BackupRoot != wantBackupRoot {
		t.Fatalf("backup_root = %q, want %q", m.BackupRoot, wantBackupRoot)
	}
	if len(m.Entries) != 4 {
		t.Fatalf("entries = %d, want 4 (one per client)", len(m.Entries))
	}

	var replaced, installed int
	for _, e := range m.Entries {
		if e.Skill != "agentic-sdd-plan" {
			t.Fatalf("unexpected skill in entry: %+v", e)
		}
		wantPath := filepath.Join(home, clientSubpath(t, e.Client), "agentic-sdd-plan")
		if e.Path != wantPath {
			t.Fatalf("entry %+v path = %q, want %q", e, e.Path, wantPath)
		}
		switch e.Action {
		case actionReplaced:
			replaced++
			if e.Client != "claude" {
				t.Fatalf("unexpected replaced client: %+v", e)
			}
			if e.Backup != filepath.Join("claude", "agentic-sdd-plan") {
				t.Fatalf("backup path = %q", e.Backup)
			}
			if e.SHA256 == "" {
				t.Fatal("sha256 empty for replaced entry")
			}
			got, err := treeDigest(os.DirFS(filepath.Join(repo, "backups", backupDir)), e.Backup)
			if err != nil {
				t.Fatal(err)
			}
			if got != e.SHA256 {
				t.Fatalf("recomputed digest %q != recorded %q", got, e.SHA256)
			}
		case actionInstalled:
			installed++
			if e.Backup != "" || e.SHA256 != "" {
				t.Fatalf("installed entry has backup/sha256: %+v", e)
			}
		default:
			t.Fatalf("unknown action %q", e.Action)
		}
	}
	if replaced != 1 || installed != 3 {
		t.Fatalf("replaced=%d installed=%d, want 1 and 3", replaced, installed)
	}
}

func TestReadManifestDetectsLegacyVersusFormat2(t *testing.T) {
	dir := t.TempDir()

	legacyPath := filepath.Join(dir, "legacy.json")
	legacy := manifest{Created: "20260101T000000.000000000Z", Source: "embedded", Targets: []string{"/home/.claude/skills"}, Skills: []string{"agentic-sdd-plan"}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	// A real legacy manifest never had the format 2 fields on disk at all;
	// marshal only the format 1 shape to be sure readManifest's legacy
	// detection does not depend on Go's zero-value handling of a struct it
	// never saw.
	var legacyOnly map[string]any
	if err := json.Unmarshal(data, &legacyOnly); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"format", "id", "created_at", "operation", "restored_from", "tool", "home", "backup_root", "entries"} {
		delete(legacyOnly, k)
	}
	data, err = json.Marshal(legacyOnly)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readManifest(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !got.isLegacy() {
		t.Fatalf("expected legacy manifest, got %+v", got)
	}
	if got.Created != legacy.Created || got.Source != legacy.Source || len(got.Skills) != 1 {
		t.Fatalf("legacy fields not preserved: %+v", got)
	}

	format2Path := filepath.Join(dir, "format2.json")
	f2 := manifest{
		Created: "20260102T000000.000000000Z", Source: "embedded",
		Targets: []string{"/home/.claude/skills"}, Skills: []string{"agentic-sdd-plan"},
		Format: 2, ID: "20260102T000000.000000000Z", CreatedAt: "2026-01-02T00:00:00Z",
		Operation: "apply",
		Tool:      &toolInfo{Version: "v1.0.0", Commit: "abc", Date: "2026-01-02T00:00:00Z", GoVersion: "go1.23"},
		Home:      "/home", BackupRoot: "/home/.agentic-sdd/backups",
		Entries: []entry{{Client: "claude", Skill: "agentic-sdd-plan", Path: "/home/.claude/skills/agentic-sdd-plan", Action: actionInstalled}},
	}
	data, err = json.MarshalIndent(f2, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(format2Path, data, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = readManifest(format2Path)
	if err != nil {
		t.Fatal(err)
	}
	if got.isLegacy() {
		t.Fatalf("expected format 2 manifest, got %+v", got)
	}
	if got.Operation != "apply" || got.Tool == nil || got.Tool.Version != "v1.0.0" || len(got.Entries) != 1 {
		t.Fatalf("format 2 fields not preserved: %+v", got)
	}
}

// discardWriter satisfies io.Writer without importing io/ioutil or os, so
// tests that only care about the manifest do not need to inspect Sync's
// human-readable output.
type discardWriter struct{}

func (*discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func onlyBackup(t *testing.T, backupRoot string) string {
	t.Helper()
	entries, err := os.ReadDir(backupRoot)
	if err != nil || len(entries) != 1 {
		t.Fatalf("backups: %v %v", entries, err)
	}
	return entries[0].Name()
}

func clientSubpath(t *testing.T, client string) string {
	t.Helper()
	switch client {
	case "agents":
		return filepath.Join(".agents", "skills")
	case "codex":
		return filepath.Join(".codex", "skills")
	case "claude":
		return filepath.Join(".claude", "skills")
	case "agy":
		return filepath.Join(".gemini", "antigravity-cli", "skills")
	default:
		t.Fatalf("unknown client %q", client)
		return ""
	}
}
