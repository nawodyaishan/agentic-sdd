---
name: agentic-sdd-debugging
description: "Diagnose errors, failing tests, intermittent failures and unexpected behavior in feature work, direct fixes or read-only reviews; separate evidence from hypotheses before correcting an authorized cause."
---

# Agentic SDD Debugging

Read the [shared workflow policy](../agentic-sdd-router/references/workflow-policy.md) once per unchanged context; it also applies to direct invocation. This is reusable process guidance, not a domain specialist or permission to delegate.

## Entry and authority

Use the known symptom, expected behavior, environment/version, affected scope and actual authorization. Reuse context already supplied; ask only for missing information that matters to a discriminating check. A direct-fix investigation needs no feature documents, task assignment or batch ID. Feature work uses only the relevant existing task/contract context.

Keep the invoking mode's restrictions. In review-only mode, inspect supplied evidence through permitted observation, recommend instrumentation if useful, and make no edits or write-producing test runs. Investigation alone does not authorize a fix, production mitigation, deployment, broader refactor or model/provider change. A consequential finding can block readiness without granting permission to repair it.

## Diagnosis loop

1. Reproduce within the permitted environment, or characterize an intermittent failure using available observations. Record the input/condition, expected result and what actually happened. Missing prerequisites are not proof of the target regression.
2. Separate baseline/setup/environment failures from introduced behavior. Compare a relevant known-good input/revision when available; do not reconstruct a successful baseline from assumption.
3. Trace the implicated data/control boundary with targeted source and evidence. State one supported hypothesis and a check that distinguishes it from a plausible alternative. Use versioned research only for a named factual uncertainty, then return here.
4. Perform that check only within authorization. Preserve failed results and uncertainty. If attempts add no evidence, change the investigation: inspect missing context, revisit the hypothesis or seek a decision. A retry count does not prove bad architecture and never triggers automatic escalation.
5. When correction is authorized, fix the supported cause within scope and verify the original failure plus affected regressions using the [testing guidance](../agentic-sdd-router/references/testing.md). Do not broaden a fix merely because instrumentation or cleanup would be convenient.
6. Return evidence, conclusion/confidence, unresolved alternatives, actual correction/verification if any, and the next safe action to the invoking stage. Diagnosis does not replace its completion review or advance a batch.

Keep this in the existing response/task context, not a mandatory diagnosis report. Never turn an unreproduced intermittent issue, missing setup or an exhausted effort budget into a claimed successful fix.
