# Dogfood — Review + Release Gate applied to `aara-status-deck`

Proof the skill surfaces real, specific gaps, and a live demonstration of the design-vs-behavior
firewall via the **staged release gate**. Reviewer: `aara-agent-engineer` (Review mode, v2 rubric).
Target: `aara-status-deck` v1.3 + `aaraminds-leadership-status-deck`.

## Scores (v2 rebalanced 100-point rubric)

| # | Dimension | Max | Score | Note |
|---|---|---:|---:|---|
| 1 | Problem fit | 10 | 9 | Clear recurring problem; agent earned (judgment + recurrence) |
| 2 | Role clarity | 10 | 9 | Owns monthly decks; explicit "not for briefs/memos/plans/external" |
| 3 | Scope boundaries | 10 | 8 | In/out scope explicit; **human-only decisions not framed** |
| 4 | Input contract | 8 | 7 | Mandatory + optional + gap-handling defined |
| 5 | Output contract | 8 | 8 | Six structured deliverables; evidence + verification reports |
| 6 | Workflow design | 10 | 8 | 7-step flow + verify checklist; **no max-turn / loop bound** |
| 7 | Tool & data safety | 10 | 6 | **Inherits unsandboxed `Bash`, no risk-tier/HITL** — excess agency; not full lethal trifecta |
| 8 | Guardrails & failure modes | 10 | 7 | Anti-watermelon, `[VERIFY]`, visual-QA, confidentiality; **failure modes not enumerated; HITL advisory** |
| 9 | Evaluation coverage | 12 | 7 | `evals.md`: 3 scenarios + scoring rubric, both-sided — **but never run** |
| 10 | Production readiness | 7 | 3 | Wired, versioned (v1.3); **no monitoring / rollback / kill-switch** |
| 11 | Executive usability | 5 | 5 | Produces presentation-grade exec output |
| | **Total** | **100** | **77** | Band: **useful prototype (70–79)** |

> Note: 77 under v2 is intentionally below the 81 the old rubric gave — v2 weights **Evaluation (12)**
> and **Production Readiness (7)** higher, and those are exactly where this agent is weakest (unrun
> evals, no monitoring). The rebalance is stricter on what actually makes an agent trustworthy.

## Hard gates
- Eval strategy present → gate 1 not tripped. Guardrails present → gate 2 not tripped.
- **Never run against test cases → gate 3 TRIPPED: cannot be "production-ready."**
- No fabricated metrics/owners → no hard finding. Bash breadth flagged (F-002), not an auto-fail.

## Findings (severity-tagged)
- **F-001 (Major, Evaluation):** evals designed but never executed — blocks production readiness.
- **F-002 (Major, Tool & Data Safety):** unsandboxed `Bash` inherited, no risk-tier or HITL — scope the
  tool list or sandbox it.
- **F-003 (Minor, Guardrails):** failure modes not enumerated; pre-leader human review is advisory, not
  an enforced checkpoint.
- **F-004 (Minor, Workflow/Readiness):** no max-turn bound; no rollback/kill-switch/monitoring.

## Release Gate decision (staged)

| Requested stage | Decision | Why |
|---|---|---|
| **Pilot** | **CONDITIONAL PASS** | Has the artifact, contracts, guardrails, scorecard, and eval *plan*; executed results only *recommended* at pilot. Conditions: fix F-002 (tool scope), enumerate failure modes (F-003). |
| **Production candidate** | **FAIL** | Executed eval results are **Required** and absent; monitoring + rollback/kill-switch absent. The firewall blocks it. |

```
Agent: aara-status-deck (v1.3)   Design Score: 77/100 (useful prototype)
Behavior: NOT YET RUN
Release decision: CONDITIONAL PASS (pilot) · FAIL (production candidate)
Recommendation: run the eval plan (3 scenarios + the AT&T real-deck case), read transcripts; scope the
tool list; enumerate failure modes + enforce the pre-leader review. That clears pilot conditions and is
the only path to a production-candidate PASS.
```

## Why this is the right result

A strong, four-rounds-deep design still lands at "useful prototype, pilot-conditional, production-FAIL"
— because it has never been run and lacks operational controls. That is the firewall + staged gate
working exactly as intended: design quality never substitutes for executed evidence. Same conclusion
the whole build keeps reaching — **stop reviewing on paper, run the thing.**
