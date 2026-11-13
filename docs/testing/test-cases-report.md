# Test case inventory — scope, type and feature coverage

**Snapshot:** repository working tree at commit `f559872` (2026-11-14), clean when the inventory was taken.
**Status:** an inventory, not a test run. It lists declared test cases; it does not prove a pass, create approval, change batch state or certify any runtime, model, container or native behavior.

## What this report covers

| Layer | Files | Declared cases | Notes |
|---|---:|---:|---|
| Go tests (two modules) | 148 | 1229 | `*_test.go`, including one benchmark and no fuzz targets |
| Python tests | 26 | 222 | `unittest` classes under `tests/harness` and `tests/tui` |
| Shell/container drivers | 17 | 0 | `tests/*.sh`, `tests/harness/*.sh`, `tests/qa-container/entrypoint.sh` |
| **Total declared test functions** | **174** | **1451** | five are child-process helper entry points, not independent scenarios |

`659` of the Go cases run in the default core gate; `570` are in the opt-in TUI module or behind the experimental build tag. All `222` Python cases are selected explicitly, gated by an environment variable, or driven by an opt-in PTY task — none run from `task test`.

Manual test plans, human-cycle run sheets, acceptance/evidence records and scenario JSON that no test reads are **not** counted here. This report covers executable Go/Python test files and the shell/container drivers.

The count is top-level Go `func Test…`/`func Benchmark…` and Python `def test_…` declarations. Table-driven Go tests contain **479** `t.Run` call sites, so the number of executed subtests and assertions is higher. Counts are stable for this commit, not a coverage percentage. Counts were extracted with `grep` over Go test declarations and Python AST parsing of `tests/**/test_*.py`; Appendix A and Appendix B are the generated name index. The Go total was cross-checked with `go test -list '.*'` (listing names only, not executing test bodies): 659 core + 518 nested TUI + 44 adapter + 8 experimental-tag cases = 1,229.

### Key observations

- The default core gate holds **659 of 1,451** declared cases (45%). The other 792 are opt-in, experimental, explicit or gated; the nested TUI module alone has 518.
- Features **013** and **015** account for **699 cases (48%)**, and 493 of those are in the TUI.
- **Safety/negative** files hold 137 cases; many integration files also assert refusal, rollback and stale-state paths, so the safety surface is broader than that one row.
- The Python harness is **222 cases (15%)** and never runs from `task test`.
- Features **002** and **007** have no primary test file; **005** is preserved but fenced, and **006**'s native/dispatch/pilot work remains deferred.

## How to read the structure

- **Scope** is the component or package the file exercises, not the feature that introduced it.
- **Type** is the dominant kind of check in the file. Mixed files use their strongest axis; for example a safety file may also assert exact bytes.
- **Feature** is the primary [feature record](../../specs/) supported by the file, followed by secondary features in parentheses. Feature attribution is inferred from in-file batch comments, spec references and timestamps; it is not an approval claim. A file is counted once, under its primary feature.
- **Cases** is the number of declared test functions in that file. A `0` means the file is shared fixture/support code with no assertions.
- **Selection** says whether the file is in the default core gate, opt-in, gated, or historical/fenced. The labels are: `Core` (default Go core gate), `Opt-in` (nested TUI module or experimental build tag), `Explicit` (Python suite selected by `-p`), `Gated` (environment/container opt-in), `Opt-in PTY` (PTY task), `Focused` (`harness/checks.json` focused check) and `Historical` (fenced Feature 005/006 evidence).

## Execution entry points

| Entry point | What it selects | Default? |
|---|---|---|
| `task test` / `task vet` | The 11 supported-core Go packages in `tests/supported-core-go.sh` and `harness/checks.json` | Yes |
| `task test-core` / `task vet-core` | Same selection through the checked shell selector | Alias |
| `task verify` | `harness/checks.json` required checks (`mod-verify`, `tidy-check`, `vet`, `test`, `build-darwin`, `diff-check`) | Explicit |
| `task docker-e2e` | Installer preview/apply/backup/restore in `golang:1.25-alpine`, network none | Focused check |
| `task docker-guide-e2e` | Guided CLI journeys J1–J10 in an offline container | Opt-in |
| `python3 -B -m unittest discover -s tests/harness -p test_*.py` | Individual Python harness suites (one `-p` at a time) | Explicit only |
| `task -t Taskfile.experimental.yml tui:test` | Full nested TUI Go module (Go 1.26+; a skip is not a pass) | Opt-in |
| `task -t Taskfile.experimental.yml tui:pty-test` | `test_terminal_lifecycle.py` and `test_launcher_lifecycle.py` against a built pair | Opt-in |
| `task -t Taskfile.experimental.yml tui:fleet-pty` / `tui:mcp-pty` | Fleet and MCP PTY suites against a built TUI binary | Opt-in |
| Lefthook `pre-commit` | `task test`, `task vet`, gofmt and whitespace | Local hook |
| GitHub Actions `ci.yml` | Reviewed base's trusted gate run against candidate bytes as data | Hosted CI |
| GitHub Actions `release.yml` | `task verify` before a GoReleaser release | Tag push only |

`harness/checks.json` contains check-level entries, not individual test cases. The TUI `tui:test` task exits successfully after printing a skip when Go is older than 1.26; that is explicitly not a pass. The same rule applies to every environment-gated Python suite.

## Selection lanes

| Selection lane | Files | Declared cases | How it runs |
|---|---:|---:|---|
| Default core Go gate | 88 | 659 | `task test`, `task vet`, `harness/checks.json` |
| Opt-in Go (TUI module, experimental tag, adapter) | 60 | 570 | explicit nested-module or build-tag command |
| Explicit Python suites | 16 | 146 | `python3 -B -m unittest discover -s tests/harness -p …` |
| Environment-gated Python | 5 | 37 | runs only with the named container/opt-in variable |
| Opt-in Python PTY | 5 | 39 | `task -t Taskfile.experimental.yml tui:*` |
| Shell/container drivers | 17 | 0 (drivers) | task targets or direct `sh` |

The core list grew beyond the seven packages in Feature 008's original approval through later approved changes: `internal/appops` and `internal/buildinfo` (Feature 013 B2/T04 and B14/T41), `internal/fleet` (Feature 014 B5/T10) and `internal/execution` (Feature 016 B1/T07). `tests/harness/README.md` records that history; it is not a silent gate change.

## Coverage by feature

| # | Feature | Cases (primary) | Files/drivers (primary) | Main scopes |
|---|---|---:|---:|---|
| [001](../../specs/001-cli-quality/spec.md) | CLI quality and usability | 19 | 3 | CLI, Shell |
| [002](../../specs/002-homebrew-publishing/spec.md) | Homebrew package publishing | 0 | 0 | _no primary test file_ |
| [003](../../specs/003-restore-backups/spec.md) | Restore skill backups | 18 | 3 | Installer core |
| [004](../../specs/004-vnext-harness/spec.md) | Agentic SDD vNext: verifiable single-agent lifecycle | 98 | 16 | Check CLI, Check harness, Harness, Shell |
| [005](../../specs/005-vnext-completion/spec.md) | vNext implementation completion and human-input helpers | 44 | 13 | OpenCode adapter, Shell |
| [006](../../specs/006-independent-review/spec.md) | On-demand independent review for OpenCode | 37 | 8 | CLI, Harness, Shell |
| [007](../../specs/007-taskfile-migration/spec.md) | Replace Makefile with Taskfile | 0 | 0 | _no primary test file_ |
| [008](../../specs/008-core-release-quality/spec.md) | Core product quality, maintainability and release readiness | 97 | 15 | CLI, Harness, Installer core, Shell |
| [009](../../specs/009-opencode-orchestration/spec.md) | Optional OpenCode orchestration, delegation, routing and context | 8 | 2 | Installer core |
| [010](../../specs/010-guided-cli-ux/spec.md) | Guided CLI experience, accessible presentation and flow tracing | 24 | 8 | CLI, Installer core, Shell |
| [011](../../specs/011-cli-brand-styling/spec.md) | Branded CLI styling | 66 | 7 | App operations, CLI, Installer core |
| [012](../../specs/012-cli-tui-remediation/spec.md) | CLI UI/UX remediation and hybrid TUI | 38 | 6 | CLI |
| [013](../../specs/013-unified-tui-experience/spec.md) | Unified TUI experience and animated branding | 428 | 43 | App operations, CLI, Installer core, TUI … |
| [014](../../specs/014-agent-fleet-model-overrides/spec.md) | Git-managed OpenCode fleet profiles and model overrides | 63 | 18 | CLI, Fleet engine, Installer core |
| [015](../../specs/015-fleet-manager-tui/spec.md) | Fleet Manager TUI and MCP tools | 271 | 29 | App operations, CLI, Fleet engine, Installer core … |
| [016](../../specs/016-agent-execution-context-evidence-recovery/spec.md) | Agent execution context, evidence and recovery | 112 | 8 | CLI, Execution runs |
| [017](../../specs/017-agent-context-upgrade/spec.md) | Execution context integrity hardening | 18 | 4 | CLI, Execution runs |
| [018](../../specs/018-research-evidence-handoff/spec.md) | Research routing, search MCPs and evidence handoffs | 22 | 1 | Harness |
| [019](../../specs/019-cli-tui-entrypoint/spec.md) | Short commands, safe legacy migration and TUI Guide | 88 | 7 | Build metadata, CLI, TUI, TUI E2E |

File counts include shell drivers, which have no declared case functions; case counts are Go and Python only. Two features have no test file that takes them as primary:

- **002 (Homebrew publishing)** is covered indirectly by `internal/buildinfo/release_config_test.go` (GoReleaser pair/universal-name shape), `tests/harness/test_taskfile.py` (release build profile) and `.github/workflows/release.yml`; the cask install itself is a live release operation with no automated test here.
- **007 (Taskfile migration)** is covered by `tests/harness/test_taskfile.py` and the trusted-CI suites, which take 008 as their primary feature. No test asserts historical Makefile behavior, by design.

Feature **005** owns the 44 `internal/opencodeadapter` cases and six historical shell drivers, all outside the default gate under the 008 test fence; presence here is not evidence that any of them ran or passed. Feature **006** has 37 declared cases, but its native/dispatch/pilot work remains deferred; most of its Python files are static or scratch-fixture checks.

## Coverage by type

| Type | Files/drivers | Declared cases | Typical use |
|---|---:|---:|---|
| Unit | 36 | 357 | one package or pure helper; temp dirs and injected seams only |
| Integration | 47 | 400 | several layers, real Git/filesystem, synthetic child processes |
| Contract/golden | 28 | 153 | pins a CLI, JSON, protocol, manifest or exact output bytes |
| Acceptance/UX | 25 | 228 | user journey, presentation, responsive layout or terminal behavior |
| Safety/negative | 19 | 137 | refusal, rollback, stale/conflict, traversal, secret and fail-closed paths |
| Lifecycle/PTY | 11 | 102 | signal, process-group, terminal restore or parent/child lifecycle |
| Static/fixture | 7 | 68 | reads Markdown/JSON/fixtures; no runtime model or network |
| E2E/container | 17 | 6 | real container, network-none or PTY harness |

Shell drivers appear as files with zero cases: they execute scenarios through a container or shell, not `unittest`-style cases. “Test support” is excluded from the case total: `internal/harnessevidence/receipt_fixture_test.go` builds a disposable repository/receipt fixture for the other evidence suites and contains no test function.

## Scope summary

| Scope | Files/drivers | Declared cases | Dominant types | Selection |
|---|---:|---:|---|---|
| CLI | 31 | 227 | Integration, Contract/golden, Acceptance/UX | Mixed |
| Check CLI | 1 | 2 | Contract/golden | Core |
| Installer core | 17 | 124 | Contract/golden, Integration, Unit | Core |
| App operations | 7 | 76 | Unit, Integration, Contract/golden | Core |
| Build metadata | 2 | 13 | Unit, Contract/golden | Core |
| Fleet engine | 14 | 56 | Unit, Safety/negative, Integration | Core |
| Execution runs | 10 | 123 | Integration, Unit, Safety/negative | Core |
| Check runner and evidence | 7 | 46 | Contract/golden, Integration, Safety/negative | Core |
| Experimental OpenCode adapter | 7 | 44 | Integration, Contract/golden, Safety/negative | Opt-in |
| TUI | 52 | 518 | Acceptance/UX, Unit, Integration | Opt-in |
| Python harness | 21 | 183 | Unit, Static/fixture, Integration | Mixed |
| Python TUI acceptance | 5 | 39 | Lifecycle/PTY | Opt-in PTY |
| Shell/container drivers | 17 | 0 | E2E/container, Integration | Mixed |

## Coverage by scope

Each table is one package or harness. The **Feature** column lists the primary feature first; the **Selection** column is the strongest gate that can run the file, not proof that it ran.

### CLI — `cmd/agentic-sdd`

_31 files, 227 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `b3_compat_test.go` | Non-TTY apply/restore/backups bytes | Contract/golden | 013 (+010) | 6 | Exact piped output and exit codes for write commands that later TUI work must preserve. | Core |
| `branding_summary_test.go` | Preview/apply summary rows | Acceptance/UX | 011 | 2 | Branded summary headings, wide/narrow counts, and unchanged backup id/log/undo values. | Core |
| `branding_test.go` | Brand art, animation, styles | Acceptance/UX | 011 | 10 | Static intro, no-color/narrow fallbacks, bounded animation, SGR-only styling, prose wrapping. | Core |
| `env_test.go` | Terminal/env resolution | Unit | 010 (+011) | 4 | Terminal detection, env precedence, animate opt-in, and /dev/null stdin restore behavior. | Core |
| `execution_cli_test.go` | run/task/review commands | Integration | 016 | 3 | Init/brief/status/record/clean lifecycle, direct scope, refusals, help and JSON format. | Core |
| `execution_integrity_test.go` | Execution integrity regressions | Safety/negative | 017 | 4 | Foreign checkpoint refusal, deleted/legacy briefs, and explicit repeat/force behavior. | Core |
| `fleet_apply_test.go` | fleet plan/apply | Integration | 014 | 1 | Create-apply, stale-plan refusal, and idempotent no-op apply through a temporary home. | Core |
| `fleet_doctor_test.go` | fleet doctor | Contract/golden | 014 (+015) | 4 | Offline default, validation/usage, unavailable labeling, and help surface. | Core |
| `fleet_lock_test.go` | fleet lock contention | Safety/negative | 014 | 1 | Held-lock refusal and no-write behavior during fleet apply. | Core |
| `fleet_status_golden_test.go` | fleet status bytes | Contract/golden | 015 (+014) | 4 | Byte-compatible stdout and exit codes before/after the BuildStatus refactor. | Core |
| `fleet_test.go` | fleet CLI surface | Contract/golden | 014 (+015) | 4 | Command dispatch, help legend, exit-code table, drift labels and warnings. | Core |
| `guide_branding_test.go` | Guided branding | Acceptance/UX | 011 (+010) | 5 | Single branded intro, terminal-only update choice, animation gate, narrow/wide styling. | Core |
| `guide_catalog_alias_test.go` | Teaching metadata seam | Unit | 012 | 3 | Historical private teaching symbols and shared appops metadata stay API-compatible. | Core |
| `guide_catalog_test.go` | Catalog drift | Contract/golden | 012 | 7 | Curated registry matches shipped assets, complete cards, human examples, stable order, no model claims. | Core |
| `guide_explorer_test.go` | Guide explorer | Acceptance/UX | 012 (+014) | 10 | Detail statuses/navigation, profiles/presets, missing packs, safe fallback, bounded private trace. | Core |
| `guide_plain_compatibility_test.go` | Plain guide branch | Contract/golden | 012 (+011) | 1 | Legacy script-readable plain branch remains byte-identical after branding. | Core |
| `guide_presentation_test.go` | Install review/presentation | Acceptance/UX | 012 | 15 | Review, back/confirm/decline, no-op, failure guidance, log safety, quoting and definition layout. | Core |
| `guide_test.go` | Guide journeys | Integration | 010 (+012) | 4 | Flags, install/decline/repeat, log/restore reader, EOF, and README example prompts. | Core |
| `guide_updates_test.go` | Update check in guide | Integration | 011 | 4 | Only explicit terminal choice checks; one call per choice, state rendering, trace privacy. | Core |
| `guide_ux_test.go` | Menu and restore impact | Acceptance/UX | 012 | 2 | Menu return paths and restore-impact disclosure before confirmation. | Core |
| `logs_test.go` | Apply/restore logs | Integration | 010 (+003) | 6 | Exact log/stdout transcripts, partial failure transcripts, and structured backup IDs in trace. | Core |
| `main_test.go` | Top-level commands | Integration | 001 (+002, 003, 009, 010, 019) | 19 | Help/usage, preview/apply, embedded source, version, restore forms, logs, byte identity, topic help. | Core |
| `mcp_test.go` | mcp read commands | Contract/golden | 015 | 15 | Dispatch, status/doctor/preview text+JSON, engine errors, deterministic no-TTY output, zero writes. | Core |
| `mcp_usage_test.go` | mcp usage/error rendering | Contract/golden | 015 | 3 | Usage-vs-runtime error codes, ambiguous target and missing explicit path messages. | Core |
| `mcp_write_test.go` | mcp write commands | Safety/negative | 015 | 26 | Preview/apply/restore grammar, auth/expect-config contract, stale/conflict refusal, no-op, JSON, no secrets. | Core |
| `opencode_disabled_test.go` | Default core build boundary | Safety/negative | 008 (+009) | 7 | Experimental OpenCode files/imports are unreachable; denial has no side effects; help stays honest. | Core |
| `opencode_test.go` | Experimental OpenCode commands | Integration | 006 (+009) | 8 | Offline preview/apply, admission-blocked session, pending reviews, evidence validate/export. | Experimental tag |
| `trace_test.go` | Flow trace | Contract/golden | 010 | 6 | Trace command/flag names, JSON Lines, guide flow, normalization, disabled no-op, failure isolation. | Core |
| `tui_test.go` | TUI dispatch/launcher | Integration | 019 (+013) | 21 | Dispatch matrix, sibling handshake, protocol query, bounded output, signal relay, exit propagation, legacy wrapper. | Core |
| `ui_test.go` | Terminal summaries/help/errors | Acceptance/UX | 010 | 3 | Terminal counts and color, help topics, and error hints only on terminals. | Core |
| `updates_test.go` | Update checker wrapper | Integration | 011 (+013) | 19 | Request contract, comparison/states, malicious metadata, limits, redirect refusal, shared-checker parity. | Core |

### Check CLI — `cmd/sdd-check`

_1 files, 2 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `main_test.go` | Usage and blocked result | Contract/golden | 004 | 2 | Usage errors exit 2, help exits 0, missing prerequisite yields structured blocked JSON. | Core |

### Installer core — `internal/skillsync`

_17 files, 124 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `identity_test.go` | Source digest | Contract/golden | 013 | 5 | Digest stable across preview/apply/no-op, ignores destination, tracks core/pack/preset/manifest bytes. | Core |
| `inventory_test.go` | Selection inventory | Contract/golden | 011 | 10 | 18/19 skills, 14+4 profiles, origins/statuses, read-only, unsafe parents, pack error taxonomy. | Core |
| `manifest_test.go` | Backup manifest | Contract/golden | 003 | 2 | Format-2 identity/tool/home/entries/digests on apply; legacy vs format-2 detection. | Core |
| `mcp_compat_test.go` | OpenCode/MCP isolation | Safety/negative | 015 | 1 | opencode.json(c) stays byte-identical and no MCP state appears during --with-opencode preview/apply. | Core |
| `mcp_config_test.go` | MCP plan builder | Unit | 015 | 20 | add/change/leave/conflict ops, unrelated preservation, unsafe targets, JSONC refusal, auth recipes, no secret echo. | Core |
| `mcp_test.go` | MCP status/preview | Contract/golden | 015 | 14 | Default servers, v1/v2/JSONC/ambiguity/invalid/symlink/auth states, read-only guarantees. | Core |
| `mcp_transaction_test.go` | MCP apply/restore | Integration | 015 | 20 | Exact bytes, 0700/0600 evidence, intent phases, stale digest, fail-closed probe, rollback, lock, restore. | Core |
| `opencode_adapters_test.go` | Adapter exclusion | Contract/golden | 014 | 1 | adapters/ holds exactly two files; --with-opencode installs 14 profiles without them. | Core |
| `opencode_pack_test.go` | OpenCode pack assets | Contract/golden | 009 | 1 | Embedded==on-disk, v2 frontmatter, deny-first rules, model pins, relative links. | Core |
| `opencode_sync_test.go` | --with-opencode install | Integration | 009 | 7 | Default isolation, additive profiles, preservation, format-3 digests, idempotence, symlink rollback, restore. | Core |
| `package_test.go` | Distribution contract | Contract/golden | 008 (+018, 006) | 4 | Exact 18 skill IDs, packaged link resolution, byte-identical install/no-op/restore in an isolated home. | Core |
| `permissions_test.go` | Permission semantics | Safety/negative | 008 | 4 | Mode-only diff is a no-op without backup; requested modes under umask; restore normalizes modes. | Core |
| `restore_test.go` | Restore | Integration | 003 (+013) | 9 | Preview no-write; format-2/1 round trips; no-op; rollback; traversal, symlink and digest rejections. | Core |
| `result_test.go` | OperationResult API | Contract/golden | 013 | 13 | Typed outcomes, preview/apply/no-op parity, failure before/after backup, rollback reporting. | Core |
| `selection_test.go` | Client subset selection | Integration | 013 | 5 | Normalization, rejection before writes, subset apply/restore round trip, inventory parity. | Core |
| `summary_test.go` | Plan summary parser | Contract/golden | 010 (+011) | 1 | Every real preview/apply/restore line classified; counts/client order; unknown lines reported. | Core |
| `sync_test.go` | Sync plan/apply/rollback | Integration | 003 (+013) | 7 | Preview no-write; backup+replace across four clients; symlink/non-directory refusal; remove+install rollback. | Core |

### App operations — `internal/appops`

_7 files, 76 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `fleet_test.go` | Fleet seam | Integration | 015 (+014) | 7 | Status counts, source digest, create/update/remove/conflict ops, token revalidation, backups, offline doctor. | Core |
| `mcp_test.go` | MCP read seam | Unit | 015 | 21 | Request absolutization/immutability, doctor order/vocabulary, auth never passes, no exec/leak. | Core |
| `mcp_write_test.go` | MCP write seam | Unit | 015 | 12 | Request validation/immutability, preview no-write, apply/restore round trip, deterministic tokens. | Core |
| `ops_test.go` | Install/restore seam | Integration | 013 | 9 | Request validation/immutability, token stability, ErrNeedsPreview on drift, subset round trip. | Core |
| `routing_test.go` | Agent routing table | Contract/golden | 013 | 4 | Curated routing matches routing.md and preset frontmatter; manual-failover notes; defensive copy. | Core |
| `teaching_test.go` | Teaching metadata | Unit | 013 (+011) | 7 | Curated cards, neutral unknown fallback, concept order, defensive slices, caveat wording. | Core |
| `update_test.go` | Release check | Unit | 011 (+013) | 16 | Fixed endpoint/headers, numeric compare, invalid tags, body cap/close, redirect refusal, cancellation. | Core |

### Build metadata — `internal/buildinfo`

_2 files, 13 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `buildinfo_test.go` | Identity/protocol | Unit | 019 (+014) | 11 | Dev defaults fail validation, distribution/capability matrices, protocol JSON, companion verification. | Core |
| `release_config_test.go` | GoReleaser/cask shape | Contract/golden | 019 (+013, 002) | 2 | Four builds and universal names, one paired archive, cask and quarantine hook, no brews/Formula. | Core |

### Fleet engine — `internal/fleet`

_14 files, 56 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `apply_test.go` | Apply plan | Integration | 014 | 1 | Model-only render, format-4 sidecar, no-op/stale/conflict/adopt/remove, symlink/traversal refusal, rollback. | Core |
| `diff_test.go` | Preview/diff/plan files | Unit | 014 | 6 | Deterministic encode, preview ops/refusals, CRLF/Unicode/bounded diff, removal-first order, stale-plan refusal. | Core |
| `doctor_test.go` | Offline doctor | Unit | 014 | 14 | Offline not_checked, label ladder, auth marker, fail-closed sanitized errors, no network. | Core |
| `integrity_test.go` | Reseal/digest/lock/restore | Safety/negative | 014 | 6 | Resealed plan cannot overwrite drift; digest/path/symlink checks; one lock winner; restore provenance. | Core |
| `negative_matrix_test.go` | Negative matrix rows | Safety/negative | 014 (+009) | 6 | Failed-state sidecar, alias-to-alias, 009-vs-fleet ownership, adapter non-writing, no default-agent write. | Core |
| `parse_test.go` | Strict config parser | Unit | 014 | 1 | Defaults, size cap, UTF-8, duplicate keys, unknown/policy fields, traversal, model grammar, no secret echo. | Core |
| `plan_test.go` | Plan identity + lock | Unit | 014 | 3 | Canonical field order, path-invariant/tamper plan ID, 0600 save/load, stale revalidation, lock recovery. | Core |
| `render_test.go` | Model render/permissions | Contract/golden | 014 (+009) | 2 | Byte-identical render, variant/CRLF, block-scalar fail-closed, all 14 real profiles, adapter deny policy. | Core |
| `resolve_test.go` | Source resolution | Unit | 014 | 2 | 18-name golden map, adapters/ directory, alias/cycle/symlink refusal, embedded==on-disk pack. | Core |
| `restore_preview_test.go` | Restore preview | Contract/golden | 014 | 2 | Preview equals applied restore; legacy/failed/corrupt/incomplete/symlink refusal; zero writes. | Core |
| `restore_test.go` | Restore apply | Integration | 014 | 1 | Replaced bytes, dropped created files, unrelated kept, provenance, backup-id validation, list coexistence. | Core |
| `scale_015_test.go` | Scale evidence 18/100 | Integration | 015 (+014) | 4 | Correctness and zero writes for status/preview/plan/diff at 18 and 100; logs medians, no threshold. | Core |
| `scale_test.go` | Scale evidence 14 | Integration | 014 | 2 | Doctor and BuildPlan pipelines record median latency/alloc; asserts correctness only. | Core |
| `status_test.go` | Status projection | Unit | 014 (+015) | 6 | Counts, managed/pending/drifted, unmanaged/legacy/unsafe, disabled, determinism, invalid config. | Core |

### Execution runs — `internal/execution`

_10 files, 123 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `briefintegrity_test.go` | Brief provenance/freshness | Safety/negative | 017 | 4 | Path/digest registration, missing/changed/unregistered/symlink/over-budget briefs, status stale mapping. | Core |
| `checkpoint_identity_test.go` | Checkpoint ownership | Safety/negative | 017 | 1 | Foreign run ID conflicts in load/status/check/record/review without writes; matching/missing stay prepared. | Core |
| `checkpoint_test.go` | Checkpoint/status machine | Unit | 016 | 14 | State precedence, normalized batch digest, progress vs drift, contradictory evidence, no approval wording. | Core |
| `checks_test.go` | RunCheck subprocess | Integration | 016 | 19 | Pass/fail/signal/timeout/output cap, summary, argv secrecy, serialization, killed writer stays partial. | Core |
| `record_test.go` | Claim ingestion | Contract/golden | 016 | 8 | Fixture parse, task resolution, invalid shapes, passed stays awaiting verification, no artifact on reject. | Core |
| `repository_test.go` | Git identity/scope | Integration | 016 | 12 | HEAD/branch/worktree digest, exclusions, feature/direct scope, traversal/symlink guards, stable digest. | Core |
| `reviewpackage_test.go` | Review package | Integration | 016 | 18 | Committed/staged/unstaged/untracked/rename/delete/mode/binary, caps, sensitive/conflict/head mismatch, no mutation. | Core |
| `taskbrief_test.go` | Brief extraction | Unit | 016 (+017) | 19 | Heading/checkbox forms, T1 vs T10, fences, CRLF, unresolved/duplicate refs, digest, force. | Core |
| `taskbrief_transaction_test.go` | Run-backed brief write | Safety/negative | 017 | 9 | Live lock conflict, contradictory identity, registration, history kept, legacy repair, drift, lock release. | Core |
| `workspace_test.go` | Run workspace/lock/clean | Integration | 016 | 19 | Run-ID grammar, 0700/0600, self-ignoring root, clean git status, symlink refusal, lock reclaim, clean preview/apply. | Core |

### Check runner and evidence — `internal/harnesscheck`, `internal/harnessevidence`

_7 files, 46 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `contract_test.go` | Checks contract | Contract/golden | 004 | 5 | Invalid mutations rejected, required count/selection rules, tidy snapshot, docker --pull=never. | Core |
| `process_linux_test.go` | Process-group control | Lifecycle/PTY | 004 | 3 | Cancel stops owned group not unrelated; leader cannot pass with live child; bounded wait reports unconfirmed. | Core |
| `runner_test.go` | Runner/evidence | Integration | 004 | 14 | Pass+private logs, fail stops later steps, blocked prereq, truncation, path refusals, literal argv. | Core |
| `export_test.go` | Receipt export | Safety/negative | 004 | 5 | Canary redaction before write, private mode, no overwrite, in-checkout/dotdot/symlink refusal, credential scrub. | Core |
| `receipt_fixture_test.go` | Shared fixture | Test support | 004 | 0 | Builds disposable repo, receipt, result and identity for other suites; contains no assertions. | Core |
| `snapshot_test.go` | Git snapshot identity | Contract/golden | 004 | 5 | Python-derived golden digests, staged/unstaged/untracked/deleted/mode detection, unsafe scope/symlink/fifo refusal. | Core |
| `validate_test.go` | Receipt validation | Contract/golden | 004 | 14 | Consistency only, stale/unknown identity, invented observations, evidence boundaries, Ready rules, no promotion. | Core |

### Experimental OpenCode adapter — `internal/opencodeadapter`

_7 files, 44 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `acp_test.go` | ACP framing | Unit | 005 | 5 | Oversize frame, mid-message EOF, bad JSON/version, id+params round trip, embedded newline rejected. | Opt-in |
| `adapter_test.go` | Candidate/preview/apply | Integration | 005 | 10 | Inactive namespaced candidate, preflight, apply/no-op/rollback, comments+mode kept, symlink/secret refusal. | Opt-in |
| `files_unix_test.go` | File guards | Safety/negative | 005 | 1 | FIFO read and symlinked config destination refused. | Opt-in |
| `pending_test.go` | Admission/pending | Safety/negative | 005 | 5 | Admission always denied, unknown rationale normalized, bounded option parse, no replay, private dir. | Opt-in |
| `receipt_test.go` | Receipt projection | Contract/golden | 005 (+006) | 8 | Composes checks+native outcome then validates; refuses failed-over-completed, identity drift, invalid context. | Opt-in |
| `runner_test.go` | Owned process/env | Lifecycle/PTY | 005 | 3 | Terminate reaps only owned group, unrelated survives, minimalEnv leaks no credentials. | Opt-in |
| `session_test.go` | ACP session | Integration | 005 | 12 | Usage-reported happy path, provider error not success, elicitation/permission block, violations, deadline. | Opt-in |

### TUI — `experiments/tui`, separate Go module

_52 files, 518 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `activity_test.go` | Activity/Help model | Unit | 013 | 11 | Immediate local render, empty/error states, bounded log load, navigation, update wiring, plain mode. | Opt-in |
| `app_test.go` | Root App shell | Acceptance/UX | 013 | 22 | Five-route navigation/focus, contextual footer, breakpoints, brand tick, exact cell bounds, ASCII escaping. | Opt-in |
| `b2_acceptance_test.go` | Catalog black-box acceptance | Acceptance/UX | 013 | 22 | Curated enrichment, inventory-derived categories, search, selection across filter/resize, detail, zero writes. | Opt-in |
| `b8_confirmation_test.go` | Confirmation/footer/help | Safety/negative | 013 | 11 | One confirmation surface, inert backdrop, reverse Tab, eligibility matrices; writes only on explicit keys. | Opt-in |
| `b8_impact_test.go` | Impact projection | Unit | 013 | 5 | Classifies each planned change once (new/replace/remove/unchanged/unknown), never mutates plan/token; 94-entry stress. | Opt-in |
| `b8_preview_test.go` | Install preview/no-op | Integration | 013 | 13 | Grouped exact totals, unknown/refused never look successful, no-op makes zero host calls and leaves home unchanged. | Opt-in |
| `b9_home_setup_test.go` | Home setup/labels/results | Acceptance/UX | 013 | 16 | Evidence-derived setup state, goal-first Home, plain labels, typed outcomes, generation-bound refresh. | Opt-in |
| `branding_test.go` | Brand mark/reveal | Acceptance/UX | 013 | 11 | Identity constants, finite deterministic reveal, logo alignment, width fit, 7-bit ASCII. | Opt-in |
| `catalog_b2_test.go` | CatalogModel | Unit | 013 | 13 | Enrichment/unknown fallback, filter fields, categories, selection preservation, detail card, stale filter results. | Opt-in |
| `catalog_routing_test.go` | Routing metadata display | Acceptance/UX | 013 | 10 | Profiles show provider/primary/fallbacks/policy; skills/concepts invent nothing; badge, section order, narrow wrap. | Opt-in |
| `catalog_test.go` | Catalog load/model | Unit | 013 | 13 | Opt-in/TTY gates, config normalize, read-only load, counts/errors, status hints, nav/filter/headless. | Opt-in |
| `cmd/tui-companion/main_test.go` | Companion entrypoint | Lifecycle/PTY | 019 (+013) | 18 | Protocol/version/help non-interactive, flag exclusivity, TTY gates, legacy forwarding, fail-closed identity. | Opt-in |
| `contrast_test.go` | Palette/colored chrome | Acceptance/UX | 013 | 10 | Mocha role hexes and WCAG ratios, color-off plain path, canvas/top bar/focus/help bounds. | Opt-in |
| `feedback_t20_test.go` | Write feedback/status block | Lifecycle/PTY | 013 | 16 | starting/writing/flushing phases, stale tick ignored, error/refusal/quit stop, notice contrast, bounds. | Opt-in |
| `fleet_detail_scroll_test.go` | Fleet detail pane | Acceptance/UX | 015 | 2 | At width 120 focus detail and scroll without moving selection; 80/100 keep compact footer. | Opt-in |
| `fleet_doctor_test.go` | Diagnostics panel | Acceptance/UX | 015 | 11 | Doctor defaults offline, explicit binary runs once, exact vocabulary, focus/esc/scroll, stale reply discarded. | Opt-in |
| `fleet_edit_test.go` | Model picker/staging | Integration | 015 | 6 | Exact model validation/suggestions, one/many edits, enable/disable consequence, dirty discard/keep, zero writes. | Opt-in |
| `fleet_help_w3d_test.go` | Fleet help topic | Acceptance/UX | 015 | 5 | Topic listed without reordering, exact truths, fits widths, reachable from Quick start, hint skippable. | Opt-in |
| `fleet_negative_test.go` | Fleet failure matrix | Safety/negative | 015 | 31 | All 30 spec scenarios: parse/symlink refusal, stale digest, drift, rollback, lock conflict, geometry, no optimistic success. | Opt-in |
| `fleet_production_test.go` | Production fleet adapters | Integration | 015 | 6 | Reader maps status/missing source without writes; editor/planner preview, backups, restore refusal write nothing. | Opt-in |
| `fleet_readonly_test.go` | Default reader/read-only journey | Safety/negative | 015 | 2 | DefaultFleetReader reports not available; journey leaves home/file mtimes, contents and backups untouched. | Opt-in |
| `fleet_recovery_test.go` | Fleet backup recovery | Integration | 015 | 4 | Backup listing/exact preview, consent defaults No and dispatches once, stale/modified/legacy refusals, outcome refresh. | Opt-in |
| `fleet_review_test.go` | Review Changes view | Acceptance/UX | 015 | 5 | Impact summary counts, source diff rendering/color, long-diff paging, stale reply discarded, filter/scroll continuity. | Opt-in |
| `fleet_scale_test.go` | Fleet scale/latency | Integration | 015 | 4 | 18/100-instance inventories, 100-op review and >1,000-line diff reachable; benchmark asserts p95 <= 100ms. | Opt-in |
| `fleet_test.go` | Fleet hub/overview/profiles | Acceptance/UX | 015 | 8 | Four-entry hub with wrap, lazy read only after opening, overview, search/filter/detail, responsive sizes, stale generation. | Opt-in |
| `fleet_write_test.go` | Fleet deployment apply | Integration | 015 | 11 | Review counts/diff, consent defaults No and dispatches once, dirty/conflict blocks, live gate hides, single slot. | Opt-in |
| `guide_test.go` | Guide destination | Unit | 019 | 7 | No engine/exec/network imports; key 5/g/help entry, return-origin/reset, per-route scroll, plain narrow bounds. | Opt-in |
| `install_test.go` | InstallModel journey | Unit | 013 | 15 | Step order/cancel/back, no client, invalid preset, consent invalidation, resize keeps No, call counts, no preview writes. | Opt-in |
| `integration_b4b5_test.go` | Install/restore/activity/onboarding | Integration | 013 | 12 | Preview-confirm-apply journeys, restore via single slot, cross-screen write refusal, Ctrl+C deferral, token retention. | Opt-in |
| `integration_regression_test.go` | Cross-route regressions | Integration | 013 | 7 | ASCII/Unicode bounds at four widths, footer quit, footer/scroll persistence, scroll-up after end, last selection visible. | Opt-in |
| `layout_b2_test.go` | Layout/wrap helpers | Unit | 013 | 3 | Prose wraps on word boundaries, category control reachable, one-cell gutters at 60/80/110. | Opt-in |
| `library_help_test.go` | Library rows/Activity help | Acceptance/UX | 013 | 7 | Row title/description with exact ID, task search terms, Router recommendation, detail hierarchy, help sections. | Opt-in |
| `mcp_a_hub_regression_test.go` | Install/Manage hub regression | Integration | 015 | 2 | Four hub children keep keys/indices with MCP Tools fourth; Fleet Manager reachable by cursor and key. | Opt-in |
| `mcp_test.go` | MCP Tools screens/write host | Integration | 015 | 13 | Overview/nav/esc, selection/auth toggles, preview token, default-No and tiny geometry, diagnostics, busy refusal. | Opt-in |
| `modal_t19_test.go` | Modal compositor | Acceptance/UX | 013 | 9 | Centered fit, fullscreen tiny fallback, Unicode/ANSI/OSC safety, focus/scroll restoration, resize never confirms. | Opt-in |
| `onboarding_t34_test.go` | Optional Quick start | Integration | 013 | 6 | Tour summary reflects choices, skip/reopen/back preserve selections/consent, post-setup prompt, no browser/clipboard/agent. | Opt-in |
| `onboarding_test.go` | Shared onboarding model | Acceptance/UX | 013 | 11 | Step order/finish, safe back/skip/cancel, copyable links, tiny bounds, OSC8 gating, no writes. | Opt-in |
| `opener_test.go` | Browser opener/hyperlinks | Unit | 013 | 11 | Only fixed HTTPS links launch, arbitrary URLs refused, platform argv, SSH/headless/failure fallbacks, OSC8 gating. | Opt-in |
| `ops_host_fleet_test.go` | Fleet write kinds/single host | Integration | 015 (+013) | 7 | Fleet verbs/request validation, transcript identity, source fallback, nil-seam refusal, single-slot conflict, completion identity. | Opt-in |
| `ops_host_test.go` | Single-write host/busy policy | Lifecycle/PTY | 013 | 17 | Completes once, concurrent refusal, invalid rejected before slot, log failures, private transcript, busy defers quit/signals. | Opt-in |
| `preferences_t30_test.go` | Session preferences | Acceptance/UX | 013 | 5 | Help exposes motion/symbol/offline toggles, idle status cluster empty, notice contrast, Activity reports real state. | Opt-in |
| `presentation_escaping_test.go` | ASCII data escaping | Safety/negative | 013 | 4 | Paths/IDs/notices render reversible 7-bit text while preview/write requests keep exact original bytes. | Opt-in |
| `presentation_t18_test.go` | Shell chrome | Acceptance/UX | 013 | 8 | Breadcrumbs, factual status cluster, stage-accurate footer, marker/gutter geometry, breakpoints, safe glyphs, long/wide data. | Opt-in |
| `presentation_test.go` | Pure text/Home helpers | Unit | 013 | 22 | Paragraph/section wrap with cell/grapheme/ANSI widths, Home counts/unknown metadata/width/read-only. | Opt-in |
| `recovery_test.go` | Backup inspection presentation | Acceptance/UX | 013 | 11 | Fresh-vs-replacement breakdown, legacy/empty/unusable honesty, log association never guessed, stale/unsafe refusal. | Opt-in |
| `responsive_t35_test.go` | Responsive matrix | Acceptance/UX | 013 | 8 | 8 widths x 4 heights plus 140/100/80/60 samples, five states, tiny-terminal safety, inline snapshots. | Opt-in |
| `restore_test.go` | RestoreModel journey | Integration | 013 | 13 | Enter opens details only, single apply, no-change honesty, subset isolation, exact preview, drift invalidates consent. | Opt-in |
| `safety_review_test.go` | Safety/state review fixes | Safety/negative | 013 | 9 | Confirmation inert below usable geometry, stale reads cannot overwrite newer, no-op refresh, router override, contextual key. | Opt-in |
| `signals_test.go` | OS signal policy | Lifecycle/PTY | 013 | 6 | Platform signal set, forwarding incl. repeats, idempotent stop/terminal release, no-send safety, SIGKILL residual risk documented. | Opt-in |
| `theme_adaptive_test.go` | Theme tokens/profiles | Unit | 013 | 8 | Dark palette AAA pairs, role tokens, TrueColor/256/16 mappings, late background reply, color-off, no local raw hex. | Opt-in |
| `update_state_test.go` | Update check manager | Unit | 013 | 10 | Not-started state, one check per session, env/explicit disable, retry, stale suppression, cancellation, transitions. | Opt-in |
| `wrapping_test.go` | Path wrap helper | Unit | 013 | 1 | wrapPath keeps an emoji ZWJ grapheme cluster intact when splitting at width 3. | Opt-in |

### Python harness — `tests/harness`

_21 files, 183 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `test_appops_core_selection.py` | Core package selector text | Static/fixture | 008 (+013, 014, 016) | 9 | Eleven core packages listed identically; no broad discovery or excluded/experimental selection. | Explicit |
| `test_connected.py` | Runner to receipt integration | Integration | 008 (+004) | 3 | Observed runs, authority/freshness errors, interruption stays not_run, sentinels survive reload. | Explicit |
| `test_core_workflow.py` | Core instruction text | Static/fixture | 008 | 11 | Required human-input/direct-fix/review clauses, link integrity, honest not_run labels. | Explicit |
| `test_independent_review.py` | Inactive review example | Static/fixture | 006 | 9 | Two-agent steps, denied tools, sensitive reads, model/variant pinning. | Explicit |
| `test_independent_review_dispatch.py` | Dispatch environment | Unit | 006 | 1 | PWD/HOME/XDG isolated; no inherited API key. | Explicit |
| `test_independent_review_docs.py` | Review reference text | Static/fixture | 006 | 6 | Trigger/blocked-route/no-edit/untrusted-text rules present; seeded deletion detected. | Explicit |
| `test_independent_review_fixtures.py` | Review reconciliation/freshness | Integration | 006 | 13 | Report reconciliation, scope/freshness/scheduling rules on scratch repos. | Explicit |
| `test_offline_transport.py` | S0 byte-quota/HTTP controls | Unit | 004 | 21 | Request size/streaming refusals against a loopback fake gate. | Gated |
| `test_opencode.py` | OpenCode template/catalog | Static/fixture | 004 | 7 | Deny-by-default provider/MCP/plugin policies, no warming, catalog invariants. | Explicit |
| `test_p1_driver.py` | P1 host driver cleanup | Unit | 004 | 3 | Container ID cleanup and evidence files survive timeouts/errors. | Explicit |
| `test_p1_network.py` | P1 loopback courier | Integration | 004 | 5 | Network-none courier relays only permitted body/headers; denial paths never reach upstream. | Gated |
| `test_p1_transport.py` | P1 policy/transport | Unit | 004 | 9 | Budget/dispatch-ticket/single-request policy refusals on a fake exchange. | Explicit |
| `test_receipts.py` | Receipt/evidence consumer | Unit | 008 (+004) | 18 | Identity, coverage, authority, secret-canary and symlink rejections; valid positives kept. | Explicit |
| `test_recovery.py` | sdd-check interruption/deadline | Integration | 008 (+004) | 2 | Owned interruption preserves partials, no later-check replay, unrelated process survives. | Explicit |
| `test_research_workflow.py` | Research skill references | Static/fixture | 018 | 22 | Clauses, link integrity, evidence-handoff fields, privacy/effort boundaries, mutation rejection. | Explicit |
| `test_s0_native.py` | Native request construction | Integration | 004 | 1 | Pinned runtime sends only to the fake gate and uses fake output. | Gated |
| `test_supported_core_go.py` | Core Go selection | Unit | 008 | 8 | Exact 11-package argv, sanitized env, fail-closed exit propagation, task/alias parity. | Explicit |
| `test_taskfile.py` | Task CLI/release profile | Integration | 008 (+007) | 11 | Preview/apply entry points, tag/release guards, no experimental tag in release build, trusted paths. | Explicit |
| `test_tokenizer_source.py` | Pinned tokenizer wheel/data | Static/fixture | 004 | 4 | Exact hashes and local encodings in a network-none lab; skip is not a pass. | Gated |
| `test_trusted_ci.py` | Trusted-gate policy | Unit | 008 | 14 | Anti-weakening: rejects missing bootstrap, dropped contracts, empty Make, forged receipts. | Explicit |
| `test_trusted_ci_container.py` | Trusted gate in Docker | E2E/container | 008 | 6 | Positive/negative fixtures, non-root/no-network, CID cleanup on interruption. | Gated |

### Python TUI acceptance — `tests/tui`

_5 files, 39 declared cases._

| File | Scope | Type | Feature | Cases | What it verifies | Selection |
|---|---|---|---|---:|---|---|
| `test_fleet_lifecycle.py` | Fleet Manager under PTY | Lifecycle/PTY | 015 | 4 | Preview-only claim, resize survival, clean quit, read-only home. | Opt-in PTY |
| `test_launcher_lifecycle.py` | Parent/child launcher pair | Lifecycle/PTY | 019 (+013) | 19 | Canonical vs legacy entry behavior, deprecation notice, signal relay, exit codes. | Opt-in PTY |
| `test_mcp_lifecycle.py` | MCP Tools under PTY | Lifecycle/PTY | 015 | 3 | Three default servers, navigation, confirm defaults to No, read-only. | Opt-in PTY |
| `test_readonly_catalog.py` | Catalog browsing under PTY | Lifecycle/PTY | 013 | 3 | Real shipped inventory shown, curated detail, browsing writes nothing, clean quit. | Opt-in PTY |
| `test_terminal_lifecycle.py` | Terminal lifecycle of asdx | Lifecycle/PTY | 019 (+013) | 10 | Alternate screen/line discipline restored, early error, signal deferral. | Opt-in PTY |

### Shell and container drivers — `tests/`

_17 drivers, no unittest-style cases._

| Driver | Scope | Type | Feature | What it verifies | Selection |
|---|---|---|---|---|---|
| `tests/build-qa-image.sh` | Feature 005 QA image build | E2E/container | 005 | Builds the pinned QA image offline from local binaries. | Historical |
| `tests/docker-e2e.sh` | Installer container wrapper | E2E/container | 001 (+003, 008) | Runs the installer E2E in golang:1.25-alpine with network none and no pull. | Focused |
| `tests/e2e-in-container.sh` | Installer preview/apply/backup/restore | E2E/container | 001 (+003) | Preview writes nothing, manifest v2, idempotent re-apply, exact restore inverse. | Focused |
| `tests/guide-e2e-in-container.sh` | Guided CLI journeys J1-J10 | E2E/container | 010 (+011) | Guided install/decline/restore, non-TTY, color/NO_COLOR, narrow, animation, update check. | Opt-in |
| `tests/guide-e2e.sh` | Guide container wrapper | E2E/container | 010 | Runs guided journeys in an offline alpine container with no pull. | Opt-in |
| `tests/harness/diagnosis-fixtures.sh` | B5 diagnosis fault oracles | E2E/container | 008 (+004) | Distinguishes tautology vs contract, setup vs no-op, bad advice vs restore on a pinned tree. | Opt-in |
| `tests/harness/independent-review-transaction.sh` | Manual review transaction | E2E/container | 006 | Scratch-only digest-checked walkthrough; never a real home. | Historical |
| `tests/harness/native-independent-review-dispatch.sh` | Review dispatch namespace probe | E2E/container | 006 | Offline rootless user/network namespace dispatch with no external inference. | Historical |
| `tests/harness/native-independent-review.sh` | Review example probe | E2E/container | 006 | Pinned 2.2.5 binary with no network/home/credentials. | Historical |
| `tests/harness/native-opencode.sh` | B3 native adapter probe | E2E/container | 004 | Runs native_opencode.py against a pinned binary with network none. | Opt-in |
| `tests/harness/trusted-ci-fixtures.sh` | B7 fixture driver | E2E/container | 008 | Runs policy tests then container fixtures with a private evidence root. | Opt-in |
| `tests/qa-container/entrypoint.sh` | Feature 005 QA entrypoint | E2E/container | 005 | Installs skills, writes inactive candidate config, prints identity. | Historical |
| `tests/run-qa-image.sh` | Feature 005 QA image runner | E2E/container | 005 | Offline read-only run, opt-in --online, no host home or credentials. | Historical |
| `tests/supported-core-go.sh` | Core Go test/vet selection | Integration | 008 | Runs 11 packages, checks required dirs, sanitized offline env. | Core |
| `tests/vnext-container.sh` | Feature 005 human-cycle container | E2E/container | 005 | Disposable offline container with read-only repo and --batch mode. | Historical |
| `tests/vnext-model-free-checks.sh` | Feature 005 model-free checks | E2E/container | 005 | version, preview no-write, 18x4 apply, no-op re-apply, backup/replace. | Historical |
| `tests/vnext-online-check.sh` | Feature 005 online check | E2E/container | 005 | DNS/connectivity checks and one opt-in prompt; never prints credentials. | Historical |


## Gaps, caveats and maintenance notes

- **No root or `internal/version` tests.** `go test .` and `./internal/version` are required gate targets, but neither package has a `*_test.go`; the gate compiles them and trusts the packages they import. This is a real coverage gap, not a hidden pass.
- **Python is never default.** `task test` runs Go only. The harness suites are selected one file at a time, and the repository explicitly forbids broad `tests/harness/` discovery. A green core gate says nothing about the 222 Python cases.
- **Skips are not passes.** Five environment-gated Python suites and the nested TUI task can skip when a container, binary, mount or Go version is absent. The files say so in their docstrings; a completed CI run with skips has not executed those cases.
- **Two Go modules.** `experiments/tui` has its own `go.mod` and Go 1.26 requirement. `task test` cannot select it; `tui:test` is the only full-suite entry point.
- **Historical lanes remain in the tree.** Feature 005's `tests/vnext-*`, `tests/build-qa-image.sh`, `tests/run-qa-image.sh` and the 005 QA image are retained evidence, not a work queue. Feature 006's native review probes are likewise fenced out of default selection. Do not add them to an aggregate gate or read a stored observation as a current pass.
- **Multi-feature files are counted once.** `cmd/agentic-sdd/main_test.go`, `tests/tui/test_launcher_lifecycle.py` and several TUI suites span two or three features. The matrix uses the primary feature; the per-file tables show the secondary ones.
- **Contract coupling.** Distribution counts (18/19 skills, 14/4 profiles), model IDs, permission-rule order, routing prose, `.goreleaser.yml` and `harness/checks.json` are pinned as fixtures. Legitimate asset renames require deliberate test rebaselining; otherwise these suites fail loudly.
- **Cross-language parity.** `internal/harnessevidence/snapshot_test.go` contains golden digests derived independently by `tests/harness/go_snapshot_fixture.py`, and `tests/harness/receipt_tools.py` is a separate Python consumer of the same receipt shape. The duplication is intentional; changing one side without the other is a defect.
- **Minor dead weight.** `experiments/tui/theme_adaptive_test.go` still declares unused “Daylight Lagoon” light-palette constants after the light theme was removed. They do not affect the suite, but they are misleading fixture text.
- **Timing-sensitive case.** `experiments/tui/fleet_scale_test.go` asserts p95 ≤ 100 ms for 100-row navigation/render, and it is the only wall-clock threshold in the TUI suite. It is the most likely flake on a loaded machine.
- **No browser/frontend tests.** There is no JavaScript/TypeScript test layer in this repository; the CLI, TUI and Python harness are the whole automated surface.

## Appendix A — Go test case names

Generated from `^func (Test|Benchmark|Fuzz)` declarations. Names are grouped by package; the file-level type, feature and selection are in the tables above.

#### CLI — `cmd/agentic-sdd` (31 files, 227 cases)

- **`b3_compat_test.go`** (6): `TestB3CompatApplySuccessNonTTY`, `TestB3CompatApplyNoOpNonTTY`, `TestB3CompatRestorePreviewNonTTY`, `TestB3CompatRestoreApplyNonTTY`, `TestB3CompatBackupsListingNonTTY`, `TestB3CompatFailureBeforeBackupNonTTY`
- **`branding_summary_test.go`** (2): `TestBrandingSummaryPreview`, `TestBrandingSummaryApply`
- **`branding_test.go`** (10): `TestBrandingStaticIntroWide`, `TestBrandingStaticNoColorHasNoANSI`, `TestBrandingNarrowFallback`, `TestBrandingArtAtSixtyColumns`, `TestBrandingAnimationChunksAndLimits`, `TestBrandingAnimationSuppressed`, `TestBrandingAnimationNilWait`, `TestBrandingWrapProse`, `TestBrandingStyles`, `TestBrandingStackedCounts`
- **`env_test.go`** (4): `TestIsTerminal`, `TestResolveEnv`, `TestRunEnvRestoreWithDevNullStdin`, `TestResolveEnvAnimate`
- **`execution_cli_test.go`** (3): `TestExecutionCLIInitBriefStatusRecordClean`, `TestExecutionCLIDirectScopeAndRefusals`, `TestExecutionCLIHelpAndFormat`
- **`execution_integrity_test.go`** (4): `TestExecutionIntegrityForeignCheckpointRefused`, `TestExecutionIntegrityDeletedRegisteredBrief`, `TestExecutionIntegrityLegacyUnregisteredBrief`, `TestExecutionIntegrityRepeatAndForce`
- **`fleet_apply_test.go`** (1): `TestFleetApplyCLI`
- **`fleet_doctor_test.go`** (4): `TestFleetDoctorCLIOfflineDefault`, `TestFleetDoctorCLIValidationAndUsage`, `TestFleetDoctorCLIUnavailableLabelIsStillSuccess`, `TestFleetDoctorCLIHelpSurface`
- **`fleet_lock_test.go`** (1): `TestFleetLock`
- **`fleet_status_golden_test.go`** (4): `TestFleetStatusGoldenWithoutInstall`, `TestFleetStatusGoldenManagedDriftAndUnsafe`, `TestFleetStatusGoldenInvalidConfigExit`, `TestFleetStatusGoldenMatchesBuildStatus`
- **`fleet_test.go`** (4): `TestFleetCLI`, `TestFleetHelpLegend`, `TestFleetExitCodesTable`, `TestFleetStatusDriftLabelsAndWarnings`
- **`guide_branding_test.go`** (5): `TestGuideBrandingIntroOnceAndTerminalMenu`, `TestGuideBrandingPlainBranchExact`, `TestGuideBrandingAnimationGate`, `TestGuideBrandingNarrowInstall`, `TestGuideBrandingWideStyling`
- **`guide_catalog_alias_test.go`** (3): `TestTeachingRegistryIsSharedMetadata`, `TestTeachingWrappersMatchSharedAPI`, `TestTeachingCustomWrapperKeepsNeutralFallback`
- **`guide_catalog_test.go`** (7): `TestTeachingRegistryMatchesShippedAssets`, `TestTeachingKnownCardsComplete`, `TestTeachingExamplesAreHumanRequests`, `TestTeachingExamplesDoNotRepeatLabels`, `TestTeachingCustomFallback`, `TestTeachingConceptsStableOrder`, `TestTeachingNoModelIDsOrCostClaims`
- **`guide_explorer_test.go`** (10): `TestGuideExplorerCoreDetailStatusesAndNavigation`, `TestGuideExplorerProfilesIncludeBasesAndOptionalPresets`, `TestGuideExplorerMissingOptionalPackKeepsCoreAvailable`, `TestGuideExplorerUnreadableOptionalPackIsUnavailable`, `TestGuideExplorerUnknownCustomIDSafeFallback`, `TestGuideExplorerConceptsStable`, `TestGuideExplorerMainMenuReturnAndListEOF`, `TestGuideExplorerEOFAndInvalidInput`, `TestGuideExplorerTraceIsBoundedAndPrivate`, `TestGuideExplorerDetailBackNavigationAndGroupedHeaders`
- **`guide_plain_compatibility_test.go`** (1): `TestGuidePlainBrandingCompatibility`
- **`guide_presentation_test.go`** (15): `TestGuidePresentationEmbeddedPackReview`, `TestGuidePresentationCustomSingleSkill`, `TestGuidePresentationNarrowRecords`, `TestGuidePresentationReviewBackThenDecline`, `TestGuidePresentationReviewBackThenConfirm`, `TestGuidePresentationReviewBackChangesOptions`, `TestGuidePresentationFailureGuidance`, `TestGuidePresentationNoOpCreatesNothing`, `TestGuidePresentationLogsSafetyError`, `TestGuideSelectionCommandQuotingRoundTrip`, `TestGuidePresentationDefinitionHangingIndent`, `TestGuidePresentationDefinitionBlockFormat`, `TestGuidePresentationDefinitionLabelWiderThanWidth`, `TestGuidePresentationDefinitionFieldsFitNarrowWidth`, `TestGuidePresentationDefinitionEmptyText`
- **`guide_test.go`** (4): `TestGuideNonInteractiveAndFlags`, `TestGuideInstallAndDeclineAndRepeat`, `TestGuideLogsRestoreSharedReaderAndEOF`, `TestGuideExamplePromptsAppearInREADME`
- **`guide_updates_test.go`** (4): `TestGuideUpdatesOnlyExplicitTerminalChoice`, `TestGuideUpdatesChoiceFiveAndRepeat`, `TestGuideUpdatesStates`, `TestGuideUpdatesTracePrivacyAndLegacyBytes`
- **`guide_ux_test.go`** (2): `TestGuideUXMenuReturns`, `TestGuideUXRestoreImpactBeforeConfirmation`
- **`logs_test.go`** (6): `TestApplyLogAndStdoutExact`, `TestRestoreLogAndStdoutExact`, `TestApplyPackFailurePreservesPartialTranscript`, `TestApplyAndRestoreTraceCarryStructuredBackupID`, `TestApplyFailureBeforeBackupTraceOmitsBackupID`, `TestApplyPostBackupFailureReportsBackupID`
- **`main_test.go`** (19): `TestRunHelpAndUsageErrors`, `TestRunPreviewAndApplyCompatibility`, `TestRunVersion`, `TestRunPreviewAndApplyFromEmbeddedSource`, `TestRunOperationalFailureUsesExitOne`, `TestRunBackupsEqualsRestoreList`, `TestRunRestorePreviewEqualsBareID`, `TestRunRestoreApplyFormsEquivalent`, `TestRunRestoreUsageErrors`, `TestRunRestoreStateErrorsExitOne`, `TestRunRestoreNonInteractiveMissingID`, `TestRunRestoreInteractiveFlow`, `TestApplyLogAndLogsCommand`, `TestApplyFailureLog`, `TestRestoreApplyLog`, `TestApplyRefusesSymlinkedLogDirectory`, `TestRunOpenCodeFlagsAndHelp`, `TestPlainOutputByteIdentity`, `TestTopicHelpCanonicalAndMigration`
- **`mcp_test.go`** (15): `TestMCPDispatch`, `TestMCPStatusText`, `TestMCPStatusNotConfigured`, `TestMCPStatusJSON`, `TestMCPDoctorText`, `TestMCPDoctorJSON`, `TestMCPPreviewText`, `TestMCPPreviewJSON`, `TestMCPEngineError`, `TestMCPMissingReportIsAnError`, `TestMCPRequestValidationError`, `TestMCPRequestArguments`, `TestMCPNoTTYOutputIsDeterministic`, `TestMCPWritesNothing`, `TestMCPHelpAndExistingCommands`
- **`mcp_usage_test.go`** (3): `TestMCPErrorCodeUsageVersusRuntime`, `TestMCPTargetRendersAmbiguity`, `TestMCPTargetRendersMissingExplicitPath`
- **`mcp_write_test.go`** (26): `TestMCPWritePreviewPlanText`, `TestMCPWritePreviewPlanJSON`, `TestMCPWritePreviewConflictExitsOne`, `TestMCPWritePreviewAuthChoices`, `TestMCPWritePreviewUnknownServerUsage`, `TestMCPWriteApplyRequiresExpectConfig`, `TestMCPWriteApplyStaleDigestRefused`, `TestMCPWriteApplyConflictRefused`, `TestMCPWriteApplyMissingTargetRefused`, `TestMCPWriteApplyWritesOnce`, `TestMCPWriteApplyNoOp`, `TestMCPWriteApplyJSON`, `TestMCPWriteApplyEngineFailures`, `TestMCPWriteApplyPassesTokenAndDigestExactly`, `TestMCPWriteUsageErrors`, `TestMCPWriteRestoreListEmpty`, `TestMCPWriteRestoreListPopulatedJSON`, `TestMCPWriteRestorePreviewTextAndJSON`, `TestMCPWriteRestorePreviewConflict`, `TestMCPWriteRestoreApplyRequiresIDAndDigest`, `TestMCPWriteRestoreApplyStaleDigest`, `TestMCPWriteRestoreApplyConflict`, `TestMCPWriteRestoreApplyPromptDefaultNo`, `TestMCPWriteRestoreApplyPromptYes`, `TestMCPWriteRestoreApplyNonInteractive`, `TestMCPWriteNoCredentialsOrFullConfig`
- **`opencode_disabled_test.go`** (7): `TestDefaultHelpAdvertisesOnlySupportedCommands`, `TestDefaultOpenCodeDenialHasNoSideEffects`, `TestDefaultHelpOpenCodeOnlyExplainsUnavailability`, `TestDefaultOpenCodeDenialIgnoresPathAndNativeLookup`, `TestDefaultBuildExcludesExperimentalFilesAndAdapterImport`, `TestDefaultTransitiveDependenciesExcludeExperimentalBackend`, `TestDefaultGoListExcludesExperimentalFiles`
- **`opencode_test.go`** (8): `TestOpenCodeCommandsStayOfflineAndDefaultToPreview`, `TestOpenCodeApplyFailureRetainsActionableBackupObservation`, `TestOpenCodeSessionPrepareStaysOffline`, `TestOpenCodeSessionRunIsAdmissionBlocked`, `TestOpenCodeSessionPendingReviewsWithoutReentry`, `TestOpenCodeSessionUnknownOperationRefused`, `TestOpenCodeEvidenceSnapshotRequiresExplicitSafeScope`, `TestOpenCodeEvidenceReceiptValidateExportRoundTrip`
- **`trace_test.go`** (6): `TestTraceCommandAndFlagNames`, `TestTraceJSONLines`, `TestTraceGuideFlow`, `TestTraceOpenTraceNormalizesModeAndAppends`, `TestTraceDisabledIsNoOp`, `TestTraceOpenFailuresIsolateCommand`
- **`tui_test.go`** (21): `TestTUIDispatchMatrix`, `TestTUIExplicitRequestForwarding`, `TestTUICompanionArgs`, `TestTUICommandUsageErrors`, `TestTUIWithoutTTYNeverLaunches`, `TestResolveTUICompanion`, `TestQueryTUICompanionProtocol`, `TestBoundedBufferTruncates`, `TestLaunchTUIHandshake`, `TestLaunchTUIPropagatesExitCodeAndForwardsArguments`, `TestLaunchTUIRelaysHandledSignals`, `TestLaunchTUIQueuesSignalBeforeChildStarts`, `TestLaunchTUIReprocessFailureFailsClosed`, `TestLaunchTUIFailsClosedWithoutCompanion`, `TestLaunchTUITraceRecordsDecision`, `TestRunStartTraceNamesBarePreview`, `TestCanonicalBuildHasNoLegacyIdentity`, `TestLegacyExactTopLevelQueriesIdentifyBothNames`, `TestLegacyForwardsEveryOtherInvocation`, `TestLegacyResolvesSymlinkedInstallation`, `TestLegacyMissingOrInvalidTargetFailsClosed`
- **`ui_test.go`** (3): `TestTerminalSummaryAndColor`, `TestHelpTopics`, `TestErrorHintsTerminalOnly`
- **`updates_test.go`** (19): `TestUpdatesClientConfig`, `TestUpdatesRequestContract`, `TestUpdatesResultStates`, `TestUpdatesComparator`, `TestUpdatesInvalidCurrentUnavailable`, `TestUpdatesFailurePrecedesInvalidCurrent`, `TestUpdatesMaliciousFields`, `TestUpdatesIgnoresUntrustedReleaseFields`, `TestUpdatesNon200`, `TestUpdatesBodyLimit`, `TestUpdatesClosesBody`, `TestUpdatesTransportFailure`, `TestUpdatesRejectsRedirect`, `TestUpdatesPrintNewer`, `TestUpdatesPrintStates`, `TestUpdatesSharedCheckerEquivalence`, `TestUpdatesStateMapping`, `TestUpdatesSharedRequestParity`, `TestUpdatesSharedPolicyConstants`

#### Check CLI — `cmd/sdd-check` (1 files, 2 cases)

- **`main_test.go`** (2): `TestUsageAndInvalidArguments`, `TestMissingPrerequisiteHasStructuredBlockedResult`

#### Installer core — `internal/skillsync` (17 files, 124 cases)

- **`identity_test.go`** (5): `TestSyncSourceDigestStableAcrossPreviewAndApply`, `TestSyncSourceDigestTracksCoreSource`, `TestSyncSourceDigestTracksOpenCodePack`, `TestRestoreSourceDigestStableAndManifestSensitive`, `TestRestoreSourceDigestTracksPayload`
- **`inventory_test.go`** (10): `TestInventoryContractValues`, `TestInventoryEmbeddedCoreOnly`, `TestInventoryEmbeddedPackAndPresets`, `TestInventoryCustomSourceWithoutPack`, `TestInventoryPackSourceErrorsDistinguishMissingFromInvalid`, `TestInventoryCoreTargetStatuses`, `TestInventoryUnsafeTargetParents`, `TestInventoryProfileStatuses`, `TestInventoryUnreadablePackSourceIsNotMissing`, `TestInventoryRejectsFreeFirstWithoutOpenCode`
- **`manifest_test.go`** (2): `TestApplyWritesFormat2Manifest`, `TestReadManifestDetectsLegacyVersusFormat2`
- **`mcp_compat_test.go`** (1): `TestMCPCompatWithOpenCodeLeavesOpenCodeConfigByteIdentical`
- **`mcp_config_test.go`** (20): `TestMCPPlanHomeRequired`, `TestMCPPlanAddPreservesUnrelatedAndIsDeterministic`, `TestMCPPlanAddCreatesMissingContainers`, `TestMCPPlanChangeManagedEntry`, `TestMCPPlanLeaveAndNoOp`, `TestMCPPlanDisabledNeverReenabled`, `TestMCPPlanConflictRefusal`, `TestMCPPlanUserModifiedManagedEntryConflicts`, `TestMCPPlanMissingTargetNeverCreates`, `TestMCPPlanV1SchemaRefused`, `TestMCPPlanInvalidDocumentsRefused`, `TestMCPPlanAmbiguityBothTargets`, `TestMCPPlanUnsafeTargets`, `TestMCPPlanJSONCRefusalAndManualSnippet`, `TestMCPPlanJSONCNoOpNeedsNoWrite`, `TestMCPPlanAuthRecipes`, `TestMCPPlanSelectionValidation`, `TestMCPPlanLiteralCredentialRefusedWithoutEcho`, `TestMCPPlanIgnoresOwnershipForAnotherTarget`, `TestMCPPlanManifestFormatRefused`
- **`mcp_test.go`** (14): `TestMCPDefaultServers`, `TestMCPStatus_HomeRequired`, `TestMCPStatus_NoConfig`, `TestMCPStatus_JSONV2_UnrelatedServerAndUnknownFields`, `TestMCPStatus_JSONC_CommentsAndTrailingCommas`, `TestMCPStatus_V1LegacySchema`, `TestMCPStatus_AmbiguityBothJSONAndJSONC`, `TestMCPStatus_MalformedInvalidDuplicateOversize`, `TestMCPStatus_DisabledServer`, `TestMCPStatus_SymlinkTarget`, `TestMCPStatus_CodeGraphLookupAndIndexState`, `TestMCPStatus_AuthModes`, `TestMCPStatus_ExplicitConfigPathOutsideDefaultScope`, `TestMCPPreview_Semantics`
- **`mcp_transaction_test.go`** (20): `TestMCPApplyAddWritesExactBytesBackupAndManifest`, `TestMCPApplyPreservesTargetMode`, `TestMCPApplyChangeManagedEntry`, `TestMCPApplyNoOpCreatesNoSecondBackup`, `TestMCPApplyConflictRefusesAndAvoidsWrite`, `TestMCPApplyStaleDigestRefusesWithoutWrite`, `TestMCPApplyRuntimeProbeFailsClosed`, `TestMCPApplyFailureBeforeRenamePreservesTarget`, `TestMCPApplyFailureAfterRenameRollsBackEntries`, `TestMCPApplyFailureAfterManifestLeavesRecoverableEvidence`, `TestMCPApplyNeedsRecoveryWhenRollbackCannotVerify`, `TestMCPApplyLockRefusesCompetingWriter`, `TestMCPLockStaleTakeoverAndFreshRefusal`, `TestMCPRestorePreviewAndApplyPreserveUnrelated`, `TestMCPRestoreRefusesUserEditedManagedEntry`, `TestMCPRestoreRequiresExplicitDigest`, `TestMCPRestoreRefusesMissingCorruptOrAbsentBackup`, `TestMCPRestoreFailureBeforeRenamePreservesTarget`, `TestMCPRestoreFailureAfterRenameRollsBack`, `TestMCPTransactionIntentEvidence`
- **`opencode_adapters_test.go`** (1): `TestWithOpenCodeExcludesFleetAdapters`
- **`opencode_pack_test.go`** (1): `TestOpenCodePackAssetsAndPermissions`
- **`opencode_sync_test.go`** (7): `TestDefaultIsolationAndPreviewNoWrite`, `TestApplyWithOpenCodeAdditive`, `TestApplyWithOpenCodeFleetPreservationAndOptionB`, `TestApplyWithOpenCodeFreeFirstAndFileDigestTamper`, `TestApplyWithOpenCodeIdempotentAndSubsequentDefaultApply`, `TestApplyWithOpenCodeSymlinkRejectionAndRollback`, `TestRestoreFormat3OpenCodeRoundTrip`
- **`package_test.go`** (4): `TestCanonicalDistributionIncludesDebuggingAndSiblingReferences`, `TestIndependentReviewReferenceIsPackaged`, `TestResearchSearchWorkflowReferenceIsPackaged`, `TestCanonicalInstallPreviewNoopRestoreInIsolatedHome`
- **`permissions_test.go`** (4): `TestPermissionOnlyDifferenceIsNoOpWithoutNewBackup`, `TestInstalledModesUseRequestedModesSubjectToUmask`, `TestRestoreNormalizesModesInsteadOfRestoringBackedUpModes`, `TestPermissionProbeChild`
- **`restore_test.go`** (9): `TestListBackupsEmptyOrMissingRoot`, `TestListBackupsOrderingLegacyAndSummary`, `TestRestorePreviewWritesNothing`, `TestRestoreFormat2RoundTrip`, `TestRestoreFormat1PutsBackOnlySavedTrees`, `TestRestoreInstallRollsBackMixedReplaceAndRemove`, `TestRestoreRefusesNonDirectoryTarget`, `TestRestoreRefusesSymlinkInInstalledTree`, `TestLoadBackupRejections`
- **`result_test.go`** (13): `TestOperationResultZeroValueIsNotSuccess`, `TestSyncResultReplacementAndManifestIdentity`, `TestSyncResultPreviewIsReadOnly`, `TestSyncResultFailureBeforeBackup`, `TestSyncResultFailureAfterBackup`, `TestInstallChangesReportsRollbackAfterBackup`, `TestRestoreResultStructuredOutcomes`, `TestSyncWrapperResultPreviewEquivalence`, `TestSyncWrapperResultApplyEquivalence`, `TestSyncWrapperResultNoChangeEquivalence`, `TestRestoreWrapperResultPreviewEquivalence`, `TestRestoreWrapperResultApplyEquivalence`, `TestRestoreErrorEquivalence`
- **`selection_test.go`** (5): `TestNormalizeClientSelection`, `TestClientSelectionRejectedBeforeWrites`, `TestSubsetApplyBackupRestoreRoundTrip`, `TestSubsetPreviewApplyInventoryParity`, `TestSubsetSelectionWithOpenCodePack`
- **`summary_test.go`** (1): `TestSummarizePlanDrift`
- **`sync_test.go`** (7): `TestSyncBacksUpAndReplaces`, `TestSyncRefusesSymlinkedBackupDirectory`, `TestSyncRefusesSourceSymlinkBeforeChanges`, `TestSyncRefusesConflictingTargetBeforeChanges`, `TestSyncRefusesSymlinkBeforeChanges`, `TestRollbackRestoresOriginalAndRemovesNewSkill`, `TestInstallChangesRollsBackRemoveOnLaterFailure`

#### App operations — `internal/appops` (7 files, 76 cases)

- **`fleet_test.go`** (7): `TestFleetStatus`, `TestFleetSourcePreview`, `TestFleetPreview`, `TestFleetApply`, `TestFleetRestore`, `TestFleetDoctor`, `TestFleetApplyRequestValidation`
- **`mcp_test.go`** (21): `TestMCPRequestsRequireHome`, `TestMCPRequestsAbsolutizePaths`, `TestMCPRequestsAllowEmptyOptionalPaths`, `TestMCPPreviewRequestCopiesSelection`, `TestMCPPreviewRequestPreservesNilSelection`, `TestMCPStatusIsReadOnly`, `TestMCPPreviewIsReadOnlyWithoutToken`, `TestMCPDoctorCheckOrderAndVocabulary`, `TestMCPDoctorMissingConfig`, `TestMCPDoctorAmbiguityBlocked`, `TestMCPDoctorLegacySchemaBlocked`, `TestMCPDoctorMissingCodeGraphBinary`, `TestMCPDoctorIndexAbsentIsNotPass`, `TestMCPDoctorAuthNeverPasses`, `TestMCPDoctorDoesNotExecuteOrLeak`, `TestMCPSourceIsOfflineOnly`, `TestMCPDoctorTargetAndSchemaLabels`, `TestMCPDoctorBinaryLabelsUsePresenceOnly`, `TestMCPDoctorIndexLabels`, `TestMCPDoctorAuthLabelsNeverPass`, `TestMCPDoctorRuntimeAndConnectivityAreNotChecked`
- **`mcp_write_test.go`** (12): `TestMCPPlanRequestValidationAndImmutability`, `TestMCPApplyRequestValidationAndImmutability`, `TestMCPRestorePreviewRequestValidation`, `TestMCPRestoreRequestValidation`, `TestMCPRestoreListValidationAndBehavior`, `TestMCPPlanPreviewNoWrite`, `TestMCPApplyAndRestoreCallEngineOnFixtures`, `TestMCPPlanPreviewToken`, `TestMCPRestorePreviewToken`, `TestMCPApplyTokenAndDigestRevalidation`, `TestMCPRestoreTokenAndDigestRevalidation`, `TestMCPWriteSourceIsOfflineOnly`
- **`ops_test.go`** (9): `TestNewInstallRequestValidatesAndCopies`, `TestNewRestoreRequestValidates`, `TestOperationIDStaleness`, `TestPreviewTokenStableAndApplyParity`, `TestApplyInstallRequiresMatchingPreview`, `TestUnselectedDestinationEditDoesNotInvalidatePreview`, `TestApplyRefusesEmptyStaleAndDoubleTokens`, `TestSubsetRoundTripThroughSeam`, `TestApplyRestoreRequiresMatchingPreview`
- **`routing_test.go`** (4): `TestAgentRoutingMatchesDocumentedPolicy`, `TestPresetRoutingMatchesFrontmatter`, `TestAgentRoutingDefensiveCopy`, `TestAgentRoutingUnknownProfile`
- **`teaching_test.go`** (7): `TestTeachingForKnownSkillCard`, `TestTeachingForKnownPresetProfileCard`, `TestTeachingForUnknownFallsBackNeutrally`, `TestTeachingCustomCardAlwaysFallsBack`, `TestTeachingCardsIncludesConceptsAndIsDefensive`, `TestTeachingConceptsStableOrderAndDefensive`, `TestTeachingKindAndGroupConstants`
- **`update_test.go`** (16): `TestUpdateStateZeroValueIsFailure`, `TestCheckLatestReleaseRequestContract`, `TestCheckLatestReleaseStates`, `TestCheckLatestReleaseComparator`, `TestCheckLatestReleaseInvalidCurrentUnavailable`, `TestCheckLatestReleaseFailurePrecedesInvalidCurrent`, `TestCheckLatestReleaseUntrustedMetadata`, `TestCheckLatestReleaseIgnoresUntrustedReleaseFields`, `TestCheckLatestReleaseNon200`, `TestCheckLatestReleaseBodyLimit`, `TestCheckLatestReleaseClosesBody`, `TestCheckLatestReleaseTransportFailure`, `TestCheckLatestReleaseNilResponse`, `TestCheckLatestReleaseNilInputs`, `TestCheckLatestReleaseRefusesRedirect`, `TestCheckLatestReleaseContextCancellation`

#### Build metadata — `internal/buildinfo` (2 files, 13 cases)

- **`buildinfo_test.go`** (11): `TestDefaultsAreHonestAndUnverifiable`, `TestIdentityValidate`, `TestIdentityEqualAndString`, `TestDistributionMatrix`, `TestFleetCapabilityDefaultsToPreviewAndFailsSafe`, `TestFleetCapabilityMatrix`, `TestReleaseConfigKeepsFleetPreviewOnly`, `TestProtocolInfoJSONIsStable`, `TestParseProtocolInfoRejectsUnknownAndTrailingData`, `TestVerifyCompanion`, `TestVerifyCompanionRejectsUninjectedParent`
- **`release_config_test.go`** (2): `TestGoReleaserPairContract`, `TestUniversalNameMappingCoversEveryBuild`

#### Fleet engine — `internal/fleet` (14 files, 56 cases)

- **`apply_test.go`** (1): `TestFleetApply`
- **`diff_test.go`** (6): `TestFleetEncodeConfigDeterministic`, `TestFleetPreviewSource`, `TestFleetLineDiff`, `TestFleetPlanFiles`, `TestFleetPlanFilesLongUnicodePath`, `TestFleetPlanFilesSymlinkRefusal`
- **`doctor_test.go`** (14): `TestFleetDoctorOfflineByDefault`, `TestFleetDoctorLabelsFromLocalEvidence`, `TestFleetDoctorOpenAIRouteOnly`, `TestFleetDoctorAuthMarkerRequiredForAuthenticated`, `TestFleetDoctorUnknownMarkerIsNotAuthentication`, `TestFleetDoctorUnavailableWhenInspectionFails`, `TestFleetDoctorProbeErrorYieldsUnavailable`, `TestFleetDoctorCatalogProviderOnlyDump`, `TestFleetDoctorMalformedDumpFailsClosed`, `TestFleetDoctorHelpResolvedListingOnlyUsesListingSubcommands`, `TestFleetDoctorModelRouteGrammar`, `TestFleetDoctorMergesDumpAndBinEvidence`, `TestFleetDoctorEmptyModels`, `TestFleetDoctorModelsFromConfig`
- **`integrity_test.go`** (6): `TestFleetApplyIntegrity`, `TestFleetBackupDigestVerification`, `TestFleetLockAtomic`, `TestFleetLockOwnership`, `TestFleetLockOwnerRecordIsDiagnostic`, `TestFleetRestoreIntegrity`
- **`negative_matrix_test.go`** (6): `TestFleetSidecarApplyingAndFailedState`, `TestFleetNegativeMatrixRow3DuplicateInstalledNameOrAliasToAlias`, `TestFleetNegativeMatrixRow5OpenCodeOverlapVsFleet`, `TestFleetNegativeMatrixRow12SpecPlanAdapterNonwritingPolicy`, `TestFleetNegativeMatrixRow18OrchestratorInstallDoesNotChangeDefault`, `TestFleetNegativeMatrixRow19GitRevertRequiresExplicitApply`
- **`parse_test.go`** (1): `TestFleetParse`
- **`plan_test.go`** (3): `TestFleetPlan`, `TestFleetLockUnit`, `TestFleetPlanReservedAndRouteWarnings`
- **`render_test.go`** (2): `TestFleetRender`, `TestFleetPermissions`
- **`resolve_test.go`** (2): `TestFleetResolve`, `TestFleetResolveRealPack`
- **`restore_preview_test.go`** (2): `TestFleetRestorePreview`, `TestFleetRestorePreviewRefusals`
- **`restore_test.go`** (1): `TestFleetRestore`
- **`scale_015_test.go`** (4): `TestFleet015ScaleBuildStatus`, `TestFleet015ScalePreviewSource`, `TestFleet015ScalePlanFiles`, `TestFleet015ScaleLineDiffOver1000Lines`
- **`scale_test.go`** (2): `TestFleetDoctorScaleRecordsTimings`, `TestFleetBuildPlanScaleRecordsTimings`
- **`status_test.go`** (6): `TestFleetStatusZeroOneAndReadOnly`, `TestFleetStatusManagedPendingAndDrifted`, `TestFleetStatusUnmanagedLegacyAndUnsafe`, `TestFleetStatusDisabled`, `TestFleetStatusScale`, `TestFleetStatusRefusesInvalidConfig`

#### Execution runs — `internal/execution` (10 files, 123 cases)

- **`briefintegrity_test.go`** (4): `TestBriefRegisterContract`, `TestBriefInspectArtifactsDiagnosesDrift`, `TestStatusBriefFreshness`, `TestBriefNextInvestigationReasons`
- **`checkpoint_identity_test.go`** (1): `TestLoadCheckpointRequiresOwningRunID`
- **`checkpoint_test.go`** (14): `TestHighestStatePrecedence`, `TestNormalizedBatchScopeDigest`, `TestCheckpointRoundTrip`, `TestStatusPrepared`, `TestStatusSourceAndWorktreeFreshness`, `TestStatusMissingArtifact`, `TestStatusInterruptedCheck`, `TestStatusBlocked`, `TestStatusContradictoryEvidence`, `TestStatusScopeMismatch`, `TestStatusNeverUsesApprovalLanguage`, `TestStatusStateMachineTable`, `TestLoadCheckpointRejectsUnknownState`, `TestStatusUnknownRun`
- **`checks_test.go`** (19): `TestCheckHelperProcess`, `TestCheckPassReachesVerifiedEvidence`, `TestCheckArtifactsAndPrivateProvenance`, `TestCheckFailureFarFromTail`, `TestCheckTimeout`, `TestCheckSignalIsInterrupted`, `TestCheckOutputLimitTerminatesAndTruncates`, `TestCheckSummaryTruncation`, `TestCheckSummaryTailFallback`, `TestCheckNoSecretEcho`, `TestCheckMissingProgramIsNotStarted`, `TestCheckRequestValidation`, `TestCheckConcurrentLockAndRunningStatus`, `TestCheckReleasesLockOnEveryPath`, `TestCheckWorktreeChangedDuringCheck`, `TestCheckLaterPassReachesVerifiedEvidence`, `TestCheckKilledWriterLeavesInterruptedPartialState`, `TestCheckEvidenceWriteFailureIsNotAPass`, `TestCheckDirectScopeDefaultsTask`
- **`record_test.go`** (8): `TestRecordParseValidFixtures`, `TestRecordParseTaskResolution`, `TestRecordRejectsInvalidInput`, `TestRecordIngestPassedRemainsAwaitingVerification`, `TestRecordIngestBlockedYieldsBlocked`, `TestRecordIngestRejectsInvalidWithoutArtifact`, `TestRecordIngestTaskMismatch`, `TestRecordIngestDirectDefault`
- **`repository_test.go`** (12): `TestRepositoryOpenAndObserve`, `TestRepositoryRejectsNonRepository`, `TestRepositoryRejectsUnbornHead`, `TestRepositoryObservationTracksWorktree`, `TestRepositoryObserveExcludingSources`, `TestFeatureScope`, `TestFeatureScopeFailsClosed`, `TestDirectScope`, `TestValidateRelativePath`, `TestPathIdentity`, `TestReadBoundedRegularRefusesSymlink`, `TestDigestBytesIsStable`
- **`reviewpackage_test.go`** (18): `TestReview_Committed`, `TestReview_Staged`, `TestReview_Unstaged`, `TestReview_Untracked`, `TestReview_Rename`, `TestReview_Delete`, `TestReview_ModeChange`, `TestReview_Binary`, `TestReview_LargeFile`, `TestReview_MaxTotalBytes`, `TestReview_Sensitive`, `TestReview_Symlink`, `TestReview_SubmoduleGitlink`, `TestReview_ConflictedIndex`, `TestReview_HeadMismatch`, `TestReview_NoMutation`, `TestReview_InvalidBase`, `TestReview_CheckpointIntegration`
- **`taskbrief_test.go`** (19): `TestTaskBrief_HeadingT1`, `TestTaskBrief_HeadingT01`, `TestTaskBrief_T1_vs_T10`, `TestTaskBrief_CheckboxForm`, `TestTaskBrief_NestedHeadingsAndNonTaskBoundary`, `TestTaskBrief_CodeFences`, `TestTaskBrief_MalformedFence`, `TestTaskBrief_CRLF`, `TestTaskBrief_UnresolvedReference`, `TestTaskBrief_DuplicateReferenceInSpec`, `TestTaskBrief_MissingTask`, `TestTaskBrief_WrongBatchOnlyTask`, `TestTaskBrief_MissingAndDuplicateBatch`, `TestTaskBrief_DuplicateTaskInBatch`, `TestTaskBrief_UnparseableTaskHeading`, `TestTaskBrief_ScopeDigest_ProgressOnly`, `TestTaskBrief_IdempotenceAndForce`, `TestTaskBrief_FileSizeLimitAndUTF8`, `TestTaskBrief_DirectScopeRefusal`
- **`taskbrief_transaction_test.go`** (9): `TestTaskBrief_Transaction_LiveWriterLock`, `TestTaskBrief_Transaction_ContradictoryRequestAndDirectScope`, `TestTaskBrief_Transaction_RegistrationAndStatus`, `TestTaskBrief_Transaction_PreservesPriorTaskState`, `TestTaskBrief_Transaction_LegacyRepairAndCheckpointFailure`, `TestTaskBrief_Transaction_TamperedBriefForce`, `TestTaskBrief_Transaction_SourceAndBatchDrift`, `TestTaskBrief_Transaction_LockReleasedOnError`, `TestTaskBrief_PureExtraction`
- **`workspace_test.go`** (19): `TestRunIDFormat`, `TestValidateRunID`, `TestWorkspaceInitCreatesPrivateIgnoredRun`, `TestWorkspaceInitFailClosedOnVisibleIgnore`, `TestWorkspaceInitRequiresGitRepository`, `TestWorkspaceInitRejectsUnbornHead`, `TestWorkspaceInitRejectsScopeRootSubdirectory`, `TestWorkspaceInitExplicitRootOutsideRepository`, `TestWorkspaceInitExplicitRootInsideRepositoryReportsVisibility`, `TestWorkspaceInitRefusesSymlinkedScratchParent`, `TestWorkspaceInitRefusesSymlinkRunRoot`, `TestResolveRunRoot`, `TestLockConflictAndRelease`, `TestLockOwnershipCheckedRelease`, `TestLockStaleReclaim`, `TestCleanPreviewApplyAndAbsent`, `TestCleanRefusesLiveLock`, `TestCleanRefusesUnownedAndSymlinkedRuns`, `TestCleanRefusesMovedRun`

#### Check runner and evidence — `internal/harnesscheck`, `internal/harnessevidence` (7 files, 46 cases)

- **`contract_test.go`** (5): `TestContractValidation`, `TestRepositoryContract`, `TestSelection`, `TestTidyUsesIndependentSnapshotRatherThanGitIndex`, `TestContainerRecipeCannotPullImplicitly`
- **`process_linux_test.go`** (3): `TestCancellationStopsOwnedGroupNotUnrelatedProcess`, `TestLeaderCannotPassWithLiveDescendant`, `TestTerminationWaitIsBoundedAndReportsUnconfirmed`
- **`runner_test.go`** (14): `TestCheckHelper`, `TestPassAndPrivateEvidence`, `TestFailureStopsLaterStepsAndChecks`, `TestMissingExecutableAndPrerequisiteBlock`, `TestBoundedOutput`, `TestTidyDetectsNewUntrackedSum`, `TestPathsFailBeforeExecution`, `TestFocusedDoesNotClaimRequired`, `TestTimeoutAndCancellation`, `TestTimeoutCoversAllStepsNotEachStep`, `TestArgvIsLiteralAndEnvOverridesControlLookup`, `TestMissingEvidenceCannotPass`, `TestLogRetainsAnEarlierWriteError`, `TestClosedLogCanDrainLateOutputWithoutWriting`
- **`export_test.go`** (5): `TestExportRedactsBeforeWriteAndValidates`, `TestExportRefusesOverwriteAndUnsafeTargets`, `TestExportRejectsUnapprovedFieldsWithoutEchoingValues`, `TestExportRejectsInvalidReceiptAndMalformedJSON`, `TestExportScrubsCredentialForms`
- **`receipt_fixture_test.go`** (0): _no test functions (shared fixture)_
- **`snapshot_test.go`** (5): `TestSnapshotMatchesReferenceGolden`, `TestSnapshotDetectsWorkingTreeChanges`, `TestSnapshotRefusesUnsafeScopesAndFiles`, `TestSnapshotExplicitScopeIgnoresUnrelatedFiles`, `TestSnapshotExplicitNamedIgnoredFileIsIncluded`
- **`validate_test.go`** (14): `TestValidateAcceptsValidDirectFixWithoutTouchingWorkspace`, `TestValidateRejectsSchemaAndSemanticInconsistency`, `TestValidateRejectsStaleOrUnknownIdentity`, `TestValidateCannotInventObservedValues`, `TestValidateAcceptsIndependentlyObservedSyntheticValues`, `TestValidateEvidenceBoundaries`, `TestValidateReadyRules`, `TestValidateCancelledOrIncompleteCannotBePromoted`, `TestValidateStatusCannotBeRelabelledAsAnotherOutcome`, `TestValidateRequiredCoverageCannotBeForgedBySelectionLabel`, `TestValidateSavedReceiptSurvivesReloadButChangedCodeBlocksReuse`, `TestValidateWorkspaceMoveRequiresReconciliation`, `TestValidateRejectsForeignResultAndContract`, `TestValidateRejectsDirectFixTaskState`

#### Experimental OpenCode adapter — `internal/opencodeadapter` (7 files, 44 cases)

- **`acp_test.go`** (5): `TestACPFrameBoundRejected`, `TestACPFrameMidMessageEOFRejected`, `TestACPFrameInvalidJSONAndWrongVersion`, `TestACPRoundTripPreservesIDAndParams`, `TestACPWriteRejectsMultiline`
- **`adapter_test.go`** (10): `TestCandidateIsInactiveAndNamespaced`, `TestPreflightProjectionAndRejectedInputs`, `TestConfigPreviewApplyNoopAndRollback`, `TestExistingConfigCommentsModeAndInterveningEdit`, `TestUnsafePathsSensitiveSourceAndChangedCandidate`, `TestCandidateCannotEnableReviewWritesOrRemoveGuardPermissions`, `TestPreparedBackupSurvivesFailureAndInterveningWriter`, `TestReadInputRefusesCredentialStoreNames`, `TestJSONCValidationPreservesStringMeaningAndRejectsAmbiguity`, `TestRollbackRejectsCorruptedBackupAndForeignManifest`
- **`files_unix_test.go`** (1): `TestSpecialFilesAndSymlinkDestinationsAreRejected`
- **`pending_test.go`** (5): `TestAdmitIsAlwaysDeniedWithoutControls`, `TestNewPendingSummaryNormalizesUnknownRationale`, `TestParseOptionsExtractsBoundedChoices`, `TestWriteReadPendingRoundTripAndRefusals`, `TestWritePendingRequiresPrivateNonDiscoveryDirectory`
- **`receipt_test.go`** (8): `TestProjectCompletedTurnComposesChecksAndValidates`, `TestProjectRefusesFailedTurnOverCompletedResult`, `TestProjectRequiresEndTurnForCompletedNativeTurn`, `TestProjectRecordsMissingNativeObservationAsUnknown`, `TestProjectWritesPendingSidecarOutsideDiscovery`, `TestProjectRefusesIdentityDriftAndContractMismatch`, `TestProjectRefusesInvalidContextAndObservation`, `TestProjectElicitationFromFakePeer`
- **`runner_test.go`** (3): `TestACPSleepHelper`, `TestSyntheticCancellationPreservesUnrelatedProcesses`, `TestSpawnOwnedProcessUsesAllowlistedEnvironment`
- **`session_test.go`** (12): `TestACPFakePeerProcess`, `TestSessionHappyPathCompletesWithUsage`, `TestSessionProviderErrorExitZeroIsNotSuccess`, `TestSessionElicitationBlocksWithPendingHuman`, `TestSessionPermissionBlocksWithoutApproval`, `TestSessionChildMetadataBlocksAndInvalidatesUsage`, `TestSessionForeignSessionRefused`, `TestSessionProtocolViolations`, `TestSessionDeadlineExceededStopsTheSession`, `TestSessionWrongVersionRefused`, `TestSessionModelNotOfferedRefused`, `TestSessionRetryIsABoundedObservation`

#### TUI — `experiments/tui`, separate Go module (52 files, 518 cases)

- **`activity_test.go`** (11): `TestActivity_ImmediateLocalRender`, `TestActivity_EmptyState`, `TestActivity_BoundedLogLoading`, `TestActivity_SelectedLogNavigation`, `TestActivity_VersionAndHelpMetadata`, `TestActivity_UpdateCheckIntegration`, `TestActivity_PlainModeAndBoundsSafety`, `TestActivity_BubbleTeaLifecycle`, `TestActivityASCIIDataEscapingIsReversible`, `TestActivityAppReadsOnlyThroughCommands`, `TestActivityModelUpdateSchedulesReads`
- **`app_test.go`** (22): `TestAppInitialState`, `TestAppRouteNavigation`, `TestAppTabFocusCycling`, `TestAppExploreSelectionPreservedAcrossRoutes`, `TestAppFilterPrecedence`, `TestAppSingleContextualFooter`, `TestAppResponsiveLayoutBreakpoints`, `TestAppBrandTickIntegration`, `TestAppSessionControls`, `TestAppInstallHubAndActivityContent`, `TestAppHelpOverlay`, `TestAppHomeEndNavigation`, `TestTerminalExactBounds`, `TestASCIIAll7Bit`, `TestRouteSwitchFocusAndHomeSetup`, `TestAppBrandTickLifecycle`, `TestInstallBackupRootAndHints`, `TestAppMessageRoutingAndFilterExecution`, `TestScrollableLongContentAndEndMarkers`, `TestFooterAlwaysVisibleAtShortHeights`, `TestUnicodePathASCIIReversibleEscaping`, `TestBrandTickOffscreenSuppression`
- **`b2_acceptance_test.go`** (22): `TestB2KnownItemIsEnrichedFromSharedMetadata`, `TestB2UnknownItemFallsBackNeutrally`, `TestB2UnknownItemDetailShowsNoFabricatedState`, `TestB2CategoriesCountTheRealInventory`, `TestB2CategoriesDoNotFabricateAbsentContent`, `TestB2PurposeCategoryFollowsTeachingGroup`, `TestB2CategoryFilterSelectsMatchingItems`, `TestB2CategorySwitchRestoresEveryItem`, `TestB2SearchMatchesDisplayNameAndPurpose`, `TestB2SearchMatchesExactIDAndCategory`, `TestB2ZeroMatchSearchStaysUsable`, `TestB2PresetCategoryIsOptionalAndDerived`, `TestB2AsyncSearchResultIsExecutedAndConsumed`, `TestB2SelectionSurvivesCategoryFilterAndResize`, `TestB2DetailShowsCompleteRichFields`, `TestB2DetailSpacingIsReadable`, `TestB2DetailCardHonoursItsWidthBudget`, `TestB2LongPathStaysReachableByPaging`, `TestB2CellBoundsAtEveryBreakpoint`, `TestB2ReadOnlyRunWritesNothing`, `TestB2HomeShowsInventoryDerivedCounts`, `TestB2HomeOffersRecommendedInstallAction`
- **`b8_confirmation_test.go`** (11): `TestB8ConfirmationHasOneAuthoritativeSurface`, `TestB8ConfirmationBackdropIsInert`, `TestB8RestoreConfirmationOwnsInputAndQDoesNotQuit`, `TestB8InstallConfirmationQSafeCancelsWithoutQuitting`, `TestB8ShiftTabCyclesFocusInReverse`, `TestB8ContextualFooterAndHelpShareEligibility`, `TestB8HelpModalQDoesNotQuit`, `TestB8RestoreUnknownActionBlocksConfirmation`, `TestB8InstallEligibilityMatrixOnlyYStartsAWrite`, `TestB8RestoreConfirmationEligibilityMatrix`, `TestB8RestoreImpactTotalsAndPathsRemainInspectable`
- **`b8_impact_test.go`** (5): `TestB8ImpactProjectionReconcilesEveryItemOnce`, `TestB8ImpactGroupsUseTypedDestinationIdentity`, `TestB8ImpactNoOpClassification`, `TestB8ImpactStress94Entries`, `TestB8ImpactSummaryUsesExactCounts`
- **`b8_preview_test.go`** (13): `TestB8InstallPreviewLeadsWithTotalsAndDestinationGroups`, `TestB8PreviewGroupSelectionExpandsOneDestination`, `TestB8UnknownActionIsExplicitAndBlocksConfirmation`, `TestB8IncompleteOrRefusedPreviewNeverLooksSuccessful`, `TestB8NoOpPreviewIsReadOnlyAndRequiresNoConfirmation`, `TestB8NoOpChangeOptionsInvalidatesPreview`, `TestB8ChangedLaterPreviewRequiresNormalConfirmation`, `TestB8NoOpAppNeverCallsTheWriteHost`, `TestB8NoOpHomeAndLibraryUseExistingRoutes`, `TestB8RealNoOpLeavesTemporaryHomeUnchanged`, `TestB8PinnedSummaryWhileStressDetailsPage`, `TestB8PreviewBoundsAcrossBreakpoints`, `TestB8PreviewIdentityIsPresentationOnly`
- **`b9_home_setup_test.go`** (16): `TestDeriveSetupStateClassifiesLocalEvidence`, `TestHomeLeadsWithPurposeThenSetupThenOneAction`, `TestHomeStatesUseEvidenceAndOneAction`, `TestHomeKeyboardPrimaryAndRecovery`, `TestHomeNoForcedOnboardingOrNetworkState`, `TestPlainLabelsKeepEngineIdentityAndDestinations`, `TestPreviewAndResultsUsePlainClientLabels`, `TestExplicitEmptySelectionIsRejected`, `TestInstallResultOutcomeText`, `TestUnknownInstallOutcomeIsClassifiedHonestly`, `TestRestoreUnknownResultText`, `TestInstallRefreshUsesSameHomeSourceAndOptions`, `TestNoChangeAndRestoreRefreshKeepTheKnownScope`, `TestRefreshResultsAreGenerationBound`, `TestRefreshFailureKeepsOperationOutcomeAndShowsUnknown`, `TestReplaceCatalogPreservesSelection`
- **`branding_test.go`** (11): `TestBrandIdentityConstants`, `TestBrandStaticStateIsComplete`, `TestBrandRevealIsFiniteAndBounded`, `TestBrandLogoBoxStaysAligned`, `TestBrandRevealIsDeterministic`, `TestBrandRevealStartsPartialAndSettlesOnIdentity`, `TestBrandHeaderFitsTerminalWidth`, `TestBrandHeaderCollapsesWithoutWidth`, `TestBrandHeaderASCIIIsSevenBit`, `TestBrandHeaderUnicodeKeepsTextLabels`, `TestBrandHeaderIsUnstyledText`
- **`catalog_b2_test.go`** (13): `TestB2CatalogItemAdditiveFieldsAndEnrichment`, `TestB2UnknownItemNeutralFallback`, `TestB2FilterValueIncludesAdditiveFields`, `TestB2CategoriesAndActiveCategory`, `TestB2SetCategoryFiltersVisibleItemsAndPreservesSelection`, `TestB2DetailCardRendersRichSections`, `TestB2StaleFilterSafety`, `TestB2NewCatalogModelDoesNotMutateCallerSnapshot`, `TestB2OutOfOrderFilterResultsDiscarded`, `TestB2FilteredSelectionPreservedAcrossCategory`, `TestB2MismatchFixtureCountsAuthoritativeKindOrigin`, `TestB2DisplayItemPreservesCuratedMetadata`, `TestB2DetailCardWrapsProseAtNarrowWidth`
- **`catalog_routing_test.go`** (10): `TestRoutingProfileItemsCarryCuratedRouting`, `TestRoutingSummaryBadgeIsCompactAndBounded`, `TestRoutingFilterValueIncludesRoutingTerms`, `TestRoutingDetailCardSectionOrderAndContent`, `TestRoutingOrchestratorShowsActiveSessionModel`, `TestRoutingSkillAndConceptCardsOmitAgentSection`, `TestRoutingUncuratedProfileIsHonest`, `TestRoutingDetailCardWrapsAtNarrowWidth`, `TestRoutingDisplayItemOverrideKeepsRouting`, `TestRoutingPresetDetailShowsLadder`
- **`catalog_test.go`** (13): `TestLaunchAllowed`, `TestTUIConfigNormalize`, `TestLoadCatalogReadOnly`, `TestLoadCatalogCoreOnly`, `TestLoadCatalogRequiresHome`, `TestLoadCatalogMissingPack`, `TestStatusBadgeAndHints`, `TestSummarizePreview`, `TestCatalogModelNavigation`, `TestCatalogModelFilter`, `TestCatalogModelFilterKeys`, `TestCatalogProgramHeadless`, `TestCatalogModelEmpty`
- **`cmd/tui-companion/main_test.go`** (18): `TestProtocolInfoIsStableAndNonInteractive`, `TestProtocolInfoWinsOverOtherFlags`, `TestVersionIsNonInteractive`, `TestHelpIsPublicAndNonInteractive`, `TestUnknownFlagFailsWithUsage`, `TestUnexpectedArgumentFails`, `TestInfoFlagsAreMutuallyExclusive`, `TestFreeFirstRequiresOpenCodePack`, `TestFleetConfigFlagIsAccepted`, `TestNonTTYDoesNotLaunch`, `TestForcedInteractiveDoesNotLaunch`, `TestCanonicalBuildHasNoLegacyIdentity`, `TestLegacyExactTopLevelQueriesIdentifyBothNames`, `TestLegacyProtocolInfoIsForwardedWithoutNotice`, `TestLegacyForwardsEveryOtherInvocation`, `TestLegacyResolvesSymlinkedInstallation`, `TestLegacyMissingOrInvalidTargetFailsClosed`, `TestLegacyProtocolInfoWithOtherFlagsStaysSilent`
- **`contrast_test.go`** (10): `TestThemePaletteRolesMeetWCAG`, `TestThemeUsesAgenticDarkPalette`, `TestThemeColorOffIsPlain`, `TestColoredViewKeepsLegacyPlainPathUnstyled`, `TestColoredViewSetsTerminalCanvas`, `TestColoredTopBarUsesSurfacePaddingAndDivider`, `TestColoredTopBarCollapsesOnShortTerminals`, `TestColoredFocusIndicatorStaysTextualAndAccented`, `TestColoredChromePreservesControlsAndBounds`, `TestColoredHelpOverlayKeepsTitleAndFooter`
- **`feedback_t20_test.go`** (16): `TestT20SpinnerStartsAdvancesAndStops`, `TestT20StaleTickIsIgnored`, `TestT20StaticMarkerWhenMotionIsUnavailable`, `TestT20HostPhaseSequenceDrivesFeedback`, `TestT20WriteErrorKeepsSpinnerStoppedAndNoticeComplete`, `TestT20QuitStopsSpinnerAndKeepsDeferredExitVisible`, `TestT20BusyPolicyPrecedenceAndNoDuplicateNotice`, `TestT20NoticePersistsInReservedStatusBlock`, `TestT20LongNoticeStaysCompleteInBody`, `TestT20NoFabricatedProgressOrNativeProgress`, `TestT20AC15NoticeRoleKeepsContrast`, `TestT20AC16BoundsWithReservedStatusNotice`, `TestT20AC16ModalSafetyWithStatusNotice`, `TestT20AC16ConfirmationSemanticsUnchanged`, `TestT20JourneyKeepsPolicyAndLogReference`, `TestT20RefusalStopsSpinnerAndPersists`
- **`fleet_detail_scroll_test.go`** (2): `TestFleetProfilesWideDetailScroll`, `TestFleetProfilesNarrowLayoutUnchanged`
- **`fleet_doctor_test.go`** (11): `TestFleetDoctorDefaultsOfflineWithoutInspectingAnything`, `TestFleetDoctorExplicitBinaryRunsOnceAndLabelsExactVocabulary`, `TestFleetDoctorLabelVocabularyIsExact`, `TestFleetDoctorFocusTabBackspaceAndClear`, `TestFleetDoctorEscReturnsAndKeepsTheUnderlyingView`, `TestFleetDoctorPanelIsBoundsSafeAndScrolls`, `TestFleetDoctorStaleReplyIsDiscarded`, `TestFleetDoctorDoesNotChangeTheLiveGate`, `TestAppOpsFleetDoctorDefaultConsultsNothing`, `TestFleetDoctorPanelReachableThroughTheShell`, `TestAppOpsFleetDoctorRefusesMissingOrUnusableBinary`
- **`fleet_edit_test.go`** (6): `TestFleetModelPickerExactValidAndInvalid`, `TestFleetModelPickerSuggestionSelection`, `TestFleetStageOneAndManyEdits`, `TestFleetEnableDisableConsequenceManagedRemoval`, `TestFleetDirtyProtectionDiscardAndKeep`, `TestFleetStagingZeroWritesAndNoRealHome`
- **`fleet_help_w3d_test.go`** (5): `TestW3DFleetHelpTopicIsListed`, `TestW3DFleetHelpTopicStatesExactTruths`, `TestW3DFleetHelpFitsReviewedWidths`, `TestW3DFleetHelpIsReachableFromQuickStart`, `TestW3DFleetHintIsOptionalAndSkippable`
- **`fleet_negative_test.go`** (31): `TestNegativeMatrixAll30ScenariosMapped`, `TestNegative01_NoFleetConfig`, `TestNegative02_ParseErrorAndInvalidJSON`, `TestNegative03_SymlinkAndPathTraversalRefusal`, `TestNegative04_ExactUnknownModelCatalogedUnknown`, `TestNegative05_ModelOnlyChangePreservesPromptAndPermissions`, `TestNegative06_DisableManagedInstanceVsUnmanagedCollision`, `TestNegative07_StagedEditEscDiscardConfirmation`, `TestNegative08_StaleEditorReplyAndDigestMismatch`, `TestNegative09_SourceSavedOpenCodeUnchanged`, `TestNegative10_SavedSourceInvalidBlocksPlan`, `TestNegative11_StalePlanBetweenPreviewAndConfirm`, `TestNegative12_TargetChangedDriftBlocksApply`, `TestNegative13_LegacyUnknownProvenanceNotAdopted`, `TestNegative14_UnmanagedCollisionUntouched`, `TestNegative15_InterruptedApplyRollbackRecorded`, `TestNegative16_RestoreBackupCorruptOrDriftedRefusal`, `TestNegative17_TwoRapidEntersOnApplySingleDispatch`, `TestNegative18_NavigationAndResizeDuringApplyIgnored`, `TestNegative19_LiteralKeysInInputs`, `TestNegative20_ConfirmationGeometryBounds`, `TestNegative21_LargeDiffUnicodeReachability`, `TestNegative22_NonTTYOrScriptedLaunchPreserved`, `TestNegative23_LiveGatePreviewOnlyDisablesApplyAndRestore`, `TestNegative24_DoctorCatalogedUnknownAuth`, `TestNegative25_ReviewerIsolationNeverFalselyClaimedSecure`, `TestNegative26_GitMetadataRefusedNoCrash`, `TestNegative27_SharedLockAndSingleSlotConflict`, `TestNegative28_ZeroReadsWhenFleetUnopened`, `TestNegative29_UIReportsFailureNeverOptimisticSuccess`, `TestNegative30_ReadOnlyTargetRefusesWithoutDamage`
- **`fleet_production_test.go`** (6): `TestAppOpsFleetReaderMapsStatusWithoutWrites`, `TestAppOpsFleetReaderReportsMissingSourceTruthfully`, `TestAppOpsFleetEditorPreviewNoWrites`, `TestAppOpsFleetEditorPreviewPreservesTopLevelMetadata`, `TestAppOpsFleetPlannerPreviewAndNoWrites`, `TestAppOpsFleetPlannerBackupsAndRestoreRefusal`
- **`fleet_readonly_test.go`** (2): `TestFleetDefaultReaderSafeNotAvailable`, `TestFleetZeroWritesAndNoRealHome`
- **`fleet_recovery_test.go`** (4): `TestFleetRecoveryListsBackupsAndExactPreview`, `TestFleetRestoreConfirmationDefaultsToNoAndDispatchesOnce`, `TestFleetRecoveryRefusalsExplainWhatHappened`, `TestFleetRestoreResultShowsRealOutcomeAndRefreshes`
- **`fleet_review_test.go`** (5): `TestFleetReviewChangesImpactSummary`, `TestFleetReviewSourceDiffRenderingAndColor`, `TestFleetReviewLongDiffPaging`, `TestFleetStaleEditorReplyDiscarded`, `TestFleetViewFilterScrollContinuity`
- **`fleet_scale_test.go`** (4): `TestFleetTUIScale18And100`, `TestFleetTUIDeploymentReview100OpsAndLargeDiff`, `TestFleet100RowNavigationPerformance`, `BenchmarkFleet100NavigationAndRender`
- **`fleet_test.go`** (8): `TestInstallHubFourEntriesAndWrap`, `TestFleetLazyNoReadWhenNeverOpened`, `TestFleetOverviewRendering`, `TestFleetProfilesSearchAndFilter`, `TestFleetProfilesDetailCard`, `TestFleetResponsiveSizes`, `TestFleetContextContinuity`, `TestFleetStaleGenerationDiscarded`
- **`fleet_write_test.go`** (11): `TestFleetDeploymentReviewPlanCountsAndDiff`, `TestFleetApplyConfirmationDefaultsToNoAndDispatchesOnce`, `TestFleetApplyBlocksDirtyDraftAndConflicts`, `TestFleetApplyResultRecordsRealOutcomeAndRefreshes`, `TestFleetLiveGateHidesApplyAndRestore`, `TestFleetApplyFixtureHomeWritesThroughHost`, `TestFleetApplyPendingIgnoresNavigationResizeAndRepeat`, `TestFleetApplyConfirmationCannotConfirmUnseenAtSmallSize`, `TestFleetSingleSlotCrossKindRefusal`, `TestFleetAppUsesProductionAdaptersWithoutStartupReads`, `TestFleetAppPreviewOnlyCannotDispatch`
- **`guide_test.go`** (7): `TestGuidePresentationOnlySourceAudit`, `TestGuideKey5AndKeyGAndHelpModal`, `TestGuideRoutesAndEscBReturnOrigin`, `TestGuideReturnResetOnOrdinaryNavigationAndNonGuideUnchanged`, `TestInstallResetJourneyAndGuideCompletedResultReset`, `TestGuidePlainASCIIAndNarrowLayoutBounds`, `TestRouteScrollFiveRoutesIndependent`
- **`install_test.go`** (15): `TestInstallCancelAndBack`, `TestInstallNoClient`, `TestInstallInvalidPreset`, `TestInstallChangedSettingsInvalidatesConfirmation`, `TestInstallFocusAndResizeRetainsSafeNo`, `TestInstallPreviewAndApplyCallCounts`, `TestInstallDoubleEnter`, `TestInstallStaleAsyncResult`, `TestInstallExactPreviewPaths`, `TestInstallNoWritesInPreview`, `TestInstallResultStates`, `TestInstallPreExecutionPolicyDisplayed`, `TestInstallPlainModeUnstyledAndBoundsSafe`, `TestInstallOpenCodeProfilesSeparate`, `TestInstallScrollingAndRestart`
- **`integration_b4b5_test.go`** (12): `TestInstallHubNavigation`, `TestInstallPreviewConfirmAndApply`, `TestRestoreJourneyAppliesThroughHost`, `TestSingleWriteRefusalAcrossScreens`, `TestCtrlCDeferredDuringWrite`, `TestRefusalKeepsConfirmationToken`, `TestNeedsPreviewReturnsToPreview`, `TestUpdateCheckOncePerSessionAndControls`, `TestOnboardingSkipFinishAndExplicitLink`, `TestOnboardingEscCancels`, `TestActivityRendersModelAndKeepsHelp`, `TestIntegratedLayoutsRemainBoundsSafe`
- **`integration_regression_test.go`** (7): `TestIntegratedASCIIUnicodePathBounds`, `TestIntegratedFooterAlwaysOffersQuit`, `TestIntegratedContextualFooterAndScrollPersistence`, `TestIntegratedInstallCommandHintsUseActualFlags`, `TestIntegratedScrollUpImmediatelyAfterEnd`, `TestIntegratedLastCatalogSelectionIsVisible`, `TestIntegratedHomePreservesInventoryNotice`
- **`layout_b2_test.go`** (3): `TestB2ReadableProseWordBoundaries`, `TestB2CategoryFocusAndVisibleControl`, `TestB2ContentGuttersSeparatePanels`
- **`library_help_test.go`** (7): `TestLibrary_RowFormatting`, `TestLibrary_TaskSearchInFilterValue`, `TestLibrary_RouterRecommendationAndStatePreservation`, `TestLibrary_DetailCardSectionsAndUnknownFallbacks`, `TestLibrary_DetailHierarchyKeepsPathsInAdvancedFiles`, `TestActivity_StructuredFactsNoProseScraping`, `TestActivity_HelpSectionsSelectionAndOrganization`
- **`mcp_a_hub_regression_test.go`** (2): `TestMCPAHubRegressionEntries`, `TestMCPAHubRegressionFleetCursorAndKeys`
- **`mcp_test.go`** (13): `TestMCPOverviewCoverage`, `TestMCPHubNavigationAndEsc`, `TestMCPSelectValidationAndAuthToggles`, `TestMCPPreviewTokenAndGeneration`, `TestMCPConfirmDefaultNoAndTinyGeometry`, `TestMCPDiagnosticsAndRecovery`, `TestOpsHostMCPPairwiseBusyRefusal`, `TestOpsHostMCPHeadersAndSource`, `TestMCPDefaultAdapterPurity`, `TestMCPHelpSectionTopic`, `TestFleetHelpLastSection`, `TestMCPWriteOutcomeHandling`, `TestMCPQuitFromDiagnosticsHeadless`
- **`modal_t19_test.go`** (9): `TestT19GeometryAndModes`, `TestT19CenteredBoxGeometry`, `TestT19FullscreenFallbackReachesAllContent`, `TestT19UnicodeBoundsAndASCII`, `TestT19BackdropIsANSIAndOSCSafe`, `TestT19FocusAndScrollRestoration`, `TestT19ResizeHelpAndBackgroundNeverConfirm`, `TestT19CancelAndStalePreviewKeepSafety`, `TestT19BusyPolicyIsNeverCovered`
- **`onboarding_t34_test.go`** (6): `TestT34SetupSummaryUsesCurrentChoices`, `TestT34SkipReopenBackPreserveChoicesAndConsent`, `TestT34ReopenFromHomePreservesInstallChoices`, `TestT34PostSetupOffersCopyablePrompt`, `TestT34PlainNarrowAndNoFuturePlaceholder`, `TestT34QuickStartHasNoAutomationPath`
- **`onboarding_test.go`** (11): `TestOnboardingStepOrderAndExplicitFinish`, `TestOnboardingBackIsSafe`, `TestOnboardingSkipAndCancelAreSafe`, `TestOnboardingPromptAndLinksAreCopyable`, `TestOnboardingBoundsAtBreakpoints`, `TestOnboardingTinyWidthsDoNotPanic`, `TestOnboardingPlainAndASCIIHaveNoHyperlinks`, `TestOnboardingOSC8OnlyWhenColorAndUnicode`, `TestOnboardingLinksOpenOnlyOnExplicitKey`, `TestOnboardingSummaryFromCatalog`, `TestOnboardingDoesNotWriteOrAutomate`
- **`opener_test.go`** (11): `TestOpenerFixedHTTPSLinks`, `TestOpenerRejectsArbitraryDestinations`, `TestOpenerLinuxResolvesOnceAndLaunchesArgumentVector`, `TestOpenerDarwinUsesFixedOpen`, `TestOpenerMissingProgramRetainsURL`, `TestOpenerRelativePathIsRefused`, `TestOpenerSSHSessionDoesNotLaunch`, `TestOpenerHeadlessLinuxDoesNotLaunch`, `TestOpenerLaunchFailureRetainsURL`, `TestOpenerUnsupportedPlatformRetainsURL`, `TestHyperlinkPlainAndOSC8`
- **`ops_host_fleet_test.go`** (7): `TestFleetWriteKindVerbs`, `TestFleetWriteRequestValidation`, `TestFleetWriteTranscriptIdentity`, `TestFleetWriteSourceFallsBackToConfigPath`, `TestFleetWriteNilSeamRefuses`, `TestFleetWriteSharesSingleSlotBothWays`, `TestFleetWriteCompleteRequiresMatchingIdentity`
- **`ops_host_test.go`** (17): `TestOpsHostRunsOnlyWhenCommandRunsAndCompletesOnce`, `TestOpsHostRefusesSecondConcurrentWriteWithoutQueueing`, `TestOpsHostRejectsInvalidRequestsBeforeClaimingSlot`, `TestOpsHostLogOpenFailureDoesNotWrite`, `TestOpsHostReportsLogFlushFailure`, `TestOpsHostDefaultFileLogWritesPrivateTranscript`, `TestOpsHostProductionEngineInstallsNoOpThenReplaces`, `TestOpsHostDispatchesRestoreThroughTheSameSingleSlot`, `TestSummarizeWriteResultStates`, `TestAppIdleQuitCancelsReadsOnce`, `TestAppIdleSignalQuitsWithTerminalCleanupPolicy`, `TestAppBusyIgnoresQuitEscapeAndNavigation`, `TestAppCtrlCAndRepeatedSignalsDeferExitUntilResult`, `TestAppStaleWriteResultsAreIgnored`, `TestAppRefusesDoubleSubmitWithoutCallingEngine`, `TestAppInitialWriteStartsBusyBeforeProgramRun`, `TestAppBusyFooterReplacesQuitHintAndPolicyIsShownBeforeExecution`
- **`preferences_t30_test.go`** (5): `TestT30HelpSessionPreferencesReachable`, `TestT30RestrainedHeaderStatusCluster`, `TestT30NeutralNoticeRoleContrast`, `TestActivityPreferencesReportActualSessionState`, `TestAppActivityPreferencesFollowRootToggles`
- **`presentation_escaping_test.go`** (4): `TestInstallPathEscapingIsReversibleAndRequestExact`, `TestInstallRuntimeASCIIToggleEscapesData`, `TestCatalogNoticesAndRootNoticeEscaping`, `TestRestorePathEscapingIsReversible`
- **`presentation_t18_test.go`** (8): `TestT18BreadcrumbDerivation`, `TestT18StatusClusterFactual`, `TestT18StageAccurateFooter`, `TestT18StableMarkerAndGutterGeometry`, `TestT18BoundsAtAllBreakpoints`, `TestT18SafeGlyphsAndNoEmoji`, `TestT18LongWideAndCombiningData`, `TestT18RouteTabPills`
- **`presentation_test.go`** (22): `TestPresentationParagraphWrapsOnWordBoundaries`, `TestPresentationParagraphDoesNotSplitWordsThatFit`, `TestPresentationParagraphIndentsEveryLine`, `TestPresentationParagraphIndentIsBoundedAndNotDuplicated`, `TestPresentationParagraphPreservesBlankBoundaries`, `TestPresentationParagraphHardwrapsSingleLongTokenCellAware`, `TestPresentationParagraphHardwrapKeepsGraphemeClusters`, `TestPresentationParagraphUnicodeCellWidths`, `TestPresentationParagraphANSIWidthsAndEscapes`, `TestPresentationParagraphKeepsOnlyIndentLeadingSpace`, `TestPresentationParagraphSmallWidthStaysWithinWidth`, `TestPresentationSectionSpacingAndIndent`, `TestPresentationSectionDoesNotDoubleBlanksOrIndents`, `TestPresentationSectionWrapsWithinWidth`, `TestPresentationSectionEmptyBodyHasNoSeparatorBlank`, `TestHomeOverviewDerivesCountsFromCatalog`, `TestHomeOverviewMarksAbsentOptionalContent`, `TestHomeOverviewEmptyCatalogIsReadable`, `TestHomeOverviewRespectsWidth`, `TestHomeOverviewLongPathsFullyRecoverable`, `TestHomeOverviewDeterministicAndReadOnly`, `TestHomeOverviewUnknownMetadataIsNotFabricated`
- **`recovery_test.go`** (11): `TestRecovery_BreakdownFreshInstallVsReplacements`, `TestRecovery_LegacyBackupHandling`, `TestRecovery_EmptyFormatTwoBackupIsNotEmptyWording`, `TestRecovery_EmptyLegacyEntryStaysUnknown`, `TestRecovery_UnusableBackupRendersUnknownNotZero`, `TestRecovery_LogAssociationIsNeverGuessedFromFilenames`, `TestRecovery_ManifestAndLogKeysInDetails`, `TestRecovery_StaleManifestReadIsDiscarded`, `TestReadBackupManifestContentRefusesUnsafeFiles`, `TestReadBackupManifestContentRejectsSwapAfterValidation`, `TestRecovery_DirectHomeNavigation`
- **`responsive_t35_test.go`** (8): `TestT35BreakpointsAndReviewGeometries`, `TestT35StateCoverage`, `TestT35UnicodeAndLongPaths`, `TestT35FooterLastItemAndFocus`, `TestT35ConfirmationFitAndFallback`, `TestT35ConfirmationResizeKeepsSafeNo`, `TestT35RepresentativeScreenSnapshots`, `TestT35TinyTerminalSafety`
- **`restore_test.go`** (13): `TestRestoreEnterOpensDetailsOnlyAndWritesNothing`, `TestRestoreFullJourneyAppliesOnce`, `TestRestoreNoChangeIsHonest`, `TestRestoreSubsetBackupLeavesUnselectedClientUntouched`, `TestRestorePreviewShowsExactOverwriteAndRemovalPaths`, `TestRestoreDriftInvalidatesConsent`, `TestRestoreCorruptStaleAbsentAndRefusedStates`, `TestRestoreStaleResultAndBackAreSafe`, `TestRestoreCancelPathsAreSafe`, `TestRestoreEmptyRootIsValidEmptyState`, `TestRestoreOrderingAndNoFilesystemWorkInUpdate`, `TestClassifyRestoreOutcome`, `TestRestorePlainAndBoundsSafe`
- **`safety_review_test.go`** (9): `TestConfirmationBelowUsableGeometryCannotSubmit`, `TestCatalogStaleFilterResultsCannotRestoreOlderRows`, `TestCatalogStaleFilterResultCannotRestoreAfterRefresh`, `TestActivityReadPreservesNewerSectionAndSelection`, `TestActivityStaleReadCannotOverwriteNewerRead`, `TestNoOpPreviewRefreshesInventoryWithoutWriting`, `TestRouterRecommendationOverrideDoesNotSurviveEsc`, `TestBackupDetailsOwnsLowercaseMThroughApp`, `TestCatalogSetFilterRejectsStaleQueryEnvelope`
- **`signals_test.go`** (6): `TestShutdownSignalsMatchPlatformContract`, `TestSignalHandlerForwardsEverySignalAndRepeat`, `TestSignalHandlerStopIsIdempotentAndStopsDelivery`, `TestSignalHandlerWithoutSendDoesNotPanic`, `TestTerminalReleaseIsIdempotentAndDisarmable`, `TestWritePolicyDocumentsResidualRisk`
- **`theme_adaptive_test.go`** (8): `TestThemeAgenticDarkActualPairsMeetWCAG`, `TestAdaptiveThemeDarkPaletteRolesAndTokens`, `TestThemeReferenceProfilesActualPairs`, `TestAppColorProfileMessageUsesReferenceMappings`, `TestAppBackgroundColorAdaptation`, `TestAppLateBackgroundColorMsgPreservesState`, `TestAppColorOffNoQueryNoPaint`, `TestScreenPaletteAuditNoLocalRawColors`
- **`update_state_test.go`** (10): `TestUpdateState_InitialState`, `TestUpdateState_AutomaticCheckOnceAcrossRevisits`, `TestUpdateState_DisabledByEnvironment`, `TestUpdateState_DisabledExplicitly`, `TestUpdateState_RetryCheck`, `TestUpdateState_StaleResultSuppression`, `TestUpdateState_CancellationOnDisable`, `TestUpdateState_StatusTransitions`, `TestUpdateState_AppopsMockTransportIntegration`, `TestUpdateState_RenderStatusThemes`
- **`wrapping_test.go`** (1): `TestWrapPathPreservesGraphemeClusters`


## Appendix B — Python test case names

Generated from `unittest` classes and `def test_…` methods. Each method inherits its file's scope, type, feature and selection from the tables above. Gated files list their tests even when the gate would skip them.

#### Python harness — `tests/harness` (21 files, 183 cases)

- **`tests/harness/test_appops_core_selection.py`** (9): `SelectorParityTests`: `test_taskfile_test_and_vet_select_all_eleven_packages`, `test_harness_contract_required_vet_and_test_select_all_eleven_packages`, `test_core_shell_script_lists_required_directories_and_selection`, `test_lefthook_inherits_core_selection_without_its_own_package_list`, `test_all_maintained_selectors_agree_on_the_same_package_list`, `test_no_excluded_or_aggregate_selection_is_introduced`, `test_existing_seven_packages_remain_selected`; `PackageCountGuidanceTests`: `test_agents_md_count_and_list_guidance_is_current`, `test_contributing_count_and_list_guidance_is_current`
- **`tests/harness/test_connected.py`** (3): `ConnectedOfflineTests`: `test_result_receipt_freshness_authority_and_location_reconciliation`, `test_actual_interruption_receipts_stay_partial_after_reload`, `test_executed_defect_and_corrected_checks_do_not_waive_open_findings`
- **`tests/harness/test_core_workflow.py`** (11): `CoreWorkflowTextTests`: `test_existing_core_rules_are_characterized`, `test_each_seeded_rule_removal_is_detected`, `test_softened_authority_clauses_are_detected`, `test_core_reference_targets_exist_and_missing_target_fails`, `test_scenario_manifest_covers_core_cases_without_model_claims`, `test_missing_case_unknown_rule_and_invented_pass_are_rejected`, `test_incomplete_request_separates_facts_assumptions_and_scope`, `test_completion_summary_prioritizes_behavior_and_risk_review`, `test_getting_started_has_three_bounded_worked_paths`, `test_public_entrypoints_link_the_worked_paths`, `test_security_reporting_does_not_invent_a_private_channel`
- **`tests/harness/test_independent_review.py`** (9): `IndependentReviewExampleTests`: `test_example_is_inactive_and_has_no_providers_or_commands`, `test_top_level_policies_deny_providers_integrations_and_execution`, `test_exactly_one_coordinator_and_one_reviewer_profile`, `test_both_profiles_bind_model_and_variant_explicitly_and_bound_steps`, `test_profiles_start_with_deny_and_contain_no_template_markers`, `test_only_named_skill_and_reads_are_allowed_for_both_profiles`, `test_only_the_coordinator_may_launch_exactly_the_reviewer`, `test_sensitive_reads_are_denied_for_both_profiles`, `test_no_active_default_or_lockdown_template_changed`
- **`tests/harness/test_independent_review_dispatch.py`** (1): `DispatchEnvironmentTests`: `test_pwd_matches_project_instead_of_inherited_ssh_home`
- **`tests/harness/test_independent_review_docs.py`** (6): `IndependentReviewDocContractTests`: `test_reference_keeps_every_required_rule`, `test_pointers_reach_the_reference`, `test_runbook_distinguishes_agent_selection_from_skill_loading`, `test_runbook_keeps_native_symlink_failure_open`, `test_seeded_deletion_is_detected`, `test_seeded_blocked_route_softening_is_detected`
- **`tests/harness/test_independent_review_fixtures.py`** (13): `ReconciliationFixtureTests`: `test_each_fixture_reconciles_to_its_independent_expectation`, `test_every_rule_changes_some_fixture_outcome`, `test_blocked_or_stale_outcomes_never_say_ready`; `SchedulingContractTests`: `test_one_explicit_request_on_new_candidate_launches`, `test_active_reviewer_blocks_a_second_launch`, `test_implicit_or_automatic_retry_never_launches`, `test_identical_completed_review_is_not_duplicated`, `test_new_explicit_request_after_interruption_is_allowed_and_explicit`, `test_changed_candidate_after_completed_review_needs_a_new_explicit_request`; `FreshnessFixtureTests`: `test_each_relevant_change_invalidates_identity_and_revert_restores_it`, `test_unrelated_change_outside_scope_keeps_identity`, `test_consumer_marks_review_stale_after_a_real_relevant_change`, `test_workspace_without_git_is_missing_evidence_not_a_pass`
- **`tests/harness/test_offline_transport.py`** (21): `BudgetTests`: `test_positive_reservation_is_durable_and_never_refunded`, `test_wrong_model_paths_and_unreviewed_body_fields_never_reserve`, `test_duplicate_keys_nonfinite_and_invalid_caps_fail_before_reservation`, `test_requests_from_auxiliary_and_new_sessions_share_one_counter`, `test_concurrent_attempts_cannot_overspend_reserved_output`, `test_input_and_output_aggregate_limits_are_independent`, `test_expiry_and_cancellation_stop_new_dispatch`, `test_journal_cannot_be_reset_or_reopened_on_restart`, `test_unknown_real_token_accounting_is_not_renamed_to_bytes`, `test_one_native_alternative_cap_is_supported_but_two_caps_are_ambiguous`, `test_only_literal_loopback_upstream_and_no_path_override`, `test_deleted_journal_stops_before_dispatch_and_errors_do_not_echo_input`; `HTTPTests`: `test_positive_forwarding_drops_auth_headers_and_shares_request_limit`, `test_malformed_or_wrong_model_never_reaches_fake_endpoint`, `test_redirect_or_http_error_is_not_retried_or_followed`, `test_oversized_response_spends_reservation_and_stops`, `test_absolute_deadline_bounds_slow_upstream`, `test_buffered_sse_is_bounded_and_never_claimed_as_native_compaction`, `test_ambiguous_http_framing_and_upgrade_never_forward`, `test_unknown_http_method_does_not_echo_attacker_controlled_text`, `test_trickle_response_cannot_extend_absolute_deadline`
- **`tests/harness/test_opencode.py`** (7): `TemplateTests`: `test_no_implicit_provider_or_integration_access`, `test_commands_cannot_inherit_a_model_or_launch_children`, `test_profiles_are_bounded_and_start_with_deny`, `test_higher_authority_lockdown_has_no_allow_exception`, `test_usage_catalog_is_not_permission_to_run`; `ManualAdoptionExercise`: `test_preview_backup_noop_restore_and_new_file_removal`, `test_symlink_parent_symlink_and_special_file_are_not_regular`
- **`tests/harness/test_p1_driver.py`** (3): `DriverTests`: `test_policy_timeout_still_removes_exact_cid`, `test_bad_cid_is_not_removed`, `test_changed_source_is_recorded_as_failed_not_success`
- **`tests/harness/test_p1_network.py`** (5): `CourierTests`: `test_real_courier_process_dispatches_once_and_strips_private_material`, `test_redirect_is_spent_and_not_followed_or_retried`, `test_slow_courier_is_killed_and_cannot_dispatch_again`, `test_courier_entry_cannot_run_without_allocation_and_tls_is_verified`, `test_http_framing_denials_never_reach_loopback_upstream`
- **`tests/harness/test_p1_transport.py`** (9): `PilotTests`: `test_one_canonical_text_request_and_only_curated_observations`, `test_wrong_model_auth_prompt_caps_or_body_never_dispatch`, `test_all_parallel_replays_share_one_spent_attempt`, `test_native_cli_json_quoted_literal_is_normalized_without_extra_text`, `test_quoted_literal_with_extra_instructions_is_never_admitted`, `test_restart_or_missing_journal_cannot_reset_allocation`, `test_expired_or_cancelled_run_never_dispatches`, `test_upstream_redirect_or_failure_is_spent_without_retry`, `test_missing_usage_or_excess_output_does_not_manufacture_success`
- **`tests/harness/test_receipts.py`** (18): `ReceiptTests`: `test_valid_direct_fix_does_not_invent_feature_or_mutate_project`, `test_snapshot_detects_index_worktree_untracked_deleted_and_mode`, `test_scope_is_explicit_and_unsafe_files_do_not_get_followed`, `test_git_identity_does_not_inherit_external_index_or_global_configuration`, `test_scoped_directory_cannot_traverse_a_symlink_ancestor`, `test_stale_and_unknown_identity_never_establish_reusable_pass`, `test_schema_unknown_reasons_and_semantic_consistency_fail_closed`, `test_required_coverage_cannot_be_forged_by_selection_label`, `test_unobserved_billing_session_model_and_paid_permission_cannot_be_invented`, `test_independently_observed_synthetic_native_values_are_not_blanket_rejected`, `test_evidence_missing_tampered_public_or_symlink_is_not_a_pass`, `test_dotdot_evidence_and_export_paths_cannot_escape_boundaries`, `test_ready_requires_coverage_fresh_evidence_and_no_unresolved_findings`, `test_export_redacts_before_write_and_refuses_unapproved_fields_and_paths`, `test_saved_receipt_survives_reload_but_changed_code_tasks_or_authority_block_reuse`, `test_workspace_move_requires_reconciliation_not_automatic_replay`, `test_cancelled_or_incomplete_result_cannot_be_promoted_to_completed_or_ready`, `test_check_status_cannot_be_relabelled_as_different_execution_outcome`
- **`tests/harness/test_recovery.py`** (2): `RecoveryTests`: `test_owned_cancellation_preserves_partial_files_and_unrelated_process`, `test_deadline_persists_incomplete_and_never_replays_remaining_check`
- **`tests/harness/test_research_workflow.py`** (22): `ResearchWorkflowContractTests`: `test_research_skill_states_the_contract`, `test_preserved_008_clauses_are_verbatim`, `test_handoff_fields_appear_in_contract_order`, `test_claim_labels_and_privacy_boundaries_are_explicit`, `test_effort_levels_are_a_heuristic_not_a_permission`, `test_research_md_is_conditional_only`, `test_description_is_trigger_focused_and_not_generic_research`, `test_reference_is_linked_from_the_research_skill`, `test_reference_covers_capability_and_claim_discipline`, `test_reference_does_not_claim_installed_providers`, `test_stage_pointers_link_to_the_canonical_skill_and_state_their_boundary`, `test_stage_pointers_do_not_duplicate_the_shared_contract`, `test_policy_and_librarian_carry_precedence_privacy_and_untrusted_rules`, `test_librarian_keeps_its_pinned_model_and_denied_permissions`, `test_each_required_rule_removal_is_detected`, `test_softened_privacy_untrusted_snippet_approval_and_stage_return_are_detected`, `test_verification_docs_are_not_tests_softening_is_detected`, `test_mocked_private_query_is_not_dispatched_or_redacted_at_runtime`, `test_scenario_manifest_covers_required_cases_without_model_claims`, `test_manifest_mutations_are_rejected`, `test_handoff_cases_validate_the_seven_field_contract`, `test_trigger_cases_cover_positive_and_negative_prompts`
- **`tests/harness/test_s0_native.py`** (1): `NativeRequestTests`: `test_pinned_runtime_sends_only_to_fake_gate_and_uses_fake_output`
- **`tests/harness/test_supported_core_go.py`** (8): `SupportedCoreSelectionTests`: `test_exact_test_and_vet_selection`, `test_offline_environment_and_inherited_filters_cleared`, `test_nonzero_failure_propagation_from_go`, `test_invalid_mode_or_extra_args_do_not_run_go`, `test_missing_core_directory_or_file_fails`, `test_task_entry_points_forward_cache`, `test_default_task_and_core_alias_parity_and_goflags_neutralization`; `CoreGateParityTests`: `test_checks_contract_taskfile_and_script_core_parity`
- **`tests/harness/test_taskfile.py`** (11): `TaskfileTests`: `test_default_lists_tasks_without_executing_them`, `test_tag_requires_version_without_creating_tag`, `test_trusted_verification_requires_reviewed_base`, `test_verify_passes_output_and_offline_environment_to_runner`, `test_command_line_cache_overrides_inherited_cache`, `test_task_test_and_vet_sanitize_environment_and_select_core`, `test_experimental_taskfile_separation`; `TrustedPathsTests`: `test_ci_uses_only_reviewed_base_taskfile`, `test_taskfile_changes_require_review`, `test_trusted_base_rejects_changed_local_recipe_contract`, `test_goreleaser_uses_default_inactive_core_profile`
- **`tests/harness/test_tokenizer_source.py`** (4): `TokenizerSourceTests`: `test_no_remote_code_or_silent_truncation_is_loaded`, `test_special_token_ids_match_independent_public_data`, `test_known_text_and_long_input_do_not_use_character_estimates`, `test_plain_chat_example_has_explicit_local_not_hosted_identity`
- **`tests/harness/test_trusted_ci.py`** (14): `TrustedPolicyTests`: `test_canonical_required_commands_are_retained`, `test_unadopted_contract_rejection`, `test_probes_presence_and_collision_checks`, `test_removed_replaced_or_relabelled_required_gate_is_rejected`, `test_nonroot_cwd_is_blocked_instead_of_silently_reinterpreted`, `test_nonzero_transport_or_missing_mismatched_result_cannot_be_completed`, `test_test_and_gate_changes_are_review_inputs_not_silently_approved`, `test_revision_observation_is_actual_root_head_not_an_export_or_parent_guess`, `test_staging_includes_untracked_bytes_but_no_git_cache_or_backups`, `test_staging_rejects_links_and_special_files_without_following_them`, `test_staging_refuses_credential_paths_instead_of_copying_local_secrets`, `test_candidate_container_gets_only_its_workspace_and_private_check_cache`, `test_gate_does_not_need_candidate_receipts_to_determine_check_outcome`, `test_whitespace_checks_source_patch_not_generated_build_artifacts`
- **`tests/harness/test_trusted_ci_container.py`** (6): `TrustedContainerTests`: `test_bootstrap_and_candidate_contract_weakening_block_before_execution`, `test_local_and_ci_entries_agree_and_candidate_boundary_is_observed`, `test_empty_task_deleted_coverage_and_forged_receipt_cannot_hide_noop_defect`, `test_deleted_coverage_and_activation_mutant_fails_via_boundary_probe`, `test_cleanup_stops_only_cid_owned_container`, `test_interrupted_gate_cleans_owned_containers_and_retains_partial_candidate`

#### Python TUI acceptance — `tests/tui` (5 files, 39 cases)

- **`tests/tui/test_fleet_lifecycle.py`** (4): `FleetLifecycleTest`: `test_fleet_overview_is_read_only_and_never_claims_live`, `test_resize_keeps_surface_alive_and_back_reachable`, `test_quit_from_tiny_fleet_view_exits_cleanly`, `test_filter_field_treats_navigation_and_quit_keys_as_text`
- **`tests/tui/test_launcher_lifecycle.py`** (19): `LauncherLifecycleTest`: `test_explicit_tui_launches_the_verified_companion`, `test_bare_asdx_opens_the_tui`, `test_bare_asdx_core_keeps_the_preview`, `test_bare_legacy_cli_keeps_the_preview_and_warns`, `test_legacy_cli_tui_reaches_the_tui`, `test_bare_legacy_tui_wrapper_opens_the_tui`, `test_legacy_tui_protocol_info_is_pure_json`, `test_legacy_cli_version_identifies_both_names`, `test_missing_sibling_fails_closed_without_preview`, `test_nonterminal_stdin_fails_closed_with_core_hint`, `test_nonterminal_explicit_tui_fails_closed`, `test_parent_propagates_child_failure_and_forwards_repo`, `test_parent_relays_sigint_and_child_restores_terminal`, `test_parent_relays_sigterm_and_child_restores_terminal`, `test_parent_relays_sighup_and_child_restores_terminal`, `test_inflight_sigint_defers_exit_through_the_parent`, `test_inflight_repeated_sigterm_defers_exit_through_the_parent`, `test_inflight_sighup_defers_exit_through_the_parent`, `test_forced_interactive_parent_refuses_to_launch`
- **`tests/tui/test_mcp_lifecycle.py`** (3): `MCPLifecycleTest`: `test_mcp_overview_lists_three_defaults_and_writes_nothing`, `test_mcp_select_preview_diagnostics_and_back`, `test_mcp_resize_cannot_confirm_and_back_stays_reachable`
- **`tests/tui/test_readonly_catalog.py`** (3): `ReadOnlyCatalogTest`: `test_explore_lists_the_real_shipped_inventory`, `test_explore_categories_and_curated_detail`, `test_browsing_session_writes_nothing_into_the_home`
- **`tests/tui/test_terminal_lifecycle.py`** (10): `TerminalLifecycleTest`: `test_quit_restores_terminal_and_leaves_alt_screen`, `test_early_error_exits_nonzero_without_touching_terminal`, `test_nonterminal_input_does_not_enter_alternate_screen`, `test_forced_interactive_script_mode_does_not_enter_alternate_screen`, `test_idle_sigint_restores_terminal_and_leaves_alt_screen`, `test_idle_sigterm_restores_terminal_and_leaves_alt_screen`, `test_idle_sighup_restores_terminal_and_leaves_alt_screen`, `test_inflight_sigint_defers_exit_until_write_completes`, `test_inflight_repeated_sigterm_defers_exit_until_write_completes`, `test_inflight_sighup_defers_exit_until_write_completes`


## Appendix C — Supporting fixture and oracle helpers

These files support the cases above but are not `unittest` cases or Go tests:

- `tests/harness/receipt_tools.py` — independent reference consumer/exporter for receipt and result JSON.
- `tests/harness/go_snapshot_fixture.py` — independently derives the Go snapshot golden digests.
- `tests/harness/independent_review_contract.py` — reference consumer for independent-review reports.
- `tests/harness/native_independent_review.py`, `native_independent_review_dispatch.py`, `native_opencode.py` — offline native probe bodies.
- `tests/harness/offline_transport.py`, `p1_transport.py`, `p1_native.py`, `s0_worker.py`, `s0-fixtures.py`, `p1-preflight.py` — S0/P1 transport, worker and preflight helpers.
- `tests/harness/trusted_ci.py`, `diagnosis_fixtures.py`, `validate_contracts.py` — trusted-gate, diagnosis and JSON-schema/contract validators.
- `harness/checks.json`, `tests/harness/contracts.json`, `tests/harness/*-scenarios.json` — declarative check/scenario manifests.
- `tests/qa-container/Dockerfile` — pinned Feature 005 QA image definition used by the historical drivers.

## Scope of this report

This report was generated from source at the commit above. Re-run the extraction after test files, build tags or `harness/checks.json` change. It is not a substitute for `task test`, `task vet`, `task verify`, the trusted-CI gate or a feature's `spec.md` approval section.
