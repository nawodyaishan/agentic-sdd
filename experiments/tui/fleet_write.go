package tui

// Fleet deployment review and apply dispatch. The screen owns only view and
// draft state: it builds a typed WriteRequest from an already-previewed plan
// and returns StartWriteMsg through tea. It never calls appops.ApplyFleet or
// fleet.ApplyPlan directly, never writes a file and never renders an optimistic
// success before the host returns a structured result.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/fleet"
)

// FleetPlanPreviewMsg delivers the read-only deployment plan projection.
type FleetPlanPreviewMsg struct {
	Generation uint64
	Source     string
	Preview    FleetPlanPreview
	Err        error
}

// fleetWriteOutcome is the display-only result of one fleet write. It is set
// only from a real WriteResultMsg or WriteRefusedMsg.
type fleetWriteOutcome struct {
	Headline     string
	Detail       string
	BackupID     string
	LogPath      string
	Rollback     string
	NeedsPreview bool
}

// Empty reports whether no write has completed for this kind yet.
func (o fleetWriteOutcome) Empty() bool { return o.Headline == "" }

// requestPlanPreview queues one fresh, non-adopting plan read from the saved
// source. It performs no I/O itself. A dirty draft is never planned, because the
// engine plans only the saved source.
func (m FleetModel) requestPlanPreview() (FleetModel, tea.Cmd) {
	if m.planner == nil {
		return m, nil
	}
	if m.dirtyCount > 0 {
		m.planErr = ""
		m.planError = nil
		m.planPreview = FleetPlanPreview{}
		return m, nil
	}
	source := strings.TrimSpace(m.sourcePath)
	if source == "" {
		m.planPreview = FleetPlanPreview{Available: false, Message: "Select an explicit fleet.json source before planning."}
		m.planErr = ""
		m.planError = nil
		return m, nil
	}
	m.planGen++
	m.planLoading = true
	m.planErr = ""
	m.planError = nil
	generation := m.planGen
	planner := m.planner
	return m, func() tea.Msg {
		preview, err := planner.PreviewPlan(context.Background(), source)
		return FleetPlanPreviewMsg{Generation: generation, Source: source, Preview: preview, Err: err}
	}
}

// requestApply opens the default-No apply confirmation when, and only when, a
// fresh conflict-free plan exists, the saved source matches the draft, the live
// capability is enabled and a resolved home is available. It never dispatches
// a write itself.
func (m FleetModel) requestApply() (FleetModel, tea.Cmd) {
	if m.writePending {
		return m, nil
	}
	if !m.live() {
		m.applyOutcome = fleetWriteOutcome{
			Headline: "Preview only",
			Detail:   "Live fleet deployment is disabled in this build. No key can start an apply; review the plan and use the documented CLI only after the reviewed live capability is enabled.",
		}
		m.confirmApply = false
		return m, nil
	}
	if m.dirtyCount > 0 {
		m.applyOutcome = fleetWriteOutcome{
			Headline: "source not saved",
			Detail:   "The personal source has unsaved edits. Saving it is a separate operation (B4 source save is not authorized in this build); apply always plans from the saved source.",
		}
		m.confirmApply = false
		return m, nil
	}
	if strings.TrimSpace(m.cfg.Home) == "" {
		m.applyOutcome = fleetWriteOutcome{Headline: "unavailable", Detail: "Fleet apply needs a resolved home; the TUI was started without one."}
		m.confirmApply = false
		return m, nil
	}
	if !m.planPreview.Available || m.planPreview.Plan == nil || m.planPreview.Token == "" {
		m.applyOutcome = fleetWriteOutcome{Headline: "no fresh plan", Detail: "Generate a fresh deployment plan from the saved source first (v), then review every file change."}
		m.confirmApply = false
		return m, nil
	}
	if m.planPreview.Counts[fleet.FileOpConflict] > 0 {
		m.applyOutcome = fleetWriteOutcome{
			Headline: "blocked by conflicts",
			Detail:   "Apply is blocked while conflicts remain. Resolve them with the documented CLI; the TUI offers no automatic import, overwrite or bulk repair.",
		}
		m.confirmApply = false
		return m, nil
	}
	m.confirmApply = true
	return m, nil
}

// confirmApplyAndDispatch converts the confirmed plan into one typed
// WriteRequest and hands it to the root shell. A second confirmation while a
// write is pending is inert.
func (m FleetModel) confirmApplyAndDispatch() (FleetModel, tea.Cmd) {
	if !m.confirmApply || m.writePending || !m.live() {
		m.confirmApply = false
		return m, nil
	}
	preview := m.planPreview
	if !preview.Available || preview.Plan == nil || preview.Token == "" || strings.TrimSpace(m.cfg.Home) == "" {
		m.confirmApply = false
		return m, nil
	}
	m.confirmApply = false
	m.writePending = true
	m.writeKind = WriteFleetApply
	req := WriteRequest{
		Kind: WriteFleetApply,
		Fleet: FleetWriteRequest{
			Home:       m.cfg.Home,
			ConfigPath: m.sourcePath,
			PlanID:     preview.PlanID,
			Plan:       preview.Plan,
			Token:      preview.Token,
			Label:      "fleet apply " + preview.PlanID,
			Source:     m.sourcePath,
		},
	}
	return m, startWriteCmd(req)
}

// handleFleetWriteResult records the real structured outcome and refreshes the
// Fleet read. Nothing is marked successful before this message exists.
func (m FleetModel) handleFleetWriteResult(msg WriteResultMsg) (FleetModel, tea.Cmd) {
	if !msg.Kind.isFleet() {
		return m, nil
	}
	m.writePending = false
	m.writeStep = ""
	summary := SummarizeWriteResult(msg)
	outcome := fleetWriteOutcome{
		Headline:     summary.Headline,
		Detail:       fleetResultDetail(msg, summary),
		BackupID:     summary.BackupID,
		LogPath:      summary.LogPath,
		Rollback:     summary.Rollback,
		NeedsPreview: summary.NeedsPreview,
	}
	switch msg.Kind {
	case WriteFleetApply:
		m.applyOutcome = outcome
		m.applyOutcome.BackupID = fleetApplyBackup(msg, summary.BackupID)
		m.planPreview = FleetPlanPreview{}
		m.confirmApply = false
		m.reviewScroll = 0
	case WriteFleetRestore:
		m.restoreOutcome = outcome
		m.restoreOutcome.BackupID = fleetRestoreBackup(msg, summary.BackupID)
		m.restorePreview = FleetRestorePreview{}
		m.confirmRestore = false
		m.recoveryScroll = 0
	case WriteSourceSave:
		m.sourceOutcome = outcome
	}

	updated, cmd := m.Reload()
	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if msg.Kind == WriteFleetRestore {
		refreshed, backupCmd := updated.requestBackups()
		updated = refreshed
		if backupCmd != nil {
			cmds = append(cmds, backupCmd)
		}
	}
	switch len(cmds) {
	case 0:
		return updated, nil
	case 1:
		return updated, cmds[0]
	default:
		return updated, tea.Batch(cmds...)
	}
}

// fleetResultDetail prefers the fleet engine's exact counts over the generic
// install wording.
func fleetResultDetail(msg WriteResultMsg, summary WriteSummary) string {
	if msg.Err != nil {
		return summary.Detail
	}
	switch {
	case msg.Result.FleetApply != nil:
		apply := msg.Result.FleetApply
		if apply.NoOp {
			return "No managed OpenCode file changed; the installed files already match the plan."
		}
		return fmt.Sprintf("Created %d, updated %d, removed %d managed file(s).", apply.Created, apply.Updated, apply.Removed)
	case msg.Result.FleetRestore != nil:
		restore := msg.Result.FleetRestore
		if restore.NoOp {
			return "No file changed; the installed files already match the backup."
		}
		return fmt.Sprintf("Restored %d file(s) and removed %d file(s) created by the recorded transaction.", restore.Restored, restore.Removed)
	default:
		return summary.Detail
	}
}

func fleetApplyBackup(msg WriteResultMsg, fallback string) string {
	if msg.Result.FleetApply != nil && msg.Result.FleetApply.BackupID != "" {
		return msg.Result.FleetApply.BackupID
	}
	return fallback
}

func fleetRestoreBackup(msg WriteResultMsg, fallback string) string {
	if msg.Result.FleetRestore != nil && msg.Result.FleetRestore.BackupID != "" {
		return msg.Result.FleetRestore.BackupID
	}
	return fallback
}

// live reports the reviewed build capability. It is never a per-screen toggle
// and is not a user preference.
func (m FleetModel) live() bool { return m.cfg.FleetLive }

// WritePending reports a fleet write this screen has dispatched but whose
// start/result has not been observed yet. The root shell treats it exactly like
// an accepted write so navigation cannot race a second dispatch.
func (m FleetModel) WritePending() bool { return m.writePending }

// deploymentReviewLines renders the Review Changes view: the source review from
// W2-B when edits are staged, plus the fresh deployment plan and its complete
// scrollable diff after the source is saved.
func (m FleetModel) deploymentReviewLines(width int) []string {
	var lines []string

	header := "Install & Manage / Fleet Manager · Review Changes"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)
	lines = append(lines, m.renderTabBar(FleetViewReview))

	sub := "Review staged source edits and the fresh OpenCode deployment plan (read-only)"
	if m.theme.Color {
		sub = m.theme.Subtitle.Render(sub)
	}
	lines = append(lines, sub, "")

	if m.previewLoading {
		lines = append(lines, "  Computing source diff preview...", "")
	} else if m.previewErr != "" {
		errText := "  Preview failed: " + m.previewErr
		if m.theme.Color {
			errText = m.theme.Bad.Render(errText)
		}
		lines = append(lines, errText, "")
	}

	// 1. Impact summary first (FR-08/FR-09, AC-19).
	var impactRows []string
	src := m.sourcePath
	if src == "" {
		src = m.status.SourcePath
	}
	if src == "" {
		src = "(none selected)"
	}
	impactRows = append(impactRows, fmt.Sprintf("Staged in-memory edits: %d edit(s)", m.dirtyCount))
	impactRows = append(impactRows, fmt.Sprintf("Personal source file: %s", src))

	baseDig := m.baseDigest
	if baseDig == "" {
		baseDig = m.sourcePreview.BaseDigest
	}
	if baseDig == "" {
		baseDig = "(base digest pending preview)"
	}
	impactRows = append(impactRows, fmt.Sprintf("Base digest (CAS): %s", baseDig))
	impactRows = append(impactRows, "Installed agent files: Unchanged (0 OpenCode files modified)")
	impactRows = append(impactRows, "Save status: NOT SAVED (in-memory draft only; B4 source save not authorized)")
	lines = append(lines, Section("Impact summary", impactRows, width)...)
	lines = append(lines, "")

	// 2. Exact source diff.
	var diffLines []string
	if m.dirtyCount == 0 {
		diffLines = []string{"  No changes staged. Select an instance in Profiles (press p) to stage a model or enablement change."}
	} else if len(m.sourcePreview.DiffLines) > 0 {
		diffLines = m.sourcePreview.DiffLines
	} else if m.sourcePreview.Message != "" {
		diffLines = []string{"  " + m.sourcePreview.Message}
	} else {
		diffLines = m.fallbackDiffLines()
	}

	maxDiffRows := m.height - 18
	if maxDiffRows < 6 {
		maxDiffRows = 6
	}

	diffTitle := "Personal source diff (fleet.json)"
	if m.theme.Color {
		diffTitle = m.theme.Section.Render(diffTitle)
	}
	lines = append(lines, diffTitle, "")
	lines = append(lines, m.renderReviewDiff(diffLines, width, maxDiffRows)...)
	lines = append(lines, "")

	notice := "Note: Staged edits remain in memory only. No save button is offered (B4 not authorized)."
	if m.theme.Color {
		notice = m.theme.Hint.Render(notice)
	}
	lines = append(lines, notice, "")
	lines = append(lines, m.deploymentSection(width)...)
	lines = append(lines, "")

	footer := "v: fresh plan · a: apply (live only) · j/k: scroll · p/Esc: profiles · o: overview · H: Home · ?: help"
	if m.theme.Color {
		footer = m.theme.Hint.Render(footer)
	}
	lines = append(lines, footer)
	return lines
}

// deploymentSection renders the fresh plan identity, counts, last real result
// and the default-No apply prompt.
func (m FleetModel) deploymentSection(width int) []string {
	var lines []string
	title := "OpenCode deployment plan"
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	lines = append(lines, title, "")

	if m.dirtyCount > 0 {
		lines = append(lines, "  The personal source has unsaved edits. Save it before deployment; B4 source save is not authorized in this build.")
		lines = append(lines, "  No plan is generated from an unsaved draft.")
		if !m.applyOutcome.Empty() {
			lines = append(lines, "")
			lines = append(lines, m.renderApplyOutcome(width)...)
		}
		return lines
	}

	capability := "Deployment capability: Preview only — live deployment is disabled in this build."
	if m.live() {
		capability = "Deployment capability: live (reviewed build)."
	}
	if m.theme.Color && !m.live() {
		capability = m.theme.Warn.Render(capability)
	}
	lines = append(lines, "  "+capability)

	if m.planLoading {
		lines = append(lines, "  Computing a fresh plan from the saved source...")
		return lines
	}
	if m.planErr != "" {
		planErr := m.planError
		if planErr == nil {
			planErr = errors.New(m.planErr)
		}
		lines = append(lines, m.guidanceLines(width, planErr)...)
		return lines
	}
	if !m.planPreview.Available {
		message := m.planPreview.Message
		if message == "" {
			message = "No deployment plan yet."
		}
		lines = append(lines, "  "+message)
		lines = append(lines, "  Press v to generate a fresh plan from the saved source.")
		return lines
	}

	preview := m.planPreview
	rows := []string{
		fmt.Sprintf("Plan identity: %s", preview.PlanID),
		fmt.Sprintf("Scope: %s · renderer %s", nonEmpty(preview.Scope, "global"), nonEmpty(preview.RendererVersion, "1")),
		fmt.Sprintf("Source revision: %s", nonEmpty(shortDigest(preview.SourceRev), "(none)")),
		fmt.Sprintf("Config digest: %s", nonEmpty(shortDigest(preview.ConfigDigest), "(none)")),
		fmt.Sprintf("Changes: %d create · %d update · %d remove · %d no-op · %d conflict",
			preview.Counts[fleet.FileOpCreate], preview.Counts[fleet.FileOpUpdate],
			preview.Counts[fleet.FileOpRemove], preview.Counts[fleet.FileOpNoOp],
			preview.Counts[fleet.FileOpConflict]),
		"Order: removals and conflicts are listed before routine updates.",
	}
	lines = append(lines, Section("Deployment impact", rows, width)...)
	lines = append(lines, "")

	if preview.Counts[fleet.FileOpConflict] > 0 {
		warning := "Apply is blocked while conflicts remain. Resolve them with the documented CLI; the TUI offers no automatic import, overwrite or bulk repair."
		if m.theme.Color {
			warning = m.theme.Warn.Render(warning)
		}
		lines = append(lines, "  "+warning, "")
	}

	if !m.applyOutcome.Empty() {
		lines = append(lines, m.renderApplyOutcome(width)...)
		lines = append(lines, "")
	}

	if m.confirmApply {
		prompt := "Apply to OpenCode files? [y/N]"
		if m.theme.Color {
			prompt = m.theme.Warn.Bold(true).Render(prompt)
		}
		lines = append(lines, "  "+prompt)
		lines = append(lines, fmt.Sprintf("  This writes the %d planned managed file change(s) under %s.",
			preview.Counts[fleet.FileOpCreate]+preview.Counts[fleet.FileOpUpdate]+preview.Counts[fleet.FileOpRemove], preview.Plan.TargetRoot))
		lines = append(lines, "  No Git operation, no prompt/permission change and no model call occurs.")
		lines = append(lines, "  Press y to confirm; n, Enter or Esc keeps the default No.")
		lines = append(lines, "")
	}

	maxRows := m.height - 24
	if maxRows < 6 {
		maxRows = 6
	}
	diffTitle := "Deployment diff (per file)"
	if m.theme.Color {
		diffTitle = m.theme.Section.Render(diffTitle)
	}
	lines = append(lines, diffTitle, "")
	lines = append(lines, m.renderReviewDiff(preview.DiffLines, width, maxRows)...)
	return lines
}

// renderApplyOutcome renders the real result of the last apply. It is never
// shown before WriteResultMsg.
func (m FleetModel) renderApplyOutcome(width int) []string {
	rows := []string{
		"Outcome: " + m.applyOutcome.Headline,
		"Detail: " + m.applyOutcome.Detail,
	}
	if m.applyOutcome.BackupID != "" {
		rows = append(rows, "Backup: "+m.applyOutcome.BackupID)
	}
	if m.applyOutcome.Rollback != "" {
		rows = append(rows, "Rollback: "+m.applyOutcome.Rollback)
	}
	if m.applyOutcome.LogPath != "" {
		rows = append(rows, "Operation log: "+m.applyOutcome.LogPath)
	}
	title := "Last apply result"
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	return Section(title, rows, width)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// guidanceLines translates a typed error into what happened / what stayed
// unchanged / next safe action without masking the technical cause and without
// offering adoption, force or bulk repair.
func (m FleetModel) guidanceLines(width int, err error) []string {
	what, unchanged, next := fleetErrorGuidance(err)
	rows := []string{
		"What happened: " + what,
		"What stayed unchanged: " + unchanged,
		"Next safe action: " + next,
	}
	title := "Error detail"
	if m.theme.Color {
		title = m.theme.Bad.Render(title)
	}
	return Section(title, rows, width)
}

// fleetErrorGuidance maps typed engine errors to honest recovery guidance.
func fleetErrorGuidance(err error) (what, unchanged, next string) {
	message := redactError(err)
	switch {
	case err == nil:
		return "No error.", "All observed state is unchanged.", "Continue reviewing."
	case errors.Is(err, appops.ErrNeedsPreview):
		return "The preview no longer matches the current source, plan or target state.", "No file was written.", "Generate a fresh plan, review it and confirm again."
	case errors.Is(err, fleet.ErrStalePlan):
		return "The plan or its source/rendered content changed after it was generated.", "No file was written.", "Discard this plan, reload the source and generate a fresh plan."
	case errors.Is(err, fleet.ErrLegacyProvenance):
		return "This backup or record has legacy/unknown provenance, so its last-applied state cannot be trusted.", "No file was written and no state was guessed.", "Select a completed managed backup or use the documented CLI recovery path."
	case errors.Is(err, fleet.ErrUnsafeTarget):
		return "A target file was modified, recreated or is not a safe regular file.", "No file was overwritten or deleted.", "Inspect the target and restore only after it matches the recorded state."
	case errors.Is(err, fleet.ErrPlanConflict):
		return "The plan contains a conflict that needs a human decision.", "No conflicting file was touched.", "Review the conflict and resolve it with the documented CLI."
	default:
		return nonEmpty(message, "The operation could not be completed."), "No file was written by this step.", "Reload the status, then retry the read-only preview."
	}
}
