# Alert Design — Page-Worthy vs Noise

## When to use this reference

Reach for this when on-call is acknowledging alerts without acting, when leadership asks why an incident took 40 minutes to detect despite 30 active alerts, when a new service is about to go live and the team is copy-pasting Prometheus alerts from an old service, or when reviewing a brownfield service's alert backlog and trying to cut it down. Also use it when defining alert routing for a multi-service estate where some services page and others ticket.

## The single rule — every paging alert needs an immediate human action

If, when an alert fires, the on-call's first action is "look at Grafana and decide if it's real," the alert is wrong. It is either too noisy (drop the threshold or the alert), it lacks context (add fields and a runbook link), or it should be ticket-tier rather than page-tier.

Paging means waking someone up. Reserve it for situations where a human must act in minutes, with a known first move, against a clear target. Everything else is a ticket, a dashboard panel, or a metric on a trend chart.

## The severity ladder

Three tiers. More is overhead; fewer collapses signal.

| Tier | Trigger | Delivery | Response SLA | Volume target |
|---|---|---|---|---|
| **Page** | User-facing SLO is burning fast, or an irreversible action is imminent (data loss, runaway cost) | PagerDuty / Opsgenie phone + push | < 5 min ack, < 15 min mitigate | < 1 per service per week in steady state |
| **Ticket** | A symptom of degradation that does not need immediate human action — capacity headroom, slow burn, dependency wobble | GitHub issue / Jira via webhook | Next business day | < 5 per service per week |
| **Info / dashboard** | Operational signal — useful while investigating, not alertable | Dashboard panel only, optionally logged | n/a | unlimited |

A page that fires more than once a week with no incident attached has the wrong threshold or is the wrong alert. Audit it monthly; demote or delete.

## Alert on symptoms, not causes

The principle: alert on what the user feels; let causes show up in dashboards and traces when the operator drills in.

| Symptom (alert on this) | Cause (do NOT page on this) |
|---|---|
| SLO burn rate > 14.4× over 5m and 1h | CPU > 80% |
| End-to-end p99 latency > 1 s for 10 min | Garbage collection pause count up |
| DLQ growth > 100 messages in 15 min | Pod restart count > 0 |
| Outbox lag > 60 s | Disk I/O queue depth high |
| Cosmos 429 rate > 1% of requests | RU consumption at 70% |
| Service Bus subscriber lag > 30 s | Connection pool saturation > 50% |

Cause alerts are tempting because they're easy to write. They are also why nobody trusts the page-volume. The pattern: a single user-visible symptom can have ten causes; if you page on all ten, the operator drowns. Page on the symptom; surface the causes on a dashboard the operator opens *after* the symptom alert lands.

There are exceptions — three classes of cause alert that earn a page:

1. **Imminent capacity exhaustion of an unrecoverable resource.** Disk free space < 10% on a stateful volume. Database storage > 90% of provisioned. Quota approaching ceiling. These cannot wait for the symptom because by then it's too late.
2. **Detector for a known-silent failure mode.** A consumer group that has stopped acknowledging — no symptom because no traffic, but the queue is silently growing. Worker that has been processing zero messages for > 15 min during business hours.
3. **Saturation alarms on async-only paths.** A worker fleet with no synchronous SLO needs cause-tier alerts (queue depth, processing lag) because there is no symptom-tier signal until the user notices stale data.

## PromQL patterns — Spring Boot 21+ and Go 1.25+

The stack: Spring Boot services expose metrics via Micrometer's Prometheus registry on `/actuator/prometheus`. Go services expose via `otelhttp` instrumentation through the OTel Collector's Prometheus exporter, or directly via `promhttp.Handler()`. The metric *names* differ; the alert shapes are the same.

### Availability — burn-rate alert (use multi-window pattern)

Spring Boot (Micrometer naming):

```promql
# Recording rule — 5m error ratio
- record: app:http_error_ratio:5m
  expr: |
    sum by (service) (
      rate(http_server_requests_seconds_count{status=~"5.."}[5m])
    )
    /
    sum by (service) (
      rate(http_server_requests_seconds_count[5m])
    )

# Alert — 14.4x burn (page-tier)
- alert: HighErrorBurn
  expr: |
    app:http_error_ratio:5m > (14.4 * 0.001)
    and
    app:http_error_ratio:1h > (14.4 * 0.001)
  for: 2m
  labels:
    severity: page
  annotations:
    summary: "{{ $labels.service }} burning error budget at 14.4x"
    runbook: "https://wiki/runbooks/{{ $labels.service }}/high-error-burn"
```

Go with `otelhttp`:

```promql
- record: app:http_error_ratio:5m
  expr: |
    sum by (service_name) (
      rate(http_server_request_duration_seconds_count{http_response_status_code=~"5.."}[5m])
    )
    /
    sum by (service_name) (
      rate(http_server_request_duration_seconds_count[5m])
    )
```

### Latency — symptom alert tied to SLO threshold

```promql
- alert: SlowP99Latency
  expr: |
    histogram_quantile(
      0.99,
      sum by (le, service) (rate(http_server_requests_seconds_bucket[5m]))
    ) > 0.5
  for: 10m
  labels:
    severity: page
  annotations:
    summary: "{{ $labels.service }} p99 above 500ms for 10m"
```

The `for: 10m` clause matters: latency is bumpy. A single 5-minute spike during a deployment is not page-worthy; sustained degradation is.

### Saturation — alert before the cliff, not at the cliff

```promql
# Cosmos RU — page before you start getting 429s
- alert: CosmosRUSaturation
  expr: |
    max by (account, database, container) (
      avg_over_time(
        azure_documentdb_normalized_ru_consumption_percentage[5m]
      )
    ) > 80
  for: 10m
  labels:
    severity: page
  annotations:
    runbook: "https://wiki/runbooks/cosmos/ru-saturation"

# Service Bus subscription dead-letter growth
- alert: ServiceBusDLQGrowth
  expr: |
    increase(azure_servicebus_subscription_dead_letter_messages[15m]) > 100
  for: 5m
  labels:
    severity: page
```

### Liveness without traffic — the silent worker

```promql
# Worker pod is running but processing zero messages during business hours
- alert: WorkerSilent
  expr: |
    rate(messages_processed_total[15m]) == 0
    and on (service) up{job="worker"} == 1
    and hour() >= 7 and hour() < 22
  for: 15m
  labels:
    severity: page
  annotations:
    summary: "{{ $labels.service }} processed zero messages for 15m"
```

Without the `hour()` clamp this fires at 03:00 during a quiet window and pages an exhausted on-call for nothing.

## Anti-pattern alerts — delete these on sight

The pack's brownfield audits surface the same noisy alerts repeatedly. Cut them.

| Anti-pattern alert | Why it's noise | What to do instead |
|---|---|---|
| `CPU > 80% for 5m` | CPU is not user-visible. Modern services should run hot. | Delete. Track CPU on a capacity dashboard, alert only at sustained > 95% for autoscaling triggers. |
| `Memory > 90%` | JVM and Go runtimes use whatever they're given. | Delete. Alert on OOMKilled events or restart count, not on % used. |
| `Pod restart count > 0` | Restarts during rolling deploys are normal. | Alert on restart count > 3 in 30m, scoped to outside deploy windows. |
| `Any 5xx in last 5m` | A single 5xx during a deploy is meaningless. | Use burn-rate alerts. |
| `Disk I/O latency > 50 ms` | Azure managed disks have variable latency. | Delete unless tied to a user-visible symptom. |
| `Active connection count > N` | Without knowing pool size, the number is meaningless. | Alert on pool saturation ratio (used / max) > 80%. |
| `GC pause > 100 ms` | One pause is fine. | Alert on accumulated GC time as % of wall clock > 10% over 15m. |
| `Latency > average + 2σ` | Statistical alerts on noisy signals page constantly. | Use fixed thresholds tied to SLOs. |
| `Test environment alerts paging prod on-call` | Wrong routing. | Route by environment label; non-prod never pages. |

## Runbooks — the price of admission

Every paging alert ships with a runbook entry. The entry has four sections, fits on one screen, and is linked from the alert annotation.

1. **What this means** — one sentence: "the order service is returning 5xx faster than its monthly budget allows."
2. **First action** — the single most likely fix or diagnostic. "Check Grafana 'Order Service Health'; if Payment dependency is red, see Payment runbook. If RU saturation is the cause, raise the Cosmos autoscale ceiling per `cosmos-saturation.md`."
3. **Escalation** — who to call if first action doesn't work and time elapsed since alert > X.
4. **How to silence** — link to the silence command. Operators silence during planned maintenance; without an explicit "how to silence" step they either won't, or they'll silence wrong.

If you cannot write the four sections, the alert is not page-worthy. Make it a ticket.

## Alert ownership and routing

Every alert has exactly one owning team. The label is mandatory. Routing maps `team` → notification channel:

```yaml
route:
  receiver: default-ticket
  routes:
    - matchers: [severity="page", team="orders"]
      receiver: orders-pagerduty
    - matchers: [severity="page", team="payments"]
      receiver: payments-pagerduty
    - matchers: [severity="ticket"]
      receiver: github-issues
    - matchers: [environment!="prod"]
      receiver: dev-slack
```

Routes are explicit. No service should fall through to a default that pages everyone — that turns the central on-call into the alert graveyard.

## Inhibitions and grouping — kill the alert storm

When a dependency fails, every dependent service alerts. Inhibitions express "if A is firing, do not fire B":

```yaml
inhibit_rules:
  - source_matchers: [alertname="CosmosOutage", severity="page"]
    target_matchers: [alertname="HighErrorBurn", severity="page"]
    equal: [region]
```

Grouping bundles related alerts into a single notification: `group_by: [service, alertname]`, `group_wait: 30s`. A burst of 50 pod restart alerts during a rolling deploy becomes one notification instead of fifty.

## Brownfield retrofit — cutting an existing alert backlog

The 23-alert service is the standard brownfield case. The process:

1. **List every alert** with its last-fired timestamp and total fire count over 90 days. Use the Prometheus / Alertmanager API; do not eyeball Grafana.
2. **Bucket** into: never fired (delete), fires regularly but no incident attached (delete or demote), fires regularly with incident response (keep), never fired and protects against a documented failure mode (keep).
3. **For every kept paging alert**, write or update the runbook. If you can't write it, demote to ticket.
4. **For every dropped alert**, leave a one-line comment in source control: "Dropped 2026-05-21: fired 47× in 90 days, zero incident actions."
5. **Add a quarterly review** in the team's calendar. Alert hygiene rots otherwise.

Target after cleanup: 4–8 paging alerts per service. More than 10 in steady state and you have not finished the cleanup.

## Anti-patterns

- **Copy-paste alerts from one service to the next.** Each service has different SLOs and failure modes. Generic alerts fit nobody well.
- **Alerts without owners.** Default-routed alerts get ignored by everyone.
- **Per-instance alerts in autoscaled fleets.** One pod misbehaving on a fleet of 30 is normal. Aggregate by service, then alert.
- **Mean-based latency alerts.** Means hide tails. Always percentile-based.
- **Alerts that fire on deploys.** Either add `for:` clauses long enough to absorb the deploy, or label deploys and inhibit during them.
- **No alert testing.** When was the last time the team actually tested that the page-tier delivery works? Quarterly fire drill is non-negotiable.

## Verification questions

1. Does every paging alert have a labeled owning team, a runbook URL annotation, and a documented first action?
2. In steady state (no active incident), does each service generate fewer than one page per week?
3. Are CPU / memory / GC pause / restart alerts demoted to capacity dashboards rather than firing pages?
4. For brownfield services: when was the last alert audit, and how many alerts were dropped or demoted?
5. Are deploys labeled so deploy-window alerts can be inhibited?
6. Is there a quarterly fire-drill that exercises the paging path end-to-end?

## What this is not

This reference covers the *taxonomy and design* of alerts. The SLO-derived burn-rate math that drives the most important paging alerts lives in `slo-design-patterns.md` — read that first if you don't already have SLOs defined. For the underlying metric instrumentation (histograms, cardinality discipline, label conventions), see `observability-design.md`. For how alerts feed into incident response and on-call rotation, see the resilience and security skills rather than this reference.
