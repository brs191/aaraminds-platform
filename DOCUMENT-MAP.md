# Document Map

Single entry point for the AaraMinds Platform document set. When documents disagree, the one higher in this hierarchy wins for its layer; `schemas/` always wins on data shape.

## Authority hierarchy

| Level | Document | Layer it governs | Status |
|---|---|---|---|
| 1 | `governance/AaraMinds_Agent_Platform_BRD_v2.1.docx` | Business case: market, positioning, scope, roadmap, decision gates (DG-001) | Draft for review, 2026-07-04 |
| 2 | `governance/PRD_AaraMinds_Agent_Platform_v1.3.md` | Runtime substrate: manifests, tool contracts, approval boundaries, memory, audit, eval gate, four hard proofs | Buildable spec baseline, 2026-07-03 |
| 3 | `execution-package/mvp-prd.md` (+ rubric, backlog, schemas doc, plans) | Agent Factory layer: intake → classification → generation → readiness verdict | Draft v0.1, 2026-07-05 |
| 4 | `docs/` (proof flow, release-gate thresholds, runtime verification notes) | Implementation-facing notes for the harness in `platform/` | Active |
| — | `schemas/*.schema.json` | Machine contracts of record, shared by levels 2–4 | Enforced by `aapctl validate` |
| — | `governance/readiness-rubric.yaml` | Canonical readiness scoring config (weights, thresholds, critical checks) | Enforced by `aapctl readiness`; the narrative in `execution-package/readiness-scoring-rubric.md` is rationale only |

## How the layers reconcile

The BRD (greenfield, market-facing) sequences Factory first and defers runtime operations to Phase 4. The PRD (internal, build-facing) already built a runtime **proof harness** — that is not the deferred Phase 4 runtime console; it is the design-time proof substrate the Factory's Readiness Engine consumes (`aapctl prove` gate results feed rubric areas 2, 3, 5, 6, 7, 9). Division of labor:

- **PRD v1.3** owns: what the harness enforces and proves (manifest, contracts, boundaries, memory, audit, eval gate).
- **execution-package** owns: how an agent idea becomes a governed, readiness-scored artifact folder.
- **BRD v2.1** owns: why this exists, for whom, what is in/out of scope, and the funding/decision gates.

## Reading order

New to the project: `README.md` → this file → BRD §1/§18 → `execution-package/README.md` → PRD §3 (four hard proofs) → `docs/ba-agent-proof-flow.md`.

Building the Factory: `execution-package/mvp-backlog.md` → `readiness-scoring-rubric.md` → `artifact-schemas.md` → PRD §10–§17 for the substrate contracts.

## Other documents

- `Ranking.md` (root) — master ranking of skills, personas, agents, tools. Canonical inventory.
- `governance/` working docs — GTM plans, critical analysis (2026-06-03), guardrails, blocked actions, traceability design, session logs.
- `governance/archive/` — superseded material: BRD v2.0, Critical Analysis 2026-05-21, AaraMind_Factory.md (historical snapshot), Repo Context Platform design (separate product, parked).
- `skills-pack/`, `instruction-os/` — governed by their own CLAUDE.md files; not part of the AAP spec chain.

## Maintenance rule

A document enters `governance/archive/` the day it is superseded. Nothing in `archive/` is authoritative. Update this map whenever a document is added, versioned, or archived.
