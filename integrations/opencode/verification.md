# B3 verification and support limits

Recorded 2026-10-28 against candidate OpenCode **2.2.0**, Linux amd64.
These are observations, not approval, support certification or billing claims.

## Reproduce without provider access

```sh
python3 -B -m unittest discover -s tests/harness -p test_opencode.py
sh tests/harness/native-opencode.sh
```

The first command uses Python standard library only (7 tests). The second uses
an existing Docker image and the existing binary, makes no model calls and
does not read the host credential store (9 tests). The wrapper uses the local
image content ID
`sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39`
(`python:3.13-slim` at development time); absent image/binary/Docker is a
blocked prerequisite, not permission to pull/install anything. This fixture is
Linux amd64 only. `OPENCODE_TEST_BINARY` can identify another reviewed 2.2.0
binary; the test refuses another version. Do not redirect it to an untrusted
binary or relax mounts/network to make a failure disappear.

The test server uses a fresh ephemeral database, loopback Basic authentication
and only the positive API endpoint allowlist in `native_opencode.py`. The
container is removed on exit; no credential-bearing logs/database are retained.
Only synthetic fixtures are written. The API permission checks do not run the
corresponding editor/shell/browser tool and never send an LLM prompt.

## Observed

| Check | Result / evidence boundary |
| --- | --- |
| Template invariants and usage catalog | PASS: explicit free model references, current-session command flags, no command shell blocks, deny-first profiles, providers disabled, empty fallback list |
| Manual adoption exercise | PASS: synthetic unrelated config preserved; backup/no-op, original/new-file rollback and intervening-edit preconditions exercised; symlinks/FIFO identified with `lstat` |
| Primary agents | PASS: native registry contains draft/build/review as primary agents |
| Draft/build permission evaluation | PASS: selected feature edit/source edit allowed respectively; production/other-feature/mixed patch resources denied for draft; build shell asks, config edit denied |
| Secrets/external/children | PASS: environment/database resources, external directory and child action denied for each profile |
| Review permission evaluation | PASS: read allowed; edit, shell, Code Mode, glob/grep, web, MCP tool/resources and browser-MCP action denied |
| Global versus project policy | PASS: a project and agent `execute` allow cannot lift the global `execute` deny |
| Provider catalog | Empty under provider-denied config. No inference or credentialed-provider positive control; this is not a billing/capability test |
| Config/command collision | PASS: `.opencode` command description wins over same-name direct definition |
| Skill collision | PASS: one synthetic same-ID skill exports project marker, not global marker, via complete local HTTP JSON |
| Invalid policy | PASS: invalid action produces a policy warning; other valid rules remain usable. Invalid statement does not count as a restriction |
| MCP startup | PASS: global integration denial hides synthetic connector and no startup sentinel appears despite project allow; explicit global allow in scratch starts the positive control |
| External plugin startup | PASS: discovered synthetic JS cannot execute under denial; explicit path allow in scratch executes its positive-control sentinel |

Both positive controls run **only inside the same credential-free, network-none
container**. They write scratch sentinels and do not confer real integration
authorization. Policy rules are not a sandbox for arbitrary plugin code.

## Discovery failures retained / fixture corrections

Initial cold one-shot `api --standalone` probes returned empty command/plugin
registries and deny results before extensions finished loading. Removing plugin
policy or command model fields did not resolve that; no such weakened setting
is retained. The fixture now uses a persistent isolated server, explicit config
reload and one second of extension initialization before assertions. Agent
read/edit positive checks then pass as well as denials. The command-list API
exports only `name`/`description`, not the template/model/agent/subagent fields;
earlier assertions of those fields were fixture errors, not evidence that the
runtime ignored the config. No inference retry or provider switch was used.

The container logs report unsupported recursive directory watching. Explicit
reload is used in fixtures; automatic hot-reload behavior in this container is
not certified. The initialization delay is a test accommodation, not a general
readiness API or recovery guarantee.

## Open acceptance findings

- **B3-F1 — Full tool boundaries blocked.** Permission evaluation is not a
  real model-originated edit/patch/shell/browser/MCP/Code Mode invocation, nor
  canonical symlink-path isolation. All unproven review routes remain disabled.
  Strict review requires a sanitized read-only export; do not adopt a broader
  review mode on the strength of these permission probes.
- **B3-F2 — Selection/tool compatibility blocked.** No model/command execution,
  incompatible-tool probe, unavailable-model fallback or per-session observed
  selection was run. The catalog records the user's proposed exact IDs, not
  verified access. Model tests were subsequently permitted in principle, but
  price/privacy/key-cap/allowance prerequisites remain unestablished. No B3
  model requests were made. MiMo and experimental Nemotron remain dormant.
- **B3-F3 — Privacy/spend prerequisites blocked.** Zero price, no training and
  ZDR are separate. No current endpoint prices, private eligibility or dedicated
  key cap were verified. The earlier incident leaves remaining allowance
  unknown. No cost/safety claim may be inferred from a free suffix or local
  event accounting; see [safety.md](safety.md).
- **B3-F4 — Real adoption/support not authorized or performed.** No active
  config/global policy/service/account changed. Backup/rollback were synthetic
  exercises, not a race-resistant installer or actual user rollback. macOS and
  complete effective live skill precedence remain untested. The synthetic skill
  case does not resolve the historical live CLI catalog truncation.

Batch outcome is **incomplete**, technical self-review **Blocked**, workflow
**awaiting human review**. T05/T06 are partially delivered; blocked acceptance
has not been waived and B4 is not authorized by test success.
