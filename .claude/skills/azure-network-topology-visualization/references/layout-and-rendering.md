# Layout and rendering (ELK/D2, boundary nodes, levels of detail)

Stage 2. Turn the discovered graph into a readable diagram. Two failure modes to kill: unreadable hand-placed coordinates (the "wall of boxes"), and a missing external boundary (RC-3 — no Internet/gateway node type).

## Use a real layout engine

- **ELK (Eclipse Layout Kernel)** is the gold standard for compound graphs with ports and orthogonal edge routing — exactly network topology. It handles hierarchy (VNet > subnet > NIC), disconnected-component packing, and overlap removal, which is what keeps 100+ nodes legible.
- Consume ELK through **D2** (`terrastruct/d2`) — Go-native (matches the antr engine), bundles ELK as a layout plugin, renders SVG/PNG programmatically (`oss.terrastruct.com/d2`, layout `elk`). This replaces hand-written mxGraph coordinates.
- If staying inside CloudNetDraw's draw.io path, keep its logic-based hub-spoke placement for HLD and reserve ELK for dense MLD/LLD where manual placement breaks down. Do not hand-place coordinates in new code (that was RC-3's neighbour failure).

## Draw the external boundary (RC-3)

The renderer must have node *types* for the outside world, or the most important edges can never be drawn:

- `Internet` (synthetic node) — the source for exposure edges.
- `ExpressRoute` circuit, `VPN Gateway`, `NAT Gateway`, on-prem block.
- `Public IP` as an attachable node (so an exposed NIC shows its public surface).

When a peering target is outside the discovered scope, draw an **external-stub node** ("peers outside this view") rather than dropping the edge — a dangling edge with a missing target is silently discarded by draw.io.

## Levels of detail

| Level | Contents | Use |
|---|---|---|
| HLD | VNets + peerings + hub/gateways/firewall + per-VNet severity rollup | exec / architecture review |
| MLD | + subnets, address space, NSG/UDR presence badges | operations / segmentation review |
| LLD | + NICs, private endpoints, public IPs | deep security review (antr already enumerates these) |

Generate all three from the same graph; switch detail, not data.

## Icons and branding

Use Azure's official service icon set (AzViz's icon mapping is a good reference). Keep the legend honest — it must reflect the severity scale the overlay actually applies (see `severity-overlay.md`).

## Export targets

- **draw.io** is the primary target — it round-trips cleanly into Confluence/tWiki (proven in the existing pipeline). Keep it.
- **SVG/PNG** via D2 for static reports.
- An interactive web canvas (Cytoscape.js / D2 SVG with zoom/filter/drill + portal links) is the Phase-4-plus "live artifact" — out of scope for the first cut.

## Done when

A 100+-node estate renders without overlap at HLD/MLD/LLD; Internet/ER/VPN GW/NAT/public-IP nodes appear where present; out-of-scope peer targets render as stubs, not dropped edges; draw.io imports into Confluence without manual fixup.
