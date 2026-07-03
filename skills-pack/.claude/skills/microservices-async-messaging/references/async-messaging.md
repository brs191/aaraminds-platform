# Skill — Asynchronous Messaging and Event-Driven Architecture

## Purpose

Design asynchronous, event-driven communication between services. This skill covers when to use async messaging instead of synchronous calls, how to ensure reliable delivery, and how to structure event-driven workflows. Use this when services need to react to each other's state changes without tight coupling.

## When to Choose Async Over Sync

### Use Synchronous (REST, gRPC)

**When:**
- Caller needs an immediate response (user waiting for a page)
- Caller needs to know if the operation succeeded (validate input, reserve inventory)
- Latency is critical (<100ms)
- Failure is catastrophic and must be handled immediately

**Examples:**
- GET /products (query, no side effects)
- POST /cart/items (validate and add to cart immediately)
- GET /order/{id} (check order status)

### Use Asynchronous (Messages, Events)

**When:**
- Caller doesn't need immediate response
- Multiple services need to react to the same event
- Failure can be retried later (eventual consistency acceptable)
- Decoupling is valuable (services don't know about each other)

**Examples:**
- OrderPlaced event → triggers Payment, Fulfillment, Notification services
- CustomerProfileUpdated event → triggers downstream cache invalidation
- ProcessingComplete message → enqueued for async worker

## Pattern 1 — Publish-Subscribe (Pub-Sub) Events

**Architecture:**
- Service publishes an event to a broker (Service Bus Topic, Event Grid)
- Multiple subscribers receive the event independently
- No explicit coordination; each subscriber processes independently

**Example — e-commerce order:**
```
Order service publishes: OrderPlaced { orderId, customerId, items }
  ↓
Payment service subscribes: charges card, publishes PaymentProcessed
Fulfillment service subscribes: creates FulfillmentOrder
Notification service subscribes: sends order confirmation email
Inventory service subscribes: decrements stock
```

**Azure implementation:**
- Service Bus Topics: durable, supports complex routing
- Event Grid: low-latency, built for Azure events
- Event Hubs: high-volume, supports replay

**Advantages:**
- Decoupled: Order service doesn't know about Payment or Inventory
- Scalable: new subscribers can be added without changing Order service
- Resilient: if Fulfillment is slow, other services aren't affected

**Challenges:**
- Debugging: who subscribed? in what order?
- Event ordering: events may arrive out of order across zones
- Exactly-once delivery: hard to guarantee (idempotency keys help)

## Pattern 2 — Request-Reply (Messaging)

**Architecture:**
- Service A sends a message to a queue, waits for a reply
- Service B reads the message, processes, sends reply back
- Similar to synchronous but asynchronous implementation

**Example:**
```
Order service sends: InventoryRequest { orderId, items }
  → Inventory service reads
  → Inventory service sends: InventoryResponse { available: true/false }
Order service receives response and proceeds
```

**Azure implementation:**
- Service Bus Queues: with correlation IDs to match requests and replies
- Direct point-to-point messaging

**When to use:**
- You need a response but can tolerate delay (seconds, not milliseconds)
- Caller needs the result before proceeding

**Challenges:**
- Correlation: matching requests to replies (use correlation IDs)
- Timeout: how long to wait for a response?
- Cleanup: orphaned replies if requester crashes

## Pattern 3 — Choreography (Event-Driven Orchestration)

**Architecture:**
- Services react to events from other services
- No central coordinator; services know which events to listen for
- Workflow emerges from event chain reactions

**Example — order fulfillment choreography:**
```
Order service: emits OrderPlaced
  ↓
Payment service: receives OrderPlaced
  → charges card
  → emits PaymentAuthorized (or PaymentFailed)
    ↓
Fulfillment service: receives PaymentAuthorized
  → creates FulfillmentOrder
  → reserves inventory
  → emits FulfillmentStarted
    ↓
Inventory service: receives FulfillmentStarted
  → decrements stock
  → emits InventoryDecremented
    ↓
Notification service: receives events, sends emails
```

**Advantages:**
- Decoupled: no single orchestrator that knows all steps
- Scalable: new steps can be added (new subscribers)
- Resilient: failure of one subscriber doesn't block others

**Challenges:**
- Hard to understand: workflow is implicit in event chain
- Debugging: "why didn't X happen?" requires tracing event flow
- Compensation: if a step fails, manually coordinate compensations

## Pattern 4 — Orchestration (Explicit Coordinator)

**Architecture:**
- Central orchestrator (Durable Functions, dedicated orchestrator service) owns the workflow
- Orchestrator calls services in sequence, handles compensations
- Workflow logic is explicit and auditable

**Example — order orchestration:**
```
OrderOrchestrator:
  1. Call PaymentService.charge()
  2. If success:
       Call InventoryService.reserve()
  3. If InventoryService fails:
       Call PaymentService.refund() [compensation]
  4. If all succeed:
       Call NotificationService.sendConfirmation()
```

**Azure implementation:**
- Durable Functions: orchestrator functions + activity functions
- Dedicated orchestrator service: custom logic, more control

**Advantages:**
- Explicit: workflow is readable and testable
- Debuggable: centralized logging, state machine visible
- Compensation: built-in support for retries and rollback

**Challenges:**
- Coupling: orchestrator knows about all services
- Scalability: orchestrator becomes a bottleneck
- State: orchestrator maintains workflow state (must be durable)

## Pattern 5 — Message Ordering and Idempotency

**Problem 1 — Out-of-order delivery:**
Messages arrive in a different order than sent. Order service publishes:
1. OrderCreated
2. OrderPaid
3. OrderShipped

Fulfillment service receives: OrderPaid, OrderCreated, OrderShipped (wrong order)

**Solution:**
- Use partitioned topics: messages with the same orderId go to the same partition (guarantees order)
- Use version numbers: include sequence numbers in events
- Idempotent processing: processing the same event twice should be safe

**Problem 2 — Duplicate delivery:**
A message is delivered twice (network retry, broker restart, consumer restart).

**Solution — Idempotency key:**
```
Message includes: { requestId: UUID, eventType: OrderPlaced, ... }

Consumer:
  Check if requestId exists in idempotency store
  If yes: return cached result
  If no: process message, store (requestId → result)
```

**Azure implementation:**
- Service Bus: supports sessions (ordered delivery within partition)
- Idempotency store: Cosmos DB, Redis, or database with unique constraint

## Decision Framework — Async Pattern Selection

| Scenario | Pattern | Trade-off |
|---|---|---|
| One service reacts to an event | Pub-Sub | Simple, decoupled |
| Multiple services react | Pub-Sub | Implicit workflow |
| Workflow is complex, needs compensation | Orchestration | Explicit, coupled |
| Services are unknown at design time | Pub-Sub (choreography) | Harder to debug |
| Low-latency, high-throughput | Event Hubs (pub-sub) | Less semantics |
| Exactly-once, ordered delivery needed | Partitioned queues + idempotency | Complex implementation |

## Worked Example — Order with Payment, Inventory, Notification

**Choice:** Pub-Sub with choreography + transactional outbox

**Implementation:**
```
Order service:
  1. Create Order in database
  2. Insert into outbox: OrderPlaced event
  3. Outbox worker publishes to Service Bus Topic

Subscribers:
  - Payment service:
      Read OrderPlaced
      Charge card
      Publish PaymentAuthorized (or PaymentFailed)
  
  - Inventory service:
      Read OrderPlaced
      Reserve stock
      Publish InventoryReserved (or InventoryUnavailable)
  
  - Notification service:
      Read PaymentAuthorized
      Send confirmation email
      Read PaymentFailed
      Send payment-failed email

Order service (consumer):
  Read PaymentFailed or InventoryUnavailable
  Cancel order, emit OrderCancelled

Compensation (optional):
  If InventoryUnavailable after PaymentAuthorized:
    Refund service reads both events
    Issues refund and notifies customer
```

## Verification Questions

1. **Decoupling:** Does the sender know about the receiver? (Pub-sub: no; orchestration: yes)

2. **Message loss:** If a message is lost, is there a recovery path? (DLQ, replay, audit trail)

3. **Ordering:** Do messages need to arrive in order? If yes, how is it enforced?

4. **Idempotency:** If a message is processed twice, is the result the same?

5. **Debugging:** Can you trace an event through all subscribers and see what happened?

6. **Compensations:** If a step fails, what reverses the previous changes?

## What to read next

- For patterns in detail: `patterns/event-driven-architecture.md`, `async-messaging.md`, `../../microservices-data-architecture/references/patterns/saga.md`
- For resilience with messaging: `../../microservices-resilience/references/resilience-patterns.md`
- For Azure service mapping: `../../azure-service-mapping/references/azure-mapping.md`
