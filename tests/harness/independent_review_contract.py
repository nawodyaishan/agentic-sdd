"""Test-only reference consumer for the independent-review report contract (C3).

Used only by offline fixtures in test_independent_review_fixtures.py. It is not
shipped, not a product parser and not an automated Ready setter. It applies the
most conservative reading of the contract to scripted reports, so the fixtures can
show that a bad report is not relied on. It says nothing about whether a model
produces or obeys such a report.
"""

RULES = (
    "identity", "execution", "evidence", "violations", "malformed",
    "parent-disposition", "valid-findings", "needs-evidence",
)
DISPOSITIONS = ("valid", "disputed", "needs evidence")
FINDING_FIELDS = ("id", "severity", "impact", "location", "evidence")


def reconcile(report, parent, current_identity, skip=frozenset()):
    """Return the effective verdict a parent may rely on, with reasons.

    Effective values are Ready, Needs changes, Blocked or Stale. Only a current,
    completed, violation-free report whose findings are all dispositioned without
    an open valid or needs-evidence item can keep Ready. A disputed finding does not
    block; it stays visible through the parent's rationale.
    """
    if "identity" not in skip and report.get("identity") != current_identity:
        return {"effective": "Stale", "reasons": ["candidate or evidence changed after the review identity"]}

    reasons = []
    blocked = False
    if "execution" not in skip and report.get("execution") != "completed":
        reasons.append(f"execution {report.get('execution')}")
        blocked = True
    if "evidence" not in skip and report.get("evidence") != "current":
        reasons.append(f"evidence {report.get('evidence')}")
        blocked = True
    if "violations" not in skip and report.get("violations"):
        reasons.append("prohibited operation attempted")
        blocked = True

    downgrade = False
    for finding in report.get("findings") or []:
        if "malformed" not in skip and not all(finding.get(field) for field in FINDING_FIELDS):
            reasons.append(f"malformed finding {finding.get('id', '?')}")
            blocked = True
            continue
        disposition = parent.get(finding.get("id"))
        if "parent-disposition" not in skip and disposition not in DISPOSITIONS:
            reasons.append(f"no parent disposition for {finding.get('id')}")
            blocked = True
        if "needs-evidence" not in skip and disposition == "needs evidence":
            reasons.append(f"{finding['id']} needs evidence")
            blocked = True
        if "valid-findings" not in skip and disposition == "valid":
            reasons.append(f"{finding['id']} valid and open")
            downgrade = True

    reported = report.get("verdict")
    if reported is None:
        reasons.append("no verdict")
        blocked = True

    if blocked:
        return {"effective": "Blocked", "reasons": reasons}
    if reported == "Ready" and downgrade:
        return {"effective": "Needs changes", "reasons": reasons}
    return {"effective": reported, "reasons": reasons}


def plan_launch(history, request):
    """Decide whether one reviewer may launch. Returns (launch, reason).

    history items: identity, execution, active. A request must be explicit; the
    harness never launches or retries a reviewer by itself.
    """
    if any(item.get("active") for item in history):
        return False, "a reviewer is already active"
    if not request.get("explicit"):
        return False, "no explicit user request; no implicit launch or automatic retry"
    if any(item["identity"] == request["identity"] and item["execution"] == "completed" for item in history):
        return False, "identical candidate already reviewed; keep the existing result"
    return True, "explicit request for a new or changed candidate"
