# Pattern: Cache-Aside

## Problem

Hot data read on every request — user profile, product catalog, configuration — strains the primary database and slows the user-facing path. The same query runs millions of times per minute against a row that changes once a day. Cache-aside puts an in-memory layer between the application and the database; the application reads from the cache, falls back to the DB on miss, and populates the cache for next time.

## Use When

- Read-to-write ratio is high (>10:1) for the data in question
- The data is small enough to fit in cache (or you can shard sensibly)
- Staleness on the order of seconds-to-minutes is acceptable
- Cache miss latency is acceptable (cold reads will hit the DB)

## Avoid When

- Strong read-your-writes consistency is required (use cache invalidation carefully or skip cache)
- The data churns more than it's read (every read is fresh; cache adds latency)
- Cache TTL must be near-zero to be safe — at that point, just hit the DB
- Cached data is large and rarely re-read; cache fills with cold data and evicts hot data

## Azure Implementation

### Implementation Steps

1. Identify hot read paths via query metrics — top 10 queries by call rate
2. Choose a cache: Azure Cache for Redis (recommended), or in-process cache for single-instance reads
3. Implement read: check cache first; on miss, read DB, write to cache with TTL, return result
4. Choose invalidation strategy: TTL-based (simplest), explicit invalidation on write (more consistent), or pub/sub
5. Set TTL based on acceptable staleness (product details: 5 min; user profile: 30 sec; config: 5 min)
6. Monitor hit rate; aim for >80% on hot keys
7. Handle cache failures gracefully — degrade to DB-only reads, never error to user

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Cache | Azure Cache for Redis (Standard tier) | Persistence on for warm restart, replication for HA |
| Client library | StackExchange.Redis (.NET), go-redis (Go) | Connection pool, automatic reconnect |
| Cache invalidation | Redis pub/sub or app-driven DELETE | On write, publish invalidation event or DELETE key |
| Metrics | Application Insights | Custom `cache_hit_ratio` per key prefix |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Read latency | Strongly improved on hits (sub-ms vs. 10–100ms DB) |
| Write latency | Slightly slower (write + cache invalidation) |
| Cost | Cache adds ~$30–500/month; usually justified by DB savings |
| Consistency | Eventual; staleness up to TTL |
| Complexity | Adds invalidation logic; cache failures must be handled |

## Common Failure Modes

- **Stampede on cache miss** — Hot key expires; thousands of concurrent requests miss and all hit the DB simultaneously.
  - Detection: DB CPU spike correlated with cache miss for a hot key.
  - Prevention: Single-flight (only one request fetches; others wait); probabilistic early refresh.

- **Stale data after write** — Writer updates DB but forgets to invalidate cache; readers see old data for TTL duration.
  - Detection: User reports "I changed this; why does it still show the old value?"
  - Prevention: Invalidate cache in the same code path as the DB write; or use write-through.

- **Negative caching not handled** — Code caches "not found" indefinitely; entity appears later but cache says "missing".
  - Detection: Newly created entity is invisible to readers for TTL period.
  - Prevention: Short TTL for negative cache entries; invalidate on creation.

- **Cache as source of truth** — App treats cache as authoritative; cache loss = data loss.
  - Detection: After cache restart, "data" disappears.
  - Prevention: Cache is always a derived view; the DB is the source of truth.

## Decision Signals

Use cache-aside when:
- A small number of queries dominate read load
- DB CPU is high during peak; queries are repeatable lookups
- Read latency exceeds budget (P95 > 50ms for cacheable reads)

Skip when:
- Reads are unique per call (no repetition to cache)
- Strong consistency required (write must be immediately visible)

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Azure Cache for Redis | Cache layer | Managed, replicated, persistent option |
| Azure SQL / Cosmos | Source of truth | Cache invalidated on writes here |
| Application Insights | Hit rate telemetry | Track cache effectiveness |
| CDN | Edge cache | For static or semi-static content (images, HTML) |

## Go Implementation Notes

```go
func GetProduct(ctx context.Context, id string) (*Product, error) {
    if p, ok := cache.Get(ctx, "product:"+id); ok {
        return p, nil
    }
    p, err := db.GetProduct(ctx, id)
    if err != nil {
        return nil, err
    }
    cache.Set(ctx, "product:"+id, p, 5*time.Minute)
    return p, nil
}
```
For invalidation on write: `cache.Delete(ctx, "product:"+id)` immediately after DB write commits.

For stampede protection: use `singleflight.Group` to coalesce concurrent misses.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends caching when a hot read path is described
- `detect_architecture_risks` — flags caches without invalidation, infinite TTLs, cache as source of truth
- `generate_caching_design` — produces key strategy, TTL, and invalidation flow for the described data
- `map_patterns_to_azure_services` — picks Redis tier based on size and HA requirements

## Related Patterns

- **Database per Service** — each service can have its own cache layer
- **CQRS** — read store is effectively a caching layer
- **Circuit Breaker** — protect against cache failures
- **CDN** — extends caching to the edge

## References

- Skill: `../../../microservices-data-architecture/references/data-architecture.md` — caching in the broader data strategy
- Skill: `../../../azure-microservices-cost-review/references/cost-and-tradeoffs.md` — cache cost vs. DB tier savings
- Pattern: `../../../microservices-data-architecture/references/patterns/cqrs.md` — when caching grows into full read-side projection
