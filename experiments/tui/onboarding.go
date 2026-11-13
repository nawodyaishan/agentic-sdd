package tui

import (
	"fmt"
	"sort"
	"strings"

	"agentic-sdd/internal/skillsync"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// FirstRouterPrompt is the copyable first prompt the shared onboarding tour
// ends with. It is the same everyday prompt README documents for starting or
// planning a feature. The tour never copies it automatically and never runs an
// agent: the user selects and copies the visible text.
const FirstRouterPrompt = "Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for my combined approval. Stop before implementing."

// QuickStartName is the user-facing name of the optional, in-session tour. It
// is a presentation label: the model type keeps its historical name so the
// existing host wiring stays stable.
const QuickStartName = "Quick start"

// PostSetupSectionTitle heads the copyable first-prompt guidance offered after
// a successful setup.
const PostSetupSectionTitle = "First prompt & next steps"

// FleetHint is the optional, skippable pointer to Fleet Manager shown on the
// tour's welcome step. It adds no state: the tour still writes nothing, and
// skipping or cancelling leaves it unread. It names no path and makes no
// deployment, authentication or entitlement claim.
const FleetHint = "Optional later: Fleet Manager (Install & Manage, key f) shows the configured agent models. Deployment and restore are preview only in this build, and nothing is written from here."

// MCPHint is the optional, skippable pointer to MCP Tools shown on the
// tour's welcome step. It adds no state: the tour still writes nothing, and
// skipping or cancelling leaves it unread. It names no path and makes no
// network, credential or tool execution claim.
const MCPHint = "Optional later: MCP Tools (Install & Manage, key t) inspects CodeGraph, Context7 and Exa. Configuration preview and restore are offline and previewed before any write."

// PostSetupBody returns the exact, copyable first Router prompt and the plain
// next steps offered after a successful setup. The caller renders the lines;
// nothing is copied, opened or executed, and no client or agent is contacted.
func PostSetupBody() []string {
	return []string{
		FirstRouterPrompt,
		"",
		"Next steps:",
		"1. Paste the prompt into the agent client of your choice when you are ready.",
		"2. Inspect installed skills again any time with `agentic-sdd preview` (read-only).",
		"3. Re-open this " + QuickStartName + " from Home with o; nothing is saved between sessions.",
		"",
		"Nothing is copied automatically, and no agent, client, clipboard or browser is started.",
	}
}

// OnboardingStep is one screen of the shared, skippable tour.
type OnboardingStep int

const (
	// OnboardingWelcome introduces the product and the read-only tour.
	OnboardingWelcome OnboardingStep = iota
	// OnboardingInstalled explains what an installation would place on disk.
	OnboardingInstalled
	// OnboardingReview summarizes the selected clients and source.
	OnboardingReview
	// OnboardingFinish shows the first router prompt and trusted links.
	OnboardingFinish
)

// String is the human-readable step title used in the tour header.
func (s OnboardingStep) String() string {
	switch s {
	case OnboardingWelcome:
		return "Welcome"
	case OnboardingInstalled:
		return "What gets installed"
	case OnboardingReview:
		return "Client and source review"
	case OnboardingFinish:
		return "Finish"
	default:
		return "Unknown"
	}
}

// OnboardingOutcome is the explicit result of the tour. Completed means the
// user reached the finish step; Skipped and Cancelled are terminal exits that
// make no changes. Done reports those terminal exits, while Finished reports
// the completed finish step so a host can show the prompt before closing.
type OnboardingOutcome int

const (
	// OnboardingActive is the welcome, installed or review step.
	OnboardingActive OnboardingOutcome = iota
	// OnboardingCompleted means the finish step is showing.
	OnboardingCompleted
	// OnboardingSkipped means the user chose to skip the tour.
	OnboardingSkipped
	// OnboardingCancelled means the user left the tour.
	OnboardingCancelled
)

// Terminal reports whether the outcome is a terminal exit (skip or cancel).
func (o OnboardingOutcome) Terminal() bool {
	return o == OnboardingSkipped || o == OnboardingCancelled
}

// String is the explicit outcome label used in the terminal view.
func (o OnboardingOutcome) String() string {
	switch o {
	case OnboardingActive:
		return "active"
	case OnboardingCompleted:
		return "completed"
	case OnboardingSkipped:
		return "skipped"
	case OnboardingCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// OnboardingSummary is the read-only review data the tour shows. It is plain
// data: the host may build it from the catalog, from a preview selection or
// from its own local snapshot. Empty strings mean "not reported", never a
// guessed path.
type OnboardingSummary struct {
	Clients           []string
	Home              string
	Repo              string
	SkillCount        int
	ProfileCount      int
	WithOpenCode      bool
	OpenCodeFreeFirst bool
}

// NewOnboardingSummary derives the review summary from a read-only catalog. It
// reports only what the snapshot contains: unique sorted core-skill clients,
// kind counts and optional-content presence. It never inspects the filesystem,
// network, provider or authentication state.
func NewOnboardingSummary(catalog Catalog) OnboardingSummary {
	summary := OnboardingSummary{
		Home:         catalog.Home,
		Repo:         catalog.Repo,
		SkillCount:   catalog.SkillCount(),
		ProfileCount: catalog.ProfileCount(),
	}

	seen := make(map[string]bool)
	for _, item := range catalog.Items {
		switch item.Origin {
		case skillsync.InventoryOriginPack:
			summary.WithOpenCode = true
		case skillsync.InventoryOriginPreset:
			summary.OpenCodeFreeFirst = true
		}
		if item.Kind != skillsync.InventoryKindSkill || item.Origin != skillsync.InventoryOriginCore {
			continue
		}
		for _, client := range item.Clients {
			name := strings.TrimSpace(client.Client)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			summary.Clients = append(summary.Clients, name)
		}
	}
	sort.Strings(summary.Clients)
	return summary
}

// NewOnboardingSummaryFromSetup derives the tour's review summary from the
// read-only catalog plus the current in-session setup choices. It copies the
// selection so later installer edits cannot mutate the tour, reads no files or
// network, and never creates a preview token or consent.
func NewOnboardingSummaryFromSetup(catalog Catalog, install InstallModel) OnboardingSummary {
	summary := NewOnboardingSummary(catalog)
	summary.Clients = append([]string(nil), install.SelectedClients()...)
	summary.WithOpenCode = install.WithOpenCode()
	summary.OpenCodeFreeFirst = install.OpenCodeFreeFirst()
	return summary
}

// OnboardingModel is the shared, skippable onboarding tour. It is a pure
// Bubble Tea model: it holds no opener, clipboard, filesystem or process
// handle, and rendering or initializing it has no side effect. An explicit o/a
// key press returns an OpenLinkMsg for the host to act on; the tour itself
// never launches a browser.
type OnboardingModel struct {
	cfg     TUIConfig
	theme   Theme
	summary OnboardingSummary
	step    OnboardingStep
	outcome OnboardingOutcome
	width   int
	height  int
	scroll  int
}

// OpenLinkMsg asks the host to open one trusted link after an explicit user
// action. The onboarding model never opens it itself, so the host can keep the
// URL visible and show the opener's fallback explanation.
type OpenLinkMsg struct {
	Link Link
}

// NewOnboardingModel builds the tour for one read-only summary. cfg is
// normalized, so callers may leave Width and Height zero. Color=false produces
// plain, unstyled output; ASCII=true disables terminal hyperlinks.
func NewOnboardingModel(cfg TUIConfig, summary OnboardingSummary) OnboardingModel {
	cfg = cfg.Normalize()
	return OnboardingModel{
		cfg:     cfg,
		theme:   newThemeForConfig(cfg),
		summary: summary,
		step:    OnboardingWelcome,
		outcome: OnboardingActive,
		width:   cfg.Width,
		height:  cfg.Height,
	}
}

// Init does nothing: onboarding never opens a browser, copies text or runs an
// agent at startup.
func (m OnboardingModel) Init() tea.Cmd { return nil }

// Update handles window size, navigation, scrolling and explicit link keys.
// Navigation and scrolling never write or launch anything.
func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m OnboardingModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// A skipped or cancelled tour is terminal: later keys cannot resume it.
	if m.outcome.Terminal() {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.outcome = OnboardingCancelled
		m.scroll = 0
		return m, nil
	case "s":
		m.outcome = OnboardingSkipped
		m.scroll = 0
		return m, nil
	case "o":
		return m, openLinkCmd(LinkReadme)
	case "a":
		return m, openLinkCmd(LinkAuthor)
	case "enter", "right", "l", "tab":
		return m.nextStep(), nil
	case "left", "h", "backspace", "b":
		return m.previousStep(), nil
	case "j", "down":
		m.scroll++
		return m, nil
	case "k", "up":
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	case "pgdown", "pagedown", "ctrl+d":
		m.scroll += 4
		return m, nil
	case "pgup", "pageup", "ctrl+u":
		m.scroll -= 4
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	}
	return m, nil
}

func (m OnboardingModel) nextStep() OnboardingModel {
	switch m.step {
	case OnboardingWelcome:
		m.step = OnboardingInstalled
	case OnboardingInstalled:
		m.step = OnboardingReview
	case OnboardingReview:
		m.step = OnboardingFinish
		m.outcome = OnboardingCompleted
	case OnboardingFinish:
		// Already at the explicit finish step; Enter is a safe no-op.
	}
	m.scroll = 0
	return m
}

func (m OnboardingModel) previousStep() OnboardingModel {
	switch m.step {
	case OnboardingInstalled:
		m.step = OnboardingWelcome
	case OnboardingReview:
		m.step = OnboardingInstalled
	case OnboardingFinish:
		m.step = OnboardingReview
		m.outcome = OnboardingActive
	case OnboardingWelcome:
		// Safe no-op: the tour never wraps past its first step.
	}
	m.scroll = 0
	return m
}

func openLinkCmd(link Link) tea.Cmd {
	return func() tea.Msg { return OpenLinkMsg{Link: link} }
}

// Step reports the current step.
func (m OnboardingModel) Step() OnboardingStep { return m.step }

// Outcome reports the explicit tour result.
func (m OnboardingModel) Outcome() OnboardingOutcome { return m.outcome }

// Finished reports whether the tour reached the finish step.
func (m OnboardingModel) Finished() bool { return m.outcome == OnboardingCompleted }

// Done reports whether the tour ended through skip or cancel and the host may
// close it. The finish step is not done so its prompt stays visible.
func (m OnboardingModel) Done() bool { return m.outcome.Terminal() }

// Reopen returns the tour at its first step with the same summary, theme and
// size. Re-opening is an in-session action: it never changes setup choices,
// revives a preview token or creates consent, and it never writes a first-run
// preference.
func (m OnboardingModel) Reopen() OnboardingModel {
	m.step = OnboardingWelcome
	m.outcome = OnboardingActive
	m.scroll = 0
	return m
}

// Prompt returns the exact first router prompt shown on the finish step.
func (m OnboardingModel) Prompt() string { return FirstRouterPrompt }

// Summary returns the read-only review data.
func (m OnboardingModel) Summary() OnboardingSummary { return m.summary }

// Scroll reports the current vertical scroll offset.
func (m OnboardingModel) Scroll() int { return m.scroll }

// SetSize updates the tour viewport.
func (m *OnboardingModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTheme updates the active styling theme without resetting state.
func (m *OnboardingModel) SetTheme(theme Theme) {
	m.theme = theme
}

// Lines renders the whole tour at width terminal cells, without applying the
// viewport height. Every produced line fits width (or width is non-positive and
// no content is rendered), so a host viewport never has to truncate prose. A
// non-positive width falls back to the last window size; if that is also
// non-positive the result is empty rather than an unbounded line.
func (m OnboardingModel) Lines(width int) []string {
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		return nil
	}

	theme := m.theme
	lines := make([]string, 0, 48)
	lines = append(lines, styledParagraph(BrandName, width, 0, theme.Title.Render)...)
	lines = append(lines, styledParagraph(BrandTagline, width, 0, theme.Subtitle.Render)...)
	lines = append(lines, styledParagraph("by "+BrandAuthor, width, 0, theme.Hint.Render)...)
	lines = append(lines, "")
	lines = append(lines, styledParagraph(fmt.Sprintf(
		"%s · %s | step %d of 4 | optional, read-only and skippable",
		QuickStartName, m.step.String(), int(m.step)+1,
	), width, 0, theme.Subtitle.Render)...)
	lines = append(lines, "")

	if m.outcome.Terminal() {
		title := QuickStartName + " skipped"
		if m.outcome == OnboardingCancelled {
			title = QuickStartName + " cancelled"
		}
		lines = append(lines, m.section(title,
			"No files were written, no browser opened, nothing was copied and no agent ran. Return to the main menu when you are ready.", width)...)
		lines = append(lines, "")
		lines = append(lines, m.hintLines(width)...)
		return lines
	}

	switch m.step {
	case OnboardingWelcome:
		lines = append(lines, m.section("Welcome",
			"This optional "+QuickStartName+" explains what Agentic-SDD installs and how to start.\n"+
				"It is read-only: no files are written, no browser opens and no agent runs.\n"+
				"You can skip at any time, go back without changing anything, or re-open it later from Home with o.", width)...)
		lines = append(lines, "")
		lines = append(lines, m.section("What this tour covers",
			"1. What gets installed\n"+
				"2. A client and source review\n"+
				"3. Your first agentic-sdd-router prompt", width)...)
		lines = append(lines, "")
		lines = append(lines, m.section("Fleet models (optional)", FleetHint, width)...)
		lines = append(lines, "")
		lines = append(lines, m.section("MCP tools (optional)", MCPHint, width)...)

	case OnboardingInstalled:
		lines = append(lines, m.section("What gets installed",
			m.installedBody(), width)...)
		lines = append(lines, "")
		lines = append(lines, m.section("Safety",
			"Nothing is written during this tour. Install & Manage previews every destination and requires an explicit confirmation before it changes files.\n"+
				"Existing skills are backed up before replacement; unrelated files are left untouched.", width)...)

	case OnboardingReview:
		lines = append(lines, m.section("Client and source review",
			m.reviewBody(), width)...)
		lines = append(lines, "")
		lines = append(lines, m.section("Read-only",
			"This summary comes from the local read-only inventory. It makes no network, provider or authentication claims, and no writes, backups or restores happen here.", width)...)

	case OnboardingFinish:
		lines = append(lines, m.section("You're ready",
			QuickStartName+" complete. Copy the first prompt below into your agent of choice.\n"+
				"Nothing is copied automatically and no agent is started. Select the text and copy it yourself.\n"+
				"Next steps: inspect installed skills with `agentic-sdd preview`, then run `agentic-sdd-router` when you are ready.\n"+
				"Re-open this "+QuickStartName+" any time from Home with o; nothing is saved between sessions.", width)...)
		lines = append(lines, "")
		lines = append(lines, styledParagraph("First prompt:", width, 0, theme.Section.Render)...)
		lines = append(lines, styledParagraph(FirstRouterPrompt, width, 2, theme.Accent.Render)...)
		lines = append(lines, "")
		lines = append(lines, m.linkLines(width)...)
	}

	lines = append(lines, "")
	lines = append(lines, m.hintLines(width)...)
	return lines
}

// Render joins Lines at width.
func (m OnboardingModel) Render(width int) string {
	return strings.Join(m.Lines(width), "\n")
}

// View applies the current vertical scroll and height, then returns a Bubble
// Tea view. AltScreen follows the run's interactive setting, exactly like the
// other screens; plain sessions stay unstyled because the theme is zero.
func (m OnboardingModel) View() tea.View {
	lines := m.Lines(m.width)
	if m.height > 0 && len(lines) > m.height {
		maxScroll := len(lines) - m.height
		scroll := m.scroll
		if scroll > maxScroll {
			scroll = maxScroll
		}
		if scroll < 0 {
			scroll = 0
		}
		lines = lines[scroll : scroll+m.height]
	}

	view := tea.NewView(strings.Join(lines, "\n"))
	view.AltScreen = m.cfg.Interactive
	view.WindowTitle = BrandName + " " + QuickStartName
	return view
}

func (m OnboardingModel) installedBody() string {
	clients := "Clients: selected later in Install & Manage."
	if len(m.summary.Clients) > 0 {
		clients = "Clients: " + clientPlainNames(m.summary.Clients) + "."
	}

	optional := "Optional content: not selected."
	switch {
	case m.summary.WithOpenCode && m.summary.OpenCodeFreeFirst:
		optional = "Optional content: OpenCode agent profiles and the free-model template preference."
	case m.summary.WithOpenCode:
		optional = "Optional content: OpenCode agent profiles."
	case m.summary.OpenCodeFreeFirst:
		optional = "Optional content: the free-model template preference."
	}

	return fmt.Sprintf(
		"Skills in this selection: %d. Core skills are placed in the client skill directories you select; optional content is included only when you select it.\n%s\n%s",
		m.summary.SkillCount, clients, optional,
	)
}

func (m OnboardingModel) reviewBody() string {
	clients := "No client destinations were reported for this selection."
	if len(m.summary.Clients) > 0 {
		clients = "Clients: " + clientPlainNames(m.summary.Clients) + "."
	}

	home := "Home directory: not reported."
	if m.summary.Home != "" {
		home = "Home directory: " + m.summary.Home
	}
	repo := "Repository source: embedded skills in this binary."
	if m.summary.Repo != "" {
		repo = "Repository source: " + m.summary.Repo
	}

	return fmt.Sprintf(
		"%s\n%s\n%s\nSkills in this selection: %d\nProfiles in this selection: %d",
		clients, home, repo, m.summary.SkillCount, m.summary.ProfileCount,
	)
}

func (m OnboardingModel) linkLines(width int) []string {
	enabled := m.cfg.Color && !m.cfg.ASCII
	lines := styledParagraph("Links", width, 0, m.theme.Section.Render)
	lines = append(lines, "")
	lines = append(lines, m.linkPair("Online guide / README:", LinkReadme, width, enabled)...)
	lines = append(lines, "")
	lines = append(lines, m.linkPair("Author profile (by "+BrandAuthor+"):", LinkAuthor, width, enabled)...)
	lines = append(lines, "")
	lines = append(lines, styledParagraph(
		"Press o to open the online guide or a to open the author profile. Opening is explicit. In a remote SSH session or without a desktop opener, the URL stays visible above so you can copy it and open it locally.",
		width, 2, m.theme.Hint.Render,
	)...)
	return lines
}

func (m OnboardingModel) linkPair(label string, link Link, width int, enabled bool) []string {
	lines := styledParagraph(label, width, 2, m.theme.Hint.Render)
	url := link.URL()
	if url == "" {
		return append(lines, styledParagraph("unavailable", width, 4, m.theme.Bad.Render)...)
	}

	const indent = 4
	if width > indent && ansi.StringWidth(url) <= width-indent {
		return append(lines, strings.Repeat(" ", indent)+Hyperlink(url, link, enabled))
	}
	// A URL wider than the pane is shown as plain, hard-wrapped, copyable text.
	// Hyperlinking one wrapped URL would hide the target across line breaks.
	return append(lines, styledParagraph(url, width, indent, m.theme.Accent.Render)...)
}

func (m OnboardingModel) hintLines(width int) []string {
	parts := []string{"Enter/Right next", "Left back", "j/k scroll"}
	if m.step == OnboardingFinish {
		parts = append(parts, "o open guide", "a open author")
	}
	parts = append(parts, "s skip", "Esc cancel")
	return styledParagraph(strings.Join(parts, " | "), width, 0, m.theme.Hint.Render)
}

// section renders one titled, indented body block. Both the title and body go
// through Paragraph, so a long title wraps instead of overflowing a narrow
// viewport.
func (m OnboardingModel) section(title, body string, width int) []string {
	lines := styledParagraph(title, width, 0, m.theme.Section.Render)
	lines = append(lines, "")
	return append(lines, styledParagraph(body, width, 2, m.theme.Hint.Render)...)
}

// styledParagraph wraps text to width with indent and applies render to each
// produced line. render may be nil for unstyled output.
func styledParagraph(text string, width, indent int, render func(...string) string) []string {
	lines := Paragraph(text, width, indent)
	if render == nil {
		return lines
	}
	for i := range lines {
		lines[i] = render(lines[i])
	}
	return lines
}
