# Agentic SDD default MCP integration study

**Date:** 2026-11-12
**Scope:** CodeGraph, Context7 and Exa as the default MCP tool set for Agentic SDD; install-time availability checks; missing-piece installation; safe OpenCode configuration.
**Method:** repository inspection, current vendor documentation via Exa, and an independent read-only research subagent. External facts were checked on 2026-11-12 and can change.
**Status:** research and recommendation only. It does not authorize code changes, real-home writes, package installation or release.

## 1. Executive recommendation

Make the **default tool set known and checked by default**, but keep **writes and package installation explicit and consent-based**:

1. **Default `preview`:** report the state of CodeGraph, Context7 and Exa without writing any config, installing any package or touching the network. Show the exact missing pieces and the exact next command.
2. **`--with-mcp`:** include an MCP configuration merge in the same preview/apply transaction, with a digest, backup, atomic write and no-op behavior. This configures only servers that already exist or are remote; it does not install binaries.
3. **`mcp install-missing`:** the only path allowed to run an official installer/package manager, after printing the exact argv and obtaining explicit consent. CodeGraph is the only default that needs a local binary; Context7 and Exa are remote.
4. **Never delegate default configuration to vendor installers** (`codegraph install`, `ctx7 setup`). They also write agent instructions, rules or skills and create a second owner of the same files. Use their documented `--print-config`/manual snippets only as references.

This gives a one-command experience when the owner asks for it, while preserving Agentic SDD's existing guarantee that `preview`/`apply` do not silently modify `~/.config/opencode/opencode.json` or install software.

## 2. Current repository state

| Area | Verified current behavior |
|---|---|
| Skill install | `preview`/`apply` install the 18 portable skills into `~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills` and `~/.gemini/antigravity-cli/skills`. Preview is the default; apply backs up and stages. |
| `--with-opencode` | Adds the orchestration skill and installs 14 OpenCode agent profiles into `<home>/.config/opencode/agents/`. It does not read or write `opencode.json(c)`. |
| OpenCode config | Tests intentionally seed `~/.config/opencode/opencode.json` and assert the installer preserves it. There is no MCP server entry, availability check or installer. |
| MCP references | Profiles allow `codegraph_*`, `context7_*` and `exa_*`; docs describe them. That is permission to use a server, not proof that one is configured. |
| Config transaction precedent | The build-tagged `agentic-sdd opencode config` path has digest pinning, backup, atomic write and rollback, but it replaces a project config and is excluded from the core gate. It is a useful transaction model, not a reusable MCP merge. |
| Backup/restore | Manifest formats 1–3 cover skill trees and `opencode-agents/*.md`; there is no config-file backup/restore path. |
| Isolation | `--home` relocates destinations; tests use temporary homes. The supported-core gate and offline isolation are binding. |

**Conclusion:** installation is greenfield for MCP. The installer has no concept of a server manifest, a config merge, a binary check or a missing-piece installer.

## 3. External contracts verified

### OpenCode V2

OpenCode V2 stores servers under `mcp.servers.<name>`; this is not the older `mcp.<name>` shape. Local servers use `type: "local"` with a `command` array; remote servers use `type: "remote"` with an absolute `url`. Relevant fields include `environment`, `headers`, `oauth`, `disabled`, `codemode`, `timeout` and `protocol`. Global config is `~/.config/opencode/opencode.json(c)`; project config is `opencode.json(c)` or `.opencode/opencode.json(c)`. OpenCode also exposes `opencode mcp add/list/auth` and `opencode mcp add --global`.

Source: [OpenCode MCP servers](https://opencode.ai/docs/mcp-servers/), [OpenCode config](https://opencode.ai/docs/config/).

### CodeGraph

Local CLI 1.6.2 confirms the exact OpenCode entry with:

```sh
codegraph install --print-config opencode
```

It prints:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "codegraph": {
        "type": "local",
        "command": ["codegraph", "serve", "--mcp"],
        "disabled": false,
        "codemode": false
      }
    }
  }
}
```

`codegraph init` creates `.codegraph/` and indexes the project; `codegraph serve --mcp` is the server; the watcher auto-syncs, while `codegraph sync` and `codegraph status` are the explicit checks. `codegraph install` also writes agent config and an `AGENTS.md` section, so Agentic SDD should not call it as a config writer.

Sources: [CodeGraph installation](https://colbymchenry.github.io/codegraph/getting-started/installation/), [CodeGraph MCP server](https://colbymchenry.github.io/codegraph/reference/mcp-server/), [CodeGraph CLI](https://colbymchenry.github.io/codegraph/reference/cli/).

### Context7

Hosted remote endpoint `https://mcp.context7.com/mcp`, with optional `Authorization: Bearer <key>` or OAuth at `/mcp/oauth`. `npx ctx7 setup --opencode` can configure it, but it also writes rules and a skill. A local stdio alternative is `npx -y @upstash/context7-mcp`; the hosted remote is preferred for a default because it needs no local package and supports OAuth.

Source: [Context7 all clients](https://context7.com/docs/resources/all-clients), [Context7 repository](https://github.com/upstash/context7).

### Exa

Hosted remote endpoint `https://mcp.exa.ai/mcp`. Keyless search/fetch works at a free rate limit; OAuth (`?login`) or an API key (`x-api-key`) enables higher limits and `agent_run`. OpenCode uses the standard remote shape under `mcp`, with `type: remote`.

Source: [Exa MCP](https://exa.ai/docs/get-started/exa-mcp), [Exa authentication](https://exa.ai/auth.md).

## 4. Integration options

| Option | Writes config | Installs binaries | Safety | UX | Verdict |
|---|---|---|---|---|---|
| **A. Detect/report only** | No | No | Highest | User copies snippets/runs commands | Recommended default and first phase |
| **B. Opt-in config merge** | Yes, with digest/backup | No | Good | One `--with-mcp` transaction | Recommended phase 2 |
| **C. Config merge + explicit missing-piece install** | Yes | Yes, official commands only | Medium; needs consent/pinning | Best one-stop setup | Recommended opt-in phase 3 |
| **D. Delegate to vendor installers** | Yes, plus vendor-owned files | Yes | Low control; overlapping owners | Easiest initially | Do not use by default |

Option D is especially unsafe here: CodeGraph installs an MCP entry **and** an instructions-file section; Context7 setup writes config, rules and a skill. Agentic SDD already owns skills and agent profiles, so those writes would collide with its ownership model.

## 5. Recommended CLI surface

```text
agentic-sdd mcp status  [--home PATH] [--config PATH] [--json]
agentic-sdd mcp doctor  [--home PATH] [--config PATH] [--network]
agentic-sdd mcp preview [--servers codegraph,context7,exa] [--home PATH] [--config PATH]
agentic-sdd mcp apply   --expect-config DIGEST --backup-root DIR [--servers ...]
agentic-sdd mcp restore --backup ID [--home PATH]
agentic-sdd mcp uninstall [--servers ...] [--home PATH]
agentic-sdd mcp install-missing --servers codegraph [--yes]
```

- `mcp status` is the read-only default if `mcp` is invoked without a subcommand.
- `mcp preview` prints the semantic diff and a digest; `mcp apply` requires that digest, exactly like the existing config transaction.
- `mcp install-missing` prints the exact argv and requires `--yes` or interactive confirmation. It never runs during `preview`.
- `--with-mcp` is optional sugar: with `preview` it includes the MCP plan; with `apply` it performs **config merge only**. `--with-opencode` implies no MCP write; it may print one hint on a TTY. Existing non-TTY output stays byte-identical.
- Do not add network probing to the default path. `mcp doctor --network` is explicit and tests reachability only; it is not a capability or authentication proof.

### 5.1 Optional OpenCode launch with a suggested prompt (free model)

To make a target machine easier to start, add an explicit launch helper that opens one OpenCode instance with a suggested prompt and a locally available **free** OpenCode model:

```text
agentic-sdd opencode launch --free --workspace DIR [--task TEXT] [--model MODEL] [--dry-run]
```

Behavior:

1. Resolve `opencode` with `exec.LookPath`; never use a shell. Refuse with actionable guidance if it is missing.
2. With `--free`, run the read-only local model catalog (`opencode models`) and choose the first available route from the repository's free-first ladder:
   `opencode/deepseek-v4.1-flash-free` → `opencode/glm-5.3-flash-free` → `opencode/mimo-v2.6-flash-free` → `opencode/beam-free`.
   Never silently fall back to a paid route. An explicit `--model` overrides the ladder.
3. If no free route is available, exit 2 with the exact model names and setup guidance; do not launch.
4. Build a bounded suggested prompt from a fixed template plus the workspace path and optional `--task` text. The default should point at `AGENTS.md`, the installed Agentic SDD skills, and the current feature/task context. Do not read arbitrary files into the prompt and do not include secrets or credentials.
5. Launch the interactive instance with `opencode --prompt <text>` in the workspace, or use `opencode run --model <model> <message>` only when non-interactive execution is explicitly requested. Do not pass `--auto` by default; permission prompts remain the user's decision.
6. `--dry-run` prints the resolved executable, workspace, selected model, and exact argv without launching. The command is explicit; `preview`/`apply` must never launch OpenCode.
7. Keep this separate from the existing build-tagged `opencode session` evidence harness. It is a convenience launcher, not a model/session execution harness.

Example:

```sh
agentic-sdd opencode launch --free --workspace . \
  --task "Use the Agentic SDD skills to inspect this repository and propose the next safe step."
```

This gives the target machine a one-command start without copying a prompt or hunting for a free model, while preserving the no-paid-fallback and no-auto-approval rules. It does not itself call a model; OpenCode starts the session after the user confirms.

## 6. Availability checks and states

| Check | Method | Rule |
|---|---|---|
| OpenCode | `exec.LookPath("opencode")`, fixed `opencode --version` with timeout | Never shell; major version selects v2/v1 schema |
| Config file | `<home>/.config/opencode/opencode.json(c)` or `--config` | If both `.json` and `.jsonc` exist, block rather than guess |
| Config shape | Parse JSONC; inspect `mcp.servers` and legacy `mcp.<name>` | Preserve unknown keys; block ambiguity |
| CodeGraph | `exec.LookPath("codegraph")`; `codegraph --version`; optional `status --json` only with explicit consent | Do not auto-init `.codegraph/` during install |
| Node/npx | `node --version`, `npx --version` | Only needed for local Context7/CodeGraph npm paths |
| Existing servers | Parse config names | Report managed vs unmanaged; duplicates conflict |
| Credentials | `os.LookupEnv` only | Report `set`/`unset`; never print values |
| Network | Off by default | `doctor --network` only; short timeout; no credential-bearing request |

States:

| State | Meaning |
|---|---|
| `ready` | Config parseable, managed entries present/enabled, required binaries present, no conflict |
| `degraded` | Optional binary/credential missing, entry disabled, index absent or version unverified; still exits 0 |
| `blocked` | Malformed/ambiguous config, both config files, unmanaged conflict, symlink/special file, unwritable destination; exit 2 |
| `not_checked` | Network, OAuth and live MCP handshake were not attempted |

## 7. Config merge algorithm

1. Resolve global scope first; project scope only with explicit `--scope project`/`--config`.
2. Detect V2 `mcp.servers` vs legacy `mcp.<name>`; block on mismatch. Never silently migrate.
3. Preserve every unknown top-level key, unmanaged server and unknown field inside a managed entry. Never re-enable a server the user disabled.
4. If a managed name exists with different values, **conflict**: refuse unless `--replace <name>` is explicitly supplied.
5. Secrets use `{env:NAME}` placeholders only. Never write literal API keys or OAuth tokens.
6. JSONC comments are a problem: a normal marshal loses them. Refuse by default and print the exact snippet for manual insertion; allow `--allow-reformat` only as a warned explicit choice. Do not add a JSONC dependency just for this.
7. Back up the original to `<home>/.agentic-sdd/mcp/backups/<stamp>/` with `0600`, record the digest/mode, write a same-directory temp file, `fsync`, rename and `fsync` the directory. Never put config backups in the repo.
8. Record a sidecar `<home>/.agentic-sdd/mcp/manifest.json` (0600): managed names, config path, original/installed digests, backup path, tool version, timestamp, schema and actions. Do not add unknown fields to `opencode.json`.
9. Idempotency: compare canonical semantic JSON. Identical is “up to date” with no backup/write. `uninstall` removes only entries whose digest still matches; user-edited entries are refused.

## 8. Missing-piece installation

| Piece | Recommended action | Do not do |
|---|---|---|
| CodeGraph CLI | Official package/platform installer; pin a version; verify `codegraph --version`; then Agentic SDD writes the merged entry | Do not run `codegraph install` by default; it writes agent config and `AGENTS.md` |
| CodeGraph index | Per project: `codegraph init -y` only in an ordinary checkout, then `codegraph status` | Do not index home/root/backups/vaults or unrelated repos |
| Context7 | Nothing for the hosted server; optional `ctx7` CLI only for CLI lookup | Do not run `ctx7 setup --opencode` as the default config writer |
| Exa | Nothing; hosted remote. Keyless is rate-limited; OAuth/API key optional | Do not write keys into config or repo files |
| Node/npx | Report only; user installs via their package manager | Do not install Node silently |

Consent rules: print exact argv; require `--yes` or an interactive `y`; never in `preview`; refuse root unless explicitly overridden; timeout and bound output; record installed version; print the matching uninstall command. No `curl | bash`, no silent version drift. Pin versions in the manifest; floating `npx -y` is opt-in and labelled.

## 9. Proposed manifest

Embed a versioned `mcp-servers.json` through the existing root `go:embed` (no new package), for example:

```json
{
  "version": 1,
  "servers": [
    {
      "name": "codegraph",
      "type": "local",
      "command": ["codegraph", "serve", "--mcp"],
      "codemode": false,
      "requires": [{"binary": "codegraph", "minVersion": "1.6.2"}],
      "install": {"kind": "package", "manager": "npm", "package": "@colbymchenry/codegraph", "pin": "1.6.2"},
      "capabilities": ["codegraph_explore"]
    },
    {
      "name": "context7",
      "type": "remote",
      "url": "https://mcp.context7.com/mcp/oauth",
      "auth": {"mode": "oauth-or-key", "env": "CONTEXT7_API_KEY", "required": false},
      "install": {"kind": "none"},
      "capabilities": ["resolve-library-id", "query-docs"]
    },
    {
      "name": "exa",
      "type": "remote",
      "url": "https://mcp.exa.ai/mcp",
      "auth": {"mode": "oauth-key-or-keyless", "env": "EXA_API_KEY", "required": false},
      "install": {"kind": "none"},
      "capabilities": ["web_search_exa", "web_fetch_exa"]
    }
  ]
}
```

Treat the Context7 header/endpoint and Exa tool list as verified-at-implementation decisions, not frozen constants.

## 10. Security and privacy

- **Secrets:** placeholders only; read env presence, never values. Config backups may contain secrets: `0700` directories, `0600` files, never repo-local.
- **External queries:** Context7/Exa receive queries. Keep the existing rule that private code, paths and secrets never enter external searches; configuration alone does not enforce it.
- **CodeGraph:** `.codegraph/` is local, self-ignored and must stay uncommitted; never index private homes/backups/vaults.
- **Supply chain:** avoid vendor installers that write shared files; pin versions; record provenance; no root installs; no package installation in tests or CI.
- **Redaction:** status may show config path, server names, URL scheme/host and boolean credential presence; never dump full config or environment.

## 11. Testing and gates

Keep the **supported-core gate**. Put detection/merge/transaction code in existing `internal/skillsync` and CLI code in `cmd/agentic-sdd`; do not create a new package or import the build-tagged `internal/opencodeadapter` into the core path. Embed the manifest with the existing root `go:embed`.

Tests:

- `internal/skillsync`: V1/V2 detection, merge, unknown-field/server preservation, conflict refusal, disabled preservation, JSONC refusal, no-op, placeholders, malformed/symlink/both-config blocks.
- Golden configs: empty, V2 with unrelated servers, V1, JSONC comments, existing managed entry, conflicting entry, both `.json`/`.jsonc`.
- CLI: isolated `--home`; preview writes nothing; apply requires digest; injected failure restores original; backup/restore round trip; fake `opencode`/`codegraph`/`npx`/`node` scripts on `PATH`; no network.
- Refusals: root, missing consent, unknown schema, user-edited entry on uninstall.
- Gates: `task test`, `task vet`, `git diff --check`, `task preview`. `mcp doctor --network`, `mcp install-missing`, real MCP handshakes, OAuth and vendor installers remain `not_run` in offline evidence.

## 12. Phased plan

| Phase | Scope | Likely files |
|---|---|---|
| **P0** | Read-only `mcp status`/`doctor`; manifest; detection; no writes/network | `cmd/agentic-sdd/mcp.go`, `internal/skillsync/mcp.go`, `mcp-servers.json`, `skillsfiles.go`, tests, docs |
| **P1** | `mcp preview/apply/restore/uninstall`; merge, conflicts, backup, atomic write, sidecar | `internal/skillsync/mcp_config.go`, `mcp_merge.go`, `mcp_transaction.go`, CLI/goldens |
| **P2** | `mcp install-missing`; explicit network doctor; version checks | `internal/skillsync/mcp_install.go` with injected runner; refusal/consent tests |
| **P2.5** | `opencode launch --free` convenience launcher; free-model detection; suggested prompt; dry-run | `cmd/agentic-sdd/opencode_launch.go`, small model-catalog parser, tests |
| **P3** | `--with-mcp` convenience; TUI/appops preview tokens; project scope | `cmd/agentic-sdd/main.go`, `internal/appops/mcp.go`, TUI |

Start with P0. It tells the user exactly what is missing without changing config or installing anything; it also gives the default tool set a truthful readiness surface.

## 13. Open decisions for the owner

1. Should the default be **detect/report only** (recommended) or should `apply --with-opencode` write MCP config by default?
2. Global-only first, or project scope too? What if both `.json` and `.jsonc` exist?
3. Refuse JSONC comments or allow `--allow-reformat`?
4. Refuse conflicting managed entries or permit `--replace <name>`?
5. CodeGraph install: official package/platform installer vs a pinned npm package?
6. Context7 default: OAuth `/mcp/oauth` vs API key `/mcp`; exact header name to verify.
7. Exa default: keyless, OAuth, or API key; which tools list?
8. Per-server `codemode` values; CodeGraph prints `false`.
9. Separate MCP sidecar vs extending the skill backup manifest to format 4 (recommend separate).
10. Is `--with-mcp` needed, or are the subcommands enough?
11. Is any network probe acceptable, and what does it prove?
12. Should `opencode launch` be interactive by default or require `--launch`, and which free route wins when several are available?
13. Should the suggested prompt be task-specific, feature-aware, or a fixed safe template with `--task` appended?

## 14. Sources

- [OpenCode MCP servers](https://opencode.ai/docs/mcp-servers/)
- [OpenCode config](https://opencode.ai/docs/config/)
- [CodeGraph installation](https://colbymchenry.github.io/codegraph/getting-started/installation/)
- [CodeGraph MCP server](https://colbymchenry.github.io/codegraph/reference/mcp-server/)
- [CodeGraph CLI](https://colbymchenry.github.io/codegraph/reference/cli/)
- [Context7 all clients](https://context7.com/docs/resources/all-clients)
- [Context7 repository](https://github.com/upstash/context7)
- [Exa MCP](https://exa.ai/docs/get-started/exa-mcp)
- [Exa authentication](https://exa.ai/auth.md)

External documentation is advisory; verify exact headers, endpoints and config schemas against the installed versions during implementation.
