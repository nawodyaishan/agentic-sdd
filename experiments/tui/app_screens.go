package tui

// app_screens.go owns the root shell's B4/B5 integration layer:
//
//   - Install & Manage becomes a hub with two child screens, the install
//     journey and the backup/restore journey;
//   - Activity & Help renders the ActivityModel (About, version, bounded
//     read-only logs, command and key reference);
//   - Home owns the once-per-session update check status, explicit retry and
//     the visible offline/disable control;
//   - the skippable onboarding overlay and the explicit-only trusted opener;
//   - the screen-side helper that turns a confirmed preview token into a
//     StartWriteMsg instead of calling the engine directly.
//
// Everything here is presentation and routing. Update and View never read the
// filesystem or the network: commands do that work outside the event loop.

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

// installSection selects the Install & Manage hub or one of its child screens.
type installSection int

const (
	// installSectionHub is the two-entry destination menu plus the read-only
	// preview command hints and backup directory.
	installSectionHub installSection = iota
	// installSectionInstall is the preview-first InstallModel journey.
	installSectionInstall
	// installSectionBackups is the RestoreModel backup/restore journey.
	installSectionBackups
)

// Default read-only operation endpoints, shown as copyable commands. They are
// hints only; the TUI never runs a shell.
const (
	previewGuideURL = ReadmeURL
)

// ---------------------------------------------------------------------------
// Confirmed-write seam
// ---------------------------------------------------------------------------

// startWriteCmd delivers one confirmed write request to the root shell. The
// screens never call appops.ApplyInstall/ApplyRestore themselves: only the
// root model's OpsHost may start a transaction.
func startWriteCmd(req WriteRequest) tea.Cmd {
	return func() tea.Msg { return StartWriteMsg{Request: req} }
}

// installWriteRequest builds the immutable write request for the confirmed
// install preview. It fails rather than guessing when the configured home or
// the selected clients are invalid.
func (m InstallModel) installWriteRequest() (WriteRequest, error) {
	selected := m.SelectedClients()
	if len(selected) == 0 {
		return WriteRequest{}, fmt.Errorf("%w: client selection is empty", ErrWriteInvalid)
	}
	if m.previewToken == "" {
		return WriteRequest{}, fmt.Errorf("%w: a validated preview is required before a write", ErrWriteInvalid)
	}
	opts := skillsync.Options{
		ClientIDs:         selected,
		WithOpenCode:      m.withOpenCode,
		OpenCodeFreeFirst: m.freeFirst,
	}
	req, err := appops.NewInstallRequest(m.repo, m.home, true, opts)
	if err != nil {
		return WriteRequest{}, err
	}
	command := "apply --clients " + strings.Join(selected, ",")
	if m.withOpenCode {
		command += " --with-opencode"
	}
	if m.freeFirst {
		command += " --opencode-free-first"
	}
	return WriteRequest{
		Kind:    WriteInstall,
		Install: req,
		Token:   m.previewToken,
		Command: command,
		Label:   fmt.Sprintf("install for %d client(s)", len(selected)),
	}, nil
}

// restoreWriteRequest builds the immutable write request for the confirmed
// restore preview.
func (m RestoreModel) restoreWriteRequest(backupID, token string) (WriteRequest, error) {
	if backupID == "" {
		return WriteRequest{}, fmt.Errorf("%w: a selected backup is required", ErrWriteInvalid)
	}
	if token == "" {
		return WriteRequest{}, fmt.Errorf("%w: a validated preview is required before a write", ErrWriteInvalid)
	}
	req, err := appops.NewRestoreRequest(m.cfg.Repo, m.cfg.Home, backupID, true)
	if err != nil {
		return WriteRequest{}, err
	}
	return WriteRequest{
		Kind:    WriteRestore,
		Restore: req,
		Token:   token,
		Command: "restore " + backupID,
		Label:   "restore backup " + backupID,
	}, nil
}

// resetForList returns the recovery screen to a clean backup browser. It never
// writes; it only discards a stale read-only transaction view.
func (m RestoreModel) resetForList() RestoreModel {
	m.clearTransaction()
	m.stage = RestoreStageList
	m.loading = false
	if m.cfg.Home != "" {
		m.listErr = ""
	}
	return m
}

// SelectionStartFor reports the wrapped line index where the highlighted
// backup begins, so the root viewport can keep the selection visible.
func (m RestoreModel) SelectionStartFor(width int) int {
	return m.listSelectionStart(width)
}

// Notice returns the screen's latest refusal or consent-drift explanation.
func (m InstallModel) Notice() string { return m.notice }

// Notice returns the screen's latest refusal or consent-drift explanation.
func (m RestoreModel) Notice() string { return m.notice }

// ---------------------------------------------------------------------------
// Home: automatic update check status and controls
// ---------------------------------------------------------------------------

// automaticUpdateCheckCmd starts the once-per-session automatic check. The
// manager deduplicates re-entry, resizing and re-rendering; a disabled or
// already-attempted session returns nil.
func (a *App) automaticUpdateCheckCmd() tea.Cmd {
	if a.updateMgr == nil {
		return nil
	}
	cmd, ok := a.updateMgr.AutomaticCheck(version.Version)
	if !ok {
		return nil
	}
	return cmd
}

// retryUpdateCheckCmd starts one explicit, user-triggered retry. It never runs
// automatically and does nothing while checks are disabled.
func (a *App) retryUpdateCheckCmd() tea.Cmd {
	if a.updateMgr == nil {
		return nil
	}
	cmd, ok := a.updateMgr.RetryCheck(version.Version)
	if !ok {
		a.notice = "Automatic update checks are disabled; press d to enable them before retrying. Nothing was downloaded."
		return nil
	}
	return cmd
}

// toggleUpdateChecks flips the session offline control. Disabling cancels any
// in-flight request cooperatively; enabling allows an explicit retry but never
// repeats the automatic attempt.
func (a *App) toggleUpdateChecks() {
	if a.updateMgr == nil {
		return
	}
	if a.updateMgr.IsDisabled() {
		a.updateMgr.Enable()
		a.notice = "Automatic update checks enabled for this session. Press r to check now; no request is made until you do."
		return
	}
	a.updateMgr.Disable()
	a.notice = "Automatic update checks disabled for this session. No update request is made while disabled."
}

// homeUpdateLines renders the Home update-check status without blocking local
// content: the first frame already shows the local inventory, then the status
// follows the asynchronous result.
func (a App) homeUpdateLines(width int) []string {
	if a.updateMgr == nil {
		return nil
	}
	lines := []string{a.theme.Section.Render("Update check")}
	lines = append(lines, "  "+a.updateMgr.RenderStatus(a.theme))

	control := "  r retry · d disable automatic checks"
	if a.updateMgr.IsDisabled() {
		control = "  Automatic checks are off for this session · d enable automatic checks"
	}
	lines = append(lines, wrapPath(control, width)...)
	for _, line := range wrapPath("  The check requests only the published release version from GitHub; no credentials or local inventory are sent. No automatic upgrade, download or browser launch happens.", width) {
		lines = append(lines, a.theme.Hint.Render(line))
	}
	for _, line := range wrapPath("  Online guide / README: "+previewGuideURL, width) {
		lines = append(lines, a.theme.Accent.Render(line))
	}
	lines = append(lines, wrapPath("  Press g to open the guide in your browser (explicit action), or a for the author profile.", width)...)
	return lines
}

// ---------------------------------------------------------------------------
// Install & Manage hub and child rendering
// ---------------------------------------------------------------------------

// installRouteLines renders the hub or the active child screen at width.
func (a App) installRouteLines(width int) []string {
	switch a.installSection {
	case installSectionInstall:
		return a.install.Lines(width)
	case installSectionBackups:
		return a.restore.Lines(width)
	default:
		return a.installHubLines(width)
	}
}

// installHubLines is the Install & Manage destination menu. It keeps the
// existing read-only preview command hints and the real backup directory
// visible, so the hub never hides the supported CLI escape hatch.
func (a App) installHubLines(width int) []string {
	var b strings.Builder

	b.WriteString(a.theme.Title.Render("Install & Manage"))
	b.WriteString("\n")
	b.WriteString(a.theme.Subtitle.Render("Preview-first install and update, plus backup recovery"))
	b.WriteString("\n\n")

	type hubEntry struct {
		key   string
		title string
		desc  string
	}
	entries := []hubEntry{
		{
			key:   "i",
			title: "Install or update",
			desc:  "Choose clients and optional packs, review the exact read-only impact, then confirm a default-No apply.",
		},
		{
			key:   "b",
			title: "Backups & restore",
			desc:  "Browse recorded backups, inspect one read-only plan, then confirm a default-No restore.",
		},
	}
	rows := make([]string, 0, len(entries))
	for i, entry := range entries {
		marker := " "
		if i == a.installHubIndex {
			marker = ">"
		}
		row := fmt.Sprintf("%s [%s] %s", marker, entry.key, entry.title)
		if i == a.installHubIndex {
			row = a.theme.Accent.Render(row)
		}
		rows = append(rows, row)
		rows = append(rows, "     "+entry.desc)
	}
	b.WriteString(strings.Join(Section(a.theme.Section.Render("Destinations"), rows, width), "\n"))
	b.WriteString("\n\n")

	b.WriteString(a.theme.Section.Render("Execution safety"))
	b.WriteString("\n")
	for _, line := range wrapPath("  "+WritePolicyText, width) {
		b.WriteString(a.theme.Warn.Render(line) + "\n")
	}
	b.WriteString("\n")

	repoFlag := ""
	if a.cfg.Repo != "" {
		repoFlag = " --repo " + quoteHintPath(a.cfg.Repo)
	}
	homeFlag := ""
	if a.cfg.Home != "" {
		homeFlag = " --home " + quoteHintPath(a.cfg.Home)
	}

	b.WriteString(a.theme.Section.Render("Read-only preview commands"))
	b.WriteString("\n")
	for _, line := range wrapPath("  Preview core skills: agentic-sdd preview"+repoFlag+homeFlag, width) {
		b.WriteString(a.theme.Accent.Render(line) + "\n")
	}
	for _, line := range wrapPath("  Preview with OpenCode pack: agentic-sdd preview --with-opencode"+repoFlag+homeFlag, width) {
		b.WriteString(a.theme.Accent.Render(line) + "\n")
	}
	for _, line := range wrapPath("  Preview a restore: agentic-sdd restore preview BACKUP_ID"+repoFlag+homeFlag, width) {
		b.WriteString(a.theme.Accent.Render(line) + "\n")
	}
	b.WriteString("\n")

	b.WriteString(a.theme.Section.Render("Backups"))
	b.WriteString("\n")
	backupRoot := a.catalog.BackupRoot
	if backupRoot == "" {
		backupRoot = restoreBackupRoot(a.catalog.Repo, a.catalog.Home)
	}
	for _, line := range wrapPath("  Backups directory: "+backupRoot, width) {
		b.WriteString(a.theme.Accent.Render(line) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(a.theme.Hint.Render("hub controls: j/k select · i install · b backups · Enter open · Esc navigation"))

	return strings.Split(b.String(), "\n")
}

// handleInstallRouteKey routes hub, install and backup keys. Read-only and
// write transactions stay inside the child models; the root model only
// chooses the destination, resets its own scroll on a stage change and keeps
// a backup selection visible.
func (a *App) handleInstallRouteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if a.installSection == installSectionHub {
		switch key {
		case "j", "down":
			a.installHubIndex = (a.installHubIndex + 1) % 2
			return *a, nil
		case "k", "up":
			a.installHubIndex = (a.installHubIndex - 1 + 2) % 2
			return *a, nil
		case "i":
			return *a, a.openInstallScreen()
		case "b":
			return *a, a.openBackupsScreen()
		case "enter":
			if a.installHubIndex == 1 {
				return *a, a.openBackupsScreen()
			}
			return *a, a.openInstallScreen()
		case "esc":
			a.focus = FocusNav
			return *a, nil
		}
		return *a, nil
	}

	width, height := a.childContentWidth(), a.childBodyHeight()

	if a.installSection == installSectionInstall {
		// The result screen has no internal back action: Enter or Esc closes
		// this journey and returns to the hub.
		if a.install.Step() == InstallStepResult && (key == "esc" || key == "enter") {
			return *a, a.closeInstallHub("")
		}
		// Review, preview, confirmation and result scroll the whole page; the
		// root viewport owns that offset so nothing is hidden.
		if installScrollsAt(a.install.Step()) && isScrollKey(key) {
			a.scrollChildContent(key, width, height)
			return *a, nil
		}
		prev := a.install.Step()
		updated, cmd := a.install.Update(msg)
		if m, ok := updated.(InstallModel); ok {
			a.install = m
		}
		if a.install.Step() != prev {
			a.contentScroll = 0
		}
		return *a, cmd
	}

	// Backups & restore.
	if a.restore.Stage() == RestoreStageList && key == "esc" {
		return *a, a.closeInstallHub("")
	}
	if a.restore.Stage() != RestoreStageList && isScrollKey(key) {
		a.scrollChildContent(key, width, height)
		return *a, nil
	}
	prev := a.restore.Stage()
	updated, cmd := a.restore.Update(msg)
	a.restore = updated
	if a.restore.Stage() != prev {
		a.contentScroll = 0
	}
	if a.restore.Stage() == RestoreStageList {
		a.revealRestoreSelection(width, height)
	}
	return *a, cmd
}

// installScrollsAt reports the install stages whose j/k moves the page rather
// than a selection cursor.
func installScrollsAt(step InstallStep) bool {
	switch step {
	case InstallStepReview, InstallStepPreview, InstallStepConfirm, InstallStepResult:
		return true
	default:
		return false
	}
}

// isScrollKey reports the keys the root viewport handles for a child screen.
func isScrollKey(key string) bool {
	switch key {
	case "up", "k", "down", "j", "pgup", "pageup", "ctrl+u", "pgdown", "pagedown", "ctrl+d", "home", "g":
		return true
	default:
		return false
	}
}

// scrollChildContent moves the root viewport offset with the same step sizes
// the standalone screens document.
func (a *App) scrollChildContent(key string, width, height int) {
	switch key {
	case "up", "k":
		a.contentScroll--
	case "down", "j":
		a.contentScroll++
	case "pgup", "pageup", "ctrl+u":
		a.contentScroll -= 4
	case "pgdown", "pagedown", "ctrl+d":
		a.contentScroll += 4
	case "home", "g":
		a.contentScroll = 0
	}
	a.clampContentScroll(width, height)
}

// clampContentScroll keeps a scroll offset inside the rendered content.
func (a *App) clampContentScroll(width, height int) {
	maxOffset := len(a.getContentLines(width)) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if a.contentScroll < 0 {
		a.contentScroll = 0
	}
	if a.contentScroll > maxOffset {
		a.contentScroll = maxOffset
	}
}

// revealRestoreSelection scrolls just enough to keep the highlighted backup
// row inside the viewport.
func (a *App) revealRestoreSelection(width, height int) {
	start := a.restore.SelectionStartFor(width)
	if start < a.contentScroll {
		a.contentScroll = start
	}
	if start >= a.contentScroll+height {
		a.contentScroll = start - height + 1
	}
	a.clampContentScroll(width, height)
}

// openInstallScreen activates the install journey. Its model performs no I/O
// until an operation command runs.
func (a *App) openInstallScreen() tea.Cmd {
	a.installSection = installSectionInstall
	a.contentScroll = 0
	a.install.SetSize(a.childContentWidth(), a.childBodyHeight())
	return nil
}

// openBackupsScreen activates the recovery screen and starts one bounded
// read-only backup discovery.
func (a *App) openBackupsScreen() tea.Cmd {
	a.installSection = installSectionBackups
	a.contentScroll = 0
	a.restore = a.restore.resetForList()
	a.restore.SetSize(a.childContentWidth(), a.childBodyHeight())
	m, cmd := a.restore.Reload()
	a.restore = m
	return cmd
}

// closeInstallHub returns to the destination hub. The optional notice is shown
// by the hub's parent notice section.
func (a *App) closeInstallHub(notice string) tea.Cmd {
	a.installSection = installSectionHub
	a.contentScroll = 0
	if notice != "" {
		a.notice = notice
	}
	return nil
}

// childContentWidth matches the root viewport's body width exactly, so child
// wrapping and the root pager agree on every row.
func (a App) childContentWidth() int {
	width := a.width
	if a.sidebarVisible() {
		width -= 19
	}
	width--
	if width < 1 {
		return 1
	}
	return width
}

// childBodyHeight matches the root viewport's body row budget.
func (a App) childBodyHeight() int {
	rows := a.height - a.headerRowCount() - 2
	if rows < 1 {
		return 1
	}
	return rows
}

// childWritePending reports a child screen that has committed to a write but
// whose StartWriteMsg may not have reached the root model yet. Navigation and
// quit keys are ignored in that window, exactly like an accepted write.
func (a App) childWritePending() bool {
	return a.install.Step() == InstallStepRunning || a.restore.Busy()
}

// ---------------------------------------------------------------------------
// Activity & Help
// ---------------------------------------------------------------------------

// ActivityReloadedMsg carries a refreshed ActivityModel built outside Update.
type ActivityReloadedMsg struct {
	Model ActivityModel
}

// activityRouteLines renders the ActivityModel and the TUI-only controls that
// are not part of the model's CLI reference.
func (a App) activityRouteLines(width int) []string {
	lines := a.activity.Lines(width)
	lines = append(lines, "")
	guideURL := ReadmeURL
	authorURL := AuthorURL
	if a.cfg.Color && !a.ascii {
		guideURL = Hyperlink(guideURL, LinkReadme, true)
		authorURL = Hyperlink(authorURL, LinkAuthor, true)
	}
	controls := []string{
		"g           Open the online guide in your browser: " + guideURL,
		"a           Open the author profile: " + authorURL,
		"r           Retry the update check and refresh the bounded log list",
		"d           Disable or enable automatic update checks for this session",
		"[ / ]       Show older / newer operation logs",
		"j/k PgUp/Dn Scroll this reference; logs are read-only and bounded",
	}
	lines = append(lines, Section(a.theme.Section.Render("TUI Controls"), controls, width)...)
	return lines
}

// reloadActivityCmd rescans the log directory and the selected log content
// outside Update. The activity model is copied; a stale scan cannot mutate the
// live model directly.
func (a App) reloadActivityCmd() tea.Cmd {
	model := a.activity
	return func() tea.Msg {
		model.ReloadLogs()
		return ActivityReloadedMsg{Model: model}
	}
}

// selectActivityLogCmd moves the bounded log selection outside Update.
func (a App) selectActivityLogCmd(delta int) tea.Cmd {
	model := a.activity
	return func() tea.Msg {
		if delta > 0 {
			model.SelectNextLog()
		} else {
			model.SelectPrevLog()
		}
		return ActivityReloadedMsg{Model: model}
	}
}

// handleActivityRouteKey routes Activity keys. Key handling that would read
// files is dispatched as a command; Update itself does no I/O.
func (a *App) handleActivityRouteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	width, height := a.childContentWidth(), a.childBodyHeight()

	switch key {
	case "esc":
		a.focus = FocusNav
		return *a, nil
	case "r":
		cmds := make([]tea.Cmd, 0, 2)
		if cmd := a.retryUpdateCheckCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, a.reloadActivityCmd())
		return *a, tea.Batch(cmds...)
	case "d":
		a.toggleUpdateChecks()
		return *a, nil
	case "g":
		return *a, a.openLinkCmd(LinkReadme)
	case "a":
		return *a, a.openLinkCmd(LinkAuthor)
	case "]", "n":
		return *a, a.selectActivityLogCmd(1)
	case "[", "p":
		return *a, a.selectActivityLogCmd(-1)
	}
	if isScrollKey(key) {
		a.scrollChildContent(key, width, height)
	}
	return *a, nil
}

// ---------------------------------------------------------------------------
// Onboarding overlay
// ---------------------------------------------------------------------------

// OpenLinkResultMsg carries the trusted opener's explanation back to Update.
type OpenLinkResultMsg struct {
	Result OpenResult
}

// openLinkCmd performs one explicitly requested open. It is the only path that
// can start a process, and it runs in a command, never in Update or View.
func (a App) openLinkCmd(link Link) tea.Cmd {
	opener := a.opener
	if opener == nil {
		return nil
	}
	return func() tea.Msg {
		return OpenLinkResultMsg{Result: opener.Open(link)}
	}
}

// openOnboardingCmd builds the skippable tour from the read-only catalog
// summary. It performs no I/O and never opens a link.
func (a *App) openOnboardingCmd() tea.Cmd {
	model := NewOnboardingModel(a.cfg, NewOnboardingSummary(a.catalog))
	model.SetSize(a.childContentWidth(), a.childBodyHeight())
	a.onboarding = &model
	a.contentScroll = 0
	return nil
}

// handleOnboardingKey routes overlay keys. Scroll keys are handled by the root
// viewport; navigation, skip and explicit link keys go to the tour.
func (a *App) handleOnboardingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "?" {
		a.helpOpen = true
		a.helpScroll = 0
		return *a, nil
	}
	if key == "ctrl+c" {
		return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonInterruptKey})
	}
	if key == "q" {
		return *a, a.requestShutdown(ShutdownRequestMsg{Reason: ShutdownReasonQuitKey})
	}
	if isScrollKey(key) {
		a.scrollChildContent(key, a.childContentWidth(), a.childBodyHeight())
		return *a, nil
	}
	if a.onboarding == nil {
		return *a, nil
	}
	// The finish step is complete, not cancelled: Esc closes the overlay while
	// keeping the completed outcome and the visible, copyable prompt.
	if a.onboarding.Finished() && key == "esc" {
		a.onboarding = nil
		a.notice = "Onboarding closed. The first router prompt stays available in Activity & Help; nothing was copied and no agent ran."
		return *a, nil
	}

	updated, cmd := a.onboarding.Update(msg)
	if model, ok := updated.(OnboardingModel); ok {
		a.onboarding = &model
	}
	if a.onboarding != nil && a.onboarding.Done() {
		outcome := a.onboarding.Outcome()
		a.onboarding = nil
		a.notice = "Onboarding " + outcome.String() + ". No files were written, nothing was copied, no browser opened and no agent ran."
	}
	return *a, cmd
}

// ---------------------------------------------------------------------------
// Trusted screens footer and helper accessors
// ---------------------------------------------------------------------------

// footerOverride returns the contextual footer for screens whose keys are not
// covered by the generic navigation footer. An empty result keeps the generic
// footer (and never runs while the help overlay owns the screen).
func (a App) footerOverride() string {
	if a.helpOpen {
		return ""
	}
	if a.onboarding != nil {
		if a.width < 60 {
			return "Enter next s skip Esc cancel q quit"
		}
		if a.width < 90 {
			return "Enter next s skip Esc cancel o/a links q quit"
		}
		return "Enter next s skip Esc cancel o guide a author j/k scroll ? help q quit"
	}
	switch a.activeRoute {
	case RouteInstall:
		switch a.installSection {
		case installSectionHub:
			if a.width < 60 {
				return "i install b backups q quit"
			}
			if a.width < 80 {
				return "j/k select i install b backups Esc nav q quit"
			}
			return "j/k select i install b backups Enter open Esc nav q quit"
		case installSectionInstall:
			if a.width < 60 {
				return "Esc back y confirm q quit"
			}
			return "Esc back y confirm PgUp/Dn scroll ? help q quit"
		default:
			if a.width < 60 {
				return "Esc back y confirm q quit"
			}
			return "Esc back Enter open y confirm r refresh ? help q quit"
		}
	case RouteActivity:
		if a.width < 60 {
			return "r retry d check g guide [ ] logs q quit"
		}
		return "r retry d check g guide a author [ ] logs j/k scroll q quit"
	default:
		return ""
	}
}

// homeLogDir mirrors the private log location the CLI and the write host use.
// It never creates or reads the directory.
func homeLogDir(home string) string {
	if strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".agentic-sdd", "logs")
}
