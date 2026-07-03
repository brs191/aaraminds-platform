---
name: microservices-async-messaging
description: Decides sync vs async service communication and designs the messaging topology for Azure-hosted microservices — broker selection (Service Bus, Event Grid, Event Hubs), pub-sub vs queue semantics, ordering, delivery guarantees, dead-letter handling, and distributed tracing across async boundaries. Use when adding an inter-service call and deciding sync vs async, choosing between Service Bus / Event Grid / Event Hubs, designing an event topology, or debugging a tracing gap across an async boundary. Do not use for cross-service data consistency like saga or outbox (use microservices-data-architecture).
version: 1.0.1
last_updated: 2026-05-30
---

# Microservices Async Messaging

## When to use

Trigger this skill when the question is about how services talk to each other and whether the talking should be synchronous or asynchronous. Common triggers: "should this be a REST call or a message," "which Azure broker do I use," "we need ordering — does Service Bus or Event Hubs fit," "tracing breaks at the message boundary," "consumer is overwhelmed during peak."

Do **not** use this skill for: cross-service data consistency patterns like saga, outbox, or CQRS (use `microservices-data-architecture`); resilience controls like retry and circuit breaker (`microservices-resilience`); SLO design or alert configuration (`azure-microservices-observability`).

## The critical decision rule — async by default for cross-service writes

For *cross-service write* paths in a microservices architecture, **async messaging is the default**. Synchronous chains across services compound latency, compound failure (one slow downstream stalls the whole chain), and break independent deployability. Reach for sync only when the caller genuinely needs the response inline — typically a read-style operation, or when the user is waiting in real time.

For *reads* across services, sync is fine when latency budget permits, the data is small, and the downstream is reliable. Otherwise read via a local projection populated by events (see `microservices-data-architecture`).

## The communication-shape selector

| Situation | Shape | Azure choice |
|---|---|---|
| Caller needs the answer to respond to its caller (read on user path) | **Sync HTTP** or **sync gRPC** | Direct call, possibly through API Management |
| Caller emits a fact for any number of subscribers (order created, payment authorized) | **Async pub-sub** | Service Bus topic (commands with ordering) or Event Grid (events, push) |
| Caller hands off work to one consumer with at-least-once + DLQ | **Async queue** | Service Bus queue |
| High-volume telemetry / clickstream / IoT (millions of events/sec) | **Async stream** | Event Hubs (partitioned, kept for replay) |
| Caller wants to react to Azure resource events (blob uploaded, etc.) | **Reactive event** | Event Grid system topic |

See `references/async-messaging.md` for the broker-by-broker comparison with throughput, ordering, retention, and cost notes.

## Communication-design logic

1. **For every new inter-service call, ask first:** does the *caller* need the result inline to respond? If no, default to async. If yes, sync is acceptable but evaluate whether the caller can degrade gracefully when the call fails.

2. **For async, choose the broker by semantics, not preference:**
   - **Service Bus queue** — point-to-point work handoff with at-least-once delivery, DLQ, ordering within a session. Default for commands.
   - **Service Bus topic** — pub-sub with filtered subscriptions. Use when multiple services need the same event with selective filtering.
   - **Event Grid** — push-based, low-latency event notifications, native integration with Azure resource events. Use for reactive workflows. No replay; events expire.
   - **Event Hubs** — partitioned stream, high throughput, replayable. Use for telemetry, clickstream, IoT, or when consumers may want to replay history.

3. **For ordering:** Service Bus *sessions* give per-session FIFO. Event Hubs gives per-partition order. Across the system, do not assume global order — design idempotency assuming reordering.

4. **For delivery guarantees:** brokers give **at-least-once**, never exactly-once. Consumers must be idempotent. See `microservices-data-architecture` → idempotent consumer pattern.

5. **For dead-letter:** every Service Bus queue/topic has a DLQ. Configure max delivery count (default 10 is often too high; 3-5 is more useful), and have an actual operational process for the DLQ — alerts on messages landing in DLQ, runbook for inspection and replay.

6. **For tracing across async boundaries:** propagate W3C `traceparent` context as a *message property*. OpenTelemetry SDKs do this automatically if configured; without it, the trace breaks at every queue. See `references/patterns/distributed-tracing.md`.

## Worked example — brownfield: replacing a sync chain with async events

Setup: existing order flow on AKS makes a sync REST chain `order → inventory → payment → notification`. End-to-end p99 latency is 4.2 seconds; the user sees a 4-second spinner on checkout. Failures cascade: when notification is slow, the whole chain stalls because the chain is synchronous.

Decision walk:

1. **Identify which steps the caller needs inline.** The user pressing "Place Order" needs to know inventory was reserved and payment was authorized; everything else (notification, fulfillment, analytics) can happen after the response. The synchronous boundary moves: `order → inventory → payment` stays sync; `notification` and downstream become async.
2. **Emit an `OrderConfirmed` event after payment authorization.** Publish to a Service Bus topic with subscriptions for `notification-service`, `fulfillment-service`, and `analytics-service`. Each consumer is independent; one's failure does not block the others. See `references/patterns/event-driven-architecture.md`.
3. **Use Service Bus topic, not Event Grid.** Reasons: we need DLQ for poison messages, we need per-customer session ordering for some consumers (e.g., notification ordering per user), and consumers may want filter subscriptions later. Event Grid lacks DLQ and session ordering. See `references/async-messaging.md`.
4. **Propagate trace context.** Include `traceparent` as a Service Bus message property; consumers extract it on receive. Without this, the order-placement trace ends at the publish and a separate trace begins at the consumer, breaking root-cause analysis. See `references/patterns/distributed-tracing.md`.
5. **Idempotent consumers.** Notification service uses the `orderId` as the dedup key (don't send the same confirmation email twice). See `microservices-data-architecture` → idempotent consumer.
6. **Measure the win.** p99 user-visible latency drops from 4.2 s to ~600 ms (removes notification's 3 s tail). Notification is now eventually consistent — emails arrive within 2-5 seconds; acceptable.

## Anti-pattern — using sync REST for fire-and-forget side effects

**Bad:** "After we create the order, call the notification service synchronously to send the confirmation email." The order service blocks on the notification service's response. If notification is slow, the order endpoint is slow. If notification is down, the order endpoint returns 500 even though the order was created.

**Why it fails:** Coupling two services' availability into a single failure mode for no benefit. The user doesn't need to wait for the email; they just need to know the order was placed. The sync call adds latency, adds a failure mode, and forces tight coupling of deployment lifecycle.

**Detection signal:** look for synchronous calls in the request path that produce side effects the user is not waiting for: `notificationClient.send(...)`, `analyticsClient.track(...)`, `auditClient.log(...)`. If the user can't tell whether it ran, it shouldn't be in the synchronous path.

**Fix:** Publish an event to Service Bus topic after the local transaction commits. Consumers handle the side effect. Use the transactional outbox pattern so the event publish cannot fail independently of the local transaction. See `microservices-data-architecture`.

## Verification questions

1. For every cross-service write: is the caller's caller actually waiting for this result, or could it be async?
2. For every async producer: is the broker choice justified by semantics (ordering, DLQ, replay), not by familiarity?
3. For every consumer: is it idempotent — would re-delivery produce the same outcome?
4. For every Service Bus queue/topic: is there a DLQ alert and a documented inspection runbook?
5. For traces that cross async boundaries: does `traceparent` propagate as a message property, and do consumer spans link to the producer's trace?
6. For event topologies: is there a documented event catalog (event name, producer, consumers, schema version) — or is it tribal knowledge?

## What to read next

- `references/async-messaging.md` — broker comparison (Service Bus / Event Grid / Event Hubs), ordering semantics, DLQ design
- `references/patterns/async-messaging.md` — pub-sub vs. queue patterns in detail
- `references/patterns/event-driven-architecture.md` — event topology, choreography vs. orchestration, event catalog
- `references/patterns/distributed-tracing.md` — W3C context propagation across async hops, OpenTelemetry config
- `microservices-data-architecture` skill — outbox, idempotent consumer, saga (the patterns that ride on this transport)
- `azure-service-mapping` skill — for the broader Azure-service decision context
