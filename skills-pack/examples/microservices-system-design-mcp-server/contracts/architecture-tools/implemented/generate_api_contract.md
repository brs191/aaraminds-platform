# Tool Contract — generate_api_contract

**Status:** Implemented (v9.0, May 2026)
**Implementation:** `examples/microservices-system-design-mcp-server/internal/services/apicontract/`

## Purpose

Generate an OpenAPI-shaped API contract for a proposed microservices system. The tool takes services with their resources and operations and deterministically produces REST endpoints (method, path, success status, error responses), per-service security, contract-quality findings, and a contract-readiness score.

This is contract scaffolding and review, not full OpenAPI document generation. It produces the structural skeleton and flags the contract problems that show up most often before an external consumer integrates: unsecured endpoints, missing versioning, unpaginated list endpoints, inconsistent base paths, and missing capability ownership.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string (optional)",
  "api_style": "string (optional: rest (default) | grpc | graphql)",
  "versioning_strategy": "string (optional: uri | header | none)",
  "services": [
    {
      "name": "string (required, unique)",
      "business_capability": "string (recommended)",
      "base_path": "string (optional, e.g. /orders)",
      "auth": "string (none | api_key | oauth2 | mtls)",
      "resources": [
        {
          "name": "string (required)",
          "operations": ["list | get | create | update | delete"],
          "paginated": "boolean (applies to list)",
          "versioned": "boolean"
        }
      ]
    }
  ]
}
```

**Field semantics:**

- `versioning_strategy` — empty or `none` raises a medium finding; a strategy must exist before breaking changes can ship.
- `auth` — empty/`none` raises a high `unsecured_endpoint` finding for that service.
- `base_path` — defaults to `/<service-name>`; a non-empty value not starting with `/` raises a low finding.
- `resources[].operations` — drive endpoint generation. A `list` operation maps to `GET /{base}/{resource}`; `get/update/delete` map to the `/{id}` item path; `create` maps to `POST` on the collection.

**Validation:** `system_name` non-empty; at least one service; service names non-empty and unique. Failures return a structured error result.

## Output Schema

```json
{
  "system_name": "string",
  "api_contracts": [
    {
      "service": "string",
      "base_path": "string",
      "security": "none | api_key | oauth2 | mtls",
      "endpoints": [
        { "method": "string", "path": "string", "summary": "string",
          "success_status": "integer", "error_responses": ["integer"] }
      ]
    }
  ],
  "contract_findings": [
    { "severity": "low|medium|high", "category": "string",
      "description": "string", "services_affected": ["string"], "recommendation": "string" }
  ],
  "openapi_summary": { "openapi_version": "3.1.0", "total_paths": "integer", "total_operations": "integer" },
  "contract_score": "integer 0-100",
  "contract_rating": "string",
  "summary": "string"
}
```

**Endpoint generation rules:** `list`→`GET` collection (200), `get`→`GET` item (200, 404), `create`→`POST` collection (201, 400), `update`→`PUT` item (200, 400, 404), `delete`→`DELETE` item (204, 404). Every endpoint includes `401` and `500` error responses.

**Scoring:** start at 100; high finding −15, medium −8, low −3; floor 0.

**Rating bands:** 90–100 production-ready · 75–89 directionally sound · 60–74 material gaps · 40–59 significant work · 0–39 not ready.

**Finding categories:** `missing_versioning`, `unsecured_endpoint`, `no_pagination`, `inconsistent_base_path`, `missing_capability`.

**Output bounding:** endpoints bounded by (services × resources × operations); findings bounded combinatorially. No free-form text outside `summary`, `description`, `recommendation`.

## Risk Tier

**Low.** Read-only generation. No backend, no state, no network.

## Authorization

No per-tool authorization beyond authentication.

## Human Approval

Not required.

## Failure Modes

| Failure | Cause | Response shape |
|---|---|---|
| `system_name is required` | Missing/whitespace system_name | Error result |
| `at least one service is required` | Empty services array | Error result |
| `every service must have a non-empty name` | Empty service name | Error result |
| `duplicate service name: X` | Duplicate service names | Error result |
| `input_json must be valid JSON...` | Malformed input_json | Error result with parser detail |
| Internal marshal failure | (should never happen) | `internal error: failed to format result` |

No timeouts, rate limits, or retries (deterministic in-memory computation).

## Observability

INFO: `tool call started`, `tool call completed` (system, services, operations, score), `tool call rejected` (error). WARN: `input_json missing`, `input_json failed to parse`. Metrics: calls by status; score distribution; operations-per-call. Tracing: participates in transport trace; starts no spans.

## Determinism

Fully deterministic. Endpoints sorted by path then method; findings sorted by severity descending then category. Same input → same output byte-for-byte (modulo `MarshalIndent` whitespace). See `testdata/apicontract-output-ecommerce.json`.

## Compatibility

**Version:** 1.0.0. **Non-breaking in 1.x:** new optional input fields; new output fields; new finding categories (handle unknown gracefully). **Breaking (2.0):** renaming/removing fields; type changes; scoring changes that change which inputs produce which scores.

## Example

See `testdata/apicontract-input-ecommerce.json` and `testdata/apicontract-output-ecommerce.json`.
