package tui

import (
	"context"
	"fmt"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
)

// MCPView selects the active local sub-view within MCP Tools.
type MCPView int

const (
	MCPViewOverview MCPView = iota
	MCPViewSelect
	MCPViewPreview
	MCPViewConfirm
	MCPViewDiagnostics
	MCPViewRecovery
)

func (v MCPView) String() string {
	switch v {
	case MCPViewOverview:
		return "Overview"
	case MCPViewSelect:
		return "Select Servers"
	case MCPViewPreview:
		return "Preview"
	case MCPViewConfirm:
		return "Confirm Changes"
	case MCPViewDiagnostics:
		return "Diagnostics"
	case MCPViewRecovery:
		return "Recovery"
	default:
		return "Overview"
	}
}

// mcpWriteOutcome is the display-only result of one MCP write operation.
type mcpWriteOutcome struct {
	Headline     string
	Detail       string
	BackupID     string
	LogPath      string
	Rollback     string
	NeedsPreview bool
}

func (o mcpWriteOutcome) Empty() bool { return o.Headline == "" }

// MCPStatusMsg delivers one asynchronous MCP status projection.
type MCPStatusMsg struct {
	Generation uint64
	Status     MCPStatusView
	Err        error
}

// MCPPlanPreviewMsg delivers one asynchronous plan preview projection.
type MCPPlanPreviewMsg struct {
	Generation uint64
	Preview    MCPPlanPreviewView
	Err        error
}

// MCPDoctorMsg delivers one asynchronous diagnostic report.
type MCPDoctorMsg struct {
	Generation uint64
	Doctor     MCPDoctorView
	Err        error
}

// MCPBackupListMsg delivers one asynchronous list of managed MCP backups.
type MCPBackupListMsg struct {
	Generation uint64
	Backups    []MCPBackupView
	Err        error
}

// MCPRestorePreviewMsg delivers one asynchronous restore preview.
type MCPRestorePreviewMsg struct {
	Generation uint64
	Preview    MCPRestorePreviewView
	Err        error
}

// MCPModel manages view state and offline interaction for MCP Tools.
// It never imports os, os/exec, or net/http: all I/O is performed through the
// reader/planner seams and OpsHost.
type MCPModel struct {
	cfg        TUIConfig
	theme      Theme
	reader     MCPReader
	planner    MCPPlanner
	width      int
	height     int
	configPath string

	// Navigation & state
	view       MCPView
	prevView   MCPView
	generation uint64
	loading    bool
	loadErr    string
	status     MCPStatusView
	hasLoaded  bool

	// Overview state
	overviewCursor int

	// Select state
	selected        map[string]bool
	authChoices     map[string]string // "context7": "oauth"|"api-key", "exa": "keyless"|"oauth"|"api-key"
	selectCursor    int
	exaAcknowledged bool
	selectNotice    string

	// Preview state
	planPreview   MCPPlanPreviewView
	planGen       uint64
	planLoading   bool
	planErr       string
	token         string
	baseDigest    string
	previewScroll int
	filter        string
	isFiltering   bool

	// Confirm state
	confirmSelection bool // default false (No selected)
	tinyGeometry     bool

	// Diagnostics state
	doctor        MCPDoctorView
	doctorGen     uint64
	doctorLoading bool
	doctorErr     string
	doctorScroll  int

	// Recovery state
	backups          []MCPBackupView
	backupIndex      int
	recoveryGen      uint64
	recoveryLoading  bool
	recoveryErr      string
	recoveryScroll   int
	restorePreview   MCPRestorePreviewView
	restoreGen       uint64
	restoreLoading   bool
	restoreErr       string
	confirmRestore   bool // default false (No selected)
	inRestorePreview bool

	// Shared single-slot write tracking
	writePending bool
	writeKind    WriteKind
	outcome      mcpWriteOutcome
}

// NewMCPModel constructs an MCPModel with default seams.
func NewMCPModel(cfg TUIConfig, reader MCPReader) MCPModel {
	return NewMCPModelWithSeams(cfg, reader, DefaultMCPPlanner{})
}

// NewMCPModelWithSeams constructs an MCPModel with explicit seams.
func NewMCPModelWithSeams(cfg TUIConfig, reader MCPReader, planner MCPPlanner) MCPModel {
	if reader == nil {
		reader = DefaultMCPReader{}
	}
	if planner == nil {
		planner = DefaultMCPPlanner{}
	}

	selected := map[string]bool{
		"codegraph": true,
		"context7":  true,
		"exa":       true,
	}
	authChoices := map[string]string{
		"context7": "oauth",
		"exa":      "keyless",
	}

	return MCPModel{
		cfg:             cfg,
		theme:           newThemeForConfig(cfg),
		reader:          reader,
		planner:         planner,
		width:           cfg.Width,
		height:          cfg.Height,
		view:            MCPViewOverview,
		generation:      1,
		selected:        selected,
		authChoices:     authChoices,
		exaAcknowledged: false,
		status: MCPStatusView{
			Available: false,
			Servers:   defaultServerViews(),
		},
	}
}

// SetTheme updates the rendering theme without touching view state.
func (m *MCPModel) SetTheme(theme Theme) {
	m.theme = theme
}

// SetSize updates the viewport geometry. Resizing cannot confirm:
// it resets confirmation selections and enforces the tiny-geometry guard.
func (m *MCPModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if width < 60 || height < 8 {
		m.tinyGeometry = true
	} else {
		m.tinyGeometry = false
	}
	// AC-71: Resize cannot confirm. Keep No selected on any geometry change.
	m.confirmSelection = false
	m.confirmRestore = false
}

// ActiveView returns the current sub-view.
func (m MCPModel) ActiveView() MCPView { return m.view }

// Breadcrumb returns the human-readable navigation breadcrumb for MCP Tools.
func (m MCPModel) Breadcrumb() string {
	return "Install & Manage / MCP Tools"
}

// BreadcrumbSegments returns the breadcrumb path components.
func (m MCPModel) BreadcrumbSegments() []string {
	return []string{"Install & Manage", "MCP Tools"}
}

// WantsHome reports whether key should navigate to Home.
func (m MCPModel) WantsHome(key string) bool {
	if m.view == MCPViewOverview && !m.isFiltering {
		return key == "h"
	}
	return false
}

// WantsClose reports whether key should close MCP Tools and return to the hub.
func (m MCPModel) WantsClose(key string) bool {
	if m.view == MCPViewOverview && !m.isFiltering {
		return key == "esc"
	}
	return false
}

// ShouldScrollBody reports whether the root shell viewport should scroll child content.
func (m MCPModel) ShouldScrollBody(key string) bool {
	if m.isFiltering {
		return false
	}
	switch m.view {
	case MCPViewPreview, MCPViewDiagnostics, MCPViewRecovery:
		return true
	default:
		return false
	}
}

// IsInputFocused reports whether a printable key belongs to the MCP filter.
func (m MCPModel) IsInputFocused() bool { return m.isFiltering }

// Reload discards the current status projection and starts a fresh lazy read.
// It is used after a successful MCP write; it never writes or executes a binary.
func (m MCPModel) Reload() (MCPModel, tea.Cmd) {
	m.hasLoaded = false
	m.loading = false
	return m.EnsureLoaded()
}

// WritePending reports whether an MCP write has been dispatched and not completed.
func (m MCPModel) WritePending() bool { return m.writePending }

// Outcome returns the display outcome of the last MCP write.
func (m MCPModel) Outcome() mcpWriteOutcome { return m.outcome }

// EnsureLoaded triggers the lazy load of status if not already loaded.
func (m MCPModel) EnsureLoaded() (MCPModel, tea.Cmd) {
	if m.hasLoaded || m.loading {
		return m, nil
	}
	m.loading = true
	m.generation++
	gen := m.generation
	reader := m.reader
	home := m.cfg.Home
	configPath := m.configPath
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		st, err := reader.Status(ctx, home, configPath, "")
		return MCPStatusMsg{
			Generation: gen,
			Status:     st,
			Err:        err,
		}
	}
}

// Lines renders the active MCP view.
func (m MCPModel) Lines(width int) []string {
	switch m.view {
	case MCPViewSelect:
		return mcpSelectLines(m, width)
	case MCPViewPreview:
		return mcpPreviewLines(m, width)
	case MCPViewConfirm:
		return mcpConfirmLines(m, width)
	case MCPViewDiagnostics:
		return mcpDiagnosticsLines(m, width)
	case MCPViewRecovery:
		return mcpRecoveryLines(m, width)
	default:
		return mcpOverviewLines(m, width)
	}
}

// Update handles events and state transitions for MCPModel.
func (m MCPModel) Update(msg tea.Msg) (MCPModel, tea.Cmd) {
	switch msg := msg.(type) {
	case MCPStatusMsg:
		if msg.Generation != m.generation {
			return m, nil
		}
		m.loading = false
		if msg.Err != nil {
			m.loadErr = msg.Err.Error()
			return m, nil
		}
		m.loadErr = ""
		m.status = msg.Status
		m.hasLoaded = true
		return m, nil

	case MCPPlanPreviewMsg:
		if msg.Generation != m.planGen {
			return m, nil
		}
		m.planLoading = false
		if msg.Err != nil {
			m.planErr = msg.Err.Error()
			return m, nil
		}
		m.planErr = ""
		m.planPreview = msg.Preview
		m.token = msg.Preview.Token
		m.baseDigest = msg.Preview.BaseDigest
		return m, nil

	case MCPDoctorMsg:
		if msg.Generation != m.doctorGen {
			return m, nil
		}
		m.doctorLoading = false
		if msg.Err != nil {
			m.doctorErr = msg.Err.Error()
			return m, nil
		}
		m.doctorErr = ""
		m.doctor = msg.Doctor
		return m, nil

	case MCPBackupListMsg:
		if msg.Generation != m.recoveryGen {
			return m, nil
		}
		m.recoveryLoading = false
		if msg.Err != nil {
			m.recoveryErr = msg.Err.Error()
			return m, nil
		}
		m.recoveryErr = ""
		m.backups = msg.Backups
		return m, nil

	case MCPRestorePreviewMsg:
		if msg.Generation != m.restoreGen {
			return m, nil
		}
		m.restoreLoading = false
		if msg.Err != nil {
			m.restoreErr = msg.Err.Error()
			return m, nil
		}
		m.restoreErr = ""
		m.restorePreview = msg.Preview
		m.inRestorePreview = true
		return m, nil

	case WriteStartedMsg:
		if msg.Kind == WriteMCPApply || msg.Kind == WriteMCPRestore {
			m.writePending = true
			m.writeKind = msg.Kind
		}
		return m, nil

	case WriteResultMsg:
		if msg.Kind == WriteMCPApply || msg.Kind == WriteMCPRestore {
			m.writePending = false
			if msg.Err != nil {
				m.outcome = mcpWriteOutcome{
					Headline: "write failed",
					Detail:   msg.Err.Error(),
					LogPath:  msg.LogPath,
				}
			} else {
				verb := "applied"
				if msg.Kind == WriteMCPRestore {
					verb = "restored"
				}
				m.outcome = mcpWriteOutcome{
					Headline: fmt.Sprintf("MCP %s successfully", verb),
					Detail:   fmt.Sprintf("Backup %s recorded", msg.Result.BackupID),
					BackupID: msg.Result.BackupID,
					LogPath:  msg.LogPath,
				}
				// Reload status after write
				m.hasLoaded = false
				return m.EnsureLoaded()
			}
		}
		return m, nil

	case WriteRefusedMsg:
		if msg.Kind == WriteMCPApply || msg.Kind == WriteMCPRestore {
			m.writePending = false
			m.outcome = mcpWriteOutcome{
				Headline: "write refused",
				Detail:   msg.Reason,
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes keyboard input across the local MCP views.
func (m MCPModel) handleKey(msg tea.KeyPressMsg) (MCPModel, tea.Cmd) {
	key := msg.String()

	// Global view switch to diagnostics
	if key == "d" && m.view != MCPViewDiagnostics && !m.isFiltering && m.view != MCPViewConfirm {
		m.prevView = m.view
		m.view = MCPViewDiagnostics
		return m.requestDoctor()
	}

	switch m.view {
	case MCPViewOverview:
		return m.handleOverviewKey(key)
	case MCPViewSelect:
		return m.handleSelectKey(key)
	case MCPViewPreview:
		return m.handlePreviewKey(key)
	case MCPViewConfirm:
		return m.handleConfirmKey(key)
	case MCPViewDiagnostics:
		return m.handleDiagnosticsKey(key)
	case MCPViewRecovery:
		return m.handleRecoveryKey(key)
	default:
		return m, nil
	}
}

// handleOverviewKey routes keys on the Overview view.
func (m MCPModel) handleOverviewKey(key string) (MCPModel, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.overviewCursor < 2 {
			m.overviewCursor++
		}
	case "k", "up":
		if m.overviewCursor > 0 {
			m.overviewCursor--
		}
	case "s", "enter":
		m.view = MCPViewSelect
		m.selectNotice = ""
	case "v":
		m.view = MCPViewPreview
		return m.requestPlanPreview()
	case "r", "e":
		m.view = MCPViewRecovery
		m.inRestorePreview = false
		m.confirmRestore = false
		return m.requestBackupList()
	}
	return m, nil
}

// handleSelectKey routes keys on the server selection view.
func (m MCPModel) handleSelectKey(key string) (MCPModel, tea.Cmd) {
	// Options:
	// 0: codegraph checkbox
	// 1: context7 checkbox
	// 2: context7 auth choice (oauth / api-key)
	// 3: exa checkbox
	// 4: exa auth choice (keyless / oauth / api-key)
	// 5: exa keyless acknowledgement
	maxCursor := 5

	switch key {
	case "j", "down":
		if m.selectCursor < maxCursor {
			m.selectCursor++
		}
	case "k", "up":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case " ", "space":
		m.toggleSelectOption()
	case "v", "enter":
		if !m.validateSelection() {
			return m, nil
		}
		m.view = MCPViewPreview
		return m.requestPlanPreview()
	case "esc":
		m.view = MCPViewOverview
		m.selectNotice = ""
	}
	return m, nil
}

// toggleSelectOption toggles checkboxes and cycles radio options in Select.
func (m *MCPModel) toggleSelectOption() {
	switch m.selectCursor {
	case 0:
		m.selected["codegraph"] = !m.selected["codegraph"]
	case 1:
		m.selected["context7"] = !m.selected["context7"]
	case 2:
		if m.authChoices["context7"] == "oauth" {
			m.authChoices["context7"] = "api-key"
		} else {
			m.authChoices["context7"] = "oauth"
		}
	case 3:
		m.selected["exa"] = !m.selected["exa"]
	case 4:
		switch m.authChoices["exa"] {
		case "keyless":
			m.authChoices["exa"] = "oauth"
		case "oauth":
			m.authChoices["exa"] = "api-key"
		default:
			m.authChoices["exa"] = "keyless"
		}
	case 5:
		m.exaAcknowledged = !m.exaAcknowledged
	}
}

// validateSelection verifies prerequisite and acknowledgement constraints.
func (m *MCPModel) validateSelection() bool {
	if m.selected["exa"] && m.authChoices["exa"] == "keyless" && !m.exaAcknowledged {
		m.selectNotice = "Exa keyless requires acknowledging the visible rate limits and privacy terms."
		return false
	}
	m.selectNotice = ""
	return true
}

// handlePreviewKey routes keys on the preview view.
func (m MCPModel) handlePreviewKey(key string) (MCPModel, tea.Cmd) {
	if m.isFiltering {
		switch key {
		case "esc":
			m.isFiltering = false
			m.filter = ""
		case "enter":
			m.isFiltering = false
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
		default:
			if len(key) == 1 {
				m.filter += key
			}
		}
		return m, nil
	}

	switch key {
	case "/":
		m.isFiltering = true
	case "enter", "y":
		if len(m.planPreview.Conflicts) > 0 {
			return m, nil // blocked by conflicts
		}
		m.view = MCPViewConfirm
		m.confirmSelection = false // default No!
	case "j", "down":
		m.previewScroll++
	case "k", "up":
		if m.previewScroll > 0 {
			m.previewScroll--
		}
	case "esc":
		m.view = MCPViewSelect
	}
	return m, nil
}

// handleConfirmKey routes keys inside the focus-trapped confirmation subview.
func (m MCPModel) handleConfirmKey(key string) (MCPModel, tea.Cmd) {
	if m.tinyGeometry {
		if key == "esc" {
			m.view = MCPViewPreview
		}
		return m, nil
	}

	switch key {
	case "tab", "h", "l", "left", "right":
		m.confirmSelection = !m.confirmSelection
	case "n":
		m.confirmSelection = false
	case "y":
		m.confirmSelection = true
		return m.confirmApplyCmd()
	case "enter":
		if !m.confirmSelection {
			// No selected -> cancel
			m.view = MCPViewPreview
			return m, nil
		}
		return m.confirmApplyCmd()
	case "esc":
		m.view = MCPViewPreview
		m.confirmSelection = false
	}
	return m, nil
}

// confirmApplyCmd builds and delivers the confirmed apply write request.
func (m MCPModel) confirmApplyCmd() (MCPModel, tea.Cmd) {
	if m.planPreview.Token == "" {
		m.outcome = mcpWriteOutcome{Headline: "no preview token", Detail: "A validated preview is required before a write."}
		m.view = MCPViewPreview
		return m, nil
	}
	m.writePending = true
	m.writeKind = WriteMCPApply
	m.view = MCPViewPreview
	m.confirmSelection = false

	req := WriteRequest{
		Kind: WriteMCPApply,
		MCP: MCPWriteRequest{
			Home:         m.cfg.Home,
			ConfigPath:   m.planPreview.TargetPath,
			PlanID:       m.planPreview.PlanDigest,
			Plan:         m.planPreview.RawPlan,
			Token:        m.planPreview.Token,
			ExpectDigest: m.planPreview.BaseDigest,
			Label:        fmt.Sprintf("configure %d MCP tool(s)", len(m.planPreview.Selected)),
			Source:       "mcp:" + m.planPreview.TargetPath,
		},
		Token: m.planPreview.Token,
	}
	return m, func() tea.Msg {
		return StartWriteMsg{Request: req}
	}
}

// handleDiagnosticsKey routes keys on the diagnostics view.
func (m MCPModel) handleDiagnosticsKey(key string) (MCPModel, tea.Cmd) {
	switch key {
	case "r":
		return m.requestDoctor()
	case "j", "down":
		m.doctorScroll++
	case "k", "up":
		if m.doctorScroll > 0 {
			m.doctorScroll--
		}
	case "esc":
		if m.prevView != 0 {
			m.view = m.prevView
		} else {
			m.view = MCPViewOverview
		}
	}
	return m, nil
}

// handleRecoveryKey routes keys on the recovery view.
func (m MCPModel) handleRecoveryKey(key string) (MCPModel, tea.Cmd) {
	if m.confirmRestore {
		if m.tinyGeometry {
			if key == "esc" {
				m.confirmRestore = false
			}
			return m, nil
		}
		switch key {
		case "tab", "left", "right":
			m.confirmSelection = !m.confirmSelection
		case "n":
			m.confirmSelection = false
		case "y":
			m.confirmSelection = true
			return m.confirmRestoreCmd()
		case "enter":
			if !m.confirmSelection {
				m.confirmRestore = false
				return m, nil
			}
			return m.confirmRestoreCmd()
		case "esc":
			m.confirmRestore = false
		}
		return m, nil
	}

	if m.inRestorePreview {
		switch key {
		case "r", "enter":
			m.confirmRestore = true
			m.confirmSelection = false // default No!
		case "j", "down":
			m.recoveryScroll++
		case "k", "up":
			if m.recoveryScroll > 0 {
				m.recoveryScroll--
			}
		case "esc":
			m.inRestorePreview = false
		}
		return m, nil
	}

	// Backup list view
	switch key {
	case "j", "down":
		if m.backupIndex < len(m.backups)-1 {
			m.backupIndex++
		}
	case "k", "up":
		if m.backupIndex > 0 {
			m.backupIndex--
		}
	case "v", "enter":
		if len(m.backups) > 0 && m.backupIndex >= 0 && m.backupIndex < len(m.backups) {
			b := m.backups[m.backupIndex]
			return m.requestRestorePreview(b.ID)
		}
	case "esc":
		m.view = MCPViewOverview
	}
	return m, nil
}

// confirmRestoreCmd builds and delivers the confirmed restore write request.
func (m MCPModel) confirmRestoreCmd() (MCPModel, tea.Cmd) {
	if m.restorePreview.Token == "" || m.restorePreview.BackupID == "" {
		m.outcome = mcpWriteOutcome{Headline: "no restore preview", Detail: "Preview a backup before restoring."}
		m.confirmRestore = false
		return m, nil
	}
	m.writePending = true
	m.writeKind = WriteMCPRestore
	m.confirmRestore = false

	req := WriteRequest{
		Kind: WriteMCPRestore,
		MCP: MCPWriteRequest{
			Home:         m.cfg.Home,
			ConfigPath:   m.restorePreview.TargetPath,
			BackupID:     m.restorePreview.BackupID,
			Token:        m.restorePreview.Token,
			ExpectDigest: m.restorePreview.BaseDigest,
			Label:        fmt.Sprintf("restore MCP backup %s", m.restorePreview.BackupID),
			Source:       "mcp:" + m.restorePreview.TargetPath,
		},
		Token: m.restorePreview.Token,
	}
	return m, func() tea.Msg {
		return StartWriteMsg{Request: req}
	}
}

// requestPlanPreview starts one asynchronous plan preview calculation.
func (m MCPModel) requestPlanPreview() (MCPModel, tea.Cmd) {
	m.planLoading = true
	m.planGen++
	gen := m.planGen
	planner := m.planner
	home := m.cfg.Home
	configPath := m.configPath

	selected := make([]string, 0, len(m.selected))
	for name, ok := range m.selected {
		if ok {
			selected = append(selected, name)
		}
	}
	sort.Strings(selected)

	auth := make(map[string]string, len(m.authChoices))
	for k, v := range m.authChoices {
		auth[k] = v
	}

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		preview, err := planner.PreviewPlan(ctx, home, configPath, selected, auth)
		return MCPPlanPreviewMsg{
			Generation: gen,
			Preview:    preview,
			Err:        err,
		}
	}
}

// requestDoctor starts one asynchronous diagnostic read.
func (m MCPModel) requestDoctor() (MCPModel, tea.Cmd) {
	m.doctorLoading = true
	m.doctorGen++
	gen := m.doctorGen
	reader := m.reader
	home := m.cfg.Home
	configPath := m.configPath
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		doc, err := reader.Doctor(ctx, home, configPath, "", "", "")
		return MCPDoctorMsg{
			Generation: gen,
			Doctor:     doc,
			Err:        err,
		}
	}
}

// requestBackupList starts one asynchronous backup list read.
func (m MCPModel) requestBackupList() (MCPModel, tea.Cmd) {
	m.recoveryLoading = true
	m.recoveryGen++
	gen := m.recoveryGen
	planner := m.planner
	home := m.cfg.Home
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		backups, err := planner.ListBackups(ctx, home)
		return MCPBackupListMsg{
			Generation: gen,
			Backups:    backups,
			Err:        err,
		}
	}
}

// requestRestorePreview starts one asynchronous restore preview read.
func (m MCPModel) requestRestorePreview(backupID string) (MCPModel, tea.Cmd) {
	m.restoreLoading = true
	m.restoreGen++
	gen := m.restoreGen
	planner := m.planner
	home := m.cfg.Home
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		preview, err := planner.PreviewRestore(ctx, home, backupID)
		return MCPRestorePreviewMsg{
			Generation: gen,
			Preview:    preview,
			Err:        err,
		}
	}
}
