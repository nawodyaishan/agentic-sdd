package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

// guideExamplePrompts are the two copy-pasteable first prompts shown on the
// install closing screen. The README "Everyday prompts" list is their single
// owner; guide_test.go asserts both stay verbatim substrings of ../../README.md
// so they cannot drift.
var guideExamplePrompts = []string{
	"Use `agentic-sdd-router`. Inspect this repository, ask at most the consequential questions you can't resolve from the code, and draft `spec.md`, `plan.md`, and `tasks.md` for my combined approval. Stop before implementing.",
	"Use `agentic-sdd-implement` to fix this nil-pointer bug. Keep the diff minimal, run the focused test suite, and report what changed.",
}

// prompter owns every guide question. It reads from one shared reader so a
// scripted multi-answer stdin is never lost between steps.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
	env *Env
}

// ask prints question, reads one line and classifies the answer for the trace.
// kind is one of "default", "choice", "blank", "invalid" or "eof".
func (p *prompter) ask(id, question string, valid []string, def string) (answer, kind string) {
	p.env.Trace.Event("prompt.shown", "prompt", id, "default", def)
	fmt.Fprint(p.out, question)
	line, ok := readLine(p.in)
	if !ok {
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "eof")
		return "", "eof"
	}
	answer = strings.TrimSpace(line)
	if answer == "" {
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "default", "choice", def)
		return def, "default"
	}
	for _, v := range valid {
		if strings.EqualFold(answer, v) {
			p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "choice", "choice", answer)
			return answer, "choice"
		}
	}
	p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "invalid")
	return answer, "invalid"
}

// confirm asks a [y/N] question. The default is No, and eof reports
// end-of-input so the caller can cancel immediately instead of continuing.
func (p *prompter) confirm(id, question string) (yes bool, eof bool) {
	p.env.Trace.Event("prompt.shown", "prompt", id, "default", "n")
	fmt.Fprint(p.out, question)
	line, ok := readLine(p.in)
	if !ok {
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "eof")
		p.env.Trace.Event("confirm", "prompt", id, "decision", "no")
		return false, true
	}
	trimmed := strings.TrimSpace(line)
	yes = strings.EqualFold(trimmed, "y") || strings.EqualFold(trimmed, "yes")
	kind := "invalid"
	switch {
	case trimmed == "":
		kind = "default"
	case yes || strings.EqualFold(trimmed, "n") || strings.EqualFold(trimmed, "no"):
		kind = "choice"
	}
	if kind == "choice" {
		choiceVal := "n"
		if yes {
			choiceVal = "y"
		}
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", "choice", "choice", choiceVal)
	} else {
		p.env.Trace.Event("prompt.answered", "prompt", id, "kind", kind)
	}
	decision := "no"
	if yes {
		decision = "yes"
	}
	p.env.Trace.Event("confirm", "prompt", id, "decision", decision)
	return yes, false
}

// guideRuntime carries the effectful dependencies of one guide run so tests
// can inject an installed version, a wait function and an update check without
// mutable package-global hooks. Production defaults are built by runGuide.
type guideRuntime struct {
	installed string
	wait      func(time.Duration)
	check     func(string) updateResult
}

// runGuide implements "agentic-sdd guide": a numbered-prompt wizard over the
// existing preview, apply, restore and logs operations. It accepts only
// --repo and --home and never reads stdin without an interactive terminal.
// The production wrapper injects build metadata and a network client factory;
// the client and request are constructed only when menu choice 5 is accepted.
func runGuide(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env) int {
	return runGuideWithRuntime(args, stdout, stderr, stdin, env, guideRuntime{
		installed: version.Version,
		wait:      time.Sleep,
		check: func(current string) updateResult {
			return checkLatestRelease(newUpdateClient(), current)
		},
	})
}

// runGuideWithRuntime is the testable guide body. It keeps the original
// runGuide signature at the production seam and never mutates shared state.
func runGuideWithRuntime(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env, rt guideRuntime) int {
	fs := flag.NewFlagSet("agentic-sdd guide", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var repo, home string
	fs.StringVar(&repo, "repo", "", "repository containing skills-files (default: skills embedded in this binary)")
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printTopicHelp(stdout, "guide")
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "run agentic-sdd help guide for the accepted flags")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	// Fail fast and read nothing when stdin is not a terminal (FR-06).
	if env == nil || !env.Interactive {
		fmt.Fprintln(stderr, "error: guide requires an interactive terminal; run \"agentic-sdd preview\", \"agentic-sdd apply\", or \"agentic-sdd help guide\"")
		env.Hint(stderr, "run agentic-sdd preview to inspect planned changes, or agentic-sdd help guide")
		return 2
	}

	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	// Branding belongs to terminal stdout only and is written once, before the
	// first menu. The non-terminal branch keeps the original transcript.
	terminal := env.StdoutTTY
	if terminal && repo != "" {
		// Terminal suggestions and receipts always name the resolved source.
		// The legacy non-terminal branch keeps the argument exactly as given.
		repo, err = filepath.Abs(repo)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	}
	if terminal {
		printBrandedGuideIntro(stdout, env, rt.installed, env.Animate, rt.wait)
		printWrappedProse(stdout, "", "Skill and profile changes require confirmation.", proseWidth(env))
		printWrappedProse(stdout, "", "Write confirmation defaults to No.", proseWidth(env))
	}

	// One shared default-sized reader for the whole run: runRestore's own
	// bufio.NewReader returns this same reader, so scripted answers are not
	// swallowed between steps.
	reader := bufio.NewReader(stdin)
	p := &prompter{in: reader, out: stdout, env: env}

	invalidCount := 0
	for {
		env.Trace.Event("screen.shown", "screen", "menu")
		if terminal {
			printGuideMenuTerminal(stdout, env)
		} else {
			printGuideMenu(stdout)
		}
		env.Trace.Event("prompt.shown", "prompt", "menu", "default", "1")
		line, ok := readLine(reader)
		if !ok {
			env.Trace.Event("prompt.answered", "prompt", "menu", "kind", "eof")
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		}
		ans := strings.ToLower(strings.TrimSpace(line))
		switch {
		case ans == "":
			env.Trace.Event("prompt.answered", "prompt", "menu", "kind", "default", "choice", "1")
			ans = "1"
		case ans == "1" || ans == "2" || ans == "3" || ans == "4" || ans == "q" || ans == "quit" || (terminal && (ans == "5" || ans == "6")):
			if ans == "quit" {
				ans = "q"
			}
			env.Trace.Event("prompt.answered", "prompt", "menu", "kind", "choice", "choice", ans)
		default:
			env.Trace.Event("prompt.answered", "prompt", "menu", "kind", "invalid")
			invalidCount++
			if invalidCount >= 3 {
				fmt.Fprintf(stderr, "error: invalid menu selection %q\n", strings.TrimSpace(line))
				return 2
			}
			if terminal {
				fmt.Fprintln(stdout, "Please enter 1, 2, 3, 4, 5, 6, or q.")
			} else {
				fmt.Fprintln(stdout, "Please enter 1, 2, 3, 4, or q.")
			}
			continue
		}
		invalidCount = 0

		switch ans {
		case "q":
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		case "5":
			// Only an accepted terminal-stdout menu choice reaches here. The
			// production closure builds the client and request exactly once;
			// this loop merely returns to the menu without retrying.
			fmt.Fprintln(stdout, "\nChecking GitHub for the latest CLI release (up to 5 seconds)...")
			result := updateResult{state: updateFailed}
			if rt.check != nil {
				result = rt.check(rt.installed)
			}
			env.Trace.Event("screen.shown", "screen", "updates.result")
			printUpdateResult(stdout, result)
			continue
		case "6":
			// Optional, terminal-only learning explorer. It is read-only and
			// reports whether the user quit the guide or returned to the menu.
			quit, code := runGuideExplorer(repo, absHome, p, stderr)
			if quit {
				return code
			}
			continue
		case "4":
			env.Trace.Event("screen.shown", "screen", "about")
			if terminal {
				printGuideAboutTerminal(stdout, env)
			} else {
				printGuideAbout(stdout)
			}
			continue
		case "3":
			if err := showLatestLog(absHome, stdout); err != nil {
				if terminal && !guideLogsEmpty(err) {
					fmt.Fprintln(stderr, "error:", err)
					return 1
				}
				fmt.Fprintln(stdout, "No install or restore logs found yet.")
			}
			continue
		case "2":
			backups, err := skillsync.ListBackups(repo, home)
			if err != nil {
				fmt.Fprintln(stderr, "error:", err)
				return 1
			}
			if len(backups) == 0 {
				if terminal {
					fmt.Fprintln(stdout, "No backups found.")
					printWrappedProse(stdout, "", "Backups are created automatically when an installation replaces existing files.", proseWidth(env))
				} else {
					fmt.Fprintln(stdout, "No backups found.")
				}
				continue
			}
			if terminal {
				fmt.Fprintln(stdout)
				printCautionProse(stdout, env, "", "Restoring can replace current files and remove freshly installed items.", proseWidth(env))
				printWrappedProse(stdout, "", "Review the selected backup's preview before confirming.", proseWidth(env))
			}
			restoreHome := home
			if terminal {
				restoreHome = absHome
			}
			return runRestore(restoreArgs(repo, restoreHome), stdout, stderr, reader, env)
		case "1":
			return runGuideInstall(repo, home, absHome, p, stderr, env)
		}
	}
}

// runGuideInstall runs the guided install flow. Raw per-skill plan lines are
// buffered so the terminal shows only the concise summary and closing screen,
// while the apply log file keeps the full transcript.
func runGuideInstall(repo, home, absHome string, p *prompter, stderr io.Writer, env *Env) int {
	if env.StdoutTTY {
		return runGuideInstallTerminal(repo, home, absHome, p, stderr, env)
	}
	stdout := p.out

	env.Trace.Event("screen.shown", "screen", "install.explain")
	if env.StdoutTTY {
		printInstallExplanationTerminal(stdout, env, absHome)
	} else {
		printInstallExplanation(stdout, absHome)
	}

	withOpenCode, eof := p.confirm("install.with_opencode", "Add the OpenCode orchestration pack (extra agent profiles)? [y/N]: ")
	if eof {
		fmt.Fprintln(stdout, "Cancelled.")
		return 0
	}
	freeFirst := false
	if withOpenCode {
		freeFirst, eof = p.confirm("install.free_first", "Prefer Free models first for routine OpenCode profiles? [y/N]: ")
		if eof {
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		}
	}

	opts := skillsync.Options{WithOpenCode: withOpenCode, OpenCodeFreeFirst: freeFirst}
	var planBuf bytes.Buffer
	if err := skillsync.SyncWithOptions(repo, home, false, opts, &planBuf); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	sum := skillsync.SummarizePlan(planBuf.String())
	env.Trace.Event("screen.shown", "screen", "install.preview")
	env.Trace.Event("plan.summary", "install", sum.Install, "replace", sum.Replace, "up_to_date", sum.UpToDate)
	if env.StdoutTTY {
		printGuidePreviewTerminal(stdout, env, sum)
	} else {
		printGuidePreview(stdout, sum)
	}

	if sum.TotalChanges() == 0 {
		fmt.Fprintln(stdout, "All skills are up to date. Nothing to install.")
		return 0
	}

	okConfirm, eof := p.confirm("install.confirm", fmt.Sprintf("Install these %d items into %s? [y/N]: ", sum.TotalChanges(), absHome))
	if eof || !okConfirm {
		fmt.Fprintln(stdout, "Cancelled.")
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
	res := runOperation(repo, home, cmdName, &applyBuf, stderr, env, func(out io.Writer) error {
		return skillsync.SyncWithOptions(repo, home, true, opts, out)
	})
	if res.exitCode != 0 {
		return res.exitCode
	}

	env.Trace.Event("screen.shown", "screen", "install.done")
	if env.StdoutTTY {
		fmt.Fprintf(stdout, "%s %s\n", env.Green(statusPrefix(env)), heading(env, fmt.Sprintf("Installed %d items", sum.TotalChanges())))
		if res.backupID != "" {
			fmt.Fprintln(stdout, "Backup: "+res.backupID)
		}
		if logPath := operationLogPath(applyBuf.String()); logPath != "" {
			fmt.Fprintln(stdout, "Log: "+logPath)
		}
		fmt.Fprintln(stdout, "Undo any time: agentic-sdd restore")
		fmt.Fprintln(stdout)
		printHeading(stdout, env, "Next, in a project, tell your agent:")
		for _, prompt := range guideExamplePrompts {
			printWrappedProse(stdout, "  ", prompt, proseWidth(env))
		}
		return 0
	}
	fmt.Fprintf(stdout, "%s Installed %d items\n", env.Green(statusPrefix(env)), sum.TotalChanges())
	if res.backupID != "" {
		fmt.Fprintln(stdout, "Backup: "+res.backupID)
	}
	if logPath := operationLogPath(applyBuf.String()); logPath != "" {
		fmt.Fprintln(stdout, "Log: "+logPath)
	}
	fmt.Fprintln(stdout, "Undo any time: agentic-sdd restore")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Next, in a project, tell your agent:")
	for _, prompt := range guideExamplePrompts {
		printWrapped(stdout, "  ", prompt, 72)
	}
	return 0
}

// guideLogsEmpty reports whether a log read failed only because there is no
// log to show yet. Permission, refused-path and other safety errors stay
// errors; the terminal guide must not present them as a friendly empty state.
func guideLogsEmpty(err error) bool {
	return err != nil && (os.IsNotExist(err) || strings.HasPrefix(err.Error(), "no operation logs in "))
}

// operationLogPath returns the "Log: <path>" line runOperation appends to the
// buffered transcript, so the closing screen can name the log's location
// without dumping the per-skill lines. It returns "" when the line is absent.
func operationLogPath(transcript string) string {
	idx := strings.LastIndex(transcript, "Log: ")
	if idx < 0 {
		return ""
	}
	line, _, _ := strings.Cut(transcript[idx+len("Log: "):], "\n")
	return strings.TrimSpace(line)
}

// restoreArgs builds the existing restore command for the guide's restore
// option. It always applies, reusing the current selection and confirmation.
func restoreArgs(repo, home string) []string {
	args := []string{"--apply"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	if home != "" {
		args = append(args, "--home", home)
	}
	return args
}

func printGuideMenu(out io.Writer) {
	fmt.Fprintln(out, "Agentic SDD — engineering skills for your coding agents")
	fmt.Fprintln(out, "Nothing is written until you confirm.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "What would you like to do?")
	fmt.Fprintln(out, "  1) Install or update skills   (default)")
	fmt.Fprintln(out, "  2) Restore an earlier backup")
	fmt.Fprintln(out, "  3) Show the last install log")
	fmt.Fprintln(out, "  4) What is this?")
	fmt.Fprintln(out, "  q) Quit")
	fmt.Fprint(out, "Choose [1]: ")
}

// printGuideMenuTerminal is the styled menu for terminal stdout. The branded
// entry already carries the title, tagline, credit, version and docs, so this
// menu never repeats the legacy brand title or tagline. Choice 5 is disclosed
// as a network action and exists only on this branch.
func printGuideMenuTerminal(out io.Writer, env *Env) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, strings.Repeat("-", min(24, proseWidth(env))))
	printHeading(out, env, "Main menu")
	fmt.Fprintln(out, "  Get started:")
	fmt.Fprintln(out, "    1) Preview and install skills   (default)")
	fmt.Fprintln(out, "    4) About and getting started")
	fmt.Fprintln(out, "    6) Explore skills and agent profiles")
	fmt.Fprintln(out, "  Manage & logs:")
	fmt.Fprintln(out, "    2) Restore from a backup")
	fmt.Fprintln(out, "    3) View latest install or restore log")
	fmt.Fprintln(out, "  Help:")
	fmt.Fprintln(out, "    5) Check for CLI updates (contacts GitHub)")
	fmt.Fprintln(out, "    q) Quit")
	fmt.Fprintln(out)
	printWrappedProse(out, "  ", "Enter selects 1. Type q to exit.", proseWidth(env))
	printWrappedProse(out, "  ", "EOF exits; Ctrl+C interrupts the process.", proseWidth(env))
	fmt.Fprintln(out)
	fmt.Fprint(out, "Choose [1]: ")
}

// proseWidth is the guide's prose budget: 72 columns or the terminal width,
// whichever is smaller. Commands and complete paths are never passed through
// the wrapper.
func proseWidth(env *Env) int {
	width := brandWidth(env)
	if width > brandProseMax {
		width = brandProseMax
	}
	return width
}

// printHeading wraps plain heading text and applies bold per line, so narrow
// terminals never receive overlong styled prose.
func printHeading(out io.Writer, env *Env, text string) {
	for _, line := range wrapProse(text, proseWidth(env)) {
		fmt.Fprintln(out, heading(env, line))
	}
}

// printWrappedProse prefixes and rune-wraps guide-owned prose. Indivisible
// path/command tokens are preserved by wrapProse.
func printWrappedProse(out io.Writer, prefix, text string, width int) {
	available := width - utf8.RuneCountInString(prefix)
	if available < 1 {
		available = 1
	}
	for _, line := range wrapProse(text, available) {
		fmt.Fprintln(out, prefix+line)
	}
}

// printCautionProse is printWrappedProse with the meaningful-caution accent.
func printCautionProse(out io.Writer, env *Env, prefix, text string, width int) {
	available := width - utf8.RuneCountInString(prefix)
	if available < 1 {
		available = 1
	}
	for _, line := range wrapProse(text, available) {
		fmt.Fprintln(out, caution(env, prefix+line))
	}
}

func printGuideAbout(out io.Writer) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Agentic SDD installs reusable engineering skills for coding agents.")
	fmt.Fprintln(out, "It writes into four client directories, backs up anything it replaces,")
	fmt.Fprintln(out, "and can undo an install with \"agentic-sdd restore\".")
	fmt.Fprintln(out, "Every action is also a plain command: preview, apply, backups,")
	fmt.Fprintln(out, "restore, logs. Nothing is written until you confirm.")
	fmt.Fprintln(out)
}

// printGuideAboutTerminal keeps the About content accurate for the guide-only
// update action while repeating no branding beyond the already-shown entry.
func printGuideAboutTerminal(out io.Writer, env *Env) {
	fmt.Fprintln(out)
	printHeading(out, env, "About and getting started")
	printWrappedProse(out, "", "Agentic SDD installs reusable engineering skills for coding agents.", proseWidth(env))
	printWrappedProse(out, "", "It writes into four client directories, backs up anything it replaces, and can undo an install with \"agentic-sdd restore\".", proseWidth(env))
	printWrappedProse(out, "", "Installer actions are also plain commands: preview, apply, backups, restore, logs. The update check is guide-only and uses GitHub.", proseWidth(env))
	printWrappedProse(out, "", "Restoring can replace current files and remove freshly installed items.", proseWidth(env))
	printWrappedProse(out, "", "Upgrading the CLI does not update installed skills.", proseWidth(env))
	printWrappedProse(out, "", "After a CLI upgrade, preview skill changes and review before applying.", proseWidth(env))
	printWrappedProse(out, "", "After installing, open your coding agent in a project and ask it to use agentic-sdd-router to scope your work.", proseWidth(env))
	printWrappedProse(out, "", "To learn what the installed skills and agent profiles do, choose 6 from the main menu.", proseWidth(env))
}

func printInstallExplanation(out io.Writer, absHome string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "This installs skills into four places your agents read from:")
	fmt.Fprintln(out, "  ~/.agents/skills")
	fmt.Fprintln(out, "  ~/.codex/skills")
	fmt.Fprintln(out, "  ~/.claude/skills")
	fmt.Fprintln(out, "  ~/.gemini/antigravity-cli/skills")
	fmt.Fprintln(out, "Home: "+absHome)
}

func printInstallExplanationTerminal(out io.Writer, env *Env, absHome string) {
	fmt.Fprintln(out)
	printHeading(out, env, "This installs skills into four places your agents read from:")
	fmt.Fprintln(out, "  ~/.agents/skills")
	fmt.Fprintln(out, "  ~/.codex/skills")
	fmt.Fprintln(out, "  ~/.claude/skills")
	fmt.Fprintln(out, "  ~/.gemini/antigravity-cli/skills")
	fmt.Fprintln(out, "Home: "+absHome)
}

func printGuidePreview(out io.Writer, sum skillsync.PlanSummary) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Preview — nothing written yet")
	for _, c := range sum.Clients {
		fmt.Fprintf(out, "  %-5s %d to install, %d to replace, %d up to date\n", c.Client, c.Install, c.Replace, c.UpToDate)
	}
	fmt.Fprintf(out, "  Total: %d to install, %d to replace.\n", sum.Install, sum.Replace)
	fmt.Fprintln(out, "  Existing skills are backed up before they are replaced.")
	fmt.Fprintln(out, "  (Run agentic-sdd preview for the full per-skill list.)")
}

func printGuidePreviewTerminal(out io.Writer, env *Env, sum skillsync.PlanSummary) {
	fmt.Fprintln(out)
	printHeading(out, env, "Preview — nothing written yet")
	if brandWidth(env) < brandArtWidth {
		for _, c := range sum.Clients {
			printStackedCounts(out, c.Client, c.Install, c.Replace, c.UpToDate)
		}
		printStackedCounts(out, "Total", sum.Install, sum.Replace, sum.UpToDate)
	} else {
		for _, c := range sum.Clients {
			fmt.Fprintf(out, "  %-5s %d to install, %d to replace, %d up to date\n", c.Client, c.Install, c.Replace, c.UpToDate)
		}
		fmt.Fprintf(out, "  Total: %d to install, %d to replace.\n", sum.Install, sum.Replace)
	}
	printCautionProse(out, env, "  ", "Existing skills are backed up before they are replaced.", proseWidth(env))
	printWrappedProse(out, "  ", "(Run agentic-sdd preview for the full per-skill list.)", proseWidth(env))
}

// printWrapped writes text word-wrapped to width columns, prefixing every
// line with prefix. Guide-owned prose stays within 72 columns so no terminal
// wrapping logic is needed.
func printWrapped(out io.Writer, prefix, text string, width int) {
	words := strings.Fields(text)
	line := prefix
	for _, word := range words {
		if line != prefix && len(line)+1+len(word) > width {
			fmt.Fprintln(out, line)
			line = prefix
		}
		if line == prefix {
			line += word
		} else {
			line += " " + word
		}
	}
	fmt.Fprintln(out, line)
}
