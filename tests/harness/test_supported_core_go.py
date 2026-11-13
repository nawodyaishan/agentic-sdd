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
CORE_PACKAGES = [
    ".",
    "./internal/version",
    "./internal/skillsync",
    "./internal/harnesscheck",
    "./internal/harnessevidence",
    "./internal/appops",
    "./internal/buildinfo",
    "./internal/fleet",
    "./internal/execution",
    "./cmd/agentic-sdd",
    "./cmd/sdd-check",
]


class SupportedCoreSelectionTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(SCRIPT.is_file(), "missing explicit supported-core Go selection")
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name) / "checkout with spaces"
        (self.root / "tests").mkdir(parents=True)
        shutil.copyfile(SCRIPT, self.root / "tests/supported-core-go.sh")
        shutil.copyfile(ROOT / "Taskfile.yml", self.root / "Taskfile.yml")
        (self.root / "go.mod").write_text("module agentic-sdd\n\ngo 1.22\n")
        for pkg_dir in [
            "internal/version",
            "internal/skillsync",
            "internal/harnesscheck",
            "internal/harnessevidence",
            "internal/appops",
            "internal/buildinfo",
            "internal/fleet",
            "internal/execution",
            "cmd/agentic-sdd",
            "cmd/sdd-check",
        ]:
            (self.root / pkg_dir).mkdir(parents=True, exist_ok=True)
        (self.root / "cmd/agentic-sdd/opencode_disabled.go").write_text("// fixture\n")
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
            "sys.exit(int(os.environ.get('FAIL_GO', '0')))\n"
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
            "GOFLAGS": "-tags=agentic_sdd_experimental",
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
                self.assertEqual([row["argv"] for row in self.observations()], [[mode, *CORE_PACKAGES]])

    def test_offline_environment_and_inherited_filters_cleared(self):
        result = self.run_profile("test", GOFLAGS="-tags=agentic_sdd_experimental")
        self.assertEqual(result.returncode, 0, result.stderr)
        for row in self.observations():
            self.assertEqual(row["env"], {
                "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
                "GOFLAGS": "", "GOENV": "off", "GOWORK": "off", "GOCACHE": self.env["GOCACHE"],
            })

    def test_nonzero_failure_propagation_from_go(self):
        for code in ("7", "9"):
            with self.subTest(code=code):
                self.capture.unlink(missing_ok=True)
                result = self.run_profile("test", FAIL_GO=code)
                self.assertEqual(result.returncode, int(code))
                self.assertEqual(len(self.observations()), 1)

    def test_invalid_mode_or_extra_args_do_not_run_go(self):
        for args in ((), ("all",), ("test", "./...")):
            with self.subTest(args=args):
                result = self.run_profile(*args)
                self.assertEqual(result.returncode, 2)
                self.assertEqual(self.observations(), [])

    def test_missing_core_directory_or_file_fails(self):
        cases = [
            self.root / "go.mod",
            self.root / "cmd/agentic-sdd/opencode_disabled.go",
            self.root / "internal/version",
            self.root / "cmd/sdd-check",
            self.root / "internal/skillsync",
            self.root / "internal/harnesscheck",
            self.root / "internal/harnessevidence",
            self.root / "internal/appops",
            self.root / "internal/buildinfo",
            self.root / "internal/fleet",
            self.root / "internal/execution",
            self.root / "cmd/agentic-sdd",
        ]
        for target in cases:
            with self.subTest(target=target.name):
                self.capture.unlink(missing_ok=True)
                is_dir = target.is_dir()
                if is_dir:
                    shutil.rmtree(target)
                else:
                    target.unlink()
                result = self.run_profile("test")
                self.assertEqual(result.returncode, 2)
                self.assertIn("missing required core", result.stderr)
                self.assertEqual(self.observations(), [])
                if is_dir:
                    target.mkdir(parents=True, exist_ok=True)
                else:
                    target.write_text("// restored\n")

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
                self.assertEqual(len(rows), 1)
                self.assertEqual(rows[0]["argv"], [mode, *CORE_PACKAGES])
                self.assertEqual(rows[0]["env"]["GOCACHE"], cache)

    def test_default_task_and_core_alias_parity_and_goflags_neutralization(self):
        task = shutil.which("task")
        self.assertIsNotNone(task)

        # self.env has GOFLAGS="-tags=agentic_sdd_experimental"
        for default_task, alias_task, mode in (("test", "test-core", "test"), ("vet", "vet-core", "vet")):
            with self.subTest(mode=mode):
                self.capture.unlink(missing_ok=True)
                res_default = subprocess.run([task, default_task], cwd=self.root, env=self.env, text=True, capture_output=True)
                self.assertEqual(res_default.returncode, 0, res_default.stderr)
                obs_default = self.observations()
                self.assertEqual(len(obs_default), 1)
                self.assertEqual(obs_default[0]["argv"], [mode, *CORE_PACKAGES])
                self.assertEqual(obs_default[0]["env"]["GOFLAGS"], "")

                self.capture.unlink(missing_ok=True)
                res_alias = subprocess.run([task, alias_task], cwd=self.root, env=self.env, text=True, capture_output=True)
                self.assertEqual(res_alias.returncode, 0, res_alias.stderr)
                obs_alias = self.observations()
                self.assertEqual(len(obs_alias), 1)
                self.assertEqual(obs_alias[0]["argv"], [mode, *CORE_PACKAGES])
                self.assertEqual(obs_alias[0]["env"]["GOFLAGS"], "")


class CoreGateParityTests(unittest.TestCase):
    def test_checks_contract_taskfile_and_script_core_parity(self):
        contract = json.loads((ROOT / "harness/checks.json").read_text())
        required = {check["id"]: check for check in contract["checks"] if "required" in check["scopes"]}
        self.assertEqual(set(required), {"mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"})
        self.assertEqual(required["test"]["steps"][0]["argv"], ["go", "test", *CORE_PACKAGES])
        self.assertEqual(required["vet"]["steps"][0]["argv"], ["go", "vet", *CORE_PACKAGES])

        taskfile = (ROOT / "Taskfile.yml").read_text()
        core_args = " ".join(CORE_PACKAGES)
        self.assertIn(f"go test {core_args}", taskfile)
        self.assertIn(f"go vet {core_args}", taskfile)
        self.assertIn("GOFLAGS= ", taskfile)
        self.assertIn("test-core:", taskfile)
        self.assertIn("vet-core:", taskfile)

        script = (ROOT / "tests/supported-core-go.sh").read_text()
        self.assertIn("GOFLAGS=", script)
        self.assertIn(" ".join(CORE_PACKAGES), script)


if __name__ == "__main__":
    unittest.main()
