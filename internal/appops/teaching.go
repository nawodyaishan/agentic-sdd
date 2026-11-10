package appops

// This file is the single curated source of non-executable teaching metadata
// for the CLI guide and the TUI catalog. It explains only: it never enumerates
// the install set, grants permissions, selects a model or approves work. Exact
// membership still comes from source inventory in internal/skillsync.

// TeachingCard is one curated, non-executable learning entry. DisplayName is a
// readable known-entry label; unknown entries use their exact ID instead of an
// invented name.
type TeachingCard struct {
	ID          string
	DisplayName string
	Kind        string
	Group       string
	Purpose     string
	When        string
	Example     string
	Caveat      string
}

const (
	// TeachingKindSkill marks a shipped or custom skill.
	TeachingKindSkill = "skill"
	// TeachingKindProfile marks a shipped or custom agent profile.
	TeachingKindProfile = "profile"
	// TeachingKindConcept marks a maintained key concept.
	TeachingKindConcept = "concept"

	// TeachingGroupCustom is the honest group for unlisted local items.
	TeachingGroupCustom = "Custom"
	// TeachingGroupConcept groups the maintained key-concept cards.
	TeachingGroupConcept = "Key concepts"
)

// Caveats separate a template's declared intent from enforcement. They render
// as one scannable safety badge per kind, so no card can imply a sandbox,
// model access, permission grant or approval that the tool does not provide.
// The extended explanation lives in the key-concept cards instead.
const (
	TeachingSkillCaveat   = "Advisory guidance · No automated agent execution · Human approval required"
	TeachingProfileCaveat = "Template intent only · No automated agent execution · Human approval required"
	TeachingConceptCaveat = "Teaching concept · No automated agent execution · Human approval required"
)

var teachingRegistry = []TeachingCard{
	// Core skills shipped in skills-files/, grouped by purpose.
	{
		ID: "agentic-sdd-router", DisplayName: "SDD Router", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Route a request through combined spec, plan and tasks approval into bounded, approved work.",
		When:    "You describe an outcome or a resumption and want the right SDD entry point chosen.",
		Example: "Use `agentic-sdd-router` to inspect this repository and draft the three documents for my combined approval.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-spec", DisplayName: "Spec Drafter", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Draft the feature spec as the first of three documents for one combined approval.",
		When:    "A feature outcome is agreed and its scope and acceptance need a written draft.",
		Example: "Use `agentic-sdd-spec` to turn this goal into a draft spec and stop before planning.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-plan", DisplayName: "Plan Drafter", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Draft the feature plan and tasks from the draft spec for one combined approval.",
		When:    "A draft spec exists and the work needs boundaries, dependencies and batches.",
		Example: "Use `agentic-sdd-plan` to draft the plan and tasks from my draft spec and stop for approval.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-tasks", DisplayName: "Task Batcher", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Order feature tasks into explicit batches for the same combined approval.",
		When:    "The plan exists and execution order, dependencies and batch boundaries need to be explicit.",
		Example: "Use `agentic-sdd-tasks` to split this plan into ordered batches and stop for approval.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-bootstrap", DisplayName: "SDD Bootstrap", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Establish minimal SDD pointers, feature conventions and the combined-approval location in an existing repository.",
		When:    "A repository has SDD rules to preserve but no agreed place for spec, plan, tasks or approval.",
		Example: "Use `agentic-sdd-bootstrap` to establish the minimal SDD pointers here without changing product code.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-grill-me", DisplayName: "Plan Stress Test", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Challenge a plan or decision through focused questions.",
		When:    "You want a deliberate stress test before relying on a plan or direction.",
		Example: "Use `agentic-sdd-grill-me` to challenge this plan and show me where it is weakest.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-grilling", DisplayName: "Decision Interview", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Resolve a consequential or blocked decision by asking the questions that actually matter.",
		When:    "An unresolved behavior, scope or risk decision prevents authorized work.",
		Example: "Use `agentic-sdd-grilling` to clarify this decision before we implement anything.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-questionnaire", DisplayName: "Discovery Questionnaire", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Prepare recipient-specific discovery questions, or reconcile returned answers with source requirements.",
		When:    "Another person holds knowledge you need and the questions must match their role.",
		Example: "Use `agentic-sdd-questionnaire` to prepare focused questions for the person who owns this system.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-research-spec", DisplayName: "Fact Research", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Resolve one named fact from primary sources and return it to the invoking stage.",
		When:    "A concrete unknown blocks drafting, implementation or review.",
		Example: "Use `agentic-sdd-research-spec` to resolve this named fact and return the evidence with its sources.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-implement", DisplayName: "Batch Implementer", Kind: TeachingKindSkill, Group: "Implement/debug",
		Purpose: "Implement one approved batch or direct fix and verify the result.",
		When:    "A batch or bounded fix is approved and you want focused implementation rather than a new plan.",
		Example: "Use `agentic-sdd-implement` for this approved batch, keep edits inside its scope and report the checks.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-debugging", DisplayName: "Failure Diagnosis", Kind: TeachingKindSkill, Group: "Implement/debug",
		Purpose: "Diagnose errors, failing tests and unexpected behavior, separating evidence from hypotheses.",
		When:    "Something is broken or intermittent and the cause is not yet established.",
		Example: "Use `agentic-sdd-debugging` to find the cause of this failing test and report evidence before changing code.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-drift-retro", DisplayName: "Drift Reconciliation", Kind: TeachingKindSkill, Group: "Implement/debug",
		Purpose: "Bring material drift back to canonical sources and capture the lessons worth keeping.",
		When:    "Implementation or documentation has diverged from the approved record and the difference matters.",
		Example: "Use `agentic-sdd-drift-retro` to reconcile this drift with the approved documents and record what changed.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-verification-review", DisplayName: "Verification Review", Kind: TeachingKindSkill, Group: "Verify/review",
		Purpose: "Review completed work against requirements and evidence while keeping review-only requests non-mutating.",
		When:    "A batch or fix claims completion and needs an evidence-based check.",
		Example: "Use `agentic-sdd-verification-review` to check this finished batch against its acceptance criteria.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-architecture-review", DisplayName: "Architecture Review", Kind: TeachingKindSkill, Group: "Verify/review",
		Purpose: "Assess a consequential feature design and return a technical architecture verdict.",
		When:    "A design's cost, security or operational risk is hard to reverse and deserves a second look.",
		Example: "Use `agentic-sdd-architecture-review` to challenge this design and return a verdict with its risks.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-teach", DisplayName: "Concept Explainer", Kind: TeachingKindSkill, Group: "Learn/reflect",
		Purpose: "Explain an explicitly requested concept, or maintain a separately requested learning workspace.",
		When:    "You want to understand an idea rather than advance a feature.",
		Example: "Use `agentic-sdd-teach` to explain how combined approval works in this project.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-wait-what", DisplayName: "Plain Restatement", Kind: TeachingKindSkill, Group: "Learn/reflect",
		Purpose: "Re-explain the current proposal, progress or decision in plain language.",
		When:    "The conversation has become dense and you want the current position restated simply.",
		Example: "Use `agentic-sdd-wait-what` to restate where this proposal stands in plain language.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-handoff", DisplayName: "Session Handoff", Kind: TeachingKindSkill, Group: "Learn/reflect",
		Purpose: "Prepare a sanitized, canonical-first transfer record for the next session.",
		When:    "Work pauses or moves to another agent and the next session needs the state and open decisions.",
		Example: "Use `agentic-sdd-handoff` to write a sanitized handover for the next session.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-writing-for-agents", DisplayName: "Agent Instruction Writing", Kind: TeachingKindSkill, Group: "Learn/reflect",
		Purpose: "Write or maintain agent-consumed skills, instruction files and references.",
		When:    "Agent instructions need precise triggers, progressive detail and observable completion.",
		Example: "Use `agentic-sdd-writing-for-agents` to tighten this instruction file for another agent.",
		Caveat:  TeachingSkillCaveat,
	},
	{
		ID: "agentic-sdd-orchestrate", DisplayName: "Request Orchestration", Kind: TeachingKindSkill, Group: "Scope/plan",
		Purpose: "Classify a request, delegate only bounded work and reconcile what comes back.",
		When:    "You explicitly want the optional orchestration experience rather than a direct answer.",
		Example: "Use `agentic-sdd-orchestrate` to classify this request and propose the smallest safe delegation.",
		Caveat:  TeachingSkillCaveat,
	},

	// Base OpenCode profiles shipped in opencode-pack/agents/.
	{
		ID: "sdd-orchestrator", DisplayName: "Orchestrator", Kind: TeachingKindProfile, Group: "Main communicator",
		Purpose: "Hold the human conversation, classify the request and supervise bounded delegation.",
		When:    "You want one primary agent that keeps approvals and reconciles specialist work.",
		Example: "Use the `sdd-orchestrator` role for this session and keep approvals with me.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-debug-diagnose", DisplayName: "Debug Diagnostician", Kind: TeachingKindProfile, Group: "Read-only investigation",
		Purpose: "Diagnose an error or failing test and separate evidence from hypotheses.",
		When:    "You need the cause established before anyone edits code.",
		Example: "Use `sdd-debug-diagnose` to establish this failure's cause and return evidence before any fix.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-test-design", DisplayName: "Test Designer", Kind: TeachingKindProfile, Group: "Test assistance",
		Purpose: "Design focused test cases, fixtures and commands without editing or running them.",
		When:    "A change needs a test plan before implementation or review.",
		Example: "Use `sdd-test-design` to propose focused tests and fixtures for this change.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-research-librarian", DisplayName: "Research Librarian", Kind: TeachingKindProfile, Group: "Read-only investigation",
		Purpose: "Resolve a named technical fact from primary sources without repository writes.",
		When:    "A concrete external fact blocks a decision or draft.",
		Example: "Use `sdd-research-librarian` to resolve this API question from primary sources.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-docs-worker", DisplayName: "Docs Worker", Kind: TeachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Write or update documentation in an explicitly assigned path set.",
		When:    "Documentation changes are authorized and their paths are known.",
		Example: "Use `sdd-docs-worker` for the assigned documentation paths in this batch.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-architecture-advisor", DisplayName: "Architecture Advisor", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Advise on architecture, boundaries and operational risk without approving or implementing.",
		When:    "You want a second architectural opinion before committing to a direction.",
		Example: "Use `sdd-architecture-advisor` to review this design's boundaries and operational risk.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-visual-inspector", DisplayName: "Visual Inspector", Kind: TeachingKindProfile, Group: "Read-only investigation",
		Purpose: "Describe images, screenshots and rendered pages from disk for an agent that cannot see them.",
		When:    "Visual evidence must be transcribed accurately without acting on it.",
		Example: "Use `sdd-visual-inspector` to describe this screenshot from disk.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-openai-architect", DisplayName: "OpenAI Architect", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Provide an optional external architecture and specification review in advisory mode.",
		When:    "A high-impact design wants another independent advisory pass.",
		Example: "Use `sdd-openai-architect` for an independent architecture opinion on this design.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-openai-implement", DisplayName: "OpenAI Implementer", Kind: TeachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Implement approved engineering tasks and difficult refactoring inside the assigned scope.",
		When:    "A complex approved change needs an additional implementation profile.",
		Example: "Use `sdd-openai-implement` for this approved task inside its stated scope.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-architecture-review-9r", DisplayName: "Architecture Review 9R", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Return a technical architecture verdict on risk without replacing human combined approval.",
		When:    "A consequential design needs an independent advisory verdict.",
		Example: "Use `sdd-architecture-review-9r` for an architecture verdict on this design's risks.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-code-review-9r", DisplayName: "Code Review 9R", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Review SDD work and verification evidence independently.",
		When:    "Completed work needs an independent review of code and evidence.",
		Example: "Use `sdd-code-review-9r` to review this completed batch and its evidence.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-review-verification-1", DisplayName: "Verification Reviewer 1", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Review explicitly delegated SDD work and verification evidence independently.",
		When:    "A reviewer different from the implementer is wanted for a delegated batch.",
		Example: "Use `sdd-review-verification-1` to review this delegated batch independently.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-review-verification-2", DisplayName: "Verification Reviewer 2", Kind: TeachingKindProfile, Group: "Independent/supplementary review",
		Purpose: "Provide a supplementary review that is not the sole consequential reviewer.",
		When:    "A second opinion is useful but cannot carry the final verdict alone.",
		Example: "Use `sdd-review-verification-2` for a supplementary opinion beside the main verdict.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-lightweight", DisplayName: "Lightweight Discovery", Kind: TeachingKindProfile, Group: "Read-only investigation",
		Purpose: "Handle bounded discovery and source-backed summaries for a larger session.",
		When:    "Several searches or reads would otherwise consume the main agent's context.",
		Example: "Use `sdd-lightweight` for this bounded lookup so the main session keeps its context.",
		Caveat:  TeachingProfileCaveat,
	},

	// Optional Free-first preset profiles shipped in opencode-pack/presets/.
	{
		ID: "sdd-implement-1", DisplayName: "Implementer 1", Kind: TeachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Routine implementation of one delegated batch or bounded fix.",
		When:    "A straightforward batch or bounded fix has been delegated and the preset matches the work.",
		Example: "Use `sdd-implement-1` for this approved routine batch and stop at its boundary.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-implement-2", DisplayName: "Implementer 2", Kind: TeachingKindProfile, Group: "Implementation/documentation",
		Purpose: "Complex or ambiguous implementation of one delegated batch.",
		When:    "A delegated batch needs more context or reasoning than routine work.",
		Example: "Use `sdd-implement-2` for this complex approved batch and report any uncertainty.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-test-implement-1", DisplayName: "Test Implementer 1", Kind: TeachingKindProfile, Group: "Test assistance",
		Purpose: "Focused test implementation for a delegated batch.",
		When:    "Test work is explicitly delegated and routine.",
		Example: "Use `sdd-test-implement-1` for the focused tests in this batch.",
		Caveat:  TeachingProfileCaveat,
	},
	{
		ID: "sdd-test-implement-2", DisplayName: "Test Implementer 2", Kind: TeachingKindProfile, Group: "Test assistance",
		Purpose: "Focused test implementation for a more complex delegated batch.",
		When:    "A delegated batch needs additional test implementation effort.",
		Example: "Use `sdd-test-implement-2` for the focused tests in this complex batch.",
		Caveat:  TeachingProfileCaveat,
	},

	// Key concepts, declared last so teachingConcepts keeps a stable order.
	{
		ID: "spec-plan-tasks", DisplayName: "Spec, Plan and Tasks", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "Three documents drafted together: what to build, how to build it and in what order.",
		When:    "Starting a feature that needs scope, design and ordered batches before implementation.",
		Example: "Use one combined approval for `spec.md`, `plan.md` and `tasks.md` before any implementation.",
		Caveat:  TeachingConceptCaveat,
	},
	{
		ID: "skill-profile", DisplayName: "Skill vs Profile", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "A skill teaches reusable workflow steps; a profile declares an agent's role, focus and tool boundaries.",
		When:    "You are unsure whether a task needs a workflow or a delegated role.",
		Example: "Use a skill for reusable workflow steps and a profile for a delegated role.",
		Caveat:  TeachingConceptCaveat,
	},
	{
		ID: "main-subagent", DisplayName: "Main Agent and Subagent", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "The main agent keeps the conversation and authority; a subagent receives one bounded brief and returns evidence.",
		When:    "Deciding whether work should stay in the main session or be delegated.",
		Example: "Use the main agent for the conversation and a subagent for one bounded brief.",
		Caveat:  TeachingConceptCaveat,
	},
	{
		ID: "preview-apply", DisplayName: "Preview and Apply", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "Preview shows the plan without writing; apply performs the reviewed changes after explicit confirmation.",
		When:    "About to install or change files and you want to see the effect first.",
		Example: "Use preview to see the plan and apply only after my explicit confirmation.",
		Caveat:  TeachingConceptCaveat,
	},
	{
		ID: "free-first", DisplayName: "Free-first Presets", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "Optional profile presets intended for provider free tiers, with limits and availability beyond this tool's control.",
		When:    "Considering the optional presets and wanting to understand their intent and limits.",
		Example: "Use the Free-first presets only after reading their intent and limits.",
		Caveat:  TeachingConceptCaveat,
	},
	{
		ID: "human-approval", DisplayName: "Human Approval", Kind: TeachingKindConcept, Group: TeachingGroupConcept,
		Purpose: "A human grants the combined spec, plan and tasks approval and authorizes each batch; no agent can substitute.",
		When:    "Clarifying who decides scope or when implementation may start.",
		Example: "Use the combined approval gate: no batch starts without my explicit authorization.",
		Caveat:  TeachingConceptCaveat,
	},
}

// TeachingFor returns the curated card for a known ID of the requested kind.
// Unknown IDs get an honest generic card for that kind: no invented role,
// model, permission or runtime behavior, and the exact ID as its display name.
func TeachingFor(id, kind string) TeachingCard {
	for _, card := range teachingRegistry {
		if card.ID == id && card.Kind == kind {
			return card
		}
	}
	return TeachingCustomCard(id, kind)
}

// TeachingCustomCard builds the neutral fallback for an unlisted item of the
// given kind. It deliberately ignores the curated registry so a local item
// cannot inherit a curated role by reusing a known ID.
func TeachingCustomCard(id, kind string) TeachingCard {
	card := TeachingCard{
		ID:          id,
		DisplayName: id,
		Kind:        kind,
		Group:       TeachingGroupCustom,
		When:        "Unknown here, so this guide cannot say when it helps.",
		Example:     "Read this item's own files and explain it before relying on it.",
	}
	switch kind {
	case TeachingKindSkill:
		card.Purpose = "Custom skill — purpose not documented by Agentic SDD."
		card.Caveat = TeachingSkillCaveat
	case TeachingKindProfile:
		card.Purpose = "Custom profile — purpose not documented by Agentic SDD."
		card.Caveat = TeachingProfileCaveat
	case TeachingKindConcept:
		card.Purpose = "Custom concept — not documented by Agentic SDD."
		card.Caveat = TeachingConceptCaveat
	default:
		card.Purpose = "Custom item — not documented by Agentic SDD."
		card.Caveat = TeachingConceptCaveat
	}
	return card
}

// TeachingCards returns every curated card, including the key concepts, in
// stable display order. Callers get an independent slice.
func TeachingCards() []TeachingCard {
	return append([]TeachingCard{}, teachingRegistry...)
}

// TeachingConcepts returns the key-concept cards in their stable display order.
// Callers get an independent slice.
func TeachingConcepts() []TeachingCard {
	var out []TeachingCard
	for _, card := range teachingRegistry {
		if card.Kind == TeachingKindConcept {
			out = append(out, card)
		}
	}
	return out
}
