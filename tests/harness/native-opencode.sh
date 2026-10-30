#!/bin/sh
# No model calls, host credentials, network, Docker socket or writable checkout.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
binary=${OPENCODE_TEST_BINARY:-$(command -v opencode)}
image=sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39
docker image inspect "$image" >/dev/null
exec docker run --rm --pull=never --network=none --read-only \
  --cap-drop=ALL --security-opt=no-new-privileges --pids-limit=128 \
  --memory=1g --cpus=1 --user=65534:65534 \
  --tmpfs /tmp:rw,nosuid,nodev,mode=1777,size=512m \
  -e HOME=/tmp/home -e XDG_CONFIG_HOME=/tmp/config \
  -e XDG_DATA_HOME=/tmp/data -e XDG_CACHE_HOME=/tmp/cache \
  -e XDG_STATE_HOME=/tmp/state -e SDD_NATIVE_FIXTURE=1 \
  --mount "type=bind,src=$binary,dst=/usr/local/bin/opencode,readonly" \
  --mount "type=bind,src=$root/integrations/opencode,dst=/templates,readonly" \
  --mount "type=bind,src=$root/tests/harness/native_opencode.py,dst=/probe.py,readonly" \
  "$image" python3 /probe.py
