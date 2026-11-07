package skillsync

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	skillsfiles "agentic-sdd"
)

// Feature 009 / Batch B1 / T02 — Check 1.
//
// This is a static distribution and permission contract for the opt-in
// OpenCode pack. It verifies packaging and frontmatter shape, not whether an
// agent obeys prose. Both sources are checked: the tree embedded in the
// binary (what an installed agentic-sdd actually ships) and the on-disk
// directory (what `--repo` serves), so the two cannot silently drift.

const openCodePackDir = "opencode-pack"

var openCodePackFiles = []string{
	"skills/agentic-sdd-orchestrate/SKILL.md",
	"skills/agentic-sdd-orchestrate/references/dispatch-contract.md",
	"skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md",
	"skills/agentic-sdd-orchestrate/references/context-and-cost.md",
	"routing.md",
	"README.md",
}

// openCodeAgentProfiles is the exact `--with-opencode` set: nine new
// profiles (including the two optional OpenAI specialists) plus the five
// deny-by-default profiles that are backed up and updated in place.
var openCodeAgentProfiles = []string{
	"sdd-orchestrator",
	"sdd-debug-diagnose",
	"sdd-test-design",
	"sdd-research-librarian",
	"sdd-docs-worker",
	"sdd-architecture-advisor",
	"sdd-visual-inspector",
	"sdd-openai-architect",
	"sdd-openai-implement",
	"sdd-architecture-review-9r",
	"sdd-code-review-9r",
	"sdd-review-verification-1",
	"sdd-review-verification-2",
	"sdd-lightweight",
}

// openCodeFreeFirstProfiles is the opt-in `--opencode-free-first` set.
var openCodeFreeFirstProfiles = []string{
	"sdd-implement-1",
	"sdd-implement-2",
	"sdd-test-implement-1",
	"sdd-test-implement-2",
}

// openCodeHighIntensityRoles are the five profiles allowed to reach the
// external Exa tools. Every other deny-by-default profile must deny exa_*.
var openCodeHighIntensityRoles = map[string]bool{
	"sdd-architecture-review-9r": true,
	"sdd-code-review-9r":         true,
	"sdd-architecture-advisor":   true,
	"sdd-research-librarian":     true,
	"sdd-openai-architect":       true,
}

// openCodeReviewRoles may read outside the workspace and run the two
// read-only git inspection commands.
var openCodeReviewRoles = map[string]bool{
	"sdd-architecture-review-9r": true,
	"sdd-code-review-9r":         true,
	"sdd-review-verification-1":  true,
	"sdd-review-verification-2":  true,
	"sdd-openai-architect":       true,
}

// openCodeDenyByDefaultRoles are every read-only subagent.
var openCodeDenyByDefaultRoles = []string{
	"sdd-debug-diagnose",
	"sdd-test-design",
	"sdd-research-librarian",
	"sdd-architecture-advisor",
	"sdd-visual-inspector",
	"sdd-openai-architect",
	"sdd-architecture-review-9r",
	"sdd-code-review-9r",
	"sdd-review-verification-1",
	"sdd-review-verification-2",
	"sdd-lightweight",
}

// openCodeAgentModels pins the model each managed agent profile must declare.
// sdd-orchestrator is deliberately absent: it uses the session model.
var openCodeAgentModels = map[string]string{
	"sdd-debug-diagnose":         "opencode/deepseek-v4.1-flash-free",
	"sdd-test-design":            "opencode/deepseek-v4.1-flash-free",
	"sdd-research-librarian":     "opencode/deepseek-v4.1-flash-free",
	"sdd-docs-worker":            "opencode/deepseek-v4.1-flash-free",
	"sdd-architecture-advisor":   "9router/ag/claude-sonnet-5-5-high",
	"sdd-visual-inspector":       "opencode/glm-5.3-flash-free",
	"sdd-openai-architect":       "openai/gpt-6.1-sol",
	"sdd-openai-implement":       "openai/gpt-6.1-terra",
	"sdd-architecture-review-9r": "9router/ag/claude-opus-5-5-high",
	"sdd-code-review-9r":         "9router/ag/claude-opus-5-5-high",
	"sdd-review-verification-1":  "opencode-go/glm-5.3-flash",
	"sdd-review-verification-2":  "opencode/beam-free",
	"sdd-lightweight":            "opencode/beam-free",
}

// openCodeRequiredLinks lists the links the skill must contain. Sibling
// Agentic-SDD skills are linked with ../ and are installed next to this
// skill, so they resolve inside skills-files/.
var openCodeRequiredLinks = []string{
	"references/dispatch-contract.md",
	"references/failure-and-reconciliation.md",
	"references/context-and-cost.md",
	"../agentic-sdd-router/references/workflow-policy.md",
	"../agentic-sdd-router/references/human-input.md",
}

// openCodeV1Patterns matches V1 frontmatter keys and action names that
// OpenCode v2 rejects. `permissions:` must not match `permission:`.
var openCodeV1Patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^permission\s*:`),
	regexp.MustCompile(`(?m)^tools\s*:`),
	regexp.MustCompile(`(?m)^\s*(?:-\s*)?action\s*:\s*["']?(bash|task|write|patch)["']?\s*$`),
}

// permissionRule is one { action, resource, effect } entry.
type permissionRule struct {
	action   string
	resource string
	effect   string
}

func (r permissionRule) String() string {
	return fmt.Sprintf("{action: %q, resource: %q, effect: %q}", r.action, r.resource, r.effect)
}

// openCodePackRoots returns the embedded and on-disk pack trees keyed by
// source name.
func openCodePackRoots(t *testing.T) map[string]fs.FS {
	t.Helper()
	embedded, err := fs.Sub(skillsfiles.OpenCodePackFiles, openCodePackDir)
	if err != nil {
		t.Fatalf("embedded pack: %v", err)
	}
	return map[string]fs.FS{
		"embedded": embedded,
		"on-disk":  os.DirFS(filepath.Join("..", "..", openCodePackDir)),
	}
}

func openCodeSkillsRoots(t *testing.T) map[string]fs.FS {
	t.Helper()
	embedded, err := fs.Sub(skillsfiles.Files, "skills-files")
	if err != nil {
		t.Fatalf("embedded skills: %v", err)
	}
	return map[string]fs.FS{
		"embedded": embedded,
		"on-disk":  os.DirFS(filepath.Join("..", "..", "skills-files")),
	}
}

// TestOpenCodePackAssetsAndPermissions is Check 1 from spec.md §7.
func TestOpenCodePackAssetsAndPermissions(t *testing.T) {
	packRoots := openCodePackRoots(t)
	skillsRoots := openCodeSkillsRoots(t)

	var everyFile []string
	everyFile = append(everyFile, openCodePackFiles...)
	for _, name := range openCodeAgentProfiles {
		everyFile = append(everyFile, "agents/"+name+".md")
	}
	for _, name := range openCodeFreeFirstProfiles {
		everyFile = append(everyFile, "presets/free-first/"+name+".md")
	}

	// Every required asset exists as a regular file in both trees, and the
	// embedded copy is byte-identical to the on-disk copy.
	for _, rootName := range []string{"embedded", "on-disk"} {
		root := packRoots[rootName]
		for _, name := range everyFile {
			info, err := fs.Stat(root, name)
			if err != nil {
				t.Fatalf("%s: required pack file %q missing: %v", rootName, name, err)
			}
			if !info.Mode().IsRegular() {
				t.Fatalf("%s: %q is not a regular file (mode %v)", rootName, name, info.Mode())
			}
			if info.Size() == 0 {
				t.Fatalf("%s: %q is empty", rootName, name)
			}
		}
	}
	for _, name := range everyFile {
		embedded, err := fs.ReadFile(packRoots["embedded"], name)
		if err != nil {
			t.Fatalf("embedded read %q: %v", name, err)
		}
		onDisk, err := fs.ReadFile(packRoots["on-disk"], name)
		if err != nil {
			t.Fatalf("on-disk read %q: %v", name, err)
		}
		if string(embedded) != string(onDisk) {
			t.Fatalf("%q differs between the embedded and on-disk pack", name)
		}
	}

	// SKILL.md frontmatter names the skill and has a usable description, and
	// every relative link resolves in both trees.
	skillData := openCodeReadFile(t, packRoots["on-disk"], "skills/agentic-sdd-orchestrate/SKILL.md")
	fields, _ := openCodeParseFrontmatter(t, "SKILL.md", skillData)
	if got := fields["name"]; got != "agentic-sdd-orchestrate" {
		t.Fatalf("SKILL.md name = %q, want agentic-sdd-orchestrate", got)
	}
	if strings.TrimSpace(fields["description"]) == "" {
		t.Fatal("SKILL.md has no description")
	}
	seenLinks := map[string]bool{}
	for _, match := range regexp.MustCompile(`\]\(([^)\s]+)\)`).FindAllStringSubmatch(string(skillData), -1) {
		target := match[1]
		if i := strings.IndexByte(target, '#'); i >= 0 {
			target = target[:i]
		}
		if target == "" || strings.Contains(target, "://") {
			continue
		}
		seenLinks[target] = true
		var roots map[string]fs.FS
		var rel string
		if strings.HasPrefix(target, "../agentic-sdd-router/") {
			roots = skillsRoots
			rel = strings.TrimPrefix(target, "../")
		} else {
			roots = packRoots
			rel = path.Join("skills/agentic-sdd-orchestrate", target)
		}
		for _, rootName := range []string{"embedded", "on-disk"} {
			info, err := fs.Stat(roots[rootName], rel)
			if err != nil || !info.Mode().IsRegular() {
				t.Fatalf("%s: SKILL.md link %q does not resolve to %q: %v", rootName, target, rel, err)
			}
		}
	}
	for _, link := range openCodeRequiredLinks {
		if !seenLinks[link] {
			t.Fatalf("SKILL.md does not link %q", link)
		}
	}

	// Agent and preset frontmatter: valid v2 shape, no v1 keys or actions.
	for _, name := range openCodeAgentProfiles {
		openCodeCheckProfile(t, packRoots, "agents/"+name+".md", name)
	}
	for _, name := range openCodeFreeFirstProfiles {
		openCodeCheckProfile(t, packRoots, "presets/free-first/"+name+".md", name)
	}

	// Deny-by-default ordering: * deny first, MCP allows present, role-specific
	// exa allow or deny, and subagent/shell/edit denies last so last-match-wins
	// keeps them enforced.
	for _, name := range openCodeDenyByDefaultRoles {
		rel := "agents/" + name + ".md"
		data := openCodeReadFile(t, packRoots["on-disk"], rel)
		_, rules := openCodeParseFrontmatter(t, rel, data)
		if len(rules) < 2 {
			t.Fatalf("%s: expected permission rules, got %d", rel, len(rules))
		}
		first := rules[0]
		if first.action != "*" || first.resource != "*" || first.effect != "deny" {
			t.Fatalf("%s: first permission rule = %v, want {action: \"*\", resource: \"*\", effect: \"deny\"}", rel, first)
		}
		for _, action := range []string{"read", "glob", "grep", "skill", "execute", "codegraph_*", "context7_*"} {
			if !openCodeHasRule(rules, action, "*", "allow") {
				t.Fatalf("%s: missing allow rule for %s", rel, action)
			}
		}
		// MCP allows must appear after the blanket deny.
		for _, action := range []string{"execute", "codegraph_*", "context7_*"} {
			if openCodeRuleIndex(rules, action, "*", "allow") < 1 {
				t.Fatalf("%s: %s allow precedes the blanket deny", rel, action)
			}
		}
		if openCodeHighIntensityRoles[name] {
			for _, action := range []string{"exa_web_search_exa", "exa_web_fetch_exa"} {
				if !openCodeHasRule(rules, action, "*", "allow") {
					t.Fatalf("%s: high-intensity role is missing %s", rel, action)
				}
			}
			if openCodeHasRule(rules, "exa_*", "*", "deny") {
				t.Fatalf("%s: high-intensity role must not deny exa_*", rel)
			}
		} else if !openCodeHasRule(rules, "exa_*", "*", "deny") {
			t.Fatalf("%s: routine role must explicitly deny exa_*", rel)
		}
		for _, resource := range []string{"*.env", "*.env.*"} {
			if !openCodeHasRule(rules, "read", resource, "deny") {
				t.Fatalf("%s: missing read deny for %s", rel, resource)
			}
		}
		if name == "sdd-lightweight" {
			for _, resource := range []string{"*secret*", "*credential*"} {
				if !openCodeHasRule(rules, "read", resource, "deny") {
					t.Fatalf("%s: missing read deny for %s", rel, resource)
				}
			}
		}
		if !openCodeHasRule(rules, "subagent", "*", "deny") {
			t.Fatalf("%s: missing subagent deny", rel)
		}
		last := rules[len(rules)-1]
		if last.action != "edit" || last.resource != "*" || last.effect != "deny" {
			t.Fatalf("%s: final permission rule = %v, want an edit deny", rel, last)
		}
		if idx := openCodeRuleIndex(rules, "subagent", "*", "deny"); idx < len(rules)-3 {
			t.Fatalf("%s: subagent deny at %d is not trailing (%d rules)", rel, idx, len(rules))
		}
		if openCodeReviewRoles[name] {
			if !openCodeHasRule(rules, "external_directory", "*", "allow") {
				t.Fatalf("%s: review role is missing the external_directory allow", rel)
			}
			for _, command := range []string{"git diff", "git status --short"} {
				if !openCodeHasRule(rules, "shell", command, "allow") {
					t.Fatalf("%s: review role is missing shell allow %q", rel, command)
				}
			}
			if openCodeHasRule(rules, "shell", "*", "deny") {
				t.Fatalf("%s: review role must not deny all shell", rel)
			}
		} else if !openCodeHasRule(rules, "shell", "*", "deny") {
			t.Fatalf("%s: read-only role must deny shell", rel)
		}
	}

	// The docs writer is the only writing specialist: it allows edit but still
	// denies shell, subagent and external search.
	docsRel := "agents/sdd-docs-worker.md"
	docsData := openCodeReadFile(t, packRoots["on-disk"], docsRel)
	_, docsRules := openCodeParseFrontmatter(t, docsRel, docsData)
	if docsRules[0].action != "*" || docsRules[0].resource != "*" || docsRules[0].effect != "deny" {
		t.Fatalf("%s: first permission rule must be the blanket deny, got %v", docsRel, docsRules[0])
	}
	for _, action := range []string{"read", "edit", "glob", "grep", "skill", "execute", "codegraph_*", "context7_*"} {
		if !openCodeHasRule(docsRules, action, "*", "allow") {
			t.Fatalf("%s: missing allow rule for %s", docsRel, action)
		}
	}
	for _, action := range []string{"subagent", "shell", "exa_*"} {
		if !openCodeHasRule(docsRules, action, "*", "deny") {
			t.Fatalf("%s: missing deny rule for %s", docsRel, action)
		}
	}

	// sdd-openai-implement is the optional OpenAI GPT-6.1 Terra writer: allows
	// edit/execute/codegraph/context7, denies .env reads, subagent and exa_*.
	openAIImplRel := "agents/sdd-openai-implement.md"
	openAIImplData := openCodeReadFile(t, packRoots["on-disk"], openAIImplRel)
	_, openAIImplRules := openCodeParseFrontmatter(t, openAIImplRel, openAIImplData)
	for _, action := range []string{"edit", "execute", "codegraph_*", "context7_*"} {
		if !openCodeHasRule(openAIImplRules, action, "*", "allow") {
			t.Fatalf("%s: missing allow rule for %s", openAIImplRel, action)
		}
	}
	for _, action := range []string{"subagent", "exa_*"} {
		if !openCodeHasRule(openAIImplRules, action, "*", "deny") {
			t.Fatalf("%s: missing deny rule for %s", openAIImplRel, action)
		}
	}
	if !strings.Contains(string(openAIImplData), "blocked") {
		t.Fatalf("%s: missing stop/blocked rule on missing model access or batch authorization", openAIImplRel)
	}

	// The primary profile uses the session model: no pinned model key, and it
	// allows subagent.
	orchestrator := openCodeReadFile(t, packRoots["on-disk"], "agents/sdd-orchestrator.md")
	orchFields, orchRules := openCodeParseFrontmatter(t, "sdd-orchestrator.md", orchestrator)
	if orchFields["mode"] != "primary" {
		t.Fatalf("sdd-orchestrator mode = %q, want primary", orchFields["mode"])
	}
	if _, ok := orchFields["model"]; ok {
		t.Fatalf("sdd-orchestrator must not pin a model, found %q", orchFields["model"])
	}
	if !openCodeHasRule(orchRules, "subagent", "*", "allow") {
		t.Fatal("sdd-orchestrator must allow the subagent action")
	}

	// The four presets are Free-first writers: deepseek free, MCP allows, and
	// explicit subagent/exa denies.
	for _, name := range openCodeFreeFirstProfiles {
		rel := "presets/free-first/" + name + ".md"
		data := openCodeReadFile(t, packRoots["on-disk"], rel)
		fields, rules := openCodeParseFrontmatter(t, rel, data)
		if fields["mode"] != "subagent" {
			t.Fatalf("%s: mode = %q, want subagent", rel, fields["mode"])
		}
		if got := fields["model"]; got != "opencode/deepseek-v4.1-flash-free" {
			t.Fatalf("%s: model = %q, want opencode/deepseek-v4.1-flash-free", rel, got)
		}
		for _, action := range []string{"edit", "execute", "codegraph_*", "context7_*"} {
			if !openCodeHasRule(rules, action, "*", "allow") {
				t.Fatalf("%s: missing allow rule for %s", rel, action)
			}
		}
		for _, action := range []string{"subagent", "exa_*"} {
			if !openCodeHasRule(rules, action, "*", "deny") {
				t.Fatalf("%s: missing deny rule for %s", rel, action)
			}
		}
	}
}

// openCodeCheckProfile parses one agent or preset file and asserts the shared
// v2 contract: required scalar keys, at least one permission rule, and no v1
// keys or action names anywhere in the file.
func openCodeCheckProfile(t *testing.T, roots map[string]fs.FS, rel, name string) {
	t.Helper()
	data := openCodeReadFile(t, roots["on-disk"], rel)
	for _, pattern := range openCodeV1Patterns {
		if match := pattern.Find(data); match != nil {
			t.Fatalf("%s: uses OpenCode v1 syntax %q", rel, match)
		}
	}
	fields, rules := openCodeParseFrontmatter(t, rel, data)
	if strings.TrimSpace(fields["description"]) == "" {
		t.Fatalf("%s: missing description", rel)
	}
	if fields["mode"] == "" {
		t.Fatalf("%s: missing mode", rel)
	}
	if want, ok := openCodeAgentModels[name]; ok {
		if fields["mode"] != "subagent" {
			t.Fatalf("%s: mode = %q, want subagent", rel, fields["mode"])
		}
		if got := fields["model"]; got != want {
			t.Fatalf("%s: model = %q, want %q", rel, got, want)
		}
	}
	if len(rules) == 0 {
		t.Fatalf("%s: missing permissions", rel)
	}
	if name == "sdd-lightweight" && fields["steps"] != "10" {
		t.Fatalf("%s: steps = %q, want 10", rel, fields["steps"])
	}
}

func openCodeReadFile(t *testing.T, root fs.FS, name string) []byte {
	t.Helper()
	data, err := fs.ReadFile(root, name)
	if err != nil {
		t.Fatalf("read %q: %v", name, err)
	}
	return data
}

// openCodeParseFrontmatter reads the leading --- block into scalar fields and
// ordered permission rules. The pack uses a small, fixed YAML subset, so a
// dependency-free parser is sufficient and keeps the shape check honest.
func openCodeParseFrontmatter(t *testing.T, name string, data []byte) (map[string]string, []permissionRule) {
	t.Helper()
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatalf("%s: missing opening frontmatter delimiter", name)
	}
	fields := map[string]string{}
	var rules []permissionRule
	inPermissions := false
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			return fields, rules
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		if strings.HasPrefix(trimmed, "- ") {
			if !inPermissions {
				t.Fatalf("%s: list item outside permissions: %q", name, trimmed)
			}
			key, value, ok := openCodeScalar(strings.TrimPrefix(trimmed, "- "))
			if !ok || key != "action" {
				t.Fatalf("%s: malformed permission item %q", name, trimmed)
			}
			rules = append(rules, permissionRule{action: value})
			continue
		}
		key, value, ok := openCodeScalar(trimmed)
		if !ok {
			t.Fatalf("%s: unparsable frontmatter line %q", name, trimmed)
		}
		if !indented {
			if key == "permissions" {
				inPermissions = true
				continue
			}
			inPermissions = false
			fields[key] = value
			continue
		}
		if !inPermissions || len(rules) == 0 {
			t.Fatalf("%s: unexpected indented key %q", name, key)
		}
		switch key {
		case "resource":
			rules[len(rules)-1].resource = value
		case "effect":
			rules[len(rules)-1].effect = value
		default:
			t.Fatalf("%s: unexpected permission field %q", name, key)
		}
	}
	t.Fatalf("%s: missing closing frontmatter delimiter", name)
	return nil, nil
}

func openCodeScalar(text string) (string, string, bool) {
	i := strings.IndexByte(text, ':')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(text[:i]), openCodeUnquote(strings.TrimSpace(text[i+1:])), true
}

func openCodeUnquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func openCodeHasRule(rules []permissionRule, action, resource, effect string) bool {
	return openCodeRuleIndex(rules, action, resource, effect) >= 0
}

func openCodeRuleIndex(rules []permissionRule, action, resource, effect string) int {
	for i, rule := range rules {
		if rule.action == action && rule.resource == resource && rule.effect == effect {
			return i
		}
	}
	return -1
}
