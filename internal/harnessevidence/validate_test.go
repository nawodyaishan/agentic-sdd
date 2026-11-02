package harnessevidence

// Semantic validation tests. They mirror the reference consumer's cases:
// consistency and freshness only, never consent or review reasoning.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var identityFields = []string{
	"base_revision", "diff", "config", "skills", "environment", "runtime", "harness", "contract",
}

func TestValidateAcceptsValidDirectFixWithoutTouchingWorkspace(t *testing.T) {
	f := newReceiptFixture(t)
	before := f.workspaceFiles()
	if err := f.validate(); err != nil {
		t.Fatalf("valid direct-fix receipt rejected: %v", err)
	}
	after := f.workspaceFiles()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("validation modified the target workspace")
	}
	if f.receipt["feature"] != nil || f.receipt["batch"] != nil {
		t.Fatalf("direct fix invented feature or batch state")
	}
	if got := f.receipt["task_ids"].([]any); len(got) != 0 {
		t.Fatalf("direct fix invented task IDs")
	}
}

func TestValidateRejectsSchemaAndSemanticInconsistency(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(f *receiptFixture)
	}{
		{"unknown field", func(f *receiptFixture) { f.receipt["approved"] = true }},
		{"invented batch", func(f *receiptFixture) { f.receipt["batch"] = "invented-batch" }},
		{"invented workflow state", func(f *receiptFixture) { f.receipt["workflow_state"] = "invented" }},
		{"missing unknown reason", func(f *receiptFixture) { delete(f.receiptUnknowns(), "/cost_usd") }},
		{"blank unknown reason", func(f *receiptFixture) { f.receiptUnknowns()["/cost_usd"] = "   " }},
		{"unknown reason for non-null", func(f *receiptFixture) {
			f.receiptUnknowns()["/not/a/field"] = "not a real observation"
		}},
		{"duplicate check", func(f *receiptFixture) {
			checks := f.receipt["checks"].([]any)
			f.receipt["checks"] = append(checks, cloneValue(t, checks[0]))
		}},
		{"nonzero exit as pass", func(f *receiptFixture) { f.receiptCheck()["exit_code"] = 9 }},
		{"ended before started", func(f *receiptFixture) {
			f.receiptCheck()["ended_at"] = "2026-10-28T23:59:59Z"
		}},
		{"truncated pass", func(f *receiptFixture) { f.receiptCheck()["truncated"] = true }},
		{"pass without evidence", func(f *receiptFixture) { f.receiptCheck()["evidence"] = nil }},
		{"focused ready", func(f *receiptFixture) { f.receipt["review"] = "Ready" }},
		{"invented authorization", func(f *receiptFixture) {
			f.receipt["authorization_ref"] = "receipt invents permission"
		}},
		{"execution contradicts checks", func(f *receiptFixture) { f.receipt["execution"] = "incomplete" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			f := newReceiptFixture(t)
			mutation.mutate(f)
			f.syncResult()
			if err := f.validate(); err == nil {
				t.Fatalf("inconsistent receipt was accepted")
			}
		})
	}

	t.Run("missing current authorization", func(t *testing.T) {
		f := newReceiptFixture(t)
		in := f.input()
		in.AuthorizationRef = ""
		if err := f.validateWith(in); err == nil {
			t.Fatalf("receipt was accepted without reconciled authorization")
		}
	})
}

func TestValidateRejectsStaleOrUnknownIdentity(t *testing.T) {
	for _, name := range identityFields {
		t.Run(name, func(t *testing.T) {
			f := newReceiptFixture(t)
			in := f.input()
			in.Identity = f.identityWith(name, "changed")
			if err := f.validateWith(in); err == nil {
				t.Fatalf("stale identity %q was accepted", name)
			}
		})
		t.Run(name+" unknown", func(t *testing.T) {
			f := newReceiptFixture(t)
			f.receiptIdentity()[name] = nil
			f.receiptUnknowns()["/identity/"+name] = "not observed"
			var identity Identity
			mustRoundTrip(t, f.receiptIdentity(), &identity)
			in := f.input()
			in.Identity = identity
			if err := f.validateWith(in); err == nil {
				t.Fatalf("unknown identity %q was accepted for pass reuse", name)
			}
		})
	}
}

func TestValidateCannotInventObservedValues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(f *receiptFixture)
	}{
		{"cost", func(f *receiptFixture) {
			f.receipt["cost_usd"] = 0
			delete(f.receiptUnknowns(), "/cost_usd")
		}},
		{"session", func(f *receiptFixture) {
			f.receipt["session_id"] = "fabricated-session"
			delete(f.receiptUnknowns(), "/session_id")
		}},
		{"paid", func(f *receiptFixture) { f.receiptLimits()["paid_authorized"] = true }},
		{"model", func(f *receiptFixture) {
			f.receiptModel()["id"] = "fabricated-model"
			delete(f.receiptUnknowns(), "/model/id")
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newReceiptFixture(t)
			c.mutate(f)
			f.syncResult()
			if err := f.validate(); err == nil {
				t.Fatalf("unobserved %s claim was accepted", c.name)
			}
		})
	}
}

func TestValidateAcceptsIndependentlyObservedSyntheticValues(t *testing.T) {
	f := newReceiptFixture(t)
	f.receipt["session_id"] = "synthetic-session"
	f.receipt["cost_usd"] = 0.01
	f.receiptModel()["provider"] = "synthetic"
	f.receiptModel()["id"] = "synthetic-model"
	f.receiptModel()["variant"] = "fixture"
	f.receiptLimits()["paid_authorized"] = true
	f.receiptLimits()["spend_usd"] = 0.25
	for _, ptr := range []string{"/session_id", "/cost_usd", "/model/provider", "/model/id", "/model/variant"} {
		delete(f.receiptUnknowns(), ptr)
	}
	in := f.input()
	in.PaidAuthorized = true
	in.Observations = Observations{
		"/session_id":     "synthetic-session",
		"/cost_usd":       json.Number("0.01"),
		"/model/provider": "synthetic",
		"/model/id":       "synthetic-model",
		"/model/variant":  "fixture",
	}
	if err := f.validateWith(in); err != nil {
		t.Fatalf("independently observed synthetic values rejected: %v", err)
	}
}

func TestValidateEvidenceBoundaries(t *testing.T) {
	t.Run("tampered", func(t *testing.T) {
		f := newReceiptFixture(t)
		if err := os.WriteFile(f.log, []byte("replaced evidence at the same path\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := f.validate(); err == nil {
			t.Fatalf("tampered evidence was accepted")
		}
	})
	t.Run("public", func(t *testing.T) {
		f := newReceiptFixture(t)
		if err := os.Chmod(f.log, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := f.validate(); err == nil {
			t.Fatalf("public evidence was accepted")
		}
	})
	t.Run("missing", func(t *testing.T) {
		f := newReceiptFixture(t)
		if err := os.Remove(f.log); err != nil {
			t.Fatal(err)
		}
		if err := f.validate(); err == nil {
			t.Fatalf("missing evidence was accepted")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		f := newReceiptFixture(t)
		other := filepath.Join(f.root, "outside.log")
		if err := os.WriteFile(other, []byte(fixtureLogText), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(f.log); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(other, f.log); err != nil {
			t.Fatal(err)
		}
		if err := f.validate(); err == nil {
			t.Fatalf("symlinked evidence was accepted")
		}
	})
	t.Run("dotdot traversal", func(t *testing.T) {
		f := newReceiptFixture(t)
		escaped := f.evidence + "/../outside.log"
		if err := os.WriteFile(filepath.Join(f.root, "outside.log"), []byte(fixtureLogText), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(f.workspace, 0o700); err != nil {
			t.Fatal(err)
		}
		f.receiptCheck()["evidence"] = escaped
		f.syncResult()
		f.logDigests[escaped] = sha256Hex([]byte(fixtureLogText))
		if err := f.validate(); err == nil {
			t.Fatalf("dotdot evidence path was accepted")
		}
	})
	t.Run("evidence root inside workspace", func(t *testing.T) {
		f := newReceiptFixture(t)
		in := f.input()
		in.EvidenceRoot = f.workspace
		if err := f.validateWith(in); err == nil {
			t.Fatalf("evidence root inside the target workspace was accepted")
		}
	})
}

func TestValidateReadyRules(t *testing.T) {
	f := newReceiptFixture(t)
	f.receiptSelection()["scope"] = "required"
	f.syncResult()
	f.receipt["review"] = "Ready"
	if err := f.validate(); err != nil {
		t.Fatalf("valid ready receipt rejected: %v", err)
	}

	findings := []any{map[string]any{
		"id": "F1", "location": "code.txt", "impact": "contract defect",
		"evidence": "fixture", "disposition": "needs evidence",
	}}
	f.receipt["findings"] = cloneValue(t, findings)
	if err := f.validate(); err == nil {
		t.Fatalf("ready receipt with unresolved findings was accepted")
	}

	resolved := cloneValue(t, findings).([]any)
	resolved[0].(map[string]any)["disposition"] = "disputed"
	f.receipt["findings"] = resolved
	if err := f.validate(); err != nil {
		t.Fatalf("ready receipt with resolved findings rejected: %v", err)
	}

	f.receipt["findings"] = append(resolved, cloneValue(t, resolved[0]))
	if err := f.validate(); err == nil {
		t.Fatalf("duplicate finding IDs were accepted")
	}
}

func TestValidateCancelledOrIncompleteCannotBePromoted(t *testing.T) {
	for _, status := range []string{"cancelled", "incomplete", "failed", "blocked"} {
		t.Run(status, func(t *testing.T) {
			f := newReceiptFixture(t)
			f.receipt["execution"] = status
			f.receiptCheck()["status"] = status
			if status == "blocked" {
				f.receiptCheck()["started_at"] = nil
				f.receiptCheck()["exit_code"] = nil
			}
			f.syncResult()
			if err := f.validate(); err != nil {
				t.Fatalf("%s receipt rejected: %v", status, err)
			}
			f.receipt["review"] = "Ready"
			if err := f.validate(); err == nil {
				t.Fatalf("%s receipt was promoted to Ready", status)
			}
			f.receipt["review"] = nil
			f.receipt["execution"] = "completed"
			if err := f.validate(); err == nil {
				t.Fatalf("%s receipt was promoted to completed", status)
			}
		})
	}
}

func TestValidateStatusCannotBeRelabelledAsAnotherOutcome(t *testing.T) {
	f := newReceiptFixture(t)
	f.receiptCheck()["status"] = "failed"
	f.receipt["execution"] = "cancelled"
	f.syncResult()
	if err := f.validate(); err == nil {
		t.Fatalf("check outcome was relabelled as a different execution")
	}
}

func TestValidateRequiredCoverageCannotBeForgedBySelectionLabel(t *testing.T) {
	f := newReceiptFixture(t)
	f.receiptSelection()["scope"] = "required"
	f.syncResult()
	f.receipt["review"] = "Ready"
	// The contract's only check covers required, so a missing auth reference is
	// what forbids Ready here; the focused/required label alone is not consent.
	in := f.input()
	in.AuthorizationRef = ""
	if err := f.validateWith(in); err == nil {
		t.Fatalf("Ready was accepted without reconciled authorization")
	}
}

func TestValidateSavedReceiptSurvivesReloadButChangedCodeBlocksReuse(t *testing.T) {
	f := newReceiptFixture(t)
	output := filepath.Join(f.evidence, "saved.json")
	if err := Export(marshalJSON(t, f.receipt), output, f.workspace, nil); err != nil {
		t.Fatalf("export: %v", err)
	}
	saved, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(saved, f.input()); err != nil {
		t.Fatalf("reloaded receipt rejected: %v", err)
	}

	in := f.input()
	in.AuthorizationRef = "new explicit authorization"
	if err := Validate(saved, in); err == nil {
		t.Fatalf("receipt was reused under a different authority")
	}

	for _, name := range []string{"code.txt", "tasks.md"} {
		path := filepath.Join(f.workspace, name)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		changed := append(append([]byte{}, original...), []byte("changed while session was absent\n")...)
		if err := os.WriteFile(path, changed, 0o644); err != nil {
			t.Fatal(err)
		}
		snapshot, err := Snapshot(f.workspace, []string{"."})
		if err != nil {
			t.Fatal(err)
		}
		current := f.identity
		current.BaseRevision = strp(snapshot.BaseRevision)
		current.Diff = strp(snapshot.Diff)
		stale := f.input()
		stale.Identity = current
		if err := Validate(saved, stale); err == nil {
			t.Fatalf("receipt was reused after %s changed", name)
		}
		present, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(present), "changed while session was absent") {
			t.Fatalf("validation modified dirty state: %v", err)
		}
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Validate(saved, f.input()); err != nil {
		t.Fatalf("restored workspace rejected: %v", err)
	}
}

func TestValidateWorkspaceMoveRequiresReconciliation(t *testing.T) {
	f := newReceiptFixture(t)
	saved := cloneValue(t, f.receipt).(map[string]any)
	moved := filepath.Join(f.root, "moved")
	if err := os.Rename(f.workspace, moved); err != nil {
		t.Fatal(err)
	}
	in := f.input()
	in.Workspace = moved
	if err := Validate(marshalJSON(t, saved), in); err == nil {
		t.Fatalf("moved workspace was accepted without reconciliation")
	}
	data, err := os.ReadFile(filepath.Join(moved, "tasks.md"))
	if err != nil || string(data) != "B1 awaiting human review\n" {
		t.Fatalf("workspace move verification disturbed the checkout: %v", err)
	}
}

func TestValidateRejectsForeignResultAndContract(t *testing.T) {
	t.Run("contract", func(t *testing.T) {
		f := newReceiptFixture(t)
		in := f.input()
		in.Contract = append(append([]byte{}, f.contract...), '\n')
		if err := f.validateWith(in); err == nil {
			t.Fatalf("contract mismatch was accepted")
		}
	})
	t.Run("result workspace", func(t *testing.T) {
		f := newReceiptFixture(t)
		f.result["workspace"] = "/somewhere/else"
		in := f.input()
		in.Result = marshalJSON(t, f.result)
		if err := f.validateWith(in); err == nil {
			t.Fatalf("foreign result workspace was accepted")
		}
	})
	t.Run("result shape", func(t *testing.T) {
		f := newReceiptFixture(t)
		in := f.input()
		in.Result = []byte(`{"version":1,"unknown":true}`)
		if err := f.validateWith(in); err == nil {
			t.Fatalf("unknown result field was accepted")
		}
	})
	t.Run("result absence", func(t *testing.T) {
		f := newReceiptFixture(t)
		in := f.input()
		in.Result = nil
		if err := f.validateWith(in); err == nil {
			t.Fatalf("missing result was accepted")
		}
	})
}

func TestValidateRejectsDirectFixTaskState(t *testing.T) {
	f := newReceiptFixture(t)
	f.receipt["workflow_state"] = "invented direct-fix task state"
	f.receiptUnknowns()["/workflow_state"] = "unused"
	if err := f.validate(); err == nil {
		t.Fatalf("direct fix invented canonical task state")
	}
}
