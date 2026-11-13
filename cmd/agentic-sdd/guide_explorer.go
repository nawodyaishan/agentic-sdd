package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
)

// explorerItem is one selectable row in a category list. The numeric index in
// the rendered list is its position in the item slice, so choice numbers never
// depend on parsing or re-sorting after display.
type explorerItem struct {
	id      string
	kind    string
	card    teachingCard
	entries []skillsync.InventoryEntry
	preset  bool
}

// Group display order from Feature 011 FR-12. Unknown groups are still
// grouped, after the documented ones, by name and then ID.
var (
	explorerSkillGroups = []string{
		"Scope/plan",
		"Implement/debug",
		"Verify/review",
		"Learn/reflect",
	}
	explorerProfileGroups = []string{
		"Main communicator",
		"Read-only investigation",
		"Implementation/documentation",
		"Test assistance",
		"Independent/supplementary review",
	}
)

// runGuideExplorer is the terminal-only menu 6 explorer. It reports quit=true
// when the guide should exit (EOF or an explicit q) and quit=false when the
// caller should show the main menu again. It is read-only: it inspects managed
// source/target inventory and renders teaching text, and never installs,
// dispatches, reads unrelated user profiles or contacts a provider.
func runGuideExplorer(repo, home string, p *prompter, stderr io.Writer) (quit bool, code int) {
	if p == nil || p.env == nil {
		fmt.Fprintln(stderr, "error: explorer requires an interactive terminal")
		return true, 2
	}
	for {
		p.env.Trace.Event("screen.shown", "screen", "learn.menu")
		printExplorerCategories(p.out, p.env)
		q, c, mainMenu := explorerCategories(repo, home, p, stderr)
		if q || c != 0 {
			return q, c
		}
		if mainMenu {
			return false, 0
		}
	}
}

// explorerCategories renders and handles one categories prompt. mainMenu=true
// means the caller should show the main menu; otherwise the explorer should
// render categories again after a completed category view.
func explorerCategories(repo, home string, p *prompter, stderr io.Writer) (quit bool, code int, mainMenu bool) {
	invalid := 0
	for {
		answer, kind := p.ask(
			"learn.category",
			"Enter 1-3, b for the main menu, or q to exit guide: ",
			[]string{"1", "2", "3", "b", "q"},
			"b",
		)
		switch kind {
		case "eof":
			return true, 0, false
		case "invalid":
			invalid++
			if invalid >= 3 {
				fmt.Fprintln(stderr, "error: invalid explorer selection")
				return true, 2, false
			}
			fmt.Fprintln(p.out, "Please enter 1, 2, 3, b, or q.")
			continue
		}
		invalid = 0
		switch strings.ToLower(answer) {
		case "b":
			return false, 0, true
		case "q":
			return true, 0, false
		case "1":
			if q, c := explorerSkills(repo, home, p, stderr); q || c != 0 {
				return q, c, false
			}
		case "2":
			if q, c := explorerProfiles(repo, home, p, stderr); q || c != 0 {
				return q, c, false
			}
		case "3":
			if q, c := explorerConcepts(p, stderr); q || c != 0 {
				return q, c, false
			}
		}
		return false, 0, false
	}
}

// explorerSkills lists the authoritative core-skill IDs from inventory. The
// optional OpenCode pack is deliberately not requested here.
func explorerSkills(repo, home string, p *prompter, stderr io.Writer) (quit bool, code int) {
	inv, err := skillsync.InspectSelection(repo, home, skillsync.Options{})
	if err != nil {
		fmt.Fprintln(stderr, "error: cannot read the selected core skill source")
		return true, 1
	}
	items := make([]explorerItem, 0, len(inv.SkillIDs))
	for _, id := range inv.SkillIDs {
		card := explorerCard(id, "skill")
		items = append(items, explorerItem{
			id:      id,
			kind:    "skill",
			card:    card,
			entries: explorerEntries(inv.Entries, id, "skill"),
		})
	}
	sortExplorerItems(items, "skill")
	return explorerList(p, stderr, "learn.skills", "Core skills", "core skills", items, nil)
}

// explorerProfiles lists only ProfileIDs: the orchestrate skill and other pack
// files that inventory may also return are not profile rows. An absent optional
// pack is an honest unavailable category, not an empty managed install.
func explorerProfiles(repo, home string, p *prompter, stderr io.Writer) (quit bool, code int) {
	inv, err := skillsync.InspectSelection(repo, home, skillsync.Options{WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		p.env.Trace.Event("screen.shown", "screen", "learn.profiles")
		if errors.Is(err, skillsync.ErrPackMissing) {
			printHeading(p.out, p.env, "OpenCode profiles: missing")
			printWrappedProse(p.out, "", "The optional OpenCode pack is not present in the selected source. Core skills remain available, and nothing is marked as installed.", proseWidth(p.env))
			return false, 0
		}
		printHeading(p.out, p.env, "OpenCode profiles: unavailable")
		printWrappedProse(p.out, "", "The selected OpenCode pack source could not be read safely. Core skills remain available, and nothing is marked as installed.", proseWidth(p.env))
		return false, 0
	}
	items := make([]explorerItem, 0, len(inv.ProfileIDs))
	for _, id := range inv.ProfileIDs {
		card := explorerCard(id, "profile")
		entries := explorerEntries(inv.Entries, id, "profile")
		items = append(items, explorerItem{
			id:      id,
			kind:    "profile",
			card:    card,
			entries: entries,
			preset:  explorerIsPreset(entries),
		})
	}
	sortExplorerItems(items, "profile")
	return explorerList(p, stderr, "learn.profiles", "OpenCode profile templates", "profile templates", items, nil)
}

// explorerConcepts renders the stable curated concept list. Concepts have no
// source or destination files, so inventory is never consulted.
func explorerConcepts(p *prompter, stderr io.Writer) (quit bool, code int) {
	cards := teachingConcepts()
	items := make([]explorerItem, 0, len(cards))
	for _, card := range cards {
		if card.ID == "" {
			continue
		}
		if card.Kind == "" {
			card.Kind = "concept"
		}
		items = append(items, explorerItem{id: card.ID, kind: "concept", card: card})
	}
	return explorerList(p, stderr, "learn.concepts", "Key concepts", "concepts", items, nil)
}

// explorerList renders one numbered list and its prompt. It returns to the
// categories on b, blank or EOF-at-EOF semantics as documented: blank returns
// to categories, EOF exits the guide, and q exits the guide.
func explorerList(p *prompter, stderr io.Writer, screenID, title, countLabel string, items []explorerItem, extra []string) (quit bool, code int) {
	for {
		p.env.Trace.Event("screen.shown", "screen", screenID)
		printExplorerListHeader(p.out, p.env, title, countLabel, len(items))
		if len(items) == 0 {
			printWrappedProse(p.out, "", "No items are available in the selected source.", proseWidth(p.env))
			return false, 0
		}
		currentGroup := ""
		for i, item := range items {
			if item.card.Group != "" && item.card.Group != currentGroup {
				currentGroup = item.card.Group
				fmt.Fprintln(p.out)
				fmt.Fprintf(p.out, "  --- %s ---\n", currentGroup)
			}
			printExplorerListRow(p.out, p.env, i+1, item)
		}
		for _, line := range extra {
			fmt.Fprintln(p.out, line)
		}
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, "Destination file status is shown in each item's details; it never means enabled, running, or authenticated.")
		fmt.Fprintln(p.out)
		valid := make([]string, 0, len(items)+2)
		for i := range items {
			valid = append(valid, strconv.Itoa(i+1))
		}
		valid = append(valid, "b", "q")
		invalid := 0
		for {
			answer, kind := p.ask(
				"learn.item",
				"Enter a number for details, b to go back, or q to exit guide: ",
				valid,
				"b",
			)
			switch kind {
			case "eof":
				return true, 0
			case "invalid":
				invalid++
				if invalid >= 3 {
					fmt.Fprintln(stderr, "error: invalid explorer item selection")
					return true, 2
				}
				fmt.Fprintln(p.out, "Please enter a listed number, b, or q.")
				continue
			}
			invalid = 0
			choice := strings.ToLower(answer)
			switch choice {
			case "b":
				return false, 0
			case "q":
				return true, 0
			}
			n, err := strconv.Atoi(choice)
			if err != nil || n < 1 || n > len(items) {
				fmt.Fprintln(p.out, "Please enter a listed number, b, or q.")
				continue
			}
			if q, c := explorerDetail(p, items[n-1]); q || c != 0 {
				return q, c
			}
			break // a detail view returns to this list
		}
	}
}

// explorerDetail renders one card and its actual destination records. Enter
// returns to the list, q exits the guide and EOF exits safely.
func explorerDetail(p *prompter, item explorerItem) (quit bool, code int) {
	p.env.Trace.Event("screen.shown", "screen", "learn.detail")
	printExplorerDetail(p.out, p.env, item)
	invalid := 0
	for {
		answer, kind := p.ask(
			"learn.detail",
			"Press Enter to return to the list, b to go back, or q to exit guide: ",
			[]string{"b", "q"},
			"enter",
		)
		switch kind {
		case "eof":
			return true, 0
		case "invalid":
			invalid++
			if invalid >= 3 {
				return false, 0
			}
			fmt.Fprintln(p.out, "Press Enter or b to return to the list, or q to exit guide.")
			continue
		}
		if strings.EqualFold(answer, "q") {
			return true, 0
		}
		return false, 0
	}
}

func printExplorerCategories(out io.Writer, env *Env) {
	printHeading(out, env, "Explore skills and agent profiles")
	printWrappedProse(out, "", "This screen is read-only: it explains managed skills and profile templates, and where their files would live. It installs nothing, starts no agent, and contacts no model or provider.", proseWidth(env))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "1) Core skills")
	fmt.Fprintln(out, "2) OpenCode profiles")
	fmt.Fprintln(out, "3) Key concepts")
	fmt.Fprintln(out, "b) Main menu")
	fmt.Fprintln(out, "q) Exit guide")
	fmt.Fprintln(out)
}

func printExplorerListHeader(out io.Writer, env *Env, title, countLabel string, count int) {
	printHeading(out, env, title)
	if count == 1 {
		countLabel = strings.TrimSuffix(countLabel, "s")
	}
	printWrappedProse(out, "", fmt.Sprintf("Available in the selected source: %d %s.", count, countLabel), proseWidth(env))
	fmt.Fprintln(out)
}

func printExplorerListRow(out io.Writer, env *Env, number int, item explorerItem) {
	purpose := item.card.Purpose
	if purpose == "" {
		purpose = "Purpose not documented."
	}
	suffix := ""
	if item.preset {
		suffix = " (optional preset)"
	}
	prefix := fmt.Sprintf("  %d) %s — ", number, item.id)
	available := proseWidth(env) - utf8.RuneCountInString(prefix)
	if available < 1 {
		available = 1
	}
	indent := strings.Repeat(" ", utf8.RuneCountInString(prefix))
	for i, line := range wrapProse(purpose+suffix, available) {
		if i == 0 {
			fmt.Fprintln(out, prefix+line)
			continue
		}
		fmt.Fprintln(out, indent+line)
	}
}

func printExplorerDetail(out io.Writer, env *Env, item explorerItem) {
	card := item.card
	if card.ID == "" {
		card.ID = item.id
	}
	if card.Kind == "" {
		card.Kind = item.kind
	}
	printHeading(out, env, card.ID)
	fmt.Fprintf(out, "Type: %s\n", card.Kind)
	if card.Group != "" {
		fmt.Fprintf(out, "Purpose group: %s\n", card.Group)
	}
	width := proseWidth(env)
	printDefinition(out, "Purpose: ", explorerCardField(card.Purpose), width)
	if card.When != "" {
		printDefinition(out, "When to use: ", card.When, width)
	}
	if card.Example != "" {
		printDefinition(out, "Example: ", card.Example, width)
	}
	if card.Caveat != "" {
		printDefinition(out, "Safety: ", card.Caveat, width)
	}
	if item.preset {
		fmt.Fprintln(out, "Optional preset: yes (a Free-first template, not a running process)")
	}
	if item.kind == "concept" {
		fmt.Fprintln(out, "No destination: concepts are explanatory text only.")
		fmt.Fprintln(out)
		return
	}
	fmt.Fprintln(out, "Available in selected source: yes")
	fmt.Fprintln(out, "Destination file status at the selected home (not runtime state):")
	if len(item.entries) == 0 {
		fmt.Fprintln(out, "  None recorded at the selected home.")
	} else {
		entries := append([]skillsync.InventoryEntry(nil), item.entries...)
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Path != entries[j].Path {
				return entries[i].Path < entries[j].Path
			}
			return entries[i].Client < entries[j].Client
		})
		for _, entry := range entries {
			status := string(entry.Status)
			if status == "" {
				status = "status unavailable"
			}
			origin := entry.Origin
			if origin == "" {
				origin = "unknown"
			}
			fmt.Fprintf(out, "  %s — %s; origin %s\n", entry.Path, status, origin)
		}
	}
	if item.kind == "profile" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Next steps:")
		fmt.Fprintln(out, "  To install: "+canonicalCLIName+" apply --with-opencode")
		if card.Example != "" {
			fmt.Fprintf(out, "  To use in your coding agent: %s\n", card.Example)
		}
	} else if item.kind == "skill" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Next steps:")
		fmt.Fprintln(out, "  To install: "+canonicalCLIName+" apply")
		if card.Example != "" {
			fmt.Fprintf(out, "  To use in your coding agent: %s\n", card.Example)
		}
	}
	fmt.Fprintln(out)
}

func explorerCardField(value string) string {
	if value == "" {
		return "Purpose not documented."
	}
	return value
}

// explorerCard returns the curated card for an ID, filling in only display-safe
// fallbacks. The catalog's generic custom-source text remains authoritative.
func explorerCard(id, kind string) teachingCard {
	card := teachingFor(id, kind)
	if card.ID == "" {
		card.ID = id
	}
	if card.Kind == "" {
		card.Kind = kind
	}
	if card.Group == "" {
		card.Group = "Other"
	}
	if card.Purpose == "" {
		card.Purpose = "Purpose not documented."
	}
	return card
}

func explorerEntries(entries []skillsync.InventoryEntry, id, kind string) []skillsync.InventoryEntry {
	var matched []skillsync.InventoryEntry
	for _, entry := range entries {
		if entry.ID == id && entry.Kind == kind {
			matched = append(matched, entry)
		}
	}
	return matched
}

func explorerIsPreset(entries []skillsync.InventoryEntry) bool {
	for _, entry := range entries {
		if entry.Origin == "preset" {
			return true
		}
	}
	return false
}

// sortExplorerItems groups by documented purpose order, then group name and
// ID, so the same source always renders the same numbered list.
func sortExplorerItems(items []explorerItem, kind string) {
	sort.SliceStable(items, func(i, j int) bool {
		ri, rj := explorerGroupRank(kind, items[i].card.Group), explorerGroupRank(kind, items[j].card.Group)
		if ri != rj {
			return ri < rj
		}
		if items[i].card.Group != items[j].card.Group {
			return items[i].card.Group < items[j].card.Group
		}
		return items[i].id < items[j].id
	})
}

func explorerGroupRank(kind, group string) int {
	order := explorerSkillGroups
	if kind == "profile" {
		order = explorerProfileGroups
	}
	for i, known := range order {
		if strings.EqualFold(known, group) {
			return i
		}
	}
	return len(order)
}
