# Pattern — Partition Key Design (Cosmos DB)

## Problem

Cosmos DB partitions data across physical partitions by the partition key. The choice of partition key determines whether queries are partition-scoped (fast, cheap) or cross-partition (slow, expensive), whether RU consumption is balanced or concentrated on a hot partition, and whether the container scales to your traffic shape or chokes on it. The wrong key cannot be changed — re-partitioning is a copy operation. **Spend the hour up front; you cannot spend it later.**

## Use when

- Designing a new Cosmos container (any API: NoSQL, Mongo vCore, Cassandra)
- Sizing a Cosmos container expected to exceed 20GB or 10K RU/s
- Diagnosing 429 throttling errors when overall RU/s is under provisioned
- Reviewing a peer's Cosmos schema before it ships

## Avoid when

- Container is small (<20GB), low-traffic (<5K RU/s peak), and on serverless tier — Cosmos handles small containers fine even with a suboptimal key
- Data is genuinely flat (one key has one value) — but this is rare and worth questioning

## The three properties of a good partition key

A good partition key satisfies all three:

1. **High cardinality.** Many distinct values. The number of distinct keys should be at least 10–100× the number of physical partitions you expect.
   - Good: `/userId` (millions of users), `/orderId` (millions of orders), `/deviceId` (thousands+)
   - Bad: `/region` (a handful), `/status` (5–10 values), `/tenantTier` (3 values)

2. **Even traffic distribution.** No single key value should carry > 10% of traffic. Distribution should be roughly uniform across keys at any point in time.
   - Good: `/orderId` (each order touched a handful of times then mostly idle)
   - Bad: `/tenantId` when one tenant is 60% of traffic (hot partition)
   - Bad: `/date` (today's partition is hot, all others are cold)

3. **Query alignment.** The dominant query filter should include the partition key. If most queries don't filter by it, every query is a cross-partition fan-out.
   - Good: choose `/tenantId` if most queries are tenant-scoped
   - Bad: choose `/createdAt` if most queries are by user ID

If a candidate key fails any of the three, keep looking.

## Implementation steps

### Step 1 — list the top-5 to top-10 queries

Before considering any key, write down:

```
Q1: getOrderById(id) → filter: id=?
Q2: listOrdersByCustomer(customerId, statusFilter) → filter: customerId=?, status=?
Q3: listRecentOrders(tenantId, hours) → filter: tenantId=?, createdAt>?
Q4: updateOrderStatus(id, status) → filter: id=?
Q5: aggregateOrdersByDay(tenantId, date) → filter: tenantId=?, dateBucket=?
```

The partition key candidates fall out of this list.

### Step 2 — score candidates against the three properties

| Candidate | Cardinality | Distribution | Query alignment | Verdict |
|---|---|---|---|---|
| `/id` | Highest | Even | Only Q1 and Q4 use it | Bad — Q2, Q3, Q5 become cross-partition |
| `/customerId` | High | Mostly even | Q2 uses it; Q3, Q5 don't | Mixed |
| `/tenantId` | Medium | Skewed (big tenants) | Q3, Q5 align | Hot-partition risk |
| `/tenantId_yyyymm` (synthetic) | High | Even after monthly bucketing | Q3, Q5 align | Best for this case |

### Step 3 — when no natural key works, synthesize

Two synthesis patterns:

**Time-bucketed synthetic key**: `tenantId_yyyymm` or `tenantId_yyyymmdd`. Spreads writes for hot tenants across many partitions; bounds the partition size.
- Query: filter by `tenantId_yyyymm IN ('t1_202605', 't1_202604', ...)` for a date range.
- Trade-off: queries need to know the bucket; rolling time windows require multiple buckets.

**Hash-bucketed synthetic key**: `tenantId_hashbucket` where bucket = `hash(orderId) % 20`. Spreads writes for a hot tenant across 20 partitions deterministically.
- Query: filter by `tenantId_hashbucket IN ('t1_0', 't1_1', ..., 't1_19')` — query each bucket.
- Trade-off: every tenant-scoped query fans out across 20 partition values; more expensive than a single bucket, less than full cross-partition.

**Hierarchical partition keys** (preferred where available): `/tenantId`, `/userId`. Queries at the tenant level are partition-scoped to that tenant's sub-partitions; queries at the user level hit one physical partition. Avoids most synthesis pain.

### Step 4 — validate with realistic data

Before locking the key in:

1. Generate or import realistic data volumes (at least 1% of expected prod scale).
2. Run the top-5 queries; capture `x-ms-request-charge` per query.
3. Run a write workload at 2× expected peak; watch `Normalized RU Consumption (Max)` per partition. Should stay under 80%.
4. Identify the largest single partition value (the "fattest customer"). If that one value's traffic exceeds 10% of total, reconsider.

If validation fails, the key is wrong. **Easier to change here than in production.**

### Step 5 — document the choice

In the service's README or ADR:

```
Partition key: /tenantId_yyyymm
Rationale:
  - Top-5 queries align: Q3 and Q5 filter by tenantId+date bucket
  - Top tenant carries 40% of traffic; monthly bucketing distributes across 12 partitions/year
  - Validated with 3-month synthetic dataset at 2x peak load; max partition RU stayed under 70%
Migration path if wrong: change feed → new container → dual-write window → switchover
```

## Trade-offs

| Choice | Gain | Cost |
|---|---|---|
| Natural key (e.g., `/tenantId`) | Simple queries | Hot-partition risk on skewed tenants |
| Time-bucketed synthetic | Hot-tenant safe | Query complexity (date range = multiple buckets) |
| Hash-bucketed synthetic | Most even distribution | Every query is a small fan-out |
| Hierarchical key | Best of all worlds where available | Newer feature; verify SDK / driver support |

## Common failure modes

### Hot partition under skewed tenant load
**Detection**: 429 throttling when overall RU/s is under provisioned; `Normalized RU Consumption (Max)` near 100% on one partition while others are low.
**Fix**: switch to a synthetic key (time-bucket or hash-bucket), or move the hot tenant to its own dedicated container.

### Cross-partition query is the dominant query
**Detection**: most queries omit the partition key in the WHERE clause; `x-ms-request-charge` is unexpectedly high.
**Fix**: re-partition by the field that the dominant query filters by. If two dominant queries filter by different fields, consider a CQRS read model in a second container with a different key.

### Picked `/id` because it's unique
**Detection**: every query other than `getById` is cross-partition.
**Fix**: pick by query filter, not by uniqueness. `/id` is only correct if `getById` is 95%+ of traffic.

### No validation under load before launch
**Detection**: 429 errors only appear in production; staging traffic was too low to surface hot partitions.
**Fix**: load-test partition key choice against 2× expected peak before launch.

### Synthetic key never collapsed for queries
**Detection**: queries do single-bucket lookups by `tenantId_yyyymm` and miss prior months; users complain about missing history.
**Fix**: explicit multi-bucket queries (`WHERE pk IN (...)`) or hierarchical key for free roll-up.

## MCP tool opportunities

- **`recommend_partition_key`** — given a list of top-N queries and rough cardinality of each candidate field, return scored partition key candidates with rationale.
- **`detect_hot_partition_risk`** — given a sample of production query telemetry (key value frequencies), flag skewed keys with >10% concentration.
- **`generate_partition_key_validation_plan`** — output a test plan (data shape, query mix, RU thresholds) to validate a candidate key before launch.

## What to read next

- `../cosmos-db-design.md` — RU sizing, indexing, consistency
- `../data-migration-patterns.md` — Cosmos re-partition via change feed
- `../query-anti-patterns.md` — cross-partition query detection
- `connection-pool-sizing.md` — sibling pattern for Postgres connections
