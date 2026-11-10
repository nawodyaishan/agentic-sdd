package main

import (
	"testing"

	"agentic-sdd/internal/appops"
)

// These tests guard the seam between the CLI's historical private teaching
// symbols and the single curated metadata source in internal/appops. They are
// additive: the pre-existing catalog fixture keeps its own assertions.

func TestTeachingRegistryIsSharedMetadata(t *testing.T) {
	shared := appops.TeachingCards()
	if len(teachingRegistry) != len(shared) {
		t.Fatalf("private teachingRegistry has %d cards, shared source has %d", len(teachingRegistry), len(shared))
	}
	for i := range shared {
		if teachingRegistry[i] != shared[i] {
			t.Errorf("card %d diverged from the shared source:\n private: %+v\n shared:  %+v", i, teachingRegistry[i], shared[i])
		}
	}
}

func TestTeachingWrappersMatchSharedAPI(t *testing.T) {
	for _, tc := range []struct{ id, kind string }{
		{"agentic-sdd-router", teachingKindSkill},
		{"sdd-implement-1", teachingKindProfile},
		{"preview-apply", teachingKindConcept},
		{"local-unlisted-item", teachingKindSkill},
	} {
		got, want := teachingFor(tc.id, tc.kind), appops.TeachingFor(tc.id, tc.kind)
		if got != want {
			t.Errorf("teachingFor(%q, %q) = %+v, want shared %+v", tc.id, tc.kind, got, want)
		}
		if got.DisplayName != want.DisplayName || got.DisplayName == "" {
			t.Errorf("teachingFor(%q, %q) display name = %q, want shared %q", tc.id, tc.kind, got.DisplayName, want.DisplayName)
		}
	}

	concepts := teachingConcepts()
	sharedConcepts := appops.TeachingConcepts()
	if len(concepts) != len(sharedConcepts) {
		t.Fatalf("private concepts = %d, shared concepts = %d", len(concepts), len(sharedConcepts))
	}
	for i := range sharedConcepts {
		if concepts[i] != sharedConcepts[i] {
			t.Errorf("concept %d diverged from the shared source: %+v", i, concepts[i])
		}
	}
}

func TestTeachingCustomWrapperKeepsNeutralFallback(t *testing.T) {
	for _, kind := range []string{teachingKindSkill, teachingKindProfile, teachingKindConcept} {
		card := customTeachingCard("local-unlisted-item", kind)
		if card.ID != "local-unlisted-item" || card.Kind != kind {
			t.Errorf("custom card for %s lost its identity: %+v", kind, card)
		}
		if card.Group != teachingGroupCustom {
			t.Errorf("custom card for %s group = %q, want %q", kind, card.Group, teachingGroupCustom)
		}
		if card.DisplayName != "local-unlisted-item" {
			t.Errorf("custom card for %s display name = %q, want the neutral exact ID", kind, card.DisplayName)
		}
	}

	// The private helper still bypasses curated metadata for a known ID.
	if card := customTeachingCard("agentic-sdd-router", teachingKindSkill); card.Group != teachingGroupCustom {
		t.Errorf("customTeachingCard borrowed a curated role: %+v", card)
	}
}
