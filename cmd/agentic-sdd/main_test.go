package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/version"
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
		{"help", "--help", "Usage: asdx-core", 0},
		{"help command", "help apply", "Commands:", 0},
		{"command help", "preview --help", "Usage: asdx-core", 0},
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
			if !strings.Contains(stdout.String(), "asdx-core dev") {
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
	if code != 2 || !strings.Contains(errOut.String(), "asdx-core backups") {
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

func TestApplyLogAndLogsCommand(t *testing.T) {
	root := t.TempDir()
	repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
	skill := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"preview", "--repo", repo, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("preview exited %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(logsDir(home)); !os.IsNotExist(err) {
		t.Fatalf("preview created logs: %v", err)
	}
	stdout.Reset()
	if code := runNonInteractive([]string{"apply", "--repo", repo, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("apply exited %d: %s", code, stderr.String())
	}
	entries, err := os.ReadDir(logsDir(home))
	if err != nil || len(entries) != 1 {
		t.Fatalf("apply logs: %v, %v", entries, err)
	}
	logPath := filepath.Join(logsDir(home), entries[0].Name())
	info, err := os.Stat(logPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("log permissions: %v, %v", info, err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil || !strings.Contains(string(data), "install codex/agentic-sdd-plan") || !strings.Contains(string(data), "Result: success") {
		t.Fatalf("log content: %q, %v", data, err)
	}
	stdout.Reset()
	if code := runNonInteractive([]string{"logs", "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("logs exited %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), string(data)) {
		t.Fatalf("logs output missing latest log: %q", stdout.String())
	}
}

func TestApplyFailureLog(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"apply", "--repo", filepath.Join(root, "missing"), "--home", home}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "error:") {
		t.Fatalf("apply exited %d, stderr %q", code, stderr.String())
	}
	entries, err := os.ReadDir(logsDir(home))
	if err != nil || len(entries) != 1 {
		t.Fatalf("failure logs: %v, %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(logsDir(home), entries[0].Name()))
	if err != nil || !strings.Contains(string(data), "Result: failure") {
		t.Fatalf("failure log: %q, %v", data, err)
	}
}

func TestRestoreApplyLog(t *testing.T) {
	repo, home, id := setupBackupCLI(t)
	var stdout, stderr bytes.Buffer
	if code := runNonInteractive([]string{"restore", "apply", id, "--repo", repo, "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("restore exited %d: %s", code, stderr.String())
	}
	stdout.Reset()
	if code := runNonInteractive([]string{"logs", "--home", home}, &stdout, &stderr); code != 0 {
		t.Fatalf("logs exited %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Command: restore "+id) || !strings.Contains(stdout.String(), "Result: success") {
		t.Fatalf("latest log does not describe restore: %q", stdout.String())
	}
}

func TestApplyRefusesSymlinkedLogDirectory(t *testing.T) {
	root := t.TempDir()
	home, outside := filepath.Join(root, "home"), filepath.Join(root, "outside")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, ".agentic-sdd")); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"apply", "--home", home}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "refusing non-directory log path") {
		t.Fatalf("apply exited %d, stderr %q", code, stderr.String())
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("log wrote through symlink: %v, %v", entries, err)
	}
}

func TestRunOpenCodeFlagsAndHelp(t *testing.T) {
	t.Run("help orchestrate prints options", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"help", "orchestrate"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("help orchestrate exited %d, stderr %q", code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, "--with-opencode") || !strings.Contains(got, "--opencode-free-first") {
			t.Fatalf("help orchestrate output missing expected options:\n%s", got)
		}
	})

	t.Run("opencode-free-first alone exits 2", func(t *testing.T) {
		home := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"preview", "--opencode-free-first", "--home", home}, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("preview exited %d, want 2; stderr %q", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "--opencode-free-first requires --with-opencode") {
			t.Fatalf("stderr missing requirement message: %q", stderr.String())
		}

		stderr.Reset()
		code = runNonInteractive([]string{"apply", "--opencode-free-first", "--home", home}, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("apply exited %d, want 2; stderr %q", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "--opencode-free-first requires --with-opencode") {
			t.Fatalf("stderr missing requirement message: %q", stderr.String())
		}
	})

	t.Run("preview with-opencode and apply with-opencode", func(t *testing.T) {
		home := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"preview", "--with-opencode", "--home", home}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("preview exited %d: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "install opencode-agents/sdd-orchestrator.md") {
			t.Fatalf("preview output missing opencode-agents: %s", stdout.String())
		}
		if _, err := os.Stat(filepath.Join(home, ".config", "opencode")); !os.IsNotExist(err) {
			t.Fatalf("preview created .config/opencode: %v", err)
		}

		stdout.Reset()
		code = runNonInteractive([]string{"apply", "--with-opencode", "--home", home}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("apply exited %d: %s", code, stderr.String())
		}
		if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "agents", "sdd-orchestrator.md")); err != nil {
			t.Fatalf("missing installed agent file: %v", err)
		}
		entries, err := os.ReadDir(logsDir(home))
		if err != nil || len(entries) != 1 {
			t.Fatalf("apply logs: %v, %v", entries, err)
		}
		logData, err := os.ReadFile(filepath.Join(logsDir(home), entries[0].Name()))
		if err != nil || !strings.Contains(string(logData), "Command: apply --with-opencode") {
			t.Fatalf("log content missing command: %q, %v", logData, err)
		}
	})

	t.Run("opencode-harness denial in default build", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"opencode-harness"}, &stdout, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "unavailable in this build") {
			t.Fatalf("opencode-harness exited %d, stderr %q", code, stderr.String())
		}
		stdout.Reset()
		stderr.Reset()
		code = runNonInteractive([]string{"help", "opencode-harness"}, &stdout, &stderr)
		if code != 0 || !strings.Contains(stdout.String(), "unavailable in this build") {
			t.Fatalf("help opencode-harness exited %d, stdout %q", code, stdout.String())
		}
	})
}

// TestPlainOutputByteIdentity (Check C2) pins the exact non-interactive
// output and exit codes of the commands and error paths listed in the plan.
// Any change to plain output, including an accidental terminal presentation
// leak, fails this test. The unknown-command case compares against a captured
// printUsage transcript so the two additive "guide" lines planned for the
// usage block are accounted for without rewording anything here.
func TestPlainOutputByteIdentity(t *testing.T) {
	newFixture := func(t *testing.T) (repo, home string) {
		t.Helper()
		root := t.TempDir()
		repo, home = filepath.Join(root, "repo"), filepath.Join(root, "home")
		source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
		if err := os.MkdirAll(source, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(home, 0755); err != nil {
			t.Fatal(err)
		}
		return repo, home
	}
	invoke := func(t *testing.T, args ...string) (code int, stdout, stderr string) {
		t.Helper()
		var out, errOut bytes.Buffer
		code = runNonInteractive(args, &out, &errOut)
		return code, out.String(), errOut.String()
	}
	check := func(t *testing.T, name string, code, wantCode int, stdout, stderr, wantStdout, wantStderr string) {
		t.Helper()
		if code != wantCode {
			t.Fatalf("%s: exit %d, want %d; stdout %q; stderr %q", name, code, wantCode, stdout, stderr)
		}
		if stdout != wantStdout {
			t.Fatalf("%s: stdout = %q, want %q", name, stdout, wantStdout)
		}
		if stderr != wantStderr {
			t.Fatalf("%s: stderr = %q, want %q", name, stderr, wantStderr)
		}
	}

	t.Run("bare preview one-skill fixture", func(t *testing.T) {
		repo, home := newFixture(t)
		code, stdout, stderr := invoke(t, "--repo", repo, "--home", home)
		want := "install agents/agentic-sdd-plan\n" +
			"install codex/agentic-sdd-plan\n" +
			"install claude/agentic-sdd-plan\n" +
			"install agy/agentic-sdd-plan\n" +
			"Preview only. Run with apply (or --apply) to install.\n"
		check(t, "bare preview", code, 0, stdout, stderr, want, "")
	})

	t.Run("version", func(t *testing.T) {
		code, stdout, stderr := invoke(t, "version")
		want := fmt.Sprintf("asdx-core %s\n  commit:     %s\n  date:       %s\n  go version: %s\n",
			version.Version, version.Commit, version.Date, version.GoVersion)
		check(t, "version", code, 0, stdout, stderr, want, "")
	})

	t.Run("version extra", func(t *testing.T) {
		code, stdout, stderr := invoke(t, "version", "extra")
		check(t, "version extra", code, 2, stdout, stderr, "", "error: usage: asdx-core version\n")
	})

	t.Run("restore without id non-interactive", func(t *testing.T) {
		repo, home := newFixture(t)
		code, stdout, stderr := invoke(t, "restore", "--repo", repo, "--home", home)
		check(t, "restore without id", code, 2, stdout, stderr, "",
			"error: restore requires a backup id; run \"asdx-core backups\" to list them\n")
	})

	t.Run("restore invalid id", func(t *testing.T) {
		repo, home := newFixture(t)
		code, stdout, stderr := invoke(t, "restore", "notanid", "--repo", repo, "--home", home)
		check(t, "restore invalid id", code, 2, stdout, stderr, "", "error: invalid backup id \"notanid\"\n")
	})

	t.Run("opencode-free-first requires with-opencode", func(t *testing.T) {
		_, home := newFixture(t)
		code, stdout, stderr := invoke(t, "preview", "--opencode-free-first", "--home", home)
		check(t, "opencode-free-first alone", code, 2, stdout, stderr, "",
			"error: --opencode-free-first requires --with-opencode\n")
	})

	t.Run("logs without a log directory", func(t *testing.T) {
		_, home := newFixture(t)
		code, stdout, stderr := invoke(t, "logs", "--home", home)
		if code != 1 {
			t.Fatalf("logs: exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
		}
		if stdout != "" {
			t.Fatalf("logs: stdout = %q, want empty", stdout)
		}
		if !strings.HasPrefix(stderr, "error: open ") || !strings.Contains(stderr, "/.agentic-sdd/logs: no such file or directory") {
			t.Fatalf("logs: stderr = %q, want missing log directory error", stderr)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		var usageBuf bytes.Buffer
		printUsage(&usageBuf)
		code, stdout, stderr := invoke(t, "wat")
		check(t, "unknown command", code, 2, stdout, stderr, "",
			"error: unknown command \"wat\"\n"+usageBuf.String())
	})
}

func TestTopicHelpCanonicalAndMigration(t *testing.T) {
	topics := []string{"preview", "apply", "backups", "logs", "restore", "guide", "version", "tui", "run", "task", "review"}
	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			var out bytes.Buffer
			if !printTopicHelp(&out, topic) {
				t.Fatalf("printTopicHelp(%q) returned false", topic)
			}
			got := out.String()
			for _, want := range []string{"Writes files:", "Example: asdx-core", "Related: asdx-core"} {
				if !strings.Contains(got, want) {
					t.Fatalf("topic %s missing %q:\n%s", topic, want, got)
				}
			}
			if strings.Contains(got, "agentic-sdd "+topic) {
				t.Fatalf("topic %s still uses legacy command in example:\n%s", topic, got)
			}
		})
	}
	for _, topic := range []string{"preview", "guide", "version", "tui"} {
		var out bytes.Buffer
		printTopicHelp(&out, topic)
		got := out.String()
		for _, want := range []string{"asdx", "agentic-sdd", "agentic-sdd-tui"} {
			if !strings.Contains(got, want) {
				t.Fatalf("user-facing topic %s missing migration term %q:\n%s", topic, want, got)
			}
		}
	}
}
