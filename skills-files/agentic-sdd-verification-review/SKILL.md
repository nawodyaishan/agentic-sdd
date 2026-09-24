---
name: agentic-sdd-verification-review
description: "Review implementation against the approved packet spec, plan and task with focused evidence and drift checks."
---

# Agentic SDD Verification Review

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. Consult the [specialist map](../agentic-sdd-router/references/specialists.md) only when assigning or loading domain expertise.

Use after implementation or for an explicit review. Read the actual diff, the packet's approved `spec.md` and `plan.md` sections, current approved `tasks.md` section, applicable contracts/constraints and available test evidence. Consult linked top-level source sections only where the change or a conflict requires them.

Compare behavior to packet acceptance, design boundaries and task outcome. Check scope, compatibility, permissions/security, error handling, tests and significant source or roadmap drift. An agent's `Ready` verdict cannot replace missing human stage signoffs or authorize merge, deployment or live infrastructure work.

Reuse a prior check only if code, relevant dependencies/configuration, inputs and environment still match. Rerun affected, failed or repository-required checks. Choose focused tests and static checks for changed behavior; use rendering/schema/policy validation or an authorized Terraform plan for infrastructure. Do not claim success for checks not run.

Return `Ready`, `Needs changes` or `Blocked`, with concrete file/line findings, checks/results, deviations from the approved packet and unresolved risk. Fix in-scope defects or route material requirement/design departures to the affected packet stage for human review. Summarize evidence in the current task/response; create `verification.md` only for an independent audit record, established convention or explicit request. Avoid a separate no-drift file.
