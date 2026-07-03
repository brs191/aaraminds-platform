# Pattern: Retry-Timeout

## Problem

Network calls fail transiently — dropped packets, brief overload, momentary GC pauses on the receiver. Without retries, the caller surfaces every transient blip as a user-facing error. Without timeouts, a hung receiver blocks the caller's thread indefinitely, exhausting connection pools. The right combination handles transient faults without amplifying real incidents.

## Use When

- The downstream operation is idempotent (safe to retry without duplicate side-effects)
- Failures are observed to be transient (5xx errors recover within seconds; 4xx errors don't)
- The caller can afford the added latency of a few retries (interactive request budget allows it)
- You have a timeout budget — total time waiting must stay within the caller's own SLA

## Avoid When

- The operation is not idempotent and there's no idempotency key (don't retry "charge card")
- Failures are deterministic (4xx errors won't get better with another attempt)
- The downstream is already overloaded — retries make it worse (use circuit breaker instead)
- Retry budget exceeds the caller's request budget (retrying for 30s on a 2s API)

## Azure Implementation

### Implementation Steps

1. Set a per-call timeout for every outbound dependency call (HTTP, DB, messaging)
2. Choose a retry policy: max attempts, base delay, backoff strategy (exponential), jitter
3. Classify errors: retry transient (timeout, 502/503/504); don't retry permanent (400, 401, 403, 404, 409)
4. Cap the total retry budget — total wait should be ≤ caller's deadline minus safety margin
5. Add jitter to backoff to prevent thundering herd when many clients retry the same downstream
6. Emit retry metrics; alert when retry rate exceeds normal baseline (signals downstream degradation)
7. Pair with circuit breakers — break the loop when retries are not helping

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| HTTP client | Polly (.NET), retry middleware (Go) | Exponential backoff with jitter, max 3 attempts |
| Service Bus client | Built-in retry policy | Default exponential, with DLQ for poison messages |
| Database | Entity Framework / SQL driver | Transient fault handling (retry on connection drop) |
| API client SDKs | Azure SDK auto-retry | Configurable, on by default for most SDKs |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Reliability | Hides transient errors; raises end-to-end success rate |
| Latency | Each retry adds delay — P99 latency can balloon under degraded conditions |
| Downstream pressure | Retries multiply load — risky during incidents (retry storm) |
| Idempotency burden | Receiver must handle duplicates safely or you risk double-effects |
| Observability cost | Need to log attempts to debug why a request was slow |

## Common Failure Modes

- **Retry storm** — All callers retry simultaneously when downstream is slow, multiplying load and prolonging the incident.
  - Detection: Request rate to a downstream service spikes 3–10x during its incident.
  - Prevention: Add jitter (random delay) to retry backoff; pair with circuit breakers.

- **Retry budget exceeds request budget** — Service A has 2s SLA, retries downstream with 30s budget, A times out anyway after wasting 30s.
  - Detection: P99 latency far exceeds SLA during dependency incidents.
  - Prevention: Set total retry timeout ≤ caller's SLA minus 100–500ms safety margin.

- **Retrying non-idempotent operations** — POST /charge retried after timeout charges customer twice.
  - Detection: Customer complaints of duplicate charges aligned with downstream latency spikes.
  - Prevention: Require an idempotency key on every retryable mutation; only retry GET/PUT/DELETE without keys.

- **Retrying permanent errors** — Retrying a 401 Unauthorized 5 times doesn't help; it just wastes the budget.
  - Detection: Logs show retries on 4xx codes.
  - Prevention: Classify errors before retry; retry only on transient codes (5xx, 408, timeouts).

## Decision Signals

Use retry-timeout when:
- The downstream call is on the request path AND the operation is idempotent
- Logs show occasional transient errors (timeouts, 503s, connection resets) that vanish on second attempt

Skip when:
- The mutation lacks idempotency keys
- You're inside a long-running batch where you can fail fast and resume

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Polly / gobreaker | Client retry policy | Composable policies (timeout + retry + circuit breaker) |
| Service Bus | Built-in retry | Configurable per-receiver, supports DLQ for poison messages |
| Application Insights | Retry telemetry | Track retry count per dependency to spot trends |
| Azure SDK | Auto-retry built-in | Most Azure SDKs retry transient errors by default |

## Go Implementation Notes

Use `github.com/cenkalti/backoff/v4` for exponential backoff with jitter:
- `backoff.NewExponentialBackOff()` with `MaxElapsedTime` to cap total budget
- Wrap in `backoff.Retry(operation, b)` for idempotent calls
- Combine with `context.WithTimeout(ctx, perCallTimeout)` for per-attempt limits

For HTTP clients, prefer middleware that classifies errors and only retries transient codes (5xx, network errors), not 4xx.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends retry-timeout for every described synchronous call
- `detect_architecture_risks` — flags retries without timeouts, retries without jitter, retries on non-idempotent operations
- `analyze_resilience_posture` — scores retry policy quality across all dependencies
- `generate_idempotency_design` — sketches the idempotency key scheme to make retries safe

## Related Patterns

- **Circuit Breaker** — kicks in when retries aren't helping; stops the retry storm
- **Idempotent Consumer** — required to make retries safe for mutations
- **Bulkhead** — limits the damage of a retry storm to one resource pool

## References

- Skill: `../resilience-patterns.md` — exponential backoff math and jitter formulas
- Pattern: `circuit-breaker.md` — failsafe when retries fail consistently
- Pattern: `../../../microservices-data-architecture/references/patterns/idempotent-consumer.md` — prerequisite for safe retries on mutations
