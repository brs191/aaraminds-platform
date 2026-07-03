---
id: microservices-design/01-decompose-monolith
area: microservices-design
exercises:
  - .claude/skills/microservices-architecture-design/references/domain-decomposition.md
  - .claude/skills/microservices-architecture-design/references/service-boundaries.md
pass_threshold: 7/10
last_run: 2026-05-30
last_result: pass
---

# Decompose a checkout monolith

## Context

Attach `03-domain-decomposition.md`, `04-service-boundaries.md`. The responder is advising a team that has decided to break apart a checkout monolith.

## Prompt

We run a single "checkout" service that handles: cart management, pricing and promotions, tax calculation, payment authorization, order capture, fulfillment routing, and customer notifications. It's a single Go service backed by one PostgreSQL database. The team wants to break it into microservices. Propose the decomposition: which services, what each owns, where the boundaries are, and what's likely to go wrong.

## Rubric

A response passes if it covers at least 7 of the following 10 points:

- [ ] **Bounded contexts named** — proposes services aligned to bounded contexts (e.g., cart, pricing, tax, payment, order, fulfillment, notification) rather than CRUD-level splits
- [ ] **Each service owns its data** — explicitly assigns data ownership per service (no shared tables), naming the data each service owns
- [ ] **Identifies cross-context coupling** — flags places where the old schema co-mingled data (e.g., orders rows referencing promotion details directly) and proposes how the new boundaries resolve them
- [ ] **No distributed transaction** — explicitly rejects 2PC across the new services and proposes saga / events for cross-service workflow
- [ ] **Synchronous vs. asynchronous classified** — for each cross-service interaction, says whether it's a synchronous call or an event, and justifies the choice
- [ ] **Migration approach** — suggests strangler-fig or another incremental strategy, not a big-bang cutover
- [ ] **Boundary risks named with detection signal** — at least two named risks (chatty dependency, distributed monolith, shared DB sneak-in, hot aggregate) with how you'd notice them
- [ ] **Team ownership called out** — each service has a single owning team; flags any service that ends up with ambiguous ownership
- [ ] **Mentions ADR or design canvas** — recommends capturing the decomposition in an ADR or the service-boundary canvas before coding starts
- [ ] **Avoids over-decomposition** — does not propose 15+ tiny services for what can fit in 6–8; or, if it does, defends the proliferation with a concrete reason

## Reference output

> **Proposed services**
>
> | Service | Owned data | Why it's its own service |
> |---|---|---|
> | cart-service | carts, cart_items | High write, stateful, customer-session-scoped; lifecycle different from orders |
> | pricing-service | price_lists, promotions, discount_rules | Independent change cadence (marketing-driven) and rule complexity |
> | tax-service | tax_rates, jurisdiction_rules | Regulatory updates; isolation lets compliance team move independently |
> | payment-service | payment_intents, payment_methods | PCI scope confined here; isolation reduces audit surface |
> | order-service | orders, order_line_items, order_status | Core domain aggregate; orchestrates the workflow |
> | fulfillment-orchestrator | fulfillment_jobs, dispatch_state | Long-running workflow, distinct SLO from synchronous APIs |
> | notification-service | notification_templates, send_log | Async fan-out, different scaling profile, no contention with hot paths |
>
> Seven services, not fifteen. The temptation to split "order capture" from "order status query" or "cart create" from "cart update" should be resisted — they share the aggregate and the team. Split only when there's a real coupling problem.
>
> **Data ownership rule**
> Each row exists in exactly one service's database. No FK across DBs; references are by ID. Cross-service queries either call the owner's API or maintain a local projection (CQRS read model) where the API call is too chatty.
>
> **Cross-service interactions**
>
> - cart → pricing: synchronous call during cart updates (user is waiting; result is needed immediately)
> - cart → tax: synchronous call at checkout time only
> - order created → fulfillment: **event**; fulfillment is async and decoupled
> - order created → notification: **event**; fire-and-forget
> - payment authorized → order: **event**; order moves to paid status on receipt
> - any state-changing cross-service workflow: **saga** (order → reserve inventory → authorize payment → confirm), with compensation per step
>
> No 2PC. Order workflow is a choreographed saga: each step is a local transaction + outbox event. Compensation is explicit per step (refund payment, release inventory).
>
> **Migration approach**
> Strangler-fig. Put a routing facade (API gateway or simple reverse proxy) in front of the monolith. Extract one bounded context at a time, starting with the least entangled: notification-service first (it's already async-ish and has no inbound dependencies). Then tax-service (read-mostly, regulatory-driven, naturally isolated). Then payment (PCI motivates the split anyway). Cart and pricing later — they're hottest. Order last because it orchestrates everything else.
>
> Each extraction: build the new service, dual-write or shadow read during transition, switch the route, monitor, decommission the code in the monolith.
>
> **Risks to watch**
> - **Distributed monolith via chatty calls.** Cart → pricing → tax → discount → loyalty becomes a 100-ms chain per cart update. Detection: P95 latency on cart update climbs; trace shows >5 hops. Prevention: cache aggressively in the cart, pre-compute when feasible, consider a CQRS read view for hot lookups.
> - **Shared DB sneak-in.** Under deadline pressure, one team adds a foreign key from their new service to another's table for "just this one query." Detection: schema review or network policy audit reveals cross-service DB connections. Prevention: network policies that allow only the owner to connect to its DB; enforce in CI.
> - **Saga without compensation.** "We'll add compensation later." Detection: incidents where a partial workflow leaves inconsistent state. Prevention: ADR per saga lists every step's compensation explicitly before the saga ships.
> - **Hot aggregate.** Order becomes the bottleneck because every event touches it. Detection: order-service is the loudest service in incidents. Prevention: consider event sourcing for order if append rate is dominant, or split the aggregate (order-capture vs. order-fulfillment).
>
> **Team ownership**
> Cart, pricing → commerce-team. Tax → compliance-team. Payment → payments-team (audited separately). Order, fulfillment, notification → platform-team. Service-boundary canvas captures this; ADRs gate any change to ownership.
>
> **Document before coding**
> One ADR per service describing capability, owned data, public API, events emitted, events consumed, owning team. One overall ADR for the migration strategy and the order of extraction. These exist before the first line of new service code is written.

## Notes

This is the most demanding prompt in the pack — it asks for both decomposition design and risk awareness in a single response. Catches LLMs that produce a generic "split it by entity" answer instead of an opinionated decomposition with named risks.
