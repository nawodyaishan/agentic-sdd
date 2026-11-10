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
//
// It is the secondary, advanced detail block of Home. The goal-first purpose,
// setup state and single primary action are composed above it.
func HomeOverview(catalog Catalog, width int) []string {
	lines := Section("Local inventory", inventoryLines(catalog), width)

	lines = append(lines, "")
	lines = append(lines, Section("Client status (core skills)", clientStatusLines(catalog), width)...)

	lines = append(lines, "")
	lines = append(lines, Section("Optional content", []string{
		fmt.Sprintf("OpenCode agent profiles: %s", optionalPresence(catalog, skillsync.InventoryOriginPack)),
		fmt.Sprintf("Prefer free models in the provided templates: %s", optionalPresence(catalog, skillsync.InventoryOriginPreset)),
	}, width)...)

	if len(catalog.Notices) > 0 {
		lines = append(lines, "")
		lines = append(lines, Section("Notices", noticeLines(catalog.Notices), width)...)
	}
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
		"Read-only local inventory; no files are changed here.",
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

// ---------------------------------------------------------------------------
// Home setup state
// ---------------------------------------------------------------------------

// SetupStateKind is the evidence-based setup state Home reports. It is derived
// from the local read-only snapshot and, when the latest refresh failed, from
// that failure. It never means authenticated, connected, enabled or running.
type SetupStateKind string

const (
	// SetupStateFirstUse means a complete local read found the core source but
	// no core destination holds content. It is not inferred from absent data
	// alone: the reported destination statuses must all be missing.
	SetupStateFirstUse SetupStateKind = "first-use"
	// SetupStateCurrent means every reported core skill destination matches the
	// selected source.
	SetupStateCurrent SetupStateKind = "current"
	// SetupStatePartial means at least one core destination is missing or
	// refused while others are present.
	SetupStatePartial SetupStateKind = "partial"
	// SetupStateModified means at least one present core destination differs
	// from the selected source.
	SetupStateModified SetupStateKind = "modified"
	// SetupStateUnknown means the snapshot cannot support a coverage
	// conclusion, so nothing is guessed.
	SetupStateUnknown SetupStateKind = "unknown"
	// SetupStateReadFailure means the latest local inventory read failed. The
	// counts shown are from the previous successful read and may be stale.
	SetupStateReadFailure SetupStateKind = "read-failure"
)

// SetupEvidence is the derived coverage evidence behind a setup state. Every
// number comes from the catalog snapshot; none is a fixed product count.
type SetupEvidence struct {
	Skills     int
	Profiles   int
	CoreSkills int
	// Destinations counts reported core-skill destination rows.
	Destinations int
	Matching     int
	Modified     int
	Missing      int
	Refused      int
	// Other counts destination statuses the snapshot did not describe, so they
	// are never folded into a claim.
	Other int
}

// Present reports how many reported core destinations hold content.
func (e SetupEvidence) Present() int { return e.Matching + e.Modified }

// SetupState is the derived Home setup state plus the evidence behind it.
type SetupState struct {
	Kind     SetupStateKind
	Evidence SetupEvidence
	// ReadErr is the latest inventory read failure, empty when the last read
	// succeeded.
	ReadErr string
}

// DeriveSetupState classifies one local snapshot. readErr is the latest
// inventory refresh error, if any. The function is pure: it reads no file,
// performs no change and invents no count.
func DeriveSetupState(catalog Catalog, readErr string) SetupState {
	state := SetupState{Evidence: deriveSetupEvidence(catalog)}
	if trimmed := strings.TrimSpace(readErr); trimmed != "" {
		state.Kind = SetupStateReadFailure
		state.ReadErr = trimmed
		return state
	}

	ev := state.Evidence
	switch {
	case ev.CoreSkills == 0 || ev.Destinations == 0:
		// No reported core destination means this snapshot cannot support a
		// coverage conclusion. "New installation" is never inferred from
		// absent data alone.
		state.Kind = SetupStateUnknown
	case ev.Present() == 0 && ev.Refused == 0 && ev.Other == 0 && ev.Missing == ev.Destinations:
		// A complete read found the core set but no destination holds content.
		state.Kind = SetupStateFirstUse
	case ev.Missing > 0 || ev.Refused > 0:
		state.Kind = SetupStatePartial
	case ev.Modified > 0:
		state.Kind = SetupStateModified
	case ev.Other > 0:
		state.Kind = SetupStateUnknown
	case ev.Matching == ev.Destinations:
		state.Kind = SetupStateCurrent
	default:
		state.Kind = SetupStateUnknown
	}
	return state
}

// deriveSetupEvidence counts the reported core-skill destinations by status.
func deriveSetupEvidence(catalog Catalog) SetupEvidence {
	ev := SetupEvidence{Skills: catalog.SkillCount(), Profiles: catalog.ProfileCount()}
	for _, item := range catalog.Items {
		if item.Kind != skillsync.InventoryKindSkill || item.Origin != skillsync.InventoryOriginCore {
			continue
		}
		ev.CoreSkills++
		for _, client := range item.Clients {
			ev.Destinations++
			switch client.Status {
			case skillsync.InventoryStatusMatchesSource:
				ev.Matching++
			case skillsync.InventoryStatusDiffersFromSource:
				ev.Modified++
			case skillsync.InventoryStatusUnavailable:
				ev.Refused++
			case skillsync.InventoryStatusMissing:
				ev.Missing++
			default:
				ev.Other++
			}
		}
	}
	return ev
}

// Headline is the one-line setup state Home shows first. It is the state, not
// a claim about a provider, account, credential or running agent.
func (s SetupState) Headline() string {
	switch s.Kind {
	case SetupStateFirstUse:
		return "Nothing installed in this home yet."
	case SetupStateCurrent:
		return "Current — every core skill destination matches the selected source."
	case SetupStatePartial:
		return "Partly installed — some core skill destinations are missing or refused."
	case SetupStateModified:
		return "Installed with local modifications."
	case SetupStateReadFailure:
		return "Unknown — the last local inventory read failed."
	default:
		return "Unknown — the local snapshot cannot support a coverage conclusion."
	}
}

// PrimaryAction is the single setup or manage action label for this state.
func (s SetupState) PrimaryAction() string {
	if s.Kind == SetupStateCurrent {
		return "Manage installation"
	}
	return "Set up / Install Agentic-SDD"
}

// Detail explains the state and the safe next step without claiming an
// operation that did not happen.
func (s SetupState) Detail(home string) string {
	switch s.Kind {
	case SetupStateFirstUse:
		where := "this home"
		if strings.TrimSpace(home) != "" {
			where = home
		}
		return "The read-only inventory found the core source but no managed content at its destinations in " + where + ". Nothing has been written."
	case SetupStateCurrent:
		return "No action is required. Manage installation still previews every destination and asks before changing a file."
	case SetupStatePartial:
		return "Run setup to preview the missing destinations. Existing files are backed up before replacement and nothing is written without your confirmation."
	case SetupStateModified:
		return "Present files differ from the selected source. Setup backs up existing content before replacing it and requires explicit confirmation first."
	case SetupStateReadFailure:
		return "The read error was: " + s.ReadErr + ". Counts below come from the last successful read and may be stale; this failure is not an operation outcome."
	default:
		return "The snapshot did not report enough core destination evidence to classify this home. Nothing was inferred from missing data."
	}
}

// EvidenceLine is the secondary derived count line for the setup state.
func (s SetupState) EvidenceLine() string {
	ev := s.Evidence
	if ev.Destinations == 0 {
		return fmt.Sprintf("Evidence: %d skills · %d profiles in this selection; no core client destinations were reported.", ev.Skills, ev.Profiles)
	}
	line := fmt.Sprintf(
		"Evidence: %d/%d core destinations present (%d matching, %d modified) · %d not present · %d refused",
		ev.Present(), ev.Destinations, ev.Matching, ev.Modified, ev.Missing, ev.Refused,
	)
	if ev.Other > 0 {
		line += fmt.Sprintf(" · %d with an unrecognized status", ev.Other)
	}
	line += fmt.Sprintf(". Source content: %d skills · %d profiles.", ev.Skills, ev.Profiles)
	return line
}

// HomePurposeLines is the two-line value proposition Home opens with.
func HomePurposeLines() []string {
	return []string{
		"Install shared Agentic SDD skills for your coding agents and keep them up to date.",
		"Every change is previewed first and requires your explicit confirmation.",
	}
}

// HomeSetupLines renders the setup-state section. The evidence line is
// secondary to the state itself and never replaces it.
func HomeSetupLines(state SetupState, home string, width int) []string {
	body := []string{state.Headline(), state.Detail(home), state.EvidenceLine()}
	return Section("Setup", body, width)
}

// HomeActionLines renders the single primary action plus the discovery and
// recovery links. It never repeats the setup state as a second action.
func HomeActionLines(state SetupState, width int) []string {
	return Section("Recommended next step", []string{
		state.PrimaryAction() + "  (Enter or s)",
		"Browse Skills (key 3): read-only catalog of every managed skill and profile.",
		"Recovery (key b): inspect recorded backups and restore one after a read-only preview.",
	}, width)
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
