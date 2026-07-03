# Redis on Azure — Cache, Session Store, Hot-Path Data

## Purpose

Redis is implicitly in the pack stack — every non-trivial microservices system uses it for caching, session storage, idempotency keys, rate limiting, or hot reads. This reference promotes Redis to first-class: when to use it, tier selection, persistence options, cluster mode, eviction, geo-replication, and the caching patterns that decide whether Redis helps or just shifts the failure mode. Focus is **Azure Cache for Redis**; self-hosting on VMs is out of scope.

## When to use Redis

- **Cache** in front of Postgres / Cosmos / Mongo / Azure SQL — read-through, cache-aside, write-through patterns
- **Session store** — short-lived per-user state with TTL
- **Idempotency keys** — "have we processed this request before?" — set-NX with TTL
- **Rate limiting** — counters with TTL, sliding window via sorted sets
- **Distributed lock** — `SET key value NX PX 30000` (with Redlock for multi-node strictness — but think twice; distributed locks are hard to get right)
- **Leaderboards / sorted indexes** — sorted sets (`ZADD`, `ZRANGE`)
- **Pub/Sub for low-stakes signaling** (not as a primary message bus — use Service Bus)

When **not** to use Redis:

- Durable system of record — Redis is a cache, not a database. Even with persistence enabled, treat it as flushable.
- Large objects (>100KB) — they hog memory; serialize differently or store the blob elsewhere with a Redis pointer.
- Workflows needing transactional consistency with the primary DB — caching doesn't fit; redesign with outbox + read model (see `microservices-data-architecture`).

## Service tiers

| Tier | Use case | Notable features |
|---|---|---|
| **Basic** | Dev only | Single node, no SLA, no HA |
| **Standard** | Light prod, small dev | Two-node primary/replica, 99.9 SLA |
| **Premium** | Default prod | Clustering, persistence (AOF/RDB), VNet, geo-replication, 99.9 SLA |
| **Enterprise** | High throughput, low latency, RediSearch / RedisJSON / RedisTimeSeries modules | Redis Enterprise on Azure; 99.999 SLA available |
| **Enterprise Flash** | Very large datasets (>200GB) at lower cost | Combines RAM + flash for hot+cold data |

Default for prod: **Premium**. Pick Enterprise only when you need Redis Modules (RediSearch, RedisJSON) or extreme SLA. Don't use Basic / Standard in prod.

## Persistence — AOF vs RDB vs both vs neither

Persistence options on Premium / Enterprise:

| Mode | Behavior | Recovery |
|---|---|---|
| **None** | Pure cache; Redis-restart loses everything | Empty cache; expect cold-start latency |
| **RDB** (snapshot) | Periodic dump (e.g., every hour) | Restore to last snapshot; up to 1hr data loss |
| **AOF** (append-only file) | Every write logged to file | Restore to seconds before crash; slower writes |
| **Both** | RDB for fast restart + AOF for fewer lost writes | Use both when data loss matters |

For a **pure cache**: persistence none. Redis restart = cache miss storm — handle that in app design (warm-up, gradual rollout). Don't pay for persistence on a cache.

For a **session store**: AOF or both. Users get logged out on Redis restart otherwise.

For an **idempotency / dedupe key store**: at least RDB. Persistence loss means a "we already processed this" key is gone — the request gets reprocessed.

## Cluster mode

Premium tier supports **clustering** — data partitioned across shards (10 shards default max, configurable).

When to enable:
- Dataset > 30GB (a single Premium node tops out around there)
- Need higher throughput (>100K ops/sec)
- Want horizontal scale, not vertical

Cost: clustering changes the client connection model. Redis client must be **cluster-aware** (`StackExchange.Redis` with `endpoint:port,abortConnect=false` etc.; Lettuce in Java with `RedisClusterClient`). Single-key commands work; multi-key commands require all keys hash to the same slot — use **hash tags** like `{user:123}:profile` and `{user:123}:settings` to co-locate.

Don't enable clustering "just in case." If you don't need it, you pay the cognitive cost (multi-key gotchas, slot rebalancing) for nothing.

## Eviction policy

When Redis hits memory limit, what happens?

| Policy | Behavior | When |
|---|---|---|
| `noeviction` (default) | Writes fail with OOM error | **Wrong default for cache.** Causes outages. |
| `allkeys-lru` | Evict least recently used across all keys | Default for general caching |
| `allkeys-lfu` | Evict least frequently used | Skewed access patterns (a few keys very hot) |
| `volatile-lru` | Evict LRU among keys with TTL | When some keys are "permanent" |
| `volatile-ttl` | Evict shortest-TTL first | Session-like data |
| `allkeys-random` | Random eviction | Rarely useful |

**Set `maxmemory-policy` explicitly.** Don't leave it at `noeviction`:

```bash
az redis update --name redis-prod --resource-group rg-prod \
  --set redisConfiguration.maxmemory-policy=allkeys-lru
```

Or via Terraform:

```hcl
resource "azurerm_redis_cache" "main" {
  # ...
  redis_configuration {
    maxmemory_policy = "allkeys-lru"
  }
}
```

## Geo-replication

Available on Premium (passive) and Enterprise (active-active).

| Mode | When |
|---|---|
| **Passive geo-replication** (Premium) | DR — secondary is read-only; manual failover | Cross-region disaster recovery |
| **Active geo-replication** (Enterprise) | Multi-region writes with CRDTs | Global low-latency apps |

Passive is the common case. Active geo-replication is expensive and the conflict-resolution semantics (CRDTs per data type) require careful design — don't enable unless multi-region writes are a named requirement.

## Networking

- Premium / Enterprise: **VNet injection** or **private endpoint** — keep Redis off the public internet
- Standard: only public endpoint with firewall + access keys; not appropriate for prod with sensitive data
- TLS: enable `enableNonSslPort = false`; force TLS 1.2 minimum, prefer 1.3
- Auth: Entra ID auth supported on Enterprise (Premium has access keys + Entra in preview / GA — verify)

```hcl
resource "azurerm_redis_cache" "main" {
  name                = "redis-orders-prod"
  resource_group_name = azurerm_resource_group.main.name
  location            = "westeurope"
  capacity            = 1
  family              = "P"
  sku_name            = "Premium"
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"

  redis_configuration {
    maxmemory_policy           = "allkeys-lru"
    rdb_backup_enabled         = true
    rdb_backup_frequency       = 60
    rdb_storage_connection_string = azurerm_storage_account.redis_backup.primary_blob_connection_string
  }

  subnet_id = azurerm_subnet.redis.id
}
```

## Common caching patterns

See the pattern card `patterns/caching-patterns.md` for the depth on cache-aside, read-through, write-through, write-behind, and stampede prevention. The short version:

- **Cache-aside** (default) — app reads from cache, falls through to DB on miss, populates cache. Simple, but stale on writes.
- **Read-through** — cache library does the fall-through. Same characteristics; nicer code.
- **Write-through** — app writes to cache and DB synchronously. Slower writes; cache always fresh.
- **Write-behind** — app writes to cache; cache flushes to DB async. Fast writes; cache is source of truth temporarily; risk of data loss.

Pick cache-aside by default. Other patterns when the trade-off is named and accepted.

## Observability

- Azure Monitor metrics: connected clients, used memory %, evicted keys, server load, network bandwidth
- Slow log via `SLOWLOG GET` (configure `slowlog-log-slower-than` and `slowlog-max-len`)
- Diagnostic settings → Log Analytics for SOC 2 audit log requirements

Required dashboards:
- Memory used vs maxmemory (alert at 80%)
- Evicted keys per second (alert if non-zero on a "should never evict" cache, e.g., session store with adequate sizing)
- Connected clients (alert at 80% of `maxclients`)
- Server load (alert at 80%)
- Cache hit rate (alert if < 80% sustained — indicates cache is undersized or invalidation is too aggressive)

## Anti-patterns

- **Default `noeviction` policy.** Writes start failing with OOM error; outage. Always set an explicit policy.
- **Persistence on a pure cache.** Wasted cost; cache should be flushable.
- **Treating Redis as durable.** Even with AOF, treat it as best-effort. System of record stays in Postgres / Cosmos / etc.
- **Large objects in Redis.** Serialize differently, split keys, or store the blob elsewhere with a pointer.
- **No connection pooling.** Each app instance opens hundreds of connections; clients reuse instead. `StackExchange.Redis` and modern clients pool by default — verify.
- **Synchronous Redis call in critical path with no timeout.** Redis stall = app stall. Set client timeout (50–200ms typical).
- **`KEYS *` in production.** Blocks the server. Use `SCAN` for iteration.
- **Cache stampede.** Cache miss causes N concurrent requests to hit the DB. See `patterns/caching-patterns.md` for prevention.
- **Distributed lock as primary primitive.** Distributed locks are hard. Most "I need a lock" cases are better solved with optimistic locking, idempotency keys, or DB-level concurrency primitives.

## Verification questions

1. Is the tier Premium or higher for prod (not Basic / Standard)?
2. Is `maxmemory-policy` explicitly set (not `noeviction`)?
3. Is TLS forced and the non-SSL port disabled?
4. Is the cache on a private network (VNet injection or private endpoint)?
5. Is persistence configuration aligned with the use case (none for pure cache, AOF for session/idempotency)?
6. Are cache hit rate, eviction rate, and memory % on alerted dashboards?
7. Is the caching pattern named (cache-aside, etc.) and stampede prevention considered?

## What to read next

- `patterns/caching-patterns.md` — cache-aside, read-through, write-behind, stampede prevention
- `engine-selection.md` — Redis as cache vs Cosmos / Postgres as primary
- `ha-dr-data-tier.md` — Redis geo-replication and DR positioning
- `azure-microservices-observability` skill — cache dashboards
- `azure-microservices-cost-review` skill — Premium vs Enterprise tier cost
