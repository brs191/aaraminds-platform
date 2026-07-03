---
name: microservices-resilience
description: Designs resilience controls for service-to-service calls and rollout patterns for Azure-hosted microservices. Covers timeouts, retries with jitter, circuit breakers, bulkheads, queue-based load leveling, graceful degradation, and rollout strategies (blue-green, canary, strangler fig). Use when adding a new outbound dependency, reviewing an incident where one slow service took down others, deciding a rollout strategy for a risky change, or evaluating whether a service has the controls to survive its dependencies failing. Do not use for cross-service data consistency (use microservices-data-architecture) or for messaging-broker choice (use microservices-async-messaging).
version: 1.0.0
last_updated: 2026-05-18
---

# Microservices Resilience

## When to use

Trigger this skill when the question is about how a service behaves when its dependencies misbehave: timeouts on outbound calls, retry policy, circuit breaker thresholds, bulkhead isolation, fallback / graceful degradation, queue-based load leveling, and rollout safety (blue-green, canary, strangler). Common triggers: "we had an incident where service A took down service B," "should we retry on this call," "how do I roll out a risky migration without a big-bang cutover," "circuit breaker keeps tripping."

Do **not** use this skill for: data consistency across services (`microservices-data-architecture`); choosing between sync REST and async messaging (`microservices-async-messaging`); SLO definition or alert design (`azure-microservices-observability`).

## The critical decision rule — every outbound call has a timeout

If the call has no timeout, the caller's reliability is bounded by the *slowest* response the dependency has ever produced. That's never the SLO you want. Every outbound call — HTTP, gRPC, database, Service Bus receive — has a configured timeout. Defaults are not timeouts; "no timeout" is a bug.

Once timeouts exist, then ask the next questions: should this call be retried, and if so with what backoff and jitter; is a circuit breaker warranted; do callers of *this* service need to be isolated from each other.

## The resilience-control selector

| Question | Control | Reference |
|---|---|---|
| Call may hang indefinitely | **Timeout** (always) | `references/resilience-patterns.md` |
| Call fails transiently (network, throttling) | **Retry with exponential backoff + jitter** | `references/patterns/retry-timeout.md` |
| Dependency is failing for sustained period | **Circuit breaker** (Closed → Open → Half-Open) | `references/patterns/circuit-breaker.md` |
| Multiple downstreams; one slow downstream starves others | **Bulkhead** (thread/connection pool isolation) | `references/patterns/bulkhead.md` |
| Inbound burst overwhelms processing capacity | **Queue-based load leveling** (Service Bus buffer) | `references/resilience-patterns.md` |
| Rolling out risky change without downtime | **Blue-green or canary deploy** | `references/patterns/blue-green-canary.md` |
| Replacing legacy system incrementally | **Strangler fig** | `references/patterns/strangler-fig.md` |

For the conceptual interplay of these controls (e.g., circuit breaker + retry = "stop retrying when breaker is open"), see `references/resilience-patterns.md`.

## Resilience-design logic

1. **Always-on:** every outbound call has a timeout. There is no exception. Default values in HTTP clients (`http.DefaultClient` in Go, `RestTemplate` without explicit config in Spring) are not safe; configure explicitly.

2. **Retry decision:** retry **only** if the operation is idempotent (or can be made so with an idempotency key). Retry **only** on transient failures (5xx, network timeouts, throttling), never on 4xx. Use exponential backoff with jitter — never fixed delay, never zero delay. Cap retries at 3 unless you have a specific reason.

3. **Circuit breaker decision:** add a breaker per dependency when sustained failure of that dependency would otherwise generate so much retry traffic that the dependency cannot recover. The breaker's job is "give the dependency room to heal."

4. **Bulkhead decision:** isolate connection pools per dependency when the service calls multiple downstreams and one downstream's slowness must not starve the others. For Spring Boot: separate `HttpClient` per downstream. For Go: separate `http.Client` and connection pools.

5. **Load leveling decision:** front any bursty inbound work (incoming webhooks, scheduled jobs that fan out) with a Service Bus queue. The queue absorbs the burst; the consumer processes at sustainable rate.

6. **Rollout strategy:**
   - **Blue-green** when the change is risky and you need an instant rollback path. Two full environments, traffic-switch the gateway/Front Door. Container Apps revisions support this natively.
   - **Canary** when you want to gradually expose risk. Route 5% → 25% → 50% → 100% with metric gates between steps.
   - **Strangler fig** when replacing a legacy system. New service handles new routes; legacy handles existing routes; routes migrate one at a time over months.

## Worked example — brownfield: a single slow downstream is taking down the gateway

Setup: existing Spring Boot API gateway on Container Apps calls 6 downstream microservices. One downstream (`pricing-service`) has been responding in 8-15 seconds during peak load. The gateway is timing out user requests across *all* routes — not just routes that need pricing. P99 latency on unrelated routes has degraded too.

Decision walk:

1. **Diagnose the spread.** The gateway uses a single `HttpClient` shared across all downstreams. Slow pricing calls hold connections in the shared pool; other routes can't acquire a connection and queue up. This is a textbook bulkhead-missing failure.
2. **Apply bulkhead.** Configure one `HttpClient` per downstream with its own connection pool. Pricing gets 20 connections; other downstreams get 10 each. Pricing's slowness can no longer starve the others. See `references/patterns/bulkhead.md`.
3. **Add timeout per call.** Pricing's call gets a 2 s timeout (we'd rather fail fast than hold the user). Per-route timeouts are configurable in the gateway. See `references/patterns/retry-timeout.md`.
4. **Add circuit breaker on pricing.** Resilience4j circuit breaker — Open after 50% failure rate over 30 s window, 30 s open-state, single half-open probe. When the breaker is Open, return the last-cached price with `X-Stale-Price: true` header (fallback) rather than failing. See `references/patterns/circuit-breaker.md`.
5. **Observability.** Emit `circuit_breaker_state{dependency="pricing-service"}` and `bulkhead_pool_saturation{dependency="pricing-service"}` via OpenTelemetry; alert if either is sustained for 5 minutes. See `azure-microservices-observability` skill.
6. **Roll out via canary.** The change touches every outbound call in the gateway. Deploy to 10% of traffic via Container Apps revision split, watch P99 latency and error rate, then 50%, then 100%. See `references/patterns/blue-green-canary.md`.

## Anti-pattern — naive retry without idempotency or backoff

**Bad:** "Make our payment service more reliable: add 3 retries to the charge call." The retry is implemented as a `for i := 0; i < 3; i++ { time.Sleep(100ms); try() }` loop with no jitter, no backoff, no idempotency key.

**Why it fails:**
- Without an idempotency key, the second retry on a transient timeout may produce a duplicate charge — the original call may have succeeded server-side; the timeout was just the response getting lost.
- Without backoff, all callers retry simultaneously; the downstream gets a coordinated spike right after the failure, making recovery harder.
- Without jitter, multi-instance callers synchronize their retry windows — same problem, larger amplitude.

**Detection signal:** look for retry loops with a fixed `sleep` value, or with no idempotency parameter on the request. In Go, `for range []int{1,2,3} { resp, err := http.Get(...) }`. In Spring, `@Retryable` without `backoff = @Backoff(delay = X, multiplier = Y)`.

**Fix:** require an idempotency key on every retried mutation. Use Resilience4j (Java) or `cenkalti/backoff` (Go) for exponential backoff with jitter. Classify error: only retry transient failures, never 4xx.

## Verification questions

1. Does every outbound call in this service have an explicit timeout?
2. For every retry: is the operation idempotent (with key if needed), and is backoff exponential with jitter?
3. For every dependency: is there a circuit breaker with documented thresholds, or an explicit decision that none is needed?
4. For multi-downstream services: is there bulkhead isolation, or proof that no downstream can saturate the others?
5. For risky rollouts: is there a canary plan with metric gates, or a blue-green strategy with documented rollback?
6. Are resilience-control signals (breaker state, pool saturation, retry count) emitted as metrics and alerted on?

## What to read next

- `references/resilience-patterns.md` — the conceptual integration of timeouts, retries, breakers, bulkheads
- `references/patterns/retry-timeout.md` — backoff math, jitter formulas, idempotency-key design
- `references/patterns/circuit-breaker.md` — state-machine, Resilience4j and Go implementations, threshold tuning
- `references/patterns/bulkhead.md` — connection-pool isolation, per-tenant bulkheads
- `references/patterns/blue-green-canary.md` — Container Apps revision-based rollout, metric gates
- `references/patterns/strangler-fig.md` — incremental legacy replacement, route-by-route migration
- `microservices-async-messaging` skill — for load leveling via Service Bus
- `azure-microservices-observability` skill — for emitting and alerting on resilience-control signals
