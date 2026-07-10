# Procedures and functions (scalar vs inline TVF, EXECUTE AS)

## Function types — the choice that dominates performance

- **Inline table-valued function (iTVF)** — a single `RETURN (SELECT ...)`. The
  optimizer **inlines** it into the calling query, so it's essentially a
  parameterized view. Almost always the right choice. Prefer iTVFs for reusable
  query logic.

```sql
CREATE OR ALTER FUNCTION dbo.RecentOrders(@customerId int)
RETURNS TABLE AS
RETURN (SELECT o.Id, o.Total FROM dbo.Orders o WHERE o.CustomerId = @customerId);
-- used with CROSS APPLY or FROM dbo.RecentOrders(@id)
```

- **Multi-statement TVF (mTVF)** — `RETURNS @t TABLE (...) ... INSERT @t ...`.
  Historically a performance trap: the optimizer estimated a fixed low row count
  (1, then 100), producing bad plans. Interleaved execution (2017+/compat 140+)
  improved estimates, but iTVFs are still preferred. Avoid mTVFs for hot paths.

- **Scalar function** — a per-row function call. Classically a serial,
  per-row-invoked performance killer that also blocked parallelism. **Scalar UDF
  inlining** (2019+/compat 150+) can inline eligible scalar UDFs into the query,
  removing much of the penalty — but not all scalar UDFs qualify. Prefer
  expressing the logic inline or as an iTVF; if you must use a scalar UDF, verify
  it inlines (check the plan) and keep it deterministic.

## Procedures

```sql
CREATE OR ALTER PROCEDURE dbo.GetOrders @customerId int
AS
BEGIN
  SET NOCOUNT ON;
  SELECT Id, Total, Status FROM dbo.Orders WHERE CustomerId = @customerId;
END;
```
- Use `CREATE OR ALTER` for idempotent deploys.
- `OUTPUT` parameters and result sets both work; document the contract.
- `WITH RECOMPILE` on the procedure recompiles every call (heavy — prefer
  statement-level `OPTION (RECOMPILE)` on the specific sniff-sensitive statement).
- `SET NOCOUNT ON` first — avoids extra `DONE_IN_PROC` messages.

## EXECUTE AS and security context

`WITH EXECUTE AS OWNER|SELF|'user'` runs the module under another principal — see
`dynamic-sql-and-security.md`. Combined with ownership chaining, this is how you
grant `EXECUTE` on a procedure without granting table rights. Keep it deliberate
and least-privileged.

## Determinism and schemabinding

`WITH SCHEMABINDING` on a function makes it deterministic-eligible, prevents
underlying schema changes that would break it, and is required for indexed views
and some RLS predicates. Add it to functions used in constraints/indexes.

## Review checklist

1. Is this logic better as an inline TVF than a scalar/multi-statement function?
2. If a scalar UDF: does it inline (2019+/compat 150+)? Is it deterministic?
3. `SET NOCOUNT ON` present; `CREATE OR ALTER` for deploys?
4. `EXECUTE AS` least-privileged and intentional?
5. Should it be `WITH SCHEMABINDING` (used in an index/constraint/RLS)?
