"""Offline structural contract for the Feature 023 publisher boundary.

These checks inspect workflow text, not GitHub's execution or branch rules.
Behavioral event/CI fixtures require the preflight's public input contract.
"""

from pathlib import Path
import hashlib
import os
import re
import shutil
import subprocess
import tempfile
import unittest

from test_b7_release_scripts import COMMIT, TAG, VERSION, write_release
from test_b7_release_workflow import _find_step, _jobs, _run_block, _step_name

WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/release-publish.yml"


def jobs(text):
    block = re.search(r"(?ms)^jobs:\s*\n(.*?)(?=^\S|\Z)", text)
    if not block:
        return {}
    source = block.group(1)
    matches = list(re.finditer(r"(?m)^  ([\w-]+):\s*$", source))
    return {
        match.group(1): source[match.end():matches[i + 1].start() if i + 1 < len(matches) else len(source)]
        for i, match in enumerate(matches)
    }


def header(job):
    return re.split(r"(?m)^    steps:\s*$", job, maxsplit=1)[0]


def permissions(text, indent):
    match = re.search(rf"(?m)^{' ' * indent}permissions:\s*(\{{\}})?\s*$", text)
    if not match:
        return None
    if match.group(1):
        return {}
    result = {}
    for line in text[match.end():].splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if len(line) - len(line.lstrip()) <= indent:
            break
        entry = re.fullmatch(r"\s+([\w-]+):\s*(\w+)\s*(?:#.*)?", line)
        if not entry:
            return None
        result[entry.group(1)] = entry.group(2)
    return result


class PublisherBoundaryContract(unittest.TestCase):
    def setUp(self):
        self.text = WORKFLOW.read_text(encoding="utf-8")
        self.jobs = jobs(self.text)

    def test_publisher_requires_a_separate_preflight_job(self):
        self.assertIn("publish", self.jobs)
        publish = header(self.jobs["publish"])
        match = re.search(r"(?m)^    needs:\s*(?:\[([^\]]+)\]|([\w-]+))\s*$", publish)
        self.assertIsNotNone(match, "publish must depend on a separate preflight")
        dependencies = re.findall(r"[\w-]+", match.group(1) or match.group(2))
        self.assertTrue(dependencies)
        for name in dependencies:
            self.assertIn(name, self.jobs)
            self.assertNotEqual(name, "publish")
        self.assertNotRegex(publish, r"(?i)\b(always|failure|cancelled)\s*\(",
                            "publish must not bypass a failed preflight")

    def test_preflight_has_only_read_permissions_and_no_publishing_secrets(self):
        self.assertEqual(permissions(self.text, 0), {}, "workflow permissions must default to none")
        publish = header(self.jobs.get("publish", ""))
        match = re.search(r"(?m)^    needs:\s*(?:\[([^\]]+)\]|([\w-]+))\s*$", publish)
        self.assertIsNotNone(match, "publish needs a read-only preflight")
        for name in re.findall(r"[\w-]+", match.group(1) or match.group(2)):
            with self.subTest(job=name):
                self.assertIn(name, self.jobs)
                job = self.jobs[name]
                scopes = permissions(header(job), 4)
                self.assertIsNotNone(scopes, "declare explicit job permissions")
                self.assertTrue(all(value in ("read", "none") for value in scopes.values()), scopes)
                self.assertNotRegex(job, r"(?i)\bsecrets\s*\.")
                self.assertNotRegex(job, r"(?m)^\s*environment\s*:")
                self.assertNotRegex(job, r"(?i)\bgh\s+release\s+(create|upload|edit|delete)\b")
                self.assertNotRegex(job, r"(?i)\bgit\s+(?:-C\s+\S+\s+)?push\b")


FAKE_GIT = r"""#!/bin/sh
set -eu
if [ "$1" = clone ] && [ "$2" = --quiet ] &&
   [ "$3" = https://github.com/nawodyaishan/homebrew-tap.git ] && [ "$#" = 4 ]; then
  exec "$REAL_GIT" clone --quiet "$LOCAL_TAP" "$4"
fi
if [ "$1" = -C ] && [ "$2" = "$RUNNER_TEMP/homebrew-tap" ]; then
  shift 2
  case "$1" in
    pull|add|push) exec "$REAL_GIT" -C "$RUNNER_TEMP/homebrew-tap" "$@" ;;
    -c) exec "$REAL_GIT" -C "$RUNNER_TEMP/homebrew-tap" "$@" ;;
  esac
fi
echo "unexpected git command" >&2
exit 99
"""

FAKE_GH = r"""#!/usr/bin/python3
import os, pathlib, shutil, sys
args = sys.argv[1:]
root = pathlib.Path(os.environ["FAKE_RELEASE"])
tag = os.environ["TAG_NAME"]
if args == ["api", "repos/nawodyaishan/homebrew-tap", "--jq", ".default_branch"]:
    print("main")
elif args == ["auth", "setup-git"]:
    pass
elif args == ["release", "view", tag]:
    sys.exit(0 if (root / "created").exists() else 1)
elif args == ["release", "create", tag, "--verify-tag", "--title", tag, "--generate-notes"]:
    root.mkdir(exist_ok=True)
    (root / "created").write_text(tag)
elif len(args) >= 4 and args[:3] == ["release", "upload", tag]:
    assets = root / "assets"
    assets.mkdir(exist_ok=True)
    # Refuse an existing name without overwriting any existing asset.
    for name in args[3:]:
        if (assets / pathlib.Path(name).name).exists():
            sys.exit(1)
    for name in args[3:]:
        shutil.copyfile(name, assets / pathlib.Path(name).name)
else:
    print("unexpected gh command: " + repr(args), file=sys.stderr)
    sys.exit(99)
"""


class OfflinePublisherStepTests(unittest.TestCase):
    """Execute the real B7 step scripts; local git and fake gh are the only sinks."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.work = self.root / "work"
        self.work.mkdir()
        self.runner = self.root / "runner"
        self.runner.mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.remote = self.root / "tap.git"
        self.sink = self.root / "release"
        self.git = shutil.which("git")
        self.assertIsNotNone(self.git)
        self.env = {
            "PATH": str(self.bin) + os.pathsep + os.environ.get("PATH", "/usr/bin:/bin"),
            "HOME": str(self.root),
            "GIT_CONFIG_NOSYSTEM": "1",
            "GIT_CONFIG_GLOBAL": os.devnull,
            "GIT_ALLOW_PROTOCOL": "file",
            "GIT_TERMINAL_PROMPT": "0",
            "GIT_AUTHOR_NAME": "Test", "GIT_AUTHOR_EMAIL": "test@example.invalid",
            "GIT_COMMITTER_NAME": "Test", "GIT_COMMITTER_EMAIL": "test@example.invalid",
            "REAL_GIT": self.git, "LOCAL_TAP": str(self.remote),
            "RUNNER_TEMP": str(self.runner), "FAKE_RELEASE": str(self.sink),
            "TAG_NAME": TAG, "TAG_COMMIT": COMMIT, "GH_REPO": "test/repo",
        }
        for name, content in (("git", FAKE_GIT), ("gh", FAKE_GH)):
            path = self.bin / name
            path.write_text(content)
            path.chmod(0o755)
        self.git_cmd("init", "--bare", "--initial-branch=main", str(self.remote))
        seed = self.root / "seed"
        self.git_cmd("clone", str(self.remote), str(seed))
        (seed / "Casks").mkdir()
        (seed / "Casks/agentic-sdd.rb").write_text('cask "agentic-sdd" do\n  version "0.12.9"\nend\n')
        self.git_cmd("-C", str(seed), "add", "Casks/agentic-sdd.rb")
        self.git_cmd("-C", str(seed), "commit", "-m", "seed")
        self.git_cmd("-C", str(seed), "push", "origin", "HEAD:main")
        _, out, result = write_release(self.work)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.artifacts = out
        tools = self.work / "release-tools" / "scripts"
        tools.mkdir(parents=True)
        for name in ("release_cask.py", "release_manifest.py"):
            shutil.copy2(WORKFLOW.parents[2] / "scripts" / name, tools / name)
        publisher = _jobs(WORKFLOW.read_text())["publish"]
        self.scripts = {}
        for key, prefix in (("preflight", "Read Homebrew tap"),
                            ("release", "Create or reuse the GitHub release"),
                            ("tap", "Update Homebrew tap")):
            _, step = _find_step(publisher, lambda body: _step_name(body).startswith(prefix))
            self.assertIsNotNone(step, f"missing publisher step {prefix}")
            self.scripts[key] = _run_block(step)

    def git_cmd(self, *args):
        return subprocess.run([self.git, *args], env=self.env, cwd=self.root,
                              capture_output=True, text=True, check=True, timeout=30).stdout.strip()

    def run_step(self, name):
        return subprocess.run(["bash", "-c", self.scripts[name]], cwd=self.work,
                              env=self.env, capture_output=True, text=True, timeout=30)

    def remote_cask(self):
        return self.git_cmd("--git-dir", str(self.remote), "show", "refs/heads/main:Casks/agentic-sdd.rb")

    def assert_ok(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_success_uploads_exact_artifacts_and_updates_local_tap(self):
        self.assert_ok(self.run_step("preflight"))
        self.assertFalse(self.sink.exists())
        self.assert_ok(self.run_step("release"))
        assets = self.sink / "assets"
        self.assertEqual({p.name for p in assets.iterdir()}, {p.name for p in self.artifacts.iterdir()})
        for path in self.artifacts.iterdir():
            self.assertEqual((assets / path.name).read_bytes(), path.read_bytes())
        self.assert_ok(self.run_step("tap"))
        text = self.remote_cask()
        archive = self.artifacts / f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        self.assertIn(f'version "{VERSION}"', text)
        self.assertIn(hashlib.sha256(archive.read_bytes()).hexdigest(), text)
        self.assertIn(f"/{TAG}/{archive.name}", text)

    def test_downgrade_refuses_before_release_or_tap_write(self):
        clone = self.root / "seed"
        (clone / "Casks/agentic-sdd.rb").write_text('cask "agentic-sdd" do\n  version "0.14.0"\nend\n')
        self.git_cmd("-C", str(clone), "add", "Casks/agentic-sdd.rb")
        self.git_cmd("-C", str(clone), "commit", "-m", "newer")
        self.git_cmd("-C", str(clone), "push", "origin", "HEAD:main")
        before = self.remote_cask()
        self.assertNotEqual(self.run_step("preflight").returncode, 0)
        self.assertFalse(self.sink.exists())
        self.assertEqual(self.remote_cask(), before)

    def test_partial_upload_retry_refuses_existing_asset_without_tap_write(self):
        self.assert_ok(self.run_step("preflight"))
        self.sink.mkdir()
        (self.sink / "created").write_text(TAG)
        assets = self.sink / "assets"
        assets.mkdir()
        existing = assets / "checksums.txt"
        existing.write_text("previous upload")
        before = self.remote_cask()
        self.assertNotEqual(self.run_step("release").returncode, 0)
        self.assertEqual(existing.read_text(), "previous upload")
        self.assertEqual({p.name for p in assets.iterdir()}, {"checksums.txt"})
        self.assertEqual(self.remote_cask(), before)

    def test_tampered_archive_refuses_tap_update_after_upload(self):
        self.assert_ok(self.run_step("preflight"))
        self.assert_ok(self.run_step("release"))
        before = self.remote_cask()
        archive = self.artifacts / f"agentic-sdd_{VERSION}_darwin_all.tar.gz"
        archive.write_bytes(archive.read_bytes() + b"tampered")
        result = self.run_step("tap")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.remote_cask(), before)
        self.assertNotEqual((self.sink / "assets" / archive.name).read_bytes(), archive.read_bytes())

    def test_rerun_after_tap_update_refuses_equal_version_without_extra_writes(self):
        self.assert_ok(self.run_step("preflight"))
        self.assert_ok(self.run_step("release"))
        self.assert_ok(self.run_step("tap"))
        before = self.git_cmd("--git-dir", str(self.remote), "rev-parse", "refs/heads/main")
        assets = {p.name: p.read_bytes() for p in (self.sink / "assets").iterdir()}
        self.assertNotEqual(self.run_step("tap").returncode, 0)
        self.assertEqual(self.git_cmd("--git-dir", str(self.remote), "rev-parse", "refs/heads/main"), before)
        self.assertEqual({p.name: p.read_bytes() for p in (self.sink / "assets").iterdir()}, assets)


if __name__ == "__main__":
    unittest.main()
