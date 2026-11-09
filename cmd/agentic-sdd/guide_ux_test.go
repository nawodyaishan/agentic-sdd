package main

import (
	"strings"
	"testing"
)

func TestGuideUXMenuReturns(t *testing.T) {
	repo, home := guideFixture(t)
	spy := &guideUpdateSpy{result: updateResult{state: updateUnavailable, latest: "2.0.0"}}
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "5\n4\nq\n", env, guideRuntimeWithSpy("dev", spy))
	if code != 0 || errOut != "" || spy.calls != 1 {
		t.Fatalf("code=%d stderr=%q checks=%d", code, errOut, spy.calls)
	}
	for _, want := range []string{
		"Development build", "release comparison unavailable",
		"1) Preview and install skills", "2) Restore from a backup",
		"3) View latest install or restore log", "4) About and getting started",
		"5) Check for CLI updates (contacts GitHub)",
		"Enter selects 1. Type q to exit.", "EOF exits; Ctrl+C interrupts the process.",
		"GitHub release metadata received.", "Latest published release: 2.0.0",
		"Development or non-release build; version comparison unavailable.",
		"Upgrading the CLI does not update installed skills.",
		"Restoring can replace current files and remove freshly installed items.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "Skill and profile changes require confirmation."); got != 1 {
		t.Errorf("safety notice count=%d, want 1", got)
	}
	if strings.Contains(out, "Nothing is written until you confirm.") {
		t.Error("overbroad no-write promise remains")
	}
	if got := strings.Count(out, "Main menu\n"); got != 3 {
		t.Errorf("main menu boundaries=%d, want 3", got)
	}
	assertNoWrites(t, home)
}

func TestGuideUXRestoreImpactBeforeConfirmation(t *testing.T) {
	repo, home, _ := setupBackupCLI(t)
	env := &Env{Interactive: true, StdoutTTY: true, Width: 80}
	code, out, errOut := runBrandedGuide(t, repo, home, "2\n1\nn\n", env, brandingRuntime("1.0.0", nil))
	impact := strings.Index(out, "Restoring can replace current files and remove freshly installed items.")
	confirm := strings.Index(out, "Restore these skills?")
	if code != 0 || errOut != "" || impact < 0 || confirm < impact {
		t.Fatalf("impact must precede confirmation: code=%d stderr=%q\n%s", code, errOut, out)
	}
}
