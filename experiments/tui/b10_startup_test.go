package tui

// B10/T23 initial-frame and generation tests (Feature 022, AC-19 and AC-21).
//
// Each case blocks a real reader on a channel, observes the first frame while the
// read is still stalled, then changes the home, route or log directory and
// releases the older result. The stale result must be ignored, the live request
// must still be applied, and a read error must be visible. Elapsed times are
// recorded for the record; none of them is asserted against a threshold.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// b10Call is one stalled read. The test releases it through close(release) and
// sets status/err before doing so; the channel close is the happens-before edge.
type b10Call struct {
	home    string
	release chan struct{}
	status  MCPStatusView
	err     error
}

// b10StalledReader blocks every Status call until its own call is released and
// reports each started call on entered, so a test can prove the reader really is
// in flight rather than merely scheduled.
type b10StalledReader struct {
	mu      sync.Mutex
	calls   []*b10Call
	entered chan *b10Call
}

func newB10StalledReader() *b10StalledReader {
	return &b10StalledReader{entered: make(chan *b10Call, 8)}
}

func (r *b10StalledReader) Status(ctx context.Context, home, configPath, projectPath string) (MCPStatusView, error) {
	call := &b10Call{home: home, release: make(chan struct{})}
	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()
	r.entered <- call
	select {
	case <-call.release:
		return call.status, call.err
	case <-ctx.Done():
		return MCPStatusView{}, ctx.Err()
	}
}

func (r *b10StalledReader) Doctor(context.Context, string, string, string, string, string) (MCPDoctorView, error) {
	return MCPDoctorView{}, nil
}

// callCount reports how many reads were started.
func (r *b10StalledReader) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// b10Start runs one command on its own goroutine and returns its message channel.
func b10Start(cmd tea.Cmd) <-chan tea.Msg {
	out := make(chan tea.Msg, 1)
	go func() { out <- cmd() }()
	return out
}

// TestB10MCPStalledStatusRendersUsefulFirstFrame covers AC-21 for the MCP status
// read: opening the screen while the reader is stalled renders a useful frame,
// and the read result is applied afterwards.
func TestB10MCPStalledStatusRendersUsefulFirstFrame(t *testing.T) {
	reader := newB10StalledReader()
	app := b10MCPApp(t, reader, &fakeMCPPlanner{})

	started := time.Now()
	cmd := app.openMCPScreen()
	if cmd == nil {
		t.Fatal("opening MCP Tools scheduled no read")
	}
	firstFrameFor := time.Since(started)

	// Observe the first frame before the read has produced anything.
	frame := b10RenderMCP(app)
	for _, want := range []string{
		"Reading the MCP configuration status",
		"CodeGraph",
		"Context7",
		"Exa",
		"state not read yet",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("first frame with a stalled reader is missing %q:\n%s", want, frame)
		}
	}

	result := b10Start(cmd)
	call := <-reader.entered
	// The read is genuinely in flight: no message can be ready yet.
	select {
	case msg := <-result:
		t.Fatalf("the stalled read returned %T before it was released", msg)
	default:
	}
	// The frame stays useful while the read is stalled.
	if again := b10RenderMCP(app); !strings.Contains(again, "Reading the MCP configuration status") {
		t.Fatalf("stalled frame lost its loading state:\n%s", again)
	}

	call.status = MCPStatusView{Available: true, Home: call.home, ConfigPath: "/home/opencode.json"}
	close(call.release)
	next, _ := b10UpdateApp(t, app, <-result)

	loaded := b10RenderMCP(next)
	if strings.Contains(loaded, "Reading the MCP configuration status") {
		t.Fatalf("loading state survived an applied status:\n%s", loaded)
	}
	if strings.Contains(loaded, "state not read yet") {
		t.Fatalf("applied status still claims it was not read:\n%s", loaded)
	}
	if got := next.mcp.status.ConfigPath; got != "/home/opencode.json" {
		t.Fatalf("status config path = %q, want the read result", got)
	}
	t.Logf("MCP first frame with a stalled reader: %s (no threshold asserted)", firstFrameFor)
}

// TestB10MCPStaleHomeResultIsSuppressed covers AC-21's stale-result rule for the
// status read: changing the home makes the older request stale, its result is
// ignored, and the live request is still applied.
func TestB10MCPStaleHomeResultIsSuppressed(t *testing.T) {
	reader := newB10StalledReader()
	homeA := t.TempDir()
	homeB := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: homeA}
	m := NewMCPModelWithSeams(cfg, reader, &fakeMCPPlanner{})

	m, staleCmd := m.EnsureLoaded()
	staleResult := b10Start(staleCmd)
	staleCall := <-reader.entered
	if staleCall.home != homeA {
		t.Fatalf("first read home = %q, want %q", staleCall.home, homeA)
	}

	// The session's home changes and the screen starts a fresh read.
	m.cfg.Home = homeB
	m, liveCmd := m.Reload()
	liveResult := b10Start(liveCmd)
	liveCall := <-reader.entered
	if liveCall.home != homeB {
		t.Fatalf("second read home = %q, want %q", liveCall.home, homeB)
	}

	// Release the older read first.
	staleCall.status = MCPStatusView{Available: true, Home: homeA, ConfigPath: "/stale/opencode.json"}
	close(staleCall.release)
	afterStale, _ := m.Update(<-staleResult)
	if afterStale.status.ConfigPath != "" {
		t.Fatalf("a result for the previous home was applied: %+v", afterStale.status)
	}
	if !afterStale.loading {
		t.Fatal("a stale result cleared the live loading state")
	}
	if afterStale.hasLoaded {
		t.Fatal("a stale result marked the screen as loaded")
	}

	// Release the live read.
	liveCall.status = MCPStatusView{Available: true, Home: homeB, ConfigPath: "/live/opencode.json"}
	close(liveCall.release)
	afterLive, _ := m.Update(<-liveResult)
	if afterLive.status.ConfigPath != "/live/opencode.json" {
		t.Fatalf("live result was not applied: %+v", afterLive.status)
	}
	if afterLive.loading || !afterLive.hasLoaded {
		t.Fatalf("live result state = loading:%v loaded:%v, want applied", afterLive.loading, afterLive.hasLoaded)
	}
}

// TestB10MCPStatusErrorIsVisible covers AC-21's error requirement: a failed read
// is reported on the screen instead of being swallowed.
func TestB10MCPStatusErrorIsVisible(t *testing.T) {
	reader := newB10StalledReader()
	app := b10MCPApp(t, reader, &fakeMCPPlanner{})

	cmd := app.openMCPScreen()
	result := b10Start(cmd)
	call := <-reader.entered
	call.err = errors.New("opencode.json is unreadable")
	close(call.release)
	next, _ := b10UpdateApp(t, app, <-result)

	frame := b10RenderMCP(next)
	if !strings.Contains(frame, "Status error: opencode.json is unreadable") {
		t.Fatalf("read error is not visible:\n%s", frame)
	}
	if next.mcp.hasLoaded {
		t.Fatal("a failed read marked the status as loaded")
	}
}

// b10UpdateApp sends one message and keeps the concrete App.
func b10UpdateApp(t *testing.T, app App, msg tea.Msg) (App, tea.Cmd) {
	t.Helper()
	next, cmd := app.Update(msg)
	updated, ok := next.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", next)
	}
	return updated, cmd
}

// TestB10ActivityInitialReadIsDeferred covers AC-21 for the log source: the model
// does no construction I/O, the launcher's first message schedules one
// generation-bound read, and the screen renders useful content meanwhile.
func TestB10ActivityInitialReadIsDeferred(t *testing.T) {
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()

	var reads int32
	release := make(chan struct{})
	app := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{})
	app.activity.logReader = func(string) ([]LogSummary, error) {
		atomic.AddInt32(&reads, 1)
		<-release
		return nil, nil
	}

	// Construction must not read the log directory.
	if got := atomic.LoadInt32(&reads); got != 0 {
		t.Fatalf("construction read the log directory %d time(s), want 0", got)
	}
	started := time.Now()
	firstFrame := app.View().Content
	firstFrameFor := time.Since(started)
	// The first frame is the Home route and must be useful without any log read.
	if !strings.Contains(firstFrame, "Home") || !strings.Contains(firstFrame, "Local inventory") {
		t.Fatalf("first frame lacked useful local content:\n%s", firstFrame)
	}

	// The launcher's first message queues exactly one generation-bound read.
	app, cmd := b10UpdateApp(t, app, appStartMsg{})
	if cmd == nil {
		t.Fatal("the start message scheduled no log read")
	}
	if got := atomic.LoadInt32(&reads); got != 0 {
		t.Fatalf("the read ran inside Update %d time(s)", got)
	}
	if !app.activity.loading {
		t.Fatal("the model does not report an in-flight read")
	}

	// Entering Activity queues its own read and renders the loading state while
	// both reads are stalled.
	app, routeCmd := b10UpdateApp(t, app, tea.KeyPressMsg{Code: '4', Text: "4"})
	routeFrame := app.View().Content
	if !strings.Contains(routeFrame, "Activity & Help") {
		t.Fatalf("Activity route frame missing content:\n%s", routeFrame)
	}
	if !strings.Contains(routeFrame, "Reading local operation logs") {
		t.Fatalf("Activity route frame does not report the stalled read:\n%s", routeFrame)
	}
	if routeCmd == nil {
		t.Fatal("entering Activity queued no log read")
	}

	close(release)
	// The route's read is the live generation and is applied.
	for _, msg := range runB9Cmd(routeCmd) {
		app, _ = b10UpdateApp(t, app, msg)
	}
	if app.activity.loading {
		t.Fatal("an applied read left the model loading")
	}
	if !app.activity.loaded {
		t.Fatal("an applied read did not mark the model loaded")
	}
	// The earlier start-up read is now stale and cannot replace the live state.
	for _, msg := range runB9Cmd(cmd) {
		app, _ = b10UpdateApp(t, app, msg)
	}
	finalFrame := app.View().Content
	if strings.Contains(finalFrame, "Reading local operation logs") {
		t.Fatalf("loading state survived the applied read:\n%s", finalFrame)
	}
	if !strings.Contains(finalFrame, "No install or restore logs found yet") {
		t.Fatalf("the accepted empty read is not shown truthfully:\n%s", finalFrame)
	}
	t.Logf("Activity first frame with a stalled log source: %s; first frame after start: %s",
		firstFrameFor, time.Since(started))
}

// TestB10ActivityStaleGenerationAndDirectoryAreSuppressed proves the log read is
// bound to both its generation and its directory: an older result is rejected,
// the live read keeps its loading state, and an error from the live read is
// visible.
func TestB10ActivityStaleGenerationAndDirectoryAreSuppressed(t *testing.T) {
	var reads int32
	release := make(chan struct{})
	model := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 100,
		LogReader: func(readDir string) ([]LogSummary, error) {
			atomic.AddInt32(&reads, 1)
			<-release
			if strings.Contains(readDir, "broken") {
				return nil, errors.New("log directory is unreadable")
			}
			return []LogSummary{{Name: "apply.log", Path: readDir + "/apply.log"}}, nil
		},
		ContentReader: func(string, int64, int) (string, error) { return "log body", nil },
	})
	model.SetLogDir("/logs/first")

	staleCmd := model.StartLogRead(0)
	staleResult := b10Start(staleCmd)
	// Change the directory: the in-flight result no longer belongs to the model.
	model.SetLogDir("/logs/second")
	liveCmd := model.StartLogRead(0)
	liveResult := b10Start(liveCmd)

	close(release)
	stale := (<-staleResult).(ActivityLogReadMsg)
	if model.ApplyLogRead(stale) {
		t.Fatal("a result for the previous log directory was accepted")
	}
	if !model.loading {
		t.Fatal("a stale result cleared the live loading state")
	}
	if model.LogContent() != "" {
		t.Fatalf("stale content was applied: %q", model.LogContent())
	}

	live := (<-liveResult).(ActivityLogReadMsg)
	if !model.ApplyLogRead(live) {
		t.Fatal("the live read was rejected")
	}
	if model.loading || !model.loaded {
		t.Fatalf("live read state = loading:%v loaded:%v", model.loading, model.loaded)
	}
	if live.LogDir != "/logs/second" {
		t.Fatalf("live read was bound to %q, want /logs/second", live.LogDir)
	}
	if got := atomic.LoadInt32(&reads); got != 2 {
		t.Fatalf("log reads = %d, want 2", got)
	}

	// A failed live read is visible and does not claim an empty directory.
	model.SetLogDir("/logs/broken")
	failCmd := model.StartLogRead(0)
	// release is already closed, so this read returns its error immediately.
	failResult := b10Start(failCmd)
	failed := (<-failResult).(ActivityLogReadMsg)
	if !model.ApplyLogRead(failed) {
		t.Fatal("the failing read was rejected as stale")
	}
	frame := strings.Join(model.Lines(100), "\n")
	if !strings.Contains(frame, "Error reading logs: log directory is unreadable") {
		t.Fatalf("log read error is not visible:\n%s", frame)
	}
}

// b10Unbatch returns the commands a queued batch will run. A Batch command only
// yields its children, so this inspects it instead of running a read; a command
// that blocks instead is a test bug and fails rather than hangs.
func b10Unbatch(t *testing.T, cmd tea.Cmd) []tea.Cmd {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command was queued")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if batch, ok := msg.(tea.BatchMsg); ok {
			return []tea.Cmd(batch)
		}
		t.Fatalf("queued command returned %T, want a BatchMsg of unexecuted reads", msg)
	case <-time.After(5 * time.Second):
		// A batch never blocks; a read that does means the command was executed
		// instead of inspected, and the guard keeps the test from hanging.
		t.Fatal("queued command blocked instead of yielding its children")
	}
	return nil
}

// b10Collect runs every command concurrently.
func b10Collect(cmds []tea.Cmd) chan tea.Msg {
	out := make(chan tea.Msg, len(cmds))
	for _, cmd := range cmds {
		go func(c tea.Cmd) { out <- c() }(cmd)
	}
	return out
}

// TestB10DeferredInventoryRead covers AC-21 for the inventory source: the
// launcher's deferred read is the only initial reader, the first frame reports
// the pending read, the accepted result updates the setup state, and a stale
// generation is ignored.
func TestB10DeferredInventoryRead(t *testing.T) {
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()

	var loads int32
	release := make(chan struct{})
	entered := make(chan struct{}, 4)
	loaded := b9CoreCatalog(b9All("matches source"))
	app := NewAppWithScreens(cfg, Catalog{}, AppOptions{DeferredInventory: true}, AppScreenOptions{
		CatalogRefresh: func(readCfg TUIConfig) (Catalog, error) {
			atomic.AddInt32(&loads, 1)
			entered <- struct{}{}
			<-release
			out := loaded
			out.Home = readCfg.Home
			return out, nil
		},
	})

	if got := atomic.LoadInt32(&loads); got != 0 {
		t.Fatalf("construction loaded the inventory %d time(s), want 0", got)
	}
	unreadFrame := app.View().Content
	if strings.Contains(unreadFrame, "the state above is from the last successful read") {
		t.Fatalf("the first frame claims a previous successful read:\n%s", unreadFrame)
	}

	// The launcher's first message queues the deferred read, after the screen
	// seams are installed, and the pending notice makes the frame truthful.
	app, startCmd := b10UpdateApp(t, app, appStartMsg{})
	if startCmd == nil {
		t.Fatal("the start message queued no read")
	}
	if got := atomic.LoadInt32(&loads); got != 0 {
		t.Fatalf("the deferred read ran inside Update %d time(s)", got)
	}
	if !app.inventoryPending {
		t.Fatal("the deferred session does not report a pending inventory read")
	}
	firstFrame := app.View().Content
	if !strings.Contains(firstFrame, "Reading the local inventory") {
		t.Fatalf("first frame does not report the pending inventory read:\n%s", firstFrame)
	}
	if strings.Contains(firstFrame, "the state above is from the last successful read") {
		t.Fatalf("first frame claims a previous successful read:\n%s", firstFrame)
	}

	initialCmds := b10Unbatch(t, startCmd)
	if len(initialCmds) != 2 {
		t.Fatalf("queued reads = %d, want the inventory read and the dispatch-free activity read", len(initialCmds))
	}
	initial := b10Collect(initialCmds)
	<-entered

	// A second, newer read makes the first one stale. Its scope is recorded on
	// the app below, so the stale result can never satisfy it.
	newer := app.startInventoryRead(app.inventoryScope)
	newerMsgs := b10Collect([]tea.Cmd{newer})
	<-entered

	close(release)
	var refreshes []CatalogRefreshMsg
	for len(refreshes) < 2 {
		select {
		case msg := <-initial:
			if refresh, ok := msg.(CatalogRefreshMsg); ok {
				refreshes = append(refreshes, refresh)
			}
		case msg := <-newerMsgs:
			if refresh, ok := msg.(CatalogRefreshMsg); ok {
				refreshes = append(refreshes, refresh)
			}
		}
	}
	if len(refreshes) != 2 {
		t.Fatalf("inventory refreshes = %d, want 2", len(refreshes))
	}
	stale, live := refreshes[0], refreshes[1]
	if stale.generation > live.generation {
		stale, live = live, stale
	}

	afterStale, _ := b10UpdateApp(t, app, stale)
	if !afterStale.inventoryPending {
		t.Fatal("a stale inventory result cleared the live pending state")
	}
	if afterStale.catalog.SkillCount() != 0 {
		t.Fatalf("a stale inventory generation was applied: %d skill(s)", afterStale.catalog.SkillCount())
	}

	afterLive, _ := b10UpdateApp(t, afterStale, live)
	if afterLive.inventoryPending {
		t.Fatal("the accepted inventory read left the session pending")
	}
	if !afterLive.inventoryLoaded {
		t.Fatal("the accepted inventory read did not record a successful read")
	}
	if afterLive.catalog.SkillCount() == 0 {
		t.Fatalf("the accepted inventory was not applied: %+v", afterLive.catalog)
	}
	appliedFrame := afterLive.View().Content
	if strings.Contains(appliedFrame, "Reading the local inventory") {
		t.Fatalf("pending notice survived the applied read:\n%s", appliedFrame)
	}
}

// TestB10DeferredInventoryReadErrorIsVisible proves a failed initial inventory
// read is reported as the Home setup state instead of aborting the launch.
func TestB10DeferredInventoryReadErrorIsVisible(t *testing.T) {
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()
	app := NewAppWithScreens(cfg, Catalog{}, AppOptions{DeferredInventory: true}, AppScreenOptions{
		CatalogRefresh: func(TUIConfig) (Catalog, error) {
			return Catalog{}, errors.New("home inventory is unreadable")
		},
	})

	app, initialCmd := b10UpdateApp(t, app, appStartMsg{})
	if initialCmd == nil {
		t.Fatal("the start message queued no inventory read")
	}
	applied, frame := runB9App(t, app, initialCmd)
	if !strings.Contains(frame, "home inventory is unreadable") && !strings.Contains(frame, "Inventory read failed") {
		t.Fatalf("inventory read error is not visible:\n%s", frame)
	}
	if applied.inventoryPending {
		t.Fatal("a failed inventory read left the session pending")
	}
	if got := DeriveSetupState(applied.catalog, applied.inventoryErr).Kind; got != SetupStateReadFailure {
		t.Fatalf("setup state = %v, want read failure", got)
	}
}

// runB9App applies one command's messages to the app and returns the updated
// shell with its rendered view.
func runB9App(t *testing.T, app App, cmd tea.Cmd) (App, string) {
	t.Helper()
	for _, msg := range runB9Cmd(cmd) {
		app, _ = b10UpdateApp(t, app, msg)
	}
	return app, app.View().Content
}

// TestB10IndependentInstancesShareNoState covers AC-19: two independently
// configured shells keep their own seams, homes, drafts and read results.
func TestB10IndependentInstancesShareNoState(t *testing.T) {
	homeA := t.TempDir()
	homeB := t.TempDir()

	readerA := &fakeMCPReader{statusRes: MCPStatusView{Available: true, Home: homeA, ConfigPath: "/a/opencode.json"}}
	readerB := &fakeMCPReader{statusRes: MCPStatusView{Available: true, Home: homeB, ConfigPath: "/b/opencode.json"}}

	cfgA := TUIConfig{Width: 100, Height: 30, Home: homeA}.Normalize()
	cfgB := TUIConfig{Width: 100, Height: 30, Home: homeB}.Normalize()
	appA := NewAppWithScreens(cfgA, Catalog{Home: homeA}, AppOptions{}, AppScreenOptions{MCPReader: readerA, MCPPlanner: &fakeMCPPlanner{}})
	appB := NewAppWithScreens(cfgB, Catalog{Home: homeB}, AppOptions{}, AppScreenOptions{MCPReader: readerB, MCPPlanner: &fakeMCPPlanner{}})

	// A draft edit in one instance is not visible in the other.
	appA.mcp.selected["exa"] = false
	appA.mcp.confirmSelection = true
	appA.mcp.exaAcknowledged = true
	appA.mcp.filter = "context"
	if !appB.mcp.selected["exa"] || appB.mcp.confirmSelection || appB.mcp.exaAcknowledged || appB.mcp.filter != "" {
		t.Fatalf("instances share draft state: A=%+v B=%+v", appA.mcp.selected, appB.mcp.selected)
	}

	// Each instance reads through its own seam and its own home.
	mA, cmdA := appA.mcp.EnsureLoaded()
	msgA := cmdA().(MCPStatusMsg)
	if msgA.Home != homeA {
		t.Fatalf("instance A read home %q, want %q", msgA.Home, homeA)
	}
	appA.mcp, _ = mA.Update(msgA)
	mB, cmdB := appB.mcp.EnsureLoaded()
	msgB := cmdB().(MCPStatusMsg)
	if msgB.Home != homeB {
		t.Fatalf("instance B read home %q, want %q", msgB.Home, homeB)
	}
	appB.mcp, _ = mB.Update(msgB)

	if appA.mcp.status.ConfigPath != "/a/opencode.json" || appB.mcp.status.ConfigPath != "/b/opencode.json" {
		t.Fatalf("instances shared a read result: A=%q B=%q",
			appA.mcp.status.ConfigPath, appB.mcp.status.ConfigPath)
	}
	if len(readerA.calls) != 1 || len(readerB.calls) != 1 {
		t.Fatalf("seam calls A=%v B=%v, want one each", readerA.calls, readerB.calls)
	}
}

// TestB10MCPReadContextIsInstanceScoped proves the cooperative read context is
// instance state: cancelling one session stops its next read from starting and
// leaves an independent session working.
func TestB10MCPReadContextIsInstanceScoped(t *testing.T) {
	home := t.TempDir()
	cfg := TUIConfig{Width: 100, Height: 30, Home: home}.Normalize()
	app := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{})

	// The session context is created per App instance and is never shared.
	if app.readCtx == nil || app.cancelRead == nil {
		t.Fatal("the session did not create its own read context")
	}
	other := NewAppWithScreens(cfg, Catalog{Home: home}, AppOptions{}, AppScreenOptions{})
	if other.readCtx == app.readCtx {
		t.Fatal("two sessions share one read context")
	}

	probe := &b10ContextProbe{}
	app.mcp = NewMCPModelWithSeams(cfg, probe, &fakeMCPPlanner{})
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	app.mcp.SetReadContext(cancelled)
	_, cmd := app.mcp.EnsureLoaded()
	if cmd == nil {
		t.Fatal("no read command was produced")
	}
	msg, ok := cmd().(MCPStatusMsg)
	if !ok {
		t.Fatalf("read command returned %T", msg)
	}
	if !errors.Is(probe.saw, context.Canceled) {
		t.Fatalf("seam received context %v, want a cancelled one", probe.saw)
	}
	if !errors.Is(msg.Err, context.Canceled) {
		t.Fatalf("cancelled session read error = %v, want context.Canceled", msg.Err)
	}

	app.mcp = NewMCPModelWithSeams(cfg, &fakeMCPReader{statusRes: MCPStatusView{Available: true, Home: home}}, &fakeMCPPlanner{})
	app.mcp.SetReadContext(other.readCtx)
	_, liveCmd := app.mcp.EnsureLoaded()
	liveMsg := liveCmd().(MCPStatusMsg)
	if liveMsg.Err != nil {
		t.Fatalf("independent session read failed: %v", liveMsg.Err)
	}
}

// b10ContextProbe records the context state a read seam actually receives.
type b10ContextProbe struct {
	saw error
}

func (p *b10ContextProbe) Status(ctx context.Context, _, _, _ string) (MCPStatusView, error) {
	p.saw = ctx.Err()
	return MCPStatusView{}, ctx.Err()
}

func (p *b10ContextProbe) Doctor(context.Context, string, string, string, string, string) (MCPDoctorView, error) {
	return MCPDoctorView{}, nil
}
