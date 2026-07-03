# aara-copilot-cost-reviewer — package index

- Status: pilot-ready (behavior-evaluated; integration-validated) · Design score: 86/100 (strong pilot candidate)
- Release decision: PASS (pilot) — 9/9 cases passed incl. live integration run vs the copilot-token-budget MCP (2026-06-18) / production-candidate pending rate-freshness check + readiness pkg · Owner: AaraMinds
- Built via the agent-engineering factory; method skill `copilot-cost-optimization`.

## Artifacts
| File | Purpose |
|---|---|
| ../../.claude/agents/aara-copilot-cost-reviewer.md | Runnable agent (wired) |
| AGENT_SPEC.md | Descriptive contract |
| agent-card.json | A2A interop card |
| eval-plan.md | Behavior contract + golden cases |
| tool-risk-register.md | Tool risks (sensitive billing data) |
| review-scorecard.md | Design score + findings |
| release-gate.json | Staged decision |

## Note
Built against the **AI-Credit** (usage-based) cost model that replaced premium requests on 2026-06-01.
All rates/allowances are version-sensitive and carried as [VERIFY] — the agent re-verifies each review.
