---
name: microservices-api-design
description: Designs and reviews HTTP and gRPC API contracts for Azure-hosted microservices, including REST resource modeling, gRPC service definitions, versioning, error semantics, pagination, idempotency, API gateway placement (Azure API Management), and Backend-for-Frontend patterns. Use when designing a new API surface, reviewing an OpenAPI spec, deciding REST vs. gRPC, planning an API version bump, choosing whether to front a service with API Management, or designing a BFF for a specific client. Do not use for code-level handler review (use pr-review-azure-microservices once it exists) or for messaging contracts (use microservices-async-messaging).
version: 1.0.0
last_updated: 2026-05-18
---

# Microservices API Design

## When to use

Trigger this skill when the question is about the *contract* of an API: REST resource shape, gRPC service definition, versioning policy, error model, pagination, idempotency on writes, gateway placement, or whether to introduce a Backend-for-Frontend layer. Common triggers: "should this be REST or gRPC," "how do I version this without breaking clients," "where should validation live — gateway or service," "should mobile and web hit the same API."

Do **not** use this skill for: code-level handler review (`pr-review-azure-microservices` once it exists); messaging contracts and event schemas (`microservices-async-messaging`); auth and authz flows (`azure-microservices-security`); rate limiting and throttling configuration (`azure-service-mapping` for APIM specifics).

## The critical decision rule — design for the consumer, version for incompatibility

The API contract is the consumer's product, not the service's. Design field shapes, error responses, and resource hierarchies for the easiest consumer integration, not for the easiest internal implementation.

For versioning: introduce a new major version only when the change is **breaking** (removed fields, changed types, changed semantics). Additive changes (new optional fields, new endpoints) ship on the existing version. Use URI versioning (`/v1/orders`) for major versions; never expose internal version numbers (build numbers, commit SHAs) in the API path.

## The API-design selector

| Concern | Default | Reference |
|---|---|---|
| External / customer-facing API | **REST + OpenAPI** | `references/api-design.md` |
| Internal service-to-service, latency-sensitive, typed contracts | **gRPC** | `references/api-design.md` |
| Need rate limiting, auth, throttling, transformation at the edge | **Azure API Management gateway** | `references/patterns/api-gateway.md` |
| Multiple clients (mobile, web, partner) with different shapes | **Backend-for-Frontend** | `references/patterns/backend-for-frontend.md` |
| Bulk read / large response | **Cursor-based pagination** | `references/api-design.md` |
| Retryable write | **Idempotency-Key header** | `references/api-design.md` |

## API-design logic

1. **REST vs. gRPC:**
   - REST + OpenAPI for external APIs, browser-facing endpoints, and any contract crossing organizational boundaries. The schema language (OpenAPI 3.x) is universal; client tooling exists in every stack.
   - gRPC for internal service-to-service where latency matters, contracts are strongly typed, and both ends are owned by the same team or org. Protobuf gives compact wire + codegen + streaming.
   - **Do not** expose gRPC directly to browsers — use gRPC-Web through API Management, or wrap with a REST adapter at the gateway. Browsers cannot speak HTTP/2 trailers reliably enough.

2. **Resource modeling:**
   - Nouns, not verbs. `POST /orders` to create, not `POST /createOrder`.
   - Avoid leaking implementation: no `/api/v1/services/order-svc/db/orders`. The consumer doesn't care which service or DB holds the data.
   - Keep nesting to 2 levels max. `/orders/{id}/items` is fine; `/customers/{id}/orders/{id}/items/{id}/refunds/{id}` is not. Flatten with query parameters or filter endpoints.

3. **Versioning:**
   - Major version in URI: `/v1/orders`, `/v2/orders` when breaking changes ship. Old version stays alive for a documented deprecation window (90 days minimum for external; 30 days for internal).
   - Never use header-based versioning as the *only* signaling. It's invisible in URLs, hard to test from a browser, and frequently bypassed.
   - Document the deprecation in the OpenAPI spec (`deprecated: true`) and emit a deprecation log event on each call.

4. **Error model:**
   - Use HTTP status codes correctly: 400 for client validation, 401 for missing/invalid auth, 403 for authorized but forbidden, 404 for resource not found, 409 for conflict, 422 for semantic validation, 500 for server failure, 503 for downstream unavailable.
   - Body shape: `{"code": "RESOURCE_NOT_FOUND", "message": "...", "details": {...}, "trace_id": "..."}`. Stable `code` field is the contract; `message` is human-readable.

5. **Idempotency on writes:** require an `Idempotency-Key` header on every POST/PATCH/DELETE that the client might retry. Service stores the (key, response) tuple for 24h; replays return the cached response without re-executing the side effect.

6. **API Management:** front the service with APIM when you need (a) external auth (OAuth 2.1 with Entra ID), (b) per-consumer rate limiting, (c) request/response transformation, (d) developer-portal documentation, or (e) versioning at the edge. If none of those apply, skip APIM — it adds latency and operational surface.

7. **Backend-for-Frontend:** introduce a BFF when web and mobile clients need materially different response shapes (e.g., mobile needs trimmed payloads to save bandwidth; web wants pre-joined data for rendering). The BFF is a thin aggregation layer that calls 2-5 internal services and produces the shape the client wants.

## Worked example — brownfield: adding cursor pagination to an existing list endpoint

Setup: existing Spring Boot order service exposes `GET /v1/orders` that returns *all* orders for the authenticated customer in one response. As accounts accumulate history, mobile clients are timing out and the service is OOMing on heavy users. Need to add pagination without breaking existing clients.

Decision walk:

1. **Pick the pagination shape.** Cursor-based, not offset-based. Reasons: orders are inserted continuously; offset pagination skips records when new orders insert during pagination. Cursor on a stable sort key (`created_at` + `id` as tiebreaker) avoids skipping. See `references/api-design.md`.
2. **Backward compatibility.** Current `GET /v1/orders` returns `{"orders": [...]}`. Existing clients will not break if we add `limit` and `cursor` query params (default `limit=20`) and add `next_cursor` to the response. Old clients ignore the new fields. *Do not* change the existing field shape.
3. **Default limit + max limit.** Default `limit=20`. Hard cap at `limit=100`; requests with `limit > 100` are clamped to 100 (return a `X-Pagination-Capped: true` header). Without a max limit, a misbehaving client can request `limit=1000000` and OOM the service again.
4. **Cursor opacity.** Cursor is base64-encoded `{"created_at": "...", "id": "..."}`. Opaque to clients — they pass it back unmodified. Lets us change the cursor implementation without breaking clients.
5. **Update OpenAPI spec.** Add `limit`, `cursor` query params; add `next_cursor` field to response. Keep old `orders` field; do not bump version. Re-publish the OpenAPI spec.
6. **Deprecation strategy.** Once pagination is adopted, deprecate the "return everything" semantics by capping the page in a future version. For now, the new params are opt-in.
7. **Test on a large account.** Synthesize an account with 50,000 orders; verify paging is stable, response size is bounded, and total fetch time is acceptable. See `azure-microservices-observability` for the metrics to add.

## Anti-pattern — leaking implementation through the URL

**Bad:** `POST /api/v1/services/order-service/internal/db/orders` for order creation. The URL encodes the service name, the internal/external split, and the persistence layer.

**Why it fails:**
- The consumer is now coupled to the internal naming. When you split the service or move the data, every consumer URL breaks.
- It signals to consumers that internal details are part of the contract; they will write code that depends on those details.
- It's almost always longer and uglier than the equivalent clean URL.

**Detection signal:** URLs that include `service`, `internal`, `db`, `repo`, build-team names, or company-internal acronyms. Also: URLs that reveal the persistence shape (`/api/orders/sql/...`).

**Fix:** `POST /v1/orders`. The consumer doesn't need to know which service handles it. Internal routing (which service, which DB) lives behind the gateway.

## Verification questions

1. Does the URL contain any implementation details (service name, internal, db, repo)? If yes, redesign.
2. Are HTTP status codes correctly mapped to error categories (400 vs. 422 vs. 409 vs. 500)?
3. Does every retryable write (POST, PATCH, DELETE) accept an `Idempotency-Key` and honor it?
4. Is pagination cursor-based, with a default and a max `limit`?
5. For versioning: is there a documented deprecation window for the previous version, and is the deprecation visible in the OpenAPI spec?
6. Is APIM justified by at least one specific need (auth, rate limit, transformation, dev portal), or is it cargo-culted?

## What to read next

- `references/api-design.md` — REST and gRPC contract details, OpenAPI patterns, error model
- `references/patterns/api-gateway.md` — APIM placement, routing rules, policy framework
- `references/patterns/backend-for-frontend.md` — when and how to introduce a BFF; aggregation patterns
- `microservices-async-messaging` skill — for the event side of the API contract (events are also a contract)
- `azure-microservices-security` skill — for the OAuth 2.1 + Entra ID auth flow that sits in front of the API
- `azure-service-mapping` skill — for the APIM tier decision and its cost/throughput trade-offs
