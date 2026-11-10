package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
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
func (r catalogRow) Title() string {
	if r.item.Name != "" {
		return r.item.Name
	}
	return r.item.ID
}
func (r catalogRow) Description() string { return r.item.Summary() }

// CatalogModel is the catalog browser screen.
type CatalogModel struct {
	displayItem         *CatalogItem
	cfg                 TUIConfig
	theme               Theme
	catalog             Catalog
	list                list.Model
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
	theme := NewTheme(cfg.Color)

	// Clone catalog.Items and each item's Clients slice so caller snapshot is never mutated.
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

	return CatalogModel{
		cfg:            cfg,
		theme:          theme,
		catalog:        catalog,
		list:           browser,
		width:          cfg.Width,
		height:         cfg.Height,
		activeCategory: CategoryAll,
	}
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
		if msg.Generation != m.filterGeneration || !strings.EqualFold(msg.Category, m.activeCategory) {
			// Out-of-order or stale envelope from another category/generation: discard.
			return m, nil
		}
		return m.handleFilterResult(msg.Msg)
	case list.FilterMatchesMsg:
		return m.handleFilterResult(msg)
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
	return func() tea.Msg {
		msg := cmd()
		if msg == nil {
			return nil
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			var taggedBatch tea.BatchMsg
			for _, c := range batch {
				taggedBatch = append(taggedBatch, m.tagCmd(c, gen))
			}
			return taggedBatch
		}
		return tea.BatchMsg{
			func() tea.Msg { return msg },
			func() tea.Msg {
				return FilterEnvelope{
					Generation: gen,
					Category:   category,
					Query:      query,
					Msg:        msg,
				}
			},
		}
	}
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
			m.detail = true
		}
		return nil, true
	}
	return nil, false
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

	b.WriteString(m.theme.Section.Render("Destination status:"))
	b.WriteString("\n")
	if len(item.Clients) == 0 {
		b.WriteString(m.theme.Hint.Render("  No destination entries reported."))
		b.WriteString("\n")
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
		if client.Path != "" {
			pathW := maxW - 4
			if pathW < 16 {
				pathW = 16
			}
			for _, line := range wrapPath(client.Path, pathW) {
				b.WriteString(m.theme.Hint.Render("    " + line))
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n")

	if item.Purpose != "" {
		b.WriteString(m.theme.Section.Render("Purpose:"))
		b.WriteString("\n")
		for _, line := range Paragraph(item.Purpose, maxW, 2) {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if item.When != "" {
		b.WriteString(m.theme.Section.Render("When:"))
		b.WriteString("\n")
		for _, line := range Paragraph(item.When, maxW, 2) {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if item.Example != "" {
		b.WriteString(m.theme.Section.Render("Example:"))
		b.WriteString("\n")
		for _, line := range Paragraph(item.Example, maxW, 2) {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if item.Caveat != "" {
		b.WriteString(m.theme.Section.Render("Caveat:"))
		b.WriteString("\n")
		for _, line := range Paragraph(item.Caveat, maxW, 2) {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

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

// SetDetail sets whether the detail card is showing.
func (m *CatalogModel) SetDetail(d bool) { m.detail = d }

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
// without simulating every keystroke.
func (m *CatalogModel) SetFilter(text string) { m.list.SetFilterText(text) }

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
