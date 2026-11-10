package tui

import (
	"image/color"
	"os"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// Daylight Lagoon light palette constants.
// Every text role sits on #F8FAFA (canvas), #EDF3F4 (surface), or #E0E9EB (selectedBg)
// and clears the proposed AAA >= 7:1 floor, while borders and indicators clear >= 3:1.
const (
	wantPaletteLightBackground  = "#F8FAFA"
	wantPaletteLightSurface     = "#EDF3F4"
	wantPaletteLightForeground  = "#0A1F26"
	wantPaletteLightMuted       = "#233A3F"
	wantPaletteLightDim         = "#30484D"
	wantPaletteLightPrimary     = "#661772"
	wantPaletteLightSecondary   = "#004F59"
	wantPaletteLightSuccess     = "#0B5322"
	wantPaletteLightWarning     = "#643E00"
	wantPaletteLightError       = "#7E1414"
	wantPaletteLightBorder      = "#688487"
	wantPaletteLightBorderFocus = "#661772"
	wantPaletteLightSelectedBg  = "#E0E9EB"
)

// TestThemeAgenticDarkActualPairsMeetWCAG tests the Agentic Dark (Catppuccin Mocha) palette
// against the WCAG AAA (7:1 text, 3:1 non-text) floor across canvas, surface,
// and selected backgrounds for every actual pair.
func TestThemeAgenticDarkActualPairsMeetWCAG(t *testing.T) {
	theme := NewTheme(true)
	if !theme.Color {
		t.Fatal("NewTheme(true) must enable color")
	}
	if !theme.IsDark {
		t.Fatal("NewTheme(true) must have IsDark=true")
	}

	canvas := contrastRGB(t, theme.CanvasColor())
	surface := contrastRGB(t, theme.SurfaceColor())
	selected := contrastRGB(t, theme.SelectedBgColor())

	if got := contrastHex(t, theme.CanvasColor()); got != wantPaletteBackground {
		t.Fatalf("canvas = %s, want %s", got, wantPaletteBackground)
	}
	if got := contrastHex(t, theme.SurfaceColor()); got != wantPaletteSurface {
		t.Fatalf("surface = %s, want %s", got, wantPaletteSurface)
	}
	if got := contrastHex(t, theme.SelectedBgColor()); got != wantPaletteSelectedBg {
		t.Fatalf("selectedBg = %s, want %s", got, wantPaletteSelectedBg)
	}

	// Text roles on Canvas and Rail/Header surface must clear >= 7:1 AAA.
	textRoles := []struct {
		name string
		fg   color.Color
	}{
		{"Text", theme.Text.GetForeground()},
		{"Title", theme.Title.GetForeground()},
		{"Subtitle", theme.Subtitle.GetForeground()},
		{"Section", theme.Section.GetForeground()},
		{"Accent", theme.Accent.GetForeground()},
		{"OK", theme.OK.GetForeground()},
		{"Warn", theme.Warn.GetForeground()},
		{"Bad", theme.Bad.GetForeground()},
		{"Hint", theme.Hint.GetForeground()},
		{"NavActive", theme.NavActive.GetForeground()},
		{"NavInactive", theme.NavInactive.GetForeground()},
		{"DefaultForeground", theme.DefaultForeground()},
		{"Dim", theme.Dim.GetForeground()},
		{"Info", theme.Info.GetForeground()},
		{"Notice", theme.Notice.GetForeground()},
	}

	for _, role := range textRoles {
		fg := contrastRGB(t, role.fg)
		for _, bg := range []struct {
			bgName string
			bgCol  color.RGBA
		}{
			{"canvas", canvas},
			{"surface", surface},
		} {
			ratio := contrastRatio(fg, bg.bgCol)
			if ratio+1e-9 < 7.0 {
				t.Errorf("Role %s on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
					role.name, bg.bgName, ratio, contrastHex(t, role.fg), contrastHex(t, bg.bgCol))
			}
		}
	}

	// Selected row actual text pairs: Text and Muted must clear >= 7:1 AAA on SelectedBg.
	selectedTextRoles := []struct {
		name string
		fg   color.Color
	}{
		{"SelectedText", theme.Selected.GetForeground()},
		{"SelectedMuted", theme.Muted.GetForeground()},
	}
	for _, role := range selectedTextRoles {
		fg := contrastRGB(t, role.fg)
		ratio := contrastRatio(fg, selected)
		if ratio+1e-9 < 7.0 {
			t.Errorf("Selected text role %s on selectedBg contrast %.2f:1 (%s on %s), want >= 7.0:1",
				role.name, ratio, contrastHex(t, role.fg), contrastHex(t, selected))
		}
	}

	// Non-text roles must clear >= 3:1 against canvas, surface, and selectedBg.
	nonTextRoles := []struct {
		name string
		fg   color.Color
	}{
		{"Border", theme.Border.GetForeground()},
		{"BorderFocus", theme.BorderFocus.GetForeground()},
		{"HeaderDivider", theme.HeaderDivider.GetForeground()},
	}
	for _, role := range nonTextRoles {
		fg := contrastRGB(t, role.fg)
		for _, bg := range []struct {
			bgName string
			bgCol  color.RGBA
		}{
			{"canvas", canvas},
			{"surface", surface},
			{"selected", selected},
		} {
			ratio := contrastRatio(fg, bg.bgCol)
			if ratio+1e-9 < 3.0 {
				t.Errorf("Non-text role %s on %s contrast %.2f:1 (%s on %s), want >= 3.0:1",
					role.name, bg.bgName, ratio, contrastHex(t, role.fg), contrastHex(t, bg.bgCol))
			}
		}
	}

	// Reverse pairs: canvas color used as text on status or accent backgrounds must clear >= 7:1.
	statusBgs := []struct {
		name string
		bg   color.Color
	}{
		{"Accent", theme.Tokens.Accent},
		{"Success", theme.Tokens.Success},
		{"Warning", theme.Tokens.Warning},
		{"Error", theme.Tokens.Error},
		{"Info", theme.Tokens.Info},
	}
	for _, s := range statusBgs {
		bgRGB := contrastRGB(t, s.bg)
		ratio := contrastRatio(canvas, bgRGB)
		if ratio+1e-9 < 7.0 {
			t.Errorf("Reverse text canvas on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
				s.name, ratio, contrastHex(t, theme.CanvasColor()), contrastHex(t, s.bg))
		}
	}
}

// TestAdaptiveThemeDarkPaletteRolesAndTokens checks the extended semantic tokens
// in Dark mode (Dim, SelectedBg, BorderFocus, Info) and verifies AAA contrast.
func TestAdaptiveThemeDarkPaletteRolesAndTokens(t *testing.T) {
	theme := NewAdaptiveTheme(true, true)
	if !theme.Color {
		t.Fatal("NewAdaptiveTheme(true, true) must enable color")
	}
	if !theme.IsDark {
		t.Fatal("NewAdaptiveTheme(true, true) must have IsDark=true")
	}

	canvas := contrastRGB(t, theme.CanvasColor())
	surface := contrastRGB(t, theme.SurfaceColor())
	selected := contrastRGB(t, theme.SelectedBgColor())

	// Dim role must reach >= 7:1 against canvas and surface.
	dimFG := contrastRGB(t, theme.Dim.GetForeground())
	for _, bg := range []struct {
		bgName string
		bgCol  color.RGBA
	}{
		{"canvas", canvas},
		{"surface", surface},
	} {
		ratio := contrastRatio(dimFG, bg.bgCol)
		if ratio+1e-9 < 7.0 {
			t.Errorf("Dark Dim on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
				bg.bgName, ratio, contrastHex(t, theme.Dim.GetForeground()), contrastHex(t, bg.bgCol))
		}
	}

	// BorderFocus must reach >= 3:1 against canvas, surface, and selectedBg.
	focusFG := contrastRGB(t, theme.BorderFocus.GetForeground())
	if ratio := contrastRatio(focusFG, canvas); ratio < 3.0 {
		t.Errorf("Dark BorderFocus on canvas contrast %.2f:1, want >= 3.0:1", ratio)
	}
	if ratio := contrastRatio(focusFG, surface); ratio < 3.0 {
		t.Errorf("Dark BorderFocus on surface contrast %.2f:1, want >= 3.0:1", ratio)
	}
	if ratio := contrastRatio(focusFG, selected); ratio < 3.0 {
		t.Errorf("Dark BorderFocus on selected contrast %.2f:1, want >= 3.0:1", ratio)
	}

	// Selected text must reach >= 7:1 on SelectedBg.
	textFG := contrastRGB(t, theme.Text.GetForeground())
	if ratio := contrastRatio(textFG, selected); ratio < 7.0 {
		t.Errorf("Dark Text on SelectedBg contrast %.2f:1, want >= 7.0:1", ratio)
	}
}

// TestThemeReferenceProfilesActualPairs verifies each reference profile against
// the actual pairs the UI paints. TrueColor and the chosen ANSI-256 mappings
// must meet AAA text / 3:1 non-text. ANSI-16 is explicitly best-effort: the
// standard reference palette cannot make every accent role AAA, so the test
// pins the documented failing roles instead of claiming they pass. None of
// this is a promise for a user-remapped terminal palette.
func TestThemeReferenceProfilesActualPairs(t *testing.T) {
	type pair struct {
		name string
		fg   color.Color
		bg   color.Color
		min  float64
	}

	profiles := []struct {
		name       string
		profile    colorprofile.Profile
		wantCanvas string
		aaa        bool
	}{
		{"TrueColor", colorprofile.TrueColor, "#1E1E2E", true},
		{"ANSI256", colorprofile.ANSI256, "#1C1C1C", true},
		{"ANSI16", colorprofile.ANSI, "#000000", false},
	}

	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			theme := NewThemeWithProfile(true, true, p.profile)
			if theme.Profile != p.profile {
				t.Fatalf("theme.Profile = %v, want %v", theme.Profile, p.profile)
			}
			if got := contrastHex(t, theme.CanvasColor()); got != p.wantCanvas {
				t.Fatalf("canvas = %s, want %s for %v", got, p.wantCanvas, p.profile)
			}

			canvas := contrastRGB(t, theme.CanvasColor())
			surface := contrastRGB(t, theme.SurfaceColor())
			selected := contrastRGB(t, theme.SelectedBgColor())

			// These structural pairs are required for every color profile: body
			// text, the selected row, and the meaningful borders. They are the
			// pairs a user cannot avoid seeing.
			required := []pair{
				{"Text/canvas", theme.Text.GetForeground(), canvas, 7},
				{"Text/surface", theme.Text.GetForeground(), surface, 7},
				{"SelectedText/selected", theme.Selected.GetForeground(), selected, 7},
				{"Muted/canvas", theme.Muted.GetForeground(), canvas, 7},
				{"Muted/surface", theme.Muted.GetForeground(), surface, 7},
				{"SelectedMuted/selected", theme.Muted.GetForeground(), selected, 7},
				{"Border/canvas", theme.Border.GetForeground(), canvas, 3},
				{"Border/surface", theme.Border.GetForeground(), surface, 3},
				{"Border/selected", theme.Border.GetForeground(), selected, 3},
				{"BorderFocus/canvas", theme.BorderFocus.GetForeground(), canvas, 3},
				{"BorderFocus/surface", theme.BorderFocus.GetForeground(), surface, 3},
				{"BorderFocus/selected", theme.BorderFocus.GetForeground(), selected, 3},
				{"HeaderTitle/surface", theme.HeaderTitle.GetForeground(), surface, 7},
				{"HeaderMuted/surface", theme.HeaderMuted.GetForeground(), surface, 7},
				{"HeaderDivider/surface", theme.HeaderDivider.GetForeground(), surface, 3},
			}

			if p.aaa {
				// Semantic text roles and the reverse status pairs are design
				// guards for the profiles that promise AAA.
				required = append(required,
					pair{"Title/canvas", theme.Title.GetForeground(), canvas, 7},
					pair{"Subtitle/canvas", theme.Subtitle.GetForeground(), canvas, 7},
					pair{"Section/canvas", theme.Section.GetForeground(), canvas, 7},
					pair{"Accent/canvas", theme.Accent.GetForeground(), canvas, 7},
					pair{"OK/canvas", theme.OK.GetForeground(), canvas, 7},
					pair{"Warn/canvas", theme.Warn.GetForeground(), canvas, 7},
					pair{"Bad/canvas", theme.Bad.GetForeground(), canvas, 7},
					pair{"Hint/canvas", theme.Hint.GetForeground(), canvas, 7},
					pair{"NavActive/canvas", theme.NavActive.GetForeground(), canvas, 7},
					pair{"NavInactive/canvas", theme.NavInactive.GetForeground(), canvas, 7},
					pair{"Dim/canvas", theme.Dim.GetForeground(), canvas, 7},
					pair{"Info/canvas", theme.Info.GetForeground(), canvas, 7},
					pair{"Notice/canvas", theme.Notice.GetForeground(), canvas, 7},
					pair{"HeaderAccent/surface", theme.HeaderAccent.GetForeground(), surface, 7},
					pair{"Canvas/Accent", theme.CanvasColor(), theme.Tokens.Accent, 7},
					pair{"Canvas/Success", theme.CanvasColor(), theme.Tokens.Success, 7},
					pair{"Canvas/Warning", theme.CanvasColor(), theme.Tokens.Warning, 7},
					pair{"Canvas/Error", theme.CanvasColor(), theme.Tokens.Error, 7},
					pair{"Canvas/Info", theme.CanvasColor(), theme.Tokens.Info, 7},
				)
			}

			for _, check := range required {
				ratio := contrastRatio(contrastRGB(t, check.fg), contrastRGB(t, check.bg))
				if ratio+1e-9 < check.min {
					t.Errorf("%s contrast %.2f:1, want >= %.1f:1", check.name, ratio, check.min)
				}
			}

			if p.aaa {
				return
			}

			// ANSI-16 is best-effort. The standard palette's bright magenta and
			// bright red cannot clear AAA on black, so those roles must stay
			// documented as failing rather than being presented as AAA.
			if claim := theme.ContrastClaim(); !strings.Contains(claim, "best-effort") {
				t.Errorf("ANSI-16 claim = %q, want an explicit best-effort statement", claim)
			}
			bestEffort := []pair{
				{"Title/canvas", theme.Title.GetForeground(), canvas, 7},
				{"Accent/canvas", theme.Accent.GetForeground(), canvas, 7},
				{"NavActive/canvas", theme.NavActive.GetForeground(), canvas, 7},
				{"Bad/canvas", theme.Bad.GetForeground(), canvas, 7},
				{"HeaderAccent/surface", theme.HeaderAccent.GetForeground(), surface, 7},
			}
			for _, check := range bestEffort {
				ratio := contrastRatio(contrastRGB(t, check.fg), contrastRGB(t, check.bg))
				if ratio >= check.min {
					t.Errorf("%s now meets %.1f:1 on the ANSI-16 reference palette; update the best-effort claim and mapping", check.name, check.min)
				}
			}
		})
	}

	// The remapping limitation is part of the claim, not a test-only log line.
	for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI256, colorprofile.ANSI} {
		claim := NewThemeWithProfile(true, true, p).ContrastClaim()
		if !strings.Contains(claim, "user-remapped") {
			t.Errorf("profile %v claim does not disclose user-remapped palettes: %q", p, claim)
		}
	}
}

// TestAppColorProfileMessageUsesReferenceMappings proves Bubble Tea's reported
// profile selects the explicit reference mapping without resetting route,
// filter or selection state, and that a TUIConfig profile is an explicit seam
// before the message arrives.
func TestAppColorProfileMessageUsesReferenceMappings(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 24, Color: true, Interactive: true}, contrastCatalog())
	app = contrastPress(t, app, tea.KeyPressMsg{Code: '3', Text: "3"})
	app.catalogModel.SetFilter("beta")
	if app.theme.Profile != colorprofile.TrueColor {
		t.Fatalf("initial profile = %v, want TrueColor", app.theme.Profile)
	}

	updated, _ := app.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI256})
	app = updated.(App)
	if app.theme.Profile != colorprofile.ANSI256 {
		t.Fatalf("after ANSI256 message profile = %v", app.theme.Profile)
	}
	if got := contrastHex(t, app.theme.CanvasColor()); got != "#1C1C1C" {
		t.Fatalf("ANSI256 canvas = %s, want #1C1C1C", got)
	}
	if app.ActiveRoute() != RouteExplore || app.catalogModel.list.FilterValue() != "beta" {
		t.Fatalf("profile message reset route/filter: route=%v filter=%q", app.ActiveRoute(), app.catalogModel.list.FilterValue())
	}
	if app.install.theme.Profile != colorprofile.ANSI256 || app.restore.theme.Profile != colorprofile.ANSI256 || app.activity.theme.Profile != colorprofile.ANSI256 {
		t.Fatal("profile message did not propagate to child themes")
	}

	updated, _ = app.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	app = updated.(App)
	if app.theme.Profile != colorprofile.ANSI {
		t.Fatalf("after ANSI message profile = %v", app.theme.Profile)
	}
	if got := contrastHex(t, app.theme.CanvasColor()); got != "#000000" {
		t.Fatalf("ANSI canvas = %s, want #000000", got)
	}
	if !strings.Contains(app.theme.ContrastClaim(), "best-effort") {
		t.Fatalf("ANSI claim = %q", app.theme.ContrastClaim())
	}

	// A non-color profile keeps the explicit identity rather than disabling it.
	updated, _ = app.Update(tea.ColorProfileMsg{Profile: colorprofile.NoTTY})
	app = updated.(App)
	if app.theme.Profile != colorprofile.ANSI {
		t.Fatalf("non-color profile changed the theme to %v", app.theme.Profile)
	}

	// The config seam resolves before any message arrives.
	seeded := NewApp(TUIConfig{Width: 100, Height: 24, Color: true, Profile: colorprofile.ANSI256}, contrastCatalog())
	if seeded.theme.Profile != colorprofile.ANSI256 {
		t.Fatalf("configured profile = %v, want ANSI256", seeded.theme.Profile)
	}
	if seeded.install.theme.Profile != colorprofile.ANSI256 || seeded.restore.theme.Profile != colorprofile.ANSI256 || seeded.catalogModel.theme.Profile != colorprofile.ANSI256 {
		t.Fatal("configured profile did not reach the child constructors")
	}
}

// TestAppBackgroundColorAdaptation verifies that root App queries background color
// in interactive colored mode, retains the single agentic-dark identity when BackgroundColorMsg arrives,
// and starts with tested dark palette before and after any reply.
func TestAppBackgroundColorAdaptation(t *testing.T) {
	app := NewApp(TUIConfig{Width: 80, Height: 24, Color: true, Interactive: true}, contrastCatalog())

	// Initial theme is dark before any reply.
	if !app.theme.IsDark {
		t.Fatal("initial theme must be Dark")
	}
	if got := contrastHex(t, app.theme.CanvasColor()); got != wantPaletteBackground {
		t.Fatalf("initial canvas = %s, want %s", got, wantPaletteBackground)
	}

	// Init must batch RequestBackgroundColor in interactive colored mode.
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}

	// Simulate receiving BackgroundColorMsg for light background.
	// tea.BackgroundColorMsg wraps a color.Color. We supply an arbitrary light color (e.g. white).
	lightMsg := tea.BackgroundColorMsg{Color: color.RGBA{R: 240, G: 245, B: 245, A: 255}}
	if lightMsg.IsDark() {
		t.Fatal("test color must be identified as light by msg.IsDark()")
	}

	updated, _ := app.Update(lightMsg)
	appUpdated, ok := updated.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", updated)
	}

	// B10 T29: Single dark identity; background-color messages must NOT switch theme to light.
	if !appUpdated.theme.IsDark {
		t.Fatal("after BackgroundColorMsg, theme.IsDark must still be true")
	}
	if got := contrastHex(t, appUpdated.theme.CanvasColor()); got != wantPaletteBackground {
		t.Fatalf("after light msg canvas = %s, want %s", got, wantPaletteBackground)
	}
	if got := contrastHex(t, appUpdated.theme.DefaultForeground()); got != wantPaletteForeground {
		t.Fatalf("after light msg foreground = %s, want %s", got, wantPaletteForeground)
	}

	// Verify child screens retain the dark theme.
	if !appUpdated.catalogModel.theme.IsDark {
		t.Error("catalogModel theme must remain dark")
	}
	if !appUpdated.install.theme.IsDark {
		t.Error("install theme must remain dark")
	}
	if !appUpdated.restore.theme.IsDark {
		t.Error("restore theme must remain dark")
	}
	if !appUpdated.activity.theme.IsDark {
		t.Error("activity theme must remain dark")
	}

	// Now simulate receiving a dark BackgroundColorMsg.
	darkMsg := tea.BackgroundColorMsg{Color: color.RGBA{R: 10, G: 20, B: 25, A: 255}}
	if !darkMsg.IsDark() {
		t.Fatal("test color must be identified as dark by msg.IsDark()")
	}
	updatedDark, _ := appUpdated.Update(darkMsg)
	appDark := updatedDark.(App)
	if !appDark.theme.IsDark {
		t.Fatal("after dark BackgroundColorMsg, theme.IsDark must be true")
	}
}

// TestAppLateBackgroundColorMsgPreservesState verifies that receiving a late
// background message does not switch scheme, reset selection, route, filter query, or scroll.
func TestAppLateBackgroundColorMsgPreservesState(t *testing.T) {
	app := NewApp(TUIConfig{Width: 80, Height: 24, Color: true, Interactive: true}, contrastCatalog())

	// Switch to Explore route (index 2).
	app = contrastPress(t, app, tea.KeyPressMsg{Code: '3', Text: "3"})
	if app.ActiveRoute() != RouteExplore {
		t.Fatalf("route = %v, want RouteExplore", app.ActiveRoute())
	}

	// Set filter query in catalog.
	app.catalogModel.SetFilter("beta")
	if app.catalogModel.list.FilterValue() != "beta" {
		t.Fatalf("filter value = %q, want beta", app.catalogModel.list.FilterValue())
	}

	// Deliver late light BackgroundColorMsg.
	lightMsg := tea.BackgroundColorMsg{Color: color.RGBA{R: 240, G: 245, B: 245, A: 255}}
	updated, _ := app.Update(lightMsg)
	appUpdated := updated.(App)

	// State must be 100% preserved.
	if appUpdated.ActiveRoute() != RouteExplore {
		t.Errorf("route changed to %v, want RouteExplore", appUpdated.ActiveRoute())
	}
	if appUpdated.catalogModel.list.FilterValue() != "beta" {
		t.Errorf("filter value changed to %q, want beta", appUpdated.catalogModel.list.FilterValue())
	}
	if !appUpdated.theme.IsDark {
		t.Error("theme must remain dark")
	}
	if got := contrastHex(t, appUpdated.theme.CanvasColor()); got != wantPaletteBackground {
		t.Errorf("canvas changed to %s, want %s", got, wantPaletteBackground)
	}
}

// TestAppColorOffNoQueryNoPaint verifies that color-off (Color=false) sessions
// emit no background query, have nil canvas/foreground, and no ANSI escapes.
func TestAppColorOffNoQueryNoPaint(t *testing.T) {
	app := NewApp(TUIConfig{Width: 80, Height: 24, Color: false, Interactive: true}, contrastCatalog())

	// Plain theme:
	if app.theme.Color {
		t.Fatal("plain theme must have Color=false")
	}
	if app.theme.CanvasColor() != nil {
		t.Errorf("plain theme canvas = %v, want nil", app.theme.CanvasColor())
	}
	if app.theme.DefaultForeground() != nil {
		t.Errorf("plain theme foreground = %v, want nil", app.theme.DefaultForeground())
	}

	// BackgroundColorMsg in color-off mode must be a no-op:
	lightMsg := tea.BackgroundColorMsg{Color: color.RGBA{R: 240, G: 245, B: 245, A: 255}}
	updated, _ := app.Update(lightMsg)
	appUpdated := updated.(App)
	if appUpdated.theme.Color {
		t.Fatal("plain app must not enable color on BackgroundColorMsg")
	}

	// View must have nil background and foreground colors:
	view := appUpdated.View()
	if view.BackgroundColor != nil {
		t.Errorf("plain view BackgroundColor = %v, want nil", view.BackgroundColor)
	}
	if view.ForegroundColor != nil {
		t.Errorf("plain view ForegroundColor = %v, want nil", view.ForegroundColor)
	}
	if strings.ContainsRune(view.Content, '\x1b') {
		t.Fatalf("plain view contains ANSI escape sequences:\n%q", view.Content)
	}
}

// TestScreenPaletteAuditNoLocalRawColors audits the screen source files to ensure
// no feature-local raw hex colors are used in screen presentations.
func TestScreenPaletteAuditNoLocalRawColors(t *testing.T) {
	files := []string{
		"catalog_screen.go",
		"install.go",
		"restore.go",
		"activity.go",
		"onboarding.go",
	}

	hexPattern := regexp.MustCompile(`"#[0-9a-fA-F]{3,8}"`)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}
		matches := hexPattern.FindAll(data, -1)
		if len(matches) > 0 {
			t.Errorf("file %s contains local raw hex colors: %v", file, matches)
		}
	}
}
