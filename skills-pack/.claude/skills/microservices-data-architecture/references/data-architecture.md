# Skill — Microservices Data Architecture

## Purpose

Design data consistency and state management patterns when services interact across boundaries. This skill covers CQRS, saga, transactional outbox, and event sourcing — patterns that solve the hard problem of maintaining consistency in a distributed system. Use this when service boundaries are defined and you need to decide how services share and synchronize state.

## When You Need Data Architecture Patterns

**You don't need them if:**
- Every service-to-service interaction is a simple query (no state change)
- All state changes are within a single service (no distributed transaction)
- Eventual consistency is not acceptable and you want to keep using 2-phase commit (acceptable for <10ms latency, <100 RPS)

**You need them if:**
- A business transaction spans multiple services
- Services must react to each other's state changes
- You want to avoid distributed transactions
- You need to audit or reconstruct state (event sourcing)

## Pattern 1 — Transactional Outbox

**Problem:** A service needs to both update its local state and publish an event. If it updates the database and then publishes the event, a crash between the two leaves the event unpublished.

**Solution:** Publish the event to the same database transaction. A separate worker reads the outbox table and publishes to the message broker.

**Implementation:**
```
BEGIN TRANSACTION
  UPDATE Order SET status = 'Paid'
  INSERT INTO outbox (aggregate_id, event_type, payload)
    VALUES (order_id, 'OrderPaid', {...})
COMMIT

[Outbox worker polls periodically]
  SELECT * FROM outbox WHERE published_at IS NULL
  FOR EACH row:
    PUBLISH to Service Bus/Event Hub
    UPDATE outbox SET published_at = NOW()
```

**Azure implementation:**
- Outbox table in Azure SQL or Cosmos DB
- Worker in Azure Container Apps or Azure Functions
- Publish to Service Bus or Event Grid

**When to use:**
- Services need guaranteed, at-least-once event delivery
- You can tolerate eventual consistency (5-60 second delay)

**Trade-offs:**
- Added latency (outbox worker polling adds seconds)
- Added complexity (outbox table, worker process, idempotency handling)
- Solves the consistency problem but not coordination

## Pattern 2 — Saga (Long-Running Transaction)

**Problem:** A business transaction spans multiple services and requires coordination. If one service fails, others must compensate (undo their changes).

**Solution:** Implement the transaction as a workflow with compensation steps. Two models: orchestration and choreography.

**Orchestration model (recommended):**
```
OrderService: creates Order
→ calls PaymentService.ProcessPayment()
  PaymentService: charges card
  → returns success or failure
→ if success: calls InventoryService.Reserve()
  InventoryService: reserves stock
  → returns success or failure
→ if InventoryService fails: calls PaymentService.Refund() [compensation]
→ if all succeed: commits Order
```

**Choreography model (event-driven):**
```
OrderService: creates Order, emits OrderCreated
→ PaymentService receives OrderCreated, charges card
  → emits PaymentSucceeded or PaymentFailed
→ InventoryService receives OrderCreated, reserves stock
  → emits InventoryReserved or InventoryFailed
→ OrderService receives events, coordinates compensation
```

**Azure implementation:**
- Orchestration: Durable Functions or a dedicated orchestrator service
- Choreography: Service Bus or Event Grid + service handlers

**When to use:**
- Multi-step business transactions (order → payment → fulfillment)
- Compensation steps are well-defined
- You can tolerate eventual consistency

**Trade-offs:**
- Complexity: saga definition, compensation logic
- Debugging: distributed transaction is hard to trace
- Testing: requires mocking multiple services

## Pattern 3 — CQRS (Command-Query Responsibility Segregation)

**Problem:** Read and write workloads have incompatible scaling requirements. Reporting queries are slow because they scan millions of orders; concurrent writes stall.

**Solution:** Separate the write model (optimized for transactions) from the read model (optimized for queries). Maintain read model via events or replication.

**Implementation:**
```
Write model (Transactional):
  Order service: single-threaded writes, strong consistency
  Database: Azure SQL with row-level locking

Read model (Optimized for querying):
  OrderReadService: denormalized view optimized for reports
  Database: Cosmos DB with projections
  Sync: Order service publishes OrderCreated, OrderPaid events
       → Worker projects events into read model
```

**Query example:**
```
Write model: ORDER table with status, line items
Read model: ORDER_REPORT table with aggregated cost, fulfillment status, ...
  (can be searched easily, no joins required)
```

**Azure implementation:**
- Write model: Azure SQL (transactional, normalized)
- Read model: Cosmos DB or Azure Search (denormalized, queryable)
- Sync: Service Bus, Event Grid, or Durable Functions

**When to use:**
- Read-heavy workloads (reporting, dashboards)
- Writes are latency-sensitive, reads are not
- You want to scale reads and writes independently

**Trade-offs:**
- Complexity: maintain two models, keep them in sync
- Latency: reads are eventually consistent (data lag of seconds to minutes)
- Cost: double the storage for two models

## Pattern 4 — Event Sourcing

**Problem:** You need to audit all state changes or reconstruct state as of a past point in time. Traditional databases overwrite old data.

**Solution:** Store only events (immutable facts), derive current state by replaying events.

**Implementation:**
```
Events table:
  OrderCreated { orderId, customerId, items }
  ItemAddedToOrder { orderId, itemId, quantity }
  PaymentAuthorized { orderId, amount }
  OrderConfirmed { orderId }

Current state = replay all events for orderId, apply in order

State as of T = replay events up to time T
```

**Azure implementation:**
- Event store: Event Hubs, Cosmos DB, Azure SQL
- Snapshots: cache state every N events (replay 1000 events is slower than replaying 100 + snapshot)
- Projections: read models derived from events

**When to use:**
- Auditability is non-negotiable (financial transactions, medical records)
- You need temporal queries (what was the state on 2026-01-15?)
- You need to debug by replaying a sequence of events

**Trade-offs:**
- Complexity: high. Event schema evolution is difficult, replaying is slow
- Storage: events can grow large; need cleanup/archival strategy
- Debugging: "what's the current state?" requires replaying all events

## Decision Framework — Which Pattern to Use

| Scenario | Pattern | Reason |
|---|---|---|
| One service updates, others read | No pattern needed | Direct query or cache |
| Multi-service transaction, eventual OK | Saga | Proven, well-understood |
| Single service must publish event reliably | Transactional Outbox | Guaranteed at-least-once delivery |
| Reads and writes have different scale | CQRS | Separate optimization |
| Must audit all state changes | Event Sourcing | Immutable record |
| Multi-service + audit + scale | CQRS + Event Sourcing | Complex but necessary for some domains |

## Worked Example — Order with Payment and Fulfillment

**Scenario:** Place order (Order service), charge payment (Payment service), reserve inventory and fulfill (Fulfillment service).

**Choice:** Saga (orchestration) + Transactional Outbox

**Implementation:**
```
Order service:
  BEGIN TRANSACTION
    INSERT Order { status: Created }
    INSERT outbox { event: OrderCreated }
  COMMIT
  → Outbox worker publishes OrderCreated

Payment service:
  ON OrderCreated:
    Attempt payment
    IF success:
      INSERT Payment { status: Authorized }
      INSERT outbox { event: PaymentAuthorized }
      COMMIT
      → Outbox worker publishes PaymentAuthorized
    IF failure:
      INSERT Payment { status: Failed }
      COMMIT
      → Order service receives PaymentFailed, cancels order

Fulfillment service:
  ON PaymentAuthorized:
    Reserve inventory
    CREATE FulfillmentOrder
    (If inventory unavailable, emit InventoryUnavailable)

Order service:
  ON InventoryUnavailable:
    Cancel order, refund payment [Saga compensation]
```

## Verification Questions

1. **Consistency scope:** What data must be strongly consistent? Is it within one service?

2. **Event delivery:** If you publish an event, must it be received? (At-least-once via outbox, or best-effort?)

3. **Compensation:** If a step fails, can you reverse it? Is reversal automatic or manual?

4. **Read consistency:** For queries, is eventual consistency acceptable? How stale can data be?

5. **Event sourcing:** Do you need to audit state changes or reconstruct history? If not, don't use event sourcing.

## What to read next

- For resilience patterns that work with saga/outbox: `../../microservices-resilience/references/resilience-patterns.md`
- For patterns in detail: `patterns/saga.md`, `patterns/transactional-outbox.md`, `patterns/cqrs.md`, `patterns/event-sourcing.md`
- For Azure implementation specifics: `../../azure-service-mapping/references/azure-mapping.md`
