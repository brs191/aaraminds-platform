# SLO Design Patterns — Defining What "Working" Means

## When to use this reference

Pull this up when a service needs SLOs written for the first time, when an existing SLO is firing alerts that nobody trusts, when leadership asks "what's our availability number" and the team has three different answers, or when planning an error-budget policy that actually changes engineering behavior. Also use it when a brownfield service has metrics but no SLOs — instrumentation without targets is just decoration.

## The framing — SLOs exist to make tradeoffs, not to feel good

An SLO is a contract between the team running the service and everyone downstream of it (users, other services, the business). Its job is to let you answer two questions without arguing:

1. **Is the service healthy enough right now?** — burn-rate alerts settle this in minutes.
2. **Is the service healthy enough to ship the next risky change?** — error budget consumption settles this in days.

If your SLO doesn't drive at least one of these decisions, it's vanity. Delete it or fix it.

## Choosing SLIs that matter

An SLI (Service Level Indicator) is the *measurement*; the SLO is the *target*. Pick SLIs from the user's perspective, not the system's. The user does not care that your CPU is at 40%; they care that their request returned the right answer within a tolerable time.

For a request-driven service (HTTP, gRPC), the canonical four:

| SLI | What it measures | Source in this stack | Common trap |
|---|---|---|---|
| **Availability** | Fraction of valid requests that complete without server error | Prometheus counter: `http_server_requests_seconds_count{status!~"5.."}` over total | Counting 4xx as failure — client errors are not your fault and dilute the signal |
| **Latency** | Fraction of valid requests faster than threshold | Histogram: `histogram_quantile(0.99, ...)` against budget | Using mean / p50; users feel the tail, not the average |
| **Correctness** | Fraction of responses that are right | Business-specific check (downstream invariants, reconciliation) | Skipping it because "hard to measure" — pick a proxy (e.g., downstream reconciliation pass rate) |
| **Freshness** (for async-driven projections) | Fraction of reads served from data younger than X | OTel gauge: event timestamp − projection commit timestamp | Measuring at the producer only; you want end-to-end lag including replication |

For an event-driven worker (consumer of Service Bus / Event Hubs), the canonical set shifts:

| SLI | Definition |
|---|---|
| **Throughput** | Messages processed per second meeting deadline |
| **End-to-end lag** | p99 of (process completion time − message enqueue time) |
| **Dead-letter rate** | Fraction of messages routed to DLQ in the window |
| **Idempotency violations** | Count of duplicate side-effects detected (should be 0; alert if > 0) |

Pick three to five SLIs per service. More than five and nobody remembers them; fewer than three and you can't distinguish "slow" from "wrong" from "down."

## Picking targets — by user-facing tier

There is no universal "99.9%." Targets follow the consequence of failure.

| Service tier | Availability target | Latency target (p99) | Why |
|---|---|---|---|
| **External user-facing API** (checkout, login) | 99.95% (~21 min/month) | 300–500 ms | Money on the line; humans waiting |
| **Internal user-facing API** (admin tools, ops dashboards) | 99.9% (~43 min/month) | 1 s | Humans waiting but no revenue impact |
| **Service-to-service synchronous API** | 99.95% (caller's budget is yours + theirs) | 100–300 ms | Latency adds across hops; budget compounds |
| **Async worker** | 99.5% throughput, p99 lag < 60 s | n/a | Retry absorbs short outages |
| **Batch / scheduled job** | 99% success per run | n/a | Re-run tomorrow is acceptable |

The compounding rule: if service A calls B and C synchronously, A's availability is at most `A_own × B × C`. If B and C are both 99.9%, A starts the day at 99.7% before its own bugs. Either budget for the multiplication, push the calls async, or harden the dependency.

Do not pick 99.99% unless you've staffed for it. Four nines is roughly 4 minutes/month of downtime. That demands multi-region active-active, sub-minute failover, chaos engineering as a habit, and an on-call rotation that responds in single-digit minutes. Most internal services should sit at 99.9% and stop apologizing.

## Error budgets — the part that changes behavior

The error budget is `100% − SLO target`. At 99.9% availability, the budget is 0.1% of requests — roughly 43 minutes/month at constant traffic. This is **not** a guideline; it is the fuel for an explicit policy.

Suggested policy (write this into the team's runbook, get the PM and EM to sign):

- **Budget healthy (< 50% consumed)**: ship freely. Risky changes (schema migrations, framework upgrades) allowed.
- **Budget tight (50–90% consumed)**: ship only changes with explicit rollback plans. Pause non-essential experiments.
- **Budget exhausted (> 100% consumed)**: freeze feature work. Reliability fixes only until next window. Postmortem required.

The freeze is the point. Without it, the SLO is theater. Engineering will keep shipping risky changes; the budget will stay underwater; on-call will burn out. The freeze converts reliability from a vague aspiration into a calendar event.

Window choice: **30-day rolling**. Calendar-month windows produce a discontinuity on day 1 that hides chronic burn. A 30-day rolling window means today's incidents matter tomorrow.

## Multi-window multi-burn-rate alerts — the standard pattern

Single-threshold alerts on error rate are a known anti-pattern: either they fire constantly on small spikes or they miss slow burns that exhaust the budget over hours. The Google SRE workbook formula — multi-window, multi-burn-rate — is the standard. Use it.

The math: a burn rate of N means you're burning budget N times faster than sustainable. At burn rate 1, the budget lasts exactly the SLO window (30 days). At burn rate 14.4, the budget exhausts in 2 hours.

The four alerts (for a 30-day window, 99.9% SLO):

| Burn rate | Short window | Long window | Severity | Time-to-exhaust |
|---|---|---|---|---|
| 14.4 | 5 min | 1 h | Page | 2 hours |
| 6 | 30 min | 6 h | Page | 5 hours |
| 3 | 2 h | 1 day | Ticket | 10 days |
| 1 | 6 h | 3 days | Ticket | 30 days |

Two windows per alert: the short window catches the spike; the long window suppresses transient blips. Both must exceed the burn rate threshold for the alert to fire. This kills most false positives without losing real signal.

PromQL skeleton for a 14.4× burn-rate alert on a Spring Boot service exposing the Micrometer Prometheus registry:

```promql
(
  sum(rate(http_server_requests_seconds_count{service="order-svc",status=~"5.."}[5m]))
  /
  sum(rate(http_server_requests_seconds_count{service="order-svc"}[5m]))
) > (14.4 * 0.001)
and
(
  sum(rate(http_server_requests_seconds_count{service="order-svc",status=~"5.."}[1h]))
  /
  sum(rate(http_server_requests_seconds_count{service="order-svc"}[1h]))
) > (14.4 * 0.001)
```

The `0.001` is `1 − SLO` (i.e., the allowed error fraction). For a Go service exposing `otelhttp` metrics, the equivalent counter is `http_server_request_duration_seconds_count`.

Record these as recording rules to avoid repeating the math in every dashboard:

```yaml
groups:
- name: slo-order-svc
  interval: 30s
  rules:
  - record: slo:availability:order-svc:5m
    expr: |
      sum(rate(http_server_requests_seconds_count{service="order-svc",status!~"5.."}[5m]))
      /
      sum(rate(http_server_requests_seconds_count{service="order-svc"}[5m]))
  - record: slo:error_budget_burn:order-svc:1h
    expr: |
      (1 - slo:availability:order-svc:1h) / 0.001
```

Then alert on `slo:error_budget_burn:order-svc:1h > 14.4 and slo:error_budget_burn:order-svc:5m > 14.4`.

## Latency SLOs — pick the right percentile, define the threshold precisely

Latency SLI shape: "X% of requests complete within Y milliseconds." Not "mean latency below Y." Mean hides the bad tail; the tail is what users feel.

Threshold guidance:

- **p99** is the working percentile for user-facing APIs. Beyond p99 (p99.9, p99.99), individual outliers dominate; the signal becomes noisy.
- **p50** is useful as a *secondary* SLI to catch broad degradation, not the primary.
- Pick threshold from user research, not from "what we currently do." If the team is hitting p99 of 800 ms and the user research says 500 ms is the perception boundary, the SLO is 500 ms — and the burn rate tells you how far you are from the budget you actually owe.

PromQL with histogram buckets (requires `_bucket` series, which Spring Boot Micrometer emits when you enable `management.metrics.distribution.percentiles-histogram.http.server.requests=true`):

```promql
histogram_quantile(
  0.99,
  sum by (le, service) (rate(http_server_requests_seconds_bucket{service="order-svc"}[5m]))
)
```

Express the SLO as a *ratio*, not a raw percentile, so it composes with availability:

```promql
sum(rate(http_server_requests_seconds_bucket{service="order-svc",le="0.5"}[5m]))
/
sum(rate(http_server_requests_seconds_count{service="order-svc"}[5m]))
```

That gives you "fraction of requests under 500 ms" — feed it into the same burn-rate math as availability.

## Per-tier SLO patterns

**User-facing synchronous API (e.g., checkout):** Availability 99.95%, p99 latency < 400 ms, freshness n/a. Burn rate alerts at 14.4× / 6× page; 3× / 1× ticket.

**Internal API (e.g., admin tools):** Availability 99.9%, p99 < 1 s. Single page-tier burn alert at 14.4× suffices; lower urgency means ticket-tier alerts at 6× and below.

**Async worker consuming Service Bus:** Throughput SLO (e.g., 99% of messages processed within 60 s of enqueue), DLQ rate < 0.01% per day. Throughput failure pages; DLQ rate trends to a daily ticket.

**Read model / projection:** Freshness SLO (p99 staleness < 30 s end-to-end). Source: `current_time − last_committed_event_timestamp`. Brownfield retrofit pattern: instrument the projection commit path first, then the source emit path, then compute the difference downstream in Grafana — don't try to instrument the gap directly.

## SLOs for brownfield services — the instrumentation gap

When the service predates the SLO discipline, you cannot wait for "perfect" instrumentation before setting one. The sequence:

1. **Find the existing signal.** Almost every service already emits something — access logs, App Service request logs, an LB metric. Define a provisional SLI from what exists, even if imperfect.
2. **Set a target slightly below current performance.** If the service has been running at ~99.85% by the provisional measure, set the SLO at 99.8% for the first quarter. The goal is to start the budget cycle, not to hit ambitious numbers immediately.
3. **Refactor the SLI as instrumentation improves.** When OTel is wired in and you have proper histograms, switch the SLI; reset the budget. Document the cutover.
4. **Resist the urge to backfill history.** SLOs are forward-looking. Don't claim "we've been at 99.9% all year" based on data you didn't have. The budget starts today.

## Anti-patterns

- **SLOs without an error-budget policy.** Numbers on a dashboard with no consequence. Engineering won't change behavior.
- **SLO on system metrics (CPU, memory).** These are not user-visible. Alert on them as *cause* signals, not as SLOs.
- **One SLO per endpoint.** Death by configuration. Roll up by service or by user journey; alert at the journey level.
- **99.99% by default.** You will not staff for it. Pick targets you can actually defend.
- **Setting targets from peer envy** ("competitor claims 99.999%"). Marketing claims are not SLOs.
- **No freshness SLO on CQRS read models.** The whole point of the projection is to be current enough; if you don't measure staleness, you don't know if the pattern is working.

## Verification questions

1. Does every customer-visible service have at least three SLIs covering availability, latency, and either correctness or freshness?
2. Is there a written error-budget policy with explicit freeze conditions, signed off by EM and PM?
3. Are burn-rate alerts using the multi-window multi-burn-rate pattern, not single-threshold error rate?
4. Are SLI calculations encoded as Prometheus recording rules, so dashboards and alerts use the same math?
5. For event-driven workers: is end-to-end lag instrumented as the difference between enqueue and commit timestamps?
6. Has the SLO window been chosen as a rolling window, not a calendar month?

## What this is not

This reference covers *defining* SLOs and the burn-rate alerts that derive directly from them. For the broader alert taxonomy (cause alerts, dependency alerts, capacity alerts that are not SLO-derived), see `alert-design.md`. For the underlying metric instrumentation patterns (histograms vs gauges, cardinality budgets, labeling conventions), see `observability-design.md`. For how SLOs feed into incident response runbooks, see the resilience skill rather than this reference.
