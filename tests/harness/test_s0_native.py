"""S0 native request construction with fake responses, never real inference."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import offline_transport as ot


MODEL = "deepseek-v4.1-flash-free"


@unittest.skipUnless(os.environ.get("SDD_S0_CONTAINER") == "1", "requires the explicit network-none fixture wrapper")
class NativeRequestTests(unittest.TestCase):
    def test_pinned_runtime_sends_only_to_fake_gate_and_uses_fake_output(self):
        self.assertNotEqual(0, os.getuid())
        version = subprocess.check_output(["opencode", "--version"], timeout=20).decode().strip()
        self.assertEqual("opencode v2.2.5", version, "freeze the explicitly selected observed binary; do not claim a version range")
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            project = root / "project"; project.mkdir()
            requests, shapes = [], []
            class Fake(BaseHTTPRequestHandler):
                def do_POST(self):
                    data = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                    requests.append({"model": data["model"], "stream": data.get("stream", False),
                                     "auth_forwarded": self.headers.get("Authorization") is not None})
                    if data.get("stream"):
                        chunk = dict(id="chatcmpl-s0", object="chat.completion.chunk", created=0, model=MODEL,
                                     choices=[dict(index=0, delta=dict(role="assistant", content="S0_FAKE_ONLY"), finish_reason=None)])
                        finish = dict(id="chatcmpl-s0", object="chat.completion.chunk", created=0, model=MODEL,
                                      choices=[dict(index=0, delta={}, finish_reason="stop")])
                        payload = ("data: " + json.dumps(chunk) + "\n\ndata: " + json.dumps(finish) + "\n\ndata: [DONE]\n\n").encode()
                        kind = "text/event-stream"
                    else:
                        payload = json.dumps(dict(id="chatcmpl-s0", object="chat.completion", created=0, model=MODEL,
                            choices=[dict(index=0, message=dict(role="assistant", content="S0_FAKE_ONLY"), finish_reason="stop")])).encode()
                        kind = "application/json"
                    self.send_response(200)
                    self.send_header("Content-Type", kind)
                    self.send_header("Content-Length", str(len(payload)))
                    self.end_headers(); self.wfile.write(payload)
                def log_message(self, *_args):
                    pass
            upstream = ThreadingHTTPServer(("127.0.0.1", 0), Fake)
            self.start(upstream)
            class Observe(ot.Budget):
                def admit(self, raw, **options):
                    data = json.loads(raw)
                    # Fixed metadata booleans only, never prompts/headers/values.
                    shape = dict(exact_model=data.get("model") == MODEL,
                                       max_tokens=data.get("max_tokens") if type(data.get("max_tokens")) is int else None,
                                       has_alternative_cap="max_completion_tokens" in data,
                                       stream=data.get("stream") is True,
                                       field_names=[key for key in ("model", "messages", "max_tokens", "max_completion_tokens", "tools", "stream") if key in data])
                    shapes.append(shape)
                    try:
                        reservation = super().admit(raw, **options)
                    except ot.Denied:
                        shape["admitted"] = False
                        raise
                    shape["admitted"] = True
                    return reservation
            budget = Observe(root / "allocation.jsonl", model=MODEL, max_requests=6, per_input=131072,
                             per_output=2048, total_input=786432, total_output=12288, seconds=30)
            relay = ot.relay_server(budget, f"http://127.0.0.1:{upstream.server_port}{ot.PATH}")
            self.start(relay)
            reference = "sdd-offline/" + MODEL
            config = dict(model=reference, default_agent="s0", warming=False, formatter=False, snapshots=False,
                compaction=dict(auto=False), experimental=dict(background_compaction=False),
                permissions=[dict(action="*", resource="*", effect="deny")],
                policies=[dict(action="provider.use", resource="*", effect="deny"),
                          dict(action="provider.use", resource="sdd-offline", effect="allow"),
                          dict(action="integration.use", resource="*", effect="deny"),
                          dict(action="integration.use", resource="sdd-offline", effect="allow"),
                          *[dict(action=action, resource="*", effect="deny") for action in ("mcp.use", "skill.use", "plugin.use")]],
                agents={"s0": dict(mode="primary", model=reference, steps=1,
                                   system="Synthetic offline request-shape test. Do not use tools."),
                        **{name: dict(model=reference) for name in ("title", "summary", "compaction")}},
                providers={"sdd-offline": dict(name="S0 fake only", env=["S0_FAKE_KEY"], package="@opencode/ai/providers/openai-compatible",
                    settings=dict(baseURL=f"http://127.0.0.1:{relay.server_port}/v1"),
                    models={MODEL: dict(modelID=MODEL, name="Synthetic fake response, not DeepSeek inference",
                                        limit=dict(context=32768, input=16384, output=1024))})})
            (project / "opencode.jsonc").write_text(json.dumps(config))
            env = {"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": str(root / "home"),
                   "XDG_CONFIG_HOME": str(root / "config"), "XDG_DATA_HOME": str(root / "data"),
                   "XDG_CACHE_HOME": str(root / "cache"), "XDG_STATE_HOME": str(root / "state"),
                   "TMPDIR": str(root), "OPENCODE_PASSWORD": "synthetic-local-only", "S0_FAKE_KEY": "synthetic-not-a-provider-key"}
            result = subprocess.run(["opencode", "run", "--standalone", "--format", "json", "--agent", "s0",
                                     "--model", reference, "Return the synthetic offline sentinel."],
                                    cwd=project, env=env, capture_output=True, timeout=40)
            print(json.dumps(dict(kind="offline native fake-response fixture, not model selection acceptance", version=version,
                                  native_exit=result.returncode, shapes=shapes, fake_upstream=requests,
                                  budget=budget.observation(), model_requests=0)))
            self.assertEqual(0, result.returncode, (result.stderr + result.stdout).decode(errors="replace")[-2000:])
            self.assertIn(b"S0_FAKE_ONLY", result.stdout, "fake output must actually return through the native runner")
            self.assertTrue(requests, "must observe actual HTTP dispatch, not just permission evaluation")
            self.assertTrue(all(r["model"] == MODEL and not r["auth_forwarded"] for r in requests))
            self.assertLessEqual(len(requests), 6)
            self.assertEqual(len(requests), budget.observation()["requests_reserved"])
            self.assertTrue(all(not s["admitted"] for s in shapes if "max_tokens" not in s["field_names"] and not s["has_alternative_cap"]))

    def start(self, server):
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        def stop():
            server.shutdown(); server.server_close(); thread.join(timeout=3)
            self.assertFalse(thread.is_alive())
        self.addCleanup(stop)


if __name__ == "__main__":
    unittest.main()
