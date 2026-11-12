package tui

// W3-D coverage for the Fleet Manager help topic and the optional onboarding
// hint. The assertions are presentation-only: no source file, backup, home,
// client or agent is touched, and no command is executed.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// w3dFlatten turns a wrapped render into one space-separated string so copy
// assertions survive line wrapping at any width.
func w3dFlatten(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func w3dActivity() ActivityModel {
	return NewActivityModel(ActivityConfig{
		Theme:  NewTheme(false),
		Width:  80,
		Height: 30,
		Metadata: AboutMetadata{
			Version: "dev-feature-015",
			Commit:  "1234567",
			Date:    "2026-11-12",
			Author:  BrandAuthor,
			URL:     "https://github.com/nawodyaishan/agentic-sdd#readme",
		},
	})
}

// TestW3DFleetHelpTopicIsListed proves the topic is discoverable without
// disturbing the existing section order or keys.
func TestW3DFleetHelpTopicIsListed(t *testing.T) {
	previous := []struct {
		section HelpSection
		key     string
	}{
		{HelpSectionAll, "0"},
		{HelpSectionActivity, "1"},
		{HelpSectionQuickStart, "2"},
		{HelpSectionShortcuts, "3"},
		{HelpSectionCommands, "4"},
		{HelpSectionTroubleshooting, "5"},
		{HelpSectionDiagnostics, "6"},
		{HelpSectionAbout, "7"},
		{HelpSectionPreferences, "8"},
	}

	sections := AllHelpSections()
	if len(sections) != len(previous)+1 {
		t.Fatalf("help sections = %d, want %d", len(sections), len(previous)+1)
	}
	for i, want := range previous {
		if sections[i].Section != want.section || sections[i].Key != want.key {
			t.Fatalf("help section %d changed to %+v, want section %v key %q", i, sections[i], want.section, want.key)
		}
	}
	fleet := sections[len(sections)-1]
	if fleet.Section != HelpSectionFleet {
		t.Fatalf("last help section = %+v, want HelpSectionFleet", fleet)
	}
	if fleet.Key != "9" || fleet.Title != "Fleet Manager" {
		t.Fatalf("fleet help metadata = %+v, want key 9 and title Fleet Manager", fleet)
	}
}

// TestW3DFleetHelpTopicStatesExactTruths proves the rendered topic carries the
// approved FR-22 wording: explicit source, staged model change, the separate
// source save, previewed deployment, default-No, managed restore and the
// capability distinctions.
func TestW3DFleetHelpTopicStatesExactTruths(t *testing.T) {
	m := w3dActivity()
	m.SelectSection(HelpSectionFleet)
	text := w3dFlatten(strings.Join(m.Lines(80), " "))

	for _, want := range []string{
		"Fleet Manager",
		"explicit path",
		"fleet.json",
		"press s on Overview",
		"m to pick a model",
		"separate, explicit write",
		"not available in this TUI build",
		"Review (c)",
		"previewing writes nothing",
		"needs an explicit y",
		"Enter, n or Esc means no",
		"managed backup",
		"Recovery (e)",
		"Role vs instance",
		"Personal source vs rendered file vs installed agent",
		"not selected primary",
		"not authenticated",
		"not runtime-verified",
		"Preview only",
		"No subscription, price, entitlement or quota",
		"No automatic adoption",
		"Nothing is committed, pushed",
		"docs/fleet-manager-tui.md",
		"agentic-sdd help fleet",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Fleet Manager help is missing %q:\n%s", want, text)
		}
	}

	// No home scan, inferred path or private path example is exposed.
	for _, forbidden := range []string{"/home/", "~/", "C:\\Users", "scan your home"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("Fleet Manager help contains %q; it must not expose or imply a private path", forbidden)
		}
	}
}

// TestW3DFleetHelpFitsReviewedWidths proves the topic wraps instead of
// overflowing at the reviewed layouts.
func TestW3DFleetHelpFitsReviewedWidths(t *testing.T) {
	for _, width := range []int{110, 60, 40} {
		m := w3dActivity()
		m.SelectSection(HelpSectionFleet)
		for _, line := range m.Lines(width) {
			if got := ansi.StringWidth(line); got > width {
				t.Fatalf("width %d: fleet help line width %d exceeds %d: %q", width, got, width, line)
			}
		}
		if len(m.Lines(width)) == 0 {
			t.Fatalf("width %d: fleet help rendered no lines", width)
		}
	}
}

// TestW3DFleetHelpIsReachableFromQuickStart proves Quick start points at the
// topic and the Fleet surface without claiming it is live.
func TestW3DFleetHelpIsReachableFromQuickStart(t *testing.T) {
	m := w3dActivity()
	m.SelectSection(HelpSectionQuickStart)
	text := w3dFlatten(strings.Join(m.Lines(80), " "))
	if !strings.Contains(text, "Fleet Manager") {
		t.Fatalf("Quick start does not point at Fleet Manager:\n%s", text)
	}
	if !strings.Contains(text, "preview only in this build") {
		t.Fatalf("Quick start does not scope Fleet to preview only:\n%s", text)
	}
}

// TestW3DFleetHintIsOptionalAndSkippable proves the welcome-step hint is
// present, wrapped, and never blocks the tour's skip exit. The finish step must
// stay free of the Fleet surface so the deferred-surface contract is intact.
func TestW3DFleetHintIsOptionalAndSkippable(t *testing.T) {
	welcome := NewOnboardingModel(TUIConfig{Width: 80, Height: 24}, NewOnboardingSummaryFromSetup(contrastCatalog(), t34Setup(t)))
	text := w3dFlatten(welcome.Render(80))
	if !strings.Contains(text, "Fleet models (optional)") {
		t.Fatalf("welcome step is missing the optional Fleet hint:\n%s", text)
	}
	if !strings.Contains(text, w3dFlatten(FleetHint)) {
		t.Fatalf("welcome step does not show the exact Fleet hint:\n%s", text)
	}
	if strings.Contains(text, "/home/") || strings.Contains(text, "~/") {
		t.Fatalf("Fleet hint exposes a private path example:\n%s", text)
	}

	for _, width := range []int{40, 60, 80, 110} {
		for _, line := range welcome.Lines(width) {
			if got := ansi.StringWidth(line); got > width {
				t.Fatalf("width %d: Fleet hint line width %d exceeds %d: %q", width, got, width, line)
			}
		}
	}

	skipped, cmd := t34Update(t, welcome, t34Key(tea.KeyEscape))
	if cmd != nil {
		t.Fatal("cancelling the tour emitted a command")
	}
	if !skipped.Done() || skipped.Outcome() != OnboardingCancelled {
		t.Fatalf("cancelling left outcome %v done=%v, want cancelled", skipped.Outcome(), skipped.Done())
	}

	finish := welcome
	for i := 0; i < 3; i++ {
		finish = t34Next(t, finish)
	}
	if strings.Contains(finish.Render(80), "Fleet") {
		t.Fatal("finish step still exposes the Fleet surface")
	}
}
