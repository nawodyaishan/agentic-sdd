"""Offline host-driver cleanup/evidence tests; never start Docker or OpenCode."""
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest


spec = importlib.util.spec_from_file_location("p1_driver", Path(__file__).with_name("p1-preflight.py"))
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


class DriverTests(unittest.TestCase):
    def test_policy_timeout_still_removes_exact_cid(self):
        with tempfile.TemporaryDirectory() as tmp:
            cidfile = Path(tmp) / "policy.cid"
            calls = []
            def docker(*args, **options):
                calls.append(args)
                if args[0] == "run":
                    cidfile.write_text("a" * 64)
                    raise subprocess.TimeoutExpired("docker", 45)
                return subprocess.CompletedProcess(args, 0, b"", b"")
            with self.assertRaises(subprocess.TimeoutExpired):
                driver.run_policy(docker, ["run"], cidfile, Path(tmp))
            self.assertEqual(("rm", "--force", "a" * 64), calls[-1])

    def test_bad_cid_is_not_removed(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "policy.cid"; path.write_text("untrusted-name")
            calls = []
            self.assertFalse(driver.remove_container(lambda *a, **k: calls.append(a), path))
            self.assertEqual([], calls)

    def test_changed_source_is_recorded_as_failed_not_success(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp)
            result = dict(completed=True, source_identity="before")
            driver.write_observation(output, result, "after")
            self.assertFalse(json.loads((output / "observation.json").read_text())["completed"])
            self.assertTrue(result["source_changed"])


if __name__ == "__main__":
    unittest.main()
