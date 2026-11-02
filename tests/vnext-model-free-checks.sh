#!/bin/sh
# Model-free batch checks for the 005 final human cycle.
#
# Intended to run inside tests/vnext-container.sh (or any disposable Linux
# container with agentic-sdd, /src read-only and /work writable). It uses
# /tmp/home and /work/qa-project only, starts no model/provider request, and
# exits nonzero if any check fails.
set -u

CLI=${AGENTIC_SDD:-agentic-sdd}
H=/tmp/home
W=/work/qa-project
pass=0
fail=0
skip=0

ok() { pass=$((pass + 1)); printf 'PASS %s\n' "$1"; }
bad() { fail=$((fail + 1)); printf 'FAIL %s: %s\n' "$1" "$2"; }
skipped() { skip=$((skip + 1)); printf 'SKIP %s: %s\n' "$1" "$2"; }

rm -rf "$H" "$W"
mkdir -p "$W" || { printf 'FAIL setup: cannot create %s\n' "$W"; exit 1; }

# 1. CLI identity
if "$CLI" version >/tmp/qa-version 2>&1; then
  ok cli-version
else
  bad cli-version "agentic-sdd version failed"
fi

# 2. OpenCode availability (mounted read-only by the recipe)
if command -v opencode >/dev/null 2>&1; then
  if opencode --version 2>/dev/null | grep -q 'v2\.2\.5'; then
    ok opencode-version
  else
    bad opencode-version "expected opencode v2.2.5"
  fi
else
  skipped opencode-version "opencode not mounted (NO_OPENCODE=1?)"
fi

# 3. Preview writes nothing
if "$CLI" preview --home "$H" >/tmp/qa-preview 2>&1 &&
  grep -q 'Preview only' /tmp/qa-preview &&
  [ ! -e "$H/.agents/skills" ]; then
  ok preview-no-write
else
  bad preview-no-write "preview output or empty-home check failed"
fi

# 4. Apply installs 18 skills into each of four clients (72 copies)
if "$CLI" apply --home "$H" >/tmp/qa-apply1 2>&1; then
  total=$(find "$H/.agents/skills" "$H/.codex/skills" "$H/.claude/skills" \
    "$H/.gemini/antigravity-cli/skills" -name SKILL.md 2>/dev/null | wc -l | tr -d ' ')
  links=$(find "$H/.agents" "$H/.codex" "$H/.claude" "$H/.gemini" -type l 2>/dev/null | wc -l | tr -d ' ')
  if [ "$total" = 72 ] && [ "$links" = 0 ]; then
    ok apply-72-zero-links
  else
    bad apply-72-zero-links "SKILL.md=$total symlinks=$links"
  fi
else
  bad apply-72-zero-links "apply failed"
fi

# 5. Second apply is a no-op
if "$CLI" apply --home "$H" >/tmp/qa-apply2 2>&1 &&
  grep -q 'All skills are up to date\.' /tmp/qa-apply2; then
  ok apply-noop
else
  bad apply-noop "second apply was not a no-op"
fi

# 6. Edit one installed copy; apply replaces it and records a backup
marker="$H/.agents/skills/agentic-sdd-router/SKILL.md"
printf '\nqa-marker\n' >>"$marker"
if "$CLI" apply --home "$H" >/tmp/qa-apply3 2>&1 &&
  grep -q 'replace + back up agents/agentic-sdd-router' /tmp/qa-apply3 &&
  "$CLI" backups --home "$H" 2>/dev/null | head -1 | grep -q '1 replaced, 0 installed across agents'; then
  ok edit-replaced-with-backup
else
  bad edit-replaced-with-backup "replace or backup summary missing"
fi

# 7. Canonical copy restored
if ! grep -q qa-marker "$marker" 2>/dev/null &&
  diff /src/skills-files/agentic-sdd-router/SKILL.md "$marker" >/dev/null 2>&1; then
  ok canonical-after-apply
else
  bad canonical-after-apply "edited bytes survived the apply"
fi

# 8. Restore returns the pre-apply copy; re-apply returns canonical
backup_id=$("$CLI" backups --home "$H" 2>/dev/null | head -1 | awk '{print $2}')
if [ -n "$backup_id" ] &&
  "$CLI" restore "$backup_id" --apply --home "$H" >/tmp/qa-restore 2>&1 &&
  grep -q qa-marker "$marker" 2>/dev/null &&
  ! diff /src/skills-files/agentic-sdd-router/SKILL.md "$marker" >/dev/null 2>&1; then
  ok restore-pre-apply-copy
else
  bad restore-pre-apply-copy "restore did not return the pre-apply copy"
fi
if "$CLI" apply --home "$H" >/tmp/qa-apply4 2>&1 &&
  ! grep -q qa-marker "$marker" 2>/dev/null; then
  ok restore-then-canonical
else
  bad restore-then-canonical "re-apply did not restore canonical bytes"
fi

# 9. Candidate and preflight stay inactive/blocked
if "$CLI" opencode candidate --feature demo --build-path src >/tmp/qa-candidate.jsonc 2>&1 &&
  "$CLI" opencode preflight --config /tmp/qa-candidate.jsonc >/tmp/qa-preflight.json 2>&1 &&
  grep -q '"online_ready":false' /tmp/qa-preflight.json &&
  grep -q '"target_runtime":"2\.2\.5"' /tmp/qa-preflight.json &&
  grep -q 'providers-disabled' /tmp/qa-preflight.json &&
  grep -q 'strict-token-egress-accounting-unverified' /tmp/qa-preflight.json; then
  ok candidate-preflight
else
  bad candidate-preflight "preflight did not report the blocked candidate"
fi

# 10. Config preview writes nothing
if "$CLI" opencode config --workspace "$W" --candidate /tmp/qa-candidate.jsonc >/tmp/qa-config.json 2>&1 &&
  grep -q '"original":"absent"' /tmp/qa-config.json &&
  grep -q '"noop":false' /tmp/qa-config.json &&
  grep -q '"destination":"\.opencode/opencode\.jsonc"' /tmp/qa-config.json &&
  [ ! -e "$W/.opencode" ]; then
  ok config-preview-no-write
else
  bad config-preview-no-write "config preview wrote or returned unexpected data"
fi

# 11. Admission stays fail-closed
if "$CLI" opencode session prepare --workspace "$W" --executable /usr/local/bin/opencode >/tmp/qa-prepare.json 2>&1 &&
  grep -q '"execution":"prepared"' /tmp/qa-prepare.json &&
  "$CLI" opencode session run --workspace "$W" --executable /usr/local/bin/opencode --prompt hello >/tmp/qa-run.json 2>&1 &&
  grep -q '"execution":"blocked"' /tmp/qa-run.json; then
  ok admission-blocked
else
  bad admission-blocked "prepare/run did not stay prepared/blocked"
fi

# 12. Evidence identity/refusals (optional: needs git)
if command -v git >/dev/null 2>&1; then
  if git -C "$W" init -q 2>/dev/null &&
    git -C "$W" config user.email qa@example.invalid &&
    git -C "$W" config user.name QA &&
    printf 'hi\n' >"$W/code.txt" &&
    git -C "$W" add . &&
    git -C "$W" commit -qm fixture &&
    "$CLI" opencode evidence snapshot --workspace "$W" --scope . >/tmp/qa-snapshot.json 2>&1 &&
    grep -Eq '"diff":"sha256-v1:[0-9a-f]{64}"' /tmp/qa-snapshot.json &&
    grep -Eq '"base_revision":"[0-9a-f]{40}"' /tmp/qa-snapshot.json; then
    ok evidence-snapshot
  else
    bad evidence-snapshot "snapshot identity missing or malformed"
  fi
  if "$CLI" opencode evidence snapshot --workspace "$W" --scope ../outside >/dev/null 2>&1; then
    bad evidence-refuses-traversal "unsafe scope was accepted"
  else
    ok evidence-refuses-traversal
  fi
else
  skipped evidence-snapshot "git absent in this image; run the host fallback"
  skipped evidence-refuses-traversal "git absent in this image"
fi

printf '\nSummary: %d passed, %d failed, %d skipped\n' "$pass" "$fail" "$skip"
[ "$fail" -eq 0 ]
