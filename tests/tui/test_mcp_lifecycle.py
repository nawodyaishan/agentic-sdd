"""MCP Tools PTY lifecycle, navigation and resize checks (Feature 015 T42).

Feature 015 / MCP-D. This harness starts the already-built nested TUI under a
real PTY with an isolated ``HOME``/``TMPDIR`` and drives the shipped key map:

* launch -> ``2`` (Install & Manage hub) -> ``t`` (MCP Tools) renders the
  Overview with exactly three default rows: CodeGraph, Context7 and Exa;
* ``j``/``k`` move the Overview cursor, ``s``/Enter opens server selection,
  ``v`` opens the read-only configuration preview, ``d`` opens diagnostics,
  and Esc walks back Select -> Overview -> hub without writing;
* the Select -> Preview -> Confirm path keeps ``No`` selected by default; a
  narrow resize during confirmation shows the tiny-geometry guard and cannot
  dispatch a write;
* ``q`` quits cleanly, leaves the alternate screen and restores the line
  discipline;
* the run is read-only: the isolated home gains no ``.agentic-sdd/mcp``
  backup/manifest tree, the real home string never reaches the screen and the
  real ``~/.config/opencode`` entry set is unchanged. No model, credential,
  network or package-install path is exercised.

Prerequisites are explicit and never satisfied by skipping:

* POSIX ``pty``/``termios``/``select``/``SIGWINCH`` (Linux or macOS);
* a writable, existing ``/tmp/opencode`` (override with
  ``AGENTIC_SDD_TUI_PTY_TMPDIR``);
* ``AGENTIC_SDD_TUI_TEST_BINARY`` pointing at a built nested TUI binary.

Any missing prerequisite raises an actionable ``BLOCKED`` error, so the test
process exits non-zero instead of reporting a silent all-skipped pass.

Run explicitly, never through broad discovery::

    AGENTIC_SDD_TUI_TEST_BINARY=/tmp/opencode/tui-spike \\
        python3 -B -m unittest discover -s tests/tui -p test_mcp_lifecycle.py

The opt-in ``tui:mcp-pty`` task builds the binary into an approved temporary
directory and supplies the variable; see ``Taskfile.experimental.yml``.
"""

import os
import select
import signal
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
ENV_ASCII = "AGENTIC_SDD_ASCII"
ENV_NO_UPDATE_CHECK = "AGENTIC_SDD_NO_UPDATE_CHECK"

# Approved temporary base. The harness refuses to run anywhere it cannot prove
# it is isolated from the user's real home.
TEMP_BASE = os.environ.get("AGENTIC_SDD_TUI_PTY_TMPDIR", "/tmp/opencode")

ALT_SCREEN_ENTER = b"\x1b[?1049h"
ALT_SCREEN_LEAVE = b"\x1b[?1049l"

# Geometry sequence from T42: the AC breakpoints, then the smallest usable
# terminal where only guidance fits.
FULL_SIZE = (32, 100)
MEDIUM_SIZE = (10, 60)
TINY_SIZE = (8, 40)

STARTUP_TIMEOUT = 20.0
KEY_TIMEOUT = 10.0
SHUTDOWN_TIMEOUT = 20.0

HUB_READY = b"[t] MCP Tools"
MCP_OVERVIEW_READY = b"Supported MCP tools (3 defaults)"
MCP_SERVERS = (b"CodeGraph", b"Context7", b"Exa")
MCP_SELECT_READY = b"Select Servers"
MCP_PREVIEW_READY = b"Configuration Preview"
MCP_CONFIRM_READY = b"Confirm Changes"
MCP_DIAGNOSTICS_READY = b"Contextual configuration, prerequisite, and offline readiness checks"
MCP_TINY_GUARD = b"Terminal geometry too small"
# Affirmative-action vocabulary that must never be reachable while a confirm
# view only has room for its "too small" guard.
AFFIRMATIVE_MARKERS = (b"[  Yes, apply changes  ]",)
PANIC_MARKERS = (b"panic:", b"goroutine ")
# Color SGR forms. Bubble Tea still sends terminal-mode sequences such as
# ``CSI > 4 m``; only palette/truecolor declarations prove a colored surface.
COLOR_MARKERS = (b"\x1b[38;", b"\x1b[48;", b"\x1b[31m", b"\x1b[32m", b"\x1b[33m")

REAL_HOME = os.path.expanduser("~")

BINARY = ""


def setUpModule():
    """Validate the opt-in harness prerequisites once for the whole module."""

    global BINARY
    if os.name != "posix" or pty is None or termios is None or fcntl is None:
        raise RuntimeError(
            "BLOCKED: the MCP PTY harness needs POSIX pty/termios "
            "(Linux or macOS); this host cannot run it. This is not a pass."
        )
    if not hasattr(signal, "SIGWINCH"):
        raise RuntimeError(
            "BLOCKED: this platform has no SIGWINCH, so the resize cases "
            "cannot be driven against a real PTY. This is not a pass."
        )
    if not os.path.isdir(TEMP_BASE) or not os.access(TEMP_BASE, os.W_OK):
        raise RuntimeError(
            "BLOCKED: %r must exist and be writable so every home and TMPDIR "
            "stays isolated. Create it or set AGENTIC_SDD_TUI_PTY_TMPDIR. "
            "This is not a pass." % TEMP_BASE
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
            "-p test_mcp_lifecycle.py\n"
            "or run the opt-in task: task -t Taskfile.experimental.yml "
            "tui:mcp-pty\n"
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


def _set_winsize(fd, rows, cols):
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))


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

    def __init__(self, args, env, rows, cols):
        self.master, slave = pty.openpty()
        try:
            _set_winsize(slave, rows, cols)
            self.initial_attributes = termios.tcgetattr(slave)
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
        self.final_attributes = None
        self._closed = False

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        self.close()
        return False

    def mark(self):
        """Return an output offset so a later assertion only sees new bytes."""

        return len(self.output)

    def since(self, mark):
        return bytes(self.output[mark:])

    def pump_until(self, predicate, timeout):
        """Read until predicate(all output), EOF or the deadline."""

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

    def send(self, data):
        if self.process.poll() is not None:
            raise AssertionError(
                "cannot send %r: the child already exited with %r; output so "
                "far: %r"
                % (data, self.process.poll(), bytes(self.output[-2000:]))
            )
        os.write(self.master, data)

    def resize(self, rows, cols):
        """Resize the PTY and make the child observe it.

        The child is not a session leader on this slave, so the kernel has no
        foreground process group to signal. Delivering SIGWINCH explicitly is
        what a real terminal emulator does; Bubble Tea then re-reads the size
        from its own tty.
        """

        _set_winsize(self.master, rows, cols)
        if self.process.poll() is not None:
            raise AssertionError(
                "cannot resize: the child already exited with %r; output so "
                "far: %r"
                % (self.process.poll(), bytes(self.output[-2000:]))
            )
        os.kill(self.process.pid, signal.SIGWINCH)

    def finish(self, timeout=SHUTDOWN_TIMEOUT):
        """Wait for EOF and child exit, then record the restored attributes."""

        if not self.pump_until_eof(timeout):
            self.close()
            raise AssertionError(
                "child did not exit within %.1fs; output so far: %r"
                % (timeout, bytes(self.output[-2000:]))
            )
        try:
            self.exit_code = self.process.wait(timeout=5.0)
        except subprocess.TimeoutExpired:
            self.close()
            raise AssertionError(
                "child closed its terminal but did not exit; output so far: %r"
                % bytes(self.output[-2000:])
            )
        self.final_attributes = termios.tcgetattr(self.master)

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


def _session_env(home, tmpdir):
    """Build a minimal, controlled environment for one MCP PTY run.

    A fresh mapping intentionally drops every inherited variable, including
    credentials, proxies and developer overrides such as
    ``AGENTIC_SDD_FORCE_INTERACTIVE``, so the shell cannot change what this
    harness measures and no model/network input is available to the child.
    """

    return {
        "HOME": home,
        "PATH": os.environ.get("PATH", "/usr/local/bin:/usr/bin:/bin"),
        "TERM": "xterm-256color",
        "LANG": "C.UTF-8",
        "LC_ALL": "C.UTF-8",
        "TMPDIR": tmpdir,
        # Plain/ASCII/no-color is part of AC-33; the assertions below require
        # that no palette or truecolor escape and no Unicode glyph reaches the
        # terminal.
        "NO_COLOR": "1",
        ENV_ASCII: "1",
        # The first Home entry starts the once-per-session update check. MCP
        # inspection must not contact GitHub, so this opts out explicitly.
        ENV_NO_UPDATE_CHECK: "1",
        ENV_TUI: "1",
    }


def _path_fingerprint(path):
    """Describe a path's shallow tree so a write outside the sandbox is visible."""

    try:
        info = os.stat(path)
    except FileNotFoundError:
        return ("absent",)
    except OSError as exc:  # pragma: no cover - exotic filesystem state
        return ("error", exc.errno)
    entries = ()
    if os.path.isdir(path):
        try:
            entries = tuple(
                sorted((name, int(os.stat(os.path.join(path, name)).st_mtime_ns)) for name in os.listdir(path))
            )
        except OSError:  # pragma: no cover - race with another process
            entries = (("<unreadable>", 0),)
    return ("present", int(info.st_mtime_ns), entries)


def _mcp_relevant_real_paths():
    return (
        os.path.join(REAL_HOME, ".config", "opencode"),
        os.path.join(REAL_HOME, ".agentic-sdd"),
    )


class MCPLifecycleTest(unittest.TestCase):
    def _workdir(self):
        work = tempfile.TemporaryDirectory(
            prefix="agentic-sdd-tui-mcp-pty-", dir=TEMP_BASE
        )
        self.addCleanup(work.cleanup)
        home = os.path.join(work.name, "home")
        tmpdir = os.path.join(work.name, "tmp")
        os.makedirs(home, mode=0o700)
        os.makedirs(tmpdir, mode=0o700)
        self.assertNotEqual(
            os.path.realpath(home),
            os.path.realpath(REAL_HOME),
            "the isolated home must never be the real home",
        )
        return home, tmpdir

    def _seed_existing_config(self, home):
        """Create a minimal existing global config so Preview is applicable.

        v1 never creates a missing configuration, so an empty home correctly
        produces a target-missing conflict and cannot reach Confirm. This is a
        fixture for the confirmation path, not a bypass of that refusal.
        """
        config_dir = os.path.join(home, ".config", "opencode")
        os.makedirs(config_dir, mode=0o700, exist_ok=True)
        with open(os.path.join(config_dir, "opencode.json"), "w", encoding="utf-8") as fh:
            fh.write('{"mcp":{"servers":{}}}\n')

    def _assert_terminal_restored(self, session):
        final = session.final_attributes
        self.assertIsNotNone(
            final, "terminal attributes were never sampled after child exit"
        )
        lflag = final[3]
        self.assertTrue(
            lflag & termios.ICANON,
            "canonical input was not restored after exit; lflag=%#x" % lflag,
        )
        self.assertTrue(
            lflag & termios.ECHO,
            "local echo was not restored after exit; lflag=%#x" % lflag,
        )
        self.assertTrue(
            lflag & termios.ISIG,
            "signal generation was not restored after exit; lflag=%#x" % lflag,
        )
        self.assertEqual(
            session.initial_attributes,
            final,
            "terminal attributes after exit differ from the pre-launch state",
        )

    def _assert_no_panic(self, output, context):
        for marker in PANIC_MARKERS:
            self.assertNotIn(
                marker,
                output,
                "%s: the TUI reported a crash (%r); output tail: %r"
                % (context, marker, output[-2000:]),
            )

    def _assert_alt_screen_intact(self, session, context):
        output = bytes(session.output)
        self.assertIsNone(
            session.process.poll(),
            "%s: the TUI exited early with %r; output tail: %r"
            % (context, session.process.poll(), output[-2000:]),
        )
        self.assertNotIn(
            ALT_SCREEN_LEAVE,
            output,
            "%s: terminal teardown started while the app was still running; "
            "output tail: %r" % (context, output[-2000:]),
        )
        self.assertEqual(
            output.count(ALT_SCREEN_ENTER),
            1,
            "%s: the alternate screen was entered more than once, which means "
            "a corrupted or restarted surface; output tail: %r"
            % (context, output[-2000:]),
        )

    def _assert_no_mcp_writes(self, home, context):
        mcp_root = os.path.join(home, ".agentic-sdd", "mcp")
        self.assertFalse(
            os.path.exists(mcp_root),
            "%s: the read-only MCP inspection created %r; a preview-only run "
            "must write no backup or manifest" % (context, mcp_root),
        )

    def _open_mcp(self, session):
        """Launch -> hub -> MCP Tools, asserting each transition."""

        self.assertTrue(
            session.pump_until(
                lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
            ),
            "the interactive TUI never entered the alternate screen; output "
            "so far: %r" % bytes(session.output[-2000:]),
        )
        session.send(b"2")
        self.assertTrue(
            session.pump_until(lambda out: HUB_READY in out, KEY_TIMEOUT),
            "the Install & Manage hub never advertised the MCP Tools "
            "entry; output so far: %r" % bytes(session.output[-3000:]),
        )
        session.send(b"t")
        self.assertTrue(
            session.pump_until(
                lambda out: MCP_OVERVIEW_READY in out, KEY_TIMEOUT
            ),
            "t did not open the MCP Overview; output so far: %r"
            % bytes(session.output[-3000:]),
        )

    def test_mcp_overview_lists_three_defaults_and_writes_nothing(self):
        home, tmpdir = self._workdir()
        real_before = tuple(
            (_path_fingerprint(path), path) for path in _mcp_relevant_real_paths()
        )
        env = _session_env(home, tmpdir)

        with _PtySession(["--home", home], env, *FULL_SIZE) as session:
            self._open_mcp(session)
            output = bytes(session.output)

            # The Overview names exactly the three default servers.
            for expected in MCP_SERVERS:
                self.assertIn(
                    expected,
                    output,
                    "the MCP Overview is missing %r; output tail: %r"
                    % (expected, output[-3000:]),
                )
            for expected in (
                b"MCP Tools",
                b"Overview",
                b"connection not tested",
            ):
                self.assertIn(
                    expected,
                    output,
                    "the MCP Overview is missing %r; output tail: %r"
                    % (expected, output[-3000:]),
                )

            # Plain/ASCII/no-color (AC-33) and no accidental real-home path.
            for marker in COLOR_MARKERS:
                self.assertNotIn(
                    marker,
                    output,
                    "NO_COLOR=1 still emitted a color escape (%r); output "
                    "tail: %r" % (marker, output[-2000:]),
                )
            self.assertNotIn(
                b"\xc2\xb7",
                output,
                "AGENTIC_SDD_ASCII=1 still rendered the middle-dot glyph",
            )
            self.assertNotIn(
                os.fsencode(os.path.realpath(REAL_HOME)),
                output,
                "the session printed the real home path, so it is not fully "
                "scoped to the isolated home",
            )
            self._assert_alt_screen_intact(session, "overview")
            self._assert_no_panic(output, "overview")
            self._assert_no_mcp_writes(home, "overview")

            session.send(b"q")
            session.finish(SHUTDOWN_TIMEOUT)

        completed = bytes(session.output)
        self.assertEqual(
            session.exit_code,
            0,
            "q should quit cleanly; got exit %r, output tail: %r"
            % (session.exit_code, completed[-2000:]),
        )
        self.assertIn(
            ALT_SCREEN_LEAVE,
            completed,
            "quitting did not leave the alternate screen; tail: %r"
            % completed[-2000:],
        )
        self.assertLess(
            completed.index(ALT_SCREEN_ENTER),
            completed.rindex(ALT_SCREEN_LEAVE),
            "alternate-screen enter/leave sequences are out of order",
        )
        self._assert_terminal_restored(session)

        for before, path in real_before:
            self.assertEqual(
                before,
                _path_fingerprint(path),
                "the session changed the real %r tree; a PTY run must stay in "
                "its isolated home" % path,
            )

    def test_mcp_select_preview_diagnostics_and_back(self):
        home, tmpdir = self._workdir()
        env = _session_env(home, tmpdir)

        with _PtySession(["--home", home], env, *FULL_SIZE) as session:
            self._open_mcp(session)

            # Overview cursor navigation must keep the session alive.
            session.send(b"j")
            session.send(b"j")
            self.assertTrue(
                session.pump_until(
                    lambda out: b"Exa" in out, KEY_TIMEOUT
                ),
                "j navigation lost the MCP Overview; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            session.send(b"k")
            self._assert_alt_screen_intact(session, "overview j/k")
            self.assertIsNone(
                session.process.poll(),
                "cursor navigation quit the application",
            )

            # Enter the server-selection step.
            mark = session.mark()
            session.send(b"s")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_SELECT_READY in session.since(mark), KEY_TIMEOUT
                ),
                "s did not open server selection; output tail: %r"
                % bytes(session.output[-3000:]),
            )

            # Navigate the selection list, then acknowledge the Exa keyless
            # limits (option index 5) so Enter can reach a read-only preview.
            for _ in range(5):
                session.send(b"j")
            session.send(b" ")
            mark = session.mark()
            session.send(b"\r")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_PREVIEW_READY in session.since(mark), KEY_TIMEOUT
                ),
                "Enter did not open the configuration preview; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            self.assertTrue(
                session.pump_until(
                    lambda out: b"Plan identity" in out, KEY_TIMEOUT
                ),
                "the preview never rendered its plan identity; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            self._assert_alt_screen_intact(session, "preview")
            self._assert_no_mcp_writes(home, "preview")

            # Esc walks back Preview -> Select -> Overview.
            mark = session.mark()
            session.send(b"\x1b")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_SELECT_READY in session.since(mark), KEY_TIMEOUT
                ),
                "Esc did not return to server selection; output tail: %r"
                % bytes(session.output[-2000:]),
            )
            mark = session.mark()
            session.send(b"\x1b")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_OVERVIEW_READY in session.since(mark), KEY_TIMEOUT
                ),
                "Esc did not return to the MCP Overview; output tail: %r"
                % bytes(session.output[-2000:]),
            )

            # Diagnostics is reachable with d and reports no live connection.
            mark = session.mark()
            session.send(b"d")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_DIAGNOSTICS_READY in out, KEY_TIMEOUT
                )
                and MCP_DIAGNOSTICS_READY in session.since(mark),
                "d did not open MCP diagnostics; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            self.assertIn(
                b"remote-connectivity",
                bytes(session.output),
                "diagnostics must report the runtime connection check; "
                "output tail: %r" % bytes(session.output[-2000:]),
            )
            self.assertIn(
                b"not checked",
                bytes(session.output),
                "diagnostics must label the runtime connection as not checked; "
                "output tail: %r" % bytes(session.output[-2000:]),
            )
            mark = session.mark()
            session.send(b"\x1b")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_OVERVIEW_READY in session.since(mark), KEY_TIMEOUT
                ),
                "Esc did not return from diagnostics; output tail: %r"
                % bytes(session.output[-2000:]),
            )

            self._assert_no_mcp_writes(home, "navigation")
            session.send(b"q")
            session.finish(SHUTDOWN_TIMEOUT)

        self.assertEqual(
            session.exit_code,
            0,
            "q after the navigation sequence should exit cleanly; got %r, "
            "output tail: %r"
            % (session.exit_code, bytes(session.output[-2000:])),
        )
        self.assertIn(ALT_SCREEN_LEAVE, bytes(session.output))
        self._assert_terminal_restored(session)

    def test_mcp_resize_cannot_confirm_and_back_stays_reachable(self):
        home, tmpdir = self._workdir()
        self._seed_existing_config(home)
        env = _session_env(home, tmpdir)

        with _PtySession(["--home", home], env, *FULL_SIZE) as session:
            self._open_mcp(session)

            # Reach the default-No confirmation through selection.
            session.send(b"s")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_SELECT_READY in out, KEY_TIMEOUT
                ),
                "s did not open server selection; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            for _ in range(5):
                session.send(b"j")
            session.send(b" ")
            mark = session.mark()
            session.send(b"\r")
            self.assertTrue(
                session.pump_until(
                    lambda out: MCP_PREVIEW_READY in session.since(mark), KEY_TIMEOUT
                ),
                "Enter did not open the configuration preview; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            mark = session.mark()
            session.send(b"\r")
            self.assertTrue(
                session.pump_until(
                    lambda out: b"No, cancel" in session.since(mark), KEY_TIMEOUT
                ),
                "Enter did not open the default-No confirmation; output tail: %r"
                % bytes(session.output[-3000:]),
            )
            self.assertIn(
                b"No, cancel",
                bytes(session.output),
                "the confirmation must default to No; output tail: %r"
                % bytes(session.output[-2000:]),
            )

            # A narrow resize during confirmation shows the guard and keeps
            # No selected: Enter must not dispatch a write from tiny geometry.
            session.resize(*TINY_SIZE)
            mark = session.mark()
            tiny = session.pump_until(
                lambda out: MCP_TINY_GUARD in out[mark:], KEY_TIMEOUT
            )
            self.assertTrue(
                tiny,
                "40x8 did not show the tiny-geometry guard on the confirm "
                "view; fresh output: %r" % session.since(mark)[-2000:],
            )
            self.assertNotIn(
                AFFIRMATIVE_MARKERS[0],
                session.since(mark),
                "tiny geometry exposed an affirmative action that cannot be "
                "seen in full; fresh output: %r" % session.since(mark)[-2000:],
            )
            session.send(b"\r")
            time.sleep(0.5)
            self.assertIsNone(
                session.process.poll(),
                "Enter from tiny geometry must not confirm or crash; the app "
                "exited instead",
            )
            self._assert_alt_screen_intact(session, "tiny confirm")
            self._assert_no_mcp_writes(home, "tiny confirm")

            # Esc cancels back to Preview from the smallest geometry.
            mark = session.mark()
            session.send(b"\x1b")
            returned = session.pump_until(
                lambda out: MCP_PREVIEW_READY in out[mark:], KEY_TIMEOUT
            )
            self.assertTrue(
                returned,
                "Esc did not cancel back to Preview from tiny geometry; fresh "
                "output: %r" % session.since(mark)[-2000:],
            )

            # The full surface still renders after the round trip.
            session.resize(*FULL_SIZE)
            mark = session.mark()
            restored = session.pump_until(
                lambda out: MCP_PREVIEW_READY in out[mark:], KEY_TIMEOUT
            )
            self.assertTrue(
                restored,
                "the preview did not repaint after returning to 100x32; fresh "
                "output: %r" % session.since(mark)[-2000:],
            )
            self._assert_alt_screen_intact(session, "restored 100x32")

            session.send(b"q")
            session.finish(SHUTDOWN_TIMEOUT)

        self.assertEqual(
            session.exit_code,
            0,
            "q after the resize sequence should exit cleanly; got %r, output "
            "tail: %r" % (session.exit_code, bytes(session.output[-2000:])),
        )
        self.assertIn(ALT_SCREEN_LEAVE, bytes(session.output))
        self._assert_terminal_restored(session)
        self._assert_no_mcp_writes(home, "resize")


if __name__ == "__main__":
    unittest.main()
