# AaraMinds Agent Factory — MVP PRD (v0.1)

Derived from BRD v2.1 (§10.1, §12, §18, §25). Brownfield: builds on the existing AAP harness in `platform/`, schemas in `schemas/`, and gates in `docs/release-gate-thresholds.md`.

**Scope boundary vs PRD v1.3:** `governance/PRD_AaraMinds_Agent_Platform_v1.3.md` is authoritative for the runtime substrate (manifest schema §10, tool contract standard §11, approval model §12, memory §13, identity §14, eval gate §16, audit §17). This document specifies only the Factory layer on top of it. Where the two touch (e.g., manifest fields, contract fields), PRD v1.3 and `schemas/` win.

## 1. Problem and outcome

Agent ideas today become production candidates without a repeatable path through classification, contracts, identity, evaluation, and readiness. The MVP outcome: **an agent idea in, a validated artifact folder and a trusted Agent Readiness Report out** — with every verdict computed from verifiable checks.

## 2. Users

Primary: Enterprise AI Architect, Engineering Lead. Secondary: Security Reviewer, Compliance Lead, Business Owner. (RACI: BRD v2.1 §22.)

## 3. What the MVP is / is not

**Is:** intake → classification → artifact generation → validation → readiness verdict → export. CLI-first (`aapctl`), with generated Markdown/JSON/YAML artifacts in a repo folder per agent.
**Is not:** runtime execution, gateway enforcement, approval workflow UI, catalog web app, A2A. (BRD §10.2/§10.3.)

## 4. Functional specification

| # | Capability | Spec | Existing base |
|---|---|---|---|
| F1 | Agent intake | Structured intake file (`agent-intake.yaml`) with required fields; `aapctl validate` rejects incomplete intake (AC-001) | NEW; validation pattern exists in `internal/runtime/loader.go` |
| F2 | Autonomy & risk classification | Level 1–5 + risk tier from intake answers (action risk, data sensitivity, reversibility, user/financial/production impact); levels >3 require sign-off fields (AC-002) | NEW; boundary vocabulary exists in manifest schema |
| F3 | Blueprint generation | Generate the 12-artifact folder from intake + template (AC-003) | Partial: `examples/ba-agent.manifest.yaml`, 3 contracts exist |
| F4 | Tool contract generation | Scaffold contract YAML pinned to `schemas/mcp-tool-contract.schema.json`; lint incl. example-invocation validation (AC-004) | EXISTS: `aapctl contracts` / `mcp-tools`, harness lint |
| F5 | Identity spec generation | Scaffold + validate against `schemas/agent-identity-spec.schema.json` (AC-005) | NEW schema (this package); Entra mapping open per `runtime-verification-notes.md` |
| F6 | Data & evidence contract | Scaffold + validate against `schemas/data-evidence-contract.schema.json` (AC-006 input) | NEW schema (this package); memory citation gate EXISTS |
| F7 | Evaluation plan generation | 7-category plan with thresholds referencing `release-gate-thresholds.md` (AC-007) | Thresholds EXIST; plan generator NEW |
| F8 | Readiness engine | `aapctl readiness <agent-dir>` → `readiness-report.json` per rubric; enforces state: below threshold cannot set manifest `status: active` (AC-008) | NEW; consumes existing `aapctl prove` gate results |
| F9 | Export / re-import round-trip | Deterministic folder export; import reproduces identical check results (AC-009) | Partial: loader handles YAML/JSON |
| F10 | Security review view | Rendered checklist: high-risk actions, ASI mapping, classifications, gates, audit obligations (AC-011) | NEW (Markdown render acceptable for MVP) |
| F11 | Compliance evidence map | Generated per agent (AC-012) | NEW |
| F12 | Templates | BA, Scrum Master, Migration QA generate complete folders passing readiness (AC-010) | BA partial (manifest + 3 contracts EXIST) |

## 5. Domain model

System of record for governed agents. Entities and their current representation:

| Entity | Representation | Status |
|---|---|---|
| Agent / AgentVersion | `agent-manifest` (agent_id, manifest_version, status lifecycle) | EXISTS |
| AgentTemplate | template folder under `templates/` | NEW |
| ToolContract / Version | `mcp-tool-contract` YAML, pinned in manifest `allowed_tools` | EXISTS |
| AgentIdentitySpec | `agent-identity-spec.schema.json` | NEW |
| DataSourceMapping | `data-evidence-contract.schema.json` | NEW |
| ApprovalRule | manifest `approval_boundaries` + `approval-request.schema.json` | EXISTS |
| EvaluationPlan | `evaluation-plan.md` (section-validated) | NEW |
| EvaluationRun | `eval-run.schema.json` | EXISTS |
| ReadinessReport | `readiness-report.schema.json` | NEW |
| RiskItem | `risk-register.md` (schema in v0.2) | NEW |
| ComplianceEvidenceMap | `compliance-evidence-map.md` | NEW |
| ReviewerSignoff | readiness report `autonomy.signoffs` + audit event | NEW (audit EXISTS) |
| RuntimeConnector | deferred to Phase 4 (`runtime-verification-notes.md` decisions open) | DEFERRED |
| GatewayPolicy | deferred to Phase 2 gateway adoption | DEFERRED |
| AuditEvent | `audit-event.schema.json` | EXISTS |

MVP persistence: filesystem + git is the system of record (repo folder per agent). PostgreSQL catalog is Phase 2 — do not build a database before the artifact model stabilizes.

## 6. Release criteria

All 12 BRD v2.1 acceptance criteria pass; three templates run end-to-end (AC-010); pilot exit criteria per BRD §21.2 measured. Readiness rubric v0.1 calibrated or consciously revised.

## 7. Non-goals repeated for emphasis

No web UI in MVP (CLI + rendered Markdown). No runtime. No gateway. No database. Each is a deliberate deferral with a named phase.
