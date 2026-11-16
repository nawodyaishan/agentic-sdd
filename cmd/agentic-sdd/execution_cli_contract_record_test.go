package main

// Feature 021 Batch B4 (T07–T08): G8 record validation, G10 flag edges and
// G11 adapter wiring for the execution CLI contract. No production code is
// touched; assertions are structural (exit codes, stream separation, JSON
// envelopes, artifact presence/absence), never golden files.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/execution"
)

// recordContractSetGitIsolation mirrors runExecutionCLI's hermetic Git
// environment for direct run(...) calls that carry a custom stdin reader.
func recordContractSetGitIsolation(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_OPTIONAL_LOCKS", "0")
}

// recordContractRecordFiles lists the raw record artifacts in a run.
func recordContractRecordFiles(t *testing.T, runDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(runDir, "records"))
	if err != nil {
		t.Fatalf("read records dir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

// recordContractAssertUsageError asserts the flag-edge contract: exit 2,
// empty stdout and a single "error:" line on stderr with no long usage block.
func recordContractAssertUsageError(t *testing.T, code int, stdout, stderr string) {
	t.Helper()
	if code != 2 {
		t.Fatalf("exit %d, want 2: stdout %q stderr %q", code, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty on a usage error", stdout)
	}
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "error:") {
		t.Fatalf("stderr = %q, want a single error: line", stderr)
	}
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr must not print the long usage block: %q", stderr)
	}
}

// recordContractCheckpointSnapshot captures the checkpoint bytes and check
// artifact list so flag-edge cases can prove they changed nothing.
func recordContractCheckpointSnapshot(t *testing.T, runDir string) ([]byte, []string) {
	t.Helper()
	checkpoint, err := os.ReadFile(filepath.Join(runDir, "checkpoint.json"))
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(runDir, "checks"))
	if err != nil {
		t.Fatalf("read checks dir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return checkpoint, names
}

func recordContractAssertUnchanged(t *testing.T, runDir string, beforeCheckpoint []byte, beforeChecks []string) {
	t.Helper()
	afterCheckpoint, afterChecks := recordContractCheckpointSnapshot(t, runDir)
	if !bytes.Equal(afterCheckpoint, beforeCheckpoint) {
		t.Fatal("usage error changed checkpoint.json")
	}
	if strings.Join(afterChecks, "\x00") != strings.Join(beforeChecks, "\x00") {
		t.Fatalf("usage error changed check artifacts: before %v after %v", beforeChecks, afterChecks)
	}
}

// TestExecutionContractRecordValidation covers G8: strict stdin record
// ingestion. Unknown fields and oversize input exit 2 with empty stdout, an
// "error:" stderr and no record artifact; a valid stdin payload is persisted
// verbatim with its digest.
func TestExecutionContractRecordValidation(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	recordContractSetGitIsolation(t)
	invokeStdin := func(payload string, extra ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		args := []string{"run", "record", "--repo", dir, "--run", identity.RunID, "--task", "T01", "--result", "-", "--format", "json"}
		args = append(args, extra...)
		code := run(args, &stdout, &stderr, strings.NewReader(payload), false)
		return code, stdout.String(), stderr.String()
	}

	t.Run("unknown field", func(t *testing.T) {
		before := recordContractRecordFiles(t, identity.RunDir)
		payload := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture","bogus":1}`
		code, stdout, stderr := invokeStdin(payload)
		recordContractAssertUsageError(t, code, stdout, stderr)
		if after := recordContractRecordFiles(t, identity.RunDir); len(after) != len(before) {
			t.Fatalf("invalid record wrote an artifact: before %v after %v", before, after)
		}
	})

	t.Run("oversize", func(t *testing.T) {
		before := recordContractRecordFiles(t, identity.RunDir)
		payload := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"` + strings.Repeat("a", 1<<20) + `"}`
		code, stdout, stderr := invokeStdin(payload)
		recordContractAssertUsageError(t, code, stdout, stderr)
		if after := recordContractRecordFiles(t, identity.RunDir); len(after) != len(before) {
			t.Fatalf("oversize record wrote an artifact: before %v after %v", before, after)
		}
	})

	t.Run("stdin success", func(t *testing.T) {
		payload := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture report","unknowns":["not independently verified"]}`
		code, stdout, stderr := invokeStdin(payload)
		if code != 0 {
			t.Fatalf("run record via stdin exit %d: %s", code, stderr)
		}
		var envelope struct {
			SchemaVersion int                   `json:"schemaVersion"`
			Kind          string                `json:"kind"`
			Record        execution.ClaimRecord `json:"record"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("decode claim: %v\n%s", err, stdout)
		}
		if envelope.SchemaVersion != execution.SchemaVersion || envelope.Kind != "execution.run.record" {
			t.Fatalf("claim envelope = schemaVersion %d kind %q", envelope.SchemaVersion, envelope.Kind)
		}
		claim := envelope.Record
		if claim.RecordID == "" || claim.TaskID != "T01" || claim.RawPath == "" {
			t.Fatalf("claim = %+v", claim)
		}
		raw, err := os.ReadFile(claim.RawPath)
		if err != nil {
			t.Fatalf("read raw claim: %v", err)
		}
		if string(raw) != payload {
			t.Fatalf("raw claim = %q, want the verbatim payload", raw)
		}
		if claim.RawDigest == "" || claim.RawDigest != execution.DigestBytes([]byte(payload)) {
			t.Fatalf("claim digest = %q, want the payload digest", claim.RawDigest)
		}
		if len(recordContractRecordFiles(t, identity.RunDir)) == 0 {
			t.Fatal("successful stdin record left no artifact")
		}
	})
}

// TestExecutionContractFlagEdges covers G10: run check without the "--"
// separator, with an empty command vector, and with a negative timeout all
// exit 2 with empty stdout, a single "error:" stderr line and no artifact
// change.
func TestExecutionContractFlagEdges(t *testing.T) {
	dir := executionTestRepo(t)
	identity := initExecutionIntegrityRun(t, dir)

	t.Run("missing separator", func(t *testing.T) {
		beforeCheckpoint, beforeChecks := recordContractCheckpointSnapshot(t, identity.RunDir)
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--format", "json")
		recordContractAssertUsageError(t, code, stdout, stderr)
		recordContractAssertUnchanged(t, identity.RunDir, beforeCheckpoint, beforeChecks)
	})

	t.Run("empty vector", func(t *testing.T) {
		beforeCheckpoint, beforeChecks := recordContractCheckpointSnapshot(t, identity.RunDir)
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--format", "json", "--")
		recordContractAssertUsageError(t, code, stdout, stderr)
		recordContractAssertUnchanged(t, identity.RunDir, beforeCheckpoint, beforeChecks)
	})

	t.Run("negative timeout", func(t *testing.T) {
		beforeCheckpoint, beforeChecks := recordContractCheckpointSnapshot(t, identity.RunDir)
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--timeout", "-1s", "--format", "json", "--", "sh", "-c", "exit 0")
		recordContractAssertUsageError(t, code, stdout, stderr)
		recordContractAssertUnchanged(t, identity.RunDir, beforeCheckpoint, beforeChecks)
	})
}

// TestExecutionContractAdapterWiring covers G11: --direct conflicts, explicit
// --run-root visibility, feature vs direct --task defaults, invalid review
// base and review cap flags.
func TestExecutionContractAdapterWiring(t *testing.T) {
	dir := executionTestRepo(t)

	t.Run("direct conflicts with feature batch", func(t *testing.T) {
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "init", "--direct", "--feature", "feature", "--batch", "B1")
		if code != 2 {
			t.Fatalf("conflicting init exit %d, want 2: stdout %q stderr %q", code, stdout, stderr)
		}
		if !strings.Contains(stderr, "error:") || !strings.Contains(stderr, "--direct") {
			t.Fatalf("conflicting init stderr = %q, want the --direct conflict", stderr)
		}
	})

	t.Run("explicit run root outside the repository", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "runs")
		code, stdout, stderr := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--run-root", root, "--format", "json")
		if code != 0 {
			t.Fatalf("explicit run root init exit %d: %s", code, stderr)
		}
		identity := decodeRunIdentity(t, stdout)
		if filepath.Clean(identity.RunRoot) != filepath.Clean(root) {
			t.Fatalf("run root = %q, want %q", identity.RunRoot, root)
		}
		if identity.WorkspaceGitVisible {
			t.Fatal("an outside-repository run root must not be Git-visible")
		}
	})

	t.Run("explicit run root inside the repository is reported visible", func(t *testing.T) {
		insideDir := executionTestRepo(t)
		scratch := filepath.Join(insideDir, "scratch-runs")
		code, stdout, stderr := runExecutionCLI(t, insideDir, "run", "init", "--feature", "feature", "--batch", "B1", "--run-root", scratch)
		if code != 0 {
			t.Fatalf("inside-repository run root init exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "workspace git visible: yes") {
			t.Fatalf("stdout does not surface Git visibility:\n%s", stdout)
		}
	})

	t.Run("feature record requires task, direct defaults", func(t *testing.T) {
		feature := initExecutionIntegrityRun(t, dir)
		payload := `{"schemaVersion":1,"taskId":"direct","result":"passed","summary":"fixture"}`
		recordPath := filepath.Join(t.TempDir(), "record.json")
		if err := os.WriteFile(recordPath, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}

		code, _, stderr := runExecutionCLI(t, dir, "run", "record", "--run", feature.RunID, "--result", recordPath, "--format", "json")
		if code != 2 || !strings.Contains(stderr, "--task") {
			t.Fatalf("feature record without --task exit %d, want 2 naming --task: %s", code, stderr)
		}

		code, stdout, stderr := runExecutionCLI(t, dir, "run", "init", "--direct", "--label", "fixture-direct", "--format", "json")
		if code != 0 {
			t.Fatalf("direct init exit %d: %s", code, stderr)
		}
		direct := decodeRunIdentity(t, stdout)
		code, stdout, stderr = runExecutionCLI(t, dir, "run", "record", "--run", direct.RunID, "--result", recordPath, "--format", "json")
		if code != 0 {
			t.Fatalf("direct record without --task exit %d: %s", code, stderr)
		}
		var envelope struct {
			SchemaVersion int                   `json:"schemaVersion"`
			Kind          string                `json:"kind"`
			Record        execution.ClaimRecord `json:"record"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("decode direct claim: %v\n%s", err, stdout)
		}
		if envelope.Record.TaskID != "direct" {
			t.Fatalf("direct record task = %q, want direct", envelope.Record.TaskID)
		}
	})

	t.Run("feature check requires task, direct defaults", func(t *testing.T) {
		shPath, err := exec.LookPath("sh")
		if err != nil {
			t.Skip("not_run: sh unavailable")
		}
		feature := initExecutionIntegrityRun(t, dir)
		code, _, stderr := runExecutionCLI(t, dir, "run", "check", "--run", feature.RunID, "--format", "json", "--", shPath, "-c", "exit 0")
		if code != 2 || !strings.Contains(stderr, "--task") {
			t.Fatalf("feature check without --task exit %d, want 2 naming --task: %s", code, stderr)
		}

		code, stdout, stderr := runExecutionCLI(t, dir, "run", "init", "--direct", "--label", "fixture-check", "--format", "json")
		if code != 0 {
			t.Fatalf("direct init exit %d: %s", code, stderr)
		}
		direct := decodeRunIdentity(t, stdout)
		code, stdout, stderr = runExecutionCLI(t, dir, "run", "check", "--run", direct.RunID, "--format", "json", "--", shPath, "-c", "exit 0")
		if code != 0 {
			t.Fatalf("direct check without --task exit %d: %s", code, stderr)
		}
		var envelope struct {
			Check execution.CheckRecord `json:"check"`
		}
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatalf("decode direct check: %v\n%s", err, stdout)
		}
		if envelope.Check.TaskID != "direct" {
			t.Fatalf("direct check task = %q, want direct", envelope.Check.TaskID)
		}
	})

	t.Run("invalid review base", func(t *testing.T) {
		identity := initExecutionIntegrityRun(t, dir)
		code, stdout, stderr := runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "--format", "json")
		if code != 2 {
			t.Fatalf("invalid base exit %d, want 2: stdout %q stderr %q", code, stdout, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Fatalf("invalid base wrote stdout: %q", stdout)
		}
		if !strings.Contains(stderr, "error:") {
			t.Fatalf("invalid base stderr = %q, want an error: line", stderr)
		}
	})

	t.Run("review cap flags", func(t *testing.T) {
		identity := initExecutionIntegrityRun(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "cap-fixture.txt"), []byte(strings.Repeat("x", 4096)+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		base := executionGitOutput(t, dir, "rev-parse", "HEAD")
		code, stdout, stderr := runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", base, "--max-file-bytes", "10", "--max-total-bytes", "10", "--format", "json")
		if code != 3 {
			t.Fatalf("capped review package exit %d, want 3: stdout %q stderr %q", code, stdout, stderr)
		}
		var manifest execution.ReviewManifest
		if err := json.Unmarshal([]byte(stdout), &manifest); err != nil {
			t.Fatalf("decode capped manifest: %v\n%s", err, stdout)
		}
		if manifest.Complete {
			t.Fatal("capped review package must not claim complete")
		}
		if manifest.SchemaVersion != execution.SchemaVersion || manifest.Kind != "execution.review.package" {
			t.Fatalf("manifest envelope = schemaVersion %d kind %q", manifest.SchemaVersion, manifest.Kind)
		}
		capped := false
		for _, reason := range manifest.PartialReasons {
			if reason == "max_file_bytes_exceeded" || reason == "max_total_bytes_exceeded" {
				capped = true
			}
		}
		if !capped {
			t.Fatalf("partial reasons = %v, want a cap reason", manifest.PartialReasons)
		}
	})
}
