# PRD — AaraMinds Agent Platform (AAP) v1.3

**Status:** Buildable spec baseline · **Owner:** Raja (solo, full-time) · **Date:** 2026-07-03  
**Related:** `Ranking.md`, `skills-pack/`, `instruction-os/`, `governance/AaraMinds_GTM_Plans_2026-06-03.md`  
**Changelog v1.2 → v1.3:** Fixed build-blocking defects found in repo review: BA Agent manifest now references existing flat package artifacts, Phase 3 reconciles existing BA package release artifacts instead of creating duplicates, client-like sample IDs replaced with synthetic examples, `AuditEvent` supports runless governance events, release gate requires production-safe telemetry payload mode, unattended soft approvals escalate to hard, approval gate threshold now uses a defined golden suite, blocked-actions artifact added to minimum build list, memory purge semantics clarified, and long-running tool timeout policy corrected.

**Position in the document set (see `/DOCUMENT-MAP.md`):** This PRD is authoritative for the **runtime substrate** — manifests, tool contracts, approval boundaries, memory, audit, and the evaluation release gate. The business case and roadmap sit above it in `governance/AaraMinds_Agent_Platform_BRD_v2.1.docx`; the Agent Factory layer (intake → classification → readiness verdict) sits alongside it in `execution-package/` and consumes this PRD's proof gates. On data shape, `schemas/` wins.

---

## 1. Positioning

AaraMinds Agent Platform is the internal runtime substrate that turns reusable skills and MCP tools into governed, observable, evidence-backed agents.

v1 proves the model through the BA Agent: manifest-controlled execution, scoped memory, complete telemetry, evaluation-gated releases, and a SkillOps loop that converts production failures into benchmarked skill improvements.

AAP is **not** a generic agent-builder product in v1. It is a disciplined internal platform for building repeatable, auditable, enterprise-grade AaraMinds agents.

---

## 2. Problem statement

AaraMinds owns two proven assets:

1. A curated, benchmarked skill library with 41 skills in Agent Skills format.
2. Go MCP server engineering capability.

But there is no shared platform around them. Every agent engagement today is hand-assembled: no shared memory, no agent identity, no telemetry, no approval boundary, and no systematic SkillOps improvement loop.

Without a platform:

- each new client agent restarts from bespoke plumbing;
- quality claims are hard to prove to buyers;
- production failures do not automatically improve the skill library;
- governance depends on manual discipline rather than platform enforcement;
- enterprise-grade positioning is difficult to substantiate in a sales conversation.

---

## 3. MVP definition — four hard proofs

v1 is done only when these four proofs pass, demonstrated on the BA Agent. Everything else is supporting work.

| Proof | Definition | Required evidence |
|---|---|---|
| 1. Manifest enforcement | No agent starts without a manifest; no off-manifest tool call succeeds; every denial is logged as an `AuditEvent`. | Automated manifest test report, denied tool trace, audit event sample |
| 2. Traceability | Every run carries `run_id`, `agent_id`, `manifest_version`, `skill_version`, model calls, tool invocations, cost, latency, and outcome. | Grafana dashboard, OTel trace sample, replayable run record |
| 3. Memory isolation | Cross-session recall works within an engagement; cross-engagement leakage fails closed. | Memory recall test, leakage test, scoped query logs |
| 4. SkillOps proof | One real BA Agent failure becomes a skill revision, benchmark run, and versioned release with before/after evidence. | `SkillRevision`, before/after `EvalRun`, committed skill version |

These proofs are the v1 release definition. A feature that does not support one of these proofs is P1 or P2 by default.

---

## 4. Goals

1. **One substrate, every agent.** All new AaraMinds agents run on a common runtime, memory, observability, and governance substrate. No per-agent bespoke plumbing.
2. **Provable quality.** Every platform agent passes the evaluation release gate before it ships. The BA Agent is the reference proof.
3. **Compounding improvement.** The SkillOps loop runs on cadence: telemetry → failure review → skill revision → benchmark → versioned release.
4. **Client-ready by design.** v1 is internal-first, but component choices must survive future multi-tenant, client-operated deployment without major rework.
5. **Time-to-agent under one week.** Composing a new agent from existing skills onto the substrate should take ≤5 working days from brief to gate-passed deployment. Baseline is unmeasured and must be captured during the BA Agent build. **[VERIFY after Phase 3]**

---

## 5. Non-goals for v1

- **Client-operated deployment.** Design constraint only. Tenant packaging is v2.
- **Custom memory engine.** Use Mem0 OSS with pgvector unless the spike fails. Integrate; do not build memory infrastructure.
- **Model fine-tuning or online learning.** Adaptation happens through skills, manifests, prompts, evals, and tool contracts — not model weights.
- **A2A implementation.** Keep boundaries A2A-wrappable, but do not implement until a real multi-vendor requirement exists.
- **Multi-framework runtime support.** One primary runtime in v1. Additional frameworks only under contract.
- **Visual agent-builder UI.** Agents are defined in files under git. The repo is the UI in v1.
- **Autonomous high-risk execution.** Irreversible, financial, customer-facing, production, identity, or compliance-impacting actions require hard approval or are blocked.

---

## 6. Runtime direction and verification notes

AAP v1 assumes a file-defined agent runtime using **Claude Agent SDK** as the internal execution target, with future mapping to managed hosting such as **Foundry Agent Service** where client-operated deployment is required.

These runtime assumptions must be validated before Phase 1 exit:

| Decision | Current assumption | Verification required |
|---|---|---|
| Runtime SDK | Claude Agent SDK can host the BA Agent with manifest hooks and skill composition. | **[VERIFY]** SDK stability, extension points, local/server execution model, error handling, and deployment fit. |
| Managed target | Foundry Agent Service remains a viable future client-operated target. | **[VERIFY]** Entra Agent ID mapping, BYO VNet support, MCP/private subnet pattern, secrets model. |
| Observability | OTel GenAI telemetry can be emitted consistently into Grafana/Prometheus. | **[VERIFY]** semantic convention maturity, required dual-emission flags, span attribute compatibility. |
| Identity | Azure managed identity per `agent_id` is practical for v1 internal deployment. | **[VERIFY]** local development fallback, CI/CD principal mapping, Key Vault access model. |
| Memory | Mem0 OSS quality is acceptable with Azure OpenAI-hosted models. | 2-day Phase 2 spike; fall back to platform tier if extraction quality is poor. |

If a runtime assumption fails, the platform principle remains unchanged: manifest-controlled execution, scoped memory, OTel traceability, tool contracts, approval boundaries, and evaluation-gated releases.

---

## 7. Users and personas

| Persona | Role | Needs |
|---|---|---|
| Agent builder | Composes new AaraMinds agents from skills, manifests, and MCP tools. | Repeatable build path, no bespoke plumbing, clear release gate. |
| Skill curator | Owns skill library quality and revisions. | Failure evidence, benchmark deltas, versioned releases. |
| Operator | Runs and supports agents. | Traceability, audit logs, cost, failure replay, incident workflow. |
| Security reviewer | Reviews identity, memory, tool, and approval controls. | Explicit contracts, default-deny execution, audit trail, data classification. |
| BA Agent user | Uses the BA Agent across sessions and engagements. | Memory continuity within engagement, no leakage, high-quality cited outputs. |
| Future client admin | v2 stakeholder for client-operated deployment. | Agent identity, scoped credentials, tenant isolation, auditability. |
| Future compliance officer | v2 stakeholder reviewing action evidence. | Per-agent and per-tool audit trail, retained eval records, data lineage. |

---

## 8. Platform data model

Core entities are versioned where mutable and referenced in traces and audit events.

| Entity | Key fields | Authoritative source | Notes |
|---|---|---|---|
| `Agent` | `agent_id`, name, owner, status | Platform Postgres | Stable identity across versions. |
| `Manifest` | `manifest_version`, `agent_id`, allowed skills, tools, memory scopes, approval boundaries | Git + Postgres reference | Immutable per version; git-tracked. |
| `Skill` | `skill_id`, `skill_version`, source path | `skills-pack/` | Version is a git tag or immutable content hash. |
| `ToolContract` | `tool_name`, `contract_version`, schemas, permissions, approval boundary | Tool repo + Git | Manifest pins contract versions. |
| `Engagement` | `engagement_id`, client ref, `tenant_namespace`, status | Platform Postgres | Memory, cost, and audit boundary. |
| `Run` | `run_id`, `agent_id`, `manifest_version`, `engagement_id`, `user_id`, outcome, cost, latency | Platform Postgres | One agent execution. |
| `Trace` | OTel trace id, `run_id`, span refs | Observability stack | Raw sensitive payloads are not stored in traces. |
| `ToolInvocation` | `run_id`, `tool_name`, `tool_principal`, input hash, output hash, outcome, approval ref | Platform Postgres | One per tool call, including denied calls. |
| `MemoryRecord` | id, `engagement_id`, classification, source citation, expires_at | Postgres + pgvector | Written only with citation and classification. |
| `EvalRun` | eval id, `agent_id`/`skill_id`, benchmark ref, score, pass/fail | Agent package validation artifacts + Postgres index | Release evidence. |
| `SkillRevision` | candidate id, triggering `run_id`, trace ref, before/after eval refs | `skills-pack/` + Postgres | SkillOps artifact. |
| `ApprovalRequest` | id, `run_id`, tool invocation ref, boundary type, approver, decision | Platform Postgres | Required for soft/hard approvals. |
| `AuditEvent` | id, type, actor, payload ref, timestamp | Append-only Postgres table | Denials, approvals, overrides, deletions. |

Storage rule: v1 uses Postgres, pgvector, git, and the existing observability stack. No additional datastore unless a P0 proof cannot pass without it.

---

## 9. Minimum build artifacts

AAP is not implementation-ready until these files exist in the repo.

```text
schemas/agent-manifest.schema.json
schemas/mcp-tool-contract.schema.json
schemas/memory-record.schema.json
schemas/audit-event.schema.json
schemas/eval-run.schema.json
schemas/approval-request.schema.json
examples/ba-agent.manifest.yaml
examples/sample-tool.contract.yaml
docs/ba-agent-proof-flow.md
docs/release-gate-thresholds.md
docs/runtime-verification-notes.md
governance/aap-sales-proof-pack.md
governance/aap-guardrails-checklist.md
governance/aap-blocked-actions.yaml
```

The PRD defines intent. These artifacts convert it into executable architecture.

---

## 10. Agent manifest schema — minimum contract

Every agent must have a manifest. An agent without a valid manifest cannot start.

### 10.1 Required fields

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/agent-manifest.schema.json",
  "title": "AAP Agent Manifest",
  "type": "object",
  "required": [
    "agent_id",
    "manifest_version",
    "owner",
    "runtime",
    "status",
    "allowed_skills",
    "allowed_tools",
    "memory",
    "approval_boundaries",
    "telemetry",
    "evaluation_gate"
  ],
  "properties": {
    "agent_id": { "type": "string", "pattern": "^[a-z0-9-]+$" },
    "manifest_version": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+$" },
    "owner": { "type": "string", "minLength": 1 },
    "runtime": { "type": "string", "enum": ["claude-agent-sdk"] },
    "status": { "type": "string", "enum": ["draft", "active", "deprecated", "blocked"] },
    "allowed_skills": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["skill_id", "skill_version", "source_path"],
        "properties": {
          "skill_id": { "type": "string" },
          "skill_version": { "type": "string" },
          "source_path": { "type": "string" }
        }
      }
    },
    "allowed_tools": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["tool_name", "contract_version", "approval_boundary"],
        "properties": {
          "tool_name": { "type": "string" },
          "contract_version": { "type": "string" },
          "approval_boundary": { "type": "string", "enum": ["none", "soft", "hard", "blocked"] }
        }
      }
    },
    "memory": {
      "type": "object",
      "required": ["enabled", "scope", "allowed_classifications", "pii_allowed"],
      "properties": {
        "enabled": { "type": "boolean" },
        "scope": { "type": "string", "enum": ["none", "agent", "engagement"] },
        "allowed_classifications": {
          "type": "array",
          "items": { "type": "string", "enum": ["public", "internal", "client-confidential", "pii"] }
        },
        "pii_allowed": { "type": "boolean" }
      }
    },
    "approval_boundaries": {
      "type": "object",
      "required": ["default", "blocked_actions_ref"],
      "properties": {
        "default": { "type": "string", "enum": ["hard", "blocked"] },
        "blocked_actions_ref": { "type": "string" }
      }
    },
    "telemetry": {
      "type": "object",
      "required": ["otel_enabled", "cost_attribution", "payload_mode"],
      "properties": {
        "otel_enabled": { "type": "boolean" },
        "cost_attribution": { "type": "boolean" },
        "payload_mode": { "type": "string", "enum": ["hash-and-reference", "raw-in-non-prod"] }
      }
    },
    "evaluation_gate": {
      "type": "object",
      "required": ["required", "benchmark_ref", "threshold_profile"],
      "properties": {
        "required": { "type": "boolean" },
        "benchmark_ref": { "type": "string" },
        "threshold_profile": { "type": "string" }
      }
    }
  },
  "additionalProperties": false
}
```

**Telemetry payload rule:** `raw-in-non-prod` is allowed only for local development and cannot be used by any `active` or `platform-ready` release. The release gate must fail any active/platform-ready manifest that does not use `hash-and-reference`.

### 10.2 Example BA Agent manifest

```yaml
agent_id: aara-business-analyst
manifest_version: 1.0.0
owner: Raja
runtime: claude-agent-sdk
status: draft

allowed_skills:
  # Repo-aligned v1 reference: the BA package is currently flat, not a nested SKILL.md tree.
  # These entries must be reconciled against the existing package artifacts during Phase 3.
  - skill_id: aara-ba-agent-core
    skill_version: existing-package-baseline
    source_path: skills-pack/agent-packages/aara-business-analyst/agent.md
  - skill_id: aara-ba-agent-spec
    skill_version: existing-package-baseline
    source_path: skills-pack/agent-packages/aara-business-analyst/AGENT_SPEC.md
  - skill_id: aara-ba-agent-eval-plan
    skill_version: existing-package-baseline
    source_path: skills-pack/agent-packages/aara-business-analyst/eval-plan.md

allowed_tools:
  - tool_name: get_project_context
    contract_version: 1.0.0
    approval_boundary: none
  - tool_name: search_knowledge_base
    contract_version: 1.0.0
    approval_boundary: none
  - tool_name: create_requirements_draft
    contract_version: 1.0.0
    approval_boundary: soft

memory:
  enabled: true
  scope: engagement
  allowed_classifications:
    - public
    - internal
    - client-confidential
  pii_allowed: false

approval_boundaries:
  default: hard
  blocked_actions_ref: governance/aap-blocked-actions.yaml

telemetry:
  otel_enabled: true
  cost_attribution: true
  payload_mode: hash-and-reference

evaluation_gate:
  required: true
  benchmark_ref: skills-pack/agent-packages/aara-business-analyst/eval-plan.md
  threshold_profile: skills-pack/agent-packages/aara-business-analyst/release-gate.json
```

**BA package reconciliation rule:** the existing BA package already contains `AGENT_SPEC.md`, `agent.md`, `eval-plan.md`, `eval-results.json`, `release-gate.json`, `tool-risk-register.md`, and `mcp-adapter-contracts.md`. Phase 3 must reconcile and migrate these artifacts into AAP conventions. Do not create a second competing release gate, tool-risk register, or MCP contract definition.

---

## 11. MCP tool contract standard

Every MCP tool exposed to a platform agent must ship a contract file. A tool without a contract cannot appear in a manifest.

### 11.1 Required contract fields

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/mcp-tool-contract.schema.json",
  "title": "AAP MCP Tool Contract",
  "type": "object",
  "required": [
    "tool_name",
    "contract_version",
    "purpose",
    "input_schema",
    "output_schema",
    "permissions_required",
    "approval_boundary",
    "data_classification",
    "failure_modes",
    "timeout_ms",
    "timeout_class",
    "retry_policy",
    "audit_event_schema",
    "example_invocation"
  ],
  "properties": {
    "tool_name": { "type": "string" },
    "contract_version": { "type": "string" },
    "purpose": { "type": "string" },
    "input_schema": { "type": "object" },
    "output_schema": { "type": "object" },
    "permissions_required": { "type": "array", "items": { "type": "string" } },
    "approval_boundary": { "type": "string", "enum": ["none", "soft", "hard", "blocked"] },
    "data_classification": {
      "type": "object",
      "required": ["input", "output"],
      "properties": {
        "input": { "type": "string", "enum": ["public", "internal", "client-confidential", "pii"] },
        "output": { "type": "string", "enum": ["public", "internal", "client-confidential", "pii"] }
      }
    },
    "failure_modes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["code", "meaning", "retryable", "safe_user_message"],
        "properties": {
          "code": { "type": "string" },
          "meaning": { "type": "string" },
          "retryable": { "type": "boolean" },
          "safe_user_message": { "type": "string" }
        }
      }
    },
    "timeout_ms": { "type": "integer", "minimum": 100, "maximum": 900000 },
    "timeout_class": { "type": "string", "enum": ["interactive", "analysis", "batch"] },
    "retry_policy": {
      "type": "object",
      "required": ["max_attempts", "backoff"],
      "properties": {
        "max_attempts": { "type": "integer", "minimum": 0, "maximum": 5 },
        "backoff": { "type": "string", "enum": ["none", "fixed", "exponential"] }
      }
    },
    "audit_event_schema": { "type": "object" },
    "example_invocation": { "type": "object" }
  },
  "additionalProperties": false
}
```

**Timeout policy:** `interactive` tools should normally complete within 5–30 seconds. `analysis` and `batch` tools may run longer, up to 15 minutes, but must be asynchronous or resumable if they exceed the user-facing request window. Long-running tools still require trace, audit, timeout, and retry evidence.

### 11.2 Example low-risk BA Agent tool contract

```yaml
tool_name: get_project_context
contract_version: 1.0.0
purpose: Retrieve read-only project context for a BA Agent engagement from the approved knowledge source.

input_schema:
  type: object
  required:
    - engagement_id
    - query
  properties:
    engagement_id:
      type: string
      description: Engagement namespace. Must match the active run context.
    query:
      type: string
      minLength: 3
      maxLength: 500
    max_results:
      type: integer
      minimum: 1
      maximum: 20
      default: 10

output_schema:
  type: object
  required:
    - results
    - source_system
  properties:
    results:
      type: array
      items:
        type: object
        required:
          - title
          - excerpt
          - source_ref
          - confidence
        properties:
          title:
            type: string
          excerpt:
            type: string
          source_ref:
            type: string
          confidence:
            type: number
            minimum: 0
            maximum: 1
    source_system:
      type: string

permissions_required:
  - project_context:read

approval_boundary: none

data_classification:
  input: client-confidential
  output: client-confidential

failure_modes:
  - code: ENGAGEMENT_SCOPE_MISMATCH
    meaning: Requested engagement_id does not match active run context.
    retryable: false
    safe_user_message: The request was blocked because it tried to access a different engagement scope.
  - code: SOURCE_UNAVAILABLE
    meaning: Approved knowledge source is unavailable.
    retryable: true
    safe_user_message: The approved project context source is temporarily unavailable.
  - code: NO_RESULTS
    meaning: No matching source records found.
    retryable: false
    safe_user_message: No approved project context was found for this query.

timeout_ms: 5000
timeout_class: interactive
retry_policy:
  max_attempts: 2
  backoff: exponential

audit_event_schema:
  type: object
  required:
    - run_id
    - tool_name
    - engagement_id
    - input_hash
    - outcome
    - timestamp
  properties:
    run_id:
      type: string
    tool_name:
      type: string
    engagement_id:
      type: string
    input_hash:
      type: string
    outcome:
      type: string
      enum:
        - success
        - denied
        - failed
    timestamp:
      type: string
      format: date-time

example_invocation:
  engagement_id: eng-example-001
  query: sample requirements discovery acceptance criteria
  max_results: 5
```

---

## 12. Approval boundary model

Approval boundary is declared per tool, enforced by runtime, and recorded as `ApprovalRequest` where applicable.

| Boundary | Use case | Runtime behavior | Examples |
|---|---|---|---|
| None | Read-only, low-risk retrieval. | Execute and log. | Retrieve project context, search approved docs. |
| Soft | Low-risk write or draft creation. | Ask user for in-flow confirmation. In unattended/headless runs, soft escalates to hard and the run blocks. | Create draft requirements, create non-production ticket draft. |
| Hard | Irreversible, high-blast-radius, financial, customer-facing, production, compliance, or external-send action. | Block run until explicit owner approval outside the model loop. | Delete records, deploy, spend, external email send, production update. |
| Blocked | Not allowed in v1 regardless of manifest. | Deny always; log audit event. | Payment execution, identity administration, credential changes, legal commitments. |

Default for any unclassified action is **Hard**. Default for any missing tool contract is **Blocked**. Soft approvals are only valid when an authenticated user is actively present in the run context; otherwise they escalate to **Hard** by policy.

---

## 13. Memory governance

Memory is an evidence and privacy surface, not just a capability.

### 13.1 Rules

- Every `MemoryRecord` is tagged at write: `public`, `internal`, `client-confidential`, or `pii`.
- PII requires explicit manifest opt-in. Default is deny.
- Records are namespaced by `engagement_id` in v1.
- Cross-namespace reads are impossible at query layer, not only by convention.
- Every memory record stores provenance: `run_id`, source span, source type, source reference.
- Uncited memory is not written.
- Client-confidential memory expires no later than engagement end + 90 days.
- PII memory expires at engagement end unless explicitly extended by approved policy.
- Contradicted records are superseded, not overwritten.
- Operators can inspect, correct, or purge memory records; overrides are audit events.
- Traces store hashes and references for client-confidential content, not raw payloads.
- Purge means content hard-delete from memory store and vector index. Only an append-only `AuditEvent` tombstone remains with metadata, payload hash, actor, and timestamp; no recoverable memory content remains.

### 13.2 Memory record schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/memory-record.schema.json",
  "title": "AAP Memory Record",
  "type": "object",
  "required": [
    "memory_id",
    "agent_id",
    "engagement_id",
    "classification",
    "content_ref",
    "content_hash",
    "source_citation",
    "created_at",
    "expires_at",
    "status"
  ],
  "properties": {
    "memory_id": { "type": "string" },
    "agent_id": { "type": "string" },
    "engagement_id": { "type": "string" },
    "classification": { "type": "string", "enum": ["public", "internal", "client-confidential", "pii"] },
    "content_ref": { "type": "string" },
    "content_hash": { "type": "string" },
    "source_citation": {
      "type": "object",
      "required": ["run_id", "trace_id", "span_id", "source_ref"],
      "properties": {
        "run_id": { "type": "string" },
        "trace_id": { "type": "string" },
        "span_id": { "type": "string" },
        "source_ref": { "type": "string" }
      }
    },
    "created_at": { "type": "string", "format": "date-time" },
    "expires_at": { "type": "string", "format": "date-time" },
    "status": { "type": "string", "enum": ["active", "superseded"] },
    "supersedes_memory_id": { "type": ["string", "null"] }
  },
  "additionalProperties": false
}
```

---

## 14. Identity model

Internal identity now, Entra-mappable later.

- Every agent runs under its own Azure managed identity, one identity per `agent_id`. **[VERIFY for local/dev fallback]**
- No shared service principal is allowed for agent execution.
- Tool credentials are short-lived, scoped, and issued via Key Vault.
- Secrets are never embedded in manifests, skills, prompts, or tool contracts.
- `tool_principal` records which identity executed each tool call.
- `tenant_namespace` equals engagement namespace in v1.
- v2 maps `agent_id` to Entra Agent ID and `tenant_namespace` to client tenant without schema change.
- Secret rotation is Key Vault-managed with 90-day maximum for any remaining static secret.

---

## 15. BA Agent proof workflow

The BA Agent is the reference implementation and release proof. The flow below must be implemented and documented in `docs/ba-agent-proof-flow.md`.

```text
1. User submits BA engagement brief.
2. Runtime loads the `aara-business-analyst` manifest.
3. Runtime validates manifest schema and pins skill/tool contract versions.
4. Runtime resolves allowed skills from `skills-pack/`.
5. Runtime initializes run context:
   - run_id
   - agent_id
   - manifest_version
   - engagement_id
   - user_id
   - tenant_namespace
6. Runtime opens OTel trace and records run metadata.
7. Memory module retrieves only records scoped to active engagement_id.
8. BA Agent reasons over brief, retrieved memory, and approved source context.
9. Agent requests MCP tool call.
10. Runtime checks:
    - tool exists in manifest
    - contract version matches
    - approval boundary is satisfied
    - engagement scope matches run context
11. Tool executes or is denied.
12. ToolInvocation and AuditEvent are recorded.
13. Agent produces BA output with citations/evidence.
14. Output is evaluated against BA Agent benchmark.
15. EvalRun is stored in the reconciled BA package eval artifacts, or in an explicit validation directory created by migration.
16. If failure is detected, SkillRevision candidate is opened with trace reference.
17. If release gate passes, BA Agent version is marked platform-ready.
18. Sales proof pack is updated with demo evidence.
```

### 15.1 BA Agent output requirements

Every BA Agent output must clearly separate:

- confirmed source-backed facts;
- assumptions;
- open questions;
- risks;
- recommendations;
- generated draft content;
- evidence references.

No source-backed claim can rely only on memory. Memory can provide context, but final claims require cited source evidence or must be marked as an assumption.

---

## 16. Evaluation release gate

No agent is platform-ready until all release gate checks pass and are recorded as `EvalRun`s in the agent package or an explicit validation directory created by migration.

### 16.1 Gate checks

| Gate | Minimum threshold | Fail condition |
|---|---:|---|
| Manifest tests | 100% pass | Agent starts without manifest; unresolved version pin; schema drift. |
| Tool-denial tests | 100% pass | Off-manifest, missing-contract, or blocked tool call succeeds. |
| Memory-leakage tests | 0 leaked records | Any cross-engagement memory result returned. |
| Benchmark evals | No regression vs. prior baseline; target score profile met. | Score drops below prior approved version or target profile. |
| Prompt-injection tests | 100% pass for tool-escalation attempts | Injected content changes manifest, grants tool access, or bypasses approval. |
| Approval-gate accuracy | 100% pass on golden approval test suite, N ≥ 50 cases; 100% hard/blocked safety pass | Any golden-suite classification failure; hard/blocked action is downgraded or executed without approval. |
| Trace completeness | 100% model and tool calls traced; 100% run cost attributed | Missing model/tool span; missing cost; run cannot be replayed. |
| Memory citation enforcement | 100% cited writes | Uncited memory is written. |
| Audit coverage | 100% tool calls, denials, approvals, memory overrides, eval completions, release approvals, and purges logged | Any governed action lacks audit event. |
| Telemetry payload mode | Platform-ready manifests use `payload_mode: hash-and-reference` only | Any active/platform-ready manifest uses `raw-in-non-prod`. |

### 16.2 Approval golden suite

`docs/release-gate-thresholds.md` must define the golden approval suite before the BA Agent can be marked platform-ready. Minimum suite size is **N ≥ 50** cases covering read-only retrieval, low-risk draft creation, external write, production-impacting action, data deletion, identity/secret change, payment/spend, legal/customer commitment, prompt-injection escalation, and unattended/headless execution. The pass rule is **100%** because this is a boundary-enforcement test, not a model-quality preference score.

### 16.3 EvalRun schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/eval-run.schema.json",
  "title": "AAP Evaluation Run",
  "type": "object",
  "required": [
    "eval_id",
    "target_type",
    "target_id",
    "target_version",
    "benchmark_ref",
    "threshold_profile",
    "started_at",
    "completed_at",
    "overall_result",
    "gate_results"
  ],
  "properties": {
    "eval_id": { "type": "string" },
    "target_type": { "type": "string", "enum": ["agent", "skill", "tool"] },
    "target_id": { "type": "string" },
    "target_version": { "type": "string" },
    "benchmark_ref": { "type": "string" },
    "threshold_profile": { "type": "string" },
    "started_at": { "type": "string", "format": "date-time" },
    "completed_at": { "type": "string", "format": "date-time" },
    "overall_result": { "type": "string", "enum": ["pass", "fail", "needs-review"] },
    "gate_results": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["gate", "result", "score", "evidence_ref"],
        "properties": {
          "gate": { "type": "string" },
          "result": { "type": "string", "enum": ["pass", "fail", "needs-review"] },
          "score": { "type": ["number", "null"] },
          "evidence_ref": { "type": "string" }
        }
      }
    }
  },
  "additionalProperties": false
}
```

### 16.4 Release rule

An agent version cannot be marked `platform-ready` unless:

1. all P0 proof tests pass;
2. all release gate checks pass;
3. no open hard-blocking security issue exists;
4. the manifest, tool contracts, benchmark records, and sales proof pack are version-aligned;
5. active/platform-ready manifest uses `payload_mode: hash-and-reference`;
6. approval-gate tests pass against the golden approval suite with at least 50 cases;
7. the release owner signs off in the release record.

---

## 17. Audit and approval schemas

### 17.1 AuditEvent schema

`AuditEvent` supports both run-scoped events and governance events that occur outside a single run, such as engagement-close purge, batch eval completion, and release approval.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/audit-event.schema.json",
  "title": "AAP Audit Event",
  "type": "object",
  "required": [
    "audit_event_id",
    "event_type",
    "actor_type",
    "actor_id",
    "context_type",
    "context_id",
    "payload_ref",
    "payload_hash",
    "timestamp"
  ],
  "properties": {
    "audit_event_id": { "type": "string" },
    "event_type": {
      "type": "string",
      "enum": [
        "agent_started",
        "manifest_validated",
        "tool_invoked",
        "tool_denied",
        "approval_requested",
        "approval_granted",
        "approval_denied",
        "memory_written",
        "memory_superseded",
        "memory_purged",
        "eval_completed",
        "skill_revision_opened",
        "release_approved"
      ]
    },
    "actor_type": { "type": "string", "enum": ["agent", "user", "tool_principal", "approver", "system"] },
    "actor_id": { "type": "string" },
    "context_type": { "type": "string", "enum": ["run", "engagement", "agent", "skill", "release", "eval", "system"] },
    "context_id": { "type": "string" },
    "run_id": { "type": ["string", "null"] },
    "payload_ref": { "type": "string" },
    "payload_hash": { "type": "string" },
    "timestamp": { "type": "string", "format": "date-time" }
  },
  "allOf": [
    {
      "if": { "properties": { "context_type": { "const": "run" } } },
      "then": { "properties": { "run_id": { "type": "string" } } }
    }
  ],
  "additionalProperties": false
}
```

Examples: `tool_denied` uses `context_type: run`; `memory_purged` on engagement close uses `context_type: engagement`; `eval_completed` for batch evaluation uses `context_type: eval`; `release_approved` uses `context_type: release`.

### 17.2 ApprovalRequest schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aaraminds.local/schemas/approval-request.schema.json",
  "title": "AAP Approval Request",
  "type": "object",
  "required": [
    "approval_request_id",
    "run_id",
    "tool_invocation_id",
    "approval_boundary",
    "requested_action",
    "risk_summary",
    "runtime_mode",
    "status",
    "created_at"
  ],
  "properties": {
    "approval_request_id": { "type": "string" },
    "run_id": { "type": "string" },
    "tool_invocation_id": { "type": "string" },
    "approval_boundary": { "type": "string", "enum": ["soft", "hard"] },
    "requested_action": { "type": "string" },
    "risk_summary": { "type": "string" },
    "runtime_mode": { "type": "string", "enum": ["interactive", "unattended"] },
    "status": { "type": "string", "enum": ["pending", "approved", "denied", "expired"] },
    "approver_id": { "type": ["string", "null"] },
    "decision_reason": { "type": ["string", "null"] },
    "created_at": { "type": "string", "format": "date-time" },
    "decided_at": { "type": ["string", "null"], "format": "date-time" }
  },
  "additionalProperties": false
}
```

---

## 18. Requirements

### P0 — must have

| # | Requirement | Acceptance criteria |
|---|---|---|
| P0-1 | Agent manifest per schema, pinning skills, tool contracts, memory scopes, telemetry, and approval boundaries. | Proof 1 passes; invalid manifest blocks startup. |
| P0-2 | Runtime skeleton with manifest hooks and approval enforcement. | BA Agent hello-world flow runs; off-manifest tool denied. |
| P0-3 | OTel GenAI telemetry into Grafana/Prometheus. | Proof 2 passes; all model/tool spans traceable and cost-attributed. |
| P0-4 | Memory tier using Mem0 OSS + pgvector on Azure Postgres Flexible Server. | Proof 3 passes; classification and citation enforced at write. |
| P0-5 | Guardrails baseline with default-deny, blocked list, and OWASP Agentic Top 10 review. | Checklist exists; BA Agent passes; blocked list enforced. |
| P0-6 | SkillOps loop using the existing BA package eval/release-gate pattern, migrated into a consistent AAP eval record pattern. | Proof 4 passes with one real BA Agent failure converted into skill revision. |
| P0-7 | BA Agent rebuilt on platform as reference implementation. | All four proofs and release gate green. |
| P0-8 | Sales proof pack. | Pack exists in `governance/`; usable in client conversation without live setup. |
| P0-9 | Minimum build artifacts from section 9. | All listed files exist and are internally consistent. |
| P0-10 | Release gate thresholds. | Gate thresholds documented and enforced in CI/manual release checklist, including golden approval suite N ≥ 50 and `payload_mode: hash-and-reference` for platform-ready manifests. |

### P1 — should have

- Terraform module for substrate: Postgres, Key Vault, managed identities, monitoring, via GitHub Actions OIDC.
- Second agent migration, preferably network-topology-reviewer, after BA Agent passes all four proofs.
- Cost dashboard per agent per engagement.
- Langfuse self-hosted only if prompt versioning or human annotation is painful in Grafana.
- Release report generator that exports gate results and trace samples into the sales proof pack.

### P2 — future considerations

- Client-operated packaging using Foundry Agent Service or equivalent managed target.
- Entra Agent ID per agent.
- BYO VNet and private MCP subnet.
- Multi-tenancy: promote `tenant_namespace` from engagement to client tenant.
- A2A adapter only when a real two-vendor requirement appears.
- Zep/Graphiti optional temporal memory backend where point-in-time recall is a client requirement.

---

## 19. Ownership register

All roles are Raja today. Naming roles prevents undocumented craft and defines future hiring seams.

| Role | Owns | Artifact of record |
|---|---|---|
| Platform runtime owner | SDK runtime, manifest enforcement, approval flow | Runtime repo, manifest schema |
| Skills curator | Skills-pack content, revisions, Ranking.md | `skills-pack/`, `Ranking.md` |
| Infra owner | Azure resources, identities, secrets, Terraform | IaC module, Key Vault |
| Eval owner | Benchmarks, release gate, validation records | Agent package eval artifacts or `skills-pack/validation/` after migration |
| Security owner | Guardrails checklist, blocked list, OWASP review, incidents | Guardrails checklist |
| Telemetry owner | OTel pipeline, dashboards, trace retention | Grafana dashboards |
| Memory owner | Memory governance, retention, deletion, leakage tests | Memory schema, memory access module |
| Release owner | Gate sign-off, versioning, changelog, proof pack | Release records |

Incident rule: any production agent failure gets either a `SkillRevision` candidate or a platform issue within 48 hours.

---

## 20. Success metrics

### Leading metrics

| Metric | Target |
|---|---:|
| BA Agent runs traced end-to-end with cost attribution | 100% within 2 weeks of Phase 3 start |
| Unmanifested tool calls succeeding | 0 |
| Tool denials logged | 100% |
| Memory writes with citation | 100% |
| Cross-engagement leakage | 0 records |
| Skill revisions through SkillOps loop | ≥2 in first quarter |

### Lagging metrics

| Metric | Target |
|---|---:|
| Time-to-agent for second migrated agent | ≤5 working days |
| BA Agent eval score vs. pre-platform baseline | No regression |
| Memory-on improvement on multi-session tasks | Positive delta; baseline captured in Phase 3 **[VERIFY]** |
| Sales proof pack shipped | Binary yes/no |
| Platform-ready BA Agent release | Binary yes/no |

---

## 21. Open questions

1. **Mem0 OSS vs. platform tier** — does self-hosted extraction quality hold with Azure OpenAI-hosted models? Blocking Phase 2 decision.
2. **OTel GenAI compatibility** — do experimental conventions require dual emission for Grafana compatibility? Resolve in Phase 1.
3. **Client segment for v2** — which client-operated segment should be targeted first? Feed from GTM review.
4. **BA Agent skill scope** — does the manifest include instruction-os communication skills at runtime or engineering skills only?
5. **Trace retention window** — set based on real volumes and cost during Phase 1.
6. **Runtime SDK maturity** — validate Claude Agent SDK extension hooks and deployment model before locking Phase 1 architecture. **[VERIFY]**
7. **Managed target mapping** — validate Foundry Agent Service assumptions before v2 design starts. **[VERIFY]**

---

## 22. Kill and defer rules

### Kill in v1

- Visual agent-builder UI.
- Custom memory engine.
- Fine-tuning or online learning.
- Autonomous execution for high-risk actions.
- Multi-framework runtime support.
- A2A implementation without a real external interoperability requirement.
- Client-operated deployment packaging before BA Agent proves all four MVP proofs.

### Defer unless proven necessary

- Langfuse, unless Grafana cannot support prompt versioning and annotation needs.
- Full Terraform completion, unless manual setup blocks proof execution; IaC must land before v2.
- Second-agent migration, until BA Agent passes all four proofs.
- Foundry Agent Service packaging, until internal runtime is stable.
- Temporal memory backend, until a client requirement demands point-in-time recall.

### Scope control rule

Any addition to P0 requires one of the following:

1. remove an existing P0 item;
2. extend the timeline explicitly;
3. prove that the addition is required for one of the four hard proofs.

No silent scope growth.

---

## 23. Implementation backlog

### Phase 1 — substrate, weeks 1–4

| Task | Output |
|---|---|
| Create repo structure for schemas, examples, docs, governance. | Minimum build artifact folders. |
| Implement `agent-manifest.schema.json`. | Manifest validation. |
| Create `ba-agent.manifest.yaml` aligned to existing flat BA package artifacts. | First concrete manifest without nonexistent paths. |
| Implement tool contract schema. | Contract validation. |
| Create sample `get_project_context` tool contract. | First governed MCP tool contract. |
| Build runtime skeleton. | Hello-world agent. |
| Enforce off-manifest tool denial. | Proof 1 evidence. |
| Emit OTel spans for model/tool/run. | Proof 2 evidence. |
| Create Grafana dashboard v0. | Traceability demo. |
| Create guardrails checklist and blocked actions list. | Security baseline including `governance/aap-blocked-actions.yaml`. |
| Validate runtime assumptions. | Runtime verification note. |

**Phase 1 exit:** Proof 1 and Proof 2 pass on traced hello-world agent.

### Phase 2 — memory, weeks 5–7

| Task | Output |
|---|---|
| Deploy Postgres + pgvector. | Memory substrate. |
| Spike Mem0 OSS extraction quality. | Go/no-go decision. |
| Implement memory access module with engagement namespace enforcement. | Scoped memory reads/writes. |
| Enforce memory classification and citation at write. | Memory governance proof. |
| Implement retention and purge operation. | Deletion control. |
| Write leakage tests. | Proof 3 evidence. |

**Phase 2 exit:** Proof 3 passes.

### Phase 3 — SkillOps + BA Agent, weeks 8–12

| Task | Output |
|---|---|
| Rebuild BA Agent on platform runtime. | Reference implementation. |
| Reconcile existing BA package artifacts: `AGENT_SPEC.md`, `agent.md`, `eval-plan.md`, `eval-results.json`, `release-gate.json`, `tool-risk-register.md`, `mcp-adapter-contracts.md`. | Single source of truth for BA package gate, risks, and MCP contracts. |
| Implement BA Agent proof workflow. | `docs/ba-agent-proof-flow.md`. |
| Migrate/reuse existing BA release gate and eval artifacts instead of creating parallel definitions. | AAP-compatible `EvalRun` mapped to current package artifacts. |
| Run BA Agent benchmark baseline. | Eval record. |
| Run release gate. | Platform-ready decision. |
| Capture one real failure trace. | SkillRevision candidate. |
| Revise skill and rerun benchmark. | Proof 4 evidence. |
| Build sales proof pack. | Buyer-facing asset. |
| Review quarter metrics. | v2 planning input. |

**Phase 3 exit:** Proof 4 passes; all P0 acceptance criteria green.

---

## 24. Architecture view

```text
User / Operator
   |
   v
Agent Runtime
   |-- loads Agent Manifest
   |-- resolves Skills
   |-- validates MCP Tool Contracts
   |-- enforces Approval Boundaries
   |-- opens OTel Trace
   |
   +--> Memory Access Module
   |       |-- engagement namespace check
   |       |-- classification/citation enforcement
   |       +-- Mem0 OSS + pgvector/Postgres
   |
   +--> MCP Tool Gateway
   |       |-- manifest allowlist check
   |       |-- contract version check
   |       |-- approval boundary check
   |       |-- audit event write
   |       +-- Go MCP servers
   |
   +--> Model Layer
   |       +-- approved model endpoint [VERIFY per environment]
   |
   +--> Observability
   |       |-- OTel GenAI spans
   |       |-- cost metrics
   |       +-- Grafana/Prometheus
   |
   +--> Evaluation Harness
   |       |-- benchmark tests
   |       |-- safety tests
   |       |-- memory leakage tests
   |       +-- EvalRun record
   |
   +--> SkillOps
           |-- failure trace
           |-- SkillRevision candidate
           |-- benchmark before/after
           +-- versioned release
```

---

## 25. Sales proof pack

The sales proof pack must be credible without requiring a live demo environment.

Required contents:

1. AAP architecture one-pager.
2. BA Agent demo script.
3. Evaluation summary with release gate results.
4. Trace dashboard screenshot.
5. Sample OTel trace walkthrough.
6. SkillOps before/after example.
7. Security and governance checklist.
8. Sample manifest and tool contract.
9. Memory isolation test result.
10. Clear statement of v1 internal scope and v2 client-operated roadmap.

This is the commercial proof that AaraMinds agents are governed, observable, and evidence-backed.

---

## 26. Timeline

No hard external deadline; cadence-driven. Solo full-time.

| Phase | Duration | Focus | Exit criteria |
|---|---:|---|---|
| Phase 1 | Weeks 1–4 | Substrate | Proof 1 + Proof 2 pass on traced hello-world agent. |
| Phase 2 | Weeks 5–7 | Memory | Proof 3 passes with scoped recall and leakage test. |
| Phase 3 | Weeks 8–12 | SkillOps + BA Agent | Proof 4 passes; all P0 acceptance criteria green. |
| v2 | Next quarter | Client-operated packaging and additional agent migrations | Depends on BA Agent proof and GTM alignment. |

Dependency note: Phase 3 depends on existing BA Agent assets in the flat package `skills-pack/agent-packages/aara-business-analyst/`: `AGENT_SPEC.md`, `agent.md`, `eval-plan.md`, `eval-results.json`, `release-gate.json`, `tool-risk-register.md`, and `mcp-adapter-contracts.md`. Do not assume a nested `skills/` subtree or a separate `skills-pack/validation/ba-agent-benchmark.yaml` exists unless created deliberately as part of a migration.

---

## 27. Readiness checklist

AAP v1.3 is ready for implementation when all are true:

- [ ] Minimum build artifacts are created.
- [ ] Manifest schema is valid and tested.
- [ ] BA Agent manifest exists and references only existing package artifacts or explicitly created migration artifacts.
- [ ] MCP tool contract schema is valid and tested.
- [ ] At least one sample MCP tool contract exists.
- [ ] Approval boundary enforcement is implemented in runtime skeleton.
- [ ] OTel trace emits run/model/tool spans.
- [ ] Memory access module enforces engagement namespace.
- [ ] Memory writes require classification and citation.
- [ ] Release gate thresholds are encoded in evaluation harness or checklist, including golden approval suite N ≥ 50.
- [ ] BA Agent proof workflow is documented.
- [ ] Runtime assumptions are verified or marked as active risks.
- [ ] Sales proof pack skeleton exists.
- [ ] Platform-ready manifests require `payload_mode: hash-and-reference`.
- [ ] `governance/aap-blocked-actions.yaml` exists and is referenced by manifests.
- [ ] Memory purge deletes content/vector data and leaves only an AuditEvent tombstone.

---

## 28. Recommendation

Build AAP v1, but keep it narrow.

This is not a generic platform effort. It is a controlled internal substrate to prove that AaraMinds agents can be built, governed, evaluated, observed, and improved with repeatable evidence.

The BA Agent must remain the proof case. No second agent, client-operated packaging, visual UI, or framework expansion should enter P0 until the BA Agent passes the four hard proofs.

**Decision:** Build v1.  
**Execution posture:** narrow, evidence-first, approval-gated, traceable.  
**Next artifact:** implement the minimum build artifact set from Section 9.

