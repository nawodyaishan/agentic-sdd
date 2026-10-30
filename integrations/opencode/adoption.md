# Manual adoption and rollback

**Not installed or certified.** Keep the shipped JSONC files under
`integrations/opencode/` until the applicable native/tool/model blockers in
[verification.md](verification.md) are resolved. Development/batch approval is
not installation authorization. The existing skill installer handles skills
only; its manifests must not be used for runtime config.

## Preview — no active changes

1. Establish the exact CLI/service version and Location separately. Pin 2.2.0
   for this candidate. Inventory global/system/Console settings, all ancestor
   direct and `.opencode` configs, commands, agents, skills and executable
   plugins. Do not execute discovered slash commands to inspect them. Export
   only sanitized paths, IDs, versions and source digests, never entire configs,
   credentials, headers or native databases. API source lists do not prove that
   an invalid or ignored setting was enforced.
2. Resolve collisions explicitly. Ordinary `.opencode` configs override direct
   configs even above/below repository boundaries; policy precedence reverses
   that and broader/global/system sources can deny lower-level allows. Commands
   from Markdown and JSON share a registry. Inspect **all effective commands**
   for shell interpolation blocks; permissions do not guard those blocks.
3. Stage a proposal outside active discovery and Git, in a user-owned 0700
   directory. Use `.opencode/opencode.jsonc` as the eventual one project-config
   convention. Preserve unrelated keys/commands and JSONC comments. Conflicting
   `sdd-*` definitions, duplicate keys, ambiguous JSONC edits or unknown sources
   require manual resolution; do not concatenate documents or blindly copy the
   whole template over an existing file.
4. Replace `__FEATURE__` with the one agreed folder slug and `__BUILD_PATH__`
   with the narrow agreed source directory. For direct fixes, omit the unused
   feature/task edit grant instead of inventing a feature. Review the resulting
   literal paths. Add any further exact files/directories as explicit reviewed
   rules, not `*` or all of `specs/*`. Do not enable glob/grep in review: their
   resources are patterns/queries, not a filesystem isolation boundary.
5. Keep providers denied. The selected DeepSeek name is a default preference,
   not tested access or billing authorization. All command and agent model
   references must agree with the reviewed selection; primary-agent selection
   alone does not change a session model. Complete [safety.md](safety.md) before
   a separately authorized online test. MCP and unapproved plugins remain
   disabled. Some enforcement/Console builtin plugins cannot be disabled per
   V2 docs; use a clean runtime with no untrusted plugin discovery or accounts.
6. Show the complete staged diff, allowed effects and denial assumptions. Test
   in an isolated credential-free fixture first. A review workspace must be a
   sanitized **read-only export** with no private host mounts. Read rules and
   secret filename patterns are defense in depth, not a complete secret scanner
   or canonical/symlink-path proof. Record unresolved routes as unavailable.

## Backup, apply and verify — separate explicit authorization

1. Require no concurrent config writer. Inspect every parent from the reviewed
   root to the destination with `lstat`, rejecting symlinks/non-directories;
   reject destination symlinks/special files with `lstat` too. Missing parents
   can be explicitly created under the reviewed root. Refuse a source/staged
   symlink or a backup path inside active discovery. Recheck before replacement.
2. Compare current and staged bytes/digests. Identical content is a no-op:
   **no rewrite and no new backup**. Otherwise record original existence,
   canonical destination, source digest, original digest/permissions and planned
   installed digest. In 0700 external backup storage, save each existing file
   byte-for-byte with 0600 backup/manifest permissions **before** replacement.
   Record a missing original explicitly; do not manufacture an empty original.
3. After the user's reviewed installation decision, replace only those files,
   preserving appropriate original permissions. Use staged content on the same
   filesystem for an atomic file replacement where available. Recheck identity
   and expected pre-change digest immediately before replacing. If anything
   changed, stop and reconcile; never overwrite an intervening edit.
4. Verify current digest and effective profile/command/source inventory and
   startup warnings. Permission probes must include both allowed and denied
   cases; a configuration that denies everything is not a functioning draft
   mode. Preserve unknown observations. OpenCode reloads configuration; do not
   restart/change the shared service just to adopt settings.

## Rollback — digest checked

1. Recheck parents/destination type and identity; do not follow symlinks. Check
   that current bytes still match the manifest's **installed digest**. If not,
   preserve them and request reconciliation. An old backup is not authority to
   discard later edits. Check backup bytes against the **original digest**.
2. If the file existed before installation, restore its exact backed-up bytes
   and permissions. If it was freshly added, remove only that exact file after
   checking its installed digest and original absence. Do not delete the whole
   `.opencode` directory or unrelated definitions. Verify effective config again.
3. Keep the manifest, outcome and retained backup under the user's retention
   policy. Never use this process to copy or restore a credential database.

`tests/harness/test_opencode.py` exercises these operations on synthetic plain
JSON/temp files, including unrelated fields, no-op, original/new-file rollback,
intervening edits, parent/destination symlinks and a FIFO. It is a procedure
exercise, **not** a generalized installer, automated JSONC merge or proof of
race-resistant installation in an adversarial filesystem. No real-home
adoption/rollback has been performed.
