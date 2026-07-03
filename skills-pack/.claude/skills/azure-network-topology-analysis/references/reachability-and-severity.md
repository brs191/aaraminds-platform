# Reachability, findings, and severity

With open edges decided (`nsg-route-evaluation.md`), reachability is a graph traversal and findings fall out of it. Severity is then a function of *who can reach what*, never of rule text.

## Graph model

- **Nodes:** subnet, workload/NIC, public IP, gateway, firewall/NVA, private endpoint, and a synthetic `Internet` node. Optionally a `Sensitive` tag on nodes hosting regulated/data-tier workloads.
- **Edges:** the open directed edges from the four gates, each carrying its evidence (admin verdict, deciding NSG rule, effective route + next hop, peering path).
- Keep the model cloud-neutral so the same traversal serves the future AWS adapter; Azure specifics stay in the ingest/effective layers.

## Reachability passes

- **Exposure:** BFS/DFS from the `Internet` node over open edges. Anything reached is internet-reachable; the path is the evidence.
- **Segmentation:** traverse from each lower-tier subnet toward higher-tier/sensitive nodes. Any open path that crosses a boundary that should be isolated is a segmentation break.
- **Blast radius:** from a candidate-compromised node, count reachable sensitive nodes — this sizes severity.

## The five v1 finding types

| Finding | Detection logic |
|---|---|
| Over-permissive NSG (reachable) | An effective Allow from a broad source (`Internet`/`0.0.0.0/0`/wide service tag) **and** a reachable path exists to the target. Broad rule with no path → latent, not this finding. |
| Address-space / CIDR overlap | Two connected (peered/transit) VNets or subnets have overlapping prefixes — breaks routing and signals mis-segmentation. Pure graph/IP-math check. |
| Transitive-peering exposure | A spoke→hub→spoke (or on-prem→spoke) open path exists that crosses an intended isolation boundary via forwarding/gateway transit/AVNM connectivity. |
| Orphaned public endpoint | A public IP with null `ipConfiguration`, or a public-facing endpoint with no NSG/route protecting it, or unused per flow logs. |
| Missing tier segmentation | An open path directly between tiers that policy says must be mediated (e.g., `web → db` without going through an app tier or firewall). |

**Watch the default allow, not just the explicit rules.** A sensitive subnet with no `DenyVnetInBound` is reachable VNet-wide via the default `AllowVnetInBound` even when the only *explicit* rule is narrow. Surface that as an over-permissive/segmentation finding in its own right — it is usually more severe than the narrow rule — rather than reporting only the explicit allow (see `nsg-route-evaluation.md`, Gate 2).

Every finding object carries: `type`, `severity`, `resourceId`, `owningResourceGroup`, and `evidence` = { reachable path, deciding admin/NSG rule(s), effective route + next hop, exposure (public IP / internet edge) }. No evidence, no high severity.

## Severity model

Severity is reachability × blast radius × data sensitivity — not the rule's wording.

| Severity | Condition |
|---|---|
| Critical | Internet-reachable path to a sensitive/data-tier workload, or a reachable path that exposes a large blast radius of sensitive nodes. |
| High | Internet-reachable path to a non-sensitive workload, **or** a cross-tier path to a sensitive workload. |
| Medium | Internal over-broad reachability that is bounded (no sensitive target, contained blast radius). |
| Low | Weak hygiene with a real but low-impact path (e.g., over-broad rule reaching one non-sensitive host). |
| Informational / latent | Broad rule or risky config with **no current reachable path** (e.g., `0.0.0.0/0` allow but route is `None`/firewalled and no public IP). Report it as latent so a future route change doesn't silently open it. |

The latent tier matters: it is how you record "this is one UDR change away from critical" without crying wolf today.

## Deduplicate against Defender for Cloud

Microsoft Defender for Cloud already computes internet-exposure and attack-path findings over its cloud security graph (now surfaced through Security Exposure Management). Before emitting:

- **Consume** Defender's internet-exposure and attack-path signals; where a finding matches one, corroborate and cite it rather than raising a duplicate.
- **Add** what Defender does not: grounding in the org's own network standards, the structured topology verdict, the latent-path tier, and pre-deployment (proposed-topology) analysis.
- **Never** re-derive what Defender already provides — that is the "reimplementing Defender" anti-pattern from the router.

## Output contract

Emit a structured findings list: each item `{ type, severity, resourceId, owningResourceGroup, evidence, recommendation_ref }`, sorted by severity then blast radius. High/critical route to the network-architecture team; medium/low ticket to the owning resource group. This list is exactly what the `analyze_risks` MCP tool returns and what the eval set scores precision/recall against.

Sources: [Defender attack paths](https://learn.microsoft.com/en-us/azure/defender-for-cloud/concept-attack-path), [Defender internet exposure analysis](https://learn.microsoft.com/en-us/azure/defender-for-cloud/internet-exposure-analysis).
