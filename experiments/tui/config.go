// Package tui is the experimental, read-only terminal UI spike for Feature 012
// batch B2.
//
// It lives in the nested module experiments/tui so the root module stays at Go
// 1.23 with zero external dependencies. The package owns presentation
// configuration and the read-only catalog adapter only: it never writes to a
// destination, never plans or performs a backup, and never imports the CLI's
// Env or Tracer. Domain state is read exclusively through
// skillsync.InspectSelection, and plan transcripts are classified with the
// pure skillsync.SummarizePlan function.
package tui

const (
	// EnvTUI is the explicit opt-in for the experimental terminal UI. Only the
	// exact value "1" enables it; anything else keeps the standard CLI.
	EnvTUI = "AGENTIC_SDD_TUI"
	// EnvForceInteractive is the automated-journey switch the CLI already uses.
	// When it is "1", scripted runs must never enter an alternate-screen UI.
	EnvForceInteractive = "AGENTIC_SDD_FORCE_INTERACTIVE"

	// DefaultWidth and DefaultHeight are the assumed viewport size in cells
	// before the terminal reports its real size.
	DefaultWidth  = 80
	DefaultHeight = 24
)

// TUIConfig is the complete presentation configuration of one TUI run. It is
// deliberately independent of the CLI's Env type: the TUI build does not share
// the command package, so it cannot reach flags, tracers or prompts by
// accident.
//
// Every field is data. Nothing here resolves paths, creates directories or
// touches a managed destination; those decisions stay in the CLI and in
// internal/skillsync.
type TUIConfig struct {
	// Repo is the checkout to read skills from. Empty selects the assets
	// embedded in the binary, matching the CLI's default.
	Repo string
	// Home is the directory whose client skill folders are inspected. It is
	// required: there is no sensible default that cannot inspect the wrong
	// home.
	Home string
	// Width and Height are the initial viewport size in terminal cells.
	Width  int
	Height int
	// Color enables styled output. False renders plain text only, which is what
	// headless runs and tests use.
	Color bool
	// Interactive enables terminal features that only make sense on a real
	// terminal, such as the alternate screen buffer.
	Interactive bool
	// WithOpenCode includes the optional OpenCode pack skills and profiles.
	WithOpenCode bool
	// OpenCodeFreeFirst includes the Free-first preset profiles. It requires
	// WithOpenCode, mirroring the CLI flag.
	OpenCodeFreeFirst bool
}

// Normalize returns a copy with default dimensions applied. Non-positive
// dimensions are replaced, never clamped to a one-cell viewport.
func (c TUIConfig) Normalize() TUIConfig {
	if c.Width <= 0 {
		c.Width = DefaultWidth
	}
	if c.Height <= 0 {
		c.Height = DefaultHeight
	}
	return c
}

// LaunchDecision records whether the TUI may run and, when it may not, the
// reason to print. A disabled decision is never an error: it means the caller
// should use the standard CLI instead.
type LaunchDecision struct {
	Enabled bool
	Reason  string
}

// LaunchAllowed decides whether the experimental TUI may replace the standard
// CLI for this run. The rules are deliberately strict and ordered:
//
//  1. AGENTIC_SDD_TUI=1 must be set explicitly. The TUI is opt-in only.
//  2. AGENTIC_SDD_FORCE_INTERACTIVE=1 must not be set. The CLI treats that
//     switch as a fake terminal, so checking it before stdoutTTY prevents
//     automated journeys from entering an alternate screen and hanging.
//  3. stdout must be a real terminal.
//
// getenv is injected so tests never mutate the process environment.
func LaunchAllowed(getenv func(string) string, stdoutTTY bool) LaunchDecision {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if getenv(EnvTUI) != "1" {
		return LaunchDecision{Reason: EnvTUI + "=1 is not set; using the standard CLI"}
	}
	if getenv(EnvForceInteractive) == "1" {
		return LaunchDecision{Reason: EnvForceInteractive + "=1 keeps scripted runs on the standard CLI"}
	}
	if !stdoutTTY {
		return LaunchDecision{Reason: "stdout is not a terminal; using the standard CLI"}
	}
	return LaunchDecision{Enabled: true}
}
