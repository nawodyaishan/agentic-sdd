package main

// Feature 022 Batch B9 (T21): built canonical/legacy pair contracts (AC-20).
//
// These tests build the real binaries with the release link-time identity and
// run them with separate stdout and stderr pipes. They assert that the legacy
// name preserves canonical stdout bytes, writes only its own deprecation notice
// to stderr, forwards flag terminators and nonzero exits unchanged, and answers
// only the two exact top-level help/version queries locally.

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const b9DeprecationNotice = "agentic-sdd: deprecated compatibility name; running asdx-core\n"

// b9Pair holds the two binaries built from this package.
type b9Pair struct {
	canonical string
	legacy    string
	home      string
}

// b9BuildPair builds the canonical asdx-core binary and the legacy
// agentic-sdd binary from the same source. Binaries live in a temporary
// directory outside the checkout, and the legacy identity is injected exactly
// as the release pipeline does.
func b9BuildPair(t *testing.T) b9Pair {
	t.Helper()
	dir := t.TempDir()
	pair := b9Pair{
		canonical: filepath.Join(dir, "asdx-core"),
		legacy:    filepath.Join(dir, "agentic-sdd"),
		home:      filepath.Join(dir, "home"),
	}
	if err := os.MkdirAll(pair.home, 0o755); err != nil {
		t.Fatalf("create fixture home: %v", err)
	}
	build := func(output string, ldflags string) {
		args := []string{"build", "-o", output}
		if ldflags != "" {
			args = append(args, "-ldflags", ldflags)
		}
		args = append(args, ".")
		cmd := exec.Command("go", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", filepath.Base(output), err, out)
		}
	}
	build(pair.canonical, "")
	build(pair.legacy, "-X main.legacyTarget=asdx-core -X main.legacyName=agentic-sdd")
	return pair
}

// b9Run runs one binary with separate stdout/stderr pipes and the fixture home.
func (p b9Pair) b9Run(t *testing.T, binary string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "HOME="+p.home)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s %v: %v", filepath.Base(binary), args, err)
		}
		code = exitErr.ExitCode()
	}
	return code, out.String(), errOut.String()
}

// b9AssertForwarded pins the wrapper's whole forwarding contract for one argv.
func b9AssertForwarded(t *testing.T, pair b9Pair, wantCode int, args ...string) {
	t.Helper()
	canonicalCode, wantStdout, wantStderr := pair.b9Run(t, pair.canonical, args...)
	if canonicalCode != wantCode {
		t.Fatalf("%v: canonical exit %d, want %d; stderr %q", args, canonicalCode, wantCode, wantStderr)
	}
	code, stdout, stderr := pair.b9Run(t, pair.legacy, args...)
	if code != wantCode {
		t.Fatalf("%v: legacy exit %d, want %d; stderr %q", args, code, wantCode, stderr)
	}
	if stdout != wantStdout {
		t.Fatalf("%v: legacy stdout differs from canonical:\nlegacy %q\ncanonical %q", args, stdout, wantStdout)
	}
	if stderr != b9DeprecationNotice+wantStderr {
		t.Fatalf("%v: legacy stderr is not exactly the notice plus canonical stderr:\nlegacy %q\nwant %q", args, stderr, b9DeprecationNotice+wantStderr)
	}
}

func TestB9LegacyWrapperForwardsInvocations(t *testing.T) {
	pair := b9BuildPair(t)

	cases := []struct {
		name string
		code int
		args []string
	}{
		{name: "json report", code: 0, args: []string{"mcp", "status", "--home", pair.home, "--json"}},
		{name: "restore list json", code: 0, args: []string{"mcp", "restore", "list", "--home", pair.home, "--json"}},
		{name: "unknown subcommand exits two", code: 2, args: []string{"mcp", "bogus"}},
		{name: "help word is forwarded", code: 0, args: []string{"help"}},
		{name: "flag terminator is forwarded", code: 2, args: []string{"preview", "--", "--home"}},
		{name: "tui terminal gate exits two", code: 2, args: []string{"tui"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b9AssertForwarded(t, pair, tc.code, tc.args...)
		})
	}
}

func TestB9LegacyWrapperHelpVersionExceptions(t *testing.T) {
	pair := b9BuildPair(t)

	canonicalCode, canonicalHelp, canonicalErr := pair.b9Run(t, pair.canonical, "help")
	if canonicalCode != 0 || canonicalErr != "" {
		t.Fatalf("canonical help exit %d; stderr %q", canonicalCode, canonicalErr)
	}

	for _, flag := range []string{"--help", "-h"} {
		code, stdout, stderr := pair.b9Run(t, pair.legacy, flag)
		if code != 0 {
			t.Fatalf("%s: exit %d, want 0; stderr %q", flag, code, stderr)
		}
		if stderr != "" {
			t.Fatalf("%s: stderr %q, want empty (no deprecation notice)", flag, stderr)
		}
		wantPrefix := "agentic-sdd is a deprecated compatibility name for asdx-core.\n" +
			"Run asdx-core for the canonical command; the commands and flags are identical.\n\n"
		if !strings.HasPrefix(stdout, wantPrefix) {
			t.Fatalf("%s: stdout does not start with the legacy header:\n%s", flag, stdout)
		}
		if got := strings.TrimPrefix(stdout, wantPrefix); got != canonicalHelp {
			t.Fatalf("%s: legacy help body differs from canonical help:\nlegacy %q\ncanonical %q", flag, got, canonicalHelp)
		}
	}

	canonicalCode, canonicalVersion, canonicalErr := pair.b9Run(t, pair.canonical, "version")
	if canonicalCode != 0 || canonicalErr != "" {
		t.Fatalf("canonical version exit %d; stderr %q", canonicalCode, canonicalErr)
	}
	canonicalLines := strings.SplitN(canonicalVersion, "\n", 2)
	if len(canonicalLines) != 2 {
		t.Fatalf("canonical version = %q", canonicalVersion)
	}
	wantFirst := "agentic-sdd " + strings.TrimPrefix(canonicalLines[0], "asdx-core ") + " (legacy compatibility name for asdx-core)\n"

	code, stdout, stderr := pair.b9Run(t, pair.legacy, "--version")
	if code != 0 {
		t.Fatalf("--version: exit %d, want 0; stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("--version: stderr %q, want empty (no deprecation notice)", stderr)
	}
	lines := strings.SplitN(stdout, "\n", 2)
	if len(lines) != 2 || lines[0]+"\n" != wantFirst {
		t.Fatalf("--version first line = %q, want %q", lines[0], wantFirst)
	}
	if lines[1] != canonicalLines[1] {
		t.Fatalf("--version canonical fields changed:\nlegacy %q\ncanonical %q", lines[1], canonicalLines[1])
	}
}
