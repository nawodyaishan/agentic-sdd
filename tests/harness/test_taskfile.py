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

    def test_task_test_and_vet_sanitize_environment_and_select_core(self):
        core_args = ". ./internal/version ./internal/skillsync ./internal/harnesscheck ./internal/harnessevidence ./cmd/agentic-sdd ./cmd/sdd-check"
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            fake_go = root / "go"
            fake_go.write_text(
                '#!/bin/sh\n'
                'printf "%s|%s|%s|%s|%s|%s|%s|%s\\n" '
                '"$GOTOOLCHAIN" "$GOPROXY" "$GOSUMDB" "$GOENV" "$GOWORK" "$GOFLAGS" "$GOCACHE" "$*" > "$CAPTURE"\n'
            )
            fake_go.chmod(0o755)
            capture = root / "capture"
            env = os.environ.copy()
            env.update(
                PATH=str(root) + os.pathsep + env["PATH"],
                CAPTURE=str(capture),
                GOTOOLCHAIN="auto",
                GOPROXY="https://invalid.example",
                GOSUMDB="example",
                GOENV="/unreviewed/env",
                GOWORK="/unreviewed/work",
                GOFLAGS="-tags=agentic_sdd_experimental",
                GOCACHE="/custom/cache",
            )
            for mode in ("test", "vet"):
                with self.subTest(mode=mode):
                    capture.unlink(missing_ok=True)
                    result = subprocess.run(["task", mode], cwd=ROOT, env=env, text=True, capture_output=True)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    captured = capture.read_text().strip()
                    parts = captured.split("|")
                    self.assertEqual(parts[0], "local")
                    self.assertEqual(parts[1], "off")
                    self.assertEqual(parts[2], "off")
                    self.assertEqual(parts[3], "off")
                    self.assertEqual(parts[4], "off")
                    self.assertEqual(parts[5], "")
                    self.assertEqual(parts[6], "/custom/cache")
                    self.assertEqual(parts[7], f"{mode} {core_args}")

    def test_experimental_taskfile_separation(self):
        root_taskfile = (ROOT / "Taskfile.yml").read_text()
        self.assertNotIn("opencode-help", root_taskfile)
        self.assertNotIn("Taskfile.experimental.yml", root_taskfile)
        self.assertNotIn("includes:", root_taskfile)

        result = self.task()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotIn("opencode-help", result.stdout)

        exp_taskfile = ROOT / "Taskfile.experimental.yml"
        self.assertTrue(exp_taskfile.is_file(), "Taskfile.experimental.yml must exist")
        exp_text = exp_taskfile.read_text()
        self.assertIn("opencode-help:", exp_text)
        self.assertIn("-tags agentic_sdd_experimental", exp_text)

        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            fake_go = root / "go"
            fake_go.write_text(
                '#!/bin/sh\n'
                'printf "%s|%s|%s|%s|%s|%s|%s\\n" '
                '"$GOTOOLCHAIN" "$GOPROXY" "$GOSUMDB" "$GOENV" "$GOWORK" "$GOCACHE" "$*" > "$CAPTURE"\n'
            )
            fake_go.chmod(0o755)
            env = os.environ.copy()
            capture = root / "capture"
            env.update(
                PATH=str(root) + os.pathsep + env["PATH"],
                CAPTURE=str(capture),
                GOTOOLCHAIN="auto",
                GOPROXY="https://invalid.example",
                GOSUMDB="example",
                GOENV="/unreviewed/env",
                GOWORK="/unreviewed/work",
                GOCACHE="/custom/cache",
            )
            res = subprocess.run(
                ["task", "-t", str(exp_taskfile), "opencode-help"],
                cwd=ROOT,
                env=env,
                text=True,
                capture_output=True,
            )
            self.assertEqual(res.returncode, 0, res.stderr)
            captured = capture.read_text()
            self.assertIn("local|off|off|off|off|", captured)
            self.assertIn("/custom/cache", captured)
            self.assertIn("-tags agentic_sdd_experimental", captured)
            self.assertIn("opencode --help", captured)



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

    def test_goreleaser_uses_default_inactive_core_profile(self):
        goreleaser = (ROOT / ".goreleaser.yml").read_text()
        self.assertNotIn("agentic_sdd_experimental", goreleaser)
        self.assertIn("-trimpath", goreleaser)


if __name__ == "__main__":
    unittest.main()
