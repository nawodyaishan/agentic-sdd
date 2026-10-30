"""Offline artifact checks. These do not certify native enforcement."""
import json
from pathlib import Path
import hashlib
import os
import stat
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
ADAPTER = ROOT / "integrations/opencode"


class TemplateTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue((ADAPTER / "opencode.jsonc").is_file(), "inactive native template is missing")
        self.config = json.loads((ADAPTER / "opencode.jsonc").read_text())

    def test_no_implicit_provider_or_integration_access(self):
        self.assertIn({"action": "provider.use", "resource": "*", "effect": "deny"}, self.config["policies"])
        self.assertFalse(any(p["action"] == "provider.use" and p["effect"] == "allow" for p in self.config["policies"]))
        self.assertFalse(any(p["action"] == "integration.use" and p["resource"].startswith(("plugin:", "mcp:")) and p["effect"] == "allow" for p in self.config["policies"]))
        for kind in ("mcp", "plugin"):
            self.assertIn({"action": "integration.use", "resource": kind + ":*", "effect": "deny"}, self.config["policies"])
        self.assertFalse(self.config["warming"])
        for key in ("providers", "plugins", "mcp", "references", "instructions"):
            self.assertNotIn(key, self.config)

    def test_commands_cannot_inherit_a_model_or_launch_children(self):
        for name, command in self.config["commands"].items():
            self.assertTrue(name.startswith("sdd-"))
            self.assertFalse(command["subagent"])
            self.assertNotIn("!`", command["template"])
            self.assertEqual(command["model"], "opencode/deepseek-v4.1-flash-free")
            self.assertIn(command["agent"], self.config["agents"])

    def test_profiles_are_bounded_and_start_with_deny(self):
        self.assertEqual(set(self.config["agents"]), {"sdd-draft", "sdd-build", "sdd-review"})
        for agent in self.config["agents"].values():
            self.assertEqual(agent["mode"], "primary")
            self.assertEqual(agent["model"], "opencode/deepseek-v4.1-flash-free")
            self.assertLessEqual(agent["steps"], 20)
            self.assertEqual(agent["permissions"][0], {"action": "*", "resource": "*", "effect": "deny"})
        review = self.config["agents"]["sdd-review"]["permissions"]
        self.assertEqual({r["action"] for r in review if r["effect"] == "allow"}, {"read", "skill", "question"})
        edits = [r["resource"] for r in self.config["agents"]["sdd-draft"]["permissions"] if r["action"] == "edit" and r["effect"] == "allow"]
        self.assertEqual(edits, ["specs/__FEATURE__/spec.md", "specs/__FEATURE__/plan.md", "specs/__FEATURE__/tasks.md"])

    def test_higher_authority_lockdown_has_no_allow_exception(self):
        policy = json.loads((ADAPTER / "policy-lockdown.jsonc").read_text())
        self.assertEqual([p for p in policy["policies"] if p["effect"] == "allow"], [])
        self.assertIn({"action": "provider.use", "resource": "*", "effect": "deny"}, policy["policies"])

    def test_usage_catalog_is_not_permission_to_run(self):
        catalog_path = ADAPTER / "model-usage.json"
        self.assertTrue(catalog_path.is_file(), "model usage catalog is missing")
        catalog = json.loads(catalog_path.read_text())
        self.assertFalse(catalog["execution_enabled"])
        self.assertEqual(catalog["automatic_fallbacks"], [])
        selected = {item["role"]: item for item in catalog["models"]}
        self.assertEqual(selected["everyday"]["id"], "opencode/deepseek-v4.1-flash-free")
        self.assertEqual(selected["quick"]["id"], "opencode/glm-5.3-flash-free")
        self.assertEqual(selected["harder"]["id"], "openrouter/xiaomi/mimo-v2.6-pro")
        self.assertEqual(selected["experimental-reasoning"]["data_scope"], "synthetic-only")
        for model in catalog["models"]:
            self.assertFalse(model["capability_verified"])
            self.assertFalse(model["execution_enabled"])


class ManualAdoptionExercise(unittest.TestCase):
    """Exercise the documented operations in scratch, NOT a config installer.

    JSONC merge decisions remain human-owned; synthetic inputs are plain JSON.
    No test calls the real installer, runtime home or a provider.
    """

    def test_preview_backup_noop_restore_and_new_file_removal(self):
        with tempfile.TemporaryDirectory(prefix="sdd-adopt-") as tmp:
            root = Path(tmp)
            root.chmod(0o700)
            target = root / "project/.opencode/opencode.jsonc"
            target.parent.mkdir(parents=True)
            original = b'{"username":"unrelated","commands":{"team":{"template":"keep"}}}\n'
            target.write_bytes(original)
            # Manual review preserves unrelated fields; never replace the whole
            # existing document with the shipped template.
            proposed = json.loads(original)
            proposed["commands"].update(json.loads((ADAPTER / "opencode.jsonc").read_text())["commands"])
            staged = root / "staged.jsonc"
            staged.write_text(json.dumps(proposed))
            self.assertEqual(target.read_bytes(), original, "preview mutated destination")
            self.assertEqual(json.loads(staged.read_text())["username"], "unrelated")
            self.assertEqual(json.loads(staged.read_text())["commands"]["team"], {"template": "keep"})
            backup = root / "backup"
            backup.mkdir(mode=0o700)
            self.assertTrue(stat.S_ISREG(target.lstat().st_mode))
            (backup / "original").write_bytes(target.read_bytes())
            (backup / "original").chmod(0o600)
            before = hashlib.sha256(original).hexdigest()
            installed = hashlib.sha256(staged.read_bytes()).hexdigest()
            manifest = {"path": str(target), "existed": True, "original_sha256": before, "installed_sha256": installed}
            (backup / "manifest.json").write_text(json.dumps(manifest))
            (backup / "manifest.json").chmod(0o600)
            target.write_bytes(staged.read_bytes())
            # Identical proposal: neither rewrite nor another backup.
            stamp = target.stat().st_mtime_ns
            self.assertEqual(target.read_bytes(), staged.read_bytes())
            self.assertEqual(target.stat().st_mtime_ns, stamp)
            self.assertEqual(len(list(backup.iterdir())), 2)
            # Intervening edits make the rollback digest precondition false.
            target.write_text('{"username":"changed later"}')
            self.assertNotEqual(hashlib.sha256(target.read_bytes()).hexdigest(), installed)
            self.assertIn("changed later", target.read_text())
            # Resolve the intervening edit in this fixture only, then roll back.
            target.write_bytes(staged.read_bytes())
            self.assertEqual(hashlib.sha256(target.read_bytes()).hexdigest(), manifest["installed_sha256"])
            self.assertEqual(hashlib.sha256((backup / "original").read_bytes()).hexdigest(), manifest["original_sha256"])
            target.write_bytes((backup / "original").read_bytes())
            self.assertEqual(target.read_bytes(), original)
            fresh = root / "fresh.jsonc"
            self.assertFalse(fresh.exists())
            fresh.write_bytes(staged.read_bytes())
            self.assertEqual(hashlib.sha256(fresh.read_bytes()).hexdigest(), installed)
            fresh.unlink()  # Only this known fresh fixture file, after the check.
            self.assertFalse(fresh.exists())

    def test_symlink_parent_symlink_and_special_file_are_not_regular(self):
        with tempfile.TemporaryDirectory(prefix="sdd-adopt-") as tmp:
            root = Path(tmp)
            original = root / "original"
            original.write_text("preserve")
            link = root / "link"
            link.symlink_to(original)
            fifo = root / "fifo"
            os.mkfifo(fifo)
            self.assertFalse(stat.S_ISREG(link.lstat().st_mode))
            self.assertFalse(stat.S_ISREG(fifo.lstat().st_mode))
            parent = root / "parent"
            parent.symlink_to(root, target_is_directory=True)
            self.assertTrue(parent.is_symlink())
            self.assertEqual(original.read_text(), "preserve")


if __name__ == "__main__":
    unittest.main()
