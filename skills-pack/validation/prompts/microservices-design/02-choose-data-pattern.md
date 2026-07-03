---
id: microservices-design/02-choose-data-pattern
area: microservices-design
exercises:
  - .claude/skills/microservices-data-architecture/references/data-architecture.md
  - .claude/skills/microservices-data-architecture/references/patterns/saga.md
  - .claude/skills/microservices-data-architecture/references/patterns/transactional-outbox.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Choose a data pattern for cross-service consistency

## Context

Attach `05-data-architecture.md` and the saga / transactional-outbox / event-sourcing pattern cards.

## Prompt

We have three services: order-service, inventory-service, and payment-service. When a customer places an order, we must (a) create the order, (b) reserve inventory, (c) authorize payment, and (d) on any failure, roll back. We don't have distributed transactions. What pattern combination do you recommend, and what's the second-best alternative if you can't have the first?

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Primary recommendation: saga + transactional outbox + idempotent consumers** — names all three as the working combination (one alone isn't enough)
- [ ] **Orchestration vs. choreography decision** — picks one and justifies; mentions Durable Functions / workflow engine for orchestration if going that route, or per-service event subscriptions for choreography
- [ ] **Compensation per step** — for each step, names the compensation (cancel order, release reservation, refund authorization)
- [ ] **Idempotency keys** — explains how duplicate event delivery is handled (idempotency key per saga step, dedup table or broker-level)
- [ ] **Outbox motivation** — explains *why* outbox is needed (atomic DB-write + event publish) and what breaks without it
- [ ] **Eventual consistency surfaced in UX** — calls out that the customer sees "processing" state during saga execution, not a synchronous "success"
- [ ] **Second-best alternative** — names event sourcing (with the ledger driving projections) or, if rejecting that, explains why event sourcing is heavier than needed
- [ ] **Trade-offs named** — at least one explicit trade-off (latency vs. consistency, operational complexity vs. correctness, schema discipline cost)

## Reference output

> **Primary recommendation: saga (orchestration) + transactional outbox + idempotent consumers.**
>
> One alone is not enough. The saga gives you the rollback (compensation) story; the outbox gives you reliable event publication atomic with the DB write; idempotent consumers give you safety under the broker's at-least-once delivery. Skip any one and a real incident will surface the missing piece.
>
> **Orchestration over choreography.** With three services and a clear linear flow (order → reserve → authorize → confirm), an orchestrator (Azure Durable Functions or a small in-house state machine) is easier to reason about and debug than fan-out events. Choreography is better when 5+ services react to the same events; here, the workflow is bounded and the order matters.
>
> **Steps and compensations**
>
> | Step | Local transaction | Compensation |
> |---|---|---|
> | 1. Create order | order-service: INSERT order(pending) + outbox event | cancel order (mark cancelled) |
> | 2. Reserve inventory | inventory-service: INSERT reservation + decrement available + outbox event | release reservation, restore available |
> | 3. Authorize payment | payment-service: INSERT auth + outbox event | void authorization |
> | 4. Confirm order | order-service: UPDATE order(confirmed) + outbox event | (terminal; no compensation needed) |
>
> If step 3 fails: orchestrator triggers compensation for steps 2 then 1, in reverse.
>
> **Outbox motivation.** Without it, the producer must commit to its DB *and* publish to the broker as two operations. If the broker is down between commit and publish, the event is lost — the order exists but inventory never gets the reservation event. With outbox: the event is written to an `outbox` table in the same DB transaction, and a relay process publishes from there. Atomic; never lost.
>
> **Idempotency.** Each consumer (inventory, payment) maintains a `processed_messages(message_id)` table. Before doing the local transaction, check if the saga's step-ID has already been processed. The check + the side-effect + the dedup-row INSERT all run in one DB transaction. Result: a retry-driven duplicate delivery is a no-op.
>
> **Surfacing eventual consistency.** The customer-facing API returns a 202 Accepted with an order ID and a "processing" status; the UI polls or subscribes to updates. Returning "order placed!" the instant the orchestrator starts is a lie that will bite during the first incident.
>
> **Second-best alternative: event sourcing for the order aggregate.** If we wanted full audit and the ability to replay state, we'd source the order as a sequence of events (OrderCreated, InventoryReserved, PaymentAuthorized, OrderConfirmed). The orchestrator becomes a process-manager reacting to the event log; projections feed the read APIs. This gives stronger audit and replay but adds significant complexity: event schema evolution, snapshotting, projector lag. For three services and a linear flow, it's overkill; we'd reach for it only if compliance demanded the audit trail or we expected multiple read projections.
>
> **Trade-offs to call out**
> - Eventual consistency is real: the moment after the customer clicks "place order", the inventory levels in the catalog UI may not yet reflect the reservation. Acceptable; surface it.
> - Saga adds operational surface: dashboards for in-flight sagas, alerts for sagas exceeding max duration, runbook for stuck sagas.
> - Outbox adds DB load: every state-change writes two rows (domain + outbox) and the relay adds polling load. Usually negligible; quantify if you're at the edge.

## Notes

The "second best" framing forces the responder to engage with the trade-off, not just recite the primary answer. Catches LLMs that respond with "saga" without explaining why each accompanying pattern is needed.
