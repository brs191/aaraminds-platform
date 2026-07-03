---
name: azure-network-topology-analysis
description: Build a deterministic graph of an Azure network estate (VNets, subnets, peering, NSGs, route tables, gateways, public endpoints) and analyze it for reachability-based security and segmentation risk — computing what can actually reach what, not guessing from config text. Use when reviewing or auditing an Azure subscription's network topology, hunting over-permissive NSGs, over-broad routes, transitive-peering exposure, orphaned public endpoints, or missing tier segmentation, or when an architect needs a structured, evidence-backed risk verdict on a deployed or proposed network design. Do not use for non-Azure clouds, for building the MCP server that exposes this analysis (use mcp-go-server-building), or for app-layer/identity microservice security (use azure-microservices-security).
version: 1.1.0
last_updated: 2026-06-03
---

# Azure Network Topology Analysis

## When to use

Use this when you need a structured, defensible answer to "what can reach what, and where is the network exposed?" across an Azure estate — a topology review, a security/segmentation audit, drift detection between approved design and deployed reality, or a pre-deployment check on a proposed change. It covers the deployed network plane: VNets, subnets and address space, VNet peering and transitivity, route tables and effective routes, NSGs and Azure Virtual Network Manager security admin rules, firewalls, gateways, and public endpoints.

Do not use it for: non-Azure networks (no adapter yet — say so and stop); building the MCP server that exposes this analysis as tools (use `mcp-go-server-building`); app-layer or workload identity security (use `azure-microservices-security`); or cost questions about a topology (that is a separate, forthcoming skill — for now use `azure-microservices-cost-review` for actuals only).

## Decision rule: reachability is computed, never generated

The one thing that, if forgotten, makes everything else wrong: **the language model never decides reachability or severity.** Address-space overlap, NSG rule precedence, effective routes, peering transitivity, and "is this path real" are deterministic computations over a graph built from authoritative Azure data. The model's only jobs are to explain a finding in plain language and to help phrase the report. If you catch yourself asking the model "do these CIDRs overlap?" or "is this NSG over-permissive?", stop — compute it.

A corollary: severity is a property of a *reachable path*, not of a rule's text. An NSG rule allowing `0.0.0.0/0` on 22 is only critical if an effective route to the internet and a public IP also exist, and no AVNM admin rule blocks it first. No path, no high severity.

## The work

Run the pipeline in order; each stage routes to a reference for depth.

| Stage | What you do | Reference |
|---|---|---|
| 1. Ingest inventory | Pull VNets, subnets, peerings, NSGs, route tables, public IPs, gateways across subscriptions with Resource Graph (KQL) | `references/resource-graph-ingest.md` |
| 2. Get evaluated truth | Resolve effective security rules, effective routes, topology, and next hop via Network Watcher — declared config is not enough | `references/network-watcher-effective.md` |
| 3. Evaluate access | Apply AVNM security-admin-rule → NSG precedence, route longest-prefix precedence, and peering transitivity to decide allow/deny per edge | `references/nsg-route-evaluation.md` |
| 4. Reachability + severity | Compute paths over the graph, attach evidence, assign severity, and dedupe against Defender for Cloud signals | `references/reachability-and-severity.md` |

The five v1 finding types this skill must produce: over-permissive NSG (reachable), address-space/CIDR overlap, transitive-peering exposure, orphaned public endpoint, and missing segmentation between workload tiers. Each finding carries the owning resource group and the reachable-path evidence.

## Worked example (brownfield)

An existing hub-and-spoke estate. Spoke A's `web` subnet NSG has an inbound rule `Allow * 0.0.0.0/0 -> 22` at priority 200. Reading the rule alone, a naive reviewer flags it critical.

Compute instead: the NIC's effective routes show `0.0.0.0/0 -> Internet` (no firewall UDR overriding it), the VM has an associated public IP, and no AVNM `Deny`/`Always Allow` admin rule pre-empts the NSG. That is a real internet→SSH path → **critical, with the path as evidence**. Now the same rule in Spoke B: effective routes send `0.0.0.0/0 -> VirtualAppliance` (Azure Firewall) and there is no public IP. Same rule text, **no reachable path → informational, not critical.** The graph plus effective routes is what separates the two; the rule text alone would have cried wolf on B.

## Anti-patterns

- **Severity from rule text alone.** Flagging `0.0.0.0/0` rules without checking effective routes, public exposure, and AVNM admin-rule precedence. Detection signal: a high/critical finding with no path evidence attached. Fix: every high+ finding must cite rule + effective route + exposure.
- **Reimplementing Defender for Cloud.** Re-deriving internet-exposure and attack-path findings Defender already computes on its cloud security graph. Fix: consume Defender's exposure/attack-path signal and add what it lacks (your standards, the structured topology verdict) — do not rebuild it.

## Verification questions

1. Does every high/critical finding cite an actual reachable path (rule + effective route + exposure), not just config text?
2. Did you account for AVNM security admin rules (evaluated *before* NSGs: Allow continues; Always Allow / Deny terminate) **and check each rule's source scope** — an `Internet`-tag Deny closes only public-sourced traffic, not the intra-VNet or peered paths the NSG still allows?
3. Did you apply the *default* NSG rules? A sensitive subnet with no `DenyVnetInBound` above the default `AllowVnetInBound` (65000) is reachable from the entire VNet on all ports — a narrow explicit allow denies nothing.
4. Did you use *effective* routes and *effective* security rules (subnet + NIC aggregate), not just declared UDRs/NSGs — remembering effective security rules do **not** include AVNM admin rules, which must be pulled and applied separately?
5. Did you treat peering as non-transitive by default, accounting for gateway transit and hub routing?
6. Did you dedupe against Defender for Cloud exposure/attack-path signals rather than duplicating them?
7. Did you check firewall/NVA **DNAT** rules? A backend with `publicIp: null` can still be internet-reachable if a firewall DNAT rule maps its public IP:port to that backend — `publicIp: null` is not proof of safety.
8. Is the graph sourced from Resource Graph + Network Watcher (not a portal screenshot or a stale export)?

## What to read next

- The four references above, in pipeline order.
- `azure-microservices-security` — app-layer identity, zero-trust, and segmentation patterns that complement network findings.
- `soc2-iso27001-controls-mapping` — map findings to SOC 2 / ISO 27001 controls for audit reporting.
- `mcp-go-server-building` — to expose this analysis as `get_topology` / `analyze_risks` MCP tools.
- `ai-evaluation-harness` — to build the precision/recall eval set this skill needs before it is trusted on a real subscription.
