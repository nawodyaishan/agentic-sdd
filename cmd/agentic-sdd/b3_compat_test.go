package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"agentic-sdd/internal/version"
)

// This file pins the current non-TTY (piped/scripted) bytes and exit codes of
// the write commands that Feature 013 B3 must preserve (AC-10). Every
// assertion is an exact string comparison after normalizing only the values
// that are generated per run: the backup directory stamp, the log file name
// and the manifest's RFC3339Nano creation time. No assertion is a substring
// check.
//
// Fixtures use t.TempDir() and an explicit --repo, so backups land under the
// temporary repo and nothing ever touches the real user home.

var (
	// Generated log file: "<stamp>-<random digits>.log".
	b3CompatLogFileRE = regexp.MustCompile(`[0-9]{8}T[0-9]{6}\.[0-9]{9}Z-[0-9]+\.log`)
	// Generated backup directory: "<backup root>/<stamp>".
	b3CompatBackupIDRE = regexp.MustCompile(`(backups/)[0-9]{8}T[0-9]{6}\.[0-9]{9}Z`)
	// Manifest CreatedAt, formatted with time.RFC3339Nano (trailing zeros of
	// the fractional second are trimmed, so the fraction is optional).
	b3CompatCreatedAtRE = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z`)
)

// b3CompatNormalize replaces only the genuinely variable parts of a
// transcript with fixed placeholders. Log names are normalized first so the
// backup-directory rule cannot partially rewrite one.
func b3CompatNormalize(s string) string {
	s = b3CompatLogFileRE.ReplaceAllString(s, "<log-file>")
	s = b3CompatBackupIDRE.ReplaceAllString(s, "${1}<backup-id>")
	return b3CompatCreatedAtRE.ReplaceAllString(s, "<created-at>")
}

// b3CompatFixture builds a temporary repo with exactly one agentic-sdd-*
// skill and a temporary home. With seedOld the home already holds a
// different copy of that skill under claude, so apply replaces and backs it
// up; otherwise apply installs all four client destinations fresh.
func b3CompatFixture(t *testing.T, seedOld bool) (repo, home string) {
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
	if seedOld {
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

// b3CompatRun invokes the CLI through the non-TTY seam used by every
// compatibility test: no terminal on stdout/stderr, empty non-interactive
// stdin.
func b3CompatRun(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = runNonInteractive(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func b3CompatAssert(t *testing.T, name string, code, wantCode int, stdout, stderr, wantStdout, wantStderr string) {
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

// b3CompatOnlyBackupID returns the single backup directory name under repo's
// backup root, failing when the count is anything but one.
func b3CompatOnlyBackupID(t *testing.T, repo string) string {
	t.Helper()
	root := filepath.Join(repo, "backups")
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected exactly one backup under %s, got %v %v", root, entries, err)
	}
	return entries[0].Name()
}

// TestB3CompatApplySuccessNonTTY pins a successful apply on a fresh home:
// four install lines, the exact Installed summary with the backup path and
// the Log line.
func TestB3CompatApplySuccessNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, false)
	code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home)
	want := "install agents/agentic-sdd-plan\n" +
		"install codex/agentic-sdd-plan\n" +
		"install claude/agentic-sdd-plan\n" +
		"install agy/agentic-sdd-plan\n" +
		"Installed 4 skills; backup: " + filepath.Join(repo, "backups", "<backup-id>") + "\n" +
		"Log: " + filepath.Join(home, ".agentic-sdd", "logs", "<log-file>") + "\n"
	b3CompatAssert(t, "apply", code, 0, b3CompatNormalize(stdout), stderr, want, "")

	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); err != nil {
		t.Fatalf("expected installed skill: %v", err)
	}
	b3CompatOnlyBackupID(t, repo)
}

// TestB3CompatApplyNoOpNonTTY pins the second apply on an unchanged home: the
// no-change line plus the Log line, and no additional backup.
func TestB3CompatApplyNoOpNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, false)
	if code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home); code != 0 {
		t.Fatalf("setup apply exited %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home)
	want := "up to date agents/agentic-sdd-plan\n" +
		"up to date codex/agentic-sdd-plan\n" +
		"up to date claude/agentic-sdd-plan\n" +
		"up to date agy/agentic-sdd-plan\n" +
		"All skills are up to date.\n" +
		"Log: " + filepath.Join(home, ".agentic-sdd", "logs", "<log-file>") + "\n"
	b3CompatAssert(t, "no-op apply", code, 0, b3CompatNormalize(stdout), stderr, want, "")

	b3CompatOnlyBackupID(t, repo)
}

// TestB3CompatRestorePreviewNonTTY pins the restore preview transcript for a
// real apply-produced backup. The codex destination (installed by apply) is
// removed so the installed-entry branch reports "up to date", and the claude
// destination (replaced by apply) is removed so the replaced-entry branch
// reports the "restore" verb. Preview writes nothing.
func TestB3CompatRestorePreviewNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, true)
	if code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home); code != 0 {
		t.Fatalf("setup apply exited %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	id := b3CompatOnlyBackupID(t, repo)
	for _, client := range []string{".codex", ".claude"} {
		if err := os.RemoveAll(filepath.Join(home, client, "skills", "agentic-sdd-plan")); err != nil {
			t.Fatal(err)
		}
	}
	code, stdout, stderr := b3CompatRun(t, "restore", "preview", id, "--repo", repo, "--home", home)
	want := "remove + back up agents/agentic-sdd-plan\n" +
		"up to date codex/agentic-sdd-plan\n" +
		"restore claude/agentic-sdd-plan\n" +
		"remove + back up agy/agentic-sdd-plan\n" +
		"Preview only. Run with --apply (or restore apply) to restore.\n"
	b3CompatAssert(t, "restore preview", code, 0, stdout, stderr, want, "")

	// Preview must not have written: the deleted claude destination stays
	// absent and the untouched installs remain in place.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("preview created the deleted claude destination: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")); err != nil {
		t.Fatalf("preview removed an install: %v", err)
	}
	b3CompatOnlyBackupID(t, repo)
}

// TestB3CompatRestoreApplyNonTTY pins the restore apply transcript for the
// same apply-produced backup after one installed destination (codex) is
// deleted, plus the Restored summary, the Log line, exit 0 and the resulting
// home state.
func TestB3CompatRestoreApplyNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, true)
	if code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home); code != 0 {
		t.Fatalf("setup apply exited %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	id := b3CompatOnlyBackupID(t, repo)
	if err := os.RemoveAll(filepath.Join(home, ".codex", "skills", "agentic-sdd-plan")); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := b3CompatRun(t, "restore", "apply", id, "--repo", repo, "--home", home)
	want := "remove + back up agents/agentic-sdd-plan\n" +
		"up to date codex/agentic-sdd-plan\n" +
		"replace + back up claude/agentic-sdd-plan\n" +
		"remove + back up agy/agentic-sdd-plan\n" +
		"Restored 3 skills; backup: " + filepath.Join(repo, "backups", "<backup-id>") + "\n" +
		"Log: " + filepath.Join(home, ".agentic-sdd", "logs", "<log-file>") + "\n"
	b3CompatAssert(t, "restore apply", code, 0, b3CompatNormalize(stdout), stderr, want, "")

	// The restore removes the fresh installs, restores claude's saved
	// content and leaves the already-deleted codex destination alone.
	if got := read(t, filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")); got != "old" {
		t.Fatalf("restore did not revert claude content: %q", got)
	}
	for _, client := range []string{".agents", ".codex", ".gemini"} {
		if _, err := os.Stat(filepath.Join(home, client, "skills", "agentic-sdd-plan")); !os.IsNotExist(err) {
			t.Fatalf("expected %s destination to be absent, got %v", client, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected the apply backup plus the restore backup, got %v %v", entries, err)
	}
}

// TestB3CompatBackupsListingNonTTY pins the backups listing line for the
// apply-produced backup: id, creation time, operation, tool version, source
// and summary, in that exact field order. A second invocation must be
// byte-identical because the recorded timestamp no longer changes.
func TestB3CompatBackupsListingNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, true)
	if code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home); code != 0 {
		t.Fatalf("setup apply exited %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	id := b3CompatOnlyBackupID(t, repo)
	code, stdout, stderr := b3CompatRun(t, "backups", "--repo", repo, "--home", home)
	want := "1) " + id + "  <created-at>  apply  tool " + version.Version +
		"  source " + filepath.Join(repo, "skills-files") +
		"  1 replaced, 3 installed across agents, agy, claude, codex\n"
	b3CompatAssert(t, "backups", code, 0, b3CompatNormalize(stdout), stderr, want, "")

	code, again, stderr := b3CompatRun(t, "backups", "--repo", repo, "--home", home)
	b3CompatAssert(t, "backups repeated", code, 0, again, stderr, stdout, "")
}

// TestB3CompatFailureBeforeBackupNonTTY pins a failure that happens while
// planning, before any backup is written: a destination that is not a
// directory. The error line and exit code are exact, the only stdout is the
// Log line, and no backup root is created.
func TestB3CompatFailureBeforeBackupNonTTY(t *testing.T) {
	repo, home := b3CompatFixture(t, false)
	target := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := b3CompatRun(t, "apply", "--repo", repo, "--home", home)
	wantStdout := "Log: " + filepath.Join(home, ".agentic-sdd", "logs", "<log-file>") + "\n"
	wantStderr := "error: refusing non-directory target " + target + "\n"
	b3CompatAssert(t, "failure before backup", code, 1, b3CompatNormalize(stdout), stderr, wantStdout, wantStderr)

	if _, err := os.Stat(filepath.Join(repo, "backups")); !os.IsNotExist(err) {
		t.Fatalf("expected no backup root, got %v", err)
	}
}
