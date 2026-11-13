package tui_test

// Integration coverage for B4/B5 in the nested TUI. Every test uses fake
// operations, fake transports and temporary homes: no real client directory,
// GitHub request or browser launch happens here.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// integrationLog is an in-memory operation transcript; it never touches disk.
type integrationLog struct{}

func (integrationLog) Write(p []byte) (int, error) { return len(p), nil }
func (integrationLog) Flush() error                { return nil }
func (integrationLog) Close() error                { return nil }

// writeCounters records how often each engine write seam ran.
type writeCounters struct {
	install int32
	restore int32
}

// integrationHost builds a fake single-write host. The apply seams return
// structured results immediately; progress reaches the model through the
// App's own message routing rather than program.Send.
func integrationHost(counters *writeCounters) *tui.OpsHost {
	return tui.NewOpsHost(tui.OpsHostOptions{
		ApplyInstall: func(req appops.InstallRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&counters.install, 1)
			return appops.ApplyResult{
				Operation:    skillsync.OperationApply,
				Outcome:      skillsync.OutcomeApplied,
				BackupID:     "20261110T120000.000000000Z",
				BackupPath:   "/tmp/fake-backup/20261110T120000.000000000Z",
				Applied:      []skillsync.ResultChange{{ID: "alpha-skill", Client: "claude", Path: "/tmp/home/.claude/skills/alpha-skill", Action: skillsync.ChangeInstall}},
				PreviewToken: token,
			}, nil
		},
		ApplyRestore: func(req appops.RestoreRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&counters.restore, 1)
			return appops.ApplyResult{
				Operation:    skillsync.OperationRestore,
				Outcome:      skillsync.OutcomeApplied,
				BackupID:     "20261110T130000.000000000Z",
				BackupPath:   "/tmp/fake-backup/20261110T130000.000000000Z",
				Applied:      []skillsync.ResultChange{{ID: "alpha-skill", Client: "claude", Path: "/tmp/home/.claude/skills/alpha-skill", Action: skillsync.ChangeRestore}},
				PreviewToken: token,
			}, nil
		},
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return integrationLog{}, "/tmp/fake-logs/write.log", nil
		},
	})
}

// fakeInstallPreview is a deterministic read-only preview with one planned
// change and a stable token.
func fakeInstallPreview(token string, calls *int32) tui.PreviewOp {
	return func(req appops.InstallRequest, out io.Writer) (appops.PreviewResult, error) {
		if calls != nil {
			atomic.AddInt32(calls, 1)
		}
		return appops.PreviewResult{
			Operation:    skillsync.OperationApply,
			Outcome:      skillsync.OutcomePreview,
			Planned:      []skillsync.ResultChange{{ID: "alpha-skill", Client: "claude", Path: "/tmp/home/.claude/skills/alpha-skill", Action: skillsync.ChangeInstall}},
			PreviewToken: token,
		}, nil
	}
}

// fakeRestoreOps returns one valid backup and a confirmable read-only plan.
func fakeRestoreOps(previewCalls *int32) tui.RestoreOps {
	return tui.RestoreOps{
		List: func(repo, home string) ([]skillsync.Backup, error) {
			return []skillsync.Backup{{
				ID:      "20260101T000000.000000000Z",
				Summary: "seed install",
			}}, nil
		},
		Preview: func(req appops.RestoreRequest) (appops.PreviewResult, error) {
			if previewCalls != nil {
				atomic.AddInt32(previewCalls, 1)
			}
			return appops.PreviewResult{
				Operation:    skillsync.OperationRestore,
				Outcome:      skillsync.OutcomePreview,
				Planned:      []skillsync.ResultChange{{ID: "alpha-skill", Client: "claude", Path: "/tmp/home/.claude/skills/alpha-skill", Action: skillsync.ChangeRestore}},
				PreviewToken: "restore-token",
			}, nil
		},
	}
}

// fakeUpdateManager counts release checks and never touches the network.
func fakeUpdateManager(calls *int32) *tui.UpdateCheckManager {
	manager := tui.NewUpdateCheckManager(
		tui.WithEnvLookup(func(string) string { return "" }),
		tui.WithCheckFunc(func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
			atomic.AddInt32(calls, 1)
			return appops.UpdateResult{State: appops.UpdateEqual, Current: "1.2.3", Latest: "1.2.3"}
		}),
	)
	manager.Enable()
	return manager
}

// fakeOpener records launches without starting a process. A desktop session is
// simulated so the trusted resolution path is exercised.
type fakeOpener struct {
	launches int32
}

func (f *fakeOpener) opener() *tui.Opener {
	return &tui.Opener{
		GOOS: "linux",
		Getenv: func(key string) string {
			if key == "DISPLAY" {
				return ":0"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/usr/bin/xdg-open", nil },
		Launch: func(program string, args []string) error {
			atomic.AddInt32(&f.launches, 1)
			return nil
		},
	}
}

// integratedApp constructs the shell with every seam faked and a temporary
// home, so no test reads a real client directory.
func integratedApp(t *testing.T, cfg tui.TUIConfig, host *tui.OpsHost, screens tui.AppScreenOptions) tui.App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	cfg = cfg.Normalize()
	catalog := testCatalog()
	catalog.Home = cfg.Home
	catalog.Repo = cfg.Repo
	if host == nil {
		host = integrationHost(&writeCounters{})
	}
	return tui.NewAppWithScreens(cfg, catalog, tui.AppOptions{Ops: host}, screens)
}

// updateApp sends one message and keeps the concrete App.
func updateApp(t *testing.T, app tui.App, msg tea.Msg) (tui.App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(tui.App)
	if !ok {
		t.Fatalf("Update returned %T, want tui.App", next)
	}
	return updated, cmd
}

// pressApp sends one key press and keeps the concrete App.
func pressApp(t *testing.T, app tui.App, msg tea.KeyPressMsg) (tui.App, tea.Cmd) {
	t.Helper()
	return updateApp(t, app, msg)
}

// drainApp executes a command, feeds its message batch back and repeats. Timers
// longer than executeCmd's budget are skipped, exactly like the package's
// existing command helper.
func drainApp(t *testing.T, app tui.App, cmd tea.Cmd) tui.App {
	t.Helper()
	for _, msg := range executeCmd(cmd) {
		if msg == nil {
			continue
		}
		next, follow := updateApp(t, app, msg)
		app = next
		if follow != nil {
			app = drainApp(t, app, follow)
		}
	}
	return app
}

// openInstallJourney walks Home → hub → clients → options → review → preview →
// confirm and returns the model at the confirmation step.
func openInstallJourney(t *testing.T, app tui.App) tui.App {
	t.Helper()
	app, _ = pressApp(t, app, runes("2")) // Install & Manage hub
	app, _ = pressApp(t, app, runes("i")) // clients
	app, _ = pressApp(t, app, key(tea.KeyEnter))
	app, _ = pressApp(t, app, key(tea.KeyEnter))
	app, cmd := pressApp(t, app, key(tea.KeyEnter)) // preview
	app = drainApp(t, app, cmd)
	app, _ = pressApp(t, app, key(tea.KeyEnter)) // preview → confirm
	return app
}

// openRestoreJourney walks Home → hub → backups → details → preview → confirm.
func openRestoreJourney(t *testing.T, app tui.App) tui.App {
	t.Helper()
	app, _ = pressApp(t, app, runes("2")) // hub
	app, cmd := pressApp(t, app, runes("b"))
	app = drainApp(t, app, cmd) // backup discovery
	app, _ = pressApp(t, app, key(tea.KeyEnter))
	app, cmd = pressApp(t, app, key(tea.KeyEnter)) // preview
	app = drainApp(t, app, cmd)
	app, _ = pressApp(t, app, key(tea.KeyEnter)) // confirm
	return app
}

// TestInstallHubNavigation covers the two-destination hub, the read-only
// command hints and the return path to navigation.
func TestInstallHubNavigation(t *testing.T) {
	counters := &writeCounters{}
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 44}, integrationHost(counters), tui.AppScreenOptions{
		InstallPreview: fakeInstallPreview("install-token", nil),
		RestoreOps:     fakeRestoreOps(nil),
	})

	app, _ = pressApp(t, app, runes("2"))
	view := app.View().Content
	for _, want := range []string{"Install & Manage", "Install or update", "Backups & restore", "asdx-core preview", "Backups directory"} {
		if !strings.Contains(view, want) {
			t.Fatalf("hub view missing %q:\n%s", want, view)
		}
	}

	// i opens the install journey; Esc cancels to the result, Esc closes home.
	app, _ = pressApp(t, app, runes("i"))
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if got := app.View().Content; !strings.Contains(got, "Install or update") {
		t.Fatalf("Esc from the cancelled journey did not return to the hub:\n%s", got)
	}

	// b opens the recovery screen and Esc returns to the hub.
	app, cmd := pressApp(t, app, runes("b"))
	app = drainApp(t, app, cmd)
	if got := app.View().Content; !strings.Contains(got, "Backups & recovery") && !strings.Contains(got, "seed install") {
		t.Fatalf("backups screen not rendered:\n%s", got)
	}
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if got := app.View().Content; !strings.Contains(got, "Install or update") {
		t.Fatalf("Esc from backups did not return to the hub:\n%s", got)
	}

	// Esc from the hub itself returns focus to navigation.
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if app.Focus() != tui.FocusNav {
		t.Fatalf("Esc from hub focus = %v, want FocusNav", app.Focus())
	}
	if counters.install != 0 || counters.restore != 0 {
		t.Fatalf("navigation wrote something: %+v", counters)
	}
}

// TestInstallPreviewConfirmAndApply covers preview → default-No → affirmative
// confirmation → one host apply, with duplicate keys unable to double-write.
func TestInstallPreviewConfirmAndApply(t *testing.T) {
	counters := &writeCounters{}
	var previewCalls int32
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(counters), tui.AppScreenOptions{
		InstallPreview: fakeInstallPreview("install-token", &previewCalls),
	})

	app = openInstallJourney(t, app)
	view := app.View().Content
	if !strings.Contains(view, "Confirm Installation") {
		t.Fatalf("not on the confirmation step:\n%s", view)
	}
	if !strings.Contains(view, "Only typing 'y' confirms") {
		t.Fatalf("confirmation does not explain default-No policy:\n%s", view)
	}

	// Enter is inert, twice.
	app, cmd := pressApp(t, app, key(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("Enter in confirmation emitted %T, want no command", cmd())
	}
	app, cmd = pressApp(t, app, key(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("second Enter in confirmation emitted %T, want no command", cmd())
	}
	if atomic.LoadInt32(&counters.install) != 0 {
		t.Fatalf("Enter started a write: %d", counters.install)
	}

	// y emits exactly the StartWriteMsg; a duplicate y while running is inert.
	app, startCmd := pressApp(t, app, runes("y"))
	if startCmd == nil {
		t.Fatal("y emitted no write request")
	}
	app, dupCmd := pressApp(t, app, runes("y"))
	if dupCmd != nil {
		t.Fatalf("duplicate y emitted %T, want no command", dupCmd())
	}
	app, dupCmd = pressApp(t, app, key(tea.KeyEnter))
	if dupCmd != nil {
		t.Fatalf("Enter while running emitted %T, want no command", dupCmd())
	}
	app, hostCmd := updateApp(t, app, startCmd())
	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("accepted write has no status")
	}
	app, _ = updateApp(t, app, tui.WriteProgressMsg{
		ID:     status.ID,
		Kind:   tui.WriteInstall,
		Step:   tui.WriteStepWriting,
		Detail: "apply --clients claude",
	})
	if view := app.View().Content; !strings.Contains(view, "phase: writing") {
		t.Fatalf("running screen does not show the host phase:\n%s", view)
	}
	app = drainApp(t, app, hostCmd)

	if got := atomic.LoadInt32(&counters.install); got != 1 {
		t.Fatalf("install applies = %d, want exactly 1", got)
	}
	view = app.View().Content
	if !strings.Contains(view, "Installation Applied Successfully") {
		t.Fatalf("result view missing applied title:\n%s", view)
	}
	if !strings.Contains(view, "20261110T120000.000000000Z") {
		t.Fatalf("result view missing backup identity:\n%s", view)
	}
	if !strings.Contains(view, "/tmp/fake-logs/write.log") {
		t.Fatalf("result view missing operation log path:\n%s", view)
	}
	if app.Busy() {
		t.Fatal("write stayed busy after its result")
	}
}

// TestRestoreJourneyAppliesThroughHost covers list → details → preview →
// independent confirmation → one host restore with honest result evidence.
func TestRestoreJourneyAppliesThroughHost(t *testing.T) {
	counters := &writeCounters{}
	var previewCalls int32
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(counters), tui.AppScreenOptions{
		RestoreOps: fakeRestoreOps(&previewCalls),
	})

	app = openRestoreJourney(t, app)
	view := app.View().Content
	if !strings.Contains(view, "Confirm restore") {
		t.Fatalf("not on restore confirmation:\n%s", view)
	}
	if !strings.Contains(view, "Enter does not confirm") {
		t.Fatalf("restore confirmation is not default-No:\n%s", view)
	}

	app, cmd := pressApp(t, app, key(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("Enter confirmed a restore: %T", cmd())
	}
	if atomic.LoadInt32(&counters.restore) != 0 {
		t.Fatal("Enter started a restore write")
	}

	app, cmd = pressApp(t, app, runes("y")) // revalidate, then StartWriteMsg
	app = drainApp(t, app, cmd)

	if got := atomic.LoadInt32(&counters.restore); got != 1 {
		t.Fatalf("restore applies = %d, want exactly 1", got)
	}
	if app.Busy() {
		t.Fatal("restore remained busy after the result")
	}
	view = app.View().Content
	if !strings.Contains(view, "Restore applied") {
		t.Fatalf("restore result missing applied title:\n%s", view)
	}
	if !strings.Contains(view, "20261110T130000.000000000Z") {
		t.Fatalf("restore result missing pre-restore backup:\n%s", view)
	}
	if !strings.Contains(view, "/tmp/fake-logs/write.log") {
		t.Fatalf("restore result missing operation log:\n%s", view)
	}
}

// TestSingleWriteRefusalAcrossScreens proves a second screen cannot start a
// transaction while one is active, and a refusal cannot clobber the running
// screen's state.
func TestSingleWriteRefusalAcrossScreens(t *testing.T) {
	counters := &writeCounters{}
	home := t.TempDir()
	app := integratedApp(t, tui.TUIConfig{Home: home, Width: 100, Height: 32}, integrationHost(counters), tui.AppScreenOptions{
		InstallPreview: fakeInstallPreview("install-token", nil),
		RestoreOps:     fakeRestoreOps(nil),
	})

	app = openInstallJourney(t, app)
	app, startCmd := pressApp(t, app, runes("y"))
	startMsg, ok := startCmd().(tui.StartWriteMsg)
	if !ok {
		t.Fatalf("confirm command returned %T, want StartWriteMsg", startCmd())
	}
	app, hostCmd := updateApp(t, app, startMsg)
	if !app.Busy() {
		t.Fatal("accepted write is not busy")
	}

	restoreReq, err := appops.NewRestoreRequest("", home, "20260101T000000.000000000Z", true)
	if err != nil {
		t.Fatal(err)
	}
	app, refusedCmd := updateApp(t, app, tui.StartWriteMsg{Request: tui.WriteRequest{
		Kind:    tui.WriteRestore,
		Restore: restoreReq,
		Token:   "restore-token",
		Label:   "second screen restore",
	}})
	msgs := executeCmd(refusedCmd)
	if len(msgs) != 1 {
		t.Fatalf("refusal produced %d messages, want 1", len(msgs))
	}
	refusal, ok := msgs[0].(tui.WriteRefusedMsg)
	if !ok {
		t.Fatalf("refusal message = %T, want WriteRefusedMsg", msgs[0])
	}
	if refusal.Kind != tui.WriteRestore {
		t.Fatalf("refusal kind = %v, want WriteRestore", refusal.Kind)
	}
	app, _ = updateApp(t, app, refusal)
	if !app.Busy() {
		t.Fatal("refusal cleared the running install transaction")
	}
	if got := atomic.LoadInt32(&counters.install); got != 0 {
		t.Fatalf("install apply ran before the host command: %d", got)
	}

	// Finish the accepted install write; exactly one engine call happens.
	app = drainApp(t, app, hostCmd)
	if got := atomic.LoadInt32(&counters.install); got != 1 {
		t.Fatalf("install applies = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&counters.restore); got != 0 {
		t.Fatalf("refused restore ran %d times, want 0", got)
	}
}

// TestCtrlCDeferredDuringWrite covers exit-after-safe-completion: Ctrl+C while
// a write is in flight records intent without quitting, then exits once the
// host result (already flush-complete) arrives.
func TestCtrlCDeferredDuringWrite(t *testing.T) {
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 30}, integrationHost(&writeCounters{}), tui.AppScreenOptions{
		InstallPreview: fakeInstallPreview("install-token", nil),
	})

	app = openInstallJourney(t, app)
	app, startCmd := pressApp(t, app, runes("y"))
	startMsg := startCmd().(tui.StartWriteMsg)
	app, hostCmd := updateApp(t, app, startMsg)

	app, quitCmd := pressApp(t, app, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quitCmd != nil {
		t.Fatalf("Ctrl+C during a write returned %T, want a deferred request", quitCmd())
	}
	if !app.ExitPending() {
		t.Fatal("Ctrl+C did not record a deferred exit")
	}
	if !app.Busy() {
		t.Fatal("deferred exit cleared the write early")
	}
	if reason, _ := app.ExitRequest(); reason != tui.ShutdownReasonInterruptKey {
		t.Fatalf("exit reason = %v, want Ctrl+C", reason)
	}

	// q and Esc during the write stay ignored.
	app, cmd := pressApp(t, app, runes("q"))
	if cmd != nil {
		t.Fatalf("q during a write returned %T, want no command", cmd())
	}
	app, cmd = pressApp(t, app, key(tea.KeyEscape))
	if cmd != nil {
		t.Fatalf("Esc during a write returned %T, want no command", cmd())
	}
	// Navigation keys cannot leave the owning screen either.
	app, cmd = pressApp(t, app, runes("3"))
	if cmd != nil {
		t.Fatalf("route jump during a write returned %T, want no command", cmd())
	}
	if app.ActiveRoute() != tui.RouteInstall {
		t.Fatalf("route jump during a write moved to %v", app.ActiveRoute())
	}

	msgs := executeCmd(hostCmd)
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		var follow tea.Cmd
		app, follow = updateApp(t, app, msg)
		if follow != nil {
			if quit := executeCmd(follow); len(quit) > 0 {
				if _, ok := quit[0].(tea.QuitMsg); ok {
					return // the deferred exit ran after the result
				}
			}
		}
	}
	t.Fatalf("write result did not trigger the deferred exit; exitPending=%v", app.ExitPending())
}

// TestRefusalKeepsConfirmationToken proves a host refusal never started a
// transaction: both screens return to their independent confirmation with the
// same preview token so the user can retry without re-previewing.
func TestRefusalKeepsConfirmationToken(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(&writeCounters{}), tui.AppScreenOptions{
			InstallPreview: fakeInstallPreview("install-token", nil),
		})
		app = openInstallJourney(t, app)
		app, cmd := pressApp(t, app, runes("y"))
		first := cmd().(tui.StartWriteMsg)

		// The host refuses before accepting the start; the root model routes the
		// typed refusal back to the install screen.
		app, _ = updateApp(t, app, tui.WriteRefusedMsg{Kind: tui.WriteInstall, Reason: "another write is running"})
		view := app.View().Content
		if !strings.Contains(view, "Confirm Installation") {
			t.Fatalf("refusal did not return to confirmation:\n%s", view)
		}
		if !strings.Contains(view, "another write is running") {
			t.Fatalf("refusal reason not shown:\n%s", view)
		}
		app, cmd = pressApp(t, app, runes("y"))
		second := cmd().(tui.StartWriteMsg)
		if second.Request.Token != first.Request.Token || second.Request.Token == "" {
			t.Fatalf("retry token = %q, want the retained %q", second.Request.Token, first.Request.Token)
		}
	})

	t.Run("restore", func(t *testing.T) {
		app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(&writeCounters{}), tui.AppScreenOptions{
			RestoreOps: fakeRestoreOps(nil),
		})
		app = openRestoreJourney(t, app)
		app, revalidateCmd := pressApp(t, app, runes("y"))
		app, startCmd := updateApp(t, app, revalidateCmd())
		first := startCmd().(tui.StartWriteMsg)

		app, _ = updateApp(t, app, tui.WriteRefusedMsg{Kind: tui.WriteRestore, Reason: "the operation host is unavailable"})
		view := app.View().Content
		if !strings.Contains(view, "Confirm restore") {
			t.Fatalf("refusal did not return to restore confirmation:\n%s", view)
		}
		app, revalidateCmd = pressApp(t, app, runes("y"))
		app, startCmd = updateApp(t, app, revalidateCmd())
		second := startCmd().(tui.StartWriteMsg)
		if second.Request.Token == "" || second.Request.Token != first.Request.Token {
			t.Fatalf("retry token = %q, want the retained %q", second.Request.Token, first.Request.Token)
		}
	})
}

// TestNeedsPreviewReturnsToPreview covers engine drift: a refused write must
// discard the stale consent and show a fresh read-only plan.
func TestNeedsPreviewReturnsToPreview(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		var previewCalls int32
		app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(&writeCounters{}), tui.AppScreenOptions{
			InstallPreview: fakeInstallPreview("install-token", &previewCalls),
		})
		app = openInstallJourney(t, app)
		app, cmd := pressApp(t, app, runes("y"))
		start := cmd().(tui.StartWriteMsg)
		app, _ = updateApp(t, app, start)
		status, ok := app.WriteStatus()
		if !ok {
			t.Fatal("accepted write has no status")
		}
		app, _ = updateApp(t, app, tui.WriteResultMsg{
			ID:     status.ID,
			Kind:   tui.WriteInstall,
			Result: appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed},
			Err:    appops.ErrNeedsPreview,
		})
		if app.Busy() {
			t.Fatal("needs-preview kept the write busy")
		}
		view := app.View().Content
		if !strings.Contains(strings.ToLower(view), "fresh preview") {
			t.Fatalf("needs-preview did not explain the fresh preview requirement:\n%s", view)
		}
	})

	t.Run("restore", func(t *testing.T) {
		var previewCalls int32
		app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 32}, integrationHost(&writeCounters{}), tui.AppScreenOptions{
			RestoreOps: fakeRestoreOps(&previewCalls),
		})
		app = openRestoreJourney(t, app)
		app, revalidateCmd := pressApp(t, app, runes("y"))
		app, startCmd := updateApp(t, app, revalidateCmd())
		start := startCmd().(tui.StartWriteMsg)
		app, _ = updateApp(t, app, start)
		status, ok := app.WriteStatus()
		if !ok {
			t.Fatal("accepted restore has no status")
		}

		app, cmd := updateApp(t, app, tui.WriteResultMsg{
			ID:     status.ID,
			Kind:   tui.WriteRestore,
			Result: appops.ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeFailed},
			Err:    appops.ErrNeedsPreview,
		})
		app = drainApp(t, app, cmd) // the automatic fresh read-only preview
		if app.Busy() {
			t.Fatal("needs-preview kept the restore busy")
		}
		view := app.View().Content
		// The explanation is wrapped at the viewport width, so match it with
		// whitespace normalized instead of pinning one line break.
		flat := strings.Join(strings.Fields(view), " ")
		if !strings.Contains(flat, "changed after confirmation") && !strings.Contains(flat, "fresh read-only preview") {
			t.Fatalf("needs-preview did not explain the refresh:\n%s", view)
		}
	})
}

// TestUpdateCheckOncePerSessionAndControls covers the sole automatic network
// exception: one attempt, explicit retry, and the visible disable control.
func TestUpdateCheckOncePerSessionAndControls(t *testing.T) {
	var calls int32
	manager := fakeUpdateManager(&calls)
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 30}, nil, tui.AppScreenOptions{
		UpdateManager: manager,
	})

	initCmd := app.Init()
	// The automatic request is asynchronous: local Home content and the
	// "checking" status render before any result arrives.
	if manager.State() != tui.UpdateStatusChecking {
		t.Fatalf("automatic state = %v, want checking", manager.State())
	}
	for i := 0; i < 10; i++ {
		app, _ = pressApp(t, app, key(tea.KeyPgDown))
	}
	view := app.View().Content
	if !strings.Contains(view, "Client status (core skills)") {
		t.Fatalf("Home local content did not render while checking:\n%s", view)
	}
	if !strings.Contains(view, "Checking for updates (GitHub)") {
		t.Fatalf("Home did not show the checking status:\n%s", view)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("check ran before its command executed: %d", got)
	}

	app = drainApp(t, app, initCmd)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("automatic checks = %d, want 1", got)
	}
	if manager.State() != tui.UpdateStatusCurrent {
		t.Fatalf("update state = %v, want current", manager.State())
	}
	view = app.View().Content
	if !strings.Contains(view, "Update check") || !strings.Contains(view, "up to date") {
		t.Fatalf("Home missing updated status:\n%s", view)
	}

	// Leaving and re-entering Home does not repeat the request.
	app, _ = pressApp(t, app, runes("2"))
	app, _ = pressApp(t, app, runes("1"))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("re-entering Home repeated the check: %d", got)
	}
	// Resize and re-render are also inert.
	app, _ = updateApp(t, app, tea.WindowSizeMsg{Width: 90, Height: 24})
	_ = app.View()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("resize repeated the check: %d", got)
	}

	// Explicit retry works and is not blocked by the once-per-session flag.
	app, cmd := pressApp(t, app, runes("r"))
	app = drainApp(t, app, cmd)
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("retry checks = %d, want 2", got)
	}

	// The visible disable control stops the automatic request.
	app, _ = pressApp(t, app, runes("d"))
	if !manager.IsDisabled() {
		t.Fatal("d did not disable update checks")
	}
	before := atomic.LoadInt32(&calls)
	app, cmd = pressApp(t, app, runes("r"))
	if cmd != nil {
		t.Fatalf("retry while disabled emitted %T, want no command", cmd())
	}
	if got := atomic.LoadInt32(&calls); got != before {
		t.Fatalf("disabled retry performed %d checks, want 0", got-before)
	}
	if !strings.Contains(app.View().Content, "Automatic checks are off") {
		t.Fatalf("Home does not show the disabled state:\n%s", app.View().Content)
	}

	// Enabling allows an explicit retry again.
	app, _ = pressApp(t, app, runes("d"))
	if manager.IsDisabled() {
		t.Fatal("second d did not enable update checks")
	}
	app, cmd = pressApp(t, app, runes("r"))
	app = drainApp(t, app, cmd)
	if got := atomic.LoadInt32(&calls); got != before+1 {
		t.Fatalf("enabled retry checks = %d, want %d", got, before+1)
	}
}

// TestOnboardingSkipFinishAndExplicitLink covers the skippable overlay, the
// copyable router prompt and the explicit-only opener.
func TestOnboardingSkipFinishAndExplicitLink(t *testing.T) {
	opener := &fakeOpener{}
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 30}, nil, tui.AppScreenOptions{
		Opener: opener.opener(),
	})

	// Rendering, startup and navigation never open a link.
	_ = app.View()
	if got := atomic.LoadInt32(&opener.launches); got != 0 {
		t.Fatalf("render launched %d browsers", got)
	}

	app, _ = pressApp(t, app, runes("o"))
	view := app.View().Content
	if !strings.Contains(view, "Welcome") || !strings.Contains(view, "step 1 of 4") {
		t.Fatalf("o did not open the onboarding tour:\n%s", view)
	}

	// s skips immediately and writes nothing.
	app, _ = pressApp(t, app, runes("s"))
	if strings.Contains(app.View().Content, "step 1 of 4") {
		t.Fatalf("skip left the tour open:\n%s", app.View().Content)
	}
	if !strings.Contains(app.Notice(), "skipped") {
		t.Fatalf("skip notice = %q, want skipped", app.Notice())
	}
	if got := atomic.LoadInt32(&opener.launches); got != 0 {
		t.Fatalf("skip launched %d browsers", got)
	}

	// Walk to the finish step; Enter is safe there and shows the prompt.
	app, _ = pressApp(t, app, runes("o"))
	for i := 0; i < 3; i++ {
		app, _ = pressApp(t, app, key(tea.KeyEnter))
	}
	view = app.View().Content
	if !strings.Contains(view, "step 4 of 4") {
		t.Fatalf("finish step not reached:\n%s", view)
	}
	for i := 0; i < 4; i++ {
		app, _ = pressApp(t, app, key(tea.KeyPgDown))
	}
	view = app.View().Content
	flat := strings.Join(strings.Fields(view), " ")
	if !strings.Contains(flat, "agentic-sdd-router") || !strings.Contains(flat, "for my combined approval") {
		t.Fatalf("finish step missing the copyable router prompt:\n%s", view)
	}
	if got := atomic.LoadInt32(&opener.launches); got != 0 {
		t.Fatalf("finish step launched %d browsers", got)
	}

	// o is the explicit open action; the opener is called only now.
	app, cmd := pressApp(t, app, runes("o"))
	if cmd == nil {
		t.Fatal("o on the finish step emitted no link request")
	}
	app = drainApp(t, app, cmd)
	if got := atomic.LoadInt32(&opener.launches); got != 1 {
		t.Fatalf("explicit open launched %d browsers, want 1", got)
	}
	if !strings.Contains(app.Notice(), "Opened") {
		t.Fatalf("opener explanation = %q, want an Opened notice", app.Notice())
	}

	// Esc at the completed finish step closes without marking it cancelled.
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if strings.Contains(app.View().Content, "step 4 of 4") {
		t.Fatalf("Esc did not close the completed tour:\n%s", app.View().Content)
	}
}

// TestOnboardingEscCancels proves cancellation is a terminal, side-effect-free
// exit.
func TestOnboardingEscCancels(t *testing.T) {
	app := integratedApp(t, tui.TUIConfig{Width: 100, Height: 30}, nil, tui.AppScreenOptions{})
	app, _ = pressApp(t, app, runes("o"))
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if strings.Contains(app.View().Content, "step 1 of 4") {
		t.Fatalf("Esc left the tour open:\n%s", app.View().Content)
	}
	if !strings.Contains(app.Notice(), "cancelled") {
		t.Fatalf("Esc notice = %q, want cancelled", app.Notice())
	}
}

// TestActivityRendersModelAndKeepsHelp covers ActivityModel rendering, its
// bounded log empty state and the existing help overlay.
func TestActivityRendersModelAndKeepsHelp(t *testing.T) {
	home := t.TempDir()
	app := integratedApp(t, tui.TUIConfig{Home: home, Width: 100, Height: 30}, nil, tui.AppScreenOptions{})

	app, _ = pressApp(t, app, runes("4"))
	view := app.View().Content
	for _, want := range []string{"Activity & Help", "Version & Build", "No install or restore logs found", "https://github.com/nawodyaishan/agentic-sdd#readme"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Activity view missing %q:\n%s", want, view)
		}
	}
	// The TUI-only controls sit after the model's reference sections, so reach
	// them with the same page-down control the footer advertises.
	for i := 0; i < 30; i++ {
		app, _ = pressApp(t, app, key(tea.KeyPgDown))
	}
	if view = app.View().Content; !strings.Contains(view, "TUI Controls") {
		t.Fatalf("Activity view never reached TUI Controls:\n%s", view)
	}
	// Return to the top so the help overlay assertion sees the opening frame.
	for i := 0; i < 30; i++ {
		app, _ = pressApp(t, app, key(tea.KeyPgUp))
	}

	app, _ = pressApp(t, app, runes("?"))
	view = app.View().Content
	if !strings.Contains(view, "Keyboard & Navigation Reference") {
		t.Fatalf("help overlay did not open over Activity:\n%s", view)
	}
	app, _ = pressApp(t, app, key(tea.KeyEscape))
	if app.HelpOpen() {
		t.Fatal("Esc did not close the help overlay")
	}
}

// TestIntegratedLayoutsRemainBoundsSafe checks the requested breakpoints and
// short heights across the new hub, children, Activity and onboarding in plain
// ASCII mode: no row wider than the terminal, no hidden footer.
func TestIntegratedLayoutsRemainBoundsSafe(t *testing.T) {
	heights := []int{8, 12, 16, 24}
	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		for _, height := range heights {
			cfg := tui.TUIConfig{Width: width, Height: height, Color: false, ASCII: true}
			app := integratedApp(t, cfg, nil, tui.AppScreenOptions{
				InstallPreview: fakeInstallPreview("install-token", nil),
				RestoreOps:     fakeRestoreOps(nil),
			})

			check := func(label string) {
				t.Helper()
				view := app.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) != height {
					t.Fatalf("%s w=%d h=%d produced %d rows, want %d", label, width, height, len(lines), height)
				}
				for row, line := range lines {
					if got := ansi.StringWidth(line); got > width {
						t.Fatalf("%s w=%d h=%d row %d width %d > %d: %q", label, width, height, row, got, width, line)
					}
					for _, r := range line {
						if r > 127 {
							t.Fatalf("%s w=%d h=%d row %d is not ASCII: %q", label, width, height, row, line)
						}
					}
				}
				footer := strings.ToLower(lines[len(lines)-1])
				if !strings.Contains(footer, "quit") && !strings.Contains(footer, " q") {
					t.Fatalf("%s w=%d h=%d footer hidden: %q", label, width, height, footer)
				}
			}

			app, _ = pressApp(t, app, runes("2")) // hub
			check("hub")
			app, _ = pressApp(t, app, runes("i")) // install clients
			check("install")
			app, _ = pressApp(t, app, key(tea.KeyEscape)) // cancel journey → result
			app, _ = pressApp(t, app, key(tea.KeyEscape)) // result → hub
			app, cmd := pressApp(t, app, runes("b"))
			app = drainApp(t, app, cmd) // restore list
			check("backups")
			app, _ = pressApp(t, app, key(tea.KeyEscape)) // hub
			app, _ = pressApp(t, app, runes("4"))
			check("activity")
			app, _ = pressApp(t, app, runes("1"))
			app, _ = pressApp(t, app, runes("o"))
			check("onboarding")
		}
	}
}
