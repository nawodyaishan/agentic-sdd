"""Owned deterministic check interruption, NOT native agent recovery/compaction."""
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[2]


@unittest.skipUnless(sys.platform == "linux", "runner execution is Linux-only; native recovery not tested")
class RecoveryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp.cleanup)
        cls.root = Path(cls.temp.name)
        cls.binary = cls.root / "sdd-check"
        # A fresh credential-free child environment, not a cloned native HOME/DB.
        cls.env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(cls.root / "home"),
                   "TMPDIR": str(cls.root), "GOCACHE": str(cls.root / "go-cache"), "CGO_ENABLED": "0",
                   "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off", "GOENV": "off"}
        (cls.root / "home").mkdir(mode=0o700)
        subprocess.run([shutil.which("go"), "build", "-o", str(cls.binary), "./cmd/sdd-check"],
                       cwd=ROOT, env=cls.env, check=True, capture_output=True, timeout=120)

    def exercise(self, interrupt):
        with tempfile.TemporaryDirectory(dir=self.root) as tmp:
            root = Path(tmp)
            workspace = root / "project"
            workspace.mkdir()
            partial, forbidden = workspace / "partial.txt", workspace / "must-not-run.txt"
            (workspace / "tasks.md").write_text("fixture authorization stops at B1\n")
            commands = [
                ["sh", "-c", 'trap "exit 130" TERM; while :; do printf tick >> "$1"; sleep 0.05; done', "fixture", str(partial)],
                ["sh", "-c", 'printf forbidden > "$1"', "fixture", str(forbidden)],
            ]
            checks = [dict(id="first" if i == 0 else "later", cwd=".", scopes=["required"], prerequisites=["sh"],
                           timeout_seconds=10, effects=dict(writes=["synthetic scratch only"], network="none"),
                           steps=[dict(argv=argv, env={})], local_recipe=argv, expected_exit=0, evidence=f"{i}.log")
                      for i, argv in enumerate(commands)]
            contract = root / "checks.json"
            contract.write_text(json.dumps(dict(version=1, checks=checks)))
            output = root / "evidence"
            unrelated = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"], env=self.env)
            process = None
            try:
                process = subprocess.Popen([str(self.binary), "--scope", "required", "--workspace", str(workspace),
                                            "--contract", str(contract), "--output", str(output),
                                            "--deadline", "10s" if interrupt else "400ms"],
                                           env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                deadline = time.monotonic() + 5
                while not partial.exists() and process.poll() is None and time.monotonic() < deadline:
                    time.sleep(0.01)
                self.assertTrue(partial.exists(), "fixture must reach the writing process before interruption")
                if interrupt:
                    process.send_signal(signal.SIGINT)
                stdout, stderr = process.communicate(timeout=10)
                self.assertEqual(1, process.returncode, stderr)
                result = json.loads(stdout)
                self.assertEqual("cancelled" if interrupt else "incomplete", result["execution"])
                self.assertEqual(result["execution"], result["checks"][0]["status"])
                self.assertEqual("not_run", result["checks"][1]["status"])
                self.assertIsNone(result["checks"][1]["exit_code"])
                self.assertFalse(forbidden.exists(), "no new check is scheduled after interruption")
                self.assertIsNone(unrelated.poll(), "unrelated fixture process must survive")
                count = partial.stat().st_size
                time.sleep(0.2)
                self.assertEqual(count, partial.stat().st_size, "owned writer must stop")
                self.assertGreater(count, 0, "partial changes survive interruption")
                self.assertEqual("fixture authorization stops at B1\n", (workspace / "tasks.md").read_text())
                saved = json.loads((output / "result.json").read_text())
                self.assertEqual(result, saved, "persisted/reloaded result must not become a pass")
                self.assertNotIn("review", saved, "runner must not manufacture Ready")
                self.assertNotIn("authorization_ref", saved, "runner must not manufacture authority")
                for entry in output.iterdir():
                    self.assertEqual(0, entry.stat().st_mode & 0o077)
            finally:
                if process is not None and process.poll() is None:
                    # Only fixture-owned runner: let its signal handler stop owned children.
                    process.send_signal(signal.SIGTERM)
                    process.communicate(timeout=10)
                unrelated.terminate()
                unrelated.wait(timeout=5)

    def test_owned_cancellation_preserves_partial_files_and_unrelated_process(self):
        self.exercise(interrupt=True)

    def test_deadline_persists_incomplete_and_never_replays_remaining_check(self):
        self.exercise(interrupt=False)


if __name__ == "__main__":
    unittest.main()
