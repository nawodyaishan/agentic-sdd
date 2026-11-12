package skillsync

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// helper to assert that no .agentic-sdd or MCP state directories were created.
func assertNoMCPStateCreated(t *testing.T, home string) {
	t.Helper()
	for _, p := range []string{
		filepath.Join(home, ".agentic-sdd"),
		filepath.Join(home, ".agentic-sdd", "mcp"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("read-only operation created %s", p)
		}
	}
}

// helper to assert file bytes remain unchanged.
func assertFileBytesUnchanged(t *testing.T, path string, original []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read back %s: %v", path, err)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("file %s was modified; original %q, got %q", path, string(original), string(data))
	}
}

func TestMCPDefaultServers(t *testing.T) {
	servers := MCPDefaultServers()
	if len(servers) != 3 {
		t.Fatalf("expected 3 default servers, got %d", len(servers))
	}

	names := []string{servers[0].Name, servers[1].Name, servers[2].Name}
	expectedNames := []string{"codegraph", "context7", "exa"}
	if !reflect.DeepEqual(names, expectedNames) {
		t.Fatalf("expected server order %v, got %v", expectedNames, names)
	}

	if servers[0].Kind != MCPServerLocal {
		t.Errorf("codegraph kind = %q, want %q", servers[0].Kind, MCPServerLocal)
	}
	if servers[0].Recipe != `{"type":"local","command":["codegraph","serve","--mcp"],"disabled":false,"codemode":false}` {
		t.Errorf("unexpected codegraph recipe: %s", servers[0].Recipe)
	}

	if servers[1].Kind != MCPServerRemote {
		t.Errorf("context7 kind = %q, want %q", servers[1].Kind, MCPServerRemote)
	}
	if servers[1].Recipe != `{"type":"remote","url":"https://mcp.context7.com/mcp/oauth"}` {
		t.Errorf("unexpected context7 recipe: %s", servers[1].Recipe)
	}

	if servers[2].Kind != MCPServerRemote {
		t.Errorf("exa kind = %q, want %q", servers[2].Kind, MCPServerRemote)
	}
	if servers[2].Recipe != `{"type":"remote","url":"https://mcp.exa.ai/mcp"}` {
		t.Errorf("unexpected exa recipe: %s", servers[2].Recipe)
	}
}

func TestMCPStatus_HomeRequired(t *testing.T) {
	if _, err := BuildMCPStatus("", "", ""); err == nil {
		t.Fatal("BuildMCPStatus accepted an empty home directory")
	}
	if _, err := PreviewMCPConfig("", "", nil); err == nil {
		t.Fatal("PreviewMCPConfig accepted an empty home directory")
	}
}

func TestMCPStatus_NoConfig(t *testing.T) {
	home := t.TempDir()

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus failed: %v", err)
	}

	if status.ConfigPath != "" {
		t.Errorf("ConfigPath = %q, want empty", status.ConfigPath)
	}
	if status.ConfigFormat != "" {
		t.Errorf("ConfigFormat = %q, want empty", status.ConfigFormat)
	}
	if status.Schema != "none" {
		t.Errorf("Schema = %q, want none", status.Schema)
	}
	if status.ConfigDigest != "" {
		t.Errorf("ConfigDigest = %q, want empty", status.ConfigDigest)
	}

	if len(status.Servers) != 3 {
		t.Fatalf("Servers count = %d, want 3", len(status.Servers))
	}
	for _, srv := range status.Servers {
		if srv.State != MCPStateNotConfigured {
			t.Errorf("server %q State = %q, want not-configured", srv.Name, srv.State)
		}
		if srv.Evidence != "static" && srv.Evidence != "local-probe" {
			t.Errorf("server %q Evidence = %q", srv.Name, srv.Evidence)
		}
		if srv.AuthMode != "not-checked" {
			t.Errorf("server %q AuthMode = %q, want not-checked", srv.Name, srv.AuthMode)
		}
	}

	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_JSONV2_UnrelatedServerAndUnknownFields(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`{
  "$schema": "https://opencode.ai/schema.json",
  "theme": "dark",
  "extra_setting": 42,
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"],
        "disabled": false,
        "codemode": false
      },
      "unrelated_tool": {
        "type": "remote",
        "url": "https://example.com/tool"
      }
    }
  }
}`)
	targetPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}

	if status.ConfigPath != targetPath {
		t.Errorf("ConfigPath = %q, want %q", status.ConfigPath, targetPath)
	}
	if status.ConfigFormat != "json" {
		t.Errorf("ConfigFormat = %q, want json", status.ConfigFormat)
	}
	if status.Schema != "v2" {
		t.Errorf("Schema = %q, want v2", status.Schema)
	}
	if status.ConfigDigest == "" {
		t.Error("ConfigDigest is empty")
	}

	// Must report exactly the three default servers; unrelated_tool is preserved silently
	if len(status.Servers) != 3 {
		t.Fatalf("Servers count = %d, want 3", len(status.Servers))
	}

	serverMap := make(map[string]MCPServerStatus)
	for _, s := range status.Servers {
		serverMap[s.Name] = s
	}

	cg := serverMap["codegraph"]
	if cg.State != MCPStateConfigured && cg.State != MCPStateMissingPrerequisite {
		t.Errorf("codegraph State = %q, want configured or missing-prerequisite", cg.State)
	}
	if cg.Disabled {
		t.Error("codegraph Disabled = true, want false")
	}

	c7 := serverMap["context7"]
	if c7.State != MCPStateNotConfigured {
		t.Errorf("context7 State = %q, want not-configured", c7.State)
	}

	exa := serverMap["exa"]
	if exa.State != MCPStateNotConfigured {
		t.Errorf("exa State = %q, want not-configured", exa.State)
	}

	assertFileBytesUnchanged(t, targetPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_JSONC_CommentsAndTrailingCommas(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`// top-level comment
{
  /* block comment */
  "mcp": {
    "servers": {
      // server comment with url http://foo.bar
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp/oauth",
      },
      "exa": {
        "type": "remote",
        "url": "https://mcp.exa.ai/mcp",
      },
    },
  },
}
`)
	targetPath := filepath.Join(configDir, "opencode.jsonc")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus on jsonc: %v", err)
	}

	if status.ConfigFormat != "jsonc" {
		t.Errorf("ConfigFormat = %q, want jsonc", status.ConfigFormat)
	}
	if status.Schema != "v2" {
		t.Errorf("Schema = %q, want v2", status.Schema)
	}

	serverMap := make(map[string]MCPServerStatus)
	for _, s := range status.Servers {
		serverMap[s.Name] = s
	}

	c7 := serverMap["context7"]
	if c7.State != MCPStateConfigured {
		t.Errorf("context7 State = %q, want configured", c7.State)
	}
	if c7.AuthMode != "oauth" {
		t.Errorf("context7 AuthMode = %q, want oauth", c7.AuthMode)
	}

	exa := serverMap["exa"]
	if exa.State != MCPStateConfigured {
		t.Errorf("exa State = %q, want configured", exa.State)
	}
	if exa.AuthMode != "keyless" {
		t.Errorf("exa AuthMode = %q, want keyless", exa.AuthMode)
	}

	assertFileBytesUnchanged(t, targetPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_V1LegacySchema(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`{
  "mcp": {
    "codegraph": {
      "type": "local",
      "command": ["codegraph", "serve", "--mcp"],
      "enabled": true
    }
  }
}`)
	targetPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus on V1: %v", err)
	}

	if status.Schema != "v1" {
		t.Errorf("Schema = %q, want v1", status.Schema)
	}
	if status.LegacyReason == "" {
		t.Error("LegacyReason is empty for V1 schema")
	}

	for _, s := range status.Servers {
		if s.State != MCPStateSchemaMismatch {
			t.Errorf("server %q State = %q, want schema-mismatch", s.Name, s.State)
		}
		if s.Conflict != status.LegacyReason {
			t.Errorf("server %q Conflict = %q, want %q", s.Name, s.Conflict, status.LegacyReason)
		}
	}

	assertFileBytesUnchanged(t, targetPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_AmbiguityBothJSONAndJSONC(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	jsonPath := filepath.Join(configDir, "opencode.json")
	jsoncPath := filepath.Join(configDir, "opencode.jsonc")
	content := []byte(`{"mcp":{"servers":{}}}`)
	if err := os.WriteFile(jsonPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsoncPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}

	if status.Ambiguity == "" {
		t.Error("Ambiguity was not set when both .json and .jsonc exist")
	}
	if status.ConfigPath != "" {
		t.Errorf("ConfigPath = %q, want empty on ambiguity", status.ConfigPath)
	}

	for _, s := range status.Servers {
		if s.State != MCPStateConflict {
			t.Errorf("server %q State = %q, want conflict", s.Name, s.State)
		}
		if s.Conflict != status.Ambiguity {
			t.Errorf("server %q Conflict = %q, want %q", s.Name, s.Conflict, status.Ambiguity)
		}
	}

	assertFileBytesUnchanged(t, jsonPath, content)
	assertFileBytesUnchanged(t, jsoncPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_MalformedInvalidDuplicateOversize(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "malformed JSON syntax",
			content: []byte(`{ "mcp": { "servers": `),
		},
		{
			name:    "duplicate keys at top level",
			content: []byte(`{"mcp":{"servers":{}}, "mcp":{"servers":{}}}`),
		},
		{
			name:    "duplicate keys nested in servers",
			content: []byte(`{"mcp":{"servers":{"exa":{}, "exa":{}}}}`),
		},
		{
			name:    "invalid UTF-8",
			content: []byte{0xff, 0xfe, 0xfd, 0x00},
		},
		{
			name:    "oversize file (> 1 MiB)",
			content: append([]byte(`{"mcp":{"servers":{}}}`), bytes.Repeat([]byte(" "), 1024*1024+10)...),
		},
		{
			name:    "empty file",
			content: []byte(``),
		},
		{
			name:    "mcp is not an object",
			content: []byte(`{"mcp": "not an object"}`),
		},
		{
			name:    "mcp.servers is not an object",
			content: []byte(`{"mcp": {"servers": ["not an object"]}}`),
		},
		{
			name:    "unclosed JSONC block comment",
			content: []byte(`{"mcp": {"servers": {}}} /* unclosed comment`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, ".config", "opencode")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			targetPath := filepath.Join(configDir, "opencode.json")
			if strings.Contains(tc.name, "JSONC") {
				targetPath = filepath.Join(configDir, "opencode.jsonc")
			}
			if err := os.WriteFile(targetPath, tc.content, 0o600); err != nil {
				t.Fatal(err)
			}

			status, err := BuildMCPStatus(home, "", "")
			if err != nil {
				t.Fatalf("BuildMCPStatus returned error %v; want nil error with invalid Schema", err)
			}
			if status.Schema != "invalid" {
				t.Errorf("Schema = %q, want invalid", status.Schema)
			}
			for _, s := range status.Servers {
				if s.State != MCPStateUnknown {
					t.Errorf("server %q State = %q, want unknown", s.Name, s.State)
				}
				if s.Conflict == "" {
					t.Errorf("server %q Conflict is empty, want explanation", s.Name)
				}
			}

			assertFileBytesUnchanged(t, targetPath, tc.content)
			assertNoMCPStateCreated(t, home)
		})
	}
}

func TestMCPStatus_DisabledServer(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp/oauth",
        "disabled": true
      }
    }
  }
}`)
	targetPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}

	serverMap := make(map[string]MCPServerStatus)
	for _, s := range status.Servers {
		serverMap[s.Name] = s
	}

	c7 := serverMap["context7"]
	if c7.State != MCPStateDisabled {
		t.Errorf("context7 State = %q, want disabled", c7.State)
	}
	if !c7.Disabled {
		t.Error("context7 Disabled = false, want true")
	}

	assertFileBytesUnchanged(t, targetPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_SymlinkTarget(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	realFile := filepath.Join(home, "real_opencode.json")
	content := []byte(`{"mcp":{"servers":{}}}`)
	if err := os.WriteFile(realFile, content, 0o600); err != nil {
		t.Fatal(err)
	}

	symlinkTarget := filepath.Join(configDir, "opencode.json")
	if err := os.Symlink(realFile, symlinkTarget); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus on symlink: %v", err)
	}

	if status.Schema != "unknown" {
		t.Errorf("Schema = %q, want unknown", status.Schema)
	}
	for _, s := range status.Servers {
		if s.State != MCPStateUnsafeTarget {
			t.Errorf("server %q State = %q, want unsafe-target", s.Name, s.State)
		}
		if !strings.Contains(s.Conflict, "symlink") {
			t.Errorf("server %q Conflict = %q, want symlink mention", s.Name, s.Conflict)
		}
	}

	assertFileBytesUnchanged(t, realFile, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_CodeGraphLookupAndIndexState(t *testing.T) {
	// Control PATH to an empty directory so codegraph binary is absent
	emptyBinDir := t.TempDir()
	t.Setenv("PATH", emptyBinDir)

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`{
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"]
      }
    }
  }
}`)
	targetPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. With project index present
	projectWithIndex := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectWithIndex, ".codegraph"), 0o700); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", projectWithIndex)
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}

	serverMap := make(map[string]MCPServerStatus)
	for _, s := range status.Servers {
		serverMap[s.Name] = s
	}

	cg := serverMap["codegraph"]
	if cg.State != MCPStateMissingPrerequisite {
		t.Errorf("codegraph State = %q, want missing-prerequisite", cg.State)
	}
	if cg.PrerequisiteMet {
		t.Error("codegraph PrerequisiteMet = true, want false")
	}
	if cg.Evidence != "local-probe" {
		t.Errorf("codegraph Evidence = %q, want local-probe", cg.Evidence)
	}
	if cg.IndexState != "present" {
		t.Errorf("codegraph IndexState = %q, want present", cg.IndexState)
	}

	// 2. With project index absent
	projectNoIndex := t.TempDir()
	statusNoIndex, err := BuildMCPStatus(home, "", projectNoIndex)
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}
	cgNoIndex := statusNoIndex.Servers[0]
	if cgNoIndex.IndexState != "absent" {
		t.Errorf("codegraph IndexState = %q, want absent", cgNoIndex.IndexState)
	}

	// 3. Without project path
	statusNoProject, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}
	cgNoProject := statusNoProject.Servers[0]
	if cgNoProject.IndexState != "not-checked" {
		t.Errorf("codegraph IndexState = %q, want not-checked", cgNoProject.IndexState)
	}

	assertFileBytesUnchanged(t, targetPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPStatus_AuthModes(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	content := []byte(`{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp",
        "headers": {
          "Authorization": "Bearer {env:CONTEXT7_KEY}"
        }
      },
      "exa": {
        "type": "remote",
        "url": "https://mcp.exa.ai/mcp?login"
      }
    }
  }
}`)
	targetPath := filepath.Join(configDir, "opencode.json")
	if err := os.WriteFile(targetPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, "", "")
	if err != nil {
		t.Fatalf("BuildMCPStatus: %v", err)
	}

	serverMap := make(map[string]MCPServerStatus)
	for _, s := range status.Servers {
		serverMap[s.Name] = s
	}

	if c7Auth := serverMap["context7"].AuthMode; c7Auth != "api-key" {
		t.Errorf("context7 AuthMode = %q, want api-key", c7Auth)
	}
	if exaAuth := serverMap["exa"].AuthMode; exaAuth != "oauth" {
		t.Errorf("exa AuthMode = %q, want oauth", exaAuth)
	}
}

func TestMCPStatus_ExplicitConfigPathOutsideDefaultScope(t *testing.T) {
	home := t.TempDir()
	customDir := t.TempDir()
	customPath := filepath.Join(customDir, "custom_opencode.json")
	content := []byte(`{
  "mcp": {
    "servers": {
      "exa": {
        "type": "remote",
        "url": "https://mcp.exa.ai/mcp"
      }
    }
  }
}`)
	if err := os.WriteFile(customPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	status, err := BuildMCPStatus(home, customPath, "")
	if err != nil {
		t.Fatalf("BuildMCPStatus with explicit config: %v", err)
	}

	if status.ConfigPath != customPath {
		t.Errorf("ConfigPath = %q, want %q", status.ConfigPath, customPath)
	}
	if status.ConfigFormat != "json" {
		t.Errorf("ConfigFormat = %q, want json", status.ConfigFormat)
	}

	preview, err := PreviewMCPConfig(home, customPath, []string{"exa"})
	if err != nil {
		t.Fatalf("PreviewMCPConfig with explicit config: %v", err)
	}
	if preview.TargetPath != customPath {
		t.Errorf("preview TargetPath = %q, want %q", preview.TargetPath, customPath)
	}
	if preview.Format != "json" {
		t.Errorf("preview Format = %q, want json", preview.Format)
	}

	assertFileBytesUnchanged(t, customPath, content)
	assertNoMCPStateCreated(t, home)
}

func TestMCPPreview_Semantics(t *testing.T) {
	t.Run("no config on disk defaults to add all 3", func(t *testing.T) {
		home := t.TempDir()

		preview, err := PreviewMCPConfig(home, "", nil)
		if err != nil {
			t.Fatalf("PreviewMCPConfig: %v", err)
		}

		wantTarget := filepath.Join(home, ".config", "opencode", "opencode.json")
		if preview.TargetPath != wantTarget {
			t.Errorf("TargetPath = %q, want %q", preview.TargetPath, wantTarget)
		}
		if preview.Format != "json" {
			t.Errorf("Format = %q, want json", preview.Format)
		}
		if preview.BaseDigest != "" {
			t.Errorf("BaseDigest = %q, want empty", preview.BaseDigest)
		}
		if preview.NoOp {
			t.Error("NoOp = true, want false")
		}

		if len(preview.Entries) != 3 {
			t.Fatalf("Entries count = %d, want 3", len(preview.Entries))
		}
		for _, e := range preview.Entries {
			if e.Op != "add" {
				t.Errorf("entry %q Op = %q, want add", e.Name, e.Op)
			}
			if e.Recipe == "" {
				t.Errorf("entry %q Recipe is empty for add", e.Name)
			}
		}

		assertNoMCPStateCreated(t, home)
	})

	t.Run("all configured results in leave and no-op", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatal(err)
		}

		content := []byte(`{
  "mcp": {
    "servers": {
      "codegraph": { "type": "local", "command": ["codegraph", "serve", "--mcp"] },
      "context7": { "type": "remote", "url": "https://mcp.context7.com/mcp/oauth" },
      "exa": { "type": "remote", "url": "https://mcp.exa.ai/mcp" }
    }
  }
}`)
		targetPath := filepath.Join(configDir, "opencode.json")
		if err := os.WriteFile(targetPath, content, 0o600); err != nil {
			t.Fatal(err)
		}

		preview, err := PreviewMCPConfig(home, "", nil)
		if err != nil {
			t.Fatalf("PreviewMCPConfig: %v", err)
		}

		if !preview.NoOp {
			t.Error("NoOp = false, want true when all entries are leave")
		}
		if len(preview.Conflicts) != 0 {
			t.Errorf("Conflicts = %v, want empty", preview.Conflicts)
		}

		for _, e := range preview.Entries {
			if e.Op != "leave" {
				t.Errorf("entry %q Op = %q, want leave", e.Name, e.Op)
			}
			if e.Reason != "already configured; exact merge preview is MCP-B" {
				t.Errorf("entry %q Reason = %q", e.Name, e.Reason)
			}
		}

		assertFileBytesUnchanged(t, targetPath, content)
		assertNoMCPStateCreated(t, home)
	})

	t.Run("disabled server preserved as disabled and no-op", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatal(err)
		}

		content := []byte(`{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp/oauth",
        "disabled": true
      }
    }
  }
}`)
		targetPath := filepath.Join(configDir, "opencode.json")
		if err := os.WriteFile(targetPath, content, 0o600); err != nil {
			t.Fatal(err)
		}

		// Select only context7
		preview, err := PreviewMCPConfig(home, "", []string{"context7"})
		if err != nil {
			t.Fatalf("PreviewMCPConfig: %v", err)
		}

		if !preview.NoOp {
			t.Error("NoOp = false, want true when selected server is disabled")
		}
		if len(preview.Entries) != 1 {
			t.Fatalf("Entries count = %d, want 1", len(preview.Entries))
		}
		if preview.Entries[0].Op != "disabled" {
			t.Errorf("entry Op = %q, want disabled", preview.Entries[0].Op)
		}

		assertFileBytesUnchanged(t, targetPath, content)
		assertNoMCPStateCreated(t, home)
	})

	t.Run("unknown server name returns error", func(t *testing.T) {
		home := t.TempDir()
		_, err := PreviewMCPConfig(home, "", []string{"unknown_server"})
		if err == nil {
			t.Fatal("PreviewMCPConfig accepted an unknown server name")
		}
		if !strings.Contains(err.Error(), "unknown server name") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("conflict target reports conflict and NoOp false", func(t *testing.T) {
		home := t.TempDir()
		configDir := filepath.Join(home, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			t.Fatal(err)
		}

		content := []byte(`{
  "mcp": {
    "codegraph": { "enabled": true }
  }
}`)
		targetPath := filepath.Join(configDir, "opencode.json")
		if err := os.WriteFile(targetPath, content, 0o600); err != nil {
			t.Fatal(err)
		}

		preview, err := PreviewMCPConfig(home, "", nil)
		if err != nil {
			t.Fatalf("PreviewMCPConfig on V1: %v", err)
		}

		if preview.NoOp {
			t.Error("NoOp = true, want false when conflict exists")
		}
		if len(preview.Conflicts) == 0 {
			t.Error("Conflicts is empty, want V1 schema conflict")
		}
		for _, e := range preview.Entries {
			if e.Op != "conflict" {
				t.Errorf("entry %q Op = %q, want conflict", e.Name, e.Op)
			}
		}

		assertFileBytesUnchanged(t, targetPath, content)
		assertNoMCPStateCreated(t, home)
	})
}
