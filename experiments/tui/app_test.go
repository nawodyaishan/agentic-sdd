package tui_test

import (
	"strings"
	"testing"
	"time"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestAppInitialState(t *testing.T) {
	cfg := tui.TUIConfig{Width: 120, Height: 30, Interactive: true}
	cat := testCatalog()
	app := tui.NewApp(cfg, cat)

	if app.ActiveRoute() != tui.RouteHome {
		t.Fatalf("ActiveRoute() = %v, want RouteHome", app.ActiveRoute())
	}
	if app.Focus() != tui.FocusNav {
		t.Fatalf("Focus() = %v, want FocusNav", app.Focus())
	}

	routes := app.Routes()
	if len(routes) != 4 {
		t.Fatalf("len(Routes()) = %d, want 4 destinations", len(routes))
	}
	expectedRoutes := []string{
		"Home",
		"Install & Manage",
		"Explore",
		"Activity & Help",
	}
	for i, want := range expectedRoutes {
		if routes[i].Title != want {
			t.Errorf("routes[%d].Title = %q, want %q", i, routes[i].Title, want)
		}
	}
}

func TestAppRouteNavigation(t *testing.T) {
	cfg := tui.TUIConfig{Width: 120, Height: 30, Interactive: true}
	app := tui.NewApp(cfg, testCatalog())

	// Down arrow moves nav index to Install & Manage
	next, _ := app.Update(key(tea.KeyDown))
	app = next.(tui.App)
	if app.NavIndex() != 1 {
		t.Fatalf("after Down NavIndex = %d, want 1", app.NavIndex())
	}

	// Enter activates the route
	next, _ = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteInstall {
		t.Fatalf("after Enter ActiveRoute = %v, want RouteInstall", app.ActiveRoute())
	}

	// Tab returns focus to Nav to continue navigating routes
	next, _ = app.Update(key(tea.KeyTab))
	app = next.(tui.App)

	// j moves to Explore (index 2)
	next, _ = app.Update(runes("j"))
	app = next.(tui.App)
	next, _ = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteExplore {
		t.Fatalf("after j + Enter ActiveRoute = %v, want RouteExplore", app.ActiveRoute())
	}

	// Number keys jump directly
	next, _ = app.Update(runes("1"))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteHome {
		t.Fatalf("after 1 ActiveRoute = %v, want RouteHome", app.ActiveRoute())
	}

	next, _ = app.Update(runes("4"))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteActivity {
		t.Fatalf("after 4 ActiveRoute = %v, want RouteActivity", app.ActiveRoute())
	}
}

func TestAppTabFocusCycling(t *testing.T) {
	cfg := tui.TUIConfig{Width: 120, Height: 30, Interactive: true}
	app := tui.NewApp(cfg, testCatalog())

	if app.Focus() != tui.FocusNav {
		t.Fatalf("initial Focus = %v, want FocusNav", app.Focus())
	}

	next, _ := app.Update(key(tea.KeyTab))
	app = next.(tui.App)
	if app.Focus() != tui.FocusContent {
		t.Fatalf("after Tab Focus = %v, want FocusContent", app.Focus())
	}

	next, _ = app.Update(key(tea.KeyTab))
	app = next.(tui.App)
	if app.Focus() != tui.FocusNav {
		t.Fatalf("after second Tab Focus = %v, want FocusNav", app.Focus())
	}
}

func TestAppExploreSelectionPreservedAcrossRoutes(t *testing.T) {
	cfg := tui.TUIConfig{Width: 120, Height: 30, Interactive: true}
	app := tui.NewApp(cfg, testCatalog())

	// Navigate to Explore (activates route and focuses content)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteExplore {
		t.Fatalf("ActiveRoute = %v, want RouteExplore", app.ActiveRoute())
	}
	if app.Focus() != tui.FocusContent {
		t.Fatalf("after route activation Focus = %v, want FocusContent", app.Focus())
	}

	// Move cursor down in catalog list (to beta-skill)
	next, _ = app.Update(runes("j"))
	app = next.(tui.App)
	selected, ok := app.CatalogSelected()
	if !ok || selected.ID != "beta-skill" {
		t.Fatalf("CatalogSelected = %q (ok=%v), want beta-skill", selected.ID, ok)
	}

	// Switch to Home
	next, _ = app.Update(key(tea.KeyTab)) // back to nav
	app = next.(tui.App)
	next, _ = app.Update(runes("1"))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteHome {
		t.Fatalf("ActiveRoute = %v, want RouteHome", app.ActiveRoute())
	}

	// Return to Explore
	next, _ = app.Update(runes("3"))
	app = next.(tui.App)
	selected, ok = app.CatalogSelected()
	if !ok || selected.ID != "beta-skill" {
		t.Fatalf("after returning to Explore, CatalogSelected = %q, want beta-skill preserved", selected.ID)
	}
}

func TestAppFilterPrecedence(t *testing.T) {
	cfg := tui.TUIConfig{Width: 120, Height: 30, Interactive: true}
	app := tui.NewApp(cfg, testCatalog())

	// Go to Explore (route activation focuses Content)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)

	// Press / to start filtering
	next, _ = app.Update(runes("/"))
	app = next.(tui.App)

	if !app.IsFiltering() {
		t.Fatal("after / IsFiltering() = false, want true")
	}

	// While filtering, pressing 'q' must not quit the app; it must be filter text
	next, cmd := app.Update(runes("q"))
	app = next.(tui.App)
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("pressing q while filtering caused a quit command; want filter text input")
		}
	}

	// Pressing 'm' and 'u' must also be filter text, not toggle motion or symbols
	initialMotion := app.Motion()
	next, _ = app.Update(runes("m"))
	app = next.(tui.App)
	if app.Motion() != initialMotion {
		t.Fatal("pressing m while filtering changed Motion; want filter text input")
	}
}

func TestAppSingleContextualFooter(t *testing.T) {
	cfg := tui.TUIConfig{Width: 80, Height: 24, Color: false}
	app := tui.NewApp(cfg, testCatalog())

	view := app.View().Content
	if !strings.Contains(view, "tab focus") && !strings.Contains(view, "Tab") && !strings.Contains(view, "tab") {
		t.Fatalf("footer missing Tab hint:\n%s", view)
	}
	if !strings.Contains(view, "quit") && !strings.Contains(view, "q") {
		t.Fatalf("footer missing quit hint:\n%s", view)
	}
	// Verify Bubbles default help bar is disabled (no duplicate footer)
	if strings.Contains(view, "↑/k up • ↓/j down • / filter") {
		t.Fatalf("found duplicate default child help in view:\n%s", view)
	}
}

func TestAppResponsiveLayoutBreakpoints(t *testing.T) {
	cat := testCatalog()

	// 1. Wide layout (>= 110)
	appWide := tui.NewApp(tui.TUIConfig{Width: 120, Height: 30}, cat)
	next, _ := appWide.Update(runes("3")) // Explore
	appWide = next.(tui.App)
	viewWide := appWide.View().Content
	if !strings.Contains(viewWide, "alpha-skill") {
		t.Fatalf("wide view missing catalog list items:\n%s", viewWide)
	}
	if !strings.Contains(viewWide, "Destination status") {
		t.Fatalf("wide view (>=110) must show side-by-side details pane:\n%s", viewWide)
	}

	// 2. Medium layout (80-109)
	appMed := tui.NewApp(tui.TUIConfig{Width: 90, Height: 26}, cat)
	next, _ = appMed.Update(runes("3"))
	appMed = next.(tui.App)
	viewMed := appMed.View().Content
	if !strings.Contains(viewMed, "alpha-skill") {
		t.Fatalf("medium view missing catalog items:\n%s", viewMed)
	}
	if !strings.Contains(viewMed, "Destination status") {
		t.Fatalf("medium view (80-109) must show details pane:\n%s", viewMed)
	}

	// 3. Narrow layout (60-79)
	appNarrow := tui.NewApp(tui.TUIConfig{Width: 70, Height: 24}, cat)
	next, _ = appNarrow.Update(runes("3"))
	appNarrow = next.(tui.App)
	viewNarrow := appNarrow.View().Content
	if !strings.Contains(viewNarrow, "alpha-skill") {
		t.Fatalf("narrow view missing catalog items:\n%s", viewNarrow)
	}
	// In 60-79, list is single pane by default; details only shown when opened
	if strings.Contains(viewNarrow, "Destination status") {
		t.Fatalf("narrow view (60-79) should not show side-by-side details by default:\n%s", viewNarrow)
	}

	// 4. Safe minimal layout (< 60)
	appSmall := tui.NewApp(tui.TUIConfig{Width: 50, Height: 20}, cat)
	viewSmall := appSmall.View().Content
	if len(viewSmall) == 0 {
		t.Fatal("small view (< 60) returned empty content")
	}

	// 5. Viewport too small (< 40x8)
	appTooSmall := tui.NewApp(tui.TUIConfig{Width: 30, Height: 6}, cat)
	viewTooSmall := appTooSmall.View().Content
	if !strings.Contains(viewTooSmall, "Terminal too small") && !strings.Contains(viewTooSmall, "resize") {
		t.Fatalf("too small view missing resize guidance:\n%s", viewTooSmall)
	}
}

func TestAppBrandTickIntegration(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: true}
	app := tui.NewApp(cfg, testCatalog())

	// Initial brand state on Home with motion=true is not done
	if app.BrandDone() {
		t.Fatal("initial brand state on interactive Home with motion=true should not be done")
	}

	// Init() should return tick command for Home
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("app.Init() returned nil command, want brand tick cmd")
	}

	// Advance through all frames
	for i := 0; i < tui.BrandFrames+2; i++ {
		next, _ := app.Update(tui.BrandTickMsg{})
		app = next.(tui.App)
	}
	if !app.BrandDone() {
		t.Fatal("after BrandFrames ticks, brand state must be Done")
	}

	// Further ticks do not restart or schedule new ticks
	next, nextCmd := app.Update(tui.BrandTickMsg{})
	app = next.(tui.App)
	if nextCmd != nil {
		t.Fatal("after brand Done, tick msg should return nil command")
	}
}

func TestAppSessionControls(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: true}
	app := tui.NewApp(cfg, testCatalog())

	if !app.Motion() {
		t.Fatal("initial Motion = false, want true")
	}

	// Press m to toggle motion
	next, _ := app.Update(runes("m"))
	app = next.(tui.App)
	if app.Motion() {
		t.Fatal("after m Motion = true, want false")
	}
	// Turning motion off settles brand state immediately
	if !app.BrandDone() {
		t.Fatal("disabling motion must settle brand state")
	}

	// Press u to toggle symbol mode
	if app.ASCII() {
		t.Fatal("initial ASCII = true, want false")
	}
	next, _ = app.Update(runes("u"))
	app = next.(tui.App)
	if !app.ASCII() {
		t.Fatal("after u ASCII = false, want true")
	}
}

func TestAppTruthfulB1Placeholders(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 30}
	app := tui.NewApp(cfg, testCatalog())

	// Install & Manage
	next, _ := app.Update(runes("2"))
	app = next.(tui.App)
	viewInstall := app.View().Content
	if !strings.Contains(viewInstall, "Install & Manage") {
		t.Fatalf("Install view missing title:\n%s", viewInstall)
	}
	if !strings.Contains(viewInstall, "preview") && !strings.Contains(viewInstall, "read-only") && !strings.Contains(viewInstall, "B4") {
		t.Fatalf("Install view missing truthful B1 explanation:\n%s", viewInstall)
	}

	// Activity & Help
	next, _ = app.Update(runes("4"))
	app = next.(tui.App)
	viewHelp := app.View().Content
	if !strings.Contains(viewHelp, "Activity & Help") {
		t.Fatalf("Help view missing title:\n%s", viewHelp)
	}
	if !strings.Contains(viewHelp, "https://github.com/nawodyaishan/agentic-sdd#readme") {
		t.Fatalf("Help view missing online guide URL:\n%s", viewHelp)
	}
}

func TestAppHelpOverlay(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 30}
	app := tui.NewApp(cfg, testCatalog())

	if app.HelpOpen() {
		t.Fatal("initial HelpOpen = true, want false")
	}

	// Press ? to open help
	next, _ := app.Update(runes("?"))
	app = next.(tui.App)
	if !app.HelpOpen() {
		t.Fatal("after ? HelpOpen = false, want true")
	}

	view := app.View().Content
	if !strings.Contains(view, "Keyboard & Navigation Reference") {
		t.Fatalf("help view missing title:\n%s", view)
	}

	// Press Esc to close help
	next, _ = app.Update(key(tea.KeyEscape))
	app = next.(tui.App)
	if app.HelpOpen() {
		t.Fatal("after Esc HelpOpen = true, want false")
	}
}

func TestAppHomeEndNavigation(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 30}
	app := tui.NewApp(cfg, testCatalog())

	// In Nav: End goes to index 3 (Activity & Help), Home goes to 0 (Home)
	next, _ := app.Update(key(tea.KeyEnd))
	app = next.(tui.App)
	if app.NavIndex() != 3 {
		t.Fatalf("after End NavIndex = %d, want 3", app.NavIndex())
	}
	next, _ = app.Update(key(tea.KeyHome))
	app = next.(tui.App)
	if app.NavIndex() != 0 {
		t.Fatalf("after Home NavIndex = %d, want 0", app.NavIndex())
	}

	// In Explore Content: End goes to last catalog item, Home to first
	next, _ = app.Update(runes("3")) // Explore focuses Content
	app = next.(tui.App)

	next, _ = app.Update(key(tea.KeyEnd))
	app = next.(tui.App)
	selected, ok := app.CatalogSelected()
	if !ok || selected.ID != "zulu-profile" {
		t.Fatalf("after End CatalogSelected = %q, want zulu-profile (last item)", selected.ID)
	}

	next, _ = app.Update(key(tea.KeyHome))
	app = next.(tui.App)
	selected, ok = app.CatalogSelected()
	if !ok || selected.ID != "alpha-skill" {
		t.Fatalf("after Home CatalogSelected = %q, want alpha-skill (first item)", selected.ID)
	}
}

func TestTerminalExactBounds(t *testing.T) {
	cat := testCatalog()
	widths := []int{40, 60, 79, 80, 109, 110}
	heights := []int{8, 12, 16, 24, 30}
	routes := []string{"1", "2", "3", "4"}

	for _, w := range widths {
		for _, h := range heights {
			for _, r := range routes {
				cfg := tui.TUIConfig{Width: w, Height: h, Color: false}
				app := tui.NewApp(cfg, cat)
				next, _ := app.Update(runes(r))
				app = next.(tui.App)

				view := app.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) > h {
					t.Errorf("w=%d h=%d route=%s produced %d lines > height %d", w, h, r, len(lines), h)
				}
				for lineIdx, line := range lines {
					lineWidth := ansi.StringWidth(line)
					if lineWidth > w {
						t.Errorf("w=%d h=%d route=%s line %d width %d > %d:\n%q", w, h, r, lineIdx, lineWidth, w, line)
					}
				}

				// Also test with help open
				nextHelp, _ := app.Update(runes("?"))
				appHelp := nextHelp.(tui.App)
				viewHelp := appHelp.View().Content
				helpLines := strings.Split(viewHelp, "\n")
				if len(helpLines) > h {
					t.Errorf("w=%d h=%d route=%s (help) produced %d lines > height %d", w, h, r, len(helpLines), h)
				}
				for lineIdx, line := range helpLines {
					lineWidth := ansi.StringWidth(line)
					if lineWidth > w {
						t.Errorf("w=%d h=%d route=%s (help) line %d width %d > %d:\n%q", w, h, r, lineIdx, lineWidth, w, line)
					}
				}
			}

			// In Explore at medium 80-109, details pane MUST be visible side-by-side with list
			if w >= 80 && w <= 109 && h >= 16 {
				cfg := tui.TUIConfig{Width: w, Height: h, Color: false}
				app := tui.NewApp(cfg, cat)
				next, _ := app.Update(runes("3")) // Explore
				app = next.(tui.App)
				view := app.View().Content
				if !strings.Contains(view, "Destination status") {
					t.Errorf("w=%d h=%d medium layout MUST show side-by-side details pane:\n%s", w, h, view)
				}
			}
		}
	}
}

func TestASCIIAll7Bit(t *testing.T) {
	cat := testCatalog()
	routes := []string{"1", "2", "3", "4"}

	for _, r := range routes {
		cfg := tui.TUIConfig{Width: 100, Height: 26, ASCII: true, Color: false}
		app := tui.NewApp(cfg, cat)
		next, _ := app.Update(runes(r))
		app = next.(tui.App)

		// Test normal view
		view := app.View().Content
		for i, rChar := range view {
			if rChar > 127 {
				t.Fatalf("route %s ASCII view contains non-7-bit character %q (U+%04X) at index %d:\n%s", r, rChar, rChar, i, view)
			}
		}

		// Test help overlay
		nextHelp, _ := app.Update(runes("?"))
		appHelp := nextHelp.(tui.App)
		viewHelp := appHelp.View().Content
		for i, rChar := range viewHelp {
			if rChar > 127 {
				t.Fatalf("route %s (help) ASCII view contains non-7-bit character %q (U+%04X) at index %d:\n%s", r, rChar, rChar, i, viewHelp)
			}
		}
	}

	// Verify Unicode mode contains emoji nav icons
	cfgUni := tui.TUIConfig{Width: 100, Height: 26, ASCII: false, Color: false}
	appUni := tui.NewApp(cfgUni, cat)
	viewUni := appUni.View().Content
	if !strings.Contains(viewUni, "🏠") {
		t.Fatalf("Unicode view missing 🏠 emoji nav icon:\n%s", viewUni)
	}
}

func TestRouteSwitchFocusAndHomeSetup(t *testing.T) {
	cfg := tui.TUIConfig{Width: 100, Height: 26}
	app := tui.NewApp(cfg, testCatalog())

	// Initially on Home with FocusNav
	if app.Focus() != tui.FocusNav {
		t.Fatalf("initial Focus = %v, want FocusNav", app.Focus())
	}

	// Activating route (e.g. Enter on Home) moves focus to Content
	next, _ := app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	if app.Focus() != tui.FocusContent {
		t.Fatalf("after Enter activation Focus = %v, want FocusContent", app.Focus())
	}

	// In Home with FocusContent, pressing Enter activates primary action (Install)
	next, _ = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteInstall {
		t.Fatalf("after Enter on Home primary action ActiveRoute = %v, want RouteInstall", app.ActiveRoute())
	}
}

func TestAppBrandTickLifecycle(t *testing.T) {
	cat := testCatalog()

	// Default is static when cfg.Motion is false (even if Interactive is true)
	cfgDefault := tui.TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: false}
	appDefault := tui.NewApp(cfgDefault, cat)
	if !appDefault.BrandDone() {
		t.Fatal("cfg.Motion=false must start brand state as Done")
	}

	// Short height collapses brand and disables ticking
	cfgShort := tui.TUIConfig{Width: 100, Height: 12, Interactive: true, Motion: true}
	appShort := tui.NewApp(cfgShort, cat)
	nextShort, cmdShort := appShort.Update(tui.BrandTickMsg{})
	if cmdShort != nil {
		t.Fatal("brand tick should not schedule next tick when height < 14 (header collapsed)")
	}
	appShort = nextShort.(tui.App)

	// User input skips brand animation
	cfgAnim := tui.TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: true}
	appAnim := tui.NewApp(cfgAnim, cat)
	if appAnim.BrandDone() {
		t.Fatal("animated app should start with BrandDone=false")
	}
	// Any user key skips animation
	nextKey, _ := appAnim.Update(runes("j"))
	appAnim = nextKey.(tui.App)
	if !appAnim.BrandDone() {
		t.Fatal("user key press must immediately settle brand state (skip animation)")
	}
}

func TestInstallBackupRootAndHints(t *testing.T) {
	cat := testCatalog()
	cat.BackupRoot = "/tmp/test-home/.config/agentic-sdd/backups"
	cfg := tui.TUIConfig{Width: 100, Height: 30, Repo: "/tmp/custom-repo"}
	app := tui.NewApp(cfg, cat)

	next, _ := app.Update(runes("2")) // Install
	app = next.(tui.App)
	view := app.View().Content

	if !strings.Contains(view, cat.BackupRoot) {
		t.Fatalf("Install view must show real catalog.BackupRoot %q:\n%s", cat.BackupRoot, view)
	}
	if strings.Contains(view, "agentic-sdd restore preview\n") {
		t.Fatalf("Install view must not give incomplete restore command without backup selection:\n%s", view)
	}
	if !strings.Contains(view, "/tmp/custom-repo") {
		t.Fatalf("Install view should retain repo flag hint when custom repo is configured:\n%s", view)
	}
}

func executeCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()
	select {
	case msg := <-done:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var msgs []tea.Msg
			for _, c := range batch {
				msgs = append(msgs, executeCmd(c)...)
			}
			return msgs
		}
		return []tea.Msg{msg}
	case <-time.After(20 * time.Millisecond):
		// Skip long/blocking timers (like cursor blinks)
		return nil
	}
}

func TestAppMessageRoutingAndFilterExecution(t *testing.T) {
	cat := testCatalog()
	cfg := tui.TUIConfig{Width: 100, Height: 26}
	app := tui.NewApp(cfg, cat)

	dispatch := func(cmd tea.Cmd) {
		for _, m := range executeCmd(cmd) {
			if m != nil {
				next, c := app.Update(m)
				app = next.(tui.App)
				if c != nil {
					for _, follow := range executeCmd(c) {
						if follow != nil {
							next, _ := app.Update(follow)
							app = next.(tui.App)
						}
					}
				}
			}
		}
	}

	// Switch to Explore (focuses Content)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)

	// Press / to enter filter
	next, cmd := app.Update(runes("/"))
	app = next.(tui.App)
	dispatch(cmd)

	// Type "beta"
	for _, r := range "beta" {
		next, cmd = app.Update(runes(string(r)))
		app = next.(tui.App)
		dispatch(cmd)
	}

	// Press Enter to finalize the filter
	next, cmd = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	dispatch(cmd)

	if app.IsFiltering() {
		t.Fatal("after Enter filter must be finalized (IsFiltering=false)")
	}

	// Filter should now match only beta-skill
	visible := app.VisibleCatalogItems()
	if len(visible) != 1 || visible[0].ID != "beta-skill" {
		t.Fatalf("after typing beta and executing filter msgs, VisibleItems = %v, want [beta-skill]", ids(visible))
	}

	// Press End in filtered Explore: should select index 0 (the only visible item)
	next, _ = app.Update(key(tea.KeyEnd))
	app = next.(tui.App)
	selected, ok := app.CatalogSelected()
	if !ok || selected.ID != "beta-skill" {
		t.Fatalf("after End with filtered list, CatalogSelected = %q, want beta-skill", selected.ID)
	}
}

func TestScrollableLongContentAndEndMarkers(t *testing.T) {
	// Create catalog with very long path containing distinct end marker
	cat := tui.Catalog{
		Items: []tui.CatalogItem{
			{
				ID:     "long-path-skill",
				Kind:   "skill",
				Origin: "core",
				Group:  "Core skills",
				Clients: []tui.ClientStatus{
					{
						Client: "claude",
						Status: skillsync.InventoryStatusMatchesSource,
						Path:   "/tmp/very/long/nested/path/to/verify/wrapping/and/scrolling/without/clipping/to/destination/with/many/many/directories/and/levels/down/alpha-skill-TAIL-MARKER-XYZ",
					},
				},
			},
		},
	}

	cfg := tui.TUIConfig{Width: 80, Height: 12, Color: false}
	app := tui.NewApp(cfg, cat)

	// Go to Explore
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)

	viewInitial := app.View().Content
	if strings.Contains(viewInitial, "MARKER-XYZ") {
		t.Fatalf("initial short view should have tail marker below fold:\n%s", viewInitial)
	}

	// Press PageDown to scroll detail pane down
	next, _ = app.Update(key(tea.KeyPgDown))
	app = next.(tui.App)

	viewScrolled := app.View().Content
	if !strings.Contains(viewScrolled, "MARKER-XYZ") {
		t.Fatalf("after PageDown detail pane must reveal end marker:\n%s", viewScrolled)
	}

	// Verify exact bounds: every line <= width, total lines <= height
	lines := strings.Split(viewScrolled, "\n")
	if len(lines) > 12 {
		t.Fatalf("viewScrolled has %d lines > height 12", len(lines))
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w > 80 {
			t.Fatalf("line %d width %d > 80:\n%q", i, w, l)
		}
	}
}

func TestFooterAlwaysVisibleAtShortHeights(t *testing.T) {
	cat := testCatalog()
	routes := []string{"1", "2", "3", "4"}
	heights := []int{8, 12}

	for _, h := range heights {
		for _, r := range routes {
			cfg := tui.TUIConfig{Width: 80, Height: h, Color: false}
			app := tui.NewApp(cfg, cat)
			next, _ := app.Update(runes(r))
			app = next.(tui.App)

			view := app.View().Content
			lines := strings.Split(view, "\n")
			if len(lines) == 0 {
				t.Fatalf("h=%d r=%s produced empty view", h, r)
			}
			lastLine := lines[len(lines)-1]
			if !strings.Contains(lastLine, "quit") && !strings.Contains(lastLine, "q") {
				t.Fatalf("h=%d r=%s footer not visible on bottom row:\n%s", h, r, view)
			}

			// Also verify with help open
			nextHelp, _ := app.Update(runes("?"))
			appHelp := nextHelp.(tui.App)
			viewHelp := appHelp.View().Content
			helpLines := strings.Split(viewHelp, "\n")
			lastHelpLine := helpLines[len(helpLines)-1]
			if !strings.Contains(lastHelpLine, "quit") && !strings.Contains(lastHelpLine, "q") {
				t.Fatalf("h=%d r=%s (help) footer not visible on bottom row:\n%s", h, r, viewHelp)
			}
		}
	}
}

func TestUnicodePathASCIIReversibleEscaping(t *testing.T) {
	unicodeMarker := "/tmp/home/résumé-🚀-файл/END-OF-PATH"
	cat := tui.Catalog{
		Items: []tui.CatalogItem{
			{
				ID:     "unicode-skill",
				Kind:   "skill",
				Origin: "core",
				Group:  "Core skills",
				Clients: []tui.ClientStatus{
					{
						Client: "claude",
						Status: skillsync.InventoryStatusMatchesSource,
						Path:   unicodeMarker,
					},
				},
			},
		},
	}

	cfg := tui.TUIConfig{Width: 90, Height: 24, ASCII: true, Color: false}
	app := tui.NewApp(cfg, cat)
	next, _ := app.Update(runes("3")) // Explore
	app = next.(tui.App)

	view := app.View().Content

	// Every character in view must be 7-bit ASCII
	for i, rChar := range view {
		if rChar > 127 {
			t.Fatalf("view contains non-ASCII character %q (U+%04X) at index %d:\n%s", rChar, rChar, i, view)
		}
	}

	// Must contain \u00e9 for 'é' (not replaced by blank space or deleted)
	if !strings.Contains(view, `\u00e9`) && !strings.Contains(view, `\u00E9`) {
		t.Fatalf("view must preserve 'é' as \\u00e9, got:\n%s", view)
	}

	// Must contain \U0001f680 for rocket emoji
	if !strings.Contains(view, `\U0001f680`) && !strings.Contains(view, `\U0001F680`) && !strings.Contains(view, `\u1f680`) {
		t.Fatalf("view must preserve rocket emoji as \\U0001f680, got:\n%s", view)
	}

	// Must contain END-OF-PATH marker
	if !strings.Contains(view, "END-OF-PATH") {
		t.Fatalf("view must preserve END-OF-PATH marker, got:\n%s", view)
	}
}

func TestBrandTickOffscreenSuppression(t *testing.T) {
	cat := testCatalog()
	cfg := tui.TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: true}
	app := tui.NewApp(cfg, cat)

	// Switch to Route 2 (Install & Manage)
	next, _ := app.Update(runes("2"))
	app = next.(tui.App)
	if app.ActiveRoute() != tui.RouteInstall {
		t.Fatalf("ActiveRoute = %v, want RouteInstall", app.ActiveRoute())
	}

	// Send BrandTickMsg while on Install route: should NOT schedule next tick
	next, cmd := app.Update(tui.BrandTickMsg{})
	if cmd != nil {
		t.Fatal("BrandTickMsg on non-Home route must return nil command (no offscreen animation)")
	}
	app = next.(tui.App)

	// Return to Home: ticking should resume
	next, cmd = app.Update(runes("1"))
	app = next.(tui.App)
	if cmd == nil && !app.BrandDone() {
		t.Fatal("returning to Home with unfinished motion must resume brand tick")
	}
}
