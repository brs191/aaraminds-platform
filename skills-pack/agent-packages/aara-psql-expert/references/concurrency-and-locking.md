# Concurrency, locking, and correctness under contention

PostgreSQL is MVCC: readers never block writers and writers never block readers.
Contention comes from **writers touching the same rows** and from **DDL taking
strong locks**. Most concurrency bugs are lost updates, deadlocks, or a
migration that took an `ACCESS EXCLUSIVE` lock on a hot table.

## Isolation levels

- **Read Committed** (default) — each statement sees a snapshot taken at
  statement start. Two concurrent updates to the same row serialize; the second
  re-reads the latest committed row. Prone to non-repeatable reads and lost
  updates across a read-then-write gap in application code.
- **Repeatable Read** — snapshot fixed at transaction start; prevents
  non-repeatable and phantom reads. A write conflict raises
  `could not serialize access` (SQLSTATE 40001) — the app must retry.
- **Serializable** — full serializability via predicate locks (SSI). Strongest
  correctness; also throws 40001 serialization failures that must be retried.
  Use for invariants that span multiple rows/tables and can't be enforced by a
  constraint.

Design rule: if you use Repeatable Read or Serializable, the application **must**
have a retry loop on 40001. State that explicitly in any draft that relies on
them.

## Row-level locks

- `SELECT ... FOR UPDATE` — locks matched rows against concurrent update/delete.
- `FOR NO KEY UPDATE` — weaker; allows concurrent key-preserving updates.
- `FOR SHARE` / `FOR KEY SHARE` — shared locks for read-then-write patterns and
  FK checks.
- `SKIP LOCKED` — skip rows already locked (job-queue pattern).
- `NOWAIT` — fail immediately instead of waiting.

## The lost-update trap

Read-modify-write in application code across two statements loses concurrent
updates:

```sql
-- BAD: two clients read 10, both write 9; one decrement is lost
SELECT stock FROM items WHERE id = 1;         -- app computes stock - 1
UPDATE items SET stock = 9 WHERE id = 1;
```

Fix by doing it atomically in SQL, or by locking:

```sql
-- GOOD: atomic
UPDATE items SET stock = stock - 1 WHERE id = 1 AND stock > 0;
-- or lock the row first
SELECT stock FROM items WHERE id = 1 FOR UPDATE;
```

## Idempotent upserts

`INSERT ... ON CONFLICT` is race-free and cheaper than catching
`unique_violation` in a loop:

```sql
INSERT INTO counters(key, n) VALUES ($1, 1)
ON CONFLICT (key) DO UPDATE SET n = counters.n + 1
RETURNING n;
```

Requires a unique or exclusion constraint on the conflict target. Use
`DO NOTHING` for insert-if-absent. `[PG18]` `RETURNING` can expose both `OLD` and
`NEW` for the affected row.

## Job queue pattern (`SKIP LOCKED`)

```sql
WITH job AS (
  SELECT id FROM jobs WHERE status = 'queued'
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE jobs SET status = 'running' FROM job WHERE jobs.id = job.id
RETURNING jobs.*;
```

Each worker grabs a different row without blocking; this is the canonical
Postgres-as-queue pattern.

## Deadlocks

A deadlock is two transactions each holding a lock the other needs. Postgres
detects it and kills one (SQLSTATE 40P01). Prevention:

- Acquire locks in a **consistent order** across all code paths (e.g. always
  lock rows by ascending id).
- Keep transactions short; don't hold locks across external calls or user think
  time.
- Prefer single atomic statements over multi-statement lock sequences.

## DDL locks — the operational footgun

Many `ALTER TABLE` forms take `ACCESS EXCLUSIVE`, blocking **all** reads and
writes for the duration — catastrophic on a hot table if the operation rewrites
it. See `migrations-and-ddl-safety.md` for which operations are safe, which
rewrite, and how to avoid long locks (`NOT VALID` + `VALIDATE`,
`CREATE INDEX CONCURRENTLY`, `SET lock_timeout`).

## Review checklist

1. Is there a read-then-write gap that can lose updates? Make it atomic or lock.
2. If using Repeatable Read/Serializable, is there a 40001 retry loop?
3. Could an upsert (`ON CONFLICT`) replace a catch-`unique_violation` loop?
4. Are locks acquired in a consistent order to avoid deadlocks?
5. For queue-like access, is `FOR UPDATE SKIP LOCKED` used?
