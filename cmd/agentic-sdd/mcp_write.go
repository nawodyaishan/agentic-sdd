package main

// Feature 015 MCP-B / T37 — `agentic-sdd mcp preview|apply|restore`.
//
// MCP-A added the read-only MCP surface (status, offline doctor, a static
// preview and help). MCP-B turns the same engine into an explicitly authorized
// write surface: a structured plan, a digest-checked apply and an entry-level
// restore. This file owns the parser, the flags and the rendering for that
// write surface; detection, planning, the transaction, the backup and the
// ownership manifest stay in internal/appops and internal/skillsync behind the
// immutable MCP requests both the CLI and the TUI use.
//
// Contract preserved from MCP-A and 014/015:
//   - 2 usage, flag, validation or unsafe-target refusal;
//   - 1 runtime, stale, conflict, rollback or unsupported refusal;
//   - 0 a produced report, a real no-op or a completed apply/restore.
//
// Nothing here contacts the network, starts an OAuth flow, installs a package
// or invokes a model. A write happens only when the target digest matches the
// one the reviewed plan reported, and only through the engine's own lock,
// revalidation, backup and rollback path.

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

// The write seams mirror the MCP-A seams: production always calls
// internal/appops, and tests replace them to assert how the CLI maps a flag, a
// plan, a result and a failure onto output and exit codes. Their signatures use
// only local and skillsync types, so the CLI contract stays testable without a
// real OpenCode home, a compiled skill tree or a process start.
var (
	mcpPlanPreview    = mcpPlanPreviewViaEngine
	mcpApplyPlan      = mcpApplyViaEngine
	mcpRestoreList    = mcpRestoreListViaEngine
	mcpRestorePreview = mcpRestorePreviewViaEngine
	mcpRestoreApply   = mcpRestoreViaEngine

	// mcpPromptIn feeds the default-No restore confirmation. runMCP receives no
	// stdin handle, so this seam keeps the prompt testable while production
	// reads the real terminal, exactly like the existing skills restore path.
	mcpPromptIn io.Reader = os.Stdin
)

// mcpBackupSummary is the CLI view of one managed MCP backup. It is local so
// tests can describe a backup list without constructing appops values.
type mcpBackupSummary struct {
	ID        string
	Path      string
	CreatedAt string
	Servers   []string
}

// --- engine adapters ------------------------------------------------------

func mcpPlanPreviewViaEngine(home, configPath string, selected []string, auth map[string]string) (*skillsync.MCPPlan, string, error) {
	req, err := appops.NewMCPPlanRequest(home, configPath, selected, auth)
	if err != nil {
		return nil, "", err
	}
	res, err := appops.MCPPlanPreview(req)
	if err != nil {
		return nil, "", err
	}
	return res.Plan, res.Token, nil
}

func mcpApplyViaEngine(home, configPath string, plan *skillsync.MCPPlan, token, expectDigest, openCodeBin string) (*skillsync.MCPApplyResult, error) {
	req, err := appops.NewMCPApplyRequest(home, configPath, plan, token, expectDigest, openCodeBin)
	if err != nil {
		return nil, err
	}
	res, err := appops.MCPApply(req)
	if err != nil {
		return nil, err
	}
	return res.Result, nil
}

func mcpRestoreListViaEngine(home string) ([]mcpBackupSummary, error) {
	req, err := appops.NewMCPRestoreListRequest(home)
	if err != nil {
		return nil, err
	}
	infos, err := appops.MCPRestoreList(req)
	if err != nil {
		return nil, err
	}
	out := make([]mcpBackupSummary, 0, len(infos))
	for _, info := range infos {
		out = append(out, mcpBackupSummary{
			ID:        info.ID,
			Path:      info.Path,
			CreatedAt: info.CreatedAt,
			Servers:   append([]string(nil), info.Servers...),
		})
	}
	return out, nil
}

func mcpRestorePreviewViaEngine(home, configPath, backupID string) (*skillsync.MCPRestorePreview, string, error) {
	req, err := appops.NewMCPRestorePreviewRequest(home, configPath, backupID)
	if err != nil {
		return nil, "", err
	}
	res, err := appops.MCPRestorePreview(req)
	if err != nil {
		return nil, "", err
	}
	return res.Preview, res.Token, nil
}

func mcpRestoreViaEngine(home, configPath, backupID, token, expectDigest string) (*skillsync.MCPApplyResult, error) {
	req, err := appops.NewMCPRestoreRequest(home, configPath, backupID, token, expectDigest)
	if err != nil {
		return nil, err
	}
	res, err := appops.MCPRestore(req)
	if err != nil {
		return nil, err
	}
	return res.Result, nil
}

// --- flags ----------------------------------------------------------------

// mcpAuthChoices collects repeatable --auth server=mode selections. The set of
// accepted choices is fixed here so a typo is a usage error instead of a
// silently ignored preference, and so no recipe is ever chosen implicitly.
var mcpAuthModes = map[string][]string{
	"context7": {"oauth", "api-key"},
	"exa":      {"keyless", "oauth", "api-key"},
}

type mcpAuthChoices struct {
	values map[string]string
	order  []string
}

func (a *mcpAuthChoices) String() string {
	if a == nil || len(a.values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(a.order))
	for _, name := range a.order {
		parts = append(parts, name+"="+a.values[name])
	}
	return strings.Join(parts, ",")
}

func (a *mcpAuthChoices) Set(raw string) error {
	name, mode, ok := strings.Cut(raw, "=")
	name, mode = strings.TrimSpace(name), strings.TrimSpace(mode)
	if !ok || name == "" || mode == "" {
		return fmt.Errorf("--auth wants server=mode, for example --auth context7=oauth")
	}
	allowed, known := mcpAuthModes[name]
	if !known {
		return fmt.Errorf("--auth does not apply to server %q; authentication choices exist for context7 and exa", name)
	}
	if !mcpContains(allowed, mode) {
		return fmt.Errorf("--auth %s wants one of %s", name, strings.Join(allowed, ", "))
	}
	if a.values == nil {
		a.values = map[string]string{}
	}
	if _, seen := a.values[name]; !seen {
		a.order = append(a.order, name)
	}
	a.values[name] = mode
	return nil
}

// Map returns a copy so the engine cannot mutate the parsed flag state.
func (a *mcpAuthChoices) Map() map[string]string {
	if a == nil || len(a.values) == 0 {
		return nil
	}
	out := make(map[string]string, len(a.values))
	for name, mode := range a.values {
		out[name] = mode
	}
	return out
}

// mcpWriteFlags holds the flags of the MCP-B subcommands: the shared MCP flags
// without --project, which only the read-only status/doctor/preview need.
type mcpWriteFlags struct {
	mcpFlags
	servers      string
	auth         mcpAuthChoices
	expectConfig string
	opencodeBin  string
}

func newMCPWriteFlagSet(name string, common *mcpFlags) *flag.FlagSet {
	fs := flag.NewFlagSet("agentic-sdd mcp "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&common.home, "home", "", "user home directory")
	fs.StringVar(&common.config, "config", "", "OpenCode configuration file to inspect")
	fs.BoolVar(&common.json, "json", false, "emit stable JSON")
	return fs
}

// parseMCPWriteArgs parses one write subcommand: flags, an optional single
// positional backup id and nothing else. Flags and the id may be interleaved,
// matching the existing restore command.
func parseMCPWriteArgs(fs *flag.FlagSet, args []string, usage func(io.Writer), allowID bool, stdout, stderr io.Writer) (id string, code int, ok bool) {
	remaining := args
	for {
		if err := fs.Parse(remaining); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				usage(stdout)
				return "", 0, false
			}
			fmt.Fprintln(stderr, "error:", err)
			fmt.Fprintln(stderr, "note: run \"agentic-sdd mcp help\" for the accepted flags")
			return "", 2, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return id, 0, true
		}
		if !allowID {
			fmt.Fprintf(stderr, "error: unexpected argument %q\n", rest[0])
			return "", 2, false
		}
		if id != "" {
			fmt.Fprintf(stderr, "error: unexpected argument %q\n", rest[0])
			fmt.Fprintln(stderr, "note: restore accepts exactly one backup id")
			return "", 2, false
		}
		id, remaining = rest[0], rest[1:]
	}
}

// splitOptionalServers mirrors the preview rule: an empty value keeps the
// engine's default selection, while a value that only contains separators is a
// usage error rather than a silent "all default servers".
func splitOptionalServers(raw string, stderr io.Writer) ([]string, int) {
	if strings.TrimSpace(raw) == "" {
		return nil, 0
	}
	selected := splitMCPServers(raw)
	if len(selected) == 0 {
		fmt.Fprintln(stderr, "error: --servers requires at least one server name")
		fmt.Fprintln(stderr, "note: names are codegraph, context7 and exa")
		return nil, 2
	}
	return selected, 0
}

// mcpWriteErrorCode maps the MCP-B failure classes onto the documented codes.
// Validation and an unsafe target are usage errors; a stale plan, a conflict, a
// rollback and an unsupported runtime or schema are runtime refusals.
func mcpWriteErrorCode(err error) int {
	switch {
	case errors.Is(err, skillsync.ErrMCPUsage),
		errors.Is(err, skillsync.ErrMCPUnsafeTarget):
		return 2
	}
	return 1
}

// mcpWriteFailureNote adds the one next step a person needs after a refusal.
// It never replaces the error line or changes the exit code.
func mcpWriteFailureNote(stderr io.Writer, err error) {
	switch {
	case errors.Is(err, appops.ErrNeedsPreview), errors.Is(err, skillsync.ErrMCPStale):
		fmt.Fprintln(stderr, "note: nothing was written; run \"agentic-sdd mcp preview\" again and review the new digest")
	case errors.Is(err, skillsync.ErrMCPConflict):
		fmt.Fprintln(stderr, "note: nothing was overwritten; v1 has no replace or adopt option")
	case errors.Is(err, skillsync.ErrMCPUnsafeTarget):
		fmt.Fprintln(stderr, "note: nothing was read or written outside the intended target")
	case errors.Is(err, skillsync.ErrMCPUnsupported):
		fmt.Fprintln(stderr, "note: nothing was written; this build refuses edits it cannot prove safe")
	case errors.Is(err, skillsync.ErrMCPUsage):
		fmt.Fprintln(stderr, "note: correct the request and run \"agentic-sdd mcp help\" for the accepted values")
	default:
		fmt.Fprintln(stderr, "note: re-run \"agentic-sdd mcp preview\" before retrying")
	}
}

// --- apply ----------------------------------------------------------------

func runMCPApply(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpWriteFlags
	fs := newMCPWriteFlagSet("apply", &flags.mcpFlags)
	fs.StringVar(&flags.servers, "servers", "", "servers to configure")
	fs.Var(&flags.auth, "auth", "authentication choice for one server")
	fs.StringVar(&flags.expectConfig, "expect-config", "", "base configuration digest the reviewed preview printed")
	fs.StringVar(&flags.opencodeBin, "opencode-bin", "", "local OpenCode binary used for the bounded runtime probe")

	_, code, ok := parseMCPWriteArgs(fs, args, printMCPApplyUsage, false, stdout, stderr)
	if !ok {
		return code
	}
	if strings.TrimSpace(flags.expectConfig) == "" {
		fmt.Fprintln(stderr, "error: mcp apply requires --expect-config with the base digest preview printed")
		fmt.Fprintln(stderr, "note: run \"agentic-sdd mcp preview\" first; an empty or missing digest never applies")
		return 2
	}
	selected, code := splitOptionalServers(flags.servers, stderr)
	if code != 0 {
		return code
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	// Apply always rebuilds the plan from the current files instead of trusting
	// a token from an earlier process: the digest check below is what ties the
	// reviewed preview to this write.
	plan, token, err := mcpPlanPreview(home, flags.config, selected, flags.auth.Map())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}
	if plan == nil {
		fmt.Fprintln(stderr, "error: mcp apply could not build a configuration plan")
		return 1
	}
	if code := refuseUnapplicablePlan(stderr, *plan, flags.expectConfig); code != 0 {
		return code
	}

	result, err := mcpApplyPlan(home, flags.config, plan, token, flags.expectConfig, flags.opencodeBin)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}
	if result == nil {
		fmt.Fprintln(stderr, "error: mcp apply returned no result")
		return 1
	}

	view := mcpResultView{
		Command:    "apply",
		Home:       home,
		BaseDigest: plan.BaseDigest,
		PlanDigest: plan.PlanDigest,
		Result:     *result,
	}
	if flags.json {
		return writeMCPResultJSON(stdout, stderr, view)
	}
	printMCPResult(stdout, "MCP apply", view)
	return 0
}

// refuseUnapplicablePlan is the CLI half of the digest contract: it refuses a
// conflicted plan, a missing target and a stale digest before any request that
// could write is built. The engine refuses the same cases again under its lock.
func refuseUnapplicablePlan(stderr io.Writer, plan skillsync.MCPPlan, expect string) int {
	if len(plan.Conflicts) > 0 {
		fmt.Fprintln(stderr, "error: mcp apply refused: the plan reports a conflict")
		for _, conflict := range plan.Conflicts {
			fmt.Fprintf(stderr, "  conflict: %s\n", conflict)
		}
		fmt.Fprintln(stderr, "note: nothing was written; v1 has no replace or adopt option")
		return 1
	}
	if strings.TrimSpace(plan.TargetPath) == "" || strings.TrimSpace(plan.BaseDigest) == "" {
		fmt.Fprintln(stderr, "error: mcp apply refused: no existing global OpenCode configuration target")
		fmt.Fprintln(stderr, "note: v1 never creates a configuration; add the snippet from \"agentic-sdd mcp preview\" manually, then preview again")
		return 1
	}
	if expect != plan.BaseDigest {
		fmt.Fprintf(stderr, "error: stale plan: --expect-config %s does not match the current configuration digest %s\n", expect, plan.BaseDigest)
		fmt.Fprintln(stderr, "note: nothing was written; run \"agentic-sdd mcp preview\" and apply the digest it prints")
		return 1
	}
	return 0
}

// --- restore --------------------------------------------------------------

func runMCPRestore(args []string, stdout, stderr io.Writer, env *Env) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: mcp restore requires a subcommand")
		fmt.Fprintln(stderr, "note: use list, preview or apply")
		printMCPRestoreUsage(stderr)
		return 2
	}

	sub, subArgs := args[0], args[1:]
	switch sub {
	case "help", "--help", "-h":
		printMCPRestoreUsage(stdout)
		return 0
	case "list":
		return runMCPRestoreList(subArgs, stdout, stderr, env)
	case "preview":
		return runMCPRestorePreviewCommand(subArgs, stdout, stderr, env)
	case "apply":
		return runMCPRestoreApply(subArgs, stdout, stderr, env)
	}
	fmt.Fprintf(stderr, "error: unknown mcp restore command %q\n", sub)
	fmt.Fprintln(stderr, "note: use list, preview or apply")
	printMCPRestoreUsage(stderr)
	return 2
}

func runMCPRestoreList(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpWriteFlags
	fs := newMCPWriteFlagSet("restore list", &flags.mcpFlags)

	_, code, ok := parseMCPWriteArgs(fs, args, printMCPRestoreUsage, false, stdout, stderr)
	if !ok {
		return code
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	backups, err := mcpRestoreList(home)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}

	if flags.json {
		return writeMCPRestoreListJSON(stdout, stderr, home, backups)
	}
	printMCPRestoreList(stdout, home, backups)
	return 0
}

func runMCPRestorePreviewCommand(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpWriteFlags
	fs := newMCPWriteFlagSet("restore preview", &flags.mcpFlags)

	id, code, ok := parseMCPWriteArgs(fs, args, printMCPRestoreUsage, true, stdout, stderr)
	if !ok {
		return code
	}
	if id == "" {
		fmt.Fprintln(stderr, "error: mcp restore preview requires a backup id")
		fmt.Fprintln(stderr, "note: list them with \"agentic-sdd mcp restore list\"")
		return 2
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	preview, token, err := mcpRestorePreview(home, flags.config, id)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}
	if preview == nil {
		fmt.Fprintln(stderr, "error: mcp restore preview returned no preview")
		return 1
	}

	if flags.json {
		if code := writeMCPRestorePreviewJSON(stdout, stderr, home, *preview, token); code != 0 {
			return code
		}
	} else {
		printMCPRestorePreview(stdout, home, *preview, token)
	}
	if len(preview.Conflicts) > 0 {
		fmt.Fprintln(stderr, "error: this backup cannot be restored without overwriting later edits")
		fmt.Fprintln(stderr, "note: v1 has no force option; the conflicting entries above stay untouched")
		return 1
	}
	return 0
}

func runMCPRestoreApply(args []string, stdout, stderr io.Writer, env *Env) int {
	var flags mcpWriteFlags
	fs := newMCPWriteFlagSet("restore apply", &flags.mcpFlags)
	fs.StringVar(&flags.expectConfig, "expect-config", "", "base configuration digest the reviewed restore preview printed")

	id, code, ok := parseMCPWriteArgs(fs, args, printMCPRestoreUsage, true, stdout, stderr)
	if !ok {
		return code
	}
	if id == "" {
		fmt.Fprintln(stderr, "error: mcp restore apply requires a backup id")
		fmt.Fprintln(stderr, "note: list them with \"agentic-sdd mcp restore list\"")
		return 2
	}
	if strings.TrimSpace(flags.expectConfig) == "" {
		fmt.Fprintln(stderr, "error: mcp restore apply requires --expect-config with the digest restore preview printed")
		fmt.Fprintln(stderr, "note: run \"agentic-sdd mcp restore preview\" first; a missing digest never restores")
		return 2
	}

	home, code := resolveMCPHome(flags.home, stderr)
	if code != 0 {
		return code
	}

	preview, token, err := mcpRestorePreview(home, flags.config, id)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}
	if preview == nil {
		fmt.Fprintln(stderr, "error: mcp restore preview returned no preview")
		return 1
	}
	if code := refuseUnapplicableRestore(stderr, *preview, flags.expectConfig); code != 0 {
		return code
	}

	if !flags.json {
		printMCPRestorePreview(stdout, home, *preview, token)
	}
	promptOut := stdout
	if flags.json {
		// Keep a real interactive prompt out of the JSON document.
		promptOut = stderr
	}
	proceed, code := confirmMCPRestore(promptOut, env, *preview)
	if !proceed {
		return code
	}

	result, err := mcpRestoreApply(home, flags.config, id, token, flags.expectConfig)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		mcpWriteFailureNote(stderr, err)
		return mcpWriteErrorCode(err)
	}
	if result == nil {
		fmt.Fprintln(stderr, "error: mcp restore returned no result")
		return 1
	}

	view := mcpResultView{
		Command:    "restore apply",
		Home:       home,
		BaseDigest: preview.BaseDigest,
		Result:     *result,
	}
	if flags.json {
		return writeMCPResultJSON(stdout, stderr, view)
	}
	printMCPResult(stdout, "MCP restore apply", view)
	return 0
}

// refuseUnapplicableRestore refuses the same unsafe states apply refuses, but
// against a restore preview: managed-entry conflicts, a missing target and a
// digest that no longer matches the reviewed preview.
func refuseUnapplicableRestore(stderr io.Writer, preview skillsync.MCPRestorePreview, expect string) int {
	if len(preview.Conflicts) > 0 {
		fmt.Fprintln(stderr, "error: mcp restore apply refused: the preview reports a conflict")
		for _, conflict := range preview.Conflicts {
			fmt.Fprintf(stderr, "  conflict: %s\n", conflict)
		}
		fmt.Fprintln(stderr, "note: nothing was written; later edits to a managed entry are never erased")
		return 1
	}
	if strings.TrimSpace(preview.TargetPath) == "" || strings.TrimSpace(preview.BaseDigest) == "" {
		fmt.Fprintln(stderr, "error: mcp restore apply refused: the recorded target is not an existing OpenCode configuration")
		fmt.Fprintln(stderr, "note: v1 never re-creates a missing configuration; review \"agentic-sdd mcp status\" first")
		return 1
	}
	if expect != preview.BaseDigest {
		fmt.Fprintf(stderr, "error: stale restore: --expect-config %s does not match the current configuration digest %s\n", expect, preview.BaseDigest)
		fmt.Fprintln(stderr, "note: nothing was written; run \"agentic-sdd mcp restore preview\" again and review the new digest")
		return 1
	}
	return 0
}

// confirmMCPRestore is the separate default-No confirmation. A scripted run is
// already gated by an explicit id and digest, so it never prompts and proceeds.
// An interactive run must answer y or yes before anything is written: proceed
// is false for every other answer, including end of input, so a declined or
// exhausted confirmation never reaches the writer.
func confirmMCPRestore(stdout io.Writer, env *Env, preview skillsync.MCPRestorePreview) (proceed bool, code int) {
	if env == nil || !env.Interactive {
		return true, 0
	}
	fmt.Fprintf(stdout, "Restore managed MCP entries from %s into %s? [y/N]: ", mcpText(preview.BackupID), mcpText(preview.TargetPath))
	line, ok := readLine(bufio.NewReader(mcpPromptIn))
	answer := strings.TrimSpace(line)
	if !ok {
		fmt.Fprintln(stdout, "Cancelled. Nothing was written.")
		return false, 0
	}
	if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
		fmt.Fprintln(stdout, "Cancelled. Nothing was written.")
		return false, 0
	}
	env.Trace.Event("confirm", "prompt", "mcp.restore", "decision", "yes")
	return true, 0
}

// --- usage ----------------------------------------------------------------

func printMCPApplyUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp apply --expect-config SHA256 [--opencode-bin PATH]
                             [--servers NAME,...] [--auth SERVER=MODE]
                             [--home PATH] [--config PATH] [--json]

Write the selected managed MCP entries into the existing global OpenCode
configuration. The write proceeds only when the current file digest equals
--expect-config and the bounded local runtime probe succeeds.

Flags:
  --expect-config SHA256  Base digest the reviewed preview printed (required).
  --opencode-bin PATH     Local OpenCode binary for the version/schema probe.
  --servers NAME,...      Servers to configure (default: all default servers).
  --auth SERVER=MODE      context7=oauth|api-key; exa=keyless|oauth|api-key.
  --home PATH             User home directory (defaults to your home).
  --config PATH           Explicit OpenCode configuration file to write.
  --json                  Emit stable JSON instead of text.

A missing --expect-config, a missing target, a conflict or a stale digest
writes nothing, and this build never creates a configuration file. Apply takes
one local process: the runtime probe, which uses --opencode-bin when supplied.
Exit codes: 2 usage or validation, 1 stale/conflict/unsupported/runtime,
0 applied or no-op.
`)
}

func printMCPRestoreUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd mcp restore list [--home PATH] [--json]
       agentic-sdd mcp restore preview ID [--home PATH] [--config PATH] [--json]
       agentic-sdd mcp restore apply ID --expect-config SHA256
                                  [--home PATH] [--config PATH] [--json]

Restore only the managed MCP entries recorded in this tool's ownership manifest.
A later edit to a managed entry is a conflict, never something to erase, and
whole-file restore is not part of v1.

Flags:
  --expect-config SHA256  Digest the reviewed restore preview printed (apply only).
  --home PATH             User home directory (defaults to your home).
  --config PATH           Explicit OpenCode configuration file to inspect.
  --json                  Emit stable JSON instead of text.

An interactive run asks one default-No confirmation before apply. A scripted
run never prompts, but still requires the id and the digest. Exit codes: 2 usage
or validation, 1 stale/conflict/unsupported/runtime, 0 applied or no-op.
`)
}

// --- plan rendering -------------------------------------------------------

// printMCPPlan renders the structured MCP-B preview: the exact target, both
// digests, the token that binds them, and one entry per selected server with
// the exact text an apply would write.
func printMCPPlan(out io.Writer, home string, plan skillsync.MCPPlan, token string) {
	fmt.Fprintln(out, "MCP preview — structured plan (read-only, no network, no writes)")
	fmt.Fprintf(out, "  Home:        %s\n", mcpText(home))
	if strings.TrimSpace(plan.TargetPath) == "" {
		fmt.Fprintln(out, "  Target:      not configured (no existing global OpenCode configuration)")
		fmt.Fprintln(out, "  Next step:   add the managed entries manually; this build never creates a configuration")
	} else {
		fmt.Fprintf(out, "  Target:      %s\n", plan.TargetPath)
	}
	fmt.Fprintf(out, "  Format:      %s\n", mcpText(plan.Format))
	fmt.Fprintf(out, "  Schema:      %s\n", mcpText(plan.Schema))
	fmt.Fprintf(out, "  Base digest: %s\n", mcpText(plan.BaseDigest))
	fmt.Fprintf(out, "  Plan digest: %s\n", mcpText(plan.PlanDigest))
	fmt.Fprintf(out, "  Selected:    %s\n", joinOrNone(plan.Selected))
	if len(plan.AuthChoices) > 0 {
		fmt.Fprintf(out, "  Auth:        %s\n", joinMCPAuth(plan.AuthChoices))
	}
	if plan.RuntimeVersion == "" {
		fmt.Fprintln(out, "  Runtime:     not probed (preview executes no binary)")
	} else {
		fmt.Fprintf(out, "  Runtime:     %s (%s)\n", plan.RuntimeVersion, mcpText(plan.RuntimeEvidence))
	}
	switch {
	case len(plan.Conflicts) > 0:
		fmt.Fprintln(out, "  Result:      refused until the conflicts below are resolved")
	case plan.NoOp:
		fmt.Fprintln(out, "  Result:      no change (the configuration already matches this selection)")
	default:
		fmt.Fprintf(out, "  Result:      change ready; apply with --expect-config %s\n", mcpText(plan.BaseDigest))
	}

	fmt.Fprintf(out, "  Entries (%d):\n", len(plan.Entries))
	if len(plan.Entries) == 0 {
		fmt.Fprintln(out, "    none")
	}
	for _, entry := range plan.Entries {
		fmt.Fprintf(out, "    %-10s %-9s %s\n", mcpText(entry.Name), mcpText(entry.Op), mcpText(entry.Reason))
		printMCPEntryText(out, "current", entry.OldText)
		printMCPEntryText(out, "planned", entry.NewText)
	}
	fmt.Fprintf(out, "  Conflicts:   %s\n", joinOrNone(plan.Conflicts))
	fmt.Fprintf(out, "  Not checked: %s\n", strings.Join(mcpNotChecked(plan.NotChecked), ", "))
	fmt.Fprintf(out, "  Token:       %s\n", mcpText(token))
	fmt.Fprintln(out, "Preview only. No write.")
}

// printMCPEntryText renders one exact entry text. An empty text means the
// entry does not exist on that side, which is stated instead of left blank.
func printMCPEntryText(out io.Writer, label, text string) {
	lines := mcpTextLines(text)
	switch len(lines) {
	case 0:
		fmt.Fprintf(out, "      %s: none\n", label)
	case 1:
		fmt.Fprintf(out, "      %s: %s\n", label, lines[0])
	default:
		fmt.Fprintf(out, "      %s:\n", label)
		for _, line := range lines {
			fmt.Fprintf(out, "        %s\n", line)
		}
	}
}

func mcpTextLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// printMCPRestoreList renders the managed backup list. An empty list is a fact,
// not an error.
func printMCPRestoreList(out io.Writer, home string, backups []mcpBackupSummary) {
	fmt.Fprintln(out, "MCP restore — managed MCP backups (read-only)")
	fmt.Fprintf(out, "  Home:     %s\n", mcpText(home))
	fmt.Fprintf(out, "  Backups (%d):\n", len(backups))
	if len(backups) == 0 {
		fmt.Fprintln(out, "    none (no managed MCP backup is recorded)")
	}
	for _, backup := range backups {
		fmt.Fprintf(out, "    %s  created %s  servers: %s\n",
			mcpText(backup.ID),
			mcpText(backup.CreatedAt),
			joinOrNone(backup.Servers),
		)
		if backup.Path != "" {
			fmt.Fprintf(out, "      path: %s\n", backup.Path)
		}
	}
	fmt.Fprintln(out, "Preview is read-only; apply needs an id and --expect-config.")
}

// printMCPRestorePreview renders the exact current-versus-restore comparison.
func printMCPRestorePreview(out io.Writer, home string, preview skillsync.MCPRestorePreview, token string) {
	fmt.Fprintln(out, "MCP restore preview — entry-level, read-only")
	fmt.Fprintf(out, "  Home:        %s\n", mcpText(home))
	fmt.Fprintf(out, "  Backup:      %s\n", mcpText(preview.BackupID))
	fmt.Fprintf(out, "  Target:      %s\n", mcpText(preview.TargetPath))
	fmt.Fprintf(out, "  Base digest: %s\n", mcpText(preview.BaseDigest))
	switch {
	case len(preview.Conflicts) > 0:
		fmt.Fprintln(out, "  Result:      refused until the conflicts below are resolved")
	case len(preview.Entries) == 0:
		fmt.Fprintln(out, "  Result:      no change (no managed entry differs from the backup)")
	default:
		fmt.Fprintf(out, "  Result:      restore ready; apply with --expect-config %s\n", mcpText(preview.BaseDigest))
	}
	fmt.Fprintf(out, "  Entries (%d):\n", len(preview.Entries))
	if len(preview.Entries) == 0 {
		fmt.Fprintln(out, "    none")
	}
	for _, entry := range preview.Entries {
		fmt.Fprintf(out, "    %-10s %-9s %s\n", mcpText(entry.Name), mcpText(entry.Op), mcpText(entry.Reason))
		printMCPEntryText(out, "current", entry.OldText)
		printMCPEntryText(out, "restore", entry.NewText)
	}
	fmt.Fprintf(out, "  Conflicts:   %s\n", joinOrNone(preview.Conflicts))
	fmt.Fprintf(out, "  Token:       %s\n", mcpText(token))
	fmt.Fprintln(out, "Preview only. No write.")
}

// mcpResultView carries what apply and restore apply share, so one renderer
// shows the real result for both instead of two drifting formats.
type mcpResultView struct {
	Command    string
	Home       string
	BaseDigest string
	PlanDigest string
	Result     skillsync.MCPApplyResult
}

func printMCPResult(out io.Writer, title string, view mcpResultView) {
	result := view.Result
	fmt.Fprintf(out, "%s — digest-checked write\n", title)
	fmt.Fprintf(out, "  Home:        %s\n", mcpText(view.Home))
	fmt.Fprintf(out, "  Target:      %s\n", mcpText(result.TargetPath))
	fmt.Fprintf(out, "  Base digest: %s\n", mcpText(view.BaseDigest))
	if view.PlanDigest != "" {
		fmt.Fprintf(out, "  Plan digest: %s\n", view.PlanDigest)
	}
	if result.NoOp {
		fmt.Fprintln(out, "  Result:      no change (the configuration already matched; no backup was created)")
	} else {
		fmt.Fprintln(out, "  Result:      applied")
	}
	fmt.Fprintf(out, "  Applied:     %s\n", joinOrNone(result.Applied))
	if result.BackupID == "" && result.BackupPath == "" {
		fmt.Fprintln(out, "  Backup:      none")
	} else {
		fmt.Fprintf(out, "  Backup:      %s at %s\n", mcpText(result.BackupID), mcpText(result.BackupPath))
	}
	fmt.Fprintf(out, "  Installed:   %s\n", mcpText(result.InstalledDigest))
	fmt.Fprintf(out, "  Runtime:     %s\n", mcpText(result.RuntimeVersion))
	fmt.Fprintf(out, "  Rollback:    %s\n", mcpRollbackText(result.Rollback))
}

// mcpRollbackText reports rollback evidence, including an incomplete one. It
// never rounds a partial undo up to success.
func mcpRollbackText(info skillsync.RollbackInfo) string {
	switch {
	case !info.Attempted:
		return "not needed"
	case info.Complete:
		return "complete (the original managed entries were restored)"
	case info.Error == "":
		return "INCOMPLETE: the managed entries may be partially written; recover from the backup above"
	default:
		return "INCOMPLETE: " + info.Error + "; recover from the backup above"
	}
}

func joinMCPAuth(choices map[string]string) string {
	names := make([]string, 0, len(choices))
	for name := range choices {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+choices[name])
	}
	return strings.Join(parts, ", ")
}

func mcpContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// --- JSON rendering -------------------------------------------------------

// The MCP-B JSON documents are additive to MCP-A: the preview keeps its
// command/mode/preview_only identity and gains the plan fields, so a script
// written against the read-only build keeps parsing the same document.
type mcpJSONPlanReport struct {
	Command         string             `json:"command"`
	Mode            string             `json:"mode"`
	Home            string             `json:"home"`
	Target          string             `json:"target"`
	Format          string             `json:"format"`
	Schema          string             `json:"schema"`
	BaseDigest      string             `json:"base_digest"`
	PlanDigest      string             `json:"plan_digest"`
	Selected        []string           `json:"selected"`
	Auth            map[string]string  `json:"auth"`
	Entries         []mcpJSONPlanEntry `json:"entries"`
	Conflicts       []string           `json:"conflicts"`
	NoOp            bool               `json:"no_op"`
	RuntimeVersion  string             `json:"runtime_version"`
	RuntimeEvidence string             `json:"runtime_evidence"`
	NotChecked      []string           `json:"not_checked"`
	Token           string             `json:"token"`
	PreviewOnly     bool               `json:"preview_only"`
	Write           string             `json:"write"`
}

type mcpJSONPlanEntry struct {
	Name    string `json:"name"`
	Op      string `json:"op"`
	Reason  string `json:"reason"`
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
	OldHash string `json:"old_hash"`
	NewHash string `json:"new_hash"`
}

type mcpJSONWriteResult struct {
	Command         string          `json:"command"`
	Mode            string          `json:"mode"`
	Home            string          `json:"home"`
	Target          string          `json:"target"`
	BaseDigest      string          `json:"base_digest"`
	PlanDigest      string          `json:"plan_digest"`
	NoOp            bool            `json:"no_op"`
	Applied         []string        `json:"applied"`
	BackupID        string          `json:"backup_id"`
	BackupPath      string          `json:"backup_path"`
	InstalledDigest string          `json:"installed_digest"`
	RuntimeVersion  string          `json:"runtime_version"`
	Rollback        mcpJSONRollback `json:"rollback"`
}

type mcpJSONRollback struct {
	Attempted bool   `json:"attempted"`
	Complete  bool   `json:"complete"`
	Error     string `json:"error"`
}

type mcpJSONRestoreList struct {
	Command string                 `json:"command"`
	Mode    string                 `json:"mode"`
	Home    string                 `json:"home"`
	Count   int                    `json:"count"`
	Backups []mcpJSONRestoreBackup `json:"backups"`
}

type mcpJSONRestoreBackup struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	CreatedAt string   `json:"created_at"`
	Servers   []string `json:"servers"`
}

type mcpJSONRestorePreview struct {
	Command     string             `json:"command"`
	Mode        string             `json:"mode"`
	Home        string             `json:"home"`
	BackupID    string             `json:"backup_id"`
	Target      string             `json:"target"`
	BaseDigest  string             `json:"base_digest"`
	Entries     []mcpJSONPlanEntry `json:"entries"`
	Conflicts   []string           `json:"conflicts"`
	Token       string             `json:"token"`
	PreviewOnly bool               `json:"preview_only"`
	Write       string             `json:"write"`
}

func mcpJSONPlanEntries(entries []skillsync.MCPPlanEntry) []mcpJSONPlanEntry {
	out := make([]mcpJSONPlanEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, mcpJSONPlanEntry{
			Name:    entry.Name,
			Op:      entry.Op,
			Reason:  entry.Reason,
			OldText: entry.OldText,
			NewText: entry.NewText,
			OldHash: entry.OldHash,
			NewHash: entry.NewHash,
		})
	}
	return out
}

func writeMCPPlanJSON(stdout, stderr io.Writer, home string, plan skillsync.MCPPlan, token string) int {
	payload := mcpJSONPlanReport{
		Command:         "preview",
		Mode:            "read-only",
		Home:            home,
		Target:          plan.TargetPath,
		Format:          plan.Format,
		Schema:          plan.Schema,
		BaseDigest:      plan.BaseDigest,
		PlanDigest:      plan.PlanDigest,
		Selected:        append([]string{}, plan.Selected...),
		Auth:            plan.AuthChoices,
		Entries:         mcpJSONPlanEntries(plan.Entries),
		Conflicts:       append([]string{}, plan.Conflicts...),
		NoOp:            plan.NoOp,
		RuntimeVersion:  plan.RuntimeVersion,
		RuntimeEvidence: plan.RuntimeEvidence,
		NotChecked:      mcpNotChecked(plan.NotChecked),
		Token:           token,
		PreviewOnly:     true,
		Write:           "none",
	}
	if payload.Auth == nil {
		payload.Auth = map[string]string{}
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func writeMCPRestoreListJSON(stdout, stderr io.Writer, home string, backups []mcpBackupSummary) int {
	payload := mcpJSONRestoreList{
		Command: "restore list",
		Mode:    "read-only",
		Home:    home,
		Count:   len(backups),
		Backups: make([]mcpJSONRestoreBackup, 0, len(backups)),
	}
	for _, backup := range backups {
		payload.Backups = append(payload.Backups, mcpJSONRestoreBackup{
			ID:        backup.ID,
			Path:      backup.Path,
			CreatedAt: backup.CreatedAt,
			Servers:   append([]string{}, backup.Servers...),
		})
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func writeMCPRestorePreviewJSON(stdout, stderr io.Writer, home string, preview skillsync.MCPRestorePreview, token string) int {
	payload := mcpJSONRestorePreview{
		Command:     "restore preview",
		Mode:        "read-only",
		Home:        home,
		BackupID:    preview.BackupID,
		Target:      preview.TargetPath,
		BaseDigest:  preview.BaseDigest,
		Entries:     mcpJSONPlanEntries(preview.Entries),
		Conflicts:   append([]string{}, preview.Conflicts...),
		Token:       token,
		PreviewOnly: true,
		Write:       "none",
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func writeMCPResultJSON(stdout, stderr io.Writer, view mcpResultView) int {
	result := view.Result
	payload := mcpJSONWriteResult{
		Command:         view.Command,
		Mode:            "write",
		Home:            view.Home,
		Target:          result.TargetPath,
		BaseDigest:      view.BaseDigest,
		PlanDigest:      view.PlanDigest,
		NoOp:            result.NoOp,
		Applied:         append([]string{}, result.Applied...),
		BackupID:        result.BackupID,
		BackupPath:      result.BackupPath,
		InstalledDigest: result.InstalledDigest,
		RuntimeVersion:  result.RuntimeVersion,
		Rollback: mcpJSONRollback{
			Attempted: result.Rollback.Attempted,
			Complete:  result.Rollback.Complete,
			Error:     result.Rollback.Error,
		},
	}
	if err := writeMCPJSON(stdout, payload); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}
