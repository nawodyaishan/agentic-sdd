package tui

// B3/T06 composed-confirmation tests (Feature 022, FR-05, AC-05, B-01/B-02).
//
// Every case drives the composed root App with fake seams. A key that would
// dispatch a write is detected by inspecting the StartWriteMsg it returns; that
// message is never delivered back into Update, so no OpsHost or Fleet write runs.

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	b3MCPTarget   = "/b3-home/.config/opencode/opencode.json"
	b3BackupID    = "b3-backup"
	b3FleetSource = "/b3/fleet.json"
)

func b3Update(t *testing.T, app App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want tui.App", next)
	}
	return updated, cmd
}

func b3Press(t *testing.T, app App, r rune) App {
	t.Helper()
	app, _ = b3Update(t, app, tea.KeyPressMsg{Code: r, Text: string(r)})
	return app
}

func b3Named(t *testing.T, app App, code rune) (App, tea.Cmd) {
	t.Helper()
	return b3Update(t, app, tea.KeyPressMsg{Code: code})
}

func b3Resize(t *testing.T, app App, width, height int) App {
	t.Helper()
	app, _ = b3Update(t, app, tea.WindowSizeMsg{Width: width, Height: height})
	return app
}

// b3DispatchesWrite reports whether cmd would hand a write request to the root
// shell. It only inspects the produced message.
func b3DispatchesWrite(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case StartWriteMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if b3DispatchesWrite(c) {
				return true
			}
		}
	}
	return false
}

// b3Visible reports whether every essential decision fact is on the rendered
// screen. The sidebar column (everything left of the divider) is removed per
// row, so a wrapped body line is matched as the user reads it; whitespace is
// removed on both sides so wrapping cannot hide a match.
func b3Visible(app App, facts []string) bool {
	var body []string
	for _, row := range strings.Split(ansi.Strip(app.View().Content), "\n") {
		if i := strings.Index(row, "│"); i >= 0 {
			row = row[i+len("│"):]
		}
		body = append(body, row)
	}
	plain := strings.Join(strings.Fields(strings.Join(body, "\n")), "")
	for _, fact := range facts {
		if !strings.Contains(plain, strings.Join(strings.Fields(fact), "")) {
			return false
		}
	}
	return true
}

func b3Screen(app App) string { return ansi.Strip(app.View().Content) }

// b3MCPApp opens the composed app on MCP Tools with a fake reader and planner,
// then places the MCP model in the requested confirmation state after the
// geometry is applied (MCP SetSize resets confirmation selection by design).
func b3MCPApp(t *testing.T, width, height int, view MCPView) App {
	t.Helper()
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()
	app := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{})
	app = b3Resize(t, app, width, height)
	app.activeRoute = RouteInstall
	app.installSection = installSectionMCP
	app.focus = FocusContent // the user is inside the content pane, not the sidebar
	app.contentScroll = 0
	mcp := NewMCPModelWithSeams(cfg, &fakeMCPReader{}, &fakeMCPPlanner{})
	mcp.SetSize(app.childContentWidth(), app.childBodyHeight())
	mcp.planPreview = MCPPlanPreviewView{
		Token:      "b3-mcp-token",
		TargetPath: b3MCPTarget,
		BaseDigest: "sha256:b3base",
		PlanDigest: "sha256:b3plan",
		Selected:   []string{"codegraph"},
	}
	mcp.backups = []MCPBackupView{{ID: b3BackupID}}
	mcp.restorePreview = MCPRestorePreviewView{
		Available:  true,
		BackupID:   b3BackupID,
		TargetPath: b3MCPTarget,
		BaseDigest: "sha256:b3base",
		Token:      "b3-mcp-restore-token",
	}
	mcp.view = view
	mcp.confirmSelection = false
	mcp.confirmRestore = view == MCPViewRecovery
	app.mcp = mcp
	return app
}

func b3MCPApplyFacts() []string {
	return []string{"Apply configuration changes", b3MCPTarget, "Safety policy", "Yes, apply changes", "No, cancel"}
}

func b3MCPRestoreFacts() []string {
	return []string{"Restore managed MCP backup", b3BackupID, "Safety policy", "Yes, restore backup", "No, cancel"}
}

// b3AssertNoHiddenDispatch presses y and fails if a write would be dispatched
// while the essential facts are not all on screen at dispatch time.
func b3AssertNoHiddenDispatch(t *testing.T, app App, facts []string, label string) App {
	t.Helper()
	visible := b3Visible(app, facts)
	screen := b3Screen(app)
	t.Logf("%s: decision visible=%v", label, visible)
	next, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if b3DispatchesWrite(cmd) && !visible {
		t.Fatalf("%s: affirmative y dispatched a write while essential decision content was hidden:\n%s", label, screen)
	}
	if !visible && !strings.Contains(next.notice, "not fully visible") {
		t.Fatalf("%s: hidden decision refused without the visibility notice (notice=%q):\n%s", label, next.notice, screen)
	}
	if visible && !b3DispatchesWrite(cmd) {
		t.Fatalf("%s: decision is visible but y did not dispatch (over-blocked):\n%s", label, screen)
	}
	return app
}

// TestB3MCPApplyHiddenDecisionCannotDispatch reproduces B-01 for the MCP apply
// confirmation at the reviewed 100x16 size and smaller layouts. Positive
// control: at 100x30 the decision is visible and y still dispatches.
func TestB3MCPApplyHiddenDecisionCannotDispatch(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 16}, {100, 12}, {80, 10}, {60, 8}} {
		app := b3MCPApp(t, size.w, size.h, MCPViewConfirm)
		b3AssertNoHiddenDispatch(t, app, b3MCPApplyFacts(), "mcp apply "+sizeLabel(size.w, size.h))
	}

	app := b3MCPApp(t, 100, 30, MCPViewConfirm)
	if !b3Visible(app, b3MCPApplyFacts()) {
		t.Fatalf("positive control: 100x30 MCP apply decision not visible:\n%s", b3Screen(app))
	}
	_, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !b3DispatchesWrite(cmd) {
		t.Fatal("positive control: visible 100x30 MCP apply did not dispatch on y")
	}
}

// TestB3MCPRestoreHiddenDecisionCannotDispatch covers the MCP recovery restore
// confirmation with the same invariant at the reviewed size.
func TestB3MCPRestoreHiddenDecisionCannotDispatch(t *testing.T) {
	for _, size := range []struct{ w, h int }{{100, 16}, {80, 10}} {
		app := b3MCPApp(t, size.w, size.h, MCPViewRecovery)
		b3AssertNoHiddenDispatch(t, app, b3MCPRestoreFacts(), "mcp restore "+sizeLabel(size.w, size.h))
	}

	// Positive control: with the decision visible, a real Yes still dispatches,
	// so the invariant above cannot pass merely because nothing can dispatch.
	app := b3MCPApp(t, 100, 30, MCPViewRecovery)
	if !b3Visible(app, b3MCPRestoreFacts()) {
		t.Fatalf("positive control: 100x30 MCP restore decision not visible:\n%s", b3Screen(app))
	}
	_, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !b3DispatchesWrite(cmd) {
		t.Fatal("positive control: visible 100x30 MCP restore did not dispatch on y")
	}
}

// TestB3MCPHiddenDecisionCannotBeRevealedByResizeOrScroll starts at a visible
// size, then shrinks or scrolls the composed screen. The affirmative key must
// fail closed at the moment it arrives.
func TestB3MCPHiddenDecisionCannotBeRevealedByResizeOrScroll(t *testing.T) {
	app := b3MCPApp(t, 100, 30, MCPViewConfirm)
	app = b3Resize(t, app, 100, 12)
	app.mcp.confirmSelection = true // a stale Yes choice must not survive a resize
	b3AssertNoHiddenDispatch(t, app, b3MCPApplyFacts(), "mcp after resize to 100x12")

	scrolled := b3MCPApp(t, 100, 12, MCPViewConfirm)
	scrolled, _ = b3Named(t, scrolled, tea.KeyPgDown)
	b3AssertNoHiddenDispatch(t, scrolled, b3MCPApplyFacts(), "mcp after pgdown at 100x12")
}

// TestB3MCPConfirmationOwnsRouteKeys reproduces B-02: a route-jump key pressed
// while the MCP confirmation is open must not navigate away from the decision.
func TestB3MCPConfirmationOwnsRouteKeys(t *testing.T) {
	app := b3MCPApp(t, 100, 30, MCPViewConfirm)
	app = b3Press(t, app, '1')
	if app.activeRoute != RouteInstall || app.mcp.ActiveView() != MCPViewConfirm {
		t.Fatalf("route key 1 navigated away from the MCP confirmation (route=%v view=%v)", app.activeRoute, app.mcp.ActiveView())
	}
}

// TestB3MCPConfirmationTabIsTruthful reproduces the advertised "Tab/Arrow
// toggle" control: Tab must move the Yes/No choice inside the confirmation, and
// Esc must still cancel back to the preview without a write.
func TestB3MCPConfirmationTabIsTruthful(t *testing.T) {
	app := b3MCPApp(t, 100, 30, MCPViewConfirm)
	app, _ = b3Named(t, app, tea.KeyTab)
	if !app.mcp.confirmSelection {
		t.Fatalf("Tab did not toggle the advertised confirmation choice:\n%s", b3Screen(app))
	}
	// The Yes focused by Tab must be the choice Enter applies.
	if _, cmd := b3Named(t, app, tea.KeyEnter); !b3DispatchesWrite(cmd) {
		t.Fatalf("Enter after Tab did not apply the focused Yes choice:\n%s", b3Screen(app))
	}

	cancel := b3MCPApp(t, 100, 30, MCPViewConfirm)
	cancel, cmd := b3Named(t, cancel, tea.KeyEscape)
	if b3DispatchesWrite(cmd) || cancel.mcp.ActiveView() != MCPViewPreview {
		t.Fatalf("Esc on confirmation: dispatched=%v view=%v, want no write and preview", b3DispatchesWrite(cmd), cancel.mcp.ActiveView())
	}
}

// TestB3FleetRestoreHiddenDecisionCannotDispatch reproduces B-01 for the Fleet
// restore confirmation at the reviewed 100x20 size. The prompt sits after the
// restore diff, so at this size it is off screen.
func TestB3FleetRestoreHiddenDecisionCannotDispatch(t *testing.T) {
	facts := b3FleetRestoreFacts()
	for _, size := range []struct{ w, h int }{{100, 20}, {80, 12}, {60, 8}} {
		app := b3FleetApp(t, size.w, size.h, FleetViewRecovery)
		b3AssertNoHiddenDispatch(t, app, facts, "fleet restore "+sizeLabel(size.w, size.h))
	}

	app := b3FleetApp(t, 100, 40, FleetViewRecovery)
	if !b3Visible(app, facts) {
		t.Fatalf("positive control: 100x40 Fleet restore decision not visible:\n%s", b3Screen(app))
	}
	_, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !b3DispatchesWrite(cmd) {
		t.Fatal("positive control: visible 100x40 Fleet restore did not dispatch on y")
	}
}

// TestB3FleetHiddenDecisionCannotBeRevealedByResizeOrScroll mirrors the MCP
// check for Fleet restore: a visible decision that is then shrunk or scrolled
// away must not remain confirmable.
func TestB3FleetHiddenDecisionCannotBeRevealedByResizeOrScroll(t *testing.T) {
	app := b3FleetApp(t, 100, 20, FleetViewRecovery)
	app = b3Resize(t, app, 80, 12)
	app.fleet.confirmRestore = true // a stale confirmation must not survive the shrink
	b3AssertNoHiddenDispatch(t, app, b3FleetRestoreFacts(), "fleet restore after resize to 80x12")

	scrolled := b3FleetApp(t, 100, 20, FleetViewRecovery)
	scrolled.contentScroll = 6 // the user has scrolled the restore diff down
	b3AssertNoHiddenDispatch(t, scrolled, b3FleetRestoreFacts(), "fleet restore scrolled at 100x20")
}

// TestB3FleetLongTargetPathWrapsAndStillConfirms guards the wrap-aware check: a
// long managed target path wraps across body rows beside the sidebar. The
// decision is still fully on screen and must confirm; it must not be refused
// merely because the sidebar text interleaves the wrapped path.
func TestB3FleetLongTargetPathWrapsAndStillConfirms(t *testing.T) {
	app := b3FleetApp(t, 110, 40, FleetViewReview)
	longTarget := "/tmp/b3/" + strings.Repeat("managed-target-segment-", 4) + "/.config/opencode/agents"
	app.fleet.planPreview.Plan.TargetRoot = longTarget
	facts := []string{"Apply to OpenCode files? [y/N]", "This writes the", longTarget, "Press y to confirm"}
	if !b3Visible(app, facts) {
		t.Fatalf("long target path: decision not visible in the body window:\n%s", b3Screen(app))
	}
	if _, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"}); !b3DispatchesWrite(cmd) {
		t.Fatal("visible decision with a wrapped long target path did not dispatch on y")
	}
}

// TestB3FleetApplyHiddenDecisionCannotDispatch covers the Fleet deployment apply
// confirmation at the reviewed 100x20 size.
func TestB3FleetApplyHiddenDecisionCannotDispatch(t *testing.T) {
	facts := b3FleetApplyFacts()
	for _, size := range []struct{ w, h int }{{100, 20}, {80, 12}} {
		app := b3FleetApp(t, size.w, size.h, FleetViewReview)
		b3AssertNoHiddenDispatch(t, app, facts, "fleet apply "+sizeLabel(size.w, size.h))
	}

	app := b3FleetApp(t, 100, 40, FleetViewReview)
	if !b3Visible(app, facts) {
		t.Fatalf("positive control: 100x40 Fleet apply decision not visible:\n%s", b3Screen(app))
	}
	_, cmd := b3Update(t, app, tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !b3DispatchesWrite(cmd) {
		t.Fatal("positive control: visible 100x40 Fleet apply did not dispatch on y")
	}
}

// TestB3FleetConfirmationOwnsRootKeys keeps Fleet's existing ownership and
// default-No behavior pinned while the new visibility gate is added.
func TestB3FleetConfirmationOwnsRootKeys(t *testing.T) {
	app := b3FleetApp(t, 100, 40, FleetViewRecovery)
	app = b3Press(t, app, '1')
	if app.activeRoute != RouteInstall || !app.fleet.IsConfirmingRestore() {
		t.Fatalf("route key 1 escaped the Fleet restore confirmation (route=%v confirming=%v)", app.activeRoute, app.fleet.IsConfirmingRestore())
	}
	app, cmd := b3Named(t, app, tea.KeyEnter)
	if b3DispatchesWrite(cmd) || app.fleet.IsConfirmingRestore() {
		t.Fatal("Enter on the default-No Fleet restore dispatched or stayed confirming")
	}
}

func b3FleetApp(t *testing.T, width, height int, view FleetView) App {
	t.Helper()
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home, FleetConfig: b3FleetSource, FleetLive: true}.Normalize()
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	planner := &fakeFleetPlanner{plan: testFleetPlanPreview(), restore: testFleetRestorePreview()}
	app := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{
		FleetReader:  reader,
		FleetEditor:  &fakeFleetEditor{},
		FleetPlanner: planner,
	})
	app = b3Resize(t, app, width, height)
	app.activeRoute = RouteInstall
	app.installSection = installSectionFleet
	app.focus = FocusContent
	app.contentScroll = 0

	fleet := NewFleetModelWithSeams(cfg, reader, &fakeFleetEditor{}, planner)
	status := testFleetStatusView()
	status.SourcePath = b3FleetSource
	fleet.status = status
	fleet.hasLoaded = true
	fleet.SetSize(app.childContentWidth(), app.childBodyHeight())
	fleet.view = view
	fleet.planPreview = testFleetPlanPreview()
	fleet.restorePreview = testFleetRestorePreview()
	fleet.backups = []FleetBackupView{{ID: b3BackupID, Path: "/b3/backups/" + b3BackupID}}
	fleet.backupIndex = 0
	if view == FleetViewRecovery {
		fleet.confirmRestore = true
	} else {
		fleet.confirmApply = true
	}
	app.fleet = fleet
	return app
}

func b3FleetRestoreFacts() []string {
	return []string{"Restore this managed backup? [y/N]", "This reverses the recorded transaction", b3BackupID, "Press y to confirm"}
}

func b3FleetApplyFacts() []string {
	return []string{"Apply to OpenCode files? [y/N]", "This writes the", "/tmp/fleet-target", "Press y to confirm"}
}

func sizeLabel(w, h int) string { return fmt.Sprintf("%dx%d", w, h) }
