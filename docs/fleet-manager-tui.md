# Fleet Manager in the terminal UI

The Fleet Manager is the agent-profile surface inside the packaged terminal UI
(`agentic-sdd tui`). It shows the instances configured in one personal
`fleet.json`, stages model changes, previews the managed OpenCode agent files
that would change, and lists managed backups for restore.

It is a UI over the same rules as the `agentic-sdd fleet` CLI. It reads no
credentials, invokes no model, makes no provider or network call, and makes no
authentication, entitlement or runtime claim. Nothing here is a screenshot or a
recorded run: the wording below describes behavior and keys in the current
build.

**Status of the current build:** deployment and restore are **preview only**.
The review and recovery screens show the exact plan, but the live write is
disabled unless a reviewed build enables it. The source editor also cannot save
`fleet.json` yet; saving the personal source is a separate, explicitly approved
capability that this build does not ship.

**Not in this build:** clearing an existing model override back to canonical
inheritance is a CLI/editor operation in v1 (L-03); the in-app picker accepts
exact replacement model values, but removing an override key requires editing
`fleet.json` directly. Also note that `--fleet-config PATH` can preselect the
source at launch (`agentic-sdd tui --fleet-config PATH`) without manual path entry.

## Where it lives

Install & Manage → Fleet Manager, or press `f` from the Install & Manage hub.
The breadcrumb reads `Install & Manage / Fleet Manager`, and the surface has
four views:

| View | Key | Shows |
|---|---|---|
| Overview | `1` or `o` | Explicit source path and read state, installed agents, counts, last outcome, deployment capability |
| Profiles | `2` or `p` | Configured instances, filter, staged model and enablement edits |
| Review | `3` or `c` | The deployment plan for the managed files, with a default-No confirmation |
| Recovery | `4` or `e` | Managed backups, restore preview, default-No confirmation |

`r` reloads the view (and re-lists backups in Recovery); `?` opens the help.

## First use

Pick one option. None of them scans a home directory, guesses a private path,
clones a repository or contacts a Git host.

1. **Use an existing local file.** Press `s` on Overview and type the exact path
   of your `fleet.json` (or pass `--fleet-config PATH` at launch). The path is read as given; a symlinked or escaping path,
   a non-regular file or an unsupported path is refused rather than followed.
2. **Just look.** Read the **Fleet Manager** topic in Activity & Help, or this
   guide. Both are read-only and create nothing.
3. **Prefer the CLI.** Run `agentic-sdd help fleet`, then
   `agentic-sdd fleet status --config PATH` to inspect the same source without
   the UI.

The selected source is a session choice. It is not written to a preference file,
and the next session starts with no source until you choose one again.

## Five things to keep apart

- **Role vs instance.** A *role* is the job an agent profile does, such as
  reviewer or implementer. An *instance* is one configured entry with its own
  ID, provider, model and enabled state. Two instances can share one role.
- **Personal source vs rendered file vs installed agent.** The personal source
  is the `fleet.json` you own. The rendered file is the managed OpenCode agent
  file generated from it. The installed agent is the file OpenCode actually
  loads. They can differ until you apply and OpenCode reloads.
- **Installed is not selected primary.** An installed profile says nothing about
  which model is configured as primary elsewhere.
- **Installed is not authenticated.** No provider credential, account state or
  keychain entry is read or checked here.
- **Installed is not runtime-verified.** No model is invoked, so nothing here
  shows that a model answered, that a fallback worked, or that a provider is
  reachable.

## The journey

1. **Choose the source.** Overview shows the explicit path and the read state.
   With nothing selected it stays `(none selected)` — there is no default file.
2. **Change a model.** In Profiles, select an instance and press `m`, type or
   pick a `<provider>/<model-id>[#variant]` value and confirm. The edit is staged
   in memory and the source preview refreshes; the file on disk is unchanged.
   `t` stages an enable or disable change. A disable of an unmanaged file is
   refused with an explanation instead of a force option.
3. **Save the personal source.** A separate, explicit write. It is **not
   available in this build**: edit `fleet.json` in your own editor, then press
   `r` to reload. Staged edits that were never saved do not survive the reload,
   and the UI warns before discarding them.
4. **Preview the deployment.** Review shows the exact create/update/removal/no-op
   diff for the managed files, including any conflict. Previewing writes nothing,
   creates no backup and acquires no lock.
5. **Apply with default-No.** Deployment requires an explicit `y`. Enter, `n` and
   Esc all mean No. In a build with live deployment enabled, an apply takes an
   operation lock, backs up every replaced managed file and rolls back on partial
   failure. In this build the same screen reports **Preview only**.
6. **Restore a managed backup.** Recovery lists the managed fleet backups. A
   restore previews the exact current-versus-restore files, and the restore also
   requires an explicit `y`. Legacy, unknown, modified-target and corrupt backups
   are refused.

## Conflicts and limits

- **A modified managed file is a conflict.** If an installed managed file was
  edited by hand, the plan stops for a human decision; the UI never merges text
  speculatively and never repairs it silently.
- **No automatic adoption, bulk repair or force.** Unmanaged same-name files are
  left untouched. Adoption remains an explicit CLI decision with its own backup.
- **No subscription, price, entitlement or quota state** is shown, checked or
  promised. Model selection is a name, not a billing claim.
- **No Git operation.** Nothing is committed, pushed, tagged or sent to a remote;
  choosing a path is not repository access.
- **No new persistence.** The source choice, staged edits and view state are
  session-only.

## CLI equivalents and further reading

```sh
agentic-sdd fleet status  --config PATH   # read-only source and instance state
agentic-sdd fleet plan    --config PATH   # preview the managed-file changes
agentic-sdd fleet apply   --plan PATH     # apply a reviewed plan (separate authorization)
agentic-sdd fleet restore --backup ID     # preview a managed-backup restore
agentic-sdd fleet doctor  --config PATH   # opt-in local-only model-route diagnostics
agentic-sdd help fleet                    # the CLI reference for these commands
```

- [Fleet profiles worked guide](../specs/014-agent-fleet-model-overrides/fleet-worked-guide.md)
  — the canonical CLI workflow: stable JSON schema, strict parser rules, plan
  identity, apply locking and backup/rollback behavior, and the entitlement
  caveats for provider lanes.
- [Feature 015 spec](../specs/015-fleet-manager-tui/spec.md) — the Fleet Manager
  requirements and acceptance criteria this surface is built against.
- [TUI architecture](tui-architecture.md) — how the packaged TUI and the nested
  experimental module relate.

Nothing in this guide authorizes a live apply, a real-home write, a release or a
publication. Those remain separate human decisions.
