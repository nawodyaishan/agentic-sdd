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
	// InstallOutcomeUnknown means the engine returned without a definite write
	// outcome. It is never reported as success or as a failure.
	InstallOutcomeUnknown InstallOutcome = "unknown"
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
	// impactCursor and impactExpanded are presentation-only state for the
	// derived impact projection. They are reset when the preview identity
	// changes and never influence the plan, token or write request.
	impactCursor   int
	impactExpanded map[int]bool
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
	theme := newThemeForConfig(cfg)

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

// SetTheme updates the active styling theme without resetting state.
func (m *InstallModel) SetTheme(theme Theme) {
	m.theme = theme
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
	m.resetImpactPresentation()
	m.confirmNoSelected = true
	if m.step == InstallStepConfirm || m.step == InstallStepPreview {
		m.step = InstallStepReview
	}
}

// resetImpactPresentation discards grouping state. It never touches the
// authoritative plan; it only makes the next render start collapsed.
func (m *InstallModel) resetImpactPresentation() {
	m.impactCursor = 0
	m.impactExpanded = nil
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
	m.resetImpactPresentation()
	m.notice = ""
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
		m.resetImpactPresentation()
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
		return InstallOutcomeUnknown
	}
}

// appliedChangeCounts separates newly installed destinations from replacements
// using the typed applied actions only. An action this build does not know is
// not counted as either.
func appliedChangeCounts(applied []skillsync.ResultChange) (installed, updated int) {
	for _, change := range applied {
		switch change.Action {
		case skillsync.ChangeInstall:
			installed++
		case skillsync.ChangeReplace:
			updated++
		}
	}
	return installed, updated
}

// clientSelectionText renders selected clients as plain display names with
// their engine IDs kept visible, so a confirmation cannot hide the identity it
// will act on.
func clientSelectionText(ids []string) string {
	if len(ids) == 0 {
		return "none (an explicit empty selection is invalid)"
	}
	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		labels = append(labels, ClientSelectionLabel(id))
	}
	return strings.Join(labels, ", ")
}

// clientPlainNames renders clients for orientation prose, where the exact
// engine ID and destination belong in the following review/confirmation
// details instead.
func clientPlainNames(ids []string) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, ClientDisplayName(id))
	}
	return strings.Join(names, ", ")
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
				m.validationErr = "Prefer free models in the provided templates requires the OpenCode agent profiles option."
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
		if m.PreviewNoOp() {
			// A zero-change preview is not consent for a later state. Leaving
			// it discards the read-only result so returning always re-previews.
			m.invalidateConfirmation()
			return m, nil
		}
		m.step = InstallStepReview
		return m, nil
	}
	if keyMatches(msg, "q") {
		m.outcome = InstallOutcomeCancelled
		m.step = InstallStepResult
		return m, nil
	}

	noOp := m.PreviewNoOp()
	if noOp {
		// The read-only no-op state accepts only inspection and non-write
		// route actions. The App owns Home/Library navigation; these keys are
		// inert here so a standalone model can never start a write either.
		switch {
		case keyMatches(msg, "o"):
			m.step = InstallStepOptions
			m.invalidateConfirmation()
			return m, nil
		case keyMatches(msg, "enter", "y", "Y", "n", "N"):
			return m, nil
		}
	}

	if keyMatches(msg, "x") {
		m.toggleAllImpactDetails()
		return m, nil
	}
	if keyMatches(msg, "[") {
		m.moveImpactCursor(-1)
		return m, nil
	}
	if keyMatches(msg, "]") {
		m.moveImpactCursor(1)
		return m, nil
	}
	if keyMatches(msg, " ", "space") {
		m.toggleImpactGroup()
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
		// Only a complete, classifiable preview with a valid token may open
		// the default-No confirmation. A no-op, unknown or incomplete result
		// never advances.
		if noOp {
			return m, nil
		}
		if !m.previewConfirmable() {
			if m.previewProjection().hasUnknown() {
				m.notice = "Confirmation is disabled because the preview contains an action this screen cannot classify. Nothing was written; inspect the plan or use the explicit CLI."
			}
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
	// Cancel/back is safe. q cancels this journey safely and never quits
	// through the confirmation.
	if classifyConfirmKey(msg.String()) == confirmKeyIgnore {
		return m, nil
	}
	switch {
	case keyMatches(msg, "y", "Y"):
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
		m.resetImpactPresentation()
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

// data escapes Unicode source data in ASCII mode so paths, IDs, notices and
// errors stay reversible on a 7-bit terminal. It never changes the request,
// token or engine value; callers escape only at the render boundary.
func (m InstallModel) data(text string) string {
	if m.cfg.ASCII {
		return escapeData(text)
	}
	return text
}

// Lines returns the full slice of wrapped lines for the current step at width.
func (m InstallModel) Lines(width int) []string {
	if width <= 0 {
		width = 80
	}
	var lines []string
	switch m.step {
	case InstallStepClients:
		lines = m.clientsLines(width)
	case InstallStepOptions:
		lines = m.optionsLines(width)
	case InstallStepReview:
		lines = m.reviewLines(width)
	case InstallStepPreview:
		lines = m.previewLines(width)
	case InstallStepConfirm:
		lines = m.confirmLines(width)
	case InstallStepRunning:
		lines = m.runningLines(width)
	case InstallStepResult:
		lines = m.resultLines(width)
	default:
		lines = []string{"Unknown step"}
	}
	if m.cfg.ASCII {
		for i := range lines {
			lines[i] = toASCII(lines[i])
		}
	}
	return lines
}

func (m InstallModel) clientsLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Client Target Selection"))
	lines = append(lines, m.theme.Subtitle.Render("Choose which coding clients receive the shared Agentic SDD skills"))
	lines = append(lines, "")

	if m.validationErr != "" {
		lines = append(lines, m.theme.Bad.Render("  ! "+m.data(m.validationErr)))
		lines = append(lines, "")
	}

	glyphs := GlyphSet(m.cfg.ASCII)
	var clientRows []string
	for i, id := range m.supportedClients {
		cursor := glyphs.Gutter + " "
		if i == m.clientCursor {
			cursor = glyphs.Cursor + " "
		}
		check := glyphs.CheckEmpty
		state := "not selected"
		if m.selectedClients[id] {
			check = glyphs.CheckMark
			state = "selected"
		}
		targetPath := m.data(ClientTargetDir(m.home, id))
		row := fmt.Sprintf("%s%s %s — %s", cursor, check, ClientDisplayName(id), state)
		if i == m.clientCursor && m.theme.Color {
			row = m.theme.Selected.Bold(true).Render(row)
		}
		clientRows = append(clientRows, row)
		clientRows = append(clientRows, fmt.Sprintf("      id %s · destination %s", id, targetPath))
	}

	lines = append(lines, Section("Supported Coding Clients", clientRows, width)...)
	lines = append(lines, "")
	lines = append(lines, Section("Shared directory", []string{
		"Shared Agents Skills is one shared skills directory, not a provider, account or runtime.",
	}, width)...)
	lines = append(lines, "")

	notes := []string{
		"The core skill set is installed as a complete, supported unit. Client selection specifies which tool directories are managed.",
		"At least one client is required; an explicit empty selection is invalid and no pack-only installation is offered.",
	}
	lines = append(lines, Section("Installation Unit Policy", notes, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Space: toggle · ↑/↓: move · Enter: next · Esc / q: cancel"))
	return lines
}

func (m InstallModel) optionsLines(width int) []string {
	var lines []string

	lines = append(lines, m.theme.Title.Render("Install & Manage · Optional Integrations"))
	lines = append(lines, m.theme.Subtitle.Render("Configure optional OpenCode agent profiles and the free-model template preference"))
	lines = append(lines, "")

	if m.validationErr != "" {
		lines = append(lines, m.theme.Bad.Render("  ! "+m.data(m.validationErr)))
		lines = append(lines, "")
	}

	glyphs := GlyphSet(m.cfg.ASCII)
	c0 := glyphs.Gutter + " "
	if m.optionsCursor == 0 {
		c0 = glyphs.Cursor + " "
	}
	chk0 := glyphs.CheckEmpty
	if m.withOpenCode {
		chk0 = glyphs.CheckMark
	}

	c1 := glyphs.Gutter + " "
	if m.optionsCursor == 1 {
		c1 = glyphs.Cursor + " "
	}
	chk1 := glyphs.CheckEmpty
	if m.freeFirst {
		chk1 = glyphs.CheckMark
	}

	row0 := fmt.Sprintf("%s%s OpenCode agent profiles", c0, chk0)
	if m.optionsCursor == 0 && m.theme.Color {
		row0 = m.theme.Selected.Bold(true).Render(row0)
	}

	row1 := fmt.Sprintf("%s%s Prefer free models in the provided templates", c1, chk1)
	if m.optionsCursor == 1 && m.theme.Color {
		row1 = m.theme.Selected.Bold(true).Render(row1)
	}

	optionsRows := []string{
		row0,
		fmt.Sprintf("    Profiles destination: %s", m.data(OpenCodeAgentsDir(m.home))),
		"    Adds OpenCode agent profile templates and pack skills. This is separate from the core client selection above.",
		"",
		row1,
		"    A template preference only: it does not check, enable or guarantee model availability, quota, credentials or privacy.",
		"    Requires the OpenCode agent profiles option; no pack-only installation is offered.",
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
		source = m.data(m.repo)
	}

	summaryRows := []string{
		"Source repository: " + source,
		"Target home directory: " + m.data(m.home),
		"Backups directory: " + m.data(BackupRootDir(m.repo, m.home)),
	}
	lines = append(lines, Section("Source & Environment", summaryRows, width)...)
	lines = append(lines, "")

	var clientRows []string
	for _, id := range m.SelectedClients() {
		clientRows = append(clientRows, fmt.Sprintf("• %s → %s", ClientSelectionLabel(id), m.data(ClientTargetDir(m.home, id))))
	}
	lines = append(lines, Section(fmt.Sprintf("Core Client Destinations (%d)", len(clientRows)), clientRows, width)...)
	lines = append(lines, "")

	var opencodeRows []string
	if m.withOpenCode {
		opencodeRows = append(opencodeRows, "Status: enabled")
		opencodeRows = append(opencodeRows, "Profiles destination: "+m.data(OpenCodeAgentsDir(m.home)))
		opencodeRows = append(opencodeRows, "Stated separately from the core client skill destinations above.")
	} else {
		opencodeRows = append(opencodeRows, "Status: disabled (core skills only)")
	}
	if m.freeFirst {
		opencodeRows = append(opencodeRows, "Template preference: prefer free models in the provided templates (preference only, not an availability or quota guarantee)")
	} else {
		opencodeRows = append(opencodeRows, "Template preference: default (none)")
	}
	lines = append(lines, Section("OpenCode agent profiles & presets (separate)", opencodeRows, width)...)
	lines = append(lines, "")

	lines = append(lines, m.theme.Hint.Render("Enter: generate exact read-only preview · Esc / b: back to options · ↑/↓: scroll"))
	return lines
}

// previewProjection derives the immutable presentation projection from the
// current typed preview. It never modifies the plan or token.
func (m InstallModel) previewProjection() impactProjection {
	if m.previewResult == nil {
		return impactProjection{}
	}
	return projectImpact(m.previewResult.Planned)
}

// PreviewNoOp reports whether the preview is a successful, complete plan whose
// every item is explicitly up to date. Only that state offers the read-only
// Already up to date branch; an empty, refused, incomplete or unknown plan
// never does.
func (m InstallModel) PreviewNoOp() bool {
	if m.err != nil || m.previewResult == nil || m.previewToken == "" {
		return false
	}
	if m.previewResult.Outcome == skillsync.OutcomeFailed {
		return false
	}
	return m.previewProjection().noOp()
}

// previewConfirmable reports whether the preview may open the default-No
// confirmation. A failed outcome, a plan this screen cannot fully classify, or
// an incomplete result never may.
func (m InstallModel) previewConfirmable() bool {
	if m.previewLoading || m.err != nil || m.previewResult == nil || m.previewToken == "" {
		return false
	}
	if m.previewResult.Outcome == skillsync.OutcomeFailed {
		return false
	}
	return !m.previewProjection().hasUnknown()
}

// previewIncomplete reports a successful preview result that produced no
// confirmation token, which must not look like a usable plan.
func (m InstallModel) previewIncomplete() bool {
	return m.err == nil && !m.previewLoading && m.previewResult != nil && m.previewToken == ""
}

func (m InstallModel) previewSourceLabel() string {
	if m.repo != "" {
		return m.repo
	}
	return "Embedded portable skills (built-in)"
}

// previewPinnedLines is the concise summary the App keeps visible while the
// destination details page. It is empty whenever no usable plan exists.
func (m InstallModel) previewPinnedLines(width int) []string {
	if m.previewLoading || m.err != nil || m.previewResult == nil {
		return nil
	}
	lines := m.previewTitleLines(width)
	return append(lines, m.previewSummaryLines(width)...)
}

// previewTitleLines is the shared title block for the preview step.
func (m InstallModel) previewTitleLines(width int) []string {
	return []string{
		m.theme.Title.Render("Install & Manage · Exact Preview Impact"),
		m.theme.Subtitle.Render("Read-only inspection of planned destination changes"),
		"",
	}
}

// previewSummaryLines is the action-total summary. It stays first and is
// pinned by the root viewport while the details scroll.
func (m InstallModel) previewSummaryLines(width int) []string {
	if m.previewLoading || m.err != nil || m.previewResult == nil {
		return nil
	}
	projection := m.previewProjection()
	rows := Paragraph(impactSummary(projection), width, 2)
	for i := range rows {
		rows[i] = m.theme.Accent.Render(rows[i])
	}
	return rows
}

func (m InstallModel) previewLines(width int) []string {
	lines := m.previewTitleLines(width)

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
			"Preview failed: " + m.data(m.err.Error()),
			"Press Esc or b to go back and check options or paths.",
			"This is not a successful no-op: nothing was written and nothing can be confirmed from a failed preview.",
		}
		lines = append(lines, Section("Preview Error", errBody, width)...)
		if m.notice != "" {
			lines = append(lines, "")
			lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
		}
		return lines
	}

	if m.previewResult == nil {
		lines = append(lines, Section("No Preview", []string{"No preview available. Press Enter in Review to generate preview."}, width)...)
		return lines
	}

	lines = append(lines, m.previewSummaryLines(width)...)
	lines = append(lines, "")
	if m.previewResult.Outcome == skillsync.OutcomeFailed {
		lines = append(lines, Section(m.theme.Warn.Render("Preview refused"), []string{
			"The preview reported a failed outcome without an error. It is not a successful no-op and cannot be confirmed.",
			"Nothing was written. Go back and run a fresh preview; if this repeats, inspect the source and destinations with the explicit CLI.",
		}, width)...)
		lines = append(lines, "")
	}
	lines = append(lines, m.previewDetailLines(width)...)
	return lines
}

// previewDetailLines renders the destination groups and their expandable item
// and path details. It never reorders or rewrites the plan.
func (m InstallModel) previewDetailLines(width int) []string {
	projection := m.previewProjection()
	var lines []string

	if m.previewIncomplete() {
		lines = append(lines, Section(m.theme.Warn.Render("Preview incomplete"), []string{
			"This preview did not produce a confirmation token, so it is not confirmable.",
			"Nothing has been written. Go back and run a fresh preview; if this repeats, inspect the source and destinations with the explicit CLI.",
		}, width)...)
		lines = append(lines, "")
	}

	if m.PreviewNoOp() {
		lines = append(lines, m.previewNoOpLines(projection, width)...)
		lines = append(lines, "")
	} else if projection.total > 0 {
		lines = append(lines, Section("Planned destinations", []string{
			"Action totals are shown above. Select a destination group with [ ] and press Space to expand or collapse its items; x shows or hides every path.",
		}, width)...)
		lines = append(lines, "")
	}

	if projection.total == 0 {
		lines = append(lines, Section(m.theme.Warn.Render("No planned items"), []string{
			"The preview reported a successful operation but no planned destination items. This is not treated as a successful up-to-date result.",
			"Nothing was written. Use the explicit CLI preview to inspect the source, or go back and check the selected clients and options.",
		}, width)...)
		lines = append(lines, "")
	}

	if len(projection.groups) > 0 {
		lines = append(lines, m.impactGroupLines(projection, width)...)
		lines = append(lines, "")
	}

	if projection.hasUnknown() {
		lines = append(lines, Section(m.theme.Warn.Render(fmt.Sprintf("Unrecognized planned actions (%d)", projection.count(impactUnknown))), []string{
			"At least one planned action kind is not recognized by this screen. It is shown explicitly and is not treated as a new or up-to-date item.",
			"Confirmation is disabled until authoritative planning resolves it. Nothing has been written.",
		}, width)...)
		lines = append(lines, "")
	}

	lines = append(lines, Section("Preview identity", []string{
		"Source: " + m.data(m.previewSourceLabel()),
		"Target home: " + m.data(m.home),
	}, width)...)
	lines = append(lines, "")

	noteBody := []string{
		"This preview is read-only. Nothing has been written.",
	}
	if m.PreviewNoOp() {
		noteBody = append(noteBody, "Every planned destination already matches the source, so no confirmation, backup or operation log is offered here.")
	} else if !m.previewConfirmable() {
		noteBody = append(noteBody, "This preview is not confirmable from this screen. Nothing will be written.")
	} else {
		noteBody = append(noteBody, "Proceeding to confirmation will require affirmative confirmation ('y') before any file write.")
	}
	lines = append(lines, Section("Read-Only Notice", noteBody, width)...)
	lines = append(lines, "")

	if m.notice != "" {
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
		lines = append(lines, "")
	}
	lines = append(lines, m.previewHintLines(width)...)
	return lines
}

// previewNoOpLines is the read-only Already up to date panel. It is neutral,
// not a warning, and it offers inspection and route actions only.
func (m InstallModel) previewNoOpLines(projection impactProjection, width int) []string {
	body := []string{
		"Every planned destination already matches the selected source. This complete preview would change nothing, so no confirmation is offered and nothing can be written from this screen.",
		"No files will be modified; no backup and no operation log are created by this result.",
		fmt.Sprintf("Planned items: %d · changes: 0", projection.total),
	}
	return Section(m.theme.Accent.Render("Already up to date"), body, width)
}

// previewHintLines documents the currently eligible read-only actions. The
// root contextual footer shows the same vocabulary at a narrower budget.
func (m InstallModel) previewHintLines(width int) []string {
	if m.PreviewNoOp() {
		return Paragraph(m.theme.Hint.Render("h Home · l Library · o change options · x view paths · Esc/b back · q quit"), width, 0)
	}
	if !m.previewConfirmable() {
		return Paragraph(m.theme.Hint.Render("Esc/b back to review · x view paths · ↑/↓: scroll · q quit"), width, 0)
	}
	return Paragraph(m.theme.Hint.Render("Enter: confirmation · x view paths · [ ]/Space group details · Esc/b back · ↑/↓: scroll · q quit"), width, 0)
}

// impactGroupLines renders one row per typed destination and, when expanded,
// its original items with complete paths.
func (m InstallModel) impactGroupLines(projection impactProjection, width int) []string {
	glyphs := GlyphSet(m.cfg.ASCII)
	var lines []string
	for i, group := range projection.groups {
		marker := glyphs.Gutter
		if i == m.impactCursor {
			marker = glyphs.Cursor
		}
		state := "[+]"
		if m.impactExpanded[i] {
			state = "[-]"
		}
		head := fmt.Sprintf("%s %s %s", marker, state, impactGroupSummary(group, m.data))
		if i == m.impactCursor && m.theme.Color {
			head = m.theme.Selected.Bold(true).Render(head)
		}
		lines = append(lines, Paragraph(head, width, 0)...)
		if !m.impactExpanded[i] {
			continue
		}
		for _, item := range group.items {
			lines = append(lines, Paragraph(fmt.Sprintf("• %s — %s", m.data(impactItemLabel(item)), impactItemIdentity(item, m.data)), width, 4)...)
			lines = append(lines, Paragraph("Path: "+m.data(item.change.Path), width, 6)...)
		}
	}
	return lines
}

func (m *InstallModel) moveImpactCursor(delta int) {
	groups := len(projectImpact(m.plannedOrNil()).groups)
	if groups == 0 {
		return
	}
	m.impactCursor += delta
	if m.impactCursor < 0 {
		m.impactCursor = 0
	}
	if m.impactCursor >= groups {
		m.impactCursor = groups - 1
	}
}

func (m InstallModel) plannedOrNil() []skillsync.ResultChange {
	if m.previewResult == nil {
		return nil
	}
	return m.previewResult.Planned
}

func (m *InstallModel) toggleImpactGroup() {
	groups := len(projectImpact(m.plannedOrNil()).groups)
	if groups == 0 {
		return
	}
	if m.impactCursor < 0 || m.impactCursor >= groups {
		m.impactCursor = 0
	}
	if m.impactExpanded == nil {
		m.impactExpanded = make(map[int]bool)
	}
	m.impactExpanded[m.impactCursor] = !m.impactExpanded[m.impactCursor]
}

func (m *InstallModel) toggleAllImpactDetails() {
	groups := len(projectImpact(m.plannedOrNil()).groups)
	if groups == 0 {
		return
	}
	allExpanded := m.impactExpanded != nil
	for i := 0; i < groups; i++ {
		if !m.impactExpanded[i] {
			allExpanded = false
			break
		}
	}
	if allExpanded {
		m.impactExpanded = nil
		return
	}
	m.impactExpanded = make(map[int]bool, groups)
	for i := 0; i < groups; i++ {
		m.impactExpanded[i] = true
	}
}

// modalSection renders one confirmation section with a themed heading. The
// root modal compositor frames and pages it; the standalone Render path uses
// the same parts so wording cannot diverge.
func modalSection(theme Theme, title string, rows []string, width int) []string {
	lines := Section(title, rows, width)
	if len(lines) > 0 && theme.Color {
		lines[0] = theme.Section.Render(lines[0])
	}
	return lines
}

// modalParts splits the confirmation into the pinned warning and decision
// policy plus the scrollable target/impact body that the root compositor owns.
// No handler, token or consent state is involved.
func (m InstallModel) modalParts(width int) modalParts {
	warnBody := []string{
		"Ready to apply installation with previewed changes.",
		fmt.Sprintf("Target home: %s", m.data(m.home)),
		fmt.Sprintf("Selected clients: %s", clientSelectionText(m.SelectedClients())),
	}
	for _, id := range m.SelectedClients() {
		warnBody = append(warnBody, fmt.Sprintf("• %s → %s", ClientSelectionLabel(id), m.data(ClientTargetDir(m.home, id))))
	}
	if m.withOpenCode {
		warnBody = append(warnBody, fmt.Sprintf("OpenCode agent profiles: enabled (destination %s)", m.data(OpenCodeAgentsDir(m.home))))
	}
	if m.freeFirst {
		warnBody = append(warnBody, "Template preference: prefer free models in the provided templates (preference only)")
	}

	policyBody := []string{
		"Writes cannot be interrupted midway once started. If you press Ctrl+C or exit during installation, the process will wait for the active write and backup to finish safely before exiting.",
		"Abrupt process termination (e.g. SIGKILL) bypasses rollback and may leave destinations in an incomplete state.",
	}
	choiceBody := []string{
		"Proceed with installation?",
		"",
		"  [ No ]      Yes",
		"",
		"(Default is No. Only typing 'y' confirms; 'Y' is the same explicit key. Enter does not confirm.)",
	}

	body := modalSection(m.theme, "Target Configuration", warnBody, width)
	policy := modalSection(m.theme, "Confirmation", choiceBody, width)
	if m.notice != "" {
		policy = append(policy, "")
		policy = append(policy, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}

	return modalParts{
		Title:    "Install & Manage · Confirm Installation",
		Subtitle: "Review pre-execution policy and confirm application",
		Warning:  modalSection(m.theme, "Pre-Execution Policy", policyBody, width),
		Policy:   policy,
		Body:     body,
		Footer:   "y: confirm and apply · n / Esc / b: cancel / go back · Enter: safe No (no action)",
	}
}

func (m InstallModel) confirmLines(width int) []string {
	parts := m.modalParts(width)

	var lines []string
	lines = append(lines, m.theme.Title.Render(parts.Title))
	lines = append(lines, m.theme.Subtitle.Render(parts.Subtitle))
	lines = append(lines, "")
	lines = append(lines, parts.Warning...)
	lines = append(lines, "")
	lines = append(lines, parts.Body...)
	lines = append(lines, "")
	lines = append(lines, parts.Policy...)
	lines = append(lines, "")
	lines = append(lines, m.theme.Hint.Render(parts.Footer))
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
		body = append(body, "Detail: "+m.data(m.writeDetail))
	}
	lines = append(lines, Section("Write In Progress", body, width)...)
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}
	return lines
}

func (m InstallModel) resultLines(width int) []string {
	var lines []string

	switch m.outcome {
	case InstallOutcomeApplied:
		lines = append(lines, m.theme.OK.Render("Install & Manage · Installation Applied Successfully"))
		lines = append(lines, m.theme.Subtitle.Render("The planned destinations were written"))
		lines = append(lines, "")

		var body []string
		if m.applyResult != nil {
			installed, updated := appliedChangeCounts(m.applyResult.Applied)
			switch {
			case len(m.applyResult.Applied) == 0:
				body = append(body, "The engine reported success without listing the changed items; inspect the destinations before relying on this result.")
			default:
				body = append(body, fmt.Sprintf("Applied changes: %d items (%d new, %d updated)", len(m.applyResult.Applied), installed, updated))
				if installed > 0 {
					body = append(body, fmt.Sprintf("New destinations installed: %d", installed))
				}
				if updated > 0 {
					body = append(body, fmt.Sprintf("Existing content updated (backed up first): %d", updated))
				}
				if other := len(m.applyResult.Applied) - installed - updated; other > 0 {
					body = append(body, fmt.Sprintf("Applied items with an unrecognized action (not counted as new or updated): %d", other))
				}
			}
			if m.applyResult.BackupID != "" {
				body = append(body, fmt.Sprintf("Backup created: %s", m.data(m.applyResult.BackupID)))
			}
			if m.applyResult.BackupPath != "" {
				body = append(body, fmt.Sprintf("Backup location: %s", m.data(m.applyResult.BackupPath)))
			}
		} else {
			body = append(body, "The engine reported success without a structured change list; inspect the destinations before relying on this result.")
		}
		body = append(body, fmt.Sprintf("Target home: %s", m.data(m.home)))
		body = append(body, fmt.Sprintf("Selected clients: %s", clientSelectionText(m.SelectedClients())))
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.data(m.logPath))
		}
		lines = append(lines, Section("Installation Summary", body, width)...)
		lines = append(lines, "")

		nextBody := []string{
			"Optional next step: run `agentic-sdd-router` in your terminal to start planning your feature or workflow.",
			"Inspect installed skills with `agentic-sdd preview` or explore via the TUI catalog.",
		}
		lines = append(lines, Section("Recommended Next Action", nextBody, width)...)
		lines = append(lines, "")
		lines = append(lines, Section(PostSetupSectionTitle, PostSetupBody(), width)...)

	case InstallOutcomeNoChange:
		lines = append(lines, m.theme.Accent.Render("Install & Manage · Already Current"))
		lines = append(lines, m.theme.Subtitle.Render("All destinations already match the selected source"))
		lines = append(lines, "")

		body := []string{
			"Every planned destination was already up to date with the selected source.",
			"No files were modified, and no backup was created.",
		}
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.data(m.logPath))
		}
		lines = append(lines, Section("Status", body, width)...)

	case InstallOutcomeRefused:
		lines = append(lines, m.theme.Bad.Render("Install & Manage · Installation Refused"))
		lines = append(lines, m.theme.Subtitle.Render("Configuration or destinations changed since preview"))
		lines = append(lines, "")

		body := []string{
			"The apply operation was refused because configuration options, source files, or target destinations changed since the preview token was generated.",
			"Pre-execution revalidation detected drift. To protect your files, a fresh preview and confirmation are required.",
			"Target home: " + m.data(m.home),
			"Selected clients: " + clientSelectionText(m.SelectedClients()),
		}
		lines = append(lines, Section("Refusal Reason", body, width)...)

	case InstallOutcomeFailed:
		title := "Install & Manage · Installation Failed"
		if m.applyResult != nil && m.applyResult.Rollback.Attempted {
			if m.applyResult.Rollback.Complete {
				title = "Install & Manage · Installation Failed (rolled back)"
			} else {
				title = "Install & Manage · Installation Failed (incomplete)"
			}
		}
		lines = append(lines, m.theme.Bad.Render(title))
		lines = append(lines, m.theme.Subtitle.Render("An error occurred during installation"))
		lines = append(lines, "")

		var body []string
		if m.err != nil {
			body = append(body, "Error: "+m.data(m.err.Error()))
		}
		if m.applyResult != nil {
			if m.applyResult.Rollback.Attempted {
				if m.applyResult.Rollback.Complete {
					body = append(body, "Rollback status: Succeeded. Committed changes were reverted.")
				} else {
					body = append(body, fmt.Sprintf("Rollback status: INCOMPLETE (%s). Destinations may be in a partial state.", m.data(m.applyResult.Rollback.Error)))
				}
			} else {
				body = append(body, "Rollback status: not attempted. No committed change is claimed; inspect the destinations if you are unsure.")
			}
			if m.applyResult.BackupID != "" {
				body = append(body, fmt.Sprintf("Pre-failure backup: %s at %s", m.data(m.applyResult.BackupID), m.data(m.applyResult.BackupPath)))
			}
		}
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.data(m.logPath))
		}
		body = append(body, "Target home: "+m.data(m.home))
		body = append(body, "Selected clients: "+clientSelectionText(m.SelectedClients()))
		body = append(body, "Recovery: inspect recorded backups under Install & Manage → Backups & restore (Esc to close, then 2 → b). No restore starts without its own preview and confirmation.")
		lines = append(lines, Section("Failure Details", body, width)...)

	case InstallOutcomeUnknown:
		lines = append(lines, m.theme.Warn.Render("Install & Manage · Installation Result Unknown"))
		lines = append(lines, m.theme.Subtitle.Render("The engine returned without a definite write outcome"))
		lines = append(lines, "")

		body := []string{
			"This screen does not claim success or failure. Inspect the destinations and the operation log before retrying.",
		}
		if m.applyResult != nil && m.applyResult.BackupID != "" {
			body = append(body, fmt.Sprintf("Backup reference: %s at %s", m.data(m.applyResult.BackupID), m.data(m.applyResult.BackupPath)))
		}
		if m.logPath != "" {
			body = append(body, "Operation log: "+m.data(m.logPath))
		}
		body = append(body, "Target home: "+m.data(m.home))
		body = append(body, "Selected clients: "+clientSelectionText(m.SelectedClients()))
		body = append(body, "Recovery: inspect the destinations and recorded backups under Install & Manage → Backups & restore before retrying. No restore starts without its own preview and confirmation.")
		lines = append(lines, Section("Unknown Result", body, width)...)

	case InstallOutcomeCancelled:
		lines = append(lines, m.theme.Warn.Render("Install & Manage · Installation Cancelled"))
		lines = append(lines, m.theme.Subtitle.Render("Cancelled before any write"))
		lines = append(lines, "")

		body := []string{
			"The installation workflow was cancelled before the confirmed write started.",
			"No files were modified on disk.",
		}
		lines = append(lines, Section("Status", body, width)...)

	default:
		lines = append(lines, m.theme.Hint.Render("Install & Manage · Ready"))
	}

	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.Hint.Render("Enter / Esc / q: close · r: reconfigure from beginning · ↑/↓: scroll"))
	return lines
}

// ClientDisplayName returns the plain, user-facing name for a client ID. The
// engine ID stays the identity: destinations, requests and logs keep using it.
func ClientDisplayName(id string) string {
	switch id {
	case "agents":
		return "Shared Agents Skills"
	case "codex":
		return "Codex CLI"
	case "claude":
		return "Claude Code"
	case "agy":
		return "Antigravity CLI"
	default:
		return id
	}
}

// ClientSelectionLabel returns the display name plus the engine ID for the
// review and confirmation surfaces, where both the plain name and the exact
// identity must stay visible.
func ClientSelectionLabel(id string) string {
	name := ClientDisplayName(id)
	if name == id {
		return id
	}
	return name + " (" + id + ")"
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
