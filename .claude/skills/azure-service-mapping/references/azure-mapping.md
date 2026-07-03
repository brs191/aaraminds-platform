# Skill — Azure Service Mapping for Microservices

## Purpose

Map microservices architecture decisions to Azure platform services. This skill provides decision frameworks for choosing the right Azure service for hosting, data storage, messaging, and observability. Use this when you have a microservices design and need to select Azure resources.

## Decision Framework — Hosting Compute

### For containerized microservices:

**Azure Container Apps (recommended for microservices)**
- Managed Kubernetes-like platform, serverless pricing
- Built-in autoscaling, secrets, monitoring
- Container-native (Docker images)
- Scale to zero (no cost when idle)
- **When:** Standard microservices, cloud-native apps
- **Cost:** ~$0.15/vCPU-hour + ingress
- Trade-off: Less control than AKS, good default

**Azure App Service (alternative for .NET/Node/Python)**
- Managed platform for web apps, easier than containers
- Built-in scaling, monitoring, slots
- Native code deployment (no Docker required)
- **When:** Simpler services, teams unfamiliar with containers
- **Cost:** ~$15–100/month per instance
- Trade-off: Less portable than containers

**Azure Kubernetes Service (AKS) (for advanced scenarios)**
- Managed Kubernetes, full control
- Custom networking, multi-tenant, on-premises integration
- **When:** Complex orchestration, hybrid scenarios, existing Kubernetes investment
- **Cost:** ~$0.10–0.15/vCPU-hour + node cost (~$100/month per node)
- Trade-off: Operational complexity, not serverless

**Recommendation:**
- Start with Container Apps (simplest, cost-effective)
- Move to AKS only if Container Apps constraints block you

## Decision Framework — Data Storage

### Relational Data (Orders, Customers, Products with ACID guarantees)

**Azure Database for PostgreSQL Flexible Server (recommended — the pack default relational store)**
- Fully managed PostgreSQL; ACID transactions, strong consistency, rich data types
- Automatic backups, HA with a zone-redundant standby
- Lower cost than managed SQL Server; no licensing lock-in
- **When:** the default for a new service — structured, transactional data with joins
- **Cost:** ~$30–100/month for a small Flexible Server
- Trade-off: smaller managed-feature surface than Azure SQL (no SQL Agent, no linked servers)

**Azure SQL Database (only when SQL Server-specific features are needed)**
- Fully managed SQL Server; ACID, strong consistency, automatic backups, HA replicas
- **When:** the workload genuinely needs T-SQL features — SQL Agent jobs, linked servers — or an existing SQL Server schema is being lifted as-is
- **Cost:** DTU model (~$15–100/month) or vCore model (~$0.15–0.30/vCore-hour)
- Trade-off: no open-source portability; default to Postgres unless a SQL-Server feature forces it

### Non-relational / High-scale Data (Caches, Counters, Sessions)

**Azure Redis (in-memory cache)**
- Key-value store, extremely fast
- Atomic operations, pub-sub
- **When:** Caching, sessions, real-time counters
- **Cost:** ~$30/month for small instance
- Trade-off: Data loss if instance restarts (unless persistence enabled)

**Azure Cosmos DB (distributed, globally-replicated database)**
- Multi-model (SQL, MongoDB, Cassandra, Table API)
- Worldwide replication, high availability
- Eventually consistent or strong consistency (at higher cost)
- **When:** Geo-distributed, scale beyond single region, high throughput
- **Cost:** Consumption model (~$1.25 per million RUs) or provisioned (expensive)
- Trade-off: Cost, complexity (tuning RUs is non-trivial)

### Decision Table

| Scenario | Azure Service | Reason |
|---|---|---|
| Customer orders, transactions | Postgres Flexible Server | ACID, strong consistency, the default relational store |
| User sessions, cache | Redis | Fast, ephemeral |
| Geo-distributed reads | Cosmos DB | Global replication |
| Time-series data (telemetry) | Application Insights / Log Analytics | Built for observability |
| Search and analytics | Azure Cognitive Search | Full-text search, faceting |

## Decision Framework — Messaging and Events

### Between-service communication:

**Azure Service Bus (recommended for reliable messaging)**
- Queues and topics, durable
- Dead-letter queues, session handling
- At-least-once delivery semantics
- **When:** Guaranteed delivery, ordering within partition
- **Cost:** ~$10–50/month depending on throughput
- Trade-off: ~50ms latency, not real-time

**Azure Event Hubs (for high-volume, streaming events)**
- Partition-based topics, real-time
- Replay capability (retain events for days)
- High throughput (millions of events/second)
- **When:** Logs, telemetry, time-series data
- **Cost:** ~$15–100/month depending on throughput
- Trade-off: Less transaction semantics than Service Bus

**Azure Event Grid (for reactive events)**
- Pub-sub with built-in subscriptions
- Serverless, low-latency
- Azure-native events (Storage, App Service, etc.)
- **When:** Reactive workflows, Azure service integration
- **Cost:** $0.60 per million events
- Trade-off: Less durability than Event Hubs

### Decision Table

| Scenario | Azure Service | Reason |
|---|---|---|
| Order → Payment → Fulfillment saga | Service Bus | Reliable, ordered delivery |
| System logs flowing to storage | Event Hubs | High volume, replay |
| Azure Storage upload triggers processor | Event Grid | Native integration, event-driven |

## Decision Framework — API Management

**Azure API Management (APIM)**
- Gateway for all APIs
- Rate limiting, authentication, caching
- Developer portal for API discovery
- **When:** Multiple services, external APIs, governance needed
- **Cost:** ~$100–500/month depending on capacity
- Trade-off: Operational overhead, latency (APIM is a proxy)

**Application Gateway + WAF**
- Layer 7 load balancer
- SSL termination, URL-based routing
- Web Application Firewall for security
- **When:** Simple routing, WAF needed, lower cost
- **Cost:** ~$20–100/month
- Trade-off: Less rich API features (no developer portal)

**Container Apps Ingress**
- Built into Container Apps, no separate cost
- Basic routing, SSL termination
- **When:** Simple services, cost-sensitive
- Trade-off: Limited features compared to APIM

## Decision Framework — Observability

**Application Insights**
- Monitoring, tracing, diagnostics for applications
- Dependency tracking, performance monitoring
- Log Analytics integration
- **When:** Every app should emit to Application Insights
- **Cost:** ~$5–50/month depending on ingestion volume

**Log Analytics Workspace**
- Central repository for logs (from all sources)
- KQL (Kusto Query Language) for analysis
- Alerts, dashboards
- **When:** Centralized logging, cross-service correlation
- **Cost:** ~$10–100/month depending on ingestion

**Azure Monitor**
- Unified monitoring across Azure resources
- Metrics from containers, databases, VMs
- **When:** Infrastructure monitoring

**Recommendation:** Application Insights + Log Analytics Workspace. Everything goes to Log Analytics; app-specific traces go to Application Insights.

## Worked Example — E-Commerce Microservices on Azure

**Services and Azure resources:**

| Service | Hosting | Database | Messaging | Observability |
|---|---|---|---|---|
| API Gateway | APIM or Container Apps Ingress | — | — | Application Insights |
| Order Service | Container Apps | Postgres Flexible Server | Service Bus (outbox) | Application Insights → Log Analytics |
| Payment Service | Container Apps | Postgres Flexible Server | Service Bus (topics) | Application Insights → Log Analytics |
| Inventory Service | Container Apps | Postgres Flexible Server (or Cosmos for high scale) | Service Bus (topics) | Application Insights → Log Analytics |
| Fulfillment Service | Container Apps | Postgres Flexible Server | Service Bus (topics) | Application Insights → Log Analytics |
| Notification Service | Azure Functions (email handler) | — | Service Bus (queue) | Application Insights → Log Analytics |
| Catalog Service | Container Apps | Postgres Flexible Server | Event Grid (cache invalidation) | Application Insights → Log Analytics |

**Cost estimate (small scale, US East):**
- Container Apps: 5 services × 2 instances × 0.5 vCPU × $0.15/vCPU-hour × 730 hours/month = ~$270/month
- Postgres Flexible Server (small, 3 services): 3 × ~$50/month = ~$150/month [VERIFY — illustrative]
- Service Bus (standard): ~$30/month
- APIM (consumption): ~$100/month
- Log Analytics: ~$30/month
- Application Insights: ~$30/month
- **Total: ~$610/month**

**Cost optimization:**
- Use Container Apps scale-to-zero for non-critical services (Notification)
- Cache data in Redis instead of querying constantly ($30/month)
- Use Azure Functions for event handlers instead of always-on services
- Reserve Postgres Flexible Server compute (25–30% discount for a 1-year commitment)

## Verification Questions

1. **Compute:** Does Container Apps meet your requirements, or do you need AKS?

2. **Data:** Is your data relational (SQL) or non-relational (Cosmos)? Do you need strong consistency?

3. **Messaging:** Is Service Bus (ordered, reliable) or Event Hubs (high-volume) right for your events?

4. **Gateway:** Do you need APIM governance, or is Container Apps Ingress sufficient?

5. **Observability:** Is everything logging to Log Analytics? Can you trace a request end-to-end?

6. **Cost:** What's your monthly budget? Does the architecture fit?

## What to read next

- For resilience patterns in Azure: `../../microservices-resilience/references/resilience-patterns.md`
- For observability design: `../../azure-microservices-observability/references/observability-design.md`
- For security in Azure: `../../azure-microservices-security/references/security-design.md`
- Azure documentation: learn.microsoft.com/en-us/azure/architecture/guide/
