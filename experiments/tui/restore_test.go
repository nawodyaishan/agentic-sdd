package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// restoreFixture creates a temporary repo/home pair and one real install
// backup: agentic-sdd-plan already exists under claude (so its old content is
// saved) and is installed fresh for the other three clients.
func restoreFixture(t *testing.T) (repo, home, backupID string) {
	t.Helper()
	root := t.TempDir()
	repo = filepath.Join(root, "repo")
	home = filepath.Join(root, "home")

	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := skillsync.Sync(repo, home, true, io.Discard); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("seed backups = %d, want 1", len(backups))
	}
	return repo, home, backups[0].ID
}

func restoreConfig(repo, home string) TUIConfig {
	return TUIConfig{Repo: repo, Home: home, Width: 80, Height: 24, Color: false}
}

func newLoadedRestoreModel(t *testing.T, cfg TUIConfig) RestoreModel {
	t.Helper()
	m := NewRestoreModel(cfg, RestoreOps{})
	m = restoreStep(t, m, m.Init())
	return m
}

// restoreStep executes one tea.Cmd and feeds its message back to the model,
// returning the model and any command the state transition produced.
func restoreStep(t *testing.T, m RestoreModel, cmd tea.Cmd) RestoreModel {
	t.Helper()
	if cmd == nil {
		return m
	}
	m, _ = restoreStepCmd(t, m, cmd)
	return m
}

func restoreStepCmd(t *testing.T, m RestoreModel, cmd tea.Cmd) (RestoreModel, tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("expected a command, got nil")
	}
	msg := cmd()
	if msg == nil {
		t.Fatalf("command returned no message")
	}
	return m.Update(msg)
}

// restoreWriteStep emulates the root single-write host: the screen only emits
// a StartWriteMsg, so the test executes the real revalidated apply and feeds the
// typed result back. This preserves the filesystem coverage without giving the
// screen a direct engine call.
func restoreWriteStep(t *testing.T, m RestoreModel, cmd tea.Cmd) RestoreModel {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a write request command, got nil")
	}
	msg, ok := cmd().(StartWriteMsg)
	if !ok {
		t.Fatalf("write command returned %T, want StartWriteMsg", cmd())
	}
	if msg.Request.Kind != WriteRestore {
		t.Fatalf("write request kind = %v, want WriteRestore", msg.Request.Kind)
	}
	result, err := appops.ApplyRestore(msg.Request.Restore, msg.Request.Token, io.Discard)
	updated, _ := m.Update(WriteResultMsg{
		ID:     appops.OperationID{Request: 1, Generation: 1},
		Kind:   WriteRestore,
		Result: result,
		Err:    err,
	})
	return updated
}

func restoreKey(code rune) tea.KeyPressMsg {
	if code < 32 || code == 127 {
		return tea.KeyPressMsg{Code: code}
	}
	return tea.KeyPressMsg{Code: code, Text: string(code)}
}

func restoreEnter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
func restoreEsc() tea.KeyPressMsg   { return tea.KeyPressMsg{Code: tea.KeyEscape} }

func restoreLines(m RestoreModel, width int) string {
	return strings.Join(m.Lines(width), "\n")
}

// snapshotRestoreHome records every file under root so a test can prove a
// transition wrote nothing.
func snapshotRestoreHome(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func assertRestoreHomeUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("home file count changed: %d -> %d", len(before), len(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Fatalf("home file %s changed from %q to %q", path, content, after[path])
		}
	}
}

func restoreBackupCount(t *testing.T, repo, home string) int {
	t.Helper()
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	return len(backups)
}

// TestRestoreEnterOpensDetailsOnlyAndWritesNothing covers AC-06's first half:
// opening a backup, previewing it and even starting confirmation write
// nothing, and Enter is never consent.
func TestRestoreEnterOpensDetailsOnlyAndWritesNothing(t *testing.T) {
	repo, home, id := restoreFixture(t)
	cfg := restoreConfig(repo, home)
	m := newLoadedRestoreModel(t, cfg)
	if len(m.Backups()) != 1 {
		t.Fatalf("backups = %d, want 1", len(m.Backups()))
	}
	before := snapshotRestoreHome(t, home)

	m, _ = m.Update(restoreEnter())
	if m.Stage() != RestoreStageDetails {
		t.Fatalf("stage after Enter = %v, want details", m.Stage())
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))

	m, cmd := m.Update(restoreEnter())
	if m.Stage() != RestoreStagePreview || cmd == nil {
		t.Fatalf("stage after preview Enter = %v, cmd nil=%t; want preview + command", m.Stage(), cmd == nil)
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))
	m = restoreStep(t, m, cmd)
	if m.PreviewResult().PreviewToken == "" {
		t.Fatal("preview produced no token")
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))

	m, cmd = m.Update(restoreEnter())
	if m.Stage() != RestoreStageConfirm || cmd != nil {
		t.Fatalf("stage after confirm Enter = %v, cmd nil=%t; want confirm, no command", m.Stage(), cmd == nil)
	}
	if !strings.Contains(restoreLines(m, 80), "Enter does not confirm") {
		t.Fatal("confirmation does not state that Enter is inert")
	}
	m, cmd = m.Update(restoreEnter())
	if m.Stage() != RestoreStageConfirm || cmd != nil {
		t.Fatalf("Enter confirmed a restore: stage=%v cmd nil=%t", m.Stage(), cmd == nil)
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))
	if got := restoreBackupCount(t, repo, home); got != 1 {
		t.Fatalf("backups after inert confirmation = %d, want 1", got)
	}
	_ = id
}

// TestRestoreFullJourneyAppliesOnce covers the async apply path, its result
// evidence and duplicate-confirmation/double-result protection.
func TestRestoreFullJourneyAppliesOnce(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))

	m, _ = m.Update(restoreEnter()) // details
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	m, _ = m.Update(restoreEnter()) // confirm

	m, revalidateCmd := m.Update(restoreKey('y'))
	if m.Stage() != RestoreStageRevalidating || revalidateCmd == nil {
		t.Fatalf("stage after y = %v, cmd nil=%t; want revalidating", m.Stage(), revalidateCmd == nil)
	}
	// A duplicate confirmation while revalidating must not dispatch anything.
	if dup, cmd := m.Update(restoreKey('y')); cmd != nil || dup.Stage() != RestoreStageRevalidating {
		t.Fatalf("duplicate y changed state: stage=%v cmd nil=%t", dup.Stage(), cmd == nil)
	}
	m, applyCmd := restoreStepCmd(t, m, revalidateCmd)
	if m.Stage() != RestoreStageRunning || applyCmd == nil {
		t.Fatalf("stage after revalidation = %v, cmd nil=%t; want running", m.Stage(), applyCmd == nil)
	}
	m = restoreWriteStep(t, m, applyCmd)

	if m.Stage() != RestoreStageResult {
		t.Fatalf("stage after apply = %v, want result", m.Stage())
	}
	out, ok := m.Outcome()
	if !ok || out.Kind != RestoreOutcomeApplied {
		t.Fatalf("outcome = %+v ok=%t, want applied", out, ok)
	}
	if out.SafetyBackupID == "" || out.SafetyBackupPath == "" {
		t.Fatalf("applied result lacks pre-restore backup identity: %+v", out)
	}
	if got := restoreBackupCount(t, repo, home); got != 2 {
		t.Fatalf("backups after restore = %d, want 2 (seed + pre-restore)", got)
	}
	assertFile := func(path, want string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("%s = %q, want %q", path, data, want)
		}
	}
	assertFile(filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md"), "old")
	if _, err := os.Lstat(filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")); !os.IsNotExist(err) {
		t.Fatalf("fresh install under agents was not removed by restore: err=%v", err)
	}

	// A repeated y and a re-delivered apply message cannot apply twice.
	if again, cmd := m.Update(restoreKey('y')); cmd != nil || again.Stage() != RestoreStageResult {
		t.Fatalf("duplicate y after result changed state: stage=%v cmd nil=%t", again.Stage(), cmd == nil)
	}
	before := restoreBackupCount(t, repo, home)
	m2 := m
	m2, _ = m2.Update(WriteResultMsg{ID: appops.OperationID{Generation: 1}, Kind: WriteRestore, Err: errors.New("stale")})
	if m2.Stage() != RestoreStageResult {
		t.Fatalf("stale apply message changed stage to %v", m2.Stage())
	}
	if got := restoreBackupCount(t, repo, home); got != before {
		t.Fatalf("stale apply message wrote a backup: %d -> %d", before, got)
	}
}

// TestRestoreNoChangeIsHonest proves a no-op restore reports no change and
// creates no pre-restore backup.
func TestRestoreNoChangeIsHonest(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	// Re-applying the same source makes every destination match the source,
	// but the restore plan still targets the seed backup. Apply the restore
	// once, then preview/confirm the same backup again: the second restore is
	// the no-change case (every destination already matches).
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	m, _ = m.Update(restoreEnter())
	m, revalidateCmd := m.Update(restoreKey('y'))
	m, applyCmd := restoreStepCmd(t, m, revalidateCmd)
	m = restoreWriteStep(t, m, applyCmd)
	if out, _ := m.Outcome(); out.Kind != RestoreOutcomeApplied {
		t.Fatalf("first restore outcome = %v, want applied", out.Kind)
	}

	// Return to the list (refreshes it), open the original seed backup again
	// and restore it a second time.
	m, listCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, listCmd)
	if len(m.Backups()) != 2 {
		t.Fatalf("backups after refresh = %d, want 2", len(m.Backups()))
	}
	// Select the seed backup explicitly: it is the oldest entry.
	m.setSelection(len(m.Backups()) - 1)
	m, _ = m.Update(restoreEnter())
	m, previewCmd = m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	m, _ = m.Update(restoreEnter())
	m, revalidateCmd = m.Update(restoreKey('y'))
	m, applyCmd = restoreStepCmd(t, m, revalidateCmd)
	m = restoreWriteStep(t, m, applyCmd)
	out, _ := m.Outcome()
	if out.Kind != RestoreOutcomeNoChange {
		t.Fatalf("second restore outcome = %v (%+v), want no-change", out.Kind, out)
	}
	if out.SafetyBackupID != "" {
		t.Fatalf("no-change restore reports a backup it should not have written: %q", out.SafetyBackupID)
	}
	if got := restoreBackupCount(t, repo, home); got != 2 {
		t.Fatalf("backups after no-change restore = %d, want 2", got)
	}
	if !strings.Contains(restoreLines(m, 80), "No changes needed") {
		t.Fatal("no-change result lacks an honest message")
	}
}

// TestRestoreSubsetBackupLeavesUnselectedClientUntouched covers AC-06's subset
// requirement: only recorded changes are restored and unrelated destinations
// remain byte-identical.
func TestRestoreSubsetBackupLeavesUnselectedClientUntouched(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	agents := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	for path, content := range map[string]string{claude: "claude-old", agents: "agents-old"} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := skillsync.SyncWithOptions(repo, home, true, skillsync.Options{ClientIDs: []string{"claude"}}, io.Discard); err != nil {
		t.Fatalf("subset install: %v", err)
	}
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil || len(backups) != 1 {
		t.Fatalf("subset backups = %d err=%v, want 1", len(backups), err)
	}
	if !strings.Contains(backups[0].Summary, "across claude") {
		t.Fatalf("subset summary = %q, want only claude", backups[0].Summary)
	}

	if err := os.WriteFile(filepath.Join(claude, "SKILL.md"), []byte("drifted"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentsBefore := snapshotRestoreHome(t, filepath.Join(home, ".agents"))

	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	for _, change := range m.PreviewResult().Planned {
		if change.Client != "claude" {
			t.Fatalf("subset restore planned an unselected client change: %+v", change)
		}
	}
	m, _ = m.Update(restoreEnter())
	m, revalidateCmd := m.Update(restoreKey('y'))
	m, applyCmd := restoreStepCmd(t, m, revalidateCmd)
	m = restoreWriteStep(t, m, applyCmd)
	if out, _ := m.Outcome(); out.Kind != RestoreOutcomeApplied {
		t.Fatalf("subset restore outcome = %v, want applied", out.Kind)
	}
	data, err := os.ReadFile(filepath.Join(claude, "SKILL.md"))
	if err != nil || string(data) != "claude-old" {
		t.Fatalf("claude restored to %q err=%v, want claude-old", data, err)
	}
	assertRestoreHomeUnchanged(t, agentsBefore, snapshotRestoreHome(t, filepath.Join(home, ".agents")))
}

// TestRestorePreviewShowsExactOverwriteAndRemovalPaths covers the required
// inspectable impact.
func TestRestorePreviewShowsExactOverwriteAndRemovalPaths(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)

	var replace, remove int
	for _, change := range m.PreviewResult().Planned {
		switch change.Action {
		case skillsync.ChangeReplace:
			replace++
		case skillsync.ChangeRemove:
			remove++
		}
	}
	if replace != 1 || remove != 3 {
		t.Fatalf("planned replace/remove = %d/%d, want 1/3", replace, remove)
	}
	text := restoreLines(m, 120)
	overwritePath := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
	removePath := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	for _, want := range []string{
		"Will overwrite existing content (1)",
		overwritePath,
		"Will remove content that did not exist before that backup (3)",
		removePath,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("preview missing %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "Nothing has been written") {
		t.Fatal("preview does not state that nothing was written")
	}
}

// TestRestoreDriftInvalidatesConsent proves changed destination state forces a
// fresh preview and confirmation before any write.
func TestRestoreDriftInvalidatesConsent(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	m, _ = m.Update(restoreEnter())
	if m.Stage() != RestoreStageConfirm {
		t.Fatalf("stage = %v, want confirm", m.Stage())
	}
	dest := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")
	if err := os.WriteFile(dest, []byte("changed after preview"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A content-only change keeps the same overwrite path and is still safely
	// replaced after being saved; a destination disappearing changes the
	// proposed change set (remove -> up-to-date) and must invalidate consent.
	removed := filepath.Join(home, ".agents", "skills", "agentic-sdd-plan")
	if err := os.RemoveAll(removed); err != nil {
		t.Fatal(err)
	}
	m, revalidateCmd := m.Update(restoreKey('y'))
	m, applyCmd := restoreStepCmd(t, m, revalidateCmd)
	if applyCmd != nil {
		t.Fatal("drift dispatched an apply command")
	}
	if m.Stage() != RestoreStagePreview {
		t.Fatalf("stage after drift = %v, want preview", m.Stage())
	}
	if !strings.Contains(restoreLines(m, 80), "changed after the preview") {
		t.Fatal("drift was not explained")
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "changed after preview" {
		t.Fatalf("drift write occurred: %q err=%v", data, err)
	}
	if got := restoreBackupCount(t, repo, home); got != 1 {
		t.Fatalf("drift wrote a backup: %d, want 1", got)
	}

	// The refreshed preview can be confirmed again, and now revalidation
	// matches the new plan.
	m, _ = m.Update(restoreEnter())
	m, yesCmd := m.Update(restoreKey('y'))
	m, applyCmd = restoreStepCmd(t, m, yesCmd)
	if m.Stage() != RestoreStageRunning || applyCmd == nil {
		t.Fatalf("fresh confirmation did not start a restore: stage=%v cmd nil=%t", m.Stage(), applyCmd == nil)
	}
	m = restoreWriteStep(t, m, applyCmd)
	if out, _ := m.Outcome(); out.Kind != RestoreOutcomeApplied {
		t.Fatalf("fresh confirmation outcome = %v, want applied", out.Kind)
	}
}

// TestRestoreCorruptStaleAbsentAndRefusedStates proves unsafe or unusable
// backups explain recovery and never offer confirmation.
func TestRestoreCorruptStaleAbsentAndRefusedStates(t *testing.T) {
	t.Run("corrupt", func(t *testing.T) {
		repo := t.TempDir()
		home := t.TempDir()
		id := "20260101T000000.000000000Z"
		dir := filepath.Join(repo, "backups", id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		m := newLoadedRestoreModel(t, restoreConfig(repo, home))
		if len(m.Backups()) != 1 || m.Backups()[0].Unusable == "" {
			t.Fatalf("corrupt backup not reported unusable: %+v", m.Backups())
		}
		m, _ = m.Update(restoreEnter())
		if m.Stage() != RestoreStageDetails {
			t.Fatalf("stage = %v, want details", m.Stage())
		}
		text := restoreLines(m, 80)
		if !strings.Contains(text, "Unusable backup") || !strings.Contains(text, "cannot be previewed or restored") {
			t.Fatalf("corrupt details lack recovery:\n%s", text)
		}
		if next, cmd := m.Update(restoreKey('p')); cmd != nil || next.Stage() != RestoreStageDetails {
			t.Fatalf("corrupt backup offered a preview: stage=%v cmd nil=%t", next.Stage(), cmd == nil)
		}
	})

	t.Run("stale home", func(t *testing.T) {
		repo, _, _ := restoreFixture(t)
		otherHome := t.TempDir()
		m := newLoadedRestoreModel(t, restoreConfig(repo, otherHome))
		if len(m.Backups()) != 1 || !strings.Contains(m.Backups()[0].Unusable, "different home") {
			t.Fatalf("stale backup not reported: %+v", m.Backups())
		}
		m, _ = m.Update(restoreEnter())
		text := restoreLines(m, 80)
		if !strings.Contains(text, "different home") || !strings.Contains(text, "Recovery:") {
			t.Fatalf("stale backup lacks recovery:\n%s", text)
		}
		if next, cmd := m.Update(restoreKey('p')); cmd != nil || next.Stage() != RestoreStageDetails {
			t.Fatalf("stale backup offered a preview: stage=%v cmd nil=%t", next.Stage(), cmd == nil)
		}
	})

	t.Run("absent", func(t *testing.T) {
		repo, home, id := restoreFixture(t)
		m := newLoadedRestoreModel(t, restoreConfig(repo, home))
		if err := os.RemoveAll(filepath.Join(repo, "backups", id)); err != nil {
			t.Fatal(err)
		}
		m, _ = m.Update(restoreEnter())
		m, previewCmd := m.Update(restoreEnter())
		m = restoreStep(t, m, previewCmd)
		if m.Stage() != RestoreStagePreview || m.PreviewResult().PreviewToken != "" {
			t.Fatalf("absent backup produced a token: stage=%v token=%q", m.Stage(), m.PreviewResult().PreviewToken)
		}
		text := restoreLines(m, 80)
		if !strings.Contains(text, "absent") || !strings.Contains(text, "No confirmation is available") {
			t.Fatalf("absent backup lacks recovery:\n%s", text)
		}
		if next, cmd := m.Update(restoreEnter()); cmd != nil || next.Stage() != RestoreStagePreview {
			t.Fatalf("absent backup offered confirmation: stage=%v cmd nil=%t", next.Stage(), cmd == nil)
		}
	})

	t.Run("refused destination", func(t *testing.T) {
		repo, home, _ := restoreFixture(t)
		dest := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan")
		if err := os.RemoveAll(dest); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), dest); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		m := newLoadedRestoreModel(t, restoreConfig(repo, home))
		m, _ = m.Update(restoreEnter())
		m, previewCmd := m.Update(restoreEnter())
		m = restoreStep(t, m, previewCmd)
		text := restoreLines(m, 80)
		if m.PreviewResult().PreviewToken != "" || !strings.Contains(text, "refused for safety") {
			t.Fatalf("refused destination produced a token or no recovery:\n%s", text)
		}
		if !strings.Contains(text, "No confirmation is available") {
			t.Fatalf("refused destination offered confirmation:\n%s", text)
		}
	})
}

// TestRestoreStaleResultAndBackAreSafe proves a late read-only result cannot
// overwrite a newer screen state, and backing out never writes.
func TestRestoreStaleResultAndBackAreSafe(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	before := snapshotRestoreHome(t, home)
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m, _ = m.Update(restoreEsc())
	if m.Stage() != RestoreStageDetails {
		t.Fatalf("Esc from preview = %v, want details", m.Stage())
	}
	m = restoreStep(t, m, previewCmd)
	if m.Stage() != RestoreStageDetails || m.PreviewResult().PreviewToken != "" {
		t.Fatalf("stale preview overwrote state: stage=%v token=%q", m.Stage(), m.PreviewResult().PreviewToken)
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))
}

// TestRestoreCancelPathsAreSafe checks that n/Esc before y never writes.
func TestRestoreCancelPathsAreSafe(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	before := snapshotRestoreHome(t, home)
	m, _ = m.Update(restoreEnter())
	m, previewCmd := m.Update(restoreEnter())
	m = restoreStep(t, m, previewCmd)
	m, _ = m.Update(restoreEnter())
	m, cmd := m.Update(restoreKey('n'))
	if cmd != nil || m.Stage() != RestoreStagePreview {
		t.Fatalf("n cancel: stage=%v cmd nil=%t", m.Stage(), cmd == nil)
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))
	m, _ = m.Update(restoreEsc())
	if m.Stage() != RestoreStageDetails {
		t.Fatalf("Esc from preview = %v, want details", m.Stage())
	}
	m, _ = m.Update(restoreEsc())
	if m.Stage() != RestoreStageList {
		t.Fatalf("Esc from details = %v, want list", m.Stage())
	}
	if _, cmd := m.Update(restoreEsc()); cmd == nil {
		t.Fatal("Esc from list did not emit a close request")
	} else if _, ok := cmd().(RestoreCloseMsg); !ok {
		t.Fatalf("Esc from list emitted %T, want RestoreCloseMsg", cmd())
	}
	assertRestoreHomeUnchanged(t, before, snapshotRestoreHome(t, home))
}

// TestRestoreEmptyRootIsValidEmptyState covers a missing backup root.
func TestRestoreEmptyRootIsValidEmptyState(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	m := newLoadedRestoreModel(t, restoreConfig(repo, home))
	if m.Stage() != RestoreStageList || len(m.Backups()) != 0 {
		t.Fatalf("empty root state = %v/%d", m.Stage(), len(m.Backups()))
	}
	text := restoreLines(m, 80)
	if !strings.Contains(text, "No backups found.") || !strings.Contains(text, "valid empty state") {
		t.Fatalf("empty state lacks recovery:\n%s", text)
	}
	if strings.Contains(text, "unavailable") {
		t.Fatalf("empty root reported an error:\n%s", text)
	}
}

// TestRestoreOrderingAndNoFilesystemWorkInUpdate uses injected operations to
// prove deterministic newest-first ordering and that Update/Lines do no work.
func TestRestoreOrderingAndNoFilesystemWorkInUpdate(t *testing.T) {
	calls := struct{ list, preview int }{}
	ops := RestoreOps{
		List: func(repo, home string) ([]skillsync.Backup, error) {
			calls.list++
			return []skillsync.Backup{
				{ID: "20260101T000000.000000000Z", Summary: "older"},
				{ID: "20260301T000000.000000000Z", Summary: "newest"},
				{ID: "20260201T000000.000000000Z", Summary: "middle"},
			}, nil
		},
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			calls.preview++
			return appops.PreviewResult{PreviewToken: "token", Outcome: skillsync.OutcomePreview}, nil
		},
	}
	cfg := TUIConfig{Repo: "", Home: t.TempDir(), Width: 80, Height: 24}
	m := NewRestoreModel(cfg, ops)
	if calls.list != 0 || calls.preview != 0 {
		t.Fatalf("constructor performed work: %+v", calls)
	}
	cmd := m.Init()
	if calls.list != 0 {
		t.Fatal("Init performed the list read instead of scheduling it")
	}
	m = restoreStep(t, m, cmd)
	backups := m.Backups()
	if len(backups) != 3 || backups[0].ID != "20260301T000000.000000000Z" || backups[2].ID != "20260101T000000.000000000Z" {
		t.Fatalf("ordering = %v, want newest first", backupIDs(backups))
	}

	m, _ = m.Update(restoreEnter())
	_ = m.View(80, 24)
	_ = m.Lines(80)
	if calls.preview != 0 {
		t.Fatal("opening details or rendering performed a preview")
	}
	m, previewCmd := m.Update(restoreEnter())
	if calls.preview != 0 {
		t.Fatal("preview ran inside Update")
	}
	m, _ = restoreStepCmd(t, m, previewCmd)
	if calls.preview != 1 {
		t.Fatalf("preview calls = %d, want 1", calls.preview)
	}
}

func backupIDs(backups []skillsync.Backup) []string {
	ids := make([]string, len(backups))
	for i, b := range backups {
		ids[i] = b.ID
	}
	return ids
}

// TestClassifyRestoreOutcome covers every honest result distinction.
func TestClassifyRestoreOutcome(t *testing.T) {
	cases := []struct {
		name   string
		result appops.ApplyResult
		err    error
		want   RestoreOutcomeKind
	}{
		{"applied", appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil, RestoreOutcomeApplied},
		{"no change", appops.ApplyResult{Outcome: skillsync.OutcomeNoChange}, nil, RestoreOutcomeNoChange},
		{"needs preview", appops.ApplyResult{Outcome: skillsync.OutcomeFailed}, appops.ErrNeedsPreview, RestoreOutcomeNeedsPreview},
		{"cancelled", appops.ApplyResult{}, context.Canceled, RestoreOutcomeCancelled},
		{"rolled back", appops.ApplyResult{Rollback: skillsync.RollbackInfo{Attempted: true, Complete: true}}, errors.New("boom"), RestoreOutcomeRolledBack},
		{"incomplete", appops.ApplyResult{Rollback: skillsync.RollbackInfo{Attempted: true, Complete: false}}, errors.New("boom"), RestoreOutcomeIncomplete},
		{"failed", appops.ApplyResult{}, errors.New("boom"), RestoreOutcomeFailed},
		{"unknown success", appops.ApplyResult{Outcome: skillsync.OutcomePreview}, nil, RestoreOutcomeFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyRestoreOutcome(tc.result, tc.err); got != tc.want {
				t.Fatalf("classify = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRestorePlainAndBoundsSafe proves the screen uses no styles in plain mode
// and never emits a row wider than the requested cells.
func TestRestorePlainAndBoundsSafe(t *testing.T) {
	repo, home, _ := restoreFixture(t)
	cfg := restoreConfig(repo, home)
	m := newLoadedRestoreModel(t, cfg)

	stages := []RestoreModel{m}
	details := m
	details, _ = details.Update(restoreEnter())
	stages = append(stages, details)
	preview, cmd := details.Update(restoreEnter())
	preview = restoreStep(t, preview, cmd)
	stages = append(stages, preview)
	confirm, _ := preview.Update(restoreEnter())
	stages = append(stages, confirm)
	result := confirm
	result.stage = RestoreStageResult
	result.result = &RestoreOutcome{Kind: RestoreOutcomeApplied, BackupID: "20260101T000000.000000000Z", SafetyBackupID: "20260102T000000.000000000Z", SafetyBackupPath: filepath.Join(home, ".agentic-sdd", "backups", "20260102T000000.000000000Z")}
	stages = append(stages, result)

	for _, width := range []int{20, 40, 80} {
		for i, stage := range stages {
			lines := stage.Lines(width)
			if len(lines) == 0 {
				t.Fatalf("stage %d width %d produced no lines", i, width)
			}
			for _, line := range lines {
				if strings.Contains(line, "\x1b") {
					t.Fatalf("plain mode emitted an escape sequence at stage %d width %d: %q", i, width, line)
				}
				if got := ansi.StringWidth(line); got > width {
					t.Fatalf("stage %d width %d line width %d: %q", i, width, got, line)
				}
			}
		}
	}
	if got := m.View(0, 10); got != "" {
		t.Fatalf("View(0,10) = %q, want empty", got)
	}
}
