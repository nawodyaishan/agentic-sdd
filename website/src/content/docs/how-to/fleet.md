---
title: Set up an agent fleet
description: Choose an explicit fleet source, change models or enablement, plan, apply, and recover from drift or conflicts.
---

Fleet keeps one personal `fleet.json` that binds instance IDs to canonical OpenCode pack sources plus optional model overrides, then installs the rendered profiles into the OpenCode agent directory. This page is a CLI walkthrough. The shipped TUI exposes Fleet Manager as **preview only**, so every write below is an `asdx-core` command you run yourself. Field-level grammar is in [Fleet reference](../reference/fleet.md); flags and exit codes for the whole family are in [CLI reference](../reference/cli.md).

## 1. Choose the source explicitly

- `fleet status`, `fleet plan` and `fleet doctor` all take `--config PATH`. `status` and `plan` require it; nothing defaults the path and no product environment variable selects one (`cmd/agentic-sdd/fleet.go`, `cmd/agentic-sdd/fleet_plan.go`, `internal/fleet/plan.go`).
- `--home PATH` scopes every derived path: the target root `<home>/.config/opencode/agents`, the backup store `<home>/.agentic-sdd/backups`, and the live pointer `<home>/.agentic-sdd/fleet/current.json` (`internal/fleet/manifest.go` `TargetRootForHome`, `internal/fleet/plan.go` `LivePointerPath`, `internal/fleet/apply.go` `ApplyPlan`). Every subcommand except `fleet doctor` accepts `--home`; it falls back to the OS user home, so pass an isolated directory while you experiment.
- `--repo PATH` changes where the canonical pack is read from: `skillsync.OpenCodePackFS` reads `<repo>/opencode-pack`, and an omitted `--repo` uses the pack embedded in the binary. It does not move the target root or the backup store, because fleet always resolves backups through `skillsync.BackupRootFor("", home)` (`internal/skillsync/sync.go` `backupRootFor`, `internal/fleet/apply.go`).

Keep the config outside the installed agent directory. Reading a personal source and writing rendered profiles are separate steps, and no CLI command rewrites the source you pass.

## 2. Write a schema-version-1 source

The smallest useful document declares the version and one instance. `model` may be omitted to inherit the source role's canonical model:

```json
{
  "schemaVersion": 1,
  "instances": {
    "orchestrator": { "source": "agent:sdd-orchestrator" }
  }
}
```

The parser is strict JSON (1 MiB cap, valid UTF-8, no duplicate keys, no unknown fields, no trailing data, nesting depth 32) in `internal/fleet/parse.go` and `internal/fleet/fleet.go`. YAML and JSONC are not accepted, and a permission, tool or routing key such as `permissions`, `shell` or `fallback` is rejected as an unknown field rather than merged into the rendered profile. The inert fixtures under `website/tests/examples/fleet/` cover one valid document, one minimal document, five rejected shapes, two unaccepted encodings and one stale-source example.

Before planning, read the resolved inventory:

```sh
asdx-core fleet status --config /path/to/fleet.json --home /path/to/isolated-home
```

Illustrative output (wording from `cmd/agentic-sdd/fleet.go`; not a captured run):

```text
Fleet Status:
  Configuration: /path/to/fleet.json (digest: 0000…)
  Target root:   /path/to/isolated-home/.config/opencode/agents
  Instances (1):
    - orchestrator:
        Model:
        Source:           agent:sdd-orchestrator
        State:            enabled
        Installed:        absent
        Selected primary: not selected (offline default)
        Runtime:          not checked (offline)
```

`status` is read-only and acquires no lock. `Installed` is one of the installed-state labels in [Fleet reference](../reference/fleet.md); `Selected primary` and `Runtime` stay offline defaults because `status` never contacts a provider or runs a model.

## 3. Change a model or the enablement state

- **Model.** `model` is `<provider>/<model>` with an optional `#variant` suffix, for example `opencode-go/glm-5.3-flash` or `opencode/deepseek-v4.1-flash-free`. An empty, omitted or `null` model leaves the canonical source model untouched; it is not a "clear the model" instruction.
- **Enablement.** Omit `enabled` (or set `null`) to keep the instance enabled. `"enabled": false` marks it for removal, and that removal still obeys ownership: an unmanaged same-name file is preserved, while a clean managed profile is removed only after its bytes are saved in the transaction backup (`internal/fleet/plan.go` `classifyDisabled`, `internal/fleet/apply.go`).
- **Naming.** Instance IDs match `^[a-z0-9][a-z0-9-]{0,63}$`. `spec`, `plan` and `vision` are accepted but reserved, and both `plan` and `status` warn about them instead of blocking.

Fleet rewrites exactly one top-level `model:` scalar and refuses any render that would change a second frontmatter field or the permission block (`internal/fleet/render.go` `Render`, `VerifyModelOnly`). It cannot be used to change tool permissions, fallbacks or routing policy.

## 4. Preview without writing

```sh
asdx-core fleet plan --config /path/to/fleet.json --home /path/to/isolated-home
```

`plan` performs no write at all unless `--output PATH` is given. It reports one operation per instance — `create`, `update`, `remove`, `noop` or `conflict` — and ends with a line such as:

```text
Summary: 1 to create, 0 to update, 0 to remove, 0 up to date, 0 conflict(s) — nothing was written.
```

When the plan contains a conflict, `plan` still prints the summary, writes the file if you asked for one, and exits `1` with `error: plan contains conflicts; resolve them before applying`. `plan` never resolves a conflict for you.

Save the plan when you intend to apply:

```sh
asdx-core fleet plan --config /path/to/fleet.json --home /path/to/isolated-home --output /path/to/fleet-plan.json
```

A saved plan is JSON with a `planId` that is the SHA-256 of a canonical payload, so the same inputs produce the same identity on a different machine (`internal/fleet/plan.go` `CanonicalBytes`, `ComputePlanID`). The plan embeds absolute paths, so save it with the `0600` permissions `SavePlan` applies and do not commit it or paste it into a report.

## 5. Apply the saved plan

```sh
asdx-core fleet apply --plan /path/to/fleet-plan.json --home /path/to/isolated-home
```

`apply` is the only fleet command that installs profiles. Before writing it recomputes the `planId`, re-reads the config from the plan's recorded path, re-renders every enabled instance, compares each target file against the plan's `lastKnownInstalled` digest, and refuses on any difference (`internal/fleet/apply.go`). Success prints the counts, the backup directory and the live pointer path; a plan whose operations are all `noop` prints `No changes: every managed agent profile already matches the plan.` and writes nothing.

A plan that was previewed and then invalidated fails closed. A stale plan exits `1`; a plan file that no longer matches its own identity is malformed and exits `2` (`cmd/agentic-sdd/fleet.go`, `cmd/agentic-sdd/fleet_plan.go`):

| Message fragment | Exit | Meaning | What to do |
| --- | --- | --- | --- |
| `plan ID mismatch` | `2` | The plan file was edited or is corrupt | Regenerate it with `fleet plan --output` |
| `configuration changed` | `1` | The source at the plan's `configPath` has a different digest | Re-plan from the current source |
| `canonical source changed` | `1` | A canonical pack source file changed | Re-plan against the current pack |
| `target file "…" changed since the plan was generated` | `1` | A rendered profile was edited or replaced on disk | Inspect it, then re-plan; adoption stays an explicit CLI decision |
| `fleet lock is held by another process` | `1` | Another fleet write holds `…/.agentic-sdd/backups/.fleet.lock` | Wait for it to finish, then re-run |

## 6. Recover installed profiles with restore

```sh
asdx-core fleet restore --backup BACKUP_ID --home /path/to/isolated-home
```

Restore reverses exactly the files one recorded fleet transaction touched. It requires a format-4 `fleet-manifest.json` sidecar in that backup, a transaction state of `applied`, and unchanged target bytes; a modified target is refused rather than overwritten, and a backup without a fleet sidecar is refused as unknown provenance (`internal/fleet/restore.go` `RestoreBackup`). The restore is itself backed up through the same engine, so a failed rollback leaves recoverable state.

The CLI has no fleet backup listing command. List `<home>/.agentic-sdd/backups` yourself or use the TUI Recovery view, which lists fleet backups through `fleet.ListFleetBackups`; `--backup` still takes the ID you picked. Restoring a backup is not `git revert`, and it never deletes files the transaction did not create.

## 7. Read drift before you change it

Fleet compares three digests per instance: the last applied digest recorded in the fleet sidecar, the digest currently on disk, and the freshly rendered desired digest. `plan` and `status` mirror the same three-state table, so they agree about conflicts (`internal/fleet/plan.go` `classifyEnabled`, `internal/fleet/status.go` `classifyInstalledState`).

| Observed state | Plan operation | Reason text you will see |
| --- | --- | --- |
| Managed, current equals last applied and desired | `noop` | none |
| Managed, current equals last applied but desired differs | `update` | none |
| Managed file missing | `conflict` | `managed file is missing; restore it or remove its recorded ownership before applying` |
| Managed file edited locally | `conflict` | `manual drift: installed file differs from the last applied digest` |
| Local edit and a new desired render | `conflict` | `manual drift conflicts with the desired change; no speculative overwrite` |
| Unmanaged file at the target path | `conflict` | `unmanaged file already exists; use --adopt after review` |
| Sidecar record with legacy or unknown provenance | `conflict` | `legacy/unknown provenance; use --adopt after review` |

Recovery order that matches the source: inspect the file, restore a managed backup when the previous bytes are what you want, or pass `--adopt` to `fleet plan` when you have decided to take ownership of the existing content. `--adopt` is only ever a CLI flag; the shared TUI preview request hardcodes `adopt=false` (Feature 015 FR-12), and there is no fix-all, force or bulk-repair path.

## 8. Diagnose model routes offline

```sh
asdx-core fleet doctor --config /path/to/fleet.json
```

`doctor` reads the routes declared in your config and labels each one from local evidence only (`internal/fleet/doctor.go` `RunDoctor`). With no `--opencode-bin`, every route is `not checked (offline)`. With an explicit `--opencode-bin`, it runs only `--help` and, when help advertises one, `models`, then labels the route `configured`, `catalog-visible`, `authenticated`, `unknown` or `unavailable`. Those labels describe local files and local command output. They never prove that a provider is reachable, that a model will answer, or that a subscription entitles you to a route; `openai/*` is a route string, not evidence of a ChatGPT plan. `doctor` exits `0` whenever it produces a report, including a report full of `unavailable` labels.

## 9. Know what the TUI does and does not do

The TUI Fleet Manager is useful for reading status, browsing instances, staging an in-memory model or enablement change, and reviewing a rendered file diff. In the shipped build it stops there:

- **Source save is deferred.** The screens can stage edits, but `experiments/tui/fleet_write.go` reports `source not saved` with the note that saving the personal source is a separate operation and that B4 source save is not authorized in this build.
- **No plan from an unsaved draft.** A fresh deployment plan is computed only from the saved source; with staged edits the review view says so instead of planning the draft.
- **Preview only.** Pressing apply or restore without the reviewed live capability answers `Preview only` and starts nothing. The capability is a build-time marker, not a preference, environment variable, config key or keystroke (`internal/buildinfo`, Feature 015 FR-20).
- **A live build differs at the confirmation, not at the engine.** With the live marker, the same review flow opens a default-No confirmation before dispatching the identical `appops` request; conflicts, missing home and stale previews still block it. No shipped binary is verified to carry that marker.
- **No adoption.** The TUI never constructs an adopting plan, so conflict recovery remains a CLI decision.

Because the TUI cannot save the source, the practical loop in the shipped build is: edit `fleet.json` in your own editor, then run `fleet plan` and `fleet apply` from the CLI.

## 10. Keep each store separate

Fleet's `fleet.json` source, its format-4 sidecar, the skill backup store and the MCP ownership store are distinct schemas in distinct roots. A skill restore never reverses a fleet transaction, and a fleet restore never restores skills. Locations and formats are in [Files and backups reference](../reference/files-and-backups.md); the fleet backup root is `<home>/.agentic-sdd/backups`, the same root skills use when assets are embedded, which is why the sidecar format — not the directory name — decides who owns a backup.
