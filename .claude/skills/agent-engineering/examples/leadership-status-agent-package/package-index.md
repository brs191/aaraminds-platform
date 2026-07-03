# leadership-status-agent — package index

> Worked example of a full `agent-engineering` Create/Review output. Illustrative; behavior NOT
> run-tested.

- Status: pilot-candidate · Design score: 77/100 (useful prototype)
- Release decision: CONDITIONAL PASS (pilot) / FAIL (production candidate) · Owner: R. Bhupathiraju

## Artifacts
| File | Purpose |
|---|---|
| agent.md | Runnable pointer (→ `aara-status-deck`) |
| AGENT_SPEC.md | Descriptive contract |
| agent-card.json | A2A interop card |
| eval-plan.md | Behavior contract + golden cases (not executed) |
| review-scorecard.md | Design score + F-001…F-004 findings |
| release-gate.json | Staged decision (validates with check-release-gate.py) |
| tool-risk-register.md | Tool risks (F-002 Bash) |
| improvement-backlog.md | P0–P3 fixes |

## Agent Engineering Result (summary)
- **Top strengths:** earned problem; strong contracts; reuses a validated persona.
- **Top risks:** F-001 evals never run; F-002 unscoped Bash; F-003 advisory HITL; F-004 no monitoring.
- **Recommendation:** clear F-002/F-003 for a clean pilot; run the eval plan to pursue a production-
  candidate PASS. This package is the template for the 3-agent dogfood.
