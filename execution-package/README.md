# Execution Package — AaraMinds Agent Factory MVP

Companion to `governance/AaraMinds_Agent_Platform_BRD_v2.1.docx`. This package maps the greenfield BRD onto the existing AAP platform assets and makes the MVP buildable.

**Position in the document set (see `/DOCUMENT-MAP.md`):** this package governs the Agent Factory layer only. The runtime substrate — manifests, tool contracts, approval boundaries, memory, audit, eval gate — is governed by `governance/PRD_AaraMinds_Agent_Platform_v1.3.md`, which this package consumes but does not override. On data shape, `schemas/` wins.

## Contents

| File | Purpose | BRD anchor |
|---|---|---|
| `phase0-validation-plan.md` | Stakeholder interviews, DG-001 packaging decision, Phase 0 exit review | §21.1, §28.1 |
| `mvp-prd.md` | Functional spec, domain model, release criteria | §18, §25 |
| `readiness-scoring-rubric.md` | Rubric v0.1: weights, decision rules, critical blockers, harness wiring | BR-010, AC-008 |
| `artifact-schemas.md` | Validation specs for all 12 MVP artifacts; index of JSON schemas | AC-003/004/005/007/009 |
| `mvp-backlog.md` | 12 epics, stories marked EXISTS / EXTEND / NEW, sequencing | §20 Phase 1 |
| `reference-architecture.md` | Component view, build-vs-adopt decisions, deliberate non-existence list | §14, §27 |
| `templates/` | BA, Scrum Master, Migration QA agent template specs | AC-010, §26 |

New JSON schemas added to `../schemas/`: `agent-identity-spec.schema.json`, `readiness-report.schema.json`, `data-evidence-contract.schema.json` (alongside the six existing AAP schemas).

## Brownfield mapping summary

The BRD was written greenfield; this package reconciles it with what already exists:

| BRD requirement | Existing asset | Delta |
|---|---|---|
| BR-005 MCP tool contracts | `schemas/mcp-tool-contract.schema.json`, 3 BA contracts, `aapctl validate` lint | Scaffold command + registry index |
| BR-008 approval boundaries | Manifest `approval_boundaries`, approval golden suite gate (N≥50) | Classification questionnaire feeding it |
| BR-009 evaluation plan | `docs/release-gate-thresholds.md`, `eval-run.schema.json` | 7-category plan generator |
| BR-010 readiness scoring | `aapctl prove` gate results | Readiness Engine (hero epic) composing them per rubric |
| BR-006 identity spec | Open item in `docs/runtime-verification-notes.md` | New schema + Entra Agent ID spike |
| BR-017 traces/audit | `otel.go`, `audit-event.schema.json`, tamper-evident audit | Collector/Grafana validation before prod |
| AC-010 templates | `examples/ba-agent.manifest.yaml` + proof flow | Scrum Master + Migration QA from scratch |

## Order of work

1. Phase 0 (`phase0-validation-plan.md`) — interviews + DG-001. **Nothing else is funded until this exits.**
2. Phase 1 build per `mvp-backlog.md` sequencing; BA Agent template is the running test fixture.
3. Pilot per BRD §21.2; calibrate rubric; then Phase 2 decisions (gateway adoption, PostgreSQL catalog, console).
