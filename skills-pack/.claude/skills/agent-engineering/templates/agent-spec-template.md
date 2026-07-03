# Template — AGENT_SPEC.md (the descriptive contract)

Every generated or reviewed agent must satisfy this contract (model-card → system-card lineage). Fill
in; missing sections are findings, not silent gaps. Pair with `agent-card.json` (A2A interop — see
`references/agent-package-contract.md`).

```md
## 1. Identity
- Agent name:   - Version:   - Owner:   - Runtime target:
- Status: draft | pilot-candidate | production-candidate | production   - Last reviewed:

## 2. Business purpose
- Business problem:   - Target users:   - Job-to-be-done (beneficiary · task/outcome · measurable value):
- Expected measurable improvement:
- Why an AI agent is justified (vs single call / workflow):

## 3. Scope boundary
- In scope:
- Out of scope:                       # the agent must explicitly state what it will NOT do
- Human-only decisions:               # irreversible / financial / policy / scope — agent must not decide alone

## 4. Input contract
| Input | Type | Required | Validation | Missing-input behavior |
(+ optional inputs with defaults)

## 5. Output contract
- Structured output schema / shape:   - Output modes:   - Downstream consumer:

## 6. Tools + permissions
| Tool | Purpose | Risk tier (read=low / write/irreversible/financial=high) | Scope (least-privilege) | Guardrail |

## 7. Workflow & orchestration
- Pattern (single / chaining / routing / orchestrator-workers / evaluator-optimizer):
- Decision points:   - Stopping conditions (max turns/retries):   - Escalation path:

## 8. Guardrails & HITL
- Input / output / tool-level guardrails (typed: block | flag | confirm), at the side effect:
- Human-approval points (high-risk/irreversible/financial/low-confidence/lethal-trifecta):
- Lethal-trifecta analysis (private data + untrusted content + external comms → gate one leg):

## 9. Failure modes
| Failure mode | Detection | Mitigation | Escalation |

## 10. Evaluation
- Functional / behavioral / safety cases (happy + edge + ≥1 adversarial); golden set size; CI gate.
  (Detail in eval-plan-template.md.)

## 11. Security & governance (enterprise)
- OWASP Agentic exposure (ASI01–10) noted:   - Scoped non-human identity:   - Audit/tracing:
- Data-governance / retention (if regulated):

## 12. Deployment & monitoring
- Versioning:   - Rollback/disable path:   - Kill switch (tested):   - Canary:
- Monitoring thresholds (error/cost/policy-violation/drift):   - Budget caps:

## 13. Limitations & residual risks
```
