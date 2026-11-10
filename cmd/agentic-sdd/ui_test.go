package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// uiFixture builds the one-skill repository used by the presentation checks.
// When seedClaude is true it also creates an older claude copy, so the plan
// reports one replacement instead of four installs.
func uiFixture(t *testing.T, seedClaude bool) (repo, home string) {
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
	if seedClaude {
		old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
		if err := os.MkdirAll(old, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return repo, home
}

// TestTerminalSummaryAndColor is Check C3 (AC-04): terminal preview and apply
// add correct counts, carry only explicitly passed flags into the next
// command, and style only when color is enabled.
func TestTerminalSummaryAndColor(t *testing.T) {
	t.Run("preview summary without color", func(t *testing.T) {
		repo, home := uiFixture(t, false)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Width: 80})
		if code != 0 {
			t.Fatalf("preview exited %d: %s", code, errOut.String())
		}
		got := out.String()
		wants := []string{
			"install claude/agentic-sdd-plan",
			"Preview only. Run with apply (or --apply) to install.",
			"Summary (preview — nothing was written)",
			"  Total: 4 to install, 0 to replace (backed up first), 0 up to date — nothing was written.",
			"  Next: agentic-sdd apply --repo " + repo + " --home " + home,
		}
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Fatalf("preview output missing %q:\n%s", want, got)
			}
		}
		if n := strings.Count(got, "1 to install, 0 to replace, 0 up to date"); n != 4 {
			t.Fatalf("expected four per-client install counts, found %d:\n%s", n, got)
		}
		for _, client := range []string{"agents", "codex", "claude", "agy"} {
			line := fmt.Sprintf("  %-16s 1 to install, 0 to replace, 0 up to date", client)
			if !strings.Contains(got, line) {
				t.Fatalf("preview output missing client line %q:\n%s", line, got)
			}
		}
		if strings.Contains(got, "\x1b") {
			t.Fatalf("preview with color disabled emitted ANSI escapes:\n%q", got)
		}
	})

	t.Run("preview summary with color", func(t *testing.T) {
		repo, home := uiFixture(t, false)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"preview", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Color: true, Width: 80})
		if code != 0 {
			t.Fatalf("preview exited %d: %s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "\x1b[") {
			t.Fatalf("preview with color enabled emitted no ANSI escape:\n%q", out.String())
		}
	})

	t.Run("next command carries only explicit flags", func(t *testing.T) {
		cases := []struct {
			name    string
			repo    string
			rawHome string
			opts    skillsync.Options
			want    string
		}{
			{"home omitted", "/tmp/skills", "", skillsync.Options{}, "agentic-sdd apply --repo /tmp/skills"},
			{"no flags", "", "", skillsync.Options{}, "agentic-sdd apply"},
			{"opencode pack", "", "", skillsync.Options{WithOpenCode: true, OpenCodeFreeFirst: true}, "agentic-sdd apply --with-opencode --opencode-free-first"},
			{"unsafe values quoted", "/tmp/my skills", "/tmp/home dir", skillsync.Options{}, "agentic-sdd apply --repo '/tmp/my skills' --home '/tmp/home dir'"},
		}
		for _, tc := range cases {
			if got := formatNextApplyCommand(tc.repo, tc.rawHome, tc.opts); got != tc.want {
				t.Fatalf("%s: formatNextApplyCommand = %q, want %q", tc.name, got, tc.want)
			}
		}
	})

	t.Run("apply summary and untouched log", func(t *testing.T) {
		repo, home := uiFixture(t, true)
		var out, errOut bytes.Buffer
		code := runEnv([]string{"apply", "--repo", repo, "--home", home}, &out, &errOut, strings.NewReader(""), &Env{StdoutTTY: true, Color: true, Width: 80})
		if code != 0 {
			t.Fatalf("apply exited %d: %s", code, errOut.String())
		}
		got := out.String()
		for _, want := range []string{
			"Installed 4 items (3 new, 1 replaced); backup: ",
			"  Undo: agentic-sdd restore",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("apply output missing %q:\n%s", want, got)
			}
		}
		if !strings.Contains(got, "\x1b[") {
			t.Fatalf("apply summary with color enabled emitted no ANSI escape:\n%q", got)
		}
		entries, err := os.ReadDir(logsDir(home))
		if err != nil || len(entries) != 1 {
			t.Fatalf("expected one apply log, got %v, %v", entries, err)
		}
		logData := read(t, filepath.Join(logsDir(home), entries[0].Name()))
		if strings.Contains(logData, "Undo: agentic-sdd restore") || strings.Contains(logData, "Summary (apply") {
			t.Fatalf("log file contains terminal-only summary text:\n%s", logData)
		}
		if !strings.Contains(logData, "Result: success") {
			t.Fatalf("log file missing pre-feature transcript tail:\n%s", logData)
		}
	})
}

// TestHelpTopics is Check C4 (AC-05): eight distinct topic outputs exit 0 and
// an unknown topic keeps the exact pre-feature usage error.
func TestHelpTopics(t *testing.T) {
	topics := []string{"preview", "apply", "backups", "logs", "restore", "guide", "version", "tui"}
	headers := make(map[string]string, len(topics))
	for _, topic := range topics {
		var out, errOut bytes.Buffer
		code := runNonInteractive([]string{"help", topic}, &out, &errOut)
		if code != 0 {
			t.Fatalf("help %s exited %d: %s", topic, code, errOut.String())
		}
		got := out.String()
		usageIdx := strings.Index(got, "Usage: agentic-sdd")
		if usageIdx < 0 {
			t.Fatalf("help %s output missing global usage:\n%s", topic, got)
		}
		firstLine, _, _ := strings.Cut(got, "\n")
		firstLine = strings.TrimSpace(firstLine)
		if !strings.HasPrefix(firstLine, topic+" ") {
			t.Fatalf("help %s header %q does not start with the topic name", topic, firstLine)
		}
		if headerIdx := strings.Index(got, firstLine); headerIdx < 0 || headerIdx >= usageIdx {
			t.Fatalf("help %s topic header is not before the usage block:\n%s", topic, got)
		}
		if other, dup := headers[firstLine]; dup {
			t.Fatalf("help %s shares header %q with help %s", topic, firstLine, other)
		}
		headers[firstLine] = topic
		for _, want := range []string{"Writes files:", "Example:", "Related:"} {
			if !strings.Contains(got, want) {
				t.Fatalf("help %s output missing %q:\n%s", topic, want, got)
			}
		}
		if strings.Contains(got, "\x1b") {
			t.Fatalf("help %s output contains ANSI escapes:\n%q", topic, got)
		}
	}

	var out, errOut bytes.Buffer
	if code := runNonInteractive([]string{"help", "unknown"}, &out, &errOut); code != 2 {
		t.Fatalf("help unknown exited %d, want 2; stdout %q", code, out.String())
	}
	wantErr := "error: usage: agentic-sdd help [preview|apply|backups|logs|restore|tui]\n"
	if errOut.String() != wantErr {
		t.Fatalf("help unknown stderr = %q, want %q", errOut.String(), wantErr)
	}
	if out.String() != "" {
		t.Fatalf("help unknown stdout = %q, want empty", out.String())
	}
}

// TestErrorHintsTerminalOnly is Check C5 (AC-06): each listed error gains one
// hint line on a terminal stderr, keeps its exit code, and gains nothing when
// stderr is not a terminal or when the log path is rejected for safety.
func TestErrorHintsTerminalOnly(t *testing.T) {
	cases := []struct {
		name string
		args func(repo, home string) []string
		hint string
	}{
		{
			"restore without id",
			func(repo, home string) []string { return []string{"restore", "--repo", repo, "--home", home} },
			"run agentic-sdd backups to list ids, or agentic-sdd guide to pick one",
		},
		{
			"unknown command",
			func(string, string) []string { return []string{"wat"} },
			"run agentic-sdd help to see the commands",
		},
		{
			"top-level flag error",
			func(string, string) []string { return []string{"--wat"} },
			"run agentic-sdd help preview for the accepted flags",
		},
		{
			"backups flag error",
			func(string, string) []string { return []string{"backups", "--wat"} },
			"run agentic-sdd help backups for the accepted flags",
		},
		{
			"restore flag error",
			func(string, string) []string { return []string{"restore", "--wat"} },
			"run agentic-sdd help restore for the accepted flags",
		},
		{
			"logs flag error",
			func(string, string) []string { return []string{"logs", "--wat"} },
			"run agentic-sdd help logs for the accepted flags",
		},
		{
			"invalid backup id",
			func(repo, home string) []string {
				return []string{"restore", "notanid", "--repo", repo, "--home", home}
			},
			"ids look like 20260101T000000.000000000Z; list them with agentic-sdd backups",
		},
		{
			"free-first without with-opencode",
			func(_, home string) []string { return []string{"preview", "--opencode-free-first", "--home", home} },
			"add --with-opencode, or drop --opencode-free-first",
		},
		{
			"logs without a log directory",
			func(_, home string) []string { return []string{"logs", "--home", home} },
			"logs are written by apply and restore; run agentic-sdd preview first to see what apply would do",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, home := uiFixture(t, false)
			args := tc.args(repo, home)

			var plainOut, plainErr bytes.Buffer
			plainCode := run(args, &plainOut, &plainErr, strings.NewReader(""), false)

			var ttyOut, ttyErr bytes.Buffer
			ttyCode := runEnv(args, &ttyOut, &ttyErr, strings.NewReader(""), &Env{Width: 80, StderrTTY: true})

			if plainCode != ttyCode {
				t.Fatalf("exit code changed with a terminal stderr: %d vs %d", plainCode, ttyCode)
			}
			if strings.Contains(plainErr.String(), "hint:") {
				t.Fatalf("non-terminal stderr contains a hint:\n%q", plainErr.String())
			}
			if ttyOut.String() != plainOut.String() {
				t.Fatalf("stdout changed with a terminal stderr: %q vs %q", ttyOut.String(), plainOut.String())
			}
			errorLine, _, _ := strings.Cut(ttyErr.String(), "\n")
			if !strings.HasPrefix(errorLine, "error:") {
				t.Fatalf("expected an unchanged error line first, got %q", ttyErr.String())
			}
			want := errorLine + "\nhint: " + tc.hint + "\n"
			if !strings.HasPrefix(ttyErr.String(), want) {
				t.Fatalf("terminal stderr = %q, want it to start with %q", ttyErr.String(), want)
			}
		})
	}

	t.Run("logs with symlinked log path gets no hint", func(t *testing.T) {
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
		var out, errOut bytes.Buffer
		code := runEnv([]string{"logs", "--home", home}, &out, &errOut, strings.NewReader(""), &Env{Width: 80, StderrTTY: true})
		if code != 1 {
			t.Fatalf("logs exited %d, want 1; stderr %q", code, errOut.String())
		}
		if !strings.Contains(errOut.String(), "refusing non-directory log path") {
			t.Fatalf("stderr = %q, want the log-path rejection", errOut.String())
		}
		if strings.Contains(errOut.String(), "hint:") {
			t.Fatalf("rejected log path printed a hint:\n%q", errOut.String())
		}
	})
}
