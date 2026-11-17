#!/bin/sh
set -eu

# Verify deterministic two-build binary identity from a clean git archive HEAD
# export outside the repository. Never builds from a dirty working tree, never
# writes into the source tree, and never runs GoReleaser publish commands.

root=$(cd "$(dirname "$0")/.." && pwd -P)
ref=${1:-HEAD}

scratch_base=${SDD_REPRO_SCRATCH:-}
if [ -z "$scratch_base" ]; then
  if [ -d /dev/shm ] && [ -w /dev/shm ]; then
    scratch_base=/dev/shm
  else
    scratch_base=${TMPDIR:-/tmp}
  fi
fi

work=$(mktemp -d "$scratch_base/sdd-repro-check.XXXXXX")
trap 'rm -rf -- "$work"' EXIT INT TERM

mkdir -p "$work/src1" "$work/src2" "$work/out1" "$work/out2" "$work/cache1" "$work/cache2"
git -C "$root" archive "$ref" | tar -x -C "$work/src1"
git -C "$root" archive "$ref" | tar -x -C "$work/src2"

full_commit=$(git -C "$root" rev-parse "$ref")
short_commit=$(git -C "$root" rev-parse --short "$ref")
commit_date=$(TZ=UTC git -C "$root" show -s --format=%cd --date=format-local:%Y-%m-%dT%H:%M:%SZ "$ref")
version=$(git -C "$root" describe --tags --exact-match "$ref" 2>/dev/null || printf "dev+%s" "$short_commit")
goversion=$(GOTOOLCHAIN=local GOENV=off go env GOVERSION)
gomodcache=$(GOTOOLCHAIN=local GOENV=off go env GOMODCACHE)

cli_ldflags="-s -w -X agentic-sdd/internal/version.Version=$version -X agentic-sdd/internal/version.Commit=$short_commit -X agentic-sdd/internal/version.Date=$commit_date -X agentic-sdd/internal/version.GoVersion=$goversion -X agentic-sdd/internal/buildinfo.Revision=$version -X agentic-sdd/internal/buildinfo.Commit=$full_commit -X agentic-sdd/internal/buildinfo.SourceDigest=git:$full_commit -X agentic-sdd/internal/buildinfo.DistributionMarker=generic"
tui_ldflags="-s -w -X agentic-sdd/internal/buildinfo.Revision=$version -X agentic-sdd/internal/buildinfo.Commit=$full_commit -X agentic-sdd/internal/buildinfo.SourceDigest=git:$full_commit -X agentic-sdd/internal/buildinfo.DistributionMarker=generic"

if command -v sha256sum >/dev/null 2>&1; then
  hash_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  hash_cmd="shasum -a 256"
else
  echo "reproducible-build-check: missing sha256sum or shasum" >&2
  exit 2
fi

arches=${SDD_REPRO_ARCHES:-"amd64 arm64"}

for arch in $arches; do
  for i in 1 2; do
    src="$work/src$i"
    out="$work/out$i/$arch"
    cache="$work/cache$i"
    mkdir -p "$out"
    CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" GOCACHE="$cache" GOMODCACHE="$gomodcache" \
      GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= \
      go -C "$src" build -trimpath -ldflags "$cli_ldflags" -o "$out/asdx-core" ./cmd/agentic-sdd
    CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" GOCACHE="$cache" GOMODCACHE="$gomodcache" \
      GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= \
      go -C "$src" build -trimpath -ldflags "$cli_ldflags -X main.legacyTarget=asdx-core -X main.legacyName=agentic-sdd" -o "$out/agentic-sdd" ./cmd/agentic-sdd
    CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" GOCACHE="$cache" GOMODCACHE="$gomodcache" \
      GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= \
      go -C "$src/experiments/tui" build -trimpath -ldflags "$tui_ldflags" -o "$out/asdx" ./cmd/tui-companion
    CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" GOCACHE="$cache" GOMODCACHE="$gomodcache" \
      GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS= \
      go -C "$src/experiments/tui" build -trimpath -ldflags "$tui_ldflags -X main.legacyTarget=asdx -X main.legacyName=agentic-sdd-tui" -o "$out/agentic-sdd-tui" ./cmd/tui-companion
  done

  for bin in asdx-core agentic-sdd asdx agentic-sdd-tui; do
    h1=$($hash_cmd "$work/out1/$arch/$bin" | awk '{print $1}')
    h2=$($hash_cmd "$work/out2/$arch/$bin" | awk '{print $1}')
    printf "darwin/%-5s %-16s build1=%s build2=%s\n" "$arch" "$bin" "$h1" "$h2"
    if [ "$h1" != "$h2" ]; then
      echo "reproducible-build-check: MISMATCH for darwin/$arch/$bin" >&2
      exit 1
    fi
  done
done
