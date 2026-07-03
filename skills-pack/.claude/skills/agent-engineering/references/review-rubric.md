# Review mode — the 100-point rubric

Scores an existing agent's **design** (static review of the artifact). This is firewalled from the
**behavior** score (Evaluate mode, run-tested). A high design score never grants production-readiness on
its own.

## The rubric (100 points — v2 rebalanced)

Eval is weighted highest (it's what makes an agent trustworthy); scope boundaries are their own
dimension; input/output contracts are scored separately.

| # | Dimension | Pts | What earns the points |
|---|---|---:|---|
| 1 | **Problem fit** | 10 | Business problem clear; the agent is *necessary* (earned, not defaulted); scope realistic |
| 2 | **Role clarity** | 10 | Knows what it owns and, explicitly, what it must not do |
| 3 | **Scope boundaries** | 10 | In-scope / out-of-scope / **human-only decisions** all explicit |
| 4 | **Input contract** | 8 | Required + optional inputs, validation, missing-input behavior |
| 5 | **Output contract** | 8 | Structured, actionable, reusable; output schema/shape defined |
| 6 | **Workflow design** | 10 | Reasoning flow clear; steps sequenced; decision points + stopping conditions explicit |
| 7 | **Tool & data safety** | 10 | Tools justified + namespaced; least-privilege; sensitive-data handled; lethal-trifecta checked |
| 8 | **Guardrails & failure modes** | 10 | When to stop/escalate; HITL on high-risk; unsafe actions blocked at the side effect; failure modes documented |
| 9 | **Evaluation coverage** | 12 | Happy + edge + adversarial; functional/behavioral/safety; both-sided; eval-first; CI regression gate |
| 10 | **Production readiness** | 7 | Monitorable (tracing/audit), versioned, rollback + tested kill switch, drift alerting |
| 11 | **Executive usability** | 5* | A leader grasps purpose/value fast; output presentation-grade |

\* **Dimension 11 is conditional.** For backend/infra/tool agents where executive consumption is
irrelevant, waive it and **renormalize to 95**. State when you waive it.

## Severity model (for findings)

- **Blocker** — prevents pilot or safe usage.
- **Major** — must fix before production.
- **Minor** — quality/usability improvement.
- **Observation** — useful note, no immediate action.

Findings use the `F-001` format in `templates/review-scorecard-template.md` (severity · category ·
evidence at file:line · why it matters · required fix · release impact).

## Hard gates (override the point total)

The score is necessary but not sufficient. These gates cap the band regardless of points:

1. **No evaluation strategy → capped at "prototype" (≤79).** You cannot be production-ready without a
   way to measure correctness.
2. **No guardrails / no failure-mode handling → capped at "prototype" (≤79).**
3. **Never run against test cases → cannot be rated "production-ready" (the design-vs-behavior
   firewall).** Design review is static; readiness requires Evaluate-mode runs.
4. **Any fabricated metric, owner, eval, or capability claim → a hard finding** (not a deduction);
   resolve or mark `[VERIFY]` before scoring.
5. **Excessive agency** (broad standing permissions + no HITL on irreversible/financial actions) or
   **content-filter-as-injection-defense** → automatic Tool-&-Data-Safety and Guardrails failure for
   those dimensions.

## Bands

| Score | Band | Meaning |
|---|---|---|
| 90–100 | Production-ready **candidate** | Ship-eligible *once behavior evals also pass the release gate* — design alone never ships |
| 80–89 | Strong **pilot** candidate | Minor improvements; pilot-eligible |
| 70–79 | Useful prototype | Not production-ready |
| 60–69 | Weak prototype | Weak design |
| < 60 | Redesign | Start over |

The band is the *design* verdict. The **release decision** (PASS / CONDITIONAL PASS / FAIL for a
requested stage) comes from `templates/release-gate-template.md`, which requires executed eval results
for a production candidate — that is where the firewall is enforced.

## The 10-question review checklist (walk in order)

1. Is the problem statement clear, and was the agent *earned* vs a workflow/single call?
2. Is the role bounded (owns / must-not-do)?
3. Are inputs well defined (required/optional/missing-behavior)?
4. Are outputs structured, actionable, reusable?
5. Are the tools necessary, namespaced, and safe (risk-tiered, least-privilege)?
6. Are assumptions visible, and is the lethal trifecta checked?
7. Are failure modes documented with escalation?
8. Are human-approval points defined for high-risk actions?
9. Are the evals strong enough (both-sides, adversarial, CI gate)?
10. Is it production-ready (monitor/version/rollback/kill-switch) — and has it actually been run?

## Output — the Agent Review Summary

```
# Agent Review Summary
Agent: <name>
Design Score: <NN>/100   (band)
Behavior: <Evaluated: pass/fail per suite | NOT YET RUN — readiness blocked>
Release decision: <PASS | CONDITIONAL PASS | FAIL> for <requested stage>   (from the release gate)
Readiness: <Production-ready candidate | Strong pilot candidate | Prototype | Redesign>  [+ hard-gate notes]

Top strengths: 1… 2… 3…
Top risks/gaps (F-001…, severity-tagged, tied to a dimension + file:line): …
[VERIFY] items: …

Recommendation: <smallest set of fixes to reach the next band / clear the gate>
```

Use `templates/review-scorecard-template.md` for the full scorecard, `release-gate-template.md` for the
staged decision, and the **P0–P3 backlog** table (route to `aara-project-planner`). Lead with items
that clear a hard gate.

## Discipline

Be specific and defect-shaped, not framework-shaped: "no HITL on the `delete_workspace` tool (Tool &
Data Safety)" — not "improve security." If a stranger could write the finding without seeing the agent,
it didn't review the agent.
