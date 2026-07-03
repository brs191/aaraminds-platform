---
id: architecture-review/02-event-sourcing-fit
area: architecture-review
exercises:
  - .claude/skills/microservices-data-architecture/references/patterns/event-sourcing.md
  - .claude/skills/microservices-data-architecture/references/patterns/cqrs.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Is event sourcing the right fit here?

## Context

Attach the event-sourcing and CQRS pattern cards.

## Prompt

Two systems are asking whether to adopt event sourcing. For each, decide yes/no with justification:

**System A:** A retail-banking ledger. Every account balance change must be auditable for at least seven years; regulators may demand "what was the balance at 14:32 on 2024-03-15?". The team has experience with Kafka and event-driven systems.

**System B:** A CRM contact-management service for a small SaaS. Users edit contact records (name, email, tags). Most data is current-state; some teams want a "who changed this?" feature later.

Be opinionated.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **System A: YES, with reasoning** — recommends event sourcing for the ledger and cites the natural fit (every change is an event; replay gives historic balance; audit and regulator queries become free)
- [ ] **System B: NO, with reasoning** — recommends against full event sourcing for the CRM; the cost dominates the benefit
- [ ] **System B alternative named** — suggests transactional outbox + audit table, or temporal tables, or simple change-log approach instead of full event sourcing
- [ ] **Snapshot requirement for A** — for the ledger, calls out that snapshots will be needed for performance once accounts have years of history
- [ ] **Schema evolution discipline** — mentions that event sourcing requires versioning discipline; old events live forever, so schema changes need upcasting
- [ ] **CQRS pairing for A** — recommends a CQRS read model (statement projections) feeding customer-facing queries; the event store is not queried directly for "show me last month's statement"
- [ ] **Concurrency / contention** — for the ledger, addresses how concurrent writes to the same account are serialized (optimistic concurrency on aggregate version, single-writer per account, or explicit locks)
- [ ] **Avoids "use event sourcing because audit"** trap — explicitly states that needing audit is *necessary* but not sufficient for event sourcing; cheaper audit patterns exist
- [ ] **Team-readiness factor** — names that System A's team has experience with event-driven systems (matters); System B's team's experience is not described and might tip the answer

## Reference output

> **System A (retail-banking ledger): yes, event sourcing is the right fit.**
>
> Every balance change is naturally an event: a deposit, a withdrawal, a transfer leg, a fee. The current balance is a fold over the event stream. Audit is free — the event log *is* the audit trail. The regulator's question "what was the balance at 14:32 on 2024-03-15?" becomes "replay events up to that timestamp, return the resulting balance." A traditional model would need a parallel `account_balance_history` table that drifts from the live one and has the same audit-trail problem one indirection deeper.
>
> What you need to do well:
> - **Snapshots.** Replaying every event back to account opening becomes slow once accounts have years of history. Snapshot every N events (e.g., every 100 transactions) and load latest-snapshot + tail-events.
> - **Schema discipline.** Events live forever. Adding a field is fine if it's optional; removing or renaming is forbidden. Upcasters convert old event versions to the current shape on load. Build a schema registry into the team's process.
> - **Concurrency.** Per-account writes must serialize — you cannot have two concurrent transfers both reading balance X and writing X-Y. Optimistic concurrency on aggregate version (`expected_version` in the append) is the standard solution; conflicts retry. For very hot accounts, a single-writer-per-account dispatcher.
> - **CQRS pairing.** Don't query the event store for "show me last month's statement." Project events into a `statements-read-model` (Cosmos or Postgres) and serve queries from there. Projector lag is acceptable; surface "as-of" timestamps.
>
> The team's Kafka/event-driven background is a tailwind. Without it, the learning curve alone might push the choice the other way.
>
> **System B (CRM contact management): no, event sourcing is overkill.**
>
> The shape is wrong. Most operations update current state (correct an email, add a tag); the historical view is a maybe-later nice-to-have, not a business requirement. Adopting event sourcing for this is paying the full operational tax (event store, projections, schema discipline, snapshots, retraining the team) for a benefit that's worth a fraction of that cost.
>
> Cheaper patterns that solve the *actual* need ("who changed this?"):
> - **Audit table.** Each contact write also writes an `audit_log` row: `(contact_id, field, old_value, new_value, changed_by, changed_at)`. Sufficient for the maybe-later feature without changing the core model. Cost: one trigger or one repository method; no schema discipline; no learning curve.
> - **Transactional outbox + event log table.** Slightly more structured: every write emits a change event into a local table that other services can subscribe to or that powers the audit feature. Path to event sourcing later if the requirement evolves, without committing now.
> - **Temporal tables** (system-versioned tables in SQL Server / Postgres). The DB keeps the history transparently; queries can use `FOR SYSTEM_TIME AS OF` for point-in-time reads. Less code in the application, although more DB-specific.
>
> The fallacy to avoid: "We might need audit later, so let's event-source now." The audit need can be met by far cheaper patterns; event sourcing is a big architectural commitment justified only when the domain is genuinely event-shaped *and* the audit/replay/projections value is high. For a small-team CRM, it's not.
>
> **The general rule (worth stating)**
> Needing audit is *necessary* for choosing event sourcing but not sufficient. The decision is: is the domain genuinely event-shaped (state changes drive workflow), and do you want multiple read views derived from the same history? If yes to both, event sourcing pays. If audit is the only driver, an audit table or temporal tables are usually cheaper and just as effective.

## Notes

The dual-case format forces the responder to make a "no" recommendation, which is harder than a yes. Catches the "use this pattern because audit" bait that the event-sourcing card explicitly warns against.
