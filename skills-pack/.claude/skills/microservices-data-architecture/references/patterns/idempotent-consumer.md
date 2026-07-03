# Pattern: Idempotent Consumer

## Problem

Reliable messaging systems guarantee at-least-once delivery — which means the same message may arrive twice. The same is true of HTTP retries, saga step replays, and outbox relays. If a consumer naively processes every delivery, duplicates cause double-charges, double-shipments, and corrupted state. An idempotent consumer treats the second delivery of the same logical message as a no-op.

## Use When

- Messages arrive over at-least-once channels (Service Bus, Event Hubs, Kafka)
- Retries are enabled (HTTP retry middleware, saga replay, manual reprocessing)
- The consumer's action has side-effects (charge, ship, send email, update state)
- Operations team needs to safely replay messages from a dead-letter queue

## Avoid When

- The operation is naturally idempotent already (PUT replacing a value, setting a flag)
- The consumer is purely read-only (a query that triggers no state change)
- Duplicates are detected and handled upstream (deduplication at the broker level)

## Azure Implementation

### Implementation Steps

1. Choose an idempotency key per message — `messageId`, `correlationId`, or a domain-specific deduplication key
2. Maintain a deduplication store: SQL table, Cosmos container, or Redis with TTL
3. On every incoming message: check if the key has been processed; if yes, ack and skip
4. Process and persist the side-effect AND the dedup key in the same transaction
5. Set retention on dedup store appropriate to redelivery window (Service Bus duplicate detection: max 7 days)
6. Monitor "duplicate detected" rate; spikes indicate broker or relay misbehavior

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Dedup store | Azure SQL table | `processed_messages(message_id, processed_at)` |
| Fast dedup | Redis | SET with NX flag + TTL |
| Broker-level dedup | Service Bus Premium | Built-in duplicate detection (window up to 7 days) |
| Distributed dedup | Cosmos DB | Item per message ID, TTL-based cleanup |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Correctness | Strongly improved — duplicates can't corrupt state |
| Storage cost | Dedup store grows with traffic; needs retention policy |
| Latency | Adds a read (sometimes a write) per message |
| Operational safety | Enables safe DLQ replay and reprocessing |
| Window limits | Dedup is bounded; messages older than retention can re-process |

## Common Failure Modes

- **Dedup check outside the transaction** — Consumer checks dedup, processes, then writes the dedup key separately. Crash between leaves no dedup record, next delivery re-processes.
  - Detection: Audit shows duplicate side-effects with same message ID after a crash.
  - Prevention: Write dedup key in the same DB transaction as the side-effect.

- **Wrong idempotency key** — Using a key that's not unique per logical operation (e.g., customer ID instead of order ID), causing legitimate distinct messages to be silently dropped.
  - Detection: Customer reports missing operations; logs show "duplicate skipped" for valid messages.
  - Prevention: Key per logical operation, not per actor; document the key choice.

- **Dedup TTL too short** — Message redelivered after key expired; processed twice.
  - Detection: Spike in duplicates after broker incident with long delivery delay.
  - Prevention: Set TTL ≥ max broker retention window (Service Bus default: 14 days).

- **No dedup at all** — Consumer trusts "exactly-once" claims from broker that don't actually hold under failure.
  - Detection: Double-charges/double-ships in production.
  - Prevention: Always implement consumer-side dedup; treat broker guarantees as best-effort.

## Decision Signals

Implement idempotent consumer when:
- Any consumer reads from a message queue or event stream
- HTTP handler accepts retries (POST with retry policy)
- A saga step receives messages

Skip only when:
- Operation is naturally idempotent (PUT, DELETE, "set X to Y")
- The consumer has no side-effects (read-only query handler)

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Service Bus | Source channel | Use message ID or session ID as dedup key |
| Service Bus Premium | Built-in dedup | Enable duplicate detection (broker-side) |
| SQL Server | Dedup persistence | Transactional dedup with side-effect write |
| Redis | Fast dedup | Low-latency `SETNX` for high-throughput consumers |

## Go Implementation Notes

Pattern using SQL:
```sql
INSERT INTO processed_messages (message_id, processed_at)
VALUES ($1, now())
ON CONFLICT (message_id) DO NOTHING
RETURNING message_id;
```
If RETURNING is empty, this is a duplicate — skip side-effect and ack. Otherwise, perform side-effect in the same transaction as the INSERT, then commit and ack.

For Redis: `SET key:<msgID> 1 NX EX 86400` — if returns nil, duplicate.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends idempotent consumer for every message receiver and retried HTTP endpoint
- `detect_architecture_risks` — flags consumers without dedup logic on at-least-once channels
- `generate_idempotency_design` — produces idempotency key strategy and storage choice for the described workload

## Related Patterns

- **Transactional Outbox** — produces at-least-once events that consumers must dedupe
- **Saga** — every step receiver must be idempotent
- **Retry-Timeout** — caller retries amplify the duplicate-delivery problem

## References

- Skill: `../../../microservices-async-messaging/references/async-messaging.md` — idempotency in messaging context
- Pattern: `transactional-outbox.md` — produces the duplicates this pattern handles
- Pattern: `saga.md` — every saga step receiver must be idempotent
