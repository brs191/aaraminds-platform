#!/usr/bin/env python3
"""Validate the agent-engineering JSON schemas + run firewall negative tests.

Stdlib-only by default; uses `jsonschema` for full validation if installed
(`pip install jsonschema --break-system-packages`). Exit 0 if all checks pass, 1 otherwise.
"""
import json, sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SCHEMA_DIR = HERE.parent / "schemas"
SCHEMAS = ["eval-case", "eval-result", "trace-review", "release-gate"]

# (schema, instance, should_validate) — firewall negatives must FAIL validation.
CASES = [
    ("release-gate", {"agent": "x", "agent_version": "1", "requested_stage": "production_candidate",
                      "decision": "PASS", "design_score": 92, "behavior_evaluated": False,
                      "evidence": {"executed_eval_results": False}}, False),
    ("release-gate", {"agent": "x", "agent_version": "1", "requested_stage": "pilot",
                      "decision": "CONDITIONAL_PASS", "design_score": 77}, True),
    ("eval-result", {"case_id": "A-001", "agent": "x", "agent_version": "1",
                     "executed": False, "passed": True, "grader": "judge"}, False),
    ("eval-result", {"case_id": "A-001", "agent": "x", "agent_version": "1",
                     "executed": True, "passed": True, "grader": "code"}, True),
]

def main() -> int:
    ok = True
    schemas = {}
    for name in SCHEMAS:
        p = SCHEMA_DIR / f"{name}.schema.json"
        try:
            schemas[name] = json.loads(p.read_text())
            print(f"OK   valid JSON: {name}.schema.json")
        except Exception as e:
            print(f"FAIL invalid JSON: {name}.schema.json — {e}"); ok = False

    try:
        from jsonschema import Draft202012Validator
    except ImportError:
        print("\nNOTE: install `jsonschema` for full schema + firewall validation "
              "(pip install jsonschema --break-system-packages). Skipped negative tests.")
        return 0 if ok else 1

    for name, sch in schemas.items():
        try:
            Draft202012Validator.check_schema(sch)
            print(f"OK   schema compiles: {name}")
        except Exception as e:
            print(f"FAIL schema error: {name} — {e}"); ok = False

    print("\nFirewall negative/positive tests:")
    for name, inst, should in CASES:
        v = Draft202012Validator(schemas[name])
        valid = not list(v.iter_errors(inst))
        status = "OK  " if valid == should else "FAIL"
        if valid != should: ok = False
        print(f"{status} {name}: expected {'valid' if should else 'REJECTED'}, got {'valid' if valid else 'rejected'}")
    return 0 if ok else 1

if __name__ == "__main__":
    sys.exit(main())
