# MVP Backlog (Brownfield Delta)

Status legend: **EXISTS** (harness already does it), **EXTEND** (modify existing code), **NEW** (build), **DONE ✓** (shipped). Estimate class: S (≤1 day), M (2–4 days), L (1–2 weeks).

> **Progress (2026-07-05):** Epics 1–4 (intake, classifier, scaffold + section
> validator, contract lint), Epic 8 (Readiness Engine incl. state enforcement),
> and Epic 9 (export round-trip) are **DONE ✓** — see `aapctl intake|classify|
> scaffold|sections|readiness|export`. The prompt-injection and memory-citation
> proof gates shipped with Epic 8 follow-up. Remaining: Epic 5 identity
> provisioning spike, Epic 6 consistency check, Epic 7 golden-suite population,
> Epics 10–12, and the two non-BA templates.

## Epic 1 — Agent Intake and Catalog
| Story | Status | Est |
|---|---|---|
| Define `agent-intake.schema.json` (required fields per AC-001) | NEW | S |
| `aapctl intake validate` command | EXTEND (`cmd/aapctl`, `internal/runtime/loader.go`) | S |
| Catalog record = manifest + intake in git; lifecycle via manifest `status` enum | EXISTS (schema) / EXTEND (docs) | S |

## Epic 2 — Autonomy and Risk Classifier
| Story | Status | Est |
|---|---|---|
| Classification questionnaire → level 1–5 + risk tier (deterministic rules) | NEW | M |
| Sign-off fields required for level >3; recorded as audit events | EXTEND (audit EXISTS) | S |

## Epic 3 — Blueprint Generator
| Story | Status | Est |
|---|---|---|
| Folder scaffold: 12 artifacts from template + intake | NEW | M |
| Section validator for Markdown artifacts (per artifact-schemas.md) | NEW | M |

## Epic 4 — MCP Tool Contract Generator
| Story | Status | Est |
|---|---|---|
| Contract YAML validation + example-invocation lint | EXISTS (`aapctl validate`, harness) | — |
| Contract scaffold command (`aapctl contracts new`) | EXTEND | S |
| Registry index generation (mcp-tool-contracts.md) | NEW | S |

## Epic 5 — Identity and Permission Mapper
| Story | Status | Est |
|---|---|---|
| Validate against `agent-identity-spec.schema.json` | NEW (schema added) | S |
| Scaffold from intake (scopes derived from tool contracts' `permissions_required`) | NEW | M |
| Entra Agent ID mapping spike (per runtime-verification-notes: managed identity per agent_id, local dev fallback) | NEW (spike) | M |

## Epic 6 — Data and Evidence Mapper
| Story | Status | Est |
|---|---|---|
| Validate against `data-evidence-contract.schema.json` | NEW (schema added) | S |
| Consistency check: memory `allowed_classifications` vs domain classifications | EXTEND (memory validation EXISTS) | S |

## Epic 7 — Evaluation Plan Generator
| Story | Status | Est |
|---|---|---|
| 7-category plan scaffold with thresholds from `release-gate-thresholds.md` | NEW | M |
| Approval golden suite scaffolding (N≥50 case categories) | EXTEND (gate EXISTS, suite structure NEW) | M |

## Epic 8 — Readiness Engine (hero epic)
| Story | Status | Est |
|---|---|---|
| Rubric config file (weights, checks, critical blockers) versioned in repo | NEW | S |
| `aapctl readiness <agent-dir>` → `readiness-report.json` (schema added) | NEW | L |
| Consume `aapctl prove` gate results as harness-check inputs | EXTEND (`internal/runtime/proof.go`) | M |
| State enforcement: manifest cannot move to `active` without Pass verdict | EXTEND (`internal/runtime/engine.go`) | M |
| Markdown rendering of readiness report | NEW | S |

## Epic 9 — Artifact Export and Re-import
| Story | Status | Est |
|---|---|---|
| Deterministic export ordering + manifest of hashes | NEW | M |
| Re-import reproduces identical check results (AC-009 round-trip) | NEW | M |

## Epic 10 — Security/Governance Review View
| Story | Status | Est |
|---|---|---|
| ASI01–ASI10 checklist template + completeness computation | NEW | M |
| Review view render: high-risk actions, classifications, gates, audit obligations | NEW | S |

## Epic 11 — Compliance Evidence Map
| Story | Status | Est |
|---|---|---|
| Evidence map template + generation (AI Act role, ISO 42001 fields, NIST mapping) | NEW | M |

## Epic 12 — Admin and Configuration
| Story | Status | Est |
|---|---|---|
| Rubric/threshold config loading and versioning | NEW | S |
| CI job: validate all agents' folders on PR (GitHub Actions, OIDC) | NEW | S |

## Sequencing

Week 1–2: Epics 1, 2, 4 (intake, classifier, contract scaffold). Week 2–4: Epics 3, 5, 6, 7 (generators). Week 4–6: Epics 8, 9 (readiness + round-trip), then 10–12. Templates (see `templates/`) are built alongside as the test fixtures — BA Agent first since its manifest and contracts already exist.
