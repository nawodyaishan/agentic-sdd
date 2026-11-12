package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// FleetView selects the active sub-view within Fleet Manager.
type FleetView int

const (
	FleetViewOverview FleetView = iota
	FleetViewProfiles
	FleetViewReview
	FleetViewRecovery
)

func (v FleetView) String() string {
	switch v {
	case FleetViewOverview:
		return "Overview"
	case FleetViewProfiles:
		return "Profiles"
	case FleetViewReview:
		return "Review Changes"
	case FleetViewRecovery:
		return "Recovery"
	default:
		return "Overview"
	}
}

// FleetStatusMsg delivers the result of an asynchronous fleet status read.
type FleetStatusMsg struct {
	Generation uint64
	Source     string
	Status     FleetStatusView
	Err        error
}

// FleetModel manages view state and read-only interaction for Fleet Manager.
type FleetModel struct {
	cfg    TUIConfig
	theme  Theme
	reader FleetReader
	editor FleetEditor
	width  int
	height int

	// Navigation & state
	view       FleetView
	sourcePath string
	generation uint64
	loading    bool
	loadErr    string
	status     FleetStatusView
	hasLoaded  bool

	// Profiles view state
	selectedIndex int
	filter        string
	isFiltering   bool
	detailOpen    bool // compact drill-down toggle
	scrollOffset  int
	detailScroll  int
	focusDetail   bool // in side-by-side mode (width >= 110)

	// Overview source editing state
	editingSource bool
	sourceInput   string

	// Draft state in memory (FR-07)
	draft          map[string]FleetModelEdit
	baseDigest     string
	dirtyCount     int
	sourcePreview  FleetSourcePreview
	previewGen     uint64
	previewLoading bool
	previewErr     string

	// Model Picker state (FR-04)
	editingModel bool
	modelInput   string
	pickerIndex  int
	pickerErr    string

	// Enablement consequence dialog & dirty protection (FR-05, FR-07)
	confirmDisable  bool
	disableTargetID string
	disableWarning  string

	confirmDiscard bool

	// Review view scroll offset
	reviewScroll int

	// Read-only deployment planner/restorer seam. Nil keeps a truthful
	// preview-only default; the screens still never write.
	planner FleetPlanner

	// Deployment review state (W2-C).
	planPreview   FleetPlanPreview
	planGen       uint64
	planLoading   bool
	planErr       string
	planError     error
	confirmApply  bool
	applyOutcome  fleetWriteOutcome
	sourceOutcome fleetWriteOutcome

	// Recovery state (W2-C).
	backups         []FleetBackupView
	backupIndex     int
	recoveryGen     uint64
	recoveryLoading bool
	recoveryErr     string
	recoveryError   error
	recoveryScroll  int

	restorePreview FleetRestorePreview
	restoreGen     uint64
	restoreLoading bool
	restoreErr     string
	restoreError   error
	restoreOutcome fleetWriteOutcome
	confirmRestore bool

	// Shared single-slot write bookkeeping. writePending is set when this
	// screen dispatches StartWriteMsg and cleared only by a real host message.
	writePending bool
	writeKind    WriteKind
	writeStep    WriteStep

	// Opt-in local diagnostics panel (W3-A / T20). Contextual, not a route:
	// doctorOpen replaces the current view until Esc returns to it.
	doctor          FleetDoctor
	doctorOpen      bool
	doctorField     int
	doctorBinInput  string
	doctorDumpInput string
	doctorGen       uint64
	doctorLoading   bool
	doctorRan       bool
	doctorErr       string
	doctorError     error
	doctorView      FleetDoctorView
	doctorScroll    int
}

// NewFleetModel constructs a FleetModel with safe read-only defaults.
// It performs no file reads or network calls during construction.
func NewFleetModel(cfg TUIConfig, reader FleetReader) FleetModel {
	return NewFleetModelWithSeams(cfg, reader, DefaultFleetEditor{}, DefaultFleetPlanner{})
}

// NewFleetModelWithEditor constructs a FleetModel with an explicit reader and
// editor and the safe default planner.
func NewFleetModelWithEditor(cfg TUIConfig, reader FleetReader, editor FleetEditor) FleetModel {
	return NewFleetModelWithSeams(cfg, reader, editor, DefaultFleetPlanner{})
}

// NewFleetModelWithSeams constructs a FleetModel with explicit read-only seams.
// Any nil seam is replaced with a truthful not-available default, never a
// writer.
func NewFleetModelWithSeams(cfg TUIConfig, reader FleetReader, editor FleetEditor, planner FleetPlanner) FleetModel {
	if reader == nil {
		reader = DefaultFleetReader{}
	}
	if editor == nil {
		editor = DefaultFleetEditor{}
	}
	if planner == nil {
		planner = DefaultFleetPlanner{}
	}
	source := cfg.FleetConfig
	return FleetModel{
		cfg:        cfg,
		theme:      newThemeForConfig(cfg),
		reader:     reader,
		editor:     editor,
		planner:    planner,
		doctor:     DefaultFleetDoctor{},
		view:       FleetViewOverview,
		sourcePath: source,
		generation: 1,
		width:      cfg.Width,
		height:     cfg.Height,
		draft:      make(map[string]FleetModelEdit),
	}
}

// SetPlanner updates the read-only deployment/recovery seam.
func (m *FleetModel) SetPlanner(planner FleetPlanner) {
	if planner == nil {
		planner = DefaultFleetPlanner{}
	}
	m.planner = planner
}

// SetEditor updates the preview editor seam.
func (m *FleetModel) SetEditor(editor FleetEditor) {
	if editor == nil {
		editor = DefaultFleetEditor{}
	}
	m.editor = editor
}

// SetTheme updates the rendering theme without touching state.
func (m *FleetModel) SetTheme(theme Theme) {
	m.theme = theme
}

// SetSize updates the viewport geometry allocated to Fleet Manager.
func (m *FleetModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// ActiveView reports the current active sub-view (Overview or Profiles).
func (m FleetModel) ActiveView() FleetView {
	return m.view
}

// View satisfies the tea.Model interface.
func (m FleetModel) View() tea.View {
	return tea.NewView(strings.Join(m.Lines(m.width), "\n"))
}

// SetView explicitly sets the active sub-view.
func (m *FleetModel) SetView(v FleetView) {
	m.view = v
	m.detailOpen = false
	m.focusDetail = false
}

// Breadcrumb returns the human-readable navigation breadcrumb for Fleet Manager.
func (m FleetModel) Breadcrumb() string {
	return "Install & Manage / Fleet Manager"
}

// BreadcrumbSegments returns the breadcrumb path components.
func (m FleetModel) BreadcrumbSegments() []string {
	return []string{"Install & Manage", "Fleet Manager"}
}

// IsInputFocused reports whether a text entry field or modal prompt is currently capturing keystrokes.
func (m FleetModel) IsInputFocused() bool {
	return m.isFiltering || m.editingSource || m.editingModel || m.confirmDisable || m.confirmDiscard ||
		m.confirmApply || m.confirmRestore || m.doctorOpen
}

// WantsHome reports whether key is an explicit request to jump to the Home route.
func (m FleetModel) WantsHome(key string) bool {
	if m.IsInputFocused() {
		return false
	}
	return key == "H"
}

// WantsClose reports whether key should close Fleet Manager and return to the Install & Manage hub.
func (m FleetModel) WantsClose(key string) bool {
	if m.IsInputFocused() || m.detailOpen || m.writePending {
		return false
	}
	if m.dirtyCount > 0 {
		return false
	}
	if key == "esc" && m.view == FleetViewOverview {
		return true
	}
	return false
}

// isWide reports whether the Profiles view should render in side-by-side mode.
// In standalone views, width >= 110 qualifies. Inside the App shell (where the
// sidebar consumes 20 columns at terminal width >= 110), the allocated child
// content width is >= 90 while cfg.Width is >= 110.
func (m FleetModel) isWide(width int) bool {
	if width >= 110 {
		return true
	}
	if width >= 90 && m.cfg.Width >= 110 {
		return true
	}
	return false
}

// ShouldScrollBody reports whether key should scroll the parent content viewport.
func (m FleetModel) ShouldScrollBody(key string) bool {
	if m.IsInputFocused() {
		return false
	}
	if m.view == FleetViewOverview {
		return isScrollKey(key)
	}
	if m.view == FleetViewProfiles && m.detailOpen && !m.isWide(m.width) {
		return isScrollKey(key)
	}
	return false
}

// Status returns the latest status snapshot.
func (m FleetModel) Status() FleetStatusView {
	return m.status
}

// SourcePath returns the session-configured source path.
func (m FleetModel) SourcePath() string {
	return m.sourcePath
}

// Generation returns the current read generation sequence number.
func (m FleetModel) Generation() uint64 {
	return m.generation
}

// IsLoading reports whether an asynchronous status read is in flight.
func (m FleetModel) IsLoading() bool {
	return m.loading
}

// Draft returns a shallow copy of the staged in-memory edits.
func (m FleetModel) Draft() map[string]FleetModelEdit {
	out := make(map[string]FleetModelEdit, len(m.draft))
	for k, v := range m.draft {
		out[k] = v
	}
	return out
}

// DirtyCount returns the number of unsaved in-memory edits.
func (m FleetModel) DirtyCount() int {
	return m.dirtyCount
}

// BaseDigest returns the CAS base digest from loaded status or preview.
func (m FleetModel) BaseDigest() string {
	return m.baseDigest
}

// SourcePreview returns the current staged source preview.
func (m FleetModel) SourcePreview() FleetSourcePreview {
	return m.sourcePreview
}

// IsEditingModel reports whether the Model Picker overlay is active.
func (m FleetModel) IsEditingModel() bool {
	return m.editingModel
}

// IsConfirmingDisable reports whether the disable consequence dialog is active.
func (m FleetModel) IsConfirmingDisable() bool {
	return m.confirmDisable
}

// IsConfirmingDiscard reports whether the unsaved-edits discard prompt is active.
func (m FleetModel) IsConfirmingDiscard() bool {
	return m.confirmDiscard
}

// IsConfirmingApply reports whether the default-No deployment confirmation is active.
func (m FleetModel) IsConfirmingApply() bool { return m.confirmApply }

// IsConfirmingRestore reports whether the default-No restore confirmation is active.
func (m FleetModel) IsConfirmingRestore() bool { return m.confirmRestore }

// PlanPreview returns the current read-only deployment projection.
func (m FleetModel) PlanPreview() FleetPlanPreview { return m.planPreview }

// RestorePreview returns the current read-only restore projection.
func (m FleetModel) RestorePreview() FleetRestorePreview { return m.restorePreview }

// Backups returns the current managed backup list.
func (m FleetModel) Backups() []FleetBackupView {
	return append([]FleetBackupView(nil), m.backups...)
}

// ApplyOutcome returns the real last apply outcome, if one completed.
func (m FleetModel) ApplyOutcome() fleetWriteOutcome { return m.applyOutcome }

// RestoreOutcome returns the real last restore outcome, if one completed.
func (m FleetModel) RestoreOutcome() fleetWriteOutcome { return m.restoreOutcome }

// LiveEnabled reports the reviewed build capability.
func (m FleetModel) LiveEnabled() bool { return m.live() }

// Edits returns a slice of all staged in-memory edits sorted by InstanceID.
func (m FleetModel) Edits() []FleetModelEdit {
	edits := make([]FleetModelEdit, 0, len(m.draft))
	for _, e := range m.draft {
		edits = append(edits, e)
	}
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].InstanceID < edits[j].InstanceID
	})
	return edits
}

// StageModel stages a model override for the given instance ID.
func (m *FleetModel) StageModel(instanceID, model string) error {
	if err := ValidateModelSyntax(model); err != nil {
		return err
	}
	var target FleetInstanceView
	found := false
	for _, inst := range m.status.Instances {
		if inst.InstanceID == instanceID {
			target = inst
			found = true
			break
		}
	}
	if !found {
		target = FleetInstanceView{
			InstanceID:       instanceID,
			SourceProvenance: "personal override",
			Enabled:          true,
		}
	}
	m.stageModel(target, model)
	return nil
}

// StageToggle stages an enablement toggle for the given instance ID.
func (m *FleetModel) StageToggle(instanceID string) error {
	var target FleetInstanceView
	found := false
	for _, inst := range m.status.Instances {
		if inst.InstanceID == instanceID {
			target = inst
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("instance %q not found", instanceID)
	}
	currentEnabled := target.Enabled
	if edit, ok := m.draft[instanceID]; ok {
		currentEnabled = edit.Enabled
	}
	if currentEnabled && strings.Contains(strings.ToLower(target.InstalledState), "unmanaged") {
		return errors.New("cannot disable: unmanaged instance cannot be removed by fleet toggle")
	}
	m.stageToggle(target, !currentEnabled)
	return nil
}

// DiscardEdits clears all staged draft edits and resets dirty state.
func (m *FleetModel) DiscardEdits() {
	m.draft = make(map[string]FleetModelEdit)
	m.dirtyCount = 0
	m.sourcePreview = FleetSourcePreview{}
	m.confirmDiscard = false
}

func (m *FleetModel) stageModel(selected FleetInstanceView, model string) {
	enabled := selected.Enabled
	if existing, ok := m.draft[selected.InstanceID]; ok {
		enabled = existing.Enabled
	}
	m.draft[selected.InstanceID] = FleetModelEdit{
		InstanceID: selected.InstanceID,
		Source:     selected.SourceProvenance,
		Model:      model,
		Enabled:    enabled,
	}
	m.dirtyCount = len(m.draft)
}

func (m *FleetModel) stageToggle(selected FleetInstanceView, enabled bool) {
	model := selected.Model
	if existing, ok := m.draft[selected.InstanceID]; ok {
		model = existing.Model
	}
	m.draft[selected.InstanceID] = FleetModelEdit{
		InstanceID: selected.InstanceID,
		Source:     selected.SourceProvenance,
		Model:      model,
		Enabled:    enabled,
	}
	m.dirtyCount = len(m.draft)
}

func (m *FleetModel) openModelPicker() {
	selected, ok := m.SelectedInstance()
	if !ok {
		return
	}
	m.editingModel = true
	m.modelInput = ""
	m.pickerIndex = 0
	m.pickerErr = ""
	if edit, hasEdit := m.draft[selected.InstanceID]; hasEdit && edit.Model != "" {
		m.modelInput = edit.Model
	}
}

func (m FleetModel) pickerSuggestions() []string {
	selected, ok := m.SelectedInstance()
	currentModel := ""
	if ok {
		currentModel = selected.Model
		if edit, has := m.draft[selected.InstanceID]; has && edit.Model != "" {
			currentModel = edit.Model
		}
	}
	suggestions := KnownModelSuggestions(m.status.Instances, currentModel, m.modelInput)
	if m.modelInput != "" && ValidateModelSyntax(m.modelInput) == nil {
		found := false
		for _, s := range suggestions {
			if s == m.modelInput {
				found = true
				break
			}
		}
		if !found {
			suggestions = append([]string{m.modelInput}, suggestions...)
		}
	}
	return suggestions
}

func (m FleetModel) requestSourcePreview() (FleetModel, tea.Cmd) {
	if m.editor == nil {
		return m, nil
	}
	m.previewGen++
	m.previewLoading = true
	m.previewErr = ""
	gen := m.previewGen
	source := m.sourcePath
	editor := m.editor
	edits := m.Edits()
	cmd := func() tea.Msg {
		preview, err := editor.PreviewSource(context.Background(), source, edits)
		return FleetSourcePreviewMsg{
			Generation: gen,
			Source:     source,
			Preview:    preview,
			Err:        err,
		}
	}
	return m, cmd
}

// FilteredInstances returns the list of instances matching the current filter.
func (m FleetModel) FilteredInstances() []FleetInstanceView {
	instances := m.status.Instances
	if strings.TrimSpace(m.filter) == "" {
		return instances
	}
	q := strings.ToLower(strings.TrimSpace(m.filter))
	var out []FleetInstanceView
	for _, inst := range instances {
		if strings.Contains(strings.ToLower(inst.InstanceID), q) ||
			strings.Contains(strings.ToLower(inst.Role), q) ||
			strings.Contains(strings.ToLower(inst.ProviderRoute), q) ||
			strings.Contains(strings.ToLower(inst.Model), q) ||
			strings.Contains(strings.ToLower(inst.CompactModel), q) ||
			strings.Contains(strings.ToLower(inst.InstalledState), q) ||
			strings.Contains(strings.ToLower(inst.SourceProvenance), q) {
			out = append(out, inst)
		}
	}
	return out
}

// SelectedInstance returns the currently selected instance, if any.
func (m FleetModel) SelectedInstance() (FleetInstanceView, bool) {
	filtered := m.FilteredInstances()
	if len(filtered) == 0 {
		return FleetInstanceView{}, false
	}
	if m.selectedIndex < 0 {
		return filtered[0], true
	}
	if m.selectedIndex >= len(filtered) {
		return filtered[len(filtered)-1], true
	}
	return filtered[m.selectedIndex], true
}

// Init begins an asynchronous read if the model has not yet loaded.
func (m FleetModel) Init() tea.Cmd {
	if m.hasLoaded || m.loading {
		return nil
	}
	_, cmd := m.Reload()
	return cmd
}

// EnsureLoaded begins an asynchronous read if the model has not yet loaded,
// returning the updated model state (incremented generation and loading flag)
// alongside the command.
func (m FleetModel) EnsureLoaded() (FleetModel, tea.Cmd) {
	if m.hasLoaded || m.loading {
		return m, nil
	}
	return m.Reload()
}

// Reload queues a fresh asynchronous status read with an incremented generation.
func (m FleetModel) Reload() (FleetModel, tea.Cmd) {
	if m.reader == nil {
		return m, nil
	}
	m.generation++
	m.loading = true
	m.loadErr = ""
	gen := m.generation
	source := m.sourcePath
	reader := m.reader
	cmd := func() tea.Msg {
		status, err := reader.Status(context.Background(), source)
		return FleetStatusMsg{
			Generation: gen,
			Source:     source,
			Status:     status,
			Err:        err,
		}
	}
	return m, cmd
}

// Update handles incoming messages and keyboard input.
func (m FleetModel) Update(msg tea.Msg) (FleetModel, tea.Cmd) {
	switch msg := msg.(type) {
	case FleetStatusMsg:
		// Stale reply check: generation or source path mismatch is discarded.
		if msg.Generation != m.generation || msg.Source != m.sourcePath {
			return m, nil
		}
		m.loading = false
		m.hasLoaded = true
		if msg.Err != nil {
			m.loadErr = redactError(msg.Err)
			if msg.Status.SourcePath != "" || msg.Status.SourceState != "" {
				m.status = msg.Status
			}
			return m, nil
		}
		m.status = msg.Status
		m.loadErr = ""
		filtered := m.FilteredInstances()
		if m.selectedIndex >= len(filtered) {
			m.selectedIndex = max(0, len(filtered)-1)
		}
		return m, nil

	case FleetSourcePreviewMsg:
		// Stale reply check: generation or source path mismatch is discarded.
		// Never clear draft or overwrite a newer preview.
		if msg.Generation != m.previewGen || msg.Source != m.sourcePath {
			return m, nil
		}
		m.previewLoading = false
		if msg.Err != nil {
			m.previewErr = msg.Err.Error()
			return m, nil
		}
		m.sourcePreview = msg.Preview
		if msg.Preview.BaseDigest != "" {
			m.baseDigest = msg.Preview.BaseDigest
		} else if msg.Preview.OldDigest != "" {
			m.baseDigest = msg.Preview.OldDigest
		}
		m.previewErr = ""
		return m, nil

	case FleetPlanPreviewMsg:
		// A stale generation or a different source is discarded: it cannot
		// replace a newer plan or resurrect a consumed one.
		if msg.Generation != m.planGen || msg.Source != m.sourcePath {
			return m, nil
		}
		m.planLoading = false
		if msg.Err != nil {
			m.planErr = redactError(msg.Err)
			m.planError = msg.Err
			m.planPreview = FleetPlanPreview{}
			return m, nil
		}
		m.planErr = ""
		m.planError = nil
		m.planPreview = msg.Preview
		m.reviewScroll = 0
		return m, nil

	case FleetBackupsMsg:
		if msg.Generation != m.recoveryGen {
			return m, nil
		}
		m.recoveryLoading = false
		if msg.Err != nil {
			m.recoveryErr = redactError(msg.Err)
			m.recoveryError = msg.Err
			m.backups = nil
			m.backupIndex = 0
			m.restorePreview = FleetRestorePreview{}
			m.restoreErr = ""
			m.restoreError = nil
			return m, nil
		}
		m.recoveryErr = ""
		m.recoveryError = nil
		m.backups = append([]FleetBackupView(nil), msg.Backups...)
		if len(m.backups) == 0 {
			m.backupIndex = 0
		} else if m.backupIndex >= len(m.backups) {
			m.backupIndex = len(m.backups) - 1
		}
		if m.backupIndex < 0 {
			m.backupIndex = 0
		}
		m.recoveryScroll = 0
		return m, nil

	case FleetRestorePreviewMsg:
		// A reply for another backup or an older preview request is stale.
		selected, _ := m.selectedBackup()
		if msg.Generation != m.restoreGen || msg.BackupID != selected.ID {
			return m, nil
		}
		m.restoreLoading = false
		if msg.Err != nil {
			m.restoreErr = redactError(msg.Err)
			m.restoreError = msg.Err
			m.restorePreview = FleetRestorePreview{}
			return m, nil
		}
		m.restoreErr = ""
		m.restoreError = nil
		m.restorePreview = msg.Preview
		m.recoveryScroll = 0
		return m, nil

	case FleetDoctorMsg:
		// A reply for an older explicit run is discarded; it can never replace
		// a newer report or clear the typed paths.
		if msg.Generation != m.doctorGen {
			return m, nil
		}
		m.doctorLoading = false
		m.doctorScroll = 0
		if msg.Err != nil {
			m.doctorErr = redactError(msg.Err)
			m.doctorError = msg.Err
			return m, nil
		}
		m.doctorErr = ""
		m.doctorError = nil
		m.doctorRan = true
		m.doctorView = msg.View
		return m, nil

	case WriteStartedMsg:
		if !msg.Kind.isFleet() {
			return m, nil
		}
		m.writePending = true
		m.writeKind = msg.Kind
		m.writeStep = WriteStepStarting
		return m, nil

	case WriteProgressMsg:
		if !msg.Kind.isFleet() || !m.writePending {
			return m, nil
		}
		m.writeStep = msg.Step
		return m, nil

	case WriteResultMsg:
		if !msg.Kind.isFleet() {
			return m, nil
		}
		return m.handleFleetWriteResult(msg)

	case WriteRefusedMsg:
		if !msg.Kind.isFleet() {
			return m, nil
		}
		m.writePending = false
		m.writeStep = ""
		outcome := fleetWriteOutcome{Headline: "write refused", Detail: nonEmpty(msg.Reason, "The write was refused before it started.")}
		switch msg.Kind {
		case WriteFleetApply:
			m.applyOutcome = outcome
			m.confirmApply = false
		case WriteFleetRestore:
			m.restoreOutcome = outcome
			m.confirmRestore = false
		case WriteSourceSave:
			m.sourceOutcome = outcome
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m FleetModel) handleKey(msg tea.KeyPressMsg) (FleetModel, tea.Cmd) {
	key := msg.String()

	// 1. Source path input mode (Overview)
	if m.editingSource {
		switch key {
		case "esc":
			m.editingSource = false
			m.sourceInput = m.sourcePath
			return m, nil
		case "enter":
			m.editingSource = false
			newPath := strings.TrimSpace(m.sourceInput)
			if newPath != m.sourcePath {
				m.previewGen++
				m.sourcePreview = FleetSourcePreview{}
				m.baseDigest = ""
				m.planGen++
				m.planPreview = FleetPlanPreview{}
				m.planErr = ""
				m.planError = nil
				m.planLoading = false
				m.confirmApply = false
				m.applyOutcome = fleetWriteOutcome{}
				m.recoveryGen++
				m.backups = nil
				m.recoveryErr = ""
				m.recoveryError = nil
				m.recoveryLoading = false
				m.restoreGen++
				m.restorePreview = FleetRestorePreview{}
				m.restoreErr = ""
				m.restoreError = nil
				m.restoreLoading = false
			}
			m.sourcePath = newPath
			m.hasLoaded = false
			return m.Reload()
		case "backspace":
			if len(m.sourceInput) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.sourceInput)
				m.sourceInput = m.sourceInput[:len(m.sourceInput)-size]
			}
			return m, nil
		case "ctrl+u":
			m.sourceInput = ""
			return m, nil
		default:
			if len(msg.Text) > 0 {
				m.sourceInput += msg.Text
			} else if len(key) == 1 {
				m.sourceInput += key
			}
			return m, nil
		}
	}

	// 2. Filter input mode (Profiles)
	if m.isFiltering {
		switch key {
		case "esc":
			m.isFiltering = false
			return m, nil
		case "enter":
			m.isFiltering = false
			return m, nil
		case "backspace":
			if len(m.filter) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.filter)
				m.filter = m.filter[:len(m.filter)-size]
				m.clampSelection()
			}
			return m, nil
		case "ctrl+u":
			m.filter = ""
			m.clampSelection()
			return m, nil
		case "down":
			filtered := m.FilteredInstances()
			if m.selectedIndex < len(filtered)-1 {
				m.selectedIndex++
			}
			return m, nil
		case "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil
		default:
			if len(msg.Text) > 0 {
				m.filter += msg.Text
				m.clampSelection()
			} else if len(key) == 1 {
				m.filter += key
				m.clampSelection()
			}
			return m, nil
		}
	}

	// 3. Model Picker input mode (Profiles)
	if m.editingModel {
		switch key {
		case "esc":
			m.editingModel = false
			m.pickerErr = ""
			m.modelInput = ""
			return m, nil
		case "enter":
			selectedModel := strings.TrimSpace(m.modelInput)
			suggestions := m.pickerSuggestions()
			if selectedModel == "" && len(suggestions) > 0 && m.pickerIndex >= 0 && m.pickerIndex < len(suggestions) {
				selectedModel = suggestions[m.pickerIndex]
			}
			if err := ValidateModelSyntax(selectedModel); err != nil {
				m.pickerErr = err.Error()
				return m, nil
			}
			selected, ok := m.SelectedInstance()
			if !ok {
				m.editingModel = false
				return m, nil
			}
			m.stageModel(selected, selectedModel)
			m.editingModel = false
			m.pickerErr = ""
			m.modelInput = ""
			return m.requestSourcePreview()
		case "backspace":
			if len(m.modelInput) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.modelInput)
				m.modelInput = m.modelInput[:len(m.modelInput)-size]
				m.pickerIndex = 0
				m.pickerErr = ""
			}
			return m, nil
		case "ctrl+u":
			m.modelInput = ""
			m.pickerIndex = 0
			m.pickerErr = ""
			return m, nil
		case "down":
			suggestions := m.pickerSuggestions()
			if len(suggestions) > 0 && m.pickerIndex < len(suggestions)-1 {
				m.pickerIndex++
			}
			return m, nil
		case "up":
			if m.pickerIndex > 0 {
				m.pickerIndex--
			}
			return m, nil
		case "tab":
			suggestions := m.pickerSuggestions()
			if len(suggestions) > 0 && m.pickerIndex >= 0 && m.pickerIndex < len(suggestions) {
				m.modelInput = suggestions[m.pickerIndex]
				m.pickerIndex = 0
				m.pickerErr = ""
			}
			return m, nil
		default:
			if len(msg.Text) > 0 {
				m.modelInput += msg.Text
				m.pickerIndex = 0
				m.pickerErr = ""
			} else if len(key) == 1 {
				m.modelInput += key
				m.pickerIndex = 0
				m.pickerErr = ""
			}
			return m, nil
		}
	}

	// 4. Disable consequence confirmation dialog (Profiles)
	if m.confirmDisable {
		switch key {
		case "y", "Y", "enter":
			selected, ok := m.SelectedInstance()
			if ok && selected.InstanceID == m.disableTargetID {
				m.stageToggle(selected, false)
			}
			m.confirmDisable = false
			m.disableTargetID = ""
			return m.requestSourcePreview()
		case "n", "N", "esc":
			m.confirmDisable = false
			m.disableTargetID = ""
			return m, nil
		default:
			return m, nil
		}
	}

	// 5. Dirty protection discard prompt
	if m.confirmDiscard {
		switch key {
		case "k", "K":
			m.confirmDiscard = false
			if m.view == FleetViewOverview {
				m.view = FleetViewProfiles
			}
			return m, nil
		case "s", "S", "esc":
			m.confirmDiscard = false
			return m, nil
		case "d", "D":
			m.DiscardEdits()
			if m.view == FleetViewProfiles {
				m.view = FleetViewOverview
			}
			return m, nil
		default:
			return m, nil
		}
	}

	// 5a. Deployment apply confirmation. Only an explicit y confirms; Enter is
	// the default No, so a stray keypress can never write. Below the usable
	// geometry the confirmation is not visible, so only cancellation is
	// accepted.
	if m.confirmApply {
		if m.width < 60 || m.height < 8 {
			switch key {
			case "n", "N", "enter", "esc":
				m.confirmApply = false
			}
			return m, nil
		}
		switch key {
		case "y", "Y":
			return m.confirmApplyAndDispatch()
		case "n", "N", "enter", "esc":
			m.confirmApply = false
			return m, nil
		default:
			return m, nil
		}
	}

	// 5b. Managed-backup restore confirmation, same default-No policy.
	if m.confirmRestore {
		if m.width < 60 || m.height < 8 {
			switch key {
			case "n", "N", "enter", "esc":
				m.confirmRestore = false
			}
			return m, nil
		}
		switch key {
		case "y", "Y":
			return m.confirmRestoreAndDispatch()
		case "n", "N", "enter", "esc":
			m.confirmRestore = false
			return m, nil
		default:
			return m, nil
		}
	}

	// 5c. Contextual local diagnostics panel. It owns every key while open so
	// printable characters are literal path text.
	if m.doctorOpen {
		return m.handleDoctorKey(msg)
	}

	// 6. Normal navigation keys
	switch key {
	case "1", "o", "O":
		m.view = FleetViewOverview
		m.detailOpen = false
		m.focusDetail = false
		m.disableWarning = ""
		return m, nil

	case "2", "p", "P":
		m.view = FleetViewProfiles
		m.disableWarning = ""
		return m, nil

	case "3", "c", "C":
		m.view = FleetViewReview
		m.detailOpen = false
		m.focusDetail = false
		m.disableWarning = ""
		m.reviewScroll = 0
		if m.dirtyCount == 0 && !m.planPreview.Available && !m.planLoading {
			return m.requestPlanPreview()
		}
		return m, nil

	case "4", "e", "E":
		m.view = FleetViewRecovery
		m.detailOpen = false
		m.focusDetail = false
		m.disableWarning = ""
		if len(m.backups) == 0 && !m.recoveryLoading {
			return m.requestBackups()
		}
		return m, nil

	case "tab":
		if m.view == FleetViewProfiles && m.isWide(m.width) {
			m.focusDetail = !m.focusDetail
			return m, nil
		}
		if m.view == FleetViewOverview {
			m.view = FleetViewProfiles
		} else {
			m.view = FleetViewOverview
			m.detailOpen = false
		}
		return m, nil

	case "r", "R":
		if m.view == FleetViewRecovery {
			return m.requestBackups()
		}
		return m.Reload()

	case "?":
		// Help overlay is handled by root shell
		return m, nil
	}

	// Sub-view specific keys
	if m.view == FleetViewOverview {
		switch key {
		case "s", "S":
			m.editingSource = true
			m.sourceInput = m.sourcePath
			return m, nil
		case "d", "D":
			// Contextual local diagnostics, offline by default.
			return m.openDoctor(), nil
		case "enter":
			m.view = FleetViewProfiles
			return m, nil
		case "esc":
			if m.dirtyCount > 0 {
				m.confirmDiscard = true
				return m, nil
			}
		}
		return m, nil
	}

	// Profiles view keys
	if m.view == FleetViewProfiles {
		filtered := m.FilteredInstances()
		switch key {
		case "/":
			m.isFiltering = true
			m.disableWarning = ""
			return m, nil

		case "l", "L":
			if m.isWide(m.width) {
				m.disableWarning = ""
				m.focusDetail = !m.focusDetail
				return m, nil
			}

		case "m", "M":
			if len(filtered) > 0 {
				m.disableWarning = ""
				m.openModelPicker()
			}
			return m, nil

		case "t", "T":
			selected, ok := m.SelectedInstance()
			if !ok {
				return m, nil
			}
			m.disableWarning = ""
			currentEnabled := selected.Enabled
			if edit, has := m.draft[selected.InstanceID]; has {
				currentEnabled = edit.Enabled
			}
			if currentEnabled {
				if strings.Contains(strings.ToLower(selected.InstalledState), "unmanaged") {
					m.disableWarning = "Cannot disable: unmanaged file cannot be removed by fleet toggle (an explicit CLI decision is required)."
					return m, nil
				}
				m.confirmDisable = true
				m.disableTargetID = selected.InstanceID
				return m, nil
			}
			m.stageToggle(selected, true)
			return m.requestSourcePreview()

		case "enter":
			if m.isWide(m.width) {
				if len(filtered) > 0 {
					m.disableWarning = ""
					m.openModelPicker()
				}
				return m, nil
			}
			if len(filtered) > 0 {
				m.detailOpen = !m.detailOpen
				m.detailScroll = 0
			}
			return m, nil

		case "esc":
			if m.focusDetail {
				m.focusDetail = false
				return m, nil
			}
			if m.detailOpen {
				m.detailOpen = false
				return m, nil
			}
			if m.filter != "" {
				m.filter = ""
				m.clampSelection()
				return m, nil
			}
			if m.dirtyCount > 0 {
				m.confirmDiscard = true
				return m, nil
			}
			m.view = FleetViewOverview
			return m, nil

		case "j", "down":
			m.disableWarning = ""
			if m.focusDetail && m.isWide(m.width) {
				m.detailScroll++
				return m, nil
			}
			if len(filtered) > 0 && m.selectedIndex < len(filtered)-1 {
				m.selectedIndex++
				m.detailScroll = 0
			}
			return m, nil

		case "k", "up":
			m.disableWarning = ""
			if m.focusDetail && m.isWide(m.width) {
				if m.detailScroll > 0 {
					m.detailScroll--
				}
				return m, nil
			}
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.detailScroll = 0
			}
			return m, nil

		case "home", "g":
			m.disableWarning = ""
			if !m.focusDetail {
				m.selectedIndex = 0
				m.detailScroll = 0
			} else {
				m.detailScroll = 0
			}
			return m, nil

		case "end", "G":
			m.disableWarning = ""
			if !m.focusDetail && len(filtered) > 0 {
				m.selectedIndex = len(filtered) - 1
				m.detailScroll = 0
			}
			return m, nil

		case "pgdown", "ctrl+d":
			m.disableWarning = ""
			if m.focusDetail {
				m.detailScroll += 4
			} else if len(filtered) > 0 {
				m.selectedIndex = min(len(filtered)-1, m.selectedIndex+5)
				m.detailScroll = 0
			}
			return m, nil

		case "pgup", "ctrl+u":
			m.disableWarning = ""
			if m.focusDetail {
				m.detailScroll = max(0, m.detailScroll-4)
			} else if len(filtered) > 0 {
				m.selectedIndex = max(0, m.selectedIndex-5)
				m.detailScroll = 0
			}
			return m, nil
		}
	}

	// Review Changes view keys
	if m.view == FleetViewReview {
		switch key {
		case "v", "V":
			return m.requestPlanPreview()
		case "a", "A":
			return m.requestApply()
		case "j", "down":
			m.reviewScroll++
			return m, nil
		case "k", "up":
			if m.reviewScroll > 0 {
				m.reviewScroll--
			}
			return m, nil
		case "pgdown", "ctrl+d":
			m.reviewScroll += 8
			return m, nil
		case "pgup", "ctrl+u":
			m.reviewScroll = max(0, m.reviewScroll-8)
			return m, nil
		case "home", "g":
			m.reviewScroll = 0
			return m, nil
		case "end", "G":
			total := len(m.sourcePreview.DiffLines)
			if m.dirtyCount == 0 {
				total = len(m.planPreview.DiffLines)
			}
			if total > 0 {
				m.reviewScroll = max(0, total-6)
			}
			return m, nil
		case "p", "P", "esc":
			m.view = FleetViewProfiles
			return m, nil
		case "o", "O":
			m.view = FleetViewOverview
			return m, nil
		}
		return m, nil
	}

	// Recovery view keys. Preview is read-only and always available; the
	// restore confirmation is gated by the reviewed live capability.
	if m.view == FleetViewRecovery {
		switch key {
		case "j", "down":
			if len(m.backups) > 0 && m.backupIndex < len(m.backups)-1 {
				m.backupIndex++
				m.restorePreview = FleetRestorePreview{}
				m.restoreErr = ""
				m.restoreError = nil
				m.recoveryScroll = 0
			}
			return m, nil
		case "k", "up":
			if m.backupIndex > 0 {
				m.backupIndex--
				m.restorePreview = FleetRestorePreview{}
				m.restoreErr = ""
				m.restoreError = nil
				m.recoveryScroll = 0
			}
			return m, nil
		case "home", "g":
			m.backupIndex = 0
			m.restorePreview = FleetRestorePreview{}
			m.restoreErr = ""
			m.restoreError = nil
			m.recoveryScroll = 0
			return m, nil
		case "end", "G":
			if len(m.backups) > 0 {
				m.backupIndex = len(m.backups) - 1
				m.restorePreview = FleetRestorePreview{}
				m.restoreErr = ""
				m.restoreError = nil
				m.recoveryScroll = 0
			}
			return m, nil
		case "enter", "v", "V":
			return m.requestRestorePreview()
		case "t", "T":
			return m.requestRestore()
		case "pgdown", "ctrl+d":
			m.recoveryScroll += 8
			return m, nil
		case "pgup", "ctrl+u":
			m.recoveryScroll = max(0, m.recoveryScroll-8)
			return m, nil
		case "<":
			m.recoveryScroll = max(0, m.recoveryScroll-8)
			return m, nil
		case ">":
			m.recoveryScroll += 8
			return m, nil
		case "d", "D":
			// Contextual local diagnostics, offline by default.
			return m.openDoctor(), nil
		case "p", "P", "esc":
			m.view = FleetViewProfiles
			return m, nil
		case "o", "O":
			m.view = FleetViewOverview
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m *FleetModel) clampSelection() {
	filtered := m.FilteredInstances()
	if len(filtered) == 0 {
		m.selectedIndex = 0
		return
	}
	if m.selectedIndex >= len(filtered) {
		m.selectedIndex = len(filtered) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}
