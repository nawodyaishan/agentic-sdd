#!/usr/bin/env python3
"""Trusted docs-only CI launcher, identity emitter and base-owned evaluator.

Every PR-authored byte this module reads is data. A candidate checkout supplies
content; it never supplies a verdict. The authoritative result is derived from
base-owned policy, actual container exits, an independently re-read source
inventory and independently scanned build output.

Frozen development interface (plan C5; workflow lane B5-W1-B authors against
this, not against a competing launcher)::

    scripts/docs-ci.py --trusted DIR --candidate DIR --output DIR \
        --mode pr|main|release|rollback [--action run|identity|provision|evaluate] \
        [--artifact DIR [--record FILE]] [--expected-revision SHA] \
        [--artifact-run RUN_ID] [--run-id RUN_ID]
    scripts/docs-ci.py --mode links --trusted DIR --output DIR [--network]

Actions
  run        identity -> policy/dependency gates -> provision -> candidate
             checks in a bounded networkless container -> evaluator -> status.
             Default action. Refuses to reuse a non-empty output directory.
             `--mode main --artifact DIR` publishes the validated built site
             into DIR, and only the site: the trust record stays in --output.
             `--artifact` is refused in `pr` mode, so no candidate build can be
             published or promoted.
  identity   resolve and write identity.json only; executes nothing.
  provision  identity + gates + reviewed-lock provisioning only; no candidate
             code and no status verdict.
  evaluate   base-owned evaluator over launcher records plus read-only inputs.
             Runs after the candidate has exited; may be invoked on its own in
             a later workflow step against the same --output.

Modes
  pr         untrusted candidate: explicit base SHA, candidate SHA and head ref
             are all required and re-resolved after the candidate exits. The
             workflow must fetch --candidate-ref (for example
             refs/pull/<number>/head) into the candidate checkout; a ref that
             is missing, moved or resolves elsewhere is blocked.
  main       trusted main tree (candidate defaults to --trusted); builds and
             validates the same way, for the owned upload step.
  release    read-only observation of a retained main artifact. Never blocks
             product release: it always exits 0 and records blocking_release
             false.
  rollback   verification of a retained artifact before an authorized rollback
             run. Publishes nothing; an unavailable/expired artifact is blocked.
             The retained trust record must be bound to an explicit
             `--expected-revision` and/or `--artifact-run`; the current HEAD is
             never the reference, because a rollback restores an older revision
             by design. `--record FILE` names the trust record when the retained
             site is stored flat instead of `<DIR>/site` + `<DIR>/status.json`.
  links      read-only external-link and declared-dependency inventory for the
             optional scheduled lane. Observation only: it is never
             authoritative, never blocks a release and never fetches a URL
             unless `--network` is passed; reachability results stay
             observations. A completed inventory exits 0; only a usage or IO
             error exits 1. See the observation section near the end of this file.

Test seams (used by focused fixtures; no test-only bypass of a gate)
  Context(runner=...)       replaces every external command invocation.
  Context(uid=, gid=, posix=) replaces host identity probes.
  run_cli(argv, context=)   replaces sys.argv and the process-level runner.
  main(argv, context=)      same, returning an exit code.

Dependency provisioning is the only phase allowed network access, and it never
sees candidate files: it copies the reviewed trusted lock into launcher scratch
and runs `npm ci --ignore-scripts` there. Candidate checks run with
--network=none, a read-only root, dropped capabilities, no host home, no Docker
socket and no shared cache. Their container user is `--user uid:gid`, so the
`$HOME` tmpfs is mounted writable for that non-root user (mode=1777), not
root-owned.

Local-only trees are never reviewed input: `.codegraph/` (machine-local index)
and `backups/` (private local backups) are skipped by the source inventory and
recorded as an observation, while a real credential file or another sensitive
directory still refuses the run.

Adopted validation (Q1): the trusted base owns the whole transitive validation
surface - launcher, site scripts, plugins, tests, site config, package files,
this suite and `Taskfile.yml` - and an ordered check list with exact argv. The
base evaluator requires every adopted check id, in order, with the recorded
argv and exit 0. The Node suite is part of the lane, so a build that only looks
green is not enough.

Publication allowlist (Q2): `website/docs-manifest.json` in the trusted base is
the allowlist. A candidate page not adopted by the base needs trusted policy
review, the content tree must equal the adopted L/C source set before the build,
and every built HTML route must map back to an adopted page id. A dirty trusted
main tree is recorded and makes the run non-authoritative; a non-authoritative
run never publishes.
"""
import argparse
import bisect
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys

from subprocess import TimeoutExpired as _TimeoutExpired

SCHEMA_IDENTITY = "docs-ci-identity/1"
SCHEMA_RECORDS = "docs-ci-records/1"
SCHEMA_STATUS = "docs-ci-status/1"

RESULT_PASS = "pass"
RESULT_FAIL = "fail"
RESULT_BLOCKED = "blocked"
RESULT_DEPENDENCY = "needs-trusted-dependency-review"
RESULT_POLICY = "needs-trusted-policy-review"
RESULT_OBSERVED = "observed"

EXIT_CODES = {RESULT_PASS: 0, RESULT_OBSERVED: 0, RESULT_FAIL: 1}

FULL_SHA = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
IMAGE_PIN = re.compile(r"^[a-z0-9][a-z0-9._/-]*(?::[A-Za-z0-9._-]+)?@sha256:[0-9a-f]{64}$")
SAFE_RELATIVE = re.compile(r"^[A-Za-z0-9._/-]+$")
# npm script names may carry a scope separator, e.g. `docs:check`.
SAFE_SCRIPT_NAME = re.compile(r"^[A-Za-z0-9._:@/-]+$")

MAX_SOURCE_FILES = 20000
MAX_SOURCE_BYTES = 64 << 20
MAX_ARTIFACT_FILES = 20000
MAX_ARTIFACT_BYTES = 256 << 20
MAX_DEPENDENCY_FILES = 200000
MAX_JSON_BYTES = 4 << 20

EXCLUDED_ANY = frozenset({
    ".git", "node_modules", "__pycache__", ".astro", ".cache", "dist", ".venv", "coverage",
})
SENSITIVE_DIRS = frozenset({
    ".local", ".config", ".opencode", ".claude", ".codex", ".agents", ".ssh", ".aws",
    ".azure", ".kube", ".gnupg", ".secrets", "backups",
})
SENSITIVE_FILES = frozenset({
    "auth.json", "credentials.json", "tokens.json", ".netrc", ".git-credentials", ".npmrc",
    ".pypirc", ".envrc",
})
# Documented, gitignored local trees. They exist only in a developer checkout, are
# never reviewed candidate content, and are skipped with a recorded observation
# instead of aborting a local run. Real credential files are still refused.
LOCAL_ONLY_DIRS = frozenset({".codegraph", "backups"})
# Repository-root local build output (`.gitignore`: `bin/`). It is never build
# input or publication content, and a developer checkout may hold hundreds of
# megabytes of binaries there. Only the root directory is skipped: a nested
# `bin/` stays inventoried.
LOCAL_ONLY_ROOT_DIRS = frozenset({"bin"})
FORBIDDEN_ARTIFACT_PARTS = (
    ".git/", ".agentic-sdd/", "docs/handovers/", "docs/testing/", "specs/", "backups/",
)

ALLOWED_CONTAINER_ENV = frozenset({
    "PATH", "HOME", "TMPDIR", "LANG", "LC_ALL", "TZ", "CI", "NODE_ENV",
    "ASTRO_TELEMETRY_DISABLED", "ASTRO_OUT_DIR", "ASTRO_CACHE_DIR", "VITE_CACHE_DIR",
})
FORBIDDEN_CONTAINER_ENV = frozenset({
    "NODE_OPTIONS", "NODE_PATH", "DOCKER_HOST", "DOCKER_CONFIG", "GITHUB_TOKEN",
    "GH_TOKEN", "HOMEBREW_TAP_TOKEN",
})
FORBIDDEN_CONTAINER_PREFIXES = ("ACTIONS_", "GITHUB_", "NPM_CONFIG_", "npm_config_")

# Embedded default policy. It intentionally carries no image pin: a trusted base
# (or its workflow) must adopt an exact digest through --image or a policy file
# before anything executes. Everything else is base-owned and immutable for a
# run; the candidate never contributes policy fields.
DEFAULT_POLICY = {
    "schema": "docs-ci-policy/1",
    "site_root": "website",
    "site_manifest": "website/docs-manifest.json",
    "dependency_files": [
        "website/package.json",
        "website/package-lock.json",
        "website/.node-version",
    ],
    # The protected surface is the whole transitive validation path, not just
    # this launcher: a candidate edit to any validator, plugin, test, site
    # config or task wrapper could make a green check meaningless. Each entry is
    # compared byte-for-byte with the trusted base before anything executes.
    # `website/package.json` and `website/package-lock.json` are also dependency
    # files; the dependency gate runs first, so they keep their separate
    # `needs-trusted-dependency-review` result.
    "protected_paths": [
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
    ],
    "image": None,
    "dependencies_dir": "website/node_modules",
    "artifact_root": "website/dist",
    # Built output plus caches. The repo-root `.cache` is the scratch base the
    # site's disposable-build tests use; it is still launcher-owned scratch, not
    # a host cache or the staged source tree.
    "declared_write_paths": ["website/dist", "website/.cache", "website/.astro", ".cache"],
    # Ordered adopted check list. The base evaluator requires every id here, in
    # this order, with this exact argv and exit 0; a missing, renamed, reordered
    # or silently skipped check is a failure, never a pass. `requires_script`
    # names the trusted npm script, `requires_file` a trusted file inside the
    # site root, so a deleted validator cannot quietly turn into a no-op.
    #
    # Export has no npm script in the trusted package (the real CLI is
    # `node scripts/export.mjs`). `--check` refuses a missing output directory,
    # so the write and the drift check are two separate adopted checks over the
    # same writable artifact mount; together they prove the export is
    # deterministic and matches the manifest allowlist.
    "checks": [
        {"id": "build", "argv": ["npm", "run", "build"], "requires_script": "build"},
        {"id": "docs-check", "argv": ["npm", "run", "docs:check"],
         "requires_script": "docs:check"},
        {"id": "docs-check-built",
         "argv": ["npm", "run", "docs:check-built", "--", "--built", "dist", "--source"],
         "requires_script": "docs:check-built"},
        {"id": "docs-export-write", "argv": ["node", "scripts/export.mjs", "--out", "dist/export"],
         "requires_file": "scripts/export.mjs"},
        {"id": "docs-export-check",
         "argv": ["node", "scripts/export.mjs", "--out", "dist/export", "--check"],
         "requires_file": "scripts/export.mjs"},
        {"id": "node-test", "argv": ["npm", "test"], "requires_script": "test"},
    ],
    "build_env": {"ASTRO_TELEMETRY_DISABLED": "1"},
    "limits": {"cpus": "2", "memory": "2g", "pids": 256, "timeout_seconds": 900},
}

POLICY_KEYS = frozenset(DEFAULT_POLICY)
CHECK_KEYS = frozenset({"id", "argv", "requires_script", "requires_file"})
CONTENT_ROOT_SUFFIX = "src/content/docs"
# Only Markdown is a canonical page source. A `.mdx`/`.markdown`/`.mdoc` file
# in the content tree is an unadopted build input, never an extra page.
CONTENT_SUFFIXES = frozenset({".md"})
# Framework-generated, non-content HTML: Astro/Starlight emits a 404 route that
# no manifest page owns. This set is base-owned by this protected launcher
# file; widening it requires a trusted-base change, not a candidate edit.
FRAMEWORK_PAGE_ROUTES = frozenset({"404"})
LIMIT_KEYS = frozenset({"cpus", "memory", "pids", "timeout_seconds"})
CANDIDATE_PROGRAMS = frozenset({"npm", "node"})


class Refusal(Exception):
    """A bounded refusal that maps to a non-pass result, never to a fallback."""

    def __init__(self, result, reason, detail=""):
        if result not in (RESULT_BLOCKED, RESULT_FAIL, RESULT_DEPENDENCY, RESULT_POLICY):
            raise ValueError("unsupported refusal result")
        super().__init__(reason)
        self.result = result
        self.reason = reason
        self.detail = detail


class RunnerTimeout(Exception):
    """An injected or real command exceeded its bounded deadline."""


class Completed:
    """Minimal completed-process record; nonzero exits are data, not errors."""

    __slots__ = ("returncode", "stdout", "stderr")

    def __init__(self, returncode, stdout=b"", stderr=b""):
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def subprocess_runner(argv, *, cwd=None, env=None, timeout=None):
    try:
        completed = subprocess.run(
            [os.fspath(item) for item in argv], cwd=cwd, env=env,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout,
        )
    except _TimeoutExpired as error:
        raise RunnerTimeout("bounded command timed out") from error
    return Completed(completed.returncode, completed.stdout or b"", completed.stderr or b"")


class Context:
    """Injection seam for every external command and host identity probe."""

    def __init__(self, runner=None, *, uid=None, gid=None, posix=None):
        self.runner = runner or subprocess_runner
        self.uid = os.getuid() if uid is None else uid
        self.gid = os.getgid() if gid is None else gid
        self.posix = (os.name == "posix") if posix is None else posix

    def run(self, argv, *, cwd=None, env=None, timeout=None):
        return self.runner(list(argv), cwd=cwd, env=env, timeout=timeout)


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path, limit=MAX_JSON_BYTES):
    path = Path(path)
    data = path.read_bytes()
    if len(data) > limit:
        raise Refusal(RESULT_BLOCKED, "file-exceeds-bound", path.name)
    return sha256_bytes(data)


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def read_json(path, limit=MAX_JSON_BYTES):
    path = Path(path)

    def unique(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("duplicate JSON key")
            result[key] = value
        return result

    data = path.read_bytes()
    if len(data) > limit:
        raise ValueError("JSON exceeds bound")
    return json.loads(data.decode("utf-8"), object_pairs_hook=unique)


def write_json(path, data):
    path = Path(path)
    if path.exists() or path.is_symlink():
        raise Refusal(RESULT_BLOCKED, "refusing-to-overwrite-artifact", path.name)
    with path.open("x", encoding="utf-8") as stream:
        json.dump(data, stream, indent=2, sort_keys=True)
        stream.write("\n")


def write_status(path, data):
    """Status may be replaced by a re-run evaluator; content stays deterministic."""
    path = Path(path)
    temporary = path.with_name("." + path.name + ".partial")
    if temporary.exists() or temporary.is_symlink():
        raise Refusal(RESULT_BLOCKED, "refusing-to-overwrite-artifact", temporary.name)
    with temporary.open("x", encoding="utf-8") as stream:
        json.dump(data, stream, indent=2, sort_keys=True)
        stream.write("\n")
    os.replace(temporary, path)


def write_bytes(path, data):
    path = Path(path)
    if path.exists() or path.is_symlink():
        raise Refusal(RESULT_BLOCKED, "refusing-to-overwrite-artifact", path.name)
    path.write_bytes(data)


def safe_relative(value):
    if not isinstance(value, str) or not value or len(value) > 240:
        return False
    if not SAFE_RELATIVE.match(value) or value.startswith(("/", "-")):
        return False
    return ".." not in Path(value).parts


def safe_script_name(value):
    if not isinstance(value, str) or not value or len(value) > 120:
        return False
    if not SAFE_SCRIPT_NAME.match(value) or value.startswith("-"):
        return False
    return ".." not in Path(value).parts


# ---------------------------------------------------------------------------
# Policy: base-owned, digest-bound, strict


def resolve_policy(trusted, policy_path=None, image=None):
    """Return (policy, source label). A policy file must live inside --trusted."""
    trusted = Path(trusted).resolve()
    policy = copy.deepcopy(DEFAULT_POLICY)
    source = "embedded"
    candidate_path = None
    if policy_path:
        candidate_path = Path(policy_path)
        source = "argument"
    elif (trusted / "scripts/docs-ci-policy.json").exists():
        candidate_path = trusted / "scripts/docs-ci-policy.json"
        source = "scripts/docs-ci-policy.json"
    if candidate_path is not None:
        path = candidate_path
        if path.is_symlink():
            raise Refusal(RESULT_BLOCKED, "policy-symlink-refused")
        resolved = path.resolve()
        if resolved != trusted and trusted not in resolved.parents:
            raise Refusal(RESULT_BLOCKED, "policy-outside-trusted-base")
        if not resolved.is_file():
            raise Refusal(RESULT_BLOCKED, "policy-unreadable")
        try:
            overlay = read_json(resolved)
        except (OSError, ValueError):
            raise Refusal(RESULT_BLOCKED, "policy-invalid")
        if not isinstance(overlay, dict):
            raise Refusal(RESULT_BLOCKED, "policy-invalid")
        unknown = set(overlay) - POLICY_KEYS
        if unknown:
            raise Refusal(RESULT_BLOCKED, "policy-unknown-key", sorted(unknown)[0])
        policy.update(overlay)
    if image:
        policy["image"] = image
        if source == "embedded":
            source = "argument-image"
    validate_policy(policy)
    return policy, source


def validate_policy(policy):
    if policy.get("schema") != DEFAULT_POLICY["schema"]:
        raise Refusal(RESULT_BLOCKED, "policy-schema-unsupported")
    image = policy.get("image")
    if image is not None and (not isinstance(image, str) or not IMAGE_PIN.match(image)):
        raise Refusal(RESULT_BLOCKED, "policy-image-not-digest-pinned")
    for key in ("site_root", "site_manifest", "dependencies_dir", "artifact_root"):
        if not safe_relative(policy.get(key)):
            raise Refusal(RESULT_BLOCKED, "policy-path-invalid", key)
    if not policy["site_manifest"].startswith(policy["site_root"] + "/"):
        raise Refusal(RESULT_BLOCKED, "policy-manifest-outside-site-root")
    if not policy["artifact_root"].startswith(policy["site_root"] + "/"):
        raise Refusal(RESULT_BLOCKED, "policy-artifact-outside-site-root")
    if not policy["dependencies_dir"].startswith(policy["site_root"] + "/"):
        raise Refusal(RESULT_BLOCKED, "policy-dependencies-outside-site-root")
    for key in ("dependency_files", "protected_paths", "declared_write_paths"):
        values = policy.get(key)
        if not isinstance(values, list) or not values:
            raise Refusal(RESULT_BLOCKED, "policy-list-invalid", key)
        if len(values) != len(set(values)):
            raise Refusal(RESULT_BLOCKED, "policy-list-duplicate", key)
        for value in values:
            if not safe_relative(value):
                raise Refusal(RESULT_BLOCKED, "policy-path-invalid", key + ":" + str(value))
    if policy["artifact_root"] not in policy["declared_write_paths"]:
        raise Refusal(RESULT_BLOCKED, "policy-artifact-not-declared-writable")
    for relative in policy["declared_write_paths"]:
        if relative.startswith(policy["site_root"] + "/"):
            continue
        # A single-segment root scratch directory (for example the repo-root
        # `.cache` a site test writes) is allowed. It is still a writable mount
        # into launcher-owned scratch, asserted by assert_writable_scratch; it
        # never overlays the read-only staged source on the host.
        if len(Path(relative).parts) != 1:
            raise Refusal(RESULT_BLOCKED, "policy-write-outside-site-root", relative)
    if not isinstance(policy["build_env"], dict):
        raise Refusal(RESULT_BLOCKED, "policy-env-invalid")
    for key, value in policy["build_env"].items():
        if key not in ALLOWED_CONTAINER_ENV or key in FORBIDDEN_CONTAINER_ENV:
            raise Refusal(RESULT_BLOCKED, "policy-env-key-refused", key)
        if key.startswith(FORBIDDEN_CONTAINER_PREFIXES):
            raise Refusal(RESULT_BLOCKED, "policy-env-key-refused", key)
        if not isinstance(value, str) or len(value) > 512 or "\x00" in value:
            raise Refusal(RESULT_BLOCKED, "policy-env-value-refused", key)
    limits = policy.get("limits")
    if not isinstance(limits, dict) or set(limits) != LIMIT_KEYS:
        raise Refusal(RESULT_BLOCKED, "policy-limits-invalid")
    if not isinstance(limits["pids"], int) or not 16 <= limits["pids"] <= 4096:
        raise Refusal(RESULT_BLOCKED, "policy-limits-invalid", "pids")
    if not isinstance(limits["timeout_seconds"], int) or not 30 <= limits["timeout_seconds"] <= 3600:
        raise Refusal(RESULT_BLOCKED, "policy-limits-invalid", "timeout_seconds")
    if not re.fullmatch(r"[0-9]+(?:\.[0-9]+)?", str(limits["cpus"])):
        raise Refusal(RESULT_BLOCKED, "policy-limits-invalid", "cpus")
    if not re.fullmatch(r"[0-9]+(?:g|m)", str(limits["memory"])):
        raise Refusal(RESULT_BLOCKED, "policy-limits-invalid", "memory")
    checks = policy.get("checks")
    if not isinstance(checks, list) or not checks:
        raise Refusal(RESULT_BLOCKED, "policy-checks-invalid")
    seen = set()
    for check in checks:
        if not isinstance(check, dict) or not set(check) <= CHECK_KEYS:
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid")
        if "id" not in check or "argv" not in check:
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid")
        if not safe_relative(check["id"]):
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid", "id")
        if check["id"] in seen:
            raise Refusal(RESULT_BLOCKED, "policy-check-duplicate", check["id"])
        seen.add(check["id"])
        argv = check["argv"]
        if not isinstance(argv, list) or not argv or any(not isinstance(item, str) for item in argv):
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid", check["id"])
        if argv[0] not in CANDIDATE_PROGRAMS:
            raise Refusal(RESULT_BLOCKED, "policy-check-program-refused", check["id"])
        if "requires_script" in check and not safe_script_name(check["requires_script"]):
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid", check["id"])
        if "requires_file" in check and not safe_relative(check["requires_file"]):
            raise Refusal(RESULT_BLOCKED, "policy-check-invalid", check["id"])


def policy_digest(policy):
    return sha256_bytes(canonical(policy).encode("utf-8"))


# ---------------------------------------------------------------------------
# Trees: bounded inventory, staging, sensitive-input refusal


def _skipped(relative, skip):
    return any(relative == item or item in relative.parents for item in skip)


def note_local_only(observations, relative):
    """Record a skipped local-only tree; the run continues with an observation."""
    if observations is None:
        return
    observations.append(
        "skipped local-only directory " + relative.as_posix()
        + " (machine-local state, never reviewed input; not inventoried)")


def scan_tree(root, *, skip=(), observations=None):
    """Bounded inventory of regular files, refusing symlinks and credentials."""
    root = Path(root)
    if not root.is_dir() or root.is_symlink():
        raise Refusal(RESULT_BLOCKED, "source-root-unavailable")
    if root.name in SENSITIVE_DIRS:
        raise Refusal(RESULT_BLOCKED, "sensitive-source-root")
    skip = tuple(Path(item) for item in skip)
    records = {}
    total = 0
    for directory, dirs, files in os.walk(root, followlinks=False):
        base = Path(directory)
        kept = []
        for name in dirs:
            path = base / name
            relative = path.relative_to(root)
            if name in EXCLUDED_ANY or _skipped(relative, skip):
                continue
            if name in LOCAL_ONLY_DIRS or (len(relative.parts) == 1
                                           and name in LOCAL_ONLY_ROOT_DIRS):
                note_local_only(observations, relative)
                continue
            if name in SENSITIVE_DIRS:
                raise Refusal(RESULT_BLOCKED, "sensitive-source-directory", relative.as_posix())
            if path.is_symlink():
                raise Refusal(RESULT_BLOCKED, "source-symlink-refused", relative.as_posix())
            kept.append(name)
        dirs[:] = kept
        for name in files:
            path = base / name
            relative = path.relative_to(root)
            if name in EXCLUDED_ANY or _skipped(relative, skip):
                continue
            if (name in SENSITIVE_FILES or name == ".env" or name.startswith(".env.")
                    or Path(name).suffix.lower() in (".db", ".sqlite", ".sqlite3")):
                raise Refusal(RESULT_BLOCKED, "credential-file-in-source", relative.as_posix())
            info = path.lstat()
            if not stat.S_ISREG(info.st_mode):
                raise Refusal(RESULT_BLOCKED, "source-not-regular", relative.as_posix())
            data = path.read_bytes()
            total += len(data)
            if total > MAX_SOURCE_BYTES or len(records) >= MAX_SOURCE_FILES:
                raise Refusal(RESULT_BLOCKED, "source-exceeds-bound")
            records[relative.as_posix()] = {
                "sha256": sha256_bytes(data), "size": len(data),
                "executable": bool(info.st_mode & 0o111),
            }
    return records


def inventory_digest(records):
    return sha256_bytes(canonical(records).encode("utf-8"))


def source_skip(policy):
    """Paths excluded from the source inventory: generated output and mounts."""
    return list(policy["declared_write_paths"]) + [policy["dependencies_dir"]]


def stage_source(source, destination, *, skip=(), observations=None):
    """Copy reviewed inputs into launcher-owned scratch; the candidate stays read-only."""
    source, destination = Path(source), Path(destination)
    records = scan_tree(source, skip=skip, observations=observations)
    destination.mkdir(mode=0o700)
    for relative, observed in sorted(records.items()):
        target = destination / relative
        target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        data = (source / relative).read_bytes()
        if sha256_bytes(data) != observed["sha256"]:
            raise Refusal(RESULT_BLOCKED, "source-changed-while-staging", relative)
        target.write_bytes(data)
        target.chmod(0o755 if observed["executable"] else 0o644)
    for relative in skip:
        (destination / relative).mkdir(mode=0o700, parents=True, exist_ok=True)
    return records


def subtree_digests(root, *, prefix=""):
    """Digest map for a protected subtree; refuses links and Git metadata."""
    root = Path(root)
    if root.is_symlink() or not root.is_dir():
        raise Refusal(RESULT_POLICY, "protected-path-unavailable")
    digests = {}
    for directory, dirs, files in os.walk(root, followlinks=False):
        base = Path(directory)
        if ".git" in dirs:
            raise Refusal(RESULT_POLICY, "protected-path-git-metadata")
        dirs[:] = [name for name in dirs if name != ".git"]
        for name in sorted(files):
            path = base / name
            relative = path.relative_to(root)
            if path.is_symlink() or not stat.S_ISREG(path.lstat().st_mode):
                raise Refusal(RESULT_POLICY, "protected-path-not-regular", relative.as_posix())
            key = (Path(prefix) / relative).as_posix()
            digests[key] = sha256_file(path, limit=MAX_SOURCE_BYTES)
    return digests


def protected_digests(root, *, policy):
    result = {}
    for relative in policy["protected_paths"]:
        path = Path(root) / relative
        if path.is_symlink():
            raise Refusal(RESULT_POLICY, "protected-path-symlink", relative)
        if path.is_dir():
            result[relative] = subtree_digests(path, prefix=relative)
        elif path.is_file():
            result[relative] = {relative: sha256_file(path, limit=MAX_SOURCE_BYTES)}
        else:
            raise Refusal(RESULT_POLICY, "protected-path-missing", relative)
    return result


def compare_protected(trusted, candidate, *, policy):
    base = protected_digests(trusted, policy=policy)
    proposed = protected_digests(candidate, policy=policy)
    for relative in policy["protected_paths"]:
        if base[relative] == proposed[relative]:
            continue
        # Report the exact changed file when the protected path is a subtree;
        # the declared path is only a fallback for a pure add/remove of it.
        differing = sorted(
            key for key in set(base[relative]) | set(proposed[relative])
            if base[relative].get(key) != proposed[relative].get(key)
        )
        raise Refusal(RESULT_POLICY, "candidate-changed-protected-path",
                      differing[0] if differing else relative)
    return base


def dependency_digests(root, *, policy):
    result = {}
    for relative in policy["dependency_files"]:
        path = Path(root) / relative
        if path.is_symlink() or not path.is_file():
            raise Refusal(RESULT_DEPENDENCY, "dependency-file-missing", relative)
        result[relative] = sha256_file(path, limit=MAX_JSON_BYTES)
    return result


def check_dependency_equality(trusted, candidate, *, policy):
    base = dependency_digests(trusted, policy=policy)
    proposed = dependency_digests(candidate, policy=policy)
    for relative, digest in base.items():
        if proposed[relative] != digest:
            raise Refusal(RESULT_DEPENDENCY, "dependency-lock-drift", relative)
    return base


def check_required_script(trusted, *, policy):
    """Every invoked npm script and file must exist in the reviewed trusted base.

    A check whose script or validator file is absent would otherwise exit 0 or
    do nothing, so an absent target is a policy refusal before execution.
    """
    package = Path(trusted) / policy["site_root"] / "package.json"
    try:
        data = read_json(package)
    except (OSError, ValueError):
        raise Refusal(RESULT_POLICY, "trusted-package-invalid")
    scripts = data.get("scripts") if isinstance(data, dict) else None
    if not isinstance(scripts, dict):
        raise Refusal(RESULT_POLICY, "trusted-package-scripts-missing")
    for check in policy["checks"]:
        required = check.get("requires_script")
        if required and required not in scripts:
            raise Refusal(RESULT_POLICY, "trusted-check-script-missing", required)
        required_file = check.get("requires_file")
        if required_file:
            target = Path(trusted) / policy["site_root"] / required_file
            if target.is_symlink() or not target.is_file():
                raise Refusal(RESULT_POLICY, "trusted-check-file-missing", required_file)


def required_page_policy(trusted_manifest, candidate_manifest):
    """The trusted manifest is the publication allowlist, not a minimum set.

    A candidate may reorder pages, but it may not add, remove, reclassify or
    move one: an unadopted id is metadata-only until a trusted-base commit
    adopts it. Unregistered pages must not publish, and a candidate-only
    manifest addition therefore needs trusted policy review.
    """
    if not isinstance(trusted_manifest, dict) or not isinstance(candidate_manifest, dict):
        raise Refusal(RESULT_POLICY, "manifest-invalid")
    for key in ("schemaVersion", "site", "base"):
        if trusted_manifest.get(key) != candidate_manifest.get(key):
            raise Refusal(RESULT_POLICY, "manifest-origin-drift", key)
    trusted_pages = trusted_manifest.get("pages")
    candidate_pages = candidate_manifest.get("pages")
    if not isinstance(trusted_pages, list) or not trusted_pages:
        raise Refusal(RESULT_POLICY, "trusted-manifest-pages-missing")
    if not isinstance(candidate_pages, list) or not candidate_pages:
        raise Refusal(RESULT_POLICY, "candidate-manifest-empty")
    candidate_by_id = {}
    sources = set()
    for page in candidate_pages:
        if not isinstance(page, dict) or not isinstance(page.get("id"), str):
            raise Refusal(RESULT_POLICY, "candidate-manifest-invalid")
        if page["id"] in candidate_by_id:
            raise Refusal(RESULT_POLICY, "candidate-manifest-duplicate-id", page["id"])
        if page.get("source") in sources:
            raise Refusal(RESULT_POLICY, "candidate-manifest-duplicate-source", page["id"])
        sources.add(page.get("source"))
        candidate_by_id[page["id"]] = page
    adopted = {}
    required = []
    for page in trusted_pages:
        if not isinstance(page, dict) or not isinstance(page.get("id"), str):
            raise Refusal(RESULT_POLICY, "trusted-manifest-invalid")
        if page["id"] in adopted:
            raise Refusal(RESULT_POLICY, "trusted-manifest-duplicate-id", page["id"])
        adopted[page["id"]] = page
        observed = candidate_by_id.get(page["id"])
        if observed is None:
            raise Refusal(RESULT_POLICY, "required-page-removed", page["id"])
        if observed.get("class") != page.get("class") or observed.get("source") != page.get("source"):
            raise Refusal(RESULT_POLICY, "required-page-reclassified", page["id"])
        if page.get("class") not in ("L", "C"):
            continue
        required.append({"id": page["id"], "source": page["source"], "class": page["class"]})
    for identifier in candidate_by_id:
        if identifier not in adopted:
            raise Refusal(RESULT_POLICY, "candidate-manifest-unadopted-page", identifier)
    if not required:
        raise Refusal(RESULT_POLICY, "trusted-manifest-required-set-empty")
    return required


def content_root(policy):
    return policy["site_root"] + "/" + CONTENT_ROOT_SUFFIX


def scan_content_tree(root, policy):
    """Inventory of the canonical content tree: regular Markdown files only.

    `root` may be the candidate checkout or the launcher's staged copy. A
    symlink or special entry is not a page here; the bounded source scan still
    refuses it before staging. Any other extension in the content tree is an
    unadopted build input and is refused.
    """
    prefix = content_root(policy)
    base = Path(root) / prefix
    if base.is_symlink() or not base.is_dir():
        raise Refusal(RESULT_POLICY, "content-root-unavailable", prefix)
    observed = set()
    for directory, dirs, names in os.walk(base, followlinks=False):
        current = Path(directory)
        dirs[:] = [name for name in dirs if not (current / name).is_symlink()]
        for name in names:
            path = current / name
            try:
                info = path.lstat()
            except OSError:
                raise Refusal(RESULT_POLICY, "content-unreadable",
                              path.relative_to(root).as_posix())
            if stat.S_ISLNK(info.st_mode) or not stat.S_ISREG(info.st_mode):
                # Shape refusals stay with the bounded source scan, which
                # refuses every symlink or special file before staging; here a
                # non-regular entry is simply not a page.
                continue
            relative = path.relative_to(root).as_posix()
            if path.suffix.lower() not in CONTENT_SUFFIXES:
                raise Refusal(RESULT_POLICY, "candidate-content-extension", relative)
            observed.add(relative)
    return observed


def check_content_inventory(adopted_pages, root, policy):
    """Candidate content files must equal the adopted L/C source set exactly.

    This runs before the build and again over the staged copy after it. It is
    what stops an unregistered, deferred or private page from entering the
    build: the allowlist is the manifest, not whatever is on disk.
    """
    observed = scan_content_tree(root, policy)
    adopted = {page["source"] for page in adopted_pages}
    for relative in sorted(observed - adopted):
        raise Refusal(RESULT_POLICY, "candidate-content-unregistered", relative)
    for relative in sorted(adopted - observed):
        raise Refusal(RESULT_POLICY, "adopted-content-missing", relative)
    return sorted(observed)


def adopted_checks(policy):
    return [{"id": check["id"], "argv": list(check["argv"])} for check in policy["checks"]]


def compare_check_records(policy, recorded):
    """Problems with executed checks: exact ids, order, argv and exit 0.

    The records are written by this launcher, but `--action evaluate` re-reads
    them from disk, so the base evaluator treats them as data. A check that is
    missing, renamed, reordered, given different arguments or silently skipped
    cannot pass by omission.
    """
    adopted = adopted_checks(policy)
    if not isinstance(recorded, list):
        return ["records-not-a-list"]
    problems = []
    observed_ids = []
    by_id = {}
    for entry in recorded:
        if not isinstance(entry, dict) or not isinstance(entry.get("id"), str):
            problems.append("invalid-record")
            continue
        observed_ids.append(entry["id"])
        if entry["id"] in by_id:
            problems.append("duplicate:" + entry["id"])
        by_id[entry["id"]] = entry
    for check in adopted:
        entry = by_id.get(check["id"])
        if entry is None:
            problems.append("missing:" + check["id"])
            continue
        if list(entry.get("argv") or []) != check["argv"]:
            problems.append("argv:" + check["id"])
        if not isinstance(entry.get("exit_code"), int) or entry["exit_code"] != 0:
            problems.append("exit:" + check["id"])
    adopted_ids = [check["id"] for check in adopted]
    for identifier in observed_ids:
        if identifier not in adopted_ids:
            problems.append("unadopted:" + identifier)
    if observed_ids != adopted_ids:
        problems.append("order")
    return problems


def load_manifest(root, policy):
    path = Path(root) / policy["site_manifest"]
    if path.is_symlink() or not path.is_file():
        raise Refusal(RESULT_POLICY, "manifest-unavailable")
    try:
        return read_json(path)
    except (OSError, ValueError):
        raise Refusal(RESULT_POLICY, "manifest-invalid")


# ---------------------------------------------------------------------------
# Git identity: explicit base/candidate SHA agreement, never a guessed tree


def git_environment(home):
    return {
        "PATH": "/usr/local/bin:/usr/bin:/bin",
        "HOME": str(home),
        "LC_ALL": "C",
        "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_OPTIONAL_LOCKS": "0",
        "GIT_TERMINAL_PROMPT": "0",
    }


GIT_BASE = ["git", "--no-pager", "-c", "core.fsmonitor=false", "-C"]


def git_revision(ctx, root, env, *, expected=None, ref=None, label="root"):
    root = Path(root)
    top = ctx.run(GIT_BASE + [str(root), "rev-parse", "--show-toplevel"], env=env, timeout=20)
    if top.returncode != 0:
        raise Refusal(RESULT_BLOCKED, "git-identity-unavailable", label)
    try:
        observed_top = Path(os.fsdecode(top.stdout).strip()).resolve()
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "git-identity-unavailable", label)
    if observed_top != root.resolve():
        raise Refusal(RESULT_BLOCKED, "git-root-mismatch", label)
    head = ctx.run(GIT_BASE + [str(root), "rev-parse", "--verify", "HEAD"], env=env, timeout=20)
    sha = os.fsdecode(head.stdout).strip() if head.returncode == 0 else ""
    if not FULL_SHA.match(sha):
        raise Refusal(RESULT_BLOCKED, "git-identity-unavailable", label)
    if expected and sha != expected:
        raise Refusal(RESULT_BLOCKED, label + "-sha-mismatch", sha)
    if ref:
        resolved = ctx.run(GIT_BASE + [str(root), "rev-parse", "--verify", ref + "^{commit}"],
                           env=env, timeout=20)
        observed = os.fsdecode(resolved.stdout).strip() if resolved.returncode == 0 else ""
        if not FULL_SHA.match(observed):
            raise Refusal(RESULT_BLOCKED, "candidate-ref-unavailable", ref)
        if observed != sha:
            raise Refusal(RESULT_BLOCKED, "candidate-ref-mismatch", ref)
    return sha


def tree_state(ctx, root, home):
    """Record whether the trusted tree is clean; never guess.

    A dirty trusted tree still produces observations, but its result cannot be
    authoritative publication evidence: the site would be built from bytes that
    are not the reviewed revision. An unreadable state is recorded as not clean
    with its own named reason rather than assumed clean.
    """
    state = {"clean": False, "recorded": False, "reason": "trusted-tree-state-unavailable",
             "entries": [], "count": 0}
    completed = ctx.run(
        GIT_BASE + [str(root), "status", "--porcelain=v1", "--untracked-files=normal"],
        env=git_environment(home), timeout=30,
    )
    if completed.returncode != 0:
        return state
    entries = [line for line in os.fsdecode(completed.stdout).splitlines() if line.strip()]
    state["recorded"] = True
    state["count"] = len(entries)
    state["entries"] = entries[:5]
    if entries:
        state["reason"] = "dirty-trusted-tree"
    else:
        state["clean"] = True
        state["reason"] = ""
    return state


# ---------------------------------------------------------------------------
# Bounded container: constructed environment, no host leakage


CONTAINER_ENV_BASE = {
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "HOME": "/tmp/home",
    "TMPDIR": "/tmp",
    "LANG": "C.UTF-8",
    "LC_ALL": "C.UTF-8",
    "TZ": "UTC",
}

CONTAINER_WORKSPACE = "/workspace"


def host_environment(home):
    return {
        "PATH": "/usr/local/bin:/usr/bin:/bin",
        "HOME": str(home),
        "TMPDIR": str(home),
        "LC_ALL": "C",
    }


def container_environment(policy):
    env = dict(CONTAINER_ENV_BASE)
    for key, value in policy["build_env"].items():
        env[key] = value
    for key, value in env.items():
        if key in FORBIDDEN_CONTAINER_ENV or key.startswith(FORBIDDEN_CONTAINER_PREFIXES):
            raise Refusal(RESULT_BLOCKED, "container-env-key-refused", key)
        if key not in ALLOWED_CONTAINER_ENV:
            raise Refusal(RESULT_BLOCKED, "container-env-key-refused", key)
        if not isinstance(value, str) or "\x00" in value or len(value) > 512:
            raise Refusal(RESULT_BLOCKED, "container-env-value-refused", key)
    return env


def image_reference(policy):
    image = policy.get("image")
    if not image:
        raise Refusal(RESULT_BLOCKED, "image-pin-not-adopted")
    if not IMAGE_PIN.match(image):
        raise Refusal(RESULT_BLOCKED, "policy-image-not-digest-pinned")
    return image


def check_workdir(policy):
    """Containers start in the policy's site root, where its package.json lives.

    A policy check such as `npm run build` resolves its script from the working
    directory. Starting in the workspace root would find no package.json there
    and every real run would fail; the workspace itself is only a mount point.
    """
    site_root = policy.get("site_root")
    if not safe_relative(site_root):
        raise Refusal(RESULT_BLOCKED, "policy-path-invalid", "site_root")
    return CONTAINER_WORKSPACE + "/" + site_root


def assert_check_workdir(staged, policy, check):
    """The staged tree must hold the package and file the check invokes."""
    required_file = check.get("requires_file")
    if required_file:
        target = Path(staged) / policy["site_root"] / required_file
        if target.is_symlink() or not target.is_file():
            raise Refusal(RESULT_BLOCKED, "check-workdir-file-missing", required_file)
    required = check.get("requires_script")
    if not required:
        return policy["site_root"]
    package = Path(staged) / policy["site_root"] / "package.json"
    if package.is_symlink() or not package.is_file():
        raise Refusal(RESULT_BLOCKED, "check-workdir-package-missing", check["id"])
    try:
        data = read_json(package)
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "check-workdir-package-invalid", check["id"])
    scripts = data.get("scripts") if isinstance(data, dict) else None
    if not isinstance(scripts, dict) or required not in scripts:
        raise Refusal(RESULT_BLOCKED, "check-workdir-script-missing", required)
    return policy["site_root"]


def mount_spec(source, destination, *, writable=False):
    source = Path(source)
    if not source.is_absolute() or "," in str(source) or "," in destination:
        raise Refusal(RESULT_BLOCKED, "unsafe-mount", destination)
    if not destination.startswith("/") or destination == "/":
        raise Refusal(RESULT_BLOCKED, "unsafe-mount", destination)
    if source.is_symlink() or not source.is_dir():
        raise Refusal(RESULT_BLOCKED, "mount-source-unavailable", destination)
    refuse_sensitive_mount(source, home=Path.home())
    spec = "type=bind,src=" + str(source) + ",dst=" + destination
    return spec if writable else spec + ",readonly"


def refuse_sensitive_mount(source, *, home):
    resolved = Path(source).resolve()
    home = Path(home).resolve()
    if resolved == home or home in resolved.parents:
        raise Refusal(RESULT_BLOCKED, "host-home-mount-refused")
    if resolved == Path("/"):
        raise Refusal(RESULT_BLOCKED, "host-root-mount-refused")
    text = str(resolved)
    if "docker.sock" in text or "/.ssh" in text or "/.aws" in text or "/.gnupg" in text:
        raise Refusal(RESULT_BLOCKED, "socket-or-credential-mount-refused")


def container_argv(ctx, policy, *, mounts, workdir, environment, network, cidfile, command):
    if network not in ("none", "bridge"):
        raise Refusal(RESULT_BLOCKED, "container-network-invalid")
    cidfile = Path(cidfile)
    if not cidfile.is_absolute() or "," in str(cidfile):
        raise Refusal(RESULT_BLOCKED, "unsafe-cidfile")
    limits = policy["limits"]
    argv = [
        "docker", "run", "--rm", "--pull=never", "--network=" + network, "--read-only",
        "--cap-drop=ALL", "--security-opt=no-new-privileges",
        "--pids-limit=" + str(limits["pids"]), "--memory=" + limits["memory"],
        "--cpus=" + limits["cpus"], "--user=" + str(ctx.uid) + ":" + str(ctx.gid),
        "--cidfile", str(cidfile),
        # `exec` is explicit: Docker's default tmpfs is `noexec`, and the Node
        # suite spawns fixture executables it writes under TMPDIR. The root
        # filesystem stays read-only and the network stays off, so this changes
        # where a helper may run, not what the check may reach.
        "--tmpfs", "/tmp:rw,nosuid,nodev,exec,mode=1777,size=512m",
        # The container runs as `--user uid:gid`, so a mode=700 $HOME would be
        # root-owned and every npm write (`$HOME/.npm`) would fail EACCES.
        "--tmpfs", "/tmp/home:rw,nosuid,nodev,exec,mode=1777,size=64m",
        "--workdir", workdir,
    ]
    for spec in mounts:
        argv += ["--mount", spec]
    for key in sorted(environment):
        argv += ["-e", key + "=" + environment[key]]
    return argv + [image_reference(policy), *command]


def candidate_mounts(policy, *, staged, dependencies, writes):
    mounts = [mount_spec(staged, "/workspace")]
    mounts.append(mount_spec(dependencies, "/workspace/" + policy["dependencies_dir"]))
    for relative in policy["declared_write_paths"]:
        source = Path(writes) / relative
        source.mkdir(mode=0o700, parents=True, exist_ok=True)
        mounts.append(mount_spec(source, "/workspace/" + relative, writable=True))
    assert_writable_scratch(mounts, writes)
    return mounts


def assert_writable_scratch(mounts, scratch):
    """Writable mounts must sit inside launcher-owned scratch; no shared cache."""
    scratch = Path(scratch).resolve()
    writable = []
    for spec in mounts:
        if ",readonly" in spec:
            continue
        source = Path(spec.split(",dst=")[0].split("src=", 1)[1]).resolve()
        if source != scratch and scratch not in source.parents:
            raise Refusal(RESULT_BLOCKED, "shared-writable-mount-refused", str(source))
        writable.append(source)
    if not writable:
        raise Refusal(RESULT_BLOCKED, "no-writable-scratch")
    return writable


def check_dependency_tree(root):
    """Read-only dependency tree: links stay inside it, nothing special escapes."""
    root = Path(root)
    if root.is_symlink() or not root.is_dir():
        raise Refusal(RESULT_BLOCKED, "dependencies-missing")
    resolved_root = root.resolve()
    files = 0
    total = 0
    inventory = {}
    for directory, dirs, names in os.walk(root, followlinks=False):
        base = Path(directory)
        for name in sorted(dirs):
            path = base / name
            if path.is_symlink():
                target = path.resolve()
                if target != resolved_root and resolved_root not in target.parents:
                    raise Refusal(RESULT_BLOCKED, "dependency-symlink-escape", name)
                dirs.remove(name)
        for name in sorted(names):
            path = base / name
            if path.is_symlink():
                target = path.resolve()
                if target != resolved_root and resolved_root not in target.parents:
                    raise Refusal(RESULT_BLOCKED, "dependency-symlink-escape", name)
                continue
            info = path.lstat()
            if not stat.S_ISREG(info.st_mode):
                raise Refusal(RESULT_BLOCKED, "dependency-not-regular", name)
            files += 1
            total += info.st_size
            if files > MAX_DEPENDENCY_FILES or total > (2 << 30):
                raise Refusal(RESULT_BLOCKED, "dependencies-exceed-bound")
            inventory[path.relative_to(root).as_posix()] = info.st_size
    if files == 0:
        raise Refusal(RESULT_BLOCKED, "dependencies-empty")
    return {
        "files": files,
        "bytes": total,
        "inventory_sha256": sha256_bytes(canonical(inventory).encode("utf-8")),
        "digest_kind": "path-size inventory, not a content attestation",
    }


def cleanup_containers(ctx, cidfiles, home):
    uncertain = False
    for path in cidfiles:
        path = Path(path)
        if not path.exists():
            continue
        try:
            identifier = path.read_text().strip()
        except OSError:
            uncertain = True
            continue
        if not re.fullmatch(r"[0-9a-f]{64}", identifier):
            uncertain = True
            continue
        try:
            stopped = ctx.run(["docker", "rm", "--force", identifier],
                              env=host_environment(home), timeout=20)
        except (OSError, RunnerTimeout):
            uncertain = True
            continue
        if stopped.returncode and b"No such container" not in stopped.stderr:
            uncertain = True
    return uncertain


def provision_dependencies(ctx, policy, *, trusted, private, cidfiles=None):
    """Reviewed-lock provisioning: trusted files only, lifecycle scripts disabled.

    `cidfiles` is the caller's ownership ledger for `finally` cleanup. The path
    is recorded *before* the container starts, so a command that times out or is
    interrupted still leaves an owned container the caller can remove.
    """
    provision = Path(private) / "provision"
    provision.mkdir(mode=0o700)
    for source_name, target_name in (("package.json", "package.json"),
                                     ("package-lock.json", "package-lock.json")):
        source = Path(trusted) / policy["site_root"] / source_name
        if source.is_symlink() or not source.is_file():
            raise Refusal(RESULT_DEPENDENCY, "trusted-lock-missing", source_name)
        target = provision / target_name
        target.write_bytes(source.read_bytes())
        target.chmod(0o600)
    cidfile = Path(private) / "provision.cid"
    if cidfiles is not None:
        cidfiles.append(cidfile)
    mounts = [mount_spec(provision, "/provision", writable=True)]
    assert_writable_scratch(mounts, provision)
    argv = container_argv(
        ctx, policy, mounts=mounts, workdir="/provision",
        environment=container_environment(policy), network="bridge",
        cidfile=cidfile,
        command=["npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"],
    )
    completed = ctx.run(argv, cwd=private, env=host_environment(private),
                        timeout=policy["limits"]["timeout_seconds"])
    log = Path(private) / "provision.log"
    write_bytes(log, completed.stdout + completed.stderr)
    if completed.returncode != 0:
        raise Refusal(RESULT_BLOCKED, "dependency-provisioning-failed")
    modules = provision / "node_modules"
    if modules.is_symlink() or not modules.is_dir():
        raise Refusal(RESULT_BLOCKED, "dependency-provisioning-incomplete")
    dependencies = Path(private) / "dependencies"
    shutil.move(str(modules), str(dependencies))
    summary = check_dependency_tree(dependencies)
    summary["log"] = "private/provision.log"
    summary["log_sha256"] = sha256_file(log, limit=MAX_ARTIFACT_BYTES)
    return summary


def execute_checks(ctx, policy, *, private, staged, dependencies, writes, cidfiles=None):
    """Every candidate command runs in its own bounded, networkless container."""
    logs = Path(private) / "logs"
    logs.mkdir(mode=0o700)
    results = []
    recorded = cidfiles if cidfiles is not None else []
    for check in policy["checks"]:
        assert_check_workdir(staged, policy, check)
        cidfile = Path(private) / ("check-" + check["id"] + ".cid")
        # Record ownership before the container starts: a timeout or a signal
        # must still leave a cidfile the caller's `finally` cleanup can remove.
        recorded.append(cidfile)
        argv = container_argv(
            ctx, policy,
            mounts=candidate_mounts(policy, staged=staged, dependencies=dependencies, writes=writes),
            workdir=check_workdir(policy), environment=container_environment(policy),
            network="none", cidfile=cidfile, command=check["argv"],
        )
        completed = ctx.run(argv, cwd=private, env=host_environment(private),
                            timeout=policy["limits"]["timeout_seconds"])
        log = logs / (check["id"] + ".log")
        write_bytes(log, completed.stdout + completed.stderr)
        results.append({
            "id": check["id"],
            "argv": list(check["argv"]),
            "exit_code": completed.returncode,
            "log": "private/logs/" + check["id"] + ".log",
            "log_sha256": sha256_file(log, limit=MAX_ARTIFACT_BYTES),
        })
    return results, recorded


# ---------------------------------------------------------------------------
# Output artifact: no escapes, no history, no executable surface


def scan_artifact(root):
    root = Path(root)
    if root.is_symlink() or not root.is_dir():
        raise Refusal(RESULT_FAIL, "artifact-missing")
    resolved_root = root.resolve()
    files = {}
    escapes = []
    total = 0
    for directory, dirs, names in os.walk(root, followlinks=False):
        base = Path(directory)
        relative_base = base.relative_to(root)
        for name in sorted(dirs):
            path = base / name
            relative = (relative_base / name).as_posix()
            if path.is_symlink():
                escapes.append({"kind": "symlink", "path": relative})
                dirs.remove(name)
                continue
            if name in (".git", "node_modules"):
                escapes.append({"kind": "forbidden-directory", "path": relative})
                dirs.remove(name)
        for name in sorted(names):
            path = base / name
            relative = (relative_base / name).as_posix()
            info = path.lstat()
            if stat.S_ISLNK(info.st_mode):
                escapes.append({"kind": "symlink", "path": relative})
                continue
            if not stat.S_ISREG(info.st_mode):
                escapes.append({"kind": "special-file", "path": relative})
                continue
            if info.st_nlink > 1:
                escapes.append({"kind": "hardlink", "path": relative})
            if ".." in Path(relative).parts:
                escapes.append({"kind": "traversal", "path": relative})
            if any(relative.startswith(prefix) or (relative + "/").startswith(prefix)
                   for prefix in FORBIDDEN_ARTIFACT_PARTS):
                escapes.append({"kind": "forbidden-source", "path": relative})
            resolved = path.resolve()
            if resolved != resolved_root and resolved_root not in resolved.parents:
                escapes.append({"kind": "outside-root", "path": relative})
                continue
            data = path.read_bytes()
            total += len(data)
            if total > MAX_ARTIFACT_BYTES or len(files) >= MAX_ARTIFACT_FILES:
                raise Refusal(RESULT_FAIL, "artifact-exceeds-bound")
            files[relative] = {"sha256": sha256_bytes(data), "size": len(data)}
    return {
        "files": len(files),
        "bytes": total,
        "sha256": inventory_digest(files),
        "escapes": escapes,
        "inventory": files,
    }


def source_route(source, prefix):
    """The public content route for one adopted source path.

    `index.md` is the section root (`""` for the site root, `foo` for
    `foo/index.md`), so the route and the built HTML route agree.
    """
    route = source[len(prefix):-3]
    if route == "index":
        return ""
    if route.endswith("/index"):
        return route[: -len("/index")]
    return route


def artifact_routes(adopted_pages, policy):
    """Map every adopted page to the built HTML routes it may own."""
    prefix = content_root(policy) + "/"
    routes = {}
    expected = []
    for page in adopted_pages:
        source = page.get("source", "")
        if not source.startswith(prefix) or not source.endswith(".md"):
            raise Refusal(RESULT_POLICY, "required-page-source-unexpected", page.get("id", ""))
        route = source_route(source, prefix)
        if route in routes:
            raise Refusal(RESULT_POLICY, "adopted-page-route-collision", route)
        routes[route] = page["id"]
        if route == "":
            expected.append({"id": page["id"], "paths": ["index.html"]})
        else:
            expected.append({"id": page["id"], "paths": [route + "/index.html", route + ".html"]})
    return routes, expected


def built_artifact_route(relative):
    """The content route a built HTML file represents, or None when not HTML."""
    if not relative.endswith(".html"):
        return None
    name = relative[: -len(".html")]
    if name == "index":
        return ""
    if name.endswith("/index"):
        return name[: -len("/index")]
    return name


def reconcile_artifact(inventory, adopted_pages, policy):
    """Every built HTML content route must be an adopted page; nothing else.

    An extra built route, a route for an unregistered page or a missing adopted
    route is a failure. Non-HTML output (assets, search index, sitemap, export
    twins) is not a page and is left to the separate artifact scan.
    """
    routes, expected = artifact_routes(adopted_pages, policy)
    problems = []
    for relative in sorted(inventory):
        route = built_artifact_route(relative)
        if route is None or route in FRAMEWORK_PAGE_ROUTES:
            continue
        if route not in routes:
            problems.append("unregistered-route:" + relative)
    for page in expected:
        if not any(candidate in inventory for candidate in page["paths"]):
            problems.append("missing-route:" + page["id"])
    return problems


def validate_artifact_destination(destination, *, trusted, candidate, output):
    """A publication directory the launcher may create, never an input tree."""
    destination = Path(destination)
    if not destination.is_absolute():
        raise Refusal(RESULT_BLOCKED, "absolute-artifact-required")
    if ".." in destination.parts:
        raise Refusal(RESULT_BLOCKED, "unsafe-artifact-path")
    if destination.is_symlink():
        raise Refusal(RESULT_BLOCKED, "artifact-symlink-refused")
    for parent in destination.parents:
        if parent.exists() and parent.is_symlink():
            raise Refusal(RESULT_BLOCKED, "artifact-symlink-refused")
    resolved = destination.resolve()
    home = Path.home().resolve()
    if resolved == home or home in resolved.parents:
        raise Refusal(RESULT_BLOCKED, "home-directory-refused")
    resolved_output = Path(output).resolve()
    if (resolved == resolved_output or resolved_output in resolved.parents
            or resolved in resolved_output.parents):
        raise Refusal(RESULT_BLOCKED, "artifact-overlaps-evidence")
    for root in [Path(trusted)] + ([Path(candidate)] if candidate is not None else []):
        root = root.resolve()
        if resolved == root or root in resolved.parents or resolved in root.parents:
            raise Refusal(RESULT_BLOCKED, "artifact-overlaps-input")
    return destination


def prepare_artifact_destination(destination, *, trusted, candidate, output):
    """Validate and create the publication directory, never an input tree."""
    destination = validate_artifact_destination(
        destination, trusted=trusted, candidate=candidate, output=output)
    if destination.exists():
        if not destination.is_dir():
            raise Refusal(RESULT_BLOCKED, "artifact-not-a-directory")
        if list(destination.iterdir()):
            raise Refusal(RESULT_BLOCKED, "artifact-not-empty")
        if destination.stat().st_mode & 0o077:
            raise Refusal(RESULT_BLOCKED, "artifact-permissions-too-open")
    else:
        destination.mkdir(mode=0o700, parents=True)
    return destination


def publish_artifact(source, destination, *, expected_digest, trusted, candidate, output):
    """Copy the already-validated site; the published tree must re-scan identically."""
    source = Path(source)
    scanned = scan_artifact(source)
    if scanned["escapes"]:
        raise Refusal(RESULT_FAIL, "artifact-escape", scanned["escapes"][0]["kind"])
    if not expected_digest or scanned["sha256"] != expected_digest:
        raise Refusal(RESULT_FAIL, "artifact-digest-drift")
    target = prepare_artifact_destination(
        destination, trusted=trusted, candidate=candidate, output=output)
    for relative in sorted(scanned["inventory"]):
        item = target / relative
        item.parent.mkdir(mode=0o755, parents=True, exist_ok=True)
        shutil.copyfile(source / relative, item)
        item.chmod(0o644)
    published = scan_artifact(target)
    if published["escapes"] or published["sha256"] != scanned["sha256"]:
        raise Refusal(RESULT_BLOCKED, "artifact-publication-mismatch")
    return {"path": str(target), "files": published["files"],
            "bytes": published["bytes"], "sha256": published["sha256"]}


# ---------------------------------------------------------------------------
# Identity: resolved base/candidate SHAs, policy, lock and image


def assert_launcher_origin(trusted):
    expected = (Path(trusted) / "scripts/docs-ci.py").resolve()
    actual = Path(__file__).resolve()
    if actual != expected:
        raise Refusal(RESULT_BLOCKED, "launcher-not-loaded-from-trusted-base")
    return sha256_file(actual, limit=MAX_SOURCE_BYTES)


def resolve_identity(ctx, *, trusted, candidate, mode, base_sha, candidate_sha, candidate_ref,
                     policy, policy_source, home):
    environment = git_environment(home)
    identity = {
        "schema": SCHEMA_IDENTITY,
        "mode": mode,
        "base_sha": git_revision(ctx, trusted, environment, expected=base_sha, label="base"),
        "candidate_sha": None,
        "candidate_ref": candidate_ref or "",
        "policy_source": policy_source,
        "policy_sha256": policy_digest(policy),
        "lock_sha256": dependency_digests(trusted, policy=policy),
        "image": policy.get("image") or "",
        "launcher_sha256": assert_launcher_origin(trusted),
        "required_pages": [],
        "trusted_tree": tree_state(ctx, trusted, home),
    }
    if mode in ("pr", "main"):
        target = Path(candidate) if candidate is not None else Path(trusted)
        identity["candidate_sha"] = git_revision(
            ctx, target, environment, expected=candidate_sha,
            ref=candidate_ref if mode == "pr" else None, label="candidate",
        )
        if mode == "main" and identity["candidate_sha"] != identity["base_sha"]:
            raise Refusal(RESULT_BLOCKED, "main-candidate-sha-mismatch")
        identity["required_pages"] = [page["id"] for page in trusted_required_pages(trusted, policy)]
    return identity


def preexecution_gates(trusted, candidate, *, policy):
    """Everything here runs before any candidate byte can execute."""
    check_required_script(trusted, policy=policy)
    locks = check_dependency_equality(trusted, candidate, policy=policy)
    protected = compare_protected(trusted, candidate, policy=policy)
    required = required_page_policy(
        load_manifest(trusted, policy), load_manifest(candidate, policy),
    )
    content = check_content_inventory(required, candidate, policy)
    return {
        "lock_sha256": locks,
        "protected_paths": sorted(protected),
        "required_pages": [page["id"] for page in required],
        "required_page_records": required,
        "content_pages": content,
        "adopted_checks": [check["id"] for check in policy["checks"]],
    }


# ---------------------------------------------------------------------------
# Roots, scratch and evidence layout


def prepare_roots(*, trusted, candidate, output, mode, fresh):
    home = Path.home().resolve()
    trusted_path = Path(trusted)
    if trusted_path.is_symlink() or not trusted_path.is_dir():
        raise Refusal(RESULT_BLOCKED, "trusted-root-unavailable")
    trusted_path = trusted_path.resolve()
    if trusted_path == home:
        raise Refusal(RESULT_BLOCKED, "home-directory-refused")
    if not (trusted_path / "scripts").is_dir() and not (trusted_path / ".git").exists():
        raise Refusal(RESULT_BLOCKED, "trusted-root-not-a-checkout")
    candidate_path = None
    if mode in ("pr", "main") and candidate is not None:
        candidate_path = Path(candidate)
        if candidate_path.is_symlink() or not candidate_path.is_dir():
            raise Refusal(RESULT_BLOCKED, "candidate-root-unavailable")
        candidate_path = candidate_path.resolve()
        if candidate_path == home:
            raise Refusal(RESULT_BLOCKED, "home-directory-refused")
    if mode == "pr":
        if candidate_path is None:
            raise Refusal(RESULT_BLOCKED, "candidate-root-required")
        if candidate_path == trusted_path:
            raise Refusal(RESULT_BLOCKED, "separate-candidate-required")
    if mode == "main":
        candidate_path = candidate_path or trusted_path
    output_path = Path(output)
    if not output_path.is_absolute():
        raise Refusal(RESULT_BLOCKED, "absolute-output-required")
    if ".." in output_path.parts:
        raise Refusal(RESULT_BLOCKED, "unsafe-output-path")
    if output_path.is_symlink():
        raise Refusal(RESULT_BLOCKED, "output-symlink-refused")
    for parent in output_path.parents:
        if parent.exists() and parent.is_symlink():
            raise Refusal(RESULT_BLOCKED, "output-symlink-refused")
    resolved_output = output_path.resolve()
    for root in [trusted_path] + ([candidate_path] if candidate_path else []):
        if (resolved_output == root or root in resolved_output.parents
                or resolved_output in root.parents):
            raise Refusal(RESULT_BLOCKED, "output-overlaps-input")
    if resolved_output == home or home in resolved_output.parents:
        raise Refusal(RESULT_BLOCKED, "home-directory-refused")
    if fresh:
        if output_path.exists():
            if list(output_path.iterdir()):
                raise Refusal(RESULT_BLOCKED, "output-not-empty")
            if output_path.stat().st_mode & 0o077:
                raise Refusal(RESULT_BLOCKED, "output-permissions-too-open")
        else:
            output_path.mkdir(mode=0o700, parents=True)
    else:
        if not output_path.is_dir():
            raise Refusal(RESULT_BLOCKED, "output-unavailable")
    return trusted_path, candidate_path, output_path


def assert_bounded_host(ctx):
    if not ctx.posix:
        raise Refusal(RESULT_BLOCKED, "posix-host-required")
    if ctx.uid == 0:
        raise Refusal(RESULT_BLOCKED, "non-root-host-required")


def write_identity(output, identity):
    write_json(Path(output) / "identity.json", identity)


def read_identity(output):
    path = Path(output) / "identity.json"
    if path.is_symlink() or not path.is_file():
        raise Refusal(RESULT_BLOCKED, "identity-record-unavailable")
    try:
        return read_json(path)
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "identity-record-invalid")


def read_records(output):
    path = Path(output) / "records.json"
    if path.is_symlink() or not path.is_file():
        raise Refusal(RESULT_BLOCKED, "run-records-unavailable")
    try:
        return read_json(path)
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "run-records-invalid")


def prior_run_id(output):
    """Preserve a run id recorded by an earlier invocation of the same output."""
    path = Path(output) / "status.json"
    if path.is_symlink() or not path.is_file():
        return ""
    try:
        document = read_json(path)
    except (OSError, ValueError):
        return ""
    if not isinstance(document, dict):
        return ""
    run_block = document.get("run")
    return str(run_block.get("id") or "") if isinstance(run_block, dict) else ""


# ---------------------------------------------------------------------------
# Base-owned evaluator: runs after candidate exit over read-only inputs


def trusted_required_pages(trusted, policy):
    manifest = load_manifest(trusted, policy)
    pages = manifest.get("pages")
    if not isinstance(pages, list) or not pages:
        raise Refusal(RESULT_POLICY, "trusted-manifest-pages-missing")
    required = []
    for page in pages:
        if not isinstance(page, dict) or not isinstance(page.get("id"), str):
            raise Refusal(RESULT_POLICY, "trusted-manifest-invalid")
        if page.get("class") in ("L", "C"):
            required.append({"id": page["id"], "source": page.get("source", ""),
                             "class": page["class"]})
    return required


def read_candidate_report(output, artifact_root):
    """A candidate report is untrusted data; it never becomes a verdict."""
    for path in (Path(artifact_root) / "candidate-report.json",
                 Path(output) / "candidate-report.json"):
        if path.is_symlink() or not path.is_file():
            continue
        try:
            data = read_json(path, limit=256 << 10)
        except (OSError, ValueError):
            return {"present": True, "valid": False, "claimed_result": None,
                    "path": path.name, "authoritative": False}
        claimed = None
        if isinstance(data, dict):
            for key in ("result", "status", "conclusion"):
                if isinstance(data.get(key), str):
                    claimed = data[key]
                    break
        return {"present": True, "valid": isinstance(data, dict), "claimed_result": claimed,
                "path": path.name, "authoritative": False}
    return {"present": False, "valid": False, "claimed_result": None,
            "path": "", "authoritative": False}


def _detail(error):
    return error.reason + (":" + str(error.detail) if error.detail else "")


def evaluate_records(ctx, *, trusted, candidate, output, mode, policy, policy_source, identity,
                     records, home):
    """Independent base-owned validation; candidate state is never consulted."""
    output = Path(output)
    checks = []

    def note(identifier, status, detail=""):
        entry = {"id": identifier, "status": status}
        if detail:
            entry["detail"] = detail
        checks.append(entry)
        return status == "pass"

    staged = output / records.get("staged", "private/staged")
    artifact_root = output / records.get("artifact_dir", "private/writes/" + policy["artifact_root"])

    try:
        current = resolve_identity(
            ctx, trusted=trusted, candidate=candidate, mode=mode,
            base_sha=identity.get("base_sha"), candidate_sha=identity.get("candidate_sha"),
            candidate_ref=identity.get("candidate_ref") or None,
            policy=policy, policy_source=policy_source, home=home,
        )
        drift = [key for key in ("base_sha", "candidate_sha", "candidate_ref", "policy_sha256",
                                 "image", "launcher_sha256", "lock_sha256", "trusted_tree")
                 if current.get(key) != identity.get(key)]
        note("identity-consistency", "pass" if not drift else "fail", ",".join(drift))
    except Refusal as error:
        note("identity-consistency", "fail", _detail(error))

    try:
        compare_protected(trusted, staged, policy=policy)
        if policy_digest(policy) != identity.get("policy_sha256"):
            note("policy-integrity", "fail", "policy-digest-drift")
        else:
            note("policy-integrity", "pass")
    except Refusal as error:
        note("policy-integrity", "fail", _detail(error))

    try:
        check_dependency_equality(trusted, staged, policy=policy)
        note("dependency-integrity", "pass")
    except Refusal as error:
        note("dependency-integrity", "fail", _detail(error))

    candidate_checks = records.get("checks") or []
    executed = records.get("executed") is True
    if not executed or not candidate_checks:
        note("candidate-completion", "fail", "no-executed-check-record")
    else:
        problems = compare_check_records(policy, candidate_checks)
        note("candidate-completion", "pass" if not problems else "fail", ",".join(problems))

    adopted = []
    try:
        adopted = trusted_required_pages(trusted, policy)
    except Refusal as error:
        note("required-pages", "fail", _detail(error))
    else:
        try:
            required_page_policy(load_manifest(trusted, policy), load_manifest(staged, policy))
            note("required-pages", "pass")
        except Refusal as error:
            note("required-pages", "fail", _detail(error))
    try:
        check_content_inventory(adopted, staged, policy)
        note("content-inventory", "pass")
    except Refusal as error:
        note("content-inventory", "fail", _detail(error))

    source_summary = {"before_sha256": records.get("inventory_before", ""),
                      "after_sha256": records.get("inventory_after", ""),
                      "changed": records.get("changed", [])}
    try:
        observed = inventory_digest(scan_tree(staged, skip=source_skip(policy)))
        if records.get("inventory_before") != records.get("inventory_after"):
            note("source-inventory", "fail", "changed-during-execution")
        elif observed != records.get("inventory_after"):
            note("source-inventory", "fail", "changed-after-recording")
        else:
            note("source-inventory", "pass")
        source_summary["observed_sha256"] = observed
    except Refusal as error:
        note("source-inventory", "fail", _detail(error))

    artifact_summary = {"files": 0, "bytes": 0, "sha256": "", "escapes": []}
    inventory = {}
    try:
        scanned = scan_artifact(artifact_root)
        inventory = scanned["inventory"]
        artifact_summary = {"files": scanned["files"], "bytes": scanned["bytes"],
                            "sha256": scanned["sha256"], "escapes": scanned["escapes"]}
        note("artifact-integrity", "pass" if not scanned["escapes"] else "fail",
             ",".join(sorted({entry["kind"] for entry in scanned["escapes"]})))
    except Refusal as error:
        note("artifact-integrity", "fail", _detail(error))

    if adopted and inventory:
        try:
            problems = reconcile_artifact(inventory, adopted, policy)
        except Refusal as error:
            note("artifact-pages", "fail", _detail(error))
        else:
            note("artifact-pages", "pass" if not problems else "fail", ",".join(problems))
    else:
        note("artifact-pages", "fail", "required-pages-or-artifact-unavailable")

    report = read_candidate_report(output, artifact_root)
    note("candidate-report-is-data", "observed",
         "claimed=" + str(report.get("claimed_result")))

    failing = [check for check in checks if check["status"] == "fail"]
    if failing:
        result = RESULT_FAIL
        reason = "base-evaluator-failed"
    else:
        result = RESULT_PASS
        reason = "base-evaluator-pass"
    return result, reason, checks, artifact_summary, source_summary, report


def inspect_retained(*, artifact, expected_revision=None, artifact_run=None, record=None):
    """Verify a retained main artifact before an authorized release/rollback use.

    The retained record is bound to an explicit owner-supplied revision and/or
    run id. It is deliberately not compared with the current HEAD: a rollback
    restores an older trusted revision, so current HEAD is the wrong reference
    and would refuse every real rollback artifact.
    """
    artifact = Path(artifact)
    if artifact.is_symlink() or not artifact.is_dir():
        raise Refusal(RESULT_BLOCKED, "retained-artifact-unavailable")
    if record is not None:
        # Flat retained layout: the downloaded site is the artifact, and the
        # trust record is downloaded beside it from the same run.
        record_path = Path(record)
        site = artifact
    else:
        record_path = artifact / "status.json"
        site = artifact / "site"
    if record_path.is_symlink() or not record_path.is_file():
        raise Refusal(RESULT_BLOCKED, "retained-artifact-unavailable", record_path.name)
    try:
        prior = read_json(record_path)
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "retained-status-invalid")
    if not isinstance(prior, dict) or prior.get("schema") != SCHEMA_STATUS:
        raise Refusal(RESULT_BLOCKED, "retained-status-invalid")
    if prior.get("mode") != "main":
        raise Refusal(RESULT_BLOCKED, "retained-status-not-main")
    if prior.get("result") != RESULT_PASS or prior.get("authoritative") is not True:
        raise Refusal(RESULT_BLOCKED, "retained-status-not-passing")
    scanned = scan_artifact(site)
    if scanned["escapes"]:
        raise Refusal(RESULT_FAIL, "retained-artifact-escape", scanned["escapes"][0]["kind"])
    artifact_block = prior.get("artifact")
    recorded = artifact_block.get("sha256") if isinstance(artifact_block, dict) else None
    if not recorded or recorded != scanned["sha256"]:
        raise Refusal(RESULT_FAIL, "retained-artifact-digest-mismatch")
    identity_block = prior.get("identity")
    prior_identity = identity_block if isinstance(identity_block, dict) else {}
    run_block = prior.get("run")
    prior_run = str(run_block.get("id") or "") if isinstance(run_block, dict) else ""
    if expected_revision:
        if prior_identity.get("base_sha") != expected_revision:
            raise Refusal(RESULT_BLOCKED, "retained-artifact-from-other-revision")
    if artifact_run:
        if str(prior_run) != str(artifact_run):
            raise Refusal(RESULT_BLOCKED, "retained-artifact-run-mismatch")
    if not expected_revision and not artifact_run:
        raise Refusal(RESULT_BLOCKED, "retained-artifact-revision-unbound")
    return {"files": scanned["files"], "bytes": scanned["bytes"],
            "sha256": scanned["sha256"], "escapes": [],
            "revision": prior_identity.get("base_sha"),
            "run_id": str(prior_run)}


# ---------------------------------------------------------------------------
# Status documents and deterministic exit codes


def identity_view(identity):
    keys = ("base_sha", "candidate_sha", "candidate_ref", "policy_sha256", "policy_source",
            "lock_sha256", "image", "launcher_sha256", "required_pages", "trusted_tree")
    return {key: identity.get(key) for key in keys}


def status_document(*, mode, result, reason, identity, candidate_checks, evaluator_checks,
                    source_summary, artifact_summary, report, observations=None, detail="",
                    run_id=""):
    tree = (identity or {}).get("trusted_tree") or {}
    authoritative = result == RESULT_PASS and mode in ("pr", "main")
    authoritative_reason = ""
    if authoritative and mode == "main" and tree.get("clean") is not True:
        # A dirty trusted tree still produced observations, but the site was not
        # built from the reviewed revision, so it is not publication evidence.
        authoritative = False
        authoritative_reason = str(tree.get("reason") or "trusted-tree-state-unavailable")
    return {
        "schema": SCHEMA_STATUS,
        "mode": mode,
        "result": result,
        "reason": reason,
        "detail": detail,
        "authoritative": authoritative,
        "authoritative_reason": authoritative_reason,
        "preview": {"authoritative": False,
                    "reason": "local-preview-not-a-publication-verdict"},
        "blocking_release": False,
        "run": {"id": str(run_id or "")},
        "identity": identity_view(identity or {}),
        "candidate": {
            "executed": bool(candidate_checks),
            "checks": candidate_checks,
            "report": report,
        },
        "evaluator": {
            "phase": "base-owned-post-exit",
            "after_candidate_exit": True,
            "checks": evaluator_checks,
        },
        "source_inventory": source_summary,
        "artifact": artifact_summary,
        "observations": observations or [],
        "unknowns": [
            "live GitHub Actions/Pages/environment state is not observed by this launcher",
        ],
    }


def blocked_document(mode, result, reason, *, identity=None, detail="", run_id=""):
    return status_document(
        mode=mode, result=result, reason=reason, identity=identity or {},
        candidate_checks=[], evaluator_checks=[],
        source_summary={}, artifact_summary={"files": 0, "bytes": 0, "sha256": "", "escapes": []},
        report={"present": False, "valid": False, "claimed_result": None,
                "path": "", "authoritative": False},
        detail=detail, run_id=run_id,
    )


def exit_code(result):
    return EXIT_CODES.get(result, 2)


def emit(payload):
    print(json.dumps(payload, sort_keys=True))


# ---------------------------------------------------------------------------
# Actions


def start_output(args, *, fresh):
    trusted, candidate, output = prepare_roots(
        trusted=args.trusted, candidate=args.candidate, output=args.output,
        mode=args.mode, fresh=fresh,
    )
    private = output / "private"
    if not private.exists():
        private.mkdir(mode=0o700)
    elif private.is_symlink() or not private.is_dir():
        raise Refusal(RESULT_BLOCKED, "private-scratch-unavailable")
    return trusted, candidate, output, private


def resolve_launch_identity(ctx, args, *, policy, policy_source, trusted, candidate, private):
    return resolve_identity(
        ctx, trusted=trusted, candidate=candidate, mode=args.mode,
        base_sha=args.base_sha, candidate_sha=args.candidate_sha,
        candidate_ref=args.candidate_ref, policy=policy,
        policy_source=policy_source, home=private,
    )


def preflight_image(ctx, policy, private):
    completed = ctx.run(["docker", "image", "inspect", image_reference(policy)],
                        env=host_environment(private), timeout=30)
    if completed.returncode != 0:
        raise Refusal(RESULT_BLOCKED, "pinned-image-unavailable")


def action_identity(ctx, args):
    policy, policy_source = resolve_policy(args.trusted, args.policy, args.image)
    trusted, candidate, output, private = start_output(args, fresh=True)
    identity = resolve_launch_identity(ctx, args, policy=policy, policy_source=policy_source,
                                      trusted=trusted, candidate=candidate, private=private)
    write_identity(output, identity)
    emit({"action": "identity", "result": RESULT_PASS, "output": str(output),
          "identity": identity_view(identity)})
    return 0


def action_provision(ctx, args):
    if args.mode not in ("pr", "main"):
        raise Refusal(RESULT_BLOCKED, "provision-mode-unsupported")
    policy, policy_source = resolve_policy(args.trusted, args.policy, args.image)
    image_reference(policy)
    trusted, candidate, output, private = start_output(args, fresh=True)
    assert_bounded_host(ctx)
    identity = resolve_launch_identity(ctx, args, policy=policy, policy_source=policy_source,
                                       trusted=trusted, candidate=candidate, private=private)
    write_identity(output, identity)
    gates = preexecution_gates(trusted, candidate, policy=policy)
    preflight_image(ctx, policy, private)
    dependencies = provision_dependencies(ctx, policy, trusted=trusted, private=private)
    write_json(output / "dependencies.json", {
        "schema": "docs-ci-dependencies/1",
        "policy_sha256": identity["policy_sha256"],
        "image": identity["image"],
        "required_pages": gates["required_pages"],
        "dependencies": dependencies,
    })
    emit({"action": "provision", "result": RESULT_PASS, "output": str(output),
          "identity": identity_view(identity), "dependencies": dependencies})
    return 0


def evaluator_argv(trusted, output, args, candidate):
    launcher = Path(trusted) / "scripts/docs-ci.py"
    argv = [sys.executable, "-B", str(launcher), "--trusted", str(trusted),
            "--output", str(output), "--mode", args.mode, "--action", "evaluate"]
    if candidate is not None:
        argv += ["--candidate", str(candidate)]
    if args.policy:
        argv += ["--policy", args.policy]
    if args.image:
        argv += ["--image", args.image]
    if args.base_sha:
        argv += ["--base-sha", args.base_sha]
    if args.candidate_sha:
        argv += ["--candidate-sha", args.candidate_sha]
    if args.candidate_ref:
        argv += ["--candidate-ref", args.candidate_ref]
    if args.run_id:
        argv += ["--run-id", str(args.run_id)]
    return argv


def action_run(ctx, args):
    if args.mode not in ("pr", "main"):
        raise Refusal(RESULT_BLOCKED, "run-mode-unsupported")
    if args.artifact and args.mode != "main":
        raise Refusal(RESULT_BLOCKED, "artifact-only-for-trusted-main")
    policy, policy_source = resolve_policy(args.trusted, args.policy, args.image)
    image_reference(policy)
    trusted, candidate, output, private = start_output(args, fresh=True)
    assert_bounded_host(ctx)
    identity = resolve_launch_identity(ctx, args, policy=policy, policy_source=policy_source,
                                       trusted=trusted, candidate=candidate, private=private)
    write_identity(output, identity)
    gates = preexecution_gates(trusted, candidate, policy=policy)
    if args.artifact:
        # Fail a wrong publication path before any candidate byte can execute;
        # the directory itself is created only after the evaluator passes.
        validate_artifact_destination(args.artifact, trusted=trusted,
                                      candidate=candidate, output=output)
    preflight_image(ctx, policy, private)
    staged = private / "staged"
    observations = []
    before = stage_source(candidate, staged, skip=source_skip(policy),
                          observations=observations)
    writes = private / "writes"
    writes.mkdir(mode=0o700)
    cidfiles = []
    cleanup_uncertain = False
    try:
        dependencies = provision_dependencies(ctx, policy, trusted=trusted, private=private,
                                              cidfiles=cidfiles)
        checks, _ = execute_checks(
            ctx, policy, private=private, staged=staged,
            dependencies=private / "dependencies", writes=writes, cidfiles=cidfiles,
        )
    finally:
        # Every recorded container is removed even when a command timed out or
        # the run was interrupted; `cidfiles` is filled before each `run`.
        cleanup_uncertain = cleanup_containers(ctx, cidfiles, private)
    if cleanup_uncertain:
        raise Refusal(RESULT_BLOCKED, "owned-container-cleanup-uncertain")
    if args.mode == "main":
        tree = identity.get("trusted_tree") or {}
        if tree.get("clean") is True:
            observations.append("trusted main tree is clean")
        else:
            observations.append(
                "trusted main tree is not clean (" + str(tree.get("reason"))
                + "): the run stays non-authoritative")
    observations.append(
        "local previews are not authoritative: only a clean trusted-main run is publication evidence")
    after = scan_tree(staged, skip=source_skip(policy), observations=observations)
    changed = sorted(
        set(before) ^ set(after)
        | {name for name in set(before) & set(after) if before[name] != after[name]}
    )
    records = {
        "schema": SCHEMA_RECORDS,
        "mode": args.mode,
        "executed": True,
        "staged": "private/staged",
        "artifact_dir": "private/writes/" + policy["artifact_root"],
        "checks": checks,
        "dependencies": dependencies,
        "inventory_before": inventory_digest(before),
        "inventory_after": inventory_digest(after),
        "changed": changed,
        "required_pages": gates["required_pages"],
        "adopted_checks": gates["adopted_checks"],
        "content_pages": gates["content_pages"],
        "trusted_tree": identity.get("trusted_tree") or {},
        "policy_sha256": identity["policy_sha256"],
        "observations": observations,
    }
    write_json(output / "records.json", records)
    completed = ctx.run(
        evaluator_argv(trusted, output, args, candidate), cwd=private,
        env=host_environment(private), timeout=policy["limits"]["timeout_seconds"],
    )
    status_path = output / "status.json"
    if completed.returncode != 0 and not status_path.is_file():
        raise Refusal(RESULT_BLOCKED, "base-evaluator-not-completed")
    if status_path.is_symlink() or not status_path.is_file():
        raise Refusal(RESULT_BLOCKED, "base-evaluator-not-completed")
    try:
        status = read_json(status_path)
    except (OSError, ValueError):
        raise Refusal(RESULT_BLOCKED, "base-evaluator-status-invalid")
    if not isinstance(status, dict) or status.get("schema") != SCHEMA_STATUS:
        raise Refusal(RESULT_BLOCKED, "base-evaluator-status-invalid")
    bound = status.get("identity") or {}
    for key in ("base_sha", "candidate_sha", "policy_sha256", "image"):
        if bound.get(key) != identity.get(key):
            raise Refusal(RESULT_BLOCKED, "base-evaluator-identity-mismatch", key)
    published = None
    if args.artifact and status.get("result") == RESULT_PASS and status.get("authoritative") is True:
        # Publish only after the base-owned evaluator passed *and* the run is
        # authoritative: the trust record stays in --output, and the published
        # tree must re-scan to the same digest the status recorded. A failing
        # run, or a dirty trusted tree, publishes nothing.
        published = publish_artifact(
            output / records["artifact_dir"], args.artifact,
            expected_digest=(status.get("artifact") or {}).get("sha256"),
            trusted=trusted, candidate=candidate, output=output,
        )
    emit({"action": "run", "result": status.get("result"), "reason": status.get("reason"),
          "output": str(output), "artifact": (published or {}).get("path", ""),
          "published": published, "authoritative": status.get("authoritative"),
          "authoritative_reason": status.get("authoritative_reason"),
          "checks": [check.get("id") for check in checks],
          "identity": identity_view(identity)})
    return exit_code(status.get("result"))


def action_evaluate(ctx, args):
    if args.mode not in ("pr", "main"):
        raise Refusal(RESULT_BLOCKED, "evaluate-mode-unsupported")
    policy, policy_source = resolve_policy(args.trusted, args.policy, args.image)
    trusted, candidate, output = prepare_roots(
        trusted=args.trusted, candidate=args.candidate, output=args.output,
        mode=args.mode, fresh=False,
    )
    identity = read_identity(output)
    records = read_records(output)
    if records.get("mode") != args.mode:
        raise Refusal(RESULT_BLOCKED, "run-records-mode-mismatch")
    if records.get("policy_sha256") != identity.get("policy_sha256"):
        raise Refusal(RESULT_BLOCKED, "run-records-identity-mismatch")
    result, reason, checks, artifact_summary, source_summary, report = evaluate_records(
        ctx, trusted=trusted, candidate=candidate, output=output, mode=args.mode,
        policy=policy, policy_source=policy_source, identity=identity, records=records,
        home=output / "private",
    )
    run_id = args.run_id or prior_run_id(output)
    document = status_document(
        mode=args.mode, result=result, reason=reason, identity=identity,
        candidate_checks=records.get("checks") or [], evaluator_checks=checks,
        source_summary=source_summary, artifact_summary=artifact_summary, report=report,
        observations=[str(item) for item in (records.get("observations") or [])],
        run_id=run_id,
    )
    write_status(output / "status.json", document)
    emit({"action": "evaluate", "result": result, "reason": reason, "output": str(output),
          "authoritative": document.get("authoritative"),
          "authoritative_reason": document.get("authoritative_reason"),
          "identity": identity_view(identity)})
    return exit_code(result)


def action_retained(ctx, args):
    if not args.artifact:
        raise Refusal(RESULT_BLOCKED, "artifact-argument-required")
    if args.artifact_run is not None and not re.fullmatch(r"[0-9]{1,20}", str(args.artifact_run)):
        raise Refusal(RESULT_BLOCKED, "artifact-run-invalid")
    policy, policy_source = resolve_policy(args.trusted, args.policy, args.image)
    trusted, candidate, output, private = start_output(args, fresh=True)
    identity = resolve_launch_identity(ctx, args, policy=policy, policy_source=policy_source,
                                       trusted=trusted, candidate=None, private=private)
    try:
        artifact = inspect_retained(
            artifact=args.artifact,
            expected_revision=args.expected_revision,
            artifact_run=args.artifact_run, record=args.record,
        )
        result = RESULT_PASS if args.mode == "rollback" else RESULT_OBSERVED
        reason = "retained-artifact-verified"
        detail = ""
        status = "pass"
    except Refusal as error:
        if args.mode != "release":
            raise
        result = RESULT_OBSERVED
        reason = "release-docs-observation-failed"
        detail = _detail(error)
        artifact = {"files": 0, "bytes": 0, "sha256": "", "escapes": []}
        status = "fail"
    document = status_document(
        mode=args.mode, result=result, reason=reason, identity=identity,
        candidate_checks=[],
        evaluator_checks=[{"id": "retained-artifact", "status": status, "detail": detail}],
        source_summary={}, artifact_summary=artifact,
        report={"present": False, "valid": False, "claimed_result": None,
                "path": "", "authoritative": False},
        observations=["release-docs-sync is read-only and cannot block product release"],
        detail=detail, run_id=args.run_id,
    )
    write_status(output / "status.json", document)
    emit({"action": args.mode, "result": result, "reason": reason, "output": str(output),
          "identity": identity_view(identity)})
    return 0 if args.mode == "release" else exit_code(result)


# ---------------------------------------------------------------------------
# External link and declared-dependency observation (--mode links)
#
# Observation-only, read-only and non-blocking. This mode exists so the optional
# scheduled lane can enumerate external links and declared dependencies without
# turning a third-party site's behaviour into a merge or release gate:
#   * no network fetch happens without an explicit `--network`;
#   * a fetch is refused for any non-http(s) URL and for a literal loopback,
#     private, link-local, reserved or local-only host, and redirects are never
#     followed, so one recorded URL cannot become a probe of an internal service;
#   * reachability results are observations only and can never set
#     `authoritative`, produce a `pass` or change `blocking_release`;
#   * any completed inventory exits 0, even when URLs look wrong; only a usage
#     or IO error exits 1.
# The authoritative lanes (pr/main/release/rollback) never read this record.

LINKS_SCHEMA = "docs-ci-links-observation/1"
LINKS_EXIT_CONTRACT = "0 on completion, 1 on usage or IO error"
LINK_ORIGIN_SOURCE = "source"
LINK_ORIGIN_BUILT = "built"
LINK_SOURCE_SUFFIXES = (".md", ".mdx")
LINK_BUILT_SUFFIXES = (".html", ".htm")
MAX_LINK_FILES = 20000
MAX_LINK_BYTES = 64 << 20
MAX_LINK_OCCURRENCES = 20000
MAX_RECORDED_OCCURRENCES = 4000
MAX_URL_LENGTH = 2048
MAX_REACHABILITY_URLS = 500
MIN_REACHABILITY_TIMEOUT = 1
MAX_REACHABILITY_TIMEOUT = 30
DEFAULT_REACHABILITY_TIMEOUT = 3
DEFAULT_REACHABILITY_BUDGET = 420
DEFAULT_REACHABILITY_MAX_URLS = 200
LINK_USER_AGENT = "agentic-sdd-docs-links-observation/1 (read-only inventory)"

HTML_HREF_PATTERN = re.compile(r"""(?is)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'`=<>]+))""")
HTML_SRC_PATTERN = re.compile(r"""(?is)\bsrc\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'`=<>]+))""")
MARKDOWN_LINK_PATTERN = re.compile(r"(?<!!)\[[^\]]*\]\(\s*<?([A-Za-z][A-Za-z0-9+.\-]*:[^\s)<>]*)")
MARKDOWN_IMAGE_PATTERN = re.compile(r"!\[[^\]]*\]\(\s*<?([A-Za-z][A-Za-z0-9+.\-]*:[^\s)<>]*)")
MARKDOWN_REFERENCE_PATTERN = re.compile(r"(?m)^ {0,3}\[[^\]]+\]:\s*<?(\S+?)(?:>|\s|$)")
AUTOLINK_PATTERN = re.compile(r"<((?:https?:)?//[^>\s]+)>")
FENCE_TOGGLE_PATTERN = re.compile(r"^ {0,3}(`{3,}|~{3,})")
INLINE_CODE_PATTERN = re.compile(r"`[^`\n]*`")

SOURCE_LINK_PATTERNS = (
    ("link", MARKDOWN_LINK_PATTERN),
    ("image", MARKDOWN_IMAGE_PATTERN),
    ("reference-definition", MARKDOWN_REFERENCE_PATTERN),
    ("autolink", AUTOLINK_PATTERN),
    ("html-href", HTML_HREF_PATTERN),
    ("html-src", HTML_SRC_PATTERN),
)
BUILT_LINK_PATTERNS = (
    ("html-href", HTML_HREF_PATTERN),
    ("html-src", HTML_SRC_PATTERN),
)


class LinksError(Exception):
    """A links-observation usage or IO error; maps to exit 1, never to a verdict."""

    def __init__(self, reason, detail=""):
        super().__init__(reason)
        self.reason = reason
        self.detail = detail


class LinksUsageError(LinksError):
    """Invalid invocation of the observation mode."""


class LinksArgumentParser(argparse.ArgumentParser):
    """Usage errors exit 1, not argparse's default 2 (links scheme is 0/1)."""

    def error(self, message):
        raise LinksUsageError("usage-error", message)


def links_mode_requested(argv):
    for index, item in enumerate(argv):
        if item == "--mode" and index + 1 < len(argv) and argv[index + 1] == "links":
            return True
        if item == "--mode=links":
            return True
    return False


def links_parser():
    parser = LinksArgumentParser(
        prog="docs-ci.py --mode links",
        description=(
            "Read-only external-link and declared-dependency observation. "
            "Never authoritative, never blocks a release, and never fetches a "
            "URL unless --network is passed."
        ),
        epilog=(
            "Exit: 0 a completed inventory was recorded; 1 a usage or IO error. "
            "Reachability results are observations and cannot change the result."
        ),
    )
    parser.add_argument("--mode", choices=("links",), default="links",
                        help="observation mode (links only)")
    parser.add_argument("--trusted", required=True,
                        help="checkout to inventory; read only, never modified")
    parser.add_argument("--output", required=True,
                        help="launcher-owned evidence directory (absolute)")
    parser.add_argument("--artifact",
                        help="built site directory to inventory (default: <trusted>/website/dist)")
    parser.add_argument("--network", action="store_true",
                        help="probe each unique external URL once; observations only")
    parser.add_argument("--timeout", type=int, default=DEFAULT_REACHABILITY_TIMEOUT,
                        help="per-request timeout in seconds, %d-%d"
                             % (MIN_REACHABILITY_TIMEOUT, MAX_REACHABILITY_TIMEOUT))
    parser.add_argument("--max-urls", type=int, default=DEFAULT_REACHABILITY_MAX_URLS,
                        help="maximum unique URLs probed with --network (1-%d)" % MAX_REACHABILITY_URLS)
    parser.add_argument("--budget-seconds", type=int, default=DEFAULT_REACHABILITY_BUDGET,
                        help="overall reachability budget in seconds (1-3600)")
    return parser


def mask_markdown_code(text):
    """Blank fenced blocks and inline code spans, preserving offsets and lines."""
    lines = text.split("\n")
    fence = ""
    masked = []
    for line in lines:
        toggle = FENCE_TOGGLE_PATTERN.match(line)
        if fence:
            if toggle and toggle.group(1)[0] == fence:
                fence = ""
            masked.append(" " * len(line))
            continue
        if toggle:
            fence = toggle.group(1)[0]
            masked.append(" " * len(line))
            continue
        masked.append(INLINE_CODE_PATTERN.sub(lambda match: " " * len(match.group(0)), line))
    return "\n".join(masked)


def classify_external_link(value, *, same_site_host=""):
    """Return the external-URL record for an href/URL, or None when not external."""
    if not isinstance(value, str):
        return None
    text = value.strip()
    if not text or len(text) > MAX_URL_LENGTH or "\x00" in text:
        return None
    if text.startswith("//"):
        scheme = "protocol-relative"
        remainder = text[2:]
    else:
        scheme, separator, remainder = text.partition("://")
        if not separator:
            return None
        scheme = scheme.lower()
        if scheme not in ("http", "https"):
            return None
    authority = remainder.split("/", 1)[0].split("?", 1)[0].split("#", 1)[0]
    host = authority.rsplit("@", 1)[-1]
    if host.startswith("["):
        closing = host.find("]")
        host = host[:closing + 1] if closing != -1 else host
    else:
        host = host.split(":", 1)[0]
    host = host.lower()
    if not host:
        return None
    scope = "same-site" if same_site_host and host == same_site_host else "external"
    return {"url": text, "scheme": scheme, "host": host, "scope": scope}


def link_occurrences(text, patterns):
    """Yield (kind, raw value, 1-based line) for every pattern hit in text."""
    if not text:
        return []
    line_starts = [0]
    line_starts.extend(index + 1 for index, char in enumerate(text) if char == "\n")
    found = []
    for kind, pattern in patterns:
        for match in pattern.finditer(text):
            raw = ""
            for group in match.groups():
                if group:
                    raw = group
                    break
            if not raw:
                continue
            found.append((kind, raw, bisect.bisect_right(line_starts, match.start())))
    return found


def _link_files(root, suffixes):
    """Bounded regular-file list for one read-only inventory root."""
    entries = []
    skipped = []
    total = 0
    for directory, dirs, names in os.walk(root, followlinks=False):
        base = Path(directory)
        kept = []
        for name in sorted(dirs):
            path = base / name
            if name in EXCLUDED_ANY or name in SENSITIVE_DIRS:
                skipped.append({"path": path.relative_to(root).as_posix(),
                                "reason": "excluded-directory"})
                continue
            if path.is_symlink():
                skipped.append({"path": path.relative_to(root).as_posix(), "reason": "symlink"})
                continue
            kept.append(name)
        dirs[:] = kept
        for name in sorted(names):
            path = base / name
            if Path(name).suffix.lower() not in suffixes:
                continue
            relative = path.relative_to(root).as_posix()
            info = path.lstat()
            if stat.S_ISLNK(info.st_mode):
                skipped.append({"path": relative, "reason": "symlink"})
                continue
            if not stat.S_ISREG(info.st_mode):
                skipped.append({"path": relative, "reason": "special-file"})
                continue
            if len(entries) >= MAX_LINK_FILES or total + info.st_size > MAX_LINK_BYTES:
                skipped.append({"path": relative, "reason": "exceeds-bound"})
                continue
            entries.append((path, relative))
            total += info.st_size
    return entries, skipped, total


def inventory_links(root, suffixes, *, origin, patterns, same_site_host, markdown):
    """Read-only, bounded inventory of one tree; never raises on odd content."""
    root = Path(root)
    summary = {
        "available": False, "root": str(root), "files": 0, "bytes": 0,
        "skipped": [], "truncated": False,
    }
    if root.is_symlink() or not root.is_dir():
        return summary, []
    entries, skipped, total = _link_files(root, suffixes)
    occurrences = []
    for path, relative in entries:
        try:
            data = path.read_bytes()
        except OSError as error:
            raise LinksError("inventory-file-unreadable",
                             relative + ":" + type(error).__name__)
        text = data.decode("utf-8", errors="replace")
        if markdown:
            text = mask_markdown_code(text)
        seen = set()
        for kind, raw, line in link_occurrences(text, patterns):
            classified = classify_external_link(raw, same_site_host=same_site_host)
            if classified is None:
                continue
            key = (classified["url"], kind, line)
            if key in seen:
                continue
            seen.add(key)
            if len(occurrences) >= MAX_LINK_OCCURRENCES:
                summary["truncated"] = True
                break
            occurrences.append({
                "url": classified["url"], "host": classified["host"],
                "scheme": classified["scheme"], "scope": classified["scope"],
                "kind": kind, "origin": origin, "path": relative, "line": line,
            })
        if summary["truncated"]:
            break
    summary.update({"available": True, "files": len(entries), "bytes": total,
                    "skipped": skipped})
    return summary, occurrences


def summarize_links(occurrences):
    """Exact counts, per-host and per-URL listings for external occurrences only."""
    external = [entry for entry in occurrences if entry["scope"] == "external"]
    urls = {}
    hosts = {}
    kinds = {}
    schemes = {}
    for entry in external:
        record = urls.setdefault(entry["url"], {
            "url": entry["url"], "host": entry["host"], "occurrences": 0,
            "kinds": set(), "origins": set(),
        })
        record["occurrences"] += 1
        record["kinds"].add(entry["kind"])
        record["origins"].add(entry["origin"])
        hosts[entry["host"]] = hosts.get(entry["host"], 0) + 1
        kinds[entry["kind"]] = kinds.get(entry["kind"], 0) + 1
        schemes[entry["scheme"]] = schemes.get(entry["scheme"], 0) + 1
    listing = []
    for key in sorted(urls):
        record = urls[key]
        listing.append({
            "url": record["url"], "host": record["host"],
            "occurrences": record["occurrences"],
            "kinds": sorted(record["kinds"]), "origins": sorted(record["origins"]),
        })
    return {
        "external_occurrences": len(external),
        "unique_external_urls": len(listing),
        "unique_hosts": len(hosts),
        "internal_absolute_occurrences": len(occurrences) - len(external),
        "by_kind": {kind: kinds[kind] for kind in sorted(kinds)},
        "by_scheme": {scheme: schemes[scheme] for scheme in sorted(schemes)},
        "hosts": [{"host": host, "occurrences": hosts[host]} for host in sorted(hosts)],
        "urls": listing,
    }


def links_policy(trusted):
    """Base-owned policy when it loads; embedded defaults otherwise, always labelled."""
    try:
        return resolve_policy(trusted)[0], "resolved"
    except Refusal as error:
        return copy.deepcopy(DEFAULT_POLICY), "embedded-default-after-policy-refusal:" + error.reason


def manifest_site_host(trusted, policy):
    """The published site host, so built same-site absolute URLs are not external."""
    path = Path(trusted) / policy["site_manifest"]
    try:
        if path.is_symlink() or not path.is_file():
            return ""
        data = read_json(path, limit=MAX_JSON_BYTES)
    except (OSError, ValueError):
        return ""
    site = data.get("site") if isinstance(data, dict) else None
    if not isinstance(site, str) or "://" not in site:
        return ""
    return site.split("://", 1)[1].split("/", 1)[0].split(":", 1)[0].lower()


def dependency_observation(trusted, policy):
    """Declared dependency inventory; reads files only, never a registry."""
    site_root = Path(trusted) / policy["site_root"]
    package = site_root / "package.json"
    lock = site_root / "package-lock.json"
    record = {
        "available": False, "package_json": "package.json", "lockfile": {"present": False},
        "declared": [], "declared_count": 0, "notes": [],
    }
    try:
        if package.is_symlink() or not package.is_file():
            record["notes"].append("package.json unavailable")
            return record
        data = read_json(package, limit=MAX_JSON_BYTES)
    except (OSError, ValueError):
        record["notes"].append("package.json unreadable")
        return record
    if not isinstance(data, dict):
        record["notes"].append("package.json is not an object")
        return record
    declared = {}
    for group in ("dependencies", "devDependencies"):
        block = data.get(group)
        if not isinstance(block, dict):
            continue
        for name in sorted(block):
            if isinstance(block[name], str):
                declared[name] = {"range": block[name], "group": group}
    lock_record = {"present": False}
    try:
        if lock.is_file() and not lock.is_symlink():
            lock_record = {
                "present": True,
                "sha256": sha256_file(lock, limit=MAX_JSON_BYTES),
                "bytes": lock.stat().st_size,
            }
            lock_data = read_json(lock, limit=MAX_JSON_BYTES)
            packages = lock_data.get("packages") if isinstance(lock_data, dict) else None
            if isinstance(packages, dict):
                lock_record["resolved_packages"] = len(packages)
    except (OSError, ValueError):
        lock_record["valid"] = False
    record.update({
        "available": True,
        "lockfile": lock_record,
        "declared": [{"name": name, "range": declared[name]["range"],
                      "group": declared[name]["group"]} for name in sorted(declared)],
        "declared_count": len(declared),
    })
    return record


def reachability_status(http_status):
    if http_status is None:
        return "unreachable"
    if 200 <= http_status <= 399:
        return "reachable"
    if 400 <= http_status <= 499:
        return "client-error"
    if 500 <= http_status <= 599:
        return "server-error"
    return "unexpected-status"


PROBE_SCHEMES = frozenset({"http", "https"})
LOCAL_HOST_SUFFIXES = (".localhost", ".local", ".internal", ".home.arpa")


def _literal_address(host):
    """The host as an IP address, or None when it is a name (IPv4-mapped unwrapped)."""
    import ipaddress

    try:
        address = ipaddress.ip_address(host.strip().strip("[]"))
    except ValueError:
        return None
    if address.version == 6 and address.ipv4_mapped is not None:
        return address.ipv4_mapped
    return address


def probe_refusal(url):
    """Why the observation must not fetch this URL, or "" when it may.

    Only http(s) is fetched, and only for a public host: a literal loopback,
    private, link-local, reserved, multicast or unspecified address is refused,
    as are the local-only name forms (`localhost`, `*.localhost`, `*.local`,
    `*.internal`, `*.home.arpa`). This is refusal before any socket is opened.
    """
    import urllib.parse

    try:
        parts = urllib.parse.urlsplit(url)
    except ValueError:
        return "url-unparseable"
    if parts.scheme.lower() not in PROBE_SCHEMES:
        return "scheme-not-http"
    host = (parts.hostname or "").strip().lower().rstrip(".")
    if not host:
        return "host-missing"
    address = _literal_address(host)
    if address is not None:
        if (address.is_private or address.is_loopback or address.is_link_local
                or address.is_multicast or address.is_reserved or address.is_unspecified):
            return "local-target"
        return ""
    if host == "localhost" or host.endswith(LOCAL_HOST_SUFFIXES):
        return "local-target"
    return ""


def _probe_opener():
    """An opener that never follows a redirect, so no hop is fetched implicitly."""
    import urllib.request

    class _NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            return None

    return urllib.request.build_opener(_NoRedirect)


def _probe_fetch(opener, request, timeout):
    """The single network call the observation makes; patched by focused tests."""
    return opener.open(request, timeout=timeout)


def probe_one_url(url, *, timeout):
    """One bounded GET for an allowed URL; imports network machinery only then."""
    import urllib.error
    import urllib.request

    refusal = probe_refusal(url)
    if refusal:
        return {"url": url, "status": "not-probed", "http_status": None,
                "error": refusal, "refused": True}
    observed = {"url": url, "status": "unreachable", "http_status": None, "error": "",
                "refused": False}
    request = urllib.request.Request(
        url, method="GET", headers={"User-Agent": LINK_USER_AGENT, "Accept": "*/*"},
    )
    try:
        with _probe_fetch(_probe_opener(), request, timeout) as response:
            observed["http_status"] = getattr(response, "status", None)
            response.read(2048)
            observed["status"] = reachability_status(observed["http_status"])
    except urllib.error.HTTPError as error:
        observed["http_status"] = error.code
        observed["status"] = reachability_status(error.code)
        if 300 <= error.code <= 399:
            observed["error"] = "redirect-not-followed"
    except Exception as error:  # noqa: BLE001 - an observation must never abort
        observed["error"] = type(error).__name__
    return observed


def probe_external_urls(urls, *, timeout, max_urls, budget_seconds):
    """Sequential, bounded reachability observations; never influences a verdict."""
    import time as _time

    started = _time.monotonic()
    results = []
    not_probed = 0
    for index, url in enumerate(urls):
        if index >= max_urls or _time.monotonic() - started > budget_seconds:
            not_probed = len(urls) - index
            break
        results.append(probe_one_url(url, timeout=timeout))
    return {
        "performed": True,
        "timeout_seconds": timeout,
        "max_urls": max_urls,
        "budget_seconds": budget_seconds,
        "probed": len(results),
        "not_probed": not_probed,
        "refused": sum(1 for entry in results if entry.get("refused")),
        "affects_result": False,
        "results": results,
    }


def prepare_links_roots(trusted, output, artifact):
    trusted_path = Path(trusted)
    if trusted_path.is_symlink() or not trusted_path.is_dir():
        raise LinksError("trusted-root-unavailable", str(trusted_path))
    trusted_path = trusted_path.resolve()
    home = Path.home().resolve()
    if trusted_path == Path("/") or trusted_path == home:
        raise LinksError("unsafe-trusted-root", str(trusted_path))
    output_path = Path(output)
    if not output_path.is_absolute():
        raise LinksError("absolute-output-required", str(output_path))
    if ".." in output_path.parts:
        raise LinksError("unsafe-output-path", str(output_path))
    if output_path.is_symlink():
        raise LinksError("output-symlink-refused", str(output_path))
    resolved_output = output_path.resolve()
    if resolved_output == home or home in resolved_output.parents:
        raise LinksError("home-directory-refused", str(output_path))
    if (resolved_output == trusted_path
            or trusted_path in resolved_output.parents
            or resolved_output in trusted_path.parents):
        raise LinksError("output-overlaps-input", str(output_path))
    if resolved_output.exists() and not resolved_output.is_dir():
        raise LinksError("output-not-a-directory", str(resolved_output))
    resolved_output.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_path = None
    if artifact:
        candidate = Path(artifact)
        if not candidate.is_symlink() and candidate.is_dir():
            artifact_path = candidate.resolve()
    return trusted_path, resolved_output, artifact_path


def build_links_observation(*, trusted, artifact, network, timeout, max_urls, budget_seconds):
    """Assemble the observation document; every field is read-only evidence."""
    policy, policy_source = links_policy(trusted)
    same_site_host = manifest_site_host(trusted, policy)
    source_root = Path(trusted) / policy["site_root"] / "src" / "content" / "docs"
    built_root = Path(artifact) if artifact else Path(trusted) / policy["artifact_root"]

    source_summary, source_occurrences = inventory_links(
        source_root, LINK_SOURCE_SUFFIXES, origin=LINK_ORIGIN_SOURCE,
        patterns=SOURCE_LINK_PATTERNS, same_site_host=same_site_host, markdown=True,
    )
    built_summary, built_occurrences = inventory_links(
        built_root, LINK_BUILT_SUFFIXES, origin=LINK_ORIGIN_BUILT,
        patterns=BUILT_LINK_PATTERNS, same_site_host=same_site_host, markdown=False,
    )
    occurrences = source_occurrences + built_occurrences
    summary = summarize_links(occurrences)
    recorded = occurrences[:MAX_RECORDED_OCCURRENCES]

    reachability = {
        "performed": False,
        "reason": "network-not-requested",
        "refused": 0,
        "affects_result": False,
        "results": [],
    }
    if network:
        reachability = probe_external_urls(
            [entry["url"] for entry in summary["urls"]],
            timeout=timeout, max_urls=max_urls, budget_seconds=budget_seconds,
        )

    observations = [
        "external links are enumerated from source pages and, when present, the built artifact",
        "no URL is fetched unless --network is passed; reachability results are observations only",
        "this record never sets authoritative or blocking_release and never produces a pass",
    ]
    if reachability.get("refused"):
        observations.append(
            "%d non-http(s) or local/private target(s) were refused before any request"
            % reachability["refused"]
        )
    if reachability.get("performed"):
        observations.append(
            "redirects are not followed; a 3xx answer is recorded as reachable and "
            "the redirect target is not fetched"
        )
    if not source_summary["available"]:
        observations.append("source docs directory unavailable: inventory covers the built artifact only")
    if not built_summary["available"]:
        observations.append("built artifact unavailable: inventory covers source pages only")
    if summary["internal_absolute_occurrences"]:
        observations.append(
            "%d same-origin absolute URL(s) excluded from the external inventory"
            % summary["internal_absolute_occurrences"]
        )
    if source_summary["truncated"] or built_summary["truncated"]:
        observations.append("occurrence list truncated at the bounded maximum; counts are partial")
    if len(occurrences) > len(recorded):
        observations.append(
            "detail list records the first %d occurrences; counts remain exact"
            % MAX_RECORDED_OCCURRENCES
        )

    return {
        "schema": LINKS_SCHEMA,
        "mode": "links",
        "result": RESULT_OBSERVED,
        "reason": "external-link-and-dependency-inventory",
        "authoritative": False,
        "blocking_release": False,
        "exit_contract": LINKS_EXIT_CONTRACT,
        "inputs": {
            "trusted": str(trusted),
            "source_root": str(source_root),
            "source_available": source_summary["available"],
            "artifact": str(built_root),
            "artifact_available": built_summary["available"],
            "same_site_host": same_site_host,
            "policy_source": policy_source,
        },
        "network_requested": bool(network),
        "source": {key: source_summary[key] for key in
                   ("available", "root", "files", "bytes", "skipped", "truncated")},
        "built": {key: built_summary[key] for key in
                  ("available", "root", "files", "bytes", "skipped", "truncated")},
        "counts": {key: summary[key] for key in
                   ("external_occurrences", "unique_external_urls", "unique_hosts",
                    "internal_absolute_occurrences", "by_kind", "by_scheme")},
        "hosts": summary["hosts"],
        "urls": summary["urls"],
        "occurrences": recorded,
        "occurrences_truncated": len(occurrences) > len(recorded),
        "dependencies": dependency_observation(trusted, policy),
        "reachability": reachability,
        "observations": observations,
        "unknowns": [
            "a third-party site's future state is not observed by this inventory",
            "reachability says nothing about whether a linked claim is still correct",
        ],
        "limitations": [
            "not a link-failure gate, a merge gate, a release gate or an authoritative pass",
            "bare URLs inside prose and fenced code are not counted as links",
        ],
    }


def action_links(args):
    """Run the observation and write exactly one record; exit 0 unless IO/usage failed."""
    timeout = args.timeout
    if not MIN_REACHABILITY_TIMEOUT <= timeout <= MAX_REACHABILITY_TIMEOUT:
        raise LinksUsageError("timeout-out-of-range", str(timeout))
    max_urls = args.max_urls
    if not 1 <= max_urls <= MAX_REACHABILITY_URLS:
        raise LinksUsageError("max-urls-out-of-range", str(max_urls))
    budget = args.budget_seconds
    if not 1 <= budget <= 3600:
        raise LinksUsageError("budget-seconds-out-of-range", str(budget))
    trusted, output, artifact = prepare_links_roots(args.trusted, args.output, args.artifact)
    document = build_links_observation(
        trusted=trusted, artifact=artifact, network=args.network,
        timeout=timeout, max_urls=max_urls, budget_seconds=budget,
    )
    record = output / "links.json"
    write_status(record, document)
    emit({
        "mode": "links",
        "result": document["result"],
        "reason": document["reason"],
        "authoritative": False,
        "blocking_release": False,
        "network": document["network_requested"],
        "counts": document["counts"],
        "output": str(output),
        "record": str(record),
    })
    return 0


def run_links_cli(argv, *, context=None):
    """Links entrypoint: 0 on a completed inventory, 1 on a usage or IO error."""
    parser = links_parser()
    try:
        args = parser.parse_args(list(argv))
    except SystemExit as error:
        return int(error.code or 0)
    except LinksUsageError as error:
        return _links_refusal(error)
    try:
        return action_links(args)
    except LinksUsageError as error:
        return _links_refusal(error)
    except LinksError as error:
        return _links_refusal(error)
    except (OSError, ValueError) as error:
        return _links_refusal(LinksError("unavailable-input-or-operation",
                                         type(error).__name__))


def _links_refusal(error):
    emit({
        "mode": "links",
        "result": "error",
        "reason": error.reason,
        "detail": str(error.detail),
        "authoritative": False,
        "blocking_release": False,
        "exit_contract": LINKS_EXIT_CONTRACT,
    })
    print("docs-ci.py --mode links: " + error.reason
          + (": " + str(error.detail) if error.detail else ""), file=sys.stderr)
    return 1


# ---------------------------------------------------------------------------
# CLI


def build_parser():
    parser = argparse.ArgumentParser(
        prog="docs-ci.py",
        description="Trusted docs-only launcher, identity emitter and base-owned evaluator.",
    )
    parser.add_argument("--trusted", required=True, help="reviewed base checkout")
    parser.add_argument("--candidate", help="candidate checkout (data only)")
    parser.add_argument("--output", required=True, help="launcher-owned evidence directory")
    parser.add_argument("--mode", required=True, choices=("pr", "main", "release", "rollback"))
    parser.add_argument("--action", default="run",
                        choices=("run", "identity", "provision", "evaluate"))
    parser.add_argument("--policy", help="policy file inside --trusted")
    parser.add_argument("--image", help="adopted digest-pinned toolchain image")
    parser.add_argument("--base-sha")
    parser.add_argument("--candidate-sha")
    parser.add_argument("--candidate-ref")
    parser.add_argument("--artifact",
                        help="main: publish the validated site here; "
                             "release/rollback: retained site directory")
    parser.add_argument("--record",
                        help="retained trust record (status.json) for a flat retained site")
    parser.add_argument("--expected-revision",
                        help="retained artifact's trusted revision; rollback binds to this, "
                             "never to the current HEAD")
    parser.add_argument("--artifact-run",
                        help="prior main run id whose retained artifact is reused")
    parser.add_argument("--run-id", help="recorded run provenance for this invocation")
    return parser


def run_cli(argv, *, context=None):
    ctx = context or Context()
    argv = list(argv)
    # `--mode links` is the separately contracted observation lane (exit 0/1);
    # it is dispatched before the frozen launch parser so pr/main/release/
    # rollback behaviour, its mode choices and its 0/1/2 exit contract cannot
    # change.
    if links_mode_requested(argv):
        return run_links_cli(argv, context=ctx)
    parser = build_parser()
    try:
        args = parser.parse_args(list(argv))
    except SystemExit as error:
        return int(error.code or 0)
    for name in ("base_sha", "candidate_sha", "expected_revision"):
        value = getattr(args, name)
        if value is None:
            continue
        if not FULL_SHA.match(str(value).lower()):
            emit({"result": RESULT_BLOCKED, "reason": "sha-format-invalid", "detail": name})
            return 2
        setattr(args, name, str(value).lower())
    if args.run_id is not None and not re.fullmatch(r"[0-9]{1,20}", str(args.run_id)):
        emit({"result": RESULT_BLOCKED, "reason": "run-id-invalid", "detail": "run_id"})
        return 2
    if args.policy:
        args.policy = str(Path(args.policy).resolve())
    try:
        if args.action == "identity":
            return action_identity(ctx, args)
        if args.action == "provision":
            return action_provision(ctx, args)
        if args.action == "evaluate":
            return action_evaluate(ctx, args)
        if args.mode in ("release", "rollback"):
            return action_retained(ctx, args)
        return action_run(ctx, args)
    except RunnerTimeout:
        error = Refusal(RESULT_BLOCKED, "bounded-command-timeout")
        return _refuse(args, error)
    except Refusal as error:
        return _refuse(args, error)
    except (OSError, ValueError) as error:
        return _refuse(args, Refusal(RESULT_BLOCKED, "unavailable-input-or-operation",
                                     type(error).__name__))
    except (KeyError, TypeError, IndexError) as error:
        return _refuse(args, Refusal(RESULT_BLOCKED, "invalid-record-or-policy",
                                     type(error).__name__))


def _refuse(args, error):
    document = blocked_document(args.mode, error.result, error.reason, detail=str(error.detail),
                               run_id=getattr(args, "run_id", "") or "")
    written = False
    try:
        output = Path(args.output)
        if output.is_dir() and not output.is_symlink():
            write_status(output / "status.json", document)
            written = True
    except (OSError, Refusal):
        written = False
    emit({"result": error.result, "reason": error.reason, "detail": str(error.detail),
          "status": "status.json" if written else ""})
    return 0 if args.mode == "release" else 2


def main(argv=None, context=None):
    return run_cli(sys.argv[1:] if argv is None else argv, context=context)


if __name__ == "__main__":
    sys.exit(main())
