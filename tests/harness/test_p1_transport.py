"""P1 policy tests. The exchange is fake; never call a real provider here."""
import concurrent.futures
import json
from pathlib import Path
import tempfile
import unittest

import p1_transport as p1


MODEL = "deepseek-v4.1-flash-free"
PROMPT = "Reply with exactly P1_OK."


def request(**overrides):
    data = dict(model=MODEL, messages=[dict(role="user", content=PROMPT)], max_completion_tokens=256, stream=True)
    data.update(overrides)
    return json.dumps(data).encode()


class PilotTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.calls = []
        self.now = [1.0]
        owner = self
        class Fake(p1.Pilot):
            def exchange(self, payload, headers, deadline):
                owner.calls.append((json.loads(payload), headers, deadline))
                return 200, json.dumps(dict(id="fake-response", model=MODEL,
                    choices=[dict(message=dict(content="P1_OK", reasoning_content="MUST-NOT-EXPORT"), finish_reason="stop")],
                    usage=dict(prompt_tokens=5, completion_tokens=3, total_tokens=8))).encode()
        self.pilot = Fake(self.root / "allocation.jsonl", "local-fixture-token", deadline=601, clock=lambda: self.now[0])
        self.headers = {"X-SDD-P1": "local-fixture-token", "User-Agent": "genuine-offline-fixture",
                        "Authorization": "Bearer DO-NOT-FORWARD", "Cookie": "DO-NOT-FORWARD"}

    def test_one_canonical_text_request_and_only_curated_observations(self):
        result = self.pilot.process(request(tools=[{"private": "must not reach provider"}]), self.headers)
        self.assertEqual(1, len(self.calls))
        payload, headers, deadline = self.calls[0]
        self.assertEqual(dict(model=MODEL, messages=[dict(role="user", content="Reply with exactly P1_OK.")],
                              max_tokens=256, stream=False), payload)
        self.assertEqual({"User-Agent": "genuine-offline-fixture", "Content-Type": "application/json"}, headers)
        self.assertEqual(91, deadline)
        self.assertIn(b"P1_OK", result)
        self.assertNotIn(b"MUST-NOT-EXPORT", result)
        snapshot = self.pilot.snapshot()
        self.assertEqual(1, snapshot["attempts"])
        self.assertEqual(5, snapshot["result"]["usage"]["prompt_tokens"])
        self.assertIsNone(snapshot["result"]["billing_usd"])
        persisted = (self.root / "allocation.jsonl").read_text()
        self.assertNotIn("MUST-NOT-EXPORT", persisted)
        self.assertNotIn("DO-NOT-FORWARD", persisted)
        self.assertNotIn("local-fixture-token", persisted)

    def test_wrong_model_auth_prompt_caps_or_body_never_dispatch(self):
        for raw, headers in [(request(model="deepseek-v4.1-flash"), self.headers),
                             (request(), {}), (request(), {"X-SDD-P1": "wrong"}),
                             (request(messages=[dict(role="user", content="not the approved prompt")]), self.headers),
                             (request(max_completion_tokens=257), self.headers),
                             (request(max_tokens=256), self.headers),
                             (b"x" * 4097, self.headers),
                             (b'{"model":"first","model":"second"}', self.headers)]:
            with self.subTest(raw=raw), self.assertRaises(p1.Denied):
                self.pilot.process(raw, headers)
        self.assertEqual([], self.calls)
        self.assertEqual(0, self.pilot.snapshot()["attempts"])

    def test_all_parallel_replays_share_one_spent_attempt(self):
        def invoke(_):
            try:
                self.pilot.process(request(), self.headers)
                return True
            except p1.Denied:
                return False
        with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
            outcomes = list(pool.map(invoke, range(20)))
        self.assertEqual(1, sum(outcomes))
        self.assertEqual(1, len(self.calls))

    def test_native_cli_json_quoted_literal_is_normalized_without_extra_text(self):
        self.pilot.process(request(messages=[dict(role="system", content="discard this synthetic native framing"),
                                             dict(role="user", content=json.dumps(PROMPT))]), self.headers)
        self.assertEqual([dict(role="user", content=PROMPT)], self.calls[0][0]["messages"])

    def test_quoted_literal_with_extra_instructions_is_never_admitted(self):
        with self.assertRaises(p1.Denied):
            self.pilot.process(request(messages=[dict(role="user", content=json.dumps(PROMPT) + " extra")]), self.headers)
        self.assertEqual([], self.calls)

    def test_restart_or_missing_journal_cannot_reset_allocation(self):
        with self.assertRaises((ValueError, OSError)):
            p1.Pilot(self.root / "allocation.jsonl", "new", deadline=601)
        (self.root / "allocation.jsonl").unlink()
        with self.assertRaises(p1.Denied):
            self.pilot.process(request(), self.headers)
        self.assertEqual([], self.calls)

    def test_expired_or_cancelled_run_never_dispatches(self):
        self.now[0] = 601
        with self.assertRaises(p1.Denied):
            self.pilot.process(request(), self.headers)
        self.now[0] = 1
        self.pilot.cancel()
        with self.assertRaises(p1.Denied):
            self.pilot.process(request(), self.headers)
        self.assertEqual([], self.calls)

    def test_upstream_redirect_or_failure_is_spent_without_retry(self):
        def failure(*_args):
            self.calls.append("failed upstream")
            return 302, b"ignored"
        self.pilot.exchange = failure
        for _ in range(2):
            with self.assertRaises(p1.Denied):
                self.pilot.process(request(), self.headers)
        self.assertEqual(1, len(self.calls))
        self.assertEqual(1, self.pilot.snapshot()["attempts"])

    def test_missing_usage_or_excess_output_does_not_manufacture_success(self):
        def missing(*_args):
            return 200, b'{"choices":[{"message":{"content":"text"},"finish_reason":"stop"}]}'
        self.pilot.exchange = missing
        with self.assertRaises(p1.Denied):
            self.pilot.process(request(), self.headers)
        self.assertIsNone(self.pilot.snapshot()["result"]["usage"])
        self.assertEqual(1, self.pilot.snapshot()["attempts"])


if __name__ == "__main__":
    unittest.main()
