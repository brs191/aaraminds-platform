---
name: azure-service-mapping
description: Maps architecture concepts (compute, data store, messaging, caching, discovery, sidecar) to specific Azure services with capacity, cost, and trade-off context — Container Apps vs AKS vs App Service, Azure SQL vs Cosmos vs Postgres, Service Bus vs Event Grid vs Event Hubs, Redis, API Management vs Front Door. Use when picking the Azure service for a new microservice, comparing two options, justifying a choice in an ADR, or auditing a design for misaligned selection. Do not use for cost-only optimization (use azure-microservices-cost-review) or raw capability lookup (use Azure docs).
version: 1.0.1
last_updated: 2026-05-30
---

# Azure Service Mapping

## When to use

Trigger this skill when the question is "which Azure service for this concept": compute hosting, data store, messaging broker, cache, gateway, discovery, sidecar. Common triggers: "Container Apps or AKS for this service," "Service Bus or Event Grid for this event flow," "Cosmos DB or Postgres for this workload," "do I need APIM for this API."

Do **not** use this skill for: cost-only optimization of an existing choice (`azure-microservices-cost-review`); raw Azure service capability lookup (go to Microsoft Learn); discussions of non-Azure clouds (out of scope per pack stack).

## The critical decision rule — Container Apps is the default compute platform

For Azure-hosted microservices, **Azure Container Apps is the default compute platform** unless one of these is true: (a) you need AKS-specific features (custom controllers, complex networking, large daemonsets, GPU workloads), (b) you have an existing AKS investment with team familiarity and tooling already in place, or (c) you need fine-grained control over the Kubernetes control plane.

Container Apps gives you scale-to-zero, KEDA-driven autoscaling, Dapr if you want it, revisions for blue-green, and managed identity — without operating the Kubernetes control plane. App Service is for non-container workloads or simple web apps; default to Container Apps over App Service when starting fresh.

## The Azure-service selector

| Concept | Default | Alternatives | Reference |
|---|---|---|---|
| **Compute** — stateless microservice | **Azure Container Apps** | AKS (when justified); App Service (non-container) | `references/azure-mapping.md` |
| **Compute** — stateful (rare) | **AKS with StatefulSet** | Container Apps replicas (no persistent identity) | `references/azure-mapping.md` |
| **Relational DB** | **Azure Database for PostgreSQL Flexible Server** | Azure SQL (when SQL Server-specific features needed) | `references/azure-mapping.md` |
| **Document / NoSQL DB** | **Cosmos DB (NoSQL API)** | MongoDB Atlas on Azure (when Mongo-specific tooling matters) | `references/azure-mapping.md` |
| **Cache** | **Azure Cache for Redis** | Cosmos DB integrated cache (limited) | `references/patterns/cache-aside.md` |
| **Messaging — queue / topic with ordering, DLQ** | **Service Bus** | n/a — Service Bus is the right answer | `references/azure-mapping.md` |
| **Messaging — reactive Azure resource events** | **Event Grid** | Service Bus (when DLQ matters) | `references/azure-mapping.md` |
| **Messaging — high-volume stream / replay** | **Event Hubs** | Service Bus topic (lower throughput, more semantics) | `references/azure-mapping.md` |
| **API Gateway** | **Azure API Management** | Container Apps Ingress (light), Front Door (global routing) | `references/azure-mapping.md` |
| **Global routing / WAF** | **Azure Front Door** | Application Gateway (regional) | `references/azure-mapping.md` |
| **Secrets** | **Azure Key Vault + Managed Identity** | n/a — this is the only correct answer for the stack | `references/azure-mapping.md` |
| **Service discovery** | **Container Apps internal DNS** (for CA estate) | AKS Service / CoreDNS; Dapr name resolution | `references/patterns/service-discovery.md` |
| **Service mesh** (only when justified) | **Dapr** (lightweight) | Istio on AKS (heavy, capable) | `references/patterns/service-mesh.md` |
| **Sidecar** for cross-cutting concerns | **Dapr** | Custom sidecar (rarely justified) | `references/patterns/sidecar.md` |

For full capacity, cost, and trade-off notes per service, see `references/azure-mapping.md`.

## Selection logic

1. **Compute:** Container Apps unless you have a specific AKS reason. State the reason in the ADR — "we need GPU" is a reason, "we like Kubernetes" is not. Switching from Container Apps to AKS later is a 1-2 sprint migration; the cost of starting with AKS unnecessarily is years of operational overhead.

2. **Data:** match the workload pattern.
   - Transactional, relational, joins, ACID — Postgres Flexible Server. SQL only if you need T-SQL features (SQL Agent, linked servers, etc.).
   - Document/key-value with high write throughput, partition-friendly access patterns — Cosmos DB.
   - Heavy aggregation/analytical — separate read model (CQRS), not the operational store.

3. **Messaging:** see `microservices-async-messaging` for the semantic decision. From the Azure side: Service Bus is the default; Event Grid for Azure-native event reactions; Event Hubs only when throughput or replay matters.

4. **Gateway:** APIM only when you have specific needs (auth, rate limit, transformation, dev portal). Container Apps ingress is fine for internal-only APIs. Front Door for global routing or WAF needs.

5. **Discovery + mesh:** within a single Container Apps environment, internal DNS handles service discovery for free; mesh is overkill. Reach for Dapr only when you genuinely need pub-sub abstractions, state management, or cross-runtime workflow primitives. Istio only when you have a multi-cluster, multi-protocol estate that needs strong traffic shaping and observability conventions.

6. **Secrets:** Key Vault, accessed via Managed Identity. There is no second-best option in this stack. Anything else (env vars, config files, separate secrets stores) is a security finding.

## Worked example — brownfield: picking the data store for a new service in an existing estate

Setup: a new `recommendation-service` is being added to an existing Container Apps estate. It needs to store ~10M items, with point lookups by customer ID and occasional batch refresh from analytics. Existing services use Postgres Flexible Server. Team is comfortable with Postgres; less so with Cosmos DB.

Decision walk:

1. **Match workload to data shape.** Read pattern is point lookup by customer ID; data is denormalized (one row per customer with their top-N recommendations as a JSON column or similar). Write pattern is batch refresh (overwrite or upsert).
2. **Evaluate Postgres.** A single Postgres table with `customer_id` PK can serve point lookups in single-digit ms. 10M rows is small for Postgres. Batch refresh is straightforward `INSERT ... ON CONFLICT`.
3. **Evaluate Cosmos.** Cosmos with `/customer_id` partition key would also work and would scale further. But it adds operational surface (RU sizing, partition-key design, schema-less discipline) the team doesn't currently have.
4. **Decide.** Postgres. Reasoning: the workload fits within Postgres comfortably; the team has operational competence; adding Cosmos would introduce a new operational discipline with no measurable benefit. Document in ADR with the volume and access pattern, and a re-evaluation trigger ("if we exceed 100M rows or 5k point-lookups/sec, re-evaluate Cosmos"). See `references/azure-mapping.md`.
5. **Don't over-cache.** Skip Redis for now; Postgres point lookup at this scale is fast enough. Add cache only when measurements show it's needed. See `references/patterns/cache-aside.md`.

## Anti-pattern — choosing AKS because "Kubernetes is the standard"

**Bad:** Starting a new 3-service microservice estate on AKS because "everyone uses Kubernetes" or "we want portability later."

**Why it fails:**
- AKS operational surface (cluster upgrades, node-pool sizing, addon management, RBAC, NetworkPolicy, ingress-controller updates) is multi-engineer-quarter work to keep healthy. For 3 services, it's massively overprovisioned.
- "Portability later" is rarely exercised; the team that built on AKS usually stays on AKS. Container Apps containers are still portable — they're plain OCI images.
- Container Apps gives you the actual operational properties teams want from Kubernetes (rolling deploys, scale-to-zero, KEDA-style autoscaling, managed identity) at a fraction of the cost.

**Detection signal:** ADR or design doc says "AKS for portability" or "AKS for the ecosystem" without naming a specific feature the team needs. Or: AKS chosen for a 2-3 service estate where the team has 0-1 SREs.

**Fix:** Default to Container Apps. Document specific AKS-requiring features if any exist (GPU, custom CRDs, daemonset patterns, large operator ecosystem). If none, Container Apps is correct.

## Verification questions

1. For each Azure service in the design: is there a one-line "why this service over the alternative" written down?
2. For compute: is the choice Container Apps, or is there a documented reason to use AKS / App Service?
3. For data: does the workload pattern match the chosen store's strengths (point lookup → KV/document; relational + joins → SQL/Postgres; high write throughput → Cosmos)?
4. For messaging: is the broker chosen for *semantics* (ordering, DLQ, replay), not familiarity?
5. For gateway: is APIM justified by a specific need, or did it appear by default? If by default, drop it.
6. For mesh / sidecar: is there a measurable problem they solve, or are they speculative architecture?

## What to read next

- `references/azure-mapping.md` — full decision matrix per Azure service category, with capacity / cost / trade-off notes
- `references/patterns/cache-aside.md` — when and how to add a cache; Redis patterns
- `references/patterns/service-discovery.md` — Container Apps internal DNS, AKS Service, Dapr name resolution
- `references/patterns/service-mesh.md` — Dapr vs. Istio decision, sidecar-based concerns
- `references/patterns/sidecar.md` — when to use a sidecar pattern at all
- `azure-microservices-cost-review` skill — for cost-driven re-evaluation of an existing service choice
- `microservices-architecture-design` skill — for the broader system-level Azure mapping in context
