package fleet

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const renderCanonical = `---
description: Example implementation role.
mode: subagent
model: opencode/deepseek-v4.1-flash-free
permissions:
  - action: "*"
    resource: "*"
    effect: deny
  - action: read
    resource: "*"
    effect: allow
  - action: glob
    resource: "*"
    effect: allow
  - action: grep
    resource: "*"
    effect: allow
  - action: skill
    resource: "*"
    effect: allow
  - action: execute
    resource: "*"
    effect: allow
  - action: codegraph_*
    resource: "*"
    effect: allow
  - action: context7_*
    resource: "*"
    effect: allow
  - action: exa_*
    resource: "*"
    effect: deny
  - action: external_directory
    resource: "*"
    effect: allow
  - action: read
    resource: "*.env"
    effect: deny
  - action: subagent
    resource: "*"
    effect: deny
  - action: shell
    resource: "*"
    effect: deny
  - action: edit
    resource: "*"
    effect: deny
---

Load the role and follow its shared workflow policy.

---

A horizontal rule in the body must survive.
`

const renderCRLF = "---\r\ndescription: CRLF role.\r\nmode: subagent\r\nmodel: opencode/deepseek-v4.1-flash-free\r\npermissions:\r\n  - action: \"*\"\r\n    resource: \"*\"\r\n    effect: deny\r\n  - action: read\r\n    resource: \"*\"\r\n    effect: allow\r\n---\r\nBody with CRLF.\r\n"

const renderNoModel = `---
description: Role without a pinned model.
mode: subagent
steps: 10
permissions:
  - action: "*"
    resource: "*"
    effect: deny
  - action: read
    resource: "*"
    effect: allow
---

Body.
`

// adapterPolicyFixture mirrors the reviewed read-only spec/plan adapter
// policy. The real sources live under opencode-pack/adapters/ and are
// fleet-only render sources, not part of the 009 --with-opencode install set.
const adapterPolicyFixture = `---
description: Draft one feature document as a read-only draft returned to the parent.
mode: subagent
model: 9router/ag/claude-opus-5-5-high
permissions:
  - action: "*"
    resource: "*"
    effect: deny
  - action: read
    resource: "*"
    effect: allow
  - action: glob
    resource: "*"
    effect: allow
  - action: grep
    resource: "*"
    effect: allow
  - action: skill
    resource: "*"
    effect: allow
  - action: execute
    resource: "*"
    effect: allow
  - action: codegraph_*
    resource: "*"
    effect: allow
  - action: context7_*
    resource: "*"
    effect: allow
  - action: exa_*
    resource: "*"
    effect: deny
  - action: read
    resource: "*.env"
    effect: deny
  - action: read
    resource: "*.env.*"
    effect: deny
  - action: subagent
    resource: "*"
    effect: deny
  - action: shell
    resource: "*"
    effect: deny
  - action: edit
    resource: "*"
    effect: deny
---

Return the complete draft to the parent.
`

func mustRender(t *testing.T, in RenderInput) Rendered {
	t.Helper()
	rendered, err := Render(in)
	if err != nil {
		t.Fatalf("Render(%s): %v", in.label(), err)
	}
	return rendered
}

func mustFailRender(t *testing.T, in RenderInput, want string) {
	t.Helper()
	_, err := Render(in)
	if err == nil {
		t.Fatalf("Render(%s) succeeded, want error containing %q", in.label(), want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Render(%s) error = %q, want substring %q", in.label(), err, want)
	}
}

func bodyBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	parts, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("splitFrontmatter: %v", err)
	}
	return parts.body
}

func TestFleetRender(t *testing.T) {
	t.Run("identical without override", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{InstanceID: "sdd-implement-1", SourceBytes: []byte(renderCanonical)})
		if !bytes.Equal(rendered.Bytes, []byte(renderCanonical)) {
			t.Fatal("render without a model changed canonical bytes")
		}
		if rendered.ModelChanged {
			t.Fatal("render without a model reported a model change")
		}
		if rendered.DesiredDigest != digestBytes(rendered.Bytes) {
			t.Fatal("DesiredDigest is not the digest of the staged bytes")
		}
	})

	t.Run("identical with same model", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{
			InstanceID:  "sdd-implement-1",
			SourceBytes: []byte(renderCanonical),
			Model:       "opencode/deepseek-v4.1-flash-free",
		})
		if !bytes.Equal(rendered.Bytes, []byte(renderCanonical)) {
			t.Fatal("re-rendering the canonical model changed bytes")
		}
		if rendered.ModelChanged {
			t.Fatal("equal model reported a change")
		}
	})

	t.Run("override changes only model", func(t *testing.T) {
		const model = "9router/ag/gemini-3.8-flash-high#high"
		rendered := mustRender(t, RenderInput{
			InstanceID:  "sdd-implement-9r-1",
			SourceBytes: []byte(renderCanonical),
			Model:       model,
		})
		if !rendered.ModelChanged {
			t.Fatal("model override was not reported")
		}
		if !bytes.Contains(rendered.Bytes, []byte("model: "+model+"\n")) {
			t.Fatalf("rendered file does not contain the selected model:\n%s", rendered.Bytes)
		}
		if !bytes.Equal(bodyBytes(t, rendered.Bytes), bodyBytes(t, []byte(renderCanonical))) {
			t.Fatal("body bytes changed")
		}
		if err := VerifyModelOnly([]byte(renderCanonical), rendered.Bytes); err != nil {
			t.Fatalf("VerifyModelOnly: %v", err)
		}
		sourceDigest, err := PermissionDigest([]byte(renderCanonical))
		if err != nil {
			t.Fatal(err)
		}
		renderedDigest, err := PermissionDigest(rendered.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if sourceDigest != renderedDigest {
			t.Fatal("permission digest changed on a model-only render")
		}
	})

	t.Run("re-render is identical", func(t *testing.T) {
		in := RenderInput{
			InstanceID:  "sdd-implement-9r-1",
			SourceBytes: []byte(renderCanonical),
			Model:       "9router/ag/gemini-3.8-flash-high#high",
		}
		first := mustRender(t, in)
		second := mustRender(t, RenderInput{InstanceID: in.InstanceID, SourceBytes: first.Bytes, Model: in.Model})
		if !bytes.Equal(first.Bytes, second.Bytes) {
			t.Fatal("re-rendering staged bytes was not identical")
		}
		if second.ModelChanged {
			t.Fatal("re-rendering staged bytes reported a model change")
		}
	})

	t.Run("variant preserved", func(t *testing.T) {
		source := strings.Replace(renderCanonical, "model: opencode/deepseek-v4.1-flash-free", "model: opencode-go/deepseek-v4.1-flash#high", 1)
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(source), Model: "opencode-go/qwen3.8-flash#medium"})
		if !bytes.Contains(rendered.Bytes, []byte("model: opencode-go/qwen3.8-flash#medium\n")) {
			t.Fatalf("variant was not preserved:\n%s", rendered.Bytes)
		}
	})

	t.Run("CRLF preserved", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(renderCRLF), Model: "9router/ag/gemini-3.8-flash-high"})
		if !bytes.HasPrefix(rendered.Bytes, []byte("---\r\n")) {
			t.Fatal("opening delimiter line ending changed")
		}
		if !bytes.Contains(rendered.Bytes, []byte("model: 9router/ag/gemini-3.8-flash-high\r\n")) {
			t.Fatalf("replacement line did not use CRLF:\n%q", rendered.Bytes)
		}
		if !bytes.Equal(bodyBytes(t, rendered.Bytes), bodyBytes(t, []byte(renderCRLF))) {
			t.Fatal("CRLF body changed")
		}
	})

	t.Run("CRLF model insert", func(t *testing.T) {
		crlfNoModel := strings.ReplaceAll(renderNoModel, "\n", "\r\n")
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(crlfNoModel), Model: "9router/ag/gemini-3.8-flash-high"})
		if !bytes.Contains(rendered.Bytes, []byte("mode: subagent\r\nmodel: 9router/ag/gemini-3.8-flash-high\r\nsteps: 10\r\n")) {
			t.Fatalf("model was not inserted with CRLF:\n%s", rendered.Bytes)
		}
		if bytes.Contains(bytes.ReplaceAll(rendered.Bytes, []byte("\r\n"), nil), []byte("\n")) {
			t.Fatalf("rendered CRLF file contains lone LF:\n%q", rendered.Bytes)
		}
	})

	t.Run("block scalar fail closed", func(t *testing.T) {
		blockScalarSource := strings.Replace(renderCanonical, "model: opencode/deepseek-v4.1-flash-free", "model: |\n  opencode/deepseek-v4.1-flash-free", 1)
		mustFailRender(t, RenderInput{SourceBytes: []byte(blockScalarSource), Model: "9router/ag/gemini-3.8-flash-high"}, "block scalar")
	})

	t.Run("missing model inserted after mode", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(renderNoModel), Model: "opencode/beam-free"})
		if !bytes.Contains(rendered.Bytes, []byte("mode: subagent\nmodel: opencode/beam-free\nsteps: 10\n")) {
			t.Fatalf("model was not inserted after mode:\n%s", rendered.Bytes)
		}
		if !bytes.Equal(bodyBytes(t, rendered.Bytes), bodyBytes(t, []byte(renderNoModel))) {
			t.Fatal("body changed when inserting a model")
		}
		again := mustRender(t, RenderInput{SourceBytes: rendered.Bytes, Model: "opencode/beam-free"})
		if !bytes.Equal(rendered.Bytes, again.Bytes) {
			t.Fatal("inserted model was not stable on re-render")
		}
	})

	t.Run("missing model appended without mode", func(t *testing.T) {
		source := "---\ndescription: No mode.\n---\nBody.\n"
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(source), Model: "opencode/beam-free"})
		if !bytes.Contains(rendered.Bytes, []byte("description: No mode.\nmodel: opencode/beam-free\n---\n")) {
			t.Fatalf("model was not appended to frontmatter:\n%s", rendered.Bytes)
		}
	})

	t.Run("duplicate model rejected", func(t *testing.T) {
		source := strings.Replace(renderCanonical, "model: opencode/deepseek-v4.1-flash-free", "model: opencode/deepseek-v4.1-flash-free\nmodel: opencode/beam-free", 1)
		mustFailRender(t, RenderInput{SourceBytes: []byte(source), Model: "opencode/beam-free"}, "duplicate model")
	})

	t.Run("dashes in body preserved", func(t *testing.T) {
		source := strings.Replace(renderCanonical, "A horizontal rule in the body must survive.", "---\nAnother body rule.\n---", 1)
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(source), Model: "opencode/beam-free"})
		if !bytes.Equal(bodyBytes(t, rendered.Bytes), bodyBytes(t, []byte(source))) {
			t.Fatal("body containing --- was not preserved")
		}
	})

	t.Run("missing closing delimiter rejected", func(t *testing.T) {
		mustFailRender(t, RenderInput{SourceBytes: []byte("---\ndescription: broken\n")}, "missing closing delimiter")
	})

	t.Run("missing opening delimiter rejected", func(t *testing.T) {
		mustFailRender(t, RenderInput{SourceBytes: []byte("description: broken\n---\n")}, "missing opening delimiter")
	})

	t.Run("orchestrator without model unchanged", func(t *testing.T) {
		source := "---\ndescription: Primary.\nmode: primary\npermissions:\n  - action: subagent\n    resource: \"*\"\n    effect: allow\n---\n\nBody.\n"
		rendered := mustRender(t, RenderInput{InstanceID: "sdd-orchestrator", SourceBytes: []byte(source)})
		if !bytes.Equal(rendered.Bytes, []byte(source)) {
			t.Fatal("orchestrator without a model changed")
		}
	})

	t.Run("invalid model rejected", func(t *testing.T) {
		mustFailRender(t, RenderInput{SourceBytes: []byte(renderCanonical), Model: "bad model"}, "invalid model")
		mustFailRender(t, RenderInput{SourceBytes: []byte(renderCanonical), Model: "bad\nmodel"}, "invalid model")
		mustFailRender(t, RenderInput{SourceBytes: []byte(renderCanonical), Model: " opencode/beam-free"}, "invalid model")
	})

	t.Run("staged bytes round trip in temp dir", func(t *testing.T) {
		dir := t.TempDir()
		const model = "9router/ag/gemini-3.8-flash-high#high"
		rendered := mustRender(t, RenderInput{InstanceID: "sdd-implement-9r-1", SourceBytes: []byte(renderCanonical), Model: model})
		path := filepath.Join(dir, "sdd-implement-9r-1.md")
		if err := os.WriteFile(path, rendered.Bytes, 0o644); err != nil {
			t.Fatalf("stage rendered bytes: %v", err)
		}
		staged, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read staged bytes: %v", err)
		}
		if digestBytes(staged) != rendered.DesiredDigest {
			t.Fatal("staged digest does not match DesiredDigest")
		}
		again := mustRender(t, RenderInput{InstanceID: "sdd-implement-9r-1", SourceBytes: staged, Model: model})
		if !bytes.Equal(again.Bytes, rendered.Bytes) {
			t.Fatal("re-rendering the staged file changed bytes")
		}
	})

	t.Run("render resolved contract", func(t *testing.T) {
		rendered, err := RenderResolved(Resolved{
			InstanceID:  "sdd-implement-9r-1",
			SourceName:  "sdd-implement-1",
			SourceBytes: []byte(renderCanonical),
			Model:       "9router/ag/gemini-3.8-flash-high",
		})
		if err != nil {
			t.Fatalf("RenderResolved: %v", err)
		}
		if rendered.InstanceID != "sdd-implement-9r-1" {
			t.Fatalf("InstanceID = %q", rendered.InstanceID)
		}
		if err := VerifyModelOnly([]byte(renderCanonical), rendered.Bytes); err != nil {
			t.Fatalf("VerifyModelOnly: %v", err)
		}
	})

	t.Run("real pack profile round trip", func(t *testing.T) {
		path := filepath.Join("..", "..", "opencode-pack", "agents", "sdd-architecture-review-9r.md")
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		same := mustRender(t, RenderInput{SourceBytes: source, Model: "9router/ag/claude-opus-5-5-high"})
		if !bytes.Equal(same.Bytes, source) {
			t.Fatal("re-rendering the real profile changed bytes")
		}
		changed := mustRender(t, RenderInput{SourceBytes: source, Model: "9router/ag/gemini-3.8-flash-high"})
		if err := VerifyModelOnly(source, changed.Bytes); err != nil {
			t.Fatalf("VerifyModelOnly on real profile: %v", err)
		}
		if !bytes.Equal(bodyBytes(t, source), bodyBytes(t, changed.Bytes)) {
			t.Fatal("real profile body changed")
		}
	})

	t.Run("real pack render all 14", func(t *testing.T) {
		agentsDir := filepath.Join("..", "..", "opencode-pack", "agents")
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			t.Fatalf("read agents dir: %v", err)
		}

		var count int
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			count++
			name := e.Name()
			instanceID := strings.TrimSuffix(name, ".md")
			path := filepath.Join(agentsDir, name)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			// 1. Re-render without model change (or no-op for orchestrator)
			same := mustRender(t, RenderInput{InstanceID: instanceID, SourceBytes: source})
			if !bytes.Equal(same.Bytes, source) {
				t.Fatalf("%s: re-rendering without model change modified bytes", name)
			}

			// 2. Render with model override
			const overrideModel = "9router/ag/gemini-3.8-flash-high#high"
			rendered := mustRender(t, RenderInput{InstanceID: instanceID, SourceBytes: source, Model: overrideModel})

			// 3. Verify body bytes are identical
			if !bytes.Equal(bodyBytes(t, source), bodyBytes(t, rendered.Bytes)) {
				t.Fatalf("%s: body bytes changed after model override", name)
			}

			// 4. Verify model-only contract and permissions preservation
			if err := VerifyModelOnly(source, rendered.Bytes); err != nil {
				t.Fatalf("%s: VerifyModelOnly failed: %v", name, err)
			}

			// 5. Verify re-render stability
			again := mustRender(t, RenderInput{InstanceID: instanceID, SourceBytes: rendered.Bytes, Model: overrideModel})
			if !bytes.Equal(again.Bytes, rendered.Bytes) {
				t.Fatalf("%s: re-render not stable", name)
			}

			// 6. Verify desired digest matches staged bytes
			if rendered.DesiredDigest != digestBytes(rendered.Bytes) {
				t.Fatalf("%s: DesiredDigest mismatch", name)
			}
		}

		if count != 14 {
			t.Fatalf("expected exactly 14 agent profiles in opencode-pack/agents, found %d", count)
		}
	})
}

func TestFleetPermissions(t *testing.T) {
	t.Run("model override preserves normalized set", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(renderCanonical), Model: "opencode/beam-free"})
		equal, err := PermissionSetsEqual([]byte(renderCanonical), rendered.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Fatal("model-only render changed the permission set")
		}
	})

	t.Run("normalization ignores order quoting and line endings", func(t *testing.T) {
		reordered := `---
description: Reordered.
mode: subagent
model: opencode/beam-free
permissions:
  - action: EDIT
    resource: "*"
    effect: DENY
  - action: read
    resource: "*.env"
    effect: deny
  - action: "read"
    resource: "*"
    effect: "allow"
  - action: "*"
    resource: "*"
    effect: deny
---
Body.
`
		source := "---\r\ndescription: Reordered.\r\nmode: subagent\r\nmodel: opencode/beam-free\r\npermissions:\r\n  - action: \"*\"\r\n    resource: \"*\"\r\n    effect: deny\r\n  - action: read\r\n    resource: \"*\"\r\n    effect: allow\r\n  - action: read\r\n    resource: \"*.env\"\r\n    effect: deny\r\n  - action: edit\r\n    resource: \"*\"\r\n    effect: deny\r\n---\r\nBody.\r\n"
		equal, err := PermissionSetsEqual([]byte(source), []byte(reordered))
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Fatal("normalized permission sets should be equal")
		}
		sourceDigest, err := PermissionDigest([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		reorderedDigest, err := PermissionDigest([]byte(reordered))
		if err != nil {
			t.Fatal(err)
		}
		if sourceDigest != reorderedDigest {
			t.Fatal("permission digests differ for equivalent sets")
		}
	})

	t.Run("permission delta fails closed", func(t *testing.T) {
		altered := strings.Replace(renderCanonical, "  - action: edit\n    resource: \"*\"\n    effect: deny", "  - action: edit\n    resource: \"*\"\n    effect: allow", 1)
		if altered == renderCanonical {
			t.Fatal("fixture did not change")
		}
		if err := VerifyModelOnly([]byte(renderCanonical), []byte(altered)); err == nil || !strings.Contains(err.Error(), "permission set changed") {
			t.Fatalf("VerifyModelOnly error = %v, want permission set changed", err)
		}
		equal, err := PermissionSetsEqual([]byte(renderCanonical), []byte(altered))
		if err != nil {
			t.Fatal(err)
		}
		if equal {
			t.Fatal("edit deny to allow compared equal")
		}
	})

	t.Run("body delta fails closed", func(t *testing.T) {
		altered := renderCanonical + "extra\n"
		if err := VerifyModelOnly([]byte(renderCanonical), []byte(altered)); err == nil || !strings.Contains(err.Error(), "body bytes changed") {
			t.Fatalf("VerifyModelOnly error = %v, want body bytes changed", err)
		}
	})

	t.Run("frontmatter delta fails closed", func(t *testing.T) {
		altered := strings.Replace(renderCanonical, "description: Example implementation role.", "description: Changed role.", 1)
		if err := VerifyModelOnly([]byte(renderCanonical), []byte(altered)); err == nil || !strings.Contains(err.Error(), "frontmatter changed outside model") {
			t.Fatalf("VerifyModelOnly error = %v, want frontmatter changed outside model", err)
		}
	})

	t.Run("MCP wildcard lanes preserved", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(renderCanonical), Model: "opencode/beam-free"})
		rules, err := parsePermissions(frontmatterFields(t, rendered.Bytes))
		if err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"execute", "codegraph_*", "context7_*", "exa_*", "external_directory"} {
			if !hasRule(rules, action, "*", "allow") && !hasRule(rules, action, "*", "deny") {
				t.Fatalf("lane %q disappeared after render", action)
			}
		}
		narrowed := strings.Replace(renderCanonical, "  - action: codegraph_*\n    resource: \"*\"\n    effect: allow", "  - action: codegraph_*\n    resource: \"*\"\n    effect: deny", 1)
		equal, err := PermissionSetsEqual([]byte(renderCanonical), []byte(narrowed))
		if err != nil {
			t.Fatal(err)
		}
		if equal {
			t.Fatal("codegraph lane change compared equal")
		}
	})

	t.Run("conflicting duplicate rules rejected", func(t *testing.T) {
		source := `---
description: Conflicting.
mode: subagent
permissions:
  - action: read
    resource: "*"
    effect: allow
  - action: read
    resource: "*"
    effect: deny
---
Body.
`
		if _, err := PermissionDigest([]byte(source)); err == nil || !strings.Contains(err.Error(), "conflicting permission rules") {
			t.Fatalf("PermissionDigest error = %v, want conflicting permission rules", err)
		}
	})

	t.Run("adapter policy is deny by default and non-writing", func(t *testing.T) {
		fields, rules := parseAdapterPolicy(t, []byte(adapterPolicyFixture))
		assertAdapterPolicy(t, "fixture", fields, rules)
	})

	t.Run("real adapter sources match the reviewed policy", func(t *testing.T) {
		fixtureDigest, err := PermissionDigest([]byte(adapterPolicyFixture))
		if err != nil {
			t.Fatalf("PermissionDigest(fixture): %v", err)
		}
		for _, name := range []string{"sdd-spec-9r", "sdd-plan-9r"} {
			path := filepath.Join("..", "..", "opencode-pack", "adapters", name+".md")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			fields, rules := parseAdapterPolicy(t, data)
			assertAdapterPolicy(t, name, fields, rules)
			digest, err := PermissionDigest(data)
			if err != nil {
				t.Fatalf("PermissionDigest(%s): %v", name, err)
			}
			if digest != fixtureDigest {
				t.Fatalf("%s permission set differs from the reviewed adapter policy fixture", name)
			}
		}
	})

	t.Run("adapter model override preserves policy", func(t *testing.T) {
		rendered := mustRender(t, RenderInput{SourceBytes: []byte(adapterPolicyFixture), Model: "9router/ag/gemini-3.8-flash-high"})
		if err := VerifyModelOnly([]byte(adapterPolicyFixture), rendered.Bytes); err != nil {
			t.Fatalf("VerifyModelOnly: %v", err)
		}
		equal, err := PermissionSetsEqual([]byte(adapterPolicyFixture), rendered.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Fatal("adapter policy changed on model override")
		}
	})
}

func frontmatterFields(t *testing.T, data []byte) []byte {
	t.Helper()
	parts, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("splitFrontmatter: %v", err)
	}
	return parts.fields
}

func parseAdapterPolicy(t *testing.T, data []byte) (map[string]string, []permissionRule) {
	t.Helper()
	parts, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("splitFrontmatter: %v", err)
	}
	fields := map[string]string{}
	for _, line := range scanFMLines(parts.fields) {
		content := bytes.TrimSuffix(parts.fields[line.start:line.contentEnd], []byte("\r"))
		key, value, ok := topLevelKey(content)
		if !ok {
			continue
		}
		fields[key] = scalarValue(value)
	}
	rules, err := parsePermissions(parts.fields)
	if err != nil {
		t.Fatalf("parsePermissions: %v", err)
	}
	return fields, rules
}

// assertAdapterPolicy is the reviewed contract for a fleet spec/plan adapter:
// read-only, non-writing and non-shell, with no delegation or external search.
func assertAdapterPolicy(t *testing.T, label string, fields map[string]string, rules []permissionRule) {
	t.Helper()
	if fields["mode"] != "subagent" {
		t.Fatalf("%s mode = %q, want subagent", label, fields["mode"])
	}
	if fields["model"] == "" {
		t.Fatalf("%s has no model", label)
	}
	if len(rules) == 0 {
		t.Fatalf("%s has no permission rules", label)
	}
	first := rules[0]
	if first.Action != "*" || first.Resource != "*" || first.Effect != "deny" {
		t.Fatalf("%s first rule = %+v, want blanket deny", label, first)
	}
	for _, action := range []string{"read", "glob", "grep", "skill", "execute", "codegraph_*", "context7_*"} {
		if !hasRule(rules, action, "*", "allow") {
			t.Fatalf("%s missing allow for %s", label, action)
		}
	}
	for _, action := range []string{"subagent", "shell", "edit", "external_directory", "exa_*"} {
		if hasRule(rules, action, "*", "allow") {
			t.Fatalf("%s must not allow %s", label, action)
		}
	}
	for _, action := range []string{"subagent", "shell", "edit", "exa_*"} {
		if !hasRule(rules, action, "*", "deny") {
			t.Fatalf("%s missing deny for %s", label, action)
		}
	}
	for _, resource := range []string{"*.env", "*.env.*"} {
		if !hasRule(rules, "read", resource, "deny") {
			t.Fatalf("%s missing read deny for %s", label, resource)
		}
	}
	if last := rules[len(rules)-1]; last.Action != "edit" || last.Effect != "deny" {
		t.Fatalf("%s last rule = %+v, want edit deny", label, last)
	}
}

func hasRule(rules []permissionRule, action, resource, effect string) bool {
	for _, rule := range rules {
		if strings.EqualFold(rule.Action, action) && rule.Resource == resource && strings.EqualFold(rule.Effect, effect) {
			return true
		}
	}
	return false
}
