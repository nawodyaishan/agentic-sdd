package opencodeadapter

// Session orchestration over the owned ACP process. This is the narrow native
// transport plus strict admission and pending-human normalization. Development
// never starts a real model; the driver is exercised only by fake peers.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Usage is the native-reported usage observation. It is recorded as such, not
// as provider-verified billing (contract C6).
type Usage struct {
	InputTokens  int64    `json:"input_tokens,omitempty"`
	OutputTokens int64    `json:"output_tokens,omitempty"`
	TotalTokens  int64    `json:"total_tokens,omitempty"`
	ContextSize  int64    `json:"context_size,omitempty"`
	CostUSD      *float64 `json:"cost_usd,omitempty"`
	Reported     string   `json:"reported,omitempty"`
}

// SessionOutcome is the bounded result of a session operation. Execution is one
// of prepared, blocked, completed, cancelled or incomplete. A non-nil Pending
// means blocked with pending_human, independent of the cancellation reason.
type SessionOutcome struct {
	Execution  string          `json:"execution"`
	SessionID  string          `json:"session_id,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
	Usage      *Usage          `json:"usage,omitempty"`
	Pending    *PendingSummary `json:"pending,omitempty"`
	Blockers   []string        `json:"blockers,omitempty"`
	Notes      []string        `json:"notes,omitempty"`
}

// SessionParams is the explicit, narrow session request.
type SessionParams struct {
	Executable string
	Workspace  string
	Model      string
	Mode       string
	Prompt     string
	Deadline   time.Duration
}

func (p SessionParams) deadline() time.Duration {
	if p.Deadline <= 0 {
		return acpDefaultDeadline
	}
	return p.Deadline
}

func (p SessionParams) validate(run bool) error {
	if p.Executable == "" || strings.ContainsAny(p.Executable, "\x00 \t\r\n") {
		return fmt.Errorf("explicit fixed executable identity required")
	}
	if _, err := projectRoot(p.Workspace); err != nil {
		return fmt.Errorf("explicit absolute project workspace required")
	}
	for _, id := range []string{p.Model, p.Mode} {
		if id != "" && !metadataID.MatchString(id) {
			return fmt.Errorf("invalid native model/mode identifier")
		}
	}
	if p.deadline() <= 0 || p.deadline() > 24*time.Hour {
		return fmt.Errorf("session deadline out of range")
	}
	if run {
		if strings.TrimSpace(p.Prompt) == "" || len(p.Prompt) > 64<<10 {
			return fmt.Errorf("a single bounded prompt is required")
		}
	}
	return nil
}

// Prepare validates an offline session request and returns what would run, with
// the admission controls that block native execution. It starts no process.
func Prepare(workspace, executable, model, mode string) (SessionOutcome, error) {
	p := SessionParams{Workspace: workspace, Executable: executable, Model: model, Mode: mode}
	if err := p.validate(false); err != nil {
		return SessionOutcome{}, err
	}
	return SessionOutcome{
		Execution: "prepared",
		Blockers:  Admit(LaunchDeclaration{Model: model}),
		Notes:     []string{"native-session-not-started"},
	}, nil
}

// Run executes a native session gated by strict admission. Under this feature's
// boundary admission is always denied, so Run returns blocked before starting a
// process; the transport is reachable only once independent controls exist.
func Run(launch LaunchDeclaration, params SessionParams) (SessionOutcome, error) {
	if err := params.validate(true); err != nil {
		return SessionOutcome{}, err
	}
	if unmet := Admit(launch); len(unmet) != 0 {
		return SessionOutcome{Execution: "blocked", Blockers: unmet, Notes: []string{"native-prompt-admission-denied"}}, nil
	}
	// Unreachable while Admit is non-empty: the controls above can only be
	// satisfied by independently reviewed operator provisioning, not by this
	// declaration. Kept here so the transport is the single real execution path.
	return runOwned(params)
}

// runOwned starts the owned process and drives the ACP session to one outcome,
// then terminates and reaps only the owned process group.
func runOwned(params SessionParams) (SessionOutcome, error) {
	proc, err := spawnOwnedProcess(params.Executable, []string{"acp"}, minimalEnv())
	if err != nil {
		return SessionOutcome{}, err
	}
	out := runSession(proc.Stream(), params, time.Now().Add(params.deadline()))
	_ = proc.Terminate()
	_ = proc.Wait()
	return out, nil
}

// minimalEnv is the allowlisted environment for the owned process: no inherited
// HOME/XDG/provider/proxy or credential variables (contract C4). A real run
// would be provisioned by the operator with an explicit dedicated state root;
// development never reaches this path with a real native process.
func minimalEnv() []string {
	return []string{"PATH=" + os.Getenv("PATH")}
}

// rpcRequest is a JSON-RPC 2.0 request sent by the client.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// driver carries the state of one session exchange.
type driver struct {
	stream    acpStream
	r         *bufio.Reader
	deadline  time.Time
	nextID    int
	sessionID string
	workspace string
	out       SessionOutcome
	observed  int64
	done      bool
}

func (d *driver) remaining() time.Duration {
	rem := time.Until(d.deadline)
	if rem < 0 {
		rem = 0
	}
	return rem
}

func (d *driver) setReadDeadline(limit time.Duration) {
	rem := d.remaining()
	if rem < limit {
		limit = rem
	}
	_ = d.stream.SetReadDeadline(time.Now().Add(limit))
}

func (d *driver) account(m acpMessage) {
	d.observed += int64(len(m.Method) + len(m.Params) + len(m.Result) + len(m.ID))
	if d.observed > acpMaxObservations {
		d.fail("observations-exceeded-bound")
	}
}

func (d *driver) note(s string) {
	if len(d.out.Notes) < 32 {
		d.out.Notes = append(d.out.Notes, s)
	}
}

func (d *driver) block(s string) {
	d.out.Execution = "blocked"
	d.out.Blockers = append(d.out.Blockers, s)
	d.done = true
}

func (d *driver) fail(s string) {
	d.out.Execution = "incomplete"
	d.note(s)
	d.done = true
}

func (d *driver) readNext() (acpMessage, bool) {
	if d.done {
		return acpMessage{}, false
	}
	m, err := readACP(d.r)
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			d.fail("missing-terminal-event")
		case errors.Is(err, os.ErrDeadlineExceeded):
			d.fail("deadline-exceeded")
		default:
			d.fail("transport-error")
		}
		return acpMessage{}, false
	}
	d.account(m)
	return m, true
}

// roundTrip sends a control request and waits for its response, routing any
// interleaved notification/request through handle.
func (d *driver) roundTrip(method string, params any, into any) bool {
	id := d.nextID
	d.nextID++
	d.setReadDeadline(acpControlTimeout)
	d.stream.SetWriteDeadline(time.Now().Add(acpControlTimeout))
	if err := writeACP(d.stream, rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}); err != nil {
		d.fail("control-write-failed")
		return false
	}
	return d.awaitResponse(id, into)
}

func (d *driver) awaitResponse(id int, into any) bool {
	for !d.done {
		m, ok := d.readNext()
		if !ok {
			return false
		}
		if m.isResponse() {
			if !m.idAsInt(id) {
				d.fail("unexpected-response-id")
				return false
			}
			if m.Error != nil {
				d.fail("native-error")
				return false
			}
			if into != nil {
				if err := json.Unmarshal(m.Result, into); err != nil {
					d.fail("malformed-response")
					return false
				}
			}
			return true
		}
		if !d.handle(m) {
			return false
		}
	}
	return false
}

// handle routes a non-response message. It returns false when the session must
// stop (terminal condition), true to keep reading.
func (d *driver) handle(m acpMessage) bool {
	switch {
	case m.isRequest():
		switch m.Method {
		case "elicitation/create":
			return d.onElicitation(m)
		case "session/request_permission":
			return d.onPermission(m)
		default:
			d.fail("unknown-native-request")
			return false
		}
	case m.isNotification() && m.Method == "session/update":
		var u sessionUpdate
		if err := json.Unmarshal(m.Params, &u); err != nil {
			d.fail("malformed-session-update")
			return false
		}
		return d.onSessionUpdate(u)
	default:
		d.note("discarded-notification:" + m.Method)
		return true
	}
}

func (d *driver) onSessionUpdate(u sessionUpdate) bool {
	if u.SessionID != "" && u.SessionID != d.sessionID {
		d.block("foreign-session")
		return false
	}
	if raw, ok := u.Meta["opencode/child-session"]; ok && string(raw) != "null" {
		d.out.Usage = nil
		d.block("child-session-metadata")
		return false
	}
	switch u.SessionUpdate {
	case "usage_update":
		if d.out.Usage == nil {
			d.out.Usage = &Usage{}
		}
		d.out.Usage.TotalTokens = u.Used
		d.out.Usage.ContextSize = u.Size
		if u.Cost != nil && u.Cost.Currency == "USD" {
			amt := u.Cost.Amount
			d.out.Usage.CostUSD = &amt
		}
		d.out.Usage.Reported = "native-reported-not-verified"
	case "session_info_update":
		if raw, ok := u.Meta["opencode/retry"]; ok && string(raw) != "null" {
			d.note("retry-observed")
		}
		if _, ok := u.Meta["opencode/compaction"]; ok {
			d.note("compaction-marker-observed")
		}
	case "tool_call", "available_commands_update", "config_option_update", "compaction_update", "compaction_summary_chunk", "plan", "plan_chunk":
		// bounded observation; no effect on outcome
	default:
		d.note("discarded-update:" + u.SessionUpdate)
	}
	return true
}

func (d *driver) onElicitation(m acpMessage) bool {
	var e elicitationCreate
	if err := json.Unmarshal(m.Params, &e); err != nil || e.Mode != "form" {
		d.fail("malformed-elicitation")
		return false
	}
	if e.SessionID != "" && e.SessionID != d.sessionID {
		d.block("foreign-session")
		return false
	}
	summary, err := newPendingSummary(d.sessionID, d.workspace, e.Message, parseOptions(e.RequestedSchema), "unknown")
	if err != nil {
		d.fail("unbounded-elicitation")
		return false
	}
	d.out.Pending = &summary
	d.out.StopReason = "pending_human"
	d.block("pending-human-input")
	d.sendCancel()
	return false
}

func (d *driver) onPermission(m acpMessage) bool {
	var p permissionRequest
	if err := json.Unmarshal(m.Params, &p); err != nil || p.SessionID == "" {
		d.fail("malformed-permission-request")
		return false
	}
	if p.SessionID != d.sessionID {
		d.block("foreign-session")
		return false
	}
	// Consequential permission: keep unanswered, cancel, block; never approve,
	// reject-with-guess or fabricate a grant.
	summary, err := newPendingSummary(d.sessionID, d.workspace, "permission request requires human decision", nil, "unknown")
	if err != nil {
		d.fail("unbounded-permission-request")
		return false
	}
	d.out.Pending = &summary
	d.out.StopReason = "pending_human"
	d.block("pending-permission")
	d.sendCancel()
	return false
}

// sendCancel requests native cancellation of the owned prompt. It is
// fire-and-forget here; confirmation is awaited separately, bounded to
// acpCancelWait, before the process group is terminated.
func (d *driver) sendCancel() {
	id := d.nextID
	d.nextID++
	d.setReadDeadline(acpControlTimeout)
	d.stream.SetWriteDeadline(time.Now().Add(acpControlTimeout))
	if err := writeACP(d.stream, rpcRequest{JSONRPC: "2.0", ID: id, Method: "session/cancel", Params: cancelParams{SessionID: d.sessionID}}); err != nil {
		d.note("cancel-dispatch-uncertain")
	}
}

func offers(opts []acpConfigOption, id, value string) bool {
	for _, o := range opts {
		if o.ID != id {
			continue
		}
		for _, v := range o.Values {
			if v == value {
				return true
			}
		}
	}
	return false
}

// runSession drives the ACP handshake, session creation, optional model/mode
// selection and one prompt, returning a bounded outcome. The caller closes the
// stream and terminates only the owned process group after this returns.
func runSession(stream acpStream, params SessionParams, deadline time.Time) SessionOutcome {
	d := &driver{
		stream:   stream,
		r:        bufio.NewReaderSize(stream, acpMaxMessage+2),
		deadline: deadline,
		nextID:   1,
	}
	d.out.Execution = "incomplete"

	init := initializeParams{ProtocolVersion: acpProtocolVersion, ClientInfo: acpInfo{Name: "agentic-sdd", Version: "dev"}}
	init.ClientCapabilities.Elicitation.Form = true
	var initRes initializeResult
	if !d.roundTrip("initialize", init, &initRes) {
		return d.finish()
	}
	if initRes.ProtocolVersion != acpProtocolVersion {
		d.block("unsupported-acp-protocol-version")
		return d.finish()
	}
	if initRes.AgentCapabilities == nil {
		d.block("missing-agent-capabilities")
		return d.finish()
	}

	var newRes sessionNewResult
	if !d.roundTrip("session/new", sessionNewParams{Cwd: params.Workspace, AdditionalDirectories: []string{}, MCPServers: []any{}}, &newRes) {
		return d.finish()
	}
	if newRes.SessionID == "" {
		d.block("missing-session-id")
		return d.finish()
	}
	d.sessionID = newRes.SessionID
	d.out.SessionID = newRes.SessionID
	d.workspace = params.Workspace

	for _, sel := range [][2]string{{"model", params.Model}, {"mode", params.Mode}} {
		if sel[1] == "" {
			continue
		}
		if !offers(newRes.ConfigOptions, sel[0], sel[1]) {
			d.block(sel[0] + "-not-offered")
			return d.finish()
		}
		var ignored map[string]any
		if !d.roundTrip("session/set_config_option", setConfigOptionParams{SessionID: d.sessionID, ConfigID: sel[0], Value: sel[1]}, &ignored) {
			return d.finish()
		}
	}

	promptID := d.nextID
	d.nextID++
	d.stream.SetWriteDeadline(time.Now().Add(acpControlTimeout))
	promptReq := promptParams{SessionID: d.sessionID, Prompt: []promptBlock{{Type: "text", Text: params.Prompt}}}
	if err := writeACP(d.stream, rpcRequest{JSONRPC: "2.0", ID: promptID, Method: "session/prompt", Params: promptReq}); err != nil {
		d.fail("prompt-dispatch-failed")
		return d.finish()
	}

	d.setReadDeadline(d.remaining())
	var pr promptResult
	if !d.awaitResponse(promptID, &pr) {
		if d.out.Pending != nil {
			// Bounded cancellation confirmation; uncertainty is recorded, never
			// auto-resumed. The caller still terminates the process group.
			if !d.confirmCancellation(promptID) {
				d.note("cancellation-unconfirmed")
			}
		}
		return d.finish()
	}
	switch pr.StopReason {
	case "end_turn":
		d.out.Execution = "completed"
		d.out.StopReason = "end_turn"
		if pr.Usage != nil {
			d.out.Usage = &Usage{InputTokens: pr.Usage.InputTokens, OutputTokens: pr.Usage.OutputTokens, TotalTokens: pr.Usage.TotalTokens, Reported: "native-reported-not-verified"}
		}
	case "cancelled":
		d.out.Execution = "cancelled"
		d.out.StopReason = "cancelled"
	default:
		// end_turn establishes only that the native turn ended; any other
		// stop reason (provider failure, error) is not task success.
		d.out.Execution = "incomplete"
		d.out.StopReason = pr.StopReason
		d.note("unrecognized-stop-reason")
	}
	return d.finish()
}

// finish records the outcome and ensures the owned transport is closed. The
// caller terminates the process group after this returns.
func (d *driver) finish() SessionOutcome {
	if d.out.Execution == "" {
		d.out.Execution = "incomplete"
	}
	if d.out.Pending != nil && d.out.Execution != "blocked" {
		d.out.Execution = "blocked"
	}
	_ = d.stream.Close()
	return d.out
}

// confirmCancellation reads briefly (bounded to acpCancelWait) for the prompt
// response confirming native cancellation. It skips the cancel acknowledgement
// and any interleaved notifications. A false result means the stop was not
// confirmed, so the caller records uncertain effects and never resumes.
func (d *driver) confirmCancellation(promptID int) bool {
	d.stream.SetReadDeadline(time.Now().Add(acpCancelWait))
	for {
		m, err := readACP(d.r)
		if err != nil {
			return false
		}
		if !m.isResponse() {
			continue
		}
		if !m.idAsInt(promptID) {
			continue // cancel ack or another control response
		}
		var pr promptResult
		if err := json.Unmarshal(m.Result, &pr); err != nil {
			return false
		}
		return pr.StopReason == "cancelled"
	}
}
