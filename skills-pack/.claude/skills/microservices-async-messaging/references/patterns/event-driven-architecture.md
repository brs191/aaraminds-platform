# Pattern: Event-Driven Architecture

## Problem

Request-response architectures force the caller to know about every service that should react to a change. The order service must remember to call payment, inventory, notification, and analytics — and every new consumer requires a code change in the order service. Event-driven architecture inverts the dependency: the producer announces what happened; any consumer can subscribe without the producer knowing.

## Use When

- Multiple downstream services react to the same business event (order created → email, analytics, inventory, ML training)
- The producer should not know who consumes its events
- New consumers should be added without modifying the producer
- The system is naturally event-shaped (state changes drive workflow)

## Avoid When

- The producer needs a response from a specific consumer (use request-response)
- There's only one consumer and the call is request-shaped (just call the API)
- Strict ordering across consumers is required (events are pub-sub; ordering is per-stream at best)
- The team can't tolerate eventual consistency

## Azure Implementation

### Implementation Steps

1. Identify domain events worth publishing: state changes other services care about (OrderCreated, PaymentApproved)
2. Define event schemas: versioned, self-describing, include all data consumers need (no callbacks)
3. Choose the broker: Service Bus Topics (transactional), Event Grid (Azure-native), Event Hubs (high volume)
4. Producers publish via outbox pattern (atomic with DB write)
5. Consumers subscribe to topics; each consumer has its own queue/subscription
6. Consumers are idempotent (at-least-once delivery)
7. Document the event catalog: name, schema, owner, consumers

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Topic-based pub-sub | Service Bus Topics + Subscriptions | One subscription per consumer, filters supported |
| Azure-native events | Event Grid | Storage, App Service, custom topics |
| High-volume streams | Event Hubs | Partitioned, replayable, hours/days retention |
| Schema registry | Custom or Azure Schema Registry | Track event versions and compatibility |
| Event catalog | Internal docs portal | Producer/consumer mapping for governance |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Coupling | Strongly reduced — producer doesn't know consumers |
| Extensibility | New consumers add without producer change |
| Consistency | Eventual; "system state" lives across many services |
| Debugging | Harder — no single call graph; events fan out |
| Schema discipline | Critical — bad event design propagates everywhere |
| Operational overhead | Event catalog, schema registry, DLQs per subscription |

## Common Failure Modes

- **Event as command** — Producer thinks "PlaceOrder" is an event; it's actually a command directed at the order service.
  - Detection: "Event" has exactly one consumer; consumer's failure means producer's intent failed.
  - Prevention: Events describe past facts (`OrderPlaced`, past tense); commands name imperatives (`PlaceOrder`).

- **Schema breakage cascade** — Producer changes event schema; 7 downstream consumers break silently.
  - Detection: Consumer DLQ spikes after producer deploy.
  - Prevention: Backward-compatible schema changes only; new fields with defaults; deprecate before removing.

- **Lost event = lost state** — Producer publishes outside DB transaction; broker outage drops event; downstream state diverges forever.
  - Detection: Periodic reconciliation finds mismatch between producer and consumer state.
  - Prevention: Transactional outbox for publication; reconciliation jobs as backstop.

- **Spaghetti event flow** — Dozens of events, no documentation; nobody can trace a business workflow.
  - Detection: New engineers can't explain how X happens.
  - Prevention: Event catalog with producer/consumer/schema; visualize flows for key workflows.

## Decision Signals

Adopt event-driven architecture when:
- Multiple services consume the same business event
- Frequent "add consumer X to this notification" tasks
- Producer should not own the list of consumers

Skip when:
- Single producer, single consumer, command shape — just call the API
- System-wide strict ordering across events required

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Service Bus Topics | Reliable pub-sub | Per-consumer subscription, DLQ, filters |
| Event Grid | Reactive events | Serverless, native Azure event sources |
| Event Hubs | Stream / log | High volume, partitioned, replayable |
| Schema Registry | Schema management | Compatibility enforcement |

## Go Implementation Notes

Event envelope:
```go
type EventEnvelope struct {
    ID            uuid.UUID     `json:"id"`
    Type          string        `json:"type"`     // "order.created"
    Version       int           `json:"version"`  // 1
    OccurredAt    time.Time     `json:"occurredAt"`
    CorrelationID string        `json:"correlationId"`
    Payload       json.RawMessage `json:"payload"`
}
```
Producers publish via outbox. Consumers register handlers per event type; handlers idempotent. Use envelope ID as idempotency key.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends EDA when describing fan-out from one producer to multiple consumers
- `detect_architecture_risks` — flags events without producers' outbox, missing schema versioning, "event" that's really a command
- `generate_event_catalog` — produces the event registry from described workflows
- `generate_architecture_decision_record` — drafts ADR for EDA vs. orchestration vs. request-response

## Related Patterns

- **Async Messaging** — the mechanism EDA runs on
- **Transactional Outbox** — reliable event publication
- **Idempotent Consumer** — required at every subscriber
- **CQRS** — natural pair; events drive read-model projections
- **Saga** — orchestrated EDA for multi-step workflows

## References

- Skill: `../async-messaging.md` — pub-sub mechanics, ordering, idempotency
- Pattern: `async-messaging.md` — underlying transport
- Pattern: `../../../microservices-data-architecture/references/patterns/transactional-outbox.md` — reliable producer
- Pattern: `../../../microservices-data-architecture/references/patterns/idempotent-consumer.md` — reliable consumer
