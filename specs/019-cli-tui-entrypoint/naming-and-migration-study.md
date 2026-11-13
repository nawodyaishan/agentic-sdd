# Naming and safe migration study — short default command

**Status:** Advisory study, 2026-11-13. **Not a specification, approval, batch authorization or release plan.** It does not change Feature 019's authority; any adopted name or alias must be reconciled into `spec.md`, `plan.md` and `tasks.md` and receive combined human approval before implementation.

**Question:** Can `agentic-sdd` use a shorter, easier default command such as `asdd` or `asd`, and how should that be migrated safely?

**Method:** Exa web searches on 2026-11-13 for existing CLI tools, packages, GitHub projects and same-domain agentic/SDD products using candidate names. Search results are evidence of collision risk, not a trademark, package-registry or shell-availability reservation.

## 1. Finding: the obvious short names are already occupied

| Candidate | Existing use found | Assessment |
|---|---|---|
| **`asdd`** | OneHillAI/ASDD — “Agentic Spec-Driven Development” CLI (`asdd init`, installs `asdd`); efactory/ASDD framework and `/asdd-bootstrap`; ericrisco `asdd` skill | **Reject.** Direct same-domain collision with an agentic SDD product. |
| **`asd`** | `asd` terminal session multiplexer (`asd new`, `asd ui`); `asd` diff viewer; `asd` file viewer; `asd`/`via` tunnel CLI; `asd` Android SDK downloader; common medical acronym | **Reject as installed default.** Multiple active CLI collisions. |
| `sdd` | `sdd` spec-driven CLI forks and plugins; SDD Tool; other spec-driven projects | Reject. Same-domain and generic. |
| `sddx` | `sddx` spec-driven development with proof; `sddx-workflow` | Reject. |
| `specd` | `@specd/cli`, SpecD spec-driven platform | Reject. |
| `asdl` | Abstract Syntax Description Language / CPython ASDL tools | Reject; compiler-domain collision. |
| `asdk` | SDK abbreviation; multiple projects and products | Reject; ambiguous. |
| `asdgo` | Go web framework `asdgo` | Reject. |
| `asdkit` | ASD anomaly-detection toolkit | Reject. |
| `asdcli` | ASD/Viaalink CLI installer and docs | Reject. |
| `asdf` | Very widely used version manager | **Never use.** |
| `agsd` | AGSD visual-novel script engine format | Weak collision; possible fallback but less intuitive. |
| **`asdx`** | No notable CLI found; minor unrelated uses (`asdx.jar`, `asdex` JAX package) | **Best short candidate found**, subject to registry/trademark checks. |

**Conclusion:** There is no safe three-letter `asd*` default left. **`asdd` is the worst choice** because it is exactly the acronym and product category of existing agentic spec-driven tools. A four-letter name is the practical minimum.

## 2. Recommendation

### Preferred: `asdx` for the human entrypoint, `asdx-core` for scripts

```text
asdx                 → TUI (new default)
asdx-core            → CLI (existing commands/flags)
asdx-core tui        → explicit compatibility launcher, if retained
```

`asdx` is short, typeable, and Exa found no notable command collision. It keeps the “ASD” association without claiming the occupied `asd`/`asdd` names.

**Do not** install `asd` or `asdd` as global commands. If the owner personally likes one, use a shell alias in their own profile; never ship it as the product default.

### Naming is not yet implementation

Feature 019 is unapproved and unimplemented. Decide the final shipped names **now**, before B1, rather than shipping `agentic-sdd` as the TUI and renaming again later. A second rename multiplies docs, cask, protocol and migration work.

## 3. Two migration options

### Option A — safest: new names, old names remain compatible

Canonical:

```text
asdx                 TUI
asdx-core            CLI
```

Legacy compatibility for one deprecation window:

```text
agentic-sdd          legacy CLI wrapper → asdx-core
agentic-sdd-tui      legacy TUI wrapper → asdx
```

Behavior:

- Existing `agentic-sdd preview|apply|run|task|review` continues to run the CLI through the wrapper; it is **not** silently changed into a TUI.
- Existing `agentic-sdd-tui` still opens the TUI.
- New users use `asdx` and `asdx-core`.
- Legacy names print a concise deprecation notice to stderr without changing stdout, JSON or exit codes.

This avoids breaking existing scripts while introducing the easy default. It changes the current 019 draft's decision that `agentic-sdd` itself becomes the TUI; that is the trade-off for safety.

### Option B — current 019 direction: `agentic-sdd` becomes the TUI

```text
agentic-sdd          TUI
agentic-sdd-core     CLI
asdx                 optional alias → agentic-sdd
asdx-core            optional alias → agentic-sdd-core
```

This is simpler conceptually but **breaks existing `agentic-sdd <cli-command>` scripts** and any automation that expects bare `agentic-sdd` to preview. It makes the short name an alias rather than the canonical command, which weakens the naming benefit.

**Recommendation: Option A**, unless the owner explicitly accepts the breaking change in Option B.

## 4. Safe migration sequence

### Phase 0 — decide before implementation

1. Choose Option A or B and the deprecation window.
2. Reconcile the names into Feature 019's decisions before combined approval.
3. Verify `asdx` availability beyond Exa: GitHub org/repo, npm, PyPI, Homebrew, common package managers, trademarks and shell completion names.
4. Check the target machine for existing `asdx`/`asdx-core` commands and PATH shadowing. Do not overwrite a binary the installer does not own.

### Phase 1 — additive release

Ship the new canonical binaries and the legacy wrappers together:

- Archive members: `asdx`, `asdx-core`, and (for Option A) wrappers `agentic-sdd`, `agentic-sdd-tui`.
- Cask links all shipped names; quarantine handling covers each executable.
- Wrapper behavior is deterministic: no shell interpolation, no PATH search for the target, no silent fallback, same exit code and streams.
- `agentic-sdd` wrapper dispatches to `asdx-core`, **not** to the TUI under Option A.
- `agentic-sdd-tui` wrapper dispatches to `asdx`.
- `--version`/`--help` on legacy names identify the compatibility name and the canonical target.
- Keep `~/.agentic-sdd/` state, backups, logs and `AGENTIC_SDD_*` environment variables unchanged. Do not force a data-directory migration for a rename.
- Update README, getting-started, installer reference, execution-runs, TUI, guide and MCP/fleet docs with one migration table.
- Do not rewrite historical specs, handovers, release reports or accepted test records.

### Phase 2 — deprecation window

- Publish release notes naming the old and new commands and the removal release.
- Keep wrapper stdout/JSON/exit behavior stable; emit the notice on stderr only.
- Test fresh install, upgrade from the previous release, wrapper execution, PATH conflicts, uninstall and rollback.
- Keep shell completions for both names during the window.

### Phase 3 — removal

- Remove legacy wrappers only after the announced window.
- Keep the state directory and environment variables stable unless a separate, approved migration exists.
- Provide a final migration note and a rollback path: install the previous release if an old script cannot be updated immediately.

## 5. Packaging and protocol details

- **Archive/cask:** one paired archive can contain both canonical names plus compatibility wrappers; verify GoReleaser/cask support for multiple linked names rather than assuming duplicate names work. Local `release:snapshot` is the proof, not a static name check alone.
- **Protocol:** product labels and protocol revision change with the canonical names; `asdx-core` verifies `asdx` as its sibling. Old/new pairs must fail closed with rebuild guidance.
- **No hijack:** under Option A, `agentic-sdd` must never become the TUI silently. That is the primary safety property.
- **No global `asd`/`asdd`:** the installer must not create them, even as convenience aliases.
- **No automatic state rename:** `~/.agentic-sdd` and `AGENTIC_SDD_*` stay for compatibility; new `ASDX_*` aliases may be added later only as an explicit, tested compatibility layer.
- **Uninstall:** remove only names/files owned by this product; never remove a pre-existing command it did not install.

## 6. Verification checklist

| Area | Check |
|---|---|
| Name availability | Registry/GitHub/Homebrew/trademark check for `asdx`, `asdx-core`; warning if a command already exists on PATH |
| Canonical behavior | `asdx` opens TUI; `asdx-core preview` runs CLI; bare `asdx-core` previews |
| Legacy behavior | `agentic-sdd preview` still runs CLI; `agentic-sdd-tui` still opens TUI; stderr deprecation only |
| Packaging | Archive/cask contain the intended names; wrappers are regular executables; checksums match |
| Protocol | `asdx-core tui` verifies sibling `asdx`; mismatched/missing sibling fails closed |
| State | Existing `~/.agentic-sdd` backups/logs remain readable; no forced directory migration |
| Docs | Active docs use canonical names; one migration table; historical files untouched |
| Rollback | Previous release can be reinstalled; wrappers can be removed without touching managed state |
| Platforms | macOS cask/universal binaries; Linux/source where supported; unrun platforms reported `not_run` |

## 7. Decisions for the owner

1. **Name:** accept `asdx` + `asdx-core`, or choose another verified short name.
2. **Migration option:** Option A (safe legacy `agentic-sdd` CLI wrapper) or Option B (breaking `agentic-sdd` TUI).
3. **Deprecation window:** at least one minor release; exact removal release.
4. **Legacy wrappers:** whether `agentic-sdd-tui` remains for one window, and whether `agentic-sdd-core` (never shipped) is needed at all.

Until these are confirmed, treat `asdx` as a proposal only. No binary rename, cask change or implementation is authorized by this study.

## 8. Sources consulted via Exa

- [OneHillAI/ASDD](https://github.com/OneHillAI/asdd) — Agentic Spec-Driven Development CLI using `asdd`.
- [efactory-llc/asdd](https://github.com/ffactory-inc/asdd) — ASDD framework and `/asdd-bootstrap`.
- [ericrisco/rsc-harness `asdd` skill](https://skillsmp.com/creators/ericrisco/rsc-harness/skills-asdd) — agentic SDD dispatcher.
- [benenen/asd](https://github.com/benenen/asd) — `asd` terminal session multiplexer (npm).
- [vdeantoni/asd](https://github.com/vdeantoni/asd) — `asd` terminal diff viewer.
- [asd-engineering/asd-cli](https://github.com/asd-engineering/asd-cli) — `asd`/`via` tunnel CLI.
- [mikecubed/sdd](https://github.com/mikecubed/sdd) and [glapsfun/sddx](https://github.com/glapsfun/sddx) — existing spec-driven `sdd`/`sddx` tools.
- [@specd/cli](https://www.npmjs.com/package/@specd/cli) — SpecD spec-driven platform.
- [ASDL](https://asdl.sourceforge.net/), [ASDK](https://asdk.com/docs/developers), [asdgo](https://pkg.go.dev/github.com/asdgo/asdgo), [asdkit](https://github.com/TakuyaFujimura/dcase-asd-toolkit) — other `asd*` collisions.
- [asdf](https://github.com/asdf-vm/asdf) — widely used version manager; avoid `asdf`.
