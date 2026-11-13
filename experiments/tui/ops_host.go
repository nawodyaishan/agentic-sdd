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
// ApplyResult. Fleet source-save, fleet apply and fleet restore run through the
// same single slot as install and restore: they are three more kinds on this one
// host, never a second writer. It never parses transcripts, never invents a
// percentage and never touches a destination itself.

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
	"agentic-sdd/internal/fleet"
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

// ErrFleetUnavailable reports a fleet write whose engine seam is not wired in
// this build. It is always a refusal: the host opens no transcript and calls no
// engine, so the operation can never look like a successful write.
var ErrFleetUnavailable = errors.New("fleet write seam is not wired in this build")

// ErrMCPUnavailable reports an MCP write whose engine seam is not wired in
// this build. It is always a refusal.
var ErrMCPUnavailable = errors.New("mcp write seam is not wired in this build")

// WriteKind identifies which engine operation a write transaction runs.
type WriteKind int

const (
	// WriteInstall runs one revalidated install apply.
	WriteInstall WriteKind = iota + 1
	// WriteRestore runs one revalidated restore apply.
	WriteRestore
	// WriteFleetApply runs one revalidated fleet deployment apply. The plan is
	// held in memory and identified by its plan ID, never by a plan file.
	WriteFleetApply
	// WriteFleetRestore restores one managed fleet backup.
	WriteFleetRestore
	// WriteSourceSave saves the personal fleet source file.
	WriteSourceSave
	// WriteMCPApply runs one revalidated MCP configuration apply.
	WriteMCPApply
	// WriteMCPRestore restores one managed MCP backup.
	WriteMCPRestore
)

// String returns the engine-style verb for display and log headers.
func (k WriteKind) String() string {
	switch k {
	case WriteInstall:
		return "apply"
	case WriteRestore:
		return "restore"
	case WriteFleetApply:
		return "fleet-apply"
	case WriteFleetRestore:
		return "fleet-restore"
	case WriteSourceSave:
		return "save-config"
	case WriteMCPApply:
		return "mcp-apply"
	case WriteMCPRestore:
		return "mcp-restore"
	default:
		return "unknown"
	}
}

// isFleet reports whether the kind is one of the fleet write kinds. All of them
// carry their payload in WriteRequest.Fleet and resolve their own home.
func (k WriteKind) isFleet() bool {
	switch k {
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		return true
	default:
		return false
	}
}

// isMCP reports whether the kind is one of the MCP write kinds. All of them
// carry their payload in WriteRequest.MCP and resolve their own home.
func (k WriteKind) isMCP() bool {
	switch k {
	case WriteMCPApply, WriteMCPRestore:
		return true
	default:
		return false
	}
}

// FleetWriteRequest is one confirmed fleet write. It is TUI-local: it carries
// only resolved identities the caller already obtained from the shared seam, so
// no screen can join, clean or invent a path here.
//
//   - Home is the resolved home used for the private operation log. It is
//     required even when the personal source lives outside it.
//   - ConfigPath is the selected personal fleet source file.
//   - PlanID identifies the in-memory plan a fleet apply was confirmed against.
//     It is an identity label, never a temporary plan file.
//   - Plan is the immutable in-memory plan the apply carries to the shared
//     appops seam. It is an identity payload, never a second plan file, and the
//     seam deep-copies and revalidates it before any write.
//   - BackupID identifies the managed fleet backup a restore was confirmed for.
//   - Token is the preview token the apply/restore/save seam revalidates.
//   - Label is a short human-readable description of the confirmed change set.
//   - Source is the display label for the fleet source; empty falls back to
//     ConfigPath so a fleet write is never recorded as "embedded".
type FleetWriteRequest struct {
	Home       string
	ConfigPath string
	PlanID     string
	Plan       *fleet.Plan
	BackupID   string
	Token      string
	Label      string
	Source     string
}

// MCPWriteRequest is one confirmed MCP configuration write. It is TUI-local:
// it carries only resolved identities obtained from the shared seam.
//
//   - Home is the resolved home used for the private operation log and backups.
//   - ConfigPath is the target OpenCode configuration file path.
//   - PlanID identifies the in-memory plan an MCP apply was confirmed against.
//   - Plan is the in-memory plan payload for the engine seam.
//   - BackupID identifies the managed MCP backup to restore.
//   - Token is the preview token the apply/restore seam revalidates.
//   - ExpectDigest is the base digest expected before mutation.
//   - OpenCodeBin is the optional local OpenCode executable probe path.
//   - Label is a short human-readable description of the confirmed change set.
//   - Source is the display label; empty falls back to "mcp:<ConfigPath>".
type MCPWriteRequest struct {
	Home         string
	ConfigPath   string
	PlanID       string
	Plan         *skillsync.MCPPlan
	BackupID     string
	Token        string
	ExpectDigest string
	OpenCodeBin  string
	Label        string
	Source       string
}

// WriteRequest is one approved write transaction. It is constructed from the
// immutable appops request types, so the destination, selection and engine
// options cannot change after confirmation. Token is the preview token the
// apply seam revalidates before writing.
type WriteRequest struct {
	Kind    WriteKind
	Install appops.InstallRequest
	Restore appops.RestoreRequest
	// Fleet carries the payload for WriteFleetApply, WriteFleetRestore and
	// WriteSourceSave. It must stay zero for install and restore.
	Fleet FleetWriteRequest
	// MCP carries the payload for WriteMCPApply and WriteMCPRestore.
	MCP   MCPWriteRequest
	Token string
	// Command is the transcript header, for example "apply --with-opencode".
	// For fleet and mcp kinds the header is derived and this must be empty or match it.
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
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		return r.Fleet.Home
	case WriteMCPApply, WriteMCPRestore:
		return r.MCP.Home
	default:
		return ""
	}
}

// token returns the preview token the matching engine seam revalidates.
func (r WriteRequest) token() string {
	if r.Kind.isFleet() {
		return r.Fleet.Token
	}
	if r.Kind.isMCP() {
		return r.MCP.Token
	}
	return r.Token
}

// command returns the transcript command header. Fleet and MCP headers are derived so
// they cannot drift from the confirmed identity.
func (r WriteRequest) command() string {
	if cmd := r.fleetCommand(); cmd != "" {
		return cmd
	}
	if cmd := r.mcpCommand(); cmd != "" {
		return cmd
	}
	if r.Command != "" {
		return r.Command
	}
	return r.Kind.String()
}

// fleetCommand returns the exact fleet command header for the request kind, or
// "" for a non-fleet kind.
func (r WriteRequest) fleetCommand() string {
	switch r.Kind {
	case WriteFleetApply:
		if r.Fleet.PlanID == "" {
			return "fleet apply"
		}
		return "fleet apply --plan " + r.Fleet.PlanID
	case WriteFleetRestore:
		if r.Fleet.BackupID == "" {
			return "fleet restore"
		}
		return "fleet restore " + r.Fleet.BackupID
	case WriteSourceSave:
		return "fleet config save"
	default:
		return ""
	}
}

// mcpCommand returns the exact mcp command header for the request kind, or
// "" for a non-mcp kind.
func (r WriteRequest) mcpCommand() string {
	switch r.Kind {
	case WriteMCPApply:
		if r.MCP.ConfigPath != "" {
			return "mcp apply --config " + r.MCP.ConfigPath
		}
		return "mcp apply"
	case WriteMCPRestore:
		if r.MCP.BackupID != "" {
			return "mcp restore " + r.MCP.BackupID
		}
		return "mcp restore"
	default:
		return ""
	}
}

// source returns the transcript source identity, or "embedded". Fleet and MCP writes
// always report their selected source and path, never "embedded".
func (r WriteRequest) source() string {
	switch r.Kind {
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		if r.Fleet.Source != "" {
			return r.Fleet.Source
		}
		return r.Fleet.ConfigPath
	case WriteMCPApply, WriteMCPRestore:
		if r.MCP.Source != "" {
			return r.MCP.Source
		}
		if r.MCP.ConfigPath != "" {
			return "mcp:" + r.MCP.ConfigPath
		}
		return "mcp"
	}
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

// FleetWriter runs one revalidated fleet write against the shared appops seam.
// The three fleet kinds share this signature because they share one write slot
// and one structured result; the seam itself decides which engine call runs.
type FleetWriter func(req FleetWriteRequest, token string, out io.Writer) (appops.ApplyResult, error)

// MCPWriter runs one revalidated MCP write against the shared seam.
type MCPWriter func(req MCPWriteRequest, token string, out io.Writer) (appops.ApplyResult, error)

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
// production appops engine and the private file logger for install, restore,
// fleet apply and fleet restore.
//
// Fleet source save has no production default: B4's native source-write surface
// is separately gated, so leaving SaveFleetSource nil fails that operation with
// ErrFleetUnavailable instead of reporting a write that never ran.
type OpsHostOptions struct {
	ApplyInstall InstallApplier
	ApplyRestore RestoreApplier
	// ApplyFleet runs one revalidated fleet deployment apply.
	ApplyFleet FleetWriter
	// ApplyFleetRestore runs one revalidated managed-backup restore.
	ApplyFleetRestore FleetWriter
	// SaveFleetSource saves the personal fleet source file.
	SaveFleetSource FleetWriter
	// ApplyMCP runs one revalidated MCP configuration apply.
	ApplyMCP MCPWriter
	// ApplyMCPRestore runs one revalidated managed MCP backup restore.
	ApplyMCPRestore MCPWriter
	OpenLog         WriteLogFactory
	Now             func() time.Time
}

// OpsHost owns the single write slot. It is safe to share through App copies;
// BeginWrite, Complete and progress delivery synchronize on its mutex.
type OpsHost struct {
	mu           sync.Mutex
	nextRequest  uint64
	active       *WriteStatusView
	completed    appops.OperationID
	hasCompleted bool

	applyInstall      InstallApplier
	applyRestore      RestoreApplier
	applyFleet        FleetWriter
	applyFleetRestore FleetWriter
	saveFleetSource   FleetWriter
	applyMCP          MCPWriter
	applyMCPRestore   MCPWriter
	openLog           WriteLogFactory
	now               func() time.Time
	progressSink      func(tea.Msg)
}

// NewOpsHost builds one host with the supplied seams. Nil install/restore/fleet
// apply/fleet restore/log seams fall back to the production implementation; a
// nil SaveFleetSource seam stays nil so the separately gated source save is
// refused rather than faked.
func NewOpsHost(opts OpsHostOptions) *OpsHost {
	host := &OpsHost{
		applyInstall:      opts.ApplyInstall,
		applyRestore:      opts.ApplyRestore,
		applyFleet:        opts.ApplyFleet,
		applyFleetRestore: opts.ApplyFleetRestore,
		saveFleetSource:   opts.SaveFleetSource,
		applyMCP:          opts.ApplyMCP,
		applyMCPRestore:   opts.ApplyMCPRestore,
		openLog:           opts.OpenLog,
		now:               opts.Now,
	}
	if host.applyInstall == nil {
		host.applyInstall = appops.ApplyInstall
	}
	if host.applyRestore == nil {
		host.applyRestore = appops.ApplyRestore
	}
	if host.applyFleet == nil {
		host.applyFleet = appOpsFleetApply
	}
	if host.applyFleetRestore == nil {
		host.applyFleetRestore = appOpsFleetRestore
	}
	if host.applyMCP == nil {
		host.applyMCP = appOpsMCPApply
	}
	if host.applyMCPRestore == nil {
		host.applyMCPRestore = appOpsMCPRestore
	}
	if host.openLog == nil {
		host.openLog = openFileWriteLog
	}
	if host.now == nil {
		host.now = time.Now
	}
	return host
}

// appOpsFleetApply is the production fleet apply seam. It rebuilds the shared
// immutable request from the confirmed in-memory plan and resolved home, then
// lets appops revalidate the preview token before writing. A malformed plan or
// home is a typed refusal with no engine call.
func appOpsFleetApply(req FleetWriteRequest, token string, out io.Writer) (appops.ApplyResult, error) {
	request, err := appops.NewFleetApplyRequest(req.Plan, req.Home)
	if err != nil {
		return appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed}, err
	}
	return appops.ApplyFleet(request, token, out)
}

// appOpsFleetRestore is the production managed-backup restore seam. It rebuilds
// the shared immutable request from the confirmed backup identity and resolved
// home, then revalidates the restore preview token before writing.
func appOpsFleetRestore(req FleetWriteRequest, token string, out io.Writer) (appops.ApplyResult, error) {
	request, err := appops.NewFleetRestoreApplyRequest(req.Home, req.BackupID)
	if err != nil {
		return appops.ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeFailed}, err
	}
	return appops.ApplyFleetRestore(request, token, out)
}

// appOpsMCPApply is the production MCP apply seam. It rebuilds the shared
// immutable request from the confirmed plan and resolved home, then lets
// appops revalidate the token and digest before writing.
func appOpsMCPApply(req MCPWriteRequest, token string, out io.Writer) (appops.ApplyResult, error) {
	request, err := appops.NewMCPApplyRequest(
		req.Home, req.ConfigPath, req.Plan, token, req.ExpectDigest, req.OpenCodeBin,
	)
	if err != nil {
		return appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed}, err
	}
	res, err := appops.MCPApply(request)
	base := appops.ApplyResult{Operation: skillsync.OperationApply, Outcome: skillsync.OutcomeFailed}
	if err != nil {
		return base, err
	}
	if res.Result == nil {
		return base, fmt.Errorf("mcp apply returned no result")
	}
	base.Outcome = skillsync.OutcomeApplied
	if res.Result.NoOp {
		base.Outcome = skillsync.OutcomeNoChange
	}
	base.BackupID = res.Result.BackupID
	base.BackupPath = res.Result.BackupPath
	base.SourceDigest = res.Result.InstalledDigest
	base.Rollback = res.Result.Rollback
	for _, name := range res.Result.Applied {
		base.Applied = append(base.Applied, skillsync.ResultChange{ID: name, Kind: "mcp", Action: skillsync.ChangeReplace})
	}
	return base, nil
}

// appOpsMCPRestore is the production managed-backup restore seam.
func appOpsMCPRestore(req MCPWriteRequest, token string, out io.Writer) (appops.ApplyResult, error) {
	request, err := appops.NewMCPRestoreRequest(
		req.Home, req.ConfigPath, req.BackupID, token, req.ExpectDigest,
	)
	if err != nil {
		return appops.ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeFailed}, err
	}
	res, err := appops.MCPRestore(request)
	base := appops.ApplyResult{Operation: skillsync.OperationRestore, Outcome: skillsync.OutcomeFailed}
	if err != nil {
		return base, err
	}
	if res.Result == nil {
		return base, fmt.Errorf("mcp restore returned no result")
	}
	base.Outcome = skillsync.OutcomeApplied
	if res.Result.NoOp {
		base.Outcome = skillsync.OutcomeNoChange
	}
	base.BackupID = res.Result.BackupID
	base.BackupPath = res.Result.BackupPath
	base.SourceDigest = res.Result.InstalledDigest
	base.Rollback = res.Result.Rollback
	for _, name := range res.Result.Applied {
		base.Applied = append(base.Applied, skillsync.ResultChange{ID: name, Kind: "mcp", Action: skillsync.ChangeReplace})
	}
	return base, nil
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
	seams := writeSeams{
		applyInstall:      h.applyInstall,
		applyRestore:      h.applyRestore,
		applyFleet:        h.applyFleet,
		applyFleetRestore: h.applyFleetRestore,
		saveFleetSource:   h.saveFleetSource,
		applyMCP:          h.applyMCP,
		applyMCPRestore:   h.applyMCPRestore,
		openLog:           h.openLog,
		now:               h.now,
	}
	home := req.home()
	h.mu.Unlock()

	cmd := func() tea.Msg {
		return h.run(req, id, home, seams)
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
		if r.Fleet != (FleetWriteRequest{}) {
			return fmt.Errorf("%w: %s does not take a fleet payload", ErrWriteInvalid, r.Kind)
		}
		if r.MCP != (MCPWriteRequest{}) {
			return fmt.Errorf("%w: %s does not take an mcp payload", ErrWriteInvalid, r.Kind)
		}
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		if r.MCP != (MCPWriteRequest{}) {
			return fmt.Errorf("%w: %s does not take an mcp payload", ErrWriteInvalid, r.Kind)
		}
		return r.validateFleet()
	case WriteMCPApply, WriteMCPRestore:
		if r.Fleet != (FleetWriteRequest{}) {
			return fmt.Errorf("%w: %s does not take a fleet payload", ErrWriteInvalid, r.Kind)
		}
		return r.validateMCP()
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

// validateFleet rejects a fleet request whose kind, identity or token does not
// match its payload. It never weakens the install/restore checks.
func (r WriteRequest) validateFleet() error {
	fleet := r.Fleet
	if fleet.Token == "" {
		return fmt.Errorf("%w: %s requires the resolved preview token", ErrWriteInvalid, r.Kind)
	}
	if r.Token != "" && r.Token != fleet.Token {
		return fmt.Errorf("%w: %s preview token does not match the fleet request", ErrWriteInvalid, r.Kind)
	}
	if fleet.Home == "" {
		return fmt.Errorf("%w: %s requires a resolved home", ErrWriteInvalid, r.Kind)
	}
	switch r.Kind {
	case WriteFleetApply:
		if fleet.ConfigPath == "" {
			return fmt.Errorf("%w: %s requires the fleet config path", ErrWriteInvalid, r.Kind)
		}
		if fleet.BackupID != "" {
			return fmt.Errorf("%w: %s does not take a backup identity", ErrWriteInvalid, r.Kind)
		}
		if fleet.PlanID == "" {
			return fmt.Errorf("%w: %s requires the plan identity", ErrWriteInvalid, r.Kind)
		}
	case WriteFleetRestore:
		if fleet.PlanID != "" || fleet.Plan != nil {
			return fmt.Errorf("%w: %s does not take a plan identity", ErrWriteInvalid, r.Kind)
		}
		if fleet.BackupID == "" {
			return fmt.Errorf("%w: %s requires the backup identity", ErrWriteInvalid, r.Kind)
		}
	case WriteSourceSave:
		if fleet.ConfigPath == "" {
			return fmt.Errorf("%w: %s requires the fleet config path", ErrWriteInvalid, r.Kind)
		}
		if fleet.PlanID != "" || fleet.Plan != nil || fleet.BackupID != "" {
			return fmt.Errorf("%w: %s takes neither a plan nor a backup identity", ErrWriteInvalid, r.Kind)
		}
	}
	if want := r.fleetCommand(); r.Command != "" && r.Command != want {
		return fmt.Errorf("%w: %s command header must be %q", ErrWriteInvalid, r.Kind, want)
	}
	return nil
}

// validateMCP rejects an MCP request whose kind, identity or token does not
// match its payload.
func (r WriteRequest) validateMCP() error {
	mcp := r.MCP
	if mcp.Token == "" {
		return fmt.Errorf("%w: %s requires the resolved preview token", ErrWriteInvalid, r.Kind)
	}
	if r.Token != "" && r.Token != mcp.Token {
		return fmt.Errorf("%w: %s preview token does not match the mcp request", ErrWriteInvalid, r.Kind)
	}
	if mcp.Home == "" {
		return fmt.Errorf("%w: %s requires a resolved home", ErrWriteInvalid, r.Kind)
	}
	switch r.Kind {
	case WriteMCPApply:
		if mcp.ConfigPath == "" {
			return fmt.Errorf("%w: %s requires the mcp config path", ErrWriteInvalid, r.Kind)
		}
		if mcp.BackupID != "" {
			return fmt.Errorf("%w: %s does not take a backup identity", ErrWriteInvalid, r.Kind)
		}
		if mcp.PlanID == "" && mcp.Plan == nil {
			return fmt.Errorf("%w: %s requires the plan identity", ErrWriteInvalid, r.Kind)
		}
	case WriteMCPRestore:
		if mcp.PlanID != "" || mcp.Plan != nil {
			return fmt.Errorf("%w: %s does not take a plan identity", ErrWriteInvalid, r.Kind)
		}
		if mcp.BackupID == "" {
			return fmt.Errorf("%w: %s requires the backup identity", ErrWriteInvalid, r.Kind)
		}
	}
	if want := r.mcpCommand(); r.Command != "" && r.Command != want {
		return fmt.Errorf("%w: %s command header must be %q", ErrWriteInvalid, r.Kind, want)
	}
	return nil
}

// writeSeams is the immutable set of engine seams one claimed write captured.
// Passing it by value keeps the command independent of later host changes.
type writeSeams struct {
	applyInstall      InstallApplier
	applyRestore      RestoreApplier
	applyFleet        FleetWriter
	applyFleetRestore FleetWriter
	saveFleetSource   FleetWriter
	applyMCP          MCPWriter
	applyMCPRestore   MCPWriter
	openLog           WriteLogFactory
	now               func() time.Time
}

// fleetWriter returns the seam for one fleet kind, or nil when this build has
// no wiring for it.
func (s writeSeams) fleetWriter(kind WriteKind) FleetWriter {
	switch kind {
	case WriteFleetApply:
		return s.applyFleet
	case WriteFleetRestore:
		return s.applyFleetRestore
	case WriteSourceSave:
		return s.saveFleetSource
	default:
		return nil
	}
}

// mcpWriter returns the seam for one MCP kind, or nil when this build has
// no wiring for it.
func (s writeSeams) mcpWriter(kind WriteKind) MCPWriter {
	switch kind {
	case WriteMCPApply:
		return s.applyMCP
	case WriteMCPRestore:
		return s.applyMCPRestore
	default:
		return nil
	}
}

// run executes one claimed write. It is called from a tea.Cmd goroutine only.
func (h *OpsHost) run(req WriteRequest, id appops.OperationID, home string, seams writeSeams) WriteResultMsg {
	result := WriteResultMsg{ID: id, Kind: req.Kind, Label: req.Label, FinishedAt: seams.now()}

	// A fleet kind without its seam must never look like a write that ran. The
	// refusal happens before the transcript is opened and before any engine
	// call, so no log file and no destination change is produced.
	var fleetWriter FleetWriter
	if req.Kind.isFleet() {
		fleetWriter = seams.fleetWriter(req.Kind)
		if fleetWriter == nil {
			result.Err = fmt.Errorf("%w: %s", ErrFleetUnavailable, req.Kind)
			result.FinishedAt = seams.now()
			return result
		}
	}

	var mcpWriter MCPWriter
	if req.Kind.isMCP() {
		mcpWriter = seams.mcpWriter(req.Kind)
		if mcpWriter == nil {
			result.Err = fmt.Errorf("%w: %s", ErrMCPUnavailable, req.Kind)
			result.FinishedAt = seams.now()
			return result
		}
	}

	log, path, err := seams.openLog(req.Kind, home)
	if err != nil {
		result.Err = fmt.Errorf("open operation log: %w", err)
		result.FinishedAt = seams.now()
		return result
	}
	result.LogPath = path
	command := req.command()
	_, _ = fmt.Fprintf(log, "Started: %s\nCommand: %s\nSource: %s\nHome: %s\n",
		result.FinishedAt.UTC().Format(time.RFC3339Nano), command, req.source(), home)

	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepStarting})
	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepWriting, Detail: command})

	switch req.Kind {
	case WriteInstall:
		result.Result, result.Err = seams.applyInstall(req.Install, req.Token, log)
	case WriteRestore:
		result.Result, result.Err = seams.applyRestore(req.Restore, req.Token, log)
	case WriteFleetApply, WriteFleetRestore, WriteSourceSave:
		result.Result, result.Err = fleetWriter(req.Fleet, req.token(), log)
	case WriteMCPApply, WriteMCPRestore:
		result.Result, result.Err = mcpWriter(req.MCP, req.token(), log)
	}
	result.FinishedAt = seams.now()

	h.emit(WriteProgressMsg{ID: id, Kind: req.Kind, Step: WriteStepFlushing, Detail: path})

	outcome := "success"
	if result.Err != nil {
		outcome = "failure"
		_, _ = fmt.Fprintf(log, "error: %v\n", result.Err)
	}
	_, _ = fmt.Fprintf(log, "Result: %s\nFinished: %s\n", outcome, seams.now().UTC().Format(time.RFC3339Nano))
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
		app.activeRequest = *opts.InitialWrite
		app.hasActiveRequest = true
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
