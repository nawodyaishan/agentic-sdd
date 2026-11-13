package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// usableGeometry reports whether the terminal is large enough for the
// interactive layouts. Below it the viewport shows only resize guidance, so no
// confirmation may accept an affirmative key.
func (a App) usableGeometry() bool {
	return a.width >= 40 && a.height >= 8
}

// renderViewport reserves chrome before rendering any content. Only already
// wrapped body rows are paged; a path or warning never disappears horizontally.
// Help and the install/restore confirmation screens are composed through the
// single bounded modal compositor (modal.go).
func (a App) renderViewport() string {
	if a.width <= 0 || a.height <= 0 {
		return ""
	}
	a = a.displayCopy()
	if !a.usableGeometry() {
		footer := []string{"q quit"}
		body := a.viewportLines("Terminal too small. Resize to at least 40x8.", a.width)
		return assembleView(nil, body, footer, a.width, a.height, 0)
	}
	if a.modalActive() {
		return a.renderModal()
	}
	return a.renderBackdrop()
}

// renderBackdrop renders the normal route content that sits behind a modal or
// fills the viewport when no modal is active. It never renders help content:
// help is always composed as a bounded modal.
func (a App) renderBackdrop() string {
	return a.renderBackdropMode(true)
}

// renderNeutralBackdrop renders the shell chrome without any route content. A
// confirmation modal uses it so the inactive backdrop never repeats the same
// confirmation under its own overlay or exposes a second active surface.
func (a App) renderNeutralBackdrop() string {
	return a.renderBackdropMode(false)
}

func (a App) renderBackdropMode(showContent bool) string {
	// Chrome has a fixed footer budget, even at the smallest usable height.
	fText := a.viewportFooter()
	if a.ascii {
		fText = toASCII(fText)
	}
	if a.width >= 80 {
		status := a.StatusCluster()
		if a.ascii {
			status = toASCII(status)
		}
		if ansi.StringWidth(fText)+ansi.StringWidth(status)+3 <= a.width {
			fText += " | " + status
		}
	}
	// The divider, the persistent notice block and the key hint, in that
	// order: the notice sits at a stable reserved location instead of covering
	// content, and the hint stays the last row the busy footer replaces.
	footer := []string{a.footerDivider()}
	footer = append(footer, a.statusNoticeRows(a.width)...)
	footer = append(footer, a.theme.Hint.Render(fText))
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
	bodyHeight := a.bodyBudget()
	contentWidth := a.width
	sideWidth := 0
	if sidebar {
		sideWidth = 18
		contentWidth -= sideWidth + 1
	}
	// Keep text one cell away from the terminal edge or navigation divider.
	contentWidth--
	glyphs := GlyphSet(a.ascii)
	var body []string
	if !showContent {
		// Neutral chrome for a confirmation modal: no route content, no
		// second copy of the confirmation and no interactive backdrop surface.
		body = make([]string, bodyHeight)
	} else if a.activeRoute == RouteExplore {
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
				body = append(body, padCells(left, listWidth)+glyphs.DividerV+" "+right[row])
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
		if pinned := a.pinnedContentLines(contentWidth); len(pinned) > 0 {
			// The preview summary is pinned: paging moves only the
			// destination details below it. It bypasses viewportLines, so the
			// session's 7-bit conversion is applied explicitly.
			if a.ascii {
				for i := range pinned {
					pinned[i] = toASCII(pinned[i])
				}
			}
			body = append(body, pinned...)
			remaining := bodyHeight - len(pinned)
			if remaining < 1 {
				remaining = 1
			}
			rest := a.viewportLines(strings.Join(a.pinnedDetailLines(contentWidth), "\n"), contentWidth)
			body = append(body, pageRows(rest, remaining, a.contentScroll)...)
			if len(body) > bodyHeight {
				body = body[:bodyHeight]
			}
		} else {
			lines := a.getContentLines(contentWidth)
			offset := a.contentScroll
			body = pageRows(a.viewportLines(strings.Join(lines, "\n"), contentWidth), bodyHeight, offset)
		}
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
			body[row] = padCells(left, sideWidth) + glyphs.DividerV + right
		}
	}
	return assembleView(header, body, footer, a.width, a.height, 0)
}

// headerPadding is the left/right breathing room inside the colored top bar.
const headerPadding = 2

// footerRowCount reports the footer row budget (1 row for tiny terminals <= 8 height, 2 rows otherwise)
// plus the rows reserved for a persistent status notice.
func (a App) footerRowCount() int {
	rows := 2
	if a.height <= 8 {
		rows = 1
	}
	return rows + len(a.statusNoticeRows(a.width))
}

// statusNoticeRows renders the persistent execution notice as a reserved footer
// block above the key hint. It never truncates: the complete text is wrapped to
// the viewport width, so an error, a refusal, a consent-drift explanation or a
// backup/log reference stays readable instead of scrolling away. It returns nil
// when there is nothing to show, while a write owns the busy footer, on tiny
// terminals, and when the notice would not fit beside at least one body row —
// in those cases getContentLines keeps the historical scrollable body
// placement, which also preserves the complete text.
func (a App) statusNoticeRows(width int) []string {
	if a.notice == "" || a.write != nil || a.height <= 8 || width <= 0 {
		return nil
	}
	text := a.notice
	if a.ascii {
		text = toASCII(text)
	}
	rows := Paragraph("Notice: "+text, width, 0)
	if budget := a.height - a.headerRowCount() - 2; len(rows) > budget {
		return nil
	}
	if a.theme.Color {
		for i, row := range rows {
			rows[i] = a.theme.Notice.Render(row)
		}
	}
	return rows
}

// bodyBudget calculates the shared body height available between header and footer.
func (a App) bodyBudget() int {
	if a.width < 40 || a.height < 8 {
		return max(1, a.height-1)
	}
	h := a.height - a.headerRowCount() - a.footerRowCount()
	if h < 1 {
		return 1
	}
	return h
}

func (a App) routeTitle(r Route) string {
	for _, info := range a.routes {
		if info.Route == r {
			return info.Title
		}
	}
	return "Home"
}

// breadcrumbSegments derives the hierarchical navigation breadcrumb for the current screen and stage.
func (a App) breadcrumbSegments() []string {
	if a.helpOpen {
		return []string{a.routeTitle(a.activeRoute), "Help"}
	}
	switch a.activeRoute {
	case RouteHome:
		if a.onboarding != nil {
			return []string{"Home", "Onboarding"}
		}
		return []string{"Home"}
	case RouteInstall:
		switch a.installSection {
		case installSectionHub:
			return []string{"Install & Manage"}
		case installSectionInstall:
			return []string{"Install & Manage", "Install", a.install.Step().String()}
		case installSectionBackups:
			stage := a.restore.Stage()
			switch stage {
			case RestoreStageList:
				return []string{"Install & Manage", "Backups"}
			case RestoreStageDetails:
				return []string{"Install & Manage", "Backups", "Details"}
			case RestoreStagePreview:
				return []string{"Install & Manage", "Backups", "Preview"}
			case RestoreStageConfirm:
				return []string{"Install & Manage", "Backups", "Confirm"}
			case RestoreStageRevalidating:
				return []string{"Install & Manage", "Backups", "Revalidating"}
			case RestoreStageRunning:
				return []string{"Install & Manage", "Backups", "Restoring"}
			case RestoreStageResult:
				return []string{"Install & Manage", "Backups", "Result"}
			default:
				return []string{"Install & Manage", "Backups"}
			}
		default:
			return []string{"Install & Manage"}
		}
	case RouteExplore:
		if a.catalogModel.InDetail() {
			return []string{"Explore", "Details"}
		}
		if a.catalogModel.ActiveCategory() != "" && a.catalogModel.ActiveCategory() != CategoryAll {
			for _, cat := range a.catalogModel.Categories() {
				if cat.ID == a.catalogModel.ActiveCategory() {
					return []string{"Explore", cat.Label}
				}
			}
		}
		return []string{"Explore"}
	case RouteActivity:
		return []string{"Activity & Help"}
	case RouteGuide:
		if a.onboarding != nil {
			return []string{"Guide", "Onboarding"}
		}
		return []string{"Guide"}
	default:
		return []string{a.routeTitle(a.activeRoute)}
	}
}

// Breadcrumb returns the derived non-interactive breadcrumb string for the current view.
func (a App) Breadcrumb() string {
	glyphs := GlyphSet(a.ascii)
	return strings.Join(a.breadcrumbSegments(), glyphs.BreadcrumbSep)
}

// StatusCluster returns concise active or actionable update notices for the top bar header.
// It never claims connectivity or says an enabled checker is "online".
// Routine idle/current/disabled/unavailable states do not clutter the header;
// complete details are maintained in Diagnostics.
func (a App) StatusCluster() string {
	if a.updateMgr == nil || a.updateMgr.IsDisabled() {
		return ""
	}
	switch a.updateMgr.State() {
	case UpdateStatusChecking:
		if a.activeRoute == RouteHome {
			return "checking for updates"
		}
		return ""
	case UpdateStatusNewer:
		return "update available"
	default:
		return ""
	}
}

// SessionPreferences returns a compact description of session preferences
// (motion, symbol set, and offline mode) for Help and Diagnostics.
func (a App) SessionPreferences() string {
	glyphs := GlyphSet(a.ascii)
	sep := " " + glyphs.Bullet + " "
	var parts []string
	if a.motion {
		parts = append(parts, "motion:on")
	} else {
		parts = append(parts, "motion:off")
	}
	if a.ascii {
		parts = append(parts, "ascii:on")
	} else {
		parts = append(parts, "ascii:off")
	}
	if a.updateMgr != nil && a.updateMgr.IsDisabled() {
		parts = append(parts, "upd:disabled")
	} else {
		parts = append(parts, "upd:enabled")
	}
	return strings.Join(parts, sep)
}

// topBarRows builds the colored top bar. Normal heights get a padded surface
// row, the title/control row and a colored divider; shorter terminals drop
// decoration before controls, and below height 12 a single compact row remains.
// Plain sessions keep the historical one-row unstyled heading so headless
// output is unchanged.
func (a App) topBarRows(focus string) []string {
	if !a.theme.Color {
		title := a.headerTitleText(focus)
		if a.width >= 80 {
			status := a.StatusCluster()
			if ansi.StringWidth(title)+ansi.StringWidth(status)+2 <= a.width {
				pad := a.width - ansi.StringWidth(title) - ansi.StringWidth(status)
				return []string{a.theme.Title.Render(title) + strings.Repeat(" ", pad) + a.theme.Hint.Render(status)}
			}
		}
		return []string{a.theme.Title.Render(title)}
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
	bc := a.Breadcrumb()
	title := BrandName + " / " + bc + " [" + focus + "]"
	if a.width < 60 {
		title = BrandName + " [" + focus + "]"
	} else if a.width >= 80 {
		title += " by " + BrandAuthor
	}
	return title
}

// topBarTitle fits the title into the padded bar without splitting the textual
// focus indicator: attribution goes first, then ancestor breadcrumb segments.
func (a App) topBarTitle(focus string) string {
	available := a.width - headerPadding*2
	if available < 1 {
		available = 1
	}
	segments := a.breadcrumbSegments()
	sep := GlyphSet(a.ascii).BreadcrumbSep
	focusText := " [" + focus + "]"
	authorText := " by " + BrandAuthor

	// 1. Try full: BrandName / Segments... [focus] by BrandAuthor
	full := BrandName + " / " + strings.Join(segments, sep) + focusText
	if a.width >= 80 && ansi.StringWidth(full+authorText) <= available {
		return full + authorText
	}
	if ansi.StringWidth(full) <= available {
		return full
	}

	// 2. Collapse ancestor breadcrumb segments before reducing body
	for len(segments) > 1 {
		segments = segments[1:]
		shortened := BrandName + " / " + strings.Join(segments, sep) + focusText
		if ansi.StringWidth(shortened) <= available {
			return shortened
		}
	}

	// 3. Fallback to BrandName [focus]
	minimal := BrandName + focusText
	if ansi.StringWidth(minimal) <= available {
		return minimal
	}
	return ansi.Truncate(minimal, available, "")
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
	status := ""
	statusWidth := 0
	if a.width >= 80 {
		status = a.StatusCluster()
		statusWidth = ansi.StringWidth(status)
	}
	if statusWidth > 0 && rightPad >= statusWidth+2 {
		padBetween := rightPad - statusWidth
		return a.theme.HeaderBar.Render(strings.Repeat(" ", headerPadding)) +
			rendered +
			a.theme.HeaderBar.Render(strings.Repeat(" ", padBetween)) +
			a.theme.HeaderMuted.Render(status) +
			a.theme.HeaderBar.Render(strings.Repeat(" ", headerPadding))
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
	glyphs := GlyphSet(a.ascii)
	labels := []string{"1 Home", "2 Install", "3 Explore", "4 Help", "5 Guide"}
	rendered := make([]string, len(labels))
	for i, label := range labels {
		if i == a.navIndex {
			pill := "[" + label + "]"
			if a.theme.Color {
				rendered[i] = a.theme.Selected.Bold(true).Render(pill)
			} else {
				rendered[i] = "[" + glyphs.Cursor + label + "]"
			}
		} else {
			pill := " " + label + " "
			if a.theme.Color {
				rendered[i] = a.theme.NavInactive.Render(pill)
			} else {
				rendered[i] = pill
			}
		}
	}
	return strings.Join(rendered, " ")
}

func (a App) sidebarNavigation(width int) []string {
	glyphs := GlyphSet(a.ascii)
	labels := []string{"1 Home", "2 Install", "3 Explore", "4 Help", "5 Guide"}
	rows := make([]string, len(labels))
	for i, label := range labels {
		if i == a.navIndex {
			row := glyphs.Cursor + " " + label
			if a.theme.Color {
				rows[i] = a.theme.Selected.Bold(true).Render(row)
			} else {
				rows[i] = row
			}
		} else {
			row := glyphs.Gutter + " " + label
			if a.theme.Color {
				rows[i] = a.theme.NavInactive.Render(row)
			} else {
				rows[i] = row
			}
		}
	}
	return rows
}

func (a App) viewportFooter() string {
	sep := " · "
	if a.ascii {
		sep = " - "
	}
	if a.width < 60 {
		switch {
		case a.helpOpen:
			return "?/Esc close" + sep + "PgUp/Dn scroll"
		case a.activeRoute == RouteExplore && a.catalogModel.IsFiltering():
			return "Esc cancel" + sep + "Enter search" + sep + "q quit"
		case a.focus == FocusCategories:
			return "j/k select" + sep + "Enter list" + sep + "q quit"
		case a.focus == FocusNav:
			return "j/k select" + sep + "Enter open" + sep + "q quit"
		case a.activeRoute == RouteExplore && !a.catalogModel.InDetail():
			return "/ filter" + sep + "Enter details" + sep + "q quit"
		case a.activeRoute == RouteHome:
			return "Enter setup" + sep + "PgDn scroll" + sep + "q quit"
		default:
			return "Esc back" + sep + "q quit"
		}
	}
	switch {
	case a.helpOpen:
		return "?/Esc close" + sep + "PgUp/Dn scroll"
	case a.activeRoute == RouteExplore && a.catalogModel.IsFiltering():
		return "Esc cancel" + sep + "Enter search" + sep + "Ctrl+C quit"
	case a.focus == FocusCategories:
		return "j/k category" + sep + "Enter list" + sep + "Tab next" + sep + "q quit"
	case a.focus == FocusNav:
		return "j/k select" + sep + "Enter open" + sep + "Tab pane" + sep + "? help" + sep + "q quit"
	case a.activeRoute == RouteExplore && !a.catalogModel.InDetail():
		return "/ filter" + sep + "c category" + sep + "Enter details" + sep + "q quit"
	case a.activeRoute == RouteHome:
		return "Enter setup" + sep + "PgDn scroll" + sep + "? help" + sep + "q quit"
	default:
		return "Esc back" + sep + "PgUp/Dn scroll" + sep + "? help" + sep + "q quit"
	}
}

func quoteHintPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

// Clamp before handling an input, so repeated PageDown at the bottom cannot
// accumulate an invisible offset that takes many PageUp presses to undo.
func (a *App) clampScrollOffsets() {
	if a.clampModalScroll() {
		return
	}
	display := a.displayCopy()
	rows := display.bodyBudget()
	width := display.width
	if display.sidebarVisible() {
		width -= 19
	}
	width = max(1, width-1)
	if pinned := display.pinnedContentLines(width); len(pinned) > 0 {
		rows = max(1, rows-len(pinned))
		content := display.viewportLines(strings.Join(display.pinnedDetailLines(width), "\n"), width)
		a.contentScroll = min(a.contentScroll, max(0, len(content)-rows))
	} else {
		content := display.viewportLines(strings.Join(display.getContentLines(width), "\n"), width)
		a.contentScroll = min(a.contentScroll, max(0, len(content)-rows))
	}
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
// The child screens render their own dynamic paths; the ASCII presentation flag
// is propagated to their copies so their data helpers activate, while their
// authoritative requests keep the original configuration.
func (a App) displayCopy() App {
	if !a.ascii {
		return a
	}
	a.catalog.Home = escapeData(a.catalog.Home)
	a.catalog.BackupRoot = escapeData(a.catalog.BackupRoot)
	a.catalog.Notices = append([]string(nil), a.catalog.Notices...)
	for i := range a.catalog.Notices {
		a.catalog.Notices[i] = escapeData(a.catalog.Notices[i])
	}
	a.cfg.Home = escapeData(a.cfg.Home)
	a.cfg.Repo = escapeData(a.cfg.Repo)
	a.notice = escapeData(a.notice)
	a.inventoryErr = escapeData(a.inventoryErr)
	a.install.cfg.ASCII = true
	a.restore.cfg.ASCII = true
	a.guide.cfg.ASCII = true
	a.catalogModel.cfg.ASCII = true
	a.catalogModel.catalog.Notices = append([]string(nil), a.catalogModel.catalog.Notices...)
	for i := range a.catalogModel.catalog.Notices {
		a.catalogModel.catalog.Notices[i] = escapeData(a.catalogModel.catalog.Notices[i])
	}
	if a.onboarding != nil {
		onboarding := *a.onboarding
		onboarding.cfg.ASCII = true
		a.onboarding = &onboarding
	}
	a.catalogModel.catalog.Items = append([]CatalogItem(nil), a.catalogModel.catalog.Items...)
	// A recommendation override owns the detail card while it is open; the
	// ASCII-safe copy must escape that item, never replace it with the list
	// selection.
	item, ok := a.catalogModel.Selected()
	if a.catalogModel.detailOverride != nil {
		item, ok = *a.catalogModel.detailOverride, true
	}
	if ok {
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
