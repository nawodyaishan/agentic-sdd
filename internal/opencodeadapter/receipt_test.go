package opencodeadapter

// Receipt projection tests. They compose synthetic runner results with
// hand-built and fake-peer native observations; no real native process, model,
// provider, credential store or canonical project state is touched.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentic-sdd/internal/harnesscheck"
	"agentic-sdd/internal/harnessevidence"
)

const projectionContract = `{
  "version": 1,
  "checks": [
    {"id": "test", "cwd": ".", "scopes": ["baseline", "focused", "required"],
     "prerequisites": [], "timeout_seconds": 30,
     "effects": {"writes": ["fixture state"], "network": "none"},
     "steps": [{"argv": ["true"], "env": {}}], "local_recipe": ["true"],
     "expected_exit": 0, "evidence": "test.log"},
    {"id": "vet", "cwd": ".", "scopes": ["baseline", "focused", "required"],
     "prerequisites": [], "timeout_seconds": 30,
     "effects": {"writes": ["fixture state"], "network": "none"},
     "steps": [{"argv": ["true"], "env": {}}], "local_recipe": ["true"],
     "expected_exit": 0, "evidence": "vet.log"}
  ]
}`

func digestHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fixtureScopeIdentity() harnessevidence.ScopeIdentity {
	return harnessevidence.ScopeIdentity{
		BaseRevision: "0123456789abcdef0123456789abcdef01234567",
		Diff:         "sha256-v1:" + strings.Repeat("a", 64),
	}
}

func passingCheck(id, logPath string) harnesscheck.CheckResult {
	start := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Second)
	exit := 0
	return harnesscheck.CheckResult{
		ID: id, Status: "passed", ExitCode: &exit,
		StartedAt: &start, EndedAt: &end, Evidence: &logPath,
		Truncated: false, Reason: "process exited zero",
	}
}

func blockedCheck(id string) harnesscheck.CheckResult {
	return harnesscheck.CheckResult{ID: id, Status: "blocked", Reason: "prerequisite missing"}
}

func projectionResult(t *testing.T, workspace, contractDigest, execution string, checks ...harnesscheck.CheckResult) []byte {
	t.Helper()
	scope := "required"
	ids := make([]string, 0, len(checks))
	for _, c := range checks {
		ids = append(ids, c.ID)
	}
	data, err := json.Marshal(harnesscheck.Result{
		Version:        1,
		Workspace:      workspace,
		ContractSHA256: contractDigest,
		Selection:      harnesscheck.Selection{Scope: scope, CheckIDs: ids},
		Execution:      execution,
		Checks:         checks,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func baseContext(t *testing.T, contractDigest string) ReceiptContext {
	t.Helper()
	identity := fixtureScopeIdentity()
	return ReceiptContext{
		Synthetic:        true,
		RunID:            "fixture-run",
		Mode:             "build",
		RequestRef:       "synthetic request",
		AuthorizationRef: strPtr("fixture authorization"),
		Feature:          strPtr("005-vnext-completion"),
		Batch:            strPtr("B4"),
		TaskIDs:          []string{"T10"},
		AllowedEffects:   []string{"synthetic fixture state"},
		Profile:          strPtr("synthetic-build"),
		Tools:            []string{},
		Limits: harnessevidence.Limits{
			DeadlineSeconds: 60, Steps: 5, TransientRetries: 0,
			PaidAuthorized: false, SpendUSD: nil,
		},
		Identity: ContextIdentity{
			Config:      strPtr("fixture-config"),
			Skills:      strPtr("fixture-skills"),
			Environment: strPtr("fixture-environment"),
			Runtime:     strPtr("fixture-runtime"),
			Harness:     strPtr("fixture-harness"),
			Contract:    strPtr(contractDigest),
		},
		IdentityBefore: identity,
		IdentityAfter:  identity,
		Model: harnessevidence.Model{
			Provider: strPtr("synthetic"), ID: strPtr("synthetic-model"),
		},
		NextAction: "human reviews the batch",
	}
}

func observation(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func strPtr(s string) *string { return &s }

func privateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestProjectCompletedTurnComposesChecksAndValidates(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logs := map[string]string{}
	for _, name := range []string{"test.log", "vet.log"} {
		path := filepath.Join(evidenceRoot, name)
		data := []byte(name + " independent assertion passed\n")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
		logs[path] = digestHex(data)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed",
		passingCheck("test", filepath.Join(evidenceRoot, "test.log")),
		passingCheck("vet", filepath.Join(evidenceRoot, "vet.log")))

	cost := 0.02
	native := observation(t, SessionOutcome{
		Execution: "completed", SessionID: "ses_fixture", StopReason: "end_turn",
		Usage: &Usage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3, CostUSD: &cost,
			Reported: "native-reported-not-verified"},
	})
	context := baseContext(t, digest)

	receipt, pendingPath, err := Project(context, resultBytes, native, "")
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if pendingPath != "" {
		t.Fatalf("completed turn wrote a pending sidecar: %q", pendingPath)
	}
	if receipt.Execution != "completed" || receipt.Review != nil {
		t.Fatalf("projection fabricated execution or review: %+v", receipt)
	}
	if receipt.SessionID == nil || *receipt.SessionID != "ses_fixture" {
		t.Fatalf("native session not recorded: %+v", receipt.SessionID)
	}
	if receipt.CostUSD == nil || *receipt.CostUSD != cost {
		t.Fatalf("native cost observation not recorded: %+v", receipt.CostUSD)
	}
	if len(receipt.Checks) != 2 || receipt.Selection.Scope != "required" {
		t.Fatalf("checks-derived projection mismatch: %+v", receipt.Selection)
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harnessevidence.ParseReceipt(data); err != nil {
		t.Fatalf("projected receipt is not v1: %v", err)
	}

	identity := harnessevidence.Identity{
		BaseRevision: &context.IdentityBefore.BaseRevision,
		Diff:         &context.IdentityBefore.Diff,
		Config:       context.Identity.Config,
		Skills:       context.Identity.Skills,
		Environment:  context.Identity.Environment,
		Runtime:      context.Identity.Runtime,
		Harness:      context.Identity.Harness,
		Contract:     context.Identity.Contract,
	}
	err = harnessevidence.Validate(data, harnessevidence.ValidateInput{
		Workspace:        workspace,
		Identity:         identity,
		AuthorizationRef: "fixture authorization",
		Contract:         contract,
		Result:           resultBytes,
		EvidenceRoot:     evidenceRoot,
		LogDigests:       logs,
		Observations: harnessevidence.Observations{
			"/session_id":     "ses_fixture",
			"/cost_usd":       cost,
			"/model/provider": "synthetic",
			"/model/id":       "synthetic-model",
		},
	})
	if err != nil {
		t.Fatalf("projected receipt failed independent validation: %v", err)
	}
}

func TestProjectRefusesFailedTurnOverCompletedResult(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logPath := filepath.Join(evidenceRoot, "test.log")
	if err := os.WriteFile(logPath, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed", passingCheck("test", logPath))
	context := baseContext(t, digest)

	for _, outcome := range []SessionOutcome{
		{Execution: "incomplete", SessionID: "ses_fixture"},
		{Execution: "cancelled", SessionID: "ses_fixture"},
		{Execution: "blocked", SessionID: "ses_fixture"},
	} {
		if _, _, err := Project(context, resultBytes, observation(t, outcome), ""); err == nil {
			t.Fatalf("native %s turn was receipted over a completed result", outcome.Execution)
		}
	}
}

func TestProjectRequiresEndTurnForCompletedNativeTurn(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logPath := filepath.Join(evidenceRoot, "test.log")
	if err := os.WriteFile(logPath, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed", passingCheck("test", logPath))
	context := baseContext(t, digest)
	outcome := SessionOutcome{Execution: "completed", SessionID: "ses_fixture", StopReason: "max_tokens"}
	if _, _, err := Project(context, resultBytes, observation(t, outcome), ""); err == nil {
		t.Fatalf("completed native turn without end_turn was accepted")
	}
}

func TestProjectRecordsMissingNativeObservationAsUnknown(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logPath := filepath.Join(evidenceRoot, "test.log")
	if err := os.WriteFile(logPath, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed", passingCheck("test", logPath))
	context := baseContext(t, digest)

	receipt, _, err := Project(context, resultBytes, nil, "")
	if err != nil {
		t.Fatalf("projection without a native observation: %v", err)
	}
	if receipt.SessionID != nil || receipt.CostUSD != nil {
		t.Fatalf("absent native observation produced values: %+v", receipt)
	}
	for _, ptr := range []string{"/session_id", "/cost_usd"} {
		if strings.TrimSpace(receipt.Unknowns[ptr]) == "" {
			t.Fatalf("missing explicit unknown reason for %s", ptr)
		}
	}

	completedNoUsage := observation(t, SessionOutcome{
		Execution: "completed", SessionID: "ses_fixture", StopReason: "end_turn",
	})
	receipt, _, err = Project(context, resultBytes, completedNoUsage, "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.CostUSD != nil {
		t.Fatalf("missing native cost became a value: %+v", receipt.CostUSD)
	}
	if strings.TrimSpace(receipt.Unknowns["/cost_usd"]) == "" {
		t.Fatalf("missing cost reason: %+v", receipt.Unknowns)
	}
}

func TestProjectWritesPendingSidecarOutsideDiscovery(t *testing.T) {
	workspace := privateDir(t)
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "blocked",
		blockedCheck("test"), blockedCheck("vet"))
	context := baseContext(t, digest)
	pendingDir := privateDir(t)
	pending := PendingSummary{
		Version:    pendingVersion,
		Run:        "pending_human",
		Session:    "ses_fixture",
		Workspace:  workspace,
		Request:    "native pending human input",
		Question:   "Which package manager?",
		Options:    []PendingOption{{ID: "bun", Label: "bun"}, {ID: "npm", Label: "npm"}},
		Rationale:  "unknown",
		Scope:      "consequential human answer required before resumption",
		NextAction: "human supplies the answer; reconcile scope/authority before re-entry",
	}
	outcome := SessionOutcome{Execution: "blocked", SessionID: "ses_fixture", StopReason: "pending_human", Pending: &pending}

	receipt, path, err := Project(context, resultBytes, observation(t, outcome), pendingDir)
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if path == "" {
		t.Fatalf("pending question did not produce a sidecar")
	}
	if receipt.Execution != "blocked" || receipt.Review != nil {
		t.Fatalf("blocked projection fabricated execution or review: %+v", receipt)
	}
	if !strings.Contains(receipt.NextAction, path) {
		t.Fatalf("next action does not reference the pending sidecar: %q", receipt.NextAction)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("pending sidecar missing: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("pending sidecar is not private: %v", info.Mode().Perm())
	}
	summary, _, err := ReadPending(path)
	if err != nil {
		t.Fatalf("pending sidecar is unreadable: %v", err)
	}
	if summary.Question != "Which package manager?" || len(summary.Options) != 2 {
		t.Fatalf("pending summary was altered: %+v", summary)
	}
	if _, _, err := Project(context, resultBytes, observation(t, outcome), pendingDir); err == nil {
		t.Fatalf("pending sidecar was overwritten")
	}
}

func TestProjectRefusesIdentityDriftAndContractMismatch(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logPath := filepath.Join(evidenceRoot, "test.log")
	if err := os.WriteFile(logPath, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed", passingCheck("test", logPath))

	drifted := baseContext(t, digest)
	drifted.IdentityAfter.Diff = "sha256-v1:" + strings.Repeat("b", 64)
	if _, _, err := Project(drifted, resultBytes, nil, ""); err == nil {
		t.Fatalf("identity drift between check start and end was accepted")
	}

	mismatched := baseContext(t, digest)
	if _, _, err := Project(mismatched, projectionResult(t, workspace, digestHex([]byte("other")), "completed", passingCheck("test", logPath)), nil, ""); err == nil {
		t.Fatalf("result/contract identity mismatch was accepted")
	}
}

func TestProjectRefusesInvalidContextAndObservation(t *testing.T) {
	workspace := privateDir(t)
	evidenceRoot := privateDir(t)
	logPath := filepath.Join(evidenceRoot, "test.log")
	if err := os.WriteFile(logPath, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "completed", passingCheck("test", logPath))

	direct := baseContext(t, digest)
	direct.Mode = "direct-fix"
	if _, _, err := Project(direct, resultBytes, nil, ""); err == nil {
		t.Fatalf("direct fix invented feature/batch/task state")
	}
	direct.Feature, direct.Batch, direct.TaskIDs = nil, nil, nil
	if _, _, err := Project(direct, resultBytes, nil, ""); err != nil {
		t.Fatalf("valid direct fix context rejected: %v", err)
	}

	missingBatch := baseContext(t, digest)
	missingBatch.Batch = nil
	if _, _, err := Project(missingBatch, resultBytes, nil, ""); err == nil {
		t.Fatalf("feature mode without batch identity was accepted")
	}

	unsupported := baseContext(t, digest)
	unsupported.Mode = "autonomous"
	if _, _, err := Project(unsupported, resultBytes, nil, ""); err == nil {
		t.Fatalf("unsupported mode was accepted")
	}

	noIdentity := baseContext(t, digest)
	noIdentity.IdentityBefore.Diff = ""
	if _, _, err := Project(noIdentity, resultBytes, nil, ""); err == nil {
		t.Fatalf("context without measured identity was accepted")
	}

	if _, _, err := Project(baseContext(t, digest), resultBytes, []byte(`{"execution":"completed","extra":true}`), ""); err == nil {
		t.Fatalf("unknown observation field was accepted")
	}
	if _, _, err := Project(baseContext(t, digest), []byte(`{"version":2}`), nil, ""); err == nil {
		t.Fatalf("foreign check result was accepted")
	}
}

func TestProjectElicitationFromFakePeer(t *testing.T) {
	outcome := runScenario(t, "elicitation", baseParams(), acpDefaultDeadline)
	if outcome.Execution != "blocked" || outcome.Pending == nil {
		t.Fatalf("fake peer did not stop on a human question: %+v", outcome)
	}
	workspace := privateDir(t)
	contract := []byte(projectionContract)
	digest := digestHex(contract)
	resultBytes := projectionResult(t, workspace, digest, "blocked",
		blockedCheck("test"), blockedCheck("vet"))
	context := baseContext(t, digest)
	pendingDir := privateDir(t)

	receipt, path, err := Project(context, resultBytes, observation(t, outcome), pendingDir)
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if path == "" || receipt.Execution != "blocked" || receipt.Review != nil {
		t.Fatalf("elicitation projection mismatch: %+v path=%q", receipt, path)
	}
	if receipt.SessionID == nil || *receipt.SessionID != fakeSession {
		t.Fatalf("native session identity not recorded: %+v", receipt.SessionID)
	}
	summary, _, err := ReadPending(path)
	if err != nil {
		t.Fatalf("pending sidecar unreadable: %v", err)
	}
	if summary.Question != "Which package manager?" || summary.Rationale != "unknown" {
		t.Fatalf("pending summary invented content: %+v", summary)
	}
}
