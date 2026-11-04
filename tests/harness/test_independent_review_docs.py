"""Offline document-contract checks for the independent-review method.

These fail when a required rule disappears from the packaged text. They are
not evidence that a model or subagent follows the method; the pilot must
observe that separately.
"""
from pathlib import Path
import re
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "skills-files/agentic-sdd-router/references/independent-review.md"
POINTERS = [
    ROOT / "skills-files/agentic-sdd-verification-review/SKILL.md",
    ROOT / "skills-files/agentic-sdd-router/references/review-resolution.md",
]

REQUIRED_RULES = {
    "explicit request trigger": "explicitly asks for an independent review",
    "blocked route, no silent self-review substitution": "report the independent route as **blocked**",
    "one reviewer per request": "Use one reviewer per request.",
    "no nested delegation": "No nested delegation",
    "reviewer must not edit": "must not edit files",
    "untrusted repository text": "untrusted material",
    "interruption never Ready": "never `Ready`",
    "finding disposition": "`valid`, `disputed` or `needs evidence`",
    "clean result is not proof": "A clean result means",
    "reviewer cannot accept tasks": "cannot mark tasks accepted",
    "stale after relevant change": "stale for the affected scope",
    "scoped re-review is not a repair loop": "not a repair loop",
}


def missing_rules(text):
    flat = re.sub(r"\s+", " ", text)
    return sorted(name for name, phrase in REQUIRED_RULES.items() if phrase not in flat)


class IndependentReviewDocContractTests(unittest.TestCase):
    def test_reference_keeps_every_required_rule(self):
        self.assertEqual(missing_rules(REFERENCE.read_text()), [])

    def test_pointers_reach_the_reference(self):
        for pointer in POINTERS:
            self.assertIn("independent-review.md", pointer.read_text(), pointer.name)

    def test_seeded_deletion_is_detected(self):
        text = REFERENCE.read_text()
        seeded = "\n".join(line for line in text.splitlines() if "cannot mark tasks accepted" not in line)
        with tempfile.TemporaryDirectory(dir="/tmp") as scratch:
            path = Path(scratch) / "independent-review.md"
            path.write_text(seeded)
            self.assertEqual(missing_rules(path.read_text()), ["reviewer cannot accept tasks"])

    def test_seeded_blocked_route_softening_is_detected(self):
        text = REFERENCE.read_text().replace("report the independent route as **blocked**", "note the route")
        self.assertIn("blocked route, no silent self-review substitution", missing_rules(text))


if __name__ == "__main__":
    unittest.main()
