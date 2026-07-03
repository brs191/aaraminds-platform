---
name: azure-data-tier-design
description: Designs the operational data tier for Azure-hosted microservices — engine selection, schema and index design, query execution, transactions and isolation, partitioning, connection pooling, caching, sizing, HA/DR, and zero-downtime migration. Covers Postgres, Cosmos DB, MongoDB, plus Azure SQL, Redis, and graph engines (Neo4j, Cosmos Gremlin). Use when choosing an engine, designing a partition key, diagnosing slow queries or throttling, planning a migration, sizing pools, or designing HA/DR. Do not use for cross-service consistency (use microservices-data-architecture) or messaging-engine choice (use azure-service-mapping).
version: 1.2.1
last_updated: 2026-05-30
---

# Azure Data Tier Design

## When to use

Trigger this skill when the question is about a specific data engine on Azure, not about how services exchange state. Common triggers: "Postgres or Cosmos for this service," "what partition key for this container," "how do I migrate this column without downtime," "Cosmos RUs keep throttling — what's wrong," "we're hitting Postgres max_connections in production," "should we go Cosmos Mongo or Atlas," "Neo4j or Cosmos Gremlin for this graph workload."

Do **not** use this skill for: cross-service consistency or CQRS / outbox / saga (use `microservices-data-architecture`); messaging engine choice like Service Bus vs Event Hubs (use `azure-service-mapping`); broader cost review of the data tier (use `azure-microservices-cost-review`).

## The critical decision rule — engine choice is driven by access pattern, then consistency, then ops budget

In that order. Most fatal engine choices come from inverting it: someone picks Cosmos DB because "it scales" before knowing the access pattern, then discovers the queries need joins that Cosmos can't do, and bolts on a synonym table or a second engine to compensate. Or someone defaults to Postgres for everything and then fights connection limits, RU-style scaling, and global writes for the next 18 months.

Make the call in this sequence:

1. **Access pattern.** Relational with joins → Postgres. Document with bounded query shapes → Cosmos DB (SQL API). Document with the full MongoDB query surface → Cosmos for Mongo vCore or MongoDB Atlas. Key-value lookups at extreme scale → Cosmos DB. Time-series → Postgres with TimescaleDB extension or Azure Data Explorer (out of scope here). Graph traversal — variable-length paths, reachability, blast-radius, pattern match → a graph engine; see `references/graph-databases.md`.
2. **Consistency requirement.** Strong across regions → Cosmos with Strong (expensive) or single-region Postgres with read replicas. Read-your-writes within a session → Cosmos Session (default) or Postgres primary reads. Eventual is fine → any.
3. **Ops budget.** "We have one DBA-equivalent" → managed services only. "We need predictable cost, low ops" → Postgres Flexible Server or Cosmos serverless / autoscale. "We have a real ops team" → wider option set including Atlas, Hyperscale Citus.

Skipping step 1 is the most common failure mode in this pack's worked examples.

## Engine selection at a glance

| Workload | Default engine | Why | See |
|---|---|---|---|
| Relational, transactional, <10TB | Azure Database for PostgreSQL Flexible Server | Mature, joins, ACID, cheapest for the shape | `references/postgres-on-azure.md` |
| Relational, transactional, >10TB or sharded write | Azure Cosmos DB for PostgreSQL (Hyperscale Citus) | Distributed Postgres with shard-aware queries | `references/postgres-on-azure.md` |
| Relational, brownfield SQL Server lift-and-shift | Azure SQL Managed Instance | T-SQL surface, Service Broker, SQL Agent | `references/azure-sql-on-azure.md` |
| Relational, .NET-heavy team with SQL Server features in scope | Azure SQL Database | Query Store, RCSI, Hyperscale | `references/azure-sql-on-azure.md` |
| Relational, inherited MySQL workload | Azure Database for MySQL Flexible Server | Brownfield only; Postgres is the new-work default | `references/azure-mysql-on-azure.md` |
| Document, bounded queries, multi-region writes | Azure Cosmos DB (NoSQL/SQL API) | Native multi-region, RU model, change feed | `references/cosmos-db-design.md` |
| Document, full MongoDB query surface | Cosmos DB for MongoDB vCore **or** MongoDB Atlas on Azure | Mongo aggregation pipeline, real Mongo semantics | `references/mongodb-on-azure.md` |
| Key-value, hot reads, session / idempotency / rate-limit | Azure Cache for Redis | Sub-ms reads, eviction, persistence options | `references/redis-on-azure.md` |
| Read-side denormalized projection (CQRS read model) | Cosmos DB (NoSQL) | Cheap fan-out, JSON shape per query | `references/cosmos-db-design.md` + `microservices-data-architecture` |
| Analytical / BI / long time-range aggregation | Microsoft Fabric Warehouse, Synapse Serverless, ADX | Separate engine; OLTP source feeds via CDC | `references/analytical-engines.md` |
| Graph / connected data — variable-length traversal, blast-radius, pattern match | Neo4j (deep traversal) or Cosmos DB for Gremlin (shallow, on-stack) | Native property graph vs managed but partition-limited; the call is traversal depth | `references/graph-databases.md` |

Default to **Postgres Flexible Server** for new services unless the access pattern names one of the other rows. **Redis** is implicitly in the stack as a cache / session store / hot-path helper, not a primary store.

## The work — sequence for a new or evolving data tier

1. Name the access pattern (read:write ratio, query shapes, joins, multi-region need). Write it down before engine choice — `references/engine-selection.md`.
2. Pick the engine. If pattern is mixed, pick for the *write* shape; use a derived read model in a second engine if needed (CQRS, see `microservices-data-architecture`).
3. Define RPO/RTO and pick the HA/DR pattern that meets them — `references/ha-dr-data-tier.md`.
4. Engine-specific design: tier, HA mode, connection pooling, partition key (Cosmos), Query Store (Azure SQL), eviction (Redis). See engine reference + `references/patterns/connection-pool-sizing.md` / `references/patterns/partition-key-design.md` / `references/patterns/caching-patterns.md`.
5. Choose isolation level explicitly — `references/transactions-and-isolation.md`.
6. Plan migrations (expand/contract; dual-write is rarely safe) — `references/data-migration-patterns.md`.
7. Wire query-plan, wait-stats, and capacity dashboards **before** go-live — `references/query-execution-and-indexing.md`, `references/wait-stats-and-blocking.md`, `azure-microservices-observability`.

## Cross-cutting concerns

The data-tier surface has five engine-agnostic concerns. Each has its own reference:

- **Query execution and indexing** — plan capture, index types, composite ordering, statistics maintenance. `references/query-execution-and-indexing.md`
- **Wait stats, blocking, deadlocks** — why a query is slow when the plan is fine; lock contention; deadlock graphs. `references/wait-stats-and-blocking.md`
- **Transactions and isolation** — isolation level matrix per engine; lost update, write skew, phantom read; optimistic vs pessimistic. `references/transactions-and-isolation.md`
- **HA / DR** — RPO/RTO targets; failover groups; multi-region patterns; backup discipline; quarterly testing. `references/ha-dr-data-tier.md`
- **Caching patterns** — cache-aside, read-through, write-through, write-behind; stampede prevention. `references/patterns/caching-patterns.md`

Hit these *together* during data-tier design. Designing one in isolation produces a system that's fast in the happy path and broken at the edges.

## Worked example — brownfield: order service hitting Postgres connection limit on Container Apps

Setup: Spring Boot order service on Azure Container Apps, 20 active revisions across blue/green plus canary, each with a HikariCP pool size of 30. Postgres Flexible Server General Purpose D4s (max_connections = 859 at this tier). Production now intermittently rejects new connections; replicas are starved when traffic spikes.

Decision walk:

1. **Diagnose, don't resize.** 20 revisions × 30 = 600 connections steady-state, but blue/green during a deploy briefly doubles that to 1200, exceeding 859. The symptom is connection refusals; the cause is pool sizing × replica count, not Postgres tier.
2. **Fix the architecture, not the symptom.** Don't raise tier just to buy headroom. Add PgBouncer (built into Postgres Flexible Server — `pgBouncer.enabled = true`, transaction pooling mode). Application connects to PgBouncer; PgBouncer maintains a small server pool against Postgres. App-side HikariCP can stay at 30; PgBouncer collapses these into far fewer backend connections. See `references/patterns/connection-pool-sizing.md`.
3. **Cap revisions.** Container Apps' default revision retention is unbounded by count. Set `revisionSuffix` strategy + `properties.configuration.maxInactiveRevisions: 3`. Old revisions hold no inbound traffic but their pods stay warm and connections open for the grace window.
4. **Validate.** Add `pg_stat_activity` dashboard to Grafana; alert if active connections > 70% of max_connections sustained for 5 minutes. See `azure-microservices-observability`.
5. **Migration shape.** PgBouncer enable is a server-side flag change — zero downtime, takes effect within a minute. Revision retention is a Container Apps config change — zero impact. No application redeploy required.

The wrong answer here is to upgrade Postgres to Memory Optimized E8s for the higher max_connections. That's roughly 2.5× the cost and treats the symptom.

## Anti-pattern — over-provisioning Cosmos RUs to mask a hot partition key

**Bad:** A Cosmos container partitioned by `/tenantId` is throttling at peak hours. Engineering response: raise the RU/s budget from 10,000 to 50,000. Throttling subsides. Bill jumps 5×.

**Why it fails:** Cosmos distributes RUs evenly across physical partitions. If one tenant is 60% of traffic, that tenant's physical partition can use at most its share — roughly 1/N of the budget. Adding RUs adds capacity to *cold* partitions while the hot one still throttles. The bill grows linearly; the throttling does not stop.

**Detection signal:** `Normalized RU Consumption` per partition in Azure Monitor shows one partition near 100% while others sit under 20%. Or: `429 Too Many Requests` continues after raising RUs.

**Fix:** Re-partition. Either choose a higher-cardinality key (`/tenantId-yyyymm` if traffic is time-bounded; `/orderId` if reads are by order), or split high-volume tenants into a dedicated container. Re-partitioning Cosmos requires a copy operation — use the Change Feed + a second container with the new key, dual-read, cut over. See `references/data-migration-patterns.md` and `references/patterns/partition-key-design.md`.

## Verification questions

1. Has the access pattern (query shapes + read:write ratio + multi-region need) been written down before the engine was named?
2. Are RPO and RTO targets documented, and does the HA/DR design match?
3. For Postgres: is PgBouncer enabled, and is `(replicas × pool_size)` under `max_connections × 0.7` even during blue/green?
4. For Azure SQL: is Query Store on, and is RCSI enabled?
5. For Cosmos: does the partition key have high cardinality, even traffic distribution, and alignment with the most common query filter?
6. For Redis: is `maxmemory-policy` set explicitly (not the default `noeviction`)?
7. For any new schema: is there a documented expand/contract migration plan, or are you relying on downtime windows?
8. Are slow-query, wait-stats, and capacity dashboards in Grafana **before** the service ships, not after the first incident?
9. Has failover (HA and cross-region where claimed) been tested in the last quarter?
10. If choosing Cosmos: have you confirmed the query shape doesn't need joins or full Mongo aggregation that the SQL API can't do?

## What to read next

Engines: `references/engine-selection.md` · `references/postgres-on-azure.md` · `references/azure-sql-on-azure.md` · `references/azure-mysql-on-azure.md` · `references/cosmos-db-design.md` · `references/mongodb-on-azure.md` · `references/redis-on-azure.md` · `references/analytical-engines.md` · `references/graph-databases.md`

Cross-cutting: `references/schema-design.md` · `references/partitioning.md` · `references/query-execution-and-indexing.md` · `references/wait-stats-and-blocking.md` · `references/transactions-and-isolation.md` · `references/ha-dr-data-tier.md` · `references/data-migration-patterns.md` · `references/query-anti-patterns.md`

Pattern cards: `references/patterns/partition-key-design.md` · `references/patterns/connection-pool-sizing.md` · `references/patterns/caching-patterns.md`

Related skills: `microservices-data-architecture` (cross-service CQRS / saga / outbox above this skill) · `azure-microservices-cost-review` (engine cost shape) · `azure-microservices-observability` (dashboards) · `azure-microservices-security` (Entra, Private Link, Key Vault)
