# Pattern: CQRS

## Problem

Reads and writes have different shapes, different scaling needs, and different consistency requirements. The order service must write strongly-consistent transactions but also serve dashboards that aggregate millions of orders. Forcing both through the same model means writes are slowed by indexes for queries, queries are slowed by row-locking from writes, and the schema becomes a compromise that fits neither well.

## Use When

- The read load is significantly higher than the write load (10x+ ratio is common)
- Reads need denormalized, projected views that don't match the write schema (dashboards, search, aggregations)
- The team can tolerate eventual consistency between write and read sides
- Different consistency or availability requirements apply to reads vs. writes

## Avoid When

- Read and write loads are similar — the added complexity outweighs the benefit
- The team has no experience with eventual consistency — debugging is harder
- Reads need to be strongly consistent with the most recent write (read-your-writes scenario)
- The domain is simple CRUD with no analytical or search needs

## Azure Implementation

### Implementation Steps

1. Identify the bounded contexts where reads and writes diverge significantly
2. Choose write side: Azure SQL or PostgreSQL for ACID transactions, normalized schema
3. Choose read side(s): one or more stores optimized per query pattern (Cognitive Search, Cosmos, Redis, columnar warehouse)
4. Connect them with an event stream: write side publishes events via outbox; projectors consume and update read stores
5. Surface eventual consistency in the API contract (return version numbers, "last updated" timestamps)
6. Handle read-your-writes with sticky reads or by returning the writer's view from the response

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Write side | Azure SQL | Normalized schema, strict transactions, smaller resource tier |
| Read side (search) | Azure Cognitive Search | Indexed projection, full-text and faceted queries |
| Read side (aggregation) | Cosmos DB or Synapse | Denormalized for dashboard queries |
| Event stream | Service Bus / Event Hubs | Carries write-side events to projectors |
| Projector | Container Apps worker | Subscribes to events, updates read store |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Read performance | Strongly improved — queries served from optimized stores |
| Write performance | Slightly improved — write store has fewer indexes |
| Consistency | Eventual between sides; reads can be stale |
| Complexity | High — two schemas, projector code, event plumbing |
| Cost | More services and storage; offset by smaller individual tiers |
| Operational overhead | Projector lag monitoring, rebuild procedures |

## Common Failure Modes

- **Projector lag** — Read store falls behind write store; users see stale data.
  - Detection: Lag metric (newest event timestamp minus newest processed) exceeds threshold.
  - Prevention: Alert on lag >N seconds; scale projector horizontally; partition by aggregate.

- **Projector divergence** — Bug in projector causes read store to deviate from write store; data is silently wrong.
  - Detection: Periodic reconciliation job compares aggregates between write and read.
  - Prevention: Idempotent projectors; rebuildable from event stream; reconciliation tests.

- **Read-your-writes surprises** — User submits write, immediately reads, sees old data (read served from lagging projector).
  - Detection: User reports "I just saved this, where did it go?"
  - Prevention: Return the new state in the write response; sticky reads to write store for N seconds.

- **Schema drift** — Write schema evolves; projectors break or silently lose new fields.
  - Detection: Read store missing fields present in writes.
  - Prevention: Version event contracts; projectors must explicitly handle new fields.

## Decision Signals

Use CQRS when:
- Dashboards or search use cases dominate read traffic
- Write transactions are blocked by indexes serving queries
- The same data must be presented in 3+ different shapes

Skip when:
- Plain CRUD with simple list/detail views — a well-indexed SQL DB is enough
- Team lacks experience with async event flow

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Azure SQL | Write side | ACID, transactions, normalized schema |
| Cognitive Search | Read projection | Full-text, faceting, fast search |
| Cosmos DB | Read projection | Denormalized aggregates, geo-distributed |
| Service Bus | Event channel | Reliable propagation of writes to projectors |

## Go Implementation Notes

Write side commands: `CreateOrderCommand`, `UpdateOrderCommand` — write to SQL, append outbox event.

Read side queries: `GetOrderSummaryQuery`, `SearchOrdersQuery` — read from Cognitive Search.

Projector: subscribes to `order.created`, `order.updated` from Service Bus; calls Cognitive Search index API. Keeps a checkpoint per partition for replay.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — suggests CQRS when read patterns differ significantly from write patterns
- `detect_architecture_risks` — flags CQRS without read-store reconciliation
- `generate_projection_plan` — sketches the event → projection mapping for the described read views
- `map_patterns_to_azure_services` — selects appropriate read-side store per query pattern

## Related Patterns

- **Event Sourcing** — pairs well; events are the natural source for projectors
- **Transactional Outbox** — reliable event publication from write side to projectors
- **Materialized View** — the read store is a materialized view in CQRS

## References

- Skill: `../data-architecture.md` — when to choose CQRS vs. plain DB scaling
- Pattern: `event-sourcing.md` — natural complement; events drive projections
- Pattern: `transactional-outbox.md` — reliable event publication for projectors
