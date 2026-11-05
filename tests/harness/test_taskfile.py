"""Behavioral checks for the Task developer interface; never run live tasks."""
import importlib.util
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("trusted_ci", ROOT / "tests/harness/trusted_ci.py")
trusted = importlib.util.module_from_spec(spec)
spec.loader.exec_module(trusted)


@unittest.skipUnless(shutil.which("task"), "Go Task is unavailable")
class TaskfileTests(unittest.TestCase):
    def task(self, *args):
        env = os.environ.copy()
        for name in ("V", "MSG", "SDD_TRUSTED"):
            env.pop(name, None)
        return subprocess.run(["task", *args], cwd=ROOT, env=env, text=True, capture_output=True)

    def test_default_lists_tasks_without_executing_them(self):
        result = self.task()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("preview", result.stdout)
        self.assertIn("verify-trusted", result.stdout)
        self.assertNotIn("go run", result.stdout)

    def test_tag_requires_version_without_creating_tag(self):
        result = self.task("tag")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("V is required", result.stderr + result.stdout)

    def test_trusted_verification_requires_reviewed_base(self):
        result = self.task("verify-trusted", "SDD_TRUSTED=")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SDD_TRUSTED", result.stderr + result.stdout)

    def test_verify_passes_output_and_offline_environment_to_runner(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            fake_go = root / "go"
            fake_go.write_text('#!/bin/sh\nprintf "%s\\n" "$GOTOOLCHAIN|$GOPROXY|$GOSUMDB|$*" > "$CAPTURE"\n')
            fake_go.chmod(0o755)
            output = root / "evidence"
            env = os.environ.copy()
            env.update(PATH=str(root) + os.pathsep + env["PATH"], CAPTURE=str(root / "capture"),
                       GOTOOLCHAIN="auto", GOPROXY="https://invalid.example", GOSUMDB="example",
                       SDD_OUTPUT=str(root / "wrong-output"))
            result = subprocess.run(["task", "verify", "SDD_OUTPUT=" + str(output)], cwd=ROOT,
                                    env=env, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            captured = (root / "capture").read_text()
            self.assertIn("local|off|off|", captured)
            self.assertIn("--scope required", captured)
            self.assertIn("--output " + str(output), captured)

    def test_command_line_cache_overrides_inherited_cache(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            fake_go = root / "go"
            fake_go.write_text('#!/bin/sh\nprintf "%s" "$GOCACHE" > "$CAPTURE"\n')
            fake_go.chmod(0o755)
            env = os.environ.copy()
            env.update(PATH=str(root) + os.pathsep + env["PATH"], CAPTURE=str(root / "capture"), GOCACHE="/wrong")
            result = subprocess.run(["task", "test", "GOCACHE=/desired"], cwd=ROOT, env=env, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual((root / "capture").read_text(), "/desired")


class TrustedPathsTests(unittest.TestCase):
    def test_ci_uses_only_reviewed_base_taskfile(self):
        workflow = (ROOT / '.github/workflows/ci.yml').read_text()
        self.assertIn('go-task/setup-task@a00fbb05ce67b35648be3c78cbc9fd85354c757e', workflow)
        self.assertIn('task --dir "$GITHUB_WORKSPACE/gate-trusted" verify-trusted', workflow)
        self.assertIn('"SDD_TRUSTED=$GITHUB_WORKSPACE/gate-trusted"', workflow)
        self.assertIn('"SDD_CANDIDATE=$GITHUB_WORKSPACE/gate-candidate"', workflow)
        self.assertNotIn('task --dir "$GITHUB_WORKSPACE/gate-candidate"', workflow)

    def test_taskfile_changes_require_review(self):
        self.assertEqual(trusted.review_paths({}, {"Taskfile.yml": {"sha256": "x"}}), ["Taskfile.yml"])

    def test_trusted_base_rejects_changed_local_recipe_contract(self):
        candidate = json.loads((ROOT / "harness/checks.json").read_text())
        original = copy.deepcopy(candidate)
        original["checks"][0]["local_recipe"] = ["make", "mod-verify"]
        with self.assertRaisesRegex(ValueError, "reviewed adoption required"):
            trusted.required_contract(original, candidate)


if __name__ == "__main__":
    unittest.main()
