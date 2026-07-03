# Partitioning — Table-Level

## Purpose

Table-level partitioning splits a single logical table into multiple physical pieces. Done right, it enables fast queries on large tables, painless archival, and bounded maintenance windows. Done wrong, it's complexity for no gain. This reference covers Postgres declarative partitioning, Azure SQL partition functions, MySQL partitioning, the sliding-window pattern, and partition pruning verification. Cosmos / Mongo **partition keys** are a different concept entirely — covered in `patterns/partition-key-design.md`.

## When to partition (and when not)

Real signals to partition:

- **Single table > 100GB** and queries always filter on a known dimension (time, tenant, region)
- **Time-series with retention** — drop old data efficiently by dropping a partition instead of `DELETE FROM ... WHERE created_at < ...`
- **Index size dominates memory budget** — partitioning lets each partition's index fit in shared buffers
- **Maintenance windows constrained by `VACUUM` / index rebuild on huge tables**

Not signals to partition:

- "We might get big someday" — premature partitioning has real operational cost
- The table is "only" 20GB — modern engines with proper indexing handle that fine
- You want faster queries on filters that don't align with the partition key — partitioning won't help

Default: **don't partition until you have a specific reason.** Indexes solve more problems than partitioning does.

## Postgres declarative partitioning

Postgres 10+ supports declarative partitioning. Three strategies:

### RANGE (most common)

Time-series or numeric range:

```sql
CREATE TABLE orders (
  id BIGSERIAL,
  customer_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  total NUMERIC NOT NULL,
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE orders_2026_05 PARTITION OF orders
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE orders_2026_06 PARTITION OF orders
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
```

**The PK must include the partition column.** Postgres can't enforce uniqueness across partitions on a column that isn't part of the partition expression. The PK becomes `(id, created_at)`.

### LIST

Categorical:

```sql
CREATE TABLE orders (...) PARTITION BY LIST (region);
CREATE TABLE orders_us PARTITION OF orders FOR VALUES IN ('US');
CREATE TABLE orders_eu PARTITION OF orders FOR VALUES IN ('EU', 'UK');
```

### HASH

Even distribution:

```sql
CREATE TABLE events (...) PARTITION BY HASH (user_id);
CREATE TABLE events_0 PARTITION OF events FOR VALUES WITH (modulus 4, remainder 0);
CREATE TABLE events_1 PARTITION OF events FOR VALUES WITH (modulus 4, remainder 1);
-- etc.
```

### `pg_partman` for time-series automation

Manually creating monthly partitions is tedious and error-prone. `pg_partman` automates:

```sql
SELECT partman.create_parent(
  p_parent_table => 'public.orders',
  p_control      => 'created_at',
  p_type         => 'range',
  p_interval     => 'monthly',
  p_premake      => 4   -- pre-create 4 future partitions
);
```

Schedule `partman.run_maintenance_proc()` via `pg_cron` to keep it rolling. Add retention to drop old partitions automatically:

```sql
UPDATE partman.part_config
SET retention = '12 months', retention_keep_table = false
WHERE parent_table = 'public.orders';
```

This is the **sliding window** pattern: new partitions auto-created at the leading edge; old partitions auto-dropped at the trailing edge; table size bounded.

### Partition pruning verification (Postgres)

The planner skips partitions that can't match the filter:

```sql
EXPLAIN SELECT * FROM orders
WHERE created_at >= '2026-05-15' AND created_at < '2026-05-20';
-- Plan should show only orders_2026_05 in the scan list
```

If the plan shows all partitions scanned despite a date filter, pruning isn't happening. Common causes: function on the partition column (`WHERE date_trunc('day', created_at) = ...`), parameterized prepared statement that can't prune at plan time (mitigate with `plan_cache_mode = 'force_custom_plan'`), or filter doesn't reference the partition column.

## Azure SQL partitioning

Azure SQL uses **partition functions** + **partition schemes**. Different mental model than Postgres but the operational story is similar.

```sql
-- 1. Partition function defines boundary values
CREATE PARTITION FUNCTION pf_orders_by_month (DATETIME2)
AS RANGE RIGHT FOR VALUES (
  '2026-01-01', '2026-02-01', '2026-03-01',
  '2026-04-01', '2026-05-01', '2026-06-01'
);

-- 2. Partition scheme maps boundaries to filegroups
CREATE PARTITION SCHEME ps_orders_by_month
AS PARTITION pf_orders_by_month
ALL TO ([PRIMARY]);

-- 3. Table uses the partition scheme
CREATE TABLE orders (
  id BIGINT IDENTITY,
  customer_id BIGINT NOT NULL,
  created_at DATETIME2 NOT NULL,
  total DECIMAL(10, 2) NOT NULL,
  CONSTRAINT pk_orders PRIMARY KEY NONCLUSTERED (id, created_at)
) ON ps_orders_by_month(created_at);

-- 4. Clustered index aligned with the partition scheme
CREATE CLUSTERED INDEX cx_orders_created_at ON orders (created_at)
ON ps_orders_by_month(created_at);
```

**`RANGE RIGHT` vs `RANGE LEFT`**: RIGHT means each boundary value belongs to the *right* (higher) partition — `'2026-05-01'` belongs to the May partition, not April. Pick RIGHT for date boundaries — it matches calendar semantics.

### Sliding window on Azure SQL — `SWITCH` is the killer feature

The pattern: SPLIT a new partition at the leading edge, MERGE old partitions at the trailing edge, SWITCH partitions between tables to add or remove data:

```sql
-- Add boundary for next month
ALTER PARTITION FUNCTION pf_orders_by_month()
SPLIT RANGE ('2026-07-01');

-- Switch IN a staging table into a new empty partition (instant)
ALTER TABLE orders_2026_07_staging
SWITCH TO orders PARTITION N;

-- Switch OUT the oldest partition to a scratch table, then truncate
ALTER TABLE orders SWITCH PARTITION 1 TO orders_old;
TRUNCATE TABLE orders_old;

-- Merge the freed-up boundary
ALTER PARTITION FUNCTION pf_orders_by_month()
MERGE RANGE ('2026-01-01');
```

`SWITCH` is a metadata-only operation. Moving a partition between tables takes milliseconds, regardless of row count — that's what makes Azure SQL partitioning powerful for lifecycle management.

### Azure SQL Hyperscale interaction

Hyperscale's page-server architecture provides some of what partitioning offers natively (storage scale, fast backups). Evaluate whether you still need partitioning if you're on Hyperscale; often you don't, unless lifecycle management (drop-by-partition) is the goal.

## MySQL partitioning

InnoDB supports partitioning similar to Postgres / Azure SQL:

```sql
CREATE TABLE orders (
  id BIGINT AUTO_INCREMENT,
  customer_id BIGINT NOT NULL,
  created_at DATETIME NOT NULL,
  total DECIMAL(10, 2),
  PRIMARY KEY (id, created_at)
)
PARTITION BY RANGE (TO_DAYS(created_at)) (
  PARTITION p202605 VALUES LESS THAN (TO_DAYS('2026-06-01')),
  PARTITION p202606 VALUES LESS THAN (TO_DAYS('2026-07-01')),
  PARTITION pmax    VALUES LESS THAN MAXVALUE
);
```

Major caveats vs Postgres / Azure SQL:

- **Foreign keys are not supported on partitioned InnoDB tables.** Large constraint for normalized schemas. If FKs are load-bearing, MySQL partitioning is the wrong tool — consider application-layer sharding instead.
- Every unique index (including PK) must include the partition column.
- `ALTER TABLE ... REORGANIZE PARTITION` for sliding window — heavier operation than Postgres / Azure SQL SWITCH.

Partition pruning check: `EXPLAIN PARTITIONS SELECT ...` — shows which partitions are touched.

## Partition pruning failures and fixes

| Symptom | Cause | Fix |
|---|---|---|
| All partitions scanned | Filter doesn't reference partition column | Add partition column to WHERE |
| All partitions scanned despite filter | Function wraps partition column (`date_trunc`, `EXTRACT`, `CAST`) | Filter on raw column; use a generated column if you need a derived filter |
| Pruning works in SSMS / psql but not from app | Parameter sniffing / generic plan | Force literal, use `OPTION (RECOMPILE)` (Azure SQL), or `plan_cache_mode = 'force_custom_plan'` (Postgres) |
| Pruning works for one query but not the next | Mixed types in comparison | Match types exactly; avoid implicit conversion |

## Worked example — brownfield: adding monthly partitioning to a 5-year-old Postgres orders table

Setup: existing Postgres Flexible Server hosting an `orders` table at 80GB / 200M rows accumulated over 5 years. Reporting queries scan the whole table; `VACUUM` takes hours. Retention policy says "drop orders > 7 years"; right now that requires a multi-hour `DELETE` that bloats the table further.

Decision walk:

1. **Confirm the trigger.** `pg_stat_user_tables` shows the table is the largest in the DB. Reporting queries with `WHERE created_at` filters scan ~50% of pages because old and new data are interleaved on disk. `VACUUM` is a recurring incident. Yes, partition.
2. **Choose strategy.** RANGE on `created_at`, monthly partitions. Aligns with the dominant query filter shape; aligns with retention policy (drop one partition per month after 7 years).
3. **Plan the migration.** Postgres can't convert a non-partitioned table to partitioned in place. Pattern: create new partitioned table, dual-write, backfill, cut over.
4. **Set up partitioning.** Create `orders_partitioned` partitioned by RANGE on `created_at`. Use `pg_partman` to manage monthly partitions, with 60 months historical pre-created and 4 months ahead.
5. **Dual-write.** App version N+1 writes to both `orders` and `orders_partitioned`. Verify counts match for new rows over a day. See `data-migration-patterns.md`.
6. **Backfill historical data.** 200M rows. Batched copy per month: `INSERT INTO orders_partitioned SELECT * FROM orders WHERE created_at >= $start AND created_at < $end`. Run in parallel batches across non-overlapping months. Throttle if replication lag rises.
7. **Validate.** Run a sample of reporting queries against both tables; assert identical results. Run for 1 week with dual-write before cut-over.
8. **Cut over.** Single transaction: `ALTER TABLE orders RENAME TO orders_old; ALTER TABLE orders_partitioned RENAME TO orders;`. Application sees no change in name; queries land on the partitioned version.
9. **Stop dual-write.** App version N+2 writes only to `orders` (now the partitioned table). Drop `orders_old` after a 30-day rollback window.
10. **Enable retention.** `UPDATE partman.part_config SET retention = '7 years', retention_keep_table = false WHERE parent_table = 'public.orders'`.
11. **Schedule maintenance.** `pg_cron` runs `partman.run_maintenance_proc()` nightly. New partitions appear automatically at the leading edge; old ones drop automatically at the trailing edge.
12. **Verify pruning.** `EXPLAIN` on reporting queries shows only the month-relevant partitions touched.

Total elapsed: 4–6 weeks for 200M rows on a reasonably-sized instance. Downtime at cut-over: zero (a rename is metadata-only).

After this, dropping a month of old data is `DROP TABLE orders_2019_05` — instantaneous, no bloat. `VACUUM` per-partition finishes in minutes, not hours. Reporting queries with date filters touch one or two partitions instead of the whole 200M-row table.

## Anti-patterns

- **Partitioning a small table.** Operational overhead with no benefit; queries can get *slower* if pruning doesn't kick in.
- **Wrong partition key.** Filters don't include it → every query becomes a full-table scan across all partitions.
- **Function-wrapped partition column.** `date_trunc`, `EXTRACT`, `CAST`, `LOWER` — all defeat pruning. Use raw column.
- **Too many partitions.** One per day on a 5-year table = 1825 partitions. Postgres planner overhead becomes the bottleneck.
- **Too few partitions.** One per year on a fast-growing table → each partition is 50GB; pruning helps less.
- **Foreign key needed to a partitioned MySQL table.** Not supported. Either don't partition, or accept the FK loss.
- **Partitioning without a retention plan.** Partition count grows forever; planner overhead dominates query time eventually.
- **Forgetting clustered-index alignment (Azure SQL).** Clustered index must use the partition scheme or every partition operation gets expensive.
- **`DELETE FROM orders WHERE created_at < ...` when retention is the goal.** Slow, bloats the table. Drops are faster.
- **Treating Cosmos partition key as table partitioning.** Different concept. Cosmos partitions are physical and managed by the engine; you choose the key, not the boundaries. See `patterns/partition-key-design.md`.

## Verification questions

1. Has the table size and dominant query filter been measured (not assumed) to justify partitioning?
2. Does the partition column appear in the WHERE clause of the dominant queries?
3. Has partition pruning been verified via `EXPLAIN` / execution plan / `EXPLAIN PARTITIONS`?
4. Is there a retention plan (drop old partitions automatically), or will partition count grow without bound?
5. For Postgres: is `pg_partman` configured with pre-create lead time, or are partitions hand-managed?
6. For Azure SQL: is the clustered index aligned with the partition scheme?
7. For MySQL: was FK loss on partitioned tables accepted, or did this rule out partitioning?
8. Has the sliding-window operation (SWITCH on Azure SQL, `pg_partman` maintenance on Postgres) been tested in staging?

## What to read next

- `schema-design.md` — schema decisions that drive partitioning fit
- `postgres-on-azure.md` — `pg_partman`, `pg_cron`, Flexible Server extension list
- `azure-sql-on-azure.md` — Hyperscale interaction, partition scheme + clustered index
- `azure-mysql-on-azure.md` — partitioning caveats in MySQL
- `cosmos-db-design.md` — Cosmos partitions are physical, not table-level (different concept)
- `patterns/partition-key-design.md` — Cosmos partition key design
- `data-migration-patterns.md` — migrating an existing table to a partitioned shape
- `query-execution-and-indexing.md` — partition pruning as a query-plan check
- `ha-dr-data-tier.md` — partitioning interaction with backup / restore size
