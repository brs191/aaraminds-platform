# Tool Contract: generate_resilience_plan

## Status

Implemented

## Purpose

Generate a resilience plan for a microservices system: per-dependency timeout / retry / circuit-breaker configuration, bulkhead notes, queue-based load-leveling notes for workers, fallback strategies, detection signals with alert thresholds, recommended next steps, and a coverage score.

## Risk Level

Low. Informational; the plan is reviewed and implemented by humans.

## Approval Required

No.

## Input Schema

```json
{
  "system_name": "string (required)",
  "description": "string",
  "services": [
    {
      "name": "string (required)",
      "type": "api | gateway | worker | function",
      "criticality": "high | medium | low",
      "stateful": "boolean",
      "replicated": "boolean"
    }
  ],
  "dependencies": [
    {
      "from": "string (required)",
      "to": "string (required)",
      "idempotent": "boolean"
    }
  ],
  "external_apis": [
    {
      "name": "string",
      "used_by": ["string", "..."],
      "stated_sla": "string"
    }
  ],
  "non_functional_requirements": {
    "availability_target": "string",
    "latency_p99_ms": "integer"
  }
}
```

## Output Schema

```json
{
  "system_name": "string",
  "dependency_controls": [
    {
      "from": "string",
      "to": "string",
      "timeout": "string",
      "retry_attempts": "integer",
      "retry_backoff": "string",
      "circuit_breaker": "string",
      "idempotency_key": "boolean",
      "notes": "string"
    }
  ],
  "bulkhead_notes": [{"service": "string", "notes": "string"}],
  "load_leveling_notes": [{"service": "string", "notes": "string"}],
  "fallbacks": [{"dependency": "string", "strategy": "string", "detail": "string"}],
  "detection_signals": [{"signal": "string", "source": "string", "alert_when": "string"}],
  "next_steps": ["string", "..."],
  "coverage_score": "integer 0-100",
  "summary": "string"
}
```

## Rules

- High-criticality boundary (caller or callee is high-crit): tighter timeouts (~1.5s), fewer retries (2), tighter breaker (3 failures → open 60s).
- Medium boundary: 2s timeout, 3 retries, breaker (5 failures → open 30s).
- External API: always 5s timeout, 2 retries, error-rate breaker (10% over 1 min), idempotency key required for mutations, classify errors before retrying.
- Idempotent dependency: explicit idempotency key not required (operation is naturally safe).
- Single-replica high-crit service: bulkhead note recommending replication.
- Worker service: load-leveling note recommending queue-based smoothing with DLQ.
- Fallback strategy: `fail_fast` for high-crit dependencies; `degrade` for others (cache/default with explicit status surfacing).

## Detection signals

The plan always includes the standard set:
- Circuit-breaker state (alert if open >60s)
- Retry count per request (alert if avg > 0.5 over 5 min)
- Dependency P99 latency (alert if > target × 2 over 5 min)
- DLQ depth (alert on any non-zero depth)
- Request rejection rate (alert if > 1% over 5 min)

## Errors

- `system_name is required`
- `at least one service is required`
- `services[N].name is required`
- `dependencies[N] requires both from and to`

## Coverage score

Composed: 40 for dependency controls, 15 each for bulkhead, load-leveling, fallbacks, detection signals. Always returns 100 when all sections are populated.
