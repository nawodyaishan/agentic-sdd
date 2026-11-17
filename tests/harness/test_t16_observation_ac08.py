"""Feature 022 B7/T16 offline fixtures for AC-08.

AC-08 requires synthetic workflow-event evidence that the token publication
route reaches observation, that root/relative CSS/JS references resolve the base
path once, and that delayed or failed observations stay recorded without
silently becoming successful or becoming a new blocking CDN policy.

Two workflows are read as text: ``docs.yml`` (the callable observation entrypoint
and the deployment observation) and the dedicated ``docs-observe.yml`` observer
that reaches a completed ``Release publish`` run through ``workflow_run``. The
observation step bodies are extracted verbatim and executed as local shell with
stub ``curl``, ``sleep``, ``git`` and ``gh`` binaries on ``PATH``, and the
resulting records are asserted. Stdlib only, matching the repository harness
style (no YAML dependency).

Event model used below (`TOKEN_SUPPRESSED_EXCEPTIONS`): GitHub starts no new
workflow run for an event that the repository's `GITHUB_TOKEN` created, with
`workflow_dispatch` and `repository_dispatch` as the documented exceptions. A
reusable `workflow_call` is a direct call from the calling job, not an event, so
suppression does not apply to it; a `workflow_run` event is created by the
platform for a completed run, so it is not token-suppressed either.
"""

import json
import os
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
WORKFLOW = ROOT / ".github" / "workflows" / "docs.yml"
OBSERVE_WORKFLOW = ROOT / ".github" / "workflows" / "docs-observe.yml"
PUBLISH_WORKFLOW = ROOT / ".github" / "workflows" / "release-publish.yml"
MANIFEST = ROOT / "website" / "docs-manifest.json"

OBSERVATION_JOB = "post-publish-observation"
FALLBACK_JOB = "release-status"
SMOKE_JOB = "smoke"
ROUTE_STEP = "Record the delivery route of this observation"
BINDING_STEP = "Bind the observation to the published revision"
SMOKE_STEP = "Poll for a servable deployment and record the observation"
SMOKE_UPLOAD_STEP = "Upload the deployment observation"

OBSERVE_JOB = "observe"
RESOLVE_STEP = "Resolve the released tag and commit (read-only)"
OBSERVE_UPLOAD_STEP = "Upload post-publish observation evidence"
OBSERVE_ROUTE = "workflow-run"
OBSERVE_DELIVERY = "post-publish-run-event"


REVISION = "0123456789abcdef0123456789abcdef01234567"
OTHER_REVISION = "fedcba9876543210fedcba9876543210fedcba98"
SOURCE_RUN = "314159"
OBSERVATION_RUN = "271828"

# Documented GitHub behaviour: events created with the repository token start no
# workflow run, except these two dispatch events.
TOKEN_SUPPRESSED_EXCEPTIONS = {"workflow_dispatch", "repository_dispatch"}


# ---------------------------------------------------------------------------
# Workflow text helpers (no YAML dependency, matching tests/harness style)
# ---------------------------------------------------------------------------


def workflow_text():
    return WORKFLOW.read_text(encoding="utf-8")


def comment_prose(text):
    """All comment text with the `#` markers and line wrapping removed."""
    lines = []
    for line in text.splitlines():
        stripped = line.lstrip()
        if stripped.startswith("#"):
            lines.append(stripped[1:].strip())
    return " ".join(" ".join(lines).split())


def top_section(text, key):
    """Member lines of a column-0 `key:` block, comments and blanks removed."""
    lines = text.splitlines()
    try:
        start = lines.index(key + ":")
    except ValueError:
        return None
    body = []
    for line in lines[start + 1:]:
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if not line.startswith(" "):
            break
        body.append(line)
    return body


def trigger_names(text):
    """Top-level trigger names declared under `on:`."""
    body = top_section(text, "on")
    if body is None:
        return None
    names = []
    for line in body:
        indent = len(line) - len(line.lstrip(" "))
        if indent == 2 and re.match(r"^  [A-Za-z_][A-Za-z0-9_-]*:\s*$", line):
            names.append(line.strip()[:-1])
    return names


def workflow_call_inputs(text):
    """`on.workflow_call.inputs` as {name: {field: value}}."""
    lines = text.splitlines()
    try:
        start = lines.index("  workflow_call:")
    except ValueError:
        return None
    inputs = {}
    current = None
    for line in lines[start + 1:]:
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        indent = len(line) - len(line.lstrip(" "))
        if indent <= 2:
            break
        if indent == 4:
            continue
        if indent == 6 and line.strip().endswith(":"):
            current = line.strip()[:-1]
            inputs[current] = {}
            continue
        if indent == 8 and current is not None and ":" in line:
            key, value = line.strip().split(":", 1)
            inputs[current][key.strip()] = value.strip().strip('"')
    return inputs


def job_blocks(text):
    """Job name -> its member lines, split by indentation."""
    jobs = {}
    lines = text.splitlines()
    try:
        start = lines.index("jobs:")
    except ValueError:
        return jobs
    current = None
    for line in lines[start + 1:]:
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if not line.startswith(" "):
            break
        indent = len(line) - len(line.lstrip(" "))
        if indent == 2:
            current = line.strip().split(":", 1)[0]
            jobs[current] = []
            continue
        if indent >= 4 and current is not None:
            jobs[current].append(line)
    return jobs


def job_scalar(job_lines, key):
    """A job-level scalar (`if:`, `timeout-minutes:`, `needs:`, ...)."""
    for line in job_lines:
        match = re.match(r"^\s{4}" + re.escape(key) + r":\s*(.*)$", line)
        if match:
            value = match.group(1).strip()
            return value
    return None


def step_block(job_lines, name):
    """Lines of one step body, located by its `- name:` line and indentation."""
    start = None
    indent = None
    for index, line in enumerate(job_lines):
        if line.strip() == "- name: " + name:
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


def step_run_text(block):
    """The dedented body of a step's `run: |` block scalar."""
    if block is None:
        return None
    for index, line in enumerate(block):
        match = re.match(r"^(\s*)run:\s*\|\s*$", line)
        if not match:
            continue
        base = len(match.group(1))
        body = []
        for item in block[index + 1:]:
            if not item.strip():
                body.append("")
                continue
            if len(item) - len(item.lstrip(" ")) <= base:
                break
            body.append(item[base + 2:] if item.startswith(" " * (base + 2)) else item.lstrip())
        return "\n".join(body)
    return None


def job_run_text(job_lines, step_name):
    block = step_block(job_lines, step_name)
    if block is None:
        raise AssertionError("step %r is not declared in this job" % step_name)
    text = step_run_text(block)
    if text is None:
        raise AssertionError("step %r has no run body" % step_name)
    return text


def shell_function(script, name):
    """Extract a `name() { ... }` definition verbatim from a step body."""
    lines = script.splitlines()
    for index, line in enumerate(lines):
        if line.strip() == name + "() {":
            body = [line]
            for item in lines[index + 1:]:
                body.append(item)
                if item.strip() == "}":
                    return "\n".join(body)
            break
    raise AssertionError("function %s() is not defined in the step body" % name)


def step_env(block):
    """`KEY: value` pairs declared under a step's `env:` key."""
    values = {}
    inside = False
    base = None
    for line in block:
        if not inside:
            match = re.match(r"^(\s*)env:\s*$", line)
            if match:
                inside = True
                base = len(match.group(1))
            continue
        indent = len(line) - len(line.lstrip(" "))
        if not line.strip():
            continue
        if indent <= base:
            break
        if ":" in line:
            key, value = line.strip().split(":", 1)
            values[key.strip()] = value.strip().strip('"')
    return values


def observe_text():
    return OBSERVE_WORKFLOW.read_text(encoding="utf-8")


def trigger_block(text, name):
    """Member lines of one `on:` trigger, comments and blanks removed."""
    lines = text.splitlines()
    try:
        start = lines.index("on:")
    except ValueError:
        return None
    body = []
    collecting = False
    for line in lines[start + 1:]:
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if not line.startswith(" "):
            break
        indent = len(line) - len(line.lstrip(" "))
        if not collecting:
            if indent == 2 and line.strip() == name + ":":
                collecting = True
            continue
        if indent <= 2:
            break
        body.append(line)
    return body if collecting else None


def job_permissions(job_lines):
    """A job's `permissions:` map; `permissions: {}` is an empty map."""
    for index, line in enumerate(job_lines):
        match = re.match(r"^(\s*)permissions:\s*(\{\s*\})?\s*$", line)
        if not match:
            continue
        if match.group(2):
            return {}
        base = len(match.group(1))
        values = {}
        for item in job_lines[index + 1:]:
            if not item.strip():
                continue
            if len(item) - len(item.lstrip(" ")) <= base:
                break
            key, value = item.strip().split(":", 1)
            values[key.strip()] = value.strip()
        return values
    return None


def run_bodies(text):
    """Every `run: |` block scalar in a workflow, dedented."""
    lines = text.splitlines()
    bodies = []
    index = 0
    while index < len(lines):
        match = re.match(r"^(\s*)run:\s*\|\s*$", lines[index])
        if not match:
            index += 1
            continue
        base = len(match.group(1))
        body = []
        index += 1
        while index < len(lines):
            line = lines[index]
            if not line.strip():
                body.append("")
                index += 1
                continue
            if len(line) - len(line.lstrip(" ")) <= base:
                break
            body.append(line[base + 2:] if line.startswith(" " * (base + 2))
                        else line.lstrip())
            index += 1
        bodies.append("\n".join(body))
    return bodies



# ---------------------------------------------------------------------------
# Offline execution of an extracted step body
# ---------------------------------------------------------------------------


FAKE_CURL = '''#!/usr/bin/env python3
"""Serve a scripted response for each URL, record the call, emulate -f/-w/-o."""
import json
import os
import sys

state = os.environ["FAKE_STATE"]
argv = sys.argv[1:]
url = argv[-1]
out = None
fmt = None
fail = False
index = 0
while index < len(argv):
    item = argv[index]
    if item in ("-o", "--output"):
        out = argv[index + 1]
        index += 2
        continue
    if item in ("-w", "--write-out"):
        fmt = argv[index + 1]
        index += 2
        continue
    if item in ("-H", "--header", "--max-time"):
        index += 2
        continue
    if item.startswith("-") and not item.startswith("--") and "f" in item.lstrip("-"):
        fail = True
    index += 1

with open(os.path.join(state, "calls"), "a", encoding="utf-8") as handle:
    handle.write(url + "\\n")

with open(os.path.join(state, "responses.json"), encoding="utf-8") as handle:
    responses = json.load(handle)
with open(os.path.join(state, "counters.json"), encoding="utf-8") as handle:
    counters = json.load(handle)

entries = responses.get(url)
if entries is None:
    code, body = 404, ""
else:
    if isinstance(entries[0], int):
        entries = [entries]
    position = min(counters.get(url, 0), len(entries) - 1)
    code, body = entries[position]
counters[url] = counters.get(url, 0) + 1
with open(os.path.join(state, "counters.json"), "w", encoding="utf-8") as handle:
    json.dump(counters, handle)

if fmt:
    sys.stdout.write(str(fmt).replace("%{http_code}", str(code)))
elif out is None:
    sys.stdout.write(body)
if out is not None and out != os.devnull:
    with open(out, "w", encoding="utf-8") as handle:
        handle.write(body)
sys.exit(22 if (fail and code >= 400) else 0)
'''

FAKE_SLEEP = '''#!/bin/sh
echo "$1" >> "$FAKE_STATE/sleeps"
exit 0
'''

FAKE_GIT = '''#!/bin/sh
echo "$*" >> "$FAKE_STATE/git-calls"
case "$1" in
  -C) shift 2 ;;
esac
case "$1" in
  rev-parse) printf '%s\\n' "${FAKE_GIT_HEAD:-}"; exit "${FAKE_GIT_RC:-0}" ;;
esac
echo "unexpected git call: $*" >&2
exit 99
'''

FAKE_GH = '''#!/usr/bin/env python3
"""Serve one scripted read-only API document per path and apply the tested --jq field.

Only `gh api PATH --jq .a.b` is supported, which is all the resolve step uses. A
path without a scripted document fails like an unavailable API. No network.
"""
import json
import os
import sys

state = os.environ["FAKE_STATE"]
argv = sys.argv[1:]
if not argv or argv[0] != "api":
    sys.stderr.write("fake gh: only `api` is supported\\n")
    sys.exit(99)

path = None
jq = None
index = 1
while index < len(argv):
    item = argv[index]
    if item in ("--jq", "-q"):
        jq = argv[index + 1]
        index += 2
        continue
    if item.startswith("-"):
        index += 1
        continue
    if path is None:
        path = item
    index += 1

with open(os.path.join(state, "gh-calls"), "a", encoding="utf-8") as handle:
    handle.write((path or "") + "\\n")

with open(os.path.join(state, "gh-responses.json"), encoding="utf-8") as handle:
    responses = json.load(handle)
document = responses.get(path)
if document is None:
    sys.stderr.write("fake gh: no scripted response for %s\\n" % path)
    sys.exit(1)

if jq is None:
    sys.stdout.write(json.dumps(document))
    sys.exit(0)

value = document
for field in jq.lstrip(".").split("."):
    if not isinstance(value, dict):
        value = None
        break
    value = value.get(field)
if value is None:
    sys.stdout.write("null\\n")
else:
    sys.stdout.write(str(value) + "\\n")
sys.exit(0)
'''



def base_environment(home):
    """A constructed environment: no inherited proxy, token, or Git config."""
    return {
        "PATH": "/usr/local/bin:/usr/bin:/bin",
        "HOME": str(home),
        "LC_ALL": "C",
        "TMPDIR": str(home.parent),
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_TERMINAL_PROMPT": "0",
    }


class OfflineStepCase(unittest.TestCase):
    """Runs one extracted step body with stub binaries and a scratch state dir."""

    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="t16-ac08-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.state = self.root / "state"
        self.state.mkdir()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.out = self.root / "out"
        self.home = self.root / "home"
        self.home.mkdir()
        (self.state / "counters.json").write_text("{}", encoding="utf-8")
        (self.state / "responses.json").write_text("{}", encoding="utf-8")
        for name, body in (("curl", FAKE_CURL), ("sleep", FAKE_SLEEP), ("git", FAKE_GIT)):
            path = self.bin / name
            path.write_text(body, encoding="utf-8")
            path.chmod(0o755)

    def script(self, text):
        return self.root / "step.sh"

    def run_step(self, text, env=None):
        script = self.script(text)
        script.write_text(text, encoding="utf-8")
        environment = base_environment(self.home)
        environment["PATH"] = str(self.bin) + ":" + environment["PATH"]
        environment["FAKE_STATE"] = str(self.state)
        environment.update(env or {})
        return subprocess.run(
            ["bash", str(script)], cwd=str(self.root), env=environment,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=120,
        )

    def record(self, name):
        path = self.out / name
        self.assertTrue(path.is_file(), "expected record %s was not written" % name)
        return json.loads(path.read_text(encoding="utf-8"))

    def calls(self):
        path = self.state / "calls"
        if not path.is_file():
            return []
        return [line for line in path.read_text(encoding="utf-8").splitlines() if line]


# ---------------------------------------------------------------------------
# AC-08: the token publication route must reach observation
# ---------------------------------------------------------------------------


class ObservationRouteTests(OfflineStepCase):
    """The route that survives `GITHUB_TOKEN` event suppression."""

    def test_token_created_release_event_cannot_deliver_the_observation(self):
        text = workflow_text()
        triggers = trigger_names(text)
        self.assertIsNotNone(triggers, "docs.yml declares no `on:` triggers")
        self.assertIn("release", triggers)
        self.assertIn("workflow_call", triggers)

        # Delivery model: an event created with the repository token starts no
        # run, so the `release` trigger cannot reach this workflow for an
        # in-repo publish (release-publish.yml creates the release that way).
        delivered = "release" in TOKEN_SUPPRESSED_EXCEPTIONS
        self.assertFalse(
            delivered,
            "the model says a token-created release event is delivered, which "
            "would make the reusable route unnecessary")

        jobs = job_blocks(text)
        self.assertIn(OBSERVATION_JOB, jobs, "the callable observation job is missing")
        condition = job_scalar(jobs[OBSERVATION_JOB], "if")
        self.assertEqual(
            (condition or "").replace(" ", ""), "github.event_name=='workflow_call'",
            "the callable observation must be selected by the call, not by an event")
        fallback = job_scalar(jobs[FALLBACK_JOB], "if")
        self.assertEqual((fallback or "").replace(" ", ""), "github.event_name=='release'")

        # The workflow states the limitation instead of relying on the trigger.
        prose = comment_prose(text)
        self.assertIn("does not start a new workflow run for an event", prose)
        self.assertIn("publication in this repository runs with the workflow's own",
                      prose.lower())

    def test_in_repo_route_is_the_dedicated_workflow_run_observer(self):
        """The in-repo route is docs-observe.yml; workflow_call stays for others."""
        prose = comment_prose(workflow_text())
        self.assertIn("docs-observe.yml", prose,
                      "the header must name the in-repo observation route")
        self.assertIn("workflow_run", prose)
        self.assertIn("workflow_call", prose,
                      "the reusable entrypoint stays for suitably-scoped callers")
        # The permission caveat: the deploy job's scopes are why a
        # `contents: read` publisher cannot call this file, and why the
        # publisher is not widened.
        self.assertIn("pages: write", prose)
        self.assertIn("id-token: write", prose)
        self.assertIn("contents: read", prose)
        self.assertIn("deliberately not widened", prose)

    def test_reusable_observation_requires_a_full_sha_revision(self):
        inputs = workflow_call_inputs(workflow_text())
        self.assertIsNotNone(inputs, "docs.yml declares no `workflow_call` inputs")
        self.assertIn("revision", inputs)
        self.assertEqual(inputs["revision"].get("required"), "true")
        self.assertEqual(inputs["revision"].get("type"), "string")
        # Provenance-only inputs are optional, so a caller cannot be forced to
        # invent a tag or run id it does not have.
        for name in ("tag", "source_run_id"):
            self.assertEqual(inputs.get(name, {}).get("required"), "false")

        validation = job_run_text(
            job_blocks(workflow_text())[OBSERVATION_JOB],
            "Validate the supplied revision, tag and source run id")
        accepted = {"REVISION": REVISION, "OBSERVED_TAG": "v0.13.0",
                    "SOURCE_RUN_ID": SOURCE_RUN}
        completed = self.run_step(validation, accepted)
        self.assertEqual(completed.returncode, 0, completed.stderr.decode("utf-8", "replace"))

        rejected = [
            {"REVISION": "", "OBSERVED_TAG": "", "SOURCE_RUN_ID": ""},
            {"REVISION": REVISION[:-1], "OBSERVED_TAG": "", "SOURCE_RUN_ID": ""},
            {"REVISION": "main", "OBSERVED_TAG": "", "SOURCE_RUN_ID": ""},
            {"REVISION": REVISION.upper(), "OBSERVED_TAG": "", "SOURCE_RUN_ID": ""},
            {"REVISION": REVISION, "OBSERVED_TAG": 'v1"; rm -rf /', "SOURCE_RUN_ID": ""},
            {"REVISION": REVISION, "OBSERVED_TAG": "", "SOURCE_RUN_ID": "run-12"},
        ]
        for fixture in rejected:
            with self.subTest(fixture=fixture):
                completed = self.run_step(validation, fixture)
                self.assertNotEqual(
                    completed.returncode, 0,
                    "a revision/tag/run-id fixture that must be refused was accepted")

    def test_observation_job_records_its_route_and_publishes_nothing(self):
        text = workflow_text()
        jobs = job_blocks(text)
        job = jobs[OBSERVATION_JOB]

        # Read-only surface: no environment, no secrets, no publishing scope.
        self.assertIsNone(job_scalar(job, "environment"),
                          "the observation job must not use an environment")
        self.assertIn("contents: read", "\n".join(job))
        joined = "\n".join(job)
        for forbidden in ("secrets.", "id-token", "pages: write", "upload-pages-artifact",
                          "deploy-pages", "persist-credentials: true"):
            self.assertNotIn(forbidden, joined,
                             "the observation job must not contain %r" % forbidden)
        timeout = job_scalar(job, "timeout-minutes")
        self.assertIsNotNone(timeout, "the observation job has no timeout")
        self.assertLessEqual(int(timeout), 15)

        # The route record states what was and was not inspected.
        env = {
            "OUT_DIR": str(self.out),
            "OBSERVATION_ROUTE": "workflow-call",
            "OBSERVATION_DELIVERY": "reusable-workflow-call",
            "OBSERVATION_RUN_ID": OBSERVATION_RUN,
            "REVISION": REVISION,
            "PUBLISHED_TAG": "v0.13.0",
            "SOURCE_RUN_ID": SOURCE_RUN,
        }
        completed = self.run_step(job_run_text(job, ROUTE_STEP), env)
        self.assertEqual(completed.returncode, 0, completed.stderr.decode("utf-8", "replace"))
        record = self.record("observation.json")
        self.assertEqual(record["schema"], "docs-observation-route/1")
        self.assertEqual(record["route"], "workflow-call")
        self.assertEqual(record["delivery"], "reusable-workflow-call")
        self.assertEqual(record["event"], "workflow_call")
        self.assertEqual(record["revision"], REVISION)
        self.assertEqual(record["tag"], "v0.13.0")
        self.assertEqual(record["source_run_id"], SOURCE_RUN)
        self.assertEqual(record["observation_run_id"], OBSERVATION_RUN)
        self.assertEqual(record["result"], "observed")
        self.assertIs(record["authoritative"], False)
        self.assertIs(record["blocking_release"], False)
        self.assertEqual(record["artifact_identity"], "not-inspected")
        self.assertEqual(record["source_binding"], "binding.json")

        # The upload is unconditional, so a failed observation still leaves a
        # record instead of an absent artifact.
        upload = step_block(job, "Upload post-publish observation evidence")
        self.assertIsNotNone(upload)
        self.assertIn("if: always()", "\n".join(upload))
        self.assertIn("if-no-files-found: warn", "\n".join(upload))

    def test_route_record_refuses_a_mismatched_identifier(self):
        job = job_blocks(workflow_text())[OBSERVATION_JOB]
        script = job_run_text(job, ROUTE_STEP)
        for fixture in (
            {"REVISION": "main", "PUBLISHED_TAG": "", "SOURCE_RUN_ID": ""},
            {"REVISION": REVISION, "PUBLISHED_TAG": "", "SOURCE_RUN_ID": "abc"},
            {"REVISION": REVISION, "PUBLISHED_TAG": 'x"y', "SOURCE_RUN_ID": ""},
        ):
            with self.subTest(fixture=fixture):
                env = {"OUT_DIR": str(self.out), "OBSERVATION_ROUTE": "workflow-call",
                       "OBSERVATION_DELIVERY": "reusable-workflow-call",
                       "OBSERVATION_RUN_ID": OBSERVATION_RUN}
                env.update(fixture)
                completed = self.run_step(script, env)
                self.assertNotEqual(completed.returncode, 0)
                self.assertFalse((self.out / "observation.json").exists(),
                                 "a refused route must not write an observation record")

    def test_fallback_release_route_records_that_it_is_the_fallback(self):
        job = job_blocks(workflow_text())[FALLBACK_JOB]
        env = {
            "OUT_DIR": str(self.out),
            "OBSERVATION_ROUTE": "release-event",
            "OBSERVATION_DELIVERY": "publication-event",
            "OBSERVATION_RUN_ID": OBSERVATION_RUN,
            "PUBLISHED_TAG": "v0.13.0",
        }
        completed = self.run_step(job_run_text(job, ROUTE_STEP), env)
        self.assertEqual(completed.returncode, 0, completed.stderr.decode("utf-8", "replace"))
        record = self.record("observation.json")
        self.assertEqual(record["route"], "release-event")
        self.assertEqual(record["delivery"], "publication-event")
        self.assertIs(record["blocking_release"], False)
        self.assertIs(record["authoritative"], False)

    def test_any_publisher_callsite_satisfies_the_reusable_contract(self):
        """T16 owns docs.yml only; a publisher callsite is T14/H work.

        `release-publish.yml` is owner-accepted work under review, so it is read
        but never edited here. When that callsite is added it must call this
        reusable workflow with the published revision and read-only scope; until
        then the gap is reported with the batch rather than asserted away.
        """
        publish = ROOT / ".github" / "workflows" / "release-publish.yml"
        text = publish.read_text(encoding="utf-8")
        callsites = [line for line in text.splitlines() if "workflows/docs.yml" in line]
        if not callsites:
            return
        for line in callsites:
            self.assertIn("uses: ./.github/workflows/docs.yml", line.strip())
        jobs = job_blocks(text)
        calling = [name for name, lines in jobs.items()
                   if "workflows/docs.yml" in "\n".join(lines)]
        self.assertTrue(calling, "a docs.yml callsite must be its own job")
        for name in calling:
            body = "\n".join(jobs[name])
            self.assertIn("revision:", body,
                          "the callsite must pass the published revision")
            self.assertIn("contents: read", body,
                          "the callsite must grant only read scope to the observation")


# ---------------------------------------------------------------------------
# AC-08: same-source binding before any observation is recorded
# ---------------------------------------------------------------------------


class SameSourceBindingTests(OfflineStepCase):
    """The observation refuses to record a source it did not check out."""

    def binding_env(self, git_head, git_rc="0"):
        return {
            "TRUSTED_DIR": str(self.root / "trusted"),
            "REVISION": REVISION,
            "OUT_DIR": str(self.out),
            "HOME": str(self.home),
            "FAKE_GIT_HEAD": git_head,
            "FAKE_GIT_RC": git_rc,
        }

    def test_matching_revision_is_recorded_as_bound(self):
        job = job_blocks(workflow_text())[OBSERVATION_JOB]
        completed = self.run_step(job_run_text(job, BINDING_STEP),
                                 self.binding_env(REVISION))
        self.assertEqual(completed.returncode, 0, completed.stderr.decode("utf-8", "replace"))
        record = self.record("binding.json")
        self.assertEqual(record["schema"], "docs-observation-binding/1")
        self.assertEqual(record["expected_revision"], REVISION)
        self.assertEqual(record["observed_revision"], REVISION)
        self.assertIs(record["matched"], True)

    def test_mismatched_revision_fails_before_a_successful_observation(self):
        job = job_blocks(workflow_text())[OBSERVATION_JOB]
        completed = self.run_step(job_run_text(job, BINDING_STEP),
                                 self.binding_env(OTHER_REVISION))
        self.assertNotEqual(completed.returncode, 0,
                            "a mismatched source must fail the observation step")
        record = self.record("binding.json")
        self.assertIs(record["matched"], False)
        self.assertEqual(record["observed_revision"], OTHER_REVISION)
        self.assertFalse((self.out / "observation.json").exists(),
                         "no successful observation record may be written for another source")


# ---------------------------------------------------------------------------
# AC-08: asset references resolve the base path exactly once
# ---------------------------------------------------------------------------


class AssetUrlTests(OfflineStepCase):
    """The smoke step's single origin/base resolver."""

    def resolve_values(self, references):
        script = shell_function(
            job_run_text(job_blocks(workflow_text())[SMOKE_JOB], SMOKE_STEP),
            "resolve_url")
        lines = [script, ""]
        for reference in references:
            lines.append("resolve_url %s" % _shell_quote(reference))
        completed = self.run_step("\n".join(lines), {
            "SITE_ORIGIN": "https://nawodyaishan.github.io",
            "SITE_BASE": "/agentic-sdd",
        })
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        return completed.stdout.decode("utf-8").splitlines()

    def test_every_reference_form_carries_the_base_path_once(self):
        fixtures = [
            # form, reference, expected resolved URL
            ("absolute", "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("protocol-relative", "//nawodyaishan.github.io/agentic-sdd/assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("root-relative with base", "/agentic-sdd/assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("root-relative without base", "/assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("page-relative", "assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("dot-relative", "./assets-v3/page.js",
             "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js"),
            ("base root", "/agentic-sdd",
             "https://nawodyaishan.github.io/agentic-sdd"),
        ]
        resolved = self.resolve_values([item[1] for item in fixtures])
        self.assertEqual(len(resolved), len(fixtures))
        for (form, reference, expected), observed in zip(fixtures, resolved):
            with self.subTest(form=form):
                self.assertEqual(observed, expected)
                self.assertEqual(observed.count("/agentic-sdd"), 1,
                                 "%s resolved the base path more than once" % reference)

    def test_legacy_concatenation_double_applied_the_base_path(self):
        """The previous form is reproduced, so this fixture is not vacuous."""
        site = "https://nawodyaishan.github.io/agentic-sdd"
        href = "/agentic-sdd/assets-v3/page.js"
        legacy = site + href
        self.assertEqual(legacy.count("/agentic-sdd"), 2)
        resolved = self.resolve_values([href])
        self.assertEqual(resolved[0].count("/agentic-sdd"), 1)
        self.assertNotEqual(resolved[0], legacy)

    def test_manifest_origin_and_base_match_the_workflow(self):
        manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
        env = step_env(step_block(job_blocks(workflow_text())[SMOKE_JOB], SMOKE_STEP))
        self.assertEqual(env.get("SITE_ORIGIN"), manifest["site"],
                         "the observed origin drifted from website/docs-manifest.json")
        self.assertEqual(env.get("SITE_BASE"), manifest["base"],
                         "the observed base path drifted from website/docs-manifest.json")

    def test_retry_policy_is_unchanged(self):
        env = step_env(step_block(job_blocks(workflow_text())[SMOKE_JOB], SMOKE_STEP))
        self.assertEqual(env.get("MAX_ATTEMPTS"), "40")
        self.assertEqual(env.get("RETRY_DELAY"), "15")
        script = job_run_text(job_blocks(workflow_text())[SMOKE_JOB], SMOKE_STEP)
        self.assertIn("Cache-Control: no-cache", script)
        self.assertIn("Pragma: no-cache", script)


def _shell_quote(value):
    return "'" + value.replace("'", "'\\''") + "'"


# ---------------------------------------------------------------------------
# AC-08: delayed and failed observations stay recorded and non-blocking
# ---------------------------------------------------------------------------


PAGE_URL = "https://nawodyaishan.github.io/agentic-sdd/"


def page_html(css="/agentic-sdd/assets-v3/page.css",
              script="/agentic-sdd/assets-v3/page.js"):
    links = ""
    if css is not None:
        links += '<link rel="stylesheet" href="%s" />' % css
    if script is not None:
        links += '<script type="module" src="%s"></script>' % script
    return "<!doctype html><html><head>%s</head><body>site</body></html>" % links


class DeploymentObservationTests(OfflineStepCase):
    """The post-deploy observation records every outcome and blocks nothing."""

    def run_smoke(self, responses, attempts="3", routes="/llms.txt"):
        (self.state / "responses.json").write_text(json.dumps(responses),
                                                   encoding="utf-8")
        script = job_run_text(job_blocks(workflow_text())[SMOKE_JOB], SMOKE_STEP)
        return self.run_step(script, {
            "SITE_ORIGIN": "https://nawodyaishan.github.io",
            "SITE_BASE": "/agentic-sdd",
            "OUT_DIR": str(self.out),
            "MAX_ATTEMPTS": attempts,
            "RETRY_DELAY": "0",
            "REQUIRED_ROUTES": routes,
        })

    def test_delayed_deployment_is_recorded_as_eventually_servable(self):
        responses = {
            PAGE_URL: [[404, ""], [404, ""], [200, page_html()]],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.css": [200, ""],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js": [200, ""],
            "https://nawodyaishan.github.io/agentic-sdd/llms.txt": [200, ""],
        }
        completed = self.run_smoke(responses)
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        record = self.record("observation.json")
        self.assertEqual(record["schema"], "docs-deployment-observation/1")
        self.assertEqual(record["outcome"], "observed-servable")
        self.assertIs(record["servable"], True)
        self.assertEqual(record["attempts"], 3)
        self.assertEqual(record["max_attempts"], 3)
        self.assertIs(record["authoritative"], False)
        self.assertIs(record["blocking"], False)
        self.assertIs(record["blocking_release"], False)
        statuses = {item["reference"]: item["status"] for item in record["urls"]}
        self.assertEqual(statuses["/agentic-sdd/assets-v3/page.css"], 200)
        self.assertEqual(statuses["/agentic-sdd/assets-v3/page.js"], 200)
        self.assertEqual(statuses["/llms.txt"], 200)
        for item in record["urls"]:
            self.assertEqual(item["url"].count("/agentic-sdd"), 1, item["url"])

    def test_never_servable_deployment_is_recorded_and_fails_visibly(self):
        responses = {PAGE_URL: [404, ""]}
        completed = self.run_smoke(responses, attempts="4")
        self.assertNotEqual(completed.returncode, 0,
                            "an unservable deployment must be visible as a failure")
        record = self.record("observation.json")
        self.assertEqual(record["outcome"], "observed-unservable")
        self.assertIs(record["servable"], False)
        self.assertEqual(record["attempts"], 4)
        self.assertIs(record["blocking"], False)
        self.assertIs(record["authoritative"], False)
        sleeps = (self.state / "sleeps").read_text(encoding="utf-8").split()
        self.assertEqual(len(sleeps), 3, "the retry policy was applied 4 times, not once")

    def test_page_without_assets_is_observed_as_incomplete(self):
        responses = {PAGE_URL: [200, page_html(css=None, script=None)]}
        completed = self.run_smoke(responses)
        self.assertNotEqual(completed.returncode, 0)
        record = self.record("observation.json")
        self.assertEqual(record["outcome"], "observed-incomplete")
        self.assertIs(record["servable"], False)
        self.assertEqual(record["urls"], [])

    def test_missing_asset_is_recorded_per_url_and_not_a_success(self):
        responses = {
            PAGE_URL: [200, page_html()],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.css": [404, ""],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js": [200, ""],
            "https://nawodyaishan.github.io/agentic-sdd/llms.txt": [200, ""],
        }
        completed = self.run_smoke(responses)
        self.assertNotEqual(completed.returncode, 0)
        record = self.record("observation.json")
        self.assertEqual(record["outcome"], "observed-unservable")
        self.assertIs(record["servable"], False)
        statuses = {item["reference"]: item["status"] for item in record["urls"]}
        self.assertEqual(statuses["/agentic-sdd/assets-v3/page.css"], 404)
        self.assertEqual(statuses["/agentic-sdd/assets-v3/page.js"], 200)

    def test_relative_and_root_relative_references_are_all_followed(self):
        responses = {
            PAGE_URL: [200, page_html(css="./assets-v3/page.css",
                                      script="/assets-v3/page.js")],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.css": [200, ""],
            "https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js": [200, ""],
            "https://nawodyaishan.github.io/agentic-sdd/llms.txt": [200, ""],
        }
        completed = self.run_smoke(responses)
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        record = self.record("observation.json")
        self.assertEqual(record["outcome"], "observed-servable")
        fetched = self.calls()
        self.assertIn("https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.css",
                      fetched)
        self.assertIn("https://nawodyaishan.github.io/agentic-sdd/assets-v3/page.js",
                      fetched)
        for url in fetched:
            self.assertNotIn("/agentic-sdd/agentic-sdd", url)

    def test_smoke_job_stays_a_non_blocking_observation(self):
        jobs = job_blocks(workflow_text())
        job = jobs[SMOKE_JOB]
        joined = "\n".join(job)
        self.assertEqual(job_scalar(job, "continue-on-error"), "true")
        self.assertEqual(job_scalar(job, "needs"), "[deploy]")
        self.assertEqual((job_scalar(job, "if") or "").replace(" ", ""),
                         "needs.deploy.result=='success'")
        self.assertNotIn("id-token", joined)
        self.assertNotIn("pages: write", joined)
        self.assertNotIn("environment:", joined)
        self.assertNotIn("deploy-pages", joined)
        upload = step_block(job, SMOKE_UPLOAD_STEP)
        self.assertIsNotNone(upload, "a failed observation would leave no evidence")
        self.assertIn("if: always()", "\n".join(upload))
        self.assertIn("docs-deployment-observation", "\n".join(upload))


# ---------------------------------------------------------------------------
# AC-08: the dedicated observer for a completed publish run
# ---------------------------------------------------------------------------


API_ROOT = "repos/nawodyaishan/agentic-sdd"
TAG_OBJECT = "b" * 40


def release_documents(tag="v0.13.0", commit=REVISION, status="ahead",
                      object_type="commit", object_sha=None):
    """The read-only API documents the resolve step reads for one release."""
    sha = object_sha if object_sha is not None else commit
    documents = {
        API_ROOT + "/releases/latest": {"tag_name": tag},
        API_ROOT + "/git/ref/tags/" + tag: {
            "object": {"type": object_type, "sha": sha}},
        API_ROOT + "/compare/%s...main" % commit: {"status": status},
    }
    if object_type == "tag":
        documents[API_ROOT + "/git/tags/" + sha] = {
            "object": {"type": "commit", "sha": commit}}
    return documents


class WorkflowRunObserverTests(OfflineStepCase):
    """docs-observe.yml: read-only, non-blocking observation of a publish run."""

    def setUp(self):
        super().setUp()
        gh = self.bin / "gh"
        gh.write_text(FAKE_GH, encoding="utf-8")
        gh.chmod(0o755)
        (self.state / "gh-responses.json").write_text("{}", encoding="utf-8")
        self.github_env = self.root / "github-env"
        self.github_env.write_text("", encoding="utf-8")

    def job(self):
        jobs = job_blocks(observe_text())
        self.assertIn(OBSERVE_JOB, jobs, "docs-observe.yml declares no observer job")
        return jobs[OBSERVE_JOB]

    def resolve(self, documents, branch="main", **overrides):
        (self.state / "gh-responses.json").write_text(
            json.dumps(documents), encoding="utf-8")
        env = {
            "GH_TOKEN": "ghs_fixture_read_only",
            "GH_REPO": "nawodyaishan/agentic-sdd",
            "DEFAULT_BRANCH": branch,
            "GITHUB_ENV": str(self.github_env),
        }
        env.update(overrides)
        return self.run_step(job_run_text(self.job(), RESOLVE_STEP), env)

    def resolved(self):
        values = {}
        for line in self.github_env.read_text(encoding="utf-8").splitlines():
            if "=" in line:
                key, value = line.split("=", 1)
                values[key] = value
        return values

    def api_calls(self):
        path = self.state / "gh-calls"
        return path.read_text(encoding="utf-8").splitlines() if path.is_file() else []

    # ------------------------------------------------------------ structure

    def test_trigger_is_a_completed_publish_run_only(self):
        text = observe_text()
        self.assertEqual(trigger_names(text), ["workflow_run"],
                         "the observer must be selected by a workflow_run only")
        block = trigger_block(text, "workflow_run")
        self.assertIsNotNone(block, "no workflow_run trigger is declared")
        stripped = [line.strip() for line in block]
        self.assertIn('workflows: ["Release publish"]', stripped)
        self.assertIn("types: [completed]", stripped)
        # The observed workflow name must be the publisher that actually exists.
        publish_name = re.search(
            r"(?m)^name:\s*(.+?)\s*$",
            PUBLISH_WORKFLOW.read_text(encoding="utf-8")).group(1)
        self.assertEqual(publish_name, "Release publish")
        self.assertIn('workflows: ["%s"]' % publish_name, stripped)

    def test_only_a_successful_publish_is_observed(self):
        condition = (job_scalar(self.job(), "if") or "").replace(" ", "")
        self.assertEqual(condition,
                         "github.event.workflow_run.conclusion=='success'",
                         "a failed or cancelled publish must not be observed")

    def test_workflow_and_job_hold_read_scope_only(self):
        text = observe_text()
        self.assertRegex(text, r"(?m)^permissions:\s*\{\s*\}\s*$",
                         "workflow-level permissions must be empty")
        job = self.job()
        self.assertEqual(job_permissions(job), {"contents": "read"},
                         "the observer must hold contents: read and nothing else")
        joined = "\n".join(job)
        for forbidden in ("secrets.", "environment:", "id-token", "pages:",
                          "deploy-pages", "upload-pages-artifact",
                          "persist-credentials: true", "workflow_call", "needs:"):
            self.assertNotIn(forbidden, joined,
                             "the observer must not contain %r" % forbidden)
        timeout = job_scalar(job, "timeout-minutes")
        self.assertIsNotNone(timeout, "the observer job has no timeout")
        self.assertLessEqual(int(timeout), 10)

    def test_the_default_branch_head_is_never_used_as_the_observed_revision(self):
        """The publish run's head_sha is the default branch, not the release."""
        text = observe_text()
        self.assertNotIn("workflow_run.head_sha", text)
        self.assertNotIn("github.sha", text)
        checkout = step_block(self.job(), "Checkout the published revision")
        self.assertIsNotNone(checkout, "the observer never checks out the revision")
        joined = "\n".join(checkout)
        self.assertIn("ref: ${{ env.RESOLVED_REVISION }}", joined)
        self.assertIn("persist-credentials: false", joined)

    def test_every_action_uses_a_pin_an_existing_workflow_already_uses(self):
        pins = set()
        for path in (WORKFLOW, PUBLISH_WORKFLOW):
            pins.update(re.findall(r"(?m)^\s+uses:\s*\S+@(\S+)",
                                   path.read_text(encoding="utf-8")))
        uses = re.findall(r"(?m)^\s+uses:\s*(\S+)@(\S+)", observe_text())
        self.assertTrue(uses, "the observer declares no action at all")
        for action, ref in uses:
            with self.subTest(action=action):
                self.assertRegex(ref, r"^[0-9a-f]{40}$")
                self.assertIn(ref, pins,
                              "unpinned or foreign action %s@%s" % (action, ref))

    def test_every_run_body_parses_and_interpolates_no_expression(self):
        bodies = run_bodies(observe_text())
        self.assertTrue(bodies, "docs-observe.yml declares no run body")
        script = self.root / "parse.sh"
        for body in bodies:
            first = body.splitlines()[0] if body.strip() else "<empty>"
            with self.subTest(body=first):
                self.assertNotIn("${{", body,
                                 "an expression inside a run body is a script injection surface")
                script.write_text(body, encoding="utf-8")
                completed = subprocess.run(["bash", "-n", str(script)],
                                           stdout=subprocess.PIPE,
                                           stderr=subprocess.PIPE)
                self.assertEqual(completed.returncode, 0,
                                 completed.stderr.decode("utf-8", "replace"))

    # ------------------------------------------------------ revision resolve

    def test_strict_tag_with_a_commit_on_the_default_branch_is_resolved(self):
        completed = self.resolve(release_documents())
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        self.assertEqual(self.resolved(), {"RESOLVED_TAG": "v0.13.0",
                                           "RESOLVED_REVISION": REVISION})
        calls = self.api_calls()
        self.assertIn(API_ROOT + "/releases/latest", calls)
        self.assertIn(API_ROOT + "/compare/%s...main" % REVISION, calls)

    def test_an_identical_commit_is_still_on_the_default_branch(self):
        completed = self.resolve(release_documents(status="identical"))
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        self.assertEqual(self.resolved().get("RESOLVED_REVISION"), REVISION)

    def test_annotated_tag_is_dereferenced_once_to_its_commit(self):
        completed = self.resolve(release_documents(object_type="tag",
                                                   object_sha=TAG_OBJECT))
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        self.assertEqual(self.resolved().get("RESOLVED_REVISION"), REVISION)
        self.assertIn(API_ROOT + "/git/tags/" + TAG_OBJECT, self.api_calls())
        # The tag object itself is never used as the observed revision.
        self.assertNotEqual(self.resolved().get("RESOLVED_REVISION"), TAG_OBJECT)

    def test_anything_that_is_not_a_strict_release_tag_is_refused(self):
        for tag in ("v0.13", "0.13.0", "v0.13.0-rc1", "v01.2.3", "main", "",
                    "v0.13.0 ", "v0.13.0\nv0.14.0", "v0.13.0.1",
                    "v0.13.0; rm -rf /", "release/v0.13.0"):
            with self.subTest(tag=tag):
                completed = self.resolve(release_documents(tag=tag))
                self.assertNotEqual(completed.returncode, 0,
                                    "tag %r was accepted" % tag)
                self.assertEqual(self.resolved(), {},
                                 "a refused tag must not export a revision")

    def test_a_tag_that_does_not_resolve_to_a_full_commit_is_refused(self):
        for sha in ("A" * 40, "a" * 39, "a" * 41, "g" * 40, "main", "",
                    REVISION + "\nX", " " + REVISION):
            with self.subTest(sha=sha):
                completed = self.resolve(release_documents(commit=sha,
                                                           object_sha=sha))
                self.assertNotEqual(completed.returncode, 0)
                self.assertEqual(self.resolved(), {})

    def test_a_tag_that_points_at_something_other_than_a_commit_is_refused(self):
        for object_type in ("tree", "blob"):
            with self.subTest(object_type=object_type):
                completed = self.resolve(release_documents(object_type=object_type))
                self.assertNotEqual(completed.returncode, 0)
                self.assertEqual(self.resolved(), {})

        documents = release_documents(object_type="tag", object_sha=TAG_OBJECT)
        documents[API_ROOT + "/git/tags/" + TAG_OBJECT] = {
            "object": {"type": "tree", "sha": REVISION}}
        completed = self.resolve(documents)
        self.assertNotEqual(completed.returncode, 0,
                            "an annotated tag of a tree must not be observed")
        self.assertEqual(self.resolved(), {})

    def test_a_released_commit_outside_the_default_branch_is_refused(self):
        for status in ("behind", "diverged", "unknown", ""):
            with self.subTest(status=status):
                completed = self.resolve(release_documents(status=status))
                self.assertNotEqual(completed.returncode, 0,
                                    "an ancestry answer of %r was accepted" % status)
                self.assertEqual(self.resolved(), {})

        # An unavailable compare answer refuses rather than passing by default.
        documents = release_documents()
        documents.pop(API_ROOT + "/compare/%s...main" % REVISION)
        completed = self.resolve(documents)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(self.resolved(), {})

    def test_an_unavailable_release_api_refuses(self):
        completed = self.resolve({})
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(self.resolved(), {})

    def test_a_missing_or_unsafe_default_branch_refuses_before_any_api_call(self):
        for branch in ("", "-x", "a..b", "ma in", "main;rm", " main"):
            with self.subTest(branch=branch):
                completed = self.resolve(release_documents(), branch=branch)
                self.assertNotEqual(completed.returncode, 0)
                self.assertEqual(self.resolved(), {})
                self.assertEqual(self.api_calls(), [],
                                 "no API call may follow an unsafe branch name")

    # ------------------------------------------------------------- evidence

    def binding_env(self, git_head, git_rc="0"):
        return {
            "TRUSTED_DIR": str(self.root / "trusted"),
            "REVISION": REVISION,
            "OUT_DIR": str(self.out),
            "HOME": str(self.home),
            "FAKE_GIT_HEAD": git_head,
            "FAKE_GIT_RC": git_rc,
        }

    def test_the_observed_checkout_is_bound_to_the_resolved_revision(self):
        completed = self.run_step(job_run_text(self.job(), BINDING_STEP),
                                  self.binding_env(REVISION))
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        record = self.record("binding.json")
        self.assertEqual(record["schema"], "docs-observation-binding/1")
        self.assertEqual(record["expected_revision"], REVISION)
        self.assertEqual(record["observed_revision"], REVISION)
        self.assertIs(record["matched"], True)

    def test_a_mismatched_checkout_fails_before_an_observation_is_recorded(self):
        completed = self.run_step(job_run_text(self.job(), BINDING_STEP),
                                  self.binding_env(OTHER_REVISION))
        self.assertNotEqual(completed.returncode, 0)
        record = self.record("binding.json")
        self.assertIs(record["matched"], False)
        self.assertEqual(record["observed_revision"], OTHER_REVISION)
        self.assertFalse((self.out / "observation.json").exists(),
                         "no observation may be recorded for another source")

    def route_env(self, **overrides):
        env = {
            "OUT_DIR": str(self.out),
            "OBSERVATION_ROUTE": OBSERVE_ROUTE,
            "OBSERVATION_DELIVERY": OBSERVE_DELIVERY,
            "OBSERVATION_RUN_ID": OBSERVATION_RUN,
            "REVISION": REVISION,
            "PUBLISHED_TAG": "v0.13.0",
            "SOURCE_RUN_ID": SOURCE_RUN,
        }
        env.update(overrides)
        return env

    def test_route_record_matches_the_callable_observation_shape(self):
        completed = self.run_step(job_run_text(self.job(), ROUTE_STEP),
                                  self.route_env())
        self.assertEqual(completed.returncode, 0,
                         completed.stderr.decode("utf-8", "replace"))
        record = self.record("observation.json")
        self.assertEqual(record["schema"], "docs-observation-route/1")
        self.assertEqual(record["route"], OBSERVE_ROUTE)
        self.assertEqual(record["delivery"], OBSERVE_DELIVERY)
        self.assertEqual(record["event"], "workflow_run")
        self.assertEqual(record["revision"], REVISION)
        self.assertEqual(record["tag"], "v0.13.0")
        self.assertEqual(record["source_run_id"], SOURCE_RUN)
        self.assertEqual(record["observation_run_id"], OBSERVATION_RUN)
        self.assertEqual(record["result"], "observed")
        self.assertIs(record["authoritative"], False)
        self.assertIs(record["blocking_release"], False)
        self.assertEqual(record["artifact_identity"], "not-inspected")
        self.assertEqual(record["source_binding"], "binding.json")

    def test_route_record_refuses_a_mismatched_or_non_strict_identifier(self):
        fixtures = (
            {"REVISION": "main"},
            {"REVISION": REVISION[:-1]},
            {"REVISION": REVISION + "\nX"},
            {"SOURCE_RUN_ID": "abc"},
            {"SOURCE_RUN_ID": ""},
            {"SOURCE_RUN_ID": SOURCE_RUN + "\n9"},
            {"PUBLISHED_TAG": 'x"y'},
            {"PUBLISHED_TAG": "v0.13"},
            {"PUBLISHED_TAG": ""},
            {"PUBLISHED_TAG": "v0.13.0\nv0.14.0"},
            {"OBSERVATION_RUN_ID": "x1"},
        )
        for fixture in fixtures:
            with self.subTest(fixture=fixture):
                completed = self.run_step(job_run_text(self.job(), ROUTE_STEP),
                                          self.route_env(**fixture))
                self.assertNotEqual(completed.returncode, 0)
                self.assertFalse((self.out / "observation.json").exists(),
                                 "a refused route must not write an observation record")

    def test_evidence_upload_is_unconditional(self):
        upload = step_block(self.job(), OBSERVE_UPLOAD_STEP)
        self.assertIsNotNone(upload, "a failed observation would leave no evidence")
        text = "\n".join(upload)
        self.assertIn("if: always()", text)
        self.assertIn("if-no-files-found: warn", text)
        self.assertIn("docs-post-publish-observation", text)

    def test_the_publisher_is_neither_widened_nor_given_a_callsite(self):
        text = PUBLISH_WORKFLOW.read_text(encoding="utf-8")
        for marker in ("docs-observe", "docs.yml", "workflow_call",
                       "workflows/docs.yml"):
            with self.subTest(marker=marker):
                self.assertNotIn(marker, text)


if __name__ == "__main__":
    unittest.main()
