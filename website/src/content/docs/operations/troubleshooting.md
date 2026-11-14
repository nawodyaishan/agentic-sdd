---
title: Troubleshooting
description: Symptom, safe diagnostic, expected result and recovery for common failures.
---

Start with what the local tooling can prove. A configuration read, an `mcp doctor`
line or a cross-reference in a catalog tells you what a file says, not whether a
provider is reachable, a credential is valid or a server is connected. Read a
successful local probe as "nothing is obviously misconfigured on disk", never as
connectivity evidence.

Every diagnostic below is local and offline unless the step says otherwise. Quoted
messages are taken from this checkout's source, not from a captured session. Do not
paste home paths, logs, backups or transcripts into a public issue without reviewing
them first; see [Trust boundaries](../security/trust.md).

## Nothing changed after I ran `asdx-core`

**Symptom.** The command listed work but left the client directories untouched.

**Safe diagnostic.** `asdx-core help apply`. The bare command is a read-only preview by
design; only `apply` writes.

**Expected result.** The preview prints the planned actions and writes nothing.

**Recovery.** Re-run with the write mode your environment authorizes:
`asdx-core apply`. If you expected a specific client subset, see the next item.

## A destination path was refused

**Symptom.** One of these messages names a path and nothing in that tree was modified:

- `refusing special file or symlink <path>`
- `refusing non-directory target <path>`
- `refusing non-directory path component <path>`
- `refusing non-directory backup path <path>`
- `refusing non-directory log path <path>`

**Safe diagnostic.** `ls -ld` the named path and each parent, and compare it with the
destination table in [Files and backups](../reference/files-and-backups.md).

**Expected result.** A real directory where a skill directory belongs, and a real file
or absent path where a managed file belongs. The refusal happens before a write.

**Recovery.** Point `--home` at a home whose client directories are ordinary
directories, or replace a symlinked destination with a real directory *after* deciding
what the link pointed to. Do not delete an unrelated tree to make the installer pass;
that refusal is protecting content the tool did not create. Log paths also must not be
symlinks, so `no operation logs in <path>` and a log-path refusal are different
findings: the first means no log exists yet, the second means the path is unsafe.

## Client selection was rejected

**Symptom.** `unknown client <id>; supported clients are <ids>`, or
`client selection is empty; omit the selection to use every supported client`.

**Safe diagnostic.** `asdx-core help apply`. There is no CLI `--clients` flag in this
checkout; client selection is a terminal-UI option, and a direct CLI invocation with no
selection targets every supported destination.

**Expected result.** The four supported IDs are `agents`, `codex`, `claude` and `agy`;
their destination paths are owned by [Client destinations](../introduction/clients.md)
and [Files and backups](../reference/files-and-backups.md).

**Recovery.** Select clients in the TUI's Install & Manage route, or run the CLI without
a selection and let it update all supported destinations.

## Pack options were rejected

**Symptom.** `--opencode-free-first requires --with-opencode`.

**Safe diagnostic.** `asdx-core help tui` or `asdx-core help apply`. The free-first
preset changes the profiles that the pack installs, so it has no meaning without the
pack.

**Expected result.** Both flags accepted together, or the preset dropped.

**Recovery.** Add `--with-opencode`, or remove `--opencode-free-first`. If the message is
`opencode pack source is missing`, the selected `--repo` checkout has no
`opencode-pack/` directory: use the embedded assets by omitting `--repo`, or use a
complete checkout.

## The terminal UI will not start

**Symptom.** `error: terminal UI unavailable: <reason>`, followed by
`hint: run <cli> preview to use the standard CLI; nothing was written`.

**Safe diagnostic.** `asdx-core version`, then inspect the sibling binary's location and
identity. The launcher accepts only a real, executable, same-install `asdx` whose build
identity matches this CLI. Named refusals you may see:

- `the companion <companion> is not installed beside <cli>`
- `refusing the symlinked companion <path>`
- `the companion <path> is not executable`
- `companion identity (<parent>) does not match this CLI (<peer>)`
- `companion protocol revision <n> does not match this CLI's revision <m>`
- `build identity is not verifiable (<fields> missing or still a development placeholder)`

**Expected result.** Both binaries report the same version, commit and protocol revision.
A development build can legitimately fail the identity check: two locally built binaries
have independent identities until they are built as a matched pair.

**Recovery.** Install the paired release artifacts, or rebuild both binaries together.
Do not symlink one name at the other and do not substitute a different tool: the launcher
does not search `PATH` and does not fall back. The legacy `agentic-sdd-tui` name forwards
to `asdx` for one deprecation window; if you see
`deprecated compatibility name; running`, the wrapper worked and the target is the
canonical binary.

## The terminal UI refuses to open in a script

**Symptom.** `error: <cli> tui needs a real input and output terminal`, or a message
noting that `AGENTIC_SDD_FORCE_INTERACTIVE` keeps scripted runs on the standard CLI.

**Safe diagnostic.** Check whether stdin and stdout are terminals and whether
`AGENTIC_SDD_FORCE_INTERACTIVE=1` is set. The launcher refuses to open an alternate
screen in a pipe or a forced-interactive script.

**Expected result.** No alternate screen, no partial UI, and nothing written.

**Recovery.** Run `asdx-core preview` for scripted checks, or run `asdx` directly from a
real terminal. If a session ends with `the terminal UI terminated abnormally`, the child
was signalled; re-run from a terminal rather than treating it as a successful operation.

## A model route behaves unexpectedly

**Symptom.** A profile's model does not appear to be used, or a fleet file is rejected
with `invalid model <route>: want <provider>/<model>[#variant]`.

**Safe diagnostic.** `asdx-core fleet doctor --config <path> --opencode-bin <path>` when
you have an explicit local OpenCode executable. Ordinary
`fleet doctor` is offline and reports `not checked (offline)` with the reason
`offline default: no local tool was offered, so nothing was checked`.

**Expected result.** Doctor labels are `configured`, `catalog-visible`, `authenticated`,
`unknown` and `unavailable`. A `catalog-visible` label means a local catalog listed the
route; `authenticated` means a local authentication marker was present. Neither means the
model works, that quota exists or that the provider will serve it. Only a host-side
successful call can show that.

**Recovery.** Fix the route grammar first. Then confirm provider configuration and
credentials in the host agent, not in Agentic-SDD. Treat any local probe as configuration
evidence only. See [Agent profile catalog](../reference/profiles.md) and
[Fleet reference](../reference/fleet.md).

## The documented fallback did not run

**Symptom.** A pinned model failed and nothing switched automatically.

**Safe diagnostic.** Read the profile's routing note and the global policy. The shipped
statement is `A written ladder is policy, not automatic failover`: the parent decides
whether to retry, checks compatibility first, and reports the model that actually ran.

**Expected result.** The agent stops or asks; there is no hidden failover engine. Free
ladders stay on free models. A paid rung needs separate human authorization and may
incur charges.

**Recovery.** Choose the next rung yourself, check context/output compatibility before
switching, and record which model actually ran. A listed route is a documented
preference, not a promise: provider access, credentials, quota, pricing, availability and
runtime use are not claimed by Agentic-SDD.

## Both MCP configuration files exist

**Symptom.** `both <opencode.json> and <opencode.jsonc> exist; refusing to guess which target to manage`.

**Safe diagnostic.** `asdx-core mcp status` and `asdx-core mcp preview`. Name one target
explicitly with `--config` if you know which file your host reads.

**Expected result.** No target is chosen while two defaults exist; nothing is written.
The same rule applies to a symlink or out-of-scope target.

**Recovery.** Consolidate to one configuration file, or pass `--config` explicitly.
There is no "pick newest" or "merge both" behaviour in this build.

## MCP was configured but still does not answer

**Symptom.** The managed entry exists but a client still cannot use the server, or
`unknown server <name>` / `codegraph is a local server and has no authentication choice`
appears while choosing a server.

**Safe diagnostic.** `asdx-core mcp preview` and `asdx-core mcp doctor`. Managed names are
`codegraph`, `context7` and `exa`. Configuration is not authentication: remote entries
use `{env:NAME}` placeholders that the host must resolve.

**Expected result.** A written configuration and an honest auth state. No command performs
a network call, OAuth flow or connection test; asking for one with `--network` or
`--connect` returns `is not available in this build; network and connection checks are deferred`.

**Recovery.** Set the referenced environment variables in the host's environment, restart
the host, and verify the server there. A successful config preview is not a connectivity
claim. See [MCP reference](../reference/mcp.md) and [MCP how-to](../how-to/mcp.md).

## MCP apply refused the plan

**Symptom.** One of these refusals, with a `note:` line stating what was and was not touched:

- `nothing was written; run <preview command> again and review the new digest` — the target changed after preview, or the plan is stale;
- `nothing was overwritten; v1 has no replace or adopt option` — an existing conflicting entry the plan will not take over;
- `contains a literal credential in <field>; move it to an {env:NAME} placeholder before applying`;
- `no local opencode executable was found; pass --opencode-bin` — the runtime probe could not find a local OpenCode binary;
- `nothing was read or written outside the intended target` — an unsafe target was refused.

**Safe diagnostic.** Re-run `asdx-core mcp preview` and compare digests. Check that the
selected file is the one the host reads and that its entries use environment placeholders.

**Expected result.** Either a new digest to review, or a refusal that changed nothing.

**Recovery.** Apply the digest from the fresh preview. Replace literal secrets with
placeholders before retrying, and pass `--opencode-bin` when OpenCode is not on `PATH`.
Do not hand-edit the ownership manifest to force an apply; it records what the tool
installed.

## A permission was denied

**Symptom.** The operating system reports a permission error, or the host agent refuses a
tool call the skill asked for.

**Safe diagnostic.** Split the two cases. If the refusal comes from the host agent, the
message and the rule are the host's: Agentic-SDD does not intercept tool calls or widen
permissions. If the error names a path, check its owner and mode, and check whether the
selected home is writable by the current user.

**Expected result.** Installation writes only under the selected home (or the explicit
`--repo` checkout for its backup root); logs and MCP state are owner-only. Agent profile
rules apply only as far as the host enforces them.

**Recovery.** Fix ownership or mode on the selected home, or choose a different `--home`.
Do not run the installer with elevated privileges and do not widen a directory to
world-writable to make an install pass. For host-side denials, change the profile or the
runtime permission configuration deliberately; see [Trust boundaries](../security/trust.md).

## `run check` will not start

**Symptom.** `run check requires -- before the command vector` or
`run check requires a command after --`.

**Safe diagnostic.** `asdx-core help run`. The separator is required, there is no
implicit shell, and globs are not expanded.

**Expected result.** A command vector is captured exactly, with your permissions, in the
repository root.

**Recovery.** Use `asdx-core run check -- run <args>`, or `sh -c '<command>'` if shell
semantics are what you intend. Remember that the check inherits your authority and
environment; see [Execution run reference](../reference/execution.md). Related usage
errors, such as `--feature and --batch are required unless --direct is used` and
`--run and --result are required`, mean the required identifying flags were omitted; run
the family's `help` topic for the accepted grammar.

## When to stop and ask

Stop and ask a maintainer instead of improvising when:

- a refusal names a path you do not recognize and cannot explain;
- a build-identity or protocol mismatch appears after you installed what you believed was
  a single release;
- resolving a conflict would require deleting another tool's files, editing a manifest
  by hand, or running as root;
- the only remaining explanation is that a provider, model or MCP server "should" work.

If the failure is a suspected defect in your own code rather than a refused path or a rejected flag, work it as a diagnosis with the [Research and debugging tutorial](../tutorials/research-and-debugging.md) instead of retrying flags.

Report the command, its exact message, and what you expected. For a sensitive report,
follow the channel described in `CONTRIBUTING.md` rather than opening a public issue with
details. Untested capabilities are listed in
[Known limitations](known-limitations.md).

## Browser does not open from the TUI

Expected on headless or SSH sessions, on unsupported platforms, or when the platform opener fails: the TUI does not claim success. The exact URL is shown in the `Notice:` footer — open it manually in a browser on your local machine.
