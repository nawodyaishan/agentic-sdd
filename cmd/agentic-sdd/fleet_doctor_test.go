package main

// Feature 014 B5 / T09 — CLI seam tests for `fleet doctor`.
// Seam: run(...) exactly like the other command tests. All fixtures use
// t.TempDir(); no real home, no network, no model process beyond local
// shell fixtures. Focused: go test ./cmd/agentic-sdd -run TestFleetDoctor.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fleetDoctorConfigFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fleet.json")
	contents := `{
  "schemaVersion": 1,
  "fleet": "personal",
  "instances": {
    "sdd-implement-1": {"source": "preset:sdd-implement-1", "model": "opencode-go/deepseek-v4.1-flash#high", "enabled": true},
    "sdd-implement-9r-1": {"source": "preset:sdd-implement-1", "model": "9router/ag/gemini-3.8-flash-high", "enabled": true},
    "sdd-openai-architect": {"source": "agent:sdd-openai-architect", "model": "openai/gpt-5", "enabled": true}
  }
}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runFleetCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := runNonInteractive(append([]string{"fleet"}, args...), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestFleetDoctorCLIOfflineDefault(t *testing.T) {
	cfg := fleetDoctorConfigFixture(t)
	code, stdout, stderr := runFleetCLI(t, "doctor", "--config", cfg)
	if code != 0 {
		t.Fatalf("offline doctor exited %d: stdout %q stderr %q", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("offline doctor wrote to stderr: %q", stderr)
	}
	for _, want := range []string{
		"opencode-go/deepseek-v4.1-flash#high",
		"9router/ag/gemini-3.8-flash-high",
		"openai/gpt-5",
		"[not checked (offline)]",
		"no network or model calls",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("report missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "[authenticated]") || strings.Contains(stdout, "[catalog-visible]") {
		t.Fatalf("offline default inferred entitlement: %s", stdout)
	}
	// The personal config's unrelated fields (source, enabled) stay out of
	// the route report (NFR-08: only relevant model IDs).
	if strings.Contains(stdout, "preset:sdd-implement-1") {
		t.Fatalf("report echoed config internals: %s", stdout)
	}
}

func TestFleetDoctorCLIValidationAndUsage(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"doctor", "--nope"}},
		{"extra argument", []string{"doctor", "extra"}},
		{"missing config file", []string{"doctor", "--config", filepath.Join(t.TempDir(), "gone.json")}},
		{"invalid config", []string{"doctor", "--config", func() string {
			p := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}()}},
		{"policy key smuggled in", []string{"doctor", "--config", func() string {
			// Strict parse rejects unknown fields (permissions), so doctor
			// must refuse before labeling anything (exit 2, nothing leaked).
			p := filepath.Join(t.TempDir(), "policy.json")
			body := `{"schemaVersion":1,"instances":{"a":{"source":"agent:x","model":"opencode-go/x1","permissions":{"edit":"allow"}}}}`
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			return p
		}()}},
		{"unknown fleet command", []string{"wat"}},
		{"bare fleet", nil},
	} {
		code, stdout, stderr := runFleetCLI(t, tc.args...)
		if code != 2 {
			t.Errorf("%s: exit %d, want 2 (stdout %q stderr %q)", tc.name, code, stdout, stderr)
		}
		if stdout != "" {
			t.Errorf("%s: validation path printed to stdout: %q", tc.name, stdout)
		}
		if strings.Contains(stderr, "allow") {
			t.Errorf("%s: validation error echoed config content: %q", tc.name, stderr)
		}
	}
}

func TestFleetDoctorCLIUnavailableLabelIsStillSuccess(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-opencode")
	code, stdout, stderr := runFleetCLI(t, "doctor", "--config", fleetDoctorConfigFixture(t), "--opencode-bin", missing)
	if code != 0 {
		t.Fatalf("a produced report is a success even when the tool is missing: %d %q %q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "[unavailable]") {
		t.Fatalf("missing binary must label unavailable:\n%s", stdout)
	}
	if !strings.Contains(stdout, "tools inspected: (none") && !strings.Contains(stdout, "inspection note") {
		t.Fatalf("failure must be reported honestly:\n%s", stdout)
	}
	if strings.Contains(stdout, "[authenticated]") {
		t.Fatalf("a failed inspection cannot authenticate anything:\n%s", stdout)
	}
}

func TestFleetDoctorCLIHelpSurface(t *testing.T) {
	if code, stdout, stderr := runFleetCLI(t, "doctor", "--help"); code != 0 || !strings.Contains(stdout, "Usage: asdx-core fleet doctor") {
		t.Fatalf("doctor --help: code %d stdout %q stderr %q", code, stdout, stderr)
	}
	var out, errOut bytes.Buffer
	if code := runNonInteractive([]string{"help", "fleet"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "doctor") {
		t.Fatalf("help fleet: code %d out %q err %q", code, out.String(), errOut.String())
	}
}
