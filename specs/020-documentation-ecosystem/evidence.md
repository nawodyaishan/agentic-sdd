# Feature 020 — Implementation truth, documentation audit and migration

**Snapshot:** clean starting checkout `2fc31e9f9c4d019e2b9f3b4f004978028808c74b`, 2026-11-14. The test inventory describes its earlier `f559872` snapshot. Refresh changed claims at implementation start.
**Status:** source-backed planning evidence, not release certification or feature acceptance.

## r2 risk-resolution observations

The follow-up request authorized DeepSeek Flash analysis for each remaining risk area, still within specification/planning scope. Seven additional bounded read-only workers investigated release evidence, Pages activation, platform/link/dependencies, CI trust, testing/coverage, usability/search/accessibility and migration/version/TUI. All used `opencode-go/deepseek-v4.1-flash`, no fallback. Their findings were proposals until reconciled by the main author; no worker proposal automatically became an architectural verdict.

### Public release: existence verified, executable contents not verified

The main author independently repeated these **unauthenticated, read-only** public probes on 2026-11-14:

| Primary source | Observation | What it does not prove |
|---|---|---|
| [Latest release API](https://api.github.com/repos/nawodyaishan/agentic-sdd/releases/latest) / [v0.11.0](https://github.com/nawodyaishan/agentic-sdd/releases/tag/v0.11.0) | HTTP 200; non-draft/non-prerelease v0.11.0, published `2026-11-13T19:36:14Z` | Passing local install, human acceptance or archive member contents |
| [Tag ref](https://api.github.com/repos/nawodyaishan/agentic-sdd/git/ref/tags/v0.11.0) / [annotated tag object](https://api.github.com/repos/nawodyaishan/agentic-sdd/git/tags/e9cf0588a4041838242e2f113eaa87bc3d40962a) | Annotated tag peels to commit `2fc31e9f9c4d019e2b9f3b4f004978028808c74b`, this investigation's checkout baseline | Release binaries were executed or every source capability is enabled in them |
| Release assets in the API | `agentic-sdd_0.11.0_darwin_all.tar.gz` (32,521,165 bytes) and `checksums.txt`; archive metadata SHA-256 `42e5d832b4427f1390b59322d1641ec953bcdce7f2c033f5af6f17d60f495f82` | Archive membership, executable protocol identity or macOS quarantine/upgrade success |
| [Published tap cask](https://raw.githubusercontent.com/nawodyaishan/homebrew-tap/main/Casks/agentic-sdd.rb) | v0.11.0, matching archive URL/hash, `binary` declarations for `asdx`, `asdx-core`, `agentic-sdd`, `agentic-sdd-tui` | Those executables were extracted, identity-checked or installed here |

This **supersedes r1's overly broad “source newer than verified release artifacts” concern**. Release existence/tag/cask metadata are verified; archive/runtime/live-install evidence remains unverified. Public metadata cannot change Feature 019's approval or human-review state. No release archive was downloaded or executed.

Release-sync builds consume a reviewed, revision-pinned release metadata snapshot; routine PR builds never query “latest.” Optional external refresh records its timestamp and fails visibly if unavailable, without inventing support or blocking unrelated offline checks.

### Pages: currently unavailable, history/settings not inferred

| Primary source | Observation, 2026-11-14 | Interpretation limit |
|---|---|---|
| [Repository API](https://api.github.com/repos/nawodyaishan/agentic-sdd) | HTTP 200; public repository, `default_branch: main`, `has_pages: false`, empty `homepage` | Current reported flag, not its historical cause or all administrative settings |
| [Pages API](https://api.github.com/repos/nawodyaishan/agentic-sdd/pages) | HTTP 404 | Alone would not establish disabled settings or authorization state |
| [Proposed project URL](https://nawodyaishan.github.io/agentic-sdd/) | HTTP 404 | No live site at that URL now; not proof it never existed or no inbound web links exist |

No custom domain is assumed. Owner-only Pages source, Actions policy, protected-environment capabilities and retention remain unverified. T10 checks them through documented read-only owner/admin views before any separately authorized activation. Do not paste tokens, environment contents or account metadata into public evidence.

Default launch creates **no speculative web redirects**; add one only for an evidenced prior published URL. GitHub guide compatibility stubs are still required. A current 404 is not permission to discard historical links.

### Research recommendations reconciled

- **CI:** a candidate-defined advisory `pull_request` lane is economical but forgeable. r2 chooses a separate **reviewed-base docs launcher** for authoritative PR evidence, preserving the product gate unchanged. Initial trusted adoption/dependency refresh is explicit; no candidate script runs on the host or while dependency fetching has network access.
- **Freshness:** replace commit-ancestry gating with hashes of explicit reviewed source inputs. Build identity, docs applicability and review evidence are separate. No unshallow Git-history dependency, automatic review stamping or claim that an ancestor commit proves accuracy.
- **Diagrams:** choose reviewable Mermaid sources and pre-rendered static PNG projections, no runtime diagram renderer or new browser dependency in routine CI. PNG is preferred over unsanitized active SVG; text alternatives remain mandatory.
- **Coverage:** no measured baseline/regeneration tool was found. Define an optional exact measurement contract and a test-initiative owner/drift disposition, not a new measurement project or invented percentage.
- **Usability/search:** concrete task cards and 20 fixed query expectations replace vague “walkthrough/search review.” Unavailable participants or failed critical queries cannot silently become successful acceptance.
- **TUI:** reuse fixed trusted links and URL fallback; explicitly test the nil-opener path and full URL visibility, without exposing refused arbitrary URLs as executable links.

## Method and confidence

Eight bounded parallel investigations covered existing docs, skills/spec history/fleet, platform research, audiences, CLI/TUI, architecture/MCP/execution, AI discovery, and testing/CI. All used catalog-confirmed `opencode-go/deepseek-v4.1-flash` (DeepSeek V4.1 Flash 1017), no fallback. The main author (GPT-6.1 Sol) reconciled consequential claims against source, command help and ledgers. No authentication/quota interrogation or provider benchmark was performed; successful dispatch proves this model route worked for these tasks, not future availability. Per-child or aggregate billing was not exposed; no cost figure is asserted.

CodeGraph supplied current source and call relationships; Markdown/configuration were read directly. Sources and test presence are not executed behavior. Contradictory worker conclusions were narrowed:

- `experiments/tui/` is the canonical nested TUI source, **not** proof that the whole TUI remains an unsupported spike.
- The build-time fleet-live marker gates **TUI fleet dispatch**; do not infer the separate CLI fleet apply is disabled by that marker.
- Not all commands pass through appops. CLI execution calls `internal/execution` directly; fleet plan calls `fleet.BuildPlan` directly. TUI/install/MCP use typed seams where implemented.
- Later accepted task rows in 013 and 017 outrank stale “awaiting review” overview prose. Do not rewrite owning records to make summaries agree.
- The ambient agent catalog/global routing table is not the shipped pack.

### Executable observations in this session

`GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off` with trace disabled:

| Observation | Result | Limit |
|---|---|---|
| `go run ./cmd/agentic-sdd --help` | exit 0; canonical `asdx-core`, default preview, fleet/MCP/execution families | Help only; no installation/provider behavior |
| `go build -o <temporary>/asdx-core ./cmd/agentic-sdd` | exit 0 | CLI build only; no release identity or TUI build |
| Built binary `--help`, `help orchestrate`, `fleet help`, `mcp help`, `run help`, `task help`, `review help` | all exit 0 | MCP family help includes write/restore omitted by top-level description |

Full local help captures: `/tmp/opencode/spec020-help-6a3v60wi/` (ephemeral, not required to understand this record). No real-home apply, runtime model/MCP call, OAuth, release or website build was performed. Documentation static-suite results are recorded in `tasks.md`, not inferred here.

## Verified capability-to-documentation inventory

**I** = implemented source and tests inspected/present, not rerun here; **E** = executable observation above; **G** = implemented but gated/partially verified; **X** = experimental/excluded; **H** = historical/planned. All public statements still need a distribution qualifier.

| Component | Implemented behavior / evidence | Existing documentation | Problem | Proposed action |
|---|---|---|---|---|
| CLI entrypoints (I/E) | `cmd/agentic-sdd/main.go:runEnv`, stdlib `flag`; preview default, explicit apply; `internal/buildinfo` names; `main_test.go`, `tui_test.go` | README; installer reference | Tables omit fleet/mcp/run/task/review; command-specific flags mistaken for globals | Full validated command inventory; short README; compatibility owner |
| TUI (I/G) | `experiments/tui/app.go:RouteGuide`, five routes; Guide/Help; real-terminal gates; `guide_test.go`, responsive/PTY suites | README; guide-flows; tui-architecture | Four-destination README; mixes numbered CLI guide, historical spike and current TUI | Distinct TUI reference and architecture; maintain source-path distinction |
| Core skills (I) | `skills-files/`; exact package test contract `internal/skillsync/package_test.go`; 18 canonical IDs | README; getting-started; skill files | Catalog/procedure repeated; installed catalog can omit available source skills | Generate facts from canonical assets, link normative instructions |
| Optional orchestration (I/G) | `opencode-pack/skills/agentic-sdd-orchestrate/`; manual bounded delegation; profile loading belongs to client | orchestration-guide; pack README | “Every way” claim exceeds coverage; may imply a dispatcher/security boundary | Shared optional how-to + lifecycle/trust explanation |
| Profiles/presets/adapters (I) | `opencode-pack/agents/` 14; `presets/free-first/` four; `adapters/` two excluded from ordinary pack install; `opencode_*_test.go` | README; routing.md; subagent-fleet-comparison study | routing.md CLI example says 12; local fleet not equal to shipped set | Source-generated installed catalog; explicitly separate adapter/fleet/local state |
| Model routing (I/G) | Frontmatter model pins; orchestrator inherits active session; `internal/appops/routing.go`, `routing_test.go`; written fallback policy | routing.md; orchestration-guide | Model availability, context/pricing and local routes volatile; no automatic fallback engine | Verified grammar/semantics; dated optional examples; no provider availability guarantee |
| Fleet core/CLI (I/G) | `internal/fleet/{parse,resolve,render,plan,apply,restore,doctor}.go`; strict schema v1, model-only rewrite, provenance/locks; fleet tests | worked guide in spec014; fleet-manager-tui | Useful current guide hidden in specs; CLI versus TUI write capability confused | Fleet how-to/reference; isolate plan-file writes, apply and live TUI marker |
| Fleet TUI (G) | `fleet_production.go`, `fleet_write.go`, `buildinfo.FleetLiveEnabled`; source save B4 not implemented/authorized; `fleet_*_test.go` | fleet-manager-tui; 015 acceptance packet | Preview-only is accurate for shipped default; existing plumbing is not authorization to enable live | Document default marker, staging, source-save/model-clear limits; do not “complete” B4 |
| MCP read/config transactions (I/G/E) | `cmd/.../mcp*.go`, `internal/skillsync/mcp*.go`, appops seam, TUI adapter; digest-gated apply/managed restore; tests | mcp-tools; default-integration study | Global help says read-only; configuration ≠ auth/connectivity; separate backup store underexplained | Current MCP guide/reference/security boundaries; note top-level help defect separately |
| Install/source/client selection (I) | `skillsync/source.go`, `clients.go`, `sync.go`; embed or explicit repo, supported-client order, TUI subset seam; package/selection tests | installer-reference; README | Duplicated destinations; subset service not a blanket CLI flag | Reference actual entrypoint flags and source precedence |
| Backup/restore (I) | `manifest.go`, `restore.go`, permissions and rollback tests; formats 1–3; fleet sidecar format 4 is separate; MCP separate manifest | installer-reference; repeated orchestration/README sections | Different stores/formats easily conflated; checksums not signatures | One recovery guide + file/format reference, cross-links |
| Execution workflows (I/G/E) | `run.go`, `task.go`, `review.go` → `internal/execution`; identity/locks/brief integrity/check evidence/review packages; 016/017 tests | execution-runs | Too much feature-state prose for users; “no network” can hide child-command permissions | Usage/reference/recovery with explicit inherited permissions and private argv/log warning |
| Teaching/research/debug/review (I/G) | Skill instructions; `test_core_workflow.py`, `test_research_workflow.py`; 018 instruction/fixture tests | getting-started; orchestration/research study | Instruction/fixture coverage can be mistaken for model behavior | Bounded examples with static/manual/model evidence labels |
| Updates/network/browser (I) | `appops/update.go`, `experiments/tui/opener.go`; fixed release endpoint; explicit trusted links; update/opener tests | README; guide-flows | CLI guide explicit update versus TUI session check are different | Subsystem network matrix and safe browser fallback documentation |
| Packaging/platforms (I/G) | `.goreleaser.yml`, release tests/workflow; public v0.11.0 tag/release/cask metadata at baseline commit | README; 013/019 ledgers; Snap report | Release metadata exists; archive members/runtime/live macOS install unverified; Linux package study not implementation | Version-specific compatibility page; distinguish metadata, archive inspection and live install |
| Tests/CI (I) | 11-package selector, explicit Python, nested TUI; trusted candidate gate | CONTRIBUTING; tests/harness README; test inventory | Counts ≠ coverage; no docs pipeline/measurement baseline | Public contributor testing page; detailed inventories remain engineering-side |
| Native independent review (X/H) | `internal/opencodeadapter` and experimental CLI excluded from default build; 006 IR-06-01 unresolved | integrations/opencode; 006 records | Historical adoption framing can look like current support | Preserve records; public known-limitations warning, not tutorial |

### Concrete asset and command facts

- Core suffixes: `architecture-review`, `bootstrap`, `debugging`, `drift-retro`, `grilling`, `grill-me`, `handoff`, `implement`, `plan`, `questionnaire`, `research-spec`, `router`, `spec`, `tasks`, `teach`, `verification-review`, `wait-what`, `writing-for-agents`. Prefix each with `agentic-sdd-`. Do not use the active harness's skill list as the catalog.
- Canonical client IDs/destinations from `clients.go` / `sync.go:clientTargets`: `agents` → `~/.agents/skills`; `codex` → `~/.codex/skills`; `claude` → `~/.claude/skills`; `agy` → `~/.gemini/antigravity-cli/skills`. Pack profiles target `~/.config/opencode/agents`. These paths do not prove identical client/model behavior.
- CLI command families: preview/apply, version, help, backups, logs, restore, guide, tui, fleet, mcp, run, task, review. `opencode`/`opencode-harness` are compile-time experimental and unavailable in default core builds; `orchestrate` is a help topic, not an invented execution command.
- Fleet `plan --output` writes a plan file even though destinations are not changed. “All previews write nothing anywhere” is too broad.
- Backups with explicit `--repo` live at `<repo>/backups/`; embedded-source backups use `<home>/.agentic-sdd/backups/`; logs still use the selected home. Fleet/MCP/execution have distinct state.

## Specification history 006–019

This is an investigation map, **not another status ledger**. Links point to actual current owning `tasks.md`; do not promote a summary into approval.

| Feature | What carried forward | Evidence/lifecycle qualification |
|---|---|---|
| [006](../006-independent-review/tasks.md) | Portable independent-review method and offline probes | Native B4 deferred; IR-06-01 unresolved; fenced, not a supported runtime |
| [007](../007-taskfile-migration/tasks.md) | Taskfile adoption/history | Historical old-base migration gap; do not select old-transition validation |
| [008](../008-core-release-quality/tasks.md) | Supported-core quality gate/trust and exclusion fence | Open release-readiness items remain owned there; current 11-package selection reflects later additions |
| [009](../009-opencode-orchestration/tasks.md) | Optional pack, delegation guidance, routing | Accepted records; runtime/profile/provider permissions still environment-dependent |
| [010](../010-guided-cli-ux/tasks.md) | Numbered guide, trace, terminal/non-TTY behavior | Accepted; preserve output/exit contracts |
| [011](../011-cli-brand-styling/tasks.md) | Branding/presentation and explicit update checks | Implementation present; some original human/URL observations incomplete |
| [012](../012-cli-tui-remediation/tasks.md) | Readable catalogs and original isolated TUI prototype | Historical prototype status does not describe later accepted TUI |
| [013](../013-unified-tui-experience/tasks.md) | Unified TUI, pair identity, package/cask, theme/help/recovery | Accepted rows and v0.8.0 record; stale pending-review heading remains; not blanket present-platform proof |
| [014](../014-agent-fleet-model-overrides/tasks.md) | Offline fleet engine, model overrides/provenance | Implemented scope awaits human/Opus acceptance; FR-15 deferred; do not inherit old package-count prose |
| [015](../015-fleet-manager-tui/tasks.md) | Fleet waves + MCP transactions/TUI | W1/W2 and MCP implementation accepted; W3 review pending; source-save B4 unauthorized; live TUI default preview |
| [016](../016-agent-execution-context-evidence-recovery/tasks.md) | Typed run/brief/check/record/review services | Implementation complete, human review pending; observations never authority |
| [017](../017-agent-context-upgrade/tasks.md) | Owning-run/brief/checkpoint integrity | Accepted B1; does not accept 016; two-file interruption needs explicit reconciliation |
| [018](../018-research-evidence-handoff/tasks.md) | Research instructions and evidence handoff | Accepted 2026-11-14; static fixture/instruction evidence only, live activation/retrieval untested |
| [019](../019-cli-tui-entrypoint/tasks.md) | Canonical names, wrappers, Guide/help | Feature ledger human review pending; public v0.11.0/tag/cask metadata now verified at baseline, but archive/runtime/live-install and the ledger's local snapshot omissions remain distinct |

## Documentation findings

| ID | Verified finding | Impact / disposition |
|---|---|---|
| D-01 | `docs/README.md` and `docs/getting-started.md` link `README.md#support-and-release-scope`, which has no heading | Preserve legacy anchor at README, then canonical support-page links |
| D-02 | `docs/installer-reference.md` links `README.md#where-the-skills-go`; actual heading is “Where the files go” | Compatibility anchor and corrected canonical reference |
| D-03 | README CLI overview and installer command table omit five families; README TUI text still says four destinations | Refresh from actual dispatch/routes, not feature chronology |
| D-04 | `opencode-pack/routing.md:81` says 12 profiles; source/table contain 14 | Correct active count and source-validate it; no route change |
| D-05 | orchestration-guide's “every way” claim excludes newer fleet/MCP/execution surfaces | Narrow scope; split tutorial/how-to/reference ownership |
| D-06 | docs index misses guide-flows, MCP/fleet guides and multiple studies | Public/engineering index with explicit classification |
| D-07 | README, installer-reference and orchestration-guide duplicate install/flags/profile/restore tables | Move detailed owner once, keep short journey links |
| D-08 | `guide-flows.md`'s no-browser statement belongs to numbered CLI guide only | Preserve scoped accuracy, not a product-wide claim |
| D-09 | `docs/subagent-fleet-comparison.md` describes a machine-local fleet; source pack differs | Retain study as dated/local, exclude public default catalog |
| D-10 | `docs/tui-architecture.md` combines earlier toolchain audit and evolved TUI | Preserve historical portions; extract current explanation with source-backed maturity labels |
| D-11 | `docs/README.md` cleanup claim is historical; multiple handovers exist now | Index as session pointers with canonical-ledger caveat, not “only latest exists” |
| D-12 | Two unresolved router-reference links are recorded by 019 T09/T11 | Resolve exact affected source links during migration audit; packaged-skill fixes need their own bounded authorization if product instructions change |

### Separate product findings, not Spec 020 implementation

- **P-01 (confirmed help mismatch):** global CLI help calls MCP read-only and omits its write grammar, while family help/dispatch expose writes. Document the correct surface; track a separate help correction, do not expand into CLI redesign.
- **P-02 (needs verification):** clarify managed global MCP write target versus explicit `--config` overrides with actual target validation; do not promise arbitrary-file writes.
- **P-03 (documented limitation):** fleet TUI cannot clear a model override to inheritance; source-save B4 remains gated. No new writer in this feature.
- **P-04 (security/documentation warning):** execution child commands inherit permissions/environment and persist argv/logs privately, not encrypted. Never pass secrets in argv or publish raw run artifacts.
- Research raised additional MCP acknowledgment/token-hardening questions. They are **unverified hypotheses**, not established vulnerabilities; no exploit/security verdict or product fix is claimed. Route any investigation to a separately authorized issue without publishing sensitive detail.

## Test evidence integration

The [test report](../../docs/testing/test-cases-report.md) describes 1,451 declared Go/Python functions (1,229 Go + 222 Python), including five child helpers, at its pinned snapshot. It states that 659 Go cases are in the default core gate and 570 are opt-in/experimental; nested TUI accounts for 518. These numbers describe **selection and declarations**, not test execution, statement/branch coverage or a verification percentage.

No measured `coverprofile`/coverage baseline was found in the inspected report, Taskfiles or harness check configuration. Do not report “zero coverage”; report **coverage not measured/available**. Go coverage examples must call it statement coverage, not branch coverage.

| Important workflow | Available test/source evidence | Missing or insufficient evidence | Documentation action |
|---|---|---|---|
| Skill install/no-op/backup/restore | `sync_test.go`, `restore_test.go`, `manifest_test.go`, `permissions_test.go`, `package_test.go` | Real-home/platform operations not exercised here | Executable isolated example; guarantees limited to actual cases |
| Optional pack/model selection | `opencode_sync_test.go`, `opencode_pack_test.go`, `appops/routing_test.go` | No proof providers/models are available or follow workflow | Profile facts and manual fallback, not model certification |
| Fleet plan/drift/restore | `internal/fleet/*_test.go`, CLI/appops tests | 014 acceptance/live TUI/source-save limits | Preview-first examples; link owning feature limitations |
| MCP configuration/recovery | `mcp_config_test.go`, `mcp_transaction_test.go`, CLI/TUI tests | OAuth/connectivity deliberately untested; arbitrary targets need verification | Separate config tests from provider connectivity; temp config only |
| Run/brief/check/review | `internal/execution/*_test.go`, CLI integrity tests | Linux-focused evidence; child command is not contained; no AI reviewer isolation | Local fake command example, privacy and partial-result cases |
| TUI navigation/terminal lifecycle | Nested Go tests and explicit PTY scripts | Not included in `task test`; old toolchain can produce successful skip | Explicit prerequisite/result classification, selected route/link checks only |
| macOS cask/archive/upgrade | Static release-shape tests; 013/019 ledgers; public v0.11.0/tag/cask metadata | Archive members/runtime not inspected; no fresh live install; 019 local snapshot row remains not_run | Metadata-supported release claim, separately qualified binary/install evidence |
| Client/model learning workflow | Static skill/fixture suites, human-cycle historical records | No cross-client parity, learning/cost effectiveness or current live activation evidence | Label illustrative/manual expectations and untested claims |
| Docs/site/examples | Existing text/link-presence assertions only | No full link/build/search/browser pipeline | Add focused docs-only checks; no repository-wide test rewrite |
| Root embed/version package | Default gate compiles them; inventory notes no direct tests | Compile is not behavior coverage | Explain gap; delegate missing product tests to testing initiative |

Public contributor testing docs explain the two Go modules/toolchains, fixtures, environment sanitation, explicit suite selection, coverage commands and result interpretation. Detailed counts/coverage artifacts remain under `docs/testing/`; suite execution ownership remains `tests/harness/README.md` and actual selectors. The testing initiative owns inventory regeneration; 020 links it and detects stale source-revision/selection claims without creating a competing generator.

Current CI: `.github/workflows/ci.yml` uses `pull_request_target`, reviewed-base workflow/tooling and isolated candidate-as-data, `contents: read`, and skips `docs/**`/`specs/**`-only changes. `release.yml` runs `task verify` on a `v*` tag. New docs checks must be additive and must not weaken this trust contract or select fenced validations.

## Documentation migration matrix

Destination shorthand **P/** means proposed `website/src/content/docs/`; **keep** retains the existing repository path. “Current” means useful/source-aligned portions, not that every paragraph was runtime-tested. Active-guide migration extracts/reviews content, then leaves a short compatibility page at the original path with preserved important anchors. No completed specification is rewritten.

| Existing file | Current purpose | Accuracy / classification | New destination | Migration action |
|---|---|---|---|---|
| `README.md` | Entry + large manual | Mixed current/stale/duplicate | README + P/index, getting-started, reference | Shorten; preserve important anchors; distribute detailed owners |
| `CONTRIBUTING.md` | Development entry | Core selection accurate; testing/docs detail scattered | Keep entry → P/contributing/* | Retain concise bootstrap and links; no duplicate full test matrix |
| `docs/README.md` | Documentation/status index | Useful but incomplete/dead anchors | Keep engineering index → public site | Rewrite index/classification, not full public manual |
| `docs/getting-started.md` | Safe first use/workflow branches | Useful; support anchor broken | P/getting-started/first-five-minutes; tutorials | Extract bounded examples; compatibility page/anchors |
| `docs/installer-reference.md` | Commands/paths/formats | Valuable formats; command overview incomplete | P/reference/cli; files-and-backups; how-to/recovery | Split by need, retain old deep-link anchors |
| `docs/orchestration-guide.md` | Core and optional full manual | Valuable, overbroad completeness/duplicates | P/tutorials/first-feature; how-to/orchestration; reference/profiles | Split/reconcile; retain section anchors to exact destinations |
| `docs/guide-flows.md` | Numbered CLI implementation trace | Current for CLI guide, not TUI | P/reference/cli; architecture/interfaces | Extract public steps; retain engineering trace at current path |
| `docs/execution-runs.md` | Run/evidence contract | Useful; pending-state/network caveats | P/how-to/execution; reference/execution; architecture/evidence | Review inherited-process caveat; compatibility page |
| `docs/mcp-tools.md` | MCP CLI/TUI/configuration | Useful current transaction boundaries | P/how-to/mcp; reference/mcp | Preserve recipes/refusals; add store/target qualifications |
| `docs/fleet-manager-tui.md` | Fleet TUI usage | Accurate preview default; source-save/model-clear limits | P/how-to/fleet; reference/tui | Preserve limitations; avoid “enable live” instruction |
| `docs/codegraph.md` | Repo tooling setup/sync | Useful development procedure | Keep; P/contributing/development link | Keep engineering owner, avoid making optional tool a public prerequisite |
| `docs/tui-architecture.md` | Evolved architecture + old audit | Mixed historical/current sections | Keep history; P/architecture/interfaces | Extract current truth, retain audit evidence path |
| `docs/mcp-default-integration-study.md` | Install-time MCP proposal | Advisory, some ideas unimplemented | Keep engineering research | Label/index; not current instructions |
| `docs/research-skills-and-subagents-current-state.md` | Research/fleet snapshot | Advisory, machine-sensitive | Keep; P/how-to/research-debug extracts current source facts | No copied machine configuration |
| `docs/snap-ubuntu-release-report.md` | Linux packaging study | Proposal, not shipped package | Keep research; P/releases/compatibility link if useful | Never turn Snap/deb recommendation into installation instructions |
| `docs/subagent-fleet-comparison.md` | Local fleet comparison | Machine-local snapshot | Keep research | Index/classify; exclude from shipped-profile reference |
| `docs/agent-fleet-spec-review-2026-11-10.md` | Feature review input | Historical technical opinion | Keep | Add engineering-index pointer, not public authority |
| `docs/006-independent-review-spec.md` | Earlier draft | Superseded historical input | Keep | Point index to owning spec006; do not rewrite body |
| `docs/agentic-sdd-domain-study-2026-11-13.md` | Domain/code snapshot | Dated source study | Keep | Useful architecture input, not canonical live glossary |
| `docs/Agentic-SDD-vNext-Adoption-Plan.md` | vNext adoption proposal | Historical advisory | Keep | Engineering research index |
| `docs/Agentic-SDD-Workflow-Audit-2026-09-24.md` | Workflow audit | Historical | Keep | Preserve links and superseding-report relationship |
| `docs/compass_artifact_wf-97cc6285-b594-5b61-ae30-2894cfe0b595_text_markdown.md` | External research artifact | Historical | Keep | Descriptive index label; no rename for cosmetics |
| `docs/reddit_research.md` | External audience input | Historical/anecdotal | Keep | No adoption inference or public current policy |
| `docs/vnext-project-status.md` | 004/005 status snapshot | Historical | Keep | Add historical index pointer |
| `docs/vnext-remaining-work-guide.md` | Earlier remaining-work map | Historical, not work authorization | Keep | Link canonical ledgers; no new queue |
| `docs/vnext-human-test-guide.md` | Pinned-image procedure | Historical bounded evidence | Keep | Preserve fence; exclude site search |
| `docs/vnext-human-test-plan.md` | Human test design | Historical | Keep | Preserve, not a current test result |
| `docs/vnext-human-cycle-run-sheet.md` | Scenario recording template | Historical | Keep | No inferred scenario passes |
| `docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md` | Product compass | Canonical strategy, aspirations qualified | Keep; public overview links | No copied roadmap or changed strategy |
| `docs/testing/test-cases-report.md` | Declared-case inventory | Snapshot, not coverage/run | Keep engineering; P/contributing/testing links | Preserve detailed inventory; refresh through its own initiative |
| `docs/handovers/*.md` | Session pointers | Per-session, not authority | Keep | Engineering index, never public-search export |
| `docs/agentic-sdd-architecture.png` | README illustration | Must compare with current source before reuse | Keep or replaced reviewed small asset | No reuse merely for visual polish; alt text + capture/source note |
| `integrations/opencode/README.md` | Native-adoption entry/evidence | Historical blocked work | Keep | Add unambiguous historical framing without changing evidence |
| `integrations/opencode/adoption.md` | 005 adoption record | Historical | Keep | Index as history |
| `integrations/opencode/usage.md` | Native helper usage record | Historical/fenced | Keep | Not a current supported how-to |
| `integrations/opencode/safety.md` | Safety/correction record | Historical with valuable boundaries | Keep; public security derives current facts | Do not reinterpret as certification |
| `integrations/opencode/verification.md` | 005 verification limits | Historical | Keep | Preserve limitation evidence |
| `integrations/opencode/recovery.md` | 005 recovery subset | Historical | Keep | Keep exact scope; current recovery guide uses implemented services |
| `integrations/opencode/supported-version.md` | Pinned runtime versions | Historical, not general compatibility | Keep | Not current support matrix |
| `opencode-pack/README.md` / `routing.md` | Installed-package context/policy | Active, one stale count; volatile model details | Keep normative routing; P/reference/profiles facts | Correct count, source-validate catalog; no changed fallback semantics |
| `skills-files/**/SKILL.md` and references | Normative portable instructions | Current package source | Keep; P/reference/skills catalog | Link, never move/copy normative bodies into a competing manual |
| `specs/006-*` through `specs/019-*` | Requirements/approval/ledger/evidence | Mixed lifecycle, authoritative for own scope | Keep | Link evidence by source, no historical rewriting |
| `specs/014-agent-fleet-model-overrides/fleet-worked-guide.md` | Worked fleet usage | Valuable draft/current-source input | P/how-to/fleet | Review against current CLI; retain original accepted engineering path |
| `tests/harness/README.md` | Suite selection and fixtures | Canonical detailed execution map | Keep; P/contributing/testing links | No duplicated full inventory; preserve explicit patterns/fence |

Before migration, T01 captures an exact path/heading/link manifest. T08 extends this matrix if new tracked docs appear; unknown files are retained until classified. Redirects/stubs must preserve fragments or expose an explicit fragment-to-destination map. Site navigation uses user tasks, never feature numbers.
