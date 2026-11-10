"""Black-box read-only checks for the experimental nested TUI.

Feature 013 / B2 lane. This harness starts the already-built
``experiments/tui`` binary under a real PTY and observes what the process
actually does while the user browses the catalog:

* the Explore screen lists the real shipped inventory, not a mock;
* the category control and the curated detail pane come from that inventory;
* Home offers the required setup action;
* a browsing session, including category and search interaction, writes
  nothing into the inspected home;
* the session ends cleanly when the user quits.

The harness is deliberately separate from ``test_terminal_lifecycle.py``:
that file asserts terminal attributes and exit codes, this file asserts what
the catalog shows and that the run stays read-only. Neither asserts a screen
snapshot, so a layout change cannot make them brittle. Because a terminal only
redraws changed cells, a check that needs a whole screen re-renders it first by
switching routes, then reads the visible text.

Prerequisites are explicit and never satisfied by skipping:

* POSIX ``pty``/``termios``/``select`` (Linux or macOS);
* ``AGENTIC_SDD_TUI_TEST_BINARY`` pointing at a built nested TUI binary.

Any missing prerequisite raises an actionable ``BLOCKED`` error, so the test
process exits non-zero instead of reporting a silent all-skipped pass.

Run explicitly, never through broad discovery::

    AGENTIC_SDD_TUI_TEST_BINARY=/path/to/tui-spike \\
        python3 -B -m unittest discover -s tests/tui -p test_readonly_catalog.py
"""

import os
import re
import select
import struct
import subprocess
import tempfile
import time
import unittest

try:  # POSIX-only modules; the module gate below reports a clear blocker.
    import fcntl
    import pty
    import termios
except ImportError:  # pragma: no cover - exercised only on non-POSIX hosts
    fcntl = None
    pty = None
    termios = None

ENV_TEST_BINARY = "AGENTIC_SDD_TUI_TEST_BINARY"
ENV_TUI = "AGENTIC_SDD_TUI"

ALT_SCREEN_ENTER = b"\x1b[?1049h"

# A wide, tall viewport keeps the catalog list column wide enough to show a
# complete shipped skill ID.
TERMINAL_ROWS = 40
TERMINAL_COLUMNS = 120
STARTUP_TIMEOUT = 15.0
SHUTDOWN_TIMEOUT = 15.0

# Markers that only a real, populated catalog produces: the per-row coverage
# summary (the narrow list column may shorten it to "clients in…") and an
# exact shipped identifier.
ROW_SUMMARY = b"clients in"
SHIPPED_ID_PREFIX = b"agentic-sdd-"

# The category control is a user-visible label with the count it derived from
# the inventory: "Category: All (33)".
CATEGORY_HEADING = re.compile(rb"Category: ([A-Za-z ]+?) \((\d+)\)")

# The curated detail pane, in reading order. The labels are the shared
# teaching vocabulary, not a pixel snapshot.
CURATED_DETAIL_MARKERS = (
    b"Purpose:",
    b"When:",
    b"Example:",
    b"Caveat:",
    b"Destination status:",
    b"Next action:",
)

# A fresh temporary home holds no installed destination, so every per-client
# status must be the truthful "not installed" state.
TRUTHFUL_MISSING_STATUS = b"Template available (not installed)"

# The setup action the spec requires Home to offer visibly.
HOME_SETUP_ACTION = b"Install Agentic-SDD"

BINARY = ""


def setUpModule():
    """Validate the opt-in harness prerequisites once for the whole module."""

    global BINARY
    if os.name != "posix" or pty is None or termios is None or fcntl is None:
        raise RuntimeError(
            "BLOCKED: the TUI read-only harness needs POSIX pty/termios "
            "(Linux or macOS); this host cannot run it. This is not a pass."
        )
    BINARY = _required_binary()


def _required_binary():
    raw = os.environ.get(ENV_TEST_BINARY, "").strip()
    if not raw:
        raise RuntimeError(
            "BLOCKED: %s is not set. Build the nested TUI and point this "
            "variable at it, for example:\n"
            "  (cd experiments/tui && GOTOOLCHAIN=local GOWORK=off go build "
            "-o /tmp/opencode/tui-spike ./cmd/tui-spike)\n"
            "  AGENTIC_SDD_TUI_TEST_BINARY=/tmp/opencode/tui-spike "
            "python3 -B -m unittest discover -s tests/tui "
            "-p test_readonly_catalog.py\n"
            "or run the opt-in task: task -t Taskfile.experimental.yml "
            "tui:pty-test\n"
            "A missing prerequisite is blocked, not skipped." % ENV_TEST_BINARY
        )

    path = os.path.abspath(raw)
    if not os.path.isfile(path):
        raise RuntimeError(
            "BLOCKED: %s points at %r, which is not a file. Rebuild the "
            "nested TUI binary and retry." % (ENV_TEST_BINARY, path)
        )
    if not os.access(path, os.X_OK):
        raise RuntimeError(
            "BLOCKED: %s points at %r, which is not executable. Rebuild the "
            "nested TUI binary and retry." % (ENV_TEST_BINARY, path)
        )
    return path


def _read_available(fd, timeout):
    """Read one chunk, returning None for a timeout and b"" for EOF/EIO."""

    try:
        ready, _, _ = select.select([fd], [], [], max(0.0, timeout))
    except InterruptedError:  # pragma: no cover - signal timing dependent
        return None
    if not ready:
        return None
    try:
        return os.read(fd, 65536)
    except OSError:
        # Linux reports EIO on the master once every slave descriptor is closed.
        return b""


class _PtySession:
    """One child process on its own PTY, with bounded observation."""

    def __init__(self, args, env):
        self.master, slave = pty.openpty()
        try:
            fcntl.ioctl(
                slave,
                termios.TIOCSWINSZ,
                struct.pack("HHHH", TERMINAL_ROWS, TERMINAL_COLUMNS, 0, 0),
            )
            self.process = subprocess.Popen(
                [BINARY] + list(args),
                stdin=slave,
                stdout=slave,
                stderr=slave,
                env=env,
                close_fds=True,
            )
        except BaseException:
            os.close(self.master)
            raise
        finally:
            os.close(slave)

        self.output = bytearray()
        self.eof = False
        self.exit_code = None
        self._closed = False

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        self.close()
        return False

    def pump_until(self, predicate, timeout):
        """Read until predicate(output), EOF or the deadline; report predicate."""

        deadline = time.monotonic() + timeout
        while True:
            if predicate(bytes(self.output)):
                return True
            if self.eof:
                return False
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                return False
            chunk = _read_available(self.master, min(0.05, remaining))
            if chunk is None:
                continue
            if not chunk:
                self.eof = True
                return False
            self.output.extend(chunk)

    def pump_until_eof(self, timeout):
        deadline = time.monotonic() + timeout
        while not self.eof:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                return False
            chunk = _read_available(self.master, min(0.05, remaining))
            if chunk is None:
                continue
            if not chunk:
                self.eof = True
                break
            self.output.extend(chunk)
        return True

    def pump_for(self, duration):
        """Read for a bounded period so the renderer can draw one frame.

        A terminal repaints only the cells that changed, so a test that needs
        a whole screen must let each intermediate frame render before sending
        the next key; otherwise the renderer coalesces them and the unchanged
        cells are never emitted again.
        """

        deadline = time.monotonic() + duration
        while not self.eof:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                return
            chunk = _read_available(self.master, min(0.05, remaining))
            if chunk is None:
                continue
            if not chunk:
                self.eof = True
                return
            self.output.extend(chunk)

    def send(self, data):
        if self.process.poll() is not None:
            raise AssertionError(
                "cannot send %r: the child already exited with %r; output so "
                "far: %r" % (data, self.process.poll(), bytes(self.output[-2000:]))
            )
        os.write(self.master, data)

    def finish(self, timeout=SHUTDOWN_TIMEOUT):
        """Wait for EOF and child exit."""

        if not self.pump_until_eof(timeout):
            self.close()
            raise AssertionError(
                "child did not exit within %.1fs; output so far: %r"
                % (timeout, bytes(self.output[-2000:]))
            )
        self.exit_code = self.process.wait(timeout=5.0)

    def close(self):
        if self._closed:
            return
        self._closed = True
        if self.process.poll() is None:
            self.process.kill()
            try:
                self.process.wait(timeout=5.0)
            except subprocess.TimeoutExpired:  # pragma: no cover
                pass
        try:
            os.close(self.master)
        except OSError:  # pragma: no cover
            pass


def _session_env(home):
    """Build a minimal, controlled environment for one TUI run.

    A fresh mapping intentionally drops inherited controls such as
    AGENTIC_SDD_FORCE_INTERACTIVE, so a developer's shell cannot change what
    this harness measures. Both HOME and --home stay inside the temporary
    directory.
    """

    return {
        "HOME": home,
        "PATH": os.environ.get("PATH", "/usr/local/bin:/usr/bin:/bin"),
        "TERM": "xterm-256color",
        "LANG": "C.UTF-8",
        "LC_ALL": "C.UTF-8",
        "TMPDIR": tempfile.gettempdir(),
        # Reduced color keeps captured bytes focused on catalog content.
        "NO_COLOR": "1",
        ENV_TUI: "1",
    }


def _tree_snapshot(root):
    """Record every file below root with its size."""

    tree = {}
    for directory, _dirs, files in os.walk(root):
        for name in files:
            path = os.path.join(directory, name)
            try:
                tree[os.path.relpath(path, root)] = os.path.getsize(path)
            except OSError:  # pragma: no cover - racing removal
                continue
    return tree


def _category_headings(output):
    """Return every rendered (label, count) category heading, in order.

    A terminal redraws only changed cells, so the same heading may appear more
    than once with different values; callers decide which occurrence matters.
    """

    return [
        (label.strip(), int(count))
        for label, count in CATEGORY_HEADING.findall(output)
    ]


class ReadOnlyCatalogTest(unittest.TestCase):
    def test_explore_lists_the_real_shipped_inventory(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-catalog-") as home:
            env = _session_env(home)
            with _PtySession(["--home", home], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    entered,
                    "the interactive TUI never entered the alternate screen; "
                    "output so far: %r" % bytes(session.output[-2000:]),
                )
                # "3" opens the Explore destination.
                session.send(b"3")
                listed = session.pump_until(
                    lambda out: ROW_SUMMARY in out, STARTUP_TIMEOUT
                )
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)

            output = bytes(session.output)
            self.assertEqual(
                session.exit_code,
                0,
                "quitting after browsing should exit cleanly; output tail: %r"
                % output[-2000:],
            )
            self.assertTrue(
                listed,
                "the Explore screen never rendered a catalog row with real "
                "destination coverage; the catalog is not reading the real "
                "inventory. Output tail: %r" % output[-2000:],
            )
            self.assertIn(
                SHIPPED_ID_PREFIX,
                output,
                "the Explore screen never showed an exact shipped skill ID; "
                "output tail: %r" % output[-2000:],
            )

    def test_explore_categories_and_curated_detail(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-catalog-") as home:
            env = _session_env(home)
            with _PtySession(["--home", home], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    entered,
                    "the interactive TUI never entered the alternate screen; "
                    "output so far: %r" % bytes(session.output[-2000:]),
                )
                self.assertTrue(
                    session.pump_until(
                        lambda out: HOME_SETUP_ACTION in out, STARTUP_TIMEOUT
                    ),
                    "Home never offered the required setup action; output so "
                    "far: %r" % bytes(session.output[-2000:]),
                )

                session.send(b"3")
                self.assertTrue(
                    session.pump_until(
                        lambda out: ROW_SUMMARY in out, STARTUP_TIMEOUT
                    ),
                    "Explore never rendered the real inventory; output so far: "
                    "%r" % bytes(session.output[-2000:]),
                )
                initial = _category_headings(bytes(session.output))
                self.assertTrue(
                    initial,
                    "Explore never showed the category control; output so far: "
                    "%r" % bytes(session.output[-2000:]),
                )
                self.assertEqual(
                    initial[-1][0].lower(),
                    b"all",
                    "Explore should start on the all category, saw %r"
                    % (initial[-1],),
                )
                self.assertGreater(
                    initial[-1][1],
                    0,
                    "the all category reports no items; the count is not "
                    "derived from the inventory",
                )

                # Cycle the category, then switch routes and back so the whole
                # screen is re-rendered instead of only the changed cells. Each
                # intermediate frame must render before the next key.
                session.send(b"]")
                session.pump_for(0.5)
                session.send(b"1")
                session.pump_for(0.5)
                session.send(b"3")
                switched = session.pump_until(
                    lambda out: any(
                        label.lower() != b"all"
                        for label, count in _category_headings(out)
                        if count > 0
                    ),
                    STARTUP_TIMEOUT,
                )
                # The curated detail card is longer than the initial viewport.
                # Page through it so every documented section is proven
                # reachable instead of assuming it fits on the first frame.
                session.send(b"\x1b[6~")
                session.pump_for(0.5)
                session.send(b"\x1b[6~")
                session.pump_for(0.5)
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)

            output = bytes(session.output)
            self.assertTrue(
                switched,
                "cycling the category never showed a different populated "
                "category; output tail: %r" % output[-2000:],
            )
            self.assertEqual(
                session.exit_code,
                0,
                "quitting after browsing categories should exit cleanly; "
                "output tail: %r" % output[-2000:],
            )
            for marker in CURATED_DETAIL_MARKERS:
                self.assertIn(
                    marker,
                    output,
                    "the detail pane never showed the curated %r section; the "
                    "metadata is not reaching the rendered catalog. Output "
                    "tail: %r" % (marker, output[-2000:]),
                )
            self.assertIn(
                TRUTHFUL_MISSING_STATUS,
                output,
                "the detail pane never reported the truthful missing "
                "destination status for a fresh home. Output tail: %r"
                % output[-2000:],
            )

    def test_browsing_session_writes_nothing_into_the_home(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-catalog-") as home:
            before = _tree_snapshot(home)
            env = _session_env(home)
            with _PtySession(["--home", home], env) as session:
                session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                # Explore, cycle a category, search, open the detail card,
                # scroll it, then quit.
                session.send(b"3")
                session.pump_until(
                    lambda out: ROW_SUMMARY in out, STARTUP_TIMEOUT
                )
                session.send(b"]")
                session.send(b"/")
                self.assertTrue(
                    session.pump_until(
                        lambda out: b"Filter:" in out, STARTUP_TIMEOUT
                    ),
                    "pressing / never opened the search input; output tail: "
                    "%r" % bytes(session.output[-2000:]),
                )
                for char in b"router":
                    session.send(bytes([char]))
                session.send(b"\r")  # accept the search
                session.send(b"\r")  # open the detail card
                session.send(b"\x1b[B")
                session.send(b" ")
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)

            after = _tree_snapshot(home)
            self.assertEqual(
                session.exit_code,
                0,
                "the browsing session should exit cleanly; output tail: %r"
                % bytes(session.output[-2000:]),
            )
            self.assertEqual(
                before,
                after,
                "a read-only browsing session changed the inspected home: "
                "before=%r after=%r" % (before, after),
            )


if __name__ == "__main__":
    unittest.main()
