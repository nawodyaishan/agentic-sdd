package tui

// W2-C deployment and recovery tests. Fixtures and fake seams only: no real
// home, model, network or Git operation is reached. Live apply is exercised in
// a temporary home fixture.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

// fakeFleetPlanner is the deterministic read-only deployment/recovery seam.
type fakeFleetPlanner struct {
	plan       FleetPlanPreview
	planErr    error
	backups    []FleetBackupView
	backupErr  error
	restore    FleetRestorePreview
	restoreErr error

	planCalls    int
	backupCalls  int
	restoreCalls int
}

func (p *fakeFleetPlanner) PreviewPlan(ctx context.Context, source string) (FleetPlanPreview, error) {
	p.planCalls++
	if p.planErr != nil {
		return FleetPlanPreview{}, p.planErr
	}
	return p.plan, nil
}

func (p *fakeFleetPlanner) ListBackups(ctx context.Context) ([]FleetBackupView, error) {
	p.backupCalls++
	if p.backupErr != nil {
		return nil, p.backupErr
	}
	return append([]FleetBackupView(nil), p.backups...), nil
}

func (p *fakeFleetPlanner) PreviewRestore(ctx context.Context, backupID string) (FleetRestorePreview, error) {
	p.restoreCalls++
	if p.restoreErr != nil {
		return FleetRestorePreview{}, p.restoreErr
	}
	return p.restore, nil
}

func testFleetPlanPreview() FleetPlanPreview {
	plan := &fleet.Plan{
		PlanID:          "plan-abc123",
		Scope:           "global",
		RendererVersion: "1",
		TargetRoot:      "/tmp/fleet-target",
		ProposedOps: []fleet.ProposedOp{{
			InstanceID: "sdd-implement-1",
			File:       "sdd-implement-1.md",
			Op:         "create",
		}},
	}
	return FleetPlanPreview{
		Available:       true,
		SourcePath:      "/test/fleet.json",
		PlanID:          plan.PlanID,
		Scope:           "global",
		RendererVersion: "1",
		Token:           "plan-token-abc",
		Counts:          map[string]int{"total": 1, fleet.FileOpCreate: 1},
		Files: []FleetFileDiff{{
			InstanceID: "sdd-implement-1",
			Path:       "/tmp/fleet-target/sdd-implement-1.md",
			Op:         fleet.FileOpCreate,
			DiffLines:  []string{"+new agent"},
		}},
		DiffLines: []string{"=== create · sdd-implement-1 · /tmp/fleet-target/sdd-implement-1.md", "+new agent"},
		Plan:      plan,
	}
}

func testFleetRestorePreview() FleetRestorePreview {
	return FleetRestorePreview{
		Available:  true,
		BackupID:   "20260101T000000.000000000Z",
		BackupPath: "/tmp/backups/20260101T000000.000000000Z",
		Counts:     map[string]int{"total": 1, fleet.FileOpUpdate: 1},
		Files: []FleetFileDiff{{
			InstanceID: "sdd-implement-1",
			Path:       "/tmp/fleet-target/sdd-implement-1.md",
			Op:         fleet.FileOpUpdate,
			DiffLines:  []string{"-old agent", "+restored agent"},
		}},
		DiffLines: []string{"=== update · sdd-implement-1 · /tmp/fleet-target/sdd-implement-1.md", "-old agent", "+restored agent"},
		Token:     "restore-token-abc",
	}
}

func newFleetWriteModel(t *testing.T, live bool, planner FleetPlanner) (FleetModel, *trackingFleetReader) {
	t.Helper()
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	cfg := TUIConfig{
		Width:       110,
		Height:      34,
		Color:       false,
		Home:        t.TempDir(),
		FleetConfig: "/test/fleet.json",
		FleetLive:   live,
	}
	model := NewFleetModelWithSeams(cfg, reader, &fakeFleetEditor{}, planner)
	status := testFleetStatusView()
	status.SourcePath = "/test/fleet.json"
	model.status = status
	model.hasLoaded = true
	return model, reader
}

// fleetUpdateKey sends one key and returns the model plus an executed command
// message when the command is immediate (never a timer).
func fleetUpdateKey(t *testing.T, model FleetModel, code rune, text string) (FleetModel, tea.Msg) {
	t.Helper()
	msg := tea.KeyPressMsg{Code: code}
	if text != "" {
		msg.Text = text
	}
	model, cmd := model.Update(msg)
	if cmd == nil {
		return model, nil
	}
	return model, cmd()
}

func TestFleetDeploymentReviewPlanCountsAndDiff(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, true, planner)

	model, cmd := model.requestPlanPreview()
	if cmd == nil {
		t.Fatal("requestPlanPreview returned no command")
	}
	model, _ = model.Update(cmd())
	model.SetView(FleetViewReview)

	view := strings.Join(model.Lines(110), "\n")
	for _, want := range []string{
		"OpenCode deployment plan",
		"Deployment capability: live (reviewed build).",
		"Plan identity: plan-abc123",
		"Changes: 1 create · 0 update · 0 remove · 0 no-op · 0 conflict",
		"Order: removals and conflicts are listed before routine updates.",
		"Deployment diff (per file)",
		"=== create · sdd-implement-1",
		"+new agent",
		"v: fresh plan · a: apply (live only)",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("deployment review missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(strings.ToLower(view), "apply plan") {
		t.Errorf("deployment review contains the forbidden phrase 'apply plan':\n%s", view)
	}
}

func TestFleetApplyConfirmationDefaultsToNoAndDispatchesOnce(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, true, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview

	model, _ = model.requestApply()
	if !model.IsConfirmingApply() {
		t.Fatal("requestApply did not open the confirmation")
	}

	// Enter is the default No: it cancels without a command.
	model, msg := fleetUpdateKey(t, model, tea.KeyEnter, "")
	if model.IsConfirmingApply() {
		t.Fatal("Enter confirmed the apply; the default must be No")
	}
	if msg != nil {
		t.Fatalf("Enter produced %T, want no command", msg)
	}

	model, _ = model.requestApply()
	model, msg = fleetUpdateKey(t, model, 'y', "y")
	start, ok := msg.(StartWriteMsg)
	if !ok {
		t.Fatalf("y produced %T, want StartWriteMsg", msg)
	}
	if start.Request.Kind != WriteFleetApply {
		t.Fatalf("kind = %v, want WriteFleetApply", start.Request.Kind)
	}
	fleetReq := start.Request.Fleet
	if fleetReq.Home == "" || fleetReq.ConfigPath != "/test/fleet.json" {
		t.Fatalf("fleet request home/path = %q/%q", fleetReq.Home, fleetReq.ConfigPath)
	}
	if fleetReq.PlanID != "plan-abc123" || fleetReq.Plan == nil || fleetReq.Token != "plan-token-abc" {
		t.Fatalf("fleet identity = %+v", fleetReq)
	}
	if fleetReq.Source == "" || fleetReq.Source == "embedded" {
		t.Fatalf("fleet source = %q, want a non-embedded source", fleetReq.Source)
	}

	// A second affirmative key while the write is pending cannot dispatch again.
	model, second := fleetUpdateKey(t, model, 'y', "y")
	if second != nil {
		t.Fatalf("second y produced %T, want no second dispatch", second)
	}
	if !model.WritePending() {
		t.Fatal("the model dropped its pending-write guard")
	}
}

func TestFleetApplyBlocksDirtyDraftAndConflicts(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, true, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview

	if err := model.StageModel("sdd-implement-1", "opencode-go/deepseek-v4.1-flash"); err != nil {
		t.Fatal(err)
	}
	model, _ = model.requestApply()
	if model.IsConfirmingApply() {
		t.Fatal("apply opened while the source had unsaved edits")
	}
	if !strings.Contains(strings.ToLower(model.ApplyOutcome().Headline), "source not saved") {
		t.Fatalf("outcome = %+v, want a source-not-saved refusal", model.ApplyOutcome())
	}

	model.DiscardEdits()
	conflicting := testFleetPlanPreview()
	conflicting.Counts = map[string]int{"total": 1, fleet.FileOpConflict: 1}
	model.planPreview = conflicting
	model, _ = model.requestApply()
	if model.IsConfirmingApply() {
		t.Fatal("apply opened while the plan had conflicts")
	}
	if !strings.Contains(strings.ToLower(model.ApplyOutcome().Headline), "conflict") {
		t.Fatalf("outcome = %+v, want a conflict refusal", model.ApplyOutcome())
	}
}

func TestFleetApplyResultRecordsRealOutcomeAndRefreshes(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, reader := newFleetWriteModel(t, true, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview
	model, _ = model.requestApply()
	model, msg := fleetUpdateKey(t, model, 'y', "y")
	start := msg.(StartWriteMsg)

	model, _ = model.Update(WriteStartedMsg{ID: opIDForTest(1), Kind: WriteFleetApply})
	before := reader.calls
	model, cmd := model.Update(WriteResultMsg{
		ID:   opIDForTest(1),
		Kind: WriteFleetApply,
		Result: appops.ApplyResult{
			Outcome:    skillsync.OutcomeApplied,
			FleetApply: &fleet.ApplyResult{Created: 1, BackupID: "backup-apply-1"},
		},
	})
	if start.Request.Kind != WriteFleetApply {
		t.Fatalf("dispatch kind = %v", start.Request.Kind)
	}
	if model.WritePending() {
		t.Fatal("a real result did not clear the pending-write guard")
	}
	outcome := model.ApplyOutcome()
	if outcome.Headline != "completed" || !strings.Contains(outcome.Detail, "Created 1") {
		t.Fatalf("outcome = %+v, want the real created count", outcome)
	}
	if outcome.BackupID != "backup-apply-1" {
		t.Fatalf("backup = %q, want backup-apply-1", outcome.BackupID)
	}
	if model.PlanPreview().Available {
		t.Fatal("a consumed plan preview was retained")
	}
	if cmd == nil {
		t.Fatal("the result did not queue a status refresh")
	}
	model, _ = model.Update(cmd())
	if reader.calls != before+1 {
		t.Fatalf("status reads = %d, want %d after the result", reader.calls, before+1)
	}
}

func TestFleetLiveGateHidesApplyAndRestore(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, false, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview

	model, msg := fleetUpdateKey(t, model, 'a', "a")
	if msg != nil {
		t.Fatalf("preview-only apply produced %T, want nothing", msg)
	}
	if model.IsConfirmingApply() {
		t.Fatal("preview-only build opened an apply confirmation")
	}
	view := strings.Join(model.Lines(110), "\n")
	if !strings.Contains(view, "Preview only") {
		t.Fatalf("review view is missing the Preview only label:\n%s", view)
	}

	// No key sequence can dispatch an apply in a preview-only build.
	for _, key := range []struct {
		code rune
		text string
	}{{'a', "a"}, {'y', "y"}, {tea.KeyEnter, ""}} {
		model, msg = fleetUpdateKey(t, model, key.code, key.text)
		if _, ok := msg.(StartWriteMsg); ok {
			t.Fatal("a preview-only build dispatched a write")
		}
	}

	// Restore is gated by the same capability.
	model.view = FleetViewRecovery
	model.backups = []FleetBackupView{{ID: "20260101T000000.000000000Z", Files: 1}}
	model.backupIndex = 0
	model.restorePreview = testFleetRestorePreview()
	model, msg = fleetUpdateKey(t, model, 't', "t")
	if msg != nil || model.IsConfirmingRestore() {
		t.Fatalf("preview-only restore produced %T / confirm=%t", msg, model.IsConfirmingRestore())
	}
	if !strings.Contains(model.RestoreOutcome().Headline, "Preview only") {
		t.Fatalf("restore outcome = %+v, want Preview only", model.RestoreOutcome())
	}
}

func TestFleetApplyFixtureHomeWritesThroughHost(t *testing.T) {
	home := t.TempDir()
	configPath := writeFleetFixture(t, home, singleFleetInstance("9router/ag/gemini-3.8-flash-high"))
	cfg := TUIConfig{
		Width:       110,
		Height:      36,
		Color:       false,
		Home:        home,
		FleetConfig: configPath,
		FleetLive:   true,
	}
	cat := Catalog{Home: home, Repo: ""}
	app := NewAppWithScreens(cfg, cat, AppOptions{Ops: NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return &fleetMemoryLog{}, "/tmp/fleet-fixture.log", nil
		},
	})}, AppScreenOptions{
		CatalogRefresh: func(TUIConfig) (Catalog, error) { return cat, nil },
	})

	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "c")
	if app.fleet.ActiveView() != FleetViewReview {
		t.Fatalf("view = %v, want Review", app.fleet.ActiveView())
	}
	if !app.fleet.PlanPreview().Available {
		t.Fatalf("review did not produce a plan: %+v", app.fleet.PlanPreview())
	}
	app, _ = fleetKeyCmd(t, app, "a")
	if !app.fleet.IsConfirmingApply() {
		t.Fatal("apply confirmation did not open in the live fixture build")
	}
	app, cmd := fleetKeyCmd(t, app, "y")
	app = drainFleetApp(t, app, cmd)

	if app.Busy() {
		t.Fatal("the write slot is still busy after the fixture apply")
	}
	outcome := app.fleet.ApplyOutcome()
	if outcome.Headline != "completed" || outcome.BackupID == "" {
		t.Fatalf("apply outcome = %+v, want a completed result with a backup", outcome)
	}
	target := filepath.Join(home, ".config", "opencode", "agents", "sdd-implement-1.md")
	if data, err := os.ReadFile(target); err != nil || len(data) == 0 {
		t.Fatalf("fixture apply did not create the managed agent file: %v", err)
	}
}

func TestFleetApplyPendingIgnoresNavigationResizeAndRepeat(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, true, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview
	model, _ = model.requestApply()
	model, msg := fleetUpdateKey(t, model, 'y', "y")
	if _, ok := msg.(StartWriteMsg); !ok {
		t.Fatalf("confirm produced %T, want StartWriteMsg", msg)
	}

	// Navigation and a resize cannot start, submit or cancel a second write.
	model, msg = fleetUpdateKey(t, model, 'a', "a")
	if msg != nil {
		t.Fatalf("navigation during a pending apply produced %T", msg)
	}
	model, _ = fleetUpdateKey(t, model, 'j', "j")
	model.SetSize(50, 6)
	model, msg = fleetUpdateKey(t, model, 'y', "y")
	if msg != nil {
		t.Fatalf("a repeat key during a pending apply produced %T", msg)
	}
	if !model.WritePending() {
		t.Fatal("resize/navigation cleared the pending-write guard")
	}
}

func TestFleetApplyConfirmationCannotConfirmUnseenAtSmallSize(t *testing.T) {
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview()}
	model, _ := newFleetWriteModel(t, true, planner)
	model.planPreview = planner.plan
	model.view = FleetViewReview
	model, _ = model.requestApply()
	if !model.IsConfirmingApply() {
		t.Fatal("confirmation did not open")
	}

	// Below the usable geometry the confirmation is not rendered, so y must
	// not dispatch it.
	model.SetSize(50, 6)
	model, msg := fleetUpdateKey(t, model, 'y', "y")
	if msg != nil {
		t.Fatalf("unseen confirmation dispatched %T", msg)
	}
	if !model.IsConfirmingApply() {
		t.Fatal("an unseen y cancelled the confirmation instead of leaving it pending")
	}

	// Restoring the usable geometry makes the same confirmation visible and
	// confirmable.
	model.SetSize(110, 34)
	model, msg = fleetUpdateKey(t, model, 'y', "y")
	if _, ok := msg.(StartWriteMsg); !ok {
		t.Fatalf("visible confirmation produced %T, want StartWriteMsg", msg)
	}
}

func TestFleetSingleSlotCrossKindRefusal(t *testing.T) {
	host := NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return &fleetMemoryLog{}, "/tmp/fleet-slot.log", nil
		},
		ApplyFleet: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyFleetRestore: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		SaveFleetSource: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
	})
	home := t.TempDir()
	apply := WriteRequest{Kind: WriteFleetApply, Fleet: FleetWriteRequest{
		Home: home, ConfigPath: "/tmp/fleet.json", PlanID: "plan-1", Token: "t1", Source: "fleet.json",
	}}
	restore := WriteRequest{Kind: WriteFleetRestore, Fleet: FleetWriteRequest{
		Home: home, BackupID: "20260101T000000.000000000Z", Token: "t2", Source: "fleet backup",
	}}
	save := WriteRequest{Kind: WriteSourceSave, Fleet: FleetWriteRequest{
		Home: home, ConfigPath: "/tmp/fleet.json", Token: "t3", Source: "fleet.json",
	}}

	started, cmd, err := host.BeginWrite(apply)
	if err != nil {
		t.Fatalf("apply BeginWrite: %v", err)
	}
	if _, _, err := host.BeginWrite(restore); !errors.Is(err, ErrWriteInProgress) {
		t.Fatalf("restore during apply = %v, want ErrWriteInProgress", err)
	}
	if _, _, err := host.BeginWrite(save); !errors.Is(err, ErrWriteInProgress) {
		t.Fatalf("save during apply = %v, want ErrWriteInProgress", err)
	}
	cmd()
	host.Complete(started.ID)

	restoreStarted, restoreCmd, err := host.BeginWrite(restore)
	if err != nil {
		t.Fatalf("restore BeginWrite after release: %v", err)
	}
	if _, _, err := host.BeginWrite(apply); !errors.Is(err, ErrWriteInProgress) {
		t.Fatalf("apply during restore = %v, want ErrWriteInProgress", err)
	}
	restoreCmd()
	host.Complete(restoreStarted.ID)
}

func TestFleetAppUsesProductionAdaptersWithoutStartupReads(t *testing.T) {
	home := t.TempDir()
	cfg := TUIConfig{Width: 110, Height: 32, Home: home, FleetConfig: filepath.Join(home, "fleet.json")}
	app := NewApp(cfg, Catalog{Home: home})

	if _, ok := app.fleetReader.(AppOpsFleetReader); !ok {
		t.Fatalf("fleet reader = %T, want AppOpsFleetReader", app.fleetReader)
	}
	if _, ok := app.fleetEditor.(AppOpsFleetEditor); !ok {
		t.Fatalf("fleet editor = %T, want AppOpsFleetEditor", app.fleetEditor)
	}
	if _, ok := app.fleetPlanner.(AppOpsFleetPlanner); !ok {
		t.Fatalf("fleet planner = %T, want AppOpsFleetPlanner", app.fleetPlanner)
	}
	if app.fleet.hasLoaded || app.fleet.IsLoading() {
		t.Fatal("constructing the app read the fleet source before Fleet was opened")
	}
	if app.fleet.Status().TotalInstances != 0 {
		t.Fatal("startup populated fleet inventory")
	}
}

func TestFleetAppPreviewOnlyCannotDispatch(t *testing.T) {
	var fleetCalls int
	host := NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return &fleetMemoryLog{}, "/tmp/fleet-preview.log", nil
		},
		ApplyFleet: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			fleetCalls++
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
		ApplyFleetRestore: func(FleetWriteRequest, string, io.Writer) (appops.ApplyResult, error) {
			fleetCalls++
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
	})
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview(), restore: testFleetRestorePreview()}
	cfg := TUIConfig{Width: 110, Height: 32, Home: t.TempDir(), FleetConfig: "/test/fleet.json"}
	cat := Catalog{Home: cfg.Home}
	app := NewAppWithScreens(cfg, cat, AppOptions{Ops: host}, AppScreenOptions{
		FleetReader:  &trackingFleetReader{customRes: testFleetStatusView()},
		FleetPlanner: planner,
		CatalogRefresh: func(TUIConfig) (Catalog, error) {
			return cat, nil
		},
	})

	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "c")
	app, cmd := fleetKeyCmd(t, app, "a")
	app = drainFleetApp(t, app, cmd)
	app, cmd = fleetKeyCmd(t, app, "y")
	app = drainFleetApp(t, app, cmd)
	if fleetCalls != 0 {
		t.Fatalf("preview-only build ran %d fleet write seam(s)", fleetCalls)
	}
	view := strings.Join(app.fleet.Lines(110), "\n")
	if !strings.Contains(view, "Preview only") {
		t.Fatalf("preview-only review is missing the label:\n%s", view)
	}
}

type fleetMemoryLog struct {
	mu      sync.Mutex
	content strings.Builder
	flushed bool
	closed  bool
}

func (l *fleetMemoryLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.content.Write(p)
}

func (l *fleetMemoryLog) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushed = true
	return nil
}

func (l *fleetMemoryLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

func opIDForTest(request uint64) appops.OperationID {
	return appops.OperationID{Request: request, Generation: 1}
}

// fleetKeyCmd sends one key and returns the App with the unexecuted command.
func fleetKeyCmd(t *testing.T, app App, key string) (App, tea.Cmd) {
	t.Helper()
	var code rune
	var text string
	switch key {
	case "enter":
		code = tea.KeyEnter
	case "esc":
		code = tea.KeyEscape
	default:
		code = rune(key[0])
		text = key
	}
	next, cmd := app.Update(tea.KeyPressMsg{Code: code, Text: text})
	return next.(App), cmd
}

// drainFleetApp executes a command and feeds every immediate message back,
// including Batches. Timers are skipped.
func drainFleetApp(t *testing.T, app App, cmd tea.Cmd) App {
	t.Helper()
	for _, msg := range fleetCmdMessages(cmd) {
		if msg == nil {
			continue
		}
		next, follow := app.Update(msg)
		app = next.(App)
		if follow != nil {
			app = drainFleetApp(t, app, follow)
		}
	}
	return app
}

func fleetCmdMessages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var msgs []tea.Msg
			for _, sub := range batch {
				msgs = append(msgs, fleetCmdMessages(sub)...)
			}
			return msgs
		}
		return []tea.Msg{msg}
	case <-time.After(10 * time.Second):
		return nil
	}
}
