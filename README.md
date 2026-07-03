# AaraMinds Platform

This repository is the canonical AaraMinds workspace and the implementation home for the AaraMinds Agent Platform (AAP).

It was copied from `/home/raja/projects/aaraminds` without symlinks. After migration, this folder is the source of truth.

## Layout

```text
platform/       AAP local runtime proof harness
schemas/        JSON schemas required by PRD v1.3
examples/       BA Agent manifest and sample tool contract
tool-contracts/ Tool contracts used by the proof harness
docs/           Proof flow, thresholds, runtime verification notes
governance/     PRD, guardrails, sales proof pack, blocked actions
skills-pack/    Engineering skills, agents, MCP server, validation pack
instruction-os/ Communication personas and skills
diagrams/       Architecture and product diagrams
```

## Quick Proof

```bash
cd platform
go test ./...
go run ./cmd/aapctl prove
```

The proof command writes `out/proofs/phase1-proof.json` at the repository root.
