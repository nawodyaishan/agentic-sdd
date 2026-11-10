package tui

import (
	"fmt"
	"strings"
	"testing"

	"agentic-sdd/internal/appops"
	"agentic-sdd/internal/skillsync"
	"github.com/charmbracelet/x/ansi"
)

// These tests cover the B2 owner review fix: an Explore agent profile must
// show its provider, primary model, documented fallbacks and the routing
// policy note, while a skill or concept shows no invented routing state. They
// use the curated metadata in internal/appops; nothing here reads the network,
// a provider or a real home directory.

// routingProfileItem builds one unenriched profile row.
func routingProfileItem(id, origin string) CatalogItem {
	return CatalogItem{ID: id, Kind: skillsync.InventoryKindProfile, Origin: origin, Group: GroupOpenCodeProfiles}
}

// TestRoutingProfileItemsCarryCuratedRouting proves profile rows are enriched
// with the shared routing metadata, and that skills and concepts stay empty.
func TestRoutingProfileItemsCarryCuratedRouting(t *testing.T) {
	profile := routingProfileItem("sdd-debug-diagnose", skillsync.InventoryOriginPack)
	enrichItem(&profile)

	if profile.AgentMode != "subagent" {
		t.Errorf("profile mode = %q, want subagent", profile.AgentMode)
	}
	if profile.AgentProvider != "OpenCode Console (free)" {
		t.Errorf("profile provider = %q, want the documented free Console lane", profile.AgentProvider)
	}
	if profile.AgentModel != "opencode/deepseek-v4.1-flash-free" {
		t.Errorf("profile model = %q, want the documented pinned model", profile.AgentModel)
	}
	wantFallbacks := []string{"opencode-go/deepseek-v4.1-flash", "9router/ag/gemini-3.8-flash-high"}
	if strings.Join(profile.AgentFallbacks, ",") != strings.Join(wantFallbacks, ",") {
		t.Errorf("profile fallbacks = %v, want %v", profile.AgentFallbacks, wantFallbacks)
	}
	if !strings.Contains(profile.RoutingNote, "policy, not automatic failover") {
		t.Errorf("profile routing note = %q, want the written-ladder policy statement", profile.RoutingNote)
	}

	skill := CatalogItem{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore}
	enrichItem(&skill)
	concept := CatalogItem{ID: "preview-apply", Kind: appops.TeachingKindConcept, Origin: skillsync.InventoryOriginCore}
	enrichItem(&concept)
	for _, item := range []CatalogItem{skill, concept} {
		if item.AgentMode != "" || item.AgentProvider != "" || item.AgentModel != "" ||
			len(item.AgentFallbacks) != 0 || item.RoutingNote != "" {
			t.Errorf("%s item %q carries routing metadata it must not have: %+v", item.Kind, item.ID, item)
		}
	}
}

// TestRoutingSummaryBadgeIsCompactAndBounded proves a profile row shows a
// compact provider badge while an uncurated long model string stays inside the
// list bound instead of overflowing it.
func TestRoutingSummaryBadgeIsCompactAndBounded(t *testing.T) {
	profile := routingProfileItem("sdd-debug-diagnose", skillsync.InventoryOriginPack)
	enrichItem(&profile)
	if summary := profile.Summary(); !strings.Contains(summary, "OpenCode Console (free)") {
		t.Errorf("profile summary %q does not show the provider badge", summary)
	}

	skill := CatalogItem{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore}
	enrichItem(&skill)
	wantSkillSummary := fmt.Sprintf("%s · %s · %d/%d clients installed",
		skill.Kind, skill.Origin, skill.InstalledClients(), len(skill.Clients))
	if got := skill.Summary(); got != wantSkillSummary {
		t.Errorf("skill summary = %q, want the unchanged %q", got, wantSkillSummary)
	}

	long := CatalogItem{
		ID:         "local-unlisted-profile",
		Kind:       skillsync.InventoryKindProfile,
		Origin:     skillsync.InventoryOriginPack,
		AgentModel: strings.Repeat("long-model-name-", 8),
	}
	enrichItem(&long)
	parts := strings.Split(long.Summary(), " · ")
	badge := parts[len(parts)-1]
	if width := ansi.StringWidth(badge); width > maxSummaryBadgeWidth {
		t.Errorf("long routing badge %q is %d cells wide, want at most %d", badge, width, maxSummaryBadgeWidth)
	}
}

// TestRoutingFilterValueIncludesRoutingTerms proves search reaches the provider,
// model and documented fallbacks of a profile.
func TestRoutingFilterValueIncludesRoutingTerms(t *testing.T) {
	profile := routingProfileItem("sdd-review-verification-1", skillsync.InventoryOriginPack)
	enrichItem(&profile)

	filter := profile.FilterValue()
	for _, want := range []string{"opencode-go/glm-5.3-flash", "opencode-go/deepseek-v4.1-flash", "opencode-go/minimax-m3.1-flash", "opencode go"} {
		if !strings.Contains(filter, want) {
			t.Errorf("filter value %q is missing the routing term %q", filter, want)
		}
	}
}

// TestRoutingDetailCardSectionOrderAndContent proves the profile detail card
// shows the routing section after the identity block and before the destination
// status, with labelled values, numbered fallbacks and the policy note.
func TestRoutingDetailCardSectionOrderAndContent(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		routingProfileItem("sdd-architecture-review-9r", skillsync.InventoryOriginPack),
	}}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
	card := m.DetailCard(100)

	routingIndex := strings.Index(card, "Agent routing:")
	statusIndex := strings.Index(card, "Destination status:")
	if routingIndex < 0 {
		t.Fatalf("profile detail card has no Agent routing section:\n%s", card)
	}
	if statusIndex < 0 || routingIndex > statusIndex {
		t.Errorf("Agent routing must come before Destination status:\n%s", card)
	}

	for _, want := range []string{
		"Mode: subagent",
		"Provider: 9Router",
		"Primary model: 9router/ag/claude-opus-5-5-high",
		"1. 9router/ag/gemini-3.8-flash-high",
		"2. opencode-go/deepseek-v4.1-flash",
		"provisional",
		"policy, not automatic failover",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("profile detail card is missing %q:\n%s", want, card)
		}
	}
	for _, claim := range []string{"is running", "authenticated", "always available", "guaranteed"} {
		if strings.Contains(strings.ToLower(card), claim) {
			t.Errorf("profile detail card claims runtime or availability state (%q):\n%s", claim, card)
		}
	}
}

// TestRoutingOrchestratorShowsActiveSessionModel proves the orchestrator card
// presents the active session model and the human-selected fallback note
// instead of an invented pinned model.
func TestRoutingOrchestratorShowsActiveSessionModel(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		routingProfileItem("sdd-orchestrator", skillsync.InventoryOriginPack),
	}}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
	card := m.DetailCard(100)

	if !strings.Contains(card, "Primary model: Active session model") {
		t.Errorf("orchestrator card does not show the active session model:\n%s", card)
	}
	if !strings.Contains(card, "human-selected session model only") {
		t.Errorf("orchestrator card does not show the documented fallback note:\n%s", card)
	}
}

// TestRoutingSkillAndConceptCardsOmitAgentSection proves a non-profile card
// does not gain an agent routing section it has no policy for.
func TestRoutingSkillAndConceptCardsOmitAgentSection(t *testing.T) {
	for _, item := range []CatalogItem{
		{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
		{ID: "preview-apply", Kind: appops.TeachingKindConcept, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
	} {
		cat := Catalog{Items: []CatalogItem{item}}
		m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
		card := m.DetailCard(100)
		if strings.Contains(card, "Agent routing:") {
			t.Errorf("%s card shows an agent routing section:\n%s", item.Kind, card)
		}
	}
}

// TestRoutingUncuratedProfileIsHonest proves an unlisted profile does not gain
// a guessed provider or invented fallbacks.
func TestRoutingUncuratedProfileIsHonest(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{{
		ID:         "local-unlisted-profile",
		Kind:       skillsync.InventoryKindProfile,
		Origin:     skillsync.InventoryOriginPack,
		Group:      GroupOpenCodeProfiles,
		AgentModel: "vendor/unknown-model",
	}}}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
	card := m.DetailCard(100)

	if strings.Contains(card, "Provider: vendor") || strings.Contains(card, "Provider: Vendor") {
		t.Errorf("unlisted profile invented a provider from an arbitrary string:\n%s", card)
	}
	if !strings.Contains(card, "Provider: not documented by Agentic SDD") {
		t.Errorf("unlisted profile does not state that its provider is undocumented:\n%s", card)
	}
	if !strings.Contains(card, "Fallbacks: none documented for this profile") {
		t.Errorf("unlisted profile does not state that no fallbacks are documented:\n%s", card)
	}
}

// TestRoutingDetailCardWrapsAtNarrowWidth proves the routing section wraps
// instead of overflowing a narrow detail pane.
func TestRoutingDetailCardWrapsAtNarrowWidth(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		routingProfileItem("sdd-architecture-advisor", skillsync.InventoryOriginPack),
	}}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
	card := m.DetailCard(40)
	for i, line := range strings.Split(card, "\n") {
		if width := ansi.StringWidth(line); width > 40 {
			t.Errorf("line %d is %d cells wide at width 40: %q", i, width, line)
		}
	}
}

// TestRoutingDisplayItemOverrideKeepsRouting proves the parent's displayItem
// override still renders the curated routing metadata.
func TestRoutingDisplayItemOverrideKeepsRouting(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		routingProfileItem("sdd-debug-diagnose", skillsync.InventoryOriginPack),
	}}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 24}, cat)
	selected, ok := m.Selected()
	if !ok {
		t.Fatal("no profile selected")
	}
	displayCopy := selected
	displayCopy.ID = "\\u0073dd-debug-diagnose"
	m.displayItem = &displayCopy

	card := m.DetailCard(100)
	if !strings.Contains(card, "Agent routing:") {
		t.Errorf("displayItem override dropped the routing section:\n%s", card)
	}
	if !strings.Contains(card, "Primary model: opencode/deepseek-v4.1-flash-free") {
		t.Errorf("displayItem override dropped the curated primary model:\n%s", card)
	}
}

// TestRoutingPresetDetailShowsLadder proves a Free-first preset card shows its
// frontmatter model and the ordered ladder, including the authorization
// boundary before the paid rungs.
func TestRoutingPresetDetailShowsLadder(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{{
		ID:     "sdd-implement-1",
		Kind:   skillsync.InventoryKindProfile,
		Origin: skillsync.InventoryOriginPreset,
		Group:  GroupFreeFirstPresets,
	}}}
	m := NewCatalogModel(TUIConfig{Width: 120, Height: 24}, cat)
	card := m.DetailCard(120)

	for _, want := range []string{
		"Primary model: opencode/deepseek-v4.1-flash-free",
		"1. opencode/glm-5.3-flash-free",
		"2. opencode/mimo-v2.6-flash-free",
		"4. opencode-go/glm-5.3-flash",
		"6. opencode-go/mimo-v2.6-flash",
		"authorization",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("preset detail card is missing %q:\n%s", want, card)
		}
	}
}
