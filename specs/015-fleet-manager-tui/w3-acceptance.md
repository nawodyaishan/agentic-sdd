# Feature 015 — Wave 3 Acceptance Evidence (2026-11-12)

**Status:** Wave 3 implementation complete and integration-green; **awaiting human review**. No acceptance, release, real-home apply or B4 authorization is implied.
**Baseline:** checkout `d5eaa7a` plus the uncommitted Feature 015 working tree.
**Scope:** W3-A doctor/live capability, W3-B scale/performance/negative evidence, W3-C PTY/accessibility/paired artifact, W3-D docs/help/onboarding.
**Parents:** [`spec.md`](spec.md), [`plan.md`](plan.md), [`tasks.md`](tasks.md).

## 1. Delivered

| Lane | Delivered |
|---|---|
| **W3-A** | Contextual Fleet diagnostics panel (`d`) with explicit local `opencode-bin`/optional catalog-dump paths; offline `not checked (offline)` default; only the six reviewed labels; no PATH scan/network/model. `buildinfo.FleetCapability` (`preview` default, `live` opt-in, invalid fails safe); injected only through the companion launch into `TUIConfig.FleetLive`. No release-config change. |
| **W3-B** | 18/100-instance scale tests for status/source preview/plan/files/diff; >1,000-line Unicode diff; 30/30 negative scenarios mapped; measured 100-row navigation/render p95. |
| **W3-C** | Fleet PTY harness (`tests/tui/test_fleet_lifecycle.py`), task `tui:fleet-pty`, launch/resize/quit/literal-input coverage; paired-binary build and identity checks. |
| **W3-D** | Fleet Manager Help topic (section 9), Quick Start pointer, optional/skippable onboarding hint, `docs/fleet-manager-tui.md`, README pointer. |

## 2. Verification

| Check | Result | Evidence |
|---|---|---|
| Root core tests (10 packages) | **Pass** | `task test` |
| Root vet | **Pass** | `task vet` |
| Nested tests | **Pass** | `go test -count=1 ./...` in `experiments/tui` |
| Nested vet | **Pass** | `go vet ./...` |
| Official nested task | **Pass** | `task -t Taskfile.experimental.yml tui:test` |
| Fleet PTY | **Pass (4 tests)** | `task -t Taskfile.experimental.yml tui:fleet-pty` |
| Existing terminal PTY | **Pass (10 tests)** | `task -t Taskfile.experimental.yml tui:pty-test` |
| Existing launcher PTY | **Pass (13 tests)** | same task |
| Paired artifact | **Built** | `task -t Taskfile.experimental.yml tui:pair`; `agentic-sdd-tui --protocol-info` / `--version` |
| Formatting / whitespace | **Clean** | `git diff --check`; `gofmt` clean on changed files |
| Negative matrix | **30/30 mapped and passing** (scenarios 8 and 9 are plumbing-level until B4; do not imply an end-to-end save was exercised) | `experiments/tui/fleet_negative_test.go` |
| Doctor | **11/11 tests pass** | `fleet_doctor_test.go` |
| Live gate default | **Preview only** | `FleetCapability` default `preview`; UI hides/disables apply/restore |

### Scale / performance (W3-B, host `linux/amd64`, i7-1165G7, 4 CPUs)

| Measurement | Result |
|---|---|
| `BuildStatus` 18 instances | median 1.844 ms; worst 2.082 ms |
| `BuildStatus` 100 instances | median 10.508 ms; worst 12.099 ms |
| `PreviewSource` 18 / 100 | median 1.88 ms / 11.797 ms |
| `PlanFiles` 18 / 100 ops | median 2.004 ms / 12.678 ms |
| `LineDiff` 2,200 lines | median 5.645 ms |
| 100-row TUI navigation/render | min 181 µs; median 194 µs; **p95 263 µs**; max 1.229 ms |
| Benchmark | 142,063 ns/op; 17,544 B/op; 443 allocs/op |

NFR-01 target (p95 ≤100 ms) is met with substantial margin. These are recorded fixture measurements, not broad guarantees.

### Paired binary identity

`agentic-sdd-tui --protocol-info` reported `protocol_revision: 1`, `distribution: generic`, revision `dev+d5eaa7a67d74-dirty`, commit `d5eaa7a67d74…`, source digest `local-sha256:247bbc47…`. The root binary was built with the `homebrew` distribution marker in the same pair.

## 3. Acceptance criteria coverage (Wave 3)

- **AC-04 / AC-05:** responsive 110/80/60 and 8/10/16, tiny-terminal guidance, literal keys, PTY resize; **covered by headless + PTY**.
- **AC-33:** keyboard/accessibility; **partial**: PTY/headless/keyboard/resize verified; dedicated Fleet reduced-motion and contrast evidence is not run.
- **AC-19 / AC-21 / AC-22 / AC-24 / AC-25:** full plan/diff, stale-plan, drift/legacy refusal, scoped restore, error guidance; **covered by engine/appops/TUI tests**.
- **AC-26 / AC-27 / AC-28:** no adopt, typed shared seam, one write slot, refresh; **covered**.
- **AC-31 / AC-35:** isolated fixtures, no real-home/network/model writes; **covered**.
- **AC-32:** p95 and large-diff reachability recorded above; **covered**.
- **AC-34:** paired artifact built and identity-inspected; **covered for this candidate**.
- **AC-36:** default `preview`, no runtime toggle, tests inject live; **covered**.
- **AC-37 / AC-38:** owner walkthrough and completion packet — packet is this document; the walkthrough remains **pending owner review**.
- **AC-02 / AC-13:** help/first-use wording; **covered**.
- **AC-15:** source preview and consent; **partially covered** (source diff preview exists; source-save consent is B4).
- **AC-06 / AC-07 / AC-08 / AC-09 / AC-10 / AC-11 / AC-12:** inventory/model/role/orchestrator; **covered by earlier waves and regression tests**.
- **AC-16 / AC-17 / AC-18:** B4 source save — **not implemented / not authorized**; these remain open.
- **AC-20:** two confirmations; **partially covered** (deployment apply confirmation exists with default-No; source-save consent is B4).
- **AC-23:** real results; **covered**.
- **AC-29 / AC-30:** 009/014 regression, toolchain isolation; **covered by `task test`/nested gates**.

## 4. Not run / not authorized

- **End-to-end source save (scenarios 8 and 9):** negative scenarios 8 and 9 test plumbing-level editor rejection and isolation; full end-to-end save execution was not exercised and is pending B4.
- **Owner walkthrough (AC-37):** not performed by the agent; requires the owner.
- **Real macOS/other-platform install and cask exercise:** not run in this environment.
- **Live deployment/restore with a real OpenCode home:** not run and not authorized; default build is `Preview only`.
- **B4 `SaveConfig` / `fleet config save`:** not implemented; source-write surface remains separately gated.
- **Feature 014 Opus/human verdict and release:** still pending. 015 live apply/restore remains blocked behind it.
- **Accessibility statistical/user study:** not run; keyboard/PTY/plain/ASCII evidence only.
- **Trusted-CI/tag/cask/tap publication:** not authorized and not performed.

## 5. Residual caveats

1. **Live capability is not enabled in release.** `FleetCapabilityMarker` defaults to `preview`; changing it to `live` in a release build is a separate release decision after 014 acceptance.
2. **B4 editor fidelity.** The production editor preserves top-level `fleet`/`preset` metadata via `LoadConfigFile`, while arbitrary formatting/key order (such as explicit `enabled:true`) and a full raw-text edit remain B4 concerns.
3. **Source save is absent.** Review Changes can show the source diff but cannot persist it; the help/docs say so explicitly.
4. **Recovery doctor is contextual.** Diagnostics is a panel, not a fifth persistent tab; it defaults offline and never searches PATH.
5. **Green tests are not acceptance.** No 014 runtime, real-home or release claim is made.

## 6. Post-review fixes

Reconciliation of findings from Gemini provisional code/architecture reviews and the verification review:

| ID | Finding | Type | Status |
|---|---|---|---|
| **M-01** | Missing `--fleet-config` flag in launcher/companion | Architecture | **Fixed in code/docs**: added to `cmd/agentic-sdd/tui.go`, companion launcher, `tuirun`, and `TUIConfig.FleetConfig`. |
| **M-02** | Top-level metadata loss during draft reconstruction | Code/Data | **Fixed in code/docs**: top-level `fleet`/`preset` preserved via `LoadConfigFile`; arbitrary key order/formatting and raw text edit remain B4 concerns. |
| **L-01** | Wide-terminal detail pane focus/elision | UI | **Fixed in code/docs**: bounds and focus checks in 110-column view verified. |
| **L-02** | Acceptance criteria overclaims (AC-15, AC-20, AC-33) | Verification | **Fixed in docs**: marked partially covered (AC-15/AC-20) and partial (AC-33). |
| **L-03** | Model-clear limitation in picker | Functional | **Documented limitation**: clearing an override to canonical inheritance is a CLI/editor operation in v1 (`docs/fleet-manager-tui.md`). |
| **A3-1** | Plan claimed `modalKind` reuse for Fleet confirmations | Architecture | **Fixed in docs**: Fleet apply/restore confirmations are inline subviews with default-No and focus-trap guards; `modal.go` remains compositor for global modals. |
| **V1** | AC-15/AC-20 source-save consent overclaim | Verification | **Fixed in docs**: marked partially covered; source-save consent is B4. |
| **V2** | Negative scenarios 8/9 implied an end-to-end save | Verification | **Fixed in docs/tests**: annotated as plumbing-level; end-to-end save pending B4. |
| **V3** | Spec approval still said Wave 2 awaiting review | Verification | **Fixed in docs**: Wave 2 acceptance recorded consistently. |
| **V4** | Task checkboxes did not reflect accepted waves | Verification | **Fixed in docs**: completed tasks ticked; B4 remains unchecked; T25 notes owner walkthrough pending. |
| **V5** | README Fleet pointer implied source save ships | Verification | **Fixed in docs**: README now says source save pending separate approval. |
| **V6** | AC-33 accessibility evidence was thin | Verification | **Fixed in docs**: marked partial; dedicated contrast/reduced-motion runs not separated. |

**Post-fix verification (2026-11-12):** `task test`, `task vet`, nested `go test ./...`/`go vet`, `tui:fleet-pty` (4/4), `tui:pty-test` (10 terminal + 13 launcher) and `git diff --check` all pass after the fixes. The new `--fleet-config` forwarding, top-level metadata preservation, and wide-detail focus tests pass. No live capability, B4 save, real-home or release operation was performed.

## 7. Recommended disposition

**Preview-only, no-go for a live release.** The TUI is safe to inspect/stage/plan and its recovery preview is exercisable against fixtures, but live deployment/restore must stay disabled until:
- Feature 014 receives its human/Opus verdict;
- a reviewed build injects `FleetCapability=live`;
- the owner accepts the Wave 3 evidence and decides B4.

**Next human action:** review this packet and the uncommitted tree; decide whether to (a) accept Wave 3, (b) authorize B4 source save, and/or (c) authorize a release candidate.
