---
id: cross-cutting/01-azure-mapping-tradeoff
area: cross-cutting
exercises:
  - .claude/skills/azure-service-mapping/references/azure-mapping.md
  - .claude/skills/microservices-async-messaging/references/patterns/async-messaging.md
  - .claude/skills/azure-service-mapping/references/patterns/cache-aside.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Justify a pattern → Azure service mapping

## Context

Attach `09-azure-mapping.md` and the relevant pattern cards (async-messaging, cache-aside, service-mesh, api-gateway).

## Prompt

I need to pick Azure services for the following patterns in a mid-scale Azure architecture (~100 req/s peak, multi-tenant SaaS):

1. Async messaging between order, payment, fulfillment
2. Cache for product catalog reads
3. Service mesh for internal traffic
4. API gateway for external traffic

For each, recommend a specific Azure service, justify the choice, and name a credible alternative with the trade-off.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Service Bus over Event Hubs for transactional messaging** — picks Service Bus with the right reason (reliable, DLQ, sessions for ordering); names Event Hubs as the alternative for high-volume streaming
- [ ] **Service Bus Standard sized for the load** — explicitly picks Standard not Premium at 100 req/s
- [ ] **Cache: Redis with rationale** — picks Azure Cache for Redis with a tier (Standard typically), notes hit-rate target, mentions invalidation strategy
- [ ] **Cache alternative** — names CDN as an alternative for static product images / pages, or in-process cache for very hot small data
- [ ] **Mesh: Dapr or Istio with reason** — picks based on team-readiness and feature need; doesn't recommend Istio unreflectively
- [ ] **Mesh alternative or skip** — acknowledges that at 100 req/s with a small cluster, Container Apps + Dapr is plenty; full Istio is overkill
- [ ] **Gateway: APIM or Container Apps Ingress** — picks based on whether developer portal / policies / rate-limit-per-consumer matter; if not, Container Apps Ingress is free and sufficient
- [ ] **Each choice has a stated trade-off** — every recommendation names what is given up by the choice (cost, latency, operational complexity, feature set)

## Reference output

> **1. Async messaging — Azure Service Bus Standard.**
>
> Reasons: transactional semantics (the order/payment/fulfillment flow needs reliability, not raw throughput), dead-letter queue support for poison messages, session-based ordering (`sessionId = orderId`) for per-order serial processing, ~$10–15/month at this volume.
>
> Standard not Premium: Premium is for guaranteed throughput at very high scale (≥20 MB/s ingress sustained); 100 req/s is two orders of magnitude below where Premium pays. Premium is ~$550+/month.
>
> Alternative: Event Hubs. Pick it if the messages are telemetry-shaped (high volume, replay capability needed, ordering by partition). Here they're transactional commands/events, not telemetry, so Event Hubs is the wrong fit. Trade-off if you went there anyway: more raw throughput, but you lose the DLQ ergonomics and the session-ordering primitives that the order saga benefits from.
>
> **2. Cache — Azure Cache for Redis (Standard tier, C1).**
>
> Product catalog is read-heavy (assume 10:1 read/write ratio); a cache hit avoids the SQL round-trip and serves in sub-millisecond. Standard tier gives replication for HA at low cost (~$40/month). Target hit rate ≥80%; cache TTL 5 minutes for product details (catalog churn is low), 30 seconds for inventory snippets (more volatile).
>
> Invalidation strategy: explicit `DEL product:{id}` on product update, in the same code path as the DB write. Pub/sub for cross-replica invalidation if cache is sharded.
>
> Alternative: Azure Front Door / CDN for static product imagery and pages (cache at edge). Different layer, useful in parallel to Redis. In-process cache (`bigcache`, `freecache`) for very small hot data sets in a single service; cheap but doesn't share across replicas.
>
> Trade-off of choosing Redis: another moving part to operate, ~$40/month, cache-coherency bugs (stale reads after writes) are now possible and must be handled. The alternative trade-off — no cache — is higher SQL load and worse P95 latency.
>
> **3. Service mesh — Dapr (with Container Apps).**
>
> At this scale and team size (implied by the question), Dapr in Container Apps gives mTLS, service invocation, pub/sub abstraction, and observability via sidecar without the operational cost of Istio's control plane. Cluster maintenance is Microsoft's problem.
>
> Reasons: app-level APIs (state, pub/sub, invoke via localhost) decouple from specific Azure services; mTLS is automatic; the control plane is managed; the team's learning curve is shallow.
>
> Alternative: Istio on AKS. Pick it if you need fine-grained traffic shifting, mature multi-cluster, large-scale resilience features (outlier detection, complex circuit breakers) — none of which are warranted at 100 req/s. Linkerd is a third option: simpler than Istio but lacks the breadth of Dapr's app-level APIs. Trade-off of choosing Istio anyway: significantly more operational complexity, control-plane resource cost (~0.5 vCPU + memory), longer onboarding.
>
> Honestly: at 100 req/s, mesh is often deferrable entirely. Container Apps + Application Insights gives you observability and mTLS-on-ingress without a mesh. If you can defer mesh by 6 months and ship features, do that. The mesh becomes worthwhile when the cluster reaches ~10+ services or you need uniform policy enforcement.
>
> **4. API gateway — Azure API Management (Standard tier) if governance matters; otherwise Container Apps Ingress.**
>
> Decide by the questions:
> - Do you need a developer portal for external consumers? → APIM.
> - Do you need rate-limit-per-consumer-key, quota management, transformations? → APIM.
> - Do you have many external API consumers? → APIM.
> - None of the above? → Container Apps Ingress (free, handles routing, TLS, basic auth).
>
> For a 100 req/s SaaS, APIM Standard at ~$250/month gives the governance package; Container Apps Ingress is free. Pick the cheap one until you actually need the features.
>
> Alternative: Application Gateway + WAF — strong web-app firewall, simpler than APIM, no developer portal. Useful if security is the dominant concern and governance isn't. Front Door is the right answer if you're going multi-region.
>
> **Summary trade-off**
>
> | Choice | Cost / month | What you give up |
> |---|---|---|
> | Service Bus Standard | ~$15 | High-volume streaming features (use Event Hubs if that becomes the need) |
> | Redis Standard | ~$40 | One more moving part; cache-coherency discipline |
> | Dapr in Container Apps | ~$0 (in platform) | Some advanced mesh features (multi-cluster, fine traffic shifting) |
> | Container Apps Ingress | $0 | Developer portal, per-consumer quotas (defer APIM until needed) |
> | **Total** | **~$55** for the cross-cutting infra | |

## Notes

Tests whether the responder applies the size/scale framing from `09-azure-mapping.md` and the cost framing from `12-cost-and-tradeoffs.md`. Catches LLMs that default to the highest tier of every service.
