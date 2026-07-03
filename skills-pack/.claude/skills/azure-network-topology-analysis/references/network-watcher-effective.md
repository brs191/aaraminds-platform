# Effective truth with Network Watcher

Declared config (from ARG) is not what the data plane enforces. Network Watcher resolves the *effective* state — the aggregate of subnet + NIC NSGs, and the merged system/UDR/BGP route table — and answers point reachability questions. This is the layer that turns "the config says" into "the network actually does."

## Effective security rules (per NIC)

The applied inbound/outbound set is the **aggregate of the subnet NSG and the NIC NSG**. Resolve it per NIC:

```bash
az network nic list-effective-nsg --name <nic> --resource-group <rg> -o json
```

API: `NetworkInterfaces - GetEffectiveNetworkSecurityGroups`. The response lists each effective rule with the association (subnet, NIC, or default) it came from. Use this — not the declared NSG — when deciding whether a 5-tuple is allowed.

Constraint that shapes the pipeline: **the effective APIs require a provisioned NIC.** A subnet with no NICs has no "effective" view. So cover empty/!-yet-deployed subnets from ARG declared config, and use effective resolution where NICs exist. For pre-deployment (proposed) topology, you have only declared config — evaluate it with the same rules engine, flagged as "declared, not effective."

**Effective security rules do not include AVNM.** Network Watcher's effective rules reflect NSGs only — they do *not* incorporate Azure Virtual Network Manager security admin rules, which are evaluated ahead of NSGs. In an AVNM-governed estate, an exposure read from effective rules alone is wrong in both directions: a port can look denied but be force-opened by an `AlwaysAllow`, or look allowed but be closed by a `Deny`. Always pull the Network Manager admin rules separately and apply Gate 1 (`nsg-route-evaluation.md`) on top of the effective-rules view.

## Effective routes (per NIC)

```bash
az network nic show-effective-route-table --name <nic> --resource-group <rg> -o json
```

API: `NetworkInterfaces - GetEffectiveRouteTable`. Returns every route applying to the NIC — system routes, user-defined routes (UDRs), and BGP routes — already merged by Azure's precedence. Each entry has `addressPrefix`, `nextHopType`, and `nextHopIpAddress`. `nextHopType` is the field reachability hinges on:

| `nextHopType` | Meaning for reachability |
|---|---|
| `Internet` | Egress to public internet for that prefix |
| `VirtualNetworkGateway` | To on-prem / other network via VPN/ER |
| `VnetLocal` | Within the VNet |
| `VirtualAppliance` | To an NVA / Azure Firewall (`nextHopIpAddress`) — often the firewalled path |
| `VnetPeering` / `VirtualNetworkPeering` | To a peered VNet |
| `None` | Black-holed — traffic dropped (a UDR `0.0.0.0/0 -> None` is a deliberate sink) |

A `0.0.0.0/0 -> VirtualAppliance` route is the signal that egress is forced through a firewall; `0.0.0.0/0 -> Internet` with a public IP is the signal of direct exposure.

## Topology, next hop, IP flow verify

- **Topology** — returns the resources in a resource group / VNet and their associations (VNet, subnet, NIC, NSG, route table, gateway, load balancer). A fast way to seed/cross-check the graph for a scope.
- **Next hop** — `az network watcher show-next-hop --source-ip <vm-ip> --dest-ip <ip> ...` evaluates the NIC's effective routes for a destination and returns the next hop type, IP, and route ID. Use it to *confirm* a computed egress path against Azure's own resolver.
- **IP flow verify** — `az network watcher test-ip-flow --direction Inbound --protocol TCP --local <ip:port> --remote <ip:port> ...` returns Allow/Deny against the effective NSG for a specific 5-tuple, and names the deciding rule. Use it to *confirm* an NSG verdict on a contested finding.

Treat next-hop and IP-flow-verify as **verification oracles**: compute reachability yourself over the graph, then spot-check high-severity findings against these so the verdict matches Azure's own evaluation.

## Traffic evidence (optional, for prioritization)

VNet flow logs + Traffic Analytics show whether a permitted path is actually *used*, which helps rank findings (an open-but-idle path vs an open-and-busy one). Build on **VNet flow logs** — NSG flow logs stop accepting new resources after 30 Jun 2025 and retire 30 Sep 2027. This is optional for v1 analysis and central to the later cost-forecasting skill.

## Permissions and cost

- Network Watcher data-plane reads need `Reader` plus the Network Watcher actions; keep this identity read-only.
- The effective and diagnostic calls are per-NIC/per-query — at estate scale, batch and cache. Resolve effective rules/routes once per NIC per run and reuse across all findings touching that NIC.

Sources: [Network Watcher overview](https://learn.microsoft.com/en-us/azure/network-watcher/network-watcher-overview), [VNet flow logs migration](https://learn.microsoft.com/en-us/azure/network-watcher/nsg-flow-logs-migrate).
