---
title: Contributing docs and changes
description: Style, impact review, ownership, and contribution paths for docs, skills, profiles, ADRs, and releases.
---

This page is for contributors changing public documentation, skills, or profiles.
Setup, build, and toolchain guidance lives in [Development setup](./development.md);
test selection and result categories live in [Testing and verification](./testing.md).

## Docs style

Write for a newcomer who can read code but has not used this project before.
Keep each page to one job: tutorials teach by doing, how-to pages give recipes,
references list exact values, and explanations give reasons and boundaries.
Do not repeat a full reference table inside a tutorial; link the reference page.

- Every page starts with frontmatter containing exactly `title` and `description`.
- Back behavior claims with the owning source. Name the file or package as a
  plain code span, for example `cmd/agentic-sdd/main.go` or
  `internal/skillsync`, so a reviewer can find the evidence without guessing.
- Show only short, relevant excerpts. Mark any illustrative command output as
  illustrative; never present invented output as captured evidence.
- Relative `.md` links may target only listed launch and complete pages
  (for example [CLI reference](../reference/cli.md) or
  [skills reference](../reference/skills.md)) and repository source paths,
  which are written as plain code spans rather than links. Never link
  `docs/*.md` migration sources. Do not add hardcoded public-site navigation
  links: the site was observed reachable on 2026-11-16, and relative `.md`
  links keep the project base correct.
- Every table states its scope in the surrounding prose, and every diagram or
  image needs a text equivalent or caption. Do not rely on color alone.
- Keep generated sections intact. Text between markers such as
  `<!-- generated:reference/cli:start -->` and its matching end marker is
  produced from source by `website/scripts/reference.mjs`; hand edits inside
  the markers are overwritten and fail the parity check.

## Docs impact review

Before opening a pull request, map the change to the pages it affects:

| Source change | Likely page owner |
|---|---|
| CLI families, flags, defaults, exit codes | [CLI reference](../reference/cli.md) |
| Skill catalog, categories, install destinations | [skills reference](../reference/skills.md) |
| Pack profiles, presets, routing | [profiles reference](../reference/profiles.md) |
| Configuration locations, precedence, env vars | [configuration reference](../reference/configuration.md) |
| Backup formats, managed paths, stores | [files and backups reference](../reference/files-and-backups.md) |
| Command names, wrappers, state, environment | [migration](../releases/migration.md), [compatibility](../releases/compatibility.md) |

The table above names starting points, not a complete analysis. A change that
alters user-visible behavior, managed files, or support claims needs its pages
updated in the same pull request, or a tracked follow-up owned by the author.
Deterministic link, parity, and generated-section checks fail on drift
regardless of what the pull request description says.

## Page ownership

Each public page has exactly one narrative owner. This lane owns this page,
[compatibility](../releases/compatibility.md), and
[migration](../releases/migration.md), plus an entry pointer in `CONTRIBUTING.md`.
Central registration (`website/docs-manifest.json`, navigation, generated
blocks, and build configuration) belongs to the integrator; contributors
propose a page ID and path in the pull request but do not edit the registry.
The sitemap acceptance record is the scope owner: do not publish pages outside
the accepted launch and complete sets.

## Add a documentation page

1. Create the file under `website/src/content/docs/`, for example
   `website/src/content/docs/how-to/my-topic.md`.
2. Start with frontmatter containing exactly `title` and `description`.
3. Link only to accepted pages and write repository source paths as code
   spans, following the style rules above.
4. Name the page ID, audience, and evidence sources in the pull request so the
   integrator can register the page without guessing.

## Change a skill or profile

Skills live as canonical sources in `skills-files/agentic-sdd-*/`; each skill
is normative through its own `SKILL.md`, including frontmatter and relative
references. Profiles, presets, and adapters live under `opencode-pack/`.

- Keep `SKILL.md` frontmatter and relative references valid.
- Use `task preview` to see how the change would affect installed skills.
  `task apply` writes to real user skill directories, so run it only when you
  intend to update your own installation.
- Adding or editing skill files never installs them anywhere by itself;
  installation happens only through an explicit preview-and-confirm flow or
  `apply`. Do not describe a new file as shipped or enabled.
- Do not assert a new skill, profile, or preset count in prose. Catalog totals
  are generated from source; prose that hard-codes a total rots.
- The installer selects clients through its own entry points: the TUI selects
  clients, and there is no CLI `--clients` flag. Pack profiles install under
  `~/.config/opencode/agents`; fleet adapters are not installed by the
  ordinary pack flow. Document the actual entry point used, not an assumed one.

## Source validation before a pull request

Run the checks that cover what changed, from the repository root:

```sh
task test
task vet
git diff --check
```

Optional text and fixture checks use only the standard library:

```sh
python3 -B -m unittest discover -s tests/harness -p test_core_workflow.py
```

These guard instruction wording, scenario coverage, and onboarding links; they
do not call a model or prove model behavior.

Docs validation runs from the repository root with Node.js `24.22.3`
(`website/package.json`):

```sh
npm --prefix website run docs:check                 # links, fragments, metadata, freshness
ASDX_CORE_BINARY=/abs/path/asdx-core npm --prefix website run docs:reference:check
npm --prefix website test
```

`docs:reference:check` needs a binary built from the revision you are checking; the
build recipe is in `website/scripts/reference/README.md`. Do not refresh a freshness
fingerprint to make `docs:check` pass. A page whose source changed needs its content
reviewed first, and the reviewed page then gets a fresh fingerprint. A check that cannot
run because the toolchain or binary is missing is `not_run`, never a pass.

## ADR, issue, and release contributions

- A routine fix needs only a focused pull request: explain the behavior
  changed, how it was checked, and any effect on installed skills or backups.
  Update `README.md` and the Taskfile when commands or paths change. Never
  commit `backups/`, local overrides, or personal skill content.
- A material change to naming, scope, or an installation contract needs an
  Architecture Decision Record proposed alongside the change. Routine edits do
  not need one.
- File non-sensitive bug reports as repository issues with reproduction steps
  and sanitized expected versus actual results. Never post credentials,
  private backups, raw session traces, or exploit details. Sensitive reports
  use the repository's private vulnerability reporting option, or a public
  issue requesting a private channel; there is no dedicated security email, so
  do not assume a private address exists or send secrets publicly.
- Release notes and support claims route through the
  [compatibility](../releases/compatibility.md) and
  [migration](../releases/migration.md) pages, which own the exact tag,
  platform, wrapper, and state facts. Point release-related pull requests at
  those pages instead of restating version facts elsewhere.

## Migration stubs and anchors

The old guides in `docs/` that moved to these pages are kept as migration stubs. Each
keeps its old heading anchors, names its canonical page, and makes no normative claim.
Add a redirect or heading mapping there, not as a duplicate procedure here.

## Publication and observation

The docs workflow (`.github/workflows/docs.yml`) has two entrypoints: its own triggers
(validation, the protected deploy and rollback jobs, and a scheduled check) and a
read-only `workflow_call` route for post-publish observation. The observation records
`authoritative: false` and `blocking_release: false`, and a failed or delayed observation
is recorded as such. No publishing workflow calls the observation route yet, so wiring it
into the release publisher is a separate change that needs its own owner authorization.

## Docs deployment preflight

Owner-only preflight before enabling docs deployment: confirm the Pages source/branch, the Actions workflow wiring, the environment protection rules, and artifact/log retention. The deploy job stays inert until the `DOCS_PAGES_DEPLOY_ENABLED` repository variable is `true`; rollback stays inert until `DOCS_PAGES_ROLLBACK_ENABLED` is set and runs only as a protected manual dispatch. Any unavailable GitHub setting stays `unknown` — never assumed — and blocks activation until the owner confirms it.
