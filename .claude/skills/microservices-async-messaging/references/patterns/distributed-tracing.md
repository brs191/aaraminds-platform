# Pattern: Distributed Tracing

## Problem

When a user request fans out across 5+ services, debugging "why was this slow" or "why did this fail" with single-service logs is impossible. Each service logs its piece, but no one has the end-to-end view. Distributed tracing connects all the spans from a single logical request, showing the full path, timing, and failure points across services.

## Use When

- Requests traverse 3+ services
- On-call engineers must answer "where did this slow down" or "where did this fail"
- Performance regressions need attribution to a specific service or call
- Compliance or audit requires traceability of every business action

## Avoid When

- Single-service system with no inter-service calls
- Cost of trace storage is prohibitive (high-traffic systems must sample)
- Team lacks observability infrastructure (Application Insights, Jaeger, etc.)

## Azure Implementation

### Implementation Steps

1. Adopt OpenTelemetry (vendor-neutral) as the instrumentation standard
2. Auto-instrument SDKs (HTTP clients, DB drivers, messaging clients) — minimizes code changes
3. Propagate trace context across boundaries: HTTP headers (`traceparent`), messaging properties, async tasks
4. Configure exporters to Application Insights or OpenTelemetry Collector → backend
5. Sample wisely: 100% for errors, 10% for normal traffic, head-based sampling for predictability
6. Build trace-driven dashboards: span latency, span error rate, service dependency graph
7. Use trace ID in logs — when an error occurs, the trace ID joins logs to the trace

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Trace backend | Application Insights | Auto-collects spans, dependency tracking |
| Instrumentation | OpenTelemetry SDKs | Per-language (Go, .NET, Java, Python) |
| Mesh-level | Dapr / Istio | Sidecars emit spans automatically |
| Collector | OpenTelemetry Collector on AKS | Buffer and route to multiple backends |
| Log correlation | Application Insights | Logs joined by trace ID and span ID |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Debuggability | Strongly improved — full request path visible |
| Performance impact | Low (~1–2% overhead) with sampling |
| Storage cost | Significant at high traffic — sample appropriately |
| Instrumentation effort | Initial setup non-trivial; ongoing maintenance per language |
| Privacy | Spans may contain PII; redaction needed |

## Common Failure Modes

- **Broken context propagation** — Trace context dropped at a service boundary (async work, threadpool, new HTTP client); trace ends prematurely.
  - Detection: Traces always end at the same service; downstream services have orphan spans.
  - Prevention: Propagate `traceparent` through all boundaries; pass context.Context through Go calls.

- **Cardinality explosion in span attributes** — Adding user IDs or unique URLs as span attributes; backend storage explodes.
  - Detection: Application Insights ingestion costs spike; queries slow.
  - Prevention: Use attributes for low-cardinality classification (status, region); use logs for high-cardinality details.

- **Sampling drops critical traces** — Random 10% sampling drops traces of rare errors; can't debug them.
  - Detection: Errors reported with no trace available.
  - Prevention: Tail-based sampling (sample after seeing error); always-sample on errors.

- **PII in traces** — Email addresses, account numbers in span attributes; compliance violation.
  - Detection: Privacy audit finds PII in observability data.
  - Prevention: Sanitize span attributes at collector; never log raw PII; use IDs not names.

## Decision Signals

Adopt distributed tracing when:
- 3+ services on a single user request path
- On-call cannot answer "where did this fail" quickly
- Latency regressions take days to attribute

Skip when:
- Single-service or monolith — single-service traces are enough
- No observability backend in place yet (set that up first)

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Application Insights | Backend | Native span ingestion, dependency map, end-to-end view |
| OpenTelemetry Collector | Buffer / multi-export | Vendor-neutral, can send to multiple backends |
| Dapr | Auto-tracing | Sidecar emits spans for invoke/pub-sub |
| Istio | Mesh-level tracing | Spans for every cross-service call |

## Go Implementation Notes

OpenTelemetry Go:
```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("order-service")
ctx, span := tracer.Start(ctx, "CreateOrder")
defer span.End()
span.SetAttributes(attribute.String("order.id", orderID))
```
Pass `ctx` through every call — HTTP, DB, messaging — so context propagates automatically.

Use middleware to start a span per HTTP request; instrument DB driver (`go.opentelemetry.io/contrib/instrumentation/...`) for query spans.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends tracing when describing multi-service requests
- `detect_architecture_risks` — flags missing propagation, high-cardinality attributes, PII in spans
- `generate_tracing_setup` — drafts OpenTelemetry config for the described stack
- `analyze_trace_coverage` — scores tracing instrumentation across services

## Related Patterns

- **Structured Logging** — logs joined to traces by trace ID
- **Service Mesh** — provides tracing without app instrumentation
- **Saga** — saga ID flows through traces for multi-step debugging
- **Idempotent Consumer** — span shows duplicate detection events

## References

- Skill: `../../../azure-microservices-observability/references/observability-design.md` — distributed tracing fundamentals
- Pattern: `../../../azure-service-mapping/references/patterns/service-mesh.md` — mesh-level tracing
- Pattern: `../../../microservices-data-architecture/references/patterns/saga.md` — tracing required to debug saga failures
