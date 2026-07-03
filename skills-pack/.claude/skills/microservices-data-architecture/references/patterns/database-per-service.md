# Pattern: Database per Service

## Problem

Shared databases create hidden coupling between services. A schema change in one service breaks another. A long-running query from one team locks tables for everyone. The "independent" services are actually a distributed monolith — they share the most critical resource. Database per service makes each service the sole owner of its data, accessible only through its API or its events.

## Use When

- Services must deploy and evolve independently — schema changes can't require coordinated releases
- Different services have different storage needs (one wants SQL, another wants Cosmos, a third wants Redis)
- Workload isolation matters — one service's queries should not slow another
- Compliance or security demands data segregation per service

## Avoid When

- Strong cross-service transactions are required (use a monolith or saga, not shared DB)
- The data is genuinely shared and small (reference data: countries, currencies) — duplicate or use a reference service
- The team is too small to operate many databases (one DB per service has operational cost)

## Azure Implementation

### Implementation Steps

1. For each service, choose its storage based on its workload (SQL for transactions, Cosmos for scale, Redis for cache)
2. Forbid direct DB access from other services — only via the owner's API or its published events
3. For cross-service reads, the consuming service either calls the owner's API or maintains its own projection
4. For cross-service writes, use sagas (no 2PC across DBs)
5. Implement per-service backup, recovery, and monitoring
6. Document the data ownership map: who owns what entity (the "single source of truth" rule)

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Per-service SQL | Azure SQL elastic pool | Each DB isolated, shared compute saves cost |
| Per-service NoSQL | Cosmos DB per service | Separate accounts or databases per service |
| Per-service cache | Redis per service | Isolated cache, no cross-service key collisions |
| Cross-service queries | Service APIs or read projections | Never direct DB access |
| Cross-service writes | Sagas via Service Bus | Coordinated through events, not 2PC |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Independence | Services can evolve, deploy, scale independently |
| Operational overhead | More DBs to manage, monitor, back up |
| Cross-service queries | Harder; require API calls or maintained projections |
| Consistency | Eventual across services; saga or compensation for cross-service writes |
| Cost | Higher base cost (one DB per service); offset by right-sizing each |

## Common Failure Modes

- **Sneaky direct DB access** — A service reaches into another service's DB for "convenience" or "performance".
  - Detection: Network audit shows cross-service DB connections; schema change in one service breaks another.
  - Prevention: Network policies block direct DB access; require all reads through API or events.

- **Distributed monolith via API chatter** — Each cross-service read becomes a chain of API calls; pages are slow.
  - Detection: Single user request fans out into 10+ inter-service calls.
  - Prevention: Cache hot data locally; use projections for read-heavy patterns; consider CQRS.

- **Inconsistent reference data** — Country list, status codes, etc. duplicated and drifting across services.
  - Detection: Bug reports about inconsistent dropdown options.
  - Prevention: One reference service owns the data; others cache it with TTL.

- **Operational sprawl** — 30 DBs, each with its own backup, monitoring, alerting setup. Nobody can keep up.
  - Detection: Backup failures discovered weeks late; missing alerts.
  - Prevention: Standard operational tooling (Azure Policy, ARM templates); minimum standards enforced per DB.

## Decision Signals

Adopt database per service when:
- Schema changes in one service block another service's release
- Workloads diverge (OLTP and OLAP fighting in the same DB)
- Compliance or audit demands per-service data isolation

Skip when:
- Cross-service strong consistency is non-negotiable
- The system is small (1–2 services); one DB is fine

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Azure SQL elastic pool | Per-service SQL DBs | Cost-efficient isolation |
| Cosmos DB | Per-service NoSQL | Scale and shape per service |
| Service Bus | Cross-service events | Decouples consumers from producer schema |
| Application Insights | Per-service observability | Track per-DB performance independently |

## Go Implementation Notes

Each service has its own DB connection string injected as env var. Migration tool (`goose`, `golang-migrate`) versioned per-service in its repo. Never share DB credentials across services.

For cross-service reads, prefer event-driven projections over synchronous API calls when the data is read frequently. Use circuit breaker + cache on synchronous calls when projections are overkill.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends DB-per-service when independent deployment is described as a goal
- `detect_architecture_risks` — flags shared DB references or direct DB access across services
- `generate_data_ownership_map` — produces the entity → owning service table
- `map_patterns_to_azure_services` — picks DB technology per service based on workload

## Related Patterns

- **API Gateway** — clients route through the gateway; per-service DBs stay hidden
- **Saga** — coordinates writes across service boundaries since 2PC isn't possible
- **CQRS** — handles cross-service read needs via projections
- **Service Boundaries** — DB-per-service follows from clear bounded contexts

## References

- Skill: `../../../microservices-architecture-design/references/service-boundaries.md` — data ownership rules across services
- Skill: `../data-architecture.md` — when to maintain projections vs. call APIs
- Pattern: `saga.md` — cross-service write coordination
