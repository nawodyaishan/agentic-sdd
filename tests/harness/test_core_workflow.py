"""Source/fixture regression checks only; never execute prompts or call a model."""

from pathlib import Path
import copy
import json
import re
import unittest


ROOT = Path(__file__).resolve().parents[2]
SKILLS = ROOT / "skills-files"
HUMAN = "agentic-sdd-router/references/human-input.md"
POLICY = "agentic-sdd-router/references/workflow-policy.md"
IMPLEMENT = "agentic-sdd-implement/SKILL.md"

# Expected clauses come from 008 R2/R3/C2, not from a policy evaluator or model.
# They guard text, not the runtime meaning or enforcement of natural language.
RULES = {
    "discover": (HUMAN, [r"Investigate facts discoverable", r"Reuse settled answers", r"inaccessible facts remain unknown"]),
    "incomplete-scope": (HUMAN, [r"For an incomplete request, distinguish established facts from assumptions", r"propose scope, exclusions and acceptance conditions"]),
    "questions": (HUMAN, [r"at most three independent questions per round", r"recommendation and credible alternatives with consequences", r"Defer a question whose answer depends"]),
    "clear": (HUMAN, [r"Clear requests proceed without a mandatory interview"]),
    "direct-fix": (IMPLEMENT, [r"Do not create feature documents, batch IDs or task-state writes for this path"]),
    "limited-document": ("agentic-sdd-router/SKILL.md", [r"Draft only that document", r"do not draft downstream documents"]),
    "review-only": (POLICY, [r"An explicitly requested review returns findings and changes nothing: no edits to code, documentation, approval records, task status or batch state"]),
    "research": ("agentic-sdd-research-spec/SKILL.md", [r"Return to the interrupted stage after answering it", r"Never create feature documents just to hold research for a direct fix"]),
    "teaching": ("agentic-sdd-teach/SKILL.md", [r"Do not force an interview, create lesson files/assets", r"not feature approval, batch execution or permission to create learning files"]),
    "questionnaire": ("agentic-sdd-questionnaire/SKILL.md", [r"Drafting is not permission to send, contact a person, share project information or start implementation", r"An answer alone cannot approve a feature, waive a finding or authorize a batch"]),
    "bare-continue": (POLICY, [r'A bare "continue" is not authorization for the next batch']),
    "freshness": (POLICY, [r"Verify current code state before trusting continuation notes or old test results"]),
    "ready-verdict": (POLICY, [r"An agent's `Ready` verdict is a technical assessment, never human approval"]),
    "combined-approval": (POLICY, [r"Record the combined approval once", r"No implementation before that approval"]),
    "batch-boundary": (POLICY, [r"After combined approval \*\*and\*\* authorization to implement, execute ONE selected batch", r"then stop for human review"]),
    "partial-acceptance": (HUMAN, [r"Preserve acceptance scope and technical findings/missing evidence; no automatic waiver"]),
    "live-spend": (HUMAN, [r"Tool/spending/live operation permission", r"Separate action/environment/data/limits decision; feature approval alone supplies none"]),
    "explicit-batch": (POLICY, [r"A clearly scoped new batch authorization may advance past an earlier review checkpoint, but it does not authorize later batches"]),
    "completion": (IMPLEMENT, [r"changed behavior and actual checks", r"risk-relevant review points, limitations and the next permitted action"]),
    "completion-owner": (IMPLEMENT, [r"one completion-review invocation", r"Do not request a second review of unchanged work from the router"]),
    "review-diagnosis": ("agentic-sdd-debugging/SKILL.md", [r"make no edits or write-producing test runs", r"Missing prerequisites are not proof of the target regression"]),
    "material-change": (POLICY, [r"Material changes to requirements, design or batch scope need review of the affected changes", r"approval for unaffected work is preserved"]),
}
CASE_IDS = {
    "discover-before-asking", "reuse-settled-answer", "ready-question-frontier",
    "clear-request", "direct-fix", "spec-only", "review-only", "research-return",
    "teaching-opt-in", "questionnaire-no-send", "bare-continue", "stale-handover",
    "ready-is-not-consent", "approval-without-batch", "accepted-partial",
    "green-checks", "live-and-spend", "explicit-batch-positive",
    "completion-summary", "review-diagnosis", "material-scope-change",
}


class CoreWorkflowTextTests(unittest.TestCase):
    def assert_rule(self, rule, documents):
        path, patterns = RULES[rule]
        text = " ".join(documents[path].split())
        for pattern in patterns:
            self.assertIsNotNone(re.search(pattern, text, re.I), f"{rule}: missing clause in {path}: {pattern}")

    def assert_cases(self, manifest):
        self.assertEqual(manifest["version"], 1)
        self.assertEqual(manifest["evidence_type"], "static instruction coverage")
        cases = manifest["scenarios"]
        self.assertEqual(len(cases), len(CASE_IDS))
        self.assertEqual({case["id"] for case in cases}, CASE_IDS)
        for case in cases:
            self.assertEqual(case["agent_observation"], "not_run")
            for key in ("context", "request", "expected_action", "forbidden_effects", "rules"):
                self.assertTrue(case[key], f"{case['id']}: missing {key}")
            self.assertTrue(set(case["rules"]) <= RULES.keys())
        self.assertEqual({rule for case in cases for rule in case["rules"]}, RULES.keys())

    def assert_links(self, owner, text):
        for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)", text):
            if "://" not in target:
                path = target.partition("#")[0]
                if path:
                    self.assertTrue((SKILLS / owner).parent.joinpath(path).is_file(), f"{owner}: missing {target}")

    def test_existing_core_rules_are_characterized(self):
        documents = {path: (SKILLS / path).read_text() for path, _ in RULES.values()}
        for rule in RULES:
            with self.subTest(rule=rule):
                self.assert_rule(rule, documents)

    def test_each_seeded_rule_removal_is_detected(self):
        for rule, (path, patterns) in RULES.items():
            with self.subTest(rule=rule):
                text = " ".join((SKILLS / path).read_text().split())
                mutated, count = re.subn(patterns[0], "[removed clause]", text, flags=re.I)
                self.assertGreater(count, 0, "seed must change actual source")
                with self.assertRaises(AssertionError):
                    self.assert_rule(rule, {path: mutated})

    def test_softened_authority_clauses_are_detected(self):
        seeds = [
            ("bare-continue", 'A bare "continue" is not authorization', 'A bare "continue" is authorization'),
            ("explicit-batch", "does not authorize later batches", "does authorize later batches"),
            ("live-spend", "feature approval alone supplies none", "feature approval alone supplies permission"),
            ("completion-owner", "one completion-review invocation", "two completion-review invocations"),
        ]
        for rule, old, new in seeds:
            with self.subTest(rule=rule):
                path = RULES[rule][0]
                text = " ".join((SKILLS / path).read_text().split())
                self.assertIn(old, text)
                with self.assertRaises(AssertionError):
                    self.assert_rule(rule, {path: text.replace(old, new)})

    def test_core_reference_targets_exist_and_missing_target_fails(self):
        for path in {path for path, _ in RULES.values()}:
            self.assert_links(path, (SKILLS / path).read_text())
        text = (SKILLS / HUMAN).read_text()
        old = "../../agentic-sdd-questionnaire/SKILL.md"
        self.assertIn(old, text)
        with self.assertRaises(AssertionError):
            self.assert_links(HUMAN, text.replace(old, "../../agentic-sdd-missing/SKILL.md"))

    def test_scenario_manifest_covers_core_cases_without_model_claims(self):
        self.assert_cases(json.loads((Path(__file__).parent / "core-workflow-scenarios.json").read_text()))

    def test_missing_case_unknown_rule_and_invented_pass_are_rejected(self):
        manifest = json.loads((Path(__file__).parent / "core-workflow-scenarios.json").read_text())
        missing = copy.deepcopy(manifest)
        missing["scenarios"].pop()
        unknown = copy.deepcopy(manifest)
        unknown["scenarios"][0]["rules"].append("unknown-rule")
        invented = copy.deepcopy(manifest)
        invented["scenarios"][0]["agent_observation"] = "pass"
        for mutation in (missing, unknown, invented):
            with self.assertRaises(AssertionError):
                self.assert_cases(mutation)

    def test_incomplete_request_separates_facts_assumptions_and_scope(self):
        text = (SKILLS / "agentic-sdd-router/references/human-input.md").read_text()
        self.assertRegex(text, r"(?is)for an incomplete request.{0,300}established facts")
        for phrase in ("assumptions", "scope", "exclusions", "acceptance conditions"):
            self.assertIn(phrase, text)

    def test_completion_summary_prioritizes_behavior_and_risk_review(self):
        text = (SKILLS / "agentic-sdd-implement/SKILL.md").read_text()
        self.assertIn("changed behavior", text)
        self.assertRegex(text, r"risk[- ]relevant review points")

    def test_getting_started_has_three_bounded_worked_paths(self):
        guide = ROOT / "docs/getting-started.md"
        self.assertTrue(guide.is_file(), "missing concise core onboarding guide")
        text = guide.read_text()
        for phrase in (
            "Direct fix: no feature paperwork",
            "Feature: one combined approval and one batch",
            "Review and resume: acceptance is not continuation",
            'go run ./cmd/agentic-sdd preview --home "$demo_home"',
            "GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go version",
            "not a security sandbox",
            "not model-compliance evidence",
            "Teaching is optional",
        ):
            self.assertIn(phrase, text)

    def test_public_entrypoints_link_the_worked_paths(self):
        for name, target in (
            ("README.md", "docs/getting-started.md"),
            ("CONTRIBUTING.md", "docs/getting-started.md"),
            ("docs/README.md", "getting-started.md"),
        ):
            with self.subTest(path=name):
                self.assertIn(target, (ROOT / name).read_text())

    def test_security_reporting_does_not_invent_a_private_channel(self):
        text = (ROOT / "CONTRIBUTING.md").read_text()
        for phrase in (
            "Do not post credentials",
            "if available",
            "public issue requesting a private reporting channel",
            "No dedicated security email or response-time guarantee",
        ):
            self.assertIn(phrase, text)


if __name__ == "__main__":
    unittest.main()
