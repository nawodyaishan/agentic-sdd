---
title: Skill catalog
description: "The 18 portable core skills plus the optional orchestration skill: categories, entry points, and where each normative instruction lives."
---

Source-derived skill and reference facts. The normative instructions remain in the packaged `SKILL.md` files; this page groups skills for lookup without copying their bodies.

## Beginner-facing skill categories

Core skills live under `skills-files/` and install by default with `asdx-core apply`. Four core skills set `disable-model-invocation: true` (`agentic-sdd-grill-me`, `agentic-sdd-handoff`, `agentic-sdd-teach`, `agentic-sdd-wait-what`), so agents only invoke them when you ask for them explicitly.

### Scope and planning

Use these skills to route a request, establish repository SDD pointers, resolve blocking unknowns, and draft `spec.md`, `plan.md`, and `tasks.md` for combined human approval:

| Skill | What it is for | Normative source |
|---|---|---|
| `agentic-sdd-router` | Starting point: classifies a request into a feature, direct fix, or explicit teaching path and routes to the matching stage skill. | [`skills-files/agentic-sdd-router/SKILL.md`](../../../../../skills-files/agentic-sdd-router/SKILL.md) |
| `agentic-sdd-bootstrap` | Establishes minimal SDD pointers, feature conventions, and the single combined-approval location in an existing repo. | [`skills-files/agentic-sdd-bootstrap/SKILL.md`](../../../../../skills-files/agentic-sdd-bootstrap/SKILL.md) |
| `agentic-sdd-spec` | Drafts the feature `spec.md` as the first of three documents for combined approval. | [`skills-files/agentic-sdd-spec/SKILL.md`](../../../../../skills-files/agentic-sdd-spec/SKILL.md) |
| `agentic-sdd-plan` | Drafts `plan.md` from the draft spec with specialist and tool assignments, continuing into tasks. | [`skills-files/agentic-sdd-plan/SKILL.md`](../../../../../skills-files/agentic-sdd-plan/SKILL.md) |
| `agentic-sdd-tasks` | Orders feature tasks into explicit execution batches for combined approval. | [`skills-files/agentic-sdd-tasks/SKILL.md`](../../../../../skills-files/agentic-sdd-tasks/SKILL.md) |
| `agentic-sdd-grilling` | Clarifies a consequential behavior, scope, or risk decision that blocks authorized work (or challenges a plan on request). | [`skills-files/agentic-sdd-grilling/SKILL.md`](../../../../../skills-files/agentic-sdd-grilling/SKILL.md) |
| `agentic-sdd-grill-me` | Explicitly invoked stress test that challenges a plan or decision with focused questions. | [`skills-files/agentic-sdd-grill-me/SKILL.md`](../../../../../skills-files/agentic-sdd-grill-me/SKILL.md) |
| `agentic-sdd-questionnaire` | Prepares recipient-specific discovery questions when someone else holds required knowledge, or reconciles returned answers. | [`skills-files/agentic-sdd-questionnaire/SKILL.md`](../../../../../skills-files/agentic-sdd-questionnaire/SKILL.md) |
| `agentic-sdd-research-spec` | Resolves one named, consequential fact from primary sources when it blocks an SDD stage, then returns to that stage. | [`skills-files/agentic-sdd-research-spec/SKILL.md`](../../../../../skills-files/agentic-sdd-research-spec/SKILL.md) |

### Implementation and debugging

Use these skills after approval (or for a bounded direct fix) to implement changes, diagnose failures, and reconcile material drift:

| Skill | What it is for | Normative source |
|---|---|---|
| `agentic-sdd-implement` | Implements one approved batch or one bounded direct fix and runs focused verification. | [`skills-files/agentic-sdd-implement/SKILL.md`](../../../../../skills-files/agentic-sdd-implement/SKILL.md) |
| `agentic-sdd-debugging` | Diagnoses errors, failing tests, and intermittent failures while separating evidence from hypotheses before fixing. | [`skills-files/agentic-sdd-debugging/SKILL.md`](../../../../../skills-files/agentic-sdd-debugging/SKILL.md) |
| `agentic-sdd-drift-retro` | Reconciles material drift back to canonical sources and records useful lessons without routine status bloat. | [`skills-files/agentic-sdd-drift-retro/SKILL.md`](../../../../../skills-files/agentic-sdd-drift-retro/SKILL.md) |

### Verification and review

Use these skills to review completed work or consequential feature architecture without replacing human combined approval:

| Skill | What it is for | Normative source |
|---|---|---|
| `agentic-sdd-verification-review` | Reviews a batch, direct fix, or feature against requirements and engineering evidence; keeps review-only requests read-only. | [`skills-files/agentic-sdd-verification-review/SKILL.md`](../../../../../skills-files/agentic-sdd-verification-review/SKILL.md) |
| `agentic-sdd-architecture-review` | Assesses consequential feature architecture and operational risk as a technical verdict. | [`skills-files/agentic-sdd-architecture-review/SKILL.md`](../../../../../skills-files/agentic-sdd-architecture-review/SKILL.md) |

### Learning, handoff, and authoring

Use these skills for plain-language explanations, session transfers, or maintaining agent-consumed instructions:

| Skill | What it is for | Normative source |
|---|---|---|
| `agentic-sdd-teach` | Explicitly invoked: explains a requested concept or maintains a separately requested learning workspace. | [`skills-files/agentic-sdd-teach/SKILL.md`](../../../../../skills-files/agentic-sdd-teach/SKILL.md) |
| `agentic-sdd-wait-what` | Explicitly invoked: restates the current proposal, progress, or decision in plain language. | [`skills-files/agentic-sdd-wait-what/SKILL.md`](../../../../../skills-files/agentic-sdd-wait-what/SKILL.md) |
| `agentic-sdd-handoff` | Explicitly invoked: prepares a sanitized canonical-first handover record for the next session. | [`skills-files/agentic-sdd-handoff/SKILL.md`](../../../../../skills-files/agentic-sdd-handoff/SKILL.md) |
| `agentic-sdd-writing-for-agents` | Writes or updates agent-consumed skills, instruction files, and references with clear triggers and completion criteria. | [`skills-files/agentic-sdd-writing-for-agents/SKILL.md`](../../../../../skills-files/agentic-sdd-writing-for-agents/SKILL.md) |

### Optional orchestration skill source

The optional `agentic-sdd-orchestrate` skill is installed only with `--with-opencode` (enabled by default in `asdx`, disabled by default in `asdx-core`):

- `agentic-sdd-orchestrate`: classifies requests, issues compact task briefs, supervises bounded delegation, and reconciles returned evidence while keeping approvals with the main agent — [`opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md`](../../../../../opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md).

## Related reference pages

- [Profiles and presets reference](./profiles.md) — OpenCode agent profiles, free-first presets, and fleet adapters.
- [Files and backups](./files-and-backups.md) — client destinations where skills and agent profiles are installed.
- [TUI reference](./tui.md) — browsing skills and profiles in `asdx` (`3` Explore).

<!-- generated:reference/skills:start -->
## Core skills

18 skills are defined in `skills-files/`, one directory with a `SKILL.md` each. The default `asdx-core apply` selection (also reachable through the `agentic-sdd` wrapper) selects this set without any opt-in pack flag.

| Skill ID | Model-invocable | Description |
| --- | --- | --- |
| `agentic-sdd-architecture-review` | yes | Review consequential feature architecture and operational risk as a technical verdict, without replacing the human combined approval. |
| `agentic-sdd-bootstrap` | yes | Manually establish minimal SDD pointers, feature conventions and the single combined-approval location using existing repository files. |
| `agentic-sdd-debugging` | yes | Diagnose errors, failing tests, intermittent failures and unexpected behavior in feature work, direct fixes or read-only reviews; separate evidence from hypotheses before correcting an authorized cause. |
| `agentic-sdd-drift-retro` | yes | Handle material drift in feature work or a direct fix, update canonical sources, and capture useful lessons without routine reports. |
| `agentic-sdd-grill-me` | no | Explicitly challenge a plan or decision through focused clarification. |
| `agentic-sdd-grilling` | yes | Challenge a plan on request, or clarify consequential behavior, scope or risk when an unresolved decision blocks authorized work. |
| `agentic-sdd-handoff` | no | Prepare a sanitized canonical-first transfer when the user requests a handoff or repository guidance requires a session handover. |
| `agentic-sdd-implement` | yes | Implement one approved batch, or one direct fix, with assigned specialist guidance loaded in the main agent and focused verification. |
| `agentic-sdd-plan` | yes | Draft a feature plan from the draft spec with specialist and tool assignments, then continue to tasks for one combined approval. |
| `agentic-sdd-questionnaire` | yes | Prepare a recipient-specific discovery questionnaire when another person holds needed knowledge, or reconcile returned answers with source requirements. |
| `agentic-sdd-research-spec` | yes | Resolve a named, consequential fact blocking an SDD stage (spec, plan, tasks, review, debugging or direct fix) and return to that stage. Not for generic research, topic reports, background delegation or mandatory research files. |
| `agentic-sdd-router` | yes | Route SDD intake and feature work through combined spec/plan/tasks approval, batched implementation and direct fixes. |
| `agentic-sdd-spec` | yes | Draft a feature spec from the relevant SRS and roadmap sections as the first of three documents for one combined approval. |
| `agentic-sdd-tasks` | yes | Draft ordered feature tasks and explicit execution batches, then present spec, plan and tasks for one combined approval. |
| `agentic-sdd-teach` | no | Teach an explicitly requested concept with a focused explanation, or maintain a separately requested learning workspace. |
| `agentic-sdd-verification-review` | yes | Review a batch, direct fix or whole feature against requirements and engineering evidence; resolve findings within authority, and keep review-only requests non-mutating. |
| `agentic-sdd-wait-what` | no | Re-explain the current proposal, progress or decision in plain language when explicitly asked. |
| `agentic-sdd-writing-for-agents` | yes | Write or maintain agent-consumed skills, instruction files and references with precise triggers, progressive disclosure and observable completion criteria. |

4 of 18 core skills declare `disable-model-invocation: true` and are explicitly invocable only: `agentic-sdd-grill-me`, `agentic-sdd-handoff`, `agentic-sdd-teach`, `agentic-sdd-wait-what`.

## Optional orchestration skill

1 skill is defined in `opencode-pack/skills/`. It is selected only together with the opt-in OpenCode pack (`--with-opencode`) and is not part of the default core selection.

| Skill ID | Model-invocable | Description |
| --- | --- | --- |
| `agentic-sdd-orchestrate` | yes | Use when the human explicitly requests bounded delegation or enables the optional Agentic-SDD orchestration experience. The main agent classifies the request, issues compact task briefs, supervises children, reconciles evidence and reports back, while retaining human approval and the existing Agentic-SDD authority. Not for ordinary direct fixes or single-tool actions. |

## Derived counts

| Class | Source root | Count |
| --- | --- | --- |
| Core skills | `skills-files/` | 18 |
| Optional orchestration skills | `opencode-pack/skills/` | 1 |

## Source evidence

Facts above are parsed from 19 `SKILL.md` files with the pinned `yaml` parser; fingerprints are recorded in the documentation manifest. This block describes repository source, not the installation state of any machine.
<!-- generated:reference/skills:end -->
