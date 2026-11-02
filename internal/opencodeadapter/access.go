package opencodeadapter

import (
	"fmt"
	"regexp"
	"strings"
)

var featureGrant = regexp.MustCompile(`^specs/[a-z0-9][a-z0-9-]{0,63}/(spec|plan|tasks)\.md$`)

func defaultDeny(raw any) bool {
	p, ok := raw.(map[string]any)
	return ok && len(p) == 3 && p["action"] == "*" && p["resource"] == "*" && p["effect"] == "deny"
}

func validateAccess(c map[string]any) error {
	root, ok := c["permissions"].([]any)
	if !ok || len(root) != 1 || !defaultDeny(root[0]) {
		return fmt.Errorf("candidate default permissions must deny effects")
	}
	if compaction, ok := c["compaction"].(map[string]any); ok {
		if auto, ok := compaction["auto"].(bool); ok && auto {
			return fmt.Errorf("candidate warming spec prohibits automatic compaction")
		}
	}
	if experimental, ok := c["experimental"].(map[string]any); ok {
		if background, ok := experimental["background_compaction"].(bool); ok && background {
			return fmt.Errorf("candidate warming spec prohibits speculative background compaction")
		}
	}
	agents := c["agents"].(map[string]any)
	if len(agents) != 3 {
		return fmt.Errorf("candidate must use the three reviewed primary profiles")
	}
	for _, name := range []string{"sdd-draft", "sdd-build", "sdd-review"} {
		raw, ok := agents[name]
		if !ok {
			return fmt.Errorf("reviewed primary profile missing")
		}
		a := raw.(map[string]any)
		rules, ok := a["permissions"].([]any)
		if !ok || len(rules) == 0 || !defaultDeny(rules[0]) {
			return fmt.Errorf("profile default denial missing")
		}
		denied := map[string]bool{}
		protectedRead := false
		for _, raw := range rules[1:] {
			rule, ok := raw.(map[string]any)
			if !ok || len(rule) != 3 {
				return fmt.Errorf("invalid permission shape")
			}
			action, aok := rule["action"].(string)
			resource, rok := rule["resource"].(string)
			effect, eok := rule["effect"].(string)
			if !aok || !rok || !eok || resource == "" || strings.Contains(resource, "../") || strings.Contains(resource, "\\") || strings.HasPrefix(resource, "/") {
				return fmt.Errorf("invalid permission resource")
			}
			if effect == "deny" {
				if action == "read" {
					denied[resource] = true
					protectedRead = true
				}
				continue
			}
			if effect != "allow" && effect != "ask" {
				return fmt.Errorf("unsupported permission effect")
			}
			switch action {
			case "read":
				if effect != "allow" || protectedRead {
					return fmt.Errorf("read grant may not override protected denials")
				}
			case "skill":
				if effect != "allow" || resource != "agentic-sdd-*" {
					return fmt.Errorf("unreviewed skill permission")
				}
			case "question":
				if effect != "allow" || resource != "*" {
					return fmt.Errorf("unsupported question permission")
				}
			case "edit":
				feature := featureGrant.MatchString(resource)
				build := strings.HasSuffix(resource, "/*") && literalBuildPath(strings.TrimSuffix(resource, "/*"))
				if effect != "allow" || name == "sdd-review" || (name == "sdd-draft" && !feature) || (!feature && !build) {
					return fmt.Errorf("unreviewed write permission")
				}
			case "shell":
				if name != "sdd-build" || effect != "ask" || resource != "*" {
					return fmt.Errorf("unreviewed shell permission")
				}
			default:
				return fmt.Errorf("unreviewed tool permission")
			}
		}
		for _, p := range []string{"*.env*", "*.db*", "*.pem", ".git/*", "secrets/*", "*/secrets/*", ".opencode/*"} {
			if !denied[p] {
				return fmt.Errorf("protected read denial missing")
			}
		}
	}
	return nil
}
