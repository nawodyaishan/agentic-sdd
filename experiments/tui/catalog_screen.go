package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	item CatalogItem
}

func (r catalogRow) FilterValue() string { return r.item.FilterValue() }
func (r catalogRow) Title() string       { return FormatCatalogRowTitle(r.item) }
func (r catalogRow) Description() string { return FormatCatalogRowDescription(r.item) }

// CatalogModel is the catalog browser screen.
type CatalogModel struct {
	displayItem *CatalogItem
	// detailOverride is the curated recommendation shown in the detail card
	// independently of the list selection. Escape and opening another entry
	// clear it, so it can never replace a later item's details.
	detailOverride      *CatalogItem
	cfg                 TUIConfig
	theme               Theme
	catalog             Catalog
	list                list.Model
	delegate            list.DefaultDelegate
	detail              bool
	width               int
	height              int
	activeCategory      string
	filterGeneration    int
	pendingSelectedID   string
	pendingSelectedKind string
}

// FilterEnvelope tags an asynchronous filter result with its model generation,
// category ID, and filter query to prevent out-of-order stale overwrites.
type FilterEnvelope struct {
	Generation int
	Category   string
	Query      string
	Msg        tea.Msg
}

// NewCatalogModel builds the browser for one read-only snapshot. cfg is normalized,
// so callers may leave Width and Height zero.
func NewCatalogModel(cfg TUIConfig, catalog Catalog) CatalogModel {
	cfg = cfg.Normalize()
	theme := newThemeForConfig(cfg)

	catalog = cloneCatalog(catalog)

	rows := catalogRows(catalog.Items, CategoryAll)
	delegate := list.NewDefaultDelegate()

	browser := list.New(rows, delegate, cfg.Width, listHeight(cfg.Height))
	browser.Title = "Skills & profiles"
	browser.SetShowStatusBar(true)
	browser.SetShowPagination(false)
	browser.SetShowHelp(true)
	browser.SetFilteringEnabled(true)
	browser.SetStatusBarItemName("entry", "entries")
	browser.DisableQuitKeybindings()

	model := CatalogModel{
		cfg:            cfg,
		theme:          theme,
		catalog:        catalog,
		list:           browser,
		delegate:       delegate,
		width:          cfg.Width,
		height:         cfg.Height,
		activeCategory: CategoryAll,
	}
	model.applyThemeToList()
	return model
}

// cloneCatalog copies the item and client slices so a caller's snapshot is
// never mutated by enrichment or rendering.
func cloneCatalog(catalog Catalog) Catalog {
	clonedItems := make([]CatalogItem, len(catalog.Items))
	for i, it := range catalog.Items {
		itemCopy := it
		if it.Clients != nil {
			itemCopy.Clients = make([]ClientStatus, len(it.Clients))
			copy(itemCopy.Clients, it.Clients)
		}
		enrichItem(&itemCopy)
		clonedItems[i] = itemCopy
	}
	catalog.Items = clonedItems
	return catalog
}

// catalogRows builds the list rows for one category in catalog order.
func catalogRows(items []CatalogItem, category string) []list.Item {
	rows := make([]list.Item, 0, len(items))
	for _, item := range items {
		if itemMatchesCategory(item, category) {
			rows = append(rows, catalogRow{item: item})
		}
	}
	return rows
}

// ReplaceCatalog swaps in a refreshed snapshot without losing the active
// category, filter query or the selected item identity. A selected item that
// no longer exists falls back to the first visible row; no stale row is kept.
func (m *CatalogModel) ReplaceCatalog(catalog Catalog) tea.Cmd {
	prevID, prevKind := "", ""
	if item, ok := m.Selected(); ok {
		prevID, prevKind = item.ID, item.Kind
	}
	category := m.activeCategory
	if category == "" {
		category = CategoryAll
	}
	query := m.list.FilterValue()

	m.catalog = cloneCatalog(catalog)
	m.filterGeneration++
	gen := m.filterGeneration
	rows := catalogRows(m.catalog.Items, category)
	cmd := m.list.SetItems(rows)

	m.pendingSelectedID = prevID
	m.pendingSelectedKind = prevKind
	if query != "" {
		m.list.SetFilterText(query)
	}
	selectIdx := -1
	for i, row := range rows {
		item, ok := row.(catalogRow)
		if !ok {
			continue
		}
		if prevID != "" && item.item.ID == prevID && item.item.Kind == prevKind {
			selectIdx = i
			break
		}
	}
	if selectIdx >= 0 {
		m.list.Select(selectIdx)
	} else if len(rows) > 0 {
		m.list.Select(0)
	} else {
		m.list.ResetSelected()
	}
	return m.tagCmd(cmd, gen)
}

// Init asks for the real terminal size. The alternate screen stays off until a
// View says otherwise, so a resize is safe on any terminal.
func (m CatalogModel) Init() tea.Cmd { return tea.RequestWindowSize }

// Update routes keys: the browser owns navigation, `/` filtering and detail
// opening; `q` and ctrl+c quit from either screen.
func (m CatalogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, listHeight(msg.Height))
		return m, nil
	case tea.KeyPressMsg:
		if cmd, handled := m.handleKey(msg); handled {
			return m, cmd
		}
	case FilterEnvelope:
		if msg.Generation != m.filterGeneration ||
			!strings.EqualFold(msg.Category, m.activeCategory) ||
			msg.Query != m.list.FilterValue() {
			// Out-of-order or stale envelope from another category, generation
			// or query: discard. The query check keeps a programmatic filter
			// change from accepting a result computed for different text.
			return m, nil
		}
		return m.handleFilterResult(msg.Msg)
	case list.FilterMatchesMsg:
		// Only a result that passed through tagCmd's generation and category
		// check may change the visible rows. An untagged result belongs to an
		// older query and is discarded rather than applied over newer state.
		return m, nil
	}

	filterBefore := m.list.FilterValue()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.list.FilterValue() != filterBefore {
		m.filterGeneration++
		cmd = m.tagCmd(cmd, m.filterGeneration)
	}
	return m, cmd
}

func (m CatalogModel) handleFilterResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	// If list is in a category other than "all", verify visible items belong to the active category.
	if m.activeCategory != "" && m.activeCategory != CategoryAll {
		for _, it := range m.VisibleItems() {
			if !itemMatchesCategory(it, m.activeCategory) {
				var matching []list.Item
				for _, item := range m.catalog.Items {
					if itemMatchesCategory(item, m.activeCategory) {
						matching = append(matching, catalogRow{item: item})
					}
				}
				cmd = m.list.SetItems(matching)
				cmd = m.tagCmd(cmd, m.filterGeneration)
				break
			}
		}
	}

	// Reapply pending selection (kind + ID) against visible filtered results.
	if m.pendingSelectedID != "" {
		foundIdx := -1
		for idx, it := range m.VisibleItems() {
			if it.ID == m.pendingSelectedID && it.Kind == m.pendingSelectedKind {
				foundIdx = idx
				break
			}
		}
		if foundIdx >= 0 {
			m.list.Select(foundIdx)
		}
	}

	return m, cmd
}

func (m *CatalogModel) tagCmd(cmd tea.Cmd, gen int) tea.Cmd {
	if cmd == nil {
		return nil
	}
	category := m.activeCategory
	query := m.list.FilterValue()

	var tag func(tea.Cmd) tea.Cmd
	tag = func(nested tea.Cmd) tea.Cmd {
		if nested == nil {
			return nil
		}
		return func() tea.Msg {
			msg := nested()
			if msg == nil {
				return nil
			}
			if batch, ok := msg.(tea.BatchMsg); ok {
				taggedBatch := make(tea.BatchMsg, 0, len(batch))
				for _, sub := range batch {
					if tagged := tag(sub); tagged != nil {
						taggedBatch = append(taggedBatch, tagged)
					}
				}
				return taggedBatch
			}
			// Only a list filter result needs the generation/category check.
			// Other messages pass through untouched so no unrelated command
			// (for example a quit) can be swallowed by the envelope.
			if _, ok := msg.(list.FilterMatchesMsg); !ok {
				return msg
			}
			return FilterEnvelope{
				Generation: gen,
				Category:   category,
				Query:      query,
				Msg:        msg,
			}
		}
	}
	return tag(cmd)
}

// handleKey reports whether the browser consumed the key. Unhandled keys fall
// through to the list, which is what makes `/` filtering and cursor movement
// work without re-implementing them.
func (m *CatalogModel) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	key := msg.String()

	if m.detail {
		switch key {
		case "b", "B", "esc", "backspace":
			m.detail = false
			m.displayItem = nil
			m.detailOverride = nil
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
	case "s", "S":
		if m.OpenRouterDetails() {
			return nil, true
		}
	case "home":
		m.list.Select(0)
		return nil, true
	case "end":
		if n := len(m.list.VisibleItems()); n > 0 {
			m.list.Select(n - 1)
		}
		return nil, true
	case "enter":
		if _, ok := m.Selected(); ok {
			m.detailOverride = nil
			m.detail = true
		}
		return nil, true
	}
	return nil, false
}

// OpenRouterDetails displays the curated SDD Router item in the detail card
// without executing any example or mutating the underlying list cursor. The
// override is presentation-only and is cleared by Esc or by opening another
// entry.
func (m *CatalogModel) OpenRouterDetails() bool {
	for i, it := range m.catalog.Items {
		if it.ID == RouterSkillID {
			itemCopy := m.catalog.Items[i]
			enrichItem(&itemCopy)
			m.detailOverride = &itemCopy
			m.detail = true
			return true
		}
	}
	return false
}

// View renders the current screen. AltScreen is opt-in per run so headless and
// scripted invocations cannot take over the terminal.
func (m CatalogModel) View() tea.View {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("agentic-sdd · skills catalog"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render(fmt.Sprintf(
		"%d skills · %d profiles · read-only",
		m.catalog.SkillCount(), m.catalog.ProfileCount(),
	)))
	b.WriteString("\n")
	b.WriteString(m.theme.Accent.Render(fmt.Sprintf("Tip: %s (%s)", RouterRecommendationText, RouterRecommendationHint)))
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
func (m CatalogModel) detailView() string {
	return m.renderDetailCard(m.width, true)
}

func (m CatalogModel) renderDetailCard(width int, includeFooter bool) string {
	item, ok := m.Selected()
	if m.detailOverride != nil {
		item, ok = *m.detailOverride, true
	}
	if m.displayItem != nil {
		item, ok = *m.displayItem, true
		enrichItem(&item)
	}
	if !ok {
		return m.theme.Hint.Render("No entry selected. Press b to return to the list.")
	}

	maxW := width
	if maxW <= 0 {
		maxW = 40
	}

	var b strings.Builder
	if item.Name != "" && item.Name != item.ID {
		b.WriteString(m.theme.Accent.Render(item.Name))
		b.WriteString("\n")
		b.WriteString(m.theme.Hint.Render("ID: " + item.ID))
		b.WriteString("\n")
	} else {
		b.WriteString(m.theme.Accent.Render(item.ID))
		b.WriteString("\n")
	}

	meta := fmt.Sprintf("%s · %s · %s", item.Kind, item.Origin, item.Group)
	for _, line := range Paragraph(meta, maxW, 0) {
		b.WriteString(m.theme.Subtitle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if item.Kind == skillsync.InventoryKindProfile {
		b.WriteString(m.routingSection(item, maxW))
	}

	b.WriteString(m.theme.Section.Render("Destination status: (Installed state)"))
	b.WriteString("\n")
	if len(item.Clients) == 0 {
		for _, line := range Paragraph("No destination entries reported (no supported clients).", maxW, 2) {
			b.WriteString(m.theme.Hint.Render(line))
			b.WriteString("\n")
		}
	}
	for _, client := range item.Clients {
		statusText := StatusBadge(client.Status)
		clientLabel := fmt.Sprintf("%-16s", client.Client)
		if ansi.StringWidth("  "+clientLabel+" "+statusText) <= maxW {
			b.WriteString("  ")
			b.WriteString(m.theme.Accent.Render(clientLabel))
			b.WriteString(" ")
			b.WriteString(m.badge(client.Status))
			b.WriteString("\n")
		} else if ansi.StringWidth("  "+client.Client+" "+statusText) <= maxW {
			b.WriteString("  ")
			b.WriteString(m.theme.Accent.Render(client.Client))
			b.WriteString(" ")
			b.WriteString(m.badge(client.Status))
			b.WriteString("\n")
		} else {
			b.WriteString("  ")
			b.WriteString(m.theme.Accent.Render(client.Client))
			b.WriteString("\n")
			b.WriteString("    ")
			b.WriteString(m.badge(client.Status))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("Purpose: (Why use it)"))
	b.WriteString("\n")
	purposeText := item.Purpose
	if purposeText == "" {
		purposeText = "Purpose not documented by Agentic SDD."
	}
	for _, line := range Paragraph(purposeText, maxW, 2) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("When: (When to use)"))
	b.WriteString("\n")
	whenText := item.When
	if whenText == "" {
		whenText = "When to use not documented."
	}
	for _, line := range Paragraph(whenText, maxW, 2) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("Example: (Invocation)"))
	b.WriteString("\n")
	exampleText := item.Example
	if exampleText == "" {
		exampleText = "Example prompt not documented."
	}
	for _, line := range Paragraph(exampleText, maxW, 2) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("Caveat: (What it does not do)"))
	b.WriteString("\n")
	caveatText := item.Caveat
	if caveatText == "" {
		caveatText = "No specific caveats documented."
	}
	for _, line := range Paragraph(caveatText, maxW, 2) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("Advanced files: (Destination paths)"))
	b.WriteString("\n")
	hasPaths := false
	for _, client := range item.Clients {
		if client.Path != "" {
			hasPaths = true
			for _, line := range wrapPath("  "+client.Client+": "+client.Path, maxW) {
				b.WriteString(m.theme.Hint.Render(line))
				b.WriteString("\n")
			}
		}
	}
	if !hasPaths {
		for _, line := range Paragraph("No destination file paths recorded.", maxW, 2) {
			b.WriteString(m.theme.Hint.Render(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(m.theme.Section.Render("Next action:"))
	b.WriteString("\n")
	hintLines := wrapPath("  "+item.InstallHint(), maxW)
	for _, line := range hintLines {
		b.WriteString(m.theme.Accent.Render(line))
		b.WriteString("\n")
	}
	for _, line := range Paragraph("Preview only; this browser never installs, replaces or backs up files.", maxW, 2) {
		b.WriteString(m.theme.Hint.Render(line))
		b.WriteString("\n")
	}

	if includeFooter {
		b.WriteString("\n")
		b.WriteString(m.theme.Hint.Render("b / esc back · q quit"))
	}
	return b.String()
}

// routingSection renders the documented model policy of an agent profile. It
// comes after the identity block and before destination status, and every line
// goes through Paragraph so a long model, fallback or note wraps instead of
// being clipped at the pane edge. The section states configuration only: it
// makes no credential, quota, availability or runtime claim.
func (m CatalogModel) routingSection(item CatalogItem, maxW int) string {
	var b strings.Builder
	b.WriteString(m.theme.Section.Render("Agent routing:"))
	b.WriteString("\n")

	labelled := func(label, value string) {
		for _, line := range Paragraph(label+" "+value, maxW, 2) {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	labelled("Mode:", item.AgentMode)
	labelled("Provider:", item.AgentProvider)
	labelled("Primary model:", item.AgentModel)

	if len(item.AgentFallbacks) == 0 {
		labelled("Fallbacks:", "none documented for this profile")
	} else {
		b.WriteString("  Fallbacks:")
		b.WriteString("\n")
		for i, fallback := range item.AgentFallbacks {
			for _, line := range Paragraph(fmt.Sprintf("%d. %s", i+1, fallback), maxW, 4) {
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
	}

	if item.RoutingNote != "" {
		labelled("Routing note:", item.RoutingNote)
	}
	b.WriteString("\n")
	return b.String()
}

// badge styles the shared status vocabulary by outcome.
func (m CatalogModel) badge(status skillsync.InventoryStatus) string {
	text := StatusBadge(status)
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
func (m CatalogModel) Selected() (CatalogItem, bool) {
	row, ok := m.list.SelectedItem().(catalogRow)
	if !ok {
		return CatalogItem{}, false
	}
	return row.item, true
}

// VisibleItems returns the rows the filter currently shows. It is a read-only
// view of list state, useful to a host and to tests.
func (m CatalogModel) VisibleItems() []CatalogItem {
	items := m.list.VisibleItems()
	rows := make([]CatalogItem, 0, len(items))
	for _, item := range items {
		if row, ok := item.(catalogRow); ok {
			rows = append(rows, row.item)
		}
	}
	return rows
}

// InDetail reports whether the detail card is showing.
func (m CatalogModel) InDetail() bool { return m.detail }

// SetDetail sets whether the detail card is showing. Closing the card also
// drops any recommendation override, so a later entry shows its own details.
func (m *CatalogModel) SetDetail(d bool) {
	m.detail = d
	if !d {
		m.displayItem = nil
		m.detailOverride = nil
	}
}

// SetShowHelp controls whether the child list help bar is visible.
func (m *CatalogModel) SetShowHelp(show bool) { m.list.SetShowHelp(show) }

// IsFiltering reports whether the search/filter text input currently has focus.
func (m CatalogModel) IsFiltering() bool { return m.list.SettingFilter() }

// SetSize updates the screen's dimensions.
func (m *CatalogModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, listHeight(height))
}

// SetTheme updates the active styling theme and synchronizes list delegate and filter styles.
func (m *CatalogModel) SetTheme(theme Theme) {
	m.theme = theme
	m.cfg.Color = theme.Color
	m.applyThemeToList()
}

func (m *CatalogModel) applyThemeToList() {
	if !m.theme.Color {
		m.delegate.Styles = list.DefaultItemStyles{
			NormalTitle:   lipgloss.NewStyle().Padding(0, 0, 0, 2),
			NormalDesc:    lipgloss.NewStyle().Padding(0, 0, 0, 2),
			SelectedTitle: lipgloss.NewStyle().Bold(true).Padding(0, 0, 0, 1).Border(lipgloss.Border{Left: ">"}, false, false, false, true),
			SelectedDesc:  lipgloss.NewStyle().Padding(0, 0, 0, 1).Border(lipgloss.Border{Left: ">"}, false, false, false, true),
			DimmedTitle:   lipgloss.NewStyle().Padding(0, 0, 0, 2),
			DimmedDesc:    lipgloss.NewStyle().Padding(0, 0, 0, 2),
			FilterMatch:   lipgloss.NewStyle().Underline(true),
		}
		m.list.Styles = list.Styles{}
		m.list.SetDelegate(m.delegate)
		return
	}

	ds := list.NewDefaultItemStyles(m.theme.IsDark)
	ds.NormalTitle = m.theme.Text.Padding(0, 0, 0, 2)
	ds.NormalDesc = m.theme.Muted.Padding(0, 0, 0, 2)
	ds.SelectedTitle = m.theme.Selected.Bold(true).Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(m.theme.Tokens.BorderFocus)
	ds.SelectedDesc = m.theme.Selected.Foreground(m.theme.Tokens.Muted).Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(m.theme.Tokens.BorderFocus)
	ds.DimmedTitle = m.theme.Dim.Padding(0, 0, 0, 2)
	ds.DimmedDesc = m.theme.Dim.Padding(0, 0, 0, 2)
	ds.FilterMatch = m.theme.Accent.Underline(true)
	m.delegate.Styles = ds
	m.list.SetDelegate(m.delegate)

	ls := list.DefaultStyles(m.theme.IsDark)
	ls.Title = m.theme.Title
	ls.Spinner = m.theme.Accent
	if m.theme.Tokens.Accent != nil {
		ls.Filter.Cursor.Color = m.theme.Tokens.Accent
	}
	ls.Filter.Focused.Prompt = m.theme.Accent
	ls.Filter.Blurred.Prompt = m.theme.Muted
	ls.Filter.Focused.Text = m.theme.Text
	ls.Filter.Blurred.Text = m.theme.Muted
	ls.Filter.Focused.Placeholder = m.theme.Dim
	ls.Filter.Blurred.Placeholder = m.theme.Dim
	ls.DefaultFilterCharacterMatch = m.theme.Accent.Underline(true)
	ls.StatusBar = m.theme.Muted.Padding(0, 0, 1, 2)
	ls.StatusBarFilterCount = m.theme.Dim
	ls.StatusBarActiveFilter = m.theme.Text
	ls.StatusEmpty = m.theme.Muted
	ls.NoItems = m.theme.Muted
	m.list.Styles = ls
}

// ListView renders the list component alone.
func (m CatalogModel) ListView() string { return m.list.View() }

// DetailCard renders the detail card alone formatted for width.
func (m CatalogModel) DetailCard(width int) string { return m.renderDetailCard(width, false) }

// DetailCardLines returns the individual wrapped lines of the detail card.
func (m CatalogModel) DetailCardLines(width int) []string {
	content := m.renderDetailCard(width, false)
	return strings.Split(content, "\n")
}

// SetFilter applies a filter as if it had been typed after `/`. A host can use
// it to preseed a search from an argument; tests use it to assert filtering
// without simulating every keystroke. It bumps the filter generation, so a
// result computed for an earlier query can never be applied over it.
func (m *CatalogModel) SetFilter(text string) {
	m.list.SetFilterText(text)
	m.filterGeneration++
}

// Size reports the viewport the screen last used.
func (m CatalogModel) Size() (width, height int) { return m.width, m.height }

func listHeight(total int) int {
	if h := total - headerLines - footerLines; h > minListHeight {
		return h
	}
	return minListHeight
}

func wrapPath(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	return strings.Split(ansi.Hardwrap(text, maxWidth, true), "\n")
}

// Categories returns defensive copies of the available category options and their item counts.
func (m CatalogModel) Categories() []CatalogCategory {
	counts := map[string]int{
		CategoryAll:      len(m.catalog.Items),
		CategoryCore:     0,
		CategoryPack:     0,
		CategoryProfiles: 0,
		CategoryPresets:  0,
	}

	var purposeGroups []string
	purposeCounts := make(map[string]int)

	for _, it := range m.catalog.Items {
		baseCat := classifyItem(it)
		counts[baseCat]++

		group := strings.TrimSpace(it.Category)
		if group != "" {
			if purposeCounts[group] == 0 {
				purposeGroups = append(purposeGroups, group)
			}
			purposeCounts[group]++
		}
	}

	result := []CatalogCategory{
		{ID: CategoryAll, Label: "All", Count: counts[CategoryAll]},
		{ID: CategoryCore, Label: "Core skills", Count: counts[CategoryCore]},
		{ID: CategoryPack, Label: "OpenCode pack", Count: counts[CategoryPack]},
		{ID: CategoryProfiles, Label: "OpenCode profiles", Count: counts[CategoryProfiles]},
		{ID: CategoryPresets, Label: "Free-first presets", Count: counts[CategoryPresets]},
	}

	for _, g := range purposeGroups {
		result = append(result, CatalogCategory{
			ID:    "purpose:" + g,
			Label: g,
			Count: purposeCounts[g],
		})
	}

	out := make([]CatalogCategory, len(result))
	copy(out, result)
	return out
}

// ActiveCategory returns the active category ID, defaulting to "all".
func (m CatalogModel) ActiveCategory() string {
	if m.activeCategory == "" {
		return CategoryAll
	}
	return m.activeCategory
}

// isCategoryValid reports whether a category ID is a known base or existing purpose category.
func (m CatalogModel) isCategoryValid(id string) bool {
	idLower := strings.ToLower(id)
	switch idLower {
	case CategoryAll, CategoryCore, CategoryPack, CategoryProfiles, CategoryPresets:
		return true
	default:
		if strings.HasPrefix(idLower, "purpose:") {
			group := id[len("purpose:"):]
			if group == "" {
				return false
			}
			for _, it := range m.catalog.Items {
				if strings.EqualFold(it.Category, group) {
					return true
				}
			}
		}
		return false
	}
}

// SetCategory changes the active category filter, rebuilding visible membership while
// preserving stable selection (kind + ID) if present and any active search filter.
func (m *CatalogModel) SetCategory(id string) tea.Cmd {
	if id == "" {
		id = CategoryAll
	}

	if !m.isCategoryValid(id) {
		return nil
	}

	m.filterGeneration++
	gen := m.filterGeneration
	m.activeCategory = id

	var prevSelectedID string
	var prevSelectedKind string
	if item, ok := m.Selected(); ok {
		prevSelectedID = item.ID
		prevSelectedKind = item.Kind
	}
	m.pendingSelectedID = prevSelectedID
	m.pendingSelectedKind = prevSelectedKind

	var matching []list.Item
	selectIdx := -1
	for _, it := range m.catalog.Items {
		if itemMatchesCategory(it, id) {
			if prevSelectedID != "" && it.ID == prevSelectedID && it.Kind == prevSelectedKind {
				selectIdx = len(matching)
			}
			matching = append(matching, catalogRow{item: it})
		}
	}

	cmd := m.list.SetItems(matching)

	if selectIdx >= 0 {
		m.list.Select(selectIdx)
	} else if len(matching) > 0 {
		m.list.Select(0)
	} else {
		m.list.ResetSelected()
	}

	return m.tagCmd(cmd, gen)
}
