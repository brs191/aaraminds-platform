# Reserved Instances and Savings Plans — Commitment Strategy

## When to use this reference

When reviewing a monthly bill and deciding whether to commit to Reserved Instances (RIs), Azure Savings Plans, or stay on pay-as-you-go (PAYG); when justifying a 20–30% cost-cut ask from leadership where commitments are the realistic lever; or when an existing RI portfolio is delivering less savings than expected and quarterly rebalancing is overdue. Do not use this for choosing the *runtime platform itself* (see `compute-tier-cost-analysis.md`).

## The one-sentence rule

Commit to RIs or Savings Plans only against the workload's **sustained minimum** — never against the average, and never against a future plan. The bill is real today; the future is conditional.

## The three commitment options in one paragraph

**Pay-as-you-go (PAYG)** is the default — full price, full flexibility, no commitment. **Reserved Instances** lock a specific SKU (e.g., D4s_v5 in West Europe) for 1 or 3 years for ~30–60% off PAYG; the discount is the largest but the lock-in is rigid — wrong region, wrong SKU, wrong family means you keep paying for capacity you can't use. **Azure Savings Plans for Compute** commit to an hourly *dollar* spend for 1 or 3 years for ~17–28% off PAYG; the discount is smaller than RIs but applies across VM families, regions, and even some PaaS services (App Service, Container Apps Dedicated). Savings Plans are the right default; RIs are the right precision instrument when the workload is locked-in to a specific SKU long-term.

## Decision table — which commitment to pick

| Workload property | Pick | Why |
|---|---|---|
| Steady VM fleet on one SKU family, multi-year roadmap | 3-year RI on that SKU | Maximum discount; you've already absorbed the lock-in risk |
| Steady spend, evolving SKU mix (D-series today, E-series next year) | 1-year or 3-year Savings Plan | Discount follows you across families |
| Mix of AKS nodes + App Service + Container Apps Dedicated | Savings Plan | Cross-service applicability; RIs don't cover Container Apps |
| Cosmos DB at sustained ≥4,000 RU/s | Cosmos DB Reserved Capacity (1 or 3 year) | Separate RI program; up to ~65% off provisioned throughput |
| Azure SQL or Postgres at sustained vCore count | SQL / Postgres Reserved Capacity | Separate RI program per database service |
| Cache for Redis Premium tier, steady use | Redis Reserved Capacity | Up to ~55% off; only for Premium/Enterprise tiers |
| Spiky workload, no clear floor | PAYG | Commitment will leak money on idle hours |
| Pre-launch product, traffic unknown | PAYG for 90 days, then re-evaluate | Don't commit on assumptions; commit on measured floor |

## Sizing the commitment — the only way that works

The fatal mistake is sizing against the average. The average includes idle time you'd never commit to in isolation. Use the **p10 of hourly vCPU consumption over the last 90 days**:

1. Pull 90 days of vCPU-hours per SKU family from Azure Monitor / Cost Management exports.
2. Bucket into hourly samples.
3. Sort ascending. Take the 10th percentile — the level you ran at *or above* 90% of the hours.
4. Subtract 10% safety margin. That's your commitment ceiling.

If the p10 is 8 vCPU sustained on D-series, commit to 7 vCPU on a Savings Plan. The remaining demand flows to PAYG. The 7 vCPU is consumed by the commitment regardless of which D-series SKU runs; PAYG absorbs spikes and SKU mix changes.

Concrete: 30-day average says 12 vCPU sustained. The p10 says 6 vCPU. You commit to 5 vCPU. The other 7 vCPU floats. If you'd committed to 12, you'd have paid for ~6 vCPU of idle every hour the workload sat near its floor.

## Break-even math

RI break-even (1-year, ~37% discount): you must use the reserved capacity for **~7.5 months out of 12** to come out ahead vs PAYG. The reservation is paid up-front (or monthly amortized) regardless of use.

Savings Plan break-even (1-year, ~17% discount): you must spend the committed hourly amount for **~10 months out of 12** equivalent. Lower discount but the threshold is lower too because unused hours within the day still count toward the commitment if other consumption fills them.

3-year commitments roughly double these required-use thresholds in absolute hours, but the per-month break-even ratio is similar. Pick 3-year only when you're confident the workload exists in 36 months.

| Scenario | Suggested commitment | Expected savings |
|---|---|---|
| 24/7 VM at known SKU for 3 years | 3-year RI | 55–62% |
| 24/7 VM at known SKU for 1 year | 1-year RI | 30–40% |
| 24/7 mixed VM fleet for 3 years | 3-year Savings Plan | 23–28% |
| 24/7 mixed VM fleet for 1 year | 1-year Savings Plan | 15–18% |
| 12 h/day VM (50% utilization) | Savings Plan sized to that 12-hour floor only | 8–14% on overall bill |
| <8 h/day workload | PAYG | Commitment leaks money |

## The quarterly rebalancing protocol

RI and Savings Plan portfolios are not "set and forget." Workloads change; commitments don't, unless you actively rebalance. Run this every quarter (block 2 hours on the calendar):

1. **Pull the RI utilization report.** Azure Portal → Reservations → each reservation → Utilization tab. Or via API: `Microsoft.Capacity/reservationOrders/reservations`. Target: ≥95% utilization. Anything under 80% is leaking money.

2. **Pull the Savings Plan utilization report.** Same path. Same target.

3. **Identify under-utilized reservations.** For each: what changed? Did a service scale down? Did a workload move regions? Did someone migrate AKS to Container Apps and forget the RI was tied to AKS node SKU?

4. **Exchange or split.** RIs allow exchanges (within the same reservation order, no penalty) for a different SKU or region. Use this aggressively. Do not let a 60%-utilized RI run to term — exchange it for the SKU you now need.

5. **Cancel if unrecoverable.** Microsoft allows up to $50,000 of RI cancellations per year per billing profile, with a ~12% early-termination fee. Use this for clearly stranded RIs (workload retired).

6. **Top up under-committed PAYG.** Look at SKUs running >$200/month sustained on PAYG that have no reservation. Add a Savings Plan or new RI.

7. **Update the commitment register.** Track every reservation in a spreadsheet or table: SKU, term, end date, monthly amortized cost, current utilization, owner. Without the register, the next quarterly review starts from zero.

## How to read an RI utilization report

The Azure portal RI utilization view shows daily utilization % per reservation. What each pattern means:

| Pattern | Diagnosis | Action |
|---|---|---|
| Flat ~100% all month | Healthy. Workload is meeting or exceeding the commitment. | Renew at end of term; consider scaling commitment up |
| Flat ~70%, no trend | Under-sized workload or oversized commitment | Exchange to smaller SKU or fewer instances |
| Decline from 100% to 50% over 30 days | Workload migrated or scaled down | Investigate; exchange RI to match new workload |
| 100% weekdays, 0% weekends | Dev/test workload with autostop | Either drop the RI (dev/test shouldn't have one) or extend autostop |
| Spiky 0–100% daily | Workload is bursty; RI is wrong instrument | Exchange to Savings Plan or move to PAYG |
| 100% with PAYG charges in same SKU | Under-committed; workload exceeds the reservation | Add another RI or top up the Savings Plan |

The RI utilization view is *the* artifact to bring to the FinOps review. "We saved $X this quarter because all reservations stayed >95%" is the win. "Reservation Y dropped to 40% because we migrated service Z" is the diagnosis that leads to the exchange.

## What RIs and Savings Plans don't cover

- **Container Apps Consumption** — pure consumption pricing, no commitment program. Only Container Apps Dedicated workload profiles are eligible for Savings Plans (treated as VMs underneath).
- **Azure Functions Consumption plan** — no commitment program; commit by switching to Premium plan + Savings Plan if predictable.
- **Service Bus, Event Hubs, Event Grid** — no reserved capacity except for Event Hubs Dedicated clusters (separate program, large minimum).
- **Log Analytics ingestion** — has a separate Commitment Tier program (100 GB/day, 200 GB/day, etc.) with 15–30% off pay-per-GB. Operationally distinct from RIs/Savings Plans; treat as its own commitment decision.
- **Application Gateway, Front Door, API Management** — no RI program; pick the right SKU instead.
- **Bandwidth/egress** — no commitment program; the only lever is architectural (reduce cross-region traffic).

## Brownfield — auditing an existing RI portfolio

Inheriting an RI portfolio is common. The first audit:

1. Export all reservations from the billing account: Portal → Reservations → Export, or `az reservations reservation-order list`.
2. For each: SKU, term remaining, monthly amortized cost, current utilization %.
3. Cross-reference with current workload inventory. Is the SKU still in use? Is the region still active?
4. Tag every reservation with an owner. Unowned reservations are how dead commitments survive for years.
5. Build a renewal calendar — auto-renew is off by default in many tenants. A 1-year RI that lapses without renewal becomes PAYG silently; the bill spikes 40%+ overnight.
6. Sort by leakage (PAYG cost in the SKU that the RI failed to cover, plus unused RI capacity). Fix the worst three this quarter.

## Anti-patterns

- **Sizing RIs to the average usage.** Average includes idle. You will pay for ~half the idle hours forever. Use p10.
- **3-year commitments on a 1-year roadmap.** The discount is alluring; the lock-in is severe. Use 1-year unless the workload has 36 months of demonstrated stability.
- **Buying RIs before measuring.** A "20% off" framing without 90 days of utilization data is a guess. The 20% off is only real if you actually use ≥80% of the reservation.
- **Forgetting Cosmos / SQL / Redis reserved capacity exist.** Compute RIs and Savings Plans get all the attention, but the data tier is often 40–60% of the bill. A 50%-off Cosmos reserved capacity on a steady 10K RU/s container is $7K–$10K/year saved with one purchase.
- **Stranded RIs after migration.** Service moved from AKS to Container Apps; the AKS node-SKU RI is orphaned. Exchange immediately — Microsoft allows it free within the same reservation order.
- **No quarterly review.** RI utilization decays silently. Without the rhythm, the portfolio drifts to ~70% utilization in a year and the "savings" become a tax.
- **Renewing without re-sizing.** End of 1-year term comes up; renewal at the same size is the path of least resistance. Re-run the p10 analysis every renewal; the workload has changed.

## What this is not

This reference is the commitment-strategy lens. For *which compute platform to run on in the first place* (which determines what's even eligible for an RI), see `compute-tier-cost-analysis.md`. For detecting *idle* resources before deciding whether to commit to them (committing to idle = burning money faster), see `idle-resource-detection.md`. For Cosmos DB throughput sizing (the input to a Cosmos reserved capacity decision), see `data-tier-cost-optimization.md`. For the general cost framework and per-service formulas, see `cost-and-tradeoffs.md`.
