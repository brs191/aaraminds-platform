# Trace Sampling Strategies — Cost, Fidelity, Debuggability

## When to use this reference

Reach for this when designing an OTel Collector configuration for a new service, when the tracing bill from Azure Monitor or Grafana Tempo is climbing faster than traffic, when on-call complains that "I can't find the trace for the failed request" during an incident, or when the team is debating whether to enable distributed tracing in production at all because of cost fears. Also use it when retrofitting tracing onto a brownfield service where you cannot afford a 100% sample rate from day one.

## The core tradeoff

Tracing has three knobs that pull against each other:

1. **Cost** — every retained span costs storage, network, and query CPU. At 1 KB/span and 50 spans/request, 1000 RPS retains 50 MB/s. That's ~4 TB/day at full retention.
2. **Fidelity** — what fraction of requests are inspectable end-to-end. 100% means every request's trace can be pulled by ID; 1% means you can only see 1-in-100 requests, and the one you want during an incident is usually not in the kept set.
3. **Debuggability** — the probability that *the trace you need* is the one you kept. Random 1% sampling fails this; you wanted the failed request, but the kept trace shows a healthy one.

You cannot have all three. Pick the strategy that matches the service tier.

## The two sampler families

**Head-based sampling** decides at the trace's first span — typically at the API gateway or the first instrumented service — whether to keep the whole trace. Decision propagates via the `tracestate` header. Cheap to compute; runs in the SDK or at the edge collector; no buffering. Loses error-targeted fidelity because the decision is made before the request fails.

**Tail-based sampling** buffers all spans of a trace until the trace ends, then decides whether to keep it based on attributes (presence of error, latency above threshold, specific endpoint). Captures the traces that matter; pays for it with collector memory (must hold all in-flight traces for the trace window, typically 10–30 s) and a deployment that cannot be horizontally sharded naively — all spans of a trace must reach the same collector instance.

Most real systems use both. Head-based at the gateway for baseline coverage (e.g., 10%); tail-based at the collector to ensure 100% of errors and slow traces are kept regardless of the head decision.

## OTel Collector sampler choices

The OTel Collector ships these samplers. Pick by service tier and traffic profile.

| Sampler | Where it runs | Decision input | When to use |
|---|---|---|---|
| `AlwaysOn` / `AlwaysOff` | SDK | constant | Dev / local only. Never in prod. |
| `TraceIDRatioBased` | SDK | hash(traceID) | Head-based baseline. Default at 10% for user-facing services. |
| `ParentBased` | SDK | parent's decision | Always wrap whichever sampler is your root. Honors upstream's choice so a trace is consistent end-to-end. |
| `probabilistic_sampler` (processor) | Collector | hash(traceID) | Same as TraceIDRatioBased but at the collector, useful when SDK side is fixed and you need to drop more before storage. |
| `tail_sampling` (processor) | Collector | full trace attributes | The workhorse for production. Compose multiple policies (always sample errors, always sample slow, sample 5% of normal). |

The SDK sampler that should be the default in every service:

```yaml
# OTel SDK config (envvar-driven)
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1   # 10% head-based
```

`parentbased_traceidratio` means: if there is a parent span context (incoming `traceparent` header with a sampling decision), honor it. Otherwise, sample 10% based on trace ID hash. This keeps traces consistent — a trace is either fully sampled or fully dropped across all services it traverses.

## Tail-based sampling — the production pattern

This belongs in the OTel Collector, not the SDK. The pattern: every span goes to the collector at 100% from the SDK's perspective; the collector buffers, then decides.

```yaml
# OTel Collector config
processors:
  tail_sampling:
    decision_wait: 15s          # how long to buffer a trace before deciding
    num_traces: 100000          # max in-flight traces buffered
    expected_new_traces_per_sec: 1000
    policies:
      # Always keep errors
      - name: errors
        type: status_code
        status_code:
          status_codes: [ERROR]
      # Always keep slow traces (>1s end-to-end)
      - name: slow
        type: latency
        latency:
          threshold_ms: 1000
      # Always keep traces tagged for debugging
      - name: debug-tag
        type: string_attribute
        string_attribute:
          key: debug.force_sample
          values: ["true"]
      # Keep 100% of low-volume critical endpoints
      - name: checkout
        type: string_attribute
        string_attribute:
          key: http.route
          values: ["/v1/checkout"]
      # Sample 5% of everything else
      - name: baseline
        type: probabilistic
        probabilistic:
          sampling_percentage: 5

exporters:
  otlp/tempo:
    endpoint: tempo:4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tail_sampling, batch]
      exporters: [otlp/tempo]
```

The policy order does not matter — `tail_sampling` evaluates all policies and keeps the trace if any matches. The result: 100% of errors and slow traces survive (the ones you want during an incident), plus a 5% statistical baseline for normal traffic.

The cost: collector memory. At 100,000 in-flight traces × average 20 KB/trace = ~2 GB working set. The collector cannot be horizontally sharded naively because spans of a trace must land on the same instance. Two options:

1. **Single replicated collector with consistent hashing.** Use a `loadbalancingexporter` upstream to route spans by trace ID to a fixed set of tail-sampling collectors. Each backend sees a partition of trace IDs and can independently buffer + decide.
2. **Sidecar collectors.** Each service runs an OTel Collector sidecar doing tail sampling for its own traces only. Loses cross-service tail decisions (a trace that errors in service B but starts in A may be dropped at A's collector). Use only when cross-service tracing is not critical.

The standard pack pattern is option 1: an OTel Collector StatefulSet on AKS / Container Apps with `loadbalancingexporter` routing, scaled to handle peak in-flight trace volume.

## Per-service overrides

A monolith of one sampler config across the estate is wrong. Two services with different SLOs and different traffic shapes need different rates.

| Service tier | Head rate | Tail policy |
|---|---|---|
| **External user-facing API** (checkout, payment) | 100% from SDK (let tail decide) | Keep all errors, all > 500ms, 10% baseline |
| **Internal user-facing API** | 50% head | Keep all errors, all > 1s, 5% baseline |
| **Service-to-service synchronous** | parentbased (inherits upstream) | Keep all errors, 2% baseline |
| **Async worker (high volume)** | 1% head | Keep all errors, 0.1% baseline |
| **Batch job** | 100% head, no tail | Volume is low; just keep all of it |
| **Health check / probe traffic** | AlwaysOff at the SDK | Drop entirely; never reaches the collector |

Health-check traffic deserves special mention. Kubernetes liveness probes and load-balancer health checks generate enormous span volume that has zero diagnostic value. Drop at the SDK with a sampler that returns `AlwaysOff` when `url.path` is the health endpoint:

```java
// Spring Boot — exclude actuator from instrumentation
@Bean
public OtelSamplingRuleApplier excludeActuator() {
    return SamplingRule
        .when(SpanData::getName, name -> name.startsWith("/actuator"))
        .sample(Sampler.alwaysOff());
}
```

Or at the collector with a `filter` processor before `tail_sampling`:

```yaml
processors:
  filter/drop_health:
    error_mode: ignore
    traces:
      span:
        - 'attributes["http.route"] == "/healthz"'
        - 'attributes["http.route"] == "/readyz"'
```

## The fidelity vs cost calculation

Concrete numbers from a typical user-facing service:

- 1000 RPS, average 25 spans/request, 1 KB/span = 25 MB/s ingested at the collector
- Tail policy: 100% errors (~0.1% of traffic), 100% slow (~0.5%), 5% baseline = ~5.6% retention
- Stored volume: 1.4 MB/s = 120 GB/day, ~3.6 TB/month

Compare to 100% head-based retention: 25 MB/s = 2.1 TB/day, ~63 TB/month — and 99% of those traces are uninteresting successful requests.

The Azure Monitor / Application Insights pricing model charges per GB ingested. Grafana Tempo on Azure managed disk costs storage + read replicas; cheaper per GB than App Insights but you pay for the collector infrastructure. The tail-sampling pattern saves an order of magnitude in either backend.

| Backend | Cost model | Best for |
|---|---|---|
| **Grafana Tempo + Azure Blob** | Object storage + collector compute; trace-by-ID lookups are cheap, search is expensive | High-volume estates where most traces are pulled by trace ID from a log |
| **Azure Monitor (App Insights)** | Per-GB ingest + per-GB retention beyond 90 days | Estates already on Log Analytics; cost predictable but climbs fast |
| **Azure Managed Grafana with Tempo data source** | Tempo cost + managed Grafana subscription | Pack default — Azure-native + the rest of Grafana for free |

## Force-sampling for debugging

When a user reports a problem, the on-call needs the trace for *their* request. With 5% sampling, the odds are bad. The escape hatch: propagate a `debug.force_sample=true` baggage entry and tail-sample on it.

```http
GET /v1/orders/123 HTTP/1.1
baggage: debug.force_sample=true
```

The SDK propagates baggage to all downstream services. The tail sampler's `debug-tag` policy keeps any trace carrying that attribute. Build it into a feature flag or a debug query parameter that the on-call can toggle.

This also enables **per-customer sampling** — set `debug.force_sample=true` in middleware when the request carries a customer ID on a known watch list. The cost is targeted: only those customers' requests are kept at 100%; everyone else stays at baseline.

## Brownfield retrofit — turning tracing on safely

When a service has no tracing today, the wrong move is "enable 100% sampling and see what happens." That blows out costs immediately. The sequence:

1. **Start at 1% head, no tail.** Get OTel SDK wired, get spans flowing, see the storage cost. One week.
2. **Add the tail sampler with `error` + `latency` policies only.** Baseline rate still 1%. Now incidents are debuggable; cost is still controlled. Two weeks.
3. **Raise baseline to 5–10%** once you understand the cost curve. Add force-sample support for the on-call escape hatch.
4. **Per-service tuning.** Drop the rate on chatty internal services; raise it on user-facing ones.

Do not enable tracing on every service simultaneously. Pick one critical path (e.g., the checkout flow), instrument it end-to-end, prove value, then expand. A half-instrumented estate is worse than no tracing — traces stop at the first un-instrumented hop and the operator chases ghosts.

## Anti-patterns

- **100% sampling in production "for completeness."** Wrong default. Storage, query CPU, and bill all explode. Use tail sampling.
- **Random 1% baseline with no error capture.** Statistically you will never have the trace for the request you need to debug. Tail-sample errors at 100%.
- **Different samplers in different services on the same trace path.** A trace gets dropped at hop 2 because that service's SDK decided no, even though hop 1 said yes. Use `parentbased` everywhere.
- **Tail sampling on a single-replica collector with no buffering capacity.** OOM kills under load; you lose the traces you most wanted (the incident is exactly when traffic spikes).
- **No way to force-sample.** On-call cannot debug a specific user's request. Build the escape hatch.
- **Tracing health-check traffic.** Burns budget on signal that has no value. Drop at SDK or first collector hop.

## Verification questions

1. Is every service using `parentbased` as the outermost sampler so trace decisions are consistent across hops?
2. Is the OTel Collector running a `tail_sampling` processor that keeps 100% of errors and 100% of slow traces?
3. Is the collector deployed with load-balanced trace-ID-consistent routing so tail sampling actually sees full traces?
4. Is health-check / probe traffic dropped before it reaches the trace backend?
5. Is there a documented debug-tag mechanism for forcing per-request sampling during an incident?
6. Has each service's sampling rate been tuned to its tier, or is everyone using the same default?

## What this is not

This reference covers *trace sampling*. The broader instrumentation layer — what spans to create, what attributes to set, propagation conventions — lives in `observability-design.md`. SLO-derived burn-rate alerts that consume metrics (not traces) live in `slo-design-patterns.md` and `alert-design.md`. For log volume and cost control, which is a parallel problem on a different signal type, see `log-volume-and-cost-control.md`.
