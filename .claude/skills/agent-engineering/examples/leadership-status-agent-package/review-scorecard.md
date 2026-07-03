# Agent Review Scorecard — Leadership Status Agent

Worked example. Reviewer: `aara-agent-engineer` (Review mode, v2 rubric). Static design review.

## Summary
- Agent: leadership-status-agent v1.3 · Review date: 2026-06-18
- Design score: **77/100** · Band: **useful prototype (70–79)**
- Release recommendation: **CONDITIONAL PASS (pilot) / FAIL (production candidate)**

## Score breakdown
| Category | Max | Score | Assessment |
|---|---:|---:|---|
| Problem fit | 10 | 9 | Clear recurring problem; agent earned |
| Role clarity | 10 | 9 | Explicit not-for list |
| Scope boundaries | 10 | 8 | Human-only decisions not framed |
| Input contract | 8 | 7 | Strong; minor gaps |
| Output contract | 8 | 8 | Six structured deliverables |
| Workflow design | 10 | 8 | 7-step flow; no max-turn bound |
| Tool & data safety | 10 | 6 | Unsandboxed Bash inherited, no risk-tier/HITL (F-002) |
| Guardrails & failure modes | 10 | 7 | Strong gates; HITL advisory (F-003) |
| Evaluation coverage | 12 | 7 | 3 scenarios + rubric — never run |
| Production readiness | 7 | 3 | No monitoring/rollback/kill-switch (F-004) |
| Executive usability | 5 | 5 | Presentation-grade output |
| **Total** | **100** | **77** | |

## Findings
- **F-001 (Major, Evaluation):** evals designed, never executed — blocks production readiness.
- **F-002 (Major, Tool & Data Safety):** unsandboxed Bash inherited, no risk-tier/HITL — scope it.
- **F-003 (Minor, Guardrails):** pre-leader review is advisory, not an enforced gate.
- **F-004 (Minor, Readiness):** no max-turn bound; no monitoring/rollback/kill-switch.

## Improvement backlog → improvement-backlog.md
## Release decision → release-gate.json
