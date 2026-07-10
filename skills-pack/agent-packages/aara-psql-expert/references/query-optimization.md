# Query optimization and reading EXPLAIN

Optimization is evidence-driven: never guess, read the plan. The agent's first
move on any tuning task is to ask for `EXPLAIN (ANALYZE, BUFFERS)` output and the
relevant DDL, then reason from the actual node costs and row estimates.

## The one command that matters

```sql
EXPLAIN (ANALYZE, BUFFERS, VERBOSE, SETTINGS) <query>;
```

- `ANALYZE` runs the query and reports **actual** time and rows (careful: it
  executes — for writes, wrap in a transaction you `ROLLBACK`).
- `BUFFERS` shows shared hit/read — the truest signal of I/O.
- Compare **estimated rows vs actual rows** at each node. A large gap is the
  root cause of most bad plans: the planner chose a strategy on wrong
  cardinality.

## Reading a plan

Read inside-out, bottom-up. For each node check: estimated vs actual rows, actual
time, loops, and buffers. Watch for:

- **Seq Scan on a big table** with a selective filter → missing or unusable
  index, or a non-sargable predicate.
- **Rows Removed by Filter** large → the index isn't covering the predicate;
  consider a partial or composite index.
- **Nested Loop** with high `loops` and a big inner side → bad join order from a
  cardinality misestimate; often fixed by better stats or an index.
- **Hash Join** spilling to disk (`Batches > 1`) → `work_mem` too low for this
  query, or the build side is larger than estimated.
- **Sort** with `Sort Method: external merge Disk` → raise `work_mem` for the
  session or add an index that provides the order.
- **Bitmap Heap Scan** with high `Heap Blocks: exact/lossy` → many rows per
  block; a different index or clustering may help.

## Sargability — the most common self-inflicted slow query

A predicate must be expressible against the index to use it. Common breakers:

- `WHERE lower(email) = $1` won't use a plain index on `email` → use an
  **expression index** `CREATE INDEX ON users (lower(email))`.
- `WHERE created_at::date = $1` wraps the column in a function → rewrite as a
  range `WHERE created_at >= $1 AND created_at < $1 + 1`.
- `WHERE col + 0 = $1`, `WHERE col LIKE '%x'` (leading wildcard), implicit type
  casts (`bigint_col = '123'` vs `= 123`) — all can defeat an index.

## Statistics

The planner depends on `ANALYZE` statistics. Fixes for misestimates:

- Stale stats → `ANALYZE table;` (autovacuum usually handles this; heavy-churn
  tables may need tuned autovacuum).
- Correlated columns (planner assumes independence) → **extended statistics**:
  `CREATE STATISTICS s (dependencies, ndistinct) ON a, b FROM t;` then `ANALYZE`.
- Skewed columns → raise `default_statistics_target` or per-column
  `ALTER TABLE ... ALTER COLUMN c SET STATISTICS 1000;`.
- `[PG18]` `pg_upgrade` now carries most optimizer statistics across major
  upgrades, so plans stabilize faster post-upgrade.

## Join and scan strategy quick reference

- Nested loop: best when the outer side is tiny and the inner has an index.
- Hash join: best for large, unsorted equi-joins that fit `work_mem`.
- Merge join: best when both inputs are already sorted on the join key.
- Index-only scan: possible when all needed columns are in the index (use
  `INCLUDE` covering indexes) and the visibility map is fresh (vacuum).

## PostgreSQL 18 performance notes

- **Async I/O** speeds sequential scans, bitmap heap scans, and vacuum up to ~3×
  on capable storage. A seq scan is less catastrophic than on older versions —
  but a selective query should still use an index.
- **Skip scan** lets a multicolumn B-tree be used even when the leading column
  isn't in the predicate, in more cases than before. This changes some
  index-design tradeoffs (see `indexing.md`).

## Tuning workflow the agent should follow

1. Get `EXPLAIN (ANALYZE, BUFFERS)` and the DDL (tables, existing indexes).
2. Find the node where estimated and actual rows diverge most.
3. Classify: missing/unusable index, non-sargable predicate, bad stats, or
   insufficient `work_mem`.
4. Propose the smallest fix (rewrite predicate, add/adjust index, refresh/extend
   stats, session `work_mem`), and predict the plan change.
5. Draft the change; never claim a speedup without the plan evidence.
