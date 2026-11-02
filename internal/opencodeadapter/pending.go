package opencodeadapter

// Pending-human normalization: a bounded, sanitized, resumable question summary.
// It is an observation of a native stop, never a second approval ledger, and it
// never fabricates an answer, rationale or approver.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pendingVersion is the summary schema version written by this adapter.
const pendingVersion = 1

// PendingOption is one allowlisted form/permission choice.
type PendingOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PendingSummary is the sanitized resumable question captured when the owned
// prompt stops on a consequential human input. Rationale stays "unknown" when
// native data does not supply one.
type PendingSummary struct {
	Version      int             `json:"version"`
	Run          string          `json:"run"`
	Session      string          `json:"session"`
	Workspace    string          `json:"workspace"`
	Request      string          `json:"request"`
	Authority    string          `json:"authority,omitempty"`
	Question     string          `json:"question"`
	Options      []PendingOption `json:"options"`
	Rationale    string          `json:"rationale"`
	Scope        string          `json:"scope"`
	Evidence     []string        `json:"evidence,omitempty"`
	NextAction   string          `json:"next_action"`
}

// MaxPendingBytes bounds a persisted summary and its answer payload.
const MaxPendingBytes = acpMaxMessage

// newPendingSummary builds the required summary fields from an allowlisted
// question and options. Every free-text field is bounded; an empty rationale is
// normalized to the explicit unknown value rather than invented.
func newPendingSummary(session, workspace, question string, options []PendingOption, rationale string) (PendingSummary, error) {
	if session == "" || workspace == "" {
		return PendingSummary{}, fmt.Errorf("pending summary requires session and workspace identity")
	}
	q := strings.TrimSpace(question)
	if q == "" {
		return PendingSummary{}, fmt.Errorf("pending summary requires an exact question")
	}
	if len(q) > 4096 {
		return PendingSummary{}, fmt.Errorf("pending question exceeds bound")
	}
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		rationale = "unknown"
	}
	if len(rationale) > 4096 {
		return PendingSummary{}, fmt.Errorf("pending rationale exceeds bound")
	}
	opts := make([]PendingOption, 0, len(options))
	seen := map[string]bool{}
	for _, o := range options {
		id := strings.TrimSpace(o.ID)
		label := strings.TrimSpace(o.Label)
		if id == "" || label == "" || len(id) > 128 || len(label) > 512 || seen[id] {
			return PendingSummary{}, fmt.Errorf("invalid pending option")
		}
		seen[id] = true
		opts = append(opts, PendingOption{ID: id, Label: label})
	}
	return PendingSummary{
		Version:    pendingVersion,
		Run:        "pending_human",
		Session:    session,
		Workspace:  workspace,
		Request:    "native pending human input",
		Question:   q,
		Options:    opts,
		Rationale:  rationale,
		Scope:      "consequential human answer required before resumption",
		NextAction: "human supplies the answer; reconcile scope/authority before re-entry",
	}, nil
}

// parseOptions extracts bounded string options from a JSON Schema subset shaped
// like `properties.<key>.oneOf[{const,title}]`. It is intentionally strict: it
// never interprets arbitrary schemas, and returns no options when it cannot
// decode the expected shape.
func parseOptions(schema json.RawMessage) []PendingOption {
	var root struct {
		Properties map[string]struct {
			OneOf []struct {
				Const string `json:"const"`
				Title string `json:"title"`
			} `json:"oneOf"`
		} `json:"properties"`
	}
	d := json.NewDecoder(bytes.NewReader(schema))
	if err := d.Decode(&root); err != nil {
		return nil
	}
	var out []PendingOption
	for _, prop := range root.Properties {
		for _, o := range prop.OneOf {
			if o.Const == "" || o.Title == "" {
				return nil
			}
			out = append(out, PendingOption{ID: o.Const, Label: o.Title})
		}
	}
	return out
}

// WritePending persists a sanitized pending summary to a new owner-private file
// outside active discovery, refusing an existing file (no overwrite/replay).
func WritePending(dir string, s PendingSummary) (string, error) {
	if s.Version != pendingVersion {
		return "", fmt.Errorf("pending summary version mismatch")
	}
	if s.Session == "" || s.Workspace == "" || strings.TrimSpace(s.Question) == "" {
		return "", fmt.Errorf("pending summary missing required fields")
	}
	dir, err := privateDirectory(dir)
	if err != nil {
		return "", err
	}
	if activePath(dir) {
		return "", fmt.Errorf("pending storage must be outside active discovery")
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil || len(data) > MaxPendingBytes {
		return "", fmt.Errorf("pending summary exceeds bound")
	}
	data = append(data, '\n')
	path := filepath.Join(dir, "pending-"+sanitizeToken(s.Session)+".json")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("pending summary already exists; re-entry requires a fresh reconcile, not a replay")
	}
	if err := atomicWrite(path, data, 0600); err != nil {
		return "", err
	}
	return path, nil
}

// ReadPending loads and validates a persisted pending summary. It returns the
// summary and its digest; a stale, foreign or tampered file is refused rather
// than replayed into re-entry.
func ReadPending(path string) (PendingSummary, string, error) {
	data, err := ReadInput(path)
	if err != nil {
		return PendingSummary{}, "", err
	}
	var s PendingSummary
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil || s.Version != pendingVersion {
		return PendingSummary{}, "", fmt.Errorf("invalid pending summary")
	}
	if s.Session == "" || s.Workspace == "" || strings.TrimSpace(s.Question) == "" || s.Rationale == "" || s.NextAction == "" {
		return PendingSummary{}, "", fmt.Errorf("pending summary missing required fields")
	}
	return s, digest(data), nil
}

// sanitizeToken reduces an identifier to a safe filename fragment.
func sanitizeToken(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		out = "unknown"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
