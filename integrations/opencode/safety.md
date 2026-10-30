# Evaluation safety and model usage

## Incident correction and scope

The user supplied `INCIDENT-2026-10-28-openai-spend.md` (2026-10-28). It
attributes approximately **$0.825 of OpenRouter charges** to the previous
evaluation and reports copied live credential stores in temporary directories.
The previous approximately $0.706 event estimate was incomplete. Separate
subscription consumption also matters even when reported cost is zero.
The report remains outside this repository; do not copy its forensic details,
account records, credentials or raw traces into Git.

This adapter does not repair the account, revoke keys or secure old copies.
The current change corrects the evidence summary, adds repository guidance and
ships inert deny-by-default configuration plus credential-free native tests.
Those are best-effort prevention, not a universal billing firewall.

## Controls delivered now

- `opencode.jsonc` is an **inactive template**. All providers are denied. Its
  explicit free default cannot silently inherit a paid session/command model.
  `policy-lockdown.jsonc` is a separate higher-authority example, also inactive.
- `model-usage.json` is a usage catalog, **not runtime configuration or consent**.
  Every entry is disabled and capability-unverified; automatic fallback is empty.
  Exact model allowlisting is not equivalent to provider allowlisting:
  `provider.use` controls a provider, not individual models behind OpenRouter.
- No evaluation launcher, model subprocess, SDK inference or HTTP generation
  route is shipped. The native probe's positive endpoint allowlist contains
  only config/catalog observation, local session creation, permission evaluation
  and location reload. It never submits a prompt, command or compaction request.
- The native wrapper uses a locally available pinned image, `--pull=never`,
  `--network=none`, non-root UID, read-only mounts/root, dropped capabilities,
  no-new-privileges, bounded memory/PIDs and ephemeral tmpfs. It mounts only the
  binary, templates and probe, **not a home, credential database or socket**.
- Native fixtures use a fresh database. The fixed local HTTP password in the
  probe protects only its loopback test server; it is not a provider credential.
- Extra model execution needs the independent controls below. Setting
  `EVAL_ALLOW_PAID=1`, choosing a primary agent or passing a CLI `--model` is
  not sufficient authorization or a cap. Repository prose cannot stop an
  arbitrary host process or a user from invoking the CLI outside these tests.

## Selected models

These IDs came from the user's supplied selection, not a successful B3 request.

| Use | Exact OpenCode model reference | Initial status |
| --- | --- | --- |
| Everyday coding | `opencode/deepseek-v4.1-flash-free` | Explicit template default; providers still denied |
| Quick actions | `opencode/glm-5.3-flash-free` | One proposed manual alternative, not automatic fallback |
| Harder work | `openrouter/xiaomi/mimo-v2.6-pro` | Paid promotion candidate, not an enabled initial profile |
| Experimental reasoning | `openrouter/nvidia/nemotron-3-ultra-550b-a55b:free` | Synthetic-only, separate experiment |

Economy/Standard/Deep initially all name the same DeepSeek default. This keeps
the approved one-default/at-most-one-alternative scope rather than silently
enabling a four-model experiment. GLM can be the single explicitly selected
alternative once verified. MiMo and Nemotron promotion/testing are separate
runs, not automatic "upgrades". Subagents remain denied regardless of the
catalog's task categories. The earlier MiniMax proposal is superseded by this
selection. Other models from the supplied free catalog are not enabled here.

The user subsequently permitted testing these named low-cost/free models.
**No such inference was performed in B3**: current per-endpoint price,
capability, private-data eligibility and dedicated paid-key cap are not
established. Permission to test does not establish these prerequisites, and no
new run allocation or monthly remaining balance is inferred.

## Before any separately authorized online test

1. Record exact provider/model/variant, synthetic input identity, authorization,
   active session count (one), children (zero), maximum total requests including
   auxiliary calls/retries, input/output tokens, deadline and stop condition.
   Default to no retries. Hold the model, tools and inputs fixed for baseline
   versus candidate; do not introduce a judging model without its own scope.
2. Verify the exact free endpoint currently advertises zero input/output cost
   and tools, then prove tool compatibility with a bounded synthetic probe.
   Catalog presence alone is not capability evidence. Free can still consume
   rate limits/subscription allowance. Never strip `:free`, use an auto-router,
   probe other providers or fall back on failure.
3. For paid use, the user supplies a **dedicated evaluation key**, scoped to
   permitted models/endpoints where available, with a verified provider-side
   cap no higher than the explicit remaining run/month allocation. Never export
   or copy the primary database or OAuth store. No key values in config, logs,
   arguments, Git or copied session databases. If the necessary control cannot
   be verified, remain Blocked; do not substitute a soft estimated budget.
4. Verify current endpoint prices and remaining account/month allowance. Count
   cached/uncached input, output/reasoning, auxiliary requests, retries, failed
   requests and in-flight work. Record run/session IDs across every store, not
   just the primary database. Observe cumulative usage after each bounded unit;
   unknown/delayed usage stops new work. Reconcile against provider usage before
   another allocation. An event-stream counter is not an invoice or hard cap.
5. Keep the test program's execution environment separate from any provider
   credential service. Use only a user-provisioned scoped secret reference and
   explicit destination/egress control. Do not relax `native-opencode.sh` to
   add network, credentials or host mounts; an online fixture needs separate
   reviewed isolation. Keep owner-only storage and an explicit cleanup owner.
6. Enable only the exact reviewed change after explicit adoption authorization.
   Verify actual command/session selection before private or consequential work.
   A blocked free route stays blocked. No warming/background inference.

## Privacy is separate from price

Free inference, no training and zero data retention are distinct assertions.
The user reports private-use eligibility for the two OpenCode free models, but
the underlying policy/endpoint evidence has not been supplied or independently
verified here. Until that is resolved, this catalog permits **synthetic input
only**, not private repository code. Nemotron remains synthetic-only even if
zero cost is verified. Do not infer coding intelligence from context length.

For OpenRouter private inputs, independently verify the actual endpoint,
no-training and ZDR eligibility. Official routing documentation supports this
API request-body fragment (it is **not** an OpenCode config field):

```json
{"provider": {"data_collection": "deny", "zdr": true, "allow_fallbacks": false}}
```

Prefer user-controlled account/key guardrails as well. Verify restrictions on
the actual runtime request path; current V2 docs say agent `request` overlays
are preserved but not sent, so putting this fragment on an agent would not
establish protection. ZDR applies to inference routing, not arbitrary plugins,
search tools or their backends. No eligible endpoint means Blocked, not relaxed
privacy. Official ZDR documentation also discloses in-flight safety screening
and possible legal retention; do not promise unconditional non-retention.

Sources consulted through Context7 and official pages on 2026-10-28:
[provider policies](https://openrouter.ai/docs/guides/privacy/provider-logging),
[ZDR](https://openrouter.ai/docs/guides/features/zdr),
[OpenCode agents](https://opencode.ai/v2/docs/agents),
[policies](https://opencode.ai/v2/docs/policies).

## User-controlled remediation and reported status

On 2026-10-29 the user reported exposed OpenRouter-key rotation and the dedicated
evaluation credential with the proposed $0.25 lifetime cap **both completed**;
see [spec §Approval](../../specs/004-vnext-harness/spec.md#approval). No secret
value was requested or read. Scope/cap enforcement, prices/privacy and remaining
allowance have not been independently verified. This report authorizes no model
request or live config change. Do not repeat the dashboard actions implicitly.

- Verify nonsecret key/account scope and caps before any separately authorized
  online allocation; never infer a billing guarantee from the report alone.
- Decide whether to keep or revoke the subscription-sharing credential.
- Review and remove the old credential-bearing sandboxes identified by the
  incident, including any backups/copies. Their directory entries still existed
  at B3 preflight; contents were not opened or copied. No claim of remediation.
- Review credential storage and shell-history hygiene independently.

Do not run broad recursive deletion or logout/account changes as part of an
adapter batch. Agree on exact paths/credentials and authorization first.
