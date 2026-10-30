# Offline receipt validation (B6 subset)

The [v1 schema](receipt.schema.json) remains the single portable receipt shape.
The [example](example-direct-fix.json) is synthetic syntax, not reusable evidence.
`tests/harness/receipt_tools.py` is a **development reference consumer**, using
the existing optional Python/jsonschema tooling. It is not a new production
runtime, native-log exporter, agent runner, automatic ledger or CI trust gate.
The Go runner still emits `result.json`, not a completed observation receipt.

## Independent inputs and usage

The caller manually curates the schema's allowlisted fields from permitted
observations, referencing the actual request and canonical approval/tasks. Never
feed it an auth store, complete config/environment, native database or trace.
Do not write a portable receipt until its free text is safe to export.

From a development script with `tests/harness` on its import path:

```python
from receipt_tools import snapshot, validate_receipt, export_receipt

# Explicitly select all relevant code, inputs, dependency/config and task paths.
# Do not select a real home/auth store. These are illustrative fixture paths.
code = snapshot(workspace, ["src", "go.mod", "specs/fixture/tasks.md"])
current_identity = {**independently_observed_identity, **code}

validate_receipt(
    curated_receipt, workspace=workspace, identity=current_identity,
    authorization_ref=actual_current_authorization_ref,
    contract=trusted_contract_bytes, result=independently_observed_result,
    evidence_root=private_external_root,
    log_digests=previously_observed_log_digests,
    # Omit these when unavailable: session/model/cost default to unknown/null.
    observations=independently_observed_native_values,
    paid_authorized=False,
)
export_receipt(curated_receipt, new_external_file, workspace=workspace,
               redactions=known_literal_redactions)
```

Variables above are supplied by the caller; they are not values inferred from
the receipt. `observations` uses `/session_id`, `/cost_usd`, `/model/provider`,
`/model/id` and `/model/variant` keys. Missing observations require null values
and nonempty JSON-Pointer explanations. Missing billing is not zero. A reported
key cap does not create paid-run authority; all online prerequisites still apply.

Capture log SHA-256 digests from the actual trusted execution observation and
preserve them separately with that result. **Do not recompute expected digests
from possibly changed logs when validating old evidence.** A matching receipt,
result and digest manifest can all be fabricated by an untrusted candidate;
trusted CI provenance is T10 work, not supplied by this function.

Capture the selected code/config/input identity at execution and check it again
after the check. Do not attach an old result to a newly measured identity merely
because the current tree looks correct. If formatting/tidy changed relevant
files, preserve that result but obtain fresh authorized checks for the new state.
The reference validator consumes these observations; it does not automate the
capture/run/export lifecycle or establish that the observer is trustworthy.

`validate_receipt` checks schema, null reasons, selected trusted IDs/coverage,
unique findings, check times/status/exit/evidence, independently supplied
observations/authorization reference, and current identity. It never supplies
consent or evaluates a reviewer's engineering reasoning. `Ready` claims require
complete required coverage and no unresolved valid/needs-evidence findings.
Schema v1 has no machine-readable finding-resolution field: preserve resolved
findings in canonical tasks; conservatively omit them from a fresh receipt's
open-findings list only with an explicit canonical resolution reference.
Direct fixes cannot carry canonical task/workflow state or invented feature IDs.

## Identity and freshness

`snapshot(workspace, paths)` requires a Git-root workspace and unique literal
repository-relative file/directory scopes (`.` selects the repository). No
glob/pathspec magic, traversal or `.git` scope is allowed. It returns HEAD and
`sha256-v1:<digest>` for sorted canonical JSON records containing:

- scope paths;
- scoped HEAD and staged-index entries (path, Git mode/object ID/stage);
- working paths, kind/absence, permission bits and SHA-256 of file bytes;
- relevant untracked files, including deleted explicitly selected paths.

Git ignores apply without reading user-global config; an explicitly named
ignored **file** is included. Tracked files remain included. Git omits untracked
FIFOs/devices, so metadata inspection also refuses nonignored special files.
Symlinks, submodules and unmerged indexes are unsupported and fail closed.
Git observations use a minimal environment, no global/system config, no inherited
external index or credential variables, and no optional index refresh locks.
No content or complete manifest is exported by `snapshot`, only its digest.

The bounded reference handles at most 10,000 inspected files, 32 MiB per file,
128 MiB working bytes and 10 MiB Git output; Git operations have 15-second
timeouts. It requires a quiescent trusted filesystem: **not** a concurrent-writer
sandbox or an atomic cross-process Git snapshot. Hashes describe only selected
files. Omitted dependencies cannot be detected automatically; unknown or changed
scope/config/environment/runtime/skills/harness/contract prevents reuse. Remeasure
after formatting, merge, interrupted writes or a location move. A moved workspace
blocks old receipt reuse even if its scoped content digest is unchanged.

## Export boundary and retention

Exports require an existing private parent outside the target checkout, no
symlink components or traversal, and a new filename. Output is exclusive, 0600
and flushed; no existing evidence is overwritten. Evidence validation requires
private regular logs inside the authorized external root, with stored digests.
These checks assume no concurrent untrusted filesystem writer.

Known literal values and common bearer/key-value credential forms are redacted
in memory **before any receipt bytes are written**. Unknown fields are rejected;
errors do not echo rejected values. Synthetic canary tests cover both export
content and rejection paths. This is **not universal secret detection**: encoded,
split or unidentified secrets and sensitive prose require human curation. Raw
check logs are not sanitized by this receipt helper; keep them private and never
copy them into Git or a portable receipt. Keep default raw retention 7 days and
sanitized receipts 30 days with manual cleanup; explicitly retained acceptance
evidence needs an owner/expiry. No native database/snapshot pruning is performed.

See [fixture commands and evidence scope](../../tests/harness/README.md#b6-offline-receipts-and-recovery).
