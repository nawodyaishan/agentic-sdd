"""S0 offline-only byte-quota reference; never a live inference gateway."""
import json
import http.client
import math
import os
from pathlib import Path
import stat
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit


PATH = "/v1/chat/completions"
ACCOUNTING = "utf8-bytes-offline"


class Denied(ValueError):
    pass


class Budget:
    def __init__(self, journal, *, model, max_requests, per_input, per_output, total_input, total_output,
                 seconds, clock=time.monotonic, accounting=ACCOUNTING):
        quotas = (max_requests, per_input, per_output, total_input, total_output)
        if (accounting != ACCOUNTING or not isinstance(model, str) or not model or
                any(type(x) is not int or x <= 0 for x in quotas) or
                type(seconds) not in (int, float) or not math.isfinite(seconds) or seconds <= 0):
            raise ValueError("explicit offline byte limits required")
        self.journal = Path(journal).absolute()
        if ".." in self.journal.parts or any(p.is_symlink() for p in [self.journal, *self.journal.parents]):
            raise ValueError("journal symlink/traversal refused")
        if self.journal.parent.stat().st_mode & 0o077:
            raise ValueError("private journal parent required")
        self.model, self.max_requests = model, max_requests
        self.per_input, self.per_output = per_input, per_output
        self.total_input, self.total_output = total_input, total_output
        self.clock, self.deadline = clock, clock() + seconds
        self.lock = threading.Lock()
        self.requests = self.input_bytes = self.output_reserved = 0
        self.stopped = False
        fd = os.open(self.journal, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        info = os.fstat(fd)
        self.file_identity = (info.st_dev, info.st_ino)
        os.close(fd)
        self._record()

    def _record(self):
        try:
            fd = os.open(self.journal, os.O_WRONLY | os.O_APPEND | os.O_NOFOLLOW)
            with os.fdopen(fd, "a") as stream:
                info = os.fstat(stream.fileno())
                if (not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077 or
                        (info.st_dev, info.st_ino) != self.file_identity):
                    raise OSError("journal identity changed")
                stream.write(json.dumps(self._observation(), sort_keys=True) + "\n")
                stream.flush()
                os.fsync(stream.fileno())
        except OSError:
            self.stopped = True
            raise Denied("journal unavailable; no forwarding") from None

    def _observation(self):
        return dict(accounting=ACCOUNTING, offline_only=True, requests_reserved=self.requests,
                    input_bytes=self.input_bytes, output_bytes_reserved=self.output_reserved, stopped=self.stopped,
                    stop_reason=getattr(self, "stop_reason", None))

    def remaining(self):
        with self.lock:
            remaining = self.deadline - self.clock()
            if self.stopped or not math.isfinite(remaining) or remaining <= 0:
                raise Denied("allocation stopped or expired")
            return remaining

    def admit(self, raw, *, method="POST", path=PATH):
        def unique(pairs):
            result = {}
            for key, value in pairs:
                if key in result:
                    raise Denied("duplicate JSON key")
                result[key] = value
            return result
        def nonfinite(_value):
            raise Denied("nonfinite JSON")
        if method != "POST" or path != PATH or not isinstance(raw, bytes) or len(raw) > self.per_input:
            raise Denied("unsupported method/path/body size")
        try:
            data = json.loads(raw, object_pairs_hook=unique, parse_constant=nonfinite)
            allowed = {"model", "messages", "max_tokens", "max_completion_tokens", "stream", "stream_options", "tools", "tool_choice",
                       "temperature", "top_p", "stop", "parallel_tool_calls"}
            if not isinstance(data, dict) or set(data) - allowed or data.get("model") != self.model:
                raise Denied("unsupported model or request fields")
            caps = [key for key in ("max_tokens", "max_completion_tokens") if key in data]
            if len(caps) != 1:
                raise Denied("exactly one output cap required")
            output = data[caps[0]]
            if type(output) is not int or not 0 < output <= self.per_output:
                raise Denied("explicit bounded output required")
            if type(data.get("stream", False)) is not bool or not isinstance(data.get("messages"), list) or not data["messages"]:
                raise Denied("unsupported stream/messages")
            for message in data["messages"]:
                if not isinstance(message, dict) or message.get("role") not in ("system", "developer", "user", "assistant", "tool"):
                    raise Denied("unsupported message")
                content = message.get("content")
                if isinstance(content, list):
                    if any(not isinstance(item, dict) or set(item) != {"type", "text"} or
                           item["type"] != "text" or not isinstance(item["text"], str) for item in content):
                        raise Denied("only inline text is supported")
                elif not isinstance(content, str):
                    raise Denied("only inline text is supported")
        except (ValueError, TypeError, KeyError, RecursionError):
            raise Denied("invalid or unsupported request") from None
        with self.lock:
            if self.stopped or not math.isfinite(self.clock()) or self.clock() >= self.deadline:
                raise Denied("allocation stopped or expired")
            if (self.requests >= self.max_requests or self.input_bytes + len(raw) > self.total_input or
                    self.output_reserved + output > self.total_output):
                raise Denied("allocation exhausted")
            self.requests += 1
            self.input_bytes += len(raw)
            self.output_reserved += output
            self._record()  # Commit before any upstream dispatch; never refund.
            return dict(input_bytes=len(raw), output_bytes_reserved=output)

    def observation(self):
        with self.lock:
            return self._observation()

    def fail(self, reason):
        # Do not persist arbitrary exception strings, headers, prompts or keys.
        with self.lock:
            self.stopped = True
            self.stop_reason = reason if reason in ("Denied", "OSError", "TimeoutError", "IncompleteRead", "RemoteDisconnected", "cancelled") else "stopped"
            self._record()


def upstream_address(url):
    parsed = urlsplit(url)
    port = parsed.port
    if (parsed.scheme != "http" or parsed.hostname != "127.0.0.1" or not port or not 0 < port < 65536 or
            url != f"http://127.0.0.1:{port}{PATH}"):
        raise ValueError("only a literal loopback fake endpoint is supported")
    return "127.0.0.1", port


def relay_server(budget, upstream):
    if os.environ.get("SDD_S0_CONTAINER") != "1" or os.getuid() == 0:
        raise ValueError("use the unprivileged network-none fixture wrapper")
    host, port = upstream_address(upstream)

    class Relay(BaseHTTPRequestHandler):
        # No request URL, exception body, header or prompt is ever logged.
        def log_message(self, *_args):
            pass

        def reject(self):
            payload = b'{"error":"offline gate denied"}'
            self.close_connection = True
            self.send_response(403)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def send_error(self, _code, message=None, explain=None):
            self.reject()

        def do_POST(self):
            connection = None
            dispatched = False
            try:
                self.connection.settimeout(min(2, budget.remaining()))
                lengths = self.headers.get_all("Content-Length", [])
                if (self.path != PATH or len(lengths) != 1 or not lengths[0].isascii() or
                        not lengths[0].isdecimal() or not 0 < int(lengths[0]) <= budget.per_input or
                        self.headers.get("Transfer-Encoding") or self.headers.get("Upgrade") or
                        self.headers.get("Content-Type") != "application/json"):
                    raise Denied("unsupported HTTP request")
                raw = self.rfile.read(int(lengths[0]))
                if len(raw) != int(lengths[0]):
                    raise Denied("partial HTTP body")
                reservation = budget.admit(raw, method="POST", path=self.path)
                connection = http.client.HTTPConnection(host, port, timeout=min(2, budget.remaining()))
                with budget.lock:
                    # Cancellation takes this lock too: an already admitted
                    # send may finish before cancellation, never be refunded.
                    if budget.stopped or budget.clock() >= budget.deadline:
                        raise Denied("stopped before send")
                    dispatched = True
                    connection.request("POST", PATH, body=raw, headers={"Content-Type": "application/json"})
                connection.sock.settimeout(min(2, budget.remaining()))
                wire_socket = connection.sock
                response = connection.getresponse()
                content_type = response.getheader("Content-Type", "").split(";", 1)[0]
                if response.status != 200 or content_type not in ("application/json", "text/event-stream"):
                    raise Denied("upstream rejected; do not retry")
                limit = reservation["output_bytes_reserved"]
                chunks, size = [], 0
                while True:
                    remaining = budget.remaining()
                    if response.isclosed():
                        break
                    # Retain the socket: HTTPResponse can take ownership after
                    # Connection: close. Its file still keeps the descriptor.
                    wire_socket.settimeout(min(2, remaining))
                    chunk = response.read1(min(4096, limit + 1 - size))
                    if not chunk:
                        break
                    chunks.append(chunk)
                    size += len(chunk)
                    if size > limit:
                        raise Denied("response exceeds reservation")
                budget.remaining()  # Also withhold a response after cancellation.
                self.close_connection = True
                self.send_response(200)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(size))
                self.end_headers()
                self.wfile.write(b"".join(chunks))
            except (ValueError, OSError, http.client.HTTPException) as error:
                if dispatched:
                    try:
                        budget.fail(type(error).__name__)
                    except Denied:
                        pass  # Persistence uncertainty is already fail-closed.
                try:
                    self.reject()
                except OSError:
                    pass
            finally:
                if connection is not None:
                    connection.close()

        do_GET = do_CONNECT = do_PUT = do_PATCH = do_DELETE = do_OPTIONS = lambda self: self.reject()

    server = ThreadingHTTPServer(("127.0.0.1", 0), Relay)
    server.daemon_threads = True
    return server
