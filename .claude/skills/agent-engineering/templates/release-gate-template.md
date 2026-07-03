# Template — Agent Release Gate (staged)

The discrete go/no-go. This is where the design-vs-behavior firewall is *operationalized*: executed
eval results are **required** for production candidate, not for prototype. Decide whether the agent may
move to a requested stage.

## Identity
- Agent:   - Version:   - Requested stage: prototype | pilot | production candidate | production
- Reviewer:   - Date:   - Evidence reviewed:

## Required-evidence matrix

| Evidence | Prototype | Pilot | Production candidate | Status |
|---|---|---|---|---|
| Runnable agent artifact | Required | Required | Required | |
| AGENT_SPEC (spec contract) | Recommended | Required | Required | |
| Input / output contracts | Recommended | Required | Required | |
| Tool/data contract | If tools | Required if tools | Required if tools | |
| Guardrails + failure modes | Recommended | Required | Required | |
| Review scorecard (design score) | Recommended | Required | Required | |
| Eval plan (golden/edge/adversarial) | Recommended | Required | Required | |
| **Executed eval results** | No | Recommended | **Required** | |
| Security review (OWASP-Agentic, trifecta) | If high-risk | Required if med/high | Required | |
| Monitoring **plan** (spec + thresholds) | No | Recommended | Required | |
| Rollback / kill-switch **runbook** (tested) | No | Recommended | Required | |
| Human-approval model | If med/high risk | Required if med/high | Required | |
| Monitoring **live** + canary + rollback exercised | — | — | (the *production* stage, after deploy) | |

> Candidate vs production: a production **candidate** has proven behavior + a monitoring *plan* + a
> *tested* rollback runbook + adapter contracts — it is deploy-ready. The **production** stage additionally
> requires those controls **live** (monitoring active, canary done, rollback exercised in prod). You
> cannot have live production telemetry before you deploy, so it gates the final stage, not the candidate.

## Decision: PASS | CONDITIONAL PASS | FAIL
- **PASS** — no blockers; required evidence complete for the stage; score + eval thresholds met; risks
  acceptable for the stage.
- **CONDITIONAL PASS** — no severe safety blocker; usable under explicit pilot constraints; required
  fixes documented with owner + timeline.
- **FAIL** — any blocker; tool/data safety unclear; weak scope boundaries; eval coverage missing for a
  med/high-risk agent; or the agent may produce unsafe outputs / unsupported decisions / uncontrolled
  data exposure.

**Firewall rule (precise):** a production-candidate/production stage **cannot receive a PASS without
executed eval results** (+ `behavior_evaluated`). A **CONDITIONAL_PASS** at those stages is allowed when
the explicit condition is "run and pass the behavioral eval suite before production release." So missing
executed evals blocks PASS — it does not by itself force FAIL.

## Decision record
```md
Decision: PASS | CONDITIONAL PASS | FAIL
Requested stage:
Reason:
Evidence summary:
Conditions (if CONDITIONAL):
Required fixes: | Priority | Fix | Owner | Due | Required before |
Residual risks:
Next review trigger: major prompt/agent change · new tool/data source · new user group ·
  production incident · eval regression · policy/control change
```
