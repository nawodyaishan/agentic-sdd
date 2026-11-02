"""S0 byte-quota/HTTP controls; no real model or credential is permitted."""
import concurrent.futures
import json
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import os
from pathlib import Path
import socket
import tempfile
import threading
import time
import unittest

import offline_transport as ot


MODEL = "deepseek-v4.1-flash-free"


def body(**changes):
    value = dict(model=MODEL, messages=[dict(role="user", content="synthetic sentinel")], max_tokens=512, stream=False)
    value.update(changes)
    return json.dumps(value).encode()


class BudgetTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.now = [100.0]
        self.options = dict(model=MODEL, max_requests=2, per_input=4096, per_output=1024,
                            total_input=8192, total_output=2048, seconds=10, clock=lambda: self.now[0])

    def gate(self, **changes):
        return ot.Budget(self.root / "allocation.jsonl", **(self.options | changes))

    def test_positive_reservation_is_durable_and_never_refunded(self):
        gate = self.gate()
        reservation = gate.admit(body())
        self.assertEqual(512, reservation["output_bytes_reserved"])
        self.assertEqual(len(body()), reservation["input_bytes"])
        journal = [json.loads(line) for line in (self.root / "allocation.jsonl").read_text().splitlines()]
        self.assertEqual(1, journal[-1]["requests_reserved"])
        gate.fail("synthetic uncertain upstream outcome")
        with self.assertRaises(ot.Denied):
            gate.admit(body())
        self.assertEqual(1, gate.observation()["requests_reserved"])

    def test_wrong_model_paths_and_unreviewed_body_fields_never_reserve(self):
        gate = self.gate()
        for raw, kwargs in [(body(model="deepseek-v4.1-flash"), {}), (body(model="gpt-paid"), {}),
                            (body(), {"method": "GET"}), (body(), {"path": "/v1/chat/completions?route=paid"}),
                            (body(), {"path": "/v1/responses"}), (body(provider={"route": "auto"}), {}),
                            (body(messages=[{"role": "user", "content": [{"type": "image_url", "image_url": "https://example.invalid"}]}]), {})]:
            with self.subTest(raw=raw, kwargs=kwargs), self.assertRaises(ot.Denied):
                gate.admit(raw, **kwargs)
        self.assertEqual(0, gate.observation()["requests_reserved"])

    def test_duplicate_keys_nonfinite_and_invalid_caps_fail_before_reservation(self):
        gate = self.gate()
        for raw in [b'{"model":"first","model":"second"}', b'{"x": NaN}', b'[]', b'{}',
                    *[body(max_tokens=x) for x in (True, 0, -1, 2048, None, 1.5)],
                    body(max_completion_tokens=512), body(stream="yes")]:
            with self.subTest(raw=raw), self.assertRaises(ot.Denied):
                gate.admit(raw)
        self.assertEqual(0, gate.observation()["requests_reserved"])

    def test_requests_from_auxiliary_and_new_sessions_share_one_counter(self):
        gate = self.gate()
        gate.admit(body(messages=[dict(role="user", content="session one main")]))
        gate.admit(body(messages=[dict(role="user", content="session two title or compaction")]))
        with self.assertRaises(ot.Denied):
            gate.admit(body())
        self.assertEqual(2, gate.observation()["requests_reserved"])

    def test_concurrent_attempts_cannot_overspend_reserved_output(self):
        gate = self.gate(total_output=512, max_requests=20)
        def attempt(_):
            try:
                gate.admit(body())
                return True
            except ot.Denied:
                return False
        with concurrent.futures.ThreadPoolExecutor(max_workers=12) as pool:
            results = list(pool.map(attempt, range(30)))
        self.assertEqual(1, sum(results))
        self.assertEqual(512, gate.observation()["output_bytes_reserved"])

    def test_input_and_output_aggregate_limits_are_independent(self):
        gate = self.gate(total_input=len(body()))
        gate.admit(body())
        with self.assertRaises(ot.Denied):
            gate.admit(body())
        self.assertEqual(len(body()), gate.observation()["input_bytes"])

    def test_expiry_and_cancellation_stop_new_dispatch(self):
        gate = self.gate()
        self.now[0] = 110.0
        with self.assertRaises(ot.Denied):
            gate.admit(body())
        self.assertEqual(0, gate.observation()["requests_reserved"])
        gate.fail("cancelled")
        self.now[0] = 100.0
        with self.assertRaises(ot.Denied):
            gate.admit(body())

    def test_journal_cannot_be_reset_or_reopened_on_restart(self):
        gate = self.gate()
        gate.admit(body())
        with self.assertRaises((ValueError, OSError)):
            self.gate()
        self.assertEqual(0, (self.root / "allocation.jsonl").stat().st_mode & 0o077)

    def test_unknown_real_token_accounting_is_not_renamed_to_bytes(self):
        for accounting in ("deepseek-tokens", "unknown", None):
            with self.subTest(accounting=accounting), self.assertRaises(ValueError):
                self.gate(accounting=accounting)

    def test_one_native_alternative_cap_is_supported_but_two_caps_are_ambiguous(self):
        gate = self.gate()
        data = json.loads(body())
        data["max_completion_tokens"] = data.pop("max_tokens")
        self.assertEqual(512, gate.admit(json.dumps(data).encode())["output_bytes_reserved"])
        with self.assertRaises(ot.Denied):
            gate.admit(body(max_completion_tokens=512))

    def test_only_literal_loopback_upstream_and_no_path_override(self):
        for url in ("https://opencode.ai/inference/openai/v1/chat/completions", "http://localhost:3000/v1/chat/completions",
                    "http://127.0.0.1:3000/other", "http://127.0.0.1:3000/v1/chat/completions?x=1",
                    "http://user@127.0.0.1:3000/v1/chat/completions", "http://127.0.0.1:3000/v1/chat/completions#fragment"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                ot.upstream_address(url)
        self.assertEqual(("127.0.0.1", 3000), ot.upstream_address("http://127.0.0.1:3000/v1/chat/completions"))

    def test_deleted_journal_stops_before_dispatch_and_errors_do_not_echo_input(self):
        gate = self.gate()
        (self.root / "allocation.jsonl").unlink()
        with self.assertRaises(ot.Denied):
            gate.admit(body())
        with self.assertRaises(ot.Denied) as error:
            gate.admit(b'SYNTHETIC-SECRET-DO-NOT-ECHO')
        self.assertNotIn('SYNTHETIC-SECRET', str(error.exception))
        self.assertTrue(gate.observation()["stopped"])


@unittest.skipUnless(os.environ.get("SDD_S0_CONTAINER") == "1", "HTTP fixtures require explicit network-none container")
class HTTPTests(unittest.TestCase):
    def setUp(self):
        self.assertNotEqual(0, os.getuid())
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.requests = []
        self.status, self.response, self.content_type, self.delay = 200, b'{"offline":"fake"}', "application/json", 0
        self.trickle = 0
        owner = self
        class Fake(BaseHTTPRequestHandler):
            def do_POST(self):
                raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
                owner.requests.append(dict(path=self.path, body=raw, headers=dict(self.headers)))
                time.sleep(owner.delay)
                self.send_response(owner.status)
                self.send_header("Content-Type", owner.content_type)
                self.send_header("Content-Length", str(len(owner.response)))
                if owner.status == 302:
                    self.send_header("Location", "https://example.invalid/must-not-follow")
                self.end_headers()
                try:
                    if owner.trickle:
                        for byte in owner.response:
                            self.wfile.write(bytes([byte])); self.wfile.flush(); time.sleep(owner.trickle)
                    else:
                        self.wfile.write(owner.response)
                except (BrokenPipeError, ConnectionResetError):
                    pass
            def log_message(self, *_args):
                pass
        self.upstream = ThreadingHTTPServer(("127.0.0.1", 0), Fake)
        self.start(self.upstream)
        self.gate = ot.Budget(self.root / "allocation.jsonl", model=MODEL, max_requests=2, per_input=65536,
                              per_output=1024, total_input=131072, total_output=2048, seconds=5)
        self.relay = ot.relay_server(self.gate, f"http://127.0.0.1:{self.upstream.server_port}{ot.PATH}")
        self.start(self.relay)

    def start(self, server):
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        def stop():
            server.shutdown(); server.server_close(); thread.join(timeout=3)
            self.assertFalse(thread.is_alive())
        self.addCleanup(stop)

    def request(self, raw=None, *, path=ot.PATH, headers=None):
        connection = http.client.HTTPConnection("127.0.0.1", self.relay.server_port, timeout=3)
        try:
            connection.request("POST", path, body=body() if raw is None else raw,
                               headers={"Content-Type": "application/json", **(headers or {})})
            response = connection.getresponse()
            return response.status, response.read(), response.getheader("Content-Type")
        finally:
            connection.close()

    def test_positive_forwarding_drops_auth_headers_and_shares_request_limit(self):
        canary = "SYNTHETIC-CREDENTIAL-MUST-STAY-OUT"
        status, response, _ = self.request(headers={"Authorization": "Bearer " + canary, "Cookie": canary,
                                                    "Proxy-Authorization": canary, "X-API-Key": canary})
        self.assertEqual((200, self.response), (status, response))
        self.assertEqual(1, len(self.requests))
        self.assertEqual(ot.PATH, self.requests[0]["path"])
        self.assertNotIn(canary, json.dumps(self.requests[0]["headers"]))
        self.assertNotIn(canary, (self.root / "allocation.jsonl").read_text())
        self.assertEqual(200, self.request()[0])  # Could be an auxiliary call.
        self.assertEqual(403, self.request()[0])  # Same allocation, not reset.
        self.assertEqual(2, len(self.requests))

    def test_malformed_or_wrong_model_never_reaches_fake_endpoint(self):
        for raw, path in ((body(model="paid-model"), ot.PATH), (b'{"broken":', ot.PATH),
                          (body(), ot.PATH + "?route=other")):
            self.assertEqual(403, self.request(raw, path=path)[0])
        self.assertEqual([], self.requests)

    def test_redirect_or_http_error_is_not_retried_or_followed(self):
        self.status = 302
        self.assertEqual(403, self.request()[0])
        self.assertEqual(403, self.request()[0])
        self.assertEqual(1, len(self.requests))
        self.assertTrue(self.gate.observation()["stopped"])
        self.assertEqual(1, self.gate.observation()["requests_reserved"])

    def test_oversized_response_spends_reservation_and_stops(self):
        self.response = b"x" * 513
        self.assertEqual(403, self.request()[0])
        self.assertEqual(512, self.gate.observation()["output_bytes_reserved"])
        self.assertEqual(403, self.request()[0])
        self.assertEqual(1, len(self.requests))

    def test_absolute_deadline_bounds_slow_upstream(self):
        self.gate.deadline = self.gate.clock() + 0.1
        self.delay = 0.5
        started = time.monotonic()
        self.assertEqual(403, self.request()[0])
        self.assertLess(time.monotonic() - started, 0.4)
        self.assertEqual(403, self.request()[0])
        self.assertTrue(self.gate.observation()["stopped"])

    def test_buffered_sse_is_bounded_and_never_claimed_as_native_compaction(self):
        self.content_type = "text/event-stream"
        self.response = b'data: {"choices":[]}\n\ndata: [DONE]\n\n'
        status, response, content_type = self.request(body(stream=True))
        self.assertEqual((200, self.response, self.content_type), (status, response, content_type))

    def test_ambiguous_http_framing_and_upgrade_never_forward(self):
        for headers in ("Transfer-Encoding: chunked\r\n", "Content-Length: 4\r\nContent-Length: 4\r\n",
                        "Content-Length: 4\r\nUpgrade: websocket\r\n", "Content-Length: 9999999\r\n"):
            with self.subTest(headers=headers):
                with socket.create_connection(("127.0.0.1", self.relay.server_port), timeout=2) as sock:
                    sock.sendall(("POST " + ot.PATH + " HTTP/1.1\r\nHost: 127.0.0.1\r\n" + headers +
                                  "Content-Type: application/json\r\nConnection: close\r\n\r\n{}\r\n").encode())
                    self.assertIn(b"403", sock.recv(1024).split(b"\r\n")[0])
        self.assertEqual([], self.requests)

    def test_unknown_http_method_does_not_echo_attacker_controlled_text(self):
        canary = "SYNTHETIC-CANARY-VERB"
        connection = http.client.HTTPConnection("127.0.0.1", self.relay.server_port, timeout=3)
        try:
            connection.request(canary, ot.PATH)
            response = connection.getresponse()
            self.assertEqual(403, response.status)
            self.assertNotIn(canary.encode(), response.read())
        finally:
            connection.close()
        self.assertEqual([], self.requests)

    def test_trickle_response_cannot_extend_absolute_deadline(self):
        self.gate.deadline = self.gate.clock() + 0.15
        self.trickle = 0.04
        started = time.monotonic()
        self.assertEqual(403, self.request()[0])
        self.assertLess(time.monotonic() - started, 0.4)
        self.assertTrue(self.gate.observation()["stopped"])


if __name__ == "__main__":
    unittest.main()
