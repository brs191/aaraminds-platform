---
name: azure-microservices-cost-review
description: Reviews and optimizes the cost posture of an Azure microservices architecture, covering compute (Container Apps, AKS, App Service), data (Azure SQL, Cosmos DB, Redis), messaging (Service Bus, Event Grid, Event Hubs), and idle-resource detection. Use when reviewing a monthly Azure bill, sizing infrastructure for a new service, deciding between scale-to-zero and reserved instances, or producing a FinOps recommendation. Do not use for choosing which Azure service to use functionally (use azure-service-mapping) or for cost discussions on AWS/GCP (out of scope).
version: 1.0.0
last_updated: 2026-05-18
---

# Azure Microservices Cost Review

## When to use

Trigger this skill when the question is about Azure spend: reviewing the monthly bill, sizing a new service, choosing a pricing model (consumption vs. provisioned vs. reserved), evaluating whether autoscaling is delivering what it promised, or producing a FinOps recommendation that goes to engineering leadership.

Do **not** use this skill for: choosing *which* Azure service to use (use `azure-service-mapping`); cost discussions on AWS or GCP (out of scope per pack context); developer-laptop dev/test cost questions (different tradeoff profile than production).

## The critical decision rule — measure before you optimize

Most cost recommendations are wrong because the recommender never looked at the actual usage profile. Before proposing any change, get the data:

- **Cost Management exports** to a Storage account, aggregated by resource group and tag for the last 30 days minimum
- **Resource-level utilization** — vCPU, memory, request count, IOPS — pulled from Azure Monitor / Log Analytics
- **Idle/saturation signals** — Container Apps replicas idling at <10% vCPU, Cosmos DB containers at <30% RU consumption, Service Bus topics with growing backlog

A recommendation like "switch to reserved instances" without 30 days of utilization data is a guess, not a recommendation. Lead with the data, then with the recommendation.

## The cost-review framework

| Cost lever | What to measure | Common wins |
|---|---|---|
| Compute right-sizing | vCPU and memory utilization p95 over 7 days | Container Apps at 2 vCPU running steadily at 0.4 → drop to 0.5 vCPU |
| Scale-to-zero | Idle hours per day for non-critical workloads | Background workers idle 16 h/day on Container Apps → enable scale-to-zero, ~66% savings |
| Reserved instances | Steady-state vCPU-hours per month for >12 months horizon | 1-year RI on AKS node pool: 20-30% off pay-as-you-go |
| Cosmos DB throughput | RU consumption p95 vs. provisioned RU/s | Autoscale max with consumption averaging <50% → drop the max; consider serverless for spiky |
| Storage tiers | Access frequency on blob containers | Move cold blobs to Cool / Archive tier; lifecycle policies automate it |
| Egress | Cross-region and cross-cloud egress GB/month | Collocate dependent services in the same region; cache aggressively |
| Idle databases | Connection count + read/write IOPS over 7 days | An Azure SQL DB at 0 connections for 2 weeks is paying rent for nothing — pause or drop |
| Log retention | Log Analytics workspace ingestion GB/day, retention days | Move audit logs to cheap Storage after 30 days in Log Analytics |

See `references/cost-and-tradeoffs.md` for the full cost model with per-service formulas and Azure pricing references.

## Worked example — brownfield: reviewing a quarterly bill spike

Setup: monthly Azure spend jumped 35% quarter-over-quarter for a 12-service microservices estate on Container Apps with Cosmos DB and Service Bus. No major feature launches in the period. Leadership asks: where did the money go?

Decision walk:

1. **Pull Cost Management data by resource group.** Two RGs account for 80% of the increase: `rg-order-prod` and `rg-analytics-prod`.
2. **Drill into `rg-order-prod`.** Container Apps spend is flat; Cosmos DB doubled. Open the Cosmos DB cost breakdown. RU consumption is up 2.3x; provisioned throughput was bumped from 4000 to 10000 RU/s in week 3 of the quarter.
3. **Why the RU bump?** Check Application Insights for query patterns. A new dashboard query is doing cross-partition scans on every page load. The fix is a query-shape change, not a throughput bump. Roll throughput back, fix the query.
4. **Drill into `rg-analytics-prod`.** Log Analytics ingestion is up 4x. Trace it to a new service that started logging at DEBUG in production. Move to INFO, set sampling, savings appear immediately.
5. **Propose two-week cleanup plan.** Cosmos throughput rollback (after query fix) + Log Analytics ingestion fix should recover most of the increase. Quantify expected savings before doing the work.

References: `references/cost-and-tradeoffs.md` (Cosmos and Log Analytics cost models, query-pattern detection).

## Anti-pattern — recommending reserved instances without utilization data

**Bad:** A FinOps reviewer sees high Container Apps spend and recommends switching to 1-year reserved instances "for 20-30% savings."

**Why it fails:** The workload may be spiky or seasonal. Reserved instances lock in a baseline commitment; if the actual steady-state vCPU is below the RI commitment, you've paid for capacity you don't use. Reserved instances are correct only when the workload's *minimum* sustained capacity exceeds the RI commitment for the full term.

**Detection signal:** the recommendation arrives without a chart of 7-30 days of actual vCPU-hours, or without identification of the minimum sustained level. "We always run at least X vCPU" with no evidence is the smell.

**Fix:** Pull the 30-day utilization data. Identify the steady-state minimum (p10 of vCPU-hours, conservatively). Size RIs to *that* minimum, leave the spike capacity on pay-as-you-go. Re-evaluate quarterly.

## Verification questions

1. Is the recommendation backed by at least 30 days of actual utilization data, not assumptions?
2. For right-sizing: does the new size leave headroom for p95 + 50% spike before throttling?
3. For reserved instances: is the committed baseline below the p10 of actual usage for the last 90 days?
4. For autoscale changes: is there an alert if min replicas saturate at >80% for 5+ minutes?
5. For storage tier moves: is the lifecycle policy reversible if access patterns change?
6. Did you quantify expected monthly savings *and* expected effort (engineering hours) before proposing the change?

## What to read next

- `references/cost-and-tradeoffs.md` — full cost model with per-service formulas, decision tables, and trade-off framework
- `references/compute-tier-cost-analysis.md` — Container Apps vs AKS vs App Service per-tier cost shapes, scale-to-zero behavior, when each is cheaper
- `references/reserved-instances-and-savings-plans.md` — when to commit, p10-based sizing, quarterly rebalancing protocol, how to read RI utilization reports
- `references/idle-resource-detection.md` — eight detection patterns with KQL queries and remediation playbooks for stranded databases, oversized services, hot-tier cold data, and dev/test running 24/7
- `references/data-tier-cost-optimization.md` — Cosmos RU sizing, Azure SQL DTU vs vCore, Postgres Flexible Server tier selection, storage tier transitions, backup retention cost
- `azure-service-mapping` skill — when the question is *which* service to use (cost is one input among several)
- `microservices-architecture-design` skill — for the broader context of cost as one design dimension
- Azure Cost Management documentation — for exports, anomaly detection, and budgets (re-verify URL quarterly; check `references/cost-and-tradeoffs.md` for the current link)
