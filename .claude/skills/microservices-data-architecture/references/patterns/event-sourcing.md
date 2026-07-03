# Pattern: Event Sourcing

## Problem

Storing only the current state loses the history of how it got there. "Why did this customer's status become inactive?" requires audit logs, scattered audit tables, or guesswork from change timestamps. Event sourcing stores the full sequence of state-changing events as the source of truth; the current state is derived by replaying the events. Audit becomes free, and you gain the ability to project new views from history.

## Use When

- Audit, compliance, or regulatory requirements demand a tamper-evident history of all changes
- The domain is naturally event-shaped (banking transactions, IoT readings, version-controlled documents)
- You need to derive multiple read models from the same write side (pairs with CQRS)
- Time-travel debugging or "what was the state at time T?" queries are required

## Avoid When

- The domain is CRUD with no audit requirements — the cost dominates the benefit
- The team has no experience with event modeling; learning curve is steep
- Storage growth is unacceptable (events accumulate forever unless snapshotted)
- Strong global consistency across aggregates is required

## Azure Implementation

### Implementation Steps

1. Model the domain as aggregates that emit events (OrderPlaced, OrderShipped, OrderCancelled)
2. Choose an event store: Cosmos DB, Azure SQL with an `events` table, or a specialized event-store-as-a-service
3. Append events with optimistic concurrency control (version number per aggregate)
4. Build current state by replaying events for the aggregate from the start (or from a snapshot)
5. Add snapshots for aggregates with long event histories (read state from snapshot + recent events)
6. Project events into read models (search, dashboard, etc.) via projectors
7. Plan event schema evolution — events are immutable; add new fields with defaults

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Event store | Cosmos DB | Partition by aggregate ID, append-only, optimistic concurrency via `etag` |
| Event store (SQL) | Azure SQL | `events` table with `(aggregate_id, version)` unique constraint |
| Snapshots | Blob Storage or same store | One row per aggregate with state + version |
| Projectors | Container Apps workers | Subscribe to events, update read stores |
| Schema registry | Custom or Azure Schema Registry | Track event versions and migrations |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Auditability | Perfect — every state change is preserved |
| Storage cost | Grows unbounded; snapshots and retention policies needed |
| Read performance | Slow without snapshots or projections; fast with them |
| Schema flexibility | Events are immutable; evolution requires versioning discipline |
| Complexity | High — event modeling, snapshotting, projections, replay tooling |
| Debugging | Easier (time-travel) but requires tooling |

## Common Failure Modes

- **Unbounded event growth** — Aggregate has thousands of events; reads become slow.
  - Detection: P99 read latency climbs over months; replay times exceed seconds.
  - Prevention: Snapshot every N events (e.g., 100); load latest snapshot + tail events.

- **Schema evolution breaks replay** — Old events lack a field added later; replay code crashes on missing data.
  - Detection: Replay fails on historical aggregates after deploy.
  - Prevention: Version events; upcasters convert old events to current shape; never remove fields.

- **Concurrent write conflict storms** — High-contention aggregate (popular product) sees many optimistic-concurrency rejections.
  - Detection: Retry rate on commands climbs; user-facing latency spikes.
  - Prevention: Re-shape the aggregate boundary (split into smaller aggregates); or serialize via single-writer.

- **Event store as a queue** — Treating the event store like a message broker; subscribers compete on the same partition.
  - Detection: Projectors miss events or duplicate processing.
  - Prevention: Keep event store as a log; use a separate broker (Service Bus / Event Hubs) for fan-out.

## Decision Signals

Use event sourcing when:
- Regulatory or business requirement: full audit trail of state changes
- You repeatedly answer "what was the state at time T?"
- The domain has multiple read views that should be derived from the same history

Skip when:
- Simple CRUD with no audit need
- Team unfamiliar with event-driven thinking

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Cosmos DB | Event store | Partition by aggregate, append-only, optimistic concurrency |
| Azure SQL | Alternative event store | Familiar tooling, transactions, indexes on aggregate ID |
| Blob Storage | Snapshots | Cheap, durable, periodic state checkpoints |
| Service Bus | Fan-out to projectors | Reliable delivery, DLQ, decouples store from consumers |

## Go Implementation Notes

Event interface:
```go
type Event interface {
    AggregateID() uuid.UUID
    EventType() string
    OccurredAt() time.Time
}
```
Aggregate root replays events: `func (a *Order) Apply(e Event)` updates state. Repository: load events for ID, replay, return aggregate. Save: append new events with version check (`WHERE current_version = expected_version`).

Snapshot strategy: every 100 events, write snapshot to blob; on load, read latest snapshot + events after.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests event sourcing when audit, history, or replay requirements appear
- `detect_architecture_risks` — flags event sourcing without snapshots on long-lived aggregates
- `generate_event_schema` — drafts initial event types and aggregate model from a described domain
- `generate_architecture_decision_record` — drafts ADR comparing event sourcing vs. CDC vs. audit tables

## Related Patterns

- **CQRS** — natural pair; events drive projections to read models
- **Transactional Outbox** — alternative when full event sourcing is too heavy
- **Saga** — saga events can be sourced into the event log
- **Snapshot** — required for performance on long aggregate histories

## References

- Skill: `../data-architecture.md` — event sourcing vs. CRUD with audit vs. CDC
- Pattern: `cqrs.md` — projections from event stream
- Pattern: `transactional-outbox.md` — lighter-weight alternative when full ES is too heavy
