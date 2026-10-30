#!/bin/sh
# Read only a frozen Git tree/toolchain; no credentials, host home or network.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
base=1b50933193c622197eeab45fa0a8005b425d76b9
image=sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39
docker image inspect "$image" >/dev/null
toolchain=$(GOTOOLCHAIN=local go env GOROOT)
output=$(mktemp -d /tmp/opencode/sdd-b5-diagnosis.XXXXXX)
mkdir "$output/source" "$output/evidence"
chmod 700 "$output/source" "$output/evidence"
git -C "$repo" archive "$base" | tar -xf - -C "$output/source"
echo "Private fixture evidence: $output/evidence"
exec docker run --rm --pull=never --network=none --read-only \
  --cap-drop=ALL --security-opt=no-new-privileges --pids-limit=256 \
  --memory=2g --cpus=2 --user="$(id -u):$(id -g)" \
  --tmpfs /tmp:rw,exec,nosuid,nodev,mode=1777,size=1g \
  -e HOME=/tmp/home -e GOROOT=/toolchain -e PATH=/toolchain/bin:/usr/local/bin:/usr/bin:/bin \
  -e GOCACHE=/tmp/go-cache -e GOMODCACHE=/tmp/go-mod-cache -e GOENV=off \
  -e GOTOOLCHAIN=local -e GOPROXY=off -e GOSUMDB=off -e CGO_ENABLED=0 \
  -e GOMAXPROCS=2 -e SDD_FIXTURE_CONTAINER=1 \
  --mount "type=bind,src=$toolchain,dst=/toolchain,readonly" \
  --mount "type=bind,src=$output/source,dst=/source,readonly" \
  --mount "type=bind,src=$output/evidence,dst=/evidence" \
  --mount "type=bind,src=$repo/tests/harness/diagnosis_fixtures.py,dst=/probe.py,readonly" \
  "$image" python3 -B /probe.py
