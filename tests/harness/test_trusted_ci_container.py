"""Explicit Docker acceptance fixtures. Never automatic model/native calls."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import signal
import time
import unittest

import trusted_ci as gate


ROOT = Path(__file__).resolve().parents[2]
BASE = "1b50933193c622197eeab45fa0a8005b425d76b9"


@unittest.skipUnless(os.environ.get("SDD_TRUSTED_CI_CONTAINERS") == "1", "explicit container fixture authorization/opt-in required")
class TrustedContainerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp.cleanup)
        cls.root = Path(cls.temp.name)
        cls.base = cls.root / "frozen-b4"; cls.base.mkdir()
        archive = cls.root / "source.tar"
        with archive.open("wb") as stream:
            subprocess.run(["git", "archive", BASE], cwd=ROOT, stdout=stream, check=True, timeout=30)
        with tarfile.open(archive) as source:
            for entry in source:
                path = cls.base / entry.name
                if not path.is_relative_to(cls.base) or ".." in Path(entry.name).parts:
                    raise ValueError("unsafe frozen source")
                if entry.isdir():
                    path.mkdir(parents=True, exist_ok=True)
                elif entry.isfile():
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_bytes(source.extractfile(entry).read())
                else:
                    raise ValueError("special file in frozen fixture")
        subprocess.run(["docker", "image", "inspect", gate.IMAGE], check=True, capture_output=True, timeout=15)

    def setUp(self):
        self.temp_case = tempfile.TemporaryDirectory(dir=self.root)
        self.addCleanup(self.temp_case.cleanup)
        self.case = Path(self.temp_case.name)
        self.trusted, self.candidate = self.case / "trusted", self.case / "candidate"
        shutil.copytree(self.base, self.trusted); shutil.copytree(self.base, self.candidate)
        # Explicit development reference, not an assertion B7 is already trusted
        # on main. CI adoption requires a separately reviewed merge/commit.
        for name in ("tests/harness/trusted_ci.py", "tests/harness/trusted-noop-probe.go.txt", "Makefile"):
            path = self.trusted / name; path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes((ROOT / name).read_bytes())
        self.env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(self.case),
                    "TMPDIR": str(self.case), "SDD_HOST_CANARY": "synthetic-host-env-not-for-candidate"}

    def invoke(self, name, local=False):
        output = self.case / name
        if local:
            argv = ["make", "-C", str(self.trusted), "verify-trusted", "SDD_TRUSTED=" + str(self.trusted),
                    "SDD_CANDIDATE=" + str(self.candidate), "SDD_OUTPUT=" + str(output)]
        else:
            argv = ["python3", "-B", str(self.trusted / "tests/harness/trusted_ci.py"), "run", "--trusted", str(self.trusted),
                    "--candidate", str(self.candidate), "--output", str(output)]
        result = subprocess.run(argv, env=self.env, capture_output=True, text=True, timeout=1100)
        summary = json.loads((output / "evidence/gate.json").read_text())
        self.persist(name, result.returncode, summary, output)
        return result, summary

    def persist(self, name, returncode, summary, output=None):
        destination = os.environ.get("SDD_CI_EVIDENCE_ROOT")
        if not destination:
            return
        root = Path(destination)
        if not root.is_absolute() or root.is_symlink() or root.stat().st_mode & 0o077 or root.resolve().is_relative_to(ROOT):
            raise ValueError("explicit private external fixture evidence root required")
        retained = root / self._testMethodName / name
        retained.mkdir(mode=0o700, parents=True)
        if output is not None:
            shutil.copytree(output / "evidence", retained / "evidence")
        observation = dict(kind="deterministic-ci-fixture-not-native-evaluation", actual_exit=returncode,
                           execution=summary["execution"], fixture=self._testMethodName,
                           development_reference="frozen B4 plus explicitly supplied B7 implementation; not deployed trust",
                           additional_model_requests=0, retained_logs="evidence/checks/*.log" if output else None)
        with (retained / "observation.json").open("x") as stream:
            json.dump(observation, stream, indent=2); stream.write("\n")
        for entry in retained.rglob("*"):
            entry.chmod(0o700 if entry.is_dir() else 0o600)

    def test_bootstrap_and_candidate_contract_weakening_block_before_execution(self):
        probe = self.trusted / "tests/harness/trusted-noop-probe.go.txt"
        probe.unlink()
        result, summary = self.invoke("missing-bootstrap")
        self.assertEqual(2, result.returncode)
        self.assertEqual("blocked", summary["execution"])
        self.assertNotIn("runner_sha256", summary)
        probe.write_bytes((ROOT / "tests/harness/trusted-noop-probe.go.txt").read_bytes())
        contract = self.candidate / "harness/checks.json"
        data = json.loads(contract.read_text()); data["checks"] = data["checks"][1:]
        contract.write_text(json.dumps(data))
        result, summary = self.invoke("removed-gate")
        self.assertEqual(2, result.returncode)
        self.assertNotIn("check_result", summary)

    def test_local_and_ci_entries_agree_and_candidate_boundary_is_observed(self):
        secret = self.case / "host-sentinel"; secret.write_text("synthetic-host-file-not-for-candidate")
        source = '''package skillsync_test
import ("net"; "os"; "testing"; "time")
func TestSyntheticCandidateBoundary(t *testing.T) {
 if os.Getuid()==0 { t.Fatal("root candidate") }
 if os.Getenv("SDD_HOST_CANARY")!="" || os.Getenv("GITHUB_TOKEN")!="" { t.Fatal("host environment leaked") }
 if _,err:=os.ReadFile(HOST_PATH); err==nil { t.Fatal("host file mounted") }
 if _,err:=os.Stat("/var/run/docker.sock"); err==nil { t.Fatal("Docker socket mounted") }
 if err:=os.WriteFile("/sdd-root-write",[]byte("x"),0600); err==nil { t.Fatal("root writable") }
 conn,err:=net.DialTimeout("tcp","1.1.1.1:443",200*time.Millisecond)
 if err==nil { conn.Close(); t.Fatal("unexpected outbound network") }
}
'''.replace("HOST_PATH", json.dumps(str(secret)))
        (self.candidate / "internal/skillsync/boundary_test.go").write_text(source)
        # A Ready claim is deliberately present but is never a gate input.
        (self.candidate / "receipt.json").write_text('{"review":"Ready","execution":"completed"}')
        local_result, local = self.invoke("local", local=True)
        ci_result, ci = self.invoke("ci")
        self.assertEqual(0, local_result.returncode, local)
        self.assertEqual(0, ci_result.returncode, ci)
        for summary in (local, ci):
            self.assertEqual(gate.REQUIRED, [c["id"] for c in summary["check_result"]["checks"]])
            self.assertTrue(all(c["status"] == "passed" for c in summary["check_result"]["checks"]))
            self.assertIn("internal/skillsync/boundary_test.go", summary["review_required_paths"])
        self.assertEqual(local["canonical_contract_sha256"], ci["canonical_contract_sha256"])
        self.assertEqual(local["candidate_input_sha256"], ci["candidate_input_sha256"])

    def test_empty_make_deleted_coverage_and_forged_receipt_cannot_hide_noop_defect(self):
        path = self.candidate / "internal/skillsync/sync.go"
        data = path.read_text(); self.assertEqual(1, data.count("if same {"))
        path.write_text(data.replace("if same {", "if false && same {"))
        for test in self.candidate.rglob("*_test.go"):
            test.unlink()  # Only this fixture's disposable source, never the checkout.
        (self.candidate / "Makefile").write_text("test:\n\t@true\n")
        subprocess.run(["make", "test"], cwd=self.candidate, env=self.env, check=True, capture_output=True, timeout=10)
        (self.candidate / "receipt.json").write_text('{"review":"Ready","execution":"completed"}')
        result, summary = self.invoke("weakened")
        self.assertNotEqual(0, result.returncode)
        self.assertEqual("failed", summary["execution"], summary)
        self.assertIn("Makefile", summary["review_required_paths"])
        checks = {c["id"]: c for c in summary["check_result"]["checks"]}
        self.assertEqual("failed", checks["test"]["status"])
        self.assertIn("independent no-op contract", Path(checks["test"]["evidence"]).read_text())
        self.assertEqual("not_run", checks["build-darwin"]["status"])

    def test_cleanup_stops_only_cid_owned_container(self):
        env = gate.safe_environment(self.case)
        files = [self.case / "one.cid", self.case / "two.cid"]
        try:
            for path in files:
                subprocess.run(["docker", "run", "-d", "--rm", "--pull=never", "--network=none", "--read-only",
                                "--cap-drop=ALL", "--security-opt=no-new-privileges", f"--user={os.getuid()}:{os.getgid()}",
                                "--cidfile", str(path), gate.IMAGE, "sleep", "60"],
                               env=env, check=True, capture_output=True, timeout=15)
            self.assertFalse(gate.cleanup_containers(files[:1], env))
            cid = files[1].read_text().strip()
            observed = subprocess.check_output(["docker", "inspect", "--format", "{{.State.Running}}", cid], env=env, timeout=10)
            self.assertEqual(b"true", observed.strip())
            self.persist("owned-cleanup", 0, {"execution": "completed"})
        finally:
            self.assertFalse(gate.cleanup_containers(files, env))

    def test_interrupted_gate_cleans_owned_containers_and_retains_partial_candidate(self):
        (self.candidate / "internal/skillsync/cancel_test.go").write_text('''package skillsync_test
import ("os"; "testing"; "time")
func TestSDDWaitingFixture(t *testing.T) {
 if err:=os.WriteFile("sdd-cancel-marker",[]byte("partial"),0600);err!=nil {t.Fatal(err)}
 time.Sleep(60*time.Second)
}
''')
        output = self.case / "interrupted"
        process = subprocess.Popen(["python3", "-B", str(self.trusted / "tests/harness/trusted_ci.py"), "run",
                                    "--trusted", str(self.trusted), "--candidate", str(self.candidate), "--output", str(output)],
                                   env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        marker = output / "private/candidate/internal/skillsync/sdd-cancel-marker"
        try:
            deadline = time.monotonic() + 150
            while not marker.exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(0.1)
            self.assertTrue(marker.exists(), "target check must actually start before cancellation")
            process.send_signal(signal.SIGTERM)
            process.communicate(timeout=40)
            self.assertEqual(1, process.returncode)
            summary = json.loads((output / "evidence/gate.json").read_text())
            self.assertEqual("cancelled", summary["execution"], summary)
            self.assertEqual(b"partial", marker.read_bytes())
            for path in (output / "private").glob("*.cid"):
                cid = path.read_text().strip()
                observed = subprocess.run(["docker", "inspect", "--format", "{{.State.Running}}", cid],
                                          env=gate.safe_environment(self.case), capture_output=True, timeout=10)
                self.assertNotEqual(b"true", observed.stdout.strip(), "owned container remained active")
            self.persist("interrupted", process.returncode, summary, output)
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                process.communicate(timeout=40)
            self.assertFalse(gate.cleanup_containers(list((output / "private").glob("*.cid")), gate.safe_environment(self.case)))


if __name__ == "__main__":
    unittest.main()
