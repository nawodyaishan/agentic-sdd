# Routing and model policy (opt-in OpenCode pack)

This pack pins a model per agent profile and documents the permitted manual alternatives. **A written ladder is policy, not automatic failover.** The parent agent decides whether to retry, checks compatibility first, and reports the model that actually ran.

## Provider lanes

| Lane | Prefix | Notes |
|---|---|---|
| Free OpenCode Console | `opencode/` | No Go credential required. Limited daily requests, 200,000 context / 32,000 output on the four free Flash models; `beam-free` is larger. Availability changes. |
| OpenCode Go | `opencode-go/` | Requires Go access. Draws on the plan allowance or paid Extra Usage. |
| 9Router | `9router/` | Local OpenAI-compatible endpoint (`127.0.0.1:20128/v1`) plus upstream account/quota. Claude and Gemini/Argon models live here. |
| OpenAI (optional) | `openai/` | Connected OpenAI provider in OpenCode (`openai/gpt-6.1-sol`, `openai/gpt-6.1-terra`). Optional request-only specialists; stop (`blocked`) on missing access or unauthorized spending rather than silent fallback. |

Lanes are not interchangeable: a Go model is not a substitute for a free Console model and a Console model is not a substitute for a 9Router model. The only documented exceptions are the Claude/9Router fallback matrix and the two Beam rows below.

## Pinned models in this pack

| Profile | Mode | Pinned model | Documented alternatives |
|---|---|---|---|
| `sdd-orchestrator` | primary | none — uses the active session model | human-selected session model only |
| `sdd-debug-diagnose` | subagent | `opencode/deepseek-v4.1-flash-free` | `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high` |
| `sdd-test-design` | subagent | `opencode/deepseek-v4.1-flash-free` | `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high` |
| `sdd-research-librarian` | subagent | `opencode/deepseek-v4.1-flash-free` | `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high` |
| `sdd-docs-worker` | subagent | `opencode/deepseek-v4.1-flash-free` | `opencode-go/deepseek-v4.1-flash`, `9router/ag/gemini-3.8-flash-high` |
| `sdd-architecture-advisor` | subagent | `9router/ag/claude-sonnet-5-5-high` | `9router/ag/gemini-4-argon-high`, `opencode-go/deepseek-v4.1-flash#max` |
| `sdd-visual-inspector` | subagent | `opencode/glm-5.3-flash-free` | `opencode-go/glm-5.3-flash` |
| `sdd-openai-architect` | subagent | `openai/gpt-6.1-sol` | `openai/gpt-6.1-sol-fast`, `openrouter/openai/gpt-6.1-sol` (stop if unauthorized/unavailable; no silent fallback) |
| `sdd-openai-implement` | subagent | `openai/gpt-6.1-terra` | `openai/gpt-6.1-terra-fast`, `openrouter/openai/gpt-6.1-terra` (stop if unauthorized/unavailable; no silent fallback) |
| `sdd-architecture-review-9r` | subagent | `9router/ag/claude-opus-5-5-high` | Claude-agent exception below |
| `sdd-code-review-9r` | subagent | `9router/ag/claude-opus-5-5-high` | Claude-agent exception below |
| `sdd-review-verification-1` | subagent | `opencode-go/glm-5.3-flash` | `opencode-go/deepseek-v4.1-flash`, `opencode-go/minimax-m3.1-flash` |
| `sdd-review-verification-2` | subagent | `opencode/beam-free` | see the Beam cross-lane rules |
| `sdd-lightweight` | subagent | `opencode/beam-free` (`steps: 10`) | see the Beam cross-lane rules |

`opencode-pack/presets/free-first/` ships four writer/test profiles for `--opencode-free-first`: `sdd-implement-1`, `sdd-implement-2`, `sdd-test-implement-1` and `sdd-test-implement-2`, all pinned to `opencode/deepseek-v4.1-flash-free`.

`--with-opencode` leaves the 9Router spec/plan/writer agents (`sdd-spec-9r`, `sdd-plan-9r`, `sdd-implement-9r-*`, `sdd-implement-9r-cheap-*`), the Console free writers (`sdd-implement-free-*`), `vision`, `opencode.json` and `AGENTS.md` untouched.

## Free-first ladder

When the free-first preset is installed and free DeepSeek is unavailable, the parent may try each of these **once**, in order:

1. `opencode/deepseek-v4.1-flash-free` (200k context / 32k output)
2. `opencode/glm-5.3-flash-free`
3. `opencode/mimo-v2.6-flash-free`

If the free ladder is exhausted, the work is `blocked` unless the human separately authorizes the scoped Go ladder:

4. `opencode-go/deepseek-v4.1-flash`
5. `opencode-go/glm-5.3-flash`
6. `opencode-go/minimax-m3.1-flash`
7. `opencode-go/mimo-v2.6-flash`

The Go ladder can incur charges and must be reported as used. Never move from a free model to a paid model solely because the free one is slow, and never move a free-lane task to 9Router.

## Preserved Claude/9Router fallback exception

The five Claude agents keep their existing exception: `9router/ag/claude-opus-5-5-high` or `claude-sonnet-5-5-high` primary, then `9router/ag/gemini-3.8-flash-high`, then native `opencode-go/deepseek-v4.1-flash`. This applies to `sdd-spec-9r`, `sdd-plan-9r`, `sdd-architecture-review-9r`, `sdd-code-review-9r` and `sdd-implement-9r-2`.

Compatibility checks before switching: Claude is 1,000,000 context / 128,000 output; Gemini and Argon are 1,048,576 / 65,536; Go DeepSeek is 1,000,000 / 384,000. A specification, plan or review produced by a fallback stays provisional under the global policy; a fallback architecture or code review needs an Opus review before it is the final consequential technical verdict.

## The two Beam cross-lane rules

`sdd-lightweight` and `sdd-review-verification-2` use the free Console Beam model as their primary but have Go fallbacks recorded in the global fallback matrix. Two rules apply:

1. **No automatic cross-lane switch.** A Beam failure does not authorize a Go model. The parent must obtain explicit human authorization for that specific retry and must state that it may incur charges.
2. **Report and keep it supplementary.** Once authorized, use the documented Go fallback, report the model that actually ran, and keep `sdd-review-verification-2` a supplementary assessment rather than the sole consequential reviewer. `sdd-lightweight` remains a bounded read-only discovery role even on the fallback model.

## MCP access rules

- Every deny-by-default subagent allows `execute` together with `codegraph_*` and `context7_*`. That covers both Code Mode (servers exposed through `execute`) and direct MCP exposure when a server sets `codemode: false`. Tools called inside `execute` still enforce their own `<server>_<tool>` permission.
- Only the high-intensity roles — `sdd-architecture-review-9r`, `sdd-code-review-9r`, `sdd-architecture-advisor`, `sdd-research-librarian`, `sdd-openai-architect` — allow `exa_web_search_exa` and `exa_web_fetch_exa`. `sdd-research-librarian` also allows `webfetch` and `websearch`. Every other profile denies `exa_*`.
- External query rule: never put private repository code, file contents, internal identifiers, credentials, secrets or infrastructure details into an external search or fetch query.
- Review roles may read outside the workspace and run only `git diff` and `git status --short`. All other subagents deny shell entirely, and every subagent denies `subagent` and `edit` (writers allow `edit` but still deny `subagent`).
- `.env` files, and `*secret*` / `*credential*` paths on `sdd-lightweight`, are denied last so the last matching rule wins.

## CLI usage

```text
agentic-sdd [preview|apply]                              # 18 core skills only
agentic-sdd [preview|apply] --with-opencode              # + agentic-sdd-orchestrate, + 12 agent profiles
agentic-sdd [preview|apply] --with-opencode --opencode-free-first
                                                         # + the 4 Free-first writer/test profiles
agentic-sdd backups                                     # list backups (skills and opencode-agents)
agentic-sdd restore [ID] [--apply]                      # restore a backup without extra flags
```

`preview` is the default and writes nothing. `--opencode-free-first` without `--with-opencode` exits `2`. `apply` backs up every replaced profile first, keeps unrelated files untouched, and is a no-op when the files already match. Restore accepts backups written with or without `--with-opencode`.

Running `apply` against the real `~/.config/opencode` changes the live installation and needs its own explicit authorization.
