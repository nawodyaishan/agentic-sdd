package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/skillsync"
)

func testFleetCatalog() Catalog {
	return Catalog{
		Home: "/tmp/test-home",
		Repo: "/tmp/test-repo",
		Items: []CatalogItem{
			{
				ID:     "sdd-router",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
			},
		},
	}
}

func testFleetStatusView() FleetStatusView {
	return FleetStatusView{
		Available:             true,
		SourcePath:            "/test/fleet.json",
		SourceState:           "loaded",
		PersonalConfigSummary: "Loaded (3 instances configured)",
		InstalledSummary:      "2 current, 1 drifted",
		TotalInstances:        3,
		ManagedCount:          3,
		EnabledCount:          2,
		DisabledCount:         1,
		DriftedCount:          1,
		UnmanagedCount:        0,
		LastOutcome:           "Clean status check",
		NextAction:            "Inspect profiles or review changes",
		RuntimeStatus:         "not checked",
		Instances: []FleetInstanceView{
			{
				InstanceID:        "sdd-implement-9r-1",
				Role:              "sdd-implementer",
				ProviderRoute:     "9router",
				Model:             "ag/gemini-3.8-flash-high",
				CompactModel:      "gemini-3.8-flash-high",
				Variant:           "high",
				Enabled:           true,
				InstalledState:    "managed current",
				SourceProvenance:  "opencode-pack/agents",
				ChangedCount:      0,
				PermissionSummary: "tools: read, edit, bash; network: local",
				RuntimeState:      "not checked",
			},
			{
				InstanceID:        "sdd-orchestrator",
				Role:              "sdd-orchestrator",
				ProviderRoute:     "opencode-go",
				Model:             "deepseek-v4.1-flash",
				CompactModel:      "deepseek-v4.1-flash",
				Variant:           "default",
				Enabled:           true,
				InstalledState:    "drifted",
				SourceProvenance:  "personal override",
				ChangedCount:      2,
				PermissionSummary: "tools: all; approvals: required",
				RuntimeState:      "not checked",
				DriftDetail:       "model changed from claude-sonnet",
			},
			{
				InstanceID:        "sdd-spec-9r",
				Role:              "sdd-spec-reviewer",
				ProviderRoute:     "9router",
				Model:             "claude-opus-5.5",
				CompactModel:      "opus-5.5",
				Variant:           "max",
				Enabled:           false,
				InstalledState:    "disabled (managed removal)",
				SourceProvenance:  "preset: free-first",
				ChangedCount:      0,
				PermissionSummary: "tools: read-only; network: none",
				RuntimeState:      "not checked",
			},
		},
	}
}

type trackingFleetReader struct {
	calls     int
	lastPath  string
	customErr error
	customRes FleetStatusView
}

func (r *trackingFleetReader) Status(ctx context.Context, source string) (FleetStatusView, error) {
	r.calls++
	r.lastPath = source
	if r.customErr != nil {
		return FleetStatusView{}, r.customErr
	}
	res := r.customRes
	if res.SourcePath == "" && source != "" {
		res.SourcePath = source
	}
	return res, nil
}

func fleetPress(t *testing.T, app App, code rune, text string) App {
	t.Helper()
	var msg tea.KeyPressMsg
	if text != "" {
		msg = tea.KeyPressMsg{Code: code, Text: text}
	} else {
		msg = tea.KeyPressMsg{Code: code}
	}
	next, cmd := app.Update(msg)
	a := next.(App)
	if cmd != nil {
		msgOut := cmd()
		if msgOut != nil {
			next2, _ := a.Update(msgOut)
			a = next2.(App)
		}
	}
	return a
}

func fleetKey(t *testing.T, app App, key string) App {
	t.Helper()
	var code rune
	text := key
	switch key {
	case "enter":
		code = tea.KeyEnter
		text = ""
	case "esc":
		code = tea.KeyEscape
		text = ""
	case "down":
		code = tea.KeyDown
		text = ""
	case "up":
		code = tea.KeyUp
		text = ""
	case "tab":
		code = tea.KeyTab
		text = ""
	case "ctrl+u":
		msg := tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
		next, cmd := app.Update(msg)
		a := next.(App)
		if cmd != nil {
			if msgOut := cmd(); msgOut != nil {
				next2, _ := a.Update(msgOut)
				a = next2.(App)
			}
		}
		return a
	default:
		if len(key) == 1 {
			code = rune(key[0])
		}
	}
	return fleetPress(t, app, code, text)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestInstallHubThreeEntriesAndWrap(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// 1. Navigate to RouteInstall (hub)
	app = fleetKey(t, app, "2")
	if app.ActiveRoute() != RouteInstall {
		t.Fatalf("route = %v, want RouteInstall", app.ActiveRoute())
	}
	if app.installSection != installSectionHub {
		t.Fatalf("installSection = %v, want installSectionHub", app.installSection)
	}
	if app.installHubIndex != 0 {
		t.Fatalf("initial hubIndex = %d, want 0", app.installHubIndex)
	}

	// 2. Wrap forward across all 3 entries: 0 -> 1 -> 2 -> 0
	app = fleetKey(t, app, "j")
	if app.installHubIndex != 1 {
		t.Fatalf("after j, hubIndex = %d, want 1", app.installHubIndex)
	}
	app = fleetKey(t, app, "down")
	if app.installHubIndex != 2 {
		t.Fatalf("after down, hubIndex = %d, want 2", app.installHubIndex)
	}
	app = fleetKey(t, app, "j")
	if app.installHubIndex != 0 {
		t.Fatalf("after wrap j, hubIndex = %d, want 0", app.installHubIndex)
	}

	// 3. Wrap backward: 0 -> 2 -> 1 -> 0
	app = fleetKey(t, app, "k")
	if app.installHubIndex != 2 {
		t.Fatalf("after backward k, hubIndex = %d, want 2", app.installHubIndex)
	}
	app = fleetKey(t, app, "up")
	if app.installHubIndex != 1 {
		t.Fatalf("after backward up, hubIndex = %d, want 1", app.installHubIndex)
	}
	app = fleetKey(t, app, "k")
	if app.installHubIndex != 0 {
		t.Fatalf("after backward k, hubIndex = %d, want 0", app.installHubIndex)
	}

	// 4. Enter on index 0 opens Install screen
	app = fleetKey(t, app, "enter")
	if app.installSection != installSectionInstall {
		t.Fatalf("enter on index 0: installSection = %v, want installSectionInstall", app.installSection)
	}
	// Return to hub via Esc
	app = fleetKey(t, app, "esc")
	// If install is at clients step, Esc from nav/hub closes
	app.installSection = installSectionHub

	// 5. Enter on index 1 opens Backups screen
	app.installHubIndex = 1
	app = fleetKey(t, app, "enter")
	if app.installSection != installSectionBackups {
		t.Fatalf("enter on index 1: installSection = %v, want installSectionBackups", app.installSection)
	}
	// Esc returns to hub
	app = fleetKey(t, app, "esc")
	if app.installSection != installSectionHub {
		t.Fatalf("esc from backups: installSection = %v, want installSectionHub", app.installSection)
	}

	// 6. Enter on index 2 opens Fleet Manager
	app.installHubIndex = 2
	app = fleetKey(t, app, "enter")
	if app.installSection != installSectionFleet {
		t.Fatalf("enter on index 2: installSection = %v, want installSectionFleet", app.installSection)
	}

	// Breadcrumb on Fleet
	if bc := app.fleet.Breadcrumb(); bc != "Install & Manage / Fleet Manager" {
		t.Fatalf("Fleet breadcrumb = %q, want 'Install & Manage / Fleet Manager'", bc)
	}

	// Esc from Fleet Overview returns to Hub
	app = fleetKey(t, app, "esc")
	if app.installSection != installSectionHub {
		t.Fatalf("esc from fleet overview: installSection = %v, want installSectionHub", app.installSection)
	}

	// 7. Shortcut 'f' opens Fleet directly from hub
	app = fleetKey(t, app, "f")
	if app.installSection != installSectionFleet {
		t.Fatalf("pressing f from hub: installSection = %v, want installSectionFleet", app.installSection)
	}

	// 8. Home affordance: 'H' returns to Home route
	app = fleetKey(t, app, "H")
	if app.ActiveRoute() != RouteHome {
		t.Fatalf("pressing H from Fleet: route = %v, want RouteHome", app.ActiveRoute())
	}
}

func TestFleetLazyNoReadWhenNeverOpened(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// Navigate across Home, Explore, Activity, and Install Hub
	app = fleetKey(t, app, "3") // Explore
	if app.ActiveRoute() != RouteExplore {
		t.Fatalf("route = %v, want RouteExplore", app.ActiveRoute())
	}
	app = fleetKey(t, app, "4") // Activity
	if app.ActiveRoute() != RouteActivity {
		t.Fatalf("route = %v, want RouteActivity", app.ActiveRoute())
	}
	app = fleetKey(t, app, "2") // Install Hub
	if app.ActiveRoute() != RouteInstall || app.installSection != installSectionHub {
		t.Fatalf("route = %v, section = %v, want Install Hub", app.ActiveRoute(), app.installSection)
	}
	app = fleetKey(t, app, "1") // Home
	if app.ActiveRoute() != RouteHome {
		t.Fatalf("route = %v, want RouteHome", app.ActiveRoute())
	}

	// Ensure fleet reader was NEVER invoked
	if reader.calls != 0 {
		t.Fatalf("lazy violation: reader.Status() called %d times without Fleet ever being opened", reader.calls)
	}

	// Now explicitly open Fleet
	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	if app.installSection != installSectionFleet {
		t.Fatalf("section = %v, want installSectionFleet", app.installSection)
	}
	if reader.calls != 1 {
		t.Fatalf("after opening fleet: reader.Status() called %d times, want 1", reader.calls)
	}
}

func TestFleetOverviewRendering(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false, FleetConfig: "/test/fleet.json"}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// Open Fleet
	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")

	view := strings.Join(app.fleet.Lines(100), "\n")

	// Truthful Overview facts
	for _, want := range []string{
		"Install & Manage / Fleet Manager · Overview",
		"Personal configuration (fleet.json)",
		"Source path: /test/fleet.json",
		"Source state: loaded",
		"Summary: Loaded (3 instances configured)",
		"Installed OpenCode agents",
		"Status: 2 current, 1 drifted",
		"Total configured instances: 3 · Managed: 3",
		"Enabled: 2 · Disabled: 1",
		"Drifted: 1 · Unmanaged: 0",
		"Last operation outcome: Clean status check",
		"Recommended action: Inspect profiles or review changes",
		"Runtime verification: not checked (offline default, no models invoked)",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("Overview missing %q:\n%s", want, view)
		}
	}

	// Strict negative assertions: NO "Ready" label, no save/apply writes
	if strings.Contains(strings.ToLower(view), "ready") {
		t.Errorf("Overview contains forbidden 'ready' label:\n%s", view)
	}
	for _, forbidden := range []string{"apply", "save", "write", "install now"} {
		if strings.Contains(strings.ToLower(view), forbidden) && !strings.Contains(view, "Install & Manage") {
			t.Errorf("Overview contains forbidden write action %q:\n%s", forbidden, view)
		}
	}
}

func TestFleetProfilesSearchAndFilter(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// Open Fleet and switch to Profiles
	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "p") // Switch to Profiles view

	if app.fleet.ActiveView() != FleetViewProfiles {
		t.Fatalf("ActiveView = %v, want FleetViewProfiles", app.fleet.ActiveView())
	}

	// Initially all 3 instances present
	if len(app.fleet.FilteredInstances()) != 3 {
		t.Fatalf("initial filtered count = %d, want 3", len(app.fleet.FilteredInstances()))
	}

	// Press '/' to start filtering
	app = fleetKey(t, app, "/")
	if !app.fleet.IsInputFocused() {
		t.Fatal("expected IsInputFocused() = true while filtering")
	}

	// Type query "orchestrator" with literal characters (including 'o', 'r', 'c', 'h')
	for _, ch := range "orchestrator" {
		app = fleetPress(t, app, ch, string(ch))
	}

	filtered := app.fleet.FilteredInstances()
	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 for 'orchestrator'", len(filtered))
	}
	if filtered[0].InstanceID != "sdd-orchestrator" {
		t.Fatalf("filtered instance = %q, want 'sdd-orchestrator'", filtered[0].InstanceID)
	}

	// Verify literal keys: type 'q', 'j', 'k', '1', '2' while filtering; they must not quit or switch routes!
	app = fleetPress(t, app, 'q', "q")
	if app.quitting {
		t.Fatal("literal 'q' in filter must not quit")
	}
	if app.ActiveRoute() != RouteInstall {
		t.Fatal("filter input must not trigger route switch")
	}

	// Press Esc to exit filter mode
	app = fleetKey(t, app, "esc")
	if app.fleet.IsInputFocused() {
		t.Fatal("after esc: IsInputFocused() should be false")
	}

	// Press Esc again to clear filter
	app = fleetKey(t, app, "esc")
	if len(app.fleet.FilteredInstances()) != 3 {
		t.Fatalf("after clearing filter: count = %d, want 3", len(app.fleet.FilteredInstances()))
	}
}

func TestFleetProfilesDetailCard(t *testing.T) {
	cfg := TUIConfig{Width: 120, Height: 35, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	// Open Fleet Profiles
	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "p")

	// Render side-by-side view (width 120 >= 110)
	view := strings.Join(app.fleet.Lines(120), "\n")

	// Checks detail card content for the selected instance (sdd-implement-9r-1)
	for _, want := range []string{
		"Instance ID: sdd-implement-9r-1",
		"Canonical Role: sdd-implementer (canonical policy role)",
		"Provider Route: 9router",
		"Full Model: ag/gemini-3.8-flash-high (variant: high)",
		"Compact Model: gemini-3.8-flash-high",
		"Source Provenance: opencode-pack/agents",
		"Engine Permissions: tools: read, edit, bash; network: local",
		"Enabled: true",
		"Installed State: managed current",
		"Runtime Status: unverified (not checked — requires OpenCode",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("Profiles detail missing %q:\n%s", want, view)
		}
	}

	// Ensure no entitlement claims
	for _, forbidden := range []string{"chatgpt authenticated", "free forever", "ready"} {
		if strings.Contains(strings.ToLower(view), forbidden) {
			t.Errorf("Profiles detail contains forbidden claim %q:\n%s", forbidden, view)
		}
	}
}

func TestFleetResponsiveSizes(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "p") // Profiles

	// 1. Width 110: side-by-side contains divider
	lines110 := app.fleet.Lines(110)
	if len(lines110) == 0 {
		t.Fatal("lines at 110 cols is empty")
	}
	hasDivider := false
	for _, l := range lines110 {
		if strings.Contains(l, "│") || strings.Contains(l, "|") {
			hasDivider = true
			break
		}
	}
	if !hasDivider {
		t.Error("width 110 should render side-by-side divider")
	}

	// 2. Width 80: compact list, drilldown toggle on Enter
	lines80 := app.fleet.Lines(80)
	if len(lines80) == 0 {
		t.Fatal("lines at 80 cols is empty")
	}
	// Press Enter to open drill-down
	app = fleetKey(t, app, "enter")
	drilldown80 := strings.Join(app.fleet.Lines(80), "\n")
	if !strings.Contains(drilldown80, "Press Esc or Enter to return") {
		t.Errorf("width 80 detail drilldown missing return prompt:\n%s", drilldown80)
	}
	// Press Enter again to close drill-down
	app = fleetKey(t, app, "enter")

	// 3. Width 60: single pane
	lines60 := app.fleet.Lines(60)
	if len(lines60) == 0 {
		t.Fatal("lines at 60 cols is empty")
	}

	// 4. Width < 60 or Height < 8: resize guidance
	app.fleet.SetSize(50, 20)
	smallW := strings.Join(app.fleet.Lines(50), "\n")
	if !strings.Contains(smallW, "Terminal too small") {
		t.Fatalf("width 50 missing resize guidance:\n%s", smallW)
	}

	app.fleet.SetSize(80, 6)
	smallH := strings.Join(app.fleet.Lines(80), "\n")
	if !strings.Contains(smallH, "Terminal too small") {
		t.Fatalf("height 6 missing resize guidance:\n%s", smallH)
	}

	// 5. Heights 8, 10, 16 must not panic and must produce output
	for _, h := range []int{8, 10, 16} {
		app.fleet.SetSize(80, h)
		hLines := app.fleet.Lines(80)
		if len(hLines) == 0 {
			t.Errorf("height %d produced empty output", h)
		}
	}
}

func TestFleetContextContinuity(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	app := NewAppWithScreens(cfg, testFleetCatalog(), AppOptions{}, AppScreenOptions{FleetReader: reader})

	app = fleetKey(t, app, "2")
	app = fleetKey(t, app, "f")
	app = fleetKey(t, app, "p") // Profiles

	// Move cursor to instance index 1 (sdd-orchestrator)
	app = fleetKey(t, app, "j")
	inst, ok := app.fleet.SelectedInstance()
	if !ok || inst.InstanceID != "sdd-orchestrator" {
		t.Fatalf("selected = %v, want sdd-orchestrator", inst.InstanceID)
	}

	// Set filter to "orchestrator"
	app.fleet.filter = "orchestrator"

	// Switch view to Overview (key 'o')
	app = fleetKey(t, app, "o")
	if app.fleet.ActiveView() != FleetViewOverview {
		t.Fatalf("view = %v, want Overview", app.fleet.ActiveView())
	}

	// Switch route to Explore (key '3')
	app = fleetKey(t, app, "3")
	if app.ActiveRoute() != RouteExplore {
		t.Fatalf("route = %v, want RouteExplore", app.ActiveRoute())
	}

	// Switch back to RouteInstall (Fleet is retained)
	app = fleetKey(t, app, "2")
	if app.installSection != installSectionFleet {
		t.Fatalf("section = %v, want installSectionFleet", app.installSection)
	}

	// Switch back to Profiles (key 'p')
	app = fleetKey(t, app, "p")
	if app.fleet.ActiveView() != FleetViewProfiles {
		t.Fatalf("view = %v, want Profiles", app.fleet.ActiveView())
	}

	// Assert filter and selection survived
	if app.fleet.filter != "orchestrator" {
		t.Fatalf("filter was lost, got %q, want 'orchestrator'", app.fleet.filter)
	}
	instAfter, ok := app.fleet.SelectedInstance()
	if !ok || instAfter.InstanceID != "sdd-orchestrator" {
		t.Fatalf("selection was lost, got %v, want sdd-orchestrator", instAfter.InstanceID)
	}
}

func TestFleetStaleGenerationDiscarded(t *testing.T) {
	cfg := TUIConfig{Width: 100, Height: 30, Color: false, FleetConfig: "/good/fleet.json"}
	reader := &trackingFleetReader{customRes: testFleetStatusView()}
	m := NewFleetModel(cfg, reader)
	m, _ = m.Reload() // m.generation is now 2

	currentStatus := m.Status()

	// 1. Send message with older generation (gen 1)
	staleMsg := FleetStatusMsg{
		Generation: 1,
		Source:     "/good/fleet.json",
		Status: FleetStatusView{
			TotalInstances: 999,
		},
	}
	m, _ = m.Update(staleMsg)
	if m.Status().TotalInstances == 999 {
		t.Fatal("stale generation 1 was accepted; should be discarded")
	}

	// 2. Send message with mismatched source path
	wrongPathMsg := FleetStatusMsg{
		Generation: m.Generation(),
		Source:     "/wrong/fleet.json",
		Status: FleetStatusView{
			TotalInstances: 888,
		},
	}
	m, _ = m.Update(wrongPathMsg)
	if m.Status().TotalInstances == 888 {
		t.Fatal("mismatched source path message was accepted; should be discarded")
	}

	// Verify status remains intact
	if m.Status().TotalInstances != currentStatus.TotalInstances {
		t.Fatalf("status was modified, want %d, got %d", currentStatus.TotalInstances, m.Status().TotalInstances)
	}
}
