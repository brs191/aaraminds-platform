# Transactions and Isolation — Cross-Engine

## Purpose

Isolation level choice silently determines correctness in multi-writer systems. Most data-tier bugs that look like "lost data" or "the count is off by 1" are isolation-level bugs in disguise — write skew, lost update, phantom reads. This reference covers the isolation surface across Postgres, Azure SQL, MySQL, Cosmos DB, and MongoDB, and the rules for picking the right level per operation.

## The four classical anomalies (recap, briefly)

1. **Dirty read** — read uncommitted data; reader sees a change that may be rolled back.
2. **Non-repeatable read** — read a row twice in one transaction, get different values (someone updated it in between).
3. **Phantom read** — read a range twice in one transaction, get different sets of rows (someone inserted into the range).
4. **Lost update** — two transactions read the same row, both compute a new value, both write — one update is lost.

A fifth, often overlooked:

5. **Write skew** — two transactions read overlapping data, each commits a write that's individually valid but jointly violates an invariant (e.g., the famous "two doctors on call" problem).

## Postgres — MVCC with three isolation levels

| Level | Anomalies prevented | Cost |
|---|---|---|
| `READ COMMITTED` (default) | Dirty read | None |
| `REPEATABLE READ` (snapshot) | + Non-repeatable read, phantom read | Snapshot kept; serialization failures possible |
| `SERIALIZABLE` | + write skew, lost update | Higher; serialization failures more common |

Postgres uses **MVCC**: readers never block writers; writers never block readers (for the same row at different snapshots). This is the key Postgres property — read transactions never need to wait, regardless of isolation.

**Default for OLTP**: `READ COMMITTED`. Correct for 95% of services. The application handles lost-update via:

- **Optimistic locking**: add a `version` column; UPDATE with `WHERE version = ?`; bump version on success; retry on zero-row-affected.
- **`SELECT ... FOR UPDATE`**: lock the row for the rest of the transaction; readers using SELECT (without FOR UPDATE) still don't block.

For "give me a consistent view across multiple queries": `REPEATABLE READ`. Postgres' RR is snapshot isolation — read the database as of transaction start; commit only succeeds if no conflict.

For "I need write-skew prevention" (rare): `SERIALIZABLE`. Postgres detects serialization conflicts and aborts; application must retry. Use only when an invariant absolutely requires it.

Set per transaction:

```sql
BEGIN ISOLATION LEVEL REPEATABLE READ;
-- ...
COMMIT;
```

Or globally in `postgresql.conf` via Flexible Server configuration. **Don't change the default globally** unless you know all applications will tolerate it.

## Azure SQL — six isolation levels, RCSI is the OLTP default

Azure SQL inherits SQL Server's six isolation levels:

| Level | Prevents | Mechanism |
|---|---|---|
| `READ UNCOMMITTED` | Nothing | No shared locks |
| `READ COMMITTED` (default, with locking) | Dirty read | Shared locks released after read |
| `READ COMMITTED SNAPSHOT` (RCSI) | Dirty read | Row versions; no shared locks (recommended OLTP default) |
| `REPEATABLE READ` | + Non-repeatable read | Shared locks held until commit |
| `SNAPSHOT` | + Phantom read | Row version; transaction-level consistency; **write skew possible** |
| `SERIALIZABLE` | + Write skew | Range locks |

**Critical setting**: enable RCSI on every Azure SQL OLTP database:

```sql
ALTER DATABASE [orders] SET READ_COMMITTED_SNAPSHOT ON;
ALTER DATABASE [orders] SET ALLOW_SNAPSHOT_ISOLATION ON;
```

Effect: readers use row versions, no shared locks. Most read-write deadlocks vanish. Writers still lock for update.

Set per transaction:

```sql
SET TRANSACTION ISOLATION LEVEL SNAPSHOT;
BEGIN TRANSACTION;
-- ...
COMMIT;
```

**Snapshot isolation is not serializable** — write skew is still possible. If you need full serializability, use `SERIALIZABLE`; if you can manage write skew via application-level invariants or unique constraints, `SNAPSHOT` is cheaper.

## MySQL — InnoDB, default REPEATABLE READ

MySQL InnoDB's default is `REPEATABLE READ`. Different from Postgres / Azure SQL.

| Level | Behavior |
|---|---|
| `READ UNCOMMITTED` | Dirty reads |
| `READ COMMITTED` | Standard read committed |
| `REPEATABLE READ` (default) | Snapshot isolation with next-key locking |
| `SERIALIZABLE` | Range locks held |

InnoDB's REPEATABLE READ uses **next-key locks** — gap locks plus row locks, which prevents phantom reads (different from standard SQL REPEATABLE READ that allows phantoms).

**OLTP recommendation**: switch to `READ COMMITTED`:

```sql
SET GLOBAL transaction_isolation = 'READ-COMMITTED';
```

Reasons:
- Fewer lock waits (no gap locks)
- Aligns with what most application code expects (most ORMs assume read-committed-ish semantics)
- REPEATABLE READ's gap locks cause surprise locking issues

Verify with `SELECT @@transaction_isolation`. Apply at server level on Flexible Server via configuration.

## Cosmos DB — single-partition transactions only

Cosmos DB's transaction model is fundamentally different:

- **Single-partition transactions** via `TransactionalBatch` (NoSQL API) — operate on multiple items within one partition key, atomically
- **Cross-partition transactions**: **not supported**. By design.
- Consistency level (Strong, Bounded Staleness, Session, etc.) addresses read behavior, not transaction scope

Implication: design the partition key so items that need atomic update share a partition (`patterns/partition-key-design.md`). For order with line items, partition by `orderId` and put both order and lines in the same container — `TransactionalBatch` updates both atomically.

If you need atomic updates across partitions, redesign: either change partition key, or use saga (`microservices-data-architecture`) for eventually-consistent multi-step transactions with compensation.

## MongoDB — multi-document transactions

MongoDB supports multi-document transactions across collections (within a replica set or sharded cluster):

```javascript
const session = client.startSession();
session.startTransaction({
  readConcern: { level: 'snapshot' },
  writeConcern: { w: 'majority' }
});
try {
  await db.orders.insertOne({ ... }, { session });
  await db.inventory.updateOne({ ... }, { ... }, { session });
  await session.commitTransaction();
} catch (err) {
  await session.abortTransaction();
  throw err;
} finally {
  await session.endSession();
}
```

Isolation: snapshot. Write concern: usually `{w: 'majority'}` for durability.

Multi-document transactions have overhead — slower than single-document atomic operations. Use sparingly. Most Mongo applications shouldn't need them; design data to keep atomic updates within one document where possible (embedded arrays / subdocuments).

## Optimistic vs pessimistic locking

| Pattern | When | Cost |
|---|---|---|
| **Pessimistic** (`SELECT FOR UPDATE`) | High contention; need to hold across multiple statements | Blocks other writers; deadlock risk |
| **Optimistic** (version column) | Low-to-medium contention; retry is cheap | Retries on conflict; no blocking |

Optimistic locking pattern (engine-agnostic):

```sql
-- Read
SELECT id, total, version FROM orders WHERE id = ?;

-- Compute new state
new_total := total + price;
new_version := version + 1;

-- Conditional update
UPDATE orders
SET total = ?, version = ?
WHERE id = ? AND version = ?;

-- If 0 rows affected: someone else updated; reread and retry
```

Default to optimistic. Use pessimistic only when retries are infeasible or contention is very high.

## Common bugs and how isolation maps to fixes

### Lost update — "the count is wrong by N"

```
T1: read counter (10) → compute 11 → write 11
T2: read counter (10) → compute 11 → write 11
Final: 11, lost one increment
```

**Fixes**:

1. Optimistic locking with version column.
2. Atomic increment: `UPDATE counters SET n = n + 1 WHERE id = ?` (engine handles atomicity).
3. Move counter to Redis (`INCR` is atomic).

### Write skew — "two doctors went off call"

```
T1: SELECT count(*) FROM oncall (= 2). count > 1, so I can go off. UPDATE me = off.
T2: SELECT count(*) FROM oncall (= 2). count > 1, so I can go off. UPDATE me = off.
After both commit: count = 0. Invariant violated.
```

**Fixes**:

1. Serializable isolation (Postgres, Azure SQL serializable).
2. Add a constraint that enforces the invariant directly (e.g., a database constraint or a separate "on-call slot" table with limited rows).
3. Pessimistic lock on a sentinel row that all "go off call" transactions must acquire.

### Phantom read — "the count changed mid-transaction"

```
T1 BEGIN
T1: SELECT count(*) FROM orders WHERE status='pending' → 50
T2 BEGIN; INSERT INTO orders ... pending; COMMIT;
T1: SELECT count(*) FROM orders WHERE status='pending' → 51
```

**Fixes**:

1. Snapshot isolation (Azure SQL SNAPSHOT, Postgres REPEATABLE READ) — both reads see 50.
2. Range locks (SERIALIZABLE) — blocks the insert in T2 until T1 commits.

## Anti-patterns

- **Hot read counter in OLTP**. Lock contention on every increment. Move to Redis INCR.
- **`SELECT FOR UPDATE` held across HTTP calls.** Long-held locks; blocks every other writer. Don't hold transactions across external service calls.
- **Mixing isolation levels per-query without justification.** Inconsistency of behavior; harder to reason about.
- **Default REPEATABLE READ in MySQL with app code assuming READ COMMITTED.** Surprising lock behavior.
- **Not enabling RCSI in Azure SQL.** Avoidable read-write deadlocks on every busy table.
- **Using SERIALIZABLE everywhere "just to be safe."** Severe lock contention; hidden in non-prod loads, painful in prod.
- **Cross-partition Cosmos transactions assumed to work.** They don't. Saga or partition-key redesign required.

## Verification questions

1. For Azure SQL: is RCSI enabled (`SELECT name, is_read_committed_snapshot_on FROM sys.databases`)?
2. For MySQL: is isolation level set to READ COMMITTED at the server level?
3. For Postgres: is the default READ COMMITTED, with optimistic locking patterns documented for contention cases?
4. For Cosmos: do cross-partition atomic-update needs map to a saga or a partition-key redesign, not "we'll add a transaction"?
5. For all engines: is "lost update" handled (optimistic locking, atomic primitives, or Redis)?
6. Has the team explicitly considered write-skew scenarios for invariants that span rows?

## What to read next

- `wait-stats-and-blocking.md` — isolation level affects which waits dominate
- `postgres-on-azure.md` — Postgres MVCC and `SELECT FOR UPDATE` patterns
- `azure-sql-on-azure.md` — RCSI enablement, snapshot vs serializable
- `cosmos-db-design.md` — TransactionalBatch and partition-aligned design
- `mongodb-on-azure.md` — multi-document transaction patterns
- `microservices-data-architecture` skill — saga for cross-partition / cross-service transactions
