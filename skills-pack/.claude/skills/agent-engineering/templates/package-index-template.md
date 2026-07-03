# Template — agent package index

The manifest of a delivered agent package. Create mode emits all of these (a simple agent may keep the
spec sections in one file; split when complexity warrants).

```md
# {{agent-name}} — package index

- Status: draft | pilot-candidate | production-candidate | production
- Design score: {{NN}}/100 ({{band}})   - Release decision: PASS | CONDITIONAL PASS | FAIL
- Owner:   - Version:   - Last reviewed:

## Artifacts
| File | Purpose |
|---|---|
| agent.<md/agent.md/toml> | Runnable agent file (target format) |
| AGENT_SPEC.md | Descriptive contract (role, scope, contracts, guardrails, security, deployment) |
| agent-card.json | A2A machine-readable interop card |
| eval-plan.md + golden-dataset | Behavior contract + cases |
| review-scorecard.md | Design score + findings + backlog |
| release-gate.md | Staged go/no-go decision record |
| improvement-backlog.md | P0–P3 fixes |
| AGENTS.md | (if repo-resident) project-instruction companion |

## Deliverables produced this run
- [ ] Runnable file (valid for target)
- [ ] AGENT_SPEC.md   - [ ] agent-card.json
- [ ] eval-plan + golden dataset
- [ ] review scorecard (+ [VERIFY] list)
- [ ] release-gate decision
- [ ] improvement backlog
```

## Final response format (what the agent returns to the user)
```md
# Agent Engineering Result
## Executive summary
## Files created / updated
## Agent readiness score (design /100 + behavior status)
## Release decision (PASS | CONDITIONAL PASS | FAIL, requested stage)
## Top strengths
## Top risks
## Required fixes (P0–P3)
## How to use
```
