# Simulate a change and emit the forecast

This is where fixed and variable come together. Take the topology graph (from `azure-network-topology-analysis`) and a proposed change, apply the change to the graph, and compute the cost delta as fixed-exact + variable-band.

## Apply the change to the graph

A proposed change is a delta on the graph: add/remove a gateway or firewall, add a Private Endpoint, change a gateway SKU, add or re-route a peering, swap a public IP for NAT, move a workload across regions. Apply it to an in-memory copy of the graph so you can diff before vs after — never mutate the live model. The same graph the reviewer uses is the substrate here, so the two agents agree on what exists.

## Compute the delta

**Fixed delta (exact):**

```
fixed_delta = Σ fixed_meter(resource_added) − Σ fixed_meter(resource_removed)
```

Pull every meter live (`cost-model-fixed.md`) for the target region/SKU. This number is exact.

**Variable delta (band):** for each path whose meters change under the new topology, compute the change in per-GB cost and multiply by the traffic on that path:

```
variable_delta = Σ over changed_paths ( traffic_GB_path × ( per_GB_after − per_GB_before ) )
```

evaluated at the low and high traffic figures (`cost-model-variable.md`) to give a band. Example transformations:

- **Force egress through a firewall:** before = egress meter only; after = egress meter + firewall per-GB. Δ = firewall per-GB × egress_GB.
- **Hub-spoke → full mesh:** removes hub-transit peering hops but adds direct cross-region peering legs; Δ depends on where the traffic concentrates — compute per leg.
- **Public IP egress → NAT gateway:** adds NAT base (fixed) + NAT per-GB (variable); compare against the public-IP path it replaces.

## The forecast contract

Emit a structured object — never a bare total:

```
{
  "change": "<the proposed delta>",
  "fixed_delta_monthly": <exact number>,           // with the meters + region cited
  "variable_delta_monthly": { "low": <n>, "high": <n> },  // the band
  "traffic_basis": "<30-day VNet flow logs | assumed X TB/mo>",
  "dominant_driver": "<usually a per-GB meter>",
  "meters": [ ...cited Retail Prices meterNames... ],
  "assumptions": [ ... ],
  "verify": [ "<region/SKU/rate to confirm>" ]
}
```

Lead the human-readable summary with the dominant driver and the band, not the SKU list. "This change adds ~$900/mo fixed, and $0.016/GB of inspected egress — at your measured 40–60 TB/mo that's $640–$960/mo variable, which dominates" is a useful forecast; "$4,212/month" is not.

## Reconcile with actuals — but keep them distinct

The Azure Cost MCP Server (shared with the Cost Optimizer) holds **actuals**. Use it two ways: to source the traffic basis where flow logs feed it, and to back-test a past change's forecast against what the bill actually did. But a design-time **forecast** and a billing **actual** are different computations — share the price source, reconcile after the fact, and never present one as the other. Tagging them distinctly is verification question 5.

## What this skill does not do

- It does not assess security/blast-radius of the change — that is `azure-network-topology-analysis` (`simulate_change` there returns the security/blast-radius delta; this returns the cost delta). A full pre-deployment verdict composes both.
- It does not read or rightsize the live bill — that is `azure-microservices-cost-review`.

This cost delta is exactly what the `forecast_cost` MCP tool returns, and what a pre-deployment gate weighs alongside the security delta.
