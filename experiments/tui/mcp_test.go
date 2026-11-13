package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
)

// fakeMCPReader is a deterministic in-memory MCPReader for testing.
type fakeMCPReader struct {
	mu        sync.Mutex
	statusRes MCPStatusView
	statusErr error
	doctorRes MCPDoctorView
	doctorErr error
	calls     []string
}

func (f *fakeMCPReader) Status(ctx context.Context, home, configPath, projectPath string) (MCPStatusView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("status:%s:%s", home, configPath))
	if f.statusErr != nil {
		return MCPStatusView{}, f.statusErr
	}
	return f.statusRes, nil
}

func (f *fakeMCPReader) Doctor(ctx context.Context, home, configPath, projectPath, openCodeBin, codeGraphBin string) (MCPDoctorView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("doctor:%s:%s", home, configPath))
	if f.doctorErr != nil {
		return MCPDoctorView{}, f.doctorErr
	}
	return f.doctorRes, nil
}

// fakeMCPPlanner is a deterministic in-memory MCPPlanner for testing.
type fakeMCPPlanner struct {
	mu         sync.Mutex
	planRes    MCPPlanPreviewView
	planErr    error
	backupsRes []MCPBackupView
	backupsErr error
	restoreRes MCPRestorePreviewView
	restoreErr error
	calls      []string
}

func (f *fakeMCPPlanner) PreviewPlan(ctx context.Context, home, configPath string, selected []string, auth map[string]string) (MCPPlanPreviewView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("plan:%s:%s:%s", home, configPath, strings.Join(selected, ",")))
	if f.planErr != nil {
		return MCPPlanPreviewView{}, f.planErr
	}
	return f.planRes, nil
}

func (f *fakeMCPPlanner) ListBackups(ctx context.Context, home string) ([]MCPBackupView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("backups:%s", home))
	if f.backupsErr != nil {
		return nil, f.backupsErr
	}
	return f.backupsRes, nil
}

func (f *fakeMCPPlanner) PreviewRestore(ctx context.Context, home, backupID string) (MCPRestorePreviewView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("restore-preview:%s:%s", home, backupID))
	if f.restoreErr != nil {
		return MCPRestorePreviewView{}, f.restoreErr
	}
	return f.restoreRes, nil
}

type fakeWriteLog struct {
	buf strings.Builder
}

func (l *fakeWriteLog) Write(p []byte) (n int, err error) {
	return l.buf.Write(p)
}

func (l *fakeWriteLog) Flush() error {
	return nil
}

func (l *fakeWriteLog) Close() error {
	return nil
}

func mcpKey(t *testing.T, m MCPModel, key string) MCPModel {
	t.Helper()
	next, _ := m.Update(tea.KeyPressMsg{Code: []rune(key)[0], Text: key})
	return next
}

func mcpNamedKey(t *testing.T, m MCPModel, name string) MCPModel {
	t.Helper()
	var msg tea.KeyPressMsg
	switch name {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case "space":
		msg = tea.KeyPressMsg{Code: 0, Text: " "}
		t.Logf("space key String() = %q", msg.String())
	default:
		msg = tea.KeyPressMsg{Code: []rune(name)[0], Text: name}
	}
	next, _ := m.Update(msg)
	return next
}

func testMCPRequest(t *testing.T, kind WriteKind) WriteRequest {
	t.Helper()
	home := t.TempDir()
	configPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	mcp := MCPWriteRequest{
		Home:         home,
		ConfigPath:   configPath,
		Token:        "mcp-test-token",
		ExpectDigest: "sha256:base123",
		Label:        "configure MCP tools",
		Source:       "mcp:" + configPath,
	}
	switch kind {
	case WriteMCPApply:
		mcp.PlanID = "sha256:plan-abc123"
	case WriteMCPRestore:
		mcp.BackupID = "20261113T120000Z"
	}
	return WriteRequest{Kind: kind, MCP: mcp, Token: "mcp-test-token"}
}

func testInstallWriteRequest(t *testing.T) WriteRequest {
	t.Helper()
	request, err := appops.NewInstallRequest("", t.TempDir(), true, skillsync.Options{})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	return WriteRequest{
		Kind:    WriteInstall,
		Install: request,
		Token:   "install-preview-token",
		Command: "apply --with-opencode",
	}
}

func testFleetRequest(t *testing.T, kind WriteKind) WriteRequest {
	t.Helper()
	home := t.TempDir()
	fleet := FleetWriteRequest{
		Home:       home,
		ConfigPath: filepath.Join(home, "fleet.json"),
		Token:      "fleet-preview-token",
		Label:      "Fleet change",
		Source:     "personal fleet.json",
	}
	switch kind {
	case WriteFleetApply:
		fleet.PlanID = "sha256:plan-abc123"
	case WriteFleetRestore:
		fleet.BackupID = "20260101T000000.000000000Z"
	}
	return WriteRequest{Kind: kind, Fleet: fleet}
}

func testRestoreWriteRequest(t *testing.T) WriteRequest {
	t.Helper()
	home := t.TempDir()
	req, err := appops.NewRestoreRequest("", home, "20260101T000000.000000000Z", true)
	if err != nil {
		t.Fatalf("NewRestoreRequest: %v", err)
	}
	return WriteRequest{
		Kind:    WriteRestore,
		Restore: req,
		Token:   "restore-preview-token",
		Command: "restore 20260101T000000.000000000Z",
	}
}

// TestMCPOverviewCoverage asserts the three default servers (CodeGraph, Context7, Exa)
// are rendered with purpose, kind, prerequisites and next action.
func TestMCPOverviewCoverage(t *testing.T) {
	reader := &fakeMCPReader{
		statusRes: MCPStatusView{
			Available: true,
			Servers: []MCPServerView{
				{
					Name:              "codegraph",
					DisplayName:       "CodeGraph",
					Kind:              "local",
					Purpose:           "repository code intelligence",
					ConfigState:       "not configured",
					Prerequisite:      "codegraph CLI",
					PrerequisiteMet:   true,
					VerificationState: "connection not tested",
					NextAction:        "Select and preview configuration",
				},
				{
					Name:              "context7",
					DisplayName:       "Context7",
					Kind:              "remote",
					Purpose:           "hosted library documentation",
					ConfigState:       "not configured",
					Prerequisite:      "none",
					PrerequisiteMet:   true,
					AuthMode:          "oauth",
					VerificationState: "connection not tested",
					NextAction:        "Select and choose OAuth or API-key",
				},
				{
					Name:              "exa",
					DisplayName:       "Exa",
					Kind:              "remote",
					Purpose:           "hosted web research",
					ConfigState:       "not configured",
					Prerequisite:      "none",
					PrerequisiteMet:   true,
					AuthMode:          "keyless",
					VerificationState: "connection not tested",
					NextAction:        "Select and acknowledge keyless limits",
				},
			},
		},
	}

	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	m := NewMCPModelWithSeams(cfg, reader, &fakeMCPPlanner{})

	if m.ActiveView() != MCPViewOverview {
		t.Fatalf("active view = %v, want MCPViewOverview", m.ActiveView())
	}
	if bc := m.Breadcrumb(); bc != "Install & Manage / MCP Tools" {
		t.Fatalf("breadcrumb = %q, want 'Install & Manage / MCP Tools'", bc)
	}

	// Update with status
	m, _ = m.Update(MCPStatusMsg{
		Generation: m.generation,
		Status:     reader.statusRes,
	})

	lines := m.Lines(100)
	joined := strings.Join(lines, "\n")

	for _, want := range []string{"CodeGraph", "Context7", "Exa", "repository code intelligence", "hosted library documentation", "hosted web research", "local", "remote"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Overview missing %q:\n%s", want, joined)
		}
	}
}

// TestMCPHubNavigationAndEsc asserts the four hub children, dispatch via 't',
// and Esc returning to the hub.
func TestMCPHubNavigationAndEsc(t *testing.T) {
	if len(installHubEntries) != 4 {
		t.Fatalf("hub entries = %d, want 4", len(installHubEntries))
	}
	if installHubEntries[3].key != "t" || installHubEntries[3].section != installSectionMCP {
		t.Fatalf("fourth hub entry = %+v, want key 't' and installSectionMCP", installHubEntries[3])
	}

	app := newMCPAHubApp(t)

	// Direct 't' shortcut
	app = fleetKey(t, app, "t")
	if app.installSection != installSectionMCP {
		t.Fatalf("after 't', installSection = %v, want installSectionMCP", app.installSection)
	}

	// Esc returns to hub
	app = fleetKey(t, app, "esc")
	if app.installSection != installSectionHub {
		t.Fatalf("after esc from MCP, installSection = %v, want installSectionHub", app.installSection)
	}

	// Cursor navigation to index 3 and Enter
	app = fleetKey(t, app, "k") // wraps backward to index 3
	if app.installHubIndex != 3 {
		t.Fatalf("after k, installHubIndex = %d, want 3", app.installHubIndex)
	}
	app = fleetKey(t, app, "enter")
	if app.installSection != installSectionMCP {
		t.Fatalf("enter on index 3, installSection = %v, want installSectionMCP", app.installSection)
	}
	app = fleetKey(t, app, "esc")
	if app.installSection != installSectionHub {
		t.Fatalf("second esc, installSection = %v, want installSectionHub", app.installSection)
	}
}

// TestMCPSelectValidationAndAuthToggles tests Exa keyless acknowledgement,
// validation blocking preview, Context7 OAuth toggle, and server selection.
func TestMCPSelectValidationAndAuthToggles(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	planner := &fakeMCPPlanner{}
	m := NewMCPModelWithSeams(cfg, &fakeMCPReader{}, planner)

	// Enter Select view via 's'
	m = mcpKey(t, m, "s")
	if m.ActiveView() != MCPViewSelect {
		t.Fatalf("active view = %v, want MCPViewSelect", m.ActiveView())
	}

	// Defaults: exaAcknowledged is false
	if m.exaAcknowledged {
		t.Fatal("exaAcknowledged defaults to true, want false")
	}
	if m.authChoices["context7"] != "oauth" {
		t.Fatalf("context7 auth = %q, want 'oauth'", m.authChoices["context7"])
	}
	if m.authChoices["exa"] != "keyless" {
		t.Fatalf("exa auth = %q, want 'keyless'", m.authChoices["exa"])
	}

	// Validation blocks preview when Exa is keyless and not acknowledged
	m = mcpNamedKey(t, m, "enter")
	if m.ActiveView() != MCPViewSelect {
		t.Fatalf("unacknowledged Exa moved to %v, want MCPViewSelect", m.ActiveView())
	}
	if !strings.Contains(m.selectNotice, "acknowledging") {
		t.Fatalf("selectNotice = %q, want rate limits/privacy warning", m.selectNotice)
	}

	// Move cursor to Context7 auth choice (index 2) and toggle
	m.selectCursor = 2
	m, _ = m.handleSelectKey(" ")
	if m.authChoices["context7"] != "api-key" {
		t.Fatalf("after space on Context7 auth, got %q, want 'api-key'", m.authChoices["context7"])
	}
	m, _ = m.handleSelectKey(" ")
	if m.authChoices["context7"] != "oauth" {
		t.Fatalf("after second space on Context7 auth, got %q, want 'oauth'", m.authChoices["context7"])
	}

	// Move cursor to Exa auth choice (index 4) and cycle
	m.selectCursor = 4
	m, _ = m.handleSelectKey(" ")
	if m.authChoices["exa"] != "oauth" {
		t.Fatalf("after space on Exa auth, got %q, want 'oauth'", m.authChoices["exa"])
	}
	m, _ = m.handleSelectKey(" ")
	if m.authChoices["exa"] != "api-key" {
		t.Fatalf("after space on Exa auth, got %q, want 'api-key'", m.authChoices["exa"])
	}
	m, _ = m.handleSelectKey(" ")
	if m.authChoices["exa"] != "keyless" {
		t.Fatalf("after space on Exa auth, got %q, want 'keyless'", m.authChoices["exa"])
	}

	// Move cursor to Exa acknowledgement (index 5) and toggle
	m.selectCursor = 5
	m, _ = m.handleSelectKey(" ")
	if !m.exaAcknowledged {
		t.Fatal("after space on cursor 5, exaAcknowledged = false, want true")
	}

	// Now validation passes and moves to Preview
	m, cmd := m.handleSelectKey("enter")
	if m.ActiveView() != MCPViewPreview {
		t.Fatalf("acknowledged Exa did not move to preview, view = %v", m.ActiveView())
	}
	if cmd == nil {
		t.Fatal("requestPlanPreview emitted nil command")
	}

	// Deselect CodeGraph on cursor 0
	m.view = MCPViewSelect
	m.selectCursor = 0
	m, _ = m.handleSelectKey(" ")
	if m.selected["codegraph"] {
		t.Fatal("after space on codegraph, selected = true, want false")
	}
}

// TestMCPPreviewTokenAndGeneration asserts preview token gating, target/digest/entries
// projection, zero writes, and stale generation discarding.
func TestMCPPreviewTokenAndGeneration(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	planner := &fakeMCPPlanner{
		planRes: MCPPlanPreviewView{
			Available:  true,
			TargetPath: "/test/opencode.json",
			Format:     "json",
			BaseDigest: "sha256:base123",
			PlanDigest: "sha256:plan456",
			Selected:   []string{"codegraph", "context7"},
			Token:      "mcp-token-test-1",
			Entries: []MCPEntryDiffView{
				{
					Name:      "codegraph",
					Op:        "add",
					Reason:    "configured server",
					DiffLines: []string{`+ "codegraph": {"type": "local"}`},
				},
			},
		},
	}
	m := NewMCPModelWithSeams(cfg, &fakeMCPReader{}, planner)
	m.view = MCPViewPreview
	m.planGen = 2

	// Case 1: Stale generation discarded
	staleMsg := MCPPlanPreviewMsg{
		Generation: 1, // older than planGen 2
		Preview: MCPPlanPreviewView{
			Token: "stale-token",
		},
	}
	m, _ = m.Update(staleMsg)
	if m.token == "stale-token" {
		t.Fatal("stale generation 1 updated preview token, want discarded")
	}

	// Case 2: Matching generation accepted
	freshMsg := MCPPlanPreviewMsg{
		Generation: 2,
		Preview:    planner.planRes,
	}
	m, _ = m.Update(freshMsg)
	if m.token != "mcp-token-test-1" {
		t.Fatalf("token = %q, want 'mcp-token-test-1'", m.token)
	}
	if m.baseDigest != "sha256:base123" {
		t.Fatalf("baseDigest = %q, want 'sha256:base123'", m.baseDigest)
	}

	lines := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(lines, "/test/opencode.json") {
		t.Errorf("preview missing target path:\n%s", lines)
	}
	if !strings.Contains(lines, "sha256:base123") {
		t.Errorf("preview missing base digest:\n%s", lines)
	}
	if !strings.Contains(lines, "codegraph") {
		t.Errorf("preview missing entry name:\n%s", lines)
	}

	// Case 3: Empty token blocks apply
	mNoToken := m
	mNoToken.planPreview.Token = ""
	mNoToken.token = ""
	mNoToken.view = MCPViewConfirm
	mNoToken.confirmSelection = true
	mNoToken, cmd := mNoToken.confirmApplyCmd()
	if cmd != nil {
		t.Fatal("confirmApplyCmd without token emitted command, want nil")
	}
	if mNoToken.outcome.Headline != "no preview token" {
		t.Fatalf("outcome = %+v, want 'no preview token'", mNoToken.outcome)
	}

	// Case 4: Preview with conflicts blocks advancing to Confirm
	mConflict := m
	mConflict.planPreview.Conflicts = []string{"unmanaged conflict with server 'exa'"}
	mConflict = mcpNamedKey(t, mConflict, "enter")
	if mConflict.ActiveView() == MCPViewConfirm {
		t.Fatal("preview with conflicts transitioned to Confirm, want blocked")
	}
}

// TestMCPConfirmDefaultNoAndTinyGeometry asserts default No, Enter canceling,
// tiny geometry guard, and inert double confirmation.
func TestMCPConfirmDefaultNoAndTinyGeometry(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	m := NewMCPModelWithSeams(cfg, &fakeMCPReader{}, &fakeMCPPlanner{})
	m.view = MCPViewConfirm
	m.confirmSelection = false
	m.planPreview = MCPPlanPreviewView{
		Token:      "valid-token",
		TargetPath: "/test/opencode.json",
		BaseDigest: "base123",
		PlanDigest: "plan456",
		Selected:   []string{"codegraph"},
	}

	// Default is No
	lines := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(lines, "No, cancel") {
		t.Errorf("confirm view missing 'No, cancel':\n%s", lines)
	}

	// Enter on default No cancels and returns to Preview
	mCancel := mcpNamedKey(t, m, "enter")
	if mCancel.ActiveView() != MCPViewPreview {
		t.Fatalf("enter on No moved to %v, want MCPViewPreview", mCancel.ActiveView())
	}

	// Esc cancels and returns to Preview
	mEsc := mcpNamedKey(t, m, "esc")
	if mEsc.ActiveView() != MCPViewPreview {
		t.Fatalf("esc on confirm moved to %v, want MCPViewPreview", mEsc.ActiveView())
	}

	// Tiny geometry cannot confirm
	mTiny := m
	mTiny.SetSize(50, 6) // tiny width < 60, height < 8
	if !mTiny.tinyGeometry {
		t.Fatal("tinyGeometry = false, want true")
	}
	mTiny = mcpKey(t, mTiny, "y")
	if mTiny.writePending {
		t.Fatal("tiny geometry dispatched write on 'y', want blocked")
	}
	mTiny = mcpNamedKey(t, mTiny, "enter")
	if mTiny.writePending {
		t.Fatal("tiny geometry dispatched write on enter, want blocked")
	}

	// Normal confirmation with 'y'
	mNormal := m
	mNormal, cmd := mNormal.handleConfirmKey("y")
	if !mNormal.writePending {
		t.Fatal("after 'y', writePending = false, want true")
	}
	if cmd == nil {
		t.Fatal("confirmApplyCmd emitted nil command")
	}
	msg := cmd()
	startMsg, ok := msg.(StartWriteMsg)
	if !ok || startMsg.Request.Kind != WriteMCPApply {
		t.Fatalf("emitted msg = %+v, want StartWriteMsg with WriteMCPApply", msg)
	}

	// Double confirmation inert while writePending:
	// After first confirmation, view moved back to MCPViewPreview and writePending is true.
	// Pressing 'y' from Preview reopens Confirm with default No and does not dispatch a write.
	mPending, cmd2 := mNormal.handleKey(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd2 != nil {
		t.Fatal("pressing 'y' from Preview while writePending dispatched command, want nil")
	}
	if !mPending.writePending {
		t.Fatal("writePending should remain true")
	}
	if mPending.confirmSelection {
		t.Fatal("confirm reopened with confirmSelection = true, want default No (false)")
	}
}

// TestMCPDiagnosticsAndRecovery asserts contextual offline diagnostics,
// backup listing, preview restore, and default No on restore.
func TestMCPDiagnosticsAndRecovery(t *testing.T) {
	reader := &fakeMCPReader{
		doctorRes: MCPDoctorView{
			Available: true,
			Checks: []MCPDoctorCheckView{
				{Name: "OpenCode configuration target", State: "pass", Detail: "valid JSON"},
				{Name: "CodeGraph CLI", State: "pass", Detail: "/usr/local/bin/codegraph"},
				{Name: "Remote connectivity", State: "not-checked", Detail: "offline mode (no network test in TUI)"},
			},
			NotChecked: []string{"network", "model inference"},
		},
	}
	planner := &fakeMCPPlanner{
		backupsRes: []MCPBackupView{
			{ID: "20261113T120000Z", Path: "/backups/1", CreatedAt: "2026-11-13 12:00:00", Servers: []string{"codegraph"}},
		},
		restoreRes: MCPRestorePreviewView{
			Available:  true,
			BackupID:   "20261113T120000Z",
			TargetPath: "/test/opencode.json",
			BaseDigest: "sha256:current123",
			Token:      "restore-token-1",
			Entries: []MCPEntryDiffView{
				{Name: "codegraph", Op: "restore", Reason: "restored from backup"},
			},
		},
	}

	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	m := NewMCPModelWithSeams(cfg, reader, planner)

	// 'd' opens Diagnostics
	m, cmdDoc := m.handleKey(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.ActiveView() != MCPViewDiagnostics {
		t.Fatalf("view = %v, want MCPViewDiagnostics", m.ActiveView())
	}
	if cmdDoc == nil {
		t.Fatal("diagnostics key emitted nil command")
	}

	// Update with doctor result
	m, _ = m.Update(MCPDoctorMsg{Generation: m.doctorGen, Doctor: reader.doctorRes})
	diagLines := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(diagLines, "connection not tested") && !strings.Contains(diagLines, "offline mode") {
		t.Errorf("diagnostics view missing offline wording:\n%s", diagLines)
	}

	// Esc returns to Overview
	m = mcpNamedKey(t, m, "esc")
	if m.ActiveView() != MCPViewOverview {
		t.Fatalf("esc from diagnostics moved to %v, want MCPViewOverview", m.ActiveView())
	}

	// 'r' from Overview enters Recovery
	m, cmdRec := m.handleOverviewKey("r")
	if m.ActiveView() != MCPViewRecovery {
		t.Fatalf("view = %v, want MCPViewRecovery", m.ActiveView())
	}
	if cmdRec == nil {
		t.Fatal("recovery key emitted nil command")
	}

	// Update with backup list
	m, _ = m.Update(MCPBackupListMsg{Generation: m.recoveryGen, Backups: planner.backupsRes})
	recLines := strings.Join(m.Lines(100), "\n")
	if !strings.Contains(recLines, "20261113T120000Z") {
		t.Errorf("recovery list missing backup ID:\n%s", recLines)
	}

	// Enter on backup previews restore
	m, cmdRest := m.handleRecoveryKey("enter")
	if cmdRest == nil {
		t.Fatal("enter on backup emitted nil command")
	}
	m, _ = m.Update(MCPRestorePreviewMsg{Generation: m.restoreGen, Preview: planner.restoreRes})
	if !m.inRestorePreview {
		t.Fatal("inRestorePreview = false, want true")
	}

	// 'r' enters restore confirmation
	m, _ = m.handleRecoveryKey("r")
	if !m.confirmRestore {
		t.Fatal("confirmRestore = false, want true")
	}
	if m.confirmSelection {
		t.Fatal("confirmSelection = true on restore confirm, want default false (No)")
	}

	// Enter on No cancels restore
	m, _ = m.handleRecoveryKey("enter")
	if m.confirmRestore {
		t.Fatal("enter on No left confirmRestore = true, want false")
	}

	// Re-enter confirmation and confirm with 'y'
	m, _ = m.handleRecoveryKey("r")
	m, cmdConf := m.handleRecoveryKey("y")
	if cmdConf == nil {
		t.Fatal("confirm restore emitted nil command")
	}
	msg := cmdConf()
	startMsg, ok := msg.(StartWriteMsg)
	if !ok || startMsg.Request.Kind != WriteMCPRestore {
		t.Fatalf("emitted msg = %+v, want StartWriteMsg with WriteMCPRestore", msg)
	}
}

// TestOpsHostMCPPairwiseBusyRefusal asserts mutual exclusion and busy refusal
// between MCP writes and all other write kinds in both directions.
func TestOpsHostMCPPairwiseBusyRefusal(t *testing.T) {
	host := NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return &fakeWriteLog{}, "/tmp/operation.log", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyRestore: func(appops.RestoreRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyFleet: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyFleetRestore: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyMCP: func(MCPWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{BackupID: "mcp-bak"}, nil
		},
		ApplyMCPRestore: func(MCPWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{BackupID: "mcp-restore-bak"}, nil
		},
	})

	mcpApplyReq := testMCPRequest(t, WriteMCPApply)
	mcpRestoreReq := testMCPRequest(t, WriteMCPRestore)
	installReq := testInstallWriteRequest(t)
	restoreReq := testRestoreWriteRequest(t)
	fleetApplyReq := testFleetRequest(t, WriteFleetApply)
	fleetRestoreReq := testFleetRequest(t, WriteFleetRestore)

	allOtherRequests := []WriteRequest{
		installReq,
		restoreReq,
		fleetApplyReq,
		fleetRestoreReq,
		mcpApplyReq,
		mcpRestoreReq,
	}

	// Scenario 1: Active WriteMCPApply blocks all other write kinds
	t.Run("active MCP apply blocks all kinds", func(t *testing.T) {
		started, run, err := host.BeginWrite(mcpApplyReq)
		if err != nil || run == nil {
			t.Fatalf("BeginWrite MCP apply: %v", err)
		}
		for _, req := range allOtherRequests {
			_, cmd, err := host.BeginWrite(req)
			if !errors.Is(err, ErrWriteInProgress) {
				t.Errorf("BeginWrite %v during active MCP apply got %v, want ErrWriteInProgress", req.Kind, err)
			}
			if cmd != nil {
				t.Errorf("refused request %v emitted command", req.Kind)
			}
		}
		run()
		host.Complete(started.ID)
	})

	// Scenario 2: Active WriteMCPRestore blocks all other write kinds
	t.Run("active MCP restore blocks all kinds", func(t *testing.T) {
		started, run, err := host.BeginWrite(mcpRestoreReq)
		if err != nil || run == nil {
			t.Fatalf("BeginWrite MCP restore: %v", err)
		}
		for _, req := range allOtherRequests {
			_, cmd, err := host.BeginWrite(req)
			if !errors.Is(err, ErrWriteInProgress) {
				t.Errorf("BeginWrite %v during active MCP restore got %v, want ErrWriteInProgress", req.Kind, err)
			}
			if cmd != nil {
				t.Errorf("refused request %v emitted command", req.Kind)
			}
		}
		run()
		host.Complete(started.ID)
	})

	// Scenario 3: Reverse direction: Active Install blocks MCP apply and restore
	t.Run("active Install blocks MCP writes", func(t *testing.T) {
		started, run, err := host.BeginWrite(installReq)
		if err != nil || run == nil {
			t.Fatalf("BeginWrite Install: %v", err)
		}
		for _, mcpReq := range []WriteRequest{mcpApplyReq, mcpRestoreReq} {
			_, cmd, err := host.BeginWrite(mcpReq)
			if !errors.Is(err, ErrWriteInProgress) {
				t.Errorf("BeginWrite %v during active Install got %v, want ErrWriteInProgress", mcpReq.Kind, err)
			}
			if cmd != nil {
				t.Errorf("refused MCP request %v emitted command", mcpReq.Kind)
			}
		}
		run()
		host.Complete(started.ID)
	})

	// Scenario 4: Reverse direction: Active FleetApply blocks MCP apply and restore
	t.Run("active Fleet apply blocks MCP writes", func(t *testing.T) {
		started, run, err := host.BeginWrite(fleetApplyReq)
		if err != nil || run == nil {
			t.Fatalf("BeginWrite Fleet apply: %v", err)
		}
		for _, mcpReq := range []WriteRequest{mcpApplyReq, mcpRestoreReq} {
			_, cmd, err := host.BeginWrite(mcpReq)
			if !errors.Is(err, ErrWriteInProgress) {
				t.Errorf("BeginWrite %v during active Fleet apply got %v, want ErrWriteInProgress", mcpReq.Kind, err)
			}
			if cmd != nil {
				t.Errorf("refused MCP request %v emitted command", mcpReq.Kind)
			}
		}
		run()
		host.Complete(started.ID)
	})
}

// TestOpsHostMCPHeadersAndSource asserts source() and command() transcript headers
// for WriteMCPApply and WriteMCPRestore.
func TestOpsHostMCPHeadersAndSource(t *testing.T) {
	home := "/home/testuser"
	configPath := home + "/.config/opencode/opencode.json"

	applyReq := WriteRequest{
		Kind: WriteMCPApply,
		MCP: MCPWriteRequest{
			Home:         home,
			ConfigPath:   configPath,
			PlanID:       "plan123",
			Token:        "tok123",
			ExpectDigest: "dig123",
			Source:       "mcp:" + configPath,
		},
		Token: "tok123",
	}

	if h := applyReq.home(); h != home {
		t.Fatalf("home = %q, want %q", h, home)
	}
	if src := applyReq.source(); src != "mcp:"+configPath {
		t.Fatalf("source = %q, want %q (must never be 'embedded')", src, "mcp:"+configPath)
	}
	if cmd := applyReq.command(); cmd != "mcp apply --config "+configPath {
		t.Fatalf("command = %q, want %q", cmd, "mcp apply --config "+configPath)
	}

	restoreReq := WriteRequest{
		Kind: WriteMCPRestore,
		MCP: MCPWriteRequest{
			Home:       home,
			ConfigPath: configPath,
			BackupID:   "20261113T120000Z",
			Token:      "tok456",
			Source:     "mcp:" + configPath,
		},
		Token: "tok456",
	}

	if src := restoreReq.source(); src != "mcp:"+configPath {
		t.Fatalf("source = %q, want %q", src, "mcp:"+configPath)
	}
	if cmd := restoreReq.command(); cmd != "mcp restore 20261113T120000Z" {
		t.Fatalf("command = %q, want 'mcp restore 20261113T120000Z'", cmd)
	}

	// Validate method
	if err := applyReq.validate(); err != nil {
		t.Fatalf("apply validation error: %v", err)
	}
	if err := restoreReq.validate(); err != nil {
		t.Fatalf("restore validation error: %v", err)
	}

	// Payload mismatch validation error
	invalidReq := WriteRequest{
		Kind: WriteInstall,
		MCP:  applyReq.MCP, // install kind with MCP payload
	}
	if err := invalidReq.validate(); err == nil {
		t.Fatal("validate on payload mismatch returned nil, want error")
	}
}

// TestMCPDefaultAdapterPurity asserts DefaultMCPReader and DefaultMCPPlanner
// never mutate files, read a real home, or perform network operations.
func TestMCPDefaultAdapterPurity(t *testing.T) {
	reader := DefaultMCPReader{}
	planner := DefaultMCPPlanner{}
	ctx := context.Background()

	// Empty home returns available=false without reading or error
	st, err := reader.Status(ctx, "", "", "")
	if err != nil {
		t.Fatalf("status with empty home returned error: %v", err)
	}
	if st.Available {
		t.Fatal("status with empty home returned Available=true, want false")
	}
	if len(st.Servers) != 3 {
		t.Fatalf("default server rows = %d, want 3", len(st.Servers))
	}

	// Doctor with empty home returns available=false
	doc, err := reader.Doctor(ctx, "", "", "", "", "")
	if err != nil {
		t.Fatalf("doctor with empty home returned error: %v", err)
	}
	if doc.Available {
		t.Fatal("doctor with empty home returned Available=true, want false")
	}

	// Planner with empty home safely returns errors or empty slices without panics
	_, err = planner.PreviewPlan(ctx, "", "", []string{"codegraph"}, nil)
	if err == nil {
		t.Fatal("planner preview with empty home succeeded, want error")
	}
	_, err = planner.ListBackups(ctx, "")
	if err == nil {
		t.Fatal("planner list backups with empty home succeeded, want error")
	}
	_, err = planner.PreviewRestore(ctx, "", "bak1")
	if err == nil {
		t.Fatal("planner preview restore with empty home succeeded, want error")
	}
}

// TestMCPHelpSectionTopic asserts HelpSectionMCP exists in AllHelpSections()
// and MCPHelpLines() states the required topics truth.
func TestMCPHelpSectionTopic(t *testing.T) {
	sections := AllHelpSections()
	var found bool
	for _, s := range sections {
		if s.Section == HelpSectionMCP {
			found = true
			if s.Key != "10" {
				t.Fatalf("HelpSectionMCP key = %q, want '10'", s.Key)
			}
			if s.Title != "MCP Tools" {
				t.Fatalf("HelpSectionMCP title = %q, want 'MCP Tools'", s.Title)
			}
		}
	}
	if !found {
		t.Fatal("HelpSectionMCP not found in AllHelpSections()")
	}

	lines := strings.Join(MCPHelpLines(), "\n")
	for _, want := range []string{
		"MCP Tools",
		"press t",
		"Overview",
		"Select servers",
		"Preview configuration",
		"default-No",
		"Diagnostics",
		"Recovery",
		"Offline only",
		"CodeGraph",
		"Context7",
		"Exa",
	} {
		if !strings.Contains(lines, want) {
			t.Errorf("MCPHelpLines missing %q:\n%s", want, lines)
		}
	}
}

// TestFleetHelpLastSection asserts the updated expectation that MCP Tools is the final section.
func TestFleetHelpLastSection(t *testing.T) {
	sections := AllHelpSections()
	last := sections[len(sections)-1]
	if last.Section != HelpSectionMCP || last.Key != "10" {
		t.Fatalf("last section = %+v, want HelpSectionMCP (10)", last)
	}
	fleet := sections[len(sections)-2]
	if fleet.Section != HelpSectionFleet || fleet.Key != "9" {
		t.Fatalf("second-to-last section = %+v, want HelpSectionFleet (9)", fleet)
	}
}

// TestMCPWriteOutcomeHandling asserts Outcome() and Reload() state transitions.
func TestMCPWriteOutcomeHandling(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}
	m := NewMCPModelWithSeams(cfg, &fakeMCPReader{}, &fakeMCPPlanner{})

	// Success outcome
	resMsg := WriteResultMsg{
		Kind: WriteMCPApply,
		Result: appops.ApplyResult{
			BackupID: "bak-123",
		},
		LogPath: "/logs/test.log",
	}
	m, _ = m.Update(resMsg)
	if m.writePending {
		t.Fatal("after WriteResultMsg, writePending = true, want false")
	}
	if m.outcome.Headline != "MCP applied successfully" {
		t.Fatalf("outcome = %+v, want success headline", m.outcome)
	}
	if m.outcome.BackupID != "bak-123" {
		t.Fatalf("backupID = %q, want 'bak-123'", m.outcome.BackupID)
	}

	// Refused outcome
	refMsg := WriteRefusedMsg{
		Kind:   WriteMCPApply,
		Reason: "another operation in progress",
	}
	m, _ = m.Update(refMsg)
	if m.outcome.Headline != "write refused" {
		t.Fatalf("outcome = %+v, want 'write refused'", m.outcome)
	}

	// Failed outcome
	failMsg := WriteResultMsg{
		Kind: WriteMCPApply,
		Err:  errors.New("injected apply error"),
	}
	m, _ = m.Update(failMsg)
	if m.outcome.Headline != "write failed" {
		t.Fatalf("outcome = %+v, want 'write failed'", m.outcome)
	}
}

func TestMCPQuitFromDiagnosticsHeadless(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 30, Home: t.TempDir()}, Catalog{})
	app.activeRoute = RouteInstall
	app.installSection = installSectionMCP
	app.mcp.view = MCPViewDiagnostics
	next, cmd := app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q from MCP diagnostics returned no quit command")
	}
	updated, ok := next.(App)
	if !ok || !updated.quitting {
		t.Fatalf("q from MCP diagnostics did not begin quit: %#v", next)
	}
}
