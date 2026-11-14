---
title: Agent profile catalog
description: "The optional OpenCode orchestration pack: orchestrator and subagent profiles, free-first presets, fleet adapters, and routing policy."
---

Source-derived profile facts. A pinned model is not proof of provider availability or automatic failover.

This catalog documents the 14 shipped pack profiles (`opencode-pack/agents/`), 4 optional Free-first presets (`opencode-pack/presets/free-first/`), and 2 resolver-only fleet adapters (`opencode-pack/adapters/`). See [the orchestration how-to](../how-to/orchestration.md) for enablement and delegation steps, and [the specialists tutorial](../tutorials/specialists.md) for a safe fixture walkthrough.

<!-- generated:reference/profiles:start -->
## Pack agent profiles

14 agent profiles are defined in `opencode-pack/agents/` and selected by `--with-opencode`. `sdd-orchestrator` is the human-facing `primary` profile and pins no model: it inherits the session model. A pinned model is configuration only; this page makes no claim about provider, credential, quota or model availability.

Permission columns summarize the ordered `permissions` rules using last-match-wins selection per tool; resource-scoped allow rules are reported as `scoped` and not enumerated here. `sdd-lightweight` declares `steps: 10`.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-architecture-advisor` | `subagent` | `9router/ag/claude-sonnet-5-5-high` | base `deny-by-default`, 15 rules | `deny` | `deny` | `deny` | `allow` |
| `sdd-architecture-review-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-code-review-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-debug-diagnose` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-docs-worker` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `deny` | `deny` | `deny` |
| `sdd-lightweight` | `subagent` | `opencode/beam-free` | base `deny-by-default`, 16 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-openai-architect` | `subagent` | `openai/gpt-6.1-sol` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-openai-implement` | `subagent` | `openai/gpt-6.1-terra` | base `explicit-rules`, 8 rules | `allow` | `not-declared` | `deny` | `deny` |
| `sdd-orchestrator` | `primary` | session model (inherit-session) | base `explicit-rules`, 1 rule | `not-declared` | `not-declared` | `allow` | `not-declared` |
| `sdd-research-librarian` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 17 rules | `deny` | `deny` | `deny` | `allow` |
| `sdd-review-verification-1` | `subagent` | `opencode-go/glm-5.3-flash` | base `deny-by-default`, 16 rules | `deny` | `scoped` | `deny` | `deny` |
| `sdd-review-verification-2` | `subagent` | `opencode/beam-free` | base `deny-by-default`, 16 rules | `deny` | `scoped` | `deny` | `deny` |
| `sdd-test-design` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-visual-inspector` | `subagent` | `opencode/glm-5.3-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |

## Free-first preset templates

4 templates are defined in `opencode-pack/presets/free-first/`. They are selected only when both `--with-opencode` and `--opencode-free-first` are passed; `--with-opencode` alone does not install them.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-implement-1` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-implement-2` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-test-implement-1` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-test-implement-2` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |

## Fleet adapter profiles

2 profiles are defined in `opencode-pack/adapters/`. They are resolver-only render sources for reviewed fleet model routes and are not installed by the ordinary OpenCode pack: `--with-opencode` reads `opencode-pack/agents/` only.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-plan-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-spec-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |

## Derived counts

| Class | Selected by | Count |
| --- | --- | --- |
| Pack agent profiles | `--with-opencode` | 14 |
| Free-first preset templates | `--with-opencode --opencode-free-first` | 4 |
| Fleet adapter profiles | not installed by the pack | 2 |

## Source evidence

Facts above are parsed from 20 profile files with the pinned `yaml` parser; fingerprints are recorded in the documentation manifest. This block describes repository source, not runtime permission enforcement, model routing success or the installation state of any machine.
<!-- generated:reference/profiles:end -->

## Profile installation classes and untouched ambient files

The installer (`internal/skillsync/sync.go`, `cmd/agentic-sdd/main.go`) and the fleet resolver (`internal/fleet/resolve.go`) treat the three source directories separately:

- **14 shipped pack profiles (`opencode-pack/agents/`):** Selected by `--with-opencode` (on by default in `asdx`; opt-in on `asdx-core`) and written to `<home>/.config/opencode/agents/`. Five pre-existing deny-by-default roles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`, `sdd-lightweight`) are backed up before replacement (`internal/skillsync/opencode_pack_test.go`).
- **4 optional Free-first presets (`opencode-pack/presets/free-first/`):** Selected only when `--with-opencode` and `--opencode-free-first` are both passed; `--opencode-free-first` alone exits `2` (`cmd/agentic-sdd/main.go`). They install `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, and `sdd-test-implement-2` after backing up replaced files.
- **2 fleet adapters (`opencode-pack/adapters/`):** `sdd-spec-9r` and `sdd-plan-9r` are resolver-only render sources for `asdx-core fleet` and are never installed by `--with-opencode` (`internal/skillsync/opencode_adapters_test.go`).
- **Ambient local configuration untouched:** Local `sdd-implement-9r-*`, `sdd-implement-free-*`, `vision`, `opencode.json`, and `<home>/.config/opencode/AGENTS.md` are never modified or removed by `--with-opencode` (`opencode-pack/README.md`, `opencode-pack/routing.md`).

## Model-pin grammar versus actual availability

Every pinned `model:` string in profile frontmatter and every override in `fleet.json` is syntax metadata only:

- **Grammar:** The fleet schema (`internal/fleet/fleet.go`) requires `<provider>/<model>[#variant]` matching `^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`. The website catalog collector (`website/scripts/reference/catalogs.mjs`) enforces `<provider>/<model>` syntax on all 19 subagent profiles, presets, and adapters while requiring `sdd-orchestrator` to omit `model:` and inherit the active session model.
- **Grammar is not availability:** A valid `model:` string or catalog entry never proves provider access, OAuth or API-key credentials, quota, subscription tier (for example `openai/*` does not establish a ChatGPT subscription per `internal/fleet/doctor.go`), or runtime model availability.
- **Offline inspection:** `asdx-core fleet doctor` performs no network or model calls; without `--opencode-bin`, routes are labeled `not checked (offline)` (`internal/fleet/doctor.go`). See [the fleet reference](./fleet.md).

## Mode and tool permission policy

All 20 repository files use OpenCode v2 frontmatter (`mode`, ordered `permissions:` rules evaluated last-match-wins); v1 `permission:`, `tools:`, and legacy `bash`/`task`/`write`/`patch` actions are rejected by `internal/skillsync/opencode_pack_test.go`.

- **Primary vs subagent:** Only `sdd-orchestrator` sets `mode: primary` and allows `subagent`; all 19 other profiles, presets, and adapters set `mode: subagent` and deny `subagent` so nested delegation is disabled.
- **Deny-by-default baseline:** Read-only roles open with `*` / `*` -> `deny`, allow `read`, `glob`, `grep`, `skill`, `execute`, `codegraph_*`, and `context7_*`, then append trailing `deny` rules for `read` on `*.env` / `*.env.*` (plus `*secret*` / `*credential*` on `sdd-lightweight`), `subagent`, `shell`, and `edit`.
- **Write scope:** Only `sdd-docs-worker`, `sdd-openai-implement`, and the 4 `opencode-pack/presets/free-first/` templates allow `edit`.
- **Scoped review commands:** The 5 review roles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`, `sdd-openai-architect`) allow `external_directory` reads and `shell` only for `git diff` and `git status --short`.
- **External search:** Only the 5 high-intensity roles (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-architecture-advisor`, `sdd-research-librarian`, `sdd-openai-architect`) allow `exa_web_search_exa` and `exa_web_fetch_exa` (`sdd-research-librarian` also allows `webfetch` and `websearch`); all other deny-by-default roles and writer presets deny `exa_*`.
- **Model-only fleet rendering:** `internal/fleet/render.go` (`VerifyModelOnly`) permits changing only the top-level `model:` scalar and fails closed if any permission rule or prompt body changes.
- **Instruction boundary:** Frontmatter permissions and prompt rules are instruction-level controls, not an operating-system sandbox (`opencode-pack/README.md`).

## Manual routing and fallback policy

`opencode-pack/routing.md` defines four provider lanes (`opencode/`, `opencode-go/`, `9router/`, `openai/`), the Free-first ladder (`opencode/deepseek-v4.1-flash-free` -> `opencode/glm-5.3-flash-free` -> `opencode/mimo-v2.6-flash-free`), the preserved 5-agent Claude/9Router exception (`sdd-spec-9r`, `sdd-plan-9r`, `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-implement-9r-2`), and the two Beam cross-lane rules (`sdd-lightweight`, `sdd-review-verification-2`).

**Written routing is policy, not automatic failover.** Neither `asdx-core` nor `opencode-pack` switches providers or models automatically (`opencode-pack/routing.md`, `opencode-pack/skills/agentic-sdd-orchestrate/references/failure-and-reconciliation.md`). Any fallback requires manual parent selection, context/output/modality compatibility checks, and explicit human authorization before entering a paid lane (`opencode-go/`, `openai/`).

## Curated source pointers and related references

- **Canonical pack and routing policy:** `opencode-pack/agents/`, `opencode-pack/presets/free-first/`, `opencode-pack/adapters/`, `opencode-pack/routing.md`, `opencode-pack/README.md`
- **Installer, backup and validation code:** `internal/skillsync/sync.go`, `internal/skillsync/opencode_pack_test.go`, `internal/skillsync/opencode_adapters_test.go`, `cmd/agentic-sdd/main.go`
- **Fleet grammar, renderer and diagnostics:** `internal/fleet/fleet.go`, `internal/fleet/resolve.go`, `internal/fleet/render.go`, `internal/fleet/doctor.go`
- **Reference collector:** `website/scripts/reference/catalogs.mjs`
- **Related pages:** [Orchestration how-to](../how-to/orchestration.md), [Specialists tutorial](../tutorials/specialists.md), [Fleet reference](./fleet.md), [CLI reference](./cli.md), [Skill catalog](./skills.md), [Files and backups](./files-and-backups.md), [Configuration reference](./configuration.md)
