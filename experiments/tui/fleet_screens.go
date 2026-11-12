package tui

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"
)

// Lines renders the active Fleet Manager sub-view for the given width.
func (m FleetModel) Lines(width int) []string {
	if width < 60 || m.height < 8 {
		warning := "Terminal too small for Fleet Manager. Resize to at least 60x8."
		if m.theme.Color {
			warning = m.theme.Warn.Render(warning)
		}
		return []string{warning, "Press Esc to return."}
	}

	if m.confirmDiscard {
		return m.discardPromptLines(width)
	}

	// The diagnostics panel is contextual: it replaces the current view until
	// Esc returns, so it is never a fifth persistent tab.
	if m.doctorOpen {
		return m.doctorLines(width)
	}

	switch m.view {
	case FleetViewProfiles:
		return m.profilesLines(width)
	case FleetViewReview:
		return m.deploymentReviewLines(width)
	case FleetViewRecovery:
		return m.recoveryLines(width)
	default:
		return m.overviewLines(width)
	}
}

// ---------------------------------------------------------------------------
// Overview view
// ---------------------------------------------------------------------------

func (m FleetModel) overviewLines(width int) []string {
	var lines []string

	// Title and header
	header := "Install & Manage / Fleet Manager · Overview"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)

	// Tab switcher
	tabs := m.renderTabBar(FleetViewOverview)
	lines = append(lines, tabs)

	sub := "Agent fleet status and model overrides (read-only inspection)"
	if m.theme.Color {
		sub = m.theme.Subtitle.Render(sub)
	}
	lines = append(lines, sub, "")

	if m.loading {
		lines = append(lines, "  Loading fleet status...", "")
	} else if m.loadErr != "" {
		errText := "  Status read failed: " + m.loadErr
		if m.theme.Color {
			errText = m.theme.Bad.Render(errText)
		}
		lines = append(lines, errText, "")
	}

	// 1. Personal configuration section
	var sourceRows []string
	if m.editingSource {
		input := m.sourceInput + "_"
		if m.theme.Color {
			input = m.theme.Selected.Render(input)
		}
		sourceRows = append(sourceRows, "Source path: "+input)
		sourceRows = append(sourceRows, "  (Enter to load · Esc to cancel)")
	} else {
		src := m.sourcePath
		if src == "" {
			src = m.status.SourcePath
		}
		if src == "" {
			src = "(none selected — press s to specify explicit path)"
		}
		sourceRows = append(sourceRows, "Source path: "+src)
		state := m.status.SourceState
		if state == "" {
			state = "not selected"
		}
		sourceRows = append(sourceRows, "Source state: "+state)
		if m.status.PersonalConfigSummary != "" {
			sourceRows = append(sourceRows, "Summary: "+m.status.PersonalConfigSummary)
		}
	}
	lines = append(lines, Section("Personal configuration (fleet.json)", sourceRows, width)...)
	lines = append(lines, "")

	// 2. Installed OpenCode agents section
	installedSummary := m.status.InstalledSummary
	if installedSummary == "" {
		installedSummary = "No managed instances inspected"
	}
	var installedRows []string
	installedRows = append(installedRows, "Managed location: ~/.config/opencode/agents")
	installedRows = append(installedRows, "Status: "+installedSummary)
	lines = append(lines, Section("Installed OpenCode agents", installedRows, width)...)
	lines = append(lines, "")

	// 3. Inventory counts section
	var countRows []string
	countRows = append(countRows, fmt.Sprintf("Total configured instances: %d · Managed: %d",
		m.status.TotalInstances, m.status.ManagedCount))
	countRows = append(countRows, fmt.Sprintf("Enabled: %d · Disabled: %d",
		m.status.EnabledCount, m.status.DisabledCount))
	countRows = append(countRows, fmt.Sprintf("Drifted: %d · Unmanaged: %d",
		m.status.DriftedCount, m.status.UnmanagedCount))
	lines = append(lines, Section("Inventory counts", countRows, width)...)
	lines = append(lines, "")

	// 4. Status & recommendations section
	var recRows []string
	outcome := m.status.LastOutcome
	if outcome == "" {
		outcome = "None"
	}
	recRows = append(recRows, "Last operation outcome: "+outcome)
	next := m.status.NextAction
	if next == "" {
		next = "Inspect agent profiles (press p or Enter) or set an explicit fleet.json source (press s)"
	}
	recRows = append(recRows, "Recommended action: "+next)
	recRows = append(recRows, "Runtime verification: not checked (offline default, no models invoked)")
	capability := "Preview only — live deployment and restore are disabled in this build"
	if m.live() {
		capability = "live (reviewed build) — deployment and restore confirmations are available"
	}
	recRows = append(recRows, "Deployment capability: "+capability)
	lines = append(lines, Section("Status & recommendations", recRows, width)...)
	lines = append(lines, "")

	// Footer / controls hint
	hint := "Controls: p/Enter: inspect profiles · s: edit source · d: diagnostics · r: reload · H: Home · Esc: hub · ?: help"
	if m.theme.Color {
		hint = m.theme.Hint.Render(hint)
	}
	lines = append(lines, hint)

	return lines
}

// ---------------------------------------------------------------------------
// Profiles view
// ---------------------------------------------------------------------------

func (m FleetModel) profilesLines(width int) []string {
	var lines []string

	// Title and header
	header := "Install & Manage / Fleet Manager · Profiles"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)

	// Tab switcher
	tabs := m.renderTabBar(FleetViewProfiles)
	lines = append(lines, tabs)

	sub := "Browse and search configured agent instances"
	if m.dirtyCount > 0 {
		sub = fmt.Sprintf("Browse and search configured agent instances · %d unsaved edit(s)", m.dirtyCount)
	}
	if m.theme.Color {
		sub = m.theme.Subtitle.Render(sub)
	}
	lines = append(lines, sub)

	if m.disableWarning != "" {
		warn := "  " + m.disableWarning
		if m.theme.Color {
			warn = m.theme.Warn.Render(warn)
		}
		lines = append(lines, warn)
	}

	// Search filter bar
	filterBar := m.renderFilterBar(width)
	lines = append(lines, filterBar, "")

	filtered := m.FilteredInstances()
	if len(filtered) == 0 {
		empty := "  No matching instances found."
		if m.status.TotalInstances == 0 {
			empty = "  No instances available. Select an explicit source file in Overview (press o)."
		}
		if m.theme.Color {
			empty = m.theme.Hint.Render(empty)
		}
		lines = append(lines, empty, "")
		lines = append(lines, m.renderProfilesFooter())
		return lines
	}

	selected, hasSelected := m.SelectedInstance()

	// Layout selection by width
	if m.isWide(width) {
		// Side-by-side: list on left, detail on right
		bodyHeight := m.height - 8
		if bodyHeight < 6 {
			bodyHeight = 6
		}
		leftWidth := 46
		rightWidth := width - leftWidth - 3
		if rightWidth < 30 {
			rightWidth = 30
		}

		leftLines := m.renderInstanceList(filtered, leftWidth, bodyHeight)
		var rightLines []string
		if m.confirmDisable && hasSelected {
			rightLines = m.renderDisableConsequenceCard(selected, rightWidth)
		} else if m.editingModel && hasSelected {
			rightLines = m.renderModelPickerCard(selected, rightWidth)
		} else if hasSelected {
			rightLines = m.renderDetailCard(selected, rightWidth)
			if m.detailScroll > 0 {
				if m.detailScroll < len(rightLines) {
					rightLines = rightLines[m.detailScroll:]
				} else if len(rightLines) > 0 {
					rightLines = rightLines[len(rightLines)-1:]
				}
			}
		} else {
			rightLines = []string{"Select an instance to view details."}
		}

		glyphs := GlyphSet(m.cfg.ASCII)
		maxRows := max(len(leftLines), len(rightLines))
		for r := 0; r < maxRows; r++ {
			left := ""
			if r < len(leftLines) {
				left = leftLines[r]
			}
			right := ""
			if r < len(rightLines) {
				right = rightLines[r]
			}
			line := padCells(left, leftWidth) + " " + glyphs.DividerV + " " + right
			lines = append(lines, line)
		}
	} else if m.confirmDisable && hasSelected {
		// Drill-down disable consequence view (compact widths)
		lines = append(lines, m.renderDisableConsequenceCard(selected, width)...)
	} else if m.editingModel && hasSelected {
		// Drill-down model picker view (compact widths)
		lines = append(lines, m.renderModelPickerCard(selected, width)...)
	} else if m.detailOpen && hasSelected {
		// Drill-down detail view (compact widths)
		backPrompt := "← Press Esc or Enter to return to instance list"
		if m.theme.Color {
			backPrompt = m.theme.Accent.Render(backPrompt)
		}
		lines = append(lines, backPrompt, "")
		lines = append(lines, m.renderDetailCard(selected, width)...)
	} else {
		// Single-pane list view
		bodyHeight := m.height - 8
		if bodyHeight < 6 {
			bodyHeight = 6
		}
		listLines := m.renderInstanceList(filtered, width, bodyHeight)
		lines = append(lines, listLines...)
	}

	lines = append(lines, "", m.renderProfilesFooter())
	return lines
}

func (m FleetModel) renderTabBar(active FleetView) string {
	ov := "o Overview"
	pr := "p Profiles"
	rv := "c Review Changes"
	rec := "e Recovery"

	formatTab := func(label string, view FleetView) string {
		if active == view {
			label = "[" + label + "]"
			if m.theme.Color {
				return m.theme.Selected.Bold(true).Render(label)
			}
			return label
		}
		return " " + label + " "
	}

	return "  " + formatTab(ov, FleetViewOverview) + "    " +
		formatTab(pr, FleetViewProfiles) + "    " +
		formatTab(rv, FleetViewReview) + "    " +
		formatTab(rec, FleetViewRecovery)
}

func (m FleetModel) renderFilterBar(width int) string {
	if m.isFiltering {
		input := "Filter: " + m.filter + "_"
		if m.theme.Color {
			return "  " + m.theme.Selected.Render(input) + "  " + m.theme.Hint.Render("(Enter: apply · Esc: cancel)")
		}
		return "  " + input + "  (Enter: apply · Esc: cancel)"
	}
	if m.filter != "" {
		count := len(m.FilteredInstances())
		txt := fmt.Sprintf("Filter: %q (%d matching) · press / to change, Esc to clear", m.filter, count)
		if m.theme.Color {
			return "  " + m.theme.Accent.Render(txt)
		}
		return "  " + txt
	}
	txt := "Filter: (none) · press / to search instances"
	if m.theme.Color {
		return "  " + m.theme.Hint.Render(txt)
	}
	return "  " + txt
}

func (m FleetModel) renderInstanceList(instances []FleetInstanceView, width, maxRows int) []string {
	var rows []string
	glyphs := GlyphSet(m.cfg.ASCII)

	// Keep selected row visible
	scroll := m.scrollOffset
	if m.selectedIndex < scroll {
		scroll = m.selectedIndex
	}
	if m.selectedIndex >= scroll+maxRows {
		scroll = m.selectedIndex - maxRows + 1
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + maxRows
	if end > len(instances) {
		end = len(instances)
	}

	for i := scroll; i < end; i++ {
		inst := instances[i]
		marker := glyphs.Gutter
		if i == m.selectedIndex {
			marker = glyphs.Cursor
		}

		enBadge := "[+]"
		if !inst.Enabled {
			enBadge = "[-]"
		}

		compactModel := inst.CompactModel
		if compactModel == "" {
			compactModel = inst.Model
		}

		stateBadge := inst.InstalledState
		if stateBadge == "" {
			stateBadge = "not installed"
		}

		// Stage reflection in list row (FR-04, FR-07, AC-08, AC-10)
		if edit, hasEdit := m.draft[inst.InstanceID]; hasEdit {
			if !edit.Enabled {
				enBadge = "[-]*"
			} else {
				enBadge = "[+]*"
			}
			if edit.Model != "" && edit.Model != inst.Model {
				oldC := inst.CompactModel
				if oldC == "" {
					oldC = CompactModelName(inst.Model)
				}
				newC := CompactModelName(edit.Model)
				compactModel = fmt.Sprintf("%s->%s*", oldC, newC)
			} else {
				compactModel = compactModel + "*"
			}
			stateBadge = stateBadge + " [staged]"
		}

		drift := ""
		if inst.ChangedCount > 0 {
			drift = fmt.Sprintf(" (+%d)", inst.ChangedCount)
		}

		row := fmt.Sprintf("%s %s %-16s %-16s %s",
			marker, enBadge, inst.InstanceID, compactModel, stateBadge+drift)

		if i == m.selectedIndex {
			if m.theme.Color {
				row = m.theme.Selected.Render(ansi.Truncate(row, width, ""))
			} else {
				row = ansi.Truncate(row, width, "")
			}
		} else {
			row = ansi.Truncate(row, width, "")
		}
		rows = append(rows, row)
	}

	return rows
}

func (m FleetModel) renderDetailCard(inst FleetInstanceView, width int) []string {
	var body []string

	body = append(body, fmt.Sprintf("Instance ID:       %s", inst.InstanceID))
	roleDesc := inst.Role
	if roleDesc != "" && roleDesc != inst.InstanceID {
		roleDesc = fmt.Sprintf("%s (canonical policy role)", inst.Role)
	} else if roleDesc == "" {
		roleDesc = "(same as instance)"
	}
	body = append(body, fmt.Sprintf("Canonical Role:    %s", roleDesc))
	body = append(body, "")

	route := inst.ProviderRoute
	if route == "" {
		route = "unspecified"
	}
	body = append(body, fmt.Sprintf("Provider Route:    %s", route))
	fullModel := inst.Model
	if fullModel == "" {
		fullModel = "(default)"
	}
	if inst.Variant != "" {
		fullModel += " (variant: " + inst.Variant + ")"
	}
	body = append(body, fmt.Sprintf("Full Model:        %s", fullModel))
	if inst.CompactModel != "" && inst.CompactModel != inst.Model {
		body = append(body, fmt.Sprintf("Compact Model:     %s", inst.CompactModel))
	}
	body = append(body, "")

	prov := inst.SourceProvenance
	if prov == "" {
		prov = "standard pack"
	}
	body = append(body, fmt.Sprintf("Source Provenance: %s", prov))

	perm := inst.PermissionSummary
	if perm == "" {
		perm = "engine defaults (no overrides)"
	}
	body = append(body, fmt.Sprintf("Engine Permissions: %s", perm))
	body = append(body, "")

	enStr := "true"
	if !inst.Enabled {
		enStr = "false (disabled)"
	}
	body = append(body, fmt.Sprintf("Enabled:           %s", enStr))
	state := inst.InstalledState
	if state == "" {
		state = "not installed"
	}
	body = append(body, fmt.Sprintf("Installed State:   %s", state))
	body = append(body, fmt.Sprintf("Change/Drift Count: %d differences", inst.ChangedCount))
	if inst.DriftDetail != "" {
		body = append(body, fmt.Sprintf("Drift Details:     %s", inst.DriftDetail))
	}
	body = append(body, "")

	// Staged in-memory edits display (FR-07, AC-10, AC-14)
	if edit, hasEdit := m.draft[inst.InstanceID]; hasEdit {
		body = append(body, "--- Staged Edits (in-memory draft) ---")
		if edit.Model != "" && edit.Model != inst.Model {
			body = append(body, fmt.Sprintf("Staged Model:      %s -> %s (unsaved)", inst.Model, edit.Model))
		}
		if edit.Enabled != inst.Enabled {
			enState := "enabled"
			if !edit.Enabled {
				enState = "disabled (staged for managed removal upon apply)"
			}
			body = append(body, fmt.Sprintf("Staged State:      %s (unsaved)", enState))
		}
		body = append(body, "Draft Persistence: in-memory only (NOT saved to disk)")
		body = append(body, "")
	}

	body = append(body, "Runtime Status:    unverified (not checked — requires OpenCode execution)")

	title := "Profile details: " + inst.InstanceID
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	return Section(title, body, width)
}

func (m FleetModel) renderModelPickerCard(inst FleetInstanceView, width int) []string {
	var body []string

	body = append(body, fmt.Sprintf("Target Instance:   %s", inst.InstanceID))
	currentModel := inst.Model
	if edit, has := m.draft[inst.InstanceID]; has && edit.Model != "" {
		currentModel = fmt.Sprintf("%s (staged: %s)", inst.Model, edit.Model)
	}
	body = append(body, fmt.Sprintf("Current Model:     %s", currentModel))
	body = append(body, "")

	inputLine := "Model Input:       " + m.modelInput + "_"
	if m.theme.Color {
		inputLine = m.theme.Selected.Render(inputLine)
	}
	body = append(body, inputLine)
	body = append(body, "                   (type provider/model[#variant] or select suggestion below)")

	if m.pickerErr != "" {
		errLine := "Validation Error:  " + m.pickerErr
		if m.theme.Color {
			errLine = m.theme.Bad.Render(errLine)
		}
		body = append(body, "", errLine)
	}

	body = append(body, "", "Available Suggestions:")
	suggestions := m.pickerSuggestions()
	glyphs := GlyphSet(m.cfg.ASCII)
	maxSuggestions := 6
	for i, s := range suggestions {
		if i >= maxSuggestions {
			remaining := len(suggestions) - maxSuggestions
			body = append(body, fmt.Sprintf("    ... and %d more (type to narrow)", remaining))
			break
		}
		marker := glyphs.Gutter
		if i == m.pickerIndex {
			marker = glyphs.Cursor
		}
		row := fmt.Sprintf("  %s %s", marker, s)
		if i == m.pickerIndex {
			if m.theme.Color {
				row = m.theme.Selected.Render(row)
			}
		}
		body = append(body, row)
	}

	body = append(body, "")
	body = append(body, "Controls: Enter: stage model · Esc: cancel · Down/Up: select suggestion · Tab: use suggestion")

	title := "Model Picker · " + inst.InstanceID
	if m.theme.Color {
		title = m.theme.Section.Render(title)
	}
	return Section(title, body, width)
}

func (m FleetModel) renderDisableConsequenceCard(inst FleetInstanceView, width int) []string {
	var body []string

	body = append(body, fmt.Sprintf("Target Instance: %s", inst.InstanceID))
	body = append(body, fmt.Sprintf("Role:            %s", inst.Role))
	body = append(body, fmt.Sprintf("Installed State: %s", inst.InstalledState))
	body = append(body, "")

	consequence := []string{
		"CONSEQUENCE NOTICE (FR-05 / AC-11):",
		"Disabling this managed instance will stage a managed removal of the installed",
		"OpenCode agent file upon apply. A backup of the agent file will be preserved",
		"in the backup root before removal.",
		"",
		"Unmanaged files cannot be deleted by fleet enablement toggle.",
		"",
		"Stage disable in-memory draft? [y/N]",
	}
	body = append(body, consequence...)

	title := "Confirm Disable · " + inst.InstanceID
	if m.theme.Color {
		title = m.theme.Warn.Render(title)
	}
	return Section(title, body, width)
}

func (m FleetModel) renderProfilesFooter() string {
	if m.editingModel {
		hint := "Enter: stage model · Esc: cancel · Down/Up: select suggestion · Tab: use suggestion"
		if m.theme.Color {
			return m.theme.Hint.Render(hint)
		}
		return hint
	}
	if m.confirmDisable {
		hint := "y/Enter: confirm disable · n/Esc: cancel"
		if m.theme.Color {
			return m.theme.Hint.Render(hint)
		}
		return hint
	}
	var hint string
	if m.isWide(m.width) {
		if m.focusDetail {
			hint = "j/k: scroll detail · l: focus list · /: filter · m: model picker · t: toggle · c: review · o: overview · Esc: back · ?: help"
		} else {
			hint = "j/k: navigate · l: focus detail · /: filter · m: model picker · t: toggle · c: review · o: overview · Esc: back · ?: help"
		}
	} else {
		hint = "j/k: navigate · /: filter · m: model picker · t: toggle · Enter: details · c: review · o: overview · Esc: back · ?: help"
	}
	if m.theme.Color {
		return m.theme.Hint.Render(hint)
	}
	return hint
}
