"""One-attempt P1 observer. Real execution requires the separate reviewed driver."""
import hashlib
import hmac
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import re
import ssl
import stat
import subprocess
import sys
import threading
import time


MODEL = "deepseek-v4.1-flash-free"
PROMPT = "Reply with exactly P1_OK."
PATH = "/v1/chat/completions"
HOST = "opencode.ai"
UPSTREAM_PATH = "/inference/openai/v1/chat/completions"
ALLOCATION = "p1-20261101-one-of-six"


def decode(raw):
    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise Denied("duplicate JSON field")
            result[key] = value
        return result
    def invalid(_value):
        raise Denied("nonfinite JSON")
    try:
        return json.loads(raw, object_pairs_hook=unique, parse_constant=invalid)
    except (ValueError, TypeError, RecursionError):
        raise Denied("invalid JSON") from None


class Denied(ValueError):
    pass


class Pilot:
    def __init__(self, journal, secret, *, deadline, clock=time.monotonic, real=False, fake_address=None):
        self.journal = Path(journal).absolute()
        if (".." in self.journal.parts or any(p.is_symlink() for p in [self.journal, *self.journal.parents]) or
                self.journal.parent.stat().st_mode & 0o077 or not secret):
            raise ValueError("private stable allocation and local capability required")
        if real and (os.environ.get("P1_REAL_ALLOCATION") != ALLOCATION or str(self.journal) != "/control/allocation.jsonl"):
            raise ValueError("real mode requires the explicit allocation-bound driver")
        if fake_address is not None and (fake_address[0] != "127.0.0.1" or type(fake_address[1]) is not int):
            raise ValueError("fake upstream must be numeric loopback")
        self.secret, self.deadline, self.clock = secret, deadline, clock
        self.real, self.fake_address = real, fake_address
        self.lock = threading.Lock()
        self.attempts, self.stopped, self.result = 0, False, None
        self.denials = []
        fd = os.open(self.journal, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        info = os.fstat(fd); os.close(fd)
        self.inode = (info.st_dev, info.st_ino)
        self.record("armed")

    def record(self, phase):
        try:
            fd = os.open(self.journal, os.O_WRONLY | os.O_APPEND | os.O_NOFOLLOW)
            with os.fdopen(fd, "a") as stream:
                info = os.fstat(stream.fileno())
                if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077 or (info.st_dev, info.st_ino) != self.inode:
                    raise OSError("allocation changed")
                stream.write(json.dumps(dict(allocation=ALLOCATION, real_upstream=self.real, phase=phase,
                                             attempts=self.attempts, result=self.result), sort_keys=True) + "\n")
                stream.flush(); os.fsync(stream.fileno())
        except OSError:
            self.stopped = True
            raise Denied("allocation unavailable; never reset or refund") from None

    def remaining(self, end):
        remaining = min(end, self.deadline) - self.clock()
        if self.stopped or not 0 < remaining <= 600:
            raise Denied("stopped or deadline expired")
        return remaining

    def process(self, raw, headers):
        supplied = headers.get("X-SDD-P1", "")
        if not isinstance(supplied, str) or not supplied.isascii() or not hmac.compare_digest(supplied, self.secret):
            raise Denied("local capability rejected")
        if not isinstance(raw, bytes) or not 0 < len(raw) <= 4096:
            raise Denied("native body exceeds approved limit")
        data = decode(raw)
        allowed = {"model", "messages", "max_tokens", "max_completion_tokens", "stream", "stream_options",
                   "tools", "tool_choice", "parallel_tool_calls", "temperature", "top_p", "stop"}
        if not isinstance(data, dict) or set(data) - allowed or data.get("model") != MODEL:
            raise Denied("unapproved request/model")
        caps = [k for k in ("max_tokens", "max_completion_tokens") if k in data]
        if len(caps) != 1 or type(data[caps[0]]) is not int or data[caps[0]] != 256:
            raise Denied("exactly one approved output cap required")
        messages = data.get("messages")
        if not isinstance(messages, list) or any(not isinstance(m, dict) or m.get("role") not in ("system", "user") for m in messages):
            raise Denied("only initial text observation supported")
        users = [m for m in messages if m["role"] == "user"]
        content = users[0].get("content") if len(users) == 1 else None
        # Exact 2.2.5 CLI JSON-quotes its positional literal. Accept only that
        # exact encoding or the literal/text part, never arbitrary JSON/text.
        if content not in (PROMPT, json.dumps(PROMPT), [{"type": "text", "text": PROMPT}]):
            raise Denied("only the approved literal prompt may be observed")
        user_agent = headers.get("User-Agent", "")
        if not isinstance(user_agent, str) or len(user_agent) > 256 or any(ord(c) < 32 or ord(c) > 126 for c in user_agent):
            raise Denied("invalid client identification")
        payload = json.dumps(dict(model=MODEL, messages=[dict(role="user", content=PROMPT)],
                                  max_tokens=256, stream=False), separators=(",", ":")).encode()
        assert len(payload) <= 4096
        with self.lock:
            self.remaining(self.deadline)
            if self.attempts:
                raise Denied("one attempt already reserved; stop")
            self.attempts = 1
            self.result = dict(status="reserved", native_body_sha256=hashlib.sha256(raw).hexdigest(),
                               canonical_body_sha256=hashlib.sha256(payload).hexdigest(),
                               input_count_mode="observed-afterward-not-pre-guaranteed", usage=None, billing_usd=None)
            self.record("reserved-before-network")
        end = min(self.deadline, self.clock() + 90)
        try:
            status, response = self.exchange(payload, {"User-Agent": user_agent, "Content-Type": "application/json"}, end)
            self.result["http_status"] = status
            if status != 200:
                raise Denied("upstream failed; no retry/redirect/credentials")
            body = decode(response)
            if not isinstance(body, dict):
                raise Denied("unexpected provider document")
            usage = body.get("usage", {})
            fields = ("prompt_tokens", "completion_tokens", "total_tokens")
            if (not isinstance(usage, dict) or any(type(usage.get(k)) is not int or not 0 <= usage[k] <= 10**9 for k in fields)
                    or usage["total_tokens"] != usage["prompt_tokens"] + usage["completion_tokens"]):
                raise Denied("usage unavailable; stop")
            self.result["usage"] = {k: usage[k] for k in fields}
            self.result["output_cap_observed"] = usage["completion_tokens"] <= 256
            if not self.result["output_cap_observed"]:
                raise Denied("output usage exceeds requested cap")
            self.result["provider_model"] = body.get("model") if isinstance(body.get("model"), str) and len(body["model"]) < 128 else None
            self.result["provider_response_id"] = body.get("id") if isinstance(body.get("id"), str) and re.fullmatch(r"[\w-]{1,128}", body["id"]) else None
            choice = body["choices"][0]
            message = choice["message"]
            text = message.get("content") or ""
            if not isinstance(text, str) or len(text.encode()) > 4096 or message.get("tool_calls"):
                raise Denied("only bounded final text is supported")
            self.result.update(status="observed", final_text=text, finish_reason=choice.get("finish_reason") if
                               choice.get("finish_reason") in ("stop", "length", "content_filter") else None)
            self.remaining(end)
            with self.lock:
                self.record("observed-stop-after-one")
            # Deliberately omit provider reasoning/private headers from exports
            # and the client envelope. This is not a native tool/lifecycle run.
            chunk = dict(id=self.result["provider_response_id"] or "p1-envelope", object="chat.completion.chunk",
                         created=0, model=MODEL, choices=[dict(index=0, delta=dict(role="assistant", content=text), finish_reason=None)])
            finish = dict(id=chunk["id"], object="chat.completion.chunk", created=0, model=MODEL,
                          choices=[dict(index=0, delta={}, finish_reason="stop")], usage=self.result["usage"])
            return ("data: " + json.dumps(chunk) + "\n\ndata: " + json.dumps(finish) + "\n\ndata: [DONE]\n\n").encode()
        except (ValueError, OSError, KeyError, IndexError, TypeError, http.client.HTTPException):
            with self.lock:
                self.result["status"] = "incomplete"
                self.stopped = True
                self.record("uncertain-or-failed-stop-no-refund")
            raise Denied("P1 stopped; inspect private curated observation, never retry") from None

    def snapshot(self):
        with self.lock:
            return dict(attempts=self.attempts, result=self.result, real_upstream=self.real)

    def cancel(self):
        with self.lock:
            self.stopped = True
            self.record("cancelled-no-refund")

    def exchange(self, payload, headers, end):
        if not self.real and not self.fake_address:
            raise Denied("no fake upstream configured")
        # A killable courier bounds DNS as well as connect/read time. A timed-out
        # thread must never wake later and send an unobserved extra request.
        env = {"P1_JOURNAL": str(self.journal), "PYTHONNOUSERSITE": "1"}
        env["P1_REAL_ALLOCATION" if self.real else "P1_FAKE_COURIER"] = ALLOCATION if self.real else "1"
        packet = json.dumps(dict(payload=payload.decode(), headers=headers, end=end,
                                 fake_address=self.fake_address if not self.real else None)).encode()
        try:
            response = subprocess.run([sys.executable, "-I", "-B", str(Path(__file__).resolve()), "--courier"],
                                      input=packet, capture_output=True, env=env, timeout=self.remaining(end))
        except subprocess.TimeoutExpired:
            raise Denied("courier killed at deadline; never retry") from None
        if response.returncode:
            raise Denied("courier failed; never retry")
        observed = decode(response.stdout)
        return observed["status"], observed["body"].encode()


def courier():
    real = os.environ.get("P1_REAL_ALLOCATION") == ALLOCATION
    if not real and os.environ.get("P1_FAKE_COURIER") != "1":
        raise Denied("courier not authorized")
    journal = Path(os.environ["P1_JOURNAL"])
    if real and str(journal) != "/control/allocation.jsonl":
        raise Denied("wrong real allocation")
    if (any(p.is_symlink() for p in [journal, *journal.parents]) or
            journal.stat().st_mode & 0o077 or journal.stat().st_size > 65536):
        raise Denied("invalid allocation source")
    allocation = decode(journal.read_text().splitlines()[-1])
    if (allocation["phase"] != "reserved-before-network" or allocation["attempts"] != 1 or
            allocation["real_upstream"] is not real or allocation["allocation"] != ALLOCATION):
        raise Denied("no spent reservation")
    packet = decode(sys.stdin.buffer.read(8193))
    payload = packet["payload"].encode()
    expected = dict(model=MODEL, messages=[dict(role="user", content=PROMPT)], max_tokens=256, stream=False)
    if len(payload) > 4096 or decode(payload) != expected:
        raise Denied("noncanonical payload")
    headers = packet["headers"]
    if (set(headers) != {"User-Agent", "Content-Type"} or headers["Content-Type"] != "application/json" or
            not isinstance(headers["User-Agent"], str) or len(headers["User-Agent"]) > 256 or
            any(ord(c) < 32 or ord(c) > 126 for c in headers["User-Agent"])):
        raise Denied("unapproved headers")
    end = packet["end"]
    def remaining():
        value = end - time.monotonic()
        if not 0 < value <= 90:
            raise Denied("courier expired")
        return value
    remaining()
    # A second courier/restart cannot reuse the reservation, even if the first
    # had an uncertain DNS/TLS/read outcome. Never remove this ticket to retry.
    fd = os.open(journal.parent / "dispatch.ticket", os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w") as stream:
        stream.write(ALLOCATION + "\n"); stream.flush(); os.fsync(stream.fileno())
    if real:
        if packet["fake_address"] is not None:
            raise Denied("real route cannot be overridden")
        connection = http.client.HTTPSConnection(HOST, 443, timeout=remaining(), context=ssl.create_default_context())
    else:
        address = packet["fake_address"]
        if not isinstance(address, list) or len(address) != 2 or address[0] != "127.0.0.1" or type(address[1]) is not int:
            raise Denied("fake courier requires literal loopback")
        connection = http.client.HTTPConnection(*address, timeout=remaining())
    try:
        connection.request("POST", UPSTREAM_PATH, body=payload, headers=headers)
        wire_socket = connection.sock
        wire_socket.settimeout(remaining())
        response = connection.getresponse()
        if response.status != 200:
            return dict(status=response.status, body="")
        if response.getheader("Content-Type", "").split(";", 1)[0] != "application/json":
            raise Denied("unexpected response type")
        chunks, size = [], 0
        while not response.isclosed():
            wire_socket.settimeout(remaining())
            chunk = response.read1(min(4096, 131073 - size))
            if not chunk:
                break
            size += len(chunk)
            if size > 131072:
                raise Denied("bounded response exceeded")
            chunks.append(chunk)
        remaining()
        return dict(status=response.status, body=b"".join(chunks).decode())
    finally:
        connection.close()


def server(pilot):
    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_args):
            pass
        def reply(self, code, data, kind="application/json"):
            self.close_connection = True
            self.send_response(code)
            self.send_header("Content-Type", kind)
            self.send_header("Content-Length", str(len(data)))
            self.end_headers(); self.wfile.write(data)
        def do_GET(self):
            data = json.dumps(dict(ready=True, attempts=pilot.snapshot()["attempts"], denials=pilot.denials)).encode()
            self.reply(200 if self.path == "/ready" else 403, data if self.path == "/ready" else b'{"denied":true}')
        def send_error(self, *_args, **_kwargs):
            self.reply(403, b'{"denied":true}')
        def do_POST(self):
            try:
                self.connection.settimeout(min(3, pilot.remaining(pilot.deadline)))
                length = self.headers.get_all("Content-Length", [])
                if (self.path != PATH or len(length) != 1 or not length[0].isascii() or not length[0].isdecimal() or
                        not 0 < int(length[0]) <= 4096 or self.headers.get("Transfer-Encoding") or self.headers.get("Upgrade") or
                        self.headers.get("Content-Type") != "application/json"):
                    raise Denied("unsupported framing/path")
                raw = self.rfile.read(int(length[0]))
                if len(raw) != int(length[0]):
                    raise Denied("partial body")
                result = pilot.process(raw, self.headers)
                self.reply(200, result, "text/event-stream")
            except (ValueError, OSError) as error:
                reason = str(error) if isinstance(error, Denied) else "transport-error"
                with pilot.lock:
                    if len(pilot.denials) < 64:
                        pilot.denials.append(reason)
                try:
                    self.reply(403, b'{"denied":true}')
                except OSError:
                    pass
    result = ThreadingHTTPServer(("0.0.0.0", 8000), Handler)
    result.daemon_threads = True
    return result


def main():
    if os.getuid() == 0:
        raise ValueError("non-root gate required")
    real = os.environ.get("P1_MODE") == "live"
    if os.environ.get("P1_MODE") not in ("fake", "live"):
        raise ValueError("explicit fake/live mode required")
    fake = None
    if not real:
        class Fake(BaseHTTPRequestHandler):
            def do_POST(self):
                payload = decode(self.rfile.read(int(self.headers["Content-Length"])))
                if (self.path != UPSTREAM_PATH or set(payload) != {"model", "messages", "max_tokens", "stream"} or
                        self.headers.get("Authorization") or self.headers.get("X-SDD-P1")):
                    self.send_error(400); return
                data = json.dumps(dict(id="fake-p1-response", model=MODEL, choices=[dict(message=dict(content="P1_OK"), finish_reason="stop")],
                                       usage=dict(prompt_tokens=5, completion_tokens=3, total_tokens=8))).encode()
                self.send_response(200); self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(data))); self.end_headers(); self.wfile.write(data)
            def log_message(self, *_args):
                pass
        fake = ThreadingHTTPServer(("127.0.0.1", 0), Fake)
        threading.Thread(target=fake.serve_forever, daemon=True).start()
    pilot = Pilot(Path("/control/allocation.jsonl"), os.environ["P1_GATE_TOKEN"],
                  deadline=float(os.environ["P1_DEADLINE"]), real=real,
                  fake_address=("127.0.0.1", fake.server_port) if fake else None)
    server(pilot).serve_forever()


if __name__ == "__main__":
    if sys.argv[1:] == ["--courier"]:
        try:
            print(json.dumps(courier()))
        except (ValueError, OSError, KeyError, TypeError, http.client.HTTPException):
            print('{"error":"courier stopped without retry"}')
            raise SystemExit(2)
    else:
        main()
