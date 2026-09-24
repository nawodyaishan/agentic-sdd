#!/bin/sh
set -eu

go build -o /tmp/agentic-sdd ./cmd/agentic-sdd
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
repo=$root/repo
home=$root/home
mkdir -p "$repo/skills-files/agentic-sdd-plan" "$home/.claude/skills/agentic-sdd-plan" "$home/.claude/skills/unrelated"
printf 'new\n' > "$repo/skills-files/agentic-sdd-plan/SKILL.md"
printf 'old\n' > "$home/.claude/skills/agentic-sdd-plan/SKILL.md"
printf 'keep\n' > "$home/.claude/skills/unrelated/SKILL.md"

/tmp/agentic-sdd preview --repo "$repo" --home "$home" > "$root/preview.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = old
test ! -e "$repo/backups"

/tmp/agentic-sdd apply --repo "$repo" --home "$home" > "$root/apply.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = new
test "$(cat "$home/.claude/skills/unrelated/SKILL.md")" = keep
backup=$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d)
test -n "$backup"
test "$(cat "$backup/claude/agentic-sdd-plan/SKILL.md")" = old

/tmp/agentic-sdd apply --repo "$repo" --home "$home" > "$root/repeat.txt"
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 1
if /tmp/agentic-sdd unknown --repo "$repo" --home "$home" > "$root/invalid.txt" 2>&1; then
  echo 'invalid command unexpectedly succeeded' >&2
  exit 1
fi

echo 'Docker end to end checks passed.'
