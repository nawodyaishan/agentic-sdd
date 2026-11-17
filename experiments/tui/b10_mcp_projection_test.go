package tui

// B10/T22 truthful outcome projection tests (Feature 022, AC-13 TUI portion).
//
// The MCP child screen and the root notice must describe one structured write
// result the same way: an applied write, a no-op, a completed or incomplete
// rollback, a partial write and a log failure. A no-op never reports a backup,
// and a failure keeps the facts the engine reported instead of replacing them
// with one generic sentence.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// b10MCPApp builds a root shell whose MCP Tools screen uses the supplied seams,
// so no test reads a real home or configuration.
func b10MCPApp(t *testing.T, reader MCPReader, planner MCPPlanner) App {
	t.Helper()
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()
	app := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{
		MCPReader:  reader,
		MCPPlanner: planner,
	})
	next, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated
}

// b10DeliverMCPResult routes one MCP write result through the root shell the way
// the operation host does: the accepted write slot carries the identity first,
// so a stale or foreign result cannot be applied.
func b10DeliverMCPResult(t *testing.T, app App, msg WriteResultMsg) (App, tea.Cmd) {
	t.Helper()
	msg.ID = appops.OperationID{Request: 1, Generation: 1}
	app.write = &writeState{id: msg.ID, kind: msg.Kind}
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated, cmd
}

// b10RenderMCP renders the MCP Tools screen inside the root viewport.
func b10RenderMCP(app App) string {
	app.activeRoute = RouteInstall
	app.installSection = installSectionMCP
	app.focus = FocusContent
	return app.View().Content
}

// TestB10MCPProjectionAgreesWithRootNotice proves the child outcome and the root
// notice come from one projection: same headline, same detail, for every
// structured case the feature names.
func TestB10MCPProjectionAgreesWithRootNotice(t *testing.T) {
	cases := []struct {
		name        string
		msg         WriteResultMsg
		headline    string
		backupID    string
		rollback    string
		mustRender  []string
		neverRender []string
	}{
		{
			name: "applied",
			msg: WriteResultMsg{
				Kind: WriteMCPApply,
				Result: appops.ApplyResult{
					Outcome:    skillsync.OutcomeApplied,
					BackupID:   "20260101T000000.000000000Z",
					BackupPath: "/home/.agentic-sdd/mcp/backups/20260101T000000.000000000Z",
					Applied:    []skillsync.ResultChange{{ID: "codegraph", Kind: "mcp"}},
				},
				LogPath: "/home/.agentic-sdd/logs/apply.log",
			},
			headline:   "completed",
			backupID:   "20260101T000000.000000000Z",
			rollback:   "none",
			mustRender: []string{"Last operation: completed", "Wrote 1 planned changes.", "Backup ID: 20260101T000000.000000000Z", "Operation log:"},
		},
		{
			name: "no-op reports no backup",
			msg: WriteResultMsg{
				Kind:    WriteMCPApply,
				Result:  appops.ApplyResult{Outcome: skillsync.OutcomeNoChange},
				LogPath: "/home/.agentic-sdd/logs/apply.log",
			},
			headline:    "no changes",
			backupID:    "",
			rollback:    "none",
			mustRender:  []string{"Last operation: no changes", "Every planned destination was already up to date."},
			neverRender: []string{"Backup ID:", "Backup path:", "recorded"},
		},
		{
			name: "rolled back after a failure",
			msg: WriteResultMsg{
				Kind: WriteMCPRestore,
				Result: appops.ApplyResult{
					Outcome:  skillsync.OutcomeFailed,
					BackupID: "safety-backup",
					Rollback: skillsync.RollbackInfo{Attempted: true, Complete: true},
				},
				Err:     errors.New("mcp restore failed after replacing the target"),
				LogPath: "/home/.agentic-sdd/logs/restore.log",
			},
			headline:   "rolled back",
			backupID:   "safety-backup",
			rollback:   "complete",
			mustRender: []string{"Last operation: rolled back", "every committed change was undone", "Rollback: complete", "Engine: mcp restore failed"},
		},
		{
			name: "incomplete rollback keeps the partial facts",
			msg: WriteResultMsg{
				Kind: WriteMCPApply,
				Result: appops.ApplyResult{
					Outcome:    skillsync.OutcomeFailed,
					BackupID:   "recovery-backup",
					BackupPath: "/home/.agentic-sdd/mcp/backups/recovery-backup",
					Rollback:   skillsync.RollbackInfo{Attempted: true, Error: "target is read-only"},
					Applied:    []skillsync.ResultChange{{ID: "context7", Kind: "mcp"}},
				},
				Err:     errors.New("mcp apply needs recovery"),
				LogPath: "/home/.agentic-sdd/logs/apply.log",
			},
			headline:   "incomplete",
			backupID:   "recovery-backup",
			rollback:   "incomplete",
			mustRender: []string{"Last operation: incomplete", "Rollback: INCOMPLETE", "Reported written before the failure: context7", "Backup path:"},
		},
		{
			name: "log failure stays visible",
			msg: WriteResultMsg{
				Kind: WriteMCPApply,
				Result: appops.ApplyResult{
					Outcome:  skillsync.OutcomeApplied,
					BackupID: "bak-log",
				},
				LogPath: "/home/.agentic-sdd/logs/apply.log",
				LogErr:  errors.New("sync: input/output error"),
			},
			headline:   "completed",
			backupID:   "bak-log",
			rollback:   "none",
			mustRender: []string{"The operation log could not be flushed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := b10MCPApp(t, &fakeMCPReader{}, &fakeMCPPlanner{})
			next, _ := b10DeliverMCPResult(t, app, tc.msg)

			outcome := next.mcp.Outcome()
			if outcome.Headline != tc.headline {
				t.Fatalf("child headline = %q, want %q", outcome.Headline, tc.headline)
			}
			if outcome.BackupID != tc.backupID {
				t.Fatalf("child backup = %q, want %q (a no-op records none)", outcome.BackupID, tc.backupID)
			}
			if outcome.Rollback != tc.rollback {
				t.Fatalf("child rollback = %q, want %q", outcome.Rollback, tc.rollback)
			}

			// One projection: the root notice is exactly the child's headline and
			// detail, so the two screens cannot describe the same write
			// differently.
			wantNotice := outcome.Headline + ": " + outcome.Detail
			if next.Notice() != wantNotice {
				t.Fatalf("root notice = %q, want the child projection %q", next.Notice(), wantNotice)
			}

			rendered := b10RenderMCP(next)
			for _, want := range tc.mustRender {
				if !strings.Contains(rendered, want) {
					t.Errorf("MCP screen missing %q:\n%s", want, rendered)
				}
			}
			for _, never := range tc.neverRender {
				if strings.Contains(rendered, never) {
					t.Errorf("MCP screen invented %q:\n%s", never, rendered)
				}
			}
		})
	}
}

// TestB10MCPRefusalDoesNotClaimAWrite proves a refused request reports the
// refusal and never a result, backup or rollback.
func TestB10MCPRefusalDoesNotClaimAWrite(t *testing.T) {
	app := b10MCPApp(t, &fakeMCPReader{}, &fakeMCPPlanner{})
	next, cmd := app.Update(WriteRefusedMsg{
		Kind:   WriteMCPApply,
		Err:    ErrWriteInProgress,
		Reason: "a write is already running",
	})
	updated := next.(App)
	if cmd != nil {
		t.Fatal("a refusal scheduled follow-up work")
	}
	outcome := updated.mcp.Outcome()
	if outcome.Headline != "write refused" || outcome.Detail != "a write is already running" {
		t.Fatalf("refusal outcome = %+v", outcome)
	}
	if outcome.BackupID != "" || outcome.Rollback != "" {
		t.Fatalf("refusal invented structured write facts: %+v", outcome)
	}
}

// TestB10MCPSummaryKeepsEngineError proves the one shared summary carries the
// engine's own error text, so a failure is not flattened into a generic
// sentence that hides the cause from both the root notice and the child.
func TestB10MCPSummaryKeepsEngineError(t *testing.T) {
	msg := WriteResultMsg{
		Kind:   WriteMCPApply,
		Result: appops.ApplyResult{Outcome: skillsync.OutcomeFailed},
		Err:    errors.New("mcp apply failed before writing the configuration; nothing changed"),
	}
	summary := SummarizeWriteResult(msg)
	if !strings.Contains(summary.Detail, "mcp apply failed before writing") {
		t.Fatalf("summary dropped the engine error: %q", summary.Detail)
	}
	if summary.BackupID != "" {
		t.Fatalf("summary invented a backup: %q", summary.BackupID)
	}
}

// TestB10ProjectMCPApplyResultPreservesFactsOnError proves the host projection
// keeps the engine's structured result when the write fails: the host must not
// turn a recoverable failure into "no facts at all".
func TestB10ProjectMCPApplyResultPreservesFactsOnError(t *testing.T) {
	engineResult := &skillsync.MCPApplyResult{
		TargetPath:      "/home/.config/opencode/opencode.json",
		BackupID:        "20260101T000000.000000000Z",
		BackupPath:      "/home/.agentic-sdd/mcp/backups/20260101T000000.000000000Z",
		InstalledDigest: "sha256:abc",
		Applied:         []string{"codegraph", "exa"},
		Rollback:        skillsync.RollbackInfo{Attempted: true, Complete: false, Error: "target is read-only"},
	}
	writeErr := errors.New("mcp apply needs recovery")

	got, err := projectMCPApplyResult(skillsync.OperationApply, "mcp apply", engineResult, writeErr)
	if !errors.Is(err, writeErr) {
		t.Fatalf("error = %v, want the engine error", err)
	}
	if got.Outcome != skillsync.OutcomeFailed {
		t.Fatalf("outcome = %q, want failed", got.Outcome)
	}
	if got.BackupID != engineResult.BackupID || got.BackupPath != engineResult.BackupPath {
		t.Fatalf("backup facts lost: %+v", got)
	}
	if !got.Rollback.Attempted || got.Rollback.Complete {
		t.Fatalf("rollback facts lost: %+v", got.Rollback)
	}
	if len(got.Applied) != 2 {
		t.Fatalf("partial applied facts lost: %+v", got.Applied)
	}

	// A missing result with no error is still a failure: a write that reports
	// nothing cannot be presented as successful.
	got, err = projectMCPApplyResult(skillsync.OperationApply, "mcp apply", nil, nil)
	if err == nil {
		t.Fatal("a nil engine result was accepted as success")
	}
	if got.Outcome != skillsync.OutcomeFailed {
		t.Fatalf("nil result outcome = %q, want failed", got.Outcome)
	}

	// A no-op reports no backup and no applied change.
	got, err = projectMCPApplyResult(skillsync.OperationApply, "mcp apply", &skillsync.MCPApplyResult{NoOp: true}, nil)
	if err != nil {
		t.Fatalf("no-op returned an error: %v", err)
	}
	if got.Outcome != skillsync.OutcomeNoChange || got.BackupID != "" || len(got.Applied) != 0 {
		t.Fatalf("no-op projection = %+v", got)
	}
}

// TestB10MCPAdaptersHonorCancelledContext proves the production seams return the
// cancellation before touching a home, so a quitting session cannot start a new
// read and no real directory is inspected by this test.
func TestB10MCPAdaptersHonorCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	missing := "/nonexistent-b10-home"

	if _, err := (DefaultMCPReader{}).Status(ctx, missing, "", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Status error = %v, want context.Canceled", err)
	}
	if _, err := (DefaultMCPReader{}).Doctor(ctx, missing, "", "", "", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Doctor error = %v, want context.Canceled", err)
	}
	if _, err := (DefaultMCPPlanner{}).PreviewPlan(ctx, missing, "", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("PreviewPlan error = %v, want context.Canceled", err)
	}
	if _, err := (DefaultMCPPlanner{}).ListBackups(ctx, missing); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListBackups error = %v, want context.Canceled", err)
	}
	if _, err := (DefaultMCPPlanner{}).PreviewRestore(ctx, missing, "20260101T000000.000000000Z"); !errors.Is(err, context.Canceled) {
		t.Fatalf("PreviewRestore error = %v, want context.Canceled", err)
	}
}

// TestB10MCPRestoreOutcomeIsVisibleWhereItHappened proves a completed restore
// returns to the backup list and reports its outcome there, instead of leaving
// the user on a preview the write already consumed.
func TestB10MCPRestoreOutcomeIsVisibleWhereItHappened(t *testing.T) {
	app := b10MCPApp(t, &fakeMCPReader{}, &fakeMCPPlanner{})
	app.mcp.view = MCPViewRecovery
	app.mcp.inRestorePreview = true
	app.mcp.restorePreview = MCPRestorePreviewView{
		Available:  true,
		BackupID:   "20260101T000000.000000000Z",
		TargetPath: "/home/.config/opencode/opencode.json",
		Token:      "restore-token",
	}
	app.mcp.backups = []MCPBackupView{{ID: "20260101T000000.000000000Z"}}
	app.mcp.recoveryLoading = false

	next, _ := b10DeliverMCPResult(t, app, WriteResultMsg{
		Kind: WriteMCPRestore,
		Result: appops.ApplyResult{
			Outcome:  skillsync.OutcomeApplied,
			BackupID: "safety-backup",
		},
		LogPath: "/home/.agentic-sdd/logs/restore.log",
	})

	if next.mcp.inRestorePreview {
		t.Fatal("the consumed restore preview is still open")
	}
	rendered := b10RenderMCP(next)
	if !strings.Contains(rendered, "Last operation: completed") {
		t.Fatalf("the restore outcome is not visible on the recovery screen:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Backup ID: safety-backup") {
		t.Fatalf("the recorded safety backup is not shown:\n%s", rendered)
	}
}

// b10FailedMCPApply runs one real apply through the production appops seam and
// the production host projection, and returns the message the root shell would
// receive. The ownership manifest is corrupted after the preview, so the engine
// fails after the rename and performs a real rollback on an isolated temp home.
func b10FailedMCPApply(t *testing.T) WriteResultMsg {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(configPath, []byte("{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "opencode")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho '2.1.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	planReq, err := appops.NewMCPPlanRequest(home, configPath, []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("NewMCPPlanRequest: %v", err)
	}
	planRes, err := appops.MCPPlanPreview(planReq)
	if err != nil {
		t.Fatalf("MCPPlanPreview: %v", err)
	}

	stateDir := filepath.Join(home, ".agentic-sdd", "mcp")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "manifest.json"), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	req := MCPWriteRequest{
		Home:         home,
		ConfigPath:   configPath,
		Plan:         planRes.Plan,
		Token:        planRes.Token,
		ExpectDigest: planRes.Plan.BaseDigest,
		OpenCodeBin:  bin,
	}
	result, err := appOpsMCPApply(req, planRes.Token, io.Discard)
	if err == nil {
		t.Fatal("the fixture apply was reported as success")
	}
	return WriteResultMsg{
		Kind:    WriteMCPApply,
		Result:  result,
		Err:     err,
		LogPath: filepath.Join(home, ".agentic-sdd", "logs", "apply.log"),
	}
}

// TestB10MCPFailedApplyProjectsRollbackNotBareFailure is the F-01 regression
// test: a real failed apply that rolled back must reach the TUI as "rolled back"
// with its backup and log facts, not as a bare "failed" that hides that the
// engine already undid the write.
func TestB10MCPFailedApplyProjectsRollbackNotBareFailure(t *testing.T) {
	msg := b10FailedMCPApply(t)

	outcome := mcpOutcomeFromResult(msg)
	if outcome.Headline != "rolled back" {
		t.Fatalf("headline = %q, want rolled back (a rollback ran)", outcome.Headline)
	}
	if outcome.Rollback != "complete" {
		t.Fatalf("rollback = %q, want complete", outcome.Rollback)
	}
	if outcome.BackupID == "" || outcome.BackupID != msg.Result.BackupID {
		t.Fatalf("backup id = %q, want the engine's %q", outcome.BackupID, msg.Result.BackupID)
	}
	if outcome.BackupPath == "" || outcome.BackupPath != msg.Result.BackupPath {
		t.Fatalf("backup path = %q, want the engine's %q", outcome.BackupPath, msg.Result.BackupPath)
	}
	if outcome.LogPath != msg.LogPath {
		t.Fatalf("log path = %q, want %q", outcome.LogPath, msg.LogPath)
	}
	if len(outcome.PartiallyApplied) != 0 {
		t.Fatalf("a rolled-back apply reported partial writes: %v", outcome.PartiallyApplied)
	}
	if !strings.Contains(outcome.Detail, "Engine: ") {
		t.Fatalf("detail dropped the engine error: %q", outcome.Detail)
	}

	// The root shell and the child screen show the same truth.
	app := b10MCPApp(t, &fakeMCPReader{}, &fakeMCPPlanner{})
	next, _ := b10DeliverMCPResult(t, app, msg)
	if got := next.mcp.Outcome(); got.Headline != "rolled back" || got.Rollback != "complete" {
		t.Fatalf("child outcome = %+v, want a completed rollback", got)
	}
	rendered := b10RenderMCP(next)
	if !strings.Contains(rendered, "Last operation: rolled back") {
		t.Errorf("MCP screen missing the rolled-back headline:\n%s", rendered)
	}
	if strings.Contains(rendered, "Last operation: failed") {
		t.Errorf("MCP screen presented a rolled-back write as a bare failure:\n%s", rendered)
	}
	// The structured rows are asserted on the child section itself: the engine's
	// own message is long, so a full-viewport render can scroll them out of view.
	rows := strings.Join(mcpOutcomeLines(next.mcp, 100), "\n")
	for _, want := range []string{"Rollback: complete", "Backup ID: " + outcome.BackupID} {
		if !strings.Contains(rows, want) {
			t.Errorf("MCP outcome rows missing %q:\n%s", want, rows)
		}
	}
}

// TestB10MCPIncompleteRollbackProjectsRecovery proves the same projection keeps
// an incomplete rollback distinct: the engine result reaches the screen with the
// recovery backup and the incomplete state, not a flattened failure.
func TestB10MCPIncompleteRollbackProjectsRecovery(t *testing.T) {
	engineResult := &skillsync.MCPApplyResult{
		TargetPath: "/home/.config/opencode/opencode.json",
		BackupID:   "recovery-backup",
		BackupPath: "/home/.agentic-sdd/mcp/backups/recovery-backup",
		Rollback:   skillsync.RollbackInfo{Attempted: true, Complete: false, Error: "target is read-only"},
	}
	projected, err := projectMCPApplyResult(
		skillsync.OperationApply, "mcp apply", engineResult, errors.New("mcp apply needs recovery"))
	if err == nil {
		t.Fatal("a needs-recovery apply was reported as success")
	}
	msg := WriteResultMsg{
		Kind:    WriteMCPApply,
		Result:  projected,
		Err:     err,
		LogPath: "/home/.agentic-sdd/logs/apply.log",
	}

	outcome := mcpOutcomeFromResult(msg)
	if outcome.Headline != "incomplete" {
		t.Fatalf("headline = %q, want incomplete", outcome.Headline)
	}
	if outcome.Rollback != "incomplete" {
		t.Fatalf("rollback = %q, want incomplete", outcome.Rollback)
	}
	if outcome.BackupID != engineResult.BackupID || outcome.BackupPath != engineResult.BackupPath {
		t.Fatalf("recovery facts lost: %+v", outcome)
	}

	app := b10MCPApp(t, &fakeMCPReader{}, &fakeMCPPlanner{})
	next, _ := b10DeliverMCPResult(t, app, msg)
	rows := strings.Join(mcpOutcomeLines(next.mcp, 100), "\n")
	for _, want := range []string{"Last operation: incomplete", "Rollback: INCOMPLETE", "Backup path: " + engineResult.BackupPath} {
		if !strings.Contains(rows, want) {
			t.Errorf("MCP outcome rows missing %q:\n%s", want, rows)
		}
	}
}
