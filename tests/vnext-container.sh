#!/bin/sh
# Model-free, user-driven container for the 005 final human cycle.
#
# Starts an interactive shell in a disposable Linux container with:
#   - no network, no host home, no credentials, no Docker socket;
#   - a read-only repository and read-only agentic-sdd binary mounted in;
#   - the host's opencode binary mounted read-only and available by default
#     (override with OPENCODE_BINARY, disable with NO_OPENCODE=1);
#   - one explicitly chosen host scratch directory mounted read-write at /work.
#
# You run the model-free scenarios yourself. This script never starts a model,
# provider request, native ACP prompt or installation into a real home.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
image=${SDD_CONTAINER_IMAGE:-sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39}
agentic=${AGENTIC_SDD_BINARY:-$root/bin/agentic-sdd}
opencode=${OPENCODE_BINARY:-}
if [ -z "$opencode" ] && [ "${NO_OPENCODE:-0}" != "1" ]; then
  opencode=$(command -v opencode 2>/dev/null || true)
fi
workspace=${SDD_CONTAINER_WORKSPACE:-}

die() {
  printf 'error: %s\n' "$1" >&2
  exit 2
}

[ -n "$workspace" ] || die "set SDD_CONTAINER_WORKSPACE to a new disposable directory (absolute path)"
[ -d "$workspace" ] || die "workspace does not exist: $workspace"
workspace=$(CDPATH= cd -- "$workspace" && pwd -P)
if [ "$workspace" = "$root" ]; then
  die "refusing the implementation checkout as the disposable workspace"
fi
case "$workspace/" in
  "$root"/*) die "workspace must be outside the implementation checkout" ;;
esac
if [ ! -f "$agentic" ] || [ ! -x "$agentic" ]; then
  die "build the CLI first (make build) or set AGENTIC_SDD_BINARY"
fi
if [ -n "$opencode" ] && { [ ! -f "$opencode" ] || [ ! -x "$opencode" ]; }; then
  die "OPENCODE_BINARY must be an executable file when set"
fi
if [ -z "${TERM:-}" ]; then
  die "an interactive terminal is required"
fi

docker image inspect "$image" >/dev/null 2>&1 || die "container image not present locally; this script never pulls"

set -- \
  --rm -it --pull=never --network=none --read-only \
  --cap-drop=ALL --security-opt=no-new-privileges \
  --pids-limit=256 --memory=1g --cpus=1 \
  --user "$(id -u):$(id -g)" \
  --tmpfs /tmp:rw,nosuid,nodev,mode=1777,size=512m \
  -e HOME=/tmp/home -e XDG_CONFIG_HOME=/tmp/config \
  -e XDG_DATA_HOME=/tmp/data -e XDG_CACHE_HOME=/tmp/cache \
  -e XDG_STATE_HOME=/tmp/state \
  --mount "type=bind,src=$workspace,dst=/work" \
  --mount "type=bind,src=$root,dst=/src,readonly" \
  --mount "type=bind,src=$agentic,dst=/usr/local/bin/agentic-sdd,readonly" \
  --workdir /work

if [ -n "$opencode" ]; then
  opencode=$(CDPATH= cd -- "$(dirname -- "$opencode")" && pwd -P)/$(basename "$opencode")
  set -- "$@" --mount "type=bind,src=$opencode,dst=/usr/local/bin/opencode,readonly"
fi

exec docker run "$@" "$image" sh -c '
  mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" "$XDG_STATE_HOME" || exit 1
  printf "Model-free container. No network, credentials or host home.\n"
  printf "Workspace: /work (writable scratch)   Repo: /src (read-only)\n"
  if command -v opencode >/dev/null 2>&1; then
    printf "OpenCode: %s (mounted read-only; no provider/network)\n" "$(opencode --version 2>/dev/null || echo present)"
  else
    printf "note: opencode is not mounted; set OPENCODE_BINARY to the host binary\n"
  fi
  command -v git >/dev/null 2>&1 || printf "note: git is absent; evidence identity steps are optional (see guide)\n"
  exec sh
'
