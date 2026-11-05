"""Deterministic 2.2.5 tool dispatch, never online inference.

Run in a fresh rootless user/network namespace with loopback enabled. The fake
endpoint emits scripted tool calls; it is NOT a reviewer or a live-pilot route.
No host credential/config directory is mounted or copied into the fresh HOME.
"""
import base64
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from urllib.request import Request, urlopen

PIN = "8d060e1d2fbb9bb44cf09ae6ed1f3f6a7cff070b5227e2f90970132b23a92050"
PARENT = "sdd-review-coordinator"
CHILD = "sdd-independent-reviewer"
REFERENCE = "sdd-offline/dispatch-fixture"


def native_environment(root, project):
    # 2.2.5 run/run.ts trusts PWD before process.cwd(). Keep both identical.
    return dict(PATH="/usr/local/bin:/usr/bin:/bin", HOME=str(root / "home"),
                PWD=str(project), XDG_CONFIG_HOME=str(root / "config"),
                XDG_DATA_HOME=str(root / "data"), XDG_CACHE_HOME=str(root / "cache"),
                XDG_STATE_HOME=str(root / "state"), TMPDIR=str(root),
                OPENCODE_PASSWORD="offline-local-only", SDD_FAKE_KEY="synthetic-only")


@unittest.skipUnless(os.environ.get("SDD_IR_DISPATCH") == "1", "isolated VM fixture only")
class NativeDispatchTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if {name for _, name in socket.if_nameindex()} != {"lo"}:
            raise RuntimeError("requires a network namespace with loopback only")
        host_namespace = os.environ["SDD_IR_HOST_NETNS"]
        if not host_namespace.startswith("net:[") or os.readlink("/proc/self/ns/net") == host_namespace:
            raise RuntimeError("do not use the host network namespace")
        cls.binary = os.environ["SDD_IR_BINARY"]
        if hashlib.sha256(Path(cls.binary).read_bytes()).hexdigest() != PIN:
            raise RuntimeError("unreviewed binary")

    def setUp(self):
        self.root = Path(tempfile.mkdtemp(prefix="dispatch-", dir=os.environ["SDD_IR_OUTPUT"]))
        self.root.chmod(0o700)
        self.project = self.root / "fixture"
        (self.project / "src").mkdir(parents=True)
        (self.project / "guidance").mkdir()
        (self.project / "src/allowed.txt").write_text("READ_ALLOWED_SENTINEL\n")
        (self.project / "guidance/method.md").write_text("GUIDANCE_ALLOWED_SENTINEL\n")
        (self.root / "excluded.txt").write_text("EXCLUDED_ANSWER_SENTINEL\n")
        (self.project / "src/escape.txt").symlink_to(self.root / "excluded.txt")
        self.requests = []
        self.child_waiting = threading.Event()
        self.release = threading.Event()
        self.cancel_case = False
        self.symlink_case = False
        guidance = Path(os.environ["SDD_IR_GUIDANCE"])
        for source in guidance.rglob("*"):
            if source.is_symlink() or not (source.is_dir() or source.is_file()):
                raise RuntimeError("guidance export contains a symlink or special file")
        shutil.copytree(guidance, self.project / ".agents/skills")
        fixture = self

        class Fake(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                if len(fixture.requests) >= 12:
                    self.send_error(403, "offline fixture request ceiling")
                    return
                fixture.requests.append(body)
                messages = body["messages"]
                texts = json.dumps(messages)
                is_child = any(m.get("role") == "user" and "OFFLINE_CHILD" in str(m.get("content"))
                               for m in messages)
                has_results = any(m.get("role") == "tool" for m in messages)
                if "OFFLINE_LOCATION" in texts:
                    delta, finish = {"content": "OFFLINE_LOCATION_OK"}, "stop"
                elif is_child and fixture.cancel_case:
                    fixture.child_waiting.set()
                    fixture.release.wait(15)
                    delta, finish = {"content": "OFFLINE_CANCEL_RELEASE"}, "stop"
                elif has_results:
                    delta, finish = {"content": "OFFLINE_SCRIPT_DONE_NOT_A_MODEL_REVIEW"}, "stop"
                else:
                    calls = ([('read', {"path": "src/allowed.txt"}),
                              ('read', {"path": "guidance/method.md"}),
                              ('skill', {"id": "agentic-sdd-verification-review"}),
                              ('read', {"path": ".agents/skills/agentic-sdd-router/references/independent-review.md"}),
                              ('read', {"path": "../excluded.txt"}),
                              ('write', {"path": "src/allowed.txt", "content": "CHANGED"}),
                              ('edit', {"path": "src/allowed.txt", "oldString": "READ", "newString": "CHANGED"}),
                              ('patch', {"patchText": "*** Begin Patch\n*** Add File: changed\n+x\n*** End Patch"}),
                              ('shell', {"command": "touch prohibited-shell"}),
                              ('execute', {"code": "return 1"}),
                              ('webfetch', {"url": "https://example.invalid"}),
                              ('subagent', {"agent": "general", "description": "Forbidden nested child",
                                            "prompt": "Never execute"})]
                             if is_child else
                             [('subagent', {"agent": CHILD, "description": "Offline boundary fixture",
                                            "prompt": "OFFLINE_CHILD synthetic boundary check"})])
                    if is_child and fixture.symlink_case:
                        calls.append(('read', {"path": "src/escape.txt"}))
                    delta = {"role": "assistant", "tool_calls": [
                        dict(index=i, id=f"call_fixture_{i}", type="function",
                             function={"name": name, "arguments": json.dumps(args)})
                        for i, (name, args) in enumerate(calls)]}
                    finish = "tool_calls"
                chunks = [dict(id="chatcmpl-offline", object="chat.completion.chunk", created=0,
                               model="dispatch-fixture", choices=[dict(index=0, delta=delta, finish_reason=None)]),
                          dict(id="chatcmpl-offline", object="chat.completion.chunk", created=0,
                               model="dispatch-fixture", choices=[dict(index=0, delta={}, finish_reason=finish)])]
                payload = ("".join("data: " + json.dumps(c) + "\n\n" for c in chunks) + "data: [DONE]\n\n").encode()
                try:
                    self.send_response(200)
                    self.send_header("Content-Type", "text/event-stream")
                    self.send_header("Content-Length", str(len(payload)))
                    self.end_headers()
                    self.wfile.write(payload)
                except (BrokenPipeError, ConnectionResetError):
                    pass

        self.fake = ThreadingHTTPServer(("127.0.0.1", 0), Fake)
        self.fake.daemon_threads = True
        self.thread = threading.Thread(target=self.fake.serve_forever, daemon=True)
        self.thread.start()
        permission = [dict(action="*", resource="*", effect="deny"),
                      dict(action="read", resource="src/allowed.txt", effect="allow"),
                      dict(action="read", resource="src/escape.txt", effect="allow"),
                      dict(action="read", resource="guidance/method.md", effect="allow"),
                      dict(action="skill", resource="agentic-sdd-verification-review", effect="allow"),
                      dict(action="read", resource=".agents/skills/agentic-sdd-router/references/independent-review.md", effect="allow")]
        agents = {name: dict(mode=mode, model=REFERENCE, steps=4,
                            system="OFFLINE synthetic dispatch test only", permissions=permission + extra)
                  for name, mode, extra in [(PARENT, "primary", [dict(action="subagent", resource=CHILD, effect="allow")]),
                                             (CHILD, "subagent", [])]}
        self.config = dict(warming=False, formatter=False, snapshots=False, compaction=dict(auto=False),
                           experimental=dict(background_compaction=False, subagent_depth=1),
                           permissions=[dict(action="*", resource="*", effect="deny")],
                           policies=[dict(action="provider.use", resource="*", effect="deny"),
                                     dict(action="provider.use", resource="sdd-offline", effect="allow"),
                                     dict(action="integration.use", resource="*", effect="deny"),
                                     dict(action="integration.use", resource="sdd-offline", effect="allow"),
                                     dict(action="integration.use", resource="skill:agentic-sdd-verification-review", effect="allow")],
                           agents=agents,
                           providers={"sdd-offline": dict(name="Offline scripted response only", env=["SDD_FAKE_KEY"],
                               package="@opencode/ai/providers/openai-compatible",
                               settings=dict(baseURL=f"http://127.0.0.1:{self.fake.server_port}/v1"),
                               models={"dispatch-fixture": dict(modelID="dispatch-fixture",
                                   limit=dict(context=32768, input=16384, output=2048),
                                   capabilities=dict(tools=True, input=["text"], output=["text"]))})})
        (self.project / "opencode.jsonc").write_text(json.dumps(self.config))
        self.env = native_environment(self.root, self.project)
        self.log = (self.root / "server.log").open("w")
        sock = socket.socket(); sock.bind(("127.0.0.1", 0)); self.port = sock.getsockname()[1]; sock.close()
        self.server = subprocess.Popen([self.binary, "serve", "--hostname", "127.0.0.1", "--port", str(self.port)],
                                       cwd=self.project, env=self.env, stdout=self.log, stderr=self.log)
        for _ in range(100):
            try:
                self.api("GET", "/api/info")
                break
            except OSError:
                if self.server.poll() is not None:
                    self.fail("server exited: " + (self.root / "server.log").read_text())
                time.sleep(.1)
        else:
            self.fail("private server unavailable")

    def tearDown(self):
        self.release.set()
        if hasattr(self, "parent"):
            context = self.api("GET", f"/api/session/{self.parent}/context")["data"]
            (self.root / "parent-context.json").write_text(json.dumps(context, indent=2))
            for message in context:
                if message.get("type") == "error":
                    print("OFFLINE FIXTURE ERROR:", message)
        self.server.terminate()
        try:
            self.server.wait(timeout=10)
        except subprocess.TimeoutExpired:
            self.server.kill(); self.server.wait(timeout=5)
        self.log.close()
        self.fake.shutdown(); self.fake.server_close(); self.thread.join(timeout=3)
        # Synthetic-only transcript, no credential headers or real inference.
        (self.root / "scripted-requests.json").write_text(json.dumps(self.requests, indent=2))

    def api(self, method, path, body=None):
        auth = "Basic " + base64.b64encode(b"opencode:offline-local-only").decode()
        req = Request(f"http://127.0.0.1:{self.port}" + path, method=method,
                      data=json.dumps(body).encode() if body is not None else None,
                      headers={"Authorization": auth, "Content-Type": "application/json"})
        with urlopen(req, timeout=10) as response:
            return None if response.status == 204 else json.load(response)

    def start_parent(self):
        parent = self.api("POST", "/api/session", dict(agent=PARENT, title="Offline scripted boundary",
                            model=dict(providerID="sdd-offline", id="dispatch-fixture"),
                            location=dict(directory=str(self.project))))["data"]
        self.parent = parent["id"]
        self.api("POST", f'/api/session/{parent["id"]}/prompt',
                 dict(text="OFFLINE_PARENT PARENT_ONLY_CONTEXT_SENTINEL"))
        return parent["id"]

    def wait_inactive(self, seconds=20):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if not self.api("GET", "/api/session/active")["data"]:
                return
            time.sleep(.1)
        self.fail("owned session(s) remained active")

    def check_dispatch(self):
        parent = self.start_parent()
        self.wait_inactive()
        sessions = self.api("GET", "/api/session")["data"]
        children = [s for s in sessions if s.get("parentID") == parent]
        self.assertEqual(1, len(children), sessions)
        context = self.api("GET", f'/api/session/{children[0]["id"]}/context')["data"]
        (self.root / "child-context.json").write_text(json.dumps(context, indent=2))
        text = json.dumps(context)
        self.assertNotIn("PARENT_ONLY_CONTEXT_SENTINEL", text)
        self.assertIn("READ_ALLOWED_SENTINEL", text)
        self.assertIn("GUIDANCE_ALLOWED_SENTINEL", text)
        self.assertIn("Independent review method", text)
        self.assertIn("Agentic SDD Verification Review", text)
        tools = [t for m in context if m.get("type") == "assistant"
                 for t in m.get("content", []) if t.get("type") == "tool"]
        self.assertEqual(12 + int(self.symlink_case), len(tools), tools)
        self.assertEqual(["completed"] * 4 + ["error"] * 8,
                         [t["state"]["status"] for t in tools[:12]])
        self.assertEqual("READ_ALLOWED_SENTINEL\n", (self.project / "src/allowed.txt").read_text())
        self.assertFalse((self.project / "prohibited-shell").exists())
        self.assertFalse((self.project / "changed").exists())
        self.assertFalse([s for s in sessions if s.get("parentID") == children[0]["id"]])
        self.assertTrue("EXCLUDED_ANSWER_SENTINEL" not in text,
                        "Unsafe read: allowed symlink returned excluded synthetic sentinel; see child-context.json")
        if self.symlink_case:
            self.assertEqual("error", tools[-1]["state"]["status"])
        print("ACTUAL DISPATCH: file/guidance/skill reads; external, write/edit/patch, shell, execute, web, nested agent refused")

    def test_actual_child_reads_denials_and_preservation(self):
        self.check_dispatch()

    def test_symlink_escape_must_be_refused(self):
        # Expected to FAIL on the observed 2.2.5 route. Do not turn a breach
        # into a passing negative test or claim general path enforcement.
        self.symlink_case = True
        self.check_dispatch()

    def test_owned_parent_child_interrupt(self):
        self.cancel_case = True
        parent = self.start_parent()
        self.assertTrue(self.child_waiting.wait(timeout=15), "scripted child never reached blocked request")
        active = self.api("GET", "/api/session/active")["data"]
        self.assertEqual(2, len(active), active)
        self.assertIn(parent, active)
        started = time.monotonic()
        result = self.api("POST", f"/api/session/{parent}/interrupt", {})
        self.assertTrue(result["interrupted"], result)
        for sid in active:
            if sid != parent:
                self.api("POST", f"/api/session/{sid}/interrupt", {})
        self.wait_inactive(seconds=5)
        self.assertLess(time.monotonic() - started, 5)
        for sid in active:
            context = self.api("GET", f"/api/session/{sid}/context")["data"]
            (self.root / ("cancel-" + sid + ".json")).write_text(json.dumps(context, indent=2))
            self.assertEqual("idle", context[-1].get("type"))
            self.assertEqual("interrupted", context[-1].get("outcome"))
        count = len(self.requests)
        time.sleep(.3)
        self.assertEqual(count, len(self.requests), "new scripted request after interrupt")
        print("ACTUAL LOCAL CANCELLATION: parent and child inactive within 5 seconds; not upstream-provider certification")

    def test_cli_honors_corrected_pwd_and_finds_coordinator(self):
        Path(self.env["HOME"]).mkdir(mode=0o700, exist_ok=True)
        stale_env = self.env | {"PWD": self.env["HOME"]}
        command = [self.binary, "--standalone", "run",
                   "--agent", PARENT, "--model", REFERENCE, "--title", "Offline location regression",
                   "--format", "json", "OFFLINE_LOCATION"]
        stale = subprocess.run(command, cwd=self.project, env=stale_env,
                               capture_output=True, text=True, timeout=20)
        (self.root / "stale-pwd.jsonl").write_text(stale.stdout)
        self.assertEqual(1, stale.returncode, stale.stdout + stale.stderr)
        self.assertIn("Agent not found", stale.stdout)
        self.assertEqual([], self.requests, "bad location must fail before scripted transport")
        result = subprocess.run(
            command,
            cwd=self.project, env=self.env, capture_output=True, text=True, timeout=20)
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)
        self.assertIn("OFFLINE_LOCATION_OK", result.stdout)
        sessions = self.api("GET", "/api/session")["data"]
        locations = {s["location"]["directory"] for s in sessions}
        self.assertEqual({self.env["HOME"], str(self.project)}, locations)
        self.assertTrue(all(s["agent"] == PARENT for s in sessions))
        print("ACTUAL CLI LOCATION: corrected PWD resolved fixture coordinator and returned scripted sentinel")


if __name__ == "__main__":
    unittest.main(verbosity=2)
