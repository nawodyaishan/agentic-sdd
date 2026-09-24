#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
docker run --rm \
  --network none \
  --mount "type=bind,src=$repo,dst=/src,readonly" \
  --workdir /src \
  -e GOTOOLCHAIN=local \
  -e GOCACHE=/tmp/go-cache \
  golang:1.25-alpine sh tests/e2e-in-container.sh
