package skillsync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Fixtures and helpers shared by the MCP-B config and transaction tests. Every
// test uses an isolated temporary home; no test reads or writes a real user
// configuration, contacts the network or invokes a real opencode binary.

const mcpTestBaseConfig = `{
  "$schema": "https://opencode.ai/schema.json",
  "theme": "dark",
  "extra_setting": 42,
  "mcp": {
    "servers": {
      "unrelated": {
        "type": "remote",
        "url": "https://example.test/mcp",
        "custom_field": "kept"
      }
    }
  }
}
`

func mcpTestConfigDir(home string) string {
	return filepath.Join(home, ".config", "opencode")
}

func mcpTestConfigPath(home string) string {
	return filepath.Join(mcpTestConfigDir(home), "opencode.json")
}

// mcpWriteTestConfig writes one configuration fixture at 0644 and returns its
// path.
func mcpWriteTestConfig(t *testing.T, home, name, content string) string {
	t.Helper()
	dir := mcpTestConfigDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mcpReadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// mcpDecodeJSON parses a document preserving number literals.
func mcpDecodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root map[string]any
	if err := dec.Decode(&root); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, data)
	}
	return root
}

func mcpTestServers(t *testing.T, data []byte) map[string]any {
	t.Helper()
	root := mcpDecodeJSON(t, data)
	mcpMap, ok := root["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("document has no mcp object: %s", data)
	}
	servers, ok := mcpMap["servers"].(map[string]any)
	if !ok {
		t.Fatalf("document has no mcp.servers object: %s", data)
	}
	return servers
}

func mcpTestEntry(t *testing.T, data []byte, name string) map[string]any {
	t.Helper()
	servers := mcpTestServers(t, data)
	entry, ok := servers[name].(map[string]any)
	if !ok {
		t.Fatalf("server %q is missing from %s", name, data)
	}
	return entry
}

// mcpFakeOpenCode writes a fake `opencode --version` executable in the test's
// temporary directory. Tests never execute a real opencode binary.
func mcpFakeOpenCode(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode")
	script := "#!/bin/sh\necho '" + output + "'\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// mcpWriteTestManifest writes an ownership manifest fixture at 0600.
func mcpWriteTestManifest(t *testing.T, home string, manifest mcpOwnershipManifest) {
	t.Helper()
	dir := filepath.Join(home, ".agentic-sdd", "mcp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mcpTestManifest(target string, entries map[string]string) mcpOwnershipManifest {
	manifest := mcpOwnershipManifest{
		Format:          mcpManifestFormat,
		ManifestVersion: mcpManifestVersion,
		Target:          target,
		TargetFormat:    mcpFormatJSON,
		Schema:          mcpSchemaV2,
	}
	for name, digest := range entries {
		manifest.Entries = append(manifest.Entries, mcpManifestEntry{Name: name, InstalledDigest: digest})
	}
	return mcpSortedTestManifest(manifest)
}

func mcpSortedTestManifest(manifest mcpOwnershipManifest) mcpOwnershipManifest {
	manifest.Entries = mcpSortManifestEntries(manifest.Entries)
	return manifest
}

func mcpTestEntryHash(t *testing.T, entry map[string]any) string {
	t.Helper()
	_, hash, err := mcpCanonicalEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func mcpTestBuildAll(t *testing.T, home string) *MCPPlan {
	t.Helper()
	plan, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	return plan
}

func mcpTestOps(plan *MCPPlan) map[string]string {
	ops := make(map[string]string, len(plan.Entries))
	for _, entry := range plan.Entries {
		ops[entry.Name] = entry.Op
	}
	return ops
}

func TestMCPPlanHomeRequired(t *testing.T) {
	if _, err := BuildMCPPlan("", "", nil, nil); !errors.Is(err, ErrMCPUsage) {
		t.Fatalf("empty home error = %v, want ErrMCPUsage", err)
	}
}

func TestMCPPlanAddPreservesUnrelatedAndIsDeterministic(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	original := mcpReadTestFile(t, path)

	plan, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.TargetPath != path {
		t.Fatalf("TargetPath = %q, want %q", plan.TargetPath, path)
	}
	if plan.Format != mcpFormatJSON || plan.Schema != mcpSchemaV2 {
		t.Fatalf("format/schema = %q/%q, want json/v2", plan.Format, plan.Schema)
	}
	if plan.BaseDigest != mcpBytesDigest(original) {
		t.Fatalf("BaseDigest does not describe the current bytes")
	}
	if plan.PlanDigest == "" || plan.PlanDigest == plan.BaseDigest {
		t.Fatalf("PlanDigest = %q, want a distinct non-empty digest", plan.PlanDigest)
	}
	if plan.NoOp {
		t.Fatal("add plan reported a no-op")
	}
	if plan.RuntimeVersion != "not-checked" || plan.RuntimeEvidence != "not-checked" {
		t.Fatalf("read-only plan claimed runtime evidence: %q/%q", plan.RuntimeVersion, plan.RuntimeEvidence)
	}
	if len(plan.NotChecked) == 0 {
		t.Fatal("plan did not record what it did not check")
	}
	wantOps := map[string]string{"codegraph": mcpOpAdd, "context7": mcpOpAdd, "exa": mcpOpAdd}
	if got := mcpTestOps(plan); !reflect.DeepEqual(got, wantOps) {
		t.Fatalf("ops = %v, want %v", got, wantOps)
	}
	if got := plan.AuthChoices; got["context7"] != mcpAuthOAuth || got["exa"] != mcpAuthKeyless {
		t.Fatalf("AuthChoices = %v, want documented defaults", got)
	}

	bytesOut := plan.TargetBytes()
	if len(bytesOut) == 0 {
		t.Fatal("writable plan carried no target bytes")
	}
	servers := mcpTestServers(t, bytesOut)
	if len(servers) != 4 {
		t.Fatalf("servers = %v, want the unrelated server plus three managed ones", servers)
	}
	unrelated, ok := servers["unrelated"].(map[string]any)
	if !ok || unrelated["url"] != "https://example.test/mcp" || unrelated["custom_field"] != "kept" {
		t.Fatalf("unrelated server was not preserved: %v", servers["unrelated"])
	}
	codegraph := mcpTestEntry(t, bytesOut, "codegraph")
	if codegraph["type"] != "local" {
		t.Fatalf("codegraph type = %v", codegraph["type"])
	}
	command, _ := codegraph["command"].([]any)
	if len(command) != 3 || command[0] != "codegraph" || command[1] != "serve" || command[2] != "--mcp" {
		t.Fatalf("codegraph command = %v", codegraph["command"])
	}
	if context7 := mcpTestEntry(t, bytesOut, "context7"); context7["url"] != "https://mcp.context7.com/mcp/oauth" {
		t.Fatalf("context7 url = %v", context7["url"])
	}
	if exa := mcpTestEntry(t, bytesOut, "exa"); exa["url"] != "https://mcp.exa.ai/mcp" {
		t.Fatalf("exa url = %v", exa["url"])
	}

	root := mcpDecodeJSON(t, bytesOut)
	if root["theme"] != "dark" || fmt.Sprint(root["extra_setting"]) != "42" {
		t.Fatalf("unrelated top-level fields were not preserved: %v", root)
	}

	assertFileBytesUnchanged(t, path, original)
	assertNoMCPStateCreated(t, home)

	again := mcpTestBuildAll(t, home)
	if again.PlanDigest != plan.PlanDigest {
		t.Fatalf("plan digest is not deterministic: %q vs %q", plan.PlanDigest, again.PlanDigest)
	}
	if !bytes.Equal(again.TargetBytes(), bytesOut) {
		t.Fatal("planned target bytes are not deterministic")
	}
}

func TestMCPPlanAddCreatesMissingContainers(t *testing.T) {
	for name, content := range map[string]string{
		"no mcp key":       "{\n  \"theme\": \"dark\"\n}\n",
		"empty mcp object": "{\n  \"mcp\": {}\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			path := mcpWriteTestConfig(t, home, "opencode.json", content)
			plan := mcpTestBuildAll(t, home)
			bytesOut := plan.TargetBytes()
			if len(bytesOut) == 0 {
				t.Fatalf("plan for %q carried no bytes", name)
			}
			servers := mcpTestServers(t, bytesOut)
			if len(servers) != 3 {
				t.Fatalf("servers = %v, want three adds", servers)
			}
			assertFileBytesUnchanged(t, path, []byte(content))
		})
	}
}

func TestMCPPlanChangeManagedEntry(t *testing.T) {
	home := t.TempDir()
	oldEntry := map[string]any{
		"type":     "local",
		"command":  []any{"codegraph", "serve"},
		"disabled": false,
		"codemode": false,
	}
	oldText, oldHash, err := mcpCanonicalEntry(oldEntry)
	if err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"mcp\": {\n    \"servers\": {\n      \"codegraph\": " + oldText + "\n    }\n  }\n}\n"
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	mcpWriteTestManifest(t, home, mcpTestManifest(path, map[string]string{"codegraph": oldHash}))

	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if len(plan.Entries) != 1 {
		t.Fatalf("entries = %v, want one", plan.Entries)
	}
	entry := plan.Entries[0]
	if entry.Op != mcpOpChange {
		t.Fatalf("op = %q, want change (%s)", entry.Op, entry.Reason)
	}
	if entry.OldText != oldText || entry.OldHash != oldHash {
		t.Fatalf("old entry evidence = %q/%q", entry.OldText, entry.OldHash)
	}
	if entry.NewHash == "" || entry.NewHash == entry.OldHash {
		t.Fatal("change did not record a new entry digest")
	}
	codegraph := mcpTestEntry(t, plan.TargetBytes(), "codegraph")
	command, _ := codegraph["command"].([]any)
	if len(command) != 3 {
		t.Fatalf("managed change did not update the command: %v", codegraph)
	}
}

func TestMCPPlanLeaveAndNoOp(t *testing.T) {
	home := t.TempDir()
	codegraph, err := mcpDesiredEntry("codegraph", "")
	if err != nil {
		t.Fatal(err)
	}
	context7, err := mcpDesiredEntry("context7", mcpAuthOAuth)
	if err != nil {
		t.Fatal(err)
	}
	exa, err := mcpDesiredEntry("exa", mcpAuthKeyless)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{"mcp": map[string]any{"servers": map[string]any{
		"codegraph": codegraph,
		"context7":  context7,
		"exa":       exa,
	}}}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := mcpWriteTestConfig(t, home, "opencode.json", string(data)+"\n")

	plan := mcpTestBuildAll(t, home)
	if !plan.NoOp {
		t.Fatalf("identical configuration planned a write: %v", plan.Entries)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("identical configuration reported conflicts: %v", plan.Conflicts)
	}
	if plan.TargetBytes() != nil {
		t.Fatal("no-op plan carried target bytes")
	}
	for _, entry := range plan.Entries {
		if entry.Op != mcpOpLeave {
			t.Fatalf("entry %q op = %q, want leave", entry.Name, entry.Op)
		}
	}
	assertFileBytesUnchanged(t, path, []byte(string(data)+"\n"))

	// An entry with the same managed fields but an omitted disabled/codemode
	// and an unrelated extra field is still a no-op, never a rewrite.
	minimal := map[string]any{"mcp": map[string]any{"servers": map[string]any{
		"codegraph": map[string]any{
			"type":    "local",
			"command": []any{"codegraph", "serve", "--mcp"},
			"timeout": json.Number("30"),
		},
	}}}
	data, err = json.MarshalIndent(minimal, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	home2 := t.TempDir()
	mcpWriteTestConfig(t, home2, "opencode.json", string(data)+"\n")
	plan, err = BuildMCPPlan(home2, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if !plan.NoOp || plan.Entries[0].Op != mcpOpLeave {
		t.Fatalf("minimal equivalent entry planned a write: %v", plan.Entries[0])
	}
}

func TestMCPPlanDisabledNeverReenabled(t *testing.T) {
	home := t.TempDir()
	content := `{
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve"],
        "disabled": true
      }
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.Entries[0].Op != mcpOpDisabled {
		t.Fatalf("op = %q, want disabled", plan.Entries[0].Op)
	}
	if !plan.NoOp {
		t.Fatal("a disabled server should not plan a write")
	}
	assertFileBytesUnchanged(t, path, []byte(content))

	// A disabled server in a document that also needs other additions keeps
	// its disabled state and its exact entry.
	full := mcpTestBuildAll(t, home)
	codegraph := mcpTestEntry(t, full.TargetBytes(), "codegraph")
	disabled, ok := codegraph["disabled"].(bool)
	if !ok || !disabled {
		t.Fatalf("disabled state was lost: %v", codegraph)
	}
}

func TestMCPPlanConflictRefusal(t *testing.T) {
	home := t.TempDir()
	content := `{
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["someone-elses-binary", "serve"]
      }
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.Entries[0].Op != mcpOpConflict {
		t.Fatalf("op = %q, want conflict", plan.Entries[0].Op)
	}
	if len(plan.Conflicts) == 0 {
		t.Fatal("conflict plan recorded no conflict reason")
	}
	if plan.TargetBytes() != nil {
		t.Fatal("conflict plan carried target bytes")
	}

	_, err = ApplyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, "2.0.0"),
	})
	if !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("apply error = %v, want ErrMCPConflict", err)
	}
	assertFileBytesUnchanged(t, path, []byte(content))
	assertNoMCPStateCreated(t, home)
}

func TestMCPPlanUserModifiedManagedEntryConflicts(t *testing.T) {
	home := t.TempDir()
	entry := map[string]any{
		"type":    "local",
		"command": []any{"codegraph", "serve", "--mcp"},
	}
	text, hash, err := mcpCanonicalEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"mcp\": {\n    \"servers\": {\n      \"codegraph\": " + text + "\n    }\n  }\n}\n"
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	// The recorded installed digest is different, so the entry is user-edited.
	mcpWriteTestManifest(t, home, mcpTestManifest(path, map[string]string{"codegraph": mcpBytesDigest([]byte("other"))}))
	if hash == "" {
		t.Fatal("empty entry hash")
	}

	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.Entries[0].Op != mcpOpConflict {
		t.Fatalf("op = %q, want conflict", plan.Entries[0].Op)
	}
	if !strings.Contains(plan.Entries[0].Reason, "modified") {
		t.Fatalf("reason = %q, want a user-modified explanation", plan.Entries[0].Reason)
	}
}

func TestMCPPlanMissingTargetNeverCreates(t *testing.T) {
	home := t.TempDir()
	plan, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.TargetBytes() != nil {
		t.Fatal("missing-target plan carried bytes")
	}
	if plan.BaseDigest != "" {
		t.Fatalf("BaseDigest = %q, want empty", plan.BaseDigest)
	}
	if len(plan.Conflicts) == 0 || !strings.Contains(strings.Join(plan.Conflicts, " "), "no creation in v1") {
		t.Fatalf("conflicts = %v, want a no-creation refusal", plan.Conflicts)
	}
	defaultPath := mcpTestConfigPath(home)
	if _, err := ApplyMCPPlan(home, "", plan, MCPApplyOptions{
		Home: home, ExpectDigest: plan.BaseDigest, OpenCodeBin: mcpFakeOpenCode(t, "2.0.0"),
	}); err == nil {
		t.Fatal("apply accepted a missing target")
	}
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("apply created %s", defaultPath)
	}
	assertNoMCPStateCreated(t, home)
	if snippet := MCPManualSnippetForPlan(plan); !strings.Contains(snippet, "\"codegraph\"") {
		t.Fatalf("manual snippet = %q, want the managed servers", snippet)
	}
}

func TestMCPPlanV1SchemaRefused(t *testing.T) {
	home := t.TempDir()
	content := `{
  "mcp": {
    "codegraph": {
      "type": "local",
      "command": ["codegraph", "serve", "--mcp"],
      "enabled": true
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPUnsupported) {
		t.Fatalf("error = %v, want ErrMCPUnsupported for the V1 schema", err)
	}
	assertFileBytesUnchanged(t, path, []byte(content))
	assertNoMCPStateCreated(t, home)
}

func TestMCPPlanInvalidDocumentsRefused(t *testing.T) {
	cases := map[string]string{
		"malformed":           `{"mcp": {`,
		"duplicate key":       `{"mcp": {}, "mcp": {"servers": {}}}`,
		"top-level array":     `[]`,
		"mcp not an object":   `{"mcp": []}`,
		"servers not object":  `{"mcp": {"servers": []}}`,
		"entry not an object": `{"mcp": {"servers": {"codegraph": "codegraph"}}}`,
		"command not array":   `{"mcp": {"servers": {"codegraph": {"type": "local", "command": "codegraph"}}}}`,
		"unknown type":        `{"mcp": {"servers": {"codegraph": {"type": "stdio", "command": ["x"]}}}}`,
		"empty local command": `{"mcp": {"servers": {"codegraph": {"type": "local", "command": []}}}}`,
		"empty remote url":    `{"mcp": {"servers": {"context7": {"type": "remote", "url": ""}}}}`,
		"disabled not bool":   `{"mcp": {"servers": {"exa": {"type": "remote", "url": "https://x.test", "disabled": "yes"}}}}`,
		"headers not object":  `{"mcp": {"servers": {"exa": {"type": "remote", "url": "https://x.test", "headers": []}}}}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			path := mcpWriteTestConfig(t, home, "opencode.json", content)
			if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPConflict) {
				t.Fatalf("error = %v, want ErrMCPConflict", err)
			}
			assertFileBytesUnchanged(t, path, []byte(content))
			assertNoMCPStateCreated(t, home)
		})
	}

	t.Run("invalid UTF-8", func(t *testing.T) {
		home := t.TempDir()
		path := mcpWriteTestConfig(t, home, "opencode.json", "")
		if err := os.WriteFile(path, []byte{0xff, 0xfe, '{', '}'}, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPConflict) {
			t.Fatalf("error = %v, want ErrMCPConflict", err)
		}
	})
}

func TestMCPPlanAmbiguityBothTargets(t *testing.T) {
	home := t.TempDir()
	mcpWriteTestConfig(t, home, "opencode.json", "{\"mcp\":{}}\n")
	mcpWriteTestConfig(t, home, "opencode.jsonc", "{\"mcp\":{}}\n")
	if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("error = %v, want ErrMCPConflict for both targets", err)
	}
}

func TestMCPPlanUnsafeTargets(t *testing.T) {
	t.Run("symlinked target", func(t *testing.T) {
		home := t.TempDir()
		realPath := filepath.Join(home, "elsewhere.json")
		if err := os.WriteFile(realPath, []byte("{\"mcp\":{}}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		dir := mcpTestConfigDir(home)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "opencode.json")
		if err := os.Symlink(realPath, link); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("error = %v, want ErrMCPUnsafeTarget", err)
		}
		if _, err := os.Stat(realPath); err != nil {
			t.Fatalf("refused target disappeared: %v", err)
		}
	})

	t.Run("symlinked parent", func(t *testing.T) {
		home := t.TempDir()
		realDir := filepath.Join(home, "real-opencode")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(realDir, "opencode.json")
		if err := os.WriteFile(path, []byte("{\"mcp\":{}}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		configDir := mcpTestConfigDir(home)
		if err := os.MkdirAll(filepath.Dir(configDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realDir, configDir); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("error = %v, want ErrMCPUnsafeTarget", err)
		}
	})

	t.Run("traversal component", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", "{\"mcp\":{}}\n")
		raw := mcpTestConfigDir(home) + string(filepath.Separator) + "sub" +
			string(filepath.Separator) + ".." + string(filepath.Separator) + "opencode.json"
		if _, err := BuildMCPPlan(home, raw, nil, nil); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("error = %v, want ErrMCPUnsafeTarget", err)
		}
	})

	t.Run("directory target", func(t *testing.T) {
		home := t.TempDir()
		dir := mcpTestConfigDir(home)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "opencode.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("error = %v, want ErrMCPUnsafeTarget", err)
		}
	})

	t.Run("explicit path outside the global scope", func(t *testing.T) {
		home := t.TempDir()
		outside := filepath.Join(home, "project", "opencode.json")
		if err := os.MkdirAll(filepath.Dir(outside), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(outside, []byte("{\"mcp\":{}}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := BuildMCPPlan(home, outside, nil, nil); !errors.Is(err, ErrMCPUnsafeTarget) {
			t.Fatalf("error = %v, want ErrMCPUnsafeTarget", err)
		}
		assertFileBytesUnchanged(t, outside, []byte("{\"mcp\":{}}\n"))
	})
}

func TestMCPPlanJSONCRefusalAndManualSnippet(t *testing.T) {
	home := t.TempDir()
	content := `{
  // keep this comment exactly
  "theme": "dark",
  "mcp": {
    "servers": {
      /* an unrelated server */
      "unrelated": { "type": "remote", "url": "https://example.test/mcp" }
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.jsonc", content)
	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if !errors.Is(err, ErrMCPUnsupported) {
		t.Fatalf("error = %v, want ErrMCPUnsupported", err)
	}
	if plan == nil {
		t.Fatal("JSONC refusal returned no plan evidence")
	}
	if plan.Format != mcpFormatJSONC || plan.TargetPath != path {
		t.Fatalf("plan target/format = %q/%q", plan.TargetPath, plan.Format)
	}
	if plan.TargetBytes() != nil {
		t.Fatal("JSONC refusal carried target bytes")
	}
	message := err.Error()
	if !strings.Contains(message, "\"codegraph\"") || !strings.Contains(message, "\"servers\"") {
		t.Fatalf("refusal = %q, want an exact manual snippet", message)
	}
	assertFileBytesUnchanged(t, path, []byte(content))
	assertNoMCPStateCreated(t, home)
}

func TestMCPPlanJSONCNoOpNeedsNoWrite(t *testing.T) {
	home := t.TempDir()
	content := `{
  // managed servers already present
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"],
        "disabled": false,
        "codemode": false
      },
      "context7": { "type": "remote", "url": "https://mcp.context7.com/mcp/oauth" },
      "exa": { "type": "remote", "url": "https://mcp.exa.ai/mcp" }
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.jsonc", content)
	plan, err := BuildMCPPlan(home, "", nil, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if !plan.NoOp {
		t.Fatalf("JSONC plan that needs no write reported %v", plan.Entries)
	}
	if plan.TargetBytes() != nil {
		t.Fatal("JSONC no-op carried bytes")
	}
	assertFileBytesUnchanged(t, path, []byte(content))
}

func TestMCPPlanAuthRecipes(t *testing.T) {
	t.Run("context7 api-key and exa api-key", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		plan, err := BuildMCPPlan(home, "", []string{"context7", "exa"}, map[string]string{
			"context7": "api-key",
			"exa":      "api-key",
		})
		if err != nil {
			t.Fatalf("BuildMCPPlan: %v", err)
		}
		bytesOut := plan.TargetBytes()
		context7 := mcpTestEntry(t, bytesOut, "context7")
		if context7["url"] != "https://mcp.context7.com/mcp" {
			t.Fatalf("context7 url = %v", context7["url"])
		}
		if oauth, ok := context7["oauth"].(bool); !ok || oauth {
			t.Fatalf("context7 oauth = %v, want false", context7["oauth"])
		}
		headers, _ := context7["headers"].(map[string]any)
		if headers["Authorization"] != "Bearer {env:CONTEXT7_API_KEY}" {
			t.Fatalf("context7 headers = %v, want an env placeholder", headers)
		}
		exaHeaders, _ := mcpTestEntry(t, bytesOut, "exa")["headers"].(map[string]any)
		if exaHeaders["x-api-key"] != "{env:EXA_API_KEY}" {
			t.Fatalf("exa headers = %v, want an env placeholder", exaHeaders)
		}
		if plan.AuthChoices["context7"] != "api-key" || plan.AuthChoices["exa"] != "api-key" {
			t.Fatalf("AuthChoices = %v", plan.AuthChoices)
		}
		// No literal credential may appear in the planned bytes.
		if bytes.Contains(bytesOut, []byte("Bearer sk")) {
			t.Fatal("planned bytes contain a literal credential")
		}
	})

	t.Run("exa oauth", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		plan, err := BuildMCPPlan(home, "", []string{"exa"}, map[string]string{"exa": "oauth"})
		if err != nil {
			t.Fatalf("BuildMCPPlan: %v", err)
		}
		if exa := mcpTestEntry(t, plan.TargetBytes(), "exa"); exa["url"] != "https://mcp.exa.ai/mcp?login" {
			t.Fatalf("exa url = %v", exa["url"])
		}
	})

	t.Run("unknown auth choices are usage errors", func(t *testing.T) {
		home := t.TempDir()
		mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
		for name, auth := range map[string]map[string]string{
			"context7 token": {"context7": "token"},
			"exa bearer":     {"exa": "bearer"},
			"codegraph auth": {"codegraph": "oauth"},
			"unknown target": {"other": "oauth"},
		} {
			if _, err := BuildMCPPlan(home, "", nil, auth); !errors.Is(err, ErrMCPUsage) {
				t.Fatalf("%s: error = %v, want ErrMCPUsage", name, err)
			}
		}
	})
}

func TestMCPPlanSelectionValidation(t *testing.T) {
	home := t.TempDir()
	mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)

	if _, err := BuildMCPPlan(home, "", []string{"bogus"}, nil); !errors.Is(err, ErrMCPUsage) {
		t.Fatalf("unknown server error = %v, want ErrMCPUsage", err)
	}
	plan, err := BuildMCPPlan(home, "", []string{"exa", "context7"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if !reflect.DeepEqual(plan.Selected, []string{"context7", "exa"}) {
		t.Fatalf("Selected = %v, want canonical order", plan.Selected)
	}
	if len(plan.Entries) != 2 {
		t.Fatalf("entries = %v, want only the selected servers", plan.Entries)
	}
	if _, ok := mcpTestServers(t, plan.TargetBytes())["codegraph"]; ok {
		t.Fatal("an unselected server was written")
	}
}

func TestMCPPlanLiteralCredentialRefusedWithoutEcho(t *testing.T) {
	home := t.TempDir()
	content := `{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp",
        "headers": { "Authorization": "Bearer sk-live-abcdef123456" }
      }
    }
  }
}
`
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	plan, err := BuildMCPPlan(home, "", []string{"context7"}, nil)
	if !errors.Is(err, ErrMCPConflict) {
		t.Fatalf("error = %v, want ErrMCPConflict", err)
	}
	if strings.Contains(err.Error(), "sk-live-abcdef123456") {
		t.Fatalf("error echoed the literal credential: %q", err.Error())
	}
	if plan != nil {
		t.Fatalf("plan was rendered despite a literal credential: %+v", plan)
	}
	assertFileBytesUnchanged(t, path, []byte(content))
}

func TestMCPPlanIgnoresOwnershipForAnotherTarget(t *testing.T) {
	home := t.TempDir()
	entry := map[string]any{
		"type":    "local",
		"command": []any{"codegraph", "serve"},
	}
	text, hash, err := mcpCanonicalEntry(entry)
	if err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"mcp\": {\n    \"servers\": {\n      \"codegraph\": " + text + "\n    }\n  }\n}\n"
	path := mcpWriteTestConfig(t, home, "opencode.json", content)
	mcpWriteTestManifest(t, home, mcpTestManifest(path+".other", map[string]string{"codegraph": hash}))

	plan, err := BuildMCPPlan(home, "", []string{"codegraph"}, nil)
	if err != nil {
		t.Fatalf("BuildMCPPlan: %v", err)
	}
	if plan.Entries[0].Op != mcpOpConflict {
		t.Fatalf("op = %q, want conflict for a foreign ownership record", plan.Entries[0].Op)
	}
}

func TestMCPPlanManifestFormatRefused(t *testing.T) {
	home := t.TempDir()
	path := mcpWriteTestConfig(t, home, "opencode.json", mcpTestBaseConfig)
	dir := filepath.Join(home, ".agentic-sdd", "mcp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{\"format\": 99, \"target\": "+fmt.Sprintf("%q", path)+"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildMCPPlan(home, "", nil, nil); !errors.Is(err, ErrMCPUnsupported) {
		t.Fatalf("error = %v, want ErrMCPUnsupported", err)
	}
}
