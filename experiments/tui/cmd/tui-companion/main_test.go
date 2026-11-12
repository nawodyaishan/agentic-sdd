package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"agentic-sdd/internal/buildinfo"
)

// fatalHome fails the test if any query resolves the user's home. The
// informational queries must not inspect the filesystem.
func fatalHome(t *testing.T) func() (string, error) {
	t.Helper()
	return func() (string, error) {
		t.Fatal("resolved the user home during a non-interactive query")
		return "", nil
	}
}

func TestProtocolInfoIsStableAndNonInteractive(t *testing.T) {
	want, err := buildinfo.CurrentProtocolInfo(buildinfo.ProductCompanion).JSON()
	if err != nil {
		t.Fatalf("CurrentProtocolInfo().JSON() error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--protocol-info"}, &stdout, &stderr, environment{
		getenv: func(key string) string {
			if key == "AGENTIC_SDD_FORCE_INTERACTIVE" {
				return "1"
			}
			return ""
		},
		stdinTTY:    false,
		stdoutTTY:   false,
		userHomeDir: fatalHome(t),
	})
	if code != 0 {
		t.Fatalf("--protocol-info exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("--protocol-info stderr = %q, want empty", stderr.String())
	}
	if got := stdout.String(); got != string(want) {
		t.Fatalf("--protocol-info output = %q, want %q", got, want)
	}
	parsed, err := buildinfo.ParseProtocolInfo([]byte(want))
	if err != nil {
		t.Fatalf("ParseProtocolInfo(contract output) error: %v", err)
	}
	if parsed.Product != buildinfo.ProductCompanion {
		t.Fatalf("protocol product = %q, want %q", parsed.Product, buildinfo.ProductCompanion)
	}
	if parsed.ProtocolRevision != buildinfo.ProtocolRevision {
		t.Fatalf("protocol revision = %d, want %d", parsed.ProtocolRevision, buildinfo.ProtocolRevision)
	}
}

func TestProtocolInfoWinsOverOtherFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--protocol-info", "--with-opencode=false", "--opencode-free-first"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 0 {
		t.Fatalf("--protocol-info with other flags exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"product":"agentic-sdd-tui"`) {
		t.Fatalf("--protocol-info output = %q, want the companion product", stdout.String())
	}
}

func TestVersionIsNonInteractive(t *testing.T) {
	identity := buildinfo.CurrentIdentity()
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 0 {
		t.Fatalf("--version exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		buildinfo.ProductCompanion + " " + identity.Revision,
		"commit:        " + identity.Commit,
		"source digest: " + identity.SourceDigest,
		"distribution:  " + string(buildinfo.CurrentDistribution()),
		"protocol:      " + strconv.Itoa(buildinfo.ProtocolRevision),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("--version output %q does not contain %q", out, want)
		}
	}
}

func TestHelpIsPublicAndNonInteractive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 0 {
		t.Fatalf("--help exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("--help stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"Usage: agentic-sdd-tui",
		"--home PATH",
		"--repo PATH",
		"--fleet-config PATH",
		"--with-opencode",
		"--opencode-free-first",
		"--version",
		"--protocol-info",
		"NO_COLOR",
		"AGENTIC_SDD_ASCII",
		"AGENTIC_SDD_REDUCED_MOTION",
		"AGENTIC_SDD_FORCE_INTERACTIVE",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("--help output does not contain %q:\n%s", want, stdout.String())
		}
	}
}

func TestUnknownFlagFailsWithUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--not-a-flag"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("unknown flag exit = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unknown flag stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error:") || !strings.Contains(stderr.String(), "Usage: agentic-sdd-tui") {
		t.Fatalf("unknown flag stderr = %q, want an error and usage", stderr.String())
	}
}

func TestUnexpectedArgumentFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"open"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("unexpected argument exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unexpected argument "open"`) {
		t.Fatalf("unexpected argument stderr = %q", stderr.String())
	}
}

func TestInfoFlagsAreMutuallyExclusive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version", "--protocol-info"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("conflicting info flags exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("conflicting info flags stderr = %q", stderr.String())
	}
}

func TestFreeFirstRequiresOpenCodePack(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--with-opencode=false", "--opencode-free-first"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("free-first without pack exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--opencode-free-first requires --with-opencode") {
		t.Fatalf("free-first without pack stderr = %q", stderr.String())
	}
}

func TestFleetConfigFlagIsAccepted(t *testing.T) {
	// The flag must parse on the public surface; the value is forward-only, so
	// reaching the same terminal gate as a no-flag launch proves acceptance
	// without starting a UI. The test seam does not expose the forwarded value;
	// tuirun owns that assignment.
	var stdout, stderr bytes.Buffer
	code := run([]string{"--fleet-config", "/tmp/example/fleet.json"}, &stdout, &stderr, environment{
		getenv:      func(string) string { return "" },
		stdinTTY:    false,
		stdoutTTY:   true,
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("--fleet-config launch exit = %d, want 2 (stderr: %s)", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("--fleet-config was not accepted: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "stdin is not a terminal") {
		t.Fatalf("--fleet-config stderr = %q, want the stdin terminal gate", stderr.String())
	}
}

func TestNonTTYDoesNotLaunch(t *testing.T) {
	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		want      string
	}{
		{name: "stdin", stdinTTY: false, stdoutTTY: true, want: "stdin is not a terminal"},
		{name: "stdout", stdinTTY: true, stdoutTTY: false, want: "stdout is not a terminal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(nil, &stdout, &stderr, environment{
				getenv:      func(string) string { return "" },
				stdinTTY:    tt.stdinTTY,
				stdoutTTY:   tt.stdoutTTY,
				userHomeDir: fatalHome(t),
			})
			if code != 2 {
				t.Fatalf("non-TTY launch exit = %d, want 2 (stderr: %s)", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("non-TTY launch stderr = %q, want %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestForcedInteractiveDoesNotLaunch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, environment{
		getenv: func(key string) string {
			if key == "AGENTIC_SDD_FORCE_INTERACTIVE" {
				return "1"
			}
			return ""
		},
		stdinTTY:    true,
		stdoutTTY:   true,
		userHomeDir: fatalHome(t),
	})
	if code != 2 {
		t.Fatalf("forced-interactive launch exit = %d, want 2 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "AGENTIC_SDD_FORCE_INTERACTIVE=1") {
		t.Fatalf("forced-interactive stderr = %q", stderr.String())
	}
}
