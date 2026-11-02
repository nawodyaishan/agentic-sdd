# Inactive configuration helper — 005/B2

The `agentic-sdd opencode` group prepares and checks **file inputs only**. It
does not invoke `opencode`, connect to a native/shared service, probe a model,
read a credential store or certify runtime enforcement. Source declarations in
its output are labeled `target_runtime` and `candidate_*`, not actual native
observations. Support remains [uncertified](supported-version.md).

The intended candidate is exact OpenCode 2.2.5 with the selected synthetic-only
Nemotron Ultra `:free` planning reference. All providers remain denied. The
candidate adds current-session helper commands and uses supported typed skill,
plugin and MCP denials; it is a new inactive proposal, not a re-certification of
004's older template. No alternative/fallback or online operation is enabled.

## File-only operations

From a checkout, `go run ./cmd/agentic-sdd opencode --help` prints the same help
as a newly built binary. Existing installed release binaries may lack this group.
The examples below describe user operations; they were not run on the real home.

```sh
agentic-sdd opencode candidate --feature 005-vnext-completion --build-path skills-files
agentic-sdd opencode preflight --config /absolute/private-staging/candidate.jsonc
agentic-sdd opencode config preview --workspace /absolute/project --candidate /absolute/private-staging/candidate.jsonc
```

`candidate` emits inactive JSON to stdout, not to an automatically discovered
path. Review it and, when desired, save it outside the project and active skill/
configuration discovery. Do not redirect it into a live config as an unreviewed
shortcut. Empty `--feature` omits feature-document edit grants for direct fixes.
`--build-path` is an explicit narrow literal project-relative directory.

`preflight` validates the candidate's fixed inactive model/policies/profiles and
projects only metadata. It always reports `online_ready: false`, including after
an optional `--observation FILE`. That file accepts only bounded fields: `version`,
profile/command/skill ID arrays, enum warnings (`invalid-policy`, `ignored-setting`,
`unknown-source`) and `truncated`. Its origin is **caller-supplied-not-verified**;
unknown/truncated/mismatched native state remains a blocker. B3 will supply owned
native lifecycle integration; B2 does not silently start a process to inventory it.
Do not supply full API/config/transcript exports, account data or native databases.

Configuration inputs support UTF-8 JSON/JSONC comments and trailing commas for
validation. Staged/original bytes are preserved, never round-tripped/merged.
Duplicate keys, unresolved placeholders, missing protected denials, review write
grants, child/shell-interpolating commands and known secret-bearing fields are
rejected without echoing rejected values. This is not universal secret detection:
the user must curate inputs and keep all credentials out of them.

## Explicit config transaction — separate live authorization

This capability is implemented/tested on temporary files, **not permission to
install into the user's setup or start the final human usage cycle**. Keep all
selected implementation complete before that later launch. The root skill
installer's `apply`/`restore` and manifests are unchanged and separate.

`opencode config` defaults to read-only preview. It emits the fixed relative
destination `.opencode/opencode.jsonc`, original digest (or `absent`), candidate
digest and no-op flag. The candidate must be external to the selected project and
active discovery. Preserve unrelated settings/comments by manually curating a
whole proposed file; no automatic JSONC merge, ancestor/global config rewrite,
service reload/restart or credential migration is performed.

Only after actual separately scoped authorization, explicit `config apply`
requires that preview's `--expect-original` and `--expect-candidate`, plus an
existing owner-private `--backup-root` separate from project/candidate/discovery.
Re-check the complete candidate diff; hash pins identify bytes, not user consent
or policy effectiveness. Existing config containing credentials must be reconciled
by the user rather than copied through this helper.

Before replacement, the helper reserves a private transaction directory, backs
up the original bytes/mode and persists `manifest.json`. It refuses changed
inputs/parents, symlinks/special files and intervening edits. Identical bytes
produce no rewrite or backup. Transactions assume a quiescent trusted filesystem;
they are not a race-proof sandbox against a concurrent hostile writer.

`config rollback --workspace DIR --backup TRANSACTION_DIR` validates the distinct
runtime-config manifest and backup/current digests and mode. It restores original
bytes/mode or removes only a verified newly added config file; it does not remove
the `.opencode` directory or unrelated state. A later edit blocks rollback. A
repeated already-restored rollback is a no-op, not a new backup.

If apply/rollback fails after backup reservation, the CLI exits nonzero and emits
an `execution: incomplete` transaction observation when possible. Preserve the
backup and reconcile phase/current files before any retry; `prepared` is not
proof that replacement occurred or that nothing changed. Do not delete backup
directories to retry, treat wrapper exit as model success or put private backups
in Git. Native prompt execution/receipts are later B3/B4, not delivered by B2.
