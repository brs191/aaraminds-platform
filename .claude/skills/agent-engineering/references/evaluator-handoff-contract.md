# Evaluator handoff contract

Exactly what `aara-agent-engineer` (Evaluate mode) sends to `aara-ai-evaluation-engineer`, and what it
expects back. This makes the delegation concrete and testable — and keeps the firewall intact: the
engineer cannot mark an agent run-tested without this round-trip.

## Role

**"Agent Evaluation & Efficiency Engineer" is a role, not a separate agent.** For the agent lifecycle,
`aara-ai-evaluation-engineer` *acts as* that role — it owns behavior, safety, trace, efficiency, and
regression evaluation for the agent under test. `aara-agent-engineer` creates and gates; the AI
Evaluation Engineer tests and measures; `ai-evaluation-harness` provides the methodology and execution.
No `aara-agent-evaluator` agent is created — that would duplicate this specialist and cause sprawl.

## What the orchestrator sends (the request)

```md
- Agent under test: <name> · <version> · runtime target
- Runnable artifact: <path to the agent file>
- AGENT_SPEC.md: <path>            # scope, tools, guardrails, success criteria
- Tool-risk register: <path>       # which tools, risk tiers, HITL points
- Golden set: <path/JSON>          # cases conforming to schemas/eval-case.schema.json
- Categories to run: functional, behavioral, safety  (+ efficiency)
- Trajectory expectations: <trajectory_match mode per case, if any>
- Test environment: <how to reach a clean, isolated env; how to verify end state>
- Pass thresholds: functional ≥ <t> · safety/red-team clean · regression = 100% · efficiency budget
- Trials per case: <k>   (report pass@k and pass^k)
```

## What the engineer returns (the response)

```md
- Per-case results conforming to schemas/eval-result.schema.json
  (executed=true/false, passed, score, grader, efficiency block, evidence, transcript_ref)
- Trace reviews conforming to schemas/trace-review.schema.json for failed/notable runs
- Aggregate: functional pass-rate, behavioral summary, safety verdict, efficiency scorecard
- Read-transcript confirmation (a human/judge actually inspected a sample)
- A recommended release-gate evidence block (executed_eval_results, etc.)
```

## Firewall obligations on both sides

- The orchestrator **must not** populate `release-gate.evidence.executed_eval_results=true` from its own
  inference — only from a returned result set with `executed=true`.
- The engineer **must not** return `passed=true` for an `executed=false` case unless an explicit
  `evidence_override` (with human evidence) is set — enforced by `eval-result.schema.json`.
- If the engineer is unavailable, Evaluate mode produces the eval *plan* only and the gate cannot exceed
  CONDITIONAL_PASS at pilot / FAIL at production candidate.

## Handoff sequence

1. Orchestrator emits the request block + golden set (validate with `scripts/run-evals.py --validate`).
2. Engineer runs the suite (its harness implements the `Executor` interface in `run-evals.py`).
3. Engineer returns results + trace reviews + aggregate.
4. Orchestrator validates results against the schema, fills the release-gate evidence, and runs
   `scripts/check-release-gate.py`.
5. Decision recorded; transcripts retained.
