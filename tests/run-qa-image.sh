#!/bin/sh
# Run the pinned feature-005 QA image.
# Default: offline, read-only root, no host home or credentials.
# --online: outbound network + explicit DNS so you can log in inside.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
image=${QA_IMAGE:-agentic-sdd-qa:005}
workspace=${SDD_QA_WORKSPACE:-}
online=0
check=0

die() { printf 'error: %s\n' "$1" >&2; exit 2; }
for arg in "$@"; do
  case "$arg" in
    --online) online=1 ;;
    --check) check=1 ;;
    -h|--help)
      printf 'usage: SDD_QA_WORKSPACE=DIR %s [--online] [--check]\n' "$0"
      exit 0 ;;
    *) die "unknown argument $arg; use --online, --check or --help" ;;
  esac
done

[ -n "$workspace" ] || die "set SDD_QA_WORKSPACE to a disposable directory"
[ -d "$workspace" ] || die "workspace does not exist: $workspace"
workspace=$(CDPATH= cd -- "$workspace" && pwd -P)
if [ "$workspace" = "$root" ]; then
  die "refusing the implementation checkout as the disposable workspace"
fi
case "$workspace/" in
  "$root"/*) die "workspace must be outside the implementation checkout" ;;
esac
docker image inspect "$image" >/dev/null 2>&1 || die "image $image is not built; run tests/build-qa-image.sh"

network=none
if [ "$online" = 1 ]; then
  network=bridge
fi

set -- \
  --rm --pull=never --network "$network" --read-only \
  --cap-drop=ALL --security-opt=no-new-privileges \
  --pids-limit=256 --memory=2g --cpus=1 \
  --tmpfs /tmp:rw,nosuid,nodev,exec,mode=1777,size=1g \
  -e XDG_CONFIG_HOME=/tmp/config -e XDG_DATA_HOME=/tmp/data \
  -e XDG_CACHE_HOME=/tmp/cache -e XDG_STATE_HOME=/tmp/state \
  -e SDD_ONLINE="$online" \
  --mount "type=bind,src=$workspace,dst=/home/coder/work" \
  --workdir /home/coder/work

if [ -d "$root" ]; then
  set -- "$@" --mount "type=bind,src=$root,dst=/src,readonly"
fi
if [ "$online" = 1 ]; then
  for server in ${SDD_CONTAINER_DNS:-1.1.1.1 8.8.8.8}; do
    set -- "$@" --dns "$server"
  done
fi

if [ "$check" = 1 ]; then
  exec docker run "$@" "$image" sh /src/tests/vnext-model-free-checks.sh
fi

exec docker run -it "$@" "$image"
