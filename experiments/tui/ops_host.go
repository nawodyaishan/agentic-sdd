package tui

// This file owns the nested TUI's single-write operation host. It is the only
// place where the shell starts a write transaction:
//
//   - BeginWrite claims the one write slot and returns a tea.Cmd that runs the
//     operation outside Update/View;
//   - the typed start/progress/result messages below carry the
//     appops.OperationID assigned to that transaction;
//   - a second BeginWrite while one is active is refused, never queued;
//   - the result is emitted only after the transcript has been flushed and
//     closed, so the root model can defer an exit until then.
//
// The engine contracts stay in internal/appops: this host reuses the immutable
// InstallRequest/RestoreRequest, the preview token and the structured
// ApplyResult. It never parses transcripts, never invents a percentage and never
// touches a destination itself.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// WritePolicyText is the execution policy shown before a write is confirmed.
// It is deliberately one text constant so confirmation screens, help and the
// busy footer cannot drift apart.
const WritePolicyText = "Writes cannot be abandoned once they start. q, Esc and navigation are ignored while a write runs. Ctrl+C, SIGINT, SIGTERM or SIGHUP requests exit only after the write and its operation log are complete. A permanently wedged write can only be stopped with SIGKILL, which cannot guarantee terminal cleanup."

// WriteBusyMarker is a stable ASCII marker the busy footer always contains.
// Lifecycle tests use it to know a write is genuinely in flight; it never
// replaces a text label with a symbol.
const WriteBusyMarker = "WRITE-IN-PROGRESS"

// ErrWriteInProgress reports that a write transaction is already running.
var ErrWriteInProgress = errors.New("a write is already running")

// ErrWriteInvalid reports a write request that cannot be started safely.
var ErrWriteInvalid = errors.New("write request is invalid")

// WriteKind identifies which engine operation a write transaction runs.
type WriteKind int

const (
	// WriteInstall runs one revalidated install apply.
	WriteInstall WriteKind = iota + 1
	// WriteRestore runs one revalidated restore apply.
	WriteRestore
)

// String returns the engine-style verb for display and log headers.
func (k WriteKind) String() string {
	switch k {
	case WriteInstall:
		return "apply"
	case WriteRestore:
		return "restore"
	default:
		return "unknown"
	}
}

// WriteRequest is one approved write transaction. It is constructed from the
// immutable appops request types, so the destination, selection and engine
// options cannot change after confirmation. Token is the preview token the
// apply seam revalidates before writing.
type WriteRequest struct {
	Kind    WriteKind
	Install appops.InstallRequest
	Restore appops.RestoreRequest
	Token   string
	// Command is the transcript header, for example "apply --with-opencode".
	Command string
	// Label is a short human-readable description of the confirmed change set.
	Label string
}

// home returns the destination home the request was resolved against.
func (r WriteRequest) home() string {
	switch r.Kind {
	case WriteInstall:
		return r.Install.Home()
	case WriteRestore:
		return r.Restore.Home()
	default:
		return ""
	}
}

// source returns the transcript source identity, or "embedded".
func (r WriteRequest) source() string {
	var repo string
	switch r.Kind {
	case WriteInstall:
		repo = r.Install.Repo()
	case WriteRestore:
		repo = r.Restore.Repo()
	}
	if repo == "" {
		return "embedded"
	}
	return repo
}

// WriteStep is a real phase the host performs. There is no percentage: the
// engine does not report partial progress, so the UI must not invent it.
type WriteStep string

const (
	// WriteStepStarting means the operation log is open and the engine is about
	// to be called.
	WriteStepStarting WriteStep = "starting"
	// WriteStepWriting means the apply engine call is running.
	WriteStepWriting WriteStep = "writing"
	// WriteStepFlushing means the engine has returned and its transcript is
	// being flushed and closed.
	WriteStepFlushing WriteStep = "flushing"
)

// WriteStartedMsg is the typed start message for one accepted write. Its
// identity is assigned before any command runs, so the root model can refuse
// input from that point onward.
type WriteStartedMsg struct {
	ID        appops.OperationID
	Kind      WriteKind
	Label     string
	StartedAt time.Time
}

// WriteProgressMsg is an honest phase update for one active write. Detail is
// optional context; it never contains a percentage.
type WriteProgressMsg struct {
	ID     appops.OperationID
	Kind   WriteKind
	Step   WriteStep
	Detail string
}

// WriteResultMsg is the single terminal message for one write. It is emitted
// only after the operation log has been flushed and closed, so an exit-after-
// completion policy can quit immediately on receipt without losing the log.
type WriteResultMsg struct {
	ID      appops.OperationID
	Kind    WriteKind
	Label   string
	Result  appops.ApplyResult
	Err     error
	LogPath string
	LogErr  error
	// FinishedAt is when the engine returned, before the log flush finished.
	FinishedAt time.Time
}

// WriteRefusedMsg reports that a requested write never started. It carries no
// OperationID because no transaction was accepted.
type WriteRefusedMsg struct {
	Kind   WriteKind
	Reason string
	Err    error
}

// StartWriteMsg asks the root model to begin one confirmed write transaction.
// Screens emit it after affirmative confirmation of the current preview token;
// they never call the engine directly.
type StartWriteMsg struct {
	Request WriteRequest
}

// ShutdownReason identifies why an exit was requested.
type ShutdownReason int

const (
	// ShutdownReasonQuitKey is an idle q key.
	ShutdownReasonQuitKey ShutdownReason = iota + 1
	// ShutdownReasonInterruptKey is a Ctrl+C key event.
	ShutdownReasonInterruptKey
	// ShutdownReasonSignal is SIGINT, SIGTERM, SIGHUP or the platform fallback.
	ShutdownReasonSignal
)

// String returns a stable display label.
func (r ShutdownReason) String() string {
	switch r {
	case ShutdownReasonQuitKey:
		return "quit key"
	case ShutdownReasonInterruptKey:
		return "Ctrl+C"
	case ShutdownReasonSignal:
		return "OS signal"
	default:
		return "shutdown"
	}
}

// ShutdownRequestMsg routes one key or OS signal through the same busy-state
// policy. Repeated messages are allowed; the root model treats repeats as
// idempotent requests rather than a hard kill.
type ShutdownRequestMsg struct {
	Reason ShutdownReason
	// Signal is the OS signal name for ShutdownReasonSignal, for example
	// "SIGTERM".
	Signal string
}

// signalRegistrar is the narrow seam around os/signal so tests can prove
// registration, forwarding and stopping without delivering real signals.
type signalRegistrar interface {
	Notify(chan<- os.Signal, ...os.Signal)
	Stop(chan<- os.Signal)
}

// osSignalRegistrar is the production registrar.
type osSignalRegistrar struct{}

func (osSignalRegistrar) Notify(ch chan<- os.Signal, sig ...os.Signal) { signal.Notify(ch, sig...) }
func (osSignalRegistrar) Stop(ch chan<- os.Signal)                     { signal.Stop(ch) }

// SignalHandler forwards OS shutdown signals as ShutdownRequestMsg values
// until Stop is called. Every received signal is forwarded, including repeats:
// the root model decides that a repeat never force-kills a write.
type SignalHandler struct {
	signals   chan os.Signal
	registrar signalRegistrar

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// NotifyShutdownSignals registers the platform's shutdown signals before
// Program.Run starts and forwards each one through send (usually
// program.Send). The caller must invoke Stop on every exit path.
func NotifyShutdownSignals(send func(tea.Msg)) *SignalHandler {
	return newSignalHandler(shutdownSignals(), osSignalRegistrar{}, send)
}

// newSignalHandler builds a handler around a fresh signal channel. Tests use a
// fake registrar to send synthetic signals through the same forwarding path.
func newSignalHandler(signals []os.Signal, registrar signalRegistrar, send func(tea.Msg)) *SignalHandler {
	channel := make(chan os.Signal, len(signals)+1)
	handler := &SignalHandler{
		signals:   channel,
		registrar: registrar,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	registrar.Notify(channel, signals...)

	go func() {
		defer close(handler.done)
		for {
			select {
			case sig, ok := <-channel:
				if !ok {
					return
				}
				if send == nil {
					continue
				}
				send(ShutdownRequestMsg{
					Reason: ShutdownReasonSignal,
					Signal: shutdownSignalName(sig),
				})
			case <-handler.stop:
				return
			}
		}
	}()
	return handler
}

// Stop unregisters delivery and waits for the forwarding goroutine to finish.
// It is idempotent: repeated calls do not double-stop or deadlock, so normal
// exit, handled errors and handled signals can all call it.
func (h *SignalHandler) Stop() {
	if h == nil {
		return
	}
	h.stopOnce.Do(func() {
		// os/signal guarantees no delivery after Stop returns, so the
		// forwarding goroutine can be stopped without draining a signal.
		h.registrar.Stop(h.signals)
		close(h.stop)
		<-h.done
	})
}

// TerminalRelease serializes the host's own fallback terminal restoration. On
// a normal Program.Run return Bubble Tea has already restored the terminal, so
// the caller disarms this path to avoid a second restoration. On a returned
// error the deferred Release runs at most once, covering launch failures that
// happen before Bubble Tea's internal shutdown. It is safe for normal exit,
// handled error and handled signal paths to all defer Release.
type TerminalRelease struct {
	mu      sync.Mutex
	release func() error
	armed   bool
	done    bool
	err     error
}

// NewTerminalRelease wraps the program's ReleaseTerminal method.
func NewTerminalRelease(release func() error) *TerminalRelease {
	return &TerminalRelease{release: release, armed: release != nil}
}

// Disarm records that the program's own cleanup already restored the terminal.
func (t *TerminalRelease) Disarm() {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.armed = false
	t.mu.Unlock()
}

// Release performs the fallback restoration at most once and caches its error.
func (t *TerminalRelease) Release() error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.armed || t.done {
		return t.err
	}
	t.done = true
	t.err = t.release()
	return t.err
}

// WriteStatusView is an immutable snapshot of the active write for display.
type WriteStatusView struct {
	ID        appops.OperationID
	Kind      WriteKind
	Label     string
	StartedAt time.Time
	Step      WriteStep
	Detail    string
}

// InstallApplier runs one revalidated install and returns its structured
// result. It is a narrow function seam, not an interface hierarchy.
type InstallApplier func(req appops.InstallRequest, token string, out io.Writer) (appops.ApplyResult, error)

// RestoreApplier runs one revalidated restore.
type RestoreApplier func(req appops.RestoreRequest, token string, out io.Writer) (appops.ApplyResult, error)

// WriteLog is one write transcript destination. Flush must persist buffered
// data; Close must always run, including after a failed write.
type WriteLog interface {
	io.Writer
	Flush() error
	Close() error
}

// WriteLogFactory opens the transcript sink for one write. It runs inside the
// operation command, never in Update/View. path is displayed to the user.
type WriteLogFactory func(kind WriteKind, home string) (log WriteLog, path string, err error)

// OpsHostOptions injects the host's narrow seams. The zero value uses the
// production appops engine and the private file logger.
type OpsHostOptions struct {
	ApplyInstall InstallApplier
	ApplyRestore RestoreApplier
	OpenLog      WriteLogFactory
	Now          func() time.Time
}

// OpsHost owns the single write slot. It is safe to share through App copies;
// BeginWrite, Complete and progress delivery synchronize on its mutex.
type OpsHost struct {
	mu           sync.Mutex
	nextRequest  uint64
	active       *WriteStatusView
	completed    appops.OperationID
	hasCompleted bool

	applyInstall InstallApplier
	applyRestore RestoreApplier
	openLog      WriteLogFactory
	now          func() time.Time
	progressSink func(tea.Msg)
}

// NewOpsHost builds one host with the supplied seams. Nil seams fall back to
// the production implementation.
func NewOpsHost(opts OpsHostOptions) *OpsHost {
	host := &OpsHost{
		applyInstall: opts.ApplyInstall,
		applyRestore: opts.ApplyRestore,
		openLog:      opts.OpenLog,
		now:          opts.Now,
	}
	if host.applyInstall == nil {
		host.applyInstall = appops.ApplyInstall
	}
	if host.applyRestore == nil {
		host.applyRestore = appops.ApplyRestore
	}
	if host.openLog == nil {
		host.openLog = openFileWriteLog
	}
	if host.now == nil {
		host.now = time.Now
	}
	return host
}

// SetProgressSink routes progress messages to the running program, usually
// program.Send. It is optional: without a sink the phases are still enforced,
// they are just not rendered.
func (h *OpsHost) SetProgressSink(sink func(tea.Msg)) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.progressSink = sink
	h.mu.Unlock()
}

// ActiveWrite returns a snapshot of the active write, if any.
func (h *OpsHost) ActiveWrite() (WriteStatusView, bool) {
	if h == nil {
		return WriteStatusView{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active == nil {
		return WriteStatusView{}, false
	}
	return *h.active, true
}

// BeginWrite claims the single write slot and returns the typed start message
// plus the command that performs the operation. It performs no filesystem or
// network I/O: the log is opened inside the command. A second call while a
// write is active returns ErrWriteInProgress and no transaction is queued.
func (h *OpsHost) BeginWrite(req WriteRequest) (WriteStartedMsg, tea.Cmd, error) {
	if h == nil {
		return WriteStartedMsg{}, nil, fmt.Errorf("%w: no operation host", ErrWriteInvalid)
	}
	if err := req.validate(); err != nil {
		return WriteStartedMsg{}, nil, err
	}

	h.mu.Lock()
	if h.active != nil {
		active := *h.active
		h.mu.Unlock()
		return WriteStartedMsg{}, nil, fmt.Errorf("%w: %s %v is still running", ErrWriteInProgress, active.Kind, active.ID)
	}
	h.nextRequest++
	id := appops.OperationID{Request: h.nextRequest, Generation: 1}
	started := WriteStartedMsg{
		ID:        id,
		Kind:      req.Kind,
		Label:     req.Label,
		StartedAt: h.now(),
	}
	status := &WriteStatusView{
		ID:        id,
		Kind:      req.Kind,
		Label:     req.Label,
		StartedAt: started.StartedAt,
	}
	h.active = status
	// Capture the seams and the home under the lock so the command never reads
	// mutable host state while Update runs.
	applyInstall := h.applyInstall
	applyRestore := h.applyRestore
	openLog := h.openLog
	now := h.now
	home := req.home()
	h.mu.Unlock()

	cmd := func() tea.Msg {
		return h.run(req, id, home, openLog, now, applyInstall, applyRestore)
	}
	return started, cmd, nil
}

// Complete releases the write slot when the root model accepts the matching
// result message. A mismatched or repeated identity changes nothing and
// reports false, so a stale result cannot clear a newer transaction.
func (h *OpsHost) Complete(id appops.OperationID) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.active == nil || h.active.ID != id {
		return false
	}
	h.active = nil
	h.completed = id
	h.hasCompleted = true
	return true
}

// Completed returns the identity of the last accepted write result.
func (h *OpsHost) Completed() (appops.OperationID, bool) {
	if h == nil {
		return appops.OperationID{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.completed, h.hasCompleted
}

// validate rejects a request that cannot be executed safely before any slot is
// claimed. The apply seam repeats the token check; this is the first line.
func (r WriteRequest) validate() error {
	switch r.Kind {
	case WriteInstall, WriteRestore:
	default:
		return fmt.Errorf("%w: unknown operation kind", ErrWriteInvalid)
	}
	if r.Token == "" {
		return fmt.Errorf("%w: a preview token is required", ErrWriteInvalid)
	}
	if r.home() == "" {
		return fmt.Errorf("%w: a resolved home is required", ErrWriteInvalid)
	}
	return nil
}

// run executes one claimed write. It is called from a tea.Cmd goroutine only.
func (h *OpsHost) run(
	req WriteRequest,
	id appops.OperationID,
	home string,
	openLog WriteLogFactory,
	now func() time.Time,
	applyInstall InstallApplier,
	applyRestore RestoreApplier,
) WriteResultMsg {
	result := WriteResultMsg{ID: id, Kind: req.Kind, Label: req.Label, FinishedAt: now()}

	log, path, err := openLog(req.Kind, home)
	if err != nil {
		result.Err = fmt.Errorf("open operation log: %w", err)
		result.FinishedAt = now()
		return result
	}
	result.LogPath = path
	command := req.Command
	if command == "" {
		command = req.Kind.String()
	}
	_, _ = fmt.Fprintf(log, "Started: %s\nCommand: %s\nSource: %s\nHome: %s\n",
		result.FinishedAt.UTC().Format(time.RFC3339Nano), command, req.source(), home)

	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepStarting})
	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepWriting, Detail: command})

	switch req.Kind {
	case WriteInstall:
		result.Result, result.Err = applyInstall(req.Install, req.Token, log)
	case WriteRestore:
		result.Result, result.Err = applyRestore(req.Restore, req.Token, log)
	}
	result.FinishedAt = now()

	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepFlushing, Detail: path})

	outcome := "success"
	if result.Err != nil {
		outcome = "failure"
		_, _ = fmt.Fprintf(log, "error: %v\n", result.Err)
	}
	_, _ = fmt.Fprintf(log, "Result: %s\nFinished: %s\n", outcome, now().UTC().Format(time.RFC3339Nano))
	flushErr := log.Flush()
	closeErr := log.Close()
	result.LogErr = errors.Join(flushErr, closeErr)
	if result.LogErr != nil {
		result.LogErr = fmt.Errorf("operation log %s: %w", path, result.LogErr)
	}
	return result
}

// emit delivers one progress message when a sink is installed. The sink is
// copied under the lock so Update and the command goroutine never race.
func (h *OpsHost) emit(msg tea.Msg) {
	if h == nil {
		return
	}
	h.mu.Lock()
	sink := h.progressSink
	h.mu.Unlock()
	if sink == nil {
		return
	}
	sink(msg)
}

// fileWriteLog is the production transcript sink: a private regular file in
// the shared log directory, flushed and closed by the host.
type fileWriteLog struct {
	file *os.File
}

func (f *fileWriteLog) Write(p []byte) (int, error) { return f.file.Write(p) }
func (f *fileWriteLog) Flush() error                { return f.file.Sync() }
func (f *fileWriteLog) Close() error                { return f.file.Close() }

// openFileWriteLog mirrors the CLI's private log location and path checks
// without importing package main. The nested module owns no log discovery
// policy; it only writes the transcript it is given here.
func openFileWriteLog(_ WriteKind, home string) (WriteLog, string, error) {
	if home == "" {
		return nil, "", errors.New("write log home is required")
	}
	dir := filepath.Join(home, ".agentic-sdd", "logs")
	for _, path := range []string{home, filepath.Join(home, ".agentic-sdd"), dir} {
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		if !info.IsDir() {
			return nil, "", fmt.Errorf("refusing non-directory log path %s", path)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	file, err := os.CreateTemp(dir, stamp+"-*.log")
	if err != nil {
		return nil, "", err
	}
	return &fileWriteLog{file: file}, file.Name(), nil
}

// AppOptions injects the root model's host seams. The zero value uses the
// production operation host and performs no cancellable read work.
type AppOptions struct {
	// Ops is the shared single-write host. Nil builds the production host.
	Ops *OpsHost
	// CancelReads cancels cooperative background read work when an idle
	// shutdown begins. It is called at most once.
	CancelReads func()
	// InitialWrite claims the write slot before the program starts. The host
	// and PTY lifecycle harnesses use it; a normal interactive app leaves it
	// nil.
	InitialWrite *WriteRequest
}

// NewAppWithOptions constructs the root shell with injected host seams. A nil
// Ops uses the production engine; an InitialWrite claims the single write slot
// before Init and queues its start/result messages for the first command batch.
func NewAppWithOptions(cfg TUIConfig, catalog Catalog, opts AppOptions) App {
	app := newApp(cfg, catalog)
	app.ops = opts.Ops
	if app.ops == nil {
		app.ops = NewOpsHost(OpsHostOptions{})
	}
	app.cancelReads = opts.CancelReads
	if opts.InitialWrite != nil {
		started, cmd, err := app.ops.BeginWrite(*opts.InitialWrite)
		if err != nil {
			app.notice = err.Error()
			return app
		}
		app.acceptStarted(started)
		app.initCmds = append(app.initCmds,
			func() tea.Msg { return started },
			cmd,
		)
	}
	return app
}

// SummarizeWriteResult classifies one write result for display without
// inventing states the engine did not report. A write is never "cancelled":
// confirmed writes either finish with a structured outcome or fail, and the
// residual wedged-process limitation is documented by WritePolicyText.
type WriteSummary struct {
	Headline     string
	Detail       string
	Outcome      skillsync.Outcome
	BackupID     string
	LogPath      string
	NeedsPreview bool
	LogFailed    bool
	Rollback     string
}

// SummarizeWriteResult turns the structured result into text-labelled states.
func SummarizeWriteResult(msg WriteResultMsg) WriteSummary {
	summary := WriteSummary{
		Outcome:  msg.Result.Outcome,
		BackupID: msg.Result.BackupID,
		LogPath:  msg.LogPath,
	}
	summary.NeedsPreview = errors.Is(msg.Err, appops.ErrNeedsPreview)
	summary.LogFailed = msg.LogErr != nil

	rollback := msg.Result.Rollback
	switch {
	case rollback.Attempted && rollback.Complete:
		summary.Rollback = "complete"
	case rollback.Attempted:
		summary.Rollback = "incomplete"
	default:
		summary.Rollback = "none"
	}

	switch {
	case summary.NeedsPreview:
		summary.Headline = "needs a fresh preview"
		summary.Detail = "Inputs, source identity or planned changes changed after confirmation. No files were written; preview and confirm again."
	case msg.Err != nil && rollback.Attempted && rollback.Complete:
		summary.Headline = "rolled back"
		summary.Detail = "The write failed and every committed change was undone."
	case msg.Err != nil && rollback.Attempted:
		summary.Headline = "incomplete"
		summary.Detail = "The write failed and rollback did not complete. Destinations may be partially changed; use the recorded backup and log to recover."
	case msg.Err != nil:
		summary.Headline = "failed"
		summary.Detail = "The write failed before any planned change was committed."
	case msg.Result.Outcome == skillsync.OutcomeApplied:
		summary.Headline = "completed"
		summary.Detail = fmt.Sprintf("Wrote %d planned changes.", len(msg.Result.Applied))
	case msg.Result.Outcome == skillsync.OutcomeNoChange:
		summary.Headline = "no changes"
		summary.Detail = "Every planned destination was already up to date."
	default:
		summary.Headline = "finished"
		summary.Detail = "The engine returned without a write outcome."
	}
	if summary.LogFailed {
		summary.Detail += " The operation log could not be flushed: " + msg.LogErr.Error()
	}
	return summary
}
