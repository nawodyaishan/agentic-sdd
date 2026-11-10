// Command tui-spike runs the experimental terminal UI.
//
// It is not wired into the agentic-sdd binary. Launching it requires an
// explicit AGENTIC_SDD_TUI=1 opt-in and a real terminal, exactly like the
// future CLI integration described in docs/tui-architecture.md.
//
// The interactive start-up, signal ownership and terminal-release policy live
// in cmd/internal/tuirun, which the packaged tui-companion command shares. This
// command only parses the legacy experimental flags and supplies the real
// terminal facts.
package main

import (
	"flag"
	"os"

	"github.com/charmbracelet/x/term"

	"agentic-sdd/experiments/tui/cmd/internal/tuirun"
)

func main() {
	os.Exit(run())
}

func run() int {
	home := flag.String("home", "", "home directory whose client folders are inspected (default: the current user home)")
	repo := flag.String("repo", "", "skills checkout to read (default: the embedded assets)")
	flag.Parse()

	return tuirun.Run(tuirun.Launch{
		Program:      "tui-spike",
		Home:         *home,
		Repo:         *repo,
		WithOpenCode: true,
		RequireOptIn: true,
		StdinTTY:     term.IsTerminal(os.Stdin.Fd()),
		StdoutTTY:    term.IsTerminal(os.Stdout.Fd()),
		Stderr:       os.Stderr,
		Getenv:       os.Getenv,
		UserHomeDir:  os.UserHomeDir,
	})
}
