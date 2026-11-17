"""B7/T14 offline regressions for FR-06/AC-06: release workflow privilege boundary.

Owner decision 2026-11-17 ("Tag trust", plan 12.9 item 10) splits the release in two:
  - release.yml ("Release build"): tag-triggered, no environment, no secrets, no
    token. It builds, verifies and uploads the artifact set as a run artifact.
  - release-publish.yml ("Release publish"): runs only from `workflow_run` of a
    successful tag build. GitHub takes that workflow file from the default branch,
    so a tag cannot change the code that holds the tap token.

Stdlib-only text checks, matching the repository harness style. Step scripts are
also executed as subprocesses with stub `gh`, `git` and `sleep` binaries on PATH,
so the tag check, the ancestry check and the CI gate run without contacting GitHub.
No workflow is run and nothing is published.
"""

from pathlib import Path
import os
import re
import shutil
import subprocess
import tempfile
import json
import unittest


ROOT = Path(__file__).resolve().parents[2]
BUILD_WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"
PUBLISH_WORKFLOW = ROOT / ".github" / "workflows" / "release-publish.yml"
GORELEASER = ROOT / ".goreleaser.yml"
MANIFEST_SCRIPT = ROOT / "scripts" / "release_manifest.py"

FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
CI_WAIT_SECONDS = 40 * 60
COMMIT = "1234567890abcdef1234567890abcdef12345678"
OTHER_COMMIT = "fedcba0987654321fedcba0987654321fedcba09"
TAG = "v0.13.0"
BUILD_NAME = "Release build"
PUBLISHER_RUN = "222"
PUBLISHER_SUITE = 901
CI_SUITE = 900

# Stub gh: serves the next scripted poll (poll-N.jsonl), repeating the last one,
# and counts calls. A gh-fail marker makes it exit non-zero like a failed API call.
FAKE_GH = """#!/bin/sh
state="$FAKE_STATE"
case "$*" in
  *actions/runs/*)
    # Run lookup (check_suite_id): not a poll, so it is counted separately.
    n=$(cat "$state/suite-calls" 2>/dev/null || echo 0)
    echo $((n + 1)) > "$state/suite-calls"
    if [ -f "$state/suite-fail" ]; then
      echo "fake suite failure" >&2
      exit 1
    fi
    cat "$state/suite-id"
    exit 0
    ;;
esac
n=$(cat "$state/gh-calls" 2>/dev/null || echo 0)
n=$((n + 1))
echo "$n" > "$state/gh-calls"
if [ -f "$state/gh-fail" ]; then
  echo "fake gh failure" >&2
  exit 1
fi
f="$state/poll-$n.jsonl"
[ -f "$f" ] || f="$state/poll-last.jsonl"
cat "$f"
"""

# Stub sleep: records the requested interval instead of waiting.
FAKE_SLEEP = """#!/bin/sh
echo "$1" >> "$FAKE_STATE/sleeps"
exit 0
"""

# Stub git: records each call; fetch, rev-parse and merge-base exit or print as scripted.
FAKE_GIT = """#!/bin/sh
echo "$*" >> "$FAKE_STATE/git-calls"
case "$1" in
  fetch) exit "${FAKE_FETCH_RC:-0}" ;;
  rev-parse) echo "${FAKE_TAG_COMMIT:-}"; exit "${FAKE_REVPARSE_RC:-0}" ;;
  merge-base) exit "${FAKE_ANCESTOR_RC:-0}" ;;
esac
echo "unexpected git call: $*" >&2
exit 99
"""


def _top_block(text, key):
    match = re.search(rf"(?ms)^{re.escape(key)}:(.*?)(?=^\S|\Z)", text)
    if not match:
        raise AssertionError(f"missing top-level block {key}")
    return match.group(1)


def _on_keys(text):
    return re.findall(r"(?m)^  ([A-Za-z_]+):", _top_block(text, "on"))


def _workflow_name(text):
    match = re.search(r"(?m)^name:\s*(.+?)\s*$", text)
    return match.group(1) if match else None


def _jobs(text):
    block = _top_block(text, "jobs")
    matches = list(re.finditer(r"(?m)^  ([A-Za-z0-9_-]+):\s*$", block))
    jobs = {}
    for index, match in enumerate(matches):
        end = matches[index + 1].start() if index + 1 < len(matches) else len(block)
        jobs[match.group(1)] = block[match.end():end]
    return jobs


def _split_job(job_text):
    """Return (job-level header text, list of step bodies)."""
    parts = re.split(r"(?m)^      - ", job_text)
    return parts[0], parts[1:]


def _step_name(step_body):
    match = re.match(r"name:\s*(.+)", step_body)
    return match.group(1).strip() if match else ""


def _perm_map(text):
    lines = text.splitlines()
    for index, line in enumerate(lines):
        match = re.match(r"^(\s*)permissions:\s*(\{\s*\})?\s*$", line)
        if not match:
            continue
        if match.group(2):
            return {}
        indent = len(match.group(1))
        result = {}
        for nxt in lines[index + 1:]:
            if not nxt.strip():
                continue
            if len(nxt) - len(nxt.lstrip()) <= indent:
                break
            key, value = nxt.strip().split(":", 1)
            result[key.strip()] = value.strip()
        return result
    raise AssertionError("no permissions block")


def _steps(job_text):
    return _split_job(job_text)[1]


def _find_step(job_text, predicate):
    for index, body in enumerate(_steps(job_text)):
        if predicate(body):
            return index, body
    return None, None


def _is_tooling_checkout(body):
    return "uses: actions/checkout@" in body and "ref: ${{ vars.RELEASE_TOOLS_SHA }}" in body


def _is_pin_check(body):
    return "RELEASE_TOOLS_SHA: ${{ vars.RELEASE_TOOLS_SHA }}" in body and "uses:" not in body


def _is_ci_gate(body):
    return "check-runs" in body


def _is_ancestry_check(body):
    return "merge-base --is-ancestor" in body


def _is_token_bearing(body):
    return any(marker in body for marker in
               ("github.token", "github-token", "secrets.", "GH_TOKEN", "gh release "))


def _run_block(body):
    """Return the dedented script of a step's `run: |` block."""
    lines = body.splitlines()
    start = next(i for i, line in enumerate(lines) if line == "        run: |")
    out = []
    for line in lines[start + 1:]:
        if not line.strip():
            out.append("")
            continue
        if not line.startswith("          "):
            break
        out.append(line[10:])
    return "\n".join(out) + "\n"


def _job_head(job_text):
    return _split_job(job_text)[0]


def _read(path):
    # A missing file reads as empty so each structural assertion reports on its own.
    return path.read_text() if path.exists() else ""


BUILD_TEXT = _read(BUILD_WORKFLOW)
PUBLISH_TEXT = _read(PUBLISH_WORKFLOW)


class BuildWorkflowTests(unittest.TestCase):
    """release.yml: the tag-triggered build. It holds no environment, secret or token."""

    @classmethod
    def setUpClass(cls):
        cls.text = BUILD_TEXT
        cls.jobs = _jobs(cls.text)

    def test_name_and_trigger_are_the_tag_build_only(self):
        self.assertEqual(_workflow_name(self.text), BUILD_NAME)
        self.assertEqual(_on_keys(self.text), ["push"])
        self.assertIn('- "v*"', _top_block(self.text, "on"))

    def test_workflow_level_permissions_are_empty(self):
        self.assertEqual(_perm_map(self.text), {})

    def test_exactly_one_build_job(self):
        self.assertEqual(set(self.jobs), {"verify"})

    def test_build_job_is_read_only_and_has_no_environment(self):
        head = _job_head(self.jobs["verify"])
        self.assertEqual(_perm_map(head), {"contents": "read"})
        self.assertNotIn("environment:", self.text)
        self.assertRegex(head, r"(?m)^    timeout-minutes:\s*\d+\s*$")

    def test_build_file_holds_no_secret_token_or_publishing_step(self):
        for marker in ("secrets.", "github.token", "GH_TOKEN", "GITHUB_TOKEN",
                       "HOMEBREW_TAP_TOKEN", "id-token", "gh release", "git push",
                       "workflow_run", "release-tools", "download-artifact"):
            with self.subTest(marker=marker):
                self.assertNotIn(marker, self.text)

    def test_build_runs_no_candidate_tests(self):
        # Plan 12.4: the build job runs no candidate tests.
        self.assertNotRegex(self.text, r"task\s+verify")
        self.assertNotRegex(self.text, r"go\s+test")

    def test_build_skips_publication_and_never_snapshots(self):
        self.assertIn("--skip=publish", self.jobs["verify"])
        self.assertNotIn("--snapshot", self.jobs["verify"])

    def test_artifacts_are_written_and_verified_before_upload(self):
        _, steps = _split_job(self.jobs["verify"])
        write = next(i for i, b in enumerate(steps) if "release_manifest.py write" in b)
        verify = next(i for i, b in enumerate(steps) if "release_manifest.py verify" in b)
        upload = next(i for i, b in enumerate(steps)
                      if "actions/upload-artifact@" in b and "name: release-artifacts" in b)
        # write and verify run in one step, in that order; the upload comes after both.
        self.assertLessEqual(write, verify)
        self.assertLess(verify, upload)
        verify_step = steps[verify]
        self.assertLess(verify_step.index("release_manifest.py write"),
                        verify_step.index("release_manifest.py verify"))
        self.assertIn('--commit "$GITHUB_SHA"', self.jobs["verify"])

    def test_verification_record_is_sanitized_and_uploaded_even_on_failure(self):
        _, steps = _split_job(self.jobs["verify"])
        record = [b for b in steps if "release_manifest.py record" in b]
        self.assertEqual(len(record), 1)
        self.assertIn("if: always()", record[0])
        self.assertIn("name: release-verification", self.text)

    def test_every_checkout_does_not_persist_credentials(self):
        checkouts = 0
        _, steps = _split_job(self.jobs["verify"])
        for body in steps:
            if "uses: actions/checkout@" in body:
                checkouts += 1
                with self.subTest(step=_step_name(body)):
                    self.assertRegex(body, r"(?m)^\s+persist-credentials:\s*false\s*$")
        self.assertGreaterEqual(checkouts, 1)

    def test_every_action_is_pinned_to_a_full_commit_sha(self):
        uses = re.findall(r"(?m)^\s+uses:\s*(\S+)@(\S+)", self.text)
        self.assertTrue(uses)
        for action, ref in uses:
            with self.subTest(action=action):
                self.assertRegex(ref, FULL_SHA)
                self.assertNotRegex(ref, r"^v\d")

    def test_artifact_upload_is_pinned_and_bounded(self):
        self.assertIn("uses: actions/upload-artifact@", self.text)
        self.assertRegex(self.text, r"(?m)^\s+retention-days:\s*\d+\s*$")
        self.assertNotIn("--clobber", self.text)

    def test_goreleaser_version_is_still_pinned(self):
        self.assertIn('version: "v2.14.0"', self.text)

    def test_every_job_has_a_timeout(self):
        for name, job in self.jobs.items():
            with self.subTest(job=name):
                self.assertRegex(job, r"(?m)^    timeout-minutes:\s*\d+\s*$")


class PublishWorkflowTests(unittest.TestCase):
    """release-publish.yml: the only file that holds the tap token and write scope."""

    @classmethod
    def setUpClass(cls):
        cls.text = PUBLISH_TEXT
        cls.jobs = _jobs(cls.text)
        cls.head = _job_head(cls.jobs["publish"])
        cls.steps = _steps(cls.jobs["publish"])

    def test_name_and_trigger_is_workflow_run_of_the_build_only(self):
        self.assertEqual(_workflow_name(self.text), "Release publish")
        self.assertEqual(_on_keys(self.text), ["workflow_run"])
        block = _top_block(self.text, "on")
        self.assertRegex(block, r'(?m)^    workflows:\s*\["Release build"\]\s*$')
        self.assertRegex(block, r"(?m)^    types:\s*\[completed\]\s*$")

    def test_no_tag_push_or_other_trigger_can_reach_the_publisher(self):
        for marker in ("push:", "pull_request", "workflow_dispatch", "schedule:",
                       "repository_dispatch"):
            with self.subTest(marker=marker):
                self.assertNotIn(marker, self.text)
        # `fetch-tags:` is an option, not a trigger; only a bare `tags:` key would be one.
        self.assertIsNone(re.search(r"(?<![\w-])tags:", self.text))

    def test_build_name_in_trigger_matches_the_build_workflow_name(self):
        self.assertEqual(_workflow_name(BUILD_TEXT), BUILD_NAME)
        self.assertIn(f'workflows: ["{BUILD_NAME}"]', self.text)

    def test_publish_runs_only_for_a_successful_tag_push(self):
        self.assertRegex(self.head, r"(?m)^    if:")
        self.assertIn("github.event.workflow_run.conclusion == 'success'", self.head)
        self.assertIn("github.event.workflow_run.event == 'push'", self.head)
        self.assertIn("startsWith(github.event.workflow_run.head_branch, 'v')", self.head)

    def test_workflow_level_permissions_are_empty(self):
        self.assertEqual(_perm_map(self.text), {})

    def test_exactly_one_publish_job_and_no_cross_job_needs(self):
        self.assertEqual(set(self.jobs), {"publish"})
        self.assertNotIn("needs:", self.text)

    def test_publish_job_permissions_are_exactly_contents_checks_and_actions_read(self):
        self.assertEqual(_perm_map(self.head),
                         {"contents": "write", "checks": "read", "actions": "read"})
        self.assertNotIn("checks: write", self.text)
        self.assertNotIn("actions: write", self.text)
        self.assertNotIn("id-token", self.text)

    def test_publish_job_is_gated_serialized_and_bounded(self):
        self.assertRegex(self.head, r"(?m)^    environment:\s*release\s*$")
        self.assertRegex(self.head, r"(?m)^    timeout-minutes:\s*\d+\s*$")
        self.assertRegex(
            self.head,
            r"(?ms)^    concurrency:\s*\n\s+group:\s*\S+\s*\n\s+cancel-in-progress:\s*false\s*$",
        )
        group = re.search(r"(?m)^\s+group:\s*(\S+)\s*$", self.head).group(1)
        self.assertNotIn("${{", group, "concurrency group must be a fixed string")

    def test_tag_and_commit_come_only_from_the_build_run_payload(self):
        self.assertIn("TAG_NAME: ${{ github.event.workflow_run.head_branch }}", self.head)
        self.assertIn("TAG_COMMIT: ${{ github.event.workflow_run.head_sha }}", self.head)
        # The workflow_run context carries the default branch in GITHUB_SHA and
        # GITHUB_REF_NAME, so neither may appear anywhere in this file.
        for name in ("GITHUB_SHA", "GITHUB_REF_NAME", "github.sha", "github.ref_name",
                     "github.ref"):
            with self.subTest(name=name):
                self.assertNotIn(name, self.text)
        self.assertIn("GITHUB_RUN_ID", self.text)

    def test_tag_is_validated_by_the_first_step_before_any_other_use(self):
        self.assertTrue(self.steps)
        self.assertTrue(_step_name(self.steps[0]).startswith("Validate the tag"),
                        _step_name(self.steps[0]))
        script = _run_block(self.steps[0])
        self.assertIn('[[ ! "${TAG_NAME:-}" =~ ^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$ ]]',
                      script)
        self.assertIn('[[ ! "${TAG_COMMIT:-}" =~ ^[0-9a-f]{40}$ ]]', script)

    def test_no_run_block_interpolates_an_expression(self):
        for body in self.steps:
            if "        run: |" in body:
                with self.subTest(step=_step_name(body)):
                    self.assertNotIn("${{", _run_block(body))

    def test_tagged_commit_is_checked_out_explicitly_with_tags(self):
        _, checkout = _find_step(self.jobs["publish"],
                                 lambda body: _step_name(body) == "Checkout the tagged commit")
        self.assertIsNotNone(checkout)
        self.assertIn("uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683", checkout)
        self.assertIn("ref: ${{ github.event.workflow_run.head_sha }}", checkout)
        self.assertRegex(checkout, r"(?m)^\s+fetch-depth:\s*0\s*$")
        self.assertRegex(checkout, r"(?m)^\s+fetch-tags:\s*true\s*$")
        self.assertRegex(checkout, r"(?m)^\s+persist-credentials:\s*false\s*$")

    def test_default_branch_ancestry_step_is_defined_and_uses_the_tag_commit(self):
        _, step = _find_step(self.jobs["publish"], _is_ancestry_check)
        self.assertIsNotNone(step, "publish job has no default-branch ancestry check")
        self.assertIn("DEFAULT_BRANCH: ${{ github.event.repository.default_branch }}", step)
        self.assertIn('git merge-base --is-ancestor "$TAG_COMMIT" "origin/$DEFAULT_BRANCH"', step)
        self.assertIn('rev-parse --verify "refs/tags/${TAG_NAME}^{commit}"', _run_block(step))
        self.assertNotIn("${{", _run_block(step))
        self.assertNotIn("github.token", step)
        self.assertNotIn("secrets.", step)

    def test_pin_check_precedes_tooling_checkout_and_every_token_bearing_step(self):
        pin_idx, pin = _find_step(self.jobs["publish"], _is_pin_check)
        tooling_idx, tooling = _find_step(self.jobs["publish"], _is_tooling_checkout)
        ancestry_idx, _ = _find_step(self.jobs["publish"], _is_ancestry_check)
        self.assertIsNotNone(pin_idx, "publish job has no RELEASE_TOOLS_SHA pin check")
        self.assertIsNotNone(tooling_idx, "publish job has no pinned tooling checkout")
        self.assertLess(pin_idx, tooling_idx)
        self.assertLess(tooling_idx, ancestry_idx)
        self.assertIn("set -euo pipefail", _run_block(pin))
        for position, body in enumerate(self.steps):
            if _is_token_bearing(body):
                with self.subTest(step=_step_name(body)):
                    self.assertGreater(position, tooling_idx)
                    self.assertGreater(position, ancestry_idx)

    def test_publisher_steps_run_in_the_documented_order(self):
        def first(predicate):
            index, _ = _find_step(self.jobs["publish"], predicate)
            self.assertIsNotNone(index)
            return index

        order = [
            first(lambda b: _step_name(b).startswith("Validate the tag")),
            first(lambda b: _step_name(b) == "Checkout the tagged commit"),
            first(_is_pin_check),
            first(_is_tooling_checkout),
            first(_is_ancestry_check),
            first(_is_ci_gate),
            first(lambda b: "download-artifact@" in b),
            first(lambda b: _step_name(b).startswith("Confirm the tagged commit")),
            first(lambda b: "Read Homebrew tap" in _step_name(b)),
            first(lambda b: "gh release create" in b),
            first(lambda b: "Update Homebrew tap" in _step_name(b)),
        ]
        self.assertEqual(order, sorted(order))
        self.assertEqual(len(set(order)), len(order))

    def test_ci_gate_is_read_only_paginated_and_uses_the_job_token(self):
        _, gate = _find_step(self.jobs["publish"], _is_ci_gate)
        self.assertIsNotNone(gate, "publish job has no CI gate step")
        self.assertIn("gh api --paginate", gate)
        self.assertIn("repos/${GH_REPO}/commits/${TAG_COMMIT}/check-runs", gate)
        self.assertIn("GH_TOKEN: ${{ github.token }}", gate)
        self.assertNotIn("secrets.", gate)
        self.assertNotRegex(gate, r"gh api\s+(-X|--method)\s+(POST|PUT|PATCH|DELETE)")
        self.assertIn("check-ci", gate)
        self.assertIn('--suite-id "$suite"', gate)
        self.assertIn('--commit "$TAG_COMMIT"', gate)
        self.assertNotIn("--run-id", gate)

    def test_ci_gate_precedes_every_download_and_publication_step(self):
        gate_idx, _ = _find_step(self.jobs["publish"], _is_ci_gate)
        self.assertIsNotNone(gate_idx)
        for index, body in enumerate(self.steps):
            if index == gate_idx:
                continue
            if ("download-artifact@" in body or "gh release " in body
                    or "HOMEBREW_TAP_TOKEN" in body or re.search(r"git (-C \S+ )?push\b", body)):
                with self.subTest(step=_step_name(body)):
                    self.assertGreater(index, gate_idx)

    def test_artifacts_download_from_the_build_run_with_an_explicit_run_and_token(self):
        _, download = _find_step(self.jobs["publish"], lambda b: "download-artifact@" in b)
        self.assertIsNotNone(download)
        self.assertIn("name: release-artifacts", download)
        self.assertIn("path: release-artifacts", download)
        self.assertIn("run-id: ${{ github.event.workflow_run.id }}", download)
        self.assertIn("github-token: ${{ github.token }}", download)

    def test_suite_id_is_resolved_from_this_run_and_validated_before_any_poll(self):
        _, gate = _find_step(self.jobs["publish"], _is_ci_gate)
        script = _run_block(gate)
        lookup = script.find("repos/${GH_REPO}/actions/runs/${GITHUB_RUN_ID}\" --jq '.check_suite_id'")
        self.assertGreaterEqual(lookup, 0, "suite id lookup missing from the CI gate")
        self.assertLess(lookup, script.index("for ((attempt"))
        self.assertIn('[[ ! "$suite" =~ ^[1-9][0-9]*$ ]]', script)
        self.assertEqual(script.count('--suite-id "$suite"'), 2)
        self.assertNotIn("details_url", gate)

    def test_projection_keeps_check_suite_id_and_drops_the_url(self):
        _, gate = _find_step(self.jobs["publish"], _is_ci_gate)
        self.assertIn("{name, status, conclusion, check_suite: {id: .check_suite.id}} | @json", gate)
        self.assertNotIn("details_url", gate)

    @unittest.skipUnless(shutil.which("jq"), "jq is not installed")
    def test_projection_filter_yields_the_suite_id_for_real_check_run_json(self):
        _, gate = _find_step(self.jobs["publish"], _is_ci_gate)
        match = re.search(r"--jq '(\.check_runs\[\][^']*)'", gate)
        self.assertIsNotNone(match)
        sample = {"check_runs": [
            {"name": "test", "status": "completed", "conclusion": "success",
             "details_url": "https://github.com/o/r/actions/runs/222/job/1",
             "check_suite": {"id": 900, "head_branch": "main"}},
            {"name": "legacy", "status": "queued", "conclusion": None},
        ]}
        result = subprocess.run(["jq", "-r", match.group(1)], input=json.dumps(sample),
                                capture_output=True, text=True, timeout=30)
        self.assertEqual(result.returncode, 0, result.stderr)
        runs = [json.loads(line) for line in result.stdout.splitlines()]
        self.assertEqual(runs[0], {"name": "test", "status": "completed", "conclusion": "success",
                                   "check_suite": {"id": 900}})
        self.assertEqual(runs[1], {"name": "legacy", "status": "queued", "conclusion": None,
                                   "check_suite": {"id": None}})
        self.assertNotIn("details_url", result.stdout)

    def test_ci_wait_is_bounded_polling_of_wait_ci_and_fits_the_job_timeout(self):
        _, gate = _find_step(self.jobs["publish"], _is_ci_gate)
        script = _run_block(gate)
        polls = int(re.search(r"(?m)^polls=(\d+)$", script).group(1))
        interval = int(re.search(r"(?m)^\s+sleep (\d+)$", script).group(1))
        self.assertEqual(polls * interval, CI_WAIT_SECONDS, "wait budget must be 40 minutes")
        self.assertIn("wait-ci", script)
        self.assertIn("check-ci", script)
        timeout = int(re.search(r"(?m)^    timeout-minutes:\s*(\d+)\s*$", self.head).group(1))
        self.assertGreater(timeout, CI_WAIT_SECONDS // 60, "job timeout must outlast the CI wait")

    def test_publish_rechecks_commit_and_artifacts_before_any_write(self):
        names = [_step_name(body) for body in self.steps]
        verify_idx = next(i for i, b in enumerate(self.steps)
                          if "release_manifest.py" in b and " verify " in b)
        upload_idx = next(i for i, b in enumerate(self.steps) if "gh release upload" in b)
        self.assertLess(verify_idx, upload_idx, names)
        self.assertIn('--commit "$TAG_COMMIT"', self.text)
        self.assertIn("rev-parse", self.text)

    def test_downgrade_guard_runs_before_release_publication_and_again_before_push(self):
        guard_steps = [i for i, b in enumerate(self.steps)
                       if "release_cask.py" in b and "check-downgrade" in b]
        upload_idx = next(i for i, b in enumerate(self.steps) if "gh release upload" in b)
        push_idx = next(i for i, b in enumerate(self.steps)
                        if re.search(r"git (-C \S+ )?push\b", b))
        self.assertGreaterEqual(len(guard_steps), 2)
        self.assertLess(guard_steps[0], upload_idx)
        self.assertLess(upload_idx, push_idx)
        self.assertGreater(guard_steps[-1], upload_idx)

    def test_tap_update_step_refuses_before_it_renders_the_cask(self):
        # The final tap step must run the guard (missing cask or leftover formula refuses)
        # before the cask is rendered and committed; the order is part of the contract.
        _, step = _find_step(self.jobs["publish"], lambda b: "Update Homebrew tap" in _step_name(b))
        self.assertIsNotNone(step)
        script = _run_block(step)
        guard = script.index("release_cask.py check-downgrade")
        render = script.index("release_cask.py render")
        commit = script.index("commit --quiet")
        self.assertLess(guard, render)
        self.assertLess(render, commit)

    def test_release_assets_upload_without_clobber(self):
        self.assertNotIn("--clobber", self.text)
        self.assertIn("gh release upload", self.text)

    def test_cask_is_rendered_from_verified_manifest_not_rebuilt(self):
        self.assertIn("release_cask.py render", self.text)
        self.assertIn("release-artifacts/manifest.json", self.text)
        self.assertNotIn("goreleaser", self.text)

    def test_cask_render_is_bound_to_the_tagged_commit(self):
        _, render = _find_step(self.jobs["publish"],
                               lambda b: "release_cask.py" in b and " render " in b)
        self.assertIsNotNone(render, "publish job has no cask render step")
        self.assertIn('--commit "$TAG_COMMIT"', render)

    def test_tap_push_step_has_no_silent_success_branch(self):
        # The guard refuses an equal or newer tap version before the render, so the cask text
        # always changes; an "already matches" exit-0 branch could never be reached.
        _, step = _find_step(self.jobs["publish"], lambda b: "Update Homebrew tap" in _step_name(b))
        self.assertIsNotNone(step)
        script = _run_block(step)
        self.assertNotIn("exit 0", script)
        self.assertNotIn("diff --cached --quiet", script)
        self.assertNotIn("nothing to push", script)
        self.assertRegex(script, r"(?m)^\s*git -C \"\$tap\" .*commit --quiet")
        self.assertRegex(script, r"(?m)^\s*git -C \"\$tap\" push --quiet origin HEAD\s*$")

    def test_release_and_tap_use_the_tag_name_not_the_default_branch(self):
        for body in self.steps:
            if "release_cask.py" in body or "gh release " in body:
                with self.subTest(step=_step_name(body)):
                    self.assertIn("TAG_NAME", body)
        self.assertIn('--tag "$TAG_NAME"', self.text)
        self.assertIn('agentic-sdd $TAG_NAME', self.text)

    # --- Decision 2 (plan 12.8): tap-update scripts run from a pinned tooling commit. ---

    def test_tooling_checkout_is_pinned_to_repository_variable_without_credentials(self):
        _, tooling = _find_step(self.jobs["publish"], _is_tooling_checkout)
        self.assertIsNotNone(tooling)
        self.assertRegex(tooling, r"uses: actions/checkout@[0-9a-f]{40}")
        self.assertRegex(tooling, r"(?m)^\s+path:\s*release-tools\s*$")
        self.assertRegex(tooling, r"(?m)^\s+persist-credentials:\s*false\s*$")
        self.assertNotIn("github.token", tooling)
        self.assertNotIn("secrets.", tooling)

    def test_tap_token_steps_run_only_after_the_tooling_checkout_and_ancestry(self):
        tooling_idx, _ = _find_step(self.jobs["publish"], _is_tooling_checkout)
        ancestry_idx, _ = _find_step(self.jobs["publish"], _is_ancestry_check)
        tap_steps = [index for index, body in enumerate(self.steps)
                     if "HOMEBREW_TAP_TOKEN" in body]
        self.assertGreaterEqual(len(tap_steps), 2)
        for index in tap_steps:
            with self.subTest(step=index):
                self.assertGreater(index, tooling_idx)
                self.assertGreater(index, ancestry_idx)

    def test_tap_token_is_only_in_tap_steps(self):
        for body in self.steps:
            if "HOMEBREW_TAP_TOKEN" in body:
                with self.subTest(step=_step_name(body)):
                    self.assertIn("Homebrew tap", _step_name(body))

    def test_secrets_are_referenced_only_by_the_publisher_file(self):
        self.assertEqual(self.text.count("secrets.HOMEBREW_TAP_TOKEN"),
                         sum(body.count("secrets.HOMEBREW_TAP_TOKEN") for body in self.steps))
        self.assertNotIn("secrets.", BUILD_TEXT)

    def test_scripts_run_only_from_the_pinned_tooling_checkout(self):
        tooling_idx, _ = _find_step(self.jobs["publish"], _is_tooling_checkout)
        invocations = re.findall(r"(\S*)scripts/release_(?:manifest|cask)\.py", self.text)
        self.assertTrue(invocations)
        for prefix in invocations:
            self.assertEqual(prefix, "release-tools/", f"unpinned script path prefix {prefix!r}")
        for index, body in enumerate(self.steps):
            if re.search(r"scripts/release_(?:manifest|cask)\.py", body):
                with self.subTest(step=_step_name(body)):
                    self.assertGreater(index, tooling_idx)

    def test_unset_or_non_sha_variable_fails_closed(self):
        _, pin = _find_step(self.jobs["publish"], _is_pin_check)
        script = _run_block(pin)
        base_env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin")}
        refused = {
            "unset": None,
            "empty": "",
            "branch name": "main",
            "release tag": "v0.13.0",
            "39 characters": "a" * 39,
            "41 characters": "a" * 41,
            "uppercase": "A" * 40,
            "non-hex": "g" * 40,
        }
        for label, value in refused.items():
            with self.subTest(value=label):
                env = dict(base_env)
                if value is not None:
                    env["RELEASE_TOOLS_SHA"] = value
                result = subprocess.run(["bash", "-c", script], env=env, capture_output=True,
                                        text=True, timeout=30, cwd="/")
                self.assertNotEqual(result.returncode, 0, result.stderr)
                self.assertIn("RELEASE_TOOLS_SHA", result.stderr)
        env = dict(base_env, RELEASE_TOOLS_SHA="0123456789abcdef0123456789abcdef01234567")
        ok = subprocess.run(["bash", "-c", script], env=env, capture_output=True, text=True,
                            timeout=30, cwd="/")
        self.assertEqual(ok.returncode, 0, ok.stderr)


class EveryWorkflowTests(unittest.TestCase):
    """Properties that must hold for both release workflow files together."""

    def test_both_files_pin_every_action_and_persist_no_checkout_credentials(self):
        for path in (BUILD_WORKFLOW, PUBLISH_WORKFLOW):
            text = path.read_text()
            with self.subTest(file=path.name):
                for action, ref in re.findall(r"(?m)^\s+uses:\s*(\S+)@(\S+)", text):
                    self.assertRegex(ref, FULL_SHA, action)
                checkouts = len(re.findall(r"uses: actions/checkout@", text))
                persisted_false = len(re.findall(r"(?m)^\s+persist-credentials:\s*false\s*$", text))
                self.assertEqual(checkouts, persisted_false)

    def test_every_job_in_both_files_has_a_timeout(self):
        for path in (BUILD_WORKFLOW, PUBLISH_WORKFLOW):
            for name, job in _jobs(path.read_text()).items():
                with self.subTest(file=path.name, job=name):
                    self.assertRegex(job, r"(?m)^    timeout-minutes:\s*\d+\s*$")

    def test_workflow_level_permissions_are_empty_in_both_files(self):
        self.assertEqual(_perm_map(BUILD_TEXT), {})
        self.assertEqual(_perm_map(PUBLISH_TEXT), {})

    def test_only_one_workflow_is_triggered_by_a_tag_push(self):
        self.assertEqual(_on_keys(BUILD_TEXT), ["push"])
        self.assertEqual(_on_keys(PUBLISH_TEXT), ["workflow_run"])


def _ci_run(name, status="completed", conclusion="success", suite=CI_SUITE):
    """A check run as the workflow projects it (no details_url)."""
    return {
        "name": name,
        "status": status,
        "conclusion": conclusion,
        "check_suite": {"id": suite},
    }


class CiGateStepTests(unittest.TestCase):
    """Runs the publish CI-gate step script with stub gh and sleep (no GitHub contact)."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.bin = self.root / "bin"
        self.state = self.root / "state"
        self.work = self.root / "work"
        self.runner_temp = self.root / "runner"
        for path in (self.bin, self.state, self.runner_temp,
                     self.work / "release-tools" / "scripts"):
            path.mkdir(parents=True)
        for name, body in (("gh", FAKE_GH), ("sleep", FAKE_SLEEP)):
            stub = self.bin / name
            stub.write_text(body)
            stub.chmod(0o755)
        shutil.copy(MANIFEST_SCRIPT, self.work / "release-tools" / "scripts" / "release_manifest.py")
        _, gate = _find_step(_jobs(PUBLISH_TEXT)["publish"], _is_ci_gate)
        self.script = _run_block(gate)
        # The publisher run's own check-suite id, as the run lookup returns it.
        (self.state / "suite-id").write_text(f"{PUBLISHER_SUITE}\n")

    def tearDown(self):
        self._tmp.cleanup()

    def _polls(self, *polls):
        for number, runs in enumerate(polls, 1):
            (self.state / f"poll-{number}.jsonl").write_text(
                "".join(json.dumps(run) + "\n" for run in runs))
        (self.state / "poll-last.jsonl").write_text(
            "".join(json.dumps(run) + "\n" for run in polls[-1]))

    def _gh_calls(self):
        path = self.state / "gh-calls"
        return int(path.read_text()) if path.exists() else 0

    def _sleeps(self):
        path = self.state / "sleeps"
        return path.read_text().split() if path.exists() else []

    def _execute(self):
        env = {
            "PATH": f"{self.bin}{os.pathsep}{os.environ.get('PATH', '/usr/bin:/bin')}",
            "HOME": str(self.root),
            "FAKE_STATE": str(self.state),
            "RUNNER_TEMP": str(self.runner_temp),
            "GH_REPO": "nawodyaishan/agentic-sdd",
            "TAG_COMMIT": COMMIT,
            "GITHUB_RUN_ID": PUBLISHER_RUN,
        }
        return subprocess.run(["bash", "-c", self.script], env=env, cwd=self.work,
                              capture_output=True, text=True, timeout=120)

    def test_in_progress_run_blocks_until_it_completes_then_publishes(self):
        self._polls(
            [_ci_run("test", status="in_progress", conclusion=None),
             _ci_run("publish", status="in_progress", conclusion=None, suite=PUBLISHER_SUITE)],
            [_ci_run("test", status="in_progress", conclusion=None)],
            [_ci_run("test")],
        )
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self._gh_calls(), 3)
        self.assertEqual(self._sleeps(), ["30", "30"])
        self.assertIn("completed successfully", result.stdout)

    def test_timeout_fails_closed_after_the_bounded_poll_budget(self):
        self._polls([_ci_run("test", status="in_progress", conclusion=None)])
        polls = int(re.search(r"(?m)^polls=(\d+)$", self.script).group(1))
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("did not complete within 40 minutes", result.stderr)
        self.assertEqual(self._gh_calls(), polls)
        self.assertEqual(len(self._sleeps()), polls - 1)
        self.assertNotIn("completed successfully", result.stdout)

    def test_completed_failure_fails_fast_even_while_another_run_is_pending(self):
        self._polls([_ci_run("lint", conclusion="failure"),
                     _ci_run("slow", status="in_progress", conclusion=None)])
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("lint (failure)", result.stderr)
        self.assertEqual(self._gh_calls(), 1)
        self.assertEqual(self._sleeps(), [])

    def test_unknown_status_fails_at_once_without_polling_the_budget(self):
        self._polls([_ci_run("test"), _ci_run("odd", status="stale", conclusion=None)])
        result = self._execute()
        self.assertEqual(result.returncode, 1, result.stderr)
        self.assertIn("refusing to publish", result.stderr)
        self.assertEqual(self._gh_calls(), 1)
        self.assertEqual(self._sleeps(), [])

    def test_no_runs_fails_immediately_without_waiting(self):
        self._polls([])
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no check runs", result.stderr)
        self.assertEqual(self._gh_calls(), 1)
        self.assertEqual(self._sleeps(), [])

    def test_api_error_fails_closed(self):
        self._polls([_ci_run("test")])
        (self.state / "gh-fail").write_text("")
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("fake gh failure", result.stderr)
        self.assertNotIn("completed successfully", result.stdout)
        self.assertEqual(self._sleeps(), [])

    def test_all_completed_success_passes_without_waiting(self):
        self._polls([_ci_run("test"), _ci_run("docs")])
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self._gh_calls(), 1)
        self.assertEqual(self._sleeps(), [])

    def test_build_run_on_the_same_commit_counts_as_another_suite_and_must_pass(self):
        # The build run is a different check suite on the tagged commit; it is not excluded.
        build_suite = 777
        self._polls([_ci_run("verify", suite=build_suite, conclusion="failure")])
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("verify (failure)", result.stderr)

    def test_own_suite_run_is_excluded_even_while_incomplete(self):
        self._polls([_ci_run("test"),
                     _ci_run("publish", status="in_progress", conclusion=None,
                             suite=PUBLISHER_SUITE)])
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self._gh_calls(), 1)

    def test_foreign_suite_run_is_not_excluded_by_a_url_naming_the_publisher_run(self):
        sneaky = _ci_run("sneaky", status="in_progress", conclusion=None)
        sneaky["details_url"] = "https://github.com/nawodyaishan/agentic-sdd/actions/runs/222/job/1"
        self._polls([_ci_run("test"), sneaky], [_ci_run("test"), _ci_run("sneaky")])
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self._gh_calls(), 2)
        self.assertEqual(self._sleeps(), ["30"])

    def test_run_without_a_check_suite_is_counted_not_excluded(self):
        legacy_pending = {"name": "legacy", "status": "in_progress", "conclusion": None}
        legacy_done = {"name": "legacy", "status": "completed", "conclusion": "success"}
        self._polls([_ci_run("test"), legacy_pending], [_ci_run("test"), legacy_done])
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self._gh_calls(), 2)

    def test_suite_lookup_failure_aborts_before_any_poll(self):
        self._polls([_ci_run("test")])
        (self.state / "suite-fail").write_text("")
        result = self._execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("cannot resolve the check-suite id", result.stderr)
        self.assertEqual(self._gh_calls(), 0)
        self.assertNotIn("completed successfully", result.stdout)

    def test_unusable_suite_id_aborts_before_any_poll(self):
        self._polls([_ci_run("test")])
        for value in ("null", "", "0", "abc", "12 3"):
            with self.subTest(value=value):
                (self.state / "suite-id").write_text(value + "\n")
                result = self._execute()
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("not a positive integer", result.stderr)
                self.assertEqual(self._gh_calls(), 0)


class TagValidationStepTests(unittest.TestCase):
    """Runs the publisher's first step (strict tag and commit pattern) under bash."""

    @classmethod
    def setUpClass(cls):
        _, step = _find_step(_jobs(PUBLISH_TEXT)["publish"],
                             lambda b: _step_name(b).startswith("Validate the tag"))
        cls.script = _run_block(step)

    def _run(self, tag, commit=COMMIT):
        env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"),
               "TAG_NAME": tag, "TAG_COMMIT": commit}
        return subprocess.run(["bash", "-c", self.script], env=env, capture_output=True,
                              text=True, timeout=30, cwd="/")

    def test_accepts_release_tags_and_a_full_lowercase_commit(self):
        for tag in ("v0.13.0", "v10.20.300", "v0.0.0"):
            with self.subTest(tag=tag):
                self.assertEqual(self._run(tag).returncode, 0, tag)

    def test_refuses_anything_that_is_not_a_plain_release_tag(self):
        for tag in ("v0.13", "v0.13.0-rc1", "0.13.0", "v0.13.0 ", "v0.13.0\n", "v01.2.3",
                    "main", "", "v0.13.0; rm -rf /", "v0.13.0/../x", "v0.13.0.1"):
            with self.subTest(tag=tag):
                result = self._run(tag)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("refusing to publish", result.stderr)

    def test_refuses_a_commit_that_is_not_a_full_lowercase_sha(self):
        for commit in ("main", "A" * 40, "a" * 39, "a" * 41, "g" * 40, ""):
            with self.subTest(commit=commit):
                result = self._run(TAG, commit=commit)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("refusing to publish", result.stderr)


class DefaultBranchAncestryTests(unittest.TestCase):
    """Runs the publish ancestry step with a stub git (no repository, no GitHub contact)."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.root = Path(self._tmp.name)
        self.bin = self.root / "bin"
        self.state = self.root / "state"
        self.bin.mkdir()
        self.state.mkdir()
        stub = self.bin / "git"
        stub.write_text(FAKE_GIT)
        stub.chmod(0o755)
        _, step = _find_step(_jobs(PUBLISH_TEXT)["publish"], _is_ancestry_check)
        self.script = _run_block(step)

    def tearDown(self):
        self._tmp.cleanup()

    def _execute(self, branch="main", fetch_rc=0, ancestor_rc=0, tag_commit=COMMIT,
                 revparse_rc=0):
        env = {
            "PATH": f"{self.bin}{os.pathsep}{os.environ.get('PATH', '/usr/bin:/bin')}",
            "HOME": str(self.root),
            "FAKE_STATE": str(self.state),
            "TAG_NAME": TAG,
            "TAG_COMMIT": COMMIT,
            "DEFAULT_BRANCH": branch,
            "FAKE_FETCH_RC": str(fetch_rc),
            "FAKE_ANCESTOR_RC": str(ancestor_rc),
            "FAKE_TAG_COMMIT": tag_commit,
            "FAKE_REVPARSE_RC": str(revparse_rc),
        }
        return subprocess.run(["bash", "-c", self.script], env=env, cwd=self.root,
                              capture_output=True, text=True, timeout=60)

    def _git_calls(self):
        path = self.state / "git-calls"
        return path.read_text().splitlines() if path.exists() else []

    def test_reachable_tag_commit_passes_after_fetching_the_default_branch(self):
        result = self._execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self._git_calls()
        self.assertEqual(len(calls), 3)
        self.assertTrue(calls[0].startswith("fetch "))
        self.assertIn("+refs/heads/main:refs/remotes/origin/main", calls[0])
        self.assertNotIn("--depth", calls[0])
        self.assertTrue(calls[1].startswith("rev-parse --verify refs/tags/v0.13.0"))
        self.assertEqual(calls[2], f"merge-base --is-ancestor {COMMIT} origin/main")

    def test_fetch_failure_fails_closed_before_any_ancestry_decision(self):
        result = self._execute(fetch_rc=1)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("cannot fetch default branch main", result.stderr)
        self.assertEqual(len(self._git_calls()), 1)

    def test_tag_that_does_not_resolve_to_the_build_commit_is_refused(self):
        result = self._execute(tag_commit=OTHER_COMMIT)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing to publish", result.stderr)
        self.assertEqual(len(self._git_calls()), 2)

    def test_tag_resolution_error_is_refused(self):
        result = self._execute(revparse_rc=128, tag_commit="")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing to publish", result.stderr)

    def test_commit_not_reachable_from_default_branch_is_refused(self):
        result = self._execute(ancestor_rc=1)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing to publish", result.stderr)

    def test_ancestry_check_error_also_fails_closed(self):
        result = self._execute(ancestor_rc=128)
        self.assertNotEqual(result.returncode, 0)

    def test_missing_or_unsafe_default_branch_refuses_before_any_git_call(self):
        for branch in ("", "-x", "a..b", "ma in", "main;rm"):
            with self.subTest(branch=branch):
                result = self._execute(branch=branch)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("default branch name", result.stderr)
                self.assertEqual(self._git_calls(), [])


class GoReleaserConfigIsReadOnlyInputTests(unittest.TestCase):
    def test_goreleaser_keeps_homebrew_cask_token_template_and_no_snapshot(self):
        text = GORELEASER.read_text()
        self.assertIn('token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"', text)


if __name__ == "__main__":
    unittest.main()
