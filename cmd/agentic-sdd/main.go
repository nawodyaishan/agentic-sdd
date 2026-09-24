package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"agentic-sdd/internal/skillsync"
	"agentic-sdd/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
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
			if len(args) > 2 || (len(args) == 2 && args[1] != "preview" && args[1] != "apply") {
				fmt.Fprintln(stderr, "error: usage: agentic-sdd help [preview|apply]")
				return 2
			}
			printUsage(stdout)
			return 0
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
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	}
	if err := skillsync.Sync(repo, home, command == "apply", stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd [preview|apply] [--repo PATH] [--home PATH]
       agentic-sdd version
       agentic-sdd help [preview|apply]

Commands:
  preview  Show planned changes without writing files (default).
  apply    Back up changed existing skills, then install them.
  version  Print version, commit, and build date information.

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
