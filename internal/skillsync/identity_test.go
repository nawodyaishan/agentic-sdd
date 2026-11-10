package skillsync

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writePackFixture lays out a minimal on-disk OpenCode pack beside a core
// skill so source-identity tests can change one source at a time.
func writePackFixture(t *testing.T, repo string) {
	t.Helper()
	writeTestSkillSource(t, repo, "agentic-sdd-plan", "new")
	files := map[string]string{
		"opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md": "orchestrate",
		"opencode-pack/agents/sdd-base.md":                      "base profile",
		"opencode-pack/presets/free-first/sdd-free.md":          "free profile",
	}
	for rel, content := range files {
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSyncSourceDigestStableAcrossPreviewAndApply proves the source identity
// ignores the destination state it is compared against and agrees between a
// preview and the apply that writes.
func TestSyncSourceDigestStableAcrossPreviewAndApply(t *testing.T) {
	repo, home := newSkillFixture(t)
	var out bytes.Buffer

	preview, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SourceDigest == "" {
		t.Fatal("preview result carries no source digest")
	}
	again, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if again.SourceDigest != preview.SourceDigest {
		t.Fatalf("preview digest changed between identical runs: %q != %q", again.SourceDigest, preview.SourceDigest)
	}

	// A destination change alters the plan but not the source identity.
	writeOldSkill(t, home, "claude", "agentic-sdd-plan", "old")
	withDestination, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if withDestination.SourceDigest != preview.SourceDigest {
		t.Fatalf("destination state changed the source digest: %q != %q", withDestination.SourceDigest, preview.SourceDigest)
	}

	applied, err := SyncWithOptionsResult(repo, home, true, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied {
		t.Fatalf("apply outcome = %q", applied.Outcome)
	}
	if applied.SourceDigest != preview.SourceDigest {
		t.Fatalf("apply digest %q != preview digest %q", applied.SourceDigest, preview.SourceDigest)
	}

	noChange, err := SyncWithOptionsResult(repo, home, true, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if noChange.Outcome != OutcomeNoChange || noChange.SourceDigest != preview.SourceDigest {
		t.Fatalf("no-change result = %+v", noChange)
	}
}

// TestSyncSourceDigestTracksCoreSource proves a changed or added source file
// changes the digest.
func TestSyncSourceDigestTracksCoreSource(t *testing.T) {
	repo, home := newSkillFixture(t)
	var out bytes.Buffer

	base, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(base.SourceDigest) != 64 {
		t.Fatalf("source digest = %q, want 64 hex characters", base.SourceDigest)
	}

	writeTestSkillSource(t, repo, "agentic-sdd-plan", "changed")
	changed, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if changed.SourceDigest == base.SourceDigest {
		t.Fatalf("changed source content kept digest %q", base.SourceDigest)
	}

	extra := filepath.Join(repo, "skills-files", "agentic-sdd-plan", "extra.md")
	if err := os.WriteFile(extra, []byte("extra"), 0644); err != nil {
		t.Fatal(err)
	}
	added, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if added.SourceDigest == changed.SourceDigest {
		t.Fatalf("added source file kept digest %q", changed.SourceDigest)
	}
}

// TestSyncSourceDigestTracksOpenCodePack proves the optional pack skill and
// profile sources the plan would install are part of the identity, and that
// enabling the preset adds its sources.
func TestSyncSourceDigestTracksOpenCodePack(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	writePackFixture(t, repo)
	var out bytes.Buffer

	core, err := SyncWithOptionsResult(repo, home, false, Options{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := SyncWithOptionsResult(repo, home, false, Options{WithOpenCode: true}, &out)
	if err != nil {
		t.Fatal(err)
	}
	preset, err := SyncWithOptionsResult(repo, home, false, Options{WithOpenCode: true, OpenCodeFreeFirst: true}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if core.SourceDigest == pack.SourceDigest {
		t.Fatal("enabling the OpenCode pack kept the core-only digest")
	}
	if pack.SourceDigest == preset.SourceDigest {
		t.Fatal("enabling the Free-first preset kept the same digest")
	}

	agentsProfile := filepath.Join(repo, "opencode-pack", "agents", "sdd-base.md")
	if err := os.WriteFile(agentsProfile, []byte("changed profile"), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := SyncWithOptionsResult(repo, home, false, Options{WithOpenCode: true}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if changed.SourceDigest == pack.SourceDigest {
		t.Fatal("changed pack profile content kept the same digest")
	}
}

// TestRestoreSourceDigestStableAndManifestSensitive proves the restore
// identity ignores destination state, agrees between preview and apply, and
// covers the manifest bytes.
func TestRestoreSourceDigestStableAndManifestSensitive(t *testing.T) {
	repo, home, backupRoot, id := setupBaseBackup(t)
	backupDir := filepath.Join(backupRoot, id)
	var out bytes.Buffer

	preview, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SourceDigest == "" {
		t.Fatal("restore preview carries no source digest")
	}
	again, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if again.SourceDigest != preview.SourceDigest {
		t.Fatalf("restore preview digest changed: %q != %q", again.SourceDigest, preview.SourceDigest)
	}

	// Removing a destination changes the restore plan, not the backup identity.
	if err := os.RemoveAll(filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")); err != nil {
		t.Fatal(err)
	}
	destinationChanged, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if destinationChanged.SourceDigest != preview.SourceDigest {
		t.Fatalf("destination state changed the restore digest: %q != %q", destinationChanged.SourceDigest, preview.SourceDigest)
	}

	// The manifest bytes are part of the identity; a valid semantic change
	// (the recorded source label) is enough to change it.
	m, err := readManifest(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.Source = "a different source label"
	writeIdentityManifest(t, backupDir, m)
	manifestChanged, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	if manifestChanged.SourceDigest == preview.SourceDigest {
		t.Fatal("changed manifest bytes kept the restore digest")
	}

	applied, err := RestoreResult(repo, home, id, true, &out)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Outcome != OutcomeApplied || applied.SourceDigest != manifestChanged.SourceDigest {
		t.Fatalf("restore apply digest %q, want %q (outcome %q)", applied.SourceDigest, manifestChanged.SourceDigest, applied.Outcome)
	}
}

// TestRestoreSourceDigestTracksPayload proves a consistent payload change
// (one whose manifest digest was updated with it, so validation still
// accepts the backup) changes the restore identity.
func TestRestoreSourceDigestTracksPayload(t *testing.T) {
	repo, home, backupRoot, id := setupBaseBackup(t)
	backupDir := filepath.Join(backupRoot, id)
	var out bytes.Buffer

	base, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("claude", "agentic-sdd-plan")
	payload := filepath.Join(backupDir, rel, "SKILL.md")
	if err := os.WriteFile(payload, []byte("tampered but consistently recorded"), 0644); err != nil {
		t.Fatal(err)
	}
	m, err := readManifest(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest, err := treeDigest(os.DirFS(backupDir), rel)
	if err != nil {
		t.Fatal(err)
	}
	updated := false
	for i := range m.Entries {
		if filepath.Join(m.Entries[i].Client, m.Entries[i].Skill) == rel {
			m.Entries[i].SHA256 = digest
			updated = true
		}
	}
	if !updated {
		t.Fatalf("backup has no entry for %s", rel)
	}
	writeIdentityManifest(t, backupDir, m)

	changed, err := RestoreResult(repo, home, id, false, &out)
	if err != nil {
		t.Fatalf("consistently changed backup was refused: %v", err)
	}
	if changed.SourceDigest == base.SourceDigest {
		t.Fatal("changed backup payload kept the restore digest")
	}
}

// writeIdentityManifest rewrites a backup manifest exactly as the engine
// writes it (indented JSON plus a newline).
func writeIdentityManifest(t *testing.T, backupDir string, m manifest) {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
