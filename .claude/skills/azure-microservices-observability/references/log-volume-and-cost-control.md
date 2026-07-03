# Log Volume and Cost Control

## When to use this reference

Pull this when the Log Analytics bill is rising faster than headcount, when a service is ingesting > 1 GB/day per replica without a clear reason, when reviewing a new service's logging setup before it goes to production, when an engineer proposes "log everything and we'll filter later," or when planning a Log Analytics workspace structure for a new estate. Also use it during incident postmortems when the cause turned out to be impossible to find in the noise.

## The single rule — logs answer "what happened in this incident," nothing else

Logs are not for telemetry (use metrics). Logs are not for request flow (use traces). Logs are for the human, after the alert, looking at a specific time window, asking what business decisions the service made and which ones failed. Every other use is overflow that costs money and obscures the actual signal.

If you cannot point at a log line and say "an operator would read this during an incident," it is noise. Drop it.

## Structured logging — non-negotiable

Free-text logs are unsearchable at scale. Every service in the pack emits JSON with a known schema. The minimum required fields:

```json
{
  "timestamp": "2026-05-21T10:23:45.123Z",
  "level": "INFO",
  "service": "order-svc",
  "version": "1.4.2",
  "env": "prod",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "event": "order_persisted",
  "order_id": "ord-789",
  "customer_id": "cus-456",
  "duration_ms": 42
}
```

Required fields: `timestamp`, `level`, `service`, `version`, `env`, `trace_id`, `event`, `duration_ms` where applicable. Without `trace_id` you cannot correlate logs to the trace; without `event` you cannot query for specific business decisions.

**Spring Boot 21+** — use Logback's `JsonEncoder` (built into `logstash-logback-encoder` or `logback-json-classic`), bound to slf4j MDC for trace context:

```xml
<appender name="JSON" class="ch.qos.logback.core.ConsoleAppender">
  <encoder class="net.logstash.logback.encoder.LogstashEncoder">
    <includeMdcKeyName>trace_id</includeMdcKeyName>
    <includeMdcKeyName>span_id</includeMdcKeyName>
    <customFields>{"service":"order-svc","version":"${app.version}","env":"${app.env}"}</customFields>
  </encoder>
</appender>
```

OTel's Logback bridge populates MDC with `trace_id` and `span_id` automatically — wire it via `LogbackAppender` from `io.opentelemetry.instrumentation:opentelemetry-logback-appender-1.0`.

**Go 1.25+** — use `log/slog` with a JSON handler:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}).WithAttrs([]slog.Attr{
    slog.String("service", "order-svc"),
    slog.String("version", buildVersion),
    slog.String("env", env),
}))

// Inject trace context per request
ctx := r.Context()
sc := trace.SpanContextFromContext(ctx)
logger.LogAttrs(ctx, slog.LevelInfo, "order_persisted",
    slog.String("trace_id", sc.TraceID().String()),
    slog.String("span_id", sc.SpanID().String()),
    slog.String("order_id", orderID),
    slog.Int64("duration_ms", elapsed.Milliseconds()),
)
```

Free-text logging (`log.Println("Order " + id + " saved")`) is banned. The reviewer should reject the PR.

## Log levels in production

Four levels. Each has a rule.

| Level | Production use | Example |
|---|---|---|
| **ERROR** | The user is affected, or an invariant was violated. Page-worthy or postmortem-worthy. | "Payment authorization failed after 3 retries" |
| **WARN** | Something unexpected but recovered. Worth grepping for during postmortems. | "Circuit breaker tripped on payment-svc; falling back" |
| **INFO** | A business decision was made. The operator will want to see it during incident triage. | "Order placed", "Refund initiated", "Subscription upgraded" |
| **DEBUG** | Internal control flow. Disabled in production. | "Entered validateOrder()" |

Production runs at **INFO**. DEBUG goes off, full stop. Engineers who want DEBUG can run a single pod with a higher log level via a config override; never the whole fleet.

The common failure: services run at DEBUG "temporarily" during an incident and never get dialed back. Set a CI/CD check that fails the deployment if `LOG_LEVEL=DEBUG` in a production manifest. Make it an alert if DEBUG-level lines are observed in prod for more than 1 hour.

## What NOT to log — the hard list

**Never.** No exceptions. Catch these in code review.

1. **Secrets, tokens, API keys.** Even at DEBUG, even in dev — engineers copy/paste dev logs into tickets. Use a redaction wrapper that scrubs known patterns (`Bearer\s+\S+`, `password=`, AWS key shapes, Azure connection strings).
2. **PII subject to GDPR / SOC 2.** Email addresses, full names, phone numbers, government IDs, payment instrument data. Log the *internal ID* (`customer_id`), not the email. If an engineer needs to find the customer, they join to the customer DB out-of-band.
3. **Full request bodies.** A 50 KB JSON payload landing in Log Analytics 1000× per second is bill-destroying and almost never useful. Log the request *summary* (endpoint, request_id, content_length), not the body.
4. **Full response bodies.** Same reason. Log the status and a small set of fields the operator actually needs.
5. **Stack traces on every WARN.** Stack traces are large. Reserve them for ERROR. Even on ERROR, deduplicate identical stacks within a 5-minute window.
6. **High-cardinality identifiers in the message text (vs structured fields).** "Processed order 789 for customer 456" → "order_processed" event with structured fields. Log message text should be the *event name*; identifiers go in fields.
7. **Health-check and probe traffic.** Liveness probes hit every pod every 10 s. Logging each is 8,640 lines/pod/day for zero value. Filter at the access-log layer.

**Sometimes, with care:**

- **Request payloads > 1 KB**: only if redacted, only at DEBUG, only sampled (e.g., 1% via slog handler middleware). Never in steady production state.
- **Full SQL queries with parameters**: pre-redact parameters; log the statement template plus param shape (`SELECT * FROM users WHERE id = ?` with `param_count: 1`).
- **External API responses**: log status and latency; the body only if you control PII boundary (often you don't).

## Log Analytics workspace tiers and pricing

Azure Log Analytics charges by ingested GB plus retention. The pack uses three tiers in combination.

| Tier | Ingest cost (approx, varies by region) | Retention cost | Query | Use for |
|---|---|---|---|---|
| **Analytics (Interactive)** | High (the default) | First 31 days free; ~$0.13/GB/month after | Full KQL, fast | Last 30 days of all production logs |
| **Basic Logs** | ~20% of Analytics tier | First 8 days free; charged after | Restricted KQL, slower, cost per query | High-volume, low-query-frequency data — full access logs, OTel debug logs |
| **Archive** | Storage only, ~$0.02/GB/month | Up to 7 years | Must be restored to query | Compliance retention beyond 90 days |

Lifecycle for a typical service's logs:

```
Ingest (Analytics tier)
  ↓ after 30 days
Move to Archive (via table-level lifecycle policy)
  ↓ at 12 months (or compliance window)
Delete
```

Configure on the table:

```kusto
.alter table AppLogs policy retention softdelete = 30d
.alter table AppLogs policy archive = 'enabled'
.alter table AppLogs policy retention { "SoftDeletePeriod": "365.00:00:00", "Recoverability": "Disabled" }
```

Or in Terraform on the workspace:

```hcl
resource "azurerm_log_analytics_workspace_table" "app_logs" {
  workspace_id        = azurerm_log_analytics_workspace.main.id
  name                = "AppLogs"
  plan                = "Analytics"
  retention_in_days   = 30
  total_retention_in_days = 365
}
```

Daily ingestion cap as a safety net — prevents a runaway log loop from costing five figures over a weekend:

```hcl
resource "azurerm_log_analytics_workspace" "main" {
  name                = "law-prod"
  daily_quota_gb      = 50
  reservation_capacity_in_gb_per_day = 100   # commitment tier discount
}
```

`daily_quota_gb` is a hard stop, not a budget warning. Pick it slightly above expected peak so legitimate spikes survive. Alert at 80% of quota so an operator can investigate before the cap kicks in (lost logs during the cap window).

## Dropping noise — collector vs ingestion

Two places to drop log volume: at the **OTel Collector** before it ever leaves the cluster, or at **Log Analytics ingestion** via transformation rules. Each has costs.

| Drop at | Cost | Reversibility | Use for |
|---|---|---|---|
| **OTel Collector** | Free (no bytes leave the cluster) | Logs are gone; no second chance | Known-noise: health probes, DEBUG in prod, high-volume duplicates |
| **Log Analytics ingestion transformation (DCR)** | Charged for ingest of all bytes; transform runs before storage commit | Transform can be updated; old data is what you kept | PII redaction, schema normalization, sampling |
| **Query-time filter** | Full ingest + storage cost; just filter the query | All data retained | Last resort; cheapest engineering effort but most expensive bill |

Default to dropping at the collector. The collector pattern:

```yaml
processors:
  filter/drop_healthchecks:
    error_mode: ignore
    logs:
      log_record:
        - 'attributes["http.route"] == "/healthz"'
        - 'attributes["http.route"] == "/readyz"'
        - 'attributes["http.route"] == "/metrics"'

  filter/drop_debug_in_prod:
    error_mode: ignore
    logs:
      log_record:
        - 'severity_text == "DEBUG" and resource.attributes["env"] == "prod"'

  transform/redact_secrets:
    log_statements:
      - context: log
        statements:
          - replace_pattern(body, "Bearer\\s+[\\w\\-\\.]+", "Bearer [REDACTED]")
          - replace_pattern(body, "password=\\S+", "password=[REDACTED]")

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [filter/drop_healthchecks, filter/drop_debug_in_prod, transform/redact_secrets, batch]
      exporters: [azuremonitor]
```

For PII / regulatory redaction that *must* run before bytes leave the boundary, do it in the SDK redactor *and* in the collector — defense in depth.

## KQL patterns for cost diagnosis

Find the noisiest emitters in the workspace:

```kql
AppLogs
| where TimeGenerated > ago(1d)
| summarize bytes_ingested_mb = sum(estimate_data_size(*)) / 1024.0 / 1024.0,
            line_count = count()
  by Service, EventName
| order by bytes_ingested_mb desc
| take 20
```

Find services emitting more than 1 GB/day per replica:

```kql
AppLogs
| where TimeGenerated > ago(1d)
| summarize bytes_per_replica_gb = sum(estimate_data_size(*)) / 1024.0 / 1024.0 / 1024.0
            / dcount(InstanceId)
  by Service
| where bytes_per_replica_gb > 1.0
| order by bytes_per_replica_gb desc
```

Find log lines that exceed the 8 KB per-line threshold (usually request/response bodies that escaped review):

```kql
AppLogs
| where TimeGenerated > ago(1h)
| extend line_size = estimate_data_size(*)
| where line_size > 8192
| summarize fat_lines = count(), avg_size = avg(line_size)
  by Service, EventName
| order by fat_lines desc
```

Run these weekly. Anything in the top 20 by volume that does not have a clear incident-debug purpose is a candidate for dropping at the collector.

## Brownfield retrofit — cutting an existing service's log bill

The standard case: a service has been emitting unstructured logs at DEBUG for two years; the bill is climbing; nobody wants to touch the logging.

1. **Profile what's there.** Run the KQL queries above. Identify the top 5 noisy events.
2. **For each noisy event, classify:** kept (operator uses it during incidents), demoted (move to DEBUG, drop in collector), or deleted (no value). Annotate in source control.
3. **Set the daily quota cap immediately** to current consumption × 1.2. Future regressions hit the cap, not the bill.
4. **Convert to structured emission incrementally.** Replace the noisiest call sites first; leave the long tail for later. Free-text logs work in the new pipeline — they just lack searchability.
5. **Enable archive tier for tables older than 30 days.** Often the largest single bill component is retention of old data nobody queries.
6. **Quarterly review:** re-run the KQL diagnostics. Log shape rots.

Do not try to migrate everything in one sprint. The goal is a slope-change in the bill, not perfection.

## Anti-patterns

- **Logging at DEBUG in production "temporarily."** Becomes permanent. CI gate prevents it.
- **Logging full request bodies.** Bill explosion + PII leak in one pattern.
- **Free-text concatenation in log messages.** Unsearchable; defeats every downstream tool.
- **Stack traces on every WARN.** Multiplies volume; obscures the real ERROR stacks.
- **No daily quota cap on the workspace.** One runaway loop equals a weekend's worth of budget.
- **Retention "in case we need it" with no archive tier.** 12-month Analytics-tier retention is roughly 12× the cost of archive.
- **Dropping logs only at query time.** Pays for full ingest + storage; the cheapest engineering effort, the most expensive bill.
- **Different log schemas per service.** Querying across services becomes impossible. Standardize on the field set; enforce in shared logging library.

## Verification questions

1. Is every service emitting structured JSON with at minimum `timestamp`, `level`, `service`, `env`, `trace_id`, `event`?
2. Is the production log level INFO, with a CI/CD check blocking DEBUG in prod manifests?
3. Are health-check and probe requests dropped before reaching the log backend?
4. Does the Log Analytics workspace have a `daily_quota_gb` cap set, with an 80% utilization alert?
5. Are tables transitioned to Archive tier after 30 days, with a defined retention end?
6. Has the team run the per-service volume KQL queries in the last quarter, and are the top noisy emitters justified?
7. Is there a PII/secret redaction layer both in the SDK and at the collector?

## What this is not

This reference covers *log content, volume, and ingestion cost*. The schema fields beyond the minimum, log correlation to traces, and the role of logs in the three-pillar instrumentation model live in `observability-design.md`. For the trace side of cost control, see `trace-sampling-strategies.md`. For SLOs computed from metrics (not logs), see `slo-design-patterns.md`. Audit-log discipline for compliance overlaps but is governed by `azure-microservices-security` — that skill defines what *must* be retained and for how long, regardless of cost.
