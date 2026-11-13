package main

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

// executionTestRepo creates a disposable Git repository with a minimal feature
// fixture. Tests never touch a real home or the repository's own checkout.
func executionTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL="+os.DevNull,
			"GIT_CONFIG_SYSTEM="+os.DevNull,
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q", ".")
	if err := os.MkdirAll(filepath.Join(dir, "feature"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `# Fixture spec

### FR-01 — Example requirement

The fixture must produce a deterministic brief.

## Acceptance

| ID | Condition |
|---|---|
| **AC-01** | The brief contains the selected task. |
`
	tasks := `# Fixture tasks

## B1 — Example batch

**Outcome:** One fixture task.

### T01 — First task

- Work only in the fixture.
- Verify: ` + "`true`" + `.

### T02 — Unrelated task

- This must not appear in the T01 brief.
`
	if err := os.WriteFile(filepath.Join(dir, "feature", "spec.md"), []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature", "tasks.md"), []byte(tasks), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", ".")
	runGit("commit", "-qm", "fixture")
	return dir
}

func runExecutionCLI(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_OPTIONAL_LOCKS", "0")
	var stdout, stderr bytes.Buffer
	args = append([]string(nil), args...)
	insertAt := len(args)
	for index, arg := range args {
		if arg == "--" {
			insertAt = index
			break
		}
	}
	args = append(args[:insertAt], append([]string{"--repo", dir}, args[insertAt:]...)...)
	code := runNonInteractive(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decodeRunIdentity(t *testing.T, out string) execution.RunIdentity {
	t.Helper()
	var identity execution.RunIdentity
	if err := json.Unmarshal([]byte(out), &identity); err != nil {
		t.Fatalf("decode run identity: %v\n%s", err, out)
	}
	if identity.SchemaVersion != execution.SchemaVersion {
		t.Fatalf("schemaVersion = %d, want %d", identity.SchemaVersion, execution.SchemaVersion)
	}
	if identity.RunID == "" || identity.RunDir == "" {
		t.Fatalf("incomplete identity: %+v", identity)
	}
	return identity
}

func executionGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_OPTIONAL_LOCKS=0",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func TestExecutionCLIInitBriefStatusRecordClean(t *testing.T) {
	dir := executionTestRepo(t)
	code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--feature", "feature", "--batch", "B1", "--format", "json")
	if code != 0 {
		t.Fatalf("run init exit %d: %s", code, errOut)
	}
	identity := decodeRunIdentity(t, out)
	if identity.Scope.Kind != execution.ScopeFeature || identity.Scope.BatchID != "B1" {
		t.Fatalf("scope = %+v", identity.Scope)
	}
	if identity.WorkspaceGitVisible {
		t.Fatal("default repository-local run root should be Git-invisible")
	}

	code, out, errOut = runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01")
	if code != 0 {
		t.Fatalf("task brief exit %d: %s", code, errOut)
	}
	if !strings.Contains(out, "task: T01") {
		t.Fatalf("task brief output missing task: %s", out)
	}
	brief, err := os.ReadFile(filepath.Join(identity.RunDir, "briefs", "T01.md"))
	if err != nil {
		t.Fatalf("read brief: %v", err)
	}
	if !strings.Contains(string(brief), "First task") || strings.Contains(string(brief), "Unrelated task") {
		t.Fatalf("brief selected the wrong task section:\n%s", brief)
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "check", "--run", identity.RunID, "--task", "T01", "--format", "json", "--", "/bin/sh", "-c", "exit 0")
	if code != 0 {
		t.Fatalf("run check pass exit %d: %s", code, errOut)
	}
	var checkEnvelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Check         execution.CheckRecord `json:"check"`
	}
	if err := json.Unmarshal([]byte(out), &checkEnvelope); err != nil {
		t.Fatalf("decode check: %v\n%s", err, out)
	}
	if checkEnvelope.SchemaVersion != execution.SchemaVersion || checkEnvelope.Kind != "execution.run.check" {
		t.Fatalf("check envelope = %+v", checkEnvelope)
	}
	check := checkEnvelope.Check
	if check.Result != execution.CheckPassed {
		t.Fatalf("check result = %q, want passed", check.Result)
	}
	if check.StdoutPath == "" || check.StderrPath == "" {
		t.Fatalf("check log paths missing: %+v", check)
	}

	base := executionGitOutput(t, dir, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", base, "--format", "json")
	if code != 0 {
		t.Fatalf("review package exit %d: %s", code, errOut)
	}
	var manifest execution.ReviewManifest
	if err := json.Unmarshal([]byte(out), &manifest); err != nil {
		t.Fatalf("decode review manifest: %v\n%s", err, out)
	}
	if !manifest.Complete || manifest.SchemaVersion != execution.SchemaVersion || manifest.Kind != "execution.review.package" {
		t.Fatalf("review manifest = %+v", manifest)
	}
	foundNew := false
	for _, file := range manifest.Files {
		if file.Path == "new.txt" {
			foundNew = true
		}
	}
	if !foundNew {
		t.Fatalf("review package did not include new.txt: %+v", manifest.Files)
	}

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runExecutionCLI(t, dir, "review", "package", "--run", identity.RunID, "--base", base, "--format", "json")
	if code != 3 {
		t.Fatalf("sensitive review package exit %d, want 3: %s", code, errOut)
	}
	if err := json.Unmarshal([]byte(out), &manifest); err != nil {
		t.Fatalf("decode partial review manifest: %v\n%s", err, out)
	}
	if manifest.Complete || len(manifest.PartialReasons) == 0 {
		t.Fatalf("sensitive review package should be partial: %+v", manifest)
	}
	foundSensitive := false
	for _, reason := range manifest.PartialReasons {
		if reason == "sensitive_path" {
			foundSensitive = true
		}
	}
	if !foundSensitive {
		t.Fatalf("partial reasons = %v, want sensitive_path", manifest.PartialReasons)
	}

	recordPath := filepath.Join(t.TempDir(), "record.json")
	record := `{"schemaVersion":1,"taskId":"T01","result":"passed","summary":"fixture report","changedPaths":[],"checks":[],"blockers":[],"unknowns":["not independently verified"]}`
	if err := os.WriteFile(recordPath, []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runExecutionCLI(t, dir, "run", "record", "--run", identity.RunID, "--task", "T01", "--result", recordPath, "--format", "json")
	if code != 0 {
		t.Fatalf("run record exit %d: %s", code, errOut)
	}
	var claimEnvelope struct {
		SchemaVersion int                   `json:"schemaVersion"`
		Kind          string                `json:"kind"`
		Record        execution.ClaimRecord `json:"record"`
	}
	if err := json.Unmarshal([]byte(out), &claimEnvelope); err != nil {
		t.Fatalf("decode claim: %v\n%s", err, out)
	}
	if claimEnvelope.SchemaVersion != execution.SchemaVersion || claimEnvelope.Kind != "execution.run.record" {
		t.Fatalf("claim envelope = %+v", claimEnvelope)
	}
	claim := claimEnvelope.Record
	if claim.Result != execution.ClaimPassed || claim.RawPath == "" {
		t.Fatalf("claim = %+v", claim)
	}

	code, out, errOut = runExecutionCLI(t, dir, "run", "status", "--run", identity.RunID, "--format", "json")
	if code != 0 {
		t.Fatalf("run status exit %d: %s", code, errOut)
	}
	var status execution.Status
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("decode status: %v\n%s", err, out)
	}
	if status.State != execution.StateAwaitingVerification {
		t.Fatalf("state = %q, want awaiting-verification", status.State)
	}
	if len(status.Records) != 1 || status.Records[0].RecordID != claim.RecordID {
		t.Fatalf("status records = %+v", status.Records)
	}
	if status.State == execution.StateVerifiedEvidence {
		t.Fatal("a record claim became verified evidence")
	}

	code, _, errOut = runExecutionCLI(t, dir, "run", "clean", "--run", identity.RunID)
	if code != 0 {
		t.Fatalf("run clean preview exit %d: %s", code, errOut)
	}
	if _, err := os.Stat(identity.RunDir); err != nil {
		t.Fatalf("clean preview removed the run: %v", err)
	}
	code, _, errOut = runExecutionCLI(t, dir, "run", "clean", "--run", identity.RunID, "--yes")
	if code != 0 {
		t.Fatalf("run clean apply exit %d: %s", code, errOut)
	}
	if _, err := os.Stat(identity.RunDir); !os.IsNotExist(err) {
		t.Fatalf("run directory still exists after clean: %v", err)
	}
}

func TestExecutionCLIDirectScopeAndRefusals(t *testing.T) {
	dir := executionTestRepo(t)
	code, out, errOut := runExecutionCLI(t, dir, "run", "init", "--direct", "--label", "fixture-direct", "--format", "json")
	if code != 0 {
		t.Fatalf("run init --direct exit %d: %s", code, errOut)
	}
	identity := decodeRunIdentity(t, out)
	if identity.Scope.Kind != execution.ScopeDirect {
		t.Fatalf("scope kind = %q, want direct", identity.Scope.Kind)
	}
	code, _, errOut = runExecutionCLI(t, dir, "task", "brief", "--run", identity.RunID, "--task", "T01")
	if code != 2 || !strings.Contains(errOut, "direct-scope") {
		t.Fatalf("direct task brief = exit %d, stderr %q", code, errOut)
	}

	code, _, errOut = runExecutionCLI(t, dir, "run", "init", "--feature", "feature")
	if code != 2 {
		t.Fatalf("run init without batch exit %d: %s", code, errOut)
	}
	code, _, errOut = runExecutionCLI(t, dir, "run", "status", "--run", "run-20260101T000000Z-abcdef")
	if code != 1 {
		t.Fatalf("missing run status exit %d: %s", code, errOut)
	}
	code, _, errOut = runExecutionCLI(t, dir, "run", "check", "--run", "run-20260101T000000Z-abcdef", "--task", "T01", "--", "/bin/sh", "-c", "exit 0")
	if code != 1 {
		t.Fatalf("missing run check exit %d: %s", code, errOut)
	}
	code, _, errOut = runExecutionCLI(t, dir, "task", "brief", "--run", "run-20260101T000000Z-abcdef", "--task", "T01")
	if code != 1 {
		t.Fatalf("missing run task brief exit %d: %s", code, errOut)
	}
	code, _, errOut = runExecutionCLI(t, dir, "run", "status", "--run", "not-a-run")
	if code != 2 {
		t.Fatalf("invalid run ID exit %d: %s", code, errOut)
	}
	code, _, errOut = runExecutionCLI(t, dir, "run", "record", "--run", identity.RunID, "--result", filepath.Join(dir, "missing.json"))
	if code != 2 {
		t.Fatalf("missing record file exit %d: %s", code, errOut)
	}
}

func TestExecutionCLIHelpAndFormat(t *testing.T) {
	for _, topic := range []string{"run", "task", "review"} {
		var stdout, stderr bytes.Buffer
		code := runNonInteractive([]string{"help", topic}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("help %s exit %d: %s", topic, code, stderr.String())
		}
		got := stdout.String()
		if !strings.Contains(got, "Writes files:") {
			t.Fatalf("help %s missing writes note:\n%s", topic, got)
		}
		for _, want := range []string{"Example: asdx-core " + topic, "Related: asdx-core help "} {
			if !strings.Contains(got, want) {
				t.Fatalf("help %s missing %q:\n%s", topic, want, got)
			}
		}
		if strings.Contains(got, "agentic-sdd "+topic) {
			t.Fatalf("help %s still uses legacy command in example:\n%s", topic, got)
		}
	}

	t.Run("task help and brief help explain Feature 017 contract", func(t *testing.T) {
		for _, args := range [][]string{{"help", "task"}, {"task", "help"}, {"task", "brief", "--help"}} {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("%v exit %d: %s", args, code, stderr.String())
			}
			got := stdout.String()
			for _, want := range []string{
				"asdx-core task brief --run ID --task T01 [--force]",
				"owning run",
				"checkpoint.json",
				"digest",
				"--force",
				"--direct",
				"observation",
				"never",
				"approval",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("%v missing %q:\n%s", args, want, got)
				}
			}
		}
	})

	t.Run("run and review help use asdx-core", func(t *testing.T) {
		cases := []struct {
			args []string
			want []string
		}{
			{[]string{"run", "help"}, []string{"Usage: asdx-core run init", "asdx-core run status", "asdx-core run check", `asdx-core help run`, `"agentic-sdd" wraps "asdx-core"`}},
			{[]string{"run", "init", "--help"}, []string{"Usage: asdx-core run init"}},
			{[]string{"review", "help"}, []string{"Usage: asdx-core review package", "asdx-core review help", "exits 3", `"agentic-sdd" wraps "asdx-core"`}},
			{[]string{"review", "package", "--help"}, []string{"Usage: asdx-core review package"}},
		}
		for _, tc := range cases {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("%v exit %d: %s", tc.args, code, stderr.String())
			}
			got := stdout.String()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("%v missing %q:\n%s", tc.args, want, got)
				}
			}
			if strings.Contains(got, "Usage: agentic-sdd") {
				t.Fatalf("%v still prints legacy usage:\n%s", tc.args, got)
			}
		}
	})

	t.Run("flag errors reference asdx-core help", func(t *testing.T) {
		for _, args := range [][]string{
			{"task", "brief", "--wat"},
			{"run", "init", "--wat"},
			{"review", "package", "--wat"},
		} {
			var stdout, stderr bytes.Buffer
			code := runNonInteractive(args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("%v exit %d, want 2", args, code)
			}
			if !strings.Contains(stderr.String(), `note: run "asdx-core help" for the accepted flags`) {
				t.Fatalf("%v stderr missing asdx-core help note: %q", args, stderr.String())
			}
		}
	})

	dir := executionTestRepo(t)
	var initOut, initErr bytes.Buffer
	if code := runNonInteractive([]string{"run", "init", "--repo", dir, "--feature", "feature", "--batch", "B1"}, &initOut, &initErr); code != 0 {
		t.Fatalf("text run init exit %d: %s", code, initErr.String())
	}
	if !strings.Contains(initOut.String(), "next: asdx-core task brief --run ") {
		t.Fatalf("feature run init next line missing asdx-core:\n%s", initOut.String())
	}
	var directOut, directErr bytes.Buffer
	if code := runNonInteractive([]string{"run", "init", "--repo", dir, "--direct", "--label", "fix"}, &directOut, &directErr); code != 0 {
		t.Fatalf("direct text run init exit %d: %s", code, directErr.String())
	}
	if !strings.Contains(directOut.String(), "next: asdx-core run record --run ") {
		t.Fatalf("direct run init next line missing asdx-core:\n%s", directOut.String())
	}

	code, _, errOut := runExecutionCLI(t, dir, "run", "init", "--direct", "--format", "yaml")
	if code != 2 {
		t.Fatalf("invalid format exit %d: %s", code, errOut)
	}
}
