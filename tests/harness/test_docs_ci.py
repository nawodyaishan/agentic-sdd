"""Adversarial fixtures for the trusted docs-only CI launcher (020 T10 / B5-W1-C).

Coverage maps to plan C4/C5 and T10 verification:
  * candidate exit 0 / self-authored JSON never become an authoritative verdict;
  * candidate edits to workflow, launcher, validator or required-page set and
    candidate-supplied policy/check attempts stop before execution;
  * lock/image identity drift stops before any container;
  * wrong base SHA and a head SHA that moves after the build both fail;
  * candidate containers cannot reach the network, host home, Docker socket or a
    shared cache, and hostile host environment keys are never forwarded;
  * artifact symlinks, hardlinks, empty page sets, private/history paths fail;
  * release mode is observational and never blocks, rollback of a missing or
    expired artifact is blocked;
  * candidate checks run in the policy's site root, which must hold the invoked
    package, and `--mode main --artifact` publishes only the validated site;
  * a retained artifact is verified against an explicit prior revision or run
    id, never against the current HEAD, and an unbound or forged one is refused;
  * docs.yml pulls the digest-pinned image before every launcher invocation.
  * the container user can write its own `$HOME` (mode=1777 tmpfs, proven with
    the real pinned image when Docker is present) and a runner timeout still
    removes every container the launcher recorded;
  * the PR lane creates the exact `refs/pull/<n>/head` ref the launcher resolves;
  * evidence uploads carry the trust record and check logs, never the staged
    tree, `node_modules` or the build;
  * `--mode links` is observation-only, never fetches without `--network`,
    refuses non-http(s) and local/private targets, does not follow redirects and
    is deterministic;
  * documented local-only trees (`.codegraph/`, `backups/`) are skipped with a
    recorded observation, while real credential files and sensitive directories
    are still refused.

The launcher (B5-W1-A product) is ``scripts/docs-ci.py``. This authoring worktree
may not contain it; when it is absent the suite reports ``integration-pending``
explicitly instead of passing. Set ``DOCS_CI_LAUNCHER`` to an integrated copy
while authoring the fixtures; the parent run uses the repository default.

Every ``git``/``docker`` invocation is a real fake executable process with a real
exit status. They are dispatched through the launcher's documented
``Context(runner=...)`` fixture seam because the launcher deliberately rebuilds
child ``PATH`` as ``/usr/local/bin:/usr/bin:/bin`` and a fixture must not write
into system directories. The base-owned ``evaluate`` step is dispatched through
the same seam so it remains a separate launcher invocation that reads only the
on-disk identity/records after the candidate process exited.

No network, credential, deployment, tag or real home directory is used.
"""

import contextlib
import copy
import hashlib
import importlib.util
import io
import itertools
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
DEFAULT_LAUNCHER = ROOT / "scripts/docs-ci.py"
LAUNCHER = Path(os.environ.get("DOCS_CI_LAUNCHER", str(DEFAULT_LAUNCHER))).resolve()

SHA_BASE = "1" * 40
SHA_CAND = "2" * 40
SHA_MOVED = "3" * 40
SHA_FOREIGN = "4" * 40
PULL_REF = "refs/pull/7/head"
IMAGE = "node:22-bookworm@sha256:" + "a" * 64
# The reviewed runtime pin used by docs.yml; the container-$HOME test runs it.
DOCS_IMAGE = "node@sha256:dc26ae92c9fb15cf6b1d63e20f0944b2efe0f115b20e00be48fb23fceb061c54"
SITE_ROOT = "website"

REQUIRED_PAGES = [
    {"id": "start", "class": "L", "source": "website/src/content/docs/index.md"},
    {"id": "guide", "class": "C", "source": "website/src/content/docs/guide.md"},
]
PACKAGE = {
    "name": "docs-ci-fixture",
    "private": True,
    "scripts": {
        "build": "astro build",
        "docs:check": "node scripts/check.mjs",
        "docs:check-built": "node scripts/check.mjs",
        "test": "node --test tests/*.test.mjs",
    },
}

# The ordered adopted check list the trusted base runs and the evaluator
# requires. Mirrors the launcher's embedded policy; the export write/check pair
# exists because `--check` refuses a missing output directory.
ADOPTED_CHECKS = [
    {"id": "build", "argv": ["npm", "run", "build"]},
    {"id": "docs-check", "argv": ["npm", "run", "docs:check"]},
    {"id": "docs-check-built",
     "argv": ["npm", "run", "docs:check-built", "--", "--built", "dist", "--source"]},
    {"id": "docs-export-write",
     "argv": ["node", "scripts/export.mjs", "--out", "dist/export"]},
    {"id": "docs-export-check",
     "argv": ["node", "scripts/export.mjs", "--out", "dist/export", "--check"]},
    {"id": "node-test", "argv": ["npm", "test"]},
]

# Every protected path the trusted base owns; a candidate edit to any of them
# needs trusted policy review. package.json/package-lock.json are also
# dependency files and keep the separate dependency-review result.
PROTECTED_PATHS = [
    ".github/workflows",
    "scripts/docs-ci.py",
    "website/scripts",
    "website/plugins",
    "website/tests",
    "website/astro.config.mjs",
    "website/src/content.config.ts",
    "website/.markdownlint.json",
    "website/package.json",
    "website/package-lock.json",
    "tests/harness/test_docs_ci.py",
    "Taskfile.yml",
]
LOCK = {"name": "docs-ci-fixture", "lockfileVersion": 3, "packages": {}}

_LOAD_COUNTER = itertools.count()


def default_artifact():
    return {
        "files": {
            "index.html": "<!doctype html><title>Home</title>\n",
            "guide/index.html": "<!doctype html><title>Guide</title>\n",
            "sitemap.xml": "<urlset></urlset>\n",
        },
        "dirs": [],
        "symlinks": {},
        "hardlinks": {},
    }


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def artifact_digest(root):
    """Same digest the launcher records for a scanned artifact tree."""
    root = Path(root)
    files = {}
    for path in sorted(root.rglob("*")):
        if path.is_symlink() or not path.is_file():
            continue
        data = path.read_bytes()
        files[path.relative_to(root).as_posix()] = {
            "sha256": hashlib.sha256(data).hexdigest(),
            "size": len(data),
        }
    return hashlib.sha256(canonical(files).encode("utf-8")).hexdigest()


class Result:
    __slots__ = ("code", "payload")

    def __init__(self, code, payload):
        self.code = code
        self.payload = payload


FAKE_GIT = r'''#!%(python)s
import json
import sys

STATE = %(state)r
LOG = %(log)r

argv = sys.argv[1:]
try:
    with open(LOG, "a", encoding="utf-8") as stream:
        stream.write(json.dumps({"tool": "git", "argv": argv}, sort_keys=True) + "\n")
except OSError:
    pass

if "-C" not in argv:
    sys.stderr.write("fake git: missing -C\n")
    sys.exit(2)
root = argv[argv.index("-C") + 1]
try:
    with open(STATE, encoding="utf-8") as stream:
        state = json.load(stream)
except OSError:
    sys.stderr.write("fake git: state unavailable\n")
    sys.exit(3)

if "status" in argv:
    code = int(state.get("status_exit", 0))
    if code == 0:
        for line in state.get("dirty", []):
            sys.stdout.write(line + "\n")
    sys.exit(code)
if "rev-parse" not in argv:
    sys.exit(2)
if "--show-toplevel" in argv:
    if root in state["roots"]:
        sys.stdout.write(root + "\n")
        sys.exit(0)
    sys.exit(1)
if "--verify" in argv:
    target = argv[argv.index("--verify") + 1]
    if target == "HEAD":
        sha = state["roots"].get(root)
    else:
        ref = target[:-9] if target.endswith("^{commit}") else target
        sha = state.get("refs", {}).get(ref)
    if not sha:
        sys.stderr.write("fake git: unknown revision\n")
        sys.exit(1)
    sys.stdout.write(sha + "\n")
    sys.exit(0)
sys.exit(2)
'''

FAKE_DOCKER = r'''#!%(python)s
import json
import os
import sys
from pathlib import Path

SCENARIO = %(scenario)r
LOG = %(log)r


def record(kind, **extra):
    entry = {"kind": kind, "argv": list(sys.argv[1:]), "env": dict(os.environ)}
    entry.update(extra)
    try:
        with open(LOG, "a", encoding="utf-8") as stream:
            stream.write(json.dumps(entry, sort_keys=True) + "\n")
    except OSError:
        pass


def parse_mount(spec):
    data = {}
    flags = []
    for part in spec.split(","):
        if "=" in part:
            key, value = part.split("=", 1)
            data[key] = value
        elif part:
            flags.append(part)
    return {"spec": spec, "src": data.get("src", ""), "dst": data.get("dst", ""),
            "writable": "readonly" not in flags}


def read_scenario():
    with open(SCENARIO, encoding="utf-8") as stream:
        return json.load(stream)


def main():
    argv = sys.argv[1:]
    if not argv:
        return 2
    if argv[0] == "image":
        scenario = read_scenario()
        record("image", image=argv[-1])
        return scenario.get("image_exit", 0)
    if argv[0] == "rm":
        record("rm")
        return 0
    if argv[0] != "run":
        record("other")
        return 2

    scenario = read_scenario()
    mounts = []
    env_map = {}
    tmpfs = []
    cidfile = ""
    workdir = ""
    network = ""
    user = ""
    image = ""
    command = []
    index = 1
    while index < len(argv):
        token = argv[index]
        if token in ("--cidfile", "--tmpfs", "--workdir", "--mount", "-e"):
            value = argv[index + 1]
            if token == "--cidfile":
                cidfile = value
            elif token == "--workdir":
                workdir = value
            elif token == "--mount":
                mounts.append(parse_mount(value))
            elif token == "-e":
                key, _, item = value.partition("=")
                env_map[key] = item
            else:
                tmpfs.append(value)
            index += 2
            continue
        if token.startswith("--network="):
            network = token.split("=", 1)[1]
        elif token.startswith("--user="):
            user = token.split("=", 1)[1]
        elif token.startswith("-"):
            pass
        elif not image:
            image = token
        else:
            command = argv[index:]
            break
        index += 1

    record("run", network=network, user=user, workdir=workdir, image=image,
           command=command, mounts=mounts, env_map=env_map, tmpfs=tmpfs,
           cidfile=cidfile)
    if cidfile:
        with open(cidfile, "w", encoding="utf-8") as stream:
            stream.write("ab" * 32)

    provision = next((item for item in mounts if item["dst"] == "/provision"), None)
    if provision is not None or command[:2] == ["npm", "ci"]:
        if provision is not None:
            modules = Path(provision["src"]) / "node_modules" / "fixture-package"
            modules.mkdir(parents=True, exist_ok=True)
            (modules / "index.js").write_text("module.exports = 1;\n", encoding="utf-8")
            (modules / "package.json").write_text(
                json.dumps({"name": "fixture-package"}) + "\n", encoding="utf-8")
        return scenario.get("provision_exit", 0)

    artifact = scenario.get("artifact") or {}
    dist = next((item for item in mounts if item["dst"] == "/workspace/website/dist"), None)
    if dist is not None:
        base = Path(dist["src"])
        for name in artifact.get("dirs", []):
            (base / name).mkdir(parents=True, exist_ok=True)
        for name, text in artifact.get("files", {}).items():
            path = base / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding="utf-8")
        for name, target in artifact.get("symlinks", {}).items():
            path = base / name
            path.parent.mkdir(parents=True, exist_ok=True)
            os.symlink(target, path)
        for name, target in artifact.get("hardlinks", {}).items():
            path = base / name
            path.parent.mkdir(parents=True, exist_ok=True)
            os.link(base / target, path)
    return scenario.get("build_exit", 0)


if __name__ == "__main__":
    sys.exit(main())
'''


def load_launcher(path):
    name = "docs_ci_fixture_%d_%d" % (os.getpid(), next(_LOAD_COUNTER))
    spec = importlib.util.spec_from_file_location(name, str(path))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def manifest(pages=None):
    return {
        "schemaVersion": 1,
        "site": "agentic-sdd",
        "base": "/agentic-sdd",
        "pages": copy.deepcopy(REQUIRED_PAGES if pages is None else pages),
    }


def write_retained(root, files=None, *, mode="main", result="pass", authoritative=True,
                   base_sha=SHA_BASE, digest=None, include_status=True, run_id=""):
    root = Path(root)
    site = root / "site"
    site.mkdir(parents=True, exist_ok=True)
    for name, text in (files or {"index.html": "<!doctype html><title>Home</title>\n"}).items():
        path = site / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
    if include_status:
        (root / "status.json").write_text(json.dumps({
            "schema": "docs-ci-status/1",
            "mode": mode,
            "result": result,
            "authoritative": authoritative,
            "blocking_release": False,
            "run": {"id": run_id},
            "identity": {"base_sha": base_sha},
            "artifact": {"files": 1, "bytes": 1,
                         "sha256": digest if digest is not None else artifact_digest(site)},
        }, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return root


class DocsCiFixture:
    """Disposable trusted base, candidate checkout and fake process tools."""

    def __init__(self, case):
        temporary = tempfile.TemporaryDirectory(prefix="docs-ci-fixture-")
        case.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.trusted = self.root / "trusted"
        self.candidate = self.root / "candidate"
        self.output = self.root / "output"
        self.bin = self.root / "bin"
        self.state_dir = self.root / "state"
        self.log_dir = self.root / "logs"
        for path in (self.bin, self.state_dir, self.log_dir):
            path.mkdir()

        self.build_tree(self.trusted)
        shutil.copytree(self.trusted, self.candidate)
        payload = LAUNCHER.read_bytes()
        (self.trusted / "scripts/docs-ci.py").write_bytes(payload)
        (self.candidate / "scripts/docs-ci.py").write_bytes(payload)

        self.scenario_path = self.state_dir / "scenario.json"
        self.git_state_path = self.state_dir / "git.json"
        self.docker_log = self.log_dir / "docker.jsonl"
        self.git_log = self.log_dir / "git.jsonl"
        self.policy_path = None
        self.evaluate_calls = []
        self.write_tools()
        self.git_state = {
            "roots": {str(self.trusted): SHA_BASE, str(self.candidate): SHA_CAND},
            "refs": {PULL_REF: SHA_CAND},
            "dirty": [],
        }
        self.write_git_state()
        self.configure()

        self.launcher = load_launcher(self.trusted / "scripts/docs-ci.py")
        self.ctx = self.launcher.Context(runner=self.runner)

    # ---------------------------------------------------------------- trees

    def build_tree(self, root):
        (root / "scripts").mkdir(parents=True)
        (root / "scripts/docs-ci.py").write_text("# fixture placeholder\n", encoding="utf-8")
        (root / "scripts/validate.py").write_text(
            "# reviewed base validator (fixture)\n", encoding="utf-8")
        (root / ".github/workflows").mkdir(parents=True)
        (root / ".github/workflows/ci.yml").write_text(
            "name: ci\non: push\njobs: {}\n", encoding="utf-8")
        (root / "website/src/content/docs").mkdir(parents=True)
        (root / "website/scripts").mkdir(parents=True)
        (root / "website/scripts/check.mjs").write_text(
            "// reviewed validator (fixture)\n", encoding="utf-8")
        (root / "website/scripts/export.mjs").write_text(
            "// reviewed export tool (fixture)\n", encoding="utf-8")
        (root / "website/plugins").mkdir(parents=True)
        (root / "website/plugins/remark-link-routes.mjs").write_text(
            "// reviewed route plugin (fixture)\n", encoding="utf-8")
        (root / "website/tests").mkdir(parents=True)
        (root / "website/tests/fixture.test.mjs").write_text(
            "// reviewed node suite (fixture)\n", encoding="utf-8")
        (root / "tests/harness").mkdir(parents=True)
        (root / "tests/harness/test_docs_ci.py").write_text(
            "# reviewed launcher fixtures (fixture)\n", encoding="utf-8")
        (root / "website/astro.config.mjs").write_text(
            "export default {};\n", encoding="utf-8")
        (root / "website/src/content.config.ts").write_text(
            "export const collections = {};\n", encoding="utf-8")
        (root / "website/.markdownlint.json").write_text("{}\n", encoding="utf-8")
        (root / "Taskfile.yml").write_text("version: '3'\n", encoding="utf-8")
        (root / "website/package.json").write_text(
            json.dumps(PACKAGE, indent=2) + "\n", encoding="utf-8")
        (root / "website/package-lock.json").write_text(
            json.dumps(LOCK, indent=2) + "\n", encoding="utf-8")
        (root / "website/.node-version").write_text("22\n", encoding="utf-8")
        (root / "website/docs-manifest.json").write_text(
            json.dumps(manifest(), indent=2) + "\n", encoding="utf-8")
        (root / "website/src/content/docs/index.md").write_text("# Home\n", encoding="utf-8")
        (root / "website/src/content/docs/guide.md").write_text("# Guide\n", encoding="utf-8")

    def write_tools(self):
        fake_git = self.bin / "git"
        fake_git.write_text(FAKE_GIT % {
            "python": sys.executable, "state": str(self.git_state_path),
            "log": str(self.git_log),
        }, encoding="utf-8")
        fake_git.chmod(0o755)
        fake_docker = self.bin / "docker"
        fake_docker.write_text(FAKE_DOCKER % {
            "python": sys.executable, "scenario": str(self.scenario_path),
            "log": str(self.docker_log),
        }, encoding="utf-8")
        fake_docker.chmod(0o755)

    # ------------------------------------------------------------ candidates

    def candidate_file(self, relative, text):
        path = self.candidate / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
        return path

    def trusted_file(self, relative, text):
        path = self.trusted / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
        return path

    def trusted_bytes(self, relative, data):
        path = self.trusted / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        return path

    def candidate_symlink(self, relative, target):
        path = self.candidate / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        os.symlink(target, path)

    def mutate_manifest(self, mutate):
        path = self.candidate / SITE_ROOT / "docs-manifest.json"
        data = json.loads(path.read_text(encoding="utf-8"))
        mutate(data)
        path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

    def write_policy(self, **overlay):
        self.policy_path = self.trusted / "scripts/docs-ci-policy-fixture.json"
        self.policy_path.write_text(json.dumps(overlay, indent=2) + "\n", encoding="utf-8")
        return self.policy_path

    def write_candidate_policy(self, payload):
        return self.candidate_file("scripts/candidate-policy.json",
                                   json.dumps(payload, indent=2) + "\n")

    # --------------------------------------------------------------- drivers

    def runner(self, argv, *, cwd=None, env=None, timeout=None):
        argv = [os.fspath(item) for item in argv]
        program = Path(argv[0]).name
        if program in ("git", "docker"):
            completed = subprocess.run(
                [str(self.bin / program), *argv[1:]], cwd=cwd, env=env,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout,
            )
            return self.launcher.Completed(
                completed.returncode, completed.stdout or b"", completed.stderr or b"")
        if program.startswith("python"):
            try:
                index = argv.index("--action")
            except ValueError:
                index = -1
            if index >= 0 and argv[index + 1] == "evaluate":
                self.evaluate_calls.append(list(argv))
                stream = io.StringIO()
                with contextlib.redirect_stdout(stream):
                    # [python, -B, <launcher>, ...argv] -> the launcher's own argv
                    code = self.launcher.run_cli(argv[3:], context=self.ctx)
                return self.launcher.Completed(code, stream.getvalue().encode("utf-8"), b"")
        raise AssertionError("unexpected external command in fixture: %r" % (argv,))

    def run(self, argv, *, context=None):
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            code = self.launcher.run_cli(list(argv), context=context or self.ctx)
        lines = [line for line in stream.getvalue().splitlines() if line.strip()]
        payload = json.loads(lines[-1]) if lines else None
        return Result(code, payload)

    def pr_argv(self, *, output=None, base_sha=SHA_BASE, candidate_sha=SHA_CAND,
                ref=PULL_REF, image=IMAGE, extra=()):
        argv = ["--trusted", str(self.trusted), "--candidate", str(self.candidate),
                "--output", str(output or self.output), "--mode", "pr"]
        if image:
            argv += ["--image", image]
        if base_sha:
            argv += ["--base-sha", base_sha]
        if candidate_sha:
            argv += ["--candidate-sha", candidate_sha]
        if ref:
            argv += ["--candidate-ref", ref]
        return argv + list(extra)

    def run_pr(self, **kwargs):
        return self.run(self.pr_argv(**kwargs))

    def links_argv(self, output, *, network=False, extra=(), artifact=None):
        argv = ["--mode", "links", "--trusted", str(self.trusted),
                "--output", str(output)]
        if artifact is not None:
            argv += ["--artifact", str(artifact)]
        if network:
            argv += ["--network"]
        return argv + list(extra)

    def run_links(self, output, *, network=False, extra=(), artifact=None):
        return self.run(self.links_argv(output, network=network, extra=extra,
                                        artifact=artifact))

    def mode_argv(self, mode, *, output, artifact=None, image=IMAGE, base_sha=None,
                  extra=()):
        argv = ["--trusted", str(self.trusted), "--output", str(output), "--mode", mode]
        if image:
            argv += ["--image", image]
        if base_sha:
            argv += ["--base-sha", base_sha]
        if artifact is not None:
            argv += ["--artifact", str(artifact)]
        return argv + list(extra)

    def configure(self, *, build_exit=0, provision_exit=0, image_exit=0, artifact=None):
        self.scenario_path.write_text(json.dumps({
            "image_exit": image_exit,
            "provision_exit": provision_exit,
            "build_exit": build_exit,
            "artifact": default_artifact() if artifact is None else artifact,
        }, indent=2) + "\n", encoding="utf-8")

    def write_git_state(self):
        self.git_state_path.write_text(json.dumps(self.git_state, indent=2) + "\n",
                                       encoding="utf-8")

    # -------------------------------------------------------------- evidence

    def status(self, output=None):
        path = Path(output or self.output) / "status.json"
        if not path.is_file():
            return None
        return json.loads(path.read_text(encoding="utf-8"))

    def docker_entries(self):
        if not self.docker_log.is_file():
            return []
        return [json.loads(line) for line in self.docker_log.read_text(
            encoding="utf-8").splitlines() if line.strip()]

    def docker_runs(self):
        return [entry for entry in self.docker_entries() if entry["kind"] == "run"]

    def evaluator_check(self, status, identifier):
        for check in status["evaluator"]["checks"]:
            if check["id"] == identifier:
                return check
        raise AssertionError("missing evaluator check %s" % identifier)


class DocsCiAdversarialTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not LAUNCHER.is_file():
            raise unittest.SkipTest(
                "integration-pending: scripts/docs-ci.py (B5-W1-A) is absent; "
                "set DOCS_CI_LAUNCHER to an integrated copy to run these fixtures")
        if not hasattr(os, "getuid") or os.getuid() == 0:
            raise unittest.SkipTest(
                "launcher refuses a root host; root behavior is not simulated")

    def fixture(self):
        return DocsCiFixture(self)

    def assert_blocked(self, fixture, result, expected_result, expected_reason):
        self.assertEqual(result.code, 2, result.payload)
        self.assertIsNotNone(result.payload)
        self.assertEqual(result.payload.get("result"), expected_result, result.payload)
        self.assertEqual(result.payload.get("reason"), expected_reason, result.payload)
        self.assertEqual(fixture.docker_entries(), [],
                         "no container may run before this refusal")
        status = fixture.status()
        if status is not None:
            self.assertEqual(status["result"], expected_result)
            self.assertFalse(status["authoritative"])

    # ------------------------------------------------- authoritative verdict

    def test_baseline_run_is_authoritative_and_identity_bound(self):
        fixture = self.fixture()
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(result.payload["result"], "pass")
        status = fixture.status()
        self.assertEqual(status["result"], "pass")
        self.assertTrue(status["authoritative"])
        self.assertFalse(status["blocking_release"])
        identity = status["identity"]
        self.assertEqual(identity["base_sha"], SHA_BASE)
        self.assertEqual(identity["candidate_sha"], SHA_CAND)
        self.assertEqual(identity["candidate_ref"], PULL_REF)
        self.assertEqual(identity["image"], IMAGE)
        self.assertEqual(identity["policy_source"], "argument-image")
        self.assertEqual(identity["required_pages"], ["start", "guide"])
        self.assertEqual(status["artifact"]["escapes"], [])
        self.assertEqual(len(fixture.evaluate_calls), 1,
                         "the base evaluator must be a separate invocation")
        self.assertTrue(status["evaluator"]["after_candidate_exit"])

    def test_forged_candidate_success_is_never_authoritative(self):
        forged = json.dumps({"schema": "candidate-report/1", "result": "pass",
                             "authoritative": True, "checks": {"build": "pass"}})
        fixture = self.fixture()
        fixture.configure(artifact={"files": {"candidate-report.json": forged},
                                    "dirs": [], "symlinks": {}, "hardlinks": {}})
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        status = fixture.status()
        self.assertEqual(status["result"], "fail")
        self.assertFalse(status["authoritative"])
        self.assertTrue(status["candidate"]["executed"])
        self.assertEqual(status["candidate"]["checks"][0]["exit_code"], 0)
        report = status["candidate"]["report"]
        self.assertTrue(report["present"])
        self.assertTrue(report["valid"])
        self.assertEqual(report["claimed_result"], "pass")
        self.assertFalse(report["authoritative"])
        self.assertEqual(fixture.evaluator_check(status, "candidate-completion")["status"],
                         "pass")
        self.assertEqual(fixture.evaluator_check(status, "artifact-pages")["status"], "fail")
        self.assertEqual(status["reason"], "base-evaluator-failed")

    def test_candidate_prebuilt_dist_and_status_cannot_be_promoted(self):
        forged_status = json.dumps({
            "schema": "docs-ci-status/1", "mode": "pr", "result": "pass",
            "authoritative": True, "blocking_release": False,
        })
        fixture = self.fixture()
        fixture.candidate_file("website/dist/index.html", "candidate prebuilt home\n")
        fixture.candidate_file("website/dist/guide/index.html", "candidate prebuilt guide\n")
        fixture.candidate_file("website/dist/status.json", forged_status)
        fixture.configure(artifact={"files": {}, "dirs": [], "symlinks": {},
                                    "hardlinks": {}})
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        status = fixture.status()
        self.assertEqual(status["result"], "fail")
        self.assertFalse(status["authoritative"])
        self.assertEqual(fixture.evaluator_check(status, "artifact-pages")["status"], "fail")
        # The candidate's prebuilt tree was never mounted; only the launcher's
        # declared write path was scanned, and its own status is the only verdict.
        artifact = fixture.output / "private/writes" / SITE_ROOT / "dist"
        self.assertFalse((artifact / "index.html").exists())
        self.assertFalse((artifact / "status.json").exists())

    # --------------------------------------------- candidate policy attempts

    def test_candidate_workflow_edit_requires_trusted_policy_review(self):
        cases = {
            "permissions": (
                "name: ci\non: pull_request_target\npermissions: write-all\njobs: {}\n"),
            # The invoked check list/argv lives in the reviewed workflow step.
            "check-invocation": (
                "name: docs\non: pull_request_target\njobs:\n  docs:\n    steps:\n"
                "      - run: python3 scripts/docs-ci.py --policy candidate/policy.json\n"),
        }
        for label, content in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                fixture.candidate_file(".github/workflows/ci.yml", content)
                self.assert_blocked(fixture, fixture.run_pr(),
                                    "needs-trusted-policy-review",
                                    "candidate-changed-protected-path")

    def test_candidate_launcher_edit_requires_trusted_policy_review(self):
        fixture = self.fixture()
        fixture.candidate_file("scripts/docs-ci.py", "# candidate-owned launcher\n")
        self.assert_blocked(fixture, fixture.run_pr(),
                            "needs-trusted-policy-review",
                            "candidate-changed-protected-path")

    def test_candidate_validator_edit_requires_trusted_policy_review(self):
        fixture = self.fixture()
        policy = fixture.write_policy(
            protected_paths=[".github/workflows", "scripts/docs-ci.py",
                             "scripts/validate.py"])
        fixture.candidate_file("scripts/validate.py", "# candidate validators pass by default\n")
        result = fixture.run_pr(extra=["--policy", str(policy)])
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "candidate-changed-protected-path")

    def test_candidate_required_page_removal_or_reclassification_is_review(self):
        cases = {
            "removed": (lambda pages: pages.pop(1), "required-page-removed"),
            "reclassified": (lambda pages: pages[1].update({"class": "D"}),
                             "required-page-reclassified"),
            "source-moved": (lambda pages: pages[1].update(
                {"source": "website/src/content/docs/moved.md"}),
                "required-page-reclassified"),
        }
        for label, (mutate, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                fixture.mutate_manifest(lambda data: mutate(data["pages"]))
                self.assert_blocked(fixture, fixture.run_pr(),
                                    "needs-trusted-policy-review", reason)

    def test_empty_candidate_page_set_requires_policy_review(self):
        fixture = self.fixture()
        fixture.mutate_manifest(lambda data: data.update({"pages": []}))
        self.assert_blocked(fixture, fixture.run_pr(),
                            "needs-trusted-policy-review", "candidate-manifest-empty")

    def test_candidate_additive_manifest_page_requires_trusted_adoption(self):
        """Q2: the trusted manifest is the allowlist, not a minimum page set."""
        fixture = self.fixture()
        fixture.mutate_manifest(lambda data: data["pages"].append({
            "id": "extra", "class": "L",
            "source": "website/src/content/docs/extra.md"}))
        result = fixture.run_pr()
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "candidate-manifest-unadopted-page")
        self.assertEqual(result.payload["detail"], "extra")

    def test_trusted_base_adoption_of_a_new_page_is_built_and_allowed(self):
        """The same addition passes once the trusted base adopts it."""
        fixture = self.fixture()
        page = {"id": "extra", "class": "L",
                "source": "website/src/content/docs/extra.md"}
        for root in (fixture.trusted, fixture.candidate):
            path = root / "website/docs-manifest.json"
            data = json.loads(path.read_text(encoding="utf-8"))
            data["pages"].append(copy.deepcopy(page))
            path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
            source = root / page["source"]
            source.write_text("# Extra\n", encoding="utf-8")
        artifact = default_artifact()
        artifact["files"]["extra/index.html"] = "<!doctype html><title>Extra</title>\n"
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        status = fixture.status()
        self.assertTrue(status["authoritative"])
        self.assertEqual(status["identity"]["required_pages"], ["start", "guide", "extra"])
        self.assertEqual(fixture.evaluator_check(status, "artifact-pages")["status"], "pass")

    def test_candidate_supplied_policy_or_check_list_is_refused(self):
        fixture = self.fixture()
        injected = fixture.write_candidate_policy({
            "checks": [{"id": "build", "argv": ["npm", "run", "candidate-defined"]}],
        })
        result = fixture.run_pr(extra=["--policy", str(injected)])
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "policy-outside-trusted-base")
        self.assertEqual(fixture.docker_entries(), [])
        self.assertIsNone(fixture.status(), "a pre-output refusal writes no verdict")

    # ------------------------------------------------- dependency identity

    def test_lockfile_drift_stops_before_any_container(self):
        fixture = self.fixture()
        fixture.candidate_file("website/package-lock.json",
                               json.dumps({"lockfileVersion": 3,
                                           "packages": {"evil": {}}}) + "\n")
        self.assert_blocked(fixture, fixture.run_pr(),
                            "needs-trusted-dependency-review",
                            "dependency-lock-drift")

    def test_candidate_build_script_change_requires_dependency_review(self):
        fixture = self.fixture()
        package = copy.deepcopy(PACKAGE)
        package["scripts"]["build"] = "npm run candidate-defined-check"
        fixture.candidate_file("website/package.json", json.dumps(package) + "\n")
        self.assert_blocked(fixture, fixture.run_pr(),
                            "needs-trusted-dependency-review",
                            "dependency-lock-drift")

    def test_missing_image_pin_executes_nothing(self):
        fixture = self.fixture()
        result = fixture.run_pr(image=None)
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "image-pin-not-adopted")
        self.assertEqual(fixture.docker_entries(), [])

    # ---------------------------------------------------------- git identity

    def test_wrong_base_sha_is_blocked_before_execution(self):
        fixture = self.fixture()
        result = fixture.run_pr(base_sha=SHA_FOREIGN)
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "base-sha-mismatch")
        self.assertEqual(fixture.docker_entries(), [])
        status = fixture.status()
        self.assertEqual(status["result"], "blocked")
        self.assertFalse(status["authoritative"])

    def test_candidate_ref_mismatch_is_blocked(self):
        fixture = self.fixture()
        fixture.git_state["refs"][PULL_REF] = SHA_MOVED
        fixture.write_git_state()
        self.assert_blocked(fixture, fixture.run_pr(), "blocked",
                            "candidate-ref-mismatch")

    def test_head_sha_moving_after_build_fails_evaluation(self):
        fixture = self.fixture()
        first = fixture.run_pr()
        self.assertEqual(first.code, 0, first.payload)
        self.assertTrue(fixture.status()["authoritative"])

        fixture.git_state["roots"][str(fixture.candidate)] = SHA_MOVED
        fixture.git_state["refs"][PULL_REF] = SHA_MOVED
        fixture.write_git_state()

        second = fixture.run(fixture.pr_argv(extra=["--action", "evaluate"]))
        self.assertEqual(second.code, 1, second.payload)
        self.assertEqual(second.payload["result"], "fail")
        status = fixture.status()
        self.assertEqual(status["result"], "fail")
        self.assertFalse(status["authoritative"])
        check = fixture.evaluator_check(status, "identity-consistency")
        self.assertEqual(check["status"], "fail")
        self.assertIn("candidate-sha-mismatch", check.get("detail", ""))

    # ---------------------------------------- constructed execution boundary

    def test_check_workdir_follows_the_policy_site_root(self):
        fixture = self.fixture()
        launcher = fixture.launcher
        policy = copy.deepcopy(launcher.DEFAULT_POLICY)
        self.assertEqual(launcher.check_workdir(policy), "/workspace/" + SITE_ROOT)

        policy["site_root"] = "docs-site"
        policy["site_manifest"] = "docs-site/docs-manifest.json"
        policy["dependencies_dir"] = "docs-site/node_modules"
        policy["artifact_root"] = "docs-site/dist"
        policy["dependency_files"] = ["docs-site/package.json"]
        policy["declared_write_paths"] = ["docs-site/dist"]
        launcher.validate_policy(policy)
        self.assertEqual(launcher.check_workdir(policy), "/workspace/docs-site")

        # A check whose script comes from a package must find that package in
        # the staged site root, not in the workspace mount point.
        staged = fixture.root / "staged-workdir"
        (staged / SITE_ROOT).mkdir(parents=True)
        with self.assertRaises(launcher.Refusal) as caught:
            launcher.assert_check_workdir(staged, launcher.DEFAULT_POLICY,
                                           policy["checks"][0])
        self.assertEqual(caught.exception.reason, "check-workdir-package-missing")

    def test_candidate_containers_cannot_reach_network_home_socket_or_cache(self):
        fixture = self.fixture()
        hostile = {
            "GITHUB_TOKEN": "ghs_fixture_forbidden_value",
            "ACTIONS_ID_TOKEN_REQUEST_TOKEN": "oidc_fixture_forbidden_value",
            "NPM_CONFIG_REGISTRY": "https://registry.fixture.invalid/",
            "NODE_OPTIONS": "--require=/tmp/fixture-evil.js",
            "DOCKER_HOST": "unix:///tmp/fixture-docker.sock",
            "SSH_AUTH_SOCK": "/tmp/fixture-agent.sock",
            "HOMEBREW_TAP_TOKEN": "brew_fixture_forbidden_value",
        }
        with mock.patch.dict(os.environ, hostile):
            result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)

        runs = fixture.docker_runs()
        self.assertEqual(len(runs), 1 + len(ADOPTED_CHECKS), runs)
        provision = next(entry for entry in runs if entry["command"][:2] == ["npm", "ci"])
        build = next(entry for entry in runs
                     if entry["command"] == ADOPTED_CHECKS[0]["argv"])
        self.assertEqual(provision["network"], "bridge")
        for entry in runs:
            if entry is provision:
                continue
            self.assertEqual(entry["network"], "none", entry["command"])

        for flag in ("--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges"):
            self.assertIn(flag, build["argv"])
        self.assertIn("--user=%d:%d" % (os.getuid(), os.getgid()), build["argv"])
        self.assertTrue(any(item.startswith("--pids-limit=") for item in build["argv"]))
        # The check runs where the policy's package.json lives, not at the
        # workspace mount root (which has no package manifest at all).
        self.assertEqual(build["workdir"], "/workspace/" + SITE_ROOT)

        self.assertEqual({mount["dst"] for mount in build["mounts"]}, {
            "/workspace", "/workspace/website/node_modules",
            "/workspace/website/dist", "/workspace/website/.cache",
            "/workspace/website/.astro", "/workspace/.cache",
        })
        self.assertEqual({mount["dst"] for mount in provision["mounts"]}, {"/provision"})
        for entry in runs:
            for mount in entry["mounts"]:
                source = Path(mount["src"]).resolve()
                self.assertTrue(str(source).startswith(str(fixture.root)), mount)
                self.assertNotIn(str(Path.home()), str(source), mount)
                self.assertNotIn("docker.sock", str(source), mount)
                self.assertNotIn("/.ssh", str(source), mount)
                self.assertNotIn("/.aws", str(source), mount)
                if mount["writable"]:
                    allowed = ((fixture.output / "private/provision").resolve(),
                               (fixture.output / "private/writes").resolve())
                    self.assertTrue(
                        any(source == root or root in source.parents for root in allowed),
                        "writable mount outside launcher-owned scratch: %r" % (mount,))
        staged = next(item for item in build["mounts"] if item["dst"] == "/workspace")
        self.assertEqual(Path(staged["src"]).resolve(),
                         (fixture.output / "private/staged").resolve())
        self.assertFalse(staged["writable"])
        for entry in runs:
            for mount in entry["mounts"]:
                self.assertNotEqual(Path(mount["src"]).resolve(), fixture.candidate)

        allowed = set(fixture.launcher.ALLOWED_CONTAINER_ENV)
        self.assertTrue(set(build["env_map"]) <= allowed, build["env_map"])
        self.assertFalse(any(key.startswith(("ACTIONS_", "GITHUB_", "NPM_CONFIG_",
                                             "npm_config_")) for key in build["env_map"]))
        for key in ("NODE_OPTIONS", "NODE_PATH", "DOCKER_HOST", "DOCKER_CONFIG",
                    "GITHUB_TOKEN", "GH_TOKEN", "HOMEBREW_TAP_TOKEN"):
            self.assertNotIn(key, build["env_map"])
        self.assertEqual(build["env_map"]["HOME"], "/tmp/home")
        self.assertEqual(build["workdir"], "/workspace/" + SITE_ROOT)

        for entry in fixture.docker_entries():
            self.assertEqual(set(entry["env"]), {"PATH", "HOME", "TMPDIR", "LC_ALL"})
            self.assertEqual(entry["env"]["HOME"], str(fixture.output / "private"))
            blob = json.dumps(entry, sort_keys=True)
            for value in hostile.values():
                self.assertNotIn(value, blob)
            self.assertNotIn("DOCKER_HOST", blob)
            self.assertNotIn("NPM_CONFIG_REGISTRY", blob)

    def test_candidate_inputs_cannot_smuggle_credentials_or_links(self):
        cases = {
            ".env": (lambda fx: fx.candidate_file(".env", "GITHUB_TOKEN=fixture\n"),
                     "credential-file-in-source"),
            ".npmrc": (lambda fx: fx.candidate_file("website/.npmrc",
                                                    "registry=https://fixture.invalid/\n"),
                       "credential-file-in-source"),
            # scan_tree refuses a source symlink as a non-regular entry.
            "symlink": (lambda fx: fx.candidate_symlink(
                "website/src/content/docs/leaked.md", "/etc/passwd"),
                "source-not-regular"),
            "sensitive-dir": (lambda fx: fx.candidate_file(
                ".ssh/id_ed25519", "fixture-private-key\n"),
                "sensitive-source-directory"),
            "special-file": (lambda fx: os.mkfifo(fx.candidate / "website/pipe"),
                             "source-not-regular"),
        }
        for label, (prepare, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                prepare(fixture)
                result = fixture.run_pr()
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["result"], "blocked")
                self.assertEqual(result.payload["reason"], reason)
                self.assertEqual(fixture.docker_runs(), [],
                                 "candidate bytes never reach a container")
                status = fixture.status()
                self.assertFalse(status["authoritative"])
                self.assertFalse(status["candidate"]["executed"])

    def test_documented_local_only_trees_are_skipped_with_an_observation(self):
        """`.codegraph/` and `backups/` exist only in a developer checkout."""
        fixture = self.fixture()
        # The local index holds a real database and even a credential-shaped
        # file; private backups hold skills. Neither is reviewed input, and
        # descending into them would abort a legitimate local run.
        fixture.candidate_file(".codegraph/index.sqlite3", "sqlite fixture\n")
        fixture.candidate_file(".codegraph/credentials.json", "{}\n")
        fixture.candidate_file(".codegraph/socket", "not a socket\n")
        os.mkfifo(fixture.candidate / ".codegraph" / "collector.pipe")
        fixture.candidate_file("backups/skills/private/SKILL.md", "# private\n")
        # Repository-root `bin/` is gitignored local build output, not a page.
        fixture.candidate_file("bin/agentic-sdd", "binary fixture\n")
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        status = fixture.status()
        self.assertEqual(status["result"], "pass")
        self.assertTrue(status["authoritative"])
        observations = " ".join(status["observations"])
        self.assertIn("skipped local-only directory .codegraph", observations)
        self.assertIn("skipped local-only directory backups", observations)
        self.assertIn("skipped local-only directory bin", observations)
        records = json.loads((fixture.output / "records.json").read_text(encoding="utf-8"))
        self.assertEqual(sorted(records["observations"]), sorted(status["observations"]))
        self.assertTrue(records["observations"])
        # Nothing from those trees was staged into the container workspace.
        staged = fixture.output / "private/staged"
        self.assertFalse((staged / ".codegraph").exists())
        self.assertFalse((staged / "backups").exists())
        self.assertFalse((staged / "bin").exists())

    def test_local_only_skip_does_not_hide_real_credential_files(self):
        """The skip is directory-scoped; a credential file elsewhere still blocks."""
        cases = {
            "env-beside-local-tree": (".env", "credential-file-in-source"),
            "npmrc-in-site": ("website/.npmrc", "credential-file-in-source"),
            "sqlite-in-site": ("website/local.sqlite", "credential-file-in-source"),
            "private-key-dir": (".gnupg/secring.gpg", "sensitive-source-directory"),
        }
        for label, (relative, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                fixture.candidate_file(".codegraph/index.sqlite3", "sqlite fixture\n")
                fixture.candidate_file("backups/skills/private/SKILL.md", "# private\n")
                fixture.candidate_file(relative, "fixture credential\n")
                result = fixture.run_pr()
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["reason"], reason)
                self.assertEqual(fixture.docker_runs(), [])

    # --------------------------------------------------- container ownership

    def test_runner_timeout_still_removes_every_owned_container(self):
        """A timed-out command leaves a running container; it must be removed."""
        for label, program in (("provision", "npm ci"), ("candidate-check", "npm run")):
            with self.subTest(case=label):
                fixture = self.fixture()
                injected = []

                def timeout_runner(argv, *, cwd=None, env=None, timeout=None,
                                   _program=program, _fixture=fixture, _injected=injected):
                    argv = [os.fspath(item) for item in argv]
                    if (Path(argv[0]).name == "docker" and argv[1:2] == ["run"]
                            and _program in " ".join(argv)):
                        # A real container starts before the client deadline:
                        # the cidfile exists while the container still runs.
                        cidfile = Path(argv[argv.index("--cidfile") + 1])
                        cidfile.write_text("cd" * 32, encoding="utf-8")
                        _injected.append(cidfile)
                        raise _fixture.launcher.RunnerTimeout("bounded command timed out")
                    return _fixture.runner(argv, cwd=cwd, env=env, timeout=timeout)

                context = fixture.launcher.Context(runner=timeout_runner)
                result = fixture.run(fixture.pr_argv(), context=context)
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["result"], "blocked")
                self.assertEqual(result.payload["reason"], "bounded-command-timeout")
                self.assertEqual(len(injected), 1, "the timeout was not injected")
                removals = [entry for entry in fixture.docker_entries()
                            if entry["kind"] == "rm"]
                self.assertTrue(removals,
                                "the timed-out container was never removed in finally")
                self.assertTrue(
                    any("cd" * 32 in entry["argv"] for entry in removals),
                    "cleanup did not target the timed-out container id: %r" % (removals,))
                self.assertEqual(injected[0].parent, fixture.output / "private",
                                 "the cidfile is not in launcher-owned scratch")
                self.assertIn(injected[0].name, ("provision.cid", "check-build.cid"),
                              "the cidfile is not one the launcher records")

    def test_cleanup_uncertainty_is_still_a_blocked_run(self):
        """A container the launcher cannot remove is a blocked run, not a pass."""
        fixture = self.fixture()

        def failing_rm(argv, *, cwd=None, env=None, timeout=None):
            argv = [os.fspath(item) for item in argv]
            if Path(argv[0]).name == "docker" and argv[1:2] == ["rm"]:
                return fixture.launcher.Completed(1, b"", b"permission denied")
            return fixture.runner(argv, cwd=cwd, env=env, timeout=timeout)

        context = fixture.launcher.Context(runner=failing_rm)
        result = fixture.run(fixture.pr_argv(), context=context)
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["reason"], "owned-container-cleanup-uncertain")

    # ------------------------------------------------------ artifact escapes

    def test_artifact_symlink_fails(self):
        fixture = self.fixture()
        artifact = default_artifact()
        artifact["symlinks"] = {"escape.html": "../../../etc/passwd"}
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        status = fixture.status()
        self.assertEqual(status["result"], "fail")
        self.assertFalse(status["authoritative"])
        check = fixture.evaluator_check(status, "artifact-integrity")
        self.assertEqual(check["status"], "fail")
        self.assertIn("symlink", check.get("detail", ""))

    def test_artifact_hardlink_fails(self):
        fixture = self.fixture()
        probe = fixture.root / "hardlink-probe-a"
        probe.write_text("probe\n", encoding="utf-8")
        try:
            os.link(probe, fixture.root / "hardlink-probe-b")
        except OSError as error:
            self.skipTest("fixture filesystem does not support hardlinks: %s" % error)
        artifact = default_artifact()
        artifact["hardlinks"] = {"duplicate.html": "index.html"}
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        status = fixture.status()
        check = fixture.evaluator_check(status, "artifact-integrity")
        self.assertEqual(check["status"], "fail")
        self.assertIn("hardlink", check.get("detail", ""))
        self.assertFalse(status["authoritative"])

    def test_artifact_empty_or_partial_page_set_fails(self):
        cases = {
            "empty": ({}, "required-pages-or-artifact-unavailable", None),
            "partial": ({"index.html": "<!doctype html><title>Home</title>\n"},
                        "guide", "start"),
        }
        for label, (files, detail, absent) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                fixture.configure(artifact={"files": files, "dirs": [],
                                            "symlinks": {}, "hardlinks": {}})
                result = fixture.run_pr()
                self.assertEqual(result.code, 1, result.payload)
                status = fixture.status()
                self.assertEqual(fixture.evaluator_check(
                    status, "artifact-integrity")["status"], "pass")
                pages = fixture.evaluator_check(status, "artifact-pages")
                self.assertEqual(pages["status"], "fail")
                detail_text = pages.get("detail", "")
                self.assertIn(detail, detail_text)
                if absent:
                    self.assertNotIn(absent, detail_text)
                self.assertFalse(status["authoritative"])

    def test_artifact_private_and_history_paths_fail(self):
        cases = {
            ".git/config": "forbidden-directory",
            "docs/handovers/2026-11-14_020.md": "forbidden-source",
            "docs/testing/report.md": "forbidden-source",
            "specs/020-documentation-ecosystem/spec.md": "forbidden-source",
            ".agentic-sdd/runs/fixture/status.json": "forbidden-source",
            "backups/skills/fixture/SKILL.md": "forbidden-source",
        }
        for path, kind in cases.items():
            with self.subTest(path=path):
                fixture = self.fixture()
                artifact = default_artifact()
                artifact["files"][path] = "fixture history\n"
                fixture.configure(artifact=artifact)
                result = fixture.run_pr()
                self.assertEqual(result.code, 1, result.payload)
                status = fixture.status()
                check = fixture.evaluator_check(status, "artifact-integrity")
                self.assertEqual(check["status"], "fail")
                self.assertIn(kind, check.get("detail", ""))
                self.assertFalse(status["authoritative"])

    # --------------------------------------------------- release and rollback

    def test_retained_main_artifact_verifies_for_an_explicit_prior_revision(self):
        fixture = self.fixture()
        main_output = fixture.root / "main-output"
        main = fixture.run(fixture.mode_argv("main", output=main_output, base_sha=SHA_BASE,
                                             extra=["--run-id", "424242"]))
        self.assertEqual(main.code, 0, main.payload)
        main_status = json.loads((main_output / "status.json").read_text(encoding="utf-8"))
        self.assertTrue(main_status["authoritative"])
        self.assertEqual(main_status["run"]["id"], "424242")

        retained = fixture.root / "retained"
        retained.mkdir()
        shutil.copytree(main_output / "private/writes" / SITE_ROOT / "dist",
                        retained / "site")
        shutil.copyfile(main_output / "status.json", retained / "status.json")

        # Trusted main has moved on since the retained build. A rollback
        # restores the older revision, so the record is bound to that explicit
        # revision (and run id) - never to the current HEAD.
        fixture.git_state["roots"][str(fixture.trusted)] = SHA_MOVED
        fixture.write_git_state()

        release_output = fixture.root / "release-output"
        release = fixture.run(fixture.mode_argv("release", output=release_output,
                                               artifact=retained,
                                               extra=["--expected-revision", SHA_BASE]))
        self.assertEqual(release.code, 0, release.payload)
        self.assertEqual(release.payload["result"], "observed")
        release_status = json.loads(
            (release_output / "status.json").read_text(encoding="utf-8"))
        self.assertEqual(release_status["result"], "observed")
        self.assertFalse(release_status["blocking_release"])
        self.assertFalse(release_status["authoritative"])
        self.assertEqual(release_status["evaluator"]["checks"][0]["status"], "pass")

        rollback_output = fixture.root / "rollback-output"
        rollback = fixture.run(fixture.mode_argv(
            "rollback", output=rollback_output, artifact=retained,
            extra=["--expected-revision", SHA_BASE, "--artifact-run", "424242"]))
        self.assertEqual(rollback.code, 0, rollback.payload)
        rollback_status = json.loads(
            (rollback_output / "status.json").read_text(encoding="utf-8"))
        self.assertEqual(rollback_status["result"], "pass")
        self.assertFalse(rollback_status["blocking_release"])
        self.assertFalse(rollback_status["authoritative"])

    def test_retained_flat_site_with_downloaded_record_verifies(self):
        """The rollback job downloads the site flat and the record separately."""
        fixture = self.fixture()
        main_output = fixture.root / "main-output"
        published = fixture.root / "published"
        main = fixture.run(fixture.mode_argv("main", output=main_output, base_sha=SHA_BASE,
                                             artifact=published,
                                             extra=["--run-id", "5150"]))
        self.assertEqual(main.code, 0, main.payload)
        record = fixture.root / "record"
        record.mkdir()
        shutil.copyfile(main_output / "status.json", record / "status.json")

        rollback_output = fixture.root / "rollback-output"
        rollback = fixture.run(fixture.mode_argv(
            "rollback", output=rollback_output, artifact=published,
            extra=["--record", str(record / "status.json"),
                   "--expected-revision", SHA_BASE, "--artifact-run", "5150"]))
        self.assertEqual(rollback.code, 0, rollback.payload)
        rollback_status = json.loads(
            (rollback_output / "status.json").read_text(encoding="utf-8"))
        self.assertEqual(rollback_status["result"], "pass")

    def test_rollback_refuses_forged_foreign_or_unbound_artifacts(self):
        def build(fixture):
            main_output = fixture.root / "main-output"
            main = fixture.run(fixture.mode_argv("main", output=main_output,
                                                 base_sha=SHA_BASE,
                                                 extra=["--run-id", "424242"]))
            self.assertEqual(main.code, 0, main.payload)
            retained = fixture.root / "retained"
            retained.mkdir()
            shutil.copytree(main_output / "private/writes" / SITE_ROOT / "dist",
                            retained / "site")
            shutil.copyfile(main_output / "status.json", retained / "status.json")
            return retained

        cases = {
            "unbound": ([], "retained-artifact-revision-unbound"),
            "wrong-revision": (["--expected-revision", SHA_FOREIGN],
                               "retained-artifact-from-other-revision"),
            "wrong-run": (["--expected-revision", SHA_BASE, "--artifact-run", "999"],
                          "retained-artifact-run-mismatch"),
        }
        for label, (extra, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                artifact = build(fixture)
                output = fixture.root / "rollback-output"
                result = fixture.run(fixture.mode_argv("rollback", output=output,
                                                       artifact=artifact, extra=extra))
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["result"], "blocked")
                self.assertEqual(result.payload["reason"], reason)
                status = json.loads((output / "status.json").read_text(encoding="utf-8"))
                self.assertEqual(status["result"], "blocked")
                self.assertFalse(status["authoritative"])

        # A record that claims a different trusted revision than the bound one
        # is refused even when its own digest and run id are self-consistent.
        fixture = self.fixture()
        foreign = write_retained(fixture.root / "foreign", base_sha=SHA_FOREIGN,
                                 run_id="777")
        result = fixture.run(fixture.mode_argv(
            "rollback", output=fixture.root / "foreign-output", artifact=foreign,
            extra=["--expected-revision", SHA_BASE]))
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["reason"], "retained-artifact-from-other-revision")

    def test_main_run_publishes_only_the_validated_site_to_artifact(self):
        fixture = self.fixture()
        output = fixture.root / "main-output"
        published = fixture.root / "published"
        result = fixture.run(fixture.mode_argv("main", output=output, base_sha=SHA_BASE,
                                               artifact=published))
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(result.payload["artifact"], str(published))
        self.assertTrue(published.is_dir(), "run mode must populate --artifact")
        self.assertEqual(
            sorted(path.relative_to(published).as_posix()
                   for path in published.rglob("*") if path.is_file()),
            ["guide/index.html", "index.html", "sitemap.xml"])
        self.assertEqual([path for path in published.rglob("*") if path.is_symlink()], [])
        status = json.loads((output / "status.json").read_text(encoding="utf-8"))
        self.assertEqual(artifact_digest(published), status["artifact"]["sha256"])
        # The trust record stays in the evidence directory, never in the site.
        self.assertFalse((published / "status.json").exists())
        self.assertTrue((output / "status.json").is_file())
        self.assertTrue((output / "identity.json").is_file())

    def test_artifact_is_not_published_when_the_evaluator_fails(self):
        fixture = self.fixture()
        fixture.configure(build_exit=1)
        published = fixture.root / "published"
        result = fixture.run(fixture.mode_argv("main", output=fixture.root / "main-output",
                                               base_sha=SHA_BASE, artifact=published))
        self.assertEqual(result.code, 1, result.payload)
        self.assertFalse(published.exists(), "a failing run must publish nothing")

    def test_pr_run_refuses_artifact_publication(self):
        fixture = self.fixture()
        published = fixture.root / "pr-artifact"
        result = fixture.run_pr(extra=["--artifact", str(published)])
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "artifact-only-for-trusted-main")
        self.assertFalse(published.exists())
        self.assertEqual(fixture.docker_runs(), [],
                         "refused before any candidate byte executes")

    def test_invalid_artifact_path_is_blocked_before_execution(self):
        fixture = self.fixture()
        result = fixture.run(fixture.mode_argv("main", output=fixture.root / "main-output",
                                               base_sha=SHA_BASE, artifact="relative/path"))
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "absolute-artifact-required")
        self.assertEqual(fixture.docker_runs(), [],
                         "a bad publication path is refused before any container")

    def test_release_mode_never_blocks_on_failed_docs_artifact(self):
        bound = ["--expected-revision", SHA_BASE]
        cases = {
            "missing": (lambda fx: fx.root / "absent", [], "retained-artifact-unavailable"),
            "expired": (lambda fx: write_retained(fx.root / "expired",
                                                  include_status=False),
                        [], "retained-artifact-unavailable"),
            "not-passing": (lambda fx: write_retained(fx.root / "failed", result="fail"),
                            [], "retained-status-not-passing"),
            "digest-mismatch": (lambda fx: write_retained(fx.root / "tampered",
                                                          digest="0" * 64),
                                bound, "retained-artifact-digest-mismatch"),
            "foreign-revision": (lambda fx: write_retained(fx.root / "foreign",
                                                           base_sha=SHA_FOREIGN),
                                 bound, "retained-artifact-from-other-revision"),
            "unbound": (lambda fx: write_retained(fx.root / "unbound"),
                        [], "retained-artifact-revision-unbound"),
        }
        for label, (prepare, extra, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                artifact = prepare(fixture)
                output = fixture.root / "release-output"
                result = fixture.run(fixture.mode_argv("release", output=output,
                                                       artifact=artifact, extra=extra))
                self.assertEqual(result.code, 0, result.payload)
                self.assertEqual(result.payload["result"], "observed")
                status = json.loads((output / "status.json").read_text(encoding="utf-8"))
                self.assertEqual(status["result"], "observed")
                self.assertFalse(status["blocking_release"])
                self.assertFalse(status["authoritative"])
                check = status["evaluator"]["checks"][0]
                self.assertEqual(check["status"], "fail")
                self.assertIn(reason, check.get("detail", ""))

    def test_release_without_artifact_argument_is_still_non_blocking(self):
        fixture = self.fixture()
        result = fixture.run(fixture.mode_argv(
            "release", output=fixture.root / "release-output"))
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "artifact-argument-required")

    def test_rollback_missing_or_expired_artifact_is_blocked(self):
        bound = ["--expected-revision", SHA_BASE]
        cases = {
            "missing": (lambda fx: fx.root / "absent", [], "retained-artifact-unavailable"),
            "expired": (lambda fx: write_retained(fx.root / "expired",
                                                  include_status=False),
                        [], "retained-artifact-unavailable"),
            "not-passing": (lambda fx: write_retained(fx.root / "failed", result="fail"),
                            [], "retained-status-not-passing"),
            "foreign-revision": (lambda fx: write_retained(fx.root / "foreign",
                                                           base_sha=SHA_FOREIGN),
                                 bound, "retained-artifact-from-other-revision"),
            "unbound": (lambda fx: write_retained(fx.root / "unbound"),
                        [], "retained-artifact-revision-unbound"),
        }
        for label, (prepare, extra, reason) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                artifact = prepare(fixture)
                output = fixture.root / "rollback-output"
                result = fixture.run(fixture.mode_argv("rollback", output=output,
                                                       artifact=artifact, extra=extra))
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["result"], "blocked")
                self.assertEqual(result.payload["reason"], reason)
                status = json.loads((output / "status.json").read_text(encoding="utf-8"))
                self.assertEqual(status["result"], "blocked")
                self.assertFalse(status["blocking_release"])
                self.assertFalse(status["authoritative"])

    # ------------------------------------- links observation (--mode links)

    def links_page(self, fixture):
        return fixture.trusted_file(
            "website/src/content/docs/external-links.md",
            "# External links\n\n"
            "[public](https://example.com/reference)\n\n"
            "[cdn](https://cdn.example.net/logo.png)\n\n"
            "[internal admin](http://127.0.0.1:8080/admin)\n\n"
            "[relative](../guide.md)\n\n"
            "```sh\n"
            "curl https://fenced.example.com/not-a-link\n"
            "```\n",
        )

    def test_links_mode_without_network_fetches_nothing_and_stays_observational(self):
        fixture = self.fixture()
        self.links_page(fixture)
        probe = mock.Mock(side_effect=AssertionError("no URL may be fetched"))
        with mock.patch.object(fixture.launcher, "probe_one_url", probe):
            result = fixture.run_links(fixture.root / "links-output")
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(result.payload["result"], "observed")
        self.assertFalse(result.payload["authoritative"])
        self.assertFalse(result.payload["blocking_release"])
        self.assertFalse(probe.called)
        document = json.loads(
            (fixture.root / "links-output" / "links.json").read_text(encoding="utf-8"))
        self.assertEqual(document["schema"], "docs-ci-links-observation/1")
        self.assertEqual(document["result"], "observed")
        self.assertFalse(document["authoritative"])
        self.assertFalse(document["blocking_release"])
        self.assertFalse(document["network_requested"])
        self.assertFalse(document["reachability"]["performed"])
        self.assertEqual(document["reachability"]["refused"], 0)
        self.assertEqual(document["reachability"]["results"], [])
        self.assertTrue(document["source"]["available"])
        urls = {entry["url"] for entry in document["urls"]}
        self.assertIn("https://example.com/reference", urls)
        # The loopback URL is inventoried as data and never probed.
        self.assertIn("http://127.0.0.1:8080/admin", urls)
        self.assertNotIn("https://fenced.example.com/not-a-link", urls)
        self.assertGreaterEqual(document["counts"]["unique_external_urls"], 3)

    def test_links_network_answers_cannot_change_result_or_blocking(self):
        fixture = self.fixture()
        self.links_page(fixture)
        unreachable = {"url": "https://example.com/reference", "status": "unreachable",
                       "http_status": 503, "error": "HTTPError", "refused": False}
        reachable = dict(unreachable, status="reachable", http_status=200, error="")
        documents = []
        for label, observation in (("down", unreachable), ("up", reachable)):
            with mock.patch.object(fixture.launcher, "probe_one_url",
                                   return_value=dict(observation)) as probe:
                result = fixture.run_links(fixture.root / ("links-" + label),
                                           network=True)
            self.assertEqual(result.code, 0, result.payload)
            self.assertEqual(result.payload["result"], "observed")
            self.assertFalse(result.payload["authoritative"])
            self.assertFalse(result.payload["blocking_release"])
            self.assertTrue(probe.called)
            document = json.loads(
                (fixture.root / ("links-" + label) / "links.json")
                .read_text(encoding="utf-8"))
            self.assertTrue(document["network_requested"])
            self.assertTrue(document["reachability"]["performed"])
            self.assertFalse(document["reachability"]["affects_result"])
            documents.append(document)
        down, up = documents
        # The network answer changes only the reachability block.
        self.assertNotEqual(down["reachability"]["results"],
                            up["reachability"]["results"])
        for key in ("result", "reason", "authoritative", "blocking_release",
                    "counts", "hosts", "urls", "occurrences"):
            self.assertEqual(down[key], up[key], key)

    def test_links_mode_output_is_deterministic(self):
        fixture = self.fixture()
        self.links_page(fixture)
        first = fixture.run_links(fixture.root / "links-a")
        second = fixture.run_links(fixture.root / "links-b")
        self.assertEqual(first.code, 0, first.payload)
        self.assertEqual(second.code, 0, second.payload)
        self.assertEqual((fixture.root / "links-a" / "links.json").read_bytes(),
                         (fixture.root / "links-b" / "links.json").read_bytes())

    def test_links_probe_refuses_non_http_and_local_targets_before_any_fetch(self):
        launcher = load_launcher(LAUNCHER)
        cases = {
            "file:///etc/passwd": "scheme-not-http",
            "ftp://example.com/file": "scheme-not-http",
            "javascript:alert(1)": "scheme-not-http",
            "//example.com/path": "scheme-not-http",
            "data:text/plain,hello": "scheme-not-http",
            "http://localhost:8080/admin": "local-target",
            "http://api.localhost/": "local-target",
            "http://printer.local/status": "local-target",
            "http://metadata.google.internal/computeMetadata/v1/": "local-target",
            "http://service.home.arpa/": "local-target",
            "http://127.0.0.1/": "local-target",
            "http://127.1.2.3/": "local-target",
            "http://10.1.2.3/": "local-target",
            "http://172.16.0.1/": "local-target",
            "http://192.168.1.1/": "local-target",
            "http://169.254.169.254/latest/meta-data/": "local-target",
            "http://0.0.0.0/": "local-target",
            "http://[::1]/": "local-target",
            "http://[fd00::1]/": "local-target",
            "http://[fe80::1]/": "local-target",
            "http://[::ffff:127.0.0.1]/": "local-target",
        }
        fetch = mock.Mock(side_effect=AssertionError("no socket may open"))
        with mock.patch.object(launcher, "_probe_fetch", fetch):
            for url, reason in cases.items():
                with self.subTest(url=url):
                    observed = launcher.probe_one_url(url, timeout=3)
                    self.assertEqual(observed["error"], reason)
                    self.assertTrue(observed["refused"])
                    self.assertEqual(observed["status"], "not-probed")
            self.assertFalse(fetch.called)
        self.assertEqual(launcher.probe_refusal("https://example.com/docs"), "")

    def test_links_probe_fetches_one_allowed_url_without_following_redirects(self):
        launcher = load_launcher(LAUNCHER)

        class Response:
            status = 200

            def read(self, _limit):
                return b""

            def __enter__(self):
                return self

            def __exit__(self, *exc):
                return False

        with mock.patch.object(launcher, "_probe_fetch",
                               return_value=Response()) as fetch:
            observed = launcher.probe_one_url("https://example.com/docs", timeout=3)
        self.assertEqual(observed["status"], "reachable")
        self.assertEqual(observed["http_status"], 200)
        self.assertFalse(observed["refused"])
        self.assertEqual(fetch.call_count, 1)

        # The opener replaces urllib's redirect handler: a 3xx answer is
        # recorded and the redirect target is never fetched.
        import urllib.error
        import urllib.request
        handlers = launcher._probe_opener().handlers
        types = {type(handler) for handler in handlers}
        self.assertNotIn(urllib.request.HTTPRedirectHandler, types)
        no_redirect = [handler for handler in handlers
                       if type(handler).__name__ == "_NoRedirect"]
        self.assertEqual(len(no_redirect), 1)
        self.assertIsNone(no_redirect[0].redirect_request(
            None, None, 302, "Found", {}, "https://example.net/target"))
        error = urllib.error.HTTPError("https://example.com/docs", 302, "Found", {}, None)
        self.addCleanup(error.close)
        with mock.patch.object(launcher, "_probe_fetch", side_effect=error) as fetch:
            observed = launcher.probe_one_url("https://example.com/docs", timeout=3)
        self.assertEqual(fetch.call_count, 1)
        self.assertEqual(observed["http_status"], 302)
        self.assertEqual(observed["error"], "redirect-not-followed")

    # ------------------------------------------------------------ host bound

    def test_root_host_is_refused_before_execution(self):
        fixture = self.fixture()
        context = fixture.launcher.Context(runner=fixture.runner, uid=0, gid=0,
                                           posix=True)
        result = fixture.run(fixture.pr_argv(), context=context)
        self.assertEqual(result.code, 2, result.payload)
        self.assertEqual(result.payload["result"], "blocked")
        self.assertEqual(result.payload["reason"], "non-root-host-required")
        self.assertEqual(fixture.docker_entries(), [])


    # ------------------------------- adopted validation surface and checks (Q1)

    def test_embedded_policy_protects_the_full_validation_surface(self):
        """The transitive validator/tooling surface is base-owned, not a hint."""
        launcher = load_launcher(LAUNCHER)
        self.assertEqual(list(launcher.DEFAULT_POLICY["protected_paths"]),
                         PROTECTED_PATHS)
        self.assertEqual(
            [{"id": check["id"], "argv": check["argv"]}
             for check in launcher.DEFAULT_POLICY["checks"]],
            ADOPTED_CHECKS,
            "the launcher's adopted check list and the fixture contract drifted")

    def test_candidate_edits_to_the_validation_surface_require_policy_review(self):
        cases = {
            "site-script": "website/scripts/check.mjs",
            "site-export": "website/scripts/export.mjs",
            "site-plugin": "website/plugins/remark-link-routes.mjs",
            "site-test": "website/tests/fixture.test.mjs",
            "astro-config": "website/astro.config.mjs",
            "content-config": "website/src/content.config.ts",
            "markdownlint": "website/.markdownlint.json",
            "taskfile": "Taskfile.yml",
            "harness-suite": "tests/harness/test_docs_ci.py",
        }
        for label, relative in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                fixture.candidate_file(relative, "candidate-owned override\n")
                result = fixture.run_pr()
                self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                                    "candidate-changed-protected-path")
                self.assertEqual(result.payload["detail"], relative)

    def test_dependency_files_keep_their_dependency_review_result(self):
        """A protected dependency file still reports the dependency verdict."""
        def package(data):
            data["scripts"]["build"] = "astro build --candidate-defined"

        def lockfile(data):
            data["packages"] = {"evil": {}}

        cases = {
            "package-json": ("website/package.json", package),
            "package-lock": ("website/package-lock.json", lockfile),
        }
        for label, (relative, mutate) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                path = fixture.candidate / relative
                data = json.loads(path.read_text(encoding="utf-8"))
                mutate(data)
                path.write_text(json.dumps(data) + "\n", encoding="utf-8")
                self.assert_blocked(fixture, fixture.run_pr(),
                                    "needs-trusted-dependency-review",
                                    "dependency-lock-drift")

    def test_adopted_checks_run_in_order_with_exact_argv_and_isolation(self):
        fixture = self.fixture()
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        runs = [entry for entry in fixture.docker_runs()
                if entry["command"][:2] != ["npm", "ci"]]
        self.assertEqual([entry["command"] for entry in runs],
                         [check["argv"] for check in ADOPTED_CHECKS])
        self.assertEqual([Path(entry["cidfile"]).name for entry in runs],
                         ["check-%s.cid" % check["id"] for check in ADOPTED_CHECKS])
        for entry in runs:
            self.assertEqual(entry["network"], "none", entry["command"])
            self.assertIn("--read-only", entry["argv"])
            self.assertIn("--cap-drop=ALL", entry["argv"])
            self.assertIn("--security-opt=no-new-privileges", entry["argv"])
        status = fixture.status()
        recorded = status["candidate"]["checks"]
        self.assertEqual([item["id"] for item in recorded],
                         [check["id"] for check in ADOPTED_CHECKS])
        self.assertEqual([item["argv"] for item in recorded],
                         [check["argv"] for check in ADOPTED_CHECKS])
        self.assertTrue(all(item["exit_code"] == 0 for item in recorded))
        records = json.loads((fixture.output / "records.json").read_text(encoding="utf-8"))
        self.assertEqual(records["adopted_checks"],
                         [check["id"] for check in ADOPTED_CHECKS])
        self.assertEqual(records["content_pages"], [
            "website/src/content/docs/guide.md",
            "website/src/content/docs/index.md",
        ])

    def test_missing_renamed_or_skipped_adopted_check_is_never_a_pass(self):
        def drop(checks):
            checks.pop(1)

        def rename(checks):
            checks[1]["id"] = "docs-check-skipped"

        def argv(checks):
            checks[1]["argv"] = ["npm", "run", "docs:check", "--fast"]

        def nonzero(checks):
            checks[1]["exit_code"] = 1

        def append(checks):
            checks.append({"id": "surprise", "argv": ["npm", "run", "surprise"],
                           "exit_code": 0, "log": "private/logs/surprise.log",
                           "log_sha256": "0" * 64})

        def clear(checks):
            checks.clear()

        cases = {
            "missing": (drop, "missing:docs-check"),
            "renamed": (rename, "unadopted:docs-check-skipped"),
            "argv": (argv, "argv:docs-check"),
            "nonzero": (nonzero, "exit:docs-check"),
            "extra": (append, "unadopted:surprise"),
            "empty": (clear, "no-executed-check-record"),
        }
        for label, (mutate, detail) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                first = fixture.run_pr()
                self.assertEqual(first.code, 0, first.payload)
                path = fixture.output / "records.json"
                records = json.loads(path.read_text(encoding="utf-8"))
                mutate(records["checks"])
                path.write_text(json.dumps(records) + "\n", encoding="utf-8")
                second = fixture.run(fixture.pr_argv(extra=["--action", "evaluate"]))
                self.assertEqual(second.code, 1, second.payload)
                status = fixture.status()
                check = fixture.evaluator_check(status, "candidate-completion")
                self.assertEqual(check["status"], "fail")
                self.assertIn(detail, check.get("detail", ""))
                self.assertFalse(status["authoritative"])

    def test_trusted_missing_check_target_is_a_policy_refusal(self):
        cases = {
            "script": ("website/package.json",
                       lambda path: path.write_text(json.dumps({
                           "name": "docs-ci-fixture",
                           "private": True,
                           "scripts": {"build": "astro build",
                                       "docs:check": "node scripts/check.mjs",
                                       "docs:check-built": "node scripts/check.mjs"},
                       }, indent=2) + "\n", encoding="utf-8"),
                       "trusted-check-script-missing", "test"),
            "file": ("website/scripts/export.mjs",
                     lambda path: path.unlink(),
                     "trusted-check-file-missing", "scripts/export.mjs"),
        }
        for label, (relative, mutate, reason, detail) in cases.items():
            with self.subTest(case=label):
                fixture = self.fixture()
                for root in (fixture.trusted, fixture.candidate):
                    mutate(root / relative)
                result = fixture.run_pr()
                self.assertEqual(result.code, 2, result.payload)
                self.assertEqual(result.payload["result"], "needs-trusted-policy-review")
                self.assertEqual(result.payload["reason"], reason)
                self.assertEqual(result.payload["detail"], detail)
                self.assertEqual(fixture.docker_entries(), [])

    # ------------------------------- manifest allowlist and content parity (Q2)

    def test_unregistered_page_in_the_tree_requires_policy_review(self):
        fixture = self.fixture()
        fixture.candidate_file("website/src/content/docs/private.md", "# Private\n")
        result = fixture.run_pr()
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "candidate-content-unregistered")
        self.assertEqual(result.payload["detail"],
                         "website/src/content/docs/private.md")

    def test_unexpected_content_extension_requires_policy_review(self):
        fixture = self.fixture()
        fixture.candidate_file("website/src/content/docs/guide.mdx", "# Guide\n")
        result = fixture.run_pr()
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "candidate-content-extension")
        self.assertEqual(result.payload["detail"],
                         "website/src/content/docs/guide.mdx")

    def test_adopted_page_without_source_requires_policy_review(self):
        fixture = self.fixture()
        (fixture.candidate / "website/src/content/docs/guide.md").unlink()
        result = fixture.run_pr()
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "adopted-content-missing")
        self.assertEqual(result.payload["detail"],
                         "website/src/content/docs/guide.md")

    def test_deferred_page_source_entering_the_tree_is_refused(self):
        def adopt(fixture):
            deferred = {"id": "deferred", "class": "D",
                        "source": "website/src/content/docs/deferred.md"}
            for root in (fixture.trusted, fixture.candidate):
                path = root / "website/docs-manifest.json"
                data = json.loads(path.read_text(encoding="utf-8"))
                data["pages"].append(copy.deepcopy(deferred))
                path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")

        # Adopted-but-deferred metadata alone is not built and does not publish.
        fixture = self.fixture()
        adopt(fixture)
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(fixture.status()["identity"]["required_pages"], ["start", "guide"])

        # The deferred source appearing in the tree is an unadopted build input.
        fixture = self.fixture()
        adopt(fixture)
        fixture.candidate_file("website/src/content/docs/deferred.md", "# Deferred\n")
        result = fixture.run_pr()
        self.assert_blocked(fixture, result, "needs-trusted-policy-review",
                            "candidate-content-unregistered")
        self.assertEqual(result.payload["detail"],
                         "website/src/content/docs/deferred.md")

    def test_extra_built_route_fails(self):
        fixture = self.fixture()
        artifact = default_artifact()
        artifact["files"]["private/secret/index.html"] = "<!doctype html><title>Extra</title>\n"
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        status = fixture.status()
        pages = fixture.evaluator_check(status, "artifact-pages")
        self.assertEqual(pages["status"], "fail")
        self.assertIn("unregistered-route:private/secret/index.html", pages["detail"])
        self.assertFalse(status["authoritative"])

    def test_missing_adopted_route_fails_with_its_page_id(self):
        fixture = self.fixture()
        fixture.configure(artifact={"files": {"404.html": "<!doctype html><title>404</title>\n"},
                                    "dirs": [], "symlinks": {}, "hardlinks": {}})
        result = fixture.run_pr()
        self.assertEqual(result.code, 1, result.payload)
        pages = fixture.evaluator_check(fixture.status(), "artifact-pages")
        self.assertEqual(pages["status"], "fail")
        self.assertIn("missing-route:start", pages["detail"])
        self.assertIn("missing-route:guide", pages["detail"])

    def test_framework_404_route_is_allowed_but_is_not_a_page(self):
        fixture = self.fixture()
        artifact = default_artifact()
        artifact["files"]["404.html"] = "<!doctype html><title>Missing</title>\n"
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(fixture.evaluator_check(fixture.status(), "artifact-pages")["status"],
                         "pass")

    def test_nested_section_index_route_maps_to_its_adopted_page(self):
        fixture = self.fixture()
        page = {"id": "section", "class": "L",
                "source": "website/src/content/docs/section/index.md"}
        for root in (fixture.trusted, fixture.candidate):
            path = root / "website/docs-manifest.json"
            data = json.loads(path.read_text(encoding="utf-8"))
            data["pages"].append(copy.deepcopy(page))
            path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
            (root / "website/src/content/docs/section").mkdir(parents=True, exist_ok=True)
            (root / page["source"]).write_text("# Section\n", encoding="utf-8")
        artifact = default_artifact()
        artifact["files"]["section/index.html"] = "<!doctype html><title>Section</title>\n"
        fixture.configure(artifact=artifact)
        result = fixture.run_pr()
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(fixture.evaluator_check(fixture.status(), "artifact-pages")["status"],
                         "pass")

    def test_source_missing_from_the_staged_tree_fails_evaluation(self):
        fixture = self.fixture()
        first = fixture.run_pr()
        self.assertEqual(first.code, 0, first.payload)
        (fixture.output / "private/staged/website/src/content/docs/guide.md").unlink()
        second = fixture.run(fixture.pr_argv(extra=["--action", "evaluate"]))
        self.assertEqual(second.code, 1, second.payload)
        status = fixture.status()
        check = fixture.evaluator_check(status, "content-inventory")
        self.assertEqual(check["status"], "fail")
        self.assertIn("adopted-content-missing", check.get("detail", ""))
        self.assertIn("website/src/content/docs/guide.md", check.get("detail", ""))
        self.assertFalse(status["authoritative"])

    # ------------------------------------------ dirty trusted tree honesty (Q5)

    def test_clean_trusted_main_tree_is_authoritative(self):
        fixture = self.fixture()
        output = fixture.root / "main-output"
        result = fixture.run(fixture.mode_argv("main", output=output, base_sha=SHA_BASE,
                                               extra=["--run-id", "4242"]))
        self.assertEqual(result.code, 0, result.payload)
        self.assertTrue(result.payload["authoritative"])
        status = json.loads((output / "status.json").read_text(encoding="utf-8"))
        self.assertTrue(status["authoritative"])
        self.assertEqual(status["authoritative_reason"], "")
        self.assertEqual(status["identity"]["trusted_tree"],
                         {"clean": True, "count": 0, "entries": [],
                          "reason": "", "recorded": True})
        self.assertFalse(status["preview"]["authoritative"])

    def test_dirty_trusted_main_tree_is_not_authoritative(self):
        fixture = self.fixture()
        fixture.git_state["dirty"] = [" M website/package.json",
                                      "?? website/dist/index.html"]
        fixture.write_git_state()
        output = fixture.root / "main-output"
        published = fixture.root / "published"
        result = fixture.run(fixture.mode_argv("main", output=output, base_sha=SHA_BASE,
                                               artifact=published,
                                               extra=["--run-id", "4242"]))
        self.assertEqual(result.code, 0, result.payload)
        self.assertEqual(result.payload["result"], "pass")
        self.assertFalse(result.payload["authoritative"])
        self.assertEqual(result.payload["authoritative_reason"], "dirty-trusted-tree")
        status = json.loads((output / "status.json").read_text(encoding="utf-8"))
        self.assertEqual(status["result"], "pass")
        self.assertFalse(status["authoritative"])
        self.assertEqual(status["authoritative_reason"], "dirty-trusted-tree")
        tree = status["identity"]["trusted_tree"]
        self.assertFalse(tree["clean"])
        self.assertEqual(tree["count"], 2)
        self.assertEqual(tree["entries"], [" M website/package.json",
                                           "?? website/dist/index.html"])
        observations = " ".join(status["observations"])
        self.assertIn("trusted main tree is not clean", observations)
        self.assertIn("local previews are not authoritative", observations)
        self.assertFalse(published.exists(),
                         "a non-authoritative run must not publish a site")

    def test_unreadable_trusted_tree_state_is_not_authoritative(self):
        fixture = self.fixture()
        fixture.git_state["status_exit"] = 2
        fixture.write_git_state()
        output = fixture.root / "main-output"
        result = fixture.run(fixture.mode_argv("main", output=output, base_sha=SHA_BASE,
                                               extra=["--run-id", "4242"]))
        self.assertEqual(result.code, 0, result.payload)
        self.assertFalse(result.payload["authoritative"])
        self.assertEqual(result.payload["authoritative_reason"],
                         "trusted-tree-state-unavailable")
        status = json.loads((output / "status.json").read_text(encoding="utf-8"))
        self.assertFalse(status["identity"]["trusted_tree"]["recorded"])
        self.assertFalse(status["authoritative"])

WORKFLOW_PATH = ROOT / ".github/workflows/docs.yml"


class DocsContainerHomeTests(unittest.TestCase):
    """The container user runs as `--user uid:gid`; its own `$HOME` must be writable.

    An argv-level assertion alone cannot prove this: the failure mode was a
    root-owned tmpfs, so the real pinned image is run when Docker is present.
    """

    @classmethod
    def setUpClass(cls):
        if not LAUNCHER.is_file():
            raise unittest.SkipTest("integration-pending: scripts/docs-ci.py is absent")
        cls.launcher = load_launcher(LAUNCHER)
        if shutil.which("docker") is None:
            raise unittest.SkipTest("docker client is not installed")
        if not hasattr(os, "getuid") or os.getuid() == 0:
            raise unittest.SkipTest(
                "a root container user cannot observe the root-owned-$HOME defect")
        probe = subprocess.run(["docker", "image", "inspect", DOCS_IMAGE],
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        if probe.returncode != 0:
            raise unittest.SkipTest("pinned runtime image is not present locally")

    def container_argv(self):
        launcher = self.launcher
        policy = copy.deepcopy(launcher.DEFAULT_POLICY)
        policy["image"] = DOCS_IMAGE
        launcher.validate_policy(policy)
        temporary = tempfile.TemporaryDirectory(prefix="docs-ci-home-")
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        return launcher.container_argv(
            launcher.Context(), policy, mounts=[], workdir="/tmp",
            environment=launcher.container_environment(policy), network="none",
            cidfile=root / "home.cid",
            command=["sh", "-c",
                     'test -w "$HOME" && touch "$HOME/probe" && stat -c %a "$HOME"'],
        ), root

    def test_container_home_tmpfs_is_writable_by_the_container_user(self):
        argv, root = self.container_argv()
        self.assertIn("/tmp/home:rw,nosuid,nodev,exec,mode=1777,size=64m", argv)
        completed = subprocess.run(argv, cwd=root, stdout=subprocess.PIPE,
                                   stderr=subprocess.PIPE, timeout=180)
        self.assertEqual(
            completed.returncode, 0,
            "the container user cannot write $HOME: %s"
            % completed.stderr.decode("utf-8", "replace").strip())
        self.assertEqual(completed.stdout.strip(), b"1777")

        # The regression is real: the same command with the old mode=700 tmpfs
        # fails, so this assertion detects the defect and is not vacuous.
        legacy = [item.replace("mode=1777,size=64m", "mode=700,size=64m")
                  if item.startswith("/tmp/home:") else item for item in argv]
        self.assertIn("/tmp/home:rw,nosuid,nodev,exec,mode=700,size=64m", legacy)
        blocked = subprocess.run(legacy, cwd=root, stdout=subprocess.PIPE,
                                 stderr=subprocess.PIPE, timeout=180)
        self.assertNotEqual(blocked.returncode, 0,
                            "mode=700 should make $HOME unwritable for the container user")


def workflow_job_blocks():
    """Split docs.yml into job bodies by indentation, without a YAML dependency.

    Only step order inside a job matters here: the wiring defect this guards is
    a launcher invocation that happens before the pinned image is present.
    """
    jobs = {}
    current = None
    in_jobs = False
    for line in WORKFLOW_PATH.read_text(encoding="utf-8").splitlines():
        if not in_jobs:
            if line.rstrip() == "jobs:":
                in_jobs = True
            continue
        if not line.strip():
            continue
        if not line.startswith(" "):
            break
        if len(line) - len(line.lstrip(" ")) == 2:
            body = line.strip()
            if body.startswith("#"):
                continue
            current = body.split(":", 1)[0]
            jobs[current] = []
            continue
        if current is not None:
            jobs[current].append(line)
    return jobs


def workflow_step_block(job_lines, step_name):
    """Lines of one step body, located by its `- name:` line and indentation."""
    start = None
    indent = None
    for index, line in enumerate(job_lines):
        if line.strip() == "- name: " + step_name:
            start = index
            indent = len(line) - len(line.lstrip(" "))
            break
    if start is None:
        return None
    block = []
    for line in job_lines[start + 1:]:
        stripped = line.lstrip(" ")
        if stripped.startswith("- ") and (len(line) - len(stripped)) == indent:
            break
        block.append(line)
    return block


def workflow_path_list(block, key="path"):
    """The values of a step's `path:` input, whether inline or a block scalar."""
    for index, line in enumerate(block):
        match = re.match(r"^(\s*)" + re.escape(key) + r":\s*(\S.*)?$", line)
        if not match:
            continue
        inline = (match.group(2) or "").strip()
        if inline and inline not in ("|", ">", "|-", ">-", "|+", ">+"):
            return [inline]
        base_indent = len(match.group(1))
        values = []
        for item in block[index + 1:]:
            if not item.strip():
                continue
            if len(item) - len(item.lstrip(" ")) <= base_indent:
                break
            values.append(item.strip())
        return values
    return None


class DocsWorkflowWiringTests(unittest.TestCase):
    """The workflow must prepare the runtime and pass real, existing paths."""

    @classmethod
    def setUpClass(cls):
        if not WORKFLOW_PATH.is_file():
            raise AssertionError("missing docs workflow: %s" % WORKFLOW_PATH)
        cls.text = WORKFLOW_PATH.read_text(encoding="utf-8")
        cls.jobs = workflow_job_blocks()

    def test_every_bounded_launcher_job_pulls_the_pinned_image_first(self):
        for job in ("pr-validate", "main-build", "rollback"):
            with self.subTest(job=job):
                lines = self.jobs.get(job)
                self.assertIsNotNone(lines, "job %s is missing" % job)
                pulls = [index for index, line in enumerate(lines)
                         if re.search(r'docker pull\s+"\$DOCS_IMAGE"', line)]
                launches = [index for index, line in enumerate(lines)
                            if "scripts/docs-ci.py" in line]
                self.assertTrue(launches, "job %s never invokes the launcher" % job)
                self.assertTrue(
                    pulls,
                    "job %s never pulls the pinned image, so a fresh runner "
                    "blocks at pinned-image-unavailable" % job)
                self.assertLess(pulls[0], launches[0],
                                "job %s pulls the image after the launcher" % job)

    def test_runtime_image_is_digest_pinned(self):
        match = re.search(r'^\s*DOCS_IMAGE:\s*"([^"]+)"', self.text, re.MULTILINE)
        self.assertIsNotNone(match, "DOCS_IMAGE is not declared")
        self.assertRegex(
            match.group(1),
            r"^[a-z0-9][a-z0-9._/-]*(?::[A-Za-z0-9._-]+)?@sha256:[0-9a-f]{64}$")

    def test_main_build_publishes_the_artifact_and_rollback_binds_the_record(self):
        main = "\n".join(self.jobs["main-build"])
        self.assertIn("--mode main", main)
        self.assertIn('--artifact "$ARTIFACT_DIR"', main)
        rollback = "\n".join(self.jobs["rollback"])
        self.assertIn('--record "$RECORD_FILE"', rollback)
        self.assertIn('--expected-revision "$EXPECTED_REVISION"', rollback)
        self.assertIn('--artifact-run "$ROLLBACK_RUN_ID"', rollback)
        self.assertIn('--run-id "$RUN_ID"', rollback)
        self.assertIn("name: docs-trust-record", rollback)
        self.assertIn("name: docs-trust-record", main)
        self.assertIn("--artifact", main)
        self.assertNotIn("--artifact", "\n".join(self.jobs["pr-validate"]))

    def test_launcher_still_never_pulls_inside_a_run(self):
        self.assertIn('"--pull=never"', LAUNCHER.read_text(encoding="utf-8"))

    def test_pr_job_creates_the_pull_ref_the_launcher_resolves(self):
        """actions/checkout leaves `refs/remotes/pull/<n>/head`, not the passed ref."""
        pr = "\n".join(self.jobs["pr-validate"])
        match = re.search(r'--candidate-ref\s+"([^"]+)"', pr)
        self.assertIsNotNone(match, "the PR lane passes no --candidate-ref")
        self.assertEqual(match.group(1), "refs/pull/$PR_NUMBER/head")
        block = workflow_step_block(self.jobs["pr-validate"],
                                    "Materialise the pull ref the launcher resolves")
        self.assertIsNotNone(
            block, "the PR job never creates the ref form it hands the launcher")
        text = "\n".join(block)
        self.assertIn('git -C "$CANDIDATE_DIR" update-ref '
                      '"refs/pull/$PR_NUMBER/head" HEAD', text)
        self.assertLess(pr.index("update-ref"), pr.index("--candidate-ref"),
                        "the ref must exist before the launcher resolves it")

    def test_created_pull_ref_resolves_through_the_launcher_git_identity(self):
        """Prove the ref form with a real repository: absent before, found after."""
        git = shutil.which("git")
        self.assertIsNotNone(git, "git is required to prove the ref form")
        launcher = load_launcher(LAUNCHER)
        with tempfile.TemporaryDirectory(prefix="docs-ci-ref-") as temporary:
            repo = Path(temporary).resolve()
            home = repo / "home"
            home.mkdir()
            env = {
                "PATH": "/usr/local/bin:/usr/bin:/bin",
                "HOME": str(home),
                "LC_ALL": "C",
                "GIT_CONFIG_GLOBAL": os.devnull,
                "GIT_CONFIG_NOSYSTEM": "1",
                "GIT_TERMINAL_PROMPT": "0",
            }
            subprocess.run([git, "init", "-q", "-b", "main", str(repo)],
                           check=True, env=env)
            subprocess.run([git, "-C", str(repo), "-c", "user.name=fixture",
                            "-c", "user.email=fixture@example.invalid",
                            "commit", "--allow-empty", "-q", "-m", "fixture"],
                           check=True, env=env)
            observed = subprocess.run([git, "-C", str(repo), "rev-parse", "HEAD"],
                                      check=True, env=env, stdout=subprocess.PIPE)
            head = observed.stdout.decode().strip()
            context = launcher.Context()
            environment = launcher.git_environment(home)

            # The documented ref form does not exist after a plain clone or
            # checkout of that pull ref; this is the reported blocker.
            with self.assertRaises(launcher.Refusal) as caught:
                launcher.git_revision(context, repo, environment, ref=PULL_REF,
                                      label="candidate")
            self.assertEqual(caught.exception.reason, "candidate-ref-unavailable")

            # The workflow step that creates it makes the launcher's own
            # resolution succeed, bound to the checked-out head.
            subprocess.run([git, "-C", str(repo), "update-ref", PULL_REF, "HEAD"],
                           check=True, env=env)
            self.assertEqual(
                launcher.git_revision(context, repo, environment, expected=head,
                                      ref=PULL_REF, label="candidate"),
                head)

    def test_evidence_uploads_exclude_the_scratch_tree(self):
        """The whole OUT_DIR holds the staged tree, node_modules and the build."""
        cases = {
            "main-build": ("Upload run identity evidence", "main"),
            "pr-validate": ("Upload identity and check evidence", "pr"),
        }
        for job, (step_name, lane) in cases.items():
            with self.subTest(job=job):
                block = workflow_step_block(self.jobs[job], step_name)
                self.assertIsNotNone(block, "job %s has no evidence upload" % job)
                expected = [
                    "${{ runner.temp }}/docs-ci/%s/identity.json" % lane,
                    "${{ runner.temp }}/docs-ci/%s/records.json" % lane,
                    "${{ runner.temp }}/docs-ci/%s/status.json" % lane,
                    "${{ runner.temp }}/docs-ci/%s/private/logs/*.log" % lane,
                ]
                self.assertEqual(workflow_path_list(block), expected)
                for value in workflow_path_list(block):
                    self.assertNotEqual(value, "${{ runner.temp }}/docs-ci/" + lane)
                    self.assertNotIn("/private/staged", value)
                    self.assertNotIn("/private/dependencies", value)
                    self.assertNotIn("/private/writes", value)
                    self.assertRegex(value, r"\.(json|log)$")


if __name__ == "__main__":
    unittest.main()
