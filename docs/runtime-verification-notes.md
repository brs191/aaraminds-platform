# Runtime Verification Notes

Status: active verification log for PRD v1.3.

## Current Decision

AAP v1 uses a local deterministic runtime proof harness in `platform/` to validate platform contracts before binding to a hosted agent runtime. This keeps the proof work independent from Claude Agent SDK or Foundry Agent Service maturity.

## Verification Items

| Decision | Current status | Next evidence |
|---|---|---|
| Runtime SDK | Open | Validate Claude Agent SDK extension points, local/server execution model, hook support, and deployment fit before replacing the local harness. |
| Managed target | Open | Validate Foundry Agent Service Entra Agent ID mapping, BYO VNet, private MCP subnet, and secrets model before v2. |
| OTel GenAI | Partial | Local harness can emit one live OTel trace per run from existing run/tool/approval/memory trace records. GenAI operation names are limited to applicable registry values (`invoke_agent`, `execute_tool`); governance spans use `aap.*`. Still validate collector/Grafana compatibility before production. |
| Identity | Open | Validate Azure managed identity per `agent_id`; define local dev fallback that does not use shared production credentials. |
| Memory | Open | Run the Mem0 OSS + Azure OpenAI extraction-quality spike in Phase 2. |

## Local Harness Boundary

The local harness is not a production runtime. It proves manifest enforcement, tool-boundary decisions, the approval lifecycle with approver identity, a replayable and tamper-evident audit trail, trace-shaped run records with an optional OpenTelemetry projection, and scoped memory behavior. The in-memory payload store stands in for a durable content-addressed store (e.g. Azure Blob with immutability policy) in v2.

## Local Harness Coverage

Current automated coverage includes:

- JSON and YAML manifest/contract loading.
- Manifest and tool-contract validation against `schemas/`.
- Tool input validation against each contract `input_schema`.
- Contract example validation against each tool `input_schema`.
- Load-time rejection of any tool contract whose `input_schema` does not require `engagement_id` as a string property (engagement scoping is a structural invariant, not a convention).
- Off-manifest, missing-contract, contract-version, input-schema, engagement-scope, blocked-action, blocked-boundary, and approval-boundary decisions, enforced by a single shared gate for both tool invocation and tool-result recording.
- Fail-closed engagement scoping: a tool payload without a string `engagement_id` matching the active run is denied, even if its contract schema would allow it.
- Contract-specific audit payload validation against each tool `audit_event_schema`.
- Run-scoped audit events validated against `schemas/audit-event.schema.json`.
- Memory writes validated against `schemas/memory-record.schema.json`, including active-run source citation checks.
- Cross-engagement memory isolation.
- Full approval lifecycle: `approval_requested` creates a pending `ApprovalRequest` (validated against `schemas/approval-request.schema.json`); `ResolveApproval` records `approval_granted` / `approval_denied` with approver identity; grants are single-use and consumed by one matching re-invocation; interactive soft-boundary confirmation is itself recorded as a grant attributed to the run user.
- Invocation-result pairing: each successful invocation authorizes exactly one accepted result; duplicate results are denied, while denied results leave the invocation pending so a corrected result can be recorded.
- Unclassified action enforcement: action types absent from both `blocked_actions` and `classified_actions` fall under `default_unclassified_boundary` — `hard` escalates the effective approval boundary, `blocked` denies outright.
- Replayable audit trail: every audit `payload_ref` resolves in a content-addressed payload store (`cas://sha256/...`), verified by `VerifyAuditTrail`.
- Tamper-evident audit chain: each event carries `prev_event_hash`; insertion, deletion, reordering, or mutation is detected by `VerifyAuditChain`.
- Optional OpenTelemetry projection: when enabled, the runtime emits one trace per run from the same local trace records, with `aap.*` correlation attributes, `aap.audit_event_id` links back to the audit chain, and contained `gen_ai.*` mapping only for applicable agent/tool operations.
- Memory-citation gate: uncited memory writes are denied, not stored, and audited as `memory_denied` events within the tamper-evident chain (proof fields `UncitedMemoryWriteDenied`, `UncitedMemoryDenialAudited`).
- Prompt-injection tool-escalation gates: injected off-manifest tool calls are denied and audited, injected unattended writes escalate to hard approval instead of executing, and the manifest is byte-identical before and after injection scenarios (proof fields `InjectionToolDenied`, `InjectionApprovalEnforced`, `InjectionManifestUnchanged`).
- Agent Factory pipeline (added 2026-07-05): schema-validated intake, deterministic autonomy/risk classification, artifact scaffolding with section self-checks, rubric-driven readiness scoring with evidence-backed checks and an activation gate, and tamper-evident export with round-trip attestation (`aapctl intake|classify|scaffold|sections|readiness|export`).
- MCP server service tests, adapter-level design tool tests, race tests, vet, formatting checks, build, and architecture-demo golden validation through CI.
