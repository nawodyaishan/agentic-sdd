"""Static selector coverage for the additive `internal/appops` core package.

Feature 013 B2/T04 with D03 permission: `internal/appops` joins every maintained
supported-core selector without removing existing coverage or enabling excluded
005/006/007 selections. These checks read selector text only; they never build Go
and never require the package to exist in this checkout (a separate T04 lane
creates it), so a missing package directory is not counted as a pass or failure
of this file.

Selection guardrails asserted here:
- test/vet package lists in Taskfile, harness contract and core shell script agree;
- the shell script's required-directory loop lists the new package;
- the pre-existing seven packages stay selected;
- no `./...` aggregate, experimental build tag or excluded package is selected;
- package-count guidance in AGENTS.md and CONTRIBUTING.md says eight.
"""

import json
import re
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]

PRE_EXISTING = [
    ".",
    "./internal/version",
    "./internal/skillsync",
    "./internal/harnesscheck",
    "./internal/harnessevidence",
    "./cmd/agentic-sdd",
    "./cmd/sdd-check",
]
NEW_PACKAGE = "./internal/appops"
# Additive insertion: existing relative order is preserved, new package added
# immediately before the CLI packages.
CORE_PACKAGES = [*PRE_EXISTING[:5], NEW_PACKAGE, *PRE_EXISTING[5:]]
REQUIRED_DIRS = [pkg[2:] for pkg in CORE_PACKAGES if pkg != "."]

SANITIZED_ENV = "GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS="
REQUIRED_CHECK_IDS = ["mod-verify", "tidy-check", "vet", "test", "build-darwin", "diff-check"]
EXCLUDED_SELECTIONS = ("./...", "internal/opencodeadapter", "experiments/tui", "-tags=agentic_sdd_experimental")


def read(path):
    return (ROOT / path).read_text()


def backticked_packages(text):
    return [token for token in re.findall(r"`(\.[^`]*)`", text) if token == "." or token.startswith("./")]


class SelectorParityTests(unittest.TestCase):
    def test_taskfile_test_and_vet_select_all_eight_packages(self):
        taskfile = read("Taskfile.yml")
        for mode in ("test", "vet"):
            with self.subTest(mode=mode):
                expected = f"{SANITIZED_ENV} go {mode} {' '.join(CORE_PACKAGES)}"
                self.assertIn(expected, taskfile, f"Taskfile.yml {mode} must select every core package")
                self.assertNotIn(f"go {mode} ./...", taskfile)

    def test_harness_contract_required_vet_and_test_select_all_eight_packages(self):
        contract = json.loads(read("harness/checks.json"))
        required = {check["id"]: check for check in contract["checks"] if "required" in check["scopes"]}
        self.assertEqual(sorted(required), sorted(REQUIRED_CHECK_IDS))
        for mode in ("vet", "test"):
            with self.subTest(mode=mode):
                self.assertEqual(required[mode]["steps"][0]["argv"], ["go", mode, *CORE_PACKAGES])
                self.assertEqual(required[mode]["steps"][0]["env"], {})

    def test_core_shell_script_lists_required_directories_and_selection(self):
        script = read("tests/supported-core-go.sh")
        match = re.search(r"for dir in ([^;]+); do", script)
        self.assertIsNotNone(match, "supported-core script must verify required directories")
        self.assertEqual(match.group(1).split(), REQUIRED_DIRS)
        self.assertIn(f'go "$mode" {" ".join(CORE_PACKAGES)}', script)
        self.assertIn("export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOWORK=off GOFLAGS=", script)
        self.assertNotIn("$mode ./...", script)

    def test_lefthook_inherits_core_selection_without_its_own_package_list(self):
        lefthook = read("lefthook.yml")
        self.assertIn("task test", lefthook)
        self.assertIn("task vet", lefthook)
        self.assertNotIn("internal/", lefthook, "Lefthook must inherit the core selection, not duplicate it")

    def test_all_maintained_selectors_agree_on_the_same_package_list(self):
        taskfile = read("Taskfile.yml")
        script = read("tests/supported-core-go.sh")
        contract = json.loads(read("harness/checks.json"))
        required = {check["id"]: check for check in contract["checks"] if "required" in check["scopes"]}

        for mode in ("test", "vet"):
            with self.subTest(mode=mode):
                self.assertIn(f"go {mode} {' '.join(CORE_PACKAGES)}", taskfile)
                self.assertIn(f'go "$mode" {" ".join(CORE_PACKAGES)}', script)
                self.assertEqual(required[mode]["steps"][0]["argv"], ["go", mode, *CORE_PACKAGES])

    def test_no_excluded_or_aggregate_selection_is_introduced(self):
        for path in ("Taskfile.yml", "tests/supported-core-go.sh", "harness/checks.json", "lefthook.yml"):
            text = read(path)
            for token in EXCLUDED_SELECTIONS:
                with self.subTest(path=path, token=token):
                    self.assertNotIn(token, text)

    def test_existing_seven_packages_remain_selected(self):
        taskfile = read("Taskfile.yml")
        script = read("tests/supported-core-go.sh")
        contract = json.loads(read("harness/checks.json"))
        required = {check["id"]: check for check in contract["checks"] if "required" in check["scopes"]}
        for mode in ("test", "vet"):
            with self.subTest(mode=mode):
                argv = required[mode]["steps"][0]["argv"]
                for package in PRE_EXISTING:
                    self.assertIn(package, argv, f"{package} must stay selected in the harness contract")
                self.assertIn(f"go {mode} {' '.join(CORE_PACKAGES)}", taskfile)
                self.assertIn(f'go "$mode" {" ".join(CORE_PACKAGES)}', script)


class PackageCountGuidanceTests(unittest.TestCase):
    def assert_count_and_list(self, path, text):
        flat = " ".join(text.split())
        self.assertIn("8 supported-core", flat, f"{path} must state the eight-package count")
        self.assertNotIn("7 supported-core", flat, f"{path} still states the stale seven-package count")
        listed = [pkg for pkg in backticked_packages(flat) if pkg in CORE_PACKAGES]
        self.assertEqual(listed, CORE_PACKAGES, f"{path} package list must match the maintained selectors")

    def test_agents_md_count_and_list_guidance_is_current(self):
        self.assert_count_and_list("AGENTS.md", read("AGENTS.md"))

    def test_contributing_count_and_list_guidance_is_current(self):
        text = read("CONTRIBUTING.md")
        self.assert_count_and_list("CONTRIBUTING.md", text)
        flat = " ".join(text.split())
        # Both supported-core mentions (pre-commit description and default task
        # description) plus the experimental-exclusion note are current.
        self.assertEqual(flat.count("8 supported-core"), 2)
        self.assertEqual(flat.count("8-package core selection"), 1)
        self.assertNotIn("7-package", flat)


if __name__ == "__main__":
    unittest.main()
