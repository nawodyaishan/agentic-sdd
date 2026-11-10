package tui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// helper to simulate key presses with string or code
func pressKey(m InstallModel, key string) (InstallModel, tea.Cmd) {
	var msg tea.KeyPressMsg
	switch key {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\r"}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEsc}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case " ":
		msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	default:
		if len(key) == 1 {
			msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
		} else {
			msg = tea.KeyPressMsg{Text: key}
		}
	}
	updated, cmd := m.Update(msg)
	return updated.(InstallModel), cmd
}

// fakePreviewOp builds an in-memory preview result with the provided planned changes
func fakePreviewOp(planned []skillsync.ResultChange, token string, err error) PreviewOp {
	return func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		if err != nil {
			return appops.PreviewResult{
				Operation: skillsync.OperationApply,
				Outcome:   skillsync.OutcomeFailed,
			}, err
		}
		return appops.PreviewResult{
			Operation:    skillsync.OperationApply,
			Outcome:      skillsync.OutcomePreview,
			Planned:      planned,
			PreviewToken: token,
		}, nil
	}
}

func TestInstallCancelAndBack(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	// Step 0: Cancel from Clients
	m1, _ := pressKey(m, "esc")
	if m1.Step() != InstallStepResult {
		t.Fatalf("after esc at clients step = %v, want InstallStepResult", m1.Step())
	}
	if m1.Outcome() != InstallOutcomeCancelled {
		t.Fatalf("after esc at clients outcome = %v, want InstallOutcomeCancelled", m1.Outcome())
	}

	// Back from Options to Clients
	m = NewInstallModel(cfg)
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepOptions {
		t.Fatalf("after enter at clients step = %v, want InstallStepOptions", m.Step())
	}
	m, _ = pressKey(m, "b")
	if m.Step() != InstallStepClients {
		t.Fatalf("after 'b' at options step = %v, want InstallStepClients", m.Step())
	}

	// Back from Review to Options
	m, _ = pressKey(m, "enter") // to options
	m, _ = pressKey(m, "enter") // to review
	if m.Step() != InstallStepReview {
		t.Fatalf("after enter at options step = %v, want InstallStepReview", m.Step())
	}
	m, _ = pressKey(m, "esc")
	if m.Step() != InstallStepOptions {
		t.Fatalf("after esc at review step = %v, want InstallStepOptions", m.Step())
	}

	// Back from Preview to Review
	m, _ = pressKey(m, "enter") // to review
	m.SetPreviewOp(fakePreviewOp(nil, "token-1", nil))
	m, cmd := pressKey(m, "enter") // trigger preview
	if cmd == nil {
		t.Fatal("expected preview cmd from review enter")
	}
	msg := cmd()
	updated, _ := m.Update(msg)
	m = updated.(InstallModel)
	if m.Step() != InstallStepPreview {
		t.Fatalf("step after preview msg = %v, want InstallStepPreview", m.Step())
	}
	m, _ = pressKey(m, "b")
	if m.Step() != InstallStepReview {
		t.Fatalf("after 'b' at preview step = %v, want InstallStepReview", m.Step())
	}

	// Back from Confirm to Preview
	m, cmd = pressKey(m, "enter")
	msg = cmd()
	updated, _ = m.Update(msg)
	m = updated.(InstallModel)
	m, _ = pressKey(m, "enter") // to confirm
	if m.Step() != InstallStepConfirm {
		t.Fatalf("step after enter in preview = %v, want InstallStepConfirm", m.Step())
	}
	m, _ = pressKey(m, "n") // safe No / back
	if m.Step() != InstallStepPreview {
		t.Fatalf("after 'n' at confirm step = %v, want InstallStepPreview", m.Step())
	}
	if !m.ConfirmNoSelected() {
		t.Fatal("expected ConfirmNoSelected() to be true")
	}
}

func TestInstallNoClient(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	// Toggle off all supported clients
	for range m.supportedClients {
		m, _ = pressKey(m, " ")
		m, _ = pressKey(m, "down")
	}
	if len(m.SelectedClients()) != 0 {
		t.Fatalf("selected clients = %v, want 0", m.SelectedClients())
	}

	// Advance must be blocked
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepClients {
		t.Fatalf("step after enter with 0 clients = %v, want InstallStepClients", m.Step())
	}
	if m.ValidationError() == "" {
		t.Fatal("expected validation error when advancing with 0 clients")
	}
	if !strings.Contains(m.ValidationError(), "At least one client must be selected") {
		t.Fatalf("unexpected validation error: %q", m.ValidationError())
	}

	// SetClients validation
	if err := m.SetClients([]string{}); err == nil {
		t.Fatal("expected error from SetClients with empty slice")
	}
	if err := m.SetClients([]string{"nonexistent-client"}); err == nil {
		t.Fatal("expected error from SetClients with unknown client")
	}
}

func TestInstallInvalidPreset(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	// Advance to options (WithOpenCode is false)
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepOptions {
		t.Fatalf("step = %v, want InstallStepOptions", m.Step())
	}

	// Move cursor to Free-first preset and press space
	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, " ")
	if m.OpenCodeFreeFirst() {
		t.Fatal("Free-first preset should not be enabled when WithOpenCode is false")
	}
	if m.ValidationError() == "" || !strings.Contains(m.ValidationError(), "requires the OpenCode pack") {
		t.Fatalf("expected OpenCode pack required validation error, got: %q", m.ValidationError())
	}

	// SetOptions programmatic validation
	if err := m.SetOptions(false, true); err == nil {
		t.Fatal("expected error from SetOptions(false, true)")
	}

	// Enable OpenCode, then enable Free-first
	m, _ = pressKey(m, "up")
	m, _ = pressKey(m, " ") // toggle WithOpenCode -> true
	if !m.WithOpenCode() {
		t.Fatal("WithOpenCode should be true")
	}
	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, " ") // toggle FreeFirst -> true
	if !m.OpenCodeFreeFirst() {
		t.Fatal("OpenCodeFreeFirst should be true")
	}

	// Toggling WithOpenCode off must reset Free-first to false
	m, _ = pressKey(m, "up")
	m, _ = pressKey(m, " ") // toggle WithOpenCode -> false
	if m.WithOpenCode() {
		t.Fatal("WithOpenCode should be false")
	}
	if m.OpenCodeFreeFirst() {
		t.Fatal("OpenCodeFreeFirst should be automatically disabled when WithOpenCode is disabled")
	}
}

func TestInstallChangedSettingsInvalidatesConfirmation(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	m.SetPreviewOp(fakePreviewOp(nil, "token-initial", nil))

	// Advance to preview and confirm
	m, _ = pressKey(m, "enter")    // to options
	m, _ = pressKey(m, "enter")    // to review
	m, cmd := pressKey(m, "enter") // trigger preview
	updated, _ := m.Update(cmd())
	m = updated.(InstallModel)
	m, _ = pressKey(m, "enter") // to confirm

	if m.Step() != InstallStepConfirm {
		t.Fatalf("step = %v, want InstallStepConfirm", m.Step())
	}
	if m.PreviewToken() != "token-initial" {
		t.Fatalf("token = %q, want token-initial", m.PreviewToken())
	}

	// Go back to Options and change setting
	m, _ = pressKey(m, "b") // to preview
	m, _ = pressKey(m, "b") // to review
	m, _ = pressKey(m, "b") // to options
	m, _ = pressKey(m, " ") // toggle WithOpenCode

	if m.PreviewToken() != "" {
		t.Fatalf("token after change = %q, want empty", m.PreviewToken())
	}
	if m.PreviewResult() != nil {
		t.Fatal("previewResult should be nil after change")
	}
	if !m.ConfirmNoSelected() {
		t.Fatal("confirmNoSelected should be reset to true")
	}
}

func TestInstallFocusAndResizeRetainsSafeNo(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	m.SetPreviewOp(fakePreviewOp(nil, "token-safe", nil))

	// Advance to confirm
	m, _ = pressKey(m, "enter")    // options
	m, _ = pressKey(m, "enter")    // review
	m, cmd := pressKey(m, "enter") // preview
	updated, _ := m.Update(cmd())
	m = updated.(InstallModel)
	m, _ = pressKey(m, "enter") // confirm

	if m.Step() != InstallStepConfirm {
		t.Fatalf("step = %v, want InstallStepConfirm", m.Step())
	}
	if !m.ConfirmNoSelected() {
		t.Fatal("expected ConfirmNoSelected() to be true")
	}

	// Resize must preserve safe No
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(InstallModel)
	if !m.ConfirmNoSelected() {
		t.Fatal("expected ConfirmNoSelected() to remain true after resize")
	}

	// Enter must NOT confirm or emit a write request.
	m, cmd = pressKey(m, "enter")
	if cmd != nil {
		t.Fatal("Enter in confirm must not return a command")
	}
	if m.Step() != InstallStepConfirm {
		t.Fatalf("after Enter, step = %v, want InstallStepConfirm", m.Step())
	}
}

func TestInstallPreviewAndApplyCallCounts(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	var previewCalls int32

	m.SetPreviewOp(func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		atomic.AddInt32(&previewCalls, 1)
		return appops.PreviewResult{
			Operation:    skillsync.OperationApply,
			Outcome:      skillsync.OutcomePreview,
			PreviewToken: "token-count-test",
		}, nil
	})

	// Step through Clients, Options, Review: preview=0.
	m, _ = pressKey(m, "enter") // Options
	m, _ = pressKey(m, "enter") // Review
	if atomic.LoadInt32(&previewCalls) != 0 {
		t.Fatalf("before preview: preview=%d", previewCalls)
	}

	// Trigger preview
	m, cmd := pressKey(m, "enter")
	if cmd == nil {
		t.Fatal("expected preview command")
	}
	msg := cmd()
	if atomic.LoadInt32(&previewCalls) != 1 {
		t.Fatalf("previewCalls = %d, want 1", previewCalls)
	}
	updated, _ := m.Update(msg)
	m = updated.(InstallModel)

	// In preview, enter to confirm
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepConfirm {
		t.Fatalf("step = %v, want InstallStepConfirm", m.Step())
	}

	// In confirm, enter does not start a write
	m, cmd = pressKey(m, "enter")
	if cmd != nil {
		t.Fatal("expected nil command from Enter in confirm")
	}

	// Only affirmative 'y' emits the single StartWriteMsg request.
	m, applyCmd := pressKey(m, "y")
	if applyCmd == nil {
		t.Fatal("expected a write request command from 'y'")
	}
	if m.Step() != InstallStepRunning {
		t.Fatalf("step after 'y' = %v, want InstallStepRunning", m.Step())
	}
	writeMsg, ok := applyCmd().(StartWriteMsg)
	if !ok {
		t.Fatalf("confirm command returned %T, want StartWriteMsg", applyCmd())
	}
	if writeMsg.Request.Kind != WriteInstall || writeMsg.Request.Token != "token-count-test" {
		t.Fatalf("write request = %+v, want install with the current token", writeMsg.Request)
	}
	if writeMsg.Request.Install.Apply() != true {
		t.Fatal("write request must be the apply variant")
	}

	// While running, pressing 'y' or enter does NOT emit a second request.
	m, dupCmd := pressKey(m, "y")
	if dupCmd != nil {
		t.Fatal("expected nil command when pressing 'y' while running")
	}
	m, dupCmd = pressKey(m, "enter")
	if dupCmd != nil {
		t.Fatal("expected nil command when pressing enter while running")
	}

	// The host result ends the journey exactly once.
	updated, _ = m.Update(WriteResultMsg{
		ID:     appops.OperationID{Request: 1, Generation: 1},
		Kind:   WriteInstall,
		Result: appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeApplied, BackupID: "backup-123", BackupPath: "/tmp/backup-123"},
	})
	m = updated.(InstallModel)
	if m.Step() != InstallStepResult {
		t.Fatalf("step after WriteResultMsg = %v, want InstallStepResult", m.Step())
	}
	if m.Outcome() != InstallOutcomeApplied {
		t.Fatalf("outcome = %v, want InstallOutcomeApplied", m.Outcome())
	}
}

func TestInstallDoubleEnter(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	m.SetPreviewOp(fakePreviewOp(nil, "token-double", nil))

	// Go to Preview
	m, _ = pressKey(m, "enter")
	m, _ = pressKey(m, "enter")
	m, cmd := pressKey(m, "enter")
	updated, _ := m.Update(cmd())
	m = updated.(InstallModel)

	// First Enter in Preview -> advances to Confirm
	m, _ = pressKey(m, "enter")
	if m.Step() != InstallStepConfirm {
		t.Fatalf("step after 1st enter = %v, want InstallStepConfirm", m.Step())
	}

	// Second Enter in Confirm -> MUST NOT confirm or emit a write request
	m, cmd = pressKey(m, "enter")
	if m.Step() != InstallStepConfirm {
		t.Fatalf("step after 2nd enter = %v, want InstallStepConfirm", m.Step())
	}
	if cmd != nil {
		t.Fatal("double Enter emitted a command; want none")
	}

	// Third Enter in Confirm -> MUST NOT confirm
	m, cmd = pressKey(m, "enter")
	if m.Step() != InstallStepConfirm {
		t.Fatalf("step after 3rd enter = %v, want InstallStepConfirm", m.Step())
	}
	if cmd != nil {
		t.Fatal("third Enter emitted a command; want none")
	}
}

func TestInstallStaleAsyncResult(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	// Generate preview cmd (generation 1)
	m.SetPreviewOp(fakePreviewOp(nil, "token-stale-1", nil))
	m, _ = pressKey(m, "enter") // options
	m, _ = pressKey(m, "enter") // review
	m, cmd1 := pressKey(m, "enter")
	msg1 := cmd1()

	// Invalidate and start generation 2
	m.SetHome("/tmp/otherhome")
	m.SetPreviewOp(fakePreviewOp(nil, "token-fresh-2", nil))
	m, cmd2 := pressKey(m, "enter")
	_ = cmd2

	// Deliver stale msg1 (generation 1)
	updated, _ := m.Update(msg1)
	m = updated.(InstallModel)
	if m.PreviewToken() == "token-stale-1" {
		t.Fatal("stale preview result was accepted")
	}

	// Deliver fresh msg2
	msg2 := cmd2()
	updated, _ = m.Update(msg2)
	m = updated.(InstallModel)
	if m.PreviewToken() != "token-fresh-2" {
		t.Fatalf("previewToken = %q, want token-fresh-2", m.PreviewToken())
	}
}

func TestInstallExactPreviewPaths(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)

	planned := []skillsync.ResultChange{
		{Client: "claude", ID: "router", Path: "/custom/home/.claude/skills/router", Action: skillsync.ChangeInstall},
		{Client: "agents", ID: "planner", Path: "/custom/home/.agents/skills/planner", Action: skillsync.ChangeReplace},
		{Client: "codex", ID: "coder", Path: "/custom/home/.codex/skills/coder", Action: skillsync.ChangeUpToDate},
		{Client: "agy", ID: "worker", Path: "/custom/home/.gemini/antigravity-cli/skills/worker", Action: skillsync.ChangeRemove},
	}

	m.SetPreviewOp(fakePreviewOp(planned, "token-paths", nil))
	m, _ = pressKey(m, "enter") // options
	m, _ = pressKey(m, "enter") // review
	m, cmd := pressKey(m, "enter")
	updated, _ := m.Update(cmd())
	m = updated.(InstallModel)

	// Wide render (width 80)
	renderedWide := m.Render(80, 0)
	for _, p := range planned {
		if !strings.Contains(renderedWide, p.Path) {
			t.Errorf("rendered wide output missing path: %s", p.Path)
		}
		if !strings.Contains(renderedWide, p.ID) {
			t.Errorf("rendered wide output missing ID: %s", p.ID)
		}
	}

	// Narrow render (width 40): paths must still be present (wrapped)
	renderedNarrow := m.Render(40, 0)
	for _, p := range planned {
		// Even if wrapped, parts of the path must be present
		if !strings.Contains(renderedNarrow, p.ID) {
			t.Errorf("rendered narrow output missing ID: %s", p.ID)
		}
	}
}

func TestInstallNoWritesInPreview(t *testing.T) {
	tempHome := t.TempDir()
	cfg := TUIConfig{Home: tempHome}
	m := NewInstallModel(cfg)

	// Run real PreviewInstall against tempHome
	m, _ = pressKey(m, "enter")    // options
	m, _ = pressKey(m, "enter")    // review
	m, cmd := pressKey(m, "enter") // run preview
	if cmd == nil {
		t.Fatal("expected preview cmd")
	}
	msg := cmd()
	updated, _ := m.Update(msg)
	m = updated.(InstallModel)

	if m.Step() != InstallStepPreview {
		t.Fatalf("step = %v, want InstallStepPreview", m.Step())
	}

	// Assert NO files or backup directory was created
	backupDir := filepath.Join(tempHome, ".agentic-sdd", "backups")
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("preview wrote backup directory at %s", backupDir)
	}

	claudeDir := filepath.Join(tempHome, ".claude")
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Fatalf("preview wrote client directory at %s", claudeDir)
	}
}

func TestInstallResultStates(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}

	// 1. Applied
	m := NewInstallModel(cfg)
	m.step = InstallStepConfirm
	m.previewToken = "token"
	m, cmd := pressKey(m, "y")
	_ = cmd
	updated, _ := m.Update(WriteResultMsg{
		ID:   appops.OperationID{Request: 1, Generation: 1},
		Kind: WriteInstall,
		Result: appops.ApplyResult{
			Outcome:    skillsync.OutcomeApplied,
			BackupID:   "20261110-test-backup",
			BackupPath: "/tmp/backups/20261110-test-backup",
			Applied:    []skillsync.ResultChange{{ID: "sdd-implement"}},
		},
	})
	m = updated.(InstallModel)
	if m.Outcome() != InstallOutcomeApplied {
		t.Fatalf("outcome = %v, want InstallOutcomeApplied", m.Outcome())
	}
	view := m.Render(80, 0)
	if !strings.Contains(view, "Installation Applied Successfully") {
		t.Errorf("view missing success title: %s", view)
	}
	if !strings.Contains(view, "20261110-test-backup") {
		t.Errorf("view missing backup ID: %s", view)
	}
	if !strings.Contains(view, "agentic-sdd-router") {
		t.Errorf("view missing router recommendation: %s", view)
	}

	// 2. No Change
	m = NewInstallModel(cfg)
	m.step = InstallStepConfirm
	m.previewToken = "token"
	m, _ = pressKey(m, "y")
	updated, _ = m.Update(WriteResultMsg{
		ID:   appops.OperationID{Request: 1, Generation: 1},
		Kind: WriteInstall,
		Result: appops.ApplyResult{
			Outcome: skillsync.OutcomeNoChange,
		},
	})
	m = updated.(InstallModel)
	if m.Outcome() != InstallOutcomeNoChange {
		t.Fatalf("outcome = %v, want InstallOutcomeNoChange", m.Outcome())
	}
	view = m.Render(80, 0)
	if !strings.Contains(view, "No Changes Needed") {
		t.Errorf("view missing no-change title: %s", view)
	}

	// 3. Refused (ErrNeedsPreview)
	m = NewInstallModel(cfg)
	m.step = InstallStepConfirm
	m.previewToken = "token"
	m, _ = pressKey(m, "y")
	updated, _ = m.Update(WriteResultMsg{
		ID:   appops.OperationID{Request: 1, Generation: 1},
		Kind: WriteInstall,
		Err:  appops.ErrNeedsPreview,
		Result: appops.ApplyResult{
			Outcome: skillsync.OutcomeFailed,
		},
	})
	m = updated.(InstallModel)
	if m.Step() != InstallStepPreview {
		t.Fatalf("step after needs-preview = %v, want InstallStepPreview", m.Step())
	}
	if m.PreviewToken() != "" {
		t.Fatal("needs-preview must clear the stale confirmation token")
	}
	view = m.Render(80, 0)
	if !strings.Contains(strings.ToLower(view), "fresh preview") {
		t.Errorf("view missing fresh-preview explanation: %s", view)
	}

	// 4. Failed with Complete Rollback
	m = NewInstallModel(cfg)
	m.step = InstallStepConfirm
	m.previewToken = "token"
	m, _ = pressKey(m, "y")
	updated, _ = m.Update(WriteResultMsg{
		ID:   appops.OperationID{Request: 1, Generation: 1},
		Kind: WriteInstall,
		Err:  errors.New("disk full"),
		Result: appops.ApplyResult{
			Outcome:    skillsync.OutcomeFailed,
			BackupID:   "pre-fail-backup",
			BackupPath: "/tmp/backup-fail",
			Rollback: skillsync.RollbackInfo{
				Attempted: true,
				Complete:  true,
			},
		},
	})
	m = updated.(InstallModel)
	if m.Outcome() != InstallOutcomeFailed {
		t.Fatalf("outcome = %v, want InstallOutcomeFailed", m.Outcome())
	}
	view = m.Render(80, 0)
	if !strings.Contains(view, "Installation Failed") {
		t.Errorf("view missing failed title: %s", view)
	}
	if !strings.Contains(view, "Rollback status: Succeeded") {
		t.Errorf("view missing rollback succeeded: %s", view)
	}
	if !strings.Contains(view, "pre-fail-backup") {
		t.Errorf("view missing backup reference: %s", view)
	}

	// 5. Failed with Incomplete Rollback
	m = NewInstallModel(cfg)
	m.step = InstallStepConfirm
	m.previewToken = "token"
	m, _ = pressKey(m, "y")
	updated, _ = m.Update(WriteResultMsg{
		ID:   appops.OperationID{Request: 1, Generation: 1},
		Kind: WriteInstall,
		Err:  errors.New("io error"),
		Result: appops.ApplyResult{
			Outcome: skillsync.OutcomeFailed,
			Rollback: skillsync.RollbackInfo{
				Attempted: true,
				Complete:  false,
				Error:     "permission denied",
			},
		},
	})
	m = updated.(InstallModel)
	view = m.Render(80, 0)
	if !strings.Contains(view, "INCOMPLETE") {
		t.Errorf("view missing incomplete warning: %s", view)
	}

	// 6. Cancelled
	m = NewInstallModel(cfg)
	m, _ = pressKey(m, "q")
	if m.Outcome() != InstallOutcomeCancelled {
		t.Fatalf("outcome = %v, want InstallOutcomeCancelled", m.Outcome())
	}
	view = m.Render(80, 0)
	if !strings.Contains(view, "Installation Cancelled") {
		t.Errorf("view missing cancelled title: %s", view)
	}
}

func TestInstallPreExecutionPolicyDisplayed(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	m.step = InstallStepConfirm

	view := strings.ToLower(m.Render(80, 0))
	if !strings.Contains(view, "writes cannot be interrupted midway") {
		t.Errorf("confirm view missing policy text: %s", view)
	}
	if !strings.Contains(view, "wait for the active write") {
		t.Errorf("confirm view missing safe exit text: %s", view)
	}
	if !strings.Contains(view, "sigkill") {
		t.Errorf("confirm view missing SIGKILL warning: %s", view)
	}
}

func TestInstallPlainModeUnstyledAndBoundsSafe(t *testing.T) {
	cfg := TUIConfig{
		Home:  "/tmp/fakehome",
		Color: false,
	}
	m := NewInstallModel(cfg)

	// Check that color=false produces unstyled output (no ANSI SGR escapes)
	steps := []InstallStep{
		InstallStepClients,
		InstallStepOptions,
		InstallStepReview,
		InstallStepConfirm,
		InstallStepRunning,
		InstallStepResult,
	}

	for _, step := range steps {
		m.step = step
		view := m.Render(80, 0)
		stripped := ansi.Strip(view)
		if view != stripped {
			t.Errorf("step %v produced ANSI styling in plain mode", step)
		}
	}

	// Bounds safety at small dimensions: below 40x8 must warn to resize
	smallView := m.Render(39, 8)
	if !strings.Contains(smallView, "Terminal too small") {
		t.Errorf("expected small terminal warning at 39x8, got: %s", smallView)
	}

	shortView := m.Render(80, 7)
	if !strings.Contains(shortView, "Terminal too small") {
		t.Errorf("expected small terminal warning at 80x7, got: %s", shortView)
	}

	tinyView := m.Render(30, 6)
	if !strings.Contains(tinyView, "Terminal too small") {
		t.Errorf("expected small terminal warning at 30x6, got: %s", tinyView)
	}
}

func TestInstallOpenCodeProfilesSeparate(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	if err := m.SetOptions(true, false); err != nil {
		t.Fatalf("SetOptions error: %v", err)
	}

	m.step = InstallStepReview
	view := m.Render(80, 0)

	if !strings.Contains(view, "OpenCode Profiles & Presets (Separate)") {
		t.Errorf("review missing separate OpenCode section: %s", view)
	}
	if !strings.Contains(view, OpenCodeAgentsDir("/tmp/fakehome")) {
		t.Errorf("review missing OpenCode agents destination path: %s", view)
	}
}

func TestInstallScrollingAndRestart(t *testing.T) {
	cfg := TUIConfig{Home: "/tmp/fakehome"}
	m := NewInstallModel(cfg)
	m.step = InstallStepResult
	m.outcome = InstallOutcomeApplied

	// Scroll down and up
	m, _ = pressKey(m, "j")
	if m.scroll != 1 {
		t.Fatalf("scroll after 'j' = %d, want 1", m.scroll)
	}
	m, _ = pressKey(m, "k")
	if m.scroll != 0 {
		t.Fatalf("scroll after 'k' = %d, want 0", m.scroll)
	}

	// Press 'r' to restart configuration
	m, _ = pressKey(m, "r")
	if m.Step() != InstallStepClients {
		t.Fatalf("step after 'r' = %v, want InstallStepClients", m.Step())
	}
	if m.Outcome() != InstallOutcomeNone {
		t.Fatalf("outcome after 'r' = %v, want InstallOutcomeNone", m.Outcome())
	}
}
