package harnessevidence

// Safe external export tests: redaction happens before any byte is written and
// the boundary refuses overwrite, in-checkout targets and symlinked parents.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureCanary = "SYNTHETIC-SECRET-CANARY-DO-NOT-EXPORT-12345"

func TestExportRedactsBeforeWriteAndValidates(t *testing.T) {
	f := newReceiptFixture(t)
	f.receipt["request_ref"] = "request containing " + fixtureCanary
	f.receiptUnknowns()["/cost_usd"] = "Bearer " + fixtureCanary
	output := filepath.Join(f.evidence, "receipt.json")
	if err := Export(marshalJSON(t, f.receipt), output, f.workspace, []string{fixtureCanary}); err != nil {
		t.Fatalf("export: %v", err)
	}
	written, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), fixtureCanary) {
		t.Fatalf("redaction canary reached the exported receipt")
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("exported receipt is not private: %v", info.Mode().Perm())
	}
	if err := Validate(written, f.input()); err != nil {
		t.Fatalf("exported receipt does not validate: %v", err)
	}
}

func TestExportRefusesOverwriteAndUnsafeTargets(t *testing.T) {
	f := newReceiptFixture(t)
	data := marshalJSON(t, f.receipt)

	output := filepath.Join(f.evidence, "receipt.json")
	if err := Export(data, output, f.workspace, nil); err != nil {
		t.Fatal(err)
	}
	if err := Export(data, output, f.workspace, nil); err == nil {
		t.Fatalf("existing export destination was overwritten")
	}

	inside := filepath.Join(f.workspace, "leaked.json")
	if err := Export(data, inside, f.workspace, nil); err == nil {
		t.Fatalf("checkout-internal export was accepted")
	}
	if _, err := os.Lstat(inside); err == nil {
		t.Fatalf("refused export still created a file")
	}

	escaped := f.evidence + "/../project/leaked.json"
	if err := os.Chmod(f.workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Export(data, escaped, f.workspace, nil); err == nil {
		t.Fatalf("dotdot export path was accepted")
	}
	if _, err := os.Lstat(filepath.Join(f.workspace, "leaked.json")); err == nil {
		t.Fatalf("dotdot export still created a file")
	}

	linked := filepath.Join(f.root, "linked")
	if err := os.Symlink(f.evidence, linked); err != nil {
		t.Fatal(err)
	}
	if err := Export(data, filepath.Join(linked, "escape.json"), f.workspace, nil); err == nil {
		t.Fatalf("symlinked export parent was accepted")
	}
	if _, err := os.Lstat(filepath.Join(f.evidence, "escape.json")); err == nil {
		t.Fatalf("symlinked export still created a file")
	}

	if err := Export(data, filepath.Join(f.evidence, "empty.json"), f.workspace, []string{""}); err == nil {
		t.Fatalf("empty redaction string was accepted")
	}
}

func TestExportRejectsUnapprovedFieldsWithoutEchoingValues(t *testing.T) {
	f := newReceiptFixture(t)
	bad := cloneValue(t, f.receipt).(map[string]any)
	bad["raw_native_trace"] = fixtureCanary
	output := filepath.Join(f.evidence, "raw.json")
	err := Export(marshalJSON(t, bad), output, f.workspace, []string{fixtureCanary})
	if err == nil {
		t.Fatalf("unapproved field was exported")
	}
	if strings.Contains(err.Error(), fixtureCanary) {
		t.Fatalf("rejection message echoed a secret value")
	}
	if _, statErr := os.Lstat(output); statErr == nil {
		t.Fatalf("rejected export still created a file")
	}

	sensitive := cloneValue(t, f.receipt).(map[string]any)
	sensitive["unknowns"].(map[string]any)["/"+fixtureCanary] = "reason"
	err = Export(marshalJSON(t, sensitive), filepath.Join(f.evidence, "named.json"), f.workspace, []string{fixtureCanary})
	if err == nil {
		t.Fatalf("sensitive field name was exported")
	}
	if strings.Contains(err.Error(), fixtureCanary) {
		t.Fatalf("rejection message echoed a secret field name")
	}
}

func TestExportRejectsInvalidReceiptAndMalformedJSON(t *testing.T) {
	f := newReceiptFixture(t)
	output := filepath.Join(f.evidence, "invalid.json")
	if err := Export([]byte(`{"version":`), output, f.workspace, nil); err == nil {
		t.Fatalf("malformed JSON was exported")
	}
	broken := cloneValue(t, f.receipt).(map[string]any)
	broken["execution"] = "finished"
	if err := Export(marshalJSON(t, broken), output, f.workspace, nil); err == nil {
		t.Fatalf("schema-invalid receipt was exported")
	}
	if _, err := os.Lstat(output); err == nil {
		t.Fatalf("invalid export created a file")
	}
}

func TestExportScrubsCredentialForms(t *testing.T) {
	f := newReceiptFixture(t)
	f.receipt["next_action"] = "retry with api_key=abcdef123456"
	f.receipt["allowed_effects"] = []any{"Bearer opaque-credential-9876"}
	output := filepath.Join(f.evidence, "scrubbed.json")
	if err := Export(marshalJSON(t, f.receipt), output, f.workspace, nil); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := json.Unmarshal(written, &exported); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), "abcdef123456") || strings.Contains(string(written), "opaque-credential-9876") {
		t.Fatalf("credential form was not scrubbed")
	}
	if !strings.Contains(exported["next_action"].(string), "[REDACTED]") {
		t.Fatalf("scrubbed value lost its replacement marker")
	}
}
