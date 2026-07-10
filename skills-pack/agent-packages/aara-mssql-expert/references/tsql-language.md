# T-SQL language reference

T-SQL is SQL Server's procedural dialect. As with any SQL, prefer set-based
statements over row-by-row loops — a single well-written query beats a cursor.
Reach for procedural T-SQL for control flow, error handling, and dynamic SQL.

## Batches and GO

`GO` is a batch separator understood by client tools (SSMS, sqlcmd), **not** a
T-SQL keyword. Some statements (`CREATE PROCEDURE`, `CREATE VIEW`, `CREATE
FUNCTION`) must be the first statement in a batch, so they are separated by `GO`.
`GO 5` runs the batch five times. Never assume `GO` exists inside a stored
procedure body — it doesn't.

## SET options every procedure should declare

```sql
SET NOCOUNT ON;      -- suppress "N rows affected" chatter; reduces network traffic
SET XACT_ABORT ON;   -- on runtime error, abort and roll back the whole transaction
```

`SET XACT_ABORT ON` is important: without it, some errors leave a transaction
open and doomed. Turn it on in any procedure that manages a transaction.

## Variables, table variables, temp tables

```sql
DECLARE @id int = 42, @name nvarchar(100);
SELECT @name = name FROM dbo.Users WHERE id = @id;   -- assignment via SELECT
```

- **Table variable `@t`** — in-memory-ish, no column statistics (the optimizer
  assumes 1 row pre-2019 / limited estimates), no parallelism on insert. Good
  for small, known-tiny sets.
- **Temp table `#t`** — has statistics, indexable, can be large; lives in
  tempdb, visible to the session. Prefer `#t` for anything non-trivial or when
  the optimizer needs real cardinality.
- **Global temp `##t`** — visible to all sessions; rarely the right choice.

Choosing `@t` for a large set is a classic cause of bad plans (1-row estimate →
nested loop over millions). Use `#t` when size is unknown or large.

## Control flow

`IF ... ELSE`, `BEGIN...END`, `WHILE`, `CASE`. There is no `FOR` loop; iterate
with `WHILE` or, better, avoid iteration with a set-based rewrite or a numbers/
tally table.

## Upsert

T-SQL has `MERGE`, but it has well-documented concurrency and bug caveats
(race conditions without `HOLDLOCK`, trigger quirks). For a simple upsert under
concurrency, the safer pattern is often:

```sql
BEGIN TRAN;
UPDATE dbo.Counters WITH (UPDLOCK, SERIALIZABLE) SET n = n + 1 WHERE [key] = @k;
IF @@ROWCOUNT = 0
  INSERT dbo.Counters ([key], n) VALUES (@k, 1);
COMMIT;
```

or `MERGE ... WITH (HOLDLOCK)` if you use MERGE deliberately. Always state the
locking used. See `concurrency-and-isolation.md`.

## Common set-based idioms

- Window functions: `ROW_NUMBER()`, `RANK()`, `SUM() OVER (...)`, `LAG`/`LEAD`.
- `APPLY`: `CROSS APPLY` / `OUTER APPLY` for correlated table expressions and
  calling inline TVFs per row.
- `OUTPUT` clause: capture inserted/updated/deleted rows from a DML statement
  (the T-SQL analog of `RETURNING`).
- Tally/numbers table for gaps-and-islands and set-based generation instead of
  loops.

## SQL Server 2025 language additions

- Native `JSON` type + `JSON_MODIFY`, `JSON_CONTAINS`, `JSON_OBJECT_AGG`,
  `JSON_ARRAY_AGG` `[SQL2025]` — model queried JSON as the native type, not
  `NVARCHAR(MAX)`.
- Regex: `REGEXP_LIKE`, `REGEXP_REPLACE`, `REGEXP_SUBSTR` `[SQL2025]`.
- Native `VECTOR` type and vector search `[SQL2025]` for embedding workloads.

## Correctness traps

- Assigning a variable via `SELECT @v = col FROM ... ` that returns multiple
  rows silently keeps the last — use a deterministic `WHERE`/`TOP 1 ORDER BY`.
- Three-valued logic: `= NULL` never matches; use `IS NULL`. `NOT IN (subquery
  with NULLs)` returns empty — use `NOT EXISTS`.
- `@table` variable with a large set → 1-row estimate → terrible plan; use `#t`.
- Implicit conversion in predicates (`WHERE varchar_col = @nvarchar`) can force a
  scan and defeat an index (and cause `CONVERT_IMPLICIT` in the plan).
