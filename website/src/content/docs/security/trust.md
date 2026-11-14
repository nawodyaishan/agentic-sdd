---
title: Trust boundaries
description: What Agentic-SDD reads and writes, what it stays out of, and which limits are real.
---

Agentic-SDD installs instruction files and agent profiles, then gets out of the way. It
runs no daemon, holds no model-provider credentials, and cannot turn the coding agent
that loads its skills into a sandbox. This page states the boundaries the source
enforces, so you can decide what to allow. It is a factual description of the code, not
a security certification.

## Files and effects

The installer copies packaged Markdown skills, and optionally the OpenCode pack
(`agentic-sdd-orchestrate` plus agent profiles). Destinations, backup roots and
manifest formats are listed once in [Files and backups](../reference/files-and-backups.md).

| Action | Reads | Writes |
| --- | --- | --- |
| `asdx-core` (bare, `preview`) | packaged assets or `--repo` checkout; existing destinations | nothing |
| `asdx-core apply` | the same, plus existing destinations | backup directory, skill directories, optional profile files, one operation log |
| `asdx-core backups`, `logs`, `restore` without `--apply` | backup store or log directory | nothing |
| `asdx-core restore --apply` | the selected backup | destinations and one operation log |

Preview is the default; `apply` is the only mode that changes a destination
(`cmd/agentic-sdd/main.go`, `internal/skillsync/sync.go`). One explicit exception:
`fleet plan --output <path>` writes the plan file you named even though no destination
changes (`cmd/agentic-sdd/fleet_plan.go`, `internal/fleet/plan.go`). "Preview writes
nothing" describes the installer, not a path you asked the tool to create. A skill that already
matches its source is reported as up to date and is neither replaced nor backed up
again. When `--repo` is omitted, assets come from the binary's embedded tree; an
explicit checkout reads `skills-files/` and keeps backups under the checkout's own
`backups/` directory.

Path integrity is enforced before any write:

- special files and symlinks inside a managed tree are refused
  (`refusing special file or symlink <path>` in `internal/skillsync/files.go`);
- a non-directory where a skill directory or parent component belongs is refused
  (`refusing non-directory target <path>`, `refusing non-directory path component <path>`);
- a backup root or log directory that is not a real directory is refused
  (`refusing non-directory backup path <path>`, `refusing non-directory log path <path>`).

The installer writes that tree with its own modes rather than trusting the checkout:
skill directories are 0755 and files 0644 subject to the process umask; restore
normalizes modes instead of restoring whatever modes a backup contained
(`internal/skillsync/permissions_test.go`). Unrelated skills in the same client
directory are left alone.

## Network and child processes

Every row below is a source-derived statement about the current checkout, not a
measurement of a running session.

| Surface | Network | Child process |
| --- | --- | --- |
| `preview`, `apply`, `restore`, `backups`, `logs` | none; the install path does not call the release check | none |
| `asdx-core guide`, menu choice 5 only | one HTTPS `GET` to the fixed release endpoint | none |
| `asdx` (TUI) | one automatic release check per session unless disabled | none |
| `asdx-core tui` | none from the launcher; it starts the sibling `asdx` | `asdx` plus its `--protocol-info` handshake |
| `mcp status`, `doctor`, `preview` | none | none; resolves a local executable with a bounded `PATH` lookup |
| `mcp apply` | none | local `opencode --version` probe, via `--opencode-bin` or `PATH` |
| `mcp restore list`, `preview`, `apply` | none | none |
| `fleet plan --output`, `fleet apply` | none | none |
| `run`, `task`, `review` | none in the service itself | read-only `git` probes with an isolated environment; `run check` also runs the command you supply |
| `agentic-sdd`, `agentic-sdd-tui` | same as the canonical binary they forward to | replaces itself with that sibling |

The only network code in the CLI is the release check in `internal/appops/update.go`:
one fixed endpoint, no authentication, no endpoint override, redirects refused, a
five-second budget and a 64 KiB response cap. It reads release metadata only; it never
downloads, installs or upgrades anything. The CLI reaches it only through the numbered
guide's explicit update choice (`cmd/agentic-sdd/guide.go`). The TUI performs the same
check once per session automatically, and `AGENTIC_SDD_NO_UPDATE_CHECK=1` disables it
(`experiments/tui/update_state.go`, `experiments/tui/app_screens.go`); the numbered CLI
guide has no automatic equivalent.

Execution's repository probes are the one other child-process surface: `git` is invoked
read-only with global and system configuration neutralized and optional locks disabled
(`internal/execution/repository.go`). That keeps a repository probe from picking up a
user's Git configuration; it does not sandbox the check command you run yourself.

The TUI companion is resolved as a real sibling of the running binary and identity-checked
before it starts; a missing, symlinked or mismatched companion is refused rather than
silently replaced by another workflow (`cmd/agentic-sdd/tui.go`,
`internal/buildinfo/buildinfo.go`). Opening a documentation link hands a fixed
repository URL to the operating system's opener; the tool itself performs no request
(`experiments/tui/opener.go`).

## Is my data sent anywhere?

Agentic-SDD ships no telemetry and no analytics. The installer, restore, backup, fleet and
MCP paths send nothing; they read and write local files only. The two exceptions are
outbound release-metadata checks: the CLI `guide` screen's release check and the TUI's
once-per-session update check (which honours `AGENTIC_SDD_NO_UPDATE_CHECK=1`), both to
the fixed GitHub releases endpoint. Anything a coding agent or model provider does with
your prompts or code is controlled by that client, not by these skills.

## Credentials, providers and MCP

No product environment variable carries a provider credential, and no Agentic-SDD
command authenticates to a model provider. Provider logins, API keys, quota and model
availability belong to the agent host that loads the skills; see
[Configuration reference](../reference/configuration.md) for the selectors that do exist.

For MCP, the managed write is deliberately narrow (`internal/skillsync/mcp_config.go`):

- remote entries are written with `{env:NAME}` placeholders such as
  `{env:CONTEXT7_API_KEY}`, never with a stored secret;
- a selected managed entry that contains a literal credential is refused with
  `contains a literal credential in <field>; move it to an {env:NAME} placeholder before applying`;
- only managed entries under `mcp.servers` are read or replaced; unrelated servers,
  comments it cannot preserve and credentials stay outside the write;
- OAuth credentials remain the host's, and no command performs an OAuth flow.

`mcp status`, `doctor` and `preview` are read-only and offline, and ordinary `doctor`
executes no binary. Applying a reviewed plan does start a local `opencode --version`
probe for runtime evidence; when no executable is found the refusal is
`no local opencode executable was found; pass --opencode-bin`. Configuration is not
authentication and not connectivity: see [MCP reference](../reference/mcp.md).

## Instructions are not an operating-system boundary

Skills and profiles are text. Installing them changes what an agent is told and, for
the optional pack, which permission rules the host should apply. It does not change the
kernel, the filesystem boundary, the network policy or what a shell can do.

The pack states this directly: it "changes instructions and permissions, not the
operating-system sandbox", and a prompt constraint is not filesystem isolation
(`opencode-pack/README.md`). Two consequences matter:

- A permission rule is enforced only if your host enforces it. Deny-by-default profile
  rules and risk labels are host-facing configuration, not a universal guarantee.
- The installer's symlink and path checks protect its own managed trees. They are path
  containment for an installer, not a sandbox for the agent that later reads the skills.

Research, review and delegation instructions can tell an agent to stay read-only. That
is guidance, not enforcement, and a review verdict never replaces human approval
(`opencode-pack/README.md`). The historical native independent-review work is fenced
out; see [Known limitations](../operations/known-limitations.md).

## Backup, argv and log limits

These are the places where local state can outlive the reason it was written.

- **Backups are copies, not archives of intent.** A backup holds the content that was
  replaced, owner-only (0700 directories, 0600 manifests). It may contain superseded
  instructions or locally customised skills. Restore validates the backup's home
  identity and content before use, and its digests detect corruption; they are not
  signatures and do not prove who wrote the tree (`internal/skillsync/manifest.go`,
  `internal/skillsync/restore.go`).
- **Argument vectors are process-visible.** Flags are ordinary process arguments, so
  another process running as the same user can read them while a command runs. Do not
  pass secrets as command-line flags. The documented trace records flag *names* only,
  never values (`cmd/agentic-sdd/trace.go`).
- **Logs are local and owner-only, not secret-free.** The operation log lives under
  `<home>/.agentic-sdd/logs`, is created 0600 inside a 0700 directory, and records the
  command verb, source selection, absolute home path, transcript and result
  (`cmd/agentic-sdd/logs.go`). Treat it as private machine state.
- **Run scratch is private, not encrypted.** `.agentic-sdd/runs/` keeps task briefs,
  check logs, records and review packages. Default sensitive paths such as `.env`,
  private keys and credential files are omitted and listed as omissions
  (`internal/execution/reviewpackage.go`). Never publish raw run artifacts.
- **Execution inherits your authority.** `run check` runs the command you give it with
  your permissions; the service adds no isolation, and a run is never approval,
  verification or a sandbox (`internal/execution/checks.go`, [Execution run reference](../reference/execution.md)).

## Safe execution

A workable order for a first real installation:

1. `asdx-core version` — confirm which binary and build you are running.
2. `asdx-core preview --home <temporary-directory>` — read the planned changes; the
   preview writes nothing.
3. `asdx-core apply --home <temporary-directory>` — observe the backup, destinations
   and log in a throwaway home.
4. Review the plan for your real home, then apply only with the authorization your
   environment requires. This documentation's own examples never write to a real home.
5. Keep the backup identifier: `asdx-core restore` is the undo path, and
   `asdx-core logs` shows the last operation.

Stop before applying anything when the plan lists a destination you did not expect, when
a managed path is a symlink you did not create, or when the only reason to run a command
is that a check would otherwise fail. Recovery steps for specific symptoms are in
[Troubleshooting](../operations/troubleshooting.md).

## Reporting a security concern

`CONTRIBUTING.md` owns the reporting channel. For a sensitive report, use the
repository's private vulnerability reporting option if it is available. Otherwise open a
public issue that asks for a private channel without disclosing the vulnerability. No
dedicated security email address and no response-time guarantee are documented, so do
not assume a private channel exists and do not post credentials, private backups, raw
session traces or exploit details publicly.
