"""Literal P1 client in an internal-only container; never read a real home."""
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
from urllib.error import HTTPError
from urllib.request import Request, urlopen


MODEL = "opencode/deepseek-v4.1-flash-free"
PROMPT = "Reply with exactly P1_OK."
GATE = "http://p1-gate:8000"


def main():
    if os.getuid() == 0 or any(name in os.environ for name in ("OPENCODE_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY")):
        raise ValueError("non-root credential-free runtime required")
    if Path("/control").exists() or Path("/var/run/docker.sock").exists():
        raise ValueError("control/host socket must not be mounted into runtime")
    routes = Path("/proc/net/route").read_text().splitlines()[1:]
    if any(line.split()[1] == "00000000" for line in routes):
        raise ValueError("runtime has a default route")
    try:
        connection = socket.create_connection(("1.1.1.1", 443), timeout=0.2)
    except OSError:
        pass
    else:
        connection.close(); raise ValueError("runtime direct egress not denied")
    version_home = Path("/tmp/p1-version-home"); version_home.mkdir(mode=0o700)
    version = subprocess.check_output(["opencode", "--version"], timeout=15,
        env={"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(version_home),
             "XDG_CONFIG_HOME": "/tmp/p1-version-config", "XDG_DATA_HOME": "/tmp/p1-version-data",
             "XDG_CACHE_HOME": "/tmp/p1-version-cache", "XDG_STATE_HOME": "/tmp/p1-version-state"}).decode().strip()
    if version != "opencode v2.2.5":
        raise ValueError("exact approved P1 runtime required")
    for _ in range(30):
        try:
            with urlopen(GATE + "/ready", timeout=1) as response:
                assert json.load(response)["attempts"] == 0
            break
        except OSError:
            time.sleep(0.1)
    else:
        raise ValueError("P1 gate unavailable; no fallback")

    def post(data, path="/v1/chat/completions", authenticated=True):
        headers = {"Content-Type": "application/json"}
        if authenticated:
            headers["X-SDD-P1"] = os.environ["P1_GATE_TOKEN"]
        try:
            with urlopen(Request(GATE + path, data=data, headers=headers), timeout=3) as response:
                return response.status
        except HTTPError as error:
            return error.code

    fake = os.environ["P1_MODE"] == "fake"
    if fake:
        good = dict(model="deepseek-v4.1-flash-free", messages=[dict(role="user", content=PROMPT)], max_tokens=256)
        for payload, path, auth in ((json.dumps(good | {"model": "paid-wrong-model"}).encode(), "/v1/chat/completions", True),
                                    (json.dumps(good).encode(), "/v1/chat/completions", False),
                                    (b"x" * 4097, "/v1/chat/completions", True),
                                    (json.dumps(good).encode(), "/other", True),
                                    (json.dumps(good | {"max_completion_tokens": 256}).encode(), "/v1/chat/completions", True)):
            assert post(payload, path, auth) == 403
        with urlopen(GATE + "/ready", timeout=1) as response:
            assert json.load(response)["attempts"] == 0

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp); project = root / "project"; project.mkdir()
        config = dict(model=MODEL, default_agent="p1", warming=False, formatter=False, snapshots=False,
            compaction=dict(auto=False), experimental=dict(background_compaction=False),
            permissions=[dict(action="*", resource="*", effect="deny")],
            policies=[dict(action="provider.use", resource="*", effect="deny"),
                      dict(action="provider.use", resource="opencode", effect="allow"),
                      dict(action="integration.use", resource="*", effect="deny"),
                      dict(action="integration.use", resource="opencode", effect="allow"),
                      *[dict(action=action, resource="*", effect="deny") for action in ("mcp.use", "skill.use", "plugin.use", "tool.use")]],
            # steps=1 inserts an assistant limit-warning into the initial input.
            # Gate allocation, denied tools and process deadline bound this lab;
            # do not synthesize an assistant turn just to bound native steps.
            agents={"p1": dict(mode="primary", model=MODEL, system="Synthetic P1 text observation. No tools."),
                    **{key: dict(model=MODEL) for key in ("title", "summary", "compaction")}},
            providers={"opencode": dict(settings=dict(baseURL=GATE + "/v1"),
                headers={"X-SDD-P1": "{env:P1_GATE_TOKEN}"},
                models={"deepseek-v4.1-flash-free": dict(modelID="deepseek-v4.1-flash-free",
                    package="@opencode/ai/providers/openai-compatible", limit=dict(context=32768, input=16384, output=256))})})
        (project / "opencode.jsonc").write_text(json.dumps(config))
        env = {"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(root / "home"),
               "XDG_CONFIG_HOME": str(root / "config"), "XDG_DATA_HOME": str(root / "data"),
               "XDG_CACHE_HOME": str(root / "cache"), "XDG_STATE_HOME": str(root / "state"), "TMPDIR": str(root),
               "P1_GATE_TOKEN": os.environ["P1_GATE_TOKEN"], "OPENCODE_PASSWORD": "synthetic-p1-local-service"}
        result = subprocess.run(["opencode", "run", "--standalone", "--format", "json", "--agent", "p1", "--model", MODEL, PROMPT],
                                cwd=project, env=env, capture_output=True, timeout=120)
        events = []
        errors = []
        sessions = set()
        for line in result.stdout.splitlines():
            try:
                event = json.loads(line)
            except ValueError:
                continue
            if event.get("type") in ("text", "error", "step_start", "step_finish", "tool_use"):
                events.append(event["type"])
            if isinstance(event.get("sessionID"), str):
                sessions.add(event["sessionID"])
            if event.get("type") == "error":
                errors.append(event.get("error", {}).get("type"))
        observed = dict(kind="P1 fake topology" if fake else "P1 single real observation", native_version=version,
                        native_exit=result.returncode, event_types=events, error_types=errors,
                        returned_literal_seen=b"P1_OK" in result.stdout, session_ids=sorted(sessions),
                        direct_egress_denied=True, no_control_mount=True,
                        config_sha256=hashlib.sha256((project / "opencode.jsonc").read_bytes()).hexdigest())
        print(json.dumps(observed), flush=True)
        if fake:
            with urlopen(GATE + "/ready", timeout=1) as response:
                print(json.dumps(dict(fake_gate_diagnostics=json.load(response))), flush=True)
            assert result.returncode == 0 and observed["returned_literal_seen"], "native fake P1 did not complete"
            assert post(json.dumps(good).encode()) == 403, "replay must not dispatch"
            with urlopen(GATE + "/ready", timeout=1) as response:
                assert json.load(response)["attempts"] == 1


if __name__ == "__main__":
    main()
