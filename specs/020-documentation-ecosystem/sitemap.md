# Feature 020 — Public sitemap and page contract

**Status:** proposed r2. This is the page-scope owner, not authored website content.
**Root:** `website/src/content/docs/`. A row `reference/cli` means canonical `reference/cli.md` and public `/agentic-sdd/reference/cli/`. Routes are independent of feature numbers. Pages can initially be concise sections, not obligatory full-length manuals.

**Classes:** **L** essential initial launch; **C** required to complete current-feature coverage; **A** advanced/deferred; **F** optional future/deferred. Only L/C pages must ship in Feature 020. Do not publish empty A/F navigation entries.

**Types:** T tutorial (learn by doing), H how-to (complete a task), R reference (lookup), E explanation (understand). Landing pages route to these without repeating full instructions.

**Audience codes:** N new/learner; U current/experienced user; O optional OpenCode/custom-workflow user; M maintainer/contributor. Priorities and assumptions: [audiences.md](audiences.md).

**Dependency codes:** S source/evidence audit refreshed (T01); W website/URL contract (T01); G generated reference registry (T02); X safe example fixtures (T03/T06); M migration anchors (T08).

**Validation codes:** Lk source/site link and fragment check; P source/schema/help/catalog parity; X isolated deterministic example; A accessibility/keyboard/mobile review; R maintainer evidence review; V release/source-version qualifier; Q fixed search query set. Codes are requirements, not executed checks.

## Sitemap and initial acceptance inventory

| Page / class / type | Purpose and boundaries | Audience | Canonical evidence inputs, not a second prose source | Dependencies | Validation |
|---|---|---|---|---|---|
| `index` / L / landing | What/why/who/when unnecessary, capability/setup chooser, safe start | N/U | Vision §§1–8; current source inventory; README entry contract | S,W | Lk,A,V,Q; newcomer finds next step |
| `introduction/concepts` / L / E | Skill, profile, feature, direct fix, batch, evidence, consent; optional learning | N/U | Router/workflow policy; teaching skill; vision | S | R,Lk; no agent-count/runtime confusion |
| `introduction/clients` / L / R | Sole client/destination/support matrix owner; verified support versus portability and core/pack split | N/U/O | `skillsync/clients.go`, `sync.go`, source package tests; client official docs when version-specific | S,G | P,V; no assumed cross-client parity |
| `introduction/related-approaches` / C / E | Dated factual comparison and when existing workflow suffices | U/M | Audience primary-source links + actual Agentic-SDD contracts | S | R,Lk; no superiority/effectiveness claim |
| `getting-started/installation` / L / H | Prerequisites, macOS release/source build, Linux source, verification, terminal requirements | N/U | README valid sections; go.mod pair; Taskfiles; release config and actual release evidence | S,W,G | X,P,V,A; every write/network effect explicit |
| `getting-started/first-five-minutes` / L / T | Isolated home, version/help, preview and stop; expected observations | N | Current getting-started; CLI help/source; package fixtures | G,X | X,A; no real-home writes/provider keys |
| `getting-started/choose-setup` / L / H | Decision guide: CLI/TUI, core-only versus optional pack, model/provider prerequisites; links client matrix, does not duplicate it | N/U/O | CLI/TUI default flags; pack inventory; routing policy | S,G | P,R; TUI pack-on versus CLI pack-off clear |
| `tutorials/direct-fix` / L / T | One small example with proportionate scope/checks and optional teach | N/U | Router/direct-fix policy; debugging/implement/review skills | S,X | X,R; agent responses illustrative, not deterministic proof |
| `tutorials/first-feature` / L / T | One practical feature, spec→plan→tasks, combined approval, one batch, review/resume | N/U | Core stage skills and workflow policy | S,X | R,X; stop boundaries, no fabricated approval |
| `tutorials/specialists` / C / T | Explicit opt-in delegation, profile selection and a small model-routing configuration exercise | U/O | Orchestrate skill/refs; shipped profiles/presets; fleet parser | G,X; first-feature | P,X,R; no actual paid/model call in CI |
| `tutorials/research-and-debugging` / C / T | Named fact → source evidence → diagnose/check/review using same example | N/U/O | Research handoff, debugging/review skills; 018 static tests | S,X | R,X; fetched docs not test evidence |
| `how-to/skills` / L / H | Install/update core skills, choose supported client through actual available entrypoints | U/M | skillsync/source/selection; install guide | S,G,X | X,P; no invented CLI `--clients` flag |
| `how-to/orchestration` / C / H | Enable optional pack; bounded briefs, inspect results, manual fallback/continuation | O | Orchestrate references; routing policy; pack assets | G; concepts,profiles | R,P; local agents versus shipped profiles |
| `how-to/fleet` / C / H | Select explicit personal source, model/enablement changes, plan/output, apply/restore; preview-only TUI limits | O | `fleet-worked-guide.md`; fleet engine/CLI/appops/TUI; 015 limitations | G,X | X,P,V; no source-save or live-gate enablement |
| `how-to/mcp` / C / H | Inspect/plan/configure/recover three managed servers; auth placeholders/prerequisites | O | mcp-tools; mcp read/write engine, CLI and TUI | G,X; security | X,P,R; offline ≠ connected/authenticated |
| `how-to/execution` / C / H | Run init/status/check/record/clean, exact task brief, review package and recovery | U/O/M | execution-runs; `internal/execution`; CLI family help/tests | G,X; evidence explanation | X,P; private scratch, no sandbox/approval |
| `how-to/updates-and-recovery` / L / H | Skill/profile update, backup/restore, uninstall, binary rollback and config-store differences | U | Installer/restore source; MCP/fleet stores; wrapper/release records | G,X; files-and-backups | X,R,V; no “delete whole home” cleanup |
| `how-to/customization` / C / H | Existing-project adoption, local source, supported profile overrides, preservation of unrelated files | U/O/M | source resolver, skills/pack, fleet model-only render, actual parser contracts | G,X | P,X,R; no permission/model-policy rewrite disguised as config |
| `reference/cli` / L / R | Complete family/subcommand/flag/default table, output/exit contracts, numbered guide/launcher distinctions | U/O/M | `runEnv`, FlagSets, family help printers, CLI contract tests | G | P,X,Lk; exceptions (execution/fleet/MCP exits) explicit |
| `reference/tui` / L / R | Five routes, Fleet/MCP child screens, contextual keys, responsive modes, Guide/Help and update/browser effects | N/U/O | `app.go`, actions, Guide/Activity, config/launcher/opener, route/key tests | S,G | P,A,R,V; input focus/write shielding not overridden |
| `reference/configuration` / C / R | Configuration locations/precedence by subsystem, env vars, flags, JSON versus JSONC/frontmatter | U/O/M | FlagSets/env resolution; fleet parser; MCP target/parser; profile frontmatter | G | P; no fictitious unified schema; synthetic config parses |
| `reference/files-and-backups` / L / R | Managed paths/modes, embedded/repo source, skill formats 1–3, fleet format4, MCP state, execution scratch | U/M | skillsync manifest/restore; fleet/MCP manifests; execution workspace | G | P,X,R; digests not signatures, stores not interchangeable |
| `reference/skills` / L / R | Generated IDs/descriptions/category and links to normative SKILL.md/reference source | N/U/M | `skills-files/` + package/inventory/teaching registry | G | P,Lk; core/optional distinction, no copied normative bodies |
| `reference/profiles` / C / R | 14 shipped profiles, four optional presets, two fleet adapters, model pins/mode/tool policy and manual routing | O/M | pack frontmatter/routing/appops routing; install exclusion tests | G | P,V; model grammar ≠ availability; no local catalog leakage |
| `reference/fleet` / C / R | Schema v1 fields/grammar, sources/plans/drift/ownership/locks/restore/doctor result vocabulary | O/M | `internal/fleet/` parser/types/tests; CLI and appops | G | P,X,R; source-save deferred; TUI live default qualified |
| `reference/mcp` / C / R | Commands, target/auth/server schema, JSON states, apply/restore preconditions, exits and limitations | O/M | MCP types/parsers/status/transaction tests; family help | G | P,X,R; no arbitrary config target guarantee |
| `reference/execution` / C / R | Run/brief/record/check/review schemas, freshness, format/exit3 partial contracts and privacy | U/O/M | `internal/execution` types/parsers/CLI help/tests | G | P,X; claims versus verified evidence distinguished |
| `architecture/overview` / L / E | High-level context/components and who owns loading/execution/authority | N/U/M | Go modules, embed, appops/core, skills/client boundary | S | R,Lk,A; C4-style diagram + text |
| `architecture/interfaces` / C / E | CLI stdlib dispatch, TUI state/write host, typed seams, actual direct-engine paths, pairing/legacy forwarding | O/M | cmd/internal tuirun; buildinfo; CLI/ops/TUI sources | S,G | R,P; no false universal façade; fail-closed pair |
| `architecture/workflows` / C / E | Installer transactions/recovery; skill discovery; optional delegation and permissions; fleet/MCP separation | U/O/M | skillsync/appops; orchestrate; client discovery docs; fleet/MCP transaction source | S,G | R,P; 2 focused diagrams, no runtime-engine invention |
| `architecture/evidence` / C / E | Persistence, locks, run freshness/check evidence and review recovery | O/M | execution/workspace/checkpoint/checks/reviewpackage; 017 limits | S,G | R,P; inherited command permissions, interruption state |
| `operations/troubleshooting` / L / H | Symptom→safe diagnostic→expected result→recovery: install, companion, provider, MCP, fallback, permissions | N/U/O | Actual error strings/family docs; offline doctors and opener; known limitations | G,X | P,X,Lk; no connectivity claim from config probe |
| `operations/known-limitations` / L / R | Current implemented/verified/experimental/deprecated distinctions; no support from spec alone | N/U/M | Evidence inventory + owning feature ledgers + build/release identity | S | R,V,Q; historical native work clearly excluded |
| `security/trust` / L / E | Files/effects, credentials/provider/MCP privacy, instruction vs OS isolation, backup/argv/log limits, safe execution | All | Source refusal/transaction/process paths; current permission policy; sensitive-report channel in CONTRIBUTING | S,G | R,X; no invented private contact or security certification |
| `contributing/development` / L / H | Clone/toolchains/layout/build/hooks/source discovery; only Go needed for Go work, Node for site work | M | CONTRIBUTING, Taskfiles, modules, AGENTS | S,W | X,P; no real-home fixture writes |
| `contributing/testing` / L / H/R | Test execution/organization/fixtures/coverage/CI selection and interpretation; link full engineering inventory | M | Test report; suite README; selectors/checks/workflows | S,X | P,R; no broad discovery, no invented coverage threshold |
| `contributing/docs-and-changes` / C / H | Docs style/impact/ownership, add page/skill/profile, source validation, ADR/PR/issue/release contribution | M | AGENTS/workflow policy, current asset tests, this plan's governance contracts | W,G | Lk,P; no automatic installation/new skill count |
| `releases/compatibility` / L / R | Checkout vs release/platform/client support, changelog/release links and maintenance policy | All | release artifacts/evidence and owning records; no assumed latest tag | S | V,Lk; no unsupported OS/package assertions |
| `releases/migration` / L / H | Canonical/legacy names, preserved state/env, formula→cask only as evidenced; upgrade/uninstall/rollback links | U/M | 019 behavior/tests, 013 packaging records, actual tap/release evidence if available | G,M | P,X,V; announced wrapper-removal release only if established |

## Deferred inventory (not launch blockers)

| Page / class | Purpose / audience | Canonical evidence / dependencies | Validation if later authorized |
|---|---|---|---|
| `architecture/transaction-internals` / A | Deep lock/intent/digest failure paths for maintainers | Current engine/tests; accepted current explanation first | Maintainer source review and exact failure fixtures |
| `contributing/advanced-skill-workflows` / A | Detailed custom agent workflow composition | Normative skills/profile sources, no new runtime | Package/permission and example checks |
| `tutorials/infrastructure-change` / A | Longer IaC-flavored example | Demonstrated demand; isolated synthetic infra only | No live apply; scoped behavior test and boundary review |
| `tutorials/learning-lab` / A | Course-shaped independent extension | Beginner evaluation + optional teaching skills | Consent-based learner observations, no claimed outcome from generated code |
| Versioned site manuals / F | Multiple supported incompatible releases | Explicit support-policy need; version/source scheme | Cross-version links/canonicals and reproducible release builds |
| Embedded TUI documentation / F | Full offline manual viewer | Measured offline need, nested-module packaging/renderer design | No duplicated canonical source, terminal/input/security tests |
| `llms-full.txt` / F | Concatenated public corpus | Measured retrieval benefit and size budget | No private/history export; context-size and source parity checks |
| Context7/MCP documentation resource publication / F | Optional external/client retrieval | Separate authorization, public allowlist, real client demand | Public scope/indexing/version tests; no ingestion guarantee |
| Semantic/hybrid search or chatbot / F | Retrieval beyond Pagefind | Failed lexical-search evaluation and maintenance case | Held-out query metrics/cost/privacy evidence before infrastructure |

## Coverage, not page proliferation

Tutorials explain guided practice; how-to pages own action recipes; reference owns exact values; architecture owns why/boundaries. For example, the specialists tutorial links the profile reference and performs a small synthetic routing exercise; it does not repeat the full model table. Troubleshooting links recovery rather than copying transactions. Testing lives publicly as contributor guidance and engineering-side as detailed inventory/measurement evidence.

Every L/C page needs title/description, class/type, audience, reviewed-source fingerprint, release/platform applicability and safe examples where applicable. Keep metadata in plan C1's manifest; build identity and optional review-commit trace are not freshness proof. Source changes trigger one relevant impact check, not arbitrary age deadlines or automatic review stamping. Plan §7 owns exact usability/search fixtures; this page inventory does not duplicate them.
