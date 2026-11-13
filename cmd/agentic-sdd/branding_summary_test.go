package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// summaryBackupIDRE finds the generated backup stamp in terminal summary
// output without depending on the removed transcript helper.
var summaryBackupIDRE = regexp.MustCompile(`[0-9]{8}T[0-9]{6}\.[0-9]{9}Z`)

func summaryFixtureWithSpaces(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo with spaces-日本語")
	home = filepath.Join(root, "home with spaces-日本語")
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

// TestBrandingSummaryPreview covers the unchanged wide preview rows, the bold
// heading, narrow stacked counts and complete space/Unicode paths.
func TestBrandingSummaryPreview(t *testing.T) {
	t.Run("wide no color keeps rows and counts", func(t *testing.T) {
		repo, home := uiFixture(t, false)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Width: 80})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		got := out.String()
		for _, want := range []string{
			"Summary (preview — nothing was written)",
			"  Total: 4 to install, 0 to replace (backed up first), 0 up to date — nothing was written.",
			"  Next: asdx-core apply --repo " + repo + " --home " + home,
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("wide preview missing %q:\n%s", want, got)
			}
		}
		for _, client := range []string{"agents", "codex", "claude", "agy"} {
			line := "  " + client + strings.Repeat(" ", 16-len(client)) + " 1 to install, 0 to replace, 0 up to date"
			if !strings.Contains(got, line) {
				t.Fatalf("wide preview missing client row %q:\n%s", line, got)
			}
		}
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("color-disabled summary emitted ANSI:\n%q", got)
		}
		if strings.Index(got, "Summary (preview") < strings.Index(got, "Preview only.") {
			t.Fatalf("summary heading printed before the raw transcript:\n%s", got)
		}
	})

	t.Run("wide color makes the heading bold", func(t *testing.T) {
		repo, home := uiFixture(t, false)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Color: true, Width: 80})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "\x1b[1mSummary (preview — nothing was written)\x1b[0m") {
			t.Fatalf("preview heading is not bold:\n%q", out.String())
		}
	})

	t.Run("narrow stacks labeled counts", func(t *testing.T) {
		repo, home := uiFixture(t, false)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Width: 40})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		got := out.String()
		for _, want := range []string{
			"  agents\n    1 to install\n    0 to replace\n    0 up to date\n",
			"  Total\n    4 to install\n    0 to replace\n    0 up to date\n",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("narrow preview missing stacked counts %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "1 to install, 0 to replace, 0 up to date") {
			t.Fatalf("narrow preview kept alignment-dependent rows:\n%s", got)
		}
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("narrow no-color preview emitted ANSI:\n%q", got)
		}
	})

	t.Run("narrow path with spaces and Unicode stays whole", func(t *testing.T) {
		repo, home := summaryFixtureWithSpaces(t)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Width: 40})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		got := out.String()
		want := "  Next: asdx-core apply --repo '" + repo + "' --home '" + home + "'"
		if !strings.Contains(got, want+"\n") {
			t.Fatalf("narrow preview did not keep the complete next command %q:\n%s", want, got)
		}
		if !strings.Contains(got, repo) || !strings.Contains(got, home) {
			t.Fatalf("narrow preview lost a complete path:\n%s", got)
		}
	})
}

// TestBrandingSummaryApply covers the existing wide result text, the new
// heading, narrow concision, and untouched backup id/log/undo values.
func TestBrandingSummaryApply(t *testing.T) {
	t.Run("wide color keeps legacy result and backup", func(t *testing.T) {
		repo, home := uiFixture(t, true)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Color: true, Width: 80})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		got := out.String()
		for _, want := range []string{
			"\x1b[1mApply complete\x1b[0m",
			"Installed 4 items (3 new, 1 replaced); backup: ",
			"  Undo: asdx-core restore",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("wide apply summary missing %q:\n%s", want, got)
			}
		}
		id := summaryBackupIDRE.FindString(got)
		if err := skillsync.ValidateBackupID(id); err != nil {
			t.Fatalf("summary backup id %q is not whole: %v", id, err)
		}
		logLine := guideTestLineWithPrefix(got, "Log: ")
		if logLine == "" {
			t.Fatalf("wide apply summary lost the log line:\n%s", got)
		}
		logPath := strings.TrimPrefix(logLine, "Log: ")
		if _, err := os.Stat(logPath); err != nil {
			t.Fatalf("summary log path is incomplete: %v", err)
		}
		logData := read(t, logPath)
		if strings.Contains(logData, "Apply complete") || strings.Contains(logData, "Undo: asdx-core restore") {
			t.Fatalf("log file gained terminal-only summary text:\n%s", logData)
		}
	})

	t.Run("narrow keeps install result, backup, log and undo whole", func(t *testing.T) {
		repo, home := uiFixture(t, true)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Width: 40})
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		got := out.String()
		for _, want := range []string{
			"Apply complete",
			"Installed 4 items",
			"  3 new, 1 replaced",
			"Undo: asdx-core restore",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("narrow apply summary missing %q:\n%s", want, got)
			}
		}
		backupLine := guideTestLineWithPrefix(got, "Backup: ")
		if backupLine == "" {
			t.Fatalf("narrow apply summary lost the backup line:\n%s", got)
		}
		id := strings.TrimPrefix(backupLine, "Backup: ")
		if err := skillsync.ValidateBackupID(id); err != nil {
			t.Fatalf("narrow backup id %q is not whole: %v", id, err)
		}
		logLine := guideTestLineWithPrefix(got, "Log: ")
		if logLine == "" {
			t.Fatalf("narrow apply summary lost the log line:\n%s", got)
		}
		if _, err := os.Stat(strings.TrimPrefix(logLine, "Log: ")); err != nil {
			t.Fatalf("narrow log path is incomplete: %v", err)
		}
		if strings.Contains(got, "Installed 4 items (3 new, 1 replaced); backup:") {
			t.Fatalf("narrow apply summary kept the long combined row:\n%s", got)
		}
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("narrow no-color apply emitted ANSI:\n%q", got)
		}
	})
}
