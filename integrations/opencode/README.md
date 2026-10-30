# OpenCode adapter preflight

Status: B3 inactive templates and credential-free native fixtures delivered;
**support/adoption still Blocked**. No active project/global configuration or
shared service is changed. The [supported-version record](supported-version.md)
distinguishes observations from certified support.

## B3 artifacts

- [Native template](opencode.jsonc): draft/build/review primaries and prompt-only
  current-session commands; one selected feature/build path, providers denied.
- [Higher-authority lockdown example](policy-lockdown.jsonc): separate, inactive;
  never installed automatically by a project/skill installer.
- [Usage catalog](model-usage.json): user-selected DeepSeek/GLM free roles,
  dormant paid MiMo and synthetic-only Nemotron; all execution disabled.
- [Manual adoption/rollback](adoption.md), [incident-derived spending/credential
  safeguards](safety.md), and [verification limits](verification.md).
- [Manual recovery procedure](recovery.md): B6 offline reconciliation and
  deterministic-check evidence only; native agent recovery remains unobserved.

Both `.jsonc` templates deliberately use the JSON subset for dependency-free
static checks. They are examples, not automatically discovered configuration
at this location. Their references to free models are not price/access proofs.

## Inspected setup — 2026-10-28

The existing service was already running. Read-only CLI/API discovery made no model requests and changed no configuration. API calls explicitly used `location[directory]=<repository>` after observing that omission selected the user's home rather than the shell working directory. Do not infer location from where the CLI was launched.

Allowed projections only were printed: version, source paths, field names, agent/command/plugin IDs, MCP status and model identity. No credential endpoint, auth export, environment dump, provider headers or full config/log export was used. The API catalog is not a billing or capability test.

| Area | Observed | Meaning / remaining work |
| --- | --- | --- |
| CLI and service | Both 2.2.0 | Exact observed candidate release; not yet a supported adapter claim. |
| Config sources | `~/.config/opencode/opencode.json` plus global `~/.config/opencode` and `~/.opencode` discovery directories | Returned for this repository location; no project config document returned. |
| Config document keys | `$schema`, `mcp`, `plugins` | No authored SDD profiles, commands, model default or policies in that document. Does not prove all effective defaults or permissions. |
| Agents | build, general, explore, compaction, title, summary, plan | Built-in catalog; no `sdd-*` primary profiles. |
| Commands | init, review, exa:agent_research_help, exa:web_search_help | Existing discovery only; no command was executed. |
| MCP | codegraph/context7/exa connected; chrome-devtools disabled | Connection status does not prove every tool works. CodeGraph separately reported this repository unindexed despite a `.codegraph/` directory. |
| Plugins | 91 discovered entries; non-`opencode.*` ID `herdr.opencode` | IDs only; package revision and executable isolation unverified. Inventory is not a plugin-code audit. |
| Service default model | provider `openrouter`, model `openai/gpt-6.1-terra-pro` | Observed for repository location; no selection change made. |
| Current session model | `openai/gpt-6-astra`, supplied by this session's harness | Different from service default. Variant/access/cost not inferred. |
| Model catalog | Home-location query returned 418 entries across cerebras/google/groq/openai/opencode/openrouter | Availability listing only; do not treat home catalog as verified repository-specific access. Repository-default lookup was explicitly scoped. |
| Historical explicit provider routes | Sol/Terra OpenRouter routes made paid requests in B2; Astra route hit an affordable-token error. Other routes returned no-route or subscription sharing-limit errors. | Incident finding, not a recommended smoke procedure. These OpenAI evaluation routes are excluded from the new usage catalog; no automatic probing or fallback. See safety guidance. |
| Skills | Ten installed SDD bodies plus two references under `~/.agents/skills` exactly match canonical bytes | API skill response is truncated/unparseable; complete effective duplicate precedence unresolved. Loaded router's supplied path confirms that invocation's source only. |
| Instructions | Repository `AGENTS.md` and global OpenCode `AGENTS.md` supplied in session context | Their hashes are in the baseline; complete native instruction-discovery verification remains pending. |
| Startup/log observation | Last 3,000 lines: 0 warning/error matches for policy/invalid/ignored; 4 for config | Keyword counts only, not exhaustive startup validation or proof of valid policy. Raw lines were not exported. |
| Isolation | B1 initially lacked the Go image; the separately authorized pull and Docker e2e later passed. B3 uses an already local pinned Python image, no network, no credential/home mounts and fresh tmpfs. | B3 native API probes pass within their limited scope; no new image pulled. See verification record. |

## Pending before support/adoption

1. Resolve the truncated skill catalog through a version-supported bounded/export route; verify duplicate winners using the synthetic collision fixture.
2. Exercise config collisions, invalid policies and precedence in a clean isolated environment; do not attach synthetic config to the live service or real home.
3. Verify model selection/tool compatibility, allowed providers/data destinations and actual price/allowance under separately authorized access. Do not map Economy/Standard/Deep from catalog names alone.
4. Pin runtime and integration/package revisions after tests. Keep config examples inert under this directory until T05's opt-in adoption procedure exists and is authorized.
5. Historical baseline observations and their incident correction are in `harness/baseline.md`. They show no measured improvement and are not permission for further paid work. Follow safety prerequisites even when a future model test is authorized.

Future project configuration uses `.opencode/opencode.jsonc`; manual staged diff/backup/digest-checked rollback is the approved delivery approach. Skill backup/restore remains separate. All future review routes begin disabled except proven native observation; commands may not contain shell blocks.

Documentation consulted for discovery: official V2 [CLI commands](https://opencode.ai/v2/docs/cli/commands), [API](https://opencode.ai/v2/docs/api), and its OpenAPI location parameter, plus Context7 V2 CLI guidance. Product docs are moving references; observed 2.2.0 behavior controls this record.
