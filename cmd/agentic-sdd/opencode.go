package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"agentic-sdd/internal/opencodeadapter"
)

func printOpenCodeUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd opencode candidate --feature SLUG --build-path DIR
       agentic-sdd opencode preflight --config FILE [--observation FILE]
       agentic-sdd opencode config [preview] --workspace DIR --candidate FILE
       agentic-sdd opencode config apply --workspace DIR --candidate FILE --backup-root DIR --expect-original DIGEST_OR_absent --expect-candidate DIGEST
       agentic-sdd opencode config rollback --workspace DIR --backup DIR
       agentic-sdd opencode session [prepare] --workspace DIR --executable PATH [--model M] [--mode M]
       agentic-sdd opencode session run --workspace DIR --executable PATH --prompt TEXT [--model M] [--mode M] [--deadline SECS] [--launch FILE]
       agentic-sdd opencode session pending --file FILE

Candidate writes inactive JSON to stdout only. Review/save it outside project and
active discovery. Empty --feature produces direct-fix grants without feature docs.
Preflight reads bounded files only, not native processes, services or credentials.
Native observations are caller-supplied, not certified enforcement or billing.
Config defaults to read-only preview. Explicit apply/rollback are live operations
requiring actual separate authorization; identity pins do not create consent.
Backup storage must already exist, be private and separate from active discovery.
Session prepare is offline and never starts a process. Session run is a narrow
ACP client gated by strict admission: while token/egress/accounting/credential
controls are unverified, it returns blocked and starts no native process. Session
pending only reviews a persisted sanitized summary; it never re-enters or resumes.
`)
}

func runOpenCode(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printOpenCodeUsage(stdout)
		return 0
	}
	op := args[0]
	args = args[1:]
	if op != "candidate" && op != "preflight" && op != "config" && op != "session" {
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode operation")
		return 2
	}
	if op == "session" {
		return runSessionCmd(args, stdout, stderr)
	}
	if op == "config" {
		op = "preview"
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			op = args[0]
			args = args[1:]
		}
	}
	allowed := map[string]map[string]bool{
		"candidate": {"feature": true, "build-path": true},
		"preflight": {"config": true, "observation": true},
		"preview":   {"workspace": true, "candidate": true},
		"apply":     {"workspace": true, "candidate": true, "backup-root": true, "expect-original": true, "expect-candidate": true},
		"rollback":  {"workspace": true, "backup": true},
	}
	if allowed[op] == nil {
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode operation")
		return 2
	}
	fs := flag.NewFlagSet("opencode "+op, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values := map[string]*string{}
	for name := range allowed[op] {
		values[name] = fs.String(name, "", "explicit operation input")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printOpenCodeUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error: invalid opencode flags; use opencode --help")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "error: unexpected opencode argument")
		return 2
	}
	get := func(key string) string {
		if v := values[key]; v != nil {
			return *v
		}
		return ""
	}
	var result any
	var err error
	switch op {
	case "candidate":
		var data []byte
		data, err = opencodeadapter.Candidate(get("feature"), get("build-path"))
		if err == nil {
			_, err = stdout.Write(data)
		}
		if err == nil {
			return 0
		}
	case "preflight":
		var config, native []byte
		config, err = opencodeadapter.ReadInput(get("config"))
		if err == nil && get("observation") != "" {
			native, err = opencodeadapter.ReadInput(get("observation"))
		}
		if err == nil {
			result, err = opencodeadapter.Inspect(config, native)
		}
	case "preview":
		result, err = opencodeadapter.Preview(get("workspace"), get("candidate"))
	case "apply":
		result, err = opencodeadapter.Apply(get("workspace"), get("candidate"), get("backup-root"), get("expect-original"), get("expect-candidate"))
	case "rollback":
		result, err = opencodeadapter.Rollback(get("workspace"), get("backup"))
	}
	if err != nil {
		if tx, ok := result.(opencodeadapter.Transaction); ok && tx.Backup != "" {
			_ = json.NewEncoder(stdout).Encode(struct {
				Execution   string                      `json:"execution"`
				Transaction opencodeadapter.Transaction `json:"transaction"`
			}{"incomplete", tx})
		}
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if err = json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "error: cannot emit opencode observation")
		return 1
	}
	return 0
}

// runSessionCmd implements the narrow native ACP session command group:
// prepare (offline, default), run (strict admission; blocked in development),
// and pending (review a persisted sanitized summary without re-entry).
func runSessionCmd(args []string, stdout, stderr io.Writer) int {
	sub := "prepare"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "prepare", "run", "pending":
	default:
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode session operation")
		return 2
	}
	allowed := map[string]map[string]bool{
		"prepare": {"workspace": true, "executable": true, "model": true, "mode": true},
		"run":     {"workspace": true, "executable": true, "model": true, "mode": true, "prompt": true, "deadline": true, "launch": true},
		"pending": {"file": true},
	}
	fs := flag.NewFlagSet("opencode session "+sub, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values := map[string]*string{}
	for name := range allowed[sub] {
		values[name] = fs.String(name, "", "explicit session input")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printOpenCodeUsage(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "error: invalid opencode session flags; use opencode --help")
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "error: unexpected opencode session argument")
		return 2
	}
	get := func(key string) string {
		if v := values[key]; v != nil {
			return *v
		}
		return ""
	}
	switch sub {
	case "prepare":
		result, err := opencodeadapter.Prepare(get("workspace"), get("executable"), get("model"), get("mode"))
		return emitOpenCode(stdout, stderr, result, err)
	case "run":
		params := opencodeadapter.SessionParams{
			Executable: get("executable"),
			Workspace:  get("workspace"),
			Model:      get("model"),
			Mode:       get("mode"),
			Prompt:     get("prompt"),
		}
		if d := get("deadline"); d != "" {
			secs, err := strconv.ParseInt(d, 10, 64)
			if err != nil || secs <= 0 {
				fmt.Fprintln(stderr, "error: --deadline must be a positive integer of seconds")
				return 2
			}
			params.Deadline = time.Duration(secs) * time.Second
		}
		launch := opencodeadapter.LaunchDeclaration{}
		if f := get("launch"); f != "" {
			data, err := opencodeadapter.ReadInput(f)
			if err != nil {
				fmt.Fprintln(stderr, "error: cannot read launch declaration:", err)
				return 2
			}
			dec := json.NewDecoder(bytes.NewReader(data))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&launch); err != nil {
				fmt.Fprintln(stderr, "error: invalid launch declaration")
				return 2
			}
		}
		result, err := opencodeadapter.Run(launch, params)
		return emitOpenCode(stdout, stderr, result, err)
	case "pending":
		summary, sum, err := opencodeadapter.ReadPending(get("file"))
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
		return emitOpenCode(stdout, stderr, struct {
			Execution string                      `json:"execution"`
			Digest    string                      `json:"digest"`
			Pending   opencodeadapter.PendingSummary `json:"pending"`
		}{"review", sum, summary}, nil)
	}
	return 2
}

// emitOpenCode writes result as JSON on success (exit 0) or prints err to
// stderr (exit 2). A SessionOutcome with Execution "blocked" is a valid
// fail-closed observation, not an error.
func emitOpenCode(stdout, stderr io.Writer, result any, err error) int {
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "error: cannot emit opencode observation")
		return 1
	}
	return 0
}
