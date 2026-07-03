---
name: azure-microservices-observability
description: Designs and reviews observability for Azure-hosted microservices using OpenTelemetry for instrumentation and Grafana plus Prometheus for visualization, covering distributed tracing, SLOs, metrics, structured logging, and alert design. Use when adding observability to a new service, reviewing whether an existing service is properly instrumented, defining SLOs, designing alerts that page on real problems, or diagnosing why an incident took too long to detect. Do not use for MCP-server-specific observability (use mcp-go-server-building, which has its own observability reference) or for general cloud monitoring questions outside microservices.
version: 1.1.0
last_updated: 2026-05-21
---

# Azure Microservices Observability

## When to use

Trigger this skill when the question is about visibility into a running microservices system: defining SLOs and SLIs for a service, instrumenting a new service with OpenTelemetry, reviewing whether existing instrumentation actually answers "what is it doing, why is it slow, what broke," designing alerts that page on real signal not noise, or working backward from "this incident took 40 minutes to detect — what was missing."

Do **not** use this skill for: MCP server observability (use `mcp-go-server-building` — its observability reference is MCP-specific); general Azure Monitor questions outside the microservices context; APM tool evaluation (the pack standardizes on Grafana + Prometheus + OTel).

## The critical decision rule — instrument for failure modes, not for completeness

The wrong question is "what should I measure." The right question is "if this fails, how will I know within 5 minutes and find the root cause within 30?" Every metric, span, and alert must trace back to a specific failure mode you are trying to detect or diagnose.

Generic instrumentation that measures everything produces dashboards no one reads and alerts no one trusts. Targeted instrumentation that maps directly to failure modes is the only kind that survives.

## The three pillars, applied

| Pillar | What it answers | Tool in this stack | Key discipline |
|---|---|---|---|
| Distributed traces | "Where in the call chain is the latency or error?" | OpenTelemetry SDK → Grafana Tempo or Azure Monitor exporter; viewed in Grafana | Sample in production (10% typical); preserve trace context across service boundaries via W3C `traceparent` |
| Metrics | "Is the system meeting its SLOs over time?" | OpenTelemetry SDK → Prometheus (pull) or via OTLP push; viewed in Grafana | Use histograms not gauges for latency; cardinality discipline (no per-request labels) |
| Logs | "What exactly happened during this incident?" | Structured JSON via slog (Go) / Logback JSON (Spring Boot) → Log Analytics or Loki; queried in Grafana | One log line per business decision; never log to stdout under stdio MCP transport |

For the full instrumentation framework — span attribute conventions, metric naming, log schema — see `references/observability-design.md`.

## The SLO framework

Every customer-facing service has at least three SLOs:

1. **Availability** — fraction of requests that complete without server error. Typical target: 99.9% (allows ~43 minutes downtime/month).
2. **Latency** — p99 of request latency below a stated threshold. Typical target: p99 < 500ms for synchronous APIs.
3. **Freshness** (for async-driven read models) — staleness of data the service serves. Typical target: p99 < 30 seconds end-to-end lag.

Each SLO drives an alert: page when error budget burn rate exceeds 14.4× (the rate at which you'd exhaust the monthly budget in 2 hours).

For SLO definition, error-budget math, and alert design, see `references/observability-design.md`.

## Worked example — brownfield: an existing Spring Boot service produces noisy alerts

Setup: a 2-year-old Spring Boot order service on AKS has 23 active alerts in Grafana. Most fire weekly. The on-call rotation ignores most of them; the few that page are inconsistent signal. Leadership asks why a recent outage took 35 minutes to detect when there are 23 alerts.

Decision walk:

1. **Audit the alert list.** Categorize each: paging vs. ticketing vs. info. Find what each was meant to detect.
2. **For each paging alert, ask: when this fires, what is the operator's first action?** If the answer is "check Grafana, then maybe escalate" — the alert is not actionable and should be ticketing-tier or deleted.
3. **Working from the recent outage backwards:** identify the failure mode (Cosmos DB cross-partition query exhausting RUs). What signal would have caught this in 5 minutes? RU consumption rate climbing past 80% of provisioned for 3 minutes. Was there an alert? No.
4. **Construct the right alert set.** For each SLO (availability, latency, freshness, plus dependency-health like the Cosmos RU signal), one paging alert with a clear runbook entry. Total: 6 paging alerts, not 23.
5. **Decommission the rest.** Convert some to tickets, delete the rest. Alert noise is a security problem (real alerts get ignored); cleanup is operational hygiene.
6. **Add a runbook entry per remaining alert.** "When this fires, do X." If you can't write that sentence, delete the alert.

References: `references/observability-design.md` (alert design, runbook discipline, SLO-driven burn-rate alerts).

## Anti-pattern — "log everything, find the needle later"

**Bad:** A service logs every method entry/exit at DEBUG, every variable, every HTTP request body. When an incident hits, an engineer greps through 80GB of logs in Log Analytics looking for the relevant 10 lines.

**Why it fails:** Three reasons. Cost: Log Analytics ingestion is expensive at scale (1+ TB/month adds real dollars). Signal loss: the relevant lines are buried; the engineer spends 25 of their 30 minutes searching instead of fixing. Discipline rot: nobody trims the logging because "we might need it" — so it grows forever.

**Detection signal:** the service's Log Analytics ingestion is >1 GB/day per service instance, or the log schema includes line-level call traces rather than business-decision events.

**Fix:** Log at INFO for business decisions ("Order ack received", "Payment failed: invalid CVV"), DEBUG only in development. Use OTel spans to capture per-request flow — traces are the right tool for "what did this request do," not logs. Sample DEBUG in production aggressively (1% if at all). Set ingestion budgets in Log Analytics workspace settings.

## Verification questions

1. For each customer-facing service: is there at least one paging alert per SLO (availability, latency, freshness)?
2. For every paging alert: is there a runbook entry telling the operator what to do?
3. Does every external call (HTTP, DB, message bus, Key Vault) produce an OTel span with latency and error attribution?
4. Can you reconstruct a single request's flow across all services from a trace ID in Grafana within 30 seconds?
5. Is log volume per service under 1 GB/day in steady state, or is there a documented reason it isn't?
6. For async-driven read models: is end-to-end lag (event timestamp → projection commit) measured and alerted on?

## What to read next

- `references/observability-design.md` — the three pillars with concrete instrumentation patterns, span/metric/log schema conventions
- `references/slo-design-patterns.md` — choosing SLIs, error-budget policy, multi-window multi-burn-rate alert math with PromQL
- `references/alert-design.md` — page vs ticket vs info, symptom-vs-cause discipline, runbook contract, brownfield alert-backlog cleanup
- `references/trace-sampling-strategies.md` — head-based vs tail-based sampling, OTel Collector tail-sampling config, per-service overrides, force-sample escape hatch
- `references/log-volume-and-cost-control.md` — structured logging schema, Log Analytics workspace tiers, what NOT to log, dropping noise at the collector vs at ingestion
- `microservices-async-messaging` skill — for lag instrumentation patterns on event-driven flows
- `microservices-resilience` skill — for resilience metrics that pair with the observability layer (circuit-breaker state, retry counts, bulkhead saturation)
- `azure-microservices-security` skill — for audit-log discipline that overlaps with operational logs
- `mcp-go-server-building` skill — for MCP-specific observability if the service in question is an MCP server (it differs in important ways)
