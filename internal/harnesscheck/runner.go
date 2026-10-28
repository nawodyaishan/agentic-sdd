package harnesscheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxLogBytes = 10 << 20

type Options struct {
	Workspace, ContractPath, Output, Scope string
	CheckIDs                               []string
	Deadline                               time.Duration
	MaxLogBytes                            int64
}

type Selection struct {
	Scope    string   `json:"scope"`
	CheckIDs []string `json:"check_ids"`
}

// Result is deterministic execution output, NOT a complete observation receipt.
// It cannot claim Ready, approval, freshness, feature completion or model usage.
type Result struct {
	Version        int           `json:"version"`
	Workspace      string        `json:"workspace"`
	ContractSHA256 string        `json:"contract_sha256"`
	Selection      Selection     `json:"selection"`
	StartedAt      time.Time     `json:"started_at"`
	EndedAt        time.Time     `json:"ended_at"`
	Execution      string        `json:"execution"`
	Reason         string        `json:"reason"`
	Checks         []CheckResult `json:"checks"`
}

type CheckResult struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	ExitCode  *int       `json:"exit_code"`
	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	Evidence  *string    `json:"evidence"`
	Truncated bool       `json:"truncated"`
	Reason    string     `json:"reason"`
}

func ExitCode(result Result, err error) int {
	if result.Execution == "incomplete" || result.Execution == "cancelled" || result.Execution == "failed" {
		return 1
	}
	if err != nil || result.Execution == "blocked" {
		return 2
	}
	if result.Execution != "completed" {
		return 1
	}
	return 0
}

func Run(ctx context.Context, o Options) (result Result, runErr error) {
	result = Result{Version: 1, Selection: Selection{Scope: o.Scope, CheckIDs: []string{}}, Checks: []CheckResult{}, StartedAt: time.Now().UTC(), Execution: "blocked"}
	defer func() {
		if result.EndedAt.IsZero() {
			result.EndedAt = time.Now().UTC()
		}
		if runErr != nil {
			result.Reason = runErr.Error()
		}
	}()
	if !processSupported {
		return result, fmt.Errorf("check execution/cancellation not enabled on this host; Linux verified, Darwin native validation pending")
	}
	if o.Deadline <= 0 {
		return result, fmt.Errorf("positive run deadline required")
	}
	if o.MaxLogBytes == 0 {
		o.MaxLogBytes = maxLogBytes
	}
	if o.MaxLogBytes < int64(len(truncationMarker))+1 || o.MaxLogBytes > maxLogBytes {
		return result, fmt.Errorf("log byte limit must be between %d and %d", len(truncationMarker)+1, maxLogBytes)
	}
	workspace, err := canonicalDirectory(o.Workspace)
	if err != nil {
		return result, fmt.Errorf("workspace: %w", err)
	}
	result.Workspace = workspace
	info, err := os.Stat(o.ContractPath)
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return result, fmt.Errorf("contract must be a regular file of at most 1 MiB")
	}
	data, err := os.ReadFile(o.ContractPath)
	if err != nil {
		return result, err
	}
	digest := sha256.Sum256(data)
	result.ContractSHA256 = hex.EncodeToString(digest[:])
	contract, err := Parse(data)
	if err != nil {
		return result, err
	}
	selected, err := Select(contract, o.Scope, o.CheckIDs)
	if err != nil {
		return result, err
	}
	for _, ch := range selected {
		if _, err := checkDirectory(workspace, ch.CWD); err != nil {
			return result, fmt.Errorf("%s cwd: %w", ch.ID, err)
		}
		result.Selection.CheckIDs = append(result.Selection.CheckIDs, ch.ID)
	}
	output, err := prepareOutput(workspace, o.Output)
	if err != nil {
		return result, err
	}
	// Reserve the result destination before any command. Never overwrite a file.
	file, err := os.OpenFile(filepath.Join(output, "result.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return result, err
	}
	identity, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return result, err
	}
	defer func() {
		result.EndedAt = time.Now().UTC()
		if runErr != nil {
			result.Execution, result.Reason = "blocked", runErr.Error()
		}
		if err := verifyEvidence(file.Name(), identity); err != nil {
			runErr = err
			result.Execution, result.Reason = "incomplete", err.Error()
		}
		err := json.NewEncoder(file).Encode(result)
		if err == nil {
			err = file.Sync()
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = verifyEvidence(file.Name(), identity)
		}
		if err != nil {
			runErr = fmt.Errorf("persist result: %w", err)
			result.Execution = "incomplete"
			result.Reason = runErr.Error()
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, o.Deadline)
	defer cancel()
	result.Execution, result.Reason = "completed", "all selected checks passed; no review or authorization implied"
	for _, ch := range selected {
		if result.Execution != "completed" {
			result.Checks = append(result.Checks, CheckResult{ID: ch.ID, Status: "not_run", Reason: "earlier selected check did not pass"})
			continue
		}
		if ctx.Err() != nil {
			status := contextStatus(ctx)
			result.Checks = append(result.Checks, CheckResult{ID: ch.ID, Status: status, Reason: ctx.Err().Error()})
			result.Execution, result.Reason = status, ctx.Err().Error()
			continue
		}
		check := executeCheck(ctx, workspace, output, ch, o.MaxLogBytes)
		result.Checks = append(result.Checks, check)
		if check.Status != "passed" {
			result.Execution, result.Reason = check.Status, ch.ID+": "+check.Reason
		}
	}
	return result, nil
}

func contextStatus(ctx context.Context) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "incomplete"
	}
	return "cancelled"
}

func environment(overrides map[string]string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func executeCheck(ctx context.Context, workspace, output string, ch Check, limit int64) (result CheckResult) {
	result = CheckResult{ID: ch.ID, Status: "blocked", Reason: "check not executed"}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(ch.TimeoutSeconds)*time.Second)
	defer cancel()
	dir, err := checkDirectory(workspace, ch.CWD)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	logPath := filepath.Join(output, filepath.FromSlash(ch.Evidence))
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		result.Reason = err.Error()
		return result
	}
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		result.Reason = err.Error()
		return result
	}
	identity, err := file.Stat()
	if err != nil {
		_ = file.Close()
		result.Reason = err.Error()
		return result
	}
	result.Evidence = &logPath
	log := &boundedLog{file: file, remaining: limit - int64(len(truncationMarker))}
	defer func() {
		if err := log.close(); err != nil {
			result.Status, result.Reason = "incomplete", "cannot persist complete log: "+err.Error()
		}
		if err := verifyEvidence(logPath, identity); err != nil {
			result.Status, result.Reason = "incomplete", err.Error()
		}
		result.Truncated = log.truncated
		if log.truncated && result.Status == "passed" {
			result.Status, result.Reason = "incomplete", "log truncated; full evidence unavailable"
		}
		end := time.Now().UTC()
		result.EndedAt = &end
	}()
	if err := prerequisites(ctx, ch.Prerequisites, dir, log); err != nil {
		result.Reason = "prerequisite: " + err.Error()
		if ctx.Err() != nil {
			result.Status = contextStatus(ctx)
		}
		return result
	}
	// Resolve all executables first so a missing later step cannot cause partial effects.
	commands := make([][]string, len(ch.Steps))
	for i, step := range ch.Steps {
		env := environment(step.Env)
		path, err := executable(step.Argv[0], dir, env)
		if err != nil {
			result.Reason = "missing executable " + step.Argv[0] + ": " + err.Error()
			return result
		}
		commands[i] = append([]string{path}, step.Argv[1:]...)
	}
	var before map[string]moduleFile
	if ch.ID == "tidy-check" {
		before, err = modules(dir)
		if err != nil {
			result.Reason = err.Error()
			return result
		}
	}
	start := time.Now().UTC()
	result.StartedAt = &start
	for i, argv := range commands {
		// Refresh canonical cwd after preceding steps may have changed it.
		currentDir, err := checkDirectory(workspace, ch.CWD)
		if err != nil || currentDir != dir {
			result.Status, result.Reason = "incomplete", "cwd identity changed during check"
			break
		}
		_, _ = fmt.Fprintf(log, "step %d/%d\n", i+1, len(commands))
		var started bool
		result.Status, result.ExitCode, result.Reason, started = process(ctx, argv, environment(ch.Steps[i].Env), dir, log)
		if result.Status != "passed" {
			if i == 0 && !started {
				result.StartedAt = nil
			}
			break
		}
	}
	if ch.ID == "tidy-check" {
		after, err := modules(dir)
		if err != nil || !sameModules(before, after) {
			if result.Status == "passed" {
				result.Status = "failed"
			}
			result.Reason += "; module filenames/content changed (including new untracked go.sum)"
		}
	}
	return result
}

func prerequisites(ctx context.Context, names []string, dir string, log *boundedLog) error {
	env := environment(nil)
	for _, name := range names {
		var argv []string
		switch name {
		case "running Docker daemon":
			argv = []string{"docker", "info", "--format", "{{.ServerVersion}}"}
		case "preloaded golang:1.25-alpine image":
			argv = []string{"docker", "image", "inspect", "golang:1.25-alpine", "--format", "{{.Id}}"}
		default:
			if strings.ContainsAny(name, " \t\n") {
				return fmt.Errorf("unknown capability %q", name)
			}
			if _, err := executable(name, dir, env); err != nil {
				return fmt.Errorf("%s unavailable", name)
			}
			continue
		}
		resolved, err := executable(argv[0], dir, env)
		if err != nil {
			return err
		}
		argv[0] = resolved
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		status, _, reason, _ := process(probeCtx, argv, env, dir, log)
		cancel()
		if status != "passed" {
			return fmt.Errorf("%s: %s", name, reason)
		}
	}
	return nil
}
