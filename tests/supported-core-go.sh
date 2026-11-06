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

# Explicit files keep the experimental CLI test file out of the test binary.
# All production CLI files are compiled as dependencies, not runtime certification.
set -- cmd/agentic-sdd/main.go cmd/agentic-sdd/logs.go \
    cmd/agentic-sdd/opencode.go cmd/agentic-sdd/main_test.go
for file do
    if [ ! -f "$file" ]; then
        echo "missing reviewed CLI Go file: $file" >&2
        exit 2
    fi
done
for file in cmd/agentic-sdd/*.go; do
    case "$file" in
        cmd/agentic-sdd/main.go|cmd/agentic-sdd/logs.go|cmd/agentic-sdd/opencode.go|cmd/agentic-sdd/main_test.go|cmd/agentic-sdd/opencode_test.go) ;;
        *) echo "unreviewed CLI Go file; review core selection: $file" >&2; exit 2 ;;
    esac
done

# Do not inherit test filters or a foreign workspace/configuration into this scope.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=
echo "Supported-core $mode only; excluded validations not selected; not an aggregate-gate result."
go "$mode" ./internal/skillsync ./internal/harnesscheck ./internal/harnessevidence ./cmd/sdd-check
go "$mode" "$@"
