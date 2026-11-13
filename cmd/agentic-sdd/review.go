package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"agentic-sdd/internal/execution"
)

// runReview implements the public "review" family. It packages observed Git
// changes and never dispatches a reviewer or grants approval.
func runReview(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printReviewUsage(stderr)
		return 2
	}
	switch args[0] {
	case "package":
		return runReviewPackage(args[1:], stdout, stderr)
	case "help":
		printReviewUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown review command %q\n", args[0])
		printReviewUsage(stderr)
		return 2
	}
}

func runReviewPackage(args []string, stdout, stderr io.Writer) int {
	var common executionCommonFlags
	var runID, base, head string
	var maxFile, maxTotal int64 = execution.DefaultMaxFileBytes, execution.DefaultMaxTotalBytes
	fs := newExecutionFlagSet("review package", &common)
	fs.StringVar(&runID, "run", "", "run ID")
	fs.StringVar(&base, "base", "", "base commit")
	fs.StringVar(&head, "head", "", "head commit (default current HEAD)")
	fs.Int64Var(&maxFile, "max-file-bytes", execution.DefaultMaxFileBytes, "per-file content cap")
	fs.Int64Var(&maxTotal, "max-total-bytes", execution.DefaultMaxTotalBytes, "total content cap")
	if ok, code := parseExecutionFlags(fs, args, printReviewUsage, stdout, stderr); !ok {
		return code
	}
	format, err := parseExecutionFormat(common.format)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(base) == "" {
		fmt.Fprintln(stderr, "error: --run and --base are required")
		return 2
	}
	repo, code := resolveExecutionRepo(&common, stderr)
	if code != 0 {
		return code
	}
	runDir, err := resolveExecutionRunDir(repo, common.runRoot, runID)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	manifest, err := execution.BuildReviewPackage(execution.ReviewPackageRequest{
		RunDir:        runDir,
		RunID:         runID,
		RepoRoot:      repo.Root,
		Base:          base,
		Head:          head,
		MaxFileBytes:  maxFile,
		MaxTotalBytes: maxTotal,
	})
	partial := errors.Is(err, execution.ErrPartial)
	if err != nil && !partial {
		fmt.Fprintln(stderr, "error:", err)
		return executionErrorCode(err, true)
	}
	if format == "json" {
		if err := writeExecutionJSON(stdout, manifest); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	} else {
		printReviewPackage(stdout, manifest)
	}
	if partial || !manifest.Complete {
		return 3
	}
	return 0
}

func printReviewPackage(out io.Writer, manifest execution.ReviewManifest) {
	fmt.Fprintf(out, "package: %s\n", manifest.PackageID)
	fmt.Fprintf(out, "base: %s\n", manifest.Base)
	fmt.Fprintf(out, "head: %s\n", manifest.Head)
	if manifest.Complete {
		fmt.Fprintln(out, "complete: yes")
	} else {
		fmt.Fprintln(out, "complete: no (partial; do not call this a complete review)")
		for _, reason := range manifest.PartialReasons {
			fmt.Fprintf(out, "  reason: %s\n", reason)
		}
	}
	fmt.Fprintf(out, "files: %d; committed: %d; staged: %d; unstaged: %d; untracked: %d; omitted: %d\n",
		len(manifest.Files), manifest.Counts.Committed, manifest.Counts.Staged,
		manifest.Counts.Unstaged, manifest.Counts.Untracked, manifest.Counts.Omitted)
	fmt.Fprintf(out, "patch: %s\n", manifest.PatchPath)
	for _, omission := range manifest.Omissions {
		fmt.Fprintf(out, "omitted: %s (%s)\n", omission.Path, omission.Reason)
	}
	if !manifest.Complete {
		fmt.Fprintln(out, "next: inspect the omissions and reconcile before review")
	}
}

func printReviewUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: asdx-core review package --run ID --base COMMIT [--head COMMIT] [--max-file-bytes N] [--max-total-bytes N] [--repo DIR] [--run-root DIR] [--format text|json]
       asdx-core review help

Packages the actual committed, staged, unstaged and untracked change set with
provenance and explicit omissions. A partial package exits 3 and never claims
a complete review, approval, verification or reviewer isolation.
Legacy note: "agentic-sdd" wraps "asdx-core" during the deprecation window.
Use "asdx-core help review" for the topic summary.
`)
}
