# AaraMinds Platform

This repository is the canonical AaraMinds workspace and the implementation home for the AaraMinds Agent Platform (AAP).

It was copied from `/home/raja/projects/aaraminds` without symlinks. After migration, this folder is the source of truth.

## Documents

Start with `DOCUMENT-MAP.md` — it defines which document is authoritative for which layer (BRD → PRD → execution package → docs → schemas) and the reading order.

## Layout

```text
platform/          AAP local runtime proof harness (Go)
schemas/           JSON schemas — machine contracts of record (9)
examples/          BA Agent manifest and sample tool contract
tool-contracts/    Tool contracts used by the proof harness
docs/              Proof flow, release-gate thresholds, runtime verification notes
governance/        BRD v2.1, PRD v1.3, GTM, guardrails, blocked actions (+ archive/)
execution-package/ Agent Factory MVP: PRD, readiness rubric, backlog, plans, templates
skills-pack/       Engineering skills, agents, MCP server, validation pack
instruction-os/    Communication personas and skills
diagrams/          Architecture and product diagrams
Ranking.md         Master ranking — skills, personas, agents, tools
DOCUMENT-MAP.md    Document authority hierarchy and reading order
```

## Quick Proof

```bash
cd platform
go test ./...
go run ./cmd/aapctl prove
```

The proof command writes `out/proofs/phase1-proof.json` at the repository root.

To inspect the live OpenTelemetry projection locally, run:

```bash
go run ./cmd/aapctl prove -otel -otel-exporter stdout
```

For a collector, use `-otel-exporter otlp -otel-endpoint localhost:4317` or the matching `AAP_OTEL_*` / `OTEL_EXPORTER_OTLP_ENDPOINT` environment variables.
The projection emits one trace per proof run and links governed spans back to audit events with `aap.audit_event_id`.
