# Pattern — Caching Patterns

## Problem

A naive cache helps for the first 95% of requests, then makes the worst 5% catastrophic — a cache miss storm during a Redis flush, stale data after writes, dual-write divergence between cache and database, or a cache stampede where one expiring key triggers thousands of concurrent regenerations. The choice of caching pattern (cache-aside, read-through, write-through, write-behind) and the stampede-prevention strategy determine whether the cache helps or just shifts the failure mode.

## Use when

- Adding Redis (or any cache) in front of Postgres / Cosmos / Mongo / Azure SQL
- Designing a session store, idempotency-key store, or rate-limiter
- Reviewing existing cache code for failure modes that haven't bitten yet
- Diagnosing a cache-related incident (stale data, miss storm, divergence)

## Avoid when

- Workload is read-light or already fast (< 10ms p99 from the DB) — cache adds complexity for marginal benefit
- Data freshness requirement is "every read must reflect last write" — cache invalidation is hard; redesign instead
- The cache is being used to hide a fundamentally bad query — fix the query first

## The four patterns

### Cache-aside (default)

```
read(key):
  if cache has key: return cache[key]
  value = db.read(key)
  cache.set(key, value, ttl)
  return value

write(key, value):
  db.write(key, value)
  cache.invalidate(key)   # or cache.set(key, value)
```

The application orchestrates. Simple, explicit, the right default.

**Trade-off**: stale window between DB write and cache invalidation (especially if invalidation happens before the DB write commits — wrong order).

### Read-through

```
read(key):
  return cacheLibrary.get(key)   # library does fall-through to DB internally
```

The cache library handles fall-through. Same semantics as cache-aside; nicer code; locks the application to the library's behavior.

Available in libraries like Spring Cache abstraction, Caffeine, Hibernate L2 cache, .NET MemoryCache, etc.

### Write-through

```
write(key, value):
  cache.set(key, value)
  db.write(key, value)
  # if either fails: app handles transactional concerns
```

App writes to both, synchronously. Cache is never stale (assuming both writes are atomic, which they aren't — see anti-patterns).

**Trade-off**: slower writes (both paths in the critical path); no atomic guarantee across the two stores.

### Write-behind / write-back

```
write(key, value):
  cache.set(key, value)
  queueForDbWrite(key, value)   # async

backgroundWorker:
  while True:
    batch = drainQueue()
    db.batchWrite(batch)
```

App writes only to the cache; cache flushes to DB asynchronously.

**Trade-off**: fast writes; cache is *the source of truth* between write-time and DB flush; risk of data loss on cache crash before flush; complex.

Use only when write throughput requirements demand it and data loss on cache failure is acceptable.

## Choosing the right pattern

| Scenario | Pattern |
|---|---|
| Read-heavy, stale-tolerant | Cache-aside (default) |
| Read-heavy, need fresh data | Cache-aside with shorter TTL and explicit invalidation |
| Write-heavy, can't tolerate stale | Write-through (verify atomicity strategy) |
| Write-heavy, throughput-bound, can tolerate occasional loss | Write-behind |
| Session store | Cache-aside on Redis with TTL; Redis is source of truth |
| Idempotency key | `SETNX key value EX ttl` on Redis; Redis is source of truth |

Default to **cache-aside** unless the trade-off justifies otherwise.

## Implementation steps (cache-aside on Redis + Postgres)

### Step 1 — choose the key shape

Stable, deterministic. Include enough context to avoid collisions.

```
key = "orders:v1:byId:{order_id}"
```

The `v1` prefix lets you bump cache versions when serialization shape changes — old cache entries silently expire instead of returning malformed data.

### Step 2 — choose the TTL

Two questions:
- How long can the cache stay stale before it's wrong?
- What's the cache miss cost?

Default TTL: 5–60 minutes. Short enough that staleness is bounded; long enough that miss rate stays low.

For data with **explicit invalidation** (write path invalidates cache), TTL is a safety net — set higher (1 hour to 1 day).

For data with **time-based decay** (rate-limit counters, session activity), TTL is the primary mechanism — set to the natural decay time.

### Step 3 — invalidate on write

```python
def update_order(order_id, fields):
    with db.transaction():
        db.orders.update(order_id, fields)
    cache.delete(f"orders:v1:byId:{order_id}")
```

**Invalidate after the DB write commits, not before.** Otherwise a concurrent read can repopulate the cache with stale data before the DB write finalizes.

For multi-key cache entries (list views, search results), invalidate by pattern or version-bump the prefix. `SCAN + DEL` works but is expensive at scale; pattern-prefix versioning is cleaner.

### Step 4 — handle the cache miss path

The cache miss → DB → cache repopulate path runs on every cold key. If 1000 requests for the same key arrive during cold-cache, all 1000 hit the DB. This is **cache stampede** — see below.

### Step 5 — handle cache unavailability

Redis is down or slow. Two failure modes:

- **Hard failure**: every request blocks on the Redis timeout, then falls back to DB. App stays up but slow.
- **Silent degradation**: every request hits DB, DB load spikes 10×, DB falls over. Cascading failure.

Defenses:

- **Short Redis client timeout** (50–200ms). Prevents requests blocking on Redis.
- **Circuit breaker** around Redis calls — after N failures, skip Redis for the next window.
- **Capacity-plan the DB for worst-case "Redis down"**. If the DB can't handle no-cache traffic, the cache is load-bearing in a dangerous way.

## Cache stampede — the under-appreciated failure mode

Symptom: a hot key expires; the next 100ms sees 1000 concurrent requests for it; all 1000 do `cache.get → miss → db.read → cache.set`. DB takes 1000 simultaneous identical queries. DB latency spikes. Other requests waiting on DB connections back up.

Prevention strategies:

### 1. Lock-and-load (single-flight)

First miss acquires a short Redis lock; other misses wait for the loader.

```python
def get_or_load(key):
    val = cache.get(key)
    if val: return val

    lock = redis.setnx(f"lock:{key}", "1", ex=5)
    if lock:
        try:
            val = db.read(key)
            cache.set(key, val, ttl=300)
            return val
        finally:
            redis.delete(f"lock:{key}")
    else:
        # someone else is loading; wait briefly then retry from cache
        time.sleep(0.05)
        return cache.get(key) or db.read(key)
```

Pros: prevents stampede entirely. Cons: lock complexity; one failed loader can hold the lock; need timeout fallback.

### 2. Probabilistic early expiration

Refresh the cache before it expires, based on a probability that increases as expiration approaches.

```python
def get_with_early_refresh(key, ttl):
    val, expires_at = cache.get_with_ttl(key)
    now = time.time()
    remaining = expires_at - now

    # Probabilistically refresh if remaining < 20% of original TTL
    if remaining < ttl * 0.2 and random.random() < 0.1:
        async_refresh(key)

    return val
```

Spreads refresh load over time; no single moment of mass expiration.

### 3. Jittered TTLs

Add randomness to TTLs so a batch of cache-set operations doesn't all expire at the same moment.

```python
cache.set(key, value, ttl=300 + random.randint(0, 60))
```

Trivial change; surprisingly effective for "this batch of items all got cached together" scenarios.

### 4. Stale-while-revalidate

Return stale value immediately; refresh asynchronously.

```python
def get_swr(key):
    val, is_stale = cache.get_with_stale_flag(key)
    if is_stale:
        async_refresh(key)
    return val
```

Cache item never blocks the response path; refresh happens in background. Cost: brief staleness even after the refresh trigger.

## Trade-offs summary

| Pattern | Read latency | Write latency | Staleness risk | Complexity |
|---|---|---|---|---|
| Cache-aside | low (hit) / DB (miss) | DB + invalidate | medium | low |
| Read-through | low (hit) / DB (miss) | DB + invalidate | medium | medium |
| Write-through | low (hit) | cache + DB | low | medium |
| Write-behind | low (hit) | cache | high (during outage) | high |

## Common failure modes

### Cache stampede
**Detection**: DB sees a sudden burst of identical queries; one cache key just expired.
**Fix**: lock-and-load, probabilistic early expiration, jittered TTL, or stale-while-revalidate.

### Stale cache after DB write
**Detection**: user writes a value, immediately reads, sees old data.
**Fix**: explicit cache invalidation on write; or write-through; or shorter TTL with acceptance of staleness window.

### Dual-write divergence
**Detection**: cache and DB disagree; cache has data DB doesn't have, or vice versa.
**Fix**: cache invalidation only (not dual-write); or write-through with explicit atomicity strategy; or write-behind with reconciliation.

### Cache key collision after serialization change
**Detection**: deserialization errors after a deploy; old format mixed with new.
**Fix**: version the cache key prefix (e.g., `orders:v1` → `orders:v2`); old keys silently expire.

### Cache unavailability cascading to DB
**Detection**: Redis goes down; DB QPS jumps 10×; DB falls over.
**Fix**: short Redis timeout; circuit breaker; capacity plan DB for cache-down scenario.

### Hot key
**Detection**: one cache key handles 50% of cache traffic; other keys are cold.
**Fix**: client-side micro-cache (50–500ms local cache) on top of Redis; or shard the hot key.

### No TTL on session-like data
**Detection**: Redis fills up with old sessions for users long-gone; eviction kicks in unpredictably.
**Fix**: explicit TTL on every key set; verify with `TTL key` audit.

## MCP tool opportunities

- **`recommend_caching_pattern`** — given read:write ratio, staleness tolerance, and throughput targets, return cache-aside / read-through / write-through / write-behind recommendation with rationale.
- **`detect_stampede_risk`** — analyze cache code for stampede prevention; flag missing lock-and-load / jittered TTL / SWR.
- **`generate_cache_key_versioning_plan`** — given a schema change, produce a versioned-key migration plan.

## What to read next

- `../redis-on-azure.md` — Redis tier, persistence, eviction
- `../engine-selection.md` — Redis as cache vs primary store
- `connection-pool-sizing.md` — sibling pattern (Redis connection pool is also a concern)
- `partition-key-design.md` — sibling pattern (Cosmos partition key)
- `azure-microservices-resilience` skill — circuit breaker around cache calls
