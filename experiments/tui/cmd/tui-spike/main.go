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

	"agentic-sdd/experiments/tui"
	"agentic-sdd/experiments/tui/screens"
)

func main() {
	home := flag.String("home", "", "home directory whose client folders are inspected (default: the current user home)")
	repo := flag.String("repo", "", "skills checkout to read (default: the embedded assets)")
	flag.Parse()

	decision := tui.LaunchAllowed(os.Getenv, isTerminal(os.Stdout))
	if !decision.Enabled {
		fmt.Fprintf(os.Stderr, "tui-spike: %s\n", decision.Reason)
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

	cfg := tui.TUIConfig{
		Repo:         *repo,
		Home:         resolvedHome,
		Color:        os.Getenv("NO_COLOR") == "",
		Interactive:  true,
		WithOpenCode: true,
	}.Normalize()

	catalog, err := tui.LoadCatalog(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(screens.NewCatalog(cfg, catalog)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		os.Exit(1)
	}
}

// isTerminal mirrors the CLI's detection closely enough for the spike: a
// character device that is not the null device.
func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if nullInfo, err := os.Stat(os.DevNull); err == nil && os.SameFile(info, nullInfo) {
		return false
	}
	return true
}
