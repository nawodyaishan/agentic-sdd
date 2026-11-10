package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"agentic-sdd/experiments/tui"
	"agentic-sdd/internal/skillsync"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Feature 013 B2 acceptance tests for the enriched, categorized catalog.
//
// The tests are black box: they use only the exported surface of the
// experimental TUI package. The B2 metadata and category members are reached
// reflectively so a parallel lane renaming or dropping one fails the owning
// test with a precise message instead of breaking the build for every other
// lane in the same package.
//
// What this file covers:
//
//   - curated metadata enrichment from the shared registry, including the
//     neutral fallback for unknown IDs and the absence of fabricated runtime
//     state;
//   - categories derived from the authoritative inventory, never from
//     hardcoded example numbers, and never fabricated for absent content;
//   - search over exact ID, display name, purpose and category, with the
//     asynchronous Bubbles filter result actually executed and consumed;
//   - a search with zero matches staying usable instead of panicking;
//   - the optional preset category appearing only when a preset is loaded;
//   - selection identity (kind + ID) preserved across a category change, a
//     filter change and a resize;
//   - rich detail fields plus readable label/value separation, no repeated
//     field labels and reasonable section spacing;
//   - long paths and commands still reachable by paging;
//   - exact cell bounds at the documented width and height breakpoints;
//   - a Home overview with inventory-derived counts and the spec's
//     recommended installation action;
//   - a read-only run, including category and search interaction, that
//     writes nothing to the home or the source tree.
//
// No test here asserts a full-screen snapshot: the assertions target
// behaviour, values and bounds that must hold whatever the rendering looks
// like.

const (
	// b2CommandTimeout bounds a single command execution. It only prevents a
	// stuck command from hanging the suite; it never replaces an assertion.
	// A command that does not answer in time is a timer (cursor blink,
	// spinner tick) and is ignored, so a test never enters an endless tick
	// loop.
	b2CommandTimeout = 3 * time.Second

	// b2DetailWidth is the width used for detail-card assertions.
	b2DetailWidth = 100
)

// b2LabelPattern matches a "Label: value" line. The label is a short phrase
// of letters, spaces and punctuation that starts with a letter.
var b2LabelPattern = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9 &/'_-]{2,38}):`)

// ---------------------------------------------------------------------------
// Reflection helpers for the additive B2 API.
// ---------------------------------------------------------------------------

// b2ItemField returns the value of an additive metadata field on a catalog
// item. The field is read reflectively so this file keeps compiling against
// the pre-B2 package while still failing loudly when the field is missing.
func b2ItemField(t *testing.T, item tui.CatalogItem, name string) string {
	t.Helper()
	field := reflect.ValueOf(item).FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("CatalogItem.%s is not implemented yet (Feature 013 B2 metadata enrichment)", name)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("CatalogItem.%s is a %s, want an additive string field", name, field.Kind())
	}
	return field.String()
}

// b2Method returns a bound method of the catalog model. The model is passed
// by pointer so both value and pointer receivers are reachable.
func b2Method(t *testing.T, model *tui.CatalogModel, name string) reflect.Value {
	t.Helper()
	method := reflect.ValueOf(model).MethodByName(name)
	if !method.IsValid() {
		t.Fatalf("CatalogModel.%s() is not implemented yet (Feature 013 B2 categorized catalog)", name)
	}
	return method
}

// b2Category is one entry of the model's category list.
type b2Category struct {
	ID    string
	Label string
	Count int
}

// b2Categories returns the model's categories.
func b2Categories(t *testing.T, model *tui.CatalogModel) []b2Category {
	t.Helper()
	results := b2Method(t, model, "Categories").Call(nil)
	if len(results) != 1 {
		t.Fatalf("Categories() returned %d values, want 1", len(results))
	}
	slice := results[0]
	if slice.Kind() != reflect.Slice {
		t.Fatalf("Categories() returned a %s, want a slice of categories", slice.Kind())
	}
	categories := make([]b2Category, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		entry := slice.Index(i)
		categories = append(categories, b2Category{
			ID:    b2CategoryString(t, entry, "ID"),
			Label: b2CategoryString(t, entry, "Label"),
			Count: b2CategoryInt(t, entry, "Count"),
		})
	}
	return categories
}

// b2ActiveCategory returns the model's active category ID.
func b2ActiveCategory(t *testing.T, model *tui.CatalogModel) string {
	t.Helper()
	results := b2Method(t, model, "ActiveCategory").Call(nil)
	if len(results) != 1 || results[0].Kind() != reflect.String {
		t.Fatalf("ActiveCategory() must return one string, got %v", results)
	}
	return results[0].String()
}

// b2SetCategory activates a category and returns the command it produced.
func b2SetCategory(t *testing.T, model *tui.CatalogModel, id string) tea.Cmd {
	t.Helper()
	results := b2Method(t, model, "SetCategory").Call([]reflect.Value{reflect.ValueOf(id)})
	if len(results) != 1 {
		t.Fatalf("SetCategory(%q) returned %d values, want 1 command", id, len(results))
	}
	cmd, ok := results[0].Interface().(tea.Cmd)
	if !ok {
		t.Fatalf("SetCategory(%q) returned %T, want a tea.Cmd", id, results[0].Interface())
	}
	return cmd
}

// b2CategoryString reads one string field of a category entry.
func b2CategoryString(t *testing.T, entry reflect.Value, name string) string {
	t.Helper()
	field := entry.FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("the category type has no %q field (Feature 013 B2 categories)", name)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("category field %q is a %s, want a string", name, field.Kind())
	}
	return field.String()
}

// b2CategoryInt reads one integer field of a category entry.
func b2CategoryInt(t *testing.T, entry reflect.Value, name string) int {
	t.Helper()
	field := entry.FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("the category type has no %q field (Feature 013 B2 categories)", name)
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(field.Int())
	default:
		t.Fatalf("category field %q is a %s, want an int", name, field.Kind())
		return 0
	}
}

// ---------------------------------------------------------------------------
// Command helpers.
// ---------------------------------------------------------------------------

// b2RunCmds executes a command and returns the messages it produced. A batch
// is expanded into its members. Cursor blink and spinner messages are timers:
// they are returned but never treated as state.
func b2RunCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()
	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(b2CommandTimeout):
		// A command that does not answer in time is a timer command; it is
		// ignored rather than waited on forever.
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

// b2ConsumeFilterResults feeds every Bubbles filter result of a command into
// the model and reports how many results were consumed. The count proves the
// asynchronous search really ran, instead of only proving that a command was
// returned.
func b2ConsumeFilterResults(t *testing.T, model *tui.CatalogModel, cmd tea.Cmd) int {
	t.Helper()
	consumed := 0
	for _, msg := range b2RunCmds(cmd) {
		result, ok := msg.(list.FilterMatchesMsg)
		if !ok {
			continue
		}
		consumed++
		next, _ := model.Update(result)
		updated, ok := next.(tui.CatalogModel)
		if !ok {
			t.Fatalf("Update returned %T, want tui.CatalogModel", next)
		}
		*model = updated
	}
	return consumed
}

// ---------------------------------------------------------------------------
// Fixture helpers.
// ---------------------------------------------------------------------------

// b2CatalogWith builds a catalog from the given items.
func b2CatalogWith(items ...tui.CatalogItem) tui.Catalog {
	return tui.Catalog{Items: items}
}

// b2Client builds one destination status row.
func b2Client(name string, status skillsync.InventoryStatus, path string) tui.ClientStatus {
	return tui.ClientStatus{Client: name, Status: status, Path: path}
}

// b2CountsByGroup counts catalog items per existing presentation group. The
// expected category counts are derived from the authoritative inventory this
// way, never from hardcoded example numbers.
func b2CountsByGroup(cat tui.Catalog) map[string]int {
	counts := map[string]int{
		"all":      len(cat.Items),
		"core":     0,
		"pack":     0,
		"profiles": 0,
		"presets":  0,
	}
	for _, item := range cat.Items {
		switch item.Group {
		case tui.GroupCoreSkills:
			counts["core"]++
		case tui.GroupOpenCodePack:
			counts["pack"]++
		case tui.GroupOpenCodeProfiles:
			counts["profiles"]++
		case tui.GroupFreeFirstPresets:
			counts["presets"]++
		}
	}
	return counts
}

// b2CategoryByID finds a category by ID, case-insensitively.
func b2CategoryByID(categories []b2Category, id string) (b2Category, bool) {
	for _, category := range categories {
		if strings.EqualFold(category.ID, id) {
			return category, true
		}
	}
	return b2Category{}, false
}

// b2PurposeCategoryID maps a catalog item's category to the purpose category
// ID the model exposes. The item may carry the teaching group ("Scope/plan")
// or the full category ID ("purpose:Scope/plan").
func b2PurposeCategoryID(t *testing.T, item tui.CatalogItem) string {
	t.Helper()
	category := strings.TrimSpace(b2ItemField(t, item, "Category"))
	if category == "" {
		t.Fatalf("CatalogItem.Category is empty; every item needs a category identity")
	}
	if strings.HasPrefix(strings.ToLower(category), "purpose:") {
		return category
	}
	return "purpose:" + category
}

// b2CardLines returns the detail card lines without trailing blank lines.
func b2CardLines(t *testing.T, model *tui.CatalogModel, width int) []string {
	t.Helper()
	lines := model.DetailCardLines(width)
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// b2VisibleIDs returns the IDs of the visible catalog rows.
func b2VisibleIDs(items []tui.CatalogItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Metadata enrichment.
// ---------------------------------------------------------------------------

// TestB2KnownItemIsEnrichedFromSharedMetadata proves that the catalog shows
// curated purpose metadata for a shipped item instead of only its raw ID.
func TestB2KnownItemIsEnrichedFromSharedMetadata(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	var router *tui.CatalogItem
	for _, item := range model.VisibleItems() {
		if item.ID == "agentic-sdd-router" {
			copied := item
			router = &copied
			break
		}
	}
	if router == nil {
		t.Fatalf("the embedded inventory no longer offers agentic-sdd-router; the fixture assumption is stale")
	}

	name := b2ItemField(t, *router, "Name")
	if strings.TrimSpace(name) == "" {
		t.Error("the enriched item has no display name")
	}
	if name == router.ID {
		t.Errorf("the enriched display name is still the raw ID %q; the curated label was not applied", name)
	}
	if name != "SDD Router" {
		t.Errorf("display name = %q, want the curated label %q", name, "SDD Router")
	}

	for _, field := range []string{"Purpose", "When", "Example"} {
		if value := strings.TrimSpace(b2ItemField(t, *router, field)); value == "" {
			t.Errorf("the enriched item has no %s text", field)
		}
	}

	caveat := b2ItemField(t, *router, "Caveat")
	if !strings.Contains(caveat, "No automated agent execution") || !strings.Contains(caveat, "Human approval required") {
		t.Errorf("caveat = %q, want it to keep the shared safety meaning", caveat)
	}
}

// TestB2UnknownItemFallsBackNeutrally proves that an item without curated
// metadata is described honestly instead of with an invented explanation.
func TestB2UnknownItemFallsBackNeutrally(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "local-unlisted-item",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/home/.claude/skills/local-unlisted-item"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 100, Height: 24}, cat)

	item, ok := model.Selected()
	if !ok {
		t.Fatal("the catalog selected no item")
	}

	if name := b2ItemField(t, item, "Name"); name != item.ID {
		t.Errorf("unknown item display name = %q, want the exact ID %q as the neutral label", name, item.ID)
	}
	purpose := b2ItemField(t, item, "Purpose")
	if !strings.Contains(purpose, "not documented") {
		t.Errorf("unknown item purpose = %q, want the neutral 'not documented' wording", purpose)
	}
	if when := strings.TrimSpace(b2ItemField(t, item, "When")); when == "" {
		t.Error("the unknown item has no 'When' fallback text")
	}
	if example := strings.TrimSpace(b2ItemField(t, item, "Example")); example == "" {
		t.Error("the unknown item has no 'Example' fallback text")
	}
	if caveat := strings.TrimSpace(b2ItemField(t, item, "Caveat")); caveat == "" {
		t.Error("the unknown item has no 'Caveat' fallback text")
	}

	// The neutral fallback must not fabricate runtime state.
	forbidden := []string{
		"running", "runs", "enabled", "authenticated", "sandboxed",
		"pre-approved", "always available", "model access",
	}
	metadata := strings.ToLower(strings.Join([]string{
		purpose,
		b2ItemField(t, item, "When"),
		b2ItemField(t, item, "Example"),
		b2ItemField(t, item, "Caveat"),
	}, " "))
	for _, phrase := range forbidden {
		if strings.Contains(metadata, phrase) {
			t.Errorf("the unknown item's metadata claims %q; the fallback must not fabricate runtime state", phrase)
		}
	}

	// An unknown item must not be filed under a curated purpose group.
	if category := strings.TrimSpace(b2ItemField(t, item, "Category")); category != "" {
		normalized := strings.ToLower(category)
		if normalized != "custom" && normalized != "purpose:custom" {
			t.Errorf("unknown item category = %q, want the neutral %q group", category, "Custom")
		}
	}
}

// TestB2UnknownItemDetailShowsNoFabricatedState proves the rendered detail
// card of an unknown item stays honest about what the inventory knows.
func TestB2UnknownItemDetailShowsNoFabricatedState(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "local-unlisted-item",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/home/.claude/skills/local-unlisted-item"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 100, Height: 24}, cat)

	card := model.DetailCard(b2DetailWidth)
	if !strings.Contains(card, "Template available (not installed)") {
		t.Errorf("the detail card does not report the truthful missing status:\n%s", card)
	}
	for _, claim := range []string{"Installed (matches source)", "Installed (modified)", "Unavailable / refused"} {
		if strings.Contains(card, claim) {
			t.Errorf("the detail card claims %q for a destination the inventory reports as missing:\n%s", claim, card)
		}
	}
	for _, phrase := range []string{"is running", "is enabled", "authenticated", "sandboxed"} {
		if strings.Contains(strings.ToLower(card), phrase) {
			t.Errorf("the detail card fabricates runtime state (%q):\n%s", phrase, card)
		}
	}
}

// ---------------------------------------------------------------------------
// Categories.
// ---------------------------------------------------------------------------

// TestB2CategoriesCountTheRealInventory proves the category counts come from
// the authoritative inventory instead of hardcoded example numbers.
func TestB2CategoriesCountTheRealInventory(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)
	categories := b2Categories(t, &model)

	for id, want := range b2CountsByGroup(cat) {
		category, ok := b2CategoryByID(categories, id)
		if !ok {
			t.Errorf("Categories() does not offer the %q category; got %v", id, b2CategoryIDs(categories))
			continue
		}
		if category.Count != want {
			t.Errorf("category %q count = %d, want %d (derived from the inventory)", id, category.Count, want)
		}
	}

	purposeTotal := 0
	purposeCount := 0
	for _, category := range categories {
		if !strings.HasPrefix(strings.ToLower(category.ID), "purpose:") {
			continue
		}
		purposeCount++
		purposeTotal += category.Count
		if category.Count <= 0 {
			t.Errorf("purpose category %q is listed with count %d; absent content must not be listed", category.ID, category.Count)
		}
		if strings.TrimSpace(category.Label) == "" {
			t.Errorf("purpose category %q has no label", category.ID)
		}
	}
	if purposeCount == 0 {
		t.Fatal("Categories() offers no purpose category; the teaching groups must be reachable")
	}
	if purposeTotal != len(cat.Items) {
		t.Errorf("the purpose categories cover %d items, want %d; every item has exactly one teaching group", purposeTotal, len(cat.Items))
	}
}

// TestB2CategoriesDoNotFabricateAbsentContent proves that content which is
// not loaded is not counted and not invented as a category.
func TestB2CategoriesDoNotFabricateAbsentContent(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if cat.ProfileCount() != 0 {
		t.Fatalf("the core-only fixture holds %d profiles; the fixture assumption is stale", cat.ProfileCount())
	}

	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)
	categories := b2Categories(t, &model)

	for _, id := range []string{"profiles", "presets"} {
		if category, ok := b2CategoryByID(categories, id); ok && category.Count != 0 {
			t.Errorf("category %q counts %d items although no profile or preset is loaded", id, category.Count)
		}
	}
	for _, category := range categories {
		if category.Count < 0 {
			t.Errorf("category %q reports a negative count %d", category.ID, category.Count)
		}
		if category.Count > 0 && strings.EqualFold(category.ID, "all") && category.Count != len(cat.Items) {
			t.Errorf("category %q counts %d items, want %d", category.ID, category.Count, len(cat.Items))
		}
	}
}

// TestB2PurposeCategoryFollowsTeachingGroup proves that a known item is
// reachable through the purpose category of its teaching group.
func TestB2PurposeCategoryFollowsTeachingGroup(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/home/.claude/skills/agentic-sdd-router"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	item, ok := model.Selected()
	if !ok {
		t.Fatal("the catalog selected no item")
	}
	wantID := b2PurposeCategoryID(t, item)
	if !strings.EqualFold(wantID, "purpose:Scope/plan") {
		t.Errorf("purpose category ID = %q, want %q for the router's teaching group", wantID, "purpose:Scope/plan")
	}

	categories := b2Categories(t, &model)
	category, ok := b2CategoryByID(categories, wantID)
	if !ok {
		t.Fatalf("Categories() does not offer %q; got %v", wantID, b2CategoryIDs(categories))
	}
	if category.Count != 1 {
		t.Errorf("purpose category %q count = %d, want 1", category.ID, category.Count)
	}
}

// TestB2CategoryFilterSelectsMatchingItems proves that activating a category
// really narrows the visible rows to that category.
func TestB2CategoryFilterSelectsMatchingItems(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	cmd := b2SetCategory(t, &model, "core")
	b2ConsumeFilterResults(t, &model, cmd)

	visible := model.VisibleItems()
	want := b2CountsByGroup(cat)["core"]
	if len(visible) != want {
		t.Fatalf("the core category shows %d rows, want %d (derived from the inventory): %v. If SetCategory has a value receiver, the model cannot change",
			len(visible), want, b2VisibleIDs(visible))
	}
	for _, item := range visible {
		if item.Group != tui.GroupCoreSkills {
			t.Errorf("the core category shows %q from group %q", item.ID, item.Group)
		}
	}
	if got := b2ActiveCategory(t, &model); !strings.EqualFold(got, "core") {
		t.Errorf("ActiveCategory() = %q, want %q", got, "core")
	}
}

// TestB2CategorySwitchRestoresEveryItem proves that switching back to the
// "all" category restores the full inventory.
func TestB2CategorySwitchRestoresEveryItem(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true, OpenCodeFreeFirst: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	b2ConsumeFilterResults(t, &model, b2SetCategory(t, &model, "presets"))
	if got := len(model.VisibleItems()); got != b2CountsByGroup(cat)["presets"] {
		t.Fatalf("the presets category shows %d rows, want %d", got, b2CountsByGroup(cat)["presets"])
	}

	b2ConsumeFilterResults(t, &model, b2SetCategory(t, &model, "all"))
	if got := len(model.VisibleItems()); got != len(cat.Items) {
		t.Errorf("the all category shows %d rows, want %d", got, len(cat.Items))
	}
	if got := b2ActiveCategory(t, &model); !strings.EqualFold(got, "all") {
		t.Errorf("ActiveCategory() = %q, want %q", got, "all")
	}
}

// ---------------------------------------------------------------------------
// Search.
// ---------------------------------------------------------------------------

// TestB2SearchMatchesDisplayNameAndPurpose proves the search covers the
// curated display name and the purpose text, not only the raw ID.
func TestB2SearchMatchesDisplayNameAndPurpose(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	var router *tui.CatalogItem
	for _, item := range model.VisibleItems() {
		if item.ID == "agentic-sdd-router" {
			copied := item
			router = &copied
			break
		}
	}
	if router == nil {
		t.Fatal("the embedded inventory no longer offers agentic-sdd-router; the fixture assumption is stale")
	}

	filterValue := strings.ToLower(router.FilterValue())
	name := strings.ToLower(b2ItemField(t, *router, "Name"))
	if !strings.Contains(filterValue, name) {
		t.Errorf("the search text %q does not contain the display name %q", router.FilterValue(), name)
	}
	purpose := strings.ToLower(b2ItemField(t, *router, "Purpose"))
	distinctive := b2DistinctiveWord(purpose)
	if distinctive == "" {
		t.Fatal("the enriched purpose has no distinctive word to search for")
	}
	if !strings.Contains(filterValue, distinctive) {
		t.Errorf("the search text %q does not contain the purpose word %q", router.FilterValue(), distinctive)
	}

	model.SetFilter(name)
	visible := model.VisibleItems()
	if !b2ContainsID(visible, router.ID) {
		t.Errorf("searching the display name %q did not find %q; visible: %v", name, router.ID, b2VisibleIDs(visible))
	}

	model.SetFilter(distinctive)
	visible = model.VisibleItems()
	if !b2ContainsID(visible, router.ID) {
		t.Errorf("searching the purpose word %q did not find %q; visible: %v", distinctive, router.ID, b2VisibleIDs(visible))
	}
}

// TestB2SearchMatchesExactIDAndCategory proves the search also covers the
// exact ID and the category the catalog assigns, not only the display name
// and the purpose text.
func TestB2SearchMatchesExactIDAndCategory(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	var router *tui.CatalogItem
	for _, item := range model.VisibleItems() {
		if item.ID == "agentic-sdd-router" {
			copied := item
			router = &copied
			break
		}
	}
	if router == nil {
		t.Fatal("the embedded inventory no longer offers agentic-sdd-router; the fixture assumption is stale")
	}

	model.SetFilter(router.ID)
	if visible := model.VisibleItems(); !b2ContainsID(visible, router.ID) {
		t.Errorf("searching the exact ID %q did not find it; visible: %v", router.ID, b2VisibleIDs(visible))
	}

	category := strings.TrimSpace(b2ItemField(t, *router, "Category"))
	if category == "" {
		t.Fatal("the router carries no category; the search cannot be checked against it")
	}
	model.SetFilter(category)
	if visible := model.VisibleItems(); !b2ContainsID(visible, router.ID) {
		t.Errorf("searching the category %q did not find %q; visible: %v", category, router.ID, b2VisibleIDs(visible))
	}
}

// TestB2ZeroMatchSearchStaysUsable proves a search that matches nothing
// leaves an empty but usable screen instead of panicking or inventing rows,
// and that clearing it restores the inventory.
func TestB2ZeroMatchSearchStaysUsable(t *testing.T) {
	cat := b2CatalogWith(
		tui.CatalogItem{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
		tui.CatalogItem{ID: "beta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
	)
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	model.SetFilter("zzz-no-such-entry")
	if visible := model.VisibleItems(); len(visible) != 0 {
		t.Fatalf("a search that matches nothing shows %d rows: %v", len(visible), b2VisibleIDs(visible))
	}
	if item, ok := model.Selected(); ok {
		t.Errorf("an empty result still reports the selection %q", item.ID)
	}
	if card := model.DetailCard(b2DetailWidth); strings.TrimSpace(card) == "" {
		t.Error("the detail card is blank when nothing matches; it must explain the empty result")
	}

	// The empty state must still render inside the terminal bounds.
	app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 24, Color: false}, cat)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)
	next, _ = app.Update(runes("/"))
	app = next.(tui.App)
	for _, ch := range "zzz-no-such-entry" {
		next, _ = app.Update(runes(string(ch)))
		app = next.(tui.App)
	}
	b2AssertWithinBounds(t, app.View().Content, 100, 24, "zero-match search")

	model.SetFilter("")
	if visible := model.VisibleItems(); len(visible) != len(cat.Items) {
		t.Errorf("clearing the search shows %d rows, want the full inventory of %d", len(visible), len(cat.Items))
	}
	if _, ok := model.Selected(); !ok {
		t.Error("clearing the search left no selection")
	}
}

// TestB2PresetCategoryIsOptionalAndDerived proves the preset category is
// offered only for loaded presets: it counts a real preset row and is empty
// for a core-only inventory.
func TestB2PresetCategoryIsOptionalAndDerived(t *testing.T) {
	withPreset := b2CatalogWith(tui.CatalogItem{
		ID:     "opencode-free-first",
		Kind:   skillsync.InventoryKindProfile,
		Origin: skillsync.InventoryOriginPreset,
		Group:  tui.GroupFreeFirstPresets,
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, withPreset)
	categories := b2Categories(t, &model)

	presets, ok := b2CategoryByID(categories, "presets")
	if !ok {
		t.Fatalf("Categories() does not offer the presets category; got %v", b2CategoryIDs(categories))
	}
	if presets.Count != 1 {
		t.Errorf("presets category count = %d, want 1 for the loaded preset", presets.Count)
	}

	b2ConsumeFilterResults(t, &model, b2SetCategory(t, &model, "presets"))
	visible := model.VisibleItems()
	if len(visible) != 1 || visible[0].ID != "opencode-free-first" {
		t.Errorf("the presets category shows %v, want [opencode-free-first]", b2VisibleIDs(visible))
	}

	// A core-only inventory must not fabricate the optional preset content.
	coreOnly := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, b2CatalogWith(
		tui.CatalogItem{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
	))
	coreCategories := b2Categories(t, &coreOnly)
	if category, ok := b2CategoryByID(coreCategories, "presets"); ok && category.Count != 0 {
		t.Errorf("the presets category counts %d rows although no preset is loaded", category.Count)
	}
	b2ConsumeFilterResults(t, &coreOnly, b2SetCategory(t, &coreOnly, "presets"))
	if visible := coreOnly.VisibleItems(); len(visible) != 0 {
		t.Errorf("the presets category shows %v for a core-only inventory", b2VisibleIDs(visible))
	}
}

// TestB2AsyncSearchResultIsExecutedAndConsumed proves the asynchronous
// Bubbles filter actually runs: the filter result message is produced and
// consumed, and the visible rows change because of it.
func TestB2AsyncSearchResultIsExecutedAndConsumed(t *testing.T) {
	cat := b2CatalogWith(
		tui.CatalogItem{ID: "alpha-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
		tui.CatalogItem{ID: "beta-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
		tui.CatalogItem{ID: "gamma-thing", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
	)
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	if got := len(model.VisibleItems()); got != 3 {
		t.Fatalf("the unfiltered catalog shows %d rows, want 3", got)
	}

	// "/" opens the search input.
	next, cmd := model.Update(runes("/"))
	model = next.(tui.CatalogModel)
	if !model.IsFiltering() {
		t.Fatal("pressing / did not open the search input")
	}
	b2ConsumeFilterResults(t, &model, cmd)

	// Typing a character schedules the asynchronous filter.
	next, cmd = model.Update(runes("b"))
	model = next.(tui.CatalogModel)
	consumed := b2ConsumeFilterResults(t, &model, cmd)
	if consumed == 0 {
		t.Fatal("typing in the search produced no Bubbles filter result; the search never ran")
	}

	visible := b2VisibleIDs(model.VisibleItems())
	if len(visible) != 1 || visible[0] != "beta-thing" {
		t.Fatalf("after searching for b the visible rows are %v, want [beta-thing]", visible)
	}
}

// ---------------------------------------------------------------------------
// Selection identity.
// ---------------------------------------------------------------------------

// TestB2SelectionSurvivesCategoryFilterAndResize proves the selection is
// kept by its stable kind and ID, not by its row number.
func TestB2SelectionSurvivesCategoryFilterAndResize(t *testing.T) {
	cat := b2CatalogWith(
		tui.CatalogItem{ID: "alpha-skill", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
		tui.CatalogItem{ID: "beta-skill", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore, Group: tui.GroupCoreSkills},
		tui.CatalogItem{ID: "gamma-profile", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPack, Group: tui.GroupOpenCodeProfiles},
	)
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	next, _ := model.Update(runes("j"))
	model = next.(tui.CatalogModel)
	selected, ok := model.Selected()
	if !ok || selected.ID != "beta-skill" {
		t.Fatalf("after moving down the selection is %q (ok=%v), want beta-skill", selected.ID, ok)
	}

	// A category change that keeps the selected item must keep the selection.
	b2ConsumeFilterResults(t, &model, b2SetCategory(t, &model, "core"))
	afterCategory, ok := model.Selected()
	if !ok {
		t.Fatal("the selection was lost by the category change")
	}
	if afterCategory.Kind != selected.Kind || afterCategory.ID != selected.ID {
		t.Errorf("the category change moved the selection from %s/%s to %s/%s; the identity must be kept",
			selected.Kind, selected.ID, afterCategory.Kind, afterCategory.ID)
	}

	// A filter that keeps the selected item must keep the selection.
	next, cmd := model.Update(runes("/"))
	model = next.(tui.CatalogModel)
	b2ConsumeFilterResults(t, &model, cmd)
	next, cmd = model.Update(runes("s"))
	model = next.(tui.CatalogModel)
	b2ConsumeFilterResults(t, &model, cmd)
	afterFilter, ok := model.Selected()
	if !ok {
		t.Fatal("the selection was lost by the filter")
	}
	if afterFilter.Kind != selected.Kind || afterFilter.ID != selected.ID {
		t.Errorf("the filter moved the selection from %s/%s to %s/%s; the identity must be kept",
			selected.Kind, selected.ID, afterFilter.Kind, afterFilter.ID)
	}

	// A resize must keep the selection.
	next, _ = model.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	model = next.(tui.CatalogModel)
	afterResize, ok := model.Selected()
	if !ok {
		t.Fatal("the selection was lost by the resize")
	}
	if afterResize.Kind != selected.Kind || afterResize.ID != selected.ID {
		t.Errorf("the resize moved the selection from %s/%s to %s/%s; the identity must be kept",
			selected.Kind, selected.ID, afterResize.Kind, afterResize.ID)
	}
}

// ---------------------------------------------------------------------------
// Rich detail and spacing.
// ---------------------------------------------------------------------------

// TestB2DetailShowsCompleteRichFields proves the detail pane shows every
// curated field and every truthful per-client status.
func TestB2DetailShowsCompleteRichFields(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMatchesSource, "/tmp/home/.claude/skills/agentic-sdd-router"),
			b2Client("codex", skillsync.InventoryStatusDiffersFromSource, "/tmp/home/.codex/skills/agentic-sdd-router"),
			b2Client("agents", skillsync.InventoryStatusMissing, "/tmp/home/.agents/skills/agentic-sdd-router"),
			b2Client("agy", skillsync.InventoryStatusUnavailable, "/tmp/home/.agy/skills/agentic-sdd-router"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	item, ok := model.Selected()
	if !ok {
		t.Fatal("the catalog selected no item")
	}
	card := model.DetailCard(b2DetailWidth)

	if !strings.Contains(card, item.ID) {
		t.Errorf("the detail card does not show the exact ID %q:\n%s", item.ID, card)
	}
	name := b2ItemField(t, item, "Name")
	if !strings.Contains(card, name) {
		t.Errorf("the detail card does not show the display name %q:\n%s", name, card)
	}
	for _, field := range []string{"Purpose", "When", "Example", "Caveat"} {
		value := b2ItemField(t, item, field)
		if !b2ShowsValue(card, value) {
			t.Errorf("the detail card does not show the %s text %q:\n%s", field, value, card)
		}
	}
	if !strings.Contains(card, item.Origin) {
		t.Errorf("the detail card does not show the origin %q:\n%s", item.Origin, card)
	}
	if !strings.Contains(card, item.Kind) {
		t.Errorf("the detail card does not show the kind %q:\n%s", item.Kind, card)
	}

	for _, badge := range []string{
		tui.StatusBadge(skillsync.InventoryStatusMatchesSource),
		tui.StatusBadge(skillsync.InventoryStatusDiffersFromSource),
		tui.StatusBadge(skillsync.InventoryStatusMissing),
		tui.StatusBadge(skillsync.InventoryStatusUnavailable),
	} {
		if count := strings.Count(card, badge); count != 1 {
			t.Errorf("the status %q appears %d times, want exactly once:\n%s", badge, count, card)
		}
	}
	for _, client := range item.Clients {
		if !strings.Contains(card, client.Client) {
			t.Errorf("the detail card does not show the destination %q:\n%s", client.Client, card)
		}
	}
	if !strings.Contains(card, "agentic-sdd preview") {
		t.Errorf("the detail card does not offer the next action:\n%s", card)
	}
}

// TestB2DetailSpacingIsReadable proves the detail pane separates labels from
// their values, never repeats a field label and keeps readable section
// spacing. It deliberately avoids a full-screen snapshot.
func TestB2DetailSpacingIsReadable(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/home/.claude/skills/agentic-sdd-router"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)
	item, ok := model.Selected()
	if !ok {
		t.Fatal("the catalog selected no item")
	}

	lines := b2CardLines(t, &model, b2DetailWidth)
	if len(lines) == 0 {
		t.Fatal("the detail card is empty")
	}

	// The rich fields must be rendered, otherwise there is nothing to space.
	for _, field := range []string{"Purpose", "When", "Example", "Caveat"} {
		value := b2ItemField(t, item, field)
		if !b2ShowsValue(strings.Join(lines, "\n"), value) {
			t.Errorf("the detail card does not show the %s text %q:\n%s", field, value, strings.Join(lines, "\n"))
		}
	}

	seen := map[string]int{}
	blankRun, longestBlankRun := 0, 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankRun++
			if blankRun > longestBlankRun {
				longestBlankRun = blankRun
			}
			continue
		}
		blankRun = 0

		match := b2LabelPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		rest := line[len(match[0]):]
		if strings.HasPrefix(rest, "//") {
			// A bare URL, not a label and a value.
			continue
		}
		if rest != "" && !strings.HasPrefix(rest, " ") {
			t.Errorf("line %d glues the label %q to its value: %q", i, match[1], line)
		}
		label := strings.ToLower(strings.TrimSpace(match[1]))
		seen[label]++
		if seen[label] > 1 {
			t.Errorf("the field label %q is repeated on line %d: %q", match[1], i, line)
		}
	}

	if longestBlankRun == 0 {
		t.Error("the detail card has no blank line between its sections; the spacing is not readable")
	}
	if longestBlankRun > 2 {
		t.Errorf("the detail card has %d consecutive blank lines; the spacing is not readable", longestBlankRun)
	}
	if strings.TrimSpace(lines[0]) == "" {
		t.Error("the detail card starts with a blank line")
	}
}

// TestB2DetailCardHonoursItsWidthBudget proves that the detail card keeps
// the width it is asked for. The card takes a width argument, so a value that
// is longer than that width must be wrapped instead of overflowing the pane.
func TestB2DetailCardHonoursItsWidthBudget(t *testing.T) {
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/"+strings.Repeat("long-segment/", 6)+"agentic-sdd-router"),
		},
	})
	model := tui.NewCatalogModel(tui.TUIConfig{Width: 120, Height: 24}, cat)

	for _, width := range []int{40, 60, 79, 80, 100, 120} {
		card := model.DetailCard(width)
		for i, line := range strings.Split(card, "\n") {
			if cells := ansi.StringWidth(line); cells > width {
				t.Errorf("the detail card at width %d has a line of %d cells (line %d): %q", width, cells, i, line)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Layout, paging and bounds.
// ---------------------------------------------------------------------------

// TestB2LongPathStaysReachableByPaging proves the richer detail card does
// not push a long destination path out of reach. The check reconstructs the
// wrapped lines, so a path broken at a wrap boundary still counts as
// reachable: only a truncated or rewritten path fails.
func TestB2LongPathStaysReachableByPaging(t *testing.T) {
	longPath := "/tmp/" + strings.Repeat("nested-directory/", 14) + "PATH-TAIL-MARKER"
	cat := b2CatalogWith(tui.CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
		Group:  tui.GroupCoreSkills,
		Clients: []tui.ClientStatus{
			b2Client("claude", skillsync.InventoryStatusMissing, longPath),
		},
	})

	app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 12, Color: false}, cat)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)
	next, _ = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)

	reached := ""
	for page := 0; page < 40; page++ {
		reached += b2ReachableText(app.View().Content)
		if strings.Contains(reached, longPath) {
			break
		}
		next, _ = app.Update(key(tea.KeyPgDown))
		app = next.(tui.App)
	}
	if !strings.Contains(reached, longPath) {
		t.Fatalf("the long destination path is not fully reachable by paging:\n%s", reached)
	}
	if !strings.Contains(reached, "PATH-TAIL-MARKER") {
		t.Fatalf("the tail of the long destination path was truncated:\n%s", reached)
	}

	lines := strings.Split(app.View().Content, "\n")
	if len(lines) > 12 {
		t.Errorf("the view has %d lines, want at most 12", len(lines))
	}
	for i, line := range lines {
		if width := ansi.StringWidth(line); width > 100 {
			t.Errorf("line %d is %d cells wide, want at most 100: %q", i, width, line)
		}
	}
}

// TestB2CellBoundsAtEveryBreakpoint proves the enriched catalog stays inside
// the documented cell bounds at every width and height breakpoint.
func TestB2CellBoundsAtEveryBreakpoint(t *testing.T) {
	cat := b2CatalogWith(
		tui.CatalogItem{
			ID:     "agentic-sdd-router",
			Kind:   skillsync.InventoryKindSkill,
			Origin: skillsync.InventoryOriginCore,
			Group:  tui.GroupCoreSkills,
			Clients: []tui.ClientStatus{
				b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/"+strings.Repeat("long-segment/", 8)+"agentic-sdd-router"),
			},
		},
		tui.CatalogItem{
			ID:     "agentic-sdd-plan",
			Kind:   skillsync.InventoryKindSkill,
			Origin: skillsync.InventoryOriginCore,
			Group:  tui.GroupCoreSkills,
			Clients: []tui.ClientStatus{
				b2Client("claude", skillsync.InventoryStatusMissing, "/tmp/home/.claude/skills/agentic-sdd-plan"),
			},
		},
	)

	for _, width := range []int{40, 60, 79, 80, 109, 110} {
		for _, height := range []int{8, 12, 24} {
			app := tui.NewApp(tui.TUIConfig{Width: width, Height: height, Color: false}, cat)
			next, _ := app.Update(runes("3"))
			app = next.(tui.App)
			next, _ = app.Update(key(tea.KeyEnter))
			app = next.(tui.App)

			b2AssertWithinBounds(t, app.View().Content, width, height, fmt.Sprintf("w=%d h=%d detail", width, height))

			nextHelp, _ := app.Update(runes("?"))
			appHelp := nextHelp.(tui.App)
			b2AssertWithinBounds(t, appHelp.View().Content, width, height, fmt.Sprintf("w=%d h=%d help", width, height))
		}
	}
}

// ---------------------------------------------------------------------------
// Read-only guarantees.
// ---------------------------------------------------------------------------

// TestB2ReadOnlyRunWritesNothing proves a complete read-only session leaves
// the inspected home and the skills source byte-identical, and does not need
// the network to render the inventory.
func TestB2ReadOnlyRunWritesNothing(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	skillDir := filepath.Join(repo, "skills-files", "agentic-sdd-b2-local")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# local skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	homeBefore := snapshotTree(t, home)
	repoBefore := snapshotTree(t, repo)

	// A deny-all proxy proves the run does not depend on any network access.
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("http_proxy", "http://127.0.0.1:1")
	t.Setenv("https_proxy", "http://127.0.0.1:1")

	cat, err := tui.LoadCatalog(tui.TUIConfig{Repo: repo, Home: home})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if len(cat.Items) == 0 {
		t.Fatal("the local source produced no catalog rows; the fixture assumption is stale")
	}

	app := tui.NewApp(tui.TUIConfig{Width: 100, Height: 24, Color: false, Repo: repo, Home: home}, cat)
	next, _ := app.Update(runes("3"))
	app = next.(tui.App)
	next, _ = app.Update(key(tea.KeyEnter))
	app = next.(tui.App)
	next, _ = app.Update(runes("j"))
	app = next.(tui.App)
	// B2 adds category browsing and search; neither may write.
	next, _ = app.Update(runes("]"))
	app = next.(tui.App)
	next, _ = app.Update(runes("/"))
	app = next.(tui.App)
	for _, ch := range "router" {
		next, _ = app.Update(runes(string(ch)))
		app = next.(tui.App)
	}
	next, _ = app.Update(key(tea.KeyEscape))
	app = next.(tui.App)
	next, _ = app.Update(runes("j"))
	app = next.(tui.App)
	_ = app.View().Content

	homeAfter := snapshotTree(t, home)
	repoAfter := snapshotTree(t, repo)

	if !reflect.DeepEqual(homeBefore, homeAfter) {
		t.Errorf("the read-only run changed the inspected home:\nbefore: %v\nafter:  %v", homeBefore, homeAfter)
	}
	if !reflect.DeepEqual(repoBefore, repoAfter) {
		t.Errorf("the read-only run changed the skills source:\nbefore: %v\nafter:  %v", repoBefore, repoAfter)
	}
}

// TestB2HomeShowsInventoryDerivedCounts proves the Home overview reports the
// counts of the loaded inventory instead of fixed numbers.
func TestB2HomeShowsInventoryDerivedCounts(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	app := tui.NewApp(tui.TUIConfig{Width: 110, Height: 30, Color: false}, cat)
	view := b2ReachableHomeText(app)
	for _, want := range []struct {
		noun  string
		count int
	}{
		{"skills", cat.SkillCount()},
		{"profiles", cat.ProfileCount()},
	} {
		if !b2ShowsCount(view, want.noun, want.count) {
			t.Errorf("the Home overview does not show the %s count %d:\n%s", want.noun, want.count, view)
		}
	}
}

// TestB2HomeOffersRecommendedInstallAction proves the Home overview shows a
// recommended setup action. The spec requires Home to visibly offer
// "Set up / Install Agentic-SDD"; the TUI still performs no write itself.
func TestB2HomeOffersRecommendedInstallAction(t *testing.T) {
	home := t.TempDir()
	cat, err := tui.LoadCatalog(tui.TUIConfig{Home: home, WithOpenCode: true})
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	app := tui.NewApp(tui.TUIConfig{Width: 110, Height: 30, Color: false}, cat)
	view := b2ReachableHomeText(app)
	if !strings.Contains(strings.ToLower(view), "install agentic-sdd") {
		t.Errorf("the Home overview does not offer the required setup action \"Set up / Install Agentic-SDD\":\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// Small shared helpers.
// ---------------------------------------------------------------------------

// b2AssertWithinBounds checks that a rendered view never exceeds the
// terminal's cell width or height.
func b2AssertWithinBounds(t *testing.T, view string, width, height int, context string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Errorf("%s: the view has %d lines, want at most %d", context, len(lines), height)
	}
	for i, line := range lines {
		if cell := ansi.StringWidth(line); cell > width {
			t.Errorf("%s: line %d is %d cells wide, want at most %d: %q", context, i, cell, width, line)
		}
	}
}

// b2ContainsID reports whether one of the items has the given ID.
func b2ContainsID(items []tui.CatalogItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

// b2ReachableHomeText walks the Home screen from top to bottom and returns
// every line a reader can reach, so an assertion does not depend on how much
// of the overview fits in one viewport.
func b2ReachableHomeText(app tui.App) string {
	var b strings.Builder
	for page := 0; page < 20; page++ {
		b.WriteString(app.View().Content)
		b.WriteString("\n")
		next, _ := app.Update(key(tea.KeyPgDown))
		updated, ok := next.(tui.App)
		if !ok {
			break
		}
		app = updated
	}
	return b.String()
}

// b2ShowsCount reports whether the rendered text states a labeled count. It
// accepts either order ("Skills: 19" or "19 skills") so the assertion is about
// the reported number, not about one label layout.
func b2ShowsCount(rendered, noun string, count int) bool {
	quoted := regexp.QuoteMeta(noun)
	labelFirst := regexp.MustCompile(fmt.Sprintf(`(?i)%s[^0-9]{0,8}\b%d\b`, quoted, count))
	valueFirst := regexp.MustCompile(fmt.Sprintf(`(?i)\b%d\s+%s`, count, quoted))
	return labelFirst.MatchString(rendered) || valueFirst.MatchString(rendered)
}

// b2ReachableText returns the text a reader can actually see in one view:
// the content pane of every panel row, with the layout padding removed, so
// wrapped values can be reassembled.
func b2ReachableText(view string) string {
	var b strings.Builder
	for _, line := range strings.Split(view, "\n") {
		if index := strings.LastIndex(line, "|"); index >= 0 {
			line = line[index+1:]
		}
		b.WriteString(strings.TrimSpace(line))
	}
	return b.String()
}

// b2DistinctiveWord returns the longest word of a text, used as a search
// query that is derived from the data instead of hardcoded.
func b2DistinctiveWord(text string) string {
	longest := ""
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.Trim(word, ".,;:`\"'()[]")
		if len(word) >= 6 && len(word) > len(longest) {
			longest = word
		}
	}
	return longest
}

// b2CategoryIDs lists the category IDs for failure messages.
func b2CategoryIDs(categories []b2Category) []string {
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		out = append(out, fmt.Sprintf("%s(%d)", category.ID, category.Count))
	}
	return out
}

// b2NormalizeSpace collapses every whitespace run into a single space so a
// value that the renderer wrapped over several lines can still be compared
// with the value itself.
func b2NormalizeSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// b2ShowsValue reports whether the rendered text contains the value. The
// comparison ignores line wrapping and indentation.
func b2ShowsValue(rendered, value string) bool {
	return strings.Contains(b2NormalizeSpace(rendered), b2NormalizeSpace(value))
}
