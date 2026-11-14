---
title: Related approaches
description: A dated, factual comparison with other agent workflows, and when your existing process already suffices.
---

Agentic-SDD sits in a crowded space: host-native agent skills, spec-driven workflow
kits, method frameworks and ordinary engineering process. This page compares their
documented shape so you can decide whether adding anything here is worth the process.
It makes no claim that one approach works better. Effectiveness, cost, learning and
review quality are not measured by this project, and no source below is a benchmark.

## What Agentic-SDD actually provides

Source facts first, because they set the comparison:

- portable Markdown skills installed into four client skill directories, with an
  optional OpenCode pack of 14 agent profiles (`internal/skillsync/`;
  [Skill catalog](../reference/skills.md), [Agent profile catalog](../reference/profiles.md));
- a read-only preview by default, with `apply` as the only writing mode, plus backups and
  restore ([Files and backups](../reference/files-and-backups.md));
- feature documents with one combined human approval recorded in `spec.md` and batch
  state recorded in `tasks.md`, plus a separate one-change path with no feature documents
  ([Concepts](../introduction/concepts.md));
- written model-routing and delegation policy, with fallbacks chosen manually by the
  parent agent, not an automatic failover engine (`internal/appops/routing.go`);
- no daemon, server, team administration, policy engine or hosted component.

Those are the contracts this comparison relies on. The complete command surface,
including fleet, MCP and execution families, is in [CLI reference](../reference/cli.md);
anything absent from the reference pages is not supported, regardless of what a
specification says.

## Documented comparison

The primary sources below were inspected by the audience research pass recorded in
`specs/020-documentation-ecosystem/audiences.md` on **2026-11-14**. This page did not
re-fetch them; upstream projects change, so re-read the source before relying on a
detail. The "shape" column repeats what that inspection recorded, not a deeper
evaluation.

| Approach | Shape recorded on 2026-11-14 | Primary source |
| --- | --- | --- |
| Agentic-SDD | Portable skills plus an installer; feature approval and batch state kept in repository Markdown | `skills-files/`, `specs/` |
| OpenCode skills | Discovery/location and client-native usage reference for skills in that host | [opencode.ai/docs/skills](https://opencode.ai/docs/skills/) |
| OpenSpec | Change-oriented artifact lifecycle | [Fission-AI/OpenSpec](https://github.com/Fission-AI/OpenSpec) |
| Superpowers | Worked engineering skills and optional agent workflows | [obra/superpowers](https://github.com/obra/superpowers) |
| BMAD Method | Progressive method and workflow discovery | [bmad-code-org/BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) |
| GitHub Spec Kit | Entry points and workflow documentation | [github/spec-kit](https://github.com/github/spec-kit) |

Two documentation patterns informed this site without becoming product claims:

| Reference | Pattern used here | Boundary |
| --- | --- | --- |
| [OpenTofu core workflow](https://opentofu.org/docs/intro/core-workflow/) | Preview/plan/apply framing with explicit effects | An analogy, not equivalent safety enforcement |
| [Kubernetes declarative configuration](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/) | Task-focused guides with stated prerequisites | Documentation structure only, not a capability comparison |

What this table does **not** establish: which approach produces better code, which is
cheaper, which is safer, or which a team should adopt. It also does not claim that
peers lack human oversight, review artifacts or portability — several document those
properties in the material inspected.

## Real differences, stated narrowly

These are contract differences visible in source, not quality judgements:

- **Where approval lives.** Agentic-SDD records one combined human approval in the
  feature's `spec.md` and batch state in `tasks.md`; a review verdict never replaces that
  approval. Whether that is better than approval in a chat session or a pull request
  depends on whether you need the record in Git.
- **What gets installed.** Agentic-SDD copies text files into client skill directories.
  It does not become the agent runtime, so the host still owns tool permissions, model
  access and execution.
- **Delegation is optional and manual.** Profiles and fallback ladders are written policy;
  nothing dispatches itself, and a paid rung needs explicit authorization. A host or
  orchestrator that does automatic routing already covers that need.
- **Preview is a local plan, not a policy engine.** `preview` shows installer actions and
  writes nothing; it does not evaluate your permissions, approve a change or prove a
  destination is safe to modify.

## When your existing workflow already suffices

Add nothing when one of these is true:

- **The change is small and reversible.** A normal edit, test and pull request is enough.
  Agentic-SDD's own direct-fix path exists for this, and reaching for feature documents
  would be overhead.
- **Your team already enforces a process.** Branch protection, required reviews, policy
  as code and CI gates are enforcement. Agentic-SDD adds instructions and local files; it
  cannot enforce anything, so replacing an enforced process with it would be a downgrade
  in control.
- **You have no agent host.** The skills are instructions for an agent. With nothing
  loading them, there is nothing to install and no behaviour to gain.
- **You need platform guarantees.** Sandboxing, RBAC, audit trails, secret management and
  network policy belong to the platform and the host, not to a skill package. No page
  here claims them.
- **You need automatic model routing or cost control.** The ladders are documented
  preferences with manual selection. If you need a router with quota and failover, use one.
- **You need a hosted product, dashboards or team administration.** None exists here.
- **You are evaluating effectiveness.** This project has no measured comparison, trial or
  benchmark. Public peer counts and self-descriptions prove neither adoption nor outcomes.

## When the shape may fit

Consider it when several of these are true: you work across more than one agent host and
want the same written procedure in each; you want approval and batch records to live in
the repository with the code; you want optional delegation and routing rules that are
readable and reviewable rather than implicit; and you are comfortable that the tool only
writes local files and leaves enforcement to the host and platform.

The shortest honest test is the one in [First five minutes](../getting-started/first-five-minutes.md):
install into a throwaway home, run a preview, stop. If the preview output and the
document set do not match a problem you actually have, the existing workflow is the
better answer.

## Keeping the comparison fair

If you maintain or use one of the projects above and something here misdescribes it,
that is a documentation defect worth reporting through the repository's contribution
channel. Do not add a "better than" statement: record the dated source, quote what it
says, and state the limit of the comparison. See
[Trust boundaries](../security/trust.md) for how claims are scoped, and
[Known limitations](../operations/known-limitations.md) for the equivalent list on this
project's own side.
