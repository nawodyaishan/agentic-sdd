"""Actual loopback courier tests; explicitly network-none, never a real model."""
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import sys
import tempfile
import threading
import time
import unittest

import p1_transport as p1


@unittest.skipUnless(os.environ.get("P1_NETWORK_FIXTURE") == "1", "requires explicit network-none container")
class CourierTests(unittest.TestCase):
    def setUp(self):
        self.assertNotEqual(0, os.getuid())
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.calls = []
        self.status, self.delay = 200, 0
        owner = self
        class Fake(BaseHTTPRequestHandler):
            def do_POST(self):
                owner.calls.append(dict(path=self.path, headers=dict(self.headers),
                                       data=json.loads(self.rfile.read(int(self.headers["Content-Length"])))))
                time.sleep(owner.delay)
                data = json.dumps(dict(id="fake-p1", model="deepseek-v4.1-flash-free",
                    choices=[dict(message=dict(content="P1_OK", reasoning_content="PRIVATE-FAKE-REASONING"), finish_reason="stop")],
                    usage=dict(prompt_tokens=5, completion_tokens=3, total_tokens=8))).encode()
                self.send_response(owner.status)
                if owner.status == 302:
                    self.send_header("Location", "https://example.invalid/never-follow")
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data))); self.end_headers()
                try:
                    self.wfile.write(data)
                except OSError:
                    pass
            def log_message(self, *_args):
                pass
        self.fake = ThreadingHTTPServer(("127.0.0.1", 0), Fake)
        self.thread = threading.Thread(target=self.fake.serve_forever, daemon=True); self.thread.start()
        def stop():
            self.fake.shutdown(); self.fake.server_close(); self.thread.join(timeout=3)
        self.addCleanup(stop)
        self.pilot = p1.Pilot(self.root / "allocation.jsonl", "local-capability", deadline=time.monotonic() + 600,
                              fake_address=("127.0.0.1", self.fake.server_port))
        self.raw = json.dumps(dict(model="deepseek-v4.1-flash-free", messages=[dict(role="user", content="Reply with exactly P1_OK.")],
                                   max_completion_tokens=256, stream=True)).encode()
        self.headers = {"X-SDD-P1": "local-capability", "User-Agent": "native-fixture", "Authorization": "SECRET-CANARY"}

    def test_real_courier_process_dispatches_once_and_strips_private_material(self):
        result = self.pilot.process(self.raw, self.headers)
        self.assertIn(b"P1_OK", result)
        self.assertNotIn(b"PRIVATE-FAKE-REASONING", result)
        self.assertEqual(1, len(self.calls))
        self.assertEqual("/inference/openai/v1/chat/completions", self.calls[0]["path"])
        self.assertNotIn("SECRET-CANARY", json.dumps(self.calls))
        self.assertNotIn("local-capability", json.dumps(self.calls))
        self.assertEqual({"model", "messages", "max_tokens", "stream"}, set(self.calls[0]["data"]))
        self.assertTrue((self.root / "dispatch.ticket").is_file())
        with self.assertRaises(p1.Denied):
            self.pilot.process(self.raw, self.headers)
        self.assertEqual(1, len(self.calls))

    def test_redirect_is_spent_and_not_followed_or_retried(self):
        self.status = 302
        with self.assertRaises(p1.Denied):
            self.pilot.process(self.raw, self.headers)
        self.assertEqual(1, len(self.calls))
        self.assertEqual(302, self.pilot.snapshot()["result"]["http_status"])

    def test_slow_courier_is_killed_and_cannot_dispatch_again(self):
        self.pilot.deadline = time.monotonic() + 0.25
        self.delay = 1
        started = time.monotonic()
        with self.assertRaises(p1.Denied):
            self.pilot.process(self.raw, self.headers)
        self.assertLess(time.monotonic() - started, 0.8)
        self.assertEqual(1, self.pilot.snapshot()["attempts"])
        with self.assertRaises(p1.Denied):
            self.pilot.process(self.raw, self.headers)

    def test_courier_entry_cannot_run_without_allocation_and_tls_is_verified(self):
        result = subprocess.run([sys.executable, "-I", "-B", str(Path(p1.__file__).resolve()), "--courier"],
                                input=b"{}", env={}, capture_output=True, timeout=3)
        self.assertEqual(2, result.returncode)
        self.assertEqual([], self.calls)
        context = ssl.create_default_context()
        self.assertTrue(context.check_hostname)
        self.assertEqual(ssl.CERT_REQUIRED, context.verify_mode)

    def test_http_framing_denials_never_reach_loopback_upstream(self):
        gate = p1.server(self.pilot)
        thread = threading.Thread(target=gate.serve_forever, daemon=True); thread.start()
        try:
            cases = [
                b"POST /v1/chat/completions HTTP/1.1\r\nHost: local\r\nContent-Length: 1\r\nContent-Length: 1\r\n\r\nx",
                b"POST /v1/chat/completions HTTP/1.1\r\nHost: local\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n",
                b"POST /v1/chat/completions HTTP/1.1\r\nHost: local\r\nContent-Length: 4097\r\n\r\n",
                b"CONNECT opencode.ai:443 HTTP/1.1\r\nHost: local\r\n\r\n",
            ]
            for raw in cases:
                with self.subTest(raw=raw), socket.create_connection(("127.0.0.1", gate.server_port), timeout=2) as connection:
                    connection.sendall(raw)
                    self.assertIn(b" 403 ", connection.recv(4096))
            self.assertEqual([], self.calls)
            self.assertEqual(0, self.pilot.snapshot()["attempts"])
        finally:
            gate.shutdown(); gate.server_close(); thread.join(timeout=3)


if __name__ == "__main__":
    unittest.main()
