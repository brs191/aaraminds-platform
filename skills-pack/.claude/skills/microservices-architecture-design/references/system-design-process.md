# Skill — Microservices System Design Process

## Purpose

Provide a structured process for designing a microservices architecture from business capability to deployment. This skill operationalizes the 11-step sequence in the router and provides decision criteria, template outputs, and verification gates at each stage. Use this when you need to design a system end-to-end or validate that all stages of design have been addressed.

## Stage 1 — Business Capability Modeling

**Input:** Business problem statement or user story set

**Process:**
- Identify the core business capability: what is the user/customer trying to accomplish?
- List the flows that implement the capability (e.g., for e-commerce: browse, search, add-to-cart, checkout, return, fulfillment)
- Identify the actors: who uses each flow? (Customer, inventory system, payment processor, fulfillment service)
- Map any compliance or latency constraints that are non-negotiable

**Output:** Capability map with flows, actors, and constraints

**Verification gate:** Does each flow have an identified owner (a future service or external system)?

## Stage 2 — Domain Decomposition and Bounded Contexts

**Input:** Capability map from Stage 1

**Process:**
Use DDD bounded contexts as the decomposition unit, not technical concerns. A bounded context groups:
- Business concepts that belong together (Order, Payment, Inventory coexist in different bounded contexts depending on their owner)
- Consistency requirements (what must be consistent within a business transaction)
- Team ownership (one team owns one bounded context, not three)

**How to recognize bounded contexts:**
- They have a distinct language (ubiquitous language): e.g., "Order" in the Order service has a different lifecycle than "Order" in the Fulfillment service
- They have clear boundaries: data that belongs to a context does not leak into another (foreign keys OK, shared mutable state is not)
- They are independently deployable: changes to the bounded context do not require coordinated deploys with others

**Anti-pattern — functional decomposition:** Separating services by layer (UserService, ProductService, OrderService) instead of by domain. This creates artificial couplings and makes scaling unpredictable.

**Output:** Bounded context map showing relationships and primary responsibility of each context

**Verification gate:** Can a single team own one bounded context? Do the contexts have distinct vocabulary?

## Stage 3 — Service Boundaries and Data Ownership

**Input:** Bounded context map from Stage 2

**Process:**
For each bounded context, define:
- **Service name** — what is this service called in the architecture
- **Primary responsibility** — what business capability does it own
- **Data ownership** — what data belongs exclusively to this service (SSD: single-source-of-truth)
- **Data read-only references** — what data does it read-only from other services
- **Shared domain data** — what remains shared (master data: products, configuration, reference data)
- **Query patterns** — how do other services learn about this service's state (sync query, async event, cache)

**Decision framework — when is data owned vs. shared:**

| Scenario | Model | Example |
|---|---|---|
| Service controls lifecycle and is the SSD | Owned | Order service owns Order lifecycle; Fulfillment service references it, doesn't own it |
| Data changes infrequently, read widely | Shared | Product catalog is owned by Catalog service, other services cache it |
| Consistency must be transactional | Owned in one service | Payment status must be transactional; owned by Payment service |
| Eventual consistency is acceptable | Event-driven read | Fulfillment learns of paid orders via domain event, maintains its own read model |

**Output:** Service boundary document with data ownership matrix

**Verification gate:** For each service, can you name the one team that owns the data? Are there competing SSDs?

## Stage 4 — Communication Patterns

**Input:** Service boundaries and data ownership from Stage 3

**Process:**
Decide the communication style for each interaction. Don't choose yet; identify what interaction patterns emerge from Stage 3:

- **Synchronous request-reply** (REST, gRPC): when the caller needs an immediate, authoritative response
- **Asynchronous events** (pub-sub, message queue): when the caller doesn't need an immediate response or when multiple services need to react
- **Eventual consistency** (caching, eventual sync): when transactional consistency is not required

**Decision matrix:**

| Interaction | Latency requirement | Consistency requirement | Pattern | Example |
|---|---|---|---|---|
| Browse products | <100ms | Read-only, eventual OK | Cache + async sync | Product catalog cached in API gateway |
| Add to cart | <200ms | Transactional | Sync | Cart service query to inventory |
| Place order | Async OK | Transactional within Order | Sync or saga | Order service transaction, then async event |
| Fulfill order | Async OK | Eventual | Async event | Order emits OrderPlaced event; Fulfillment consumes |
| Query order status | <500ms | Eventually consistent | Cache or read model | Fulfillment service maintains read-only Order view |

**Anti-pattern — synchronous orchestration:** Calling 5 services in sequence (Order → Inventory → Payment → Fulfillment → Notification) synchronously. If any service is slow, the user waits for all. Use saga or choreography instead.

**Output:** Communication pattern matrix (service pair → pattern)

**Verification gate:** For each high-latency flow, is there a synchronous bottleneck?

## Stage 5 — Data Consistency Architecture

**Input:** Communication patterns from Stage 4

**Process:**
Decide the data consistency model for interactions that aren't straightforward synchronous calls:

- **Transactional outbox:** service publishes events reliably after its own transaction commits (database pattern)
- **Saga:** long-running transaction across services with compensating actions for rollback
- **CQRS:** separate read and write models for high-scale systems
- **Event sourcing:** store only events, derive state (for auditability or complex reconstructions)

These are not always needed. If every interaction is synchronous request-reply and within one service, you don't need them.

**Output:** Data consistency decision document (scenario → pattern)

**Verification gate:** For each async interaction, is there a clear compensation/rollback path?

## Stage 6 — Resilience Design

**Input:** Communication patterns and identified failure modes

**Process:**
For each service-to-service call, assume it can:
- Timeout (slow response)
- Fail (5xx error)
- Be unavailable (500ms+ latency)
- Return stale data (cached response)
- Duplicate (network retry)

Choose patterns:
- **Timeout:** wrap every external call in a deadline
- **Retry:** with exponential backoff and jitter (for transient failures)
- **Circuit breaker:** stop calling a failing service for a period
- **Bulkhead:** isolate resources by service to prevent cascade
- **Queue-based load leveling:** async queues as shock absorbers
- **Graceful degradation:** what features work if a dependency is slow?

**Output:** Resilience matrix (service pair → timeout, retry policy, circuit breaker, bulkhead)

**Verification gate:** Does every synchronous call have a timeout? Does slow behavior degrade gracefully?

## Stage 7 — API Contract Design

**Input:** Communication patterns from Stage 4 and service boundaries from Stage 3

**Process:**
Define the API contract for each service:
- **Protocol:** REST (JSON over HTTPS), gRPC, or messaging (Service Bus, Event Hub)
- **Versioning strategy:** semantic versioning of the API, with deprecation policy
- **Request/response schemas:** explicit, validated at the boundary
- **Error responses:** distinguish input error, auth error, service error, timeout
- **Rate limiting:** per-client limits to prevent abuse
- **Gateway:** should there be an API gateway that aggregates services?

**Anti-pattern — leaky abstraction:** Exposing internal service boundaries in the API contract. API consumers shouldn't know you have a Cart service; they know you have a Cart resource.

**Output:** API contract documentation (per service, with examples)

**Verification gate:** Can a client call the service without knowing its internal decomposition?

## Stage 8 — Azure Service Mapping

**Input:** Service boundaries (Stage 3), communication patterns (Stage 4), resilience design (Stage 6), API contracts (Stage 7)

**Process:**
For each service, decide:
- **Hosting:** Container Apps (recommended for microservices), App Service, AKS
- **Data store:** Azure SQL, Cosmos DB, PostgreSQL, Redis, Event Hubs
- **Communication:** Service Bus (messaging), Event Grid (events), APIM (API gateway)
- **Observability:** Application Insights, Log Analytics, Key Vault for secrets
- **Networking:** private endpoints, managed identity, firewall rules

Each choice has trade-offs. Enumerate them explicitly (cost, complexity, latency, consistency).

**Output:** Azure topology diagram with service-to-resource mapping

**Verification gate:** Does each service have a clear Azure resource and a documented rationale for that choice?

## Stage 9 — Observability Design

**Input:** Communication patterns (Stage 4), resilience design (Stage 6), Azure services (Stage 8)

**Process:**
Design observability for the failure modes you chose to handle:
- **Distributed tracing:** trace a request through all services (Application Insights, OpenTelemetry)
- **Metrics:** instrument timeouts, retries, circuit breaker trips, error rates
- **Structured logs:** log service boundary crossings, decisions, state changes
- **Alerts:** alert on SLOs (99.9% availability, P95 latency <200ms)

**Output:** Observability plan (metrics, traces, logs, alerts per service)

**Verification gate:** Can you detect the top 5 failure modes from logs/metrics without guessing?

## Stage 10 — Security and Compliance

**Input:** Service boundaries (Stage 3), API contracts (Stage 7), Azure services (Stage 8), observability (Stage 9)

**Process:**
Design defense-in-depth:
- **Identity:** managed identity for service-to-service, OAuth for user authentication
- **Authorization:** per-service, per-tool authorization (not just "authenticated users can do anything")
- **Network segmentation:** private endpoints, NSGs, deny-by-default inbound
- **Data classification:** PII, secrets, reference data — distinct handling
- **Audit:** every state-changing operation logged with who/when/what
- **Compliance:** if regulated (HIPAA, PCI-DSS), encode constraints as guardrails

**Output:** Security architecture (threat model, auth flows, data classification)

**Verification gate:** Can you trace data access from entry point to storage and identify all points of vulnerability?

## Stage 11 — Cost and Trade-off Analysis

**Input:** All prior stages

**Process:**
Enumerate the total cost of operation:
- Hosting (Container Apps, App Service tier, AKS node count)
- Data (storage, egress, API calls)
- Observability (Application Insights, Log Analytics ingestion)
- Managed services (Service Bus, Event Grid, APIM)

Compare to alternative architectures (monolith, fewer services, different cloud). For each trade-off:
- What are you gaining? (autonomy, scale, failure isolation)
- What are you losing? (operational complexity, cost, latency)

**Output:** Cost model (monthly Azure bill estimate) and trade-off decision document

**Verification gate:** Has someone reviewed the cost and agreed it aligns with business value?

## Template outputs to produce

By the end of the design process, you should have produced:
1. Capability map with flows
2. Bounded context map
3. Service boundary document with data ownership matrix
4. Communication pattern matrix
5. Data consistency decisions (saga, outbox, CQRS, event sourcing)
6. Resilience matrix (timeouts, retries, circuit breakers)
7. API contract documentation
8. Azure topology diagram
9. Observability plan
10. Security/compliance architecture
11. Cost model and trade-off analysis

## What to read next

- For detailed domain decomposition: `domain-decomposition.md`
- For service boundary validation: `service-boundaries.md`
- For specific patterns: pattern cards in `patterns/microservices/`
- For Azure service choice framework: `../../azure-service-mapping/references/azure-mapping.md`
