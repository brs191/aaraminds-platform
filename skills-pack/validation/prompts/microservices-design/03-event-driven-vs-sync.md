---
id: microservices-design/03-event-driven-vs-sync
area: microservices-design
exercises:
  - .claude/skills/microservices-async-messaging/references/async-messaging.md
  - .claude/skills/microservices-async-messaging/references/patterns/event-driven-architecture.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Event-driven vs. synchronous for a notification flow

## Context

Attach `07-async-messaging.md` and the event-driven-architecture pattern card.

## Prompt

When an order is created, we currently call notification-service synchronously from order-service to send the customer confirmation email. The team is considering switching to an event-driven approach: order-service emits `OrderCreated`, notification-service subscribes. Walk me through the decision: when is sync the right answer, when is event-driven? Apply both to *this specific case* and recommend.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Sync vs. async decision criterion** — articulates the rule (sync when caller needs the result before responding; async when fire-and-forget or downstream is decoupled)
- [ ] **Coupling analysis** — points out that synchronous coupling means order-service is down if notification-service is down
- [ ] **Latency analysis** — notes that sync adds notification-service latency to the order API; async returns immediately
- [ ] **Customer expectation** — addresses whether the user expects to see "email sent" confirmation, or just "order placed" (the latter is the realistic case)
- [ ] **Recommends event-driven for *this* case** — concrete recommendation, not a both-sides answer
- [ ] **At-least-once + idempotency** — flags that switching to async means notification-service must dedupe (otherwise: duplicate emails on retry)
- [ ] **Outbox or equivalent reliable publishing** — calls out that order-service must publish reliably (transactional outbox or CDC), not naive fire-after-commit
- [ ] **Anti-pattern: "event" that's secretly a command** — distinguishes `OrderCreated` (event, past tense, multiple consumers possible) from `SendOrderConfirmationEmail` (command directed at one consumer); the proposed change is genuinely event-shaped

## Reference output

> **When sync, when async**
>
> Sync is right when the caller needs the result before it can respond to *its* caller. Examples: `pricing-service` during cart update (the user is staring at the total), `auth-service` during login (we can't proceed without the token). The caller blocks because there is nothing useful to do without the result.
>
> Async (event-driven) is right when the caller can return without the downstream completing. Three signs: (1) the downstream's output isn't needed in the response, (2) multiple downstreams may eventually care, (3) decoupling availability is valuable.
>
> **This case: send confirmation email after order created.**
>
> The customer-facing API can respond "order placed" the moment the order row commits. The email goes out a few seconds later. The user does not need to see "email sent" — they need to see "order placed" and trust the email will follow. Notification-service being down for 30 seconds shouldn't prevent orders. So: this is async-shaped.
>
> Today's coupling problem made concrete: if notification-service is down, every `POST /orders` either fails or hangs for the call timeout. Order-service's availability is bounded by notification-service's, even though one is critical and one is not. That's the textbook "synchronous coupling poisons your SLA" failure.
>
> **Recommendation: switch to event-driven for this flow.**
>
> Order-service emits `OrderCreated` (past tense, describes a fact). Notification-service subscribes to that event from its own queue/subscription. Order-service no longer knows or cares who consumes the event — analytics, ML training, audit, and notification can all subscribe independently. If notification-service is down, events queue up on its subscription and process when it returns; no impact on order placement.
>
> **What you need with this change**
>
> 1. **Reliable publication from order-service.** A naive "INSERT order; publish event" is wrong: the broker can be unreachable between the two operations and the event is lost. Use the transactional outbox pattern — write the event into an `outbox` table in the same DB transaction as the order row, and a relay process publishes from there.
>
> 2. **Idempotent notification-service.** The broker delivers at-least-once. Without dedup, a retry sends the email twice. Maintain a `processed_messages(event_id)` table in notification-service; check it before sending; insert it in the same transaction as the send-log row. Duplicate delivery becomes a no-op.
>
> 3. **DLQ + alerting.** Notification-service's subscription needs a dead-letter queue for poison messages (e.g., a malformed event that fails parsing). Alert if DLQ depth grows.
>
> 4. **Trace correlation.** The event carries the order's correlation ID; notification-service's logs and traces tie back to the order so debugging "why did this user not get an email?" is one query.
>
> **Anti-pattern to avoid**
>
> Don't make the event `SendOrderConfirmationEmail`. That's a command directed at one consumer — it has all the coupling drawbacks of the sync call (order-service knows the consumer's intent) without the synchronous safety net (no immediate confirmation that the command was carried out). Real events are facts: `OrderCreated`, `PaymentAuthorized`. Consumers decide what to do with them.
>
> **Trade-off summary**
> - Better: order-service no longer bottlenecked by notification health; notification can scale and fail independently; new consumers (analytics, ML) add without changing order-service.
> - Worse: more moving parts; debugging an "email not received" complaint now requires tracing across the broker; eventual consistency means there's a window where the order exists and the email hasn't arrived yet (set customer expectation appropriately).

## Notes

The "command masquerading as event" trap is critical — it's the most common mistake when teams first try event-driven. Catches LLMs that say "use events" without engaging with the shape of the event.
