package tui

import (
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestB2CatalogItemAdditiveFieldsAndEnrichment(t *testing.T) {
	// Item without metadata should be enriched with curated metadata.
	item := CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  GroupCoreSkills,
	}
	enrichItem(&item)

	if item.Name != "SDD Router" {
		t.Errorf("item.Name = %q, want %q", item.Name, "SDD Router")
	}
	if !strings.Contains(item.Purpose, "Route a request") {
		t.Errorf("item.Purpose = %q, want router purpose", item.Purpose)
	}
	if item.When == "" || item.Example == "" {
		t.Errorf("item When/Example is empty: %+v", item)
	}
	if !strings.Contains(item.Caveat, "No automated agent execution") {
		t.Errorf("item.Caveat = %q, want safety caveat", item.Caveat)
	}
	if item.Category != "Scope/plan" {
		t.Errorf("item.Category = %q, want Scope/plan", item.Category)
	}
}

func TestB2UnknownItemNeutralFallback(t *testing.T) {
	item := CatalogItem{
		ID:     "unlisted-skill",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
	}
	enrichItem(&item)

	if item.Name != "unlisted-skill" {
		t.Errorf("unknown item.Name = %q, want exact ID", item.Name)
	}
	if !strings.Contains(item.Purpose, "not documented") {
		t.Errorf("unknown item.Purpose = %q, want not documented", item.Purpose)
	}
	if item.Category != "Custom" {
		t.Errorf("unknown item.Category = %q, want Custom", item.Category)
	}
}

func TestB2FilterValueIncludesAdditiveFields(t *testing.T) {
	item := CatalogItem{
		ID:       "test-id",
		Kind:     "skill",
		Origin:   "core",
		Group:    "Core skills",
		Name:     "Test Name",
		Purpose:  "Special Purpose Here",
		Category: "Scope/plan",
		Clients:  []ClientStatus{{Client: "claude"}},
	}
	filter := item.FilterValue()
	for _, term := range []string{"test-id", "test name", "special purpose here", "skill", "core", "scope/plan", "core skills", "claude"} {
		if !strings.Contains(filter, term) {
			t.Errorf("FilterValue %q missing term %q", filter, term)
		}
	}
}

func TestB2CategoriesAndActiveCategory(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
			{ID: "agentic-sdd-plan", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
			{ID: "agentic-sdd-orchestrate", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginPack, Group: GroupOpenCodePack},
			{ID: "sdd-lightweight", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPack, Group: GroupOpenCodeProfiles},
			{ID: "sdd-implement-1", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset, Group: GroupFreeFirstPresets},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	if active := m.ActiveCategory(); active != "all" {
		t.Errorf("initial ActiveCategory() = %q, want all", active)
	}

	categories := m.Categories()
	if len(categories) < 5 {
		t.Fatalf("expected at least 5 base categories, got %d", len(categories))
	}

	// Verify defensive copy
	categories[0].Count = 999
	categoriesAgain := m.Categories()
	if categoriesAgain[0].Count == 999 {
		t.Error("Categories() returned a mutable reference, expected defensive copy")
	}

	// Verify counts derive from inventory
	byID := make(map[string]int)
	for _, c := range categoriesAgain {
		byID[c.ID] = c.Count
	}
	if byID["all"] != 5 {
		t.Errorf("category all count = %d, want 5", byID["all"])
	}
	if byID["core"] != 2 {
		t.Errorf("category core count = %d, want 2", byID["core"])
	}
	if byID["pack"] != 1 {
		t.Errorf("category pack count = %d, want 1", byID["pack"])
	}
	if byID["profiles"] != 1 {
		t.Errorf("category profiles count = %d, want 1", byID["profiles"])
	}
	if byID["presets"] != 1 {
		t.Errorf("category presets count = %d, want 1", byID["presets"])
	}

	// Check purpose category exists and has correct count
	if byID["purpose:Scope/plan"] != 3 {
		t.Errorf("category purpose:Scope/plan count = %d, want 3", byID["purpose:Scope/plan"])
	}
}

func TestB2SetCategoryFiltersVisibleItemsAndPreservesSelection(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
			{ID: "agentic-sdd-plan", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
			{ID: "sdd-implement-1", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset, Group: GroupFreeFirstPresets},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Move cursor to second item (agentic-sdd-plan)
	m.list.Select(1)
	selected, ok := m.Selected()
	if !ok || selected.ID != "agentic-sdd-plan" {
		t.Fatalf("selected = %q, want agentic-sdd-plan", selected.ID)
	}

	// Set category to core. agentic-sdd-plan is in core, so selection should be preserved!
	cmd := m.SetCategory("core")
	if m.ActiveCategory() != "core" {
		t.Errorf("ActiveCategory() = %q, want core", m.ActiveCategory())
	}
	if len(m.VisibleItems()) != 2 {
		t.Fatalf("visible items = %d, want 2", len(m.VisibleItems()))
	}
	selectedAfter, ok := m.Selected()
	if !ok || selectedAfter.ID != "agentic-sdd-plan" {
		t.Fatalf("after SetCategory selection = %q, want agentic-sdd-plan", selectedAfter.ID)
	}
	_ = cmd

	// Switch category to presets. agentic-sdd-plan is not in presets, selection should reset safely to sdd-implement-1.
	m.SetCategory("presets")
	if len(m.VisibleItems()) != 1 {
		t.Fatalf("presets visible items = %d, want 1", len(m.VisibleItems()))
	}
	selectedPreset, ok := m.Selected()
	if !ok || selectedPreset.ID != "sdd-implement-1" {
		t.Fatalf("selection reset = %q, want sdd-implement-1", selectedPreset.ID)
	}

	// Unknown category should keep current category safely unchanged.
	m.SetCategory("nonexistent-category")
	if m.ActiveCategory() != "presets" {
		t.Errorf("after unknown category ActiveCategory() = %q, want presets", m.ActiveCategory())
	}
}

func TestB2DetailCardRendersRichSections(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{
				ID:     "agentic-sdd-router",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
				Group:  GroupCoreSkills,
				Clients: []ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/tmp/claude/router"},
				},
			},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)
	card := m.DetailCard(80)

	for _, want := range []string{
		"SDD Router",
		"agentic-sdd-router",
		"Purpose:",
		"When:",
		"Example:",
		"Caveat:",
		"Destination status:",
		"Next action:",
		"Installed (matches source)",
		"/tmp/claude/router",
		"agentic-sdd preview",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("DetailCard missing %q:\n%s", want, card)
		}
	}
}

func TestB2StaleFilterSafety(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: GroupCoreSkills},
			{ID: "sdd-implement-1", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset, Group: GroupFreeFirstPresets},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)
	cmd := m.SetCategory("core")
	if cmd != nil {
		for _, msg := range b2RunCmds(cmd) {
			updated, _ := m.Update(msg)
			m = updated.(CatalogModel)
		}
	}
	for _, it := range m.VisibleItems() {
		if it.Group != GroupCoreSkills {
			t.Fatalf("expected only core items, got %s", it.ID)
		}
	}
}

func TestB2NewCatalogModelDoesNotMutateCallerSnapshot(t *testing.T) {
	origItems := []CatalogItem{
		{
			ID:     "agentic-sdd-router",
			Kind:   skillsync.InventoryKindSkill,
			Origin: skillsync.InventoryOriginCore,
			Clients: []ClientStatus{
				{Client: "claude", Status: skillsync.InventoryStatusMissing, Path: "/tmp/path"},
			},
		},
	}
	callerCat := Catalog{Items: origItems}

	// NewCatalogModel must NOT mutate callerCat.Items in place.
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, callerCat)

	if callerCat.Items[0].Name != "" {
		t.Errorf("NewCatalogModel mutated callerCat.Items[0].Name = %q, want empty", callerCat.Items[0].Name)
	}
	if callerCat.Items[0].Purpose != "" {
		t.Errorf("NewCatalogModel mutated callerCat.Items[0].Purpose = %q, want empty", callerCat.Items[0].Purpose)
	}
	if callerCat.Items[0].Category != "" {
		t.Errorf("NewCatalogModel mutated callerCat.Items[0].Category = %q, want empty", callerCat.Items[0].Category)
	}

	// Verify mutating caller's Clients slice does not affect the model
	callerCat.Items[0].Clients[0].Client = "mutated-client"
	modelItems := m.VisibleItems()
	if len(modelItems) == 0 || modelItems[0].Clients[0].Client == "mutated-client" {
		t.Error("model shared Clients slice with caller snapshot")
	}
}

func TestB2OutOfOrderFilterResultsDiscarded(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "agentic-sdd-plan", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "sdd-implement-1", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Gen 1: SetCategory("core")
	_ = m.SetCategory("core")
	if m.ActiveCategory() != "core" {
		t.Fatalf("ActiveCategory = %q, want core", m.ActiveCategory())
	}
	genCore := m.filterGeneration

	// Gen 2: SetCategory("presets")
	_ = m.SetCategory("presets")
	if m.ActiveCategory() != "presets" {
		t.Fatalf("ActiveCategory = %q, want presets", m.ActiveCategory())
	}

	// Simulate stale out-of-order envelope from Gen 1 (core) arriving now
	staleEnvelope := FilterEnvelope{
		Generation: genCore,
		Category:   "core",
		Query:      "",
		Msg:        list.FilterMatchesMsg{},
	}
	updated, _ := m.Update(staleEnvelope)
	mUpdated := updated.(CatalogModel)

	// Active category and visible items must stay on presets (1 item), not reverted to core
	if mUpdated.ActiveCategory() != "presets" {
		t.Errorf("ActiveCategory reverted to %q, want presets", mUpdated.ActiveCategory())
	}
	if len(mUpdated.VisibleItems()) != 1 || mUpdated.VisibleItems()[0].ID != "sdd-implement-1" {
		t.Errorf("visible items corrupted by stale envelope: %+v", mUpdated.VisibleItems())
	}

	// Now send valid in-order envelope from current generation
	validEnvelope := FilterEnvelope{
		Generation: m.filterGeneration,
		Category:   "presets",
		Query:      "",
		Msg:        list.FilterMatchesMsg{},
	}
	updatedValid, _ := m.Update(validEnvelope)
	mValid := updatedValid.(CatalogModel)
	if mValid.ActiveCategory() != "presets" {
		t.Errorf("valid envelope corrupted ActiveCategory = %q", mValid.ActiveCategory())
	}
}

func TestB2FilteredSelectionPreservedAcrossCategory(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "agentic-sdd-plan", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "agentic-sdd-orchestrate", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginPack},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Filter for "plan"
	m.SetFilter("plan")
	if sel, ok := m.Selected(); !ok || sel.ID != "agentic-sdd-plan" {
		t.Fatalf("expected initial selected item agentic-sdd-plan, got %v (ok=%v)", sel.ID, ok)
	}

	// Change category to "core". "agentic-sdd-plan" is a core skill, so selection must be preserved.
	cmd := m.SetCategory("core")
	if cmd != nil {
		for _, msg := range b2RunCmds(cmd) {
			next, _ := m.Update(msg)
			m = next.(CatalogModel)
		}
	}

	afterSel, ok := m.Selected()
	if !ok || afterSel.ID != "agentic-sdd-plan" {
		t.Fatalf("after SetCategory selection = %q, want agentic-sdd-plan", afterSel.ID)
	}
}

func TestB2MismatchFixtureCountsAuthoritativeKindOrigin(t *testing.T) {
	// Item has Kind: profile, Origin: pack, but incorrect caller Group: "Core skills"
	mismatched := CatalogItem{
		ID:     "sdd-docs-worker",
		Kind:   skillsync.InventoryKindProfile,
		Origin: skillsync.InventoryOriginPack,
		Group:  "Core skills", // Incorrect group! Authoritative Kind+Origin says it is OpenCode profile
	}
	cat := Catalog{Items: []CatalogItem{mismatched}}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Categories count should classify authoritatively under profiles, NOT core
	counts := make(map[string]int)
	for _, c := range m.Categories() {
		counts[c.ID] = c.Count
	}
	if counts["core"] != 0 {
		t.Errorf("mismatched profile counted as core: %d", counts["core"])
	}
	if counts["profiles"] != 1 {
		t.Errorf("mismatched profile not counted under profiles: %d", counts["profiles"])
	}

	// SetCategory("profiles") must find it
	m.SetCategory("profiles")
	if len(m.VisibleItems()) != 1 || m.VisibleItems()[0].ID != "sdd-docs-worker" {
		t.Errorf("SetCategory(profiles) failed to show authoritative profile: %+v", m.VisibleItems())
	}

	// SetCategory("core") must NOT find it
	m.SetCategory("core")
	if len(m.VisibleItems()) != 0 {
		t.Errorf("SetCategory(core) showed mismatched profile: %+v", m.VisibleItems())
	}
}

func TestB2DisplayItemPreservesCuratedMetadata(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{
				ID:     "agentic-sdd-router",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
			},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)
	selected, ok := m.Selected()
	if !ok {
		t.Fatal("no selection")
	}

	// Simulate parent displayCopy setting displayItem with escaped ID
	displayCopyItem := selected
	displayCopyItem.ID = "\\u0061gentic-sdd-router" // Escaped ID
	m.displayItem = &displayCopyItem

	card := m.DetailCard(80)

	// DetailCard must preserve the curated purpose and name, not fall back to generic "not documented"
	if strings.Contains(card, "not documented") {
		t.Errorf("displayItem wiped curated metadata with neutral fallback:\n%s", card)
	}
	if !strings.Contains(card, "SDD Router") {
		t.Errorf("displayItem DetailCard missing DisplayName 'SDD Router':\n%s", card)
	}
	if !strings.Contains(card, "Route a request through combined spec") {
		t.Errorf("displayItem DetailCard missing curated purpose:\n%s", card)
	}
}

func TestB2DetailCardWrapsProseAtNarrowWidth(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{
				ID:     "agentic-sdd-router",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
			},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Test narrow width 40
	card40 := m.DetailCard(40)
	for i, line := range strings.Split(card40, "\n") {
		if cell := ansi.StringWidth(line); cell > 40 {
			t.Errorf("line %d width %d exceeds 40 cells: %q", i, cell, line)
		}
	}
}

func b2RunCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, member := range batch {
			msgs = append(msgs, b2RunCmds(member)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}
