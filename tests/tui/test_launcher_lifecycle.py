"""Parent-child terminal checks for ``agentic-sdd tui`` (Feature 013 B6/T14).

The root CLI verifies the sibling ``agentic-sdd-tui`` and then starts it. This
harness launches the *root CLI* under a real PTY and measures what the pair
actually does:

* explicit ``agentic-sdd tui`` and the Homebrew zero-argument default both enter
  the alternate screen and quit cleanly;
* SIGINT, SIGTERM and SIGHUP sent to the **parent** are relayed to the child,
  which restores the terminal and exits; the parent does not die first and
  orphan the child;
* a signal during an in-flight write is deferred by the child until the write
  and log flush complete, and the parent keeps waiting;
* the parent forwards ``--home``/``--repo`` and propagates the child's exit
  code;
* a non-interactive launch keeps the historical preview and never opens the
  alternate screen.

Prerequisites are explicit and never satisfied by skipping:

* POSIX ``pty``/``termios``/``select`` (Linux or macOS);
* ``AGENTIC_SDD_TUI_PAIR_DIR`` pointing at a matched ``agentic-sdd`` +
  ``agentic-sdd-tui`` pair built with the same injected identity, for example::

      task -t Taskfile.experimental.yml tui:pair

  ``task -t Taskfile.experimental.yml tui:pty-test`` builds the pair and runs
  this file explicitly. Any missing prerequisite raises an actionable
  ``BLOCKED`` error, so the process exits non-zero instead of silently skipping.

Run explicitly, never through broad discovery::

    AGENTIC_SDD_TUI_PAIR_DIR=/repo/bin/tui-pair \\
        python3 -B -m unittest discover -s tests/tui -p test_launcher_lifecycle.py
"""

import os
import select
import shutil
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

ENV_PAIR_DIR = "AGENTIC_SDD_TUI_PAIR_DIR"
ENV_TEST_HOLD_WRITE = "AGENTIC_SDD_TUI_TEST_HOLD_WRITE"
ENV_FORCE_INTERACTIVE = "AGENTIC_SDD_FORCE_INTERACTIVE"
ENV_NO_UPDATE_CHECK = "AGENTIC_SDD_NO_UPDATE_CHECK"

ALT_SCREEN_ENTER = b"\x1b[?1049h"
ALT_SCREEN_LEAVE = b"\x1b[?1049l"

# The busy footer's stable ASCII marker. It is the root model's own status
# text, not a test-only screen, so the harness can wait for a genuine in-flight
# write.
WRITE_BUSY_MARKER = b"WRITE-IN-PROGRESS"
WRITE_HOLD = "3s"
WRITE_HOLD_CHECK = 0.25

TERMINAL_ROWS = 32
TERMINAL_COLUMNS = 100
STARTUP_TIMEOUT = 20.0
SHUTDOWN_TIMEOUT = 20.0

CLI_BINARY = ""
COMPANION_BINARY = ""


def setUpModule():
    """Validate the opt-in harness prerequisites once for the whole module."""

    global CLI_BINARY, COMPANION_BINARY
    if os.name != "posix" or pty is None or termios is None or fcntl is None:
        raise RuntimeError(
            "BLOCKED: the launcher PTY harness needs POSIX pty/termios "
            "(Linux or macOS); this host cannot run it. This is not a pass."
        )
    CLI_BINARY, COMPANION_BINARY = _required_pair()


def _required_pair():
    raw = os.environ.get(ENV_PAIR_DIR, "").strip()
    if not raw:
        raise RuntimeError(
            "BLOCKED: %s is not set. Build a matched pair first:\n"
            "  task -t Taskfile.experimental.yml tui:pair\n"
            "  %s=<repo>/bin/tui-pair python3 -B -m unittest discover "
            "-s tests/tui -p test_launcher_lifecycle.py\n"
            "or run the opt-in task: task -t Taskfile.experimental.yml "
            "tui:pty-test\n"
            "A missing prerequisite is blocked, not skipped." % (ENV_PAIR_DIR, ENV_PAIR_DIR)
        )

    directory = os.path.abspath(raw)
    if not os.path.isdir(directory):
        raise RuntimeError(
            "BLOCKED: %s points at %r, which is not a directory. Rebuild the "
            "pair and retry." % (ENV_PAIR_DIR, directory)
        )
    pair = []
    for name in ("agentic-sdd", "agentic-sdd-tui"):
        path = os.path.join(directory, name)
        if not os.path.isfile(path):
            raise RuntimeError(
                "BLOCKED: %s is missing from %r. Rebuild the pair with "
                "task -t Taskfile.experimental.yml tui:pair." % (name, directory)
            )
        if not os.access(path, os.X_OK):
            raise RuntimeError(
                "BLOCKED: %r is not executable. Rebuild the pair with "
                "task -t Taskfile.experimental.yml tui:pair." % path
            )
        pair.append(path)
    return pair[0], pair[1]


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
    """One parent process on its own PTY, with bounded observation."""

    def __init__(self, args, env, stdin_terminal=True, binary=None):
        self.master, slave = pty.openpty()
        try:
            fcntl.ioctl(
                slave,
                termios.TIOCSWINSZ,
                struct.pack("HHHH", TERMINAL_ROWS, TERMINAL_COLUMNS, 0, 0),
            )
            self.initial_attributes = termios.tcgetattr(slave)
            self.process = subprocess.Popen(
                [binary or CLI_BINARY] + list(args),
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
                "cannot send %r: the parent already exited with %r; output so "
                "far: %r"
                % (data, self.process.poll(), bytes(self.output[-2000:]))
            )
        os.write(self.master, data)

    def send_signal_to_parent(self, sig):
        """Deliver a real OS signal to the root CLI, not to the companion."""

        if self.process.poll() is not None:
            raise AssertionError(
                "cannot send %r: the parent already exited with %r; output so "
                "far: %r"
                % (sig, self.process.poll(), bytes(self.output[-2000:]))
            )
        os.kill(self.process.pid, sig)

    def finish(self, timeout=SHUTDOWN_TIMEOUT):
        """Wait for EOF and parent exit, then record the restored attributes."""

        if not self.pump_until_eof(timeout):
            self.close()
            raise AssertionError(
                "the parent did not exit within %.1fs; output so far: %r"
                % (timeout, bytes(self.output[-2000:]))
            )
        try:
            self.exit_code = self.process.wait(timeout=5.0)
        except subprocess.TimeoutExpired:
            self.close()
            raise AssertionError(
                "the parent closed its terminal but did not exit; output so "
                "far: %r" % bytes(self.output[-2000:])
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
    """Build a minimal, controlled environment for one launcher run.

    A fresh mapping drops inherited controls such as AGENTIC_SDD_FORCE_INTERACTIVE
    so a developer's shell cannot change what this harness measures. HOME stays
    inside the temporary directory, so the Homebrew zero-argument default can
    resolve a home without ever reading the real one.
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
        # The first Home entry starts the once-per-session update check. This
        # harness measures terminal lifecycle, so it opts out explicitly and
        # never contacts GitHub.
        ENV_NO_UPDATE_CHECK: "1",
    }


class LauncherLifecycleTest(unittest.TestCase):
    def _assert_terminal_restored(self, session):
        final = session.final_attributes
        self.assertIsNotNone(
            final, "terminal attributes were never sampled after the parent exited"
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

    def _assert_alt_screen_cycle(self, session, expected_exit=0):
        output = bytes(session.output)
        self.assertEqual(
            session.exit_code,
            expected_exit,
            "exit %r, want %r; output tail: %r"
            % (session.exit_code, expected_exit, output[-2000:]),
        )
        self.assertIn(
            ALT_SCREEN_ENTER,
            output,
            "the child never entered the alternate screen; output tail: %r"
            % output[-2000:],
        )
        self.assertIn(
            ALT_SCREEN_LEAVE,
            output,
            "shutdown did not leave the alternate screen; tail: %r" % output[-2000:],
        )
        self.assertLess(
            output.index(ALT_SCREEN_ENTER),
            output.rindex(ALT_SCREEN_LEAVE),
            "alternate-screen enter/leave sequences are out of order",
        )
        self._assert_terminal_restored(session)

    def test_explicit_tui_launches_the_verified_companion(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            with _PtySession(["tui", "--home", home], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    entered,
                    "agentic-sdd tui never started the companion; output so "
                    "far: %r" % bytes(session.output[-2000:]),
                )
                self.assertNotIn(
                    b"terminal UI unavailable",
                    session.output,
                    "the handshake failed; output so far: %r"
                    % bytes(session.output[-2000:]),
                )
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)
        self._assert_alt_screen_cycle(session)

    def test_missing_sibling_fails_closed_without_preview(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-alone-") as root:
            home = os.path.join(root, "home")
            alone_dir = os.path.join(root, "alone")
            os.makedirs(home)
            os.makedirs(alone_dir)
            alone = os.path.join(alone_dir, os.path.basename(CLI_BINARY))
            shutil.copy2(CLI_BINARY, alone)
            os.chmod(alone, 0o755)
            env = _session_env(home)
            with _PtySession(
                ["tui", "--home", home], env, binary=alone
            ) as session:
                session.finish(STARTUP_TIMEOUT)
        output = bytes(session.output)
        self.assertEqual(session.exit_code, 1, output[-2000:])
        self.assertNotIn(ALT_SCREEN_ENTER, output)
        self.assertIn(b"is not installed beside", output)
        self.assertIn(b"agentic-sdd preview", output)
        self.assertNotIn(
            b"Preview only.",
            output,
            "a missing companion silently ran preview; output: %r"
            % output[-2000:],
        )
        self._assert_terminal_restored(session)

    def test_homebrew_default_launches_with_zero_arguments(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            with _PtySession([], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                if not entered:
                    self.fail(
                        "a bare Homebrew-marked agentic-sdd did not open the "
                        "TUI. Build the pair with the default Homebrew marker: "
                        "task -t Taskfile.experimental.yml tui:pair. Output so "
                        "far: %r" % bytes(session.output[-2000:])
                    )
                session.send(b"q")
                session.finish(SHUTDOWN_TIMEOUT)
        self._assert_alt_screen_cycle(session)

    def test_nonterminal_stdin_keeps_the_historical_preview(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            with _PtySession([], env, stdin_terminal=False) as session:
                session.finish(STARTUP_TIMEOUT)
        output = bytes(session.output)
        self.assertEqual(session.exit_code, 0, output[-2000:])
        self.assertNotIn(ALT_SCREEN_ENTER, output)
        self.assertIn(
            b"Preview only.",
            output,
            "the non-interactive default did not keep the historical preview; "
            "output: %r" % output[-2000:],
        )
        self._assert_terminal_restored(session)

    def test_nonterminal_explicit_tui_fails_closed(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            with _PtySession(
                ["tui", "--home", home], env, stdin_terminal=False
            ) as session:
                session.finish(STARTUP_TIMEOUT)
        output = bytes(session.output)
        self.assertEqual(session.exit_code, 2, output[-2000:])
        self.assertNotIn(ALT_SCREEN_ENTER, output)
        self.assertIn(b"needs a real input and output terminal", output)
        self.assertNotIn(b"Preview only.", output)
        self._assert_terminal_restored(session)

    def test_parent_propagates_child_failure_and_forwards_repo(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            missing_repo = os.path.join(home, "missing-skills-source")
            with _PtySession(
                ["tui", "--home", home, "--repo", missing_repo], env
            ) as session:
                session.finish(STARTUP_TIMEOUT)
        output = bytes(session.output)
        self.assertEqual(
            session.exit_code,
            1,
            "the parent must propagate the companion's failure; output: %r"
            % output[-2000:],
        )
        self.assertNotIn(ALT_SCREEN_ENTER, output)
        # Without a forwarded --repo the embedded assets would load and the
        # alternate screen would open. A catalog error proves the child
        # received the missing repository path.
        self.assertIn(
            b"inspect selection",
            output,
            "the companion never received the forwarded --repo; output: %r"
            % output[-2000:],
        )
        self.assertNotIn(
            b"stdin is not a terminal",
            output,
            "the child stopped at its own terminal gate instead of the "
            "forwarded repository; output: %r" % output[-2000:],
        )
        self.assertNotIn(
            b"terminal UI unavailable",
            output,
            "the handshake failed before the argument-forwarding check: %r"
            % output[-2000:],
        )
        self._assert_terminal_restored(session)

    # --- Parent -> child signal relay --------------------------------------

    def _run_idle_signal_case(self, sig):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            with _PtySession(["tui", "--home", home], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    entered,
                    "agentic-sdd tui never started the companion; output so "
                    "far: %r" % bytes(session.output[-2000:]),
                )
                session.send_signal_to_parent(sig)
                session.finish(SHUTDOWN_TIMEOUT)
        self._assert_alt_screen_cycle(session)

    def _run_inflight_signal_case(self, sig, repeats=1):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            env[ENV_TEST_HOLD_WRITE] = WRITE_HOLD
            with _PtySession(["tui", "--home", home], env) as session:
                entered = session.pump_until(
                    lambda out: ALT_SCREEN_ENTER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    entered,
                    "agentic-sdd tui never started the companion; output so "
                    "far: %r" % bytes(session.output[-2000:]),
                )
                busy = session.pump_until(
                    lambda out: WRITE_BUSY_MARKER in out, STARTUP_TIMEOUT
                )
                self.assertTrue(
                    busy,
                    "the lifecycle write never showed its in-progress marker; "
                    "output so far: %r" % bytes(session.output[-2000:]),
                )
                for _ in range(repeats):
                    session.send_signal_to_parent(sig)
                    time.sleep(0.1)
                time.sleep(WRITE_HOLD_CHECK)
                self.assertIsNone(
                    session.process.poll(),
                    "%r killed the parent or aborted the child's in-flight "
                    "write; the parent must wait for the child's deferred "
                    "exit. Output tail: %r"
                    % (sig, bytes(session.output[-2000:])),
                )
                self.assertNotIn(
                    ALT_SCREEN_LEAVE,
                    bytes(session.output),
                    "terminal teardown started before the write completed; "
                    "output tail: %r" % bytes(session.output[-2000:]),
                )
                session.finish(SHUTDOWN_TIMEOUT)
        self._assert_alt_screen_cycle(session)

    def test_parent_relays_sigint_and_child_restores_terminal(self):
        self._run_idle_signal_case(signal.SIGINT)

    def test_parent_relays_sigterm_and_child_restores_terminal(self):
        self._run_idle_signal_case(signal.SIGTERM)

    def test_parent_relays_sighup_and_child_restores_terminal(self):
        self._run_idle_signal_case(signal.SIGHUP)

    def test_inflight_sigint_defers_exit_through_the_parent(self):
        self._run_inflight_signal_case(signal.SIGINT)

    def test_inflight_repeated_sigterm_defers_exit_through_the_parent(self):
        self._run_inflight_signal_case(signal.SIGTERM, repeats=3)

    def test_inflight_sighup_defers_exit_through_the_parent(self):
        self._run_inflight_signal_case(signal.SIGHUP)

    def test_forced_interactive_parent_refuses_to_launch(self):
        with tempfile.TemporaryDirectory(prefix="agentic-sdd-tui-pair-") as home:
            env = _session_env(home)
            env[ENV_FORCE_INTERACTIVE] = "1"
            with _PtySession(["tui", "--home", home], env) as session:
                session.finish(STARTUP_TIMEOUT)
        output = bytes(session.output)
        self.assertEqual(session.exit_code, 2, output[-2000:])
        self.assertNotIn(ALT_SCREEN_ENTER, output)
        self.assertIn(ENV_FORCE_INTERACTIVE.encode(), output)
        self._assert_terminal_restored(session)


if __name__ == "__main__":
    unittest.main()
