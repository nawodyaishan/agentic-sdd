// Command tui-companion is the packaged, non-experimental terminal UI
// companion for the asdx-core CLI.
//
// It is a separate binary built from the nested experiments/tui module so the
// root CLI keeps its standard-library-only dependency contract. The root CLI
// starts this exact sibling binary for explicit `asdx-core tui`; it never
// searches PATH or a shell. The installed canonical name is `asdx`.
//
// The companion has a direct public interface too:
//
//	asdx --help
//	asdx --version
//	asdx --protocol-info
//
// Those three queries are non-interactive and perform no network or filesystem
// inspection. A direct interactive launch obeys the same forced-interactive and
// real-TTY gates as the CLI path.
//
// The same source is dual-built as the legacy `agentic-sdd-tui` compatibility
// name: a build-time legacyTarget replaces that process with the sibling `asdx`
// after a stderr-only deprecation notice. `--protocol-info` is forwarded as pure
// JSON with no notice so an older parent can verify or reject the pair cleanly.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"

	"agentic-sdd/experiments/tui/cmd/internal/tuirun"
	"agentic-sdd/internal/buildinfo"
)

// legacyTarget and legacyName are injected only into the legacy compatibility
// build of this command. The canonical asdx build leaves both empty; the
// release and local pair tasks dual-build the same source and inject:
//
//	-X main.legacyTarget=asdx -X main.legacyName=agentic-sdd-tui
var (
	legacyTarget string
	legacyName   string
)

// legacyDeps carries the process interactions the compatibility wrapper
// performs. Tests inject fakes so they never replace the test process.
type legacyDeps struct {
	executable func() (string, error)
	exec       func(path string, argv []string) error
}

func productionLegacyDeps() legacyDeps {
	return legacyDeps{executable: os.Executable, exec: execLegacyTarget}
}

func main() {
	if code, handled := runLegacyCompatibility(productionLegacyDeps(), os.Args[1:], os.Stdout, os.Stderr); handled {
		os.Exit(code)
	}
	os.Exit(run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		environment{
			getenv:      os.Getenv,
			stdinTTY:    term.IsTerminal(os.Stdin.Fd()),
			stdoutTTY:   term.IsTerminal(os.Stdout.Fd()),
			userHomeDir: os.UserHomeDir,
		},
	))
}

// runLegacyCompatibility implements the dual-built compatibility name. It
// reports handled=false for the canonical build so the normal TUI runs
// unchanged. Only exact top-level help/version are answered locally;
// `--protocol-info` and every other invocation are replaced by the canonical
// sibling. The protocol query is forwarded as pure JSON with no deprecation
// notice; all other forwarded invocations add one stderr-only notice, so
// stdout, streams and exit codes stay unchanged.
func runLegacyCompatibility(deps legacyDeps, args []string, stdout, stderr io.Writer) (int, bool) {
	target := strings.TrimSpace(legacyTarget)
	if target == "" {
		return 0, false
	}
	name := strings.TrimSpace(legacyName)
	if name == "" {
		name = "agentic-sdd-tui"
	}
	protocolInfo := requestsProtocolInfo(args)
	if !protocolInfo {
		switch {
		case isExactTopLevelHelp(args):
			printLegacyHelp(stdout, name, target)
			return 0, true
		case isExactTopLevelVersion(args):
			printLegacyVersion(stdout, name, target)
			return 0, true
		}
	}

	path, err := resolveLegacyTarget(deps.executable, target)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s: %v\n", name, err)
		fmt.Fprintf(stderr, "hint: install %s beside %s, or rebuild the pair together\n", target, name)
		return 1, true
	}
	if !protocolInfo {
		fmt.Fprintf(stderr, "%s: deprecated compatibility name; running %s\n", name, target)
	}
	if err := deps.exec(path, append([]string{path}, args...)); err != nil {
		fmt.Fprintf(stderr, "error: %s: start %s: %v\n", name, target, err)
		return 1, true
	}
	return 0, true
}

// requestsProtocolInfo reports whether the forwarded argument vector asks the
// canonical TUI for its protocol response. The notice is suppressed for that
// query so stdout stays pure JSON. A flag terminator or an explicit --version
// (which the canonical TUI rejects as mutually exclusive) ends the scan, keeping
// positional arguments untouched.
func requestsProtocolInfo(args []string) bool {
	found := false
	for _, arg := range args {
		switch arg {
		case "--", "--version":
			return false
		case "--protocol-info":
			found = true
		}
	}
	return found
}

// isExactTopLevelHelp and isExactTopLevelVersion match only the bare query
// forms. Any extra argument, subcommand or flag combination is forwarded.
func isExactTopLevelHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}

func isExactTopLevelVersion(args []string) bool {
	return len(args) == 1 && args[0] == "--version"
}

// resolveLegacyTarget returns the canonical sibling next to the running
// executable. The running path is fully resolved first (Homebrew installs a
// symlink), then the target must be a same-directory regular executable. There
// is no PATH search, shell, package-manager query or working-directory
// fallback.
func resolveLegacyTarget(executable func() (string, error), target string) (string, error) {
	if target == "" || filepath.Base(target) != target || target == "." || target == ".." {
		return "", fmt.Errorf("the injected legacy target %q is not a bare command name; %s", target, buildinfo.RebuildPairHint)
	}
	exe, err := executable()
	if err != nil {
		return "", fmt.Errorf("locate the running compatibility binary: %w; %s", err, buildinfo.RebuildPairHint)
	}
	if strings.TrimSpace(exe) == "" {
		return "", fmt.Errorf("the running compatibility binary has no path; %s", buildinfo.RebuildPairHint)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve the running compatibility binary: %w; %s", err, buildinfo.RebuildPairHint)
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("the running compatibility binary resolved to a relative path; %s", buildinfo.RebuildPairHint)
	}
	candidate := filepath.Join(filepath.Dir(resolved), target)
	info, err := os.Lstat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s is not installed beside %s; %s", target, filepath.Base(resolved), buildinfo.RebuildPairHint)
		}
		return "", fmt.Errorf("inspect %s: %w; %s", candidate, err, buildinfo.RebuildPairHint)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing the symlinked %s; install the real paired binary; %s", candidate, buildinfo.RebuildPairHint)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file; %s", candidate, buildinfo.RebuildPairHint)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable; %s", candidate, buildinfo.RebuildPairHint)
	}
	return candidate, nil
}

// printLegacyHelp identifies the compatibility name and its canonical target,
// then prints the canonical grammar. It writes nothing to stderr.
func printLegacyHelp(out io.Writer, name, target string) {
	fmt.Fprintf(out, "%s is a deprecated compatibility name for %s.\n", name, target)
	fmt.Fprintf(out, "Run %s for the canonical terminal UI; the options are identical.\n\n", target)
	printUsage(out)
}

// printLegacyVersion identifies the compatibility name and its canonical
// target while keeping the canonical version fields intact.
func printLegacyVersion(out io.Writer, name, target string) {
	identity := buildinfo.CurrentIdentity()
	fmt.Fprintf(out, "%s %s (legacy compatibility name for %s)\n", name, identity.Revision, target)
	fmt.Fprintf(out, "  commit:        %s\n", identity.Commit)
	fmt.Fprintf(out, "  source digest: %s\n", identity.SourceDigest)
	fmt.Fprintf(out, "  distribution:  %s\n", buildinfo.CurrentDistribution())
	fmt.Fprintf(out, "  protocol:      %d\n", buildinfo.ProtocolRevision)
}

// environment carries the process facts the command may not read itself, so
// tests exercise the same paths without a terminal or a real home.
type environment struct {
	getenv      func(string) string
	stdinTTY    bool
	stdoutTTY   bool
	userHomeDir func() (string, error)
}

func run(args []string, stdout, stderr io.Writer, env environment) int {
	if env.getenv == nil {
		env.getenv = func(string) string { return "" }
	}
	if env.userHomeDir == nil {
		env.userHomeDir = os.UserHomeDir
	}

	fs := flag.NewFlagSet(buildinfo.ProductCompanion, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var home, repo, fleetConfig string
	var withOpenCode, openCodeFreeFirst, showVersion, protocolInfo bool
	fs.StringVar(&home, "home", "", "home directory whose client folders are inspected (default: the current user home)")
	fs.StringVar(&repo, "repo", "", "skills checkout to read (default: the embedded assets)")
	fs.StringVar(&fleetConfig, "fleet-config", "", "explicit personal fleet.json path (default: none selected)")
	fs.BoolVar(&withOpenCode, "with-opencode", true, "include the optional OpenCode pack and profiles (default true)")
	fs.BoolVar(&openCodeFreeFirst, "opencode-free-first", false, "preset OpenCode profiles to Free-first models (requires --with-opencode)")
	fs.BoolVar(&showVersion, "version", false, "print version and build identity information")
	fs.BoolVar(&protocolInfo, "protocol-info", false, "print the private, machine-readable protocol and build identity response")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		printCLIHint(stderr)
		printUsage(stderr)
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		printCLIHint(stderr)
		printUsage(stderr)
		return 2
	}
	if showVersion && protocolInfo {
		fmt.Fprintln(stderr, "error: --version and --protocol-info are mutually exclusive")
		return 2
	}
	// Informational queries stay ahead of every terminal gate and of any home
	// resolution, so they work in scripts, pipes and tests.
	if protocolInfo {
		data, err := buildinfo.CurrentProtocolInfo(buildinfo.ProductCompanion).JSON()
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	if showVersion {
		printVersion(stdout)
		return 0
	}
	if openCodeFreeFirst && !withOpenCode {
		fmt.Fprintln(stderr, "error: --opencode-free-first requires --with-opencode")
		return 2
	}
	// An empty or whitespace-only value is unset; the companion never invents
	// or scans for a fleet.json path.
	fleetConfig = strings.TrimSpace(fleetConfig)

	return tuirun.Run(tuirun.Launch{
		Program:           buildinfo.ProductCompanion,
		Home:              home,
		Repo:              repo,
		FleetConfig:       fleetConfig,
		WithOpenCode:      withOpenCode,
		OpenCodeFreeFirst: openCodeFreeFirst,
		RequireOptIn:      false,
		// The live fleet capability is a reviewed build value injected at link
		// time (default preview). No flag, environment or config key can flip
		// it at runtime.
		FleetLive:   buildinfo.FleetLiveEnabled(),
		StdinTTY:    env.stdinTTY,
		StdoutTTY:   env.stdoutTTY,
		Stderr:      stderr,
		Getenv:      env.getenv,
		UserHomeDir: env.userHomeDir,
	})
}

// printCLIHint tells a user who typed CLI-shaped arguments that this binary is
// the terminal UI, not the command-line interface.
func printCLIHint(stderr io.Writer) {
	fmt.Fprintf(stderr, "hint: %s is the terminal UI; run %s <command> for the command-line interface\n",
		buildinfo.ProductCompanion, buildinfo.ProductCLI)
}

// printVersion uses only link-time values. It reads no environment or
// filesystem state.
func printVersion(out io.Writer) {
	identity := buildinfo.CurrentIdentity()
	fmt.Fprintf(out, "%s %s\n", buildinfo.ProductCompanion, identity.Revision)
	fmt.Fprintf(out, "  commit:        %s\n", identity.Commit)
	fmt.Fprintf(out, "  source digest: %s\n", identity.SourceDigest)
	fmt.Fprintf(out, "  distribution:  %s\n", buildinfo.CurrentDistribution())
	fmt.Fprintf(out, "  protocol:      %d\n", buildinfo.ProtocolRevision)
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: asdx [--home PATH] [--repo PATH] [--fleet-config PATH] [--with-opencode[=false]] [--opencode-free-first]
       asdx --version
       asdx --protocol-info

The asdx terminal UI. The asdx-core CLI starts this binary from the same
installation; direct invocation is supported and needs a real input and output
terminal. CLI-shaped arguments belong to asdx-core and fail here.

Options:
  --home PATH            Home directory whose client folders are inspected.
  --repo PATH            Skills checkout to read (default: embedded assets).
  --fleet-config PATH    Explicit personal fleet.json path; no path is scanned
                         or inferred, and omitting it selects no source.
  --with-opencode        Include the optional OpenCode pack and profiles (default true).
  --opencode-free-first  Preset OpenCode profiles to Free-first models
                         (requires --with-opencode).
  --version              Print version and build identity information.
  --protocol-info        Print the private protocol/build identity response.
  -h, --help             Show this help.

Environment:
  NO_COLOR                     Disable styled output.
  AGENTIC_SDD_ASCII=1          Use ASCII borders and symbols.
  AGENTIC_SDD_REDUCED_MOTION=1 Disable animation.
  AGENTIC_SDD_FORCE_INTERACTIVE=1
                               Keep scripted runs on the standard CLI; the TUI
                               never opens an alternate screen.

--version and --protocol-info are non-interactive and inspect no network or
filesystem state. The TUI never falls back to a different write workflow.
`)
}
