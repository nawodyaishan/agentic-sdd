---
title: Known limitations
description: What is implemented, verified, experimental or deprecated, and what is not supported.
---

This page keeps one honest question in front of the reader: is this capability
implemented, and has anyone observed it working in the environment you care about? Those
are different answers. Use it to check what is not supported yet before you rely on a
capability. Everything below is derived from the repository at this checkout
and from dated observations recorded in `specs/020-documentation-ecosystem/evidence.md`.
Nothing here is a support promise, and nothing is inferred from a specification alone.

## How to read the status labels

| Label | Meaning |
| --- | --- |
| Implemented | source and tests exist at this checkout; it may still be absent from a published archive |
| Verified | a dated observation with a stated scope exists; it proves only that scope |
| Experimental | present but excluded from the default build or install, or fenced out of supported use |
| Deprecated | still functional for one deprecation window in favour of the canonical name |
| Planned | recorded in a specification only; treat as unsupported |

"Implemented" is about code, not about your machine. "Verified" is about one observation,
not a general guarantee. A task checkbox, a passing unit test and a live install are three
different pieces of evidence.

## Capability status

| Capability | Status | What is established | What is not |
| --- | --- | --- | --- |
| Skill installation to four client destinations | implemented | deterministic plan, backup and rollback paths in `internal/skillsync/`; destination table in [Files and backups](../reference/files-and-backups.md) | a live install in every client, and identical client behaviour |
| Canonical `asdx-core` / `asdx` pair | implemented | canonical names, legacy forwarding and a fail-closed build-identity handshake (`internal/buildinfo/`, `cmd/agentic-sdd/tui.go`) | that a locally built pair passes identity checks without being built together |
| Legacy command names | deprecated | `agentic-sdd` and `agentic-sdd-tui` forward to the canonical pair for one deprecation window (`cmd/agentic-sdd/main.go`) | that the names survive the window; no removal release is announced here |
| Experimental entrypoints | experimental | `opencode`-family commands are built only with the `agentic_sdd_experimental` tag and are excluded from default builds (`cmd/agentic-sdd/opencode.go`, `cmd/agentic-sdd/opencode_disabled.go`) | any default availability, supported use or native independent review |
| Optional OpenCode pack | implemented | `agentic-sdd-orchestrate` plus 14 agent profiles, four free-first presets and two fleet-only adapters excluded from ordinary pack install (`internal/skillsync/package_test.go`) | provider availability, or that a loaded profile behaves as its text describes |
| Model routing and fallbacks | documented policy | curated routing table and written ladders (`internal/appops/routing.go`, `opencode-pack/routing.md`) | automatic failover, quota, pricing or provider availability |
| Fleet engine and CLI | implemented, acceptance pending | parse, resolve, render, plan, apply, restore and local-only doctor in `internal/fleet/`; plan identity and locks | that the owning feature is accepted (`specs/014-agent-fleet-model-overrides/tasks.md`); the shipped TUI surface is preview-only (`specs/015-fleet-manager-tui/tasks.md`) |
| MCP configuration transactions | implemented | digest-bound plan/apply, entry-level backup and restore, `{env:NAME}` placeholders (`internal/skillsync/mcp*.go`); wave state in `specs/015-fleet-manager-tui/tasks.md` | authentication, connectivity or that a server works |
| Execution `run` / `task` / `review` | implemented, human review pending | private run scratch, brief integrity, check evidence and review packages (`internal/execution/`, [Execution run reference](../reference/execution.md)); state in `specs/016-agent-execution-context-evidence-recovery/tasks.md` | sandboxing, reviewer isolation, approval or tamper-proof storage |
| Release v0.11.0 metadata | verified, 2026-11-14 | public release, tag and cask metadata checked unauthenticated; see `specs/020-documentation-ecosystem/evidence.md` | archive member contents, executable protocol identity and a live macOS install |
| Documentation site | implemented, unpublished | pages exist in the repository | a live site: `has_pages: false` and a 404 response were observed on 2026-11-14 |

Counts are the counts in this checkout. Package tests assert the packaged sets, not a
catalogue count, and they change when the repository changes.

## Platform and release reality

- **macOS** is the only prebuilt package path: a paired archive and a Homebrew cask. The
  metadata is verified; extracting, running and upgrading that archive on a clean machine
  is not.
- **Linux** is source-only. There is no prebuilt Linux package, and the main Linux
  entrypoint is the TUI built from source together with its CLI partner.
- **Windows** is unsupported, and its behaviour is unknown; no page claims support there.
- Release metadata is not a capability list. A tag proves a tag exists.

See [CLI reference](../reference/cli.md) for command-level contracts and
[Configuration reference](../reference/configuration.md) for selectors and precedence.

## Historical and excluded work

Some repository records describe work that is deliberately not selectable as supported
behaviour. They are retained as history, not as a roadmap or a work queue:

- native ACP, admission, online and runtime-certification experiments (Feature 005);
- native review profiles, dispatch, pilot and C2-mitigation work (Feature 006);
- the old-base Taskfile migration and transition exercises (Feature 007).

The fence is recorded in `AGENTS.md` and owned by
`specs/008-core-release-quality/spec.md` and `specs/008-core-release-quality/tasks.md`.
Those records are historical evidence: being committed, or appearing complete in an old
plan, does not re-enable the work or make it supported. No page on this site offers a
procedure for them, and no test selection is expected to include them.

## No support follows from a specification

A feature's `spec.md`, `plan.md` and `tasks.md` record intent, decisions, batches and
review state. They can exist for work that is implemented but unaccepted
(Feature 016 execution), accepted in part (Feature 015 fleet and MCP waves) or pending
human review (Feature 019 canonical naming). Read them as the owning record of state, not
as certification. In particular:

- an approval section records a human decision about a batch, not runtime verification;
- a checked task records that work was performed, not that a client, provider or platform
  was observed working;
- a passing unit test proves the tested contract, not end-to-end behaviour;
- a summary in README, a handover or a roadmap is a pointer, never approval.

## Documented gaps you may hit

- **Top-level CLI help under-describes MCP.** `asdx-core --help` summarises `mcp` as
  read-only and omits `apply` and `restore`, while `mcp help` and the dispatcher implement
  them (`cmd/agentic-sdd/main.go` versus `cmd/agentic-sdd/mcp.go`). Trust the family help.
- **No uninstall command.** Removal is a deliberate manual or restore-based action; there
  is no "delete the whole home" cleanup, and that refusal is intentional.
- **No team administration.** There is no daemon, server, organisation policy engine or
  CI enforcement. Two collaborators share conventions through Git, not through a service.
- **No automatic model failover.** Written ladders are policy; the parent agent decides
  and reports what actually ran.
- **No isolation claims.** The installer, the run scratch and prompt-level permission
  rules are not a sandbox, and digests are corruption checks rather than signatures. See
  [Trust boundaries](../security/trust.md).
- **No measurement baseline.** No committed coverage percentage, performance budget or
  regeneration tool exists, so none is published here.
- **Two-file interruption needs reconciliation.** A recovered execution run can require
  an explicit, human-authorized reconciliation step; it is never repaired automatically.

## Keeping this page honest

Update a row only when the owning evidence changes: a new dated observation, an accepted
ledger change or a source change that moves the status. Do not upgrade a status because a
page was published or a specification was approved. When a limitation here is resolved,
say which observation resolved it and keep the historical sentence out of the new claim.
