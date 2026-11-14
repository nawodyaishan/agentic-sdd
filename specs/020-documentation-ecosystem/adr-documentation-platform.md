# ADR-020 — Same-repository Markdown, Starlight and GitHub Pages

**Status:** proposed r2, 2026-11-14; accepted only through [spec §9](spec.md#9-approval). No implementation or publication.
**Convention:** no repository ADR directory/template was found. Decisions normally live beside feature plans/studies. This explicitly requested ADR therefore stays feature-local; it does not establish a second approval ledger or a new global ADR ceremony.

## Context and drivers

The [audit](evidence.md) finds valuable Markdown, fragmented current references and engineering history that must remain stable. A small Go project needs human navigation/search and AI-readable sources without a runtime backend, editor service, duplicated prose or mandatory external AI. Diátaxis provides the content model, not the website framework.

Evaluate source location separately from renderer. A same-repository static website could use several frameworks; “Markdown versus website” is a false choice when Markdown remains canonical.

## Alternatives

Ratings are qualitative engineering judgments for this repository, **not benchmarks**. Free hosting is subject to repository/plan eligibility and published quotas; no current account settings were inspected.

| Option | Git authoring / contributor experience | Setup / build / ongoing maintenance | Go-repository fit / hosting / lock-in | Decision |
|---|---|---|---|---|
| GitHub Markdown only | Best portability; PRs and raw files; no extra local toolchain | Minimal setup; manual structure, weak integrated journey/search UX | Excellent Go fit; GitHub UI host, no framework dependency | Retain as fallback, insufficient for requested public navigation/search/discoverability |
| Same-repo static site (pattern) | Atomic code/docs PR; source links and edit buttons | One isolated build/deploy workspace | Static hosting; low lock-in when prose stays Markdown | Select source/deployment pattern |
| Separate docs repo | Git-native but split reviews, source versions and relative links | Second access/release/workflow lifecycle and sync | No Go runtime impact; more drift/ownership cost | Reject: no independent publisher/access requirement |
| GitBook + Git Sync | Git sync and visual editing; bidirectional editor-originated changes | SaaS provisioning/auth/sync rules; vendor handles presentation | Hosted proprietary service; current plan costs/quotas unverified; stronger vendor dependence | Reject absent nontechnical editor requirement |
| Astro Starlight | MD/MDX and edit links; standard content collections | Small Astro config; Node build/lockfile; default Pagefind/theme | Isolated from Go; static Pages; low prose lock-in | Select renderer |
| Docusaurus | Markdown/MDX, Git PRs, React customization | More configuration/plugins; built-in version snapshots increase maintenance | Static Pages; Node/React tooling not otherwise needed | Credible runner-up if maintained multi-version docs become necessary |
| MkDocs Material | Markdown/YAML, Git PRs; familiar to Python contributors | Python environment/plugin pinning; built-in search; `mike` for versions | Static Pages; extra site toolchain despite stdlib Python test scripts | Credible runner-up, no demonstrated advantage over Starlight here |

| Option | Search | Accessibility | SEO / external discovery | Portability / AI readability | Documentation versions / extensibility | Automation |
|---|---|---|---|---|---|---|
| GitHub Markdown | Repository search, not a dedicated user-goal index | GitHub shell; author content still matters | Repository pages/raw links; little metadata/control | Highest plain-Markdown portability | Git refs only; limited presentation extension | Local lint/link/example checks possible without site |
| Same-repo static pattern | Framework-dependent | Theme plus authored-content testing | Canonicals, sitemap and metadata under project control | High if generated outputs derive from source | Depends on renderer; atomic source versioning | CI build, link and generated-parity validation |
| Separate repo | Renderer-dependent | Renderer-dependent | Good site potential; cross-repo version identity harder | High if plain Markdown, poorer source proximity | Independent versions can drift from Go releases | Requires cross-repo synchronization/contract checks |
| GitBook | Built-in service search | Vendor UI + content; no audit performed | Hosted metadata/discovery features | Sync/export helps; rich blocks can reduce raw Markdown parity | Vendor-managed capabilities/plan constraints | Sync/webhooks; service dependency for routine publication |
| Starlight | Pagefind enabled by default; static client index | Accessible defaults, keyboard/theme features; no conformance claim without testing | Astro static HTML, sitemap/metadata configuration | Plain Markdown supported; avoid routine MDX/custom components | No core Docusaurus-style version manager; Astro integrations/customization available | Build/check/lint, deterministic content/export hooks and static hosting |
| Docusaurus | Search integration/plugins or hosted DocSearch; not free hosted search guaranteed | Framework defaults; author/plugin audits necessary | Static metadata/canonical/sitemap support | Good until JSX-heavy MDX becomes required | Built-in versioning; powerful React plugin ecosystem | Broken-link/build validation; additional plugin maintenance |
| MkDocs Material | Bundled client search | Theme features, authored-content testing | Static metadata/sitemap with configuration | Good; extension-specific syntax can reduce raw parity | `mike` adds versioning; Python/Markdown plugins | Strict build and plugins; isolated requirements lock needed |

No framework eliminates editorial work or verifies product truth. No search quality, mobile performance, SEO outcome or WCAG result was measured in this drafting session.

## Decision

1. Keep canonical public Markdown in the existing repository at **`website/src/content/docs/`**, Starlight's supported default content location.
2. Keep `docs/` for engineering records, testing inventories, vision, handovers, dated research and old-path compatibility pages. `docs/README.md` routes readers to canonical public pages and engineering owners; it is not a second public content catalog. Do not symlink/copy the whole directory into search.
3. Use **Astro Starlight**, default theme/navigation/TOC/code-copy/dark mode and **Pagefind**, with only necessary content/link/export integrations.
4. Build an isolated pinned Node workspace in `website/`; no root Go module/framework change.
5. Deploy a static artifact through GitHub Actions to proposed project Pages URL `https://nawodyaishan.github.io/agentic-sdd/`, after separate authorization. No custom domain, CMS, hosted search or preview vendor.
6. Make GitHub-rendered Markdown and raw-source links first-class. Generated HTML/Markdown twins/indexes are disposable projections; old paths are compatibility links, not competing edited bodies.

### Why use the default content directory?

Starlight `docsLoader()` loads `src/content/docs/`; its wrapper hardcodes the base and exposes ID generation, not an arbitrary base option. A worker proposed symlinking root `docs/`, but that adds contributor portability risk and would expose engineering/history unless filtered. A whole-tree prebuild copy also needs strict filtering and generated-file discipline.

Direct canonical public source in the framework's standard directory is the least complex option. Moving *current topic ownership* there in a later authorized batch is acceptable because old GitHub paths/anchors retain compatibility pages and history stays in place. Do not invent a custom Astro loader or duplicate source just to preserve the visual appearance of a top-level docs tree.

### Version and URL policy

Initial site follows `main` and visibly identifies **development/checkout documentation**, build commit, reviewed-source fingerprint and verified release applicability; never imply all `main` capabilities are in the latest released archive. Public v0.11.0 matches the investigated baseline at metadata level; archive/runtime/live-install checks remain distinct. Review freshness follows plan C1's explicit source hashes, not commit ancestry or automatic stamping. A compatibility page links release tags and their GitHub Markdown snapshots: pre-migration tags use their actual `docs/`/README paths; post-migration tags use `website/src/content/docs/`. Check the file exists at the tag rather than assuming one path works across the boundary. Unreleased command/reference sections carry a clear qualifier.

Do not duplicate the entire docs tree per spec/release initially. Reconsider versioned site builds only when multiple supported incompatible releases need maintained public manuals. A later custom domain change updates one URL config, canonical/sitemap/index projections and explicit aliases, not prose everywhere.

Project URL configuration has one owner in plan C1's manifest; Astro imports `site` for the origin `https://nawodyaishan.github.io` and `base` for `/agentic-sdd` (exact slash normalization verified during B1). All generated URLs use the same resolver; test root **and** non-root base builds. Relative-Markdown rewriting is a proposed, fixture-validated integration, not an assumed Starlight feature.

## Consequences

**Benefits:** one PR can update behavior and guidance; public search excludes history by construction; static low-cost hosting; serverless search; readable source; straightforward exit back to Markdown-only if framework adoption proves burdensome.

**Costs:** Node/lockfile/security maintenance, content migration and compatibility testing, authored-content accessibility review, a small reference/export generator and version labeling. Framework defaults reduce but do not remove these.

**Rejected for initial launch:** custom design system, blanket MDX conversion, maintained release snapshots, separate repo, GitBook, embedded TUI manual, local semantic-search CLI/vector index, Context7/MCP as a required dependency, runtime Mermaid/browser rendering in CI, or candidate-defined green statuses treated as authoritative review.

**Rollback:** disable deployment without removing source; republish an available validated known-good artifact using a new authorized workflow run, or rebuild its exact source/lock/tool identity after artifact expiry. Do not assume rerunning a completed Pages deployment ID always redeploys it. Keep canonical Git-source fallback links while the site is unavailable. Disabling/unpublishing Pages is an owner operation of last resort, not an automatic response. No product configuration/state rollback.

## Primary sources and applicability

Accessed 2026-11-14. Framework APIs must be rechecked against the exact versions pinned in B1; no package/action version is approved merely by appearing on a current upstream page.

| Source | Decision supported / caveat |
|---|---|
| [Diátaxis](https://diataxis.fr/) | Four user needs/forms; conceptual architecture, not required directories/tooling |
| [Starlight configuration](https://starlight.astro.build/reference/configuration/) | `docsLoader`, Pagefind default, sidebar/edit links |
| [Starlight loader source](https://github.com/withastro/starlight/blob/main/packages/starlight/src/loaders.ts) | Fixed `src/content/docs` collection base; primary author rechecked through Context7 `/withastro/starlight` |
| [Astro GitHub Pages deployment](https://docs.astro.build/en/guides/deploy/github/) | Static Actions deployment, project `site`/`base`, isolated project path |
| [GitHub Pages custom workflows](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages) | Build/deploy dependency, Pages/id-token permissions, protected environment; primary author fetched |
| [GitHub Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits) | Hosting has quotas; no promise of unlimited free service |
| [Pagefind](https://pagefind.app/docs/) | Static generated search index, not a network search backend |
| [Docusaurus versioning](https://docusaurus.io/docs/versioning) / [deployment](https://docusaurus.io/docs/deployment) | Built-in versions and project base URL; versioning carries maintenance cost |
| [Material search](https://squidfunk.github.io/mkdocs-material/setup/setting-up-site-search/) / [versions](https://squidfunk.github.io/mkdocs-material/setup/setting-up-versioning/) | Search and `mike` alternative; no paid-tier claims |
| [GitBook Git Sync](https://docs.gitbook.com/integrations/git-sync) | Bidirectional Git/editor workflow; pricing/quotas not established |
| [GFM specification](https://github.github.com/gfm/) | Markdown portability baseline; rendering details still differ |
| [llms.txt proposal](https://llmstxt.org/) | Emerging index/Markdown-twin convention; not universal adoption; primary author fetched |
| [Context7 adding libraries](https://context7.com/docs/adding-libraries) | Public repo indexing/configurable scope; submission is not retrieval guarantee; primary author fetched |
| [Google canonical URLs](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls) | Consistent absolute canonical/sitemap signals; no guaranteed search ranking |
| [Agent Skills specification](https://agentskills.io/specification) | Client skill package conventions, not a website standard |
| [MCP resources](https://modelcontextprotocol.io/specification/latest/server/resources) | Optional runtime resource mechanism; no need to ship a server for static docs |

External research informs framework/API decisions, never establishes Agentic-SDD behavior. Exa/Context7 use is optional at authoring time; neither is called by routine documentation build, lint, examples or deploy.
