# Template: Business Analyst Agent

Status: reference template #1 — furthest along; manifest and contracts already exist.

| Field | Value |
|---|---|
| Primary job | Requirements elicitation, gap analysis, acceptance criteria, traceability |
| Default autonomy | Level 2 (Drafting); Level 3 for system updates (e.g., writing to Jira) |
| Risk tier | Medium (client-confidential inputs; no production impact) |
| Existing assets | `examples/ba-agent.manifest.yaml`; `tool-contracts/get_project_context.contract.yaml`, `search_knowledge_base.contract.yaml`, `create_requirements_draft.contract.yaml`; `docs/ba-agent-proof-flow.md` |

## Tools
| Tool | Action type | Approval boundary |
|---|---|---|
| get_project_context | read | none |
| search_knowledge_base | read | none |
| create_requirements_draft | draft-write | soft |
| (Phase 2) update_work_item | external write | hard |

## Identity
Principal: agent-identity per `agent_id` (Entra Agent ID pattern). Scopes: read on knowledge sources, draft-write on requirements store, no direct write to work-item systems until Phase 2. Full spec generated per `schemas/agent-identity-spec.schema.json`.

## Data & evidence
Domains: project context (authoritative: engagement repo), knowledge base (read-only), requirements drafts (actionable). Evidence rule: every factual claim cites a source; output separates source-backed facts / assumptions / open questions / risks / recommendations (proof-flow step 11). Uncited output behavior: `downgrade-to-assumption`.

## Evaluation focus
Golden suite: elicitation completeness, traceability correctness, citation accuracy. Prompt-injection: instructions embedded in retrieved requirements documents must not alter tool access. Approval golden suite covers draft-creation vs external-write boundary.

## Gap to readiness Pass
Missing vs rubric: intake record, identity spec, data-evidence contract, eval plan doc, ASI checklist, compliance map, readiness run. Contracts and manifest validate today.
