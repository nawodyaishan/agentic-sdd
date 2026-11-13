#!/bin/sh
set -eu

go build -o /tmp/asdx-core ./cmd/agentic-sdd
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
repo=$root/repo
home=$root/home
mkdir -p "$repo/skills-files/agentic-sdd-plan" "$home/.claude/skills/agentic-sdd-plan" "$home/.claude/skills/unrelated"
printf 'new\n' > "$repo/skills-files/agentic-sdd-plan/SKILL.md"
printf 'old\n' > "$home/.claude/skills/agentic-sdd-plan/SKILL.md"
printf 'keep\n' > "$home/.claude/skills/unrelated/SKILL.md"

/tmp/asdx-core preview --repo "$repo" --home "$home" > "$root/preview.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = old
test ! -e "$repo/backups"

/tmp/asdx-core apply --repo "$repo" --home "$home" > "$root/apply.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = new
test "$(cat "$home/.claude/skills/unrelated/SKILL.md")" = keep
backup1=$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d)
test -n "$backup1"
test "$(cat "$backup1/claude/agentic-sdd-plan/SKILL.md")" = old
grep -q '"format": 2' "$backup1/manifest.json"
grep -q '"operation": "apply"' "$backup1/manifest.json"
backup1_id=$(basename "$backup1")

/tmp/asdx-core apply --repo "$repo" --home "$home" > "$root/repeat.txt"
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 1

# List backups: the one apply-created backup shows its id and what it holds.
/tmp/asdx-core backups --repo "$repo" --home "$home" > "$root/backups.txt"
grep -q "$backup1_id" "$root/backups.txt"
grep -q 'replaced' "$root/backups.txt"
grep -q 'installed' "$root/backups.txt"
# "restore list" is the same listing.
/tmp/asdx-core restore list --repo "$repo" --home "$home" > "$root/restore-list.txt"
diff "$root/backups.txt" "$root/restore-list.txt"

# Preview a restore: shows the plan, writes nothing.
/tmp/asdx-core restore "$backup1_id" --repo "$repo" --home "$home" > "$root/restore-preview.txt"
grep -q 'Preview only' "$root/restore-preview.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = new
test -e "$home/.agents/skills/agentic-sdd-plan"
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 1

# Apply the restore: the replaced skill reverts, and the skills apply
# installed fresh (never backed up, so nothing to put back) are removed -
# restore is apply's exact inverse for a format 2 backup.
/tmp/asdx-core restore "$backup1_id" --apply --repo "$repo" --home "$home" > "$root/restore-apply.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = old
test ! -e "$home/.agents/skills/agentic-sdd-plan"
test "$(cat "$home/.claude/skills/unrelated/SKILL.md")" = keep
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 2
backup2_id=$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | grep -v "^$backup1_id\$")
backup2="$repo/backups/$backup2_id"
grep -q '"operation": "restore"' "$backup2/manifest.json"
grep -q "\"restored_from\": \"$backup1_id\"" "$backup2/manifest.json"

# Repeating the restore is a no-op: no third backup.
/tmp/asdx-core restore "$backup1_id" --apply --repo "$repo" --home "$home" > "$root/restore-repeat.txt"
grep -q 'up to date' "$root/restore-repeat.txt"
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 2

# Restoring the pre-restore backup undoes the restore, returning to the
# post-apply state: the reverted skill flips back, and the removed fresh
# install is reinstalled with its original content.
/tmp/asdx-core restore apply "$backup2_id" --repo "$repo" --home "$home" > "$root/restore-undo.txt"
test "$(cat "$home/.claude/skills/agentic-sdd-plan/SKILL.md")" = new
test "$(cat "$home/.agents/skills/agentic-sdd-plan/SKILL.md")" = new
test "$(cat "$home/.claude/skills/unrelated/SKILL.md")" = keep
test "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" = 3

if /tmp/asdx-core unknown --repo "$repo" --home "$home" > "$root/invalid.txt" 2>&1; then
  echo 'invalid command unexpectedly succeeded' >&2
  exit 1
fi

echo 'Docker end to end checks passed.'
