"""Real check -> receipt -> freshness/reload integration, NOT agent obedience.

Only synthetic files and stdlib child commands execute. No native runtime,
provider, copied user home/auth state, model call, or approval automation.
"""
import copy
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

import receipt_tools as rt


ROOT = Path(__file__).resolve().parents[2]
AUTHORITY = "synthetic offline fixture B1 only; never permission for a real feature"


@unittest.skipUnless(sys.platform == "linux", "runner execution is Linux-only; native recovery not tested")
class ConnectedOfflineTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp.cleanup)
        cls.root = Path(cls.temp.name)
        cls.env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(cls.root / "home"),
                   "TMPDIR": str(cls.root), "GOCACHE": str(cls.root / "cache"), "CGO_ENABLED": "0",
                   "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off", "GOENV": "off",
                   "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1",
                   "GIT_CONFIG_COUNT": "2", "GIT_CONFIG_KEY_0": "init.templateDir", "GIT_CONFIG_VALUE_0": "",
                   "GIT_CONFIG_KEY_1": "core.hooksPath", "GIT_CONFIG_VALUE_1": os.devnull}
        (cls.root / "home").mkdir(mode=0o700)
        cls.binary = cls.root / "sdd-check"
        subprocess.run([shutil.which("go"), "build", "-o", str(cls.binary), "./cmd/sdd-check"],
                       cwd=ROOT, env=cls.env, check=True, capture_output=True, timeout=120)
        # These are identities of the deterministic fixture, not a native agent.
        cls.fixed_identity = dict(
            config=rt.sha256(b"synthetic explicit stdlib commands; no integration or provider"),
            skills=rt.sha256(b"no agent skills executed in this deterministic fixture"),
            environment=rt.sha256(json.dumps({"platform": sys.platform, "python": sys.version,
                "go": subprocess.check_output([shutil.which("go"), "version"], env=cls.env, timeout=10).decode(),
                "env": cls.env}, sort_keys=True).encode()),
            runtime=rt.sha256(cls.binary.read_bytes()),
            harness=rt.sha256((ROOT / "tests/harness/receipt_tools.py").read_bytes()))

    def setUp(self):
        self.temp_case = tempfile.TemporaryDirectory(dir=self.root)
        self.addCleanup(self.temp_case.cleanup)
        self.case = Path(self.temp_case.name)
        self.workspace = self.case / "project"
        self.workspace.mkdir()
        self.git("init", "-q")
        (self.workspace / "code.txt").write_text("independent expected bytes\n")
        (self.workspace / "spec.md").write_text("Synthetic test authority only: " + AUTHORITY + "\n")
        (self.workspace / "tasks.md").write_text("B1 awaiting human review; B2 unauthorized\n")
        self.git("add", ".")
        self.git("-c", "user.name=Synthetic Fixture", "-c", "user.email=fixture@example.invalid",
                 "-c", "commit.gpgsign=false", "commit", "-qm", "synthetic connected input")
        self.evidence = self.case / "evidence"
        self.evidence.mkdir(mode=0o700)
        self.contract = None

    def git(self, *args):
        return subprocess.check_output(["git", "--no-optional-locks", "-C", str(self.workspace), *args],
                                       env=self.env, stderr=subprocess.PIPE, timeout=10)

    def define_contract(self, interrupted=False):
        first = ("from pathlib import Path; import time; "
                 "Path('partial.txt').write_text('preserve this partial effect\\n'); time.sleep(60)") if interrupted else (
                 "from pathlib import Path; assert Path('code.txt').read_bytes() == b'independent expected bytes\\n'")
        last = ("from pathlib import Path; Path('must-not-run.txt').write_text('unexpected replay')") if interrupted else (
                "from pathlib import Path; assert Path('tasks.md').read_text() == 'B1 awaiting human review; B2 unauthorized\\n'")
        checks = [dict(id=name, cwd=".", scopes=["required"], prerequisites=[sys.executable],
                       timeout_seconds=90, effects=dict(writes=["synthetic partial.txt"] if interrupted else [], network="none"),
                       steps=[dict(argv=[sys.executable, "-c", program], env={})],
                       local_recipe=[sys.executable, "-c", program], expected_exit=0, evidence=name + ".log")
                  for name, program in (("content", first), ("boundary", last))]
        self.contract = (json.dumps(dict(version=1, checks=checks), sort_keys=True) + "\n").encode()
        self.contract_path = self.case / "checks.json"
        self.contract_path.write_bytes(self.contract)

    def identity(self):
        return dict(self.fixed_identity, **rt.snapshot(self.workspace, ["."]), contract=rt.sha256(self.contract))

    def execute(self, name, *, interrupt=False, deadline="10s"):
        output = self.evidence / name
        process = subprocess.Popen([str(self.binary), "--scope", "required", "--workspace", str(self.workspace),
                                    "--contract", str(self.contract_path), "--output", str(output), "--deadline", deadline],
                                   env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        try:
            if interrupt:
                end = time.monotonic() + 5
                while not (self.workspace / "partial.txt").exists() and process.poll() is None and time.monotonic() < end:
                    time.sleep(0.01)
                self.assertTrue((self.workspace / "partial.txt").exists(), "must actually reach the owned writer")
                process.send_signal(signal.SIGINT)
            stdout, stderr = process.communicate(timeout=15)
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                process.communicate(timeout=10)
        result = json.loads((output / "result.json").read_text())
        self.assertEqual(result, json.loads(stdout), "stdout and persisted execution must agree")
        self.assertEqual(0 if result["execution"] == "completed" else 1, process.returncode, stderr)
        self.assertNotIn("review", result)
        self.assertNotIn("authorization_ref", result)
        logs = {ch["evidence"]: rt.sha256(Path(ch["evidence"]).read_bytes())
                for ch in result["checks"] if ch["evidence"] is not None}
        return result, logs

    def receipt(self, result, identity):
        receipt = json.loads((ROOT / "harness/evidence/example-direct-fix.json").read_text())
        receipt.update(synthetic=True, run_id="offline-connected-" + Path(result["checks"][0]["evidence"]).parent.name,
                       workspace=str(self.workspace), request_ref="synthetic component integration, not a native lifecycle",
                       authorization_ref=AUTHORITY, mode="build", feature="synthetic-fixture", batch="B1", task_ids=["F01"],
                       workflow_state="fixture B1 awaiting human review; B2 unauthorized", profile="offline-stdlib-not-an-agent",
                       identity=identity.copy(), checks=copy.deepcopy(result["checks"]), selection=copy.deepcopy(result["selection"]),
                       execution=result["execution"], review=None, findings=[],
                       allowed_effects=["synthetic fixture files and owned check process only"],
                       next_action="reconcile this observation; receipt grants no authority to rerun or enter B2")
        receipt["limits"].update(deadline_seconds=90, steps=1, transient_retries=0, paid_authorized=False, spend_usd=0)
        receipt["unknowns"] = {ptr: "no native/model/billing observation; deterministic fixture only"
                               for ptr in ("/session_id", "/model/provider", "/model/id", "/model/variant", "/cost_usd")}
        return receipt

    def validate(self, receipt, result, logs, *, identity=None, authority=AUTHORITY):
        rt.validate_receipt(receipt, workspace=self.workspace, identity=identity or self.identity(),
                            authorization_ref=authority, contract=self.contract, result=result,
                            evidence_root=self.evidence, log_digests=logs)

    def persist(self, observation):
        """Optional private retention; only generated fixture evidence, no native stores."""
        destination = os.environ.get("SDD_B8_EVIDENCE_ROOT")
        if not destination:
            return
        root = Path(destination)
        if (not root.is_absolute() or root.is_symlink() or root.resolve().is_relative_to(ROOT)
                or root.stat().st_mode & 0o077):
            raise ValueError("private external B8 evidence root required")
        target = root / self._testMethodName
        target.mkdir(mode=0o700)
        shutil.copytree(self.evidence, target / "evidence")
        observation.update(kind="actual deterministic component integration; not agent acceptance",
                           native_runs=0, model_requests=0, authority=AUTHORITY,
                           reuse="fixture workspaces removed; re-observe workspace and metadata before reuse")
        (target / "observation.json").write_text(json.dumps(observation, indent=2) + "\n")
        for entry in target.rglob("*"):
            entry.chmod(0o700 if entry.is_dir() else 0o600)

    def test_result_receipt_freshness_authority_and_location_reconciliation(self):
        self.define_contract()
        original_identity = self.identity()
        spec, tasks = [(self.workspace / name).read_bytes() for name in ("spec.md", "tasks.md")]
        result, logs = self.execute("initial")
        self.assertEqual("completed", result["execution"])
        self.assertEqual(original_identity, self.identity(), "read-only check inputs must remain fresh")
        receipt = self.receipt(result, original_identity)
        self.validate(receipt, result, logs)
        saved = self.evidence / "receipt-initial.json"
        rt.export_receipt(receipt, saved, workspace=self.workspace)
        reloaded = json.loads(saved.read_text())
        self.validate(reloaded, result, logs)
        self.assertIsNone(reloaded["review"], "execution is not engineering review")
        for authority in (None, "synthetic different B2 authorization"):
            with self.subTest(authority=authority), self.assertRaises(ValueError):
                self.validate(reloaded, result, logs, authority=authority)

        # Relevant untracked bytes invalidate a previously completed result.
        (self.workspace / "new-input.txt").write_text("relevant untracked input\n")
        with self.assertRaisesRegex(ValueError, "identity|fresh"):
            self.validate(reloaded, result, logs)
        moved = self.case / "relocated-project"
        self.workspace.rename(moved)
        self.workspace = moved
        with self.assertRaisesRegex(ValueError, "workspace|location"):
            self.validate(reloaded, result, logs)
        # Independent caller supplies fresh location, identity AND an actual rerun.
        current_identity = self.identity()
        fresh_result, fresh_logs = self.execute("after-reconciliation")
        fresh = self.receipt(fresh_result, current_identity)
        self.validate(fresh, fresh_result, fresh_logs)
        self.assertEqual(current_identity, self.identity())
        rt.export_receipt(fresh, self.evidence / "receipt-reconciled.json", workspace=self.workspace)
        self.assertEqual(spec, (self.workspace / "spec.md").read_bytes())
        self.assertEqual(tasks, (self.workspace / "tasks.md").read_bytes())
        self.assertFalse((self.workspace / "B2").exists())
        self.persist(dict(initial_execution=result["execution"], reconciled_execution=fresh_result["execution"],
                          rejected=["missing or changed authority", "untracked input staleness", "moved workspace"],
                          canonical_sentinels_unchanged=True))

    def test_actual_interruption_receipts_stay_partial_after_reload(self):
        self.define_contract(interrupted=True)
        sentinel = (self.workspace / "tasks.md").read_bytes()
        observations = []
        # Two repetitions per critical deterministic boundary, NOT reliability
        # samples of an agent cancelling/resuming native model work.
        for interrupt in (True, False):
            for repetition in (1, 2):
                with self.subTest(interrupt=interrupt, repetition=repetition):
                    partial = self.workspace / "partial.txt"
                    if partial.exists():
                        partial.unlink()  # Only this fixture-owned marker.
                    result, logs = self.execute(f"{'signal' if interrupt else 'deadline'}-{repetition}",
                                                interrupt=interrupt, deadline="10s" if interrupt else "1s")
                    self.assertTrue(partial.exists(), "target process must reach a partial effect")
                    expected = "cancelled" if interrupt else "incomplete"
                    self.assertEqual(expected, result["execution"])
                    self.assertEqual([expected, "not_run"], [ch["status"] for ch in result["checks"]])
                    self.assertFalse((self.workspace / "must-not-run.txt").exists())
                    identity = self.identity()
                    receipt = self.receipt(result, identity)
                    self.validate(receipt, result, logs)
                    saved = self.evidence / f"receipt-{'signal' if interrupt else 'deadline'}-{repetition}.json"
                    rt.export_receipt(receipt, saved, workspace=self.workspace)
                    reloaded = json.loads(saved.read_text())
                    self.validate(reloaded, result, logs)
                    for mutation in (dict(execution="completed"), dict(review="Ready")):
                        forged = copy.deepcopy(reloaded)
                        forged.update(mutation)
                        with self.assertRaises(ValueError):
                            self.validate(forged, result, logs)
                    self.assertEqual(sentinel, (self.workspace / "tasks.md").read_bytes())
                    self.assertEqual(identity, self.identity(), "receipt consumption must not mutate partial source or task state")
                    observations.append(dict(interrupt=interrupt, repetition=repetition, execution=expected,
                                             pending_check="not_run", partial_preserved=True, promotion_rejected=True))
        self.persist(dict(attempts=observations, canonical_sentinels_unchanged=True))

    def test_executed_defect_and_corrected_checks_do_not_waive_open_findings(self):
        self.define_contract()
        sentinel = (self.workspace / "tasks.md").read_bytes()
        code = self.workspace / "code.txt"
        code.write_text("deliberately wrong fixture bytes\n")
        failed_result, failed_logs = self.execute("known-defect")
        self.assertEqual("failed", failed_result["execution"])
        self.assertEqual(["failed", "not_run"], [ch["status"] for ch in failed_result["checks"]])
        failed_receipt = self.receipt(failed_result, self.identity())
        self.validate(failed_receipt, failed_result, failed_logs)
        rt.export_receipt(failed_receipt, self.evidence / "receipt-known-defect.json", workspace=self.workspace)
        # This is an independently specified fixture correction, not a model fix.
        code.write_text("independent expected bytes\n")
        fixed_result, fixed_logs = self.execute("corrected")
        self.assertEqual("completed", fixed_result["execution"])
        fixed_receipt = self.receipt(fixed_result, self.identity())
        fixed_receipt.update(review="Blocked", findings=[dict(id="FIXTURE-UPSTREAM", location="synthetic native boundary",
            impact="green component checks cannot resolve unobserved native integration", evidence="no native run in this fixture",
            disposition="needs evidence")])
        self.validate(fixed_receipt, fixed_result, fixed_logs)
        rt.export_receipt(fixed_receipt, self.evidence / "receipt-corrected-blocked.json", workspace=self.workspace)
        fixed_receipt["review"] = "Ready"
        with self.assertRaisesRegex(ValueError, "Ready|findings"):
            self.validate(fixed_receipt, fixed_result, fixed_logs)
        self.assertEqual(sentinel, (self.workspace / "tasks.md").read_bytes())
        self.persist(dict(defect_execution=failed_result["execution"], corrected_execution=fixed_result["execution"],
                          technical_verdict="Blocked", open_finding="FIXTURE-UPSTREAM",
                          premature_ready_rejected=True, canonical_sentinels_unchanged=True))


if __name__ == "__main__":
    unittest.main()
