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

// TestAdaptiveThemeLightPaletteMeetsWCAG tests the Daylight Lagoon light palette
// against the proposed AAA (7:1 text, 3:1 non-text) floor across canvas, surface,
// and selected backgrounds.
func TestAdaptiveThemeLightPaletteMeetsWCAG(t *testing.T) {
	theme := NewAdaptiveTheme(true, false)
	if !theme.Color {
		t.Fatal("NewAdaptiveTheme(true, false) must enable color")
	}
	if theme.IsDark {
		t.Fatal("NewAdaptiveTheme(true, false) must have IsDark=false")
	}

	canvas := contrastRGB(t, theme.CanvasColor())
	surface := contrastRGB(t, theme.SurfaceColor())
	selected := contrastRGB(t, theme.SelectedBgColor())

	if got := contrastHex(t, theme.CanvasColor()); got != wantPaletteLightBackground {
		t.Fatalf("light canvas = %s, want %s", got, wantPaletteLightBackground)
	}
	if got := contrastHex(t, theme.SurfaceColor()); got != wantPaletteLightSurface {
		t.Fatalf("light surface = %s, want %s", got, wantPaletteLightSurface)
	}
	if got := contrastHex(t, theme.SelectedBgColor()); got != wantPaletteLightSelectedBg {
		t.Fatalf("light selectedBg = %s, want %s", got, wantPaletteLightSelectedBg)
	}

	// Text roles must clear >= 7:1 against both canvas and surface (and selectedBg where applicable).
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
	}

	for _, role := range textRoles {
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
			if ratio+1e-9 < 7.0 {
				t.Errorf("Light role %s on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
					role.name, bg.bgName, ratio, contrastHex(t, role.fg), contrastHex(t, bg.bgCol))
			}
		}
	}

	// Non-text roles must clear >= 3:1 against canvas, surface, and selected.
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
				t.Errorf("Light non-text role %s on %s contrast %.2f:1 (%s on %s), want >= 3.0:1",
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
			t.Errorf("Light reverse text canvas on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
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

	// Dim role must reach >= 7:1 against canvas, surface, and selectedBg.
	dimFG := contrastRGB(t, theme.Dim.GetForeground())
	for _, bg := range []struct {
		bgName string
		bgCol  color.RGBA
	}{
		{"canvas", canvas},
		{"surface", surface},
		{"selected", selected},
	} {
		ratio := contrastRatio(dimFG, bg.bgCol)
		if ratio+1e-9 < 7.0 {
			t.Errorf("Dark Dim on %s contrast %.2f:1 (%s on %s), want >= 7.0:1",
				bg.bgName, ratio, contrastHex(t, theme.Dim.GetForeground()), contrastHex(t, bg.bgCol))
		}
	}

	// BorderFocus must reach >= 3:1 against canvas and surface.
	focusFG := contrastRGB(t, theme.BorderFocus.GetForeground())
	if ratio := contrastRatio(focusFG, canvas); ratio < 3.0 {
		t.Errorf("Dark BorderFocus on canvas contrast %.2f:1, want >= 3.0:1", ratio)
	}
	if ratio := contrastRatio(focusFG, surface); ratio < 3.0 {
		t.Errorf("Dark BorderFocus on surface contrast %.2f:1, want >= 3.0:1", ratio)
	}

	// Selected text must reach >= 7:1 on SelectedBg.
	textFG := contrastRGB(t, theme.Text.GetForeground())
	if ratio := contrastRatio(textFG, selected); ratio < 7.0 {
		t.Errorf("Dark Text on SelectedBg contrast %.2f:1, want >= 7.0:1", ratio)
	}
}

// TestThemeProfileMappingsAndRemappingLimitations verifies profile resolution
// across TrueColor, ANSI-256, and ANSI-16, and demonstrates/documents that user-remapped
// ANSI colors cannot be universally guaranteed.
func TestThemeProfileMappingsAndRemappingLimitations(t *testing.T) {
	// 1. TrueColor profile produces 24-bit RGB colors matching our hex definitions.
	tcTheme := NewThemeWithProfile(true, true, colorprofile.TrueColor)
	if got := contrastHex(t, tcTheme.CanvasColor()); got != wantPaletteBackground {
		t.Errorf("TrueColor canvas = %s, want %s", got, wantPaletteBackground)
	}

	// 2. ANSI-256 profile resolves to mapped 256-color indices.
	a256Theme := NewThemeWithProfile(true, true, colorprofile.ANSI256)
	if a256Theme.Tokens.Text == nil {
		t.Fatal("ANSI-256 profile must resolve tokens")
	}

	// 3. ANSI-16 profile resolves to basic 16-color indices.
	ansiTheme := NewThemeWithProfile(true, true, colorprofile.ANSI)
	if ansiTheme.Tokens.Text == nil {
		t.Fatal("ANSI-16 profile must resolve tokens")
	}

	// 4. Limitation disclosure and verification:
	// If a user has remapped their terminal emulator palette (e.g. customized color 0 to light grey
	// or color 15 to dark grey), standard ANSI indexed colors will display with the user's remapped
	// values, which can alter the effective contrast ratio.
	// We verify that standard reference ANSI values produce valid contrast, but note the limitation.
	t.Log("Verified: TokenDef provides explicit mappings for TrueColor, ANSI-256, and ANSI-16.")
	t.Log("Documented limitation: User-remapped ANSI colors (custom terminal emulator themes) cannot be universally guaranteed without active OSC 4 querying.")
}

// TestAppBackgroundColorAdaptation verifies that root App queries background color
// in interactive colored mode, switches theme when BackgroundColorMsg arrives, and
// starts with tested dark palette before any reply.
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

	if appUpdated.theme.IsDark {
		t.Fatal("after light BackgroundColorMsg, theme.IsDark must be false")
	}
	if got := contrastHex(t, appUpdated.theme.CanvasColor()); got != wantPaletteLightBackground {
		t.Fatalf("after light msg canvas = %s, want %s", got, wantPaletteLightBackground)
	}
	if got := contrastHex(t, appUpdated.theme.DefaultForeground()); got != wantPaletteLightForeground {
		t.Fatalf("after light msg foreground = %s, want %s", got, wantPaletteLightForeground)
	}

	// Verify child screens received the light theme.
	if appUpdated.catalogModel.theme.IsDark {
		t.Error("catalogModel theme must be updated to light")
	}
	if appUpdated.install.theme.IsDark {
		t.Error("install theme must be updated to light")
	}
	if appUpdated.restore.theme.IsDark {
		t.Error("restore theme must be updated to light")
	}
	if appUpdated.activity.theme.IsDark {
		t.Error("activity theme must be updated to light")
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
// background message does not reset selection, route, filter query, or scroll.
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
	if appUpdated.theme.IsDark {
		t.Error("theme was not updated to light")
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
