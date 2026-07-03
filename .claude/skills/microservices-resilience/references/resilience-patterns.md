# Skill — Microservices Resilience Patterns

## Purpose

Design failure-aware microservices that degrade gracefully under load, latency, or partial outages. This skill covers the patterns that prevent one slow or failing service from cascading failure across the entire system. Use this when services make inter-service calls and you need to decide how to handle timeouts, retries, circuit breaks, and load shedding.

## Core Principle — Assume Every Call Will Fail

Every call from one service to another will eventually:
- Timeout (network delay, GC pause, slow query)
- Fail (5xx error, exception)
- Be unavailable (service restart, pod crash, zone outage)

Design assuming all three will happen, and the system must respond defensively.

## Pattern 1 — Timeout

**Problem:** Service A calls Service B, which is slow. Service A's thread blocks indefinitely, consuming resources. If B is slow for all callers, A's thread pool exhausts and A stops responding.

**Solution:** Wrap every external call in a deadline. If the call exceeds the deadline, fail fast.

**Implementation:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Do(ctx, request)
if err == context.DeadlineExceeded {
  // Timeout — fail fast, don't retry
  return ErrTimeout
}
```

**Azure implementation:**
- Container Apps: pod timeout enforced by ingress
- App Service: request timeout configurable
- Service Bus: message lock duration (how long to process before re-delivering)

**When to use:**
- Every HTTP call to another service must have a timeout
- Every queue read must have a receive timeout (don't block forever waiting for messages)
- Every database query must have a timeout

**Recommended timeouts:**
- Synchronous API call: 1–5 seconds (typical user-facing latency bound)
- Asynchronous worker: 30–60 seconds (more lenient, background operation)
- Database query: 5–10 seconds (catch runaway queries)
- Message processing: 1 minute (time to process and respond)

**Trade-offs:**
- Too aggressive (100ms): legitimate slow requests fail
- Too lenient (30s): resource exhaustion if service is really slow

## Pattern 2 — Retry with Exponential Backoff and Jitter

**Problem:** A transient error (network blip, temporary overload) causes failure. A single retry succeeds.

**Solution:** Retry only transient failures — 5xx server errors, network timeouts, and the two retriable 4xx codes (408 Request Timeout, 429 Too Many Requests). Never retry other 4xx client errors: the request is malformed and a retry fails identically. Use exponential backoff with jitter. (The explicit retry / do-not-retry lists are below.)

**Implementation:**
```
Retry logic:
  Attempt 1: immediate
  Attempt 2: wait 1 second + jitter (±20%)
  Attempt 3: wait 2 seconds + jitter
  Attempt 4: wait 4 seconds + jitter
  Attempt 5: wait 8 seconds + jitter
  Max attempts: 5
  Max total backoff: ~30 seconds
```

**Jitter purpose:** Prevent thundering herd. If all clients retry at the same time (e.g., all retry after 2 seconds), the recovered service gets hammered again.

**What to retry:**
- 408 (Request Timeout) — server was slow
- 429 (Too Many Requests) — rate limit, back off
- 503 (Service Unavailable) — server busy, retry later
- 5xx errors (except 501)
- Network timeouts, connection resets

**What NOT to retry:**
- 400 (Bad Request) — client error, will fail again
- 401 (Unauthorized) — authentication error, will fail again
- 404 (Not Found) — resource not found, won't appear later
- 409 (Conflict) — request violates state, retry won't help
- 429 with `Retry-After: 0` — don't retry yet

**Azure implementation:**
- Service Bus: automatic retry policy configurable (exponential backoff built-in)
- Application Insights: automatic retry on transient errors
- Container Apps: Dapr sidecars handle resiliency policies

**Trade-offs:**
- Over-retrying: compounds load on a struggling service
- Under-retrying: legitimate transient errors fail
- Idempotency: every retried call must be idempotent (same result on retry)

## Pattern 3 — Circuit Breaker

**Problem:** Service B is down or saturated. Service A keeps calling it, burning retries and timeouts. B never recovers because A keeps hammering it.

**Solution:** Circuit breaker: stop calling a failing service temporarily, letting it recover.

**States:**
- **Closed:** normal operation, call the service
- **Open:** service is failing, reject calls immediately (fail fast)
- **Half-Open:** probe the service with one request; if it succeeds, close the circuit; if it fails, open again

**Implementation:**
```
Closed (normal):
  Call count < threshold? → Call service
  Error rate < threshold? → Stay closed
  Error rate > threshold for 30s? → Open

Open (failing):
  Reject all calls immediately (fail fast)
  After timeout (30-60s)? → Half-open

Half-open (probing):
  Allow next request through
  Success? → Close, resume normal traffic
  Failure? → Open, wait again
```

**Azure implementation:**
- Dapr sidecars: circuit breaker built-in
- Polly (C#): circuit breaker library
- Hystrix (deprecated, but pattern is standard)

**Metrics to track:**
- Circuit breaker state transitions (closed → open, open → half-open, etc.)
- Time spent open (how long was the service unavailable?)
- Requests rejected (fail-fast requests during open state)

**When to use:**
- Service-to-service calls where the calling service can degrade
- Circuit breaker + timeout: timeout lets individual requests fail fast; circuit breaker stops the onslaught

**Trade-offs:**
- Complexity: circuit breaker state machine is subtle
- Debugging: "why did my request fail?" — was it timeout, circuit breaker, or the service?
- False positives: slow service might trip circuit breaker even though it's recovering

## Pattern 4 — Bulkhead (Resource Isolation)

**Problem:** Service B is slow. All of Service A's threads wait for B's responses. A's thread pool exhausts. A can't respond to any caller.

**Solution:** Isolate resources by service. Dedicate a thread pool to B's calls; if B's pool exhausts, other services' calls aren't affected.

**Implementation:**
```
Thread pools:
  Global pool: 100 threads
  Payment service pool: 20 threads (dedicated)
  Inventory service pool: 20 threads (dedicated)
  Other service pool: 60 threads (shared)

If Payment service's 20 threads are blocked:
  - Inventory can still use its 20 threads
  - Other services can use the shared 60 threads
  - Payment calls fail fast (pool exhausted) instead of blocking globally
```

**Azure implementation:**
- Container Apps: separate containers per critical dependency (built-in isolation)
- Dapr: bulkhead policies per service endpoint
- Connection pooling: separate connection pools per database or external API

**When to use:**
- Critical dependencies (payment processor, auth service, inventory) that could cascade failure
- High-contention resources (database connections, thread pools)

**Trade-offs:**
- Resource overhead: multiple pools use more memory
- Configuration: need to tune pool sizes per environment
- Complexity: adds another failure mode (pool exhaustion)

## Pattern 5 — Queue-Based Load Leveling

**Problem:** Load spikes cause service overload. Requests queue up internally, threads block, service becomes unresponsive.

**Solution:** Accept requests to a queue, process asynchronously at a steady pace. Queue absorbs spikes; processing is rate-limited.

**Implementation:**
```
Request arrives
  → Add to queue (fast, non-blocking)
  → Return immediately (accepted)

Worker processes:
  → Read from queue
  → Process at steady rate (e.g., 100 requests/second)
  → Queue depth is the buffer

If queue is full:
  → Return "service busy" (shed load, don't block)
```

**Azure implementation:**
- Service Bus queues: built-in load leveling
- Event Hubs: ingestion service for high-throughput
- Container Apps jobs: process queue items at fixed rate

**When to use:**
- Background jobs, reporting, non-critical paths (latency is acceptable)
- High-volume, bursty workloads (orders arriving in waves)

**Trade-offs:**
- Latency: requests now have queue wait time (not immediate)
- Queue depth visibility: need monitoring to see queue buildup
- Deadletter queue: implement for messages that fail repeatedly

## Pattern 6 — Graceful Degradation

**Problem:** A dependency goes down. The entire feature becomes unavailable.

**Solution:** Degrade gracefully. If the dependency is unavailable, return reduced functionality instead of failure.

**Examples:**
- Product Recommendation is down? Return empty recommendations instead of failing the page load
- User Profile cache is stale? Show cached data with a "possibly out of date" indicator instead of erroring
- Shipping estimator is down? Show a generic estimate instead of an exact one

**Implementation:**
```
if order.total > 1000:
  estimated_shipping = call_shipping_service()
  if estimated_shipping fails:
    estimated_shipping = "Free shipping on orders >$1000"
    log warning: "shipping service down, using default"

return order with estimated_shipping (never null)
```

**When to use:**
- Non-critical enrichment (recommendations, ratings, detailed info)
- Features with safe defaults (shipping, tax, discounts)
- Avoid for: core transaction (payment, inventory)

**Trade-offs:**
- UX: reduced functionality is noticeable
- Defaults: must be safe (free shipping is safe, random recommendation is not)
- Debugging: need clear visibility into which features are degraded

## Pattern 7 — Dead-Letter Queue and Replay

**Problem:** A message fails processing. Retries exhaust. The message is lost or causes poison pill (breaks message processor repeatedly).

**Solution:** Route poison pills to a dead-letter queue (DLQ). Review them manually or replay after the service is fixed.

**Implementation (Service Bus):**
```
Main queue: orders
Dead-letter queue: orders_dlq

Message fails 5 times → Service Bus auto-moves to orders_dlq
Operator reviews: "why did this order fail?"
Fix the bug
Replay messages from orders_dlq to orders
```

**When to use:**
- Critical message processing (orders, payments, fulfillment)
- Any async workflow where messages must not be lost

**Azure implementation:**
- Service Bus: auto-forward dead-letter, TTL policies
- Event Hubs: manual sink for failed events

## Worked Example — Order Service Calling Payment Service

**Resilience configuration:**
```
Timeout: 5 seconds (payment should respond quickly)
Retry: 
  - Retry on 408, 429, 503, timeout
  - Up to 3 attempts
  - Backoff: 1s, 2s, 4s + jitter
Circuit breaker:
  - Open if error rate > 50% for 30s
  - Half-open after 60s
  - Require 5 successful calls to close
Bulkhead:
  - Dedicated thread pool: 30 threads
  - If exhausted, reject immediately
Graceful degradation:
  - If Payment service down, mark order as "pending payment review"
  - Operator manually processes later
```

## Verification Questions

1. **Timeout:** Does every external call have a timeout?
2. **Retry:** Are retries only on transient errors? Is the call idempotent?
3. **Circuit breaker:** When a service is down, does traffic stop immediately or pound it?
4. **Bulkhead:** If one dependency saturates, can other dependencies still respond?
5. **Degradation:** If a non-critical dependency is down, does the core flow still work?
6. **DLQ:** For critical messages, is there a path to recover on failure?

## What to read next

- For specific patterns: `patterns/circuit-breaker.md`, `patterns/retry-timeout.md`, `patterns/bulkhead.md`
- For observability of resilience: `../../azure-microservices-observability/references/observability-design.md`
- For Azure service configurations: `../../azure-service-mapping/references/azure-mapping.md`
