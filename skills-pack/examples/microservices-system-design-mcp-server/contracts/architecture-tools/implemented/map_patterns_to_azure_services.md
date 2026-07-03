# Tool Contract — map_patterns_to_azure_services

**Status:** Implemented (v9.0, May 2026)
**Implementation:** `examples/microservices-system-design-mcp-server/internal/services/azuremap/`

## Purpose

Map architecture patterns to the Azure services that implement them. The tool takes a list of pattern names and deterministically maps each — via a curated, version-dated catalog — to one or more Azure services with a role, a rationale, and alternatives. It also reports mapping findings (unknown patterns, deployment-target mismatch, missing cross-cutting patterns) and a mapping-coverage score.

This is decision support for the pattern→platform step. The catalog encodes the conventional Azure mapping for each pattern so the choice is consistent and explained, not ad hoc.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string (optional)",
  "patterns": ["string (required, >=1)"],
  "deployment_target": "string (optional: aks | container_apps | app_service | functions | hybrid)",
  "constraints": ["string (optional)"]
}
```

**Field semantics:**

- `patterns` — pattern names. Normalized (lowercase, spaces/hyphens → underscores) and de-duplicated before lookup. Unknown patterns are returned in `unmapped_patterns` and raise a low finding.
- `deployment_target` — when a pattern has a conventional platform (`container_orchestration`→aks, `serverless_compute`→functions) and the declared target differs (and is not `hybrid`), a medium `deployment_mismatch` finding is raised.

**Recognized patterns:** api_gateway, async_messaging, event_streaming, saga, cqrs, event_sourcing, circuit_breaker, service_discovery, config_management, secrets_management, relational_data, document_data, cache, container_orchestration, serverless_compute, observability, identity, blob_storage.

**Validation:** `system_name` non-empty; at least one pattern. Failures return a structured error result.

## Output Schema

```json
{
  "system_name": "string",
  "mappings": [
    {
      "pattern": "string (normalized)",
      "azure_services": [ { "name": "string", "role": "string" } ],
      "rationale": "string",
      "alternatives": ["string"]
    }
  ],
  "unmapped_patterns": ["string"],
  "mapping_findings": [
    { "severity": "low|medium|high", "category": "string",
      "description": "string", "patterns_related": ["string"], "recommendation": "string" }
  ],
  "coverage": { "mapped_count": "integer", "total_count": "integer" },
  "mapping_score": "integer 0-100",
  "mapping_rating": "string",
  "summary": "string"
}
```

**Scoring:** start at 100; high finding −15, medium −8, low −3; plus a proportional penalty up to 20 for unmapped coverage (`20 * (total-mapped)/total`); floor 0.

**Rating bands:** 90–100 strong · 75–89 sound · 60–74 material gaps · 40–59 weak · 0–39 insufficient.

**Finding categories:** `unknown_pattern`, `deployment_mismatch`, `missing_secrets_management`, `missing_observability`.

**Output bounding:** mappings bounded by distinct recognized patterns; findings bounded combinatorially. Free-form text confined to `rationale`, `description`, `recommendation`, `summary`.

## Risk Tier

**Low.** Read-only catalog lookup. No backend, no state, no network.

## Authorization

No per-tool authorization beyond authentication.

## Human Approval

Not required.

## Failure Modes

| Failure | Cause | Response shape |
|---|---|---|
| `system_name is required` | Missing/whitespace system_name | Error result |
| `at least one pattern is required` | Empty patterns array | Error result |
| `input_json must be valid JSON...` | Malformed input_json | Error result with parser detail |
| Internal marshal failure | (should never happen) | `internal error: failed to format result` |

No timeouts, rate limits, or retries (deterministic in-memory lookup).

## Observability

INFO: `tool call started`, `tool call completed` (system, patterns, mapped, score), `tool call rejected`. WARN: `input_json missing`, `input_json failed to parse`. Metrics: calls by status; score distribution; coverage ratio. Tracing: participates in transport trace; starts no spans.

## Determinism

Fully deterministic. Mappings sorted by pattern; unmapped sorted; findings sorted by severity descending then category. Same input → same output byte-for-byte (modulo `MarshalIndent` whitespace). See `testdata/azuremap-output-ecommerce.json`.

## Compatibility

**Version:** 1.0.0. The catalog is version-dated (May 2026); Azure service names must be re-verified per the freshness mechanism in `skills/mcp/00-ecosystem-facts.md`. **Non-breaking in 1.x:** new optional input fields; new output fields; new catalog entries; new finding categories. **Breaking (2.0):** renaming/removing fields; type changes; removing catalog entries; scoring changes that change which inputs produce which scores.

## Example

See `testdata/azuremap-input-ecommerce.json` and `testdata/azuremap-output-ecommerce.json`.
