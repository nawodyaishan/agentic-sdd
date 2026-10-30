"""Offline receipt tests: synthetic Git trees/logs only, no native/provider store."""
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import receipt_tools as rt


ROOT = Path(__file__).resolve().parents[2]


def digest(data):
    return hashlib.sha256(data).hexdigest()


class ReceiptTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.workspace = self.root / "project"
        self.workspace.mkdir()
        self.git("init", "-q")
        self.git("config", "user.name", "Synthetic Fixture")
        self.git("config", "user.email", "fixture@example.invalid")
        (self.workspace / "code.txt").write_text("original\n")
        (self.workspace / "tasks.md").write_text("B1 awaiting human review\n")
        self.git("add", ".")
        self.git("-c", "commit.gpgsign=false", "commit", "-qm", "synthetic fixture")
        self.evidence = self.root / "evidence"
        self.evidence.mkdir(mode=0o700)
        self.log = self.evidence / "test.log"
        self.log.write_text("independent synthetic assertion passed\n")
        self.log.chmod(0o600)
        self.contract = (ROOT / "harness/checks.json").read_bytes()
        self.identity = {k: "fixture-" + k for k in (
            "base_revision", "diff", "config", "skills", "environment", "runtime", "harness", "contract")}
        self.identity.update(rt.snapshot(self.workspace, ["."]))
        self.identity["contract"] = digest(self.contract)
        self.receipt = json.loads((ROOT / "harness/evidence/example-direct-fix.json").read_text())
        self.receipt.update(workspace=str(self.workspace), identity=self.identity.copy(), execution="completed", review=None)
        self.receipt["unknowns"] = {k: v for k, v in self.receipt["unknowns"].items() if k not in ("/identity/diff", "/identity/runtime")}
        self.check = dict(id="test", status="passed", exit_code=0,
                          started_at="2026-10-29T00:00:00Z", ended_at="2026-10-29T00:00:01Z",
                          evidence=str(self.log), truncated=False, reason="process exited zero")
        self.receipt["checks"] = [self.check.copy()]
        self.result = dict(version=1, workspace=str(self.workspace), contract_sha256=digest(self.contract),
                           selection=copy.deepcopy(self.receipt["selection"]), execution="completed", checks=[self.check.copy()])
        self.log_digests = {str(self.log): digest(self.log.read_bytes())}

    def git(self, *args):
        return subprocess.check_output(["git", "--no-optional-locks", "-C", str(self.workspace), *args],
                                       stderr=subprocess.PIPE, timeout=10,
                                       env={"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(self.root),
                                            "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1",
                                            "GIT_CONFIG_COUNT": "2", "GIT_CONFIG_KEY_0": "init.templateDir", "GIT_CONFIG_VALUE_0": "",
                                            "GIT_CONFIG_KEY_1": "core.hooksPath", "GIT_CONFIG_VALUE_1": os.devnull})

    def validate(self, receipt=None, **kwargs):
        options = dict(workspace=self.workspace, identity=self.identity, authorization_ref=self.receipt["authorization_ref"],
                       contract=self.contract, result=self.result, evidence_root=self.evidence, log_digests=self.log_digests)
        options.update(kwargs)
        return rt.validate_receipt(receipt or self.receipt, **options)

    def test_valid_direct_fix_does_not_invent_feature_or_mutate_project(self):
        before = {str(p.relative_to(self.workspace)): p.read_bytes() for p in self.workspace.rglob("*") if p.is_file()}
        self.validate()
        after = {str(p.relative_to(self.workspace)): p.read_bytes() for p in self.workspace.rglob("*") if p.is_file()}
        self.assertEqual(before, after)
        self.assertIsNone(self.receipt["feature"])
        self.assertIsNone(self.receipt["batch"])
        self.assertEqual([], self.receipt["task_ids"])

    def test_snapshot_detects_index_worktree_untracked_deleted_and_mode(self):
        base = rt.snapshot(self.workspace, ["."])
        self.assertEqual(self.git("rev-parse", "HEAD").decode().strip(), base["base_revision"])
        self.assertRegex(base["diff"], r"^sha256-v1:[0-9a-f]{64}$")
        (self.workspace / "code.txt").write_text("staged\n")
        self.git("add", "code.txt")
        (self.workspace / "code.txt").write_text("original\n")
        staged = rt.snapshot(self.workspace, ["."])
        self.assertNotEqual(base, staged, "staged content matters even when worktree equals HEAD")
        (self.workspace / "code.txt").write_text("unstaged\n")
        unstaged = rt.snapshot(self.workspace, ["."])
        self.assertNotEqual(staged, unstaged)
        (self.workspace / "new.txt").write_text("untracked\n")
        added = rt.snapshot(self.workspace, ["."])
        self.assertNotEqual(unstaged, added)
        (self.workspace / "code.txt").unlink()
        deleted = rt.snapshot(self.workspace, ["."])
        self.assertNotEqual(added, deleted)
        (self.workspace / "tasks.md").chmod(0o755)
        self.assertNotEqual(deleted, rt.snapshot(self.workspace, ["."]))

    def test_scope_is_explicit_and_unsafe_files_do_not_get_followed(self):
        focused = rt.snapshot(self.workspace, ["code.txt"])
        (self.workspace / "unrelated.txt").write_text("not a dependency\n")
        self.assertEqual(focused, rt.snapshot(self.workspace, ["code.txt"]))
        self.assertNotEqual(focused, rt.snapshot(self.workspace, ["."]))
        for paths in ([], ["../outside"], ["/absolute"], ["*.txt"]):
            with self.subTest(paths=paths), self.assertRaises(ValueError):
                rt.snapshot(self.workspace, paths)
        os.symlink(self.root / "outside-secret", self.workspace / "link")
        with self.assertRaises(ValueError):
            rt.snapshot(self.workspace, ["."])
        (self.workspace / "link").unlink()
        os.mkfifo(self.workspace / "fifo")
        with self.assertRaises(ValueError):
            rt.snapshot(self.workspace, ["."])

    def test_git_identity_does_not_inherit_external_index_or_global_configuration(self):
        expected = rt.snapshot(self.workspace, ["."])
        foreign = self.root / "foreign-index"
        foreign.write_text("not this fixture's Git index")
        with patch.dict(os.environ, {"GIT_INDEX_FILE": str(foreign), "GIT_CONFIG_GLOBAL": str(self.root / "unread-global")}):
            self.assertEqual(expected, rt.snapshot(self.workspace, ["."]))

    def test_scoped_directory_cannot_traverse_a_symlink_ancestor(self):
        outside = self.root / "outside" / "subdir"
        outside.mkdir(parents=True)
        (outside / "not-authorized.txt").write_text("synthetic external content")
        (self.workspace / "alias").symlink_to(outside.parent, target_is_directory=True)
        with self.assertRaises(ValueError):
            rt.snapshot(self.workspace, ["alias/subdir"])

    def test_stale_and_unknown_identity_never_establish_reusable_pass(self):
        for key in self.identity:
            with self.subTest(key=key):
                changed = self.identity.copy()
                changed[key] = "changed"
                with self.assertRaisesRegex(ValueError, "identity|fresh"):
                    self.validate(identity=changed)
                receipt = copy.deepcopy(self.receipt)
                receipt["identity"][key] = None
                receipt["unknowns"]["/identity/" + key] = "not observed"
                with self.assertRaises(ValueError):
                    self.validate(receipt, identity=receipt["identity"])

    def test_schema_unknown_reasons_and_semantic_consistency_fail_closed(self):
        mutations = [
            lambda r: r.update(approved=True),
            lambda r: r.update(batch="invented-batch"),
            lambda r: r.update(workflow_state="invented direct-fix task state"),
            lambda r: r["unknowns"].pop("/cost_usd"),
            lambda r: r["unknowns"].update({"/cost_usd": "   "}),
            lambda r: r["unknowns"].update({"/not/a/field": "not a real observation"}),
            lambda r: r["checks"].append(r["checks"][0].copy()),
            lambda r: r["checks"][0].update(exit_code=9),
            lambda r: r["checks"][0].update(ended_at="2026-10-28T23:59:59Z"),
            lambda r: r["checks"][0].update(truncated=True),
            lambda r: r["checks"][0].update(evidence=None),
            lambda r: r.update(review="Ready"),  # focused is not required coverage
            lambda r: r.update(authorization_ref="receipt invents permission"),
            lambda r: r.update(execution="incomplete"),
        ]
        for i, mutate in enumerate(mutations):
            with self.subTest(case=i):
                receipt = copy.deepcopy(self.receipt)
                mutate(receipt)
                observed = copy.deepcopy(self.result)
                for key in ("checks", "selection", "execution"):
                    observed[key] = copy.deepcopy(receipt[key])
                with self.assertRaises(ValueError):
                    self.validate(receipt, result=observed)
        with self.assertRaises(ValueError):
            self.validate(authorization_ref=None)

    def test_required_coverage_cannot_be_forged_by_selection_label(self):
        self.receipt["selection"]["scope"] = "required"
        self.result["selection"]["scope"] = "required"
        self.receipt["review"] = "Ready"
        with self.assertRaisesRegex(ValueError, "coverage|selection"):
            self.validate()

    def test_unobserved_billing_session_model_and_paid_permission_cannot_be_invented(self):
        for field in ("cost_usd", "session_id", "paid_authorized", "model"):
            with self.subTest(field=field):
                receipt = copy.deepcopy(self.receipt)
                if field == "cost_usd":
                    receipt[field] = 0
                    receipt["unknowns"].pop("/cost_usd")
                elif field == "session_id":
                    receipt[field] = "fabricated-session"
                    receipt["unknowns"].pop("/session_id")
                elif field == "paid_authorized":
                    receipt["limits"][field] = True
                else:
                    receipt["model"]["id"] = "fabricated-model"
                    receipt["unknowns"].pop("/model/id")
                with self.assertRaises(ValueError):
                    self.validate(receipt)

    def test_independently_observed_synthetic_native_values_are_not_blanket_rejected(self):
        self.receipt["session_id"] = "synthetic-session"
        self.receipt["cost_usd"] = 0.01
        self.receipt["model"] = dict(provider="synthetic", id="synthetic-model", variant="fixture")
        self.receipt["limits"].update(paid_authorized=True, spend_usd=0.25)
        observations = {"/session_id": "synthetic-session", "/cost_usd": 0.01,
                        "/model/provider": "synthetic", "/model/id": "synthetic-model", "/model/variant": "fixture"}
        for ptr in observations:
            self.receipt["unknowns"].pop(ptr)
        self.validate(observations=observations, paid_authorized=True)

    def test_evidence_missing_tampered_public_or_symlink_is_not_a_pass(self):
        self.log.write_text("replaced evidence at the same path\n")
        with self.assertRaises(ValueError):
            self.validate()
        self.log.write_text("independent synthetic assertion passed\n")
        self.log.chmod(0o644)
        with self.assertRaises(ValueError):
            self.validate()
        self.log.unlink()
        with self.assertRaises(ValueError):
            self.validate()
        other = self.root / "outside.log"
        other.write_text("independent synthetic assertion passed\n")
        other.chmod(0o600)
        self.log.symlink_to(other)
        with self.assertRaises(ValueError):
            self.validate()

    def test_dotdot_evidence_and_export_paths_cannot_escape_boundaries(self):
        escaped = self.evidence / ".." / "outside.log"
        escaped.write_bytes(self.log.read_bytes())
        escaped.chmod(0o600)
        self.receipt["checks"][0]["evidence"] = str(escaped)
        self.result["checks"][0]["evidence"] = str(escaped)
        with self.assertRaises(ValueError):
            self.validate(log_digests={str(escaped): digest(escaped.read_bytes())})
        self.workspace.chmod(0o700)  # An otherwise-private parent must not mask traversal.
        with self.assertRaises(ValueError):
            rt.export_receipt(self.receipt, self.evidence / ".." / "project" / "leaked.json", workspace=self.workspace)
        self.assertFalse((self.workspace / "leaked.json").exists())

    def test_ready_requires_coverage_fresh_evidence_and_no_unresolved_findings(self):
        single = json.loads(self.contract)
        single["checks"] = [ch for ch in single["checks"] if ch["id"] == "test"]
        self.contract = json.dumps(single).encode()
        self.identity["contract"] = digest(self.contract)
        self.receipt["identity"] = self.identity.copy()
        self.result["contract_sha256"] = digest(self.contract)
        self.receipt["selection"]["scope"] = self.result["selection"]["scope"] = "required"
        self.receipt["review"] = "Ready"
        self.validate()  # A validator that rejects every Ready is not adequate.
        finding = dict(id="F1", location="code.txt", impact="contract defect", evidence="fixture", disposition="needs evidence")
        self.receipt["findings"] = [finding]
        with self.assertRaises(ValueError):
            self.validate()
        finding["disposition"] = "disputed"
        self.validate()
        self.receipt["findings"].append(finding.copy())
        with self.assertRaises(ValueError):
            self.validate()

    def test_export_redacts_before_write_and_refuses_unapproved_fields_and_paths(self):
        canary = "SYNTHETIC-SECRET-CANARY-DO-NOT-EXPORT-12345"
        self.receipt["request_ref"] = "request containing " + canary
        self.receipt["unknowns"]["/cost_usd"] = "Bearer " + canary
        output = self.evidence / "receipt.json"
        rt.export_receipt(self.receipt, output, workspace=self.workspace, redactions=[canary])
        self.assertTrue(output.exists(), "export must produce a sanitized receipt")
        self.assertNotIn(canary.encode(), output.read_bytes())
        self.assertNotIn(canary, output.name)
        self.assertEqual(0, output.stat().st_mode & 0o077)
        exported = json.loads(output.read_text())
        self.validate(exported)
        with self.assertRaises(ValueError):
            rt.export_receipt(self.receipt, output, workspace=self.workspace, redactions=[canary])
        with self.assertRaises(ValueError):
            rt.export_receipt(self.receipt, self.workspace / "receipt.json", workspace=self.workspace, redactions=[canary])
        bad = copy.deepcopy(self.receipt)
        bad["raw_native_trace"] = canary
        with self.assertRaises(ValueError) as error:
            rt.export_receipt(bad, self.evidence / "raw.json", workspace=self.workspace, redactions=[canary])
        self.assertNotIn(canary, str(error.exception), "rejection messages must not echo secret values")
        self.assertFalse((self.evidence / "raw.json").exists())
        linked = self.root / "linked"
        linked.symlink_to(self.evidence, target_is_directory=True)
        with self.assertRaises(ValueError):
            rt.export_receipt(self.receipt, linked / "escape.json", workspace=self.workspace, redactions=[canary])

    def test_saved_receipt_survives_reload_but_changed_code_tasks_or_authority_block_reuse(self):
        output = self.evidence / "saved.json"
        rt.export_receipt(self.receipt, output, workspace=self.workspace)
        saved = json.loads(output.read_text())
        self.validate(saved)
        with self.assertRaises(ValueError):
            self.validate(saved, authorization_ref="new explicit B2 authorization")
        for name in ("code.txt", "tasks.md"):
            path = self.workspace / name
            original = path.read_bytes()
            path.write_bytes(original + b"changed while session was absent\n")
            current = self.identity.copy()
            current.update(rt.snapshot(self.workspace, ["."]))
            with self.assertRaisesRegex(ValueError, "identity|fresh"):
                self.validate(saved, identity=current)
            self.assertIn(b"changed while session was absent", path.read_bytes(), "validation must preserve dirty state")
            path.write_bytes(original)  # Only this fixture's own edit is restored.
        self.validate(saved)

    def test_workspace_move_requires_reconciliation_not_automatic_replay(self):
        saved = copy.deepcopy(self.receipt)
        before = rt.snapshot(self.workspace, ["."])
        moved = self.root / "moved"
        self.workspace.rename(moved)
        self.workspace = moved
        self.assertEqual(before, rt.snapshot(moved, ["."]))
        with self.assertRaisesRegex(ValueError, "workspace|location"):
            self.validate(saved)
        self.assertEqual("B1 awaiting human review\n", (moved / "tasks.md").read_text())

    def test_cancelled_or_incomplete_result_cannot_be_promoted_to_completed_or_ready(self):
        for status in ("cancelled", "incomplete", "failed", "blocked"):
            with self.subTest(status=status):
                receipt, result = copy.deepcopy(self.receipt), copy.deepcopy(self.result)
                receipt["execution"] = result["execution"] = status
                receipt["checks"][0]["status"] = result["checks"][0]["status"] = status
                if status == "blocked":
                    for check in (receipt["checks"][0], result["checks"][0]):
                        check["started_at"] = check["exit_code"] = None
                self.validate(receipt, result=result)
                receipt["review"] = "Ready"
                with self.assertRaises(ValueError):
                    self.validate(receipt, result=result)
                receipt["review"] = None
                receipt["execution"] = "completed"
                with self.assertRaises(ValueError):
                    self.validate(receipt, result=result)

    def test_check_status_cannot_be_relabelled_as_different_execution_outcome(self):
        self.receipt["checks"][0]["status"] = self.result["checks"][0]["status"] = "failed"
        self.receipt["execution"] = self.result["execution"] = "cancelled"
        with self.assertRaises(ValueError):
            self.validate()


if __name__ == "__main__":
    unittest.main()
