# Meaningful testing

Use when implementation selects checks or review evaluates their adequacy. The question is which real defect a check would detect, not whether a workflow box is filled.

- Derive expected results independently of the implementation under test. Assert observable behavior, a boundary contract or relevant side effects. Computing the expectation through the same function can make a test tautological; mentally consider a plausible incorrect implementation and whether the assertion would detect it. This is not evidence that mutation testing ran.
- For new behavior and bug fixes, prefer a focused test first where it demonstrates the intended outcome. Before changing poorly understood legacy behavior, characterize what it actually does. Use integration, schema, policy or rendering checks where those prove the boundary; prose/configuration need not follow a universal red–green ritual.
- A regression's failing check should reach the target behavior and fail for the intended reason. An import error, absent fixture, unavailable service or missing executable is a setup/baseline problem, not red–green evidence. Diagnose it before claiming a fix. Preserve the actual command/result and explain omissions.
- Doubles are appropriate when they preserve the behavior the assertion relies on. Interaction/call-order assertions are meaningful when that interaction is itself a contract, not merely an implementation detail. Include connected behavior where local doubles cannot prove producer/consumer agreement.
- After correction, verify the original failure and affected dependencies/failure cases. Reuse unaffected evidence only while relevant inputs, code, configuration and environment match. Formatting or a changed diff refreshes affected checks; do not rerun unrelated suites just for ceremony.
- Do not delete good code merely because it preceded a test, weaken a check to hide a defect, or add low-value assertions only to satisfy the process. Packaging/link checks prove distribution; Markdown wording assertions do not prove that an agent follows a skill.

Route unresolved failures to [agentic-sdd-debugging](../../agentic-sdd-debugging/SKILL.md). Respect the invoking mode: read-only diagnosis/review consumes existing evidence and recommends checks or instrumentation instead of executing write-producing commands or adding logging.
