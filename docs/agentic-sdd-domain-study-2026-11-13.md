# Agentic-SDD — Domain and Architecture Study

**Date:** 2026-11-13
**Scope:** repository working tree at commit `870330e` plus the in-progress Feature 019 changes.
**Purpose:** identify what engineering domains this repository is actually made of, with code evidence rather than README claims.
**Status:** observational study. It is not an approval record, a batch authorization, a release note or a support claim. The live authority remains each feature's `spec.md` §Approval and `tasks.md`.

---

## 1. Measured shape of the repository

| Dimension | Observation |
|---|---|
| Go code | ~121,800 LOC across 287 files (including tests) |
| Root module | `agentic-sdd`, Go 1.23, **zero external dependencies** |
| UI module | nested `experiments/tui`, Go 1.26, 6 direct Charm dependencies, ~46,600 LOC |
| CLI code | `cmd/agentic-sdd` ~20,300 LOC (24 source files, 31 test files) |
| Internal engines | `internal/` ~54,100 LOC (skillsync 15.4k, execution 11.6k, fleet 10.3k, appops 7.2k, opencodeadapter 3.9k, harnessevidence 3.1k, harnesscheck 1.4k) |
| Tests | 1,208 Go test functions in 147 files; 213 Python test functions in 41 files; 17 shell scripts; one QA container |
| Agent content | 18 skills (~1,100 Markdown lines), 14 OpenCode agent profiles, 4 presets, 4 orchestration-pack skills, 2 adapters |
| Process artifacts | 18 feature `spec.md` files, 107 Markdown docs/specs, ~24,900 lines |
| Harness | 7 declarative checks; 11-package supported-core gate |
| Live state | Feature 019 (`asdx`/`asdx-core` rename) is approved and mid-implementation |

Measurement commands used:

```sh
find . -name '*.go' -not -path './.git/*' -not -path './dist/*' | xargs wc -l
find . -name '*_test.go' | wc -l
grep -rhoE '^func (Test|Benchmark)[A-Za-z0-9_]+' --include='*_test.go' cmd internal experiments/tui | wc -l
find docs specs -name '*.md' | xargs wc -l
```

The first finding: **the prompt/skill layer is tiny; almost all code is the machinery that installs, constrains, executes, evidences and packages it.**

---

## 2. The four-plane architecture

The repository is best understood as four coupled planes, not one product.

```text
CONTENT PLANE      skills-files/ (18)  opencode-pack/ (14 agents, 4 skills, presets)
                   "what the agent is told"

CONTROL PLANE      asdx-core CLI · asdx TUI · fleet · MCP config · skillsync installer
                   "how the human configures and operates it"

EVIDENCE PLANE     harness/checks.json · sdd-check · receipts · internal/execution runs
                   "what was actually executed and observed"

GOVERNANCE PLANE   specs/ · combined approval · tasks batches · handovers · reviews
                   "who authorized what, and what state the work is in"
```

Most repositories contain one or two of these. The distinctive property here is that all four are versioned contracts with explicit authority boundaries, enforced in both code and prose.

---

## 3. Domain 1 — Context engineering (the intellectual core)

**Artifacts:** `skills-files/` (18 skills plus 9 router references), `opencode-pack/`.

This is not merely prompt writing. It is **instruction protocol design for stochastic agents**:

- **Action-first routing.** Classify the request (limited document, review-only, direct fix, research, resume, feature) before loading anything.
- **Progressive disclosure.** Each `SKILL.md` is 15–55 lines; branch-specific material lives in `references/` behind a pointer that states when to read it. The router's 488 lines are split across 9 files.
- **One authoritative owner per rule.** Approval lives only in `spec.md` §Approval; batch state only in `tasks.md`; evidence only in the run or receipt. No duplicate ledgers.
- **Explicit authority vocabulary.** Approval, authorization, execution, evidence, review verdict and completion are separate concepts. The skills repeatedly forbid collapsing them.
- **Checkable completion.** Each stage states inputs, outputs, stop conditions and prohibited effects.
- **Context economy.** `task brief` gives a child a small packet with a digest, never a transcript. The orchestrate skill classifies delegation as `D0`–`D4` and forbids delegation by default.
- **Specialist map.** Domain skills are candidates, not mandates; “no additional specialist needed” is a valid assignment.

The problem being solved: making a probabilistic agent obey **jurisdictional boundaries** without spending its context on the rulebook.

**Maturity:** shipped as the product's core content.

---

## 4. Domain 2 — Harness engineering (the differentiator)

**Artifacts:** `harness/checks.json`, `harness/checks.schema.json`, `internal/harnesscheck`, `internal/harnessevidence`, `cmd/sdd-check`, `harness/trusted-ci.md`, `harness/evidence/receipt.schema.json`.

This is a **trustworthy execution contract**, not a test script:

- Checks are executable data: `id`, `cwd`, `scopes` (`baseline`/`focused`/`required`), `prerequisites`, `effects`, `timeout_seconds`, `steps`, `expected_exit`, `evidence`.
- `argv` is passed directly: no implicit shell, no interpolation.
- Missing prerequisite → `blocked`, never `failed` and never `passed`.
- Symlink/path-escape rejection, bounded logs (10 MiB), owner-only evidence, no silent skip.
- **Receipt v1** records an 8-part identity (base revision, diff, config, skills, environment, runtime, harness, contract), model provider/id/variant, check selection, execution state, review verdict, workflow state, findings, cost and unknowns.
- **Trusted CI** loads the runner, contract and launcher from a reviewed base, not from the candidate. `pull_request_target` is handled with base/head separation, read-only permissions and pinned actions.
- **Anti-weakening** is explicit: never add a skip, filter or tolerance to make a suite pass; `excluded` and `not_run` are not `pass`.

The current check set is `mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check`, `docker-e2e`.

The problem being solved: producing evidence that is honest about **what was observed**, from code you do not trust, without letting evidence become approval.

**Maturity:** deterministic runner and receipt validation delivered; native correlation, agent recovery and connected behavioral validation remain blocked or unobserved, and the records say so.

---

## 5. Domain 3 — Execution, evidence and recovery

**Artifacts:** `internal/execution` (~11.6k LOC), the `run`/`task`/`review` CLI families, `docs/execution-runs.md`.

A local state machine for agent work:

- Private run workspace under `.agentic-sdd/runs/<id>/`, a self-ignoring tree, `0600` artifacts, per-run writer lock, atomic checkpoint writes.
- **8 checkpoint states**: `prepared`, `running`, `observed-partial`, `observed-failed`, `awaiting-verification`, `verified-evidence`, `stale`, `blocked`.
- Task briefs are bound to a run, source and batch with path and digest provenance. A deleted or replaced brief makes the run `stale` instead of silently trusted (Feature 017).
- `run check` preserves full stdout/stderr, exit/signal, timeout and output-limit outcomes.
- `run record` stores a caller-supplied result as an **untrusted claim**.
- `review package` captures committed, staged, unstaged and untracked changes with per-file omission reasons and a `partial` manifest.

This is distributed-systems thinking applied locally: locking, idempotence, staleness, crash recovery and provenance. The hard problem is truthful state under interruption and drift.

**Maturity:** Features 016 and 017 implemented; human review is the recorded state, not acceptance by a green suite.

---

## 6. Domain 4 — Orchestration and model routing (platform engineering)

**Artifacts:** `internal/fleet` (~10.3k LOC), `opencode-pack/agents/*`, `opencode-pack/presets/free-first/*`, `opencode-pack/routing.md`, `fleet doctor`.

- Strict JSON fleet config (schema v1): an instance binds a source (`agent:<name>` or `preset:<name>`) to an optional model override and an enable flag.
- Model grammar `provider/model[#variant]`; provider lanes include `opencode-go`, `9router` and `opencode` Console free models.
- Plan identity, repository lock, deterministic rendering, provenance sidecar and rollback.
- **Offline `doctor`** validates configured routes against a catalog dump or a local binary; it calls no model and no network.
- Agent profiles encode **default-deny permission matrices**: read-only reviewers deny edit and subagent; implementers allow scoped edit and shell; secrets files are denied.

The hard problem: heterogeneous providers with different capabilities, prices and privacy terms, while refusing silent substitution or automatic paid fallback.

**Maturity:** Feature 014 fleet core and Feature 015 fleet manager delivered; the profile pack is opt-in.

---

## 7. Domain 5 — CLI engineering

**Artifacts:** `cmd/agentic-sdd` (~20.3k LOC, 24 source files).

Sixteen command families: `preview`, `apply`, `restore`, `backups`, `logs`, `guide`, `tui`, `fleet`, `mcp`, `run`, `task`, `review`, `version`, `help`, `opencode`, `orchestrate`.

Engineering characteristics:

- **Preview-first**: `preview` is the default and writes nothing; `apply` is the only writer.
- **Transaction engine**: plan → preview token → digest-bound apply → backup → manifest → verify → rollback.
- **Exit-code contract** (0/1/2), plain output on non-TTY, JSON with `schemaVersion`, per-topic help, `hint:` remediation.
- **Zero-dependency root module** is a deliberate architectural constraint.
- Cross-platform process shims, signal forwarding, bounded output.

The hard problem: making destructive operations safe by default while keeping scripts and pipes compatible.

**Maturity:** shipped and stable; the explicit CLI is the noninteractive and scripting path.

---

## 8. Domain 6 — TUI engineering

**Artifacts:** `experiments/tui` — 87 top-level files, ~46.6k LOC, 50 test files, built on **Charm Bubble Tea v2 + Lip Gloss v2 + Bubbles**.

- Four destinations today: Home, Install & Manage, Explore, Activity & Help. Feature 019 adds a fifth, Guide.
- Largest screens: `install.go` (~1.9k), `restore.go` (~1.7k), `app.go` (~1.6k), `fleet.go` (~1.4k), `ops_host.go` (~1.2k), plus catalog, MCP, activity, onboarding, modal and viewport.
- **Responsive breakpoints** at 110, 80 and 60 columns; TrueColor → ANSI-256 → ANSI-16 fallbacks; `NO_COLOR`, ASCII mode and reduced-motion mode; status is never conveyed by color alone.
- **Paired-binary handshake**: the CLI verifies the companion's build identity and protocol revision; a mismatch fails closed rather than falling back.
- **PTY lifecycle tests** in Python: launcher, terminal, fleet, MCP and read-only catalog.
- One **shared operation host** (`internal/appops`) behind CLI and TUI: the UI renders, it does not reimplement writes.

The hard problem: a safe, accessible, resumable interface over multi-step destructive operations, sharing one engine with the CLI.

**Maturity:** Feature 013 accepted and published; Feature 012 is historical evidence. The nested module is deliberate, so the root CLI keeps its dependency-free contract.

---

## 9. Domain 7 — Installer and configuration management

**Artifacts:** `internal/skillsync` (~15.4k LOC, 17 test files).

- Four client destinations: `agents`, `codex`, `claude` and `agy` (Antigravity).
- Plan, backup, install and restore with manifest v1 + v2, tree digests, no-op detection and atomic rollback.
- `go:embed` ships assets inside the binary; `--repo` reads a checkout instead.
- OpenCode profile application with backups.
- **MCP config management**: JSON/JSONC detection, digest-bound apply, conflict refusal, literal-credential detection, managed-entry ownership and restore.

The hard problem: mutating user-owned configuration across four clients without data loss, and making “no change” a first-class outcome.

**Maturity:** shipped; MCP-A/B/C delivered, verification in MCP-D.

---

## 10. Domain 8 — Release and supply-chain engineering

**Artifacts:** `.goreleaser.yml`, `.github/workflows/{ci,release}.yml`, the Homebrew cask, `internal/buildinfo`.

- Three application builds (`cli-generic`, `cli-homebrew`, `tui-companion`), universal macOS binaries, paired archives and checksums.
- The cask installs both binaries; an unsigned build is handled by a post-install quarantine removal.
- Build identity is injected via `-ldflags`; the source-content digest feeds the CLI ↔ TUI handshake.
- Pinned GitHub Actions, read-only CI permissions and exact image provisioning.
- Feature 019 is a live migration exercise: `asdx`/`asdx-core` become canonical, legacy wrappers last one deprecation window, one archive carries four names, and state/environment names are preserved.

The hard problem: renaming installed binaries without breaking scripts, gates, cask links or the paired-identity protocol.

**Maturity:** release pipeline operational; the rename is in progress.

---

## 11. Domain 9 — Process and governance engineering

**Artifacts:** `specs/001`–`019`, `docs/vision/`, `docs/handovers/`, `CONTRIBUTING.md`, and the spec/plan/tasks/review/drift skills.

The repository runs its own methodology on itself:

- `spec.md` (outcome, scope, acceptance, approval) → `plan.md` (approach, contracts, specialists) → `tasks.md` (ordered tasks, batches, verification, state).
- **One combined human approval**, recorded once and tied to revisions. Batch authorization is separate. A green suite is not authorization.
- Batch ledger states, `awaiting human review`, next action, and one handover per session.
- Architecture review, independent review, verification review, drift/retro, questionnaire, grilling, teaching and handoff.
- Twelve non-negotiable principles in the vision, including “human authority is explicit”, “tests and evidence outrank narration”, “security requires technical controls” and “be honest about support”.

The hard problem: preserving human authority and truthful state across many sessions, models and context compactions.

**Maturity:** this is the repo's operating system, applied consistently.

---

## 12. Domain 10 — QA and test engineering

**Artifacts:** 147 Go test files, 41 Python files, 17 shell scripts and one QA container.

Test types present:

- Unit and integration tests per package, golden-output tests and race tests.
- Isolated temporary homes; path-escape, symlink and digest tests.
- PTY/TUI lifecycle tests, container E2E and guide E2E.
- Harness fixtures for expected failure, cancellation, deadline, resume, receipt validation and trusted-CI anti-weakening.
- Python harness drivers for native/offline transport experiments, retained as historical evidence in the blocked scope.

The stated philosophy: derive expected results independently, avoid tautological tests, treat setup failure as not red-green evidence, and reuse evidence only while inputs match.

**Maturity:** broad and actively used as the gate for every batch.

---

## 13. Cross-cutting domains

**Security and safety engineering.** Path containment, symlink rejection, digest binding, default-deny permissions, credential redaction, owner-only logs, no implicit network or model calls, and fail-closed behavior. The documents are careful to say this is *not* a sandbox and prompt restrictions are not OS isolation.

**Protocol and integration engineering.** MCP configuration for CodeGraph, Context7 and Exa; OpenCode config formats; ACP message types in `internal/opencodeadapter` (a blocked/experimental capability, not current support); and the private `--protocol-info` handshake between paired binaries.

**Technical writing and product design.** 107 docs/specs, migration tables, accessibility contracts, error remediation and the vision document.

**Build/toolchain engineering.** Task targets for test, vet, build, Docker, hooks and release; Lefthook hooks; Go 1.23 root and Go 1.26 nested module.

---

## 14. Patterns that define the codebase

1. **Preview → token → apply → backup → verify → rollback.** Every write surface follows it.
2. **Typed service, thin edges.** `internal/appops` is the single operation host; CLI and TUI only parse and render.
3. **Observation is never authority.** Run, checkpoint, receipt, review package and verdict are all explicitly demoted.
4. **Fail closed and name the block.** Missing prerequisite, stale plan, mismatched binary or ambiguous task ID stops with a reason.
5. **Digest-bound provenance.** Source trees, briefs, plans, MCP configs, receipts and archives carry identities.
6. **Zero-dep core, isolated UI.** The root module stays stdlib-only; Charm lives in a nested module.
7. **Progressive disclosure in content and code.** Short entry files with pointed references; small interfaces with deep implementations.
8. **No silent substitution.** No PATH search, no shell fallback, no automatic model fallback, no role reversal during a rename.

---

## 15. Where the difficulty concentrates

Ranked by what would break first if handled carelessly:

1. **Authority semantics** — the approval/authorization/evidence/verdict distinction. This is the real moat and the most fragile property.
2. **Truthful state under failure** — partial writes, stale checkpoints, interrupted checks, recovery without auto-repair.
3. **Context economy for agents** — keeping 18 skills short enough to be read while precise enough to constrain behavior.
4. **Cross-surface parity** — one engine behind CLI, TUI, guide, fleet and MCP, with non-TTY and accessibility contracts.
5. **Safe mutation of user configuration** — four clients, JSONC, backups, ownership and no-op detection.
6. **Packaging and identity** — paired binaries, universal builds, cask and rename wrappers.
7. **Model heterogeneity** — capability, cost, privacy and fallback rules across providers.

---

## 16. Disciplines a contributor needs

| Skill | Why it is needed |
|---|---|
| **Go systems programming** | stdlib-only core, process control, filesystem safety, locking, atomic writes, cross-platform |
| **Terminal/UI engineering** | Bubble Tea/Lip Gloss, focus and state machines, responsive layout, accessibility |
| **Agent/prompt engineering** | instruction design, progressive disclosure, context budgeting, model routing |
| **Test/CI engineering** | Go + Python, PTY, fixtures, container, trusted CI, anti-weakening |
| **Release engineering** | GoReleaser, Homebrew cask, universal binaries, build identity |
| **Security mindset** | untrusted paths, digests, permissions, credential handling |
| **Technical writing** | specs, contracts, migration tables, honest status language |
| **Process discipline** | approval, batch, evidence, handover, review |

---

## 17. Current live state

The repository is a construction site for its own methodology. Feature 019 is approved and its implementation is in progress across parallel, disjoint lanes:

- Active docs are already migrated to `asdx` (TUI) and `asdx-core` (CLI).
- `agentic-sdd` becomes a CLI wrapper to `asdx-core`; `agentic-sdd-tui` becomes a TUI wrapper to `asdx`.
- One paired archive carries four names; `~/.agentic-sdd/` and `AGENTIC_SDD_*` are preserved.
- `harness/checks.json` and the 11-package supported-core gate are deliberately unchanged.
- The working tree contains modified docs, untracked `specs/019-cli-tui-entrypoint/`, and unrelated untracked research in `specs/018/`.

---

## 18. Verdict

> **Agentic-SDD is a Go-based agent-operations platform whose primary disciplines are context engineering (agent instruction protocols) and harness engineering (trustworthy execution evidence), delivered through a transactional CLI and a large accessible TUI, packaged as a Homebrew-installable toolkit, and governed by an unusually rigorous self-hosted spec-driven process.**

The repository is not a model, an agent runtime, an IDE or a general framework. It installs files, constrains behavior with instructions and permissions, executes deterministic checks, records observations, and deliberately stops short of claiming authority it does not have.
