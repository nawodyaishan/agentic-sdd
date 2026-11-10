package appops

import (
	"reflect"
	"strings"
	"testing"
)

// These tests exercise the curated teaching metadata through its public API
// only. They never read the repository's shipped asset directories: the CLI's
// existing catalog fixture remains the authority that locks membership against
// the assets, while this package owns the single metadata source.

func TestTeachingForKnownSkillCard(t *testing.T) {
	card := TeachingFor("agentic-sdd-router", TeachingKindSkill)
	if card.ID != "agentic-sdd-router" || card.Kind != TeachingKindSkill {
		t.Fatalf("known skill identity = %q/%q, want agentic-sdd-router/skill", card.ID, card.Kind)
	}
	if card.DisplayName != "SDD Router" {
		t.Errorf("display name = %q, want %q", card.DisplayName, "SDD Router")
	}
	if card.Group != "Scope/plan" {
		t.Errorf("group = %q, want %q", card.Group, "Scope/plan")
	}
	if card.Caveat != TeachingSkillCaveat {
		t.Errorf("skill caveat = %q, want %q", card.Caveat, TeachingSkillCaveat)
	}
	if strings.TrimSpace(card.Purpose) == "" || strings.TrimSpace(card.When) == "" || strings.TrimSpace(card.Example) == "" {
		t.Errorf("known skill card is incomplete: %+v", card)
	}
}

func TestTeachingForKnownPresetProfileCard(t *testing.T) {
	card := TeachingFor("sdd-implement-1", TeachingKindProfile)
	if card.ID != "sdd-implement-1" || card.Kind != TeachingKindProfile {
		t.Fatalf("known profile identity = %q/%q, want sdd-implement-1/profile", card.ID, card.Kind)
	}
	if card.DisplayName != "Implementer 1" {
		t.Errorf("display name = %q, want %q", card.DisplayName, "Implementer 1")
	}
	if card.Group != "Implementation/documentation" {
		t.Errorf("group = %q, want %q", card.Group, "Implementation/documentation")
	}
	if card.Caveat != TeachingProfileCaveat {
		t.Errorf("profile caveat = %q, want %q", card.Caveat, TeachingProfileCaveat)
	}
}

func TestTeachingForUnknownFallsBackNeutrally(t *testing.T) {
	wantPurpose := map[string]string{
		TeachingKindSkill:   "Custom skill — purpose not documented by Agentic SDD.",
		TeachingKindProfile: "Custom profile — purpose not documented by Agentic SDD.",
		TeachingKindConcept: "Custom concept — not documented by Agentic SDD.",
	}
	wantCaveat := map[string]string{
		TeachingKindSkill:   TeachingSkillCaveat,
		TeachingKindProfile: TeachingProfileCaveat,
		TeachingKindConcept: TeachingConceptCaveat,
	}
	for _, kind := range []string{TeachingKindSkill, TeachingKindProfile, TeachingKindConcept} {
		card := TeachingFor("local-unlisted-item", kind)
		if card.ID != "local-unlisted-item" || card.Kind != kind {
			t.Errorf("fallback for %s must preserve identity, got %+v", kind, card)
		}
		if card.DisplayName != "local-unlisted-item" {
			t.Errorf("unknown %s display name = %q, want the exact ID as the neutral label", kind, card.DisplayName)
		}
		if card.Group != TeachingGroupCustom {
			t.Errorf("fallback for %s group = %q, want %q", kind, card.Group, TeachingGroupCustom)
		}
		if card.Purpose != wantPurpose[kind] {
			t.Errorf("fallback for %s purpose = %q, want %q", kind, card.Purpose, wantPurpose[kind])
		}
		if card.Caveat != wantCaveat[kind] {
			t.Errorf("fallback for %s caveat = %q, want %q", kind, card.Caveat, wantCaveat[kind])
		}
		if card.When == "" || card.Example == "" {
			t.Errorf("fallback for %s must stay a complete card: %+v", kind, card)
		}
	}

	// A known ID looked up under another kind must not borrow that role.
	borrowed := TeachingFor("agentic-sdd-spec", TeachingKindProfile)
	if borrowed.DisplayName != "agentic-sdd-spec" || borrowed.Group != TeachingGroupCustom {
		t.Errorf("known skill ID read as a profile must fall back neutrally, got %+v", borrowed)
	}
}

func TestTeachingCustomCardAlwaysFallsBack(t *testing.T) {
	// The CLI's custom-card helper deliberately ignores the curated registry so
	// an unlisted local item cannot inherit a curated role by colliding with a
	// known ID.
	for _, tc := range []struct{ id, kind string }{
		{"agentic-sdd-router", TeachingKindSkill},
		{"sdd-orchestrator", TeachingKindProfile},
		{"preview-apply", TeachingKindConcept},
		{"local-unlisted-item", TeachingKindSkill},
	} {
		card := TeachingCustomCard(tc.id, tc.kind)
		if card.ID != tc.id || card.Kind != tc.kind {
			t.Errorf("custom card identity = %q/%q, want %q/%q", card.ID, card.Kind, tc.id, tc.kind)
		}
		if card.Group != TeachingGroupCustom {
			t.Errorf("custom card %q group = %q, want %q", tc.id, card.Group, TeachingGroupCustom)
		}
		if !strings.Contains(card.Purpose, "not documented") {
			t.Errorf("custom card %q purpose = %q, want the generic undocumented text", tc.id, card.Purpose)
		}
		if card.DisplayName != tc.id {
			t.Errorf("custom card %q display name = %q, want the exact ID", tc.id, card.DisplayName)
		}
	}
}

func TestTeachingCardsIncludesConceptsAndIsDefensive(t *testing.T) {
	cards := TeachingCards()
	if len(cards) == 0 {
		t.Fatal("TeachingCards returned no cards")
	}

	kinds := map[string]int{}
	seen := map[string]bool{}
	for _, card := range cards {
		kinds[card.Kind]++
		key := card.Kind + "\x00" + card.ID
		if seen[key] {
			t.Errorf("duplicate curated card %q of kind %q", card.ID, card.Kind)
		}
		seen[key] = true

		if card.ID == "" || card.Kind == "" || card.DisplayName == "" {
			t.Errorf("card has empty ID, kind or display name: %+v", card)
		}
		if card.DisplayName != strings.TrimSpace(card.DisplayName) || strings.ContainsAny(card.DisplayName, "\r\n\t`") {
			t.Errorf("card %q display name is not a single clean label: %q", card.ID, card.DisplayName)
		}
		if runes := len([]rune(card.DisplayName)); runes < 3 || runes > 40 {
			t.Errorf("card %q display name length %d is outside 3-40 runes: %q", card.ID, runes, card.DisplayName)
		}
		if card.DisplayName == card.ID {
			t.Errorf("known card %q must have a curated readable label, not its raw ID", card.ID)
		}
		if lookup := TeachingFor(card.ID, card.Kind); lookup != card {
			t.Errorf("TeachingFor(%q, %q) disagrees with TeachingCards entry", card.ID, card.Kind)
		}
	}

	for _, kind := range []string{TeachingKindSkill, TeachingKindProfile, TeachingKindConcept} {
		if kinds[kind] == 0 {
			t.Errorf("TeachingCards has no %q cards", kind)
		}
	}
	if got, want := kinds[TeachingKindConcept], len(TeachingConcepts()); got != want {
		t.Errorf("concept count in TeachingCards = %d, TeachingConcepts = %d", got, want)
	}

	// Callers receive their own slice: mutating it cannot change the source.
	original := cards[0]
	cards[0].Purpose = "mutated"
	cards[0].DisplayName = "Mutated"
	fresh := TeachingCards()
	if fresh[0] != original {
		t.Errorf("TeachingCards leaked mutable registry state: %+v", fresh[0])
	}
	if again := TeachingCards(); !reflect.DeepEqual(again, fresh) {
		t.Error("TeachingCards must return a stable slice across calls")
	}
}

func TestTeachingConceptsStableOrderAndDefensive(t *testing.T) {
	want := []string{
		"spec-plan-tasks",
		"skill-profile",
		"main-subagent",
		"preview-apply",
		"free-first",
		"human-approval",
	}
	got := TeachingConcepts()
	if len(got) != len(want) {
		t.Fatalf("TeachingConcepts returned %d cards, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("concept %d = %q, want %q", i, got[i].ID, id)
		}
		if got[i].Kind != TeachingKindConcept || got[i].Group != TeachingGroupConcept {
			t.Errorf("concept %q kind/group = %q/%q, want concept/%q", id, got[i].Kind, got[i].Group, TeachingGroupConcept)
		}
		if got[i].DisplayName == "" || got[i].DisplayName == got[i].ID {
			t.Errorf("concept %q needs a curated readable display name, got %q", id, got[i].DisplayName)
		}
		if lookup := TeachingFor(id, TeachingKindConcept); lookup != got[i] {
			t.Errorf("TeachingFor(%q, concept) disagrees with TeachingConcepts", id)
		}
	}

	got[0].Purpose = "mutated"
	if again := TeachingConcepts(); again[0].Purpose == "mutated" {
		t.Error("TeachingConcepts must return an independent slice")
	}
}

func TestTeachingKindAndGroupConstants(t *testing.T) {
	if TeachingKindSkill != "skill" || TeachingKindProfile != "profile" || TeachingKindConcept != "concept" {
		t.Errorf("kind constants changed: %q/%q/%q", TeachingKindSkill, TeachingKindProfile, TeachingKindConcept)
	}
	if TeachingGroupCustom != "Custom" {
		t.Errorf("custom group = %q, want %q", TeachingGroupCustom, "Custom")
	}
	if TeachingGroupConcept != "Key concepts" {
		t.Errorf("concept group = %q, want %q", TeachingGroupConcept, "Key concepts")
	}
	for label, caveat := range map[string]string{
		"skill":   TeachingSkillCaveat,
		"profile": TeachingProfileCaveat,
		"concept": TeachingConceptCaveat,
	} {
		if !strings.Contains(caveat, "No automated agent execution") || !strings.Contains(caveat, "Human approval required") {
			t.Errorf("%s caveat lost its safety meaning: %q", label, caveat)
		}
	}
}
