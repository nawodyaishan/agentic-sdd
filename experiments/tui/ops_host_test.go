package tui_test

// Deterministic tests for the nested TUI's single-write operation host and the
// root model's busy/exit policy. Every fake operation runs without touching a
// real home: the engine seam is replaced, and the fake log lives in memory.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// recordingLog is an in-memory WriteLog that records the host's lifecycle.
type recordingLog struct {
	mu       sync.Mutex
	content  strings.Builder
	flushed  bool
	closed   bool
	flushErr error
	closeErr error
}

func (l *recordingLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.content.Write(p)
}

func (l *recordingLog) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushed = true
	return l.flushErr
}

func (l *recordingLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return l.closeErr
}

func (l *recordingLog) transcript() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.content.String()
}

func (l *recordingLog) lifecycle() (flushed, closed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flushed, l.closed
}

func testInstallRequest(t *testing.T) appops.InstallRequest {
	t.Helper()
	request, err := appops.NewInstallRequest("", t.TempDir(), true, skillsync.Options{})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	return request
}

func testWriteRequest(t *testing.T) tui.WriteRequest {
	t.Helper()
	return tui.WriteRequest{
		Kind:    tui.WriteInstall,
		Install: testInstallRequest(t),
		Token:   "preview-token",
		Command: "apply --with-opencode",
		Label:   "Install 2 items",
	}
}

func TestOpsHostRunsOnlyWhenCommandRunsAndCompletesOnce(t *testing.T) {
	log := &recordingLog{}
	var calls int32
	var wrote io.Writer
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return log, "/tmp/operation.log", nil
		},
		ApplyInstall: func(_ appops.InstallRequest, token string, out io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&calls, 1)
			if token != "preview-token" {
				t.Errorf("token = %q, want the confirmed preview token", token)
			}
			wrote = out
			return appops.ApplyResult{
				Operation: skillsync.OperationApply,
				Outcome:   skillsync.OutcomeApplied,
				Applied:   []skillsync.ResultChange{{Client: "claude", ID: "alpha-skill", Action: skillsync.ChangeInstall}},
			}, nil
		},
	})

	var progress []tui.WriteProgressMsg
	host.SetProgressSink(func(msg tea.Msg) {
		if update, ok := msg.(tui.WriteProgressMsg); ok {
			progress = append(progress, update)
		}
	})

	started, cmd, err := host.BeginWrite(testWriteRequest(t))
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	if started.ID.Request == 0 || started.ID.Generation == 0 {
		t.Fatalf("started ID = %+v, want a non-zero request and generation", started.ID)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("BeginWrite performed the operation inside Update: calls = %d", got)
	}
	if _, ok := host.ActiveWrite(); !ok {
		t.Fatal("BeginWrite did not claim the write slot")
	}
	if cmd == nil {
		t.Fatal("BeginWrite returned no command")
	}

	msg, ok := cmd().(tui.WriteResultMsg)
	if !ok {
		t.Fatalf("result type = %T, want tui.WriteResultMsg", msg)
	}
	if msg.ID != started.ID {
		t.Fatalf("result ID = %+v, want %+v", msg.ID, started.ID)
	}
	if msg.Err != nil {
		t.Fatalf("result error = %v", msg.Err)
	}
	if msg.Result.Outcome != skillsync.OutcomeApplied {
		t.Fatalf("outcome = %q, want applied", msg.Result.Outcome)
	}
	if msg.LogPath != "/tmp/operation.log" {
		t.Fatalf("LogPath = %q", msg.LogPath)
	}
	if wrote == nil {
		t.Fatal("the engine did not receive the operation log writer")
	}
	flushed, closed := log.lifecycle()
	if !flushed || !closed {
		t.Fatalf("log flushed=%t closed=%t, want both true before the result", flushed, closed)
	}
	transcript := log.transcript()
	for _, want := range []string{"Started:", "Command: apply --with-opencode", "Result: success", "Finished:"} {
		if !strings.Contains(transcript, want) {
			t.Fatalf("transcript missing %q:\n%s", want, transcript)
		}
	}

	var steps []tui.WriteStep
	for _, update := range progress {
		if update.ID != started.ID {
			t.Fatalf("progress ID = %+v, want %+v", update.ID, started.ID)
		}
		if strings.Contains(update.Detail, "%") {
			t.Fatalf("progress Detail %q invents a percentage", update.Detail)
		}
		steps = append(steps, update.Step)
	}
	wantSteps := []tui.WriteStep{tui.WriteStepStarting, tui.WriteStepWriting, tui.WriteStepFlushing}
	if len(steps) != len(wantSteps) {
		t.Fatalf("progress steps = %v, want %v", steps, wantSteps)
	}
	for i := range wantSteps {
		if steps[i] != wantSteps[i] {
			t.Fatalf("progress steps = %v, want %v", steps, wantSteps)
		}
	}

	if !host.Complete(msg.ID) {
		t.Fatal("Complete rejected the current result identity")
	}
	if host.Complete(msg.ID) {
		t.Fatal("Complete accepted the same result twice")
	}
	if _, ok := host.ActiveWrite(); ok {
		t.Fatal("the write slot stayed claimed after Complete")
	}
}

func TestOpsHostRefusesSecondConcurrentWriteWithoutQueueing(t *testing.T) {
	var calls int32
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return &recordingLog{}, "", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&calls, 1)
			return appops.ApplyResult{Outcome: skillsync.OutcomeNoChange}, nil
		},
	})

	first, _, err := host.BeginWrite(testWriteRequest(t))
	if err != nil {
		t.Fatalf("first BeginWrite: %v", err)
	}
	second, secondCmd, err := host.BeginWrite(testWriteRequest(t))
	if !errors.Is(err, tui.ErrWriteInProgress) {
		t.Fatalf("second BeginWrite error = %v, want ErrWriteInProgress", err)
	}
	if secondCmd != nil {
		t.Fatal("a refused write must not return a queued command")
	}
	if second.ID != (appops.OperationID{}) {
		t.Fatalf("a refused write received identity %+v", second.ID)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("refused write ran the engine %d times", got)
	}

	if !host.Complete(first.ID) {
		t.Fatal("Complete rejected the active write")
	}
	third, _, err := host.BeginWrite(testWriteRequest(t))
	if err != nil {
		t.Fatalf("BeginWrite after completion: %v", err)
	}
	if third.ID.Request != first.ID.Request+1 {
		t.Fatalf("next request = %d, want %d", third.ID.Request, first.ID.Request+1)
	}
	if third.ID.Generation != 1 {
		t.Fatalf("next generation = %d, want 1", third.ID.Generation)
	}

	completed, ok := host.Completed()
	if !ok || completed != first.ID {
		t.Fatalf("Completed() = %+v, %t, want %+v", completed, ok, first.ID)
	}
}

func TestOpsHostRejectsInvalidRequestsBeforeClaimingSlot(t *testing.T) {
	host := tui.NewOpsHost(tui.OpsHostOptions{})
	request := testWriteRequest(t)

	cases := []struct {
		name string
		req  tui.WriteRequest
	}{
		{name: "unknown kind", req: tui.WriteRequest{Kind: tui.WriteKind(99), Token: "t"}},
		{name: "missing token", req: tui.WriteRequest{Kind: tui.WriteInstall, Install: request.Install}},
		{name: "missing home", req: tui.WriteRequest{Kind: tui.WriteInstall, Token: "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, cmd, err := host.BeginWrite(tc.req); !errors.Is(err, tui.ErrWriteInvalid) || cmd != nil {
				t.Fatalf("BeginWrite error = %v, cmd nil=%t; want ErrWriteInvalid and no command", err, cmd == nil)
			}
			if _, ok := host.ActiveWrite(); ok {
				t.Fatal("an invalid request claimed the write slot")
			}
		})
	}
}

func TestOpsHostLogOpenFailureDoesNotWrite(t *testing.T) {
	var calls int32
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return nil, "", errors.New("cannot create log")
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&calls, 1)
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
	})
	started, cmd, err := host.BeginWrite(testWriteRequest(t))
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	msg := cmd().(tui.WriteResultMsg)
	if msg.ID != started.ID {
		t.Fatalf("result ID = %+v, want %+v", msg.ID, started.ID)
	}
	if msg.Err == nil || !strings.Contains(msg.Err.Error(), "open operation log") {
		t.Fatalf("result error = %v, want an operation-log error", msg.Err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("the engine ran %d times without a log", got)
	}
	if msg.Result.Outcome != "" {
		t.Fatalf("outcome = %q, want the zero value when nothing ran", msg.Result.Outcome)
	}
}

func TestOpsHostReportsLogFlushFailure(t *testing.T) {
	log := &recordingLog{flushErr: errors.New("disk full")}
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return log, "/tmp/operation.log", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			return appops.ApplyResult{Outcome: skillsync.OutcomeNoChange}, nil
		},
	})
	_, cmd, err := host.BeginWrite(testWriteRequest(t))
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	msg := cmd().(tui.WriteResultMsg)
	if msg.LogErr == nil {
		t.Fatal("LogErr = nil, want the flush failure")
	}
	if !strings.Contains(msg.LogErr.Error(), "disk full") || !strings.Contains(msg.LogErr.Error(), "/tmp/operation.log") {
		t.Fatalf("LogErr = %v, want the path and cause", msg.LogErr)
	}
	summary := tui.SummarizeWriteResult(msg)
	if !summary.LogFailed || !strings.Contains(summary.Detail, "could not be flushed") {
		t.Fatalf("summary = %+v, want a visible log-flush failure", summary)
	}
	if summary.Headline != "no changes" {
		t.Fatalf("headline = %q, want no changes even when the log failed", summary.Headline)
	}
}

func TestOpsHostDefaultFileLogWritesPrivateTranscript(t *testing.T) {
	home := t.TempDir()
	request, err := appops.NewInstallRequest("", home, true, skillsync.Options{})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	host := tui.NewOpsHost(tui.OpsHostOptions{
		ApplyInstall: func(_ appops.InstallRequest, _ string, out io.Writer) (appops.ApplyResult, error) {
			_, _ = io.WriteString(out, "one planned change\n")
			return appops.ApplyResult{Outcome: skillsync.OutcomeApplied}, nil
		},
	})
	_, cmd, err := host.BeginWrite(tui.WriteRequest{
		Kind:    tui.WriteInstall,
		Install: request,
		Token:   "token",
		Command: "apply",
	})
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	msg := cmd().(tui.WriteResultMsg)
	if msg.Err != nil || msg.LogErr != nil {
		t.Fatalf("result errors = %v / %v", msg.Err, msg.LogErr)
	}
	if !strings.HasPrefix(msg.LogPath, filepath.Join(home, ".agentic-sdd", "logs")+string(os.PathSeparator)) {
		t.Fatalf("LogPath = %q, want a private log under the temporary home", msg.LogPath)
	}
	info, err := os.Stat(msg.LogPath)
	if err != nil {
		t.Fatalf("stat log: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("log mode = %#o, want 0600", mode)
	}
	data, err := os.ReadFile(msg.LogPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	for _, want := range []string{"one planned change", "Result: success", "Finished:"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("log missing %q:\n%s", want, data)
		}
	}
}

func TestOpsHostProductionEngineInstallsNoOpThenReplaces(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	skillDir := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create source skill: %v", err)
	}
	sourceFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(sourceFile, []byte("first content\n"), 0o644); err != nil {
		t.Fatalf("write source skill: %v", err)
	}

	opts := skillsync.Options{ClientIDs: []string{"claude"}}
	request, err := appops.NewInstallRequest(repo, home, true, opts)
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	host := tui.NewOpsHost(tui.OpsHostOptions{})
	target := filepath.Join(home, ".claude", "skills", "agentic-sdd-plan", "SKILL.md")

	applyThroughHost := func(wantOutcome skillsync.Outcome) tui.WriteResultMsg {
		t.Helper()
		preview, err := appops.PreviewInstall(request, io.Discard)
		if err != nil {
			t.Fatalf("PreviewInstall: %v", err)
		}
		started, cmd, err := host.BeginWrite(tui.WriteRequest{
			Kind:    tui.WriteInstall,
			Install: request,
			Token:   preview.PreviewToken,
			Command: "apply",
		})
		if err != nil {
			t.Fatalf("BeginWrite: %v", err)
		}
		msg := cmd().(tui.WriteResultMsg)
		if msg.Err != nil || msg.LogErr != nil {
			t.Fatalf("apply errors = %v / %v", msg.Err, msg.LogErr)
		}
		if msg.Result.Outcome != wantOutcome {
			t.Fatalf("outcome = %q, want %q", msg.Result.Outcome, wantOutcome)
		}
		if !host.Complete(started.ID) {
			t.Fatal("Complete rejected the real-engine result")
		}
		return msg
	}

	first := applyThroughHost(skillsync.OutcomeApplied)
	if first.Result.BackupID == "" {
		t.Fatal("first install did not report a backup identity")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if string(data) != "first content\n" {
		t.Fatalf("installed content = %q", data)
	}

	// Reapplying an unchanged source and destination is a real no-op.
	applyThroughHost(skillsync.OutcomeNoChange)
	backups, err := os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil {
		t.Fatalf("read backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("no-op created another backup; backup count = %d, want 1", len(backups))
	}

	// A changed source produces a different token and a real replacement.
	if err := os.WriteFile(sourceFile, []byte("second content\n"), 0o644); err != nil {
		t.Fatalf("rewrite source skill: %v", err)
	}
	replaced := applyThroughHost(skillsync.OutcomeApplied)
	if replaced.Result.BackupID == "" || replaced.Result.BackupID == first.Result.BackupID {
		t.Fatalf("replacement backup = %q, want a new identity (first %q)", replaced.Result.BackupID, first.Result.BackupID)
	}
	data, err = os.ReadFile(target)
	if err != nil {
		t.Fatalf("read replaced skill: %v", err)
	}
	if string(data) != "second content\n" {
		t.Fatalf("replaced content = %q", data)
	}
	backups, err = os.ReadDir(filepath.Join(repo, "backups"))
	if err != nil {
		t.Fatalf("read backups after replacement: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("replacement backup count = %d, want 2", len(backups))
	}
}

func TestOpsHostDispatchesRestoreThroughTheSameSingleSlot(t *testing.T) {
	home := t.TempDir()
	request, err := appops.NewRestoreRequest("", home, "20260101T000000.000000000Z", true)
	if err != nil {
		t.Fatalf("NewRestoreRequest: %v", err)
	}
	log := &recordingLog{}
	var gotBackup string
	var installCalls int32
	host := tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return log, "/tmp/restore.log", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&installCalls, 1)
			return appops.ApplyResult{}, errors.New("install must not run for a restore")
		},
		ApplyRestore: func(req appops.RestoreRequest, token string, _ io.Writer) (appops.ApplyResult, error) {
			gotBackup = req.BackupID()
			if token != "restore-token" {
				t.Errorf("restore token = %q", token)
			}
			return appops.ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeNoChange}, nil
		},
	})
	started, cmd, err := host.BeginWrite(tui.WriteRequest{
		Kind:    tui.WriteRestore,
		Restore: request,
		Token:   "restore-token",
		Command: "restore 20260101T000000.000000000Z",
	})
	if err != nil {
		t.Fatalf("BeginWrite: %v", err)
	}
	if started.Kind != tui.WriteRestore {
		t.Fatalf("started kind = %v, want restore", started.Kind)
	}
	msg := cmd().(tui.WriteResultMsg)
	if msg.Err != nil || msg.Result.Operation != skillsync.OperationRestore {
		t.Fatalf("restore result = %+v / %v", msg.Result, msg.Err)
	}
	if gotBackup != request.BackupID() {
		t.Fatalf("applied backup = %q, want %q", gotBackup, request.BackupID())
	}
	if atomic.LoadInt32(&installCalls) != 0 {
		t.Fatal("install applier ran for a restore request")
	}
	if !strings.Contains(log.transcript(), "Command: restore 20260101T000000.000000000Z") {
		t.Fatalf("restore transcript missing command header:\n%s", log.transcript())
	}
}

func TestSummarizeWriteResultStates(t *testing.T) {
	cases := []struct {
		name     string
		msg      tui.WriteResultMsg
		headline string
		rollback string
	}{
		{
			name:     "applied",
			msg:      tui.WriteResultMsg{Result: appops.ApplyResult{Outcome: skillsync.OutcomeApplied, Applied: []skillsync.ResultChange{{}, {}}}},
			headline: "completed",
			rollback: "none",
		},
		{
			name:     "no change",
			msg:      tui.WriteResultMsg{Result: appops.ApplyResult{Outcome: skillsync.OutcomeNoChange}},
			headline: "no changes",
			rollback: "none",
		},
		{
			name:     "failed before backup",
			msg:      tui.WriteResultMsg{Err: errors.New("plan failed"), Result: appops.ApplyResult{Outcome: skillsync.OutcomeFailed}},
			headline: "failed",
			rollback: "none",
		},
		{
			name: "rolled back",
			msg: tui.WriteResultMsg{
				Err:    errors.New("write failed"),
				Result: appops.ApplyResult{Outcome: skillsync.OutcomeFailed, Rollback: skillsync.RollbackInfo{Attempted: true, Complete: true}},
			},
			headline: "rolled back",
			rollback: "complete",
		},
		{
			name: "incomplete rollback",
			msg: tui.WriteResultMsg{
				Err:    errors.New("write failed"),
				Result: appops.ApplyResult{Outcome: skillsync.OutcomeFailed, Rollback: skillsync.RollbackInfo{Attempted: true}},
			},
			headline: "incomplete",
			rollback: "incomplete",
		},
		{
			name:     "needs preview",
			msg:      tui.WriteResultMsg{Err: appops.ErrNeedsPreview, Result: appops.ApplyResult{Outcome: skillsync.OutcomeFailed}},
			headline: "needs a fresh preview",
			rollback: "none",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := tui.SummarizeWriteResult(tc.msg)
			if summary.Headline != tc.headline {
				t.Fatalf("Headline = %q, want %q", summary.Headline, tc.headline)
			}
			if summary.Rollback != tc.rollback {
				t.Fatalf("Rollback = %q, want %q", summary.Rollback, tc.rollback)
			}
			if tc.name == "needs preview" && !summary.NeedsPreview {
				t.Fatal("NeedsPreview = false for ErrNeedsPreview")
			}
			text := strings.ToLower(summary.Headline + " " + summary.Detail)
			if strings.Contains(text, "cancel") {
				t.Fatalf("write summary %q claims a cancellation the engine never reports", text)
			}
		})
	}
}

// writeHarness gates a fake install so tests can hold a transaction in flight
// without any filesystem work.
type writeHarness struct {
	host       *tui.OpsHost
	log        *recordingLog
	gate       chan struct{}
	started    chan struct{}
	calls      int32
	finishOnce sync.Once
}

func newWriteHarness(t *testing.T) *writeHarness {
	t.Helper()
	harness := &writeHarness{
		log:     &recordingLog{},
		gate:    make(chan struct{}),
		started: make(chan struct{}, 4),
	}
	harness.host = tui.NewOpsHost(tui.OpsHostOptions{
		OpenLog: func(tui.WriteKind, string) (tui.WriteLog, string, error) {
			return harness.log, "/tmp/gated.log", nil
		},
		ApplyInstall: func(appops.InstallRequest, string, io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&harness.calls, 1)
			harness.started <- struct{}{}
			<-harness.gate
			return appops.ApplyResult{
				Operation: skillsync.OperationApply,
				Outcome:   skillsync.OutcomeApplied,
				Applied:   []skillsync.ResultChange{{Client: "claude", ID: "alpha-skill", Action: skillsync.ChangeInstall}},
			}, nil
		},
	})
	return harness
}

func (h *writeHarness) finish() {
	h.finishOnce.Do(func() { close(h.gate) })
}

func (h *writeHarness) callCount() int32 { return atomic.LoadInt32(&h.calls) }

func testApp(t *testing.T, opts tui.AppOptions) tui.App {
	t.Helper()
	return tui.NewAppWithOptions(tui.TUIConfig{Width: 100, Height: 30}, testCatalog(), opts)
}

// startWrite feeds StartWriteMsg and returns the work command from the batch,
// without executing it. The typed start command is executed first so the test
// exercises the same message the program receives.
func startWrite(t *testing.T, app tui.App, request tui.WriteRequest) (tui.App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(tui.StartWriteMsg{Request: request})
	app = next.(tui.App)
	if cmd == nil {
		t.Fatal("StartWriteMsg returned no command")
	}
	raw := cmd()
	batch, ok := raw.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("StartWriteMsg command = %T, want a two-command batch", raw)
	}
	startMsg := batch[0]()
	started, ok := startMsg.(tui.WriteStartedMsg)
	if !ok {
		t.Fatalf("first command returned %T, want WriteStartedMsg", startMsg)
	}
	next, _ = app.Update(started)
	app = next.(tui.App)
	if !app.Busy() {
		t.Fatal("app is not busy after the typed start message")
	}
	return app, batch[1]
}

func statusID(t *testing.T, app tui.App) appops.OperationID {
	t.Helper()
	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("WriteStatus reports no active write")
	}
	return status.ID
}

func TestAppIdleQuitCancelsReadsOnce(t *testing.T) {
	var cancelled int32
	app := testApp(t, tui.AppOptions{CancelReads: func() { atomic.AddInt32(&cancelled, 1) }})

	next, cmd := app.Update(runes("q"))
	app = next.(tui.App)
	if cmd == nil {
		t.Fatal("idle q returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("idle q did not request quit")
	}
	if got := atomic.LoadInt32(&cancelled); got != 1 {
		t.Fatalf("CancelReads calls = %d, want 1", got)
	}

	// A repeated shutdown request must not cancel or quit twice.
	next, cmd = app.Update(tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGTERM"})
	app = next.(tui.App)
	if cmd != nil {
		t.Fatal("repeated idle shutdown returned another command")
	}
	if got := atomic.LoadInt32(&cancelled); got != 1 {
		t.Fatalf("CancelReads calls after repeat = %d, want 1", got)
	}
}

func TestAppIdleSignalQuitsWithTerminalCleanupPolicy(t *testing.T) {
	for _, signal := range []string{"SIGINT", "SIGTERM", "SIGHUP"} {
		t.Run(signal, func(t *testing.T) {
			app := testApp(t, tui.AppOptions{})
			next, cmd := app.Update(tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: signal})
			app = next.(tui.App)
			if cmd == nil {
				t.Fatal("idle signal returned no command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("idle %s did not request quit", signal)
			}
			if !app.ExitPending() {
				t.Fatal("ExitPending = false after an idle signal")
			}
			reason, gotSignal := app.ExitRequest()
			if reason != tui.ShutdownReasonSignal || gotSignal != signal {
				t.Fatalf("exit request = %v %q, want signal %q", reason, gotSignal, signal)
			}
		})
	}
}

func TestAppBusyIgnoresQuitEscapeAndNavigation(t *testing.T) {
	harness := newWriteHarness(t)
	app := testApp(t, tui.AppOptions{Ops: harness.host})
	app, work := startWrite(t, app, testWriteRequest(t))
	defer harness.finish()
	_ = work

	route := app.ActiveRoute()
	for _, press := range []tea.KeyPressMsg{runes("q"), key(tea.KeyEscape), runes("2"), runes("4"), runes("1"), key(tea.KeyEnter), key(tea.KeyTab)} {
		next, cmd := app.Update(press)
		app = next.(tui.App)
		if cmd != nil {
			msg := cmd()
			if _, quit := msg.(tea.QuitMsg); quit {
				t.Fatalf("key %q quit while a write was active", press.String())
			}
			if _, refused := msg.(tui.WriteRefusedMsg); refused {
				t.Fatalf("key %q started a second write", press.String())
			}
		}
	}
	if app.ActiveRoute() != route {
		t.Fatalf("navigation changed route to %v during a write", app.ActiveRoute())
	}
	if !app.Busy() {
		t.Fatal("app left the busy state after ignored keys")
	}
	if harness.callCount() != 0 {
		t.Fatalf("fake engine ran %d times without the work command", harness.callCount())
	}
	if !strings.Contains(app.Notice(), "write in progress") {
		t.Fatalf("Notice = %q, want the busy policy explanation", app.Notice())
	}
}

func TestAppCtrlCAndRepeatedSignalsDeferExitUntilResult(t *testing.T) {
	cases := []struct {
		name string
		msgs []tea.Msg
	}{
		{name: "ctrlc", msgs: []tea.Msg{tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}}},
		{name: "sigint", msgs: []tea.Msg{tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGINT"}}},
		{name: "repeated sigterm", msgs: []tea.Msg{
			tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGTERM"},
			tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGTERM"},
			tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGTERM"},
		}},
		{name: "sighup", msgs: []tea.Msg{tui.ShutdownRequestMsg{Reason: tui.ShutdownReasonSignal, Signal: "SIGHUP"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cancelled int32
			harness := newWriteHarness(t)
			app := testApp(t, tui.AppOptions{
				Ops:         harness.host,
				CancelReads: func() { atomic.AddInt32(&cancelled, 1) },
			})
			app, work := startWrite(t, app, testWriteRequest(t))

			for _, msg := range tc.msgs {
				next, cmd := app.Update(msg)
				app = next.(tui.App)
				if cmd != nil {
					if _, ok := cmd().(tea.QuitMsg); ok {
						t.Fatalf("%s quit before the write result", tc.name)
					}
				}
				if !app.Busy() {
					t.Fatalf("%s ended the write early", tc.name)
				}
				if !app.ExitPending() {
					t.Fatalf("%s did not record exit-after-completion", tc.name)
				}
			}
			if got := atomic.LoadInt32(&cancelled); got != 0 {
				t.Fatalf("read work cancelled %d times while a write ran", got)
			}

			// Run the gated fake operation until it is genuinely in flight.
			resultCh := make(chan tea.Msg, 1)
			go func() { resultCh <- work() }()
			select {
			case <-harness.started:
			case <-time.After(2 * time.Second):
				t.Fatal("fake write never started")
			}
			if got := atomic.LoadInt32(&cancelled); got != 0 {
				t.Fatalf("read work cancelled %d times during the write", got)
			}
			harness.finish()

			var result tui.WriteResultMsg
			select {
			case raw := <-resultCh:
				var ok bool
				result, ok = raw.(tui.WriteResultMsg)
				if !ok {
					t.Fatalf("work command returned %T", raw)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("fake write did not finish")
			}
			flushed, closed := harness.log.lifecycle()
			if !flushed || !closed {
				t.Fatalf("result arrived before the log flush: flushed=%t closed=%t", flushed, closed)
			}

			next, cmd := app.Update(result)
			app = next.(tui.App)
			if cmd == nil {
				t.Fatal("result after an exit request returned no command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("result after an exit request did not quit")
			}
			if app.Busy() {
				t.Fatal("app stayed busy after the result")
			}
			if got := atomic.LoadInt32(&cancelled); got != 1 {
				t.Fatalf("CancelReads calls = %d, want exactly 1 on deferred quit", got)
			}
			last, ok := app.LastWriteResult()
			if !ok || last.ID != result.ID {
				t.Fatalf("LastWriteResult = %+v, %t", last, ok)
			}
			if _, active := harness.host.ActiveWrite(); active {
				t.Fatal("host kept the write slot after the accepted result")
			}
		})
	}
}

func TestAppStaleWriteResultsAreIgnored(t *testing.T) {
	harness := newWriteHarness(t)
	app := testApp(t, tui.AppOptions{Ops: harness.host})
	app, work := startWrite(t, app, testWriteRequest(t))
	defer harness.finish()

	current := statusID(t, app)
	stale := []tui.WriteResultMsg{
		{ID: appops.OperationID{Request: current.Request, Generation: current.Generation - 1}, Kind: tui.WriteInstall},
		{ID: appops.OperationID{Request: current.Request - 1, Generation: 1}, Kind: tui.WriteInstall},
		{ID: appops.OperationID{Request: current.Request + 1, Generation: 1}, Kind: tui.WriteInstall},
	}
	for _, msg := range stale {
		next, cmd := app.Update(msg)
		app = next.(tui.App)
		if cmd != nil {
			t.Fatal("a stale result produced a command")
		}
		if !app.Busy() {
			t.Fatalf("stale result %+v cleared the active write", msg.ID)
		}
		if _, ok := app.LastWriteResult(); ok {
			t.Fatal("a stale result was stored as the last result")
		}
		if _, ok := harness.host.ActiveWrite(); !ok {
			t.Fatal("a stale result released the host write slot")
		}
	}

	resultCh := make(chan tea.Msg, 1)
	go func() { resultCh <- work() }()
	<-harness.started
	harness.finish()
	result := (<-resultCh).(tui.WriteResultMsg)
	next, _ := app.Update(result)
	app = next.(tui.App)
	if app.Busy() {
		t.Fatal("the current result did not clear the write")
	}
}

func TestAppRefusesDoubleSubmitWithoutCallingEngine(t *testing.T) {
	harness := newWriteHarness(t)
	app := testApp(t, tui.AppOptions{Ops: harness.host})
	app, work := startWrite(t, app, testWriteRequest(t))
	defer harness.finish()
	_ = work

	next, cmd := app.Update(tui.StartWriteMsg{Request: testWriteRequest(t)})
	app = next.(tui.App)
	if cmd == nil {
		t.Fatal("a duplicate StartWriteMsg returned no refusal")
	}
	raw := cmd()
	refused, ok := raw.(tui.WriteRefusedMsg)
	if !ok {
		t.Fatalf("duplicate start returned %T, want WriteRefusedMsg", raw)
	}
	if !errors.Is(refused.Err, tui.ErrWriteInProgress) {
		t.Fatalf("refusal error = %v, want ErrWriteInProgress", refused.Err)
	}
	if harness.callCount() != 0 {
		t.Fatalf("duplicate start ran the engine %d times", harness.callCount())
	}
	next, _ = app.Update(refused)
	app = next.(tui.App)
	if !app.Busy() {
		t.Fatal("duplicate start cleared the existing write")
	}
	if !strings.Contains(app.Notice(), "already running") {
		t.Fatalf("Notice = %q, want the concurrent-write explanation", app.Notice())
	}
}

func TestAppInitialWriteStartsBusyBeforeProgramRun(t *testing.T) {
	harness := newWriteHarness(t)
	request := testWriteRequest(t)
	app := tui.NewAppWithOptions(
		tui.TUIConfig{Width: 100, Height: 30},
		testCatalog(),
		tui.AppOptions{Ops: harness.host, InitialWrite: &request},
	)
	if !app.Busy() {
		t.Fatal("InitialWrite did not claim the write slot before Run")
	}
	if !strings.Contains(app.View().Content, tui.WriteBusyMarker) {
		t.Fatalf("initial view lacks the busy marker:\n%s", app.View().Content)
	}
	initCmd := app.Init()
	if initCmd == nil {
		t.Fatal("Init returned no command for the queued write")
	}
}

func TestAppBusyFooterReplacesQuitHintAndPolicyIsShownBeforeExecution(t *testing.T) {
	harness := newWriteHarness(t)
	app := testApp(t, tui.AppOptions{Ops: harness.host})
	app, work := startWrite(t, app, testWriteRequest(t))
	defer harness.finish()
	_ = work

	view := app.View().Content
	last := view[strings.LastIndex(view, "\n")+1:]
	if !strings.Contains(last, tui.WriteBusyMarker) {
		t.Fatalf("busy footer = %q, want the busy marker", last)
	}
	if !strings.Contains(last, "Ctrl+C") || !strings.Contains(last, "after completion") {
		t.Fatalf("busy footer = %q, want the exit-after-completion policy", last)
	}
	if strings.Contains(last, "q quit") {
		t.Fatalf("busy footer still offers q quit: %q", last)
	}

	// The same policy is visible before a write starts, in Install and Help.
	idle := testApp(t, tui.AppOptions{})
	next, _ := idle.Update(runes("2"))
	install := next.(tui.App).View().Content
	if !strings.Contains(install, "Execution safety") || !strings.Contains(install, "SIGKILL") {
		t.Fatalf("Install view does not show the execution policy before a write:\n%s", install)
	}
	next, _ = idle.Update(runes("?"))
	help := next.(tui.App).View().Content
	if !strings.Contains(help, "Write Execution Safety") || !strings.Contains(help, "SIGKILL") {
		t.Fatalf("Help view does not show the execution policy before a write:\n%s", help)
	}
}
