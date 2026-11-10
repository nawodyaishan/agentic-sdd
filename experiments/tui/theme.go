package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Midnight Lagoon palette. The owner-requested high-contrast shell uses this
// researched, OKLab-balanced set; every text role sits on #0B1C22 (body) or
// #123039 (header band) and clears WCAG AAA, while borders clear the 3:1
// non-text threshold. Keep these tokens centralized: screens must not invent
// role colors locally.
const (
	paletteBackground = "#0B1C22" // full-view canvas
	paletteSurface    = "#123039" // top-bar band / raised surface
	paletteForeground = "#FAF5E9" // body and header primary text
	paletteMuted      = "#B6CACB" // secondary text
	palettePrimary    = "#F39FEC" // violet accent / focus
	paletteSecondary  = "#35DADA" // cyan section accent
	paletteSuccess    = "#84D987" // labelled success
	paletteWarning    = "#EAB946" // labelled warning
	paletteError      = "#FBA7A0" // labelled error
	paletteBorder     = "#8FA9AD" // dividers and borders
)

// Theme holds the Lip Gloss styles the screens share. A theme built with
// color=false is entirely plain: every style is the zero style, whose Render
// returns its input unchanged. That keeps headless output and tests
// deterministic regardless of the terminal's color profile.
type Theme struct {
	Color bool

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

	// Header-bar roles paint their own surface background so composing them
	// preserves the band instead of resetting to the terminal default.
	HeaderBar     lipgloss.Style
	HeaderTitle   lipgloss.Style
	HeaderAccent  lipgloss.Style
	HeaderMuted   lipgloss.Style
	HeaderDivider lipgloss.Style
}

// NewTheme builds the shared style set from the Midnight Lagoon palette.
func NewTheme(color bool) Theme {
	if !color {
		return Theme{}
	}

	body := lipgloss.NewStyle()
	header := lipgloss.NewStyle().Background(lipgloss.Color(paletteSurface))
	headerText := func(c string) lipgloss.Style {
		return header.Foreground(lipgloss.Color(c))
	}

	return Theme{
		Color:       true,
		Title:       body.Bold(true).Foreground(lipgloss.Color(palettePrimary)),
		Subtitle:    body.Foreground(lipgloss.Color(paletteMuted)),
		Section:     body.Bold(true).Foreground(lipgloss.Color(paletteSecondary)),
		Accent:      body.Foreground(lipgloss.Color(palettePrimary)),
		OK:          body.Foreground(lipgloss.Color(paletteSuccess)),
		Warn:        body.Foreground(lipgloss.Color(paletteWarning)),
		Bad:         body.Foreground(lipgloss.Color(paletteError)),
		Hint:        body.Foreground(lipgloss.Color(paletteMuted)),
		NavActive:   body.Bold(true).Foreground(lipgloss.Color(palettePrimary)),
		NavInactive: body.Foreground(lipgloss.Color(paletteForeground)),
		Border:      body.Foreground(lipgloss.Color(paletteBorder)),

		HeaderBar:     header,
		HeaderTitle:   headerText(paletteForeground).Bold(true),
		HeaderAccent:  headerText(palettePrimary).Bold(true),
		HeaderMuted:   headerText(paletteMuted),
		HeaderDivider: headerText(paletteBorder),
	}
}

// CanvasColor is the full-view terminal background for a colored session, or
// nil when color is off. The root View applies it through the Bubble Tea v2
// tea.View.BackgroundColor field.
func (t Theme) CanvasColor() color.Color {
	if !t.Color {
		return nil
	}
	return lipgloss.Color(paletteBackground)
}

// DefaultForeground is the terminal foreground for unstyled body text, or nil
// when color is off. Styled roles override it; raw prose still needs AAA
// contrast against the canvas.
func (t Theme) DefaultForeground() color.Color {
	if !t.Color {
		return nil
	}
	return lipgloss.Color(paletteForeground)
}
