---
id: cross-cutting/02-pattern-card-cross-reference
area: cross-cutting
exercises:
  - .claude/skills/microservices-data-architecture/references/patterns/saga.md
  - .claude/skills/microservices-data-architecture/references/patterns/transactional-outbox.md
  - .claude/skills/microservices-data-architecture/references/patterns/idempotent-consumer.md
  - .claude/skills/microservices-async-messaging/references/patterns/event-driven-architecture.md
pass_threshold: 5/7
last_run: 2026-05-30
last_result: pass
---

# Trace a pattern's related-patterns chain

## Context

Attach the saga, transactional-outbox, idempotent-consumer, and event-driven-architecture pattern cards.

## Prompt

Starting from the saga pattern, walk the "Related Patterns" chain across at least three hops. For each hop, explain why those two patterns are related (what one enables, requires, or complements about the other). Stop when you've covered the saga / outbox / idempotent-consumer / event-driven-architecture cluster.

## Rubric

A response passes if it covers at least 5 of the following 7 points:

- [ ] **Saga → Transactional Outbox** — explains that saga steps need reliable event publication, which outbox provides
- [ ] **Outbox → Idempotent Consumer** — explains that outbox guarantees at-least-once delivery, which forces consumers to be idempotent
- [ ] **Saga → Idempotent Consumer** — explains that saga step receivers must be idempotent because saga steps may retry/replay
- [ ] **Event-Driven Architecture → Async Messaging** — names EDA's underlying transport (the messaging fabric); calls out that EDA is system-shape while async messaging is the mechanism
- [ ] **Why each pair is "related," not interchangeable** — articulates a directional relationship for at least three pairs (one requires/enables/complements the other, not just "same area")
- [ ] **Distinguishes saga from event-driven** — saga is a coordinated workflow with compensations; event-driven is a fan-out shape; they intersect but aren't the same
- [ ] **Surfaces the failure mode each pair prevents** — at least one pair has a "without both, what breaks" example (e.g., outbox without idempotent consumer = duplicates; saga without outbox = lost events)

## Reference output

> **Hop 1: Saga → Transactional Outbox.**
> Every saga step is a local DB transaction followed by an event to coordinate the next step. If the event publish happens *after* the commit but outside the transaction, a broker outage between commit and publish loses the event — the saga halts silently. The transactional outbox makes the event part of the DB transaction (written to an `outbox` table), with a relay publishing asynchronously. Without outbox, sagas are unreliable. With outbox, every committed step is guaranteed to produce its event. Directional: saga *requires* outbox to be reliable.
>
> **Hop 2: Outbox → Idempotent Consumer.**
> The outbox relay guarantees at-least-once delivery: if the relay crashes after publishing but before marking the row as sent, it re-publishes on restart. So consumers see duplicates. Without idempotency, a duplicate `InventoryReserveRequested` reserves inventory twice; a duplicate `PaymentChargeRequested` charges twice. The idempotent consumer pattern (dedup table keyed on event ID; check before processing; insert in the same transaction as the side-effect) absorbs duplicates safely. Directional: outbox's correctness depends on consumers being idempotent.
>
> **Hop 3: Saga → Idempotent Consumer.**
> A different path to the same destination. Saga retries (orchestrator re-invokes a step after a timeout; failed step compensations re-fire) can deliver the same step to a consumer twice even without outbox. Saga steps must be idempotent end-to-end so retries are safe. Directional: saga *requires* idempotent consumers regardless of how events are published.
>
> **Pairwise summary so far:**
> - Saga ↔ Outbox: saga needs reliable event delivery.
> - Outbox ↔ Idempotent Consumer: at-least-once forces deduplication.
> - Saga ↔ Idempotent Consumer: saga retries can deliver duplicates by themselves.
>
> Each pair has a clear failure mode in the absence of the other. The triad is a unit.
>
> **Hop 4: Event-Driven Architecture vs. Saga vs. Async Messaging.**
> Event-driven architecture is a *system shape*: producers publish facts about state changes; multiple consumers subscribe; the producer doesn't know who consumes. Async messaging is the *transport mechanism* that makes that shape possible — the broker (Service Bus, Event Hubs, Event Grid) delivers events.
>
> Saga is a different concern: it's a coordinated multi-step workflow with explicit compensations. Sagas can be implemented choreography-style on top of an event-driven architecture (services react to each other's events), or orchestration-style with a central coordinator (which still uses async messaging to talk to participants). The patterns intersect: a saga running in an event-driven system uses outbox to publish reliably and idempotent consumers to absorb duplicates. But you can have event-driven systems without sagas (pure fan-out: order created → analytics, ML training, audit; no coordinated workflow) and sagas without full EDA (single-purpose orchestrated workflow with no other subscribers).
>
> Directional: EDA's reliability requires outbox + idempotent consumers at the producer and consumer sides; saga running on EDA inherits both requirements.
>
> **Where this lands**
> The cluster `{saga, transactional outbox, idempotent consumer, event-driven architecture, async messaging}` is the working set for any non-trivial Azure microservices design that coordinates state across services. They aren't independent choices; they're a system. Picking one without the others is what produces the failures the pattern cards' Common Failure Modes sections warn about.

## Notes

Tests whether the responder can navigate the cross-reference graph in the pattern cards rather than describing each pattern in isolation. The reference walks four hops; the rubric accepts any solid coverage of the relationships even if the responder structures it differently.
