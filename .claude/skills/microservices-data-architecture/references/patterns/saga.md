# Pattern: Saga

## Problem

A business transaction spans multiple services, each with its own database. The classic two-phase commit (2PC) doesn't scale across distributed services — it requires holding locks across the network, which destroys availability. Without coordination, partial failures leave the system inconsistent: an order is created, payment is charged, but inventory reservation fails — and the customer is billed for nothing.

## Use When

- A business transaction must update state in 3+ services that each own their own data store
- Eventual consistency is acceptable — the final state can take seconds/minutes to settle
- You can define a clear compensation (reverse operation) for each step that might need to roll back
- The workflow has a finite, well-defined number of steps (5–10, not unbounded)

## Avoid When

- Strong immediate consistency is mandatory (banking ledger entries, regulatory locks)
- The transaction is contained within a single service — use a local DB transaction instead
- Compensation logic is ambiguous or expensive (e.g., "un-send the email" — you can't)
- The number of steps is unbounded or branches dynamically — use a workflow engine instead

## Azure Implementation

### Implementation Steps

1. Model the saga as a state machine: list each step and its compensation
2. Choose the coordination style:
   - **Orchestration** (one coordinator drives the flow) — use Azure Durable Functions
   - **Choreography** (services react to each other's events) — use Service Bus Topics
3. Make every step idempotent — saga retries will replay steps
4. Implement compensation actions for each step (refund payment, restore inventory)
5. Set a max saga duration; archive sagas that exceed it for manual review
6. Wire distributed tracing (correlation ID per saga) through every step

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Orchestrator | Durable Functions | Activity functions per step, automatic state persistence |
| Event-driven choreography | Service Bus Topics + Subscriptions | Per-service subscription, dead-letter for poison messages |
| Saga state store | Azure SQL or Cosmos | Track saga status, current step, retry count |
| Tracing | Application Insights | Correlation ID propagates through HTTP/messaging headers |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Consistency | Eventual — UI must show "processing" states, not "done" until saga completes |
| Complexity | High — every step needs forward + compensation code paths |
| Testability | Must test both happy path and every compensation branch |
| Operational visibility | Requires correlation IDs and dashboards to debug failures |
| Latency | Higher than 2PC due to async hops between steps |

## Common Failure Modes

- **Orphaned saga** — Saga starts but final ack is lost; state hangs in "in-progress" forever.
  - Detection: Sagas exceeding max duration (alert if >5 min for an order saga).
  - Prevention: Timeout each saga; archive expired ones to a review queue.

- **Cascading compensation failure** — Compensation step fails (e.g., refund API down), leaving system inconsistent.
  - Detection: Alert on compensation step error rate >0%.
  - Prevention: Make compensations idempotent and retry indefinitely with DLQ fallback.

- **Non-idempotent step** — Saga replays a step (after crash recovery) and double-charges or double-ships.
  - Detection: Audit log shows duplicate side-effects with same saga ID.
  - Prevention: Every step keys off the saga ID; check "already done" before acting.

- **Choreography pile-up** — Event-driven saga without an orchestrator becomes spaghetti; nobody knows the full flow.
  - Detection: Engineers can't draw the saga on a whiteboard.
  - Prevention: Document the saga flow; consider switching to orchestration if >3 services involved.

## Decision Signals

Use a saga when you see:
- A business action triggers cascading work in 3+ services
- A "rollback" requirement that crosses service boundaries
- Customer-visible state that goes through phases (Pending → Confirmed → Fulfilled)

Don't use a saga when you see:
- A single service that owns all the data the transaction touches
- A regulatory requirement that all-or-nothing happens at one instant

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Durable Functions | Orchestrator | Built-in state durability, replay, timers |
| Service Bus | Event broker | Reliable at-least-once delivery, DLQ, sessions for ordering |
| Application Insights | Observability | End-to-end traces via correlation IDs |
| Cosmos DB | Saga state store | If global multi-region sagas needed |

## Go Implementation Notes

In Go, the orchestration shape typically uses Azure Durable Functions or a workflow library; each saga step is an activity function with its compensation also as an activity. Outbox row writes happen inside the same DB transaction as the step's local commit; a relay process pulls from the outbox and publishes to Service Bus. Consumers dedupe on the saga step ID before performing side-effects. The example MCP server in this pack does not currently include a saga-specific service or tool (`detect_architecture_risks` and `generate_resilience_plan` are the closest related tools); a `generate_compensation_plan` tool could be added by following the existing service-package pattern.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests Saga when input mentions multi-service transactions
- `detect_architecture_risks` — flags missing compensation, non-idempotent steps, unbounded duration
- `generate_compensation_plan` — produces a step→compensation table for a described workflow
- `generate_architecture_decision_record` — drafts the ADR explaining orchestration vs. choreography choice

## Related Patterns

- **Transactional Outbox** — reliable event publication at the end of each saga step
- **Idempotent Consumer** — required for every saga step receiver
- **Circuit Breaker** — protects saga steps that call flaky downstream services
- **Distributed Tracing** — non-negotiable for debugging saga failures

## References

- Skill: `../data-architecture.md` — saga vs. event sourcing decision framework
- Pattern: `event-sourcing.md` — alternative to compensation-based rollback
- Pattern: `transactional-outbox.md` — how saga steps publish events reliably
