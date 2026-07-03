# HA / DR for the Data Tier — Cross-Engine

## Purpose

High availability (HA) and disaster recovery (DR) for the data tier is where outages get expensive. Service code can be redeployed in minutes; lost data can be lost forever. This reference covers RPO/RTO target-setting, per-engine HA/DR capabilities, multi-region patterns, backup discipline, and the failover testing cadence — across Postgres, Azure SQL, Cosmos, Mongo, MySQL, and Redis.

## Start with RPO and RTO

Before designing HA/DR, define:

- **RPO (Recovery Point Objective)**: how much data loss is acceptable, in time. RPO 0 = no loss; RPO 5 minutes = up to 5 minutes of data loss tolerated.
- **RTO (Recovery Time Objective)**: how quickly service must resume after a disaster.

Most teams say "RPO 0, RTO seconds" without thinking. Then they discover the cost (multi-region synchronous replication, expensive licensing, dedicated DR drills) and back off.

Typical realistic targets:

| Service class | RPO | RTO |
|---|---|---|
| Critical OLTP (payment, identity) | 0 (sync) | < 5 min |
| Standard OLTP (most microservices) | < 1 min | < 30 min |
| Internal services / analytics | < 1 hour | < 4 hours |
| Best-effort / dev | < 24 hours | best effort |

Document the target per service. The data-tier design should justify how it meets the target.

## What each engine provides

### Postgres Flexible Server

| Capability | RPO | RTO | Cost |
|---|---|---|---|
| Zone-Redundant HA | ~0 (sync) | 60–120s | 2× |
| Read replica (same region) | seconds (async) | minutes to promote | per replica |
| Read replica (cross region) | seconds–minutes (async) | minutes to promote | per replica + egress |
| Geo-redundant backup | 5–15 min lag | hours to restore to a new server | inexpensive |

**Default for prod**: Zone-Redundant HA + geo-redundant backups. Cross-region read replica only if cross-region read latency or DR-with-tight-RPO is required.

DR pattern: zone-redundant HA covers AZ failure; geo-redundant backup covers region failure (with longer RTO and some RPO loss).

### Azure SQL

| Capability | RPO | RTO | Cost |
|---|---|---|---|
| Business Critical tier (zone-redundant) | 0 (sync) | < 30s | premium |
| General Purpose zone-redundant | ~0 | 60–120s | small premium |
| Failover Group (cross-region) | seconds (async geo-replication) | < 1 min after trigger | + secondary server |
| Active geo-replication | seconds (async) | seconds–minutes | + replica |
| PITR | 1–35 days configurable | hours | included |
| Long-Term Retention | months–years | hours | inexpensive |

**Default for prod**: Business Critical with zone-redundancy + Failover Group to a paired region. Failover groups support automatic failover (with grace period to avoid flapping) and read-only secondary endpoint for read-scale.

Business Critical's built-in AlwaysOn replicas give the lowest RTO; General Purpose has higher RTO and uses Azure Storage redundancy under the hood.

### Cosmos DB

| Capability | RPO | RTO | Cost |
|---|---|---|---|
| Multi-region with automatic failover | typically < 1s (Session/Bounded) | seconds | replication RU per region |
| Multi-region writes | active-active | seconds | 2× RU charged across regions |
| Continuous backup (PITR) | seconds | minutes (creates new account) | small premium |

Cosmos is the strongest of the engines for global HA/DR — multi-region is first-class. Set failover priorities; enable automatic failover:

```hcl
resource "azurerm_cosmosdb_account" "main" {
  geo_location {
    location          = "westeurope"
    failover_priority = 0
  }
  geo_location {
    location          = "northeurope"
    failover_priority = 1
  }
  enable_automatic_failover = true
}
```

Multi-region write trade-off: 2× RU cost, plus conflict resolution semantics. Don't enable unless a region-local write requirement is named.

### MongoDB

| Capability | RPO | RTO | Cost |
|---|---|---|---|
| Cosmos for Mongo vCore — zone redundancy | ~0 | < 1 min | included tier-dependent |
| Cosmos for Mongo vCore — geo replica | seconds (async) | minutes | + replica node |
| Atlas — replica set (cross-AZ / region) | seconds (async) | < 1 min | per node |
| Atlas — sharded cluster | seconds (async) | minutes | per shard |

Atlas backup: Cloud Backups with configurable retention and PITR. Cosmos for Mongo: continuous backup like Cosmos NoSQL.

### Azure Database for MySQL Flexible Server

Same shape as Postgres Flexible Server:

| Capability | RPO | RTO |
|---|---|---|
| Zone-Redundant HA | ~0 | 60–120s |
| Read replica (cross region) | seconds (async) | minutes to promote |
| Geo-redundant backup | 5–15 min lag | hours to restore |

### Azure Cache for Redis

| Capability | RPO | RTO |
|---|---|---|
| Standard tier (2-node primary/replica) | seconds (replication lag) | seconds (failover) |
| Premium tier with persistence (AOF) | seconds | seconds–minute |
| Premium passive geo-replication | seconds–minutes | minutes (manual failover) |
| Enterprise active geo-replication | seconds | seconds |

Reminder: Redis is a cache. RPO/RTO conversation is meaningful for **session-store / idempotency-key** workloads, less so for pure caches (which can be cold-started). For pure cache, RPO = "unlimited; we accept a full cache miss"; for session store, RPO and RTO matter.

## Multi-region patterns

### Pattern 1 — single-region with cross-region backup

```
Region A: primary database, zone-redundant HA, geo-redundant backups
Region B: backup storage; no live replica
```

RPO: backup-lag (5–60 min). RTO: hours (restore from backup to a new server).

Cheapest. Right for non-critical workloads or where region-failure is acceptable to recover slowly.

### Pattern 2 — active/passive cross-region

```
Region A: primary, write traffic
Region B: async replica, read-only
On disaster: promote Region B to primary
```

Postgres: cross-region read replica. Azure SQL: failover group with one writeable + one readable. Cosmos: single-write region with multi-region reads.

RPO: seconds (replication lag). RTO: < 5 min after failover trigger.

Right for most critical OLTP. Application must handle the "secondary endpoint" — read-only during normal ops, promotable on failover.

### Pattern 3 — active/active cross-region

```
Region A: writes accepted; replicates to B
Region B: writes accepted; replicates to A
Conflict resolution: LWW, custom, or CRDT
```

Cosmos multi-region writes. Atlas with multi-region clusters. Redis Enterprise active geo-replication.

RPO: ~0 (each region writes locally). RTO: seconds (no failover; clients route to remaining region).

Expensive (2× write cost on Cosmos). Complex (conflict semantics). Right only when region-local write latency is a named requirement.

### Pattern 4 — paxos / consensus-based sync replication

Synchronous multi-region replication. RPO=0 globally.

Not commonly used at this layer in Azure — write latency includes inter-region quorum (50–100ms per round-trip). The cost is high enough that most architectures pick active/active with eventual consistency instead, plus careful conflict handling.

## Backup discipline

Backups exist; **tested restores** exist. Untested backups are not backups.

Quarterly minimum:

1. **PITR restore test** — restore production to a scratch instance at a known timestamp. Run a smoke query. Delete.
2. **Cross-region restore** (if geo-redundant) — restore to the paired region. Validate connectivity from a test app. Delete.
3. **Document the runbook** — exact steps, who owns, expected duration. Update after each test.

Backup retention:

- **PITR**: 7–35 days (default 7; bump to 35 for prod). Cheap.
- **LTR (Long-Term Retention)**: weekly / monthly / yearly snapshots retained for months to years. Required for SOC 2 / regulatory.

For SOC 2 / ISO 27001 evidence, see `soc2-iso27001-controls-mapping` skill.

## Failover testing cadence

Quarterly minimum for every prod service:

1. **Trigger HA failover** in staging environment. Measure actual RTO. Verify app reconnects with appropriate retry behavior.
2. **Trigger cross-region failover** (failover group / Cosmos failover) for any service with cross-region DR claim. Measure RTO.
3. **Validate replica lag** during normal operation — if replica lag is consistently > target RPO, the DR claim isn't valid.

Annually: a real region-failover drill — point traffic to the DR region for some duration. Measure customer impact. Document what didn't work.

Don't trust documented RTO/RPO numbers that haven't been verified.

## DR runbook shape

Every prod service needs a written DR runbook covering:

- RPO / RTO targets
- Failover trigger (who decides, what criteria)
- Failover steps per engine (CLI commands, exact)
- Application reconfiguration (connection strings, DNS, secrets)
- Post-failover validation (smoke tests)
- Failback plan (when primary recovers)
- Who owns and when last tested

A runbook nobody has executed in production is half-complete. Plan a controlled drill.

## Common HA/DR anti-patterns

- **"We have backups" without ever restoring.** Not actually a recovery plan.
- **Same-region "HA" via two AKS nodes in the same AZ.** Doesn't protect against AZ failure. Use zone-redundant.
- **Read replicas counted as HA.** They're async; data loss possible on primary failure. They're for read-scale, not HA.
- **No quarterly failover test.** Documented RTO is a guess.
- **Application doesn't handle reconnect during failover.** Even if DB recovers in 60s, the app stays down because of dropped pool connections. Use retry libraries with exponential backoff.
- **Geo-redundant backup with no documented restore process.** When the region fails, nobody knows the steps.
- **DR drill skipped because "we're too busy."** First real disaster will cost more than the drill.
- **Failover group without grace period.** Transient blips trigger failover; flapping causes more damage than the original issue.

## Verification questions

1. Is the RPO/RTO target documented per service, and does the data-tier design match?
2. For each engine in use: is the HA mode zone-redundant (not same-zone, not disabled)?
3. Is geo-redundant backup enabled with retention matching compliance requirements?
4. Has a PITR restore been tested in the last quarter?
5. Has HA failover been triggered in the last quarter, with RTO measured?
6. Does the DR runbook exist, including failover trigger criteria and post-failover validation?
7. Is application connection retry tested against failover events?

## What to read next

- `postgres-on-azure.md` — Flexible Server HA mode detail
- `azure-sql-on-azure.md` — Business Critical, Failover Groups
- `cosmos-db-design.md` — multi-region configuration
- `mongodb-on-azure.md` — replica set / Atlas DR
- `redis-on-azure.md` — geo-replication for non-cache use
- `data-migration-patterns.md` — DR-restore is structurally similar to migration cutover
- `soc2-iso27001-controls-mapping` skill — backup retention and audit evidence
- `azure-microservices-observability` skill — replica-lag and failover-event dashboards
