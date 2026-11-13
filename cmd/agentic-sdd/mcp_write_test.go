package main

// Feature 015 MCP-B / T37 — CLI contract tests for `agentic-sdd mcp preview`,
// `mcp apply` and `mcp restore list|preview|apply`.
//
// These tests pin the write-surface CLI only: flag grammar, the --auth and
// --expect-config contract, exit codes (2 usage/validation/unsafe target,
// 1 stale/conflict/unsupported/runtime, 0 applied/no-op/report), the default-No
// restore confirmation, stable JSON keys, script behavior without a terminal
// and the absence of any credential or whole-config dump in the output.
//
// Every local engine seam (mcpPlanPreview, mcpApplyPlan, mcpRestoreList,
// mcpRestorePreview, mcpRestoreApply) and the confirmation reader
// (mcpPromptIn) is replaced, so no test reads or writes a real OpenCode
// configuration, starts a process, contacts a remote endpoint or needs a
// compiled skill tree. The engine behavior behind internal/appops and
// internal/skillsync is covered by their own package tests.
//
// The MCP-A tests in mcp_test.go stay untouched.

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

const (
	mcpWriteHome       = "/tmp/mcp-write-home"
	mcpWriteConfigPath = "/tmp/mcp-write-home/.config/opencode/opencode.json"
	mcpWriteBackupRoot = "/tmp/mcp-write-home/.agentic-sdd/mcp/backups"
	mcpWriteBaseDigest = "sha256:base0000"
	mcpWritePlanDigest = "sha256:plan0000"
	mcpWritePlanToken  = "token-plan-0001"
	mcpWriteToken      = "token-restore-0001"
	mcpWriteBackupID   = "20261112T101500.000000000Z"
)

// mcpWriteCall records one call into an engine seam, so a test can assert the
// exact request the CLI built and that a refusal never reached the writer.
type mcpWriteCall struct {
	home        string
	config      string
	backupID    string
	token       string
	expect      string
	opencodeBin string
	servers     []string
	auth        map[string]string
	plan        *skillsync.MCPPlan
}

type mcpWriteStub struct {
	planCalls           []mcpWriteCall
	applyCalls          []mcpWriteCall
	listCalls           []mcpWriteCall
	restorePreviewCalls []mcpWriteCall
	restoreApplyCalls   []mcpWriteCall

	plan        *skillsync.MCPPlan
	planToken   string
	planErr     error
	applyResult *skillsync.MCPApplyResult
	applyErr    error

	restoreList       []mcpBackupSummary
	restoreListErr    error
	restorePreview    *skillsync.MCPRestorePreview
	restoreToken      string
	restorePreviewErr error
	restoreResult     *skillsync.MCPApplyResult
	restoreApplyErr   error
}

func (s *mcpWriteStub) install(t *testing.T) {
	t.Helper()
	prevPlan, prevApply := mcpPlanPreview, mcpApplyPlan
	prevList, prevRestorePreview, prevRestoreApply := mcpRestoreList, mcpRestorePreview, mcpRestoreApply
	prevPrompt := mcpPromptIn
	t.Cleanup(func() {
		mcpPlanPreview, mcpApplyPlan = prevPlan, prevApply
		mcpRestoreList, mcpRestorePreview, mcpRestoreApply = prevList, prevRestorePreview, prevRestoreApply
		mcpPromptIn = prevPrompt
	})

	mcpPlanPreview = func(home, config string, selected []string, auth map[string]string) (*skillsync.MCPPlan, string, error) {
		call := mcpWriteCall{home: home, config: config, servers: append([]string(nil), selected...)}
		if auth != nil {
			call.auth = make(map[string]string, len(auth))
			for name, mode := range auth {
				call.auth[name] = mode
			}
		}
		s.planCalls = append(s.planCalls, call)
		return s.plan, s.planToken, s.planErr
	}
	mcpApplyPlan = func(home, config string, plan *skillsync.MCPPlan, token, expectDigest, openCodeBin string) (*skillsync.MCPApplyResult, error) {
		s.applyCalls = append(s.applyCalls, mcpWriteCall{
			home:        home,
			config:      config,
			token:       token,
			expect:      expectDigest,
			opencodeBin: openCodeBin,
			plan:        plan,
		})
		return s.applyResult, s.applyErr
	}
	mcpRestoreList = func(home string) ([]mcpBackupSummary, error) {
		s.listCalls = append(s.listCalls, mcpWriteCall{home: home})
		return s.restoreList, s.restoreListErr
	}
	mcpRestorePreview = func(home, configPath, backupID string) (*skillsync.MCPRestorePreview, string, error) {
		s.restorePreviewCalls = append(s.restorePreviewCalls, mcpWriteCall{home: home, config: configPath, backupID: backupID})
		return s.restorePreview, s.restoreToken, s.restorePreviewErr
	}
	mcpRestoreApply = func(home, configPath, backupID, token, expectDigest string) (*skillsync.MCPApplyResult, error) {
		s.restoreApplyCalls = append(s.restoreApplyCalls, mcpWriteCall{
			home:     home,
			config:   configPath,
			backupID: backupID,
			token:    token,
			expect:   expectDigest,
		})
		return s.restoreResult, s.restoreApplyErr
	}
}

func installMCPWriteStub(t *testing.T) *mcpWriteStub {
	t.Helper()
	stub := &mcpWriteStub{
		plan:           mcpWritePlanFixture(),
		planToken:      mcpWritePlanToken,
		applyResult:    mcpWriteApplyResultFixture(),
		restoreList:    mcpWriteBackupsFixture(),
		restorePreview: mcpWriteRestorePreviewFixture(),
		restoreToken:   mcpWriteToken,
		restoreResult:  mcpWriteRestoreResultFixture(),
	}
	stub.install(t)
	return stub
}

func mcpWritePlanFixture() *skillsync.MCPPlan {
	return &skillsync.MCPPlan{
		TargetPath:      mcpWriteConfigPath,
		Format:          "json",
		Schema:          "v2",
		BaseDigest:      mcpWriteBaseDigest,
		PlanDigest:      mcpWritePlanDigest,
		Selected:        []string{"codegraph", "context7"},
		AuthChoices:     map[string]string{"context7": "oauth"},
		RuntimeVersion:  "not-checked",
		RuntimeEvidence: "not-checked",
		NotChecked:      []string{"remote connectivity", "authentication"},
		Entries: []skillsync.MCPPlanEntry{
			{
				Name:    "codegraph",
				Op:      "add",
				Reason:  "server is missing from the configuration",
				NewText: "{\n  \"type\": \"local\",\n  \"command\": [\"codegraph\", \"serve\", \"--mcp\"]\n}",
				NewHash: "hash-codegraph-desired",
			},
			{
				Name:    "context7",
				Op:      "leave",
				Reason:  "already configured and unchanged",
				OldHash: "hash-context7",
				NewHash: "hash-context7",
			},
		},
	}
}

func mcpWriteApplyResultFixture() *skillsync.MCPApplyResult {
	return &skillsync.MCPApplyResult{
		TargetPath:      mcpWriteConfigPath,
		BackupID:        mcpWriteBackupID,
		BackupPath:      mcpWriteBackupRoot + "/" + mcpWriteBackupID,
		InstalledDigest: "sha256:installed0000",
		RuntimeVersion:  "0.9.4",
		Applied:         []string{"codegraph", "context7"},
	}
}

func mcpWriteRestoreResultFixture() *skillsync.MCPApplyResult {
	result := mcpWriteApplyResultFixture()
	result.Applied = []string{"codegraph"}
	return result
}

func mcpWriteBackupsFixture() []mcpBackupSummary {
	return []mcpBackupSummary{
		{
			ID:        mcpWriteBackupID,
			Path:      mcpWriteBackupRoot + "/" + mcpWriteBackupID,
			CreatedAt: "2026-11-12T10:15:00Z",
			Servers:   []string{"codegraph", "context7"},
		},
		{
			ID:        "20261111T090000.000000000Z",
			Path:      mcpWriteBackupRoot + "/20261111T090000.000000000Z",
			CreatedAt: "2026-11-11T09:00:00Z",
			Servers:   []string{"codegraph"},
		},
	}
}

func mcpWriteRestorePreviewFixture() *skillsync.MCPRestorePreview {
	return &skillsync.MCPRestorePreview{
		BackupID:   mcpWriteBackupID,
		TargetPath: mcpWriteConfigPath,
		BaseDigest: mcpWriteBaseDigest,
		Entries: []skillsync.MCPPlanEntry{
			{
				Name:    "codegraph",
				Op:      "change",
				Reason:  "managed entry differs from the backup",
				OldText: "{\n  \"type\": \"local\",\n  \"command\": [\"codegraph\", \"serve\"]\n}",
				NewText: "{\n  \"type\": \"local\",\n  \"command\": [\"codegraph\", \"serve\", \"--mcp\"]\n}",
				OldHash: "hash-current",
				NewHash: "hash-backup",
			},
		},
	}
}

// runMCPWrite runs the real CLI entry point with a scripted, non-terminal
// stdin. home is always passed explicitly by the caller so no test can reach
// the real user home.
func runMCPWrite(args []string, stdin string, interactive bool) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut, strings.NewReader(stdin), interactive)
	return code, out.String(), errOut.String()
}

// --- preview --------------------------------------------------------------

func TestMCPWritePreviewPlanText(t *testing.T) {
	stub := installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "preview", "--home", home,
		"--servers", "codegraph,context7", "--auth", "context7=oauth",
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d, want 0; stdout %q; stderr %q", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr %q", stderr)
	}
	for _, want := range []string{
		"MCP preview — structured plan (read-only, no network, no writes)",
		"Home:        " + home,
		"Target:      " + mcpWriteConfigPath,
		"Format:      json",
		"Schema:      v2",
		"Base digest: " + mcpWriteBaseDigest,
		"Plan digest: " + mcpWritePlanDigest,
		"Selected:    codegraph, context7",
		"Auth:        context7=oauth",
		"Runtime:     not-checked (not-checked)",
		"Result:      change ready; apply with --expect-config " + mcpWriteBaseDigest,
		"Entries (2):",
		"add",
		"leave",
		"current: none",
		"planned:",
		"\"type\": \"local\"",
		"Conflicts:   none",
		"Token:       " + mcpWritePlanToken,
		"Preview only. No write.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if len(stub.planCalls) != 1 {
		t.Fatalf("plan preview called %d times, want 1", len(stub.planCalls))
	}
	call := stub.planCalls[0]
	if call.home != home {
		t.Fatalf("plan home = %q, want %q", call.home, home)
	}
	if strings.Join(call.servers, ",") != "codegraph,context7" {
		t.Fatalf("plan servers = %v", call.servers)
	}
	if call.auth["context7"] != "oauth" {
		t.Fatalf("plan auth = %v, want context7=oauth", call.auth)
	}
	if len(stub.applyCalls) != 0 || len(stub.restoreApplyCalls) != 0 {
		t.Fatalf("preview wrote: apply=%v restore=%v", stub.applyCalls, stub.restoreApplyCalls)
	}
}

func TestMCPWritePreviewPlanJSON(t *testing.T) {
	installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{"mcp", "preview", "--home", home, "--json"}, "", false)
	if code != 0 {
		t.Fatalf("exit %d, want 0; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	for _, key := range []string{
		"command", "mode", "home", "target", "format", "schema", "base_digest", "plan_digest",
		"selected", "auth", "entries", "conflicts", "no_op", "runtime_version", "runtime_evidence",
		"not_checked", "token", "preview_only", "write",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("preview JSON missing key %q: %s", key, stdout)
		}
	}
	// The preview must stay a plan projection: no whole-file or credential field.
	for _, forbidden := range []string{"config", "content", "config_bytes", "api_key", "secret", "credential"} {
		if _, ok := doc[forbidden]; ok {
			t.Fatalf("preview JSON carries a %q field: %s", forbidden, stdout)
		}
	}
	if doc["command"] != "preview" || doc["mode"] != "read-only" {
		t.Fatalf("preview identity = %v/%v", doc["command"], doc["mode"])
	}
	if doc["preview_only"] != true || doc["write"] != "none" {
		t.Fatalf("preview JSON does not declare read-only: %s", stdout)
	}
	if doc["token"] != mcpWritePlanToken {
		t.Fatalf("token = %v, want %q", doc["token"], mcpWritePlanToken)
	}
	if doc["base_digest"] != mcpWriteBaseDigest || doc["plan_digest"] != mcpWritePlanDigest {
		t.Fatalf("digests = %v/%v", doc["base_digest"], doc["plan_digest"])
	}
	auth, ok := doc["auth"].(map[string]any)
	if !ok || auth["context7"] != "oauth" {
		t.Fatalf("auth = %v, want context7=oauth", doc["auth"])
	}
	entries, ok := doc["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("entries = %v, want 2", doc["entries"])
	}
	first, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatalf("first entry is not an object: %v", entries[0])
	}
	for _, key := range []string{"name", "op", "reason", "old_text", "new_text", "old_hash", "new_hash"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("entry JSON missing key %q: %v", key, first)
		}
	}
	if first["name"] != "codegraph" || first["op"] != "add" || first["new_hash"] != "hash-codegraph-desired" {
		t.Fatalf("first entry = %v, want codegraph/add", first)
	}
}

func TestMCPWritePreviewConflictExitsOne(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.plan = mcpWritePlanFixture()
	stub.plan.Conflicts = []string{"unmanaged server named context7 differs from the managed entry"}
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{"mcp", "preview", "--home", home}, "", false)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	// The full plan is still reported; only the exit code says "not applicable".
	for _, want := range []string{
		"Result:      refused until the conflicts below are resolved",
		"Conflicts:   unmanaged server named context7 differs from the managed entry",
		"Preview only. No write.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr, "error: the plan reports a conflict") {
		t.Fatalf("stderr = %q, want a conflict error", stderr)
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a conflicted preview wrote: %v", stub.applyCalls)
	}

	code, stdout, stderr = runMCPWrite([]string{"mcp", "preview", "--home", home, "--json"}, "", false)
	if code != 1 {
		t.Fatalf("JSON exit %d, want 1; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	conflicts, ok := doc["conflicts"].([]any)
	if !ok || len(conflicts) != 1 {
		t.Fatalf("JSON conflicts = %v, want one entry", doc["conflicts"])
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a conflicted JSON preview wrote: %v", stub.applyCalls)
	}
}

func TestMCPWritePreviewAuthChoices(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		code     int
		wantAuth map[string]string
		wantErr  []string
	}{
		{
			name:     "both choices",
			args:     []string{"--auth", "context7=oauth", "--auth", "exa=keyless"},
			code:     0,
			wantAuth: map[string]string{"context7": "oauth", "exa": "keyless"},
		},
		{
			name:     "last choice wins",
			args:     []string{"--auth", "exa=keyless", "--auth", "exa=api-key"},
			code:     0,
			wantAuth: map[string]string{"exa": "api-key"},
		},
		{
			name: "no choice is never implicit",
			args: nil,
			code: 0,
		},
		{
			name:    "unsupported mode",
			args:    []string{"--auth", "context7=bogus"},
			code:    2,
			wantErr: []string{"--auth context7 wants one of oauth, api-key"},
		},
		{
			name:    "server without an auth choice",
			args:    []string{"--auth", "codegraph=oauth"},
			code:    2,
			wantErr: []string{`--auth does not apply to server "codegraph"`},
		},
		{
			name:    "missing equals",
			args:    []string{"--auth", "oauth"},
			code:    2,
			wantErr: []string{"--auth wants server=mode"},
		},
		{
			name:    "empty mode",
			args:    []string{"--auth", "exa="},
			code:    2,
			wantErr: []string{"--auth wants server=mode"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := installMCPWriteStub(t)
			home := t.TempDir()
			args := append([]string{"mcp", "preview", "--home", home}, tc.args...)
			code, stdout, stderr := runMCPWrite(args, "", false)
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.code, stdout, stderr)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr %q does not contain %q", stderr, want)
				}
			}
			if tc.code != 0 {
				if len(stub.planCalls) != 0 {
					t.Fatalf("an invalid --auth reached the engine: %v", stub.planCalls)
				}
				return
			}
			if len(stub.planCalls) != 1 {
				t.Fatalf("plan preview calls = %d, want 1", len(stub.planCalls))
			}
			got := stub.planCalls[0].auth
			if len(got) != len(tc.wantAuth) {
				t.Fatalf("auth = %v, want %v", got, tc.wantAuth)
			}
			for name, mode := range tc.wantAuth {
				if got[name] != mode {
					t.Fatalf("auth = %v, want %v", got, tc.wantAuth)
				}
			}
		})
	}
}

func TestMCPWritePreviewUnknownServerUsage(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.planErr = fmt.Errorf("%w: unknown server %q", skillsync.ErrMCPUsage, "bogus")
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{"mcp", "preview", "--home", home, "--servers", "bogus"}, "", false)
	if code != 2 {
		t.Fatalf("exit %d, want 2; stdout %q; stderr %q", code, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"MCP usage error", `unknown server "bogus"`, "note: correct the request"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a usage error wrote: %v", stub.applyCalls)
	}
}

// --- apply ----------------------------------------------------------------

func TestMCPWriteApplyRequiresExpectConfig(t *testing.T) {
	for _, args := range [][]string{
		{"mcp", "apply"},
		{"mcp", "apply", "--expect-config", ""},
		{"mcp", "apply", "--expect-config", "   "},
	} {
		stub := installMCPWriteStub(t)
		home := t.TempDir()
		code, stdout, stderr := runMCPWrite(append(args, "--home", home), "", false)
		if code != 2 {
			t.Fatalf("%v: exit %d, want 2; stdout %q; stderr %q", args, code, stdout, stderr)
		}
		if stdout != "" {
			t.Fatalf("%v: stdout = %q, want empty", args, stdout)
		}
		for _, want := range []string{
			"error: mcp apply requires --expect-config",
			"an empty or missing digest never applies",
		} {
			if !strings.Contains(stderr, want) {
				t.Fatalf("%v: stderr %q does not contain %q", args, stderr, want)
			}
		}
		calls := len(stub.planCalls) + len(stub.applyCalls)
		if calls != 0 {
			t.Fatalf("%v: an apply without a digest reached the engine (%d calls)", args, calls)
		}
	}
}

func TestMCPWriteApplyStaleDigestRefused(t *testing.T) {
	stub := installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home, "--expect-config", "sha256:stale0000",
	}, "", false)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"error: stale plan: --expect-config sha256:stale0000 does not match the current configuration digest " + mcpWriteBaseDigest,
		"nothing was written; run \"asdx-core mcp preview\" and apply the digest it prints",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a stale digest wrote: %v", stub.applyCalls)
	}
}

func TestMCPWriteApplyConflictRefused(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.plan = mcpWritePlanFixture()
	stub.plan.Conflicts = []string{"unmanaged server named exa differs from the managed entry"}
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest,
	}, "", false)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"error: mcp apply refused: the plan reports a conflict",
		"conflict: unmanaged server named exa differs from the managed entry",
		"nothing was written; v1 has no replace or adopt option",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a conflicted plan wrote: %v", stub.applyCalls)
	}
}

func TestMCPWriteApplyMissingTargetRefused(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.plan = mcpWritePlanFixture()
	stub.plan.TargetPath = ""
	stub.plan.BaseDigest = ""
	home := t.TempDir()

	// There is nothing to write and v1 never creates a configuration: the run
	// refuses instead of inventing a target.
	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest,
	}, "", false)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"mcp apply refused: no existing global OpenCode configuration target",
		"v1 never creates a configuration",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if len(stub.applyCalls) != 0 {
		t.Fatalf("a missing target wrote: %v", stub.applyCalls)
	}
}

func TestMCPWriteApplyWritesOnce(t *testing.T) {
	stub := installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home,
		"--config", mcpWriteConfigPath,
		"--expect-config", mcpWriteBaseDigest,
		"--opencode-bin", "/bin/opencode",
		"--servers", "codegraph,context7",
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d, want 0; stdout %q; stderr %q", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr %q", stderr)
	}
	for _, want := range []string{
		"MCP apply — digest-checked write",
		"Home:        " + home,
		"Target:      " + mcpWriteConfigPath,
		"Base digest: " + mcpWriteBaseDigest,
		"Plan digest: " + mcpWritePlanDigest,
		"Result:      applied",
		"Applied:     codegraph, context7",
		"Backup:      " + mcpWriteBackupID + " at " + mcpWriteBackupRoot + "/" + mcpWriteBackupID,
		"Installed:   sha256:installed0000",
		"Runtime:     0.9.4",
		"Rollback:    not needed",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Result:      no change") {
		t.Fatalf("an applied result is labeled as a no-op:\n%s", stdout)
	}

	if len(stub.planCalls) != 1 {
		t.Fatalf("plan preview calls = %d, want 1", len(stub.planCalls))
	}
	if len(stub.applyCalls) != 1 {
		t.Fatalf("apply calls = %d, want exactly 1", len(stub.applyCalls))
	}
	call := stub.applyCalls[0]
	if call.home != home || call.config != mcpWriteConfigPath {
		t.Fatalf("apply target = %q/%q", call.home, call.config)
	}
	if call.token != mcpWritePlanToken {
		t.Fatalf("apply token = %q, want %q", call.token, mcpWritePlanToken)
	}
	if call.expect != mcpWriteBaseDigest {
		t.Fatalf("apply expect digest = %q, want %q", call.expect, mcpWriteBaseDigest)
	}
	if call.opencodeBin != "/bin/opencode" {
		t.Fatalf("apply opencode bin = %q", call.opencodeBin)
	}
	if call.plan == nil || call.plan.BaseDigest != mcpWriteBaseDigest {
		t.Fatalf("apply plan = %+v, want the previewed plan", call.plan)
	}
}

func TestMCPWriteApplyNoOp(t *testing.T) {
	stub := installMCPWriteStub(t)
	result := &skillsync.MCPApplyResult{
		TargetPath:      mcpWriteConfigPath,
		InstalledDigest: "sha256:installed0000",
		RuntimeVersion:  "0.9.4",
		NoOp:            true,
	}
	stub.applyResult = result
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest,
	}, "", false)
	if code != 0 {
		t.Fatalf("a no-op must exit 0, got %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"Result:      no change (the configuration already matched; no backup was created)",
		"Backup:      none",
		"Applied:     none",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Result:      applied") {
		t.Fatalf("a no-op is labeled applied:\n%s", stdout)
	}
	if len(stub.applyCalls) != 1 {
		t.Fatalf("apply calls = %d, want exactly 1 (no second write)", len(stub.applyCalls))
	}
}

func TestMCPWriteApplyJSON(t *testing.T) {
	installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest, "--json",
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d, want 0; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	for _, key := range []string{
		"command", "mode", "home", "target", "base_digest", "plan_digest", "no_op",
		"applied", "backup_id", "backup_path", "installed_digest", "runtime_version", "rollback",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("apply JSON missing key %q: %s", key, stdout)
		}
	}
	if doc["command"] != "apply" || doc["mode"] != "write" {
		t.Fatalf("apply identity = %v/%v", doc["command"], doc["mode"])
	}
	if doc["no_op"] != false {
		t.Fatalf("no_op = %v, want false", doc["no_op"])
	}
	if doc["backup_id"] != mcpWriteBackupID || doc["installed_digest"] != "sha256:installed0000" {
		t.Fatalf("apply JSON backup/digest = %v/%v", doc["backup_id"], doc["installed_digest"])
	}
	applied, ok := doc["applied"].([]any)
	if !ok || len(applied) != 2 || applied[0] != "codegraph" {
		t.Fatalf("applied = %v, want [codegraph context7]", doc["applied"])
	}
	rollback, ok := doc["rollback"].(map[string]any)
	if !ok {
		t.Fatalf("rollback is not an object: %v", doc["rollback"])
	}
	for _, key := range []string{"attempted", "complete", "error"} {
		if _, ok := rollback[key]; !ok {
			t.Fatalf("rollback JSON missing key %q: %v", key, rollback)
		}
	}
	if rollback["attempted"] != false || rollback["error"] != "" {
		t.Fatalf("rollback = %v, want not attempted", rollback)
	}
}

func TestMCPWriteApplyEngineFailures(t *testing.T) {
	tests := []struct {
		name     string
		planErr  error
		applyErr error
		code     int
		wantErr  []string
	}{
		{
			name:     "stale after preview",
			applyErr: fmt.Errorf("%w: the target changed", skillsync.ErrMCPStale),
			code:     1,
			wantErr:  []string{"MCP configuration changed since preview", "run \"asdx-core mcp preview\" again and review the new digest"},
		},
		{
			name:     "needs a fresh preview",
			applyErr: appops.ErrNeedsPreview,
			code:     1,
			wantErr:  []string{"preview", "nothing was written"},
		},
		{
			name:     "conflict under the lock",
			applyErr: fmt.Errorf("%w: context7 was edited", skillsync.ErrMCPConflict),
			code:     1,
			wantErr:  []string{"MCP configuration conflict", "no replace or adopt option"},
		},
		{
			name:     "unsupported edit",
			applyErr: fmt.Errorf("%w: jsonc comment ownership is ambiguous", skillsync.ErrMCPUnsupported),
			code:     1,
			wantErr:  []string{"unsupported MCP configuration or runtime", "refuses edits it cannot prove safe"},
		},
		{
			name:     "runtime failure",
			applyErr: errors.New("disk full"),
			code:     1,
			wantErr:  []string{"error: disk full", "re-run \"asdx-core mcp preview\" before retrying"},
		},
		{
			name:    "unsafe target before planning",
			planErr: fmt.Errorf("%w: the target is a symlink", skillsync.ErrMCPUnsafeTarget),
			code:    2,
			wantErr: []string{"unsafe MCP configuration target", "nothing was read or written outside the intended target"},
		},
		{
			name:    "usage before planning",
			planErr: fmt.Errorf("%w: unknown auth choice", skillsync.ErrMCPUsage),
			code:    2,
			wantErr: []string{"MCP usage error", "correct the request"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := installMCPWriteStub(t)
			stub.planErr = tc.planErr
			stub.applyErr = tc.applyErr
			home := t.TempDir()

			code, stdout, stderr := runMCPWrite([]string{
				"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest,
			}, "", false)
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.code, stdout, stderr)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr %q does not contain %q", stderr, want)
				}
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if tc.planErr != nil && len(stub.applyCalls) != 0 {
				t.Fatalf("a failed plan still wrote: %v", stub.applyCalls)
			}
			if len(stub.applyCalls) > 1 {
				t.Fatalf("apply was retried: %v", stub.applyCalls)
			}
		})
	}
}

func TestMCPWriteApplyPassesTokenAndDigestExactly(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.plan = mcpWritePlanFixture()
	stub.plan.BaseDigest = "sha256:BaseCase42"
	stub.planToken = "appops-mcp-plan-preview-v1-deadbeef"
	home := t.TempDir()

	code, _, stderr := runMCPWrite([]string{
		"mcp", "apply", "--home", home,
		"--expect-config", "sha256:BaseCase42",
		"--opencode-bin", "/usr/local/bin/opencode",
		"--auth", "exa=api-key",
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	if len(stub.applyCalls) != 1 {
		t.Fatalf("apply calls = %d, want 1", len(stub.applyCalls))
	}
	call := stub.applyCalls[0]
	if call.token != "appops-mcp-plan-preview-v1-deadbeef" {
		t.Fatalf("token = %q, want the exact preview token", call.token)
	}
	if call.expect != "sha256:BaseCase42" {
		t.Fatalf("expect digest = %q, want the exact flag value", call.expect)
	}
	if call.opencodeBin != "/usr/local/bin/opencode" {
		t.Fatalf("opencode bin = %q", call.opencodeBin)
	}
	// The auth choice reaches the plan request, never the apply request: apply
	// is bound to the digest and token the reviewed preview printed.
	if got := stub.planCalls[0].auth["exa"]; got != "api-key" {
		t.Fatalf("plan auth exa = %q, want api-key", got)
	}
}

func TestMCPWriteUsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		code    int
		wantOut []string
		wantErr []string
	}{
		{
			name:    "apply positional",
			args:    []string{"mcp", "apply", "--expect-config", "sha256:x", "extra"},
			code:    2,
			wantErr: []string{`error: unexpected argument "extra"`},
		},
		{
			name:    "apply unknown flag",
			args:    []string{"mcp", "apply", "--bogus"},
			code:    2,
			wantErr: []string{"flag provided but not defined: -bogus", `run "asdx-core mcp help"`},
		},
		{
			name:    "apply help",
			args:    []string{"mcp", "apply", "--help"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp apply", "--expect-config SHA256"},
		},
		{
			name:    "apply empty server list",
			args:    []string{"mcp", "apply", "--expect-config", "sha256:x", "--servers", " , "},
			code:    2,
			wantErr: []string{"--servers requires at least one server name"},
		},
		{
			name:    "restore list positional",
			args:    []string{"mcp", "restore", "list", "extra"},
			code:    2,
			wantErr: []string{`error: unexpected argument "extra"`},
		},
		{
			name:    "restore unknown subcommand",
			args:    []string{"mcp", "restore", "bogus"},
			code:    2,
			wantErr: []string{`error: unknown mcp restore command "bogus"`, "Usage: asdx-core mcp restore"},
		},
		{
			name:    "restore help",
			args:    []string{"mcp", "restore", "help"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp restore", "restore apply ID --expect-config"},
		},
		{
			name:    "restore preview without id",
			args:    []string{"mcp", "restore", "preview"},
			code:    2,
			wantErr: []string{"mcp restore preview requires a backup id"},
		},
		{
			name:    "restore preview with two ids",
			args:    []string{"mcp", "restore", "preview", "a", "b"},
			code:    2,
			wantErr: []string{`error: unexpected argument "b"`, "restore accepts exactly one backup id"},
		},
		{
			name:    "restore apply without id",
			args:    []string{"mcp", "restore", "apply", "--expect-config", "sha256:x"},
			code:    2,
			wantErr: []string{"mcp restore apply requires a backup id"},
		},
		{
			name:    "restore apply without digest",
			args:    []string{"mcp", "restore", "apply", mcpWriteBackupID},
			code:    2,
			wantErr: []string{"mcp restore apply requires --expect-config", "a missing digest never restores"},
		},
		{
			name:    "restore apply has no opencode-bin flag",
			args:    []string{"mcp", "restore", "apply", mcpWriteBackupID, "--expect-config", "sha256:x", "--opencode-bin", "/bin/opencode"},
			code:    2,
			wantErr: []string{"flag provided but not defined: -opencode-bin"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := installMCPWriteStub(t)
			home := t.TempDir()
			args := append(append([]string{}, tc.args...), "--home", home)
			code, stdout, stderr := runMCPWrite(args, "", false)
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.code, stdout, stderr)
			}
			for _, want := range tc.wantOut {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout %q does not contain %q", stdout, want)
				}
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr %q does not contain %q", stderr, want)
				}
			}
			if tc.code != 0 && len(stub.applyCalls) != 0 {
				t.Fatalf("a usage error wrote: %v", stub.applyCalls)
			}
			if tc.code != 0 && len(stub.restoreApplyCalls) != 0 {
				t.Fatalf("a usage error restored: %v", stub.restoreApplyCalls)
			}
		})
	}
}

// --- restore --------------------------------------------------------------

func TestMCPWriteRestoreListEmpty(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.restoreList = nil
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{"mcp", "restore", "list", "--home", home}, "", false)
	if code != 0 {
		t.Fatalf("an empty list is a report, exit %d; stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr %q", stderr)
	}
	for _, want := range []string{
		"MCP restore — managed MCP backups (read-only)",
		"Home:     " + home,
		"Backups (0):",
		"none (no managed MCP backup is recorded)",
		"Preview is read-only; apply needs an id and --expect-config.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "[y/N]") {
		t.Fatalf("a scripted list prompted:\n%s", stdout)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("list emitted ANSI styling on a non-terminal stdout")
	}
	if len(stub.listCalls) != 1 || stub.listCalls[0].home != home {
		t.Fatalf("list calls = %v", stub.listCalls)
	}

	code, stdout, stderr = runMCPWrite([]string{"mcp", "restore", "list", "--home", home, "--json"}, "", false)
	if code != 0 {
		t.Fatalf("JSON exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	if doc["count"] != float64(0) {
		t.Fatalf("count = %v, want 0", doc["count"])
	}
	backups, ok := doc["backups"].([]any)
	if !ok || len(backups) != 0 {
		t.Fatalf("backups = %v, want an empty array", doc["backups"])
	}
}

func TestMCPWriteRestoreListPopulatedJSON(t *testing.T) {
	installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{"mcp", "restore", "list", "--home", home}, "", false)
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"Backups (2):",
		mcpWriteBackupID,
		"created 2026-11-12T10:15:00Z",
		"servers: codegraph, context7",
		"path: " + mcpWriteBackupRoot + "/" + mcpWriteBackupID,
		"20261111T090000.000000000Z",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}

	code, stdout, stderr = runMCPWrite([]string{"mcp", "restore", "list", "--home", home, "--json"}, "", false)
	if code != 0 {
		t.Fatalf("JSON exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	for _, key := range []string{"command", "mode", "home", "count", "backups"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("restore list JSON missing key %q: %s", key, stdout)
		}
	}
	if doc["command"] != "restore list" || doc["mode"] != "read-only" {
		t.Fatalf("restore list identity = %v/%v", doc["command"], doc["mode"])
	}
	backups, ok := doc["backups"].([]any)
	if !ok || len(backups) != 2 {
		t.Fatalf("backups = %v, want 2", doc["backups"])
	}
	first, ok := backups[0].(map[string]any)
	if !ok {
		t.Fatalf("first backup is not an object: %v", backups[0])
	}
	for _, key := range []string{"id", "path", "created_at", "servers"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("backup JSON missing key %q: %v", key, first)
		}
	}
	if first["id"] != mcpWriteBackupID || first["created_at"] != "2026-11-12T10:15:00Z" {
		t.Fatalf("first backup = %v", first)
	}
	servers, ok := first["servers"].([]any)
	if !ok || len(servers) != 2 || servers[0] != "codegraph" {
		t.Fatalf("first backup servers = %v", first["servers"])
	}
}

func TestMCPWriteRestorePreviewTextAndJSON(t *testing.T) {
	stub := installMCPWriteStub(t)
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "restore", "preview", mcpWriteBackupID, "--home", home,
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"MCP restore preview — entry-level, read-only",
		"Backup:      " + mcpWriteBackupID,
		"Target:      " + mcpWriteConfigPath,
		"Base digest: " + mcpWriteBaseDigest,
		"Result:      restore ready; apply with --expect-config " + mcpWriteBaseDigest,
		"Entries (1):",
		"change",
		"current:",
		"restore:",
		"Conflicts:   none",
		"Token:       " + mcpWriteToken,
		"Preview only. No write.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "\"mcp\":") {
		t.Fatalf("restore preview dumped whole-file content:\n%s", stdout)
	}
	if len(stub.restorePreviewCalls) != 1 {
		t.Fatalf("restore preview calls = %d, want 1", len(stub.restorePreviewCalls))
	}
	if got := stub.restorePreviewCalls[0]; got.backupID != mcpWriteBackupID || got.home != home {
		t.Fatalf("restore preview request = %+v", got)
	}
	if len(stub.restoreApplyCalls) != 0 {
		t.Fatalf("restore preview wrote: %v", stub.restoreApplyCalls)
	}

	code, stdout, stderr = runMCPWrite([]string{
		"mcp", "restore", "preview", mcpWriteBackupID, "--home", home, "--json",
	}, "", false)
	if code != 0 {
		t.Fatalf("JSON exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	for _, key := range []string{
		"command", "mode", "home", "backup_id", "target", "base_digest",
		"entries", "conflicts", "token", "preview_only", "write",
	} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("restore preview JSON missing key %q: %s", key, stdout)
		}
	}
	if doc["command"] != "restore preview" || doc["mode"] != "read-only" {
		t.Fatalf("restore preview identity = %v/%v", doc["command"], doc["mode"])
	}
	if doc["preview_only"] != true || doc["write"] != "none" || doc["token"] != mcpWriteToken {
		t.Fatalf("restore preview JSON does not declare a token-bound read: %s", stdout)
	}
	entries, ok := doc["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("entries = %v, want 1", doc["entries"])
	}
}

func TestMCPWriteRestorePreviewConflict(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.restorePreview = mcpWriteRestorePreviewFixture()
	stub.restorePreview.Conflicts = []string{"codegraph was edited after this backup"}
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "restore", "preview", mcpWriteBackupID, "--home", home,
	}, "", false)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Result:      refused until the conflicts below are resolved") {
		t.Fatalf("stdout %q hides the refusal", stdout)
	}
	if !strings.Contains(stdout, "Conflicts:   codegraph was edited after this backup") {
		t.Fatalf("stdout %q hides the conflict detail", stdout)
	}
	for _, want := range []string{
		"error: this backup cannot be restored without overwriting later edits",
		"the conflicting entries above stay untouched",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if len(stub.restoreApplyCalls) != 0 {
		t.Fatalf("a conflicted restore preview wrote: %v", stub.restoreApplyCalls)
	}
}

func TestMCPWriteRestoreApplyRequiresIDAndDigest(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing id",
			args: []string{"mcp", "restore", "apply", "--expect-config", mcpWriteBaseDigest},
			want: "mcp restore apply requires a backup id",
		},
		{
			name: "missing digest",
			args: []string{"mcp", "restore", "apply", mcpWriteBackupID},
			want: "mcp restore apply requires --expect-config",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := installMCPWriteStub(t)
			home := t.TempDir()
			code, stdout, stderr := runMCPWrite(append(tc.args, "--home", home), "y\n", true)
			if code != 2 {
				t.Fatalf("exit %d, want 2; stdout %q; stderr %q", code, stdout, stderr)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Fatalf("stderr %q does not contain %q", stderr, tc.want)
			}
			if strings.Contains(stdout, "[y/N]") {
				t.Fatalf("an incomplete restore apply prompted:\n%s", stdout)
			}
			if len(stub.restorePreviewCalls) != 0 || len(stub.restoreApplyCalls) != 0 {
				t.Fatalf("an incomplete restore apply reached the engine: preview=%v apply=%v",
					stub.restorePreviewCalls, stub.restoreApplyCalls)
			}
		})
	}
}

func TestMCPWriteRestoreApplyStaleDigest(t *testing.T) {
	stub := installMCPWriteStub(t)
	home := t.TempDir()

	// stdin says yes: the digest check must refuse before any prompt or write.
	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "restore", "apply", mcpWriteBackupID,
		"--home", home, "--expect-config", "sha256:stale0000",
	}, "y\n", true)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"error: stale restore: --expect-config sha256:stale0000 does not match the current configuration digest " + mcpWriteBaseDigest,
		"run \"asdx-core mcp restore preview\" again and review the new digest",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if strings.Contains(stdout, "[y/N]") {
		t.Fatalf("a stale restore still prompted:\n%s", stdout)
	}
	if len(stub.restoreApplyCalls) != 0 {
		t.Fatalf("a stale restore wrote: %v", stub.restoreApplyCalls)
	}
}

func TestMCPWriteRestoreApplyConflict(t *testing.T) {
	stub := installMCPWriteStub(t)
	stub.restorePreview = mcpWriteRestorePreviewFixture()
	stub.restorePreview.Conflicts = []string{"codegraph was edited after this backup"}
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "restore", "apply", mcpWriteBackupID,
		"--home", home, "--expect-config", mcpWriteBaseDigest,
	}, "y\n", true)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	for _, want := range []string{
		"error: mcp restore apply refused: the preview reports a conflict",
		"conflict: codegraph was edited after this backup",
		"later edits to a managed entry are never erased",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not contain %q", stderr, want)
		}
	}
	if strings.Contains(stdout, "[y/N]") {
		t.Fatalf("a conflicted restore still prompted:\n%s", stdout)
	}
	if len(stub.restoreApplyCalls) != 0 {
		t.Fatalf("a conflicted restore wrote: %v", stub.restoreApplyCalls)
	}
}

func TestMCPWriteRestoreApplyPromptDefaultNo(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
	}{
		{name: "blank answer", stdin: "\n"},
		{name: "n", stdin: "n\n"},
		{name: "no", stdin: "no\n"},
		{name: "eof"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := installMCPWriteStub(t)
			mcpPromptIn = strings.NewReader(tc.stdin)
			home := t.TempDir()

			code, stdout, stderr := runMCPWrite([]string{
				"mcp", "restore", "apply", mcpWriteBackupID,
				"--home", home, "--expect-config", mcpWriteBaseDigest,
			}, "", true)
			if code != 0 {
				t.Fatalf("a declined restore exits 0, got %d; stdout %q; stderr %q", code, stdout, stderr)
			}
			if !strings.Contains(stdout, "[y/N]: ") {
				t.Fatalf("the default-No prompt was not shown:\n%s", stdout)
			}
			if !strings.Contains(stdout, "Cancelled.") {
				t.Fatalf("a declined restore is not reported as cancelled:\n%s", stdout)
			}
			if !strings.Contains(stdout, "Nothing was written.") {
				t.Fatalf("a declined restore does not state that nothing was written:\n%s", stdout)
			}
			if strings.Contains(stdout, "Result:      applied") {
				t.Fatalf("a declined restore claimed success:\n%s", stdout)
			}
			if len(stub.restoreApplyCalls) != 0 {
				t.Fatalf("a declined restore wrote: %v", stub.restoreApplyCalls)
			}
		})
	}
}

func TestMCPWriteRestoreApplyPromptYes(t *testing.T) {
	for _, answer := range []string{"y\n", "Y\n", "yes\n"} {
		stub := installMCPWriteStub(t)
		mcpPromptIn = strings.NewReader(answer)
		home := t.TempDir()

		code, stdout, stderr := runMCPWrite([]string{
			"mcp", "restore", "apply", mcpWriteBackupID,
			"--home", home, "--config", mcpWriteConfigPath,
			"--expect-config", mcpWriteBaseDigest,
		}, "", true)
		if code != 0 {
			t.Fatalf("%q: exit %d; stdout %q; stderr %q", answer, code, stdout, stderr)
		}
		for _, want := range []string{
			"[y/N]:",
			"MCP restore apply — digest-checked write",
			"Result:      applied",
			"Applied:     codegraph",
			"Backup:      " + mcpWriteBackupID,
			"Rollback:    not needed",
		} {
			if !strings.Contains(stdout, strings.TrimSpace(want)) {
				t.Fatalf("%q: stdout does not contain %q:\n%s", answer, want, stdout)
			}
		}
		if len(stub.restoreApplyCalls) != 1 {
			t.Fatalf("%q: restore apply calls = %d, want exactly 1", answer, len(stub.restoreApplyCalls))
		}
		call := stub.restoreApplyCalls[0]
		if call.backupID != mcpWriteBackupID || call.token != mcpWriteToken || call.expect != mcpWriteBaseDigest {
			t.Fatalf("%q: restore request = %+v", answer, call)
		}
		if call.home != home || call.config != mcpWriteConfigPath {
			t.Fatalf("%q: restore target = %q/%q", answer, call.home, call.config)
		}
	}
}

func TestMCPWriteRestoreApplyNonInteractive(t *testing.T) {
	stub := installMCPWriteStub(t)
	// A "no" answer would cancel: it must never be read in a scripted run.
	mcpPromptIn = strings.NewReader("n\n")
	home := t.TempDir()

	code, stdout, stderr := runMCPWrite([]string{
		"mcp", "restore", "apply", mcpWriteBackupID,
		"--home", home, "--expect-config", mcpWriteBaseDigest,
	}, "", false)
	if code != 0 {
		t.Fatalf("exit %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	if strings.Contains(stdout, "[y/N]") || strings.Contains(stdout, "Cancelled") {
		t.Fatalf("a scripted restore prompted:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Result:      applied") {
		t.Fatalf("a scripted restore did not apply:\n%s", stdout)
	}
	if len(stub.restoreApplyCalls) != 1 {
		t.Fatalf("restore apply calls = %d, want exactly 1", len(stub.restoreApplyCalls))
	}

	stub = installMCPWriteStub(t)
	mcpPromptIn = strings.NewReader("n\n")
	code, stdout, stderr = runMCPWrite([]string{
		"mcp", "restore", "apply", mcpWriteBackupID,
		"--home", home, "--expect-config", mcpWriteBaseDigest, "--json",
	}, "", false)
	if code != 0 {
		t.Fatalf("JSON exit %d; stdout %q; stderr %q", code, stdout, stderr)
	}
	// A scripted --json run emits one pure JSON document: no prompt, no preview.
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	if doc["command"] != "restore apply" || doc["mode"] != "write" {
		t.Fatalf("JSON identity = %v/%v", doc["command"], doc["mode"])
	}
	if doc["backup_id"] != mcpWriteBackupID || doc["no_op"] != false {
		t.Fatalf("JSON result = %v", doc)
	}
}

// --- output hygiene -------------------------------------------------------

// TestMCPWriteNoCredentialsOrFullConfig pins that the write surface prints a
// plan/result projection only: no environment credential is echoed, no JSON
// field carries secret material and the whole configuration is never dumped.
func TestMCPWriteNoCredentialsOrFullConfig(t *testing.T) {
	const (
		context7Secret = "sk-live-context7-should-never-print"
		exaSecret      = "exa-secret-should-never-print"
		openCodeSecret = "opencode-secret-should-never-print"
	)
	t.Setenv("CONTEXT7_API_KEY", context7Secret)
	t.Setenv("EXA_API_KEY", exaSecret)
	t.Setenv("OPENCODE_API_KEY", openCodeSecret)
	installMCPWriteStub(t)
	home := t.TempDir()

	runs := [][]string{
		{"mcp", "preview", "--home", home},
		{"mcp", "preview", "--home", home, "--json"},
		{"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest},
		{"mcp", "apply", "--home", home, "--expect-config", mcpWriteBaseDigest, "--json"},
		{"mcp", "restore", "list", "--home", home},
		{"mcp", "restore", "list", "--home", home, "--json"},
		{"mcp", "restore", "preview", mcpWriteBackupID, "--home", home},
		{"mcp", "restore", "preview", mcpWriteBackupID, "--home", home, "--json"},
		{"mcp", "restore", "apply", mcpWriteBackupID, "--home", home, "--expect-config", mcpWriteBaseDigest},
		{"mcp", "restore", "apply", mcpWriteBackupID, "--home", home, "--expect-config", mcpWriteBaseDigest, "--json"},
	}
	for _, args := range runs {
		code, stdout, stderr := runMCPWrite(args, "", false)
		if code != 0 {
			t.Fatalf("%v: exit %d; stderr %q", args, code, stderr)
		}
		combined := stdout + stderr
		for _, secret := range []string{context7Secret, exaSecret, openCodeSecret} {
			if strings.Contains(combined, secret) {
				t.Fatalf("%v leaked an environment credential:\n%s", args, combined)
			}
		}
		for _, marker := range []string{"Bearer ", "Authorization", "$schema", "\"permission\"", "\"agents\""} {
			if strings.Contains(combined, marker) {
				t.Fatalf("%v printed %q, which only a whole-config dump would carry:\n%s", args, marker, combined)
			}
		}
		if strings.Contains(stdout, "--json") {
			assertMCPNoCredentialKeys(t, decodeMCPJSON(t, stdout))
		}
	}
}

// assertMCPNoCredentialKeys fails when a JSON document exposes credential-shaped
// key names; the plan projection carries entry text and digests, never secrets.
func assertMCPNoCredentialKeys(t *testing.T, doc map[string]any) {
	t.Helper()
	var walk func(value any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				lower := strings.ToLower(key)
				for _, banned := range []string{"api_key", "secret", "credential", "password", "authorization", "bearer"} {
					if strings.Contains(lower, banned) {
						t.Fatalf("JSON key %q looks like credential material", key)
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(doc)
}
