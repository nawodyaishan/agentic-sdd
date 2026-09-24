package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := syncSkills(repo, home, false, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(old, "SKILL.md")); got != "old" {
		t.Fatalf("preview changed skill: %q", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("preview created backup: %v", err)
	}
	if err := syncSkills(repo, home, true, &out); err != nil {
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
	err := syncSkills(repo, home, true, &out)
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
	err := syncSkills(repo, home, true, &out)
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
	err := syncSkills(repo, home, true, &out)
	if err == nil || !strings.Contains(err.Error(), "non-directory path component") {
		t.Fatalf("expected symlink refusal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("backup created: %v", err)
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
