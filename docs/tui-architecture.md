# TUI Architecture and Toolchain Migration Blueprint

**Current support (Feature 013, owner-accepted):** Feature 012 delivered the
read-only nested shell/catalog spike as historical evidence below. Feature 013
B1–B13 are accepted and published. The nested TUI now includes the shared
`internal/appops` operation host, preview-first install/update, backup/restore,
activity/help, onboarding, the `agentic-dark` theme and the r6 UX refinements.
The paired archive installs `asdx` (TUI) and `asdx-core` (CLI), plus the
legacy `agentic-sdd` (CLI wrapper) and `agentic-sdd-tui` (TUI wrapper) names
for one deprecation window. A bare `asdx` opens the TUI; `asdx-core` keeps the
explicit, non-TTY and scripting contracts. `asdx-core tui` is the
identity-verified compatibility launcher. `~/.agentic-sdd/` and
`AGENTIC_SDD_*` remain unchanged, and no root `internal/tui` migration is
required.

The `v0.9.0` cask release is published and the tap cask points to `0.9.0`;
`v0.8.0` remains the prior baseline and `v0.7.0` the earlier formula→cask
migration release. The explicit CLI remains fully supported and is the
noninteractive/scripting path.

**Historical record (Feature 012):** [012 — CLI UI/UX Remediation & Hybrid TUI Experience](../specs/012-cli-tui-remediation/spec.md)
**Delivered by:** 012 batch B2 (T06–T08), preserved below as history
**013 scope:** [013 — Unified TUI Experience and Animated Branding](../specs/013-unified-tui-experience/spec.md) —
B1–B13 accepted/published; B14 release/contract closeout authorized; approval in
[013 spec.md §Approval](../specs/013-unified-tui-experience/spec.md#approval)
**Status:** experimental architecture note. It records the delivered nested TUI and a
migration that does not. It is not an approval record, a batch authorization, a
release commitment or evidence that the TUI is wired into the shipped binary.
New B3 descriptions in §§3/5 are batch-under-review, not verified fact.

Line numbers and digests below were verified against the working tree on
2026-11-10 and will drift; treat the file and symbol names as the stable part of
each reference.

---

## 1. What exists today

The spike lives in a nested module so the root module can stay dependency-free:

```text
experiments/tui/                 module agentic-sdd/experiments/tui (go 1.26)
├── go.mod                       Charm v2 pins + replace agentic-sdd => ../..
├── go.sum                       48 lines: 20 external modules (4 direct, 16 indirect)
├── config.go                    TUIConfig, defaults, LaunchAllowed gating predicate
├── theme.go                     Lip Gloss v2 styles; plain when color is off
├── catalog.go                   read-only adapter over skillsync + status vocabulary
├── catalog_test.go              gating, adapter, model, and headless-program tests
├── screens/catalog.go           Bubble Tea v2 + Bubbles list catalog browser
└── cmd/tui-spike/main.go        opt-in entry point enforcing the launch gate
```

Commands (opt-in, never part of `task test` or CI):

```sh
# Tests: skips with an explicit message when the local Go is older than 1.26.
task -t Taskfile.experimental.yml tui:test

# A human run on a real terminal, from experiments/tui:
AGENTIC_SDD_TUI=1 go run ./cmd/tui-spike --home /tmp/tui-demo-home
```

Measured facts for this checkout (not guarantees):

| Fact | Value |
|---|---|
| Catalog rows from the embedded assets | 19 skills (18 core + 1 pack) and 18 profiles with `--opencode-free-first` (14 profiles without the presets) |
| External modules pinned in `experiments/tui/go.mod` | 20 (4 direct + 16 indirect) |
| `go.sum` lines | 48 |
| Built `tui-spike` binary (`-trimpath`) | 6.6 MB, including embedded skills |
| Built root `agentic-sdd` binary (`-trimpath`) | 11.2 MB |
| `go mod verify` in the nested module | does not apply: the `agentic-sdd` directory replacement has no module-cache zip hash |

Deliberately absent from 012 B2 (historical fact, preserved): Huh v2 forms,
backup/restore screens, update checks, and any wiring into `cmd/agentic-sdd`.
The 012 spike browses; it never changes anything. 013 B1/B2 retain that
read-only boundary; 013 B3's controlled operation seam below is batch-under-review
and does not retroactively make the 012 spike write-capable.

---

## 2. Module isolation and the module-path decision

The plan sketched `module agentic-sdd-tui`. That path cannot work: Go refuses an
`internal` import from outside the tree rooted at the parent of `internal`.

```text
package agentic-sdd-tui
    a.go:3:8: use of internal package agentic-sdd/internal/skillsync not allowed
```

The nested module therefore uses a path **under** the parent module path:

```go.mod
module agentic-sdd/experiments/tui

go 1.26

require (
    agentic-sdd v0.0.0
    charm.land/bubbles/v2 v2.1.0
    charm.land/bubbletea/v2 v2.0.2
    charm.land/lipgloss/v2 v2.0.3
    github.com/charmbracelet/x/ansi v0.11.7
)

replace agentic-sdd => ../..
```

This is the same shape `golang.org/x/tools/gopls` uses inside
`golang.org/x/tools`: a nested module may extend its parent's import path, so
`agentic-sdd/experiments/tui/...` is allowed to import
`agentic-sdd/internal/skillsync`, while the root module never sees the Charm
dependencies.

Alternatives considered and rejected:

| Alternative | Why rejected |
|---|---|
| `module agentic-sdd-tui` + copy of the domain types | Duplicates domain vocabulary and drifts from `skillsync`; defeats the point of a read-only adapter. |
| Charm dependencies in the root module | Breaks the Go 1.23 floor, the `golang:1.25-alpine` gates and the zero-dependency guarantee that Feature 012 explicitly preserves. |
| `go.work` linking both modules | Changes how every root command resolves modules, including gates that set `GOWORK=off`; the nested module must stay invisible to root builds. |
| Build tags inside the root module | `go.mod` still has to declare the dependencies for the tagged files, so the root module still changes. |

Guarantees the isolation must keep, all verifiable with `git status` and one
build:

```sh
git diff --exit-code -- go.mod       # root module still go 1.23, zero requires
test ! -e go.sum                     # no root go.sum
GOWORK=off go build ./...            # root builds without the nested module
(cd experiments/tui && GOWORK=off go build ./...)
go mod tidy -C . && git diff --exit-code -- go.mod go.sum   # tidy ignores the nested module
```

`go mod tidy` at the root skips nested modules, and `internal/harnesscheck`
snapshots only the top-level `go.mod`/`go.sum`, so `task tidy-check` and
`task mod-verify` keep their current meaning.

---

## 3. Domain boundary: screens ↔ `internal/appops` ↔ `internal/skillsync` (013 controlled-adapter rule, batch under review)

The eventual production host is the nested TUI host; the 012 spike proves the
read-only subset of these rules under `experiments/tui`. 013 replaces the 012
blanket rule ("the TUI reads domain state and never performs a domain
operation") with this controlled-adapter rule: **screens never mutate
files/manifests; mutations flow through typed requests/results and
`internal/appops`; the explicit CLI remains the supported entry point; the
standalone experimental launcher stays read-only until later approved batches.**

```text
                   ┌───────────────────────────────┐
   key presses ───► │ experiments/tui/screens       │  Bubble Tea model: view state,
                   │ (nested TUI host, stays nested)│  focus, filtering, selection only
                   └──────────────┬────────────────┘
                                  │ typed requests/results (no direct file mutation)
                   ┌──────────────▼────────────────┐
                   │ internal/appops (shared       │  preview/apply/log/backup/restore/
                   │ stdlib operation seam)        │  update orchestration; client selection
                   └──────────────┬────────────────┘
                                  │ structured engine results (outcome, backup
                                  │ identity/path, partial-failure/rollback info)
                   ┌──────────────▼────────────────┐
                   │ internal/skillsync            │  authoritative installer/restore engine
                   └──────────────┬────────────────┘
                                  │ apply / restore / backup manifest
                   ┌──────────────▼────────────────┐
                   │ cmd/agentic-sdd (CLI)         │  supported mutation entry point, unchanged
                   └───────────────────────────────┘
```

The 012 read-only diagram below is preserved as history; the diagram above is
the 013 B3 design under review.

```text
                  ┌───────────────────────────────┐
  key presses ───► │ experiments/tui/screens       │  Bubble Tea model: view state,
                  │ (future: internal/tui)        │  focus, filtering, selection only
                  └──────────────┬────────────────┘
                                 │ typed values (CatalogItem, ClientStatus)
                  ┌──────────────▼────────────────┐
                  │ experiments/tui (config,      │  presentation config + adapter;
                  │ theme, catalog)               │  no paths resolved, no files written
                  └──────────────┬────────────────┘
                                 │ InspectSelection (read-only), SummarizePlan (pure)
                  ┌──────────────▼────────────────┐
                  │ internal/skillsync            │  authoritative installer/restore engine
                  └──────────────┬────────────────┘
                                 │ apply / restore / backup manifest
                  ┌──────────────▼────────────────┐
                  │ cmd/agentic-sdd (CLI)         │  the only mutation entry point
                  └───────────────────────────────┘
```

| 012 spike rule (historical) | 013 B3 controlled-adapter rule (batch under review) |
|---|---|
| `skillsync.InspectSelection(repo, home, opts)` | Screens still call read-only inspect/summarize the same way; write paths go only through immutable typed requests/results carried by `internal/appops` |
| `skillsync.SummarizePlan(transcript)` on a transcript produced elsewhere | `SummarizePlan` stays presentation-only; structured engine results (outcome, backup identity/path, partial-failure/rollback info) replace transcript-text parsing as the safety authority |
| Mapping `InventoryStatus` to badges, groups and hints | Unchanged: status is reported from the inventory, not inferred |
| Path and version strings **displayed** verbatim from the inventory | Unchanged: screens display paths; they never resolve them, probe `--repo`, or run filesystem discovery of their own |
| Its own `TUIConfig` | Unchanged: no import of the CLI's `*Env`, `*Tracer`, flag set or prompt helpers |

Forbidden in screens under both rules (013 restates, does not loosen): calling
`skillsync.Sync`/`SyncWithOptions`/`Restore` directly; reading or writing the
backup manifest, logs or lock files; `os.Create`, `os.Mkdir*`, `os.Remove`,
`os.Rename`, `os.WriteFile`, or `filepath.Walk` of a destination; invoking a
shell, importing `package main`, reusing `runEnv` recursively, or scraping CLI
prose as the safety authority. Screens remain mutation-free.

Two consequences worth stating explicitly:

- **Status is reported, not inferred.** `Template available (not installed)`,
  `Installed (matches source)`, `Installed (modified)` and
  `Unavailable / refused` come from `skillsync`'s inventory. The TUI must not
  re-implement `sameSourceTarget`, and it must not claim runtime state such as
  "enabled", "running" or "verified".
- **The transcript is an input.** `SummarizePlan` is a pure classifier, so a
  future preview pane may show counts, but the CLI remains the component that
  produces the transcript. No TUI code may call the producer.

`experiments/tui/config.go` defines `TUIConfig` (source, home, width, height,
color, interactive, OpenCode options) instead of importing the CLI's `Env`.
`Catalog` and `CatalogItem` are presentation-shaped values built from the
inventory; they carry no write capability.

### Launch precedence

The TUI is opt-in and must lose to every scripted path, in this order:

| Order | Condition | Result |
|---|---|---|
| 1 | `AGENTIC_SDD_TUI` is not exactly `1` | standard CLI |
| 2 | `AGENTIC_SDD_FORCE_INTERACTIVE=1` | standard CLI |
| 3 | stdout is not a real terminal | standard CLI |
| 4 | otherwise | TUI allowed |

Rule 2 must be checked **before** rule 3 because the CLI's `resolveEnv` treats
`AGENTIC_SDD_FORCE_INTERACTIVE=1` as a fake terminal; checking
`Env.StdoutTTY` first would let automated journeys enter an alternate screen
and hang. `screens.Model.View` sets `AltScreen` only when
`TUIConfig.Interactive` is true, so headless tests render frames without
touching the terminal. With `TUIConfig.Color` false, the theme uses zero
styles and the view strips the Bubbles components' own SGR sequences once at
the view boundary (`ansi.Strip`). ANSI codes are zero-width, so the plain
layout is identical and a headless frame is deterministic.

012 batch B2 implements the predicate (`tui.LaunchAllowed`) and the runnable
`cmd/tui-spike` that enforces it. That launcher stays a standalone read-only
entry point until later approved batches. 013 leaves `guide` on its historical
CLI path (D01): there is no `guide`→TUI wiring in any approved batch.

---

## 4. Toolchain migration blueprint (012 history; no 013 root migration required)

> 013 note (batch under review): Feature 013 explicitly excludes root module
> external dependencies, root toolchain migration, and moving Charm into root
> `internal/tui`. Nested-module isolation is retained; the nested module's
> `go 1.26` directive stays independent of the root `go 1.23` floor. The
> blueprint below is preserved as 012 evidence, not an active 013 task.

### Why the change is atomic

Every core Go recipe (`task test`, `task vet`, `task tidy-check`) and both
container suites run with `GOTOOLCHAIN=local`, so the toolchain in the container
or runner must satisfy the `go` directive in `go.mod` exactly. Raising the root
module to Go 1.26 while any gate still uses `golang:1.25-alpine` fails with
"go.mod requires go >= 1.26". The pins must move together, in one reviewed
change, verified by the same gates that protect them.

### The eight pinned locations

| # | Location | Current value | What must change |
|---|---|---|---|
| 1 | `Taskfile.yml:75` (`docker-guide`) | `golang:1.25-alpine` | New image tag; keep `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOWORK=off` and `--pull=never`. |
| 2 | `.github/workflows/ci.yml:37` | `docker pull golang@sha256:1ae0735f00daffa3aaf1363a5184c0d2dc55c78e3db4ec70241cdac97bf84b59` | Re-pin to the digest of the new tag. This digest is the trusted gate's toolchain; it must equal #3 byte for byte. |
| 3 | `tests/harness/trusted_ci.py:16` | `IMAGE = "golang@sha256:1ae0735f…84b59"` | Same digest as #2. Because this file belongs to the reviewed trusted base, changing it changes the trust anchor and needs its own review of that base revision. |
| 4 | `internal/harnesscheck/runner.go:319-320` | capability `preloaded golang:1.25-alpine image` → `docker image inspect golang:1.25-alpine` | Rename the capability string and the inspected tag together. The capability name is a contract: an unknown phrase blocks the check, so every consumer in #6 must change in the same commit. |
| 5 | `tests/guide-e2e.sh:11` and `tests/docker-e2e.sh:11` | `golang:1.25-alpine` | New tag in both `docker run --pull=never` invocations. A missing local image must keep blocking rather than pulling. |
| 6 | `harness/checks.json:53` and `tests/harness/offline-b7-checks.json:5` | `"preloaded golang:1.25-alpine image"` | Rename to match #4 exactly. This is the only place a stale name is *not* a compile error, so it is the easiest one to miss. |
| 7 | `.github/workflows/release.yml:23-26` | `actions/setup-go@v5` with `go-version-file: go.mod` and `cache-dependency-path: go.sum` | No edit needed for the floor: the workflow follows `go.mod`. Do re-check the release run, and note that `cache-dependency-path` names a root `go.sum` the root module does not currently have. |
| 8 | `README.md:66`, `README.md:95`, `harness/README.md:128` | "Go 1.23 or newer"; `golang:1.25-alpine`; the capability name in prose | Update the documented floor, the Docker prerequisite and the capability list. |

### Additional references found during the audit

Not in the plan's eight, but they pin the same facts and will contradict the
migration if left alone:

| Location | Current value | Note |
|---|---|---|
| `go.mod:3` | `go 1.23` | Source of truth for the floor; raising it is the change that forces the rest. |
| `CONTRIBUTING.md:22` | "Install Go 1.23 or newer" | Contributor setup floor. |
| `harness/README.md:147` | "Go 1.23 path validation" | Prose about the runner's path rules; re-verify the claim against the new toolchain rather than editing blindly. |

No Dockerfile, devcontainer or `.tool-versions` exists; `grep -rn "golang:"`
over the tracked tree, excluding feature records and docs, returns exactly
sites 1–6 and 8. The nested module's `go 1.26` directive is intentionally
independent of all of them.

### Target version and sequencing

Choose the newest Go release whose `golang.org/x/sys`, `charm.land/*` and
Homebrew source builds are all available; the spike is written against
`go 1.26` and Charm v2 (`bubbletea v2.0.2`, `bubbles v2.1.0`,
`lipgloss v2.0.3`).

Measured floors of the pinned set, from each module's own `go.mod`:
`bubbletea v2.0.2` declares `go 1.24.2`; `bubbles v2.1.0` and
`lipgloss v2.0.3` declare `go 1.25.0`; `x/ansi v0.11.7` declares `go 1.24.2`.
The spike declares `go 1.26` anyway, matching the migration target rather than
the dependency floor. Treat the two separately: the floor is the minimum a
production adoption could technically compile on, while the target is a
platform-support decision that a separate review must record.

1. **Decide and record the target.** A new `go` directive is a platform-support
   decision for every user building from source, not a tidy-up.
2. **Raise `go.mod` and every pin in one commit.** Sites 1–6 above; docs (7–8
   and the extra references) in the same change or a linked one.
3. **Re-pull and record the new image digest.** `docker pull golang:<tag>`,
   then read the digest with `docker image inspect --format '{{index .RepoDigests 0}}'`
   and put the identical value in `ci.yml` and `trusted_ci.py`.
4. **Re-run the gate set** on the merged change:

   ```sh
   task tidy-check
   task mod-verify
   task vet
   task test
   task build-darwin
   git diff --check
   sh tests/docker-e2e.sh
   sh tests/guide-e2e.sh
   task verify-trusted SDD_TRUSTED=<reviewed-base>   # authorized environments only
   ```

5. **Only then** start Phase 3 (production TUI integration) or root-level
   dependency adoption, each with its own authorization. (Superseded for 013:
   see §5 — 013 keeps the nested host, leaves `guide` unchanged, and requires
   no root migration.)

Rollback is the same commit in reverse: restore `go.mod`, the image tags, the
digest in both files, the capability name and the documented floor. Because the
digest appears twice (#2, #3), a half-reverted state fails closed at the trusted
gate rather than quietly testing a different toolchain.

### Risk notes

| Risk | Effect | Mitigation |
|---|---|---|
| Digest drift between `ci.yml` and `trusted_ci.py` | The trusted gate pulls one toolchain and believes it verified another | Update both from one `docker image inspect` output; the gate compares the reviewed base, so a mismatch fails loudly. |
| Capability rename missed in a JSON contract | `prerequisite: unknown capability` blocks the check, or a stale name silently matches nothing | Change `runner.go` and both JSON contracts in one commit; `grep -rn "preloaded golang:"` finds all three. |
| Docs updated before gates | Readers install a toolchain the repo cannot build with | Land docs in the same change as the pins. |
| Homebrew users on older Go | No effect: the cask installs prebuilt universal binaries and never builds from source | Keep the release job pinned to the reviewed Go patch; source-checkout users still follow the README's Go floor. |
| Experimental code leaking into gates | Core CI starts requiring network or Charm modules | Keep `experiments/tui` nested, keep `tui:test` in `Taskfile.experimental.yml`, and never add its packages to the core package lists. |

---

## 5. Phased path (012 history + 013 controlled operation host, batch under review)

012-era phases preserved as history; 013 rows are batch-under-review, not
verified fact:

| Phase | Content | Status |
|---|---|---|
| B1 (Feature 012) | Stdlib fixes: definition formatting, grouping, copy, navigation, terminal-only messages | separate batch, separate state |
| B2 (Feature 012) | Nested module, read-only catalog spike, this blueprint, `tui:test` | experimental only |
| 012 Phase 2 | Toolchain migration per §4, then retain the spike as evidence | superseded for 013: no root migration is required; nested isolation retained |
| 012 Phase 3 | Promote `experiments/tui` to `internal/tui`, wire `asdx-core guide` behind the §3 gate, add Huh v2 forms and backup/log viewers | superseded: no promotion to root `internal/tui`, no `guide`→TUI wiring (`guide` unchanged per D01) |
| B1 (Feature 013) | Persistent responsive shell, accessible animated branding; Explore stays usable | accepted by owner |
| B2 (Feature 013) | Shared metadata/gate coverage, categorized rich catalog, local dashboard, agent routing details, high-contrast shell | completed — accepted by owner 2026-11-10 |
| B3 (Feature 013) | Controlled operation host: `internal/appops` seam, typed requests/results, structured engine results, explicit client selection | accepted by owner; published |
| B4 (Feature 013) | Preview-first install journey with safe async execution | accepted by owner; published |
| B5 (Feature 013) | Backup/restore, activity/help, automatic main-menu update check, browser guide, onboarding | accepted by owner; published |
| B6 (Feature 013) | Paired package/marker proof, Homebrew-default launcher, nested companion (`agentic-sdd-tui`) distribution, cask release and final acceptance | T15/T14/T21 published; T16 owner-confirmed; `v0.7.0` cask/tap published |
| B7 (Feature 013) | Semantic `agentic-dark` theme, chrome/list/glyph refinement, modals and truthful feedback | accepted by owner; published |
| B8–B13 (Feature 013) | r6 impact/no-op/confirmation, purpose-first Home, single dark theme, Library/Recovery/Help, Quick start and responsive evidence | accepted by owner; published |

013 contracts this path must preserve: non-TTY stdout stays byte-identical,
numeric menu IDs stay stable for scripted journeys, the explicit CLI remains
usable without the companion, and every mutation flows through the controlled
adapter (`internal/appops` + engine results) with screens remaining
mutation-free.

## 6. Non-goals and honest limitations

- This document does not authorize raising `go.mod`, changing container images,
  pulling new digests, publishing a release or running any live operation.
- The Homebrew cask is the supported paired-binary distribution. Generic/source
  builds retain the historical preview default and require the explicit `tui`
  command or a paired companion.
- It does not promise the 20-module dependency footprint; it is a measurement of
  one pin set, not a budget.
- Platform observations were owner-confirmed for B6/T16 on 2026-11-11, but no
  macOS/SSH/tmux/screen-reader transcripts are stored here. Terminal-remapped
  ANSI palettes remain outside universal contrast guarantees; ANSI-16 is
  best-effort, while TrueColor/ANSI-256 reference pairs meet the documented
  floors.
- `v0.7.0` is published, but the r6 UX changes have no r6 release tag. No new
  release/tag/tap operation is authorized by this note; publishing one requires
  separate authorization.
