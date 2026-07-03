# Pattern: Async Messaging

## Problem

Synchronous service-to-service calls couple availability: if B is down, A can't respond. They also tie throughput together: if B is slow, A is slow. Async messaging decouples them in time — A produces a message, the broker holds it, B processes it whenever it's ready. The caller returns immediately; the recipient catches up at its own pace.

## Use When

- The work being requested doesn't need to complete before the caller responds (fire-and-forget or eventual completion)
- Producer throughput and consumer throughput differ — broker absorbs the difference
- Reliability matters — broker buffers messages during downstream outages
- Workflow involves multiple steps with no strict latency budget (saga, batch processing)

## Avoid When

- The caller needs the result synchronously (user-facing API expecting an immediate response)
- Latency budget is sub-second and end-to-end — broker adds 10–100ms
- The team can't handle eventual consistency in the UX

## Azure Implementation

### Implementation Steps

1. Identify operations safe to make asynchronous (notifications, downstream processing, analytics)
2. Choose the broker by need: Service Bus (reliable, ordered), Event Hubs (high-volume telemetry), Event Grid (reactive)
3. Define message schemas — versioned, backward-compatible
4. Implement producers with the transactional outbox pattern (atomic with DB write)
5. Implement consumers as idempotent receivers (handles at-least-once delivery)
6. Configure DLQ for poison messages; set max delivery attempts
7. Monitor queue depth, consumer lag, DLQ count

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Reliable messaging | Service Bus Queues/Topics | At-least-once delivery, sessions for ordering |
| Streaming events | Event Hubs | Partitioned, replay capable, high throughput |
| Reactive events | Event Grid | Pub-sub for Azure events, serverless |
| Producer | Outbox + relay (Container Apps) | DB-atomic event publication |
| Consumer | Container Apps worker | Idempotent processing, DLQ on poison |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Coupling | Strongly reduced — temporal decoupling between producer and consumer |
| Reliability | Improved — broker buffers during outages |
| Latency | Increased — message takes 10–100ms+ to traverse broker |
| Complexity | Higher — duplicate handling, ordering, DLQ all become consumer concerns |
| Debugging | Harder — no synchronous stack trace to follow |
| Cost | Broker cost ($10–500+/month per namespace) |

## Common Failure Modes

- **Hidden synchronous dependency** — Producer waits for ack from consumer (defeats the purpose).
  - Detection: Producer latency correlates with consumer latency.
  - Prevention: Fire-and-forget at the producer; correlate via callback or query, not blocking wait.

- **Unbounded queue growth** — Consumer slower than producer; queue depth grows forever.
  - Detection: Queue depth metric climbs without recovery.
  - Prevention: Alert on queue depth; auto-scale consumers; apply queue-based load leveling.

- **Poison messages clogging queue** — Bad message can't be processed; consumer retries forever, blocks queue.
  - Detection: Same message ID retried >N times.
  - Prevention: Cap delivery attempts; route to DLQ; alert on DLQ count.

- **Ordering assumptions break** — Code assumes messages arrive in send order; they don't (across partitions).
  - Detection: State machine errors ("OrderShipped before OrderPaid").
  - Prevention: Use sessions/partitions for per-aggregate ordering; design idempotent and order-tolerant consumers.

## Decision Signals

Use async messaging when:
- The action can complete out-of-band (notification, analytics, downstream processing)
- Producer and consumer have different throughput profiles
- You want to insulate one service from another's outages

Skip when:
- The caller's response depends on the work being done immediately
- Latency budget too tight for broker hop

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Service Bus | Reliable transactional messaging | DLQ, sessions, ordering within partition |
| Event Hubs | High-volume streaming | Partitioned log, replay, millions/sec |
| Event Grid | Reactive pub-sub | Azure-native event sources, serverless |
| Storage Queues | Simple queue | Cheap, basic, no advanced features |

## Go Implementation Notes

Service Bus consumer skeleton:
```go
receiver := client.NewReceiverForQueue("orders", nil)
for {
    msgs, _ := receiver.ReceiveMessages(ctx, 10, nil)
    for _, msg := range msgs {
        if err := handle(ctx, msg); err != nil {
            receiver.AbandonMessage(ctx, msg, nil) // retry
            continue
        }
        receiver.CompleteMessage(ctx, msg, nil) // ack
    }
}
```
Wrap `handle` with idempotency check via message ID and structured logging with correlation ID.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests async messaging when describing notifications, batch work, or downstream processing
- `detect_architecture_risks` — flags missing DLQ, missing idempotency, ordering assumptions across partitions
- `map_patterns_to_azure_services` — picks Service Bus vs. Event Hubs vs. Event Grid by workload
- `generate_message_schema` — drafts versioned message contracts

## Related Patterns

- **Transactional Outbox** — reliable production of messages
- **Idempotent Consumer** — required for safe at-least-once delivery
- **Saga** — coordinated workflow over async messages
- **Event-Driven Architecture** — broader pattern; async messaging is the mechanism

## References

- Skill: `../async-messaging.md` — choreography vs. orchestration, ordering, idempotency
- Pattern: `../../../microservices-data-architecture/references/patterns/transactional-outbox.md` — reliable producer
- Pattern: `../../../microservices-data-architecture/references/patterns/idempotent-consumer.md` — reliable consumer
- Pattern: `event-driven-architecture.md` — system-level async pattern
