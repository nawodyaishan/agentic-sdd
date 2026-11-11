# Feature 014 — Git-Managed OpenCode Fleet Profiles & Model Overrides

**Status:** Approved 2026-11-11 — combined spec/plan/tasks approval recorded once in §13; implementation and batch state follow `tasks.md`.

**Location:** `specs/014-agent-fleet-model-overrides/spec.md` (canonical; supersedes `014-agent-fleet-model-overrides-spec.md` draft of same directory)
**Prepared:** 2026-11-10; **Revised:** 2026-11-11 (conflict resolutions R1–R11 from codebase inspection + research; approval and B1–B9 authorization recorded in §13)
**Target:** Agentic-SDD Go CLI; OpenCode V2 optional integration
**Decision record:** Owner selected **1A, 2A, 3A, 4A, 5A** on 2026-11-10
**Source snapshot:** `docs/agent-fleet-spec-review-2026-11-10.md`, repository commit reported there as `5051299`, uncommitted Feature 013 B6/B7 TUI work, `docs/subagent-fleet-comparison.md`, and the previous rejected Fleet Manager concept. The review, not this document, inspected the checkout and OpenCode v2.2.6. No fresh working-tree inspection or tests were performed in drafting this spec.

> **Approval boundary:** This specification defines desired product behavior; it does not authorize code modification, reading private credentials, installing to a real OpenCode home, running models, changing subscription settings, paying for inference, or publishing a release. Existing Agentic-SDD decisions on combined approval, one explicitly authorized batch, and human review remain in force.

## 1. Executive outcome

Deliver a **CLI-first, standard-library Go configuration system** that composes the repository's **existing, version-controlled OpenCode agent roles** with a user's **strict-JSON, Git-managed personal fleet configuration**. It must reproduce the maintainer's **18 current machine agent names** as opted-in model/provider instances, without copying each complete prompt, changing approved Feature 009 permissions, or requiring repeated manual editing of `~/.config/opencode/agents/`.

The user workflow is:

```text
Agentic-SDD repository            User's private Git repository
canonical OpenCode agent roles    fleet.json (model and enablement)
          \                         /
           \                       /
            deterministic resolver
                      |
              render + validate
                      |
              diff / dry-run plan
                      |
             explicit safe apply
                      |
       ~/.config/opencode/agents/*.md
                      |
                 OpenCode V2
```

**Core ownership rule:** Agentic-SDD owns **what the role does and what permissions apply**; the user owns **which available model powers each instance and whether it is installed/enabled**. Feature 014 must not introduce a competing delegation or fallback authority.

### 1.1 Why 014, not 013

The repository review reports that Feature 013 is already the **Unified TUI Experience**, with B6/B7 active, while Feature 009 is approved v6 and delivered under `v0.6.0`. This feature is explicitly **014**, separate from both. The supported root CLI ships first; a TUI screen is **not** a dependency or an acceptance criterion.

### 1.2 Scope chosen by the owner

| Choice | Decision | Binding consequence |
|---|---|---|
| 1A | CLI first | Build and test root CLI, with no TUI integration required for acceptance. |
| 2A | Reproduce all 18 names | Support names as generated instances/aliases and bindings, not 18 duplicated prompt bodies. |
| 3A | Separate private Git | Read personal configuration from a user-supplied local path checked out from a private repository; never manage credentials or push to Git automatically. |
| 4A | Strict JSON | Parse with Go standard library; no YAML/JSONC dependency for personal input. |
| 5A | Preserve approved 009 | Do not change `--with-opencode` behavior, Option B permissions, default routing, or existing package semantics. |

### 1.3 Definition of success

A user with the existing OpenCode pack can check out their private fleet configuration on another machine, inspect the exact resulting agent files and model routes **without making any model requests**, explicitly apply managed changes, detect later manual drift, switch one agent's model without touching its role/permissions, and safely recover after a failed or unwanted apply. The public Agentic-SDD installation and existing 009 pack remain unchanged unless this opt-in feature is invoked.

## 2. Verified-vs-reported baseline

The following is **review-reported**, not freshly reverified here:

| Item | Review-reported current state | Implication |
|---|---|---|
| Portable engineering skills | 18, located in `skills-files/` | Keep methodology and authority. |
| OpenCode pack | 14 profiles in `opencode-pack/agents/` | Reuse these source prompts and approved permissions. |
| Free-first presets | Four variants in `opencode-pack/presets/free-first/` | Do not treat them as absent roles. |
| Machine | 18 global installed profiles | Compatibility target for personal migration. |
| Fleet overlap | Five names shared; nine genuinely absent names; four more available as preset variants | Prefer instances/aliases over duplicated prompts. |
| Root Go module | `go 1.23`, no `require` | Use standard-library JSON and existing internal packages. |
| Existing installer | `--with-opencode` backs up/replaces five overlapping agents under accepted 009 Option B | Preserve exact behavior when no fleet command is used. |
| Manifest | `internal/skillsync/manifest.go` format 3 tracks backup digest but not desired applied digest | Add last-applied provenance for new fleet transactions. |
| OpenCode | Machine v2.2.6; 009 historically tested v2.2.5 | Pin exact versions for real claims; recheck against executable. |
| TUI | Experimental nested module `experiments/tui/go.mod`; Feature 013 B6/B7 underway | Do not block root-CLI 014 on TUI. |
| Independent-review isolation | IR-06-01 symlink escape unresolved | Do not claim a profile renderer fixes OS isolation. |

### 2.1 Reconciliation contract: 18 machine names

The implementation must **inspect the actual current files before importing**; names below are from the 2026-11-10 comparison and may have changed.

**Five overlaps:** `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`, `sdd-lightweight`.

**Four names already represented in Free-first presets (different model bindings):** `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1`, `sdd-test-implement-2`.

**Nine other machine names requiring instance bindings, thin adapters, or one justified canonical addition:**

- 9Router implementation variants: `sdd-implement-9r-1`, `sdd-implement-9r-2`, `sdd-implement-9r-cheap-1`, `sdd-implement-9r-cheap-2`.
- OpenCode Console Free implementation variants: `sdd-implement-free-1`, `sdd-implement-free-2`.
- Spec/plan: `sdd-spec-9r`, `sdd-plan-9r`.
- Vision: `vision`.

**A compatibility name is not a new engineering role.** `vision` is potentially distinct; first determine whether `sdd-visual-inspector` has a truly equivalent contract. Spec/plan instances must be derived from approved methodology and explicitly non-writing unless a separate accepted policy says otherwise. Do not import the machine's bare `subagent: deny` rules as adequate least-privilege control.

### 2.2 Explicit correction of earlier claims

- The assertion “13 profiles have no counterpart anywhere in the pack or presets” was false: **nine distinct names are absent and four appear under Free-first presets**.
- The machine fleet is **not uniformly more restrictive**. Some writer/spec/plan agents lack a broad deny baseline; use **effective permission behavior**, not count of explicit `edit` rules.
- Existing reviewer `external_directory` rules and the complete `sdd-lightweight` diff must be accurately inspected. Do not reverse approved 009 MCP/Code Mode permissions under the guise of preserving machine settings.
- The root `openai/*` provider namespace is not proof of ChatGPT subscription authentication. `9router/ag/...` is a provider route; Gemini/Claude/Argon are model families/upstreams. Do not invent a `9router/openai` lane without catalog evidence.

## 3. Architecture and sources of authority

### 3.1 Sources — clear precedence by responsibility

| Source | Owns | Must not own |
|---|---|---|
| `skills-files/`, existing feature records, repository `AGENTS.md` | Portable SDD methodology, feature approval, batch continuation and review boundaries | User's personal subscriptions |
| Approved Feature 009, `opencode-pack/agents/` and existing role references | Canonical prompt bodies, agent roles, effective permission baseline, delegation/fallback policy | Personal model purchases/credentials |
| Bundled/free-first preset metadata (existing or minimal index) | Reusable example model bindings and compatible source-role IDs | Authorization or fallback decisions |
| Personal **strict JSON** file in private Git checkout | Chosen names/role bindings, model IDs/variants and per-instance enablement | Prompts, tool permissions, credentials, provider routing authority |
| Fleet resolver and Go CLI | Compose, validate, preview, apply, track provenance, rollback | Model invocation, native orchestration engine, spending enforcement |
| OpenCode effective loaded config | Runtime reality of installed profiles, inheritance, enabled models and permissions | Canonical authorization; installed state is compared, not blindly trusted |

**Precedence for allowed model fields:** canonical default → explicitly selected existing preset → personal instance override. A missing override leaves the prior layer intact; explicit false for `enabled` disables the named instance. No blanket machine-state import overrides repository roles.

**Precedence for policy:** existing 009 canonical role/permissions remains authoritative; the personal configuration cannot set `permissions`, `shell`, `execute`, `subagent`, `fallback`, `automaticDelegation`, `providerSwitching`, or spending permission. Fail on such unknown keys instead of ignoring them.

**OpenCode merging is a separate concern:** A project-local/global OpenCode definition may later change effective permissions/model. The CLI must identify known conflicts and qualify its claims. Rendering correct files is not proof of effective runtime policy.

### 3.2 Canonical prompt store; no duplicate source tree

Use **existing** `opencode-pack/agents/*.md` as the prompt/policy source. Reuse `opencode-pack/presets/free-first/` for existing variant templates. A small alias/role mapping may live beside these sources; it contains **references only**, not duplicated full prompts.

If the spec/plan/vision roles cannot be generated from any approved canonical definition, the implementation may add **minimal, reviewed, reusable source adapters** under the existing pack tree. This is an exception justified by missing behavior; it must not create a parallel `catalog/roles/*/prompt.md` source.

Preserve exact prompt body bytes and permission rules for inherited profiles unless explicitly reviewed as a separate approved policy change. Only supported frontmatter fields such as `model` (including `#variant`) and the validated enablement mechanism may be synthesized from the personal config.

### 3.3 Instance identity vs. logical role

An **agent role** is a reusable behavioral contract; an **instance** is a named OpenCode-visible profile using that role with a selected provider/model. This supports many instances of one role without copying or independently maintaining its prompt.

Example: `sdd-implement-1` and `sdd-implement-9r-1` may share the same accepted implementation role but use different model identifiers. The exact source role ID is determined from inspected source files, not inferred solely from the name prefix.

Instances must have unique stable OpenCode agent IDs. An alias referring to another alias must be rejected in v1 (one-hop canonical source only); reject unknown source IDs and cycles. Never merge permissions from two templates by accident.

## 4. Strict-JSON personal source contract

The exact schema can be refined in `plan.md`; the following **proposed v1 JSON shape** conveys normative ownership and supported concepts. It is not a currently implemented CLI format.

```json
{
  "schemaVersion": 1,
  "fleet": "personal",
  "preset": "opencode-go",
  "instances": {
    "sdd-orchestrator": {
      "source": "agent:sdd-orchestrator",
      "enabled": true
    },
    "sdd-implement-1": {
      "source": "preset:sdd-implement-1",
      "model": "opencode-go/deepseek-v4.1-flash#high",
      "enabled": true
    },
    "sdd-implement-9r-1": {
      "source": "preset:sdd-implement-1",
      "model": "9router/ag/gemini-3.8-flash-high",
      "enabled": true
    },
    "sdd-implement-free-1": {
      "source": "preset:sdd-implement-1",
      "model": "opencode/deepseek-v4.1-flash-free",
      "enabled": true
    },
    "sdd-openai-architect": {
      "source": "agent:sdd-openai-architect",
      "enabled": false
    }
  }
}
```

*`source` is namespaced `agent:<name>` (from `opencode-pack/agents/*.md`) or `preset:<name>` (from `opencode-pack/presets/free-first/*.md`). R1–R2: `sdd-implement-1/2` resolve to `preset:` because no `agents/sdd-implement-*.md` exists in the checkout; 9Router/Free implement variants reuse the reviewed preset body with only `model:` changed. If `enabled` is omitted the plan must define the default explicitly (proposed default: `true`).*

### 4.1 Parsing and validation

- Parse with Go `encoding/json`, **reject unknown fields**, type errors, invalid IDs, unexpected top-level keys, duplicate object keys, aliases with path separators/traversal, overlarge input files and invalid UTF-8.
- Document schema version and maintain test fixtures for future upgrades; unknown future schema versions fail safely.
- Do not add YAML, a JSONC parser, a JSON Schema runtime dependency, an embedded scripting DSL, or arbitrary templates to the root module as part of this feature.
- Validate OpenCode's documented `provider/model` plus optional `#variant` syntax without assuming the referenced model is authenticated, supported by the current account or cost-free.
- No credentials, personal endpoint secrets, provider keys, bearer tokens or account data belong in this source. Treat even private Git config as sensitive metadata when it includes internal hostnames/paths.
- The path to the personal checkout is passed explicitly by `--config` or a **documented, user-selected pointer** stored in existing local settings. Never silently search personal Git repositories, auto-clone a remote URL or push to Git.
- The user is responsible for commits, rebases, private-remote ACLs and backup of the separate Git repository; the CLI reports source revision when available without requiring Git to be installed.

### 4.2 Provider/model awareness

Support arbitrary valid OpenCode model IDs including historical lane examples:

- `opencode/...` — Console / free or metered model as actually cataloged.
- `opencode-go/...` — OpenCode Go subscription route.
- `9router/...` — one configured upstream proxy route; 9Router model IDs can distinguish Claude, Gemini or Argon families.
- `openai/...` — route may be ChatGPT OAuth or OpenAI API depending on actual authentication; never equate the namespace with a subscription.
- Other providers allowed when the installed OpenCode catalog and user configuration recognize them.

**Config parsing is offline.** An optional diagnostics command may consult an available *local* OpenCode catalog or CLI, if explicitly requested and safe, to report `configured`, `catalog-visible`, `authenticated`, `unknown`, or `unavailable`; do not assert entitlement from a model string. No model inference, provider requests, account switching or paid API calls during plan/apply/status.

### 4.3 Disabled instances

A false `enabled` means **not runnable via the generated managed set**. Prefer **omitting the managed agent Markdown from active `agents/`** when a reliable native `disabled` frontmatter mechanism has not been demonstrated on the pinned runtime. For an already installed managed instance, removal requires explicit plan/apply plus a recoverable backup; do not delete an unrelated user-owned file. The mere presence of a template in the source repo must not make a reviewer runnable.

## 5. Functional requirements

### FR-01 — Read-only fleet inventory and source audit

Add an offline inventory operation that distinguishes canonical roles, existing presets, personal instances, installed managed files, and unmanaged files. Reconcile current 18-name machine state against 14 pack profiles and four Free-first variants on the reviewed snapshot, but rederive real counts from actual files. Record role IDs, source hashes, model routes, enabled state and permission discrepancies without reading credentials or printing private content unnecessarily.

### FR-02 — Full 18-name reproducibility through instances

The personal configuration mechanism must represent **all 18 reported machine names**, including the nine absent from the current pack and the four present via presets. Preserve user-visible names on migration. Reuse canonical role prompt/policy bodies; provide only the minimal approved additions needed for spec, plan or vision semantics. Never create 13 near-identical prompts to satisfy a count.

### FR-03 — Separate personal Git configuration

Support a file such as `/path/to/private/agentic-sdd-fleet/fleet.json` as the source of model/enablement data. Users can clone and version this configuration independently of the public project. Ensure multiple machine checkouts can produce identical intended outputs given the same source revision, personal file and supported runtime assumptions. Store no credential material or implicit private-repo clone/push automation.

### FR-04 — Deterministic field-limited rendering

Render final native OpenCode agent definitions by starting from an inspected canonical profile, changing only approved model/variant and enablement representation, and writing the intended ID to the proper filename. Preserve prompt body and inherited permissions unchanged. Identical inputs must produce identical bytes; no timestamps/secret values inside rendered files. If model binding is invalid, fail without partial output.

### FR-05 — Maintain approved 009 behavior

Normal `agentic-sdd` skill installation and **all existing `--with-opencode` and `--opencode-free-first` semantics** remain unchanged when Feature 014 is not invoked. In particular, retain 009 Option B's documented replacement behavior for the five overlaps. Never silently update or replace existing 009 routing/fallback instructions, approvals, or a global/project `AGENTS.md`.

### FR-06 — Isolated opt-in CLI contract

Expose one clearly named optional root CLI family, **tentatively**:

```sh
# Illustrative only; exact syntax must be reconciled with current CLI parser.
agentic-sdd fleet status --config /path/to/fleet.json
agentic-sdd fleet plan --config /path/to/fleet.json
agentic-sdd fleet apply --plan /path/to/plan.json
agentic-sdd fleet restore --backup <managed-backup-id>
```

`status` and default `plan` are nonmutating, offline and non-inference. A plan may be saved to a user-chosen file only with an explicit output option. Apply requires a fresh explicit action, not a silent option of normal skill sync. `restore` affects only the recorded managed transaction. `--help` and exit codes communicate conflict, unsupported version, stale plan, unknown source, and data integrity issues. No separate Go daemon, plugin execution runtime or new model session engine.

### FR-07 — Plan identity, stale-plan detection and TOCTOU safety

A plan must bind the resolved source revision/digest, personal config digest, renderer version, target root, exact file list, last-known installed hashes, proposed per-file creates/updates/removals, and any permission-relevant observations. The plan must have a deterministic integrity identity. Revalidate source and target state immediately before applying; refuse if a file, source, scope or supported runtime changed since preview. A plan ID is a consistency token, **not** approval or proof of security. No bypass of human review when installing to a real home.

### FR-08 — Transactional apply, ownership and restore

Reuse `internal/skillsync` / `internal/appops` safe mechanics where suitable. Stage and validate all intended files, acquire an appropriate operation lock, back up affected managed/current files, and commit with rollback on partial failure. Reject unsafe symlinked targets, traversal and unmanaged collisions; never edit user-owned `AGENTS.md`. A failed rollback must be recorded accurately and leave a recoverable backup; do not promise atomic durability across filesystems where unsupported. Repeat apply without drift must be an idempotent no-op. Restore validates current hashes and never discards later user changes silently.

### FR-09 — Versioned provenance and drift ledger

Introduce **manifest format 4** (or an explicitly namespaced fleet manifest of format 4 if sharing the old manifest would break the existing installer) with at least:

- Managed path, installation scope and source profile/instance ID.
- Source version/revision and canonical template digest.
- Personal configuration digest; renderer/schema version.
- **Last applied desired-content digest** for each installed file.
- Previous backup reference and its digest, where one exists.
- Installed/disabled ownership state and creation/update transaction identity.

A format-3 record's backup digest must **not** be reinterpreted as last-applied desired state. Existing skill restore semantics must continue functioning. Define conservative migration/adoption: when prior desired digest is unknown, report `unmanaged/legacy provenance unknown` and require explicit review before adoption/overwrite. Do not mutate all existing format-3 manifests automatically.

### FR-10 — Unambiguous drift handling

Compare **last applied**, **current installed**, and **newly desired** states:

| Comparison | Required behavior |
|---|---|
| No recorded ownership; target absent | Preview managed create. |
| No recorded ownership; target present | Unmanaged collision: stop unless explicit adoption path. |
| Current = last applied; new desired = current | No-op. |
| Current = last applied; new desired differs | Preview safe update. |
| Current differs from last applied; new desired equals last applied | Report manual drift; do not overwrite automatically. |
| Current differs from last applied; new desired differs | Conflict; require human decision, never perform speculative merge. |
| Managed file marked disabled in new config | Preview specific managed removal, with recoverable backup. |
| Source/renderer/target mismatch since plan | Reject stale plan and replan. |

This is **three-state comparison**, not a blanket automatic three-way text merge. Conflict resolution must not operate on free-form prompts or permission blocks without review.

### FR-11 — Exact permission preservation and safe new roles

A model-only update must preserve the entire approved effective role contract, not silently broaden shell, edit, external-directory, web, MCP or subagent access. Detect any changed permission block and reject as a non-model-only update. For **new thin adapter roles**, define safe, reviewable effective defaults: spec/plan cannot silently edit or shell; vision cannot claim unrestricted file isolation; workers require the same SDD authorization boundaries as canonical implementation roles. Effective tool policies must be tested on the actual supported OpenCode runtime, not inferred from prose.

### FR-12 — No new routing, fallback or spending authority

Do not put fallback tables, provider switching, auto-delegation, paid use flags or inference spending policy inside `fleet.json`. Preserve the approved Feature 009 routing and fallback contract. The config can express **model choice and enabled state only**. A model override does not grant paid spending, new credentials, provider privacy approval, an implementation batch or reviewer acceptance. Display route changes and warn when a selected provider/model appears different from the prior route; do not automatically call it.

### FR-13 — Orchestrator state visibility and optional activation

Existing `sdd-orchestrator` remains optional. Installing its profile is **not** the same as selecting it as OpenCode's current primary. `status` must show `installed`, `enabled`, and `selected default/session` as different states where observable. If the CLI ever exposes `activate`, require a separate explicit operation, verified backing config and safe restore. No default-agent change is required for 014. Generate a small **read-only fleet inventory/report** for human consumption and, if suitable, an optional parent-facing available-agent listing derived from actual installed state; do not pretend the orchestrator dynamically introspects the filesystem without a tested mechanism.

### FR-14 — OpenCode V2 support and effective validation

Check actual native agent discovery/precedence, `mode`, `model`, variant `#...`, permission actions, `disabled` behavior and any parent-child restrictions against a pinned executable. Use the review's v2.2.5 (historical) and v2.2.6 (machine snapshot) as **candidate test versions**, not proof. Fail clearly when supported-version constraints are unmet. The CLI may render offline without OpenCode only if it labels runtime validation `not performed` and avoids claims about successful invocation.

### FR-15 — User-driven import/adoption of existing agents

Provide a **read-only** import preview for machine profiles (when a user chooses to inspect their local path): identify which profile maps to an approved canonical source, which is a model-only variant, and which contains user-specific behavioral/policy modifications. Only import safe model/enablement assignments into `fleet.json`; **do not automatically overwrite repository prompts or accept risky machine permissions**. Unknown/modified roles must be reported for human classification. An import operation writing the private JSON source is separately authorized and must never modify live OpenCode files simultaneously.

### FR-16 — Open-source usability and documented model management

Provide a concise worked guide for: create private Git repo, write `fleet.json`, install existing canonical pack if desired, preview a 1-agent model change, apply safely, diagnose drift, restore, and reproduce on another machine. Explain provider names vs model families, selected vs authenticated/entitled models, and that `openai/*` is not necessarily ChatGPT-plan usage. No hosted marketplace, no remote package fetching and no TUI screens in this feature.

## 6. Non-functional requirements (measurable)

The implementation plan must choose representative supported hardware/test environments and record repeatable measurements against at least the sample 18-instance fleet:

| ID | Requirement |
|---|---|
| NFR-01 | Deterministic: same canonical source + exact config + renderer version yield byte-identical rendered files on repeated runs. |
| NFR-02 | Performance: offline parse/resolve/render/plan for 18 instances completes under **2 seconds** on the designated CI runner; record median of five warm runs and test worst-case valid input (e.g., 100 instances) with a documented upper bound. |
| NFR-03 | No additional runtime startup cost in non-fleet CLI paths; baseline CLI paths do not read the private config or launch new processes. |
| NFR-04 | Root Go module remains zero new third-party dependencies for this feature unless the human explicitly approves a narrowly justified change. |
| NFR-05 | Offline by default: plan/status/apply/restore perform no provider network requests or LLM inference; optional local diagnostics state exactly which tools they inspect. |
| NFR-06 | Source safety: reject unknown/duplicate JSON keys, oversized files, invalid aliases, traversal and target symlink redirects. |
| NFR-07 | Integrity: managed files have attributable source and last-applied digests; conflict detection favors preserving user data over automatic convergence. |
| NFR-08 | Privacy: never log credentials, full private JSON content, raw auth paths or unrelated files; show only relevant model IDs and sanitized diff. |
| NFR-09 | No implicit activation or model spending; installing an agent does not make it OpenCode's default primary or run a provider request. |
| NFR-10 | Compatibility: existing 009 flags, 013 experimental TUI and four-client portable skill installer work as before when 014 is unused. |
| NFR-11 | Native claims are version-scoped: document supported OpenCode executable versions and actual effective-tool tests, no blanket permission/isolation certification. |
| NFR-12 | Failure observability: distinguish rejected config, stale plan, unmanaged collision, failed stage, incomplete rollback, unsupported version and unknown provenance using stable diagnostics/exit states. |

Performance criteria are **design targets**, not claims already measured; test infrastructure may justify a revised limit in the approved `plan.md` before implementation.

## 7. Acceptance criteria and mandatory tests

Every acceptance criterion must be recorded as **Pass / Fail / Blocked / Not run** with candidate commit, commands, environment and evidence. Static prompt validation is not evidence of model behavior.

### 7.1 Representation, ownership and compatibility

- **AC-01:** Feature number 014 is unoccupied in the target checkout; existing 009 and active 013 task records remain unchanged unless a separately reviewed dependency fix is needed.
- **AC-02:** All 18 historical machine names can be represented and rendered from accepted roles/instances; the nine truly absent names are not implemented as nine unrelated duplicated full prompts; the four Free-first variants are reused where appropriate.
- **AC-03:** Same source and strict JSON config generate exactly identical file bytes on repeated runs; canonical role instructions and permission blocks are unchanged by model overrides.
- **AC-04:** Personal model/enablement changes require no modifications to the public Agentic-SDD source and can be kept in a separately cloned private Git repository.
- **AC-05:** Invalid config, duplicate keys, unsupported schema, non-string model, invalid variants, unknown canonical source, name collisions and arbitrary policy fields fail with clear diagnostics.
- **AC-06:** The existing `--with-opencode` and `--opencode-free-first` command paths and approved replacement semantics pass unchanged regression tests.
- **AC-07:** No new root Go third-party dependencies, duplicate role prompt tree, policy ledger or agent runtime are introduced.

### 7.2 CLI and deployment safety

- **AC-08:** `fleet status` and `fleet plan` (without explicitly requested output) are read-only, offline and make zero LLM/API requests.
- **AC-09:** Plan reports every create/update/remove/no-op/conflict, exact target, selected model binding and significant permission/model differences, without printing secrets.
- **AC-10:** `apply` refuses stale plans after any source digest, personal JSON, target file, scope or renderer change; plan IDs do not replace human authorization.
- **AC-11:** Existing unmanaged agent files and global/project `AGENTS.md` are never overwritten or appended to without an explicit separately scoped operation; 014 does not offer an automatic `AGENTS.md` write.
- **AC-12:** Installing/updating a managed agent creates adequate backups; a simulated interrupted multi-file apply is rolled back or leaves an explicit recoverable failure state.
- **AC-13:** Unchanged reapplication is a no-op; manual installed-file modification triggers drift/conflict and no silent overwrite.
- **AC-14:** Restore preserves unrelated user files and rejects a modified target without explicit review; it never treats `git revert` as automatically restoring live files.
- **AC-15:** Format-3 manifests remain readable by legacy paths; format-4 records contain last applied desired digest and provenance; ambiguous migrated legacy state is never fabricated as managed/clean.
- **AC-16:** `enabled: false` prevents the named managed agent from being active. If OpenCode v2.2.6 `disabled` is unproven, the profile is omitted/removed only through a safe managed transaction.

### 7.3 Permissions, provider routes and orchestration

- **AC-17:** A model-only change cannot increase effective `edit`, `shell`, `subagent`, `external_directory` or MCP access relative to canonical approved policy. Static and actual supported-runtime checks are separated.
- **AC-18:** New spec/plan instances do not gain implicit write/shell permission; worker dispatch retains existing combined approval and batch rules. Existing IR-06-01 reviewer isolation limitation remains explicit.
- **AC-19:** Provider prefix and model ID parsing work for valid `opencode`, `opencode-go`, `9router`, `openai` and user-defined providers without inventing unsupported subscription/entitlement claims.
- **AC-20:** A selected `openai/*` route is not reported as ChatGPT Plus authenticated unless independent, relevant authentication evidence is available; catalog presence alone does not suffice.
- **AC-21:** `fleet status` distinguishes configured, rendered, installed, enabled, selected primary and effective runtime availability when observable; missing evidence is `unknown` or `not checked`.
- **AC-22:** Installing the orchestrator does not modify the OpenCode default primary. Parent agent role listing is generated only by a documented, verified mechanism; no dynamic-discovery claim without proof.
- **AC-23:** Changing a personal model assignment does not mutate approved 009 fallback rules, consume quota, trigger an agent, or imply paid retry authorization.

### 7.4 Usability, regressions and release

- **AC-24:** A new user can follow a source-control setup, preview, apply, drift-conflict and restore walkthrough on a temporary fixture home without reading the maintainer's real config.
- **AC-25:** At least one offline test represents an 18-agent fleet and at least one larger fixture tests inventory/renderer scale; required NFR timings and memory are recorded rather than asserted.
- **AC-26:** Changed Go packages are covered by the actual root module tests and registered in the maintained Taskfile/CI/harness verification coverage, preserving trusted-base execution boundaries.
- **AC-27:** Existing skill installation, backup/restore, OpenCode pack packaging and source-binary embedding tests pass; 014 does not depend on Feature 013 experimental TUI module.
- **AC-28:** Source release packaging includes canonical templates and any renderer metadata required for offline `fleet plan`; installed output matches release/source content for the tested artifact.
- **AC-29:** The final implementation report records supported OpenCode version(s), effective permissions evidence, manifest migration outcomes, exact changed files, blocked risks, unrun tests, and whether release is safe. No release publication is automatic.

## 8. Minimum negative-test matrix

Tests must cover at least:

1. Invalid/duplicate JSON keys, unknown schema, unsupported model identifier, malformed instance alias and >configured-size input.
2. Personal config attempts to set `permissions`, `fallbacks`, `automaticDelegation`, `shell`, `execute` or provider billing policy — **reject**.
3. Two instances map to the same installed name or an alias references another alias — **reject**.
4. Instance target already exists and is unmanaged — **conflict**, no overwrite.
5. One of the five approved 009 overlapping profiles differs from the expected renderer; simple `--with-opencode` still follows 009, separate fleet mode does not silently override.
6. Mid-run file changed after plan — **stale plan**, no write.
7. Stage or rename fails on file 3 of 6 — backups/rollback recover intact state or report recoverable partial failure.
8. Legacy format 3 has only a backup hash — no invented last-applied desired hash; adoption is explicit.
9. Current installed file manually edited after Fleet apply — drift detected, no silent overwrite.
10. Disabled managed profile previously installed — removal preview and backup; unrelated same-name unmanaged file is preserved.
11. Source role permission block changes between plan and apply — stale plan and reject; model-only mode cannot absorb policy update.
12. `sdd-spec-9r` or `sdd-plan-9r` sourced from weak machine prompt — new adapter uses vetted nonwriting policy or deployment is blocked.
13. Symlink from `~/.config/opencode/agents/` to external protected file — reject target escape; no external write.
14. Both 9Router and OpenAI model routes appear in JSON — no authentication inference, no network model request.
15. Explicit `fleet plan` only — no real-home changes, no credentials, no model invocation.
16. Two competing Fleet applies — lock or stale-state rejection prevents clobbering.
17. Reviewer path escape in runtime negative fixture — no isolation claim, retain IR-06-01 warning.
18. Select orchestrator profile in config without changing default-agent — installation does not activate primary automatically.
19. Git user reverts a private `fleet.json` revision — CLI still requires preview/apply to change installed files.
20. Test `--with-opencode` unmodified against its accepted 009 fixtures — no regression.

## 9. Explicit non-goals and deferred work

- **No TUI implementation** or modifications to active Feature 013 B6/B7; TUI reuse is an optional future integration only after that feature's accepted readiness.
- **No local curated role/provider marketplace** or interactive recipe search in 014; those may be proposed as later scope (working names 015) once 014 is reliable.
- **No remote marketplace / package import**, untrusted skills/plugins, registry server, signatures, download hooks or publisher trust program in 014 (possible later scope, working name 016). There is no existing release signing/attestation guarantee to lean on.
- No new SDD feature or batch-approval flow, no default auto-delegation, no rewrite of Feature 009 or its fallback/permission policies, no generic agent scheduler.
- No new model/provider authentication flow, billing enforcement or subscription quota polling.
- No unsupported promises about financial savings, token use, real-provider performance, reliable 9Router upstream privacy, isolation, or global OS containment.
- No automatic repo push, private Git remote management, secret encryption platform, or default machine-to-machine synchronization daemon.
- No cross-client fleet support beyond OpenCode V2 in this feature; portable skills for other clients remain unchanged.
- No new Linux/Windows release platform claim without supported artifacts and tests; historical public binaries were macOS-focused.
- No automatic adoption of all 18 machine names into every **public** default installation; 18-name reproduction is a personal opted-in feature.
- **No machine profile reverse-engineering / import tool in 014 (FR-15 deferred by human approval 2026-11-11)**: The root CLI grammar is strictly frozen to `status|plan|apply|restore|doctor` (`plan.md` §2.3). Automated or heuristic import of unmanaged machine profiles risks smuggling unvetted permissions or overwriting repository roles; user-driven adoption of existing profiles is safely supported via explicit `fleet plan --adopt`. Read-only import preview is deferred to a future bounded batch post-014.

## 10. Integration and implementation constraints for the plan

**Repository-first inspection is mandatory.** Before generating a technical `plan.md`, confirm current tree status and the exact APIs in:

- `opencode-pack/agents/`, `opencode-pack/presets/free-first/` and relevant embed/package definitions.
- `specs/009-opencode-orchestration/{spec.md,plan.md,tasks.md}` and Feature 009 release artifacts.
- `specs/013-unified-tui-experience/{spec.md,tasks.md}` and whether Feature 013 remains active.
- `internal/skillsync/{manifest,sync,restore}.go`, `internal/appops/`, `cmd/agentic-sdd/`, the CLI's existing option parsing and release build.
- Current `AGENTS.md`, Taskfile, `harness/checks.json`, trusted CI contracts, packaging and tests.
- User's actual 18 global profiles **only if separately authorized and scoped**; a sanitized local fixture can be used instead.

**Implementation shape preference:** Extend or extract small testable helpers from `skillsync`/`appops`, rather than create a parallel installer with duplicate backups, atomic writes or conflict logic. A small distinct fleet resolver module may be justified for compositional logic if it avoids circular dependencies; do not create multiple packages merely to mimic a generic framework. Preserve all existing source locations unless necessary for the accepted task.

**Delivery batches to derive later (not authorization):**

1. Inspect/normalize identity mapping and strict-JSON source schema; reconcile all 18 historical names.
2. Implement pure resolver/renderer, role references and deterministic validation with negative fixtures.
3. Extend opt-in root CLI with status/plan and exact plan identity; no write initially.
4. Add safe managed apply/restore, format-4 provenance and legacy migration behavior.
5. Validate effective permissions/version boundaries and full regression suite; document user workflow and release readiness.

The actual batch boundaries, file touch list, acceptance/test ownership and verification steps belong in `plan.md` and `tasks.md` after user review. Do not treat the above sequence as approved implementation batches.

## 11. Risks, mitigations, and release disposition

| Risk | Mitigation / release implication |
|---|---|
| Imported machine permissions are unexpectedly broad | Do not import permission blocks; use vetted canonical profiles, effective-policy tests and explicit exceptional role review. |
| Model source and installed OpenCode runtime disagree | Version preflight, honest `unknown` state and no unsupported claims; reject unsafe apply. |
| The resolver accidentally creates a second prompt/policy store | Canonical reference mapping; render only model/enabled fields and identity. |
| Private model config is published by mistake | Document private Git checkout, no key storage, no auto push, explicit path selection and source hygiene. |
| New format breaks old 009 or skill restore | Maintain backward compatibility or namespaced v4 Fleet manifest; test exact old restore paths. |
| Drift reconciliation overwrites user edits | Hash/three-state compare, no speculative merge, explicit conflict review. |
| User expects every 18-name profile to run | Distinguish rendered/discovered from authenticated/allowed; never claim runtime entitlement. |
| CLI flags change existing 009 Option B | Isolated subcommand and regression tests; **no** implicit behavior change. |
| Current TUI feature overlaps work | Root CLI first; no `experiments/tui` changes for 014. |
| Marketplace scope returns | Defer catalogs/remote imports; reuse the 014 resolver later rather than building a separate configuration mechanism. |

**Release rule:** The feature can be considered source-complete only after the accepted scope and deterministic tests are delivered. It can be declared tested on a specific OpenCode version only when that runtime was actually exercised. It can be released only after its opted-in distribution contract, restored state integrity, supported platform and existing core regressions are verified. The public/default installer behavior must remain compatible even if model routes are unavailable for the maintainer's personal account. A documented **no-go** is a valid outcome when material supported behavior remains unverified.

## 12. Required completion report

The agent implementing this feature must deliver one consolidated report with:

1. Confirmed repository revision and dirty/untracked state before and after.
2. The final all-18-name mapping (canonical source role, alias, personal model and active/disabled state), showing the nine absent names and four presets resolved without redundant prompts.
3. Exact source/renderer manifest schema and preservation of 009's behavior.
4. New CLI commands, effective config resolution, version support and model/permissions diagnostic limits.
5. Changed paths and generated outputs, with no global `AGENTS.md` modifications.
6. Test list with commands, pass/fail/blocked/not-run, failure-path/rollback fixtures, performance measurement and package verification.
7. Open risks, unknown account entitlements and retained IR-06-01 limitation.
8. Public documentation and private-Git sample setup, including backup/restore and limitations.
9. Release recommendation and **next action requiring explicit human authorization**.

Stop for human review. A technical `Ready`, green CI, a model profile being available or a successful dry-run does **not** grant new batch authorization or permission to modify a real user's global OpenCode configuration.

## 13. Approval

**Owner decisions on product scope:** 1A / 2A / 3A / 4A / 5A (2026-11-10).
**Combined spec/plan/tasks approval:** **Approved 2026-11-11** — owner accepted 3 docs (`spec.md` rev R1–R11, `plan.md` r1, `tasks.md` r1) in chat.
**Approved implementation batches:** **B1, B2, B3, B4, B5 authorized in parallel 2026-11-11** — owner requested parallel fleet (deepseek high + gemini flash + free models). Batch ledger in `tasks.md` owns state.
**Required fixes (2026-11-11 reviews):** verification `Needs changes` + Gemini-Flash `Approve with conditions` + DeepSeek-high `Request changes` recorded in `plan.md` §5 / `tasks.md` B6–B9. **B6–B9 authorized parallel 2026-11-11** (DeepSeek high + Gemini Flash). No real-home/model/release.
**FR-15 Deferral:** Human approval authorizes deferral of FR-15 (machine profile import preview) to preserve the zero-side-effects CLI boundary and frozen command grammar (`plan.md` §2.3); adoption of existing unmanaged profiles is supported via explicit `fleet plan --adopt`.
**Permission to inspect real global agent files or private personal Git:** **Not granted by this spec**.
**Permission to write real OpenCode config or call models:** **Not granted**.
**Permission to publish/release:** **Not granted**.

### References and provenance

**Provided repository evidence:** `agent-fleet-spec-review-2026-11-10.md` (current authoritative review for this proposal), `subagent-fleet-comparison.md` (historical comparison; several claims corrected by review), previous rejected Fleet Manager 013 proposal (design inspiration only), prior Agentic-SDD context pack (older snapshot; historical), existing Feature 009 and Feature 013 canonical files **to be inspected in the target checkout**.

**External architecture principles already researched in this conversation:** [chezmoi declarative dotfile management](https://www.chezmoi.io/user-guide/setup/), [OCX OpenCode profile/registry management](https://github.com/kdcokenny/ocx), [OpenCode agent and permission documentation](https://opencode.ai/docs/agents/), and [OpenCode model catalog and availability](https://opencode.ai/docs/models/). These references inform design, but do not replace direct tests or the project's approved policy.

## 14. Conflict resolutions (2026-11-11, research-backed, no approval granted)

Verified against checkout 2026-11-11 (`opencode-pack/agents/` = 14 files, `presets/free-first/` = 4 files, `go 1.23` zero `require`, `manifest.go` format 1–3, `cmd/agentic-sdd/main.go` flat parser). Each item is normative for `plan.md`/`tasks.md`.

**R1 — Instance vs role; no second prompt store.** One-hop alias only: `SourceRef` grammar `^(agent|preset):[a-z0-9][a-z0-9-]*$` resolved by walking `agents/*.md` + `adapters/*.md` + `presets/free-first/*.md` (B6 Option A; adapters share the `agent:` namespace and duplicate names across roots are rejected). Reject unknown source, alias→alias, cycles. Reuse: `sdd-implement-9r-1/2/cheap-1/2` + `sdd-implement-free-1/2` → `preset:sdd-implement-1` (or `-2` per lane as plan confirms); 5 overlaps → `agent:<same>`; orchestrator → `agent:sdd-orchestrator`. At most 3 reviewed source files under the existing pack tree: B6 placed `sdd-spec-9r.md` and `sdd-plan-9r.md` under `opencode-pack/adapters/` (fleet-only, excluded from the 009 install set) and mapped `vision` to `agent:sdd-visual-inspector` because that canonical role already has the equivalent read-only image-transcription contract; no `vision.md` adapter was added. Each added role is non-writing/non-shell by default and reviewed as a policy change. Never create `catalog/roles/*/prompt.md`.

**R2 — Corrected source mapping.** §4 example now uses `agent:`/`preset:` prefixes. Plan adds golden test that every `SourceRef` resolves to a walked file.

**R3 — Isolated `fleet` dispatch.** Add `case "fleet": return runFleet(args[1:], …)` before `default` in `runEnv`; existing `preview/apply/backups/restore/guide/tui/opencode` paths untouched. `runFleet` owns `fleet status|plan|apply|restore [--config PATH] [--plan PATH] [--home PATH] [--repo PATH]`; `fleet restore --backup <id>` uses flag form only (never bare positional, avoiding `restore` ID grammar). Exit codes: `2` usage/validation, `1` runtime/stale/conflict, `0` ok. Add `fleet` to `isHelpTopic`/`printUsage` only.

**R4 — Provenance sidecar, manifest untouched.** Keep `manifest`/`installChanges` byte-identical. New `fleet-manifest.json` (`fleetManifestVersion: 4`, beside/under backup root or `~/.agentic-sdd/fleet/`) per managed file: `{instanceID, sourceRef, sourceDigest, configDigest, rendererVersion, desiredDigest (fileDigest of staged bytes pre-rename), installedDigest, backupRef, scope, target}`. Format-3-only history → `provenance: legacy/unknown`; `fleet status` reports `unmanaged/legacy provenance unknown`; adoption requires explicit `fleet plan --adopt` preview + apply. Never reinterpret backup digest as desired digest; never mass-rewrite old manifests.

**R5 — Start-from-canonical-bytes.** Renderer reads resolved source bytes, splits frontmatter/body at second `---`, replaces/sets only the single `model:` scalar (preserving `#variant`), writes approved identity filename. Post-render assertion: body SHA-256 + parsed permission-set equality vs source, else fail. Covers FR-11/NFR tests 5, 11 and preserves MCP Code Mode (`execute` + `codegraph_*` + `context7_*` + `exa_*` lane rules).

**R6 — Offline by default + opt-in doctor.** Default `status/plan/apply/restore` perform no network/model calls; runtime claims labeled `not checked (offline)`; version claims cite only the pinned binary actually exercised. Opt-in `fleet doctor [--opencode-bin PATH]` inspects local catalog/config only, reports `configured | catalog-visible | authenticated | unknown | unavailable`; never infers entitlement from a string; `openai/*` is route only, never proof of ChatGPT subscription; no invented `9router/openai` lane.

**R7 — Disabled means omit.** `enabled: false` ⇒ managed file absent from `agents/`; plan shows `remove + back up`. Reuse proven path: planner emits `change{target: openCodeAgentsTarget(home), skill: <id>.md, isFile:true, old:true, remove:true}` → existing `installChanges` backup/rename/rollback. Unmanaged same-name file ⇒ conflict, explicit `--adopt` only. No agent-level `disabled:` field (undocumented on pinned runtime); add only behind version gate if later proven, default stays omit.

**R8 — Strict-JSON stdlib validator (`internal/fleet/parse.go`).** `io.LimitReader` 1 MiB cap → `oversize`; `utf8.Valid` → `invalid UTF-8`; two-pass decode: `Decoder.DisallowUnknownFields` + `schemaVersion==1` gate plus token-walk duplicate detector (`Decoder.Token()` per-object `map[string]struct{}` → `duplicate key`). Key hygiene: instance IDs `^[a-z0-9][a-z0-9-]{0,63}$`, `SourceRef` per R1, model per R9; reject `/ \ ..` and path separators; forbid `permissions/shell/execute/subagent/fallback*/automaticDelegation/spending` as unknown-field errors.

**R9 — Model grammar offline.** `^(provider)[a-z0-9][a-z0-9_-]{0,31}/(model)[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#(variant)[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$`; reject whitespace/control/empty segments. Catalog/entitlement deferred to `fleet doctor`. Historic lane strings are opaque fixtures.

**R10 — Canonical paths.** `specs/014-agent-fleet-model-overrides/spec.md` (this file) + `plan.md` + `tasks.md`; slug `014-agent-fleet-model-overrides` confirmed unoccupied (specs end at 013 + this dir). Original `014-agent-fleet-model-overrides-spec.md` retained as historical superseded draft.

**R11 — Code placement + gates.** New `internal/fleet/` pure package (`fleet.go` parse/validate, `resolve.go`, `render.go`, `plan.go`) stdlib-only + reuse of `skillsync` digest/install helpers via small exported accessors; thin CLI in `cmd/agentic-sdd/fleet*.go` → `runFleet`. Update **both** `Taskfile.yml` (`test`/`vet`) and `harness/checks.json` (`test`/`vet` argv) from 9 to 10 packages; `go.mod` stays dep-free (`go mod verify`/`tidy` gate proves it). Respects 008 supported-core fence (explicit package list, isolated-home offline tests).
