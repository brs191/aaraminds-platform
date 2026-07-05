# MVP Artifact Schemas

Status: draft v0.1. Implements BRD v2.1 AC-003/004/005/007/009.
Rule: an artifact is **complete** when it passes its schema — never by reviewer judgment alone.

## Machine-validated artifacts (JSON Schema in `schemas/`)

| Artifact | Schema | Status |
|---|---|---|
| Agent manifest | `schemas/agent-manifest.schema.json` | EXISTS (validated by `aapctl validate`) |
| MCP tool contract | `schemas/mcp-tool-contract.schema.json` | EXISTS (validated + example-invocation lint) |
| Evaluation run | `schemas/eval-run.schema.json` | EXISTS |
| Approval request | `schemas/approval-request.schema.json` | EXISTS |
| Audit event | `schemas/audit-event.schema.json` | EXISTS |
| Memory record | `schemas/memory-record.schema.json` | EXISTS |
| Agent identity spec | `schemas/agent-identity-spec.schema.json` | **NEW — added with this package** |
| Readiness report | `schemas/readiness-report.schema.json` | **NEW — added with this package** |
| Data & evidence contract | `schemas/data-evidence-contract.schema.json` | **NEW — added with this package** |

## Markdown artifacts (required-section specs)

Each `.md` deliverable has a required-section list. The Readiness Engine validates presence and non-emptiness of each section (frontmatter key `sections_validated: true` is written by the validator, never by hand).

### agent-blueprint.md
Required: Business Problem; Users & Stakeholders; Expected Outcomes; Autonomy Level & Justification; Workflow Overview; Tools (must reference contract files); Data Domains (must reference data-evidence contract); Identity (must reference identity spec); Security & Approval Boundaries; Evaluation Approach; Operations & Ownership; Non-Goals.

### system-prompt.md
Required: Role & Objective; Evidence & Citation Rules; Prohibited Behaviors (must include prompt-injection refusal rule); Output Structure (must separate source-backed facts / assumptions / open questions / risks / recommendations — matches BA proof flow step 11); Escalation Rules.

### workflow-design.md
Required: Trigger & Inputs; Step Graph (numbered, each step names tool or model call); Approval Points (must match manifest `approval_boundaries`); Failure Handling per Step; Completion Criteria.

### mcp-tool-contracts.md
Index page only — one row per contract YAML with tool_name, contract_version, action_type, approval_boundary. Truth lives in `tool-contracts/*.contract.yaml` validated against the JSON schema.

### agent-identity-spec.md
Human-readable rendering of `agent-identity-spec.json`. Required: Principal; Credential Pattern; Scopes table with justification; Conditional Access; Lifecycle & Owner.

### data-and-evidence-contract.md
Human-readable rendering of `data-evidence-contract.json`. Required: Domain table; Evidence Rules; Staleness/Conflict notes.

### security-governance-checklist.md
Required: one subsection per OWASP ASI01–ASI10 with control statement and status (addressed / mitigated / not-applicable-with-reason); RBAC summary; Data Classification summary; Audit Obligations; Kill-Switch Path.

### evaluation-plan.md
Required sections = the 7 categories: Golden Tests (N≥50 approval golden suite per release-gate thresholds); Tool Accuracy; Retrieval/Evidence & Citations; Safety & Prompt Injection; Latency; Cost; Regression. Each names its threshold and `benchmark_ref`.

### readiness-report.md
Human-readable rendering of `readiness-report.json` (schema above). Generated only — hand-authored readiness reports are invalid by definition.

### compliance-evidence-map.md
Required: AI Act Role Assessment (deployer/provider + reasoning + legal-review flag); ISO 42001 Registry Fields (purpose, owner, lifecycle state, risk tier, review date); NIST AI RMF Function Mapping (Govern/Map/Measure/Manage — one line each); Open Compliance Questions.

### implementation-backlog.md
Required: Epics table; Stories with acceptance criteria; Dependencies; Estimate class.

### risk-register.md
Required: table with risk_id, description, likelihood, impact, mitigation, owner, status. (Candidate for JSON schema in v0.2.)

## Round-trip rule (AC-009)

`export` writes the folder; `import` re-validates every artifact and must reproduce identical check results in the readiness report. Any divergence fails the export gate.
