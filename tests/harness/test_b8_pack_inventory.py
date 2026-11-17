"""Static pack inventory and profile projection checks for Feature 022 B8/T17.

Contract source: specs/022-architecture-remediation/spec.md FR-14 / AC-14, plan
sections 6 and 13.6, and the owner decisions O-02 (option A) and O-03 (option A).

This module reads packaged Markdown, profile frontmatter and one JSON fixture.
It does not install the pack, call a model or provider, edit a host
configuration, or prove that a runtime enforces a permission. Provider
availability and every runtime model/permission observation stay `not_run`.
"""

from pathlib import Path
import json
import re
import unittest


ROOT = Path(__file__).resolve().parents[2]
PACK = ROOT / "opencode-pack"
SKILLS = ROOT / "skills-files"
FIXTURE = Path(__file__).parent / "b8-profile-inventory.json"
ROUTING = PACK / "routing.md"
PACK_README = PACK / "README.md"

LANE_PREFIXES = {
    "opencode/": "console-free",
    "opencode-go/": "go",
    "9router/": "9router",
    "openai/": "openai",
    "openrouter/": "openrouter",
}


def flat(text):
    return re.sub(r"\s+", " ", text)


def parse_frontmatter(text):
    """Parse the pack's small YAML subset into scalar fields and ordered rules."""
    lines = text.replace("\r\n", "\n").split("\n")
    if not lines or lines[0].strip() != "---":
        raise AssertionError("missing opening frontmatter delimiter")
    fields, rules = {}, []
    in_permissions = False
    for line in lines[1:]:
        trimmed = line.strip()
        if trimmed == "---":
            return fields, rules
        if not trimmed or trimmed.startswith("#"):
            continue
        indented = line.startswith((" ", "\t"))
        if trimmed.startswith("- "):
            if not in_permissions:
                raise AssertionError(f"list item outside permissions: {trimmed!r}")
            key, value = _scalar(trimmed[2:])
            if key != "action":
                raise AssertionError(f"malformed permission item: {trimmed!r}")
            rules.append({"action": value})
            continue
        key, value = _scalar(trimmed)
        if not indented:
            in_permissions = key == "permissions"
            if key != "permissions":
                fields[key] = value
            continue
        if not in_permissions or not rules:
            raise AssertionError(f"unexpected indented key: {key!r}")
        if key in ("resource", "effect"):
            rules[-1][key] = value
        else:
            raise AssertionError(f"unexpected permission field: {key!r}")
    raise AssertionError("missing closing frontmatter delimiter")


def _scalar(text):
    key, _, value = text.partition(":")
    if not _:
        raise AssertionError(f"unparsable frontmatter line: {text!r}")
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
        value = value[1:-1]
    return key.strip(), value


def lane_for(model):
    for prefix, lane in LANE_PREFIXES.items():
        if model.startswith(prefix):
            return lane
    return "session" if not model else "unknown"


def pack_profiles(kind):
    """Return {profile id: path} for one declared profile set directory."""
    directories = {
        "with_opencode": PACK / "agents",
        "free_first": PACK / "presets" / "free-first",
        "fleet_adapter": PACK / "adapters",
    }
    directory = directories[kind]
    return {path.stem: path for path in directory.glob("*.md")}


def routing_rows(text):
    """Return {profile id: (mode, model)} for the documented routing table."""
    rows = {}
    for line in text.splitlines():
        trimmed = line.strip()
        if not trimmed.startswith("| `sdd-"):
            continue
        cells = trimmed.split("|")
        if len(cells) < 6:
            raise AssertionError(f"unexpected routing row: {line!r}")
        identifier = cells[1].strip().strip("`")
        model_cell = cells[3].strip()
        models = re.findall(r"`([^`]+)`", model_cell)
        model = models[0] if models else model_cell.strip("`")
        if model.lower().startswith("none"):
            model = ""
        rows[identifier] = (cells[2].strip(), model)
    return rows


def pack_markdown():
    return sorted(PACK.rglob("*.md"))


class PackInventoryFixtureTests(unittest.TestCase):
    maxDiff = None

    def setUp(self):
        self.fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))

    def test_fixture_shape_claims_no_runtime_observation(self):
        fixture = self.fixture
        self.assertEqual(fixture["version"], 1)
        self.assertEqual(fixture["evidence_type"], "static pack inventory and profile projection")
        self.assertEqual(fixture["authority"], "opencode-pack/routing.md")
        for key in ("runtime_model_calls", "provider_availability"):
            self.assertEqual(fixture[key], "not_run", f"{key} must stay not_run")
        sets = fixture["profile_sets"]
        self.assertEqual(sum(len(ids) for ids in sets.values()), len(fixture["profiles"]))
        for ids in sets.values():
            self.assertEqual(len(ids), len(set(ids)), "declared profile sets must not repeat an id")
        self.assertEqual(
            set(fixture["profiles"]),
            {identifier for ids in sets.values() for identifier in ids},
        )

    def test_pack_directories_match_the_declared_sets(self):
        for kind, ids in self.fixture["profile_sets"].items():
            with self.subTest(kind=kind):
                self.assertEqual(set(pack_profiles(kind)), set(ids))

    def test_profile_frontmatter_matches_the_inventory(self):
        declared = self.fixture["profiles"]
        for kind, ids in self.fixture["profile_sets"].items():
            for identifier in ids:
                with self.subTest(profile=identifier):
                    path = pack_profiles(kind)[identifier]
                    relative = path.relative_to(ROOT)
                    fields, rules = parse_frontmatter(path.read_text(encoding="utf-8"))
                    expected = declared[identifier]
                    self.assertEqual(fields["mode"], expected["mode"], f"{relative}: mode")
                    model = fields.get("model", "")
                    self.assertEqual(
                        lane_for(model), expected["lane"], f"{relative}: lane from model {model!r}"
                    )
                    writer = any(
                        rule.get("action") == "edit"
                        and rule.get("resource") == "*"
                        and rule.get("effect") == "allow"
                        for rule in rules
                    )
                    self.assertEqual(writer, expected["writer"], f"{relative}: writer projection")
                    if identifier == "sdd-orchestrator":
                        self.assertEqual(model, "", "the orchestrator must not pin a model")
                    self.assertTrue(fields.get("description", "").strip(), f"{relative}: description")
                    if identifier != "sdd-orchestrator":
                        self.assertFalse(
                            any(
                                rule.get("action") == "subagent" and rule.get("effect") == "allow"
                                for rule in rules
                            ),
                            f"{relative}: no installed subagent profile may delegate",
                        )

    def test_writer_inventory_is_derived_and_documented(self):
        declared = self.fixture["profiles"]
        for kind in ("with_opencode", "free_first"):
            derived = {
                identifier
                for identifier in self.fixture["profile_sets"][kind]
                if declared[identifier]["writer"]
            }
            with self.subTest(kind=kind):
                self.assertEqual(derived, set(self.fixture["writer_sets"][kind]))

        routing = flat(ROUTING.read_text(encoding="utf-8"))
        self.assertIn("## Writer inventory", routing, "routing.md has no writer inventory section")
        section = routing.split("## Writer inventory", 1)[1].split("## ", 1)[0]
        readme = flat(PACK_README.read_text(encoding="utf-8"))
        for identifier in self.fixture["writer_sets"]["with_opencode"]:
            self.assertIn(f"`{identifier}`", section, f"routing writer inventory omits {identifier}")
            self.assertIn(f"`{identifier}`", readme, f"pack README writer inventory omits {identifier}")
        for identifier in self.fixture["writer_sets"]["free_first"]:
            self.assertIn(f"`{identifier}`", section, f"routing writer inventory omits {identifier}")
        self.assertIn("sdd-openai-implement", self.fixture["writer_sets"]["with_opencode"])
        self.assertIn("derived from profile frontmatter", section)

    def test_routing_policy_projects_every_pinned_model(self):
        rows = routing_rows(ROUTING.read_text(encoding="utf-8"))
        self.assertTrue(rows, "routing table parsed no rows; the documented format changed")
        for identifier in self.fixture["profile_sets"]["with_opencode"]:
            with self.subTest(profile=identifier):
                path = pack_profiles("with_opencode")[identifier]
                fields, _ = parse_frontmatter(path.read_text(encoding="utf-8"))
                model = fields.get("model", "")
                self.assertIn(identifier, rows, f"routing.md has no row for {identifier}")
                if identifier == "sdd-orchestrator":
                    self.assertEqual(rows[identifier][1], "", "orchestrator row must not pin a model")
                else:
                    self.assertEqual(rows[identifier][1], model, f"routing row for {identifier}")

        routing = flat(ROUTING.read_text(encoding="utf-8"))
        preset_models = set()
        for identifier in self.fixture["profile_sets"]["free_first"]:
            with self.subTest(profile=identifier):
                path = pack_profiles("free_first")[identifier]
                fields, _ = parse_frontmatter(path.read_text(encoding="utf-8"))
                preset_models.add(fields["model"])
                self.assertIn(identifier, routing, f"routing.md does not name preset {identifier}")
        self.assertEqual(len(preset_models), 1, "the four presets must share one pinned model")
        self.assertIn(preset_models.pop(), routing, "routing.md does not project the preset model")

        for identifier in self.fixture["profile_sets"]["fleet_adapter"]:
            fields, _ = parse_frontmatter(
                pack_profiles("fleet_adapter")[identifier].read_text(encoding="utf-8")
            )
            self.assertIn(identifier, routing, f"routing.md does not name adapter {identifier}")
            self.assertIn(fields["model"], routing, f"routing.md does not project {identifier}'s model")
            self.assertIn("never installed", routing)

    def test_routing_policy_declares_one_authority_and_no_host_pins(self):
        text = flat(ROUTING.read_text(encoding="utf-8"))
        for clause in (
            "upstream owner of the pack's model, lane and fallback policy",
            "personal override, not this policy",
            "do not import host renames, pins or provider restrictions",
            "unverified runtime facts",
        ):
            self.assertIn(clause, text, f"routing.md is missing the authority clause: {clause}")

        for path in pack_markdown():
            body = path.read_text(encoding="utf-8")
            self.assertNotIn(
                "global AGENTS.md",
                body,
                f"{path.relative_to(ROOT)} defers policy to a personal host file",
            )
            self.assertNotIn(
                "claude-agent/",
                body,
                f"{path.relative_to(ROOT)} imports the personal Claude host lane",
            )
            self.assertNotIn(
                "~/.config/opencode/agents",
                body,
                f"{path.relative_to(ROOT)} pins a personal installation path",
            )

        for identifier, (_, model) in routing_rows(ROUTING.read_text(encoding="utf-8")).items():
            if not model:
                continue
            self.assertIn(
                lane_for(model),
                set(LANE_PREFIXES.values()),
                f"{identifier}: model {model!r} is not on a documented lane prefix",
            )

    def test_reviewer_profiles_are_ordinary_and_advisory(self):
        reviewers = {
            identifier
            for identifier, entry in self.fixture["profiles"].items()
            if entry["review_class"] == "ordinary-advisory"
        }
        self.assertTrue(reviewers, "the fixture declares no advisory reviewers")
        routing = flat(ROUTING.read_text(encoding="utf-8"))
        self.assertIn("## Reviewer authority", routing, "routing.md has no reviewer authority section")
        section = routing.split("## Reviewer authority", 1)[1].split("## ", 1)[0]
        for identifier in sorted(reviewers):
            with self.subTest(profile=identifier):
                kind = next(
                    kind for kind, ids in self.fixture["profile_sets"].items() if identifier in ids
                )
                path = pack_profiles(kind)[identifier]
                raw = path.read_text(encoding="utf-8")
                fields, _ = parse_frontmatter(raw)
                body = flat(raw)
                self.assertIn("advisory", fields["description"].lower(), f"{identifier}: description")
                self.assertNotRegex(
                    fields["description"],
                    r"(?i)\bIndependently review\b",
                    f"{identifier}: description still claims independent review",
                )
                self.assertIn("strict independent", body.lower(), f"{identifier}: strict-method label")
                self.assertIn(identifier, section, f"routing.md reviewer section omits {identifier}")

        method = self.fixture["strict_independent_method"]
        self.assertIsNone(method["profile"], "no installed profile may own the strict method")
        self.assertEqual(method["runtime"], "not_installed")
        self.assertEqual(method["certification_claim"], "none")
        reference = ROOT / method["reference"]
        self.assertTrue(reference.is_file(), "the portable strict method reference is missing")
        reference_text = flat(reference.read_text(encoding="utf-8"))
        for clause in (
            "must not edit files, run tests, builds or formatters",
            "remote, MCP or code-execution routes",
            "report the independent route as **blocked**",
        ):
            self.assertIn(clause, reference_text, f"strict method lost its clause: {clause}")

    def test_pack_relative_links_resolve(self):
        for path in pack_markdown():
            text = path.read_text(encoding="utf-8")
            for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", text):
                if "://" in target or target.startswith(("#", "mailto:")):
                    continue
                relative = target.partition("#")[0]
                if not relative:
                    continue
                if relative.startswith("../skills-files/"):
                    resolved = (path.parent / relative).resolve()
                elif relative.startswith("../agentic-sdd-router/"):
                    resolved = SKILLS / relative[len("../") :]
                else:
                    resolved = (path.parent / relative).resolve()
                self.assertTrue(
                    resolved.is_file(),
                    f"{path.relative_to(ROOT)}: unresolved link {target}",
                )


if __name__ == "__main__":
    unittest.main()
