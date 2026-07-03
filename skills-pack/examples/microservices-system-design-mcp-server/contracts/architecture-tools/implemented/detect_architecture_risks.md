# Tool Contract — detect_architecture_risks

**Status:** Implemented (v9.0, May 2026)
**Implementation:** `examples/microservices-system-design-mcp-server/internal/services/archrisks/`

## Purpose

Detect architecture-level risks in a proposed microservices system. The tool takes a structured description of components, data stores, deployment target, constraints, and non-functional requirements, and produces named risks — each with a severity, a likelihood, the affected components, and a concrete mitigation — plus a risk-posture score, the decisions still missing, and recommended next steps.

This is operational and resilience risk detection, not boundary assessment (see `generate_service_boundary_canvas` for decomposition quality). It catches the risks that surface most often in real designs: single points of failure, cascading failure on synchronous chains, missing resilience controls, stateful components with no durable store, shared data stores, unencrypted sensitive data, and compliance constraints that are not reflected in the data model.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string (optional)",
  "services": [
    {
      "name": "string (required, unique)",
      "criticality": "high | medium | low (optional, default medium)",
      "stateful": "boolean (optional)",
      "replicated": "boolean (optional, true = runs HA with >1 instance)",
      "depends_on": ["string"],
      "consumes_events_from": ["string"],
      "data_stores": ["string"],
      "resilience": ["string (e.g., retry, circuit_breaker, timeout, bulkhead)"]
    }
  ],
  "data_stores": [
    {
      "name": "string",
      "kind": "string (e.g., postgres, cosmos, redis, blob)",
      "encrypted": "boolean (encryption at rest)",
      "classification": "string (e.g., pii, phi, pci, sensitive, public)"
    }
  ],
  "deployment_target": "string (optional: aks | container_apps | app_service | functions | hybrid)",
  "constraints": ["string"],
  "non_functional_requirements": {
    "availability_target": "string (e.g., \"99.9\")",
    "latency_p99_ms": "integer",
    "rto_minutes": "integer",
    "rpo_minutes": "integer"
  }
}
```

**Field semantics:**

- `criticality` — business criticality of the component. Drives cascading-failure and SLO rules. Unset is treated as `medium` and is also reported as a missing decision.
- `replicated` — whether the component runs highly available. A high fan-in component that is not replicated is a single point of failure.
- `depends_on` — synchronous dependencies. Used for fan-in and cascading-failure detection.
- `data_stores` (on a component) — names of stores the component reads/writes. A store used by more than one component is a shared-store risk.
- `resilience` — declared resilience controls. A high-criticality component with none is flagged.
- `classification` (on a data store) — `pii`, `phi`, `pci`, or `sensitive` classified stores that are not `encrypted` raise a high risk.

**Validation:**

- `system_name` must be non-empty
- At least one service is required
- Service names must be non-empty and unique

Validation failures return a structured error result. The tool does **not** repair or assume defaults for invalid input.

## Output Schema

```json
{
  "system_name": "string",
  "risks": [
    {
      "severity": "low | medium | high",
      "likelihood": "low | medium | high",
      "category": "single_point_of_failure | cascading_failure | missing_resilience | stateful_without_datastore | shared_data_store | unencrypted_sensitive_data | deployment_target_unspecified | no_availability_target | compliance_constraint_unaddressed",
      "description": "string",
      "components_affected": ["string"],
      "mitigation": "string"
    }
  ],
  "missing_decisions": ["string"],
  "next_steps": ["string"],
  "risk_posture_score": "integer 0–100 (higher is healthier)",
  "risk_rating": "string",
  "summary": "string"
}
```

**Scoring rubric (severity × likelihood deduction from 100, floored at 0):**

| severity \ likelihood | high | medium | low |
|---|---|---|---|
| high | 20 | 15 | 10 |
| medium | 10 | 8 | 5 |
| low | 3 | 2 | 1 |

**Rating bands:**

- 90–100: Low risk; operationally sound with minor follow-ups
- 75–89: Moderate risk; targeted hardening recommended before launch
- 60–74: Elevated risk; material issues to resolve before implementation
- 40–59: High risk; significant rework recommended before proceeding
- 0–39: Severe risk; the architecture needs rework before it is operable

**Output bounding:**

- `risks` has no fixed cap but is bounded by combinatorial limits on the input shape
- `next_steps` is capped at five (deduplicated mitigations of the highest-ranked risks)
- All output is structured JSON; no free-form text outside the `description`, `mitigation`, `summary`, `missing_decisions`, and `next_steps` fields

## Risk Tier

**Low.** Read-only architecture assessment. The tool inspects the input structure and returns analysis. It does not connect to any backend, query any data store, or modify any state.

## Authorization

No per-tool authorization beyond authentication. The tool reads only the input the caller already has.

## Human Approval

Not required.

## Failure Modes

| Failure | Cause | Response shape |
|---|---|---|
| `system_name is required` | Missing or whitespace-only system_name | Error result with this message |
| `at least one service is required` | Empty services array | Error result |
| `every service must have a non-empty name` | Service with empty/whitespace name | Error result |
| `duplicate service name: X` | Multiple services share a name | Error result |
| `input_json must be valid JSON matching the Input schema: ...` | input_json string is not valid JSON | Error result with parser detail |
| Internal marshal failure | (should never happen) Bug in the service | Error result `internal error: failed to format result` |

The tool has no timeouts (purely in-memory computation), no tool-level rate limits, and no retry semantics (deterministic — same input always produces the same output).

## Observability

**Log events** emitted via `log/slog` at INFO level:
- `tool call started` — with tool name
- `tool call completed` — with tool name, system name, services analyzed, risks identified, score
- `tool call rejected` — with tool name, error reason (validation failures)

**Log events** at WARN level:
- `input_json missing` — with tool name
- `input_json failed to parse` — with tool name, parse error

**Metrics** (consumer should aggregate): tool calls by status; risk-posture score distribution; risks-per-call distribution.

**Tracing:** participates in any trace propagated by the MCP transport layer; starts no spans of its own.

## Determinism

Output is fully deterministic given input. Risk ordering is stable: sorted by severity descending, then likelihood descending, then category alphabetically. Affected-component lists are sorted. The same input JSON produces the same output JSON, byte-for-byte (modulo `json.MarshalIndent` whitespace). This is what makes golden-output testing possible — see `testdata/archrisks-output-ecommerce.json`.

## Compatibility

**Version:** 1.0.0 (initial implementation, May 2026)

**Stability commitment:** input and output schemas are stable. Additions (new optional input fields, new output fields, new risk categories) are non-breaking. Renames, removals, or semantic changes to existing fields are breaking and require a major version bump.

**Non-breaking in 1.x:** new optional input fields; new output fields; new enum values for `severity`, `likelihood`, `category` (callers must handle unknown values gracefully).

**Breaking (requires 2.0):** renaming/removing any field; changing a field's type; changing the scoring matrix in a way that changes which inputs produce which scores.

## Example

See `testdata/archrisks-input-ecommerce.json` for a representative input and `testdata/archrisks-output-ecommerce.json` for the corresponding output.
