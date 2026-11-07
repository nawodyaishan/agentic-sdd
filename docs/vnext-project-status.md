# Agentic SDD Project Status Summary (004 & 005)

**Date:** 2026-11-03
**Scope:** Historical project-state snapshot after studying session handovers `2026-11-03_005b.md` and `2026-11-03_005.md` (now privately archived) and the two feature specifications ([`specs/004-vnext-harness/`](../specs/004-vnext-harness/spec.md), [`specs/005-vnext-completion/`](../specs/005-vnext-completion/spec.md)). Current authority remains in the owning feature approval and continuation records.
**Authority notice:** This document is a read-only status summary. Canonical approval lives only in each feature's `spec.md#approval` and batch/continuation state lives in each feature's `tasks.md#continuation-record`.

---

## 1. Where the Project Stands Right Now

| Dimension | Current State | Canonical Reference |
| --- | --- | --- |
| **Feature 004 (`004-vnext-harness`)** | Combined spec/plan/tasks approved; Batches **B1–B8**, offline **S0**, and one-request **P1** delivered. **Human full-feature acceptance approved** (`2026-11-01`). **Technical verdict remains `Blocked`** on open native, behavioral, PR-CI, and benefit findings. | [`specs/004-vnext-harness/spec.md#approval`](../specs/004-vnext-harness/spec.md#approval), [`harness/acceptance.md`](../harness/acceptance.md) |
| **Feature 005 (`005-vnext-completion`)** | Combined spec/plan/tasks `r1` approved (`2026-11-02`). All selected implementation batches (**B1–B5**, tasks **T01–T12**) are **delivered** and **B5 review is accepted**. The user-operated final human usage cycle in the pinned QA container (`agentic-sdd-qa:005`) was **reported complete and accepted by the user** (`2026-11-03`). | [`specs/005-vnext-completion/spec.md#approval`](../specs/005-vnext-completion/spec.md#approval), [`specs/005-vnext-completion/tasks.md#continuation-record`](../specs/005-vnext-completion/tasks.md#continuation-record) |
| **Next Authorized Batch** | **None in this historical snapshot.** All selected R0–R2 and Revision 6 helper implementation is complete. No further batch, model run, or live operation is authorized automatically. | [Owning 005 continuation](../specs/005-vnext-completion/tasks.md#continuation-record) |
| **Git & Working Tree** | `HEAD` matches `origin/main` at `8b76e87` (immediately following `v0.4.0` tag `b4fac4d`, which committed the `cmd/agentic-sdd/opencode_test.go` git-skip guard). Local working tree holds uncommitted post-cycle status updates in 7 tracked docs/spec files plus untracked `docs/README.md`, `docs/vnext-remaining-work-guide.md`, `docs/handovers/2026-11-03_005b.md`, and private local-only `docs/handovers/2026-10-29_004.md`. | `git status --short --branch` |

---

## 2. What Was Built

Across Features **004** (verifiable single-agent lifecycle foundation) and **005** (R0–R2 completion + Matt Pocock-inspired human-input helpers), the following components were built and packaged:

### 2.1 Portable 18-Skill SDD Suite (`skills-files/`, `skillsfiles.go`, `internal/skillsync/`)
- **Core lifecycle skills (10):** `agentic-sdd-router`, `agentic-sdd-spec`, `agentic-sdd-plan`, `agentic-sdd-tasks`, `agentic-sdd-implement`, `agentic-sdd-verification-review`, `agentic-sdd-architecture-review`, `agentic-sdd-research-spec`, `agentic-sdd-drift-retro`, `agentic-sdd-bootstrap` — updated in 004/005 for action-first routing, single completion-review ownership, contract readiness reconciliation, and conditional helper pointers.
- **Systematic diagnosis skill (1, added in 004 B5):** `agentic-sdd-debugging` for evidence-first root-cause investigation across feature work, direct fixes, and read-only reviews.
- **Human-input, maintenance, and productivity helpers (7, added in 005 B1):**
  1. `agentic-sdd-grill-me` — explicit user entry alias delegating to `agentic-sdd-grilling`.
  2. `agentic-sdd-grilling` — fact-first clarification / plan challenge (at most 3 ready, independent questions per round).
  3. `agentic-sdd-questionnaire` — recipient-specific discovery questionnaire drafting and attributed answer reconciliation (never auto-sends).
  4. `agentic-sdd-writing-for-agents` — portable maintenance guidance for skills, instructions, and references.
  5. `agentic-sdd-handoff` — canonical-first session transfer and resumption checks.
  6. `agentic-sdd-wait-what` — plain-language status/proposal explanation with zero state mutation.
  7. `agentic-sdd-teach` — response-only teaching by default, with an opt-in persistent teaching workspace branch when explicitly requested.
- **Shared references:** `workflow-policy.md`, `human-input.md`, `task-contracts.md`, `testing.md`, `review-resolution.md`, `specialists.md`, and `matt-pocock-notice.md` (28 packaged files across 18 skill directories, embedded in the Go binary and discoverable on disk).

### 2.2 Deterministic Check Runner & Trusted CI (`cmd/sdd-check/`, `internal/harnesscheck/`, `harness/`)
- **Executable check contract (`harness/checks.json`):** Versioned, argv-only (no shell interpretation) check definitions with scopes (`baseline`, `focused`, `required`), timeouts, prerequisites, and external owner-only log/result output.
- **Go runner (`cmd/sdd-check`):** Enforces sequential stop-on-failure, process-group tracking and bounded termination (`SIGTERM` then `SIGKILL`), log truncation metadata, and distinct outcomes (`completed`, `failed`, `blocked`, `cancelled`, `incomplete`).
- **Trusted CI gate (`tests/harness/trusted_ci.py`, `.github/workflows/ci.yml`, `harness/trusted-ci.md`):** Base-owned Python bootstrap + Go runner executing required checks inside a non-root, network-disabled (`--network=none`), read-only-root pinned Go container against disposable candidate staging, plus an independent base-owned no-op probe (`trusted-noop-probe.go.txt`) that detects weakened/deleted candidate tests.

### 2.3 OpenCode Runtime Adapter & CLI (`cmd/agentic-sdd/`, `internal/opencodeadapter/`, `adapterfiles.go`, `integrations/opencode/`)
- **Inactive configuration templates & policies (`integrations/opencode/`):** Namespaced `sdd-draft`, `sdd-build`, and `sdd-review` profiles, prompt-only current-session commands (`subagent: false`, no shell interpolation), higher-authority lockdown examples (`policy-lockdown.jsonc`), model catalog (`model-usage.json`), and operational guides (`safety.md`, `adoption.md`, `recovery.md`, `usage.md`, `supported-version.md`).
- **Preflight & reversible config transaction (`opencode preflight`, `opencode config preview|apply|rollback`):**
  - Emits an inactive `.opencode/opencode.jsonc` candidate targeting pinned **OpenCode `2.2.5`** (with providers/integrations denied by default and planning candidate `openrouter/nvidia/nemotron-3-ultra-550b-a55b:free` inert).
  - Performs sanitized file-only preflight inspection (rejecting duplicate keys, symlinks/FIFOs, and secret canaries).
  - Stages whole-file config replacements with a dedicated runtime-config backup manifest (separate from skill manifests), atomic same-filesystem writes, no-op on identical bytes, and digest-verified rollback that refuses to overwrite intervening user edits.
- **Narrow ACP stdio transport & pending-human normalization (`opencode session prepare|run|pending`):**
  - Pinned to OpenCode `2.2.5` ACP protocol 1 (`9bbb8ce1…`); bounds messages to 256 KiB and aggregate observations to 10 MiB.
  - Advertises `elicitation.form`, intercepts consequential `elicitation/create` and permission prompts **before** answering, cancels the owned prompt, terminates only the owned process group, and persists a sanitized pending-human question summary (`blocked` / `pending_human`) rather than letting non-interactive CLI mode continue without an answer.
  - Enforces strict fail-closed prompt admission (`admission.go`): blocks live inference while pre-send token proof, credential isolation, complete accounting, and auxiliary-retry controls are unverified.

### 2.4 Production v1 Evidence Consumer & Projection (`internal/harnessevidence/`, `internal/opencodeadapter/receipt.go`)
- **CLI surface (`agentic-sdd opencode evidence snapshot|validate|export|receipt`):**
  - Implements strict typed and semantic validation for `harness/evidence/receipt.schema.json` (v1) in pure Go without requiring Python in production.
  - Computes scoped `sha256-v1` Git/working-tree identities (covering `HEAD`, index, unstaged, untracked, deleted, and mode-changed files, failing closed on symlinks/FIFOs or pathspec magic).
  - Projects deterministic `sdd-check` results together with optional owned-transport native observations into sanitized external receipts and pending-human sidecars.

### 2.5 Pinned QA Container & Human Testing Suite (`tests/qa-container/`, `tests/*-qa-image.sh`, `docs/vnext-human-*`)
- **Turnkey image (`agentic-sdd-qa:005`):** Built offline (`--network=none`) via `tests/build-qa-image.sh` on `codercom/enterprise-base`, bundling **OpenCode `2.2.5`**, the built `agentic-sdd` binary, and all **18 packaged skills** preinstalled at `/home/coder/.agents/skills`.
- **Runners & checks:** `tests/run-qa-image.sh` (`--check` for 14 offline container checks; `--online` with explicit DNS and `/tmp` `exec` mount for TUI rendering) and `tests/vnext-online-check.sh` (DNS/HTTPS/binary checks + opt-in `--hi` model probe).

---

## 3. What Is Confirmed Working (Verified Evidence)

1. **Repository & Go Required Gates (6/6 Passing):**
   - `make test`, `make vet`, `go test -race ./...`, `git diff --check`, and `sdd-check required` (`mod-verify`, `tidy-check`, `vet`, `test`, 4-binary `build-darwin` cross-compile, `diff-check`) pass on the delivered implementation tree (`650f60d1…`), including under forced `umask 077`.
2. **18-Skill Installer & Distribution Safety:**
   - Embedded (`skillsfiles.go`) and on-disk (`--repo`) sources match across all 18 skill IDs and 28 files with valid local links and frontmatter.
   - Temporary-home unit tests and network-disabled `make docker-e2e` confirm preview-by-default, backup creation before replacement, identical-content no-op, restore, rollback on failure, preservation of unrelated/unnamespaced skills, and symlink/special-file rejection.
3. **Offline Adapter, Config Transaction, ACP Transport & Evidence Consumer:**
   - CLI smoke and unit tests confirm `opencode config preview` writes nothing to `.opencode`, `apply`/`rollback` preserve unrelated JSONC comments/keys and refuse intervening edits or symlinks, and `preflight` redacts/refuses sensitive canaries.
   - Synthetic fake-peer stdio tests confirm ACP handshake, exact model/mode selection, unanswered form/permission interception → `pending_human` sidecar + owned cancellation, provider error with exit-0 refusal, child-session usage invalidation, oversized frame refusal, and owned process-group cleanup.
   - Production Go evidence consumer (`internal/harnessevidence`) passes all 18 reference receipt test cases and matches independent golden `sha256-v1` digests from `tests/harness/receipt_tools.py`.
4. **Hosted Trusted CI on `main` Push:**
   - GitHub Actions push runs (`41082617938`, `41083183856`) executed the container-isolated trusted gate in ~70s and passed 6/6 required checks.
   - User-supplied artifact ZIP (`sha256:c62d8482…`) was inspected offline: all 9 archive members, embedded/separate `result.json`, and 6 check log digests matched.
5. **Isolated OpenCode `2.2.5` Offline Probes (004 B3 & S0):**
   - 9 credential-free native API probes (config discovery, permission evaluation, policy precedence/warnings, MCP/plugin startup denial) and 21 byte-budget/HTTP loopback + fake-SSE request-shape tests passed against the exact `2.2.5` binary (`sha256:8d060e1d…`).
6. **User-Operated Pinned-Container Human Usage Cycle (005):**
   - Pinned image `agentic-sdd-qa:005` (`sha256:35ed49cdc081db1f233d238568e7fd2455ede458377703b9eb05d0a30103258b`) passed offline `--check` (**14 passed, 0 failed, 0 skipped**) and online connectivity `vnext-online-check.sh` (**3 passed, 0 failed, 1 skipped**).
   - User logged in and ran the interactive workflow cycle inside the pinned container using `opencode/big-pickle` (`default` variant), reported everything working, and formally accepted the cycle results for that pinned image (`2026-11-03`).

---

## 4. What Is Left (Open Items, Blockers & Unproven Areas)

No further implementation batch is queued for Feature 005. Remaining items fall into **housekeeping decisions**, **carried technical validation gaps (from 004)**, **environment/platform certification**, and **excluded future scope (R3/R4)**. See [`docs/vnext-remaining-work-guide.md`](vnext-remaining-work-guide.md) for the step-by-step decision tracks.

### 4.1 Immediate Git & Documentation Housekeeping
- **Uncommitted 005 post-cycle documentation updates:** 7 modified files (`README.md`, `docs/vnext-human-*.md`, `integrations/opencode/supported-version.md`, `specs/005-vnext-completion/{spec,tasks}.md`) and untracked docs (`docs/README.md`, `docs/vnext-remaining-work-guide.md`, `docs/handovers/2026-11-03_005b.md`) are local and awaiting a user decision to commit/push.
- **Private handover protection:** `docs/handovers/2026-10-29_004.md` is a local-only forensic handover and must remain untracked and excluded from commits.
- **Scenario-level human run sheet:** While the user accepted the overall human cycle for `agentic-sdd-qa:005`, individual scenario rows (`1–10`, `12–15`) in [`docs/vnext-human-cycle-run-sheet.md`](vnext-human-cycle-run-sheet.md) were not filled out individually and remain recorded as unspecified rather than inferred per-scenario passes.

### 4.2 Open Technical Blockers Carried from Feature 004 (`harness/acceptance.md`)
Container acceptance and green offline tests do **not** close these findings:
- **Native admission & pre-send accounting (`S0-F1`, `S0-F2`, `B3-F2`, `B3-F3`):**
  - `agentic-sdd opencode session run` remains fail-closed (`blocked`) by default because strict pre-send hosted chat/tool/reasoning token projection, verified provider caps/allowance, and online egress/credential isolation are unproven.
  - In 004 P1, the single authorized live text-only attempt against `opencode/deepseek-v4.1-flash-free` returned **HTTP 429** and stopped (`1/6` original attempts consumed; `5` remain conditional and non-transferable).
- **Native tool boundaries & real-home config adoption (`B3-F1`, `B3-F4`):**
  - API permission probes and fake ACP peers do not prove real model-originated tool calls, canonical-path symlink boundaries, or live adoption in the user's real home directory.
- **Agent diagnosis & review behavioral compliance (`B5-F1`):**
  - All 8 controlled agent behavioral scenarios in `tests/harness/diagnosis-scenarios.json` (tautological tests, intermittent bugs, review-only non-mutation, disputed reviewer advice, single-review invocation) remain `not_run` (only the 12 deterministic fault oracles were run).
- **Native cancellation, compaction/restart recovery & billing provenance (`B6-F1`, `B6-F2`):**
  - Real OpenCode session cancellation during live inference, session compaction/restart/workspace-move recovery, and independent provider billing/session provenance correlation remain unverified (`cost: 0` in a redacted session export is not verified provider billing).
- **Trusted CI PR-path & branch protection (`B7-F1`, narrowed):**
  - Main-branch push runs and artifact contents are verified, but `pull_request_target` execution on an actual PR/fork head (divergent base vs. candidate) and remote GitHub branch-protection rules remain unobserved.
- **Connected native lifecycle & measured benefit comparison (`B8-F1`, `B8-F2`):**
  - No controlled, native-correlated end-to-end run (`draft → approval → 1 batch → review → interrupt/resume`) or matched baseline-vs-candidate evaluation with an undisclosed concrete holdout has demonstrated a measurable reduction in cost, delay, or human intervention.

### 4.3 Environment, Platform & Operational Boundaries
- **User's normal setup (real home):** Untouched and untested; the accepted cycle ran only in the pinned container (`agentic-sdd-qa:005`) with an inactive candidate config. Any real-home skill or config apply requires separate explicit user authorization.
- **macOS (Darwin) runtime support:** Binaries cross-compile (`make build-darwin`), but native process-group cancellation and OpenCode adapter behavior on macOS are untested and uncertified.
- **User-controlled credential/sandbox hygiene:** Old restricted (`0700`) `/tmp/opencode/sdd-b*` directories and external provider dashboard keys/caps remain under user control per [`integrations/opencode/safety.md`](../integrations/opencode/safety.md).

### 4.4 Explicitly Excluded Future Work (R3 / R4)
Excluded by user scope selection from Features 004 and 005 (require a new feature spec and approval if ever pursued):
- Independent subagent reviewer/worker delegation and multi-worktree concurrency pilots.
- Headless advisory AI CI jobs or autonomous code-changing loops.
- Custom continuity plugins, prompt warming, automatic model routing/escalation, and runtime adapters for clients other than OpenCode.
