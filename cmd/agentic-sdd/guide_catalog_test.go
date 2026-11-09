package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// listShippedIDs reads a shipped source directory so the curated teaching
// registry cannot drift from the assets it describes. Directories are read
// only; tests never write to the repository or a user home.
func listShippedIDs(t *testing.T, dir, suffix string, dirsOnly bool) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read shipped asset directory %s: %v", dir, err)
	}
	var ids []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if !dirsOnly {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, name, "SKILL.md")); err != nil {
				continue
			}
			ids = append(ids, name)
			continue
		}
		if dirsOnly || !strings.HasSuffix(name, suffix) {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, suffix))
	}
	return ids
}

func registryIDs(kind string) []string {
	var ids []string
	for _, card := range teachingRegistry {
		if card.Kind == kind {
			ids = append(ids, card.ID)
		}
	}
	return ids
}

func assertSameIDSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	gotSet := make(map[string]bool, len(got))
	wantSet := make(map[string]bool, len(want))
	for _, id := range got {
		gotSet[id] = true
	}
	for _, id := range want {
		wantSet[id] = true
	}
	var missing, extra []string
	for id := range wantSet {
		if !gotSet[id] {
			missing = append(missing, id)
		}
	}
	for id := range gotSet {
		if !wantSet[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("%s registry is missing shipped IDs: %v", label, missing)
	}
	if len(extra) > 0 {
		t.Errorf("%s registry has IDs not shipped in this tree: %v", label, extra)
	}
}

func TestTeachingRegistryMatchesShippedAssets(t *testing.T) {
	coreSkills := listShippedIDs(t, filepath.Join("..", "..", "skills-files"), "", true)
	packSkills := listShippedIDs(t, filepath.Join("..", "..", "opencode-pack", "skills"), "", true)
	baseProfiles := listShippedIDs(t, filepath.Join("..", "..", "opencode-pack", "agents"), ".md", false)
	presetProfiles := listShippedIDs(t, filepath.Join("..", "..", "opencode-pack", "presets", "free-first"), ".md", false)

	if len(coreSkills) == 0 || len(packSkills) == 0 || len(baseProfiles) == 0 || len(presetProfiles) == 0 {
		t.Fatalf("shipped asset directories are unexpectedly empty: core=%d pack=%d base=%d preset=%d",
			len(coreSkills), len(packSkills), len(baseProfiles), len(presetProfiles))
	}

	assertSameIDSet(t, "skill", registryIDs(teachingKindSkill), append(append([]string{}, coreSkills...), packSkills...))
	assertSameIDSet(t, "profile", registryIDs(teachingKindProfile), append(append([]string{}, baseProfiles...), presetProfiles...))

	// The explorer shows "(optional preset)" as one badge, so Purpose must not
	// repeat that label (FR-03, UX-02). Presets still need a real purpose.
	for _, id := range presetProfiles {
		card := teachingFor(id, teachingKindProfile)
		if strings.TrimSpace(card.Purpose) == "" {
			t.Errorf("preset profile %q needs a documented purpose", id)
		}
		if strings.Contains(card.Purpose, "Optional preset") {
			t.Errorf("preset profile %q purpose %q must not repeat the optional-preset badge", id, card.Purpose)
		}
	}
	for _, id := range baseProfiles {
		card := teachingFor(id, teachingKindProfile)
		if strings.Contains(card.Purpose, "Optional preset") {
			t.Errorf("base profile %q must not be labelled an optional preset", id)
		}
	}
}

func TestTeachingKnownCardsComplete(t *testing.T) {
	skillGroups := map[string]bool{
		"Scope/plan":      true,
		"Implement/debug": true,
		"Verify/review":   true,
		"Learn/reflect":   true,
	}
	profileGroups := map[string]bool{
		"Main communicator":                true,
		"Read-only investigation":          true,
		"Implementation/documentation":     true,
		"Test assistance":                  true,
		"Independent/supplementary review": true,
	}

	seen := make(map[string]string, len(teachingRegistry))
	for _, card := range teachingRegistry {
		if card.ID == "" || card.Kind == "" || card.Group == "" {
			t.Errorf("card has empty ID, kind or group: %+v", card)
			continue
		}
		if prev, dup := seen[card.ID]; dup {
			t.Errorf("duplicate registry ID %q (kinds %q and %q)", card.ID, prev, card.Kind)
		}
		seen[card.ID] = card.Kind

		for name, value := range map[string]string{
			"purpose": card.Purpose,
			"when":    card.When,
			"example": card.Example,
			"caveat":  card.Caveat,
		} {
			if strings.TrimSpace(value) == "" {
				t.Errorf("card %q/%s has empty %s", card.ID, card.Kind, name)
			}
		}

		switch card.Kind {
		case teachingKindSkill:
			if !skillGroups[card.Group] {
				t.Errorf("skill %q uses unknown group %q", card.ID, card.Group)
			}
		case teachingKindProfile:
			if !profileGroups[card.Group] {
				t.Errorf("profile %q uses unknown group %q", card.ID, card.Group)
			}
			// A learner must always receive purpose, when, example and caveat
			// for every profile offered by the selected source.
			if card.Purpose == "" || card.When == "" || card.Example == "" || card.Caveat == "" {
				t.Errorf("profile %q is missing teaching fields", card.ID)
			}
		case teachingKindConcept:
			if card.Group != teachingGroupConcept {
				t.Errorf("concept %q uses group %q, want %q", card.ID, card.Group, teachingGroupConcept)
			}
		default:
			t.Errorf("card %q uses unknown kind %q", card.ID, card.Kind)
		}

		if len(card.Purpose) > 180 {
			t.Errorf("card %q purpose is not a short readable line: %q", card.ID, card.Purpose)
		}
		if len(card.When) > 220 {
			t.Errorf("card %q when is too long: %q", card.ID, card.When)
		}
		if len(card.Example) > 240 {
			t.Errorf("card %q example is too long: %q", card.ID, card.Example)
		}
		// Safety renders as one scannable badge (FR-03, UX-03). It must still
		// deny automated execution and keep human approval explicit without
		// growing back into a paragraph of boilerplate.
		if !strings.Contains(card.Caveat, "No automated agent execution") {
			t.Errorf("card %q caveat must deny automated agent execution: %q", card.ID, card.Caveat)
		}
		if !strings.Contains(card.Caveat, "Human approval required") {
			t.Errorf("card %q caveat must keep human approval explicit: %q", card.ID, card.Caveat)
		}
		if runes := len([]rune(card.Caveat)); runes > 96 {
			t.Errorf("card %q caveat must stay a concise badge, got %d runes: %q", card.ID, runes, card.Caveat)
		}
	}
}

func TestTeachingExamplesAreHumanRequests(t *testing.T) {
	prefixes := []string{"Use ", "Ask "}
	shellMarkers := []string{
		"&&", "||", "sudo ", "$ ", "\n", "\r", "--apply",
		"go test", "go build", "go run", "git ", "curl ", "npm ", "chmod ",
	}
	for _, card := range teachingRegistry {
		example := card.Example
		hasPrefix := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(example, prefix) {
				hasPrefix = true
				break
			}
		}
		if !hasPrefix {
			t.Errorf("card %q example must read as a human request, got %q", card.ID, example)
		}
		for _, marker := range shellMarkers {
			if strings.Contains(example, marker) {
				t.Errorf("card %q example must not be a shell dispatch (contains %q): %q", card.ID, marker, example)
			}
		}
	}
}

// TestTeachingExamplesDoNotRepeatLabels guards the copy-stutter fix (UX-02):
// the detail view already labels the example, so an example must not restate
// that label or repeat a field name inside the sentence.
func TestTeachingExamplesDoNotRepeatLabels(t *testing.T) {
	cards := append([]teachingCard{}, teachingRegistry...)
	for _, kind := range []string{teachingKindSkill, teachingKindProfile, teachingKindConcept} {
		cards = append(cards, customTeachingCard("local-unlisted-item", kind))
	}
	repeated := []string{"ask your main agent", "example:", "purpose:", "when to use:"}
	for _, card := range cards {
		lower := strings.ToLower(card.Example)
		for _, label := range repeated {
			if strings.Contains(lower, label) {
				t.Errorf("card %q example repeats a field label %q: %q", card.ID, label, card.Example)
			}
		}
	}
}

func TestTeachingCustomFallback(t *testing.T) {
	cases := []struct {
		kind    string
		purpose string
	}{
		{teachingKindProfile, "Custom profile — purpose not documented by Agentic SDD."},
		{teachingKindSkill, "Custom skill — purpose not documented by Agentic SDD."},
		{teachingKindConcept, "Custom concept — not documented by Agentic SDD."},
	}
	for _, tc := range cases {
		got := teachingFor("local-unlisted-item", tc.kind)
		if got.ID != "local-unlisted-item" || got.Kind != tc.kind {
			t.Errorf("fallback for %s must preserve requested identity, got %+v", tc.kind, got)
		}
		if got.Purpose != tc.purpose {
			t.Errorf("fallback for %s purpose = %q, want generic %q", tc.kind, got.Purpose, tc.purpose)
		}
		if got.Group != teachingGroupCustom {
			t.Errorf("fallback for %s group = %q, want %q", tc.kind, got.Group, teachingGroupCustom)
		}
		if got.When == "" || got.Example == "" || got.Caveat == "" {
			t.Errorf("fallback for %s must still be a complete card: %+v", tc.kind, got)
		}
	}

	// An ID known in another kind must not borrow that role.
	borrowed := teachingFor("agentic-sdd-spec", teachingKindProfile)
	if !strings.Contains(borrowed.Purpose, "not documented") {
		t.Errorf("known skill ID looked up as a profile must fall back, got %q", borrowed.Purpose)
	}

	// The fallback invents no role, model or runtime claim.
	for _, kind := range []string{teachingKindSkill, teachingKindProfile, teachingKindConcept} {
		card := teachingFor("local-unlisted-item", kind)
		prose := strings.ToLower(card.Purpose + " " + card.When + " " + card.Example + " " + card.Caveat)
		for _, invented := range []string{"reviewer", "implementer", "architect", "read-only", "auditor", "test writer", "deepseek", "claude", "gpt", "glm", "beam"} {
			if strings.Contains(prose, invented) {
				t.Errorf("fallback for %s invents role/model %q: %q", kind, invented, prose)
			}
		}
	}
}

func TestTeachingConceptsStableOrder(t *testing.T) {
	want := []string{
		"spec-plan-tasks",
		"skill-profile",
		"main-subagent",
		"preview-apply",
		"free-first",
		"human-approval",
	}
	got := teachingConcepts()
	if len(got) != len(want) {
		t.Fatalf("teachingConcepts returned %d cards, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("concept %d = %q, want %q", i, got[i].ID, id)
		}
		if got[i].Kind != teachingKindConcept {
			t.Errorf("concept %q kind = %q, want %q", got[i].ID, got[i].Kind, teachingKindConcept)
		}
		if lookup := teachingFor(id, teachingKindConcept); lookup.ID != id || lookup.Purpose != got[i].Purpose {
			t.Errorf("teachingFor(%q, concept) does not match teachingConcepts entry", id)
		}
	}

	// Callers receive their own slice and cannot mutate the registry by accident.
	got[0].Purpose = "mutated"
	if again := teachingConcepts(); again[0].Purpose == "mutated" {
		t.Error("teachingConcepts must return an independent slice")
	}
}

func stripCodeSpans(text string) string {
	var out strings.Builder
	inSpan := false
	for _, r := range text {
		if r == '`' {
			inSpan = !inSpan
			continue
		}
		if !inSpan {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func TestTeachingNoModelIDsOrCostClaims(t *testing.T) {
	banned := []string{
		"deepseek", "claude", "sonnet", "opus", "gpt", "gemini", "glm", "mimo",
		"qwen", "beam", "openai", "anthropic", "9router", "antigravity", "opencode/",
		"free of charge", "zero cost", "no cost", "at no cost", "per token",
		"pricing", "guaranteed", "unlimited",
	}
	for _, card := range teachingRegistry {
		// Shipped IDs may legitimately contain a provider word; strip the
		// backticked ID spans so only explanatory prose is checked.
		prose := stripCodeSpans(card.Purpose + " " + card.When + " " + card.Example + " " + card.Caveat)
		prose = strings.ToLower(prose)
		for _, word := range banned {
			if strings.Contains(prose, word) {
				t.Errorf("card %q prose must omit model/cost claims, found %q", card.ID, word)
			}
		}
	}
}
