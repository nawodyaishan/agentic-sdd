---
title: Agent profile catalog
description: Optional OpenCode profiles, presets and adapters.
---

Source-derived profile facts. A pinned model is not proof of provider availability or automatic failover.

<!-- generated:reference/profiles:start -->
## Pack agent profiles

14 agent profiles are defined in `opencode-pack/agents/` and selected by `--with-opencode`. `sdd-orchestrator` is the human-facing `primary` profile and pins no model: it inherits the session model. A pinned model is configuration only; this page makes no claim about provider, credential, quota or model availability.

Permission columns summarize the ordered `permissions` rules using last-match-wins selection per tool; resource-scoped allow rules are reported as `scoped` and not enumerated here. `sdd-lightweight` declares `steps: 10`.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-architecture-advisor` | `subagent` | `9router/ag/claude-sonnet-5-5-high` | base `deny-by-default`, 15 rules | `deny` | `deny` | `deny` | `allow` |
| `sdd-architecture-review-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-code-review-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-debug-diagnose` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-docs-worker` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `deny` | `deny` | `deny` |
| `sdd-lightweight` | `subagent` | `opencode/beam-free` | base `deny-by-default`, 16 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-openai-architect` | `subagent` | `openai/gpt-6.1-sol` | base `deny-by-default`, 17 rules | `deny` | `scoped` | `deny` | `allow` |
| `sdd-openai-implement` | `subagent` | `openai/gpt-6.1-terra` | base `explicit-rules`, 8 rules | `allow` | `not-declared` | `deny` | `deny` |
| `sdd-orchestrator` | `primary` | session model (inherit-session) | base `explicit-rules`, 1 rule | `not-declared` | `not-declared` | `allow` | `not-declared` |
| `sdd-research-librarian` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 17 rules | `deny` | `deny` | `deny` | `allow` |
| `sdd-review-verification-1` | `subagent` | `opencode-go/glm-5.3-flash` | base `deny-by-default`, 16 rules | `deny` | `scoped` | `deny` | `deny` |
| `sdd-review-verification-2` | `subagent` | `opencode/beam-free` | base `deny-by-default`, 16 rules | `deny` | `scoped` | `deny` | `deny` |
| `sdd-test-design` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-visual-inspector` | `subagent` | `opencode/glm-5.3-flash-free` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |

## Free-first preset templates

4 templates are defined in `opencode-pack/presets/free-first/`. They are selected only when both `--with-opencode` and `--opencode-free-first` are passed; `--with-opencode` alone does not install them.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-implement-1` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-implement-2` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-test-implement-1` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |
| `sdd-test-implement-2` | `subagent` | `opencode/deepseek-v4.1-flash-free` | base `deny-by-default`, 14 rules | `allow` | `allow` | `deny` | `deny` |

## Fleet adapter profiles

2 profiles are defined in `opencode-pack/adapters/`. They are resolver-only render sources for reviewed fleet model routes and are not installed by the ordinary OpenCode pack: `--with-opencode` reads `opencode-pack/agents/` only.

| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `sdd-plan-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |
| `sdd-spec-9r` | `subagent` | `9router/ag/claude-opus-5-5-high` | base `deny-by-default`, 14 rules | `deny` | `deny` | `deny` | `deny` |

## Derived counts

| Class | Selected by | Count |
| --- | --- | --- |
| Pack agent profiles | `--with-opencode` | 14 |
| Free-first preset templates | `--with-opencode --opencode-free-first` | 4 |
| Fleet adapter profiles | not installed by the pack | 2 |

## Source evidence

Facts above are parsed from 20 profile files with the pinned `yaml` parser; fingerprints are recorded in the documentation manifest. This block describes repository source, not runtime permission enforcement, model routing success or the installation state of any machine.
<!-- generated:reference/profiles:end -->
