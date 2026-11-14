---
title: Compatibility and support
description: Which checkouts, releases, platforms, and clients are supported, and where to verify them.
---

Support here means a combination this project actually provides or verifies,
not a combination that merely seems plausible. Cite exact tags and evidence
levels; never assume a `latest` tag resolves to anything in particular.
For the canonical-to-legacy command mapping, see [migration](migration.md).

## Checkout versus release

A checkout is the source tree at whatever commit was cloned or reviewed. A
release is an exact published tag with its generated artifacts. Behavior
documented from source applies to the checkout it was read from; behavior tied
to an artifact applies only to the tag that produced it.

- Tag `v0.11.0` exists and its tag and cask metadata are verified.
- Archive contents, runtime identity, and a live macOS install from that tag
  are not verified. Do not describe them as tested.
- The website itself is not live, so public pages link repository source
  paths rather than a published site.

## Release history and changelog

Release history lives on the
[GitHub releases page](https://github.com/nawodyaishan/agentic-sdd/releases).
Each entry names its exact tag; treat the tag as the identifier, not any
floating label. Generated changelogs exclude documentation, test, CI, and
chore entries by configuration in `.goreleaser.yml`, so routine maintenance
commits do not appear as user-facing changes.

## Platform support

The table below summarizes what each platform gets. The surrounding text is
the definition; the table is a summary of it.

| Platform | Source checkout | Published artifact | Status |
|---|---|---|---|
| macOS (Apple silicon and Intel) | Builds from source | Homebrew cask with a paired four-name archive (universal binaries) | Tag and cask metadata verified for `v0.11.0`; live install not verified |
| Linux | Builds from source; the main entry point is the TUI built from source | No prebuilt package | Source-only |
| Windows | None | None | Unsupported and unknown |

Linux and source checkouts never require a package manager artifact that does
not exist; Windows behavior is unknown, so do not document Windows steps.

## Client support

Skills install into four client destinations:
`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`, and
`~/.antigravity/skills`. Pack profiles live separately in
`~/.config/opencode/agents`. The sole client and destination support matrix
belongs to `introduction/clients`; this page does not duplicate it. In
particular, do not assume installing to one client implies identical behavior
in another, and do not present the optional orchestration pack as part of the
core skill set. The core set is generated from `skills-files/`; the pack adds
its orchestrator skill and agent profiles on top.

## Maintenance policy

No long-term support window or maintenance schedule is announced here. The
supported surface is the current checkout plus the exact release tags named on
the releases page. Specifically:

- The supported Go verification scope is the 11 supported-core packages; the
  temporary exclusion fence for blocked historical lanes stays in force and
  their records remain evidence, not work queues.
- Legacy command wrappers remain for at least one deprecation window; no
  wrapper-removal release has been announced. See [migration](migration.md)
  for the exact names and behavior.
- Still-correct pages are not stale merely because of age. A page becomes
  stale when a source, artifact, or support fact it asserts changes, which
  triggers one relevant review of that page.

## Documentation availability

The documentation site is not live. Docs ship from this repository; no deployment claim is made about a published URL.
