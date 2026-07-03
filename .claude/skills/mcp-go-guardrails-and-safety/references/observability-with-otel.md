# Observability — OpenTelemetry → Langfuse or Phoenix

## Purpose

Runtime guardrails block what they recognize; CI evals catch regressions on known cases; **observability is where you see the unknown** — novel injection attempts that slipped through, latency spikes, the user behavior you didn't model. For Go MCP servers, the pack default is OpenTelemetry from the server, exported to Langfuse (or Phoenix as alternative). This reference covers the OTel SDK setup, span attribute design for tool calls, the Langfuse vs Phoenix decision, and the rule against logging raw args.

## Why OTel and not a custom logger

You already have `slog` for the audit log (`patterns/structured-audit-log.md`). Why also OTel?

- **Distributed traces** — when the MCP server calls Azure AI Content Safety, calls a DB, calls another service, OTel ties the spans together; logs don't.
- **LLM-aware sinks** — Langfuse and Phoenix understand "tool call" semantics; they show latency distributions, error rates, and trace flow visualizations that aren't free in vanilla logging.
- **OTel is the lingua franca** — Azure Monitor, Application Insights, Grafana, Jaeger, Honeycomb, Tempo all accept OTel; switching sinks doesn't change the server code.

Use both: slog for the audit log (compliance, SOC 2), OTel for the operational trace (latency, errors, flow). They overlap in some fields (tool name, outcome); that's fine — different consumers.

## SDK setup

```go
package telemetry

import (
    "context"
    "fmt"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func InitTracer(ctx context.Context, serviceName, version string) (func(context.Context) error, error) {
    exp, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
        otlptracehttp.WithHeaders(map[string]string{
            "Authorization": "Bearer " + os.Getenv("OTEL_EXPORTER_OTLP_HEADERS_AUTH"),
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("otlp exporter: %w", err)
    }

    res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName(serviceName),
        semconv.ServiceVersion(version),
        attribute.String("deployment.environment", os.Getenv("ENV")),
    ))

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exp),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )

    otel.SetTracerProvider(tp)
    return tp.Shutdown, nil
}
```

For stdio MCP: the OTel exporter must not write to stdout (stdio is the protocol wire — see `mcp-go-server-building`). The HTTP exporter is fine; gRPC exporter is fine; the stdout exporter is forbidden.

## Tool-call span design

Every tool call gets a span. Attributes — the schema matters because Langfuse / Phoenix will display these in dashboards and use them for filtering.

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("mcp-server")

func Trace() Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            ctx, span := tracer.Start(ctx, "tool.call",
                trace.WithSpanKind(trace.SpanKindServer),
                trace.WithAttributes(
                    attribute.String("tool.name", req.Params.Name),
                    attribute.String("tool.transport", "stdio"),
                ),
            )
            defer span.End()

            // Hash, don't log, the args. hashArgs is the shared helper from
            // runtime-guardrails-go.md.
            argsHash := hashArgs(req.Params.Arguments)
            span.SetAttributes(attribute.String("tool.args_hash", argsHash))

            res, err := next(ctx, req)

            if err != nil {
                span.SetStatus(codes.Error, err.Error())
                span.SetAttributes(attribute.String("tool.outcome", "error"))
                return res, err
            }
            if res != nil && res.IsError {
                span.SetStatus(codes.Error, "tool returned error")
                span.SetAttributes(attribute.String("tool.outcome", "tool_error"))
            } else {
                span.SetAttributes(attribute.String("tool.outcome", "success"))
            }

            // Output size, not output content
            if res != nil {
                for _, c := range res.Content {
                    if tc, ok := c.(mcp.TextContent); ok {
                        span.SetAttributes(attribute.Int("tool.output_bytes", len(tc.Text)))
                        break
                    }
                }
            }

            return res, nil
        }
    }
}

// hashArgs is defined in runtime-guardrails-go.md as a shared helper.
// Keep one definition in the guardrails package; reference from middlewares.
```

### Attribute schema

| Attribute | Type | Notes |
|---|---|---|
| `tool.name` | string | Tool identifier |
| `tool.transport` | string | `stdio` or `http` |
| `tool.outcome` | string | `success` / `error` / `tool_error` / `rate_limited` / `rejected_validation` / `rejected_injection` |
| `tool.duration_ms` | int (auto via span duration) | Tool execution time |
| `tool.args_hash` | string | Short hash of args (dedup), NOT the args |
| `tool.output_bytes` | int | Output size |
| `principal.oid` | string | If HTTP transport with auth |
| `principal.upn` | string | If HTTP transport with auth |
| `mcp.protocol_version` | string | MCP protocol version negotiated |

## The rule against logging raw args

**Never put raw tool args in span attributes or any observability payload.** Args can contain:

- Secrets (API keys, tokens — even after server-side redaction, the original args before redaction are still in memory at trace time)
- PII (emails, names, identifiers)
- Customer data (file contents, query strings, URLs)

OTel sinks aren't compliance boundaries. Traces flow through collectors, sit in storage, get retained. Treat traces as if anyone in the org can read them — because in practice, they can.

What to log instead:

- **Hash** of args for dedup (`tool.args_hash`)
- **Length** of args for size profiling (`tool.args_bytes`)
- **Schema-validated shape**: e.g., `args.has_url=true`, `args.url_domain=example.com` — derived attributes that don't leak content

If you need the args for debugging a specific incident, the audit log (`patterns/structured-audit-log.md`) is the place — it's redacted and shipped to Log Analytics with controlled access. Not OTel.

## Langfuse — the pack default

[Langfuse](https://langfuse.com) is the OSS LLM observability platform. Self-host on Azure Container Apps + Postgres + Redis + ClickHouse (or use Langfuse Cloud).

**OTel endpoint caveat**: Langfuse's OTel ingest path has evolved across versions (HTTP `/api/public/otel` for v3, distinct endpoints for traces vs. spans in earlier versions). Confirm against current Langfuse self-host docs before configuring `OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_EXPORTER_OTLP_HEADERS`. The example below shows the v3 shape.

Why default to Langfuse:
- **OSS** with clear license; self-host is straightforward (Docker Compose for dev, Container Apps for prod)
- **OTel-compatible** ingestion endpoint
- **LLM-aware UI** — tool call traces visualized as conversation flows; latency dashboards by tool name
- **Prompt management** built-in (useful if your MCP server uses LLM-driven prompts internally)
- **Cost dashboards** for downstream LLM API calls

Self-host on Azure (sketch):

```hcl
resource "azurerm_container_app" "langfuse" {
  name                         = "langfuse-prod"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"

  template {
    container {
      name   = "langfuse-web"
      image  = "langfuse/langfuse:3"
      cpu    = 1.0
      memory = "2Gi"

      env { name = "DATABASE_URL" secret_name = "db-url" }
      env { name = "NEXTAUTH_SECRET" secret_name = "auth-secret" }
      env { name = "SALT" secret_name = "salt" }
      # ... other config
    }
  }

  ingress {
    external_enabled = true
    target_port      = 3000
    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
  }
}
```

Verify the current Langfuse self-host docs for the full env var set and supporting services (ClickHouse, Redis).

Configure the Go MCP server to export to it:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=https://langfuse-prod.region.azurecontainerapps.io/api/public/otel
OTEL_EXPORTER_OTLP_HEADERS_AUTH=<langfuse public key>:<langfuse secret key base64>
```

## Phoenix — alternative

[Phoenix (Arize)](https://docs.arize.com/phoenix) is the other OSS option. Strengths:

- Stronger eval integration — Phoenix can run evals on traced data in-place
- LLM tracing + dataset management in one tool
- Single-container deploy possible for dev

Pick Phoenix over Langfuse if:
- You want eval workflows tightly coupled with traces
- The Arize commercial platform is in your future plans

Otherwise default to Langfuse. Don't run both — pick one.

## What to alert on

Setup alerts (in Langfuse, Phoenix, or your standard monitoring stack reading the same data):

- `tool.outcome = rejected_injection` rate > N/hour → injection attempts are happening, investigate
- `tool.outcome = error` rate > 5% over 10 min → server health issue
- `tool.outcome = rate_limited` rate > expected → client is misbehaving or limit too tight
- p99 `tool.duration_ms` > target (per tool) → performance regression
- Per-tool error pattern (one specific tool failing) → tool-specific bug

## Sampling

For low-traffic MCP servers (< 100 req/min), `AlwaysSample()` is fine. For high-traffic HTTP MCP servers, sample at 1–10% with always-sample on errors:

```go
sdktrace.WithSampler(sdktrace.ParentBased(
    sdktrace.TraceIDRatioBased(0.05),  // 5% baseline
))
```

For always-sampling errors and rate-limited calls, use a custom sampler that checks span attributes — most OTel SDKs support this via a head-based sampling decorator.

## Worked example — brownfield: adding OTel to an existing stdio MCP server

Setup: existing Go stdio MCP server, no observability beyond stderr `log.Printf`. Audit log just added (see `patterns/structured-audit-log.md`). Going to production with 50 internal users.

Steps:

1. **Stand up Langfuse** on Container Apps (or use Langfuse Cloud free tier for the first month). Get the public/secret key pair.
2. **Add OTel SDK** to the server. `go get go.opentelemetry.io/otel/...`.
3. **Initialize tracer in `main`** — exporter to Langfuse OTLP endpoint. Verify with a test span.
4. **Add the Trace middleware** to the chain, positioned after rate limiting and before the handler.
5. **Define the attribute schema** as documented above. Lock it down — adding attributes ad hoc creates dashboard cardinality issues later.
6. **Verify traces appear in Langfuse.** Click through to a single trace; confirm `tool.name`, `tool.outcome`, `tool.duration_ms` are present; confirm no raw args.
7. **Set up dashboards**: latency by tool, error rate by tool, injection-rejection rate by hour.
8. **Set up alerts** on the patterns above.
9. **2-week observation** — read the traces. Find at least one thing you didn't expect (a tool called more than you thought, a latency outlier, a user pattern). Document it in the pack's `../../../FEEDBACK.md`.

Total elapsed: 1 week for setup + ongoing observation. The dashboards become your operational reality once they're populated.

## Anti-patterns

- **Raw args in span attributes.** PII / secrets leak into observability storage. Hash or summarize.
- **No sampling on high-traffic HTTP MCP.** Storage costs balloon. Sample baseline traffic; always-sample errors and rejects.
- **No alerts.** Dashboards nobody looks at. Without alerts, observability is decoration.
- **Custom attribute schema per service.** Different MCP servers in the same org use different `tool.outcome` values. Standardize across the org.
- **Logging to stdout in stdio MCP.** Stdio is the protocol wire. OTel must export over HTTP/gRPC, not stdout.
- **OTel as a substitute for the audit log.** OTel is for operational visibility; audit log is for compliance evidence. Both, not either.
- **Running Langfuse and Phoenix in parallel.** Two ingestion paths, two dashboards, eventual drift. Pick one.

## Verification questions

1. Is OTel initialized in `main` with a real exporter (Langfuse/Phoenix), not the no-op default?
2. Does the Trace middleware run on every tool call?
3. Is the span attribute schema documented and consistent (`tool.name`, `tool.outcome`, etc.)?
4. Are raw args absent from span attributes (hashes / lengths only)?
5. For stdio MCP: is the OTel exporter HTTP/gRPC, never stdout?
6. Are alerts configured for `rejected_injection`, error rate, and per-tool p99 latency?
7. Is the sampling strategy appropriate for traffic volume (full for low, ratio + error-priority for high)?

## What to read next

- `runtime-guardrails-go.md` — the middleware chain Trace lives in
- `patterns/structured-audit-log.md` — the sister observability path (compliance vs operations)
- `prompt-injection-defense.md` — what `tool.outcome=rejected_injection` traces correspond to
- `eval-and-ci.md` — CI evals complementing production observability
- `../azure-microservices-observability` — broader observability surface (services beyond MCP)
- `../mcp-go-server-building` — stdio stdout rule (do not violate from OTel)
