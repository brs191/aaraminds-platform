# Evaluation Plan — aara-mssql-expert

Thresholds inherit from docs/release-gate-thresholds.md. Every category below must have executable tests before pilot (readiness rubric area 6).

## Golden Tests

Approval golden suite, N >= 50 cases across the ten mandated categories; boundary-enforcement tests require 100% pass. [TODO: seed domain-specific golden cases.]

## Tool Accuracy

Per-tool correctness against contract input/output schemas; example invocations validate at load time. [TODO: add task-level accuracy cases per tool.]

## Retrieval, Evidence, and Citations

Every factual claim in output cites a resolvable source; citation precision/recall measured on seeded cases. Memory citation enforcement: 100% cited writes.

## Safety and Prompt Injection

Injected instructions in retrieved content must not alter tools, boundaries, or goals: 100% pass on tool-escalation attempts. [Gate defined in thresholds; harness implementation pending.]

## Latency

Interactive steps within contract timeout_class budgets. Baseline measured during pilot; SLO set after baseline.

## Cost

Cost per run tracked (model, token, tool). Baseline during pilot; budget set after baseline.

## Regression

Benchmark evals: no regression vs prior approved version. Runs recorded per schemas/eval-run.schema.json with evidence refs.
