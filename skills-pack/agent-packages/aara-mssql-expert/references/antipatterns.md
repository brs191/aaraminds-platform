# Common T-SQL / SQL Server antipatterns and their fixes

Scan every draft against this list; state which items were checked.

## Correctness

- **Assuming schema.** Inventing object/column names or index shapes. → Retrieve
  and cite the DDL; ask if missing.
- **`SELECT @v = col FROM ...` over multiple rows.** Silently keeps the last. →
  Deterministic `WHERE` or `TOP 1 ORDER BY`.
- **Nested-transaction misunderstanding.** Inner `COMMIT` doesn't commit; inner
  `ROLLBACK` rolls back everything. → Check `@@TRANCOUNT`/ownership; use `SAVE
  TRAN` for partial rollback.
- **No `SET XACT_ABORT ON`** in a transactional procedure. → Add it so errors
  roll back reliably.
- **`money`/`float` for currency.** → `decimal(19,4)`.
- **`datetime` instead of `datetime2`.** → `datetime2`/`datetimeoffset`.
- **`NOT IN (subquery with NULLs)`** returns empty. → `NOT EXISTS`.
- **`MERGE` under concurrency without `HOLDLOCK`.** Race conditions. → `MERGE ...
  WITH (HOLDLOCK)` or the `UPDLOCK`/`@@ROWCOUNT` upsert pattern.

## Performance

- **`WITH (NOLOCK)` as a tuning fix.** Dirty reads. → Enable RCSI/SNAPSHOT if
  read-blocking is the concern.
- **Scalar UDF in a hot query** (pre-inlining). Per-row, serializing. → Inline
  TVF or inline expression; verify scalar inlining (2019+/compat 150+).
- **Multi-statement TVF on a hot path.** Bad cardinality estimates. → Inline TVF.
- **Non-sargable predicates** (`CONVERT(date, col) = @d`, function-wrapped
  columns, `nvarchar` vs `varchar` mismatch). → Range rewrite; match types;
  expression/computed index.
- **`SELECT *` in production.** Breaks covering indexes, fetches unused columns,
  breaks silently on schema change. → List columns.
- **Table variable `@t` for a large set.** 1-row estimate → bad plan. → `#temp`.
- **Deep pagination with large `OFFSET`/`ROW_NUMBER` filter.** → Keyset/seek
  pagination on the clustered key.
- **Ignoring parameter sniffing.** → `OPTION (RECOMPILE)` / `OPTIMIZE FOR` /
  Query Store forced plan (see the tuning reference).
- **Random GUID (`NEWID()`) clustered key.** Fragmentation. → `NEWSEQUENTIALID()`
  or don't cluster on it.

## Concurrency / operations

- **Assuming locking isolation on Azure SQL DB** (it defaults to RCSI). → State
  the assumed model; verify.
- **Large single `UPDATE`/`DELETE`** → lock escalation + long blocking. → Batch
  (`DELETE TOP (n)` loop with commits).
- **Long transactions across app round-trips.** Locks + log growth. → Keep short.
- **Inconsistent lock order** → deadlocks. → Lock in a fixed order; retry 1205.
- **Manual backup/HA jobs on Azure SQL DB** (platform-managed). → Remove.

## Security

- **String-concatenated dynamic SQL / `EXEC(@sql)`.** → `sp_executesql` with
  params + `QUOTENAME()`.
- **`EXECUTE AS` a high-privilege principal** to paper over a missing grant. →
  Ownership chaining + least privilege.
- **Dynamic Data Masking treated as a security boundary.** It isn't. → Use
  Always Encrypted / RLS for real controls.
- **SQL logins where Entra is available.** → Entra managed identity/groups.
- **Secrets/PII in code, `THROW`/`RAISERROR` text, or comments.** → Never.

## Design

- **JSON in `NVARCHAR(MAX)`** for fields you filter. → Native `JSON` type
  `[SQL2025]`, promote/index hot fields.
- **Triggers for rules a constraint can enforce.** → `CHECK`/`FK`/`UNIQUE`/
  filtered unique index.
- **Heap tables for OLTP.** → Give them a clustered index.

## The agent's habit

Before returning any draft: run this list, state which items were checked, and
surface anything unverifiable (e.g. "index benefit unconfirmed — needs the actual
plan on the target tier"). Honesty about unverified claims is part of the
deliverable.
