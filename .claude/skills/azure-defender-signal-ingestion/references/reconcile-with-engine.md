# Reconciling Defender signals with antr's deterministic findings

Stage 3. The point of ingestion is one coherent risk picture, not two overlapping ones. Join Defender signals to antr findings by resource id, dedupe, and keep what each uniquely provides.

## Join by resource id

Both sides key on Azure resource IDs (Defender via `graphComponent.entities`; antr findings via the engine's `resource`/ARM id after V4-07). Build a per-resource view:

```
resource_id → { defender: [attack_paths, exposure], antr: [findings] }
```

## Dedupe — one verdict per exposure

If Defender already asserts "internet-exposed VM X" and antr's engine would compute the same exposure, **emit one reconciled finding** citing both sources — not two. Rule of thumb:

| Signal | Source of record |
|---|---|
| Internet exposure / attack path (Defender CSPM licensed) | **Defender** (consume); antr enrichment optional |
| Internet exposure (no Defender CSPM) | **antr engine** (fallback) |
| CIDR overlap | **antr engine only** (Defender doesn't emit) |
| Missing tier segmentation | **antr engine only** |
| Reproducible CI gate verdict | **antr engine** (deterministic) regardless of Defender |

## Map Defender risk → antr severity (for display only)

Translate Defender's `riskCategories` / `potentialImpact` into antr's bucket scale (Critical/High/Medium/Info/Clean) for a consistent report and the visualization overlay — but keep the original Defender values in the evidence, and never let the mapped value drive a hard gate (it isn't reproducible).

## Keep antr's additive value visible

CIDR overlap and missing tier segmentation are antr-only. Surface them prominently when Defender is present — they are the reason the engine still runs alongside a licensed Defender.

## Done when

Each resource has one reconciled risk view (deduped by id), provenance is labeled per finding, Defender risk is mapped to antr severity for display only, and antr-only findings (CIDR/segmentation) are preserved.
