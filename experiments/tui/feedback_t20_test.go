package tui

// T20 tests for truthful write feedback and the final B7 regression
// (FR-10.4, AC-15, AC-16). They live in the package so they can assert the
// root-owned spinner generation, the reserved status block and the exact
// footer budget directly. Every operation is a fake: no real home, write,
// network, browser or process launch happens here.

import (
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// t20Log is an in-memory operation transcript.
type t20Log struct {
	buf     strings.Builder
	flushed bool
	closed  bool
}

func (l *t20Log) Write(p []byte) (int, error) { return l.buf.Write(p) }
func (l *t20Log) Flush() error                { l.flushed = true; return nil }
func (l *t20Log) Close() error                { l.closed = true; return nil }

// t20FakeHost is a gated fake operation host. The engine call blocks until
// finish releases it, so a test can hold a transaction in flight without any
// filesystem or network work.
type t20FakeHost struct {
	host   *OpsHost
	log    *t20Log
	gate   chan struct{}
	once   sync.Once
	calls  int32
	tokens []string
}

func t20NewHost(t *testing.T) *t20FakeHost {
	t.Helper()
	fake := &t20FakeHost{log: &t20Log{}, gate: make(chan struct{})}
	fake.host = NewOpsHost(OpsHostOptions{
		OpenLog: func(WriteKind, string) (WriteLog, string, error) {
			return fake.log, "/tmp/t20-operation.log", nil
		},
		ApplyInstall: func(_ appops.InstallRequest, token string, _ io.Writer) (appops.ApplyResult, error) {
			atomic.AddInt32(&fake.calls, 1)
			fake.tokens = append(fake.tokens, token)
			<-fake.gate
			return appops.ApplyResult{
				Operation: skillsync.OperationApply,
				Outcome:   skillsync.OutcomeApplied,
				Applied: []skillsync.ResultChange{{
					Client: "claude",
					ID:     "alpha-skill",
					Action: skillsync.ChangeInstall,
				}},
			}, nil
		},
	})
	t.Cleanup(fake.finish)
	return fake
}

// finish releases the fake engine exactly once.
func (f *t20FakeHost) finish() { f.once.Do(func() { close(f.gate) }) }

func (f *t20FakeHost) callCount() int32 { return atomic.LoadInt32(&f.calls) }

func t20Request(t *testing.T) WriteRequest {
	t.Helper()
	request, err := appops.NewInstallRequest("", t.TempDir(), true, skillsync.Options{ClientIDs: []string{"claude"}})
	if err != nil {
		t.Fatalf("NewInstallRequest: %v", err)
	}
	return WriteRequest{
		Kind:    WriteInstall,
		Install: request,
		Token:   "t20-preview-token",
		Command: "apply --clients claude",
		Label:   "install for 1 client",
	}
}

// t20AnimatedConfig is an interactive colored session with motion on: the only
// configuration where an indeterminate animation is truthful.
func t20AnimatedConfig() TUIConfig {
	return TUIConfig{Width: 100, Height: 30, Color: true, Interactive: true, Motion: true}
}

func t20App(t *testing.T, cfg TUIConfig, opts AppOptions) App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	return NewAppWithOptions(cfg.Normalize(), Catalog{Home: cfg.Home}, opts)
}

func t20Update(t *testing.T, app App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated, cmd
}

func t20Key(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func t20Press(t *testing.T, app App, r rune) App {
	t.Helper()
	app, _ = t20Update(t, app, t20Key(r))
	return app
}

func t20Code(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

// t20Started is the result of accepting one write through the real message
// path: the busy app, the (unexecuted) work command and the spinner tick the
// start scheduled.
type t20Started struct {
	app  App
	work tea.Cmd
	tick tea.Cmd
}

// t20Start sends StartWriteMsg and processes the typed start message exactly
// like the program would, without running the engine command.
func t20Start(t *testing.T, app App, request WriteRequest) t20Started {
	t.Helper()
	return t20Claim(t, app, request)
}

// t20ClaimCmd is t20Start for a command that carries the screen's StartWriteMsg.
func t20ClaimCmd(t *testing.T, app App, cmd tea.Cmd) t20Started {
	t.Helper()
	if cmd == nil {
		t.Fatal("the confirmed screen returned no command")
	}
	msg, ok := cmd().(StartWriteMsg)
	if !ok {
		t.Fatalf("the confirmed screen returned %T, want StartWriteMsg", cmd())
	}
	return t20Claim(t, app, msg.Request)
}

// t20Claim accepts one write through the real message path and returns the busy
// app, the (unexecuted) work command and the spinner tick the start scheduled.
func t20Claim(t *testing.T, app App, request WriteRequest) t20Started {
	t.Helper()
	app, cmd := t20Update(t, app, StartWriteMsg{Request: request})
	if cmd == nil {
		t.Fatal("StartWriteMsg returned no command")
	}
	raw := cmd()
	batch, ok := raw.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("StartWriteMsg command = %T, want a two-command batch", raw)
	}
	started, ok := batch[0]().(WriteStartedMsg)
	if !ok {
		t.Fatalf("first command returned %T, want WriteStartedMsg", batch[0]())
	}
	app, tick := t20Update(t, app, started)
	if !app.Busy() {
		t.Fatal("app is not busy after the typed start message")
	}
	return t20Started{app: app, work: batch[1], tick: tick}
}

func t20LastLine(view string) string {
	if index := strings.LastIndex(view, "\n"); index >= 0 {
		return view[index+1:]
	}
	return view
}

// t20Flat normalizes a rendered view for text assertions: escape sequences are
// stripped and wrapping is collapsed, so a complete sentence matches regardless
// of where it was wrapped.
func t20Flat(text string) string { return strings.Join(strings.Fields(ansi.Strip(text)), " ") }

func t20Widths(lines []string) []int {
	widths := make([]int, len(lines))
	for i, line := range lines {
		widths[i] = ansi.StringWidth(line)
	}
	return widths
}

// t20StatusRegion returns the footer rows below the last divider: the reserved
// notice block and the key hint.
func t20StatusRegion(view string) []string {
	lines := strings.Split(view, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		stripped := strings.TrimSpace(ansi.Strip(lines[i]))
		if stripped == "" {
			continue
		}
		if strings.Trim(stripped, "-─") == "" {
			return lines[i+1:]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Spinner lifecycle
// ---------------------------------------------------------------------------

func TestT20SpinnerStartsAdvancesAndStops(t *testing.T) {
	fake := t20NewHost(t)
	app := t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host})

	started := t20Start(t, app, t20Request(t))
	app = started.app
	if !app.spinnerActive || !app.spinnerPending {
		t.Fatal("the accepted write did not start the root-owned spinner")
	}
	if started.tick == nil {
		t.Fatal("the accepted write did not schedule a spinner tick")
	}
	first := app.writePhaseMarker()
	if first == WriteStaticMarker {
		t.Fatalf("animated session rendered the static marker %q", first)
	}
	if !strings.Contains(t20LastLine(app.View().Content), WriteBusyMarker) {
		t.Fatal("the busy footer is missing while the spinner runs")
	}

	// A tick that belongs to this generation advances exactly one frame.
	app, cmd := t20Update(t, app, started.tick())
	if cmd == nil {
		t.Fatal("a valid tick did not schedule the next frame")
	}
	second := app.writePhaseMarker()
	if second == first {
		t.Fatalf("tick did not advance the spinner: %q", second)
	}
	if !app.spinnerPending {
		t.Fatal("the advanced spinner is not pending another tick")
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("a spinner tick dispatched %d operation(s), want 0", got)
	}

	// The result stops the animation and the tick chain.
	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("no active write status to correlate the result")
	}
	app, _ = t20Update(t, app, WriteResultMsg{
		ID:      status.ID,
		Kind:    WriteInstall,
		Label:   status.Label,
		Result:  appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeApplied},
		LogPath: "/tmp/t20-operation.log",
	})
	if app.spinnerActive || app.spinnerPending {
		t.Fatal("the completed write left the spinner running")
	}
	app, cmd = t20Update(t, app, spinner.TickMsg{ID: app.spinner.ID()})
	if cmd != nil {
		t.Fatal("a tick after completion scheduled another frame")
	}
	if app.View().Content == "" {
		t.Fatal("the post-completion view is empty")
	}
}

func TestT20StaleTickIsIgnored(t *testing.T) {
	fake := t20NewHost(t)
	app := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t)).app

	before := app.writePhaseMarker()
	app, cmd := t20Update(t, app, spinner.TickMsg{ID: app.spinner.ID() + 41})
	if cmd != nil {
		t.Fatal("a stale tick scheduled another frame")
	}
	if after := app.writePhaseMarker(); after != before {
		t.Fatalf("a stale tick advanced the spinner: %q -> %q", before, after)
	}
	if !app.spinnerActive {
		t.Fatal("a stale tick stopped the live generation")
	}

	// The live generation still advances afterwards.
	app, cmd = t20Update(t, app, app.spinner.Tick())
	if cmd == nil {
		t.Fatal("the live generation stopped advancing after a stale tick")
	}
	if after := app.writePhaseMarker(); after == before {
		t.Fatal("the live tick did not advance the spinner")
	}
}

func TestT20StaticMarkerWhenMotionIsUnavailable(t *testing.T) {
	cases := []struct {
		name string
		cfg  TUIConfig
	}{
		{"motion off", TUIConfig{Width: 100, Height: 30, Color: true, Interactive: true}},
		{"non interactive", TUIConfig{Width: 100, Height: 30, Color: true, Motion: true}},
		{"plain", TUIConfig{Width: 100, Height: 30, Interactive: true, Motion: true}},
		{"ascii", TUIConfig{Width: 100, Height: 30, Color: true, Interactive: true, Motion: true, ASCII: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := t20NewHost(t)
			started := t20Start(t, t20App(t, tc.cfg, AppOptions{Ops: fake.host}), t20Request(t))
			app := started.app
			if app.spinnerActive || app.spinnerPending {
				t.Fatal("a static session started the spinner")
			}
			if started.tick != nil {
				t.Fatal("a static session scheduled a spinner tick")
			}
			line := t20LastLine(app.View().Content)
			if !strings.Contains(line, WriteStaticMarker) {
				t.Fatalf("static busy line lacks %q: %q", WriteStaticMarker, ansi.Strip(line))
			}
			if !strings.Contains(line, string(WriteStepStarting)) {
				t.Fatalf("static busy line lacks the host phase: %q", ansi.Strip(line))
			}
			app, cmd := t20Update(t, app, spinner.TickMsg{ID: 1})
			if cmd != nil {
				t.Fatal("a tick animated a static session")
			}
			if after := t20LastLine(app.View().Content); after != line {
				t.Fatalf("the static marker changed on a tick: %q -> %q", line, after)
			}
		})
	}
}

// TestT20HostPhaseSequenceDrivesFeedback runs one fake transaction end to end
// and replays the phases the host really emitted: starting, writing, flushing
// and the single terminal result. The UI phase text can only come from those
// messages.
func TestT20HostPhaseSequenceDrivesFeedback(t *testing.T) {
	fake := t20NewHost(t)
	var emitted []tea.Msg
	fake.host.SetProgressSink(func(msg tea.Msg) { emitted = append(emitted, msg) })

	started := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t))
	app := started.app

	// Release the fake engine before running its command so the whole
	// transaction completes synchronously in this goroutine.
	fake.finish()
	result, ok := started.work().(WriteResultMsg)
	if !ok {
		t.Fatalf("work command returned %T, want WriteResultMsg", started.work())
	}

	wantSteps := []WriteStep{WriteStepStarting, WriteStepWriting, WriteStepFlushing}
	if len(emitted) != len(wantSteps) {
		t.Fatalf("host emitted %d phase messages, want %d", len(emitted), len(wantSteps))
	}
	for i, want := range wantSteps {
		progress, ok := emitted[i].(WriteProgressMsg)
		if !ok {
			t.Fatalf("emitted[%d] = %T, want WriteProgressMsg", i, emitted[i])
		}
		if progress.Step != want {
			t.Fatalf("emitted[%d].Step = %q, want %q", i, progress.Step, want)
		}
		app, _ = t20Update(t, app, progress)
		if got := app.writePhaseText(); got != string(want) {
			t.Fatalf("phase text = %q, want the host's %q", got, want)
		}
		line := ansi.Strip(t20LastLine(app.View().Content))
		if !strings.Contains(line, string(want)) {
			t.Fatalf("busy line does not show phase %q: %q", want, line)
		}
		if !strings.Contains(line, WriteBusyMarker) || !strings.Contains(line, "Ctrl+C exits after completion") {
			t.Fatalf("busy line dropped the deferred-exit policy: %q", line)
		}
		if strings.Contains(line, "%") {
			t.Fatalf("busy line invented a percentage: %q", line)
		}
		if !app.spinnerActive {
			t.Fatalf("phase %q stopped the spinner before the result", want)
		}
	}

	// Flushing must not read as completion while the log is still open.
	if strings.Contains(strings.ToLower(ansi.Strip(t20LastLine(app.View().Content))), "completed") {
		t.Fatal("the flushing footer claimed completion before the result")
	}

	app, _ = t20Update(t, app, result)
	if app.Busy() {
		t.Fatal("the terminal result did not clear the write")
	}
	if app.spinnerActive || app.spinnerPending {
		t.Fatal("the terminal result did not stop the spinner")
	}
	if !fake.log.flushed || !fake.log.closed {
		t.Fatal("the fake transcript was not flushed and closed before the result")
	}
	flat := t20Flat(app.View().Content)
	if !strings.Contains(flat, "completed") || !strings.Contains(flat, "Wrote 1 planned changes.") {
		t.Fatalf("the persistent notice lost the structured result:\n%s", app.View().Content)
	}
	if strings.Contains(t20LastLine(app.View().Content), WriteBusyMarker) {
		t.Fatal("the busy footer survived the completed write")
	}
}

// TestT20WriteErrorKeepsSpinnerStoppedAndNoticeComplete proves the failure path:
// a failed write stops the animation, keeps the full explanation and its log
// reference, and never claims success.
func TestT20WriteErrorKeepsSpinnerStoppedAndNoticeComplete(t *testing.T) {
	fake := t20NewHost(t)
	started := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t))
	app := started.app
	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("no active write status")
	}
	failure := io.ErrUnexpectedEOF
	app, _ = t20Update(t, app, WriteResultMsg{
		ID:      status.ID,
		Kind:    WriteInstall,
		Result:  appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed},
		Err:     failure,
		LogPath: "/tmp/t20-operation.log",
	})
	if app.spinnerActive {
		t.Fatal("a failed write left the spinner running")
	}
	flat := t20Flat(app.View().Content)
	if !strings.Contains(flat, "failed") {
		t.Fatalf("the failure notice is missing:\n%s", app.View().Content)
	}
	if strings.Contains(strings.ToLower(flat), "completed") {
		t.Fatalf("a failed write was reported as completed:\n%s", app.View().Content)
	}
	if !strings.Contains(flat, "Notice:") {
		t.Fatal("the failure is not in the reserved status block")
	}
}

func TestT20QuitStopsSpinnerAndKeepsDeferredExitVisible(t *testing.T) {
	fake := t20NewHost(t)
	started := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t))
	app := started.app

	// Ctrl+C during a write is a deferred request: the animation keeps running
	// because the write really is still running.
	app, cmd := t20Update(t, app, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Fatal("Ctrl+C during a write returned a quit command")
	}
	if !app.ExitPending() || app.quitting {
		t.Fatal("Ctrl+C during a write did not record a deferred exit")
	}
	if !app.spinnerActive {
		t.Fatal("the deferred exit stopped the spinner while the write still runs")
	}
	if line := ansi.Strip(t20LastLine(app.View().Content)); !strings.Contains(line, "exit pending") {
		t.Fatalf("busy line dropped the deferred exit: %q", line)
	}

	status, _ := app.WriteStatus()
	app, cmd = t20Update(t, app, WriteResultMsg{
		ID:     status.ID,
		Kind:   WriteInstall,
		Result: appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeApplied},
	})
	if cmd == nil {
		t.Fatal("the deferred exit did not quit after the result")
	}
	if app.spinnerActive || app.spinnerPending {
		t.Fatal("the deferred quit left the spinner running")
	}
	app, cmd = t20Update(t, app, spinner.TickMsg{ID: app.spinner.ID()})
	if cmd != nil {
		t.Fatal("a tick after the deferred quit scheduled another frame")
	}
}

// ---------------------------------------------------------------------------
// Busy policy precedence and persistent notices
// ---------------------------------------------------------------------------

func TestT20BusyPolicyPrecedenceAndNoDuplicateNotice(t *testing.T) {
	fake := t20NewHost(t)
	app := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t)).app

	app, cmd := t20Update(t, app, t20Key('q'))
	if cmd != nil || !app.Busy() {
		t.Fatal("q abandoned or quit an active write")
	}
	view := app.View().Content
	line := ansi.Strip(t20LastLine(view))
	if !strings.Contains(line, WriteBusyMarker) {
		t.Fatalf("busy footer lost its marker: %q", line)
	}
	if strings.Contains(line, "q quit") {
		t.Fatalf("busy footer offered q quit: %q", line)
	}
	if !strings.Contains(line, "q/Esc disabled") || !strings.Contains(line, "Ctrl+C exits after completion") {
		t.Fatalf("busy footer lost the policy: %q", line)
	}
	// The explanation is shown once: the busy footer carries the policy, so the
	// reserved notice block is suppressed instead of repeating it.
	if got := strings.Count(t20Flat(view), "write in progress"); got != 1 {
		t.Fatalf("busy explanation appears %d times, want once:\n%s", got, view)
	}
	if app.HelpOpen() {
		t.Fatal("help opened during a write")
	}
	app = t20Press(t, app, '?')
	if app.HelpOpen() {
		t.Fatal("? opened help during a write")
	}
	if app.modalActive() {
		t.Fatal("a modal composed over the active write")
	}
}

func TestT20NoticePersistsInReservedStatusBlock(t *testing.T) {
	notice := "Apply refused: a write is already running for apply. Nothing was written; wait for the active operation and its log to finish."
	for _, width := range []int{60, 80, 110} {
		app := t20App(t, TUIConfig{Width: width, Height: 30}, AppOptions{})
		app.notice = notice

		view := app.View().Content
		if !strings.Contains(t20Flat(view), notice) {
			t.Fatalf("width %d: the notice was truncated:\n%s", width, view)
		}
		region := t20StatusRegion(view)
		if len(region) == 0 || !strings.Contains(t20Flat(strings.Join(region, " ")), "Apply refused") {
			t.Fatalf("width %d: the notice is not in the reserved status block:\n%s", width, view)
		}
		if !strings.Contains(t20LastLine(view), "q quit") {
			t.Fatalf("width %d: the key hint is not the last footer row:\n%s", width, view)
		}

		// Scrolling the page cannot dismiss or hide it: it is chrome, not body.
		for i := 0; i < 20; i++ {
			app, _ = t20Update(t, app, t20Code(tea.KeyPgDown))
		}
		scrolled := app.View().Content
		if !strings.Contains(t20Flat(scrolled), notice) {
			t.Fatalf("width %d: scrolling hid the notice:\n%s", width, scrolled)
		}
		// Re-rendering (a resize or a tick) never drops it either.
		app, _ = t20Update(t, app, tea.WindowSizeMsg{Width: width, Height: 30})
		if !strings.Contains(t20Flat(app.View().Content), notice) {
			t.Fatalf("width %d: a resize dropped the notice", width)
		}
	}
}

func TestT20LongNoticeStaysCompleteInBody(t *testing.T) {
	long := "Recovery detail: " + strings.Repeat("verify the resolved home and repo, then preview again. ", 24) + "END-OF-NOTICE"
	app := t20App(t, TUIConfig{Width: 60, Height: 12}, AppOptions{})
	app.notice = long

	first := app.View().Content
	if !strings.Contains(t20Flat(first), "Recovery detail:") {
		t.Fatalf("the notice is not visible at all:\n%s", first)
	}
	if strings.Contains(t20Flat(strings.Join(t20StatusRegion(first), " ")), "Recovery detail") {
		t.Fatal("a notice taller than the reserved budget consumed the body")
	}

	// It stays complete and reachable: the body keeps every row.
	seen := first
	for i := 0; i < 40; i++ {
		app, _ = t20Update(t, app, t20Code(tea.KeyPgDown))
		seen += "\n" + app.View().Content
	}
	if !strings.Contains(t20Flat(seen), "END-OF-NOTICE") {
		t.Fatal("the long notice was truncated instead of being paged in the body")
	}
}

func TestT20NoFabricatedProgressOrNativeProgress(t *testing.T) {
	fake := t20NewHost(t)
	started := t20Start(t, t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host}), t20Request(t))
	app := started.app
	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("no active write status")
	}

	for _, step := range []WriteStep{WriteStepStarting, WriteStepWriting, WriteStepFlushing} {
		app, _ = t20Update(t, app, WriteProgressMsg{ID: status.ID, Kind: WriteInstall, Step: step})
		view := app.View()
		if view.ProgressBar != nil {
			t.Fatal("a native terminal progress bar was added (D04 omitted it)")
		}
		if view.WindowTitle != BrandName || !view.AltScreen {
			t.Fatalf("the view chrome changed: title=%q altscreen=%t", view.WindowTitle, view.AltScreen)
		}
		line := ansi.Strip(t20LastLine(view.Content))
		for _, banned := range []string{"%", "ProgramStatus", "sparkline", "▁", "▂", "▃", "█"} {
			if strings.Contains(line, banned) {
				t.Fatalf("busy line contains fabricated progress %q: %q", banned, line)
			}
		}
		if !strings.Contains(line, string(step)) {
			t.Fatalf("busy line lost the real phase %q: %q", step, line)
		}
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("presentation dispatched %d operation(s), want 0", got)
	}
}

// ---------------------------------------------------------------------------
// AC-15 / AC-16 final regression
// ---------------------------------------------------------------------------

func TestT20AC15NoticeRoleKeepsContrast(t *testing.T) {
	for _, dark := range []bool{true, false} {
		theme := NewAdaptiveTheme(true, dark)
		canvas := contrastRGB(t, theme.CanvasColor())
		ratio := contrastRatio(contrastRGB(t, theme.Notice.GetForeground()), canvas)
		if ratio+1e-9 < 7 {
			t.Fatalf("dark=%t: notice contrast %.2f:1, want >= 7:1", dark, ratio)
		}
		// The role is foreground-only: it must not paint a background that
		// would silently change the measured pair.
		if rendered := theme.Notice.Render("notice"); strings.Contains(rendered, "\x1b[48") {
			t.Fatalf("dark=%t: the notice role paints its own background: %q", dark, rendered)
		}
	}

	// The reserved block uses the semantic notice token, not a local color.
	app := t20App(t, TUIConfig{Width: 100, Height: 30, Color: true}, AppOptions{})
	app.notice = "Apply refused."
	want := app.theme.Notice.Render("Notice: Apply refused.")
	if !strings.Contains(app.View().Content, want) {
		t.Fatalf("the status block did not use the theme notice role:\n%q", app.View().Content)
	}

	// A plain session paints no escape sequence at all.
	plain := t20App(t, TUIConfig{Width: 100, Height: 30}, AppOptions{})
	plain.notice = "Apply refused."
	if view := plain.View().Content; strings.Contains(view, "\x1b") {
		t.Fatalf("a color-off session emitted SGR for its notice:\n%q", view)
	}
	if plain.View().BackgroundColor != nil || plain.View().ForegroundColor != nil {
		t.Fatal("a color-off session requested a terminal background or foreground")
	}
}

func TestT20AC16BoundsWithReservedStatusNotice(t *testing.T) {
	notice := "Apply refused: the operation host is unavailable. Nothing was written; check the resolved --home and --repo, then preview again."
	for _, color := range []bool{false, true} {
		for _, width := range []int{40, 60, 79, 80, 109, 110} {
			for _, height := range []int{8, 12, 16, 24} {
				app := t20App(t, TUIConfig{Width: width, Height: height, Color: color}, AppOptions{})
				app.notice = notice
				view := app.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) != height {
					t.Fatalf("color=%t %dx%d: rows = %d, want %d", color, width, height, len(lines), height)
				}
				for row, got := range t20Widths(lines) {
					if got > width {
						t.Fatalf("color=%t %dx%d: row %d width %d: %q", color, width, height, row, got, ansi.Strip(lines[row]))
					}
				}
				if height > 8 && !strings.Contains(t20Flat(view), notice) {
					t.Fatalf("color=%t %dx%d: the notice was truncated:\n%s", color, width, height, view)
				}
				if height <= 8 && !strings.Contains(view, "Notice") {
					t.Fatalf("color=%t %dx%d: a tiny terminal dropped the notice:\n%s", color, width, height, view)
				}
			}
		}
	}
}

func TestT20AC16ModalSafetyWithStatusNotice(t *testing.T) {
	app := t20App(t, TUIConfig{Width: 100, Height: 30, Color: true}, AppOptions{})
	app.notice = "Apply refused: nothing was written."
	app = t20Press(t, app, '?')
	if !app.modalActive() {
		t.Fatal("help did not compose a bounded modal")
	}
	view := app.View().Content
	// The modal owns the body region; the reserved status block stays visible
	// below it instead of being covered or lost.
	region := t20Flat(strings.Join(t20StatusRegion(view), " "))
	if !strings.Contains(region, "nothing was written") {
		t.Fatalf("the modal covered the reserved status block:\n%s", view)
	}
	app, _ = t20Update(t, app, t20Code(tea.KeyEscape))
	if app.HelpOpen() {
		t.Fatal("Esc did not close help")
	}
	if !strings.Contains(app.View().Content, "nothing was written") {
		t.Fatal("the notice did not return after the modal closed")
	}
}

func TestT20AC16ConfirmationSemanticsUnchanged(t *testing.T) {
	fake := t20NewHost(t)
	app := t20OpenInstallConfirm(t, t20AnimatedConfig(), fake.host, "/tmp/t20/skills/alpha-skill")

	// Enter stays inert at the confirmation step.
	app, cmd := t20Update(t, app, t20Code(tea.KeyEnter))
	if cmd != nil || app.install.Step() != InstallStepConfirm {
		t.Fatalf("Enter confirmed the write: step=%v cmd=%v", app.install.Step(), cmd != nil)
	}
	// A notice never changes confirmation semantics.
	app.notice = "previous refusal"

	app, cmd = t20Update(t, app, t20Key('y'))
	if cmd == nil {
		t.Fatal("y did not request the confirmed write")
	}
	if app.install.Step() != InstallStepRunning {
		t.Fatalf("step = %v, want running", app.install.Step())
	}
	if app.install.previewToken != "t19-install-token" {
		t.Fatalf("preview token = %q, want the validated one", app.install.previewToken)
	}

	app, _ = t20Update(t, app, cmd())
	if !app.Busy() {
		t.Fatal("the confirmed write did not claim the single slot")
	}
	// A duplicate confirmation cannot start a second operation.
	app, dup := t20Update(t, app, t20Key('y'))
	if dup != nil {
		t.Fatal("a duplicate y dispatched another command during the write")
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("the engine ran %d times before its command executed", got)
	}
}

// t20OpenInstallConfirm walks the existing preview-first journey to the
// confirmation step with a fake host, so a confirmation regression is caught
// without any real write.
func t20OpenInstallConfirm(t *testing.T, cfg TUIConfig, host *OpsHost, path string) App {
	t.Helper()
	if cfg.Home == "" {
		cfg.Home = t.TempDir()
	}
	app := NewAppWithScreens(cfg.Normalize(), Catalog{Home: cfg.Home}, AppOptions{Ops: host}, AppScreenOptions{
		InstallPreview: t19InstallPreview(path),
	})
	app = t20Press(t, app, '2') // Install & Manage hub
	app = t20Press(t, app, 'i') // install journey
	app, _ = t20Update(t, app, t20Code(tea.KeyEnter))
	app, _ = t20Update(t, app, t20Code(tea.KeyEnter))
	app, cmd := t20Update(t, app, t20Code(tea.KeyEnter))
	if cmd != nil {
		app, _ = t20Update(t, app, cmd())
	}
	app, _ = t20Update(t, app, t20Code(tea.KeyEnter))
	if app.install.Step() != InstallStepConfirm {
		t.Fatalf("install step = %v, want confirm", app.install.Step())
	}
	return app
}

// TestT20JourneyKeepsPolicyAndLogReference proves the feedback never hides the
// execution policy before a write, during it, or the log reference after it.
func TestT20JourneyKeepsPolicyAndLogReference(t *testing.T) {
	fake := t20NewHost(t)
	app := t20OpenInstallConfirm(t, t20AnimatedConfig(), fake.host, "/tmp/t20/skills/alpha-skill")

	// Before the write: the confirmation step still carries the full policy.
	confirm := t20Flat(app.View().Content)
	if !strings.Contains(confirm, "Writes cannot be interrupted midway once started.") || !strings.Contains(confirm, "SIGKILL") {
		t.Fatalf("the confirmation step lost the execution policy:\n%s", app.View().Content)
	}

	app, cmd := t20Update(t, app, t20Key('y'))
	started := t20ClaimCmd(t, app, cmd)
	app = started.app

	// During the write: the running page keeps the policy and the footer keeps
	// the deferred-exit promise, beside the phase marker.
	running := t20Flat(app.View().Content)
	if !strings.Contains(running, "cannot be interrupted") || !strings.Contains(running, "Ctrl+C") {
		t.Fatalf("the running page lost the write policy:\n%s", app.View().Content)
	}
	if line := ansi.Strip(t20LastLine(app.View().Content)); !strings.Contains(line, "Ctrl+C exits after completion") {
		t.Fatalf("the busy footer lost the deferred-exit policy: %q", line)
	}
	if !app.spinnerActive {
		t.Fatal("the install journey did not start the phase spinner")
	}

	status, ok := app.WriteStatus()
	if !ok {
		t.Fatal("no active write status")
	}
	app, _ = t20Update(t, app, WriteResultMsg{
		ID:      status.ID,
		Kind:    WriteInstall,
		Label:   status.Label,
		Result:  appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeApplied},
		LogPath: "/tmp/t20-operation.log",
	})
	if app.Busy() || app.spinnerActive {
		t.Fatal("the result did not stop the write feedback")
	}
	// After the write: the result page keeps the full log reference and the
	// reserved notice keeps the structured outcome.
	done := t20Flat(app.View().Content)
	if !strings.Contains(done, "Operation log: /tmp/t20-operation.log") {
		t.Fatalf("the result page lost the operation log reference:\n%s", app.View().Content)
	}
	if !strings.Contains(done, "completed") {
		t.Fatalf("the result notice is missing:\n%s", app.View().Content)
	}
}

// TestT20RefusalStopsSpinnerAndPersists proves a refused request animates
// nothing and still leaves a complete, reserved explanation.
func TestT20RefusalStopsSpinnerAndPersists(t *testing.T) {
	fake := t20NewHost(t)
	app := t20App(t, t20AnimatedConfig(), AppOptions{Ops: fake.host})

	// A request without a validated preview token is refused before any slot is
	// claimed.
	app, cmd := t20Update(t, app, StartWriteMsg{Request: WriteRequest{Kind: WriteInstall}})
	if cmd == nil {
		t.Fatal("a refused start returned no message command")
	}
	app, _ = t20Update(t, app, cmd())
	if app.Busy() {
		t.Fatal("a refused request claimed the write slot")
	}
	if app.spinnerActive || app.spinnerPending {
		t.Fatal("a refused request started the phase spinner")
	}
	view := app.View().Content
	if !strings.Contains(t20Flat(view), "preview token is required") {
		t.Fatalf("the refusal explanation is missing:\n%s", view)
	}
	if !strings.Contains(t20Flat(strings.Join(t20StatusRegion(view), " ")), "preview token is required") {
		t.Fatalf("the refusal is not in the reserved status block:\n%s", view)
	}
	if got := fake.callCount(); got != 0 {
		t.Fatalf("a refused request ran the engine %d time(s)", got)
	}

	// A start for an unknown identity cannot claim the slot or animate.
	app, cmd = t20Update(t, app, WriteStartedMsg{ID: appops.OperationID{Request: 99, Generation: 1}, Kind: WriteInstall})
	if cmd != nil || app.spinnerActive || app.Busy() {
		t.Fatal("an unmatched start message claimed the write or started the spinner")
	}
}
