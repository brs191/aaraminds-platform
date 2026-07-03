---
id: architecture-review/01-saga-design-review
area: architecture-review
exercises:
  - .claude/skills/microservices-data-architecture/references/patterns/saga.md
  - .claude/skills/microservices-data-architecture/references/patterns/idempotent-consumer.md
  - .claude/skills/microservices-data-architecture/references/patterns/transactional-outbox.md
pass_threshold: 7/9
last_run: 2026-05-30
last_result: pass
---

# Review a saga design for compensation gaps

## Context

Attach the saga, idempotent-consumer, and transactional-outbox pattern cards.

## Prompt

A team's order saga design:
- Step 1: order-service creates order (status=pending)
- Step 2: inventory-service reserves stock
- Step 3: payment-service charges card
- Step 4: notification-service sends confirmation email
- Step 5: order-service marks order as confirmed

Each step publishes an event to Service Bus. There is no orchestrator; consumers react to events. There is no outbox; each service publishes after its DB commit. There are no idempotency checks at consumers. Review this design and flag every problem you'd block on in code review.

## Rubric

A response passes if it covers at least 7 of the following 9 points:

- [ ] **Missing outbox** — flags that "publish after commit" loses events if the broker is unreachable between the commit and the publish
- [ ] **Missing idempotency** — flags that without dedup, retries cause double-reservation, double-charge, double-email
- [ ] **Email is not a saga step** — calls out that sending an email cannot be compensated ("un-send the email" is impossible); notification belongs *outside* the saga or as a fire-and-forget post-completion side effect
- [ ] **No compensation defined** — flags that step 3 failing has no plan for reversing step 2 (release reservation) and step 1 (cancel order)
- [ ] **Choreography risk at this complexity** — notes that 5-step choreography without an orchestrator is hard to debug; recommends orchestration (Durable Functions or equivalent) for this flow
- [ ] **Step 5 is fragile** — marks that the saga completes when step 5 succeeds, but if step 5's event publish fails, the saga is orphaned in "paid but not confirmed" forever
- [ ] **No max-duration / orphan detection** — flags that there's no plan for sagas stuck mid-flight; recommends a saga state store with timeout monitoring
- [ ] **Ordering assumptions** — points out that events from different services may arrive out of order at observers, and the design seems to assume serial execution which choreography does not guarantee
- [ ] **Specific fix for each problem** — for each flagged problem, names a concrete remediation (outbox table, dedup table keyed on event ID, switch to orchestrator, etc.) rather than just describing the issue

## Reference output

> This design has every classic saga failure mode in one place. Code-review blockers, in order:
>
> **1. No transactional outbox.** "Publish after commit" is the wrong shape. If the broker is unavailable in the window between DB commit and publish, the event is lost: the order exists, but inventory never receives the reservation event, and the saga halts silently. Fix: each service writes the event into an `outbox` table in the same DB transaction as the domain change; a relay process publishes from outbox to Service Bus and marks rows as sent. Atomic, no lost events.
>
> **2. No idempotency.** Service Bus is at-least-once. Without dedup, a redelivered `InventoryReserveRequested` causes a second reservation; a redelivered `PaymentChargeRequested` causes a second charge. Customers will notice the second one. Fix: each consumer maintains `processed_events(event_id)`; before processing, INSERT-on-conflict-skip and bail if the row already exists; otherwise process and INSERT in the same transaction.
>
> **3. Email is in the wrong place.** "Send confirmation email" cannot be compensated. If step 5 fails after step 4 succeeded, you cannot un-send the email. Notification must be either (a) outside the saga entirely — fire-and-forget after the saga commits, or (b) a "best-effort" event that is not part of the consistency boundary. Fix: emit `OrderConfirmed` after step 5; notification-service subscribes independently; it is not a saga participant.
>
> **4. No compensation defined.** The design lists only the happy path. What happens if payment-service rejects the card? Inventory has a reservation, order is pending, and nothing tells anyone. Fix: every saga step has a named compensation. `OrderCreated` → `OrderCancelled`; `InventoryReserved` → `ReservationReleased`; `PaymentAuthorized` → `PaymentVoided`. The orchestrator (see next point) triggers compensations in reverse order on failure.
>
> **5. Choreography is wrong for this complexity.** Five steps across four services with no orchestrator is debugging hell. Where is "the saga"? Nowhere — it's emergent from event flows. When a customer complains, no one can answer "where is my order stuck?" Fix: introduce an orchestrator. Azure Durable Functions is the obvious choice: it persists saga state, handles retries and compensations as activity functions, and gives you a queryable status. Five steps is squarely in orchestration territory; choreography becomes preferable above ~8 services.
>
> **6. Step 5 is a single point of saga failure.** The saga completes when `OrderConfirmed` is published. If that publish fails (broker hiccup, service restart between DB write and publish), the order shows "paid" in the DB but never reaches "confirmed". Saga is orphaned. Fix: outbox solves the publish problem (#1), and the orchestrator (#5) waits for `OrderConfirmed` with a timeout and retries.
>
> **7. No orphan detection.** What if step 2 succeeds and then inventory-service crashes between publishing `InventoryReserved` and the order receiving it? Without a saga state store with max-duration enforcement, the saga is stuck forever. Fix: orchestrator persists saga state with a `started_at`; a periodic job alerts on sagas exceeding, say, 5 minutes; runbook for stuck sagas.
>
> **8. Ordering not addressed.** Service Bus subscriptions don't guarantee global ordering. The design implicitly assumes `InventoryReserved` precedes `PaymentAuthorized` at every observer. With sessions/partitioning, you can enforce per-aggregate ordering; without, observers must be tolerant. Fix: either route events through Service Bus sessions keyed on `orderId` for per-order ordering, or design consumers to handle out-of-order arrivals.
>
> **9. Summary of fixes**
> - Add transactional outbox to each saga participant.
> - Add dedup table keyed on event ID at each consumer.
> - Move notification out of the saga.
> - Introduce an orchestrator (Durable Functions); document compensation per step.
> - Add saga state store + max-duration alert.
> - Route saga events with `sessionId = orderId` for ordering.
>
> Don't approve this design until at least the outbox, idempotency, orchestrator, and compensation pieces land. The other items can come in the next iteration.

## Notes

This prompt is designed to look superficially reasonable so that an LLM without the saga pattern context will rubber-stamp it. Catches whether the LLM is actually applying the pattern card's failure-mode guidance.
