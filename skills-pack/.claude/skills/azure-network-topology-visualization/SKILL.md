---
name: azure-network-topology-visualization
description: Produce an enterprise-grade, risk-annotated diagram of an Azure network estate by discovering the topology across many subscriptions, laying it out with a real graph-layout engine, and painting deterministic reachability/severity findings onto the nodes — adopting vetted open-source rendering (CloudNetDraw, ELK/D2) rather than hand-writing draw.io XML. Use when an architect or security reviewer needs a readable hub-and-spoke / multi-hub topology picture (HLD/MLD/LLD), a Confluence-publishable network diagram that refreshes itself, or a visual where exposure findings are colour-coded on the map. Do not use for computing the findings themselves (use azure-network-topology-analysis), for generating Terraform (use azure-network-iac-generation), for cost (use azure-network-cost-forecasting), for building the MCP server (use mcp-go-server-building), or for non-Azure clouds.
version: 1.0.0
last_updated: 2026-06-15
---

# Azure Network Topology Visualization

## When to use

Use this when the deliverable is a *picture* of an Azure network estate that a VP, an architect, or a security reviewer can read and trust — a hub-and-spoke or multi-hub topology with peering edges, the internet/on-prem boundary drawn, and exposure findings coloured on the nodes. It covers discovery across subscriptions, layout, severity overlay, export to draw.io/SVG, and scheduled publishing to Confluence.

Do not use it for: computing reachability or severity (that is `azure-network-topology-analysis` — this skill *consumes* its findings); generating Terraform from intent (`azure-network-iac-generation`); cost forecasting (`azure-network-cost-forecasting`); building the MCP transport (`mcp-go-server-building`); or non-Azure clouds (no adapter — say so and stop).

This skill exists because hand-rolled draw.io rendering failed in production: a single-subscription render produced a wall of isolated boxes with every node "Clean", against a human reference carrying 288 connection edges and an internet boundary. The four root causes (single-sub discovery, unrendered cross-sub peering, no boundary node type, findings never joined to the render) are the failure modes this skill is built to prevent.

## Decision rule: adopt the map, own the risk

The one thing that, if forgotten, makes everything else wrong: **the visualization tool draws the map; it never decides severity, and it never draws a single-subscription slice of a multi-subscription estate.**

Two corollaries that follow directly:

- **Severity is painted on, not computed here.** Node colours and badges come *only* from `azure-network-topology-analysis`' `Analyze()` output, joined by Azure resource ID. If a renderer or an LLM is choosing a colour, stop — the colour is a property of a computed reachable path, not of a diagram heuristic.
- **Discovery is management-group-scoped by default.** Hub-and-spoke estates peer across subscription boundaries. Discover the whole connected estate (or an explicit subscription set) in one pass, or the remote ends of peerings will be absent and the edges will silently vanish. A diagram of one subscription's slice is not a topology.

Reuse before you build: discovery + hub-spoke layout + draw.io export is a solved, commoditised problem. Adopt and vendor CloudNetDraw (MIT); use ELK (via D2) for readable layout. Write rendering code only for the one thing OSS lacks — the severity overlay.

## The work

Run the pipeline in order; each stage routes to a reference for depth.

| Stage | What you do | Reference |
|---|---|---|
| 1. Discover (multi-sub) | Enumerate VNets, subnets, peerings (incl. cross-subscription), gateways, firewalls, public IPs across a management group with Resource Graph; auth via Managed Identity / OIDC, Reader scope | `references/discovery-and-cloudnetdraw.md` |
| 2. Lay out the map | Detect hub(s) and spokes; place with a real layout engine (ELK via D2); draw external-boundary nodes (Internet, ExpressRoute, VPN GW, NAT GW, public IP); emit HLD/MLD/LLD draw.io | `references/layout-and-rendering.md` |
| 3. Paint the risk | Join `Analyze()` findings to nodes by resource ID; apply severity fill + badge and an HLD per-VNet rollup; render cross-subscription peering edges | `references/severity-overlay.md` |
| 4. Publish + verify | Export draw.io to Confluence/tWiki on a schedule with version diff; cross-check completeness against Azure Network Watcher / Monitor Network Insights Topology | `references/publish-and-pipeline.md` |

The map-vs-risk split is load-bearing: stages 1–2 are adopted OSS (CloudNetDraw + ELK/D2), stage 3 is the code you own, stage 4 is pipeline glue.

## Worked example (brownfield)

An AT&T estate spanning six subscriptions, hub-and-spoke. A naive single-subscription render draws six VNets all claiming `10.0.0.0/16`, no peering edges, every node green — because the hub lives in a connectivity subscription that was never queried, so every spoke's peer target resolved to a missing node and draw.io dropped the edge.

Do it right: scope discovery to the management group, so the hub and all spokes land in one fixture. Detect the hub (Virtual WAN hub, or the VNet with the most peerings) and draw ExpressRoute/VPN gateway/firewall as boundary objects. Lay out with ELK so 150+ private endpoints don't overlap. Then run `Analyze()` over the same fixture and paint: the spoke with a `sensitive=true` NIC reachable from the internet renders **Critical (red)**; its firewalled sibling with the byte-identical NSG rule renders **Clean (green)**. Now the legend means something, the edges connect, and the diagram tells the exposure story the wall-of-boxes could not.

## Anti-patterns

- **Letting the renderer assign severity.** A diagram tool colouring nodes by its own heuristic (e.g., "has a public IP = red"). Detection signal: a node colour that disagrees with `Analyze()` output. Fix: colours are a pure join on `Analyze()` findings by resource ID — nothing else sets them.
- **Single-subscription render of a multi-subscription estate.** Detection signal: peering edges missing, repeated address spaces across VNets, isolated boxes. Fix: management-group scope; render `CrossSubscriptionPeerings`; draw an external-stub node when a peer target is outside the queried scope rather than dropping the edge.
- **Hand-writing draw.io mxGraph coordinates.** Re-deriving layout that ELK/CloudNetDraw already do, and badly. Fix: adopt the layout engine; reserve custom code for the overlay.
- **Shipping the client-secret auth the OSS defaults to.** CloudNetDraw ships `AZURE_CLIENT_ID/SECRET/TENANT_ID` env-var auth. Fix: override to Managed Identity / OIDC, Reader scope — no `AZURE_CLIENT_SECRET` (AaraMinds standard).

## Verification questions

1. Does every node colour trace to an `Analyze()` finding by resource ID — with zero colours assigned by the renderer or an LLM?
2. Was discovery management-group-scoped (or an explicit multi-subscription set), so cross-subscription and spoke-to-spoke peering edges actually resolve to present nodes?
3. Are `CrossSubscriptionPeerings` rendered, and is a peer target outside scope drawn as an external-stub node rather than a dropped edge?
4. Are the external-boundary nodes (Internet, ExpressRoute, VPN GW, NAT GW, public IP) drawn where they exist — not omitted as in the failure case?
5. Does the layout stay legible at 100+ nodes (ELK/D2), with HLD/MLD/LLD levels available?
6. Is discovery auth Managed Identity / OIDC, Reader-scoped, with no `AZURE_CLIENT_SECRET` anywhere?
7. Did you cross-check discovery completeness against Network Watcher / Network Insights Topology (accounting for its up-to-30h Resource Graph lag)?
8. Is the adopted OSS forked and vendored (CloudNetDraw is single-maintainer), with MIT/MPL/EPL attribution retained — and is the OSPO intake logged (non-blocking for internal-only use)?

## What to read next

- The four references above, in pipeline order.
- `azure-network-topology-analysis` — produces the findings this skill paints; the source of all severity.
- `mcp-go-server-building` — to expose visualization as a `render_topology` MCP tool alongside `get_topology` / `analyze_risks`.
- `azure-service-mapping` — resource-relationship mapping that complements topology layout.
- `ai-evaluation-harness` — to gate diagram correctness (edges present, boundary drawn, colours match findings) before trusting it on a real estate.
