"""Offline tests for scripts/release_eligibility.py tag-run.

The run record mirrors a hosted one: with a `run-name`, GitHub puts the rendered
run-name in both `name` and `display_title` (found on a rehearsal run, where a
workflow-name check refused a genuine owner tag).
"""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

HELPER = Path(__file__).resolve().parents[2] / "scripts" / "release_eligibility.py"
TAG, SHA, BLOB, OWNER, REPO, RUN, WF = "v0.14.1", "a" * 40, "b" * 40, 50957846, 111, 222, 333


def record(**changes):
    title = f"refs/tags/{TAG}|true|{OWNER}|{SHA}"
    run = {
        "id": RUN, "name": title, "display_title": title, "status": "completed",
        "conclusion": "success", "event": "push", "path": ".github/workflows/release.yml",
        "workflow_id": WF, "head_sha": SHA, "head_branch": TAG,
        "repository": {"id": REPO}, "head_repository": {"id": REPO},
        "actor": {"id": OWNER, "type": "User"},
    }
    run.update(changes)
    return run


class TagRunTests(unittest.TestCase):
    def check(self, run, blob=BLOB, ref=None):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "run.json").write_text(json.dumps(run))
            (root / "ref.json").write_text(json.dumps(
                ref or {"ref": f"refs/tags/{TAG}", "object": {"type": "commit", "sha": SHA}}))
            (root / "obj.json").write_text("null")
            return subprocess.run(
                [sys.executable, "-I", "-B", str(HELPER), "tag-run",
                 "--run", str(root / "run.json"), "--run-id", str(RUN), "--workflow-id", str(WF),
                 "--tagged-workflow-blob", BLOB, "--reviewed-workflow-blob", blob,
                 "--tag-ref", str(root / "ref.json"), "--tag-object", str(root / "obj.json"),
                 "--tag", TAG, "--tag-commit", SHA, "--repository-id", str(REPO)],
                capture_output=True, text=True, timeout=30)

    def test_hosted_shaped_owner_tag_passes(self):
        result = self.check(record())
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_refusals(self):
        title = f"refs/tags/{TAG}|true|{OWNER}|{SHA}"
        cases = {
            "tag not created": record(display_title=title.replace("|true|", "|false|")),
            "other sender": record(display_title=title.replace(str(OWNER), "1")),
            "other actor": record(actor={"id": 1, "type": "User"}),
            "bot actor": record(actor={"id": OWNER, "type": "Bot"}),
            "branch push": record(display_title=title.replace("refs/tags/", "refs/heads/")),
            "other commit": record(head_sha="c" * 40),
            "other repo": record(head_repository={"id": 999}),
            "not push": record(event="workflow_dispatch"),
            "failed build": record(conclusion="failure"),
            "other workflow": record(path=".github/workflows/evil.yml"),
        }
        for label, run in cases.items():
            with self.subTest(label):
                self.assertEqual(self.check(run).returncode, 1)

    def test_a_changed_workflow_blob_is_refused(self):
        self.assertEqual(self.check(record(), blob="d" * 40).returncode, 1)

    def test_a_tag_ref_at_another_commit_is_refused(self):
        ref = {"ref": f"refs/tags/{TAG}", "object": {"type": "commit", "sha": "e" * 40}}
        self.assertEqual(self.check(record(), ref=ref).returncode, 1)


if __name__ == "__main__":
    unittest.main()
