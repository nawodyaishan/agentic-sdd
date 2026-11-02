"""Only synthetic code in a separate network-none worker, never a native tool."""
import json
import os
from pathlib import Path
import socket
import sys


assert os.getuid() != 0
assert Path("/input/sentinel.txt").read_text() == "synthetic worker input\n"
assert os.environ.get("SDD_RUNTIME_SECRET") is None
for path in ("/var/run/docker.sock", "/fixture/offline_transport.py", "/usr/local/bin/opencode", sys.argv[1]):
    assert not Path(path).exists(), "runtime/host artifact must not enter worker"
try:
    Path("/root-write").write_text("forbidden")
except OSError:
    pass
else:
    raise AssertionError("root filesystem is writable")
for host, port in (("1.1.1.1", 443), ("127.0.0.1", 4096)):
    try:
        connection = socket.create_connection((host, port), timeout=0.2)
    except OSError:
        pass
    else:
        connection.close()
        raise AssertionError("unexpected worker network/control access")
print(json.dumps(dict(kind="separate offline worker", native_tool_execution=False, nonroot=True,
                      runtime_state_absent=True, network_denied=True, root_readonly=True, model_requests=0)))
