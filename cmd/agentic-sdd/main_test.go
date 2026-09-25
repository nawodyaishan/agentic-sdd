package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runNonInteractive calls run with empty, non-interactive stdin, for every
// test that isn't specifically exercising the interactive restore prompt.
func runNonInteractive(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, strings.NewReader(""), false)
}

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
			code := runNonInteractive(strings.Fields(tc.args), &stdout, &stderr)
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
		code := runNonInteractive(args, &stdout, &stderr)
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

func TestRunVersion(t *testing.T) {
	for _, args := range []string{"version", "--version"} {
		t.Run(args, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(strings.Fields(args), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code %d; stderr: %s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), "agentic-sdd dev") {
				t.Fatalf("output %q missing default version", stdout.String())
			}
		})
	}
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"version", "extra"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code %d, want 2; stderr: %s", code, stderr.String())
	}
}

func TestRunPreviewAndApplyFromEmbeddedSource(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"preview", "--home", home}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("preview exited %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "install claude/agentic-sdd-plan") {
		t.Fatalf("preview output %q missing embedded skill install", stdout.String())
	}
	stdout.Reset()
	code = runNonInteractive([]string{"apply", "--home", home}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply exited %d: %s", code, stderr.String())
	}
	installed := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("expected %s to exist: %v", installed, err)
	}
	backups, err := os.ReadDir(filepath.Join(home, ".agentic-sdd", "backups"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup under .agentic-sdd/backups, got %v, %v", backups, err)
	}
}

func TestRunOperationalFailureUsesExitOne(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"preview", "--repo", filepath.Join(root, "missing"), "--home", filepath.Join(root, "home")}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "error:") {
		t.Fatalf("operational failure exited %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

// setupBackupCLI runs one apply (replacing claude's pre-existing skill and
// installing it fresh under the other three clients) through the CLI
// itself, returning repo, home and the resulting backup's ID for restore
// command tests to build on.
func setupBackupCLI(t *testing.T) (repo, home, id string) {
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
	if code := runNonInteractive([]string{"apply", "--repo", repo, "--home", home}, &out, &out); code != 0 {
		t.Fatalf("setup apply failed: %d: %s", code, out.String())
	}
	entries, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one backup, got %v %v", entries, err)
	}
	return repo, home, entries[0].Name()
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func normalizeBackupLine(s string) string {
	idx := strings.LastIndex(s, "; backup: ")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func TestRunBackupsEqualsRestoreList(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		root := t.TempDir()
		repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
		var out1, out2, errOut bytes.Buffer
		if code := runNonInteractive([]string{"backups", "--repo", repo, "--home", home}, &out1, &errOut); code != 0 {
			t.Fatalf("backups exit %d: %s", code, errOut.String())
		}
		if code := runNonInteractive([]string{"restore", "list", "--repo", repo, "--home", home}, &out2, &errOut); code != 0 {
			t.Fatalf("restore list exit %d: %s", code, errOut.String())
		}
		if out1.String() != out2.String() {
			t.Fatalf("backups %q != restore list %q", out1.String(), out2.String())
		}
		if !strings.Contains(out1.String(), "No backups found.") {
			t.Fatalf("missing empty message: %s", out1.String())
		}
	})
	t.Run("populated", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out1, out2, errOut bytes.Buffer
		if code := runNonInteractive([]string{"backups", "--repo", repo, "--home", home}, &out1, &errOut); code != 0 {
			t.Fatalf("backups exit %d: %s", code, errOut.String())
		}
		if code := runNonInteractive([]string{"restore", "list", "--repo", repo, "--home", home}, &out2, &errOut); code != 0 {
			t.Fatalf("restore list exit %d: %s", code, errOut.String())
		}
		if out1.String() != out2.String() {
			t.Fatalf("backups %q != restore list %q", out1.String(), out2.String())
		}
		if !strings.Contains(out1.String(), "replaced") || !strings.Contains(out1.String(), "installed") {
			t.Fatalf("missing summary content: %s", out1.String())
		}
	})
}

func TestRunRestorePreviewEqualsBareID(t *testing.T) {
	repo, home, id := setupBackupCLI(t)
	var out1, out2, errOut bytes.Buffer
	if code := runNonInteractive([]string{"restore", id, "--repo", repo, "--home", home}, &out1, &errOut); code != 0 {
		t.Fatalf("restore ID exit %d: %s", code, errOut.String())
	}
	if code := runNonInteractive([]string{"restore", "preview", id, "--repo", repo, "--home", home}, &out2, &errOut); code != 0 {
		t.Fatalf("restore preview ID exit %d: %s", code, errOut.String())
	}
	if out1.String() != out2.String() {
		t.Fatalf("restore ID %q != restore preview ID %q", out1.String(), out2.String())
	}
	if !strings.Contains(out1.String(), "Preview only") {
		t.Fatalf("missing preview trailer: %s", out1.String())
	}
}

func TestRunRestoreApplyFormsEquivalent(t *testing.T) {
	forms := [][]string{
		{"restore", "ID", "--apply"},
		{"restore", "--apply", "ID"},
		{"restore", "apply", "ID"},
	}
	var normalized []string
	for _, form := range forms {
		repo, home, id := setupBackupCLI(t)
		args := append([]string(nil), form...)
		for i, a := range args {
			if a == "ID" {
				args[i] = id
			}
		}
		args = append(args, "--repo", repo, "--home", home)
		var out, errOut bytes.Buffer
		if code := runNonInteractive(args, &out, &errOut); code != 0 {
			t.Fatalf("%v exit %d: %s", form, code, errOut.String())
		}
		if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "old" {
			t.Fatalf("%v: expected restore to revert content, got %q", form, got)
		}
		if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")); !os.IsNotExist(err) {
			t.Fatalf("%v: fresh install not removed", form)
		}
		backups, err := os.ReadDir(filepath.Join(repo, "backups"))
		if err != nil || len(backups) != 2 {
			t.Fatalf("%v: expected 2 backups, got %v %v", form, backups, err)
		}
		normalized = append(normalized, normalizeBackupLine(out.String()))
	}
	for i := 1; i < len(normalized); i++ {
		if normalized[i] != normalized[0] {
			t.Fatalf("form %v output %q != form %v output %q", forms[i], normalized[i], forms[0], normalized[0])
		}
	}
}

func TestRunRestoreUsageErrors(t *testing.T) {
	repo, home, id := setupBackupCLI(t)
	tests := []struct {
		name string
		args []string
		want string
		code int
	}{
		{"apply flag with preview command", []string{"restore", "preview", id, "--apply"}, "--apply cannot be used", 2},
		{"apply flag with apply command", []string{"restore", "apply", id, "--apply"}, "--apply cannot be used", 2},
		{"stray extra id", []string{"restore", id, id}, "unexpected argument", 2},
		{"bad flag", []string{"restore", "--nope"}, "flag provided but not defined", 2},
		{"bad id syntax", []string{"restore", "not-an-id"}, "invalid backup id", 2},
		{"bad id syntax via preview", []string{"restore", "preview", "../etc"}, "invalid backup id", 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append(append([]string(nil), tc.args...), "--repo", repo, "--home", home)
			var out, errOut bytes.Buffer
			code := runNonInteractive(args, &out, &errOut)
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stderr %s", code, tc.code, errOut.String())
			}
			if !strings.Contains(errOut.String(), tc.want) {
				t.Fatalf("stderr %q missing %q", errOut.String(), tc.want)
			}
		})
	}
}

func TestRunRestoreStateErrorsExitOne(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := runNonInteractive([]string{"restore", "20260101T000000.000000000Z", "--repo", repo, "--home", home}, &out, &errOut)
		if code != 1 {
			t.Fatalf("exit %d, want 1; stderr %s", code, errOut.String())
		}
	})
	t.Run("digest mismatch", func(t *testing.T) {
		repo, home, id := setupBackupCLI(t)
		skillFile := filepath.Join(repo, "backups", id, "claude", "agentic-sdd-plan", "SKILL.md")
		if err := os.WriteFile(skillFile, []byte("tampered"), 0644); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		code := runNonInteractive([]string{"restore", id, "--repo", repo, "--home", home}, &out, &errOut)
		if code != 1 {
			t.Fatalf("exit %d, want 1; stderr %s", code, errOut.String())
		}
	})
	t.Run("home mismatch", func(t *testing.T) {
		repo, home, id := setupBackupCLI(t)
		otherHome := filepath.Join(filepath.Dir(home), "other-home")
		var out, errOut bytes.Buffer
		code := runNonInteractive([]string{"restore", id, "--repo", repo, "--home", otherHome}, &out, &errOut)
		if code != 1 {
			t.Fatalf("exit %d, want 1; stderr %s", code, errOut.String())
		}
	})
}

func TestRunRestoreNonInteractiveMissingID(t *testing.T) {
	repo, home, _ := setupBackupCLI(t)
	var out, errOut bytes.Buffer
	code := runNonInteractive([]string{"restore", "--repo", repo, "--home", home}, &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "agentic-sdd backups") {
		t.Fatalf("exit %d, stderr %s", code, errOut.String())
	}
}

func TestRunRestoreInteractiveFlow(t *testing.T) {
	t.Run("select then preview", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader("1\n"), true)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Select a backup") || !strings.Contains(out.String(), "Preview only") {
			t.Fatalf("output missing prompt/preview: %s", out.String())
		}
		if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "new" {
			t.Fatalf("interactive preview changed content: %q", got)
		}
	})
	t.Run("select then y applies", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--apply", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader("1\ny\n"), true)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "old" {
			t.Fatalf("expected restore to apply: %q", got)
		}
	})
	t.Run("select then n cancels", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--apply", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader("1\nn\n"), true)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "new" {
			t.Fatalf("cancel changed content: %q", got)
		}
		if !strings.Contains(out.String(), "Cancelled.") {
			t.Fatalf("missing cancelled message: %s", out.String())
		}
	})
	t.Run("blank cancels", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader("\n"), true)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "Cancelled.") {
			t.Fatalf("missing cancelled message: %s", out.String())
		}
	})
	t.Run("bad number", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader("42\n"), true)
		if code != 2 {
			t.Fatalf("exit %d, want 2; stderr %s", code, errOut.String())
		}
	})
	t.Run("empty backup list", func(t *testing.T) {
		root := t.TempDir()
		repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
		var out, errOut bytes.Buffer
		code := run([]string{"restore", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), true)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "No backups found.") {
			t.Fatalf("missing empty message: %s", out.String())
		}
	})
}
