package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// GuideAction identifies one destination or explicit link offered by GuideModel.
// Guide is presentation-only: every setup, restore, log, onboarding and update
// action routes into the existing screen or explicit opener without adding any
// write path, preview token, installer logic or network call.
type GuideAction int

const (
	GuideActionInstall GuideAction = iota
	GuideActionRestore
	GuideActionLogs
	GuideActionOnboarding
	GuideActionAboutUpdate
	GuideActionReadme
)

// GuideEntry describes one selectable step in the Guide hub.
type GuideEntry struct {
	Action GuideAction
	Key    string
	Title  string
	Desc   string
}

var defaultGuideEntries = []GuideEntry{
	{
		Action: GuideActionInstall,
		Key:    "i",
		Title:  "Install or update",
		Desc:   "Pick clients & packs, inspect the read-only preview, confirm with default-No.",
	},
	{
		Action: GuideActionRestore,
		Key:    "b",
		Title:  "Restore a backup",
		Desc:   "Browse backups, inspect a read-only restore plan, confirm with default-No.",
	},
	{
		Action: GuideActionLogs,
		Key:    "l",
		Title:  "View latest log",
		Desc:   "Open Activity & Help at Recent Activity for bounded operation logs.",
	},
	{
		Action: GuideActionOnboarding,
		Key:    "o",
		Title:  "Quick start tour",
		Desc:   "Walk through the skippable, read-only tour and copyable first prompt.",
	},
	{
		Action: GuideActionAboutUpdate,
		Key:    "a",
		Title:  "About & update check",
		Desc:   "View version, update-check status, and diagnostics in Activity & Help.",
	},
	{
		Action: GuideActionReadme,
		Key:    "g",
		Title:  "Online guide / README",
		Desc:   "Explicitly open the repository README in your browser.",
	},
}

// GuideModel is the presentation-only Guide destination model.
type GuideModel struct {
	cfg     TUIConfig
	theme   Theme
	width   int
	height  int
	cursor  int
	entries []GuideEntry
}

// NewGuideModel constructs the presentation-only Guide hub.
func NewGuideModel(cfg TUIConfig) GuideModel {
	cfg = cfg.Normalize()
	entries := make([]GuideEntry, len(defaultGuideEntries))
	copy(entries, defaultGuideEntries)
	return GuideModel{
		cfg:     cfg,
		theme:   newThemeForConfig(cfg),
		width:   cfg.Width,
		height:  cfg.Height,
		entries: entries,
	}
}

// Init implements tea.Model. Guide performs no startup I/O or network call.
func (m GuideModel) Init() tea.Cmd { return nil }

// SetSize updates the available cell dimensions.
func (m *GuideModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTheme updates the active styling theme without resetting cursor state.
func (m *GuideModel) SetTheme(theme Theme) {
	m.theme = theme
}

// Entries returns a defensive copy of the Guide action entries.
func (m GuideModel) Entries() []GuideEntry {
	out := make([]GuideEntry, len(m.entries))
	copy(out, m.entries)
	return out
}

// Cursor reports the highlighted action index.
func (m GuideModel) Cursor() int { return m.cursor }

// SelectedEntry returns the currently highlighted Guide action entry.
func (m GuideModel) SelectedEntry() GuideEntry {
	if len(m.entries) == 0 {
		return GuideEntry{}
	}
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return m.entries[0]
	}
	return m.entries[m.cursor]
}

// MoveCursor moves the highlighted action index safely without wrapping.
func (m *GuideModel) MoveCursor(delta int) {
	if len(m.entries) == 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
}

// ActionForKey maps an interactive key in Guide to its target GuideAction.
func (m GuideModel) ActionForKey(key string) (GuideAction, bool) {
	switch key {
	case "i":
		return GuideActionInstall, true
	case "b":
		return GuideActionRestore, true
	case "l":
		return GuideActionLogs, true
	case "o":
		return GuideActionOnboarding, true
	case "a":
		return GuideActionAboutUpdate, true
	case "g":
		return GuideActionReadme, true
	case "enter":
		if len(m.entries) == 0 {
			return 0, false
		}
		return m.SelectedEntry().Action, true
	default:
		return 0, false
	}
}

// Update handles resize and cursor navigation for standalone and embedded use.
func (m GuideModel) Update(msg tea.Msg) (GuideModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.MoveCursor(-1)
		case "down", "j":
			m.MoveCursor(1)
		case "home":
			m.cursor = 0
		case "end":
			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
			}
		}
	}
	return m, nil
}

// View implements tea.Model for standalone rendering.
func (m GuideModel) View() tea.View {
	content := strings.Join(m.Lines(m.width), "\n")
	if !m.cfg.Color {
		content = ansi.Strip(content)
	}
	view := tea.NewView(content)
	view.AltScreen = false
	return view
}

// Lines renders the Guide screen wrapped safely to width terminal cells.
func (m GuideModel) Lines(width int) []string {
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = DefaultWidth
	}

	var lines []string
	for _, l := range Paragraph("Guide", width, 0) {
		lines = append(lines, m.theme.Title.Render(l))
	}
	for _, l := range Paragraph("Step-by-step setup & recovery", width, 0) {
		lines = append(lines, m.theme.Subtitle.Render(l))
	}
	lines = append(lines, "")

	lines = append(lines, Paragraph(m.theme.Section.Render("Guided actions"), width, 0)...)
	lines = append(lines, "")
	glyphs := GlyphSet(m.cfg.ASCII)
	for i, entry := range m.entries {
		marker := glyphs.Gutter
		if i == m.cursor {
			marker = glyphs.Cursor
		}
		head := fmt.Sprintf("%s [%s] %s", marker, entry.Key, entry.Title)
		for _, row := range Paragraph(head, width, 2) {
			if i == m.cursor && m.theme.Color {
				row = m.theme.Selected.Bold(true).Render(row)
			}
			lines = append(lines, row)
		}
		for _, descRow := range Paragraph(entry.Desc, width, 6) {
			lines = append(lines, m.theme.Hint.Render(descRow))
		}
	}
	lines = append(lines, "")

	for _, row := range wrapPath("Online guide / README: "+ReadmeURL, width) {
		lines = append(lines, m.theme.Accent.Render(row))
	}
	lines = append(lines, "")

	lines = append(lines, Paragraph(m.theme.Section.Render("Execution safety"), width, 0)...)
	lines = append(lines, "")
	for _, row := range Paragraph(WritePolicyText, width, 2) {
		lines = append(lines, m.theme.Warn.Render(row))
	}
	lines = append(lines, "")

	for _, row := range Paragraph("j/k select · Enter open · i install · b backups · l logs · o tour · a about · g README · Esc nav", width, 0) {
		lines = append(lines, m.theme.Hint.Render(row))
	}

	if m.cfg.ASCII {
		for i := range lines {
			lines[i] = toASCII(lines[i])
		}
	}
	return lines
}
