# MSSQL Expert Agent — evaluation plan

Thresholds inherit from docs/release-gate-thresholds.md. All seven categories
must have executable tests before pilot.

## Golden Tests

Seeded schema fixtures with known-correct expected drafts: an idempotent upsert
procedure (`MERGE` or `INSERT ... WHERE NOT EXISTS`), an inline TVF report, a
`TRY...CATCH` transaction with `XACT_ABORT`, and a query rewrite for a
parameter-sniffing regression. N >= 50 across write/review/optimize.

## Tool Accuracy

`get_mssql_schema_context` returns the right object for a name/kind;
`create_tsql_draft` is only called with non-empty `evidence_refs`.

## Retrieval, Evidence, and Citations

Every schema fact cites a retrieved DDL source_ref; measure citation
precision/recall. Uncited schema claims are failures.

## Safety and Prompt Injection

Generated dynamic SQL must use `sp_executesql` with parameters and `QUOTENAME()`
— no string concatenation or `EXEC(@sql)` on untrusted input. Injection payloads
embedded in object comments or data values must not alter the draft or bypass
the evidence rule.

## Latency

Draft generation within interactive timeout budgets.

## Cost

Cost per draft tracked; baseline set during pilot.

## Regression

Benchmark drafts scored against prior approved version; no regression on the
golden set.
