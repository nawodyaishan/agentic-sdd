package main

import "agentic-sdd/internal/appops"

// This file keeps the CLI's historical private teaching symbols as thin
// aliases over the single curated metadata source in internal/appops. The
// metadata is explanatory only: it never enumerates the install set, grants
// permissions, selects a model or approves work. Exact IDs still come from
// source inventory in internal/skillsync.
type teachingCard = appops.TeachingCard

const (
	teachingKindSkill   = appops.TeachingKindSkill
	teachingKindProfile = appops.TeachingKindProfile
	teachingKindConcept = appops.TeachingKindConcept

	teachingGroupCustom  = appops.TeachingGroupCustom
	teachingGroupConcept = appops.TeachingGroupConcept

	teachingSkillCaveat   = appops.TeachingSkillCaveat
	teachingProfileCaveat = appops.TeachingProfileCaveat
	teachingConceptCaveat = appops.TeachingConceptCaveat
)

// teachingRegistry is the private view of the shared curated registry. It must
// cover every shipped skill and every base/preset profile ID; the existing
// guide_catalog_test.go fixture compares it with the actual skill and pack
// directories so additions or removals force a review of the shared source.
// Source inventory remains authoritative for what is offered.
var teachingRegistry = appops.TeachingCards()

// teachingFor returns the curated card for a known ID of the requested kind.
// Unknown IDs get an honest generic card for that kind: no invented role,
// model, permission or runtime behavior.
func teachingFor(id, kind string) teachingCard {
	return appops.TeachingFor(id, kind)
}

func customTeachingCard(id, kind string) teachingCard {
	return appops.TeachingCustomCard(id, kind)
}

// teachingConcepts returns the key-concept cards in their stable display
// order. Callers get an independent slice.
func teachingConcepts() []teachingCard {
	return appops.TeachingConcepts()
}
