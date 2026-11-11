#!/bin/sh
# Local core-only evidence, not a replacement for the trusted aggregate gate.
set -eu

if [ "$#" -ne 1 ]; then
    echo "usage: sh tests/supported-core-go.sh test|vet" >&2
    exit 2
fi
case "$1" in
    test|vet) mode=$1 ;;
    *) echo "unsupported core check: $1" >&2; exit 2 ;;
esac

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo"

# Verify required core package paths and disabled boundary file exist.
for dir in internal/version internal/skillsync internal/harnesscheck internal/harnessevidence internal/appops internal/buildinfo internal/fleet cmd/agentic-sdd cmd/sdd-check; do
    if [ ! -d "$dir" ]; then
        echo "missing required core directory: $dir" >&2
        exit 2
    fi
done
for file in go.mod cmd/agentic-sdd/opencode_disabled.go; do
    if [ ! -f "$file" ]; then
        echo "missing required core file: $file" >&2
        exit 2
    fi
done

# Do not inherit test filters or a foreign workspace/configuration into this scope.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=
echo "Supported-core $mode only; excluded validations not selected; not an aggregate-gate result."
go "$mode" . ./internal/version ./internal/skillsync ./internal/harnesscheck ./internal/harnessevidence ./internal/appops ./internal/buildinfo ./internal/fleet ./cmd/agentic-sdd ./cmd/sdd-check
