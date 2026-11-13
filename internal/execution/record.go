package execution

// Strict ingestion of caller-supplied run records. A record is an untrusted
// claim: it is persisted verbatim with its digest and can never verify a task,
// approve a batch, or authorize a resume.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxRecordSummaryBytes = 8 << 10
	maxRecordPaths        = 1000
	maxRecordChecks       = 100
	maxRecordBlockers     = 100
	maxRecordUnknowns     = 100
	maxRecordTextBytes    = 4096
	maxRecordAgentBytes   = 256
)

var blockerKinds = map[string]bool{
	"permission":    true,
	"billing":       true,
	"data_policy":   true,
	"transport":     true,
	"test":          true,
	"partial_write": true,
	"unknown":       true,
}

// ParseClaim strictly validates one caller-supplied record. taskID is the
// CLI-selected task label; a document taskId must match it when both are
// present. Unknown fields, duplicate keys, oversize input and unsafe paths are
// rejected as validation errors.
func ParseClaim(data []byte, taskID string) (ClaimRecord, error) {
	var claim ClaimRecord
	if len(data) == 0 {
		return claim, fmt.Errorf("%w: record input is empty", ErrUsage)
	}
	if len(data) > MaxRecordBytes {
		return claim, fmt.Errorf("%w: record exceeds %d bytes", ErrUsage, MaxRecordBytes)
	}
	if !utf8.Valid(data) {
		return claim, fmt.Errorf("%w: record must be valid UTF-8", ErrUsage)
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return claim, fmt.Errorf("%w: record must not contain NUL bytes", ErrUsage)
	}
	if err := noDuplicateKeys(data); err != nil {
		return claim, err
	}
	fields, err := strictObject(data, "schemaVersion", "taskId", "result", "summary",
		"changedPaths", "checks", "blockers", "agent", "unknowns")
	if err != nil {
		return claim, err
	}
	version, err := requiredInt(fields["schemaVersion"], "schemaVersion")
	if err != nil {
		return claim, err
	}
	if version != SchemaVersion {
		return claim, fmt.Errorf("%w: unsupported record schema", ErrUsage)
	}
	documentTask, err := optionalString(fields["taskId"], "taskId")
	if err != nil {
		return claim, err
	}
	resolvedTask := strings.TrimSpace(documentTask)
	if strings.TrimSpace(taskID) != "" {
		if resolvedTask != "" && resolvedTask != taskID {
			return claim, fmt.Errorf("%w: record task does not match the selected task", ErrUsage)
		}
		resolvedTask = taskID
	}
	if resolvedTask == "" {
		return claim, fmt.Errorf("%w: task ID is required", ErrUsage)
	}
	if err := validateToken("task ID", resolvedTask); err != nil {
		return claim, err
	}
	resultText, err := optionalString(fields["result"], "result")
	if err != nil {
		return claim, err
	}
	result := ClaimResult(resultText)
	switch result {
	case ClaimPassed, ClaimFailed, ClaimPartial, ClaimBlocked, ClaimNeedsInput:
	default:
		return claim, fmt.Errorf("%w: result must be passed, failed, partial, blocked or needs_input", ErrUsage)
	}
	summary, err := optionalString(fields["summary"], "summary")
	if err != nil {
		return claim, err
	}
	if len(summary) > maxRecordSummaryBytes {
		return claim, fmt.Errorf("%w: summary exceeds %d bytes", ErrUsage, maxRecordSummaryBytes)
	}
	changedPaths, err := stringArray(fields["changedPaths"], maxRecordPaths, "changedPaths")
	if err != nil {
		return claim, err
	}
	for _, changed := range changedPaths {
		if err := ValidateRelativePath(changed); err != nil {
			return claim, err
		}
	}
	checks, err := claimChecks(fields["checks"])
	if err != nil {
		return claim, err
	}
	blockers, err := claimBlockers(fields["blockers"])
	if err != nil {
		return claim, err
	}
	agent, err := claimAgent(fields["agent"])
	if err != nil {
		return claim, err
	}
	unknowns, err := stringArray(fields["unknowns"], maxRecordUnknowns, "unknowns")
	if err != nil {
		return claim, err
	}
	return ClaimRecord{
		SchemaVersion: SchemaVersion,
		TaskID:        resolvedTask,
		Result:        result,
		Summary:       summary,
		ChangedPaths:  changedPaths,
		Checks:        checks,
		Blockers:      blockers,
		Agent:         agent,
		Unknowns:      unknowns,
	}, nil
}

// IngestRecord validates, persists and applies one claim under the run lock.
// The raw document is stored unchanged with its sha256 digest. A passed claim
// yields awaiting-verification; a blocker yields blocked.
func IngestRecord(runDir, taskID string, data []byte) (ClaimRecord, error) {
	identity, err := LoadRunIdentity(runDir)
	if err != nil {
		return ClaimRecord{}, err
	}
	claim, err := ParseClaim(data, taskID)
	if err != nil {
		return ClaimRecord{}, err
	}
	lock, err := acquireRunLock(runDir, "record")
	if err != nil {
		return ClaimRecord{}, err
	}
	fail := func(cause error) (ClaimRecord, error) {
		lock.release()
		return ClaimRecord{}, cause
	}
	checkpoint, err := LoadCheckpoint(runDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fail(err)
		}
		baseline := identity.Baseline
		checkpoint = &Checkpoint{
			SchemaVersion:   SchemaVersion,
			RunID:           identity.RunID,
			UpdatedAt:       time.Now().UTC(),
			State:           StatePrepared,
			LastObservation: &baseline,
		}
	}
	if err := ensureArtifactDir(runDir, "records"); err != nil {
		return fail(err)
	}
	now := time.Now().UTC()
	recordID := ""
	rawPath := ""
	for attempt := 0; attempt < maxArtifactAttempts; attempt++ {
		recordID, err = newArtifactID("rec", now)
		if err != nil {
			return fail(err)
		}
		rawPath = filepath.Join(runDir, "records", recordID+".json")
		if _, statErr := os.Lstat(rawPath); errors.Is(statErr, fs.ErrNotExist) {
			break
		}
		rawPath = ""
	}
	if rawPath == "" {
		return fail(fmt.Errorf("%w: could not allocate a unique record ID", ErrConflict))
	}
	if err := writeFileAtomic(rawPath, data, 0o600); err != nil {
		return fail(err)
	}
	claim.RecordID = recordID
	claim.ReceivedAt = now
	claim.RawPath = rawPath
	claim.RawDigest = DigestBytes(data)
	applyClaim(checkpoint, claim)
	if err := WriteCheckpoint(runDir, checkpoint); err != nil {
		os.Remove(rawPath)
		return fail(err)
	}
	if err := lock.release(); err != nil {
		return ClaimRecord{}, err
	}
	return claim, nil
}

// applyClaim records one claim as task-level awaiting-verification, or blocked
// when the claim reports a blocker. It never produces verified-evidence.
func applyClaim(checkpoint *Checkpoint, claim ClaimRecord) {
	if checkpoint.Tasks == nil {
		checkpoint.Tasks = map[string]TaskState{}
	}
	taskState := StateAwaitingVerification
	if claim.Result == ClaimBlocked || len(claim.Blockers) > 0 {
		taskState = StateBlocked
	}
	task := checkpoint.Tasks[claim.TaskID]
	task.TaskID = claim.TaskID
	task.State = taskState
	task.LastRecordID = claim.RecordID
	task.UpdatedAt = claim.ReceivedAt
	checkpoint.Tasks[claim.TaskID] = task
	checkpoint.Records = append(checkpoint.Records, claim)
	for _, blocker := range claim.Blockers {
		duplicate := false
		for _, existing := range checkpoint.Blockers {
			if existing.Kind == blocker.Kind && existing.Detail == blocker.Detail {
				duplicate = true
				break
			}
		}
		if !duplicate {
			checkpoint.Blockers = append(checkpoint.Blockers, blocker)
		}
	}
	RecomputeRunState(checkpoint)
}

// RecomputeRunState derives the run state from task states and blockers. It
// never downgrades a claim to verified evidence.
func RecomputeRunState(checkpoint *Checkpoint) {
	if len(checkpoint.Blockers) > 0 {
		checkpoint.State = StateBlocked
		return
	}
	state := StatePrepared
	for _, task := range checkpoint.Tasks {
		state = HighestState(state, task.State)
	}
	checkpoint.State = state
}

func claimChecks(raw json.RawMessage) ([]ClaimCheck, error) {
	items, err := objectArray(raw, "checks", maxRecordChecks)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, nil
	}
	checks := make([]ClaimCheck, 0, len(items))
	for index, item := range items {
		fields, err := strictObject(item, "command", "exitCode", "logPath")
		if err != nil {
			return nil, fmt.Errorf("%w: checks[%d] is invalid", ErrUsage, index)
		}
		command, err := requiredText(fields["command"], "checks.command")
		if err != nil {
			return nil, fmt.Errorf("%w: checks[%d] command is invalid", ErrUsage, index)
		}
		check := ClaimCheck{Command: command}
		exitRaw, present := fields["exitCode"]
		if !present {
			return nil, fmt.Errorf("%w: checks[%d] exit code is required", ErrUsage, index)
		}
		exitCode, err := requiredInt(exitRaw, "checks.exitCode")
		if err != nil {
			return nil, fmt.Errorf("%w: checks[%d] exit code is invalid", ErrUsage, index)
		}
		check.ExitCode = exitCode
		logPath, err := optionalString(fields["logPath"], "checks.logPath")
		if err != nil {
			return nil, fmt.Errorf("%w: checks[%d] log path is invalid", ErrUsage, index)
		}
		if logPath != "" {
			if err := ValidateRelativePath(logPath); err != nil {
				return nil, err
			}
		}
		check.LogPath = logPath
		checks = append(checks, check)
	}
	return checks, nil
}

func claimBlockers(raw json.RawMessage) ([]Blocker, error) {
	items, err := objectArray(raw, "blockers", maxRecordBlockers)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, nil
	}
	blockers := make([]Blocker, 0, len(items))
	for index, item := range items {
		fields, err := strictObject(item, "kind", "detail")
		if err != nil {
			return nil, fmt.Errorf("%w: blockers[%d] is invalid", ErrUsage, index)
		}
		kind, err := requiredText(fields["kind"], "blockers.kind")
		if err != nil {
			return nil, fmt.Errorf("%w: blockers[%d] kind is invalid", ErrUsage, index)
		}
		if !blockerKinds[kind] {
			return nil, fmt.Errorf("%w: blockers[%d] kind is not recognized", ErrUsage, index)
		}
		detail, err := requiredText(fields["detail"], "blockers.detail")
		if err != nil {
			return nil, fmt.Errorf("%w: blockers[%d] detail is invalid", ErrUsage, index)
		}
		blockers = append(blockers, Blocker{Kind: kind, Detail: detail})
	}
	return blockers, nil
}

func claimAgent(raw json.RawMessage) (*AgentIdentity, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	fields, err := strictObject(raw, "role", "model")
	if err != nil {
		return nil, fmt.Errorf("%w: agent is invalid", ErrUsage)
	}
	role, err := optionalBoundedString(fields["role"], "agent.role", maxRecordAgentBytes)
	if err != nil {
		return nil, err
	}
	model, err := optionalBoundedString(fields["model"], "agent.model", maxRecordAgentBytes)
	if err != nil {
		return nil, err
	}
	return &AgentIdentity{Role: role, Model: model}, nil
}

// strictObject decodes a JSON object and rejects any field outside allowed.
func strictObject(data []byte, allowed ...string) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("%w: JSON object required", ErrUsage)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON object", ErrUsage)
	}
	if fields == nil {
		return nil, fmt.Errorf("%w: JSON object required", ErrUsage)
	}
	allow := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		allow[name] = true
	}
	for name := range fields {
		if !allow[name] {
			return nil, fmt.Errorf("%w: unknown field is not permitted", ErrUsage)
		}
	}
	return fields, nil
}

func requiredInt(raw json.RawMessage, name string) (int, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return 0, fmt.Errorf("%w: %s is required", ErrUsage, name)
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%w: %s must be an integer", ErrUsage, name)
	}
	return value, nil
}

func optionalString(raw json.RawMessage, name string) (string, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%w: %s must be a string", ErrUsage, name)
	}
	if strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("%w: %s contains a NUL byte", ErrUsage, name)
	}
	return value, nil
}

func optionalBoundedString(raw json.RawMessage, name string, limit int) (string, error) {
	value, err := optionalString(raw, name)
	if err != nil {
		return "", err
	}
	if len(value) > limit {
		return "", fmt.Errorf("%w: %s exceeds %d bytes", ErrUsage, name, limit)
	}
	return value, nil
}

func requiredText(raw json.RawMessage, name string) (string, error) {
	value, err := optionalBoundedString(raw, name, maxRecordTextBytes)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: %s is required", ErrUsage, name)
	}
	return value, nil
}

// stringArray decodes a bounded array of strings. A missing or null field is an
// empty list.
func stringArray(raw json.RawMessage, max int, name string) ([]string, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%w: %s must be an array of strings", ErrUsage, name)
	}
	if len(values) > max {
		return nil, fmt.Errorf("%w: %s exceeds %d entries", ErrUsage, name, max)
	}
	for _, value := range values {
		if len(value) > maxRecordTextBytes {
			return nil, fmt.Errorf("%w: %s entry exceeds %d bytes", ErrUsage, name, maxRecordTextBytes)
		}
		if strings.ContainsRune(value, 0) {
			return nil, fmt.Errorf("%w: %s entry contains a NUL byte", ErrUsage, name)
		}
	}
	return values, nil
}

// objectArray decodes a bounded array of JSON objects.
func objectArray(raw json.RawMessage, name string, max int) ([]json.RawMessage, error) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("%w: %s must be an array of objects", ErrUsage, name)
	}
	if len(items) > max {
		return nil, fmt.Errorf("%w: %s exceeds %d entries", ErrUsage, name, max)
	}
	for _, item := range items {
		trimmed := bytes.TrimSpace(item)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil, fmt.Errorf("%w: %s entries must be objects", ErrUsage, name)
		}
	}
	return items, nil
}

// noDuplicateKeys walks the token stream so duplicate object keys anywhere in
// the document are refused instead of silently taking the last value.
func noDuplicateKeys(data []byte) error {
	type level struct {
		object  bool
		wantKey bool
		keys    map[string]bool
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("%w: invalid JSON", ErrUsage)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("%w: JSON object required", ErrUsage)
	}
	stack := []level{{object: true, wantKey: true, keys: map[string]bool{}}}
	for {
		token, err = decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: invalid JSON", ErrUsage)
		}
		if len(stack) == 0 {
			return fmt.Errorf("%w: trailing data after object", ErrUsage)
		}
		current := &stack[len(stack)-1]
		if current.object && current.wantKey {
			key, ok := token.(string)
			if !ok {
				if delim, isDelim := token.(json.Delim); !isDelim || delim != '}' {
					return fmt.Errorf("%w: invalid JSON", ErrUsage)
				}
			} else {
				if current.keys[key] {
					return fmt.Errorf("%w: duplicate field is not permitted", ErrUsage)
				}
				current.keys[key] = true
				current.wantKey = false
				continue
			}
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 {
					parent := &stack[len(stack)-1]
					if parent.object {
						parent.wantKey = true
					}
				}
			case '{':
				stack = append(stack, level{object: true, wantKey: true, keys: map[string]bool{}})
			case '[':
				stack = append(stack, level{})
			}
		default:
			if current.object {
				current.wantKey = true
			}
		}
	}
	if len(stack) != 0 {
		return fmt.Errorf("%w: unterminated JSON container", ErrUsage)
	}
	return nil
}
