# Tool Contract — generate_service_boundary_canvas

**Status:** Implemented (v9.0, May 2026)
**Implementation:** `examples/microservices-system-design-mcp-server/internal/services/boundary/`

## Purpose

Generate a structured service boundary canvas for a proposed microservices system. The tool takes a structured description of proposed services and produces per-service boundary assessments, named boundary risks, recommended changes, and an overall score.

This is decomposition-quality assessment, not architecture review. It catches the boundary problems that show up most often in real designs — co-owned data, capability drift, chatty synchronous chains, circular dependencies, fan-out bottlenecks, orphaned services without owners.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string (optional, human-readable context)",
  "services": [
    {
      "name": "string (required, unique)",
      "business_capability": "string (recommended, at least two words)",
      "owns_data": ["string"],
      "depends_on": ["string"],
      "consumes_events_from": ["string"],
      "team": "string (recommended)"
    }
  ],
  "data_stores": [
    {
      "name": "string",
      "kind": "string (e.g., postgres, cosmos, redis, blob)"
    }
  ],
  "teams": [
    {
      "name": "string"
    }
  ]
}
```

**Field semantics:**

- `business_capability` — the single business capability this service is responsible for. Required for cohesion assessment. Generic one-word values like "data" or "service" are flagged as ambiguous.
- `owns_data` — list of data store names this service exclusively owns. Data stores claimed by multiple services trigger high-severity co-ownership risk.
- `depends_on` — synchronous (typically HTTP/gRPC) call dependencies. Used for chatty-dependency and circular-dependency detection.
- `consumes_events_from` — asynchronous (event-bus) dependencies. Lower coupling than synchronous; not counted toward chatty heuristic.
- `team` — single team that owns the service. Empty values trigger no-owner risk.

**Validation:**

- `system_name` must be non-empty
- At least one service is required
- Service names must be non-empty and unique

Validation failures return a structured error result. The tool does **not** attempt to repair or assume defaults for invalid input.

## Output Schema

```json
{
  "system_name": "string",
  "service_assessments": [
    {
      "service": "string",
      "capability_clarity": "clear | ambiguous | missing",
      "owns_distinct_data": "boolean",
      "dependency_health": "healthy | chatty | circular_risk",
      "owner_clarity": "single_team | no_team | multi_team_risk",
      "notes": ["string"]
    }
  ],
  "boundary_risks": [
    {
      "severity": "low | medium | high",
      "category": "data_co_ownership | chatty_dependency | circular_dependency | no_owner | capability_drift | fan_out",
      "description": "string",
      "services_affected": ["string"]
    }
  ],
  "recommended_changes": [
    {
      "action": "merge | split | clarify_ownership | introduce_async | consolidate_data",
      "targets": ["string"],
      "rationale": "string"
    }
  ],
  "overall_score": "integer 0–100",
  "overall_rating": "string",
  "summary": "string"
}
```

**Scoring rubric:**

- Start at 100
- Each high-severity risk deducts 15
- Each medium-severity risk deducts 8
- Each low-severity risk deducts 3
- Floor at 0

**Rating bands:**

- 90–100: Boundaries well-defined; minor adjustments only
- 75–89: Directionally sound; targeted improvements recommended
- 60–74: Material issues; address before implementation
- 40–59: Significant rework needed; redesign recommended
- 0–39: Boundaries are unclear or broken; restart the decomposition exercise

**Output bounding:**

- `service_assessments` is bounded by the input services count (no fan-out)
- `boundary_risks` has no fixed cap but is bounded by combinatorial limits on the input shape
- All output is structured JSON; no free-form text outside the `description`, `rationale`, `summary`, and `notes` fields

## Risk Tier

**Low.** Read-only architecture assessment. The tool inspects the input structure and returns analysis. It does not connect to any backend, query any data store, or modify any state.

## Authorization

No per-tool authorization beyond authentication (any authenticated principal may call this tool). This is acceptable because the tool reads only the input the caller already has.

## Human Approval

Not required.

## Failure Modes

| Failure | Cause | Response shape |
|---|---|---|
| `system_name is required` | Missing or whitespace-only system_name | Error result with this message |
| `at least one proposed service is required` | Empty services array | Error result |
| `every service must have a non-empty name` | Service with empty/whitespace name | Error result |
| `duplicate service name: X` | Multiple services share a name | Error result |
| `input_json must be valid JSON matching the Input schema: ...` | input_json string is not valid JSON | Error result with parser detail |
| Internal marshal failure | (should never happen) Bug in the service | Error result `internal error: failed to format result` |

The tool does not have timeouts (purely in-memory computation). It does not have rate limits at the tool level (caller's API gateway should enforce). It does not have retry semantics (deterministic — same input always produces same output).

## Observability

**Log events** emitted via `log/slog` at INFO level:
- `tool_call_started` — with tool name
- `tool_call_completed` — with tool name, system name, services analyzed count, risks identified count, score
- `tool_call_rejected` — with tool name, error reason (for validation failures)

**Log events** emitted at WARN level:
- `input_json missing` — with tool name
- `input_json failed to parse` — with tool name, parse error

**Metrics** (consumer should aggregate):
- Tool calls by status (success / validation_failure)
- Score distribution (histogram)
- Risks-per-call distribution (histogram)

**Tracing:** the tool participates in any trace propagated by the MCP transport layer. It does not start spans of its own (single-function execution).

## Determinism

Output is fully deterministic given input. Risk ordering is stable (sorted by severity descending, then by category alphabetically). Cycle detection produces sorted service lists. The same input JSON will produce the same output JSON, byte-for-byte (modulo whitespace normalization in `json.MarshalIndent`).

This determinism is what makes golden-output testing possible. See `testdata/boundary-output-ecommerce.json`.

## Compatibility

**Version:** 1.0.0 (initial implementation, May 2026)

**Stability commitment:** input and output schemas are stable. Additions (new optional input fields, new output fields, new risk categories, new recommended-change actions) are non-breaking. Renames, removals, or semantic changes to existing fields are breaking and require a major version bump.

**Field additions in 1.x that are guaranteed non-breaking:**
- New optional input fields (callers omit them)
- New output fields (callers that ignore unknown fields are unaffected)
- New enum values for `capability_clarity`, `dependency_health`, `owner_clarity`, `severity`, `category`, `action` — callers must handle unknown values gracefully

**Breaking changes (require 2.0):**
- Renaming any existing field
- Changing the type of any existing field
- Removing any output field
- Changing the scoring formula in a way that changes which inputs produce which scores

## Example

See `testdata/boundary-input-ecommerce.json` for a representative input and `testdata/boundary-output-ecommerce.json` for the corresponding output.
