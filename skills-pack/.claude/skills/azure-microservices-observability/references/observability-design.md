# Skill — Microservices Observability Design

## Purpose

Design observability (logging, metrics, tracing) for microservices systems so that failures can be diagnosed and performance can be understood. This skill ensures that when something goes wrong in production, you have enough signal to find the root cause without redeploying code or enabling debug logging. Use this after you have designed resilience, data patterns, and APIs.

## The Three Pillars of Observability

### 1. Distributed Tracing

**What:** Track a request as it flows through all services.

**Example:**
```
User requests: GET /orders/order-123

Trace captures:
  API Gateway → Order Service (50ms)
    ├─ Query database (10ms)
    ├─ Call Inventory Service (30ms)
    │  └─ Query cache (2ms)
    ├─ Call Payment Service (5ms)
    │  └─ Call external payment processor (3ms)
  → Send response (5ms)

Total latency: 50ms
Slowest child: Inventory Service (30ms)
```

**What to instrument:**
- Every RPC/HTTP call to another service (measure latency, errors)
- Every database query (measure latency, row count)
- External API calls (measure latency, errors)
- Cache hits/misses (identify cache effectiveness)

**Azure implementation:**
- Application Insights: automatic tracing for .NET, instrumentation for others
- OpenTelemetry: vendor-neutral, instrument once, send to Application Insights or other backend

**Sampling:** In production, sample traces (e.g., 10% of requests). Full tracing is too expensive.

### 2. Metrics (Time-Series Data)

**What:** Quantifiable measurements over time.

**Examples:**
```
Request latency (histogram):
  P50: 20ms
  P95: 150ms
  P99: 500ms

Request rate (counter):
  100 requests/second

Error rate (gauge):
  1% of requests fail (5xx errors)

Service health (gauge):
  Circuit breaker state: OPEN or CLOSED
  Queue depth: 42 messages waiting
```

**What to instrument:**
- Request latency (per endpoint, per service)
- Request rate (per endpoint, per status code)
- Error rate (per endpoint, error type)
- Business metrics (orders placed/min, payments processed/min)
- Resource metrics (CPU, memory, disk, connections)
- Resilience metrics (circuit breaker trips, retries, timeouts)

**Azure implementation:**
- Application Insights: automatic for .NET, Azure Monitor for infrastructure
- Custom metrics: emit via SDK

### 3. Structured Logging

**What:** Human-readable logs with structured context.

**Bad (unstructured):**
```
2026-05-18 10:23:45 Order placed: order-123
2026-05-18 10:23:46 Charging payment
2026-05-18 10:23:47 Payment successful
```

**Good (structured):**
```json
{
  "timestamp": "2026-05-18T10:23:45Z",
  "level": "INFO",
  "service": "order-service",
  "traceId": "trace-abc123",
  "event": "order_placed",
  "orderId": "order-123",
  "customerId": "cust-456",
  "items": 3,
  "total": 99.99
}

{
  "timestamp": "2026-05-18T10:23:46Z",
  "level": "INFO",
  "service": "payment-service",
  "traceId": "trace-abc123",
  "event": "payment_authorized",
  "paymentId": "pay-789",
  "orderId": "order-123",
  "amount": 99.99,
  "processor": "stripe",
  "latency_ms": 450
}
```

**What to log:**
- Service boundary crossings (entering a service, calling another service)
- State changes (order status: created → paid → shipped)
- Decisions (why did we choose this action?)
- Errors (what failed, why, context)
- External calls (to payment processor, to database, to cache)

**Log levels:**
- ERROR: Something went wrong and the user is affected
- WARN: Something unexpected but recoverable
- INFO: Normal flow (important decisions, state changes)
- DEBUG: Fine-grained (every step of algorithm) — disabled in production

**Azure implementation:**
- Application Insights / Log Analytics: SDKs emit structured logs
- slog (Go 1.21+): structured logging with JSON output

## Alert Strategy

**What to alert on:**

| Metric | Threshold | Action |
|---|---|---|
| Error rate | >5% | Page oncall immediately |
| P95 latency | >1s | Page oncall (slowness is degradation) |
| Circuit breaker | OPEN | Alert (service is failing) |
| Queue depth | >1000 messages | Alert (processing lag) |
| Disk space | <10% free | Alert (risk of full disk) |
| Payment failure | Any | Alert (revenue at risk) |

**Alert fatigue:** Too many alerts = oncall ignores all of them. Only alert on actionable problems.

**Alert routing:**
- Critical (payment failures, data loss): page oncall immediately
- Important (high error rate, performance degradation): email, create ticket
- Informational (expected spikes): log only, no alert

## SLO and SLI Definition

**SLO (Service Level Objective):** The goal you set. Example: "99.9% availability."

**SLI (Service Level Indicator):** What you actually measure. Example: "Percentage of requests that return 2xx status in <200ms."

**Example SLO definition for Order Service:**
```
Availability SLO: 99.9% (max 43 minutes downtime/month)
  SLI: Percentage of POST /orders requests returning 2xx or 3xx
  How measured: Application Insights, scoped to production
  Alert if:     Error rate > 5% (drops below 95% availability)

Latency SLO: P99 < 500ms
  SLI: 99th percentile of POST /orders duration (ingress to response)
  How measured: Application Insights latency histogram
  Alert if:     P99 > 1s (bad user experience)

Durability SLO: 99.99% (orders don't disappear)
  SLI: Percentage of orders successfully persisted
  How measured: Audit trail of created vs. persisted
  Alert if:     Any order lost (0 tolerance)
```

**How to use:**
- Monitor against SLOs daily
- If you drop below SLO, declare an incident and postmortem
- If you're well above SLO (e.g., 99.99% when goal is 99.9%), you can optimize (reduce costs)

## Worked Example — Order Service Observability

**Instrumentation:**

```
On order creation:
  Log {event: order_created, orderId, customerId, items: count, total}
  Increment metric: orders_created_total
  Trace: operation=create_order

On calling Payment service:
  Trace child span: call_payment_service
  Measure latency: payment_service_latency_ms
  Count errors: payment_service_errors_total
  Log {event: payment_initiated, orderId, amount, processor}

On database insert:
  Trace child span: database_write
  Measure latency: database_latency_ms
  Log {event: order_persisted, orderId, duration_ms}

On error:
  Log {level: error, event: order_creation_failed, orderId, reason, trace}
  Increment metric: order_creation_errors_total
  Alert if error rate > 5%
```

**Dashboards:**

**"Order Service Health" dashboard:**
- Orders created (rate graph)
- Error rate (% of requests failing)
- P50, P95, P99 latencies
- Circuit breaker state (Payment, Inventory)
- Payment service latency (dependency health)
- Queue depth (outbox backlog)

**"Order System End-to-End" dashboard:**
- Orders created → orders paid → orders fulfilled (flow chart)
- Where are failures happening? (which service has high error rate)
- Latency bottleneck? (which step is slow)

**Alerts:**

| Alert | Condition | Severity |
|---|---|---|
| HighErrorRate | order_creation_errors > 5% | Critical |
| PaymentServiceDown | payment_service_circuit=OPEN | Critical |
| SlowOrderCreation | order_creation_latency_p95 > 1s | Warning |
| OrderBacklog | outbox_queue_depth > 5000 | Warning |

## Verification Questions

1. **Tracing:** Can you follow a request from entry to exit, seeing all service calls and their latencies?

2. **Metrics:** For the top 5 failure modes (timeout, circuit breaker trip, database error, etc.), do you have a metric that detects it?

3. **Logging:** Can you reconstruct what happened during an incident from logs? (Service order, decisions, errors)

4. **Alerts:** Can you respond to every alert with a clear action? (If not, remove the alert.)

5. **SLOs:** What are your availability and latency SLOs? Are they enforced and monitored?

## What to read next

- For resilience metrics: `../../microservices-resilience/references/resilience-patterns.md`
- For Azure-specific observability: `../../azure-service-mapping/references/azure-mapping.md`
- For security auditing: `../../azure-microservices-security/references/security-design.md`
