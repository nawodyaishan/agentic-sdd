# Release readiness — Feature 008 working report (T01 provenance and ownership)

**Status:** **Preliminary.** Covers Feature 008 task **T01** only (provenance, candidate blockers, test ownership).
**Not a release verdict:** no go/no-go, no version recommendation, no release notes, no packaging or first-run claim —
those belong to B5/T09–T11.
**Authority:** the combined r1 approval is recorded once in [spec §Approval](spec.md#approval); batch state is owned
by [tasks.md](tasks.md). Nothing here creates approval, authorizes a later batch, or authorizes a live operation.
**Model used for this task:** `minimax-m3.1-flash` (opencode-go), reported as observed; no inference is drawn about its
position in any fallback list.

## 0. Method and limits

Read-only inspection of the working tree, Git metadata and file sources. The only write was this file.

- **Executed (metadata and link checks only):** Git metadata reads, `git diff --check`, a relative-Markdown-link
  resolution pass, and `find`/`sha256sum`/`diff` over documentation — **observations, not suite results**.
- **Not executed:** no production, Python, Go, container, native, model or release suite; no install or download; no
  real-home `apply`/`restore`; no `git add/commit/stash/reset/checkout`, tag, push or publication; no file
  move/rename/delete; no reading of backup **payloads**, credential stores, account records or handover contents.
- **Excluded from 008 scope:** experimental/native/P1/tokenizer/provider paths are named, not investigated (§4.4).

> **Snapshot caveat.** Every count, hash and status value below is a **T01 observation snapshot**, not current state; the
> tree is expected to change as T02 edits begin, so re-derive before use. It shows one concurrent addition by the
> **parent session** (`docs/handovers/2026-11-05_008b.md`), not another implementation agent.
> Subsequent owner-requested cleanup retained only that latest handover and privately archived/removed 33 older
> handovers. The counts below deliberately preserve the pre-cleanup inventory, not the final working-tree counts.

## 1. T01 observation snapshot — repository state

| Property | Observed | Source |
|---|---|---|
| HEAD | `0ee5aaf65df7485d872d5784b17d6435175a185f` | `git rev-parse HEAD` |
| Branch / upstream | `main` / `origin/main` | `git rev-parse` |
| Ahead / behind upstream | `0` / `0` | `git rev-list --left-right --count origin/main...HEAD` |
| **Local** tag baseline | `v0.4.0-14-g0ee5aaf`; described reachable tag `v0.4.0` | `git describe --tags --always` |
| Working tree | **not clean**: 8 modified tracked files, 22 untracked files | `git status --porcelain -uall` |
| `git diff --check` | clean (exit 0) — a whitespace check, not a test suite | `git diff --check` |

`git describe` reflects **local tags only** — a local baseline, not evidence that `v0.4.0` is the latest published
release. The tree is not a clean candidate; per plan C5, disposition of dirty/untracked inputs is an owner decision (§9).

## 2. Provenance inventory

### 2.1 Tracked structure

226 tracked paths: `tests/` 55, `internal/` 43, `docs/` 32, `skills-files/` 29, `specs/` 21, `integrations/` 12,
`harness/` 9, `cmd/` 7, `config/` 2, `audits/` 2, `.github/` 2, `.codegraph/` 1, root files 11. All tracked blobs are
regular files (`git ls-files -s` shows only `100644`/`100755`; no symlink or special entries).

The ownership boundaries in spec R5 (`cmd/`, `internal/`, `skills-files/`, `specs/`, `harness/`,
`integrations/opencode/`) are **structurally unchanged** — present, tracked, same paths and entry points. Their
*content* is **not** uniformly unmodified: inherited Feature 006 edits touch `integrations/opencode/`,
`specs/006-independent-review/`, two `tests/harness/` files and `AGENTS.md` (§2.3). "Structurally unchanged" means no
path/layout/entry-point change, not "no content drift".

### 2.2 Ignored local paths

`.gitignore` lists `.DS_Store`, `.cache/`, `backups/`, `lefthook-local.yml`, `*.test`, `bin/`, `dist/`.
`git check-ignore -v backups bin` resolves `backups` → `.gitignore:3`, `bin` → `.gitignore:6`; the ignored-root list is
exactly `.cache/`, `backups/`, `bin/`; `git ls-files -- backups bin` returns **0**. The rules are **sufficient for the
paths observed here** — a statement about these three roots only, **not** a claim that the ignore file prevents
sensitive or generated content from being committed elsewhere. No broader `.gitignore` review was performed.

- **`backups/` — metadata only; payloads not read.** One operator backup tree dated 2026-11-03 holding the four
  installer client directories (`agents`, `agy`, `claude`, `codex`) under owner-only permissions. Private prior skill
  content: **not** a release input, **not** altered, deliberately not opened.
- **`bin/` — metadata only.** Five untracked local binaries (one Linux build; four Darwin cross-builds of
  `agentic-sdd`/`sdd-check`, matching the `build-darwin` recipe outputs). **Not** candidate artifacts: they predate any
  candidate freeze and came from an unidentified tree. `dist/` does not exist, so no GoReleaser output is in the tree.

### 2.3 Untracked (22) and modified (8) paths, by observed provenance

| Group | Paths | Observed provenance |
|---|---|---|
| Feature 008 own | `specs/008-core-release-quality/{spec,plan,tasks}.md`, `…/release-readiness.md` | This batch's approved inputs and this report; untracked until the owner commits |
| Inherited Feature 006 | `specs/006-independent-review/supervised-pilot-proposal.md`; `tests/harness/native-independent-review-dispatch.sh`, `native_independent_review_dispatch.py`, `test_independent_review_dispatch.py` | Belongs to 006, **not** authorized here |
| Session handovers | 12 untracked files in `docs/handovers/` (2026-10-29_004, 2026-11-03_005b, 2026-11-04_006, 006c–006h, 2026-11-05_008, 2026-11-05_008b, 2026-11-05b) | Pointers, not approvals; `008b` added by the parent session during T01 |
| Draft status docs | `docs/README.md`, `docs/vnext-project-status.md` | **Uncertain drafts** — currency/authorship not established by T01 |
| Modified tracked (8) | `AGENTS.md`; `integrations/opencode/{independent-review,safety}.md`; `specs/006-independent-review/{plan,spec,tasks}.md`; `tests/harness/native_independent_review.py`; `tests/harness/test_independent_review_docs.py` | **Inherited 006 work with an identifiable owner**; the `AGENTS.md` edit's owner T01 did not establish |

**Handover corpus:** 34 files — 22 tracked, 12 untracked. Sizes were observed but are omitted; **contents were not read
or copied**; handovers remain secondary pointers under AGENTS.md §"Session handovers".

**Provenance honesty:** only the inherited 006 group has an identifiable owner from observed provenance. The draft
status docs and handovers are *untracked drafts of uncertain currency*, not "owned work"; T01 classifies them no further.

### 2.4 Feature records and approval anchors

| Feature | §Approval | State visible in the record |
|---|---|---|
| 004 vnext harness | `specs/004-vnext-harness/spec.md` line 174 | Retained findings |
| 005 vnext completion | `specs/005-vnext-completion/spec.md` line 394 | Accepted **container-based** cycle (narrow deviation recorded) — accepted partial scope |
| 006 independent review | `specs/006-independent-review/spec.md` line 390 | B4 **deferred by the user**; `IR-06-01` retained as a valid, high-impact access failure |
| 007 Taskfile migration | `specs/007-taskfile-migration/spec.md` line 21 | Direct trusted-branch adoption accepted; `007-F1` old-base PR-gate gap **not** waived |
| 008 core release quality | [spec §Approval](spec.md#approval) | Combined r1 approved; B1 only authorized |

## 3. Duplicate evidence — docs, history, audits

Compared by content hash and diff, not filename. **No consolidation proposed; no document changed.**

**The audit trio.** `audits/Agentic-SDD-Workflow-Audit-2026-09-24.md` and `docs/Agentic-SDD-Workflow-Audit-2026-09-24.md`
are **byte-identical** (same sha256 `00067f7ed9d6be37…`; `diff` → zero differing lines; 202 lines each) — the one real
duplicate pair. `audits/Agentic-SDD-Workflow-Audit-2026-09-24 (1).md` (sha256 `773dedcded6531ae…`, 82 lines) is **not** a
copy of either: titled *"workflow audit — corrected"*, its own text says it *"supersedes the earlier version of this
audit"*, its headings differ and the diff against the other two yields 236 differing lines. Treating `(1)` as a stray
copy would be wrong. All three were added in one commit (`2fd9f58`).

**Other groups.** The eight `docs/vnext-*` / adoption-plan / research documents have **eight distinct hashes** and
distinct heading structures — complementary, not copies. `docs/vnext-project-status.md` (untracked) overlaps
`docs/vnext-remaining-work-guide.md` in *subject*: a currency question for T02, not a byte-duplicate question.

**Inherited index.** `docs/README.md` is an untracked draft index (4 sections, 16 links). Per plan C1 it is to be
**inspected and extended only with authorized provenance, never overwritten with a second index**; T01 records its
existence and shape only.

**Link integrity (executed).** 118 Markdown files (tracked + untracked): **518** relative links, **0** unresolved; within
`skills-files/`: **67** relative links, **0** unresolved.

## 4. Test ownership and entry points (source inspection only — nothing executed)

### 4.1 The trusted required contract

`harness/checks.json` defines 7 checks; **6 carry the `required` scope**:

| ID | argv | Prereq | Local recipe | Declared effects |
|---|---|---|---|---|
| `mod-verify` | `go mod verify` | go | `task mod-verify` | Go cache; possible toolchain/module retrieval |
| `tidy-check` | `go mod tidy` | go | `task tidy-check` | `go.mod`, `go.sum`, module/cache dirs; possible retrieval |
| `vet` | `go vet ./...` | go | `task vet` | build/module cache; possible retrieval |
| `test` | `go test ./...` | go | `task test` | Go cache, temp test homes, arbitrary candidate effects → needs isolation |
| `build-darwin` | 4× `go build -trimpath -o bin/…` (darwin amd64+arm64) | go | `task build-darwin` | **writes `bin/`**; possible retrieval |
| `diff-check` | `git diff --check` | git | `git diff --check` | none; network none |

The 7th, `docker-e2e` (`sh tests/docker-e2e.sh`), is **`focused` scope only**; its prerequisites include a *preloaded*
`golang:1.25-alpine` image and it must block rather than pull implicitly. `task verify` runs `sdd-check --scope
required --contract ./harness/checks.json` with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off` and owner-only external
evidence output; `task verify-trusted` runs the **reviewed base's** `tests/harness/trusted_ci.py` against a separate
candidate and refuses to start without `SDD_TRUSTED`.

### 4.2 Go test ownership — 22 test files, 6 packages

`internal/skillsync` (install/restore/no-op/backup, manifest+digest, **the explicit 18-skill distribution contract**
and packaged-link closure in `package_test.go`) · `internal/harnesscheck` (runner contract, synthetic subprocess
fixtures, Linux process semantics) · `internal/harnessevidence` (receipt validate/export/snapshot) ·
`internal/opencodeadapter` (7 files; inactive experimental surface) · `cmd/agentic-sdd` (CLI flags, exit behaviour,
`opencode --help`) · `cmd/sdd-check` (check-runner CLI). `internal/skillsync` tests use `t.TempDir()` isolated homes.

### 4.3 Python harness ownership — core inventory (model-free / local / synthetic only)

17 files match `tests/harness/test_*.py` (what default discovery collects); 15 further modules are helpers **not**
collected by that pattern. Only these entries are candidates for 008's own release evidence; each class is read from
source:

| File(s) | Class |
|---|---|
| `test_taskfile.py` | behavioural, local: runs `task` with a **fake `go` on PATH**; asserts `--list`, `tag` refusal, `verify-trusted` refusal, `verify` env wiring, and that CI never runs the candidate Taskfile |
| `test_trusted_ci.py` | policy, stdlib — trusted-bootstrap policy checks |
| `test_receipts.py`, `test_recovery.py` | execute the local check runner in synthetic Git workspaces with a fresh credential-free `HOME` |
| `test_connected.py` | compiles and drives the runner; optional external evidence root |
| `test_independent_review_docs.py` | **source-only document-contract** checks on the packaged reference text — not evidence of model compliance |
| `test_independent_review_fixtures.py` | **synthetic fixture consumers**: scripted child reports → test-only `independent_review_contract.py`; uses the real CLI's `opencode evidence snapshot` on private scratch Git repos — no reviewer, model or VM |
| `validate_contracts.py` | schema validation of both schemas/examples (needs the existing `jsonschema` 4.x) |

### 4.4 Paths excluded from Feature 008 execution

The following are **excluded from Feature 008's execution and release evidence**. They are named only, so no blanket
discovery can pull them in; their controls, prerequisites and status stay with their canonical owners
(`tests/harness/README.md`, plan C4, [spec §2](spec.md)), and T01 did not investigate them further:

- native / client-execution: `test_s0_native.py`, `test_offline_transport.py`, `native_independent_review.py`,
  `native_independent_review_dispatch.py`, `native-opencode.sh`, `native-independent-review.sh`
- P1 provider paths: `test_p1_network.py`, `test_p1_driver.py`, `test_p1_transport.py`, `p1_native.py`, `p1-preflight.py`
- tokenizer / provider-data fixture: `test_tokenizer_source.py`
- inactive-profile / experimental-dispatch checks: `test_opencode.py`, `test_independent_review.py`,
  `test_independent_review_dispatch.py`. Model-free status alone does not make an excluded integration in scope.

Portable-reference cases in §4.3 are candidates only where they directly cover shipped core guidance; this report
does not authorize running a broad independent-review suite. No such suite was run for B1.

Separately, container-runtime suites are **not run by default in 008** and need their own authorized boundary; plan C4
assigns the trusted-isolation ones to B4/T07: `test_trusted_ci_container.py`, `trusted-ci-fixtures.sh`,
`diagnosis-fixtures.sh`.

The status of the native independent-review surface, including `IR-06-01`, is owned by
[`specs/006-independent-review/tasks.md`](../006-independent-review/tasks.md). T01 records the exclusion and proposes
no mitigation or investigation.

### 4.5 Ownership gap identified (not fixed here)

The Python harness, Taskfile and installer-E2E suites sit **outside the six-check required gate**, and their ownership is
documented only inside `tests/harness/README.md` — not in a single allowlisted entry point and not in the
contributor-facing command path. That is the surface plan C4 / task T06 must close. Any future entry point must use an
explicit allowlist, **not** blanket directory discovery — otherwise the excluded paths in §4.4 can be collected
unintentionally.

## 5. The 18 portable skills — identity, references, embeds

**Exact skill IDs (18).** `agentic-sdd-architecture-review`, `agentic-sdd-bootstrap`, `agentic-sdd-debugging`,
`agentic-sdd-drift-retro`, `agentic-sdd-grill-me`, `agentic-sdd-grilling`, `agentic-sdd-handoff`,
`agentic-sdd-implement`, `agentic-sdd-plan`, `agentic-sdd-questionnaire`, `agentic-sdd-research-spec`,
`agentic-sdd-router`, `agentic-sdd-spec`, `agentic-sdd-tasks`, `agentic-sdd-teach`, `agentic-sdd-verification-review`,
`agentic-sdd-wait-what`, `agentic-sdd-writing-for-agents`.

**Identity.** `skills-files/` holds exactly **18** directories, each with a regular-file `SKILL.md` (18/18), with **no**
symlink or special file anywhere in the tree (29 tracked files; clean working status).
`internal/skillsync/package_test.go` declares `canonicalSkills` with the same **18** IDs — an explicit distribution
contract, not a value derived from discovery — and is **set-equal** to the on-disk directories (0 missing, 0 absent).
`README.md` §"The skill set" lists the same 18 IDs; its only other `agentic-sdd-*` strings are the
`docs/agentic-sdd-architecture.png` asset path and two prose uses of the `agentic-sdd-*` prefix rule.

**Reference closure.** `skills-files/`: 67 relative links, **0** unresolved. Repository-wide: 518 across 118 files,
**0** unresolved. `package_test.go` re-checks this inside the installer via `checkPackageLinks`, over the embedded and
`--repo` sources and after installing into isolated homes — an existing automated guard, not a new one. It also asserts
`agentic-sdd-router/references/independent-review.md` is packaged and referenced from
`agentic-sdd-verification-review/SKILL.md` and `agentic-sdd-router/references/review-resolution.md`; all exist. Router
references: `human-input.md`, `independent-review.md`, `matt-pocock-notice.md`, `review-resolution.md`, `specialists.md`,
`task-contracts.md`, `testing.md`, `workflow-policy.md`; other skills carry
`agentic-sdd-questionnaire/references/questionnaire.md`, `agentic-sdd-teach/references/learning-workspace.md`,
`agentic-sdd-writing-for-agents/references/skill-mechanics.md`.

**Embed closure.** `skillsfiles.go` (module root, package `skillsfiles`) declares `//go:embed all:skills-files` over the
whole tree — 18 dirs / 29 files, all tracked and regular; because the directive names the whole directory, the embedded
set equals the on-disk set **by construction** (a source inference, not an executed embed dump). `adapterfiles.go`
declares `//go:embed` on 4 explicit paths — `integrations/opencode/{opencode.jsonc,policy-lockdown.jsonc,model-usage.json}`
and `harness/evidence/receipt.schema.json` — and **all 4 are present**. `go.mod` declares module `agentic-sdd`; release
ldflags target `-X agentic-sdd/internal/version.Version=…`. **No import path or embed path depends on anything this
feature proposes changing**, and no path change is proposed.

**Destinations (unchanged).** `internal/skillsync/sync.go` `clientTargets` fixes four in order: `~/.agents/skills`,
`~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`; `discoverSkills` selects only `agentic-sdd-*`
directories; `backupRootFor` keeps backups at `<repo>/backups` for an explicit `--repo`, else `~/.agentic-sdd/backups` —
which is why this checkout has a local `backups/` directory: an installer artifact, not repository content.

## 6. Release inputs and observed prerequisites

| Input | Content that matters |
|---|---|
| `.goreleaser.yml` | one build (`./cmd/agentic-sdd`, `CGO_ENABLED=0`, `-trimpath`); darwin amd64+arm64; `universal_binaries: replace: true` → one lipo universal binary; `tar.gz` + `checksums.txt`; ldflags inject `Version/Commit/Date/GoVersion`; `before.hooks: go mod tidy`; `brews` targets `nawodyaishan/homebrew-tap` via `{{ .Env.HOMEBREW_TAP_TOKEN }}` and **declares** a formula `test:` block |
| `.github/workflows/release.yml` | trigger `v*` tag push; `contents: write` + `id-token: write`; `environment: release`; runs `task verify`; validates the tap token via `gh api`; GoReleaser `--clean` on **`ubuntu-latest`** |
| `.github/workflows/ci.yml` | `push`/`pull_request_target` to `main`; **reviewed base** (`gate-trusted`) vs candidate as **data only** (`gate-candidate`); pinned toolchain image `golang@sha256:1ae0735f…`; Task `3.54.0` from a pinned action SHA; runs `task --dir …/gate-trusted verify-trusted` — the candidate Taskfile is never executed |
| `Taskfile.yml` | `task release` = **non-publishing local GoReleaser snapshot** (`goreleaser release --snapshot --clean`); `task tag` = annotated tag, a **separately authorized live operation**. The snapshot runs `go mod tidy` and `--clean`, so it needs bounded isolated execution and must not write into the local `bin/` |
| `internal/version/version.go` | defaults `dev`/`none`/`unknown`/`unknown` without ldflags — a local build never reports a release version |

**Distribution surface:** the published artifact is the **Darwin universal `agentic-sdd` binary only**. `sdd-check` is
cross-compiled by the `build-darwin` check but is **not** published (spec R9's "unpublished `sdd-check` executable"). The
Homebrew formula `test:` block is **declared only** — not evidence that a macOS runner executed it, and the release
workflow runs on `ubuntu-latest`, so it does not exercise that native test either.

**Observed prerequisites (versions read; nothing installed or downloaded).** Go `go1.27.1 linux/amd64` — present ·
Git `2.53.0` — present · Go Task `3.54.0` — present, **matches the CI pin** · Python `3.14.4` — present ·
`jsonschema` `4.19.2` — present · Docker CLI `29.9.1` with a responding daemon — present ·
`golang:1.25-alpine` local digest `sha256:1ae0735f…` — present, **byte-identical to the digest CI pulls** ·
`agentic-sdd-qa:005` image — present locally · Lefthook `2.2.2` — present ·
**GoReleaser — not on PATH, so a local snapshot is blocked** (installing it is not authorized) ·
**`go.sum` does not exist** although `release.yml` names it as `cache-dependency-path` — the module has no external
dependencies, so `go mod tidy` has nothing to record; flagged for B3/B5 confirmation, not changed here.
**Not observed in this session:** hosted CI run outcomes; any macOS-native behaviour.

## 7. Candidate and platform blockers

**Clean-candidate blockers.** (1) 8 modified tracked files — inherited 006 hunks plus an `AGENTS.md` edit — which must
not be silently included in, or silently excluded from, a candidate; (2) 22 untracked files — 008's own documents and
this report, inherited 006 pilot/dispatch files, 12 handovers, 2 draft status docs of uncertain currency; (3) 008's own
spec/plan/tasks are untracked, so the reviewed documents have no Git identity to pin a candidate to; (4) no `dist/` and
no snapshot artifact exist, and the local `bin/` binaries predate any candidate freeze; (5) the release version is
undecided and remains with the owner (spec §10.7).

**macOS / native-evidence blockers.** (1) This session ran on Linux `x86_64`; no macOS host was observed or provisioned
— a statement about this session's environment, not a general claim about what any account or environment could reach,
so native macOS behaviour is **unobserved here**; (2) the release workflow cross-compiles on `ubuntu-latest`, and a
cross-compiled darwin binary is **not** native macOS execution evidence (spec §10, plan C5); (3) the Homebrew formula
`test:` block is declared only — no macOS runner executing it was observed; (4) GoReleaser is absent, so the packaging
row is blocked on this host; (5) native macOS check-runner guarantees stay excluded by spec §2 — no documentation
change can certify that boundary, and T01 made no attempt to.

**Claims that cannot be made yet.** "The release artifact installs and runs on macOS" — §7 · "The required
six-check gate passes" — no Go suite was executed, and `build-darwin` writes into `bin/` · "The Python/Taskfile/harness
suites pass" — not executed · "The installer E2E passes" — `docker-e2e` not executed · "Hosted CI is green for this
candidate" — hosted evidence unobserved · "The tree is a clean candidate" — §7 · "The generic native reviewer is
safe" — out of 008 scope and false to claim; `IR-06-01` is a retained valid finding owned by
[`specs/006-independent-review/tasks.md`](../006-independent-review/tasks.md).

## 8. Findings register (T01)

| ID | Finding | Disposition | Owner / batch |
|---|---|---|---|
| T01-F1 | `audits/…09-24.md` and `docs/…09-24.md` are byte-identical | Recorded; **no move/delete proposed** (plan C1 default) | Owner, at candidate review |
| T01-F2 | `audits/…09-24 (1).md` is a corrected superseding report, not a copy | Recorded so it is not mis-consolidated | Owner, at candidate review |
| T01-F3 | `backups/` and `bin/` are ignored local artifacts | Preserved untouched; excluded from candidate inputs | B5 (T09) |
| T01-F4 | 8 modified + 22 untracked paths with mixed provenance (inherited 006 / 008-own / uncertain drafts) | Not a candidate; owner disposition required | Owner, at candidate review |
| T01-F5 | 12 of 34 handovers are untracked; handovers are pointers, not approvals | Excluded from release claims; contents not copied | T02 |
| T01-F6 | Python/Taskfile/E2E suites sit outside the six-check required gate and outside a contributor-facing entry point | Ownership gap for the release-QA entry point | B4 (T06) |
| T01-F7 | `go.sum` absent although `release.yml` names it as `cache-dependency-path` | Observation only (no external deps) | B5 (T11) |
| T01-F8 | GoReleaser absent on this host | Blocked, not waived; no install attempted | B4 / B5 (T09) |
| T01-F9 | No macOS host observed in this session; cross-compilation is not native evidence | Blocked/no-go input | B5 (T10) |
| T01-F10 | README §"Safety model" says change detection includes "file permissions"; `internal/skillsync/source.go` `sameSourceTarget` documents that permission bits are **not** compared | Confirms PQ-04 from source; documentation-first correction, no behaviour change | B3 (T05) |
| T01-F11 | The 18 IDs, packaged references, both embed sites and all relative Markdown links are internally consistent | Baseline for B3/B4/B5 | — |
| T01-F12 | `docs/README.md` (untracked draft) already indexes current/history material | Adopt-or-extend only, never overwrite | T02 |
| T01-F13 | Experimental / native / P1 / tokenizer / provider paths are **excluded from 008 execution** | Named in §4.4; status and controls stay with their canonical owners; no investigation or mitigation proposed | Canonical owners (`tests/harness/README.md`, plan C4, 006 records) |

## 9. Owner decisions — requested only when ready

T01 makes **no** decision and proposes **no** mandatory cleanup. The default stays: no moves, no deletions, no
alteration of local artifacts.

- **When a candidate is ready to be frozen:** the owner decides the disposition of the inherited 006 hunks and the
  untracked drafts, and selects the release version.
- **Nothing is requested before then** — no decision is requested on keeping the duplicated audit copy, on rebuilding
  scratch artifacts, or on any path move. Those remain the plan C1 default (no moves) unless the owner raises them.

## 10. Reproducing the snapshot

Read-only metadata and link checks; none executes a test suite, container, native client or model.

```sh
git rev-parse HEAD; git describe --tags --always; git rev-list --left-right --count origin/main...HEAD
git status --porcelain -uall; git diff --stat; git diff --check
git ls-files | wc -l; git ls-files -s | awk '$1!="100644" && $1!="100755"'           # expect: 226; no output
git status --porcelain --ignored=matching | grep '^!!'                               # expect: .cache/ backups/ bin/
git check-ignore -v backups bin; git ls-files -- backups bin | wc -l                 # expect: :3, :6, and 0
ls -1 docs/handovers | wc -l; git ls-files docs/handovers | wc -l                    # expect: 34, 22
sha256sum "audits/Agentic-SDD-Workflow-Audit-2026-09-24 (1).md" \
          "audits/Agentic-SDD-Workflow-Audit-2026-09-24.md" \
          "docs/Agentic-SDD-Workflow-Audit-2026-09-24.md"
diff -q "audits/Agentic-SDD-Workflow-Audit-2026-09-24.md" "docs/Agentic-SDD-Workflow-Audit-2026-09-24.md"
ls -1 skills-files | wc -l; ls skills-files/*/SKILL.md | wc -l                        # expect: 18, 18
find skills-files \( -type l -o ! -type d ! -type f \); git status --porcelain -- skills-files
```

Link resolution (Python standard library, read-only) covered 118 Markdown files / 518 relative links with 0 missing, and
`skills-files/` with 67 relative links / 0 missing. The canonical 18-ID comparison used the literal list in
`internal/skillsync/package_test.go` against `ls skills-files` (set-equal).

## 11. Explicitly not established by T01

- No production, test, container, native, model or release suite was executed. The only executed checks were Git
  metadata reads, `git diff --check` and relative-link resolution — **none of which is a suite result**.
- No clean candidate exists; §1 is a snapshot, not a candidate identity.
- No go/no-go, no version recommendation, no release notes, no compatibility classification.
- No hosted CI result, no macOS evidence, no packaging snapshot.
- `IR-06-01`, the deferred 006 B4, the inactive native ACP surface and the excluded experimental boundaries are
  unchanged and remain recorded in their owning documents — `IR-06-01` in
  [`specs/006-independent-review/tasks.md`](../006-independent-review/tasks.md), the experimental boundary in
  [spec §2](spec.md). T01 records it and proposes no mitigation.

**Next permitted action:** B1 (T01 + T02) verification by the parent, then `awaiting human review`. T02 must not
overwrite `docs/README.md`; B2+ require their own authorization.

## Freeze of Supported-Core Capability, Dependency and Command Map (T06a)

This section records the technical baseline and capability/dependency map frozen under Feature 008 task T06a for the supported-core product and its default-inactive experimental boundary.

### 1. Supported-Core Go Packages and Verification Ownership

The supported-core Go codebase consists of exactly 7 packages:
1. `.` (module root, package `skillsfiles`): Compile, build tag, and embed integrity for `SkillsFiles` and `AdapterFiles`. No standalone test suite; verified via compilation, package embed tests (`internal/skillsync`), and root build checks.
2. `./internal/version`: Build-time version metadata injection (`Version`, `Commit`, `Date`, `GoVersion`). Integrity verified via build/version assertion checks (`agentic-sdd version`).
3. `./internal/skillsync`: Core skill planning, backup, installation, rollback, permissions, and manifest verification. Behavioral unit and characterization tests live in `internal/skillsync/*_test.go` (including `package_test.go` and `permissions_test.go`).
4. `./internal/harnesscheck`: Deterministic check runner execution, process isolation, environment sanitization, and execution contract verification. Behavioral tests in `internal/harnesscheck/*_test.go`.
5. `./internal/harnessevidence`: Check receipt schemas, snapshot hashing, export, and verification. Behavioral tests in `internal/harnessevidence/*_test.go`.
6. `./cmd/agentic-sdd`: Core CLI entry point, flag parsing, command dispatch, and usage output. Behavioral tests in `cmd/agentic-sdd/*_test.go`.
7. `./cmd/sdd-check`: Check runner CLI executable (source-only). Behavioral tests in `cmd/sdd-check/*_test.go`.

Verification ownership distinction: `.` and `./internal/version` provide compile/embed/version metadata integrity, whereas behavioral test suites reside in `skillsync`, `harnesscheck`, `harnessevidence`, `cmd/agentic-sdd`, and `cmd/sdd-check`.

### 2. Default CLI Dispatch Surface vs. Default-Inactive Experimental Boundary

- **Supported-core CLI dispatch/help surface:** The default build exposes only core commands: `preview`, `apply`, `backups`, `logs`, `restore`, `help`, and `version`. `printUsage` advertises solely these supported commands.
- **Default-inactive `opencode` subcommand:** In default builds (compiled under `!agentic_sdd_experimental`), `cmd/agentic-sdd/opencode_disabled.go` stubs the command to exit with status 2 and diagnostic `"unavailable in this build"` immediately, before reading configuration, credentials, or launching subprocesses.
- **Experimental CLI build tag:** The actual implementation in `cmd/agentic-sdd/opencode.go` and its test file `cmd/agentic-sdd/opencode_test.go` require the `-tags agentic_sdd_experimental` Go build tag (complemented by `!agentic_sdd_experimental` on `cmd/agentic-sdd/opencode_disabled.go` and `cmd/agentic-sdd/opencode_disabled_test.go`). This severs the import edge to `internal/opencodeadapter` in all default builds and release binaries (`go list -deps ./cmd/agentic-sdd`), while `internal/opencodeadapter` itself is excluded from default verification by the explicit 7-package core selection (not by a build tag on `internal/opencodeadapter/*.go`). Both are strictly excluded from default source builds, default tests, and release binaries.

### 3. Embedded Assets: Active Core vs. Inert Resources

- **Active core assets (`skillsfiles.go`):** Embeds `all:skills-files` into `skillsfiles.Files` (`SkillsFiles`), containing the canonical 18 portable skills and referenced documentation. Extracted and installed by the core CLI into client skill directories (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, `~/.gemini/antigravity-cli/skills`).
- **Inert embedded resources (`adapterfiles.go`):** Embeds `integrations/opencode/{opencode.jsonc,policy-lockdown.jsonc,model-usage.json}` and `harness/evidence/receipt.schema.json` into `skillsfiles.AdapterFiles` (`AdapterFiles`). These resources are inert: no core command, install step, or default workflow extracts, writes, or activates them.

### 4. Command Graphs: Core Taskfile vs. Separated Experimental Tasks

- **Supported-core command graph:** The root `Taskfile.yml`, Lefthook pre-commit configuration, and `.goreleaser.yml` define and execute only the supported-core pipeline (`preview`, `apply`, `test`, `vet`, `verify`, `docker-e2e`, `hooks-install`, etc.). Default `task test` and `task vet` run the 7 supported-core Go packages, aligned with `task test-core` and `task vet-core`.
- **Separated experimental tasks:** Experimental tasks (such as `opencode-help`) are isolated in `Taskfile.experimental.yml`. This file is not automatically included in `Taskfile.yml` and is never a dependency of any default task, hook, or release step. Invoking it requires explicit `--taskfile Taskfile.experimental.yml` targeting.

### 5. Classification of Excluded Suites

The following suites and experiments remain strictly excluded from supported-core verification, CI gates, and release qualification:
- Feature 005 native ACP admission, container runtime experiments, and online model evaluation.
- Feature 006 native independent review, profile, dispatch, pilot, and C2 mitigation suites (`test_s0_native.py`, `native_independent_review*.py`, `test_independent_review*.py`, `IR-06-01` paths).
- Feature 007 historical migration and transition adoption exercises.
- Experimental adapter unit tests in `internal/opencodeadapter` and Python test `test_opencode.py`.

These suites cannot certify supported-core release readiness, cannot be run as normal dependencies, and are not marked passing by core test success.

### 6. Compatibility Note

- `agentic-sdd opencode ...` and `task opencode-help` are no longer active in default builds or root `Taskfile.yml`. Attempted execution under default builds returns exit 2 ("unavailable in this build").
- **Portable OpenCode skill installation remains fully supported:** The core installer continues to discover and install the 18 portable skills to OpenCode's standard discovery path (`~/.agents/skills`, alongside `~/.config/opencode/skills`) and other client targets. Only the separate, experimental CLI helper and runtime adapter are gated.

## Complete Core Release Verification Matrix and Development QA (B4c: T06, T07a, T08)

This section records the aggregate verification matrix, automated test evidence, and trusted-reference baseline comparison executed under Feature 008 tasks T06, T07a, and T08.

All automated verification runs were executed under strict environment isolation:
- Sanitized environment: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=`.
- No writes to real user directories: process `HOME` redirected to `/tmp/antigravity/008-b4c-b5/env-home` (0 entries before and after).
- Ephemeral build and test caches located in `/tmp/antigravity/gocache`.
- Evidence summary root: `/tmp/antigravity/008-b4c-b5/evidence-summary.json`.

### 1. Verification Matrix Across Supported-Core and Excluded Surfaces

| Surface / Check | Entry Point / Command | Environment / Scope | Exit Code | Result | Evidence / Notes |
|---|---|---|---|---|---|
| **Required Core CI Checks** (`task verify`) | `sdd-check --scope required --contract ./harness/checks.json` via `task verify` | Host offline, `SDD_OUTPUT=/tmp/antigravity/008-b4c-b5/verify` | 0 | **PASS (6/6 checks passed)**: `mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check` | `/tmp/antigravity/008-b4c-b5/verify/result.json`; contract SHA-256: `f42851cc793fb384ce26ea3ecc428caf748a454396a2017892ab7e5029d49b17`. Note: `docker-e2e` is focused scope, not required. |
| **Parity Aliases (Test)** | `task test-core` | Host offline, 7 core packages | 0 | **PASS (7/7 packages)**: `.` (no test files), `./internal/version` (no test files), `./internal/skillsync` (ok), `./internal/harnesscheck` (ok), `./internal/harnessevidence` (ok), `./cmd/agentic-sdd` (ok 0.314s), `./cmd/sdd-check` (ok) | `/tmp/antigravity/008-b4c-b5/logs/02-task-test-core.log` |
| **Parity Aliases (Vet)** | `task vet-core` | Host offline, 7 core packages | 0 | **PASS** (vet clean across all 7 supported-core packages) | `/tmp/antigravity/008-b4c-b5/logs/02-task-vet-core.log` |
| **Portable 18 Skills & Reference Closure** | `go test ./internal/skillsync` (`package_test.go`, `permissions_test.go`, etc.) | Isolated temp homes (`t.TempDir()`) | 0 | **PASS**: 18/18 canonical skills verified, 4 client targets (`.agents/skills`, `.codex/skills`, `.claude/skills`, `.gemini/antigravity-cli/skills`), 0 broken links across 67 skill links and 518 repo links, embed closure via `skillsfiles.go` | `internal/skillsync` test logs; characterization baseline |
| **Isolated Docker Installer E2E** | `sh tests/docker-e2e.sh` | Docker container, `--network none --pull=never --mount src=repo,dst=/src,readonly golang:1.25-alpine` | 0 | **PASS**: clean-container lifecycle: preview (72 plans), apply (72 installed), no-op repeat apply, backup creation, restore, and restore-undo | `/tmp/antigravity/008-b4c-b5/logs/06-docker-e2e.log` |
| **Shipped Workflow & Authority Scenarios** | `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py` | Local Python stdlib, synthetic workspaces | 0 | **PASS (11/11 tests)**: static instruction & scenario checks for feature/fix/review routes, negative authority stop points (bare continue, stale handoff, ready verdict, approval vs batch) | `/tmp/antigravity/008-b4c-b5/logs/03-test_core_workflow.py.log`; online model evaluation explicitly **not_run** |
| **Taskfile Integrity & Core Selectors** | `python3 -B -m unittest discover -s tests/harness -p test_taskfile.py` | Local Python stdlib, fake `go` on PATH | 0 | **PASS (10/10 tests)**: Taskfile `--list`, tag refusal, verify-trusted refusal, sanitized verify env, core selector parity | `/tmp/antigravity/008-b4c-b5/logs/05-test_taskfile.py.log` |
| **Core Go Package Selection Parity** | `python3 -B -m unittest discover -s tests/harness -p test_supported_core_go.py` | Local Python stdlib | 0 | **PASS (8/8 tests)**: verifies all 7 core packages selected consistently across `Taskfile.yml`, `checks.json`, and `tests/supported-core-go.sh` | `/tmp/antigravity/008-b4c-b5/logs/05-test_supported_core_go.py.log` |
| **Check Runner Receipts & Recovery Contracts** | `test_receipts.py`, `test_recovery.py`, `test_connected.py` | Local Python stdlib, synthetic Git workspaces | 0 | **PASS (23/23 tests total)**: `test_receipts.py` (18/18 pass), `test_recovery.py` (2/2 pass), `test_connected.py` (3/3 pass) | `/tmp/antigravity/008-b4c-b5/logs/03-*.log` |
| **Receipt Schema Validation** | `python3 -B tests/harness/validate_contracts.py` | Python `jsonschema` 4.x | 0 | **PASS (12/12 schema cases)**: positive and negative cases for `receipt.schema.json` and gate schemas | `/tmp/antigravity/008-b4c-b5/logs/04-validate_contracts.log`; semantic model scenarios **not_run** |
| **Trusted CI Policy & Invariant Checks** | `python3 -B -m unittest discover -s tests/harness -p test_trusted_ci.py` | Local Python stdlib | 0 | **PASS (14/14 tests)**: trusted-bootstrap policy, non-execution of candidate Taskfile, strict isolation | `/tmp/antigravity/008-b4c-b5/logs/05-test_trusted_ci.py.log` |
| **Container Anti-Weakening & Dual Probes** | `python3 -B -m unittest discover -s tests/harness -p test_trusted_ci_container.py` | Container runtime, `/tmp/antigravity/sdd-ci-evidence` | 0 | **PASS (6/6 tests)**: includes base-owned probe injection (`zz_sdd_trusted_noop_test.go` and `zz_sdd_trusted_boundary_test.go`), verifying failure on deleted tests and experimental activation mutants | Verified in B4b suite run; anti-weakening holds |
| **Designated Trusted Reference Gate (T07a)** | `tests/harness/trusted_ci.py run --trusted head-base --candidate repo` | Base `HEAD` (`181ed7db26983241ef3c1598d0e43429f48ff91f`) vs candidate | 2 | **BLOCKED (contract-comparison)**: exact error `"required contract removed/replaced; reviewed adoption required"` | `/tmp/antigravity/008-b4c-b5/t07a-vs-head/evidence/gate.json`; candidate replaced `go vet ./...` / `go test ./...` with explicit 7-package selector in `harness/checks.json` |
| **Feature 005 Native ACP & Online Model Evals** | `tests/harness/test_s0_native.py`, `native-opencode.sh` | Native runtime / provider API | — | **EXCLUDED / NOT RUN**: excluded by Feature 008 validation scope fence; no provider API or online evaluation attempted | Documented in spec §2, plan §1; cannot certify core release |
| **Feature 006 Native Independent Review & Pilot** | `native_independent_review*.py`, `test_independent_review*.py` | Native reviewer / subprocess VM | — | **EXCLUDED / NOT RUN**: retained valid access finding `IR-06-01` owned by Feature 006; excluded from core evidence | Documented in spec §2, plan §1; cannot certify core release |
| **Feature 007 Historical Migration Exercises** | Historical Makefile transition scripts | Legacy build files | — | **EXCLUDED / NOT RUN**: transition exercises complete; excluded from active gate | Documented in spec §2, plan §1 |
| **Experimental OpenCode Adapter Tests** | `go test ./internal/opencodeadapter`, `test_opencode.py` | Excluded from default 7-package core selection; CLI import severed via `!agentic_sdd_experimental` | — | **EXCLUDED / NOT RUN**: default builds compile without experimental tag; adapter is excluded from core release | Documented in T06a capability freeze |

### 2. T07a Trusted-Reference Outcome and Honest Gate Status

Task T07a executed the trusted-CI gate runner (`tests/harness/trusted_ci.py`) using the pre-B4b committed baseline (`HEAD` `181ed7db26983241ef3c1598d0e43429f48ff91f`) as the trusted reference against the current candidate checkout:
- **Command:** `GOCACHE=/tmp/antigravity/gocache GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off python3 -B /tmp/antigravity/008-b4c-b5/head-base/tests/harness/trusted_ci.py run --trusted /tmp/antigravity/008-b4c-b5/head-base --candidate /home/nawo-ubuntu1/github/agentic-sdd --output /tmp/antigravity/008-b4c-b5/t07a-vs-head`
- **Exit Code:** `2` (Execution: `blocked`, Phase: `contract-comparison`)
- **Recorded Gate Reason:** `invalid/unavailable bootstrap, input, contract or fresh evidence; no fallback`
- **Contract Difference Cause:** The candidate properly replaced the two broad required steps (`go vet ./...` and `go test ./...`) with the explicit 7-package supported-core selection in `harness/checks.json` (contract SHA-256 `f42851cc793fb384ce26ea3ecc428caf748a454396a2017892ab7e5029d49b17`), while the base required broad `go vet ./...` / `go test ./...`.
- **Exact Comparison Diagnostic:** `required contract removed/replaced; reviewed adoption required` (recorded in `/tmp/antigravity/008-b4c-b5/logs/34-t07a-contract-reason.log` and gate receipt `/tmp/antigravity/008-b4c-b5/t07a-vs-head/evidence/gate.json`).
- **Review-Required Paths Identified by Gate:** `Taskfile.yml`, `cmd/agentic-sdd/opencode_disabled_test.go`, `cmd/agentic-sdd/opencode_test.go`, `harness/checks.json`, `tests/harness/test_supported_core_go.py`, `tests/harness/test_taskfile.py`, `tests/harness/test_trusted_ci.py`, `tests/harness/test_trusted_ci_container.py`, `tests/harness/trusted-boundary-probe.go.txt`, `tests/harness/trusted_ci.py`, `tests/supported-core-go.sh`.

**Interpretation:** Under the trusted-CI security policy, a candidate cannot unilaterally alter the required CI contract of the trusted base. Because the trusted base pre-dates the B4b contract update, the gate correctly and honestly halts at `contract-comparison` refusing to execute candidate code. Consequently, **Gate G-008-01 and Task T07a remain honestly BLOCKED**. They cannot pass until the repository owner commits the reviewed B4b changes and designates that commit as the new trusted reference base.

---

## Preliminary Candidate Smoke, Compatibility Inventory, Release Notes & Go/No-Go Package (B5: T09, T10, T11)

This section records the preliminary candidate packaging status (T09), isolated first-run and disabled-boundary smoke verification (T10), compatibility inventory, draft release notes, acceptance criteria reconciliation, and final technical go/no-go recommendation (T11).

### 1. T09 Candidate Payload & Packaging Status (BLOCKED)

An honest release candidate requires a clean working tree matching an exact immutable Git revision, reproducible release packaging artifacts, and a functional packaging toolchain.

- **Candidate Baseline & Git Status:**
  - Base Commit: `181ed7db26983241ef3c1598d0e43429f48ff91f` (`main...origin/main`).
  - Working Tree State: **DIRTY** (`vcs.modified=true`). The tree contains 17 modified tracked files (uncommitted Feature 008 B4a–B5 deliverables, plus inherited `AGENTS.md` spending-policy deletion) and 5 untracked files (`Taskfile.experimental.yml`, `cmd/agentic-sdd/opencode_disabled.go`, `cmd/agentic-sdd/opencode_disabled_test.go`, `docs/vnext-project-status.md`, `tests/harness/trusted-boundary-probe.go.txt`).
  - Module Version Stamp: `agentic-sdd v0.4.1-0.20261106181427-181ed7db2698+dirty` (embedded via `go version -m`). The presence of `+dirty` precludes treating these binaries as an immutable release candidate payload.
- **Packaging Tooling (`goreleaser`):**
  - `which goreleaser`: Not installed on host PATH.
  - Impact: `task release` (GoReleaser snapshot pipeline) could not be executed on this host. Packaging parity is blocked pending GoReleaser availability on a clean commit.
- **Disposable Candidate Builds:**
  To verify cross-compilation and dependency boundary integrity, disposable binaries were compiled in `/tmp/antigravity/008-b4c-b5/dist/` under sanitized offline settings (`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= go build -trimpath`):

| Target Platform | Binary Path | Size (bytes) | SHA-256 Checksum | Binary Format / CGO |
|---|---|---|---|---|
| `linux/amd64` | `/tmp/antigravity/008-b4c-b5/dist/agentic-sdd` | 5,193,460 | `7ec0d26defc3f3793ba56140ade7af1b48ce5aab38cdda585d54d46774250b96` | ELF 64-bit LSB executable, x86-64, statically linked, not stripped (cgo: true) |
| `darwin/amd64` | `/tmp/antigravity/008-b4c-b5/dist/agentic-sdd_darwin_amd64` | 5,184,640 | `5b0ca0d33d4da9798960b98fd35ec0dd64600839145c10dbc0b8ee15776b22fa` | Mach-O 64-bit x86_64 executable (cgo: false) |
| `darwin/arm64` | `/tmp/antigravity/008-b4c-b5/dist/agentic-sdd_darwin_arm64` | 4,895,218 | `2b0b4b62bf0117c30dfe126bcc667f92b053dc3c2caa9fb0530a97927a3bde75` | Mach-O 64-bit arm64 executable (cgo: false) |

- **Compiled Dependency & Boundary Inspection:**
  - `go version -m` and `go list -deps ./cmd/agentic-sdd` confirm that `agentic-sdd/internal/opencodeadapter` is **COMPLETELY ABSENT** from the compiled dependencies of all three binaries.
  - Binary string inspection confirms that the string `agentic_sdd_experimental` appears solely within the user-facing diagnostic text of the disabled stub (`"error: opencode: unavailable in this build (requires -tags agentic_sdd_experimental)"`). No experimental logic or native adapters are compiled.
  - Unpublished tooling check: `sdd-check` remains an unpublished source-only development tool; it is not bundled into the release distribution.

### 2. T10 First-Run, Disabled-Boundary Smoke & Platform Execution

- **Isolated Linux Smoke Lifecycle:**
  Smoke tests were driven against the Linux candidate binary using isolated test homes (`--home /tmp/antigravity/008-b4c-b5/smoke-home`) and an empty process environment `HOME` (`/tmp/antigravity/008-b4c-b5/env-home`):
  1. `--version`: Exit code 0; outputs `agentic-sdd dev / commit: none / date: unknown / go version: unknown`. 0 files created.
  2. `--help`: Exit code 0; usage message lists only supported-core commands (`preview`, `apply`, `backups`, `logs`, `restore`, `help`, `version`). The `opencode` subcommand is completely absent.
  3. Disabled boundary check (`opencode --help` and `opencode session run`): Exit code 2; stderr reports `error: opencode: unavailable in this build (requires -tags agentic_sdd_experimental)`. 0 files or directories written to smoke home or process home.
  4. `preview --home smoke-home`: Exit code 0; generates exactly 72 plan lines (18 skills across 4 client destinations: `.agents/skills`, `.codex/skills`, `.claude/skills`, `.gemini/antigravity-cli/skills`). 0 files written.
  5. `apply --home smoke-home` (initial installation): Exit code 0; installs all 72 skills (18 per destination; 29 files in `.claude/skills`). Creates 1 initial backup marker directory and 1 log file.
  6. Repeat `apply --home smoke-home` (no-op verification): Exit code 0; reports `"All skills are up to date."`. Creates 0 new backup directories (count remains 1).
  7. Mutation & Backup Manifest Verification:
     - Mutated `.claude/skills/agentic-sdd-plan/SKILL.md` (original SHA-256 `41f23857e659da0aa75c2587ebd7b9c96a0c30c9d1c5fb4c7cc4c773d5cfa320` modified to hash `d8a2ee779f16c2dbfeeb2e62c69d7e874e9be03bcb9b036b090436bddd47208b`).
     - Executed `apply`: Exit code 0; output shows `replace + back up claude/agentic-sdd-plan`. Overwrites the mutated skill file with canonical content (`41f23857...`) and takes a backup immediately before replacement.
     - Confirms creation of format-2 backup manifest (`backup_id: 20261107T065405.114742569Z`, manifest tree digest `b59fae189b2abf9fa0875a7239797f87e4b6f8a143a95924ab5660aac07a0764`), safely preserving the pre-replacement mutated file bytes. Installed file matches canonical hash `41f23857...`.
  8. `backups --home smoke-home`: Exit code 0; lists the 2 historical backup records.
  9. `restore apply 20261107T065405.114742569Z --home smoke-home`: Exit code 0; reports `"Restored 1 skills"`. Faithfully restores the backed-up pre-apply mutated file content (`sha_after_restore: d8a2ee779f16c2dbfeeb2e62c69d7e874e9be03bcb9b036b090436bddd47208b`). Automatically creates a restore-undo backup directory (backup directory count increases to 3).
  10. `restore preview 20261107T065405.114742569Z --home smoke-home`: Exit code 0; confirms `"up to date claude/agentic-sdd-plan; Preview only."`.
  11. `logs --home smoke-home`: Exit code 0; records chronological execution log entries culminating in `Command: restore 20261107T065405.114742569Z; Result: success`.
  - Process isolation confirmation: Throughout the entire smoke run, `/tmp/antigravity/008-b4c-b5/env-home` remained strictly empty (0 entries), confirming that `--home` flag isolation was completely respected.
- **Native macOS & Homebrew Distribution Execution:**
  - Status: **UNOBSERVED (dropped as release requirement per 2026-11-07 owner amendment)**.
  - Rationale: The host environment is Linux `x86_64`. Cross-compilation of Darwin Mach-O binaries on Linux does not constitute native execution evidence. Per the 2026-11-07 documentation amendment, the owner explicitly directed: *"drop Native macOS / Homebrew evidence — AC-14 / T10 as a requirment, linux ci runner is enough"*. Native macOS runtime execution and Homebrew formula testing remain unobserved (not pass). Darwin cross-compilation and package integrity (universal binary, embedded skills, ldflags, no experimental dependencies) remain verified. Historical reports remain dated history and are not retroactively green. Linux local QA is NOT hosted CI observation; hosted CI observation on GitHub Actions remains required before public release.

### 3. T11 Compatibility Inventory & Migration Notes

1. **Developer Runner Migration (Makefile -> Taskfile.yml):**
   - **Change:** Build, test, and verification automation migrated from legacy `Makefile` to `Taskfile.yml` (Task 3.54.0).
   - **Impact:** Breaking change for contributors and developer workflows. Commands such as `make test`, `make vet`, and `make verify` are superseded by `task test`, `task vet`, and `task verify`. CI workflows and Lefthook hooks now execute Task directly.
2. **Default CLI Build Boundary and Disabled `opencode` Command:**
   - **Change:** In default source builds and release binaries, the experimental `opencode` subcommand is compiled out via Go build tag `!agentic_sdd_experimental` on CLI entry points (`cmd/agentic-sdd/opencode_disabled.go`), severing the import edge to `internal/opencodeadapter` (`go list -deps ./cmd/agentic-sdd`). Attempting to invoke `agentic-sdd opencode ...` exits immediately with status 2: `"error: opencode: unavailable in this build (requires -tags agentic_sdd_experimental)"`.
   - **Impact:** Any external script or workflow expecting the experimental CLI helper in default builds will receive exit code 2 without side effects.
   - **Preserved Core Support:** Portable OpenCode skill discovery and installation (`~/.agents/skills`) remains fully active in the core installer. Experimental tasks are isolated in `Taskfile.experimental.yml`.
3. **Supported-Core 7-Package Scope Alignment:**
   - **Change:** Default `task test`, `task vet`, pre-commit hooks, and `harness/checks.json` select exactly the 7 supported-core Go packages (`.`, `./internal/version`, `./internal/skillsync`, `./internal/harnesscheck`, `./internal/harnessevidence`, `./cmd/agentic-sdd`, `./cmd/sdd-check`).
   - **Impact:** Development commands no longer run broad `go test ./...` which previously pulled in experimental packages (`internal/opencodeadapter`).
4. **File Permission & Umask Semantics:**
   - **Change:** Documentation corrected (PQ-04/T05) to clarify that skill change detection compares byte contents and symlink status, not permission bits.
   - **Impact:** Clarification only. Installed skill file permissions reflect target directory inheritance and the active process umask (`022` default produces `0755` directories and `0644` files). Format-2 backup manifests record tree digests and metadata faithfully.

### 4. Version Recommendation

- **Recommended Release Tag:** Next minor release `0.x` (e.g., **`v0.5.0`** or next minor designated by the repository owner, once all release blockers are resolved, following the repository's latest `v0.4.0` release tag and module stamp `v0.4.1-0.20261106181427-181ed7db2698+dirty`).
- **Rationale:**
  - Breaking developer workflow migration (Makefile to Taskfile).
  - Explicit disabling of the experimental `opencode` CLI subcommand in default builds.
  - Core installer user CLI flags, backup manifest format 2, and destination directories remain fully backward-compatible.
  - Per spec §10.7 and engineering principles, jumping to `v1.0.0` is inappropriate: native macOS execution remains unobserved on Linux CI, and the experimental native ACP/review boundaries are fenced off rather than stabilized.
  - Final tag selection and creation remain an exclusive human owner prerogative.

### 5. Draft Release Notes

```markdown
# Release Notes — agentic-sdd 0.x (Candidate Draft)

## Highlights

- **18 Portable SDD Skills:** Ships the complete, canonical set of 18 Agentic SDD workflow skills, verified across 4 client targets: OpenCode/Agents (`~/.agents/skills`), Codex (`~/.codex/skills`), Claude Desktop (`~/.claude/skills`), and Antigravity CLI (`~/.gemini/antigravity-cli/skills`).
- **Safe Installer Engine:** Preview-by-default execution (`preview`), automated additive backups (format-2 manifests with tree digests and timestamps) before skill replacement, atomic restore capabilities (`restore apply <id>`), and rollback undo tracking.
- **Permission & Umask Semantics:** Explicit documentation and test coverage confirming content-based change detection, preserving user directory inheritance and process umask.
- **Default-Inactive Experimental Boundary:** The experimental `opencode` CLI helper is compiled out of default release builds (returning status 2 diagnostic), preventing unintended configuration or provider side effects. Experimental tasks reside in `Taskfile.experimental.yml`.
- **Modernized Core Tooling:** Developer workflows adopt `Taskfile.yml` (Task 3.54.0), Lefthook pre-commit hooks, and scoped 7-package core testing.

## Breaking Changes & Compatibility

- **Taskfile Migration:** Contributors must use `task <target>` instead of `make <target>`.
- **CLI Subcommand Gating:** `agentic-sdd opencode` is disabled by default (requires `-tags agentic_sdd_experimental`). Standard skill installation to `~/.agents/skills` remains fully functional.

## Known Limitations & Excluded Surfaces

- **macOS / Homebrew Execution:** Darwin universal binaries are cross-compiled on Linux; native macOS runtime execution and Homebrew installation remain unobserved.
- **Experimental Native Integrations:** Native ACP runtime experiments, online provider evaluation, and native independent review pilots (`IR-06-01`) remain excluded from this core release.
```

### 6. Verification Deviations Register (B4c / B5)

The following deviations were observed during B4c and B5 execution (preserved from `evidence-summary.json`):

| ID | Description | Expected | Observed | Resolution / Impact |
|---|---|---|---|---|
| **DEV-01** | T07a Gate Output Path | Expected `t07a-vs-head/summary.json` | Runner writes `t07a-vs-head/evidence/gate.json` | Evidence path confirmed; exact reason string recovered from adapter log (`34-t07a-contract-reason.log`). |
| **DEV-02** | Smoke Client Destinations | Expected `.claude`, `.cursor`, `.agents`, `.github` | `internal/skillsync` defines `.agents`, `.codex`, `.claude`, `.gemini/antigravity-cli` | Tested actual canonical 4 targets; brief destination list was stale. |
| **DEV-03** | Restore CLI Invocation Syntax | Expected `restore --latest --yes` | CLI syntax is `restore apply <id>` and `restore preview <id>` | Used canonical syntax; `--latest` is not a recognized flag. |
| **DEV-04** | Initial Backup Marker Count | Expected zero backups on repeat apply | First apply on clean home creates 1 initial marker dir; repeat apply adds 0 new backups | No-op behavior confirmed; initial marker directory is standard installer behavior. |
| **DEV-05** | Experimental Build Tag String | Expected build tag string completely absent | Tag string present in disabled stub message text | Absence of compiled experimental code and dependencies confirmed via `go version -m` and `go list -deps`. |

### 7. Acceptance Criteria Reconciliation (AC-01 through AC-23)

| Criterion | Description | Status | Evidence / Reconciliation |
|---|---|---|---|
| **AC-01** | Supported vs experimental matrix prominent and grounded | **VERIFIED (awaiting human review)** | Matrix frozen in T06a (§Freeze above) and verified across B4c matrix; fully grounded in source inspection and build dependencies. |
| **AC-02** | Route authority and side-effect boundaries preserved | **VERIFIED (awaiting human review)** | Verified in T04 static inspection and `test_core_workflow.py` (11/11 tests pass). |
| **AC-03** | Incomplete requests & static vs model behavior labeled | **VERIFIED (awaiting human review)** | `test_core_workflow.py` passes 11/11; static scenario tests labeled; online model evaluation honestly marked `not_run`. |
| **AC-04** | Negative authority scenarios prevent unauthorized continuation | **VERIFIED (awaiting human review)** | Negative controls verified in `test_core_workflow.py` (bare `continue`, stale handoff, ready verdict, approval vs batch). |
| **AC-05** | Implementation/debugging/review guidance standardized | **VERIFIED (awaiting human review)** | Completion-review ownership and concise risk-relevant outcomes unified across router and verification guidance in B2/B3. |
| **AC-06** | Historical status contradictions resolved | **VERIFIED (awaiting human review)** | PQ-02 status contradictions resolved in B2/B3 documentation reconciliation; historical approvals and findings preserved intact. |
| **AC-07** | IR-06-01 and experimental restrictions visible | **VERIFIED (awaiting human review)** | Documented in `README.md`, `release-readiness.md`, and candidate release notes; generic native review is explicitly not claimed safe. |
| **AC-08** | Permission comparison, umask, no-op, backup/restore semantics aligned | **VERIFIED (awaiting human review)** | Documentation corrected in T05; verified in `internal/skillsync` unit tests and B5 T10 isolated smoke lifecycle. |
| **AC-09** | Tracked/ignored/untracked structure inventory established | **VERIFIED (awaiting human review)** | Implementation committed at `9f6dbae` and clean worktree created at `/tmp/opencode/008-packaging-9f6dbae`; historical status draft (`9467264`) and inherited `AGENTS.md` edits (`a2e874c`) committed separately; documentation amendment staged as separate commit following implementation. |
| **AC-10** | 18 skill IDs, local references, and embeds intact | **VERIFIED (awaiting human review)** | Verified in `internal/skillsync` (`package_test.go`), 18 canonical skill directories, 4 client targets, 0 unresolved links. |
| **AC-11** | Full release verification matrix recorded | **VERIFIED (awaiting human review)** | Complete matrix recorded in B4c (§1 above), specifying exact commands, environments, checksums, and exit codes. |
| **AC-12** | Trusted CI invariants and container anti-weakening hold | **VERIFIED (awaiting human review)** | Verified by `test_trusted_ci.py` (14/14 pass) and `test_trusted_ci_container.py` (6/6 pass) with dual anti-weakening probes. |
| **AC-13** | First-use installation and worked examples repeatable | **VERIFIED (awaiting human review)** | T04 verified worked examples; T10 verified 4 client targets in isolated Linux smoke run. |
| **AC-14** | Packaging, CLI metadata, Linux first run, and distribution verified on same candidate | **VERIFIED (awaiting human review)** | Verified on clean worktree from commit `9f6dbae` using checksum-verified GoReleaser v2.20.0 (archive SHA256 `d2483159...`): snapshot succeeds (`dist/agentic-sdd_0.4.0-SNAPSHOT-9f6dbae_darwin_all.tar.gz` SHA256 `23c7a77c...`) with universal Mach-O binary and generated Homebrew formula (deprecation warning on brews field retained). Clean Linux smoke verified at `/tmp/opencode/008-clean-smoke-9f6dbae/result.json`. Per 2026-11-07 amendment, native macOS / Homebrew execution dropped as release requirement (Linux CI runner / isolated Linux evidence sufficient) and remains unobserved (not pass). |
| **AC-15** | Model scenario `not_run` status honestly maintained | **VERIFIED (awaiting human review)** | Online model evaluations and semantic tests explicitly labeled `not_run`; no synthetic or simulated passes claimed. |
| **AC-16** | Release report records commit, blockers, notes, and go/no-go | **VERIFIED (awaiting human review)** | Recorded comprehensively in this section (B5: T09, T10, T11). |
| **AC-17** | Compatibility changes classified; next `0.x` recommended | **VERIFIED (awaiting human review)** | Migration notes provided; next minor `0.x` recommended; automatic `v1.0.0` promise explicitly rejected. |
| **AC-18** | Artifact acceptance does not trigger automatic release | **VERIFIED (awaiting human review)** | Verification stops at report generation; no Git tag, release publication, or Homebrew tap push performed. |
| **AC-19** | Default commands select 7 supported-core Go packages | **VERIFIED (awaiting human review)** | Aligned in T06c; verified by `task verify` (6/6 pass), `test_taskfile.py` (10/10), and `test_supported_core_go.py` (8/8). |
| **AC-20** | Verified default-inactive experimental CLI boundary | **VERIFIED (awaiting human review)** | Implemented in T06b; verified in T10 smoke (exit 2, 0 side effects, `internal/opencodeadapter` completely absent). |
| **AC-21** | G-008-01 closed only by fresh evidence for adopted reference | **PARTIALLY OBSERVED / OPEN (GATE G-008-01)** | Hosted CI run 42355249123 on push to main verified candidate `9f6dbae` with 6/6 checks passed under existing main-push selection (`gate.json`: `trusted_revision=candidate_revision=9f6dbae`, canonical contract `f42851cc...`). Broader independent reference designation and divergent PR adoption remain open; G-008-01 is NOT closed automatically. |
| **AC-22** | Experimental tasks and tests isolated outside default path | **VERIFIED (awaiting human review)** | `Taskfile.experimental.yml` separated; experimental build tag enforced; excluded suites categorized as `excluded` / `not_run`. |
| **AC-23** | Capability and dependency mapping complete | **VERIFIED (awaiting human review)** | Frozen in T06a; verified in B5 binary dependency inspection (`internal/opencodeadapter` absent). |

### 8. Technical Go/No-Go Decision: RELEASE CANDIDATE VERIFIED; PUBLICATION BLOCKED

**Summary Verdict:**
- **Core Engineering, Clean Packaging & Linux Verification:** **READY / PASS**
  The supported-core Go implementation, 18 portable skills, safe preview/apply/backup/restore engine, Taskfile commands, default-inactive experimental boundary, and anti-weakening test suites pass all local and containerized verification checks under isolated environments.
  Clean candidate tree at commit `9f6dbaea98ce4016fdfb0580f632f9ef0401227e` passed clean `task verify` (6/6 checks at `/tmp/opencode/008-packaging-required-9f6dbae/result.json`), fresh clean Linux smoke (`/tmp/opencode/008-clean-smoke-9f6dbae/result.json`), and non-publishing snapshot packaging with checksum-verified GoReleaser v2.20.0 (`dist/agentic-sdd_0.4.0-SNAPSHOT-9f6dbae_darwin_all.tar.gz` SHA256 `23c7a77cae2d6f04dfd0b30504bf9d0fa4ada865834e3c7db5dec3b849d9611c`, universal Mach-O binary and generated Homebrew formula; deprecation warning for `brews` field retained).
- **Hosted CI Observation:** **OBSERVED / PASS (MAIN PUSH ONLY)**
  GitHub Actions run `https://github.com/nawodyaishan/agentic-sdd/actions/runs/42355249123` completed successfully in 54s on push to `main` at `9f6dbae`. Downloaded artifact `/tmp/opencode/008-ci-42355249123/gate.json` confirms canonical contract `f42851cc793fb384ce26ea3ecc428caf748a454396a2017892ab7e5029d49b17`, 6/6 required statuses passed, and dual probe digests present under existing main-push selection (`trusted_revision=candidate_revision=9f6dbae`).
- **Platform Evidence Amendment:**
  Per the 2026-11-07 documentation amendment, the repository owner explicitly directed:
  > *"i approve exact core contract / commmit changes / all changes approved to commit and push / [install GoReleaser and snapshot] do it / drop Native macOS / Homebrew evidence — AC-14 / T10 as a requirment, linux ci runner is enough"*

  Native macOS and Homebrew runtime execution evidence was dropped as a blocking release qualification requirement in favor of Linux CI runner / isolated Linux evidence sufficiency. Darwin cross-compilation and packaging integrity remain verified. Native macOS execution remains UNOBSERVED (not pass). Historical reports remain dated history and are not retroactively green. Linux local QA is NOT hosted CI observation; hosted CI observation on GitHub Actions is now observed for the main-push workflow.
- **Remaining Open Decisions & Prerogatives:**
  1. **Broader G-008-01 Adoption:** Main-push CI selection passed at `9f6dbae`, but independent trusted-reference designation and divergent PR adoption remain open. G-008-01 is not marked closed automatically.
  2. **Provisional Review Status:** Subagent implementation and verification reviews remain provisional pending Opus review verification per AGENTS.md policy.
  3. **Payload vs Documentation Distinction:** Implementation commit `9f6dbae` forms the frozen release payload; documentation updates are committed separately.
  4. **Release Tag and Publication:** Tagging (`v0.5.0` or other `0.x`), creating a GitHub Release, and publishing to the Homebrew tap remain separate human owner actions requiring explicit separate authorization. No release authority is granted or inferred.

**Next Required Action:** Awaiting human owner review of the clean packaging snapshot and hosted CI results, Opus provisional review completion, and human decision on release tagging and publication.
