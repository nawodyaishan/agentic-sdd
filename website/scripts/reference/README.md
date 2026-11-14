# Reference fact generation

`website/scripts/reference.mjs` regenerates the collector-owned blocks in
`website/src/content/docs/reference/*.md` from the source checkout and one
candidate `asdx-core` binary. Prose outside the generated markers is never
rewritten.

## Reproducible candidate binary

Parity is only meaningful when the binary was built from the revision being
inspected. Run this from the repository root (the commit is injected, so the
generated block records the exact provenance the collector verifies):

```sh
commit=$(git rev-parse HEAD)
go build -trimpath \
  -ldflags "-X agentic-sdd/internal/version.Version=dev \
    -X agentic-sdd/internal/version.Commit=$commit \
    -X agentic-sdd/internal/version.Date=$(git show -s --format=%cI "$commit") \
    -X agentic-sdd/internal/version.GoVersion=$(go env GOVERSION)" \
  -o /tmp/opencode/asdx-core-reference/asdx-core ./cmd/agentic-sdd
```

Any absolute output path is fine. `-trimpath` keeps host paths out of the
binary, and the `Commit` value is what `--commit` / `ASDX_CORE_COMMIT`
verifies. The binary is a local build artifact: never commit it, never replace
it with a downloaded release binary for parity evidence, and never fake it.

## Generate and check

```sh
# Generate the marked blocks (writes website/src/content/docs/reference/*.md)
ASDX_CORE_BINARY=/tmp/opencode/asdx-core-reference/asdx-core \
  npm --prefix website run docs:reference

# Verify without writing (exit 1 on drift)
ASDX_CORE_BINARY=/tmp/opencode/asdx-core-reference/asdx-core \
  npm --prefix website run docs:reference:check

# Same, with an explicit path and provenance check
npm --prefix website run docs:reference -- --binary /abs/asdx-core --commit "$commit"

# Generate and then check through the task wrapper
task docs:reference -- --binary /abs/asdx-core --commit "$commit"
```

The binary may be supplied either as an explicit `--binary ABSOLUTE_PATH` or
through `ASDX_CORE_BINARY`; `--binary` wins when both are present. The recipe
above is also printed by a blocked run, by
`npm --prefix website run docs:reference -- --recipe`, and by
`task docs:reference:build`, so it stays discoverable without a binary.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | `--generate` wrote the exact set, or `--check` found no drift |
| `1` | Drift, malformed markers, unsafe path or an unusable explicit binary |
| `2` | `blocked`: no `--binary` and no `ASDX_CORE_BINARY`, or an empty value |

A blocked run writes nothing and prints the build recipe above. Without a real
binary built from the inspected revision, reference parity is **blocked**, not
passed. The `website/tests/reference-cli.test.mjs` parity case follows the same
rule: with `ASDX_CORE_BINARY` unset it reports `blocked` together with this
recipe and is never counted as a pass.
