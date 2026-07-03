#!/usr/bin/env python3
"""Check a release-gate decision against the firewall + per-stage required evidence — pure stdlib.

Usage: check-release-gate.py <release-gate.json>
Firewall (Option A): a PASS at production_candidate/production requires executed eval results +
behavior_evaluated. CONDITIONAL_PASS at those stages is allowed (condition: run evals before
production). Also verifies the required-evidence items for the requested stage are present when the
decision is PASS. Exit 0 if consistent, 1 otherwise.
"""
from __future__ import annotations
import json, sys

LATER = {"production_candidate", "production"}
# Required evidence for a PASS at each stage (mirrors release-gate-template's matrix).
REQUIRED_FOR_PASS = {
    "prototype": ["runnable_artifact"],
    "pilot": ["runnable_artifact", "agent_spec", "io_contracts", "guardrails", "review_scorecard", "eval_plan"],
    # production CANDIDATE = ready to deploy: behavior proven + readiness specs/runbook (not live controls).
    "production_candidate": ["runnable_artifact", "agent_spec", "io_contracts", "guardrails",
                              "review_scorecard", "eval_plan", "executed_eval_results", "security_review",
                              "monitoring_plan", "rollback_runbook", "human_approval_model"],
}
# PRODUCTION = candidate readiness + LIVE controls (monitoring live, rollback exercised, canary done).
REQUIRED_FOR_PASS["production"] = (REQUIRED_FOR_PASS["production_candidate"]
                                   + ["monitoring", "rollback_kill_switch", "canary"])

def check(g: dict):
    problems = []
    stage = g.get("requested_stage"); decision = g.get("decision")
    ev = g.get("evidence", {}) or {}
    if decision not in {"PASS", "CONDITIONAL_PASS", "FAIL"}:
        problems.append("decision must be PASS | CONDITIONAL_PASS | FAIL.")
    if g.get("blockers") and decision == "PASS":
        problems.append("PASS with open blockers is inconsistent.")
    if decision == "PASS" and stage in LATER:
        if not ev.get("executed_eval_results"):
            problems.append(f"FIREWALL: PASS at {stage} requires evidence.executed_eval_results=true.")
        if not g.get("behavior_evaluated"):
            problems.append(f"FIREWALL: PASS at {stage} requires behavior_evaluated=true.")
    if decision == "PASS" and stage in REQUIRED_FOR_PASS:
        missing = [k for k in REQUIRED_FOR_PASS[stage] if not ev.get(k)]
        if missing:
            problems.append(f"PASS at {stage} is missing required evidence: {', '.join(missing)}.")
    return (len(problems) == 0, problems)

def suggest(g: dict) -> str:
    stage = g.get("requested_stage"); ev = g.get("evidence", {}) or {}
    if g.get("blockers"): return "FAIL (open blockers)"
    if stage in LATER and not ev.get("executed_eval_results"):
        return "CONDITIONAL_PASS (condition: run + pass the behavioral eval suite before production)"
    return g.get("decision", "—")

def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__); return 2
    g = json.loads(open(sys.argv[1]).read())
    ok, problems = check(g)
    print(f"Agent: {g.get('agent')} v{g.get('agent_version')}  stage={g.get('requested_stage')}  "
          f"decision={g.get('decision')}  design={g.get('design_score')}")
    if ok:
        print("OK  decision is firewall- and evidence-consistent.")
    else:
        print("FAIL decision is inconsistent:")
        for p in problems: print("  -", p)
        print(f"  suggested decision: {suggest(g)}")
    return 0 if ok else 1

if __name__ == "__main__":
    sys.exit(main())
