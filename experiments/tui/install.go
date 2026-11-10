package tui

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// InstallStep represents the phase of the installation journey.
type InstallStep int

const (
	// InstallStepClients is the initial step for selecting coding client destinations.
	InstallStepClients InstallStep = iota
	// InstallStepOptions is the step for selecting optional OpenCode pack and presets.
	InstallStepOptions
	// InstallStepReview is the review of resolved source, home, and destinations.
	InstallStepReview
	// InstallStepPreview is the exact, inspectable read-only impact preview.
	InstallStepPreview
	// InstallStepConfirm is the default-No confirmation step.
	InstallStepConfirm
	// InstallStepRunning indicates an apply operation is actively executing.
	InstallStepRunning
	// InstallStepResult shows the final outcome of the installation.
	InstallStepResult
)

// String returns a human-readable title for the installation step.
func (s InstallStep) String() string {
	switch s {
	case InstallStepClients:
		return "Clients"
	case InstallStepOptions:
		return "Options"
	case InstallStepReview:
		return "Review"
	case InstallStepPreview:
		return "Preview"
	case InstallStepConfirm:
		return "Confirm"
	case InstallStepRunning:
		return "Running"
	case InstallStepResult:
		return "Result"
	default:
		return "Unknown"
	}
}

// InstallOutcome describes the definite outcome reached by the install workflow.
type InstallOutcome string

const (
	InstallOutcomeNone      InstallOutcome = ""
	InstallOutcomeApplied   InstallOutcome = "applied"
	InstallOutcomeNoChange  InstallOutcome = "no-change"
	InstallOutcomeRefused   InstallOutcome = "refused"
	InstallOutcomeFailed    InstallOutcome = "failed"
	InstallOutcomeCancelled InstallOutcome = "cancelled"
)

// PreviewOp is the operation seam for generating an installation preview.
type PreviewOp func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error)

// InspectOp is the operation seam for inspecting selection inventory without writes.
type InspectOp func(repo, home string, opts skillsync.Options) (skillsync.SelectionInventory, error)

// InstallPreviewMsg carries the asynchronous result of a preview operation.
type InstallPreviewMsg struct {
	OpID   appops.OperationID
	Result appops.PreviewResult
	Err    error
}

// InstallInspectMsg carries the asynchronous result of an inspect operation.
type InstallInspectMsg struct {
	OpID      appops.OperationID
	Inventory skillsync.SelectionInventory
	Err       error
}

// InstallModel is the preview-first installation screen model for the nested TUI.
type InstallModel struct {
	cfg               TUIConfig
	theme             Theme
	step              InstallStep
	outcome           InstallOutcome
	home              string
	repo              string
	supportedClients  []string
	selectedClients   map[string]bool
	clientCursor      int
	withOpenCode      bool
	freeFirst         bool
	optionsCursor     int
	confirmNoSelected bool
	scroll            int
	width             int
	height            int
	validationErr     string

	previewLoading bool
	previewResult  *appops.PreviewResult
	previewToken   string
	applyLoading   bool
	applyResult    *appops.ApplyResult
	err            error
	notice         string
	logPath        string

	// writeStep and writeDetail mirror the root host's honest phase updates
	// while a transaction runs. They never carry a percentage.
	writeStep   WriteStep
	writeDetail string

	opID      appops.OperationID
	previewOp PreviewOp
	inspectOp InspectOp
}

// NewInstallModel builds the installation journey model.
// Exactly the canonical supported client IDs from skillsync.SupportedClientIDs
// are selectable, and by default all supported clients are selected.
func NewInstallModel(cfg TUIConfig) InstallModel {
	cfg = cfg.Normalize()
	theme := NewTheme(cfg.Color)

	supported := skillsync.SupportedClientIDs()
	selected := make(map[string]bool, len(supported))
	for _, id := range supported {
		selected[id] = true
	}

	return InstallModel{
		cfg:               cfg,
		theme:             theme,
		step:              InstallStepClients,
		outcome:           InstallOutcomeNone,
		home:              cfg.Home,
		repo:              cfg.Repo,
		supportedClients:  supported,
		selectedClients:   selected,
		clientCursor:      0,
		withOpenCode:      false,
		freeFirst:         false,
		optionsCursor:     0,
		confirmNoSelected: true,
		width:             cfg.Width,
		height:            cfg.Height,
		opID:              appops.OperationID{Request: 1, Generation: 0},
		previewOp:         appops.PreviewInstall,
		inspectOp:         skillsync.InspectSelection,
	}
}

// Init implements tea.Model.
func (m InstallModel) Init() tea.Cmd {
	return nil
}

// Step returns the current installation step.
func (m InstallModel) Step() InstallStep {
	return m.step
}

// Outcome returns the final outcome of the installation.
func (m InstallModel) Outcome() InstallOutcome {
	return m.outcome
}

// SelectedClients returns the selected client IDs in canonical order.
func (m InstallModel) SelectedClients() []string {
	var list []string
	for _, id := range m.supportedClients {
		if m.selectedClients[id] {
			list = append(list, id)
		}
	}
	return list
}

// WithOpenCode reports whether the OpenCode pack is selected.
func (m InstallModel) WithOpenCode() bool {
	return m.withOpenCode
}

// OpenCodeFreeFirst reports whether the Free-first preset is selected.
func (m InstallModel) OpenCodeFreeFirst() bool {
	return m.freeFirst
}

// PreviewResult returns the latest preview result, or nil.
func (m InstallModel) PreviewResult() *appops.PreviewResult {
	return m.previewResult
}

// ApplyResult returns the latest apply result, or nil.
func (m InstallModel) ApplyResult() *appops.ApplyResult {
	return m.applyResult
}

// PreviewToken returns the current active preview authorization token.
func (m InstallModel) PreviewToken() string {
	return m.previewToken
}

// Error returns the last recorded operation error, if any.
func (m InstallModel) Error() error {
	return m.err
}

// ValidationError returns the current input validation message, if any.
func (m InstallModel) ValidationError() string {
	return m.validationErr
}

// ConfirmNoSelected reports whether No is currently selected in confirmation.
func (m InstallModel) ConfirmNoSelected() bool {
	return m.confirmNoSelected
}

// IsRunning reports whether an apply write operation is actively in flight.
func (m InstallModel) IsRunning() bool {
	return m.step == InstallStepRunning && m.applyLoading
}

// IsLoading reports whether an async preview or apply is in flight.
func (m InstallModel) IsLoading() bool {
	return m.previewLoading || m.applyLoading
}

// SetSize updates the viewport dimensions.
func (m *InstallModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetHome sets the home directory and invalidates any existing preview/confirmation.
func (m *InstallModel) SetHome(home string) {
	if m.home != home {
		m.home = home
		m.invalidateConfirmation()
	}
}

// SetRepo sets the source repository path and invalidates any existing preview/confirmation.
func (m *InstallModel) SetRepo(repo string) {
	if m.repo != repo {
		m.repo = repo
		m.invalidateConfirmation()
	}
}

// SetClients configures the client selection explicitly.
// An empty slice is rejected with an error; unknown client IDs fail immediately.
func (m *InstallModel) SetClients(ids []string) error {
	if len(ids) == 0 {
		return errors.New("client selection is empty; at least one client is required")
	}
	normalized, err := skillsync.NormalizeClientSelection(ids)
	if err != nil {
		return err
	}
	m.selectedClients = make(map[string]bool, len(normalized))
	for _, id := range normalized {
		m.selectedClients[id] = true
	}
	m.invalidateConfirmation()
	m.validationErr = ""
	return nil
}

// SetOptions configures the OpenCode pack and Free-first preset options.
// Free-first requires WithOpenCode; attempting to enable it without the pack fails.
func (m *InstallModel) SetOptions(withOpenCode, freeFirst bool) error {
	if freeFirst && !withOpenCode {
		return errors.New("--opencode-free-first requires --with-opencode")
	}
	if m.withOpenCode != withOpenCode || m.freeFirst != freeFirst {
		m.withOpenCode = withOpenCode
		m.freeFirst = freeFirst
		m.invalidateConfirmation()
	}
	m.validationErr = ""
	return nil
}

// SetPreviewOp overrides the preview operation seam (useful for fakes).
func (m *InstallModel) SetPreviewOp(op PreviewOp) {
	if op == nil {
		m.previewOp = appops.PreviewInstall
	} else {
		m.previewOp = op
	}
}

// SetInspectOp overrides the inspect operation seam (useful for fakes).
func (m *InstallModel) SetInspectOp(op InspectOp) {
	if op == nil {
		m.inspectOp = skillsync.InspectSelection
	} else {
		m.inspectOp = op
	}
}

// invalidateConfirmation clears the preview token and forces a fresh preview
// when any configuration option, source, or target changes.
func (m *InstallModel) invalidateConfirmation() {
	m.previewToken = ""
	m.previewResult = nil
	m.confirmNoSelected = true
	if m.step == InstallStepConfirm || m.step == InstallStepPreview {
		m.step = InstallStepReview
	}
}

// PreviewCmd generates a tea.Cmd for executing the preview asynchronously.
func (m *InstallModel) PreviewCmd() tea.Cmd {
	return m.previewCmd()
}

func (m *InstallModel) previewCmd() tea.Cmd {
	m.opID.Generation++
	opID := m.opID
	m.previewLoading = true
	m.previewResult = nil
	m.previewToken = ""
	m.err = nil
	m.validationErr = ""

	selected := m.SelectedClients()
	if len(selected) == 0 {
		return func() tea.Msg {
			return InstallPreviewMsg{
				OpID: opID,
				Err:  errors.New("client selection is empty; at least one client is required"),
			}
		}
	}

	opts := skillsync.Options{
		ClientIDs:         selected,
		WithOpenCode:      m.withOpenCode,
		OpenCodeFreeFirst: m.freeFirst,
	}

	req, err := appops.NewInstallRequest(m.repo, m.home, false, opts)
	if err != nil {
		return func() tea.Msg {
			return InstallPreviewMsg{
				OpID: opID,
				Err:  err,
			}
		}
	}

	op := m.previewOp
	return func() tea.Msg {
		res, err := op(req, io.Discard)
		return InstallPreviewMsg{
			OpID:   opID,
			Result: res,
			Err:    err,
		}
	}
}

// Update handles tea messages and key routing.
func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Focus / resize preserves safe No confirmation
		return m, nil

	case InstallPreviewMsg:
		if msg.OpID.StaleFor(m.opID) {
			// Discard stale async result
			return m, nil
		}
		m.previewLoading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.previewResult = nil
			m.previewToken = ""
			return m, nil
		}
		m.previewResult = &msg.Result
		m.previewToken = msg.Result.PreviewToken
		m.err = nil
		m.notice = ""
		m.scroll = 0
		return m, nil

	case WriteStartedMsg:
		if m.step != InstallStepRunning || msg.Kind != WriteInstall {
			return m, nil
		}
		m.applyLoading = true
		m.notice = ""
		return m, nil

	case WriteProgressMsg:
		if m.step != InstallStepRunning || msg.Kind != WriteInstall {
			return m, nil
		}
		m.writeStep = msg.Step
		m.writeDetail = msg.Detail
		return m, nil

	case WriteRefusedMsg:
		// A refusal never started a transaction, so the preview token stays
		// valid and the user can retry the same confirmation.
		if m.step != InstallStepRunning || msg.Kind != WriteInstall {
			return m, nil
		}
		m.applyLoading = false
		m.writeStep = ""
		m.writeDetail = ""
		m.step = InstallStepConfirm
		m.confirmNoSelected = true
		m.notice = msg.Reason
		return m, nil

	case WriteResultMsg:
		if m.step != InstallStepRunning || msg.Kind != WriteInstall {
			return m, nil
		}
		m.applyLoading = false
		m.writeStep = ""
		m.writeDetail = ""
		m.logPath = msg.LogPath

		if errors.Is(msg.Err, appops.ErrNeedsPreview) {
			// The engine revalidated the token and refused to write. Return to
			// the read-only preview without a token so a fresh one is required.
			m.step = InstallStepPreview
			m.previewResult = nil
			m.previewToken = ""
			m.applyResult = nil
			m.err = msg.Err
			m.outcome = InstallOutcomeNone
			m.scroll = 0
			m.notice = "The apply was refused because inputs, source identity or planned destinations changed after the preview. Nothing was written; run a fresh preview and confirm again."
			return m, nil
		}

		m.applyResult = &msg.Result
		m.err = msg.Err
		m.step = InstallStepResult
		m.outcome = classifyInstallOutcome(msg.Result, msg.Err)
		m.scroll = 0
		if msg.LogErr != nil {
			m.notice = "The operation log could not be flushed: " + msg.LogErr.Error()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	default:
		return m, nil
	}
}

// classifyInstallOutcome maps the structured engine result to the only states
// the screen may claim. A result without a definite success outcome is never
// reported as success.
func classifyInstallOutcome(result appops.ApplyResult, err error) InstallOutcome {
	switch {
	case errors.Is(err, appops.ErrNeedsPreview):
		return InstallOutcomeRefused
	case err != nil || result.Outcome == skillsync.OutcomeFailed:
		return InstallOutcomeFailed
	case result.Outcome == skillsync.OutcomeNoChange:
		return InstallOutcomeNoChange
	case result.Outcome == skillsync.OutcomeApplied:
		return InstallOutcomeApplied
	default:
		return InstallOutcomeFailed
	}
}

func (m InstallModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// During writes, keys cannot cancel, repeat, or abandon the transaction.
	if m.step == InstallStepRunning {
		return m, nil
	}

	switch m.step {
	case InstallStepClients:
		return m.handleKeyClients(msg)
	case InstallStepOptions:
		return m.handleKeyOptions(msg)
	case InstallStepReview:
		return m.handleKeyReview(msg)
	case InstallStepPreview:
		return m.handleKeyPreview(msg)
	case InstallStepConfirm:
		return m.handleKeyConfirm(msg)
	case InstallStepResult:
		return m.handleKeyResult(msg)
	default:
		return m, nil
	}
}

func (m InstallModel) handleKeyClients(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if keyMatches(msg, "q", "esc") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}
	if keyMatches(msg, "up", "k") {
		if m.clientCursor > 0 {
			m.clientCursor--
		}
		return m, nil
	}
	if keyMatches(msg, "down", "j") {
		if m.clientCursor < len(m.supportedClients)-1 {
			m.clientCursor++
		}
		return m, nil
	}
	if keyMatches(msg, " ", "space") {
		id := m.supportedClients[m.clientCursor]
		m.selectedClients[id] = !m.selectedClients[id]
		m.invalidateConfirmation()
		m.validationErr = ""
		return m, nil
	}
	if keyMatches(msg, "enter", "right") {
		if len(m.SelectedClients()) == 0 {
			m.validationErr = "At least one client must be selected. Explicit empty selection is invalid."
			return m, nil
		}
		m.validationErr = ""
		m.step = InstallStepOptions
		m.optionsCursor = 0
		return m, nil
	}
	return m, nil
}

func (m InstallModel) handleKeyOptions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if keyMatches(msg, "esc", "left", "b") {
		m.step = InstallStepClients
		m.validationErr = ""
		return m, nil
	}
	if keyMatches(msg, "q") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}
	if keyMatches(msg, "up", "k") {
		if m.optionsCursor > 0 {
			m.optionsCursor--
		}
		return m, nil
	}
	if keyMatches(msg, "down", "j") {
		if m.optionsCursor < 1 {
			m.optionsCursor++
		}
		return m, nil
	}
	if keyMatches(msg, " ", "space") {
		m.validationErr = ""
		if m.optionsCursor == 0 {
			m.withOpenCode = !m.withOpenCode
			if !m.withOpenCode {
				m.freeFirst = false // Free-first requires OpenCode pack
			}
			m.invalidateConfirmation()
		} else {
			if !m.withOpenCode {
				m.validationErr = "Free-first preset requires the OpenCode pack to be enabled."
			} else {
				m.freeFirst = !m.freeFirst
				m.invalidateConfirmation()
			}
		}
		return m, nil
	}
	if keyMatches(msg, "enter", "right") {
		m.validationErr = ""
		m.step = InstallStepReview
		m.scroll = 0
		return m, nil
	}
	return m, nil
}

func (m InstallModel) handleKeyReview(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if keyMatches(msg, "esc", "left", "b") {
		m.step = InstallStepOptions
		return m, nil
	}
	if keyMatches(msg, "q") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}
	if keyMatches(msg, "up", "k") {
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	}
	if keyMatches(msg, "down", "j") {
		m.scroll++
		return m, nil
	}
	if keyMatches(msg, "enter") {
		m.step = InstallStepPreview
		m.scroll = 0
		cmd := m.previewCmd()
		return m, cmd
	}
	return m, nil
}

func (m InstallModel) handleKeyPreview(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.previewLoading {
		return m, nil
	}
	if keyMatches(msg, "esc", "left", "b") {
		m.step = InstallStepReview
		return m, nil
	}
	if keyMatches(msg, "q") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}
	if keyMatches(msg, "up", "k") {
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	}
	if keyMatches(msg, "down", "j") {
		m.scroll++
		return m, nil
	}
	if keyMatches(msg, "pgup") {
		m.scroll -= 10
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	}
	if keyMatches(msg, "pgdown") {
		m.scroll += 10
		return m, nil
	}
	if keyMatches(msg, "home", "g") {
		m.scroll = 0
		return m, nil
	}
	if keyMatches(msg, "enter") {
		// Only advance if preview succeeded and produced a valid token
		if m.err != nil || m.previewResult == nil || m.previewToken == "" {
			return m, nil
		}
		m.step = InstallStepConfirm
		m.confirmNoSelected = true
		m.scroll = 0
		return m, nil
	}
	return m, nil
}

func (m InstallModel) handleKeyConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Default-No confirmation policy:
	// Only an affirmative 'y' or 'Y' confirms.
	// Enter must NOT confirm.
	// Duplicate confirmation cannot double-apply.
	// Cancel/back is safe.
	if keyMatches(msg, "y", "Y") {
		if m.previewToken == "" {
			m.outcome = InstallOutcomeRefused
			m.step = InstallStepResult
			return m, nil
		}
		req, err := m.installWriteRequest()
		if err != nil {
			m.outcome = InstallOutcomeFailed
			m.err = err
			m.step = InstallStepResult
			return m, nil
		}
		// The root operation host owns the single write slot. This screen only
		// emits the immutable request; it never calls the apply engine.
		m.confirmNoSelected = false
		m.applyLoading = true
		m.applyResult = nil
		m.err = nil
		m.notice = ""
		m.writeStep = WriteStepStarting
		m.writeDetail = ""
		m.step = InstallStepRunning
		return m, startWriteCmd(req)
	}
	if keyMatches(msg, "n", "N", "esc", "b") {
		m.step = InstallStepPreview
		m.confirmNoSelected = true
		return m, nil
	}
	if keyMatches(msg, "q") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}
	if keyMatches(msg, "enter") {
		// Enter does NOT confirm. Safe No is preserved.
		return m, nil
	}
	return m, nil
}

func (m InstallModel) handleKeyResult(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if keyMatches(msg, "r", "R") {
		// Restart workflow
		m.step = InstallStepClients
		m.outcome = InstallOutcomeNone
		m.previewResult = nil
		m.previewToken = ""
		m.applyResult = nil
		m.err = nil
		m.notice = ""
		m.logPath = ""
		m.writeStep = ""
		m.writeDetail = ""
		m.validationErr = ""
		m.confirmNoSelected = true
		m.scroll = 0
		return m, nil
	}
	if keyMatches(msg, "up", "k") {
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	}
	if keyMatches(msg, "down", "j") {
		m.scroll++
		return m, nil
	}
	return m, nil
}

// View implements tea.Model. AltScreen is enabled only for interactive runs.
func (m InstallModel) View() tea.View {
	content := m.Render(m.width, m.height)
	if !m.cfg.Color {
		content = ansi.Strip(content)
	}
	view := tea.NewView(content)
	view.AltScreen = m.cfg.Interactive
	view.WindowTitle = "Agentic-SDD Install"
	if m.cfg.Color {
		view.BackgroundColor = m.theme.CanvasColor()
		view.ForegroundColor = m.theme.DefaultForeground()
	}
	return view
}

// Render returns the paged viewport content string for width and height.
func (m InstallModel) Render(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if width < 40 || (height > 0 && height < 8) {
		lines := Paragraph("Terminal too small. Resize to at least 40x8.", width, 0)
		return strings.Join(lines, "\n")
	}

	lines := m.Lines(width)
	if height <= 0 {
		return strings.Join(lines, "\n")
	}

	rows := paginateLines(lines, height, m.scroll)
	return strings.Join(rows, "\n")
}

func paginateLines(lines []string, height, offset int) []string {
	if height <= 0 {
		return nil
	}
	maxOffset := max(0, len(lines)-height)
	offset = max(0, min(offset, maxOffset))
	rows := make([]string, height)
	copy(rows, lines[offset:min(len(lines), offset+height)])
	return rows
}

// Lines returns the full slice of wrapped lines for the current step at width.
func (m InstallModel) Lines(width int) []string {
	if width <= 0 {
		width = 80
	}
	switch m.step {
	case InstallStepClients:
		return m.clientsLines(width)
	case InstallStepOptions:
		return m.optionsLines(width)
	case InstallStepReview:
		return m.reviewLines(width)
	case InstallStepPreview:
		return m.previewLines(width)
	case InstallStepConfirm:
		return m.confirmLines(width)
	case InstallStepRunning:
		return m.runningLines(width)
	case InstallStepResult:
		return m.resultLines(width)
	default:
		return []string{"Unknown step"}
	}
}

func (m InstallModel) clientsLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Client Target Selection"))
	lines = append(lines, m.theme.Subtitle.Render("Choose coding clients to receive Agentic SDD skills"))
	lines = append(lines, "")

	if m.validationErr != "" {
		lines = append(lines, m.theme.Bad.Render("  ! "+m.validationErr))
		lines = append(lines, "")
	}

	var clientRows []string
	for i, id := range m.supportedClients {
		cursor := "  "
		if i == m.clientCursor {
			cursor = "> "
		}
		check := "[ ]"
		if m.selectedClients[id] {
			check = "[x]"
		}
		targetPath := ClientTargetDir(m.home, id)
		row := fmt.Sprintf("%s%s %-6s · %s (%s)", cursor, check, id, targetPath, ClientDisplayName(id))
		clientRows = append(clientRows, row)
	}

	lines = append(lines, Section("Supported Coding Clients", clientRows, width)...)
	lines = append(lines, "")

	notes := []string{
		"The core skill set is installed as a complete, supported unit. Client selection specifies which tool directories are managed.",
		"At least one client is required; an explicit empty selection is invalid.",
	}
	lines = append(lines, Section("Installation Unit Policy", notes, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Space: toggle · ↑/↓: move · Enter: next · Esc / q: cancel"))
	return lines
}

func (m InstallModel) optionsLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Optional Integrations"))
	lines = append(lines, m.theme.Subtitle.Render("Configure optional OpenCode profile templates and presets"))
	lines = append(lines, "")

	if m.validationErr != "" {
		lines = append(lines, m.theme.Bad.Render("  ! "+m.validationErr))
		lines = append(lines, "")
	}

	c0 := "  "
	if m.optionsCursor == 0 {
		c0 = "> "
	}
	chk0 := "[ ]"
	if m.withOpenCode {
		chk0 = "[x]"
	}

	c1 := "  "
	if m.optionsCursor == 1 {
		c1 = "> "
	}
	chk1 := "[ ]"
	if m.freeFirst {
		chk1 = "[x]"
	}

	optionsRows := []string{
		fmt.Sprintf("%s%s Install OpenCode pack (profiles & skills)", c0, chk0),
		fmt.Sprintf("    Profile destination: %s", OpenCodeAgentsDir(m.home)),
		"    (OpenCode profile destination is stated separately from core client selection)",
		"",
		fmt.Sprintf("%s%s Free-first model preset", c1, chk1),
		"    Configures agent profile templates to prioritize zero-cost free model tiers first.",
		"    (Requires OpenCode pack; template preset only, does not guarantee model availability or quota)",
	}

	lines = append(lines, Section("Optional Additions", optionsRows, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Space: toggle · ↑/↓: move · Enter: next · Esc / b: back"))
	return lines
}

func (m InstallModel) reviewLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Review Configuration"))
	lines = append(lines, m.theme.Subtitle.Render("Verify paths and options before generating an exact preview"))
	lines = append(lines, "")

	source := "Embedded portable skills (built-in)"
	if m.repo != "" {
		source = m.repo
	}

	summaryRows := []string{
		"Source repository: " + source,
		"Target home directory: " + m.home,
		"Backups directory: " + BackupRootDir(m.repo, m.home),
	}
	lines = append(lines, Section("Source & Environment", summaryRows, width)...)
	lines = append(lines, "")

	var clientRows []string
	for _, id := range m.SelectedClients() {
		clientRows = append(clientRows, fmt.Sprintf("• %-6s -> %s", id, ClientTargetDir(m.home, id)))
	}
	lines = append(lines, Section(fmt.Sprintf("Core Client Destinations (%d)", len(clientRows)), clientRows, width)...)
	lines = append(lines, "")

	var opencodeRows []string
	if m.withOpenCode {
		opencodeRows = append(opencodeRows, "Status: Enabled")
		opencodeRows = append(opencodeRows, "Profiles destination: "+OpenCodeAgentsDir(m.home))
		opencodeRows = append(opencodeRows, "(Stated separately from core client skill destinations)")
	} else {
		opencodeRows = append(opencodeRows, "Status: Disabled (core skills only)")
	}
	if m.freeFirst {
		opencodeRows = append(opencodeRows, "Preset: Free-first template preset enabled")
	} else {
		opencodeRows = append(opencodeRows, "Preset: Default (none)")
	}
	lines = append(lines, Section("OpenCode Profiles & Presets (Separate)", opencodeRows, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Enter: generate exact read-only preview · Esc / b: back to options · ↑/↓: scroll"))
	return lines
}

func (m InstallModel) previewLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Exact Preview Impact"))
	lines = append(lines, m.theme.Subtitle.Render("Read-only inspection of planned destination changes"))
	lines = append(lines, "")

	if m.previewLoading {
		loadingBody := []string{
			"Generating exact preview... inspecting sources and comparing with destinations.",
			"This operation is strictly read-only; no files are modified.",
		}
		lines = append(lines, Section("Inspection In Progress", loadingBody, width)...)
		return lines
	}

	if m.err != nil {
		errBody := []string{
			"Preview failed: " + m.err.Error(),
			"Press Esc or b to go back and check options or paths.",
		}
		lines = append(lines, Section("Preview Error", errBody, width)...)
		if m.notice != "" {
			lines = append(lines, "")
			lines = append(lines, Paragraph(m.theme.Warn.Render(m.notice), width, 0)...)
		}
		return lines
	}

	if m.previewResult == nil {
		lines = append(lines, Section("No Preview", []string{"No preview available. Press Enter in Review to generate preview."}, width)...)
		return lines
	}

	res := m.previewResult
	var additions, replacements, unchanged, removals []skillsync.ResultChange
	for _, ch := range res.Planned {
		switch ch.Action {
		case skillsync.ChangeInstall, skillsync.ChangeRestore:
			additions = append(additions, ch)
		case skillsync.ChangeReplace:
			replacements = append(replacements, ch)
		case skillsync.ChangeUpToDate:
			unchanged = append(unchanged, ch)
		case skillsync.ChangeRemove:
			removals = append(removals, ch)
		default:
			additions = append(additions, ch)
		}
	}

	summary := fmt.Sprintf("Planned impact: %d additions · %d replacements · %d unchanged", len(additions), len(replacements), len(unchanged))
	if len(removals) > 0 {
		summary += fmt.Sprintf(" · %d removals", len(removals))
	}
	lines = append(lines, m.theme.Accent.Render("  "+summary))
	lines = append(lines, "")

	if len(additions) > 0 {
		var addLines []string
		for _, ch := range additions {
			addLines = append(addLines, fmt.Sprintf("• %s (%s)", ch.ID, ch.Client))
			addLines = append(addLines, "  Path: "+ch.Path)
		}
		lines = append(lines, Section(fmt.Sprintf("Additions (%d)", len(additions)), addLines, width)...)
		lines = append(lines, "")
	}

	if len(replacements) > 0 {
		var repLines []string
		for _, ch := range replacements {
			repLines = append(repLines, fmt.Sprintf("• %s (%s)", ch.ID, ch.Client))
			repLines = append(repLines, "  Path: "+ch.Path)
			repLines = append(repLines, "  (Existing content will be saved to backup before overwrite)")
		}
		lines = append(lines, Section(fmt.Sprintf("Replacements (%d)", len(replacements)), repLines, width)...)
		lines = append(lines, "")
	}

	if len(removals) > 0 {
		var remLines []string
		for _, ch := range removals {
			remLines = append(remLines, fmt.Sprintf("• %s (%s)", ch.ID, ch.Client))
			remLines = append(remLines, "  Path: "+ch.Path)
		}
		lines = append(lines, Section(fmt.Sprintf("Removals (%d)", len(removals)), remLines, width)...)
		lines = append(lines, "")
	}

	if len(unchanged) > 0 {
		var unLines []string
		for _, ch := range unchanged {
			unLines = append(unLines, fmt.Sprintf("• %s (%s)", ch.ID, ch.Client))
			unLines = append(unLines, "  Path: "+ch.Path)
		}
		lines = append(lines, Section(fmt.Sprintf("Unchanged (%d — matches source)", len(unchanged)), unLines, width)...)
		lines = append(lines, "")
	}

	noteBody := []string{
		"This preview is read-only. Nothing has been written.",
		"Proceeding to confirmation will require affirmative confirmation ('y') before any file write.",
	}
	lines = append(lines, Section("Read-Only Notice", noteBody, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Enter: proceed to confirmation · Esc / b: back to review · ↑/↓: scroll"))
	return lines
}

func (m InstallModel) confirmLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Confirm Installation"))
	lines = append(lines, m.theme.Subtitle.Render("Review pre-execution policy and confirm application"))
	lines = append(lines, "")

	warnBody := []string{
		"Ready to apply installation with previewed changes.",
		fmt.Sprintf("Target home: %s", m.home),
		fmt.Sprintf("Selected clients: %s", strings.Join(m.SelectedClients(), ", ")),
	}
	if m.withOpenCode {
		warnBody = append(warnBody, fmt.Sprintf("OpenCode pack: enabled (profiles in %s)", OpenCodeAgentsDir(m.home)))
	}
	if m.freeFirst {
		warnBody = append(warnBody, "Preset: Free-first template preset enabled")
	}
	lines = append(lines, Section("Target Configuration", warnBody, width)...)
	lines = append(lines, "")

	policyBody := []string{
		"Writes cannot be interrupted midway once started. If you press Ctrl+C or exit during installation, the process will wait for the active write and backup to finish safely before exiting.",
		"Abrupt process termination (e.g. SIGKILL) bypasses rollback and may leave destinations in an incomplete state.",
	}
	lines = append(lines, Section("Pre-Execution Policy", policyBody, width)...)
	lines = append(lines, "")

	choiceBody := []string{
		"Proceed with installation?",
		"",
		"  [ No ]      Yes",
		"",
		"(Default is No. Only typing 'y' confirms. Enter does not confirm.)",
	}
	lines = append(lines, Section("Confirmation", choiceBody, width)...)
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.notice), width, 0)...)
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.Hint.Render("y: confirm and apply · n / Esc / b: cancel / go back · Enter: safe No (no action)"))
	return lines
}

func (m InstallModel) runningLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Applying Installation..."))
	lines = append(lines, m.theme.Subtitle.Render("Writing skills and recording backup"))
	lines = append(lines, "")

	step := m.writeStep
	if step == "" {
		step = WriteStepStarting
	}
	body := []string{
		"Revalidating the preview token and applying changes; phase: " + string(step) + ".",
		"Please wait for the operation to complete safely. Do not terminate the process.",
		"Pre-execution policy: writes cannot be interrupted midway. Ctrl+C will wait for the active write and backup to finish safely.",
	}
	if m.writeDetail != "" {
		body = append(body, "Detail: "+m.writeDetail)
	}
	lines = append(lines, Section("Write In Progress", body, width)...)
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.notice), width, 0)...)
	}
	return lines
}

func (m InstallModel) resultLines(width int) []string {
	var lines []string

	switch m.outcome {
	case InstallOutcomeApplied:
		lines = append(lines, m.theme.OK.Render("Install & Manage · Installation Applied Successfully"))
		lines = append(lines, m.theme.Subtitle.Render("All planned skills were installed"))
		lines = append(lines, "")

		var body []string
		if m.applyResult != nil {
			body = append(body, fmt.Sprintf("Applied changes: %d items", len(m.applyResult.Applied)))
			if m.applyResult.BackupID != "" {
				body = append(body, fmt.Sprintf("Backup created: %s", m.applyResult.BackupID))
			}
			if m.applyResult.BackupPath != "" {
				body = append(body, fmt.Sprintf("Backup location: %s", m.applyResult.BackupPath))
			}
		}
		body = append(body, fmt.Sprintf("Target home: %s", m.home))
		body = append(body, fmt.Sprintf("Selected clients: %s", strings.Join(m.SelectedClients(), ", ")))
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.logPath)
		}
		lines = append(lines, Section("Installation Summary", body, width)...)
		lines = append(lines, "")

		nextBody := []string{
			"Run `agentic-sdd-router` in your terminal to start planning your feature or workflow.",
			"Inspect installed skills with `agentic-sdd preview` or explore via the TUI catalog.",
		}
		lines = append(lines, Section("Recommended Next Action", nextBody, width)...)

	case InstallOutcomeNoChange:
		lines = append(lines, m.theme.Accent.Render("Install & Manage · No Changes Needed"))
		lines = append(lines, m.theme.Subtitle.Render("All destinations already match source"))
		lines = append(lines, "")

		body := []string{
			"Every planned destination was already up to date with the selected source.",
			"No files were modified, and no backup was created.",
		}
		lines = append(lines, Section("Status", body, width)...)

	case InstallOutcomeRefused:
		lines = append(lines, m.theme.Bad.Render("Install & Manage · Installation Refused"))
		lines = append(lines, m.theme.Subtitle.Render("Configuration or destinations changed since preview"))
		lines = append(lines, "")

		body := []string{
			"The apply operation was refused because configuration options, source files, or target destinations changed since the preview token was generated.",
			"Pre-execution revalidation detected drift. To protect your files, a fresh preview and confirmation are required.",
		}
		lines = append(lines, Section("Refusal Reason", body, width)...)

	case InstallOutcomeFailed:
		lines = append(lines, m.theme.Bad.Render("Install & Manage · Installation Failed"))
		lines = append(lines, m.theme.Subtitle.Render("An error occurred during installation"))
		lines = append(lines, "")

		var body []string
		if m.err != nil {
			body = append(body, "Error: "+m.err.Error())
		}
		if m.applyResult != nil {
			if m.applyResult.Rollback.Attempted {
				if m.applyResult.Rollback.Complete {
					body = append(body, "Rollback status: Succeeded. Committed changes were reverted.")
				} else {
					body = append(body, fmt.Sprintf("Rollback status: INCOMPLETE (%s). Destinations may be in a partial state.", m.applyResult.Rollback.Error))
				}
			}
			if m.applyResult.BackupID != "" {
				body = append(body, fmt.Sprintf("Pre-failure backup: %s at %s", m.applyResult.BackupID, m.applyResult.BackupPath))
			}
		}
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.logPath)
		}
		lines = append(lines, Section("Failure Details", body, width)...)

	case InstallOutcomeCancelled:
		lines = append(lines, m.theme.Warn.Render("Install & Manage · Installation Cancelled"))
		lines = append(lines, m.theme.Subtitle.Render("Operation cancelled by user"))
		lines = append(lines, "")

		body := []string{
			"The installation workflow was cancelled.",
			"No files were modified on disk.",
		}
		lines = append(lines, Section("Status", body, width)...)

	default:
		lines = append(lines, m.theme.Hint.Render("Install & Manage · Ready"))
	}

	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.notice), width, 0)...)
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.Hint.Render("Enter / Esc / q: close · r: reconfigure from beginning · ↑/↓: scroll"))
	return lines
}

// ClientDisplayName returns the user-facing label for a client ID.
func ClientDisplayName(id string) string {
	switch id {
	case "agents":
		return "Agents (Agentic SDD global)"
	case "codex":
		return "Codex (Codex CLI)"
	case "claude":
		return "Claude (Claude Code)"
	case "agy":
		return "Agy (Antigravity CLI)"
	default:
		return id
	}
}

// ClientTargetDir returns the expected destination directory for a client in home.
func ClientTargetDir(home, id string) string {
	switch id {
	case "agents":
		return filepath.Join(home, ".agents", "skills")
	case "codex":
		return filepath.Join(home, ".codex", "skills")
	case "claude":
		return filepath.Join(home, ".claude", "skills")
	case "agy":
		return filepath.Join(home, ".gemini", "antigravity-cli", "skills")
	default:
		return filepath.Join(home, "."+id, "skills")
	}
}

// OpenCodeAgentsDir returns the profile destination directory for OpenCode pack.
func OpenCodeAgentsDir(home string) string {
	return filepath.Join(home, ".config", "opencode", "agents")
}

// BackupRootDir returns the root directory for backups.
func BackupRootDir(repo, home string) string {
	if repo != "" {
		return filepath.Join(repo, "backups")
	}
	return filepath.Join(home, ".agentic-sdd", "backups")
}

func keyMatches(msg tea.KeyPressMsg, names ...string) bool {
	str := msg.String()
	for _, n := range names {
		if str == n {
			return true
		}
	}
	switch msg.Code {
	case tea.KeyEnter:
		for _, n := range names {
			if n == "enter" {
				return true
			}
		}
	case tea.KeyEsc:
		for _, n := range names {
			if n == "esc" {
				return true
			}
		}
	case tea.KeyUp:
		for _, n := range names {
			if n == "up" {
				return true
			}
		}
	case tea.KeyDown:
		for _, n := range names {
			if n == "down" {
				return true
			}
		}
	case tea.KeyLeft:
		for _, n := range names {
			if n == "left" {
				return true
			}
		}
	case tea.KeyRight:
		for _, n := range names {
			if n == "right" {
				return true
			}
		}
	case tea.KeySpace:
		for _, n := range names {
			if n == " " || n == "space" {
				return true
			}
		}
	case 'y', 'Y':
		for _, n := range names {
			if n == "y" || n == "Y" {
				return true
			}
		}
	case 'n', 'N':
		for _, n := range names {
			if n == "n" || n == "N" {
				return true
			}
		}
	case 'q', 'Q':
		for _, n := range names {
			if n == "q" || n == "Q" {
				return true
			}
		}
	case 'r', 'R':
		for _, n := range names {
			if n == "r" || n == "R" {
				return true
			}
		}
	case 'b', 'B':
		for _, n := range names {
			if n == "b" || n == "B" {
				return true
			}
		}
	case 'k', 'K':
		for _, n := range names {
			if n == "k" || n == "K" {
				return true
			}
		}
	case 'j', 'J':
		for _, n := range names {
			if n == "j" || n == "J" {
				return true
			}
		}
	case 'g':
		for _, n := range names {
			if n == "g" {
				return true
			}
		}
	}
	return false
}
