package opencodeadapter

// Fake-peer ACP tests. The fake peer is this test binary re-executed as a
// subprocess; it speaks a scripted ACP conversation on fd 3 (ExtraFiles), so its
// fd 1 (owned by `go test`) never pollutes the protocol. No real native process
// or model is ever started.

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	fakeModel   = "openrouter/nvidia/nemotron-3-ultra-550b-a55b:free"
	fakeSession = "ses_test"
)

// TestACPFakePeerProcess is the helper-process entry point.
func TestACPFakePeerProcess(t *testing.T) {
	if os.Getenv("SDD_FAKE_PEER") != "1" {
		return
	}
	os.Exit(runFakePeer(os.Stdin, os.NewFile(3, "acp"), os.Getenv("SDD_SCENARIO")))
}

func runFakePeer(stdin io.Reader, stdout io.Writer, scenario string) int {
	r := bufio.NewReader(stdin)
	w := bufio.NewWriter(stdout)
	write := func(v any) error {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			return err
		}
		return w.Flush()
	}
	read := func() (map[string]any, bool) {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, false
		}
		var m map[string]any
		if json.Unmarshal(line, &m) != nil {
			return nil, false
		}
		return m, true
	}
	for {
		m, ok := read()
		if !ok {
			return 0
		}
		method, _ := m["method"].(string)
		id, _ := m["id"].(float64)
		switch method {
		case "initialize":
			ver := 1
			if scenario == "wrong-version" {
				ver = 2
			}
			if write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"protocolVersion": ver,
				"agentCapabilities": map[string]any{
					"loadSession":       true,
					"promptCapabilities": map[string]any{"image": true},
				},
				"authMethods": []any{},
				"agentInfo":   map[string]any{"name": "opencode", "version": Runtime},
			}}) != nil {
				return 1
			}
			if scenario == "duplicate-id" {
				if write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
					"protocolVersion": ver,
					"agentCapabilities": map[string]any{"loadSession": true},
					"authMethods":      []any{},
				}}) != nil {
					return 1
				}
			}
		case "session/new":
			if write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"sessionId": fakeSession,
				"configOptions": []any{
					map[string]any{"id": "model", "category": "model", "values": []any{fakeModel}},
					map[string]any{"id": "mode", "category": "mode", "values": []any{"primary"}},
				},
			}}) != nil {
				return 1
			}
		case "session/set_config_option":
			if write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}}) != nil {
				return 1
			}
		case "session/cancel":
			if write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}}) != nil {
				return 1
			}
		case "session/prompt":
			if !runPromptScenario(write, read, scenario, id) {
				return 0
			}
		default:
			return 0
		}
	}
}

func runPromptScenario(write func(any) error, read func() (map[string]any, bool), scenario string, id float64) bool {
	switch scenario {
	case "happy":
		return write(promptResponse(id, "end_turn", 1, 2, 3)) == nil
	case "retry":
		_ = write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
			"sessionId": fakeSession, "sessionUpdate": "session_info_update",
			"_meta": map[string]any{"opencode/retry": map[string]any{"attempt": 1, "nextRetryAt": "2026-10-02T12:00:05.000Z"}},
		}})
		return write(promptResponse(id, "end_turn", 1, 2, 3)) == nil
	case "provider-error":
		return write(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32000, "message": "provider failed"}}) == nil
	case "missing-terminal-event":
		_ = write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
			"sessionId": fakeSession, "sessionUpdate": "tool_call", "toolCallId": "call_1",
		}})
		return false
	case "elicitation":
		_ = write(map[string]any{"jsonrpc": "2.0", "id": 100, "method": "elicitation/create", "params": map[string]any{
			"mode": "form", "sessionId": fakeSession, "toolCallId": "call_1",
			"message": "Which package manager?",
			"requestedSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pm": map[string]any{"type": "string", "oneOf": []any{
						map[string]any{"const": "bun", "title": "Bun"},
						map[string]any{"const": "npm", "title": "npm"},
					}},
				},
				"required": []any{"pm"},
			},
		}})
		return cancelAndFinish(write, read, id)
	case "permission":
		_ = write(map[string]any{"jsonrpc": "2.0", "id": 100, "method": "session/request_permission", "params": map[string]any{
			"sessionId": fakeSession,
			"toolCall":  map[string]any{"title": "Edit spec.md"},
			"options":   []any{map[string]any{"optionId": "once", "kind": "allow_once", "name": "Allow once"}},
		}})
		return cancelAndFinish(write, read, id)
	case "child-session":
		_ = write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
			"sessionId": fakeSession, "sessionUpdate": "tool_call", "toolCallId": "ses_child:call_1",
			"_meta": map[string]any{"opencode/child-session": map[string]any{"id": "ses_child", "parentID": fakeSession, "depth": 1}},
		}})
		return false
	case "foreign-session":
		_ = write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
			"sessionId": "ses_other", "sessionUpdate": "tool_call", "toolCallId": "call_1",
		}})
		return false
	case "unknown-request":
		_ = write(map[string]any{"jsonrpc": "2.0", "id": 100, "method": "session/weird", "params": map[string]any{}})
		return false
	case "oversized":
		_ = write(map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
			"sessionId": fakeSession, "sessionUpdate": "tool_call", "toolCallId": strings.Repeat("x", acpMaxMessage+1),
		}})
		return false
	case "deadline":
		time.Sleep(30 * time.Second)
		return false
	}
	return false
}

// cancelAndFinish reads the cancel request, acknowledges it, then emits the
// prompt response with stopReason cancelled.
func cancelAndFinish(write func(any) error, read func() (map[string]any, bool), id float64) bool {
	m, ok := read()
	if !ok {
		return false
	}
	cid, _ := m["id"].(float64)
	_ = write(map[string]any{"jsonrpc": "2.0", "id": cid, "result": map[string]any{}})
	_ = write(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"stopReason": "cancelled"}})
	return false
}

func promptResponse(id float64, stop string, in, out, total int) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
		"stopReason": stop,
		"usage":      map[string]any{"inputTokens": in, "outputTokens": out, "totalTokens": total},
	}}
}

func spawnFakePeer(t *testing.T, scenario string) *process {
	t.Helper()
	proc, err := spawnProcess(os.Args[0], []string{"-test.run=TestACPFakePeerProcess", "--"},
		[]string{"SDD_FAKE_PEER=1", "SDD_SCENARIO=" + scenario}, 3)
	if err != nil {
		t.Fatalf("spawn fake peer: %v", err)
	}
	return proc
}

func runScenario(t *testing.T, scenario string, params SessionParams, deadline time.Duration) SessionOutcome {
	t.Helper()
	proc := spawnFakePeer(t, scenario)
	out := runSession(proc.Stream(), params, time.Now().Add(deadline))
	_ = proc.Terminate()
	_ = proc.Wait()
	return out
}

func baseParams() SessionParams {
	return SessionParams{
		Executable: os.Args[0],
		Workspace:  "/tmp/fake-workspace",
		Model:      fakeModel,
		Mode:       "primary",
		Prompt:     "Reply OK",
		Deadline:   acpDefaultDeadline,
	}
}

func TestSessionHappyPathCompletesWithUsage(t *testing.T) {
	out := runScenario(t, "happy", baseParams(), acpDefaultDeadline)
	if out.Execution != "completed" || out.StopReason != "end_turn" || out.SessionID != fakeSession {
		t.Fatalf("unexpected outcome: %+v", out)
	}
	if out.Usage == nil || out.Usage.TotalTokens != 3 || out.Usage.Reported != "native-reported-not-verified" {
		t.Fatalf("missing/unverified usage: %+v", out.Usage)
	}
}

func TestSessionProviderErrorExitZeroIsNotSuccess(t *testing.T) {
	out := runScenario(t, "provider-error", baseParams(), acpDefaultDeadline)
	if out.Execution == "completed" || out.Execution == "cancelled" {
		t.Fatalf("provider error became success: %+v", out)
	}
	if out.Execution != "incomplete" {
		t.Fatalf("expected incomplete, got %+v", out)
	}
}

func TestSessionElicitationBlocksWithPendingHuman(t *testing.T) {
	out := runScenario(t, "elicitation", baseParams(), acpDefaultDeadline)
	if out.Execution != "blocked" || out.StopReason != "pending_human" || out.Pending == nil {
		t.Fatalf("elicitation not blocked pending_human: %+v", out)
	}
	if out.Pending.Question != "Which package manager?" || len(out.Pending.Options) != 2 ||
		out.Pending.Options[0].ID != "bun" || out.Pending.Options[1].ID != "npm" {
		t.Fatalf("question/options not captured: %+v", out.Pending)
	}
	for _, n := range out.Notes {
		if n == "cancellation-unconfirmed" {
			t.Fatalf("cancellation was unconfirmed when it should have been confirmed: %+v", out.Notes)
		}
	}
}

func TestSessionPermissionBlocksWithoutApproval(t *testing.T) {
	out := runScenario(t, "permission", baseParams(), acpDefaultDeadline)
	if out.Execution != "blocked" || out.StopReason != "pending_human" || out.Pending == nil {
		t.Fatalf("permission not blocked pending_human: %+v", out)
	}
}

func TestSessionChildMetadataBlocksAndInvalidatesUsage(t *testing.T) {
	out := runScenario(t, "child-session", baseParams(), acpDefaultDeadline)
	if out.Execution != "blocked" || !contains(out.Blockers, "child-session-metadata") {
		t.Fatalf("child metadata not refused: %+v", out)
	}
	if out.Usage != nil {
		t.Fatalf("child metadata left aggregate usage intact: %+v", out.Usage)
	}
}

func TestSessionForeignSessionRefused(t *testing.T) {
	out := runScenario(t, "foreign-session", baseParams(), acpDefaultDeadline)
	if out.Execution != "blocked" || !contains(out.Blockers, "foreign-session") {
		t.Fatalf("foreign session not refused: %+v", out)
	}
}

func TestSessionProtocolViolations(t *testing.T) {
	for _, tc := range []struct {
		scenario string
		want     string
	}{
		{"duplicate-id", "incomplete"},
		{"oversized", "incomplete"},
		{"missing-terminal-event", "incomplete"},
		{"unknown-request", "incomplete"},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			out := runScenario(t, tc.scenario, baseParams(), acpDefaultDeadline)
			if out.Execution != tc.want {
				t.Fatalf("got %q want %q: %+v", out.Execution, tc.want, out)
			}
		})
	}
}

func TestSessionDeadlineExceededStopsTheSession(t *testing.T) {
	out := runScenario(t, "deadline", baseParams(), 200*time.Millisecond)
	if out.Execution != "incomplete" {
		t.Fatalf("deadline did not stop the session: %+v", out)
	}
}

func TestSessionWrongVersionRefused(t *testing.T) {
	out := runScenario(t, "wrong-version", baseParams(), acpDefaultDeadline)
	if out.Execution != "blocked" || !contains(out.Blockers, "unsupported-acp-protocol-version") {
		t.Fatalf("wrong version not refused: %+v", out)
	}
}

func TestSessionModelNotOfferedRefused(t *testing.T) {
	p := baseParams()
	p.Model = "anthropic/claude-not-offered"
	out := runScenario(t, "happy", p, acpDefaultDeadline)
	if out.Execution != "blocked" || !contains(out.Blockers, "model-not-offered") {
		t.Fatalf("unoffered model not refused: %+v", out)
	}
}

func TestSessionRetryIsABoundedObservation(t *testing.T) {
	out := runScenario(t, "retry", baseParams(), acpDefaultDeadline)
	if out.Execution != "completed" || !contains(out.Notes, "retry-observed") {
		t.Fatalf("retry not observed as bounded: %+v", out)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
