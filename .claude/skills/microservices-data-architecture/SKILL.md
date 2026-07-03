---
name: microservices-data-architecture
description: Designs data consistency and state management patterns when microservices interact across boundaries. Covers transactional outbox, saga (orchestration and choreography), CQRS, event sourcing, idempotent consumers, and database-per-service. Use when deciding between saga and 2PC, choosing CQRS vs. a unified model, designing an outbox-driven event flow, evaluating event sourcing fit, or resolving a cross-service consistency question. Do not use for broad architecture design (use microservices-architecture-design) or for sync-vs-async messaging choice (use microservices-async-messaging).
version: 1.0.0
last_updated: 2026-05-18
---

# Microservices Data Architecture

## When to use

Trigger this skill when the question is about cross-service consistency, state management across service boundaries, or specific data patterns: saga, transactional outbox, CQRS, event sourcing, database-per-service, idempotent consumer. Common triggers: "should I use saga or 2PC," "do I need event sourcing for this," "how do I reliably publish an event after a DB write," "my read load is killing the write side."

Do **not** use this skill for: the broader question of whether to use microservices (`microservices-architecture-design`); the choice between sync REST and async messaging (`microservices-async-messaging`); resilience patterns like circuit breaker or retry (`microservices-resilience`).

## The critical decision rule — eventual consistency is the default

For cross-service business transactions, **eventual consistency is the default**. Strong consistency across services requires distributed transactions, which require 2PC, which kills availability. The patterns in this skill exist because the alternative to eventual consistency at scale is "don't have microservices."

If a business requirement names strong consistency across services as non-negotiable (regulatory ledger, banking core), the answer is usually "those services are actually one bounded context — keep them in one service." See `microservices-architecture-design`.

## The pattern selector

| Problem | Pattern | Card |
|---|---|---|
| Publish an event reliably after a local DB write | Transactional outbox | `references/patterns/transactional-outbox.md` |
| Coordinate a multi-step transaction across services | Saga (orchestration or choreography) | `references/patterns/saga.md` |
| Read and write workloads have different shapes or scale profiles | CQRS | `references/patterns/cqrs.md` |
| Need to reconstruct historical state or audit every change | Event sourcing | `references/patterns/event-sourcing.md` |
| Messages may be delivered more than once | Idempotent consumer | `references/patterns/idempotent-consumer.md` |
| Service must own its data exclusively | Database per service | `references/patterns/database-per-service.md` |

For full pattern detail, decision criteria, and Azure implementations, read the card. For the conceptual sequence of how these patterns compose, see `references/data-architecture.md`.

## Pattern selection logic

1. **Always-on:** every state-changing operation across services uses **idempotent consumer** semantics. Messages will be redelivered. Code accordingly. This is not optional.

2. **Always-on:** every service owns its own data store (**database per service**). No shared mutable database. Read replicas across services are not "shared databases" — they are owned reads.

3. **If publishing an event after a state change:** use **transactional outbox** unless the published event truly does not matter (rare). The naive sequence "UPDATE then PUBLISH" loses events on crash between steps; outbox makes the publish part of the transaction.

4. **If a business transaction spans services and can take seconds to settle:** use **saga**. Choose orchestration (one coordinator drives the flow — Azure Durable Functions or an explicit state machine in code) for clarity at the cost of central coupling. Choose choreography (services react to each other's events via Service Bus topics) for decoupling at the cost of harder debugging.

5. **If read load is starving writes, or queries need shapes the write model doesn't carry:** use **CQRS**. Read model populated via outbox + consumer into Cosmos DB or denormalized SQL. Do not dual-write.

6. **If you need to reconstruct historical state, audit every change, or replay business decisions:** use **event sourcing**. This is a heavy commitment — event schema becomes a long-term contract. Don't reach for it unless the audit or replay requirement is real and named.

## Worked example — brownfield: adding CQRS to an existing Spring Boot order service

Setup: existing Spring Boot order service on AKS, single Postgres backend. Read load is saturating the primary; ad-hoc reporting queries take over 3 seconds. Write throughput is also degrading because of read-side index contention.

Decision walk:

1. **Confirm CQRS is the right fix, not just read replicas.** Read replicas with eventual consistency may suffice for queries that match the write schema. CQRS is correct only if read and write *shapes* differ — reports need aggregations or joins the write model doesn't carry. Confirmed: the reports need a denormalized customer-order summary the order model doesn't expose efficiently.
2. **Choose the read store.** Cosmos DB (denormalized JSON per query shape) for low-latency read; alternative is a separate Postgres schema with materialized views. Cosmos wins here because the query shapes will multiply over time and a JSON store handles that better. See `references/patterns/cqrs.md`.
3. **Choose the propagation mechanism.** Transactional outbox in Postgres → Azure Function consumer → Cosmos read model. Use Service Bus topic for fan-out if other services need the same events. Do **not** dual-write from the order service. See `references/patterns/transactional-outbox.md`.
4. **Handle idempotency.** Consumer must be idempotent — the same outbox row will be delivered more than once during retries. Use the outbox row ID as the dedup key in Cosmos. See `references/patterns/idempotent-consumer.md`.
5. **Migration shape.** Deploy outbox table + consumer first. Run dual-read (both Postgres and Cosmos) behind a feature flag, validate parity for 2-4 weeks, cut reads over per query, retire the Postgres read path in a follow-up release.
6. **Observability.** Emit `cqrs_lag_seconds` metric (outbox row timestamp → Cosmos commit timestamp) to Prometheus. Alert if p99 > 30s sustained for 5 minutes. See `azure-microservices-observability` skill.

## Anti-pattern — dual-write to read model and write model

**Bad:** The application code, after handling a business transaction, writes to both the write store (Postgres) and the read store (Cosmos) directly. "We'll just keep them in sync from the application."

**Why it fails:** Dual-write is not transactional. If the Postgres write succeeds and the Cosmos write fails (or vice versa), the two stores diverge silently. Over time, the read model drifts from the write model. The "fix" is usually a nightly reconciliation job, which papers over the design defect at the cost of stale data, complexity, and on-call pages.

**Detection signal:** application code that contains two consecutive write calls to different data stores, with no transactional boundary between them. Often visible as `orderRepo.save(...)` followed immediately by `cosmosClient.upsert(...)`.

**Fix:** Single source of truth (the write store) + outbox + async consumer that updates the read store. The read store derives from the write store via an asynchronous event stream. No application code writes directly to both.

## Verification questions

1. Is every cross-service state change either local-only or wrapped in saga / outbox?
2. Are all consumers idempotent — can a redelivered message be processed twice with the same result?
3. Does the application code write to exactly one data store per transaction, with downstream derivations driven by events?
4. For CQRS: is there a measured staleness budget for the read model (e.g. "p99 lag under 30 seconds"), and is it monitored?
5. For saga: is there a documented compensation path for every step that can fail?
6. For event sourcing: is the event schema versioned, and is there a documented upgrade path for breaking changes?

## What to read next

- `references/data-architecture.md` — the conceptual integration of these patterns
- `references/patterns/saga.md` — orchestration vs. choreography, Azure Durable Functions implementation
- `references/patterns/transactional-outbox.md` — outbox table schema, Azure Functions worker pattern
- `references/patterns/cqrs.md` — read model design, projection patterns
- `references/patterns/event-sourcing.md` — event store choices, snapshotting, replay
- `references/patterns/idempotent-consumer.md` — dedup strategies, message-id vs. business-id keys
- `references/patterns/database-per-service.md` — data ownership rules, cross-service query strategies
- `microservices-async-messaging` skill — for the broker layer that carries these events
- `azure-service-mapping` skill — Service Bus vs. Event Grid vs. Event Hubs decision
