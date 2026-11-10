// Command tui-companion is the packaged, non-experimental terminal UI
// companion for agentic-sdd.
//
// It is a separate binary built from the nested experiments/tui module so the
// root CLI keeps its standard-library-only dependency contract. The root CLI
// starts this exact sibling binary for explicit `agentic-sdd tui` and for the
// Homebrew interactive zero-argument default; it never searches PATH or a
// shell.
//
// The companion has a direct public interface too:
//
//	agentic-sdd-tui --help
//	agentic-sdd-tui --version
//	agentic-sdd-tui --protocol-info
//
// Those three queries are non-interactive and perform no network or filesystem
// inspection. A direct interactive launch obeys the same forced-interactive and
// real-TTY gates as the CLI path.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"

	"agentic-sdd/experiments/tui/cmd/internal/tuirun"
	"agentic-sdd/internal/buildinfo"
)

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		environment{
			getenv:      os.Getenv,
			stdinTTY:    term.IsTerminal(os.Stdin.Fd()),
			stdoutTTY:   term.IsTerminal(os.Stdout.Fd()),
			userHomeDir: os.UserHomeDir,
		},
	))
}

// environment carries the process facts the command may not read itself, so
// tests exercise the same paths without a terminal or a real home.
type environment struct {
	getenv      func(string) string
	stdinTTY    bool
	stdoutTTY   bool
	userHomeDir func() (string, error)
}

func run(args []string, stdout, stderr io.Writer, env environment) int {
	if env.getenv == nil {
		env.getenv = func(string) string { return "" }
	}
	if env.userHomeDir == nil {
		env.userHomeDir = os.UserHomeDir
	}

	fs := flag.NewFlagSet("agentic-sdd-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var home, repo string
	var withOpenCode, openCodeFreeFirst, showVersion, protocolInfo bool
	fs.StringVar(&home, "home", "", "home directory whose client folders are inspected (default: the current user home)")
	fs.StringVar(&repo, "repo", "", "skills checkout to read (default: the embedded assets)")
	fs.BoolVar(&withOpenCode, "with-opencode", true, "include the optional OpenCode pack and profiles (default true)")
	fs.BoolVar(&openCodeFreeFirst, "opencode-free-first", false, "preset OpenCode profiles to Free-first models (requires --with-opencode)")
	fs.BoolVar(&showVersion, "version", false, "print version and build identity information")
	fs.BoolVar(&protocolInfo, "protocol-info", false, "print the private, machine-readable protocol and build identity response")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		printUsage(stderr)
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		printUsage(stderr)
		return 2
	}
	if showVersion && protocolInfo {
		fmt.Fprintln(stderr, "error: --version and --protocol-info are mutually exclusive")
		return 2
	}
	// Informational queries stay ahead of every terminal gate and of any home
	// resolution, so they work in scripts, pipes and tests.
	if protocolInfo {
		data, err := buildinfo.CurrentProtocolInfo(buildinfo.ProductCompanion).JSON()
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	if showVersion {
		printVersion(stdout)
		return 0
	}
	if openCodeFreeFirst && !withOpenCode {
		fmt.Fprintln(stderr, "error: --opencode-free-first requires --with-opencode")
		return 2
	}

	return tuirun.Run(tuirun.Launch{
		Program:           "agentic-sdd-tui",
		Home:              home,
		Repo:              repo,
		WithOpenCode:      withOpenCode,
		OpenCodeFreeFirst: openCodeFreeFirst,
		RequireOptIn:      false,
		StdinTTY:          env.stdinTTY,
		StdoutTTY:         env.stdoutTTY,
		Stderr:            stderr,
		Getenv:            env.getenv,
		UserHomeDir:       env.userHomeDir,
	})
}

// printVersion uses only link-time values. It reads no environment or
// filesystem state.
func printVersion(out io.Writer) {
	identity := buildinfo.CurrentIdentity()
	fmt.Fprintf(out, "%s %s\n", buildinfo.ProductCompanion, identity.Revision)
	fmt.Fprintf(out, "  commit:        %s\n", identity.Commit)
	fmt.Fprintf(out, "  source digest: %s\n", identity.SourceDigest)
	fmt.Fprintf(out, "  distribution:  %s\n", buildinfo.CurrentDistribution())
	fmt.Fprintf(out, "  protocol:      %d\n", buildinfo.ProtocolRevision)
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd-tui [--home PATH] [--repo PATH] [--with-opencode[=false]] [--opencode-free-first]
       agentic-sdd-tui --version
       agentic-sdd-tui --protocol-info

The terminal UI companion for agentic-sdd. The agentic-sdd CLI starts this
binary from the same installation; direct invocation is supported and needs a
real input and output terminal.

Options:
  --home PATH            Home directory whose client folders are inspected.
  --repo PATH            Skills checkout to read (default: embedded assets).
  --with-opencode        Include the optional OpenCode pack and profiles (default true).
  --opencode-free-first  Preset OpenCode profiles to Free-first models
                         (requires --with-opencode).
  --version              Print version and build identity information.
  --protocol-info        Print the private protocol/build identity response.
  -h, --help             Show this help.

Environment:
  NO_COLOR                     Disable styled output.
  AGENTIC_SDD_ASCII=1          Use ASCII borders and symbols.
  AGENTIC_SDD_REDUCED_MOTION=1 Disable animation.
  AGENTIC_SDD_FORCE_INTERACTIVE=1
                               Keep scripted runs on the standard CLI; the TUI
                               never opens an alternate screen.

--version and --protocol-info are non-interactive and inspect no network or
filesystem state. The TUI never falls back to a different write workflow.
`)
}
