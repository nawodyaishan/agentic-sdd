package harnessevidence

// B4 T09 (AC-18): the runner result reader accepts exactly one schema-valid
// JSON document plus permitted trailing whitespace, and rejects malformed
// tails, extra documents and duplicate keys with bounded diagnostics.

import (
	"encoding/json"
	"strings"
	"testing"

	"agentic-sdd/internal/harnesscheck"
)

func b4ValidResult(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(harnesscheck.Result{
		Version:        1,
		Workspace:      "/synthetic/workspace",
		ContractSHA256: "synthetic-digest",
		Selection:      harnesscheck.Selection{Scope: "required", CheckIDs: []string{"test"}},
		Execution:      "blocked",
		Reason:         "synthetic",
		Checks:         []harnesscheck.CheckResult{{ID: "test", Status: "blocked", Reason: "synthetic"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParseResultAcceptsTrailingWhitespace(t *testing.T) {
	valid := b4ValidResult(t)
	for _, tail := range []string{"", " ", "\n", "\r\n\t  \n"} {
		document := append(append([]byte{}, valid...), tail...)
		got, err := ParseResult(document)
		if err != nil || got.Version != 1 || got.Workspace != "/synthetic/workspace" {
			t.Fatalf("trailing whitespace %q rejected: %+v %v", tail, got, err)
		}
	}
}

func TestParseResultRejectsMalformedTailsAndExtraDocuments(t *testing.T) {
	valid := string(b4ValidResult(t))
	for name, document := range map[string]string{
		"malformed-tail":   valid + "x",
		"truncated-second": valid + `{`,
		"comment-tail":     valid + "// trailing",
		"second-object":    valid + `{"version":1}`,
		"second-array":     valid + `[]`,
		"second-scalar":    valid + `null`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResult([]byte(document)); err == nil {
				t.Fatal("accepted a result with more than one value")
			}
		})
	}
}

func TestParseResultRejectsDuplicateKeys(t *testing.T) {
	for name, document := range map[string]string{
		"top-level":        `{"version":1,"version":1}`,
		"different-values": `{"version":1,"reason":"first","reason":"second"}`,
		"nested-check":     `{"version":1,"checks":[{"id":"test","id":"other"}]}`,
		"nested-object":    `{"version":1,"selection":{"scope":"required","scope":"focused"}}`,
		"after-nested":     `{"version":1,"selection":{"scope":"required"},"selection":{}}`,
		"escaped-key":      `{"version":1,"re\u0061son":"first","reason":"second"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResult([]byte(document)); err == nil {
				t.Fatal("accepted duplicate object keys")
			}
		})
	}
}

func TestParseResultDoesNotTreatStringContentAsKeys(t *testing.T) {
	document := `{"version":1,"reason":"text {\"version\":1,\"version\":1}"}`
	if _, err := ParseResult([]byte(document)); err != nil {
		t.Fatalf("duplicate-looking text inside a string was rejected: %v", err)
	}
}

func TestParseResultDiagnosticsAreBounded(t *testing.T) {
	hostile := strings.Repeat("A", 5000)
	for name, document := range map[string]string{
		"tail":      `{"version":1}` + hostile,
		"duplicate": `{"version":1,"reason":"` + hostile + `","reason":"again"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseResult([]byte(document))
			if err == nil {
				t.Fatal("accepted hostile input")
			}
			if len(err.Error()) > 64 || strings.Contains(err.Error(), hostile) {
				t.Fatalf("diagnostic echoed input or is too long: %d bytes", len(err.Error()))
			}
		})
	}
}

func TestParseResultAcceptsExistingFixtureResult(t *testing.T) {
	f := newReceiptFixture(t)
	if _, err := ParseResult(marshalJSON(t, f.result)); err != nil {
		t.Fatalf("existing valid fixture result rejected: %v", err)
	}
}
