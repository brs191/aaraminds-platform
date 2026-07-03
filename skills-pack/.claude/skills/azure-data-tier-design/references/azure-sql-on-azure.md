# Azure SQL on Azure — Operational Design

## Purpose

Azure SQL is not the pack's default — the stack defaults relational to Postgres. This reference exists because you will land on Azure SQL workloads (lift-and-shift from on-prem SQL Server, .NET-ecosystem teams with deep T-SQL expertise, vendor mandates) and "rewrite to Postgres" is rarely the right answer on the short clock. Coverage: flavor choice, tier selection, **Query Store** (the killer feature most teams underuse), wait stats, deadlocks, RCSI, HA/DR, and the anti-patterns that bite in production.

## When you'd land on Azure SQL

- Existing on-prem SQL Server lifting to Azure (Managed Instance path)
- .NET-heavy team; T-SQL / SSMS depth; Entity Framework Core production code
- Workload uses SQL Server-specific features without clean Postgres equivalents (Service Broker, In-Memory OLTP / Hekaton, Always Encrypted with secure enclaves, complex CLR procs, SQL Agent jobs)
- Greenfield: pick Postgres unless one of the above applies — Postgres is the pack default for new work

## The three flavors

| Flavor | When |
|---|---|
| Azure SQL Database (PaaS) | New cloud-native services; most managed, least feature-flexible |
| Azure SQL Managed Instance | Lift-and-shift from on-prem SQL Server (cross-database queries, SQL Agent, Service Broker, CLR) |
| SQL Server on Azure VM | Niche; only if you need features unavailable on PaaS or MI |

For new work on Azure SQL: **Database**. For lift: **Managed Instance**.

## Tier selection — Azure SQL Database

| Service tier | Purchase model | Use case |
|---|---|---|
| General Purpose (vCore) | Provisioned or Serverless | Default OLTP |
| Business Critical | Provisioned | Latency-sensitive OLTP, zone redundancy, AlwaysOn replicas built-in |
| Hyperscale | Provisioned | >4TB or rapid scale-up; uses page-server architecture for storage |
| Serverless (GP only) | Auto-pause | Bursty dev or low-traffic prod with idle gaps |

The DTU model is legacy. **Default to vCore** for all new work. For prod: Business Critical or Hyperscale (zone-redundant); GP for cost-sensitive paths.

## Query Store — turn it on, leave it on, look at it weekly

Query Store is the single most useful Azure SQL feature most teams underuse. It captures:

- Top-N queries by CPU, duration, logical reads, executions
- Query plan history with regression detection (the same SQL gets a different plan after stats change; you can see the regression)
- Wait stats per query
- **Plan forcing** — revert to the old plan when a new one regresses

Enable at provision time:

```sql
ALTER DATABASE [orders] SET QUERY_STORE = ON;
ALTER DATABASE [orders] SET QUERY_STORE (
  OPERATION_MODE          = READ_WRITE,
  CLEANUP_POLICY          = (STALE_QUERY_THRESHOLD_DAYS = 30),
  QUERY_CAPTURE_MODE      = AUTO,
  SIZE_BASED_CLEANUP_MODE = AUTO,
  MAX_STORAGE_SIZE_MB     = 1000
);
```

Azure Portal: Database → Query Performance Insight. Top-N queries view is the first thing to open when "something is slow."

Plan forcing example — query regressed after a stats update:

```sql
EXEC sp_query_store_force_plan @query_id = 73, @plan_id = 124;
```

Forced plans persist across upgrades. Document them; revisit quarterly. A forced plan you forgot about is a future surprise.

## Wait stats — the diagnostic table

```sql
SELECT TOP 20 wait_type, wait_time_ms, waiting_tasks_count
FROM sys.dm_os_wait_stats
WHERE wait_type NOT IN (
  'SLEEP_TASK', 'BROKER_TASK_STOP', 'CHECKPOINT_QUEUE',
  'LAZYWRITER_SLEEP', 'XE_TIMER_EVENT', 'BROKER_RECEIVE_WAITFOR'
)
ORDER BY wait_time_ms DESC;
```

Top types and meaning:

| Wait | Means | Fix direction |
|---|---|---|
| `PAGEIOLATCH_*` | Slow storage / large scans | Premium tier; review plans for table scans |
| `LCK_M_*` | Lock waits | Check blocking chains (`sys.dm_exec_requests`) |
| `WRITELOG` | Slow transaction log writes | Premium storage; shorter transactions |
| `RESOURCE_SEMAPHORE` | Memory grants exhausted | Reduce query memory; tier upsize |
| `CXPACKET` / `CXCONSUMER` | Parallelism imbalance | Tune MAXDOP (start with 8 → 4 on small DBs) |
| `ASYNC_NETWORK_IO` | Slow consumer | App-side; not the DB |

`sys.dm_os_waiting_tasks` shows what is currently waiting and on what session — pair with Query Store top-N to find the offender.

## Blocking and deadlocks

**Blocking**: long-running transaction holds a lock another transaction wants.

```sql
SELECT blocking_session_id, session_id, wait_type, wait_time, status
FROM sys.dm_exec_requests
WHERE blocking_session_id <> 0;
```

**Deadlocks**: cycle between two or more transactions. Detected by SQL Server's deadlock monitor; the loser is killed (error 1205).

Capture deadlocks via Extended Events (XEvent):

```sql
CREATE EVENT SESSION [deadlock_capture] ON SERVER
ADD EVENT sqlserver.xml_deadlock_report
ADD TARGET package0.event_file (SET filename = 'deadlock_capture.xel', max_file_size = 5)
WITH (STARTUP_STATE = ON);

ALTER EVENT SESSION [deadlock_capture] ON SERVER STATE = START;
```

Common deadlock pattern: two transactions touch the same two rows in opposite order.

**Fixes (in order of preference):**

1. **READ_COMMITTED_SNAPSHOT (RCSI)** — readers use row versions, eliminating read-write deadlocks (which are most of them).
2. **Consistent access order** — when transactions touch multiple rows/tables, always lock in the same order (e.g., always order_id ASC).
3. **Shorter transactions** — commit faster, hold locks less.
4. **Lower isolation** where safe — READ COMMITTED with RCSI is usually correct for OLTP.
5. **Retry on 1205** — application-level retry on deadlock victim error.

## Transactions and isolation

Default isolation: **READ COMMITTED with locking**. The right default for OLTP on Azure SQL is **READ COMMITTED with RCSI**.

Enable RCSI:

```sql
ALTER DATABASE [orders] SET READ_COMMITTED_SNAPSHOT ON;
ALTER DATABASE [orders] SET ALLOW_SNAPSHOT_ISOLATION ON;
```

Effect: readers see a committed snapshot, no shared lock; writers still take exclusive lock. Most read-write deadlocks vanish. Requires brief database lock during enablement — do during a maintenance window.

Isolation levels:

| Level | Default lock behavior | Phantom reads | Write skew |
|---|---|---|---|
| READ UNCOMMITTED | Dirty reads | Yes | Yes |
| READ COMMITTED (default) | Lock for read | Yes | Yes |
| READ COMMITTED + RCSI | Snapshot for read | Yes | Yes |
| REPEATABLE READ | Shared lock held | Yes | Maybe |
| SNAPSHOT | Snapshot whole transaction | No | **Yes — write skew possible** |
| SERIALIZABLE | Range locks | No | No |

Avoid SERIALIZABLE unless a business rule requires it; the lock contention is real. Use SNAPSHOT for "I need consistent reads across multiple queries" but verify write skew handling.

## Indexing

- **Clustered index** = the table's physical sort order; one per table; usually the PK
- **Nonclustered indexes** for query filters; `INCLUDE` clause for covering indexes (avoids key lookups)
- **Columnstore indexes** for analytical / aggregation queries; do not use for OLTP point reads
- **Filtered indexes** for sparse predicates (`WHERE status = 'pending'` where 95% are 'closed')
- **Missing index DMV**: `sys.dm_db_missing_index_details` — Azure SQL's suggestions; validate (it suggests wide indexes generously)

Fragmentation: modern advice is to **stop rebuilding indexes on a schedule**. Page splits hurt; rebuilds are expensive. Only rebuild when:
- Fragmentation > 30% (`sys.dm_db_index_physical_stats`), AND
- The index is large (>1000 pages), AND
- Queries actually do range scans on it

For most OLTP indexes used for point reads, fragmentation doesn't matter.

## Connection management

- Default ADO.NET pool: 100 per process per connection string
- Connection limit per database depends on tier (GP ~1000–2000; BC much higher)
- **Use Managed Identity** (not SQL auth):
  ```csharp
  var connStr = "Server=tcp:sql-orders.database.windows.net,1433;Database=orders;Authentication=Active Directory Default;";
  ```
- Retry transient errors (40197, 40501, 10928) — use EF Core's `EnableRetryOnFailure()` or Polly's `SqlAzureExecutionStrategy`. Don't roll your own.

## HA / DR

- **Business Critical** tier: zone-redundant by config; AlwaysOn replicas built in; 99.995 SLA
- **Failover groups**: automated cross-region failover; read-only secondary endpoint
- **Active geo-replication**: up to 4 secondaries in any region
- **Backups**: PITR 1–35 days; Long-Term Retention up to 10 years

Failover groups in Terraform:

```hcl
resource "azurerm_mssql_failover_group" "fg" {
  name      = "fg-orders"
  server_id = azurerm_mssql_server.primary.id

  partner_server {
    id = azurerm_mssql_server.secondary.id
  }

  databases = [azurerm_mssql_database.orders.id]

  read_write_endpoint_failover_policy {
    mode          = "Automatic"
    grace_minutes = 60
  }
}
```

Test failover quarterly. Failover via Azure CLI:

```bash
az sql failover-group set-primary \
  --name fg-orders --resource-group rg-prod \
  --server sql-orders-secondary
```

## Anti-patterns

- **Index rebuild on a maintenance schedule.** Fragmentation matters less than people think. Rebuild on signal, not schedule.
- **Cursors for set-based operations.** T-SQL is set-based; cursors are slow. Rewrite as JOINs / window functions.
- **Triggers doing business logic.** Hides logic from app review; debugging is misery. Keep business logic in services.
- **No Query Store enabled.** When 2am pager fires on slow query, you have no history. Always on.
- **DTU pricing model.** Legacy. Use vCore.
- **Implicit conversion in WHERE clauses.** `WHERE varchar_col = N'foo'` (Unicode literal on varchar column) causes scan. Match types exactly.
- **`SELECT *` with EF Core's lazy loading.** N+1 queries; loads all columns. Project explicitly.
- **SQL auth in prod.** Use Managed Identity. Stored credentials are a SOC 2 finding.

## Verification questions

1. Is Query Store enabled in READ_WRITE, with at least 30-day retention and Top-N reviewed weekly?
2. Is READ_COMMITTED_SNAPSHOT enabled (and ALLOW_SNAPSHOT_ISOLATION ON)?
3. Are deadlock XEvents captured and reviewed? Is there a deadlock-rate alert?
4. Are failover groups configured and tested quarterly?
5. Is connection retry handled by EF Core's `EnableRetryOnFailure()` or Polly, not hand-rolled?
6. Is auth Managed Identity (Entra ID), not SQL auth?
7. For Business Critical / Hyperscale: are zone-redundant deployments confirmed in Terraform?

## What to read next

- `engine-selection.md` — Azure SQL vs Postgres choice
- `query-execution-and-indexing.md` — cross-engine query analysis incl. Query Store deep-dive
- `wait-stats-and-blocking.md` — diagnostic patterns across engines
- `transactions-and-isolation.md` — isolation level matrix
- `ha-dr-data-tier.md` — failover groups, RPO/RTO targets
- `data-migration-patterns.md` — schema changes; zero-downtime in Azure SQL
