package main

import (
	"fmt"
	"io"
	"strings"

	"agentic-sdd/internal/execution"
)

// runTaskCommand implements the public "task" family. Brief extraction is
// deterministic; no model is called and no requirement is invented.
func runTaskCommand(args []string, stdout, stderr io.Writer, stdin io.Reader, env *Env) int {
	if len(args) == 0 {
		printTaskUsage(stderr)
		return 2
	}
	switch args[0] {
	case "brief":
		return runTaskBrief(args[1:], stdout, stderr)
	case "help":
		printTaskUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown task command %q\n", args[0])
		printTaskUsage(stderr)
		return 2
	}
}

func runTaskBrief(args []string, stdout, stderr io.Writer) int {
	var common executionCommonFlags
	var runID, taskID string
	var force bool
	fs := newExecutionFlagSet("task brief", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	fs.StringVar(&taskID, "task", "", "task ID")
	fs.BoolVar(&force, "force", false, "replace an existing brief with a different digest")
	if ok, code := parseExecutionFlags(fs, args, printTaskUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(taskID) == "" {
		fmt.Fprintln(stderr, "error: --run and --task are required")
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
	if _, err := execution.LoadRunIdentity(runDir); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	brief, err := execution.TaskBrief(execution.TaskBriefRequest{
		RunDir: runDir,
		TaskID: taskID,
		Force:  force,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, false)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, brief); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "brief: %s\n", brief.BriefPath)
	fmt.Fprintf(stdout, "task: %s\n", brief.TaskID)
	fmt.Fprintf(stdout, "source: %s lines %d-%d\n", brief.SourcePath, brief.LineStart, brief.LineEnd)
	fmt.Fprintf(stdout, "digest: %s\n", brief.BriefDigest)
	return 0
}

func printTaskUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: asdx-core task brief --run ID --task T01 [--force] [--repo DIR] [--run-root DIR] [--format text|json]
       asdx-core task help

Extracts exactly one task section from the owning run's canonical tasks.md,
with its acceptance references and source digests, into briefs/<task>.md under
the per-run writer lock. It fails closed on a missing, duplicate or ambiguous
task and invents no requirement.

Rules and recovery (Feature 017):
- Owning run required: the target run must exist and its checkpoint must match
  that run ID and source scope (a foreign checkpoint conflicts, exit 2).
- Direct-scope refusal: runs created with --direct do not support task briefs
  and fail closed (exit 2).
- Brief registration and digest recovery: creation records the brief path and
  digest in checkpoint.json while preserving existing task state and check,
  claim and blocker history. Re-issuing task brief registers a byte-identical
  unregistered brief after source/run validation; differing bytes require
  --force (--force never bypasses a changed source scope).
- Observation only: a brief, check or record is an observation, never human
  approval, verification or batch completion.

Legacy note: "agentic-sdd" wraps "asdx-core" during the deprecation window.
Use "asdx-core help task" for the topic summary.
`)
}
