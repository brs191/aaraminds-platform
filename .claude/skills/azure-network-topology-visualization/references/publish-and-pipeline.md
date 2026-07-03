# Publish and pipeline (Confluence, scheduling, version diff)

Stage 4. Make the diagram a living document, not a one-off PDF. An out-of-date diagram is the problem this whole capability exists to solve, so refresh has to be automatic.

## Publish to Confluence / tWiki

- draw.io is the import format — Confluence accepts it via the diagrams.net integration ("Edit > XML" / insert diagram from file). The existing tWiki pipeline already round-trips CloudNetDraw-style draw.io cleanly, so keep draw.io as the publish artifact.
- Publish HLD as the landing page; link MLD/LLD as child pages. Embed Azure portal hyperlinks on nodes (CloudNetDraw already emits these) so a reviewer can jump from the diagram to the resource.

## Schedule the refresh

- Host as an **Azure Function** (timer trigger) that re-runs the full pipeline: discover (mgmt-group, Managed Identity) → `Analyze()` → overlay → render → publish.
- Cadence: daily or on-demand. Account for Resource Graph propagation lag (changes can take up to ~30h to appear) — do not promise real-time.
- The Function's identity is the read-only Managed Identity with Reader at management-group scope; no secrets, no write.

## Version history and diff

- Keep each published diagram in Confluence version history (it does this natively); supersede rather than overwrite.
- Compute a **topology diff** between runs (added/removed VNets, new peerings, new exposed nodes, severity changes) and post it as the page summary — the diff is often more useful than the diagram itself for ops.
- Store the underlying topology JSON alongside the diagram so diffs are computed on structured data, not pixels.

## Cross-check as a gate

Before publishing, compare node/edge counts against Network Watcher / Network Insights Topology for the same scope. A large discrepancy means discovery scope or permissions are wrong — fail the run rather than publish an incomplete map (the failure mode this capability was built to prevent).

## Expose as an MCP tool (optional)

To fit the antr MCP surface, wrap the pipeline as a `render_topology` tool alongside `get_topology` / `analyze_risks` (see `mcp-go-server-building`). Input: scope + level (HLD/MLD/LLD). Output: draw.io/SVG + the topology JSON. Read-only; same Managed Identity; no write path.

## Done when

A scheduled run refreshes the Confluence page with version history, posts a topology+severity diff, fails closed on a completeness-cross-check discrepancy, and uses only read-only Managed Identity auth.
