// Command tui-spike runs the experimental read-only catalog browser.
//
// It is not wired into the agentic-sdd binary. Launching it requires an
// explicit AGENTIC_SDD_TUI=1 opt-in and a real terminal, exactly like the
// future CLI integration described in docs/tui-architecture.md.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"

	"agentic-sdd/experiments/tui"
)

func main() {
	home := flag.String("home", "", "home directory whose client folders are inspected (default: the current user home)")
	repo := flag.String("repo", "", "skills checkout to read (default: the embedded assets)")
	flag.Parse()

	stdinTTY := term.IsTerminal(os.Stdin.Fd())
	stdoutTTY := term.IsTerminal(os.Stdout.Fd())

	decision := tui.LaunchAllowed(os.Getenv, stdoutTTY)
	if !decision.Enabled {
		fmt.Fprintf(os.Stderr, "tui-spike: %s\n", decision.Reason)
		os.Exit(2)
	}
	if !stdinTTY {
		fmt.Fprintf(os.Stderr, "tui-spike: stdin is not a terminal; using the standard CLI\n")
		os.Exit(2)
	}

	resolvedHome := *home
	if resolvedHome == "" {
		current, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tui-spike: resolve home: %v\n", err)
			os.Exit(1)
		}
		resolvedHome = current
	}

	isTerm := stdinTTY && stdoutTTY
	motion := isTerm && os.Getenv("AGENTIC_SDD_REDUCED_MOTION") != "1" && os.Getenv("NO_COLOR") == ""
	ascii := os.Getenv("AGENTIC_SDD_ASCII") == "1"

	cfg := tui.TUIConfig{
		Repo:         *repo,
		Home:         resolvedHome,
		Color:        os.Getenv("NO_COLOR") == "",
		Interactive:  isTerm,
		WithOpenCode: true,
		Motion:       motion,
		ASCII:        ascii,
	}.Normalize()

	catalog, err := tui.LoadCatalog(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		os.Exit(1)
	}

	app := tui.NewApp(cfg, catalog)
	if _, err := tea.NewProgram(app).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		os.Exit(1)
	}
}
