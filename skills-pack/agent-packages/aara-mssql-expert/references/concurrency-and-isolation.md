# Concurrency and isolation (RCSI, SNAPSHOT, locks, deadlocks, queues)

The single most important fact for Azure SQL work: **Azure SQL Database enables
READ_COMMITTED_SNAPSHOT (RCSI) by default.** Under RCSI, readers use row
versioning and do not block writers (and are not blocked by them) — the opposite
of a default on-prem SQL Server instance, which uses locking READ COMMITTED.
Always state which model you are assuming; advice inverts between them.

## Isolation levels

- **READ COMMITTED (locking)** — on-prem default. Readers take shared locks;
  readers block writers and vice versa. Prone to blocking chains.
- **READ COMMITTED SNAPSHOT (RCSI)** — Azure SQL DB default. Statement-level row
  versioning; readers see the last committed row without locking. Eliminates
  most reader/writer blocking. Enabled at database level
  (`ALTER DATABASE ... SET READ_COMMITTED_SNAPSHOT ON`).
- **SNAPSHOT** — transaction-level versioning; a transaction sees a consistent
  snapshot as of its start. Update conflicts raise error 3960 → the app must
  retry. Requires `ALLOW_SNAPSHOT_ISOLATION ON`.
- **REPEATABLE READ / SERIALIZABLE** — stronger locking; SERIALIZABLE takes
  range locks (key-range) to prevent phantoms. Use for multi-row invariants that
  a constraint can't enforce; expect more blocking/deadlocks.

Both versioning modes use `tempdb` for the version store — a tempdb sizing
consideration, managed for you on Azure SQL.

## NOLOCK is not a performance fix

`WITH (NOLOCK)` = READ UNCOMMITTED: dirty reads, missing/duplicate rows from page
splits, and reads of uncommitted data that may roll back. It is not a tuning
technique. If reads are blocking under locking READ COMMITTED, the correct answer
is **enable RCSI**, not scatter `NOLOCK`. Flag every `NOLOCK` in review.

## Lock hints that are legitimate

- `WITH (UPDLOCK, HOLDLOCK)` — take update+range locks to make a read-then-write
  atomic (upsert guard).
- `WITH (READPAST)` — skip locked rows; the SQL Server queue pattern (analogous
  to Postgres `SKIP LOCKED`):
```sql
UPDATE TOP (1) dbo.Jobs WITH (READPAST, UPDLOCK)
  SET status = 'running'
  OUTPUT inserted.*
  WHERE status = 'queued';
```
- `sp_getapplock` / `sp_releaseapplock` — application-level named locks (analog
  of Postgres advisory locks) for coordinating outside the row-lock model.

## Lost update

Read-modify-write across two statements loses concurrent updates. Fix atomically:
```sql
UPDATE dbo.Items SET stock = stock - 1 WHERE id = @id AND stock > 0;
```
or lock the row with `WITH (UPDLOCK, HOLDLOCK)` inside a transaction before the
read.

## Deadlocks

Two sessions each hold a lock the other needs; SQL Server picks a victim (error
1205) and rolls it back. Prevention:
- Acquire locks in a **consistent order** across all code paths.
- Keep transactions short; never hold locks across app round-trips or waits.
- Use covering indexes to reduce lookups that widen the lock footprint.
- The app should catch 1205 and retry the transaction.

## Lock escalation

SQL Server escalates many row/page locks on one object to a single table lock
(~5000 locks threshold). A large `UPDATE`/`DELETE` can escalate and block the
table. Batch large DML (`DELETE TOP (n) ... ` in a loop with commits) to avoid
escalation and long blocking.

## Review checklist

1. Which isolation is assumed — RCSI (Azure SQL DB default) or locking? State it.
2. Any `NOLOCK`? Replace with RCSI/SNAPSHOT if the concern is read-blocking.
3. Read-then-write gap that can lose updates? Make atomic or `UPDLOCK, HOLDLOCK`.
4. If SNAPSHOT: is there a 3960 update-conflict retry?
5. Large DML that could escalate locks? Batch it.
6. Queue-style access? Use `READPAST, UPDLOCK` with `OUTPUT`.
