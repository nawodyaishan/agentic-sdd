#!/bin/sh
# Build the pinned feature-005 QA image from local binaries and the local base.
# No image or package is pulled: the build runs with --network=none.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
image=${QA_IMAGE:-agentic-sdd-qa:005}
base=codercom/enterprise-base@sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769
agentic=${AGENTIC_SDD_BINARY:-$root/bin/agentic-sdd}
opencode=${OPENCODE_BINARY:-$(command -v opencode 2>/dev/null || true)}

die() { printf 'error: %s\n' "$1" >&2; exit 2; }

[ -x "$agentic" ] || die "build the CLI first: make build"
[ -n "$opencode" ] && [ -x "$opencode" ] || die "opencode binary not found; set OPENCODE_BINARY"
docker image inspect "$base" >/dev/null 2>&1 || die "base image is not present locally"

tmp=$(mktemp -d "${TMPDIR:-/tmp}/sdd-qa-image.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

cp "$agentic" "$tmp/agentic-sdd"
cp "$opencode" "$tmp/opencode"
cp "$root/tests/qa-container/Dockerfile" "$tmp/Dockerfile"
cp "$root/tests/qa-container/entrypoint.sh" "$tmp/entrypoint.sh"
chmod 0755 "$tmp/entrypoint.sh"

{
  printf 'image: %s\n' "$image"
  printf 'base: %s\n' "$base"
  printf 'opencode_version: %s\n' "$("$opencode" --version 2>/dev/null || echo unknown)"
  printf 'opencode_sha256: %s\n' "$(sha256sum "$opencode" | awk '{print $1}')"
  printf 'agentic_sdd_sha256: %s\n' "$(sha256sum "$agentic" | awk '{print $1}')"
  printf 'source_commit: %s\n' "$(git -C "$root" rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'skills: 18\n'
} >"$tmp/identity.txt"

docker build --network=none --pull=false -t "$image" "$tmp"
docker image inspect "$image" --format 'image_id={{.Id}}'
