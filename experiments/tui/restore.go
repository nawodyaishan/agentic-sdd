package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// RestoreStage is the recovery screen's current step. The order matches plan
// §2's transaction state machine: opening a backup only reveals details, and
// a restore write can only happen after an independent default-No
// confirmation and a fresh revalidation.
type RestoreStage int

const (
	// RestoreStageList browses the discovered backups. It never writes.
	RestoreStageList RestoreStage = iota
	// RestoreStageDetails shows one backup's identity and recorded changes.
	// Opening a backup is not a restore.
	RestoreStageDetails
	// RestoreStagePreview shows the read-only restore plan and exact impact.
	RestoreStagePreview
	// RestoreStageConfirm asks for an independent default-No decision. Only
	// an explicit "y" accepts; Enter must not.
	RestoreStageConfirm
	// RestoreStageRevalidating re-runs the preview and compares its token
	// with the one consent was given for. Nothing is written in this stage.
	RestoreStageRevalidating
	// RestoreStageRunning is the single active restore write. It cannot be
	// abandoned from this screen.
	RestoreStageRunning
	// RestoreStageResult reports the structured outcome of the restore.
	RestoreStageResult
)

func (s RestoreStage) String() string {
	switch s {
	case RestoreStageList:
		return "backup list"
	case RestoreStageDetails:
		return "backup details"
	case RestoreStagePreview:
		return "restore preview"
	case RestoreStageConfirm:
		return "confirm restore"
	case RestoreStageRevalidating:
		return "revalidating preview"
	case RestoreStageRunning:
		return "restoring"
	case RestoreStageResult:
		return "restore result"
	default:
		return "unknown"
	}
}

// RestoreOps is the narrow, injectable operation seam for the recovery
// screen's read-only work. Update and View never call it directly; only the
// tea.Cmd values returned here do. Apply is deliberately absent: every write
// goes through the root operation host as a StartWriteMsg. A nil field is
// replaced by the default shared operation.
type RestoreOps struct {
	// List discovers backups under repo/home's backup root. A missing root is
	// an empty list, not an error.
	List func(repo, home string) ([]skillsync.Backup, error)
	// Preview runs the read-only restore plan and returns its consent token.
	Preview func(req appops.RestoreRequest) (appops.PreviewResult, error)
}

// DefaultRestoreOps wires the screen to the committed read-only shared
// operations. It discards the engine transcript because the CLI owns log
// formatting; the screen renders only structured results.
func DefaultRestoreOps() RestoreOps {
	return RestoreOps{
		List: skillsync.ListBackups,
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			return appops.PreviewRestore(req, io.Discard)
		},
	}
}

func (o RestoreOps) withDefaults() RestoreOps {
	defaults := DefaultRestoreOps()
	if o.List == nil {
		o.List = defaults.List
	}
	if o.Preview == nil {
		o.Preview = defaults.Preview
	}
	return o
}

// RestoreBackupsMsg carries one completed backup discovery. A stale message
// (an older generation or another request) must not change screen state.
type RestoreBackupsMsg struct {
	ID      appops.OperationID
	Backups []skillsync.Backup
	Err     error
}

// RestorePreviewMsg carries one completed read-only restore preview.
type RestorePreviewMsg struct {
	ID       appops.OperationID
	BackupID string
	Result   appops.PreviewResult
	Err      error
}

// RestoreRevalidatedMsg carries the preview a confirmation was revalidated
// against. It is not consent: the model compares its token with the one the
// user confirmed before emitting a StartWriteMsg.
type RestoreRevalidatedMsg struct {
	ID       appops.OperationID
	BackupID string
	Result   appops.PreviewResult
	Err      error
}

// RestoreCloseMsg asks the parent shell to leave the recovery screen. It is
// only emitted from the backup list; backing out of later stages stays inside
// the screen so a pending transaction is never silently abandoned.
type RestoreCloseMsg struct{}

// RestoreOutcomeKind is the honest classification of a finished restore. It
// is derived from structured engine evidence, never from display text.
type RestoreOutcomeKind string

const (
	// RestoreOutcomeApplied means destinations changed and a complete
	// pre-restore backup was recorded.
	RestoreOutcomeApplied RestoreOutcomeKind = "applied"
	// RestoreOutcomeNoChange means every destination already matched the
	// backup, so no write and no pre-restore backup happened.
	RestoreOutcomeNoChange RestoreOutcomeKind = "no-change"
	// RestoreOutcomeNeedsPreview means the operation refused to write because
	// its inputs or plan no longer matched the confirmed preview.
	RestoreOutcomeNeedsPreview RestoreOutcomeKind = "needs-preview"
	// RestoreOutcomeFailed means the operation failed without rollback
	// evidence.
	RestoreOutcomeFailed RestoreOutcomeKind = "failed"
	// RestoreOutcomeRolledBack means the operation failed and every attempted
	// undo succeeded.
	RestoreOutcomeRolledBack RestoreOutcomeKind = "rolled-back"
	// RestoreOutcomeIncomplete means rollback was attempted but not complete,
	// so destinations may be partially changed.
	RestoreOutcomeIncomplete RestoreOutcomeKind = "incomplete"
	// RestoreOutcomeCancelled means the operation reported cancellation. A
	// cancellation that arrived after work began cannot claim no writes.
	RestoreOutcomeCancelled RestoreOutcomeKind = "cancelled"
)

// RestoreOutcome is the screen's result data. BackupID is the backup being
// restored; SafetyBackupID and SafetyBackupPath identify the pre-restore
// backup the engine wrote (empty for a preview, no-change or failure before a
// backup). Applied and Planned stay empty unless the engine supplied them.
type RestoreOutcome struct {
	Kind             RestoreOutcomeKind
	BackupID         string
	SafetyBackupID   string
	SafetyBackupPath string
	Applied          []skillsync.ResultChange
	Planned          []skillsync.ResultChange
	Rollback         skillsync.RollbackInfo
	Err              string
	LogPath          string
}

// RestoreModel is the backups, restore preview, confirmation, execution and
// result screen model. It is self-contained: the parent shell routes Bubble
// Tea messages to Update and renders Lines or View. Filesystem work happens
// only inside the tea.Cmd values produced here.
type RestoreModel struct {
	cfg   TUIConfig
	theme Theme
	ops   RestoreOps

	stage RestoreStage
	opID  appops.OperationID

	width  int
	height int
	scroll int

	backups    []skillsync.Backup
	listIndex  int
	selectedID string
	loading    bool
	listErr    string

	preview        appops.PreviewResult
	previewID      string
	previewErr     string
	previewLoading bool
	driftNotice    string

	consentToken    string
	consentBackupID string

	// pendingBackupID is the backup the in-flight or finished write restores.
	// It is set when consent is revalidated and cleared when the transaction or
	// its result is done.
	pendingBackupID string
	writeStep       WriteStep
	writeDetail     string

	notice string
	result *RestoreOutcome
}

// NewRestoreModel builds the recovery screen for one configured home. It
// performs no filesystem access; Init starts an asynchronous read-only backup
// discovery. A missing --home is reported as a configuration error rather
// than guessed.
func NewRestoreModel(cfg TUIConfig, ops RestoreOps) RestoreModel {
	cfg = cfg.Normalize()
	m := RestoreModel{
		cfg:    cfg,
		theme:  NewTheme(cfg.Color),
		ops:    ops.withDefaults(),
		stage:  RestoreStageList,
		opID:   appops.OperationID{Generation: 1},
		width:  cfg.Width,
		height: cfg.Height,
	}
	if cfg.Home == "" {
		m.listErr = "Restore home is required. Recovery: pass --home to the TUI and reopen this screen. Nothing was read or changed."
	}
	return m
}

// Init starts the first read-only backup discovery. It returns nil when the
// configuration already failed validation.
func (m RestoreModel) Init() tea.Cmd {
	if m.cfg.Home == "" {
		return nil
	}
	return m.loadBackupsCmd(m.opID)
}

// Reload repeats discovery and preserves the selected backup by ID when it
// is still present. It never writes. The returned model owns the new
// operation generation, so callers must keep it.
func (m RestoreModel) Reload() (RestoreModel, tea.Cmd) {
	if m.cfg.Home == "" || m.Busy() {
		return m, nil
	}
	m.opID.Generation++
	m.loading = true
	m.listErr = ""
	m.notice = ""
	return m, m.loadBackupsCmd(m.opID)
}

// SetSize updates the available cells and clamps the scroll offset safely.
func (m *RestoreModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.scroll = m.clampScroll(m.scroll)
}

// Stage reports the current step for the parent shell's footer and tests.
func (m RestoreModel) Stage() RestoreStage { return m.stage }

// Busy reports whether a restore transaction has begun. The parent shell must
// defer quit while this is true: q/Esc/Ctrl+C cannot abandon these stages.
func (m RestoreModel) Busy() bool {
	return m.stage == RestoreStageRevalidating || m.stage == RestoreStageRunning
}

// Backups returns a copy of the discovered backup list.
func (m RestoreModel) Backups() []skillsync.Backup {
	return append([]skillsync.Backup(nil), m.backups...)
}

// SelectedBackup returns the currently highlighted backup, if any.
func (m RestoreModel) SelectedBackup() (skillsync.Backup, bool) {
	if m.listIndex < 0 || m.listIndex >= len(m.backups) {
		return skillsync.Backup{}, false
	}
	return m.backups[m.listIndex], true
}

// PreviewResult returns the last successful read-only preview.
func (m RestoreModel) PreviewResult() appops.PreviewResult { return m.preview }

// Outcome returns the result after the restore finished, if it has.
func (m RestoreModel) Outcome() (RestoreOutcome, bool) {
	if m.result == nil {
		return RestoreOutcome{}, false
	}
	return *m.result, true
}

// Update routes terminal messages. It never performs filesystem work: every
// operation is dispatched as a tea.Cmd and every result is matched against
// the current operation identity so a stale or duplicate message cannot
// overwrite newer state or authorize a second write.
func (m RestoreModel) Update(msg tea.Msg) (RestoreModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case RestoreBackupsMsg:
		return m.handleBackups(msg)
	case RestorePreviewMsg:
		return m.handlePreview(msg)
	case RestoreRevalidatedMsg:
		return m.handleRevalidated(msg)
	case WriteStartedMsg:
		if m.stage != RestoreStageRunning || msg.Kind != WriteRestore {
			return m, nil
		}
		m.writeStep = WriteStepStarting
		return m, nil
	case WriteProgressMsg:
		if m.stage != RestoreStageRunning || msg.Kind != WriteRestore {
			return m, nil
		}
		m.writeStep = msg.Step
		m.writeDetail = msg.Detail
		return m, nil
	case WriteRefusedMsg:
		// A refusal never started a write. Return to the independent
		// confirmation with its consent intact so the same plan can be retried.
		if m.stage != RestoreStageRunning || msg.Kind != WriteRestore {
			return m, nil
		}
		m.stage = RestoreStageConfirm
		m.writeStep = ""
		m.writeDetail = ""
		m.notice = msg.Reason
		return m, nil
	case WriteResultMsg:
		if m.stage != RestoreStageRunning || msg.Kind != WriteRestore {
			return m, nil
		}
		return m.handleWriteResult(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m RestoreModel) handleBackups(msg RestoreBackupsMsg) (RestoreModel, tea.Cmd) {
	if msg.ID.StaleFor(m.opID) {
		return m, nil
	}
	m.loading = false
	m.listErr = ""
	if msg.Err != nil {
		m.backups = nil
		m.listErr = "Could not read backups: " + m.data(msg.Err.Error()) +
			"\nRecovery: verify --repo and --home, then press r to retry. Nothing was changed."
		m.clampListSelection()
		return m, nil
	}
	// ListBackups already orders newest first; sorting again keeps an injected
	// source deterministic instead of trusting its iteration order.
	list := append([]skillsync.Backup(nil), msg.Backups...)
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID > list[j].ID })
	previous := m.selectedID
	m.backups = list
	m.listIndex = 0
	for i, b := range list {
		if b.ID == previous {
			m.listIndex = i
			break
		}
	}
	m.clampListSelection()
	return m, nil
}

func (m RestoreModel) handlePreview(msg RestorePreviewMsg) (RestoreModel, tea.Cmd) {
	if msg.ID.StaleFor(m.opID) || m.stage != RestoreStagePreview {
		return m, nil
	}
	m.previewLoading = false
	if msg.Err != nil {
		m.preview = appops.PreviewResult{}
		m.previewID = ""
		m.previewErr = m.describeProblem(msg.Err)
		m.clearConsent()
		return m, nil
	}
	m.preview = msg.Result
	m.previewID = msg.BackupID
	m.previewErr = ""
	m.driftNotice = ""
	m.clearConsent()
	return m, nil
}

func (m RestoreModel) handleRevalidated(msg RestoreRevalidatedMsg) (RestoreModel, tea.Cmd) {
	if msg.ID.StaleFor(m.opID) || m.stage != RestoreStageRevalidating {
		return m, nil
	}
	if msg.Err != nil {
		m.stage = RestoreStagePreview
		m.previewLoading = false
		m.previewErr = m.describeProblem(msg.Err)
		m.clearConsent()
		return m, nil
	}
	// Consent covers one exact backup and one exact proposed change set. A
	// changed backup, destination, source or plan invalidates it; the new
	// plan is shown read-only and a fresh confirmation is required.
	if m.consentToken == "" || m.consentBackupID == "" ||
		msg.BackupID != m.consentBackupID || msg.Result.PreviewToken != m.consentToken {
		m.stage = RestoreStagePreview
		m.preview = msg.Result
		m.previewID = msg.BackupID
		m.previewErr = ""
		m.driftNotice = "The backup, a destination or the source changed after the preview. Nothing was written. Review the refreshed plan and confirm again."
		m.clearConsent()
		return m, nil
	}

	token := m.consentToken
	backupID := m.consentBackupID
	req, err := m.restoreWriteRequest(backupID, token)
	if err != nil {
		m.stage = RestoreStagePreview
		m.previewErr = m.describeProblem(err)
		m.clearConsent()
		return m, nil
	}
	// Consent stays pending until the root host accepts the write: a host
	// refusal returns to this confirmation with the same plan still authorized.
	m.pendingBackupID = backupID
	m.writeStep = WriteStepStarting
	m.writeDetail = ""
	m.stage = RestoreStageRunning
	m.notice = ""
	m.beginOp()
	return m, startWriteCmd(req)
}

func (m RestoreModel) handleWriteResult(msg WriteResultMsg) (RestoreModel, tea.Cmd) {
	kind := classifyRestoreOutcome(msg.Result, msg.Err)
	backupID := m.pendingBackupID
	if backupID == "" {
		backupID = m.previewID
	}
	m.writeStep = ""
	m.writeDetail = ""
	m.pendingBackupID = ""
	m.clearConsent()

	if kind == RestoreOutcomeNeedsPreview {
		// The engine revalidated the token and refused to write. Nothing was
		// changed; a fresh read-only preview and a new default-No confirmation
		// are required.
		m.result = nil
		m.preview = appops.PreviewResult{}
		m.previewID = backupID
		m.previewErr = ""
		m.selectedID = backupID
		m.stage = RestoreStageDetails
		updated, cmd := m.beginPreview()
		if updated.stage == RestoreStagePreview {
			updated.driftNotice = "The restore was refused because the backup, a destination or the source changed after confirmation. Nothing was written; review the refreshed plan and confirm again."
		}
		return updated, cmd
	}

	m.result = &RestoreOutcome{
		Kind:             kind,
		BackupID:         backupID,
		SafetyBackupID:   msg.Result.BackupID,
		SafetyBackupPath: msg.Result.BackupPath,
		Applied:          append([]skillsync.ResultChange(nil), msg.Result.Applied...),
		Planned:          append([]skillsync.ResultChange(nil), msg.Result.Planned...),
		Rollback:         msg.Result.Rollback,
		LogPath:          msg.LogPath,
	}
	if msg.Err != nil {
		m.result.Err = msg.Err.Error()
	}
	if msg.LogErr != nil {
		m.result.Err = strings.TrimSpace(m.result.Err + " operation log: " + msg.LogErr.Error())
	}
	m.stage = RestoreStageResult
	m.scroll = 0
	return m, nil
}

func classifyRestoreOutcome(result appops.ApplyResult, err error) RestoreOutcomeKind {
	if err != nil {
		switch {
		case errors.Is(err, appops.ErrNeedsPreview):
			return RestoreOutcomeNeedsPreview
		case errors.Is(err, context.Canceled):
			return RestoreOutcomeCancelled
		case result.Rollback.Attempted && result.Rollback.Complete:
			return RestoreOutcomeRolledBack
		case result.Rollback.Attempted:
			return RestoreOutcomeIncomplete
		default:
			return RestoreOutcomeFailed
		}
	}
	switch result.Outcome {
	case skillsync.OutcomeApplied:
		return RestoreOutcomeApplied
	case skillsync.OutcomeNoChange:
		return RestoreOutcomeNoChange
	default:
		// No error and no definite success outcome: never claim success.
		return RestoreOutcomeFailed
	}
}

func (m RestoreModel) handleKey(msg tea.KeyPressMsg) (RestoreModel, tea.Cmd) {
	key := msg.String()
	// A started transaction is not cancellable from the screen. The parent
	// shell's shutdown policy decides exit-after-completion; q/Esc cannot drop
	// or repeat the write here.
	if m.Busy() {
		return m, nil
	}

	switch m.stage {
	case RestoreStageList:
		return m.handleListKey(key)
	case RestoreStageDetails:
		return m.handleDetailsKey(key)
	case RestoreStagePreview:
		return m.handlePreviewKey(key)
	case RestoreStageConfirm:
		return m.handleConfirmKey(key)
	case RestoreStageResult:
		return m.handleResultKey(key)
	}
	return m, nil
}

func (m RestoreModel) handleListKey(key string) (RestoreModel, tea.Cmd) {
	switch key {
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "home":
		m.setSelection(0)
	case "end":
		m.setSelection(len(m.backups) - 1)
	case "r":
		if !m.loading {
			return m.Reload()
		}
	case "enter":
		return m.openSelected()
	case "esc":
		return m, func() tea.Msg { return RestoreCloseMsg{} }
	}
	return m, nil
}

func (m RestoreModel) handleDetailsKey(key string) (RestoreModel, tea.Cmd) {
	switch key {
	case "esc", "b":
		m.stage = RestoreStageList
		m.clearTransaction()
	case "enter", "p":
		return m.beginPreview()
	case "up", "k":
		m.scrollBy(-1)
	case "down", "j":
		m.scrollBy(1)
	case "pgup", "pageup", "ctrl+u":
		m.scrollBy(-4)
	case "pgdown", "pagedown", "ctrl+d":
		m.scrollBy(4)
	case "home":
		m.scroll = 0
	}
	return m, nil
}

func (m RestoreModel) handlePreviewKey(key string) (RestoreModel, tea.Cmd) {
	switch key {
	case "esc", "b":
		// Leave the read-only preview; no consent exists, so nothing can be
		// written by a late result after this invalidation.
		m.stage = RestoreStageDetails
		m.invalidatePending()
		m.clearTransaction()
	case "p":
		if !m.previewLoading {
			return m.beginPreview()
		}
	case "enter":
		if m.previewLoading {
			return m, nil
		}
		if m.previewErr != "" || m.preview.PreviewToken == "" {
			m.notice = "No confirmation is available: the preview did not produce a valid plan. Press Esc to go back. Nothing was changed."
			return m, nil
		}
		m.stage = RestoreStageConfirm
		m.notice = ""
		m.scroll = 0
	case "up", "k":
		m.scrollBy(-1)
	case "down", "j":
		m.scrollBy(1)
	case "pgup", "pageup", "ctrl+u":
		m.scrollBy(-4)
	case "pgdown", "pagedown", "ctrl+d":
		m.scrollBy(4)
	case "home":
		m.scroll = 0
	}
	return m, nil
}

func (m RestoreModel) handleConfirmKey(key string) (RestoreModel, tea.Cmd) {
	switch key {
	case "y":
		return m.confirmRestore()
	case "n", "esc", "b":
		// Cancelling is always safe: no write has been dispatched.
		m.stage = RestoreStagePreview
		m.clearConsent()
		m.notice = "Restore cancelled. Nothing was written."
	case "up", "k":
		m.scrollBy(-1)
	case "down", "j":
		m.scrollBy(1)
	case "pgup", "pageup", "ctrl+u":
		m.scrollBy(-4)
	case "pgdown", "pagedown", "ctrl+d":
		m.scrollBy(4)
	case "home":
		m.scroll = 0
	case "enter":
		// Enter is deliberately inert. Default No means only an explicit y
		// may authorize a restore.
		m.notice = "Enter does not confirm. Press y to restore, or n/Esc to cancel. Nothing was written."
	}
	return m, nil
}

func (m RestoreModel) handleResultKey(key string) (RestoreModel, tea.Cmd) {
	switch key {
	case "esc", "enter":
		m.stage = RestoreStageList
		m.clearTransaction()
		// A finished restore can add a pre-restore backup; refresh the list
		// from disk rather than showing a stale snapshot.
		return m.Reload()
	case "p":
		// A fresh preview is read-only and requires a new default-No
		// confirmation. It is the recovery action for needs-preview.
		m.stage = RestoreStageDetails
		m.clearTransaction()
		return m.beginPreview()
	case "up", "k":
		m.scrollBy(-1)
	case "down", "j":
		m.scrollBy(1)
	case "pgup", "pageup", "ctrl+u":
		m.scrollBy(-4)
	case "pgdown", "pagedown", "ctrl+d":
		m.scrollBy(4)
	case "home":
		m.scroll = 0
	}
	return m, nil
}

func (m RestoreModel) openSelected() (RestoreModel, tea.Cmd) {
	if m.loading {
		m.notice = "Backups are still loading. Nothing was changed."
		return m, nil
	}
	if len(m.backups) == 0 {
		m.notice = "No backup is selected. Nothing was changed."
		return m, nil
	}
	m.clampListSelection()
	backup, ok := m.SelectedBackup()
	if !ok {
		m.notice = "The selected backup is no longer available. Press r to refresh the list. Nothing was changed."
		return m, nil
	}
	if err := skillsync.ValidateBackupID(backup.ID); err != nil {
		m.selectedID = ""
		m.stage = RestoreStageDetails
		m.previewErr = "Invalid backup selection: " + m.data(err.Error()) +
			"\nRecovery: return to the list and choose a backup whose ID matches YYYYMMDDTHHMMSS.nnnnnnnnnZ. Nothing was changed."
		m.scroll = 0
		return m, nil
	}
	m.selectedID = backup.ID
	m.stage = RestoreStageDetails
	m.clearTransaction()
	m.scroll = 0
	return m, nil
}

func (m RestoreModel) beginPreview() (RestoreModel, tea.Cmd) {
	backup, ok := m.backupByID(m.selectedID)
	if !ok {
		m.stage = RestoreStageList
		m.notice = "That backup is no longer in the list. Press r to refresh and choose another. Nothing was changed."
		return m, nil
	}
	if backup.Unusable != "" {
		m.stage = RestoreStageDetails
		m.previewErr = m.describeProblem(errors.New(backup.Unusable))
		m.notice = "This backup cannot be previewed or restored. Nothing was changed."
		return m, nil
	}
	req, err := m.request(backup.ID)
	if err != nil {
		m.stage = RestoreStagePreview
		m.previewErr = m.describeProblem(err)
		m.clearConsent()
		return m, nil
	}
	m.stage = RestoreStagePreview
	m.previewLoading = true
	m.preview = appops.PreviewResult{}
	m.previewID = backup.ID
	m.previewErr = ""
	m.driftNotice = ""
	m.notice = ""
	m.result = nil
	m.pendingBackupID = ""
	m.writeStep = ""
	m.writeDetail = ""
	m.clearConsent()
	m.scroll = 0
	opID := m.beginOp()
	return m, m.previewCmd(opID, backup.ID, req)
}

func (m RestoreModel) confirmRestore() (RestoreModel, tea.Cmd) {
	if m.previewErr != "" || m.preview.PreviewToken == "" || m.previewID == "" {
		m.notice = "The preview is not confirmable. Press Esc and run a fresh preview. Nothing was written."
		return m, nil
	}
	req, err := m.request(m.previewID)
	if err != nil {
		m.stage = RestoreStagePreview
		m.previewErr = m.describeProblem(err)
		m.clearConsent()
		return m, nil
	}
	m.consentToken = m.preview.PreviewToken
	m.consentBackupID = m.previewID
	m.stage = RestoreStageRevalidating
	m.notice = ""
	m.scroll = 0
	opID := m.beginOp()
	return m, m.revalidateCmd(opID, m.consentBackupID, req)
}

func (m *RestoreModel) beginOp() appops.OperationID {
	m.opID.Generation++
	return m.opID
}

func (m *RestoreModel) invalidatePending() {
	m.opID.Generation++
	m.previewLoading = false
}

func (m *RestoreModel) clearConsent() {
	m.consentToken = ""
	m.consentBackupID = ""
}

func (m *RestoreModel) clearTransaction() {
	m.clearConsent()
	m.pendingBackupID = ""
	m.writeStep = ""
	m.writeDetail = ""
	m.preview = appops.PreviewResult{}
	m.previewID = ""
	m.previewErr = ""
	m.previewLoading = false
	m.driftNotice = ""
	m.notice = ""
	m.scroll = 0
}

func (m RestoreModel) request(backupID string) (appops.RestoreRequest, error) {
	return appops.NewRestoreRequest(m.cfg.Repo, m.cfg.Home, backupID, false)
}

func (m RestoreModel) loadBackupsCmd(id appops.OperationID) tea.Cmd {
	repo, home := m.cfg.Repo, m.cfg.Home
	list := m.ops.List
	return func() tea.Msg {
		backups, err := list(repo, home)
		return RestoreBackupsMsg{ID: id, Backups: backups, Err: err}
	}
}

func (m RestoreModel) previewCmd(id appops.OperationID, backupID string, req appops.RestoreRequest) tea.Cmd {
	preview := m.ops.Preview
	return func() tea.Msg {
		result, err := preview(req)
		return RestorePreviewMsg{ID: id, BackupID: backupID, Result: result, Err: err}
	}
}

func (m RestoreModel) revalidateCmd(id appops.OperationID, backupID string, req appops.RestoreRequest) tea.Cmd {
	preview := m.ops.Preview
	return func() tea.Msg {
		result, err := preview(req)
		return RestoreRevalidatedMsg{ID: id, BackupID: backupID, Result: result, Err: err}
	}
}

func (m RestoreModel) backupByID(id string) (skillsync.Backup, bool) {
	for _, b := range m.backups {
		if b.ID == id {
			return b, true
		}
	}
	return skillsync.Backup{}, false
}

func (m *RestoreModel) clampListSelection() {
	if len(m.backups) == 0 {
		m.listIndex = 0
		m.selectedID = ""
		return
	}
	if m.listIndex < 0 {
		m.listIndex = 0
	}
	if m.listIndex >= len(m.backups) {
		m.listIndex = len(m.backups) - 1
	}
	m.selectedID = m.backups[m.listIndex].ID
}

func (m *RestoreModel) setSelection(index int) {
	if len(m.backups) == 0 {
		m.listIndex = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.backups) {
		index = len(m.backups) - 1
	}
	m.listIndex = index
	m.selectedID = m.backups[index].ID
	m.scroll = m.listSelectionStart(m.width)
}

func (m *RestoreModel) moveSelection(delta int) {
	m.setSelection(m.listIndex + delta)
}

func (m *RestoreModel) scrollBy(delta int) {
	m.scroll += delta
	m.scroll = m.clampScroll(m.scroll)
}

func (m RestoreModel) clampScroll(offset int) int {
	maxOffset := len(m.Lines(m.width)) - m.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

// listHeaderLines is the fixed header before the backup rows. It is shared
// with listSelectionStart so a moved selection is brought into view without
// guessing wrapped row counts.
func (m RestoreModel) listHeaderLines(width int) []string {
	lines := []string{m.theme.Title.Render("Backups & recovery")}
	lines = append(lines, Paragraph("Read-only backup browser. Opening a backup only shows details; it never restores anything.", width, 0)...)
	if m.listErr != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Bad.Render("Backup discovery unavailable"), width, 0)...)
		for _, line := range Paragraph(m.data(m.listErr), width, 2) {
			lines = append(lines, line)
		}
	} else if m.loading {
		lines = append(lines, "")
		lines = append(lines, Paragraph("Loading backups...", width, 0)...)
	} else if len(m.backups) == 0 {
		lines = append(lines, "")
		lines = append(lines, m.theme.OK.Render("No backups found."))
		for _, line := range Paragraph(m.data("No backup directory content was found at "+restoreBackupRoot(m.cfg.Repo, m.cfg.Home)+"."), width, 2) {
			lines = append(lines, line)
		}
		for _, line := range Paragraph("Recovery: backups are created by an install, update or restore that changes something. Run an install/update through this TUI or the explicit CLI, then press r. An empty or missing backup root is a valid empty state; nothing is broken and nothing was changed.", width, 2) {
			lines = append(lines, line)
		}
	}
	return lines
}

func (m RestoreModel) listEntryGroups(width int) [][]string {
	groups := make([][]string, 0, len(m.backups))
	for i, b := range m.backups {
		marker := " "
		if i == m.listIndex {
			marker = ">"
		}
		head := fmt.Sprintf("%s %s  %s", marker, m.data(b.ID), m.data(restoreIdentity(b)))
		if i == m.listIndex {
			head = m.theme.Accent.Render(head)
		}
		group := Paragraph(head, width, 0)
		if b.Unusable != "" {
			group = append(group, Paragraph(m.data("unusable: "+b.Unusable), width, 4)...)
			group = append(group, Paragraph("Recovery: this entry cannot be selected for restore; inspect or remove it with the explicit CLI. Nothing was changed.", width, 4)...)
		} else {
			group = append(group, Paragraph(m.data("source: "+restoreSource(b)), width, 4)...)
			if b.Summary != "" {
				group = append(group, Paragraph(m.data(b.Summary), width, 4)...)
			}
		}
		group = append(group, "")
		groups = append(groups, group)
	}
	return groups
}

func (m RestoreModel) listSelectionStart(width int) int {
	header := m.listHeaderLines(width)
	groups := m.listEntryGroups(width)
	start := len(header)
	if len(header) > 0 && len(groups) > 0 {
		start++
	}
	for i := 0; i < m.listIndex && i < len(groups); i++ {
		start += len(groups[i])
	}
	return start
}

func (m RestoreModel) detailsLines(width int) []string {
	backup, ok := m.backupByID(m.selectedID)
	if !ok {
		return []string{
			m.theme.Bad.Render("Backup unavailable"),
			"The selected backup is no longer in the list. Press Esc and refresh; nothing was changed.",
		}
	}
	body := []string{
		"Backup ID: " + m.data(backup.ID),
		"Created: " + m.data(restoreWhen(backup)),
		"Operation: " + m.data(restoreOperation(backup)),
		"Source: " + m.data(restoreSource(backup)),
		"Format: " + m.data(restoreFormat(backup)),
	}
	if backup.Summary != "" {
		body = append(body, "Summary: "+m.data(backup.Summary))
	}
	if backup.Legacy {
		body = append(body, "Legacy backup: only the saved skills are recorded; skills installed fresh by that apply are left as is.")
	}
	lines := Section(m.theme.Title.Render("Backup details"), body, width)
	if backup.Unusable != "" {
		lines = append(lines, "")
		lines = append(lines, Section(m.theme.Bad.Render("Unusable backup"), []string{
			m.data(backup.Unusable),
			"Recovery: this backup cannot be previewed or restored. Return to the list and choose another backup, or inspect and remove this entry with the explicit CLI. No confirmation is offered and nothing was changed.",
		}, width)...)
	} else if m.previewErr != "" {
		lines = append(lines, "")
		lines = append(lines, Section(m.theme.Bad.Render("Restore unavailable"), []string{m.data(m.previewErr)}, width)...)
	} else {
		lines = append(lines, "")
		lines = append(lines, Section("Next step", []string{
			"Press Enter or p to run a read-only restore preview. Preview writes nothing and does not restore the backup.",
		}, width)...)
	}
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}
	lines = append(lines, "")
	lines = append(lines, Paragraph(m.theme.Hint.Render("Esc back to backups | Enter/p preview | q quit from the shell"), width, 0)...)
	return lines
}

func (m RestoreModel) previewLines(width int) []string {
	lines := []string{m.theme.Title.Render("Restore preview")}
	lines = append(lines, Paragraph("Read-only plan. Nothing has been written and nothing will be restored until you explicitly confirm.", width, 0)...)
	lines = append(lines, "")
	lines = append(lines, Section("Backup", []string{
		"Backup ID: " + m.data(m.previewID),
		"Planned changes: " + fmt.Sprintf("%d", restoreChangeCount(m.preview.Planned)),
	}, width)...)

	if m.previewLoading {
		lines = append(lines, "")
		lines = append(lines, Paragraph("Reading the backup and comparing destinations ...", width, 0)...)
	}
	if m.previewErr != "" {
		lines = append(lines, "")
		lines = append(lines, Section(m.theme.Bad.Render("Restore unavailable"), []string{m.data(m.previewErr)}, width)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph("No confirmation is available. Nothing was changed. Press Esc to go back or p to retry the preview.", width, 0)...)
	}
	if m.driftNotice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.driftNotice)), width, 0)...)
	}
	if m.previewErr == "" && !m.previewLoading {
		lines = append(lines, "")
		lines = append(lines, m.previewImpactLines(m.preview, width, m.theme.Section.Render)...)
		lines = append(lines, "")
		if restoreChangeCount(m.preview.Planned) == 0 {
			lines = append(lines, Paragraph("Restoring this backup would change nothing; no pre-restore backup would be written.", width, 0)...)
		} else {
			lines = append(lines, Paragraph("Before any destination changes, the current content of every overwritten or removed path is saved to a new pre-restore backup.", width, 0)...)
		}
	}
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}
	lines = append(lines, "")
	hint := "Esc details | p refresh preview | enter confirmation"
	if m.previewErr != "" || m.previewLoading {
		hint = "Esc details | p retry preview"
	}
	lines = append(lines, Paragraph(m.theme.Hint.Render(hint), width, 0)...)
	return lines
}

func (m RestoreModel) confirmLines(width int) []string {
	lines := []string{m.theme.Title.Render("Confirm restore")}
	lines = append(lines, Paragraph(m.theme.Warn.Render("Default: No. Only pressing y confirms. Enter does not confirm."), width, 0)...)
	lines = append(lines, "")
	lines = append(lines, Paragraph(m.data(fmt.Sprintf("Restore backup %s and change the destinations listed below? Their current content is saved to a new pre-restore backup first.", m.previewID)), width, 0)...)
	lines = append(lines, "")
	lines = append(lines, m.previewImpactLines(m.preview, width, m.theme.Section.Render)...)
	lines = append(lines, "")
	lines = append(lines, Paragraph("No changes have been written yet. y restore | n/Esc cancel. Enter does not confirm.", width, 0)...)
	if m.notice != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.theme.Warn.Render(m.data(m.notice)), width, 0)...)
	}
	return lines
}

func (m RestoreModel) revalidatingLines(width int) []string {
	lines := []string{m.theme.Title.Render("Revalidating restore")}
	lines = append(lines, Paragraph("Checking that the backup, destinations and source still match the preview you confirmed. Nothing has been written yet.", width, 0)...)
	lines = append(lines, "")
	lines = append(lines, Paragraph("If anything changed, this screen shows a fresh read-only preview and asks for confirmation again. If nothing changed, the restore begins.", width, 0)...)
	return lines
}

func (m RestoreModel) runningLines(width int) []string {
	lines := []string{m.theme.Title.Render("Restoring backup")}
	lines = append(lines, Paragraph("The restore is running. This write cannot be abandoned safely.", width, 0)...)
	lines = append(lines, "")
	step := m.writeStep
	if step == "" {
		step = WriteStepStarting
	}
	lines = append(lines, Paragraph("Phase: "+string(step), width, 0)...)
	if m.writeDetail != "" {
		lines = append(lines, Paragraph(m.data("Detail: "+m.writeDetail), width, 2)...)
	}
	lines = append(lines, "")
	lines = append(lines, Paragraph("q, Esc and navigation are ignored until the engine finishes or rolls back and its result is shown. Ctrl+C requests exit after this safe completion rather than interrupting the write.", width, 0)...)
	return lines
}

func (m RestoreModel) resultLines(width int) []string {
	if m.result == nil {
		return []string{m.theme.Title.Render("Restore result"), "No result is available."}
	}
	out := *m.result
	lines := []string{m.theme.Title.Render("Restore result")}
	switch out.Kind {
	case RestoreOutcomeApplied:
		lines = append(lines, Paragraph(m.theme.OK.Render("Restore applied."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.data(fmt.Sprintf("Backup %s was restored.", out.BackupID)), width, 0)...)
		if out.SafetyBackupID != "" {
			lines = append(lines, Paragraph(m.data(fmt.Sprintf("Pre-restore backup: %s", out.SafetyBackupID)), width, 2)...)
			if out.SafetyBackupPath != "" {
				lines = append(lines, Paragraph(m.data("Path: "+out.SafetyBackupPath), width, 2)...)
			}
		} else {
			lines = append(lines, Paragraph("The engine reported success without a pre-restore backup reference; inspect the destinations before relying on rollback.", width, 2)...)
		}
		if len(out.Applied) > 0 {
			lines = append(lines, "")
			lines = append(lines, m.changesLines("Applied changes", out.Applied, width)...)
		}
	case RestoreOutcomeNoChange:
		lines = append(lines, Paragraph(m.theme.OK.Render("No changes needed."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.data(fmt.Sprintf("Every destination already matched backup %s. No destination was written and no pre-restore backup was created.", out.BackupID)), width, 0)...)
	case RestoreOutcomeNeedsPreview:
		lines = append(lines, Paragraph(m.theme.Warn.Render("Restore refused: a fresh preview is required."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph("Nothing was written. The backup, a destination or the source changed after the preview, so the confirmed plan is no longer valid.", width, 0)...)
		lines = append(lines, Paragraph("A fresh read-only preview was started; review it and confirm again, or Esc to return.", width, 0)...)
	case RestoreOutcomeRolledBack:
		lines = append(lines, Paragraph(m.theme.Bad.Render("Restore failed and was rolled back."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph("Every attempted undo succeeded; the destinations were returned to their previous state.", width, 0)...)
		lines = append(lines, m.safetyBackupLines(out, width)...)
	case RestoreOutcomeIncomplete:
		lines = append(lines, Paragraph(m.theme.Bad.Render("Restore failed and rollback was incomplete."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph("Some destinations may be partially changed. Inspect them before retrying; this screen does not claim a clean state.", width, 0)...)
		lines = append(lines, m.safetyBackupLines(out, width)...)
		if out.Rollback.Error != "" {
			lines = append(lines, Paragraph(m.data("Rollback detail: "+out.Rollback.Error), width, 2)...)
		}
	case RestoreOutcomeCancelled:
		lines = append(lines, Paragraph(m.theme.Warn.Render("Restore cancelled."), width, 0)...)
		lines = append(lines, "")
		lines = append(lines, Paragraph("The operation reported cancellation. A cancellation after work began cannot guarantee that no destination changed; inspect the references below before retrying.", width, 0)...)
		lines = append(lines, m.safetyBackupLines(out, width)...)
	default:
		lines = append(lines, Paragraph(m.theme.Bad.Render("Restore failed."), width, 0)...)
		lines = append(lines, "")
		if out.SafetyBackupID == "" {
			lines = append(lines, Paragraph("The engine reported no pre-restore backup, so it failed before saving current content. Inspect the destinations if you are unsure.", width, 0)...)
		} else {
			lines = append(lines, Paragraph("A pre-restore backup was written before the failure. Inspect it before retrying.", width, 0)...)
			lines = append(lines, m.safetyBackupLines(out, width)...)
		}
	}
	if out.Err != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.data("Error: "+out.Err), width, 2)...)
	}
	if out.Kind != RestoreOutcomeApplied && len(out.Applied) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.changesLines("Applied changes", out.Applied, width)...)
	}
	if out.LogPath != "" {
		lines = append(lines, "")
		lines = append(lines, Paragraph(m.data("Operation log: "+out.LogPath), width, 0)...)
	}
	lines = append(lines, "")
	hint := "Enter/Esc backup list"
	if out.Kind != RestoreOutcomeApplied && out.Kind != RestoreOutcomeNoChange {
		hint += " | p fresh preview"
	}
	lines = append(lines, Paragraph(m.theme.Hint.Render(hint), width, 0)...)
	return lines
}

func (m RestoreModel) safetyBackupLines(out RestoreOutcome, width int) []string {
	if out.SafetyBackupID == "" {
		return nil
	}
	lines := []string{""}
	lines = append(lines, Paragraph(m.data("Pre-restore backup: "+out.SafetyBackupID), width, 2)...)
	if out.SafetyBackupPath != "" {
		lines = append(lines, Paragraph(m.data("Path: "+out.SafetyBackupPath), width, 2)...)
	}
	return lines
}

// previewImpactLines groups the exact planned changes by action. Every path is
// shown through wrapping, never truncated: overwrite and removal impact must
// stay fully inspectable.
func (m RestoreModel) previewImpactLines(result appops.PreviewResult, width int, style func(...string) string) []string {
	var lines []string
	groups := []struct {
		action skillsync.ChangeAction
		title  string
	}{
		{skillsync.ChangeReplace, "Will overwrite existing content"},
		{skillsync.ChangeRestore, "Will restore a missing destination"},
		{skillsync.ChangeRemove, "Will remove content that did not exist before that backup"},
		{skillsync.ChangeUpToDate, "Already up to date (left untouched)"},
	}
	for _, group := range groups {
		var matches []skillsync.ResultChange
		for _, change := range result.Planned {
			if change.Action == group.action {
				matches = append(matches, change)
			}
		}
		if len(matches) == 0 {
			continue
		}
		body := make([]string, 0, len(matches))
		for _, change := range matches {
			body = append(body, m.data(change.Path))
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, Section(style(fmt.Sprintf("%s (%d)", group.title, len(matches))), body, width)...)
	}
	// Unknown actions stay visible instead of silently disappearing.
	var other []skillsync.ResultChange
	for _, change := range result.Planned {
		switch change.Action {
		case skillsync.ChangeReplace, skillsync.ChangeRestore, skillsync.ChangeRemove, skillsync.ChangeUpToDate:
		default:
			other = append(other, change)
		}
	}
	if len(other) > 0 {
		body := make([]string, 0, len(other))
		for _, change := range other {
			body = append(body, m.data(fmt.Sprintf("%s (%s)", change.Path, change.Action)))
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, Section(style(fmt.Sprintf("Other planned changes (%d)", len(other))), body, width)...)
	}
	if len(result.Planned) == 0 {
		lines = append(lines, Paragraph("The backup records no destination changes.", width, 0)...)
	}
	return lines
}

func (m RestoreModel) changesLines(title string, changes []skillsync.ResultChange, width int) []string {
	body := make([]string, 0, len(changes))
	for _, change := range changes {
		body = append(body, m.data(fmt.Sprintf("%s  %s", change.Action, change.Path)))
	}
	return Section(m.theme.Section.Render(title), body, width)
}

// Lines returns every wrapped screen line. View pages them; rendering is pure
// and performs no filesystem access.
func (m RestoreModel) Lines(width int) []string {
	if width <= 0 {
		return []string{""}
	}
	var lines []string
	switch m.stage {
	case RestoreStageList:
		lines = m.listHeaderLines(width)
		groups := m.listEntryGroups(width)
		if len(groups) > 0 {
			lines = append(lines, "")
			for _, group := range groups {
				lines = append(lines, group...)
			}
		}
		lines = append(lines, Paragraph(m.theme.Hint.Render("up/down select | enter details | r refresh | esc back | q quit from the shell"), width, 0)...)
	case RestoreStageDetails:
		lines = m.detailsLines(width)
	case RestoreStagePreview:
		lines = m.previewLines(width)
	case RestoreStageConfirm:
		lines = m.confirmLines(width)
	case RestoreStageRevalidating:
		lines = m.revalidatingLines(width)
	case RestoreStageRunning:
		lines = m.runningLines(width)
	case RestoreStageResult:
		lines = m.resultLines(width)
	}
	if m.cfg.ASCII {
		for i := range lines {
			lines[i] = toASCII(lines[i])
		}
	}
	return lines
}

// View renders the current stage bounded to the given cells. It adjusts the
// page so a moved list selection stays in view and never renders past height.
func (m RestoreModel) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := m.Lines(width)
	offset := m.scroll
	if m.stage == RestoreStageList {
		start := m.listSelectionStart(width)
		if start < offset || start >= offset+height {
			offset = start
		}
	}
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	return strings.Join(pageRows(lines, height, offset), "\n")
}

// data escapes unicode source data in ASCII mode so paths and reasons stay
// readable and cell-width safe on a 7-bit terminal.
func (m RestoreModel) data(text string) string {
	if m.cfg.ASCII {
		return escapeData(text)
	}
	return text
}

func restoreIdentity(backup skillsync.Backup) string {
	when := restoreWhen(backup)
	return when + "  " + restoreOperation(backup)
}

func restoreWhen(backup skillsync.Backup) string {
	if backup.Manifest.CreatedAt != "" {
		return backup.Manifest.CreatedAt
	}
	if backup.Manifest.Created != "" {
		return backup.Manifest.Created
	}
	if parsed, err := time.Parse("20060102T150405.000000000Z", backup.ID); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return "time not recorded"
}

func restoreOperation(backup skillsync.Backup) string {
	op := backup.Manifest.Operation
	switch {
	case op == "" || op == "apply":
		return "apply"
	case op == "restore" && backup.Manifest.RestoredFrom != "":
		return "restore from " + backup.Manifest.RestoredFrom
	case op == "restore":
		return "restore"
	default:
		return op
	}
}

func restoreSource(backup skillsync.Backup) string {
	if backup.Manifest.Source == "" {
		return "source not recorded"
	}
	return backup.Manifest.Source
}

func restoreFormat(backup skillsync.Backup) string {
	if backup.Legacy {
		return "legacy (no per-destination records)"
	}
	if backup.Manifest.Format > 0 {
		return fmt.Sprintf("format %d", backup.Manifest.Format)
	}
	return "format not recorded"
}

func restoreChangeCount(planned []skillsync.ResultChange) int {
	n := 0
	for _, change := range planned {
		if change.Action != skillsync.ChangeUpToDate {
			n++
		}
	}
	return n
}

// restoreBackupRoot mirrors the engine's public location rule for display and
// recovery text only. It never creates or reads the directory.
func restoreBackupRoot(repo, home string) string {
	if repo != "" {
		return filepath.Join(repo, "backups")
	}
	if home == "" {
		return "(unknown; no home configured)"
	}
	return filepath.Join(home, ".agentic-sdd", "backups")
}

// describeProblem turns an engine error into an honest recovery explanation.
// It never offers an execution action for an unsafe or unusable backup.
func (m RestoreModel) describeProblem(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	lower := strings.ToLower(text)
	detail := "Backup/restore error: " + m.data(text)
	switch {
	case strings.Contains(lower, "different home"):
		return detail + "\nThis backup was made for a different home. Recovery: choose a backup made for the configured --home, or reinstall to create a fresh backup. Nothing was changed."
	case strings.Contains(lower, "refusing"):
		return detail + "\nRestore refused for safety. Recovery: fix the reported destination (for example remove a symlink or special file) and run a fresh preview. Nothing was changed."
	case strings.Contains(lower, "no such file"), strings.Contains(lower, "not exist"):
		return detail + "\nThat backup is absent. Recovery: return to the list and choose an existing backup; press r to refresh. Nothing was changed."
	case strings.Contains(lower, "manifest"), strings.Contains(lower, "parse"), strings.Contains(lower, "unexpected entry"):
		return detail + "\nThe backup is corrupt or unreadable. Recovery: inspect or remove it with the explicit CLI; this screen will not offer confirmation for it. Nothing was changed."
	case strings.Contains(lower, "invalid backup id"):
		return detail + "\nRecovery: choose a backup whose ID matches YYYYMMDDTHHMMSS.nnnnnnnnnZ. Nothing was changed."
	default:
		return detail + "\nRestore is unavailable. Recovery: inspect the backup root and the configured --home/--repo, then press p to retry the preview. Nothing was changed."
	}
}
