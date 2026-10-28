"""Development schema checks only; not a runtime or authorization validator."""

import copy
from datetime import datetime
import json
from pathlib import Path
import re

from jsonschema import Draft202012Validator, FormatChecker


ROOT = Path(__file__).resolve().parents[2]
FORMATS = FormatChecker()


@FORMATS.checks("date-time")
def date_time(value):
    # jsonschema's optional RFC3339 dependency is not present on every host.
    if not isinstance(value, str):
        return True
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[Zz]|[+-]\d{2}:\d{2})", value):
        return False
    try:
        datetime.fromisoformat(value.upper().replace("Z", "+00:00"))
        return True
    except ValueError:
        return False


def load(relative):
    return json.loads((ROOT / relative).read_text())


def exercise(schema_path, document_path, cases):
    schema = load(schema_path)
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema, format_checker=FORMATS)
    document = load(document_path)
    validator.validate(document)
    for case in cases:
        changed = copy.deepcopy(document)
        parent = changed
        for key in case["path"][:-1]:
            parent = parent[key]
        parent[case["path"][-1]] = case["value"]
        valid = validator.is_valid(changed)
        if valid != case["valid"]:
            raise AssertionError(f"{case['name']}: expected valid={case['valid']}, got {valid}")


if __name__ == "__main__":
    fixtures = load("tests/harness/contracts.json")
    exercise("harness/checks.schema.json", "harness/checks.json", fixtures["check_schema_cases"])
    exercise("harness/evidence/receipt.schema.json", "harness/evidence/example-direct-fix.json", fixtures["receipt_schema_cases"])
    print("PASS: both schemas and examples; 12 positive/negative schema cases.")
    print("NOT RUN: semantic, process, model-behavior and native boundary scenarios.")
