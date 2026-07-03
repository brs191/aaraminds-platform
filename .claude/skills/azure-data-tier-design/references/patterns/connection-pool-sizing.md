# Pattern — Connection Pool Sizing (Postgres on Azure)

## Problem

Postgres connections are expensive — each is a backend process consuming ~10MB of RAM plus scheduler overhead. Application frameworks default to per-instance connection pools (HikariCP for Spring Boot, pgxpool for Go) that, when multiplied by container replicas + revisions + read replicas, can exhaust Postgres `max_connections` faster than expected. The result is connection refusal errors, blocked deploys (new revisions can't connect), and outages during traffic spikes. Sizing the pool — and adding PgBouncer in front when it's not enough — is the fix.

## Use when

- Designing connection pooling for a new Postgres-backed service on Container Apps, AKS, or App Service
- Auditing an existing service experiencing intermittent `FATAL: sorry, too many clients already` errors
- Sizing for a high-replica deployment (>10 replicas of the same service)
- Planning blue/green deploys where connection footprint briefly doubles

## Avoid when

- Single-replica dev / small service with abundant connection budget — overhead of PgBouncer isn't worth it
- Workloads that genuinely require session-pinned connections (LISTEN/NOTIFY, advisory locks, session-level prepared statements) — PgBouncer transaction mode breaks these; use session mode (which negates most pooling benefit)

## Implementation steps

### Step 1 — measure your `max_connections` budget

Postgres Flexible Server `max_connections` is tier-dependent. Verify the current value (Azure docs change):

```sql
SHOW max_connections;
```

Example values:
| Tier | max_connections |
|---|---|
| Burstable B1ms | ~85 |
| Burstable B2ms | ~171 |
| General Purpose D2s | ~429 |
| General Purpose D4s | ~859 |
| General Purpose D8s | ~1718 |
| Memory Optimized E8s | ~3438 |

Reserve ~30% headroom for admin tools, monitoring, replication, and Postgres internals. Working budget = `max_connections × 0.7`.

### Step 2 — compute steady-state demand

```
steady_state_connections = replicas × pool_size_per_replica + read_replica_consumers
```

Worked example:
- 5 replicas of an app, HikariCP `maximum-pool-size: 30`
- Steady state: 5 × 30 = 150 connections
- D4s budget (working): 859 × 0.7 = 601 → fits

### Step 3 — compute deploy-time demand (the silent killer)

Blue/green and rolling deploys briefly double the replica count. Container Apps' default revision strategy keeps old revisions warm for a grace window.

```
peak_connections = (replicas × 2) × pool_size_per_replica
```

Same worked example during deploy:
- 5 × 2 × 30 = 300 → still fits 601

But at 20 replicas:
- Steady: 20 × 30 = 600 → fits
- Peak (deploy): 40 × 30 = 1200 → exceeds 601 → connection refusals during every deploy

This is the most common surprise. **Always check peak, not steady state.**

### Step 4 — pool size per replica

Right-size the per-replica pool. Rule of thumb for OLTP:

```
pool_size = (vCPU × 2) + effective_disk_concurrency
```

For most cloud Postgres on SSD, `effective_disk_concurrency` is ~1, so `pool_size ≈ vCPU × 2 + 1`. Round up.

| App container vCPU | Pool size |
|---|---|
| 0.5 vCPU | 5–10 |
| 1 vCPU | 10 |
| 2 vCPU | 15–20 |
| 4 vCPU | 25–30 |

Don't size the pool by "I expect 100 concurrent requests". Most requests spend < 50ms in DB, so 10 connections handle 200 requests/sec at 50ms each. **A larger pool doesn't speed up Postgres; it queues more requests at the connection step.**

### Step 5 — add PgBouncer when replica × pool doesn't fit

Flexible Server has PgBouncer built in. Enable:

```hcl
resource "azurerm_postgresql_flexible_server_configuration" "pgbouncer_enabled" {
  name      = "pgbouncer.enabled"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "true"
}

resource "azurerm_postgresql_flexible_server_configuration" "pool_mode" {
  name      = "pgbouncer.pool_mode"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "transaction"
}

resource "azurerm_postgresql_flexible_server_configuration" "default_pool_size" {
  name      = "pgbouncer.default_pool_size"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "50"
}
```

App connects to port **6432** (PgBouncer) instead of **5432** (direct).

How PgBouncer changes the math:
- Application sees a connection (cheap, in-memory PgBouncer)
- PgBouncer maintains its own pool to Postgres (`default_pool_size` per user/db)
- Many app connections fan into few Postgres connections

Example: 1000 app connections via PgBouncer → 50 actual Postgres connections. Postgres connection budget consumed: 50. Application code unchanged.

### Step 6 — choose PgBouncer pool mode

| Mode | Behavior | Compatible with |
|---|---|---|
| **transaction** (default) | Connection returned to pool at transaction end | OLTP CRUD, most apps. **Default.** |
| **session** | Connection held for entire client session | Apps using LISTEN/NOTIFY, advisory locks, session prepared statements, SET. Negates most pooling benefit. |
| **statement** | Connection returned after each statement | Strictest; rarely used; breaks transactions |

**Transaction mode incompatibilities** to verify:
- LISTEN/NOTIFY — won't work
- Session-level `SET` (e.g., `SET TIME ZONE`) — lost between transactions; use `SET LOCAL` inside a transaction instead
- Prepared statements across transactions — won't work; JDBC driver setting `prepareThreshold=0` disables server-side prep
- Advisory locks held across transactions — won't work

For Spring Boot: set `spring.jpa.properties.hibernate.jdbc.lob.non_contextual_creation=true` and disable JDBC prepared statement caching at the driver level when on transaction-pooling.

### Step 7 — monitor in production

Required dashboards:

- `pg_stat_activity` — active, idle, idle in transaction counts vs `max_connections`
- PgBouncer stats (`SHOW POOLS`, exposed via the `pgbouncer` admin DB)
- App-side pool stats (HikariCP exposes JMX / Micrometer metrics; pgxpool exposes `Stats()`)

Alerts:
- `active connections > 70% of max_connections` for 5 min
- `wait_count` on app pool growing (HikariCP `hikaricp.connections.pending`)
- `pgbouncer cl_waiting > 0` for sustained periods

### Step 8 — cap revisions on Container Apps

Container Apps retains inactive revisions by default. Each retained revision holds connections during its grace window. Cap:

```hcl
resource "azurerm_container_app" "main" {
  # ...
  configuration {
    max_inactive_revisions = 3
  }
}
```

Or set revision mode to `Single` if you don't need blue/green: only one revision active at a time, connection footprint = steady state always.

## Trade-offs

| Choice | Gain | Cost |
|---|---|---|
| Larger app-side pool | More concurrent in-flight queries | Higher Postgres connection consumption |
| PgBouncer transaction mode | Massive connection fan-in | Lost LISTEN/NOTIFY, session features |
| PgBouncer session mode | Compatible with everything | Limited pooling benefit |
| Larger Postgres tier | More max_connections | Cost; doesn't fix bad pool sizing |
| Container Apps single revision mode | Predictable connection count | No blue/green; deploy = brief outage |

## Common failure modes

### Connection refused under deploy load
**Detection**: app logs show `FATAL: sorry, too many clients already` during deploys; `pg_stat_activity` count spikes above `max_connections`.
**Fix**: enable PgBouncer; cap inactive revisions; or reduce per-replica pool size.

### `idle in transaction` connections accumulate
**Detection**: `pg_stat_activity` shows many connections in `idle in transaction` state, sometimes for minutes.
**Fix**: app code holds transactions across external calls. Set `statement_timeout` and `idle_in_transaction_session_timeout` at the database level. Audit code for "transaction started, then HTTP call, then commit".

### PgBouncer transaction mode breaks LISTEN/NOTIFY
**Detection**: NOTIFY events arrive intermittently or never; payment / event notifications silent.
**Fix**: use session mode for the LISTEN/NOTIFY service; or move to a real event bus (Service Bus) and stop using Postgres NOTIFY for cross-service signaling.

### Connection storms during cold start
**Detection**: scaling event causes Postgres connection spike that briefly hits max.
**Fix**: lazy pool initialization (Hikari `initialization-fail-timeout=-1`), startup probes that succeed before traffic ramps, smaller initial pool that grows on demand.

### `prepareThreshold` left at default with PgBouncer transaction mode
**Detection**: random errors like `prepared statement "S_3" does not exist`. Particularly common with PostgreSQL JDBC.
**Fix**: set `prepareThreshold=0` in JDBC URL or driver config to disable server-side prepared statements when on PgBouncer transaction mode.

## MCP tool opportunities

- **`compute_pool_budget`** — given replicas, per-replica pool size, tier, deploy strategy, return the worst-case connection demand and whether it fits the tier budget. Recommend PgBouncer or tier upsize.
- **`detect_pool_overcommit`** — scan service config / Terraform for HikariCP / pgxpool settings and Container Apps replica counts; flag combinations exceeding budget.
- **`generate_pgbouncer_config`** — output the `azurerm_postgresql_flexible_server_configuration` blocks plus app-side driver tweaks for transaction-mode compatibility.

## What to read next

- `../postgres-on-azure.md` — Flexible Server tier, max_connections, PgBouncer enablement
- `partition-key-design.md` — sibling pattern for Cosmos
- `../data-migration-patterns.md` — connection budget during data migrations
- `../query-anti-patterns.md` — long transactions and other connection hogs
- `azure-microservices-observability` skill — pg_stat_activity and PgBouncer dashboards
