# Tool Contract — generate_observability_plan

**Status:** Implemented (v9.0, May 2026)
**Implementation:** `examples/microservices-system-design-mcp-server/internal/services/obsplan/`

## Purpose

Generate an observability plan for a proposed microservices system. The tool takes services with their criticality and type and deterministically produces per-service SLIs, SLOs (derived from criticality and any stated availability/latency targets), recommended dashboards, and recommended alerts. It also reports coverage gaps and an observability-readiness score.

This is observability scaffolding and gap analysis. It produces the SLI/SLO/dashboard/alert skeleton and flags the gaps that most often leave a system un-operable: critical services with no alerts, no dashboards, API services with no latency target, and a missing system-wide availability target.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string (optional)",
  "services": [
    {
      "name": "string (required, unique)",
      "criticality": "high | medium | low (optional, default medium)",
      "type": "api | gateway | worker | datastore (optional, default api)",
      "has_dashboards": "boolean",
      "has_alerts": "boolean"
    }
  ],
  "non_functional_requirements": {
    "availability_target": "string (e.g. \"99.9\")",
    "latency_p99_ms": "integer"
  }
}
```

**Field semantics:**

- `criticality` — sets the availability SLO objective (high 99.95%, medium 99.9%, low 99.5%) and alert severity (high→page, medium→ticket, low→info).
- `type` — `api`/`gateway` add a latency SLI and latency SLO and a latency alert; `worker` adds throughput and saturation SLIs and a backlog alert.
- `has_dashboards` / `has_alerts` — false raises a coverage gap (no_dashboards / no_alerts; no_alerts is high for critical services).
- `non_functional_requirements.latency_p99_ms` — when set, the latency SLO uses it; otherwise it defaults to 500ms and raises a `no_latency_target` gap for api/gateway services.

**Validation:** `system_name` non-empty; at least one service; service names non-empty and unique.

## Output Schema

```json
{
  "system_name": "string",
  "service_observability": [
    {
      "service": "string",
      "slis": [ { "name": "string", "description": "string", "measurement": "string" } ],
      "slos": [ { "sli": "string", "objective": "string", "window": "string" } ],
      "recommended_dashboards": ["string"],
      "recommended_alerts": [ { "name": "string", "condition": "string", "severity": "page|ticket|info" } ]
    }
  ],
  "coverage_gaps": [
    { "severity": "low|medium|high", "category": "string",
      "description": "string", "services_affected": ["string"], "recommendation": "string" }
  ],
  "observability_score": "integer 0-100",
  "observability_rating": "string",
  "summary": "string"
}
```

**SLI rules:** every service gets `availability` and `error_rate`; `api`/`gateway` add `latency_p99`; `worker` adds `throughput` and `saturation`.

**Scoring:** start at 100; high gap −15, medium −8, low −3; floor 0.

**Rating bands:** 90–100 launch-ready · 75–89 directionally sound · 60–74 material gaps · 40–59 insufficient · 0–39 largely absent.

**Gap categories:** `missing_availability_target`, `no_alerts`, `no_dashboards`, `no_latency_target`.

**Output bounding:** SLIs/SLOs/alerts bounded by service type; gaps bounded by service count. No free-form text outside `summary`, `description`, `recommendation`, and the structured SLI/alert text fields.

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

INFO: `tool call started`, `tool call completed` (system, services, coverage_gaps, score), `tool call rejected`. WARN: `input_json missing`, `input_json failed to parse`. Metrics: calls by status; score distribution; gaps-per-call. Tracing: participates in transport trace; starts no spans.

## Determinism

Fully deterministic. Gaps sorted by severity descending then category. Same input → same output byte-for-byte (modulo `MarshalIndent` whitespace). See `testdata/obsplan-output-ecommerce.json`.

## Compatibility

**Version:** 1.0.0. **Non-breaking in 1.x:** new optional input fields; new output fields; new gap categories (handle unknown gracefully). **Breaking (2.0):** renaming/removing fields; type changes; scoring changes that change which inputs produce which scores.

## Example

See `testdata/obsplan-input-ecommerce.json` and `testdata/obsplan-output-ecommerce.json`.
