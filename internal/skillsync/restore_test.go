package skillsync

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupBaseBackup runs one apply that both replaces an existing skill
// (agentic-sdd-plan, pre-existing only under claude) and installs it fresh
// under the other three clients, then returns the resulting backup's root
// and ID for restore tests to build on.
func setupBaseBackup(t *testing.T) (repo, home, backupRoot, id string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "home")
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
	var out bytes.Buffer
	if err := Sync(repo, home, true, &out); err != nil {
		t.Fatal(err)
	}
	backupRoot = filepath.Join(repo, "backups")
	id = onlyBackup(t, backupRoot)
	return repo, home, backupRoot, id
}

func cloneBackupRoot(t *testing.T, srcRoot, id string) string {
	t.Helper()
	dstRoot := t.TempDir()
	if err := copyTree(filepath.Join(srcRoot, id), filepath.Join(dstRoot, id)); err != nil {
		t.Fatal(err)
	}
	return dstRoot
}

func rewriteManifest(t *testing.T, backupDir string, mutate func(m map[string]any)) {
	t.Helper()
	path := filepath.Join(backupDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	mutate(m)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
}

func entryFor(m map[string]any, client, skill string) map[string]any {
	entries := m["entries"].([]any)
	for _, e := range entries {
		entry := e.(map[string]any)
		if entry["client"] == client && entry["skill"] == skill {
			return entry
		}
	}
	return nil
}

func TestListBackupsEmptyOrMissingRoot(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	backups, err := ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected no backups, got %v", backups)
	}
	// An existing but empty root behaves the same way.
	if err := os.MkdirAll(filepath.Join(repo, "backups"), 0755); err != nil {
		t.Fatal(err)
	}
	backups, err = ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected no backups in empty root, got %v", backups)
	}
}

func TestListBackupsOrderingLegacyAndSummary(t *testing.T) {
	repo, home, backupRoot, id1 := setupBaseBackup(t)
	// A second apply (no-op) creates no second backup, so fabricate a
	// second, later, legacy-format backup by hand to exercise ordering
	// and the legacy marker together.
	id2 := "20990101T000000.000000000Z" // lexically after any real id1
	legacyDir := filepath.Join(backupRoot, id2, "claude", "agentic-sdd-plan")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "SKILL.md"), []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	targets := clientTargets(home)
	var targetPaths []string
	for _, tg := range targets {
		targetPaths = append(targetPaths, tg.path)
	}
	legacy := map[string]any{
		"created": id2, "source": "embedded", "targets": targetPaths, "skills": []string{"agentic-sdd-plan"},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupRoot, id2, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	backups, err := ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d: %+v", len(backups), backups)
	}
	if backups[0].ID != id2 || backups[1].ID != id1 {
		t.Fatalf("expected newest first (%s, %s), got (%s, %s)", id2, id1, backups[0].ID, backups[1].ID)
	}
	if !backups[0].Legacy {
		t.Fatalf("expected legacy backup marked legacy: %+v", backups[0])
	}
	if backups[1].Legacy {
		t.Fatalf("format 2 backup marked legacy: %+v", backups[1])
	}
	if backups[1].Unusable != "" {
		t.Fatalf("valid backup marked unusable: %s", backups[1].Unusable)
	}
	if !strings.Contains(backups[1].Summary, "replaced") || !strings.Contains(backups[1].Summary, "installed") {
		t.Fatalf("summary missing counts: %q", backups[1].Summary)
	}
}

func TestRestorePreviewWritesNothing(t *testing.T) {
	repo, home, backupRoot, id := setupBaseBackup(t)
	var out bytes.Buffer
	if err := Restore(repo, home, id, false, &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"replace + back up claude/agentic-sdd-plan",
		"remove + back up agents/agentic-sdd-plan",
		"remove + back up codex/agentic-sdd-plan",
		"remove + back up agy/agentic-sdd-plan",
		"Preview only. Run with --apply (or restore apply) to restore.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("preview output missing %q:\n%s", want, text)
		}
	}
	if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "new" {
		t.Fatalf("preview changed claude skill: %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")); err != nil {
		t.Fatalf("preview removed a fresh install: %v", err)
	}
	backups, err := os.ReadDir(backupRoot)
	if err != nil || len(backups) != 1 {
		t.Fatalf("preview created a backup: %v %v", backups, err)
	}
}

func TestRestoreFormat2RoundTrip(t *testing.T) {
	repo, home, backupRoot, id1 := setupBaseBackup(t)
	claudePlan := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	agentsPlan := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	codexPlan := filepath.Join(home, ".codex", "skills", "agentic-sdd-plan")
	agyPlan := filepath.Join(home, ".gemini", "antigravity-cli", "skills", "agentic-sdd-plan")

	var out bytes.Buffer
	if err := Restore(repo, home, id1, true, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "old" {
		t.Fatalf("claude plan not reverted: %q", got)
	}
	for _, p := range []string{agentsPlan, codexPlan, agyPlan} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("fresh install %s not removed by restore: %v", p, err)
		}
	}

	backups, err := os.ReadDir(backupRoot)
	if err != nil || len(backups) != 2 {
		t.Fatalf("expected 2 backups after restore, got %v %v", backups, err)
	}
	var id2 string
	for _, b := range backups {
		if b.Name() != id1 {
			id2 = b.Name()
		}
	}
	if id2 == "" {
		t.Fatal("restore did not create a new backup")
	}
	m2, err := readManifest(filepath.Join(backupRoot, id2, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m2.Operation != "restore" || m2.RestoredFrom != id1 {
		t.Fatalf("pre-restore backup operation/restored_from = %q/%q, want restore/%s", m2.Operation, m2.RestoredFrom, id1)
	}
	if len(m2.Entries) != 4 {
		t.Fatalf("pre-restore backup entries = %d, want 4", len(m2.Entries))
	}
	for _, e := range m2.Entries {
		if e.Action != actionReplaced {
			t.Fatalf("pre-restore backup entry %+v action = %q, want replaced (everything existed before the restore)", e, e.Action)
		}
	}

	// Repeating the same restore is a no-op.
	out.Reset()
	if err := Restore(repo, home, id1, true, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "All skills are up to date.") {
		t.Fatalf("repeated restore did not report no changes: %s", out.String())
	}
	if backups, err = os.ReadDir(backupRoot); err != nil || len(backups) != 2 {
		t.Fatalf("repeated restore created another backup: %v %v", backups, err)
	}

	// Restoring the pre-restore backup undoes the restore, returning the
	// post-apply state: the reverted skill flips back, and every removed
	// skill is reinstalled with its original content.
	out.Reset()
	if err := Restore(repo, home, id2, true, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(claudePlan, "SKILL.md")); got != "new" {
		t.Fatalf("claude plan not un-reverted: %q", got)
	}
	for _, p := range []string{agentsPlan, codexPlan, agyPlan} {
		if got := read(t, filepath.Join(p, "SKILL.md")); got != "new" {
			t.Fatalf("%s not reinstalled with original content: %q", p, got)
		}
	}
}

func TestRestoreFormat1PutsBackOnlySavedTrees(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	backupRoot := filepath.Join(repo, "backups")
	id := "20260101T000000.000000000Z"
	claudeDir := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "SKILL.md"), []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}
	// A fresh install with no backup entry at all (as a real legacy apply
	// would leave one): present now, but the legacy manifest never
	// recorded it, so restore must leave it untouched.
	agentsDir := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "SKILL.md"), []byte("freshly installed"), 0644); err != nil {
		t.Fatal(err)
	}

	savedDir := filepath.Join(backupRoot, id, "claude", "agentic-sdd-plan")
	if err := os.MkdirAll(savedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(savedDir, "SKILL.md"), []byte("saved"), 0644); err != nil {
		t.Fatal(err)
	}
	targets := clientTargets(home)
	var targetPaths []string
	for _, tg := range targets {
		targetPaths = append(targetPaths, tg.path)
	}
	legacy := map[string]any{
		"created": id, "source": "embedded", "targets": targetPaths, "skills": []string{"agentic-sdd-plan"},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupRoot, id, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Restore(repo, home, id, true, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Legacy backup") {
		t.Fatalf("missing legacy notice: %s", out.String())
	}
	if got := read(t, filepath.Join(claudeDir, "SKILL.md")); got != "saved" {
		t.Fatalf("legacy restore did not put back saved tree: %q", got)
	}
	if got := read(t, filepath.Join(agentsDir, "SKILL.md")); got != "freshly installed" {
		t.Fatalf("legacy restore touched an unrecorded fresh install: %q", got)
	}
}

// TestRestoreInstallRollsBackMixedReplaceAndRemove drives installChanges
// with real restore-shaped data (skill names and a source FS taken from an
// actual backup, via loadBackup) rather than synthetic ones, covering a
// replace and a remove change together, one of which fails.
//
// Restore's own planning always Lstats a destination immediately before
// deciding its action, so a genuinely reachable "install over an existing
// non-empty directory" race can't be constructed through the public
// Restore API in a single-threaded test - by design, Restore never queues
// a plain install-rename (old: false) against a destination it just found
// to exist. This test instead calls installChanges directly, the same
// function Restore delegates to, with one change deliberately marked
// old: false despite its destination already existing (as a defensive
// regression check on installChanges itself, matching
// TestInstallChangesRollsBackRemoveOnLaterFailure in sync_test.go but with
// real backup-derived src/skill data and a remove alongside the replace).
func TestRestoreInstallRollsBackMixedReplaceAndRemove(t *testing.T) {
	_, home, baseRoot, id := setupBaseBackup(t)
	_, records, err := loadBackup(baseRoot, id, home)
	if err != nil {
		t.Fatal(err)
	}
	targets := clientTargets(home)
	targetByName := make(map[string]target, len(targets))
	for _, tg := range targets {
		targetByName[tg.name] = tg
	}
	var removeRecord, replaceRecord, blockRecord restoreRecord
	for _, r := range records {
		switch {
		case r.action == actionInstalled && removeRecord.client == "" && r.client != "codex":
			removeRecord = r
		case r.action == actionReplaced && replaceRecord.client == "":
			replaceRecord = r
		case r.client == "codex":
			blockRecord = r
		}
	}
	if removeRecord.client == "" || replaceRecord.client == "" || blockRecord.client == "" {
		t.Fatalf("test scenario missing an expected record: remove=%+v replace=%+v block=%+v", removeRecord, replaceRecord, blockRecord)
	}

	backupDir := filepath.Join(baseRoot, id)
	source := os.DirFS(backupDir)
	removeTarget := targetByName[removeRecord.client]
	replaceTarget := targetByName[replaceRecord.client]
	blockTarget := targetByName[blockRecord.client]
	removePath := filepath.Join(removeTarget.path, removeRecord.skill)
	replacePath := filepath.Join(replaceTarget.path, replaceRecord.skill)
	blockPath := filepath.Join(blockTarget.path, blockRecord.skill)

	beforeRemove := read(t, filepath.Join(removePath, "SKILL.md"))
	beforeReplace := read(t, filepath.Join(replacePath, "SKILL.md"))
	beforeBlock := read(t, filepath.Join(blockPath, "SKILL.md"))

	changes := []change{
		{target: removeTarget, skill: removeRecord.skill, old: true, remove: true},
		{target: replaceTarget, skill: replaceRecord.skill, src: replaceRecord.backupRel, old: true},
		// blockRecord's real action is "installed" (no saved tree), so in
		// genuine use this would be a remove like removeRecord above. To
		// exercise the fault deliberately, this change is instead marked
		// old: false (a plain install), even though blockPath already
		// exists non-empty from the base apply - forcing the final
		// install rename to fail with ENOTEMPTY, after the two changes
		// above have already committed.
		{target: blockTarget, skill: blockRecord.skill, src: replaceRecord.backupRel, old: false},
	}

	backupRoot := t.TempDir()
	var out bytes.Buffer
	err = installChanges(backupRoot, home, backupDir, source, nil, targets, changes, operation{kind: "restore", restoredFrom: id}, &out)
	if err == nil {
		t.Fatal("expected the forced ENOTEMPTY failure on the third change")
	}

	if got := read(t, filepath.Join(removePath, "SKILL.md")); got != beforeRemove {
		t.Fatalf("removed skill not restored by rollback: %q, want %q", got, beforeRemove)
	}
	if got := read(t, filepath.Join(replacePath, "SKILL.md")); got != beforeReplace {
		t.Fatalf("replaced skill not restored by rollback: %q, want %q", got, beforeReplace)
	}
	if got := read(t, filepath.Join(blockPath, "SKILL.md")); got != beforeBlock {
		t.Fatalf("blocked change's original directory changed: %q, want %q", got, beforeBlock)
	}
	for _, dir := range []string{removeTarget.path, replaceTarget.path, blockTarget.path} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".agentic-sdd-stage-") || strings.HasPrefix(e.Name(), ".agentic-sdd-undo-") {
				t.Fatalf("leftover staging/undo directory in %s: %s", dir, e.Name())
			}
		}
	}
}

func TestRestoreRefusesNonDirectoryTarget(t *testing.T) {
	repo, home, _, id := setupBaseBackup(t)
	claude := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.RemoveAll(claude); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claude, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Restore(repo, home, id, true, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing non-directory target") {
		t.Fatalf("expected non-directory target refusal, got %v", err)
	}
}

func TestRestoreRefusesSymlinkInInstalledTree(t *testing.T) {
	repo, home, _, id := setupBaseBackup(t)
	claude := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.Symlink(filepath.Join(claude, "SKILL.md"), filepath.Join(claude, "linked.md")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Restore(repo, home, id, true, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing special file or symlink") {
		t.Fatalf("expected installed-tree symlink refusal, got %v", err)
	}
}

func TestLoadBackupRejections(t *testing.T) {
	_, home, baseRoot, id := setupBaseBackup(t)

	t.Run("bad id syntax", func(t *testing.T) {
		if _, _, err := loadBackup(baseRoot, "not-an-id", home); err == nil || !strings.Contains(err.Error(), "invalid backup id") {
			t.Fatalf("expected invalid id error, got %v", err)
		}
	})

	t.Run("id traversal", func(t *testing.T) {
		for _, bad := range []string{"../" + id, id + "/../x", "a/b"} {
			if _, _, err := loadBackup(baseRoot, bad, home); err == nil || !strings.Contains(err.Error(), "invalid backup id") {
				t.Fatalf("id %q: expected invalid id error, got %v", bad, err)
			}
		}
	})

	t.Run("home mismatch", func(t *testing.T) {
		otherHome := t.TempDir()
		if _, _, err := loadBackup(baseRoot, id, otherHome); err == nil || !strings.Contains(err.Error(), "different home") {
			t.Fatalf("expected home mismatch error, got %v", err)
		}
	})

	run := func(name string, tamper func(dir string)) {
		t.Run(name, func(t *testing.T) {
			root := cloneBackupRoot(t, baseRoot, id)
			tamper(filepath.Join(root, id))
			_, _, err := loadBackup(root, id, home)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			t.Logf("%s: %v", name, err)
		})
	}

	run("missing manifest", func(dir string) {
		if err := os.Remove(filepath.Join(dir, "manifest.json")); err != nil {
			t.Fatal(err)
		}
	})
	run("unparseable json", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not json"), 0644); err != nil {
			t.Fatal(err)
		}
	})
	run("mismatched id", func(dir string) {
		rewriteManifest(t, dir, func(m map[string]any) { m["id"] = "20300101T000000.000000000Z" })
	})
	run("unknown top-level entry", func(dir string) {
		if err := os.MkdirAll(filepath.Join(dir, "weird"), 0755); err != nil {
			t.Fatal(err)
		}
	})
	run("non-skill entry under a client", func(dir string) {
		if err := os.MkdirAll(filepath.Join(dir, "claude", "not-a-skill"), 0755); err != nil {
			t.Fatal(err)
		}
	})
	run("symlinked client dir", func(dir string) {
		real := filepath.Join(dir, "claude")
		elsewhere := t.TempDir()
		if err := copyTree(real, filepath.Join(elsewhere, "claude")); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(real); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(elsewhere, "claude"), real); err != nil {
			t.Fatal(err)
		}
	})
	run("symlinked skill dir", func(dir string) {
		real := filepath.Join(dir, "claude", "agentic-sdd-plan")
		elsewhere := t.TempDir()
		if err := copyTree(real, filepath.Join(elsewhere, "agentic-sdd-plan")); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(real); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(elsewhere, "agentic-sdd-plan"), real); err != nil {
			t.Fatal(err)
		}
	})
	run("symlink inside a skill", func(dir string) {
		skill := filepath.Join(dir, "claude", "agentic-sdd-plan")
		if err := os.Symlink(filepath.Join(skill, "SKILL.md"), filepath.Join(skill, "linked.md")); err != nil {
			t.Fatal(err)
		}
	})
	run("missing SKILL.md", func(dir string) {
		if err := os.Remove(filepath.Join(dir, "claude", "agentic-sdd-plan", "SKILL.md")); err != nil {
			t.Fatal(err)
		}
	})
	run("entry without a saved tree", func(dir string) {
		if err := os.RemoveAll(filepath.Join(dir, "claude", "agentic-sdd-plan")); err != nil {
			t.Fatal(err)
		}
	})
	run("saved tree without an entry", func(dir string) {
		stray := filepath.Join(dir, "claude", "agentic-sdd-extra")
		if err := os.MkdirAll(stray, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stray, "SKILL.md"), []byte("orphan"), 0644); err != nil {
			t.Fatal(err)
		}
	})
	run("entry path does not match current home", func(dir string) {
		rewriteManifest(t, dir, func(m map[string]any) {
			e := entryFor(m, "claude", "agentic-sdd-plan")
			e["path"] = "/somewhere/else/agentic-sdd-plan"
		})
	})
	run("digest mismatch", func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, "claude", "agentic-sdd-plan", "SKILL.md"), []byte("tampered"), 0644); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("symlinked backup dir itself", func(t *testing.T) {
		root := t.TempDir()
		elsewhere := t.TempDir()
		if err := copyTree(filepath.Join(baseRoot, id), filepath.Join(elsewhere, id)); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(elsewhere, id), filepath.Join(root, id)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := loadBackup(root, id, home); err == nil || !strings.Contains(err.Error(), "refusing non-directory backup") {
			t.Fatalf("expected symlinked backup dir refusal, got %v", err)
		}
	})
}
