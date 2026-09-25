package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSyncBacksUpAndReplaces(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "extra.md"), []byte("reference"), 0644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(old, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "old-only.md"), []byte("keep in backup"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Sync(repo, home, false, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(old, "SKILL.md")); got != "old" {
		t.Fatalf("preview changed skill: %q", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("preview created backup: %v", err)
	}
	if err := Sync(repo, home, true, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(old, "SKILL.md")); got != "new" {
		t.Fatalf("installed %q", got)
	}
	if got := read(t, filepath.Join(old, "extra.md")); got != "reference" {
		t.Fatalf("reference %q", got)
	}
	if _, err := os.Stat(filepath.Join(old, "old-only.md")); !os.IsNotExist(err) {
		t.Fatalf("old file retained in replacement: %v", err)
	}
	for _, p := range []string{filepath.Join(home, ".agents", "skills"), filepath.Join(home, ".codex", "skills"), filepath.Join(home, ".gemini", "antigravity-cli", "skills")} {
		if got := read(t, filepath.Join(p, "agentic-sdd-plan", "SKILL.md")); got != "new" {
			t.Fatalf("%s: %q", p, got)
		}
	}
	backups, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups: %v %v", backups, err)
	}
	if got := read(t, filepath.Join(repo, "backups", backups[0].Name(), "claude", "agentic-sdd-plan", "SKILL.md")); got != "old" {
		t.Fatalf("backup %q", got)
	}
	if got := read(t, filepath.Join(repo, "backups", backups[0].Name(), "claude", "agentic-sdd-plan", "old-only.md")); got != "keep in backup" {
		t.Fatalf("old-only backup %q", got)
	}
	out.Reset()
	if err := Sync(repo, home, true, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "All skills are up to date.") {
		t.Fatalf("repeated apply did not report no changes: %s", out.String())
	}
	backups, err = os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("repeated apply created another backup: %v %v", backups, err)
	}
}

func TestSyncRefusesSymlinkedBackupDirectory(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, "backups")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Sync(repo, home, true, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing non-directory backup path") {
		t.Fatalf("expected backup symlink refusal, got %v", err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("backup wrote outside repository: %v %v", entries, err)
	}
}

func TestSyncRefusesSourceSymlinkBeforeChanges(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "SKILL.md"), filepath.Join(source, "linked.md")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Sync(repo, home, true, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing special file or symlink") {
		t.Fatalf("expected source symlink refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
	}
}

func TestSyncRefusesConflictingTargetBeforeChanges(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(filepath.Dir(conflict), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conflict, []byte("leave me"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Sync(repo, home, true, &out)
	if err == nil || !strings.Contains(err.Error(), "refusing non-directory target") {
		t.Fatalf("expected target conflict refusal, got %v", err)
	}
	if got := read(t, conflict); got != "leave me" {
		t.Fatalf("target changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
	}
}

func TestSyncRefusesSymlinkBeforeChanges(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Sync(repo, home, true, &out)
	if err == nil || !strings.Contains(err.Error(), "non-directory path component") {
		t.Fatalf("expected symlink refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
	}
}

func TestRollbackRestoresOriginalAndRemovesNewSkill(t *testing.T) {
	root := t.TempDir()
	targetRoot := filepath.Join(root, "skills")
	if err := os.MkdirAll(targetRoot, 0755); err != nil {
		t.Fatal(err)
	}
	oldName, newName := "agentic-sdd-plan", "agentic-sdd-spec"
	undo := filepath.Join(targetRoot, ".agentic-sdd-undo-test")
	for _, dir := range []string{undo, filepath.Join(targetRoot, oldName), filepath.Join(targetRoot, newName)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(undo, "SKILL.md"), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	changes := []change{
		{target: target{path: targetRoot}, skill: oldName, old: true, undo: undo},
		{target: target{path: targetRoot}, skill: newName},
	}
	if err := rollback(changes, []int{0, 1}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(targetRoot, oldName, "SKILL.md")); got != "original" {
		t.Fatalf("restored %q", got)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, newName)); !os.IsNotExist(err) {
		t.Fatalf("new skill survived rollback: %v", err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInstallChangesRollsBackRemoveOnLaterFailure(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	targetPath := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}
	removeName, failName := "agentic-sdd-remove-me", "agentic-sdd-fail"
	removeDir := filepath.Join(targetPath, removeName)
	if err := os.MkdirAll(removeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(removeDir, "SKILL.md"), []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	// failDir already exists and is non-empty, so the rename that would
	// install the staged replacement over it fails partway through the
	// install loop, after the remove change above has already been
	// committed (renamed to its undo directory).
	failDir := filepath.Join(targetPath, failName)
	if err := os.MkdirAll(failDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(failDir, "marker.md")
	if err := os.WriteFile(marker, []byte("pre-existing"), 0644); err != nil {
		t.Fatal(err)
	}
	source := fstest.MapFS{
		failName + "/SKILL.md": &fstest.MapFile{Data: []byte("new"), Mode: 0644},
	}
	tgt := target{name: "claude", path: targetPath}
	changes := []change{
		{target: tgt, skill: removeName, old: true, remove: true},
		{target: tgt, skill: failName, src: failName, old: false},
	}
	backupRoot := filepath.Join(root, "backups")
	var out bytes.Buffer
	err := installChanges(backupRoot, home, "test-source", source, []string{failName}, []target{tgt}, changes, operation{kind: "restore", restoredFrom: "20260101T000000.000000000Z"}, &out)
	if err == nil {
		t.Fatal("expected a failure from renaming over a non-empty directory")
	}
	if got := read(t, filepath.Join(removeDir, "SKILL.md")); got != "keep me" {
		t.Fatalf("removed skill not restored by rollback: %q", got)
	}
	if got := read(t, marker); got != "pre-existing" {
		t.Fatalf("failed change's original directory was modified: %q", got)
	}
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".agentic-sdd-stage-") || strings.HasPrefix(e.Name(), ".agentic-sdd-undo-") {
			t.Fatalf("leftover staging/undo directory after rollback: %s", e.Name())
		}
	}
}
