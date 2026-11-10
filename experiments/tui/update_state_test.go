package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"agentic-sdd/internal/appops"
)

func TestUpdateState_InitialState(t *testing.T) {
	mgr := NewUpdateCheckManager(WithEnvLookup(func(string) string { return "" }))
	if got := mgr.State(); got != UpdateStatusNotStarted {
		t.Fatalf("State() = %v, want %v", got, UpdateStatusNotStarted)
	}
	if mgr.Attempted() {
		t.Fatalf("Attempted() = true, want false")
	}
	if mgr.IsDisabled() {
		t.Fatalf("IsDisabled() = true, want false")
	}
	if got := mgr.Generation(); got != 0 {
		t.Fatalf("Generation() = %d, want 0", got)
	}
	if got := mgr.BadgeText(); got != "not checked" {
		t.Fatalf("BadgeText() = %q, want %q", got, "not checked")
	}
}

func TestUpdateState_AutomaticCheckOnceAcrossRevisits(t *testing.T) {
	var callCount int32
	fakeCheck := func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
		atomic.AddInt32(&callCount, 1)
		return appops.UpdateResult{
			State:   appops.UpdateEqual,
			Current: "v0.1.0",
			Latest:  "v0.1.0",
		}
	}

	mgr := NewUpdateCheckManager(
		WithCheckFunc(fakeCheck),
		WithEnvLookup(func(string) string { return "" }),
	)

	// First entry to Home triggers automatic check
	cmd, started := mgr.AutomaticCheck("v0.1.0")
	if !started || cmd == nil {
		t.Fatalf("first AutomaticCheck started = %v, cmd == nil (%v); want started = true", started, cmd == nil)
	}
	if mgr.State() != UpdateStatusChecking {
		t.Fatalf("state after first check = %v, want %v", mgr.State(), UpdateStatusChecking)
	}
	if !mgr.Attempted() {
		t.Fatalf("Attempted() = false, want true")
	}

	// Execute the command to simulate async message loop
	msg := cmd()
	if !mgr.HandleMsg(msg) {
		t.Fatalf("HandleMsg returned false, want true")
	}
	if mgr.State() != UpdateStatusCurrent {
		t.Fatalf("state after completion = %v, want %v", mgr.State(), UpdateStatusCurrent)
	}

	// Revisit Home: must NOT trigger another check
	cmd2, started2 := mgr.AutomaticCheck("v0.1.0")
	if started2 || cmd2 != nil {
		t.Fatalf("revisit AutomaticCheck started = %v, cmd = %v; want (nil, false)", started2, cmd2)
	}

	// Window resize: must NOT trigger another check
	cmd3, started3 := mgr.AutomaticCheck("v0.1.0")
	if started3 || cmd3 != nil {
		t.Fatalf("resize AutomaticCheck started = %v, cmd = %v; want (nil, false)", started3, cmd3)
	}

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("callCount = %d, want exactly 1", got)
	}
}

func TestUpdateState_DisabledByEnvironment(t *testing.T) {
	var callCount int32
	fakeCheck := func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
		atomic.AddInt32(&callCount, 1)
		return appops.UpdateResult{State: appops.UpdateEqual}
	}

	mgr := NewUpdateCheckManager(
		WithCheckFunc(fakeCheck),
		WithEnvLookup(func(k string) string {
			if k == EnvNoUpdateCheck {
				return "1"
			}
			return ""
		}),
	)

	if !mgr.IsDisabled() {
		t.Fatalf("IsDisabled() = false, want true")
	}
	if mgr.State() != UpdateStatusDisabled {
		t.Fatalf("State() = %v, want %v", mgr.State(), UpdateStatusDisabled)
	}

	cmd, started := mgr.AutomaticCheck("v0.1.0")
	if started || cmd != nil {
		t.Fatalf("AutomaticCheck when disabled returned started = %v, want false", started)
	}

	cmdRetry, startedRetry := mgr.RetryCheck("v0.1.0")
	if startedRetry || cmdRetry != nil {
		t.Fatalf("RetryCheck when disabled returned started = %v, want false", startedRetry)
	}

	if got := atomic.LoadInt32(&callCount); got != 0 {
		t.Fatalf("callCount = %d, want 0", got)
	}
}

func TestUpdateState_DisabledExplicitly(t *testing.T) {
	mgr := NewUpdateCheckManager(WithDisabled(true))
	if !mgr.IsDisabled() {
		t.Fatalf("IsDisabled() = false, want true")
	}
	if mgr.State() != UpdateStatusDisabled {
		t.Fatalf("State() = %v, want %v", mgr.State(), UpdateStatusDisabled)
	}

	cmd, started := mgr.AutomaticCheck("v0.1.0")
	if started || cmd != nil {
		t.Fatalf("AutomaticCheck returned (%v, %v), want (nil, false)", cmd, started)
	}

	// Re-enabling clears disabled state
	mgr.Enable()
	if mgr.IsDisabled() {
		t.Fatalf("IsDisabled() after Enable() = true, want false")
	}
	if mgr.State() != UpdateStatusNotStarted {
		t.Fatalf("State() after Enable() = %v, want %v", mgr.State(), UpdateStatusNotStarted)
	}
}

func TestUpdateState_RetryCheck(t *testing.T) {
	var callCount int32
	fakeCheck := func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
		count := atomic.AddInt32(&callCount, 1)
		if count == 1 {
			return appops.UpdateResult{State: appops.UpdateFailed}
		}
		return appops.UpdateResult{
			State:   appops.UpdateNewer,
			Current: "v0.1.0",
			Latest:  "v0.2.0",
		}
	}

	mgr := NewUpdateCheckManager(
		WithCheckFunc(fakeCheck),
		WithEnvLookup(func(string) string { return "" }),
	)

	// Automatic check fails
	cmd1, _ := mgr.AutomaticCheck("v0.1.0")
	mgr.HandleMsg(cmd1())
	if mgr.State() != UpdateStatusUnavailable {
		t.Fatalf("state after failed check = %v, want %v", mgr.State(), UpdateStatusUnavailable)
	}

	// Automatic check won't run again
	if _, started := mgr.AutomaticCheck("v0.1.0"); started {
		t.Fatalf("second AutomaticCheck should not start")
	}

	// Explicit retry succeeds
	cmd2, started2 := mgr.RetryCheck("v0.1.0")
	if !started2 || cmd2 == nil {
		t.Fatalf("RetryCheck returned (%v, %v), want started = true", cmd2, started2)
	}
	if mgr.State() != UpdateStatusChecking {
		t.Fatalf("state during retry = %v, want %v", mgr.State(), UpdateStatusChecking)
	}
	mgr.HandleMsg(cmd2())
	if mgr.State() != UpdateStatusNewer {
		t.Fatalf("state after retry = %v, want %v", mgr.State(), UpdateStatusNewer)
	}
	if mgr.Latest() != "v0.2.0" {
		t.Fatalf("Latest() = %q, want v0.2.0", mgr.Latest())
	}
	if got := atomic.LoadInt32(&callCount); got != 2 {
		t.Fatalf("callCount = %d, want 2", got)
	}
}

func TestUpdateState_StaleResultSuppression(t *testing.T) {
	mgr := NewUpdateCheckManager(
		WithCheckFunc(func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
			return appops.UpdateResult{State: appops.UpdateEqual, Current: "v0.1.0", Latest: "v0.1.0"}
		}),
		WithEnvLookup(func(string) string { return "" }),
	)

	// Start check 1 (gen 1)
	cmd1, _ := mgr.AutomaticCheck("v0.1.0")
	gen1 := mgr.Generation()

	// Before handling check 1, a retry is initiated (gen 2)
	cmd2, _ := mgr.RetryCheck("v0.1.0")
	gen2 := mgr.Generation()
	if gen2 <= gen1 {
		t.Fatalf("gen2 (%d) should be > gen1 (%d)", gen2, gen1)
	}

	// Produce msg1 from check 1
	msg1 := cmd1()
	handled1 := mgr.HandleMsg(msg1)
	if handled1 {
		t.Fatalf("HandleMsg for stale generation should return false, got true")
	}
	if mgr.State() != UpdateStatusChecking {
		t.Fatalf("State should remain checking, got %v", mgr.State())
	}

	// Produce msg2 from check 2
	msg2 := cmd2()
	handled2 := mgr.HandleMsg(msg2)
	if !handled2 {
		t.Fatalf("HandleMsg for current generation should return true, got false")
	}
	if mgr.State() != UpdateStatusCurrent {
		t.Fatalf("State after gen 2 = %v, want %v", mgr.State(), UpdateStatusCurrent)
	}
}

func TestUpdateState_CancellationOnDisable(t *testing.T) {
	cancelled := make(chan struct{})
	unblockCheck := make(chan struct{})

	fakeCheck := func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
		go func() {
			<-ctx.Done()
			close(cancelled)
		}()
		<-unblockCheck
		return appops.UpdateResult{State: appops.UpdateEqual}
	}

	mgr := NewUpdateCheckManager(
		WithCheckFunc(fakeCheck),
		WithEnvLookup(func(string) string { return "" }),
	)

	cmd, _ := mgr.AutomaticCheck("v0.1.0")

	// Run cmd in a goroutine to simulate background Bubble Tea runner
	msgCh := make(chan UpdateCheckResultMsg)
	go func() {
		msg := cmd()
		msgCh <- msg.(UpdateCheckResultMsg)
	}()

	// Wait briefly then disable checks while in flight
	mgr.Disable()

	// Verify the in-flight context was cancelled
	select {
	case <-cancelled:
		// Context was cancelled successfully
	case <-time.After(2 * time.Second):
		t.Fatalf("in-flight context was not cancelled when Disable() was called")
	}

	close(unblockCheck)
	msg := <-msgCh

	// The message should be rejected by HandleMsg
	if mgr.HandleMsg(msg) {
		t.Fatalf("HandleMsg after Disable() should return false")
	}
	if mgr.State() != UpdateStatusDisabled {
		t.Fatalf("State after Disable() = %v, want %v", mgr.State(), UpdateStatusDisabled)
	}
}

func TestUpdateState_StatusTransitions(t *testing.T) {
	cases := []struct {
		name          string
		result        appops.UpdateResult
		wantState     UpdateCheckState
		wantBadge     string
		wantStatusSub string
	}{
		{
			name: "equal version",
			result: appops.UpdateResult{
				State:   appops.UpdateEqual,
				Current: "v1.0.0",
				Latest:  "v1.0.0",
			},
			wantState:     UpdateStatusCurrent,
			wantBadge:     "up to date",
			wantStatusSub: "up to date (v1.0.0)",
		},
		{
			name: "ahead version",
			result: appops.UpdateResult{
				State:   appops.UpdateAhead,
				Current: "v1.1.0",
				Latest:  "v1.0.0",
			},
			wantState:     UpdateStatusCurrent,
			wantBadge:     "up to date",
			wantStatusSub: "up to date (v1.1.0)",
		},
		{
			name: "newer version",
			result: appops.UpdateResult{
				State:   appops.UpdateNewer,
				Current: "v1.0.0",
				Latest:  "v2.0.0",
			},
			wantState:     UpdateStatusNewer,
			wantBadge:     "update: v2.0.0",
			wantStatusSub: "Update available: v2.0.0 (installed: v1.0.0)",
		},
		{
			name: "dev unparseable installed version",
			result: appops.UpdateResult{
				State:   appops.UpdateUnavailable,
				Current: "",
				Latest:  "v1.0.0",
			},
			wantState:     UpdateStatusUnavailable,
			wantBadge:     "unavailable",
			wantStatusSub: "Development version; latest release is v1.0.0",
		},
		{
			name: "failed check network or timeout",
			result: appops.UpdateResult{
				State: appops.UpdateFailed,
			},
			wantState:     UpdateStatusUnavailable,
			wantBadge:     "unavailable",
			wantStatusSub: "Update check unavailable (offline or rate-limited)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewUpdateCheckManager(
				WithCheckFunc(func(ctx context.Context, client *http.Client, current string) appops.UpdateResult {
					return tc.result
				}),
				WithEnvLookup(func(string) string { return "" }),
			)

			cmd, _ := mgr.AutomaticCheck("v1.0.0")
			mgr.HandleMsg(cmd())

			if mgr.State() != tc.wantState {
				t.Fatalf("State() = %v, want %v", mgr.State(), tc.wantState)
			}
			if mgr.BadgeText() != tc.wantBadge {
				t.Fatalf("BadgeText() = %q, want %q", mgr.BadgeText(), tc.wantBadge)
			}
			if !strings.Contains(mgr.StatusText(), tc.wantStatusSub) {
				t.Fatalf("StatusText() = %q, want substring %q", mgr.StatusText(), tc.wantStatusSub)
			}
		})
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpdateState_AppopsMockTransportIntegration(t *testing.T) {
	t.Run("newer release 200 response", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				// Assert request headers
				if req.Header.Get("User-Agent") != appops.UpdateUserAgent {
					t.Errorf("User-Agent = %q, want %q", req.Header.Get("User-Agent"), appops.UpdateUserAgent)
				}
				if req.Header.Get("Accept") != appops.UpdateRequestAccept {
					t.Errorf("Accept = %q, want %q", req.Header.Get("Accept"), appops.UpdateRequestAccept)
				}

				body := `{"tag_name":"v1.5.0","draft":false,"prerelease":false}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		mgr := NewUpdateCheckManager(
			WithHTTPClient(client),
			WithEnvLookup(func(string) string { return "" }),
		)

		cmd, _ := mgr.AutomaticCheck("v1.0.0")
		mgr.HandleMsg(cmd())

		if mgr.State() != UpdateStatusNewer {
			t.Fatalf("State() = %v, want %v", mgr.State(), UpdateStatusNewer)
		}
		if mgr.Latest() != "1.5.0" {
			t.Fatalf("Latest() = %q, want 1.5.0", mgr.Latest())
		}
	})

	t.Run("rate-limited 403 response", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(bytes.NewBufferString(`{"message":"API rate limit exceeded"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		mgr := NewUpdateCheckManager(
			WithHTTPClient(client),
			WithEnvLookup(func(string) string { return "" }),
		)

		cmd, _ := mgr.AutomaticCheck("v1.0.0")
		mgr.HandleMsg(cmd())

		if mgr.State() != UpdateStatusUnavailable {
			t.Fatalf("State() = %v, want %v", mgr.State(), UpdateStatusUnavailable)
		}
	})

	t.Run("transport network error / offline", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: lookup api.github.com: no such host")
			}),
		}

		mgr := NewUpdateCheckManager(
			WithHTTPClient(client),
			WithEnvLookup(func(string) string { return "" }),
		)

		cmd, _ := mgr.AutomaticCheck("v1.0.0")
		mgr.HandleMsg(cmd())

		if mgr.State() != UpdateStatusUnavailable {
			t.Fatalf("State() = %v, want %v", mgr.State(), UpdateStatusUnavailable)
		}
	})
}

func TestUpdateState_RenderStatusThemes(t *testing.T) {
	mgr := NewUpdateCheckManager(WithEnvLookup(func(string) string { return "" }))

	// In plain mode (Color: false), no styling escape sequences should be present
	plainTheme := NewTheme(false)
	plain := mgr.RenderStatus(plainTheme)
	if plain != "Update check: not started" {
		t.Fatalf("plain RenderStatus = %q, want %q", plain, "Update check: not started")
	}

	// In color mode (Color: true), string should contain status text
	colorTheme := NewTheme(true)
	colored := mgr.RenderStatus(colorTheme)
	if !strings.Contains(colored, "Update check: not started") {
		t.Fatalf("colored RenderStatus does not contain status text: %q", colored)
	}
}
