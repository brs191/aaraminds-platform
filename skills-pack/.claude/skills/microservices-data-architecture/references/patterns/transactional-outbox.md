# Pattern: Transactional Outbox

## Problem

A service updates its database and then publishes an event — but the two operations live in different systems (database and message broker). If the broker is down or the service crashes between commit and publish, the event is lost. The database has the new state but no one knows. The transactional outbox makes the event part of the database transaction itself, then publishes asynchronously.

## Use When

- A service must publish an event whenever a domain state change is committed
- "At-least-once" delivery semantics are acceptable (consumers must be idempotent)
- The database supports transactions strong enough to write the domain row + outbox row atomically
- Losing an event is unacceptable (downstream services depend on it for consistency)

## Avoid When

- The service is read-only — there's nothing to publish
- The event is purely informational and dropping a few is fine (e.g., analytics ping)
- Your database doesn't support transactions across the relevant tables (some NoSQL stores)
- A simpler alternative fits: Change Data Capture (CDC) on the existing table

## Azure Implementation

### Implementation Steps

1. Add an `outbox` table to the service's database alongside its domain tables
2. In every state-changing transaction, INSERT an outbox row in the same transaction as the domain write
3. Run a background relay process that polls the outbox for unpublished rows
4. For each row, publish to Service Bus / Event Hubs and mark the row as `published` (or delete it)
5. Make the relay idempotent — if it crashes after publish but before marking, re-publishing on retry is safe (consumers are idempotent)
6. Monitor outbox depth; alert if the queue grows (relay is falling behind)

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Domain DB | Azure SQL / PostgreSQL | Same transaction writes domain row + outbox row |
| Outbox relay | Background worker (Container Apps job) | Polls outbox table on a tight loop |
| Event broker | Service Bus Topics or Event Hubs | Receives the published events |
| CDC alternative | Azure SQL Change Tracking | Skip the outbox table, watch the domain table directly |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Consistency | Strong — event publication is guaranteed once the DB commit succeeds |
| Latency | Adds polling delay (typically 100ms–1s) before event reaches consumers |
| Complexity | Requires a relay process and outbox schema; consumers must be idempotent |
| Database load | Extra writes per transaction; polling adds read load |
| Ordering | Per-aggregate ordering preserved; cross-aggregate ordering not guaranteed |

## Common Failure Modes

- **Relay falls behind** — Outbox table grows unbounded; events delivered hours late.
  - Detection: Outbox unpublished row count climbs steadily.
  - Prevention: Alert on outbox depth >N; scale relay horizontally; partition outbox by aggregate.

- **Duplicate publication** — Relay crashes after publish but before mark-published; restarts, re-publishes the same event.
  - Detection: Consumers see duplicate event IDs.
  - Prevention: Make the consumer idempotent (treat the outbox event ID as an idempotency key).

- **Polling overhead** — Tight polling on a large outbox table wastes DB CPU.
  - Detection: Database CPU spikes correlate with relay polling frequency.
  - Prevention: Use indexed `published=false` partial index; consider CDC for high volume.

- **Out-of-order delivery** — Relay processes rows in batches; concurrent publishers reorder events for the same aggregate.
  - Detection: Consumers see events out of sequence (state version 3 before version 2).
  - Prevention: Single-writer per aggregate ID; relay processes one aggregate sequentially.

## Decision Signals

Use transactional outbox when:
- A service must publish events that other services depend on for correctness
- You see code paths that update DB then publish — and the comment says "TODO: handle failure between"
- A consumer has had stale state because a previous event was lost

Skip when:
- CDC is available and simpler (let DB tooling watch the table directly)
- The event is fire-and-forget telemetry

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Azure SQL / PostgreSQL | Outbox table | ACID transactions ensure atomic write |
| Container Apps job | Relay process | Always-on background worker, scales horizontally |
| Service Bus | Event destination | Reliable delivery, DLQ for poison messages |
| Application Insights | Outbox depth metric | Alert if relay falls behind |

## Go Implementation Notes

Schema:
```sql
CREATE TABLE outbox (
  id UUID PRIMARY KEY,
  aggregate_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished ON outbox (occurred_at) WHERE published_at IS NULL;
```

Relay loop: `SELECT ... WHERE published_at IS NULL ORDER BY occurred_at LIMIT 100` → publish → `UPDATE ... SET published_at = now()`. Run inside `pgx` transaction or use SKIP LOCKED for concurrent relays.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends outbox whenever event publication accompanies a DB write
- `detect_architecture_risks` — flags services that publish events outside the DB transaction
- `generate_outbox_schema` — produces SQL DDL for the outbox table with correct indexes
- `generate_architecture_decision_record` — drafts the ADR comparing outbox vs. CDC vs. dual-write

## Related Patterns

- **Saga** — uses the outbox to publish saga step completions reliably
- **Idempotent Consumer** — required because the outbox guarantees at-least-once, not exactly-once
- **Event-Driven Architecture** — outbox is the reliable publisher building block
- **CQRS** — outbox events feed read model projectors

## References

- Skill: `../data-architecture.md` — outbox in the context of saga and CQRS
- Pattern: `idempotent-consumer.md` — required downstream for at-least-once delivery
- Pattern: `../../../microservices-async-messaging/references/patterns/event-driven-architecture.md` — outbox as the foundation for reliable events
