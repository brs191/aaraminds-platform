# Postgres on Azure — Operational Design

## Purpose

Run Postgres on Azure without learning the same lessons everyone else has. This reference covers tier selection, HA mode, connection management with PgBouncer, backups and PITR, extensions, networking, and the dashboards you need before go-live. The defaults here assume Azure Database for PostgreSQL Flexible Server unless otherwise stated.

## Service choice — Flexible Server (default)

Azure offers two managed Postgres products:

| Product | When |
|---|---|
| Azure Database for PostgreSQL **Flexible Server** | Default. Single-instance Postgres with zone-redundant HA, full extension surface, built-in PgBouncer. Up to ~96 vCore, ~16TB. |
| Azure Cosmos DB for PostgreSQL (**Hyperscale Citus**) | Only when data > 10TB or write rate exceeds single-instance Postgres. Sharded Postgres; queries must be shard-aware. |

Single Server is retired — do not deploy it. Migrate any remaining Single Server instances to Flexible Server.

## Tier selection

Flexible Server tiers:

| Tier | Use case | Cost shape |
|---|---|---|
| **Burstable** (B-series) | Dev, low-traffic prod, sidecar databases | Cheapest; CPU credits cap sustained load |
| **General Purpose** (D-series) | Default for prod backend services | Predictable CPU, good for OLTP |
| **Memory Optimized** (E-series) | Memory-heavy workloads, large `shared_buffers`, in-memory analytics | More RAM per vCore, ~30% more $ |

Sizing rule of thumb for OLTP: start at General Purpose D2s_v3 or D4s_v3, monitor CPU and memory pressure for 2 weeks, resize if sustained > 70%. Don't pre-size to D8s "for headroom" — the bill is real and you can scale up online.

**`max_connections` scales with tier.** Examples (verify on the current Azure docs as values change):
- D2s_v3: ~429
- D4s_v3: ~859
- D8s_v3: ~1718
- E8s_v3: ~3438

This is the most common surprise — connection limits are a function of tier, not of `shared_buffers` configuration.

## High availability

Three HA modes:

| Mode | RPO | RTO | Cost |
|---|---|---|---|
| **Disabled** | data loss possible on AZ failure | manual recovery | 1× |
| **Same-Zone HA** | ~0 | ~60–120s | 2× |
| **Zone-Redundant HA** | ~0 | ~60–120s | 2× |

For prod: **Zone-Redundant HA**. Same-Zone HA defeats the purpose of HA for any failure that takes out a zone.

Verify failover behaviour during your readiness review — trigger a failover via Azure CLI in staging:

```bash
az postgres flexible-server restart \
  --resource-group rg-prod \
  --name pg-orders-prod \
  --failover Forced
```

Measure actual RTO. Plan for 90s of connection refusal during failover and verify the application retries appropriately.

## Connection management — PgBouncer is not optional

Postgres connections are expensive (each is a backend process). Direct application-to-Postgres connections at scale exhaust `max_connections` quickly. PgBouncer multiplexes many client connections into a small server pool.

**Flexible Server has PgBouncer built in.** Enable it:

```hcl
resource "azurerm_postgresql_flexible_server_configuration" "pgbouncer" {
  name      = "pgbouncer.enabled"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "true"
}

resource "azurerm_postgresql_flexible_server_configuration" "pool_mode" {
  name      = "pgbouncer.pool_mode"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "transaction"
}
```

Pool modes:
- **transaction** (default for OLTP) — connection returned to pool at transaction end. Cannot use prepared statements across transactions, session variables, or LISTEN/NOTIFY.
- **session** — connection held for the duration of the client session. Compatible with all Postgres features but defeats most of the pooling benefit.
- **statement** — connection returned after each statement. Strictest; rare.

**Application port**: when PgBouncer is enabled, connect to port `6432` (pooled) or `5432` (direct). Use `6432` for application traffic. Use `5432` for admin tasks that need session state.

See `patterns/connection-pool-sizing.md` for the math.

## Read replicas

Flexible Server supports up to 5 read replicas in the same or different regions.

When to use:
- Read load saturates the primary
- Reporting queries can tolerate seconds of staleness
- Need read endpoints in another region for latency (multi-region read, single-region write)

When **not** to use:
- "For HA" — read replicas are not an HA mechanism. Use Zone-Redundant HA for that.
- For caching — use Redis. Replicas cost full Postgres tier each.

Replication lag is the operational hazard. Monitor `pg_stat_replication.replay_lag` on the primary; alert if > 30s sustained. If the app reads from a replica then writes to the primary, the user can write a value and not see it on the next read.

## Backups and PITR

Flexible Server provides:
- **Automatic backups** retained 7–35 days (configurable at provision time)
- **Point-in-Time Restore** to any second within retention window
- **Geo-redundant backups** if `geo_redundant_backup_enabled = true` (recommended for prod)

Configure in Terraform:

```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  # ...
  backup_retention_days        = 35
  geo_redundant_backup_enabled = true
}
```

PITR creates a *new server* — it does not restore in place. Plan the cutover. Test PITR quarterly: restore to a scratch server, run a smoke query, delete. If you've never restored, you don't have backups.

## Extensions

Allow-list extensions per server. The pack-default list:

```hcl
resource "azurerm_postgresql_flexible_server_configuration" "extensions" {
  name      = "azure.extensions"
  server_id = azurerm_postgresql_flexible_server.main.id
  value     = "PGVECTOR,POSTGIS,PG_PARTMAN,PG_CRON,PG_STAT_STATEMENTS,UUID-OSSP"
}
```

Useful ones:
- **`pg_stat_statements`** — query performance instrumentation. Enable everywhere; the overhead is negligible and the visibility is huge.
- **`pgvector`** — vector similarity search. The default for RAG / embeddings unless the dataset is huge.
- **`postgis`** — geospatial. Only if actually geospatial; it's heavy.
- **`pg_partman`** — table partitioning automation (time-series, multi-tenant).
- **`pg_cron`** — schedule jobs inside Postgres. Convenient for cleanup tasks; don't put business logic in cron jobs.
- **`uuid-ossp`** — `gen_random_uuid()` is built into Postgres 13+ now (`pgcrypto`); `uuid-ossp` is legacy but still common.

## Networking

Two modes:

| Mode | When | Trade-off |
|---|---|---|
| **Private access (VNet integration)** | Default for prod | Postgres accessible only inside the VNet; need bastion or VPN for psql |
| **Public access with firewall** | Dev / personal projects | Public endpoint; rely on firewall + Microsoft Entra ID auth |

For prod: private access, delegated subnet, private DNS zone (`privatelink.postgres.database.azure.com`). Connect from Container Apps / AKS via VNet integration.

Disable password auth in prod; use **Microsoft Entra ID authentication** with managed identity. Application connects with a token, not a stored password.

## Observability

Three layers, all needed:

1. **Query Performance Insight** (built-in) — slowest queries, wait events, query store. Enable Query Store: `pg_qs.query_capture_mode = ALL`, retention 7 days.
2. **`pg_stat_statements` to Grafana** — periodic snapshot of top-N queries by total_time, mean_time, calls. Catches slow queries before users notice.
3. **Diagnostic settings to Log Analytics** — enable `PostgreSQLLogs` and `PostgreSQLFlexSessions`. Required for SOC 2 audit log evidence; see `soc2-iso27001-controls-mapping`.

Required dashboards before go-live:
- Connections (active / idle / waiting) vs `max_connections`
- CPU and memory utilization
- Replication lag (if replicas)
- Top-10 slow queries by `mean_time`
- IOPS vs IOPS limit
- Storage used vs storage allocated (auto-grow can be enabled but plan the cap)

Alerts:
- Connections > 70% of max for 5 min
- CPU > 80% for 10 min
- Replication lag > 30s for 5 min
- Storage > 80%

## Anti-patterns

- **`pg_dump` for backup.** Use the managed backups; `pg_dump` is for ad-hoc export, not your DR strategy.
- **Storing files as `bytea`.** Use Azure Blob Storage with a URL in Postgres. The database is for relational data, not blobs.
- **`SELECT *` in production code.** Names the columns. The reasons are obvious in retrospect after a column rename breaks production.
- **No `pg_stat_statements`.** When something gets slow at 2am, the absence of query stats means you're guessing. Enable from day 1.
- **Hardcoded credentials in app config.** Use Managed Identity + Entra authentication. Connection strings in Key Vault as fallback only.
- **Long-running transactions in OLTP path.** Locks accumulate, vacuum gets blocked, performance degrades. Cap transaction time in the app (`SET LOCAL statement_timeout = '5s'`).
- **`OFFSET 100000` pagination.** Linear scan. Use keyset pagination (`WHERE id > $last_id ORDER BY id LIMIT 50`).

## Verification questions

1. Is Zone-Redundant HA enabled for prod, and has failover been tested in staging?
2. Is PgBouncer enabled in transaction-pool mode, and does the app connect on port 6432?
3. Is geo-redundant backup enabled and has PITR been tested at least once?
4. Is `pg_stat_statements` enabled and surfacing top-N queries to Grafana?
5. Are connections, CPU, IOPS, replication lag, and storage all on alerted dashboards?
6. Is Entra authentication used in prod with managed identity, not stored passwords?

## What to read next

- `engine-selection.md` — when Postgres is the wrong choice
- `patterns/connection-pool-sizing.md` — exact sizing math
- `data-migration-patterns.md` — zero-downtime schema changes
- `query-anti-patterns.md` — Postgres-specific slow-query catalog
- `azure-microservices-security` skill — Entra authentication, private endpoint, Key Vault
- `azure-microservices-cost-review` skill — tier cost shapes
