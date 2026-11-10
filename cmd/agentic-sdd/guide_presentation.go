package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
)

// guideReviewMinWidth is the preview table breakpoint from FR-11. The separate
// art/animation gate remains brandArtWidth (60).
const guideReviewMinWidth = 80

// guideClientOrder keeps destination rows in the same order the installer
// plans and writes them. Unknown client names follow, sorted.
var guideClientOrder = []string{"agents", "codex", "claude", "agy", "opencode-agents"}

// guideSkillGroupOrder mirrors the curated teaching groups so a receipt groups
// installed skills by purpose without inventing a second inventory.
var guideSkillGroupOrder = []string{
	"Scope/plan",
	"Implement/debug",
	"Verify/review",
	"Learn/reflect",
}

// guideProfileGroupOrder mirrors the curated profile role groups.
var guideProfileGroupOrder = []string{
	"Main communicator",
	"Read-only investigation",
	"Implementation/documentation",
	"Test assistance",
	"Independent/supplementary review",
}

// guideTerminalBugExample replaces the legacy nil-pointer example in the
// terminal receipt. It names a real task category without implying every user
// has the same fictional defect.
const guideTerminalBugExample = "Use `agentic-sdd-implement` to fix this bug. Keep the diff minimal, run the focused tests, and report what changed and what remains uncertain."

// guideConfirmChoice is the final terminal installation-review answer. It is
// deliberately separate from prompter.confirm's (yes, eof) contract.
type guideConfirmChoice int

const (
	guideConfirmNo guideConfirmChoice = iota
	guideConfirmYes
	guideConfirmBack
	guideConfirmEOF
)

// confirmWithBack reads one final installation review answer. Only this
// terminal review accepts b; b records a choice plus a no decision, writes
// nothing, and tells the caller to ask the options again.
func (p *prompter) confirmWithBack(id, question string) guideConfirmChoice {
	p.env.Trace.Event("prompt.shown", "prompt", id, "default", "n")
	fmt.Fprint(p.out, question)
	line, ok := readLine(p.in)
	if !ok {
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "eof")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return guideConfirmEOF
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	switch answer {
	case "":
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "default")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return guideConfirmNo
	case "y", "yes":
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "choice", "choice", "y")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "yes")
		return guideConfirmYes
	case "n", "no":
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "choice", "choice", "n")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return guideConfirmNo
	case "b":
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "choice", "choice", "b")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return guideConfirmBack
	default:
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "invalid")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return guideConfirmNo
	}
}

// runGuideInstallTerminal is the terminal-only options -> review -> result
// flow. It never duplicates installer business logic: preview and apply call
// the same skillsync functions the ordinary commands use.
func runGuideInstallTerminal(repo, home, absHome string, p *prompter, stderr io.Writer, env *Env) int {
	for {
		env.Trace.Event("screen.shown", "screen", "install.explain")
		coreInventory, err := skillsync.InspectSelection(repo, home, skillsync.Options{})
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		profileDestination, packErr := guideProfileDestination(repo, home)
		printGuideOptionsTerminal(p.out, env, absHome, coreInventory, profileDestination, packErr)

		withOpenCode, eof := p.confirm("install.with_opencode", "Add the OpenCode orchestration pack (extra agent profiles)? [y/N]: ")
		if eof {
			fmt.Fprintln(p.out, "Cancelled.")
			return 0
		}
		freeFirst := false
		if withOpenCode {
			freeFirst, eof = p.confirm("install.free_first", "Prefer Free models first for routine OpenCode profiles? [y/N]: ")
			if eof {
				fmt.Fprintln(p.out, "Cancelled.")
				return 0
			}
		}

		opts := skillsync.Options{WithOpenCode: withOpenCode, OpenCodeFreeFirst: freeFirst}
		inventory, err := skillsync.InspectSelection(repo, home, opts)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}

		var planBuf bytes.Buffer
		if err := skillsync.SyncWithOptions(repo, home, false, opts, &planBuf); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		sum := skillsync.SummarizePlan(planBuf.String())
		env.Trace.Event("screen.shown", "screen", "install.preview")
		env.Trace.Event("plan.summary", "install", sum.Install, "replace", sum.Replace, "up_to_date", sum.UpToDate)
		printGuideReviewTerminal(p.out, env, inventory, sum, opts)

		if sum.TotalChanges() == 0 {
			printGuideAlreadyUpToDateTerminal(p.out, env)
			return 0
		}

		switch p.confirmWithBack("install.confirm", "Install these changes? [y/N/b=change options]: ") {
		case guideConfirmBack:
			// Options again for the same home/repo; this round writes nothing
			// and the next preview is recomputed from the new selection.
			continue
		case guideConfirmYes:
			// Continue below to the shared operation wrapper.
		default:
			fmt.Fprintln(p.out, "Cancelled.")
			return 0
		}

		cmdName := "apply"
		if opts.WithOpenCode {
			cmdName += " --with-opencode"
			if opts.OpenCodeFreeFirst {
				cmdName += " --opencode-free-first"
			}
		}
		var applyBuf bytes.Buffer
		result := runLoggedOperation(repo, home, cmdName, &applyBuf, stderr, env, func(out io.Writer) (string, error) {
			applied, err := skillsync.SyncWithOptionsResult(repo, home, true, opts, out)
			return applied.BackupID, err
		})
		logPath := operationLogPath(applyBuf.String())
		if result.exitCode != 0 {
			printGuideInstallFailure(p.out, env, result, logPath)
			return result.exitCode
		}

		env.Trace.Event("screen.shown", "screen", "install.done")
		printGuideReceiptTerminal(p.out, env, inventory, result, logPath)
		return 0
	}
}

// guideProfileDestination asks the read-only inventory for the OpenCode profile
// target without duplicating the target-path policy in the view. It returns
// ErrPackMissing for an absent optional source.
func guideProfileDestination(repo, home string) (string, error) {
	inventory, err := skillsync.InspectSelection(repo, home, skillsync.Options{WithOpenCode: true})
	if err != nil {
		return "", err
	}
	for _, destination := range guideDestinations(inventory, skillsync.Options{WithOpenCode: true}) {
		if destination.Client == "opencode-agents" {
			return destination.Path, nil
		}
	}
	return "", nil
}

type guideDestination struct {
	Client string
	Label  string
	Path   string
}

func guideClientLabel(client string) string {
	switch client {
	case "agents":
		return "Shared agents"
	case "codex":
		return "Codex"
	case "claude":
		return "Claude Code"
	case "agy", "antigravity":
		return "Antigravity"
	case "opencode-agents":
		return "OpenCode profiles"
	default:
		return client
	}
}

// guideDestinations returns one actual destination directory per managed
// client. Entries are skill directories or profile files, so the parent is the
// destination root in both cases.
func guideDestinations(inventory skillsync.SelectionInventory, opts skillsync.Options) []guideDestination {
	paths := make(map[string]string)
	for _, entry := range inventory.Entries {
		if entry.Path == "" {
			continue
		}
		if entry.Kind == skillsync.InventoryKindProfile && !opts.WithOpenCode {
			continue
		}
		if _, seen := paths[entry.Client]; seen {
			continue
		}
		paths[entry.Client] = filepath.Dir(entry.Path)
	}
	clients := make([]string, 0, len(paths))
	for client := range paths {
		clients = append(clients, client)
	}
	orderedClients := guideOrderedClients(clients)
	destinations := make([]guideDestination, 0, len(orderedClients))
	for _, client := range orderedClients {
		destinations = append(destinations, guideDestination{
			Client: client,
			Label:  guideClientLabel(client),
			Path:   paths[client],
		})
	}
	return destinations
}

func guideOrderedClients(clients []string) []string {
	seen := make(map[string]bool, len(clients))
	for _, client := range clients {
		seen[client] = true
	}
	ordered := make([]string, 0, len(clients))
	for _, client := range guideClientOrder {
		if seen[client] {
			ordered = append(ordered, client)
			delete(seen, client)
		}
	}
	rest := make([]string, 0, len(seen))
	for client := range seen {
		rest = append(rest, client)
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}

// guideDefinitionHangingMax is the longest label that still shares its first
// line with the text. Longer labels become block headers, because a hanging
// indent of their width would leave too little room to wrap the text even at 80
// columns.
const guideDefinitionHangingMax = 16

// printDefinition writes one definition-list field such as "Purpose:" or
// "Safety:". A short label (at most guideDefinitionHangingMax runes) keeps the
// label and the first text line together, and every later line is indented by
// the label's rune count so the text stays in one aligned column. A long label
// is printed alone and the text follows as a two-space indented block.
//
// Labels carry their own separator ("Purpose: "), so a hanging indent aligns
// the text one column past the colon. Wrapping goes through wrapProse, which
// never splits a token: the only line that can exceed width is a single word or
// copyable token longer than the available column, which the caller must keep
// whole. Empty text prints the label alone rather than dropping the field.
func printDefinition(out io.Writer, label, text string, width int) {
	labelWidth := utf8.RuneCountInString(label)
	if labelWidth > guideDefinitionHangingMax {
		fmt.Fprintln(out, label)
		const indent = "  "
		available := width - utf8.RuneCountInString(indent)
		if available < 1 {
			available = 1
		}
		for _, line := range wrapProse(text, available) {
			fmt.Fprintln(out, indent+line)
		}
		return
	}
	available := width - labelWidth
	if available < 1 {
		available = 1
	}
	lines := wrapProse(text, available)
	if len(lines) == 0 {
		fmt.Fprintln(out, label)
		return
	}
	fmt.Fprintln(out, label+lines[0])
	indent := strings.Repeat(" ", labelWidth)
	for _, line := range lines[1:] {
		fmt.Fprintln(out, indent+line)
	}
}

// printGuideOptionsTerminal is stage 1. It shows actual targets under the
// selected home, explains skill versus profile, the pack and Free-first effects
// before the questions, and never prints an alternative-home shortcut.
func printGuideOptionsTerminal(out io.Writer, env *Env, absHome string, coreInventory skillsync.SelectionInventory, profileDestination string, packErr error) {
	fmt.Fprintln(out)
	printHeading(out, env, "1. Choose options")
	fmt.Fprintln(out, "Target home: "+absHome)
	printWrappedProse(out, "", "A skill is reusable guidance loaded by a coding agent. A profile is a client-specific agent configuration file, not a running process.", proseWidth(env))
	fmt.Fprintln(out)
	printHeading(out, env, "Skill destinations")
	for _, destination := range guideDestinations(coreInventory, skillsync.Options{}) {
		fmt.Fprintf(out, "  %s: %s\n", destination.Label, destination.Path)
	}
	if profileDestination != "" {
		fmt.Fprintf(out, "  %s: %s\n", guideClientLabel("opencode-agents"), profileDestination)
	} else if packErr == nil {
		fmt.Fprintln(out, "  OpenCode profiles: destination unavailable in the selected source.")
	}
	fmt.Fprintln(out)
	printHeading(out, env, "OpenCode pack (optional)")
	printWrappedProse(out, "", "It adds the agentic-sdd-orchestrate skill to each skill destination and specialist agent profile files. Existing files are backed up before they are replaced.", proseWidth(env))
	if packErr != nil {
		if errors.Is(packErr, skillsync.ErrPackMissing) {
			printCautionProse(out, env, "", "The optional OpenCode pack source is missing; core skills remain available. Selecting the pack will fail until it is restored.", proseWidth(env))
		} else {
			printCautionProse(out, env, "", "The optional OpenCode pack source is unavailable; core skills remain available. Selecting the pack will fail until the source is fixed.", proseWidth(env))
		}
	}
	printHeading(out, env, "Free-first presets (optional)")
	printWrappedProse(out, "", "Free-first changes routing defaults for four routine profiles: sdd-implement-1, sdd-implement-2, sdd-test-implement-1 and sdd-test-implement-2.", proseWidth(env))
	printWrappedProse(out, "", "It does not authenticate providers, and it guarantees neither free execution nor future free-model availability.", proseWidth(env))
	fmt.Fprintln(out)
}

type guideReviewRow struct {
	Client    string
	Label     string
	New       int
	Replace   int
	Unchanged int
}

func guideReviewRows(inventory skillsync.SelectionInventory, sum skillsync.PlanSummary) []guideReviewRow {
	rowsByClient := make(map[string]*guideReviewRow)
	var clients []string
	for _, entry := range inventory.Entries {
		if _, seen := rowsByClient[entry.Client]; seen {
			continue
		}
		rowsByClient[entry.Client] = &guideReviewRow{Client: entry.Client, Label: guideClientLabel(entry.Client)}
		clients = append(clients, entry.Client)
	}
	for _, client := range sum.Clients {
		if _, seen := rowsByClient[client.Client]; seen {
			continue
		}
		rowsByClient[client.Client] = &guideReviewRow{Client: client.Client, Label: guideClientLabel(client.Client)}
		clients = append(clients, client.Client)
	}
	for _, client := range sum.Clients {
		row := rowsByClient[client.Client]
		row.New = client.Install
		row.Replace = client.Replace
		row.Unchanged = client.UpToDate
	}
	rows := make([]guideReviewRow, 0, len(clients))
	for _, client := range guideOrderedClients(clients) {
		rows = append(rows, *rowsByClient[client])
	}
	return rows
}

func guideTableWidths(rows []guideReviewRow) (label, newColumn, replace, unchanged, total int) {
	label = utf8.RuneCountInString("Destination")
	newColumn = utf8.RuneCountInString("New")
	replace = utf8.RuneCountInString("Replace")
	unchanged = utf8.RuneCountInString("Unchanged")
	for _, row := range rows {
		if n := utf8.RuneCountInString(row.Label); n > label {
			label = n
		}
		if n := len(strconv.Itoa(row.New)); n > newColumn {
			newColumn = n
		}
		if n := len(strconv.Itoa(row.Replace)); n > replace {
			replace = n
		}
		if n := len(strconv.Itoa(row.Unchanged)); n > unchanged {
			unchanged = n
		}
	}
	total = label + 2 + newColumn + 2 + replace + 2 + unchanged
	return label, newColumn, replace, unchanged, total
}

func guideTableFits(rows []guideReviewRow, width int) bool {
	if width < guideReviewMinWidth {
		return false
	}
	_, _, _, _, total := guideTableWidths(rows)
	return total <= width
}

func printGuideChangeTable(out io.Writer, rows []guideReviewRow) {
	label, newColumn, replace, unchanged, _ := guideTableWidths(rows)
	fmt.Fprintf(out, "%-*s  %*s  %*s  %*s\n", label, "Destination", newColumn, "New", replace, "Replace", unchanged, "Unchanged")
	for _, row := range rows {
		fmt.Fprintf(out, "%-*s  %*d  %*d  %*d\n", label, row.Label, newColumn, row.New, replace, row.Replace, unchanged, row.Unchanged)
	}
}

func printGuideChangeRecords(out io.Writer, rows []guideReviewRow) {
	for _, row := range rows {
		fmt.Fprintln(out, row.Label)
		fmt.Fprintf(out, "  New: %d\n", row.New)
		fmt.Fprintf(out, "  Replace: %d\n", row.Replace)
		fmt.Fprintf(out, "  Unchanged: %d\n", row.Unchanged)
	}
}

// printGuideReviewTerminal is stage 2. Destination labels never stand in for
// paths, and the counts remain the last lines before the caller prints the
// confirmation prompt.
func printGuideReviewTerminal(out io.Writer, env *Env, inventory skillsync.SelectionInventory, sum skillsync.PlanSummary, opts skillsync.Options) {
	fmt.Fprintln(out)
	printHeading(out, env, "2. Review changes")
	rows := guideReviewRows(inventory, sum)
	if guideTableFits(rows, brandWidth(env)) {
		printGuideChangeTable(out, rows)
	} else {
		printGuideChangeRecords(out, rows)
	}
	for _, destination := range guideDestinations(inventory, opts) {
		fmt.Fprintf(out, "%s: %s\n", destination.Label, destination.Path)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Equivalent commands (copyable):")
	fmt.Fprintln(out, guidePreviewCommand(inventory.Home, inventory.Repo, opts))
	fmt.Fprintln(out, guideApplyCommand(inventory.Home, inventory.Repo, opts))
	fmt.Fprintln(out)

	// The target and complete counts are the last block before the caller
	// prints the confirmation prompt.
	fmt.Fprintln(out, "Target home: "+inventory.Home)
	uniqueSkills, skillTargets, skillEntries, profileEntries, desired := guideInventoryCounts(inventory)
	fmt.Fprintf(out, "%d unique skill%s across %s: %d entries\n",
		uniqueSkills, guidePlural(uniqueSkills), guideDestinationCount(skillTargets), skillEntries)
	if profileEntries == 0 {
		fmt.Fprintln(out, "Core-only selection: no profiles selected.")
	} else {
		fmt.Fprintf(out, "%d profile files: %d entries\n", profileEntries, profileEntries)
	}
	fmt.Fprintf(out, "%d managed entries: %d will change, %d already match.\n", desired, sum.TotalChanges(), sum.UpToDate)
	if sum.Replace > 0 {
		printCautionProse(out, env, "", "Replacements are backed up first.", proseWidth(env))
	}
}

func guidePlural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func guideDestinationCount(n int) string {
	if n == 4 {
		return "four destinations"
	}
	return fmt.Sprintf("%d destination%s", n, guidePlural(n))
}

// guideInventoryCounts derives the desired inventory from the source-selected
// IDs and managed targets, never from a fixed total. Skill entries repeat once
// per client target; profiles are one file each.
func guideInventoryCounts(inventory skillsync.SelectionInventory) (uniqueSkills, skillTargets, skillEntries, profileEntries, desired int) {
	uniqueSkills = len(inventory.SkillIDs)
	clients := make(map[string]bool)
	for _, entry := range inventory.Entries {
		switch entry.Kind {
		case skillsync.InventoryKindSkill:
			skillEntries++
			clients[entry.Client] = true
		case skillsync.InventoryKindProfile:
			profileEntries++
		}
	}
	skillTargets = len(clients)
	desired = skillEntries + profileEntries
	return uniqueSkills, skillTargets, skillEntries, profileEntries, desired
}

func guidePreviewCommand(home, repo string, opts skillsync.Options) string {
	return guideSelectionCommand("preview", home, repo, opts)
}

func guideApplyCommand(home, repo string, opts skillsync.Options) string {
	return guideSelectionCommand("apply", home, repo, opts)
}

func guideSelectionCommand(name, home, repo string, opts skillsync.Options) string {
	cmd := "agentic-sdd " + name
	if opts.WithOpenCode {
		cmd += " --with-opencode"
	}
	if opts.OpenCodeFreeFirst {
		cmd += " --opencode-free-first"
	}
	if repo != "" {
		cmd += " --repo " + shellQuoteIfNeeded(repo)
	}
	cmd += " --home " + shellQuoteIfNeeded(home)
	return cmd
}

func guideReentryCommand(home, repo string) string {
	cmd := "agentic-sdd guide --home " + shellQuoteIfNeeded(home)
	if repo != "" {
		cmd += " --repo " + shellQuoteIfNeeded(repo)
	}
	return cmd
}

func guideRecoveryCommand(id, home, repo string) string {
	cmd := "agentic-sdd restore " + id + " --home " + shellQuoteIfNeeded(home)
	if repo != "" {
		cmd += " --repo " + shellQuoteIfNeeded(repo)
	}
	return cmd
}

func printGuideAlreadyUpToDateTerminal(out io.Writer, env *Env) {
	fmt.Fprintln(out)
	printHeading(out, env, "3. Installation result")
	printHeading(out, env, "Already up to date")
	printWrappedProse(out, "", "All skills are up to date. No operation log or snapshot was created.", proseWidth(env))
}

func printGuideInstallFailure(out io.Writer, env *Env, result operationResult, logPath string) {
	fmt.Fprintln(out)
	printHeading(out, env, "3. Installation result")
	printHeading(out, env, "Installation failed")
	printWrappedProse(out, "", "Installation failed; inspect the operation log before retrying.", proseWidth(env))
	printWrappedProse(out, "", "No success was recorded. Review the log and the current files before retrying; completed changes may not have been rolled back.", proseWidth(env))
	if logPath != "" {
		fmt.Fprintln(out, "Log: "+logPath)
	}
}

// printGuideReceiptTerminal is stage 3. Changed counts come from the successful
// operation transcript, not the preview; the snapshot location comes from the
// same read-only inventory used for the review.
func printGuideReceiptTerminal(out io.Writer, env *Env, inventory skillsync.SelectionInventory, result operationResult, logPath string) {
	actual := skillsync.SummarizePlan(result.transcript)
	changed := actual.TotalChanges()
	fmt.Fprintln(out)
	printHeading(out, env, "3. Installation result")
	fmt.Fprintf(out, "%s %s\n", env.Green("Success:"), heading(env, fmt.Sprintf("Installed %d managed entries.", changed)))

	printHeading(out, env, "Installed / updated")
	_, _, _, _, desired := guideInventoryCounts(inventory)
	fmt.Fprintf(out, "Desired managed entries: %d. Changed: %d (%d new, %d replaced).\n", desired, changed, actual.Install, actual.Replace)
	printGuideChangedGroups(out, env, inventory, result.transcript)
	if logPath != "" {
		fmt.Fprintln(out, "Log: "+logPath)
	}
	printGuideRecovery(out, env, inventory, result.backupID)
	printGuideNextSteps(out, env, inventory)
}

// printGuideChangedGroups parses only successful per-entry change lines from
// the operation transcript, then groups them with the curated teaching map.
// It shows counts, not the eighteen profile cards, and never invents a group.
func printGuideChangedGroups(out io.Writer, env *Env, inventory skillsync.SelectionInventory, transcript string) {
	skills := make(map[string]bool)
	profiles := make(map[string]bool)
	for _, raw := range strings.Split(transcript, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		var rest string
		switch {
		case strings.HasPrefix(line, "install "):
			rest = strings.TrimPrefix(line, "install ")
		case strings.HasPrefix(line, "replace + back up "):
			rest = strings.TrimPrefix(line, "replace + back up ")
		default:
			continue
		}
		client, id, ok := strings.Cut(rest, "/")
		if !ok || id == "" {
			continue
		}
		if client == "opencode-agents" {
			profiles[strings.TrimSuffix(id, ".md")] = true
		} else {
			skills[id] = true
		}
	}

	printHeading(out, env, "Skills by purpose")
	if len(skills) == 0 {
		fmt.Fprintln(out, "  No skill files changed.")
	} else {
		counts := make(map[string]int)
		for id := range skills {
			counts[teachingFor(id, teachingKindSkill).Group]++
		}
		printGroupCounts(out, counts, guideSkillGroupOrder)
	}

	_, _, _, profileEntries, _ := guideInventoryCounts(inventory)
	printHeading(out, env, "Profiles by role")
	switch {
	case profileEntries == 0:
		fmt.Fprintln(out, "  none selected (core-only install)")
	case len(profiles) == 0:
		fmt.Fprintln(out, "  No profile files changed.")
	default:
		counts := make(map[string]int)
		for id := range profiles {
			counts[teachingFor(id, teachingKindProfile).Group]++
		}
		printGroupCounts(out, counts, guideProfileGroupOrder)
	}
}

func printGroupCounts(out io.Writer, counts map[string]int, preferred []string) {
	printed := make(map[string]bool, len(counts))
	for _, group := range preferred {
		if counts[group] > 0 {
			fmt.Fprintf(out, "  %s: %d\n", group, counts[group])
			printed[group] = true
		}
	}
	rest := make([]string, 0, len(counts))
	for group := range counts {
		if !printed[group] {
			rest = append(rest, group)
		}
	}
	sort.Strings(rest)
	for _, group := range rest {
		fmt.Fprintf(out, "  %s: %d\n", group, counts[group])
	}
}

func printGuideRecovery(out io.Writer, env *Env, inventory skillsync.SelectionInventory, backupID string) {
	if backupID == "" || inventory.BackupRoot == "" {
		return
	}
	printHeading(out, env, "Recovery")
	fmt.Fprintln(out, "Snapshot: "+backupID)
	fmt.Fprintln(out, "Location: "+filepath.Join(inventory.BackupRoot, backupID))
	printWrappedProse(out, "", "This snapshot restores the state before this installation. For a fresh install it removes the newly installed entries; it does not contain earlier file content.", proseWidth(env))
	fmt.Fprintln(out, "Preview restoring this snapshot:")
	fmt.Fprintln(out, guideRecoveryCommand(backupID, inventory.Home, inventory.Repo))
}

func printGuideNextSteps(out io.Writer, env *Env, inventory skillsync.SelectionInventory) {
	fmt.Fprintln(out)
	printHeading(out, env, "Next steps")
	printWrappedProse(out, "", "Open your coding agent in a project and ask agentic-sdd-router to scope the work.", proseWidth(env))
	fmt.Fprintln(out, "Plan a feature:")
	printWrappedProse(out, "  ", guideExamplePrompts[0], proseWidth(env))
	fmt.Fprintln(out, "Fix an existing bug:")
	printWrappedProse(out, "  ", guideTerminalBugExample, proseWidth(env))
	fmt.Fprintln(out)
	printWrappedProse(out, "", "Explore the available profiles: run the command below, then choose 6.", proseWidth(env))
	fmt.Fprintln(out, guideReentryCommand(inventory.Home, inventory.Repo))
}
