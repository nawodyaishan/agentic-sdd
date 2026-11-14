# Documentation index

Start here to find current guidance and distinguish it from historical research.
Repository instructions and the feature approval and task records take precedence
over advisory or historical documents. Existing paths are retained so links in
specifications, evidence and handovers continue to work.

This index is split in two: **public manual** (active guides whose procedures
move to the new site) versus **engineering records** (specs, continuations,
evidence and historical research, kept with historical-entry framing). The new
site's canonical page inventory is the [Feature 020 sitemap](../specs/020-documentation-ecosystem/sitemap.md);
site pages authored in parallel lanes are integration-pending here, never invented.

## Product vision

- [Agentic-SDD product vision and evolution strategy](vision/Agentic-SDD-Project-Vision-and-Evolution.md) —
  mission, audiences, engineering principles, and conditional long-term
  direction. It is the product compass, not a feature approval, batch record or
  release commitment.

## Current status and next decisions

### Public manual — active guides

Procedures in these guides move to the new site pages listed in the
[Feature 020 sitemap](../specs/020-documentation-ecosystem/sitemap.md)
(integration-pending until the parallel content lanes land). Until then the
guides below remain the readable source; they are not engineering verdicts.

- [Getting started](getting-started.md) — safe preview and three compact core
  experiences: direct fix, one approved feature batch, and review/resume.
- [Full usage and orchestration guide](orchestration-guide.md) — complete guide
  to core skills, `--with-opencode`, `--opencode-free-first`, OpenAI/Claude/Gemini/DeepSeek
  specialist subagents, MCP tools (`codegraph`, `context7`, `exa`), and backup/restore.
- [Installer reference](installer-reference.md) — commands, flags, backups,
  restore, file permissions, and logs.
- [Execution runs](execution-runs.md) — the approved Feature 016 `run`, `task`
  and `review` contract: private task briefs, truthful check evidence, review
  packages, and `.agentic-sdd/runs/` scratch that is observation, never approval,
  verification, reviewer isolation or a sandbox (implementation complete, awaiting human review).
- [CodeGraph setup and continuous sync](codegraph.md) — local index bootstrap,
  MCP/CLI fallback, sync checks and troubleshooting; never committed.
- [Default MCP integration study](mcp-default-integration-study.md) — CodeGraph,
  Context7 and Exa install-time checks, config merge and missing-piece options.
- [Research skills and subagents: current state](research-skills-and-subagents-current-state.md) —
  how research is routed, delegated and evidenced across the portable skills,
  the opt-in pack and the installed machine fleet.

### Engineering records — specs, continuations and operational boundaries

Retained with historical-entry framing. These records explain what was decided
and observed; they authorize no batch, execution or publication (see note at
the end of this section).

- [TUI architecture and toolchain migration blueprint](tui-architecture.md) —
  experimental Charm v2 hybrid spike and 8-point toolchain audit for Feature 012.
- [Feature 009 spec](../specs/009-opencode-orchestration/spec.md),
  [plan](../specs/009-opencode-orchestration/plan.md), and
  [tasks](../specs/009-opencode-orchestration/tasks.md) — optional OpenCode
  orchestration pack (`opencode-pack/`), delegation skill, and model routing.
- [Public support and release scope](../README.md#support-and-release-scope) —
  portable skills, installer, source-only checks and unsupported native experiments.
- [Test case inventory](testing/test-cases-report.md) — every declared Go and
  Python test plus the container drivers, organized by scope, type and feature;
  an inventory, not a test run.
- [Snap and Ubuntu release report](snap-ubuntu-release-report.md) — research on
  whether `agentic-sdd` can ship as a Snap (strict, `personal-files` or classic)
  and why a `.deb` plus Linux tarball is the recommended Ubuntu channel. It is a
  report, not a release decision or approval.
- [Feature 008 approval](../specs/008-core-release-quality/spec.md#approval),
  [batch continuation](../specs/008-core-release-quality/tasks.md#continuation-record),
  and [working release inventory](../specs/008-core-release-quality/release-readiness.md) —
  the selected core-quality effort; the inventory is not a release verdict.
- [Remaining vNext work and adoption guide](vnext-remaining-work-guide.md) —
  004/005 gap reference; use the owning feature records for actual current state.
- [Feature 005 continuation](../specs/005-vnext-completion/tasks.md#continuation-record)
  and [approval](../specs/005-vnext-completion/spec.md#approval) — canonical
  batch state and approval record.
- [Feature 004 acceptance map](../harness/acceptance.md) — retained findings and
  blocked technical validation.
- [Feature 006 approval](../specs/006-independent-review/spec.md#approval) and
  [continuation](../specs/006-independent-review/tasks.md#continuation-record) —
  native B4 deferred/incomplete; IR-06-01 remains open, not fixed by documentation.
- [Feature 007 approval](../specs/007-taskfile-migration/spec.md#approval) and
  [continuation](../specs/007-taskfile-migration/tasks.md#continuation-record) —
  direct Taskfile adoption; old-base PR evidence was not established.
- [OpenCode safety](../integrations/opencode/safety.md),
  [adoption](../integrations/opencode/adoption.md), and
  [recovery](../integrations/opencode/recovery.md) — operational boundaries.

No index entry, historical handover, accepted partial delivery or technical
verdict authorizes another batch, experimental execution or publication.

## Pinned-image human-cycle evidence (Feature 005)

- [Human test guide](vnext-human-test-guide.md) — pinned-container procedure.
- [Run sheet](vnext-human-cycle-run-sheet.md) — scenario-level recording template.
- [Human usage plan](vnext-human-test-plan.md) — scope and historical test design.

The user reported the cycle complete and accepted it for the pinned image;
individual scenario results remain unspecified. This is not cross-client,
native-runtime or general safety certification. See the continuation record.

## Advisory plans and historical research

- [vNext adoption plan](Agentic-SDD-vNext-Adoption-Plan.md) — revision 6 advisory
  proposal; its baseline predates feature 005 and is not current implementation status.
- [Workflow audit](Agentic-SDD-Workflow-Audit-2026-09-24.md) — historical audit.
- [Corrected September audit](../audits/Agentic-SDD-Workflow-Audit-2026-09-24%20%281%29.md) —
  historical superseding report, not a new approval. The original also exists at
  `audits/Agentic-SDD-Workflow-Audit-2026-09-24.md`; both paths are retained.
- [Workflow research artifact](compass_artifact_wf-97cc6285-b594-5b61-ae30-2894cfe0b595_text_markdown.md)
  and [Reddit research](reddit_research.md) — historical inputs, not current policy.
- [Architecture illustration](agentic-sdd-architecture.png) — README asset.

## Session handovers

See [handovers/](handovers/) for the latest session pointer only. Older handovers
were removed from the working tree at the owner's request; tracked versions
remain in Git history and untracked material was privately archived, not published.
Handovers are not approval records; re-read the owning continuation and current diff.
