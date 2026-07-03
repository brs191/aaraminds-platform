# Pattern: Circuit Breaker

## Problem

When a downstream service is slow or failing, naive clients keep sending requests, exhausting their own threads and connection pools waiting for timeouts. The failure of one dependency cascades: callers become unresponsive too, and clients of *those* callers fail next. A circuit breaker stops sending requests to a known-bad dependency until it recovers, freeing the caller to fail fast or degrade gracefully.

## Use When

- A downstream service is on the critical path and its failure would otherwise stall the caller
- The downstream has variable availability (third-party APIs, flaky legacy systems, geographically distant services)
- Graceful degradation is possible — the caller can return a cached result, a default, or a partial response
- You measure response latency and error rate per dependency (you need signals to trip the breaker)

## Avoid When

- The dependency is in-process or local (in-memory cache) — no network, no need
- A single failed request is itself catastrophic — circuit breakers fail-fast subsequent calls, which may be worse
- You have no fallback at all — opening the breaker just shifts the error type from "timeout" to "circuit open"

## Azure Implementation

### Implementation Steps

1. Identify each external dependency the service calls (DB, cache, other services, third-party APIs)
2. Wrap each dependency client with a circuit breaker (per-dependency, not global)
3. Configure thresholds: trip after N consecutive failures or X% error rate over Y seconds
4. Set the half-open probe: after Z seconds, allow one test request through
5. Define the fallback for the open state: cached value, default response, or fail-fast error
6. Emit metrics for each state transition (Closed → Open → Half-Open → Closed)
7. Alert when a breaker stays Open longer than expected (signals downstream incident)

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Client library | Polly (.NET) or sony/gobreaker (Go) | Per-dependency policy with thresholds |
| State (optional) | Redis | Share breaker state across pod replicas if needed |
| Metrics | Application Insights | Custom metric `circuit_breaker_state` per dependency |
| Mesh-level | Dapr / Istio | Sidecar-applied breaker policies, no app code change |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Reliability | Caller protects itself, but downstream sees fewer requests when it might recover |
| Complexity | Per-dependency tuning required; bad thresholds cause flapping |
| User experience | Better when fallbacks exist; worse if open state surfaces as raw errors |
| Observability cost | Each breaker state transition adds telemetry volume |
| Tunability | Thresholds are workload-specific — needs load testing to calibrate |

## Common Failure Modes

- **Flapping breaker** — Thresholds too tight; breaker oscillates Open/Closed every few seconds.
  - Detection: State transition rate >1/min for the same dependency.
  - Prevention: Add hysteresis (require N successful probes before closing); widen the failure window.

- **No fallback** — Breaker opens, returns 503 to caller, who has no plan. Caller's caller breaks.
  - Detection: Spike in 503 errors aligned with breaker open events.
  - Prevention: Every breaker must have a documented fallback (cache, default, or empty response).

- **Shared breaker across endpoints** — One slow endpoint trips the breaker for the whole downstream service, blocking healthy endpoints too.
  - Detection: All endpoints to service X fail when only one is degraded.
  - Prevention: Use per-endpoint breakers, not per-service breakers.

- **Probe storm in half-open** — Half-open allows all traffic through, immediately re-tripping the breaker.
  - Detection: Open → Half-Open → Open cycle <1 second apart.
  - Prevention: Half-open admits exactly one request, not "any request for X seconds".

## Decision Signals

Use a circuit breaker when you see:
- A service-to-service call where the caller has timeout-based latency spikes
- Logs showing "context deadline exceeded" or "connection refused" cascading
- An on-call rotation already responding to "service A is down because service B is down"

Skip it when:
- The call is purely local (in-memory) or the dependency has stronger SLA than yours
- You're calling a function in the same process

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Application Insights | Breaker telemetry | Tracks state transitions, dependency latency |
| Dapr | Sidecar policy | Service-mesh-level breakers without app code |
| Redis | Shared state | When breaker decisions must align across many replicas |

## Go Implementation Notes

In Go, use `sony/gobreaker` or `mercari/go-circuitbreaker`:
- Wrap each HTTP client per remote service
- Trip on consecutive failures or error ratio
- Use `OnStateChange` callback to emit Application Insights custom metric

Example structure: `internal/resilience/breakers/` defines a breaker per dependency, named after the downstream (e.g., `paymentBreaker`, `inventoryBreaker`).

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends circuit breakers wherever cross-service synchronous calls are described
- `detect_architecture_risks` — flags synchronous chains 3+ services deep with no breaker
- `analyze_resilience_posture` — scores breaker coverage across all dependencies
- `generate_architecture_decision_record` — drafts ADR for breaker policy choice (per-service vs. per-endpoint)

## Related Patterns

- **Retry-Timeout** — usually paired; breaker prevents retry storms on a sick dependency
- **Bulkhead** — isolates failure to a thread pool, complementary to breakers
- **Service Mesh** — moves breaker config out of code into infrastructure

## References

- Skill: `../resilience-patterns.md` — full breaker state machine and tuning guide
- Pattern: `retry-timeout.md` — timeout strategy that feeds the breaker's failure counter
- Pattern: `bulkhead.md` — partition resource pools so one breaker doesn't drain the others
