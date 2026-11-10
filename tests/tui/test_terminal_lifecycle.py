"""Black-box terminal-lifecycle checks for the opt-in nested TUI.

Feature 013 / T01 baseline. This harness starts the already-built
``experiments/tui`` binary under a real PTY, drives it through its opt-in
environment, and observes what the terminal and child process actually do:

* a ``q`` quit exits successfully and leaves the alternate screen;
* the line discipline is restored (canonical input, echo and signals);
* an early error exits non-zero without entering the alternate screen or
  changing the terminal attributes.

The harness is deliberately navigation-independent: it never asserts a screen
title or catalog contents, so shell/layout changes cannot make it brittle.

Prerequisites are explicit and never satisfied by skipping:

* POSIX ``pty``/``termios``/``select`` (Linux or macOS);
* ``AGENTIC_SDD_TUI_TEST_BINARY`` pointing at a built nested TUI binary.

Any missing prerequisite raises an actionable ``BLOCKED`` error, so the test
process exits non-zero instead of reporting a silent all-skipped pass.

Run explicitly, never through broad discovery::

    AGENTIC_SDD_TUI_TEST_BINARY=/path/to/tui-spike \\
        python3 -B -m unittest discover -s tests/tui -p test_terminal_lifecycle.py

The opt-in Taskfile task builds the binary into an approved temporary
directory and supplies the variable; see ``Taskfile.experimental.yml``.
"""

import os
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
ENV_FORCE_INTERACTIVE = "AGENTIC_SDD_FORCE_INTERACTIVE"

ALT_SCREEN_ENTER = b"\x1b[?1049h"
ALT_SCREEN_LEAVE = b"\x1b[?1049l"

# A fixed, non-default viewport proves the size reaches the child without
# depending on any particular layout decision.
TERMINAL_ROWS = 32
TERMINAL_COLUMNS = 100
STARTUP_TIMEOUT = 15.0
SHUTDOWN_TIMEOUT = 15.0

BINARY = ""


def setUpModule():
    """Validate the opt-in harness prerequisites once for the whole module."""

    global BINARY
    if os.name != "posix" or pty is None or termios is None or fcntl is None:
        raise RuntimeError(
            "BLOCKED: the TUI PTY lifecycle harness needs POSIX pty/termios "
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
            "-p test_terminal_lifecycle.py\n"
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

    def __init__(self, args, env, stdin_terminal=True):
        self.master, slave = pty.openpty()
        try:
            fcntl.ioctl(
                slave,
                termios.TIOCSWINSZ,
                struct.pack("HHHH", TERMINAL_ROWS, TERMINAL_COLUMNS, 0, 0),
            )
            self.initial_attributes = termios.tcgetattr(slave)
            self.process = subprocess.Popen(
                [BINARY] + list(args),
                stdin=slave if stdin_terminal else subprocess.DEVNULL,
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

    def send(self, data):
        if self.process.poll() is not None:
            raise AssertionError(
                "cannot send %r: the child already exited with %r; output so "
                "far: %r"
                % (data, self.process.poll(), bytes(self.output[-2000:]))
            )
        os.write(self.master, data)

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
        # Reduced color keeps captured bytes focused on terminal lifecycle.
        "NO_COLOR": "1",
        ENV_TUI: "1",
    }


class TerminalLifecycleTest(unittest.TestCase):
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

    def test_quit_restores_terminal_and_leaves_alt_screen(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pty-") as home:
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
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)

        output = bytes(session.output)
        self.assertEqual(
            session.exit_code,
            0,
            "q should quit cleanly; got exit %r, output tail: %r"
            % (session.exit_code, output[-2000:]),
        )
        self.assertIn(
            ALT_SCREEN_LEAVE,
            output,
            "quitting did not leave the alternate screen; tail: %r"
            % output[-2000:],
        )
        self.assertLess(
            output.index(ALT_SCREEN_ENTER),
            output.rindex(ALT_SCREEN_LEAVE),
            "alternate-screen enter/leave sequences are out of order",
        )
        self._assert_terminal_restored(session)

    def test_early_error_exits_nonzero_without_touching_terminal(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pty-") as home:
            env = _session_env(home)
            missing_repo = os.path.join(home, "missing-skills-source")
            with _PtySession(
                ["--home", home, "--repo", missing_repo], env
            ) as session:
                session.finish(STARTUP_TIMEOUT)

        output = bytes(session.output)
        self.assertNotEqual(
            session.exit_code,
            0,
            "an unreadable skills source should fail the launch; output: %r"
            % output,
        )
        self.assertNotIn(
            ALT_SCREEN_ENTER,
            output,
            "an early launch error must not take over the terminal; output: %r"
            % output,
        )
        self.assertTrue(
            output.strip(),
            "an early launch error should explain itself on the PTY",
        )
        self._assert_terminal_restored(session)

    def test_nonterminal_input_does_not_enter_alternate_screen(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pty-") as home:
            with _PtySession(
                ["--home", home], _session_env(home), stdin_terminal=False
            ) as session:
                session.finish(STARTUP_TIMEOUT)
        self.assertEqual(session.exit_code, 2, bytes(session.output))
        self.assertNotIn(ALT_SCREEN_ENTER, session.output)
        self.assertIn(b"stdin is not a terminal", session.output)
        self._assert_terminal_restored(session)

    def test_forced_interactive_script_mode_does_not_enter_alternate_screen(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pty-") as home:
            env = _session_env(home)
            env[ENV_FORCE_INTERACTIVE] = "1"
            with _PtySession(["--home", home], env) as session:
                session.finish(STARTUP_TIMEOUT)
        self.assertEqual(session.exit_code, 2, bytes(session.output))
        self.assertNotIn(ALT_SCREEN_ENTER, session.output)
        self.assertIn(ENV_FORCE_INTERACTIVE.encode(), session.output)
        self._assert_terminal_restored(session)


if __name__ == "__main__":
    unittest.main()
