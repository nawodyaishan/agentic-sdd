package opencodeadapter

// ACP protocol 1 (Agent Client Protocol) over newline-delimited JSON-RPC 2.0.
// This is the narrow owned-process transport: a small client that speaks to one
// private `opencode acp` child over stdin/stdout. It is not an agent loop, a
// broker or a shared-service client, and it never reads a credential store.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const (
	acpProtocolVersion = 1

	// acpMaxMessage bounds a single ACP message in either direction. Larger
	// frames are malformed input, not a reason to retry.
	acpMaxMessage = 256 << 10 // 256 KiB

	// acpMaxObservations bounds the aggregate retained observations across a
	// session (notifications, usage, notes). Exceeding it stops the operation.
	acpMaxObservations = 10 << 20 // 10 MiB

	// acpControlTimeout bounds each handshake/control request (initialize,
	// session/new, set_config_option, cancel).
	acpControlTimeout = 10 * time.Second

	// acpCancelWait is how long a pending-input stop waits for the native
	// cancellation to be confirmed before terminating the owned process group.
	acpCancelWait = 5 * time.Second

	// acpDefaultDeadline bounds a session operation (non-inference preparation).
	acpDefaultDeadline = 15 * time.Minute
)

// acpMessage is one newline-delimited JSON-RPC 2.0 message. A request has both
// ID and Method, a response has ID and no Method, a notification has Method and
// no ID.
type acpMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *acpError       `json:"error,omitempty"`
}

type acpError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (m acpMessage) isResponse() bool     { return m.ID != nil && m.Method == "" }
func (m acpMessage) isNotification() bool { return m.ID == nil && m.Method != "" }
func (m acpMessage) isRequest() bool      { return m.ID != nil && m.Method != "" }

// idAsInt reports whether the message carries an integer JSON-RPC id equal to
// want. OpenCode echoes the client's numeric ids; any other id is unexpected.
func (m acpMessage) idAsInt(want int) bool {
	var n json.Number
	if err := json.Unmarshal(m.ID, &n); err != nil {
		return false
	}
	v, err := n.Int64()
	return err == nil && v == int64(want)
}

// acpStream is a deadline-capable bidirectional byte stream (an owned process's
// stdin/stdout pipes, or an in-memory fake peer).
type acpStream interface {
	io.Reader
	io.Writer
	io.Closer
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// writeACP marshals and writes one bounded single-line ACP message.
func writeACP(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("cannot encode acp message")
	}
	if len(data) > acpMaxMessage {
		return fmt.Errorf("acp message exceeds bound")
	}
	if bytes.IndexByte(data, '\n') >= 0 {
		return fmt.Errorf("acp message must be single-line")
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("acp write failed")
	}
	return nil
}

// readACP reads one bounded ACP message. It rejects frames larger than the
// bound, a stream that ends mid-message, and invalid or wrong-version JSON.
func readACP(r *bufio.Reader) (acpMessage, error) {
	line, err := r.ReadSlice('\n')
	switch {
	case err == bufio.ErrBufferFull:
		return acpMessage{}, fmt.Errorf("acp message exceeds bound")
	case err == io.EOF:
		if len(line) == 0 {
			return acpMessage{}, io.EOF
		}
		return acpMessage{}, fmt.Errorf("acp stream ended mid-message")
	case err != nil:
		return acpMessage{}, fmt.Errorf("acp read failed: %w", err)
	}
	line = bytes.TrimRight(line, "\r\n")
	var m acpMessage
	if err := json.Unmarshal(line, &m); err != nil {
		return acpMessage{}, fmt.Errorf("invalid acp message")
	}
	if m.JSONRPC != "2.0" {
		return acpMessage{}, fmt.Errorf("acp jsonrpc mismatch")
	}
	return m, nil
}

// ---- Concrete ACP protocol 1 message bodies (bounded, allowlisted fields) ----

type acpInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeParams struct {
	ProtocolVersion int `json:"protocolVersion"`
	ClientCapabilities struct {
		Elicitation struct {
			Form bool `json:"form"`
		} `json:"elicitation"`
	} `json:"clientCapabilities"`
	ClientInfo acpInfo `json:"clientInfo"`
}

type initializeResult struct {
	ProtocolVersion   int            `json:"protocolVersion"`
	AgentCapabilities map[string]any `json:"agentCapabilities"`
	AuthMethods       []any          `json:"authMethods"`
}

type sessionNewParams struct {
	Cwd                 string   `json:"cwd"`
	AdditionalDirectories []string `json:"additionalDirectories"`
	MCPServers          []any    `json:"mcpServers"`
}

type acpConfigOption struct {
	ID       string   `json:"id"`
	Category string   `json:"category"`
	Values   []string `json:"values"`
}

type sessionNewResult struct {
	SessionID     string            `json:"sessionId"`
	ConfigOptions []acpConfigOption `json:"configOptions"`
}

type setConfigOptionParams struct {
	SessionID string `json:"sessionId"`
	ConfigID  string `json:"configId"`
	Value     string `json:"value"`
}

type promptBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type promptParams struct {
	SessionID string        `json:"sessionId"`
	Prompt    []promptBlock `json:"prompt"`
}

type acpUsage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	TotalTokens  int64 `json:"totalTokens"`
}

type promptResult struct {
	StopReason string    `json:"stopReason"`
	Usage      *acpUsage `json:"usage,omitempty"`
}

type cancelParams struct {
	SessionID string `json:"sessionId"`
}

// sessionUpdate is a `session/update` notification. Only the fields the narrow
// client inspects are modeled; unknown optional fields are discarded by the
// decoder rather than exported.
type sessionUpdate struct {
	SessionID     string                     `json:"sessionId"`
	SessionUpdate string                     `json:"sessionUpdate"`
	ToolCallID    string                     `json:"toolCallId,omitempty"`
	Meta          map[string]json.RawMessage `json:"_meta,omitempty"`
	Used          int64                      `json:"used,omitempty"`
	Size          int64                      `json:"size,omitempty"`
	Cost          *acpCost                   `json:"cost,omitempty"`
}

type acpCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// elicitationCreate is a server-to-client request asking for a form answer.
// The client keeps it unanswered and cancels the owned prompt instead.
type elicitationCreate struct {
	Mode            string          `json:"mode"`
	SessionID       string          `json:"sessionId"`
	ToolCallID      string          `json:"toolCallId"`
	Message         string          `json:"message"`
	RequestedSchema json.RawMessage `json:"requestedSchema"`
}

// permissionRequest is a server-to-client request asking to grant a tool call.
type permissionRequest struct {
	SessionID string          `json:"sessionId"`
	ToolCall  json.RawMessage `json:"toolCall"`
	Options   json.RawMessage `json:"options"`
}
