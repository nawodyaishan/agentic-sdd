"""Trusted-gate policy tests; no provider/native state or container required."""
import copy
import json
import os
from pathlib import Path
import tempfile
import unittest
import subprocess

import trusted_ci as gate


ROOT = Path(__file__).resolve().parents[2]


class TrustedPolicyTests(unittest.TestCase):
    def setUp(self):
        self.contract = json.loads((ROOT / "harness/checks.json").read_text())

    def test_canonical_required_commands_are_retained(self):
        selected = gate.required_contract(self.contract, copy.deepcopy(self.contract))
        self.assertEqual(["mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"],
                         [ch["id"] for ch in selected])
        self.assertEqual(["go", "test", "./..."], selected[3]["steps"][0]["argv"])

    def test_removed_replaced_or_relabelled_required_gate_is_rejected(self):
        for change in ("remove", "command", "scope", "extra", "duplicate"):
            with self.subTest(change=change):
                candidate = copy.deepcopy(self.contract)
                if change == "remove":
                    candidate["checks"] = candidate["checks"][1:]
                elif change == "command":
                    candidate["checks"][3]["steps"][0]["argv"] = ["true"]
                elif change == "scope":
                    candidate["checks"][3]["scopes"] = ["focused"]
                elif change == "extra":
                    item = copy.deepcopy(candidate["checks"][3]); item["id"] = "new-unreviewed-gate"
                    candidate["checks"].append(item)
                else:
                    candidate["checks"].append(copy.deepcopy(candidate["checks"][3]))
                with self.assertRaises(ValueError):
                    gate.required_contract(self.contract, candidate)

    def test_nonroot_cwd_is_blocked_instead_of_silently_reinterpreted(self):
        changed = copy.deepcopy(self.contract)
        changed["checks"][3]["cwd"] = "internal/skillsync"
        with self.assertRaises(ValueError):
            gate.required_contract(changed, changed)

    def test_nonzero_transport_or_missing_mismatched_result_cannot_be_completed(self):
        result = dict(execution="completed", checks=[dict(id=name, status="passed", exit_code=0, truncated=False) for name in gate.REQUIRED])
        self.assertEqual("completed", gate.execution_outcome(0, result, result))
        self.assertNotEqual("completed", gate.execution_outcome(1, result, result))
        self.assertNotEqual("completed", gate.execution_outcome(0, result, None))
        changed = copy.deepcopy(result); changed["checks"][0]["status"] = "failed"
        self.assertNotEqual("completed", gate.execution_outcome(0, result, changed))
        changed = copy.deepcopy(result); changed["checks"].pop()
        self.assertNotEqual("completed", gate.execution_outcome(0, changed, changed))

    def test_test_and_gate_changes_are_review_inputs_not_silently_approved(self):
        before = {p: "old" for p in ["Makefile", "internal/skillsync/sync_test.go", "code.go", "receipt.json"]}
        after = {"Makefile": "empty", "code.go": "new", "receipt.json": "Ready"}
        self.assertEqual(["Makefile", "internal/skillsync/sync_test.go"], gate.review_paths(before, after))

    def test_revision_observation_is_actual_root_head_not_an_export_or_parent_guess(self):
        with tempfile.TemporaryDirectory() as tmp:
            env = gate.safe_environment(Path(tmp))
            expected = subprocess.check_output(["git", "-C", str(ROOT), "rev-parse", "--verify", "HEAD"], env=env, timeout=10).decode().strip()
            self.assertEqual(expected, gate.observed_revision(ROOT, env))
            self.assertIsNone(gate.observed_revision(ROOT / "internal", env))
            self.assertIsNone(gate.observed_revision(Path(tmp), env))

    def test_staging_includes_untracked_bytes_but_no_git_cache_or_backups(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, dest = Path(tmp) / "source", Path(tmp) / "copy"
            source.mkdir()
            for name in ["source.go", "new_test.go", ".git/config", ".cache/private", "backups/private", "bin/output"]:
                path = source / name; path.parent.mkdir(parents=True, exist_ok=True); path.write_text(name)
            manifest = gate.stage_tree(source, dest)
            self.assertEqual({"source.go", "new_test.go"}, set(manifest))
            self.assertEqual("new_test.go", (dest / "new_test.go").read_text())
            self.assertFalse((dest / ".git").exists())

    def test_staging_rejects_links_and_special_files_without_following_them(self):
        for kind in ("symlink", "fifo"):
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp); source = root / "source"; source.mkdir()
                if kind == "symlink":
                    (source / "unsafe").symlink_to(root / "outside")
                else:
                    os.mkfifo(source / "unsafe")
                with self.assertRaises(ValueError):
                    gate.stage_tree(source, root / "copy")

    def test_staging_refuses_credential_paths_instead_of_copying_local_secrets(self):
        for name in (".env", "auth.json", "opencode.db", ".local/share/opencode/store", ".ssh/key"):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp); source = root / "source"; source.mkdir()
                path = source / name; path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("SYNTHETIC-CREDENTIAL-CANARY")
                with self.assertRaises(ValueError):
                    gate.stage_tree(source, root / "copy")
                self.assertFalse((root / "copy").exists())

    def test_candidate_container_gets_only_its_workspace_and_private_check_cache(self):
        argv = gate.container_argv(Path("/safe/work"), Path("/safe/cache"), Path("/safe/check.cid"), ["go", "test", "./..."], {})
        for value in ["--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pull=never"]:
            self.assertIn(value, argv)
        self.assertIn(f"--user={os.getuid()}:{os.getgid()}", argv)
        self.assertEqual(2, argv.count("--mount"))
        self.assertNotIn("/var/run/docker.sock", " ".join(argv))
        self.assertNotIn("HOME=" + str(Path.home()), " ".join(argv))
        self.assertEqual(["go", "test", "./..."], argv[-3:])
        with self.assertRaises(ValueError):
            gate.container_argv(Path("/safe/work"), Path("/safe/cache"), Path("/safe/check.cid"), ["go", "test", "./..."], {"PATH": "/workspace"})

    def test_gate_does_not_need_candidate_receipts_to_determine_check_outcome(self):
        # No schema/metadata claim in an arbitrary receipt is an execution input.
        selected = gate.required_contract(self.contract, self.contract)
        self.assertNotIn("receipt.json", json.dumps(selected))

    def test_whitespace_checks_source_patch_not_generated_build_artifacts(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); base, candidate = root / "base", root / "candidate"
            base.mkdir(); candidate.mkdir()
            (base / "code.txt").write_text("base\n")
            (candidate / "code.txt").write_text("changed\n")
            (candidate / "bin").mkdir(); (candidate / "bin/generated").write_text("generated output \n")
            self.assertEqual(0, gate.whitespace(base, candidate))
            (candidate / "code.txt").write_text("bad source \n")
            self.assertEqual(1, gate.whitespace(base, candidate))


if __name__ == "__main__":
    unittest.main()
