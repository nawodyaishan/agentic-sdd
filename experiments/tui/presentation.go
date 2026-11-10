package tui

import (
	"fmt"
	"sort"
	"strings"

	"agentic-sdd/internal/skillsync"
	"github.com/charmbracelet/x/ansi"
)

// Paragraph wraps one source line of prose on word boundaries and prefixes
// every produced line with indent spaces. It never joins separate source lines,
// never splits a word that fits, and only hard-wraps a token wider than the
// available cells (a path or command) using cell-aware grapheme-safe wrapping.
// Surrounding whitespace in text is not content: only the indent introduces
// leading spaces, and no line carries a trailing space. No produced line is
// wider than width — narrow terminals hard-wrap instead of emitting
// overlong rows the viewport would have to truncate — and a non-positive
// width means wrapping is impossible, so the source lines are returned
// unwrapped and nothing is dropped.
func Paragraph(text string, width, indent int) []string {
	source := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if width <= 0 {
		return source
	}
	if indent < 0 {
		indent = 0
	}
	// Clamp the indent so a tiny width cannot push content off screen; at
	// least one content cell always remains.
	if indent > width-1 {
		indent = width - 1
	}
	lines := make([]string, 0, len(source))
	for _, line := range source {
		lines = append(lines, wrapParagraphLine(line, width, indent)...)
	}
	return lines
}

func wrapParagraphLine(text string, width, indent int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	available := width - indent // >= 1: Paragraph clamps the indent
	prefix := strings.Repeat(" ", indent)

	var lines []string
	current := ""
	flush := func() {
		if current != "" {
			lines = append(lines, prefix+current)
			current = ""
		}
	}
	for _, word := range strings.Fields(text) {
		if ansi.StringWidth(word) > available {
			flush()
			for _, segment := range strings.Split(ansi.Hardwrap(word, available, true), "\n") {
				if segment != "" {
					lines = append(lines, prefix+segment)
				}
			}
			continue
		}
		switch {
		case current == "":
			current = word
		case ansi.StringWidth(current)+1+ansi.StringWidth(word) <= available:
			current += " " + word
		default:
			flush()
			current = word
		}
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// Section renders a title, exactly one blank line and an indented body. Body
// lines are re-wrapped at width with the same 2-space indent the rest of the
// interface uses. Leading blank lines are dropped, empty lines are kept only
// as single paragraph separators, and trailing blanks are removed, so a section
// never shows doubled blank lines or doubled indentation.
func Section(title string, body []string, width int) []string {
	var lines []string
	if strings.TrimSpace(title) != "" {
		lines = Paragraph(title, width, 0)
	}

	normalized := make([]string, 0, len(body))
	for _, line := range body {
		if strings.TrimSpace(ansi.Strip(line)) == "" {
			if len(normalized) > 0 && normalized[len(normalized)-1] != "" {
				normalized = append(normalized, "")
			}
			continue
		}
		normalized = append(normalized, line)
	}
	for len(normalized) > 0 && normalized[len(normalized)-1] == "" {
		normalized = normalized[:len(normalized)-1]
	}
	if len(normalized) == 0 {
		return lines
	}

	if len(lines) > 0 {
		lines = append(lines, "")
	}
	for _, line := range normalized {
		if line == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, Paragraph(line, width, 2)...)
	}
	return lines
}

// HomeOverview summarizes the read-only local inventory for the Home screen.
// Every fact comes from the catalog snapshot: item kinds and origins, the
// per-client statuses, the kind counts and the notices. It never queries the
// system, performs a change or claims a runtime, provider or authentication
// state that the snapshot does not describe. Per-client coverage is scoped to
// the core skill set, the supported installation unit; optional pack and
// profile content is reported as its own separate totals. All long Home, Repo,
// backup and notice strings are wrapped, never truncated, so the parent can
// scroll them back into view.
func HomeOverview(catalog Catalog, width int) []string {
	lines := Section("Local inventory", inventoryLines(catalog), width)

	lines = append(lines, "")
	lines = append(lines, Section("Client status (core skills)", clientStatusLines(catalog), width)...)

	lines = append(lines, "")
	lines = append(lines, Section("Optional content", []string{
		fmt.Sprintf("OpenCode pack: %s", optionalPresence(catalog, skillsync.InventoryOriginPack)),
		fmt.Sprintf("Free-first presets: %s", optionalPresence(catalog, skillsync.InventoryOriginPreset)),
	}, width)...)

	if len(catalog.Notices) > 0 {
		lines = append(lines, "")
		lines = append(lines, Section("Notices", noticeLines(catalog.Notices), width)...)
	}

	lines = append(lines, "")
	lines = append(lines, Section("Recommended next step", []string{
		"Set up / Install Agentic-SDD",
		"Read-only local inspection: this screen only reads the local inventory and never installs, applies or changes anything.",
	}, width)...)
	return lines
}

// inventoryLines is the body of the local-inventory section: derived counts,
// the complete directory strings and, for an empty inventory, an explicit
// recovery explanation instead of fabricated coverage.
func inventoryLines(catalog Catalog) []string {
	core, other := 0, 0
	for _, item := range catalog.Items {
		switch {
		case item.Kind == skillsync.InventoryKindSkill && item.Origin == skillsync.InventoryOriginCore:
			core++
		case item.Kind == skillsync.InventoryKindSkill && item.Origin != skillsync.InventoryOriginPack:
			// A skill whose origin is unknown is counted neutrally, never as
			// core content the snapshot did not report.
			other++
		}
	}
	lines := []string{
		fmt.Sprintf("Skills: %d", catalog.SkillCount()),
		fmt.Sprintf("Profiles: %d", catalog.ProfileCount()),
		fmt.Sprintf("Core skills: %d", core),
	}
	if other > 0 {
		lines = append(lines, fmt.Sprintf("Other skills: %d", other))
	}
	if catalog.Home != "" {
		lines = append(lines, "Home directory: "+catalog.Home)
	}
	if catalog.Repo != "" {
		lines = append(lines, "Repository source: "+catalog.Repo)
	}
	if catalog.BackupRoot != "" {
		lines = append(lines, "Backups directory: "+catalog.BackupRoot)
	}
	if len(catalog.Items) == 0 {
		lines = append(lines,
			"No managed skills or profiles found in this home yet.",
			"Recovery: verify --home and --repo, run agentic-sdd preview to inspect the selected source, then reopen this screen. This read-only view changes no files.",
		)
	}
	return lines
}

// noticeLines explains how to recover from an inventory notice instead of
// only repeating it: the core inventory still loads, and the reported source
// selection is the thing to fix.
func noticeLines(notices []string) []string {
	lines := make([]string, 0, len(notices)+1)
	lines = append(lines, notices...)
	return append(lines,
		"Recovery: check the reported source selection (for example restore the optional pack source or adjust --repo and --home) and reopen this screen; the core inventory still loads.")
}

type clientCounts struct {
	total    int
	present  int // matches source + differs from source
	matching int
	modified int
	missing  int
	refused  int
}

// clientStatusLines reports per-client coverage of the core skill set. Every
// row shows a present/total fraction, names the modified count explicitly and
// separates missing from refused destinations. It never summarizes a client as
// fully installed and never claims a runtime, authentication or approval
// state: these are file-presence comparisons only.
func clientStatusLines(catalog Catalog) []string {
	var coreItems []CatalogItem
	for _, item := range catalog.Items {
		if item.Kind == skillsync.InventoryKindSkill && item.Origin == skillsync.InventoryOriginCore {
			coreItems = append(coreItems, item)
		}
	}
	if len(coreItems) == 0 {
		if len(catalog.Items) == 0 {
			return []string{"No client destinations reported for this selection."}
		}
		return []string{"No core skills in this selection."}
	}

	counts := make(map[string]*clientCounts)
	for _, item := range coreItems {
		for _, client := range item.Clients {
			c, ok := counts[client.Client]
			if !ok {
				c = &clientCounts{}
				counts[client.Client] = c
			}
			c.total++
			switch client.Status {
			case skillsync.InventoryStatusMatchesSource:
				c.matching++
				c.present++
			case skillsync.InventoryStatusDiffersFromSource:
				c.modified++
				c.present++
			case skillsync.InventoryStatusUnavailable:
				c.refused++
			default:
				// Missing and any status the snapshot does not recognize count
				// as not present, so the row still adds up to its total.
				c.missing++
			}
		}
	}
	if len(counts) == 0 {
		return []string{"No client destinations reported for core skills."}
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		c := counts[name]
		lines = append(lines, fmt.Sprintf(
			"%s: present %d/%d (%d matching + %d modified) · %d not present · %d refused",
			name, c.present, c.total, c.matching, c.modified, c.missing, c.refused,
		))
	}
	return lines
}

// optionalPresence reports whether optional source content is part of the
// loaded selection. "Present in this selection" describes the source inventory
// only; it never means installed, enabled or active anywhere on disk.
func optionalPresence(catalog Catalog, origin string) string {
	n := 0
	for _, item := range catalog.Items {
		if item.Origin == origin {
			n++
		}
	}
	if n == 0 {
		return "not present in this selection"
	}
	noun := "items"
	if n == 1 {
		noun = "item"
	}
	return fmt.Sprintf("present in this selection (%d %s)", n, noun)
}
