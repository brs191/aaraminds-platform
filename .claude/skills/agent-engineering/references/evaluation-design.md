# Evaluate mode — functional, behavioral, safety

Scores an agent's **behavior** by running it against test cases. This is what clears production-
readiness (Review mode's design score cannot). Routes deep harness mechanics to `ai-evaluation-harness`
and `aara-ai-evaluation-engineer`; this reference is the agent-specific layer.

## The shift: evaluate the outcome, the path, and the components

Agent errors compound — a bad early decision cascades, so the *visible* failure is downstream of the
*actual* one. Evaluate at three depths and tie every score to a trace span:

- **End-to-end / outcome** — did the task succeed? Verify the **environment state**, not the agent's
  claim ("the reservation row exists in the DB," not "the agent said booked").
- **Trajectory** — was the path sound and efficient (right tools, right order where it matters, step
  count, tokens, no loops)?
- **Component** — which retriever, tool, or sub-step broke?

## Three evaluation categories (the Evaluate-mode contract)

| Category | Question | Concrete metrics |
|---|---|---|
| **Functional** | Did it do the job correctly? | Task/goal completion; tool correctness (selection + args + output); fail-to-pass & pass-to-pass tests; outcome/state verification; factual correctness |
| **Behavioral** | Did it behave well — in-scope, efficient, on-character? | Step efficiency (redundant calls, loops); plan quality/adherence; reasoning relevancy; role adherence; topic adherence; trajectory match; tone/format rubric; turn/token caps |
| **Safety / risk** | Could it cause harm or violate policy? | Policy adherence (right action, wrong policy = fail); bias/toxicity/harmful content; prompt-injection red-team; data-leakage; over-reach; adversarial stress |

A single task is often multidimensional: resolved (state check) AND < N turns (transcript) AND tone OK
(rubric) — grade by combining all three.

## Grader families (cheapest valid one wins)

1. **Code-based** — exact/regex/fuzzy match, unit tests, static analysis, **tool-call verification**,
   transcript metrics. Fast, objective, reproducible. Use wherever the check is exact.
2. **Model-based (LLM-as-judge)** — rubric scoring, NL assertions, **pairwise comparison**, reference-
   based. Flexible for open-ended output. **Pitfalls + required mitigations:** calibrate against human
   experts first; **one judge per dimension** (don't score everything at once); **give the judge an
   "Unknown" out** to prevent hallucinated grades; force empirical output (correct/incorrect or 1–5);
   periodic human spot-checks (models drift).
3. **Human** — SME review, spot-check sampling, A/B. Gold standard; calibrates the model graders.

**Never trust a score until you read transcripts.** (A 42% benchmark score was a grader bug — rigid
matching rejected a correct answer; fixed, it was 95%.)

## Trajectory / tool-call scoring (agent-specific)

- **Tool-call correctness** (deterministic): selection + input parameters + output accuracy; supports
  order-independence, frequency flexibility, partial credit.
- **Trajectory match** modes: `strict` (exact sequence — only when order is load-bearing, e.g. policy
  lookup before authorization), `unordered` (right set, any order), `subset` (no scope creep / only
  reference tools), `superset` (at least the required tools). Caveat: **grade the product, not the
  path** unless order genuinely matters — hard-coding a tool sequence is brittle; agents find valid
  alternatives.

## Building test cases

- **Start small: 20–50 cases from real or expected failures** (bug tracker, support queue, the manual
  tests you already run). Evals get harder to build the longer you wait.
- **Write unambiguous tasks with a reference solution** — gold standard: two experts independently
  reach the same verdict. A 0% pass usually means a broken task or grader, not an incapable agent.
- **Balance both sides** — test where a behavior *should* fire AND where it *shouldn't* (the
  over-triggering / negative cases). One-sided evals create one-sided optimization. This is where
  edge + adversarial cases live.
- **Isolated harness** — each trial starts clean (no leftover state) or failures correlate and inflate.
- **Partial credit** for multi-step tasks; graders resistant to hacks.
- **Report rates, not single pass/fail:** run multiple trials; report **pass@k** (≥1 of k — for
  first-try-matters tasks) and **pass^k** (all k — for customer-facing reliability).

## CI gates: capability vs regression

- **Capability evals** ("what can it do well?") start at a low pass rate — a hill to climb; don't gate
  hard.
- **Regression evals** ("does it still do what it used to?") sit at **~100%** — these are the **CI
  gates**; any drop = a break.
- **Graduation:** once a capability eval is optimized high, it graduates into the regression suite.
- Every production failure becomes a new regression case — "your dataset grows every time the agent
  embarrasses you." Baselines come free: track latency, tokens, cost/task, error rates.
- Wire into the pack's existing eval infra: emit triggering-eval JSON compatible with
  `ai-evaluation-harness` and the pack's `skill_audit`/eval runners; add online eval on sampled
  production traces for drift.

## Delegation, templates, and schemas

Evaluate mode **delegates to the existing evaluator specialist, `aara-ai-evaluation-engineer`** (+ the
`ai-evaluation-harness` skill) — this skill does not ship a duplicate evaluator agent. It supplies the
agent-specific layer and these artifacts:

- `templates/eval-plan-template.md` + golden dataset · `templates/agent-trace-review-template.md`
  (step-level, "where did it fail") · `templates/agent-efficiency-scorecard.md` (latency/tokens/tool-
  calls/loops/cost-per-success) · `templates/tool-risk-register.md` (tool-misuse / excessive-agency).
- `schemas/eval-case.schema.json`, `eval-result.schema.json`, `trace-review.schema.json`,
  `release-gate.schema.json` — machine-readable contracts so results are automation/CI-ready.

## The go/no-go

Production-ready requires: functional pass rate at target, behavioral within bounds, safety/red-team
clean, efficiency within budget, **and** the design review (rubric) at the required band with no hard
gate tripped. Design and behavior must *both* pass. Emit the result per `eval-result.schema.json` with
evidence (scores + read transcripts), not an assertion.
