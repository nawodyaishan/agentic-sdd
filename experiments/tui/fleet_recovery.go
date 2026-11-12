package tui

// Fleet recovery view. It lists managed fleet backups, previews exactly what a
// restore would change, requires a default-No confirmation and dispatches one
// typed WriteFleetRestore through the shared single write slot. It never calls
// RestoreBackup or appops directly, never deletes an unrelated file and never
// offers adoption, force or bulk repair.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/fleet"
)

// FleetBackupsMsg delivers the read-only managed backup list.
type FleetBackupsMsg struct {
	Generation uint64
	Backups    []FleetBackupView
	Err        error
}

// FleetRestorePreviewMsg delivers the read-only restore projection.
type FleetRestorePreviewMsg struct {
	Generation uint64
	BackupID   string
	Preview    FleetRestorePreview
	Err        error
}

// requestBackups queues one read-only managed-backup listing.
func (m FleetModel) requestBackups() (FleetModel, tea.Cmd) {
	if m.planner == nil {
		return m, nil
	}
	m.recoveryGen++
	m.recoveryLoading = true
	m.recoveryErr = ""
	m.recoveryError = nil
	generation := m.recoveryGen
	planner := m.planner
	return m, func() tea.Msg {
		backups, err := planner.ListBackups(context.Background())
		return FleetBackupsMsg{Generation: generation, Backups: backups, Err: err}
	}
}

// selectedBackup returns the highlighted backup, clamped to the current list.
func (m FleetModel) selectedBackup() (FleetBackupView, bool) {
	if len(m.backups) == 0 {
		return FleetBackupView{}, false
	}
	index := m.backupIndex
	if index < 0 {
		index = 0
	}
	if index >= len(m.backups) {
		index = len(m.backups) - 1
	}
	return m.backups[index], true
}

// requestRestorePreview queues the read-only current-versus-restore projection
// for the selected managed backup. It is allowed in preview-only builds because
// it writes nothing.
func (m FleetModel) requestRestorePreview() (FleetModel, tea.Cmd) {
	backup, ok := m.selectedBackup()
	if !ok || m.planner == nil {
		return m, nil
	}
	m.restoreGen++
	m.restoreLoading = true
	m.restoreErr = ""
	m.restoreError = nil
	m.restorePreview = FleetRestorePreview{}
	m.confirmRestore = false
	generation := m.restoreGen
	planner := m.planner
	return m, func() tea.Msg {
		preview, err := planner.PreviewRestore(context.Background(), backup.ID)
		return FleetRestorePreviewMsg{Generation: generation, BackupID: backup.ID, Preview: preview, Err: err}
	}
}

// requestRestore opens the default-No restore confirmation for an already
// previewed managed backup. It never dispatches the write itself.
func (m FleetModel) requestRestore() (FleetModel, tea.Cmd) {
	if m.writePending {
		return m, nil
	}
	if !m.live() {
		m.restoreOutcome = fleetWriteOutcome{
			Headline: "Preview only",
			Detail:   "Live fleet restore is disabled in this build. No key can start a restore; inspect the preview and use the documented CLI only after the reviewed live capability is enabled.",
		}
		m.confirmRestore = false
		return m, nil
	}
	if strings.TrimSpace(m.cfg.Home) == "" {
		m.restoreOutcome = fleetWriteOutcome{Headline: "unavailable", Detail: "Fleet restore needs a resolved home; the TUI was started without one."}
		m.confirmRestore = false
		return m, nil
	}
	selected, ok := m.selectedBackup()
	if !ok {
		m.restoreOutcome = fleetWriteOutcome{Headline: "no backup selected", Detail: "Select a managed fleet backup first."}
		m.confirmRestore = false
		return m, nil
	}
	if !m.restorePreview.Available || m.restorePreview.Token == "" || m.restorePreview.BackupID != selected.ID {
		m.restoreOutcome = fleetWriteOutcome{Headline: "no fresh preview", Detail: "Preview this backup (Enter or v) before restoring it."}
		m.confirmRestore = false
		return m, nil
	}
	m.confirmRestore = true
	return m, nil
}

// confirmRestoreAndDispatch hands one confirmed managed-backup restore to the
// root shell. A second confirmation while a write is pending is inert.
func (m FleetModel) confirmRestoreAndDispatch() (FleetModel, tea.Cmd) {
	if !m.confirmRestore || m.writePending || !m.live() {
		m.confirmRestore = false
		return m, nil
	}
	preview := m.restorePreview
	if !preview.Available || preview.Token == "" || strings.TrimSpace(m.cfg.Home) == "" {
		m.confirmRestore = false
		return m, nil
	}
	m.confirmRestore = false
	m.writePending = true
	m.writeKind = WriteFleetRestore
	req := WriteRequest{
		Kind: WriteFleetRestore,
		Fleet: FleetWriteRequest{
			Home:       m.cfg.Home,
			ConfigPath: m.sourcePath,
			BackupID:   preview.BackupID,
			Token:      preview.Token,
			Label:      "fleet restore " + preview.BackupID,
			Source:     "fleet backup " + preview.BackupID,
		},
	}
	return m, startWriteCmd(req)
}

// recoveryLines renders managed backups, the selected restore preview and the
// drift/legacy/conflict guidance.
func (m FleetModel) recoveryLines(width int) []string {
	var lines []string

	header := "Install & Manage / Fleet Manager · Recovery"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)
	lines = append(lines, m.renderTabBar(FleetViewRecovery))

	sub := "Managed backups, exact restore effects and drift/legacy guidance (read-only until confirmed)"
	if m.theme.Color {
		sub = m.theme.Subtitle.Render(sub)
	}
	lines = append(lines, sub, "")

	// 1. Managed backup list.
	var listRows []string
	var listErr error
	if m.recoveryLoading {
		listRows = append(listRows, "Loading managed fleet backups...")
	} else if m.recoveryErr != "" {
		listErr = m.recoveryError
		if listErr == nil {
			listErr = errors.New(m.recoveryErr)
		}
		listRows = append(listRows, "The managed backup list could not be read.")
	} else if len(m.backups) == 0 {
		listRows = append(listRows, "No managed fleet backups were found for this home.")
		listRows = append(listRows, "A skills-only backup is not a fleet backup and stays in the existing Backups & restore screen.")
	} else {
		glyphs := GlyphSet(m.cfg.ASCII)
		start, end := fleetWindow(len(m.backups), m.backupIndex, 8)
		for i := start; i < end; i++ {
			backup := m.backups[i]
			marker := glyphs.Gutter
			if i == m.backupIndex {
				marker = glyphs.Cursor
			}
			row := fmt.Sprintf("%s %s · %s · %s · %d file(s)",
				marker, backup.ID, nonEmpty(backup.Operation, "fleet"), nonEmpty(backup.State, "unknown"), backup.Files)
			if i == m.backupIndex && m.theme.Color {
				row = m.theme.Selected.Render(row)
			}
			listRows = append(listRows, row)
		}
		listRows = append(listRows, fmt.Sprintf("Showing %d-%d of %d managed backup(s).", start+1, end, len(m.backups)))
	}
	lines = append(lines, Section("Managed backups (fleet)", listRows, width)...)
	if listErr != nil {
		lines = append(lines, m.guidanceLines(width, listErr)...)
	}
	lines = append(lines, "")

	// 2. Selected backup and exact restore preview.
	if backup, ok := m.selectedBackup(); ok {
		detail := []string{
			fmt.Sprintf("Backup identity: %s", backup.ID),
			fmt.Sprintf("Operation: %s · state: %s", nonEmpty(backup.Operation, "fleet"), nonEmpty(backup.State, "unknown")),
			fmt.Sprintf("Created: %s", nonEmpty(backup.CreatedAt, "unknown")),
			fmt.Sprintf("Recorded files: %d", backup.Files),
			"Ownership: only files recorded by this managed transaction are touched. Unrelated files are never deleted.",
		}
		lines = append(lines, Section("Selected backup", detail, width)...)
		lines = append(lines, "")
	}

	if m.restoreLoading {
		lines = append(lines, "  Computing the exact restore preview...", "")
	} else if m.restoreErr != "" {
		restoreErr := m.restoreError
		if restoreErr == nil {
			restoreErr = errors.New(m.restoreErr)
		}
		lines = append(lines, m.guidanceLines(width, restoreErr)...)
		lines = append(lines, "")
	} else if m.restorePreview.Available {
		lines = append(lines, m.restorePreviewLines(width)...)
	} else {
		message := m.restorePreview.Message
		if message == "" {
			message = "No restore preview yet."
		}
		lines = append(lines, "  "+message)
		lines = append(lines, "  Select a managed backup and press Enter to see exactly what a restore would change.")
		lines = append(lines, "")
	}

	// 3. Restore capability and default-No confirmation.
	capability := "Restore capability: Preview only — live restore is disabled in this build."
	if m.live() {
		capability = "Restore capability: live (reviewed build)."
	}
	if m.theme.Color && !m.live() {
		capability = m.theme.Warn.Render(capability)
	}
	lines = append(lines, "  "+capability)

	if m.confirmRestore {
		prompt := "Restore this managed backup? [y/N]"
		if m.theme.Color {
			prompt = m.theme.Warn.Bold(true).Render(prompt)
		}
		lines = append(lines, "")
		lines = append(lines, "  "+prompt)
		lines = append(lines, fmt.Sprintf("  This reverses the recorded transaction %s and creates a pre-restore backup.", m.restorePreview.BackupID))
		lines = append(lines, "  No Git operation, prompt/permission change or model call occurs.")
		lines = append(lines, "  Press y to confirm; n, Enter or Esc keeps the default No.")
	}
	lines = append(lines, "")

	if !m.restoreOutcome.Empty() {
		lines = append(lines, m.renderRestoreOutcome(width)...)
		lines = append(lines, "")
	}

	lines = append(lines, m.recoveryGuidanceLines(width)...)
	lines = append(lines, "")

	footer := "j/k: select backup · Enter/v: preview restore · t: restore (live only) · </>: scroll diff · r: refresh · p/Esc: profiles · o: overview · H: Home · ?: help"
	if m.theme.Color {
		footer = m.theme.Hint.Render(footer)
	}
	lines = append(lines, footer)
	return lines
}

// restorePreviewLines renders the exact counts, ownership limit and bounded
// per-file diff for one restore preview.
func (m FleetModel) restorePreviewLines(width int) []string {
	var lines []string
	preview := m.restorePreview
	rows := []string{
		fmt.Sprintf("Backup: %s", preview.BackupID),
		fmt.Sprintf("Files: %d total · %d create · %d update · %d remove · %d no-op",
			preview.Counts["total"], preview.Counts[fleet.FileOpCreate], preview.Counts[fleet.FileOpUpdate],
			preview.Counts[fleet.FileOpRemove], preview.Counts[fleet.FileOpNoOp]),
		"Conflicts: the engine refuses a stale or modified target instead of overwriting it.",
	}
	lines = append(lines, Section("Restore impact", rows, width)...)
	lines = append(lines, "")

	var fileRows []string
	for _, file := range preview.Files {
		row := fmt.Sprintf("%s · %s · %s", file.Op, file.InstanceID, file.Path)
		if file.Reason != "" {
			row += " · " + file.Reason
		}
		fileRows = append(fileRows, row)
	}
	if len(fileRows) == 0 {
		fileRows = append(fileRows, "(no files recorded)")
	}
	lines = append(lines, Section("Restore files", fileRows, width)...)
	lines = append(lines, "")

	maxRows := m.height - 28
	if maxRows < 6 {
		maxRows = 6
	}
	diffTitle := "Restore diff (per file)"
	if m.theme.Color {
		diffTitle = m.theme.Section.Render(diffTitle)
	}
	lines = append(lines, diffTitle, "")
	lines = append(lines, m.renderRecoveryDiff(preview.DiffLines, width, maxRows)...)
	return lines
}

func (m FleetModel) renderRestoreOutcome(width int) []string {
	rows := []string{
		"Outcome: " + m.restoreOutcome.Headline,
		"Detail: " + m.restoreOutcome.Detail,
	}
	if m.restoreOutcome.BackupID != "" {
		rows = append(rows, "Backup: "+m.restoreOutcome.BackupID)
	}
	if m.restoreOutcome.Rollback != "" {
		rows = append(rows, "Rollback: "+m.restoreOutcome.Rollback)
	}
	if m.restoreOutcome.LogPath != "" {
		rows = append(rows, "Operation log: "+m.restoreOutcome.LogPath)
	}
	title := "Last restore result"
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	return Section(title, rows, width)
}

// recoveryGuidanceLines reports only real drift/legacy/unmanaged/conflict
// facts. It never offers adoption, force or bulk repair.
func (m FleetModel) recoveryGuidanceLines(width int) []string {
	var conflicts []FleetInstanceView
	for _, inst := range m.status.Instances {
		if inst.DriftDetail != "" {
			conflicts = append(conflicts, inst)
		}
	}
	var rows []string
	if len(conflicts) == 0 && m.status.DriftedCount == 0 && m.status.UnmanagedCount == 0 {
		rows = append(rows, "No real conflicts, drift or legacy records were reported by the last status read.")
		rows = append(rows, "Runtime verification remains not checked; a managed/current file is not an authentication or entitlement claim.")
	} else {
		rows = append(rows, fmt.Sprintf("What happened: %d instance(s) report drift, unmanaged or legacy/unknown state.", len(conflicts)))
		rows = append(rows, "What stayed unchanged: this view modified no managed file and changed no ownership.")
		rows = append(rows, "Next safe action: review each instance, reload the status after an external change, or restore a completed managed backup. Use the documented CLI for an explicit ownership decision outside the TUI.")
		for _, inst := range conflicts {
			rows = append(rows, fmt.Sprintf("%s · %s · %s", inst.InstanceID, inst.InstalledState, sanitizeFleetReason(inst.DriftDetail)))
		}
	}
	title := "Recovery guidance"
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	return Section(title, rows, width)
}

// renderRecoveryDiff pages the bounded restore diff with its own scroll offset.
func (m FleetModel) renderRecoveryDiff(diffLines []string, width, maxRows int) []string {
	if len(diffLines) == 0 {
		return []string{"  (no diff content)"}
	}
	total := len(diffLines)
	scroll := m.recoveryScroll
	if scroll < 0 {
		scroll = 0
	}
	if scroll > total-1 {
		scroll = max(0, total-1)
	}
	end := scroll + maxRows
	if end > total {
		end = total
	}

	var rows []string
	if total > maxRows {
		info := fmt.Sprintf("Showing restore diff lines %d-%d of %d (j/k, PgUp/PgDn, Home/End):", scroll+1, end, total)
		if m.theme.Color {
			info = m.theme.Hint.Render(info)
		}
		rows = append(rows, info)
	}
	for i := scroll; i < end; i++ {
		line := diffLines[i]
		if width > 4 {
			line = truncateCells(line, width-4)
		}
		if m.theme.Color {
			switch {
			case strings.HasPrefix(line, "+"):
				line = m.theme.OK.Render(line)
			case strings.HasPrefix(line, "-"):
				line = m.theme.Bad.Render(line)
			case strings.HasPrefix(line, "==="), strings.HasPrefix(line, "…"):
				line = m.theme.Accent.Render(line)
			}
		}
		rows = append(rows, "  "+line)
	}
	return rows
}

// fleetWindow returns a bounded [start,end) window that keeps index visible.
func fleetWindow(total, index, maxRows int) (int, int) {
	if maxRows < 1 {
		maxRows = 1
	}
	if index < 0 {
		index = 0
	}
	if index >= total {
		index = total - 1
	}
	start := index - maxRows + 1
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > total {
		end = total
	}
	return start, end
}

// truncateCells keeps width in terminal cells using the same helper family as
// the rest of the TUI.
func truncateCells(text string, width int) string {
	return ansi.Truncate(text, width, "")
}
