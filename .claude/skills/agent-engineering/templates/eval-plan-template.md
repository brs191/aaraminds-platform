# Template — Agent eval plan + golden dataset

The behavior contract. Fill in; details and theory in `references/evaluation-design.md`. Covers the
three categories, the grader per case, and the CI threshold. **No eval is marked passed unless executed
or backed by supplied evidence.**

## Coverage (the agent must have all three)
- **Functional** — task/goal completion, tool correctness (selection + args + output), outcome/state
  verification (check the environment, not the agent's claim).
- **Behavioral** — step efficiency, plan adherence, role/topic adherence, trajectory match, tone/format.
- **Safety / risk** — policy adherence (right action + right policy), bias/toxicity, **≥1 adversarial /
  prompt-injection case**, data-leakage, over-reach. (Mandatory for any med/high-risk agent.)

## Both-sided rule
Include cases where a behavior **should** fire AND where it **should not** (negative/over-trigger cases).
One-sided evals create one-sided optimization.

## Golden dataset

| ID | Type (happy/edge/adversarial/regression) | Input | Reference outcome (env state / expected) | Grader (code / judge / human) | Should-fire? |
|---|---|---|---|---|---|
| G-001 | happy | | | | yes |
| E-001 | edge | | | | |
| A-001 | adversarial | | | | no/blocked |
| R-001 | regression (a past failure) | | | | |

Start with **20–50 cases** from real/expected failures. Each task has a reference solution (two experts
would agree pass/fail). A 0% pass usually means a broken task/grader, not an incapable agent.

## Graders (cheapest valid one wins)
Code-based (exact/regex, unit tests, tool-call verification) → LLM-as-judge (rubric, pairwise; **one
judge per dimension; give it an "Unknown" out; calibrate to humans; read transcripts**) → human (SME
spot-check). Partial credit for multi-step tasks.

## Scoring & CI gate
- Report **rates**, not single pass/fail: pass@k (≥1 of k) and pass^k (all k — customer-facing).
- **Capability** evals start low (a hill to climb; don't hard-gate). **Regression** evals sit ~100%
  and ARE the CI gate; graduate capability → regression once optimized.
- Thresholds: functional ≥ {{target}} · safety/red-team clean · regression = 100%.
- Baselines tracked: latency, tokens, cost/task, error rate. Online eval on sampled prod traces for drift.

## Go/no-go
Production-ready requires the design band (scorecard) **AND** behavior pass (this plan, executed,
transcripts read). Both, or it is not ready.
