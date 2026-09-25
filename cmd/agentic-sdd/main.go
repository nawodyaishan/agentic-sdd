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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin, isTerminal(os.Stdin)))
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader, interactive bool) int {
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
		case "help":
			if len(args) > 2 || (len(args) == 2 && !isHelpTopic(args[1])) {
				fmt.Fprintln(stderr, "error: usage: agentic-sdd help [preview|apply|backups|restore]")
				return 2
			}
			printUsage(stdout)
			return 0
		case "backups":
			return runBackups(args[1:], stdout, stderr)
		case "restore":
			return runRestore(args[1:], stdout, stderr, stdin, interactive)
		default:
			if !strings.HasPrefix(args[0], "-") {
				fmt.Fprintf(stderr, "error: unknown command %q\n", args[0])
				printUsage(stderr)
				return 2
			}
		}
	}

	fs := flag.NewFlagSet("agentic-sdd", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var repo, home string
	var legacyApply, showVersion bool
	fs.StringVar(&repo, "repo", "", "repository containing skills-files (default: skills embedded in this binary)")
	fs.StringVar(&home, "home", "", "user home (defaults to os.UserHomeDir)")
	fs.BoolVar(&legacyApply, "apply", false, "back up and install skills (legacy form)")
	fs.BoolVar(&showVersion, "version", false, "print version information")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		printUsage(stderr)
		return 2
	}
	if showVersion {
		printVersion(stdout)
		return 0
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
	home, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if err := skillsync.Sync(repo, home, command == "apply", stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func isHelpTopic(s string) bool {
	switch s {
	case "preview", "apply", "backups", "restore":
		return true
	}
	return false
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
func runBackups(args []string, stdout, stderr io.Writer) int {
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
func runRestore(args []string, stdout, stderr io.Writer, stdin io.Reader, interactive bool) int {
	mode := ""
	if len(args) > 0 {
		switch args[0] {
		case "list", "preview", "apply":
			mode, args = args[0], args[1:]
		}
	}
	if mode == "list" {
		return runBackups(args, stdout, stderr)
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
	if interactive {
		reader = bufio.NewReader(stdin)
	}

	interactivelySelected := false
	if !haveID {
		if !interactive {
			fmt.Fprintln(stderr, `error: restore requires a backup id; run "agentic-sdd backups" to list them`)
			return 2
		}
		backups, err := skillsync.ListBackups(repo, home)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		selected, code := selectBackup(backups, reader, stdout, stderr)
		if code >= 0 {
			return code
		}
		id, interactivelySelected = selected, true
	} else if err := skillsync.ValidateBackupID(id); err != nil {
		fmt.Fprintln(stderr, "error:", err)
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
		fmt.Fprint(stdout, "Restore these skills? [y/N]: ")
		line, _ := readLine(reader)
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			fmt.Fprintln(stdout, "Cancelled.")
			return 0
		}
	}

	if err := skillsync.Restore(repo, home, id, applyRestore, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

// selectBackup lists backups and reads a numbered choice from reader. The
// returned int is -1 when id holds a valid selection to proceed with, or
// the exit code to return immediately (0 to cancel, 2 for a bad answer, 1
// if the chosen backup turns out to be unusable).
func selectBackup(backups []skillsync.Backup, reader *bufio.Reader, stdout, stderr io.Writer) (string, int) {
	if len(backups) == 0 {
		fmt.Fprintln(stdout, "No backups found.")
		return "", 0
	}
	printBackups(stdout, backups)
	fmt.Fprintf(stdout, "Select a backup [1-%d] (blank to cancel): ", len(backups))
	line, _ := readLine(reader)
	line = strings.TrimSpace(line)
	if line == "" {
		fmt.Fprintln(stdout, "Cancelled.")
		return "", 0
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(backups) {
		fmt.Fprintf(stderr, "error: invalid selection %q\n", line)
		return "", 2
	}
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
       agentic-sdd restore [ID] [--apply] [--repo PATH] [--home PATH]
       agentic-sdd restore list|preview|apply [ID] [--repo PATH] [--home PATH]
       agentic-sdd help [preview|apply|backups|restore]

Commands:
  preview        Show planned changes without writing files (default).
  apply          Back up changed existing skills, then install them.
  version        Print version, commit, and build date information.
  backups        List backups newest first (same as "restore list").
  restore        Preview or restore a backup, selected by id or number.

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
