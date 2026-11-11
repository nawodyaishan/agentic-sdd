package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"agentic-sdd/internal/fleet"
	"agentic-sdd/internal/skillsync"
)

func printFleetUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet status --config PATH [--home PATH] [--repo PATH]
       agentic-sdd fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]
       agentic-sdd fleet apply --plan PATH [--home PATH]
       agentic-sdd fleet restore --backup ID [--home PATH]
       agentic-sdd fleet doctor [--opencode-bin PATH] [--config PATH]
       agentic-sdd help fleet

Commands:
  status   Show fleet configuration and installed agent status (read-only, offline).
  plan     Preview changes and generate deterministic plan (read-only unless --output set).
  apply    Safely apply a generated plan under operation lock.
  restore  Restore managed agents from a previous backup (flag form only).
  doctor   Inspect local OpenCode environment and model routes offline.

Exit Codes:
  0  Success
  1  Runtime error, lock held, stale plan, or unmanaged conflict
  2  Usage error, flag error, invalid configuration, or malformed plan
`)
}

func printFleetStatusUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet status --config PATH [--home PATH] [--repo PATH]

Show fleet configuration and installed agent status (read-only, offline).

Flags:
  --config PATH  Path to personal fleet.json configuration (required).
  --home PATH    User home directory (defaults to os.UserHomeDir).
  --repo PATH    Repository containing skills-files (optional).
`)
}

func printFleetApplyUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet apply --plan PATH [--home PATH]

Safely apply a generated plan under operation lock.

Flags:
  --plan PATH  Path to canonical plan JSON file (required).
  --home PATH  User home directory (defaults to os.UserHomeDir).
`)
}

func printFleetRestoreUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet restore --backup ID [--home PATH]

Restore managed agents from a previous backup (flag form only).

Flags:
  --backup ID  ID of the backup to restore (required).
  --home PATH  User home directory (defaults to os.UserHomeDir).
`)
}

func printFleetDoctorUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd fleet doctor [--opencode-bin PATH] [--config PATH]

Inspect local OpenCode environment and model routes offline.

Flags:
  --opencode-bin PATH  Path to opencode binary to inspect.
  --config PATH        Path to personal fleet.json configuration.
`)
}

func runFleet(args []string, stdout, stderr io.Writer, env *Env) int {
	if len(args) == 0 {
		printFleetUsage(stderr)
		return 2
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "help", "--help", "-h":
		printFleetUsage(stdout)
		return 0
	case "status":
		return runFleetStatus(subArgs, stdout, stderr, env)
	case "plan":
		return runFleetPlan(subArgs, stdout, stderr, env)
	case "apply":
		return runFleetApply(subArgs, stdout, stderr, env)
	case "restore":
		return runFleetRestore(subArgs, stdout, stderr, env)
	case "doctor":
		return runFleetDoctor(subArgs, stdout, stderr, env)
	default:
		fmt.Fprintf(stderr, "error: unknown fleet command %q\n", sub)
		printFleetUsage(stderr)
		return 2
	}
}

func runFleetStatus(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("fleet status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var configPath, home, repo string
	fs.StringVar(&configPath, "config", "", "path to fleet.json configuration")
	fs.StringVar(&home, "home", "", "user home directory")
	fs.StringVar(&repo, "repo", "", "skills repository path")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFleetStatusUsage(stdout)
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
		printFleetStatusUsage(stderr)
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

	cfg, _, configDigest, err := fleet.LoadConfigFile(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	absRepo := ""
	if repo != "" {
		absRepo, err = filepath.Abs(repo)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 2
		}
	}
	pack, err := skillsync.OpenCodePackFS(absRepo)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	_, err = fleet.Resolve(pack, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	targetRoot := fleet.TargetRootForHome(resolvedHome)
	live, err := fleet.ReadLiveManifest(resolvedHome)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Fleet Status:\n")
	fmt.Fprintf(stdout, "  Configuration: %s (digest: %s)\n", configPath, configDigest)
	fmt.Fprintf(stdout, "  Target root:   %s\n", targetRoot)
	fmt.Fprintf(stdout, "  Instances (%d):\n", len(cfg.Instances))

	ids := make([]string, 0, len(cfg.Instances))
	for id := range cfg.Instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		inst := cfg.Instances[id]
		fileName := id + ".md"
		targetFile := filepath.Join(targetRoot, fileName)
		info, statErr := os.Lstat(targetFile)
		exists := statErr == nil
		record, recorded := live.RecordFor(targetFile)

		currentDigest := ""
		if exists && info.Mode().IsRegular() {
			if data, err := os.ReadFile(targetFile); err == nil {
				h := sha256.Sum256(data)
				currentDigest = hex.EncodeToString(h[:])
			}
		}

		installedStatus := "absent"
		switch {
		case exists && !info.Mode().IsRegular():
			installedStatus = "installed (unsafe target; refused)"
		case exists && recorded && record.IsManaged():
			if record.DesiredDigest != "" && currentDigest != record.DesiredDigest {
				installedStatus = "installed (managed, drifted)"
			} else {
				installedStatus = "installed (managed)"
			}
		case exists && info.Mode().IsRegular():
			installedStatus = "installed (unmanaged/legacy provenance unknown)"
		}

		priorModel := ""
		if recorded && record.Model != "" {
			priorModel = record.Model
		} else if exists && info.Mode().IsRegular() {
			if content, readErr := os.ReadFile(targetFile); readErr == nil {
				priorModel = fleet.ExtractModelFromContent(content)
			}
		}

		enabledStr := "enabled"
		if !inst.IsEnabled() {
			enabledStr = "disabled"
		}

		fmt.Fprintf(stdout, "    - %s:\n", id)
		fmt.Fprintf(stdout, "        Model:            %s\n", inst.Model)
		fmt.Fprintf(stdout, "        Source:           %s\n", inst.Source)
		fmt.Fprintf(stdout, "        State:            %s\n", enabledStr)
		fmt.Fprintf(stdout, "        Installed:        %s\n", installedStatus)
		fmt.Fprintf(stdout, "        Selected primary: not selected (offline default)\n")
		fmt.Fprintf(stdout, "        Runtime:          not checked (offline)\n")
		if recorded && record.IsManaged() {
			fmt.Fprintf(stdout, "        Provenance:       %s (last applied %s)\n", record.Provenance, shortDigest(record.DesiredDigest))
		}
		if priorModel != "" && inst.Model != "" && priorModel != inst.Model {
			fmt.Fprintf(stdout, "        Warning:          model route changed from %q to %q\n", priorModel, inst.Model)
		}
		if fleet.IsReservedName(id) {
			fmt.Fprintf(stdout, "        Warning:          instance uses reserved SDD name (%s)\n", id)
		}
	}

	return 0
}

func shortDigest(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func runFleetApply(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("fleet apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var planPath, home string
	fs.StringVar(&planPath, "plan", "", "path to canonical plan JSON file")
	fs.StringVar(&home, "home", "", "user home directory")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFleetApplyUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	if planPath == "" {
		fmt.Fprintln(stderr, "error: --plan is required")
		printFleetApplyUsage(stderr)
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

	plan, err := fleet.LoadPlan(planPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, fleet.ErrInvalidPlanStructure) {
			return 2
		}
		return 1
	}

	if _, err := fleet.ApplyPlan(plan, resolvedHome, stdout); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runFleetRestore(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("fleet restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var backupID, home string
	fs.StringVar(&backupID, "backup", "", "backup ID to restore")
	fs.StringVar(&home, "home", "", "user home directory")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFleetRestoreUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: usage: agentic-sdd fleet restore --backup ID [--home PATH]\n")
		return 2
	}

	if backupID == "" {
		fmt.Fprintln(stderr, "error: --backup is required")
		printFleetRestoreUsage(stderr)
		return 2
	}
	if err := skillsync.ValidateBackupID(backupID); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
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

	if _, err := fleet.RestoreBackup(resolvedHome, backupID, stdout); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runFleetDoctor(args []string, stdout, stderr io.Writer, env *Env) int {
	fs := flag.NewFlagSet("fleet doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opencodeBin, configPath string
	fs.StringVar(&opencodeBin, "opencode-bin", "", "path to opencode binary")
	fs.StringVar(&configPath, "config", "", "path to fleet.json configuration")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printFleetDoctorUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", fs.Arg(0))
		return 2
	}

	// B5/T09 wiring: report through internal/fleet's offline doctor.
	return runFleetDoctorReport(configPath, opencodeBin, stdout, stderr)
}
