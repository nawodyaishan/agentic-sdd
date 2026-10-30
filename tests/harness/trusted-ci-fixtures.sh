#!/bin/sh
# Deterministic tests only. No image pulls, native stores or model calls.
set -eu
umask 077
root=$(mktemp -d /tmp/opencode/sdd-b7-fixtures.XXXXXX)
echo "Private B7 fixture evidence: $root"
TMPDIR=/tmp/opencode python3 -B -m unittest discover -s tests/harness -p test_trusted_ci.py
TMPDIR=/tmp/opencode SDD_TRUSTED_CI_CONTAINERS=1 SDD_CI_EVIDENCE_ROOT="$root" \
  python3 -B -m unittest discover -s tests/harness -p test_trusted_ci_container.py
