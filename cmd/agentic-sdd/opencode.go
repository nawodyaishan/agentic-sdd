package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"agentic-sdd/internal/opencodeadapter"
)

func printOpenCodeUsage(out io.Writer) {
	fmt.Fprint(out, `Usage: agentic-sdd opencode candidate --feature SLUG --build-path DIR
       agentic-sdd opencode preflight --config FILE [--observation FILE]
       agentic-sdd opencode config [preview] --workspace DIR --candidate FILE
       agentic-sdd opencode config apply --workspace DIR --candidate FILE --backup-root DIR --expect-original DIGEST_OR_absent --expect-candidate DIGEST
       agentic-sdd opencode config rollback --workspace DIR --backup DIR

Candidate writes inactive JSON to stdout only. Review/save it outside project and
active discovery. Empty --feature produces direct-fix grants without feature docs.
Preflight reads bounded files only, not native processes, services or credentials.
Native observations are caller-supplied, not certified enforcement or billing.
Config defaults to read-only preview. Explicit apply/rollback are live operations
requiring actual separate authorization; identity pins do not create consent.
Backup storage must already exist, be private and separate from active discovery.
Native session execution/evidence bridge belong to later batches and are unavailable.
`)
}

func runOpenCode(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printOpenCodeUsage(stdout)
		return 0
	}
	op := args[0]
	args = args[1:]
	if op != "candidate" && op != "preflight" && op != "config" {
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode operation")
		return 2
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
