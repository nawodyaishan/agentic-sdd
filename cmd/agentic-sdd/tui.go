package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"agentic-sdd/internal/buildinfo"
)

// This file implements the Feature 013 B6/T14 launcher. Two entry points share
// one launch path:
//
//   - the explicit "agentic-sdd tui" command;
//   - the Homebrew-marked artifact's real-interactive zero-argument default.
//
// Every other invocation keeps its historical command path. The launcher only
// resolves, verifies and starts the trusted same-install companion
// "agentic-sdd-tui"; it never searches PATH, never uses a shell, never inspects
// the environment beyond the documented switches the rest of the CLI already
// reads, and never falls back to another write workflow.

const (
	// tuiProtocolTimeout bounds the non-interactive handshake. A companion that
	// does not answer is treated as broken, not as an absent feature.
	tuiProtocolTimeout = 10 * time.Second

	// tuiProtocolLimit bounds how many bytes of the handshake response are
	// kept, so a misbehaving companion cannot exhaust parent memory.
	tuiProtocolLimit = 64 << 10

	// tuiSignalBuffer is large enough for a short burst of repeated signals
	// without blocking the os/signal relay goroutine.
	tuiSignalBuffer = 8
)

// currentDistribution is the link-time packaging marker of this artifact.
// Tests replace it; production reads the injected value.
var currentDistribution = buildinfo.CurrentDistribution

// tuiLaunch is the process boundary used by the dispatcher. Tests replace it to
// observe the dispatch decision without building, verifying or starting a
// companion process.
var tuiLaunch = launchTUISession

// tuiRequest is one fully parsed terminal-UI launch request. The zero value is
// the Homebrew zero-argument default: no flags at all.
type tuiRequest struct {
	// Explicit reports that the user typed "agentic-sdd tui" instead of relying
	// on the Homebrew interactive default.
	Explicit bool
	// Home and Repo are forwarded unchanged; empty values keep the companion's
	// own defaults (the current user home and the embedded assets).
	Home string
	Repo string
	// WithOpenCode mirrors the companion's pack option, whose default is on.
	// WithOpenCodeSet records an explicit "--with-opencode[=BOOL]" so an
	// untouched option keeps the companion's default instead of the CLI's.
	WithOpenCode      bool
	WithOpenCodeSet   bool
	OpenCodeFreeFirst bool
}

// tuiProcess is one started companion process. It is an interface so the
// parent-side signal and exit-code policy can be exercised with a fake child.
type tuiProcess interface {
	// Wait waits for the process to exit.
	Wait() error
	// Signal relays a handled signal to the companion.
	Signal(os.Signal) error
	// ExitCode returns the process exit code, or -1 when it was signalled.
	ExitCode() int
}

// tuiRunner carries every process interaction the launcher performs. New tests
// inject fakes; production uses newTUIRunner.
type tuiRunner struct {
	executable func() (string, error)
	query      func(path string) (buildinfo.ProtocolInfo, error)
	start      func(path string, args []string, stdin io.Reader, stdout, stderr io.Writer) (tuiProcess, error)
	notify     func(chan<- os.Signal, ...os.Signal)
	stop       func(chan<- os.Signal)
	parentInfo func() buildinfo.ProtocolInfo

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// newTUIRunner wires the production process interactions around the parent
// streams. The child inherits the real descriptors, so terminal ownership stays
// inside the companion.
func newTUIRunner(stdin io.Reader, stdout, stderr io.Writer) *tuiRunner {
	runner := &tuiRunner{
		executable: os.Executable,
		query:      queryTUICompanionProtocol,
		start:      startTUICompanion,
		notify:     signal.Notify,
		stop:       signal.Stop,
		parentInfo: func() buildinfo.ProtocolInfo {
			return buildinfo.CurrentProtocolInfo(buildinfo.ProductCLI)
		},
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
	return runner
}

// launchTUISession is the production entry point shared by the explicit
// command and the Homebrew default.
func launchTUISession(req tuiRequest, stdin io.Reader, stdout, stderr io.Writer, env *Env) int {
	return launchTUI(newTUIRunner(stdin, stdout, stderr), req, env)
}

// tuiDefaultLaunch reports whether a zero-argument invocation must open the
// terminal UI instead of the historical preview. Only the explicitly
// Homebrew-marked artifact opts in, and only for a real, unforced interactive
// terminal. Every other bare invocation keeps the pre-feature behavior.
func (e *Env) tuiDefaultLaunch() bool {
	if e == nil {
		return false
	}
	if !currentDistribution().AllowsInteractiveDefault() {
		return false
	}
	return e.RealStdinTTY && e.RealStdoutTTY && !e.ForceInteractive
}

// runTUICommand implements "agentic-sdd tui". It accepts exactly the supported
// companion options, applies the same real-terminal gates as a direct
// companion launch, and then runs the verified sibling.
func runTUICommand(args []string, stdin io.Reader, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("agentic-sdd tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var home, repo string
	var withOpenCode, openCodeFreeFirst bool
	// The companion's documented default is on; the historical preview/apply
	// default of the CLI is off and stays untouched.
	fs.StringVar(&home, "home", "", "home directory whose client folders are inspected (default: your home)")
	fs.StringVar(&repo, "repo", "", "skills checkout to read (default: skills embedded in this binary)")
	fs.BoolVar(&withOpenCode, "with-opencode", true, "include the optional OpenCode pack and profiles (default true)")
	fs.BoolVar(&openCodeFreeFirst, "opencode-free-first", false, "preset OpenCode profiles to Free-first models (requires --with-opencode)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printTUIUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		env.Hint(stderr, "run agentic-sdd help tui for the accepted flags")
		printUsage(stderr)
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		printUsage(stderr)
		return 2
	}
	if openCodeFreeFirst && !withOpenCode {
		fmt.Fprintln(stderr, "error: --opencode-free-first requires --with-opencode")
		env.Hint(stderr, "add --with-opencode, or drop --opencode-free-first")
		return 2
	}
	withOpenCodeSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "with-opencode" {
			withOpenCodeSet = true
		}
	})
	// The companion repeats these gates; checking them here keeps a scripted
	// run from even starting a second process and never opens an alternate
	// screen. A failure is explicit, never a silent preview.
	if env.ForceInteractive {
		fmt.Fprintf(stderr, "error: %s=1 keeps scripted runs on the standard CLI\n", "AGENTIC_SDD_FORCE_INTERACTIVE")
		env.Hint(stderr, "run agentic-sdd preview to use the read-only CLI preview")
		return 2
	}
	if !env.RealStdinTTY || !env.RealStdoutTTY {
		fmt.Fprintln(stderr, "error: agentic-sdd tui needs a real input and output terminal")
		env.Hint(stderr, "run agentic-sdd preview to use the read-only CLI preview")
		return 2
	}
	return tuiLaunch(tuiRequest{
		Explicit:          true,
		Home:              home,
		Repo:              repo,
		WithOpenCode:      withOpenCode,
		WithOpenCodeSet:   withOpenCodeSet,
		OpenCodeFreeFirst: openCodeFreeFirst,
	}, stdin, stdout, stderr, env)
}

// printTUIUsage is the explicit command's own flag reference. It is a new
// help surface, so the historical usage block keeps its shape.
func printTUIUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd tui [--repo PATH] [--home PATH] [--with-opencode[=false]] [--opencode-free-first]

Opens the packaged terminal UI from the same installation. It needs a real
input and output terminal; a pipe, a script, or
AGENTIC_SDD_FORCE_INTERACTIVE=1 keeps the standard CLI and never opens an
alternate screen. A missing or mismatched companion is an error, never a
silent fallback.

Options:
  --home PATH            Home directory whose client folders are inspected.
  --repo PATH            Skills checkout to read (default: embedded assets).
  --with-opencode        Include the optional OpenCode pack and profiles (default true).
  --opencode-free-first  Preset OpenCode profiles to Free-first models
                         (requires --with-opencode).
  -h, --help             Show this help.

Presentation (environment):
  NO_COLOR                     Disable styled output.
  AGENTIC_SDD_ASCII=1          Use ASCII borders and symbols.
  AGENTIC_SDD_REDUCED_MOTION=1 Disable animation.
  AGENTIC_SDD_FORCE_INTERACTIVE=1
                               Keep scripted runs on the standard CLI.

The TUI writes only after an explicit confirmation inside the UI; it never
substitutes a different write workflow.
`)
}

// tuiCompanionArgs builds the companion's argument vector. Only options the
// user actually supplied are forwarded so the companion keeps its own
// documented defaults.
func tuiCompanionArgs(req tuiRequest) []string {
	var args []string
	if req.Home != "" {
		args = append(args, "--home", req.Home)
	}
	if req.Repo != "" {
		args = append(args, "--repo", req.Repo)
	}
	if req.WithOpenCodeSet {
		args = append(args, fmt.Sprintf("--with-opencode=%t", req.WithOpenCode))
	}
	if req.OpenCodeFreeFirst {
		args = append(args, "--opencode-free-first")
	}
	return args
}

// launchTUI resolves, verifies and runs the trusted companion. Any failure
// before or during the launch is reported and returned as a non-zero exit; the
// parent never substitutes preview or a different write workflow.
func launchTUI(runner *tuiRunner, req tuiRequest, env *Env) int {
	source := "homebrew-default"
	if req.Explicit {
		source = "explicit"
	}
	if env != nil {
		env.Trace.Event("tui.dispatch", "source", source, "decision", "launch")
	}

	path, err := resolveTUICompanion(runner.executable)
	if err != nil {
		return failTUILaunch(runner.stderr, err)
	}
	peer, err := runner.query(path)
	if err != nil {
		return failTUILaunch(runner.stderr, err)
	}
	parent := runner.parentInfo()
	if err := buildinfo.VerifyCompanion(parent, peer); err != nil {
		return failTUILaunch(runner.stderr, err)
	}

	// Register the handled signals before Start so a signal that arrives during
	// launch is queued and relayed instead of killing the parent and orphaning
	// the child in the alternate screen.
	signals := make(chan os.Signal, tuiSignalBuffer)
	runner.notify(signals, tuiShutdownSignals()...)
	forwarder := newSignalForwarder(signals)
	defer func() {
		runner.stop(signals)
		forwarder.close()
	}()

	child, err := runner.start(path, tuiCompanionArgs(req), runner.stdin, runner.stdout, runner.stderr)
	if err != nil {
		return failTUILaunch(runner.stderr, err)
	}
	forwarder.attach(child)
	waitErr := child.Wait()

	code := child.ExitCode()
	if code < 0 {
		// A signalled child has no portable exit code of its own; report the
		// abnormal termination instead of inventing success or a shell code.
		fmt.Fprint(runner.stderr, "error: the terminal UI terminated abnormally")
		if waitErr != nil {
			fmt.Fprintf(runner.stderr, ": %v", waitErr)
		}
		fmt.Fprintln(runner.stderr)
		code = 1
	}
	if env != nil {
		env.Trace.Event("tui.result", "source", source, "exit", code)
	}
	return code
}

// failTUILaunch reports one closed failure with rebuild-pair guidance and the
// explicit preview escape hatch. It never runs preview, and it always prints
// the escape hatch because this path replaces the interactive default.
func failTUILaunch(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: terminal UI unavailable: %v\n", err)
	fmt.Fprintln(stderr, "hint: run agentic-sdd preview to use the standard CLI; nothing was written")
	return 1
}

// resolveTUICompanion returns the trusted sibling next to the running
// executable. The running path is fully resolved first (Homebrew installs the
// CLI as a symlink into its Cellar), then the sibling must be a real regular
// executable file in that same directory. There is no PATH search, shell,
// package-manager query or working-directory fallback.
func resolveTUICompanion(executable func() (string, error)) (string, error) {
	exe, err := executable()
	if err != nil {
		return "", fmt.Errorf("locate the running agentic-sdd binary: %w; %s", err, buildinfo.RebuildPairHint)
	}
	if strings.TrimSpace(exe) == "" {
		return "", fmt.Errorf("the running agentic-sdd binary has no path; %s", buildinfo.RebuildPairHint)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve the running agentic-sdd binary: %w; %s", err, buildinfo.RebuildPairHint)
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("the running agentic-sdd binary resolved to a relative path; %s", buildinfo.RebuildPairHint)
	}
	companion := filepath.Join(filepath.Dir(resolved), buildinfo.ProductCompanion)
	info, err := os.Lstat(companion)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("the companion %s is not installed beside %s; %s",
				buildinfo.ProductCompanion, filepath.Base(resolved), buildinfo.RebuildPairHint)
		}
		return "", fmt.Errorf("inspect the companion %s: %w; %s", companion, err, buildinfo.RebuildPairHint)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing the symlinked companion %s; install the real paired binary; %s",
			companion, buildinfo.RebuildPairHint)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("the companion %s is not a regular file; %s", companion, buildinfo.RebuildPairHint)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("the companion %s is not executable; %s", companion, buildinfo.RebuildPairHint)
	}
	return companion, nil
}

// queryTUICompanionProtocol runs the private, non-interactive handshake. The
// response is bounded and strictly parsed; a timeout, a non-zero exit or an
// unparseable document is a hard failure.
func queryTUICompanionProtocol(path string) (buildinfo.ProtocolInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tuiProtocolTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--protocol-info")
	// Non-interactive by construction: no stdin, inherited environment so the
	// companion can read only the same documented switches this CLI already
	// uses. The parent never enumerates or filters credentials.
	cmd.Stdin = nil
	var stdout, stderr boundedBuffer
	stdout.limit = tuiProtocolLimit
	stderr.limit = tuiProtocolLimit
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return buildinfo.ProtocolInfo{}, fmt.Errorf(
			"%s --protocol-info did not answer within %s; %s",
			buildinfo.ProductCompanion, tuiProtocolTimeout, buildinfo.RebuildPairHint)
	}
	if err != nil {
		detail := ""
		if text := strings.TrimSpace(stderr.String()); text != "" {
			detail = ": " + text
		}
		return buildinfo.ProtocolInfo{}, fmt.Errorf(
			"%s --protocol-info failed: %w%s; %s",
			buildinfo.ProductCompanion, err, detail, buildinfo.RebuildPairHint)
	}
	if stdout.truncated {
		return buildinfo.ProtocolInfo{}, fmt.Errorf(
			"%s --protocol-info response exceeded %d bytes; %s",
			buildinfo.ProductCompanion, tuiProtocolLimit, buildinfo.RebuildPairHint)
	}
	return buildinfo.ParseProtocolInfo(stdout.Bytes())
}

// startTUICompanion starts the verified companion with the parent's real
// streams. When the streams are *os.File values (the normal path) the child
// inherits the descriptors, so terminal handling stays entirely in the
// companion.
func startTUICompanion(path string, args []string, stdin io.Reader, stdout, stderr io.Writer) (tuiProcess, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w; %s", filepath.Base(path), err, buildinfo.RebuildPairHint)
	}
	return &execTUIProcess{cmd: cmd}, nil
}

// execTUIProcess adapts *exec.Cmd to tuiProcess.
type execTUIProcess struct {
	cmd *exec.Cmd
}

func (p *execTUIProcess) Wait() error                { return p.cmd.Wait() }
func (p *execTUIProcess) Signal(sig os.Signal) error { return p.cmd.Process.Signal(sig) }

// ExitCode reports the child's own exit code, or -1 when it was terminated by
// a signal.
func (p *execTUIProcess) ExitCode() int {
	if p.cmd.ProcessState == nil {
		return -1
	}
	return p.cmd.ProcessState.ExitCode()
}

// boundedBuffer keeps at most limit bytes and records truncation. It never
// fails a write, so a chatty child is bounded rather than turned into a pipe
// error the parent could mistake for a protocol failure.
type boundedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	room := b.limit - b.buf.Len()
	if room <= 0 {
		b.truncated = true
		return n, nil
	}
	if len(p) > room {
		b.truncated = true
		p = p[:room]
	}
	_, _ = b.buf.Write(p)
	return n, nil
}

func (b *boundedBuffer) Bytes() []byte  { return b.buf.Bytes() }
func (b *boundedBuffer) String() string { return b.buf.String() }

// tuiShutdownSignals are the OS signals the parent relays to the companion.
// The companion owns the busy-state policy: it defers exit through an
// in-flight write and restores the terminal itself.
func tuiShutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}
}

// signalForwarder relays handled signals from the parent's signal channel to
// the child. Signals that arrive before the child exists are queued, so a
// launch-time signal cannot orphan a process holding the alternate screen.
type signalForwarder struct {
	signals chan os.Signal
	done    chan struct{}

	mu      sync.Mutex
	child   tuiProcess
	pending []os.Signal
}

func newSignalForwarder(signals chan os.Signal) *signalForwarder {
	forwarder := &signalForwarder{signals: signals, done: make(chan struct{})}
	go forwarder.loop()
	return forwarder
}

func (f *signalForwarder) loop() {
	defer close(f.done)
	for sig := range f.signals {
		f.mu.Lock()
		f.relay(sig)
		f.mu.Unlock()
	}
}

// relay forwards one signal. The caller holds f.mu.
func (f *signalForwarder) relay(sig os.Signal) {
	if f.child == nil {
		f.pending = append(f.pending, sig)
		return
	}
	// An already-exited child rejects the signal; that is not a parent error
	// because the child's exit code is still propagated.
	_ = f.child.Signal(sig)
}

// attach connects the started child and relays anything queued meanwhile.
func (f *signalForwarder) attach(child tuiProcess) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.child = child
	for _, sig := range f.pending {
		f.relay(sig)
	}
	f.pending = nil
}

// close stops the relay goroutine. The caller must have stopped signal
// delivery first (signal.Stop guarantees no more sends).
func (f *signalForwarder) close() {
	close(f.signals)
	<-f.done
}
