package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpAndUsageErrors(t *testing.T) {
	tests := []struct {
		name, args, want string
		code             int
	}{
		{"help", "--help", "Usage: agentic-sdd", 0},
		{"help command", "help apply", "Commands:", 0},
		{"command help", "preview --help", "Usage: agentic-sdd", 0},
		{"unknown command", "wat", "unknown command", 2},
		{"extra argument", "preview extra", "unexpected argument", 2},
		{"unknown flag", "--wat", "flag provided but not defined", 2},
		{"ambiguous apply", "preview --apply", "--apply cannot be used", 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(strings.Fields(tc.args), &stdout, &stderr)
			if code != tc.code {
				t.Fatalf("exit code %d, want %d; stderr: %s", code, tc.code, stderr.String())
			}
			got := stdout.String() + stderr.String()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("output %q does not contain %q", got, tc.want)
			}
		})
	}
}

func TestRunPreviewAndApplyCompatibility(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(old), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	invoke := func(args ...string) (int, string, string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}
	base := []string{"--repo", repo, "--home", home}
	var previews []string
	for _, args := range [][]string{base, append([]string{"preview"}, base...)} {
		code, output, stderr := invoke(args...)
		if code != 0 {
			t.Fatalf("preview exited %d: %s", code, stderr)
		}
		previews = append(previews, output)
		if data, err := os.ReadFile(old); err != nil || string(data) != "old" {
			t.Fatalf("preview modified target: %q, %v", data, err)
		}
		if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
			t.Fatalf("preview created backup: %v", err)
		}
	}
	if previews[0] != previews[1] {
		t.Fatalf("default preview %q differs from explicit preview %q", previews[0], previews[1])
	}
	code, _, stderr := invoke(append([]string{"apply"}, base...)...)
	if code != 0 {
		t.Fatalf("apply exited %d: %s", code, stderr)
	}
	if data, err := os.ReadFile(old); err != nil || string(data) != "new" {
		t.Fatalf("apply target: %q, %v", data, err)
	}
	code, output, stderr := invoke(append(append([]string{}, base...), "--apply")...)
	if code != 0 || !strings.Contains(output, "All skills are up to date.") {
		t.Fatalf("legacy apply exited %d, stdout %q, stderr %q", code, output, stderr)
	}
	backups, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup, got %v, %v", backups, err)
	}
	legacyHome := filepath.Join(root, "legacy-home")
	code, _, stderr = invoke("--repo", repo, "--home", legacyHome, "--apply")
	if code != 0 {
		t.Fatalf("legacy apply exited %d: %s", code, stderr)
	}
	legacySkill := filepath.Join(legacyHome, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")
	if data, err := os.ReadFile(legacySkill); err != nil || string(data) != "new" {
		t.Fatalf("legacy apply target: %q, %v", data, err)
	}
}

func TestRunOperationalFailureUsesExitOne(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"preview", "--repo", filepath.Join(root, "missing"), "--home", filepath.Join(root, "home")}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "error:") {
		t.Fatalf("operational failure exited %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}
