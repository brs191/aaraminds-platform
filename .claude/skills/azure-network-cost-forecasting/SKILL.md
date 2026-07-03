---
name: azure-network-cost-forecasting
description: Forecast the cost impact of an Azure network topology or a proposed change to it, separating fixed SKU/base fees (exact, from the Azure Retail Prices API) from variable data-processing and egress costs (an estimated band built from a traffic basis). Use when an architect needs a design-time cost delta on a topology decision (hub-spoke vs mesh, gateway SKU, Azure Firewall placement, Private Link footprint, NAT gateway vs public IP, same-region vs cross-region peering), a cost projection before a change ships, or a cost dimension alongside a topology review. Do not use for billing actuals or FinOps on already-deployed resources (use azure-microservices-cost-review), for the topology risk analysis itself (use azure-network-topology-analysis), or for non-Azure clouds.
version: 0.1.0
last_updated: 2026-06-03
---

# Azure Network Cost Forecasting

## When to use

Use this to answer "what will this network design — or this change to it — cost, and where does the money actually go?" at design time, before anything is committed. It covers the network cost surface: gateway and firewall SKUs, Private Endpoints, NAT gateway, public IPs, VNet peering (same-region and cross-region/global), data egress, and the per-GB data-processing meters that dominate real bills. It runs on the topology graph from `azure-network-topology-analysis` and produces a fixed-cost delta plus a variable-cost band.

Do not use it for: billing actuals or rightsizing of deployed resources (that is `azure-microservices-cost-review`, which reads the bill); the topology risk/reachability analysis itself (`azure-network-topology-analysis`); or non-Azure clouds.

## Decision rule: split fixed (exact) from variable (banded); never quote a false-precision total

The one thing that, if forgotten, makes the forecast worthless: **fixed costs are computable to the cent, variable costs are not.** Fixed = hourly/standing SKU fees (gateway, firewall base, Private Endpoint, NAT base, public IP) — pull these live from the Retail Prices API and they are exact. Variable = per-GB data processing and egress (firewall per-GB, NAT per-GB, cross-region peering per-GB, internet egress) — these depend on a **traffic volume you must source or assume**, so they ship as a band with the assumption stated, never as a single confident number.

The corollary architects get wrong: **the money is usually in the variable meters, not the SKU base fees.** A firewall's base fee is a rounding error next to its per-GB processing at scale. Lead the forecast with the variable driver and its traffic basis, not the SKU list.

## The work

| Stage | What you do | Reference |
|---|---|---|
| 1. Fixed costs | Query the Retail Prices API for every standing SKU in the (proposed) topology; assemble exact monthly fixed cost | `references/cost-model-fixed.md` |
| 2. Variable costs | Identify the per-GB meters each path crosses (egress, peering, NAT, firewall, Private Link); multiply by a traffic basis from VNet flow logs / Traffic Analytics, or a stated assumption | `references/cost-model-variable.md` |
| 3. Simulate + forecast | Apply the proposed change to the topology graph, compute fixed delta + variable band, reconcile vs actuals, emit the forecast | `references/simulation-and-forecast.md` |

Every forecast carries: the fixed delta (exact, with the price meters cited), the variable delta (a band with its traffic assumption and the meters), and a one-line "dominant driver" call.

## Worked example (brownfield)

A hub-and-spoke estate routes spoke egress directly to the internet; the architect proposes forcing all egress through a new Azure Firewall in the hub. Forecast the change:

- **Fixed delta (exact):** + Azure Firewall base hourly × 730 + the firewall's public IP. Pull both meters from the Retail Prices API for the region. Call it `[VERIFY region/SKU]`.
- **Variable delta (band):** every GB of egress now also pays the firewall **data-processing per-GB** meter (egress per-GB is unchanged — it still leaves Azure). Traffic basis: last 30 days of egress GB from the spokes' VNet flow logs via Traffic Analytics. Forecast = base_hourly×730 + (egress_GB_per_month × firewall_per_GB), expressed as a band across the p50–p90 of observed monthly egress.
- **Dominant driver:** at 50 TB/month the per-GB processing dwarfs the base fee — say so, and show the band, not a single total.

## Anti-patterns

- **False-precision total.** Reporting "this change costs $4,212/month" for a design with a large variable component. Detection: a single number where a traffic-dependent meter is in play. Fix: fixed exact + variable band + stated traffic basis.
- **Hardcoding prices.** Quoting per-GB or SKU rates from memory. Prices change and vary by region. Fix: always pull live from the Retail Prices API; mark any indicative figure `[VERIFY]`.
- **Conflating forecast with actuals.** Presenting a design-time forecast as if it were the bill. Fix: forecast and the Azure Cost MCP actuals are different computations; share the price source, reconcile, but keep them labeled distinctly.

## Verification questions

1. Is every fixed cost pulled live from the Retail Prices API for the correct region and SKU, not hardcoded?
2. Does every variable cost cite its per-GB meter *and* the traffic basis (flow-log derived or a stated assumption)?
3. Is the variable cost a band, not a single number, wherever traffic is uncertain?
4. Did you name the dominant driver (usually a data-processing meter), not just list SKU fees?
5. Did you keep forecast separate from the Azure Cost MCP actuals rather than blending them?
6. For cross-region/global peering and egress, did you use the right per-region rate (they differ by zone)?

## What to read next

- The three references above, in stage order.
- `azure-network-topology-analysis` — supplies the topology graph and the proposed-change delta this skill prices.
- `azure-microservices-cost-review` — for actuals/FinOps on deployed resources (the complement to this design-time forecast).
- The Azure Cost MCP Server — shared price/actuals source so this forecast and the Cost Optimizer agree on numbers.
