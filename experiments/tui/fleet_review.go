package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// FleetSourcePreview carries the read-only projection of staged personal source edits.
// It reflects the compare-and-swap base digest, the proposed new bytes/digest, and
// line diffs for user review. No mutation is performed by this projection.
type FleetSourcePreview struct {
	Available  bool
	SourcePath string
	OldBytes   []byte
	NewBytes   []byte
	OldLines   []string
	NewLines   []string
	OldDigest  string
	NewDigest  string
	BaseDigest string
	Op         string
	DiffLines  []string
	Message    string
}

// FleetSourcePreviewMsg delivers an asynchronous source preview result.
type FleetSourcePreviewMsg struct {
	Generation uint64
	Source     string
	Preview    FleetSourcePreview
	Err        error
}

// reviewLines renders the Review Changes view for the staged personal source edits.
// It leads with an impact summary, displays exact scrollable diff lines, and explicitly
// marks the state as not saved.
func (m FleetModel) reviewLines(width int) []string {
	var lines []string

	// Header & navigation breadcrumb
	header := "Install & Manage / Fleet Manager · Review Changes"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)

	// Tab bar
	tabs := m.renderTabBar(FleetViewReview)
	lines = append(lines, tabs)

	sub := "Review staged personal source modifications before saving (read-only review)"
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

	// 1. Impact summary first (FR-08 / AC-19)
	var impactRows []string
	src := m.sourcePath
	if src == "" {
		src = m.status.SourcePath
	}
	if src == "" {
		src = "(none selected)"
	}
	impactRows = append(impactRows, fmt.Sprintf("Staged in-memory edits: %d edit(s)", m.dirtyCount))
	impactRows = append(impactRows, fmt.Sprintf("Personal source file:   %s", src))

	baseDig := m.baseDigest
	if baseDig == "" {
		baseDig = m.sourcePreview.BaseDigest
	}
	if baseDig == "" {
		baseDig = "(base digest pending preview)"
	}
	impactRows = append(impactRows, fmt.Sprintf("Base digest (CAS):      %s", baseDig))
	impactRows = append(impactRows, "Installed agent files:  Unchanged (0 OpenCode files modified)")
	impactRows = append(impactRows, "Save status:            NOT SAVED (in-memory draft only; B4 source save not authorized)")

	lines = append(lines, Section("Impact summary", impactRows, width)...)
	lines = append(lines, "")

	// 2. Exact source diff section
	var diffLines []string
	if m.dirtyCount == 0 {
		diffLines = append(diffLines, "  No changes staged. Select an instance in Profiles (press p) to stage a model or enablement change.")
	} else if len(m.sourcePreview.DiffLines) > 0 {
		diffLines = m.sourcePreview.DiffLines
	} else if m.sourcePreview.Message != "" {
		diffLines = append(diffLines, "  "+m.sourcePreview.Message)
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
	renderedDiff := m.renderReviewDiff(diffLines, width, maxDiffRows)
	lines = append(lines, renderedDiff...)
	lines = append(lines, "")

	// 3. Operational boundary note
	notice := "Note: Staged edits remain in memory only. No save button is offered (B4 not authorized). Press p to return to Profiles."
	if m.theme.Color {
		notice = m.theme.Hint.Render(notice)
	}
	lines = append(lines, notice, "")

	// Footer controls
	footer := "j/k: scroll diff · p/Esc: back to profiles · o: overview · H: Home · ?: help"
	if m.theme.Color {
		footer = m.theme.Hint.Render(footer)
	}
	lines = append(lines, footer)

	return lines
}

// renderReviewDiff formats and pages diff lines with scrolling.
func (m FleetModel) renderReviewDiff(diffLines []string, width, maxRows int) []string {
	if len(diffLines) == 0 {
		return []string{"  (no diff content)"}
	}

	total := len(diffLines)
	scroll := m.reviewScroll
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
		pageInfo := fmt.Sprintf("Showing diff lines %d-%d of %d (scroll with j/k, PgUp/PgDn, Home/End):", scroll+1, end, total)
		if m.theme.Color {
			pageInfo = m.theme.Hint.Render(pageInfo)
		}
		rows = append(rows, pageInfo)
	}

	for i := scroll; i < end; i++ {
		line := diffLines[i]
		line = ansi.Truncate(line, width-4, "")

		if m.theme.Color {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				line = m.theme.OK.Render(line)
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				line = m.theme.Bad.Render(line)
			} else if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
				line = m.theme.Accent.Render(line)
			}
		}
		rows = append(rows, "  "+line)
	}

	return rows
}

// GenerateFallbackDiffLines produces a synthetic diff from staged edits when an external diff is not yet ready.
func GenerateFallbackDiffLines(edits []FleetModelEdit, sourcePath string) []string {
	if len(edits) == 0 {
		return nil
	}
	src := sourcePath
	if src == "" {
		src = "fleet.json"
	}
	var rows []string
	rows = append(rows, fmt.Sprintf("--- a/%s", src))
	rows = append(rows, fmt.Sprintf("+++ b/%s", src))
	rows = append(rows, "@@ in-memory staged draft @@")
	for _, edit := range edits {
		rows = append(rows, fmt.Sprintf("  instance %s:", edit.InstanceID))
		rows = append(rows, fmt.Sprintf("+   model: %s", edit.Model))
		enStr := "true"
		if !edit.Enabled {
			enStr = "false"
		}
		rows = append(rows, fmt.Sprintf("+   enabled: %s", enStr))
	}
	return rows
}

// fallbackDiffLines produces a synthetic diff from staged edits when an external diff is not yet ready.
func (m FleetModel) fallbackDiffLines() []string {
	return GenerateFallbackDiffLines(m.Edits(), m.sourcePath)
}

// discardPromptLines renders the modal prompt for protecting unsaved dirty edits on Esc.
func (m FleetModel) discardPromptLines(width int) []string {
	var lines []string

	header := "Install & Manage / Fleet Manager · Unsaved Edits"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header, "")

	var body []string
	body = append(body, fmt.Sprintf("You have %d unsaved edit(s) staged in memory.", m.dirtyCount))
	body = append(body, "Leaving Fleet Manager will discard these in-memory modifications.")
	body = append(body, "")
	body = append(body, "Actions:")
	body = append(body, "  [k] Keep editing   — stay on current screen with edits intact")
	body = append(body, "  [d] Discard edits  — drop all staged edits and clear draft")
	body = append(body, "  [s / Esc] Stay     — cancel prompt and stay on screen")

	lines = append(lines, Section("Discard staged edits?", body, width)...)
	lines = append(lines, "")

	hint := "Press k to keep editing, d to discard edits, or s/Esc to stay."
	if m.theme.Color {
		hint = m.theme.Hint.Render(hint)
	}
	lines = append(lines, hint)

	return lines
}

// recoveryPlaceholderLines renders the recovery placeholder view until W2-C lands.
func (m FleetModel) recoveryPlaceholderLines(width int) []string {
	var lines []string

	header := "Install & Manage / Fleet Manager · Recovery"
	if m.theme.Color {
		header = m.theme.Title.Render(header)
	}
	lines = append(lines, header)

	tabs := m.renderTabBar(FleetViewRecovery)
	lines = append(lines, tabs)

	body := []string{
		"Fleet recovery, drift resolution, and restore preview.",
		"This screen is scheduled for Wave 2 lane W2-C.",
		"",
		"Press p to inspect profiles or o for overview.",
	}
	lines = append(lines, Section("Recovery status", body, width)...)
	lines = append(lines, "")

	footer := "p: profiles · o: overview · H: Home · ?: help"
	if m.theme.Color {
		footer = m.theme.Hint.Render(footer)
	}
	lines = append(lines, footer)
	return lines
}
