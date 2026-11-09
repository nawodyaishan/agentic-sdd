package tui

import (
	"charm.land/lipgloss/v2"
)

// Theme holds the Lip Gloss styles the screens share. A theme built with
// color=false is entirely plain: every style is the zero style, whose Render
// returns its input unchanged. That keeps headless output and tests
// deterministic regardless of the terminal's color profile.
type Theme struct {
	Color bool

	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Section  lipgloss.Style
	Accent   lipgloss.Style
	OK       lipgloss.Style
	Warn     lipgloss.Style
	Bad      lipgloss.Style
	Hint     lipgloss.Style
}

// NewTheme builds the shared style set. Brand colors match the CLI's Feature
// 011 palette closely enough to feel related without committing the
// experimental spike to the branded tokens.
func NewTheme(color bool) Theme {
	if !color {
		return Theme{}
	}
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	return Theme{
		Color:    true,
		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")),
		Subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("#7C7C8A")),
		Section:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC")),
		Accent:   accent,
		OK:       lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80")),
		Warn:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")),
		Bad:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")),
		Hint:     lipgloss.NewStyle().Foreground(lipgloss.Color("#7C7C8A")),
	}
}
