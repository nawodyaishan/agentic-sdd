#!/bin/sh
# Feature 022 A8: detect drift between the root packages and the vendored copy
# in experiments/tui/vendor. Read-only: it re-vendors into a temporary
# directory outside the tree and compares, then resolves dependencies from vendor. Never downloads (offline settings).
#
#   tests/tui-vendor-check.sh           check; exit 1 on drift, 2 when blocked
#   tests/tui-vendor-check.sh --stamp   rewrite experiments/tui/vendor.stamp
#
# After changing internal/skillsync or internal/fleet: re-run
#   (cd experiments/tui && go mod vendor) and then --stamp.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd -P)
stamp_file="$root/experiments/tui/vendor.stamp"

# Digest of the non-test Go sources that the vendored copy contains.
source_digest() {
  (cd "$root" && find internal/skillsync internal/fleet -type f -name '*.go' ! -name '*_test.go' | LC_ALL=C sort |
    while read -r f; do sha256sum "$f"; done | sha256sum | cut -d' ' -f1)
}

if [ "${1:-}" = "--stamp" ]; then
  source_digest > "$stamp_file"
  echo "wrote $stamp_file"
  exit 0
fi

fail() { echo "tui-vendor-check: $1" >&2; exit 1; }
blocked() { echo "tui-vendor-check: blocked: $1" >&2; exit 2; }

[ -f "$stamp_file" ] || fail "missing $stamp_file; run with --stamp after reviewing the vendored copy"
[ "$(cat "$stamp_file")" = "$(source_digest)" ] ||
  fail "internal/skillsync or internal/fleet changed since the vendored copy was stamped; re-vendor and re-stamp"

tmp=$(mktemp -d "${TMPDIR:-/tmp}/tui-vendor-check.XXXXXX") || blocked "cannot create temporary directory"
trap 'rm -rf -- "$tmp"' EXIT INT TERM
case "$tmp" in "$root"/*) blocked "temporary directory is inside the source tree";; esac

env_off="GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS="
(cd "$root/experiments/tui" && env $env_off go mod vendor -o "$tmp/vendor") >"$tmp/vendor.log" 2>&1 ||
  { cat "$tmp/vendor.log" >&2; blocked "go mod vendor failed offline; populate the module cache first"; }

# Root-level docs and config in the vendored root copy are not inputs to the
# companion build, so they do not count as drift. Everything else is compared.
listing() {
  (cd "$1" && find . -type f ! -regex './agentic-sdd/[^/]*\.\(md\|yml\)' ! -path './agentic-sdd/LICENSE' |
    LC_ALL=C sort | while read -r f; do sha256sum "$f"; done)
}
listing "$tmp/vendor" >"$tmp/fresh.list"
listing "$root/experiments/tui/vendor" >"$tmp/current.list"
if ! diff "$tmp/fresh.list" "$tmp/current.list" >"$tmp/diff.log" 2>&1; then
  cat "$tmp/diff.log" >&2
  fail "vendored copy differs from a fresh offline vendor"
fi

# go mod verify cannot check a path-replaced module (no zip hash), so prove the
# vendored tree resolves the companion's dependencies instead.
(cd "$root/experiments/tui" && env $env_off go list -mod=vendor -deps ./cmd/tui-companion) >"$tmp/list.log" 2>&1 ||
  { cat "$tmp/list.log" >&2; fail "vendored tree does not resolve the companion's dependencies"; }

echo "tui-vendor-check: ok"
