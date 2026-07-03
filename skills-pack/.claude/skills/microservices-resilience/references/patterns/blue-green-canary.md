# Pattern: Blue-Green / Canary Deployments

## Problem

Deploying directly to production exposes 100% of users to bugs the moment a new build ships. Rolling back means another deploy and another window of risk. Blue-green and canary release strategies decouple "deploy" from "release to users", allowing the new version to run alongside the old, with traffic gradually shifted as confidence grows.

## Use When

- Production traffic matters (paying customers, revenue-critical paths)
- Bugs in new releases would cause significant impact (data corruption, outage, regression)
- The team can validate behavior against real traffic before full cutover
- Rollback must be near-instant — flipping a switch rather than re-deploying

## Avoid When

- Internal or pre-production environments with low risk
- Workload is stateful in ways that make parallel versions hard (incompatible DB migrations)
- The cost of running two production copies is prohibitive
- The release adds value only at 100% (e.g., schema migration that can't be partial)

## Azure Implementation

### Implementation Steps

1. Deploy the new version (green) alongside the existing (blue), with no traffic routed to green
2. Validate green with synthetic traffic (smoke tests, integration tests against prod)
3. Shift a small percentage of real traffic to green (canary: 1%, then 5%, then 25%, then 100%)
4. Monitor key metrics on green: error rate, latency, business KPIs
5. If green is healthy at each stage, increase traffic; if not, shift back to blue instantly
6. Once green has 100%, retain blue for a rollback window (often 24–72 hours)
7. Decommission blue once confidence is established

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Container Apps revisions | Multi-revision mode | Two revisions get weighted traffic split |
| App Service slots | Deployment slots | Swap-with-preview between staging and prod |
| AKS / Istio | Traffic split via VirtualService | Percentage-based routing per route |
| Azure Front Door | Multi-origin with weights | Route % to new origin |
| Feature flag (alternative) | Azure App Configuration | Code-level toggle independent of deploy |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Risk reduction | Strong — bugs affect small slice of users before full release |
| Rollback speed | Instant — flip traffic weights |
| Cost | Double infrastructure during the cutover window |
| Database compatibility | New code must work with old schema (backward-compatible migrations) |
| Operational complexity | Requires automation to manage shifts, metrics, rollback |
| Observability burden | Must compare metrics between blue and green continuously |

## Common Failure Modes

- **Incompatible schema migration** — Green expects new column, blue uses old; one of them errors.
  - Detection: Errors specific to one version after migration.
  - Prevention: Schema changes happen in compatible steps — add column nullable, deploy both versions, then backfill, then make required.

- **Canary metrics not representative** — 1% traffic is too small to detect regression that affects 1% of users.
  - Detection: Bug surfaces only after full rollout.
  - Prevention: Choose canary percentage to reach statistical significance; canary by user segment, not random.

- **Sticky sessions broken** — User's session bounces between blue and green; state mismatches surface as bugs.
  - Detection: Session-related errors during canary phase.
  - Prevention: Use sticky routing (session affinity) during the canary window; or make services stateless.

- **Forgotten blue** — Old version left running for weeks; cost accumulates and confusion grows.
  - Detection: Cost dashboards show duplicate services beyond rollback window.
  - Prevention: Automated cleanup of old revisions after N days.

## Decision Signals

Use blue-green or canary when:
- Production has paying customers and reputational risk
- Recent incidents were caused by a bad deploy
- Schema changes accompany code changes (need careful coordination)

Skip when:
- Internal tooling, low risk
- Cost of double infra is unjustified

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Container Apps revisions | Built-in canary | Multi-revision mode with weighted routing |
| App Service slots | Swap with preview | Validate before going live |
| Front Door | Multi-origin canary | Global traffic shifting |
| Istio VirtualService | Granular routing | Per-route, per-header canary |

## Go Implementation Notes

Apps need to handle running concurrently:
- Forward compatibility: old version handles new event types gracefully (ignore unknown fields)
- Backward compatibility: new version handles old data shapes
- Feature flags decouple "deploy" from "user-visible release"

Container Apps `--traffic-weight` example:
```
az containerapp ingress traffic set \
  --revision-weight rev-v1=90 rev-v2=10
```

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends canary when describing risky production releases
- `detect_architecture_risks` — flags incompatible schema migrations, missing rollback plan
- `generate_canary_plan` — drafts stage progression (1% → 5% → 25% → 100%) with metric gates
- `map_patterns_to_azure_services` — picks Container Apps revisions vs. App Service slots vs. Front Door

## Related Patterns

- **Feature Flags** — release control independent of deploy
- **Circuit Breaker** — protect canary from cascading failure
- **Distributed Tracing** — required to compare blue vs. green behavior

## References

- Skill: `../../../azure-service-mapping/references/azure-mapping.md` — Container Apps revisions for blue-green
- Pattern: `strangler-fig.md` — large-scale gradual migration uses similar traffic-shifting
- Pattern: `../../../microservices-async-messaging/references/patterns/distributed-tracing.md` — needed to validate canary health
