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
	fmt.Fprint(out, `Usage: agentic-sdd task brief --run ID --task T01 [--force] [--repo DIR] [--run-root DIR] [--format text|json]
       agentic-sdd task help

Extracts exactly one task section from the run's canonical tasks.md, with its
acceptance references and source digests, into briefs/<task>.md. It fails
closed on a missing, duplicate or ambiguous task and invents no requirement.
`)
}
