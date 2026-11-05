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
