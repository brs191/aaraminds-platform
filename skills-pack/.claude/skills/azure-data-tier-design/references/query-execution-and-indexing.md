# Query Execution and Indexing — Cross-Engine

## Purpose

The single most useful skill for operating data-tier workloads is reading a query plan and choosing the right index. This reference gives the cross-engine framework — Postgres `EXPLAIN`, Azure SQL Query Store, Cosmos query metrics, MySQL `EXPLAIN`, Mongo `explain()` — and the indexing fundamentals that transfer across them.

## The universal sequence — "this query is slow, what now?"

1. **Capture the plan**. Don't tune from intuition. Get the actual plan for the actual parameters.
2. **Read the plan top-down**. Identify the most expensive operator. Usually it's a scan that should be a seek, or a join that should have been narrower.
3. **Check the predicate**. Is the filter using the indexed column directly, or is it wrapped in a function / implicit cast?
4. **Check the index**. Does an appropriate index exist? Is the planner using it?
5. **Check the cardinality estimates**. Is the planner's row estimate close to actual? Off by 100× means stats are stale or correlated.
6. **Decide the fix**: rewrite query, add/modify index, update stats, partition, or accept and design around it.

## Engine-specific plan capture

### Postgres

```sql
EXPLAIN (ANALYZE, BUFFERS, VERBOSE, FORMAT TEXT)
SELECT * FROM orders WHERE customer_id = 1234 AND status = 'pending';
```

- `ANALYZE` runs the query (don't use in prod on huge writes)
- `BUFFERS` shows shared/local buffer hits — pages read vs cached
- Plan shows estimated vs actual rows; ratio > 10× = stats problem
- `auto_explain` extension captures plans for slow queries automatically:

```sql
LOAD 'auto_explain';
SET auto_explain.log_min_duration = '500ms';
SET auto_explain.log_analyze = true;
SET auto_explain.log_buffers = true;
```

Enable in `postgresql.conf` (via Azure Flexible Server configuration) for permanent capture.

### Azure SQL — Query Store does it for you

Query Store captures the plan for every executed query with its actual runtime stats. To view a specific query plan:

```sql
SELECT qsq.query_id, qsqt.query_sql_text, qsp.query_plan
FROM sys.query_store_query qsq
JOIN sys.query_store_query_text qsqt ON qsq.query_text_id = qsqt.query_text_id
JOIN sys.query_store_plan qsp ON qsq.query_id = qsp.query_id
WHERE qsqt.query_sql_text LIKE '%orders%';
```

In SSMS: right-click query → "Show Estimated Execution Plan" or "Include Actual Execution Plan." Key things to look for in the graphical plan:

- Thick arrows = many rows; thin arrows = few. Mismatch between expected and actual arrow widths = bad estimate.
- Index Seek (good) vs Index Scan vs Table Scan (worse)
- Key Lookup (RID Lookup) = nonclustered index used but additional columns fetched from base table → covering index opportunity
- Hash Match vs Nested Loops vs Merge Join — planner chose based on cardinality estimates; if wrong, often a stats issue

### MySQL

```sql
EXPLAIN SELECT * FROM orders WHERE customer_id = 1234 AND status = 'pending';
EXPLAIN ANALYZE SELECT ...;  -- MySQL 8.0+
EXPLAIN FORMAT=TREE SELECT ...;
```

Performance Schema captures slow queries:

```sql
SELECT * FROM performance_schema.events_statements_summary_by_digest
ORDER BY sum_timer_wait DESC LIMIT 20;
```

### Cosmos DB

Every Cosmos response includes `x-ms-request-charge` (RU cost) and `x-ms-query-metrics` (detailed breakdown):

```
Retrieved Document Count : 250
Retrieved Document Size  : 41250
Output Document Count    : 25
Index Hit Document Count : 25
Query Engine Times       : 12ms
Index Lookup Time        : 1ms
Document Load Time       : 8ms
Runtime Execution Time   : 3ms
```

- **Index Hit Count / Retrieved Document Count** ratio close to 1 = index used well
- **Retrieved Document Count** >> **Output Document Count** = scan; index missing for filter
- **Cross-partition** flag — true means fan-out; partition key not in filter

Use SDK to log per-query: `response.RequestCharge`, `response.Diagnostics.ToString()`.

### MongoDB

```javascript
db.orders.find({customer_id: 1234, status: 'pending'}).explain('executionStats');
```

- `winningPlan.stage` — `IXSCAN` (good) vs `COLLSCAN` (full collection scan, bad)
- `executionStats.totalDocsExamined` vs `executionStats.nReturned` — ratio close to 1 = good
- `executionTimeMillis` — actual time

For aggregation pipelines: `db.collection.aggregate([...]).explain('executionStats')`. Look at each stage's `nReturned` and `executionTimeMillisEstimate`.

## Indexing fundamentals — common across engines

### Index types

| Type | Engine | Use |
|---|---|---|
| **B-tree** | Postgres, MySQL, Azure SQL (rowstore), Cosmos secondary | Default. Range scans, equality, ORDER BY |
| **Hash** | Postgres, MySQL (Memory tables) | Equality only; rarely better than B-tree |
| **GIN** | Postgres | JSONB, full-text, arrays |
| **GiST / BRIN** | Postgres | Geometric, time-series with sequential inserts |
| **Columnstore** | Azure SQL, Synapse | Analytical aggregation over wide rows |
| **Filtered / partial** | Azure SQL, Postgres | Sparse predicate optimization |
| **Composite (multi-column)** | All | Multi-column filters and ORDER BY |
| **Inverted / RUM** | Postgres (with extensions), Mongo | Text search, full document indexing |

### Composite index ordering — the rule

For a query `WHERE a = ? AND b = ? ORDER BY c`:

- Index `(a, b, c)` — best; supports the full operation
- Index `(b, a, c)` — also good for equality (order of equality columns doesn't matter as much in B-tree)
- Index `(a, c)` — partial; can seek on `a` but has to sort the matched rows by `c`
- Index `(c, a, b)` — bad; can't seek on `a` first

Leftmost-prefix rule: a composite index on `(a, b, c)` serves queries filtering on `a`, `a + b`, `a + b + c`, but **not** `b` alone or `c` alone.

### Covering indexes

For a query `SELECT id, total FROM orders WHERE customer_id = 1234`:

- Index on `customer_id` alone: planner seeks `customer_id`, then does a row lookup for `id` and `total`
- Index on `(customer_id) INCLUDE (id, total)` (Azure SQL) or `(customer_id, id, total)` (Postgres): covering — no row lookup

`INCLUDE` columns don't participate in seek but avoid the table fetch. Use when the query returns a few columns from a wide table.

### Index maintenance

- **Postgres**: index bloat from updates; `REINDEX CONCURRENTLY` or `pg_repack` to rebuild without locking. Monitor `pg_stat_user_indexes` for unused indexes.
- **Azure SQL**: fragmentation in `sys.dm_db_index_physical_stats`. Stop rebuilding on a schedule; rebuild on signal (>30% fragmentation, large index, range-scan workload).
- **MySQL**: `OPTIMIZE TABLE` for InnoDB rebuilds. Not free; schedule during low traffic.
- **Cosmos**: indexes maintained automatically; tuning is the *indexing policy* (which paths to index), not maintenance.

### Statistics

Planner decisions depend on statistics. Stale stats = wrong plan.

- **Postgres**: `ANALYZE table_name` updates stats. `autovacuum` does this automatically but lags behind huge writes. After bulk loads: `VACUUM ANALYZE`.
- **Azure SQL**: `UPDATE STATISTICS table_name` or auto-stats (default). For volatile data, set `AUTO_UPDATE_STATISTICS_ASYNC = ON` to avoid plan-compile stalls.
- **MySQL**: `ANALYZE TABLE table_name`. InnoDB samples pages; sample size matters for large tables.

When in doubt and the planner is making bad choices on a previously-fast query, **update stats first** before adding indexes.

## Common indexing mistakes

### Too many indexes

Every index costs RU/IO per write. Detect: `pg_stat_user_indexes` shows indexes with `idx_scan = 0`. Azure SQL `sys.dm_db_index_usage_stats` shows `user_seeks + user_scans + user_lookups = 0`. **Drop unused indexes.**

### Wrong index column order

Composite `(status, customer_id)` when most queries filter by `customer_id` and rarely by status. Reorder to `(customer_id, status)`.

### Missing covering columns

Frequent query selects 2 extra columns on top of a filtered set; key lookup dominates plan. Add `INCLUDE` columns (Azure SQL) or extend the index (Postgres).

### Function-wrapped predicate

`WHERE LOWER(email) = ?` won't use a plain index on `email`. Two fixes:
- **Functional index**: `CREATE INDEX ON users (LOWER(email))` (Postgres, Azure SQL via computed column + index)
- **Normalize at write**: store `email_lower` column; index it

### Implicit conversion

`WHERE varchar_col = N'foo'` (Unicode literal on varchar column) defeats index in Azure SQL. Match types in the query.

### `OR` conditions across columns

`WHERE a = ? OR b = ?` rarely uses indexes well. Rewrite as `UNION` of two queries, each using its own index.

## Anti-patterns

- **Tuning from intuition.** Get the plan. Read the plan. Then tune.
- **Adding indexes without measuring impact.** Each costs writes. Verify the query uses the new index and the planner's estimate improved.
- **Rebuilding indexes by cron.** Bad default; do it on signal.
- **Stats not updated after bulk load.** Plans go bad; queries get slow. Always `ANALYZE` / `UPDATE STATISTICS` after big writes.
- **Index every column.** Reduces write throughput; rarely improves reads beyond a few key indexes.
- **Composite index ordered for read but ignoring sort.** Index `(a, b)` serves `WHERE a AND b`; index `(a, b, c)` also serves `WHERE a AND b ORDER BY c` — choose based on sort needs.

## Verification questions

1. Is `auto_explain` (Postgres) / Query Store (Azure SQL) / Performance Schema (MySQL) / RU logging (Cosmos) enabled to capture slow queries automatically?
2. For the top 10 slowest queries: has each plan been read, the predicate verified, and the index validated?
3. Are unused indexes (zero scans over a week) dropped?
4. Are statistics current after bulk loads (`ANALYZE` / `UPDATE STATISTICS`)?
5. For Azure SQL: is `AUTO_UPDATE_STATISTICS_ASYNC = ON` to avoid compile stalls?
6. Are covering indexes (`INCLUDE` / extended index) used for the top read queries?

## What to read next

- `postgres-on-azure.md` — Postgres-specific index types, `pg_stat_statements`
- `azure-sql-on-azure.md` — Query Store deep-dive
- `cosmos-db-design.md` — Cosmos indexing policy, partition-key alignment
- `mongodb-on-azure.md` — Mongo explain() and index hints
- `query-anti-patterns.md` — engine-specific anti-pattern catalog
- `wait-stats-and-blocking.md` — when the query plan is fine but execution still slow
