# Runtime Verification Notes

Status: active verification log for PRD v1.3.

## Current Decision

AAP v1 uses a local deterministic runtime proof harness in `platform/` to validate platform contracts before binding to a hosted agent runtime. This keeps the proof work independent from Claude Agent SDK or Foundry Agent Service maturity.

## Verification Items

| Decision | Current status | Next evidence |
|---|---|---|
| Runtime SDK | Open | Validate Claude Agent SDK extension points, local/server execution model, hook support, and deployment fit before replacing the local harness. |
| Managed target | Open | Validate Foundry Agent Service Entra Agent ID mapping, BYO VNet, private MCP subnet, and secrets model before v2. |
| OTel GenAI | Open | Confirm semantic convention maturity and Grafana compatibility. The local harness emits trace-shaped records, not live OTel spans. |
| Identity | Open | Validate Azure managed identity per `agent_id`; define local dev fallback that does not use shared production credentials. |
| Memory | Open | Run the Mem0 OSS + Azure OpenAI extraction-quality spike in Phase 2. |

## Local Harness Boundary

The local harness is not a production runtime. It proves manifest enforcement, tool-boundary decisions, audit-event shape, trace-shaped run records, and scoped memory behavior.

## Local Harness Coverage

Current automated coverage includes:

- JSON and YAML manifest/contract loading.
- Manifest and tool-contract validation against `schemas/`.
- Tool input validation against each contract `input_schema`.
- Contract example validation against each tool `input_schema`.
- Off-manifest, missing-contract, contract-version, input-schema, engagement-scope, blocked-action, blocked-boundary, and approval-boundary decisions.
- Contract-specific audit payload validation against each tool `audit_event_schema`.
- Run-scoped audit events validated against `schemas/audit-event.schema.json`.
- Memory writes validated against `schemas/memory-record.schema.json`, including active-run source citation checks.
- Cross-engagement memory isolation.
- MCP server service tests, adapter-level design tool tests, race tests, vet, formatting checks, build, and architecture-demo golden validation through CI.
