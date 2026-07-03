#!/usr/bin/env python3
"""run-evals — the eval-runner ADAPTER CONTRACT (not a full executor).

This pack does NOT execute arbitrary agents itself; behavioral execution is delegated to
`aara-ai-evaluation-engineer` / `ai-evaluation-harness`. This script defines and validates the
contract at the boundary:
  - INPUT : a golden set of cases conforming to schemas/eval-case.schema.json
  - OUTPUT: results conforming to schemas/eval-result.schema.json
  - An `Executor` interface that a real harness implements (`run(case) -> result`).

Usage:
  run-evals.py --validate <golden-set.json>     # validate cases against the case schema (stdlib)
  run-evals.py --run <golden-set.json>          # requires a wired Executor; otherwise exits 3 (not wired)

Exit: 0 ok · 1 invalid cases · 2 usage · 3 no executor wired (expected until the harness is connected).
"""
from __future__ import annotations
import json, sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SCHEMA = HERE.parent / "schemas"

class Executor:
    """Implement this in the real harness (aara-ai-evaluation-engineer side)."""
    def run(self, case: dict) -> dict:
        raise NotImplementedError("No executor wired. Connect aara-ai-evaluation-engineer / a harness.")

EXECUTOR: Executor | None = None   # a real harness assigns this.

REQUIRED_CASE = {"id", "type", "category", "input", "grader", "should_fire"}

def validate(cases: list) -> list:
    errs = []
    for i, c in enumerate(cases):
        missing = REQUIRED_CASE - set(c)
        if missing:
            errs.append(f"case[{i}] ({c.get('id','?')}) missing: {', '.join(sorted(missing))}")
    return errs

def main() -> int:
    if len(sys.argv) != 3 or sys.argv[1] not in {"--validate", "--run"}:
        print(__doc__); return 2
    mode, path = sys.argv[1], sys.argv[2]
    data = json.loads(Path(path).read_text())
    cases = data if isinstance(data, list) else data.get("cases", [])
    errs = validate(cases)
    if errs:
        print("INVALID golden set:"); [print("  -", e) for e in errs]; return 1
    print(f"OK  {len(cases)} cases conform to eval-case schema.")
    if mode == "--validate":
        return 0
    if EXECUTOR is None:
        print("NO EXECUTOR WIRED — behavioral execution is delegated to aara-ai-evaluation-engineer.")
        print("Until wired, results cannot be produced and no agent may be marked run-tested (firewall).")
        return 3
    results = [EXECUTOR.run(c) for c in cases]      # each must conform to eval-result.schema.json
    print(json.dumps(results, indent=2))
    return 0

if __name__ == "__main__":
    sys.exit(main())
