package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// renderViewport reserves chrome before rendering any content. Only already
// wrapped body rows are paged; a path or warning never disappears horizontally.
func (a App) renderViewport() string {
	if a.width <= 0 || a.height <= 0 {
		return ""
	}
	a = a.displayCopy()
	footer := a.viewportLines(a.viewportFooter(), a.width)
	if a.width < 40 || a.height < 8 {
		footer = []string{"q quit"}
		body := a.viewportLines("Terminal too small. Resize to at least 40x8.", a.width)
		return assembleView(nil, body, footer, a.width, a.height, 0)
	}
	// Chrome has a fixed footer budget, even at the smallest usable height.
	footer = []string{a.footerDivider(), a.viewportFooter()}
	if a.width >= 80 {
		footer[1] += fmt.Sprintf(" | m motion:%t u ascii:%t", a.motion, a.ascii)
	}
	footer[1] = a.theme.Hint.Render(footer[1])
	focus := "nav"
	if a.focus == FocusContent {
		focus = "content"
	} else if a.focus == FocusCategories {
		focus = "categories"
	}
	header := a.topBarRows(focus)
	sidebar := a.sidebarVisible()
	if !sidebar && (!a.theme.Color || a.height >= 12) {
		header = append(header, a.compactNavigation())
	}
	if a.activeRoute == RouteExplore && !a.helpOpen && a.height >= 12 {
		header = append(header, a.categoryHeading())
	}
	bodyHeight := a.height - len(header) - len(footer)
	contentWidth := a.width
	sideWidth := 0
	if sidebar {
		sideWidth = 18
		contentWidth -= sideWidth + 1
	}
	// Keep text one cell away from the terminal edge or navigation divider.
	contentWidth--
	var body []string
	if a.activeRoute == RouteExplore && !a.helpOpen {
		if bodyHeight < 6 && !a.catalogModel.InDetail() {
			body = a.viewportLines("Resize taller to browse. Press Enter for details.", contentWidth)
		} else if a.width >= 80 {
			listWidth := 36
			if a.width >= 130 {
				listWidth = 44
			} else if a.width >= 100 {
				listWidth = 40
			}
			detailWidth := contentWidth - listWidth - 2
			a.catalogModel.SetSize(listWidth, bodyHeight)
			listLines := strings.Split(a.catalogModel.ListView(), "\n")
			details := a.viewportLines(a.catalogModel.DetailCard(detailWidth), detailWidth)
			right := pageRows(details, bodyHeight, a.detailScroll)
			for row := 0; row < bodyHeight; row++ {
				left := ""
				if row < len(listLines) {
					// List labels are compact summaries; the full ID is in details.
					left = listLines[row]
					if a.ascii {
						left = toASCII(left)
					}
					left = ansi.Truncate(left, listWidth, "")
				}
				body = append(body, padCells(left, listWidth)+"| "+right[row])
			}
		} else if a.catalogModel.InDetail() {
			details := a.viewportLines(a.catalogModel.DetailCard(contentWidth), contentWidth)
			body = pageRows(details, bodyHeight, a.detailScroll)
		} else {
			a.catalogModel.SetSize(contentWidth, bodyHeight)
			body = strings.Split(a.catalogModel.ListView(), "\n")
			for i, line := range body {
				if a.ascii {
					line = toASCII(line)
				}
				body[i] = ansi.Truncate(line, contentWidth, "")
			}
		}
	} else {
		lines := a.getContentLines(contentWidth)
		offset := a.contentScroll
		if a.helpOpen {
			lines = a.getHelpLines(contentWidth)
			offset = a.helpScroll
		}
		body = pageRows(a.viewportLines(strings.Join(lines, "\n"), contentWidth), bodyHeight, offset)
	}
	for i := range body {
		body[i] = " " + body[i]
	}
	if sidebar {
		nav := a.sidebarNavigation(sideWidth)
		for row := 0; row < bodyHeight; row++ {
			left, right := "", ""
			if row < len(nav) {
				left = nav[row]
			}
			if row < len(body) {
				right = body[row]
			}
			if row >= len(body) {
				body = append(body, "")
			}
			body[row] = padCells(left, sideWidth) + "|" + right
		}
	}
	return assembleView(header, body, footer, a.width, a.height, 0)
}

// headerPadding is the left/right breathing room inside the colored top bar.
const headerPadding = 2

// topBarRows builds the colored top bar. Normal heights get a padded surface
// row, the title/control row and a colored divider; shorter terminals drop
// decoration before controls, and below height 12 a single compact row remains.
// Plain sessions keep the historical one-row unstyled heading so headless
// output is unchanged.
func (a App) topBarRows(focus string) []string {
	if !a.theme.Color {
		// Plain sessions keep the historical one-row unstyled heading.
		return []string{a.theme.Title.Render(a.headerTitleText(focus))}
	}
	title := a.headerTitleRow(focus)
	switch {
	case a.height >= 14:
		return []string{a.headerSurfaceRow(), title, a.headerDividerRow()}
	case a.height >= 12:
		return []string{title, a.headerDividerRow()}
	default:
		return []string{title}
	}
}

// headerTitleText is the legacy title/control text. headerTitleRow may drop
// the attribution or route first when the padded bar cannot fit it.
func (a App) headerTitleText(focus string) string {
	title := BrandName + " / " + a.routes[a.activeRoute].Title + " [" + focus + "]"
	if a.width < 60 {
		title = BrandName + " [" + focus + "]"
	} else if a.width >= 80 {
		title += " by " + BrandAuthor
	}
	return title
}

// topBarTitle fits the title into the padded bar without splitting the textual
// focus indicator: attribution goes first, then the route name.
func (a App) topBarTitle(focus string) string {
	available := a.width - headerPadding*2
	if available < 1 {
		available = 1
	}
	title := a.headerTitleText(focus)
	if ansi.StringWidth(title) <= available {
		return title
	}
	title = strings.TrimSuffix(title, " by "+BrandAuthor)
	if ansi.StringWidth(title) <= available {
		return title
	}
	title = BrandName + " [" + focus + "]"
	if ansi.StringWidth(title) <= available {
		return title
	}
	return ansi.Truncate(title, available, "")
}

// headerTitleRow paints a full-width band, pads the content two cells from
// each edge and keeps the focus indicator textual while coloring it.
func (a App) headerTitleRow(focus string) string {
	title := a.topBarTitle(focus)
	focusText := "[" + focus + "]"
	rendered := a.theme.HeaderTitle.Render(title)
	if idx := strings.Index(title, focusText); idx >= 0 {
		before, after := title[:idx], title[idx+len(focusText):]
		rendered = a.theme.HeaderTitle.Render(before) + a.theme.HeaderAccent.Render(focusText)
		if after != "" {
			rendered += a.theme.HeaderMuted.Render(after)
		}
	}
	rightPad := a.width - headerPadding*2 - ansi.StringWidth(title)
	if rightPad < 0 {
		rightPad = 0
	}
	return a.theme.HeaderBar.Render(strings.Repeat(" ", headerPadding)) +
		rendered +
		a.theme.HeaderBar.Render(strings.Repeat(" ", headerPadding+rightPad))
}

// headerSurfaceRow is the blank colored padding row that gives the title more
// visual space than a bare one-line heading.
func (a App) headerSurfaceRow() string {
	return a.theme.HeaderBar.Render(strings.Repeat(" ", a.width))
}

// headerDividerRow separates the colored bar from the body.
func (a App) headerDividerRow() string {
	rule := "─"
	if a.ascii {
		rule = "-"
	}
	return a.theme.HeaderDivider.Render(strings.Repeat(rule, a.width))
}

// footerDivider keeps the existing plain rule byte-identical and colors it for
// colored sessions.
func (a App) footerDivider() string {
	rule := "-"
	if a.theme.Color && !a.ascii {
		rule = "─"
	}
	return a.theme.Border.Render(strings.Repeat(rule, a.width))
}

// sidebarVisible reports the widths that use the persistent navigation column.
func (a App) sidebarVisible() bool {
	return a.width >= 110 || a.width >= 60 && a.width < 80
}

// headerRowCount is the number of physical header rows the current layout
// renders. The viewport and scroll clamp derive body height from this instead
// of a fixed chrome budget.
func (a App) headerRowCount() int {
	if a.width < 40 || a.height < 8 {
		return 0
	}
	rows := 1
	if a.theme.Color {
		switch {
		case a.height >= 14:
			rows += 2 // surface row + divider
		case a.height >= 12:
			rows++ // divider
		}
	}
	if !a.sidebarVisible() && (!a.theme.Color || a.height >= 12) {
		rows++
	}
	if a.activeRoute == RouteExplore && !a.helpOpen && a.height >= 12 {
		rows++
	}
	return rows
}

func (a App) compactNavigation() string {
	labels := []string{"1 Home", "2 Install", "3 Explore", "4 Help"}
	if a.width >= 80 && !a.ascii {
		labels = []string{"1 🏠 Home", "2 📦 Install", "3 🔍 Explore", "4 💡 Help"}
	}
	for i := range labels {
		if i == a.navIndex {
			labels[i] = ">" + labels[i]
		}
	}
	for i, label := range labels {
		if i == a.navIndex {
			labels[i] = a.theme.NavActive.Render(label)
		} else {
			labels[i] = a.theme.NavInactive.Render(label)
		}
	}
	return strings.Join(labels, "  ")
}

func (a App) sidebarNavigation(width int) []string {
	labels := []string{"1 Home", "2 Install", "3 Explore", "4 Help"}
	for i := range labels {
		if i == a.navIndex {
			labels[i] = a.theme.NavActive.Render("> " + labels[i])
		} else {
			labels[i] = a.theme.NavInactive.Render("  " + labels[i])
		}
	}
	return labels
}

func (a App) viewportFooter() string {
	switch {
	case a.helpOpen:
		return "?/Esc back PgUp/Dn scroll q quit"
	case a.activeRoute == RouteExplore && a.catalogModel.IsFiltering():
		return "Esc cancel Enter search Ctrl+C quit"
	case a.focus == FocusCategories:
		return "j/k category Enter list Tab next q quit"
	case a.focus == FocusNav:
		return "j/k Enter open Tab pane ? help q quit"
	case a.activeRoute == RouteExplore && !a.catalogModel.InDetail():
		return "/ filter c category Enter details q quit"
	case a.activeRoute == RouteHome:
		return "Enter setup PgDn scroll ? help q quit"
	default:
		return "Esc back PgUp/Dn scroll ? help q quit"
	}
}

func quoteHintPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

// Clamp before handling an input, so repeated PageDown at the bottom cannot
// accumulate an invisible offset that takes many PageUp presses to undo.
func (a *App) clampScrollOffsets() {
	display := a.displayCopy()
	footerRows := 2
	if display.height <= 8 {
		footerRows = 1
	}
	rows := display.height - display.headerRowCount() - footerRows
	width := display.width
	if display.sidebarVisible() {
		width -= 19
	}
	width = max(1, width-1)
	rows = max(1, rows)
	content := display.viewportLines(strings.Join(display.getContentLines(width), "\n"), width)
	help := display.viewportLines(strings.Join(display.getHelpLines(width), "\n"), width)
	a.contentScroll = min(a.contentScroll, max(0, len(content)-rows))
	a.helpScroll = min(a.helpScroll, max(0, len(help)-rows))
	if a.width >= 80 {
		width -= 34
	}
	details := display.viewportLines(display.catalogModel.DetailCard(width), width)
	a.detailScroll = min(a.detailScroll, max(0, len(details)-rows))
}

func (a App) categoryHeading() string {
	label, count := "All", len(a.catalog.Items)
	for _, category := range a.catalogModel.Categories() {
		if category.ID == a.catalogModel.ActiveCategory() {
			label, count = category.Label, category.Count
			break
		}
	}
	text := fmt.Sprintf("Category: %s (%d)   [ / ] change", label, count)
	if a.focus == FocusCategories {
		text = "> " + text
	}
	// This is a compact control summary, not the underlying item metadata.
	return a.theme.Section.Render(ansi.Truncate(text, a.width, "..."))
}

func pageRows(lines []string, height, offset int) []string {
	if height <= 0 {
		return nil
	}
	maxOffset := max(0, len(lines)-height)
	offset = max(0, min(offset, maxOffset))
	rows := make([]string, height)
	copy(rows, lines[offset:min(len(lines), offset+height)])
	return rows
}

func (a App) viewportLines(text string, width int) []string {
	if a.ascii {
		text = toASCII(text)
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		lines = append(lines, Paragraph(strings.TrimLeft(line, " "), width, indent)...)
	}
	return lines
}

// Escape dynamic data before decorative ASCII mappings so, for example, a path
// containing a real em dash is not silently renamed into a path with a hyphen.
func (a App) displayCopy() App {
	if !a.ascii {
		return a
	}
	a.catalog.Home = escapeData(a.catalog.Home)
	a.catalog.BackupRoot = escapeData(a.catalog.BackupRoot)
	a.cfg.Home = escapeData(a.cfg.Home)
	a.cfg.Repo = escapeData(a.cfg.Repo)
	a.catalogModel.catalog.Items = append([]CatalogItem(nil), a.catalogModel.catalog.Items...)
	if item, ok := a.catalogModel.Selected(); ok {
		item.ID = escapeData(item.ID)
		item.Name = escapeData(item.Name)
		item.Purpose = escapeData(item.Purpose)
		item.When = escapeData(item.When)
		item.Example = escapeData(item.Example)
		item.Caveat = escapeData(item.Caveat)
		item.Category = escapeData(item.Category)
		item.AgentMode = escapeData(item.AgentMode)
		item.AgentProvider = escapeData(item.AgentProvider)
		item.AgentModel = escapeData(item.AgentModel)
		item.RoutingNote = escapeData(item.RoutingNote)
		item.AgentFallbacks = append([]string(nil), item.AgentFallbacks...)
		for i := range item.AgentFallbacks {
			item.AgentFallbacks[i] = escapeData(item.AgentFallbacks[i])
		}
		item.Clients = append([]ClientStatus(nil), item.Clients...)
		for i := range item.Clients {
			item.Clients[i].Path = escapeData(item.Clients[i].Path)
			item.Clients[i].Client = escapeData(item.Clients[i].Client)
		}
		// The override is presentation-only; never alter list identities/filter state.
		a.catalogModel.displayItem = &item
	}
	return a
}

func escapeData(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r > 0xffff {
			fmt.Fprintf(&b, "\\U%08x", r)
		} else if r > 127 {
			fmt.Fprintf(&b, "\\u%04x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
