# Execution example fixtures

Synthetic, inert shape examples for the execution run pages (Spec 020 B9-W1-B, T21, FR-16.3). They back the how-to and reference pages with checked JSON and text, and they are not captured evidence.

- **Shape only.** Each `*.json` and `*.md` file is hand-written to match the field names and kinds in the execution service. No command produced them, and no run exists for them. Digests are zero placeholders, and paths start with `/example/`.
- **Inert fake check.** `fake-check.mjs` prints two fixed lines and exits 1. It reads no environment variable, opens no socket and writes no file. It has no shebang and no executable bit; the test runs it only with `node`, from a disposable copy of `repo/`.
- **Disposable repo.** `repo/` holds a tiny synthetic `specs/016-example/` tree. The test copies it into an owned temp directory and removes that directory afterwards.
- **Expected output.** `expected-check.json` records the exit code, stdout and stderr the fake check must produce.
- **No real home, model, network or MCP.** The test sets `HOME` to an owned empty temp directory, clears `PATH`, and never resolves a command through the environment.

The offline test is `website/tests/examples-execution.test.mjs`. Run only the focused suite:

    node --test website/tests/examples-execution.test.mjs
