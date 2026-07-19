# Runtime Verification Notes

Status: active verification log for PRD v1.3.

## Current Decision

AAP v1 uses a local deterministic runtime proof harness in `platform/` to validate platform contracts before binding to a hosted agent runtime. This keeps the proof work independent from Claude Agent SDK or Foundry Agent Service maturity.

## Verification Items

| Decision | Current status | Next evidence |
|---|---|---|
| Runtime SDK | Externally verified 2026-07-19; repo spike pending | Claude Agent SDK core (agent loop, `PreToolUse`/`PostToolUse` hooks, MCP integration, subagents, permission system) is stable and in production use; June 2026 added Dynamic Workflows (parallel subagent fan-out) and Performance Outcomes (grader-driven revision). Hook layer maps cleanly onto AAP's tool-boundary gate. Remaining: repo-side spike binding one manifest-controlled agent through SDK hooks to confirm the harness gate semantics survive translation. |
| Managed target | Externally verified 2026-07-19 (GA); binding spike pending | Foundry Agent Service is GA (March 2026) with BYO VNet private networking (no public egress, subnet injection), per-agent Entra Agent ID, OAuth OBO to external MCP servers, and BYO Storage / AI Search / Cosmos DB at rest. Terraform BYO-VNet samples exist (`foundry-samples` 15b), fitting the AzureRM stack. Remaining: binding spike for manifest + audit-chain semantics on the hosted runtime before v2 commitment. |
| OTel GenAI | Partial — conventions still not stable | Local harness emits one live OTel trace per run; GenAI operation names limited to applicable registry values (`invoke_agent`, `execute_tool`); governance spans use `aap.*`. Confirmed 2026-07-19: GenAI/MCP semconv remain in Development status with no published stabilization timeline; Grafana has begun collecting LLM traces (Loki). Keep the contained `gen_ai.*` mapping and `aap.*` namespace — do not widen `gen_ai.*` usage until semconv stabilizes. Collector/Grafana compatibility check still owed before production. |
| Identity | Model clarified 2026-07-19; dev fallback still open | Entra Agent ID assigns each agent its own identity, but credentials live on the **agent identity blueprint**, not the individual agent — blueprints centralize credential management and can use Azure Managed Identity as the credential type. Revise the AAP assumption from "managed identity per `agent_id`" to "agent identity per `agent_id`, credentials via blueprint-held managed identity." Local dev fallback without shared production credentials still undefined. |
| Memory | Open — spike still required | Mem0 OSS is actively maintained (v2.4.6, 2026-04-04, incl. Azure AI Search fixes; JSON-extraction fixes in v2.4.5) and shipped a new memory algorithm in April 2026 with vendor-reported gains on temporal (+29.6) and multi-hop (+23.1) queries `[VERIFY — vendor-reported]`. Azure integration path (Azure OpenAI + AI Search) is documented by Microsoft. Vendor numbers do not substitute for the Phase 2 extraction-quality spike on AAP's own memory-record contract. |
| MCP spec/SDK | Action required — spec revision landing 2026-07-28 | MCP 2026-07-28 is in release candidate and finalizes July 28: stateless protocol core, extensions framework (Tasks, MCP Apps), tighter OAuth/OIDC alignment, formal deprecation policy with a 12-month legacy window. Official Go SDK is v1-stable; v1.5.0 (2026-04-07) current-ish, v1.7.0+ supports the new spec. Plan: bump the pinned `modelcontextprotocol/go-sdk` in `skills-pack` MCP server to ≥ v1.7.0 and validate tool contracts against the stateless core within the deprecation window. |

## Verification Log — 2026-07-19

External re-verification of the six `[VERIFY]` runtime assumptions (web sources, checked 2026-07-19):

- **Claude Agent SDK** — core loop, hooks (`PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `SubagentStart/Stop`), MCP extension mechanism, permission system, and session persistence are stable and production-deployed; June 2026 additions: Dynamic Workflows, Performance Outcomes. Sources: [Agent SDK overview](https://code.claude.com/docs/en/agent-sdk/overview), [claude-agent-sdk-python](https://github.com/anthropics/claude-agent-sdk-python), [hooks guide](https://team400.ai/blog/2026-03-claude-agent-sdk-hooks-guide), [production patterns](https://www.digitalapplied.com/blog/claude-agent-sdk-production-patterns-guide).
- **Foundry Agent Service** — GA March 2026: [GA announcement](https://devblogs.microsoft.com/foundry/foundry-agent-service-ga/), [private networking / BYO VNet](https://learn.microsoft.com/en-us/azure/foundry/agents/how-to/virtual-networks), [Terraform BYO-VNet sample](https://github.com/microsoft-foundry/foundry-samples/tree/main/infrastructure/infrastructure-setup-terraform/15b-private-network-standard-agent-setup-byovnet/), [service overview](https://learn.microsoft.com/en-us/azure/foundry/agents/overview).
- **OTel GenAI semconv** — still Development status as of mid-2026; no stable timeline. Sources: [gen-ai spans (semconv repo)](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/gen-ai/gen-ai-spans.md), [OTel GenAI observability blog](https://opentelemetry.io/blog/2026/genai-observability/).
- **Entra Agent ID** — agent identities are credential-less; blueprints hold credentials (managed identity supported as blueprint credential). Sources: [agent identities overview](https://learn.microsoft.com/en-us/entra/agent-id/agent-identities), [what is Entra Agent ID](https://learn.microsoft.com/en-us/entra/agent-id/what-is-microsoft-entra-agent-id), [Foundry agent identity concepts](https://learn.microsoft.com/en-us/azure/foundry/agents/concepts/agent-identity).
- **Mem0 OSS** — active releases through April 2026; new memory algorithm (vendor-benchmarked); Microsoft-documented Azure integration. Sources: [mem0 repo](https://github.com/mem0ai/mem0), [SDK changelog](https://docs.mem0.ai/changelog/sdk), [Azure AI + Mem0 integration](https://devblogs.microsoft.com/foundry/azure-ai-mem0-integration/), [Mem0 2026 benchmark report](https://mem0.ai/blog/state-of-ai-agent-memory-2026) `[vendor]`.
- **MCP spec / Go SDK** — 2026-07-28 spec RC published; Go SDK v1-stable with new-spec support from v1.7.0. Sources: [2026-07-28 RC announcement](https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/), [2026 MCP roadmap](https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/), [go-sdk releases](https://github.com/modelcontextprotocol/go-sdk/releases).

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
