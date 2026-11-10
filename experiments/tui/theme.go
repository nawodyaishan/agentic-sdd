package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// Midnight Lagoon (Dark) palette. The owner-requested high-contrast shell uses this
// researched, OKLab-balanced set; every text role sits on #0B1C22 (body),
// #123039 (header band / surface), or #132E37 (selected row) and clears WCAG AAA (>= 7:1),
// while borders and indicators clear the >= 3:1 non-text threshold.
const (
	paletteBackground  = "#0B1C22" // full-view canvas
	paletteSurface     = "#123039" // top-bar band / raised surface
	paletteForeground  = "#FAF5E9" // body and header primary text
	paletteMuted       = "#B6CACB" // secondary text
	paletteDim         = "#AEC2C3" // subtle neutral text (clears AAA >= 7:1)
	palettePrimary     = "#F39FEC" // violet accent / focus
	paletteSecondary   = "#35DADA" // cyan section accent / info
	paletteSuccess     = "#84D987" // labelled success
	paletteWarning     = "#EAB946" // labelled warning
	paletteError       = "#FBA7A0" // labelled error
	paletteBorder      = "#8FA9AD" // dividers and borders
	paletteBorderFocus = "#F39FEC" // focused border (violet accent, >= 3:1)
	paletteSelectedBg  = "#132E37" // selected row background tint
)

// Daylight Lagoon (Light) palette. Inverted high-contrast companion palette;
// every text role sits on #F8FAFA (canvas), #EDF3F4 (surface), or #E0E9EB (selectedBg)
// and clears the proposed AAA >= 7:1 floor, while borders and indicators clear >= 3:1.
const (
	paletteLightBackground  = "#F8FAFA" // full-view canvas
	paletteLightSurface     = "#EDF3F4" // top-bar band / raised surface
	paletteLightForeground  = "#0A1F26" // body and header primary text
	paletteLightMuted       = "#233A3F" // secondary text
	paletteLightDim         = "#30484D" // subtle neutral text
	paletteLightPrimary     = "#661772" // violet accent / focus
	paletteLightSecondary   = "#004F59" // cyan section accent / info
	paletteLightSuccess     = "#0B5322" // labelled success
	paletteLightWarning     = "#643E00" // labelled warning
	paletteLightError       = "#7E1414" // labelled error
	paletteLightBorder      = "#688487" // dividers and borders
	paletteLightBorderFocus = "#661772" // focused border
	paletteLightSelectedBg  = "#E0E9EB" // selected row background tint
)

// TokenDef specifies explicit TrueColor, ANSI-256, and ANSI-16 representations
// for one semantic token role.
type TokenDef struct {
	TrueColor string
	ANSI256   string
	ANSI      string
}

// Resolve maps the token definition to a concrete color.Color based on the
// terminal color profile.
func (td TokenDef) Resolve(p colorprofile.Profile) color.Color {
	complete := lipgloss.Complete(p)
	return complete(
		lipgloss.Color(td.ANSI),
		lipgloss.Color(td.ANSI256),
		lipgloss.Color(td.TrueColor),
	)
}

type paletteDefs struct {
	Canvas      TokenDef
	Surface     TokenDef
	Text        TokenDef
	Muted       TokenDef
	Dim         TokenDef
	Accent      TokenDef
	SelectedBg  TokenDef
	Border      TokenDef
	BorderFocus TokenDef
	Success     TokenDef
	Warning     TokenDef
	Error       TokenDef
	Info        TokenDef
}

var darkDefs = paletteDefs{
	Canvas:      TokenDef{TrueColor: paletteBackground, ANSI256: "234", ANSI: "0"},
	Surface:     TokenDef{TrueColor: paletteSurface, ANSI256: "236", ANSI: "8"},
	Text:        TokenDef{TrueColor: paletteForeground, ANSI256: "231", ANSI: "15"},
	Muted:       TokenDef{TrueColor: paletteMuted, ANSI256: "251", ANSI: "7"},
	Dim:         TokenDef{TrueColor: paletteDim, ANSI256: "248", ANSI: "7"},
	Accent:      TokenDef{TrueColor: palettePrimary, ANSI256: "219", ANSI: "13"},
	SelectedBg:  TokenDef{TrueColor: paletteSelectedBg, ANSI256: "237", ANSI: "8"},
	Border:      TokenDef{TrueColor: paletteBorder, ANSI256: "248", ANSI: "7"},
	BorderFocus: TokenDef{TrueColor: paletteBorderFocus, ANSI256: "219", ANSI: "13"},
	Success:     TokenDef{TrueColor: paletteSuccess, ANSI256: "114", ANSI: "10"},
	Warning:     TokenDef{TrueColor: paletteWarning, ANSI256: "179", ANSI: "11"},
	Error:       TokenDef{TrueColor: paletteError, ANSI256: "217", ANSI: "9"},
	Info:        TokenDef{TrueColor: paletteSecondary, ANSI256: "44", ANSI: "14"},
}

var lightDefs = paletteDefs{
	Canvas:      TokenDef{TrueColor: paletteLightBackground, ANSI256: "231", ANSI: "15"},
	Surface:     TokenDef{TrueColor: paletteLightSurface, ANSI256: "254", ANSI: "7"},
	Text:        TokenDef{TrueColor: paletteLightForeground, ANSI256: "234", ANSI: "0"},
	Muted:       TokenDef{TrueColor: paletteLightMuted, ANSI256: "237", ANSI: "0"},
	Dim:         TokenDef{TrueColor: paletteLightDim, ANSI256: "239", ANSI: "8"},
	Accent:      TokenDef{TrueColor: paletteLightPrimary, ANSI256: "53", ANSI: "5"},
	SelectedBg:  TokenDef{TrueColor: paletteLightSelectedBg, ANSI256: "253", ANSI: "7"},
	Border:      TokenDef{TrueColor: paletteLightBorder, ANSI256: "244", ANSI: "8"},
	BorderFocus: TokenDef{TrueColor: paletteLightBorderFocus, ANSI256: "53", ANSI: "5"},
	Success:     TokenDef{TrueColor: paletteLightSuccess, ANSI256: "22", ANSI: "2"},
	Warning:     TokenDef{TrueColor: paletteLightWarning, ANSI256: "94", ANSI: "3"},
	Error:       TokenDef{TrueColor: paletteLightError, ANSI256: "88", ANSI: "1"},
	Info:        TokenDef{TrueColor: paletteLightSecondary, ANSI256: "23", ANSI: "6"},
}

func (p paletteDefs) resolve(prof colorprofile.Profile) SemanticTokens {
	return SemanticTokens{
		Canvas:      p.Canvas.Resolve(prof),
		Surface:     p.Surface.Resolve(prof),
		Text:        p.Text.Resolve(prof),
		Muted:       p.Muted.Resolve(prof),
		Dim:         p.Dim.Resolve(prof),
		Accent:      p.Accent.Resolve(prof),
		SelectedBg:  p.SelectedBg.Resolve(prof),
		Border:      p.Border.Resolve(prof),
		BorderFocus: p.BorderFocus.Resolve(prof),
		Success:     p.Success.Resolve(prof),
		Warning:     p.Warning.Resolve(prof),
		Error:       p.Error.Resolve(prof),
		Info:        p.Info.Resolve(prof),
	}
}

// SemanticTokens defines the resolved color values for all semantic UI roles.
type SemanticTokens struct {
	Canvas      color.Color
	Surface     color.Color
	Text        color.Color
	Muted       color.Color
	Dim         color.Color
	Accent      color.Color
	SelectedBg  color.Color
	Border      color.Color
	BorderFocus color.Color
	Success     color.Color
	Warning     color.Color
	Error       color.Color
	Info        color.Color
}

// Theme holds the Lip Gloss styles the screens share. A theme built with
// color=false is entirely plain: every style is the zero style, whose Render
// returns its input unchanged. That keeps headless output and tests
// deterministic regardless of the terminal's color profile.
type Theme struct {
	Color  bool
	IsDark bool

	// Tokens holds the resolved semantic color values for this theme.
	Tokens SemanticTokens

	// Existing standard roles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Section     lipgloss.Style
	Accent      lipgloss.Style
	OK          lipgloss.Style
	Warn        lipgloss.Style
	Bad         lipgloss.Style
	Hint        lipgloss.Style
	NavActive   lipgloss.Style
	NavInactive lipgloss.Style
	Border      lipgloss.Style

	// Extended semantic styles
	Text        lipgloss.Style
	Muted       lipgloss.Style
	Dim         lipgloss.Style
	Info        lipgloss.Style
	Selected    lipgloss.Style
	BorderFocus lipgloss.Style
	KeyHint     lipgloss.Style
	Notice      lipgloss.Style

	// Header-bar roles paint their own surface background so composing them
	// preserves the band instead of resetting to the terminal default.
	HeaderBar     lipgloss.Style
	HeaderTitle   lipgloss.Style
	HeaderAccent  lipgloss.Style
	HeaderMuted   lipgloss.Style
	HeaderDivider lipgloss.Style
}

// NewThemeWithProfile constructs a theme for a given color setting, light/dark mode,
// and color profile.
func NewThemeWithProfile(colorEnabled bool, isDark bool, profile colorprofile.Profile) Theme {
	if !colorEnabled {
		return Theme{}
	}

	darkTokens := darkDefs.resolve(profile)
	lightTokens := lightDefs.resolve(profile)

	ld := lipgloss.LightDark(isDark)
	tokens := SemanticTokens{
		Canvas:      ld(lightTokens.Canvas, darkTokens.Canvas),
		Surface:     ld(lightTokens.Surface, darkTokens.Surface),
		Text:        ld(lightTokens.Text, darkTokens.Text),
		Muted:       ld(lightTokens.Muted, darkTokens.Muted),
		Dim:         ld(lightTokens.Dim, darkTokens.Dim),
		Accent:      ld(lightTokens.Accent, darkTokens.Accent),
		SelectedBg:  ld(lightTokens.SelectedBg, darkTokens.SelectedBg),
		Border:      ld(lightTokens.Border, darkTokens.Border),
		BorderFocus: ld(lightTokens.BorderFocus, darkTokens.BorderFocus),
		Success:     ld(lightTokens.Success, darkTokens.Success),
		Warning:     ld(lightTokens.Warning, darkTokens.Warning),
		Error:       ld(lightTokens.Error, darkTokens.Error),
		Info:        ld(lightTokens.Info, darkTokens.Info),
	}

	body := lipgloss.NewStyle()
	header := lipgloss.NewStyle().Background(tokens.Surface)
	headerText := func(c color.Color) lipgloss.Style {
		return header.Foreground(c)
	}

	return Theme{
		Color:       true,
		IsDark:      isDark,
		Tokens:      tokens,
		Title:       body.Bold(true).Foreground(tokens.Accent),
		Subtitle:    body.Foreground(tokens.Muted),
		Section:     body.Bold(true).Foreground(tokens.Info),
		Accent:      body.Foreground(tokens.Accent),
		OK:          body.Foreground(tokens.Success),
		Warn:        body.Foreground(tokens.Warning),
		Bad:         body.Foreground(tokens.Error),
		Hint:        body.Foreground(tokens.Muted),
		NavActive:   body.Bold(true).Foreground(tokens.Accent),
		NavInactive: body.Foreground(tokens.Text),
		Border:      body.Foreground(tokens.Border),

		Text:        body.Foreground(tokens.Text),
		Muted:       body.Foreground(tokens.Muted),
		Dim:         body.Foreground(tokens.Dim),
		Info:        body.Foreground(tokens.Info),
		Selected:    body.Background(tokens.SelectedBg).Foreground(tokens.Text),
		BorderFocus: body.Foreground(tokens.BorderFocus),
		KeyHint:     body.Foreground(tokens.Muted),
		Notice:      body.Foreground(tokens.Warning),

		HeaderBar:     header,
		HeaderTitle:   headerText(tokens.Text).Bold(true),
		HeaderAccent:  headerText(tokens.Accent).Bold(true),
		HeaderMuted:   headerText(tokens.Muted),
		HeaderDivider: headerText(tokens.Border),
	}
}

// NewAdaptiveTheme builds a theme with the given color setting and light/dark mode
// using the default TrueColor profile.
func NewAdaptiveTheme(colorEnabled bool, isDark bool) Theme {
	return NewThemeWithProfile(colorEnabled, isDark, colorprofile.TrueColor)
}

// NewTheme builds the shared style set from the Midnight Lagoon palette.
// By default, it returns the tested Dark palette when color=true, or plain when color=false.
func NewTheme(colorEnabled bool) Theme {
	return NewAdaptiveTheme(colorEnabled, true)
}

// CanvasColor is the full-view terminal background for a colored session, or
// nil when color is off. The root View applies it through the Bubble Tea v2
// tea.View.BackgroundColor field.
func (t Theme) CanvasColor() color.Color {
	if !t.Color {
		return nil
	}
	return t.Tokens.Canvas
}

// DefaultForeground is the terminal foreground for unstyled body text, or nil
// when color is off. Styled roles override it; raw prose still needs AAA
// contrast against the canvas.
func (t Theme) DefaultForeground() color.Color {
	if !t.Color {
		return nil
	}
	return t.Tokens.Text
}

// SurfaceColor returns the surface/band background for a colored session, or
// nil when color is off.
func (t Theme) SurfaceColor() color.Color {
	if !t.Color {
		return nil
	}
	return t.Tokens.Surface
}

// SelectedBgColor returns the selected row background for a colored session, or
// nil when color is off.
func (t Theme) SelectedBgColor() color.Color {
	if !t.Color {
		return nil
	}
	return t.Tokens.SelectedBg
}
