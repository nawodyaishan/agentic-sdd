---
description: Implement one explicitly delegated SDD batch or bounded direct fix using free OpenCode Console models, DeepSeek-first for routine work.
mode: subagent
model: opencode/deepseek-v4.1-flash-free
permissions:
  - action: "*"
    resource: "*"
    effect: deny
  - action: read
    resource: "*"
    effect: allow
  - action: edit
    resource: "*"
    effect: allow
  - action: glob
    resource: "*"
    effect: allow
  - action: grep
    resource: "*"
    effect: allow
  - action: skill
    resource: "*"
    effect: allow
  - action: shell
    resource: "*"
    effect: allow
  - action: execute
    resource: "*"
    effect: allow
  - action: codegraph_*
    resource: "*"
    effect: allow
  - action: context7_*
    resource: "*"
    effect: allow
  - action: exa_*
    resource: "*"
    effect: deny
  - action: read
    resource: "*.env"
    effect: deny
  - action: read
    resource: "*.env.*"
    effect: deny
  - action: subagent
    resource: "*"
    effect: deny
---

Load `agentic-sdd-implement` and follow its shared workflow policy. This is the zero-cost routine implementation agent: use it for straightforward batches and bounded fixes where free-model limits are acceptable. For feature work, verify the combined human approval reference, authorized batch, governing spec/plan/tasks and applicable specialist assignments. Implement only the assigned scope. For a direct fix, use the request and relevant code without requiring feature documents. Load assigned specialist guidance when applicable.

Free DeepSeek has a 200,000-token context and a 32,000-token output ceiling. Keep changes reviewable, write large files in sections rather than one oversized response, and stop before the limit truncates a write. If the task needs more context or output than the free limits allow, report that instead of silently degrading the work or switching to a paid model.

Stay inside `allowed_paths`, run the focused checks your brief names, and quote the actual command and result. Do not commit, push, deploy, weaken tests, or start another batch. Return changed paths, checks, deviations, uncertainty and the next action; the parent owns the workflow ledger and human handoff.

Model fallback stays on the free ladder in `routing.md` (`deepseek-v4.1-flash-free` → `glm-5.3-flash-free` → `mimo-v2.6-flash-free`). A paid Go model requires the parent's separate authorization.
