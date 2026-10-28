# OpenCode adapter preflight

Status: B1 inventory/contracts only. No active project configuration, profile, command, plugin, model mapping or installer change is delivered yet. The [supported-version record](supported-version.md) distinguishes an observed release from tested support.

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
| Explicit provider routes | `openrouter/openai/gpt-5.6-sol` and `openrouter/openai/gpt-6.1-terra-pro` completed bounded fixture requests; the same prefix for `openai/gpt-6-astra` exceeded the key's affordable-token limit. IDs without the explicit provider prefix returned `provider.no-route` for Terra/Astra and a ChatGPT sharing limit for `openai/gpt-5.6-sol`. | Provider prefixes and authorization state materially affect routing. Use the explicit provider/model form and verify one tiny smoke request; never infer access from the catalog. |
| Skills | Ten installed SDD bodies plus two references under `~/.agents/skills` exactly match canonical bytes | API skill response is truncated/unparseable; complete effective duplicate precedence unresolved. Loaded router's supplied path confirms that invocation's source only. |
| Instructions | Repository `AGENTS.md` and global OpenCode `AGENTS.md` supplied in session context | Their hashes are in the baseline; complete native instruction-discovery verification remains pending. |
| Startup/log observation | Last 3,000 lines: 0 warning/error matches for policy/invalid/ignored; 4 for config | Keyword counts only, not exhaustive startup validation or proof of valid policy. Raw lines were not exported. |
| Isolation | Docker client/server 29.9.1 available; required `golang:1.25-alpine` image absent | Container fixtures blocked; image not pulled. Trusted baseline checks used a disposable host checkout, not a claimed sandbox. |

## Pending before support/adoption

1. Resolve the truncated skill catalog through a version-supported bounded/export route; verify duplicate winners using the synthetic collision fixture.
2. Exercise config collisions, invalid policies and precedence in a clean isolated environment; do not attach synthetic config to the live service or real home.
3. Verify model selection/tool compatibility, allowed providers/data destinations and actual price/allowance under separately authorized access. Do not map Economy/Standard/Deep from catalog names alone.
4. Pin runtime and integration/package revisions after tests. Keep config examples inert under this directory until T05's opt-in adoption procedure exists and is authorized.
5. Establish a reproducible baseline for the frozen behavioral scenarios before canonical skill tuning. No paid comparison or new agent session is authorized by B1 alone.

Future project configuration uses `.opencode/opencode.jsonc`; manual staged diff/backup/digest-checked rollback is the approved delivery approach. Skill backup/restore remains separate. All future review routes begin disabled except proven native observation; commands may not contain shell blocks.

Documentation consulted for discovery: official V2 [CLI commands](https://opencode.ai/v2/docs/cli/commands), [API](https://opencode.ai/v2/docs/api), and its OpenAPI location parameter, plus Context7 V2 CLI guidance. Product docs are moving references; observed 2.2.0 behavior controls this record.
