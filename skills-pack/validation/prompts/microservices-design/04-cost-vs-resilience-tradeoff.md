---
id: microservices-design/04-cost-vs-resilience-tradeoff
area: microservices-design
exercises:
  - .claude/skills/azure-microservices-cost-review/references/cost-and-tradeoffs.md
  - .claude/skills/microservices-resilience/references/resilience-patterns.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Trade cost against resilience for a small team

## Context

Attach `12-cost-and-tradeoffs.md` and `06-resilience-patterns.md`.

## Prompt

A four-person team is launching a SaaS MVP on Azure. They want microservices "because that's the right way", and their architect's first draft has: 8 services on AKS, multi-region active-active, Cosmos DB everywhere, APIM in front, Premium Service Bus, full mTLS via Istio, and ~$5,000/month Azure budget. The actual product is a small CRM with maybe 200 paying users in year one. Critique honestly and propose a leaner architecture that preserves the right invariants.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Calls out over-engineering** — explicitly says the first draft is misaligned with the team size and load; doesn't soften this
- [ ] **Right-sizes the deployment platform** — proposes Container Apps (or App Service) instead of AKS, with a concrete cost or operational-complexity argument
- [ ] **Right-sizes the data store** — proposes Azure SQL (or PostgreSQL Flexible Server) instead of Cosmos for transactional data; reserves Cosmos / NoSQL for cases that actually need it
- [ ] **Right-sizes messaging** — proposes Service Bus Standard (not Premium) at the expected message volume; or no broker at all if async is not yet needed
- [ ] **Drops or defers what doesn't pay** — at least two: APIM (use Container Apps Ingress until traffic warrants), multi-region (single region with backups for year one), Istio (Dapr or none until mesh is justified)
- [ ] **Preserves real invariants** — explicitly keeps: per-service data ownership, observability (Application Insights), security (Managed Identity + Key Vault), at least one bounded context per service
- [ ] **Considers modular monolith** — at least mentions that 200 users in year one might not need microservices at all; or, if defending microservices, names the specific reason (team boundaries, deployment independence) rather than "best practice"
- [ ] **Concrete budget delta** — provides an actual estimated monthly cost for the leaner architecture, with the saved amount visible (e.g., "$5k → $400")

## Reference output

> The first draft is misaligned by an order of magnitude. Two hundred paying users in a small CRM does not need eight services on AKS with active-active multi-region. The pattern catalog is being applied without checking the load, the team, or the product.
>
> **What's wrong specifically**
>
> - AKS for a four-person team is operational debt nobody is going to pay down. The hidden cost is the team's attention, not the compute bill.
> - Cosmos "everywhere" is a default that ignores the data shape. A small CRM is transactional, structured, and small — that's SQL territory. Cosmos's strength is global distribution and very high throughput; neither applies.
> - APIM at $250+/month buys governance (developer portal, rate-limit per consumer key, policies). None of that is needed for one app and a handful of clients.
> - Premium Service Bus is for guaranteed throughput at high scale. Standard at $10–15/month handles millions of messages per month.
> - Multi-region active-active for 200 users in year one is a multiplier on every other cost and complexity, in exchange for an SLA the product can't yet articulate.
> - Istio adds a control plane and a sidecar per pod. The team will spend more time debugging mesh than building features.
>
> **What to preserve regardless**
>
> 1. Per-service data ownership. Even if there are only 2–3 services, each owns its tables. Avoids the shared-DB sneak-in that makes future decomposition impossible.
> 2. Observability from day one. Application Insights, structured logs, correlation IDs. A single Application Insights workspace is fine.
> 3. Security primitives: Managed Identity for service-to-service auth, Key Vault for all secrets. Zero secrets in config files.
> 4. Resilience basics on outbound calls: timeouts and retries with jitter. Circuit breakers come later.
>
> **Leaner architecture**
>
> | Layer | Lean choice | Cost (rough) | Why |
> |---|---|---|---|
> | Compute | Container Apps, 2–3 services | ~$120–250 | Managed, scale to zero, no cluster ops |
> | DB | Azure SQL Database (General Purpose, 2 vCore) | ~$200 | Familiar, transactional, cheap at this scale |
> | Cache | None initially; add Redis Standard later if hot reads emerge | $0 | Don't pay for what's not needed |
> | Messaging | Service Bus Standard | ~$15 | Only if async work is genuinely needed; if not, defer |
> | Ingress | Container Apps Ingress | $0 (included) | APIM is overkill |
> | Observability | Application Insights with sampling | ~$30 | Free tier + sampling stretches the budget |
> | Secrets | Key Vault | ~$5 | Cheap, non-negotiable |
> | **Total** | | **~$370–500/month** | |
>
> $5,000 → ~$400. The team can run lean for the entire first year on the savings.
>
> **Even better question: do you need microservices at all?**
>
> A modular monolith — one deployable, two or three internal modules with clean interfaces — is often the right shape at this stage. The team can split modules into services later when (a) the team grows past 5–10 engineers and Conway's Law starts to bite, or (b) one module's scale requirements diverge from the others. Decomposing prematurely costs operational complexity that doesn't translate to anything the customer can see.
>
> If microservices are insisted on, two genuinely independent services (e.g., CRM-core and notification-worker) is enough to capture the bounded-context discipline without the operational tax.

## Notes

This is the "say the unpopular thing" prompt. Catches LLMs that politely validate over-engineered designs because the user "asked for microservices". The rubric requires explicit critique.
