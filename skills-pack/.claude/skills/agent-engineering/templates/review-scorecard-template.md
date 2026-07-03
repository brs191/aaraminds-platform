# Template — Agent Review Scorecard

The static design review. Firewalled from behavior (see `release-gate-template.md`). Uses the v2
rebalanced rubric (eval weighted highest; scope its own dimension; I/O contracts split).

## Summary
- Agent:   - Version:   - Review date:   - Reviewer:
- Design score:   /100   - Readiness band:   - Release recommendation: PASS | CONDITIONAL PASS | FAIL

## Score breakdown (100)

| Category | Max | Score | Assessment |
|---|---:|---:|---|
| Problem fit (earned the agent?) | 10 | | |
| Role clarity | 10 | | |
| Scope boundaries (in/out/human-only) | 10 | | |
| Input contract | 8 | | |
| Output contract | 8 | | |
| Workflow design (steps, stop conditions) | 10 | | |
| Tool & data safety (least-priv, trifecta) | 10 | | |
| Guardrails & failure modes | 10 | | |
| **Evaluation coverage** | **12** | | |
| Production readiness | 7 | | |
| Executive usability* | 5 | | |
| **Total** | **100** | | |

\* Conditional — waive for backend/infra agents and renormalize to 95.

## Readiness bands
90–100 production-ready **candidate** · 80–89 strong **pilot** candidate · 70–79 useful prototype ·
60–69 weak prototype · <60 redesign.

## Hard gates (override the total)
- No eval strategy **or** no guardrails/failure-mode handling → capped at prototype (≤79).
- Never run against test cases → cannot be "production-ready" (the design-vs-behavior firewall).
- Fabricated metric/owner/eval/capability → hard finding (resolve or `[VERIFY]`), not a deduction.
- Excessive agency or content-filter-as-injection-defense → automatic fail on Tool-Safety + Guardrails.

## Findings (severity: Blocker | Major | Minor | Observation)
Blocker = prevents pilot/safe use · Major = must fix before production · Minor = quality/usability ·
Observation = note, no action.

### F-001 — {{title}}
- Severity:   - Category (rubric dim):   - Evidence (file:line / quote):
- Why it matters:   - Required fix:   - Release impact:

## Top strengths / Top risks
Strengths: 1. 2. 3.   Risks: 1. 2. 3.

## Improvement backlog (P0–P3)

| Priority | Issue | Why it matters | Required fix | Owner | Required before |
|---|---|---|---|---|---|
| P0 | | | | | Pilot |
| P1 | | | | | Production |
| P2 | | | | | — |
| P3 | | | | | — |

Discipline: findings are **defect-shaped** and tied to a dimension + a file location — "no HITL on
`delete_workspace` (Tool & Data Safety, agent.md L40)," not "improve security."
