---
name: agentic-sdd-drift-retro
description: "Handle material drift in feature work or a direct fix, update canonical sources, and capture useful lessons without routine reports."
---

# Agentic SDD Drift Retro

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist and tool map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after verification finds a meaningful discrepancy, at a useful milestone, during release preparation, or on explicit request. Ordinary drift checks belong to `agentic-sdd-verification-review`.

Read the existing finding, the implicated diff and the relevant approved feature sections; for a direct fix, read the request, the diff and the constraints it touched — do not invent feature documents to run this skill. Consult only the affected top-level SRS, technical-spec and roadmap sections. Missing optional files are not drift. If no material discrepancy exists, say so briefly and create no file.

If authorized behavior changed, update the affected canonical sources and `Docs/Tasks.md` roadmap status as appropriate, preserving the feature's decision history. If implementation conflicts with approved intent, correct it or return the affected changes for human review and reconcile the dependent documents; approval for unaffected work stands. Do not rewrite approval evidence to legitimize an unauthorized departure.

Prefer tracking a concrete discrepancy in the current task entry with impact and next action; make a separate drift report only for an independent audit need or explicit request. Where the feature is mid-flight, leave the batch state and next action accurate rather than advancing it.

Write a retrospective only after a useful incident, lesson or milestone, or when requested. Batch release notes through the repository's existing release process. Release preparation does not authorize publishing or deployment.

Output the actual corrections and unresolved decisions. No routine no-drift report, per-task retrospective or completion document.
