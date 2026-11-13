package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	if !strings.Contains(stdout.String(), `"product":"asdx"`) {
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
		"Usage: asdx",
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
	if !strings.Contains(stderr.String(), "error:") || !strings.Contains(stderr.String(), "Usage: asdx") {
		t.Fatalf("unknown flag stderr = %q, want an error and usage", stderr.String())
	}
	if !strings.Contains(stderr.String(), "asdx-core") {
		t.Fatalf("unknown flag stderr = %q, want the asdx-core hint", stderr.String())
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
	if !strings.Contains(stderr.String(), "asdx-core") {
		t.Fatalf("unexpected argument stderr = %q, want the asdx-core hint", stderr.String())
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
			if !strings.Contains(stderr.String(), "asdx-core preview") {
				t.Fatalf("non-TTY launch stderr = %q, want the asdx-core preview hint", stderr.String())
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

// --- Legacy compatibility wrapper (dual-build mode) ----------------------

// withLegacyBuild injects the legacy build values for one test.
func withLegacyBuild(t *testing.T, target, name string) {
	t.Helper()
	originalTarget, originalName := legacyTarget, legacyName
	legacyTarget, legacyName = target, name
	t.Cleanup(func() { legacyTarget, legacyName = originalTarget, originalName })
}

// writeLegacyRegular creates a regular file with the given mode.
func writeLegacyRegular(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
}

// fakeLegacyDeps records the executable lookup and the replacement call
// without ever starting or replacing a process.
type fakeLegacyDeps struct {
	executable string
	execErr    error

	execPath string
	execArgv []string
	execCall int
}

func (f *fakeLegacyDeps) deps() legacyDeps {
	return legacyDeps{
		executable: func() (string, error) { return f.executable, nil },
		exec: func(path string, argv []string) error {
			f.execCall++
			f.execPath, f.execArgv = path, argv
			return f.execErr
		},
	}
}

// legacyFixture creates the running compatibility name beside its canonical
// target and returns the running path.
func legacyFixture(t *testing.T, target string) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "agentic-sdd-tui")
	writeLegacyRegular(t, exe, 0755)
	writeLegacyRegular(t, filepath.Join(dir, target), 0755)
	return exe
}

func TestCanonicalBuildHasNoLegacyIdentity(t *testing.T) {
	if legacyTarget != "" || legacyName != "" {
		t.Fatalf("canonical build has legacy identity %q/%q", legacyTarget, legacyName)
	}
	fake := &fakeLegacyDeps{executable: "/does/not/matter"}
	var stdout, stderr bytes.Buffer
	code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr)
	if handled || code != 0 {
		t.Fatalf("canonical build reported handled=%v code=%d", handled, code)
	}
	if fake.execCall != 0 {
		t.Fatal("canonical build performed a legacy replacement")
	}
}

func TestLegacyExactTopLevelQueriesIdentifyBothNames(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")
	fake := &fakeLegacyDeps{executable: "/does/not/matter"}

	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"long help", []string{"--help"}, []string{"agentic-sdd-tui", "asdx", "deprecated", "Usage: asdx"}},
		{"short help", []string{"-h"}, []string{"agentic-sdd-tui", "asdx", "deprecated", "Usage: asdx"}},
		{"version", []string{"--version"}, []string{"agentic-sdd-tui", "asdx", "legacy compatibility name", "protocol:"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, handled := runLegacyCompatibility(fake.deps(), tc.args, &stdout, &stderr)
			if !handled || code != 0 {
				t.Fatalf("handled=%v code=%d, want true/0", handled, code)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout %q does not contain %q", stdout.String(), want)
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if fake.execCall != 0 {
				t.Fatal("an exact top-level query replaced the process")
			}
		})
	}
}

func TestLegacyProtocolInfoIsForwardedWithoutNotice(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")
	exe := legacyFixture(t, "asdx")
	fake := &fakeLegacyDeps{executable: exe}
	var stdout, stderr bytes.Buffer
	code, handled := runLegacyCompatibility(fake.deps(), []string{"--protocol-info"}, &stdout, &stderr)
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d, want true/0", handled, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("--protocol-info wrote to stderr: %q", stderr.String())
	}
	if fake.execCall != 1 {
		t.Fatalf("exec calls = %d, want 1", fake.execCall)
	}
	wantPath := filepath.Join(filepath.Dir(exe), "asdx")
	if fake.execPath != wantPath {
		t.Fatalf("exec path = %q, want %q", fake.execPath, wantPath)
	}
	if want := []string{wantPath, "--protocol-info"}; !reflect.DeepEqual(fake.execArgv, want) {
		t.Fatalf("exec argv = %v, want %v", fake.execArgv, want)
	}
}

func TestLegacyForwardsEveryOtherInvocation(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")
	cases := [][]string{
		nil,
		{"--home", "/tmp/example-home"},
		{"--version", "--protocol-info"},
		{"--", "--version"},
		{"wizard"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			exe := legacyFixture(t, "asdx")
			fake := &fakeLegacyDeps{executable: exe}
			var stdout, stderr bytes.Buffer
			code, handled := runLegacyCompatibility(fake.deps(), args, &stdout, &stderr)
			if !handled || code != 0 {
				t.Fatalf("handled=%v code=%d, want true/0; stderr %q", handled, code, stderr.String())
			}
			if fake.execCall != 1 {
				t.Fatalf("exec calls = %d, want 1", fake.execCall)
			}
			wantPath := filepath.Join(filepath.Dir(exe), "asdx")
			if fake.execPath != wantPath {
				t.Fatalf("exec path = %q, want %q", fake.execPath, wantPath)
			}
			wantArgv := append([]string{wantPath}, args...)
			if !reflect.DeepEqual(fake.execArgv, wantArgv) {
				t.Fatalf("exec argv = %v, want %v", fake.execArgv, wantArgv)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			want := "agentic-sdd-tui: deprecated compatibility name; running asdx\n"
			if stderr.String() != want {
				t.Fatalf("stderr = %q, want %q", stderr.String(), want)
			}
		})
	}
}

func TestLegacyResolvesSymlinkedInstallation(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")
	root := t.TempDir()
	cellar := filepath.Join(root, "Cellar", "agentic-sdd", "0.6.0", "bin")
	exe := filepath.Join(cellar, "agentic-sdd-tui")
	writeLegacyRegular(t, exe, 0755)
	writeLegacyRegular(t, filepath.Join(cellar, "asdx"), 0755)
	link := filepath.Join(root, "bin", "agentic-sdd-tui")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	fake := &fakeLegacyDeps{executable: link}
	var stdout, stderr bytes.Buffer
	if code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr); !handled || code != 0 {
		t.Fatalf("handled=%v code=%d; stderr %q", handled, code, stderr.String())
	}
	if want := filepath.Join(cellar, "asdx"); fake.execPath != want {
		t.Fatalf("exec path = %q, want the resolved cellar sibling %q", fake.execPath, want)
	}
}

func TestLegacyMissingOrInvalidTargetFailsClosed(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")

	t.Run("absent", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd-tui")
		writeLegacyRegular(t, exe, 0755)
		fake := &fakeLegacyDeps{executable: exe}
		var stdout, stderr bytes.Buffer
		code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr)
		if !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if fake.execCall != 0 {
			t.Fatal("a missing target was still executed")
		}
		if !strings.Contains(stderr.String(), "is not installed beside") {
			t.Fatalf("stderr %q does not name the missing target", stderr.String())
		}
		if !strings.Contains(stderr.String(), buildinfo.RebuildPairHint) {
			t.Fatalf("stderr %q is missing rebuild guidance", stderr.String())
		}
	})

	t.Run("protocol info absent target is not JSON", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd-tui")
		writeLegacyRegular(t, exe, 0755)
		fake := &fakeLegacyDeps{executable: exe}
		var stdout, stderr bytes.Buffer
		if code, handled := runLegacyCompatibility(fake.deps(), []string{"--protocol-info"}, &stdout, &stderr); !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want no partial protocol JSON", stdout.String())
		}
		if !strings.Contains(stderr.String(), "asdx") {
			t.Fatalf("stderr %q does not name the missing target", stderr.String())
		}
	})

	t.Run("non-executable", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd-tui")
		writeLegacyRegular(t, exe, 0755)
		writeLegacyRegular(t, filepath.Join(dir, "asdx"), 0644)
		fake := &fakeLegacyDeps{executable: exe}
		var stdout, stderr bytes.Buffer
		if code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr); !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if !strings.Contains(stderr.String(), "is not executable") {
			t.Fatalf("stderr %q does not name the non-executable target", stderr.String())
		}
	})

	t.Run("symlinked target", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd-tui")
		writeLegacyRegular(t, exe, 0755)
		writeLegacyRegular(t, filepath.Join(dir, "real-asdx"), 0755)
		if err := os.Symlink(filepath.Join(dir, "real-asdx"), filepath.Join(dir, "asdx")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		fake := &fakeLegacyDeps{executable: exe}
		var stdout, stderr bytes.Buffer
		if code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr); !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if !strings.Contains(stderr.String(), "refusing the symlinked") {
			t.Fatalf("stderr %q does not refuse the symlink", stderr.String())
		}
	})

	t.Run("no PATH search", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd-tui")
		writeLegacyRegular(t, exe, 0755)
		pathDir := t.TempDir()
		writeLegacyRegular(t, filepath.Join(pathDir, "asdx"), 0755)
		t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		fake := &fakeLegacyDeps{executable: exe}
		var stdout, stderr bytes.Buffer
		if code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr); !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if fake.execCall != 0 {
			t.Fatal("a PATH target was executed")
		}
		if !strings.Contains(stderr.String(), "is not installed beside") {
			t.Fatalf("stderr %q used a PATH fallback", stderr.String())
		}
	})

	t.Run("exec failure", func(t *testing.T) {
		exe := legacyFixture(t, "asdx")
		fake := &fakeLegacyDeps{executable: exe, execErr: errors.New("exec format error")}
		var stdout, stderr bytes.Buffer
		if code, handled := runLegacyCompatibility(fake.deps(), nil, &stdout, &stderr); !handled || code != 1 {
			t.Fatalf("handled=%v code=%d, want true/1", handled, code)
		}
		if !strings.Contains(stderr.String(), "exec format error") {
			t.Fatalf("stderr %q does not carry the replacement failure", stderr.String())
		}
	})
}

func TestLegacyProtocolInfoWithOtherFlagsStaysSilent(t *testing.T) {
	withLegacyBuild(t, "asdx", "agentic-sdd-tui")
	exe := legacyFixture(t, "asdx")
	fake := &fakeLegacyDeps{executable: exe}
	args := []string{"--protocol-info", "--with-opencode=false"}
	var stdout, stderr bytes.Buffer
	code, handled := runLegacyCompatibility(fake.deps(), args, &stdout, &stderr)
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d, want true/0", handled, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("a protocol query with other flags wrote to stderr: %q", stderr.String())
	}
	wantArgv := append([]string{filepath.Join(filepath.Dir(exe), "asdx")}, args...)
	if !reflect.DeepEqual(fake.execArgv, wantArgv) {
		t.Fatalf("exec argv = %v, want %v", fake.execArgv, wantArgv)
	}
}
