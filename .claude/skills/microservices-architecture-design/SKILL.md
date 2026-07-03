---
name: microservices-architecture-design
description: Designs and reviews Azure-hosted microservices architectures end-to-end, covering domain decomposition (DDD), service boundaries, data architecture, resilience, async messaging, API contracts, Azure service mapping, observability, security, and cost trade-offs. Use when designing a new microservices system, reviewing an existing one, deciding whether microservices are right for the problem, producing an Architecture Decision Record, or routing a broad architecture question to the right narrower skill. Do not use for code-level PR review (use pr-review-azure-microservices) or for narrow pattern questions (use the relevant narrower skill).
version: 1.1.0
last_updated: 2026-05-19
---

# Microservices Architecture Design

## When to use

Trigger this skill when the request is about: designing a new microservices system on Azure; reviewing an existing microservices architecture; deciding whether microservices are right for the problem at all; producing an Architecture Decision Record for a distributed-systems choice; or when the user has a broad architecture question and you need to route to a narrower skill.

Do **not** use this skill for: code-level PR review (use `pr-review-azure-microservices`); narrow data-pattern questions like saga vs. CQRS (use `microservices-data-architecture`); narrow resilience questions like circuit breaker configuration (use `microservices-resilience`); MCP server design (use `mcp-go-server-building`).

## The critical first decision — should this be microservices at all?

Never recommend microservices blindly. A monolith or modular monolith is architecturally superior when at least two of these hold:

- Team size is under 20 engineers — coordination overhead outweighs isolation benefit
- The business domain is not yet decomposed — service boundaries will be wrong and require multiple refactors
- Strong consistency is required across most business operations — distributed transactions are expensive and brittle
- Deployment frequency is not an organizational constraint — a monolith deploys fast

If any two are true, lead with "consider a modular monolith first." Record the decision in an ADR. Microservices are an organizational tool first and a technical one second.

## The design sequence

System design is a directed workflow. Skipping stages couples architecture to wrong assumptions and forces rework. Walk these in order:

| # | Stage | Reference |
|---|---|---|
| 1 | Business capability modeling | `references/system-design-process.md` (Stage 1) |
| 2 | Domain decomposition and bounded contexts | `references/domain-decomposition.md` |
| 3 | Service boundaries and data ownership | `references/service-boundaries.md` |
| 4 | Communication pattern choice (sync vs. async) | `microservices-async-messaging` skill |
| 5 | Data architecture (saga, outbox, CQRS, event sourcing) | `microservices-data-architecture` skill |
| 6 | Resilience design | `microservices-resilience` skill |
| 7 | API contracts | `microservices-api-design` skill |
| 8 | Azure service mapping | `azure-service-mapping` skill |
| 9 | Observability design | `azure-microservices-observability` skill |
| 10 | Security and compliance | `azure-microservices-security` skill |
| 11 | Cost and trade-off analysis | `azure-microservices-cost-review` skill |

Stages 1-3 are this skill's territory. Stages 4-11 route to dedicated skills — each is a load-bearing decision class that deserves focused triggering.

## Pattern selection fast path

When the architectural shape is already known and the question is "which patterns fit," use this decision table to route directly to the relevant skill — skip the 11-stage walk.

| Situation | Patterns to consider | Skill |
|---|---|---|
| Read and write workloads differ significantly | CQRS, read models, caching | `../microservices-data-architecture/references/data-architecture.md` |
| Cross-service business transactions | Saga, transactional outbox | `../microservices-data-architecture/references/data-architecture.md`, `../microservices-async-messaging/references/async-messaging.md` |
| Unpredictable load spikes | Autoscaling, queue-based load leveling, bulkhead | `../microservices-resilience/references/resilience-patterns.md` |
| Service-to-service calls can fail | Circuit breaker, timeout, retry with jitter | `../microservices-resilience/references/resilience-patterns.md` |
| Events must be emitted reliably | Transactional outbox, event sourcing | `../microservices-data-architecture/references/data-architecture.md` |
| Services must discover each other | Service discovery, service mesh | `../microservices-api-design/references/api-design.md` |
| Need to roll out changes safely | Blue-green, canary, strangler-fig | `../microservices-resilience/references/resilience-patterns.md` |
| Cross-tenant data isolation is required | Database per tenant, multi-tenant schema | `references/service-boundaries.md` |

## Worked example — brownfield: extracting a payment service from a Spring Boot monolith

Setup: 4-year-old Java Spring Boot monolith handling orders, payments, and fulfillment. Payment processing is becoming a bottleneck during peak hours; the payment team wants to scale independently. Team size is 35 engineers across three squads.

Decision walk:

1. **Is decomposition justified?** Two of the criteria flip *toward* microservices: team size is over 20; payment has a distinct scaling profile. The domain is partially decomposed (payments has clear boundaries already). Strong consistency is not required across payment and other contexts. Proceed.
2. **Bounded context check.** Payment in the monolith handles: card transactions, refunds, fraud screening, ledger entries. Confirm with the payment squad which of these stay together. Result: card-transactions and refunds form one bounded context; fraud screening is increasingly distinct and will likely become its own service later.
3. **Extraction strategy.** Strangler fig pattern (see `../microservices-resilience/references/patterns/strangler-fig.md`). Build the new payment service alongside the monolith, route new requests through it via the API gateway, keep monolith handlers running for in-flight transactions, migrate dependent code paths one at a time.
4. **Data ownership.** Payment service owns its tables; monolith retains read-only access via service API during transition, no direct DB access. See `references/service-boundaries.md` for the data ownership contract.
5. **Resilience.** Payment service is critical — circuit breaker, retries, timeout policies are mandatory before cutover. See `microservices-resilience` skill.
6. **Cutover.** Feature flag the routing. Run dual-mode for 4-6 weeks, validate parity, cut over fully, remove monolith handlers in a follow-up release.

References: `references/system-design-process.md`, `references/service-boundaries.md`, `../microservices-resilience/references/patterns/strangler-fig.md`, `microservices-resilience` skill.

## Anti-pattern — "let's go microservices" before the domain is decomposed

**Bad:** A team with 15 engineers and a 2-year-old monolith proposes splitting it into 12 services because "microservices scale better." There is no DDD analysis. Bounded contexts are guessed at from package names.

**Why it fails:** Two months in, three services share a database schema (the monolith with deployment overhead added), every business transaction crosses 4 service boundaries, and the team is debugging distributed traces for what used to be a stack trace. Velocity drops, not rises.

**Detection signal:** the proposed decomposition is by *technical layer* (UserService, ProductService, OrderService) rather than by *business capability* (Sales, Fulfillment, Accounting). Functional decomposition is a near-perfect indicator the domain analysis was skipped.

**Fix:** Decompose into bounded contexts using DDD first. Then ask if those bounded contexts warrant separate services. Often the answer is "modular monolith for now; extract when one context outgrows the others."

## Verification questions before declaring a design done

1. Can one team own each service end-to-end without coordinating with another for a normal change?
2. Are all cross-service transactions named, with their consistency model and compensation path?
3. Does each service have a timeout, retry, and circuit-breaker policy for every external call?
4. Is every state-changing operation idempotent or guarded by an idempotency key?
5. Can you trace a user request through all services and see per-step latency in Grafana?
6. Is there a named owner, alert path, and runbook for every service?

If any answer is no, the design is not done. Iterate the relevant stage.

## What to read next

- `references/system-design-process.md` — the full 11-stage walkthrough with verification gates at each stage
- `references/domain-decomposition.md` — DDD bounded-context identification
- `references/service-boundaries.md` — data ownership and consistency rules
- `references/patterns/` — 21 pattern cards (saga, CQRS, circuit-breaker, strangler-fig, etc.) referenced from this skill and the narrower skills
- `microservices-data-architecture` skill — for stage 5 (data patterns)
- `azure-service-mapping` skill — for stage 8 (Azure service selection)
- `azure-microservices-cost-review` skill — for stage 11 (cost analysis)
