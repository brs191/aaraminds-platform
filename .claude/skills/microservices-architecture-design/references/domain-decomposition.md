# Skill — Domain Decomposition and Bounded Contexts

## Purpose

Identify service-shaped boundaries from a business domain using Domain-Driven Design (DDD) principles. This skill operationalizes the transition from business capability to service boundaries. Use this when you have a capability map and need to decide which parts should become services and which should stay together.

## Why DDD and Bounded Contexts

A service boundary that doesn't respect the business domain becomes a source of complexity:
- Services end up calling each other frequently (tightly coupled)
- Concepts leak across service boundaries and get re-interpreted (Order in Payment service vs. Order in Fulfillment service)
- One team can't work independently without coordinating with others
- Refactoring is expensive because the service boundary is architectural, not domain-shaped

A bounded context is a business-shaped region of the domain where:
- Concepts have a clear, consistent meaning (ubiquitous language)
- Data changes have local impact (not cascading)
- One team can own the lifecycle and rules
- The context is independent from others (low coupling, high cohesion)

## Decision Framework — How to Identify Bounded Contexts

### 1. Look for concept ambiguity

**Symptom:** The same word means different things in different parts of the domain.

**Example — e-commerce domain:**
- "Order" in the Sales context: a customer's request to buy items, with line items and pricing
- "Order" in the Fulfillment context: a shipment instruction, with physical addresses and carrier info
- "Order" in the Accounting context: a revenue transaction, with GL codes and cost allocation

These are three different bounded contexts because "Order" has incompatible meanings. A customer-order becomes a fulfillment-order becomes an accounting-order with transformation steps between them.

**Detection rule:** If "Order" looks like three different entities in a data model, it's three bounded contexts.

### 2. Look for consistency boundaries

**Concept:** What must be consistent within a single business transaction?

**Example — e-commerce:**
- Inventory and Price must be transactionally consistent within the Sales/Order context (a customer sees the price they paid, not a later price)
- Order and Shipping can be eventually consistent (order confirmed, then shipping happens later)
- Payment and Order must be transactionally consistent (order confirms only after payment clears)

**Detection rule:** If transactions frequently cross service boundaries, the boundary is wrong.

### 3. Look for change patterns

**Concept:** When the domain rules change, what else must change together?

**Example — e-commerce:**
- If the return policy changes (90 days vs. 30 days), the Returns context changes, but the Product context doesn't
- If the product pricing model changes (flat price vs. subscription vs. usage-based), the Pricing context changes, but the Inventory context doesn't
- If the fulfillment logic changes (warehouse rules, carrier selection), the Fulfillment context changes, but the Order context doesn't

**Detection rule:** If a change in business rule touches multiple services, the boundary is artificial.

### 4. Look for team ownership

**Concept:** A team should own one bounded context. Context switching between multiple contexts is expensive.

**Antipattern:** "The Auth team owns the UserService, the Product team owns the ProductService, the Order team owns the OrderService." This fragments concepts. Instead: "The Product team owns the Product, Pricing, and Inventory contexts together."

**Detection rule:** If a single team can't make a decision about a concept without coordinating with another team, the boundary is organizational, not domain-shaped.

### 5. Look for scalability drivers

**Concept:** Services scale differently. If two concepts have incompatible scaling requirements, they belong in different contexts.

**Example — e-commerce:**
- Product Catalog: read-heavy, write-once, cacheable, scales for high concurrency
- Inventory: write-heavy (every order decrements), highly contended, needs strong consistency
- These should be separate services even though both relate to "products"

**Detection rule:** If you scale one concept and the other breaks, split the boundary.

## Worked Example — E-Commerce Domain

**Starting capability map:**
- Browse products
- Search products
- Add to cart / manage cart
- Place order
- Pay
- View order status
- Return item
- Fulfill order
- Ship item
- Track shipment

**Decompose into bounded contexts:**

| Context | Responsibilities | Team | Key Entities |
|---|---|---|---|
| **Catalog** | Product information, categorization, search | Product team | Product, Category, Attribute |
| **Pricing** | Price rules, discounts, subscriptions | Pricing team | Price, Discount, Promotion |
| **Inventory** | Stock levels, allocation, reservations | Inventory team | SKU, Stock, Reservation |
| **Cart** | Shopping carts, wish lists | Frontend/Order team | Cart, CartItem |
| **Order** | Customer orders, order lifecycle | Order team | Order, OrderLine, OrderStatus |
| **Payment** | Payment processing, refunds, disputes | Fintech/Order team | Payment, Transaction, Refund |
| **Fulfillment** | Picking, packing, warehouse logic | Logistics team | FulfillmentOrder, Pick, Pack |
| **Shipping** | Carrier selection, tracking, delivery | Logistics team | Shipment, Carrier, TrackingEvent |
| **Returns** | Return processing, refund orchestration | Logistics team | Return, ReturnItem, ReturnStatus |

**Verification:**
- Catalog concepts (Product) are distinct from Order concepts (OrderLine)? ✓
- Inventory reserve-and-decrement is isolated in Inventory context? ✓
- Payment logic is encapsulated in Payment context? ✓
- Can one team own Fulfillment + Shipping + Returns? ✓ (all logistics concerns)
- Are there synchronous bottlenecks (Order → Inventory → Payment in sequence)? If yes, reconsider boundaries.

## Anti-Patterns in Decomposition

**Layered decomposition ("UserService, ProductService, OrderService")**
- Problem: Services organized by data type, not business concept
- Signal: Every order requires calling three services
- Fix: Organize by business capability (Order context that includes order data, pricing lookup, and inventory check)

**Micro-services ("one concept per service")**
- Problem: Unnecessarily fine-grained. If two concepts always change together, they're in the same context
- Signal: Service A never works without calling Service B
- Fix: Merge them or redesign the boundary

**No context boundaries ("the monolith is fine")**
- Problem: The domain grows and code becomes incoherent
- Signal: Git blame shows 5 different teams edited the same file this week
- Fix: Identify boundaries proactively before the monolith becomes unmaintainable

**Wrong scalability boundary**
- Problem: Two contexts with incompatible scaling requirements bundled together
- Signal: You autoscale Service A, but Service B still saturates a database
- Fix: Make the high-scale context independent

## Verification Questions

1. **Ubiquitous language:** Can you describe each bounded context using its own vocabulary, without translating to other contexts' terms?

2. **Team ownership:** Can a single team make a decision about this context without coordinating with another team?

3. **Change isolation:** If you change a rule in this context (e.g., "orders expire after 24 hours"), do other contexts change?

4. **Consistency boundary:** What data must be strongly consistent within this context? Can everything outside be eventually consistent?

5. **Scale isolation:** If this context needs to scale 10x, does the infrastructure stay the same or do other contexts become bottlenecks?

6. **Testing:** Can you test this context in isolation, or does every test require stubs of three other services?

## What to read next

- For detailed service boundary validation: `service-boundaries.md`
- For concrete pattern examples: pattern cards in `patterns/microservices/`
- For the full design process: `system-design-process.md`
