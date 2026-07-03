# Evaluating access: admin rules, NSGs, routes, peering

This is the deterministic core. Given a source and destination, decide whether a directed edge is open. Four gates, evaluated in order; the edge is open only if all pass. None of this is a judgment call for the model — it is computation.

## Gate 1 — AVNM security admin rules (evaluated before NSGs)

If Azure Virtual Network Manager is deployed, **security admin rules are evaluated before any NSG.** Action determines what happens next:

- `Allow` → traffic continues to NSG evaluation (Gate 2).
- `Always Allow` → traffic is permitted and evaluation **terminates** (NSGs cannot override).
- `Deny` → traffic is dropped and evaluation **terminates**.

Priority is 1–4096, lower number = higher priority; a `Deny` at 10 beats an `Allow` at 20. Miss this gate and every NSG-based verdict in an AVNM-governed estate can be wrong. If AVNM is not in use, start at Gate 2.

**Source scope is part of the gate.** An admin rule only governs traffic matching its declared source/destination. A `Deny` whose source is the `Internet` service tag closes *public-sourced* traffic but leaves the same port open from intra-VNet, peered, and on-prem sources that the NSG's broad allow still permits — so an `Internet`→3389 Deny does **not** close east-west RDP. Evaluate each admin rule against the actual source you are testing, never as a blanket open/close for the port.

## Gate 2 — NSG evaluation (subnet + NIC aggregate)

Use the **effective** rules (`network-watcher-effective.md`), which already aggregate the subnet NSG and the NIC NSG. Both associations must permit the flow:

- **Inbound:** subnet NSG, then NIC NSG — both must allow.
- **Outbound:** NIC NSG, then subnet NSG — both must allow.

Within an NSG, rules are processed by **priority ascending (100–4096), first match wins** (Allow or Deny), then the platform default rules apply. NSGs are stateful — a permitted inbound flow's response is allowed out automatically. Match on the 5-tuple (source, source port, destination, destination port, protocol), with source/destination expressible as IP/CIDR, **service tag** (e.g., `Internet`, `VirtualNetwork`, `AzureLoadBalancer`), or **application security group**.

Default rules always present (treat as implicit even if a projection omits them):

- Inbound: `AllowVNetInBound` (65000), `AllowAzureLoadBalancerInBound` (65001), `DenyAllInBound` (65500).
- Outbound: `AllowVnetOutBound` (65000), `AllowInternetOutBound` (65001), `DenyAllOutBound` (65500).

So the implicit posture is: intra-VNet allowed, inbound internet denied, outbound internet allowed — until a higher-priority rule says otherwise.

**The intra-VNet default is the trap.** Because `AllowVNetInBound` (65000) permits all VirtualNetwork→VirtualNetwork traffic, a subnet is reachable from the *entire VNet address space on every port* unless its NSG carries a `DenyVnetInBound` at a priority above 65000. A narrow rule like `allow 1433 from web` does not deny anything — it sits alongside the default allow, so app-tier→db on any port is still permitted. For a sensitive subnet, always check for an explicit VNet-inbound deny; absent one, report VNet-wide reachability, which is usually more severe than the narrow allow that first drew your eye.

## Gate 3 — routing (does a path even exist?)

A permitted NSG flow still goes nowhere without a route. Use **effective routes**; apply **longest-prefix match**, with precedence **UDR > BGP > system**. The `nextHopType` decides the edge:

- `Internet` → edge to the `Internet` node (exposure-relevant).
- `VirtualAppliance` → edge to the NVA/firewall (the flow is mediated; trace continues from the appliance).
- `VnetLocal` / `VnetPeering` → intra-VNet / to peered VNet.
- `VirtualNetworkGateway` → to on-prem/other network.
- `None` → black-holed; **no edge** (UDR `0.0.0.0/0 -> None` is a deliberate sink and *closes* exposure).

A common false alarm killer: an NSG allowing `0.0.0.0/0:22` is harmless if the effective route for `0.0.0.0/0` is `VirtualAppliance` (forced through firewall) or `None`, and there is no public IP.

## Gate 4 — peering transitivity

VNet peering is **non-transitive by default**: A↔Hub and Hub↔B does *not* give A→B. A path across spokes exists only when one of these is true:

- An NVA/Azure Firewall in the hub forwards between spokes, and the spoke UDRs point at it (`allowForwardedTraffic` on the relevant peerings).
- Gateway transit is configured (`allowGatewayTransit` on the hub peering, `useRemoteGateways` on the spoke) for on-prem reach.
- AVNM connectivity configuration establishes direct/mesh connectivity.

Encode the three peering flags as edge attributes in Gate 1's ingest, and only create a spoke→spoke reachable edge when the forwarding/transit condition actually holds. Treating peering as transitive is the most common segmentation false positive.

## Inbound DNAT — how internet traffic enters a private host

The four gates decide whether an *internal* edge is open; they assume the traffic has already entered. But a backend with **no public IP of its own** can still be internet-reachable when an Azure Firewall (or other NVA) has a **DNAT rule** that publishes it. Check this before concluding "no public IP ⇒ not exposed" — that read is the most common false negative in a hub-spoke estate.

An inbound DNAT path is real when all of these hold:

- A DNAT rule maps `<firewall public IP>:<port> → <backend private IP>:<port>` (e.g., Azure Firewall `natRules`). `sourceAddresses: ["*"]` means the whole internet is the source.
- The backend NSG allows the **post-translation source** — after DNAT the source is the firewall's private IP / subnet, so an `allow from <AzureFirewallSubnet>` rule permits it (Gate 2 on the translated flow).
- `allowForwardedTraffic: true` on the hub→spoke peering, because the DNAT'd packet is forwarded, non-originating traffic crossing VNets (Gate 4).

Compute reachability from the firewall's NAT rules, never from the backend NIC's `publicIp` field. A backend behind the firewall with **no DNAT rule targeting it** is *not* internet-reachable, even if its NSG is identical to a published peer — don't flag it (the symmetric false positive).

## Edge-open algorithm (pseudocode)

```
open(src, dst, proto, port):
  admin = avnm_admin_verdict(src, dst, proto, port)     # Gate 1
  if admin == DENY: return False
  if admin != ALWAYS_ALLOW:
      if not nsg_allows(src, dst, proto, port): return False   # Gate 2 (effective, both assoc.)
  hop = effective_next_hop(src, dst)                    # Gate 3
  if hop == None: return False
  if crosses_vnet(src, dst) and not peering_path(src, dst): return False   # Gate 4
  return True
```

Run this per candidate edge; feed the open edges to the reachability pass in `reachability-and-severity.md`.

Sources: [NSG overview](https://learn.microsoft.com/en-us/azure/virtual-network/network-security-groups-overview), [AVNM security admin rules](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-security-admins).
