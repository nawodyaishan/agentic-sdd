package tui

// Tests for the host-owned OS signal policy. They run in package tui so the
// platform signal set and the narrow registrar seam can be exercised without
// registering real process signals; forwarding still goes through the same
// goroutine the production host uses.

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// fakeRegistrar captures the channel and signals Bubble Tea would have owned,
// and records Stop calls. Sending on channel drives the real forwarding loop.
type fakeRegistrar struct {
	mu         sync.Mutex
	channel    chan<- os.Signal
	registered []os.Signal
	stops      int
}

func (r *fakeRegistrar) Notify(ch chan<- os.Signal, sig ...os.Signal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channel = ch
	r.registered = append([]os.Signal(nil), sig...)
}

func (r *fakeRegistrar) Stop(chan<- os.Signal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops++
}

func (r *fakeRegistrar) send(sig os.Signal) {
	r.mu.Lock()
	ch := r.channel
	r.mu.Unlock()
	ch <- sig
}

func (r *fakeRegistrar) registeredSignals() []os.Signal {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]os.Signal(nil), r.registered...)
}

func (r *fakeRegistrar) stopCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stops
}

func waitShutdown(t *testing.T, received <-chan tea.Msg) ShutdownRequestMsg {
	t.Helper()
	select {
	case raw := <-received:
		msg, ok := raw.(ShutdownRequestMsg)
		if !ok {
			t.Fatalf("signal handler sent %T, want ShutdownRequestMsg", raw)
		}
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("signal handler did not forward the signal")
		return ShutdownRequestMsg{}
	}
}

func TestShutdownSignalsMatchPlatformContract(t *testing.T) {
	names := make([]string, 0, len(shutdownSignals()))
	for _, sig := range shutdownSignals() {
		names = append(names, shutdownSignalName(sig))
	}
	if unixShutdownSignals {
		want := []string{"SIGINT", "SIGTERM", "SIGHUP"}
		if len(names) != len(want) {
			t.Fatalf("Unix signals = %v, want %v", names, want)
		}
		for i := range want {
			if names[i] != want[i] {
				t.Fatalf("Unix signal %d = %q, want %q", i, names[i], want[i])
			}
		}
		return
	}
	// Non-Unix fallback: exactly one portable signal, documented in
	// signals_other.go. The PTY evidence is Unix-only and is not claimed here.
	if len(names) != 1 || names[0] != os.Interrupt.String() {
		t.Fatalf("non-Unix fallback signals = %v, want only %q", names, os.Interrupt.String())
	}
}

func TestSignalHandlerForwardsEverySignalAndRepeat(t *testing.T) {
	registrar := &fakeRegistrar{}
	received := make(chan tea.Msg, 8)
	handler := newSignalHandler(shutdownSignals(), registrar, func(msg tea.Msg) {
		received <- msg
	})
	defer handler.Stop()

	registered := registrar.registeredSignals()
	if len(registered) != len(shutdownSignals()) {
		t.Fatalf("registered %d signals, want %d", len(registered), len(shutdownSignals()))
	}

	signals := shutdownSignals()
	for _, sig := range signals {
		registrar.send(sig)
	}
	for i, sig := range signals {
		msg := waitShutdown(t, received)
		if msg.Reason != ShutdownReasonSignal {
			t.Fatalf("signal %d reason = %v, want signal", i, msg.Reason)
		}
		if msg.Signal != shutdownSignalName(sig) {
			t.Fatalf("signal %d name = %q, want %q", i, msg.Signal, shutdownSignalName(sig))
		}
	}

	// Repeated signals are forwarded, not coalesced or turned into a kill; the
	// root model is responsible for treating them as the same pending exit.
	registrar.send(signals[0])
	repeat := waitShutdown(t, received)
	if repeat.Signal != shutdownSignalName(signals[0]) {
		t.Fatalf("repeat signal = %q, want %q", repeat.Signal, shutdownSignalName(signals[0]))
	}
}

func TestSignalHandlerStopIsIdempotentAndStopsDelivery(t *testing.T) {
	registrar := &fakeRegistrar{}
	received := make(chan tea.Msg, 4)
	handler := newSignalHandler(shutdownSignals(), registrar, func(msg tea.Msg) {
		received <- msg
	})

	handler.Stop()
	handler.Stop()
	if got := registrar.stopCount(); got != 1 {
		t.Fatalf("registrar Stop calls = %d, want 1", got)
	}

	// A signal arriving after Stop must not be forwarded. The fake channel is
	// still open, which mirrors an in-flight delivery the runtime may have
	// queued before Stop returned.
	registrar.send(shutdownSignals()[0])
	select {
	case msg := <-received:
		t.Fatalf("signal forwarded after Stop: %+v", msg)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSignalHandlerWithoutSendDoesNotPanic(t *testing.T) {
	registrar := &fakeRegistrar{}
	handler := newSignalHandler(shutdownSignals(), registrar, nil)
	registrar.send(shutdownSignals()[0])
	handler.Stop()
}

func TestTerminalReleaseIsIdempotentAndDisarmable(t *testing.T) {
	releaseErr := errors.New("restore failed")
	var calls int
	release := NewTerminalRelease(func() error {
		calls++
		return releaseErr
	})
	if err := release.Release(); !errors.Is(err, releaseErr) {
		t.Fatalf("Release error = %v, want %v", err, releaseErr)
	}
	if err := release.Release(); !errors.Is(err, releaseErr) {
		t.Fatalf("repeated Release error = %v, want the cached %v", err, releaseErr)
	}
	if calls != 1 {
		t.Fatalf("release calls = %d, want 1", calls)
	}
	release.Disarm()
	if err := release.Release(); !errors.Is(err, releaseErr) {
		t.Fatalf("Release after Disarm error = %v, want the cached %v", err, releaseErr)
	}
	if calls != 1 {
		t.Fatalf("release calls after Disarm = %d, want 1", calls)
	}

	// A normal Run return disarms the fallback before the deferred call.
	var normalCalls int
	normal := NewTerminalRelease(func() error {
		normalCalls++
		return nil
	})
	normal.Disarm()
	if err := normal.Release(); err != nil {
		t.Fatalf("disarmed Release: %v", err)
	}
	if normalCalls != 0 {
		t.Fatalf("disarmed release ran %d times", normalCalls)
	}

	// Launch failures can pass a nil release; both a nil function and a nil
	// receiver must be safe.
	nilFunc := NewTerminalRelease(nil)
	if err := nilFunc.Release(); err != nil {
		t.Fatalf("nil release error: %v", err)
	}
	var nilReceiver *TerminalRelease
	if err := nilReceiver.Release(); err != nil {
		t.Fatalf("nil receiver error: %v", err)
	}
}

func TestWritePolicyDocumentsResidualRisk(t *testing.T) {
	for _, want := range []string{"q, Esc", "Ctrl+C", "SIGINT", "SIGTERM", "SIGHUP", "log", "SIGKILL"} {
		if !strings.Contains(WritePolicyText, want) {
			t.Fatalf("WritePolicyText does not mention %q:\n%s", want, WritePolicyText)
		}
	}
	if !strings.Contains(WritePolicyText, "cannot be abandoned") {
		t.Fatalf("WritePolicyText does not state that a write cannot be abandoned:\n%s", WritePolicyText)
	}
}
