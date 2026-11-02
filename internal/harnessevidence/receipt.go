// Package harnessevidence is the production v1 observation receipt consumer:
// strict typed/semantic validation, scoped workspace identity and safe external
// export. It reuses internal/harnesscheck contract/result semantics instead of
// duplicating the runner, adds no JSON Schema engine, makes no Python a
// production dependency, and never writes approval, task or review state.
// The v1 wire format is unchanged; receipt v1 identity keeps sha256-v1 meaning.
package harnessevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// Limits are the declared run bounds. They are assertions, not proof.
type Limits struct {
	DeadlineSeconds  int64    `json:"deadline_seconds"`
	Steps            int64    `json:"steps"`
	TransientRetries int64    `json:"transient_retries"`
	PaidAuthorized   bool     `json:"paid_authorized"`
	SpendUSD         *float64 `json:"spend_usd"`
}

// Identity is the scoped run identity. Every field is independently observed;
// a null field is unknown, never zero or inherited.
type Identity struct {
	BaseRevision *string `json:"base_revision"`
	Diff         *string `json:"diff"`
	Config       *string `json:"config"`
	Skills       *string `json:"skills"`
	Environment  *string `json:"environment"`
	Runtime      *string `json:"runtime"`
	Harness      *string `json:"harness"`
	Contract     *string `json:"contract"`
}

func (a Identity) equal(b Identity) bool {
	for i := range a.fields() {
		if !sameOptionalText(a.fields()[i], b.fields()[i]) {
			return false
		}
	}
	return true
}

func (i Identity) fields() [8]*string {
	return [8]*string{i.BaseRevision, i.Diff, i.Config, i.Skills, i.Environment, i.Runtime, i.Harness, i.Contract}
}

// Model is the observed native model identity.
type Model struct {
	Provider *string `json:"provider"`
	ID       *string `json:"id"`
	Variant  *string `json:"variant"`
}

// Selection is the selected check scope and IDs.
type Selection struct {
	Scope    string   `json:"scope"`
	CheckIDs []string `json:"check_ids"`
}

// CheckEntry is one check outcome as recorded in a receipt. Times stay exact
// strings: a reformatted timestamp is a disagreement, not a match.
type CheckEntry struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	ExitCode  *int64  `json:"exit_code"`
	StartedAt *string `json:"started_at"`
	EndedAt   *string `json:"ended_at"`
	Evidence  *string `json:"evidence"`
	Truncated bool    `json:"truncated"`
	Reason    string  `json:"reason"`
}

// Finding is one carried finding. v1 has no machine-readable resolution field.
type Finding struct {
	ID          string `json:"id"`
	Location    string `json:"location"`
	Impact      string `json:"impact"`
	Evidence    string `json:"evidence"`
	Disposition string `json:"disposition"`
}

// Receipt is the portable v1 observation receipt shape, exactly.
type Receipt struct {
	Version          int               `json:"version"`
	Synthetic        bool              `json:"synthetic"`
	RunID            string            `json:"run_id"`
	SessionID        *string           `json:"session_id"`
	Mode             string            `json:"mode"`
	Workspace        string            `json:"workspace"`
	RequestRef       string            `json:"request_ref"`
	AuthorizationRef *string           `json:"authorization_ref"`
	Feature          *string           `json:"feature"`
	Batch            *string           `json:"batch"`
	TaskIDs          []string          `json:"task_ids"`
	AllowedEffects   []string          `json:"allowed_effects"`
	Profile          *string           `json:"profile"`
	Tools            []string          `json:"tools"`
	Limits           Limits            `json:"limits"`
	Identity         Identity          `json:"identity"`
	Model            Model             `json:"model"`
	Selection        Selection         `json:"selection"`
	Checks           []CheckEntry      `json:"checks"`
	Execution        string            `json:"execution"`
	Review           *string           `json:"review"`
	WorkflowState    *string           `json:"workflow_state"`
	Findings         []Finding         `json:"findings"`
	AffectedFiles    []string          `json:"affected_files"`
	NextAction       string            `json:"next_action"`
	CostUSD          *float64          `json:"cost_usd"`
	Unknowns         map[string]string `json:"unknowns"`
}

var (
	modes       = map[string]bool{"draft": true, "build": true, "direct-fix": true, "review": true, "resume": true}
	executions  = map[string]bool{"completed": true, "incomplete": true, "blocked": true, "cancelled": true, "failed": true}
	reviews     = map[string]bool{"Ready": true, "Needs changes": true, "Blocked": true}
	disposition = map[string]bool{"valid": true, "disputed": true, "needs evidence": true}
	statuses    = map[string]bool{"passed": true, "failed": true, "blocked": true, "incomplete": true, "cancelled": true, "not_run": true}
	scopes      = map[string]bool{"baseline": true, "focused": true, "required": true}
	unknownPtr  = regexp.MustCompile(`^/`)
)

// ParseReceipt decodes and type-checks receipt bytes against the v1 shape:
// exact field sets, no unknown fields, no duplicate JSON keys, no nulls where
// v1 forbids them, enums, lengths and date-time formats. It is schema-level
// only; semantic consistency against independent observations is Validate.
// Rejection messages never echo rejected values.
func ParseReceipt(data []byte) (Receipt, error) {
	var r Receipt
	if err := noDuplicateKeys(data); err != nil {
		return r, err
	}
	top, err := objectFields(data,
		"version", "synthetic", "run_id", "session_id", "mode", "workspace", "request_ref", "authorization_ref",
		"feature", "batch", "task_ids", "allowed_effects", "profile", "tools", "limits", "identity", "model",
		"selection", "checks", "execution", "review", "workflow_state", "findings", "affected_files",
		"next_action", "cost_usd", "unknowns")
	if err != nil {
		return r, err
	}
	get := func(name string) json.RawMessage { return top[name] }
	version, err := reqInt(get("version"), 1)
	if err != nil {
		return r, fmt.Errorf("version: %w", err)
	}
	r.Version = int(version)
	if err := reqBool(get("synthetic"), &r.Synthetic); err != nil {
		return r, fmt.Errorf("synthetic: %w", err)
	}
	if r.RunID, err = reqText(get("run_id")); err != nil {
		return r, fmt.Errorf("run_id: %w", err)
	}
	if r.SessionID, err = optText(get("session_id")); err != nil {
		return r, fmt.Errorf("session_id: %w", err)
	}
	if r.Mode, err = reqEnum(get("mode"), modes); err != nil {
		return r, fmt.Errorf("mode: %w", err)
	}
	if r.Workspace, err = reqText(get("workspace")); err != nil {
		return r, fmt.Errorf("workspace: %w", err)
	}
	if r.RequestRef, err = reqText(get("request_ref")); err != nil {
		return r, fmt.Errorf("request_ref: %w", err)
	}
	if r.AuthorizationRef, err = optText(get("authorization_ref")); err != nil {
		return r, fmt.Errorf("authorization_ref: %w", err)
	}
	if r.Feature, err = optText(get("feature")); err != nil {
		return r, fmt.Errorf("feature: %w", err)
	}
	if r.Batch, err = optText(get("batch")); err != nil {
		return r, fmt.Errorf("batch: %w", err)
	}
	if r.TaskIDs, err = textList(get("task_ids")); err != nil {
		return r, fmt.Errorf("task_ids: %w", err)
	}
	if r.AllowedEffects, err = textList(get("allowed_effects")); err != nil {
		return r, fmt.Errorf("allowed_effects: %w", err)
	}
	if r.Profile, err = optText(get("profile")); err != nil {
		return r, fmt.Errorf("profile: %w", err)
	}
	if r.Tools, err = textList(get("tools")); err != nil {
		return r, fmt.Errorf("tools: %w", err)
	}
	if r.Limits, err = parseLimits(get("limits")); err != nil {
		return r, fmt.Errorf("limits: %w", err)
	}
	if r.Identity, err = parseIdentity(get("identity")); err != nil {
		return r, fmt.Errorf("identity: %w", err)
	}
	if r.Model, err = parseModel(get("model")); err != nil {
		return r, fmt.Errorf("model: %w", err)
	}
	if r.Selection, err = parseSelection(get("selection")); err != nil {
		return r, fmt.Errorf("selection: %w", err)
	}
	if r.Checks, err = parseChecks(get("checks")); err != nil {
		return r, fmt.Errorf("checks: %w", err)
	}
	if r.Execution, err = reqEnum(get("execution"), executions); err != nil {
		return r, fmt.Errorf("execution: %w", err)
	}
	if r.Review, err = optEnum(get("review"), reviews); err != nil {
		return r, fmt.Errorf("review: %w", err)
	}
	if r.WorkflowState, err = optText(get("workflow_state")); err != nil {
		return r, fmt.Errorf("workflow_state: %w", err)
	}
	if r.Findings, err = parseFindings(get("findings")); err != nil {
		return r, fmt.Errorf("findings: %w", err)
	}
	if r.AffectedFiles, err = textList(get("affected_files")); err != nil {
		return r, fmt.Errorf("affected_files: %w", err)
	}
	if r.NextAction, err = reqText(get("next_action")); err != nil {
		return r, fmt.Errorf("next_action: %w", err)
	}
	if r.CostUSD, err = optNumber(get("cost_usd")); err != nil {
		return r, fmt.Errorf("cost_usd: %w", err)
	}
	if r.Unknowns, err = parseUnknowns(get("unknowns")); err != nil {
		return r, fmt.Errorf("unknowns: %w", err)
	}
	// The v1 schema conditions a direct fix to no feature, batch or task IDs:
	// a direct fix has no SDD tracking state to invent.
	if r.Mode == "direct-fix" && (r.Feature != nil || r.Batch != nil || len(r.TaskIDs) != 0) {
		return r, fmt.Errorf("direct fix has no feature, batch or task IDs")
	}
	return r, nil
}

func parseLimits(raw json.RawMessage) (Limits, error) {
	var l Limits
	fields, err := objectFields(raw, "deadline_seconds", "steps", "transient_retries", "paid_authorized", "spend_usd")
	if err != nil {
		return l, err
	}
	if l.DeadlineSeconds, err = reqIntAtLeast(fields["deadline_seconds"], 1); err != nil {
		return l, fmt.Errorf("deadline_seconds: %w", err)
	}
	if l.Steps, err = reqIntAtLeast(fields["steps"], 1); err != nil {
		return l, fmt.Errorf("steps: %w", err)
	}
	if l.TransientRetries, err = reqIntAtLeast(fields["transient_retries"], 0); err != nil {
		return l, fmt.Errorf("transient_retries: %w", err)
	}
	if err := reqBool(fields["paid_authorized"], &l.PaidAuthorized); err != nil {
		return l, fmt.Errorf("paid_authorized: %w", err)
	}
	if l.SpendUSD, err = optNumber(fields["spend_usd"]); err != nil {
		return l, fmt.Errorf("spend_usd: %w", err)
	}
	return l, nil
}

func parseIdentity(raw json.RawMessage) (Identity, error) {
	var i Identity
	fields, err := objectFields(raw, "base_revision", "diff", "config", "skills", "environment", "runtime", "harness", "contract")
	if err != nil {
		return i, err
	}
	values := [8]*string{}
	for j, name := range []string{"base_revision", "diff", "config", "skills", "environment", "runtime", "harness", "contract"} {
		if values[j], err = optText(fields[name]); err != nil {
			return i, fmt.Errorf("%s: %w", name, err)
		}
	}
	return Identity{values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7]}, nil
}

func parseModel(raw json.RawMessage) (Model, error) {
	var m Model
	fields, err := objectFields(raw, "provider", "id", "variant")
	if err != nil {
		return m, err
	}
	if m.Provider, err = optText(fields["provider"]); err != nil {
		return m, fmt.Errorf("provider: %w", err)
	}
	if m.ID, err = optText(fields["id"]); err != nil {
		return m, fmt.Errorf("id: %w", err)
	}
	if m.Variant, err = optText(fields["variant"]); err != nil {
		return m, fmt.Errorf("variant: %w", err)
	}
	return m, nil
}

func parseSelection(raw json.RawMessage) (Selection, error) {
	var s Selection
	fields, err := objectFields(raw, "scope", "check_ids")
	if err != nil {
		return s, err
	}
	if s.Scope, err = reqEnum(fields["scope"], scopes); err != nil {
		return s, fmt.Errorf("scope: %w", err)
	}
	if s.CheckIDs, err = textList(fields["check_ids"]); err != nil {
		return s, fmt.Errorf("check_ids: %w", err)
	}
	return s, nil
}

func parseChecks(raw json.RawMessage) ([]CheckEntry, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("must be an array")
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("must be an array")
	}
	out := make([]CheckEntry, 0, len(list))
	for _, item := range list {
		fields, err := objectFields(item, "id", "status", "exit_code", "started_at", "ended_at", "evidence", "truncated", "reason")
		if err != nil {
			return nil, err
		}
		var c CheckEntry
		if c.ID, err = reqText(fields["id"]); err != nil {
			return nil, fmt.Errorf("id: %w", err)
		}
		if c.Status, err = reqEnum(fields["status"], statuses); err != nil {
			return nil, fmt.Errorf("status: %w", err)
		}
		if c.ExitCode, err = optInt(fields["exit_code"]); err != nil {
			return nil, fmt.Errorf("exit_code: %w", err)
		}
		for _, stamp := range []struct {
			name string
			into **string
		}{{"started_at", &c.StartedAt}, {"ended_at", &c.EndedAt}} {
			if *stamp.into, err = optDateTime(fields[stamp.name]); err != nil {
				return nil, fmt.Errorf("%s: %w", stamp.name, err)
			}
		}
		if c.Evidence, err = optText(fields["evidence"]); err != nil {
			return nil, fmt.Errorf("evidence: %w", err)
		}
		if err := reqBool(fields["truncated"], &c.Truncated); err != nil {
			return nil, fmt.Errorf("truncated: %w", err)
		}
		if c.Reason, err = reqText(fields["reason"]); err != nil {
			return nil, fmt.Errorf("reason: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

func parseFindings(raw json.RawMessage) ([]Finding, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("must be an array")
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("must be an array")
	}
	out := make([]Finding, 0, len(list))
	for _, item := range list {
		fields, err := objectFields(item, "id", "location", "impact", "evidence", "disposition")
		if err != nil {
			return nil, err
		}
		var f Finding
		if f.ID, err = reqText(fields["id"]); err != nil {
			return nil, fmt.Errorf("id: %w", err)
		}
		if f.Location, err = reqText(fields["location"]); err != nil {
			return nil, fmt.Errorf("location: %w", err)
		}
		if f.Impact, err = reqText(fields["impact"]); err != nil {
			return nil, fmt.Errorf("impact: %w", err)
		}
		if f.Evidence, err = reqText(fields["evidence"]); err != nil {
			return nil, fmt.Errorf("evidence: %w", err)
		}
		if f.Disposition, err = reqEnum(fields["disposition"], disposition); err != nil {
			return nil, fmt.Errorf("disposition: %w", err)
		}
		out = append(out, f)
	}
	return out, nil
}

func parseUnknowns(raw json.RawMessage) (map[string]string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("must be an object")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("must be an object")
	}
	out := make(map[string]string, len(m))
	for key, value := range m {
		if !unknownPtr.MatchString(key) {
			return nil, fmt.Errorf("unknown reason keys must be JSON pointers")
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("unknown reasons must be nonempty strings")
		}
		out[key] = text
	}
	return out, nil
}

// objectFields decodes one JSON object with an exact field set. Missing and
// unknown field names are reported by name only; values are never echoed.
func objectFields(data []byte, required ...string) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("must be an object")
	}
	allowed := make(map[string]bool, len(required))
	for _, name := range required {
		allowed[name] = true
		if _, ok := m[name]; !ok {
			return nil, fmt.Errorf("missing field %q", name)
		}
	}
	for name := range m {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown field %q", name)
		}
	}
	return m, nil
}

// noDuplicateKeys walks the token stream so duplicate object keys anywhere in
// the document are refused instead of silently taking the last value.
func noDuplicateKeys(data []byte) error {
	type level struct {
		object  bool
		wantKey bool
		keys    map[string]bool
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON")
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("must be a JSON object")
	}
	stack := []level{{object: true, wantKey: true, keys: map[string]bool{}}}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("invalid JSON")
		}
		if len(stack) == 0 {
			return fmt.Errorf("trailing data after object")
		}
		cur := &stack[len(stack)-1]
		if cur.object && cur.wantKey {
			key, ok := tok.(string)
			if !ok {
				// Only a closing brace may appear where a key was expected;
				// let the delimiter switch below close the object.
				if delim, isDelim := tok.(json.Delim); !isDelim || delim != '}' {
					return fmt.Errorf("invalid JSON")
				}
			} else {
				if cur.keys[key] {
					return fmt.Errorf("duplicate field %q", key)
				}
				cur.keys[key] = true
				cur.wantKey = false
				continue
			}
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
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
			if cur.object {
				cur.wantKey = true
			}
		}
	}
	if len(stack) != 0 {
		return fmt.Errorf("unterminated JSON container")
	}
	return nil
}

func reqText(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || s == "" || strings.ContainsRune(s, 0) {
		return "", fmt.Errorf("must be a nonempty string")
	}
	return s, nil
}

func optText(raw json.RawMessage) (*string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	s, err := reqText(raw)
	if err != nil {
		return nil, fmt.Errorf("must be null or a nonempty string")
	}
	return &s, nil
}

func optDateTime(raw json.RawMessage) (*string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("must be null or a date-time string")
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return nil, fmt.Errorf("must be an RFC 3339 date-time")
	}
	return &s, nil
}

func reqBool(raw json.RawMessage, into *bool) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("must be a boolean")
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("must be a boolean")
	}
	return nil
}

// reqInt accepts only integer-form JSON numbers, never 1.0 or 1.5, so a
// float can never masquerade as a count or bound.
func reqInt(raw json.RawMessage, want int64) (int64, error) {
	value, err := integer(raw)
	if err != nil || value != want {
		return 0, fmt.Errorf("must be the integer %d", want)
	}
	return value, nil
}

func reqIntAtLeast(raw json.RawMessage, minimum int64) (int64, error) {
	value, err := integer(raw)
	if err != nil || value < minimum {
		return 0, fmt.Errorf("must be an integer of at least %d", minimum)
	}
	return value, nil
}

func optInt(raw json.RawMessage) (*int64, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	value, err := integer(raw)
	if err != nil {
		return nil, fmt.Errorf("must be null or an integer")
	}
	return &value, nil
}

func integer(raw json.RawMessage) (int64, error) {
	var n json.Number
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("must be a number")
	}
	value, err := n.Int64()
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	// Reject "1.0"-style forms that Int64 accepts via truncation rules.
	if n.String() != fmt.Sprintf("%d", value) {
		return 0, fmt.Errorf("must be an integer")
	}
	return value, nil
}

func optNumber(raw json.RawMessage) (*float64, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("must be null or a number")
	}
	if value < 0 || value != value {
		return nil, fmt.Errorf("must be a nonnegative number")
	}
	return &value, nil
}

func reqEnum(raw json.RawMessage, allowed map[string]bool) (string, error) {
	s, err := reqText(raw)
	if err != nil || !allowed[s] {
		return "", fmt.Errorf("has an unsupported value")
	}
	return s, nil
}

func optEnum(raw json.RawMessage, allowed map[string]bool) (*string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	s, err := reqEnum(raw, allowed)
	if err != nil {
		return nil, fmt.Errorf("must be null or a supported value")
	}
	return &s, nil
}

func textList(raw json.RawMessage) ([]string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("must be an array of strings")
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("must be an array of strings")
	}
	seen := map[string]bool{}
	for _, s := range list {
		if s == "" || strings.ContainsRune(s, 0) {
			return nil, fmt.Errorf("must contain nonempty strings")
		}
		if seen[s] {
			return nil, fmt.Errorf("must not contain duplicates")
		}
		seen[s] = true
	}
	return list, nil
}

func sameOptionalText(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
