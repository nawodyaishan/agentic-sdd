---
title: Fleet reference
description: Fleet plans, model-only rendering rules, apply and restore behavior, drift signals, and recovery for optional OpenCode agent instances.
---

The generated section below is the source-derived inventory of the `fleet.json` schema, its model grammar, the CLI surface and the build capability marker. The curated sections after it explain the vocabulary those tables use, so a plan, a status line and a doctor report can be read without guessing. For the step-by-step task, see [Set up an agent fleet](../how-to/fleet.md).

<!-- generated:reference/fleet:start -->
Source-derived fleet facts for the current checkout. Preview planning is read-only; apply and restore remain explicit write operations.

## `fleet.json` schema

Only schemaVersion 1 is accepted. The document is strict JSON, bounded to 1048576 bytes (1 MiB).

| Root field | Contract |
| --- | --- |
| schemaVersion | Must equal 1 |
| fleet | Optional fleet label; no path separators or traversal |
| preset | Optional preset label; no path separators or traversal |
| instances | Map of instance ID to instance object |

| Instance field | Contract |
| --- | --- |
| source | Required one-hop reference matching `^(agent\|preset):([a-z0-9][a-z0-9-]*)$` |
| model | Optional override matching `^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$` |
| enabled | Optional boolean; omitted or null defaults to true |

Instance IDs match `^[a-z0-9][a-z0-9-]{0,63}$`. Reserved instance names are `spec`, `plan`, `vision`.

- valid UTF-8 required
- duplicate JSON object keys rejected
- unknown fields rejected
- trailing data rejected
- JSON nesting depth limited to 32

## Model grammar

`^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`

An empty or omitted model leaves the canonical source model untouched. The optional `#variant` suffix is part of the same model token, and empty path segments are rejected.

## Source references

- Allowed namespaces: `agent` and `preset`
- Walked source roots: `agents`, `adapters`, `presets/free-first`
- Exactly one hop is resolved; instance-to-instance aliases and cycles are rejected
- A name may exist in exactly one walked root; duplicate agent/adapter names fail resolution
- Unknown sources fail instead of rendering an empty agent

## Target and sidecar

| Item | Contract |
| --- | --- |
| Scope | `global` |
| Target root | `<home>/.config/opencode/agents` |
| Sidecar file | `fleet-manifest.json` inside the backup directory |
| Sidecar format | format 4 |
| Transaction states | `applying`, `applied`, `failed` |
| Per-file provenance | `managed`, `adopted`, `restored`, `legacy`, `unknown`, `removed` |

The sidecar fields are `format`, `id`, `operation`, `restoredFrom`, `state`, `createdAt`, `home`, `backupRoot`, `backupPath`, `targetRoot`, `scope`, `sourceRev`, `configDigest`, `rendererVersion`, `rollback`, `files`. Per-file records carry `instanceID`, `sourceRef`, `sourceDigest`, `configDigest`, `rendererVersion`, `model`, `desiredDigest`, `installedDigest`, `backupRef`, `backupDigest`, `scope`, `target`, `provenance`. Fleet never writes the user's `AGENTS.md`; names are bare `.md` file names under the target root.

## CLI

| Command | Flags | Required |
| --- | --- | --- |
| `fleet status` | `--config`, `--home`, `--repo` | `--config` |
| `fleet plan` | `--config`, `--output`, `--adopt`, `--home`, `--repo` | `--config` |
| `fleet apply` | `--plan`, `--home` | `--plan` |
| `fleet restore` | `--backup`, `--home` | `--backup` |
| `fleet doctor` | `--opencode-bin`, `--config` | none |

Documented exit codes are 0 (success), 1 (runtime, lock, stale plan or unmanaged conflict) and 2 (usage, flags, invalid configuration or malformed plan).

## TUI build capability

The Fleet Manager live controls are a build-time marker, not a user preference. Valid markers are `preview` and `live`; the shipped default is `preview`, and an absent, malformed or unknown marker fails safe to preview. The marker is injected with `-X agentic-sdd/internal/buildinfo.FleetCapabilityMarker`; there is no environment, config, CLI or key override.

## Separate contracts

The `fleet.json` schema (version 1) and the `fleet-manifest.json` sidecar (format 4) are distinct from the skill backup `manifest.json` formats 1-3 and from the MCP ownership store. They do not share a schema version.
<!-- generated:reference/fleet:end -->

## Instance, source, rendered, installed

Four different objects share the word "profile". Keeping them apart is what makes a plan readable.

| Term | Meaning | Owning code |
| --- | --- | --- |
| Instance | One ID in your `fleet.json` `instances` map; the only thing you name | `internal/fleet/fleet.go` `Config`, `Instance` |
| Source | The canonical pack document the instance points at, via `agent:` or `preset:` | `internal/fleet/resolve.go` `Resolve` |
| Rendered | The exact bytes fleet would install: source bytes with only the top-level `model:` scalar replaced | `internal/fleet/render.go` `Render` |
| Installed | Whatever currently sits at `<home>/.config/opencode/agents/<instance>.md` | `internal/fleet/plan.go` `CurrentTargetHashes` |

An instance is never a source alias for another instance: exactly one hop is resolved, and instance-to-instance aliases, cycles and unknown sources fail instead of rendering an empty agent.

## Digests and identity

Every fleet decision is a digest comparison. None of them is a signature or an approval.

| Value | What it hashes |
| --- | --- |
| `sourceDigest` | The canonical pack source bytes for one instance |
| `configDigest` | The exact `fleet.json` bytes that were parsed |
| `desiredDigest` | The rendered bytes fleet would install |
| `installedDigest` / current digest | The bytes observed at the target path right now |
| `lastAppliedDigest` | The `desiredDigest` recorded by the last applied transaction |
| `sourceRev` | Instance ID, source reference and `sourceDigest` for every instance, including disabled ones |
| `planId` | A canonical plan payload, excluding machine-specific paths |

`fleet status` also prints a truncated last-applied digest so two runs can be compared by eye. The authoritative values are always the full digests in the plan and sidecar JSON. `internal/fleet/plan.go` owns `sourceRevision`, `CanonicalBytes`, `ComputePlanID` and `Revalidate`.

## Config grammar rules the table cannot show

- `instances` may be absent or empty. A zero-instance document is valid and plans nothing; it is not an error.
- `model` is normalized only by validation. An empty string, an omitted key and an explicit `null` all mean "inherit the canonical model". There is no empty-string meaning of "remove the model".
- `enabled` is a pointer: omitted or `null` is true, and only an explicit `false` disables.
- `spec`, `plan` and `vision` are valid instance IDs and are reported as reserved-name warnings.
- Permission, tool-policy and routing keys — `permissions`, `shell`, `execute`, `subagent`, `fallback`, `fallbacks`, `automaticDelegation`, `spending`, `providerSwitching` — are rejected as unknown fields at the top level and inside an instance. Fleet cannot be used to rewrite a permission block or to declare a model fallback chain.
- Only strict JSON is accepted. There is no comment or trailing-comma path, so a `.yaml` or `.jsonc` file fails as invalid JSON rather than being normalized. `website/tests/examples/fleet/` holds one accepted document, one minimal document, five rejected shapes, two unaccepted encodings and one stale-source example.

Diagnostics name fields and instance IDs only; the parser never echoes the document body (`internal/fleet/parse.go`).

## Source resolution

`Resolve` walks three allow-listed roots of the canonical pack: `agents/`, `adapters/` and `presets/free-first/`. Adapters are reviewed fleet-only render sources that share the `agent:` namespace with canonical agents, so a name may exist in exactly one of `agents/` and `adapters/`; a duplicate name fails resolution. Only `.md` files whose base name matches the source grammar are candidates, and a non-regular entry named `*.md` is refused rather than followed (`internal/fleet/resolve.go`).

Because resolution reads an `fs.FS` — the embedded pack or `<repo>/opencode-pack` — planning does not require the pack's profiles to be installed first. It does require a source that exists: an unknown reference fails with `unknown source`.

## Plans

A plan is a JSON document with `planId`, `sourceRev`, `configDigest`, `configPath`, `repoPath`, `scope`, `rendererVersion`, `targetRoot`, `fileList`, `lastKnownInstalled`, `proposedOps` and `permNotes` (`internal/fleet/plan.go` `Plan`). Identity is computed over `sourceRev`, `configDigest`, `rendererVersion`, `targetRoot`, sorted `fileList`, `lastKnownInstalled`, sorted `proposedOps` and sorted `permNotes`; `configPath` and `repoPath` are excluded so the same content and configuration produce the same `planId` on another machine.

Operation names are `create`, `update`, `remove`, `noop` and `conflict`. `remove` means a disabled instance whose managed file should be backed up and taken away; it never means "delete whatever is at this path". `conflict` is a proposal that requires a human decision and is never resolved by planning.

Saved plans are written `0600` in a `0700` directory and read back with a 4 MiB cap; `LoadPlan` refuses non-regular files and recomputes the `planId`, so a hand-edited plan is a malformed plan (exit `2`) rather than a stale one.

`fleet apply` re-checks the plan before any write and fails closed on: a `planId` mismatch, a `rendererVersion` change, an unsupported `scope`, a changed `targetRoot`, a missing `configPath`, a configuration digest change, a canonical source change, a changed `sourceDigest`, a changed model, a rendered-bytes change, a file-list mismatch with the configuration, and any target file whose digest differs from `lastKnownInstalled` (`internal/fleet/apply.go`). Planning and applying are separate actions; a `planId` is a consistency token, never an approval.

## Drift and ownership records

Ownership lives in a format-4 sidecar, `fleet-manifest.json`, inside each backup directory, plus a live pointer at `<home>/.agentic-sdd/fleet/current.json` that names the current fleet state. If the pointer is missing, fleet rebuilds the recorded state by replaying fleet transactions instead of treating every managed file as unmanaged (`internal/fleet/manifest.go` `RebuildLiveManifest`).

Per-file provenance is a six-value vocabulary, and each value changes what a later plan is allowed to assume:

| Provenance | Meaning | Later plan behavior |
| --- | --- | --- |
| `managed` | Fleet installed these bytes in a recorded transaction | Compared against last applied and freshly desired |
| `adopted` | You explicitly took ownership of existing bytes | Counted as managed once its record carries a desired digest |
| `restored` | Bytes came back from a verified backup | Compared like managed, against the restored content |
| `legacy` | An older record with no trustworthy last-applied state | Requires `--adopt`; never treated as clean |
| `unknown` | A record exists but provenance cannot be established | Requires `--adopt`; never treated as clean |
| `removed` | A managed removal was recorded in that transaction | Not live-managed afterwards; a restore refuses a recreated file |

Transaction state is `applying`, `applied` or `failed`. `applying` means the backup is complete but the transaction did not report success; it is recoverable state, not a claim that files were installed. Restore accepts only `applied`.

## Installed-state labels

`fleet status` prints an `Installed` value derived from the same three-state comparison as a plan (`internal/fleet/status.go` `InstalledState`):

| Label | Meaning |
| --- | --- |
| `absent` | No managed file and nothing to remove |
| `managed-current` | Installed bytes match both the last applied and the freshly rendered digest |
| `pending-apply` | Installed bytes match the last applied digest while the desired render differs |
| `managed-drifted` | A managed file is missing or no longer matches the last applied digest |
| `unmanaged` | A file exists that fleet does not own |
| `legacy-unknown` | A record exists whose provenance is legacy or unknown |
| `unsafe-target` | The installed path is a symlink or another special file; it is never followed |
| `disabled` | The instance is disabled while its managed file is still installed |

The CLI maps several of these to the historical `installed (…)` wording, including `installed (unmanaged/legacy provenance unknown)` and `installed (unsafe target; refused)`. The label is a projection: it is not an approval, a security verdict or a runtime claim.

## Locks

Fleet serializes writes with an operating-system lock on `<backupRoot>/.fleet.lock`, where `<backupRoot>` is `<home>/.agentic-sdd/backups` because fleet resolves the store through `skillsync.BackupRootFor("", home)`. The lock is exclusive and non-blocking: a second writer gets `fleet lock is held by another process` instead of waiting. The recorded PID and timestamp are diagnostics for that message; liveness is decided by the kernel, and `LockStaleTimeout` (30 seconds) never reclaims a lock on age alone. The file is kept after release because the stable inode is what makes the lock atomic (`internal/fleet/plan.go` `AcquireLock`, `internal/fleet/lock_posix.go`, `internal/fleet/lock_windows.go`).

The process-local single write slot in the TUI (`OpsHost`, `experiments/tui/ops_host.go`) is a separate mechanism: it stops one TUI session from dispatching two operations at once and does not replace the cross-process lock.

## Restore

`fleet restore --backup ID` reverses exactly the files one recorded apply or restore touched. The read-only projection is `PreviewRestoreBackup`, which mirrors `RestoreBackup`'s validation order — malformed ID, home and backup-directory checks, missing or corrupt sidecar, non-`applied` state, manifest identity and target checks, symlink path components, the live-pointer check, and every per-record digest, provenance, recreated-removal and saved-content check — so a refusal in the preview is the same refusal as the writer (`internal/fleet/restore_preview.go`). `RestoreBackup` remains the only writer, revalidates the same state, and creates a pre-restore backup so a failed rollback leaves recoverable state.

A target modified after the recorded transaction is refused rather than overwritten, and a backup without a fleet sidecar is refused as unknown provenance. A restore writes its own format-4 sidecar with `operation: restore` and `restoredFrom`, so it can be inspected or reversed later. The CLI has no fleet backup listing subcommand; `fleet.ListFleetBackups` powers the TUI Recovery view and returns only backups that carry a valid sidecar.

## Doctor result vocabulary

`fleet doctor` labels each declared model route from local evidence only (`internal/fleet/doctor.go`):

| Label | Evidence behind it |
| --- | --- |
| `not checked (offline)` | The default: no local inspection tool was offered |
| `configured` | The provider appears in a local dump or config listing |
| `catalog-visible` | The route appears in a local catalog listing |
| `authenticated` | The provider carries an explicit local authentication marker (`authenticated`, `oauth`, `api-key`, `key`, `subscription`, `logged-in`) |
| `unknown` | The available evidence cannot decide: neither the route nor its provider appears in a local listing, or the route fails the offline grammar check |
| `unavailable` | A requested inspection tool failed or produced nothing |

`RunDoctor` never returns an error: inspection problems become `unavailable` labels plus sanitized failure notes, and the command exits `0` whenever a report is produced. The report states which local tools were consulted. These labels are local diagnostics, not connectivity or entitlement proof: a route string, a catalog listing or a provider entry never establishes that a provider is reachable, that a model will answer, or that a plan entitles you to it. `openai/*` is treated as a route throughout the source precisely to avoid that inference.

## Where this is implemented

- `internal/fleet/fleet.go` — schema constants, instance and source grammar, model validation
- `internal/fleet/parse.go` — strict decoding, duplicate-key and depth checks
- `internal/fleet/resolve.go` — source allow-list walking and one-hop resolution
- `internal/fleet/render.go` — model-only rendering and permission-set preservation
- `internal/fleet/plan.go` — plan identity, revalidation, lock, config loading
- `internal/fleet/apply.go` — the only apply writer, re-derivation under the lock
- `internal/fleet/manifest.go` — format-4 sidecar, live pointer, provenance vocabulary
- `internal/fleet/status.go` — read-only installed-state projection
- `internal/fleet/diff.go`, `restore_preview.go` — read-only file and restore previews
- `internal/fleet/restore.go` — the only restore writer
- `internal/fleet/doctor.go` — offline model-route labels
- `cmd/agentic-sdd/fleet.go`, `fleet_plan.go`, `fleet_doctor.go` — flag grammar, output and exit codes
- `internal/appops/fleet.go` — the typed CLI/TUI seam and preview tokens
- `experiments/tui/fleet*.go` — the read-only Fleet Manager screens
