package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"agentic-sdd/internal/skillsync"
)

func TestLibrary_RowFormatting(t *testing.T) {
	// Curated item shows readable name paired with exact ID, purpose-first description
	curated := CatalogItem{
		ID:     "agentic-sdd-router",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
	}
	enrichItem(&curated)

	title := FormatCatalogRowTitle(curated)
	if title != "SDD Router (agentic-sdd-router)" {
		t.Errorf("FormatCatalogRowTitle(curated) = %q, want 'SDD Router (agentic-sdd-router)'", title)
	}

	desc := FormatCatalogRowDescription(curated)
	if !strings.HasPrefix(desc, "Route a request through combined spec, plan and tasks approval") {
		t.Errorf("FormatCatalogRowDescription(curated) = %q, want purpose-first prefix", desc)
	}
	if !strings.Contains(desc, "clients installed") {
		t.Errorf("FormatCatalogRowDescription(curated) missing install count suffix: %q", desc)
	}

	// Uncurated item falls back neutrally to exact ID
	uncurated := CatalogItem{
		ID:     "custom-user-tool",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
	}
	enrichItem(&uncurated)

	uncuratedTitle := FormatCatalogRowTitle(uncurated)
	if uncuratedTitle != "custom-user-tool" {
		t.Errorf("FormatCatalogRowTitle(uncurated) = %q, want exact ID 'custom-user-tool'", uncuratedTitle)
	}
}

func TestLibrary_TaskSearchInFilterValue(t *testing.T) {
	// Curated skill when clause includes trigger tasks like "failing tests" or "intermittent"
	debugSkill := CatalogItem{
		ID:     "agentic-sdd-debugging",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
	}
	enrichItem(&debugSkill)

	fv := debugSkill.FilterValue()
	for _, term := range []string{"failing tests", "unexpected behavior", "intermittent"} {
		if !strings.Contains(fv, term) {
			t.Errorf("FilterValue() missing task trigger term %q:\n%s", term, fv)
		}
	}
}

func TestLibrary_RouterRecommendationAndStatePreservation(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{ID: "agentic-sdd-spec", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "agentic-sdd-router", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
			{ID: "agentic-sdd-plan", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginCore},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	// Initial state
	m.SetFilter("plan")
	if m.list.FilterValue() != "plan" {
		t.Fatalf("expected filter to be 'plan', got %q", m.list.FilterValue())
	}
	m.SetCategory(CategoryCore)
	if m.ActiveCategory() != CategoryCore {
		t.Fatalf("expected category to be %q, got %q", CategoryCore, m.ActiveCategory())
	}

	// Trigger Router recommendation via 's'
	updated, _ := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	m = updated.(CatalogModel)
	if !m.InDetail() {
		t.Fatal("expected 's' to open Router detail card")
	}

	card := m.DetailCard(80)
	if !strings.Contains(card, "SDD Router") || !strings.Contains(card, "agentic-sdd-router") {
		t.Fatalf("DetailCard does not show SDD Router:\n%s", card)
	}

	// Resize while in detail view: state must remain preserved
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(CatalogModel)
	if !m.InDetail() {
		t.Fatal("resize closed detail card unexpectedly")
	}

	// Exit detail via 'b' or Esc
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m = updated.(CatalogModel)
	if m.InDetail() {
		t.Fatal("expected 'b' to exit detail view")
	}

	// Verified facts: query and category are intact
	if m.list.FilterValue() != "plan" {
		t.Errorf("filter value changed after detail exit: got %q, want 'plan'", m.list.FilterValue())
	}
	if m.ActiveCategory() != CategoryCore {
		t.Errorf("category changed after detail exit: got %q, want %q", m.ActiveCategory(), CategoryCore)
	}
}

func TestLibrary_DetailCardSectionsAndUnknownFallbacks(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{
			{
				ID:     "agentic-sdd-router",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
				Clients: []ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: "/tmp/claude/skills/router"},
				},
			},
			{
				ID:     "custom-blank-skill",
				Kind:   skillsync.InventoryKindSkill,
				Origin: skillsync.InventoryOriginCore,
			},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 80, Height: 24}, cat)

	card := m.DetailCard(80)
	for _, section := range []string{
		"Why use it",
		"When to use",
		"Invocation",
		"Caveat",
		"Destination status:",
		"Installed state",
		"Advanced files:",
		"Next action:",
	} {
		if !strings.Contains(card, section) {
			t.Errorf("DetailCard missing section %q:\n%s", section, card)
		}
	}

	// Check neutral unknown fallbacks on blank item
	blankItem := CatalogItem{
		ID:     "unknown-custom-tool",
		Kind:   skillsync.InventoryKindSkill,
		Origin: skillsync.InventoryOriginCore,
	}
	m.displayItem = &blankItem
	blankCard := m.DetailCard(80)

	if !strings.Contains(blankCard, "No destination entries reported") {
		t.Errorf("blank card missing neutral destination fallback:\n%s", blankCard)
	}
	if !strings.Contains(blankCard, "No destination file paths recorded") {
		t.Errorf("blank card missing neutral paths fallback:\n%s", blankCard)
	}
}

// TestLibrary_DetailHierarchyKeepsPathsInAdvancedFiles pins the review fix:
// the early status summary names each client and its state without duplicating
// destination paths, and the complete path appears only in Advanced files.
func TestLibrary_DetailHierarchyKeepsPathsInAdvancedFiles(t *testing.T) {
	path := "/tmp/home/.claude/skills/router"
	item := CatalogItem{
		ID:      "agentic-sdd-router",
		Name:    "SDD Router",
		Kind:    skillsync.InventoryKindSkill,
		Origin:  skillsync.InventoryOriginCore,
		Group:   GroupCoreSkills,
		Purpose: "Route a request through the SDD workflow.",
		When:    "Use when starting a feature.",
		Example: "agentic-sdd-router",
		Caveat:  "Advisory only.",
		Clients: []ClientStatus{
			{Client: "claude", Status: skillsync.InventoryStatusMatchesSource, Path: path},
			{Client: "codex", Status: skillsync.InventoryStatusMissing, Path: "/tmp/home/.codex/skills/router"},
		},
	}
	m := NewCatalogModel(TUIConfig{Width: 100, Height: 30}, Catalog{Items: []CatalogItem{item}})
	card := m.DetailCard(80)

	status := strings.Index(card, "Destination status:")
	purpose := strings.Index(card, "Purpose:")
	advanced := strings.Index(card, "Advanced files:")
	if status < 0 || purpose < 0 || advanced < 0 {
		t.Fatalf("detail card lost a required section:\n%s", card)
	}
	if !(status < purpose && purpose < advanced) {
		t.Fatalf("detail hierarchy status=%d purpose=%d advanced=%d:\n%s", status, purpose, advanced, card)
	}
	if strings.Contains(card[:advanced], path) {
		t.Fatalf("destination path appears before Advanced files:\n%s", card)
	}
	if got := strings.Count(card, path); got != 1 {
		t.Fatalf("destination path appears %d times, want exactly once in Advanced files:\n%s", got, card)
	}
	// The concise early summary still names both clients and their states.
	summary := card[status:advanced]
	for _, want := range []string{"claude", "codex", StatusBadge(skillsync.InventoryStatusMatchesSource), StatusBadge(skillsync.InventoryStatusMissing)} {
		if !strings.Contains(summary, want) {
			t.Errorf("early destination summary missing %q:\n%s", want, summary)
		}
	}
}

func TestActivity_StructuredFactsNoProseScraping(t *testing.T) {
	log := LogSummary{
		Name:    "20261110T150405.000000000Z-apply.log",
		Path:    "/tmp/test/20261110T150405.000000000Z-apply.log",
		Size:    2048,
		ModTime: time.Now(),
	}

	facts := ParseLogOperationFacts(log)
	if facts.Operation != "apply" {
		t.Errorf("Operation = %q, want 'apply'", facts.Operation)
	}
	if facts.RawTimestamp != "20261110T150405.000000000Z" {
		t.Errorf("RawTimestamp = %q, want '20261110T150405.000000000Z'", facts.RawTimestamp)
	}
	if !strings.Contains(facts.Timestamp, "Nov 10, 2026") {
		t.Errorf("Timestamp = %q, want readable date containing 'Nov 10, 2026'", facts.Timestamp)
	}
	// Unknown fields must remain strictly unknown, never guessed from log prose
	if facts.Outcome != "unknown (not recorded in structured log index)" {
		t.Errorf("Outcome = %q, want unknown notice", facts.Outcome)
	}
	if facts.ChangedCount != "unknown" || facts.SkippedCount != "unknown" || facts.FailedCount != "unknown" {
		t.Errorf("Counts must be 'unknown', got changed=%q skipped=%q failed=%q",
			facts.ChangedCount, facts.SkippedCount, facts.FailedCount)
	}
}

func TestActivity_HelpSectionsSelectionAndOrganization(t *testing.T) {
	m := NewActivityModel(ActivityConfig{
		Theme: NewTheme(false),
		Width: 80,
		Metadata: AboutMetadata{
			Version: "dev-feature-013",
			Commit:  "1234567",
			Date:    "2026-11-10",
			Author:  BrandAuthor,
			URL:     "https://github.com/nawodyaishan/agentic-sdd#readme",
		},
	})

	// Initial section is All
	if m.ActiveSection() != HelpSectionAll {
		t.Fatalf("expected initial section to be HelpSectionAll, got %v", m.ActiveSection())
	}

	// Commands section includes excluded operations marked non-executable
	m.SelectSection(HelpSectionCommands)
	cmdLines := strings.Join(m.Lines(80), "\n")
	if !strings.Contains(cmdLines, "Commands & CLI Reference") {
		t.Errorf("Commands section missing title:\n%s", cmdLines)
	}
	if !strings.Contains(cmdLines, "Excluded experimental operations") {
		t.Errorf("Commands section missing excluded operations header:\n%s", cmdLines)
	}
	if !strings.Contains(cmdLines, "non-executable") {
		t.Errorf("Excluded operations must be marked non-executable:\n%s", cmdLines)
	}

	// About section with dev build shows Development build label leading to Diagnostics
	m.SelectSection(HelpSectionAbout)
	aboutLines := strings.Join(m.Lines(80), "\n")
	if !strings.Contains(aboutLines, "Development build") {
		t.Errorf("About section missing 'Development build' label for dev version:\n%s", aboutLines)
	}
	if !strings.Contains(aboutLines, "Diagnostics") {
		t.Errorf("About section must direct user to Diagnostics for exact metadata:\n%s", aboutLines)
	}

	// Diagnostics section exposes exact commit and compiler metadata
	m.SelectSection(HelpSectionDiagnostics)
	diagLines := strings.Join(m.Lines(80), "\n")
	if !strings.Contains(diagLines, "Diagnostics & Environment") {
		t.Errorf("Diagnostics section missing title:\n%s", diagLines)
	}
	if !strings.Contains(diagLines, "1234567") {
		t.Errorf("Diagnostics section missing exact commit SHA:\n%s", diagLines)
	}

	// Troubleshooting section
	m.SelectSection(HelpSectionTroubleshooting)
	tbLines := strings.Join(m.Lines(80), "\n")
	if !strings.Contains(tbLines, "Troubleshooting") || !strings.Contains(tbLines, "Update check offline") {
		t.Errorf("Troubleshooting section missing expected topics:\n%s", tbLines)
	}

	// Session preferences section
	m.SelectSection(HelpSectionPreferences)
	prefLines := strings.Join(m.Lines(80), "\n")
	if !strings.Contains(prefLines, "Session Preferences") || !strings.Contains(prefLines, "agentic-dark") {
		t.Errorf("Preferences section missing agentic-dark or controls:\n%s", prefLines)
	}

	// Cycling sections via 't' and 'T'
	m.SelectSection(HelpSectionAll)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m = updated.(ActivityModel)
	if m.ActiveSection() != HelpSectionActivity {
		t.Errorf("after 't' expected HelpSectionActivity, got %v", m.ActiveSection())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	m = updated.(ActivityModel)
	if m.ActiveSection() != HelpSectionAll {
		t.Errorf("after 'T' expected HelpSectionAll, got %v", m.ActiveSection())
	}
}
