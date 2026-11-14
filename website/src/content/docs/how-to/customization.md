---
title: Customization and local sources
description: Adopt an existing repository, install from a local --repo checkout, select supported profile overrides, and preserve unrelated files.
---

Agentic-SDD keeps customization strictly bounded: you can adopt an existing project's conventions manually, point `asdx-core` at a local `--repo` checkout, and apply reviewed model or preset overrides without rewriting permissions or touching unrelated files.

## Adopt an existing repository (`agentic-sdd-bootstrap`)

Use `agentic-sdd-bootstrap` (`skills-files/agentic-sdd-bootstrap/SKILL.md`) when you want an existing repository to use Agentic-SDD without renaming its documentation or erasing prior records. Bootstrap runs only on explicit request and does not invoke `asdx-core apply`.

1. **Discover existing canonical files:** Locate the repository's actual top-level requirements, technical specification, and roadmap (commonly `Docs/SRS.md`, `Docs/High Level Spec.md`, and `Docs/Tasks.md`, preserving existing spelling and case), plus any `specs/` folders and feature index.
2. **Add minimal missing pointers:** In an existing instruction entry point—or a new `AGENTS.md` only if a durable entry point is needed and none exists—record only what is missing:
   - Canonical top-level document paths.
   - `specs/<nnn-slug>/` for feature `spec.md`, `plan.md`, and `tasks.md`.
   - The single location recording combined human approval (by convention, an approval section in `spec.md`), plus where batch state and continuation notes live (`tasks.md`).
   - Relevant build/test commands and the boundary between developing code and executing live operations.
3. **Preserve existing history and features:** Keep existing `specs/` folders and earlier approval records intact—even if older features used per-stage approvals—and note their actual scope rather than rewriting them.
4. **Skip boilerplate:** Do not generate generic templates, role charters, empty directories, or a runner, and do not copy client invocation or MCP settings into portable repository instructions. A second bootstrap run with unchanged needs should be a no-op.

See the inert synthetic before/after trees in `website/tests/examples/research/adopt-before/` and `website/tests/examples/research/adopt-after/` for a minimal adoption example.

## Install from a local checkout (`--repo`)

By default, `asdx-core` reads the `skills-files/` and `opencode-pack/` trees compiled into the binary via `skillsfiles.go` (`internal/skillsync/source.go`). To preview or install modified or checked-out skill sources, pass `--repo PATH` (`cmd/agentic-sdd/main.go`):

```sh
asdx-core preview --repo /path/to/local-source --home /path/to/isolated-home
asdx-core apply --repo /path/to/local-source --home /path/to/isolated-home
```

`internal/skillsync/source.go` and `internal/skillsync/sync.go` enforce these source and destination rules:

- **Skill tree layout (`sourceFS`):** `asdx-core` reads `<repo>/skills-files/`. At least one `agentic-sdd-*` directory must exist, and each `agentic-sdd-*` directory must contain a regular `SKILL.md` file.
- **Optional OpenCode pack (`openCodePackFS`):** When `--with-opencode` is set, `asdx-core` reads `<repo>/opencode-pack/` (`skills/agentic-sdd-orchestrate/`, `agents/*.md`, and `presets/free-first/*.md` when `--opencode-free-first` is also passed).
- **Refusal of symlinks and special files (`checkTreeFS`):** Symlinks, sockets, FIFOs, and non-regular files anywhere inside source trees, target skill directories, or backup roots are rejected before any destination write occurs.
- **Mode normalization:** Copied directories receive `0755` and regular files receive `0644`, regardless of source permissions. Identical trees report `up to date` and perform no backup or rewrite.
- **Backup root difference (`backupRootFor`):** Embedded runs store backups under `<home>/.agentic-sdd/backups`, whereas explicit `--repo PATH` runs store backups under `<repo>/backups` (keep `backups/` local and out of Git).

Not every command accepts `--repo`: for example, `asdx-core fleet apply`, `asdx-core fleet restore`, and `asdx-core mcp` subcommands use subsystem-specific flags (`--plan`, `--backup`, `--config`, `--home`). See [CLI reference](../reference/cli.md) and [Configuration reference](../reference/configuration.md) for exact flag tables.

A minimal synthetic `--repo` layout is shown in `website/tests/examples/research/local-source/`. For standard skill installation steps without `--repo`, see [How to install and manage skills](../how-to/skills.md).

## Supported profile and model overrides

Agentic-SDD separates three supported levels of profile customization (`internal/skillsync/sync.go`, `internal/fleet/fleet.go`, `internal/fleet/resolve.go`), detailed in [Agent profile catalog](../reference/profiles.md) and [Fleet reference](../reference/fleet.md):

| Mechanism | What it selects or changes | Source owner |
|---|---|---|
| Core only (CLI default) | 18 core skills only; leaves `~/.config/opencode/agents/` untouched | `skills-files/` |
| `--with-opencode` (TUI default) | Adds `agentic-sdd-orchestrate` and the 14 profiles in `opencode-pack/agents/` | `internal/skillsync/sync.go` |
| `--with-opencode --opencode-free-first` | Overlays the 4 free-first writer/test templates (`sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2`); requires `--with-opencode` (otherwise exits `2`) | `opencode-pack/presets/free-first/` |
| `fleet.json` schema v1 + `asdx-core fleet` | Maps personal instance IDs (`agent:<name>` or `preset:<name>`) across `agents/`, `adapters/`, and `presets/free-first/` with an optional `model` (`<provider>/<model>[#variant]`) or `enabled` flag | `internal/fleet/` |

Notes on boundaries:

- Bare `asdx-core` runs core-only preview; bare `asdx` (TUI) defaults `--with-opencode=true` (pass `asdx --with-opencode=false` for core-only).
- The two fleet adapters (`sdd-spec-9r`, `sdd-plan-9r` in `opencode-pack/adapters/`) are resolved only by `internal/fleet/resolve.go`; `--with-opencode` does not install them.
- Shipped TUI Fleet Manager controls remain `preview`-only unless a reviewed binary sets the build-time `FleetCapabilityMarker` to `live` (`../reference/fleet.md`). Use [How to manage a fleet](../how-to/fleet.md) for the CLI `fleet plan` and `fleet apply` workflow.

## Fleet model-only render limits vs policy rewrites

It is critical not to confuse **configuration selection** with a **permission or model-policy rewrite**:

- **What `fleet` allows (`internal/fleet/render.go`):** `Render` + `VerifyModelOnly` take canonical source bytes and modify **only** the single top-level `model:` scalar (or insert `model:` after `mode:` or at the end of frontmatter if absent). Leaving `model` empty inherits the canonical source model. Duplicate `model:` lines and YAML block scalars (`|` or `>`) are rejected.
- **What `fleet` refuses (`VerifyModelOnly` and `parsePermissions`):** The renderer fails closed if the prompt body bytes change, if any non-`model` frontmatter field changes, or if the normalized `permissions:` set (`action`, `resource`, `effect` in the pack's fixed YAML subset) differs from the canonical source.
- **Configuration vs policy rewrite:**
  - Passing `--repo`, `--with-opencode`, `--opencode-free-first`, or overriding `model:`/`enabled:` in `fleet.json` is **configuration** within reviewed sources.
  - Editing `permissions:`, `mode:`, `steps:`, prompt bodies, or the pack's documented fallback rules (`opencode-pack/routing.md`) is a **policy or source rewrite**. Portable skill text cannot grant runtime permissions (`skills-files/agentic-sdd-router/references/workflow-policy.md`), and changing a pinned `model:` string does not create provider availability, credentials, or automatic failover.
  - If you intentionally maintain modified profile policies, store them in your own reviewed `--repo` checkout and verify them directly rather than expecting `fleet.json` to rewrite permissions.

See `website/tests/examples/research/local-source/fleet-override.json` for an inert synthetic `fleet.json` example using a placeholder model string.

## How unrelated files and separate stores are preserved

Both `skillsync` and `fleet` isolate managed paths from your existing client setup (`internal/skillsync/sync.go`, `internal/fleet/apply.go`):

- **Unrelated skills and agent files:** `asdx-core apply` manages only `agentic-sdd-*` directories under the four client skill destinations (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`) and, with `--with-opencode`, the canonical `.md` filenames under `~/.config/opencode/agents/`. Custom skills, unrelated profiles (such as local `vision.md` or non-pack agents), `~/.config/opencode/opencode.json`, and `~/.config/opencode/AGENTS.md` are never modified.
- **No CLI `--clients` filter:** `asdx-core` has no `--clients` flag; selecting a subset of the four client destinations is available only in the TUI (`asdx`).
- **Unmanaged collisions in `fleet`:** `asdx-core fleet apply` refuses to overwrite existing unmanaged agent files unless they were explicitly reconciled via `asdx-core fleet plan --adopt`, and `fleet` never writes `AGENTS.md`.
- **Distinct backup and manifest stores:** Skill backups (`manifest.json` formats 1–3), fleet sidecars (`fleet-manifest.json` format 4), MCP ownership manifests, and `.agentic-sdd/runs/` execution scratch live in separate contracts and are not interchangeable (see [Files and backups](../reference/files-and-backups.md) and [How to update and recover](../how-to/updates-and-recovery.md)).

## Next

- Check all command flags and defaults in [CLI reference](../reference/cli.md) and [Configuration reference](../reference/configuration.md).
- Inspect fleet schema and render contracts in [Fleet reference](../reference/fleet.md) and [How to manage a fleet](../how-to/fleet.md).
- Review optional pack delegation boundaries in [How to use optional orchestration](../how-to/orchestration.md).
