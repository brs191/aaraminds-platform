# Skill — Microservices API Design

## Purpose

Design stable, versioned APIs that services expose to callers. This skill covers REST and gRPC contracts, versioning strategies, and the gateway pattern. Use this when defining the external contract for a service — what callers depend on.

## API Design Principles

### 1. Intent Over Implementation

**Bad API (leaky implementation):**
```
GET /orders/{id}/customer/address/street
GET /orders/{id}/customer/address/city
GET /orders/{id}/customer/address/zipcode
```
This exposes internal object structure. Callers navigate your domain model.

**Good API (intent-based):**
```
GET /orders/{id}
{
  "id": "order-123",
  "status": "paid",
  "shipTo": {
    "name": "Alice",
    "address": "123 Main St, Springfield, 12345"
  }
}
```
Caller gets the order. Internals can change without breaking the API.

### 2. Versioning Strategy

**Three approaches:**

**A. URL versioning (easy to understand, ugly URLs)**
```
GET /v1/orders/{id}
GET /v2/orders/{id}
```
Each version is a separate implementation. Old versions co-exist.

**B. Header versioning (clean URLs, less discoverable)**
```
GET /orders/{id} with header: Accept: application/vnd.company.order+json;version=2
```
Callers must know about versions. Cleaner URLs.

**C. Deprecation + default (evolve in place)**
```
GET /orders/{id}
Always returns latest schema.
Clients must accept additive changes (new optional fields).
Breaking changes are deprecated, not versioned.
```

**Recommendation:** Deprecation + default. Version only when absolutely required.

**Breaking changes (require new version):**
- Removing a field
- Changing field type (string → int)
- Changing response structure (object → array)
- Changing semantic meaning (status field now means something else)

**Non-breaking changes (no version bump):**
- Adding a new optional field
- Adding a new endpoint
- Adding new enum values
- Marking a field as deprecated (but still returning it)

### 3. Error Responses

**Standard HTTP status codes:**
- 200 OK — success
- 201 Created — resource created
- 204 No Content — success, no body
- 400 Bad Request — client error (invalid input)
- 401 Unauthorized — missing or invalid auth
- 403 Forbidden — authenticated but not authorized
- 404 Not Found — resource doesn't exist
- 409 Conflict — request violates state (e.g., order already paid)
- 429 Too Many Requests — rate limited
- 500 Internal Server Error — server error
- 503 Service Unavailable — server is down or overloaded

**Error response body (consistent across all errors):**
```json
{
  "error": {
    "code": "INVENTORY_UNAVAILABLE",
    "message": "Requested quantity exceeds available stock",
    "details": {
      "requested": 100,
      "available": 47
    }
  }
}
```

**Rules:**
- Every error has a code (machine-readable, stable across API versions)
- Every error has a message (human-readable, may change)
- Errors include relevant context (what went wrong, what was the constraint)
- 4xx errors are client's fault (retry won't help); 5xx errors are server's fault (retry might help)

### 4. Request Validation

**Validate at the boundary:**
- Is the JSON schema correct? (framework does this)
- Are required fields present?
- Are field values in the allowed range/format?

**Example validation logic:**
```
POST /orders
{
  "items": [ { "productId": "...", "quantity": 5 } ],
  "shippingAddress": { ... }
}

Validation:
  ✓ items is an array
  ✓ items[0].productId is a non-empty string (UUID format)
  ✓ items[0].quantity is an integer > 0
  ✓ shippingAddress is an object with required fields
```

**Fail fast on validation errors:** Return 400 with all validation errors at once, not one at a time.

### 5. Pagination

**For list endpoints (GET /orders, GET /customers), implement pagination:**

```
GET /orders?pageSize=20&pageToken=<cursor>
{
  "orders": [ ... ],
  "pageToken": "<next-cursor>",
  "hasMore": true
}
```

**Rules:**
- Default page size: 20–50 items
- Max page size: 100–1000 items (prevent abuse)
- Use cursor-based pagination (pageToken) for stability (offset-based breaks if data changes)

### 6. Rate Limiting

**Implement rate limiting at the gateway:**

```
Headers:
  X-RateLimit-Limit: 1000
  X-RateLimit-Remaining: 987
  X-RateLimit-Reset: 1234567890 (Unix timestamp)

If limit exceeded:
  Status: 429 Too Many Requests
  Retry-After: 60 (seconds)
```

**Rate limit by:**
- Per user (authenticated requests)
- Per IP (unauthenticated requests)
- Per API key (for service-to-service)

## REST vs. gRPC

### REST (JSON over HTTPS)

**When to use:**
- Public APIs (external clients)
- Human-readable debugging needed
- Lightweight clients (web browsers, mobile)

**Advantages:**
- Widely understood, HTTP is everywhere
- Easy to debug (curl, browser)
- Caching via HTTP semantics

**Disadvantages:**
- Verbose (JSON overhead)
- Weak typing (schema is separate from implementation)
- Chatty (multiple round-trips for related data)

**Example:**
```
GET /orders/order-123
GET /orders/order-123/items
GET /customers/customer-456
(3 requests for order + items + customer)
```

### gRPC (Protocol Buffers over HTTP/2)

**When to use:**
- Internal service-to-service (microservices)
- High-performance, low-latency (mobile, embedded)
- Strongly-typed contracts needed

**Advantages:**
- Compact (binary, ~10x smaller than JSON)
- Fast (HTTP/2 multiplexing, 100x faster than REST for latency)
- Strongly typed (schema is enforced)

**Disadvantages:**
- Binary (harder to debug without tools)
- HTTP/2 required (not all proxies support it)
- Less familiar to web developers

**Example (same request as above):**
```
service OrderService {
  rpc GetOrderWithDetails(GetOrderRequest) returns (OrderDetails)
}

OrderDetails includes order + items + customer (1 request)
```

## API Gateway Pattern

**Problem:** Each service has its own API. Callers must know about all services and call them individually.

**Solution:** API Gateway — single entry point that routes requests to services.

**Benefits:**
- Unified interface: callers see one API, not N services
- Cross-cutting concerns: authentication, rate limiting, logging at one place
- Service evolution: refactor service boundaries without breaking external API

**Example — e-commerce gateway:**
```
External API:
  GET /api/orders/{id}
  POST /api/orders

Gateway routes:
  GET /api/orders/{id} → Order service
  POST /api/orders → Order service (validate) → Payment service (charge) → Inventory (reserve)
```

**Azure implementation:**
- Azure API Management (APIM): full-featured gateway
- Application Gateway: layer 7 routing
- Container Apps Ingress: simple routing

**Gateway responsibilities:**
- Route requests to backend services
- Enforce authentication (OAuth, API keys)
- Rate limiting
- Request/response transformation (e.g., aggregate multiple service responses)
- Caching (cache frequently-accessed data)
- Logging and monitoring

## Worked Example — Order Service API

**REST API:**
```
GET /orders
  List orders for authenticated user
  Query params: pageSize, pageToken, status
  Returns: { orders: [...], pageToken, hasMore }

GET /orders/{orderId}
  Get order details
  Returns: { id, status, items, total, shippingAddress }

POST /orders
  Create new order
  Body: { items: [...], shippingAddress }
  Returns: { id, status, estimatedDelivery }
  Validation: items present, quantities > 0, address valid

PUT /orders/{orderId}/cancel
  Cancel order (idempotent)
  Returns: { id, status: cancelled }
  Errors: 404 if order not found, 409 if already shipped
```

**Versioning strategy:**
- Current version: v1 (no version in URL, added to headers if needed)
- Breaking change (e.g., remove items array): Create v2 only if necessary
- New fields added to response: No version bump (clients ignore unknown fields)

**Error codes:**
- INVALID_REQUEST — missing required fields
- ORDER_NOT_FOUND — order doesn't exist
- ORDER_ALREADY_SHIPPED — can't cancel
- INSUFFICIENT_INVENTORY — items not available
- UNAUTHORIZED — not authenticated

## Verification Questions

1. **Intent-based:** Does the API expose implementation details or business concepts?

2. **Stability:** Can you add fields to responses without breaking clients?

3. **Versioning:** When do you truly need a new version vs. extending the current one?

4. **Errors:** Do errors include enough context to diagnose the problem?

5. **Rate limiting:** Is there a way for well-behaved clients to know they're approaching limits?

6. **Gateway:** Is there a single entry point, or must callers know about multiple services?

## What to read next

- For specific patterns: `patterns/api-gateway.md`, `patterns/backend-for-frontend.md`
- For resilience in APIs: `../../microservices-resilience/references/resilience-patterns.md`
- For Azure service mapping: `../../azure-service-mapping/references/azure-mapping.md`
