//go:build agentic_sdd_experimental

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

	"agentic-sdd/internal/harnessevidence"
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
       agentic-sdd opencode evidence snapshot --workspace DIR --scope PATH [--scope PATH]
       agentic-sdd opencode evidence validate --receipt FILE --workspace DIR --identity FILE --contract FILE --result FILE --evidence-root DIR [--authorization REF] [--log-digest PATH=DIGEST] [--observation POINTER=JSON] [--paid-authorized]
       agentic-sdd opencode evidence export --receipt FILE --workspace DIR --output FILE [--redactions FILE]
       agentic-sdd opencode evidence receipt --context FILE --result FILE [--observation FILE] [--pending-dir DIR]

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
Evidence commands are offline receipt consumers: snapshot measures a scoped
workspace identity, validate rechecks consistency against caller-supplied current
observations, export redacts from a private file before one external write, and
receipt composes a check result with an optional native observation. They never
supply consent, review reasoning or approval, and receipt writes only an optional
pending sidecar outside active discovery.
`)
}

func runOpenCode(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printOpenCodeUsage(stdout)
		return 0
	}
	op := args[0]
	args = args[1:]
	if op != "candidate" && op != "preflight" && op != "config" && op != "session" && op != "evidence" {
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode operation")
		return 2
	}
	if op == "session" {
		return runSessionCmd(args, stdout, stderr)
	}
	if op == "evidence" {
		return runEvidenceCmd(args, stdout, stderr)
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

// listFlag collects repeatable explicit inputs such as scopes and digests.
type listFlag []string

func (l *listFlag) String() string { return strings.Join(*l, ",") }

func (l *listFlag) Set(value string) error {
	if len(*l) >= 1024 {
		return errors.New("too many repeated values")
	}
	*l = append(*l, value)
	return nil
}

// runEvidenceCmd implements the offline receipt consumer group: snapshot,
// validate, export and receipt projection. Every observation is caller
// supplied; no command supplies consent, review reasoning or approval.
func runEvidenceCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printOpenCodeUsage(stdout)
		return 0
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "snapshot":
		return runEvidenceSnapshot(args, stdout, stderr)
	case "validate":
		return runEvidenceValidate(args, stdout, stderr)
	case "export":
		return runEvidenceExport(args, stdout, stderr)
	case "receipt":
		return runEvidenceReceipt(args, stdout, stderr)
	default:
		fmt.Fprintln(stderr, "error: unknown or unavailable opencode evidence operation")
		return 2
	}
}

func newEvidenceFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("opencode evidence "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func evidenceString(fs *flag.FlagSet, name, usage string) *string {
	return fs.String(name, "", usage)
}

func evidenceGet(fs *flag.FlagSet, name string) string {
	if f := fs.Lookup(name); f != nil {
		return f.Value.String()
	}
	return ""
}

func parseEvidenceFlags(fs *flag.FlagSet, args []string, stdout, stderr io.Writer) bool {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printOpenCodeUsage(stdout)
			return false
		}
		fmt.Fprintln(stderr, "error: invalid opencode evidence flags; use opencode --help")
		return false
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "error: unexpected opencode evidence argument")
		return false
	}
	return true
}

func runEvidenceSnapshot(args []string, stdout, stderr io.Writer) int {
	fs := newEvidenceFlags("snapshot")
	workspace := evidenceString(fs, "workspace", "absolute Git workspace root")
	var scopes listFlag
	fs.Var(&scopes, "scope", "literal repository-relative scope; repeatable")
	if !parseEvidenceFlags(fs, args, stdout, stderr) {
		return 2
	}
	identity, err := harnessevidence.Snapshot(*workspace, scopes)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	return emitOpenCode(stdout, stderr, struct {
		Execution string                        `json:"execution"`
		Scope     []string                      `json:"scope"`
		Identity  harnessevidence.ScopeIdentity `json:"identity"`
	}{"observed", scopes, identity}, nil)
}

func runEvidenceValidate(args []string, stdout, stderr io.Writer) int {
	fs := newEvidenceFlags("validate")
	receiptPath := evidenceString(fs, "receipt", "receipt JSON file")
	workspace := evidenceString(fs, "workspace", "absolute current workspace root")
	identityPath := evidenceString(fs, "identity", "current identity JSON file")
	contractPath := evidenceString(fs, "contract", "trusted check contract file")
	resultPath := evidenceString(fs, "result", "independent runner result file")
	evidenceRoot := evidenceString(fs, "evidence-root", "private evidence root outside the workspace")
	authorization := evidenceString(fs, "authorization", "independently reconciled current authorization reference")
	paidAuthorized := fs.Bool("paid-authorized", false, "current paid-run authorization is separately established")
	var logDigests listFlag
	fs.Var(&logDigests, "log-digest", "PATH=SHA256 of one observed check log; repeatable")
	var observationFlags listFlag
	fs.Var(&observationFlags, "observation", "POINTER=JSON independently observed native value; repeatable")
	if !parseEvidenceFlags(fs, args, stdout, stderr) {
		return 2
	}
	receipt, err := opencodeadapter.ReadInput(*receiptPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	contract, err := opencodeadapter.ReadInput(*contractPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	result, err := opencodeadapter.ReadInput(*resultPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	identityBytes, err := opencodeadapter.ReadInput(*identityPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	var identity harnessevidence.Identity
	dec := json.NewDecoder(bytes.NewReader(identityBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&identity); err != nil {
		return emitOpenCode(stdout, stderr, nil, fmt.Errorf("invalid identity observation"))
	}
	if err := dec.Decode(new(any)); err == nil {
		return emitOpenCode(stdout, stderr, nil, fmt.Errorf("invalid identity observation"))
	}
	digests := map[string]string{}
	for _, entry := range logDigests {
		index := strings.LastIndex(entry, "=")
		if index <= 0 || index == len(entry)-1 {
			return emitOpenCode(stdout, stderr, nil, fmt.Errorf("log digest requires PATH=DIGEST"))
		}
		digests[entry[:index]] = entry[index+1:]
	}
	observations := harnessevidence.Observations{}
	for _, entry := range observationFlags {
		index := strings.Index(entry, "=")
		if index <= 0 || !strings.HasPrefix(entry, "/") {
			return emitOpenCode(stdout, stderr, nil, fmt.Errorf("observation requires POINTER=JSON"))
		}
		if len(observations) >= 16 {
			return emitOpenCode(stdout, stderr, nil, fmt.Errorf("too many observations"))
		}
		var value any
		if err := json.Unmarshal([]byte(entry[index+1:]), &value); err != nil {
			return emitOpenCode(stdout, stderr, nil, fmt.Errorf("observation value must be JSON"))
		}
		observations[entry[:index]] = value
	}
	err = harnessevidence.Validate(receipt, harnessevidence.ValidateInput{
		Workspace:        *workspace,
		Identity:         identity,
		AuthorizationRef: *authorization,
		Contract:         contract,
		Result:           result,
		EvidenceRoot:     *evidenceRoot,
		LogDigests:       digests,
		Observations:     observations,
		PaidAuthorized:   *paidAuthorized,
	})
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	return emitOpenCode(stdout, stderr, struct {
		Execution string `json:"execution"`
	}{"consistent"}, nil)
}

// readPrivateRedactions reads newline-separated literal redactions from a
// private file. Redactions never come from CLI arguments, so they cannot leak
// into shell history or process listings.
func readPrivateRedactions(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	data, err := opencodeadapter.ReadInput(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 4096 || len(out) >= 128 {
			return nil, fmt.Errorf("redaction file is not bounded")
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("redaction file is empty")
	}
	return out, nil
}

func runEvidenceExport(args []string, stdout, stderr io.Writer) int {
	fs := newEvidenceFlags("export")
	receiptPath := evidenceString(fs, "receipt", "receipt JSON file")
	workspace := evidenceString(fs, "workspace", "absolute target workspace root")
	output := evidenceString(fs, "output", "new private external receipt path")
	redactionsPath := evidenceString(fs, "redactions", "private file of literal redactions, one per line")
	if !parseEvidenceFlags(fs, args, stdout, stderr) {
		return 2
	}
	receipt, err := opencodeadapter.ReadInput(*receiptPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	redactions, err := readPrivateRedactions(*redactionsPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	if err := harnessevidence.Export(receipt, *output, *workspace, redactions); err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	return emitOpenCode(stdout, stderr, struct {
		Execution string `json:"execution"`
		Output    string `json:"output"`
	}{"exported", *output}, nil)
}

func runEvidenceReceipt(args []string, stdout, stderr io.Writer) int {
	fs := newEvidenceFlags("receipt")
	contextPath := evidenceString(fs, "context", "curated receipt context JSON file")
	resultPath := evidenceString(fs, "result", "independent runner result file")
	observationPath := evidenceString(fs, "observation", "optional owned-transport session observation")
	pendingDir := evidenceString(fs, "pending-dir", "private directory for an optional pending sidecar")
	if !parseEvidenceFlags(fs, args, stdout, stderr) {
		return 2
	}
	contextData, err := opencodeadapter.ReadInput(*contextPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	result, err := opencodeadapter.ReadInput(*resultPath)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	var observation []byte
	if *observationPath != "" {
		observation, err = opencodeadapter.ReadInput(*observationPath)
		if err != nil {
			return emitOpenCode(stdout, stderr, nil, err)
		}
	}
	var context opencodeadapter.ReceiptContext
	dec := json.NewDecoder(bytes.NewReader(contextData))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&context); err != nil {
		return emitOpenCode(stdout, stderr, nil, fmt.Errorf("invalid receipt context"))
	}
	if err := dec.Decode(new(any)); err == nil {
		return emitOpenCode(stdout, stderr, nil, fmt.Errorf("invalid receipt context"))
	}
	receipt, pending, err := opencodeadapter.Project(context, result, observation, *pendingDir)
	if err != nil {
		return emitOpenCode(stdout, stderr, nil, err)
	}
	return emitOpenCode(stdout, stderr, struct {
		Execution string                  `json:"execution"`
		Pending   string                  `json:"pending,omitempty"`
		Receipt   harnessevidence.Receipt `json:"receipt"`
	}{receipt.Execution, pending, receipt}, nil)
}

// runSessionCmd implements the narrow native ACP session command group:// prepare (offline, default), run (strict admission; blocked in development),
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
			Execution string                         `json:"execution"`
			Digest    string                         `json:"digest"`
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
