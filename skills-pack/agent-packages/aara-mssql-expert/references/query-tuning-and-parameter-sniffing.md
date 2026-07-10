# Query tuning and parameter sniffing

Tuning is evidence-driven: read the plan, don't guess. Ask for the actual
execution plan (`.sqlplan` XML) and `SET STATISTICS IO, TIME ON` output, plus the
DDL, before reasoning. Parameter sniffing is the signature SQL Server problem and
gets its own section below.

## The measurements

```sql
SET STATISTICS IO, TIME ON;   -- logical reads per table + CPU/elapsed
-- run query; also capture the Actual Execution Plan
```
- **Logical reads** are the truest I/O signal (buffer-pool page touches).
- Compare **estimated vs actual rows** at each plan operator — a large gap is the
  root cause of most bad plans.
- Query Store `[AzureSQL default on]` captures plans and runtime stats over time
  and is the best tool for regressions: find the query, compare plans, and force
  a good one.

## Reading a plan

Read right-to-left, top-to-bottom. Watch for:
- **Scan** where a **Seek** is expected → missing/unusable index or non-sargable
  predicate.
- **Key Lookup** (nonclustered seek + lookup) with high row counts → add
  `INCLUDE` columns to cover.
- **Estimated vs actual mismatch** → stale statistics or parameter sniffing.
- **Implicit conversion** warnings (`CONVERT_IMPLICIT`) → type mismatch defeating
  an index (e.g. `varchar` column vs `nvarchar` parameter).
- **Spills to tempdb** (sort/hash warning) → underestimated memory grant, often
  from a cardinality error.
- **Scan with high "Number of Executions"** inside a nested loop → bad join order
  from a misestimate.

## Sargability

- `WHERE CONVERT(date, CreatedAt) = @d` wraps the column → rewrite as a range
  `WHERE CreatedAt >= @d AND CreatedAt < DATEADD(day, 1, @d)`.
- Leading-wildcard `LIKE '%x'` can't seek. Function-wrapped columns can't seek.
- Type mismatches (`nvarchar` param vs `varchar` column) inject implicit
  conversions — match types.

## Statistics

The optimizer relies on statistics. Fixes for misestimates: `UPDATE STATISTICS`
(auto-update is on by default but can lag on large tables), filtered statistics
for skewed subsets, and ensuring auto-create/auto-update stats are enabled.

## Parameter sniffing — the SQL Server plan-stability problem

On first execution, SQL Server compiles a plan using the **specific parameter
values sniffed at compile time**, then caches and reuses it. If those values are
atypical (e.g. a customer with 2 rows vs one with 2 million), later executions
with different values get a plan optimized for the wrong cardinality — sudden
slowdowns with no code change. Diagnosis: same query, wildly different durations
depending on parameters; plan's compiled-value differs from runtime value.

Fixes, from least to most heavy-handed:
- **`OPTION (RECOMPILE)`** — recompile every execution for the actual values.
  Best when values vary widely and compile cost is acceptable; removes plan reuse.
- **`OPTION (OPTIMIZE FOR (@p = <typical value>))`** — compile for a chosen
  representative value.
- **`OPTION (OPTIMIZE FOR UNKNOWN)`** — use the average density instead of the
  sniffed value; a stable middle-ground plan.
- **Query Store forced plan** `[AzureSQL/2016+]` — pin a known-good plan without
  changing code. Often the cleanest production fix.
- **Split the procedure** — branch to different sub-procedures for skewed vs
  normal parameter ranges so each gets its own plan.
- Avoid the old **local-variable trick** (copying params to locals defeats
  sniffing but forces density estimates); prefer the explicit hints above.

## Azure SQL automatic tuning

`[AzureSQL]` Automatic plan correction can detect a regressed plan (via Query
Store) and auto-force the last good one. Recommend enabling it; still design for
plan stability rather than relying on it.

## Tuning workflow the agent follows

1. Get the actual plan + `STATISTICS IO, TIME` + DDL.
2. Find the operator with the largest estimated-vs-actual row gap.
3. Classify: missing/unusable index, non-sargable predicate, stale stats,
   implicit conversion, or parameter sniffing.
4. Propose the smallest fix and predict the plan change; if sniffing, choose the
   least-invasive option (RECOMPILE / OPTIMIZE FOR / forced plan) with rationale.
5. Never claim a speedup without plan evidence.
