---
title: Architecture overview
description: Components, who loads and writes what, and where authority actually lives in Agentic-SDD.
---

Agentic-SDD is one Go repository that ships two installed commands, an optional OpenCode pack and a documentation site. The important architectural fact is that **nothing in this repository approves work**. It copies files, records observations and starts one terminal program. Human decisions stay in the feature records a maintainer edits.

This page is the map. [`architecture/interfaces`](interfaces.md) covers the call seams, [`architecture/workflows`](workflows.md) covers install and delegation flows, and [`architecture/evidence`](evidence.md) covers runs, locks and check evidence.

## Components and their owners

| Component | Source | Responsibility |
|---|---|---|
| Embedded core assets | `skillsfiles.go` (`go:embed all:skills-files`) | 18 portable skills compiled into the binary |
| Embedded optional pack | `opencode_pack.go` (`go:embed all:opencode-pack`) | One orchestration skill, 14 agent profiles, four Free-first presets, two fleet-only adapters |
| Embedded inert resources | `adapterfiles.go` | Inactive config and evidence-schema files; embedding installs nothing |
| Installer engine | `internal/skillsync` | Source resolution, client destinations, planning, backup, staged write, rollback, restore, MCP transaction |
| Typed operation seam | `internal/appops` | Immutable requests, preview tokens, apply revalidation, fleet/MCP request types, stale-result identity |
| Fleet core | `internal/fleet` | Strict `fleet.json` parsing, source resolution, deterministic rendering, plan identity, apply/restore, format-4 sidecar |
| Execution service | `internal/execution` | Private run workspace, checkpoint, local checks, claims, review packages |
| Build identity | `internal/buildinfo` | CLI/TUI pair identity, `--protocol-info`, distribution and fleet capability markers |
| CLI | `cmd/agentic-sdd` | Flag parsing, dispatch, rendering and exit codes for `asdx-core` and legacy `agentic-sdd` |
| TUI companion | `experiments/tui` (nested module), `experiments/tui/cmd/internal/tuirun`, `experiments/tui/cmd/tui-companion` | The terminal UI installed as `asdx` |

The root module is `agentic-sdd` on Go 1.23 (`go.mod`); the TUI is a separate nested module, `agentic-sdd/experiments/tui` (`experiments/tui/go.mod`), so the CLI keeps a standard-library-only dependency contract. The package comment in `experiments/tui/cmd/tui-companion/main.go` states that contract.

### Diagram D01 — context and components

The static projection for D01 is owned by the diagram lane and is not rendered in this worktree, so no image is embedded here. Its intended content, as text:

- A human or an agent session runs either `asdx-core` (CLI) or `asdx` (TUI).
- `asdx-core tui` does not show a UI itself; it resolves, verifies and starts the sibling `asdx` process (`resolveTUICompanion` and `launchTUI` in `cmd/agentic-sdd/tui.go`).
- Both front-ends reach the same packages: `internal/appops` for install/restore/MCP writes, `internal/fleet` for fleet plans, `internal/execution` for run artifacts, and `internal/skillsync` underneath.
- The only sources of skill text are the embedded `skillsfiles.Files` and `skillsfiles.OpenCodePackFiles` trees or an explicit `--repo` checkout, resolved by `sourceFS` and `openCodePackFS` in `internal/skillsync/source.go`.
- Outputs are files outside the binary: four client skill directories, `~/.config/opencode/agents`, backup trees, the OpenCode MCP configuration and `.agentic-sdd/runs`.
- Approval is not a component. It is a human edit to a feature `spec.md`/`tasks.md`, outside every process arrow.

## Who owns loading

Loading is installer-owned. `sourceFS(repo)` returns the embedded subtree when `repo` is empty and `os.DirFS(repo)/skills-files` when it is set; `openCodePackFS(repo)` applies the same rule to `opencode-pack`. `skillsync.SyncWithOptionsResult` then discovers, plans and writes copies. No client loads the repository source tree at runtime: the four client directories are destinations produced by `clientTargets(home)`.

The terminal UI reads state through the same engine instead of its own scanner: `tui.LoadCatalog` (`experiments/tui/catalog.go`) calls `skillsync.InspectSelection` and turns the inventory into one row per managed ID with a per-client status. A missing pack becomes a notice, not invented content.

## Who owns execution (writes)

There is more than one writer, and the boundaries are narrow:

- `installChangesWithHook` in `internal/skillsync/sync.go` is the shared backup/stage/rename/rollback engine for core skills, the optional skill and agent profiles. `skillsync.ApplyAgentFiles` routes fleet profile writes through it.
- `internal/skillsync/mcp_transaction.go` owns the separate MCP configuration transaction: its own lock, snapshot, intent record and ownership manifest, targeting `~/.config/opencode` and `~/.agentic-sdd/mcp`.
- `internal/execution` writes only private run artifacts under a run root. Its package comment in `internal/execution/types.go` says it never approves work, calls a model or acts as a sandbox.
- The TUI never writes a destination directly. `tui.OpsHost.BeginWrite` claims one write slot and runs `appops` seams; `TestOpsHostRefusesSecondConcurrentWriteWithoutQueueing` in `experiments/tui/ops_host_test.go` proves a second write is refused rather than queued.
- Plan-only paths stay read-only: `skillsync.SyncWithOptionsResult(..., apply=false, ...)` is the preview variant used by `appops.PreviewInstall`.

### Diagram D02 — installer versus clients

D02's assets are also unrendered here; its text alternative:

- The installer is the only arrow into client destinations: `planChanges` → `installChangesWithHook` → rename into `clientTargets(home)`.
- A client (Claude, Codex, Gemini Antigravity CLI, or an `~/.agents` consumer) reads the copied `SKILL.md` files from its own directory. It does not call the installer.
- `asdx` hosts writes but does not reimplement one: its `OpsHost` calls the same `appops` seams. It is a process started by `asdx-core tui` or launched directly.
- The pair identity handshake (`buildinfo.VerifyCompanion`) runs before the TUI starts. A mismatch stops at the parent; it does not silently fall back to a preview or a different write workflow.

## Who owns authority

Authority is outside the code. The repository convention in `AGENTS.md` puts the single combined human approval in a feature's `spec.md` approval section and batch state in its `tasks.md`. The Go records agree:

- `RunIdentity.ApprovalReference` is labelled a reference, and `Scope` is documented as "a reference, never an approval" in `internal/execution/types.go`.
- `Checkpoint` is documented as "disposable observation, not an approval or batch ledger" in the same file.
- `internal/execution/checkpoint.go` states that a checkpoint never stores human approval, never authorizes a resume and never claims a next batch.
- `TestStatusNeverUsesApprovalLanguage` (`internal/execution/checkpoint_test.go`) fails if status output starts speaking in approval terms.

So the program can tell you what it observes. It cannot tell you that you may proceed. That answer is a human edit to the spec/tasks pair.

## Installer versus clients, in one paragraph

Skills are inert text. Installing them copies bytes from an embedded or explicit source into four client directories; it does not register the skill with a client, start an agent, call a model or grant a permission. Whatever a client does with a skill afterwards is outside this repository's control, which is why these pages never claim cross-client parity. See [`reference/skills`](../reference/skills.md) for the inventory and [`reference/files-and-backups`](../reference/files-and-backups.md) for exact paths and modes.

## What this page does not claim

- It does not claim the release archive, runtime identity or a live macOS install is verified; it describes this checkout.
- It does not promote the experimental `opencode`/`opencode-harness` commands or historical native-profile work to supported behavior.
- It does not describe TUI documentation links (a later, optional feature) or treat the documentation site as published.

Next: [`architecture/interfaces`](interfaces.md) for the actual call seams.
