// Package screens holds the experimental TUI screens. Only the read-only
// catalog browser exists in batch B2; mutation flows stay in the CLI.
package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/skillsync"
)

// Chrome is the vertical space the screen header and footer take outside the
// list viewport: the title and subtitle above, plus a spacer and hint below.
const (
	headerLines   = 2
	footerLines   = 2
	minListHeight = 6
)

// catalogRow adapts one domain row to the Bubbles default delegate. All
// formatting stays here; CatalogItem only supplies domain values.
type catalogRow struct {
	item tui.CatalogItem
}

func (r catalogRow) FilterValue() string { return r.item.FilterValue() }
func (r catalogRow) Title() string       { return r.item.ID }
func (r catalogRow) Description() string { return r.item.Summary() }

// Model is the catalog browser screen.
type Model struct {
	cfg     tui.TUIConfig
	theme   tui.Theme
	catalog tui.Catalog
	list    list.Model
	detail  bool
	width   int
	height  int
}

// NewCatalog builds the browser for one read-only snapshot. cfg is normalized,
// so callers may leave Width and Height zero.
func NewCatalog(cfg tui.TUIConfig, catalog tui.Catalog) Model {
	cfg = cfg.Normalize()
	theme := tui.NewTheme(cfg.Color)

	rows := make([]list.Item, 0, len(catalog.Items))
	for _, item := range catalog.Items {
		rows = append(rows, catalogRow{item: item})
	}
	delegate := list.NewDefaultDelegate()

	browser := list.New(rows, delegate, cfg.Width, listHeight(cfg.Height))
	browser.Title = "Skills & profiles"
	browser.SetShowStatusBar(true)
	browser.SetShowPagination(false)
	browser.SetShowHelp(true)
	browser.SetFilteringEnabled(true)
	browser.SetStatusBarItemName("entry", "entries")
	browser.DisableQuitKeybindings()

	return Model{
		cfg:     cfg,
		theme:   theme,
		catalog: catalog,
		list:    browser,
		width:   cfg.Width,
		height:  cfg.Height,
	}
}

// Init asks for the real terminal size. The alternate screen stays off until a
// View says otherwise, so a resize is safe on any terminal.
func (m Model) Init() tea.Cmd { return tea.RequestWindowSize }

// Update routes keys: the browser owns navigation, `/` filtering and detail
// opening; `q` and ctrl+c quit from either screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, listHeight(msg.Height))
		return m, nil
	case tea.KeyPressMsg:
		if cmd, handled := m.handleKey(msg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// handleKey reports whether the browser consumed the key. Unhandled keys fall
// through to the list, which is what makes `/` filtering and cursor movement
// work without re-implementing them.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	key := msg.String()

	if m.detail {
		switch key {
		case "b", "B", "esc", "backspace":
			m.detail = false
		case "q", "ctrl+c":
			return tea.Quit, true
		}
		// The detail card is static in this spike: swallow the rest rather than
		// letting list keys mutate a hidden cursor.
		return nil, true
	}

	// While the filter input is focused every printable key, including q, is
	// filter text. Only ctrl+c may still quit.
	if m.list.SettingFilter() {
		if key == "ctrl+c" {
			return tea.Quit, true
		}
		return nil, false
	}

	switch key {
	case "q", "ctrl+c":
		return tea.Quit, true
	case "enter":
		if _, ok := m.Selected(); ok {
			m.detail = true
		}
		return nil, true
	}
	return nil, false
}

// View renders the current screen. AltScreen is opt-in per run so headless and
// scripted invocations cannot take over the terminal.
func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("agentic-sdd · skills catalog"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render(fmt.Sprintf(
		"%d skills · %d profiles · read-only",
		m.catalog.SkillCount(), m.catalog.ProfileCount(),
	)))
	b.WriteString("\n")
	for _, notice := range m.catalog.Notices {
		b.WriteString(m.theme.Warn.Render("note: " + notice))
		b.WriteString("\n")
	}

	if m.detail {
		b.WriteString("\n")
		b.WriteString(m.detailView())
	} else {
		b.WriteString(m.list.View())
		b.WriteString("\n")
		b.WriteString(m.theme.Hint.Render("↑/↓ move · / filter · enter details · q quit"))
	}

	content := b.String()
	if !m.cfg.Color {
		// Bubbles components carry their own colors and Lip Gloss emits SGR
		// unconditionally, so a plain run strips styling once, at the view
		// boundary. ANSI sequences are zero-width, so layout is unchanged.
		content = ansi.Strip(content)
	}
	view := tea.NewView(content)
	view.AltScreen = m.cfg.Interactive
	view.WindowTitle = "agentic-sdd catalog"
	return view
}

// detailView renders the definition-style card for the selected row.
func (m Model) detailView() string {
	item, ok := m.Selected()
	if !ok {
		return m.theme.Hint.Render("No entry selected. Press b to return to the list.")
	}

	var b strings.Builder
	b.WriteString(m.theme.Accent.Render(item.ID))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render(fmt.Sprintf("%s · %s · %s", item.Kind, item.Origin, item.Group)))
	b.WriteString("\n\n")

	b.WriteString(m.theme.Section.Render("Destination status"))
	b.WriteString("\n")
	if len(item.Clients) == 0 {
		b.WriteString(m.theme.Hint.Render("  No destination entries reported."))
		b.WriteString("\n")
	}
	for _, client := range item.Clients {
		b.WriteString("  ")
		b.WriteString(m.theme.Accent.Render(fmt.Sprintf("%-18s", client.Client)))
		b.WriteString(" ")
		b.WriteString(m.badge(client.Status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Section.Render("Next action"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(m.theme.Accent.Render(item.InstallHint()))
	b.WriteString("\n")
	b.WriteString(m.theme.Hint.Render("  Preview only; this browser never installs, replaces or backs up files."))
	b.WriteString("\n\n")
	b.WriteString(m.theme.Hint.Render("b / esc back · q quit"))
	return b.String()
}

// badge styles the shared status vocabulary by outcome.
func (m Model) badge(status skillsync.InventoryStatus) string {
	text := tui.StatusBadge(status)
	switch status {
	case skillsync.InventoryStatusMatchesSource:
		return m.theme.OK.Render(text)
	case skillsync.InventoryStatusDiffersFromSource, skillsync.InventoryStatusMissing:
		return m.theme.Warn.Render(text)
	case skillsync.InventoryStatusUnavailable:
		return m.theme.Bad.Render(text)
	default:
		return m.theme.Hint.Render(text)
	}
}

// Selected returns the currently highlighted catalog row.
func (m Model) Selected() (tui.CatalogItem, bool) {
	row, ok := m.list.SelectedItem().(catalogRow)
	if !ok {
		return tui.CatalogItem{}, false
	}
	return row.item, true
}

// VisibleItems returns the rows the filter currently shows. It is a read-only
// view of list state, useful to a host and to tests.
func (m Model) VisibleItems() []tui.CatalogItem {
	items := m.list.VisibleItems()
	rows := make([]tui.CatalogItem, 0, len(items))
	for _, item := range items {
		if row, ok := item.(catalogRow); ok {
			rows = append(rows, row.item)
		}
	}
	return rows
}

// InDetail reports whether the detail card is showing.
func (m Model) InDetail() bool { return m.detail }

// SetFilter applies a filter as if it had been typed after `/`. A host can use
// it to preseed a search from an argument; tests use it to assert filtering
// without simulating every keystroke.
func (m *Model) SetFilter(text string) { m.list.SetFilterText(text) }

// Size reports the viewport the screen last used.
func (m Model) Size() (width, height int) { return m.width, m.height }

func listHeight(total int) int {
	if h := total - headerLines - footerLines; h > minListHeight {
		return h
	}
	return minListHeight
}
