# Feature 020 — Documentation Ecosystem Modernization & AI-Ready Developer Experience

**Revision:** r3, 2026-11-16 — README/site-attribution amendment approved and B7 authorized in §9; original approval and later B1–B6 acceptance with conditions remain preserved.
**Sources:** [product vision](../../docs/vision/Agentic-SDD-Project-Vision-and-Evolution.md) §§3, 6–8, 10–11; [repository instructions](../../AGENTS.md); [documentation index](../../docs/README.md); [test inventory](../../docs/testing/test-cases-report.md).
**Supporting records:** [implementation/documentation audit](evidence.md), [audiences](audiences.md), [platform ADR](adr-documentation-platform.md), [page inventory](sitemap.md).

## 1. Problem and outcome

Agentic-SDD needs an authoritative documentation system, not another overlapping manual. Its portable skills, optional OpenCode profiles, preview-first installer, paired CLI/TUI, fleet management, MCP configuration transactions and execution-evidence utilities have different support and trust boundaries. Existing guidance mixes current usage, earlier architecture, research, feature history and machine-specific model advice. Readers cannot reliably infer support from a feature number, an approval, a declared test or the newest paragraph in README.

The outcome is a small-project Docs-as-Code capability: a concise README, journey-oriented public Markdown, a searchable static website, preserved engineering history, source-linked references, reproducible examples and inexpensive maintenance checks. Git remains sufficient to read and contribute authoritative content. Publishing is a delivery channel, not a second source of truth.

### Current-state findings

- This is **not** a wholly Spec-006-era README. Later work updated names and many workflows. Preserve those corrections and useful guides rather than replacing them with generic prose.
- Current source uses `asdx` for TUI, `asdx-core` for CLI, and legacy compatibility wrappers. State/environment names remain unchanged. README still describes four TUI destinations despite the newer Guide route.
- CLI top-level help and older reference summaries underdescribe MCP writes; `mcp help` exposes apply and entry-level restore. Documentation must reconcile differing surfaces, not copy one help block blindly.
- Optional orchestration instructions/profile deployment are not an autonomous Go orchestration engine. Model identifiers, fallback permissions and provider availability are separate concerns.
- Public v0.11.0's tag resolves to this checkout baseline; release/cask metadata names the macOS archive and four commands. Feature 019's ledger still awaits human review; archive contents/runtime/live macOS installation were not checked. Release existence, artifact verification and human acceptance are separate facts.
- The testing report inventories declarations at `f559872`; this investigation starts at `2fc31e9`. Its counts are not line/branch coverage, passing runs or universal runtime verification.
- Feature 005/006/007 exclusions and the supported-core validation fence remain intact. Historical material must not be made into current onboarding.

See [evidence.md](evidence.md) for claim-level sources, uncertainty and migration dispositions. New product defects found during documentation work are recorded separately, not silently fixed under this feature.

## 2. Users and journeys

The audience mapping and evidence/hypothesis distinction live in [audiences.md](audiences.md). Strategic priority follows the vision, not invented adoption data:

1. Individual coding-agent users, including learners; offer one safe first success.
2. Experienced/OpenCode users and backend/full-stack engineers; offer concise references and optional depth.
3. Infrastructure engineers, maintainers, small teams and custom-workflow builders; offer explicit boundaries, recovery and contributor guidance without promising organization-wide policy enforcement.

Three reusable journeys share the same references:

- **Beginner:** what/when → choose client/setup → isolated preview → skills-only direct fix → understand verification and consent.
- **Intermediate:** one practical feature → combined spec/plan/tasks approval → one authorized batch → verification/review/resume; optionally research/debug.
- **Advanced:** optional OpenCode delegation → profiles/fleet/model configuration → MCP configuration → execution evidence/recovery → architecture/contribution.

A tutorial demonstrates a bounded workflow; it never guarantees that a model follows instructions. Readers can stop after skills-only setup. Tiny, already-understood fixes do not inherit mandatory feature paperwork.

## 3. Scope

### In scope

- README modernization; public information architecture using Diátaxis.
- Same-repository Markdown and a static public documentation website.
- Migration and compatibility for existing guides/anchors; engineering/public separation.
- Current-feature usage, architecture, operations, security, testing and contributor documentation.
- Generated or source-validated reference sections, safe executable documentation examples, deterministic documentation QA.
- Search, accessibility, stable URLs, release/checkout labeling and minimal AI-readable outputs.
- Lightweight governance, documentation-impact reporting, release synchronization and optional external research assistance.
- Evaluate TUI documentation access; implement only the bounded external-link enhancement described below if approved with this scope.

### Non-goals

No CLI parser rewrite, TUI replacement, installer/backup semantics change, model-routing/fallback change, MCP redesign, autonomous dispatcher, new skill/profile, runtime certification, repository-wide test implementation, mandatory paid AI service, custom CMS, separate documentation repository, brand redesign, chatbot, vector database, full offline embedded manual or retrospective rewrite of completed specifications.

No real-home apply, provider login, OAuth, tag, push, release, Pages enablement or live deployment is authorized by this draft or by approval of its development scope.

## 4. Functional requirements

### FR-01 — Truth and authority

Each public capability assertion must resolve to current source/schema/help, an executable test or an explicitly bounded observation. Label evidence separately from lifecycle:

- Implementation: present, partial, planned, experimental, deprecated, historical, unresolved.
- Verification: automated check actually run (revision/environment/result), tests present but not run, bounded manual observation, or unverified.
- Distribution: checkout/build identity versus published release/platform evidence.

An approved feature, green static fixture, declared test count, successful configuration probe or AI review is not universal support evidence. Public docs link a concise support/limitations page; detailed evidence stays engineering-side. Refresh material claims during implementation, especially pending 014/015/016/019 states.

`spec.md` approval and `tasks.md` batch state remain authoritative. Public summaries, generated indexes, test reports and handovers create no approval or execution permission.

### FR-02 — Canonical content and navigation

One editable canonical source per topic, standard Markdown by default, descriptive headings, consistent terminology and relative source links usable on GitHub. Public navigation follows Introduction, Getting started, Tutorials, How-to, Reference, Architecture/concepts, Operations, Security/trust, Contributing, Releases/migration. These are navigation groups, not ten mandatory landing essays.

The [sitemap](sitemap.md) assigns every proposed page a purpose, audience, source, dependency and validation method, classified **L** initial launch, **C** complete current-feature coverage, **A** advanced, or **F** optional future. L and C form feature acceptance scope. A and F are explicitly deferred unless a listed A section is incorporated into an accepted page; no empty placeholder pages are published.

Engineering history, full test inventories/coverage artifacts, research, handovers, feature approvals and harness records remain accessible in the repository but outside the website's Pagefind index and generated public exports. This does not remove publicly tracked files from GitHub search or external crawlers.

### FR-03 — README entry point

The approved full-refresh extension is in [§10](#10-readme-completeness-amendment-2026-11-16); it supplements this original requirement without discarding its compatibility and scope boundaries.

Replace the oversized manual with the requested scan-friendly sequence: product/value proposition; problem and plain-language explanation; supported capabilities/clients and limitations; benefits without unmeasured claims; workflow; platform-qualified installation; first useful safe workflow; CLI/TUI names; core versus optional orchestration; docs and architecture links; contribution/community; release/compatibility; MIT license and meaningful release/CI/license badges.

Use canonical names and a single migration-table owner. Do not imply Linux packages, Windows support, provider availability or a release migration without evidence. Keep all code examples documented. Target approximately 150–220 non-generated lines, subject to usability rather than a hard word quota. Move full references out, preserve valuable prose and old anchors with compatibility links. No full technical-reference duplication.

One small annotated screenshot may be added only if it materially clarifies navigation, is captured from the actual build, has alt text and a reproducible capture note. Text instructions remain complete. Prefer a static optimized image to an autoplay GIF; no required large terminal recording.

### FR-04 — Onboarding and tutorials

Provide prerequisites, macOS distribution versus source builds, Linux considerations, explicit unsupported/unknown platforms, setup choice, installation verification, updates and uninstall/recovery boundaries. Start with an isolated home/fixture and preview. Clearly mark any write, network call, client/provider prerequisite and cleanup.

Cover skills-only, one-agent feature lifecycle, optional specialist delegation, research-assisted work and debugging/verification. Model-routing tutorial content belongs only to optional OpenCode setup. Reuse one disposable example project where practical; each tutorial states inputs, expected observations, stop points and evidence limits. Model-produced text is illustrative, not byte-exact expected output.

### FR-05 — Current-feature reference and how-to coverage

Cover the complete current command surface and per-command flags rather than assuming global flags:

- Preview/install/update, client selection, optional pack/preset, guide, logs, backups/restore, version and paired TUI launcher.
- Fleet source selection, strict configuration, instance enablement/model overrides, preview/plan identity, apply/restore, ownership/drift, source-write/build-capability limits and offline diagnostics.
- MCP status/doctor/preview/apply/entry-level restore; global-existing-config limits, explicit auth recipes/environment placeholders, JSONC/ambiguity handling, local runtime probe, no network/OAuth/model validation.
- Execution `run`, `task brief`, `review package`: workspace identity, freshness/checkpoints, recording, cleanup, evidence/privacy and recovery.
- Core skill catalog, optional profile/preset/adapter catalog, client destinations, model grammar and manual fallback rules.
- Configuration precedence scoped by subsystem, environment variables, file locations, manifest compatibility, output/JSON contracts and command-specific exit codes.
- TUI routes, contextual keys, responsive/plain modes, help/Guide, confirmation, cancellation, write shielding, logs and update-check behavior.

Extract command/profile/skill/schema facts where reliable or validate curated content against source. The existing CLI uses Go standard-library flag parsing and custom dispatch; do not add Cobra solely for reference generation. No fabricated unified configuration schema: distinguish actual parser contracts, JSON files, profile frontmatter and environment behavior.

### FR-06 — Architecture, operations and security

Document actual CLI/TUI/appops/core boundaries, embedded/disk source resolution, skill loading owned by clients, delegation lifecycle and permission limits, fleet rendering versus model execution, MCP configuration versus connectivity, persistence/transactions, execution/check evidence, recovery and pair identity.

Use a small C4-style context/container overview plus focused Mermaid-source flows for installation/recovery, optional delegation, MCP transaction and execution evidence. Publish reviewed static image projections and accessible text equivalents; routine builds must not add a browser/diagram runtime. Maintain source/render provenance together. Do not diagram instruction guidance as an enforced runtime service.

Troubleshooting distinguishes application errors, client behavior, provider availability/authentication and model behavior. Include offline diagnostics, stale plans, unsupported runtime/target, permissions, rollback failures, configuration recovery, missing/mismatched companions and known limitations.

Security explains exactly which files can change, backups and their limits, credential handling, provider/MCP disclosure risks, permission models, external-process boundaries and safe execution recommendations. Instructions, deny rules, runs and reviewer context are not an OS sandbox. A backup is not a guarantee against every partial failure.

### FR-07 — Existing-document migration

Implement the file-level [migration matrix](evidence.md#documentation-migration-matrix). Inventory links and important heading anchors before moves. Preserve historical specifications and evidence paths; do not silently modernize accepted history.

For migrated active guides, keep minimal old-path compatibility pages with retained high-value anchors and direct canonical links; no copied manual. Stub-only files contain an optional title/status, canonical links and retained anchors, not procedures or code fences. Engineering guides retaining substantial audit/trace material are explicitly classified retained engineering documents, not stubs. Supply static website redirect pages for published URL aliases only where an old web URL exists. GitHub Markdown links cannot rely on HTTP redirects. Track deliberate archive-only destinations separately from broken links.

`docs/README.md` becomes the repository index for public entry points and engineering records, not a second public manual. Profile instructions and portable skill sources stay where installers embed them.

### FR-08 — Website experience and publication

The availability observation below is historical (2026-11-14); see [§10's dated observation](#10-readme-completeness-amendment-2026-11-16) for the now-reachable website. This pointer changes no historical deployment or approval evidence.

Use the proposed platform ADR: same-repository Astro Starlight, static GitHub Pages, framework-default navigation/theme/search. Include responsive navigation, TOC, copyable highlighted code, edit/repository/issues/releases links, dark/plain-accessible presentation, page navigation, descriptive metadata, sitemap and stable canonical URLs.

Default proposed project URL: `https://nawodyaishan.github.io/agentic-sdd/`; the public probe returned 404 on 2026-11-14 and repository metadata reported `has_pages:false`. This establishes current unavailability, not publication history or private settings. No custom domain assumed. Validate project base paths in HTML, search assets, sitemap, evidenced redirects, Markdown and AI indexes.

PRs build without deployment credentials. Authoritative evidence uses reviewed-base tooling and treats candidate source as data until isolated execution; a candidate-authored check is not trusted merely because it is green. Main-branch publishing requires separately authorized Pages configuration and deployment, protected-environment and repository-policy preflight, and same-revision artifact validation. A failed build never replaces a healthy site; deployment failures remain visible and retryable. Retained known-good artifacts permit rollback; after expiry rebuild a known-good source/dependency identity in a fresh workflow run. No external preview service initially.

### FR-09 — Minimal TUI documentation access

Select **Option A**: extend existing Guide/Help with explicit external documentation links and always-visible/selectable URL text. Keep `g`'s existing online-README behavior, keys `1–5`, existing screen routes and transaction shielding.

Offer getting started, CLI/TUI reference, skills/profiles, fleet/model configuration, MCP, recovery/troubleshooting, architecture and GitHub through one small registry. Reuse the current explicit browser-opening mechanism, fixed HTTPS allowlisted destinations and no shell interpolation. Failure must leave the URL available for manual copy; do not require a system clipboard or turn failure into application exit. Offline help still explains safety, local help commands and installation recovery without fetching docs.

Option B (additional context-sensitive links) and C (embedded Markdown manual) are deferred. No automatic browser action, new network preflight, browser-like interface, new write path or breaking CLI change. Final launch can omit TUI code if the owner explicitly removes this optional sub-scope before combined approval; the rest remains independently useful.

### FR-10 — AI-readable and discoverable content

Canonical Markdown, useful headings, consistent concepts, generated public indexes, version/source identity, canonical URLs and normal SEO are the baseline. Generate an optional `llms.txt` link index and plain Markdown page outputs from the same allowlisted source set; no separately edited prose.

Label `llms.txt` an emerging convention, not an IETF/W3C standard or universal assistant contract. Defer `llms-full.txt` until measured retrieval need, and custom MCP docs resources, semantic/hybrid retrieval and chatbot/RAG until a demonstrated problem justifies them.

Repository `AGENTS.md` may gain a short pointer to canonical docs and evidence lookup, not copied usage text or new authority. Context7 submission/indexing is optional, public-only and separately authorized; no guaranteed ingestion or version support. Agent Skills conventions describe client-consumed packages, not a claim that the website itself is a skill.

No private run logs, backups, homes, credentials, unreviewed research or raw forensic traces enter pages, search, Markdown exports or AI indexes. External pages remain untrusted data.

### FR-11 — Testing documentation integration

Use [the existing test report](../../docs/testing/test-cases-report.md) as an important engineering inventory, with actual source/test/config cross-checks; never as proof of executed coverage. Keep its detailed inventories and any measurement artifacts engineering-side.

A public contributor testing page explains organization of both Go modules, toolchains, the 11-package default core gate, explicit Python patterns, fixtures/isolated homes, opt-in TUI/PTY/container lanes, sanitized environments, optional local statement-coverage measurement, CI and the 008 fence. A documentation command is not a measured result. A user-facing verification explanation distinguishes installer verification from feature testing and model behavior.

Coverage documentation must identify command, packages, build tags, toolchain, revision, selection, skips and measurement type. Declared test counts, statement coverage, branch coverage, manual observations and untested workflows must not be conflated. Do not invent missing percentages or change gates/skip semantics.

Maintain a capability → example → test/evidence mapping. Add documentation-only example checks and reference-parity tests; record underlying missing product tests with their existing owners. Refresh inventory pointers when test organization/selection changes. Use the separate testing initiative's supported regeneration method if supplied; none was found during this audit. Otherwise retain the snapshot label and report drift to its owner, rather than invent a competing generator.

### FR-12 — Sustainable maintenance and governance

Fast deterministic PR checks: Markdown/heading/frontmatter lint, internal links/anchors, source-reference parity, public export allowlist, site build and a small offline example suite. Release checks verify docs against the release revision, naming/compatibility and release notes. Optional scheduled external-link/dependency checks do not gate routine PRs on remote services.

Changes to commands, TUI navigation, schemas/config parsers, skills, profiles/routing, MCP, manifests/restore, clients/installation, security and user-facing errors trigger one concise docs-impact report. A PR answers “updated”, “no public impact” with reason, or “tracked gap” with owner. Avoid blanket CODEOWNERS barriers and mandatory unrelated docs edits; deterministic parity failures cannot be waived by an unchecked declaration.

Use lightweight maintainers' ownership, lower-kebab-case page names, one H1, task-oriented headings, fenced language labels, safe placeholders, prerequisites/effects/expected outcomes and source/compatibility notes. Generated blocks name their producer and are not hand-edited. Human review checks implementation truth, security and external facts in AI-drafted updates.

Measure reliability rather than promise zero recurring work. Acceptance includes proposed CI budgets and observation methods in [plan.md](plan.md), no invented baseline or automatic stale-page deletion.

## 5. Nonfunctional requirements

- **NFR-01 Portability:** public instructional prose remains understandable as Markdown without MDX components; keep custom components to demonstrably necessary presentation.
- **NFR-02 Maintainability:** no product Go dependency for website infrastructure; one isolated pinned Node workspace/lockfile; no mandatory AI keys, hosted search, analytics or CMS.
- **NFR-03 Accessibility:** target WCAG 2.2 AA for authored content; keyboard/manual checks and automated audits, not an unsupported certification. No essential meaning conveyed only through color, motion or images.
- **NFR-04 Security:** explicit public-source allowlist, least-privilege publication, safe examples, no execution of untrusted PR content with deployment secrets, no hidden model/network calls.
- **NFR-05 Reliability:** clean checkout generation/build reproducible at a pinned toolchain; deterministic internal checks; visible blocked/failed/not-run outcomes. Remote dependency installation is separate from offline validation.
- **NFR-06 Compatibility:** canonical/legacy commands, state/environment names, historical paths and current core gates preserved. Do not broaden support platforms or client parity by documentation alone.
- **NFR-07 Performance:** use static assets and framework defaults. Establish measured build/page-weight/search baselines in B1/B5; proposed acceptance budgets are not claims about existing performance.

## 6. Acceptance and objective verification

| ID | Accepted outcome | Verification / task owner |
|---|---|---|
| AC-01 | README accurate, concise and useful to first-time readers | README structure/link audit; every command in reference; cold-reader walkthrough; T04/T12 |
| AC-02 | A newcomer can choose, install/preview and verify without accidental live writes | Isolated-home run: preview preserves destination digests, writes stay in owned fixtures, no real-home access; T03/T12 |
| AC-03 | Skills-only and optional orchestration cannot be confused | Two distinct entry paths; consent/provider requirements and learner/direct-fix branch reviewed; T03/T05 |
| AC-04 | All implemented public capability families have discoverable L/C content | Sitemap/evidence rows reconciled; CLI/config/skill/profile parity checks; T02/T05–T07 |
| AC-05 | CLI and TUI usage matches actual dispatch, defaults and controls | Help/source comparisons; route/key fixtures; isolated examples; T02/T06/T11 |
| AC-06 | Architecture matches implementation | Source-linked diagrams and maintainer review against CLI/TUI/core flow; T07/T12 |
| AC-07 | Security/operations limits explicit | Negative-content review for sandbox/auth/connectivity/release overclaims; recovery examples; T07/T12 |
| AC-08 | Site builds and publication workflow is safely deployable | Clean pinned build; trusted-runner/permission/preflight/base-path checks; failed-build retention and expired-artifact rollback scenarios; actual live 200/assets/search/sitemap plus nonexistent-path 404 only after authorization; T01/T10 |
| AC-09 | One canonical Git source per public document | Content-ID/source registry; no duplicate body/stub drift; GitHub rendering review; T01/T08 |
| AC-10 | References and links are automatically validated | Intentional broken flag/anchor/profile/URL-base mutations fail checks; no external API dependency; T02/T09 |
| AC-11 | Feature changes can carry proportionate docs updates | Representative CLI/skill/schema/TUI path changes produce one useful impact report; no-impact reason example; T09 |
| AC-12 | History remains accessible and distinct from product support | Migration matrix complete; preserved paths/anchors; history absent from public search/exports; T08/T12 |
| AC-13 | Agents can retrieve authoritative docs without a proprietary service | Download Markdown/index locally; source/version/canonical links checked; private-fixture export rejected; T09 |
| AC-14 | TUI access evaluated; implementation conditional on retained approved sub-scope | Required A/B/C comparison at T12; if retained, T11 proves fixed-link/browser-failure/narrow/offline and navigation/write safety. If removed, record the explicit owner exclusion, never a pass for unrun code |
| AC-15 | Content and release process can grow without a second authority | Contributor/ownership/version policy, add-page fixture and release-sync checklist; T01/T09/T10 |
| AC-16 | Burden appropriate for a small OSS project | Plan §§6–7 proposed build/size/search budgets measured or honestly not_run; no mandatory AI/search/CMS/custom retrieval service; T09/T12 |
| AC-17 | Testing docs integrate existing initiative honestly | Report/source/gate crosswalk; counts ≠ coverage ≠ passes; authorized measurement provenance or not_measured with snapshot/reason/owner; selector drift observation; T06/T12 |
| AC-18 | Migration and examples cannot silently imply unsupported behavior | Old-path and unsafe-example negative fixtures; evidence.md P-01–P-04 dispositions; source/release labels; T02/T08/T12 |

Documentation acceptance is complete when L/C content and these checks are satisfied; A/F backlog content is not silently included. B5 establishes core-system readiness, not whole-feature completion if the optional TUI implementation remains in approved scope. In that case AC-14's code portion remains pending until B6 is reviewed; otherwise record the owner's explicit exclusion. Live publication can remain `not_run` with a validated workflow until independently authorized; do not call the site “published” in that state.

## 7. Risks, dependencies and unresolved operations

Risk treatment and objective closure live once in [plan §10](plan.md#10-risk-register-and-closure-criteria). Every row has an owner, affected acceptance criterion, evidence and a next check; a selected design is not an executed validation.

- **R01 Release truth:** v0.11.0/tag/cask metadata verified; archive/runtime/live-install unverified, not falsely “unsupported.” Existing feature approvals/ledgers stay unchanged.
- **R02 Publication:** current 404/`has_pages:false` observed; owner-only settings, permissions, protection capabilities and live activation need separate evidence/authorization.
- **R03–R04 Build/link/supply chain:** standard loader, one URL owner, narrow AST link transform, exact pins, static diagrams and trusted/offline candidate execution; in-repo integration awaits B1/B5.
- **R05 Migration/freshness:** committed anchor map/stubs, one canonical source and reviewed-source fingerprints replace unverifiable ancestry assumptions.
- **R06 Test evidence:** no measured coverage/regeneration tool exists; optional measurement contract and testing-initiative owner prevent fabricated results or duplicate inventories.
- **R07–R09 Usability/search/performance:** concrete task cards, query set and measured budgets; no invented baseline, participant success or accessibility certification.
- **R10 Optional TUI:** fixed URLs, explicit action, nil-opener/full-URL failure cases; no new routes, clipboard requirement or live fleet/source-write changes.

## 8. Delivery boundary

Implementation proceeds in the batches in [tasks.md](tasks.md), after one combined approval and explicit selection of the first batch. Each batch stops for human review. No completed specification is edited by this drafting assignment.

## 9. Approval

**Combined decision:** **Approved, 2026-11-14.**

- **Human approver:** repository owner in this conversation; no additional identity asserted.
- **Approved revisions/scope:** presented spec r2, plan r3 and tasks r3, including the supporting records and isolated parallel schedule.
- **Decision evidence:** after the r3 review-set summary and the stated combined-approval/B1 boundary, the owner instructed: “start implementation using free, gemini 4 hydrogen, deepseek 4 flash max models impleemnt sub agents”. This is recorded as approval of the presented set and authorization to start the named first batch, not authorization for all six batches.
- **Implementation authorization:** **B1/T01–T02** was authorized as four isolated writers in W1, parent integration, four isolated writers in W2, integrated verification and completion review. **B2/T03–T04** is authorized under the B2 amendment in `tasks.md`, with isolated W1/W2 writers and parent integration.
- **Model scope:** free Console workers, catalog-confirmed 9Router Gemini 4 Hydrogen and DeepSeek Flash Max implementation workers. Record actual models/variants and any failures; no silent paid/cross-provider fallback.
- **Review and B2 direction:** the owner directed independent Gemini 4 Hydrogen and DeepSeek Flash reviews of B1 and use of their recommendations as the next-batch amendment to implement. Gemini returned Ready; DeepSeek returned Needs changes with B1 provenance/baseline/CLI findings. The immediate review-state, baseline and provisioning findings were corrected; remaining provenance/CLI corrections are folded into the B2 amendment. B1 human acceptance is not separately recorded.
- **Owner concurrency extension (2026-11-14):** the owner then directed: "start B2 B3 B4 B5 B6 all parallelly using max as can sub agents lanes implement using models deepseek max, opencode free, and gemini 4 hydrogen high", followed by "dont use claude models". This is recorded as authorization to start **all five remaining batches concurrently for implementation**, waiving only the recorded reviewed-batch start order. It does not accept any batch, does not change spec/plan/tasks scope or acceptance, and does not authorize commit, push, tag, release, Pages enablement, deployment or real-home apply. Batch state and the exact lane fences are in `tasks.md` §Owner concurrency override. Free Console quota was exhausted during execution; affected lanes were retried on the owner-named Gemini 4 Hydrogen High and DeepSeek Flash Max lanes, and no Claude model was used.
- **Still separate/not authorized:** real-home apply, provider authentication/configuration changes, release/tag creation and any rollback rehearsal against live Pages. Commit, push, Pages enablement and deployment were separately authorized on 2026-11-16.
- **B1–B6 acceptance (2026-11-16):** the owner accepted all six batches **with named conditions**. Accepted evidence: `main` `cec4b6b` with the docs workflow reporting an authoritative pass over the full adopted check list (build:site, docs:check, docs:check-built, export write and drift check, Node suite, reference parity with a pinned Go builder), the published site at `https://nawodyaishan.github.io/agentic-sdd/`, 39 registered pages, rendered diagram projections, and `llms.txt`/`index.json`/Markdown twins served from the live artifact. No batch is accepted unconditionally.
  - **Blocking publication:** none open. The live site, canonical URLs, sitemap, search bundle, 404 `noindex`, favicon, social card and JSON-LD were observed on 2026-11-16.
  - **Accepted limitations:** no fresh-user usability observation, no screen-reader run, no real-browser/PTY interaction, no real-home install, no Windows support claim, and no analytics or ranking measurement.
  - **Deferred conditions with owners:** six blocked adversarial cases (B4 coverage, owners named in `tasks.md`); live rollback rehearsal; release-documentation sync comparison; reference parity depends on the pinned Go builder image and is unproven on a cold runner until observed.

### README amendment decision — approved 2026-11-16

- **Requested drafting scope (2026-11-16):** the owner requested a Feature 020 amendment to fully update README, parallel DeepSeek Max research, Exa best-practice research, and corresponding amendments to spec, plan and tasks.
- **Owner clarification:** the new documentation website, `https://nawodyaishan.github.io/agentic-sdd/`, must be **mainly highlighted**, not treated as an optional footer link. This sets the proposed amendment's primary navigation priority; it is not authorization to implement or deploy.
- **Additional owner scope:** plan personal branding on the documentation website so the creator/maintainer is identifiable. This adds bounded attribution and profile navigation, not a full brand redesign or authorization to change the live site.
- **Proposed review set:** spec r3 §10, plan r4 §12 and tasks r6 B7/T13–T16. The original combined approval and subsequent implementation authorizations above remain historical authority for their actual scope.
- **Decision:** approved by the repository owner in this conversation on 2026-11-16 for the presented spec r3 §10, plan r4 §12 and tasks r6 B7/T13–T16 amendment set.
- **Decision evidence and execution authorization:** after the amendment summary and approval checkpoint, the owner instructed “start implementations with deepseek max sub agents”. This approves the presented set and authorizes B7 implementation, focused verification and completion review using `opencode-go/deepseek-v4.1-flash#max` child sessions with parent integration. No silent model/provider fallback.
- **Boundary:** B1–B6's separately recorded acceptance with conditions is preserved; B7 neither accepts nor reopens those batches. This authorization does not itself include commit, push, deployment, real-home apply or provider authentication. Earlier separately scoped live/Git authorizations do not automatically extend to these B7 changes.
- **B7 acceptance and live-operation authorization (2026-11-16):** after B7 integration, verification and completion review, the owner instructed “B7 accepeted commit and Push it” followed by “also deploy and tag new release”. This records human acceptance of B7 with its named residual limits, and authorizes atomic commit/push of B7 changes, observation of the `push: main` Docs deployment workflow, and creating/pushing the next release tag (`v0.12.0`). Pre-existing `.nojekyll`/Pagefind findings and concurrent Feature 021 files remain outside B7 scope; no real-home apply or provider authentication is authorized.
- **Narrow verification correction (2026-11-16):** after the frozen baseline suite exposed the old CLI-reference test requiring a commit hash in generated prose, the owner selected “Include correction (Recommended)” to the explicit request to align that test with metadata provenance. This authorizes a scoped `website/tests/reference-cli.test.mjs` correction, preserving real-binary/commit checks and removing no test; no generator/product behavior or gate change is authorized.
- **Personal-link validation correction (2026-11-16):** the owner selected “Retain link and fix checker (Recommended)” after the built checker rejected the approved personal website outside the docs project base. This authorizes a narrowly scoped `website/scripts/check.mjs` classification correction and built-link regression fixtures: only the exact approved maintainer website is an external-link exception; wrong documentation bases, unrelated URLs, unsafe schemes and canonical validation remain enforced. No broad suppression, new manifest schema or live action.

## 10. README completeness amendment (2026-11-16)

### Outcome and boundary

Fully refresh the repository's public front door against the current product and canonical documentation, with **[the documentation website](https://nawodyaishan.github.io/agentic-sdd/) as the primary destination**. “Fully” means every README section and example is audited, every current public capability family has an understandable route to its documentation, and obsolete claims and legacy-guide detours are removed. It does **not** mean copying all 39 public pages, command flags, skill/profile catalogs or engineering evidence into README.

This extends FR-03 and the README portions of FR-01/02/04/05/07/12. Preserve the original feature outcome and acceptance criteria. The approximately 150–220-line scan target remains advisory: completeness, safety and readable navigation take priority, with any substantial deviation explained during review. No new website platform, feature behavior, distribution target or release promise is introduced.

The owner's follow-up also adds **personal creator/maintainer visibility on the website**. This narrowly extends the existing “no brand redesign” boundary: allow factual attribution, a small shared site credit and useful profile links, while preserving Agentic-SDD's project identity, documentation-first navigation, framework theme and contributor recognition. No portfolio rebuild, new logo/photo, biography, testimonials, employer endorsement, analytics or social widgets.

### FR-13 — Complete, current and safe README

1. **Orientation and choice:** explain what Agentic-SDD is, the problem, intended users, practical benefits without unmeasured claims, and when a direct fix needs no feature paperwork. Present one main agent and portable skills as the baseline; identify optional teaching and explicit delegation without making either compulsory.
2. **Whole-product discovery:** give a short purpose and direct canonical destination for core skills/workflow, CLI/TUI, optional OpenCode profiles/orchestration, fleet/model configuration, MCP configuration, execution/review evidence, architecture, security/limitations, recovery, contribution/testing and releases/migration. Installed destinations are not evidence of equal client/model runtime support. No full command or profile catalog belongs here.
3. **Usable first success:** distinguish obtaining binaries from installing client assets. Qualify macOS distribution versus source builds and actual toolchain requirements; keep each path's executable names consistent. Lead with skills-only, isolated-home preview, expected non-mutation and a clear stop point. A fixture install, if shown, is a separately labeled optional write to a uniquely owned temporary directory, with scoped cleanup. Never use a shared fixed `/tmp/test-home` or silently fall back to the real home.
4. **Effects and evidence:** state TUI versus CLI pack defaults and label any write or network step where it occurs. Scope preview-first/no-write claims to the actual command and distinguish managed-destination non-mutation from operation logs, explicit trace settings, fixture writes and live installation. Do not imply that ordinary CLI preview creates apply/restore logs, or that every fleet/execution command is read-only. Separate checkout behavior, published artifacts, validation actually performed and unknowns; preserve existing experimental exclusions and instruction-versus-sandbox limits.
5. **Website-first navigation and compatibility:** prominently display a normal text link labeled **Documentation** to exactly `https://nawodyaishan.github.io/agentic-sdd/` immediately after the short product introduction, before the first H2, installation command and extended marketing/workflow prose. It is the primary call to action, not only a badge, image, footer item or “explore further” bullet. Provide a compact website-first start/reference route and make the site the first destination in the further-reading section. Verify selected deep links under the `/agentic-sdd/` base. Retain a clearly secondary relative source fallback to `website/src/content/docs/index.md` for repository browsing or site outage. Preserve required historical README fragments and both guide-tested everyday prompts; retain the sole migration-table owner at `releases/migration.md`. Legacy paths remain compatibility entry points, not the primary new-reader route.
6. **Complete editorial pass:** audit all headings, prose, command blocks, badges, links and optional visuals, not only the opening paragraphs. Keep release/CI/license badges meaningful and qualified, no essential image-only instructions, and no unsupported performance, learning, portability, security or provider-availability claims. A material correction that needs product-code changes is a separately owned finding, not permission to change behavior or weaken a check.

### FR-14 — Visible, factual personal branding on the documentation site

1. Show a concise creator/maintainer byline on the homepage near its introduction, plus consistent shared credit on public documentation pages. Proposed wording: **“Created and maintained by Nawodya Ishan”**, linking the name to `https://github.com/nawodyaishan`. Keep project title, setup links, search and normal navigation dominant; attribution must be visible text, not image-only or metadata-only.
2. Use one shared identity owner for repeated site attribution. Offer the evidenced personal website `https://nawodyaishan.github.io/` as a secondary “Personal website” link, alongside the GitHub profile and an existing contribution route; do not import personal contact details or unrelated profile data. Do not infer exclusive authorship of contributions, guarantees of personal support or commercial endorsement.
3. Preserve framework footer utilities, edit links, page navigation, responsive behavior, keyboard focus and light/dark presentation. No tracking, external avatar dependency, custom font, new runtime service or essential image. Public exports retain authored homepage attribution as text without copying a credit paragraph into every source page.
4. Reconcile the narrow public “site is unpublished/not live” statements that contradict the new website-first entry. Update only current publication-status prose in the homepage, compatibility, known-limitations and docs-contribution pages against dated read-only observations. Preserve historical evidence, release/runtime qualifiers, `unreleased` applicability and unreviewed status; do not infer or modify deployment settings.

### Additional acceptance

| ID | Accepted outcome | Verification / owner |
|---|---|---|
| AC-19 | Every existing README section has a keep/rewrite/move/remove disposition and every FR-13 capability has a purpose and direct current-doc route | Source-linked section/capability map, no uncategorized section or missing family; T13 producer, T16 verifier |
| AC-20 | Installation and first workflow are internally consistent, isolated and effect-labeled | Check executable paths, toolchain facts, pack defaults and fixed argv; run permitted synthetic-home examples, compare managed-destination digests and account for logs separately; T14 producer, T16 verifier |
| AC-21 | README routes readers to canonical content without breaking compatibility | Positive link/fragment and required-prompt checks; missing target/removed anchor mutations fail; no primary onboarding through legacy manuals; primary site has a dated availability observation and relative source fallback; T14/T16 |
| AC-22 | Whole-product overview is truthful without becoming a second manual | FR-13 coverage/claim review, release-versus-checkout qualifiers, one migration owner, no duplicated full reference; author walkthrough labeled separately from unfamiliar-user evidence under plan Q1; T14/T16 |
| AC-23 | Refresh is maintainable without relaxing existing QA or claiming old batches accepted | Focused entrypoint/static checks and relevant negative fixtures, recorded input identity/results/omissions, unchanged core fence and B1–B6 acceptance; T16 |
| AC-24 | The documentation website is the README's main highlighted destination | Visible Markdown link labeled Documentation to the exact homepage immediately after the introduction and before the first H2/install command; website-first getting-started/further-reading links with a secondary source fallback; removal, wrong base or moving the sole link to the footer fails deterministic checks; T14 producer, T16 verifier |
| AC-25 | Readers can identify and reach the creator/maintainer without hunting through source or a license | Visible homepage byline and shared site credit use the evidenced name, GitHub profile and secondary personal-site link, with a contribution route; T15 producer, T16 verifier |
| AC-26 | Branding stays consistent, accessible and subordinate to documentation tasks | One shared identity owner; homepage and representative deep-page HTML include text credit; mobile/light/dark/keyboard and existing footer utilities checked, source/Markdown attribution remains readable; T15/T16 |
| AC-27 | Attribution adds no unsupported claim or privacy/runtime dependency | No fabricated biography/support promise/endorsement, tracking, remote avatar/font fetch or contributor erasure; identity/link mutation fixtures and protected-tooling checks; T16 |
| AC-28 | The website-first README does not lead to contradictory current publication guidance | Named public pages distinguish observed live availability from historical 404, release applicability and unrun checks; no blanket reviewed/released/deployment-validation claim or settings change; T15/T16 |

**Dated website observation (2026-11-16):** a read-only fetch of the owner-supplied homepage succeeded and returned the Agentic-SDD documentation, navigation and getting-started/reference links. This supersedes the earlier 2026-11-14 404 only for current homepage availability; preserve the historical observation in FR-08/evidence. The fetched page still says public Pages deployment is “not yet verified”; that text is stale relative to this observation and must not be copied into README. A reachable homepage is not proof of complete asset/search/accessibility correctness, release/runtime support, human batch acceptance or permission to modify the deployed site.

A separate read-only fetch of [First five minutes](https://nawodyaishan.github.io/agentic-sdd/getting-started/first-five-minutes/) also succeeded at the correct project-base deep URL. This supports that proposed first-action link only; no full-site availability or browser/asset audit was performed.

**Identity evidence (2026-11-16):** the public [GitHub profile](https://github.com/nawodyaishan) displays “Nawodya Ishan”, handle `nawodyaishan`, and links `https://nawodyaishan.github.io`. These support the proposed display name/profile/personal-site destinations only; no biography, private data, copyright reassignment or endorsement is inferred.

The amendment's combined approval and B7 authorization are recorded in §9. B7 is separately authorized work, not an automatic continuation of the older concurrency override. Its completion satisfies its own acceptance only; it cannot close unrelated diagram, migration, CI-trust or whole-feature usability findings.
