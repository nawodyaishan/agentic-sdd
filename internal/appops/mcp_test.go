package appops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
)

// MCP-A tests exercise the read-only seam only. Every fixture is isolated under
// t.TempDir; no test reads a real home, writes an MCP configuration, executes
// opencode/codegraph, contacts Context7/Exa or uses os.UserHomeDir.

const mcpTestConfigEmpty = `{
  "mcp": {
    "servers": {}
  }
}
`

const mcpTestConfigCodeGraph = `{
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"],
        "disabled": false,
        "codemode": false
      }
    }
  }
}
`

const mcpTestConfigLegacy = `{
  "mcp": {
    "codegraph": {
      "type": "local",
      "command": ["codegraph", "serve", "--mcp"],
      "enabled": true
    }
  }
}
`

// mcpTestFixture is an isolated home with an optional default global target.
type mcpTestFixture struct {
	root      string
	home      string
	configDir string
}

func newMCPTestFixture(t *testing.T) mcpTestFixture {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	return mcpTestFixture{root: root, home: home, configDir: configDir}
}

func (f mcpTestFixture) write(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(f.configDir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// assertNoMCPWrite asserts neither the MCP state root nor its backup area was
// created. Read-only operations must leave no trace.
func (f mcpTestFixture) assertNoMCPWrite(t *testing.T) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(f.home, ".agentic-sdd"),
		filepath.Join(f.home, ".agentic-sdd", "mcp"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("read-only MCP operation created %s (stat err: %v)", path, err)
		}
	}
}

func mcpTestCheck(t *testing.T, result MCPDoctorResult, name string) MCPDoctorCheck {
	t.Helper()
	for _, check := range result.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("doctor result has no %q check", name)
	return MCPDoctorCheck{}
}

func TestMCPRequestsRequireHome(t *testing.T) {
	if _, err := NewMCPStatusRequest("", "", ""); err == nil {
		t.Fatal("NewMCPStatusRequest accepted an empty home")
	}
	if _, err := NewMCPDoctorRequest("", "", "", "", ""); err == nil {
		t.Fatal("NewMCPDoctorRequest accepted an empty home")
	}
	if _, err := NewMCPPreviewRequest("", "", nil); err == nil {
		t.Fatal("NewMCPPreviewRequest accepted an empty home")
	}
}

func TestMCPRequestsAbsolutizePaths(t *testing.T) {
	home := filepath.Join("relative", "home")
	config := filepath.Join("relative", "opencode.json")
	project := filepath.Join("relative", "project")
	cache := filepath.Join("relative", "bin", "opencode")
	codegraph := filepath.Join("relative", "bin", "codegraph")

	statusReq, err := NewMCPStatusRequest(home, config, project)
	if err != nil {
		t.Fatal(err)
	}
	doctorReq, err := NewMCPDoctorRequest(home, config, project, cache, codegraph)
	if err != nil {
		t.Fatal(err)
	}
	previewReq, err := NewMCPPreviewRequest(home, config, []string{"codegraph"})
	if err != nil {
		t.Fatal(err)
	}

	want := func(value string) string {
		t.Helper()
		abs, err := filepath.Abs(value)
		if err != nil {
			t.Fatal(err)
		}
		return abs
	}
	for label, got := range map[string]string{
		"status home":          statusReq.Home(),
		"status config":        statusReq.ConfigPath(),
		"status project":       statusReq.ProjectPath(),
		"doctor home":          doctorReq.Home(),
		"doctor config":        doctorReq.ConfigPath(),
		"doctor project":       doctorReq.ProjectPath(),
		"doctor opencode bin":  doctorReq.OpenCodeBin(),
		"doctor codegraph bin": doctorReq.CodeGraphBin(),
		"preview home":         previewReq.Home(),
		"preview config":       previewReq.ConfigPath(),
	} {
		if !filepath.IsAbs(got) {
			t.Fatalf("%s = %q, want an absolute path", label, got)
		}
	}
	if statusReq.Home() != want(home) || statusReq.ConfigPath() != want(config) {
		t.Fatalf("status request did not resolve to the expected absolute paths")
	}
	if doctorReq.CodeGraphBin() != want(codegraph) || doctorReq.OpenCodeBin() != want(cache) {
		t.Fatalf("doctor request did not resolve explicit binaries to absolute paths")
	}
	if previewReq.ConfigPath() != want(config) {
		t.Fatalf("preview request did not resolve its config path")
	}
}

func TestMCPRequestsAllowEmptyOptionalPaths(t *testing.T) {
	home := t.TempDir()
	statusReq, err := NewMCPStatusRequest(home, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if statusReq.ConfigPath() != "" || statusReq.ProjectPath() != "" {
		t.Fatalf("empty optional paths were not preserved: %+v", statusReq)
	}
	doctorReq, err := NewMCPDoctorRequest(home, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if doctorReq.ConfigPath() != "" || doctorReq.ProjectPath() != "" ||
		doctorReq.OpenCodeBin() != "" || doctorReq.CodeGraphBin() != "" {
		t.Fatalf("empty optional doctor paths were not preserved: %+v", doctorReq)
	}
}

func TestMCPPreviewRequestCopiesSelection(t *testing.T) {
	servers := []string{"codegraph", "context7"}
	req, err := NewMCPPreviewRequest(t.TempDir(), "", servers)
	if err != nil {
		t.Fatal(err)
	}
	servers[0] = "exa"
	if got := req.Selected(); len(got) != 2 || got[0] != "codegraph" || got[1] != "context7" {
		t.Fatalf("mutating the caller slice changed the request: %v", got)
	}
	returned := req.Selected()
	returned[1] = "exa"
	if got := req.Selected(); got[1] != "context7" {
		t.Fatalf("mutating the accessor result changed the request: %v", got)
	}
}

func TestMCPPreviewRequestPreservesNilSelection(t *testing.T) {
	req, err := NewMCPPreviewRequest(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Selected() != nil {
		t.Fatalf("nil selection became %#v", req.Selected())
	}
	req, err = NewMCPPreviewRequest(t.TempDir(), "", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Selected(); got == nil || len(got) != 0 {
		t.Fatalf("empty selection became %#v", got)
	}
}

func TestMCPStatusIsReadOnly(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigEmpty)
	req, err := NewMCPStatusRequest(fixture.home, configPath, fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPStatus(req)
	if err != nil {
		t.Fatalf("MCPStatus: %v", err)
	}
	if result.Status == nil {
		t.Fatal("MCPStatus returned no status projection")
	}
	fixture.assertNoMCPWrite(t)
	assertMCPConfigUnchanged(t, configPath, mcpTestConfigEmpty)
}

func TestMCPPreviewIsReadOnlyWithoutToken(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigEmpty)
	req, err := NewMCPPreviewRequest(fixture.home, configPath, []string{"codegraph"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPPreview(req)
	if err != nil {
		t.Fatalf("MCPPreview: %v", err)
	}
	if result.Preview == nil {
		t.Fatal("MCPPreview returned no preview")
	}
	// MCP-A has no token field by construction; this documents the A-only
	// contract that a preview authorizes nothing.
	fixture.assertNoMCPWrite(t)
	assertMCPConfigUnchanged(t, configPath, mcpTestConfigEmpty)
}

func TestMCPDoctorCheckOrderAndVocabulary(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigEmpty)
	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor: %v", err)
	}
	want := []string{
		"config-target",
		"config-schema",
		"codegraph-binary",
		"codegraph-index",
		"context7-auth",
		"exa-auth",
		"opencode-runtime",
		"remote-connectivity",
	}
	if len(result.Checks) != len(want) {
		t.Fatalf("got %d checks, want %d: %+v", len(result.Checks), len(want), result.Checks)
	}
	allowed := map[string]bool{
		mcpCheckPass: true, mcpCheckWarn: true, mcpCheckBlocked: true, mcpCheckNotChecked: true,
	}
	for i, check := range result.Checks {
		if check.Name != want[i] {
			t.Fatalf("check %d = %q, want %q", i, check.Name, want[i])
		}
		if !allowed[check.State] {
			t.Fatalf("check %q has unsupported state %q", check.Name, check.State)
		}
		if strings.TrimSpace(check.Detail) == "" {
			t.Fatalf("check %q has no detail", check.Name)
		}
	}
	if got := mcpTestCheck(t, result, "remote-connectivity").State; got != mcpCheckNotChecked {
		t.Fatalf("remote connectivity reported %q, want not-checked", got)
	}
	if got := mcpTestCheck(t, result, "opencode-runtime").State; got != mcpCheckNotChecked {
		t.Fatalf("runtime version reported %q, want not-checked", got)
	}
	if len(result.NotChecked) == 0 {
		t.Fatal("doctor did not list any not-checked evidence class")
	}
	seen := map[string]bool{}
	for _, value := range result.NotChecked {
		if value == "" {
			t.Fatal("not-checked list contains an empty entry")
		}
		if seen[value] {
			t.Fatalf("not-checked list repeats %q", value)
		}
		seen[value] = true
	}
	fixture.assertNoMCPWrite(t)
}

func TestMCPDoctorMissingConfig(t *testing.T) {
	fixture := newMCPTestFixture(t)
	req, err := NewMCPDoctorRequest(fixture.home, "", fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor with no configuration: %v", err)
	}
	if got := mcpTestCheck(t, result, "config-target").State; got != mcpCheckWarn {
		t.Fatalf("missing configuration target reported %q, want warn", got)
	}
	if got := mcpTestCheck(t, result, "config-schema").State; got != mcpCheckNotChecked {
		t.Fatalf("schema with no configuration reported %q, want not-checked", got)
	}
	if got := mcpTestCheck(t, result, "context7-auth").State; got == mcpCheckPass {
		t.Fatal("Context7 authentication passed without evidence")
	}
	if got := mcpTestCheck(t, result, "exa-auth").State; got == mcpCheckPass {
		t.Fatal("Exa authentication passed without evidence")
	}
	fixture.assertNoMCPWrite(t)
}

func TestMCPDoctorAmbiguityBlocked(t *testing.T) {
	fixture := newMCPTestFixture(t)
	fixture.write(t, "opencode.json", mcpTestConfigEmpty)
	fixture.write(t, "opencode.jsonc", mcpTestConfigEmpty)
	req, err := NewMCPDoctorRequest(fixture.home, "", fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor with both targets: %v", err)
	}
	if got := mcpTestCheck(t, result, "config-target"); got.State != mcpCheckBlocked {
		t.Fatalf("ambiguous targets reported %q, want blocked (%s)", got.State, got.Detail)
	}
	fixture.assertNoMCPWrite(t)
}

func TestMCPDoctorLegacySchemaBlocked(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigLegacy)
	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor with V1 configuration: %v", err)
	}
	if got := mcpTestCheck(t, result, "config-schema"); got.State != mcpCheckBlocked {
		t.Fatalf("V1 schema reported %q, want blocked (%s)", got.State, got.Detail)
	}
	fixture.assertNoMCPWrite(t)
	assertMCPConfigUnchanged(t, configPath, mcpTestConfigLegacy)
}

func TestMCPDoctorMissingCodeGraphBinary(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigCodeGraph)
	missing := filepath.Join(fixture.root, "bin", "codegraph")
	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, "", missing)
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor: %v", err)
	}
	if got := mcpTestCheck(t, result, "codegraph-binary"); got.State != mcpCheckBlocked {
		t.Fatalf("missing explicit CodeGraph binary reported %q, want blocked", got.State)
	}
	if got := mcpTestCheck(t, result, "codegraph-index"); got.State == mcpCheckPass {
		t.Fatalf("CodeGraph index passed without freshness evidence: %+v", got)
	}
	fixture.assertNoMCPWrite(t)
}

func TestMCPDoctorIndexAbsentIsNotPass(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", mcpTestConfigCodeGraph)
	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor: %v", err)
	}
	check := mcpTestCheck(t, result, "codegraph-index")
	if check.State == mcpCheckPass {
		t.Fatalf("index check passed without a freshness probe: %+v", check)
	}
	if !strings.Contains(strings.ToLower(check.Detail), "index") {
		t.Fatalf("index detail does not explain index evidence: %q", check.Detail)
	}
}

func TestMCPDoctorAuthNeverPasses(t *testing.T) {
	fixture := newMCPTestFixture(t)
	configPath := fixture.write(t, "opencode.json", `{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp",
        "headers": {"Authorization": "Bearer {env:CONTEXT7_API_KEY}"},
        "disabled": false
      },
      "exa": {
        "type": "remote",
        "url": "https://mcp.exa.ai/mcp",
        "disabled": false
      }
    }
  }
}
`)
	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor: %v", err)
	}
	for _, name := range []string{"context7-auth", "exa-auth"} {
		check := mcpTestCheck(t, result, name)
		if check.State == mcpCheckPass {
			t.Fatalf("%s passed offline: %+v", name, check)
		}
	}
}

func TestMCPDoctorDoesNotExecuteOrLeak(t *testing.T) {
	fixture := newMCPTestFixture(t)
	const secret = "SUPERSECRETTOKEN123"
	config := `{
  "mcp": {
    "servers": {
      "exa": {
        "type": "remote",
        "url": "https://mcp.exa.ai/mcp",
        "disabled": false
      }
    }
  },
  "secretNote": "` + secret + `"
}
`
	configPath := fixture.write(t, "opencode.json", config)

	sentinel := filepath.Join(fixture.root, "executed")
	script := func(name string) string {
		t.Helper()
		path := filepath.Join(fixture.root, name)
		body := "#!/bin/sh\necho executed >> " + sentinel + "\nexit 1\n"
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	openCodeBin := script("opencode")
	codeGraphBin := script("codegraph")

	req, err := NewMCPDoctorRequest(fixture.home, configPath, fixture.root, openCodeBin, codeGraphBin)
	if err != nil {
		t.Fatal(err)
	}
	result, err := MCPDoctor(req)
	if err != nil {
		t.Fatalf("MCPDoctor: %v", err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("offline doctor executed a binary (stat err: %v)", err)
	}
	var details strings.Builder
	for _, check := range result.Checks {
		details.WriteString(check.Detail)
		details.WriteByte('\n')
	}
	rendered := details.String()
	if strings.Contains(rendered, secret) {
		t.Fatal("doctor detail leaked a configuration value")
	}
	if strings.Contains(rendered, fixture.home) {
		t.Fatal("doctor detail leaked a home path")
	}
	fixture.assertNoMCPWrite(t)
}

func TestMCPSourceIsOfflineOnly(t *testing.T) {
	source, err := os.ReadFile("mcp.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{`"os/exec"`, `"net/http"`, `"net"`, "os.UserHomeDir", "os.Getenv"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("internal/appops/mcp.go imports or calls %s; the MCP-A seam must stay offline", forbidden)
		}
	}
}

func assertMCPConfigUnchanged(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("configuration at %s changed:\n%s", path, got)
	}
}

// The semantic tests below build synthetic projections so doctor labeling is
// verified independently of the engine's exact internal state spelling.

func mcpTestStatus(servers ...skillsync.MCPServerStatus) *skillsync.MCPStatus {
	return &skillsync.MCPStatus{Servers: servers}
}

func mcpTestServer(name, state string) skillsync.MCPServerStatus {
	return skillsync.MCPServerStatus{Name: name, State: skillsync.MCPState(state)}
}

func TestMCPDoctorTargetAndSchemaLabels(t *testing.T) {
	if got := mcpConfigTargetCheck(mcpTestStatus()); got.State != mcpCheckWarn {
		t.Fatalf("missing target = %q, want warn", got.State)
	}
	none := mcpTestStatus()
	none.ConfigFormat = "none"
	none.Schema = "none"
	if got := mcpConfigTargetCheck(none); got.State != mcpCheckWarn {
		t.Fatalf("none target = %q, want warn", got.State)
	}
	ambiguous := mcpTestStatus()
	ambiguous.ConfigFormat = "json"
	ambiguous.Ambiguity = "opencode.json and opencode.jsonc both exist"
	if got := mcpConfigTargetCheck(ambiguous); got.State != mcpCheckBlocked {
		t.Fatalf("ambiguous target = %q, want blocked", got.State)
	}
	unsafe := mcpTestStatus(mcpTestServer("codegraph", "unsafe-target"))
	unsafe.ConfigFormat = "json"
	if got := mcpConfigTargetCheck(unsafe); got.State != mcpCheckBlocked {
		t.Fatalf("unsafe target = %q, want blocked", got.State)
	}
	ok := mcpTestStatus()
	ok.ConfigFormat = "json"
	if got := mcpConfigTargetCheck(ok); got.State != mcpCheckPass {
		t.Fatalf("present target = %q, want pass", got.State)
	}

	if got := mcpConfigSchemaCheck(mcpTestStatus()); got.State != mcpCheckNotChecked {
		t.Fatalf("absent schema = %q, want not-checked", got.State)
	}
	if got := mcpConfigSchemaCheck(none); got.State != mcpCheckNotChecked {
		t.Fatalf("none schema = %q, want not-checked", got.State)
	}
	legacy := mcpTestStatus()
	legacy.ConfigFormat = "json"
	legacy.LegacyReason = "mcp.<name> is a V1 shape"
	if got := mcpConfigSchemaCheck(legacy); got.State != mcpCheckBlocked {
		t.Fatalf("V1 schema = %q, want blocked", got.State)
	}
	mismatch := mcpTestStatus(mcpTestServer("codegraph", "schema-mismatch"))
	mismatch.ConfigFormat = "json"
	if got := mcpConfigSchemaCheck(mismatch); got.State != mcpCheckBlocked {
		t.Fatalf("schema mismatch = %q, want blocked", got.State)
	}
	v2 := mcpTestStatus()
	v2.ConfigFormat = "jsonc"
	v2.Schema = "v2"
	if got := mcpConfigSchemaCheck(v2); got.State != mcpCheckPass {
		t.Fatalf("V2 schema = %q, want pass", got.State)
	}
	invalid := mcpTestStatus()
	invalid.ConfigFormat = "json"
	invalid.Schema = "invalid"
	if got := mcpConfigSchemaCheck(invalid); got.State == mcpCheckPass {
		t.Fatalf("invalid schema passed: %+v", got)
	}
}

func TestMCPDoctorBinaryLabelsUsePresenceOnly(t *testing.T) {
	root := t.TempDir()
	present := filepath.Join(root, "codegraph")
	if err := os.WriteFile(present, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	req, err := NewMCPDoctorRequest(root, "", root, "", present)
	if err != nil {
		t.Fatal(err)
	}
	if got := mcpCodeGraphBinaryCheck(req, mcpTestStatus()); got.State != mcpCheckPass {
		t.Fatalf("explicit present binary = %q, want pass", got.State)
	}
	missingReq, err := NewMCPDoctorRequest(root, "", root, "", filepath.Join(root, "absent"))
	if err != nil {
		t.Fatal(err)
	}
	if got := mcpCodeGraphBinaryCheck(missingReq, mcpTestStatus()); got.State != mcpCheckBlocked {
		t.Fatalf("explicit missing binary = %q, want blocked", got.State)
	}
	lookup := &skillsync.MCPStatus{CodeGraphBinary: "/usr/local/bin/codegraph"}
	if got := mcpCodeGraphBinaryCheck(MCPDoctorRequest{home: root}, lookup); got.State != mcpCheckPass {
		t.Fatalf("lookup binary = %q, want pass", got.State)
	}
	met := mcpTestStatus(skillsync.MCPServerStatus{Name: "codegraph", PrerequisiteMet: true})
	if got := mcpCodeGraphBinaryCheck(MCPDoctorRequest{home: root}, met); got.State != mcpCheckPass {
		t.Fatalf("met prerequisite = %q, want pass", got.State)
	}
	none := mcpCodeGraphBinaryCheck(MCPDoctorRequest{home: root}, mcpTestStatus())
	if none.State != mcpCheckBlocked {
		t.Fatalf("no binary = %q, want blocked", none.State)
	}
	if strings.Contains(strings.ToLower(none.Detail), "installed automatically") {
		t.Fatalf("missing-binary detail promises installation: %q", none.Detail)
	}
}

func TestMCPDoctorIndexLabels(t *testing.T) {
	if got := mcpCodeGraphIndexCheck(MCPDoctorRequest{home: t.TempDir()}, mcpTestStatus()); got.State != mcpCheckNotChecked {
		t.Fatalf("no project = %q, want not-checked", got.State)
	}
	cases := map[string]string{
		"absent":                mcpCheckWarn,
		"present":               mcpCheckWarn,
		"stale":                 mcpCheckWarn,
		"freshness not checked": mcpCheckWarn,
		"not-initialized":       mcpCheckWarn,
		"not-checked":           mcpCheckNotChecked,
		"some-future-state":     mcpCheckNotChecked,
	}
	for indexState, want := range cases {
		status := mcpTestStatus(skillsync.MCPServerStatus{Name: "codegraph", IndexState: indexState})
		got := mcpCodeGraphIndexCheck(MCPDoctorRequest{home: t.TempDir(), projectPath: t.TempDir()}, status)
		if got.State != want {
			t.Fatalf("index %q = %q, want %q", indexState, got.State, want)
		}
		if got.State == mcpCheckPass {
			t.Fatalf("index %q reported pass without a freshness probe", indexState)
		}
	}
}

func TestMCPDoctorAuthLabelsNeverPass(t *testing.T) {
	cases := []skillsync.MCPServerStatus{
		{},
		{Name: "context7", State: "not-configured"},
		{Name: "context7", State: "configured"},
		{Name: "context7", State: "needs-authentication"},
		{Name: "context7", State: "needs-auth"},
		{Name: "context7", State: "auth-not-checked"},
		{Name: "context7", State: "verified-connected"},
	}
	for _, server := range cases {
		check := mcpAuthCheck(mcpTestStatus(server), "context7", "Context7")
		if check.State == mcpCheckPass {
			t.Fatalf("auth for state %q passed offline", server.State)
		}
		if check.State == mcpCheckBlocked {
			t.Fatalf("auth for state %q blocked without runtime evidence", server.State)
		}
	}
	keyless := mcpAuthCheck(mcpTestStatus(skillsync.MCPServerStatus{
		Name:     "exa",
		State:    "configured",
		AuthMode: "keyless",
	}), "exa", "Exa")
	if keyless.State == mcpCheckPass {
		t.Fatal("keyless Exa passed authentication")
	}
	if !strings.Contains(strings.ToLower(keyless.Detail), "rate-limited") {
		t.Fatalf("keyless detail does not state the rate limit: %q", keyless.Detail)
	}
}

func TestMCPDoctorRuntimeAndConnectivityAreNotChecked(t *testing.T) {
	req, err := NewMCPDoctorRequest(t.TempDir(), "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := mcpRuntimeCheck(req, mcpTestStatus()); got.State != mcpCheckNotChecked {
		t.Fatalf("runtime = %q, want not-checked", got.State)
	}
	if got := mcpConnectivityCheck(); got.State != mcpCheckNotChecked {
		t.Fatalf("connectivity = %q, want not-checked", got.State)
	}
}
