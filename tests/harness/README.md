# Harness fixtures

`scenarios.json` freezes model-behavior prompts, source revisions, exact one-match mutations and independent expected outcomes. Apply mutations **only in disposable isolated copies**, after verifying the base revision and match count. Never edit the live repository to construct a fixture. No behavioral scenario has been executed in B1.

The historical feature fixture starts after the restore core and before its CLI integration. The regression fixture disables the identical-tree fast path. The review fixture reverses the already-absent guard for removing a freshly installed skill. Independent sentinels/backup counts and before/after tree identities are the oracles, not the implementation's own return values.

The legacy restore variant is reserved from tuning. It is not genuinely unseen to this author; an evaluator must supply a hidden concrete case before reporting unseen-case coverage. No new agent is authorized to do that automatically.

`contracts.json` separates schema assertions from future semantic/runtime assertions. Run `python3 tests/harness/validate_contracts.py` with the already available Python `jsonschema` 4.x package. It validates both schemas/examples and rejects malformed versions, paths, timeout, argv, costs, direct-fix state and timestamps. **It does not implement T07/T08's runner or semantic checks**, and passing it does not prove agent compliance, policy denial or runtime cancellation.

Behavioral runs require a frozen model/variant, permissions/tools, isolated execution context, budget and actual authorization. The scenario definitions themselves grant none. Reserve two repetitions for critical boundary/recovery cases when authorized, record failed attempts too, and report sample counts rather than reliability claims.
