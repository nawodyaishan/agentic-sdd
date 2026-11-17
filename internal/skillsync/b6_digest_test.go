package skillsync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// B6 (T12) digest-dispatch coverage. Fixtures are written as raw JSON so they
// express the on-disk shapes older and newer writers produce, not the types
// the production code uses to read them.

// b6Names returns two embedded skill names for fixtures that replace trees.
func b6Names(t *testing.T) []string {
	t.Helper()
	source, err := sourceFS("")
	if err != nil {
		t.Fatal(err)
	}
	names, err := discoverSkills(source)
	if err != nil || len(names) < 2 {
		t.Fatalf("need two embedded skills, got %v (err %v)", names, err)
	}
	return names
}

// b6ReplacedBackup installs the embedded skills, edits the named installed
// skills, and applies again, so the newest backup replaces exactly those trees.
// It returns the backup ID and directory.
func b6ReplacedBackup(t *testing.T, home string, edited ...string) (string, string) {
	t.Helper()
	if err := Sync("", home, true, io.Discard); err != nil {
		t.Fatalf("initial install: %v", err)
	}
	for _, name := range edited {
		path := filepath.Join(home, ".agents", "skills", name, "SKILL.md")
		data := b6ReadFile(t, path)
		if err := os.WriteFile(path, append(data, []byte("\nB6 local edit\n")...), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Sync("", home, true, io.Discard); err != nil {
		t.Fatalf("replacing install: %v", err)
	}
	backups, err := ListBackups("", home)
	if err != nil || len(backups) == 0 || backups[0].Unusable != "" {
		t.Fatalf("backups = %+v, err %v", backups, err)
	}
	return backups[0].ID, filepath.Join(home, ".agentic-sdd", "backups", backups[0].ID)
}

func b6ReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func b6ReadManifestDoc(t *testing.T, backupDir string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(b6ReadFile(t, filepath.Join(backupDir, "manifest.json")), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func b6WriteManifestDoc(t *testing.T, backupDir string, doc map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

// b6Entry returns the manifest entry whose saved backup path is backup.
func b6Entry(t *testing.T, doc map[string]any, backup string) map[string]any {
	t.Helper()
	entries, _ := doc["entries"].([]any)
	for _, raw := range entries {
		if entry, ok := raw.(map[string]any); ok && entry["backup"] == backup {
			return entry
		}
	}
	t.Fatalf("no manifest entry for %s", backup)
	return nil
}

// b6MakeLegacyTreeEntry rewrites one replaced skill entry to the shape older
// writers produced: no digest_version and the legacy tree digest.
func b6MakeLegacyTreeEntry(t *testing.T, backupDir string, doc map[string]any, backup string) {
	t.Helper()
	digest, err := treeDigest(os.DirFS(backupDir), backup)
	if err != nil {
		t.Fatal(err)
	}
	entry := b6Entry(t, doc, backup)
	delete(entry, "digest_version")
	entry["sha256"] = digest
}

// b6AssertRestored restores id and checks that name's installed tree is back to
// the state the backup captured. The backup holds the locally edited tree (the
// state just before the replacing apply), so the edit marker must reappear.
// The source copy never carries the marker, so its presence proves the restore.
func b6AssertRestored(t *testing.T, home, id, name string) {
	t.Helper()
	if _, err := RestoreResult("", home, id, true, io.Discard); err != nil {
		t.Fatalf("restore %s: %v", id, err)
	}
	got := b6ReadFile(t, filepath.Join(home, ".agents", "skills", name, "SKILL.md"))
	if !strings.Contains(string(got), "B6 local edit") {
		t.Fatalf("%s was not restored to the backed-up tree", name)
	}
}

func b6BackupCount(t *testing.T, home string) int {
	t.Helper()
	backups, err := ListBackups("", home)
	if err != nil {
		t.Fatal(err)
	}
	return len(backups)
}

// TestB6LegacyTreeDigestBackupStillRestores keeps format-3 backups written
// before digest_version existed restorable under the legacy tree semantics.
func TestB6LegacyTreeDigestBackupStillRestores(t *testing.T) {
	home := t.TempDir()
	names := b6Names(t)
	id, dir := b6ReplacedBackup(t, home, names[0])
	doc := b6ReadManifestDoc(t, dir)
	b6MakeLegacyTreeEntry(t, dir, doc, filepath.Join("agents", names[0]))
	b6WriteManifestDoc(t, dir, doc)

	b6AssertRestored(t, home, id, names[0])
}

// TestB6MixedDigestVersionsInOneBackup restores a backup whose tree entries use
// different digest versions: one new (v2) and one legacy (absent).
func TestB6MixedDigestVersionsInOneBackup(t *testing.T) {
	home := t.TempDir()
	names := b6Names(t)
	id, dir := b6ReplacedBackup(t, home, names[0], names[1])
	doc := b6ReadManifestDoc(t, dir)

	v2Backup := filepath.Join("agents", names[0])
	v2, err := TreeDigestV2(filepath.Join(dir, v2Backup))
	if err != nil {
		t.Fatal(err)
	}
	entry := b6Entry(t, doc, v2Backup)
	entry["digest_version"] = 2
	entry["sha256"] = v2

	b6MakeLegacyTreeEntry(t, dir, doc, filepath.Join("agents", names[1]))
	b6WriteManifestDoc(t, dir, doc)

	b6AssertRestored(t, home, id, names[0])
	b6AssertRestored(t, home, id, names[1])
}

// TestB6UnknownTreeDigestVersionRejected: any skill digest_version other than
// absent/0 (legacy) or 2 is refused before anything is written.
func TestB6UnknownTreeDigestVersionRejected(t *testing.T) {
	for _, version := range []any{1, 3, -1, 99} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			home := t.TempDir()
			names := b6Names(t)
			id, dir := b6ReplacedBackup(t, home, names[0])
			doc := b6ReadManifestDoc(t, dir)
			backup := filepath.Join("agents", names[0])
			b6MakeLegacyTreeEntry(t, dir, doc, backup)
			b6Entry(t, doc, backup)["digest_version"] = version
			b6WriteManifestDoc(t, dir, doc)

			skill := filepath.Join(home, ".agents", "skills", names[0], "SKILL.md")
			wantSkill := b6ReadFile(t, skill)
			backupsBefore := b6BackupCount(t, home)

			_, err := RestoreResult("", home, id, true, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "digest_version") {
				t.Fatalf("restore with digest_version %v error = %v, want an unsupported digest_version refusal", version, err)
			}
			if got := b6ReadFile(t, skill); string(got) != string(wantSkill) {
				t.Fatal("refused restore changed the installed skill")
			}
			if after := b6BackupCount(t, home); after != backupsBefore {
				t.Fatalf("refused restore created a backup: %d -> %d", backupsBefore, after)
			}
		})
	}
}

// TestB6AgentEntryDigestVersionRejectedAndWritesStayRaw: agent-profile entries
// keep the raw file SHA-256 and never carry a digest_version.
func TestB6AgentEntryDigestVersionRejectedAndWritesStayRaw(t *testing.T) {
	home := t.TempDir()
	if _, err := SyncWithOptionsResult("", home, true, Options{WithOpenCode: true}, io.Discard); err != nil {
		t.Fatalf("initial OpenCode install: %v", err)
	}
	agentDir := filepath.Join(home, ".config", "opencode", "agents")
	profiles, err := os.ReadDir(agentDir)
	if err != nil || len(profiles) == 0 {
		t.Fatalf("no installed profiles: %v", err)
	}
	profile := profiles[0].Name()
	path := filepath.Join(agentDir, profile)
	if err := os.WriteFile(path, append(b6ReadFile(t, path), []byte("\nB6 local edit\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncWithOptionsResult("", home, true, Options{WithOpenCode: true}, io.Discard); err != nil {
		t.Fatalf("replacing OpenCode install: %v", err)
	}
	backups, err := ListBackups("", home)
	if err != nil || len(backups) == 0 || backups[0].Unusable != "" {
		t.Fatalf("backups = %+v, err %v", backups, err)
	}
	id := backups[0].ID
	dir := filepath.Join(home, ".agentic-sdd", "backups", id)
	doc := b6ReadManifestDoc(t, dir)
	backup := filepath.Join("opencode-agents", profile)
	entry := b6Entry(t, doc, backup)
	if _, ok := entry["digest_version"]; ok {
		t.Fatalf("agent entry carries digest_version %v; agent entries use the raw digest only", entry["digest_version"])
	}
	raw, err := FileDigest(os.DirFS(dir), backup)
	if err != nil {
		t.Fatal(err)
	}
	if entry["sha256"] != raw {
		t.Fatalf("agent entry sha256 = %v, want raw file digest %s", entry["sha256"], raw)
	}

	entry["digest_version"] = 2
	b6WriteManifestDoc(t, dir, doc)
	if _, err := RestoreResult("", home, id, true, io.Discard); err == nil {
		t.Fatal("restore accepted a nonzero digest_version on an agent entry")
	}
}

// TestB6ReplacedEntryWithoutSHA256Rejected: a replaced entry must carry the
// digest it was written with; an empty one is refused, not skipped.
func TestB6ReplacedEntryWithoutSHA256Rejected(t *testing.T) {
	home := t.TempDir()
	names := b6Names(t)
	id, dir := b6ReplacedBackup(t, home, names[0])
	doc := b6ReadManifestDoc(t, dir)
	delete(b6Entry(t, doc, filepath.Join("agents", names[0])), "sha256")
	b6WriteManifestDoc(t, dir, doc)

	skill := filepath.Join(home, ".agents", "skills", names[0], "SKILL.md")
	wantSkill := b6ReadFile(t, skill)
	if _, err := RestoreResult("", home, id, true, io.Discard); err == nil {
		t.Fatal("restore accepted a replaced entry without sha256")
	}
	if got := b6ReadFile(t, skill); string(got) != string(wantSkill) {
		t.Fatal("refused restore changed the installed skill")
	}
}

// TestB6ApplyWritesDigestVersionTwoForTreeEntries: new tree entries record the
// v2 codec digest of the saved tree.
func TestB6ApplyWritesDigestVersionTwoForTreeEntries(t *testing.T) {
	home := t.TempDir()
	names := b6Names(t)
	_, dir := b6ReplacedBackup(t, home, names[0])
	backup := filepath.Join("agents", names[0])
	entry := b6Entry(t, b6ReadManifestDoc(t, dir), backup)
	if entry["digest_version"] != float64(2) {
		t.Fatalf("replaced tree entry digest_version = %v, want 2", entry["digest_version"])
	}
	want, err := TreeDigestV2(filepath.Join(dir, backup))
	if err != nil {
		t.Fatal(err)
	}
	if entry["sha256"] != want {
		t.Fatalf("replaced tree entry sha256 = %v, want TreeDigestV2 %s", entry["sha256"], want)
	}
}
