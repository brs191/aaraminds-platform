# Cosmos DB Design

## Purpose

Design Cosmos DB containers that don't bankrupt you on RUs or trap you with a hot partition. This reference covers API choice, partition key design, RU sizing strategy, indexing policy, consistency levels, multi-region configuration, change feed, and TTL. Focus is the **NoSQL/SQL API**; Cosmos for Mongo and Cosmos for Postgres have their own references.

## API choice

Cosmos exposes the same engine through multiple APIs. Pick once — APIs are not interchangeable per container.

| API | When |
|---|---|
| **NoSQL / SQL** | Default for new work. JSON documents, SQL-like query, change feed. |
| **MongoDB (vCore or RU)** | Existing Mongo apps; need real Mongo aggregation. See `mongodb-on-azure.md`. |
| **Cassandra** | Existing Cassandra apps. Niche. |
| **Gremlin** | Property-graph workloads. See `graph-databases.md` for the engine decision — partitioned Gremlin is limited on deep traversals. |
| **Table** | Lift from Azure Table Storage. Niche. |

Default to **NoSQL/SQL API** unless an existing app forces another choice.

## The partition key is the most important decision

Partition key choice determines:
- Whether queries scale (or scan)
- Whether one tenant can starve all others
- Whether the bill grows linearly or quadratically with traffic

You cannot change a partition key after data is in the container. Re-partitioning is a copy operation. **Spend an hour on this before writing any code.**

Three properties of a good partition key:

1. **High cardinality** — many distinct values. `/userId` (millions) is good; `/region` (a handful) is bad.
2. **Even traffic distribution** — no single value should carry > 10% of traffic. A single dominant tenant violates this.
3. **Query alignment** — most queries should filter by the partition key. If queries don't filter by it, every query is cross-partition (expensive scan).

See `patterns/partition-key-design.md` for the decision walk.

### Synthetic / composite partition keys

When no natural key satisfies all three properties:

- **Hierarchical partition keys** (Cosmos preview/GA in 2024+): up to 3 levels, e.g., `/tenantId`, `/userId`, `/sessionId`. Queries at any level prefix are partition-scoped.
- **Composite synthetic key**: store and write `tenantId_yyyymm` or `tenantId_hashbucket` as a separate field, partition by it. Trade query flexibility for hot-partition avoidance.

If the natural key is fine, don't synthesize. Synthetic keys complicate every read.

## RU sizing

Cosmos charges Request Units (RUs). A typical point read of a 1KB document is ~1 RU; an insert is ~5 RUs; a cross-partition scan can be hundreds. Sizing strategies:

| Mode | When | Cost shape |
|---|---|---|
| **Provisioned (Manual)** | Steady traffic | Pay for reserved RU/s, 24/7 |
| **Autoscale** | Variable traffic with predictable peak | Pay max(10% of peak, actual). Cap headroom protects against runaway cost |
| **Serverless** | Dev, low traffic (<5K RU/s peak), <50GB | Pay per request; no minimum |

**Default for prod**: Autoscale at the container level. Set the peak to 2× current p95 RU consumption. Adjust after the first month.

```hcl
resource "azurerm_cosmosdb_sql_container" "orders" {
  # ...
  autoscale_settings {
    max_throughput = 4000
  }
}
```

### How to size RU/s

1. Measure actual RU cost per operation in dev with realistic data. Cosmos returns `x-ms-request-charge` on every response.
2. Multiply by peak ops/sec to get peak RU/s.
3. Add 30% headroom (autoscale will give you 10× headroom but bills 10% minimum).
4. Set autoscale `max_throughput` to that number, rounded up to the nearest 1000.

If RUs > 100K/s, you have either huge scale or a design problem. Suspect the design first.

### Cross-partition query cost

A query without the partition key in the `WHERE` clause is a **fan-out scan** — Cosmos hits every physical partition. RU cost is roughly (RU per partition × number of partitions). At scale this is the most expensive query you can write. Detect: any query without partition key filter is a red flag.

## Indexing policy

Cosmos indexes **every property by default**. This is convenient but wasteful — writes cost RUs per indexed path, and most properties never appear in `WHERE` clauses.

For containers with high write rate, customize the indexing policy:

```json
{
  "indexingMode": "consistent",
  "includedPaths": [
    { "path": "/customerId/?" },
    { "path": "/status/?" },
    { "path": "/createdAt/?" }
  ],
  "excludedPaths": [
    { "path": "/*" }
  ],
  "compositeIndexes": [
    [
      { "path": "/customerId", "order": "ascending" },
      { "path": "/createdAt", "order": "descending" }
    ]
  ]
}
```

Rules:

- Index only the paths you query on. Everything else excluded.
- Composite indexes for `ORDER BY` on multiple fields or for `WHERE x = ? ORDER BY y`.
- `indexingMode: "consistent"` (default) is right for almost everyone. `"none"` only for write-only stores (rare).

Trade-off: tighter indexing reduces write RU cost (30–50% common) but breaks any query whose property isn't indexed. Test the query suite after changing the policy.

## Consistency levels

Five levels, weakest-to-strongest:

| Level | What it guarantees | Read RU multiplier | Multi-region write |
|---|---|---|---|
| **Eventual** | Reads may see stale data; eventual convergence | 1× | Yes |
| **Consistent Prefix** | Reads see writes in order, no gaps; may see stale | 1× | Yes |
| **Session** (default) | Read-your-writes within a session token | 1× | Yes |
| **Bounded Staleness** | Lag bounded by K versions or T seconds | 2× | Yes |
| **Strong** | Linearizable reads | 2× | No (single-region writes only) |

**Default to Session.** This matches user expectation ("I just wrote X, I should see X") in 95% of cases. Pay for stronger only when business requirements demand it.

Strong consistency forces single-region writes — incompatible with multi-region active-active. Use Bounded Staleness if you need "close to strong" with multi-region.

## Multi-region

Cosmos supports multi-region replication with single-write or multi-write topology.

| Topology | When | Conflict resolution |
|---|---|---|
| **Single-write region, multi-region reads** | Most apps; simplest | None needed; one writer |
| **Multi-region writes** | Apps needing region-local writes (low latency globally) | Last-Write-Wins (default) or custom |

Multi-region writes adds cost (RUs charged per region) and conflict semantics complexity. Don't enable unless you have a named requirement for region-local writes.

Configure failover priorities:

```hcl
resource "azurerm_cosmosdb_account" "main" {
  # ...
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

Test failover quarterly. Manually fail over via Azure CLI in staging; measure application impact.

## Change feed

Every insert and update emits an event to the change feed (deletes do not, by default — enable with `change-feed-policy` if needed). Use for:

- CQRS read model population (see `microservices-data-architecture`)
- Search index sync (Azure AI Search)
- Audit log derivation
- Event-driven downstream services

Consumers: **Azure Functions Cosmos DB trigger** (default), or a custom processor using the Change Feed processor library in code.

Don't dual-write from the application. The change feed is the supported way to fan out from a Cosmos write.

## Backup and PITR

Two modes:

| Mode | Restore granularity | Cost |
|---|---|---|
| **Periodic** (default) | 2 backups per day, restore to latest | Free |
| **Continuous (PITR)** | Any second within last 7 or 30 days | ~$0.20/GB-month + restore cost |

Use **Continuous** for prod. Restore creates a new account — plan the cutover.

## TTL

Set TTL at the container or item level for naturally-expiring data (sessions, idempotency keys, soft-deleted items). TTL deletes consume RUs from the background pool, not the request pool — but it still costs.

```json
{ "id": "abc", "data": "...", "ttl": 3600 }
```

Don't use Cosmos as a cache. Use Redis. Cosmos TTL is for "delete this when it's stale" not "this is hot for 60 seconds."

## Anti-patterns

- **No partition key strategy** — picking `/id` because it's unique. Yes, but it makes every query cross-partition. Pick by the dominant query filter.
- **Cross-partition query at scale** — `SELECT * FROM c WHERE c.status = 'pending'` without partition key filter. Cost scales with partition count.
- **Default indexing on high-write containers** — writes pay RU per indexed path; most fields never queried. Customize the policy.
- **Strong consistency by default** — 2× read cost and forfeits multi-region writes. Use Session.
- **Cosmos as a cache** — RU cost dominates; Redis is 1–2 orders of magnitude cheaper for hot reads.
- **Documents > 100KB** — Cosmos charges per KB on RU; splitting large docs across multiple items is usually cheaper.
- **No `x-ms-request-charge` logging in dev** — you ship to prod blind on RU cost per operation.

## Verification questions

1. Is the partition key documented, validated for cardinality and distribution, and tested with realistic data?
2. Are autoscale RU/s set with a measured peak, not a guess?
3. Is the indexing policy customized for high-write containers (not default-indexed)?
4. Is the consistency level Session unless a stronger one has a written justification?
5. Is multi-region write enabled only if a named requirement exists, or are we paying for it by default?
6. Is `x-ms-request-charge` logged in dev / staging for every operation, surfacing RU cost per query?

## What to read next

- `engine-selection.md` — when Cosmos is the wrong choice
- `patterns/partition-key-design.md` — the partition-key decision walk
- `data-migration-patterns.md` — re-partitioning a live container
- `mongodb-on-azure.md` — when to choose Cosmos Mongo vCore over NoSQL API
- `query-anti-patterns.md` — Cosmos-specific anti-patterns
- `microservices-data-architecture` skill — CQRS, change-feed-driven projections
