#!/bin/sh
# Offline VM check only: rootless user/network namespace, no external inference.
set -eu
if [ "$#" -lt 3 ]; then
    echo "usage: sh $0 PINNED_BINARY STAGED_GUIDANCE_DIR EVIDENCE_DIR [unittest selectors...]" >&2
    exit 2
fi
binary=$1
guidance=$2
output=$3
shift 3
umask 077
mkdir -p "$output"
chmod 700 "$output"
export SDD_IR_BINARY="$binary" SDD_IR_GUIDANCE="$guidance" SDD_IR_OUTPUT="$output"
export SDD_IR_HOST_NETNS="$(readlink /proc/self/ns/net)"
export SDD_IR_SCRIPT="$(dirname "$0")/native_independent_review_dispatch.py"
unshare --user --map-root-user --net python3 - "$@" <<'PY'
import fcntl
import os
import runpy
import socket
import struct
import sys

# Bring up only the new namespace's loopback; there is no route to a provider.
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
fcntl.ioctl(s.fileno(), 0x8914, struct.pack("16sh", b"lo", 1))
s.close()
os.environ["SDD_IR_DISPATCH"] = "1"
sys.argv[0] = os.environ["SDD_IR_SCRIPT"]
runpy.run_path(sys.argv[0], run_name="__main__")
PY
