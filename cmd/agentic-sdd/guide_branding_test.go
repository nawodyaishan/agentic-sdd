package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentic-sdd/internal/skillsync"
)

// runBrandedGuide drives the runtime-injected guide seam for presentation
// tests without touching process globals or the network.
func runBrandedGuide(t *testing.T, repo, home, stdin string, env *Env, rt guideRuntime) (int, string, string) {
	t.Helper()
	args := []string{}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	args = append(args, "--home", home)
	var stdout, stderr bytes.Buffer
	code := runGuideWithRuntime(args, &stdout, &stderr, strings.NewReader(stdin), env, rt)
	return code, stdout.String(), stderr.String()
}

// brandingRuntime is the no-network runtime for presentation tests.
func brandingRuntime(installed string, checkCalls *int) guideRuntime {
	return guideRuntime{
		installed: installed,
		wait:      func(time.Duration) {},
		check: func(string) updateResult {
			if checkCalls != nil {
				*checkCalls++
			}
			return updateResult{state: updateEqual}
		},
	}
}

func guideTestLineWithPrefix(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

// TestGuideBrandingIntroOnceAndTerminalMenu covers AC-01/AC-08 at the guide
// seam: one branded entry, no duplicate legacy title, and the terminal-only
// update choice with precise cancellation guidance.
func TestGuideBrandingIntroOnceAndTerminalMenu(t *testing.T) {
	repo, home := guideFixture(t)
	checkCalls := 0
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, stdout, stderr := runBrandedGuide(t, repo, home, "4\nq\n", env, brandingRuntime("9.9.9", &checkCalls))
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d stderr %q", code, stderr)
	}
	if checkCalls != 0 {
		t.Fatalf("update check ran %d times without choice 5", checkCalls)
	}
	for _, want := range []string{
		brandMotif,
		brandTagline,
		brandCredit,
		"Version: 9.9.9",
		"Docs: " + brandDocsURL,
		"Agentic SDD installs reusable engineering skills",
	} {
		if n := strings.Count(stdout, want); n != 1 {
			t.Fatalf("output shows %q %d times, want once:\n%s", want, n, stdout)
		}
	}
	for _, want := range []string{
		"5) Check for CLI updates (contacts GitHub)",
		"6) Explore skills and agent profiles",
		"EOF exits; Ctrl+C interrupts the process.",
		"Write confirmation defaults to No.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("terminal menu missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Agentic SDD — engineering skills for your coding agents") {
		t.Fatalf("terminal menu repeated the legacy brand title:\n%s", stdout)
	}
	if n := strings.Count(stdout, "Main menu"); n != 2 {
		t.Fatalf("menu shown %d times, want 2:\n%s", n, stdout)
	}
	if strings.ContainsRune(stdout, 0x1b) {
		t.Fatalf("color-disabled terminal guide emitted ANSI:\n%q", stdout)
	}
	assertNoWrites(t, home)
}

// TestGuideBrandingPlainBranchExact pins the interactive-stdin, non-terminal
// stdout branch byte-for-byte, including choice 5 remaining invalid and no
// branding or animation.
func TestGuideBrandingPlainBranchExact(t *testing.T) {
	repo, home := guideFixture(t)
	checkCalls := 0
	waits := 0
	env := &Env{Interactive: true, Width: 80, Animate: true}
	rt := guideRuntime{
		installed: "9.9.9",
		wait:      func(time.Duration) { waits++ },
		check: func(string) updateResult {
			checkCalls++
			return updateResult{state: updateNewer, current: "1.0.0", latest: "2.0.0"}
		},
	}
	code, stdout, stderr := runBrandedGuide(t, repo, home, "4\nq\n", env, rt)
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d stderr %q", code, stderr)
	}
	const menu = `Agentic SDD — engineering skills for your coding agents
Nothing is written until you confirm.

What would you like to do?
  1) Install or update skills   (default)
  2) Restore an earlier backup
  3) Show the last install log
  4) What is this?
  q) Quit
` + "Choose [1]: "
	const about = `
Agentic SDD installs reusable engineering skills for coding agents.
It writes into four client directories, backs up anything it replaces,
and can undo an install with "agentic-sdd restore".
Every action is also a plain command: preview, apply, backups,
restore, logs. Nothing is written until you confirm.

`
	want := menu + about + menu + "Cancelled.\n"
	if stdout != want {
		t.Fatalf("plain guide bytes changed:\ngot  %q\nwant %q", stdout, want)
	}
	if waits != 0 || checkCalls != 0 {
		t.Fatalf("plain guide performed waits=%d checks=%d, want 0/0", waits, checkCalls)
	}
	assertNoWrites(t, home)
}

// TestGuideBrandingAnimationGate proves the exact opt-in and all suppressors,
// static/animated settled-display equality, and no frame after the first
// prompt.
func TestGuideBrandingAnimationGate(t *testing.T) {
	repo, home := guideFixture(t)

	run := func(t *testing.T, env *Env) (string, int, []string) {
		t.Helper()
		waits := 0
		var prefixes []string
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		rt := guideRuntime{
			installed: "dev",
			wait: func(time.Duration) {
				waits++
				prefixes = append(prefixes, stdout.String())
			},
			check: func(string) updateResult { return updateResult{state: updateEqual} },
		}
		code := runGuideWithRuntime([]string{"--repo", repo, "--home", home}, &stdout, &stderr, strings.NewReader("q\n"), env, rt)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, stderr.String())
		}
		return stdout.String(), waits, prefixes
	}

	capable := &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 80}
	staticOut, staticWaits, _ := run(t, capable)
	capableAnimate := *capable
	capableAnimate.Animate = true
	animatedOut, animatedWaits, prefixes := run(t, &capableAnimate)
	if staticWaits != 0 || animatedWaits != 20 {
		t.Fatalf("static waits=%d animated waits=%d, want 0/20", staticWaits, animatedWaits)
	}
	if settledBrandDisplay(staticOut) != settledBrandDisplay(animatedOut) {
		t.Fatalf("settled animated display differs from static:\n got: %q\nwant: %q", animatedOut, staticOut)
	}
	for i, prefix := range prefixes {
		if strings.Contains(prefix, "Choose [1]:") {
			t.Fatalf("wait %d occurred after the first prompt:\n%s", i+1, prefix)
		}
	}
	if n := strings.Count(animatedOut, brandTagline); n != 1 {
		t.Fatalf("animated intro showed the tagline %d times, want once", n)
	}
	if strings.Contains(animatedOut[strings.Index(animatedOut, brandTagline):], "\r") {
		t.Fatal("logo redraw continued into the tagline or menu")
	}

	suppressed := []struct {
		name string
		env  *Env
	}{
		{"opt-in absent", &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 80}},
		{"color disabled", &Env{Interactive: true, StdoutTTY: true, Width: 80, Animate: true}},
		{"dumb terminal", &Env{Interactive: true, StdoutTTY: true, ASCII: true, Width: 80, Animate: true}},
		{"narrow terminal", &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 40, Animate: true}},
	}
	for _, tc := range suppressed {
		t.Run(tc.name, func(t *testing.T) {
			out, waits, _ := run(t, tc.env)
			if waits != 0 {
				t.Fatalf("suppressed animation performed %d waits", waits)
			}
			for _, want := range []string{brandCredit, "Version: dev"} {
				if !strings.Contains(out, want) {
					t.Fatalf("suppressed intro lost %q:\n%s", want, out)
				}
			}
			if tc.env.Width >= brandArtWidth {
				if !strings.Contains(out, brandTagline) {
					t.Fatalf("suppressed intro lost the tagline:\n%s", out)
				}
			}
			if !tc.env.Color && strings.ContainsRune(out, 0x1b) {
				t.Fatalf("no-color guide emitted ANSI:\n%q", out)
			}
			if tc.env.Width < brandArtWidth {
				if strings.Contains(out, "+---") || strings.Contains(out, brandWordmark2) {
					t.Fatalf("narrow guide used art:\n%s", out)
				}
				if !strings.Contains(out, "Build software with AI. Stay the\nengineer.\n") {
					t.Fatalf("narrow guide did not wrap the tagline at 40:\n%s", out)
				}
			}
		})
	}

	t.Run("non-interactive stdin", func(t *testing.T) {
		env := &Env{StdoutTTY: true, Color: true, Width: 80, Animate: true}
		checkCalls := 0
		code, out, stderr := runBrandedGuide(t, repo, home, "q\n", env, brandingRuntime("dev", &checkCalls))
		if code != 2 || out != "" {
			t.Fatalf("non-interactive guide exit=%d stdout=%q stderr=%q", code, out, stderr)
		}
		if strings.Contains(stderr, brandTagline) {
			t.Fatalf("non-interactive guide printed branding to stderr: %q", stderr)
		}
	})
}

// TestGuideBrandingNarrowInstall covers width 40 layout, stacked counts,
// wrapped prose and complete paths/commands with spaces and non-ASCII text.
func TestGuideBrandingNarrowInstall(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo with spaces-日本語")
	home := filepath.Join(root, "home with spaces-日本語")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatal(err)
	}

	checkCalls := 0
	env := &Env{Interactive: true, StdoutTTY: true, Width: 40}
	code, stdout, stderr := runBrandedGuide(t, repo, home, "1\nn\ny\n", env, brandingRuntime("1.2.3", &checkCalls))
	if code != 0 {
		t.Fatalf("exit %d: stderr %q", code, stderr)
	}
	if checkCalls != 0 {
		t.Fatalf("update check ran %d times during install", checkCalls)
	}
	if strings.ContainsRune(stdout, 0x1b) {
		t.Fatalf("narrow no-color guide emitted ANSI:\n%q", stdout)
	}
	if strings.Contains(stdout, "+---") || strings.Contains(stdout, brandWordmark2) {
		t.Fatalf("narrow guide used art:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Build software with AI. Stay the\nengineer.\n") {
		t.Fatalf("narrow guide tagline was not wrapped at 40:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Version: 1.2.3\n") {
		t.Fatalf("narrow guide lost the installed version:\n%s", stdout)
	}
	if got := guideTestLineWithPrefix(stdout, "Target home: "); got != "Target home: "+absHome {
		t.Fatalf("home path changed or wrapped: got %q, want %q", got, "Target home: "+absHome)
	}
	if n := strings.Count(stdout, "  New: 1\n"); n != 4 {
		t.Fatalf("narrow guide per-destination new count = %d, want 4:\n%s", n, stdout)
	}
	if !strings.Contains(stdout, "4 managed entries: 4 will change, 0 already match.") {
		t.Fatalf("narrow guide lost inventory versus change counts:\n%s", stdout)
	}
	if !strings.Contains(stdout, "1. Choose options") || !strings.Contains(stdout, "2. Review changes") {
		t.Fatalf("narrow guide lost a stage heading:\n%s", stdout)
	}
	if strings.Contains(stdout, "Destination") && strings.Contains(stdout, "Unchanged") {
		t.Fatalf("narrow guide used the wide table:\n%s", stdout)
	}

	snapshotLine := guideTestLineWithPrefix(stdout, "Snapshot: ")
	if snapshotLine == "" {
		t.Fatalf("narrow guide lost the snapshot line:\n%s", stdout)
	}
	id := strings.TrimPrefix(snapshotLine, "Snapshot: ")
	if err := skillsync.ValidateBackupID(id); err != nil {
		t.Fatalf("narrow guide snapshot id %q is not whole: %v", id, err)
	}
	locationLine := guideTestLineWithPrefix(stdout, "Location: ")
	if locationLine == "" {
		t.Fatalf("narrow guide lost the snapshot location:\n%s", stdout)
	}
	if _, err := os.Stat(strings.TrimPrefix(locationLine, "Location: ")); err != nil {
		t.Fatalf("narrow guide snapshot location is not complete: %v", err)
	}
	logLine := guideTestLineWithPrefix(stdout, "Log: ")
	if logLine == "" {
		t.Fatalf("narrow guide lost the log line:\n%s", stdout)
	}
	if _, err := os.Stat(strings.TrimPrefix(logLine, "Log: ")); err != nil {
		t.Fatalf("narrow guide log path is not complete: %v", err)
	}
	for _, want := range []string{
		"Preview restoring this snapshot:",
		"5) Check for CLI updates (contacts GitHub)",
		"6) Explore skills and agent profiles",
		"Write confirmation defaults to No.",
		"agentic-sdd preview",
		"Add the OpenCode orchestration pack",
		"Install these changes? [y/N/b=change options]",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("narrow guide missing %q:\n%s", want, stdout)
		}
	}
}

// TestGuideBrandingWideStyling checks bold headings and the caution accent are
// applied only on the terminal branch.
func TestGuideBrandingWideStyling(t *testing.T) {
	repo, home := guideFixture(t)
	// Seed one differing skill so the review shows a replacement and the
	// meaningful-caution accent is exercised as well as headings.
	existing := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "SKILL.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	checkCalls := 0
	env := &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 80}
	code, stdout, stderr := runBrandedGuide(t, repo, home, "1\nn\ny\n", env, brandingRuntime("dev", &checkCalls))
	if code != 0 {
		t.Fatalf("exit %d: stderr %q", code, stderr)
	}
	for _, want := range []string{
		"\x1b[1m1. Choose options\x1b[0m",
		"\x1b[1m2. Review changes\x1b[0m",
		"\x1b[1mNext steps\x1b[0m",
		"Replacements are backed up first.",
		"\x1b[33m",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("wide terminal guide missing styling %q:\n%q", want, stdout)
		}
	}
}
