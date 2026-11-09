package main

// teachingCard is curated, non-executable learning metadata for the optional
// explorer. It is explanatory only: it never enumerates the install set,
// grants permissions, selects a model or approves work. Exact IDs still come
// from source inventory in internal/skillsync.
type teachingCard struct {
	ID      string
	Kind    string
	Group   string
	Purpose string
	When    string
	Example string
	Caveat  string
}

const (
	teachingKindSkill   = "skill"
	teachingKindProfile = "profile"
	teachingKindConcept = "concept"

	teachingGroupCustom  = "Custom"
	teachingGroupConcept = "Key concepts"
)

// Caveats separate a template's declared intent from enforcement. They render
// as one scannable safety badge per kind, so no card can imply a sandbox,
// model access, permission grant or approval that the tool does not provide.
// The extended explanation lives in the key-concept cards instead.
const (
	teachingSkillCaveat   = "Advisory guidance · No automated agent execution · Human approval required"
	teachingProfileCaveat = "Template intent only · No automated agent execution · Human approval required"
	teachingConceptCaveat = "Teaching concept · No automated agent execution · Human approval required"
)

// teachingRegistry is the single curated map. It must cover every shipped
// skill and every base/preset profile ID; guide_catalog_test.go compares it
// with the actual skill and pack directories so additions or removals force a
// review here. Source inventory remains authoritative for what is offered.
var teachingRegistry = []teachingCard{
	// Core skills shipped in skills-files/, grouped by purpose.
	{
		ID: "agentic-sdd-router", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Route a request through combined spec, plan and tasks approval into bounded, approved work.",
		When:    "You describe an outcome or a resumption and want the right SDD entry point chosen.",
		Example: "Use `agentic-sdd-router` to inspect this repository and draft the three documents for my combined approval.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-spec", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Draft the feature spec as the first of three documents for one combined approval.",
		When:    "A feature outcome is agreed and its scope and acceptance need a written draft.",
		Example: "Use `agentic-sdd-spec` to turn this goal into a draft spec and stop before planning.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-plan", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Draft the feature plan and tasks from the draft spec for one combined approval.",
		When:    "A draft spec exists and the work needs boundaries, dependencies and batches.",
		Example: "Use `agentic-sdd-plan` to draft the plan and tasks from my draft spec and stop for approval.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-tasks", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Order feature tasks into explicit batches for the same combined approval.",
		When:    "The plan exists and execution order, dependencies and batch boundaries need to be explicit.",
		Example: "Use `agentic-sdd-tasks` to split this plan into ordered batches and stop for approval.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-bootstrap", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Establish minimal SDD pointers, feature conventions and the combined-approval location in an existing repository.",
		When:    "A repository has SDD rules to preserve but no agreed place for spec, plan, tasks or approval.",
		Example: "Use `agentic-sdd-bootstrap` to establish the minimal SDD pointers here without changing product code.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-grill-me", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Challenge a plan or decision through focused questions.",
		When:    "You want a deliberate stress test before relying on a plan or direction.",
		Example: "Use `agentic-sdd-grill-me` to challenge this plan and show me where it is weakest.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-grilling", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Resolve a consequential or blocked decision by asking the questions that actually matter.",
		When:    "An unresolved behavior, scope or risk decision prevents authorized work.",
		Example: "Use `agentic-sdd-grilling` to clarify this decision before we implement anything.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-questionnaire", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Prepare recipient-specific discovery questions, or reconcile returned answers with source requirements.",
		When:    "Another person holds knowledge you need and the questions must match their role.",
		Example: "Use `agentic-sdd-questionnaire` to prepare focused questions for the person who owns this system.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-research-spec", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Resolve one named fact from primary sources and return it to the invoking stage.",
		When:    "A concrete unknown blocks drafting, implementation or review.",
		Example: "Use `agentic-sdd-research-spec` to resolve this named fact and return the evidence with its sources.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-implement", Kind: teachingKindSkill, Group: "Implement/debug",
		Purpose: "Implement one approved batch or direct fix and verify the result.",
		When:    "A batch or bounded fix is approved and you want focused implementation rather than a new plan.",
		Example: "Use `agentic-sdd-implement` for this approved batch, keep edits inside its scope and report the checks.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-debugging", Kind: teachingKindSkill, Group: "Implement/debug",
		Purpose: "Diagnose errors, failing tests and unexpected behavior, separating evidence from hypotheses.",
		When:    "Something is broken or intermittent and the cause is not yet established.",
		Example: "Use `agentic-sdd-debugging` to find the cause of this failing test and report evidence before changing code.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-drift-retro", Kind: teachingKindSkill, Group: "Implement/debug",
		Purpose: "Bring material drift back to canonical sources and capture the lessons worth keeping.",
		When:    "Implementation or documentation has diverged from the approved record and the difference matters.",
		Example: "Use `agentic-sdd-drift-retro` to reconcile this drift with the approved documents and record what changed.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-verification-review", Kind: teachingKindSkill, Group: "Verify/review",
		Purpose: "Review completed work against requirements and evidence while keeping review-only requests non-mutating.",
		When:    "A batch or fix claims completion and needs an evidence-based check.",
		Example: "Use `agentic-sdd-verification-review` to check this finished batch against its acceptance criteria.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-architecture-review", Kind: teachingKindSkill, Group: "Verify/review",
		Purpose: "Assess a consequential feature design and return a technical architecture verdict.",
		When:    "A design's cost, security or operational risk is hard to reverse and deserves a second look.",
		Example: "Use `agentic-sdd-architecture-review` to challenge this design and return a verdict with its risks.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-teach", Kind: teachingKindSkill, Group: "Learn/reflect",
		Purpose: "Explain an explicitly requested concept, or maintain a separately requested learning workspace.",
		When:    "You want to understand an idea rather than advance a feature.",
		Example: "Use `agentic-sdd-teach` to explain how combined approval works in this project.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-wait-what", Kind: teachingKindSkill, Group: "Learn/reflect",
		Purpose: "Re-explain the current proposal, progress or decision in plain language.",
		When:    "The conversation has become dense and you want the current position restated simply.",
		Example: "Use `agentic-sdd-wait-what` to restate where this proposal stands in plain language.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-handoff", Kind: teachingKindSkill, Group: "Learn/reflect",
		Purpose: "Prepare a sanitized, canonical-first transfer record for the next session.",
		When:    "Work pauses or moves to another agent and the next session needs the state and open decisions.",
		Example: "Use `agentic-sdd-handoff` to write a sanitized handover for the next session.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-writing-for-agents", Kind: teachingKindSkill, Group: "Learn/reflect",
		Purpose: "Write or maintain agent-consumed skills, instruction files and references.",
		When:    "Agent instructions need precise triggers, progressive detail and observable completion.",
		Example: "Use `agentic-sdd-writing-for-agents` to tighten this instruction file for another agent.",
		Caveat:  teachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-orchestrate", Kind: teachingKindSkill, Group: "Scope/plan",
		Purpose: "Classify a request, delegate only bounded work and reconcile what comes back.",
		When:    "You explicitly want the optional orchestration experience rather than a direct answer.",
		Example: "Use `agentic-sdd-orchestrate` to classify this request and propose the smallest safe delegation.",
		Caveat:  teachingSkillCaveat,
	},

	// Base OpenCode profiles shipped in opencode-pack/agents/.
	{
		ID: "sdd-orchestrator", Kind: teachingKindProfile, Group: "Main communicator",
		Purpose: "Hold the human conversation, classify the request and supervise bounded delegation.",
		When:    "You want one primary agent that keeps approvals and reconciles specialist work.",
		Example: "Use the `sdd-orchestrator` role for this session and keep approvals with me.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-debug-diagnose", Kind: teachingKindProfile, Group: "Read-only investigation",
		Purpose: "Diagnose an error or failing test and separate evidence from hypotheses.",
		When:    "You need the cause established before anyone edits code.",
		Example: "Use `sdd-debug-diagnose` to establish this failure's cause and return evidence before any fix.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-test-design", Kind: teachingKindProfile, Group: "Test assistance",
		Purpose: "Design focused test cases, fixtures and commands without editing or running them.",
		When:    "A change needs a test plan before implementation or review.",
		Example: "Use `sdd-test-design` to propose focused tests and fixtures for this change.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-research-librarian", Kind: teachingKindProfile, Group: "Read-only investigation",
		Purpose: "Resolve a named technical fact from primary sources without repository writes.",
		When:    "A concrete external fact blocks a decision or draft.",
		Example: "Use `sdd-research-librarian` to resolve this API question from primary sources.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-docs-worker", Kind: teachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Write or update documentation in an explicitly assigned path set.",
		When:    "Documentation changes are authorized and their paths are known.",
		Example: "Use `sdd-docs-worker` for the assigned documentation paths in this batch.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-architecture-advisor", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Advise on architecture, boundaries and operational risk without approving or implementing.",
		When:    "You want a second architectural opinion before committing to a direction.",
		Example: "Use `sdd-architecture-advisor` to review this design's boundaries and operational risk.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-visual-inspector", Kind: teachingKindProfile, Group: "Read-only investigation",
		Purpose: "Describe images, screenshots and rendered pages from disk for an agent that cannot see them.",
		When:    "Visual evidence must be transcribed accurately without acting on it.",
		Example: "Use `sdd-visual-inspector` to describe this screenshot from disk.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-openai-architect", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Provide an optional external architecture and specification review in advisory mode.",
		When:    "A high-impact design wants another independent advisory pass.",
		Example: "Use `sdd-openai-architect` for an independent architecture opinion on this design.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-openai-implement", Kind: teachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Implement approved engineering tasks and difficult refactoring inside the assigned scope.",
		When:    "A complex approved change needs an additional implementation profile.",
		Example: "Use `sdd-openai-implement` for this approved task inside its stated scope.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-architecture-review-9r", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Return a technical architecture verdict on risk without replacing human combined approval.",
		When:    "A consequential design needs an independent advisory verdict.",
		Example: "Use `sdd-architecture-review-9r` for an architecture verdict on this design's risks.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-code-review-9r", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Review SDD work and verification evidence independently.",
		When:    "Completed work needs an independent review of code and evidence.",
		Example: "Use `sdd-code-review-9r` to review this completed batch and its evidence.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-review-verification-1", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Review explicitly delegated SDD work and verification evidence independently.",
		When:    "A reviewer different from the implementer is wanted for a delegated batch.",
		Example: "Use `sdd-review-verification-1` to review this delegated batch independently.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-review-verification-2", Kind: teachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Provide a supplementary review that is not the sole consequential reviewer.",
		When:    "A second opinion is useful but cannot carry the final verdict alone.",
		Example: "Use `sdd-review-verification-2` for a supplementary opinion beside the main verdict.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-lightweight", Kind: teachingKindProfile, Group: "Read-only investigation",
		Purpose: "Handle bounded discovery and source-backed summaries for a larger session.",
		When:    "Several searches or reads would otherwise consume the main agent's context.",
		Example: "Use `sdd-lightweight` for this bounded lookup so the main session keeps its context.",
		Caveat:  teachingProfileCaveat,
	},

	// Optional Free-first preset profiles shipped in opencode-pack/presets/.
	{
		ID: "sdd-implement-1", Kind: teachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Routine implementation of one delegated batch or bounded fix.",
		When:    "A straightforward batch or bounded fix has been delegated and the preset matches the work.",
		Example: "Use `sdd-implement-1` for this approved routine batch and stop at its boundary.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-implement-2", Kind: teachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Complex or ambiguous implementation of one delegated batch.",
		When:    "A delegated batch needs more context or reasoning than routine work.",
		Example: "Use `sdd-implement-2` for this complex approved batch and report any uncertainty.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-test-implement-1", Kind: teachingKindProfile, Group: "Test assistance",
		Purpose: "Focused test implementation for a delegated batch.",
		When:    "Test work is explicitly delegated and routine.",
		Example: "Use `sdd-test-implement-1` for the focused tests in this batch.",
		Caveat:  teachingProfileCaveat,
	},
	{
		ID: "sdd-test-implement-2", Kind: teachingKindProfile, Group: "Test assistance",
		Purpose: "Focused test implementation for a more complex delegated batch.",
		When:    "A delegated batch needs additional test implementation effort.",
		Example: "Use `sdd-test-implement-2` for the focused tests in this complex batch.",
		Caveat:  teachingProfileCaveat,
	},

	// Key concepts, declared last so teachingConcepts keeps a stable order.
	{
		ID: "spec-plan-tasks", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "Three documents drafted together: what to build, how to build it and in what order.",
		When:    "Starting a feature that needs scope, design and ordered batches before implementation.",
		Example: "Use one combined approval for `spec.md`, `plan.md` and `tasks.md` before any implementation.",
		Caveat:  teachingConceptCaveat,
	},
	{
		ID: "skill-profile", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "A skill teaches reusable workflow steps; a profile declares an agent's role, focus and tool boundaries.",
		When:    "You are unsure whether a task needs a workflow or a delegated role.",
		Example: "Use a skill for reusable workflow steps and a profile for a delegated role.",
		Caveat:  teachingConceptCaveat,
	},
	{
		ID: "main-subagent", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "The main agent keeps the conversation and authority; a subagent receives one bounded brief and returns evidence.",
		When:    "Deciding whether work should stay in the main session or be delegated.",
		Example: "Use the main agent for the conversation and a subagent for one bounded brief.",
		Caveat:  teachingConceptCaveat,
	},
	{
		ID: "preview-apply", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "Preview shows the plan without writing; apply performs the reviewed changes after explicit confirmation.",
		When:    "About to install or change files and you want to see the effect first.",
		Example: "Use preview to see the plan and apply only after my explicit confirmation.",
		Caveat:  teachingConceptCaveat,
	},
	{
		ID: "free-first", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "Optional profile presets intended for provider free tiers, with limits and availability beyond this tool's control.",
		When:    "Considering the optional presets and wanting to understand their intent and limits.",
		Example: "Use the Free-first presets only after reading their intent and limits.",
		Caveat:  teachingConceptCaveat,
	},
	{
		ID: "human-approval", Kind: teachingKindConcept, Group: teachingGroupConcept,
		Purpose: "A human grants the combined spec, plan and tasks approval and authorizes each batch; no agent can substitute.",
		When:    "Clarifying who decides scope or when implementation may start.",
		Example: "Use the combined approval gate: no batch starts without my explicit authorization.",
		Caveat:  teachingConceptCaveat,
	},
}

// teachingFor returns the curated card for a known ID of the requested kind.
// Unknown IDs get an honest generic card for that kind: no invented role,
// model, permission or runtime behavior.
func teachingFor(id, kind string) teachingCard {
	for _, card := range teachingRegistry {
		if card.ID == id && card.Kind == kind {
			return card
		}
	}
	return customTeachingCard(id, kind)
}

func customTeachingCard(id, kind string) teachingCard {
	card := teachingCard{
		ID:      id,
		Kind:    kind,
		Group:   teachingGroupCustom,
		When:    "Unknown here, so this guide cannot say when it helps.",
		Example: "Read this item's own files and explain it before relying on it.",
	}
	switch kind {
	case teachingKindSkill:
		card.Purpose = "Custom skill — purpose not documented by Agentic SDD."
		card.Caveat = teachingSkillCaveat
	case teachingKindProfile:
		card.Purpose = "Custom profile — purpose not documented by Agentic SDD."
		card.Caveat = teachingProfileCaveat
	case teachingKindConcept:
		card.Purpose = "Custom concept — not documented by Agentic SDD."
		card.Caveat = teachingConceptCaveat
	default:
		card.Purpose = "Custom item — not documented by Agentic SDD."
		card.Caveat = teachingConceptCaveat
	}
	return card
}

// teachingConcepts returns the key-concept cards in their stable display
// order. Callers get an independent slice.
func teachingConcepts() []teachingCard {
	var out []teachingCard
	for _, card := range teachingRegistry {
		if card.Kind == teachingKindConcept {
			out = append(out, card)
		}
	}
	return out
}
