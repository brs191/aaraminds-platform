# Pattern: API Gateway

## Problem

External clients (browsers, mobile apps, partners) shouldn't know the internal topology of microservices. Without a gateway, each client must discover every backend service, handle authentication and rate-limiting for each, deal with version differences across services, and tolerate any internal restructuring. An API gateway is the single entry point that hides the backend layout and centralizes cross-cutting concerns: auth, rate limiting, routing, transformation.

## Use When

- Multiple external client types (web, mobile, partner APIs) consume the same backend services
- Cross-cutting concerns (auth, rate limiting, logging, transformation) need consistent enforcement
- Internal service topology should not leak to clients
- Multiple services must be exposed under a single domain/API surface

## Avoid When

- Only one service is exposed externally — direct ingress is simpler
- Latency budget is too tight — gateway adds 5–20ms per call
- The gateway becomes a "smart pipe" hosting business logic (anti-pattern: distributed monolith)
- Single-team, single-app setups where governance overhead is wasted

## Azure Implementation

### Implementation Steps

1. Choose the gateway technology by feature need (APIM for governance, App Gateway for L7, Container Apps Ingress for simple)
2. Define routes: `/api/orders/*` → Order service, `/api/payments/*` → Payment service
3. Configure authentication once (OAuth/OIDC validation, JWT inspection) at the gateway
4. Apply rate limiting per consumer (free tier: 100 req/min, paid: 10,000 req/min)
5. Configure caching for cacheable GET endpoints
6. Add request/response transformation where API versions differ
7. Set up the developer portal (if using APIM) for external consumers
8. Monitor gateway latency separately from backend latency

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Full-featured gateway | Azure API Management | Developer portal, policies, rate limit, transforms |
| L7 load balancer | Application Gateway + WAF | Routing, SSL termination, WAF rules |
| Lightweight | Container Apps Ingress | Built-in, basic routing, free |
| Auth | Microsoft Entra ID / External ID | OAuth/OIDC token issuance and validation |
| Cache | APIM built-in or Redis | GET-response caching for hot endpoints |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Encapsulation | Strong — internal topology hidden from clients |
| Cross-cutting concerns | Centralized — auth, rate limit, logging in one place |
| Latency | Adds a hop (5–20ms typical) |
| Single point of failure | Yes — gateway HA is critical |
| Cost | $40–1500/month for APIM tiers; App Gateway cheaper |
| Risk of bloat | Business logic creeping into gateway is a major anti-pattern |

## Common Failure Modes

- **Gateway as smart pipe** — Business logic, data transformations, orchestration accumulate in the gateway.
  - Detection: Gateway has its own service repo with hundreds of LoC of business code.
  - Prevention: Keep gateway dumb — auth, routing, transforms only. Push logic to services or a BFF.

- **Hot route congestion** — One backend service is slow; all gateway threads/connections wait on it, blocking unrelated routes.
  - Detection: All gateway endpoints slow during a single backend's incident.
  - Prevention: Apply bulkhead per backend; circuit breaker per route; per-route timeouts.

- **Auth bypass via direct service access** — Backend services accept calls directly, skipping gateway-enforced auth.
  - Detection: Backend services receive calls without authentication context.
  - Prevention: Network policies restrict backend ingress to gateway only; backends still validate tokens.

- **Versioning conflict** — Gateway hardcodes API v1; service evolves to v2; gateway can't route v2 traffic.
  - Detection: New service version unavailable to external clients.
  - Prevention: Version routing rules; gateway policies declare which versions are exposed.

## Decision Signals

Use API gateway when:
- Multiple external client types call multiple backend services
- Auth, rate limit, or logging policies should apply uniformly
- Public API needs a developer portal and managed governance

Skip when:
- One service exposed externally — Container Apps Ingress is enough
- Internal-only mesh — service mesh handles cross-cutting concerns instead

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Azure API Management | Full gateway | Policies, developer portal, governance |
| Application Gateway + WAF | L7 + security | Lower cost, web app focused |
| Front Door | Global gateway | Multi-region, edge presence |
| Container Apps Ingress | Simple gateway | Free, basic, internal use |

## Go Implementation Notes

If hand-rolling a gateway (rare; prefer APIM), structure routes by service:
```go
mux := http.NewServeMux()
mux.Handle("/api/orders/", orderProxy)
mux.Handle("/api/payments/", paymentProxy)
```
Wrap with middleware: auth check → rate limit → log → forward. Use `httputil.ReverseProxy` for the forwarding.

For Azure native, APIM policies are XML and apply to inbound/outbound/error scopes per route.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends gateway when describing multiple external clients or services
- `detect_architecture_risks` — flags gateway with business logic, missing auth, missing rate limits
- `generate_gateway_routing_plan` — produces route table from service catalog
- `map_patterns_to_azure_services` — chooses APIM vs. App Gateway vs. Ingress

## Related Patterns

- **Backend for Frontend** — client-specific gateway on top of the general gateway
- **Service Mesh** — alternative for internal traffic; gateway handles external
- **Circuit Breaker** — applied at gateway per route
- **Rate Limiting** — gateway enforces consumer quotas

## References

- Skill: `../api-design.md` — gateway concerns vs. service concerns
- Skill: `../../../azure-microservices-security/references/security-design.md` — auth enforcement at the gateway
- Pattern: `backend-for-frontend.md` — client-tailored gateway variants
