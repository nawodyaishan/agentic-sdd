package main

// Feature 015 MCP-A / T29 — `agentic-sdd mcp` read-only CLI.
//
// MCP-A is the read-only foundation of the MCP amendment (spec §16): `status`,
// offline `doctor`, a read-only static `preview` and `help`. Nothing here
// writes a file, starts a process, opens a socket, runs a model or touches a
// credential. The write surface (`mcp apply`, `mcp restore`) belongs to MCP-B
// and is deliberately unreachable in this build, while `doctor --network` and
// `doctor --connect` are deferred out of MCP-A entirely.
//
// The parser owns dispatch, flags, exit codes and rendering only. Detection,
// state projection and diagnostics stay in internal/skillsync and
// internal/appops, reached through the immutable MCP requests those packages
// expose, so the CLI and the future TUI share one engine path.
//
// Exit codes follow 014/015: 2 usage/flag/unknown subcommand (including a
// deferred or unauthorized command), 1 an unexpected engine or I/O error, and
// 0 whenever a report is produced — including a degraded or not-configured
// one, because that IS the honest answer.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// The engine seams keep the CLI contract (dispatch, flags, exit codes, text and
// JSON shapes) testable without a real OpenCode home, a network call or a
// process start. Production always calls internal/appops; tests replace these
// to assert how the CLI maps a request, a report and an engine failure onto
// output and exit codes.
var (
	mcpNewStatusRequest  = appops.NewMCPStatusRequest
	mcpNewDoctorRequest  = appops.NewMCPDoctorRequest
	mcpNewPreviewRequest = appops.NewMCPPreviewRequest
	mcpStatusReport      = appops.MCPStatus
	mcpDoctorReport      = appops.MCPDoctor
	mcpPreviewReport     = appops.MCPPreview
)

func printMCPUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]
       agentic-sdd mcp doctor [--home PATH] [--config PATH] [--project PATH]
                              [--opencode-bin PATH] [--codegraph-bin PATH] [--json]
       agentic-sdd mcp preview [--servers NAME,...] [--home PATH] [--config PATH] [--json]
       agentic-sdd mcp help

Commands:
  status   Report the OpenCode MCP configuration target and the three default
           servers: CodeGraph (local), Context7 and Exa (remote). Default when
           no subcommand is given.
  doctor   Offline diagnostics for the same configuration and prerequisites.
  preview  Print the read-only static recipe comparison for selected servers.
  help     Show this help.

Scope of this build (MCP-A):
  MCP-A is read-only. "mcp apply", "mcp restore" and every configuration write
  are unavailable because MCP-B (the MCP write surface) is not authorized.
  "doctor --network" and "doctor --connect" are deferred out of MCP-A.
  No network call, OAuth flow, package installation, model call or file write
  is performed by any MCP-A command.

Flags:
  --home PATH           User home directory (defaults to your home).
  --config PATH         Explicit OpenCode configuration file to inspect.
  --project PATH        Project path used for prerequisite checks.
  --servers NAME,...    Servers to compare in preview (default: all defaults).
  --opencode-bin PATH   Local OpenCode binary to look up (doctor only).
  --codegraph-bin PATH  Local CodeGraph binary to look up (doctor only).
  --json                Emit stable, script-compatible JSON instead of text.

Exit Codes:
  0  Success, including a degraded or not-configured report
  1  Unexpected engine or I/O error
  2  Usage error, flag error, unknown subcommand, or an unavailable command
`)
}

func printMCPStatusUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]

Report the OpenCode MCP configuration target and the three default servers
(read-only, offline).

Flags:
  --home PATH     User home directory (defaults to your home).
  --config PATH   Explicit OpenCode configuration file to inspect.
  --project PATH  Project path used for prerequisite checks.
  --json          Emit stable JSON instead of text.
`)
}

func printMCPDoctorUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp doctor [--home PATH] [--config PATH] [--project PATH]
                       [--opencode-bin PATH] [--codegraph-bin PATH] [--json]

Offline diagnostics for the OpenCode MCP configuration and prerequisites.
No binary is executed, no network connection is attempted.

Flags:
  --home PATH           User home directory (defaults to your home).
  --config PATH         Explicit OpenCode configuration file to inspect.
  --project PATH        Project path used for prerequisite checks.
  --opencode-bin PATH   Local OpenCode binary to look up.
  --codegraph-bin PATH  Local CodeGraph binary to look up.
  --json                Emit stable JSON instead of text.

--network and --connect are deferred out of MCP-A and are rejected.
`)
}

func printMCPPreviewUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp preview [--servers NAME,...] [--home PATH] [--config PATH] [--json]

Print the read-only static recipe comparison. Nothing is written and no apply
token exists in this build.

Flags:
  --servers NAME,...  Servers to compare (default: all default servers).
  --home PATH         User home directory (defaults to your home).
  --config PATH       Explicit OpenCode configuration file to inspect.
  --json              Emit stable JSON instead of text.
`)
}

func runMCP(args []string, stdout, stderr io.Writer, env *Env) int {
	if len(args) == 0 {
		return runMCPStatus(nil, stdout, stderr, env)
	}

	sub, subArgs := args[0], args[1:]
	switch sub {
	case "help", "--help", "-h":
		printMCPUsage(stdout)
		return 0
	case "status":
		return runMCPStatus(subArgs, stdout, stderr, env)
	case "doctor":
		return runMCPDoctor(subArgs, stdout, stderr, env)
	case "preview":
		return runMCPPreview(subArgs, stdout, stderr, env)
	case "apply", "restore":
		return rejectMCPWriteCommand(sub, stderr)
	}
	if strings.HasPrefix(sub, "-") {
		// "mcp --home PATH" is still the read-only status command, so a flag in
		// the subcommand position is never an unknown command.
		return runMCPStatus(args, stdout, stderr, env)
	}
	fmt.Fprintf(stderr, "error: unknown mcp command %q\n", sub)
	fmt.Fprintln(stderr, "note: this build implements only status, doctor, preview and help")
	printMCPUsage(stderr)
	return 2
}

// rejectMCPWriteCommand refuses the MCP-B write surface instead of pretending
// the command does not exist, so a person learns why and what is available.
func rejectMCPWriteCommand(sub string, stderr io.Writer) int {
	fmt.Fprintf(stderr, "error: agentic-sdd mcp %s is not available in this build; MCP-B (the MCP write surface) is not authorized\n", sub)
	fmt.Fprintln(stderr, "note: this build implements only status, doctor, preview and help; no MCP configuration is ever written")
	printMCPUsage(stderr)
	return 2
}

// mcpFlags holds the flags shared by every read-only MCP subcommand.
type mcpFlags struct {
	home    string
	config  string
	project string
	json    bool
}

func newMCPFlagSet(name string, common *mcpFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("agentic-sdd mcp "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&common.home, "home", "", "user home directory")
	fs.StringVar(&common.config, "config", "", "OpenCode configuration file to inspect")
	fs.StringVar(&common.project, "project", "", "project path for prerequisite checks")
	fs.BoolVar(&common.json, "json", false, "emit stable JSON")
	return fs
}

// parseMCPFlags centralizes the shared flag-error behavior: --help prints the
// subcommand usage on stdout and succeeds, anything else is a usage error.
func parseMCPFlags(fs *flag.FlagSet, args []string, usage func(io.Writer), stdout, stderr io.Writer) (bool, int) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(stdout)
			return false, 0
		}
		fmt.Fprintln(stderr, "error:", err)
		fmt.Fprintln(stderr, "note: run \"agentic-sdd mcp help\" for the accepted flags")
		return false, 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return false, 2
	}
	return true, 0
}

// resolveMCPHome applies the shared --home/default rule and makes the path
// absolute so text and JSON output never depend on the caller's directory.
func resolveMCPHome(home string, stderr io.Writer) (string, int) {
	resolved, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return "", 1
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return "", 1
	}
	return resolved, 0
}

// mcpErrorCode maps engine/request validation to the documented usage exit
// code while leaving runtime and I/O failures as exit 1.
func mcpErrorCode(err error) int {
	if errors.Is(err, skillsync.ErrMCPUsage) {
		return 2
	}
	return 1
}

func runMCPStatus(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpFlags
	fs := newMCPFlagSet("status", &flags)
	ok, code := parseMCPFlags(fs, args, printMCPStatusUsage, stdout, stderr)
	if !ok {
		return code
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	req, err := mcpNewStatusRequest(home, flags.config, flags.project)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	res, err := mcpStatusReport(req)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return mcpErrorCode(err)
	}
	if res.Status == nil {
		fmt.Fprintln(stderr, "error: mcp status returned no report")
		return 1
	}

	if flags.json {
		return writeMCPStatusJSON(stdout, stderr, *res.Status)
	}
	printMCPStatus(stdout, *res.Status)
	return 0
}

func runMCPDoctor(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpFlags
	var opencodeBin, codegraphBin string
	var network, connect bool
	fs := newMCPFlagSet("doctor", &flags)
	fs.StringVar(&opencodeBin, "opencode-bin", "", "local OpenCode binary to look up")
	fs.StringVar(&codegraphBin, "codegraph-bin", "", "local CodeGraph binary to look up")
	fs.BoolVar(&network, "network", false, "deferred out of MCP-A")
	fs.BoolVar(&connect, "connect", false, "deferred out of MCP-A")

	ok, code := parseMCPFlags(fs, args, printMCPDoctorUsage, stdout, stderr)
	if !ok {
		return code
	}
	// A deferred capability is refused explicitly, including an explicit
	// --network=false, rather than silently behaving like the offline default.
	deferred := ""
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "network":
			deferred = "--network"
		case "connect":
			deferred = "--connect"
		}
	})
	if deferred != "" {
		fmt.Fprintf(stderr, "error: agentic-sdd mcp doctor %s is not available in this build; network and connection checks are deferred out of MCP-A\n", deferred)
		fmt.Fprintln(stderr, "note: ordinary doctor is offline and executes no binary")
		printMCPDoctorUsage(stderr)
		return 2
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	req, err := mcpNewDoctorRequest(home, flags.config, flags.project, opencodeBin, codegraphBin)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	res, err := mcpDoctorReport(req)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return mcpErrorCode(err)
	}
	if res.Status == nil {
		fmt.Fprintln(stderr, "error: mcp doctor returned no report")
		return 1
	}

	if flags.json {
		return writeMCPDoctorJSON(stdout, stderr, *res.Status, res.Checks, res.NotChecked)
	}
	printMCPDoctor(stdout, *res.Status, res.Checks, res.NotChecked)
	return 0
}

func runMCPPreview(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpFlags
	var servers string
	fs := newMCPFlagSet("preview", &flags)
	fs.StringVar(&servers, "servers", "", "comma-separated servers to compare")

	ok, code := parseMCPFlags(fs, args, printMCPPreviewUsage, stdout, stderr)
	if !ok {
		return code
	}

	selected := splitMCPServers(servers)
	if strings.TrimSpace(servers) != "" && len(selected) == 0 {
		fmt.Fprintln(stderr, "error: --servers requires at least one server name")
		fmt.Fprintln(stderr, "note: names are codegraph, context7 and exa")
		return 2
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	req, err := mcpNewPreviewRequest(home, flags.config, selected)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	res, err := mcpPreviewReport(req)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return mcpErrorCode(err)
	}
	if res.Preview == nil {
		fmt.Fprintln(stderr, "error: mcp preview returned no report")
		return 1
	}

	if flags.json {
		return writeMCPPreviewJSON(stdout, stderr, *res.Preview)
	}
	printMCPPreview(stdout, *res.Preview)
	return 0
}

// splitMCPServers normalizes the comma-separated --servers value: trimmed,
// empty entries dropped, order and duplicates preserved for the engine to
// validate rather than hidden by the parser.
func splitMCPServers(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if name := strings.TrimSpace(part); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// --- text rendering -------------------------------------------------------

func printMCPStatus(out io.Writer, st skillsync.MCPStatus) {
	fmt.Fprintln(out, "MCP status — offline, read-only (no network, no writes)")
	fmt.Fprintf(out, "  Home:          %s\n", mcpText(st.Home))
	printMCPTarget(out, st)

	fmt.Fprintf(out, "  Servers (%d):\n", len(st.Servers))
	for _, srv := range st.Servers {
		fmt.Fprintf(out, "    %-10s %-8s %-24s prereq: %-30s auth: %-12s evidence: %s\n",
			mcpText(srv.Name),
			mcpText(srv.Kind),
			mcpText(srv.State),
			mcpServerPrerequisite(srv),
			mcpServerAuth(srv),
			mcpText(srv.Evidence),
		)
		printMCPServerDetails(out, srv, st.ConfigPath)
	}
	if len(st.Servers) == 0 {
		fmt.Fprintln(out, "    none reported")
	}
	if st.CodeGraphBinary != "" || st.CodeGraphVersion != "" {
		line := mcpText(st.CodeGraphBinary)
		if st.CodeGraphVersion != "" {
			line += " (" + st.CodeGraphVersion + ")"
		}
		fmt.Fprintf(out, "  CodeGraph CLI: %s\n", line)
	}
	fmt.Fprintf(out, "  Not checked:   %s\n", strings.Join(mcpNotChecked(st.NotChecked), ", "))
}

// printMCPTarget renders the configuration identity block shared by status and
// doctor. A missing target is reported as "not configured", never as a path
// the tool would create.
func printMCPTarget(out io.Writer, st skillsync.MCPStatus) {
	if st.Ambiguity != "" {
		fmt.Fprintln(out, "  Configuration: ambiguous (both opencode.json and opencode.jsonc exist)")
		fmt.Fprintf(out, "  Ambiguity:     %s\n", mcpText(st.Ambiguity))
		return
	}
	if st.ConfigPath == "" {
		fmt.Fprintln(out, "  Configuration: not configured (no global OpenCode configuration found)")
		return
	}
	if st.ConfigFormat == "none" && st.Schema == "none" {
		fmt.Fprintf(out, "  Configuration: not configured (%s does not exist)\n", mcpText(st.ConfigPath))
		return
	}
	fmt.Fprintf(out, "  Configuration: %s\n", st.ConfigPath)
	fmt.Fprintf(out, "  Format:        %s\n", mcpText(st.ConfigFormat))
	fmt.Fprintf(out, "  Schema:        %s\n", mcpText(st.Schema))
	fmt.Fprintf(out, "  Digest:        %s\n", mcpText(st.ConfigDigest))
	if st.Ambiguity != "" {
		fmt.Fprintf(out, "  Ambiguity:     %s\n", st.Ambiguity)
	}
	if st.LegacyReason != "" {
		fmt.Fprintf(out, "  Legacy:        %s\n", st.LegacyReason)
	}
}

func printMCPServerDetails(out io.Writer, srv skillsync.MCPServerStatus, target string) {
	if srv.Purpose != "" {
		fmt.Fprintf(out, "      purpose: %s\n", srv.Purpose)
	}
	if srv.Managed {
		fmt.Fprintln(out, "      ownership: managed by agentic-sdd")
	} else if srv.State == skillsync.MCPStateConfigured || srv.State == skillsync.MCPStateMissingPrerequisite || srv.State == skillsync.MCPStateDisabled {
		fmt.Fprintln(out, "      ownership: not checked (MCP-A does not read the MCP-B ownership manifest)")
	}
	if srv.Disabled {
		fmt.Fprintln(out, "      state detail: disabled entry preserved; not removed")
	}
	if srv.IndexState != "" {
		fmt.Fprintf(out, "      index: %s\n", srv.IndexState)
	}
	if srv.ConfigPath != "" && srv.ConfigPath != target {
		fmt.Fprintf(out, "      config: %s\n", srv.ConfigPath)
	}
	if srv.Conflict != "" {
		fmt.Fprintf(out, "      conflict: %s\n", srv.Conflict)
	}
}

func printMCPDoctor(out io.Writer, st skillsync.MCPStatus, checks []appops.MCPDoctorCheck, notChecked []string) {
	fmt.Fprintln(out, "MCP doctor — offline diagnostics (no network, no writes)")
	fmt.Fprintf(out, "  Home:          %s\n", mcpText(st.Home))
	printMCPTarget(out, st)
	if len(st.Servers) > 0 {
		parts := make([]string, 0, len(st.Servers))
		for _, srv := range st.Servers {
			parts = append(parts, fmt.Sprintf("%s=%s", mcpText(srv.Name), mcpText(srv.State)))
		}
		fmt.Fprintf(out, "  Servers:       %s\n", strings.Join(parts, ", "))
	}
	fmt.Fprintf(out, "  Checks (%d):\n", len(checks))
	if len(checks) == 0 {
		fmt.Fprintln(out, "    none reported")
	}
	for _, check := range checks {
		fmt.Fprintf(out, "    [%-11s] %-22s %s\n",
			mcpText(check.State),
			mcpText(check.Name),
			mcpText(check.Detail),
		)
	}
	fmt.Fprintf(out, "  Not checked:   %s\n", strings.Join(mcpNotChecked(notChecked), ", "))
	fmt.Fprintln(out, "Offline only; remote connectivity, authentication and runtime version were not checked.")
}

func printMCPPreview(out io.Writer, pv skillsync.MCPPreview) {
	fmt.Fprintln(out, "MCP preview — read-only static comparison (no network, no writes)")
	fmt.Fprintf(out, "  Home:        %s\n", mcpText(pv.Home))
	if pv.TargetPath == "" {
		if len(pv.Conflicts) > 0 {
			fmt.Fprintln(out, "  Target:      not selected (resolve the conflicts below; no file was changed)")
		} else {
			fmt.Fprintln(out, "  Target:      not configured (no global OpenCode configuration found)")
		}
	} else if pv.BaseDigest == "" {
		fmt.Fprintf(out, "  Proposed:    %s (not created by this command)\n", pv.TargetPath)
	} else {
		fmt.Fprintf(out, "  Target:      %s\n", pv.TargetPath)
	}
	fmt.Fprintf(out, "  Format:      %s\n", mcpText(pv.Format))
	fmt.Fprintf(out, "  Base digest: %s\n", mcpText(pv.BaseDigest))
	fmt.Fprintf(out, "  Selected:    %s\n", joinOrNone(pv.Selected))
	if pv.NoOp {
		fmt.Fprintln(out, "  Result:      no change (the current configuration already matches)")
	} else {
		fmt.Fprintln(out, "  Result:      change (nothing is written by this command)")
	}

	fmt.Fprintf(out, "  Entries (%d):\n", len(pv.Entries))
	if len(pv.Entries) == 0 {
		fmt.Fprintln(out, "    none")
	}
	for _, entry := range pv.Entries {
		fmt.Fprintf(out, "    %-10s %-9s %s\n", mcpText(entry.Name), mcpText(entry.Op), mcpText(entry.Reason))
		printMCPRecipe(out, entry.Recipe)
	}
	fmt.Fprintf(out, "  Conflicts:   %s\n", joinOrNone(pv.Conflicts))
	fmt.Fprintf(out, "  Not checked: %s\n", strings.Join(mcpNotChecked(pv.NotChecked), ", "))
	fmt.Fprintln(out, "Preview only. No apply in this build.")
}

// printMCPRecipe renders one entry's proposed snippet. A single-line recipe
// stays on the label line; a multi-line snippet is indented as a block so it
// remains copyable.
func printMCPRecipe(out io.Writer, recipe string) {
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(recipe, "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	switch len(lines) {
	case 0:
		return
	case 1:
		fmt.Fprintf(out, "      recipe: %s\n", lines[0])
	default:
		fmt.Fprintln(out, "      recipe:")
		for _, line := range lines {
			fmt.Fprintf(out, "        %s\n", line)
		}
	}
}

// --- JSON rendering -------------------------------------------------------

type mcpJSONConfig struct {
	Path         string `json:"path"`
	Format       string `json:"format"`
	Schema       string `json:"schema"`
	Digest       string `json:"digest"`
	Ambiguity    string `json:"ambiguity,omitempty"`
	LegacyReason string `json:"legacy_reason,omitempty"`
}

type mcpJSONServer struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Purpose         string `json:"purpose"`
	State           string `json:"state"`
	Evidence        string `json:"evidence"`
	Prerequisite    string `json:"prerequisite"`
	PrerequisiteMet bool   `json:"prerequisite_met"`
	Auth            string `json:"auth"`
	Managed         bool   `json:"managed"`
	Disabled        bool   `json:"disabled"`
	ConfigPath      string `json:"config_path,omitempty"`
	Conflict        string `json:"conflict,omitempty"`
	IndexState      string `json:"index_state,omitempty"`
}

type mcpJSONStatus struct {
	Command         string          `json:"command"`
	Mode            string          `json:"mode"`
	Home            string          `json:"home"`
	Config          mcpJSONConfig   `json:"config"`
	Servers         []mcpJSONServer `json:"servers"`
	CodeGraphBinary string          `json:"codegraph_binary"`
	CodeGraphVer    string          `json:"codegraph_version"`
	OpenCodeVersion string          `json:"opencode_version"`
	NotChecked      []string        `json:"not_checked"`
}

type mcpJSONDoctor struct {
	mcpJSONStatus
	Checks  []mcpJSONDoctorCheck `json:"checks"`
	Offline bool                 `json:"offline"`
}

type mcpJSONDoctorCheck struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

type mcpJSONPreview struct {
	Command     string                `json:"command"`
	Mode        string                `json:"mode"`
	Home        string                `json:"home"`
	Target      string                `json:"target"`
	Format      string                `json:"format"`
	BaseDigest  string                `json:"base_digest"`
	Selected    []string              `json:"selected"`
	Entries     []mcpJSONPreviewEntry `json:"entries"`
	Conflicts   []string              `json:"conflicts"`
	NoOp        bool                  `json:"no_op"`
	NotChecked  []string              `json:"not_checked"`
	PreviewOnly bool                  `json:"preview_only"`
}

type mcpJSONPreviewEntry struct {
	Name   string `json:"name"`
	Op     string `json:"op"`
	Reason string `json:"reason"`
	Recipe string `json:"recipe"`
}

func mcpJSONStatusOf(command string, st skillsync.MCPStatus) mcpJSONStatus {
	out := mcpJSONStatus{
		Command: command,
		Mode:    "offline",
		Home:    st.Home,
		Config: mcpJSONConfig{
			Path:         st.ConfigPath,
			Format:       st.ConfigFormat,
			Schema:       st.Schema,
			Digest:       st.ConfigDigest,
			Ambiguity:    st.Ambiguity,
			LegacyReason: st.LegacyReason,
		},
		Servers:         make([]mcpJSONServer, 0, len(st.Servers)),
		CodeGraphBinary: st.CodeGraphBinary,
		CodeGraphVer:    st.CodeGraphVersion,
		OpenCodeVersion: st.OpenCodeVersion,
		NotChecked:      mcpNotChecked(st.NotChecked),
	}
	for _, srv := range st.Servers {
		out.Servers = append(out.Servers, mcpJSONServer{
			Name:            srv.Name,
			Type:            srv.Kind,
			Purpose:         srv.Purpose,
			State:           srv.State,
			Evidence:        srv.Evidence,
			Prerequisite:    srv.Prerequisite,
			PrerequisiteMet: srv.PrerequisiteMet,
			Auth:            srv.AuthMode,
			Managed:         srv.Managed,
			Disabled:        srv.Disabled,
			ConfigPath:      srv.ConfigPath,
			Conflict:        srv.Conflict,
			IndexState:      srv.IndexState,
		})
	}
	return out
}

func writeMCPStatusJSON(stdout, stderr io.Writer, st skillsync.MCPStatus) int {
	payload := mcpJSONStatusOf("status", st)
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func writeMCPDoctorJSON(stdout, stderr io.Writer, st skillsync.MCPStatus, checks []appops.MCPDoctorCheck, notChecked []string) int {
	payload := mcpJSONDoctor{
		mcpJSONStatus: mcpJSONStatusOf("doctor", st),
		Offline:       true,
		Checks:        make([]mcpJSONDoctorCheck, 0, len(checks)),
	}
	if len(notChecked) > 0 {
		payload.NotChecked = append([]string(nil), notChecked...)
	}
	for _, check := range checks {
		payload.Checks = append(payload.Checks, mcpJSONDoctorCheck{
			Name:   check.Name,
			State:  check.State,
			Detail: check.Detail,
		})
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func writeMCPPreviewJSON(stdout, stderr io.Writer, pv skillsync.MCPPreview) int {
	payload := mcpJSONPreview{
		Command:     "preview",
		Mode:        "read-only",
		Home:        pv.Home,
		Target:      pv.TargetPath,
		Format:      pv.Format,
		BaseDigest:  pv.BaseDigest,
		Selected:    append([]string{}, pv.Selected...),
		Entries:     make([]mcpJSONPreviewEntry, 0, len(pv.Entries)),
		Conflicts:   append([]string{}, pv.Conflicts...),
		NoOp:        pv.NoOp,
		NotChecked:  mcpNotChecked(pv.NotChecked),
		PreviewOnly: true,
	}
	for _, entry := range pv.Entries {
		payload.Entries = append(payload.Entries, mcpJSONPreviewEntry{
			Name:   entry.Name,
			Op:     entry.Op,
			Reason: entry.Reason,
			Recipe: entry.Recipe,
		})
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

// writeMCPJSON emits one indented document with lower-case keys. Nothing is
// redacted because no MCP-A report ever carries configuration content,
// credentials or token material.
func writeMCPJSON(out io.Writer, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", data)
	return err
}

// --- small labels ---------------------------------------------------------

// mcpText keeps an empty engine field visible instead of printing a blank
// column or implying the check passed.
func mcpText(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}

func mcpServerPrerequisite(srv skillsync.MCPServerStatus) string {
	if strings.TrimSpace(srv.Prerequisite) == "" {
		return "none"
	}
	if srv.PrerequisiteMet {
		return srv.Prerequisite + " (found)"
	}
	return srv.Prerequisite + " (missing)"
}

func mcpServerAuth(srv skillsync.MCPServerStatus) string {
	switch strings.TrimSpace(srv.AuthMode) {
	case "":
		return "not checked"
	case "oauth":
		return "oauth (not checked)"
	case "api-key":
		return "api-key (env placeholder)"
	case "keyless":
		return "keyless (rate-limited)"
	default:
		return srv.AuthMode
	}
}

func mcpNotChecked(list []string) []string {
	if len(list) == 0 {
		return []string{"remote connectivity", "authentication", "runtime version"}
	}
	return append([]string(nil), list...)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
