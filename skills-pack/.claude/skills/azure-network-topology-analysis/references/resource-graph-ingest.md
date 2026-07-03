# Inventory ingestion with Azure Resource Graph

Resource Graph (ARG) is the inventory layer: a read-only, indexed store you query with KQL across every subscription in scope, returning results in seconds at tenant scale. It gives you *declared* configuration — the substrate for the graph. It does **not** give you *effective* rules or routes; that resolution is Network Watcher (`network-watcher-effective.md`). Build the node set from ARG, then enrich edges with Network Watcher.

## Scope and access

- Run at management-group scope for tenant-wide reach, or pass an explicit subscription list. The agent's identity needs only `Reader` on the scope — ARG honors RBAC, so it returns exactly what the identity can see.
- ARG is eventually consistent: there is an indexing lag (typically seconds to a few minutes) after a resource changes. For audit, note the query timestamp; do not treat ARG as real-time truth for a change made moments ago.
- Page with `--skip-token` (CLI: `az graph query`) — a single query caps at 1000 rows by default; large estates need pagination.

## Resource types to pull

| Node | Resource type | Key properties |
|---|---|---|
| VNet | `microsoft.network/virtualnetworks` | `addressSpace.addressPrefixes`, `subnets`, `virtualNetworkPeerings` |
| Subnet | (nested in VNet `properties.subnets`) | `addressPrefix(es)`, `networkSecurityGroup.id`, `routeTable.id`, `serviceEndpoints`, `delegations` |
| NSG | `microsoft.network/networksecuritygroups` | `securityRules`, `defaultSecurityRules`, `subnets`, `networkInterfaces` |
| Route table | `microsoft.network/routetables` | `routes` (`addressPrefix`, `nextHopType`, `nextHopIpAddress`), `subnets`, `disableBgpRoutePropagation` |
| Public IP | `microsoft.network/publicipaddresses` | `ipConfiguration.id` (null = orphaned), `ipAddress`, `publicIPAllocationMethod` |
| NIC | `microsoft.network/networkinterfaces` | `ipConfigurations` (subnet, public IP), `networkSecurityGroup.id` |
| Gateway | `microsoft.network/virtualnetworkgateways` | `gatewayType` (Vpn/ExpressRoute), `vpnType`, `bgpSettings` |
| Firewall | `microsoft.network/azurefirewalls` | `ipConfigurations`, `firewallPolicy.id` |
| Private endpoint | `microsoft.network/privateendpoints` | `subnet.id`, `privateLinkServiceConnections` |

## Worked queries

VNets with their subnets and the NSG/route table bound to each subnet:

```kusto
Resources
| where type =~ "microsoft.network/virtualnetworks"
| mv-expand subnet = properties.subnets
| project vnetId = id, vnet = name, rg = resourceGroup, sub = subscriptionId,
          addressSpace = properties.addressSpace.addressPrefixes,
          subnet = subnet.name,
          subnetPrefix = subnet.properties.addressPrefix,
          nsgId = tostring(subnet.properties.networkSecurityGroup.id),
          routeTableId = tostring(subnet.properties.routeTable.id)
```

VNet peerings (state and the flags that govern transitivity):

```kusto
Resources
| where type =~ "microsoft.network/virtualnetworks"
| mv-expand peering = properties.virtualNetworkPeerings
| project vnet = name, rg = resourceGroup,
          peeringState = tostring(peering.properties.peeringState),
          remoteVnet = tostring(peering.properties.remoteVirtualNetwork.id),
          allowForwardedTraffic = tobool(peering.properties.allowForwardedTraffic),
          allowGatewayTransit = tobool(peering.properties.allowGatewayTransit),
          useRemoteGateways = tobool(peering.properties.useRemoteGateways)
```

Orphaned NSGs (associated to neither subnet nor NIC — noise to flag and clean):

```kusto
Resources
| where type =~ "microsoft.network/networksecuritygroups"
| where isnull(properties.subnets) and isnull(properties.networkInterfaces)
| project name, resourceGroup, subscriptionId
```

Public IPs and whether they are attached (null `ipConfiguration` = orphaned candidate):

```kusto
Resources
| where type =~ "microsoft.network/publicipaddresses"
| project name, rg = resourceGroup, ip = tostring(properties.ipAddress),
          attachedTo = tostring(properties.ipConfiguration.id),
          orphaned = isnull(properties.ipConfiguration)
```

Enrich any result with friendly subscription / management-group names by joining `ResourceContainers`:

```kusto
Resources
| where type =~ "microsoft.network/virtualnetworks"
| join kind=leftouter (
    ResourceContainers
    | where type =~ "microsoft.resources/subscriptions"
    | project subscriptionId, subName = name
  ) on subscriptionId
| project name, resourceGroup, subName
```

## From query output to graph

- One node per VNet, subnet, NIC/workload, public IP, gateway, firewall, private endpoint. Add a synthetic `Internet` node.
- Declared edges from ARG: subnet→NSG, subnet→routeTable, NIC→subnet, publicIP→NIC, VNet↔VNet (peering, with the three transitivity flags as edge attributes).
- Carry IDs verbatim (`id`) as node keys so the Network Watcher enrichment and the cloud-neutral model line up. Map Azure types onto the neutral schema here — keep raw Azure JSON out of the analysis engine.

## Caveats

- ARG returns **declared** NSG rules; it does not resolve subnet+NIC aggregation or AVNM admin-rule precedence. Never score reachability from ARG alone — that is `nsg-route-evaluation.md` working on `network-watcher-effective.md` output.
- `defaultSecurityRules` may be omitted from some ARG projections; treat the documented Azure defaults as always present (see `nsg-route-evaluation.md`).
- AVNM security admin rules are not fully represented in the per-resource ARG view — pull the Network Manager configuration separately when AVNM is in use.
- Sources: [ARG networking samples](https://learn.microsoft.com/en-us/azure/networking/fundamentals/resource-graph-samples), [ARG query language](https://learn.microsoft.com/en-us/azure/governance/resource-graph/concepts/query-language).
