"""B4 regressions for the vendored offline nested_tui qualification lane."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

HARNESS_DIR = Path(__file__).resolve().parent
if str(HARNESS_DIR) not in sys.path:
    sys.path.insert(0, str(HARNESS_DIR))

import trusted_ci as gate


ROOT = Path(__file__).resolve().parents[2]
TUI_DIR = ROOT / "experiments/tui"


def parse_go_mod(text: str) -> tuple[dict[str, str], dict[str, str]]:
    """Return ({module: version}, {module: replace_target}) from go.mod text."""
    requires: dict[str, str] = {}
    replaces: dict[str, str] = {}
    in_require_block = False
    in_replace_block = False

    for raw in text.splitlines():
        line = raw.split("//", 1)[0].strip()
        if not line:
            continue
        if in_require_block:
            if line == ")":
                in_require_block = False
                continue
            parts = line.split()
            if len(parts) >= 2:
                requires[parts[0]] = parts[1]
            continue
        if in_replace_block:
            if line == ")":
                in_replace_block = False
                continue
            if "=>" in line:
                left, right = [side.strip() for side in line.split("=>", 1)]
                mod = left.split()[0]
                replaces[mod] = right
            continue
        if line == "require (":
            in_require_block = True
            continue
        if line == "replace (":
            in_replace_block = True
            continue
        if line.startswith("require "):
            parts = line[len("require "):].strip().split()
            if len(parts) >= 2:
                requires[parts[0]] = parts[1]
            continue
        if line.startswith("replace ") and "=>" in line:
            body = line[len("replace "):].strip()
            left, right = [side.strip() for side in body.split("=>", 1)]
            mod = left.split()[0]
            replaces[mod] = right

    return requires, replaces


def parse_vendor_modules_txt(text: str) -> tuple[dict[str, str], dict[str, str], dict[str, bool], dict[str, list[str]]]:
    """Return (modules, replaces, explicit_flags, packages) from vendor/modules.txt."""
    modules: dict[str, str] = {}
    replaces: dict[str, str] = {}
    explicit_flags: dict[str, bool] = {}
    packages: dict[str, list[str]] = {}
    current_mod: str | None = None

    for raw in text.splitlines():
        line = raw.strip()
        if not line:
            continue
        if line.startswith("##"):
            if current_mod is not None and "explicit" in line:
                explicit_flags[current_mod] = True
            continue
        if line.startswith("# "):
            body = line[2:].strip()
            if "=>" in body:
                left, right = [side.strip() for side in body.split("=>", 1)]
                parts = left.split()
                if len(parts) == 1:
                    replaces[parts[0]] = right
                    current_mod = None
                elif len(parts) >= 2:
                    mod, ver = parts[0], parts[1]
                    modules[mod] = ver
                    replaces[mod] = right
                    explicit_flags.setdefault(mod, False)
                    packages.setdefault(mod, [])
                    current_mod = mod
                continue
            parts = body.split()
            if len(parts) >= 2:
                mod, ver = parts[0], parts[1]
                modules[mod] = ver
                explicit_flags.setdefault(mod, False)
                packages.setdefault(mod, [])
                current_mod = mod
            continue
        if current_mod is not None:
            packages[current_mod].append(line)

    return modules, replaces, explicit_flags, packages


def compare_go_mod_and_vendor(go_mod_text: str, modules_txt: str) -> list[str]:
    """Return list of mismatches between go.mod and vendor/modules.txt."""
    errors: list[str] = []
    req_mods, mod_replaces = parse_go_mod(go_mod_text)
    ven_mods, ven_replaces, explicit_flags, packages = parse_vendor_modules_txt(modules_txt)

    if req_mods != ven_mods:
        missing = sorted(set(req_mods) - set(ven_mods))
        extra = sorted(set(ven_mods) - set(req_mods))
        changed = sorted(
            m for m in set(req_mods) & set(ven_mods) if req_mods[m] != ven_mods[m]
        )
        if missing:
            errors.append(f"missing in vendor/modules.txt: {missing}")
        if extra:
            errors.append(f"extra in vendor/modules.txt: {extra}")
        for m in changed:
            errors.append(f"version mismatch for {m}: go.mod={req_mods[m]} vendor={ven_mods[m]}")

    if mod_replaces != ven_replaces:
        errors.append(f"replace mismatch: go.mod={mod_replaces} vendor={ven_replaces}")

    for mod in ven_mods:
        if not explicit_flags.get(mod):
            errors.append(f"module {mod} missing '## explicit' in vendor/modules.txt")
        if not packages.get(mod):
            errors.append(f"module {mod} has no vendored packages listed in vendor/modules.txt")

    return errors


class NestedTuiLaneDefinitionTests(unittest.TestCase):
    def setUp(self):
        self.q_path = ROOT / "harness/qualification.json"
        self.qualification = gate.load_json(self.q_path)
        self.checks = gate.load_json(ROOT / "harness/checks.json")

    def test_nested_tui_lane_is_active_and_uses_mod_vendor(self):
        lanes = self.qualification.get("lanes", {})
        self.assertIn("nested_tui", lanes)
        lane = lanes["nested_tui"]
        self.assertEqual(lane.get("status"), "active")
        self.assertNotIn("blocker", lane)
        self.assertEqual(lane.get("cwd"), "experiments/tui")

        checks = lane.get("checks")
        self.assertIsInstance(checks, list)
        self.assertGreaterEqual(len(checks), 3)
        by_id = {ch.get("id"): ch for ch in checks}
        self.assertEqual(len(by_id), len(checks), "duplicate check id in nested_tui")
        self.assertEqual(
            [ch.get("id") for ch in checks],
            ["tui-vendor-check", "tui-vet", "tui-test"],
        )

        for ch in checks:
            argv = ch.get("argv")
            self.assertIsInstance(argv, list)
            self.assertEqual(argv[0], "go")
            self.assertIn("-mod=vendor", argv, f"{ch.get('id')} must enforce -mod=vendor")
            self.assertNotEqual(argv[:3], ["go", "mod", "verify"])

        self.assertEqual(by_id["tui-vendor-check"]["argv"], ["go", "list", "-mod=vendor", "./..."])
        self.assertEqual(by_id["tui-vet"]["argv"], ["go", "vet", "-mod=vendor", "./..."])
        self.assertEqual(
            by_id["tui-test"]["argv"],
            ["go", "test", "-mod=vendor", "-count=1", "-timeout=15m", "./..."],
        )

    def test_no_broad_python_or_root_go_discovery_is_used(self):
        py_lane = self.qualification["lanes"]["python_suites"]
        self.assertEqual(py_lane.get("status"), "active")
        self.assertEqual(py_lane.get("discovery_policy"), "individually-named-audited-suites-only")

        for suite in py_lane["suites"]:
            suite_file = Path(suite["path"]).name
            self.assertRegex(suite_file, r"^test_[a-z0-9_]+\.py$")
            for ch in ("*", "?", "["):
                self.assertNotIn(ch, suite_file)
            self.assertEqual(
                suite["argv"],
                ["python3", "-B", "-m", "unittest", "discover", "-s", "tests/harness", "-p", suite_file],
            )
            self.assertNotIn("test_b4_", " ".join(suite["argv"]))

        # Root checks contract must not use broad ./... on the root module.
        for ch in self.checks["checks"]:
            if ch["id"] in ("vet", "test"):
                for step in ch["steps"]:
                    self.assertNotIn("./...", step["argv"])


class NestedTuiVendorSyncTests(unittest.TestCase):
    def test_vendor_directory_has_no_symlinks_and_has_modules_txt(self):
        vendor_dir = TUI_DIR / "vendor"
        modules_txt = vendor_dir / "modules.txt"
        self.assertTrue(vendor_dir.is_dir() and not vendor_dir.is_symlink())
        self.assertTrue(modules_txt.is_file() and not modules_txt.is_symlink())

        for dirpath, dirnames, filenames in os.walk(vendor_dir, followlinks=False):
            base = Path(dirpath)
            for name in dirnames + filenames:
                p = base / name
                self.assertFalse(p.is_symlink(), f"symlink forbidden in vendor tree: {p}")

    def test_vendor_modules_txt_matches_go_mod(self):
        go_mod_text = (TUI_DIR / "go.mod").read_text(encoding="utf-8")
        modules_txt = (TUI_DIR / "vendor/modules.txt").read_text(encoding="utf-8")
        errors = compare_go_mod_and_vendor(go_mod_text, modules_txt)
        self.assertEqual(errors, [], f"vendor/modules.txt out of sync with go.mod: {errors}")

    def test_compare_helper_rejects_mismatched_version_or_missing_module(self):
        go_mod_text = (TUI_DIR / "go.mod").read_text(encoding="utf-8")
        modules_txt = (TUI_DIR / "vendor/modules.txt").read_text(encoding="utf-8")

        mutated_ver = modules_txt.replace("charm.land/bubbles/v2 v2.1.0", "charm.land/bubbles/v2 v2.0.0", 1)
        self.assertTrue(any("version mismatch" in e for e in compare_go_mod_and_vendor(go_mod_text, mutated_ver)))

        mutated_missing = re.sub(
            r"# github\.com/sahilm/fuzzy v0\.1\.1\n## explicit\ngithub\.com/sahilm/fuzzy\n",
            "",
            modules_txt,
        )
        self.assertTrue(any("missing in vendor/modules.txt" in e for e in compare_go_mod_and_vendor(go_mod_text, mutated_missing)))

    def test_go_list_and_vet_with_mod_vendor_succeed_offline_with_empty_modcache(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            home = root / "home"
            tmpdir = root / "tmp"
            gocache = root / "gocache"
            empty_modcache = root / "empty-modcache"
            for d in (home, tmpdir, gocache, empty_modcache):
                d.mkdir(mode=0o700)

            env = {
                "PATH": os.environ.get("PATH", "/usr/local/bin:/usr/bin:/bin"),
                "HOME": str(home),
                "TMPDIR": str(tmpdir),
                "GOCACHE": str(gocache),
                "GOMODCACHE": str(empty_modcache),
                "GOTOOLCHAIN": "local",
                "GOPROXY": "off",
                "GOSUMDB": "off",
                "GOENV": "off",
                "GOWORK": "off",
                "GOFLAGS": "",
            }

            list_res = subprocess.run(
                ["go", "list", "-mod=vendor", "./..."],
                cwd=TUI_DIR,
                env=env,
                capture_output=True,
                text=True,
                timeout=60,
            )
            self.assertEqual(list_res.returncode, 0, list_res.stdout + list_res.stderr)
            self.assertIn("agentic-sdd/experiments/tui/cmd/tui-companion", list_res.stdout.splitlines())

            vet_res = subprocess.run(
                ["go", "vet", "-mod=vendor", "./..."],
                cwd=TUI_DIR,
                env=env,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertEqual(vet_res.returncode, 0, vet_res.stdout + vet_res.stderr)


if __name__ == "__main__":
    unittest.main()
