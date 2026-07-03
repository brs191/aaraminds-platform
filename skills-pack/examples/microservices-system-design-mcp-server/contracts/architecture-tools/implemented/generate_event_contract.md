# Tool Contract: generate_event_contract

## Status

Implemented

## Purpose

Generate a CloudEvents v1.0-shaped event contract for a domain event: producer, consumer subscriptions, payload schema with envelope fields (event_id, occurred_at, correlation_id), Azure transport binding (Service Bus / Event Grid / Event Hubs), ordering and idempotency semantics, per-consumer guidance, warnings on weak event shapes, and a markdown rendering.

## Risk Level

Low. Informational/generative; no broker resources are provisioned.

## Approval Required

No.

## Input Schema

```json
{
  "system_name": "string (required)",
  "event_name": "string (required) — past-tense, e.g. 'OrderCreated'",
  "producer": "string (required) — the emitting service",
  "consumers": ["string", "..."],
  "fields": [
    {
      "name": "string (required)",
      "type": "string | integer | boolean | object | array | date-time | uuid (required)",
      "required": "boolean",
      "description": "string",
      "sensitive": "boolean — PII / PHI / PCI"
    }
  ],
  "transport": "service_bus | event_grid | event_hubs (default: service_bus)",
  "ordering": "none | per_aggregate | global (default: per_aggregate)",
  "description": "string"
}
```

## Output Schema

```json
{
  "system_name": "string",
  "event_name": "string",
  "producer": "string",
  "consumers": [{"service": "string", "notes": "string"}],
  "schema": {
    "specversion": "1.0",
    "type": "string — com.<system>.<domain>.<event>",
    "source": "string — urn:<system>:<producer>",
    "subject": "string — <aggregate-id>",
    "dataschema": "string — schemas/<event>.v1.json",
    "data": [{"name", "type", "required", "description", "sensitive", "handling_note"}]
  },
  "transport": {
    "azure_service": "Azure Service Bus | Azure Event Grid | Azure Event Hubs",
    "topic_or_entity": "string",
    "subscriptions": ["string", "..."],
    "dlq": "string"
  },
  "ordering": "string",
  "idempotency_key": "string",
  "warnings": ["string", "..."],
  "markdown": "string — rendered contract",
  "quality_score": "integer 0-100"
}
```

## Rules

- Adds envelope fields automatically: `event_id` (idempotency key), `occurred_at` (producer timestamp), `correlation_id` (cross-service tracing).
- Detects command-shaped event names (`Create`, `Send`, `Make`, or non-past-tense) and surfaces a warning. Events describe past facts; commands belong in tool calls.
- For Service Bus: built-in DLQ. For Event Grid: blob-storage dead-lettering. For Event Hubs: explicit warning that DLQ is consumer-side responsibility.
- Sensitive fields get a handling note and trigger a sensitive-fields warning to remind teams of log redaction and audit emitter sanitisation.

## Errors

- `system_name is required`
- `event_name is required`
- `producer is required`
- `at least one field is required`
- `fields[N].name is required`
- `fields[N].type is required`

## Score

100 minus 10 per warning. Common warnings: command-shaped name, no consumers, no correlation_id supplied, Event Hubs transport, sensitive fields.
