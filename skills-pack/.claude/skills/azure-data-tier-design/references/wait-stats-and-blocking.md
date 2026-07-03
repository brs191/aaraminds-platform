# Wait Stats and Blocking — Cross-Engine Diagnostics

## Purpose

Slow queries are sometimes slow because the *plan* is bad (covered in `query-execution-and-indexing.md`). They are *also* slow because the query is waiting — on locks, on I/O, on memory grants, on log writes. Wait stats answer "why is the query slow." This reference covers Postgres, Azure SQL, MySQL, Cosmos, and Mongo wait diagnostics, plus blocking and deadlocks.

## The two questions

For any slow query:

1. **Is the plan bad?** → `query-execution-and-indexing.md`
2. **Is it waiting?** → this reference

Both can be true. Wait stats let you tell the difference.

## Postgres — `pg_stat_activity` + `pg_wait_sampling`

`pg_stat_activity` shows current sessions and what they're waiting on:

```sql
SELECT pid, state, wait_event_type, wait_event, query, backend_start
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY backend_start;
```

Wait event types (Postgres 16):

| Wait event type | Meaning |
|---|---|
| `LWLock` | Internal lightweight lock (buffer mapping, etc.) — usually transient |
| `Lock` | Row / table lock contention |
| `BufferPin` | Waiting on buffer pin; usually transient |
| `IO` | Waiting on disk I/O |
| `Client` | Waiting on client (network) |
| `Activity` | Background process idle |

For **historical** wait stats, install `pg_wait_sampling` extension:

```sql
CREATE EXTENSION pg_wait_sampling;

SELECT event_type, event, count(*) AS samples
FROM pg_wait_sampling_history
WHERE ts > now() - interval '1 hour'
GROUP BY event_type, event
ORDER BY samples DESC;
```

The extension samples `pg_stat_activity` and aggregates. Without it, you only see the current moment.

### Blocking in Postgres

```sql
SELECT
    blocked.pid          AS blocked_pid,
    blocked.query        AS blocked_query,
    blocking.pid         AS blocking_pid,
    blocking.query       AS blocking_query,
    blocking.state       AS blocking_state
FROM pg_stat_activity blocked
JOIN pg_locks bl ON bl.pid = blocked.pid AND NOT bl.granted
JOIN pg_locks bg ON bg.locktype = bl.locktype
                 AND bg.relation IS NOT DISTINCT FROM bl.relation
                 AND bg.granted
JOIN pg_stat_activity blocking ON blocking.pid = bg.pid
WHERE blocked.pid != blocking.pid;
```

Common cause: long-running transaction holding a lock. Find the offender, decide if it's safe to kill: `SELECT pg_cancel_backend(pid)` (gentle) or `pg_terminate_backend(pid)` (forceful).

### Postgres deadlocks

Postgres detects deadlocks and aborts one of the transactions (returns SQLSTATE `40P01`). Configure log capture:

```sql
log_lock_waits = on
deadlock_timeout = 1s
```

Deadlock messages land in the Postgres log; review weekly. Fix patterns are universal: consistent lock order, shorter transactions, lower isolation where safe.

## Azure SQL — `sys.dm_os_wait_stats`

The diagnostic table:

```sql
SELECT TOP 20 wait_type, wait_time_ms, waiting_tasks_count,
       signal_wait_time_ms, max_wait_time_ms
FROM sys.dm_os_wait_stats
WHERE wait_type NOT IN (
  'SLEEP_TASK', 'CHECKPOINT_QUEUE', 'XE_TIMER_EVENT',
  'BROKER_TASK_STOP', 'BROKER_RECEIVE_WAITFOR',
  'LAZYWRITER_SLEEP', 'CLR_AUTO_EVENT', 'DIRTY_PAGE_POLL'
)
ORDER BY wait_time_ms DESC;
```

Top types and meaning (recapped from `azure-sql-on-azure.md`):

| Wait | Indicates |
|---|---|
| `PAGEIOLATCH_*` | Slow I/O / large scans |
| `LCK_M_*` | Lock contention |
| `WRITELOG` | Slow log writes |
| `RESOURCE_SEMAPHORE` | Memory grant pressure |
| `CXPACKET`, `CXCONSUMER` | Parallelism imbalance |
| `ASYNC_NETWORK_IO` | Slow consumer (app-side) |
| `THREADPOOL` | Worker thread exhaustion — connection storm |

Reset stats periodically (`DBCC SQLPERF('sys.dm_os_wait_stats', CLEAR)`) so you measure recent windows, not cumulative-since-restart. Snapshot daily / hourly into a custom table for trending.

### Current waiting tasks

```sql
SELECT session_id, wait_duration_ms, wait_type, blocking_session_id, resource_description
FROM sys.dm_os_waiting_tasks
ORDER BY wait_duration_ms DESC;
```

### Azure SQL blocking

```sql
SELECT blocking_session_id, session_id, wait_type, wait_time,
       status, command, last_wait_type
FROM sys.dm_exec_requests
WHERE blocking_session_id <> 0;
```

Get the offending SQL:

```sql
SELECT er.session_id, st.text
FROM sys.dm_exec_requests er
CROSS APPLY sys.dm_exec_sql_text(er.sql_handle) st
WHERE er.blocking_session_id <> 0 OR er.session_id IN (
  SELECT blocking_session_id FROM sys.dm_exec_requests WHERE blocking_session_id <> 0
);
```

### Azure SQL deadlocks via Extended Events

```sql
CREATE EVENT SESSION [deadlock_capture] ON SERVER
ADD EVENT sqlserver.xml_deadlock_report
ADD TARGET package0.event_file (
  SET filename = 'deadlock_capture.xel', max_file_size = 5
)
WITH (STARTUP_STATE = ON);

ALTER EVENT SESSION [deadlock_capture] ON SERVER STATE = START;
```

Review captured deadlock graphs in SSMS (open the .xel file). Each graph shows the resources and processes involved — match to the application code path.

## MySQL — Performance Schema

```sql
SELECT EVENT_NAME, SUM_TIMER_WAIT, COUNT_STAR
FROM performance_schema.events_waits_summary_global_by_event_name
ORDER BY SUM_TIMER_WAIT DESC
LIMIT 20;
```

Common wait events:

| Event | Meaning |
|---|---|
| `wait/io/file/*` | File I/O |
| `wait/synch/mutex/*` | Internal mutex contention |
| `wait/lock/table/*` | Table-level lock |
| `wait/io/socket/*` | Network |

### MySQL blocking

```sql
SELECT b.trx_id AS blocking_trx, w.trx_id AS waiting_trx,
       b.trx_query AS blocking_query, w.trx_query AS waiting_query
FROM information_schema.innodb_lock_waits lw
JOIN information_schema.innodb_trx b ON b.trx_id = lw.blocking_trx_id
JOIN information_schema.innodb_trx w ON w.trx_id = lw.requesting_trx_id;
```

MySQL deadlocks: `SHOW ENGINE INNODB STATUS\G` — section "LATEST DETECTED DEADLOCK" shows the most recent. For historical: enable `innodb_print_all_deadlocks` (writes to the error log).

## Cosmos DB — the wait is throttling

Cosmos doesn't have lock-based waits — concurrency is managed via RU budget. The Cosmos equivalent of "blocked" is **429 Too Many Requests** (RU exceeded).

Detection:

- Azure Monitor metric: `Total Requests` filtered by `StatusCode = 429`
- Per-partition `Normalized RU Consumption (Max)` — if one partition hits 100%, that partition is throttling
- Application logs: SDK retries 429s automatically with backoff; the retry latency appears as elevated p99

Fixes:

1. **Hot partition** — re-partition (`patterns/partition-key-design.md`)
2. **Burst traffic exceeding autoscale max** — raise the max ceiling or smooth bursts
3. **Inefficient queries causing RU spikes** — `query-anti-patterns.md`
4. **Adding indexes to reduce read RU** — see `cosmos-db-design.md` indexing policy

Cosmos provides `Diagnostics.ToString()` in SDK responses with per-operation RU and timing breakdown — log it during incidents.

## Mongo — `db.currentOp()` and Performance Advisor

Current operations:

```javascript
db.currentOp({active: true, secs_running: {$gt: 5}})
```

Shows long-running operations with op type, query, lock state.

For Atlas: Performance Advisor surfaces slow queries with index recommendations. Use it in code review, not just incidents.

Mongo write concerns and slow secondaries: if writes are slow on `{w: 'majority'}`, a secondary is lagging or saturated. Check oplog window: `rs.printReplicationInfo()`.

## Universal pattern — top waits + top queries

Operations rhythm for any engine:

1. **Weekly**: pull top-10 waits from the last 7 days. If a wait type dominates (>30%), investigate.
2. **Weekly**: pull top-10 queries by total time (`pg_stat_statements`, Query Store, Performance Schema, Cosmos `x-ms-request-charge` aggregated, Mongo slow query log). Cross-reference with the top waits.
3. **Incident**: capture current waits + current queries during the spike; compare with baseline.
4. **Pre-deploy**: any new query against representative data; capture plan + wait profile.

## Common wait-related anti-patterns

- **Long transactions in OLTP path.** Lock holders. Cap with `statement_timeout` (Postgres) or app-side transaction time limit.
- **Reading from cache, then doing slow DB write inside the lock window.** Whole request blocks behind one slow caller.
- **Connection pool sized too large.** Worker threads exhausted, `THREADPOOL` waits in Azure SQL. See `patterns/connection-pool-sizing.md`.
- **Hot single row** (counter, status). Lock contention on every update. Move to Redis INCR or shard the counter.
- **Bulk update in OLTP path.** Holds locks; blocks readers. Move to off-hours or batch with sleeps.
- **No wait-stats baseline.** When something gets slow, you don't know if the wait profile changed. Snapshot weekly.

## Verification questions

1. For Postgres: is `pg_wait_sampling` extension enabled, with weekly review of top waits?
2. For Azure SQL: are wait stats snapshotted (cleared and re-baselined) at a known cadence, not cumulative since startup?
3. For Azure SQL: is the deadlock XEvent session running, with deadlock graphs reviewed?
4. For Cosmos: is `Normalized RU Consumption (Max)` per partition alerted, distinct from average RU?
5. For Mongo: is `currentOp()` reviewed during slowness events, and is the slow query log surfaced to the team?
6. Across engines: is there a documented escalation runbook for "the DB is slow" — wait stats → top queries → blocking → action?

## What to read next

- `query-execution-and-indexing.md` — the other half of "why is this slow"
- `transactions-and-isolation.md` — isolation choice affects lock waits
- `query-anti-patterns.md` — engine-specific patterns producing waits
- `patterns/connection-pool-sizing.md` — pool sizing as a wait-stats root cause
- `azure-microservices-observability` skill — dashboard wiring
