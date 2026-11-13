#!/bin/sh
# Model-free, user-driven container for the 005 final human cycle.
# Starts an interactive shell in a disposable Linux container with:
#
# Options:
#   --batch   run tests/vnext-model-free-checks.sh and exit (no TTY required)
#   --full    use the larger local Ubuntu base (git/bash/coreutils) instead of
#             the slim default; no image is ever pulled
#   --online  opt in to outbound network for an interactively authenticated
#             session (credentials stay in ephemeral /tmp; never mounted)
#
#   - no network, no host home, no credentials, no Docker socket;
#   - a read-only repository and read-only asdx-core binary mounted in;
#   - the host's opencode binary mounted read-only and available by default
#     (override with OPENCODE_BINARY, disable with NO_OPENCODE=1);
#   - one explicitly chosen host scratch directory mounted read-write at /work.
#
# You run the model-free scenarios yourself. This script never starts a model,
# provider request, native ACP prompt or installation into a real home.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
agentic=${AGENTIC_SDD_BINARY:-$root/bin/asdx-core}
opencode=${OPENCODE_BINARY:-}
if [ -z "$opencode" ] && [ "${NO_OPENCODE:-0}" != "1" ]; then
  opencode=$(command -v opencode 2>/dev/null || true)
fi
workspace=${SDD_CONTAINER_WORKSPACE:-}

die() {
  printf 'error: %s\n' "$1" >&2
  exit 2
}

batch=0
full=0
online=0
for arg in "$@"; do
  case "$arg" in
    --batch) batch=1 ;;
    --full) full=1 ;;
    --online) online=1 ;;
    *) die "unknown argument $arg; use --batch, --full, --online or no argument" ;;
  esac
done
network=none
if [ "$online" = 1 ]; then
  network=bridge
fi

if [ -n "${SDD_CONTAINER_IMAGE:-}" ]; then
  image=$SDD_CONTAINER_IMAGE
elif [ "$full" = 1 ]; then
  # Larger Ubuntu base with git/bash/coreutils; already present locally.
  image=sha256:191967da651ec0a567de173ef8a029080da8cd4992de192b65df54ef189be769
else
  image=sha256:57b29a7c050c4e679e4bea572e14b3965cf4b5839de2da49d55e49ffc4024f39
fi

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
  die "build the CLI first (task build) or set AGENTIC_SDD_BINARY"
fi
if [ -n "$opencode" ] && { [ ! -f "$opencode" ] || [ ! -x "$opencode" ]; }; then
  die "OPENCODE_BINARY must be an executable file when set"
fi
if [ "$batch" = 0 ] && [ -z "${TERM:-}" ]; then
  die "an interactive terminal is required (or use --batch)"
fi

docker image inspect "$image" >/dev/null 2>&1 || die "container image not present locally; this script never pulls"

set -- \
  --rm --pull=never --network "$network" --read-only \
  --cap-drop=ALL --security-opt=no-new-privileges \
  --pids-limit=256 --memory=1g --cpus=1 \
  --user "$(id -u):$(id -g)" \
  --tmpfs /tmp:rw,nosuid,nodev,exec,mode=1777,size=512m \
  -e HOME=/tmp/home -e XDG_CONFIG_HOME=/tmp/config \
  -e XDG_DATA_HOME=/tmp/data -e XDG_CACHE_HOME=/tmp/cache \
  -e XDG_STATE_HOME=/tmp/state -e SDD_ONLINE="$online" \
  --mount "type=bind,src=$workspace,dst=/work" \
  --mount "type=bind,src=$root,dst=/src,readonly" \
  --mount "type=bind,src=$agentic,dst=/usr/local/bin/asdx-core,readonly" \
  --workdir /work

if [ -n "$opencode" ]; then
  opencode=$(CDPATH= cd -- "$(dirname -- "$opencode")" && pwd -P)/$(basename "$opencode")
  set -- "$@" --mount "type=bind,src=$opencode,dst=/usr/local/bin/opencode,readonly"
fi

if [ "$online" = 1 ]; then
  # Docker may copy the host's systemd-resolved stub (127.0.0.53), which is
  # unreachable inside the container. Use explicit resolvers unless overridden.
  for server in ${SDD_CONTAINER_DNS:-1.1.1.1 8.8.8.8}; do
    set -- "$@" --dns "$server"
  done
fi

if [ "$batch" = 1 ]; then
  if [ "$online" = 1 ]; then
    printf 'note: --online does not change the model-free batch; admission stays blocked\n'
  fi
  printf 'Batch mode (%s network): no host home or credentials; workspace=%s\n' "$network" "$workspace"
  exec docker run "$@" "$image" sh /src/tests/vnext-model-free-checks.sh
fi

exec docker run -it "$@" "$image" sh -c '
  mkdir -p "$HOME" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" "$XDG_STATE_HOME" || exit 1
  printf "Container mode: %s network, no host home or mounted credentials.\n" "$([ "${SDD_ONLINE:-0}" = 1 ] && echo online || echo offline)"
  printf "Workspace: /work (writable scratch)   Repo: /src (read-only)\n"
  if [ "${SDD_ONLINE:-0}" = 1 ]; then
    printf "ONLINE: outbound network is enabled. You may log in inside; credentials stay in ephemeral /tmp.\n"
    printf "        Data sent to a provider leaves this container. No automatic model call is made.\n"
  else
    printf "OFFLINE: no network. OpenCode cannot reach a provider or complete device login.\n"
  fi
  if command -v opencode >/dev/null 2>&1; then
    printf "OpenCode: %s (mounted read-only)\n" "$(opencode --version 2>/dev/null || echo present)"
  else
    printf "note: opencode is not mounted; set OPENCODE_BINARY to the host binary\n"
  fi
  command -v git >/dev/null 2>&1 || printf "note: git is absent; evidence identity steps are optional (see guide)\n"
  if command -v asdx-core >/dev/null 2>&1; then
    if asdx-core apply --home "$HOME" >/tmp/sdd-skills.log 2>&1; then
      count=$(find "$HOME/.agents/skills" -name SKILL.md 2>/dev/null | wc -l | tr -d " ")
      printf "SDD skills: %s installed under ~/.agents/skills (also codex/claude/agy)\n" "$count"
    else
      printf "warning: SDD skill install failed; see /tmp/sdd-skills.log\n"
    fi
    if asdx-core opencode candidate --feature demo --build-path src >/work/sdd-candidate.jsonc 2>/dev/null; then
      printf "Inactive SDD config written to /work/sdd-candidate.jsonc (not activated)\n"
    fi
  fi
  printf "SDD config examples: /src/integrations/opencode (inactive; not applied)\n"
  exec sh
'
