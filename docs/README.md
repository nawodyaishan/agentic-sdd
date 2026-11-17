# Documentation index

Start here to find current guidance and distinguish it from historical research.
Repository instructions and the feature approval and task records take precedence
over advisory or historical documents. Existing paths are retained so links in
specifications, evidence and handovers continue to work.

**Start with the documentation site.** It is the current entry for setup, commands,
references and limits: <https://nawodyaishan.github.io/agentic-sdd/>, with the source
under [`website/src/content/docs/`](../website/src/content/docs/index.md). The
[Feature 020 sitemap](../specs/020-documentation-ecosystem/sitemap.md) records the page
inventory those pages were authored against.

This index is split in two: **public manual** (the old guides below, now
historical migration stubs or dated guides, each pointing to its site page) versus
**engineering records** (specs, continuations, evidence and dated research, kept with
historical-entry framing). The pages named in this index exist; none is invented here.

## Product vision

- [Agentic-SDD product vision and evolution strategy](vision/Agentic-SDD-Project-Vision-and-Evolution.md) —
  mission, audiences, engineering principles, and conditional long-term
  direction. It is the product compass, not a feature approval, batch record or
  release commitment.

## Current status and next decisions

### Public manual — migrated, with historical stubs

The procedures that lived in these files now have canonical site pages. The
retained files are migration stubs or a dated guide; they are not current
instructions and not engineering verdicts.

- [Getting started](getting-started.md) — dated pre-website walkthrough of a
  safe preview, a direct fix and one approved feature batch. Current onboarding:
  [First five minutes](../website/src/content/docs/getting-started/first-five-minutes.md)
  and [Choose your setup](../website/src/content/docs/getting-started/choose-setup.md).
- [Full usage and orchestration guide](orchestration-guide.md) — **historical migration
  stub** (Feature 020). Current pages: [Orchestrate optional specialist agents](../website/src/content/docs/how-to/orchestration.md)
  and [Agent profile catalog](../website/src/content/docs/reference/profiles.md).
- [Installer reference](installer-reference.md) — **historical migration stub**
  (Feature 020). Current pages: [CLI reference](../website/src/content/docs/reference/cli.md),
  [Files and backups](../website/src/content/docs/reference/files-and-backups.md) and
  [Updates and recovery](../website/src/content/docs/how-to/updates-and-recovery.md).
- [Execution runs](execution-runs.md) — the approved Feature 016 `run`, `task`
  and `review` contract: private task briefs, truthful check evidence, review
  packages, and `.agentic-sdd/runs/` scratch that is observation, never approval,
  verification, reviewer isolation or a sandbox (implementation complete, awaiting human review).
- [CodeGraph setup and continuous sync](codegraph.md) — local index bootstrap,
  MCP/CLI fallback, sync checks and troubleshooting; never committed.
- [Default MCP integration study](mcp-default-integration-study.md) — CodeGraph,
  Context7 and Exa install-time checks, config merge and missing-piece options.
- [Research skills and subagents: dated study (2026-11-12)](research-skills-and-subagents-current-state.md) —
  a **historical observation** of routing, delegation and installed state on that date
  (for example, the pack's research librarian was then "not installed" and no written
  precedence rule existed). It is not current policy. Current owners:
  [the research-spec skill](../skills-files/agentic-sdd-research-spec/SKILL.md),
  the [workflow policy](../skills-files/agentic-sdd-router/references/workflow-policy.md),
  and the [skill catalog](../website/src/content/docs/reference/skills.md).

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
