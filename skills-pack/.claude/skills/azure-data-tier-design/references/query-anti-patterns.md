# Query Anti-Patterns

## Purpose

Engine-specific query patterns that ship without complaint and degrade silently in production. Each entry names the symptom, the detection signal, and the fix. Reach for this reference during code review or when a service is suddenly slow.

## Postgres

### `SELECT *` in production code

**Symptom**: query plans change when columns are added; deserialization is wasteful; columns the app doesn't use are still pulled across the wire.

**Detection**: grep the codebase for `SELECT \*` or ORM equivalents (`.SelectAll()`, `entityManager.find()` without projection).

**Fix**: name columns. `SELECT id, email, status FROM orders WHERE customer_id = $1`. For ORMs, use explicit projections.

### Missing index on a high-cardinality WHERE column

**Symptom**: query slow at scale; `EXPLAIN ANALYZE` shows `Seq Scan` on a large table.

**Detection**: `pg_stat_statements` top-N by mean_time; `EXPLAIN ANALYZE` on the slowest queries.

**Fix**: `CREATE INDEX CONCURRENTLY ON table(column)`. `CONCURRENTLY` to avoid blocking writes. Index per WHERE column; composite index for multi-column filters or `WHERE a = ? ORDER BY b`.

### N+1 queries from ORM lazy loading

**Symptom**: a list view triggers N+1 round-trips — 1 for the list, N for the related rows.

**Detection**: high query count per request in slow-query log; pg_stat_statements shows many identical short queries.

**Fix**: eager-load via `JOIN` or `IN (...)` batch fetch. In JPA: `@EntityGraph` or `JOIN FETCH`. In ORMs (sqlx, pgx with sqlc): write the JOIN explicitly.

### Large `IN` list

**Symptom**: `WHERE id IN ($1, $2, ..., $5000)` exceeds parameter limits or plans poorly.

**Detection**: query with hundreds-of-thousands of parameters; protocol-level errors.

**Fix**: insert IDs into a TEMP TABLE or VALUES list, join against it. Or batch in chunks of ~1000.

### `OFFSET 100000` pagination

**Symptom**: pagination gets slower the further the user pages — Postgres scans and discards N rows before returning the page.

**Detection**: page latency grows with page number.

**Fix**: keyset pagination. `WHERE (created_at, id) < ($last_created_at, $last_id) ORDER BY created_at DESC, id DESC LIMIT 50`. Bookmark-based, constant-time per page.

### Long-running transaction in OLTP path

**Symptom**: connection held open for seconds, blocking vacuum and accumulating locks.

**Detection**: `pg_stat_activity` shows transactions with `xact_start` minutes old.

**Fix**: `SET LOCAL statement_timeout = '5s'`. Cap transaction time at the app or pool layer. Don't hold transactions across user input.

### Unindexed `LIKE '%foo%'` on large text

**Symptom**: full table scan on every search.

**Detection**: slow search endpoints; `EXPLAIN ANALYZE` shows `Seq Scan` with `Filter: text LIKE`.

**Fix**: `pg_trgm` extension + GIN index for substring search. For full-text, `tsvector` + `GIN` index. For huge corpora, push to Azure AI Search.

### JSON column without GIN index

**Symptom**: `WHERE jsonb_column @> '{"key": "value"}'` does full scan.

**Detection**: queries on JSONB columns running for seconds.

**Fix**: `CREATE INDEX CONCURRENTLY ON t USING GIN (jsonb_column)`. Or GIN on specific paths if you query a few paths often.

### `count(*)` on large table

**Symptom**: pagination shows total count; counts take 5+ seconds.

**Detection**: count queries in slow-query log on tables with millions of rows.

**Fix**: drop the count, or use `pg_class.reltuples` estimate (`SELECT reltuples FROM pg_class WHERE relname = 'orders'`). For exact counts under filter, materialize a counter table updated via triggers (only if exact count is load-bearing).

## Cosmos DB

### Cross-partition query at scale

**Symptom**: `SELECT * FROM c WHERE c.status = 'pending'` (no partition key in filter) is slow and expensive — fan-out across all physical partitions.

**Detection**: query metrics show `Index utilization < 100%` or activity log shows `cross partition` queries; `x-ms-request-charge` is unexpectedly high (>100 RU for simple-looking query).

**Fix**: include partition key in the filter: `WHERE c.tenantId = @t AND c.status = 'pending'`. If the access pattern truly needs to scan across partitions, consider a separate index or CQRS read model.

### Hot partition

**Symptom**: 429 throttling errors on a small subset of keys while overall RU/s is well under provisioned.

**Detection**: Cosmos metric `Normalized RU Consumption (Max)` near 100% while average is low. One physical partition is hot.

**Fix**: change the partition key (see `patterns/partition-key-design.md`). Or, if one tenant is hot, move that tenant to a dedicated container.

### Large items (>100KB)

**Symptom**: write RU cost is unexpectedly high; query RU cost includes loading large items.

**Detection**: `x-ms-request-charge` per insert > 30 RU. Item size visible in metrics.

**Fix**: split into multiple items (e.g., header + line items as separate items, joined client-side). Reduce included fields. Don't put blob-shaped data in Cosmos — use Blob Storage.

### Default indexing on high-write container

**Symptom**: writes cost more than expected; most paths are never queried.

**Detection**: per-write RU > 10 on small items; indexing policy is `IndexingMode: consistent` with `IncludedPaths: /*`.

**Fix**: explicit `includedPaths` for only the queried paths, `excludedPaths: /*`. Add composite indexes for multi-field queries. Reduces write RU by 30–50% typically.

### Scan-style query when point read would do

**Symptom**: query with full partition key + id (uniquely-identifying), written as `SELECT * FROM c WHERE c.id = @id AND c.pk = @pk`.

**Detection**: query is for a single document but uses `SELECT` syntax.

**Fix**: use point read SDK call (`container.read_item(id, partition_key)`). Point reads are 1 RU for small items; queries are at least 2.3 RU.

### `OFFSET ... LIMIT` pagination

**Symptom**: deep pagination on Cosmos slows like Postgres but bills RUs for skipped rows.

**Detection**: `x-ms-request-charge` grows with offset.

**Fix**: continuation tokens. The Cosmos SDK returns a token on each page; pass it to the next page request. Constant-cost per page.

### `ORDER BY` without supporting index

**Symptom**: sort fails or is very expensive.

**Detection**: `ORDER BY` returns error suggesting composite index, or RU charge is high.

**Fix**: add composite index for the `WHERE` + `ORDER BY` combination.

### Over-eager `SELECT *` projection

**Symptom**: query returns full document when the app uses 2 fields. Wire cost + RU.

**Detection**: `SELECT *` in app code; large response bodies.

**Fix**: `SELECT c.id, c.email FROM c WHERE ...`. Project only what's used.

## Azure SQL

### `SELECT *` with EF Core lazy loading

**Symptom**: N+1 query pattern from EF Core; loads all columns; serialization is wasteful.

**Detection**: Query Store top-N shows many short identical queries differing only in parameter; or `sys.dm_exec_query_stats` for a Procedure shows huge `execution_count`.

**Fix**: explicit projection: `_.Select(o => new { o.Id, o.Total })`. Disable lazy loading; use `Include()` for relationships.

### Implicit conversion in WHERE

**Symptom**: `WHERE varchar_col = N'foo'` — Unicode literal on varchar column causes full scan.

**Detection**: execution plan shows `CONVERT_IMPLICIT` warning on the predicate; index scan instead of seek.

**Fix**: match literal type to column type. Use `'foo'` (non-Unicode) for varchar; `N'foo'` for nvarchar.

### Cursors for set-based operations

**Symptom**: T-SQL stored procedure uses `DECLARE CURSOR`; slow on large sets.

**Detection**: Query Store shows the procedure with high CPU; cursor fetch loops.

**Fix**: rewrite as set-based. T-SQL is set-based; cursors are rarely the answer.

### Key Lookup dominating plan

**Symptom**: Nonclustered index used (seek) but then a Key Lookup for additional columns from the base table.

**Detection**: execution plan shows `Key Lookup (Clustered)` arrow as the expensive operator; missing-index hint may appear.

**Fix**: covering index with `INCLUDE` for the additional columns: `CREATE INDEX IX_orders_customer ON orders(customer_id) INCLUDE (total, status)`.

### Forced parameterization defeats index usage

**Symptom**: ad-hoc queries get parameterized aggressively by SQL Server; parameter sniffing leads to plan that's bad for some parameter values.

**Detection**: Query Store shows same query_id with wildly different runtimes across executions.

**Fix**: `OPTION (RECOMPILE)` for problematic queries; `OPTIMIZE FOR UNKNOWN` for stable-but-bad plans; or plan forcing if a specific plan is known-good.

### `OPTION (MAXDOP 1)` to "fix" parallelism issues

**Symptom**: someone added `OPTION (MAXDOP 1)` because a parallel plan was bad once. Now no parallelism anywhere.

**Detection**: Query Store shows the query with low DOP; CPU underutilized.

**Fix**: tune the actual issue (stats, parallelism cost threshold, or specific query hint with reason in comment). Don't blanket-disable parallelism.

### Long-running transaction in OLTP path

**Symptom**: locks held; blocking chains form; `LCK_M_*` waits dominate.

**Detection**: `sys.dm_exec_requests` shows long open transactions; `sys.dm_tran_active_transactions`.

**Fix**: cap transaction time at the app layer; use `SET LOCK_TIMEOUT 5000` for queries that must not wait long.

## MySQL

### UUID primary keys without ordered UUIDs

**Symptom**: InnoDB primary key is random UUID; insertions scatter across the clustered index; write performance degrades; secondary indexes bloat.

**Detection**: high write latency variance; large secondary index size relative to primary key data; `SHOW TABLE STATUS` shows growing index size.

**Fix**: use UUIDv7 (time-ordered) or `BIGINT AUTO_INCREMENT`. If stuck with random UUIDs and can't change, accept the cost and oversize storage.

### Default REPEATABLE READ surprising app code

**Symptom**: app expects READ COMMITTED semantics (most ORMs assume this) but MySQL is REPEATABLE READ; phantom reads don't happen but next-key locks cause unexpected blocking.

**Detection**: locking issues that don't match expected isolation; `SHOW ENGINE INNODB STATUS` shows gap locks.

**Fix**: set `transaction_isolation = 'READ-COMMITTED'` at server level (Flexible Server configuration). Verify with `SELECT @@transaction_isolation`.

### Missing index on a foreign-key column

**Symptom**: cascading delete or update is slow.

**Detection**: `EXPLAIN` on related query shows full scan.

**Fix**: index every FK column. InnoDB doesn't auto-index FK referencing columns.

### Implicit conversion (same as Azure SQL)

**Symptom**: `WHERE varchar_col = 123` (integer literal vs varchar column) causes full scan.

**Detection**: `EXPLAIN` shows `Using where` with full scan despite index existing.

**Fix**: quote string literals; match types.

### `OPTIMIZE TABLE` on a schedule

**Symptom**: scheduled `OPTIMIZE TABLE` blocks writes; tables briefly unavailable.

**Detection**: scheduled job; replication lag spikes during the window.

**Fix**: only `OPTIMIZE` when fragmentation actually matters; use `pt-online-schema-change` (Percona toolkit) or native online DDL where possible.

## Redis

### Default `noeviction` policy

**Symptom**: Redis hits memory limit; new writes start returning OOM error; downstream services fail.

**Detection**: `INFO memory` shows `maxmemory_policy:noeviction`; eviction count is zero but `used_memory` is at the cap.

**Fix**: set `maxmemory-policy = allkeys-lru` (or appropriate eviction policy). See `redis-on-azure.md`.

### `KEYS *` in production

**Symptom**: Redis blocks for seconds on every other operation while `KEYS` iterates.

**Detection**: client latency spikes; `SLOWLOG GET` shows `KEYS` commands.

**Fix**: use `SCAN` for iteration. Banish `KEYS` from the codebase.

### Large values (> 100KB)

**Symptom**: serialization cost dominates; network bandwidth saturated; eviction churns.

**Detection**: `DEBUG OBJECT key` shows large `serializedlength`.

**Fix**: split into smaller keys; or store the blob in Blob Storage with a Redis pointer.

### Cache stampede

**Symptom**: a hot key expires; thousands of concurrent requests hit the DB simultaneously.

**Detection**: DB query rate spikes; one identical query dominates the spike.

**Fix**: lock-and-load, probabilistic early expiration, stale-while-revalidate, or jittered TTLs. See `patterns/caching-patterns.md`.

### No TTL on session-like data

**Symptom**: Redis fills with stale sessions; eviction kicks in unpredictably; recently active users get evicted.

**Detection**: `KEYS *` shows many old keys; `OBJECT IDLETIME key` shows long idle times.

**Fix**: explicit TTL on every key set. Periodic audit: `TTL key` should return a positive number, not -1.

### Synchronous Redis call with no timeout

**Symptom**: Redis stall causes app stall; threads pile up; request queue backs up.

**Detection**: app pool exhausted while Redis is slow; thread dump shows waits on Redis client.

**Fix**: set client timeout (50–200ms typical). Circuit-break around Redis after N failures.

### Persistent connections from short-lived functions

**Symptom**: Azure Functions or serverless workers each open a new Redis connection; connection count explodes.

**Detection**: `CLIENT LIST` shows hundreds of connections; `maxclients` warning.

**Fix**: connection pool per process; in serverless, use a shared client (singleton pattern with proper init).

## MongoDB (Cosmos vCore, Cosmos RU-based, Atlas)

### Unbounded `$regex` without anchor

**Symptom**: `db.users.find({name: {$regex: 'smith', $options: 'i'}})` does full collection scan.

**Detection**: slow `find` with `$regex`; `explain()` shows `COLLSCAN`.

**Fix**: anchor the regex (`'^smith'` uses index prefix). Or use Atlas Search / Cosmos full-text if available. Case-insensitive regex usually can't use the index — store a normalized lowercase field for search.

### Missing index hint on selective field

**Symptom**: query slow; `explain()` shows full scan.

**Detection**: `explain('executionStats')` shows `totalDocsExamined` >> `nReturned`.

**Fix**: `db.collection.createIndex({field: 1})`. Compound index for multi-field queries. Test with `explain()` after.

### Large embedded array growing unbounded

**Symptom**: document size grows; updates get slow; eventually hits 16MB Mongo limit.

**Detection**: document size in metrics; specific documents with hundreds of array entries.

**Fix**: extract to separate collection with parent reference. Embed bounded arrays only.

### `$lookup` across large collections

**Symptom**: aggregation pipeline with `$lookup` to a large unindexed collection is very slow.

**Detection**: aggregation stage timings (`explain` on pipeline) shows `$lookup` dominating.

**Fix**: ensure index on the `foreignField`. Denormalize if `$lookup` is on the hot path — Mongo isn't optimized for joins.

### Default write concern in critical writes

**Symptom**: writes return success before replication; replica failover loses the write.

**Detection**: review of write code shows `{w: 1}` (default) on critical paths.

**Fix**: `{w: 'majority', j: true}` for important writes. Slower but durable.

### `findAndModify` for high-contention counter

**Symptom**: hot single document gets pinned under lock; throughput plateaus.

**Detection**: `db.currentOp()` shows many ops waiting on same `_id`.

**Fix**: shard the counter (multiple docs, sum client-side), or move counters out of Mongo (Redis INCR, dedicated counter service).

## Across all engines

### Logging the query but not the parameters

**Symptom**: slow query log shows `SELECT * FROM orders WHERE id = ?` — you can't reproduce.

**Fix**: log the parameters too, at least in a sampled fashion. For Postgres: `auto_explain.log_min_duration = '500ms'` + `auto_explain.log_parameter_max_length = -1`.

### Not running `EXPLAIN` / `explain()` in code review

**Symptom**: query looks fine; only at scale does it explode.

**Fix**: code review checklist includes `EXPLAIN ANALYZE` on every new query against a representative dataset. Cheap discipline; catches the worst surprises.

### Ignoring `pg_stat_statements` / Cosmos metrics until production complains

**Symptom**: slow queries discovered when users escalate.

**Fix**: weekly review of top-10 queries by total time. Add to oncall rotation or release checklist.

## Verification questions

1. Is `pg_stat_statements` enabled on every Postgres instance, with top-N reviewed weekly?
2. Are all Cosmos queries verified to include partition key in the WHERE clause?
3. Is the indexing policy customized on every high-write Cosmos container?
4. Are Postgres slow-query log and `auto_explain` enabled in prod?
5. For Mongo: does every aggregation pipeline have `explain()` output captured in code review?
6. Are pagination paths using keyset / continuation tokens, not OFFSET?

## What to read next

- `postgres-on-azure.md` — index strategy, slow-query log configuration
- `cosmos-db-design.md` — indexing policy, partition key, point-read vs query
- `mongodb-on-azure.md` — explain() patterns, aggregation pipeline tuning
- `patterns/partition-key-design.md` — preventing hot-partition anti-pattern
- `azure-microservices-observability` skill — query dashboards
