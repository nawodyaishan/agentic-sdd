# Feature 014 — Plan (r1, 2026-11-11)

**Parent spec:** `specs/014-agent-fleet-model-overrides/spec.md` (rev 2026-11-11, R1–R11) superseding `014-agent-fleet-model-overrides-spec.md` draft.
**Status:** Approved 2026-11-11 (combined spec/plan/tasks; approval recorded once in [spec §13](spec.md#13-approval)). B1–B5 implemented and awaiting human review; B6–B9 authorized in parallel. No real-home/model/release authorization.
**Target:** Root Go CLI only (`go 1.23`, stdlib-only); no `experiments/tui` changes; 009 behavior preserved.
**Provisional review:** DeepSeek-based architecture review 2026-11-11 — verdict *Approve with conditions*; the conditions are tracked in §5 (H1–H3, M1–M10, L1–L9) and §6. Final consequential verdict still requires Opus review; the human combined approval is recorded in spec §13, not inferred from a review.
**Fix reviews 2026-11-11:** verification `Needs changes` + Gemini-Flash `Approve with conditions` (H1/M1/M2) + DeepSeek-high `Request changes` (H1-H3/M1-M10/L1-L9). Required fixes recorded in §5 (B6–B9); `tasks.md` owns batch state.

## 1. Repository-first inspection (must precede implementation)

Confirm in checkout before design freeze:

1. `opencode-pack/agents/` (14 files) + `presets/free-first/` (4 files); frontmatter `model:`/`permissions:` shapes; `sdd-orchestrator.md` has no `model:`.
2. `specs/009-opencode-orchestration/{spec,plan,tasks}.md` v6 + release `v0.6.0` artifacts; Option B 5-overlap list.
3. `specs/013-unified-tui-experience/{spec,tasks}.md` B6/B7 active state — confirm no shared files.
4. `internal/skillsync/{manifest,sync,restore,files,source}.go`, `internal/appops/`, `cmd/agentic-sdd/{main,opencode_disabled}.go`, `opencode-pack.go`, `opencode_pack.go` embed coverage.
5. `AGENTS.md` fallback matrix + routing lanes; `Taskfile.yml`, `Taskfile.experimental.yml`, `tests/supported-core-go.sh`, `harness/checks.json`, `harness/` evidence rules; 008 fence (9-package gate, offline isolated-home tests).
6. Do NOT inspect real `~/.config/opencode` or private fleet repos; use sanitized fixtures.

## 2. Technical design (frozen for tasks)

### 2.1 New package `internal/fleet/` (stdlib only)

| File | Responsibility |
|---|---|
| `fleet.go` | Types: `Config{SchemaVersion,Fleet,Preset,Instances}`, `Instance{Source,Model,Enabled}`, `SourceRef{Kind,Name}`; constants (schema v1, 1 MiB cap, regexes). |
| `parse.go` | Strict-JSON load: `LimitReader(1MiB+1)` → oversize; `utf8.Valid`; two-pass decode (`DisallowUnknownFields`, `schemaVersion==1`) + token-walk duplicate detector per object level; ID regexes; `model` grammar R9; forbid policy keys as unknown-field errors; reject `/ \ ..` traversal. |
| `resolve.go` | Walk `agents/` + `adapters/` + `presets/` → allow-list (adapters share the `agent:` namespace; duplicate names across roots are rejected); validate `SourceRef` one-hop (must be real file, never instance ID); build `Resolved{InstanceID,SourceKind,SourceName,SourceBytes,SourceDigest,Model,Enabled}`; 18-name golden mapping (5 overlaps + 2 adapters + `vision`→`sdd-visual-inspector` as `agent:`, 4 free-first→`preset:`, 6 absent→preset reuse). |
| `render.go` | Start-from-canonical-bytes: split frontmatter/body at second `---` (state machine handling CRLF, missing/duplicate `model:`, `---` in body); replace/set only `model:` scalar preserving `#variant`; body byte-identical; return `desiredDigest=fileDigest(staged bytes)`; normalized permission-set equality assertion (reuse frontmatter parser pattern from `skillsync/opencode_pack_test.go:463`); fail closed on any permission delta. |
| `plan.go` | Plan identity: canonical JSON (sorted keys, explicit field order) over `{sourceRev,configDigest,rendererVersion,targetRoot,fileList,lastKnownInstalled,proposedOps,permNotes}` → SHA-256 `planID`; three-state drift compare (last-applied vs current vs desired); stale-plan revalidation set (source bytes, config bytes, target hashes, renderer version, scope). |
| `manifest.go` | Sidecar `fleet-manifest.json` v4 (separate from `manifest.go` format 1–3): per file `{instanceID,sourceRef,sourceDigest,configDigest,rendererVersion,desiredDigest,installedDigest,backupRef,scope,target,provenance}`; `legacy/unknown` when no desired digest; adoption only via `--adopt`. |
| `doctor.go` | Opt-in local inspector only (no network/model calls): parse local `opencode` config/catalog dump supplied via `--opencode-bin` help-resolved subcommands; labels `configured\|catalog-visible\|authenticated\|unknown\|unavailable`; everything else `not checked (offline)`. |

`go.mod`: no new `require`. Reuse `skillsync` digest helpers + staged file/remove/rollback path via small exported accessors (no fork of backup logic).

### 2.2 Adapters (Option A, B6 decision)

Option A keeps every new source under the existing pack tree as
`opencode-pack/adapters/`, outside the 009 `agents/` install set:

- `sdd-spec-9r.md` and `sdd-plan-9r.md`: `mode: subagent`, deny-by-default,
  allow `read/glob/grep/skill/execute/codegraph_*/context7_*`, deny
  `subagent/shell/edit/exa_*/.env`, non-writing/non-shell. Body reviewed as a
  policy change; each returns a draft to the parent and never approves.
- `vision`: **single mapping** to canonical `agent:sdd-visual-inspector`. Its
  body already is the image/PDF-rendered-page transcription role with no
  edit, shell, web or delegation, so no divergent `vision.md` adapter is
  added. Adding one would create a second prompt for an equivalent contract.
- `resolve.go` walks `agents/` + `adapters/` + `presets/free-first/`.
  `--with-opencode` still installs exactly the 14 files under `agents/`;
  adapters are never installed.

### 2.3 CLI `cmd/agentic-sdd/fleet*.go` → `runFleet`

Frozen grammar:

```text
agentic-sdd fleet status --config PATH [--home PATH] [--repo PATH]
agentic-sdd fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]
agentic-sdd fleet apply --plan PATH [--home PATH]
agentic-sdd fleet restore --backup <fleet-backup-id> [--home PATH]
agentic-sdd fleet doctor [--opencode-bin PATH] [--config PATH]
agentic-sdd help fleet
```

- `case "fleet"` before `default` in `runEnv`; per-subcommand `FlagSet(ContinueOnError)`; `fleet restore` flag form only.
- Absent-key semantics: key absent = ignored (never implicit delete); explicit `enabled:false` = managed remove + backup.
- Lock: `${backupRoot}/.fleet.lock` via `O_CREATE|O_EXCL` + PID + 30s stale timeout; non-blocking fail exit 1 (M1).
- Exits: `2` usage/validation, `1` runtime/stale/conflict/lock, `0` ok (`3` reserved for plan-conflict if needed, documented in help).
- Single fleet-manifest path: `<backupRoot>/<stamp>/fleet-manifest.json` + live pointer `~/.agentic-sdd/fleet/current.json` (copied on apply; M3). Plan file canonical hash; pre-apply revalidates all digests; plan ID is consistency token, not authorization.
- Existing `preview/apply/backups/restore/guide/tui` output byte-identical when fleet unused (parser test).

### 2.4 Provenance, drift, safety

- Keep `manifest.go` format 1–3 untouched; fleet sidecar only.
- Three-state table per spec §FR-10; no speculative merge; unmanaged collision → conflict; symlink/special/non-dir targets refused (extend `files.go`/`source.go` precedent to agent target dir + traversal).
- `enabled:false` → omit + `remove:true` staged-remove reuse (`sync.go` path); never write `AGENTS.md`; never invent agent `disabled:` without version-gated proof.
- Offline by default: `status/plan/apply/restore` zero network/model/process; `doctor` only local inspection.

## 3. Verification strategy (isolated-home, offline)

- Unit + CLI tests in `internal/fleet` + `cmd/agentic-sdd` using `t.TempDir()`; no `~/.config`, no network, `GOPROXY=off`.
- Negative matrix: spec §8 tests 1–20 mapped 1:1 to test names (`TestFleetRejectDuplicateKeys`, … `TestWithOpenCodeUnmodified`).
- Scale: 18-instance golden fixture + 100-instance synthetic fixture; record NFR-02 median-of-5 + worst-case; performance is design target, not claimed.
- Regression: `--with-opencode` + `--opencode-free-first` fixtures per 009 AC-04–AC-10 byte-identical.
- Gates: `task test`, `task vet`, `git diff --check` over 10 packages (`.` + 9 existing + `internal/fleet`); update `Taskfile.yml`, `harness/checks.json`, `tests/supported-core-go.sh` (+ `Taskfile.experimental.yml` if mirrored); `go mod verify`/`tidy` prove zero deps.
- Report: AC-01–AC-29 Pass/Fail/Blocked/Not-run with commit/commands/env; supported OpenCode version(s) only if binary actually exercised, else `Not run`; release = no-go if material behavior unverified.

## 5. Required fixes from verification + dual arch reviews (authorized 2026-11-11)

Owner authorized parallel fix fleet (DeepSeek high + Gemini Flash). No real-home/model/release.

- **B6 — Adapters + all-18 + vision mapping (H1/M10):** Option A: `opencode-pack/adapters/` (`sdd-spec-9r.md`, `sdd-plan-9r.md`, `vision.md` only if `visual-inspector` inequivalence proven, else `vision→agent:sdd-visual-inspector`); `resolve.go` walks `agents/`+`adapters/`(+presets); update `009 install set` exclusion so `--with-opencode` stays 14; golden test resolves+renders all 18 against real pack; fix `plan.md:6` M1–M5 ref, `spec/plan/tasks` DRAFT vs Approved lines, `tasks T05/T06` states, missing handover, trusted-base adoption/defer.
  - **Trusted-base disposition (B6):** the 10-package `harness/checks.json` candidate differs from the last reviewed base and is **not adopted**. Running `task verify-trusted` against a live reviewed base is a separately authorized live operation and is **deferred**; B6 records the deferral and runs no trusted-base comparison.
- **B7 — Apply/restore integrity (M1/M2/M5/L4 + M3):** re-derive three-state at apply (`current==desired` or `adopt&&desired==current` else refuse; drifted remove needs explicit decision; enforce FileList↔Ops + SourceRef/SourceDigest symmetry + resealed-plan test); `restored` provenance so managed-update→restore→replan works without forced `--adopt` (or document + test); PID-authoritative/atomic lock reclaim + ownership-checked Release + lock inside APIs or handle + concurrent test; backup-digest verify post-copy + live-pointer rebuild path; legacy `loadBackup` ignores `fleet-manifest.json` or segregated fleet root + non-`sdd-*` fixture (or narrow AC-15 + document).
- **B8 — Plan/CLI hardening (M6/M7/M8/M4 + M9/L1-L3/L5-L8):** drop `ConfigPath/RepoPath` from planID (require `--config` at apply or 0600 + sensitive docs; fix identity comment/test; sanitize error paths); `LimitReader` + non-regular reject at `LoadConfigFile/LoadPlan` + tests; exits `ErrInvalidConfig`/flag/plan-structure→2, stale/conflict/lock→1, help legend + table test; status hashes targets (drift/unmanaged/unsafe), resolves sources (use `--repo`), offline/selected-primary labels, `PermNotes` with perm digest + route-change warn (store prior model), unknown-source exit 2; reserved `spec/plan/vision` allowlist or explicit warning + test; fix CRLF insert, empty-model/null-enabled semantics, `Revalidate` require-all, symlink state dirs, block-scalar handling, 0700 + absolute home, doctor stdout cap + `authenticated` decision.
- **B9 — Scale evidence + FR-15 + docs (H2/H3/M10/L9):** real `BuildPlan` 18 + 100 temp-home scale (median-of-5, worst, alloc, runner) for NFR-02/AC-25; implement FR-15 read-only import preview or record human-approved deferral in spec §9/§13 + tasks; real-pack render all 14, §8 rows 3/5/12/18/19, sidecar `applying/failed`, managed restore→replan, fleet-backup listing tests; guide placement decision, consolidated AC-01–AC-29 report, handover `docs/handovers/2026-11-11_014*.md`.

## 6. Batches (authorization still required; parallel fleet below)

- **B1 — Schema/resolver + strict validator** (`fleet.go`, `parse.go`, `resolve.go` + golden 18-name test). No CLI, no writes.
- **B2 — Renderer + permission gate** (`render.go` + ≤3 adapters conditional + frontmatter surgery + normalized perm-equality tests).
- **B3 — CLI status/plan + identity/lock** (`fleet*.go` dispatch, `plan.go`, lock, help/exits; read-only + plan-output tests).
- **B4 — Apply/restore + sidecar/drift/adopt** (`manifest.go`, staged apply/remove reuse, three-state drift, `--adopt`, idempotent no-op, rollback fixtures).
- **B5 — Doctor/diagnostics + docs + full regression/scale** (`doctor.go`, worked guide, gate updates, 20-case matrix, scale timings, completion report).

B1+B2 are mutually independent (share only frozen `SourceRef`/grammar); B3 depends on B1/B2 contracts but can start against stubs in parallel; B4 depends on B3 plan schema; B5 is final integration. `tasks.md` assigns each batch to a distinct subagent for parallel execution with CodeGraph+Context7 read access.
