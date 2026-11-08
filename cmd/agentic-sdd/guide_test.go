package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// spyReader records whether the guide ever reads stdin without a terminal.
type spyReader struct {
	reads int
}

func (s *spyReader) Read(p []byte) (int, error) {
	s.reads++
	return 0, io.EOF
}

// guideFixture builds the one-skill repository the guide checks install from,
// leaving the home path uncreated so a read-only or declined flow can be
// proven to write nothing.
func guideFixture(t *testing.T) (repo, home string) {
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
	return repo, home
}

// runGuideInput drives one guide run through the runEnv seam with an
// interactive, terminal-like environment and scripted stdin answers.
func runGuideInput(t *testing.T, repo, home, stdin string, stderrTTY bool) (int, string, string) {
	t.Helper()
	args := []string{"guide"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	args = append(args, "--home", home)
	var stdout, stderr bytes.Buffer
	env := &Env{Interactive: true, StdoutTTY: true, StderrTTY: stderrTTY, Width: 80}
	code := runEnv(args, &stdout, &stderr, strings.NewReader(stdin), env)
	return code, stdout.String(), stderr.String()
}

// assertNoWrites accepts a home path that does not exist yet or exists empty.
func assertNoWrites(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(home)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("guide wrote into %s: %v", home, names)
	}
}

// TestGuideNonInteractiveAndFlags is Check C6 (AC-07): guide without a
// terminal exits 2 without reading stdin or writing anything, and its flags
// are limited to --repo and --home.
func TestGuideNonInteractiveAndFlags(t *testing.T) {
	t.Run("non-interactive reads and writes nothing", func(t *testing.T) {
		root := t.TempDir()
		repo, home := filepath.Join(root, "repo"), filepath.Join(root, "home")
		spy := &spyReader{}
		var stdout, stderr bytes.Buffer
		env := &Env{Width: 80, StderrTTY: true}
		code := runEnv([]string{"guide", "--repo", repo, "--home", home}, &stdout, &stderr, spy, env)
		if code != 2 {
			t.Fatalf("guide non-interactive exit %d, want 2; stderr: %s", code, stderr.String())
		}
		if spy.reads != 0 {
			t.Fatalf("guide read stdin %d times without a terminal", spy.reads)
		}
		if stdout.Len() != 0 {
			t.Fatalf("guide printed to stdout without a terminal: %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "guide requires an interactive terminal") {
			t.Fatalf("missing non-interactive message: %q", stderr.String())
		}
		if !strings.Contains(stderr.String(), "hint:") {
			t.Fatalf("missing terminal hint: %q", stderr.String())
		}
		assertNoWrites(t, home)
		if _, err := os.Stat(repo); !os.IsNotExist(err) {
			t.Fatalf("guide touched the repository: %v", err)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := &Env{Interactive: true, StderrTTY: true, Width: 80}
		code := runEnv([]string{"guide", "--wat"}, &stdout, &stderr, strings.NewReader(""), env)
		if code != 2 {
			t.Fatalf("guide --wat exit %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "error:") || !strings.Contains(stderr.String(), "hint:") {
			t.Fatalf("missing error or hint: %q", stderr.String())
		}
	})

	t.Run("unexpected argument", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := &Env{Interactive: true, Width: 80}
		code := runEnv([]string{"guide", "extra"}, &stdout, &stderr, strings.NewReader(""), env)
		if code != 2 || !strings.Contains(stderr.String(), "unexpected argument") {
			t.Fatalf("guide extra exit %d, stderr %q", code, stderr.String())
		}
	})

	t.Run("help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		env := &Env{Width: 80}
		code := runEnv([]string{"guide", "--help"}, &stdout, &stderr, strings.NewReader(""), env)
		if code != 0 {
			t.Fatalf("guide --help exit %d, want 0", code)
		}
		if !strings.Contains(stdout.String(), "guide — interactive step-by-step wizard") {
			t.Fatalf("missing guide topic help: %q", stdout.String())
		}
		if !strings.Contains(stdout.String(), "Usage: agentic-sdd") {
			t.Fatalf("missing global usage: %q", stdout.String())
		}
	})
}

// TestGuideInstallAndDeclineAndRepeat is Check C7 (AC-08): the guided install
// matches apply while printing only the concise summary, a decline writes
// nothing, and a repeat is a clean no-op.
func TestGuideInstallAndDeclineAndRepeat(t *testing.T) {
	t.Run("install matches apply and stays concise", func(t *testing.T) {
		root := t.TempDir()
		repo := filepath.Join(root, "repo")
		source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
		if err := os.MkdirAll(source, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}
		homeGuide := filepath.Join(root, "home-guide")
		homeApply := filepath.Join(root, "home-apply")

		code, stdout, stderr := runGuideInput(t, repo, homeGuide, "1\nn\ny\n", false)
		if code != 0 {
			t.Fatalf("guide install exit %d: stdout %q stderr %q", code, stdout, stderr)
		}
		for _, want := range []string{
			"Preview — nothing written yet",
			"Installed 4 items",
			"Backup: ",
			"Log: ",
			"Undo any time: agentic-sdd restore",
			"agentic-sdd-router",
		} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("guide install output missing %q:\n%s", want, stdout)
			}
		}
		for _, raw := range []string{"install claude/agentic-sdd-plan", "install agents/agentic-sdd-plan", "install codex/agentic-sdd-plan", "install agy/agentic-sdd-plan"} {
			if strings.Contains(stdout, raw) {
				t.Fatalf("guide dumped a raw per-skill line %q:\n%s", raw, stdout)
			}
		}

		backups, err := skillsync.ListBackups(repo, homeGuide)
		if err != nil || len(backups) != 1 {
			t.Fatalf("expected one backup after guide install, got %v, %v", backups, err)
		}
		logs, err := os.ReadDir(logsDir(homeGuide))
		if err != nil || len(logs) != 1 {
			t.Fatalf("expected one apply log after guide install, got %v, %v", logs, err)
		}

		var applyOut bytes.Buffer
		if code := runNonInteractive([]string{"apply", "--repo", repo, "--home", homeApply}, &applyOut, &applyOut); code != 0 {
			t.Fatalf("reference apply exit %d: %s", code, applyOut.String())
		}
		for _, dir := range []string{".agents/skills", ".codex/skills", ".claude/skills", ".gemini/antigravity-cli/skills"} {
			want := read(t, filepath.Join(homeApply, dir, "agentic-sdd-plan", "SKILL.md"))
			got := read(t, filepath.Join(homeGuide, dir, "agentic-sdd-plan", "SKILL.md"))
			if got != want {
				t.Fatalf("%s skill content = %q, want apply's %q", dir, got, want)
			}
		}
	})

	t.Run("repeat is a no-op", func(t *testing.T) {
		repo, home := guideFixture(t)
		if code, _, stderr := runGuideInput(t, repo, home, "1\nn\ny\n", false); code != 0 {
			t.Fatalf("first guide install exit %d: %s", code, stderr)
		}
		before, err := skillsync.ListBackups(repo, home)
		if err != nil || len(before) != 1 {
			t.Fatalf("expected one backup before repeat, got %v, %v", before, err)
		}
		logsBefore, err := os.ReadDir(logsDir(home))
		if err != nil || len(logsBefore) != 1 {
			t.Fatalf("expected one log before repeat, got %v, %v", logsBefore, err)
		}

		code, stdout, stderr := runGuideInput(t, repo, home, "1\nn\n", false)
		if code != 0 {
			t.Fatalf("repeat guide exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "All skills are up to date") {
			t.Fatalf("repeat did not report up-to-date: %s", stdout)
		}
		if strings.Contains(stdout, "Install these ") {
			t.Fatalf("repeat asked to confirm an empty install: %s", stdout)
		}
		after, err := skillsync.ListBackups(repo, home)
		if err != nil || len(after) != 1 {
			t.Fatalf("repeat created a backup: %v, %v", after, err)
		}
		logsAfter, err := os.ReadDir(logsDir(home))
		if err != nil || len(logsAfter) != 1 {
			t.Fatalf("repeat created a log: %v, %v", logsAfter, err)
		}
	})

	t.Run("decline writes nothing", func(t *testing.T) {
		repo, home := guideFixture(t)
		code, stdout, stderr := runGuideInput(t, repo, home, "1\nn\nn\n", false)
		if code != 0 {
			t.Fatalf("decline exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "Cancelled.") {
			t.Fatalf("decline did not print Cancelled.: %s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("opencode pack asks free-first then declines", func(t *testing.T) {
		// The one-skill fixture has no opencode-pack directory, so this flow
		// uses the embedded skill set and the embedded pack.
		home := filepath.Join(t.TempDir(), "home")
		code, stdout, stderr := runGuideInput(t, "", home, "1\ny\ny\nn\n", false)
		if code != 0 {
			t.Fatalf("pack decline exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "Prefer Free models first") {
			t.Fatalf("free-first prompt was not asked or shown: %s", stdout)
		}
		if !strings.Contains(stdout, "Cancelled.") {
			t.Fatalf("pack decline did not print Cancelled.: %s", stdout)
		}
		assertNoWrites(t, home)
		backups, err := skillsync.ListBackups("", home)
		if err != nil || len(backups) != 0 {
			t.Fatalf("pack decline created backups: %v, %v", backups, err)
		}
	})
}

// TestGuideLogsRestoreSharedReaderAndEOF is Check C8 (AC-09): guide delegates
// to logs and restore through one shared *bufio.Reader, and end-of-input at
// any prompt cancels with nothing written.
func TestGuideLogsRestoreSharedReaderAndEOF(t *testing.T) {
	t.Run("logs then restore through one shared reader", func(t *testing.T) {
		repo, home, _ := setupBackupCLI(t)
		code, stdout, stderr := runGuideInput(t, repo, home, "3\n2\n1\ny\n", false)
		if code != 0 {
			t.Fatalf("guide logs/restore exit %d: stdout %q stderr %q", code, stdout, stderr)
		}
		for _, want := range []string{"Log:", "Restore these skills? [y/N]:", "Restored"} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("guide logs/restore output missing %q:\n%s", want, stdout)
			}
		}
		if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "old" {
			t.Fatalf("guided restore did not apply: %q", got)
		}
	})

	t.Run("about returns to the menu then quits", func(t *testing.T) {
		repo, home := guideFixture(t)
		code, stdout, stderr := runGuideInput(t, repo, home, "4\nq\n", false)
		if code != 0 {
			t.Fatalf("about/quit exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "Agentic SDD installs reusable engineering skills") {
			t.Fatalf("missing about text: %s", stdout)
		}
		if !strings.Contains(stdout, "Cancelled.") {
			t.Fatalf("quit did not print Cancelled.: %s", stdout)
		}
		if strings.Count(stdout, "What would you like to do?") != 2 {
			t.Fatalf("about did not return to the menu:\n%s", stdout)
		}
		assertNoWrites(t, home)
	})

	t.Run("three invalid menu answers exit 2", func(t *testing.T) {
		repo, home := guideFixture(t)
		code, _, stderr := runGuideInput(t, repo, home, "x\ny\nz\n", false)
		if code != 2 {
			t.Fatalf("invalid menu exit %d, want 2; stderr %q", code, stderr)
		}
		if !strings.Contains(stderr, "error: invalid menu selection") {
			t.Fatalf("missing invalid selection error: %q", stderr)
		}
		assertNoWrites(t, home)
	})

	eofCases := []struct {
		name  string
		stdin string
	}{
		{"menu", ""},
		{"pack prompt", "1\n"},
		{"free-first prompt", "1\ny\n"},
		{"install confirm", "1\nn\n"},
	}
	for _, tc := range eofCases {
		t.Run("eof cancels at "+tc.name, func(t *testing.T) {
			repo, home := guideFixture(t)
			code, stdout, stderr := runGuideInput(t, repo, home, tc.stdin, false)
			if code != 0 {
				t.Fatalf("eof at %s exit %d: stdout %q stderr %q", tc.name, code, stdout, stderr)
			}
			if !strings.Contains(stdout, "Cancelled.") {
				t.Fatalf("eof at %s did not print Cancelled.: %s", tc.name, stdout)
			}
			assertNoWrites(t, home)
		})
	}
}

// TestGuideExamplePromptsAppearInREADME keeps the guide's two first prompts in
// sync with the README "Everyday prompts" list, which owns their wording.
func TestGuideExamplePromptsAppearInREADME(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	for _, prompt := range guideExamplePrompts {
		if !strings.Contains(readme, prompt) {
			t.Fatalf("guide example prompt is not a verbatim README substring: %q", prompt)
		}
	}
}
