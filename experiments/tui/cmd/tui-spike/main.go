// Command tui-spike runs the experimental terminal UI.
//
// It is not wired into the agentic-sdd binary. Launching it requires an
// explicit AGENTIC_SDD_TUI=1 opt-in and a real terminal, exactly like the
// future CLI integration described in docs/tui-architecture.md.
//
// The process owns its OS shutdown signals explicitly: Bubble Tea's default
// handler is disabled with tea.WithoutSignalHandler(), SIGINT/SIGTERM/SIGHUP
// are registered before Program.Run and forwarded as typed shutdown requests,
// and every exit path stops delivery and restores the terminal at most once.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"

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

func main() {
	os.Exit(run())
}

func run() int {
	home := flag.String("home", "", "home directory whose client folders are inspected (default: the current user home)")
	repo := flag.String("repo", "", "skills checkout to read (default: the embedded assets)")
	flag.Parse()

	stdinTTY := term.IsTerminal(os.Stdin.Fd())
	stdoutTTY := term.IsTerminal(os.Stdout.Fd())

	decision := tui.LaunchAllowed(os.Getenv, stdoutTTY)
	if !decision.Enabled {
		fmt.Fprintf(os.Stderr, "tui-spike: %s\n", decision.Reason)
		return 2
	}
	if !stdinTTY {
		fmt.Fprintf(os.Stderr, "tui-spike: stdin is not a terminal; using the standard CLI\n")
		return 2
	}

	resolvedHome := *home
	if resolvedHome == "" {
		current, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tui-spike: resolve home: %v\n", err)
			return 1
		}
		resolvedHome = current
	}

	holdWrite, err := testHoldWrite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		return 2
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
		return 1
	}

	app := tui.NewApp(cfg, catalog)
	if holdWrite > 0 {
		app, err = newLifecycleTestApp(cfg, catalog, holdWrite)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tui-spike: lifecycle test write: %v\n", err)
			return 1
		}
	}

	program := tea.NewProgram(app, tea.WithoutSignalHandler())
	app.Ops().SetProgressSink(program.Send)

	release := tui.NewTerminalRelease(program.ReleaseTerminal)
	defer func() {
		if err := release.Release(); err != nil {
			fmt.Fprintf(os.Stderr, "tui-spike: terminal release: %v\n", err)
		}
	}()
	// Register before Run so an early signal is handled by the same policy as
	// Ctrl+C instead of Bubble Tea's default quit.
	signals := tui.NotifyShutdownSignals(program.Send)
	defer signals.Stop()

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-spike: %v\n", err)
		return 1
	}
	// Bubble Tea's own Run cleanup already restored the terminal; disarm the
	// fallback so it cannot restore a second time.
	release.Disarm()
	return 0
}

// testHoldWrite reads the test-only lifecycle duration. An unset or zero value
// keeps production behavior.
func testHoldWrite() (time.Duration, error) {
	raw := os.Getenv(envTestHoldWrite)
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
