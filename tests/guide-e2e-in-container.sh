#!/bin/sh
set -eu

go build -o /tmp/asdx-core ./cmd/agentic-sdd
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
unset AGENTIC_SDD_ANIMATE

# J1 - guided fresh install, no pack: menu 1, decline the pack, confirm.
printf '1\nn\ny\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j1.jsonl" \
  /tmp/asdx-core guide --repo "$repo" --home "$home1" > "$root/j1.out"
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
  /tmp/asdx-core guide --repo "$repo" --home "$home2" > "$root/j2.out"
test -z "$(find "$home2" -mindepth 1 -print -quit)"
grep -q '"decision":"no"' "$root/j2.jsonl"

# J3 - guided restore of J1's backup after a local modification: menu 2,
# select backup 1, confirm. The backup recorded all four skills as freshly
# installed, so restoring removes them again.
printf 'modified\n' > "$home1/.claude/skills/agentic-sdd-plan/SKILL.md"
printf '2\n1\ny\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j3.jsonl" \
  /tmp/asdx-core guide --repo "$repo" --home "$home1" > "$root/j3.out"
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
/tmp/asdx-core guide --repo "$repo" --home "$home2" </dev/null \
  > "$root/j4-guide-devnull.out" 2>&1 || code=$?
test "$code" = 2
code=0
printf '' | /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/j4-guide-pipe.out" 2>&1 || code=$?
test "$code" = 2
code=0
/tmp/asdx-core restore --repo "$repo" --home "$home1" </dev/null \
  > "$root/j4-restore.out" 2>&1 || code=$?
test "$code" = 2

# J5 - scripted automation output is unchanged (no Next: summary block).
/tmp/asdx-core preview --repo "$repo" --home "$home2" \
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
  /tmp/asdx-core preview --repo "$repo" --home "$home2" \
  > "$root/j6-color.txt"
grep -q "$esc" "$root/j6-color.txt"
AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color NO_COLOR=1 \
  /tmp/asdx-core preview --repo "$repo" --home "$home2" \
  > "$root/j6-nocolor.txt"
if grep -q "$esc" "$root/j6-nocolor.txt"; then
  echo 'NO_COLOR output unexpectedly contained an escape sequence' >&2
  exit 1
fi
env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=dumb \
  /tmp/asdx-core preview --repo "$repo" --home "$home2" \
  > "$root/j6-dumb.txt"
if grep -q "$esc" "$root/j6-dumb.txt"; then
  echo 'TERM=dumb output unexpectedly contained an escape sequence' >&2
  exit 1
fi

# Static guide entry: color vs NO_COLOR, intro printed once.
# Entering 4 ("What is this?") and then q proves the branded intro is printed
# once at startup and not replayed when returning to the menu.
printf '4\nq\n' | env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color COLUMNS=80 \
  /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/guide-static-color.out"
grep -q "$esc" "$root/guide-static-color.out"
grep -q 'Build software with AI. Stay the engineer.' "$root/guide-static-color.out"
grep -q 'by nawodyaishan' "$root/guide-static-color.out"
grep -q 'Version:' "$root/guide-static-color.out"
grep -q 'Docs: https://github.com/nawodyaishan/agentic-sdd#readme' "$root/guide-static-color.out"
test "$(grep -c 'Build software with AI. Stay the engineer.' "$root/guide-static-color.out")" -eq 1

printf '4\nq\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 TERM=xterm-256color NO_COLOR=1 COLUMNS=80 \
  /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/guide-static-nocolor.out"
if grep -q "$esc" "$root/guide-static-nocolor.out"; then
  echo 'NO_COLOR guide unexpectedly contained an escape sequence' >&2
  exit 1
fi
grep -q 'Build software with AI. Stay the engineer.' "$root/guide-static-nocolor.out"
grep -q 'by nawodyaishan' "$root/guide-static-nocolor.out"
grep -q 'Version:' "$root/guide-static-nocolor.out"
grep -q 'Docs: https://github.com/nawodyaishan/agentic-sdd#readme' "$root/guide-static-nocolor.out"
test "$(grep -c 'Build software with AI. Stay the engineer.' "$root/guide-static-nocolor.out")" -eq 1

# Narrow terminal (< 60 columns, e.g. COLUMNS=40): art is omitted, plain text Agentic SDD is used.
printf 'q\n' | env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 COLUMNS=40 TERM=xterm-256color \
  /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/guide-narrow.out"
grep -q 'Agentic SDD' "$root/guide-narrow.out"
grep -Fxq 'Build software with AI. Stay the' "$root/guide-narrow.out"
grep -Fxq 'engineer.' "$root/guide-narrow.out"
if grep -q 'A G E N T I C' "$root/guide-narrow.out"; then
  echo 'Narrow guide unexpectedly printed the ASCII wordmark' >&2
  exit 1
fi
grep -q 'by nawodyaishan' "$root/guide-narrow.out"
grep -q 'Version:' "$root/guide-narrow.out"
grep -q 'Docs: https://github.com/nawodyaishan/agentic-sdd#readme' "$root/guide-narrow.out"

# Two-pass color sweep: nineteen moving-frame redraws and one settling redraw.
printf '4\nq\n' | env -u NO_COLOR AGENTIC_SDD_FORCE_INTERACTIVE=1 AGENTIC_SDD_ANIMATE=1 \
  TERM=xterm-256color COLUMNS=80 \
  /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/guide-animate.out"
grep -q "$esc" "$root/guide-animate.out"
grep -q 'Build software with AI. Stay the engineer.' "$root/guide-animate.out"
grep -q 'by nawodyaishan' "$root/guide-animate.out"
grep -q 'Version:' "$root/guide-animate.out"
grep -q 'Docs: https://github.com/nawodyaishan/agentic-sdd#readme' "$root/guide-animate.out"
test "$(grep -c 'Build software with AI. Stay the engineer.' "$root/guide-animate.out")" -eq 1
test "$(tr -cd '\015' < "$root/guide-animate.out" | wc -c)" -eq 20

# Animation suppressed when NO_COLOR=1 even if AGENTIC_SDD_ANIMATE=1 is set.
printf 'q\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 AGENTIC_SDD_ANIMATE=1 \
  TERM=xterm-256color NO_COLOR=1 \
  /tmp/asdx-core guide --repo "$repo" --home "$home2" \
  > "$root/guide-animate-nocolor.out"
if grep -q "$esc" "$root/guide-animate-nocolor.out"; then
  echo 'Animated guide with NO_COLOR unexpectedly contained an escape sequence' >&2
  exit 1
fi
test "$(tr -cd '\015' < "$root/guide-animate-nocolor.out" | wc -c)" -eq 0

# J7 - Explicit menu choice 5 update check in offline / network-less container:
# Fails gracefully with generic error, emits trace screen updates.result,
# writes nothing, and returns to the menu where q exits 0.
home_update=$root/home-update
mkdir -p "$home_update"
printf '5\nq\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j7.jsonl" \
  /tmp/asdx-core guide --repo "$repo" --home "$home_update" \
  > "$root/j7.out" 2>&1
grep -q 'Could not check for updates.' "$root/j7.out"
grep -q 'https://github.com/nawodyaishan/agentic-sdd/releases/latest' "$root/j7.out"
test -z "$(find "$home_update" -mindepth 1 -print -quit)"
grep -q '"event":"prompt.answered"' "$root/j7.jsonl"
grep -q '"choice":"5"' "$root/j7.jsonl"
grep -q '"event":"screen.shown"' "$root/j7.jsonl"
grep -q '"screen":"updates.result"' "$root/j7.jsonl"
if grep -q '"event":"op.result"' "$root/j7.jsonl"; then
  echo 'Update check unexpectedly emitted op.result in trace' >&2
  exit 1
fi

# J8 - Optional learning is local/read-only and traces no content or paths.
home_learning=$root/home-learning
mkdir -p "$home_learning"
printf '6\n1\n1\n\nb\nb\nq\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j8.jsonl" \
  /tmp/asdx-core guide --repo "$repo" --home "$home_learning" > "$root/j8.out"
grep -q 'agentic-sdd-plan' "$root/j8.out"
grep -q '"screen":"learn.detail"' "$root/j8.jsonl"
test -z "$(find "$home_learning" -mindepth 1 -print -quit)"
if grep -Eq 'agentic-sdd-plan|home-learning|"event":"op.result"' "$root/j8.jsonl"; then
  echo 'Learning trace leaked content or recorded a write operation' >&2
  exit 1
fi

# J9 - Back reopens options; declining the fresh review writes nothing.
home_review=$root/home-review
mkdir -p "$home_review"
printf '1\nn\nb\nn\nn\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  AGENTIC_SDD_TRACE="$root/j9.jsonl" \
  /tmp/asdx-core guide --repo "$repo" --home "$home_review" > "$root/j9.out"
grep -q '"choice":"b"' "$root/j9.jsonl"
test -z "$(find "$home_review" -mindepth 1 -print -quit)"
if grep -q '"event":"op.result"' "$root/j9.jsonl"; then
  echo 'Review-back/decline unexpectedly executed a write operation' >&2
  exit 1
fi

# J10 - A fixture install shows context-preserving recovery and reentry commands.
home_context="$root/home with spaces"
printf '1\nn\ny\n' | AGENTIC_SDD_FORCE_INTERACTIVE=1 NO_COLOR=1 \
  /tmp/asdx-core guide --repo "$repo" --home "$home_context" > "$root/j10.out"
grep -Fq -- "--home '$home_context'" "$root/j10.out"
grep -Fq -- "--repo $repo" "$root/j10.out"
grep -Fq 'asdx-core restore ' "$root/j10.out"
grep -Fq 'asdx-core guide ' "$root/j10.out"
snapshot=$(find "$repo/backups" -mindepth 1 -maxdepth 1 -type d | sort | tail -n 1)
test -n "$snapshot"
/tmp/asdx-core restore "$(basename "$snapshot")" --repo "$repo" --home "$home_context" > "$root/j10-recovery.out"
grep -q 'Preview only.' "$root/j10-recovery.out"
test "$(cat "$home_context/.claude/skills/agentic-sdd-plan/SKILL.md")" = new

echo 'Guide end to end checks passed.'
