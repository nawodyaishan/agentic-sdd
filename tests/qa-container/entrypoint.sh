#!/bin/sh
# Entrypoint for the pinned feature-005 QA image.
set -eu

if [ ! -f "$HOME/.agents/skills/agentic-sdd-router/SKILL.md" ]; then
  agentic-sdd apply --home "$HOME" >/tmp/qa-skills.log 2>&1 ||
    printf 'warning: SDD skill install failed; see /tmp/qa-skills.log\n'
fi

if [ -d "$HOME/work" ] && [ -w "$HOME/work" ]; then
  agentic-sdd opencode candidate --feature demo --build-path src \
    >"$HOME/work/sdd-candidate.jsonc" 2>/dev/null || true
fi

printf 'Agentic SDD QA image (feature 005).\n'
sed -n '1,8p' /etc/sdd-qa-identity 2>/dev/null || true
printf 'OpenCode: %s\n' "$(opencode --version 2>/dev/null || echo missing)"
printf 'SDD skills: %s under %s/.agents/skills\n' \
  "$(find "$HOME/.agents/skills" -name SKILL.md 2>/dev/null | wc -l | tr -d ' ')" "$HOME"
if [ "${SDD_ONLINE:-0}" = 1 ]; then
  printf 'Network: online (login and model calls are yours; nothing starts automatically)\n'
else
  printf 'Network: offline (default; no provider or device login)\n'
fi
printf 'Workspace: %s/work   Repo: /src (if mounted)   Inactive candidate: %s/work/sdd-candidate.jsonc\n' "$HOME" "$HOME"

exec "$@"
