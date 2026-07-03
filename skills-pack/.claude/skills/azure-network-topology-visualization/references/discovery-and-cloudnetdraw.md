# Multi-subscription discovery (and adopting CloudNetDraw)

Stage 1. The goal: a complete, connected estate graph in one pass — not a single-subscription slice. Single-sub discovery is the root cause (RC-1) of the production failure where peering edges vanished because the remote VNet lived in a subscription that was never queried.

## Scope: management group, not one subscription

- Discover across a **management group** (or an explicit subscription set), so a hub in a connectivity subscription and its spokes in workload subscriptions all land in the same fixture.
- Azure Resource Graph queries can span subscriptions — pass the full subscription set, not a single `subscriptionId == X` filter. The antr Phase-1 adapter (`engine/go/adapter/resourcegraph.go`) filters `where subscriptionId == %q` per query; for visualization, widen this to the management-group subscription list.
- Tell-tale that scope is wrong: multiple VNets reporting the same address space (e.g., six VNets all `10.0.0.0/16`) and zero peering edges — that is several disconnected subscriptions concatenated, not one peered estate.

## What to pull

- VNets + address space, subnets + address space, **peerings including `remoteVirtualNetwork.id`** (resolve the remote *name and subscription*), NSGs/UDRs (presence is enough for the map; effective rules come from the analysis skill), route tables, public IPs, NAT gateways.
- Gateways: ExpressRoute circuits, VPN gateways, Virtual WAN hubs — these become boundary objects in layout.
- Cross-subscription peerings must be retained as first-class edges, with the remote subscription recorded — do not collapse them into local peerings.

## Auth: Managed Identity / OIDC, never a client secret

- Use `DefaultAzureCredential` backed by a **Managed Identity** (or OIDC federated identity in CI), **Reader** at management-group scope. Read-only — no write, no `terraform apply`.
- CloudNetDraw ships a service-principal path using `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET` / `AZURE_TENANT_ID` env vars. **Override it.** No `AZURE_CLIENT_SECRET` anywhere — this is an AaraMinds locked decision (A-05). When forking, replace the auth module with `DefaultAzureCredential`.

## Adopting CloudNetDraw (MIT)

CloudNetDraw (`krhatland/cloudnetdraw`, Python, MIT) already does discovery + hub-spoke detection + draw.io export across subscriptions. Adopt it rather than rebuilding:

- **Fork and vendor it** — it is a single-maintainer project (~137★). Do not pin upstream; own the source, retain MIT attribution, log the OSPO intake (non-blocking for internal-only use).
- Useful surfaces: `cloudnetdraw query` (discovers across readable subscriptions → JSON), `cloudnetdraw hld` / `mld` (JSON → draw.io). Hub detection: Virtual WAN hub, else the VNet with the most peerings. It already draws ExpressRoute/VPN GW/Firewall as boundary objects and supports spoke-to-spoke, cross-zone, and multi-hub edge types.
- Its `utils/topology-generator.py` + `topology-validator.py` give you synthetic estates and structural validation — reuse them as test fixtures.
- The seam to preserve: CloudNetDraw emits a topology JSON keyed by Azure resource IDs. Keep those IDs intact — they are the join key for the severity overlay (stage 3).

## Cross-check

Validate discovery completeness against Azure Network Watcher / Monitor **Network Insights Topology** (native, multi-sub, JSON export). Note its Resource Graph backing can lag up to ~30h, so treat it as a completeness sanity check, not a real-time oracle.

## Done when

A single discovery pass over the management group yields a JSON graph where every peering's remote end resolves to a present node (or is explicitly marked out-of-scope), address spaces are distinct per connected VNet, and gateways are captured — with no `AZURE_CLIENT_SECRET` used.
