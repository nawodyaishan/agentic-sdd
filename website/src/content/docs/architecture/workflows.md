---
title: Architecture workflows
description: Install transactions and recovery, skill discovery, optional delegation, and why fleet and MCP stay separate.
---

This page follows three workflows over time: installing skills, installing the opt-in orchestration pack, and writing fleet or MCP configuration. They share one backup engine but not one lock, one target, or one owner.

## Diagram D03 — install and recovery

The static projection for D03 is owned by the diagram lane and is not rendered in this worktree, so no image is embedded. Its text alternative:

- Source: `sourceFS(repo)` returns either the embedded `skills-files` tree or an explicit `--repo` checkout.
- Discovery: `discoverSkills` reads top-level `agentic-sdd-*` directories.
- Plan: `planChanges` compares each discovered skill against each of the four `clientTargets(home)` destinations.
- Preview stops here. `SyncWithOptionsResult(..., apply=false, ...)` writes only the transcript.
- Apply: `installChangesWithHook` creates the backup directory, copies every destination it will replace, writes `manifest.json`, stages every new tree, then renames each stage into place while keeping an undo entry.
- Failure after staging calls `rollback(changes, done)` and `cleanupStages`, and reports a `RollbackInfo`. Failure before a complete backup reports no backup ID, because the ID is set only after the manifest is written.
- Recovery: `restore` reads that manifest and plans a new transaction through the same engine, taking its own backup of the current destinations first.

## The install transaction, step by step

All steps are in `installChangesWithHook` (`internal/skillsync/sync.go`) unless noted.

| Step | What happens | Failure behavior |
|---|---|---|
| 1 | A UTC timestamp names the backup directory under `backupRoot`; the root is created `0700` or refused if it is not a directory | Nothing written |
| 2 | Every planned replace/remove is copied into the backup under `<client>/<skill>` | Nothing installed; partial backup tree may remain |
| 3 | `manifest.json` is written (format 2, or 3 when agent files are included) with home, targets, entries, digests and tool identity | Nothing installed |
| 4 | The `afterBackup` hook runs; fleet uses it to write `fleet-manifest.json` | Aborts with no destination modified |
| 5 | Each new tree or file is staged inside the destination directory and chmodded (`0755` directories, `0644` files) | Stages are removed; no destination touched |
| 6 | Each stage is renamed over its destination; an existing destination is first renamed to an undo path | `rollback` restores completed renames and the current undo |
| 7 | Undo entries are removed and the transcript reports the backup path | — |

Two consequences matter for recovery:

- A backup ID exists only for a complete, restorable backup. `writeOutcome.backupID` is assigned immediately after the manifest is written and before any destination changes, so a later failure still tells you what to restore.
- The backup root differs by source. `backupRootFor` puts backups at `<repo>/backups` when `--repo` was given and at `<home>/.agentic-sdd/backups` for the embedded source. A restore must use the matching source, because `loadBackup` compares the manifest's targets and home with the current ones.

`runLoggedOperation` (`cmd/agentic-sdd/logs.go`) opens the operation log before the engine runs, writes the command, source, home, result and timestamps, and syncs and closes it before reporting. A log write failure changes the exit code, so a successful install with an unflushed log is not reported as success.

## Restore is planned, validated and backed up

`restore` is not a naive file copy. `skillsync.RestoreResult` (preview and apply variants) resolves the backup through `loadBackup`, which:

- validates the backup ID shape with `ValidateBackupID`;
- requires the backup directory to be a directory, and `manifest.json` to be a regular file;
- requires the manifest ID to match the directory and the recorded client targets to equal `clientTargets(home)` for the current home;
- for format 3, requires the recorded OpenCode agent target to match the current one;
- allowlists top-level entries, refuses symlinks and special files, and requires every saved skill to contain a regular `SKILL.md`.

The restore itself goes back through `installChangesWithHook` with `operation.kind == "restore"`, so the destinations it overwrites are backed up first. That is why the transcript can say "Restored N skills; backup: &lt;new id&gt;". There is no built-in uninstall command; a restore of a backup taken before the install is the supported undo. See [`reference/files-and-backups`](../reference/files-and-backups.md) for the backup layout and restore preconditions.

## Skill discovery rules

`discoverSkills` is deliberately strict, because the installer must not guess:

- only top-level entries whose name begins with `agentic-sdd-`;
- the entry must be a directory;
- `checkTreeFS` walks it and rejects symlinks and special files;
- `SKILL.md` must exist and be a regular file;
- names are sorted; an empty result is an error, not an empty install.

`planChanges` then treats an existing non-directory destination, or a destination tree containing a symlink, as a refusal. When a destination already matches the source, `sameSourceTarget` compares structure, size and SHA-256 content; modes are not compared because embedded files report a synthetic mode. That comparison is what makes a re-run a no-op: `OutcomeNoChange` with no backup. `TestOpsHostProductionEngineInstallsNoOpThenReplaces` in `experiments/tui/ops_host_test.go` exercises exactly that sequence through the TUI host.

## Diagram D04 — optional delegation

D04's assets are unrendered here; its text alternative:

- A human runs an install with `--with-opencode` (or the TUI default). `SyncWithOptionsResult` adds the orchestration skill, every file in `opencode-pack/agents`, and, with `--opencode-free-first`, the four preset files, all under `~/.config/opencode/agents`.
- The tool stops there. It installs instruction files; it does not start an agent, choose a model, or call a provider.
- Later, a human opens a session with the `sdd-orchestrator` profile. That profile loads `agentic-sdd-orchestrate`, classifies the request, writes a bounded brief, and delegates to a child.
- The child returns a structured claim. The parent inspects the real diff and command results instead of trusting the prose.
- Approval, destructive actions, releases and credentials stay in the human-facing session.

## What delegation actually installs

The pack is optional and separate from the 18 core skills:

- `--with-opencode` installs `agentic-sdd-orchestrate` into every selected client skill directory and copies `opencode-pack/agents/*.md` into `~/.config/opencode/agents`.
- `adapters/` is not installed by that flag; those are fleet-only render sources. `presets/` is installed only with `--opencode-free-first`. `TestWithOpenCodeExcludesFleetAdapters` pins the adapter exclusion.
- Agent frontmatter carries `description`, `mode`, an optional pinned `model`, `permissions`, and `steps` where relevant. `TestOpenCodePackAssetsAndPermissions` validates the shape, the OpenCode v2 `permissions:` spelling, deny-by-default ordering and resolvable links.
- Permissions are instruction-layer rules, not an operating-system sandbox. Read-only profiles open with a deny-all rule; only writer/test profiles and documentation profiles include `edit`. Every shipped profile except the primary `sdd-orchestrator` explicitly denies `subagent`, so a child cannot delegate further. `opencode-pack/README.md` states the same boundary.
- A written model ladder is policy, not automatic failover. `internal/appops/routing.go` presents it as read-only display metadata and repeats that provider access, credentials, quota and runtime use are not claimed. The actual routing table is `opencode-pack/routing.md`, and a test compares the two.

## Fleet and MCP are separate transactions

They look similar — both write configuration for an optional OpenCode workflow — but they share no target, lock, backup store or ownership record.

| Aspect | Fleet | MCP |
|---|---|---|
| Input | Explicit personal `fleet.json` (`LoadConfigFile`, schema 1, 1 MiB cap) | Existing OpenCode config `~/.config/opencode/opencode.json` or `.jsonc` |
| Target | `~/.config/opencode/agents` (`TargetRootForHome`) | Managed entries in that one config file |
| Plan identity | Canonical JSON plus `ComputePlanID`; `ApplyPlan` rechecks plan ID, renderer version, target root and config digest | Digest-bound plan and apply token; `mcp_transaction.go` rechecks the target digest |
| Lock | `${backupRoot}/.fleet.lock`, kernel `flock` (dead process releases it) | `.agentic-sdd/mcp/.lock`, 30-second stale window |
| Backup | Shared skillsync backup plus `fleet-manifest.json` (format 4) | Private managed-entry snapshot, never unrelated servers or credentials |
| Recovery record | Fleet sidecar + live pointer `<home>/.agentic-sdd/fleet/current.json` | `intent.json` states `prepared`, `renamed`, `manifest-written`, `complete`, `rolled-back`, `needs-recovery` |
| Never writes | `AGENTS.md`; symlinks; unknown `disabled:` fields | Unmanaged servers; project-local scope; beyond the global config directory |

The fleet engine reaches the shared writer through `skillsync.ApplyAgentFiles`; MCP has its own transaction code in `internal/skillsync/mcp_transaction.go`. Neither reads the other's manifest, which is why a fleet restore cannot silently revert MCP servers and an MCP restore cannot touch agent profiles.

## When a workflow stops

| Condition | Result |
|---|---|
| Preview | `OutcomePreview`, no backup, no destination change |
| Everything already matches | `OutcomeNoChange`, no backup |
| Any destination refuses (non-directory, symlink, special file) | Error before staging; no destination modified |
| Failure after a complete backup | Error plus `BackupID`; `Rollback` reports what was attempted |
| Restore of a backup from another home or mismatched targets | Refused by `loadBackup` |
| Fleet plan revalidation failure | `ErrStalePlan` or `ErrPlanConflict`, no write |
| MCP target digest mismatch | Refusal with no write |

Related: [`architecture/evidence`](evidence.md) for the run-side workflow, and [`reference/fleet`](../reference/fleet.md), [`reference/mcp`](../reference/mcp.md) for the command-level detail.
