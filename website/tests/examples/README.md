# Example-runner fixture project

Fixtures for `website/scripts/examples.mjs` (Spec 020 B2-W1-D, C3).

- **No committed executables.** Each test calls `writeStubCli` from
  `stub-cli.mjs` to generate a throwaway Node stub (mode `0o755`, shebang
  `#!<absolute node>`) inside an owned temp directory. The real candidate
  `asdx-core` binary is never required, and the runner never resolves anything
  through `PATH`.
- **Stub identity.** The generated stub identifies itself as
  `asdx-core 9.9.9-fixture` / `fixture-commit-0000000`. It records argv, cwd,
  `HOME`, `PATH` and the environment key set into a log file whose absolute
  path is baked into the script, so the record survives cleanup of the runner's
  temp root.
- **Variants.** `mutateHome` simulates a preview that wrongly writes under the
  fixture home; `versionExit` / `previewExit` simulate failing steps;
  `silentPreview` simulates an unreported result.
- **Isolation.** The runner creates its own `asdx-examples-*` root (fixture
  `home/`, plus cache/config/data/state/tmp), removes it by default, and only
  retains it with `--keep`. Tests must never point `--home` at a real user
  home.

Run the focused suite only:

```sh
node --test website/tests/examples-*.test.mjs
```
