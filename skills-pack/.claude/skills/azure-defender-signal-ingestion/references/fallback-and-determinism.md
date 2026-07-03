# Fallback and the determinism boundary

Stage 4. Defender is authoritative but external, licensed, and non-deterministic. antr's engine is deterministic, license-free, and always available. Use each for what it's good at.

## The fallback rule

antr must produce a correct exposure verdict **with or without** Defender:

```
if defender_cspm_licensed(sub) and attackpaths_returned:
    exposure = consume(defender)          # don't recompute
else:
    exposure = antr_engine.analyze(sub)   # deterministic fallback
report.label_provenance(sub)              # "Defender CSPM" | "antr engine"
```

The large fraction of subscriptions on free foundational CSPM have **no** attack-path data — for them antr's engine is the only source, which is precisely antr's reason to exist.

## The determinism boundary

| Property | Defender | antr engine |
|---|---|---|
| Reproducible for the same input | No (scores evolve) | **Yes** (fixture-tested, twin-checked) |
| Use in a hard CI gate | No | **Yes** |
| Use for enrichment / prioritization | **Yes** | yes |
| Licensed / always available | CSPM plan only | always |

So: **the CI exposure gate of record is antr's deterministic engine.** Defender enriches the report (attack paths, blast radius, EASM confirmation) and prioritizes, but a changing first-party score must never decide a reproducible pass/fail — that would make the gate flap between runs.

## Why keep computing at all when Defender exists

1. Coverage — free-CSPM subscriptions.
2. Determinism — a verdict you can fixture-test, diff, and gate in CI.
3. Explainability — the exact NSG-rule + route + path as evidence, vs Defender's proprietary score.
4. Additive findings — CIDR overlap, missing tier segmentation.
5. License-free, agent-native (MCP) delivery.

This is antr's defensible wedge; ingestion makes it *complement* Defender rather than redundantly recompute it.

## Done when

Exposure is sourced from Defender where licensed and from the engine otherwise, the CI gate of record is the deterministic engine, Defender scores never decide a hard gate, and provenance is labeled.
