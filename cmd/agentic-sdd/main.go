package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

func main() {
	env := resolveEnv(os.Stdin, os.Stdout, os.Stderr, os.Getenv)
	if tracePath := strings.TrimSpace(os.Getenv("AGENTIC_SDD_TRACE")); tracePath != "" {
		tracer, err := openTrace(tracePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: trace disabled: %v\n", err)
		} else {
			env.Trace = tracer
		}
	}
	os.Exit(runEnv(os.Args[1:], os.Stdout, os.Stderr, os.Stdin, &env))
}

// run keeps the original testable seam: it builds a plain, non-terminal Env
// (only stdin interaction comes from the argument) so every existing test
// exercises the pre-feature, byte-identical path.
func run(args []string, stdout, stderr io.Writer, stdin io.Reader, interactive bool) int {
	return runEnv(args, stdout, stderr, stdin, &Env{Interactive: interactive, Width: 80})
}

func runEnv(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env) (exitCode int) {
	if env == nil {
		env = &Env{Width: 80}
	}
	env.Trace.Event("run.start", "command", commandName(args), "flags", flagNames(args), "version", version.Version)
	env.Trace.Event("env.resolved",
		"interactive", env.Interactive,
		"stdout_tty", env.StdoutTTY,
		"stderr_tty", env.StderrTTY,
		"color", env.Color,
		"ascii", env.ASCII,
		"width", env.Width,
	)
	defer func() {
		env.Trace.Event("run.end", "exit", exitCode)
		_ = env.Trace.Close()
	}()
	command := "preview"
	explicitCommand := false
	if len(args) > 0 {
		switch args[0] {
		case "preview", "apply":
			command, explicitCommand = args[0], true
			args = args[1:]
		case "version":
			if len(args) > 1 {
				fmt.Fprintln(stderr, "error: usage: agentic-sdd version")
				return 2
			}
			printVersion(stdout)
			return 0
		case "opencode", "opencode-harness":
			return runOpenCode(args[1:], stdout, stderr)
		case "help":
			if len(args) > 2 || (len(args) == 2 && !isHelpTopic(args[1])) {
				fmt.Fprintln(stderr, "error: usage: agentic-sdd help [preview|apply|backups|logs|restore]")
				return 2
			}
			if len(args) == 2 && (args[1] == "opencode" || args[1] == "opencode-harness") {
				printOpenCodeUsage(stdout)
				return 0
			}
			if len(args) == 2 && args[1] == "orchestrate" {
				printOrchestrateUsage(stdout)
				return 0
			}
			if len(args) == 2 {
				printTopicHelp(stdout, args[1])
			}
			printUsage(stdout)
			return 0
		case "backups":
			return runBackups(args[1:], stdout, stderr, env)
		case "logs":
			return runLogs(args[1:], stdout, stderr, env)
		case "restore":
			return runRestore(args[1:], stdout, stderr, stdin, env)
		case "guide":
			return runGuide(args[1:], stdout, stderr, stdin, env)
		default:
			if !strings.HasPrefix(args[0], "-") {
				fmt.Fprintf(stderr, "error: unknown command %q\n", args[0])
				env.Hint(stderr, "run agentic-sdd help to see the commands")
				printUsage(stderr)
				return 2
			}
		}
	}

	fs := flag.NewFlagSet("agentic-sdd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var repo, home string
	var legacyApply, showVersion, withOpenCode, openCodeFreeFirst bool
	fs.StringVar(&repo, "repo", "", "repository containing skills-files (default: skills embedded in this binary)")
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	fs.BoolVar(&legacyApply, "apply", false, "back up and install skills (legacy form)")
	fs.BoolVar(&showVersion, "version", false, "print version information")
	fs.BoolVar(&withOpenCode, "with-opencode", false, "install opt-in OpenCode orchestration pack and agent profiles")
	fs.BoolVar(&openCodeFreeFirst, "opencode-free-first", false, "use Free-first model ladder preset for OpenCode implementation/test profiles")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "run agentic-sdd help preview for the accepted flags")
		printUsage(stderr)
		return 2
	}
	if showVersion {
		printVersion(stdout)
		return 0
	}
	if openCodeFreeFirst && !withOpenCode {
		fmt.Fprintln(stderr, "error: --opencode-free-first requires --with-opencode")
		env.Hint(stderr, "add --with-opencode, or drop --opencode-free-first")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		printUsage(stderr)
		return 2
	}
	if explicitCommand {
		usedLegacyApply := false
		fs.Visit(func(f *flag.Flag) { usedLegacyApply = usedLegacyApply || f.Name == "apply" })
		if usedLegacyApply {
			fmt.Fprintln(stderr, "error: --apply cannot be used with a command")
			return 2
		}
	} else if legacyApply {
		command = "apply"
	}
	// Save the flag as the user wrote it: resolveHome replaces an empty value
	// with the real home directory, but the summary's next-command line must
	// carry --home only when it was explicitly passed.
	rawHome := home
	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	opts := skillsync.Options{
		WithOpenCode:      withOpenCode,
		OpenCodeFreeFirst: openCodeFreeFirst,
	}
	if command == "apply" {
		return runApplyWithLog(repo, home, opts, stdout, stderr, env)
	}
	if !env.StdoutTTY {
		if err := skillsync.SyncWithOptions(repo, home, false, opts, stdout); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	var planBuf bytes.Buffer
	if err := skillsync.SyncWithOptions(repo, home, false, opts, io.MultiWriter(stdout, &planBuf)); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	printPreviewSummary(stdout, env, skillsync.SummarizePlan(planBuf.String()), repo, rawHome, opts)
	return 0
}

// printPreviewSummary appends the terminal-only preview counts block to the
// unchanged plan transcript. It writes nothing when stdout is not a terminal,
// which keeps piped and scripted output byte-identical.
func printPreviewSummary(out io.Writer, env *Env, sum skillsync.PlanSummary, repo, rawHome string, opts skillsync.Options) {
	if env == nil || !env.StdoutTTY {
		return
	}
	fmt.Fprintln(out)
	printHeading(out, env, "Summary (preview — nothing was written)")
	if brandWidth(env) < brandArtWidth {
		for _, c := range sum.Clients {
			printStackedCounts(out, c.Client, c.Install, c.Replace, c.UpToDate)
		}
		printStackedCounts(out, "Total", sum.Install, sum.Replace, sum.UpToDate)
	} else {
		for _, c := range sum.Clients {
			fmt.Fprintf(out, "  %-16s %d to install, %d to replace, %d up to date\n", c.Client, c.Install, c.Replace, c.UpToDate)
		}
		fmt.Fprintf(out, "  Total: %d to install, %d to replace (backed up first), %d up to date — nothing was written.\n",
			sum.Install, sum.Replace, sum.UpToDate)
	}
	if sum.TotalChanges() > 0 {
		fmt.Fprintf(out, "  Next: %s\n", formatNextApplyCommand(repo, rawHome, opts))
	}
}

// formatNextApplyCommand rebuilds the exact apply command for the next step,
// carrying only the flags the user actually supplied.
func formatNextApplyCommand(repo, rawHome string, opts skillsync.Options) string {
	cmd := "agentic-sdd apply"
	if opts.WithOpenCode {
		cmd += " --with-opencode"
	}
	if opts.OpenCodeFreeFirst {
		cmd += " --opencode-free-first"
	}
	if repo != "" {
		cmd += " --repo " + shellQuoteIfNeeded(repo)
	}
	if rawHome != "" {
		cmd += " --home " + shellQuoteIfNeeded(rawHome)
	}
	return cmd
}

// shellQuoteIfNeeded leaves ordinary path characters alone and single-quotes
// a value that a shell would otherwise split or expand.
func shellQuoteIfNeeded(s string) string {
	if s == "" {
		return "''"
	}
	ordinary := func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return true
		}
		return strings.ContainsRune("-_.:/@%+=,", r)
	}
	for _, r := range s {
		if !ordinary(r) {
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}

func isHelpTopic(s string) bool {
	switch s {
	case "preview", "apply", "backups", "logs", "restore", "guide", "version", "opencode", "opencode-harness", "orchestrate":
		return true
	}
	return false
}

func printOrchestrateUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd [preview|apply] --with-opencode [--opencode-free-first] [--repo PATH] [--home PATH]

OpenCode Orchestration Options:
  --with-opencode        Install the opt-in agentic-sdd-orchestrate skill into client
                         skill directories and place OpenCode v2 agent profiles in
                         ~/.config/opencode/agents.
  --opencode-free-first  Preset OpenCode writer and test profiles to prefer free models
                         first before scoped-approval models (requires --with-opencode).
`)
}

// resolveHome returns home unchanged if set, or the user's real home
// directory otherwise.
func resolveHome(home string) (string, error) {
	if home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}

// runBackups implements the read-only "agentic-sdd backups" command, and
// "agentic-sdd restore list" (an identical listing) delegates to it.
func runBackups(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("agentic-sdd backups", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var repo, home string
	fs.StringVar(&repo, "repo", "", "repository containing skills-files (default: skills embedded in this binary)")
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "run agentic-sdd help backups for the accepted flags")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	backups, err := skillsync.ListBackups(repo, home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	printBackups(stdout, backups)
	return 0
}

func printBackups(out io.Writer, backups []skillsync.Backup) {
	if len(backups) == 0 {
		fmt.Fprintln(out, "No backups found.")
		return
	}
	for i, b := range backups {
		if b.Unusable != "" {
			fmt.Fprintf(out, "%d) %s  unusable: %s\n", i+1, b.ID, b.Unusable)
			continue
		}
		when := b.Manifest.CreatedAt
		if when == "" {
			when = b.Manifest.Created
		}
		op := b.Manifest.Operation
		switch {
		case op == "" || op == "apply":
			op = "apply"
		case op == "restore" && b.Manifest.RestoredFrom != "":
			op = "restore from " + b.Manifest.RestoredFrom
		}
		toolVersion := "unknown"
		if b.Manifest.Tool != nil {
			toolVersion = b.Manifest.Tool.Version
		}
		legacy := ""
		if b.Legacy {
			legacy = " [legacy]"
		}
		fmt.Fprintf(out, "%d) %s%s  %s  %s  tool %s  source %s  %s\n", i+1, b.ID, legacy, when, op, toolVersion, b.Manifest.Source, b.Summary)
	}
}

// runRestore implements every form of the "agentic-sdd restore" command:
// the sub-word forms ("restore list|preview|apply [ID]") and the bare form
// ("restore [ID] [--apply]"). "restore list" is identical to "backups".
func runRestore(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env) int {
	mode := ""
	if len(args) > 0 {
		switch args[0] {
		case "list", "preview", "apply":
			mode, args = args[0], args[1:]
		}
	}
	if mode == "list" {
		return runBackups(args, stdout, stderr, env)
	}

	fs := flag.NewFlagSet("agentic-sdd restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var repo, home string
	var applyFlag bool
	fs.StringVar(&repo, "repo", "", "repository containing skills-files (default: skills embedded in this binary)")
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	fs.BoolVar(&applyFlag, "apply", false, "restore the selected backup (bare form only; use \"restore apply\" instead of combining this with a sub-word)")

	// flag.FlagSet.Parse stops at the first positional argument, so a
	// single call can't see a flag placed after the backup ID. Re-parse
	// the remainder after skipping at most one positional (the ID) until
	// nothing but flags is left, so "restore ID --apply" and
	// "restore --apply ID" both work.
	var id string
	haveID := false
	remaining := args
	for {
		if err := fs.Parse(remaining); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printUsage(stdout)
				return 0
			}
			fmt.Fprintln(stderr, "error:", err)
			env.Hint(stderr, "run agentic-sdd help restore for the accepted flags")
			return 2
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		if haveID {
			fmt.Fprintf(stderr, "error: unexpected argument %q\n", rest[0])
			return 2
		}
		id, haveID = rest[0], true
		remaining = rest[1:]
	}

	if mode != "" && applyFlag {
		fmt.Fprintln(stderr, "error: --apply cannot be used with a command")
		return 2
	}
	applyRestore := applyFlag || mode == "apply"

	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	var reader *bufio.Reader
	if env.Interactive {
		reader = bufio.NewReader(stdin)
	}

	interactivelySelected := false
	if !haveID {
		if !env.Interactive {
			fmt.Fprintln(stderr, `error: restore requires a backup id; run "agentic-sdd backups" to list them`)
			env.Hint(stderr, "run agentic-sdd backups to list ids, or agentic-sdd guide to pick one")
			return 2
		}
		backups, err := skillsync.ListBackups(repo, home)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		selected, code := selectBackup(backups, reader, stdout, stderr, env)
		if code >= 0 {
			return code
		}
		id, interactivelySelected = selected, true
	} else if err := skillsync.ValidateBackupID(id); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "ids look like 20260101T000000.000000000Z; list them with agentic-sdd backups")
		return 2
	}

	if applyRestore && interactivelySelected {
		var plan bytes.Buffer
		if err := skillsync.Restore(repo, home, id, false, &plan); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		text := strings.TrimSuffix(plan.String(), "Preview only. Run with --apply (or restore apply) to restore.\n")
		fmt.Fprint(stdout, text)
		env.Trace.Event("prompt.shown", "prompt", "restore.confirm", "default", "n")
		fmt.Fprint(stdout, "Restore these skills? [y/N]: ")
		line, ok := readLine(reader)
		trimmed := strings.TrimSpace(line)
		yes := strings.EqualFold(trimmed, "y") || strings.EqualFold(trimmed, "yes")
		kind := "invalid"
		switch {
		case !ok:
			kind = "eof"
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
			env.Trace.Event("prompt.answered", "prompt", "restore.confirm", "kind", "choice", "choice", choiceVal)
		} else {
			env.Trace.Event("prompt.answered", "prompt", "restore.confirm", "kind", kind)
		}
		if !yes {
			env.Trace.Event("confirm", "prompt", "restore.confirm", "decision", "no")
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		}
		env.Trace.Event("confirm", "prompt", "restore.confirm", "decision", "yes")
	}

	if applyRestore {
		return runRestoreWithLog(repo, home, id, stdout, stderr, env)
	}
	if err := skillsync.Restore(repo, home, id, false, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

// selectBackup lists backups and reads a numbered choice from reader. The
// returned int is -1 when id holds a valid selection to proceed with, or
// the exit code to return immediately (0 to cancel, 2 for a bad answer, 1
// if the chosen backup turns out to be unusable).
func selectBackup(backups []skillsync.Backup, reader *bufio.Reader, stdout, stderr io.Writer, env *Env) (string, int) {
	if len(backups) == 0 {
		fmt.Fprintln(stdout, "No backups found.")
		return "", 0
	}
	printBackups(stdout, backups)
	env.Trace.Event("prompt.shown", "prompt", "restore.select", "default", "")
	fmt.Fprintf(stdout, "Select a backup [1-%d] (blank to cancel): ", len(backups))
	line, ok := readLine(reader)
	line = strings.TrimSpace(line)
	if !ok {
		env.Trace.Event("prompt.answered", "prompt", "restore.select", "kind", "eof")
		fmt.Fprintln(stdout, "Cancelled.")
		return "", 0
	}
	if line == "" {
		env.Trace.Event("prompt.answered", "prompt", "restore.select", "kind", "blank")
		fmt.Fprintln(stdout, "Cancelled.")
		return "", 0
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(backups) {
		env.Trace.Event("prompt.answered", "prompt", "restore.select", "kind", "invalid")
		fmt.Fprintf(stderr, "error: invalid selection %q\n", line)
		return "", 2
	}
	env.Trace.Event("prompt.answered", "prompt", "restore.select", "kind", "choice", "choice", strconv.Itoa(n))
	b := backups[n-1]
	if b.Unusable != "" {
		fmt.Fprintf(stderr, "error: backup %s is unusable: %s\n", b.ID, b.Unusable)
		return "", 1
	}
	return b.ID, -1
}

// readLine reads one line from r, trimming its trailing newline. It
// tolerates a final line with no trailing newline before EOF.
func readLine(r *bufio.Reader) (string, bool) {
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.TrimRight(line, "\r\n"), true
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd [preview|apply] [--repo PATH] [--home PATH]
       agentic-sdd version
       agentic-sdd backups [--repo PATH] [--home PATH]
       agentic-sdd logs [--home PATH]
       agentic-sdd restore [ID] [--apply] [--repo PATH] [--home PATH]
       agentic-sdd restore list|preview|apply [ID] [--repo PATH] [--home PATH]
       agentic-sdd guide [--repo PATH] [--home PATH]
       agentic-sdd help [preview|apply|backups|logs|restore]

Commands:
  preview        Show planned changes without writing files (default).
  apply          Back up changed existing skills, then install them.
  version        Print version, commit, and build date information.
  backups        List backups newest first (same as "restore list").
  logs           Show the latest apply or restore log.
  restore        Preview or restore a backup, selected by id or number.
  guide          Interactive step-by-step wizard for terminal users.

restore forms:
  restore                   Interactively list backups and pick one to preview.
  restore ID                Preview restoring backup ID.
  restore ID --apply        Restore backup ID.
  restore list               Same listing as "backups".
  restore preview [ID]       Same as "restore [ID]" without --apply.
  restore apply [ID]         Same as "restore ID --apply".
With no ID and no --apply/apply, restoring prompts for confirmation before
writing anything. Without a terminal, an ID is required.

Flags:
  --repo PATH  Repository containing skills-files (default: skills embedded in this binary).
  --home PATH  User home for skill destinations (default: your home).
  --apply      Legacy form of the apply command; cannot be combined with a command.
  --version    Legacy form of the version command.
  -h, --help  Show this help.
`)
}

func printVersion(out io.Writer) {
	fmt.Fprintf(out, "agentic-sdd %s\n", version.Version)
	fmt.Fprintf(out, "  commit:     %s\n", version.Commit)
	fmt.Fprintf(out, "  date:       %s\n", version.Date)
	fmt.Fprintf(out, "  go version: %s\n", version.GoVersion)
}
