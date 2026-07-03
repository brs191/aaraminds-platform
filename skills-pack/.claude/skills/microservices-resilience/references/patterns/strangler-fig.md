# Pattern: Strangler Fig

## Problem

Rewriting a large legacy monolith in a single shot is high-risk and rarely succeeds — the cutover is too big, the team can't ship features during the migration, and any bug in the new system threatens the business. The strangler fig pattern grows the new system around the old, replacing functionality piece by piece behind a routing facade, until the legacy is fully strangled and can be removed.

## Use When

- A legacy monolith must be replaced but a big-bang rewrite is too risky
- The legacy system must keep serving traffic while the new one is built
- Features are well-bounded — pieces can be lifted out one at a time
- Business needs continuous feature delivery, not a multi-year freeze

## Avoid When

- The legacy is small and a focused rewrite is cheaper
- The legacy can't be fronted by a proxy (e.g., desktop-only, no network boundary)
- Bounded contexts are too entangled to extract piece by piece (DB-coupled spaghetti)
- The team can deliver a full replacement in a short, contained project

## Azure Implementation

### Implementation Steps

1. Identify a routing facade in front of the legacy: API gateway, reverse proxy, or App Gateway
2. Pick the first slice to extract: low-risk, well-bounded functionality (read-only endpoints first)
3. Build the new service for that slice; deploy alongside the legacy
4. Route that slice's traffic through the facade to the new service; keep legacy as fallback
5. Validate functional parity (shadow traffic, log comparison, A/B testing)
6. Cut over fully when confidence is established; remove that code from the legacy
7. Repeat for the next slice — gradually replacing the legacy
8. Decommission the legacy once all functionality has been strangled out

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Routing facade | Azure API Management or Application Gateway | Routes per-path to legacy or new service |
| New services | Container Apps | One service per extracted slice |
| Legacy hosting | Existing VMs / App Service / on-premises | Untouched until strangled |
| Data sync (if needed) | Azure Data Factory / CDC | Replicate legacy DB to new service DB during migration |
| Observability | Application Insights | Compare behavior between old and new |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Risk | Strongly reduced — incremental cutover, easy rollback per slice |
| Time to value | Faster — features ship from new services immediately |
| Duration | Long — total migration takes months to years |
| Operational complexity | Two systems running in parallel during migration |
| Cost | Double for the parallel period |
| Data integrity | Hard — keeping data consistent across two systems is the core challenge |

## Common Failure Modes

- **Stalled migration** — First few slices migrated; momentum dies; legacy remains 80% of the system years later.
  - Detection: Migration progress flat for months; new features still going into legacy.
  - Prevention: Executive sponsorship; per-quarter slice targets; never add to legacy after migration starts.

- **Data sync drift** — New service has its own DB synced from legacy; sync breaks or lags; behaviors diverge.
  - Detection: Reconciliation finds different values between old and new.
  - Prevention: One source of truth at any time per entity; tight reconciliation jobs; CDC over batch sync.

- **Hard-to-strangle slice** — One area of legacy is deeply coupled; can't be extracted without massive rewrite.
  - Detection: Slice migration estimate dwarfs all others.
  - Prevention: Plan to leave the hard core for last (or accept it stays); often called "the leftover legacy".

- **Behavior divergence under load** — New service passes tests but behaves differently under prod traffic.
  - Detection: Customer reports after cutover.
  - Prevention: Shadow traffic before cutover (run new in parallel, compare results, no user-facing impact).

## Decision Signals

Use strangler fig when:
- Legacy too risky to big-bang replace
- Continuous feature delivery required during migration
- Slices can be cleanly identified

Skip when:
- Legacy small and isolated; focused rewrite better
- Coupling so tight no slice can be extracted

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| API Management | Routing facade | Path-based routing to legacy or new |
| Application Gateway | L7 routing | Simpler, cheaper than APIM if no governance needed |
| Front Door | Multi-region routing | If legacy and new are in different regions |
| Data Factory / CDC | Data sync during transition | Keep two DBs in sync until cutover |

## Go Implementation Notes

Reverse proxy pattern in Go:
```go
mux.Handle("/api/orders/", newOrderService)
mux.Handle("/api/", legacyMonolith)   // catch-all to legacy
```
As slices migrate, more specific routes are added before the catch-all.

For shadow traffic: duplicate the request to both old and new; compare responses asynchronously; only return the old response to the user during shadow phase.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends strangler fig when describing legacy modernization
- `detect_architecture_risks` — flags missing routing facade, data divergence, stalled migration
- `generate_migration_plan` — produces ordered slice extraction sequence
- `generate_architecture_decision_record` — drafts ADR for strangler vs. rewrite vs. lift-and-shift

## Related Patterns

- **API Gateway** — the routing facade in front of legacy and new
- **Anti-Corruption Layer** — translates between legacy and new domain models during transition
- **Blue-Green / Canary** — slice-level cutover technique
- **CDC (Change Data Capture)** — data sync between legacy and new

## References

- Skill: `../../../microservices-architecture-design/references/system-design-process.md` — strangler fig as a multi-year design effort
- Pattern: `../../../microservices-api-design/references/patterns/api-gateway.md` — gateway is the strangler's routing facade
- Pattern: `blue-green-canary.md` — per-slice cutover technique
