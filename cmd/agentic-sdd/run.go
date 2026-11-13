package main

import (
	"fmt"
	"io"
	"strings"

	"agentic-sdd/internal/execution"
	"agentic-sdd/internal/version"
)

// runRun implements the public "run" family. The adapters stay thin: resolve
// flags, call the typed execution service, render text or JSON.
func runRun(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env) int {
	if len(args) == 0 {
		printRunUsage(stderr)
		return 2
	}
	switch args[0] {
	case "init":
		return runRunInit(args[1:], stdout, stderr)
	case "status":
		return runRunStatus(args[1:], stdout, stderr)
	case "record":
		return runRunRecord(args[1:], stdout, stderr, stdin)
	case "clean":
		return runRunClean(args[1:], stdout, stderr)
	case "check":
		return runRunCheck(args[1:], stdout, stderr)
	case "help":
		printRunUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown run command %q\n", args[0])
		printRunUsage(stderr)
		return 2
	}
}

func runRunInit(args []string, stdout, stderr io.Writer) int {
	var common executionCommonFlags
	var feature, batch, label string
	var direct bool
	fs := newExecutionFlagSet("run init", &common)
	fs.StringVar(&feature, "feature", "", "feature directory containing spec.md and tasks.md")
	fs.StringVar(&batch, "batch", "", "authorized batch ID")
	fs.BoolVar(&direct, "direct", false, "create a direct-scope run without feature files")
	fs.StringVar(&label, "label", "", "optional direct-scope label")
	if ok, code := parseExecutionFlags(fs, args, printRunUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	feature = strings.TrimSpace(feature)
	batch = strings.TrimSpace(batch)
	label = strings.TrimSpace(label)
	if direct {
		if feature != "" || batch != "" {
			fmt.Fprintln(stderr, "error: --direct cannot be combined with --feature or --batch")
			return 2
		}
	} else if feature == "" || batch == "" {
		fmt.Fprintln(stderr, "error: --feature and --batch are required unless --direct is used")
		return 2
	}

	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	var scope execution.Scope
	if direct {
		scope, err = execution.DirectScope(repo, label)
	} else {
		scope, err = execution.FeatureScope(repo, feature, batch)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	approvalReference := ""
	if scope.Kind == execution.ScopeFeature {
		approvalReference = scope.SpecPath + " §Approval"
	}
	identity, err := execution.InitRun(execution.InitRequest{
		RunRoot:           common.runRoot,
		Scope:             scope,
		ToolVersion:       version.Version,
		ApprovalReference: approvalReference,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, identity); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	printRunInit(stdout, identity)
	return 0
}

func printRunInit(out io.Writer, identity execution.RunIdentity) {
	fmt.Fprintf(out, "run: %s\n", identity.RunID)
	if identity.Scope.Kind == execution.ScopeDirect {
		label := identity.Scope.Label
		if label == "" {
			label = "(unlabelled)"
		}
		fmt.Fprintf(out, "scope: direct %s\n", label)
	} else {
		fmt.Fprintf(out, "scope: batch %s (%s)\n", identity.Scope.BatchID, identity.Scope.FeatureDir)
	}
	fmt.Fprintln(out, "state: prepared (observation only; not approval)")
	if identity.WorkspaceGitVisible {
		fmt.Fprintln(out, "workspace git visible: yes (explicit --run-root is not ignored)")
	}
	fmt.Fprintf(out, "run dir: %s\n", identity.RunDir)
	if identity.Scope.Kind == execution.ScopeFeature {
		fmt.Fprintf(out, "next: agentic-sdd task brief --run %s --task T01\n", identity.RunID)
	} else {
		fmt.Fprintf(out, "next: agentic-sdd run record --run %s --result FILE\n", identity.RunID)
	}
}

func runRunStatus(args []string, stdout, stderr io.Writer) int {
	var common executionCommonFlags
	var runID string
	fs := newExecutionFlagSet("run status", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	if ok, code := parseExecutionFlags(fs, args, printRunUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	root, err := execution.ResolveRunRoot(repo.Root, common.runRoot)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	status, err := execution.StatusRun(root, runID)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, status); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	printRunStatus(stdout, status)
	return 0
}

func printRunStatus(out io.Writer, status execution.Status) {
	fmt.Fprintf(out, "run: %s\n", status.RunID)
	if status.Scope.Kind == execution.ScopeDirect {
		label := status.Scope.Label
		if label == "" {
			label = "(unlabelled)"
		}
		fmt.Fprintf(out, "scope: direct %s\n", label)
	} else {
		fmt.Fprintf(out, "scope: batch %s (%s)\n", status.Scope.BatchID, status.Scope.FeatureDir)
	}
	fmt.Fprintf(out, "state: %s\n", status.State)
	if len(status.StaleReasons) > 0 {
		fmt.Fprintf(out, "freshness: %s\n", strings.Join(status.StaleReasons, ", "))
	}
	if status.ProgressUpdated {
		fmt.Fprintln(out, "progress: checkbox/status-only update (not stale)")
	}
	if status.LastObservation != nil {
		fmt.Fprintf(out, "observed head: %s\n", status.LastObservation.Head)
	}
	fmt.Fprintf(out, "checks: %d; records: %d; blockers: %d\n", len(status.Checks), len(status.Records), len(status.Blockers))
	for _, blocker := range status.Blockers {
		fmt.Fprintf(out, "blocker: %s — %s\n", blocker.Kind, blocker.Detail)
	}
	fmt.Fprintf(out, "next: %s\n", status.NextPermittedInvestigation)
	fmt.Fprintf(out, "run dir: %s\n", status.RunDir)
}

func runRunRecord(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	var common executionCommonFlags
	var runID, taskID, resultPath string
	fs := newExecutionFlagSet("run record", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	fs.StringVar(&taskID, "task", "", "task ID (default direct for direct-scope runs)")
	fs.StringVar(&resultPath, "result", "", "record JSON file, or - for stdin")
	if ok, code := parseExecutionFlags(fs, args, printRunUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(resultPath) == "" {
		fmt.Fprintln(stderr, "error: --run and --result are required")
		return 2
	}
	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	runDir, err := resolveExecutionRunDir(repo, common.runRoot, runID)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	identity, err := execution.LoadRunIdentity(runDir)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		if identity.Scope.Kind != execution.ScopeDirect {
			fmt.Fprintln(stderr, "error: --task is required for a feature-scope run")
			return 2
		}
		taskID = "direct"
	}
	data, err := readExecutionRecord(resultPath, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	claim, err := execution.IngestRecord(runDir, taskID, data)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, recordJSONResult{
			SchemaVersion: execution.SchemaVersion,
			Kind:          "execution.run.record",
			Record:        claim,
		}); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "record: %s\n", claim.RecordID)
	fmt.Fprintf(stdout, "task: %s\n", claim.TaskID)
	fmt.Fprintf(stdout, "reported: %s (claim; not verified)\n", claim.Result)
	fmt.Fprintf(stdout, "path: %s\n", claim.RawPath)
	return 0
}

func runRunClean(args []string, stdout, stderr io.Writer) int {
	var common executionCommonFlags
	var runID string
	var yes bool
	fs := newExecutionFlagSet("run clean", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	fs.BoolVar(&yes, "yes", false, "delete the run scratch (without it, only preview)")
	if ok, code := parseExecutionFlags(fs, args, printRunUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	root, err := execution.ResolveRunRoot(repo.Root, common.runRoot)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	result, err := execution.CleanRun(root, runID, yes)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, result); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	if result.AlreadyAbsent {
		fmt.Fprintf(stdout, "run %s is already absent\n", result.RunID)
		return 0
	}
	if result.Removed {
		fmt.Fprintf(stdout, "removed run %s: %d files, %d bytes\n", result.RunID, result.FileCount, result.TotalBytes)
		return 0
	}
	fmt.Fprintf(stdout, "preview: run %s has %d files, %d bytes\n", result.RunID, result.FileCount, result.TotalBytes)
	fmt.Fprintf(stdout, "run dir: %s\n", result.RunDir)
	fmt.Fprintln(stdout, "add --yes to remove only this run's scratch")
	return 0
}

func runRunCheck(args []string, stdout, stderr io.Writer) int {
	separator := -1
	for index, arg := range args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		fmt.Fprintln(stderr, "error: run check requires -- before the command vector")
		return 2
	}
	var common executionCommonFlags
	var runID, taskID string
	timeout := execution.DefaultCheckTimeout
	var maxOutput int64 = execution.DefaultMaxOutputBytes
	fs := newExecutionFlagSet("run check", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	fs.StringVar(&taskID, "task", "", "task ID (default direct for direct-scope runs)")
	fs.DurationVar(&timeout, "timeout", execution.DefaultCheckTimeout, "wall-clock limit; 0 disables")
	fs.Int64Var(&maxOutput, "max-output-bytes", execution.DefaultMaxOutputBytes, "per-stream stdout/stderr cap")
	if ok, code := parseExecutionFlags(fs, args[:separator], printRunUsage, stdout, stderr); !ok {
		return code
	}
	command := args[separator+1:]
	if len(command) == 0 {
		fmt.Fprintln(stderr, "error: run check requires a command after --")
		return 2
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if strings.TrimSpace(runID) == "" {
		fmt.Fprintln(stderr, "error: --run is required")
		return 2
	}
	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	runDir, err := resolveExecutionRunDir(repo, common.runRoot, runID)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	identity, err := execution.LoadRunIdentity(runDir)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		if identity.Scope.Kind != execution.ScopeDirect {
			fmt.Fprintln(stderr, "error: --task is required for a feature-scope run")
			return 2
		}
		taskID = "direct"
	}
	record, err := execution.RunCheck(execution.CheckRequest{
		RunDir:         runDir,
		RunID:          runID,
		TaskID:         taskID,
		Argv:           command,
		Timeout:        timeout,
		MaxOutputBytes: maxOutput,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, checkJSONResult{
			SchemaVersion: execution.SchemaVersion,
			Kind:          "execution.run.check",
			Check:         record,
		}); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	} else {
		printRunCheck(stdout, record)
	}
	switch record.Result {
	case execution.CheckPassed:
		return 0
	case execution.CheckTimedOut:
		return 124
	default:
		return 1
	}
}

// checkJSONResult is the versioned public JSON envelope for run check. The
// CheckRecord remains the evidence payload; the wrapper supplies the stable
// schema identity required by the public contract.
type checkJSONResult struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	Check         execution.CheckRecord `json:"check"`
}

// recordJSONResult is the versioned public JSON envelope for run record. The
// ClaimRecord is untrusted reported data.
type recordJSONResult struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	Record        execution.ClaimRecord `json:"record"`
}

func printRunCheck(out io.Writer, record execution.CheckRecord) {
	fmt.Fprintf(out, "check: %s\n", record.CheckID)
	if record.TaskID != "" {
		fmt.Fprintf(out, "task: %s\n", record.TaskID)
	}
	fmt.Fprintf(out, "result: %s\n", record.Result)
	if record.ExitCode != nil {
		fmt.Fprintf(out, "exit: %d\n", *record.ExitCode)
	}
	if record.Signal != "" {
		fmt.Fprintf(out, "signal: %s\n", record.Signal)
	}
	if record.TimedOut {
		fmt.Fprintln(out, "timed out: yes")
	}
	if record.OutputLimit {
		fmt.Fprintln(out, "output limit: yes")
	}
	if record.StdoutPath != "" {
		fmt.Fprintf(out, "stdout log: %s\n", record.StdoutPath)
	}
	if record.StderrPath != "" {
		fmt.Fprintf(out, "stderr log: %s\n", record.StderrPath)
	}
	if len(record.Summary) > 0 {
		fmt.Fprintln(out, "summary (excerpt; full logs above):")
		for _, line := range record.Summary {
			fmt.Fprintf(out, "  %s\n", line)
		}
	}
	if record.SummaryTruncated {
		fmt.Fprintln(out, "summary truncated: yes")
	}
}

func printRunUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd run init --feature DIR --batch ID [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd run init --direct [--label TEXT] [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd run status --run ID [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd run record --run ID [--task T01] --result FILE|- [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd run clean --run ID [--yes] [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd run check --run ID --task T01 [--timeout DURATION] -- COMMAND [ARG...]

Creates and inspects a private execution run under .agentic-sdd/runs/<run-id>/.
A run, checkpoint or check is an observation: it is never approval, batch
state, verification or a sandbox. Use "agentic-sdd help run" for the contract.
`)
}
