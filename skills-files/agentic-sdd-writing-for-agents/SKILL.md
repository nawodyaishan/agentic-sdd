---
name: agentic-sdd-writing-for-agents
description: Write or maintain agent-consumed skills, instruction files and references with precise triggers, progressive disclosure and observable completion criteria.
---

# Agentic SDD Writing for Agents

Read the [shared policy](../agentic-sdd-router/references/workflow-policy.md) once
per unchanged context. When authoring a skill, also read
[skill mechanics](references/skill-mechanics.md). Apply this guidance to the
authorized documents only; maintenance is not authority for a repository-wide
rewrite, runtime installation or changed workflow permissions.

1. Establish the document's consumer, trigger branches, authoritative rules and
   output. Inspect actual neighboring files and supported invocation behavior;
   distinguish a reference snapshot from the source being edited.
2. Separate ordered steps from on-demand definitions. Keep what every branch
   needs visible; disclose branch-specific material through a pointer that says
   what it covers and exactly when to reach it. Prune synonyms, not genuinely
   different triggers. Preserve hard authority/effect boundaries at entry.
3. Give each meaning one authoritative owner. Keep a rule beside its caveats;
   reference shared contracts instead of repeating them in every helper. Avoid
   caching environment facts that can be discovered cheaply and may go stale.
4. Make completion criteria checkable and exhaustive for the actual branch.
   Separate observed facts, proposed behavior and unknowns. Concise positive
   instructions help attention, but do not remove prohibitions on unauthorized
   effects, credential exposure or invented approval.
5. Inspect the revised branches, local references, invocation metadata and source
   notices. Return the changed scope, checks and remaining uncertainties. A
   shorter document or structural check is not proof of reliable model behavior
   or saved tokens; do not start a model evaluation without its own authorization.

Use existing project language; define a new compact term only when it genuinely
removes repetition without hiding meaning. Split a file when disclosure earns
its navigation cost, not to satisfy an arbitrary length or file-count target.

Adaptation: [source and license](../agentic-sdd-router/references/matt-pocock-notice.md).
