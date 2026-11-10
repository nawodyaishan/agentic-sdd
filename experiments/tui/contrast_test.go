package tui

import (
	"image/color"
	"math"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Midnight Lagoon palette roles. These constants are the single source the
// theme and these tests share; a role that drifts off the researched palette
// must fail here before a visual review has to catch it.
const (
	wantPaletteBackground = "#0B1C22"
	wantPaletteSurface    = "#123039"
	wantPaletteForeground = "#FAF5E9"
	wantPaletteMuted      = "#B6CACB"
	wantPalettePrimary    = "#F39FEC"
	wantPaletteSecondary  = "#35DADA"
	wantPaletteSuccess    = "#84D987"
	wantPaletteWarning    = "#EAB946"
	wantPaletteError      = "#FBA7A0"
	wantPaletteBorder     = "#8FA9AD"
)

func contrastRGB(t *testing.T, c color.Color) color.RGBA {
	t.Helper()
	if c == nil {
		t.Fatal("nil color")
	}
	return color.RGBAModel.Convert(c).(color.RGBA)
}

func contrastHex(t *testing.T, c color.Color) string {
	t.Helper()
	rgb := contrastRGB(t, c)
	const digits = "0123456789ABCDEF"
	return string([]byte{
		'#',
		digits[rgb.R>>4], digits[rgb.R&0x0f],
		digits[rgb.G>>4], digits[rgb.G&0x0f],
		digits[rgb.B>>4], digits[rgb.B&0x0f],
	})
}

// contrastLuminance is the WCAG 2 relative luminance of an sRGB color.
func contrastLuminance(rgb color.RGBA) float64 {
	channel := func(v uint8) float64 {
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(rgb.R) + 0.7152*channel(rgb.G) + 0.0722*channel(rgb.B)
}

// contrastRatio is the WCAG 2 contrast ratio between two opaque colors.
func contrastRatio(a, b color.RGBA) float64 {
	la, lb := contrastLuminance(a), contrastLuminance(b)
	hi, lo := math.Max(la, lb), math.Min(la, lb)
	return (hi + 0.05) / (lo + 0.05)
}

func contrastCatalog() Catalog {
	return Catalog{Items: []CatalogItem{
		{ID: "alpha-skill", Kind: "skill", Origin: "core"},
		{ID: "beta-profile", Kind: "profile", Origin: "core"},
	}}
}

func contrastPress(t *testing.T, app App, key tea.KeyPressMsg) App {
	t.Helper()
	next, _ := app.Update(key)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated
}

// TestThemePaletteRolesMeetWCAG checks every foreground role against the
// background it is actually painted on. Text must reach WCAG AAA (7:1) and
// non-text borders/dividers at least 3:1.
func TestThemePaletteRolesMeetWCAG(t *testing.T) {
	theme := NewTheme(true)
	if !theme.Color {
		t.Fatal("NewTheme(true) must enable color")
	}

	body := contrastRGB(t, theme.CanvasColor())
	surface := contrastRGB(t, theme.HeaderBar.GetBackground())
	if got := contrastHex(t, theme.CanvasColor()); got != wantPaletteBackground {
		t.Fatalf("canvas = %s, want %s", got, wantPaletteBackground)
	}
	if got := contrastHex(t, theme.HeaderBar.GetBackground()); got != wantPaletteSurface {
		t.Fatalf("header surface = %s, want %s", got, wantPaletteSurface)
	}

	roles := []struct {
		name string
		fg   color.Color
		bg   color.RGBA
		min  float64
	}{
		{"Title", theme.Title.GetForeground(), body, 7},
		{"Subtitle", theme.Subtitle.GetForeground(), body, 7},
		{"Section", theme.Section.GetForeground(), body, 7},
		{"Accent", theme.Accent.GetForeground(), body, 7},
		{"OK", theme.OK.GetForeground(), body, 7},
		{"Warn", theme.Warn.GetForeground(), body, 7},
		{"Bad", theme.Bad.GetForeground(), body, 7},
		{"Hint", theme.Hint.GetForeground(), body, 7},
		{"NavActive", theme.NavActive.GetForeground(), body, 7},
		{"NavInactive", theme.NavInactive.GetForeground(), body, 7},
		{"DefaultForeground", theme.DefaultForeground(), body, 7},
		{"Border", theme.Border.GetForeground(), body, 3},
		{"HeaderTitle", theme.HeaderTitle.GetForeground(), surface, 7},
		{"HeaderAccent", theme.HeaderAccent.GetForeground(), surface, 7},
		{"HeaderMuted", theme.HeaderMuted.GetForeground(), surface, 7},
		{"HeaderDivider", theme.HeaderDivider.GetForeground(), surface, 3},
	}

	for _, role := range roles {
		fg := contrastRGB(t, role.fg)
		ratio := contrastRatio(fg, role.bg)
		if ratio+1e-9 < role.min {
			t.Errorf("%s contrast %.2f:1 (%s on %s), want >= %.1f:1",
				role.name, ratio, contrastHex(t, role.fg), contrastHex(t, color.RGBA(role.bg)), role.min)
		}
	}

	// Header roles must paint their own surface background: composing them on
	// the body background would silently change their real contrast.
	headerRoles := []struct {
		name  string
		style interface{ GetBackground() color.Color }
	}{
		{"HeaderBar", theme.HeaderBar},
		{"HeaderTitle", theme.HeaderTitle},
		{"HeaderAccent", theme.HeaderAccent},
		{"HeaderMuted", theme.HeaderMuted},
		{"HeaderDivider", theme.HeaderDivider},
	}
	for _, role := range headerRoles {
		if got := contrastHex(t, role.style.GetBackground()); got != wantPaletteSurface {
			t.Errorf("%s background = %s, want surface %s", role.name, got, wantPaletteSurface)
		}
	}
}

// TestThemeUsesMidnightLagoonPalette pins each style to the researched palette
// instead of accepting any color pair that happens to pass the ratio check.
func TestThemeUsesMidnightLagoonPalette(t *testing.T) {
	theme := NewTheme(true)

	roles := []struct {
		name string
		got  color.Color
		want string
	}{
		{"Title", theme.Title.GetForeground(), wantPalettePrimary},
		{"Subtitle", theme.Subtitle.GetForeground(), wantPaletteMuted},
		{"Section", theme.Section.GetForeground(), wantPaletteSecondary},
		{"Accent", theme.Accent.GetForeground(), wantPalettePrimary},
		{"OK", theme.OK.GetForeground(), wantPaletteSuccess},
		{"Warn", theme.Warn.GetForeground(), wantPaletteWarning},
		{"Bad", theme.Bad.GetForeground(), wantPaletteError},
		{"Hint", theme.Hint.GetForeground(), wantPaletteMuted},
		{"NavActive", theme.NavActive.GetForeground(), wantPalettePrimary},
		{"NavInactive", theme.NavInactive.GetForeground(), wantPaletteForeground},
		{"Border", theme.Border.GetForeground(), wantPaletteBorder},
		{"HeaderTitle", theme.HeaderTitle.GetForeground(), wantPaletteForeground},
		{"HeaderAccent", theme.HeaderAccent.GetForeground(), wantPalettePrimary},
		{"HeaderMuted", theme.HeaderMuted.GetForeground(), wantPaletteMuted},
		{"HeaderDivider", theme.HeaderDivider.GetForeground(), wantPaletteBorder},
		{"HeaderBar background", theme.HeaderBar.GetBackground(), wantPaletteSurface},
		{"Canvas", theme.CanvasColor(), wantPaletteBackground},
		{"DefaultForeground", theme.DefaultForeground(), wantPaletteForeground},
	}
	for _, role := range roles {
		if got := contrastHex(t, role.got); got != role.want {
			t.Errorf("%s = %s, want %s", role.name, got, role.want)
		}
	}
}

// TestThemeColorOffIsPlain proves the color=false contract: zero styles render
// their input unchanged and no terminal background is requested.
func TestThemeColorOffIsPlain(t *testing.T) {
	theme := NewTheme(false)
	if theme.Color {
		t.Fatal("NewTheme(false) must keep Color=false")
	}

	styles := []struct {
		name  string
		style interface{ Render(...string) string }
	}{
		{"Title", theme.Title},
		{"Subtitle", theme.Subtitle},
		{"Section", theme.Section},
		{"Accent", theme.Accent},
		{"OK", theme.OK},
		{"Warn", theme.Warn},
		{"Bad", theme.Bad},
		{"Hint", theme.Hint},
		{"NavActive", theme.NavActive},
		{"NavInactive", theme.NavInactive},
		{"Border", theme.Border},
		{"HeaderBar", theme.HeaderBar},
		{"HeaderTitle", theme.HeaderTitle},
		{"HeaderAccent", theme.HeaderAccent},
		{"HeaderMuted", theme.HeaderMuted},
		{"HeaderDivider", theme.HeaderDivider},
	}
	for _, style := range styles {
		if got := style.style.Render("plain"); got != "plain" {
			t.Errorf("%s.Render changed plain text: %q", style.name, got)
		}
	}
	if theme.CanvasColor() != nil {
		t.Errorf("plain theme canvas = %v, want nil", theme.CanvasColor())
	}
	if theme.DefaultForeground() != nil {
		t.Errorf("plain theme foreground = %v, want nil", theme.DefaultForeground())
	}

	app := NewApp(TUIConfig{Width: 80, Height: 24}, contrastCatalog())
	view := app.View()
	if view.BackgroundColor != nil {
		t.Errorf("plain view background = %v, want nil", view.BackgroundColor)
	}
	if view.ForegroundColor != nil {
		t.Errorf("plain view foreground = %v, want nil", view.ForegroundColor)
	}
	if strings.ContainsRune(view.Content, '\x1b') {
		t.Fatalf("plain view contains ANSI escapes:\n%q", view.Content)
	}
}

// TestColoredViewKeepsLegacyPlainPathUnstyled checks the no-color path still has
// the single unpadded legacy header row; only colored runs get the taller bar.
func TestColoredViewKeepsLegacyPlainPathUnstyled(t *testing.T) {
	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		for _, height := range []int{8, 12, 16, 24, 30} {
			app := NewApp(TUIConfig{Width: width, Height: height, Color: false}, contrastCatalog())
			lines := strings.Split(app.View().Content, "\n")
			if len(lines) != height {
				t.Fatalf("w=%d h=%d plain view has %d rows, want %d", width, height, len(lines), height)
			}
			first := ansi.Strip(lines[0])
			if strings.TrimSpace(first) == "" {
				t.Fatalf("w=%d h=%d plain header starts with a blank surface row", width, height)
			}
			if strings.HasPrefix(first, "  ") {
				t.Fatalf("w=%d h=%d plain header gained colored-bar padding: %q", width, height, first)
			}
			if !strings.Contains(first, BrandName) {
				t.Fatalf("w=%d h=%d plain header missing brand/title: %q", width, height, first)
			}
		}
	}
}

// TestColoredViewSetsTerminalCanvas proves the full-view background comes from
// the pinned Bubble Tea v2 View API and only for colored sessions.
func TestColoredViewSetsTerminalCanvas(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 24, Color: true}, contrastCatalog())
	view := app.View()
	if view.BackgroundColor == nil {
		t.Fatal("colored view must set BackgroundColor")
	}
	if got := contrastHex(t, view.BackgroundColor); got != wantPaletteBackground {
		t.Errorf("view background = %s, want %s", got, wantPaletteBackground)
	}
	if view.ForegroundColor == nil {
		t.Fatal("colored view must set ForegroundColor")
	}
	if got := contrastHex(t, view.ForegroundColor); got != wantPaletteForeground {
		t.Errorf("view foreground = %s, want %s", got, wantPaletteForeground)
	}
}

// TestColoredTopBarUsesSurfacePaddingAndDivider checks the roomier bar shape:
// a full-width surface row, a two-cell padded title/control row and a divider.
func TestColoredTopBarUsesSurfacePaddingAndDivider(t *testing.T) {
	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		app := NewApp(TUIConfig{Width: width, Height: 24, Color: true}, contrastCatalog())
		lines := strings.Split(app.View().Content, "\n")

		surface := ansi.Strip(lines[0])
		if strings.TrimSpace(surface) != "" {
			t.Fatalf("w=%d header row 0 is not a blank surface row: %q", width, surface)
		}
		if got := ansi.StringWidth(lines[0]); got != width {
			t.Fatalf("w=%d surface row width = %d, want %d", width, got, width)
		}

		title := ansi.Strip(lines[1])
		if !strings.HasPrefix(title, "  ") || !strings.HasSuffix(title, "  ") {
			t.Fatalf("w=%d title row lacks 2-cell padding: %q", width, title)
		}
		if got := ansi.StringWidth(lines[1]); got != width {
			t.Fatalf("w=%d title row width = %d, want %d", width, got, width)
		}
		if !strings.Contains(title, BrandName) {
			t.Fatalf("w=%d title row lost the product name: %q", width, title)
		}
		if !strings.Contains(title, "[nav]") {
			t.Fatalf("w=%d title row lost the textual focus indicator: %q", width, title)
		}
		if width >= 80 && !strings.Contains(title, "by "+BrandAuthor) {
			t.Fatalf("w=%d title row lost author attribution: %q", width, title)
		}

		divider := ansi.Strip(lines[2])
		if strings.Trim(divider, "─") != "" {
			t.Fatalf("w=%d header divider row is not a full colored rule: %q", width, divider)
		}
		if got := ansi.StringWidth(lines[2]); got != width {
			t.Fatalf("w=%d divider width = %d, want %d", width, got, width)
		}
	}
}

// TestColoredTopBarCollapsesOnShortTerminals verifies the height budget:
// at least two header rows become one compact row below height 12.
func TestColoredTopBarCollapsesOnShortTerminals(t *testing.T) {
	for _, height := range []int{8, 12, 16, 24, 30} {
		app := NewApp(TUIConfig{Width: 100, Height: height, Color: true}, contrastCatalog())
		lines := strings.Split(app.View().Content, "\n")
		if len(lines) != height {
			t.Fatalf("h=%d view has %d rows, want %d", height, len(lines), height)
		}

		titleRow := 0
		switch {
		case height >= 14:
			if strings.TrimSpace(ansi.Strip(lines[0])) != "" {
				t.Fatalf("h=%d normal height lost the surface row: %q", height, ansi.Strip(lines[0]))
			}
			titleRow = 1
			if strings.Trim(ansi.Strip(lines[2]), "─") != "" {
				t.Fatalf("h=%d normal height lost the header divider: %q", height, ansi.Strip(lines[2]))
			}
		case height >= 12:
			titleRow = 0
			if strings.Trim(ansi.Strip(lines[1]), "─") != "" {
				t.Fatalf("h=%d compact height lost the header divider: %q", height, ansi.Strip(lines[1]))
			}
		default:
			titleRow = 0
		}
		title := ansi.Strip(lines[titleRow])
		if !strings.Contains(title, BrandName) || !strings.Contains(title, "[nav]") {
			t.Fatalf("h=%d title row at row %d = %q", height, titleRow, title)
		}
	}
}

// TestColoredFocusIndicatorStaysTextualAndAccented proves focus is not encoded
// by color alone: the bracket text remains and carries the accent style.
func TestColoredFocusIndicatorStaysTextualAndAccented(t *testing.T) {
	app := NewApp(TUIConfig{Width: 100, Height: 24, Color: true}, contrastCatalog())

	check := func(name string, app App) {
		t.Helper()
		content := app.View().Content
		want := app.theme.HeaderAccent.Render("[" + name + "]")
		if !strings.Contains(content, "["+name+"]") {
			t.Fatalf("focus %q missing as text", name)
		}
		if !strings.Contains(content, want) {
			t.Fatalf("focus %q is not rendered with the header accent style", name)
		}
	}
	check("nav", app)

	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyTab}) // nav -> content
	check("content", app)

	app = contrastPress(t, app, tea.KeyPressMsg{Code: '3', Text: "3"}) // Explore
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyTab})     // content -> nav
	app = contrastPress(t, app, tea.KeyPressMsg{Code: tea.KeyTab})     // nav -> categories
	check("categories", app)
}

// TestColoredChromePreservesControlsAndBounds checks the actual breakpoints the
// owner asked about: the header recalculates body rows without hiding the
// category control, footer/paging hints or terminal bounds.
func TestColoredChromePreservesControlsAndBounds(t *testing.T) {
	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		for _, height := range []int{8, 12, 16, 24, 30} {
			for _, route := range []string{"1", "2", "3", "4"} {
				app := NewApp(TUIConfig{Width: width, Height: height, Color: true}, contrastCatalog())
				app = contrastPress(t, app, tea.KeyPressMsg{Code: rune(route[0]), Text: route})
				lines := strings.Split(app.View().Content, "\n")
				if len(lines) != height {
					t.Fatalf("w=%d h=%d route=%s has %d rows, want %d", width, height, route, len(lines), height)
				}
				for row, line := range lines {
					if got := ansi.StringWidth(line); got > width {
						t.Fatalf("w=%d h=%d route=%s row %d width %d > %d", width, height, route, row, got, width)
					}
				}

				footer := ansi.Strip(lines[len(lines)-1])
				if !strings.Contains(strings.ToLower(footer), "quit") && !strings.Contains(footer, " q") {
					t.Fatalf("w=%d h=%d route=%s footer hidden: %q", width, height, route, footer)
				}

				if route == "3" && height >= 12 {
					if !strings.Contains(ansi.Strip(app.View().Content), "Category:") {
						t.Fatalf("w=%d h=%d Explore lost the category control", width, height)
					}
					if !strings.Contains(footer, "q quit") {
						t.Fatalf("w=%d h=%d Explore lost paging/quit footer: %q", width, height, footer)
					}
				}

				headerRows := app.headerRowCount()
				footerRows := 2
				if height <= 8 {
					footerRows = 1
				}
				if headerRows < 1 || headerRows+footerRows >= height {
					t.Fatalf("w=%d h=%d route=%s header %d + footer %d leaves no body row",
						width, height, route, headerRows, footerRows)
				}
			}
		}
	}
}

// TestColoredHelpOverlayKeepsTitleAndFooter checks the header does not starve
// the help overlay at short heights.
func TestColoredHelpOverlayKeepsTitleAndFooter(t *testing.T) {
	for _, height := range []int{8, 12, 16} {
		for _, width := range []int{40, 80, 110} {
			app := NewApp(TUIConfig{Width: width, Height: height, Color: true}, contrastCatalog())
			app = contrastPress(t, app, tea.KeyPressMsg{Code: '?', Text: "?"})
			lines := strings.Split(app.View().Content, "\n")
			if len(lines) != height {
				t.Fatalf("w=%d h=%d help has %d rows, want %d", width, height, len(lines), height)
			}
			if !strings.Contains(app.View().Content, "Keyboard & Navigation Reference") {
				t.Fatalf("w=%d h=%d help title hidden", width, height)
			}
			footer := ansi.Strip(lines[len(lines)-1])
			if !strings.Contains(footer, "quit") {
				t.Fatalf("w=%d h=%d help footer hidden: %q", width, height, footer)
			}
		}
	}
}
