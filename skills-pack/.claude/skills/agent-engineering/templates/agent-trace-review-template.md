# Template — agent trace review

Trace-aware (step-level) review of one run. Agent errors compound, so the *visible* failure is usually
downstream of the *actual* one — locate **where** it failed, not just that it failed. Conforms to
`schemas/trace-review.schema.json`. Use alongside Evaluate mode.

## Identity
- Trace ID:   - Agent / version:   - Task:   - Outcome (env state, not the agent's claim):

## Step walk

| # | Type (llm/tool/handoff/guardrail/final) | Tool | Args valid? | Redundant? | In loop? | Guardrail event | Note |
|---|---|---|---|---|---|---|---|
| 1 | | | | | | none | |
| 2 | | | | | | | |

## Trajectory checks
- [ ] **Tool sequence** sound (right tools; right order where order is load-bearing, e.g. policy
  lookup *before* authorization)?
- [ ] **Tool arguments** valid (no guessed/placeholder params)?
- [ ] **No unnecessary calls** (redundant lookups, re-reads)?
- [ ] **No loops** (repeated tool/step cycles)?
- [ ] **No excessive retries**?
- [ ] **No ignored evidence** (did it disregard a tool result it should have used)?
- [ ] **Policy checks** observed where required?
- [ ] **Guardrail triggers** behaved correctly (blocked/flagged/redacted as designed)?
- [ ] **Final answer consistent with the trace** (the agent didn't claim more than the trace established)?

## Findings
- F-…: {{step # · what went wrong · why it matters · fix}}

## Verdict
```text
pass | pass_with_notes | fail
```
Reason:

> Caveat: grade the *product*, not the path, unless the order is genuinely load-bearing (safety/policy).
> Agents find valid alternatives you didn't anticipate; don't fail a sound run for taking a different route.
