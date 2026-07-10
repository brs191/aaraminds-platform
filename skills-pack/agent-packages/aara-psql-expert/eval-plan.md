# PSQL Expert Agent — evaluation plan

Thresholds inherit from docs/release-gate-thresholds.md. All seven categories
must have executable tests before pilot.

## Golden Tests

Seeded schema fixtures with known-correct expected drafts: an idempotent upsert
procedure, a set-returning report function, an audit trigger, and a query
rewrite for a documented slow plan. N >= 50 cases across write/review/optimize.

## Tool Accuracy

`get_schema_context` returns the right object for a given name/kind;
`create_sql_draft` is only called with non-empty `evidence_refs`.

## Retrieval, Evidence, and Citations

Every schema fact in the output cites a retrieved DDL source_ref; measure
citation precision/recall on seeded cases. Uncited schema claims are failures.

## Safety and Prompt Injection

Generated dynamic SQL must parameterize (`%I`/`%L`, `USING`) — no string
concatenation of inputs. Injection payloads embedded in table comments or data
values must not alter the draft or bypass the evidence rule.

## Latency

Draft generation within interactive timeout budgets.

## Cost

Cost per draft tracked; baseline set during pilot.

## Regression

Benchmark drafts scored against prior approved version; no regression on the
golden set.
