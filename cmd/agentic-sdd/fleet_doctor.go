package main

// Feature 014 B5 / T09 — `fleet doctor` report wiring (local-only).
//
// B3's runFleet (fleet.go) owns the subcommand dispatch and flag grammar;
// this file owns the diagnostic behavior behind it: read the personal fleet
// config through the canonical strict parser, label every model route with
// internal/fleet's offline doctor, and print the report. The CLI exposes
// exactly the frozen flags (--config, --opencode-bin); catalog-dump evidence
// remains an internal seam for tests and later reviewed extensions.
//
// Exit codes per spec R3: 2 usage/validation (flag error, unreadable or
// invalid config), 0 when a report is produced — including a report whose
// labels say unavailable, because that IS the honest diagnostic. No network,
// no model call, no file write, and no real-home path is ever taken unless
// the user passes it explicitly.

import (
	"fmt"
	"io"
	"strings"

	"agentic-sdd/internal/fleet"
)

// runFleetDoctorReport executes the doctor for the already-parsed flags.
func runFleetDoctorReport(configPath, opencodeBin string, stdout, stderr io.Writer) int {
	var models []string
	if configPath != "" {
		got, err := fleet.DoctorModelsFromConfig(configPath)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			envHintDoctor(stderr)
			return 2
		}
		models = got
	}
	rep := fleet.RunDoctor(fleet.DoctorOptions{Models: models, OpenCodeBin: opencodeBin})
	printFleetDoctorReport(stdout, rep)
	return 0
}

// envHintDoctor keeps the validation error actionable without echoing config
// content (NFR-08). It is a plain line so non-TTY output stays deterministic.
func envHintDoctor(stderr io.Writer) {
	fmt.Fprintln(stderr, "note: fleet doctor reads only the --config file you passed; see agentic-sdd help fleet")
}

func printFleetDoctorReport(out io.Writer, rep fleet.DoctorReport) {
	fmt.Fprintln(out, "fleet doctor — offline diagnostics (agentic-sdd performs no network or model calls)")
	if rep.Offline {
		fmt.Fprintln(out, `mode: offline default — no local tool was inspected; every label is "not checked (offline)"`)
	} else {
		fmt.Fprintf(out, "mode: local inspection — %d tool source(s) consulted\n", len(rep.ToolsInspected))
	}
	if len(rep.Entries) == 0 {
		fmt.Fprintln(out, "no model routes to inspect (pass --config PATH to label a personal fleet)")
	}
	for _, e := range rep.Entries {
		fmt.Fprintf(out, "%s  [%s] %s\n", e.Model, e.Label, e.Reason)
	}
	if len(rep.ToolsInspected) > 0 {
		fmt.Fprintf(out, "tools inspected: %s\n", strings.Join(rep.ToolsInspected, ", "))
	}
	for _, f := range rep.Failures {
		fmt.Fprintf(out, "inspection note: %s\n", f)
	}
	fmt.Fprintln(out, "labels report local evidence only; a model route is never proof of entitlement")
}
