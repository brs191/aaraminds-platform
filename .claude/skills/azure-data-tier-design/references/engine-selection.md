# Engine Selection — Postgres, Cosmos DB, MongoDB on Azure

## Purpose

Pick the right engine before code is written. Engine choice is the single hardest decision to reverse — a schema migration is days, an engine swap is months. This reference walks the decision in the order the SKILL.md prescribes: access pattern, then consistency, then ops budget.

## The decision sequence

### Step 1 — name the access pattern

Before opening the Azure portal, answer these on paper:

1. **Read:write ratio.** Read-heavy (>10:1) tolerates eventual consistency on a read replica or projection. Write-heavy or balanced needs primary-direct queries.
2. **Query shapes.** Single-row by key, range scans by indexed column, multi-table joins, aggregations across millions of rows, full-text, vector similarity, geo. Enumerate the top 5–10 actual queries.
3. **Cardinality of partitioning attribute.** If most queries filter on `tenantId` and one tenant is 50% of traffic, you have a hot-partition problem in every horizontally-partitioned engine.
4. **Multi-region need.** Single region, read replicas in N regions, or multi-region writes. Each step up costs an order of magnitude more.
5. **Data volume now and in 18 months.** <100GB is irrelevant for engine choice. >10TB constrains the option set (Postgres single-instance gets painful, Cosmos and Hyperscale Citus open up).
6. **Schema flexibility.** Tight schema with foreign keys (Postgres) vs evolving document shape (Cosmos / Mongo) vs append-only event log (consider event sourcing, see `microservices-data-architecture`).

### Step 2 — match access pattern to engine

```
Relational, joins, transactional (banking, orders, inventory)
  └─ Postgres Flexible Server (default for new work)
      └─ if data > 10TB or write-sharded:
      │   └─ Cosmos for PostgreSQL (Hyperscale Citus)
      └─ if brownfield SQL Server lift:
      │   └─ Azure SQL Managed Instance
      └─ if .NET-heavy and SQL Server features in scope:
      │   └─ Azure SQL Database
      └─ if inherited MySQL workload:
          └─ Azure Database for MySQL Flexible Server (consider migration to Postgres)

Document, bounded queries, multi-region writes (catalog, user profile, settings)
  └─ Cosmos DB (NoSQL/SQL API)
      └─ if requires real Mongo aggregation pipeline:
          └─ Cosmos for MongoDB vCore  OR  MongoDB Atlas

Cache, session store, idempotency keys, rate limiter
  └─ Azure Cache for Redis (Premium tier for prod)

Key-value at extreme QPS (durable, not cache)
  └─ Cosmos DB (NoSQL)  +  optional Redis on hot path

Time-series (metrics, logs, audit)
  └─ Azure Data Explorer / Fabric KQL Database  (see analytical-engines.md)
  └─ or Postgres + TimescaleDB extension if small scale

Analytical / BI / long time-range aggregation
  └─ Microsoft Fabric Warehouse / Lakehouse  (see analytical-engines.md)
  └─ Synapse Serverless SQL for ad-hoc
  └─ Do NOT do analytics in the OLTP engine

Vector search / RAG embeddings
  └─ Postgres + pgvector extension (default, cheap)
  └─ or Azure AI Search if the dataset is huge and search-first
```

### Step 3 — consistency requirement filter

| Need | Postgres | Cosmos (SQL) | Cosmos Mongo | Atlas |
|---|---|---|---|---|
| Strong within region | Yes (primary) | Yes (Strong level) | Yes (Strong) | Yes |
| Strong across regions | No (single primary) | Yes (Strong, expensive) | Yes (Strong, expensive) | Yes |
| Read-your-writes (session) | Yes (primary reads) | Yes (Session — default) | Yes | Yes |
| Bounded staleness across regions | Read replicas with lag SLO | Yes (Bounded Staleness) | Yes | Yes |
| Eventual | Async replicas | Yes (Eventual — cheapest) | Yes | Yes |

If the requirement is "strong consistency across regions for writes," the *only* native answer is Cosmos at the Strong consistency level. The cost is multi-region RU charges and write latency that includes inter-region quorum. Verify the business actually needs this before paying for it.

### Step 4 — ops budget filter

| Budget | Choices |
|---|---|
| Solo / minimal ops, predictable cost | Postgres Flexible Server (Burstable or General Purpose) — known knobs, low surprise |
| Solo / minimal ops, traffic spikes | Cosmos DB serverless or autoscale — pay for what you use, capped headroom |
| Small team, multi-region | Cosmos DB autoscale, single-region Postgres + read replicas |
| Full ops team, want Mongo semantics | MongoDB Atlas dedicated cluster on Azure with VNet peering |
| Want Postgres but data > 10TB | Cosmos for PostgreSQL (Hyperscale Citus) — managed sharded Postgres |

## Engine deep-dives

### Azure Database for PostgreSQL Flexible Server

**Pick when:**
- Relational schema with foreign keys, joins across tables
- Strong consistency single-region; read replicas can carry eventual reads
- Data volume < 10TB
- Need extensions: `pgvector`, `postgis`, `pg_partman`, `pg_cron`, `pg_stat_statements`

**Skip when:**
- Need multi-region writes (Postgres is single-primary)
- Schema is genuinely document-shaped (don't fake it with JSONB on every column)
- Need extreme QPS on a single key (use Redis or Cosmos)

**Decision pitfall**: defaulting to "Postgres because it's familiar" when the access pattern is multi-region document writes. The retrofit is multi-month.

### Azure Cosmos DB (NoSQL / SQL API)

**Pick when:**
- Document model with predictable query shapes (always filter by partition key + ≤2 secondary attributes)
- Multi-region writes needed
- Need horizontal scale to >10TB with predictable latency
- Need change feed for derived projections (CQRS read model, search index, audit log)

**Skip when:**
- Queries require joins across containers (Cosmos has no joins between containers)
- Need full SQL aggregation surface (group by, window functions over large sets)
- Team has zero Cosmos experience and the project is < 6 months — the learning curve eats the schedule

**Decision pitfall**: picking Cosmos for "scale" without naming the partition key and the top-3 queries. See `patterns/partition-key-design.md`.

### Azure Cosmos DB for MongoDB (vCore tier)

**Pick when:**
- Existing MongoDB application being lifted to Azure
- Need the full MongoDB aggregation pipeline, transactions, change streams
- Want a managed Azure-native option without taking on Atlas as a separate vendor

**Skip when:**
- The application uses MongoDB features only on the latest server version (vCore tracks but lags by 1–2 minor versions; verify before committing)
- You don't actually need Mongo semantics — pick Cosmos NoSQL/SQL API instead, it's cheaper per RU

**Note on RU-based Cosmos Mongo (vs vCore)**: the RU-based tier of Cosmos for Mongo is the older option — pay-per-RU, more limits on Mongo features, harder to predict cost. **Default to vCore for new work.** Use RU-based only if you've validated cost and feature support.

### MongoDB Atlas on Azure

**Pick when:**
- Multi-cloud requirement (also need AWS or GCP regions)
- Need the absolute latest MongoDB version and feature surface (Atlas Search, Atlas Vector Search, etc.)
- Team has deep Atlas expertise already
- Willing to manage the vendor relationship separately from Azure

**Skip when:**
- The pack is Azure-only (default — you carry an extra vendor for no gain)
- Budget is a constraint (Atlas dedicated clusters are not cheap)

**Decision pitfall**: choosing Atlas "because Mongo" when Cosmos for MongoDB vCore would meet the actual feature need. Audit which Mongo features the app actually uses before committing to Atlas.

### Azure Cosmos DB for PostgreSQL (Hyperscale Citus)

**Pick when:**
- Need sharded Postgres — single-instance Postgres can no longer hold the data or the write rate
- Existing Postgres schema with a clear shard column
- Want to keep Postgres tooling (psql, pg_dump, ORMs) while scaling horizontally

**Skip when:**
- Data is < 1TB and you don't have a write-rate problem — Flexible Server is simpler and cheaper
- Schema has no natural shard column

### Azure SQL Database / Managed Instance (brownfield-leaning)

**Pick when:**
- Lift-and-shift from on-prem SQL Server (Managed Instance)
- .NET-heavy team with deep T-SQL / SSMS expertise; Entity Framework Core in production code
- Workload uses SQL Server-specific features without clean Postgres equivalents (Service Broker, In-Memory OLTP / Hekaton, Always Encrypted with secure enclaves)

**Skip when:**
- Greenfield with no SQL Server legacy — Postgres is the pack default

**Notable strength**: Query Store. Plan history with regression detection and plan forcing. The Azure SQL team's killer feature; underused by most. See `azure-sql-on-azure.md`.

### Azure Database for MySQL Flexible Server (brownfield only)

**Pick when:**
- Existing MySQL workload (vendor app, legacy lift)
- Migration to Postgres is not justified by the scope (large dataset, vendor mandate, team expertise gap)

**Skip when:**
- Greenfield — Postgres is the pack default, no exception
- Workload is moderate and team is on Postgres elsewhere — migrate

See `azure-mysql-on-azure.md` — the *first* question that reference asks is whether to migrate.

### Azure Cache for Redis (implicit stack member)

**Pick for:**
- Cache in front of any primary store (cache-aside is the default pattern)
- Session storage with TTL
- Idempotency keys, rate limiters, distributed counters
- Sorted-set leaderboards
- Low-stakes pub/sub (not as primary message bus — use Service Bus)

**Don't use as:**
- System of record — Redis is a cache; even with persistence enabled, treat as flushable
- Primary database — Cosmos / Postgres are durable

Default tier for prod: Premium. See `redis-on-azure.md` and `patterns/caching-patterns.md`.

### Analytical engines (Fabric, Synapse, ADX)

**Pick when:**
- Reporting / BI queries are degrading OLTP performance
- Time-range aggregations span months / years
- Power BI or Tableau connects directly to your transactional database

**Don't use for:**
- OLTP — analytical engines have high query startup; not designed for thousands of small reads/sec
- "Future-proofing" — stand up when the demand is real, not theoretical

The default analytical pattern: OLTP source → CDC → Lakehouse (OneLake / ADLS) → Fabric Warehouse / Synapse Serverless / ADX. See `analytical-engines.md`.

## Common selection mistakes

- **"Cosmos because it scales"** — without a partition key plan, this scales the bill, not the throughput.
- **"Postgres because we know it"** — fine, but multi-region writes will haunt you later. Decide once.
- **"Mongo because the team likes it"** — verify which Mongo features are load-bearing; if the answer is "we just use it like a document store with find/insert/update", Cosmos NoSQL is cheaper and Azure-native.
- **"We'll use JSONB in Postgres for everything flexible"** — works until query shapes proliferate; index management on JSONB columns gets expensive fast.
- **"Two engines for redundancy"** — multi-engine deployments roughly double operational complexity. Justify it with a named CQRS / projection rationale, not "in case one fails."

## Verification questions

1. Is the top-5 query list written down before the engine is named?
2. Has the partition / shard / index strategy been sketched at engine-choice time, not after the schema is live?
3. Has the multi-region requirement been confirmed (cost: 2–5× single region) or ruled out?
4. If choosing Cosmos: has the team done a Cosmos project before, or is this a first-time learning cost?
5. If choosing Atlas: what specifically requires Atlas over Cosmos for Mongo vCore?

## What to read next

- `postgres-on-azure.md` — Flexible Server operational detail
- `cosmos-db-design.md` — partition key, RU sizing, consistency levels
- `mongodb-on-azure.md` — Cosmos Mongo vCore vs Atlas trade-offs
- `data-migration-patterns.md` — once you've chosen, how to get there
- `microservices-data-architecture` skill — CQRS shape that may require *two* engines, not one
