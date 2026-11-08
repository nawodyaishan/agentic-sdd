#!/bin/sh
set -eu

go build -o /tmp/agentic-sdd ./cmd/agentic-sdd
root=$(mktemp -d)
trap 'rm -rf "$root"' EXIT
repo=$root/repo
home1=$root/home1
home2=$root/home2
mkdir -p "$repo/skills-files/agentic-sdd-plan" "$home1" "$home2"
printf 'new\n' > "$repo/skills-files/agentic-sdd-plan/SKILL.md"

# The guide, J4 and J5 require forced interaction to be off; make sure a
# forgotten export from the caller cannot change what these checks prove.
unset AGENTIC_SDD_FORCE_INTERACTIVE
unset NO_COLOR

# J1 - guided fresh install, no pack: menu 1, decline the pack, confirm.
printf '1\nn\ny\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j1.jsonl" \
  /tmp/agentic-sdd guide --repo "$repo" --home "$home1" > "$root/j1.out"
for target in \
  .claude/skills \
  .agents/skills \
  .codex/skills \
  .gemini/antigravity-cli/skills; do
  test "$(cat "$home1/$target/agentic-sdd-plan/SKILL.md")" = new
done
test -d "$repo/backups"
test -n "$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d)"
grep -q '"event":"run.start"' "$root/j1.jsonl"
grep -q '"event":"plan.summary"' "$root/j1.jsonl"
grep -q '"event":"confirm"' "$root/j1.jsonl"
grep -q '"decision":"yes"' "$root/j1.jsonl"
grep -q '"event":"op.result"' "$root/j1.jsonl"
grep -q '"op":"apply"' "$root/j1.jsonl"
grep -q '"event":"run.end"' "$root/j1.jsonl"

# J2 - guided install declined: menu 1, decline the pack, decline the install.
printf '1\nn\nn\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j2.jsonl" \
  /tmp/agentic-sdd guide --repo "$repo" --home "$home2" > "$root/j2.out"
test -z "$(find "$home2" -mindepth 1 -print -quit)"
grep -q '"decision":"no"' "$root/j2.jsonl"

# J3 - guided restore of J1's backup after a local modification: menu 2,
# select backup 1, confirm. The backup recorded all four skills as freshly
# installed, so restoring removes them again.
printf 'modified\n' > "$home1/.claude/skills/agentic-sdd-plan/SKILL.md"
printf '2\n1\ny\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j3.jsonl" \
  /tmp/agentic-sdd guide --repo "$repo" --home "$home1" > "$root/j3.out"
for target in \
  .claude/skills \
  .agents/skills \
  .codex/skills \
  .gemini/antigravity-cli/skills; do
  test ! -e "$home1/$target/agentic-sdd-plan"
done
grep -q '"op":"restore"' "$root/j3.jsonl"

# J4 - guide and restore without a terminal (forced interaction unset).
code=0
/tmp/agentic-sdd guide --repo "$repo" --home "$home2" </dev/null \
  > "$root/j4-guide-devnull.out" 2>&1 || code=$?
test "$code" = 2
code=0
printf '' | /tmp/agentic-sdd guide --repo "$repo" --home "$home2" \
  > "$root/j4-guide-pipe.out" 2>&1 || code=$?
test "$code" = 2
code=0
/tmp/agentic-sdd restore --repo "$repo" --home "$home1" </dev/null \
  > "$root/j4-restore.out" 2>&1 || code=$?
test "$code" = 2

# J5 - scripted automation output is unchanged (no Next: summary block).
/tmp/agentic-sdd preview --repo "$repo" --home "$home2" \
  > "$root/j5-preview.txt"
test "$(tail -n 1 "$root/j5-preview.txt")" = \
  'Preview only. Run with apply (or --apply) to install.'
if grep -q 'Next:' "$root/j5-preview.txt"; then
  echo 'plain preview unexpectedly printed a Next: summary block' >&2
  exit 1
fi

# J6 - color and plain presentation at a forced terminal.
esc=$(printf '\033')
env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color \
  /tmp/agentic-sdd preview --repo "$repo" --home "$home2" \
  > "$root/j6-color.txt"
grep -q "$esc" "$root/j6-color.txt"
AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color NO_COLOR=1 \
  /tmp/agentic-sdd preview --repo "$repo" --home "$home2" \
  > "$root/j6-nocolor.txt"
if grep -q "$esc" "$root/j6-nocolor.txt"; then
  echo 'NO_COLOR output unexpectedly contained an escape sequence' >&2
  exit 1
fi
env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=dumb \
  /tmp/agentic-sdd preview --repo "$repo" --home "$home2" \
  > "$root/j6-dumb.txt"
if grep -q "$esc" "$root/j6-dumb.txt"; then
  echo 'TERM=dumb output unexpectedly contained an escape sequence' >&2
  exit 1
fi

echo 'Guide end to end checks passed.'
