# Azure MySQL on Azure — Operational Design

## Purpose

MySQL is not the pack's default — Postgres is the relational default. This reference exists for brownfield MySQL workloads on Azure (lift-and-shift, vendor apps that mandate MySQL, legacy mismatches with team Postgres expertise). The first question this reference helps answer is *not* "how do I tune MySQL" but **"should I migrate to Postgres or keep MySQL"** — and only after that decision settles, the operational guidance for keeping MySQL well.

## First — should you migrate to Postgres?

Migration to Postgres is worth scoping when:

- The MySQL workload is moderate (< 1TB, < 5K QPS) and not feature-load-bearing
- The team's existing systems are already on Postgres; consolidating reduces operational surface
- You need features Postgres has and MySQL doesn't: rich JSON operators, GIN/GIST indexes, pgvector, generated columns at the level Postgres supports, true `RETURNING` everywhere, real `FULL OUTER JOIN`, advanced extensions

Migration is *not* worth it when:

- MySQL workload is huge (multi-TB, sharded, heavy replication topology)
- Application is vendor-supplied and MySQL-only (Magento, WordPress, many SaaS-in-a-box)
- Team has deep MySQL expertise and limited Postgres exposure — operational risk dominates
- Migration cost (weeks-months for non-trivial workloads) exceeds the consolidation benefit

If the answer is "keep MySQL", read on. If "migrate to Postgres", see `data-migration-patterns.md` for the cross-engine pattern and `postgres-on-azure.md` for the target.

## Service — Flexible Server only

Azure Database for MySQL **Single Server** is retired. The only Azure-managed option is **Flexible Server**. Self-hosting MySQL on VMs is not in scope.

## Tier selection

Flexible Server tiers mirror Postgres Flexible Server:

| Tier | Use case |
|---|---|
| **Burstable** (B-series) | Dev, low-traffic prod, sidecar DBs |
| **General Purpose** (D-series) | Default for prod OLTP |
| **Memory Optimized** (E-series) | Memory-heavy workloads; large `innodb_buffer_pool_size` |

Sizing: start at General Purpose D2ds_v4 or D4ds_v4; resize online based on observed CPU/memory pressure. Storage IOPS scales with provisioned storage size on the GP tier; check the IOPS curve when sizing.

`max_connections` is configurable but bounded by tier RAM. Default ~256 on B-series, several thousand on E-series. Verify on current Azure docs; the cap is a function of `innodb_buffer_pool_size` and RAM allocation.

## High availability

Three modes (same shape as Postgres Flexible):

- **Disabled** — no HA; data loss on AZ failure
- **Same-Zone HA** — 2× cost; protects against node failure but not AZ failure
- **Zone-Redundant HA** — 2× cost; protects against AZ failure; **default for prod**

Failover RTO ~60–120 seconds. App must handle transient connection failure during failover.

## Connection management — ProxySQL or built-in pooling

MySQL connections are cheap relative to Postgres (lighter-weight backend threads) but still finite. For high-replica deployments, use a pooler.

Options:

| Tool | When |
|---|---|
| **ProxySQL** (sidecar) | Most flexible; read/write splitting, query routing, prepared-statement support |
| **MySQL Router** (sidecar) | Group replication topologies (InnoDB Cluster) |
| **Built-in HikariCP / pgxpool equivalents** | Application-side pool; works at moderate scale |

Application-side pool sizing rule of thumb (same as Postgres): `pool_size = (vCPU × 2) + 1`. See `patterns/connection-pool-sizing.md` for the math — most of it transfers.

Difference vs Postgres: MySQL's lighter-weight connections mean you can usually run at higher pool sizes before hitting `max_connections`. But the same blue/green deploy multiplier applies — verify peak (deploy time) connection demand.

## Replication

MySQL Flexible Server supports:

- **Read replicas** — up to 10; async replication
- **Cross-region read replicas** — for read latency / DR
- **Group replication / InnoDB Cluster** — not natively offered as a managed feature on Flexible Server; if you need multi-primary, evaluate self-hosting or alternative engines

Read replicas for HA is the wrong pattern (replicas are async; data loss possible on primary failure). Use Zone-Redundant HA for HA; replicas for read scale only.

Replication lag is the operational hazard. Monitor `Seconds_Behind_Master` (now `Seconds_Behind_Source` on newer versions); alert at > 30s sustained.

## Storage engine

**InnoDB only.** MyISAM is legacy, not recommended for any new work, and lacks transactions. If you find MyISAM tables in a workload you're inheriting, plan migration to InnoDB.

## Backups and PITR

- Automatic backups retained 1–35 days (configurable)
- PITR to any second in retention window
- Geo-redundant backups: `geo_redundant_backup_enabled` for prod

```hcl
resource "azurerm_mysql_flexible_server" "main" {
  # ...
  backup_retention_days        = 30
  geo_redundant_backup_enabled = true
}
```

PITR creates a new server. Plan the cutover. Test PITR quarterly.

## Authentication

- **MySQL native** auth (password) — works but stores credentials; SOC 2 finding waiting to happen
- **Microsoft Entra ID authentication** — supported on Flexible Server; use this for prod
- Managed Identity → application connects with a token, not a stored password

Disable password auth in prod where possible. Application connects via Entra; admin tools use Entra group membership.

## Networking

- **Private access (VNet integration)** — default for prod; private endpoint with private DNS zone (`privatelink.mysql.database.azure.com`)
- **Public access with firewall rules** — dev only

## Indexing

- **Primary keys**: clustered in InnoDB (table physically organized by PK). Bad PK choice = wide row footprint.
- **Secondary indexes**: contain PK pointers; very wide PKs (UUIDs without ordering) cause huge index bloat. Use ordered UUIDs (UUIDv7) or BIGINT AUTO_INCREMENT.
- **Composite indexes**: leftmost-prefix rule applies — index on `(a, b, c)` serves queries filtering on `a`, `a+b`, `a+b+c`, but not `b` alone.
- **Covering indexes**: include columns the query needs to avoid the row lookup.
- **`EXPLAIN`** is your diagnostic; learn to read it.

## Transactions and isolation

- Default isolation: **REPEATABLE READ** (different from Postgres / SQL Server defaults — surprises many)
- For OLTP, **READ COMMITTED** is usually more appropriate: less locking, fewer phantom-read surprises in app code that doesn't expect repeatable reads

```sql
SET GLOBAL transaction_isolation = 'READ-COMMITTED';
-- or per-session:
SET SESSION transaction_isolation = 'READ-COMMITTED';
```

Set this consistently at the server level for prod; otherwise different sessions get different isolation.

## Query Performance Insight

MySQL Flexible Server includes Query Performance Insight equivalent — slow query log + Performance Schema. Enable:

```hcl
resource "azurerm_mysql_flexible_server_configuration" "slow_query_log" {
  name      = "slow_query_log"
  server_id = azurerm_mysql_flexible_server.main.id
  value     = "ON"
}

resource "azurerm_mysql_flexible_server_configuration" "long_query_time" {
  name      = "long_query_time"
  server_id = azurerm_mysql_flexible_server.main.id
  value     = "1"  # seconds
}
```

Stream slow query log + Performance Schema metrics to Log Analytics. Build top-N slow queries dashboard in Grafana.

## Anti-patterns

- **UUID primary keys without ordered UUIDs.** Random insertion order destroys InnoDB clustering. Use UUIDv7 or BIGINT.
- **MyISAM in 2026.** Legacy. Migrate to InnoDB.
- **Default REPEATABLE READ surprising the application.** Verify the isolation level matches application expectations; switch to READ COMMITTED if needed.
- **`SELECT *` everywhere.** Same as Postgres; cost is real.
- **Implicit conversions.** `WHERE varchar_col = 123` (integer literal vs varchar column) causes full scan.
- **Replicas as HA.** Async; data loss on failover. Use Zone-Redundant HA.
- **Group commit disabled / overly small `innodb_log_file_size`.** Defaults are reasonable on Flexible Server; verify before tuning.
- **MySQL native auth in prod.** Use Entra.

## Verification questions

1. Has the migrate-to-Postgres decision been made and documented?
2. Is Zone-Redundant HA enabled for prod?
3. Is geo-redundant backup enabled and PITR tested?
4. Is slow query log enabled with `long_query_time = 1` and top-N surfaced to Grafana?
5. Is isolation level explicit (READ COMMITTED for OLTP, server-level)?
6. Is auth Entra-based, not native MySQL auth?
7. Are primary keys ordered (BIGINT auto-increment or UUIDv7), not random UUIDs?

## What to read next

- `engine-selection.md` — when Postgres is the better choice
- `postgres-on-azure.md` — the migration target if applicable
- `data-migration-patterns.md` — MySQL → Postgres cross-engine migration
- `query-execution-and-indexing.md` — MySQL EXPLAIN deep-dive
- `wait-stats-and-blocking.md` — Performance Schema diagnostics
- `transactions-and-isolation.md` — REPEATABLE READ vs READ COMMITTED behavior
- `patterns/connection-pool-sizing.md` — pool sizing math
