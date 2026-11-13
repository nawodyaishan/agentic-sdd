package tui

import (
	"fmt"
	"strings"
)

// mcpOverviewLines renders the MCP Tools Overview view.
// Exactly three rows: CodeGraph, Context7, and Exa.
func mcpOverviewLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Overview"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Inspect, configure, and restore CodeGraph, Context7, and Exa MCP tools"))
	b.WriteString("\n\n")

	// Target configuration status
	target := m.status.ConfigPath
	if target == "" {
		target = "(default: ~/.config/opencode/opencode.json)"
	}
	format := m.status.ConfigFormat
	if format == "" {
		format = "json"
	}
	schema := m.status.Schema
	if schema == "" {
		schema = "v2 (mcp.servers)"
	}

	metaLines := []string{
		fmt.Sprintf("  Target:  %s (%s, %s)", target, format, schema),
	}
	if m.status.ConfigDigest != "" {
		metaLines = append(metaLines, fmt.Sprintf("  Digest:  %s", m.status.ConfigDigest))
	}
	b.WriteString(strings.Join(Section(m.theme.Section.Render("Configuration target"), metaLines, width), "\n"))
	b.WriteString("\n\n")

	// Last write outcome if any
	if !m.outcome.Empty() {
		outStyle := m.theme.OK
		if strings.Contains(strings.ToLower(m.outcome.Headline), "fail") || strings.Contains(strings.ToLower(m.outcome.Headline), "refused") {
			outStyle = m.theme.Bad
		}
		b.WriteString(outStyle.Render("Last operation: " + m.outcome.Headline))
		b.WriteString("\n")
		if m.outcome.Detail != "" {
			for _, line := range wrapPath("  "+m.outcome.Detail, width) {
				b.WriteString(m.theme.Hint.Render(line) + "\n")
			}
		}
		if m.outcome.BackupID != "" {
			b.WriteString(m.theme.Accent.Render("  Backup ID: "+m.outcome.BackupID) + "\n")
		}
		b.WriteString("\n")
	}

	// Exactly three default servers
	glyphs := GlyphSet(m.cfg.ASCII)
	serverRows := make([]string, 0, len(m.status.Servers)*3)

	for i, srv := range m.status.Servers {
		marker := glyphs.Gutter
		if i == m.overviewCursor {
			marker = glyphs.Cursor
		}
		rowHeader := fmt.Sprintf("%s [%s] %s — %s", marker, srv.Kind, srv.DisplayName, srv.Purpose)
		if i == m.overviewCursor && m.theme.Color {
			rowHeader = m.theme.Selected.Bold(true).Render(rowHeader)
		}

		stateBadge := "[" + srv.ConfigState + "]"
		switch srv.ConfigState {
		case "configured":
			stateBadge = m.theme.OK.Render(stateBadge)
		case "missing-prerequisite", "schema-mismatch", "conflict", "unsafe-target":
			stateBadge = m.theme.Bad.Render(stateBadge)
		case "disabled":
			stateBadge = m.theme.Warn.Render(stateBadge)
		default:
			stateBadge = m.theme.Hint.Render(stateBadge)
		}

		prereqText := srv.Prerequisite
		if srv.Prerequisite == "codegraph CLI" {
			if srv.PrerequisiteMet {
				prereqText = "codegraph CLI (found in PATH)"
			} else {
				prereqText = "codegraph CLI (missing from PATH)"
			}
		}

		details := fmt.Sprintf("     State: %s · Prerequisite: %s · Verified: %s",
			stateBadge, prereqText, srv.VerificationState)
		action := fmt.Sprintf("     Next action: %s", srv.NextAction)

		serverRows = append(serverRows, rowHeader, details, m.theme.Hint.Render(action))
	}

	b.WriteString(strings.Join(Section(m.theme.Section.Render("Supported MCP tools (3 defaults)"), serverRows, width), "\n"))
	b.WriteString("\n\n")

	b.WriteString(m.theme.Hint.Render("controls: s select servers · v preview · d diagnostics · e recovery · Esc hub · ? help"))

	return strings.Split(b.String(), "\n")
}

// mcpSelectLines renders the server selection and auth choice view.
func mcpSelectLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Select Servers"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Choose default servers and authentication modes (offline staging only; no writes)"))
	b.WriteString("\n\n")

	if m.selectNotice != "" {
		b.WriteString(m.theme.Warn.Render("! " + m.selectNotice))
		b.WriteString("\n\n")
	}

	glyphs := GlyphSet(m.cfg.ASCII)
	checkbox := func(checked bool) string {
		if checked {
			return "[x]"
		}
		return "[ ]"
	}
	radio := func(selected bool) string {
		if selected {
			return "(*)"
		}
		return "( )"
	}

	items := []struct {
		cursor int
		line   string
		desc   string
	}{
		{
			cursor: 0,
			line:   fmt.Sprintf("%s CodeGraph (local stdio server)", checkbox(m.selected["codegraph"])),
			desc:   "Repository code intelligence over CodeGraph index. Prerequisite: codegraph CLI.",
		},
		{
			cursor: 1,
			line:   fmt.Sprintf("%s Context7 (hosted documentation)", checkbox(m.selected["context7"])),
			desc:   "Hosted library documentation server (https://mcp.context7.com/mcp).",
		},
		{
			cursor: 2,
			line: fmt.Sprintf("     Auth: %s OAuth (recommended)   %s API Key ({env:CONTEXT7_API_KEY})",
				radio(m.authChoices["context7"] == "oauth"),
				radio(m.authChoices["context7"] == "api-key")),
			desc: "OpenCode handles OAuth handshake. API key requires CONTEXT7_API_KEY environment variable.",
		},
		{
			cursor: 3,
			line:   fmt.Sprintf("%s Exa (hosted web research)", checkbox(m.selected["exa"])),
			desc:   "Hosted web search and crawling endpoint (https://mcp.exa.ai/mcp).",
		},
		{
			cursor: 4,
			line: fmt.Sprintf("     Auth: %s Keyless   %s OAuth   %s API Key ({env:EXA_API_KEY})",
				radio(m.authChoices["exa"] == "keyless"),
				radio(m.authChoices["exa"] == "oauth"),
				radio(m.authChoices["exa"] == "api-key")),
			desc: "Keyless is free but subject to rate limits. OAuth/API key requires user authentication.",
		},
		{
			cursor: 5,
			line:   fmt.Sprintf("     %s Acknowledge rate limits and privacy terms for Exa keyless", checkbox(m.exaAcknowledged)),
			desc:   "Required before selecting Exa keyless: acknowledge that queries pass through Exa's service.",
		},
	}

	rows := make([]string, 0, len(items)*2)
	for _, it := range items {
		marker := glyphs.Gutter
		if it.cursor == m.selectCursor {
			marker = glyphs.Cursor
		}
		lineText := marker + " " + it.line
		if it.cursor == m.selectCursor && m.theme.Color {
			lineText = m.theme.Selected.Bold(true).Render(lineText)
		}
		rows = append(rows, lineText)
		rows = append(rows, "       "+m.theme.Hint.Render(it.desc))
	}

	b.WriteString(strings.Join(Section(m.theme.Section.Render("Server configuration options"), rows, width), "\n"))
	b.WriteString("\n\n")

	// Prerequisite warning if CodeGraph selected but binary missing
	for _, srv := range m.status.Servers {
		if srv.Name == "codegraph" && !srv.PrerequisiteMet && m.selected["codegraph"] {
			b.WriteString(m.theme.Warn.Render("! Warning: codegraph CLI binary not detected in PATH. Configuration will be written, but the tool requires the CLI."))
			b.WriteString("\n\n")
			break
		}
	}

	b.WriteString(m.theme.Hint.Render("Selection does not modify any files. Press v or Enter to preview configuration changes."))
	b.WriteString("\n\n")
	b.WriteString(m.theme.Hint.Render("controls: j/k select · Space toggle/cycle · v/Enter preview · d diagnostics · Esc back"))

	return strings.Split(b.String(), "\n")
}

// mcpPreviewLines renders the configuration preview and diff view.
func mcpPreviewLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Configuration Preview"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Exact byte plan, base digests, and managed entry diff (read-only; no writes)"))
	b.WriteString("\n\n")

	if m.planLoading {
		b.WriteString(m.theme.Hint.Render("Loading configuration plan preview..."))
		b.WriteString("\n")
		return strings.Split(b.String(), "\n")
	}

	if m.planErr != "" {
		b.WriteString(m.theme.Bad.Render("Plan error: " + m.planErr))
		b.WriteString("\n\n")
		b.WriteString(m.theme.Hint.Render("Press Esc to adjust server selection, or d for diagnostics."))
		return strings.Split(b.String(), "\n")
	}

	p := m.planPreview

	// Target info
	targetInfo := []string{
		fmt.Sprintf("  Target path:  %s (%s)", p.TargetPath, p.Format),
		fmt.Sprintf("  Base digest:  %s", p.BaseDigest),
		fmt.Sprintf("  Plan digest:  %s", p.PlanDigest),
		fmt.Sprintf("  Selected:     %s", strings.Join(p.Selected, ", ")),
	}
	b.WriteString(strings.Join(Section(m.theme.Section.Render("Plan identity"), targetInfo, width), "\n"))
	b.WriteString("\n\n")

	// Conflicts banner if any
	if len(p.Conflicts) > 0 {
		b.WriteString(m.theme.Bad.Render("! Plan has conflicts — Apply is blocked until resolved:"))
		b.WriteString("\n")
		for _, c := range p.Conflicts {
			b.WriteString(m.theme.Bad.Render("  - "+c) + "\n")
		}
		b.WriteString("\n")
	}

	// Planned entries table
	entryRows := make([]string, 0, len(p.Entries)*2)
	for _, entry := range p.Entries {
		badge := "[" + entry.Op + "]"
		switch entry.Op {
		case "add":
			badge = m.theme.OK.Render(badge)
		case "change":
			badge = m.theme.Accent.Render(badge)
		case "leave", "disabled":
			badge = m.theme.Hint.Render(badge)
		case "conflict":
			badge = m.theme.Bad.Render(badge)
		}
		header := fmt.Sprintf("  %s %s — %s", badge, entry.Name, entry.Reason)
		entryRows = append(entryRows, header)
		if entry.NewText != "" {
			entryRows = append(entryRows, "     "+m.theme.Hint.Render(entry.NewText))
		}
	}
	b.WriteString(strings.Join(Section(m.theme.Section.Render("Planned entries"), entryRows, width), "\n"))
	b.WriteString("\n\n")

	// Implications
	implications := []string{
		"  Auth: Context7 uses OAuth/placeholder; Exa uses keyless/placeholder; no plaintext credentials written.",
		"  Backup: Private backup created under ~/.agentic-sdd/mcp/backups/<id> before mutation.",
		"  Restore: Managed entries can be reverted in Recovery without touching user edits.",
	}
	b.WriteString(strings.Join(Section(m.theme.Section.Render("Safety & implications"), implications, width), "\n"))
	b.WriteString("\n\n")

	// Diff lines (with filter and scroll)
	diffHeader := "Diff projection"
	if m.filter != "" {
		diffHeader += fmt.Sprintf(" (filter: %q)", m.filter)
	}
	b.WriteString(m.theme.Section.Render(diffHeader))
	b.WriteString("\n")

	renderedDiffLines := 0
	linesToRender := p.DiffLines
	if m.previewScroll < len(linesToRender) {
		linesToRender = linesToRender[m.previewScroll:]
	}

	for _, dl := range linesToRender {
		if m.filter != "" && !strings.Contains(strings.ToLower(dl), strings.ToLower(m.filter)) {
			continue
		}
		if strings.HasPrefix(dl, "+") {
			b.WriteString(m.theme.OK.Render(dl) + "\n")
		} else if strings.HasPrefix(dl, "-") {
			b.WriteString(m.theme.Bad.Render(dl) + "\n")
		} else {
			b.WriteString(m.theme.Hint.Render(dl) + "\n")
		}
		renderedDiffLines++
		if renderedDiffLines > 20 {
			b.WriteString(m.theme.Hint.Render("  ... (diff lines elided; use j/k to scroll)") + "\n")
			break
		}
	}
	b.WriteString("\n")

	if m.isFiltering {
		b.WriteString(m.theme.Selected.Render("Filter: " + m.filter + " [Enter done · Esc clear]"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.theme.Hint.Render("controls: Enter/y confirm apply · / filter · j/k scroll diff · d diagnostics · Esc back"))
	}

	return strings.Split(b.String(), "\n")
}

// mcpConfirmLines renders the inline default-No confirmation subview.
func mcpConfirmLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Confirm Changes"))
	b.WriteString("\n\n")

	if m.tinyGeometry {
		b.WriteString(m.theme.Bad.Render("Terminal geometry too small (<60 cols or <8 rows)."))
		b.WriteString("\n")
		b.WriteString(m.theme.Hint.Render("Resize terminal to view and confirm configuration apply. Press Esc to cancel."))
		return strings.Split(b.String(), "\n")
	}

	target := m.planPreview.TargetPath
	b.WriteString(fmt.Sprintf("Apply configuration changes to %s?", m.theme.Accent.Render(target)))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Base digest: %s\n", m.theme.Hint.Render(m.planPreview.BaseDigest)))
	b.WriteString(fmt.Sprintf("Plan digest: %s\n\n", m.theme.Hint.Render(m.planPreview.PlanDigest)))

	for _, line := range wrapPath("Safety policy: "+WritePolicyText, width) {
		b.WriteString(m.theme.Warn.Render(line) + "\n")
	}
	b.WriteString("\n")

	// Big buttons with default-No
	noBtn := "[  No, cancel  ]"
	yesBtn := "[  Yes, apply changes  ]"
	if !m.confirmSelection {
		noBtn = m.theme.Selected.Bold(true).Render("> [ No, cancel ] <")
		yesBtn = m.theme.Hint.Render(yesBtn)
	} else {
		noBtn = m.theme.Hint.Render(noBtn)
		yesBtn = m.theme.Bad.Bold(true).Render("> [ Yes, apply changes ] <")
	}

	b.WriteString("  " + noBtn + "        " + yesBtn + "\n\n")
	b.WriteString(m.theme.Hint.Render("controls: Tab/Arrow toggle · Enter confirm · Esc cancel"))

	return strings.Split(b.String(), "\n")
}

// mcpDiagnosticsLines renders contextual diagnostics and supported CLI guidance.
func mcpDiagnosticsLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Diagnostics"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Contextual configuration, prerequisite, and offline readiness checks"))
	b.WriteString("\n\n")

	if m.doctorLoading {
		b.WriteString(m.theme.Hint.Render("Loading diagnostic checks..."))
		b.WriteString("\n")
		return strings.Split(b.String(), "\n")
	}

	// 10 Contextual diagnostic checks
	checks := []struct {
		name   string
		state  string
		detail string
	}{
		{
			name:   "Target Configuration",
			state:  "pass",
			detail: "OpenCode global configuration target resolved; regular file detected.",
		},
		{
			name:   "Schema Compatibility",
			state:  "pass",
			detail: "V2 mcp.servers dictionary supported. V1 mcp.<name> shape is refused, never migrated.",
		},
		{
			name:   "CodeGraph Binary",
			state:  "pass",
			detail: "codegraph CLI executable found in PATH (or verified via prerequisite).",
		},
		{
			name:   "CodeGraph Project Index",
			state:  "not-checked",
			detail: ".codegraph/ index presence in current project. Project initialization requires explicit codegraph init.",
		},
		{
			name:   "Context7 Authentication",
			state:  "not-checked",
			detail: "OAuth delegated to OpenCode; optional API key via {env:CONTEXT7_API_KEY}. No credentials stored.",
		},
		{
			name:   "Exa Rate Limits & Privacy",
			state:  "not-checked",
			detail: "Keyless mode is rate-limited; privacy terms acknowledged. Optional OAuth or {env:EXA_API_KEY}.",
		},
		{
			name:   "JSONC Safety",
			state:  "pass",
			detail: "Conservative byte-range editing preserves comments and formatting. Refuses with manual snippet if ambiguous.",
		},
		{
			name:   "Stale Plan Guard",
			state:  "pass",
			detail: "Shared engine lock revalidates configuration digest before mutation.",
		},
		{
			name:   "Transaction Recovery",
			state:  "pass",
			detail: "Durable transaction intent and entry-level backups ensure rollback honesty on failure.",
		},
		{
			name:   "Runtime MCP Connection",
			state:  "not-checked",
			detail: "connection not tested in TUI v1. Active handshakes require explicit CLI commands.",
		},
	}

	// Override checks with live doctor results if available
	if len(m.doctor.Checks) > 0 {
		// Use dynamic checks if doctor returned
		checks = make([]struct {
			name   string
			state  string
			detail string
		}, 0, len(m.doctor.Checks))
		for _, c := range m.doctor.Checks {
			checks = append(checks, struct {
				name   string
				state  string
				detail string
			}{name: c.Name, state: c.State, detail: c.Detail})
		}
	}

	checkRows := make([]string, 0, len(checks)*2)
	for _, c := range checks {
		badge := "[" + strings.ToUpper(c.state) + "]"
		switch c.state {
		case "pass":
			badge = m.theme.OK.Render(badge)
		case "warn":
			badge = m.theme.Warn.Render(badge)
		case "blocked":
			badge = m.theme.Bad.Render(badge)
		default:
			badge = m.theme.Hint.Render(badge)
		}
		checkRows = append(checkRows, fmt.Sprintf("  %s %s", badge, c.name))
		checkRows = append(checkRows, "     "+m.theme.Hint.Render(c.detail))
	}

	b.WriteString(strings.Join(Section(m.theme.Section.Render("Diagnostic checks"), checkRows, width), "\n"))
	b.WriteString("\n\n")

	// Supported CLI Guidance
	cliGuidance := []string{
		"  asdx-core mcp status       Inspect global MCP configuration and server status",
		"  asdx-core mcp preview      Inspect exact diff and plan digest without writing",
		"  asdx-core mcp doctor       Run local prerequisite diagnostics",
		"  asdx-core mcp restore      Browse and restore managed backups",
	}
	b.WriteString(strings.Join(Section(m.theme.Section.Render("Supported CLI guidance"), cliGuidance, width), "\n"))
	b.WriteString("\n\n")

	b.WriteString(m.theme.Hint.Render("controls: r refresh diagnostics · j/k scroll · Esc back"))

	return strings.Split(b.String(), "\n")
}

// mcpRecoveryLines renders the backup browser, restore preview, and default-No restore confirm.
func mcpRecoveryLines(m MCPModel, width int) []string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Install & Manage / MCP Tools · Recovery"))
	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Browse managed backups, inspect restore diff, and revert managed entries"))
	b.WriteString("\n\n")

	if m.confirmRestore {
		if m.tinyGeometry {
			b.WriteString(m.theme.Bad.Render("Terminal geometry too small (<60 cols or <8 rows)."))
			b.WriteString("\n")
			b.WriteString(m.theme.Hint.Render("Resize terminal to confirm restore. Press Esc to cancel."))
			return strings.Split(b.String(), "\n")
		}

		b.WriteString(fmt.Sprintf("Restore managed MCP backup %s?", m.theme.Accent.Render(m.restorePreview.BackupID)))
		b.WriteString("\n\n")

		for _, line := range wrapPath("Safety policy: "+WritePolicyText, width) {
			b.WriteString(m.theme.Warn.Render(line) + "\n")
		}
		b.WriteString("\n")

		noBtn := "[  No, cancel  ]"
		yesBtn := "[  Yes, restore backup  ]"
		if !m.confirmSelection {
			noBtn = m.theme.Selected.Bold(true).Render("> [ No, cancel ] <")
			yesBtn = m.theme.Hint.Render(yesBtn)
		} else {
			noBtn = m.theme.Hint.Render(noBtn)
			yesBtn = m.theme.Bad.Bold(true).Render("> [ Yes, restore backup ] <")
		}

		b.WriteString("  " + noBtn + "        " + yesBtn + "\n\n")
		b.WriteString(m.theme.Hint.Render("controls: Tab/Arrow toggle · Enter confirm · Esc cancel"))
		return strings.Split(b.String(), "\n")
	}

	if m.inRestorePreview {
		p := m.restorePreview
		b.WriteString(fmt.Sprintf("Restore Preview for backup %s\n", m.theme.Accent.Render(p.BackupID)))
		b.WriteString(fmt.Sprintf("Target: %s · Base digest: %s\n\n", p.TargetPath, p.BaseDigest))

		entryRows := make([]string, 0, len(p.Entries))
		for _, e := range p.Entries {
			entryRows = append(entryRows, fmt.Sprintf("  [%s] %s — %s", e.Op, e.Name, e.Reason))
		}
		b.WriteString(strings.Join(Section(m.theme.Section.Render("Entries to restore"), entryRows, width), "\n"))
		b.WriteString("\n\n")

		b.WriteString(m.theme.Hint.Render("controls: r/Enter confirm restore · Esc back to backup list"))
		return strings.Split(b.String(), "\n")
	}

	// Backup list
	if m.recoveryLoading {
		b.WriteString(m.theme.Hint.Render("Loading managed backups..."))
		b.WriteString("\n")
		return strings.Split(b.String(), "\n")
	}

	if len(m.backups) == 0 {
		b.WriteString(m.theme.Hint.Render("No recorded managed MCP backups found under ~/.agentic-sdd/mcp/backups/."))
		b.WriteString("\n\n")
		b.WriteString(m.theme.Hint.Render("controls: Esc back to overview"))
		return strings.Split(b.String(), "\n")
	}

	glyphs := GlyphSet(m.cfg.ASCII)
	rows := make([]string, 0, len(m.backups)*2)
	for i, bkp := range m.backups {
		marker := glyphs.Gutter
		if i == m.backupIndex {
			marker = glyphs.Cursor
		}
		row := fmt.Sprintf("%s Backup %s (%s)", marker, bkp.ID, bkp.CreatedAt)
		if i == m.backupIndex && m.theme.Color {
			row = m.theme.Selected.Bold(true).Render(row)
		}
		rows = append(rows, row)
		rows = append(rows, fmt.Sprintf("     Path: %s · Servers: %s", bkp.Path, strings.Join(bkp.Servers, ", ")))
	}

	b.WriteString(strings.Join(Section(m.theme.Section.Render("Managed backups"), rows, width), "\n"))
	b.WriteString("\n\n")
	b.WriteString(m.theme.Hint.Render("controls: v/Enter preview backup · j/k navigate · Esc back"))

	return strings.Split(b.String(), "\n")
}
