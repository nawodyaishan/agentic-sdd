package tui_test

import (
	"io"
	"strings"
	"testing"

	"charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

func makeTestCatalog() tui.Catalog {
	return tui.Catalog{
		Home:       "/test/home",
		Repo:       "/test/repo",
		BackupRoot: "/test/home/.agentic-sdd/backups",
		Items: []tui.CatalogItem{
			{
				ID:       "agentic-sdd-router",
				Name:     "SDD Router",
				Kind:     skillsync.InventoryKindSkill,
				Origin:   skillsync.InventoryOriginCore,
				Group:    "Core skills",
				Category: "core-workflow",
				Purpose:  "Route a request through combined spec, plan, tasks approval.",
				Clients: []tui.ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/test/home/.claude/skills/router"},
				},
			},
			{
				ID:             "agentic-sdd-implement-1",
				Name:           "Implementation Agent",
				Kind:           skillsync.InventoryKindProfile,
				Origin:         skillsync.InventoryOriginPack,
				Group:          "Agent profiles",
				Category:       "core-workflow",
				Purpose:        "Implement an approved batch with focused verification.",
				AgentMode:      "primary",
				AgentProvider:  "9router",
				AgentModel:     "gemini-3.8-flash-high",
				AgentFallbacks: []string{"gemini-3.8-flash-medium", "gemini-4-argon-medium"},
				RoutingNote:    "Manual failover ladder.",
				Clients: []tui.ClientStatus{
					{Client: "opencode", Status: skillsync.InventoryStatusMissing, Path: "/test/home/.opencode/agents/implement"},
				},
			},
		},
	}
}

// 1. Breadcrumb Derivation
func TestT18BreadcrumbDerivation(t *testing.T) {
	cat := makeTestCatalog()
	cfg := tui.TUIConfig{Width: 100, Height: 24, Color: true}
	app := tui.NewApp(cfg, cat)

	// Home
	if bc := app.Breadcrumb(); bc != "Home" {
		t.Fatalf("Home Breadcrumb = %q, want %q", bc, "Home")
	}

	// Install hub
	next, _ := app.Update(tuiRune("2"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Install & Manage" {
		t.Fatalf("Install hub Breadcrumb = %q, want %q", bc, "Install & Manage")
	}

	// Open Install journey
	next, _ = app.Update(tuiRune("i"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Install & Manage > Install > Clients" {
		t.Fatalf("Install clients Breadcrumb = %q, want %q", bc, "Install & Manage > Install > Clients")
	}

	// Advance to options
	next, _ = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Install & Manage > Install > Options" {
		t.Fatalf("Install options Breadcrumb = %q, want %q", bc, "Install & Manage > Install > Options")
	}

	// Advance to review
	next, _ = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Install & Manage > Install > Review" {
		t.Fatalf("Install review Breadcrumb = %q, want %q", bc, "Install & Manage > Install > Review")
	}

	// Back to options -> clients -> result -> hub
	next, _ = app.Update(tuiCode(tea.KeyEsc)) // Review -> Options
	app = next.(tui.App)
	next, _ = app.Update(tuiCode(tea.KeyEsc)) // Options -> Clients
	app = next.(tui.App)
	next, _ = app.Update(tuiCode(tea.KeyEsc)) // Clients -> Cancelled/Result
	app = next.(tui.App)
	next, _ = app.Update(tuiCode(tea.KeyEsc)) // Result -> Hub
	app = next.(tui.App)

	// Open Backups journey
	next, _ = app.Update(tuiRune("b"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Install & Manage > Backups" {
		t.Fatalf("Backups list Breadcrumb = %q, want %q", bc, "Install & Manage > Backups")
	}

	// Explore
	next, _ = app.Update(tuiRune("3"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Explore" {
		t.Fatalf("Explore Breadcrumb = %q, want %q", bc, "Explore")
	}

	// Activity & Help
	next, _ = app.Update(tuiRune("4"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Activity & Help" {
		t.Fatalf("Activity Breadcrumb = %q, want %q", bc, "Activity & Help")
	}

	// Help overlay
	next, _ = app.Update(tuiRune("?"))
	app = next.(tui.App)
	if bc := app.Breadcrumb(); bc != "Activity & Help > Help" {
		t.Fatalf("Help overlay Breadcrumb = %q, want %q", bc, "Activity & Help > Help")
	}
}

// 2. Factual Status Cluster and Session Preferences
func TestT18StatusClusterFactual(t *testing.T) {
	cat := makeTestCatalog()

	// Verify enabled, checking, disabled, motion, ascii states
	cfg := tui.TUIConfig{Width: 100, Height: 24, Color: true, Motion: false, ASCII: false}
	app := tui.NewApp(cfg, cat)

	cluster := app.StatusCluster()
	if strings.Contains(strings.ToLower(cluster), "online") {
		t.Fatalf("StatusCluster must NEVER claim 'online': %q", cluster)
	}

	prefs := app.SessionPreferences()
	if !strings.Contains(prefs, "motion:off") {
		t.Fatalf("SessionPreferences missing motion:off: %q", prefs)
	}
	if !strings.Contains(prefs, "ascii:off") {
		t.Fatalf("SessionPreferences missing ascii:off: %q", prefs)
	}

	// Toggle motion
	next, _ := app.Update(tuiRune("m"))
	app = next.(tui.App)
	prefs = app.SessionPreferences()
	if !strings.Contains(prefs, "motion:on") {
		t.Fatalf("SessionPreferences missing motion:on after toggle: %q", prefs)
	}

	// Toggle ascii
	next, _ = app.Update(tuiRune("u"))
	app = next.(tui.App)
	prefs = app.SessionPreferences()
	if !strings.Contains(prefs, "ascii:on") {
		t.Fatalf("SessionPreferences missing ascii:on after toggle: %q", prefs)
	}

	// Disable update checks
	next, _ = app.Update(tuiRune("1")) // Return to Home
	app = next.(tui.App)
	next, _ = app.Update(tuiRune("d")) // Toggle disabled
	app = next.(tui.App)
	prefs = app.SessionPreferences()
	if !strings.Contains(prefs, "upd:disabled") {
		t.Fatalf("SessionPreferences missing upd:disabled after toggle: %q", prefs)
	}
	if strings.Contains(strings.ToLower(prefs), "online") || strings.Contains(strings.ToLower(prefs), "connected") {
		t.Fatalf("SessionPreferences falsely inferred connectivity: %q", prefs)
	}
}

// 3. Stage-Accurate Footer Key Hints
func TestT18StageAccurateFooter(t *testing.T) {
	cat := makeTestCatalog()
	fakePreview := func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		return appops.PreviewResult{
			PreviewToken: "token-abc-123",
			Planned: []skillsync.ResultChange{
				{Path: "/test/home/.claude/skills/router", Action: skillsync.ChangeInstall},
			},
		}, nil
	}

	app := tui.NewAppWithScreens(
		tui.TUIConfig{Width: 100, Height: 24, Color: true, Home: "/test/home", Repo: "/test/repo"},
		cat,
		tui.AppOptions{},
		tui.AppScreenOptions{InstallPreview: fakePreview},
	)

	// In Home: no 'y confirm'
	view := app.View().Content
	lastLine := getLastLine(view)
	if strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Home footer falsely advertises 'y confirm': %q", lastLine)
	}

	// Go to Install journey
	next, _ := app.Update(tuiRune("2"))
	app = next.(tui.App)
	next, _ = app.Update(tuiRune("i"))
	app = next.(tui.App)

	// Step 1: Clients - y confirm must NOT be present
	lastLine = getLastLine(app.View().Content)
	if strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Install Clients footer falsely advertises 'y confirm': %q", lastLine)
	}
	if !strings.Contains(lastLine, "Space select") && !strings.Contains(lastLine, "Space toggle") {
		t.Fatalf("Install Clients footer missing Space select hint: %q", lastLine)
	}

	// Step 2: Options - y confirm must NOT be present
	next, _ = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	lastLine = getLastLine(app.View().Content)
	if strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Install Options footer falsely advertises 'y confirm': %q", lastLine)
	}

	// Step 3: Review - y confirm must NOT be present
	next, _ = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	lastLine = getLastLine(app.View().Content)
	if strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Install Review footer falsely advertises 'y confirm': %q", lastLine)
	}

	// Step 4: Preview - advance to preview and execute fake command
	var cmd tea.Cmd
	next, cmd = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	if cmd != nil {
		previewMsg := cmd()
		next, _ = app.Update(previewMsg)
		app = next.(tui.App)
	}
	// Step 4: Preview - verify preview footer does not have y confirm
	lastLine = getLastLine(app.View().Content)
	if strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Install Preview footer falsely advertises 'y confirm': %q", lastLine)
	}

	// Step 5: Confirm - advance to confirm where 'y confirm' MUST be present!
	next, _ = app.Update(tuiCode(tea.KeyEnter))
	app = next.(tui.App)
	lastLine = getLastLine(app.View().Content)
	if !strings.Contains(lastLine, "y confirm") {
		t.Fatalf("Install Confirm footer missing 'y confirm': %q", lastLine)
	}
}

// 4. Stable 1-Cell Cursor Gutter and Column Geometry
func TestT18StableMarkerAndGutterGeometry(t *testing.T) {
	glyphsUni := tui.GlyphSet(false)
	glyphsASCII := tui.GlyphSet(true)

	if ansi.StringWidth(glyphsUni.Cursor) != 1 {
		t.Fatalf("Unicode cursor %q width = %d, want 1", glyphsUni.Cursor, ansi.StringWidth(glyphsUni.Cursor))
	}
	if ansi.StringWidth(glyphsUni.Gutter) != 1 {
		t.Fatalf("Unicode gutter %q width = %d, want 1", glyphsUni.Gutter, ansi.StringWidth(glyphsUni.Gutter))
	}
	if ansi.StringWidth(glyphsASCII.Cursor) != 1 {
		t.Fatalf("ASCII cursor %q width = %d, want 1", glyphsASCII.Cursor, ansi.StringWidth(glyphsASCII.Cursor))
	}
	if ansi.StringWidth(glyphsASCII.Gutter) != 1 {
		t.Fatalf("ASCII gutter %q width = %d, want 1", glyphsASCII.Gutter, ansi.StringWidth(glyphsASCII.Gutter))
	}

	// Verify sidebar navigation row widths are stable between selected and unselected
	cat := makeTestCatalog()
	app := tui.NewApp(tui.TUIConfig{Width: 120, Height: 24, Color: true}, cat)

	view0 := app.View().Content
	next, _ := app.Update(tuiRune("j"))
	app = next.(tui.App)
	view1 := app.View().Content

	// Ensure view widths and row count are identical
	lines0 := strings.Split(view0, "\n")
	lines1 := strings.Split(view1, "\n")
	if len(lines0) != len(lines1) {
		t.Fatalf("Line count shifted on cursor movement: %d vs %d", len(lines0), len(lines1))
	}
	for i := range lines0 {
		w0 := ansi.StringWidth(lines0[i])
		w1 := ansi.StringWidth(lines1[i])
		if w0 != w1 {
			t.Fatalf("Row %d width shifted on cursor movement: %d vs %d", i, w0, w1)
		}
	}
}

// 5. Bounds and Responsive Breakpoints
func TestT18BoundsAtAllBreakpoints(t *testing.T) {
	cat := makeTestCatalog()
	widths := []int{40, 60, 79, 80, 100, 109, 110, 130}
	heights := []int{8, 12, 14, 16, 24}
	routes := []string{"1", "2", "3", "4"}

	for _, w := range widths {
		for _, h := range heights {
			for _, r := range routes {
				for _, color := range []bool{true, false} {
					for _, ascii := range []bool{true, false} {
						cfg := tui.TUIConfig{Width: w, Height: h, Color: color, ASCII: ascii}
						app := tui.NewApp(cfg, cat)
						next, _ := app.Update(tuiRune(r))
						app = next.(tui.App)

						content := app.View().Content
						lines := strings.Split(content, "\n")
						if len(lines) != h {
							t.Fatalf("w=%d h=%d r=%s c=%t a=%t lines=%d, want %d",
								w, h, r, color, ascii, len(lines), h)
						}
						for rowIdx, line := range lines {
							lineWidth := ansi.StringWidth(line)
							if lineWidth > w {
								t.Fatalf("w=%d h=%d r=%s row=%d width=%d exceeds max %d:\n%q",
									w, h, r, rowIdx, lineWidth, w, line)
							}
						}
						footer := ansi.Strip(lines[len(lines)-1])
						if !strings.Contains(strings.ToLower(footer), "quit") && !strings.Contains(footer, "q") {
							t.Fatalf("w=%d h=%d r=%s footer missing quit hint: %q", w, h, r, footer)
						}
					}
				}
			}
		}
	}
}

// 6. Safe Glyphs and No Route Emoji
func TestT18SafeGlyphsAndNoEmoji(t *testing.T) {
	cat := makeTestCatalog()
	// Test both Unicode and ASCII mode
	for _, ascii := range []bool{false, true} {
		app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 24, Color: true, ASCII: ascii}, cat)
		view := app.View().Content
		for _, emoji := range []string{"🏠", "📦", "🔍", "💡", "🚀"} {
			if strings.Contains(view, emoji) {
				t.Fatalf("View contains decorative emoji %s (ascii=%t):\n%s", emoji, ascii, view)
			}
		}
	}
}

// 7. Long Wide and Combining Data Preservation
func TestT18LongWideAndCombiningData(t *testing.T) {
	cat := tui.Catalog{
		Home:       "/home/用户/résumé",
		BackupRoot: "/home/用户/backups",
		Items: []tui.CatalogItem{
			{
				ID:       "unicode-test-item",
				Name:     "日本語・Skill",
				Kind:     skillsync.InventoryKindSkill,
				Origin:   skillsync.InventoryOriginCore,
				Group:    "Core skills",
				Category: "core-workflow",
				Purpose:  "Testing wide CJK characters and combining diacritics e\u0301.",
				Clients: []tui.ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/home/用户/résumé/skill-path"},
				},
			},
		},
	}

	app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 24, Color: true, ASCII: false}, cat)
	// Go to Explore
	next, _ := app.Update(tuiRune("3"))
	app = next.(tui.App)

	view := app.View().Content
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if w := ansi.StringWidth(line); w > 100 {
			t.Fatalf("Line %d width %d exceeds 100 cells with wide/combining characters:\n%q", i, w, line)
		}
	}
}

// 8. Route Tab Pills
func TestT18RouteTabPills(t *testing.T) {
	cat := makeTestCatalog()
	// Compact layout (width 90, height 24) uses compactNavigation
	app := tui.NewApp(tui.TUIConfig{Width: 90, Height: 24, Color: true}, cat)
	view := ansi.Strip(app.View().Content)

	// Active tab pill for Home
	if !strings.Contains(view, "[1 Home]") {
		t.Fatalf("View missing active tab pill [1 Home]:\n%s", view)
	}
	// Inactive destinations must be plain
	if strings.Contains(view, "[2 Install]") {
		t.Fatalf("Inactive route 2 Install unexpectedly has pill brackets:\n%s", view)
	}

	// Switch to route 2
	next, _ := app.Update(tuiRune("2"))
	app = next.(tui.App)
	view = ansi.Strip(app.View().Content)
	if !strings.Contains(view, "[2 Install]") {
		t.Fatalf("View missing active tab pill [2 Install]:\n%s", view)
	}
	if strings.Contains(view, "[1 Home]") {
		t.Fatalf("Inactive route 1 Home unexpectedly has pill brackets:\n%s", view)
	}
}

// Helper to extract the last line of a view
func getLastLine(view string) string {
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	return ansi.Strip(lines[len(lines)-1])
}

func tuiRune(r string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(r)[0], Text: r}
}

func tuiCode(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}
