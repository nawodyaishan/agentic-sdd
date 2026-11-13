package main

// CLI regressions for Feature 017 execution-context integrity: checkpoint
// ownership, brief provenance registration, conservative schema-v1 reuse and
// explicit repeat/force. Every fixture is a disposable temporary repository;
// no test reads or writes a real home.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"agentic-sdd/internal/execution"
)

// rewriteExecutionCheckpointRunID replaces the checkpoint's run_id with another
// syntactically valid identifier, preserving the rest of the schema-v1 document.
func rewriteExecutionCheckpointRunID(t *testing.T, runDir, runID string) []byte {
	t.Helper()
	path := filepath.Join(runDir, "checkpoint.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	var checkpoint execution.Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		t.Fatalf("decode checkpoint: %v\n%s", err, data)
	}
	checkpoint.RunID = runID
	encoded, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		t.Fatalf("encode checkpoint: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("restrict checkpoint permissions: %v", err)
	}
	return encoded
}

func decodeExecutionBrief(t *testing.T, out string) execution.Brief {
	t.Helper()
	var brief execution.Brief
	if err := json.Unmarshal([]byte(out), &brief); err != nil {
		t.Fatalf("decode task brief: %v\n%s", err, out)
	}
	if brief.SchemaVersion != execution.SchemaVersion || brief.Kind != "execution.task.brief" {
		t.Fatalf("brief envelope = %+v", brief)
	}
	return brief
}

func decodeExecutionStatus(t *testing.T, out string) execution.Status {
	t.Helper()
	var status execution.Status
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("decode run status: %v\n%s", err, out)
	}
	if status.SchemaVersion != execution.SchemaVersion || status.Kind != "execution.run.status" {
		t.Fatalf("status envelope = %+v", status)
	}
	return status
}

func initExecutionIntegrityRun(t *testing.T, dir string) execution.RunIdentity {
	t.Helper()
	code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--format", "json")
	if code != 0 {
		t.Fatalf("run init exit %d: %s", code, errOut)
	}
	return decodeRunIdentity(t, out)
}

// TestExecutionIntegrityForeignCheckpointRefused covers AC-01: a checkpoint
// carrying another syntactically valid run ID is a conflict for status and for
// checkpoint-consuming mutation, with no normal status JSON and no writes.
func TestExecutionIntegrityForeignCheckpointRefused(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	const foreignID = "run-20261113T000000Z-abcdef"
	if identity.RunID == foreignID {
		t.Fatalf("fixture run ID unexpectedly equals %q", foreignID)
	}
	foreignCheckpoint := rewriteExecutionCheckpointRunID(t, identity.RunDir, foreignID)

	code, out, errOut := runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 2 {
		t.Fatalf("foreign checkpoint status exit %d, want 2: stdout %q stderr %q", code, out, errOut)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("foreign checkpoint status wrote a normal result: %s", out)
	}
	if !strings.Contains(strings.ToLower(errOut), "conflict") {
		t.Fatalf("foreign checkpoint status stderr does not identify a conflict: %q", errOut)
	}
	afterStatus, err := os.ReadFile(filepath.Join(identity.RunDir, "checkpoint.json"))
	if err != nil {
		t.Fatalf("read checkpoint after status: %v", err)
	}
	if !bytes.Equal(afterStatus, foreignCheckpoint) {
		t.Fatal("run status changed the foreign checkpoint")
	}

	briefPath := filepath.Join(identity.RunDir, "briefs", "T01.md")
	code, out, errOut = runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01")
	if code != 2 {
		t.Fatalf("foreign checkpoint task brief exit %d, want 2: stdout %q stderr %q", code, out, errOut)
	}
	if !strings.Contains(strings.ToLower(errOut), "conflict") {
		t.Fatalf("foreign checkpoint task brief stderr does not identify a conflict: %q", errOut)
	}
	if _, err := os.Stat(briefPath); !os.IsNotExist(err) {
		t.Fatalf("task brief created %s for a foreign checkpoint: %v", briefPath, err)
	}
	afterBrief, err := os.ReadFile(filepath.Join(identity.RunDir, "checkpoint.json"))
	if err != nil {
		t.Fatalf("read checkpoint after task brief: %v", err)
	}
	if !bytes.Equal(afterBrief, foreignCheckpoint) {
		t.Fatal("task brief changed the foreign checkpoint")
	}
}

// TestExecutionIntegrityDeletedRegisteredBrief covers AC-03/AC-04: a registered
// brief whose file disappears is stale reconciliation evidence, never verified
// evidence.
func TestExecutionIntegrityDeletedRegisteredBrief(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	code, out, errOut := runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01", "--format", "json")
	if code != 0 {
		t.Fatalf("task brief exit %d: %s", code, errOut)
	}
	brief := decodeExecutionBrief(t, out)
	wantPath := filepath.Join(identity.RunDir, "briefs", "T01.md")
	if filepath.Clean(brief.BriefPath) != wantPath {
		t.Fatalf("brief path = %q, want %q", brief.BriefPath, wantPath)
	}
	if brief.BriefDigest == "" {
		t.Fatalf("brief digest is empty: %+v", brief)
	}
	if err := os.Remove(brief.BriefPath); err != nil {
		t.Fatalf("delete registered brief: %v", err)
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateStale {
		t.Fatalf("state = %q, want stale; reasons = %v", status.State, status.StaleReasons)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a deleted brief still reported verified evidence")
	}
	if !slices.Contains(status.StaleReasons, execution.ReasonMissingArtifact) {
		t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonMissingArtifact)
	}
	if status.NextPermittedInvestigation != execution.InvestigateReconcileSource {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateReconcileSource)
	}
}

// TestExecutionIntegrityLegacyUnregisteredBrief covers AC-04: a schema-v1 run
// with a brief that has no registered provenance is stale and diagnosed, not
// silently adopted or migrated; a run with no brief artifacts stays clean.
func TestExecutionIntegrityLegacyUnregisteredBrief(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	legacyContent := "# Legacy brief\n\nWritten before brief provenance existed.\n"
	legacyPath := filepath.Join(identity.RunDir, "briefs", "T01.md")
	if err := os.WriteFile(legacyPath, []byte(legacyContent), 0o600); err != nil {
		t.Fatalf("write legacy brief: %v", err)
	}

	code, out, errOut := runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	if status.State != execution.StateStale {
		t.Fatalf("state = %q, want stale; reasons = %v", status.State, status.StaleReasons)
	}
	if !slices.Contains(status.StaleReasons, execution.ReasonUnregisteredBrief) {
		t.Fatalf("stale reasons = %v, want %q", status.StaleReasons, execution.ReasonUnregisteredBrief)
	}
	if status.NextPermittedInvestigation != execution.InvestigateReconcileSource {
		t.Fatalf("next permitted investigation = %q, want %q", status.NextPermittedInvestigation, execution.InvestigateReconcileSource)
	}
	if data, err := os.ReadFile(legacyPath); err != nil {
		t.Fatalf("legacy brief is gone after status: %v", err)
	} else if string(data) != legacyContent {
		t.Fatal("run status rewrote the unregistered legacy brief")
	}

	// A separate run with an empty briefs directory predates provenance without
	// becoming invalid.
	clean := initExecutionIntegrityRun(t, dir)
	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", clean.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("clean run status exit %d: %s", code, errOut)
	}
	cleanStatus := decodeExecutionStatus(t, out)
	if cleanStatus.State != execution.StatePrepared || len(cleanStatus.StaleReasons) != 0 {
		t.Fatalf("run without brief artifacts = %q with reasons %v, want prepared with none", cleanStatus.State, cleanStatus.StaleReasons)
	}
}

// TestExecutionIntegrityRepeatAndForce covers AC-03/AC-04: repeat creation is
// idempotent, tampered content is refused without --force, and an explicit
// force replacement clears the brief-specific staleness.
func TestExecutionIntegrityRepeatAndForce(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	briefPath := filepath.Join(identity.RunDir, "briefs", "T01.md")
	for attempt := 1; attempt <= 2; attempt++ {
		code, _, errOut := runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01", "--format", "json")
		if code != 0 {
			t.Fatalf("task brief attempt %d exit %d: %s", attempt, code, errOut)
		}
	}
	original, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("read brief: %v", err)
	}

	if err := os.WriteFile(briefPath, []byte("tampered brief\n"), 0o600); err != nil {
		t.Fatalf("tamper brief: %v", err)
	}
	code, _, errOut := runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01", "--format", "json")
	if code != 2 {
		t.Fatalf("task brief over tampered content exit %d, want 2: %s", code, errOut)
	}
	if tampered, err := os.ReadFile(briefPath); err != nil {
		t.Fatalf("read tampered brief: %v", err)
	} else if string(tampered) != "tampered brief\n" {
		t.Fatal("refused task brief changed the tampered artifact")
	}

	code, _, errOut = runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01", "--force", "--format", "json")
	if code != 0 {
		t.Fatalf("forced task brief exit %d: %s", code, errOut)
	}
	restored, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("read replaced brief: %v", err)
	}
	if !bytes.Equal(restored, original) {
		t.Fatal("forced task brief did not restore the deterministic brief content")
	}

	code, out, errOut := runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	status := decodeExecutionStatus(t, out)
	for _, reason := range status.StaleReasons {
		switch reason {
		case execution.ReasonMissingArtifact,
			execution.ReasonBriefDigestMismatch,
			execution.ReasonUnregisteredBrief,
			execution.ReasonBriefInspectionFailed:
			t.Fatalf("brief-related stale reason after forced replacement: %v", status.StaleReasons)
		}
	}
}
