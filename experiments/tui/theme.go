package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

// Agentic Dark palette based on Catppuccin Mocha.
//
// The TrueColor palette is the authoritative identity: every text role sits on
// #1E1E2E (canvas), #181825 (rail/header surface), or #313244 (selected row)
// and clears WCAG AAA (>= 7:1), while borders and indicators clear the >= 3:1
// non-text threshold.
//
// The ANSI-256 reference mappings are chosen and tested to meet the same
// actual-pair floor against the documented xterm 256-color reference palette.
// The ANSI-16 palette is best-effort only: its standard reference colors have
// no accent magenta or error red that clears AAA on black, so those roles are
// explicitly not claimed as AAA. User-remapped terminal palettes (OSC 4 or a
// custom terminal theme) are outside every guarantee here; these tests cover
// reference palettes, not a user's terminal.
const (
	paletteBackground  = "#1E1E2E" // Base: full-view canvas
	paletteSurface     = "#181825" // Mantle: top-bar band / rail surface
	paletteForeground  = "#CDD6F4" // Text: body and header primary text
	paletteMuted       = "#BAC2DE" // Subtext1: secondary text
	paletteDim         = "#A6ADC8" // Subtext0: subtle neutral / tertiary text (clears AAA >= 7:1)
	palettePrimary     = "#CBA6F7" // Mauve: brand accent / focus
	paletteSecondary   = "#94E2D5" // Teal: info / section accent
	paletteSuccess     = "#A6E3A1" // Green: labelled success
	paletteWarning     = "#F9E2AF" // Yellow: labelled warning
	paletteError       = "#F38BA8" // Red: labelled error
	paletteBorder      = "#7F849C" // Overlay1: meaningful borders and dividers (>= 3:1)
	paletteBorderFocus = "#89B4FA" // Blue: focused border indicator (>= 3:1)
	paletteSelectedBg  = "#313244" // Surface0: selected row background tint
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
	Surface:     TokenDef{TrueColor: paletteSurface, ANSI256: "233", ANSI: "0"},
	Text:        TokenDef{TrueColor: paletteForeground, ANSI256: "254", ANSI: "15"},
	Muted:       TokenDef{TrueColor: paletteMuted, ANSI256: "252", ANSI: "7"},
	Dim:         TokenDef{TrueColor: paletteDim, ANSI256: "248", ANSI: "7"},
	Accent:      TokenDef{TrueColor: palettePrimary, ANSI256: "183", ANSI: "13"},
	SelectedBg:  TokenDef{TrueColor: paletteSelectedBg, ANSI256: "236", ANSI: "4"},
	Border:      TokenDef{TrueColor: paletteBorder, ANSI256: "244", ANSI: "8"},
	BorderFocus: TokenDef{TrueColor: paletteBorderFocus, ANSI256: "111", ANSI: "14"},
	Success:     TokenDef{TrueColor: paletteSuccess, ANSI256: "150", ANSI: "10"},
	Warning:     TokenDef{TrueColor: paletteWarning, ANSI256: "223", ANSI: "11"},
	Error:       TokenDef{TrueColor: paletteError, ANSI256: "211", ANSI: "9"},
	Info:        TokenDef{TrueColor: paletteSecondary, ANSI256: "116", ANSI: "14"},
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

	// Profile records which reference color profile resolved the tokens.
	// Production starts at TrueColor and follows Bubble Tea's reported profile;
	// tests may construct ANSI-256/ANSI reference themes directly.
	Profile colorprofile.Profile

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

// referenceProfile normalizes a reported profile to one of the three profiles
// this package resolves explicitly. Unknown and non-color profiles keep the
// TrueColor identity; plain/color-off output is controlled by Color, not here.
func referenceProfile(profile colorprofile.Profile) colorprofile.Profile {
	switch profile {
	case colorprofile.ANSI, colorprofile.ANSI256, colorprofile.TrueColor:
		return profile
	default:
		return colorprofile.TrueColor
	}
}

// NewThemeWithProfile constructs a theme for a given color setting and color profile.
// Under Feature 013 B10, a single dark identity (agentic-dark) is established.
func NewThemeWithProfile(colorEnabled bool, isDark bool, profile colorprofile.Profile) Theme {
	if !colorEnabled {
		return Theme{}
	}

	profile = referenceProfile(profile)
	tokens := darkDefs.resolve(profile)

	body := lipgloss.NewStyle()
	header := lipgloss.NewStyle().Background(tokens.Surface)
	headerText := func(c color.Color) lipgloss.Style {
		return header.Foreground(c)
	}

	return Theme{
		Color:       true,
		IsDark:      true,
		Profile:     profile,
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
		Notice:      body.Foreground(tokens.Text),

		HeaderBar:     header,
		HeaderTitle:   headerText(tokens.Text).Bold(true),
		HeaderAccent:  headerText(tokens.Accent).Bold(true),
		HeaderMuted:   headerText(tokens.Muted),
		HeaderDivider: headerText(tokens.Border),
	}
}

// NewAdaptiveTheme builds a theme with the given color setting using the default TrueColor profile.
// Retained for constructor compatibility; always resolves to the single agentic-dark identity.
func NewAdaptiveTheme(colorEnabled bool, isDark bool) Theme {
	return NewThemeWithProfile(colorEnabled, true, colorprofile.TrueColor)
}

// NewTheme builds the shared style set from the Agentic Dark (Catppuccin Mocha) palette.
// By default, it returns the single tested dark palette when color=true, or plain when color=false.
func NewTheme(colorEnabled bool) Theme {
	return NewAdaptiveTheme(colorEnabled, true)
}

// newThemeForConfig resolves the theme with an explicit reference profile when
// the caller supplied one; the zero/unknown profile keeps production's
// TrueColor identity until Bubble Tea reports the terminal profile.
func newThemeForConfig(cfg TUIConfig) Theme {
	return NewThemeWithProfile(cfg.Color, true, cfg.ColorProfile())
}

// ContrastClaim states only what the selected reference profile actually
// guarantees. It never promises contrast for a user-remapped terminal palette.
func (t Theme) ContrastClaim() string {
	if !t.Color {
		return "plain output; no color contrast is claimed"
	}
	switch t.Profile {
	case colorprofile.ANSI256:
		return "WCAG AAA reference contrast on the documented xterm 256-color palette (user-remapped ANSI palettes are not guaranteed)"
	case colorprofile.ANSI:
		return "best-effort 16-color contrast; the standard ANSI-16 reference palette cannot meet AAA for every accent role (user-remapped palettes are not guaranteed)"
	default:
		return "WCAG AAA reference contrast with TrueColor (user-remapped ANSI palettes are not guaranteed)"
	}
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
