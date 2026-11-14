# Feature 020 — Tasks and batch ledger

**Revision:** r4, 2026-11-14 — B1 implementation evidence and review disposition only; approved scope remains spec r2 + plan r3 + tasks r3. B1 awaiting human review; later batches unauthorized.
**Parents:** [spec.md](spec.md) r2; [plan.md](plan.md) r3; [sitemap](sitemap.md) is the page-scope owner.
**Approval reference:** [spec §9](spec.md#9-approval), the only combined approval record.
**Current stage:** B1/T01–T02 implemented and integrated. B1 is **awaiting human review**; no later batch is selected.

## Batch ledger

| Batch | Tasks / waves | Lanes / peak writers | Outcome / completion | State / next action |
|---|---|---|---|---|
| B1 — Foundations and contracts | T01 W1 → integrate → T02 W2 | 8 / **4** | Standard Starlight source/build/base, frozen helper/manifest contracts and source-backed references | `awaiting human review 2026-11-14 — integrated gates green; not accepted`; next action is human B1 review |
| B2 — First useful experience | T03 W1 → integrate → T04 W2 | 8 / **6** | Launch pages/examples/references then concise README/entry navigation | `authorized / in progress 2026-11-14 — B2 amendment; stop after review` |
| B3 — Complete current-feature knowledge | T05/T06/T07 W1 together | 11 / **11** | Subsystem/contributor/architecture/security content and reviewed diagrams | `not authorized`; reviewed B2 then explicit B3 |
| B4 — Migration and maintenance | T08/T09 W1 together | 7 / **7** | Compatibility stubs, QA/export/impact and adversarial fixtures | `not authorized`; reviewed B3 then explicit B4 |
| B5 — Publication readiness and acceptance | T10/T12 W1 → final evaluation | 6 / **6** | Safe CI/rollout plus integrated search/a11y/usability/performance evidence | `not authorized`; reviewed B4 then explicit B5; deploy separate |
| B6 — Optional minimal TUI access | T11 W1 | 4 / **4** | Fixed docs links/opener/UI/tests/docs; existing navigation/write controls intact | `not authorized`; stable URLs/reviewed B5 then explicit B6, or owner removes scope |

At each batch end: run affected checks, perform one completion review, record code state/checks/results/open findings, set `awaiting human review`, write the session handover and stop. No automatic batch progression, commit, deployment or additional model spending. Optional B6 is not silently marked complete/skipped; removal from accepted scope requires an explicit owner decision.

## Dispatch rules and ownership

Plan **C5** owns the worktree/transport/frozen-interface/test/handoff contract and §11 owns runtime throttling. R/C defaults are in plan §9; the owner authorized the B1 model mix below. Only B1 may run; later lanes remain planned/not authorized.

**B1 model exception (owner request 2026-11-14):** use free Console implementation workers plus explicit catalog-resolved `9router/ag/gemini-4-hydrogen-high` and `opencode-go/deepseek-v4.1-flash#max` overrides. Native Go may draw on allowance/charges; 9Router upstream cost is not exposed. This is a bounded B1 override, not a global/profile/model-routing change. Preserve partial work on failure and follow applicable authorized fallback rules; do not infer a paid/cross-lane retry.

**B1 W1 dispatch:** A workspace → Gemini Hydrogen High; B routes/plugin → native DeepSeek Flash Max; C manifest → free Console worker; D independent negatives → free Console worker. Each root is a distinct `020-b1-w1-{a,b,c,d}` worktree under the session-managed worktree directory. The frozen packet is `/tmp/opencode/spec020-b1/contract.md`; initial approved overlay digest `cd72503d6bdebc4db5b0628b36b25b7f7a5f2a51e40bf6d4103ae35c0c9389f1`. Node24/npm11/Go1.27/Docker are present; 4 CPUs/~9 GiB available memory/~1.5 GiB initial disk mean dependency provisioning/builds are serialized. No later batch is authorized.

Path shorthand: **P/** = `website/src/content/docs/`, **S/** = `website/scripts/`, **X/** = `website/tests/`, **A/** = `website/src/assets/`. A row owns only its listed files/prefixes, not sibling files or generated blocks. Parent owns final hub registration, package/lock/config edits after B1 producers return, generated-section writes, task state and handover.

Each lane gets one child session in one isolated worktree. Parent records its actual absolute path; no child writes in the integration checkout. Multiple sessions may reuse R/C profiles. Start the maximum **ready** lanes the actual runtime/quota/resources safely permit, not an arbitrary number of idle agents. Heavy test processes are separately throttled.

Before each wave, parent freezes approved inputs, page IDs, exported helper/launcher contracts, source snapshot, file ownership and test commands. Writers may author against those frozen contracts without waiting for another lane's prose; missing sibling outputs are reported as integration-pending, never an invented pass. No wave barrier creates a separate human approval; batch completion still does.

Brief identity is `Txx + lane ID`, for example `T05 / B3-W1-B`. The CLI task extractor recognizes numeric Txx headings, **not these lane IDs**; do not invent `task brief --task B3-W1-B`. If useful, extract the owning numeric task and issue a smaller manual lane brief containing its one ownership row, relevant AC/C-contract pointers and source digests.

All rows inherit the owning task's objective, acceptance references and stop conditions. A lane must return changed paths, focused command/results, interface/registry deltas and unknowns; parent verifies the connected outcome. No writer commits/pushes, creates other agents, changes global configuration, modifies source outside its row or begins another batch.

## B1 — Foundations and contracts

Parent preflight freezes the C1 shape, file names and representative fixture expectations. W1 has four ready lanes. Parent integrates/fixes contract-only glue and pins the exact dependency/config snapshot before the four W2 lanes start.

### T01 — Freeze the evidence/URL contract and establish the website workspace

- [x] **Objective:** implement plan C1/C4 in a minimal standard-layout Starlight project without moving existing docs yet.
- **Dependencies:** combined approval of spec r2 + plan/tasks r3 and explicit B1 authorization.
- **Files/components:** new `website/{package.json,package-lock.json,.node-version,astro.config.mjs,src/content.config.ts,docs-manifest.json}`; representative fixture content/tests; evidence records in this feature only.
- **Consumes:** current source/spec/test report; proposed ADR and sitemap.
- **Produces:** exact tool/dependency/license pins and trusted dependency snapshot; reviewed origin/base; committed source/route/old-heading/inbound-link snapshot; public allowlist; C1 review fingerprints/release metadata; minimum build with relative links/static diagram/version labels.
- **Work:** preserve the now-verified v0.11.0/tag/cask facts while keeping binary/install evidence unknown; use default loader and the exact-version unified/remark hook. Prove C1 F01–F11 before migration. Prove dependency provisioning cannot run candidate scripts; audit native hooks/platform assets offline, block on unsupported hooks rather than allow network execution. Capture timing/size baseline and current fragments before any move.
- **Verification:** clean locked build in disposable environment; root/project-base/reference/encoding/missing-target/raw-HTML/duplicate-heading/image fixtures; lifecycle-script tripwire and egress-denied build; no private/history input; generated artifacts outside Git; current Go/module/asset sources unchanged. Initial local bootstrap evidence is provisional until reviewed-base adoption, not a fabricated CI pass.
- **Acceptance:** AC-08/09/12/15/16 foundation; NFR-01–07; exact current manifest paths/IDs unique, representative Markdown readable in GitHub and site, no feature/approval history mutated.
- **Parallelization:** W1's four isolated writers below; no shared-checkout writes. Parent alone finalizes central C1 registration and adopted pins after returns.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B1-W1-A / C | Initial `website/{package.json,package-lock.json,.node-version,astro.config.mjs,src/content.config.ts}`; `X/site-*` | Minimal pinned standard-layout workspace; clean disposable build, documented dependency/license/native-hook choices; return ownership of package/config to parent |
| B1-W1-B / C | `website/plugins/remark-link-routes.mjs`, `S/lib/routes.mjs`, `X/routes-*` | C1 source/HTML/Markdown resolver + narrow plugin; root/project-base, reference/encoding/unknown-target fixtures |
| B1-W1-C / C | `S/lib/manifest.mjs`, `X/manifest-*`, `X/baseline/` | C1 schema/hash/source/release/old-anchor inventory support; immutable input/duplicate/traversal/freshness fixtures; manifest registration returned to parent |
| B1-W1-D / R | `X/foundation-negative-*` | Independent F01–F11 and lifecycle/native-hook/read-only-dependency counterexamples against frozen contracts; report red/integration-pending honestly until implementations are composed |

### T02 — Generate and source-validate the current reference contracts

- [x] **Objective:** make CLI/skill/profile/configuration facts reproducible without changing product behavior.
- **Dependencies:** T01 C1 manifest and pinned tooling.
- **Files/components:** `website/scripts/reference.mjs`, shared URL/manifest parser, `website/tests/`; generated sections in `reference/{cli,skills,profiles,configuration,fleet,mcp,execution,files-and-backups}.md`; development-only CLI parity test if required.
- **Consumes:** exact candidate binary/source, allowlisted help argv, canonical skill/profile/frontmatter and parser/type files.
- **Produces:** C2 generator/check mode; complete family/subcommand/flag/catalog map; curated semantic tables with source pointers; distinct core/preset/adapter/local-agent classification.
- **Work:** enumerate all supported command families and experimental exclusions; preserve per-command exit/JSON/default differences; no “all flags global” assumption. Use source AST/parity checks for dispatch/FlagSets; family help plus curated corrections for known global MCP help mismatch. B1 registers source-derived facts; executable synthetic parser examples are deferred to the B2/T03 example runner rather than claimed here.
- **Verification:** generate twice yields identical output at identical source/tool versions; check does not write; remove a command/flag/profile/field mapping and assert actionable failure; malformed YAML/dynamic unsupported flag registration fail closed; no provider/network/home reads.
- **Acceptance:** AC-04/05/09/10/18 foundations; source counts match 18 core, optional orchestrate, 14 pack agents, four presets and two non-default adapters at the refreshed revision. Counts are generated, not permanent assumptions if separately approved product changes occur.
- **Parallelization:** W2's four lanes after T01 integration; reference modules are separate producers. Parent owns `S/reference.mjs` composition and canonical generated blocks.
- **Integration owner:** T09 incorporates all parity checks; T12 verifies coverage against actual source.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B1-W2-A / C | `S/reference/cli.mjs`, `X/reference-cli-*` | Complete help/dispatch/flag/output inventory; deterministic snapshots, missing/unknown help failure and no provider/home reads |
| B1-W2-B / R | `S/reference/catalogs.mjs`, `X/reference-catalogs-*` | Canonical skill/profile/preset/adapter facts; malformed frontmatter/count/installation-classification cases |
| B1-W2-C / C | `S/reference/configuration.mjs`, `X/reference-configuration-*` | Parser/type/env/manifest fact map + synthetic examples; no fictitious unified schema, clear source-backed exceptions |
| B1-W2-D / C | `cmd/agentic-sdd/documentation_reference_test.go` | Go-AST/source-only dispatch/FlagSet parity test; runs independently of Node/site installation, no Go product behavior change |

**B1 integration:** parent composes C1/C2, generates initial reference blocks, validates imports/pins/fixtures and current core/static checks. Freeze outputs for B2, perform one completion review, record `awaiting human review`, stop.

**W1 integrated evidence (2026-11-14):** four W1 worktrees returned; parent copied only their listed files after verifying the base commit and frozen overlay hashes. Parent reconciled the manifest evidence fence (specs/testing reports may be hashed as engineering evidence but cannot become public pages), a baseline old-guide expectation, and a site-test copy layout that had changed the repository-relative root. Integrated offline Docker suite: **215 tests, 215 pass, 0 fail, 0 skipped**, including project-base and root-base Starlight builds and Pagefind assets. The full suite was run from a writable scratch copy with the reviewed read-only dependency tree; `go.mod` was copied because the live-scaffold test intentionally checks repository evidence. This is W1 integration evidence, not the final B1 completion review.

**W2 dispatch:** four isolated worktrees `020-b1-w2-{a,b,c,d}` with overlay digest `24b8509f44ae76141f0769f207a5afeb15159aef80b4d272b68b6321bb72b838`. A CLI reference → Gemini Hydrogen High; B catalog → free Console DeepSeek; C configuration → native DeepSeek Flash Max; D Go AST parity → Gemini Hydrogen High. All four returned and are integrated; no B2+ dispatch.

### B1 completion evidence (2026-11-14)

**State:** implementation and integration complete. Gemini 4 Hydrogen review returned **Ready**; DeepSeek Flash Max returned **Needs changes** on review-state/provenance/baseline/CLI evidence. The immediate review-state, README/migration-anchor baseline and provisioning evidence corrections are applied; remaining C2 provenance/CLI corrections are folded into the B2 amendment. **Awaiting human review**, not accepted. No commit/push/release/deploy/real-home apply.

| Surface | Actual evidence |
|---|---|
| W1-A workspace | Pinned `astro 7.4.12`, `@astrojs/starlight 0.43.1`, `@astrojs/markdown-remark 7.3.9`, `yaml 2.9.1`; Node `.node-version` 24.22.3; offline build/Pagefind; tests included in final suite |
| W1-B routes/plugin | Resolver + remark plugin; F01–F11 fixtures and integrated site link rewrite; site builds under `/agentic-sdd/` and `/` |
| W1-C manifest/evidence | C1 schema, source hashing, review fingerprints, baseline inventory; integrated manifest and 9-page registry |
| W1-D independent fixtures | 125 contract cases; integrated suite reports no skips; lifecycle/read-only dependency invariants |
| W2-A CLI reference | 29 allowlisted help captures from explicit candidate binary; 9 focused tests |
| W2-B catalogs | 24 focused tests; 18 core, orchestrate, 14 agents, 4 presets, 2 adapters; pinned `yaml` |
| W2-C configuration | 15 focused tests; fleet/MCP/execution/files/configuration source-drift checks |
| W2-D Go AST parity | `go test ./cmd/agentic-sdd -run Documentation` passes; source-only, no Node dependency |
| Generated references | `node website/scripts/reference.mjs --check --binary /tmp/opencode/spec020-b1/asdx-core` → exit 0, **0 drifted**; `--generate` populated 8 pages (725 lines) |
| Integrated Node suite | In pinned `node@sha256:dc26ae92c9fb15cf6b1d63e20f0944b2efe0f115b20e00be48fb23fceb061c54` (v24.22.3), network none, read-only repo, read-only dependency mount, `exec` tmpfs: **268 tests, 268 pass, 0 fail, 0 skipped, 22.0s** |
| Repository Go gates | `task test` passes all 11 selected packages; `task vet` passes; `go test ./cmd/agentic-sdd -run Documentation` passes |
| Static/workflow gates | `test_core_workflow.py` 11/11; `test_research_workflow.py` 22/22; `git diff --check` clean; `codegraph sync` up to date |
| Dependency evidence | Reviewed lock SHA-256 `06ab6b7b282a725be8989c61a2af745aad73a9ef1537904c08c05ee13cf3db47`; provisioned tree `/tmp/opencode/spec020-b1/dependencies/node_modules` with `.package-lock.json` SHA-256 `408d6d65b66c32328853fa63ffc8ee4084110ef588b966262cdc9973ccb9f867`; pinned Node image digest above |
| Provisioning/lifecycle proof | Credential-free container: owned lifecycle control without `--ignore-scripts` executed its hook; protected `npm ci --ignore-scripts` left the sentinel absent (both exit 0). Full suite/build used `--network=none` and read-only dependencies |
| Real-binary parity | `ASDX_CORE_BINARY=/tmp/opencode/spec020-b1/asdx-core` supplied to the integrated suite; parity test ran, not skipped; candidate path limitation recorded for reproducibility |
| Review state | Manifest pages use `reviewStatus: unreviewed`; no `reviewedAt` before human review. C1 schema/test enforce the distinction |
| T01 old-guide baseline | `website/tests/baseline/real-guides.json` records current `docs/` active-guide headings/links; `website/tests/baseline/migration-anchors.json` records README/CONTRIBUTING/OpenCode integration/pack anchors; both matched structurally; B4/T08 consumes them |
| Asset/base-path proof | `website/src/assets/fixtures/base-path.svg` rendered through the content-image pipeline; site-build asserts hashed `/agentic-sdd/_astro/base-path.*.svg` and root `/_astro/base-path.*.svg` |
| Build baseline | Standalone pinned-container build: 10 pages, **3,422 ms**, `dist` **1,269,133 bytes** (cold relative to dependency provisioning; one observation, not a threshold) |
| Completion review | Gemini 4 Hydrogen: **Ready** (provisional on its unrun shell checks). DeepSeek Flash Max: **Needs changes** (review-state/provenance/baseline/CLI). Immediate review-state, README/migration-anchor baseline and provisioning corrections applied; C2 provenance/CLI corrections assigned to the B2 amendment. Technical verdicts only; human B1 acceptance remains pending. |

**Audit corrections applied:** reference pages are registered in C1; `astro.config.mjs` validates through `loadManifest` instead of raw JSON and shares base ownership; index evidence no longer couples to `go.mod`; real guide baseline added; dependency tree identity and actual worker models recorded; `node_modules` local symlink removed and ignore rule broadened. Low-priority test-scratch/style items remain documented, not silently claimed fixed.

**Residual limits:** W2-C uses explicit Go source-text anchors, so benign comment rewraps can require a reviewed anchor update; full release/archive/live-install, Pages activation, deployment and all B2–B6 content remain `not_run`/not authorized. The host Node is 24.21.0 while the pinned container is 24.22.3; the container is the authoritative run. Docker requires `exec` on its `/tmp` tmpfs for W2-A fake-executable fixtures.

**Worker models:** W1-A Gemini 4 Hydrogen High; W1-B native DeepSeek V4.1 Flash (requested `#max`; runtime identifies model name "DeepSeek V4.1 Flash 1017"); W1-C/W1-D free Console DeepSeek; W2-A Gemini 4 Hydrogen High; W2-B free Console DeepSeek; W2-C native DeepSeek V4.1 Flash (catalog/runtime name "DeepSeek V4.1 Flash 1017"); W2-D Gemini 4 Hydrogen High. No fallback was used. The one read-only audit used free Console DeepSeek. Costs/allowance are not exposed; native Go may have drawn on allowance.



## B2 — First useful experience

W1's six lanes consume the integrated B1 snapshot and frozen page IDs. W2 begins only after launch-page/example/reference integration, so README never points to nonexistent promised content.

### B2 amendment (2026-11-14, from Gemini/DeepSeek reviews)

Gemini 4 Hydrogen returned B1 **Ready** and recommended B2 navigation/baseline safeguards; DeepSeek Flash Max returned B1 **Needs changes** on review-state/provenance/baseline/CLI evidence. The immediate B1 review-state, README/migration-anchor baseline, provisioning proof and binary-path evidence are corrected. The remaining reference provenance and CLI-reference corrections are part of this B2 amendment; they are not silently waived.

- **A-01 README gates:** preserve the two `cmd/agentic-sdd/guide.go` example prompts verbatim and keep a `docs/getting-started.md` pointer until B4, because `guide_test.go` and `test_core_workflow.py` assert them. Preserve high-value anchors: `quick-start`, `support-and-release-scope`, `where-the-skills-go`, `where-the-files-go`, `opencode-orchestration-pack---with-opencode`, `the-workflow`, `safety-model`, `explore-further`.
- **A-02 Link fence:** B2 pages may link only to B1/B2 pages that exist and to repository source paths; no relative links to B3-owned `architecture/*`, `operations/*`, `security/*`, `contributing/*`, `releases/*`, `how-to/{orchestration,fleet,mcp,execution,customization}`, or `tutorials/{specialists,research-and-debugging}`.
- **A-03 Registry test:** after B2 registration, update `website/tests/reference-manifest.test.mjs` for 19 pages, remove the hard-coded review date, assert `reviewStatus`, and require generated-section provenance only for the eight collector-produced pages; `reference/tui` is authored.
- **A-04 Navigation:** parent adds an explicit Starlight sidebar for the 14 B2 L pages and updates index metadata to `type: landing`, audience new/experienced; B1 C pages remain built but are not presented as launch nav until B3.
- **A-05 Reference pages:** keep frontmatter and generated markers intact; add curated skills categories and normative `SKILL.md` source links outside markers; create `reference/tui` with five routes/keys and current controls; recompute page/source fingerprints. Add `cmd/agentic-sdd/mcp_write.go` to the MCP factual-source set.
- **A-06 Examples:** `website/scripts/examples.mjs` accepts an explicit absolute `--binary`, uses isolated scratch and a frozen `{id,effect,argv,expected}` table, executes only `version`, help and `preview --home`/fixture cases, and refuses unknown IDs or Markdown fences. Add `docs:examples` package script and page `examples` registration; missing binary is `blocked`, not pass.
- **A-07 C2 provenance/CLI:** bind the collector to a candidate binary built from the inspected revision (identity/commit probe or explicit expected commit), record producer/commit metadata in generated sections, make `--check` emit a bounded first-difference diff, and correct the curated CLI table (preview exit `2`, `--version` root-flag semantics, experimental help topics, execution defaults). This is assigned to the B2 provenance lane and regenerated by parent.
- **A-08 Review vocabulary:** keep `reviewStatus: unreviewed` until human review; `reviewedAt` only after actual human acceptance. `appliesTo: unreleased` is the current B2/dev interval.

The amendment changes no B3–B6 scope and authorizes no live operation. B2 page IDs remain: `index`, `introduction/{concepts,clients}`, `getting-started/{choose-setup,installation,first-five-minutes}`, `tutorials/{direct-fix,first-feature}`, `how-to/{skills,updates-and-recovery}`, `reference/{cli,tui,files-and-backups,skills}` (14 L pages).


### T03 — Author launch onboarding and safe worked examples

- [ ] **Objective:** enable one useful skills-only experience before advanced configuration.
- **Dependencies:** B1 reviewed; explicit B2 authorization; T01/T02 contracts.
- **Files/components:** sitemap L introduction/getting-started/tutorials/direct-fix/first-feature/how-to-skills and essential reference sections; `website/scripts/examples.mjs`, named fixture projects under `website/tests/`.
- **Consumes:** verified current getting-started material and core workflow instructions.
- **Produces:** plain-language setup chooser, isolated first-five-minutes path, one direct fix and one feature tutorial, prerequisites/effects/outcomes/cleanup, C3 runner with positive/refusal/no-write fixtures.
- **Work:** distinguish TUI pack-on from CLI core-only default; qualify macOS/source builds and unknown Windows behavior; explain direct-fix versus consequential feature and optional teaching. Show explicit human approval placeholders, never fabricated accepted records.
- **Verification:** run documented help/version/preview and fixture-only writes; destination digests unchanged after preview; no real-home access; tutorial outputs distinguish application assertions from illustrative AI responses.
- **Acceptance:** AC-02/03 and onboarding portions of AC-01/04/07; learner can stop after preview without credentials, subagents or live writes.
- **Parallelization:** six W1 lanes below. All example registration deltas go to the parent; only the runner lane owns shared example implementation; the provenance lane owns generator/CLI correctness, not page prose.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B2-W1-A / R | `P/index.md`, `P/introduction/{concepts,clients}.md`, `P/getting-started/choose-setup.md` | Plain-language orientation/client owner/setup choice; source facts, relative links and core-vs-optional distinction |
| B2-W1-B / R | `P/getting-started/{installation,first-five-minutes}.md`, `P/how-to/{skills,updates-and-recovery}.md` | Platform-qualified isolated preview/install/recovery/uninstall path; destinations/effects/cleanup, no live claims |
| B2-W1-C / R | `P/tutorials/{direct-fix,first-feature}.md` | One reusable practical example and correct approval/one-batch stopping points; illustrative AI output distinguished |
| B2-W1-D / C | `S/examples.mjs`, `X/examples/`, top-level `X/examples-*.test.mjs` | C3 fixed-argv runner, owned fixture home/cache/output, preview non-mutation/cleanup/refusal tests; no Markdown-fence execution; missing binary blocked |
| B2-W1-E / R | `P/reference/{cli,tui,files-and-backups,skills}.md` | Essential curated reference prose outside generated markers; TUI five routes/keys; skills categories/source links; actual backup/source caveats |
| B2-W1-F / C | `website/scripts/reference.mjs`, `S/reference/cli.mjs`, `X/reference-cli-*`, related fixture tests | C2 binary revision/provenance, bounded check diff, corrected CLI table/defaults and experimental-topic note; no product behavior change |

### T04 — Modernize README and canonical entry links

- [ ] **Objective:** replace repeated manual content with a concise accurate front door.
- **Dependencies:** T03 public source destinations exist; README anchor snapshot from T01.
- **Files/components:** README and entry-navigation QA. Root CONTRIBUTING belongs to the B3 contributor lane, not this task; new visual assets remain optional.
- **Produces:** required README structure, canonical CLI/TUI distinction, setup/first-workflow link, benefits without unmeasured claims, release/CI/license badges, site/GitHub fallback links and preserved high-value anchors.
- **Work:** migrate useful details before deleting duplicates; migration table owner stays `releases/migration`; no claim the proposed Pages site or latest packaging has shipped. Replace feature-number “What's new” with release/compatibility pointers.
- **Verification:** every README command is in reference, links/fragments resolve, no duplicated full CLI/profile table; mobile/desktop GitHub reading check and no essential information hidden in visual assets.
- **Acceptance:** AC-01/03/09/18; approximately the proposed scan budget with justified exceptions, no unsupported platform/provider compatibility claims.
- **Parallelization:** two W2 lanes after T03 integration, separate files; parent applies returned sidebar/edit-link registration to central config.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B2-W2-A / R | `README.md` | Concise required structure, compatible anchors, canonical command names/source fallback and no unsupported release/provider claims |
| B2-W2-B / R | `X/entrypoints-*` | Entry/sidebar/edit-link/GitHub-mobile source checks against fixed destinations; navigation delta in return, no custom design system |

**B2 integration:** parent hydrates generated sections/registry/navigation, runs launch examples and link checks on the merged tree, performs one completion review and stops for human review.

## B3 — Complete current-feature knowledge

All eleven W1 lanes can start together after reviewed B2 and explicit B3 authorization. They consume **existing implementation evidence and frozen page/helper/example/diagram contracts**, not each other's unfinished prose. Source uncertainty is escalated; cross-links are reconciled during integration.


### T05 — Cover optional/current advanced workflows and their exact references

- [ ] **Objective:** complete current-feature how-to/tutorial/reference coverage without increasing product scope.
- **Dependencies:** reviewed B2, explicit B3 authorization, reference and examples contracts.
- **Files/components:** sitemap C tutorials/specialists and research-debugging; how-to/orchestration/fleet/mcp/execution/customization; reference/profiles/configuration/fleet/mcp/execution; recovery content.
- **Produces:** L/C workflow pages with complete source/evidence mapping; bounded routing exercise, explicit fallback/consent and limitations; synthetic config/examples.
- **Work:** distinguish shipped pack from ambient agents, written routing from automatic failover, MCP config from connectivity/OAuth, TUI preview marker from CLI writes, run evidence from approvals/sandbox, separate skill/fleet/MCP backup stores. Do not implement source-save B4, model-clear UI, native review or unknown target behavior.
- **Verification:** complete command/parser/catalog parity; isolated supported examples; invalid model/config, stale/conflict and partial-record expectations where existing behavior permits; manual/provider actions explicitly excluded.
- **Acceptance:** AC-03/04/05/07/18; every workflow named in the implementation inventory is documented or explicitly unsupported/deferred with a source.
- **Parallelization:** six independent subsystem writers; parent alone composes generated sections/registry after return.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B3-W1-A / R | `P/how-to/orchestration.md`, `P/tutorials/specialists.md`, `P/reference/profiles.md`; `X/examples/orchestration/` | Shipped/local/adapter/preset distinction, bounded delegation and manual fallback; source-catalog and safe routing exercise |
| B3-W1-B / C | `P/how-to/fleet.md`, `P/reference/fleet.md`; `X/examples/fleet/` | Fleet parser/plan/drift/restore/doctor and TUI preview/source-save limits; synthetic valid/invalid/stale examples |
| B3-W1-C / C | `P/how-to/mcp.md`, `P/reference/mcp.md`; `X/examples/mcp/` | Existing-global-config/auth-placeholder/transaction/entry-restore semantics; no connectivity/authentication overclaim |
| B3-W1-D / C | `P/how-to/execution.md`, `P/reference/execution.md`; `X/examples/execution/` | Run/brief/check/record/review contracts, partial evidence/private scratch/recovery; fake commands only |
| B3-W1-E / R | `P/tutorials/research-and-debugging.md`, `P/how-to/customization.md`; `X/examples/research/` | Named-fact research/debugging and supported customization; static/docs/model evidence separated |
| B3-W1-F / C | `P/reference/configuration.md`; `X/examples/configuration/` | One subsystem-specific config/env/precedence reference; real parser/type comparisons, no invented global schema |

### T06 — Integrate testing, contributor and release/compatibility documentation

- [ ] **Objective:** connect public contributor guidance to the separate engineering testing initiative and existing release truth.
- **Dependencies:** integrated T02/T03 source contracts and explicit B3 authorization; no dependency on unfinished T05 prose.
- **Files/components:** canonical `contributing/{development,testing,docs-and-changes}.md`, `releases/{compatibility,migration}.md`; CONTRIBUTING entry; docs-testing pointers/registry.
- **Consumes:** `docs/testing/test-cases-report.md`, suite README/selectors, current schemas/tests/CI, release records.
- **Produces:** test organization/toolchain/fixture/coverage/CI guide; explicit selection commands and result categories; test-documentation ownership/update rules; compatibility/migration/uninstall/rollback content.
- **Work:** identify declarations versus executed runs versus Go statement coverage versus manual/model observations. Document plan §5's exact optional `-count=1 -covermode=set` contract, private temporary outputs, omitted/testless-package caveats and result provenance; no measurement is needed to truthfully document `not_measured`. No `go test ./...`, broad Python discovery, new measurement task/gate or inventory generator. Link the `f559872` inventory and assign selector/test-file/fence drift observations to the testing initiative.
- **Verification:** compare examples and command selections with Taskfile/selector/checks; inspect representative tests behind capability claims; confirm no invented percentage or fresh live-install assertion; fixture coverage command only if explicitly selected/authorized, otherwise `not_run`.
- **Acceptance:** AC-15/17/18; contributor finds exactly how to test Go/site/skills/TUI and interpret results; capability map has separate declaration/executed/statement/manual/model evidence columns. Coverage is either actually measured with full provenance or explicitly not_measured with reason/snapshot/owner. Unresolved inaccurate selector/count claims remain findings, not an accepted missing percentage.
- **Parallelization:** two distinct contributor/release writers alongside all six T05 and three T07 lanes.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B3-W1-G / R | `P/contributing/{development,testing}.md`; `X/testing-docs-*` | Two-module setup, exact selected checks/fixtures, optional coverage contract and testing-initiative ownership; no new measurement/generator |
| B3-W1-H / R | `P/contributing/docs-and-changes.md`, `P/releases/{compatibility,migration}.md`, `CONTRIBUTING.md` | Contributor/skill/profile/docs/ADR/PR/release guidance, exact release qualifiers and sole migration-table owner |

### T07 — Author architecture, operations and security explanations

- [ ] **Objective:** provide high-level comprehension plus maintainers' source-backed boundaries.
- **Dependencies:** integrated B1/B2 source/evidence map, frozen D01–D05/link contracts and explicit B3 authorization; not T05/T06 prose completion.
- **Files/components:** sitemap architecture L/C, security/trust, operations/troubleshooting/known-limitations, related-approaches; authoritative Mermaid sources and reviewed static PNG projections/prose.
- **Produces:** small context/component and focused transaction/delegation/evidence diagrams with text alternatives; symptom-based troubleshooting; network/effects matrix; privacy/security/known-limitations page.
- **Work:** show actual direct CLI→execution/fleet paths alongside appops/TUI; pairing/legacy identity; source/embed loading owned by installer versus clients; separate configuration/credentials/connectivity; artifact-first/checkpoint-second recovery. Explain backups/checksums/permission/argv/log and same-user boundaries. Sensitive reporting links follow CONTRIBUTING, not an invented security address.
- **Verification:** diagram arrows/invariants checked against named symbols/tests; source/image hash and authoring-tool provenance, changed-source/stale-projection fixture; no runtime Mermaid/headless-browser dependency. Negative overclaim checklist, accessible text equivalents, no historical native profile promoted to support.
- **Acceptance:** AC-06/07 plus operations parts of AC-04; useful troubleshooting distinguishes application/client/provider/model, with safe next diagnostic and stop condition.
- **Parallelization:** architecture prose, trust/operations prose and diagram assets have separate owners; their shared factual inputs and IDs are frozen first.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B3-W1-I / C | `P/architecture/{overview,interfaces,workflows,evidence}.md` | Actual component/direct-engine/pair/transaction/delegation/evidence explanations; source arrows and diagram facts, no asset edits |
| B3-W1-J / C | `P/security/trust.md`, `P/operations/{troubleshooting,known-limitations}.md`, `P/introduction/related-approaches.md` | Source-backed trust/network/error/safety boundaries and dated factual comparison; no unsupported security/provider claims |
| B3-W1-K / R | `A/diagrams/D01*` through `A/diagrams/D05*` (`.mmd`/PNG only) | Five reviewed static projections, alt/prose requirements and render/source provenance in handoff; no runtime renderer/dependency change |

**B3 integration:** these ten content lanes own all remaining 25 L/C pages; diagram lane owns assets. Together with B2's 14 pages, every one of the 39 L/C pages has exactly one initial narrative owner. Parent checks cross-page truth, wires examples/hashes and generated blocks, runs merged validation, then performs one completion review and stops. Do not combine eleven child “passes” into a batch pass.

## B4 — Migration and maintenance

Three migration writers and four tooling/fixture writers start together from reviewed B3. They use the frozen C1/C2/C3 interfaces and target content, so no writer waits for another to choose its API or canonical destination.


### T08 — Migrate existing guide ownership and preserve old links

- [ ] **Objective:** execute the migration matrix with no competing source or historical loss.
- **Dependencies:** reviewed B3, explicit B4 authorization, target L/C pages exist.
- **Files/components:** existing active `docs/*.md`, docs index, historical-entry framing, current pack routing count, registry aliases; **not completed specs**.
- **Consumes:** T01 exact old-path/heading manifest and evidence migration matrix.
- **Produces:** completed file-level dispositions, minimal compatibility pages/fragments, public↔engineering index, any necessary previously published web alias pages.
- **Work:** preserve trace/audit sections where engineering consumers need them; keep historical files/accepted evidence in place; qualify local fleet studies; fix D-01–D-11 through canonical ownership/links. Stub-only files contain title/status, links and retained anchors, no procedures/code fences; retained engineering traces are classified separately. Investigate D-12 exact packaged-source links but create a separate fix if normative behavior would change.
- **Verification:** every migrated path/pre-migration slug is represented in committed C1 aliases; inbound links from specs/handovers checked as data; deleting an anchor fails with path/slug. Stub shape prohibits procedures/code fences; duplicate source/canonical/body ownership fails. Retained engineering traces are not copied public manuals. Default web redirects remain empty unless a prior published URL is evidenced; current 404 is not proof of no history.
- **Acceptance:** AC-09/12/18; every migration row has action/evidence/destination or explicit retained-history classification; no completed specification modified.
- **Parallelization:** three disjoint old-path groups; parent is the single alias/manifest coordinator.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B4-W1-A / R | `docs/{getting-started,installer-reference,guide-flows,execution-runs}.md` | Compatible old anchors/stubs or retained CLI trace; inbound links as data, canonical links exact |
| B4-W1-B / R | `docs/{orchestration-guide,mcp-tools,fleet-manager-tui}.md` | Reviewed migration/stub fragments with no competing manual or lost operational caveats |
| B4-W1-C / R | `docs/{README,tui-architecture}.md`, `integrations/opencode/README.md`, `opencode-pack/routing.md` | Public/engineering index and retained historical framing; active profile-count correction only, no routing-policy change |

### T09 — Add deterministic QA, docs impact and public AI-readable outputs

- [ ] **Objective:** keep docs correct cheaply and retrievable without mandatory external AI.
- **Dependencies:** integrated B1–B3 contracts and explicit B4 authorization; may author against frozen alias schema while T08 supplies its values. Full verification depends on integrated T08, not tool-authoring start.
- **Files/components:** website scripts/tests, Markdown lint config, impact mappings, source/built link checker, export; additive Taskfile targets; focused Python suite only if needed.
- **Produces:** proposed `docs:generate/check/examples/check-built/impact` interfaces; Markdown twins/public index/llms.txt; PR docs-impact guidance and governance.
- **Work:** parse links/headings with proper Markdown handling; validate source and rendered fragments (fences/inline HTML/duplicate headings/encoded paths); scope exports to canonical L/C allowlist; metadata/canonical version parity. Map all change surfaces in spec FR-12 with no required paid service. Retain source-validation exceptions as exact scoped evidence, never broad ignores.
- **Verification:** negative fixtures: broken anchor, missing family/flag/profile, unknown schema field, duplicate URL/source/body, wrong project base, leaked history/private path, unsafe example and generated drift. Changing a reviewed source fingerprint without human disposition fails affected checks; stale unrelated inputs produce one observation; full publication refuses stale L/C factual inputs. No automatic fingerprint refresh/commit-ancestry gate. Remote links/AI unavailable do not break offline checks.
- **Acceptance:** AC-10/11/13/15/16/18; generated outputs reproducible, no hand-edited parallel prose; no llms-full/vector/BM25/RAG service added.
- **Parallelization:** four isolated writers consume existing shared helpers; no lane rewrites `S/lib/` or central package/Taskfile/manifest configuration.
- **Integration owner:** T12 runs the connected suite and verifies false-positive behavior.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B4-W1-D / C | `S/check.mjs`, `website/.markdownlint.json`, `X/links-*`, `X/freshness-*` | Source/built link/fragment/metadata/duplicate/stub/freshness checks; actionable negative failures and no broad ignore |
| B4-W1-E / R | `S/export.mjs`, `X/export-*` | Canonical Markdown twins/index/llms.txt from allowlist; base/version/source/privacy/determinism tests |
| B4-W1-F / R | `S/impact.mjs`, `X/impact-*`, `P/contributing/docs-and-changes.md` | One low-noise change report, no-impact/tracked-gap rules and governance; no external AI or competing ownership |
| B4-W1-G / C | `X/negative-*` | Independent mutation/unsafe-example/forged-scope/history/private-link/alias tests against frozen CLI/contracts; pending producer integration reported honestly |

**B4 integration:** parent combines aliases/manifest deltas and package/Taskfile wrappers after workers stop; tests migrations **and** tooling together with mutation fixtures, then one completion review and human stop.

## B5 — Publication readiness and acceptance

Four CI/rollout writers and two acceptance-protocol writers start together from reviewed B4. T12 authoring/observations may use the B4 site snapshot, but final acceptance measurements must be repeated or proven reusable for the integrated B5 identity.

### T10 — Integrate safe PR/release/Pages workflows and rollback

- [ ] **Objective:** automate validation/publication without weakening trusted product CI.
- **Dependencies:** reviewed B4, explicit B5 authorization, T09 commands; actual Pages settings remain external prerequisite.
- **Files/components:** additive `.github/workflows/docs.yml` and small reviewed docs-only launcher; PR guidance; existing trusted CI/Go runner/required contract and `release.yml` untouched; deployment runbook.
- **Produces:** authoritative base-only PR launch (`pull_request_target`, contents read, candidate as data), validated trusted-main artifact/deploy pipeline, independent release-doc observation, optional scheduled external-link/dependency checks and recovery runbook.
- **Work:** follow C4's provisioning/offline native-hook rules and explicit trusted bootstrap/lock refresh. Candidate executes only in bounded networkless container; no PR secrets/OIDC/deploy token/socket/home/shared cache. Record owner-only Pages/Actions/protection/retention preflight and capability-dependent unknowns; activation/environment setting changes/deployment still require separate authorization. Do not add docs to product required scope or change existing runner argv policy.
- **Verification:** candidate changes workflow/launcher/validator/required pages/check list/lock or returns forged JSON/exit0 and cannot create authoritative success; dependency mismatch stops before execution. Base validator runs separately after candidate exit against read-only inputs; changed source inventory, moving/wrong head SHA and PR/deploy artifact escapes fail. Lifecycle/network/read-only-dependency/cache/constructed-env fixtures pass, with resolved base/candidate/policy/lock/image identity. Main publish and retained/expired rollback use protected fresh-run fixtures. Actual live page/assets/search/sitemap200 and nonexistent404 remain not_run until authorized.
- **Acceptance:** AC-08/11/15/16; reviewed-base adoption recorded, not self-approved; six required core checks/package selection unchanged. Routine validation has no live release/API/MCP/AI dependency. Docs-sync failure cannot block product release. Initial public `has_pages:false`/404 observation is recorded without inferring history or private policy.
- **Parallelization:** launcher, workflow, adversarial fixtures and runbook have separate owners under frozen C4/C5; only one writer owns the entire workflow file.
- **Stop condition:** conflict with 008/trusted-base contract or any proposed skip/exclusion requires human decision, not a bypass.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B5-W1-A / C | `scripts/docs-ci.py` | Trusted docs-only launcher/provision/evaluator/source/artifact identity under fixed C4/C5 interface; focused fake-process/offline fixtures |
| B5-W1-B / C | `.github/workflows/docs.yml` | Base/candidate checkout, read-only PR/main/release split and protected deploy/rollback wiring; no edits to existing CI/release required contracts |
| B5-W1-C / C | `tests/harness/test_docs_ci.py`, `X/ci-adversarial-*` | Independent spoofed-status/scope/lock/env/network/source-mutation/ref/artifact/retention tests; no real credentials/deployments |
| B5-W1-D / R | `P/contributing/docs-and-changes.md`, `P/releases/{compatibility,migration}.md` | Owner preflight, activation/retention/rollback procedure and release evidence snapshot refresh; unavailable settings stay unknown |

### T12 — Integrated acceptance, usability/search observations and handoff

- [ ] **Objective:** demonstrate core documentation-system readiness, not infer it from a green build.
- **Dependencies:** author protocols from integrated B4 and frozen C4/Q1–Q3 while T10 runs; final evaluation waits for integrated T10. B5 core acceptance remains before optional B6.
- **Files/components:** feature evidence/task continuation, any in-scope correction from findings, session handover. No completed older specs/approvals.
- **Produces:** AC-01–18 evidence map, conditional B6 status; plan Q1 L1/U1/M1 observations, Q2 exact20 query ranks, Q3 timing/size baseline, risk R01–R10 closure/remaining states and actual deployment status.
- **Verification:** clean pinned build; mutation/safe-example/old-anchor/duplicate/freshness checks; observed release versus binary-verification labels; Pagefind ≥18/20 expectedtop5/all criticaltop3 and no private/history results. Keyboard/plain Markdown at 320/1280 CSS px,100/200% zoom; actual search assets, text alternatives, three warm timings and cold/provisioning separation. Missing artifacts are blocked/not_run, not success.
- **Usability:** use Q1 exact cards with consent; author runs are labeled author_walkthrough, never unfamiliar-user proof. Unavailable L1/U1 leave relevant AC-01/02 usability portions partial and “fresh-user onboarding unvalidated” until actual observation or explicit owner acceptance of that named limitation. No token/cost/learning benefit claim.
- **Repository checks:** explicit core/research static suites and `git diff --check` after docs changes. If Go or Taskfile/workflows change, run repository-required affected `task test`, `task vet`, plus explicitly selected policy checks; never broad/fenced aggregate to claim a universal pass.
- **Acceptance:** every L/C page has owner/source/verification, findings resolved or clearly blocking the affected AC; budgets/result observations honest; no publication claim without actual deployment; advanced/future pages not smuggled into acceptance.
- **Completion:** set B5 `awaiting human review`; name B6 as next optional batch or request separately scoped publication authorization. Do not start either.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B5-W1-E / R | `X/search-acceptance-*`, `X/performance-*` | Fixed20 Pagefind/ranking/allowlist and Q3 timing/size capture tooling; observations returned with exact input identity |
| B5-W1-F / R | `X/accessibility-*`, `X/usability-*` | Q1 keyboard/plain/viewport checks and sanitized L1/U1/M1 recording templates; no participant contact without parent consent |

**B5 integration:** parent validates launcher/workflow/test agreement, re-evaluates final site/search/examples/metadata and updates AC/R01–R10 evidence. Parent alone records usability limitations and requests missing human/operational authority. One completion review, then stop; no live publication inferred.

## B6 — Optional minimal TUI access

Only if retained and explicitly authorized. Parent freezes fixed link IDs/URLs and existing key/return behavior before four worktrees start. No source-save/fleet/runtime work is added.

### T11 — Optional external documentation links in the existing TUI

- [ ] **Objective:** implement Option A only, not an embedded/context-browser manual.
- **Dependencies:** reviewed B5; stable URLs or owner-approved Git-source fallback; explicit B6 authorization and frozen link/UI/test contract.
- **Produces:** explicit existing Guide/Help documentation choice; fixed trusted category links and full selectable URL fallback, without a new route.
- **Work:** preserve `g`, `1–5`, focus/scroll/Guide-return and default-No/write shielding. No shell/dynamic URL/browser preflight/clipboard requirement. C5 resource/test isolation still applies.
- **Verification:** explicit-action-only launch; nil/failed/headless/SSH/unsupported paths retain literal trusted URL; unknown link refused; narrow/plain layouts and existing input/write tests unchanged; no real-browser launch.
- **Acceptance:** AC-05/14; source and docs agree; unavailable nested/PTY checks `not_run`. Parent runs connected Go/static checks and one completion review, records B6 `awaiting human review`, stops.

| Lane / profile | Exclusive write scope | Required output and focused verification |
|---|---|---|
| B6-W1-A / C | `experiments/tui/opener.go`, new `experiments/tui/docs_links.go` | Fixed trusted link constants/registry and safe URL/nil-opener fallback contract; no arbitrary URL conversion |
| B6-W1-B / C | `experiments/tui/{guide,library_help,activity,actions,app_screens}.go`; `app.go` only if explicitly necessary in frozen brief | Minimal Guide/Help action using existing routes/commands; no opener-policy or test-file edits |
| B6-W1-C / R | New `experiments/tui/docs_links_test.go`, `experiments/tui/docs_navigation_test.go` | Independent opener/UI/focus/narrow/write-shield negative cases against frozen existing seams |
| B6-W1-D / R | `P/reference/tui.md`, `P/getting-started/choose-setup.md`, `P/operations/troubleshooting.md` | Fixed-link help/manual-copy/offline documentation; provenance returned, parent reconciles with actual merged UI |

## Single readiness reconciliation

1. **Acceptance ownership:** spec table names task producers/verifiers for every AC. T12 integrates core-system evidence; T11 owns the optional TUI implementation and its completion review. Pending B6 cannot be mislabeled complete during B5.
2. **Producer/consumer agreement:** T01 produces C1 helpers/workspace; T02 separate C2 modules; T03 C3 runner/pages; T08 aliases; T09 checker/export/impact; T10 C4 launcher/workflow. Plan C5 freezes cross-lane inputs/interfaces and gives the parent sole central composition/registration ownership.
3. **Dependencies:** two explicit waves in B1/B2; eleven independent source-backed B3 lanes; B4 tools author against frozen alias contracts; final B5 evaluation waits for integration; B6 waits for stable links. No cross-batch dispatch, missing-destination migration, PR deploy or shared-checkout multiwriter.
4. **Meaningful verification:** positive examples plus non-mutation/stale/refusal/privacy/link/parity negative cases; inherited product semantics verified, not rewritten. Static checks remain static evidence, provider/model/real-home operations excluded.
5. **Material assumptions surfaced:** spec r2 architecture/risk scope is unchanged; plan/tasks r3 add C5 and an eleven-wide maximum-useful DAG. Actual runtime/allowance/resource capacity is unknown until execution preflight; no promised eleven-session throughput or measured savings. Public release/Pages facts remain bounded; live activation/rollback/adoption and implementation authorization remain separate.
6. **Lane completeness:** 44 unique lane IDs across six batches; initial public-content ownership covers all 39 accepted L/C pages exactly once (14 B2 +25 B3). The B2 amendment adds one provenance lane; later migration/maintenance/TUI updates are explicit later-batch ownership transfers.

## Drafting-session evidence and continuation

- **Source baseline:** `2fc31e9f9c4d019e2b9f3b4f004978028808c74b`; clean at start.
- **Changed scope:** seven new feature planning/supporting files and one session handover only. Existing README/site/TUI/product/specs untouched.
- **Investigation:** eight scoped workers, all DeepSeek V4.1 Flash on OpenCode Go; no model fallback. One child wrote an extra audience report despite read-only instructions; the parent inspected and moved/reconciled it into `audiences.md`, leaving no parallel research-report owner.
- **Executable observations:** CLI help and temporary CLI build/help family probes passed, detailed in `evidence.md`. No product test/release/site/live behavior inferred.
- **Validation:** see final validation results below.
- **State:** draft complete; pending one combined human review; **no implementation authorized**.
- **Next permitted action:** human reviews spec r2 + plan/tasks r3; after combined approval, explicitly authorize B1 (four T01 lanes, integration, four T02 lanes). Separate live operation authorization remains required. The actual approved revisions are recorded once in spec §9 when approval is given, not inferred from its older pending-draft revision wording.

### Draft validation results (2026-11-14)

| Check / evidence | Actual result | Limit |
|---|---|---|
| CLI build and seven help invocations | exit 0 throughout | Help/build only; captures described in evidence.md |
| `python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py` | exit 0, 11 tests OK | Static/fixture coverage, not model behavior |
| `python3 -B -m unittest discover -s tests/harness -p test_research_workflow.py` | exit 0, 22 tests OK | Static/fixture coverage, not live retrieval |
| Local planning/handover link/fragment, whitespace and changed-path scope script | exit 0; r3 eight files, 54 local links checked; only intended planning/handover paths changed | Narrow ad-hoc drafting check, not implemented site tooling |
| r3 scheduling/ownership script | exit 0; 43 unique lanes; widths 4/4, 5/2, 11, 7, 6, 4; 39/39 L/C pages have exactly one initial narrative owner; 12 task IDs and six ordered batch headings | DAG/ownership design verification, not runtime concurrency or implementation evidence |
| `git diff --check` | exit 0 | Tracked diff only; new-file whitespace checked separately |
| `codegraph sync` and `codegraph status` | already up to date | Local index, not validation of product behavior |

**Intentionally not run:** full Go test/vet cycle (no Go changes), nested TUI/PTY/container/product experiments, coverage measurement, website/dependency setup/build, browser/a11y/search benchmark, real-home apply, product provider/MCP authentication/model evaluations, release/tag/tap/Pages/deployment. Authorized research-agent/tool calls are not product-runtime verification. No skipped or unavailable operation is counted as a pass. Actual documentation CI/tools described in this draft do not exist yet.

**r1 consistency feedback (historical drafting step):** the platform and testing/CI workers performed bounded follow-up reads, not a consequential review verdict or approval. Reconciled URL/link/version/stub/input evidence and conditional TUI acceptance. The initial unprivileged-PR/ancestry choices were superseded by r2 below. No product defect or older feature ledger was changed to settle a drafting contradiction.

**r2 follow-up:** seven additional DeepSeek Flash risk workers returned source-backed proposals, reconciled by the main author. r2 supersedes r1's ancestry gate and candidate-defined unprivileged authoritative CI with source fingerprints and a reviewed-base docs launcher. Public release/cask/tag and current Pages probes were independently repeated. Risk closure is in plan §10; no “design-selected” or unmeasured item is marked implemented/passed.

**r2 focused CI feedback:** strengthened base-owned required page/check policy, independent evaluator/launcher-derived completion, pre/post source inventory, pre-execution dependency equality, read-only dependency tree, explicit fork/head SHA verification, constructed trusted/candidate environments and identity-bound adoption evidence. These are future T01/T10 fixture requirements, not proof that a docs runner or sandbox exists today. The two static suites were rerun after r2 content changes (11 + 22 OK); public probes and planning checks are the only new executed evidence.

**r3 scheduling request:** the user requested a plan/tasks design using the maximum number of useful parallel implementation subagents. Added 43 exclusive-write lanes, peak11, isolated-worktree/overlay/handoff/resource contracts and explicit integration waves; kept T01–T12 and the six batches so spec acceptance pointers remain valid. Actual implementation profile files/catalog were inspected: R/C currently use free DeepSeek Flash, not the earlier paid Go research route. No implementation agents, worktrees, builds, dependency installs or live operations were started for this revision.

This revision changed only `plan.md`, `tasks.md` and the existing session handover; spec/supporting records remain at their prior scope/revisions. Required core/research static suites were rerun: 11 +22 OK. Execution still requires combined approval of the actual spec r2/plan r3/tasks r3 set and named B1 authorization.
