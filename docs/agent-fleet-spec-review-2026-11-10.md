# Review — proposed Agent Fleet / Feature 009–013 specification

**Date:** 2026-11-10
**Status:** Read-only review; not an approval, plan, task ledger or implementation authorization.
**Reviewed material:** the proposed “Feature 009 — Optional OpenCode Orchestration Pack” summary and the proposed “Feature 013 — Agent Fleet Configuration & Profile Catalog” specification supplied in chat. Neither proposal is present in this checkout.
**Repository state inspected:** HEAD `5051299` plus the uncommitted Feature 013 B6/B7 TUI working tree; `~/.config/opencode/agents/`; `opencode models`; OpenCode v2.2.6; current OpenCode documentation via Exa.
**Related records:** [`specs/009-opencode-orchestration/spec.md`](../specs/009-opencode-orchestration/spec.md), [`specs/009-opencode-orchestration/tasks.md`](../specs/009-opencode-orchestration/tasks.md), [`specs/013-unified-tui-experience/spec.md`](../specs/013-unified-tui-experience/spec.md), [`specs/013-unified-tui-experience/tasks.md`](../specs/013-unified-tui-experience/tasks.md), [`docs/subagent-fleet-comparison.md`](subagent-fleet-comparison.md), [`docs/tui-architecture.md`](tui-architecture.md).

---

## 1. Verdict

The architecture direction is sound: **the repository should own agent roles, prompts, permissions and compatibility policy; the user should own model and provider assignments.** A declarative, preview-first configuration layer that preserves human approval and the portable SDD core is the right long-term shape.

The draft is **not safe to paste in as `specs/013-agent-fleet-manager/spec.md`**:

- Feature 013 is already the active **Unified TUI Experience** feature, with B6/B7 authorized.
- Feature 009 is already approved, implemented and released; the pasted 009 summary is stale and contradicts approved decisions.
- Several baseline facts in the proposed 013 are wrong: the “13 missing profiles” count, the “machine fleet is more restrictive” premise, the permission-diff table, and the assumption that a supported full-screen TUI already exists.
- The draft adds a second configuration/policy authority beside the existing `AGENTS.md`, `opencode-pack/`, `internal/skillsync` and Feature 009 routing rules, without defining precedence.

**Recommendation:** renumber the proposal to **014**, split it into a core fleet-override feature plus later catalog/marketplace work, and reconcile it explicitly with 009 and 013 before drafting `spec.md`, `plan.md` and `tasks.md`.

---

## 2. Blocking issues (P0)

| # | Issue | Evidence | Required correction |
|---|---|---|---|
| 1 | **Feature-number collision** | `specs/013-unified-tui-experience/` is approved and active; B6/B7 are authorized in its task ledger. A second `013-agent-fleet-manager` cannot coexist. | Renumber to **014**. Sequence it after 013/T16, or restrict v1 to root CLI work with disjoint files. |
| 2 | **The proposed 009 would reopen approved work** | `specs/009-opencode-orchestration/spec.md` is approved v6; B1–B3 are delivered and tagged `v0.6.0`. The pasted 009 says “seven subagents” and “plan/tasks not generated” — both false. | Do not replace 009. If pack scope or Option B permissions must change, record an explicit **supersession decision**. Otherwise keep 009 as-is. |
| 3 | **The “13 missing” count is misleading** | 18 machine agents − 5 overlaps = 13 not in `opencode-pack/agents/`. But 4 of those 13 (`sdd-implement-1/-2`, `sdd-test-implement-1/-2`) already exist under `opencode-pack/presets/free-first/` with different models. True missing **names** = 9. | Treat 9 as new profiles and 4 as **model variants** of existing presets. Do not create 13 new prompt files. The comparison’s “no counterpart anywhere … (agents or presets)” is false. |
| 4 | **The TUI premise is false** | The TUI is a nested experimental module (`experiments/tui/go.mod`); `docs/tui-architecture.md` says the explicit CLI remains the supported entry point. B6/B7 are still in progress and uncommitted. | Do not add a Fleet Manager to an unshipped TUI. Make 014 depend on 013/T16, or ship the fleet manager as CLI-only first. |
| 5 | **Root module is zero-dependency; YAML is not free** | `go.mod` has no `require` block. Repository guidance prefers the standard library and small explicit functions. The 013 plan explicitly keeps root `go.mod` dependency-free. | Use **JSON/JSONC** (also aligns with OpenCode config) or obtain an explicit dependency decision. Do not add YAML + JSON Schema + `internal/fleet` + `fleetstore` by implication. |
| 6 | **`--with-opencode` replacement behavior conflicts with approved 009** | The current CLI backs up and replaces the five overlapping profiles with the pack versions. 009 approved that deliberately (Option B). The draft says installation “must not replace” them and should preserve the machine’s “more restrictive” versions. | Choose explicitly: preserve old `--with-opencode` and add a new flag/preset, or make a documented breaking migration. Do not claim backward compatibility while reversing the behavior. |
| 7 | **The permission baseline is reversed for the 9 missing profiles** | Machine `sdd-implement-1`, `sdd-spec-9r`, `sdd-plan-9r`, etc. only deny `subagent`; they have **no** `*` deny baseline. Under V2 defaults, other actions may remain allowed. The five reviewers are more restrictive, but the rest are not. | Normalize imported profiles to least privilege. Do not describe the whole machine fleet as “more restrictive,” and do not preserve bare `subagent: deny` as sufficient. `sdd-spec-9r`/`plan-9r` need explicit `edit`/`shell` denies. |
| 8 | **Three-way drift detection is not supported by the current manifest** | `internal/skillsync/manifest.go` format 3 stores the **backup** digest and no last-applied desired digest or renderer version. | Add a manifest format 4 with per-file desired digest, source ID/version, scope and renderer version; define migration from format 3. |
| 9 | **There is no signing/provenance to rely on** | `.goreleaser.yml` produces `checksums.txt` only. `.github/workflows/release.yml` has no attestation, cosign or signature step. | Remove “signed/verified” claims. Either implement provenance or ship only the local curated catalog and defer remote import. |
| 10 | **“Inactive reviewer examples” has no mechanism** | OpenCode discovers every `agents/*.md`; `disabled` exists in agent configuration, but the draft does not specify whether Markdown frontmatter supports it in 2.2.6 or how the CLI toggles it. | Specify `disabled: true` and test it, or keep reviewer examples outside `agents/` until explicitly enabled. Do not claim “distributed but inactive” without evidence. |

---

## 3. Factual and security errors (P1)

| Issue | Detail |
|---|---|
| **`external_directory` is not a repo-only addition** | Machine reviewers already have `external_directory: "*"`; `docs/subagent-fleet-comparison.md` §4 says the repo adds it. That is wrong. |
| **`sdd-lightweight` diff is incomplete** | The repo version adds `skill`, `execute`, `codegraph_*`, `context7_*` and explicit denies; the comparison only mentions execute/CodeGraph/Context7. |
| **“Machine fleet: edit allowed: none” is false** | That table counts explicit rules, not effective permissions. The machine writers/spec/plan profiles have no `edit` deny, so V2 defaults may allow editing. This is the exact “effective permissions” trap the draft warns about. |
| **“Seven existing subagents”** | The repo already has 14 pack profiles + 4 presets; the machine has 18. The pasted 009/013 summaries use different baselines. |
| **“9R OpenAI” is not a current lane** | `opencode models` on 2.2.6 shows `9router/ag/...` (Claude/Gemini/Argon) and `9router/gpt-oss-*`; there is no `9router/openai/...`. OpenAI is a separate provider. Do not invent a 9R OpenAI category unless an OpenAI upstream is configured in 9Router. |
| **ChatGPT Plus is supported, but `openai/*` is not automatically subscription** | OpenCode does document ChatGPT sign-in, and `openai/gpt-6.1-sol` exists. But the provider namespace also includes API models, and ChatGPT sign-in has context caps (500k for GPT-6.1 Sol). Keep “configured / authenticated / entitlement verified” separate. |
| **Platform claim** | `.goreleaser.yml` currently builds macOS only. The spec’s “macOS/Linux” should be “supported/tested platforms,” not an implied release guarantee. |
| **Version pin** | 009 verified against 2.2.5; the machine now runs 2.2.6. Pin the exact OpenCode version(s) for permission/renderer claims and test both schema and behavior. |

---

## 4. Architecture and scope issues (P2)

### 4.1 Prompt duplication

`catalog/roles/<role>/prompt.md` plus the existing `opencode-pack/agents/*.md` creates two prompt stores. The spec says “reuse opencode-pack,” but then proposes a new role tree.

**Correction:** keep `opencode-pack/agents/` canonical. Make the catalog an index/metadata layer, and have the renderer substitute only `model`, `variant`, `disabled` and similar frontmatter. Do not migrate 14 prompt bodies for a model-switching feature.

### 4.2 Policy duplication

`fleet.yaml` proposes `providerSwitching`, `paidFallback`, `automaticDelegation` and `fallbacks`. The global `~/.config/opencode/AGENTS.md` and the repo’s 009 routing rules already own delegation and fallback policy. The example `fallbacks: []` on `sdd-implement-1` even contradicts the current fallback matrix.

**Correction:** fleet configuration should own model bindings and enablement, not delegation or fallback authority. Otherwise two policies will drift.

### 4.3 The five-profile permission diff is not a simple “revert”

009 added `execute` because OpenCode V2 exposes MCP through Code Mode; denying it removes CodeGraph/Context7. Preserving the machine reviewers would regress the approved 009 capability.

**Correction:** resolve this as a role contract (`execute` + specific MCP tools, no shell/edit) — not “keep the machine file” versus “install the pack file.” Record the decision if it changes 009 Option B.

### 4.4 The marketplace is too large for the same feature

Local catalog + recipes + remote package import + signature/provenance + prompt review + TUI + migration + 41 acceptance criteria is a multi-feature program.

**Recommended split:**

| Feature | Scope |
|---|---|
| **014** | Model/enablement overlay, safe apply/restore, status/drift. Core value. |
| **015** | Local curated catalog and starter recipes. |
| **016** | Remote package import/marketplace, only after a real package threat model and provenance tests. |

### 4.5 Remote-import security is too optimistic

Snyk’s 2026 analysis is real and relevant, but “inspect the prompt” is not a reliable security control. A remote agent prompt is executable instruction text.

**Correction:** defer remote import until allowlisting, provenance, content limits, symlink/traversal rejection and malicious-package fixtures exist — or ship local-only.

### 4.6 CLI surface is under-specified

`skillsync.Options` currently has only `WithOpenCode` and `OpenCodeFreeFirst`; there is no per-agent selection, `status`, `--config`, project scope or plan ID. The draft’s `fleet` commands are new.

**Correction:** plan the extension of `internal/skillsync` and `internal/appops`, not a parallel installer. Register any new package in `harness/checks.json`/`Taskfile`, or `task test` will not cover it.

### 4.7 Orchestrator activation is missing

Installing `sdd-orchestrator.md` does not make it the default primary. OpenCode uses `default_agent` or manual selection.

**Correction:** if 014 offers “use the orchestrator,” define an explicit activation step; do not silently change the user’s default agent.

### 4.8 Fleet discovery is hand-waved

“Dynamically resolved from the installed, enabled profile set” needs a concrete mechanism. The parent does not reliably introspect `~/.config/opencode/agents/` at runtime.

**Correction:** generate a small fleet manifest or prompt section from `status`, or document the exact discovery path.

### 4.9 No NFRs

Add measurable requirements for resolve/render time, TUI responsiveness, startup cost, large catalogs, narrow terminals, accessibility and supported OS/OpenCode versions. “Desirable” performance measurements are not enough for a TUI feature.

---

## 5. What the repository actually contains

| Item | Actual |
|---|---|
| Core skills | 18 (`skills-files/`) |
| Pack agents | 14 (`opencode-pack/agents/`) |
| Free-first presets | 4 (`opencode-pack/presets/free-first/`) |
| Machine agents | 18 (`~/.config/opencode/agents/`) |
| Pack-only names | 9 |
| Overlapping names | 5 |
| Machine names not in pack agents | 13 = 9 truly absent + 4 present only as Free-first presets |
| Root Go dependencies | none (`go.mod` is module + `go 1.23`) |
| TUI | nested experimental module, B6/B7 active |
| OpenCode | 2.2.6 on this machine |
| 009 status | approved v6, B1–B3 delivered, `v0.6.0` |

### 5.1 The 13/9 split

**Truly absent names (9):**

- `sdd-implement-9r-1`
- `sdd-implement-9r-2`
- `sdd-implement-9r-cheap-1`
- `sdd-implement-9r-cheap-2`
- `sdd-implement-free-1`
- `sdd-implement-free-2`
- `sdd-spec-9r`
- `sdd-plan-9r`
- `vision`

**Present only as Free-first presets (4):**

- `sdd-implement-1`
- `sdd-implement-2`
- `sdd-test-implement-1`
- `sdd-test-implement-2`

Most of the 9 are model/provider variants of existing roles, not new role behavior. The implementation should primarily add **bindings**, not new prompts.

---

## 6. Recommended corrected shape

```text
014 — Agent Fleet Model Overrides (core)
  opencode-pack/agents/*.md          canonical roles/prompts/permissions
  personal fleet config              enabled + model + variant only
  renderer                           substitutes model/disabled, preserves body
  manifest format 4                  desired digest + source/renderer provenance
  CLI                                status, plan, apply, restore
  reuse                              internal/skillsync, internal/appops
  no TUI dependency                  TUI integration after 013/T16
  no marketplace                     local catalog only, if any

015 — Local Fleet Catalog & Starter Recipes
016 — Remote Package Import / Marketplace (deferred, security-gated)
```

### 6.1 Decisions required before drafting 014

1. Does 014 **supersede** any part of approved 009? Recommended: no.
2. Should the 9 missing roles be **public pack defaults** or **personal presets**? Recommended: personal presets for provider-specific variants; only genuinely new roles such as `vision` belong in the public catalog.
3. Should `--with-opencode` keep its current replacement behavior? Recommended: yes, with a new `--with-fleet`/preset path for the new selection model.
4. JSON/JSONC or YAML? Recommended: JSONC.
5. TUI: wait for 013/T16, or ship CLI-only first? Recommended: CLI-only first.

---

## 7. What checked out

The Exa references themselves are real and relevant: chezmoi, OCX, OpenCode AgentHub, Agent Registry and the Snyk skill-marketplace analysis. OpenCode documentation confirms ChatGPT sign-in, `disabled`, `default_agent`, `permissions` merge behavior, and the distinction between configured models and authenticated access.

The problems are the repository baseline, the permission assumptions and the scope/security design — not the external research.

---

## 8. Review boundary

- No repository files were changed by this review.
- No OpenCode configuration was written.
- No model inference, paid call, credential access or release action was performed.
- `docs/subagent-fleet-comparison.md` remains an untracked user-supplied reference.
- This document is a technical review, not an approval, batch authorization or substitute for the human combined approval required by the repository’s SDD process.
