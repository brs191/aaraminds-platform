# Skill — Service Boundaries and Data Ownership

## Purpose

Establish clear data ownership and consistency rules at service boundaries. This skill ensures that data responsibility is explicit, preventing shared mutable state that causes runtime corruption and coordination overhead. Use this when service boundaries are identified and you need to decide what data belongs where and how services query each other.

## The Data Ownership Principle

Each service must own exactly one copy of data it is responsible for. Other services must not mutate that data. This is the single-source-of-truth (SST) principle.

- **Owned data:** only this service writes, only this service is authoritative
- **Read-only references:** other services may read (via query, cache, or event) but must not write
- **Shared domain data:** reference data, configuration, master data that multiple services reference but no one owns (or owned by one service with distribution)

Violating this causes:
- Consensus problems (which copy is authoritative when data diverges?)
- Coordination overhead (service A wants to write, but service B also owns a replica)
- Cascading failure (replica corruption spreads across services)

## Decision Framework — Owned vs. Shared vs. Reference Data

### Owned Data

**Characteristics:**
- Service controls the lifecycle (create, update, delete)
- Service enforces business rules (only this service knows what makes a valid Order)
- Service encodes the state machine (Order → Placed → Paid → Shipped)
- Other services must not write it (not even in bulk operations)

**Examples:**
- Order service owns Order (order placement, status changes, cancellation)
- Payment service owns Payment (payment processing, refunds, disputes)
- Fulfillment service owns FulfillmentOrder (warehouse task, picking state, shipping)

**How other services access owned data:**
- **Synchronous query:** "GET /orders/{id}" to Order service (caller waits)
- **Async event:** Order emits OrderPlaced event; Fulfillment consumes and creates its own FulfillmentOrder
- **Cache:** Fulfillment caches Order state locally (with TTL) to avoid querying on every decision

### Read-Only References

**Characteristics:**
- Service maintains a foreign key or ID reference, reads the data, but never writes
- Service depends on the owning service to be up (for consistency)
- Updates to the referenced data flow through the owning service

**Examples:**
- Order service references ProductId (references Catalog service's Product)
- Fulfillment service references OrderId (references Order service's Order)

**How it works:**
- Order service has OrderLine { productId, quantity, price } — the price is copied at order time, not a live reference
- Other services can query Order service to learn Order state, but only Order service writes Order

### Shared Domain Data (Master Data / Reference Data)

**Characteristics:**
- Owned by one service (the master)
- Distributed to other services (via cache, events, or periodic sync)
- Infrequently changes
- Read-only in downstream services

**Examples:**
- Product Catalog (master in Catalog service, cached in Cart service and Recommendation service)
- Azure Regions and Zones (master in one service, referenced everywhere)
- Currency Codes (reference data, immutable)

**Distribution mechanism:**
- **Caching:** Catalog service publishes ProductUpdated event; downstream services cache and invalidate on event
- **Periodic sync:** Nightly job pulls latest master data
- **Event stream:** Catalog publishes every product change; Recommendation service maintains a read model

## Consistency Boundaries

For each pair of services, decide the consistency model.

### Transactional Consistency (within one service, one transaction)

**What's consistent:**
- Order and its OrderLines (same transaction)
- Payment and its TransactionDetail (same transaction)

**How it's enforced:**
- Single database transaction, ACID guarantees
- Rollback if any part fails

**Example — order placement:**
```
BEGIN TRANSACTION
  INSERT Order { status: Placed }
  INSERT OrderLine { orderId, productId, quantity, price }
  UPDATE Inventory { reserved += quantity }  // if Inventory is owned by Order service
COMMIT or ROLLBACK
```

### Eventual Consistency (across services)

**What's eventually consistent:**
- Order state in Order service and Fulfillment service's OrderView (they'll match eventually, not immediately)
- Product price in Catalog and Cart service's CachedPrice (Cart reads stale, learns of updates via event)

**How it's enforced:**
- Asynchronous events or eventual replication
- No guarantee of immediate consistency
- Acceptable when the business can tolerate 5-minute or 5-second staleness

**Example — order fulfillment:**
```
Order service: Order marked as Paid
→ emits OrderPaid event
→ Fulfillment service receives event
→ Fulfillment creates FulfillmentOrder
→ Fulfillment may query Order service to verify (double-check)
```

### Data Ownership Decision Matrix

| Scenario | Ownership | Consistency | Pattern | Example |
|---|---|---|---|---|
| Data is modified by only one service | One service owns | Transactional | Direct ownership | Order owns OrderLine |
| Data is read by many, written by one | One service owns | Async event or cache | Event distribution | Catalog owns Product, others cache |
| Data must be consistent within a txn | One service owns | Transactional | Same transaction | Order + Payment in single txn? (no — separate) |
| Data must be consistent across txn boundaries | Multiple services | Eventual | Saga or transactional outbox | Order + Fulfillment (eventual) |
| Data is never modified | Reference | Eventual | Read-only or cache | Region list, currency codes |

## Worked Example — E-Commerce Order Flow

**Scenario:** Place an order with payment and inventory deduction.

**Services and ownership:**
| Service | Owns | Reads | Writes |
|---|---|---|---|
| **Order** | Order, OrderLine | ProductId (refs Catalog), PaymentId (refs Payment) | Order status, OrderLine |
| **Catalog** | Product, Price | — | Product metadata |
| **Inventory** | SKU, Stock, Reservation | — | Stock levels, reservations |
| **Payment** | Payment, Transaction | OrderId (refs Order) | Payment status, refunds |

**Consistency rules:**
- Order and its OrderLines must be transactionally consistent (same service, same transaction)
- Order and Payment are eventually consistent (Payment confirms after order is in DB)
- Order and Inventory are eventually consistent (Inventory is reserved asynchronously)
- Price is transactionally consistent within the order (copied at order time, not live reference)

**Data flow — place order:**
```
1. Order service: validate cart, reserve prices from Catalog (read-only query)
2. Order service: create Order + OrderLines in single transaction
3. Order service: emit OrderCreated event
4. Payment service: receives OrderCreated, initiates payment
5. Inventory service: receives OrderCreated, reserves stock
6. Payment service: emits PaymentConfirmed event
7. Order service: receives PaymentConfirmed, marks Order as Paid
8. Fulfillment service: receives PaymentConfirmed, creates FulfillmentOrder
9. Inventory service: receives FulfillmentOrder, confirms and decrements stock
```

**Verification:**
- Is there a single service that owns Order? ✓ (Order service)
- Can Order be modified without involving other services? ✓ (status updates are local)
- Is there a cascading write (Order writes Inventory)? ✗ (no, Inventory updates itself on event)
- Is there a 2-phase commit across services? ✗ (no, saga pattern with compensation instead)

## Verification Questions

1. **Single owner:** For each piece of data, can you name exactly one service that owns it?

2. **No cascading writes:** Does any service write to another service's data? (If yes, that's a boundary violation.)

3. **Read-only references:** When service A references data owned by service B, is the reference read-only or does service A expect to write?

4. **Consistency scope:** For each transaction, what data must be consistent at commit time? Is it all within one service?

5. **Eventual consistency plan:** For data that spans services, what pattern ensures eventual consistency? (Saga, outbox, event sourcing, or explicit cache invalidation?)

6. **Failure recovery:** If a service crashes mid-transaction, can the system recover unambiguously?

## Anti-Patterns

**Shared mutable state:** Two services both own the same data (e.g., both Inventory and Order service write Stock)
- Problem: Consensus failure, corruption risk
- Fix: One owner, others read or receive events

**Distributed transactions:** Spanning multiple services in a single database transaction
- Problem: Coordination overhead, hard to undo
- Fix: Saga pattern with compensating actions

**Circular dependencies:** Service A reads from B, B reads from A
- Problem: Deadlock risk, cascade failure
- Fix: Break the cycle with eventual consistency or cache

## What to read next

- For communication patterns across boundaries: `../../microservices-async-messaging/references/async-messaging.md`
- For resilience when data ownership spans services: `../../microservices-resilience/references/resilience-patterns.md`
- For patterns that encode these rules: `../../microservices-data-architecture/references/patterns/database-per-service.md`, `../../microservices-data-architecture/references/patterns/saga.md`, `../../microservices-data-architecture/references/patterns/transactional-outbox.md`
