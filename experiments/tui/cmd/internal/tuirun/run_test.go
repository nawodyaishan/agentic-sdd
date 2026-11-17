// Package tuirun launch-gate tests (Feature 022, B10/T23).
//
// These cases exercise the gates that reject a launch before any terminal, home
// or inventory read happens. They prove the failure paths are instance-local:
// the injected Getenv, Stderr and UserHomeDir seams are the only inputs, and a
// rejected launch never resolves the user's home or opens a program.
package tuirun

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"agentic-sdd/experiments/tui"
)

// gateResult is one rejected launch observation.
type gateResult struct {
	code   int
	stderr string
	// homeResolved records whether the launch resolved the default home, which
	// would mean a rejected run inspected the machine it rejected.
	homeResolved bool
}

// runGate runs one launch with injected seams and no terminal.
func runGate(launch Launch) gateResult {
	var out bytes.Buffer
	result := gateResult{}
	launch.Stderr = &out
	launch.UserHomeDir = func() (string, error) {
		result.homeResolved = true
		return "/must-not-be-used", nil
	}
	result.code = Run(launch)
	result.stderr = out.String()
	return result
}

// TestRunRejectsLegacyOptInBeforeResolvingHome proves the legacy gate rejects a
// launch that did not opt in, before the home is resolved.
func TestRunRejectsLegacyOptInBeforeResolvingHome(t *testing.T) {
	result := runGate(Launch{
		Program:      "agentic-sdd-tui",
		RequireOptIn: true,
		StdinTTY:     true,
		StdoutTTY:    true,
		Getenv:       func(string) string { return "" },
	})
	if result.code != 2 {
		t.Fatalf("exit code = %d, want 2", result.code)
	}
	if result.homeResolved {
		t.Fatal("a rejected launch resolved the default home")
	}
	if !strings.Contains(result.stderr, tui.EnvTUI+"=1 is not set") {
		t.Fatalf("stderr = %q, want the opt-in reason", result.stderr)
	}
}

// TestRunRejectsForcedInteractiveBeforeResolvingHome proves the documented
// automation switch keeps a scripted run on the standard CLI.
func TestRunRejectsForcedInteractiveBeforeResolvingHome(t *testing.T) {
	result := runGate(Launch{
		Program:   "asdx",
		StdinTTY:  true,
		StdoutTTY: true,
		Getenv: func(key string) string {
			if key == tui.EnvForceInteractive {
				return "1"
			}
			return ""
		},
	})
	if result.code != 2 {
		t.Fatalf("exit code = %d, want 2", result.code)
	}
	if result.homeResolved {
		t.Fatal("a rejected launch resolved the default home")
	}
	if !strings.Contains(result.stderr, tui.EnvForceInteractive+"=1") {
		t.Fatalf("stderr = %q, want the forced-interactive reason", result.stderr)
	}
}

// TestRunRejectsNonTTYBeforeResolvingHome proves a piped stdin or stdout never
// enters the alternate screen and never reads the home.
func TestRunRejectsNonTTYBeforeResolvingHome(t *testing.T) {
	cases := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		wantIn    string
	}{
		{name: "stdin", stdinTTY: false, stdoutTTY: true, wantIn: "stdin is not a terminal"},
		{name: "stdout", stdinTTY: true, stdoutTTY: false, wantIn: "stdout is not a terminal"},
		{name: "both", stdinTTY: false, stdoutTTY: false, wantIn: "stdin is not a terminal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runGate(Launch{
				Program:   "asdx",
				StdinTTY:  tc.stdinTTY,
				StdoutTTY: tc.stdoutTTY,
				Getenv:    func(string) string { return "" },
			})
			if result.code != 2 {
				t.Fatalf("exit code = %d, want 2", result.code)
			}
			if result.homeResolved {
				t.Fatal("a rejected launch resolved the default home")
			}
			if !strings.Contains(result.stderr, tc.wantIn) {
				t.Fatalf("stderr = %q, want %q", result.stderr, tc.wantIn)
			}
		})
	}
}

// TestRunRejectsInvalidHoldDuration proves the test-only lifecycle seam fails
// closed with a usage error instead of launching with an unintended write.
func TestRunRejectsInvalidHoldDuration(t *testing.T) {
	result := runGate(Launch{
		Program:   "asdx",
		StdinTTY:  true,
		StdoutTTY: true,
		Getenv: func(key string) string {
			if key == envTestHoldWrite {
				return "not-a-duration"
			}
			return ""
		},
	})
	if result.code != 2 {
		t.Fatalf("exit code = %d, want 2", result.code)
	}
	if !strings.Contains(result.stderr, envTestHoldWrite) {
		t.Fatalf("stderr = %q, want the hold-write variable name", result.stderr)
	}
}

// TestTestHoldWriteParsing covers the seam's own parsing rules.
func TestTestHoldWriteParsing(t *testing.T) {
	cases := []struct {
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{raw: "", want: 0},
		{raw: "1500ms", want: 1500 * time.Millisecond},
		{raw: "2s", want: 2 * time.Second},
		{raw: "0s", want: 0},
		{raw: "-1s", wantErr: true},
		{raw: "soon", wantErr: true},
	}
	for _, tc := range cases {
		got, err := testHoldWrite(func(string) string { return tc.raw })
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q was accepted as %v", tc.raw, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("%q parsed to %v, want %v", tc.raw, got, tc.want)
		}
	}
}
