# OpenCode subagent fleet: configuration and routing report

**Snapshot:** 2026-11-07 · **Scope:** global configuration, as resolved from `/home/nawo-ubuntu1` · **OpenCode:** v2.2.5

This report consolidates and corrects `opencode-subagent-fleet-report.md` and `sdd-subagent-model-routing-report.md`. The original reports remain unchanged. Project-level configuration, current account subscription, actual usage, runtime availability, and end-to-end fallback behavior were not audited. Configured models and catalogue entries are not proof of working credentials, remaining quota, or successful inference.

## Executive summary

- `opencode debug agents` lists **21 subagents**: 18 Markdown-defined agents, one JSON-defined `explorer`, and two built-ins, `explore` and `general`. **19 have explicit models**; the two built-ins have no model pinned and normally inherit the parent session's model. The 18 Markdown agents comprise 17 SDD agents and `vision`.
- Global default model: `9router/ag/claude-opus-5-5-medium`. A session can have a different selected model; this is not proof of what a particular session is using.
- The five Claude-based SDD agents have a **written exception**: Claude primary → `9router/ag/gemini-3.8-flash-high` → native `opencode-go/deepseek-v4.1-flash`. Their primary models remain Claude.
- **Fallback is policy, not verified automatic failover.** The parent must manage retries and any supported model override. The current client may restrict overrides; an ongoing long retry is not proven to switch automatically. No live inference or failover smoke test was performed for this report.
- Two other rows in the written fallback matrix conflict with its lane rule: `sdd-lightweight` and `sdd-review-verification-2` use Console Beam as primary but list Go fallbacks. This report records the conflict; it does not decide which instruction prevails or change the configuration.
- Both prior reports contain stale or unsupported claims; see the corrections section. In particular, they omit `general` from the inventory, show the old Claude fallback chain, and overstate permission boundaries.

## 1. Fleet inventory

Source: `opencode debug agents` and global agent files. `#high` and `#max` are model variants. A dash means no explicit step limit was observed, not unlimited execution. See §3 for fallback chains.

| Agent | Role | Configured primary model | Steps |
|---|---|---|---:|
| `explore` | Built-in read-oriented exploration | Inherited | — |
| `general` | Built-in general-purpose work | Inherited | — |
| `explorer` | Custom fast exploration | `9router/ag/gemini-3.8-flash-high` | — |
| `sdd-architecture-review-9r` | Architecture / operational risk review | `9router/ag/claude-opus-5-5-high` | — |
| `sdd-code-review-9r` | Independent code / evidence review | `9router/ag/claude-opus-5-5-high` | — |
| `sdd-spec-9r` | Feature specification | `9router/ag/claude-opus-5-5-high` | — |
| `sdd-plan-9r` | Feature plan and tasks | `9router/ag/claude-opus-5-5-high` | — |
| `sdd-implement-9r-2` | Complex implementation | `9router/ag/claude-sonnet-5-5-high` | — |
| `sdd-implement-9r-1` | Routine implementation | `9router/ag/gemini-3.8-flash-high` | — |
| `sdd-implement-9r-cheap-1` | Low-cost routine implementation | `9router/ag/gemini-3.8-flash-medium` | — |
| `sdd-implement-9r-cheap-2` | Low-cost complex implementation | `9router/ag/gemini-4-argon-high` | — |
| `sdd-implement-1` | Routine implementation | `opencode-go/deepseek-v4.1-flash#high` | — |
| `sdd-implement-2` | Complex implementation | `opencode-go/deepseek-v4.1-flash#max` | — |
| `sdd-implement-free-1` | Free routine implementation | `opencode/deepseek-v4.1-flash-free` | — |
| `sdd-implement-free-2` | Free larger / ambiguous implementation | `opencode/deepseek-v4.1-flash-free` | — |
| `sdd-lightweight` | Bounded source-backed discovery | `opencode/beam-free` | 10 |
| `sdd-review-verification-1` | Verification review | `opencode-go/glm-5.3-flash` | — |
| `sdd-review-verification-2` | Supplementary verification review | `opencode/beam-free` | — |
| `sdd-test-implement-1` | Focused tests | `opencode-go/qwen3.8-flash` | — |
| `sdd-test-implement-2` | Focused tests | `opencode-go/mimo-v2.6-flash` | — |
| `vision` | Image transcription / description | `opencode-go/glm-5.3-flash` | 4 |

`build` and `plan` are primary agents, not included in the 21. Maintenance agents such as `compaction` are also outside this count. The global `explorer` entry is custom, not the built-in `explore`.

## 2. Model characteristics relevant to the fleet

Limits and modalities below are **OpenCode catalogue / local configuration metadata**, not independently measured service guarantees. Context and output are token limits; output can be further constrained by the client or provider. Data came from `opencode api get /api/model` and the configured 9Router model definitions.

| Model or family | Context | Max output | Listed input |
|---|---:|---:|---|
| `9router/ag/claude-opus-5-5-{high,medium,low}` | 1,000,000 | 128,000 | text, image, PDF |
| `9router/ag/claude-sonnet-5-5-{high,medium,low}` | 1,000,000 | 128,000 | text, image, PDF |
| `9router/ag/gemini-3.8-flash-{high,medium,low}` | 1,048,576 | 65,536 | text, image, audio, video |
| `9router/ag/gemini-4-argon-{high,medium,low}` | 1,048,576 | 65,536 | text, image, audio, video |
| `opencode-go/deepseek-v4.1-flash` | 1,000,000 | 384,000 | text, image |
| `opencode-go/glm-5.3-flash` | 1,000,000 | 131,072 | text, image, video |
| `opencode-go/qwen3.8-flash` | 1,000,000 | 131,072 | text, image, video |
| `opencode-go/mimo-v2.6-flash` | 1,048,576 | 131,072 | text, image, audio, video |
| `opencode-go/minimax-m3.1-flash` | 1,000,000 | 512,000 | text, image, video |
| `opencode/deepseek-v4.1-flash-free` | 200,000 | 32,000 | text, image |
| `opencode/glm-5.3-flash-free` | 200,000 | 32,000 | text, image, video |
| `opencode/mimo-v2.6-flash-free` | 200,000 | 32,000 | text, image, audio, video |
| `opencode/beam-free` | 1,048,576 | 131,072 | text |

The Claude → Gemini fallback reduces the listed output ceiling and changes supported inputs. DeepSeek's listed inputs also differ from Claude's. Check the actual conversation, attachments, tool history and output needs before switching. `vision`'s prompt explicitly says its GLM model accepts images but not PDF attachments; convert PDF pages to images before delegation.

Global configuration explicitly declares 18 9Router models and eight OpenRouter models, not all assigned to agents. Additional 9Router entries include unsuffixed Gemini 3.8, Sonnet and Opus, GPT-OSS and two Gemini Flash Lite entries. Provider catalogue listing alone does not establish inference availability or price. OpenRouter's configured entries are not primary models in the above fleet.

## 3. Fallbacks and safe recovery

The authoritative written matrix is `~/.config/opencode/AGENTS.md`, lines 52–84 at this snapshot. The table below reproduces its entries, **including the two conflicts identified below**. These are instructions to the parent, not model arrays attached to agent definitions.

| Agent(s) | Fallback 1 | Fallback 2 |
|---|---|---|
| `sdd-implement-9r-2`, `sdd-spec-9r`, `sdd-plan-9r`, `sdd-architecture-review-9r`, `sdd-code-review-9r` | `9router/ag/gemini-3.8-flash-high` | `opencode-go/deepseek-v4.1-flash` |
| `sdd-implement-1`, `sdd-implement-2` | `opencode-go/mimo-v2.6-flash` | `opencode-go/minimax-m3.1-flash` |
| `sdd-implement-9r-1` | `9router/ag/gemini-3.8-flash-medium` | `9router/ag/gemini-4-argon-medium` |
| `sdd-implement-9r-cheap-1` | `9router/ag/gemini-4-argon-medium` | `9router/ag/gemini-3.8-flash-high` |
| `sdd-implement-9r-cheap-2` | `9router/ag/gemini-3.8-flash-high` | `9router/ag/gemini-4-argon-medium` |
| `sdd-implement-free-1` | `opencode/glm-5.3-flash-free` | `opencode/mimo-v2.6-flash-free` |
| `sdd-implement-free-2` | `opencode/beam-free` | `opencode/glm-5.3-flash-free` |
| `sdd-test-implement-1` | `opencode-go/deepseek-v4.1-flash` | `opencode-go/mimo-v2.6-flash` |
| `sdd-test-implement-2` | `opencode-go/qwen3.8-flash` | `opencode-go/deepseek-v4.1-flash` |
| `sdd-review-verification-1` | `opencode-go/deepseek-v4.1-flash` | `opencode-go/minimax-m3.1-flash` |
| `sdd-lightweight` | `opencode-go/mimo-v2.6-flash` | `opencode-go/qwen3.8-flash` |
| `sdd-review-verification-2` | `opencode-go/glm-5.3-flash` | `opencode-go/deepseek-v4.1-flash` |

No fallback matrix entry was found for `vision`, `explorer`, `explore`, or `general`.

**Conflict:** `AGENTS.md` line 56 says Console agents use only `opencode/` IDs, while lines 66 and 82 assign `opencode-go/` fallbacks to two Console-primary agents. The policy expressly authorizes cross-lane fallback for the five Claude agents, but does not similarly resolve those two rows. A parent should surface the ambiguity rather than assume the prior report's “zero violations” claim.

The policy instructs the parent to retry only on unavailability, quota or repeated transient failures; try each model at most once, and stop on permission errors or invalid requests. Before resuming interrupted writes, inspect child-session results and working-tree changes, preserve completed work and avoid duplicate side effects. Continue the same child session only where the client supports safe continuation and the requested model change. Check input and output compatibility. Report actual model and reason. Claude-fallback architecture/code reviews are provisional and require Opus review for a final consequential technical verdict; drafting still requires the ordinary combined human approval. This is a **safety procedure**, not a guarantee of uninterrupted execution.

## 4. Authorization and permissions

- The global instructions permit bounded automatic delegation to `sdd-lightweight`; other SDD agents require explicit delegation within the applicable session's higher-priority rules.
- Implementation, test, spec and plan agents deny further subagent calls in their frontmatter. Their assigned-scope and no-destructive-operation rules are primarily **instructions**, not equivalent to path-level filesystem sandboxes. In particular, test-only and `specs/`-only edits are prompt constraints; the spec/plan agent files do not themselves deny all shell use.
- The four review agents (`sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-review-verification-1`, `sdd-review-verification-2`) use deny-by-default rules with read, glob, grep, skill and selected exceptions; their shell allow rules name `git diff` and `git status --short`. Their review verdict does not replace human approval.
- `sdd-lightweight` is deny-by-default with selected read/search/CodeGraph permissions and explicit exclusions for environment, secret and credential patterns. `vision` explicitly denies editing, shell and network search/fetch, but its file permits reads broadly; it is **not** accurate to claim it blocks every sensitive file.
- The custom `explorer` has a description and model in JSON; do not infer a read-only sandbox from its name. For all agents, effective permissions depend on ordered global and agent-specific rules; consult `opencode debug agents` for the active location. OpenCode V2 uses last matching permission rule, according to its [agents documentation](https://opencode.ai/v2/docs/agents/).

## 5. Providers, cost and privacy

| Provider | Local use | Operational considerations |
|---|---|---|
| `9router/` | Local OpenAI-compatible endpoint `127.0.0.1:20128/v1` | Local service and upstream account/quota required. Actual upstream pricing and current balances were not verified; do not assume all routed use is free. |
| `opencode-go/` | Paid Go models, including native DeepSeek fallback | Requires appropriate access and credentials. May draw on allowance or paid Extra Usage. Actual plan and Extra Usage setting were not checked. |
| `opencode/` | Free Console models for four agents | Free model availability, rate limits, privacy terms and geographic eligibility vary. Do not assume free implies private or credential-independent for every setup. |

The [official Go pricing documentation](https://opencode.ai/v2/docs/console/go/) lists Go at $10/month, Go Plus at $40/month and Teams at $20/member/month. The following are **published Go plan** figures at report time, not a statement of the user's subscription or actual bill. Rates are USD per million tokens.

| Model | Input | Output | Cached read | Go monthly usage limit |
|---|---:|---:|---:|---:|
| DeepSeek V4.1 Flash, off-peak | $0.15 | $0.60 | $0.003 | $100 |
| DeepSeek V4.1 Flash, peak | $0.30 | $1.20 | $0.006 | $100 |
| GLM-5.3-Flash | $0.15 | $0.50 | $0.03 | $60 |
| Qwen3.8 Flash | $0.15 | $0.47 | $0.016 | $30 |
| MiMo V2.6 Flash | $0.14 | $0.28 | $0.0028 | $60 |
| MiniMax M3.1 Flash | $0.15 | $0.60 | $0.015 | $20 |

The documentation describes 5-hour and weekly limits of 20% and 50% of the monthly limit, respectively, and optional paid Extra Usage. DeepSeek peak is 01:00–04:00 and 06:00–10:00 UTC on weekdays. Price, offers, model availability, retention and geographic terms can change; check the provider's current terms and workspace privacy settings before sending sensitive material. Do not treat old retention claims as independently verified by this audit.

## 6. Other global configuration

`~/.config/opencode/opencode.json` contains the default model, provider definitions and the inline `explorer` entry. Five named MCP server definitions are present: `codegraph`, `exa`, `context7`, `claude-mem`, and `chrome-devtools`; the last is explicitly disabled in the inspected configuration. Configuration presence is not a connection-health check. `context7` and `claude-mem` appear in a different nesting shape from the three entries under `mcp.servers`, so verify effective availability using the running client rather than inferring it from JSON alone. Plugin files and `/look` also exist in the global configuration tree; their runtime status was not audited.

## 7. Corrections to the two source reports

1. **Inventory:** 21 resolved subagents, not 20; the old fleet report omitted `general` and miscounted Markdown files.
2. **Fallbacks:** five Claude-based agents now have Gemini then native Go DeepSeek as a written exception, not Opus/Sonnet medium then low. The two additional cross-lane matrix conflicts above make an unqualified “zero violations” incorrect.
3. **Runtime:** written fallbacks are not evidence of automatic failover, safe recovery, available quota, or successful model overrides. Historic smoke tests do not demonstrate current behavior.
4. **Permissions:** not all agents deny environment-file reads; `vision` does not have a general secret-file deny; `explorer` and spec/plan write/shell restrictions must not be represented as guaranteed sandboxes without examining effective rules.
5. **Models:** the earlier 9Router table lists 15 of the 18 configured IDs. The Go documentation now lists several free models, so the older claim that Go contains only two is obsolete. A model's appearance under one provider does not prove another provider lacks it.
6. **MCP count:** five named definitions were inspected, not six; configured, connected and enabled are different states.
7. **Pricing and privacy:** old estimates, account-specific claims, 9Router cost assertions and retention guarantees were not independently established. Published pricing is time-sensitive; no personal balance or plan was inspected.
8. **Old audit script:** it classifies all non-Console primaries as Go, so it incorrectly marks 9Router agents and does not validate the current exception or all fallback rows. Do not rely on its “zero violations” result.

## 8. Evidence and limits

- Global sources: `~/.config/opencode/AGENTS.md`, `~/.config/opencode/agents/*.md`, and non-secret fields of `~/.config/opencode/opencode.json`.
- Runtime metadata: `opencode debug agents`; `opencode api get /api/model` was filtered locally to model IDs, limits and modalities. No credential values or raw provider headers are reproduced here.
- Documentation: [OpenCode V2 agents](https://opencode.ai/v2/docs/agents/), [Go pricing and privacy](https://opencode.ai/v2/docs/console/go/). Model catalogue and public documentation can change.
- Scope excludes project-specific agent overrides, inference calls, quota or credential tests, and verification that an in-flight session can change models without replaying actions. Check the relevant project and client before operating on consequential work.
