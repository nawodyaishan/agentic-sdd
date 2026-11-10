// Package tuirun is the single interactive launch path shared by the
// experimental tui-spike command and the packaged tui-companion command.
//
// Both commands only parse their public flags and inject terminal/environment
// facts; business behavior, catalog loading and the operation host stay in the
// nested tui package and the root engine.
package tuirun

import (
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// envTestHoldWrite is a test-only lifecycle seam. When set to a positive
// duration, the binary runs a synthetic in-memory write for that long before
// becoming idle. It never opens a destination or a real log; the Python PTY
// harness uses it to prove deferred exit and terminal cleanup while a write is
// genuinely in flight. It has no effect on normal launches.
const envTestHoldWrite = "AGENTIC_SDD_TUI_TEST_HOLD_WRITE"

// Launch is one fully parsed launch request. The command owns flag parsing;
// this package owns the terminal gates and the UI start-up order.
type Launch struct {
	// Program is the public command name used in diagnostics.
	Program string
	// Home and Repo are the parsed public options. Empty Home resolves through
	// UserHomeDir; empty Repo keeps the embedded assets.
	Home string
	Repo string
	// WithOpenCode and OpenCodeFreeFirst mirror the catalog pack/preset options.
	WithOpenCode      bool
	OpenCodeFreeFirst bool
	// RequireOptIn keeps the legacy tui-spike contract: AGENTIC_SDD_TUI=1 must
	// be explicit before the alternate screen may open. The packaged companion
	// is explicit by construction and passes false.
	RequireOptIn bool
	// StdinTTY and StdoutTTY are real terminal checks made by the command, never
	// forced-terminal flags.
	StdinTTY  bool
	StdoutTTY bool
	// Stderr receives launch diagnostics and failures.
	Stderr io.Writer
	// Getenv is injected so tests never mutate the process environment.
	Getenv func(string) string
	// UserHomeDir resolves the default home only on the interactive path.
	UserHomeDir func() (string, error)
}

// Run applies the launch gates and, when they pass, runs exactly one TUI
// program. It returns the command exit code.
func Run(launch Launch) int {
	if launch.Program == "" {
		launch.Program = "agentic-sdd-tui"
	}
	if launch.Stderr == nil {
		launch.Stderr = io.Discard
	}
	if launch.Getenv == nil {
		launch.Getenv = func(string) string { return "" }
	}
	if launch.UserHomeDir == nil {
		launch.UserHomeDir = os.UserHomeDir
	}

	// Gate order is deliberate: the legacy opt-in path must reject a forced
	// interactive run before a real terminal check, so automated journeys can
	// never enter an alternate screen. The companion path keeps the same
	// safety rule without requiring the legacy opt-in.
	if launch.RequireOptIn {
		decision := tui.LaunchAllowed(launch.Getenv, launch.StdoutTTY)
		if !decision.Enabled {
			fmt.Fprintf(launch.Stderr, "%s: %s\n", launch.Program, decision.Reason)
			return 2
		}
	} else if launch.Getenv(tui.EnvForceInteractive) == "1" {
		fmt.Fprintf(launch.Stderr, "%s: %s=1 keeps scripted runs on the standard CLI\n", launch.Program, tui.EnvForceInteractive)
		return 2
	}
	if !launch.StdinTTY {
		fmt.Fprintf(launch.Stderr, "%s: stdin is not a terminal; using the standard CLI\n", launch.Program)
		return 2
	}
	if !launch.StdoutTTY {
		fmt.Fprintf(launch.Stderr, "%s: stdout is not a terminal; using the standard CLI\n", launch.Program)
		return 2
	}

	resolvedHome := launch.Home
	if resolvedHome == "" {
		current, err := launch.UserHomeDir()
		if err != nil {
			fmt.Fprintf(launch.Stderr, "%s: resolve home: %v\n", launch.Program, err)
			return 1
		}
		resolvedHome = current
	}

	holdWrite, err := testHoldWrite(launch.Getenv)
	if err != nil {
		fmt.Fprintf(launch.Stderr, "%s: %v\n", launch.Program, err)
		return 2
	}

	isTerm := launch.StdinTTY && launch.StdoutTTY
	motion := isTerm && launch.Getenv("AGENTIC_SDD_REDUCED_MOTION") != "1" && launch.Getenv("NO_COLOR") == ""
	ascii := launch.Getenv("AGENTIC_SDD_ASCII") == "1"

	cfg := tui.TUIConfig{
		Repo:              launch.Repo,
		Home:              resolvedHome,
		Color:             launch.Getenv("NO_COLOR") == "",
		Interactive:       isTerm,
		WithOpenCode:      launch.WithOpenCode,
		OpenCodeFreeFirst: launch.OpenCodeFreeFirst,
		Motion:            motion,
		ASCII:             ascii,
	}.Normalize()

	catalog, err := tui.LoadCatalog(cfg)
	if err != nil {
		fmt.Fprintf(launch.Stderr, "%s: %v\n", launch.Program, err)
		return 1
	}

	app := tui.NewApp(cfg, catalog)
	if holdWrite > 0 {
		app, err = newLifecycleTestApp(cfg, catalog, holdWrite)
		if err != nil {
			fmt.Fprintf(launch.Stderr, "%s: lifecycle test write: %v\n", launch.Program, err)
			return 1
		}
	}

	program := tea.NewProgram(app, tea.WithoutSignalHandler())
	app.Ops().SetProgressSink(program.Send)

	release := tui.NewTerminalRelease(program.ReleaseTerminal)
	defer func() {
		if err := release.Release(); err != nil {
			fmt.Fprintf(launch.Stderr, "%s: terminal release: %v\n", launch.Program, err)
		}
	}()
	// Register before Run so an early signal is handled by the same policy as
	// Ctrl+C instead of Bubble Tea's default quit.
	signals := tui.NotifyShutdownSignals(program.Send)
	defer signals.Stop()

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(launch.Stderr, "%s: %v\n", launch.Program, err)
		return 1
	}
	// Bubble Tea's own Run cleanup already restored the terminal; disarm the
	// fallback so it cannot restore a second time.
	release.Disarm()
	return 0
}

// testHoldWrite reads the test-only lifecycle duration. An unset or zero value
// keeps production behavior.
func testHoldWrite(getenv func(string) string) (time.Duration, error) {
	raw := getenv(envTestHoldWrite)
	if raw == "" {
		return 0, nil
	}
	hold, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 2s: %w", envTestHoldWrite, err)
	}
	if hold < 0 {
		return 0, fmt.Errorf("%s must not be negative", envTestHoldWrite)
	}
	return hold, nil
}

// newLifecycleTestApp builds a synthetic write that holds the single write
// slot without touching a home, destination or real log file. This is the
// deterministic in-flight state the PTY signal cases need; the production path
// never calls it.
func newLifecycleTestApp(cfg tui.TUIConfig, catalog tui.Catalog, hold time.Duration) (tui.App, error) {
	request, err := appops.NewInstallRequest("", cfg.Home, true, skillsync.Options{})
	if err != nil {
		return tui.App{}, err
	}
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return discardWriteLog{}, "discarded (lifecycle test)", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			time.Sleep(hold)
			return appops.ApplyResult{
				Operation: skillsync.OperationApply,
				Outcome:   skillsync.OutcomeNoChange,
			}, nil
		},
	})
	return tui.NewAppWithOptions(cfg, catalog, tui.AppOptions{
		Ops: host,
		InitialWrite: &tui.WriteRequest{
			Kind:    tui.WriteInstall,
			Install: request,
			Token:   "lifecycle-test-token",
			Command: "apply (lifecycle test)",
			Label:   "synthetic lifecycle write",
		},
	}), nil
}

// discardWriteLog keeps the synthetic write's transcript out of the filesystem.
type discardWriteLog struct{}

func (discardWriteLog) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriteLog) Flush() error                { return nil }
func (discardWriteLog) Close() error                { return nil }
