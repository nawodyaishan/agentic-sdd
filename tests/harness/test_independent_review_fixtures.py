"""Offline T03 fixtures for independent review: reconciliation, freshness and scheduling.

Scripted child reports exercise a test-only reference consumer
(independent_review_contract.py). Freshness uses the real `agentic-sdd opencode
evidence snapshot` binary on private scratch Git repositories. No reviewer, model,
provider, VM or real project is involved, so these checks say nothing about model
behaviour or native runtime enforcement.
"""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import independent_review_contract as irc

ROOT = Path(__file__).resolve().parents[2]
CUR = "sha256-v1:current-candidate"
OLD = "sha256-v1:earlier-candidate"
SCOPES = ("src", "specs/demo", "tests")


def clean(**changes):
    report = {"identity": CUR, "execution": "completed", "evidence": "current",
              "violations": [], "verdict": "Ready", "findings": []}
    report.update(changes)
    return report


def finding(fid, **changes):
    value = {"id": fid, "severity": "high", "impact": "behaviour is wrong",
             "location": "src/code.go:10", "evidence": "the line shows the value"}
    value.update(changes)
    return value


# (name, report, parent dispositions, independently written effective verdict)
FIXTURES = [
    ("clean candidate", clean(), {}, "Ready"),
    ("seeded defect reported as Ready", clean(findings=[finding("F1")]), {"F1": "valid"}, "Needs changes"),
    ("insufficient test reported as Needs changes",
     clean(verdict="Needs changes", findings=[finding("F2")]), {"F2": "valid"}, "Needs changes"),
    ("disputed advice does not block", clean(findings=[finding("F3")]), {"F3": "disputed"}, "Ready"),
    ("needs evidence blocks Ready", clean(findings=[finding("F4")]), {"F4": "needs evidence"}, "Blocked"),
    ("finding without parent disposition", clean(findings=[finding("F5")]), {}, "Blocked"),
    ("malformed finding without location",
     clean(findings=[finding("F6", location="")]), {"F6": "valid"}, "Blocked"),
    ("candidate changed after review", clean(identity=OLD), {}, "Stale"),
    ("missing evidence", clean(evidence="missing"), {}, "Blocked"),
    ("truncated evidence", clean(evidence="truncated"), {}, "Blocked"),
    ("cancelled mid-run reported as Ready", clean(execution="cancelled"), {}, "Blocked"),
    ("step limit exhausted", clean(execution="incomplete", verdict=None), {}, "Blocked"),
    ("provider failure", clean(execution="failed", verdict=None), {}, "Blocked"),
    ("prohibited edit attempted", clean(violations=["edit specs/demo/tasks.md"]), {}, "Blocked"),
    ("completed run without verdict", clean(verdict=None), {}, "Blocked"),
]


class ReconciliationFixtureTests(unittest.TestCase):
    def test_each_fixture_reconciles_to_its_independent_expectation(self):
        for name, report, parent, expected in FIXTURES:
            with self.subTest(fixture=name):
                self.assertEqual(irc.reconcile(report, parent, CUR)["effective"], expected)

    def test_every_rule_changes_some_fixture_outcome(self):
        # A rule no fixture depends on could be deleted without any failure.
        covered = set()
        for rule in irc.RULES:
            for _, report, parent, _ in FIXTURES:
                if irc.reconcile(report, parent, CUR)["effective"] != irc.reconcile(report, parent, CUR, skip={rule})["effective"]:
                    covered.add(rule)
        self.assertEqual(covered, set(irc.RULES))

    def test_blocked_or_stale_outcomes_never_say_ready(self):
        for name, report, parent, expected in FIXTURES:
            if expected in ("Blocked", "Stale"):
                with self.subTest(fixture=name):
                    self.assertNotEqual(irc.reconcile(report, parent, CUR)["effective"], "Ready")


class SchedulingContractTests(unittest.TestCase):
    def test_one_explicit_request_on_new_candidate_launches(self):
        self.assertEqual(irc.plan_launch([], {"identity": CUR, "explicit": True})[0], True)

    def test_active_reviewer_blocks_a_second_launch(self):
        history = [{"identity": CUR, "execution": "incomplete", "active": True}]
        self.assertFalse(irc.plan_launch(history, {"identity": OLD, "explicit": True})[0])

    def test_implicit_or_automatic_retry_never_launches(self):
        history = [{"identity": CUR, "execution": "cancelled"}]
        self.assertFalse(irc.plan_launch(history, {"identity": CUR, "explicit": False})[0])

    def test_identical_completed_review_is_not_duplicated(self):
        history = [{"identity": CUR, "execution": "completed"}]
        self.assertFalse(irc.plan_launch(history, {"identity": CUR, "explicit": True})[0])

    def test_new_explicit_request_after_interruption_is_allowed_and_explicit(self):
        history = [{"identity": CUR, "execution": "cancelled"}]
        self.assertTrue(irc.plan_launch(history, {"identity": CUR, "explicit": True})[0])

    def test_changed_candidate_after_completed_review_needs_a_new_explicit_request(self):
        history = [{"identity": OLD, "execution": "completed"}]
        self.assertTrue(irc.plan_launch(history, {"identity": CUR, "explicit": True})[0])
        self.assertFalse(irc.plan_launch(history, {"identity": CUR, "explicit": False})[0])


def build_binary(dest):
    if os.environ.get("AGENTIC_SDD_BIN"):
        return Path(os.environ["AGENTIC_SDD_BIN"])
    if shutil.which("go") is None:
        raise unittest.SkipTest("go toolchain unavailable; freshness fixtures not run")
    env = dict(os.environ, GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
               GOCACHE=os.environ.get("GOCACHE", str(ROOT / ".cache/go-build")))
    out = Path(dest) / "agentic-sdd"
    subprocess.run(["go", "build", "-trimpath", "-o", str(out), "./cmd/agentic-sdd"],
                   cwd=ROOT, env=env, check=True, capture_output=True, text=True, timeout=600)
    return out


class FreshnessFixtureTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        base = Path(os.environ.get("TMPDIR") or "/tmp/opencode")
        base.mkdir(parents=True, exist_ok=True)
        cls.base = base
        cls.scratch = Path(tempfile.mkdtemp(prefix="ir-fixtures-", dir=base))
        cls.addClassCleanup(shutil.rmtree, cls.scratch, True)
        cls.binary = build_binary(cls.scratch)

    def setUp(self):
        self.repo = Path(tempfile.mkdtemp(prefix="repo-", dir=self.scratch))
        self.git("init", "-q")
        self.write("src/code.txt", "good\n")
        self.write("specs/demo/spec.md", "spec\n")
        self.write("tests/code_test.txt", "expect 1\n")
        self.write("docs/notes.md", "tracked notes\n")
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "base")
        # Unrelated dirty work that must survive every in-scope change and restore.
        self.write("docs/notes.md", "unrelated dirty work\n")

    def git(self, *args):
        return subprocess.run(["git", "-c", "user.email=fixture@example.invalid", "-c", "user.name=fixture", *args],
                              cwd=self.repo, check=True, capture_output=True, text=True, timeout=60)

    def write(self, rel, text):
        path = self.repo / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def restore_scope(self):
        # Path-limited restore: never touches the unrelated docs/notes.md sentinel.
        self.git("reset", "-q", "--", *SCOPES)
        self.git("checkout", "-q", "--", *SCOPES)
        self.git("clean", "-fdq", "--", *SCOPES)

    def snapshot(self, workspace=None):
        args = [str(self.binary), "opencode", "evidence", "snapshot", "--workspace", str(workspace or self.repo)]
        for scope in SCOPES:
            args += ["--scope", scope]
        done = subprocess.run(args, capture_output=True, text=True, timeout=120)
        if done.returncode != 0:
            return {"status": "missing", "detail": done.stderr.strip()}
        return {"status": "current", "identity": json.loads(done.stdout)["identity"]["diff"]}

    def changes(self):
        return [
            ("unstaged edit in src", lambda: self.write("src/code.txt", "defect\n")),
            ("staged edit in src", lambda: (self.write("src/code.txt", "staged\n"), self.git("add", "src/code.txt"))),
            ("untracked file in tests", lambda: self.write("tests/new_test.txt", "new\n")),
            ("deleted file in tests", lambda: (self.repo / "tests/code_test.txt").unlink()),
            ("mode change in src", lambda: os.chmod(self.repo / "src/code.txt", 0o755)),
            ("edit in specs/demo", lambda: self.write("specs/demo/spec.md", "changed spec\n")),
        ]

    def test_each_relevant_change_invalidates_identity_and_revert_restores_it(self):
        base = self.snapshot()
        self.assertEqual(base["status"], "current")
        for name, do in self.changes():
            with self.subTest(change=name):
                do()
                changed = self.snapshot()
                self.assertEqual(changed["status"], "current")
                self.assertNotEqual(changed["identity"], base["identity"])
                self.restore_scope()
                self.assertEqual(self.snapshot()["identity"], base["identity"])
        self.assertEqual((self.repo / "docs/notes.md").read_text(), "unrelated dirty work\n")

    def test_unrelated_change_outside_scope_keeps_identity(self):
        base = self.snapshot()
        self.write("docs/notes.md", "more unrelated notes\n")
        self.assertEqual(self.snapshot()["identity"], base["identity"])

    def test_consumer_marks_review_stale_after_a_real_relevant_change(self):
        base = self.snapshot()
        report = clean(identity=base["identity"])
        self.assertEqual(irc.reconcile(report, {}, base["identity"])["effective"], "Ready")
        self.write("src/code.txt", "defect\n")
        changed = self.snapshot()
        self.assertEqual(irc.reconcile(report, {}, changed["identity"])["effective"], "Stale")
        self.restore_scope()

    def test_workspace_without_git_is_missing_evidence_not_a_pass(self):
        plain = Path(tempfile.mkdtemp(prefix="plain-", dir=self.scratch))
        (plain / "src").mkdir()
        (plain / "src/code.txt").write_text("not a repository\n")
        result = self.snapshot(plain)
        self.assertEqual(result["status"], "missing")
        self.assertEqual(irc.reconcile(clean(evidence="missing"), {}, CUR)["effective"], "Blocked")


if __name__ == "__main__":
    unittest.main()
