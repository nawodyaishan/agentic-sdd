package opencodeadapter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAdmitIsAlwaysDeniedWithoutControls(t *testing.T) {
	full := LaunchDeclaration{
		Authorization: "specs/005-vnext-completion/spec.md#approval",
		Effects:       []string{"one native prompt"},
		Model:         Model,
		Data:          "isolated-credential-free-state",
		Deadline:      "15m",
		Attempts:      1,
		TokenBound:    1024,
		SpendBound:    "$0",
	}
	if got := Admit(LaunchDeclaration{}); len(got) == 0 {
		t.Fatal("empty declaration admitted")
	}
	// Even a fully declared launch must stay denied: declarations are not proof
	// of the controls.
	got := Admit(full)
	for _, control := range admissionControls {
		if !contains(got, control) {
			t.Fatalf("missing control prerequisite %q in %v", control, got)
		}
	}
}

func TestNewPendingSummaryNormalizesUnknownRationale(t *testing.T) {
	s, err := newPendingSummary("ses_test", "/tmp/ws", "Which package manager?", []PendingOption{{ID: "bun", Label: "Bun"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != pendingVersion || s.Rationale != "unknown" || s.Run != "pending_human" || s.Question != "Which package manager?" {
		t.Fatalf("bad summary: %+v", s)
	}
	for _, bad := range []struct{ session, workspace, question string }{
		{"", "/tmp/ws", "q"},
		{"s", "", "q"},
		{"s", "/tmp/ws", ""},
		{"s", "/tmp/ws", string(make([]byte, 5000))},
	} {
		if _, err := newPendingSummary(bad.session, bad.workspace, bad.question, nil, ""); err == nil {
			t.Fatalf("accepted invalid summary %+v", bad)
		}
	}
}

func TestParseOptionsExtractsBoundedChoices(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"pm":{"type":"string","oneOf":[{"const":"bun","title":"Bun"},{"const":"npm","title":"npm"}]}},"required":["pm"]}`)
	opts := parseOptions(schema)
	if len(opts) != 2 || opts[0].ID != "bun" || opts[1].Label != "npm" {
		t.Fatalf("bad options: %+v", opts)
	}
	if got := parseOptions(json.RawMessage(`{"unknown":"shape"}`)); got != nil {
		t.Fatalf("unknown schema parsed: %+v", got)
	}
}

func privateTestDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestWriteReadPendingRoundTripAndRefusals(t *testing.T) {
	dir := privateTestDir(t)
	s, err := newPendingSummary("ses_test", "/tmp/ws", "Which package manager?", []PendingOption{{ID: "bun", Label: "Bun"}}, "unknown")
	if err != nil {
		t.Fatal(err)
	}
	path, err := WritePending(dir, s)
	if err != nil {
		t.Fatal(err)
	}
	got, sum, err := ReadPending(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Session != s.Session || got.Question != s.Question || sum == "" {
		t.Fatalf("round trip mismatch: %+v %s", got, sum)
	}
	// No overwrite/replay.
	if _, err := WritePending(dir, s); err == nil {
		t.Fatal("overwrote an existing pending summary")
	}
	// Tampered (unknown field) and wrong-version files are refused.
	tampered := filepath.Join(t.TempDir(), "pending.json")
	if err := os.WriteFile(tampered, []byte(`{"version":1,"run":"x","session":"s","workspace":"/w","question":"q","rationale":"unknown","scope":"s","next_action":"n","extra":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadPending(tampered); err == nil {
		t.Fatal("accepted tampered summary with unknown field")
	}
	badVersion := filepath.Join(t.TempDir(), "pending.json")
	if err := os.WriteFile(badVersion, []byte(`{"version":99,"run":"x","session":"s","workspace":"/w","question":"q","rationale":"unknown","scope":"s","next_action":"n"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadPending(badVersion); err == nil {
		t.Fatal("accepted wrong-version summary")
	}
}

func TestWritePendingRequiresPrivateNonDiscoveryDirectory(t *testing.T) {
	s, _ := newPendingSummary("ses_test", "/tmp/ws", "q?", nil, "")
	// A directory inside active discovery is refused.
	root := t.TempDir()
	active := filepath.Join(root, ".opencode")
	if err := os.MkdirAll(active, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := WritePending(active, s); err == nil {
		t.Fatal("wrote pending summary into active discovery")
	}
}
