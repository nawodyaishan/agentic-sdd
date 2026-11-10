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
}

// NewActivityModel constructs an ActivityModel and performs an initial bounded log scan.
// Tests can inject custom LogReader and ContentReader functions to avoid touching real files.
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

	m.ReloadLogs()
	return m
}

// ReloadLogs re-scans the log directory and reloads the selected log's bounded content.
func (m *ActivityModel) ReloadLogs() {
	if m.logReader == nil {
		return
	}

	logs, err := m.logReader(m.logDir)
	m.logs = logs
	m.logErr = err

	if len(m.logs) == 0 {
		m.selectedLog = 0
		m.cachedContent = ""
		return
	}

	if m.selectedLog >= len(m.logs) || m.selectedLog < 0 {
		m.selectedLog = 0
	}

	m.loadSelectedContent()
}

func (m *ActivityModel) loadSelectedContent() {
	if len(m.logs) == 0 || m.selectedLog < 0 || m.selectedLog >= len(m.logs) {
		m.cachedContent = ""
		return
	}

	if m.contentReader == nil {
		m.cachedContent = ""
		return
	}

	content, err := m.contentReader(m.logs[m.selectedLog].Path, DefaultMaxLogBytes, DefaultMaxLogLines)
	if err != nil {
		m.cachedContent = fmt.Sprintf("Error reading log content: %v", err)
		return
	}
	m.cachedContent = content
}

// SelectNextLog advances to the next older log file, if available.
func (m *ActivityModel) SelectNextLog() {
	if len(m.logs) <= 1 {
		return
	}
	m.selectedLog = (m.selectedLog + 1) % len(m.logs)
	m.loadSelectedContent()
}

// SelectPrevLog moves to the previous log file, if available.
func (m *ActivityModel) SelectPrevLog() {
	if len(m.logs) <= 1 {
		return
	}
	m.selectedLog = (m.selectedLog - 1 + len(m.logs)) % len(m.logs)
	m.loadSelectedContent()
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

// SetLogDir changes the log directory and triggers a reload.
func (m *ActivityModel) SetLogDir(dir string) {
	m.logDir = dir
	m.ReloadLogs()
}

// Init implements tea.Model.
func (m ActivityModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
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

	case tea.KeyPressMsg:
		switch msg.String() {
		case "r":
			m.ReloadLogs()
			if m.updateManager != nil {
				cmd, _ := m.updateManager.RetryCheck(m.metadata.Version)
				return m, cmd
			}
			return m, nil
		case "]", "n":
			m.SelectNextLog()
			return m, nil
		case "[", "p":
			m.SelectPrevLog()
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
	sections = append(sections, header)

	// Version & Build
	vBody := []string{
		fmt.Sprintf("Version: %s", m.metadata.Version),
		fmt.Sprintf("Commit:  %s", m.metadata.Commit),
		fmt.Sprintf("Date:    %s", m.metadata.Date),
	}
	sections = append(sections, Section(m.theme.Section.Render("Version & Build"), vBody, width))

	// Update Status
	if m.updateManager != nil {
		uStatus := m.updateManager.RenderStatus(m.theme)
		var uHint string
		if m.updateManager.IsDisabled() {
			uHint = "Automatic check disabled (AGENTIC_SDD_NO_UPDATE_CHECK=1 or offline mode)."
		} else {
			uHint = "Checks automatically once per session on first Home entry. Press r to retry."
		}
		uBody := []string{
			"Status: " + uStatus,
			m.theme.Hint.Render(uHint),
		}
		sections = append(sections, Section(m.theme.Section.Render("Update Status"), uBody, width))
	}

	// Online Documentation & Attribution
	dBody := []string{
		"Online guide / README: " + m.metadata.URL,
		fmt.Sprintf("Author: by %s (https://github.com/%s)", m.metadata.Author, m.metadata.Author),
		fmt.Sprintf("Vision: %s", BrandTagline),
		m.theme.Hint.Render("Open links with your platform opener or copy the URL above."),
	}
	sections = append(sections, Section(m.theme.Section.Render("Online Documentation & Attribution"), dBody, width))

	// Recent Activity & Logs
	var logBody []string
	if m.logErr != nil {
		logBody = append(logBody, m.theme.Bad.Render(fmt.Sprintf("Error reading logs: %v", m.logErr)))
	} else if len(m.logs) == 0 {
		logBody = append(logBody,
			m.theme.Hint.Render("No install or restore logs found yet."),
			"Logs are created automatically when you run apply or restore.",
			"Run agentic-sdd preview first to inspect what apply would do.",
		)
	} else {
		logBody = append(logBody, fmt.Sprintf("Recent operation logs (%d found):", len(m.logs)))
		for i, log := range m.logs {
			marker := " "
			if i == m.selectedLog {
				marker = "*"
			}
			latest := ""
			if i == 0 {
				latest = " · latest"
			}
			row := fmt.Sprintf("  %s [%d/%d] %s (%s)%s", marker, i+1, len(m.logs), log.Name, formatBytes(log.Size), latest)
			if i == m.selectedLog {
				logBody = append(logBody, m.theme.Accent.Render(row))
			} else {
				logBody = append(logBody, row)
			}
		}
		if len(m.logs) > 1 {
			logBody = append(logBody, m.theme.Hint.Render("Press [ / ] or p / n to view earlier logs."))
		}

		if m.cachedContent != "" {
			selectedLogName := m.logs[m.selectedLog].Name
			logBody = append(logBody, "")
			logBody = append(logBody, fmt.Sprintf("Log Content (%s):", selectedLogName))
			for _, cLine := range strings.Split(m.cachedContent, "\n") {
				logBody = append(logBody, "  "+cLine)
			}
		}
	}
	sections = append(sections, Section(m.theme.Section.Render("Recent Activity & Logs"), logBody, width))

	// Command Reference
	var cmdBody []string
	for _, cmd := range SupportedCommands() {
		cmdBody = append(cmdBody, fmt.Sprintf("%-22s %s", cmd.Command, cmd.Description))
	}
	sections = append(sections, Section(m.theme.Section.Render("Commands & CLI Reference"), cmdBody, width))

	// Getting Started Workflow
	workflowBody := []string{
		"1. Set up / preview skills for your client (Home / Install & Manage)",
		"2. Apply changes when ready (creates an automatic backup and log)",
		"3. Start your first agentic workflow: agentic-sdd-router",
	}
	sections = append(sections, Section(m.theme.Section.Render("Getting Started Workflow"), workflowBody, width))

	// Navigation Reference
	var navBody []string
	for _, sc := range NavigationShortcuts() {
		navBody = append(navBody, fmt.Sprintf("%-16s %s", sc.Key, sc.Description))
	}
	sections = append(sections, Section(m.theme.Section.Render("Navigation Reference"), navBody, width))

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
