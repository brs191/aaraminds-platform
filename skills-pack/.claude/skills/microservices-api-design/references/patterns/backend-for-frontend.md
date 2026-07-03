# Pattern: Backend for Frontend

## Problem

A general-purpose API can't optimize for every client. Mobile apps need slim payloads on slow networks; web apps need richer data with fewer round trips; smart TVs need yet another shape. Forcing one API to serve all leads to over-fetching, chatty calls, or feature flags everywhere. Backend for Frontend (BFF) gives each client type its own backend tailored to its needs, sitting on top of the shared core services.

## Use When

- Multiple client types (mobile, web, watch, partner) have meaningfully different needs
- A single API spec is becoming bloated with client-conditional logic
- Specific clients have constraints (low bandwidth, limited compute) general APIs can't optimize for
- Different clients have different release cycles and the API must move with them

## Avoid When

- Only one client type exists — a general API is enough
- The team can't staff a backend per client (BFFs need ownership)
- Differences between clients are trivial (just a few fields) — handle with query params
- BFF risks becoming a "smart pipe" with business logic that should live in services

## Azure Implementation

### Implementation Steps

1. Identify distinct client types with materially different needs (mobile vs. web vs. partner)
2. Create a BFF per client type, owned by the team that owns the client
3. BFFs call core domain services; they orchestrate, aggregate, and shape responses
4. Keep business logic in domain services; BFFs do composition and presentation-shaping only
5. BFFs may use GraphQL or REST — whichever fits the client team's preferences
6. Deploy BFFs as Container Apps services; route external traffic via API gateway to the correct BFF

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| BFF hosting | Container Apps | One service per BFF, auto-scaling per client demand |
| API Gateway | APIM or Front Door | Routes mobile traffic to mobile-BFF, etc. |
| Auth | Microsoft Entra ID / External ID | Token validation happens at the gateway or BFF |
| Domain services | Existing microservices | BFFs call them like any other consumer |
| GraphQL option | Hot Chocolate (.NET) or gqlgen (Go) | If client team prefers GraphQL |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Client optimization | Strong — each client gets ideal payload shape |
| Backend duplication | Each BFF reimplements composition and auth glue |
| Team alignment | Client teams own their BFF, can evolve quickly |
| Operational overhead | More services to deploy, monitor, and scale |
| Risk of bloat | BFFs can grow business logic that should live in services |
| Latency | One extra hop (BFF → domain services), 5–20ms typical |

## Common Failure Modes

- **BFF as smart pipe** — Business rules and validation move into the BFF; domain services become CRUD.
  - Detection: BFF has hundreds of LoC of business logic; same logic duplicated in another BFF.
  - Prevention: BFFs orchestrate and transform; rules live in domain services.

- **BFF reuse across clients** — One BFF added "for now" for web, then mobile adopts it, then partner — back to one bloated API.
  - Detection: Original BFF accumulates conditional logic per client.
  - Prevention: Strict policy: one BFF per client type; split when reuse appears.

- **BFF as gateway** — BFF starts doing API gateway concerns (auth, rate limit) inconsistently.
  - Detection: Each BFF reimplements auth, rate limit.
  - Prevention: API gateway handles cross-cutting; BFFs focus on composition.

- **Chatty BFF** — BFF makes 10+ sequential calls to domain services per request, becoming the bottleneck.
  - Detection: BFF latency dominated by sequential downstream calls.
  - Prevention: Parallelize calls; cache aggressively; consider domain-level aggregation endpoints.

## Decision Signals

Use BFF when:
- A mobile app team is fighting the general API for slim payloads
- Different clients have different release cycles needing API flexibility
- API spec has many client-conditional branches

Skip when:
- Single client type — general API is enough
- All clients have similar needs

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Container Apps | BFF hosting | Per-client BFF, independent scaling |
| API Management | Routes to correct BFF | Path/header-based routing |
| Azure Functions | Lightweight BFF | If BFF is mostly aggregation, serverless fits |
| Application Insights | Per-BFF observability | Track each client's API health independently |

## Go Implementation Notes

Mobile BFF returns slim payloads; web BFF returns richer ones. Structure:
```
bff-mobile/
  internal/handlers/      // HTTP endpoints
  internal/composers/     // Fan out to domain services
  internal/dtos/          // Mobile-shaped responses
bff-web/
  ...                     // Same structure, web-shaped
```
Use `errgroup` for parallel calls to domain services. Cache assembled responses where appropriate.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests BFF when describing multiple client types with different needs
- `detect_architecture_risks` — flags BFFs with business logic, duplicate BFFs, missing per-client ownership
- `generate_bff_skeleton` — drafts BFF service skeleton for a described client
- `generate_architecture_decision_record` — drafts ADR for BFF vs. general API

## Related Patterns

- **API Gateway** — handles cross-cutting; BFFs handle composition
- **Aggregator** — BFF often aggregates multiple service calls
- **Cache-Aside** — BFFs cache assembled responses
- **GraphQL** — common BFF implementation choice for flexible queries

## References

- Skill: `../api-design.md` — BFF as an API layering decision
- Pattern: `api-gateway.md` — gateway sits in front of BFFs
- Pattern: `../../../azure-service-mapping/references/patterns/cache-aside.md` — BFFs cache aggregated responses
