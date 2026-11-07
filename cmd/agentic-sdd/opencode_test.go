//go:build agentic_sdd_experimental

package main

import (
	"agentic-sdd/internal/harnesscheck"
	"agentic-sdd/internal/harnessevidence"
	"agentic-sdd/internal/opencodeadapter"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const fixtureCanaryValue = "SYNTHETIC-SECRET-CANARY-DO-NOT-EXPORT-12345"

func ptr(s string) *string { return &s }

func TestOpenCodeCommandsStayOfflineAndDefaultToPreview(t *testing.T) {
	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "candidate", "--feature", "demo", "--build-path", "src"}, &out, &errout); code != 0 || !strings.Contains(out.String(), "openrouter/nvidia/nemotron") {
		t.Fatalf("candidate: %d %s", code, errout.String())
	}
	root := t.TempDir()
	staged := filepath.Join(t.TempDir(), "candidate.jsonc")
	if err := os.WriteFile(staged, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "config", "--workspace", root, "--candidate", staged}, &out, &errout); code != 0 || !strings.Contains(out.String(), `"original":"absent"`) {
		t.Fatalf("default preview: %d %s", code, errout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".opencode")); !os.IsNotExist(err) {
		t.Fatal("preview installed")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "session", "--auto"}, &out, &errout); code != 2 {
		t.Fatal("accepted unfinished/unsafe native operation")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "apply", "--help"}, &out, &errout); code != 2 {
		t.Fatal("accepted unnamespaced config operation")
	}
}

func TestOpenCodeApplyFailureRetainsActionableBackupObservation(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-denial fixture requires unprivileged user")
	}
	root := t.TempDir()
	dir := filepath.Join(root, ".opencode")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(`{"unrelated":"preserve"}`), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := opencodeadapter.Candidate("demo", "src")
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(t.TempDir(), "candidate.jsonc")
	if err := os.WriteFile(staged, data, 0600); err != nil {
		t.Fatal(err)
	}
	p, err := opencodeadapter.Preview(root, staged)
	if err != nil {
		t.Fatal(err)
	}
	backup := t.TempDir()
	if err := os.Chmod(backup, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	var out, errs bytes.Buffer
	code := runNonInteractive([]string{"opencode", "config", "apply", "--workspace", root, "--candidate", staged, "--backup-root", backup, "--expect-original", p.Original, "--expect-candidate", p.Candidate}, &out, &errs)
	var observation struct {
		Execution   string                      `json:"execution"`
		Transaction opencodeadapter.Transaction `json:"transaction"`
	}
	if code != 2 || json.Unmarshal(out.Bytes(), &observation) != nil || observation.Execution != "incomplete" || observation.Transaction.Backup == "" {
		t.Fatalf("lost partial observation: %d %s %s", code, out.String(), errs.String())
	}
}

func TestOpenCodeSessionPrepareStaysOffline(t *testing.T) {
	root := t.TempDir()
	var out, errout bytes.Buffer
	code := runNonInteractive([]string{"opencode", "session", "prepare",
		"--workspace", root, "--executable", "/usr/bin/opencode",
		"--model", "openrouter/nvidia/nemotron-3-ultra-550b-a55b:free", "--mode", "primary"}, &out, &errout)
	if code != 0 {
		t.Fatalf("prepare: %d %s", code, errout.String())
	}
	var r struct {
		Execution string   `json:"execution"`
		Blockers  []string `json:"blockers"`
	}
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Execution != "prepared" || len(r.Blockers) == 0 {
		t.Fatalf("prepare observation missing blockers: %s", out.String())
	}
}

func TestOpenCodeSessionRunIsAdmissionBlocked(t *testing.T) {
	root := t.TempDir()
	var out, errout bytes.Buffer
	code := runNonInteractive([]string{"opencode", "session", "run",
		"--workspace", root, "--executable", "/usr/bin/opencode", "--prompt", "hello"}, &out, &errout)
	if code != 0 {
		t.Fatalf("run: %d %s", code, errout.String())
	}
	var r struct {
		Execution string   `json:"execution"`
		Blockers  []string `json:"blockers"`
	}
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Execution != "blocked" || len(r.Blockers) == 0 {
		t.Fatalf("run did not fail closed to blocked: %s", out.String())
	}
}

func TestOpenCodeSessionPendingReviewsWithoutReentry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s := opencodeadapter.PendingSummary{
		Version: 1, Run: "pending_human", Session: "ses_test", Workspace: "/tmp/ws",
		Request: "native pending human input", Question: "Which package manager?",
		Rationale: "unknown", Scope: "answer required", NextAction: "human supplies the answer",
	}
	path, err := opencodeadapter.WritePending(dir, s)
	if err != nil {
		t.Fatal(err)
	}
	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "session", "pending", "--file", path}, &out, &errout); code != 0 {
		t.Fatalf("pending: %d %s", code, errout.String())
	}
	var r struct {
		Execution string                         `json:"execution"`
		Digest    string                         `json:"digest"`
		Pending   opencodeadapter.PendingSummary `json:"pending"`
	}
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Execution != "review" || r.Digest == "" || r.Pending.Question != "Which package manager?" {
		t.Fatalf("pending review mismatch: %s", out.String())
	}
}

func TestOpenCodeSessionUnknownOperationRefused(t *testing.T) {
	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "session", "resume"}, &out, &errout); code != 2 {
		t.Fatal("accepted unavailable session operation")
	}
}

func TestOpenCodeEvidenceSnapshotRequiresExplicitSafeScope(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.name", "Synthetic Fixture")
	git("config", "user.email", "fixture@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "code.txt"), []byte("original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-qm", "fixture")

	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "evidence", "snapshot", "--workspace", repo, "--scope", "."}, &out, &errout); code != 0 || !strings.Contains(out.String(), "sha256-v1:") {
		t.Fatalf("snapshot: %d %s %s", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "evidence", "snapshot", "--workspace", repo, "--scope", "../outside"}, &out, &errout); code != 2 {
		t.Fatal("unsafe scope was accepted")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "evidence", "snapshot", "--workspace", repo}, &out, &errout); code != 2 {
		t.Fatal("snapshot without a scope was accepted")
	}
}

func TestOpenCodeEvidenceReceiptValidateExportRoundTrip(t *testing.T) {
	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	evidence := t.TempDir()
	if err := os.Chmod(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(evidence, "test.log")
	logData := []byte("independent synthetic assertion passed\n")
	if err := os.WriteFile(logPath, logData, 0600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"version":1,"checks":[{"id":"test","cwd":".","scopes":["baseline","focused","required"],"prerequisites":[],"timeout_seconds":30,"effects":{"writes":["fixture"],"network":"none"},"steps":[{"argv":["true"],"env":{}}],"local_recipe":["true"],"expected_exit":0,"evidence":"test.log"}]}`)
	contractSum := sha256.Sum256(contract)
	contractDigest := hex.EncodeToString(contractSum[:])

	exit := 0
	start := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	result := harnesscheck.Result{
		Version:        1,
		Workspace:      workspace,
		ContractSHA256: contractDigest,
		Selection:      harnesscheck.Selection{Scope: "required", CheckIDs: []string{"test"}},
		Execution:      "completed",
		Checks: []harnesscheck.CheckResult{{
			ID: "test", Status: "passed", ExitCode: &exit,
			StartedAt: &start, EndedAt: &end, Evidence: &logPath,
			Reason: "process exited zero",
		}},
	}
	scopeIdentity := harnessevidence.ScopeIdentity{BaseRevision: "base-revision", Diff: "sha256-v1:" + strings.Repeat("a", 64)}
	context := opencodeadapter.ReceiptContext{
		Synthetic: true, RunID: "fixture-run", Mode: "build",
		RequestRef:       "synthetic request " + fixtureCanaryValue,
		AuthorizationRef: ptr("fixture authorization"),
		Feature:          ptr("005-vnext-completion"), Batch: ptr("B4"), TaskIDs: []string{"T10"},
		AllowedEffects: []string{"fixture"}, Profile: ptr("fixture-profile"), Tools: []string{},
		Limits: harnessevidence.Limits{DeadlineSeconds: 60, Steps: 5},
		Identity: opencodeadapter.ContextIdentity{
			Config: ptr("fixture-config"), Skills: ptr("fixture-skills"),
			Environment: ptr("fixture-environment"), Runtime: ptr("fixture-runtime"),
			Harness: ptr("fixture-harness"), Contract: ptr(contractDigest),
		},
		IdentityBefore: scopeIdentity,
		IdentityAfter:  scopeIdentity,
		NextAction:     "human reviews the batch",
	}
	identity := harnessevidence.Identity{
		BaseRevision: ptr(scopeIdentity.BaseRevision), Diff: ptr(scopeIdentity.Diff),
		Config: context.Identity.Config, Skills: context.Identity.Skills,
		Environment: context.Identity.Environment, Runtime: context.Identity.Runtime,
		Harness: context.Identity.Harness, Contract: context.Identity.Contract,
	}

	private := t.TempDir()
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	writeJSON := func(name string, value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(private, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	contextPath := writeJSON("context.json", context)
	resultPath := writeJSON("result.json", result)
	identityPath := writeJSON("identity.json", identity)
	contractPath := filepath.Join(private, "contract.json")
	if err := os.WriteFile(contractPath, contract, 0600); err != nil {
		t.Fatal(err)
	}
	redactionPath := filepath.Join(private, "redactions.txt")
	if err := os.WriteFile(redactionPath, []byte(fixtureCanaryValue+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	var out, errout bytes.Buffer
	if code := runNonInteractive([]string{"opencode", "evidence", "receipt", "--context", contextPath, "--result", resultPath}, &out, &errout); code != 0 {
		t.Fatalf("receipt projection: %d %s", code, errout.String())
	}
	var projected struct {
		Execution string          `json:"execution"`
		Receipt   json.RawMessage `json:"receipt"`
	}
	if err := json.Unmarshal(out.Bytes(), &projected); err != nil || projected.Execution != "completed" || len(projected.Receipt) == 0 {
		t.Fatalf("projection output mismatch: %s", out.String())
	}
	receiptPath := filepath.Join(private, "receipt.json")
	if err := os.WriteFile(receiptPath, projected.Receipt, 0600); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(logData)
	validateArgs := []string{"opencode", "evidence", "validate",
		"--receipt", receiptPath, "--workspace", workspace, "--identity", identityPath,
		"--contract", contractPath, "--result", resultPath, "--evidence-root", evidence,
		"--authorization", "fixture authorization",
		"--log-digest", logPath + "=" + hex.EncodeToString(sum[:])}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive(validateArgs, &out, &errout); code != 0 || !strings.Contains(out.String(), "consistent") {
		t.Fatalf("validate: %d %s", code, errout.String())
	}

	stale := identity
	stale.Config = ptr("changed-config")
	stalePath := writeJSON("stale.json", stale)
	staleArgs := append([]string{}, validateArgs...)
	for i, arg := range staleArgs {
		if arg == identityPath {
			staleArgs[i] = stalePath
		}
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive(staleArgs, &out, &errout); code != 2 {
		t.Fatal("stale identity was accepted")
	}

	outputPath := filepath.Join(private, "exported.json")
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "evidence", "export",
		"--receipt", receiptPath, "--workspace", workspace, "--output", outputPath,
		"--redactions", redactionPath}, &out, &errout); code != 0 {
		t.Fatalf("export: %d %s", code, errout.String())
	}
	exported, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported), fixtureCanaryValue) {
		t.Fatal("redaction canary reached the exported receipt")
	}
	out.Reset()
	errout.Reset()
	if code := runNonInteractive([]string{"opencode", "evidence", "export",
		"--receipt", receiptPath, "--workspace", workspace,
		"--output", filepath.Join(workspace, "leaked.json")}, &out, &errout); code != 2 {
		t.Fatal("checkout-internal export was accepted")
	}
}
