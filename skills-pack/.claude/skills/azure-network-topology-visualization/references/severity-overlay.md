# Severity overlay (the layer you own)

Stage 3. This is the only stage that is not commodity OSS — it is the reason the tool exists. The map (stages 1–2) shows *what connects to what*; the overlay shows *where the risk is*. Failure mode RC-4: findings computed but never joined to the render, so the legend is decorative and every node renders "Clean".

## The boundary that must not blur

**Severity is computed by `azure-network-topology-analysis` (`Analyze()`), never by the renderer and never by an LLM.** This skill *joins and paints* — it does not decide. Painting is a deterministic function of `Analyze()` output. If any colour is chosen by a diagram heuristic ("public IP → red"), the deterministic-severity guarantee is broken.

## The join

- `Analyze()` returns `[]Finding{ Type, Severity, Resource, Evidence, Reachable }`. `Resource` is an Azure resource identifier.
- The diagram graph (CloudNetDraw JSON / fixture) keys nodes by Azure resource ID.
- Join finding → node **by resource ID**. Run discovery and `Analyze()` over the **same fixture** so the IDs are guaranteed to match (do not re-discover between the two — drift breaks the join).
- A node may carry several findings; take the **max severity** for the fill, and list all finding types in the node tooltip/badge.

## The paint

| Severity | Fill | Badge |
|---|---|---|
| Critical | red | 🔴 |
| High | orange | 🟠 |
| Medium | yellow | 🟡 |
| Info / latent | blue | 🔵 |
| Clean (no finding) | green | 🟢 |

- **Reachable vs latent:** a `Reachable=false` finding (e.g., a broad NSG rule with no path) is Info/latent, not High — mirror `Analyze()`'s reachability gate; never promote a latent finding to a hot colour.
- **HLD rollup:** a VNet's HLD colour = the max severity of any node inside it, so the exec view surfaces the worst case per VNet.
- **Edges:** colour or annotate an edge that carries a finding (e.g., an open internet→NIC path) so the *path*, not just the endpoint, is visible — this is what the 288-edge reference conveyed and the failed render did not.

## Cross-subscription peerings (RC-2)

Render `CrossSubscriptionPeerings` explicitly. They are a separate field from local `Peerings`; a renderer that loops only local peerings draws zero cross-sub edges. Each cross-sub edge still gets its severity treatment if a finding attaches to the path.

## Done when

Every node/edge colour is a pure function of `Analyze()` output joined by resource ID; max-severity rollup is applied per VNet at HLD; reachable and latent findings are coloured distinctly; cross-sub peering edges render; and a spot-check confirms a node's colour equals its finding's severity byte-for-byte (the renderer assigned nothing).
