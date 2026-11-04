"""Offline static checks for the inactive independent-review OpenCode example.

Expectations are written here independently of the example. These checks do not
run OpenCode, call a model or prove native enforcement; see native_independent_review.py.
"""
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
ADAPTER = ROOT / "integrations/opencode"
EXAMPLE = ADAPTER / "independent-review.jsonc"

COORDINATOR = "sdd-review-coordinator"
REVIEWER = "sdd-independent-reviewer"
MODEL = "opencode/big-pickle"
VARIANT = "default"
STEPS = {COORDINATOR: 4, REVIEWER: 6}
SKILL = "agentic-sdd-verification-review"
DENIED_ACTIONS = {
    "edit", "shell", "execute", "question", "webfetch", "websearch", "grep", "glob",
    "browser", "external_directory", "opencode_list_mcp_resources", "opencode_read_mcp_resource",
    "synthetic_write", "chrome_devtools_evaluate_script", "mcp", "plugin", "provider", "task",
}
SENSITIVE_READS = {"*.env*", "*.db*", "*.pem", ".git/*", "secrets/*", "*/secrets/*", ".opencode/*"}


def allows(permissions):
    return {(p["action"], p["resource"]) for p in permissions if p["effect"] == "allow"}


class IndependentReviewExampleTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(EXAMPLE.is_file(), "independent-review example is missing")
        self.config = json.loads(EXAMPLE.read_text())
        self.agents = self.config["agents"]

    def test_example_is_inactive_and_has_no_providers_or_commands(self):
        for key in ("providers", "plugins", "mcp", "references", "instructions", "commands", "default_agent", "model"):
            self.assertNotIn(key, self.config)
        self.assertFalse(self.config["warming"])
        self.assertFalse(self.config["formatter"])
        self.assertFalse(self.config["snapshots"])
        self.assertEqual(self.config["permissions"], [{"action": "*", "resource": "*", "effect": "deny"}])

    def test_top_level_policies_deny_providers_integrations_and_execution(self):
        policies = self.config["policies"]
        for expected in (
            {"action": "provider.use", "resource": "*", "effect": "deny"},
            {"action": "integration.use", "resource": "mcp:*", "effect": "deny"},
            {"action": "integration.use", "resource": "plugin:*", "effect": "deny"},
            {"action": "tool.use", "resource": "external_directory:*", "effect": "deny"},
            {"action": "tool.use", "resource": "execute:*", "effect": "deny"},
        ):
            self.assertIn(expected, policies)
        self.assertFalse(any(p["action"] in ("provider.use", "integration.use") and p["effect"] == "allow"
                             and not p["resource"].startswith("skill:") for p in policies))
        # The example must not carry a global subagent denial or allow. Collision
        # with an installed global denial is tested by the native probe.
        self.assertFalse(any(p["action"] == "tool.use" and p["resource"].startswith("subagent") for p in policies))

    def test_exactly_one_coordinator_and_one_reviewer_profile(self):
        self.assertEqual(set(self.agents), {COORDINATOR, REVIEWER})
        self.assertEqual(self.agents[COORDINATOR]["mode"], "primary")
        self.assertEqual(self.agents[REVIEWER]["mode"], "subagent")

    def test_both_profiles_bind_model_and_variant_explicitly_and_bound_steps(self):
        for name, agent in self.agents.items():
            self.assertEqual(agent["model"], MODEL, name)
            self.assertEqual(agent["variant"], VARIANT, name)
            self.assertEqual(agent["steps"], STEPS[name], name)

    def test_profiles_start_with_deny_and_contain_no_template_markers(self):
        for name, agent in self.agents.items():
            self.assertEqual(agent["permissions"][0], {"action": "*", "resource": "*", "effect": "deny"}, name)
            self.assertNotIn("__", json.dumps(agent), name)

    def test_only_named_skill_and_reads_are_allowed_for_both_profiles(self):
        for name, agent in self.agents.items():
            allowed = allows(agent["permissions"])
            actions = {action for action, _ in allowed}
            allowed_actions = {"read", "skill"} | ({"subagent"} if name == COORDINATOR else set())
            self.assertLessEqual(actions, allowed_actions, name)
            self.assertFalse(actions & DENIED_ACTIONS, name)
            skills = {resource for action, resource in allowed if action == "skill"}
            self.assertEqual(skills, {SKILL}, name)

    def test_only_the_coordinator_may_launch_exactly_the_reviewer(self):
        self.assertIn(("subagent", REVIEWER), allows(self.agents[COORDINATOR]["permissions"]))
        self.assertFalse(any(action == "subagent" for action, _ in allows(self.agents[REVIEWER]["permissions"])))
        for agent in self.agents.values():
            for action, resource in allows(agent["permissions"]):
                if action == "subagent":
                    self.assertEqual(resource, REVIEWER)

    def test_sensitive_reads_are_denied_for_both_profiles(self):
        for name, agent in self.agents.items():
            denied = {p["resource"] for p in agent["permissions"] if p["action"] == "read" and p["effect"] == "deny"}
            self.assertEqual(denied, SENSITIVE_READS, name)

    def test_no_active_default_or_lockdown_template_changed(self):
        for name in ("opencode.jsonc", "policy-lockdown.jsonc"):
            text = (ADAPTER / name).read_text()
            self.assertNotIn("independent", text, name)
            self.assertNotIn(REVIEWER, text, name)


if __name__ == "__main__":
    unittest.main()
