"""Core selection tests use a fake Go recorder, never native/model fixtures."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "tests/supported-core-go.sh"
PACKAGES = ["./internal/skillsync", "./internal/harnesscheck", "./internal/harnessevidence", "./cmd/sdd-check"]
CLI_FILES = ["cmd/agentic-sdd/main.go", "cmd/agentic-sdd/logs.go", "cmd/agentic-sdd/opencode.go", "cmd/agentic-sdd/main_test.go"]


class SupportedCoreSelectionTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(SCRIPT.is_file(), "missing explicit supported-core Go selection")
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name) / "checkout with spaces"
        (self.root / "tests").mkdir(parents=True)
        shutil.copyfile(SCRIPT, self.root / "tests/supported-core-go.sh")
        shutil.copyfile(ROOT / "Taskfile.yml", self.root / "Taskfile.yml")
        for name in CLI_FILES + ["cmd/agentic-sdd/opencode_test.go"]:
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("// fixture path only; fake Go never compiles this\n")
        self.capture = Path(temp.name) / "capture.jsonl"
        bin_dir = Path(temp.name) / "bin"
        bin_dir.mkdir()
        fake_go = bin_dir / "go"
        fake_go.write_text(
            f"#!{sys.executable}\n"
            "import json, os, sys\n"
            "keys = ['GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','GOENV','GOWORK','GOCACHE']\n"
            "with open(os.environ['CAPTURE'], 'a') as f:\n"
            " f.write(json.dumps({'argv':sys.argv[1:], 'env':{k:os.environ.get(k) for k in keys}})+'\\n')\n"
            "key = 'FAIL_CLI' if sys.argv[-1].endswith('main_test.go') else 'FAIL_PACKAGES'\n"
            "sys.exit(int(os.environ.get(key, '0')))\n"
        )
        fake_go.chmod(0o755)
        self.env = {
            "PATH": str(bin_dir) + os.pathsep + os.environ["PATH"],
            "HOME": temp.name,
            "CAPTURE": str(self.capture),
            "GOCACHE": str(Path(temp.name) / "cache with spaces"),
            "GOTOOLCHAIN": "auto",
            "GOPROXY": "https://invalid.example",
            "GOSUMDB": "example",
            "GOFLAGS": "-run=NoTestsSelected",
            "GOENV": "/unreviewed/environment",
            "GOWORK": "/unreviewed/go.work",
        }

    def run_profile(self, *args, **env):
        return subprocess.run(
            ["sh", "tests/supported-core-go.sh", *args], cwd=self.root,
            env=self.env | env, text=True, capture_output=True,
        )

    def observations(self):
        return [json.loads(line) for line in self.capture.read_text().splitlines()] if self.capture.exists() else []

    def test_exact_test_and_vet_selection(self):
        for mode in ("test", "vet"):
            with self.subTest(mode=mode):
                self.capture.unlink(missing_ok=True)
                result = self.run_profile(mode)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual([row["argv"] for row in self.observations()], [[mode, *PACKAGES], [mode, *CLI_FILES]])

    def test_offline_environment_and_inherited_filters_cleared(self):
        result = self.run_profile("test")
        self.assertEqual(result.returncode, 0, result.stderr)
        for row in self.observations():
            self.assertEqual(row["env"], {
                "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
                "GOFLAGS": "", "GOENV": "off", "GOWORK": "off", "GOCACHE": self.env["GOCACHE"],
            })

    def test_package_failure_stops_cli(self):
        result = self.run_profile("test", FAIL_PACKAGES="7")
        self.assertEqual(result.returncode, 7)
        self.assertEqual(len(self.observations()), 1)

    def test_cli_failure_is_not_hidden(self):
        result = self.run_profile("vet", FAIL_CLI="9")
        self.assertEqual(result.returncode, 9)
        self.assertEqual(len(self.observations()), 2)

    def test_invalid_mode_or_extra_args_do_not_run_go(self):
        for args in ((), ("all",), ("test", "./...")):
            with self.subTest(args=args):
                result = self.run_profile(*args)
                self.assertEqual(result.returncode, 2)
                self.assertEqual(self.observations(), [])

    def test_missing_or_new_cli_file_requires_selection_review(self):
        missing = self.root / CLI_FILES[0]
        original = missing.read_text()
        missing.unlink()
        result = self.run_profile("test")
        self.assertEqual(result.returncode, 2)
        self.assertEqual(self.observations(), [])
        missing.write_text(original)
        (missing.parent / "new_test.go").write_text("// unreviewed test\n")
        result = self.run_profile("test")
        self.assertEqual(result.returncode, 2)
        self.assertIn("unreviewed CLI Go file", result.stderr)
        self.assertEqual(self.observations(), [])

    def test_task_entry_points_forward_cache(self):
        task = shutil.which("task")
        self.assertIsNotNone(task, "Go Task is required for entry-point verification; do not count absence as pass")
        for mode in ("test", "vet"):
            with self.subTest(mode=mode):
                self.capture.unlink(missing_ok=True)
                cache = str(self.root / "selected cache")
                result = subprocess.run(
                    [task, mode + "-core", "GOCACHE=" + cache], cwd=self.root,
                    env=self.env, text=True, capture_output=True,
                )
                self.assertEqual(result.returncode, 0, result.stderr)
                rows = self.observations()
                self.assertEqual(len(rows), 2)
                self.assertTrue(all(row["env"]["GOCACHE"] == cache for row in rows))

    def test_aggregate_wiring_with_fake_go_is_not_a_scoped_alias(self):
        task = shutil.which("task")
        self.assertIsNotNone(task)

        def assert_aggregate(mode):
            self.capture.unlink(missing_ok=True)
            result = subprocess.run([task, mode], cwd=self.root, env=self.env, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual([row["argv"] for row in self.observations()], [[mode, "./..."]])

        # These calls reach only the fake Go recorder, never aggregate Go tests.
        for mode in ("test", "vet"):
            assert_aggregate(mode)
        fixture = self.root / "Taskfile.yml"
        text = fixture.read_text()
        self.assertIn("go test ./...", text)
        fixture.write_text(text.replace("go test ./...", "go test ./internal/skillsync", 1))
        with self.assertRaises(AssertionError):
            assert_aggregate("test")


class AggregateBoundaryTests(unittest.TestCase):
    def test_required_contract_and_aggregate_recipes_are_not_narrowed(self):
        contract = json.loads((ROOT / "harness/checks.json").read_text())
        required = {check["id"]: check for check in contract["checks"] if "required" in check["scopes"]}
        self.assertEqual(set(required), {"mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"})
        self.assertEqual(required["test"]["steps"][0]["argv"], ["go", "test", "./..."])
        self.assertEqual(required["vet"]["steps"][0]["argv"], ["go", "vet", "./..."])
        taskfile = (ROOT / "Taskfile.yml").read_text()
        self.assertIn("go test ./...", taskfile)
        self.assertIn("go vet ./...", taskfile)
        self.assertIn("test-core:", taskfile)
        self.assertIn("vet-core:", taskfile)


if __name__ == "__main__":
    unittest.main()
