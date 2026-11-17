"""Static lightweight-context, read-only and research-trigger checks for B8/T18.

Contract source: specs/022-architecture-remediation/spec.md FR-15/FR-22 and
AC-15/AC-22, plan sections 6 and 13.6.

This module reads packaged Markdown and one JSON fixture, and creates a
disposable unindexed directory to inspect. It never activates a skill, calls a
model or MCP server, retrieves a page, redacts a query, edits a checkout, or
proves runtime compliance. Word counts are static mandatory-text measurements,
not runtime token, spend or behavior evidence; every scenario observation stays
`not_run`.
"""

from pathlib import Path
import json
import re
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SKILLS = ROOT / "skills-files"
FIXTURE = Path(__file__).parent / "b8-context-inventory.json"

POLICY = SKILLS / "agentic-sdd-router/references/workflow-policy.md"
KERNEL = SKILLS / "agentic-sdd-router/references/lightweight-authority.md"
RESEARCH = SKILLS / "agentic-sdd-research-spec/SKILL.md"

# Completion outputs each lightweight branch must keep advertising (AC-22).
COMPLETION_OUTPUTS = {
    "wait-what": (
        "Re-pitch briefly",
        "The next consequential decision or already permitted action",
    ),
    "teach": ("Finish with what was taught", "the user's next learning choice"),
    "handoff": ("Check that a fresh receiver can distinguish", "next permitted action"),
}

CONDITIONAL_MARKERS = (
    "only if",
    "when ",
    "whenever",
    "on demand",
    "actually need",
    "before the request",
    "conditional",
)

READ_INSTRUCTION = re.compile(r"Read (?:the )?\[[^\]]*\]\(([^)\s#]+)")


def flat(text):
    return re.sub(r"\s+", " ", text)


def body_words(path):
    text = Path(path).read_text(encoding="utf-8")
    if text.startswith("---"):
        parts = text.split("---", 2)
        if len(parts) == 3:
            text = parts[2]
    return len(text.split())


def read_instructions(path):
    """Yield (resolved markdown target, containing sentence) for `Read [..](..)`."""
    text = flat(Path(path).read_text(encoding="utf-8"))
    for match in READ_INSTRUCTION.finditer(text):
        target = match.group(1)
        if "://" in target or target.startswith(("#", "mailto:")):
            continue
        end = text.find(". ", match.end())
        sentence = text[max(text.rfind(". ", 0, match.start()), 0) : len(text) if end < 0 else end]
        resolved = (Path(path).parent / target).resolve()
        yield resolved, sentence.lower()


def is_conditional(sentence):
    return any(marker in sentence for marker in CONDITIONAL_MARKERS)


def mandatory_closure(entry, references):
    """Follow unconditional `Read [..](..)` instructions from the declared set."""
    seen = {Path(reference).resolve() for reference in references}
    queue = list(seen)
    while queue:
        path = queue.pop(0)
        if not path.is_file():
            continue
        for target, sentence in read_instructions(path):
            if target.suffix != ".md" or not target.is_file():
                continue
            if is_conditional(sentence):
                continue
            if target not in seen:
                seen.add(target)
                queue.append(target)
    return seen


class ContextInventoryFixtureTests(unittest.TestCase):
    maxDiff = None

    def setUp(self):
        self.fixture = json.loads(FIXTURE.read_text(encoding="utf-8"))
        self.branches = self.fixture["branches"]

    def test_fixture_shape_claims_no_runtime_observation(self):
        self.assertEqual(self.fixture["version"], 1)
        for key in ("runtime_token_measurement", "model_compliance"):
            self.assertEqual(self.fixture[key], "not_run", f"{key} must stay not_run")
        self.assertEqual(set(self.branches), set(COMPLETION_OUTPUTS))
        scenarios = self.fixture["read_only_scenarios"]
        self.assertTrue(scenarios)
        for scenario in scenarios:
            self.assertEqual(scenario["agent_observation"], "not_run", scenario["id"])
            self.assertTrue(scenario["expected_route"], scenario["id"])

    def test_lightweight_branches_mandate_the_authority_kernel(self):
        for name, branch in self.branches.items():
            with self.subTest(branch=name):
                entry = ROOT / branch["entry"]
                self.assertTrue(entry.is_file(), f"{branch['entry']} missing")
                text = flat(entry.read_text(encoding="utf-8"))
                self.assertIn("lightweight-authority.md", text, f"{name}: kernel not linked")
                for reference in branch["mandatory_references"]:
                    self.assertTrue((ROOT / reference).is_file(), f"missing {reference}")
                for reference in branch["baseline_references"]:
                    self.assertTrue((ROOT / reference).is_file(), f"missing {reference}")
                self.assertNotIn(
                    "Read the [shared policy](../agentic-sdd-router/references/workflow-policy.md) once\nper unchanged context",
                    entry.read_text(encoding="utf-8"),
                    f"{name}: still mandates the full policy",
                )

        self.assertTrue(KERNEL.is_file(), "the lightweight authority kernel is missing")

    def test_mandatory_load_is_transitive_and_conditional_references_stay_conditional(self):
        conditional = {str(ROOT / path) for path in self.fixture["conditional_references"]}
        for name, branch in self.branches.items():
            with self.subTest(branch=name):
                closure = mandatory_closure(ROOT / branch["entry"], branch["mandatory_references"])
                self.assertIn(KERNEL.resolve(), closure, f"{name}: kernel is not mandatory")
                self.assertNotIn(POLICY.resolve(), closure, f"{name}: full policy became mandatory")
                for path in closure:
                    if path.name in {"human-input.md", "review-resolution.md", "testing.md", "independent-review.md"}:
                        self.assertIn(str(path), conditional, f"{name}: {path.name} is not declared conditional")

    def test_kernel_keeps_every_prohibition(self):
        text = flat(KERNEL.read_text(encoding="utf-8"))
        for rule in self.fixture["prohibitions"]:
            with self.subTest(rule=rule["id"]):
                self.assertIn(rule["pattern"], text, f"kernel lost the {rule['id']} prohibition")
        self.assertIn("Do not delegate or spend", text)
        self.assertIn("no subagents", text.lower())
        self.assertIn("credentials", text.lower())

    def test_branch_completion_outputs_are_preserved(self):
        for name, clauses in COMPLETION_OUTPUTS.items():
            with self.subTest(branch=name):
                text = flat((ROOT / self.branches[name]["entry"]).read_text(encoding="utf-8"))
                for clause in clauses:
                    self.assertIn(clause, text, f"{name}: lost its completion output {clause!r}")
        kernel = flat(KERNEL.read_text(encoding="utf-8"))
        self.assertIn("Finish the branch", kernel)
        self.assertIn("what the user may decide next", kernel)

    def test_lightweight_branches_load_fewer_mandatory_words_than_baseline(self):
        measurements = {}
        for name, branch in self.branches.items():
            with self.subTest(branch=name):
                entry = ROOT / branch["entry"]
                baseline = body_words(entry) + sum(
                    body_words(ROOT / reference) for reference in branch["baseline_references"]
                )
                mandatory = mandatory_closure(entry, branch["mandatory_references"])
                current = body_words(entry) + sum(body_words(path) for path in mandatory)
                measurements[name] = (baseline, current)
                self.assertLess(
                    current,
                    baseline,
                    f"{name}: mandatory words {current} are not fewer than baseline {baseline}",
                )
        # Static text measurements only; recorded in the test outcome, not a token claim.
        self.assertTrue(measurements)

    def test_codegraph_bootstrap_is_conditioned_on_write_authority(self):
        policy = flat(POLICY.read_text(encoding="utf-8"))
        for clause in (
            "Creating or syncing an index writes into the checkout",
            "only when the current mode has write authority over that checkout",
            "read-only review, explanation or handoff without that authority uses targeted local search",
            "creates nothing",
            "treat stale entries or a stale index as uncertain rather than current",
        ):
            self.assertIn(clause, policy, f"workflow policy lost the CodeGraph condition: {clause}")
        self.assertNotIn(
            "if the index is absent and the `codegraph` CLI is available on an ordinary project checkout, initialize it",
            policy,
            "the unconditional bootstrap instruction is still present",
        )

    def test_read_only_scenarios_declare_no_bootstrap_and_stay_unobserved(self):
        scenarios = {scenario["id"]: scenario for scenario in self.fixture["read_only_scenarios"]}
        for identifier in ("unindexed-read-only-review", "stale-index-read-only-review"):
            scenario = scenarios[identifier]
            self.assertEqual(scenario["bootstrap"], "not_allowed", identifier)
            self.assertEqual(scenario["expected_created_artifacts"], [], identifier)
            self.assertIn(scenario["expected_route"], {"targeted_local_read", "read_with_staleness_flag"})
        self.assertEqual(scenarios["write-authorized-bootstrap"]["bootstrap"], "allowed")
        self.assertEqual(scenarios["write-authorized-bootstrap"]["expected_created_artifacts"], [".codegraph/"])
        self.assertEqual(scenarios["broad-research-without-sdd-decision"]["expected_route"], "separately_authorized_method")
        self.assertEqual(scenarios["named-sdd-blocker-research"]["expected_route"], "agentic-sdd-research-spec")

        # An unindexed disposable tree stays unmodified by this static check: the
        # fixture only declares the instruction contract, it never runs an agent.
        with tempfile.TemporaryDirectory(prefix="sdd-b8-unindexed-") as tmp:
            tree = Path(tmp)
            (tree / "source.md").write_text("disposable read-only fixture\n", encoding="utf-8")
            self.assertFalse((tree / ".codegraph").exists())
            for scenario in scenarios.values():
                for artifact in scenario["expected_created_artifacts"]:
                    if artifact == ".codegraph/":
                        continue
                    self.assertFalse((tree / artifact).exists(), scenario["id"])
            self.assertFalse((tree / ".codegraph").exists())

    def test_research_trigger_and_body_name_the_same_boundary(self):
        text = RESEARCH.read_text(encoding="utf-8")
        description = re.search(r'(?m)^description:\s*"?(.+?)"?\s*$', text).group(1)
        body = flat(text.split("---", 2)[2])
        self.assertIn("name the affected SDD decision", description)
        self.assertIn("must still name the affected SDD decision", body)
        self.assertIn("separately authorized method", body)
        self.assertNotIn("Trigger only on an explicit research request or a named", body)
        self.assertIn("Return to the interrupted stage after answering it", body)
        self.assertIn("Never create feature documents just to hold research for a direct fix", body)

    def test_changed_instruction_links_resolve(self):
        documents = [KERNEL, RESEARCH, POLICY]
        documents.extend(ROOT / branch["entry"] for branch in self.branches.values())
        for path in documents:
            text = path.read_text(encoding="utf-8")
            for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", text):
                if "://" in target or target.startswith(("#", "mailto:")):
                    continue
                relative = target.partition("#")[0]
                if not relative:
                    continue
                resolved = (path.parent / relative).resolve()
                self.assertTrue(
                    resolved.is_file(),
                    f"{path.relative_to(ROOT)}: unresolved link {target}",
                )

    def test_sibling_branch_links_still_reach_the_router(self):
        for name, branch in self.branches.items():
            with self.subTest(branch=name):
                entry = ROOT / branch["entry"]
                targets = {
                    (entry.parent / target.partition("#")[0]).resolve()
                    for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", entry.read_text(encoding="utf-8"))
                    if "://" not in target and not target.startswith(("#", "mailto:"))
                }
                self.assertIn(KERNEL.resolve(), targets, f"{name}: kernel link does not resolve")
                self.assertIn(
                    (SKILLS / "agentic-sdd-router/references/matt-pocock-notice.md").resolve(),
                    targets,
                    f"{name}: attribution link does not resolve",
                )


if __name__ == "__main__":
    unittest.main()
