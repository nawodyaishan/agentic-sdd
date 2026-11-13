package tui

// actions.go is the single action/key vocabulary for the interactive shell.
// It drives dispatch eligibility, the contextual footer and the contextual help
// section so a key is never advertised where it is not handled.
//
// The vocabulary is intentionally small and explicit. It is not a generic
// routing framework and it owns no consent or operation state.

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// confirmKeyAction classifies one key while an install or restore
// confirmation modal owns focus. Every key that is not listed here is ignored
// and can never reach the inactive backdrop.
type confirmKeyAction int

const (
	confirmKeyIgnore confirmKeyAction = iota
	confirmKeyAccept
	confirmKeyCancel
	confirmKeyInert
	confirmKeyScroll
	confirmKeyHelp
	confirmKeyShutdown
)

// classifyConfirmKey is shared by the root modal dispatch and both
// confirmation screens. Only y/Y accepts; Enter is consumed as an inert safe
// No; q is a safe cancellation, never an application quit through the modal.
func classifyConfirmKey(key string) confirmKeyAction {
	switch key {
	case "y", "Y":
		return confirmKeyAccept
	case "n", "esc", "b", "q":
		return confirmKeyCancel
	case "enter":
		return confirmKeyInert
	case "up", "k", "down", "j", "pgup", "pageup", "ctrl+u", "pgdown", "pagedown", "ctrl+d", "home", "g":
		return confirmKeyScroll
	case "?":
		return confirmKeyHelp
	case "ctrl+c":
		return confirmKeyShutdown
	default:
		return confirmKeyIgnore
	}
}

// actionBinding is one context-eligible action. keys and short form the
// concise footer entry; help is the full reference line including compatible
// aliases. priority selects what survives a narrow footer: 1 is safety/exit,
// 2 is the primary action, 3 is an optional aid.
type actionBinding struct {
	keys     string
	short    string
	help     string
	priority int
}

func (b actionBinding) footer() string {
	if b.short == "" {
		return b.keys
	}
	return b.keys + " " + b.short
}

// contextActions returns the actions the current context actually handles, in
// footer order. An open help overlay is not itself an action context; the
// footer is hidden while help owns the screen, but the help content describes
// the underlying screen.
func (a App) contextActions() []actionBinding {
	if a.helpOpen {
		return nil
	}
	return a.underlyingContextActions()
}

// underlyingContextActions returns the eligible actions for the screen beneath
// any help overlay. It is used by the contextual help section.
func (a App) underlyingContextActions() []actionBinding {
	if a.onboarding != nil {
		return nil
	}
	if a.write != nil || a.childWritePending() {
		return nil
	}

	switch a.activeRoute {
	case RouteInstall:
		switch a.installSection {
		case installSectionHub:
			return []actionBinding{
				{keys: "j/k", short: "select", help: "j/k or arrows: move the destination selection", priority: 2},
				{keys: "i", short: "install", help: "i: open the install or update journey", priority: 2},
				{keys: "b", short: "backups", help: "b: open the backup and restore journey", priority: 2},
				{keys: "Enter", short: "open", help: "Enter: open the highlighted destination", priority: 3},
				{keys: "Esc", short: "nav", help: "Esc: return focus to navigation", priority: 1},
				{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
				{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
			}
		case installSectionInstall:
			return a.installActions()
		default:
			return a.restoreActions()
		}
	case RouteActivity:
		bindings := []actionBinding{
			{keys: "r", short: "retry", help: "r: retry the update check and refresh the bounded log list", priority: 2},
			{keys: "d", short: "check", help: "d: disable or enable automatic update checks for this session", priority: 2},
			{keys: "g", short: "README", help: "g: open the online guide / README explicitly", priority: 2},
			{keys: "a", short: "author", help: "a: open the author profile explicitly", priority: 3},
			{keys: "[ ]", short: "logs", help: "[ / ] (or p / n): show older or newer operation logs", priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k and PgUp/PgDn: scroll the read-only reference", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
		if a.guideReturn {
			bindings = append(bindings, actionBinding{keys: "Esc/b", short: "Guide", help: "Esc or b: return to Guide", priority: 1})
		}
		return bindings
	case RouteGuide:
		return []actionBinding{
			{keys: "j/k", short: "select", help: "j/k or arrows: highlight a guided action", priority: 2},
			{keys: "Enter", short: "open", help: "Enter: open the highlighted destination", priority: 2},
			{keys: "g", short: "README", help: "g: open the online guide / README explicitly", priority: 2},
			{keys: "Esc", short: "nav", help: "Esc: return focus to navigation", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	}
	return nil
}

// confirmActions is the decision vocabulary for the active confirmation.
func (a App) confirmActions(restore bool) []actionBinding {
	if restore {
		return []actionBinding{
			{keys: "y", short: "restore", help: "y or Y: only y or Y confirms; Enter is inert and the default is No", priority: 1},
			{keys: "n/Esc", short: "cancel", help: "n, Esc or b: safe cancel back to the preview; q also cancels this confirmation safely and never quits through the modal", priority: 1},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the confirmation", priority: 3},
			{keys: "?", short: "help", help: "?: open contextual help without leaving the confirmation", priority: 3},
		}
	}
	return []actionBinding{
		{keys: "y", short: "confirm", help: "y or Y: only y or Y confirms; Enter is inert and the default is No", priority: 1},
		{keys: "n/Esc", short: "cancel", help: "n, Esc or b: safe cancel/go back; q also cancels this installation journey safely and never quits through the modal", priority: 1},
		{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the confirmation", priority: 3},
		{keys: "?", short: "help", help: "?: open contextual help without leaving the confirmation", priority: 3},
	}
}

func (a App) installActions() []actionBinding {
	switch a.install.Step() {
	case InstallStepClients:
		cancelAction := actionBinding{keys: "Esc", short: "cancel", help: "Esc: cancel the install journey without writing; q is the root quit key", priority: 1}
		if a.guideReturn {
			cancelAction = actionBinding{keys: "Esc/b", short: "Guide", help: "Esc or b: return to Guide without writing; q is the root quit key", priority: 1}
		}
		return []actionBinding{
			{keys: "Space", short: "select", help: "Space: toggle the highlighted client", priority: 2},
			{keys: "j/k", short: "move", help: "j/k or arrows: move the client selection", priority: 3},
			{keys: "Enter", short: "next", help: "Enter or right: continue to optional integrations", priority: 2},
			cancelAction,
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit the application when idle; Esc cancels the journey without quitting", priority: 1},
		}
	case InstallStepOptions:
		return []actionBinding{
			{keys: "Space", short: "toggle", help: "Space: toggle the highlighted optional integration", priority: 2},
			{keys: "j/k", short: "move", help: "j/k or arrows: move the option selection", priority: 3},
			{keys: "Enter", short: "next", help: "Enter or right: continue to review", priority: 2},
			{keys: "Esc", short: "back", help: "Esc, b or left: back to client selection", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit the application when idle; Esc cancels the journey without quitting", priority: 1},
		}
	case InstallStepReview:
		return []actionBinding{
			{keys: "Enter", short: "preview", help: "Enter: run the exact read-only preview", priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the review", priority: 3},
			{keys: "Esc", short: "back", help: "Esc, b or left: back to options", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit the application when idle; Esc cancels the journey without quitting", priority: 1},
		}
	case InstallStepPreview:
		if a.install.IsLoading() {
			return nil
		}
		if a.install.PreviewNoOp() {
			return []actionBinding{
				{keys: "h", short: "Home", help: "h: return Home; the read-only preview is discarded and a return requires a fresh preview", priority: 1},
				{keys: "l", short: "Library", help: "l: open Explore (the skill library); returning requires a fresh preview", priority: 2},
				{keys: "o", short: "options", help: "o: change options; a fresh preview is required afterwards", priority: 2},
				{keys: "x", short: "paths", help: "x: show or hide every item path; [ ] select a destination and Space expands it", priority: 3},
				{keys: "Esc", short: "back", help: "Esc, b or left: back to review", priority: 1},
				{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
				{keys: "q", short: "quit", help: "q: quit the application when idle; this read-only preview grants nothing", priority: 1},
			}
		}
		if !a.install.previewConfirmable() {
			return []actionBinding{
				{keys: "Esc", short: "back", help: "Esc or b: back to review; this preview is not confirmable", priority: 1},
				{keys: "x", short: "paths", help: "x: show or hide every item path so the plan stays inspectable", priority: 2},
				{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the plan", priority: 3},
				{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
				{keys: "q", short: "quit", help: "q: quit the application when idle; Esc cancels or goes back without quitting", priority: 1},
			}
		}
		return []actionBinding{
			{keys: "Enter", short: "confirm", help: "Enter: open the default-No confirmation for this exact plan", priority: 2},
			{keys: "x", short: "paths", help: "x: show or hide every item path; [ ] select a destination and Space expands it", priority: 3},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the plan", priority: 3},
			{keys: "Esc", short: "back", help: "Esc, b or left: back to review", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit the application when idle; Esc cancels or goes back without quitting", priority: 1},
		}
	case InstallStepConfirm:
		return a.confirmActions(false)
	case InstallStepRunning:
		return nil
	case InstallStepResult:
		doneHelp := "Enter or Esc: close the journey and return to the hub"
		if a.guideReturn {
			doneHelp = "Enter, Esc or b: close the journey and return to Guide"
		}
		return []actionBinding{
			{keys: "Enter", short: "done", help: doneHelp, priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the result", priority: 3},
			{keys: "r", short: "restart", help: "r: reconfigure from the beginning", priority: 3},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	}
	return nil
}

func (a App) restoreActions() []actionBinding {
	switch a.restore.Stage() {
	case RestoreStageList:
		backAction := actionBinding{keys: "Esc", short: "back", help: "Esc: return to the Install & Manage hub", priority: 1}
		if a.guideReturn {
			backAction = actionBinding{keys: "Esc/b", short: "Guide", help: "Esc or b: return to Guide", priority: 1}
		}
		return []actionBinding{
			{keys: "j/k", short: "select", help: "j/k or arrows: move the backup selection", priority: 2},
			{keys: "Enter", short: "open", help: "Enter: open the backup details (never a restore)", priority: 2},
			{keys: "r", short: "refresh", help: "r: refresh the read-only backup list", priority: 2},
			backAction,
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	case RestoreStageDetails:
		return []actionBinding{
			{keys: "Enter", short: "preview", help: "Enter or p: run the read-only restore preview", priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the details", priority: 3},
			{keys: "Esc", short: "back", help: "Esc or b: back to the backup list", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	case RestoreStagePreview:
		if a.restore.IsLoading() {
			return nil
		}
		return []actionBinding{
			{keys: "Enter", short: "confirm", help: "Enter: open the default-No restore confirmation", priority: 2},
			{keys: "p", short: "re-preview", help: "p: refresh the read-only restore preview", priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the plan", priority: 3},
			{keys: "Esc", short: "back", help: "Esc or b: back to the backup details", priority: 1},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	case RestoreStageConfirm:
		return a.confirmActions(true)
	case RestoreStageRevalidating, RestoreStageRunning:
		return nil
	case RestoreStageResult:
		return []actionBinding{
			{keys: "Enter", short: "done", help: "Enter or Esc: return to the backup list and refresh it", priority: 2},
			{keys: "p", short: "preview", help: "p: run a fresh read-only preview", priority: 2},
			{keys: "j/k", short: "scroll", help: "j/k, arrows and PgUp/PgDn: scroll the result", priority: 3},
			{keys: "?", short: "help", help: "?: open full contextual help", priority: 3},
			{keys: "q", short: "quit", help: "q: quit when idle", priority: 1},
		}
	}
	return nil
}

// actionsFooter renders the short contextual footer from the shared
// vocabulary. Priority-1 safety/exit entries are reserved before optional
// actions, so a narrow view never loses the way out; actions that do not fit
// are omitted from the short footer but remain documented in full help.
func (a App) actionsFooter(width int) string {
	bindings := a.contextActions()
	if len(bindings) == 0 {
		return ""
	}
	if width <= 0 {
		width = 80
	}
	sep := " · "
	sepWidth := ansi.StringWidth(sep)

	essentialWidth := func(from int) int {
		total, count := 0, 0
		for i := from; i < len(bindings); i++ {
			if bindings[i].priority != 1 {
				continue
			}
			if count > 0 {
				total += sepWidth
			}
			total += ansi.StringWidth(bindings[i].footer())
			count++
		}
		return total
	}

	var parts []string
	used := 0
	add := func(text string) {
		if len(parts) > 0 {
			used += sepWidth
		}
		parts = append(parts, text)
		used += ansi.StringWidth(text)
	}

	for i, binding := range bindings {
		text := binding.footer()
		w := ansi.StringWidth(text)
		if binding.priority == 1 {
			add(text)
			continue
		}
		// Reserve every later essential entry and the separators that will
		// connect this optional action to them.
		reserve := essentialWidth(i + 1)
		if reserve > 0 {
			reserve += sepWidth
		}
		extra := w
		if len(parts) > 0 {
			extra += sepWidth
		}
		if used+extra+reserve > width {
			continue
		}
		add(text)
	}
	return strings.Join(parts, sep)
}

// contextHelpLines describes the same eligible actions in full, including
// aliases omitted from the short footer.
func (a App) contextHelpLines() []string {
	bindings := a.underlyingContextActions()
	if len(bindings) == 0 {
		return nil
	}
	rows := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		rows = append(rows, "  "+binding.help)
	}
	return rows
}

// contextHelpTitle names the current action context for the help overlay. It
// describes the screen beneath help, including an open confirmation modal.
func (a App) contextHelpTitle() string {
	if a.write != nil || a.childWritePending() {
		return ""
	}
	if a.activeRoute == RouteGuide {
		return "This Screen · Guide"
	}
	if a.activeRoute == RouteActivity {
		if a.guideReturn {
			return "This Screen · Activity & Help (from Guide)"
		}
		return "This Screen · Activity & Help"
	}
	if a.activeRoute != RouteInstall {
		return ""
	}
	switch a.installSection {
	case installSectionHub:
		return "This Screen · Install & Manage"
	case installSectionInstall:
		if a.install.Step() == InstallStepConfirm {
			return "This Screen · Install confirmation"
		}
		return "This Screen · Install · " + a.install.Step().String()
	default:
		if a.restore.Stage() == RestoreStageConfirm {
			return "This Screen · Restore confirmation"
		}
		return "This Screen · Backups & restore · " + a.restore.Stage().String()
	}
}
