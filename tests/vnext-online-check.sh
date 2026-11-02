#!/bin/sh
# Online-only checks for the 005 human cycle.
#
# Connectivity checks make no model call. The "hi" round trip is opt-in and
# runs exactly one tiny prompt with the model named in SDD_HI_MODEL. Never
# prints credentials. Intended for a container started with --online.
set -u

pass=0
fail=0
skip=0
ok() { pass=$((pass + 1)); printf 'PASS %s\n' "$1"; }
bad() { fail=$((fail + 1)); printf 'FAIL %s: %s\n' "$1" "$2"; }
skipped() { skip=$((skip + 1)); printf 'SKIP %s: %s\n' "$1" "$2"; }

# DNS
if getent hosts opencode.ai >/tmp/online-dns 2>/dev/null && [ -s /tmp/online-dns ]; then
  ok dns-opencode-ai
else
  bad dns-opencode-ai "cannot resolve opencode.ai; check /etc/resolv.conf and --online"
fi

# HTTPS reachability (no credentials sent)
if python3 - <<'PY' >/tmp/online-https 2>&1
import sys, urllib.error, urllib.request
try:
    req = urllib.request.Request("https://opencode.ai/", method="GET")
    with urllib.request.urlopen(req, timeout=15) as r:
        print("http", r.status)
        sys.exit(0 if r.status < 500 else 1)
except urllib.error.HTTPError as exc:
    # A 4xx still proves DNS, TLS and an HTTP response; 5xx is a server failure.
    print("http", exc.code)
    sys.exit(0 if exc.code < 500 else 1)
except Exception as exc:
    print(type(exc).__name__, exc)
    sys.exit(1)
PY
then
  ok https-opencode-ai
else
  bad https-opencode-ai "HTTPS request failed: $(head -c 200 /tmp/online-https)"
fi

# OpenCode binary
if command -v opencode >/dev/null 2>&1 && opencode --version >/tmp/online-version 2>&1; then
  ok opencode-binary
else
  bad opencode-binary "opencode is not available in this container"
fi

# Opt-in single "hi" round trip
if [ "${1:-}" != "--hi" ]; then
  skipped hi-response "connectivity only; use --hi with SDD_HI_MODEL for one prompt"
else
  model=${SDD_HI_MODEL:-}
  if [ -z "$model" ]; then
    skipped hi-response "set SDD_HI_MODEL=provider/model (exact id) to enable"
  else
    if command -v timeout >/dev/null 2>&1; then
      out=$(timeout 120 opencode run --standalone --model "$model" "hi" 2>&1)
      rc=$?
    else
      out=$(opencode run --standalone --model "$model" "hi" 2>&1)
      rc=$?
    fi
    case "$out" in
      *"Transport error"*|*EAI_AGAIN*|*"getaddrinfo"*)
        bad hi-response "network/auth error: $(printf '%s' "$out" | head -c 200)" ;;
      *)
        if [ "$rc" -eq 0 ] && [ -n "$out" ]; then
          ok hi-response
          printf 'response: %s\n' "$(printf '%s' "$out" | head -c 300)"
        else
          bad hi-response "exit=$rc output='$(printf '%s' "$out" | head -c 200)'"
        fi ;;
    esac
  fi
fi

printf '\nSummary: %d passed, %d failed, %d skipped\n' "$pass" "$fail" "$skip"
[ "$fail" -eq 0 ]
