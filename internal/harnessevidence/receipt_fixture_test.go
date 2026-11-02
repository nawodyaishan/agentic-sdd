package harnessevidence

// Shared synthetic fixture for the receipt consumer tests. Every workspace,
// receipt, contract and log is generated here; no real repository, credential
// store, native session or provider is read. The contract mirrors the shape
// the runner produces so validation reuses internal/harnesscheck unchanged.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const fixtureContract = `{
  "version": 1,
  "checks": [{
    "id": "test",
    "cwd": ".",
    "scopes": ["baseline", "focused", "required"],
    "prerequisites": [],
    "timeout_seconds": 30,
    "effects": {"writes": ["synthetic fixture state"], "network": "none"},
    "steps": [{"argv": ["true"], "env": {}}],
    "local_recipe": ["true"],
    "expected_exit": 0,
    "evidence": "test.log"
  }]
}`

const (
	fixtureLogText       = "independent synthetic assertion passed\n"
	fixtureAuthorization = "fixture authorization"
)

type receiptFixture struct {
	t          *testing.T
	root       string
	workspace  string
	evidence   string
	log        string
	contract   []byte
	identity   Identity
	receipt    map[string]any
	result     map[string]any
	logDigests map[string]string
}

func newReceiptFixture(t *testing.T) *receiptFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	workspace := filepath.Join(root, "project")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	fixtureGit(t, workspace, "init", "-q")
	fixtureGit(t, workspace, "config", "user.name", "Synthetic Fixture")
	fixtureGit(t, workspace, "config", "user.email", "fixture@example.invalid")
	writeFixtureFile(t, workspace, "code.txt", "original\n", 0o644)
	writeFixtureFile(t, workspace, "tasks.md", "B1 awaiting human review\n", 0o644)
	fixtureGit(t, workspace, "add", ".")
	fixtureGit(t, workspace, "commit", "-qm", "synthetic fixture")

	evidence := filepath.Join(root, "evidence")
	if err := os.Mkdir(evidence, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(evidence, 0o700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(evidence, "test.log")
	if err := os.WriteFile(log, []byte(fixtureLogText), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(log, 0o600); err != nil {
		t.Fatal(err)
	}

	resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	resolvedEvidence, err := filepath.EvalSymlinks(evidence)
	if err != nil {
		t.Fatal(err)
	}
	log = filepath.Join(resolvedEvidence, "test.log")

	contract := []byte(fixtureContract)
	snapshot, err := Snapshot(resolvedWorkspace, []string{"."})
	if err != nil {
		t.Fatalf("fixture snapshot: %v", err)
	}
	identity := Identity{
		BaseRevision: strp(snapshot.BaseRevision),
		Diff:         strp(snapshot.Diff),
		Config:       strp("fixture-config"),
		Skills:       strp("fixture-skills"),
		Environment:  strp("fixture-environment"),
		Runtime:      strp("fixture-runtime"),
		Harness:      strp("fixture-harness"),
		Contract:     strp(sha256Hex(contract)),
	}
	f := &receiptFixture{
		t:         t,
		root:      root,
		workspace: resolvedWorkspace,
		evidence:  resolvedEvidence,
		log:       log,
		contract:  contract,
		identity:  identity,
	}
	check := map[string]any{
		"id": "test", "status": "passed", "exit_code": 0,
		"started_at": "2026-10-29T00:00:00Z",
		"ended_at":   "2026-10-29T00:00:01Z",
		"evidence":   log, "truncated": false,
		"reason": "process exited zero",
	}
	identityMap := map[string]any{}
	mustRoundTrip(t, identity, &identityMap)
	f.receipt = map[string]any{
		"version":           1,
		"synthetic":         true,
		"run_id":            "fixture-run",
		"session_id":        nil,
		"mode":              "direct-fix",
		"workspace":         resolvedWorkspace,
		"request_ref":       "synthetic request",
		"authorization_ref": fixtureAuthorization,
		"feature":           nil,
		"batch":             nil,
		"task_ids":          []any{},
		"allowed_effects":   []any{"synthetic fixture edits"},
		"profile":           "synthetic-build",
		"tools":             []any{},
		"limits": map[string]any{
			"deadline_seconds": 900, "steps": 20, "transient_retries": 2,
			"paid_authorized": false, "spend_usd": 0,
		},
		"identity": identityMap,
		"model":    map[string]any{"provider": nil, "id": nil, "variant": nil},
		"selection": map[string]any{
			"scope": "focused", "check_ids": []any{"test"},
		},
		"checks":         []any{check},
		"execution":      "completed",
		"review":         nil,
		"workflow_state": nil,
		"findings":       []any{},
		"affected_files": []any{},
		"next_action":    "none",
		"cost_usd":       nil,
		"unknowns": map[string]any{
			"/session_id":     "no native session observed",
			"/model/provider": "no model request made",
			"/model/id":       "no model request made",
			"/model/variant":  "no model request made",
			"/cost_usd":       "no measured billing; unknown is not zero",
		},
	}
	f.result = f.resultFromReceipt()
	f.logDigests = map[string]string{log: sha256Hex([]byte(fixtureLogText))}
	return f
}

// resultFromReceipt rebuilds the independently observed runner result from the
// current receipt selection, checks and execution, as the reference fixture
// does when a receipt-level mutation must stay consistent with the result.
func (f *receiptFixture) resultFromReceipt() map[string]any {
	selection := cloneValue(f.t, f.receipt["selection"])
	checks := cloneValue(f.t, f.receipt["checks"])
	return map[string]any{
		"version":         1,
		"workspace":       f.workspace,
		"contract_sha256": sha256Hex(f.contract),
		"selection":       selection,
		"execution":       f.receipt["execution"],
		"checks":          checks,
	}
}

func (f *receiptFixture) syncResult() {
	f.result = f.resultFromReceipt()
}

func (f *receiptFixture) input() ValidateInput {
	return ValidateInput{
		Workspace:        f.workspace,
		Identity:         f.identity,
		AuthorizationRef: fixtureAuthorization,
		Contract:         f.contract,
		Result:           marshalJSON(f.t, f.result),
		EvidenceRoot:     f.evidence,
		LogDigests:       f.logDigests,
	}
}

func (f *receiptFixture) validate() error {
	return f.validateWith(f.input())
}

func (f *receiptFixture) validateWith(in ValidateInput) error {
	return Validate(marshalJSON(f.t, f.receipt), in)
}

func (f *receiptFixture) receiptCheck() map[string]any {
	return f.receipt["checks"].([]any)[0].(map[string]any)
}

func (f *receiptFixture) resultCheck() map[string]any {
	return f.result["checks"].([]any)[0].(map[string]any)
}

func (f *receiptFixture) receiptUnknowns() map[string]any {
	return f.receipt["unknowns"].(map[string]any)
}

func (f *receiptFixture) receiptIdentity() map[string]any {
	return f.receipt["identity"].(map[string]any)
}

func (f *receiptFixture) receiptSelection() map[string]any {
	return f.receipt["selection"].(map[string]any)
}

func (f *receiptFixture) receiptLimits() map[string]any {
	return f.receipt["limits"].(map[string]any)
}

func (f *receiptFixture) receiptModel() map[string]any {
	return f.receipt["model"].(map[string]any)
}

// identityWith returns the fixture identity with one field replaced.
func (f *receiptFixture) identityWith(name, value string) Identity {
	changed := f.identity
	switch name {
	case "base_revision":
		changed.BaseRevision = strp(value)
	case "diff":
		changed.Diff = strp(value)
	case "config":
		changed.Config = strp(value)
	case "skills":
		changed.Skills = strp(value)
	case "environment":
		changed.Environment = strp(value)
	case "runtime":
		changed.Runtime = strp(value)
	case "harness":
		changed.Harness = strp(value)
	case "contract":
		changed.Contract = strp(value)
	default:
		f.t.Fatalf("unknown identity field %q", name)
	}
	return changed
}

func (f *receiptFixture) workspaceFiles() map[string]string {
	f.t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(f.workspace, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() == ".git" {
			return nil
		}
		rel, relErr := filepath.Rel(f.workspace, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustRoundTrip(t *testing.T, from, into any) {
	t.Helper()
	data, err := json.Marshal(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatal(err)
	}
}

func cloneValue(t *testing.T, value any) any {
	t.Helper()
	var out any
	mustRoundTrip(t, value, &out)
	return out
}
