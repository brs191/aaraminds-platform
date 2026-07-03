# Skill — MCP-Go Observability

## Purpose

Instrument an MCP server so operators can answer "what is it doing, why is it slow, what broke" in under five minutes. Production MCP servers fail in distinctive ways — slow backend connectors, prompt-injection-driven loops, cross-tenant query patterns, exceeded rate limits — and generic observability misses the signals.

## Three pillars, with MCP-specific application

Logs, metrics, and traces. The standard model. What makes MCP servers different is what you instrument with each.

### Logs (slog)

Use `log/slog` (Go 1.21+ standard library). Configure JSON output to stderr by default. Structured, never printf-style.

```go
package telemetry

import (
	"context"
	"log/slog"
	"os"
)

func InitLogger(level slog.Level, service string, version string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	})
	logger := slog.New(handler).With(
		slog.String("service", service),
		slog.String("version", version),
	)
	slog.SetDefault(logger)
	return logger
}

// Request-scoped logger via context
type loggerCtxKey struct{}

func WithRequestLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

func Log(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
```

In handlers:

```go
logger := telemetry.Log(ctx).With(
	slog.String("tool", "get_cost_summary"),
	slog.String("request_id", reqID),
	slog.String("tenant", tenantID),
)
logger.Info("tool call started")
// ... work ...
logger.Info("tool call completed", slog.Int64("duration_ms", elapsed.Milliseconds()))
```

**Critical for stdio transport:** logs go to **stderr, never stdout.** Stdout is the MCP protocol wire. A single Println to stdout corrupts the protocol stream.

**Separate from audit.** Application logs are for debugging, can be lossy, can be sampled. Audit logs are for compliance, append-only, never sampled. Use different sinks. The pack's existing `internal/audit/` pattern that emits via stdout is illustrative; in production, audit goes to a dedicated audit sink (Azure Monitor Log Analytics, dedicated event hub, SIEM).

### Metrics (OpenTelemetry)

Use OpenTelemetry SDK. On Azure, export via the OpenTelemetry Azure Monitor exporter to land metrics in Application Insights.

Metrics to instrument from day one:

| Metric | Type | Labels | Why |
|---|---|---|---|
| `mcp_tool_calls_total` | counter | `tool`, `status` (success/failure/denied/timeout), `risk_tier` | Volume, error rate, denial rate per tool |
| `mcp_tool_call_duration_seconds` | histogram | `tool`, `status` | Latency distribution per tool |
| `mcp_authorization_decisions_total` | counter | `tool`, `decision` (allow/deny), `reason` | Authz behavior, denial reasons |
| `mcp_backend_call_duration_seconds` | histogram | `connector`, `operation`, `status` | Identify slow backends |
| `mcp_redaction_hits_total` | counter | `tool`, `pattern` | When redaction fires — sign of leak risk |
| `mcp_rate_limit_rejections_total` | counter | `tool`, `identity_type` | Rate-limit pressure |
| `mcp_active_sessions` | gauge | none | Concurrent client sessions (streamable HTTP) |

```go
import (
	"go.opentelemetry.io/otel/metric"
)

var (
	toolCallsTotal     metric.Int64Counter
	toolCallDuration   metric.Float64Histogram
	backendCallLatency metric.Float64Histogram
)

func InitMetrics(meter metric.Meter) error {
	var err error
	toolCallsTotal, err = meter.Int64Counter("mcp_tool_calls_total",
		metric.WithDescription("Total MCP tool calls"))
	if err != nil {
		return err
	}
	toolCallDuration, err = meter.Float64Histogram("mcp_tool_call_duration_seconds",
		metric.WithDescription("MCP tool call duration in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}
	// ... initialize others
	return nil
}
```

In a handler:

```go
start := time.Now()
defer func() {
	status := "success"
	if err != nil {
		status = "failure"
	}
	toolCallsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("tool", "get_cost_summary"),
		attribute.String("status", status),
		attribute.String("risk_tier", "medium"),
	))
	toolCallDuration.Record(ctx, time.Since(start).Seconds(), ...)
}()
```

### Traces (OpenTelemetry)

Distributed tracing matters most when a tool call hits multiple backends. Trace the entire request: ingress → tool handler → service layer → connector → backend.

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func GetCostSummary(ctx context.Context, input CostSummaryInput) (CostSummary, error) {
	ctx, span := otel.Tracer("mcp-server").Start(ctx, "GetCostSummary",
		trace.WithAttributes(
			attribute.String("tool", "get_cost_summary"),
			attribute.String("tenant", input.TenantID),
		),
	)
	defer span.End()

	if err := authz.Check(ctx, ...); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "authz denied")
		return CostSummary{}, err
	}
	// ... service call (which starts its own child span)
	out, err := svc.GetCostSummary(ctx, input)
	if err != nil {
		span.RecordError(err)
		return CostSummary{}, err
	}
	span.SetAttributes(attribute.Int("result.items", len(out.Items)))
	return out, nil
}
```

Trace propagation across the request: incoming streamable HTTP requests carry W3C traceparent headers. OpenTelemetry's HTTP middleware extracts them; you do not need custom propagation code.

## Azure-specific exporters

For Azure-hosted MCP servers, use the Azure Monitor OpenTelemetry exporter to send logs, metrics, and traces to Application Insights:

```go
import (
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	azexporter "github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	// Or the OTel Azure Monitor distro when stable
)
```

(Verify current exporter recommendation in `ecosystem-facts.md` — the Azure Monitor exporter for Go has been evolving rapidly. The pack's previous guidance referenced Application Insights generically; the current shape is OTel SDK → Azure Monitor exporter → Application Insights.)

Practical Azure setup:

- **Connection string in Key Vault**, loaded via Managed Identity at startup
- **Resource attribute identifies the service:** `service.name = mcp-server-finops`, `service.version = 1.2.3`, `deployment.environment = production`
- **Sampling configured at exporter** — full traces in dev, head sampling at 10% in production unless investigating an incident

## SLIs and SLOs for MCP servers

Service Level Indicators worth tracking, with starter SLOs (tune to your workload):

| SLI | Definition | Starter SLO |
|---|---|---|
| Availability | `success / total` tool calls (excluding `denied` which is not a server fault) | 99.5% over 30-day window |
| Tool latency (P95) | 95th-percentile end-to-end duration per tool | Per-tool (a `get_cost_summary` SLO is different from a `restart_service` SLO) |
| Backend latency (P99) | 99th-percentile connector call duration | < 2 seconds for read connectors, < 10 seconds for write connectors |
| Rate of denials | `denied / total` tool calls | < 1% in steady state (higher denial rate is signal of attack or misconfiguration) |
| Audit event delivery | Audit events written to sink within N seconds | 99.9% within 30 seconds |

SLO violations are alerts. SLI degradation patterns inform capacity decisions.

## What to instrument that generic frameworks miss

These are MCP-specific concerns that generic instrumentation misses:

**Tool-call sequences that indicate prompt injection.** Log the sequence of tool calls in a session; flag patterns where a read tool's output is followed immediately by a sensitive write tool with parameters that mirror the read output. Not a perfect heuristic but a useful anomaly signal.

**Cross-tenant call patterns.** An identity that normally calls tools for tenant A suddenly calls tools for tenant B. Even with proper tenant scope enforcement, the pattern itself is informative.

**Redaction hits.** When output redaction fires, that means a tool tried to return something that looked like a secret. Track these as a leading indicator. A spike in redaction hits is a sign that a recent change is creating leak risk.

**Approval requests vs grants.** Critical-tier tools should produce approval-request events and approval-decision events. Track the ratio. A drop in grants relative to requests is signal of either denied attacks or a workflow problem.

**Schema validation failures.** Tools that reject inputs because validation failed. A spike in validation failures is signal of either misuse, integration breakage, or schema drift.

## Health endpoints — not the same as MCP endpoint

For streamable HTTP deployments on Container Apps, App Service, or AKS, expose health endpoints distinct from the MCP endpoint:

```go
mux := http.NewServeMux()
mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
	// Liveness — is the process alive?
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})
mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
	// Readiness — can the server actually serve traffic?
	if !backendsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
})
mux.Handle("/mcp", mcpHandler)
```

The MCP `/mcp` endpoint is for MCP protocol traffic, not for liveness checks. Mixing them confuses orchestrators and produces false health signals.

## Common observability anti-patterns

**Logging everything at INFO.** Eventually everything is noise. Use DEBUG for verbose internal state, INFO for state changes worth correlating later, WARN for recoverable problems, ERROR for actual failures. Logger levels are not decorative.

**Logging secrets.** Even with redaction at the output layer, intermediate logs leak secrets when developers add `slog.Any("input", req)` to debug something. Use a structured logger and only log fields that are explicitly safe. Never log entire request objects.

**No correlation across services.** Tool handler logs say "calling Azure," Azure connector logs say "request started" — no shared request ID, no shared trace ID. Pass a request ID through context from the moment the request enters; include it in every log line and every backend call's headers.

**Sampling traces at the wrong rate.** 100% sampling in production is expensive (storage, query cost). 1% sampling means you can't diagnose individual customer reports because you don't have their trace. Start at 10% with sampling overrides for error traces (always sample errors) and for explicit debug headers (sample 100% when `x-debug-trace: true` is set).

**Metrics without labels.** A counter that says "tool calls = 14,237" tells you nothing. Labels (`tool`, `status`, `risk_tier`) are what make the metric actionable. But: too many labels (high cardinality) breaks the metrics backend. Don't include user IDs, request IDs, or unbounded strings as metric labels.

**Audit and application logs in the same sink.** Compliance review asks for "all tool calls in March." You hand them an Application Insights query that mixes debug noise with audit events. Worse, the audit events are sampled. Separate sinks from the start.

## What to read next

- For the security events that audit captures: `enterprise-security.md`
- For tool-design conventions that integrate with this instrumentation: `tool-design.md`
- For the project structure where `internal/telemetry/` lives: `project-structure.md`
