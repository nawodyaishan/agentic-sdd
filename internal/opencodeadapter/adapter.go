// Package opencodeadapter prepares inactive OpenCode configuration. It never
// starts a native process, contacts a provider or reads a credential store.
package opencodeadapter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	skillsfiles "agentic-sdd"
)

const Model = "openrouter/nvidia/nemotron-3-ultra-550b-a55b:free"
const Runtime = "2.2.5"
const destination = ".opencode/opencode.jsonc"
const MaxInput = 1 << 20

var identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var metadataID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/#-]{0,127}$`)

type Report struct {
	Version     int      `json:"version"`
	OnlineReady bool     `json:"online_ready"`
	Runtime     string   `json:"target_runtime"`
	Profiles    []string `json:"candidate_profiles"`
	Commands    []string `json:"candidate_commands"`
	Models      []string `json:"candidate_models"`
	Observation string   `json:"observation"`
	Blockers    []string `json:"blockers"`
}

type PreviewResult struct {
	Destination string `json:"destination"`
	Original    string `json:"original"`
	Candidate   string `json:"candidate"`
	Noop        bool   `json:"noop"`
}

type Transaction struct {
	Version       int    `json:"version"`
	Kind          string `json:"kind"`
	Root          string `json:"root"`
	Original      string `json:"original"`
	Installed     string `json:"installed"`
	OriginalMode  uint32 `json:"original_mode"`
	InstalledMode uint32 `json:"installed_mode"`
	Phase         string `json:"phase"`
	Backup        string `json:"backup,omitempty"`
}

func Candidate(feature, buildPath string) ([]byte, error) {
	if (feature != "" && !identifier.MatchString(feature)) || !literalBuildPath(buildPath) {
		return nil, fmt.Errorf("literal feature/build paths required")
	}
	data, err := skillsfiles.AdapterFiles.ReadFile("integrations/opencode/opencode.jsonc")
	if err != nil {
		return nil, fmt.Errorf("embedded candidate unavailable")
	}
	c, err := parseConfig(data)
	if err != nil {
		return nil, err
	}
	c["model"] = Model
	c["compaction"] = map[string]any{"auto": false}
	c["experimental"] = map[string]any{"background_compaction": false}
	for _, value := range c["agents"].(map[string]any) {
		a := value.(map[string]any)
		a["model"] = Model
		var rules []any
		for _, raw := range a["permissions"].([]any) {
			rule := raw.(map[string]any)
			resource := rule["resource"].(string)
			if feature == "" && strings.HasPrefix(resource, "specs/__FEATURE__/") {
				continue
			}
			resource = strings.ReplaceAll(resource, "__FEATURE__", feature)
			rule["resource"] = strings.ReplaceAll(resource, "__BUILD_PATH__", buildPath)
			rules = append(rules, rule)
		}
		a["permissions"] = rules
	}
	commands := c["commands"].(map[string]any)
	for _, raw := range commands {
		raw.(map[string]any)["model"] = Model
	}
	for _, helper := range []string{"grill-me", "grilling", "questionnaire", "writing-for-agents", "handoff", "wait-what", "teach"} {
		agent := "sdd-review"
		if helper == "grill-me" || helper == "grilling" || helper == "questionnaire" {
			agent = "sdd-draft"
		}
		commands["sdd-"+helper] = map[string]any{"description": "Use the scoped Agentic SDD " + helper + " capability", "agent": agent, "model": Model, "subagent": false,
			"template": "Use agentic-sdd-" + helper + " within the actual requested mode and allowed effects. No new authority, children or automatic file writes. Request: $ARGUMENTS"}
	}
	var policies []any
	for _, pair := range [][2]string{{"provider.use", "*"}, {"mcp.use", "*"}, {"plugin.use", "*"}, {"skill.use", "*"}, {"tool.use", "subagent:*"}, {"tool.use", "external_directory:*"}, {"tool.use", "execute:*"}} {
		policies = append(policies, map[string]any{"action": pair[0], "resource": pair[1], "effect": "deny"})
	}
	policies = append(policies, map[string]any{"action": "skill.use", "resource": "agentic-sdd-*", "effect": "allow"})
	c["policies"] = policies
	data, err = json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if _, err = validateCandidate(data); err != nil {
		return nil, err
	}
	return data, nil
}

func literalBuildPath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\:*?[]{}\x00 \t\r\n") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || part == ".git" || part == ".opencode" || part == "secrets" || strings.HasPrefix(part, ".env") {
			return false
		}
	}
	return true
}

func validateCandidate(data []byte) (map[string]any, error) {
	if strings.Contains(string(data), "__FEATURE__") || strings.Contains(string(data), "__BUILD_PATH__") {
		return nil, fmt.Errorf("unresolved candidate placeholder")
	}
	c, err := parseConfig(data)
	if err != nil {
		return nil, err
	}
	if c["model"] != Model || c["warming"] != false || c["formatter"] != false || c["snapshots"] != false {
		return nil, fmt.Errorf("candidate must retain the inactive selected defaults")
	}
	policies, ok := c["policies"].([]any)
	if !ok {
		return nil, fmt.Errorf("candidate policies required")
	}
	denied := map[string]bool{}
	for _, raw := range policies {
		p, ok := raw.(map[string]any)
		if !ok || len(p) != 3 {
			return nil, fmt.Errorf("invalid policy shape")
		}
		a, _ := p["action"].(string)
		r, _ := p["resource"].(string)
		e, _ := p["effect"].(string)
		known := map[string]bool{"provider.use": true, "tool.use": true, "integration.use": true, "integration.connect": true, "mcp.use": true, "skill.use": true, "plugin.use": true}
		if !known[a] || r == "" || (e != "deny" && e != "allow") {
			return nil, fmt.Errorf("unsupported policy statement")
		}
		if e == "allow" && !(a == "skill.use" && r == "agentic-sdd-*") {
			return nil, fmt.Errorf("candidate policy may not enable additional effects")
		}
		if e == "deny" {
			denied[a+":"+r] = true
		}
	}
	for _, key := range []string{"provider.use:*", "mcp.use:*", "plugin.use:*", "skill.use:*", "tool.use:subagent:*", "tool.use:execute:*", "tool.use:external_directory:*"} {
		if !denied[key] {
			return nil, fmt.Errorf("required deny policy missing")
		}
	}
	agents, ok := c["agents"].(map[string]any)
	if !ok || len(agents) == 0 {
		return nil, fmt.Errorf("profiles required")
	}
	for id, raw := range agents {
		a, ok := raw.(map[string]any)
		if !ok || !strings.HasPrefix(id, "sdd-") || !identifier.MatchString(id) || a["mode"] != "primary" || a["model"] != Model {
			return nil, fmt.Errorf("invalid candidate profile")
		}
	}
	commands, ok := c["commands"].(map[string]any)
	if !ok || len(commands) == 0 {
		return nil, fmt.Errorf("commands required")
	}
	for id, raw := range commands {
		cmd, ok := raw.(map[string]any)
		if !ok || !identifier.MatchString(id) || !strings.HasPrefix(id, "sdd-") || cmd["model"] != Model || cmd["subagent"] != false {
			return nil, fmt.Errorf("unsafe command definition")
		}
		template, ok := cmd["template"].(string)
		if !ok || strings.Contains(template, "!`") || strings.Contains(template, "__FEATURE__") || strings.Contains(template, "__BUILD_PATH__") {
			return nil, fmt.Errorf("unsafe or unresolved command template")
		}
		agent, _ := cmd["agent"].(string)
		if _, ok := agents[agent]; !ok {
			return nil, fmt.Errorf("unknown command profile")
		}
		if v, ok := cmd["subtask"]; ok && v != false {
			return nil, fmt.Errorf("ambiguous child command setting")
		}
	}
	defaultAgent, ok := c["default_agent"].(string)
	if !ok {
		return nil, fmt.Errorf("unknown default profile")
	}
	if _, ok := agents[defaultAgent]; !ok {
		return nil, fmt.Errorf("unknown default profile")
	}
	if err := validateAccess(c); err != nil {
		return nil, err
	}
	return c, nil
}

type nativeObservation struct {
	Version   string   `json:"version"`
	Profiles  []string `json:"profiles"`
	Commands  []string `json:"commands"`
	Skills    []string `json:"skills"`
	Warnings  []string `json:"warnings"`
	Truncated bool     `json:"truncated"`
}

// Inspect projects file inputs only. A caller-supplied native projection never
// proves runtime compatibility, permission enforcement or online admission.
func Inspect(config, native []byte) (Report, error) {
	c, err := validateCandidate(config)
	if err != nil {
		return Report{}, err
	}
	report := Report{Version: 1, Runtime: Runtime, Profiles: []string{}, Commands: []string{}, Models: []string{Model}, Observation: "not-observed",
		Blockers: []string{"providers-disabled", "native-enforcement-not-certified", "strict-token-egress-accounting-unverified", "no-live-operation-authority"}}
	for k := range c["agents"].(map[string]any) {
		report.Profiles = append(report.Profiles, k)
	}
	sort.Strings(report.Profiles)
	for k := range c["commands"].(map[string]any) {
		report.Commands = append(report.Commands, k)
	}
	sort.Strings(report.Commands)
	if len(native) > 0 {
		if _, err := parseConfig(native); err != nil {
			return Report{}, err
		}
		var n nativeObservation
		d := json.NewDecoder(strings.NewReader(string(native)))
		d.DisallowUnknownFields()
		if err := d.Decode(&n); err != nil {
			return Report{}, fmt.Errorf("invalid native metadata projection")
		}
		for _, list := range [][]string{n.Profiles, n.Commands, n.Skills} {
			for _, id := range list {
				if !metadataID.MatchString(id) {
					return Report{}, fmt.Errorf("invalid metadata identifier")
				}
			}
		}
		for _, warning := range n.Warnings {
			if warning != "invalid-policy" && warning != "ignored-setting" && warning != "unknown-source" {
				return Report{}, fmt.Errorf("unrecognized native warning projection")
			}
			report.Blockers = append(report.Blockers, warning)
		}
		if n.Version != Runtime {
			report.Blockers = append(report.Blockers, "runtime-mismatch")
		}
		if n.Truncated {
			report.Blockers = append(report.Blockers, "inventory-truncated")
		}
		report.Observation = "caller-supplied-not-verified"
	}
	return report, nil
}
