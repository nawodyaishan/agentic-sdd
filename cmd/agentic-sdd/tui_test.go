package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"agentic-sdd/internal/buildinfo"
)

// This file covers Feature 013 B6/T14: the explicit "agentic-sdd tui" command,
// the Homebrew real-interactive zero-argument default, the trusted sibling
// handshake and the parent's child-signal policy. Every process interaction is
// injected; no test builds, resolves or signals a real companion binary and no
// test touches the real user home.

// tuiTestDistribution overrides the link-time marker for one test and restores
// it afterwards.
func tuiTestDistribution(t *testing.T, distribution buildinfo.Distribution) {
	t.Helper()
	original := currentDistribution
	currentDistribution = func() buildinfo.Distribution { return distribution }
	t.Cleanup(func() { currentDistribution = original })
}

// tuiTestFixture creates a tiny source tree and home so preview runs are fast,
// deterministic and confined to a temporary directory.
func tuiTestFixture(t *testing.T) (repo, home string) {
	t.Helper()
	root := t.TempDir()
	repo, home = filepath.Join(root, "repo"), filepath.Join(root, "home")
	source := filepath.Join(repo, "skills-files", "agentic-sdd-plan")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	return repo, home
}

// TestTUIDispatchMatrix pins the full dispatch decision: arguments × real
// terminal facts × distribution marker × forced-interactive mode.
func TestTUIDispatchMatrix(t *testing.T) {
	repo, home := tuiTestFixture(t)
	// A bare invocation that keeps the historical preview resolves the default
	// home, so keep that inside the temporary directory too.
	t.Setenv("HOME", home)

	type launchCall struct {
		called bool
		req    tuiRequest
	}
	cases := []struct {
		name         string
		distribution buildinfo.Distribution
		args         []string
		stdinTTY     bool
		stdoutTTY    bool
		force        bool
		wantLaunch   bool
		wantExplicit bool
		wantCode     int
		wantInStdout string
		wantInStderr string
	}{
		{
			name:         "homebrew bare interactive launches",
			distribution: buildinfo.DistributionHomebrew,
			stdinTTY:     true,
			stdoutTTY:    true,
			wantLaunch:   true,
			wantExplicit: false,
			wantCode:     0,
			wantInStdout: "tui-launch",
		},
		{
			name:         "homebrew bare stdin pipe keeps preview",
			distribution: buildinfo.DistributionHomebrew,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "install agents/agentic-sdd-plan",
		},
		{
			name:         "homebrew bare stdout pipe keeps preview",
			distribution: buildinfo.DistributionHomebrew,
			stdinTTY:     true,
			wantCode:     0,
			wantInStdout: "Preview only. Run with apply (or --apply) to install.",
		},
		{
			name:         "homebrew forced interactive keeps preview",
			distribution: buildinfo.DistributionHomebrew,
			stdinTTY:     true,
			stdoutTTY:    true,
			force:        true,
			wantCode:     0,
			wantInStdout: "Preview only. Run with apply (or --apply) to install.",
		},
		{
			name:         "generic bare interactive keeps preview",
			distribution: buildinfo.DistributionGeneric,
			stdinTTY:     true,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "Preview only. Run with apply (or --apply) to install.",
		},
		{
			name:         "development bare interactive keeps preview",
			distribution: buildinfo.DistributionDev,
			stdinTTY:     true,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "Preview only. Run with apply (or --apply) to install.",
		},
		{
			name:         "homebrew explicit preview keeps preview",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"preview"},
			stdinTTY:     true,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "Preview only. Run with apply (or --apply) to install.",
		},
		{
			name:         "homebrew explicit version keeps version",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"--version"},
			stdinTTY:     true,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "agentic-sdd",
		},
		{
			name:         "homebrew explicit help keeps help",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"help"},
			stdinTTY:     true,
			stdoutTTY:    true,
			wantCode:     0,
			wantInStdout: "Usage: agentic-sdd",
		},
		{
			name:         "explicit tui on development marker launches",
			distribution: buildinfo.DistributionDev,
			args:         []string{"tui"},
			stdinTTY:     true,
			stdoutTTY:    true,
			wantLaunch:   true,
			wantExplicit: true,
			wantCode:     0,
			wantInStdout: "tui-launch",
		},
		{
			name:         "explicit tui on generic marker launches",
			distribution: buildinfo.DistributionGeneric,
			args:         []string{"tui"},
			stdinTTY:     true,
			stdoutTTY:    true,
			wantLaunch:   true,
			wantExplicit: true,
			wantCode:     0,
			wantInStdout: "tui-launch",
		},
		{
			name:         "explicit tui requires input terminal",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"tui"},
			stdoutTTY:    true,
			wantCode:     2,
			wantInStderr: "needs a real input and output terminal",
		},
		{
			name:         "explicit tui requires output terminal",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"tui"},
			stdinTTY:     true,
			wantCode:     2,
			wantInStderr: "needs a real input and output terminal",
		},
		{
			name:         "explicit tui refuses forced-interactive",
			distribution: buildinfo.DistributionHomebrew,
			args:         []string{"tui"},
			stdinTTY:     true,
			stdoutTTY:    true,
			force:        true,
			wantCode:     2,
			wantInStderr: "AGENTIC_SDD_FORCE_INTERACTIVE=1 keeps scripted runs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tuiTestDistribution(t, tc.distribution)
			var call launchCall
			original := tuiLaunch
			tuiLaunch = func(req tuiRequest, _ io.Reader, stdout io.Writer, _ io.Writer, _ *Env) int {
				call.called, call.req = true, req
				_, _ = io.WriteString(stdout, "tui-launch\n")
				return 0
			}
			t.Cleanup(func() { tuiLaunch = original })

			args := append([]string(nil), tc.args...)
			if len(args) > 0 {
				switch args[0] {
				case "preview", "tui":
					args = append(args, "--repo", repo)
					if !containsFlag(args, "--home") {
						args = append(args, "--home", home)
					}
				}
			}
			env := &Env{
				Width:            80,
				RealStdinTTY:     tc.stdinTTY,
				RealStdoutTTY:    tc.stdoutTTY,
				ForceInteractive: tc.force,
			}
			var stdout, stderr bytes.Buffer
			code := runEnv(args, &stdout, &stderr, strings.NewReader(""), env)
			if code != tc.wantCode {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.wantCode, stdout.String(), stderr.String())
			}
			if call.called != tc.wantLaunch {
				t.Fatalf("launch called = %v, want %v; stdout %q; stderr %q",
					call.called, tc.wantLaunch, stdout.String(), stderr.String())
			}
			if call.called && call.req.Explicit != tc.wantExplicit {
				t.Fatalf("launch request Explicit = %v, want %v", call.req.Explicit, tc.wantExplicit)
			}
			if !call.called && strings.Contains(stdout.String(), "tui-launch") {
				t.Fatalf("preview path printed the launch marker: %q", stdout.String())
			}
			if tc.wantInStdout != "" && !strings.Contains(stdout.String(), tc.wantInStdout) {
				t.Fatalf("stdout %q does not contain %q", stdout.String(), tc.wantInStdout)
			}
			if tc.wantInStderr != "" && !strings.Contains(stderr.String(), tc.wantInStderr) {
				t.Fatalf("stderr %q does not contain %q", stderr.String(), tc.wantInStderr)
			}
		})
	}
}

func containsFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

// TestTUIExplicitRequestForwarding pins the request the explicit command builds
// for the companion, including which options the user actually supplied.
func TestTUIExplicitRequestForwarding(t *testing.T) {
	tuiTestDistribution(t, buildinfo.DistributionHomebrew)
	cases := []struct {
		name string
		args []string
		want tuiRequest
	}{
		{
			name: "defaults omit unset options",
			args: []string{"tui"},
			want: tuiRequest{Explicit: true, WithOpenCode: true},
		},
		{
			name: "home and repo forwarded",
			args: []string{"tui", "--home", "/tmp/example-home", "--repo", "/tmp/example-repo"},
			want: tuiRequest{Explicit: true, Home: "/tmp/example-home", Repo: "/tmp/example-repo", WithOpenCode: true},
		},
		{
			name: "explicit pack off forwarded",
			args: []string{"tui", "--with-opencode=false"},
			want: tuiRequest{Explicit: true, WithOpenCode: false, WithOpenCodeSet: true},
		},
		{
			name: "free-first forwarded",
			args: []string{"tui", "--opencode-free-first"},
			want: tuiRequest{Explicit: true, WithOpenCode: true, OpenCodeFreeFirst: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got tuiRequest
			original := tuiLaunch
			tuiLaunch = func(req tuiRequest, _ io.Reader, _ io.Writer, _ io.Writer, _ *Env) int {
				got = req
				return 0
			}
			t.Cleanup(func() { tuiLaunch = original })

			env := &Env{Width: 80, RealStdinTTY: true, RealStdoutTTY: true}
			var stdout, stderr bytes.Buffer
			if code := runTUICommand(tc.args[1:], strings.NewReader(""), &stdout, &stderr, env); code != 0 {
				t.Fatalf("exit %d, want 0; stderr %q", code, stderr.String())
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("request = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestTUICompanionArgs pins the exact argument vector handed to the child.
func TestTUICompanionArgs(t *testing.T) {
	cases := []struct {
		name string
		req  tuiRequest
		want []string
	}{
		{"defaults forward nothing", tuiRequest{}, nil},
		{"home and repo", tuiRequest{Home: "/h", Repo: "/r"}, []string{"--home", "/h", "--repo", "/r"}},
		{"pack on", tuiRequest{WithOpenCode: true, WithOpenCodeSet: true}, []string{"--with-opencode=true"}},
		{"pack off", tuiRequest{WithOpenCode: false, WithOpenCodeSet: true}, []string{"--with-opencode=false"}},
		{"free first", tuiRequest{WithOpenCode: true, OpenCodeFreeFirst: true}, []string{"--opencode-free-first"}},
		{
			"all options",
			tuiRequest{Home: "/h", Repo: "/r", WithOpenCode: false, WithOpenCodeSet: true, OpenCodeFreeFirst: true},
			[]string{"--home", "/h", "--repo", "/r", "--with-opencode=false", "--opencode-free-first"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tuiCompanionArgs(tc.req); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("tuiCompanionArgs() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestTUICommandUsageErrors pins the explicit command's own flag contract.
func TestTUICommandUsageErrors(t *testing.T) {
	env := &Env{Width: 80, RealStdinTTY: true, RealStdoutTTY: true}
	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"help", []string{"-h"}, 0, "Usage: agentic-sdd tui"},
		{"long help", []string{"--help"}, 0, "AGENTIC_SDD_FORCE_INTERACTIVE=1"},
		{"unknown flag", []string{"--wat"}, 2, "flag provided but not defined"},
		{"unexpected argument", []string{"extra"}, 2, `unexpected argument "extra"`},
		{"free first without pack", []string{"--with-opencode=false", "--opencode-free-first"}, 2, "--opencode-free-first requires --with-opencode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runTUICommand(tc.args, strings.NewReader(""), &stdout, &stderr, env); code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.code, stdout.String(), stderr.String())
			}
			if got := stdout.String() + stderr.String(); !strings.Contains(got, tc.want) {
				t.Fatalf("output %q does not contain %q", got, tc.want)
			}
		})
	}
}

// TestTUIWithoutTTYNeverLaunches proves the explicit command fails closed and
// never runs the historical preview.
func TestTUIWithoutTTYNeverLaunches(t *testing.T) {
	original := tuiLaunch
	tuiLaunch = func(tuiRequest, io.Reader, io.Writer, io.Writer, *Env) int {
		t.Fatal("tuiLaunch called without a real terminal")
		return 1
	}
	t.Cleanup(func() { tuiLaunch = original })

	var stdout, stderr bytes.Buffer
	code := runNonInteractive([]string{"tui"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit %d, want 2; stdout %q; stderr %q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty (no preview fallback)", stdout.String())
	}
	if !strings.Contains(stderr.String(), "needs a real input and output terminal") {
		t.Fatalf("stderr = %q, want the terminal requirement", stderr.String())
	}
}

// --- Sibling resolution --------------------------------------------------

// writeRegular creates a regular file with the given mode.
func writeRegular(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
}

func TestResolveTUICompanion(t *testing.T) {
	t.Run("regular executable sibling", func(t *testing.T) {
		dir := t.TempDir()
		exe := filepath.Join(dir, "agentic-sdd")
		writeRegular(t, exe, 0755)
		companion := filepath.Join(dir, "agentic-sdd-tui")
		writeRegular(t, companion, 0755)
		got, err := resolveTUICompanion(func() (string, error) { return exe, nil })
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if got != companion {
			t.Fatalf("companion = %q, want %q", got, companion)
		}
	})

	t.Run("running executable symlink resolves to the cellar", func(t *testing.T) {
		root := t.TempDir()
		cellar := filepath.Join(root, "Cellar", "agentic-sdd", "0.6.0", "bin")
		exe := filepath.Join(cellar, "agentic-sdd")
		writeRegular(t, exe, 0755)
		companion := filepath.Join(cellar, "agentic-sdd-tui")
		writeRegular(t, companion, 0755)
		link := filepath.Join(root, "bin", "agentic-sdd")
		if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(exe, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		got, err := resolveTUICompanion(func() (string, error) { return link, nil })
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if got != companion {
			t.Fatalf("companion = %q, want %q", got, companion)
		}
	})

	failures := []struct {
		name  string
		setup func(t *testing.T, dir string) (exe string)
		want  string
	}{
		{
			name: "absent companion",
			setup: func(t *testing.T, dir string) string {
				exe := filepath.Join(dir, "agentic-sdd")
				writeRegular(t, exe, 0755)
				return exe
			},
			want: "is not installed beside",
		},
		{
			name: "non-executable companion",
			setup: func(t *testing.T, dir string) string {
				exe := filepath.Join(dir, "agentic-sdd")
				writeRegular(t, exe, 0755)
				writeRegular(t, filepath.Join(dir, "agentic-sdd-tui"), 0644)
				return exe
			},
			want: "is not executable",
		},
		{
			name: "symlinked companion",
			setup: func(t *testing.T, dir string) string {
				exe := filepath.Join(dir, "agentic-sdd")
				writeRegular(t, exe, 0755)
				target := filepath.Join(dir, "elsewhere-tui")
				writeRegular(t, target, 0755)
				if err := os.Symlink(target, filepath.Join(dir, "agentic-sdd-tui")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				return exe
			},
			want: "refusing the symlinked companion",
		},
		{
			name: "companion is a directory",
			setup: func(t *testing.T, dir string) string {
				exe := filepath.Join(dir, "agentic-sdd")
				writeRegular(t, exe, 0755)
				if err := os.Mkdir(filepath.Join(dir, "agentic-sdd-tui"), 0755); err != nil {
					t.Fatal(err)
				}
				return exe
			},
			want: "is not a regular file",
		},
		{
			name: "companion is a named pipe",
			setup: func(t *testing.T, dir string) string {
				exe := filepath.Join(dir, "agentic-sdd")
				writeRegular(t, exe, 0755)
				if err := syscall.Mkfifo(filepath.Join(dir, "agentic-sdd-tui"), 0755); err != nil {
					t.Skipf("named pipes unavailable: %v", err)
				}
				return exe
			},
			want: "is not a regular file",
		},
	}
	for _, tc := range failures {
		t.Run(tc.name, func(t *testing.T) {
			exe := tc.setup(t, t.TempDir())
			if _, err := resolveTUICompanion(func() (string, error) { return exe, nil }); err == nil {
				t.Fatal("resolve succeeded, want a failure")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}

	t.Run("executable lookup failure", func(t *testing.T) {
		if _, err := resolveTUICompanion(func() (string, error) { return "", errors.New("gone") }); err == nil {
			t.Fatal("resolve succeeded, want a failure")
		}
	})

	t.Run("relative executable path", func(t *testing.T) {
		if _, err := resolveTUICompanion(func() (string, error) { return "agentic-sdd", nil }); err == nil {
			t.Fatal("resolve accepted a relative path")
		}
	})
}

// --- Protocol-info query -------------------------------------------------

// writeCompanionScript creates an executable POSIX shell fixture that stands in
// for the companion during the handshake.
func writeCompanionScript(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the handshake fixtures need a POSIX shell")
	}
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh is unavailable")
	}
	path := filepath.Join(t.TempDir(), "agentic-sdd-tui")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func tuiTestIdentity() buildinfo.Identity {
	return buildinfo.Identity{Revision: "v9.9.9", Commit: "0123456789abcdef", SourceDigest: "local-sha256:feedface"}
}

func TestQueryTUICompanionProtocol(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	payload, err := peer.JSON()
	if err != nil {
		t.Fatal(err)
	}
	singleQuoted := "'" + strings.ReplaceAll(strings.TrimSpace(string(payload)), "'", `'\''`) + "'"

	t.Run("valid response", func(t *testing.T) {
		path := writeCompanionScript(t, "printf '%s\\n' "+singleQuoted+"\n")
		got, err := queryTUICompanionProtocol(path)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		if !reflect.DeepEqual(got, peer) {
			t.Fatalf("protocol info = %+v, want %+v", got, peer)
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		path := writeCompanionScript(t, "echo '{not json'\n")
		if _, err := queryTUICompanionProtocol(path); err == nil || !strings.Contains(err.Error(), "parse companion protocol info") {
			t.Fatalf("error = %v, want a parse failure", err)
		}
	})

	t.Run("trailing data", func(t *testing.T) {
		path := writeCompanionScript(t, "printf '%s\\n{}\\n' "+singleQuoted+"\n")
		if _, err := queryTUICompanionProtocol(path); err == nil || !strings.Contains(err.Error(), "trailing data") {
			t.Fatalf("error = %v, want trailing data to fail", err)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		path := writeCompanionScript(t, `echo '{"product":"agentic-sdd-tui","surprise":1}'`+"\n")
		if _, err := queryTUICompanionProtocol(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("error = %v, want an unknown-field failure", err)
		}
	})

	t.Run("non-zero exit", func(t *testing.T) {
		path := writeCompanionScript(t, "echo 'companion exploded' >&2\nexit 3\n")
		_, err := queryTUICompanionProtocol(path)
		if err == nil || !strings.Contains(err.Error(), "companion exploded") {
			t.Fatalf("error = %v, want the companion's stderr detail", err)
		}
	})

	t.Run("missing binary", func(t *testing.T) {
		if _, err := queryTUICompanionProtocol(filepath.Join(t.TempDir(), "absent")); err == nil {
			t.Fatal("query succeeded, want a failure")
		}
	})
}

func TestBoundedBufferTruncates(t *testing.T) {
	var buf boundedBuffer
	buf.limit = 4
	if n, err := buf.Write([]byte("abcdef")); err != nil || n != 6 {
		t.Fatalf("Write = %d, %v; want 6, nil", n, err)
	}
	if !buf.truncated || buf.String() != "abcd" {
		t.Fatalf("buffer = %q truncated=%v, want %q true", buf.String(), buf.truncated, "abcd")
	}
	if _, err := buf.Write([]byte("x")); err != nil {
		t.Fatalf("overflow write: %v", err)
	}
	if buf.String() != "abcd" {
		t.Fatalf("buffer grew past its limit: %q", buf.String())
	}
}

// --- Handshake and launch ------------------------------------------------

// fakeTUIProcess is a controllable child for the parent-side policy tests.
type fakeTUIProcess struct {
	exitCode int
	waitErr  error
	release  chan struct{}
	signals  chan os.Signal
}

func (p *fakeTUIProcess) Wait() error {
	if p.release != nil {
		<-p.release
	}
	return p.waitErr
}

func (p *fakeTUIProcess) Signal(sig os.Signal) error {
	if p.signals != nil {
		p.signals <- sig
	}
	return nil
}

func (p *fakeTUIProcess) ExitCode() int { return p.exitCode }

// fakeTUIRunner builds a runner with a present executable and injected process
// steps. Individual tests override the steps they exercise.
type fakeTUIRunner struct {
	runner    *tuiRunner
	exe       string
	started   chan struct{}
	deliver   chan<- os.Signal
	stopped   chan struct{}
	startArg  []string
	startPath string
}

func newFakeTUIRunner(t *testing.T, child *fakeTUIProcess, peer buildinfo.ProtocolInfo) *fakeTUIRunner {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "agentic-sdd")
	writeRegular(t, exe, 0755)
	writeRegular(t, filepath.Join(dir, "agentic-sdd-tui"), 0755)

	fake := &fakeTUIRunner{
		exe:     exe,
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
	fake.runner = &tuiRunner{
		executable: func() (string, error) { return fake.exe, nil },
		query:      func(string) (buildinfo.ProtocolInfo, error) { return peer, nil },
		parentInfo: func() buildinfo.ProtocolInfo {
			return buildinfo.NewProtocolInfo(buildinfo.ProductCLI, buildinfo.DistributionHomebrew, tuiTestIdentity())
		},
		notify: func(c chan<- os.Signal, _ ...os.Signal) { fake.deliver = c },
		stop:   func(chan<- os.Signal) { close(fake.stopped) },
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	fake.runner.start = func(path string, args []string, _ io.Reader, _, _ io.Writer) (tuiProcess, error) {
		fake.startPath, fake.startArg = path, args
		close(fake.started)
		return child, nil
	}
	return fake
}

// waitForSignal waits until the fake child records one relayed signal.
func (f *fakeTUIRunner) waitForSignal(t *testing.T, child *fakeTUIProcess) os.Signal {
	t.Helper()
	select {
	case sig := <-child.signals:
		return sig
	case <-time.After(2 * time.Second):
		t.Fatal("the parent never relayed a signal to the child")
		return nil
	}
}

func TestLaunchTUIHandshake(t *testing.T) {
	validIdentity := tuiTestIdentity()
	validPeer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, validIdentity)

	cases := []struct {
		name        string
		parentDist  buildinfo.Distribution
		parentID    buildinfo.Identity
		peer        buildinfo.ProtocolInfo
		wantStart   bool
		wantInError string
	}{
		{
			name:       "matched pair starts",
			parentDist: buildinfo.DistributionHomebrew,
			parentID:   validIdentity,
			peer:       validPeer,
			wantStart:  true,
		},
		{
			name:        "wrong product fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    validIdentity,
			peer:        buildinfo.NewProtocolInfo(buildinfo.ProductCLI, buildinfo.DistributionGeneric, validIdentity),
			wantInError: "reported product",
		},
		{
			name:        "protocol revision mismatch fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    validIdentity,
			peer:        func() buildinfo.ProtocolInfo { p := validPeer; p.ProtocolRevision++; return p }(),
			wantInError: "protocol revision",
		},
		{
			name:        "uninjected parent identity fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    buildinfo.Identity{},
			peer:        validPeer,
			wantInError: "cannot verify the companion",
		},
		{
			name:        "uninjected companion identity fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    validIdentity,
			peer:        buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, buildinfo.Identity{}),
			wantInError: "companion identity is not verifiable",
		},
		{
			name:        "mismatched identity fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    validIdentity,
			peer:        buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, buildinfo.Identity{Revision: "v0.0.1", Commit: "c", SourceDigest: "d"}),
			wantInError: "does not match this CLI",
		},
		{
			name:        "unknown parent marker fails closed",
			parentDist:  buildinfo.Distribution("mystery"),
			parentID:    validIdentity,
			peer:        validPeer,
			wantInError: "unknown distribution marker",
		},
		{
			name:        "unknown companion marker fails closed",
			parentDist:  buildinfo.DistributionHomebrew,
			parentID:    validIdentity,
			peer:        buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.Distribution("mystery"), validIdentity),
			wantInError: "unknown distribution marker",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			child := &fakeTUIProcess{exitCode: 0, release: make(chan struct{})}
			close(child.release)
			child.signals = make(chan os.Signal, 4)
			fake := newFakeTUIRunner(t, child, tc.peer)
			fake.runner.parentInfo = func() buildinfo.ProtocolInfo {
				return buildinfo.NewProtocolInfo(buildinfo.ProductCLI, tc.parentDist, tc.parentID)
			}

			var stdout, stderr bytes.Buffer
			fake.runner.stdout, fake.runner.stderr = &stdout, &stderr
			code := launchTUI(fake.runner, tuiRequest{Explicit: true}, &Env{Width: 80})
			started := true
			select {
			case <-fake.started:
			default:
				started = false
			}
			if started != tc.wantStart {
				t.Fatalf("child started = %v, want %v; code %d; stderr %q", started, tc.wantStart, code, stderr.String())
			}
			if tc.wantStart {
				if code != 0 {
					t.Fatalf("exit %d, want 0; stderr %q", code, stderr.String())
				}
				if stdout.String() != "" {
					t.Fatalf("stdout = %q, want empty", stdout.String())
				}
				return
			}
			if code != 1 {
				t.Fatalf("exit %d, want 1; stderr %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.wantInError) {
				t.Fatalf("stderr %q does not contain %q", stderr.String(), tc.wantInError)
			}
			if !strings.Contains(stderr.String(), "rebuild both binaries as a matched pair") {
				t.Fatalf("stderr %q is missing the rebuild-pair guidance", stderr.String())
			}
			if strings.Contains(stdout.String()+stderr.String(), "Preview only") ||
				strings.Contains(stdout.String()+stderr.String(), "install agents/") {
				t.Fatalf("a failed handshake silently ran preview: stdout %q; stderr %q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestLaunchTUIPropagatesExitCodeAndForwardsArguments(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	child := &fakeTUIProcess{exitCode: 7, release: make(chan struct{})}
	close(child.release)
	child.signals = make(chan os.Signal, 4)
	fake := newFakeTUIRunner(t, child, peer)

	var stdout, stderr bytes.Buffer
	fake.runner.stdout, fake.runner.stderr = &stdout, &stderr
	req := tuiRequest{Explicit: true, Home: "/tmp/home", Repo: "/tmp/repo", WithOpenCode: false, WithOpenCodeSet: true, OpenCodeFreeFirst: true}
	code := launchTUI(fake.runner, req, &Env{Width: 80})
	if code != 7 {
		t.Fatalf("exit %d, want the child's 7; stderr %q", code, stderr.String())
	}
	if want := []string{"--home", "/tmp/home", "--repo", "/tmp/repo", "--with-opencode=false", "--opencode-free-first"}; !reflect.DeepEqual(fake.startArg, want) {
		t.Fatalf("companion args = %v, want %v", fake.startArg, want)
	}
	if filepath.Base(fake.startPath) != buildinfo.ProductCompanion {
		t.Fatalf("started %q, want the verified sibling", fake.startPath)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty (the child owns the terminal)", stdout.String())
	}
}

func TestLaunchTUIRelaysHandledSignals(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	child := &fakeTUIProcess{exitCode: 0, release: make(chan struct{}), signals: make(chan os.Signal, 8)}
	fake := newFakeTUIRunner(t, child, peer)

	done := make(chan int, 1)
	go func() { done <- launchTUI(fake.runner, tuiRequest{Explicit: true}, &Env{Width: 80}) }()

	<-fake.started
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		fake.deliver <- sig
		if got := fake.waitForSignal(t, child); got != sig {
			t.Fatalf("child received %v, want %v", got, sig)
		}
	}
	close(child.release)
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("exit %d, want 0", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("launchTUI did not return after the child exited")
	}
	select {
	case <-fake.stopped:
	default:
		t.Fatal("signal delivery was not stopped after the child exited")
	}
}

func TestLaunchTUIQueuesSignalBeforeChildStarts(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	child := &fakeTUIProcess{exitCode: 0, release: make(chan struct{}), signals: make(chan os.Signal, 8)}
	fake := newFakeTUIRunner(t, child, peer)
	fake.runner.start = func(string, []string, io.Reader, io.Writer, io.Writer) (tuiProcess, error) {
		// A launch-time signal must be queued and delivered once the child
		// exists, not dropped and not fatal to the parent.
		fake.deliver <- syscall.SIGTERM
		return child, nil
	}
	done := make(chan int, 1)
	go func() { done <- launchTUI(fake.runner, tuiRequest{Explicit: true}, &Env{Width: 80}) }()
	if got := fake.waitForSignal(t, child); got != syscall.SIGTERM {
		t.Fatalf("child received %v, want SIGTERM", got)
	}
	close(child.release)
	<-done
}

func TestLaunchTUIReprocessFailureFailsClosed(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	child := &fakeTUIProcess{exitCode: 0, release: make(chan struct{})}
	close(child.release)
	child.signals = make(chan os.Signal, 4)
	fake := newFakeTUIRunner(t, child, peer)
	fake.runner.start = func(string, []string, io.Reader, io.Writer, io.Writer) (tuiProcess, error) {
		return nil, errors.New("exec format error")
	}
	var stdout, stderr bytes.Buffer
	fake.runner.stdout, fake.runner.stderr = &stdout, &stderr
	if code := launchTUI(fake.runner, tuiRequest{Explicit: true}, &Env{Width: 80}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "exec format error") {
		t.Fatalf("stderr %q does not contain the start failure", stderr.String())
	}
}

// TestLaunchTUIFailsClosedWithoutCompanion uses the production resolution path
// (only the executable path is injected) to prove an absent companion never
// turns into preview.
func TestLaunchTUIFailsClosedWithoutCompanion(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "agentic-sdd")
	writeRegular(t, exe, 0755)
	runner := &tuiRunner{
		executable: func() (string, error) { return exe, nil },
		query:      queryTUICompanionProtocol,
		start:      startTUICompanion,
		notify:     func(chan<- os.Signal, ...os.Signal) {},
		stop:       func(chan<- os.Signal) {},
		parentInfo: func() buildinfo.ProtocolInfo {
			return buildinfo.NewProtocolInfo(buildinfo.ProductCLI, buildinfo.DistributionHomebrew, tuiTestIdentity())
		},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	var stdout, stderr bytes.Buffer
	runner.stdout, runner.stderr = &stdout, &stderr
	if code := launchTUI(runner, tuiRequest{}, &Env{Width: 80}); code != 1 {
		t.Fatalf("exit %d, want 1; stderr %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "is not installed beside") {
		t.Fatalf("stderr %q does not name the missing companion", stderr.String())
	}
	if !strings.Contains(stderr.String(), "run agentic-sdd preview") {
		t.Fatalf("stderr %q does not suggest the explicit preview", stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty (no fallback)", stdout.String())
	}
}

// TestLaunchTUITraceRecordsDecision keeps the trace privacy contract: the new
// events name the source and exit code, never a filesystem path.
func TestLaunchTUITraceRecordsDecision(t *testing.T) {
	peer := buildinfo.NewProtocolInfo(buildinfo.ProductCompanion, buildinfo.DistributionGeneric, tuiTestIdentity())
	child := &fakeTUIProcess{exitCode: 0, release: make(chan struct{})}
	close(child.release)
	child.signals = make(chan os.Signal, 4)
	fake := newFakeTUIRunner(t, child, peer)

	var trace bytes.Buffer
	env := &Env{Width: 80, Trace: newTracer(&trace, nil, true)}
	if code := launchTUI(fake.runner, tuiRequest{}, env); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	lines := strings.Split(strings.TrimSpace(trace.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("trace lines = %d, want 2: %q", len(lines), trace.String())
	}
	var dispatch, result map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &dispatch); err != nil {
		t.Fatalf("decode dispatch event: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &result); err != nil {
		t.Fatalf("decode result event: %v", err)
	}
	if dispatch["event"] != "tui.dispatch" || dispatch["source"] != "homebrew-default" {
		t.Fatalf("dispatch event = %v", dispatch)
	}
	if result["event"] != "tui.result" || result["exit"] != float64(0) {
		t.Fatalf("result event = %v", result)
	}
	if strings.Contains(trace.String(), fake.exe) || strings.Contains(trace.String(), os.TempDir()) {
		t.Fatalf("trace leaked a filesystem path:\n%s", trace.String())
	}
}

// TestRunStartTraceNamesTUI proves the Homebrew default is traced under the
// command it actually runs.
func TestRunStartTraceNamesTUI(t *testing.T) {
	tuiTestDistribution(t, buildinfo.DistributionHomebrew)
	_, home := tuiTestFixture(t)
	t.Setenv("HOME", home)
	original := tuiLaunch
	tuiLaunch = func(tuiRequest, io.Reader, io.Writer, io.Writer, *Env) int { return 0 }
	t.Cleanup(func() { tuiLaunch = original })

	var trace bytes.Buffer
	env := &Env{Width: 80, RealStdinTTY: true, RealStdoutTTY: true, Trace: newTracer(&trace, nil, true)}
	var stdout, stderr bytes.Buffer
	if code := runEnv(nil, &stdout, &stderr, strings.NewReader(""), env); code != 0 {
		t.Fatalf("exit %d, want 0; stderr %q", code, stderr.String())
	}
	first := strings.SplitN(strings.TrimSpace(trace.String()), "\n", 2)[0]
	var event map[string]any
	if err := json.Unmarshal([]byte(first), &event); err != nil {
		t.Fatalf("decode trace: %v", err)
	}
	if event["event"] != "run.start" || event["command"] != "tui" {
		t.Fatalf("run.start = %v, want command tui", event)
	}
}
