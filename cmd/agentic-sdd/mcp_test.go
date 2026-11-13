package main

// Feature 015 MCP-A / T29 — CLI contract tests for `agentic-sdd mcp`.
//
// These tests pin the CLI surface only: dispatch, flag grammar, deferred and
// unauthorized commands, exit codes, deterministic text, stable JSON keys and
// the absence of any write or terminal dependency. The engine behavior behind
// internal/appops (configuration detection, state projection, diagnostics) is
// covered by the T26–T28 tests in their own packages; here the engine seams are
// replaced so the CLI can be exercised without a real OpenCode home, a network
// call or a process start.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
)

type mcpCall struct {
	home         string
	config       string
	project      string
	opencodeBin  string
	codegraphBin string
	servers      []string
}

type mcpStub struct {
	calls []mcpCall

	statusRes  appops.MCPStatusResult
	statusErr  error
	doctorRes  appops.MCPDoctorResult
	doctorErr  error
	previewRes appops.MCPPreviewResult
	previewErr error
	plan       *skillsync.MCPPlan
	planToken  string
	planErr    error

	// requestErr, when set, makes every request constructor fail, so the CLI's
	// validation-to-exit-2 mapping is testable without a bad real config.
	requestErr error
}

func (s *mcpStub) install(t *testing.T) {
	t.Helper()
	prevStatusReq, prevDoctorReq, prevPreviewReq := mcpNewStatusRequest, mcpNewDoctorRequest, mcpNewPreviewRequest
	prevStatus, prevDoctor, prevPreview := mcpStatusReport, mcpDoctorReport, mcpPreviewReport
	prevPlanPreview := mcpPlanPreview
	t.Cleanup(func() {
		mcpNewStatusRequest, mcpNewDoctorRequest, mcpNewPreviewRequest = prevStatusReq, prevDoctorReq, prevPreviewReq
		mcpStatusReport, mcpDoctorReport, mcpPreviewReport = prevStatus, prevDoctor, prevPreview
		mcpPlanPreview = prevPlanPreview
	})

	mcpNewStatusRequest = func(home, config, project string) (appops.MCPStatusRequest, error) {
		s.calls = append(s.calls, mcpCall{home: home, config: config, project: project})
		if s.requestErr != nil {
			return appops.MCPStatusRequest{}, s.requestErr
		}
		return appops.MCPStatusRequest{}, nil
	}
	mcpNewDoctorRequest = func(home, config, project, opencodeBin, codegraphBin string) (appops.MCPDoctorRequest, error) {
		s.calls = append(s.calls, mcpCall{home: home, config: config, project: project, opencodeBin: opencodeBin, codegraphBin: codegraphBin})
		if s.requestErr != nil {
			return appops.MCPDoctorRequest{}, s.requestErr
		}
		return appops.MCPDoctorRequest{}, nil
	}
	mcpNewPreviewRequest = func(home, config string, servers []string) (appops.MCPPreviewRequest, error) {
		s.calls = append(s.calls, mcpCall{home: home, config: config, servers: append([]string(nil), servers...)})
		if s.requestErr != nil {
			return appops.MCPPreviewRequest{}, s.requestErr
		}
		return appops.MCPPreviewRequest{}, nil
	}
	mcpStatusReport = func(appops.MCPStatusRequest) (appops.MCPStatusResult, error) {
		return s.statusRes, s.statusErr
	}
	mcpDoctorReport = func(appops.MCPDoctorRequest) (appops.MCPDoctorResult, error) {
		return s.doctorRes, s.doctorErr
	}
	mcpPreviewReport = func(appops.MCPPreviewRequest) (appops.MCPPreviewResult, error) {
		return s.previewRes, s.previewErr
	}
	mcpPlanPreview = func(home, config string, servers []string, auth map[string]string) (*skillsync.MCPPlan, string, error) {
		s.calls = append(s.calls, mcpCall{home: home, config: config, servers: append([]string(nil), servers...)})
		if s.requestErr != nil {
			return nil, "", s.requestErr
		}
		return s.plan, s.planToken, s.planErr
	}
}

func installMCPStub(t *testing.T) *mcpStub {
	t.Helper()
	stub := &mcpStub{
		statusRes:  appops.MCPStatusResult{Status: mcpFixtureStatus()},
		doctorRes:  appops.MCPDoctorResult{Status: mcpFixtureStatus(), Checks: mcpFixtureChecks(), NotChecked: []string{"remote connectivity", "authentication"}},
		previewRes: appops.MCPPreviewResult{Preview: mcpFixturePreview()},
		plan:       mcpFixturePlan(),
		planToken:  "token-abc123",
	}
	stub.install(t)
	return stub
}

func mcpFixtureStatus() *skillsync.MCPStatus {
	configPath := "/tmp/home/.config/opencode/opencode.json"
	return &skillsync.MCPStatus{
		Home:         "/tmp/home",
		ConfigPath:   configPath,
		ConfigFormat: "json",
		Schema:       "v2",
		ConfigDigest: "sha256:abc123",
		Servers: []skillsync.MCPServerStatus{
			{
				Name:            "codegraph",
				Kind:            "local",
				Purpose:         "Repository code intelligence",
				State:           "configured (global)",
				Evidence:        "static",
				Prerequisite:    "codegraph CLI",
				PrerequisiteMet: true,
				AuthMode:        "none",
				IndexState:      "present (freshness not checked)",
				ConfigPath:      configPath,
			},
			{
				Name:     "context7",
				Kind:     "remote",
				Purpose:  "Hosted library documentation",
				State:    "not configured",
				Evidence: "static",
				AuthMode: "not checked",
			},
			{
				Name:     "exa",
				Kind:     "remote",
				Purpose:  "Hosted web research",
				State:    "not configured",
				Evidence: "static",
				AuthMode: "not checked",
			},
		},
		CodeGraphBinary:  "/usr/local/bin/codegraph",
		CodeGraphVersion: "1.6.2",
		OpenCodeVersion:  "not checked (offline)",
		NotChecked:       []string{"remote connectivity", "authentication", "runtime version"},
	}
}

func mcpFixtureChecks() []appops.MCPDoctorCheck {
	return []appops.MCPDoctorCheck{
		{Name: "config-target", State: "pass", Detail: "global OpenCode configuration found"},
		{Name: "codegraph-cli", State: "warn", Detail: "codegraph CLI not found on PATH"},
		{Name: "codegraph-index", State: "not-checked", Detail: "project index freshness was not probed"},
		{Name: "schema", State: "blocked", Detail: "V1 mcp.<name> schema requires manual migration"},
	}
}

func mcpFixturePreview() *skillsync.MCPPreview {
	return &skillsync.MCPPreview{
		Home:       "/tmp/home",
		TargetPath: "/tmp/home/.config/opencode/opencode.json",
		Format:     "json",
		BaseDigest: "sha256:abc123",
		Selected:   []string{"codegraph", "context7", "exa"},
		Entries: []skillsync.MCPPreviewEntry{
			{
				Name:   "codegraph",
				Op:     "add",
				Reason: "server is missing from the configuration",
				Recipe: "{\n  \"type\": \"local\",\n  \"command\": [\"codegraph\", \"serve\", \"--mcp\"]\n}",
			},
			{
				Name:   "context7",
				Op:     "leave",
				Reason: "already configured and unchanged",
			},
		},
		Conflicts:  []string{"project config overrides context7"},
		NotChecked: []string{"remote connectivity", "authentication"},
	}
}

func mcpFixturePlan() *skillsync.MCPPlan {
	return &skillsync.MCPPlan{
		TargetPath:      "/tmp/home/.config/opencode/opencode.json",
		Format:          "json",
		Schema:          "v2",
		BaseDigest:      "sha256:abc123",
		PlanDigest:      "sha256:plan123",
		Selected:        []string{"codegraph", "context7", "exa"},
		AuthChoices:     map[string]string{"context7": "oauth", "exa": "keyless"},
		RuntimeVersion:  "not-checked",
		RuntimeEvidence: "not-checked",
		NotChecked:      []string{"remote connectivity", "authentication"},
		Entries: []skillsync.MCPPlanEntry{
			{
				Name:    "codegraph",
				Op:      "add",
				Reason:  "server is missing from the configuration",
				NewText: "{\n  \"type\": \"local\",\n  \"command\": [\"codegraph\", \"serve\", \"--mcp\"]\n}",
				NewHash: "hash-codegraph",
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

// runMCPCommand dispatches through the full CLI entry point with empty,
// non-terminal stdin, so dispatch wiring in main.go is covered too.
func runMCPCommand(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = runNonInteractive(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestMCPDispatch(t *testing.T) {
	installMCPStub(t)

	tests := []struct {
		name    string
		args    []string
		code    int
		wantOut []string
		wantErr []string
	}{
		{
			name:    "no subcommand is status",
			args:    []string{"mcp"},
			code:    0,
			wantOut: []string{"MCP status", "codegraph", "configured (global)"},
		},
		{
			name:    "explicit status",
			args:    []string{"mcp", "status"},
			code:    0,
			wantOut: []string{"MCP status"},
		},
		{
			name:    "status json",
			args:    []string{"mcp", "status", "--json"},
			code:    0,
			wantOut: []string{`"command": "status"`, `"mode": "offline"`},
		},
		{
			name:    "front flag is status",
			args:    []string{"mcp", "--json"},
			code:    0,
			wantOut: []string{`"command": "status"`},
		},
		{
			name:    "doctor",
			args:    []string{"mcp", "doctor"},
			code:    0,
			wantOut: []string{"MCP doctor", "Offline only; remote connectivity, authentication and runtime version were not checked."},
		},
		{
			name:    "doctor json",
			args:    []string{"mcp", "doctor", "--json"},
			code:    0,
			wantOut: []string{`"command": "doctor"`, `"checks"`, `"offline": true`},
		},
		{
			name:    "preview",
			args:    []string{"mcp", "preview"},
			code:    0,
			wantOut: []string{"MCP preview — structured plan", "Preview only. No write."},
		},
		{
			name:    "preview json",
			args:    []string{"mcp", "preview", "--json"},
			code:    0,
			wantOut: []string{`"command": "preview"`, `"preview_only": true`, `"token": "token-abc123"`},
		},
		{
			name:    "help",
			args:    []string{"mcp", "help"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp", "MCP-A + MCP-B", "apply"},
		},
		{
			name:    "help flag",
			args:    []string{"mcp", "--help"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp"},
		},
		{
			name:    "help short flag",
			args:    []string{"mcp", "-h"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp"},
		},
		{
			name:    "unknown subcommand",
			args:    []string{"mcp", "bogus"},
			code:    2,
			wantErr: []string{"error: unknown mcp command \"bogus\"", "status, doctor, preview, apply, restore and help"},
		},
		{
			name:    "apply requires expect config",
			args:    []string{"mcp", "apply"},
			code:    2,
			wantErr: []string{"mcp apply requires --expect-config"},
		},
		{
			name:    "restore requires subcommand",
			args:    []string{"mcp", "restore"},
			code:    2,
			wantErr: []string{"mcp restore requires a subcommand"},
		},
		{
			name:    "doctor network rejected",
			args:    []string{"mcp", "doctor", "--network"},
			code:    2,
			wantErr: []string{"--network is not available in this build", "deferred out of MCP-A"},
		},
		{
			name:    "doctor connect rejected",
			args:    []string{"mcp", "doctor", "--connect"},
			code:    2,
			wantErr: []string{"--connect is not available in this build"},
		},
		{
			name:    "status network is an unknown flag",
			args:    []string{"mcp", "status", "--network"},
			code:    2,
			wantErr: []string{"flag provided but not defined: -network"},
		},
		{
			name:    "unexpected positional",
			args:    []string{"mcp", "status", "extra"},
			code:    2,
			wantErr: []string{`error: unexpected argument "extra"`},
		},
		{
			name:    "unexpected preview positional",
			args:    []string{"mcp", "preview", "extra"},
			code:    2,
			wantErr: []string{`error: unexpected argument "extra"`},
		},
		{
			name:    "unknown flag",
			args:    []string{"mcp", "preview", "--bogus"},
			code:    2,
			wantErr: []string{"flag provided but not defined: -bogus"},
		},
		{
			name:    "empty server list",
			args:    []string{"mcp", "preview", "--servers", " , "},
			code:    2,
			wantErr: []string{"--servers requires at least one server name"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runMCPCommand(tc.args...)
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; stderr %q", code, tc.code, stdout, stderr)
			}
			if len(tc.wantErr) == 0 && stderr != "" {
				t.Fatalf("unexpected stderr %q", stderr)
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
		})
	}
}

func TestMCPStatusText(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "status", "--home", "/tmp/home")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"MCP status — offline, read-only (no network, no writes)",
		"Home:          /tmp/home",
		"Configuration: /tmp/home/.config/opencode/opencode.json",
		"Format:        json",
		"Schema:        v2",
		"Digest:        sha256:abc123",
		"Servers (3):",
		"codegraph",
		"configured (global)",
		"prereq: codegraph CLI (found)",
		"auth: none",
		"evidence: static",
		"index: present (freshness not checked)",
		"auth: not checked",
		"Not checked:   remote connectivity, authentication, runtime version",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "context7 API key") || strings.Contains(stdout, "Bearer") {
		t.Fatalf("status text leaked credential-shaped material:\n%s", stdout)
	}
}

func TestMCPStatusNotConfigured(t *testing.T) {
	stub := installMCPStub(t)
	notConfigured := &skillsync.MCPStatus{
		Home: "/tmp/home",
		Servers: []skillsync.MCPServerStatus{
			{Name: "codegraph", Kind: "local", State: "not configured", Evidence: "static", AuthMode: "none"},
		},
		NotChecked: []string{"remote connectivity", "authentication"},
	}
	stub.statusRes = appops.MCPStatusResult{Status: notConfigured}

	code, stdout, stderr := runMCPCommand("mcp", "status", "--home", "/tmp/home")
	if code != 0 {
		t.Fatalf("not-configured status must succeed, exit %d; stderr %q", code, stderr)
	}
	if !strings.Contains(stdout, "not configured (no global OpenCode configuration found)") {
		t.Fatalf("stdout %q missing not-configured target", stdout)
	}
	if strings.Contains(stdout, "Digest:") {
		t.Fatalf("stdout %q should not invent a digest without a target", stdout)
	}
}

func TestMCPStatusJSON(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "status", "--home", "/tmp/home", "--json")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	for _, key := range []string{"command", "mode", "home", "config", "servers", "codegraph_binary", "codegraph_version", "opencode_version", "not_checked"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("status JSON missing key %q: %s", key, stdout)
		}
	}
	assertMCPLowerKeys(t, doc)
	for _, want := range []string{`"prerequisite_met": true`, `"auth": "none"`, `"state": "configured (global)"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("status JSON missing %q:\n%s", want, stdout)
		}
	}
	servers, ok := doc["servers"].([]any)
	if !ok || len(servers) != 3 {
		t.Fatalf("servers = %v, want 3 entries", doc["servers"])
	}
	first, ok := servers[0].(map[string]any)
	if !ok || first["name"] != "codegraph" {
		t.Fatalf("first server = %v, want codegraph", servers[0])
	}
}

func TestMCPDoctorText(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "doctor", "--home", "/tmp/home")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"MCP doctor — offline diagnostics (no network, no writes)",
		"Checks (4):",
		"config-target",
		"codegraph-cli",
		"codegraph-index",
		"schema",
		"[pass",
		"[warn",
		"[blocked",
		"[not-checked",
		"Servers:       codegraph=configured (global), context7=not configured, exa=not configured",
		"Offline only; remote connectivity, authentication and runtime version were not checked.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	// The final line must be the last thing printed so scripts can rely on it.
	if !strings.HasSuffix(strings.TrimRight(stdout, "\n"), "Offline only; remote connectivity, authentication and runtime version were not checked.") {
		t.Fatalf("doctor report does not end with the offline disclaimer:\n%s", stdout)
	}
}

func TestMCPDoctorJSON(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "doctor", "--home", "/tmp/home", "--json")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	checks, ok := doc["checks"].([]any)
	if !ok || len(checks) != 4 {
		t.Fatalf("checks = %v, want 4 entries", doc["checks"])
	}
	check, ok := checks[3].(map[string]any)
	if !ok || check["name"] != "schema" || check["state"] != "blocked" {
		t.Fatalf("last check = %v, want schema/blocked", checks[3])
	}
	if doc["offline"] != true {
		t.Fatalf("offline = %v, want true", doc["offline"])
	}
	if got := doc["not_checked"].([]any); len(got) != 2 {
		t.Fatalf("not_checked = %v, want the doctor list", doc["not_checked"])
	}
}

func TestMCPPreviewText(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "preview", "--home", "/tmp/home")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	for _, want := range []string{
		"MCP preview — structured plan (read-only, no network, no writes)",
		"Target:      /tmp/home/.config/opencode/opencode.json",
		"Format:      json",
		"Base digest: sha256:abc123",
		"Plan digest: sha256:plan123",
		"Selected:    codegraph, context7, exa",
		"Auth:        context7=oauth, exa=keyless",
		"Entries (2):",
		"add",
		"leave",
		"planned:",
		"\"type\": \"local\"",
		"\"codegraph\"",
		"Token:       token-abc123",
		"Preview only. No write.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
}

func TestMCPPreviewJSON(t *testing.T) {
	installMCPStub(t)

	code, stdout, stderr := runMCPCommand("mcp", "preview", "--home", "/tmp/home", "--json")
	if code != 0 {
		t.Fatalf("exit %d; stderr %q", code, stderr)
	}
	doc := decodeMCPJSON(t, stdout)
	assertMCPLowerKeys(t, doc)
	for _, key := range []string{"command", "mode", "home", "target", "format", "schema", "base_digest", "plan_digest", "selected", "auth", "entries", "conflicts", "no_op", "runtime_version", "runtime_evidence", "not_checked", "token", "preview_only", "write"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("preview JSON missing key %q: %s", key, stdout)
		}
	}
	entries, ok := doc["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("entries = %v, want 2", doc["entries"])
	}
	entry, ok := entries[0].(map[string]any)
	if !ok || entry["name"] != "codegraph" || entry["op"] != "add" {
		t.Fatalf("first entry = %v, want codegraph/add", entries[0])
	}
}

func TestMCPEngineError(t *testing.T) {
	stub := installMCPStub(t)
	stub.statusErr = errors.New("engine unavailable")
	stub.doctorErr = errors.New("engine unavailable")
	stub.previewErr = errors.New("engine unavailable")
	stub.planErr = errors.New("engine unavailable")

	for _, args := range [][]string{
		{"mcp", "status"},
		{"mcp", "doctor"},
		{"mcp", "preview"},
	} {
		code, stdout, stderr := runMCPCommand(append(args, "--home", "/tmp/home")...)
		if code != 1 {
			t.Fatalf("%v: exit %d, want 1; stdout %q; stderr %q", args, code, stdout, stderr)
		}
		if stdout != "" {
			t.Fatalf("%v: stdout = %q, want empty", args, stdout)
		}
		if !strings.Contains(stderr, "error: engine unavailable") {
			t.Fatalf("%v: stderr = %q, want engine error", args, stderr)
		}
	}
}

func TestMCPMissingReportIsAnError(t *testing.T) {
	stub := installMCPStub(t)
	stub.statusRes = appops.MCPStatusResult{}

	code, stdout, stderr := runMCPCommand("mcp", "status", "--home", "/tmp/home")
	if code != 1 {
		t.Fatalf("exit %d, want 1; stdout %q; stderr %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "returned no report") {
		t.Fatalf("stderr = %q, want missing-report error", stderr)
	}
}

func TestMCPRequestValidationError(t *testing.T) {
	stub := installMCPStub(t)
	stub.requestErr = skillsync.ErrMCPUsage

	code, stdout, stderr := runMCPCommand("mcp", "preview", "--servers", "bogus", "--home", "/tmp/home")
	if code != 2 {
		t.Fatalf("exit %d, want 2; stdout %q; stderr %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "MCP usage error") {
		t.Fatalf("stderr = %q, want request validation error", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestMCPRequestArguments(t *testing.T) {
	stub := installMCPStub(t)
	home := t.TempDir()

	code, _, stderr := runMCPCommand("mcp", "preview", "--home", home, "--config", "/tmp/opencode.json", "--servers", "codegraph, exa")
	if code != 0 {
		t.Fatalf("preview exit %d; stderr %q", code, stderr)
	}
	preview := stub.calls[len(stub.calls)-1]
	if preview.home != home {
		t.Fatalf("preview home = %q, want %q", preview.home, home)
	}
	if preview.config != "/tmp/opencode.json" {
		t.Fatalf("preview config = %q", preview.config)
	}
	if strings.Join(preview.servers, ",") != "codegraph,exa" {
		t.Fatalf("preview servers = %v, want [codegraph exa]", preview.servers)
	}

	code, _, stderr = runMCPCommand("mcp", "doctor", "--home", home, "--project", "/tmp/project", "--opencode-bin", "/bin/opencode", "--codegraph-bin", "/bin/codegraph")
	if code != 0 {
		t.Fatalf("doctor exit %d; stderr %q", code, stderr)
	}
	doctor := stub.calls[len(stub.calls)-1]
	if doctor.home != home || doctor.project != "/tmp/project" || doctor.opencodeBin != "/bin/opencode" || doctor.codegraphBin != "/bin/codegraph" {
		t.Fatalf("doctor request = %+v", doctor)
	}

	code, _, stderr = runMCPCommand("mcp", "status", "--home", home, "--project", "/tmp/project")
	if code != 0 {
		t.Fatalf("status exit %d; stderr %q", code, stderr)
	}
	status := stub.calls[len(stub.calls)-1]
	if status.home != home || status.project != "/tmp/project" {
		t.Fatalf("status request = %+v", status)
	}
}

func TestMCPNoTTYOutputIsDeterministic(t *testing.T) {
	installMCPStub(t)
	home := t.TempDir()

	for _, args := range [][]string{
		{"mcp", "status", "--home", home},
		{"mcp", "status", "--home", home, "--json"},
		{"mcp", "doctor", "--home", home},
		{"mcp", "preview", "--home", home},
	} {
		first, outOne, errOne := runMCPCommand(args...)
		second, outTwo, errTwo := runMCPCommand(args...)
		if first != second || outOne != outTwo || errOne != errTwo {
			t.Fatalf("%v is not deterministic:\n%q/%q/%q\n%q/%q/%q", args, first, outOne, errOne, second, outTwo, errTwo)
		}
		if strings.Contains(outOne, "\x1b[") {
			t.Fatalf("%v emitted ANSI styling on a non-terminal stdout", args)
		}
	}
}

func TestMCPWritesNothing(t *testing.T) {
	installMCPStub(t)
	home := t.TempDir()
	config := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(config), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("{\n  \"mcp\": {\n    \"servers\": {}\n  }\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshotMCPTree(t, home)

	for _, args := range [][]string{
		{"mcp", "status", "--home", home},
		{"mcp", "status", "--home", home, "--json"},
		{"mcp", "doctor", "--home", home},
		{"mcp", "doctor", "--home", home, "--json"},
		{"mcp", "preview", "--home", home},
		{"mcp", "preview", "--home", home, "--json"},
	} {
		code, _, stderr := runMCPCommand(args...)
		if code != 0 {
			t.Fatalf("%v: exit %d; stderr %q", args, code, stderr)
		}
	}

	after := snapshotMCPTree(t, home)
	if len(before) != len(after) {
		t.Fatalf("read-only commands changed the tree: before %v after %v", before, after)
	}
	for path, want := range before {
		if got, ok := after[path]; !ok || got != want {
			t.Fatalf("%s changed: %q -> %q", path, want, got)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agentic-sdd", "mcp")); !os.IsNotExist(err) {
		t.Fatalf("MCP-A created private MCP state under .agentic-sdd/mcp: %v", err)
	}
}

// snapshotMCPTree records every file under root with its content, so a write
// anywhere in the temporary home is detected.
func snapshotMCPTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

// TestMCPHelpAndExistingCommands pins that `help mcp` and the additive usage
// line exist without disturbing the commands that were already there.
func TestMCPHelpAndExistingCommands(t *testing.T) {
	installMCPStub(t)
	home := t.TempDir()

	tests := []struct {
		name    string
		args    []string
		code    int
		wantOut []string
		wantErr []string
	}{
		{
			name:    "help mcp",
			args:    []string{"help", "mcp"},
			code:    0,
			wantOut: []string{"Usage: asdx-core mcp", "MCP-A + MCP-B", "mcp apply", "restore"},
		},
		{
			name:    "help lists mcp and fleet",
			args:    []string{"help"},
			code:    0,
			wantOut: []string{"asdx-core mcp status|doctor|preview", "asdx-core fleet status|plan|apply|restore|doctor", "asdx-core help [preview|apply|backups|logs|restore|tui|fleet|mcp|run|task|review]"},
		},
		{
			name:    "help fleet unchanged",
			args:    []string{"help", "fleet"},
			code:    0,
			wantOut: []string{"Usage: asdx-core fleet"},
		},
		{
			name:    "bare fleet unchanged",
			args:    []string{"fleet"},
			code:    2,
			wantErr: []string{"Usage: asdx-core fleet"},
		},
		{
			name:    "tui help unchanged",
			args:    []string{"tui", "--help"},
			code:    0,
			wantOut: []string{"Usage: asdx-core"},
		},
		{
			name:    "preview still defaults to skills",
			args:    []string{"preview", "--home", home},
			code:    0,
			wantOut: []string{"Preview only. Run with apply (or --apply) to install."},
		},
		{
			name:    "unknown command unchanged",
			args:    []string{"wat"},
			code:    2,
			wantErr: []string{"error: unknown command \"wat\""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runMCPCommand(tc.args...)
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
		})
	}
}

func decodeMCPJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return doc
}

// assertMCPLowerKeys fails when a JSON key is not lower_snake_case, which is
// what makes the output script-compatible across releases.
func assertMCPLowerKeys(t *testing.T, doc map[string]any) {
	t.Helper()
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if key != strings.ToLower(key) {
					t.Fatalf("JSON key %s%s is not lower case", prefix, key)
				}
				walk(prefix+key+".", child)
			}
		case []any:
			for _, child := range typed {
				walk(prefix, child)
			}
		}
	}
	walk("", doc)
}
