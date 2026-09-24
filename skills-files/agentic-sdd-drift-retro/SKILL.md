---
name: agentic-sdd-drift-retro
description: "Handle material packet drift, canonical-source updates and useful lessons without routine reports."
---

# Agentic SDD Drift Retro

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after verification finds a meaningful discrepancy, at a useful milestone, during release preparation, or on explicit request. Ordinary drift checks belong to `agentic-sdd-verification-review`.

Read the existing finding, implicated diff and relevant approved packet sections; consult only affected top-level SRS, technical-spec and roadmap sections. Missing optional files are not drift. If no material discrepancy exists, state that briefly and create no file.

If authorized behavior changed, update affected canonical sources and `Docs/Tasks.md` roadmap status as appropriate, while preserving packet decision history. If implementation conflicts with approved packet intent, correct it or return to the affected spec/plan/task stage for human review. Do not rewrite approval evidence to legitimize an unauthorized departure. Prefer tracking a concrete discrepancy in the current packet task with impact and next action; make a separate drift report only for an independent audit need or explicit request.

Write a retrospective only after a useful incident/lesson or milestone, or when requested. Batch release notes in the repository's existing release process. Release preparation does not authorize publishing or deployment.

Output actual corrections and unresolved decisions. No routine no-drift report, per-task retrospective or completion document.
