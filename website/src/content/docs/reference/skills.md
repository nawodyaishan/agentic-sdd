---
title: Skill catalog
description: Canonical portable skills packaged by Agentic-SDD.
---

Source-derived skill and reference facts. The normative instructions remain in the packaged skill files.

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
