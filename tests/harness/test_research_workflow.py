"""Static instruction and declarative-fixture checks for Feature 018.

This suite reads packaged Markdown and a JSON manifest. It does not activate a
skill, call a model or MCP server, retrieve a page, redact a query, or enforce
any security control. A pass means the expected instructions and fixture fields
are present; runtime activation, dispatch-time redaction, retrieval and provider
behavior remain `not_run`.

Contract source: specs/018-research-evidence-handoff/plan.md sections 3.1-3.6
and 5, plus the preserved 008 clauses exercised by test_core_workflow.py.
"""

from pathlib import Path
import copy
import json
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]
SKILLS = ROOT / "skills-files"

RESEARCH = SKILLS / "agentic-sdd-research-spec/SKILL.md"
REFERENCE = SKILLS / "agentic-sdd-research-spec/references/search-mcp-workflow.md"
POLICY = SKILLS / "agentic-sdd-router/references/workflow-policy.md"
SPEC = SKILLS / "agentic-sdd-spec/SKILL.md"
PLAN = SKILLS / "agentic-sdd-plan/SKILL.md"
TASKS = SKILLS / "agentic-sdd-tasks/SKILL.md"
VERIFICATION = SKILLS / "agentic-sdd-verification-review/SKILL.md"
DEBUGGING = SKILLS / "agentic-sdd-debugging/SKILL.md"
LIBRARIAN = ROOT / "opencode-pack/agents/sdd-research-librarian.md"

STAGE_FILES = (SPEC, PLAN, TASKS, VERIFICATION, DEBUGGING)
STAGE_RULES = (
    "stage-spec-pointer",
    "stage-plan-pointer",
    "stage-tasks-reuse",
    "stage-verification-readonly",
    "stage-debugging-main-agent",
)

SEVEN_FIELDS = (
    "Question / affected SDD decision",
    "Finding",
    "Evidence",
    "Applicability",
    "Contradiction / caveat",
    "Impact",
    "Next action",
)
PRESERVED_CLAUSES = (
    "Return to the interrupted stage after answering it",
    "Never create feature documents just to hold research for a direct fix",
)
CLAIM_LABELS = ("verified primary", "corroborated", "reported only", "unverified")
EFFORT_LEVELS = ("R0", "R1", "R2", "R3")

RESEARCH_LINK = r"\[[^\]]*\]\([^)]*agentic-sdd-research-spec[^)]*\)"

# Each rule is (owning file, required patterns). Patterns name observable
# behavior from plan sections 3.1-3.6; they are not a policy evaluator.
RULES = {
    # Canonical research skill.
    "research-trigger": (RESEARCH, [
        r"explicit (?:research )?request|explicitly (?:asks?|requests?)",
        r"named[^.]{0,80}(?:uncertainty|fact|question)",
        r"block",
    ]),
    "research-return": (RESEARCH, [r"Return to the interrupted stage after answering it"]),
    "no-auto-docs": (RESEARCH, [r"Never create feature documents just to hold research for a direct fix"]),
    "effort-levels": (RESEARCH, [
        r"\bR0\b", r"\bR1\b", r"\bR2\b", r"\bR3\b",
        r"decision heuristic|not (?:a )?(?:new )?(?:phase|stage|permission)",
    ]),
    "bounded-brief": (RESEARCH, [
        r"question[^.]{0,180}(?:decision|affects?)",
        r"already[- ](?:inspected|read)|sources already|existing sources",
        r"permitted tools?|privacy scope|evidence threshold",
    ]),
    "local-first": (RESEARCH, [
        r"local (?:source|code|repository)",
        r"(?:first|before (?:external|web|remote))",
    ]),
    "privacy-external-query": (RESEARCH, [
        r"private (?:repository |repo )?code",
        r"\btokens?\b",
        r"customer data",
        r"internal (?:hostnames?|hosts?)",
        r"private paths?",
        r"secrets?|credentials?",
        r"without [^.]{0,60}authoriz",
    ]),
    "untrusted-content": (RESEARCH, [
        r"untrusted",
        r"never (?:change|alter|override)",
        r"polic(?:y|ies)",
        r"permissions?",
    ]),
    "handoff-fields": (RESEARCH, [re.escape(field) for field in SEVEN_FIELDS]),
    "approval-preservation": (RESEARCH, [
        r"approved (?:requirements|design|scope)|approval state|approval record",
        r"human review",
        r"(?:never|does not|doesn't|cannot)[^.]{0,160}(?:starts? a batch|authoriz|grant|change[s]? (?:permissions|fallbacks))",
    ]),
    "no-subagents-default": (RESEARCH, [
        r"(?:do not|never|no)[^.]{0,60}(?:spawn|create|launch|delegate)[^.]{0,60}subagents?",
        r"(?:unless|only (?:when|if))[^.]{0,80}(?:user|owner)[^.]{0,60}(?:requests?|asks?)",
    ]),
    "research-md-conditional": (RESEARCH, [
        r"research\.md",
        r"(?:only|conditional|unless|not (?:create|add)[^.]{0,60}by default)",
    ]),
    # Packaged search/MCP reference.
    "capability-matrix": (REFERENCE, [
        r"codegraph|local (?:code|repository) (?:discovery|evidence|call[- ]graph)",
        r"(?:versioned|library)[^.]{0,40}(?:documentation|docs)",
        r"search",
        r"fetch",
    ]),
    "exa-discipline": (REFERENCE, [
        r"distinct angles?",
        r"dedup(?:licat|e)",
        r"fetch[^.]{0,100}verif|verif[^.]{0,100}fetch",
        r"stop[^.]{0,100}(?:settled|threshold|decision)",
    ]),
    "context7-resolution": (REFERENCE, [
        r"resolve[- ]?id|resolve(?:s|d)? (?:the )?library[- ]id|library (?:id|identifier)[^.]{0,60}(?:resolve|resolution)",
        r"query[- ]docs|version[- ]scoped (?:docs|documentation)",
    ]),
    "version-mismatch": (REFERENCE, [
        r"(?:wrong|different|mismatched?)[- ]version|version mismatch|another version",
        r"(?:reject|qualif|do not (?:present|assume|treat))",
    ]),
    "optional-site-tools": (REFERENCE, [
        r"(?:map|crawl|extract)",
        r"installed",
        r"authoriz",
    ]),
    "known-url-direct": (REFERENCE, [
        r"known (?:exact )?(?:URL|page|document)",
        r"direct(?:ly)? (?:fetch|open)|skip (?:the )?(?:broad )?search|not another broad search",
    ]),
    "unavailable-fallback": (REFERENCE, [
        r"(?:unavailable|not (?:available|connected|reachable)|fails?)",
        r"fallback|permitted alternative|report",
        r"do not (?:invent|claim|silently)",
    ]),
    "agent-run-not-implicit": (REFERENCE, [
        r"agent_run",
        r"separate[^.]{0,60}(?:authoriz|approv|permission)",
        r"hosted[^.]{0,30}research|deep research",
    ]),
    "claim-labels": (REFERENCE, [re.escape(label) for label in CLAIM_LABELS]),
    "snippet-not-evidence": (REFERENCE, [
        r"snippet",
        r"not (?:confirmation|verified|proof|evidence)|lead",
    ]),
    "runtime-facts": (REFERENCE, [
        r"runtime (?:fact|availability)|active client",
        r"not[^.]{0,140}guarantee",
    ]),
    "stop-or-report-gap": (REFERENCE, [
        r"stop[^.]{0,120}(?:settled|threshold|decision)",
        r"(?:remaining|residual|unresolved) (?:uncertainty|gap|unknown)|report[^.]{0,80}(?:gap|unknown|blocked)",
    ]),
    # Stage pointers.
    "stage-spec-pointer": (SPEC, [RESEARCH_LINK, r"spec\.md"]),
    "stage-plan-pointer": (PLAN, [RESEARCH_LINK, r"plan\.md"]),
    "stage-tasks-reuse": (TASKS, [RESEARCH_LINK, r"reuse|reused", r"research again|re-?search|search again"]),
    "stage-verification-readonly": (VERIFICATION, [
        RESEARCH_LINK,
        r"read-only",
        r"(?:documentation|docs|retrieved)[^.]{0,220}(?:never|not)[^.]{0,140}(?:executed|run)",
        r"no (?:file|state|repository|source)[^.]{0,60}(?:mutation|change|write)"
        r"|(?:changes?|modif(?:y|ies)|writes?|edits?) no (?:files?|state|approval record|batch state|project state)",
    ]),
    "stage-debugging-main-agent": (DEBUGGING, [RESEARCH_LINK, r"root[- ]cause", r"main agent"]),
    # Router policy and optional librarian body.
    "policy-precedence": (POLICY, [
        r"generic[^.]{0,40}research",
        r"precedence|takes precedence",
        r"background[- ]agent|mandatory",
    ]),
    "policy-external-privacy": (POLICY, [
        r"external quer",
        r"secrets?",
        r"credentials?",
        r"private (?:repository |repo )?code|private code",
        r"internal (?:hostnames?|hosts?)",
        r"private paths?",
        r"untrusted",
    ]),
    "librarian-untrusted": (LIBRARIAN, [
        r"untrusted",
        r"never (?:change|alter|override)",
        r"policy",
        r"permission",
    ]),
    "librarian-handoff": (LIBRARIAN, [
        r"agentic-sdd-research-spec",
        r"handoff",
        r"applicability",
        r"next action",
        r"contradiction",
    ]),
    "librarian-permissions": (LIBRARIAN, [
        r"model: opencode/deepseek-v4\.1-flash-free",
        r"mode: subagent",
    ]),
}

SCENARIO_IDS = {
    "r0-no-research",
    "r1-targeted-fact",
    "r2-comparative-decision",
    "version-mismatch-context7",
    "contradictory-snippet-only",
    "search-mcp-unavailable",
    "unauthorized-agent-run",
    "unauthorized-hosted-research",
    "credential-private-path-query",
    "valid-seven-field-handoff",
    "invalid-seven-field-handoff",
    "stage-spec-uncertainty",
    "stage-plan-version-choice",
    "stage-tasks-reuse-evidence",
    "stage-verification-docs-are-not-tests",
    "stage-debugging-named-fact",
    "policy-precedence-generic-research",
    "librarian-untrusted-output",
    "known-url-direct-fetch",
}
TRIGGER_IDS = {
    "trigger-positive-sdd-blocking-version",
    "trigger-positive-direct-fix-fact",
    "trigger-negative-generic-research",
    "trigger-negative-topic-report",
    "trigger-negative-non-sdd-request",
    "trigger-negative-delegated-route",
}


class ResearchWorkflowContractTests(unittest.TestCase):
    """Static text and fixture-shape checks only.

    These tests never activate a skill, call a model or MCP server, retrieve a
    page, redact a query, or prove a permission is enforced. No runtime
    redaction, retrieval, delegation or security-enforcement behavior is
    claimed; all such observations stay `not_run`.
    """

    maxDiff = None

    def label(self, path):
        try:
            return str(Path(path).relative_to(ROOT))
        except ValueError:
            return str(path)

    def source(self, path):
        path = Path(path)
        self.assertTrue(path.is_file(), f"missing required source: {self.label(path)}")
        return path.read_text(encoding="utf-8")

    def flat(self, path, documents=None):
        text = documents[path] if documents and path in documents else self.source(path)
        return re.sub(r"\s+", " ", text)

    def assert_rule(self, rule, documents=None):
        path, patterns = RULES[rule]
        text = self.flat(path, documents)
        for pattern in patterns:
            self.assertIsNotNone(
                re.search(pattern, text, re.I),
                f"{rule}: missing clause in {self.label(path)}: {pattern}",
            )

    def assert_links(self, path):
        path = Path(path)
        text = self.source(path)
        for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", text):
            if "://" in target or target.startswith(("#", "mailto:")):
                continue
            relative = target.partition("#")[0]
            if not relative:
                continue
            resolved = (path.parent / relative).resolve()
            self.assertTrue(resolved.is_file(), f"{self.label(path)}: unresolved link {target}")

    def assert_manifest(self, manifest):
        self.assertEqual(manifest["version"], 1)
        self.assertEqual(manifest["evidence_type"], "static instruction coverage")
        self.assertRegex(manifest["execution"], r"(?i)no model")
        self.assertIn("MCP", manifest["execution"])
        self.assertEqual(manifest["runtime_activation"], "not_run")
        self.assertEqual(set(manifest["effort_levels"]), set(EFFORT_LEVELS))
        self.assertEqual(tuple(manifest["handoff_fields"]), SEVEN_FIELDS)
        self.assertEqual(tuple(manifest["claim_labels"]), CLAIM_LABELS)
        self.assertEqual(manifest["trigger_fixture"]["coverage_type"], "static lexical")
        self.assertEqual(manifest["trigger_fixture"]["runtime_activation"], "not_run")

        cases = manifest["scenarios"]
        self.assertEqual(len(cases), len(SCENARIO_IDS))
        self.assertEqual({case["id"] for case in cases}, SCENARIO_IDS)
        for case in cases:
            self.assertEqual(case["agent_observation"], "not_run", f"{case['id']}: observation must be not_run")
            for key in ("context", "request", "expected_action", "forbidden_effects", "rules"):
                self.assertTrue(case.get(key), f"{case['id']}: missing {key}")
            self.assertTrue(set(case["rules"]) <= set(RULES), f"{case['id']}: unknown rule")
        self.assertEqual({rule for case in cases for rule in case["rules"]}, set(RULES))

        triggers = manifest["trigger_cases"]
        self.assertEqual({case["id"] for case in triggers}, TRIGGER_IDS)
        for case in triggers:
            self.assertEqual(case["agent_observation"], "not_run", f"{case['id']}: observation must be not_run")
            self.assertIn(case["expected_route"], {"research", "no_research"}, case["id"])
            self.assertTrue(case["prompt"], f"{case['id']}: missing prompt")
            self.assertTrue(case["reason"], f"{case['id']}: missing reason")
            self.assertTrue(set(case["rules"]) <= set(RULES), f"{case['id']}: unknown rule")
        self.assertEqual({case["expected_route"] for case in triggers}, {"research", "no_research"})

    def manifest(self):
        path = Path(__file__).parent / "research-workflow-scenarios.json"
        return json.loads(path.read_text(encoding="utf-8"))

    # --- canonical research skill -------------------------------------------

    def test_research_skill_states_the_contract(self):
        for rule in (
            "research-trigger",
            "research-return",
            "no-auto-docs",
            "effort-levels",
            "bounded-brief",
            "local-first",
            "privacy-external-query",
            "untrusted-content",
            "handoff-fields",
            "approval-preservation",
            "no-subagents-default",
            "research-md-conditional",
        ):
            with self.subTest(rule=rule):
                self.assert_rule(rule)

    def test_preserved_008_clauses_are_verbatim(self):
        text = self.flat(RESEARCH)
        for clause in PRESERVED_CLAUSES:
            self.assertEqual(text.count(clause), 1, f"preserved 008 clause must appear exactly once: {clause}")

    def test_handoff_fields_appear_in_contract_order(self):
        text = self.flat(RESEARCH)
        marker = SEVEN_FIELDS[0]
        self.assertEqual(text.count(marker), 1, "research skill: exactly one handoff template")
        start = text.find(marker)
        template = text[start:start + 800]
        positions = [template.find(field) for field in SEVEN_FIELDS]
        for field, position in zip(SEVEN_FIELDS, positions):
            self.assertGreaterEqual(position, 0, f"research skill: missing handoff field {field!r}")
        self.assertEqual(positions, sorted(positions), "research skill: handoff fields are out of contract order")

    def test_claim_labels_and_privacy_boundaries_are_explicit(self):
        reference = self.flat(REFERENCE)
        for label in CLAIM_LABELS:
            self.assertIn(label, reference, f"reference: missing claim label {label!r}")

        research = self.flat(RESEARCH)
        for pattern in (
            r"private (?:repository |repo )?code",
            r"\btokens?\b",
            r"customer data",
            r"internal (?:hostnames?|hosts?)",
            r"private paths?",
            r"without [^.]{0,60}authoriz",
            r"untrusted",
            r"never (?:change|alter|override)",
        ):
            self.assertIsNotNone(re.search(pattern, research, re.I), f"research skill: missing privacy/untrusted clause {pattern}")

    def test_effort_levels_are_a_heuristic_not_a_permission(self):
        text = self.flat(RESEARCH)
        for level in EFFORT_LEVELS:
            self.assertIsNotNone(re.search(rf"\b{level}\b", text), f"research skill: missing effort level {level}")
        self.assertIsNotNone(
            re.search(r"decision heuristic|not (?:a )?(?:new )?(?:phase|stage|permission)", text, re.I),
            "research skill: effort levels must be described as a heuristic, not a phase or permission",
        )

    def test_research_md_is_conditional_only(self):
        text = self.flat(RESEARCH)
        self.assertIn("research.md", text)
        self.assertIsNotNone(
            re.search(r"research\.md[^.]{0,200}(?:only|conditional|unless)", text, re.I),
            "research skill: research.md must be conditional, not automatic",
        )
        self.assertIsNone(
            re.search(r"(?:always|must|required to) create (?:a )?research\.md", text, re.I),
            "research skill: must not require research.md",
        )

    def test_description_is_trigger_focused_and_not_generic_research(self):
        text = self.source(RESEARCH)
        match = re.search(r'(?m)^description:\s*"?(.+?)"?\s*$', text)
        self.assertIsNotNone(match, "research skill: missing description")
        description = match.group(1)
        self.assertRegex(description, r"(?i)(?:named|specific)[^.]{0,60}(?:fact|uncertainty|question)")
        self.assertRegex(description, r"(?i)block|unblock|resolve")
        self.assertRegex(description, r"(?i)not for|not (?:a )?generic", "description must exclude generic research")
        self.assertRegex(description, r"(?i)generic research", "description must name the generic-research collision")
        self.assertRegex(description, r"(?i)background delegation|mandatory", "description must reject delegation and mandatory files")

    def test_reference_is_linked_from_the_research_skill(self):
        text = self.source(RESEARCH)
        self.assertIn("references/search-mcp-workflow.md", text)
        self.assert_links(RESEARCH)
        self.assertTrue(REFERENCE.is_file(), "missing packaged reference: search-mcp-workflow.md")

    # --- packaged reference -------------------------------------------------

    def test_reference_covers_capability_and_claim_discipline(self):
        for rule in (
            "capability-matrix",
            "exa-discipline",
            "context7-resolution",
            "version-mismatch",
            "optional-site-tools",
            "known-url-direct",
            "unavailable-fallback",
            "agent-run-not-implicit",
            "claim-labels",
            "snippet-not-evidence",
            "runtime-facts",
            "stop-or-report-gap",
        ):
            with self.subTest(rule=rule):
                self.assert_rule(rule)

    def test_reference_does_not_claim_installed_providers(self):
        text = self.flat(REFERENCE)
        self.assertIsNotNone(
            re.search(r"(?:not|never)[^.]{0,80}(?:installed|connected|available)|runtime fact", text, re.I),
            "reference: must not assert installation or reachability from this text",
        )

    # --- stage pointers -----------------------------------------------------

    def test_stage_pointers_link_to_the_canonical_skill_and_state_their_boundary(self):
        for path in STAGE_FILES:
            with self.subTest(stage=self.label(path)):
                self.assert_links(path)
                text = self.source(path)
                targets = [
                    target for target in re.findall(r"\[[^\]]*\]\(([^)\s]+)\)", text)
                    if "agentic-sdd-research-spec" in target
                ]
                self.assertTrue(targets, f"{self.label(path)}: no link to the canonical research skill")
                resolved = {(path.parent / target.partition("#")[0]).resolve() for target in targets}
                self.assertIn(RESEARCH.resolve(), resolved, f"{self.label(path)}: link does not resolve to the research skill")
        for rule in STAGE_RULES:
            with self.subTest(rule=rule):
                self.assert_rule(rule)

    def test_stage_pointers_do_not_duplicate_the_shared_contract(self):
        markers = (
            "Question / affected SDD decision",
            "Contradiction / caveat",
            "verified primary",
            "corroborated",
            "reported only",
        )
        for path in STAGE_FILES:
            text = self.source(path)
            for marker in markers:
                self.assertNotIn(marker, text, f"{self.label(path)}: duplicates the shared research contract: {marker}")
            for level in EFFORT_LEVELS:
                self.assertIsNone(
                    re.search(rf"\b{level}\b", text),
                    f"{self.label(path)}: must not repeat the R0-R3 effort table",
                )

    # --- policy and librarian ----------------------------------------------

    def test_policy_and_librarian_carry_precedence_privacy_and_untrusted_rules(self):
        for rule in ("policy-precedence", "policy-external-privacy", "librarian-untrusted", "librarian-handoff"):
            with self.subTest(rule=rule):
                self.assert_rule(rule)

    def test_librarian_keeps_its_pinned_model_and_denied_permissions(self):
        text = self.source(LIBRARIAN)
        self.assert_rule("librarian-permissions")
        for action in ("subagent", "shell", "edit"):
            self.assertRegex(
                text,
                rf'action: {action}\s*\n\s*resource: "\*"\s*\n\s*effect: deny',
                f"librarian: {action} must stay denied",
            )
        self.assertNotIn("agent_run", text, "librarian: agent_run must not be granted or advertised")
        self.assertNotRegex(text, r"(?i)tavily", "librarian: Tavily access must not be granted or advertised")

    # --- negative mutations -------------------------------------------------

    def test_each_required_rule_removal_is_detected(self):
        for rule, (path, patterns) in RULES.items():
            with self.subTest(rule=rule):
                text = self.source(path)
                mutated, count = re.subn(patterns[0], "[removed clause]", text, flags=re.I)
                self.assertGreater(count, 0, f"{rule}: seed must change actual source")
                with self.assertRaises(AssertionError):
                    self.assert_rule(rule, {path: mutated})

    def test_softened_privacy_untrusted_snippet_approval_and_stage_return_are_detected(self):
        seeds = (
            ("privacy-external-query", RESEARCH, r"private (?:repository |repo )?code", "project material"),
            ("policy-external-privacy", POLICY, r"internal (?:hostnames?|hosts?)", "internal notes"),
            ("untrusted-content", RESEARCH, r"untrusted", "external"),
            ("librarian-untrusted", LIBRARIAN, r"untrusted", "external"),
            ("snippet-not-evidence", REFERENCE, r"snippet", "search summary"),
            ("approval-preservation", RESEARCH, r"never starts? a batch", "may start a batch"),
            (
                "research-return",
                RESEARCH,
                r"Return to the interrupted stage after answering it",
                "Return to a suitable stage after answering it",
            ),
        )
        for rule, path, pattern, replacement in seeds:
            with self.subTest(rule=rule):
                text = self.source(path)
                mutated, count = re.subn(pattern, replacement, text, flags=re.I)
                self.assertGreater(count, 0, f"{rule}: seed must change actual source")
                with self.assertRaises(AssertionError):
                    self.assert_rule(rule, {path: mutated})

    def test_verification_docs_are_not_tests_softening_is_detected(self):
        text = self.source(VERIFICATION)
        mutated, count = re.subn(r"executed", "reported", text, flags=re.I)
        self.assertGreater(count, 0, "stage-verification-readonly: seed must change actual source")
        with self.assertRaises(AssertionError):
            self.assert_rule("stage-verification-readonly", {VERIFICATION: mutated})

    # --- declarative manifest ----------------------------------------------

    def test_mocked_private_query_is_not_dispatched_or_redacted_at_runtime(self):
        case = next(case for case in self.manifest()["scenarios"] if case["id"] == "credential-private-path-query")
        mocked = case["mocked_query"]
        draft = mocked["draft"]
        for marker in ("<redacted-token>", "internal.example.invalid", "private/repo"):
            self.assertIn(marker, draft, f"mocked query must demonstrate {marker}")
        self.assertEqual(mocked["dispatch"], "not_run", "no external dispatch is claimed")
        self.assertEqual(mocked["redaction"], "not_run", "no runtime redaction is claimed")
        self.assertEqual(case["agent_observation"], "not_run")

    def test_scenario_manifest_covers_required_cases_without_model_claims(self):
        self.assert_manifest(self.manifest())

    def test_manifest_mutations_are_rejected(self):
        base = self.manifest()
        missing_case = copy.deepcopy(base)
        missing_case["scenarios"].pop()
        unknown_rule = copy.deepcopy(base)
        unknown_rule["scenarios"][0]["rules"].append("invented-rule")
        invented_pass = copy.deepcopy(base)
        invented_pass["scenarios"][0]["agent_observation"] = "pass"
        missing_trigger = copy.deepcopy(base)
        missing_trigger["trigger_cases"].pop()
        invented_trigger_pass = copy.deepcopy(base)
        invented_trigger_pass["trigger_cases"][0]["agent_observation"] = "passed"
        for mutation in (missing_case, unknown_rule, invented_pass, missing_trigger, invented_trigger_pass):
            with self.assertRaises(AssertionError):
                self.assert_manifest(mutation)

    def test_handoff_cases_validate_the_seven_field_contract(self):
        cases = {case["id"]: case for case in self.manifest()["scenarios"]}
        valid = cases["valid-seven-field-handoff"]["handoff"]
        invalid = cases["invalid-seven-field-handoff"]["handoff"]

        self.assertEqual(valid["expected_outcome"], "accepted")
        self.assertEqual(tuple(valid["fields"]), SEVEN_FIELDS)
        self.assertTrue(all(value.strip() for value in valid["fields"].values()), "valid handoff fields must not be empty")

        self.assertEqual(invalid["expected_outcome"], "rejected")
        self.assertTrue(invalid["missing"], "invalid handoff must name the missing fields")
        self.assertEqual(set(invalid["fields"]) | set(invalid["missing"]), set(SEVEN_FIELDS))
        self.assertTrue(set(invalid["fields"]).isdisjoint(invalid["missing"]))
        for field in invalid["missing"]:
            self.assertNotIn(field, invalid["fields"])

    def test_trigger_cases_cover_positive_and_negative_prompts(self):
        triggers = self.manifest()["trigger_cases"]
        positives = [case for case in triggers if case["expected_route"] == "research"]
        self.assertGreaterEqual(len(positives), 2, "need at least two positive trigger cases")
        collision = next(case for case in triggers if case["id"] == "trigger-negative-generic-research")
        self.assertIn("research", collision["prompt"].lower(), "generic-research collision prompt must name research")
        self.assertEqual(collision["expected_route"], "no_research")
        non_sdd = next(case for case in triggers if case["id"] == "trigger-negative-non-sdd-request")
        self.assertEqual(non_sdd["expected_route"], "no_research")


if __name__ == "__main__":
    unittest.main()
