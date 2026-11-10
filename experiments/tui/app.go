package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/version"
)

// Route is an application destination in the root shell.
type Route int

const (
	RouteHome Route = iota
	RouteInstall
	RouteExplore
	RouteActivity
)

// Focus indicates whether keyboard input targets the navigation sidebar or
// the active content area.
type Focus int

const (
	FocusNav Focus = iota
	FocusContent
	FocusCategories
)

// RouteInfo holds display metadata for one root navigation destination.
type RouteInfo struct {
	Route       Route
	Key         string
	Title       string
	Icon        string
	ASCII       string
	Description string
}

// BrandTickMsg is emitted periodically to advance the branded logo reveal.
type BrandTickMsg struct{}

type appStartMsg struct{}

// App is the root Bubble Tea model for the Feature 013 unified TUI shell.
// It manages navigation across the four primary destinations, responsive
// cell-aware layout breakpoints, session-level motion and symbol settings,
// and delegates to screen components.
type App struct {
	cfg              TUIConfig
	theme            Theme
	catalog          Catalog
	activeRoute      Route
	navIndex         int
	focus            Focus
	brandState       BrandState
	brandTickPending bool
	motion           bool
	ascii            bool
	helpOpen         bool
	contentScroll    int
	routeScroll      [4]int
	detailScroll     int
	helpScroll       int
	catalogModel     CatalogModel
	width            int
	height           int
	routes           []RouteInfo
}

// NewApp constructs the root application shell.
// By default, motion is static unless explicitly requested via cfg.Motion.
// No environment variables are inspected here; presentation flags are resolved
// by the launcher or callers.
func NewApp(cfg TUIConfig, catalog Catalog) App {
	cfg = cfg.Normalize()
	theme := NewTheme(cfg.Color)

	motion := cfg.Motion && cfg.Interactive
	ascii := cfg.ASCII

	brandState := NewBrandState(motion)
	catModel := NewCatalogModel(cfg, catalog)
	catModel.SetShowHelp(false) // Single contextual footer is owned by App

	routes := []RouteInfo{
		{
			Route:       RouteHome,
			Key:         "1",
			Title:       "Home",
			Icon:        "🏠",
			ASCII:       "*",
			Description: "Dashboard & welcome",
		},
		{
			Route:       RouteInstall,
			Key:         "2",
			Title:       "Install & Manage",
			Icon:        "📦",
			ASCII:       "+",
			Description: "Configure & preview",
		},
		{
			Route:       RouteExplore,
			Key:         "3",
			Title:       "Explore",
			Icon:        "🔍",
			ASCII:       "#",
			Description: "Browse skills & profiles",
		},
		{
			Route:       RouteActivity,
			Key:         "4",
			Title:       "Activity & Help",
			Icon:        "💡",
			ASCII:       "?",
			Description: "Guides, logs & reference",
		},
	}

	return App{
		cfg:          cfg,
		theme:        theme,
		catalog:      catalog,
		activeRoute:  RouteHome,
		navIndex:     0,
		focus:        FocusNav,
		brandState:   brandState,
		motion:       motion,
		ascii:        ascii,
		catalogModel: catModel,
		width:        cfg.Width,
		height:       cfg.Height,
		routes:       routes,
	}
}

// ActiveRoute returns the currently displayed destination.
func (a App) ActiveRoute() Route { return a.activeRoute }

// NavIndex returns the highlighted index in the navigation sidebar.
func (a App) NavIndex() int { return a.navIndex }

// Focus returns the current input focus (Nav or Content).
func (a App) Focus() Focus { return a.focus }

// Routes returns the list of supported root destinations.
func (a App) Routes() []RouteInfo { return a.routes }

// Motion reports whether animation is enabled.
func (a App) Motion() bool { return a.motion }

// ASCII reports whether ASCII-only border and symbol rendering is enabled.
func (a App) ASCII() bool { return a.ascii }

// HelpOpen reports whether the contextual help overlay is active.
func (a App) HelpOpen() bool { return a.helpOpen }

// BrandDone reports whether the branding animation has completed or is static.
func (a App) BrandDone() bool { return a.brandState.Done() }

// CatalogSelected returns the item currently selected in Explore.
func (a App) CatalogSelected() (CatalogItem, bool) { return a.catalogModel.Selected() }

// VisibleCatalogItems returns the list rows currently visible.
func (a App) VisibleCatalogItems() []CatalogItem { return a.catalogModel.VisibleItems() }

// IsFiltering reports whether Explore is currently in filter text-input mode.
func (a App) IsFiltering() bool { return a.catalogModel.IsFiltering() }

// Init requests the terminal window size and emits an initial startup message.
func (a App) Init() tea.Cmd {
	return tea.Batch(tea.RequestWindowSize, func() tea.Msg { return appStartMsg{} })
}

func tickBrand() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return BrandTickMsg{}
	})
}

// Update handles incoming messages: keys, window size, and brand animation ticks.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.catalogModel.SetSize(a.contentWidth(), a.contentHeight())
		return a, nil

	case appStartMsg:
		if a.activeRoute == RouteHome && a.motion && !a.brandState.Done() && !a.helpOpen && a.height >= 14 && !a.brandTickPending {
			a.brandTickPending = true
			return a, tickBrand()
		}
		return a, nil

	case BrandTickMsg:
		a.brandTickPending = false
		if a.activeRoute == RouteHome && a.motion && !a.brandState.Done() && !a.helpOpen && a.height >= 14 {
			a.brandState = a.brandState.Advance()
			if !a.brandState.Done() {
				a.brandTickPending = true
				return a, tickBrand()
			}
		}
		return a, nil

	case tea.KeyPressMsg:
		a.clampScrollOffsets()
		return a.handleKey(msg)

	default:
		// Forward background commands (e.g. list filter match, spinner ticks) to catalogModel
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return a, cmd
	}
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Filter text input precedence: Explore's search input must consume all
	// printable characters (including q, j, k, m, u, 1-4).
	if a.activeRoute == RouteExplore && a.catalogModel.IsFiltering() {
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return *a, cmd
	}

	key := msg.String()

	// Any user input immediately settles brand reveal animation.
	if !a.brandState.Done() {
		a.brandState = NewBrandState(false)
		a.brandTickPending = false
	}

	if a.helpOpen {
		switch key {
		case "?", "esc", "enter":
			a.helpOpen = false
			a.helpScroll = 0
			return *a, nil
		case "q", "ctrl+c":
			return *a, tea.Quit
		case "pgdown", "pagedown", "ctrl+d", "down", "j":
			a.helpScroll += 4
			return *a, nil
		case "pgup", "pageup", "ctrl+u", "up", "k":
			a.helpScroll -= 4
			if a.helpScroll < 0 {
				a.helpScroll = 0
			}
			return *a, nil
		}
		return *a, nil
	}

	switch key {
	case "ctrl+c":
		return *a, tea.Quit
	case "q":
		return *a, tea.Quit
	case "?":
		a.helpOpen = true
		a.helpScroll = 0
		return *a, nil
	case "tab":
		if a.activeRoute == RouteExplore && a.focus == FocusNav {
			a.focus = FocusCategories
		} else if a.focus == FocusNav || a.focus == FocusCategories {
			a.focus = FocusContent
		} else {
			a.focus = FocusNav
		}
		return *a, nil
	case "m":
		a.motion = !a.motion
		if !a.motion {
			a.brandState = NewBrandState(false)
			a.brandTickPending = false
		} else if a.activeRoute == RouteHome && !a.brandState.Done() && !a.brandTickPending && a.height >= 14 {
			a.brandTickPending = true
			return *a, tickBrand()
		}
		return *a, nil
	case "u":
		a.ascii = !a.ascii
		return *a, nil
	case "c":
		if a.activeRoute == RouteExplore {
			a.focus = FocusCategories
		}
		return *a, nil
	case "[", "]":
		if a.activeRoute == RouteExplore {
			direction := 1
			if key == "[" {
				direction = -1
			}
			cmd := a.cycleCategory(direction)
			return *a, cmd
		}
	case "1":
		return a.switchRoute(0)
	case "2":
		return a.switchRoute(1)
	case "3":
		return a.switchRoute(2)
	case "4":
		return a.switchRoute(3)
	case "pgdown", "pagedown", "ctrl+d":
		if a.activeRoute == RouteExplore {
			a.detailScroll += 4
		} else {
			a.contentScroll += 4
		}
		return *a, nil
	case "pgup", "pageup", "ctrl+u":
		if a.activeRoute == RouteExplore {
			a.detailScroll -= 4
			if a.detailScroll < 0 {
				a.detailScroll = 0
			}
		} else {
			a.contentScroll -= 4
			if a.contentScroll < 0 {
				a.contentScroll = 0
			}
		}
		return *a, nil
	}

	if a.focus == FocusCategories {
		switch key {
		case "up", "k":
			cmd := a.cycleCategory(-1)
			return *a, cmd
		case "down", "j":
			cmd := a.cycleCategory(1)
			return *a, cmd
		case "enter":
			a.focus = FocusContent
		case "esc":
			a.focus = FocusNav
		}
		return *a, nil
	}
	if a.focus == FocusNav {
		switch key {
		case "up", "k":
			a.navIndex = (a.navIndex - 1 + len(a.routes)) % len(a.routes)
			return *a, nil
		case "down", "j":
			a.navIndex = (a.navIndex + 1) % len(a.routes)
			return *a, nil
		case "home":
			a.navIndex = 0
			return *a, nil
		case "end":
			a.navIndex = len(a.routes) - 1
			return *a, nil
		case "enter":
			return a.switchRoute(a.navIndex)
		}
		return *a, nil
	}

	// Focus is in Content
	switch a.activeRoute {
	case RouteExplore:
		if key == "esc" {
			if a.catalogModel.InDetail() {
				a.catalogModel.SetDetail(false)
				a.detailScroll = 0
				return *a, nil
			}
			a.focus = FocusNav
			return *a, nil
		}
		if a.catalogModel.InDetail() && (key == "j" || key == "down") {
			a.detailScroll++
			return *a, nil
		}
		if a.catalogModel.InDetail() && (key == "k" || key == "up") {
			if a.detailScroll > 0 {
				a.detailScroll--
			}
			return *a, nil
		}
		// Reset detail scroll on list movement
		if key == "j" || key == "k" || key == "up" || key == "down" || key == "home" || key == "end" {
			a.detailScroll = 0
		}
		var cmd tea.Cmd
		updated, cmd := a.catalogModel.Update(msg)
		if cat, ok := updated.(CatalogModel); ok {
			a.catalogModel = cat
		}
		return *a, cmd

	case RouteHome:
		switch key {
		case "enter":
			// Primary action on Home: open Install & Manage
			return a.switchRoute(1)
		case "j", "down":
			a.contentScroll++
			return *a, nil
		case "k", "up":
			if a.contentScroll > 0 {
				a.contentScroll--
			}
			return *a, nil
		case "esc":
			a.focus = FocusNav
			return *a, nil
		}

	case RouteInstall, RouteActivity:
		switch key {
		case "j", "down":
			a.contentScroll++
			return *a, nil
		case "k", "up":
			if a.contentScroll > 0 {
				a.contentScroll--
			}
			return *a, nil
		case "esc":
			a.focus = FocusNav
			return *a, nil
		}
	}

	return *a, nil
}

func (a *App) cycleCategory(direction int) tea.Cmd {
	categories := a.catalogModel.Categories()
	if len(categories) == 0 {
		return nil
	}
	index := 0
	for i, category := range categories {
		if category.ID == a.catalogModel.ActiveCategory() {
			index = i
			break
		}
	}
	index = (index + direction + len(categories)) % len(categories)
	a.detailScroll = 0
	return a.catalogModel.SetCategory(categories[index].ID)
}

func (a *App) switchRoute(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(a.routes) {
		return *a, nil
	}
	a.navIndex = index
	a.routeScroll[a.activeRoute] = a.contentScroll
	a.activeRoute = a.routes[index].Route
	a.focus = FocusContent
	a.contentScroll = a.routeScroll[a.activeRoute]

	var cmd tea.Cmd
	if a.activeRoute == RouteHome && a.motion && !a.brandState.Done() && !a.brandTickPending && a.height >= 14 && !a.helpOpen {
		a.brandTickPending = true
		cmd = tickBrand()
	}
	return *a, cmd
}

func (a App) sidebarWidth() int {
	switch {
	case a.width >= 110:
		return 22
	case a.width >= 80:
		return 18
	case a.width >= 60:
		return 14
	default:
		return a.width
	}
}

func (a App) contentWidth() int {
	if a.width < 60 {
		return a.width
	}
	if a.width >= 80 && a.width <= 109 {
		// Medium layout: compact top navigation gives full horizontal width to content
		return a.width
	}
	w := a.width - a.sidebarWidth() - 1
	if w <= 0 {
		return 40
	}
	return w
}

func (a App) contentHeight() int {
	h := a.height - 3
	if a.width >= 80 && a.width <= 109 {
		h = a.height - 4
	}
	if h <= 4 {
		return 4
	}
	return h
}

// View renders the application view, enforcing exact terminal width and height bounds
// without silently dropping headers or footers.
func (a App) View() tea.View {
	content := a.renderViewport()

	if !a.cfg.Color {
		content = ansi.Strip(content)
	}

	view := tea.NewView(content)
	view.AltScreen = a.cfg.Interactive
	view.WindowTitle = BrandName
	// Bubble Tea v2 paints the full view: the canvas and default prose color
	// come from the theme only for colored sessions. Headless/plain runs leave
	// both nil, so their output and terminal state stay untouched.
	if a.cfg.Color {
		view.BackgroundColor = a.theme.CanvasColor()
		view.ForegroundColor = a.theme.DefaultForeground()
	}
	return view
}

func (a App) getContentLines(width int) []string {
	var b strings.Builder

	switch a.activeRoute {
	case RouteHome:
		// Header
		if a.height >= 14 && width >= 40 {
			brandHeader := a.brandState.Header(width, a.ascii)
			if brandHeader != "" {
				b.WriteString(a.theme.Accent.Render(brandHeader))
				b.WriteString("\n\n")
			}
		} else {
			b.WriteString(a.theme.Title.Render(BrandName))
			if width >= 50 {
				b.WriteString(" · ")
				b.WriteString(a.theme.Subtitle.Render("Build with AI."))
			}
			b.WriteString(" · ")
			b.WriteString(a.theme.Hint.Render("by " + BrandAuthor))
			b.WriteString("\n\n")
		}

		b.WriteString(strings.Join(HomeOverview(a.catalog, width), "\n"))
		b.WriteString("\n\n")

		b.WriteString(a.theme.Section.Render("Resources"))
		b.WriteString("\n")
		guideURL := "https://github.com/nawodyaishan/agentic-sdd#readme"
		for _, line := range wrapPath("  Online guide / README: "+guideURL, width) {
			b.WriteString(line + "\n")
		}
		b.WriteString(fmt.Sprintf("  Author: by %s (https://github.com/nawodyaishan)\n", BrandAuthor))

	case RouteInstall:
		b.WriteString(a.theme.Title.Render("Install & Manage"))
		b.WriteString("\n")
		b.WriteString(a.theme.Subtitle.Render("Client target selection, preview, and backup recovery"))
		b.WriteString("\n\n")

		b.WriteString(a.theme.Section.Render("Read-only management preview"))
		b.WriteString("\n")
		b.WriteString(a.theme.Hint.Render("  Interactive installation, client selection, and restore workflows\n  arrive in B4 and B5. This interface remains read-only."))
		b.WriteString("\n\n")

		repoFlag := ""
		if a.cfg.Repo != "" {
			repoFlag = " --repo " + quoteHintPath(a.cfg.Repo)
		}
		homeFlag := ""
		if a.cfg.Home != "" {
			homeFlag = " --home " + quoteHintPath(a.cfg.Home)
		}

		b.WriteString(a.theme.Section.Render("Preview Actions"))
		b.WriteString("\n")
		for _, line := range wrapPath("  Preview core skills: agentic-sdd preview"+repoFlag+homeFlag, width) {
			b.WriteString(a.theme.Accent.Render(line) + "\n")
		}
		for _, line := range wrapPath("  Preview with OpenCode pack: agentic-sdd preview --with-opencode"+repoFlag+homeFlag, width) {
			b.WriteString(a.theme.Accent.Render(line) + "\n")
		}
		b.WriteString("\n")

		b.WriteString(a.theme.Section.Render("Recovery & Backups"))
		b.WriteString("\n")
		backupRoot := a.catalog.BackupRoot
		if backupRoot == "" {
			backupRoot = "not configured"
		}
		for _, line := range wrapPath("  Backups directory: "+backupRoot, width) {
			b.WriteString(a.theme.Accent.Render(line) + "\n")
		}
		b.WriteString("  Restore preview:\n")
		b.WriteString("    Restore requires a selected backup timestamp (interactive browser in B5).\n")
		for _, line := range wrapPath("    Example (replace BACKUP_ID): agentic-sdd restore preview BACKUP_ID"+repoFlag+homeFlag, width) {
			b.WriteString(a.theme.Accent.Render(line) + "\n")
		}

	case RouteActivity:
		b.WriteString(a.theme.Title.Render("Activity & Help"))
		b.WriteString("\n")
		b.WriteString(a.theme.Subtitle.Render("Version metadata, documentation, and navigation reference"))
		b.WriteString("\n\n")

		b.WriteString(a.theme.Section.Render("Version & Build"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Version: %s\n", version.Version))
		b.WriteString(fmt.Sprintf("  Commit:  %s\n", version.Commit))
		b.WriteString(fmt.Sprintf("  Date:    %s\n", version.Date))
		b.WriteString("\n")

		b.WriteString(a.theme.Section.Render("Online Documentation"))
		b.WriteString("\n")
		for _, line := range wrapPath("  Online guide / README: https://github.com/nawodyaishan/agentic-sdd#readme", width) {
			b.WriteString(a.theme.Accent.Render(line) + "\n")
		}
		b.WriteString(fmt.Sprintf("  Author: by %s (https://github.com/nawodyaishan)\n", BrandAuthor))
		b.WriteString("\n")

		b.WriteString(a.theme.Section.Render("Navigation Reference"))
		b.WriteString("\n")
		b.WriteString("  Tab          Cycle navigation, categories (Explore), and content\n")
		b.WriteString("  ↑/↓/j/k      Move cursor across routes or items\n")
		b.WriteString("  1 - 4        Direct jump to destination\n")
		b.WriteString("  Enter        Open item / activate selection\n")
		b.WriteString("  b / Esc      Back to list / navigation\n")
		b.WriteString("  /            Filter catalog items\n")
		b.WriteString("  c / [ / ]    Focus categories / previous / next category\n")
		b.WriteString("  pgup / pgdn  Scroll content up / down\n")
		b.WriteString("  m            Toggle motion reveal (on/off)\n")
		b.WriteString("  u            Toggle symbols (unicode/ascii)\n")
		b.WriteString("  q            Quit application\n")
		b.WriteString("\n")

		b.WriteString(a.theme.Section.Render("Activity Logs & Update Checks"))
		b.WriteString("\n")
		b.WriteString(a.theme.Hint.Render("  Log browsing and asynchronous update checks are scheduled for B5."))
	}

	return strings.Split(b.String(), "\n")
}

func (a App) getHelpLines(width int) []string {
	var b strings.Builder
	b.WriteString(a.theme.Title.Render("Keyboard & Navigation Reference"))
	b.WriteString("\n")
	b.WriteString(a.theme.Subtitle.Render("Contextual help · press ? or Esc to close"))
	b.WriteString("\n\n")

	b.WriteString(a.theme.Section.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString("  Tab          Cycle navigation, categories (Explore), and content\n")
	b.WriteString("  ↑/↓/j/k      Move selection in navigation or catalog list\n")
	b.WriteString("  1 - 4        Direct jump to destination (1:Home, 2:Install, 3:Explore, 4:Help)\n")
	b.WriteString("  Enter        Open item or activate highlighted route\n")
	b.WriteString("  b / Esc      Back to list or return focus to navigation\n")
	b.WriteString("  Home / End   Jump to top or bottom of navigation / catalog\n")
	b.WriteString("  pgup / pgdn  Scroll content up / down\n")
	b.WriteString("\n")

	b.WriteString(a.theme.Section.Render("Catalog & Search"))
	b.WriteString("\n")
	b.WriteString("  /            Filter catalog items by ID, kind, origin, or client\n")
	b.WriteString("  c            Focus the category control\n")
	b.WriteString("  [ / ]        Previous / next kind or purpose category\n")
	b.WriteString("  Esc          Clear filter or exit filter mode\n")
	b.WriteString("\n")

	b.WriteString(a.theme.Section.Render("Session Settings"))
	b.WriteString("\n")
	b.WriteString("  m            Toggle logo reveal motion animation (on/off)\n")
	b.WriteString("  u            Toggle symbol set (Unicode icons vs 7-bit ASCII)\n")
	b.WriteString("  ?            Toggle this help screen\n")
	b.WriteString("  q / Ctrl+C   Quit application\n")
	b.WriteString("\n")
	b.WriteString(a.theme.Hint.Render("Press ? or Esc to return to the previous screen."))

	return strings.Split(b.String(), "\n")
}

// assembleView constructs the final viewport string with guaranteed exact height lines
// and width bounds, keeping the header at the top, scrolling the body in the middle,
// and preserving the footer at the very bottom.
func assembleView(header, body, footer []string, width, height, scroll int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	footerCount := len(footer)
	if height <= 8 && footerCount > 1 {
		footerCount = 1
		footer = footer[len(footer)-1:]
	}

	headerCount := len(header)
	if headerCount+footerCount >= height {
		headerCount = height - footerCount - 1
		if headerCount < 0 {
			headerCount = 0
		}
		header = header[:headerCount]
	}

	bodyHeight := height - headerCount - footerCount
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	maxScroll := len(body) - bodyHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	visibleBody := make([]string, 0, bodyHeight)
	if len(body) > 0 && scroll < len(body) {
		end := scroll + bodyHeight
		if end > len(body) {
			end = len(body)
		}
		visibleBody = append(visibleBody, body[scroll:end]...)
	}
	for len(visibleBody) < bodyHeight {
		visibleBody = append(visibleBody, "")
	}

	allLines := make([]string, 0, height)
	allLines = append(allLines, header...)
	allLines = append(allLines, visibleBody...)
	allLines = append(allLines, footer...)

	for i, line := range allLines {
		if ansi.StringWidth(line) > width {
			allLines[i] = ansi.Truncate(line, width, "")
		}
	}

	if len(allLines) > height {
		allLines = allLines[:height]
	}
	return strings.Join(allLines, "\n")
}

// toASCII maps Unicode box-drawing, arrows, bullets and symbols to 7-bit ASCII equivalents,
// and reversibly escapes unknown Unicode code points as \uXXXX or \UXXXXXXXX without deleting glyphs.
func toASCII(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '•':
			b.WriteRune('*')
		case '·':
			b.WriteRune('-')
		case '►':
			b.WriteRune('>')
		case '◄':
			b.WriteRune('<')
		case '▲':
			b.WriteRune('^')
		case '▼':
			b.WriteRune('v')
		case '─':
			b.WriteRune('-')
		case '│':
			b.WriteRune('|')
		case '╭', '╮', '╯', '╰', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼':
			b.WriteRune('+')
		case '…':
			b.WriteString("...")
		case '‘', '’':
			b.WriteRune('\'')
		case '“', '”':
			b.WriteRune('"')
		case '–', '—':
			b.WriteRune('-')
		default:
			if r <= 127 {
				b.WriteRune(r)
			} else if r <= 0xFFFF {
				fmt.Fprintf(&b, "\\u%04x", r)
			} else {
				fmt.Fprintf(&b, "\\U%08x", r)
			}
		}
	}
	return b.String()
}
