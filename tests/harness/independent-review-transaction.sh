#!/bin/sh
# Scratch-only walkthrough of the manual, digest-checked procedure in
# integrations/opencode/independent-review.md. It uses a private temporary
# workspace under TMPDIR (default /tmp/opencode), never a real home or project,
# and never the 005 config helper, which rejects this candidate by design.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
candidate=$root/integrations/opencode/independent-review.jsonc
base=${TMPDIR:-/tmp/opencode}
mkdir -p "$base"
work=$(mktemp -d "$base/ir-transaction.XXXXXX")
chmod 700 "$work"
trap 'rm -rf "$work"' EXIT
ws=$work/ws
config=$ws/.opencode/opencode.jsonc
mkdir -p "$ws/.opencode"

digest() {
  if [ -e "$1" ]; then sha256sum "$1" | cut -d' ' -f1; else echo absent; fi
}
check() {
  if [ "$1" = "$2" ]; then echo "PASS: $3"; else echo "FAIL: $3 (got $1, want $2)" >&2; exit 1; fi
}
# Refuse rollback unless the current file is exactly what this apply wrote.
rollback() {
  current=$(digest "$1")
  if [ "$current" != "$3" ]; then echo "REFUSED: current config is not the applied candidate"; return 3; fi
  if [ "$(cat "$2/original.sha256")" = absent ]; then
    rm -f "$1"
  else
    cp -p "$2/original.jsonc" "$1.restore" && mv "$1.restore" "$1"
  fi
  check "$(digest "$1")" "$(cat "$2/original.sha256")" "rollback restored the original digest"
}

# The curated whole file keeps one unrelated user key, as the runbook requires.
curated=$work/curated.jsonc
{ printf '{\n  "theme": "keep",\n'; tail -n +2 "$candidate"; } > "$curated"
cand_digest=$(digest "$curated")
check "$(python3 -c 'import json,sys; json.load(open(sys.argv[1])); print("valid")' "$curated")" valid "curated candidate is valid JSON"

# A: identical bytes are a no-op; nothing is backed up or rewritten.
cp "$curated" "$config"
check "$(digest "$config")" "$cand_digest" "identical bytes need no rewrite"

# B: replace an existing user config, then roll back cleanly; a second rollback is a no-op.
printf '{"theme":"keep"}\n' > "$config"
tx=$work/tx-b; mkdir -m 700 "$tx"
cp -p "$config" "$tx/original.jsonc"; digest "$config" > "$tx/original.sha256"
orig=$(digest "$config")
cp "$curated" "$config.new" && mv "$config.new" "$config"
check "$(digest "$config")" "$cand_digest" "candidate applied over the original"
grep -q '"theme"' "$config" && echo "PASS: unrelated key preserved in the curated candidate"
rollback "$config" "$tx" "$cand_digest"
check "$(digest "$config")" "$orig" "clean rollback restores the original bytes"

# C: an intervening edit blocks rollback and leaves the edited file untouched.
tx=$work/tx-c; mkdir -m 700 "$tx"
cp -p "$config" "$tx/original.jsonc"; digest "$config" > "$tx/original.sha256"
cp "$curated" "$config.new" && mv "$config.new" "$config"
printf '\n' >> "$config"
edited=$(digest "$config")
if rollback "$config" "$tx" "$cand_digest"; then echo "FAIL: rollback ran after an intervening edit" >&2; exit 1; fi
check "$(digest "$config")" "$edited" "intervening edit left untouched after refusal"

# D: an absent original removes only the new config file; unrelated state stays.
rm -f "$config"
printf 'keep\n' > "$ws/.opencode/notes.txt"
tx=$work/tx-d; mkdir -m 700 "$tx"
echo absent > "$tx/original.sha256"
cp "$curated" "$config.new" && mv "$config.new" "$config"
rollback "$config" "$tx" "$cand_digest"
check "$(cat "$ws/.opencode/notes.txt")" keep "unrelated .opencode state preserved after rollback"
