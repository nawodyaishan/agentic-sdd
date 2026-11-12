package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/version"
)

const (
	// DefaultMaxLogsToInspect bounds the number of operation log files listed.
	DefaultMaxLogsToInspect = 10
	// DefaultMaxLogBytes bounds the maximum bytes read from a single log file (64 KiB).
	DefaultMaxLogBytes = 64 << 10
	// DefaultMaxLogLines bounds the maximum displayed lines for a single log file.
	DefaultMaxLogLines = 200
)

// LogSummary holds metadata for one discovered operation log.
type LogSummary struct {
	Name    string
	Path    string
	ModTime time.Time
	Size    int64
}

// LogReaderFunc reads available operation logs from a directory.
type LogReaderFunc func(dir string) ([]LogSummary, error)

// LogContentReaderFunc loads the bounded content of a log file.
type LogContentReaderFunc func(path string, maxBytes int64, maxLines int) (string, error)

// DefaultLogReader scans dir for regular .log files, sorting newest first
// and bounding results to DefaultMaxLogsToInspect.
// If dir does not exist or is empty, it returns an empty slice and nil error (friendly empty state).
func DefaultLogReader(dir string) ([]LogSummary, error) {
	if dir == "" {
		return nil, nil
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("refusing non-directory log path %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	entryMap := make(map[string]os.DirEntry, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && strings.HasSuffix(entry.Name(), ".log") {
			names = append(names, entry.Name())
			entryMap[entry.Name()] = entry
		}
	}

	if len(names) == 0 {
		return nil, nil
	}

	// Lexicographical sort on timestamp names (e.g. 20060102T150405.000000000Z-*.log)
	// sorts chronologically. We sort ascending so names[len-1] is the newest.
	sort.Strings(names)

	// Present newest first
	limit := len(names)
	if limit > DefaultMaxLogsToInspect {
		limit = DefaultMaxLogsToInspect
	}

	summaries := make([]LogSummary, 0, limit)
	for i := len(names) - 1; i >= len(names)-limit; i-- {
		name := names[i]
		entry := entryMap[name]
		info, _ := entry.Info()
		summaries = append(summaries, LogSummary{
			Name:    name,
			Path:    filepath.Join(dir, name),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}

	return summaries, nil
}

// DefaultLogContentReader reads up to maxBytes bytes and maxLines lines from path.
// It never modifies or writes files. If the content exceeds limits, a truncation notice is appended.
func DefaultLogContentReader(path string, maxBytes int64, maxLines int) (string, error) {
	if path == "" {
		return "", nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if maxBytes <= 0 {
		maxBytes = DefaultMaxLogBytes
	}
	if maxLines <= 0 {
		maxLines = DefaultMaxLogLines
	}

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", err
	}

	truncated := false
	if int64(len(data)) > maxBytes {
		data = data[:maxBytes]
		truncated = true
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	content := strings.Join(lines, "\n")
	if truncated {
		content = strings.TrimRight(content, "\r\n") + "\n... [truncated: log exceeds display limit]"
	}

	return content, nil
}

// AboutMetadata holds version and build identification.
type AboutMetadata struct {
	Version string
	Commit  string
	Date    string
	Author  string
	URL     string
}

// DefaultAboutMetadata returns metadata extracted from version and branding constants.
func DefaultAboutMetadata() AboutMetadata {
	return AboutMetadata{
		Version: version.Version,
		Commit:  version.Commit,
		Date:    version.Date,
		Author:  BrandAuthor,
		URL:     "https://github.com/nawodyaishan/agentic-sdd#readme",
	}
}

// CommandHelp describes one supported CLI command.
type CommandHelp struct {
	Command     string
	Description string
}

// SupportedCommands lists the CLI commands available in the agentic-sdd suite.
func SupportedCommands() []CommandHelp {
	return []CommandHelp{
		{Command: "agentic-sdd preview", Description: "Inspect pending skill additions, replacements and clients (read-only)"},
		{Command: "agentic-sdd apply", Description: "Install or update skills into client directories with automatic backup"},
		{Command: "agentic-sdd backups", Description: "List existing backup timestamps and file manifests"},
		{Command: "agentic-sdd restore", Description: "Preview or restore skills from an earlier backup"},
		{Command: "agentic-sdd logs", Description: "Display the latest installation or restore operation log"},
		{Command: "agentic-sdd version", Description: "Print binary version, commit SHA and build date"},
		{Command: "agentic-sdd guide", Description: "Run the interactive guided terminal walkthrough"},
		{Command: "agentic-sdd tui", Description: "Launch this interactive terminal UI experience"},
	}
}

// KeyHelp describes one keyboard shortcut.
type KeyHelp struct {
	Key         string
	Description string
}

// NavigationShortcuts lists the global and route keyboard shortcuts.
func NavigationShortcuts() []KeyHelp {
	return []KeyHelp{
		{Key: "Tab", Description: "Cycle navigation sidebar, categories (Explore), and content"},
		{Key: "↑ / ↓ / j / k", Description: "Move cursor across routes or items"},
		{Key: "1 - 4", Description: "Direct jump to destination (1:Home, 2:Install, 3:Explore, 4:Activity)"},
		{Key: "Enter", Description: "Open item / activate selection"},
		{Key: "b / Esc", Description: "Back to list or return focus to navigation"},
		{Key: "/", Description: "Search and filter catalog items"},
		{Key: "c / [ / ]", Description: "Focus categories / previous / next category"},
		{Key: "pgup / pgdn", Description: "Scroll content page up / page down"},
		{Key: "r", Description: "Retry update check / refresh activity and logs"},
		{Key: "m", Description: "Toggle motion animation reveal (on/off)"},
		{Key: "u", Description: "Toggle symbols (Unicode icons vs 7-bit ASCII)"},
		{Key: "q / Ctrl+C", Description: "Quit application"},
	}
}

// ActivityConfig configures the Activity/About/Help model.
type ActivityConfig struct {
	Theme         Theme
	Width         int
	Height        int
	LogDir        string
	Metadata      AboutMetadata
	UpdateManager *UpdateCheckManager
	LogReader     LogReaderFunc
	ContentReader LogContentReaderFunc
}

// ActivityModel represents the read-only Activity, About, Help and Update status screen.
// It supports immediate local rendering, bounded log inspection with an explicit empty state,
// version metadata, and Bubble Tea v2 component integration.
type ActivityModel struct {
	theme         Theme
	width         int
	height        int
	logDir        string
	metadata      AboutMetadata
	updateManager *UpdateCheckManager
	logReader     LogReaderFunc
	contentReader LogContentReaderFunc

	logs          []LogSummary
	selectedLog   int
	cachedContent string
	logErr        error
	activeSection HelpSection
	logScroll     int
	// readGeneration binds one asynchronous log read to the live model, so a
	// late result can never overwrite a newer selection, log directory or
	// refresh. StartLogRead increments it; ApplyLogRead accepts only a match.
	readGeneration uint64

	// motion and ascii mirror the root session's presentation settings so the
	// Preferences section reports the actual state instead of a hardcoded
	// guess. The root calls SetSessionPreferences before rendering; standalone
	// tests can set them directly.
	motion bool
	ascii  bool
}

// NewActivityModel constructs an ActivityModel and performs one initial bounded
// log scan. That construction-time scan is the only synchronous I/O in the
// model; interactive refreshes go through StartLogRead and ApplyLogRead.
// Tests can inject custom LogReader and ContentReader functions to avoid
// touching real files.
func NewActivityModel(cfg ActivityConfig) ActivityModel {
	m := ActivityModel{
		theme:         cfg.Theme,
		width:         cfg.Width,
		height:        cfg.Height,
		logDir:        cfg.LogDir,
		metadata:      cfg.Metadata,
		updateManager: cfg.UpdateManager,
		logReader:     cfg.LogReader,
		contentReader: cfg.ContentReader,
	}

	if m.metadata.Version == "" {
		m.metadata = DefaultAboutMetadata()
	}
	if m.logReader == nil {
		m.logReader = DefaultLogReader
	}
	if m.contentReader == nil {
		m.contentReader = DefaultLogContentReader
	}
	if m.updateManager == nil {
		m.updateManager = NewUpdateCheckManager()
	}

	m.reloadLogs()
	return m
}

// reloadLogs performs one synchronous bounded scan and content read. It is used
// by construction (and by same-package tests); App.Update never calls it, so a
// key press or message handling cannot block on the filesystem.
func (m *ActivityModel) reloadLogs() {
	if m.logReader == nil {
		return
	}
	logs, err := m.scanLogs()
	content := ""
	selected := m.selectedLog
	if len(logs) > 0 {
		if selected >= len(logs) || selected < 0 {
			selected = 0
		}
		content = m.readLogContent(logs[selected])
	}
	m.applyLogRead(logs, err, selected, content)
}

// scanLogs performs one bounded log-directory read without mutating the model.
// It is safe to call from a tea.Cmd.
func (m ActivityModel) scanLogs() ([]LogSummary, error) {
	if m.logReader == nil {
		return nil, nil
	}
	return m.logReader(m.logDir)
}

// readLogContent reads one log's bounded content without mutating the model.
func (m ActivityModel) readLogContent(log LogSummary) string {
	if m.contentReader == nil {
		return ""
	}
	content, err := m.contentReader(log.Path, DefaultMaxLogBytes, DefaultMaxLogLines)
	if err != nil {
		return fmt.Sprintf("Error reading log content: %v", err)
	}
	return content
}

// applyLogRead applies one accepted bounded read. It touches only the log list,
// the read error, the selected index and its content; the active help section,
// log scroll, theme, size and every other presentation choice of the live model
// are preserved.
func (m *ActivityModel) applyLogRead(logs []LogSummary, err error, selected int, content string) {
	m.logs = logs
	m.logErr = err
	if len(logs) == 0 {
		m.selectedLog = 0
		m.cachedContent = ""
		return
	}
	if selected < 0 || selected >= len(logs) {
		selected = 0
	}
	m.selectedLog = selected
	m.cachedContent = content
}

// StartLogRead begins one bounded log read as a tea.Cmd. It increments the
// read generation and captures the log directory, so the command performs only
// reads and ApplyLogRead can reject a result that belongs to an older
// selection, directory or refresh. Update itself never touches the filesystem.
func (m *ActivityModel) StartLogRead(selected int) tea.Cmd {
	m.readGeneration++
	generation := m.readGeneration
	model := *m
	logDir := model.logDir
	return func() tea.Msg {
		logs, err := model.scanLogs()
		content := ""
		if len(logs) > 0 {
			if selected < 0 || selected >= len(logs) {
				selected = 0
			}
			content = model.readLogContent(logs[selected])
		}
		return ActivityLogReadMsg{
			Generation: generation,
			LogDir:     logDir,
			Selected:   selected,
			Logs:       logs,
			Err:        err,
			Content:    content,
		}
	}
}

// ApplyLogRead applies a read only when it still belongs to the live model's
// generation and log directory. It reports whether the result was accepted.
func (m *ActivityModel) ApplyLogRead(msg ActivityLogReadMsg) bool {
	if msg.Generation != m.readGeneration || msg.LogDir != m.logDir {
		return false
	}
	m.applyLogRead(msg.Logs, msg.Err, msg.Selected, msg.Content)
	return true
}

// MoveLogSelection advances the bounded log selection without reading any
// file. The caller schedules the matching content read with StartLogRead. The
// previous content is cleared so it is never displayed under the newly
// selected log's name while the read is in flight.
func (m *ActivityModel) MoveLogSelection(delta int) {
	if len(m.logs) <= 1 {
		return
	}
	m.selectedLog = (m.selectedLog + delta + len(m.logs)) % len(m.logs)
	m.logScroll = 0
	m.cachedContent = ""
}

// SelectSection sets the active help/activity topic.
func (m *ActivityModel) SelectSection(s HelpSection) {
	m.activeSection = s
}

// ActiveSection reports the currently chosen help/activity topic.
func (m ActivityModel) ActiveSection() HelpSection {
	return m.activeSection
}

// SelectNextSection cycles forward through help sections.
func (m *ActivityModel) SelectNextSection() {
	total := HelpSection(len(AllHelpSections()))
	m.activeSection = (m.activeSection + 1) % total
}

// SelectPrevSection cycles backward through help sections.
func (m *ActivityModel) SelectPrevSection() {
	total := HelpSection(len(AllHelpSections()))
	m.activeSection = (m.activeSection - 1 + total) % total
}

// ScrollLog adjusts the vertical scroll offset within the raw log viewport.
func (m *ActivityModel) ScrollLog(delta int) {
	m.logScroll += delta
	if m.logScroll < 0 {
		m.logScroll = 0
	}
}

// SelectedLog returns the currently highlighted log summary, if any.
func (m ActivityModel) SelectedLog() (LogSummary, bool) {
	if len(m.logs) == 0 || m.selectedLog < 0 || m.selectedLog >= len(m.logs) {
		return LogSummary{}, false
	}
	return m.logs[m.selectedLog], true
}

// Logs returns a defensive copy of the current log summaries.
func (m ActivityModel) Logs() []LogSummary {
	res := make([]LogSummary, len(m.logs))
	copy(res, m.logs)
	return res
}

// LogContent returns the loaded content string for the currently selected log.
func (m ActivityModel) LogContent() string {
	return m.cachedContent
}

// UpdateManager returns the underlying update check state machine.
func (m ActivityModel) UpdateManager() *UpdateCheckManager {
	return m.updateManager
}

// SetSize updates the model's dimensions.
func (m *ActivityModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetTheme updates the active styling theme.
func (m *ActivityModel) SetTheme(theme Theme) {
	m.theme = theme
}

// SetSessionPreferences records the root-owned motion and symbol settings so
// the Preferences section renders actual state. It is presentation-only and
// never changes an operation, consent or update-check state.
func (m *ActivityModel) SetSessionPreferences(motion, ascii bool) {
	m.motion = motion
	m.ascii = ascii
}

// SetLogDir changes the log directory. It performs no I/O; the caller schedules
// the next read with StartLogRead.
func (m *ActivityModel) SetLogDir(dir string) {
	m.logDir = dir
}

// Init implements tea.Model.
func (m ActivityModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It schedules bounded reads as commands and
// never touches the filesystem itself; ApplyLogRead applies their results.
func (m ActivityModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case UpdateCheckResultMsg:
		if m.updateManager != nil {
			m.updateManager.HandleMsg(msg)
		}
		return m, nil

	case ActivityLogReadMsg:
		m.ApplyLogRead(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			cmds := []tea.Cmd{m.StartLogRead(m.selectedLog)}
			if m.updateManager != nil {
				if cmd, _ := m.updateManager.RetryCheck(m.metadata.Version); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		case "]", "n":
			m.MoveLogSelection(1)
			return m, m.StartLogRead(m.selectedLog)
		case "[", "p":
			m.MoveLogSelection(-1)
			return m, m.StartLogRead(m.selectedLog)
		case "t":
			m.SelectNextSection()
			return m, nil
		case "T":
			m.SelectPrevSection()
			return m, nil
		case "J":
			m.ScrollLog(4)
			return m, nil
		case "K":
			m.ScrollLog(-4)
			return m, nil
		}
	}

	return m, nil
}

// View implements tea.Model for standalone usage.
func (m ActivityModel) View() tea.View {
	lines := m.Lines(m.width)
	view := tea.NewView(strings.Join(lines, "\n"))
	view.AltScreen = false
	return view
}

// dataText renders a dynamic value for the session's symbol set. In ASCII mode
// the value is reversibly escaped (for example an em dash becomes \u2014) before
// the whole line goes through the decorative glyph mapping, so a log path or
// log line keeps its exact identity instead of being rewritten into different
// bytes. Authoritative model fields are never mutated.
func (m ActivityModel) dataText(text string) string {
	if m.ascii {
		return escapeData(text)
	}
	return text
}

// Lines renders the complete Activity, About, Help, and Update screen as a slice
// of bounds-safe lines wrapped to width.
func (m ActivityModel) Lines(width int) []string {
	if width <= 0 {
		width = DefaultWidth
	}

	var sections [][]string

	// Header
	var header []string
	for _, l := range Paragraph("Activity & Help", width, 0) {
		header = append(header, m.theme.Title.Render(l))
	}
	for _, l := range Paragraph("Version metadata, activity logs, command guide & reference", width, 0) {
		header = append(header, m.theme.Subtitle.Render(l))
	}

	// Section Navigation Pills
	var navPills []string
	for _, s := range AllHelpSections() {
		label := s.Title
		if s.Section == m.activeSection {
			label = "[" + label + "]"
			if m.theme.Color {
				label = m.theme.Selected.Bold(true).Render(label)
			}
		} else {
			label = " " + label + " "
			if m.theme.Color {
				label = m.theme.Muted.Render(label)
			}
		}
		navPills = append(navPills, label)
	}
	for _, l := range Paragraph(strings.Join(navPills, " "), width, 0) {
		header = append(header, l)
	}
	for _, l := range Paragraph("Press 't' to cycle sections, 'r' to refresh, 'g' for README, 'a' for author.", width, 0) {
		header = append(header, m.theme.Hint.Render(l))
	}
	sections = append(sections, header)

	// Version & Build / About
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionAbout {
		versionLabel := m.metadata.Version
		if IsDevBuild(m.metadata.Version) {
			versionLabel = "Development build (see Diagnostics for commit and build details)"
		}
		vBody := []string{
			fmt.Sprintf("Version: %s", versionLabel),
			fmt.Sprintf("Commit:  %s", m.metadata.Commit),
			fmt.Sprintf("Date:    %s", m.metadata.Date),
			"Product: " + BrandName,
			"Tagline: " + BrandTagline,
			fmt.Sprintf("Author:  by %s (https://github.com/%s)", m.metadata.Author, m.metadata.Author),
			"Online guide / README: " + m.metadata.URL,
			m.theme.Hint.Render("Open links explicitly: 'g' for online README, 'a' for author profile."),
		}
		sections = append(sections, Section(m.theme.Section.Render("Version & Build"), vBody, width))
	}

	// Recent Activity & Logs
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionActivity {
		var logBody []string
		if m.logErr != nil {
			logBody = append(logBody, m.theme.Bad.Render(m.dataText(fmt.Sprintf("Error reading logs: %v", m.logErr))))
		} else if len(m.logs) == 0 {
			logBody = append(logBody,
				m.theme.Hint.Render("No install or restore logs found yet."),
				"Logs are created automatically when you run apply or restore.",
				"Run agentic-sdd preview first to inspect what apply would do.",
			)
		} else {
			glyphs := GlyphSet(!m.theme.Color)
			logBody = append(logBody, fmt.Sprintf("Recent operation logs (%d found):", len(m.logs)))
			for i, log := range m.logs {
				marker := glyphs.Gutter
				if i == m.selectedLog {
					marker = glyphs.Cursor
				}
				latest := ""
				if i == 0 {
					latest = " " + glyphs.Bullet + " latest"
				}
				row := fmt.Sprintf("  %s [%d/%d] %s (%s)%s", marker, i+1, len(m.logs), m.dataText(log.Name), formatBytes(log.Size), latest)
				if i == m.selectedLog {
					if m.theme.Color {
						logBody = append(logBody, m.theme.Selected.Bold(true).Render(row))
					} else {
						logBody = append(logBody, row)
					}
				} else {
					logBody = append(logBody, row)
				}
			}
			if len(m.logs) > 1 {
				logBody = append(logBody, m.theme.Hint.Render("Press [ / ] or p / n to view earlier logs."))
			}

			// Structured operation facts for selected log
			selected := m.logs[m.selectedLog]
			facts := ParseLogOperationFacts(selected)
			logBody = append(logBody, "")
			logBody = append(logBody, fmt.Sprintf("Operation Summary (%s):", m.dataText(selected.Name)))
			logBody = append(logBody,
				"  Operation: "+m.dataText(facts.Operation),
				"  Timestamp: "+m.dataText(facts.Timestamp)+" ("+m.dataText(facts.RawTimestamp)+")",
				"  File size: "+formatBytes(facts.Size),
				"  Log path:  "+m.dataText(facts.Path),
				"  Structured outcome: "+m.dataText(facts.Outcome),
				"  Changed count:      "+facts.ChangedCount,
				"  Skipped count:      "+facts.SkippedCount,
				"  Failed count:       "+facts.FailedCount,
			)

			if m.cachedContent != "" {
				logBody = append(logBody, "")
				logBody = append(logBody, fmt.Sprintf("Log Content (%s):", m.dataText(selected.Name)))
				rawLines := strings.Split(m.cachedContent, "\n")
				startLine := m.logScroll
				if startLine > len(rawLines) {
					startLine = len(rawLines)
				}
				for _, cLine := range rawLines[startLine:] {
					logBody = append(logBody, "  "+m.dataText(cLine))
				}
			}
		}
		sections = append(sections, Section(m.theme.Section.Render("Recent Activity & Logs"), logBody, width))
	}

	// Quick start
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionQuickStart {
		workflowBody := []string{
			"1. Explore skills in the Library (Explore) to inspect purpose, triggers and caveats",
			"2. Review client configuration in Install & Manage (safe, read-only impact preview)",
			"3. Confirm changes with explicit default-No (automatic safety backup created first)",
			"4. Start your first agentic workflow: agentic-sdd-router",
			"",
			"New here? Start with SDD Router in the Library (press 's' in Explore for details).",
			"Agent models: open Fleet Manager (Install & Manage, key f) or read the Fleet Manager topic; preview only in this build.",
			"Command hint: agentic-sdd preview",
		}
		sections = append(sections, Section(m.theme.Section.Render("Quick Start Guide"), workflowBody, width))
	}

	// Shortcuts
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionShortcuts {
		var navBody []string
		for _, sc := range NavigationShortcuts() {
			navBody = append(navBody, fmt.Sprintf("%-16s %s", sc.Key, sc.Description))
		}
		sections = append(sections, Section(m.theme.Section.Render("Shortcuts & Keyboard Reference"), navBody, width))
	}

	// Commands
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionCommands {
		var cmdBody []string
		cmdBody = append(cmdBody, "Supported CLI commands:")
		for _, cmd := range SupportedCommands() {
			cmdBody = append(cmdBody, fmt.Sprintf("  %-22s %s", cmd.Command, cmd.Description))
		}
		cmdBody = append(cmdBody, "")
		cmdBody = append(cmdBody, "Excluded experimental operations (explanatory only · non-executable):")
		for _, cmd := range ExcludedCommands() {
			cmdBody = append(cmdBody, fmt.Sprintf("  %-22s %s", cmd.Command, cmd.Description))
		}
		sections = append(sections, Section(m.theme.Section.Render("Commands & CLI Reference"), cmdBody, width))
	}

	// Troubleshooting
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionTroubleshooting {
		var tbBody []string
		for i, topic := range TroubleshootingTopics() {
			if i > 0 {
				tbBody = append(tbBody, "")
			}
			tbBody = append(tbBody, m.theme.Accent.Render(topic.Title)+":")
			for _, line := range Paragraph(topic.Advice, width, 2) {
				tbBody = append(tbBody, line)
			}
			for _, line := range Paragraph("Recovery: "+topic.Recovery, width, 2) {
				tbBody = append(tbBody, m.theme.Hint.Render(line))
			}
		}
		sections = append(sections, Section(m.theme.Section.Render("Troubleshooting"), tbBody, width))
	}

	// Diagnostics
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionDiagnostics {
		diag := CollectDiagnostics(m.metadata, m.logDir, m.updateManager, m.theme)
		diagBody := []string{
			fmt.Sprintf("Version:       %s", diag.Version),
			fmt.Sprintf("Commit:        %s", diag.Commit),
			fmt.Sprintf("Build Date:    %s", diag.Date),
			fmt.Sprintf("Author:        %s", diag.Author),
			fmt.Sprintf("Go Runtime:    %s", diag.GoVersion),
			fmt.Sprintf("Platform:      %s", diag.Platform),
			fmt.Sprintf("Log Directory: %s", m.dataText(diag.LogDir)),
			fmt.Sprintf("Update Status: %s", diag.UpdateStatus),
			fmt.Sprintf("Offline Mode:  %t (AGENTIC_SDD_NO_UPDATE_CHECK=1)", diag.OfflineMode),
		}
		sections = append(sections, Section(m.theme.Section.Render("Diagnostics & Environment"), diagBody, width))
	}

	// Session Preferences
	if m.activeSection == HelpSectionAll || m.activeSection == HelpSectionPreferences {
		motionStatus := "off"
		if m.motion {
			motionStatus = "on"
		}
		symbolStatus := "Unicode"
		if m.ascii {
			symbolStatus = "ASCII"
		}
		checksStatus := "checks enabled"
		if m.updateManager != nil && m.updateManager.IsDisabled() {
			checksStatus = "checks disabled"
		}
		prefBody := []string{
			fmt.Sprintf("  m            Toggle motion animation reveal (current: motion %s)", motionStatus),
			fmt.Sprintf("  u            Toggle symbol set (current: %s)", symbolStatus),
			fmt.Sprintf("  d            Toggle update checks for this session (current: %s)", checksStatus),
			"  r            Retry update check and reload recent logs",
			"  Theme:       agentic-dark · " + m.theme.ContrastClaim(),
			m.theme.Hint.Render("Session preferences take effect immediately and persist for this session. No connectivity is claimed."),
		}
		sections = append(sections, Section(m.theme.Section.Render("Session Preferences"), prefBody, width))
	}

	// Fleet Manager. "All" keeps a compact pointer so adding the topic does not
	// bury the later reference sections; the full guidance is its own tab.
	if m.activeSection == HelpSectionAll {
		sections = append(sections, Section(m.theme.Section.Render("Fleet Manager"), []string{
			"See the Fleet Manager topic (9) for source, model, preview/apply and restore guidance; this build is preview only.",
		}, width))
	}
	if m.activeSection == HelpSectionFleet {
		sections = append(sections, Section(m.theme.Section.Render("Fleet Manager"), FleetHelpLines(), width))
	}

	// Flatten all sections with blank separators
	var lines []string
	for i, sec := range sections {
		if i > 0 && len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, sec...)
	}

	return lines
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
