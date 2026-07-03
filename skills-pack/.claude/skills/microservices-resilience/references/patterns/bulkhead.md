# Pattern: Bulkhead

## Problem

A service with one connection pool, one thread pool, and one shared queue is one slow dependency away from total failure. If the payment API hangs, every thread waits for it; even unrelated endpoints (health checks, catalog reads) stall behind the queue of stuck threads. The bulkhead pattern isolates resources so that exhaustion in one partition doesn't sink the whole ship.

## Use When

- One service depends on multiple downstream systems with different latency/availability profiles
- An incident in one dependency must not block traffic to other dependencies
- The service serves both critical and non-critical traffic that should not contend for the same resources
- You have observed (or can model) that one dependency's slowness blocked all endpoints

## Avoid When

- The service has only one downstream and one type of traffic — partitioning adds complexity for no benefit
- Resource pools are already so small that partitioning leaves each partition unable to handle normal load
- You can solve the problem more simply with timeouts and circuit breakers alone

## Azure Implementation

### Implementation Steps

1. Identify each downstream dependency the service calls
2. Allocate a separate connection pool / thread pool / semaphore per dependency
3. Size each pool based on that dependency's expected throughput and latency (Little's Law)
4. Reject (or queue with limit) requests that overflow a partition rather than spilling to other partitions
5. Optionally isolate by tenant or traffic class (premium vs. free, critical vs. background)
6. Monitor per-partition saturation; alert when one partition is full while others are idle

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Per-dependency clients | HttpClientFactory (.NET) named clients, separate Go `http.Transport`s | One transport per downstream, distinct connection pool |
| Semaphores | In-process `Semaphore` or `chan struct{}` (Go) | Cap concurrent in-flight calls per dependency |
| Container Apps replicas | Separate replica sets per traffic class | Critical traffic gets its own replicas, isolated from batch jobs |
| Service Bus | Separate queues/subscriptions per priority | Critical messages don't queue behind background work |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Reliability | Strongly improved — one bad dependency can't poison the whole service |
| Utilization | Lower — partitioned pools have spare capacity that can't be borrowed |
| Tuning | Each partition must be sized; mistakes cause artificial rejections |
| Operational complexity | More dashboards, more thresholds, more alerts |
| Latency under load | Better — no head-of-line blocking on a slow downstream |

## Common Failure Modes

- **Under-sized partitions** — Each pool too small; legitimate traffic gets rejected even though the service has idle capacity overall.
  - Detection: HTTP 429/503 rate climbs while CPU and memory show idle.
  - Prevention: Size pools to 95th-percentile concurrent load; monitor partition saturation.

- **Shared underlying resource** — Pools look separate but share the OS-level socket pool or DB connection limit. Bulkhead is theatrical.
  - Detection: One dependency's slowness still affects others despite "partitioned" pools.
  - Prevention: Confirm partitioning at every layer (HTTP transport, DB connection string, thread pool).

- **Global cascade despite bulkheads** — Bulkheads protect compute but the receiving Service Bus has one shared throughput budget; one client saturates it for all.
  - Detection: Throttle errors on Service Bus correlate with high traffic from one client.
  - Prevention: Use separate Service Bus namespaces or premium tier with throughput units per tenant.

- **Bulkheads with no fallback** — Partition saturates, rejects requests with 503, caller has no plan.
  - Detection: 503 errors propagate to user.
  - Prevention: Pair with fallback (cached response, default, graceful degradation message).

## Decision Signals

Apply bulkheads when:
- A post-mortem says "all endpoints failed because dependency X hung"
- One service calls 3+ downstreams with different reliability profiles
- Premium and free traffic share the same compute and premium SLA needs protection

Skip when:
- The service has one dependency and one traffic class — keep it simple
- Pool sizing is so small that further partitioning leaves no slack

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Container Apps | Replica isolation | Separate replica sets per traffic class or tenant |
| HttpClientFactory | Per-dependency pools | Named clients with isolated connection pools |
| Service Bus Premium | Per-namespace throughput | Tenant-level isolation of messaging capacity |
| Azure SQL elastic pools | Per-tenant DB isolation | Noisy neighbor protection at the database layer |

## Go Implementation Notes

Per-dependency `http.Transport`:
```
paymentTransport := &http.Transport{MaxIdleConnsPerHost: 20}
inventoryTransport := &http.Transport{MaxIdleConnsPerHost: 10}
```
Plus a semaphore (`chan struct{}` of size N) to bound concurrent in-flight calls per dependency. Pair with `context` for per-call timeouts.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests bulkheads when describing a service with 2+ dependencies and shared resources
- `detect_architecture_risks` — flags single connection pools shared across all downstreams
- `analyze_resilience_posture` — scores partition coverage and sizing per dependency

## Related Patterns

- **Circuit Breaker** — complementary; breaker stops calls to a sick dependency, bulkhead limits damage if breaker fails to trip
- **Queue-Based Load Leveling** — bulkhead at the queue layer; smooths bursts
- **Service Mesh** — can enforce bulkheads at the sidecar level

## References

- Skill: `../resilience-patterns.md` — pool sizing using Little's Law
- Pattern: `circuit-breaker.md` — pairs with bulkhead for layered protection
- Pattern: `retry-timeout.md` — must respect bulkhead limits to avoid amplifying load
