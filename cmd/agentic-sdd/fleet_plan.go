package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"agentic-sdd/internal/fleet"
)

func printFleetPlanUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]

Preview changes and generate a deterministic plan.

Flags:
  --config PATH  Path to personal fleet.json configuration (required).
  --output PATH  Write canonical JSON plan to specified file (writes only when flag provided).
  --adopt        Adopt existing unmanaged agent files instead of stopping with conflict.
  --home PATH    User home directory (defaults to os.UserHomeDir).
  --repo PATH    Repository containing skills-files (optional).
`)
}

func runFleetPlan(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("fleet plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var configPath, outputPath, home, repo string
	var adopt bool

	fs.StringVar(&configPath, "config", "", "path to fleet.json configuration")
	fs.StringVar(&outputPath, "output", "", "path to write canonical plan JSON")
	fs.BoolVar(&adopt, "adopt", false, "adopt existing unmanaged files instead of reporting conflict")
	fs.StringVar(&home, "home", "", "user home directory")
	fs.StringVar(&repo, "repo", "", "skills repository path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFleetPlanUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	if configPath == "" {
		fmt.Fprintln(stderr, "error: --config is required")
		printFleetPlanUsage(stderr)
		return 2
	}

	resolvedHome, err := resolveHome(home)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	resolvedHome, err = filepath.Abs(resolvedHome)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	plan, err := fleet.BuildPlan(configPath, repo, resolvedHome, adopt)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, fleet.ErrInvalidConfig) {
			return 2
		}
		return 1
	}

	// Write plan output only with explicit flag
	if outputPath != "" {
		if err := fleet.SavePlan(plan, outputPath); err != nil {
			fmt.Fprintf(stderr, "error: failed to save plan: %v\n", err)
			return 1
		}
	}

	printPlanSummary(stdout, plan, configPath, outputPath)
	if plan.HasConflicts() {
		fmt.Fprintln(stderr, "error: plan contains conflicts; resolve them before applying")
		return 1
	}
	return 0
}

func printPlanSummary(out io.Writer, plan *fleet.Plan, configPath, outputPath string) {
	fmt.Fprintf(out, "Fleet Plan (Plan ID: %s)\n", plan.PlanID)
	fmt.Fprintf(out, "Target:        %s\n", plan.TargetRoot)
	fmt.Fprintf(out, "Configuration: %s (digest: %s)\n", configPath, plan.ConfigDigest)
	fmt.Fprintf(out, "\nProposed Operations (%d):\n", len(plan.ProposedOps))

	creates, updates, removes, noops, conflicts := 0, 0, 0, 0, 0
	for _, op := range plan.ProposedOps {
		switch op.Op {
		case "create":
			creates++
			fmt.Fprintf(out, "  [create] %-24s model: %s (from %s)\n", op.File, op.Model, op.SourceRef)
		case "update":
			updates++
			verb := "update"
			if op.Adopt {
				verb = "adopt"
			}
			fmt.Fprintf(out, "  [%s] %-24s model: %s\n", verb, op.File, op.Model)
		case "remove":
			removes++
			fmt.Fprintf(out, "  [remove] %-24s (disabled in config)\n", op.File)
		case "noop":
			noops++
			if op.Reason != "" {
				fmt.Fprintf(out, "  [noop]   %-24s (%s)\n", op.File, op.Reason)
			} else {
				fmt.Fprintf(out, "  [noop]   %-24s (up to date)\n", op.File)
			}
		case "conflict":
			conflicts++
			fmt.Fprintf(out, "  [conflict] %-22s %s\n", op.File, op.Reason)
		default:
			fmt.Fprintf(out, "  [%s] %-24s\n", op.Op, op.File)
		}
	}

	fmt.Fprintf(out, "\nSummary: %d to create, %d to update, %d to remove, %d up to date, %d conflict(s) — nothing was written.\n",
		creates, updates, removes, noops, conflicts)

	if len(plan.PermNotes) > 0 {
		fmt.Fprintln(out, "\nNotes & Warnings:")
		for _, note := range plan.PermNotes {
			fmt.Fprintf(out, "  * %s\n", note)
		}
	}

	if outputPath != "" {
		fmt.Fprintf(out, "Plan saved to: %s\n", filepath.Clean(outputPath))
	} else {
		fmt.Fprintf(out, "Next: pass --output PATH to save this plan, then run agentic-sdd fleet apply --plan PATH\n")
	}
}
