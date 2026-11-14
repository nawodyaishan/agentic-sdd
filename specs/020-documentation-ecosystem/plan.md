# Feature 020 — Implementation plan

**Revision:** r3, 2026-11-14 — approved set referenced in spec §9; B1 implementation complete and awaiting human review; later batches unchanged/unauthorized.
**Input:** [spec.md](spec.md) r2, [evidence](evidence.md), [audiences](audiences.md), [ADR](adr-documentation-platform.md), [page contract](sitemap.md).
**Approval owner:** [spec §9](spec.md#9-approval). B1 implementation is authorized; publication/live operations and later batches are not.

## 1. Approach and source boundaries

Adopt the proposed ADR: ordinary public Markdown in Starlight's standard `website/src/content/docs/`, with a static Pages projection. Preserve `docs/` engineering evidence/history and old-path compatibility. No prose mirror, symlink adapter, separate repo, CMS or new Go runtime dependency.

This feature changes documentation and supporting validation. It does not repair unrelated product behavior. In particular, incomplete global MCP help is a separate defect; generated reference uses family help plus source-validated curated details and states the discrepancy until corrected independently.

### Proposed file impact

These are **future implementation paths**, not files created by this planning session.

```text
website/
  package.json                 # isolated site/tool dependencies, exact lockfile
  package-lock.json
  .node-version                # exact supported runtime selected in B1
  astro.config.mjs              # origin/base, Starlight, edit/social/sitemap
  src/content.config.ts        # standard docsLoader/docsSchema
  src/content/docs/             # canonical public Markdown: sitemap.md L/C rows
  src/assets/                  # reviewed lightweight public assets only
  docs-manifest.json            # small source/page/URL/compatibility registry
  scripts/
    reference.mjs              # help/catalog/source-derived sections + parity
    check.mjs                  # source/built links, fragments, metadata, scope
    examples.mjs               # explicitly allowlisted offline example fixtures
    export.mjs                 # Markdown twins, public index, llms.txt
    impact.mjs                 # changed-path → affected documentation report
  tests/                       # generator/link/base/export/impact negative fixtures
.github/workflows/docs.yml     # separate secretless validation/build jobs
scripts/                      # existing trusted infrastructure reused as applicable
README.md / CONTRIBUTING.md    # concise canonical entry links
docs/README.md                 # public/engineering routing index
docs/<migrated-guide>.md       # old-path/fragment compatibility, not a second manual
tests/harness/                 # explicitly selected docs contract test if needed
cmd/agentic-sdd/               # development-only reference parity test, no product API
experiments/tui/               # optional fixed docs links only (B6)
Taskfile.yml                   # additive docs targets, no default/core gate change
```

Do not create all proposed scripts if one small module can share the URL/manifest parser. Keep the observable commands/contracts below even if implemented by a single script with subcommands. Do not introduce a package/interface merely to shuffle code.

## 2. Shared contracts

### C1 — Content, URL, evidence and compatibility registry

One minimal versioned JSON manifest, read by generation, validation, export and impact reporting:

| Field | Meaning / constraint |
|---|---|
| `schemaVersion` | Initial `1`; unknown versions fail validation |
| `site` / `base` | Sole authoritative approved HTTPS origin and normalized project base; `astro.config.mjs` imports them, no independent copy or user-configurable remote fetch |
| `pages[].id` / `source` | Stable route ID and contained `.md` source under public root; unique, no traversal/symlinks |
| `type` / `audience` / `class` | Diátaxis form/audience and L/C scope; no deferred pages in published navigation |
| `sources[]` | One list of repository-relative public evidence inputs, optional symbol/heading and reviewed SHA-256 of each full file; checked for containment/existence |
| `reviewDigest` / `reviewStatus` / `reviewedAt` | Fingerprint of reviewed inputs; `reviewStatus` is `unreviewed` or `reviewed`; `reviewedAt` is present only after an actual human review, never stamped by generation |
| `reviewedCommit` | Optional traceability pointer only; no ancestry gate or full-history requirement |
| `docsVersion` / `appliesTo` | `dev` or an evidenced release tag; release URL/source commit plus platform and verification-level qualifier; never unqualified “latest” |
| `releaseEvidence[]` | Reviewed offline snapshot: fetch time, public URL, exact tag/peeled commit, platform/asset names/digests and verification level; no live “latest” lookup in routine PRs |
| `aliases[]` | Old file/URL plus supported fragment mapping; no conflicting destination |
| `generatedSections[]` | Named section, producer/input identity and target marker; prose outside markers is authored |
| `examples[]` | Named fixture/action, effect class and expected observations; no free-form shell from page fences |
| `impactPaths[]` | Explicit source prefixes/globs for useful impact reporting, not a claim of exhaustive dependency analysis |
| `diagrams[]` | Mermaid source/image paths and hashes, authoring-tool version and reviewed projection note; integrity/provenance, not proof of semantic diagram correctness |

Keep title/description in page frontmatter using Starlight schema; do not duplicate them in the registry. Display build identity, documentation applicability and reviewed-source fingerprint distinctly. Source-input hashes work in shallow clones and dirty previews without a Git ancestry walk; an ancestor commit alone proves neither accuracy nor freshness.

A mismatch is `needs-review`, never automatically stamped as reviewed. An affected PR fails its relevant documentation contract until a human either updates content or records a no-public-impact explanation and intentionally refreshes the fingerprint. Unrelated PRs receive one consolidated pre-existing-drift observation, not a new product-gate failure; full documentation publication validates all L/C factual inputs and refuses known stale contracts. A local dirty preview labels itself uncommitted and is not a publishable release artifact. File hashing can overtrigger on nonsemantic changes: explicit no-impact review is the lightweight resolution, not a parser-specific semantic hashing system.

**Resolution rules:** authored page-to-page links use relative `.md` paths/heading fragments, readable on GitHub. A **proposed small Markdown-AST link transform**, not assumed Starlight behavior, resolves those links through manifest IDs into base-prefixed website routes. T01 must prove this with root/project-base rendered-output fixtures before migration starts. Keep the transform local to links, not prose or a custom content loader. Resolve engineering/source links to the repository at an explicit ref when not published; do not transform them into nonexistent site pages. Images use the supported asset pipeline. Escape/unescape paths and fragments consistently, including encoded spaces and duplicate heading IDs.

**Concrete transform design:** one local remark plugin, `website/plugins/remark-link-routes.mjs`, wired through the documented unified/remark hook for the **exact pinned Astro version**. Current upstream documentation exposes `markdown.processor` with `unified({remarkPlugins})`; older API examples must not be mixed into the selected version. It visits only Markdown `link` and reference `definition` nodes, resolves relative `.md` destinations by C1 source ID, preserves/validates fragments and emits exactly one normalized base prefix. External URLs and code are not transformed; images remain in Astro's image pipeline. Repository-escape source links use a fixed repository/ref URL, not arbitrary redirect targets.

Reject missing source-file context, unknown page IDs, unsafe schemes/traversal and raw HTML navigation links with file/line/reason. Public authored headings must have unique slugs, avoiding framework/GitHub duplicate-heading suffix differences; retain existing historical fragments through explicit aliases instead. F01–F11 fixtures cover root/project base, fragments, encoded spaces/no double encoding, reference links, external/source links, image URL, missing target, raw HTML refusal, duplicate heading refusal and independently parsed rendered hrefs. `docs:check-built` compares actual HTML/IDs to the manifest, not only the transform's own output.

If the bounded transform cannot satisfy both GitHub and site rendering, keep existing guides intact and temporary draft links targeting canonical GitHub source; stop before B2/migration for a design amendment. Do not silently switch to extensionless links that break on GitHub, mirror prose or replace the framework.

Outputs: canonical HTML URL per public page; a source/edit URL; optional `.md` alternate URL. No duplicated canonical for Markdown twins; sitemap contains canonical HTML URLs. Raw/GitHub remains the canonical *editable source*, HTML the canonical *search-facing URL*—these are different concepts.

**Failure semantics:** missing/ambiguous page, duplicate ID/canonical, missing source/fragment, invalid generated block, unsafe path or mismatched review identity fails the documentation check with page/source/reason and exit nonzero. Historical exceptions are exact documented mappings, not broad ignore globs.

### C2 — Reference generation without changing the CLI framework

Inputs are the current source revision, a locally built `asdx-core` from that revision, allowlisted help argv and canonical asset/parser files. No shell, PATH-discovered product binary, real-home inspection, provider request or native experimental command.

1. Capture top-level and **every family/subcommand help** with bounded stdout/stderr/timeouts and fixed presentation environment. Store generated blocks under the canonical references, with producer/commit metadata.
2. Use a small explicitly maintained command inventory tied to `runEnv` and family dispatch. Family help is more complete than top-level MCP text; curated reference owns missing semantic details and flags with source pointers.
3. Validate inventories against Go `go/parser`/AST in a development-only test within the existing CLI test package, covering dispatch cases, FlagSet registrations and curated exceptions. That test consumes only Go source/its curated test expectations, **not website files, generated output, Node or the presence of a website installation**. Node `docs:check` owns website/generated parity; the Go gate remains runnable independently. Fail closed on unrecognized dynamic registration rather than quietly missing a flag. No production export endpoint/Cobra change.
4. Parse skill/profile frontmatter with a pinned reviewed YAML parser in the website toolchain; do not invent a regex YAML implementation. Generate IDs/descriptions/models/modes/preset/adapter/install classification from `skills-files/`/`opencode-pack/`.
5. Generate fields where actual structured types/schema exist; otherwise validate authored tables/examples against current parser contracts. Fleet schema v1 is a strict Go parser contract, not an assumed standalone JSON Schema. MCP/config/env precedence remains subsystem-specific.

Two modes: `generate` updates only marked generated sections; `check` produces an exact drift diff and never edits files. Missing binary/assets or help failure is `blocked/failed`, never an empty successful catalog. Generated facts do not claim model availability, runtime permission enforcement or shipping status.

### C3 — Safe example execution

Examples have stable IDs, page references, prerequisites, fixed argv or typed fixture setup, expected output/state/exit and an effect classification:

- **offline/read-only:** help/version/preview/status against synthetic homes/configs.
- **offline/fixture-write:** install/no-op/backup/restore, MCP/fleet transactions or execution scratch only inside disposable fixture roots.
- **manual/external:** real release install, provider authentication, model delegation, remote MCP/network or terminal/browser observation; never run in normal PR checks.

The runner builds/uses the candidate CLI in an approved test context, supplies explicit `--home`, `--repo`/run root where relevant, synthetic Git config, sanitized environment and fake local subprocess seams. It never runs arbitrary Markdown fences, real-home `apply`, user-controlled `curl | sh`, provider calls or live release/tag/deploy.

Assert expected paths/digests, exit codes and meaningful output subsets, not timestamps/dynamic IDs or exact AI responses. Preview cases assert destination non-mutation. A fixture explicitly lacking prerequisites is `blocked/not_run`, not a pass or silently counted skip. Delete only runner-owned temporary paths. Product defects become separately recorded findings; no new repository-wide testing mandate.

### C4 — Publication inputs, trust and outputs

Only public-root Markdown/approved assets/generated projections enter the Pages artifact. Repository history, `docs/testing/` reports, specs, handovers, runs, backups and arbitrary symlinks are never recursively copied.

Build output: HTML/static Pagefind chunks/sitemap, generated Markdown twins, `llms.txt`, content/build identity and explicit public redirects if needed. No executable/server adapter. Enforce no symlinks/hardlinks in deploy artifact, as the [Pages artifact requirements](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages#configuring-the-upload-pages-artifact-action) state.

`build → validate → upload → deploy` is ordered; deploy consumes the validated artifact **from the same main-branch revision**, never an untrusted PR artifact. `contents: read` for build; `pages: write` and `id-token: write` only for deploy, protected `github-pages` environment, reviewed/pinned actions and concurrency to avoid stale artifact publication. Deployment does not need the Homebrew token.

**Authoritative PR evidence:** an additive docs job uses the repository's reviewed-base pattern: `pull_request_target`, `permissions: contents: read`, pinned checkout of base tooling and an explicit candidate SHA into separate data paths, `persist-credentials:false`, no `secrets.*`, no environment/deploy token, no OIDC, no privileged/shared cache. Only the reviewed launcher runs on the host. Candidate npm scripts, Astro configuration/plugins and dependency hooks execute solely in a digest-pinned disposable Node container. No host home/SSH credentials/Docker socket is mounted; source checkout is read-only, work happens in owned scratch, with dropped capabilities/no-new-privileges/bounded resources and network disabled during candidate execution.

**Evaluator/builder separation:** reviewed base owns the authoritative invocation, schema/policy/reference/link/export validators and expected fixture rules. Candidate scripts/configuration can build untrusted outputs, but their exit0 or self-authored JSON is not the validation verdict. After the candidate process exits, a separate fresh offline evaluator runs the base validators against read-only candidate source/output, with trusted tooling/dependencies mounted read-only and no shared writable evaluator path. Base launcher captures actual exit/log/artifact identities outside candidate-writable scratch and binds the result to the explicit base SHA/candidate SHA/lock/image identity. Candidate validator/launcher changes require review/adoption; they cannot self-authorize new check semantics.

**Expected scope and freshness:** base also owns the manifest schema, required page/route/public-source set and executed check list for the adopted scope. Candidate pages may be additive within that policy; an empty manifest, removal/reclassification of a required page or changed required check set is `needs-trusted-policy-review`, with execution/completion blocked until scoped reviewed adoption. Batches can have incomplete draft content, but a draft partial build never counts as L/C publication readiness. The final adopted publication set covers every accepted L/C page; no candidate-supplied manifest can shrink acceptance to get green.

The launcher derives completion from that expected set, actual command exits and independently validated output/source digests. It records source inventory before/after, permits writes only to declared cache/generated-output paths, and rejects changed source inputs. It scans **both PR and deploy** output for symlink/hardlink/traversal escapes before trusted reading/extraction. A forged candidate report is data, never the completion record.

Pass only explicit allowlisted container environment keys/values (image PATH, temporary HOME, locale/timezone, CI and required documented build settings); never forward host environment wholesale, `ACTIONS_*` credentials, provider tokens, `NODE_OPTIONS`, inherited npm configuration or a user home. Dependency symlinks needed by the reviewed toolchain stay inside its read-only dependency mount; candidate-source traversal and deploy-artifact links remain refused. No claim is made that these checks certify the semantic truth of arbitrary prose; human source review still owns that.

Apply the same constructed environment and fixed launcher working directories to **trusted provisioning/evaluation**, not just candidate builds: no inherited `NODE_PATH`, `NPM_CONFIG_*`, candidate `.npmrc`, `GITHUB_TOKEN` or `ACTIONS_ID_TOKEN_REQUEST_*` in any container. Checkout candidate data using the base repository's `refs/pull/<numeric-PR-number>/head` (or explicit readable fork/head ref), then require the resolved commit to equal the event's `pull_request.head.sha`. A missing/moving/mismatched ref is blocked, never evaluated as another tree. Base checkout is the explicit reviewed `pull_request.base.sha`; the evidence includes adopted base/policy/lock/image identities and actual resolved candidate SHA.

This is a separate, small docs launcher, **not** an expansion of the existing Go-only `trusted_ci.container_argv`, required package/check set or product gate. A `pull_request` candidate-authored status may be useful as an explicitly untrusted advisory observation, but cannot satisfy authoritative acceptance or become a branch-protection required check: the candidate can rewrite it to `exit 0`. r2 selects the reviewed-base alternative for authoritative evidence rather than silently trusting the cheaper advisory lane. Container isolation shares the kernel; it is not a general security-certification claim.

**Dependency provisioning:** reviewed base owns image/runtime/package-manager pins and the trusted lock identity. In a separate credential-free provisioning container, fetch only that reviewed lock's registry packages with lifecycle scripts disabled (`npm ci --ignore-scripts`, no audit/fund, no omitted optional platform packages). Audit lock/package agreement, registry HTTPS URLs/integrities and absence of file/Git/link dependencies. Never mount a private `.npmrc`, `.env` or personal cache. Copy approved dependencies into scratch; run candidate build/examples offline.

If selected native tooling needs install hooks, B1 must prove either no hook is needed or a short explicit **trusted, pinned offline hook allowlist**; run those only after network is removed. Do not permit all lifecycle scripts in a network-enabled candidate install merely to make a build pass. Missing native assets are `blocked`; choose a simpler dependency or obtain reviewed provisioning changes. Candidate lock/image/runtime changes produce `needs-trusted-dependency-review`, not an auto-refresh or fabricated green result.

Require candidate `package.json`, lockfile, runtime pin and selected image identity to match the adopted trusted set **before provisioning/build**; a mismatch stops execution, not merely annotates an otherwise-green run. Resolve dependencies from exactly one launcher-provided read-only tree; no build-time install/plugin auto-install or writable shared `node_modules`. Redirect legitimate generated caches to declared scratch using the pinned framework's supported configuration; B1 must prove the read-only dependency build works.

**Bootstrap/adoption:** the initial base lacks this launcher/lock. Local secretless B1 evidence is provisional until the owner reviews/adopts the infrastructure into a trusted base; the new checker cannot approve itself. Subsequent dependency refresh is a separately reviewed update of the exact trusted lock/image, not a routine requirement for prose edits. Workflow, Taskfile and test additions remain review-required under existing policy. No change to `ci.yml` exclusions, the six required product checks, Go runner or `harness/checks.json` is needed. If implementation cannot preserve these boundaries, stop for a human decision.

**Main/release:** trusted `push: main` runs the same validation and produces its own artifact, never promotes a PR report/site artifact through `workflow_run`. Release-docs sync is independent of GoReleaser and read-only; it consumes recorded release/tag metadata, cannot block product publication and never receives the tap token.

### C5 — Parallel-worker contracts and integration

The user's parallel-execution request changes the scheduling design, **not** feature approval, live authorization or the six-batch scope. T01–T12 remain acceptance owners; tasks.md subdivides them into 44 bounded dispatch lanes, not 44 new approval ceremonies. Peak useful DAG width is **11 writers**, in B3. One agent definition may serve multiple isolated child sessions; do not invent 11 new profiles.

**Isolation and baseline**

- One writer per worktree; the main/integration checkout has only the parent writer. Never launch concurrent writers into the current checkout, even for apparently disjoint files. Read-only workers may share an immutable snapshot.
- Before an authorized wave, record HEAD, actual dirty/untracked state, approved feature-document digests and an explicit source overlay. Create fresh worktrees from the selected commit, then copy only approved integrated overlay files—including authorized uncommitted/new files—with path/type/mode/digest checks. Existing untracked work is not automatically part of the overlay.
- Commits are not required for this transport: children return their exact changed-file inventory, bounded diff/new files and evidence; parent validates and copies/applies only owned changes. No implicit commit/cherry-pick/push permission. Preserve each worktree until its result is reconciled; never force-remove it.
- Each brief names the **absolute worktree root**, allowed paths, immutable input identity, source task plus lane ID, dependencies, focused commands, output contract and stop conditions. The child must use that root for every read/write/command. Verify actual directory/identity before writes; if the runtime cannot establish it, stop or run serially.
- Use CodeGraph only for the actual child project when its index is available/fresh. Never query the original checkout's default index as if it described a modified worktree. Otherwise report the limitation and use targeted source reads/searches. Do not initialize new indexes or change global configuration merely to dispatch.
- Worktrees are not security sandboxes. No secrets/backup/run upload, real-home apply, model/MCP evaluation, settings change, nested delegation, Git configuration edits or another batch.

**Frozen inputs and outputs**

| Boundary | Producer / contract | Consumer / failure behavior |
|---|---|---|
| Page/source/URL inventory | Parent freezes C1 schema, accepted sitemap IDs, origin/base, old-anchor snapshot scope and diagram IDs D01–D05 before the wave | Content writers use frozen IDs; unknown destinations/schema or changed requirements return `needs_input`, not local inventions |
| Shared helper modules | B1 owners provide `scripts/lib/manifest.mjs` (validated C1 loading/fingerprints) and `scripts/lib/routes.mjs` (source/HTML/Markdown link resolution), with data/error shape fixed before parallel tests/consumers | Reference/check/export workers consume these modules without rewriting them; an API change pauses affected lanes for one coordinated update |
| Reference modules | CLI, catalog and configuration producers return deterministic facts/generated-section data under C2 through separate owned modules; `reference.mjs` is parent-owned composition | Narrative page owners edit outside generated markers; parent alone runs generation against the integrated tree |
| Examples | C3 runner owns `examples.mjs`; content lanes return example IDs, fixed argv/effect/expected state in their handoff | Parent merges registration once; no page author adds arbitrary executable Markdown or edits another lane's fixtures |
| CI launcher | T10-A owns `scripts/docs-ci.py`; fixed development interface: `--trusted DIR --candidate DIR --output DIR --mode pr|main|release|rollback`, with C4 identity/result semantics | Workflow and adversarial-fixture lanes author against this contract, not a competing launcher; changes require coordination before integration |
| Diagram projections | D01 context, D02 interfaces/pairing, D03 install/recovery, D04 delegation, D05 MCP/execution; `.mmd`/PNG/provenance paths fixed before B3 | Architecture/security writers link these IDs and supply source facts; only diagram lane writes assets, parent reconciles meaning/hashes |
| Central registration | Parent alone integrates `docs-manifest.json`, package scripts/lock, Astro/content config, root Taskfile, generated blocks and task/handover evidence | Workers return registration deltas in their handoff; never concurrently edit these hubs or stamp review evidence automatically |

For the **initial creation** of package/config/shared modules, the designated B1 producer owns its listed files; after its return, central integration ownership transfers to the parent. Per-wave owner tables are exhaustive: no implicit ownership of an entire directory because one lane edits a file there.

**Tests, handoff and reconciliation**

Each writer runs only focused checks named in its brief, using per-worktree home/output/cache paths. Shared dependency artifacts are read-only; CPU-heavy builds/Go/PTY/container tests have a separate resource semaphore (default one at a time until measured capacity supports more). Parallel authoring does not mean 11 full suites concurrently.

Return: lane/task, base+overlay digests, absolute worktree, changed/new/deleted paths, interface or registry delta, actual commands/exits, blocked/not-run items and remaining findings. Parent rejects out-of-scope paths, unexpected deletion, input drift or forged/partial evidence. A child cannot update approval, this ledger or another lane.

Parent stops writing workers before applying their results to the integration checkout, merges in DAG order, composes central registrations/generated sections, and runs connected verification against the **integrated identity**. Child successes do not add up to an integration pass. Interface/input drift invalidates affected results only; inspect partial edits before retry, never replay a completed writer blindly.

Internal wave barriers are execution dependencies, not additional human approvals. Perform one normal batch-completion review after integration, record `awaiting human review`, and stop. Cross-batch parallel execution remains unauthorized unless the human separately changes that boundary.

## 3. Website design and dependencies

- Standard Starlight docs collection, explicit user-goal sidebar groups from the sitemap, `title` and descriptions; framework-default typography/theme/components.
- Small landing page, no custom design system. Default TOC, page navigation, copyable syntax-highlighted examples and dark mode. Actual keyboard, focus, mobile layout and contrast still tested.
- GitHub edit links point to `website/` with verified full source paths; do not assume whether the framework prepends `src/content/docs` without checking rendered output.
- Project base resolver shared by HTML links, Pagefind, sitemap, aliases, assets, Markdown exports and AI index.
- Root `/agentic-sdd/llms.txt` links only approved current public pages. Markdown twins carry title, canonical HTML, source commit and applicability, with links rewritten to absolute public URLs or pinned source links.
- Add standard title/description/canonical/robots/sitemap metadata. No custom structured-data schema initially; framework-supported metadata is sufficient. No analytics/tracking/pixels by default.
- Diagrams use repository-authored Mermaid `.mmd` source with reviewed pre-rendered static **PNG** projections and prose alternatives. Render only as an authorized authoring operation with exact tool/version/source hash recorded; routine CI validates source→asset provenance/size/alt text but does not install a browser or render diagrams. Changed source without updated projection evidence fails. PNG avoids active SVG/HTML/remote-include risks; do not add a runtime Mermaid integration. Target ≤150 KiB per image; larger justified diagrams may instead be simplified/split. Asset base-path behavior is a B1 fixture.
- Pin Astro/Starlight/Node and direct validation/YAML/diagram dependencies compatible with each other at B1. Record exact resolved versions and licenses. Use `npm ci` with committed lock; avoid unpinned action/package `latest`. Dependency fetching is an explicit network provisioning step, not an offline build claim.
- Optional local preview is `localhost` only by default. PR reports/static snapshots are labeled untrusted data, never public previews or deployable artifacts; do not open a candidate snapshot with a privileged/logged-in browser. No extra preview SaaS, auth or domain is needed.

## 4. Content and migration implementation

### README migration

First extract useful sections into assigned L/C owners, preserving technical caveats. Then replace README with its concise entry contract. Before the site is live, canonical links target GitHub source; after authorized deployment, add website links and retain a source fallback.

Keep legacy anchors as small named targets/sections linking exact destinations: support scope, old/new file-location headings, quick start, usage/orchestration and command migration. Avoid an enormous empty anchor appendix; preserve high-value inbound links found by the manifest and repository search.

One migration table owner is `releases/migration.md`; README has the minimal canonical/legacy names and a link, not a second maintained state/env table. Replace “What's new (010–013)” with release notes/compatibility links, not another feature-number changelog.

### Current versus engineering content

Use [evidence.md's matrix](evidence.md#documentation-migration-matrix). Extract current content; retain historical research, specifications, evidence and suite inventories at existing paths. Public pages may deep-link those sources but readers should not need a spec number to find a usage recipe.

For each active guide record old path+headings before replacement. Compatibility stubs retain required fragments and label canonical destination. `guide-flows`/`tui-architecture` can retain their engineering-only audit/trace sections with public pointers. Do not merge a historical description into current docs merely because it is detailed.

Label `integrations/opencode/README.md` historical/fenced at entry; do not rewrite evidence bodies or unblock native review. Correct the pack routing count only, not provider policies/frontmatter. Existing packaged-source broken links are investigated narrowly; if correcting them changes normative instruction scope, track a separate fix instead of expanding this feature.

### Architecture and security diagrams

Minimum useful set, distributed across existing planned pages rather than a diagram directory:

1. Context/container overview: humans, client/model/MCP external services, skill sources, CLI/TUI, managed files and authority.
2. Component/pairing view: actual CLI direct-service paths alongside TUI/appops, same-directory identity check and legacy forwarding; avoid false all-paths façade.
3. Installation/restore transaction: source→plan→backup→write/rollback, separate stores/formats.
4. Optional delegation sequence: main scopes→explicit child→evidence→main reconciliation→human; no daemon or automatic routing.
5. MCP/execution focused flows: configuration digest/lock/recovery versus child-command execution/freshness; explicitly no sandbox/auth-connectivity guarantee.

Each names the authoritative symbols/inputs and a reviewer; changes to those symbols trigger docs impact review. The Mermaid source is authoritative; the image is a generated projection, not a second editable diagram. C4 is a useful vocabulary, not a new architecture modeling system to maintain.

## 5. Testing-documentation ownership and synchronization

- `docs/testing/test-cases-report.md` owns the separate initiative's detailed inventory snapshot. Do not duplicate its 1,451-case list or recreate its generator here.
- `tests/harness/README.md` and actual selectors own exact suite commands/fixtures/fence; the public contributor page explains selection and links back for detail.
- Add a small capability/example/evidence map in the public-page registry, not a second case database. Changed test selectors/fixtures/source contracts trigger a documentation-impact entry.
- Coverage guidance provides an **optional measurement contract**, not an invented baseline or new task/gate. Resolve the current exact 11-package list from `tests/supported-core-go.sh` and its parity contract, use temporary owner-only output/cache/home under `/tmp/opencode`, and documented sanitized environment (`GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOENV=off`, `GOWORK=off`, empty `GOFLAGS`). The exact test argv is `go test -count=1 -covermode=set -coverprofile=<owned-temp>/core.cover . ./internal/version ./internal/skillsync ./internal/harnesscheck ./internal/harnessevidence ./internal/appops ./internal/buildinfo ./internal/fleet ./internal/execution ./cmd/agentic-sdd ./cmd/sdd-check`, then `go tool cover -func=<owned-temp>/core.cover` only if the run succeeded. Never use `./...` or `-coverpkg=./...`; keep nested TUI separate and opt-in. Provision unavailable prerequisites separately rather than enabling network within the test.
- A measured result requires exact revision/dirty state, Go/OS/architecture/tool versions, selection/env-key settings, UTC interval, command/exit, skips/testless or omitted packages, and profile/summary digests. Avoid a misleading single whole-repository percentage; retain per-package denominators and `no test files`/not-in-profile states. Profiles/HTML may reveal source and remain local/private, not site assets; preserve evidence before deleting only owned scratch. No measurement is run by this feature unless explicitly selected within an authorized batch.
- Explain Go **statement** coverage, Python declaration inventory, manual walkthroughs, model behavior and intentionally excluded lanes separately. Report revision, selection, toolchain, command, exit and skips alongside measurements. No fixed percentage target until a baseline and intended risk coverage exist.
- Inventory regeneration is invoked only if the testing initiative supplies its supported method; none was supplied/found at this snapshot. Otherwise emit one drift observation with changed selector/test path, report snapshot and affected claim, owned by the testing initiative. Triggers include selector/required-set/fence changes and Go/Python test-file additions/removals. Preserve `f559872` snapshot wording and do not regenerate/edit historical counts. AC-17 can accept explicit `not_measured` with reason/owner, but cannot accept an unresolved false selection/count claim.

## 6. Automation and lifecycle

### Proposed local commands

These names are planned interfaces, **not currently executable product commands**:

| Proposed interface | Contract |
|---|---|
| `npm --prefix website ci` | Provision locked dependencies; network may be needed; not part of Go-only workflow |
| `npm --prefix website run docs:generate` | C2 generation, no prose outside markers |
| `npm --prefix website run docs:check` | Offline lint/metadata/internal links/source and generated parity/scope |
| `npm --prefix website run docs:examples` | C3 selected safe fixtures only |
| `npm --prefix website run build` | Static Astro/Starlight build plus exports |
| `npm --prefix website run docs:check-built` | Routes/fragments/base/canonicals/search/export/privacy/artifact checks |
| `npm --prefix website run docs:impact -- --base <sha>` | One changed-path report, no automatic AI edits |
| `task docs-check` / `task docs-build` | Additive convenience wrappers; never replace `task test`/`vet`/`verify` |

### PR, release and scheduled lanes

| Lane | Triggers and behavior | Failure contract / budget proposal |
|---|---|---|
| Fast PR integrity | Reviewed-base docs launcher, explicit candidate SHA; `website/**`, README/CONTRIBUTING, named active-guide/alias paths, docs tooling and mapped product inputs; not a blanket historical-docs glob | Hard fail on deterministic link/parity/build/privacy errors; target ≤2 min with pre-provisioned reviewed dependencies; no remote link/AI call |
| PR full docs build/examples | Same affected docs-contract paths; credential-free isolated candidate build, bounded fixtures, artifact for review | Target ≤5 min warm full build/check; job timeout 10 min. Dependency cold provision measured separately |
| Release synchronization | Independent read-only preparation dispatch or `release.published` observation; reviewed metadata snapshot of exact tag/asset set, no replacement of release workflow | Failure blocks only affected docs publication, not product release; refresh unavailable → blocked observation, not unrelated offline PR failure; proposed ≤5 min warm stage |
| Main publication | Trusted main push or protected explicit dispatch, only after separately authorized activation and preflight; validate/upload/deploy same revision | Build/deploy separate jobs, protected `github-pages`, no PR artifact promotion; 10-min build/5-min deploy timeout; failed build keeps healthy deployment |
| Optional scheduled checks | Weekly on trusted main: external links, dependency/security notices, small fixed search evaluation | Informational issue/artifact, not unreliable external merge gate; timeout 10 min, bounded retries |

Budgets are proposed **targets**, not measurements. T01 records initial durations/cold/warm context; T12 reports median of at least three representative local/CI observations when available. A target miss needs diagnosis and transparent scope/tooling tradeoff; do not hide failures, add skips or relax the 008 gate.

Changes to `docs/testing/test-cases-report.md`, `tests/harness/README.md` or test selectors trigger only the relevant testing-docs/impact consistency check unless public/site inputs also changed. Historical specs/handovers/harness evidence are not copied, indexed or selected for execution; inbound links from them are checked as data during migration. This separates useful synchronization from noisy full-site builds on every engineering note.

**Docs impact:** CLI flags/help, TUI routes/keys, config parser/types/env, skills, profiles/model routing, MCP, backup/restore formats, client/source/install rules, security and user-facing errors map to page IDs. Emit one report listing changed sources/likely owners. PR template asks updated/no-public-impact-with-reason/tracked-gap-with-owner. Don't require every implementation PR to modify README or get a docs specialist review. Deterministic contract failures remain failures regardless of template answer.

**Staleness:** detect changed relevant source identities or selector drift, not “last edited >90 days.” A still-correct old page is not stale solely by age. No bot auto-commits, auto-publishes AI suggestions or marks an untested feature supported.

### AI-assisted update contract

An authorized worker receives changed public source paths, affected page IDs and the bounded question. Exa can find external practice/upstream sources; Context7 resolves current versioned library facts; local source/tests establish product behavior. Return a proposed diff plus claim/source/evidence labels and uncertainty. Human reviews safety/support/compatibility; deterministic CI validates structure, links, examples and generated parity. No mandatory MCP/key/paid inference in routine builds, no repository upload or raw private traces.

## 7. Quality and retrieval evaluation

| Measure | How obtained | Initial acceptance / baseline discipline |
|---|---|---|
| First useful workflow | Scripted isolated preview plus unfamiliar learner/experienced walkthrough if available | No real-home write, correct setup/consent understanding; time/completion baseline observed, no invented percentage |
| CLI/catalog/reference coverage | Enumerated dispatch/flags/assets versus manifest/generated blocks | Every current family and supported L/C contract mapped; unknowns fail or are explicitly classified |
| Broken links | Source and built-site fragment checker, exact migration aliases | Zero unresolved internal public links/anchors; historical findings separately enumerated, no broad suppression |
| Docs build reliability | Clean pinned build and repeated observations/CI exits | Deterministic success; first baseline captured; external failures separate from content errors |
| Accessibility/performance | Keyboard at desktop/mobile widths, automated audit, page-weight/Lighthouse snapshots on representative pages | WCAG-oriented authored-content review; proposed ≤500 KiB compressed first-page JS+CSS excluding optional images, ≤10 MiB site artifact excluding tool caches; adjust only with explained evidence |
| Search | 20 fixed user questions from audience/workflow map; 1–3 expected page IDs each; actual built Pagefind | Proposed ≥18/20 have a relevant result in top5, all install/recovery/security queries top3; preserve misses as evidence |
| Retrieval provenance | Markdown/index URL, commit, source identity and canonical checks | Every exported page has correct identity and public-source mapping; no proprietary service required |
| Contributor usability | New-page/source-change fixture + human walkthrough | One obvious place to edit, one local check command, a failed fixture gives actionable file/reason |
| Release synchronization | Compare docs source/build identity and applicability with candidate release | No unreleased-only instruction presented as stable; archive/install uncertainties retained |

### Q1 — Bounded usability and accessibility protocol

Timeboxes limit evaluation effort, **not** claimed speed or a performance baseline. Record page/heading, step outcome, wrong turns, assistance and elapsed time with participant consent; retain only sanitized summaries.

| Card | Person / timebox | Task and observable result | Failure / unavailable disposition |
|---|---|---|---|
| L1 | Unfamiliar learner, not page author; 20 min | From index choose core-only/client path; follow installation to isolated version/help/preview; explain changed files, stop before apply; identify scope/checks/unknowns in direct-fix example | Real-home write, accidental apply or unclear effects is a blocking safety defect. No participant → `not_run`, not newcomer success |
| U1 | Experienced agent user unfamiliar with these docs; 15 min | Using navigation/search, find a rare flag/default, shipped-vs-local profile distinction, manual fallback and MCP recovery; record all four page IDs/headings | Needing source/spec archaeology is a discoverability finding, not a user failure; no participant → `not_run` |
| M1 | Non-author maintainer if available; 20 min | In owned scratch, map CLI/config/skill changes to pages; add one page; break a link/anchor and interpret file/reason diagnostics | Ambiguous edit owner or unactionable failure is a docs-tooling defect; author-only exercise labeled `author_walkthrough` |

An author may run all cards to catch problems but cannot supply unfamiliar-user evidence. If L1/U1 are unavailable, build readiness may still be reported; affected usability portions of AC-01/02 stay partial with **“fresh-user onboarding unvalidated.”** Whole-feature acceptance needs those observations or an explicit human acceptance of that named evidence limitation, never a silently waived check.

Use 320 and 1280 CSS-pixel viewport fixtures at 100%/200% zoom on index, first-five-minutes and a dense reference page. Check skip link/focus order, search open/close/Esc, navigation/TOC/code copy, no focus traps, and no page-level horizontal scroll except bounded code/table regions. Spot-check a screen reader where available; unavailable assistive tooling is `not_run`. Plain Markdown twins must preserve headings/lists/tables/code and textual diagram meaning without JS. Automated structure/link checks and manual observations are labeled separately; neither is blanket WCAG certification.

### Q2 — Fixed Pagefind evaluation set

Run against the actual static build and record source/lock/Pagefind identity, query, top five returned page IDs and expected-page rank. Use the pinned version's documented search API if available, otherwise the built UI; do not introduce a second search engine just for measurement. **C** marks safety/onboarding-critical lookup.

| Query | Expected page IDs | Critical |
|---|---|---|
| install on linux without homebrew | `getting-started/installation` | C |
| preview without installing anything | `getting-started/first-five-minutes`, `getting-started/installation` | C |
| what is a direct fix | `introduction/concepts`, `tutorials/direct-fix` | |
| how many documents does a feature need | `tutorials/first-feature`, `introduction/concepts` | |
| claude code skills folder | `introduction/clients`, `how-to/skills` | C |
| opencode agent profiles | `reference/profiles` | |
| model fallback not working | `operations/troubleshooting`, `how-to/orchestration` | |
| restore previous skills backup | `how-to/updates-and-recovery`, `reference/files-and-backups` | C |
| asdx-core vs asdx | `reference/cli`, `releases/migration` | |
| mcp server authentication | `how-to/mcp`, `reference/mcp` | |
| execution run cleanup | `how-to/execution`, `reference/execution` | |
| fleet config file location | `reference/fleet`, `reference/configuration` | |
| is my data sent anywhere | `security/trust` | C |
| what is not supported yet | `operations/known-limitations`, `releases/compatibility` | C |
| keyboard shortcuts in the tui | `reference/tui` | |
| add a new documentation page | `contributing/docs-and-changes` | |
| run only the core tests | `contributing/testing` | |
| which files get backed up | `reference/files-and-backups` | |
| approval before coding | `introduction/concepts`, `tutorials/first-feature` | |
| install skills for antigravity | `introduction/clients`, `how-to/skills` | C |

Initial proposed acceptance: expected page in top5 for ≥18/20 and in top3 for every C query; no broken search assets, empty critical results or out-of-allowlist/private/history pages. Misses remain named findings, not suppressed cases; fix headings/synonyms/content or return the affected acceptance scope for an explicit human decision. Keyboard search usability is manually observed, not misreported as an automated ranking test. Twenty queries are a directional acceptance fixture, not a population benchmark or a basis for generalized retrieval claims.

### Q3 — Baseline capture and result classification

Record three warm build/integrity timings (median/min/max), cold build separately from dependency provisioning, total deploy artifact bytes, and compressed JS+CSS actually loaded on index plus one deep page. Include OS/runtime/lock/source/tool identity and observation method; no Lighthouse score is invented or made a gate. Proposed budgets in §§6–7 remain targets until measured. Missing build/search artifacts mean `blocked/not_run`, not zero bytes/latency or success.

Deterministic link/parity/scope/example failures block the affected check. Manual checks yield named observed-satisfied/observed-issue/not-run records. A missed budget requires diagnosis and a reviewed tradeoff, never silent tolerance or skipped tests. No semantic/hybrid infrastructure initially: improve Pagefind content first; a custom retrieval system requires a later demonstrated need.

## 8. Rollout, rollback and operations

1. B1 infrastructure/URL/evidence contracts; retain current docs/README untouched.
2. B2 essential onboarding/reference and short README; GitHub links usable even before publication.
3. B3 complete current-feature usage/testing/architecture/security.
4. B4 migrate old paths and add deterministic QA/AI exports.
5. B5 validate publication/release workflows and integrated quality; separately request Pages enablement/deploy if desired.
6. B6 optional bounded TUI links only after stable URL verification, with separate authorization to execute that batch.

Each batch stops for human review; no deployment implied. For a broken site, retain/restore known-good artifact or build a known-good commit through the reviewed workflow. For broken migration, restore only this feature's affected compatibility/page changes after inspection; never reset unrelated work. Framework can be removed later while Markdown persists. No rollback touches user homes or model/MCP configuration.

### Publication preflight and recovery contract

Before the **first** live deploy, T10 records owner/admin read-only findings for Pages source/custom-domain state, Actions policy/token defaults, `github-pages` environment, allowed deployment branch, available protection/reviewer controls and artifact retention. Use official Settings views or documented API reads; unavailable permission is `unknown`, not disabled. Never display authentication headers/tokens. If the repository plan lacks the proposed protection control, stop for an explicit equivalent-policy decision rather than silently weakening protection.

Activation is a separate owner operation: choose GitHub Actions as Pages source and configure the reviewed environment/branch protection before enabling live deployment. Source-code approval does not perform or authorize those settings changes. Use pinned first-party checkout/upload/deploy actions and the explicit reviewed build steps, rather than a convenience action hiding dependency/build execution. Deploy job only has `pages:write`/`id-token:write`, depends on validated build, and uses a serialized Pages concurrency group. Documentation metadata remains GitHub-readable before activation.

After an authorized live deploy, verify base URL and one deep page return 200, a real sitemap and actual Pagefind/hashed asset URL under `/agentic-sdd/` return 200, canonicals/base are correct and a nonexistent route returns 404. Use actual emitted asset paths, not a guessed fixed filename. No prior web publication history was established; the default redirect list stays empty until an evidenced old URL needs one. Markdown compatibility stubs are unaffected by that decision.

Rollback order: (1) a fresh protected, explicitly authorized rollback workflow reuses an available validated previous deployment artifact after verifying its stored digest/source/lock identity; (2) if retention/API semantics prevent reuse, rebuild the known-good source with its exact reviewed lock/tool identity and revalidate; (3) owner-authorized Pages unpublication as last resort, with source links retained. Do not assume rerunning an old deployment ID works or automatically revert main. A rollback mode must explicitly bind the approved old identity; normal publication still rejects cross-revision/PR artifacts. Record retention/rollback window and test both retained/expired-artifact paths using fixtures before publication.

## 9. Specialist and tool assignments

Loaded/available during drafting: `agentic-sdd-orchestrate`, router/spec/plan/tasks, `agentic-sdd-writing-for-agents`, `documentation-generation-doc-generate`, `architecture-decision-records`.

Implementation defaults:

- **Public content/reference:** `documentation-generation-doc-generate` principal guidance; CodeGraph/local source and scoped tests, official version docs only when needed.
- **Contributor/agent instructions:** task-specific `agentic-sdd-writing-for-agents`; preserve authority and progressive disclosure.
- **ADR:** `architecture-decision-records` for a material amendment only; no mandatory additional reviewers/ADR tree.
- **Website/CI:** use `cicd-automation-workflow-automate` when implementing publication, `ui-ux-designer` for actual content/navigation accessibility review; consult Context7 for pinned Astro/Starlight/Pagefind/actions APIs and official GitHub docs for permission boundaries.
- **Optional TUI:** `golang-pro` for the bounded Go link change; existing tests/seams, no frontend application rewrite.
- **Verification:** `agentic-sdd-verification-review` for the normal authorized batch-completion review; explicitly requested independent reviewer/model per current repository policy, not an automatic paid route.

Resolve availability at task start; a written role assignment does not invoke/install it. The user requested the parallel implementation **plan**, not execution. C5 and §11 define isolated dispatch after combined approval and named-batch authorization; no nested delegation or parallel writers in one checkout. No research-session model becomes a product default.

**Execution profiles observed during r3 drafting:** `sdd-implement-1` and `sdd-implement-2` are installed; both currently pin `opencode/deepseek-v4.1-flash-free`, allow editing/shell/local code tools and deny nested delegation. The model catalog lists that free Flash model as active. This differs from the ambient summary describing Go defaults; actual profile configuration wins for a future dispatch and must be rechecked then. Listing/configuration is not a successful inference or a quota guarantee.

Tasks label **R** = routine `sdd-implement-1`, **C** = complex `sdd-implement-2`, and **P** = parent-only integration/authority. Reuse profiles across independent sessions. Do not silently override them to paid `opencode-go/deepseek-v4.1-flash`; the earlier paid read-only research scope is not future writer-spending authorization. If a route/permission/quota fails, preserve partial work, follow the currently authorized fallback policy and report conflicts/blocked lanes rather than improvising a paid route.

## 10. Risk register and closure criteria

**States:** `fact-established` is a bounded observation; `design-selected` is a proposed solution; `validation-pending` needs the named future task; `owner-operation` needs separate authorization. None means the risk has disappeared or a batch is authorized.

| ID / risk | Current state / selected treatment | Objective closure evidence | Owner / acceptance |
|---|---|---|---|
| R01 Release/ledger truth | fact-established: v0.11.0 tag matches baseline; release/cask metadata verified. Archive/runtime/live install unknown; feature ledger untouched | Every release assertion names tag/platform/evidence level; archived metadata/source mapping checks; no unrun binary/install claim. Additional live artifact verification remains separately scoped | T01/T06/T10; AC-04/05/18 |
| R02 Pages settings/activation/rollback | current URL unavailable; design-selected preflight, protected deploy, artifact retention and explicit activation; owner-operation pending | Recorded owner-only policy findings or named unknown; failed build cannot publish; retained/expired rollback fixtures; actual 200/assets/search/sitemap and nonexistent404 only after authorized deployment | T10; AC-08; live state not_run until authorized |
| R03 Loader/links/base | design-selected supported collection + one URL owner + narrow remark transform; validation-pending | F01–F11 source/built fixtures pass for `/` and `/agentic-sdd`; missing context/target and unsafe HTML fail before migration; no GitHub-broken extensionless fallback | T01/T02/T09; AC-09/10/12 |
| R04 Toolchain/dependencies/CI trust | design-selected exact reviewed lock/image, no candidate host/network execution, explicit bootstrap/native hook review; validation-pending | Version/license/lock audit; lifecycle tripwire cannot run during provisioning; build egress denied; lock drift blocked; no token/socket/cache/home leak; required core contract unchanged; reviewed-base adoption recorded | T01/T09/T10; AC-08/11/16 |
| R05 Migration/duplicate/freshness | design-selected committed old-anchor map, strict stubs, current-vs-history ownership and source fingerprints; validation-pending | Removing an inbound anchor fails; duplicate canonical/source/body fails; source change triggers one page review; full publish rejects stale factual inputs; no ancestry/full-Git dependency | T01/T08/T09; AC-09/10/12/18 |
| R06 Testing inventory/coverage | fact-established no measured baseline/supported generator found; optional measurement contract, single testing-initiative owner | AC-17 map separates declaration/run/statement/manual/model columns; authorized result metadata **or** explicit not_measured + reason/snapshot/owner; seeded selector drift yields one actionable observation | T06/T09; AC-17 |
| R07 Fresh-user usability | design-selected L1/U1/M1 cards; observations not_run | Isolated scripted safety check plus unfamiliar-user observations; unavailable participants leave AC usability partial unless human explicitly accepts that named limitation | T03/T12; AC-01/02/16 |
| R08 Search/accessibility | design-selected fixed20 queries and keyboard/plain/viewport protocol; validation-pending | ≥18/20 expectedtop5, all C expectedtop3; no export/history leakage; keyboard/nav/search observations and text alternatives; unavailable AT labeled not_run | T09/T12; AC-04/07/13/16 |
| R09 Performance/diagram/maintenance burden | design-selected static Mermaid projections/no browser dependency, exact version update policy, no mandatory AI service; validation-pending | Three timings + sizes/tool context; proposed budgets evaluated, misses explained; changed diagram-source fingerprint requires reviewed projection; no remote rendering/autoplay | T01/T07/T09/T12; AC-06/16 |
| R10 Optional TUI safety | design-selected existing fixed opener/default keys; validation-pending, B6 conditional | Explicit action only; nil opener/headless/SSH/failure retains full selectable trusted URL; unknown URL refusal; narrow/plain and existing input/write/confirmation tests unchanged | T11; AC-05/14 |

No unresolved fact is resolved by guesswork. Remaining platform pins, fixture results and human/operational observations have named owners and stop conditions; they are not tasks to execute during drafting. The combined proposal is spec r2 with plan/tasks r3; architecture/source location/main-doc policy and optional TUI scope remain unapproved.

## 11. Maximum useful parallel implementation schedule

Keep spec **r2** unchanged; this plan and tasks are **r3**. The one combined approval must identify those exact revisions when the human actually gives it. No approval or batch authorization is inferred from asking for a parallel schedule.

| Authorized batch | Dependency-safe waves | Peak ready writer lanes | Implementation lanes total | Integration barrier |
|---|---|---:|---:|---|
| B1 | W1: four T01 foundation lanes → parent integrate/freeze → W2: four T02 reference lanes | 4 | 8 | C1/C2 helpers, lock/config and generation work together before B2 |
| B2 | W1: six T03 content/example/provenance lanes → integrate → W2: two T04 README/entry-UX lanes | 6 | 8 | Launch pages exist before root entry links are replaced |
| B3 | W1: six T05 subsystem lanes + two T06 contributor/release lanes + three T07 architecture/security/diagram lanes | **11** | 11 | Cross-page/source truth and diagrams reconciled once, after all lanes |
| B4 | W1: three T08 migration lanes + four T09 QA/export/impact/negative-fixture lanes | 7 | 7 | All aliases, fresh registry and checker outputs composed before full verification |
| B5 | W1: four T10 CI/rollout lanes + two T12 acceptance-protocol lanes → parent final evaluation | 6 | 6 | Final measurements/acceptance rerun on merged revision, not earlier worktrees |
| B6, if retained | W1: four T11 opener/UI/tests/documentation lanes | 4 | 4 | Source/keys/URL docs and nested tests agree; AC-14 remains pending until reviewed |
| **Total** | One explicitly authorized batch at a time | **11 maximum useful width** | **44 lanes** | Six existing batch review stops, no new per-lane approvals |

The actual dispatch limit is `min(ready lanes, supported child-session limit, isolated-worktree capacity, provider allowance/rate limit, measured local resource budget)`. These runtime limits are **unknown until preflight**, not proven by earlier parallel read-only research. Fill available slots with ready lanes and start the next ready lane when a slot frees; never create idle agents or exceed a known limit to reach the headline count. Start all 11 B3 lanes only when the current environment actually supports 11 safe writers.

This is the maximum **useful graph width**, not a request to maximize total inference or fragment each paragraph/test into another agent. No speedup/cost saving is claimed. If safe worktree isolation, permission or allowance cannot be established, reduce concurrency or report blocked; never bypass policy.
