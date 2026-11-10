package appops

import "strings"

// AgentRouting is curated, read-only display metadata about an agent profile's
// declared model policy. It is configuration guidance for a template, not
// proof that a provider is configured, a credential exists, quota is
// available, a model is reachable, or the agent is running.
//
// Fallbacks are documented manual alternatives in preference order. A written
// ladder is policy, not automatic failover: the parent agent must decide
// whether to retry, check compatibility first, obtain any required spending
// authorization, and report the model that actually ran.
type AgentRouting struct {
	Mode      string
	Provider  string
	Model     string
	Fallbacks []string
	Note      string
}

// routingNote is appended to every curated routing entry. It keeps the display
// honest about what a documented ladder does and does not guarantee.
const routingNote = "A written ladder is policy, not automatic failover: the parent decides whether to retry, checks compatibility first, and reports the model that actually ran. Provider access, credentials, quota, pricing, availability and runtime use are not claimed."

const (
	providerFreeConsole = "OpenCode Console (free)"
	providerGo          = "OpenCode Go"
	provider9Router     = "9Router"
	providerOpenAI      = "OpenAI"
	providerSession     = "Active session"
	providerUnknown     = "not documented by Agentic SDD"
)

// freeFirstFallbacks is the documented alternative order for the Free-first
// presets: two free rungs, then the paid Go ladder that needs separate human
// authorization and may incur charges.
var freeFirstFallbacks = []string{
	"opencode/glm-5.3-flash-free",
	"opencode/mimo-v2.6-flash-free",
	"opencode-go/deepseek-v4.1-flash",
	"opencode-go/glm-5.3-flash",
	"opencode-go/minimax-m3.1-flash",
	"opencode-go/mimo-v2.6-flash",
}

// agentRouting is the single curated routing table for shipped profiles. Tests
// compare it with opencode-pack/routing.md so a routing change forces a review.
var agentRouting = map[string]AgentRouting{
	"sdd-orchestrator": {
		Mode:     "primary",
		Provider: providerSession,
		Model:    "Active session model",
		Note:     "No pinned model: this profile uses the active session model. A fallback is a human-selected session model only. " + routingNote,
	},
	"sdd-debug-diagnose": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/deepseek-v4.1-flash-free",
		Fallbacks: []string{
			"opencode-go/deepseek-v4.1-flash",
			"9router/ag/gemini-3.8-flash-high",
		},
		Note: routingNote,
	},
	"sdd-test-design": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/deepseek-v4.1-flash-free",
		Fallbacks: []string{
			"opencode-go/deepseek-v4.1-flash",
			"9router/ag/gemini-3.8-flash-high",
		},
		Note: routingNote,
	},
	"sdd-research-librarian": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/deepseek-v4.1-flash-free",
		Fallbacks: []string{
			"opencode-go/deepseek-v4.1-flash",
			"9router/ag/gemini-3.8-flash-high",
		},
		Note: routingNote,
	},
	"sdd-docs-worker": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/deepseek-v4.1-flash-free",
		Fallbacks: []string{
			"opencode-go/deepseek-v4.1-flash",
			"9router/ag/gemini-3.8-flash-high",
		},
		Note: routingNote,
	},
	"sdd-architecture-advisor": {
		Mode:     "subagent",
		Provider: provider9Router,
		Model:    "9router/ag/claude-sonnet-5-5-high",
		Fallbacks: []string{
			"9router/ag/gemini-4-argon-high",
			"opencode-go/deepseek-v4.1-flash#max",
		},
		Note: routingNote,
	},
	"sdd-visual-inspector": {
		Mode:      "subagent",
		Provider:  providerFreeConsole,
		Model:     "opencode/glm-5.3-flash-free",
		Fallbacks: []string{"opencode-go/glm-5.3-flash"},
		Note:      routingNote,
	},
	"sdd-openai-architect": {
		Mode:     "subagent",
		Provider: providerOpenAI,
		Model:    "openai/gpt-6.1-sol",
		Fallbacks: []string{
			"openai/gpt-6.1-sol-fast",
			"openrouter/openai/gpt-6.1-sol",
		},
		Note: "Optional provider-specific specialist: stop blocked when access is missing or spending is unauthorized; never silently fall back across providers. " + routingNote,
	},
	"sdd-openai-implement": {
		Mode:     "subagent",
		Provider: providerOpenAI,
		Model:    "openai/gpt-6.1-terra",
		Fallbacks: []string{
			"openai/gpt-6.1-terra-fast",
			"openrouter/openai/gpt-6.1-terra",
		},
		Note: "Optional provider-specific specialist: stop blocked when access is missing or spending is unauthorized; never silently fall back across providers. " + routingNote,
	},
	"sdd-architecture-review-9r": {
		Mode:     "subagent",
		Provider: provider9Router,
		Model:    "9router/ag/claude-opus-5-5-high",
		Fallbacks: []string{
			"9router/ag/gemini-3.8-flash-high",
			"opencode-go/deepseek-v4.1-flash",
		},
		Note: "The Claude-agent exception applies. A fallback architecture review is provisional and needs an Opus review before it is the final consequential technical verdict. " + routingNote,
	},
	"sdd-code-review-9r": {
		Mode:     "subagent",
		Provider: provider9Router,
		Model:    "9router/ag/claude-opus-5-5-high",
		Fallbacks: []string{
			"9router/ag/gemini-3.8-flash-high",
			"opencode-go/deepseek-v4.1-flash",
		},
		Note: "The Claude-agent exception applies. A fallback code review is provisional and needs an Opus review before it is the final consequential technical verdict. " + routingNote,
	},
	"sdd-review-verification-1": {
		Mode:     "subagent",
		Provider: providerGo,
		Model:    "opencode-go/glm-5.3-flash",
		Fallbacks: []string{
			"opencode-go/deepseek-v4.1-flash",
			"opencode-go/minimax-m3.1-flash",
		},
		Note: routingNote,
	},
	"sdd-review-verification-2": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/beam-free",
		Fallbacks: []string{
			"opencode-go/glm-5.3-flash (explicit authorization required)",
			"opencode-go/deepseek-v4.1-flash (explicit authorization required)",
		},
		Note: "Beam cross-lane rule: a Beam failure does not authorize a Go model; the parent must obtain explicit human authorization for that retry, which may incur charges. This assessment remains supplementary, not the sole consequential reviewer. " + routingNote,
	},
	"sdd-lightweight": {
		Mode:     "subagent",
		Provider: providerFreeConsole,
		Model:    "opencode/beam-free",
		Fallbacks: []string{
			"opencode-go/mimo-v2.6-flash (explicit authorization required)",
			"opencode-go/qwen3.8-flash (explicit authorization required)",
		},
		Note: "Beam cross-lane rule: a Beam failure does not authorize a Go model; the parent must obtain explicit human authorization for that retry, which may incur charges. This remains a bounded read-only discovery role even on the fallback model. " + routingNote,
	},
}

// presetRouting is the shared pinned model and Free-first ladder for the four
// shipped preset profiles. Their frontmatter model is checked in tests.
func presetRouting() AgentRouting {
	return AgentRouting{
		Mode:      "subagent",
		Provider:  providerFreeConsole,
		Model:     "opencode/deepseek-v4.1-flash-free",
		Fallbacks: append([]string(nil), freeFirstFallbacks...),
		Note:      "Free-first policy: try each free rung once in order. The paid OpenCode Go ladder requires separate human authorization and may incur charges. " + routingNote,
	}
}

func init() {
	for _, id := range []string{
		"sdd-implement-1",
		"sdd-implement-2",
		"sdd-test-implement-1",
		"sdd-test-implement-2",
	} {
		agentRouting[id] = presetRouting()
	}
}

// AgentRoutingFor returns the curated routing metadata for a shipped agent
// profile. The boolean is false for skills, concepts and unlisted local items;
// callers must then avoid inventing a provider, model or fallback.
func AgentRoutingFor(id string) (AgentRouting, bool) {
	routing, ok := agentRouting[id]
	if !ok {
		return AgentRouting{}, false
	}
	routing.Fallbacks = append([]string(nil), routing.Fallbacks...)
	return routing, true
}

// AgentRoutingIDs returns the curated profile IDs in stable sorted order. It is
// used by drift tests and by callers that need the complete policy surface.
func AgentRoutingIDs() []string {
	ids := make([]string, 0, len(agentRouting))
	for id := range agentRouting {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[j] < ids[i] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

// routingProviderForModel derives the display lane only for a curated model
// string. It is used by drift tests, not to guess providers for unlisted items.
func routingProviderForModel(model string) string {
	switch {
	case strings.HasPrefix(model, "opencode-go/"):
		return providerGo
	case strings.HasPrefix(model, "opencode/"):
		return providerFreeConsole
	case strings.HasPrefix(model, "9router/"):
		return provider9Router
	case strings.HasPrefix(model, "openai/"):
		return providerOpenAI
	case strings.HasPrefix(model, "openrouter/"):
		return "OpenRouter"
	default:
		return providerUnknown
	}
}
