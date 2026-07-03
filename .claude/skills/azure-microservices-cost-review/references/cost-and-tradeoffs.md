# Skill — Microservices Cost and Architecture Trade-offs

## Purpose

Estimate the total cost of ownership for a microservices architecture and make explicit trade-off decisions between cost, complexity, and capability. This skill ensures that architecture decisions are made with full visibility into their cost implications. Use this as the final step after the full design is complete.

## Cost Model — What to Count

### Compute

**Container Apps (recommended for microservices):**
```
Cost = ∑(vCPU hours × $0.15 + memory GB-hours × $0.013)

Example:
  5 services × 2 instances × 0.5 vCPU × 730 hours/month = 3,650 vCPU-hours
  3,650 × $0.15 = $547.50/month

  5 services × 2 instances × 1 GB memory × 730 hours/month = 7,300 GB-hours
  7,300 × $0.013 = $94.90/month

Total: ~$640/month
```

**Optimization:**
- Scale to zero: services that run 8 hours/day save 66% (24/8)
- Reserved instances: 1-year commitment saves 20–30%
- Right-size resources: 0.5 vCPU is often enough (not 2 vCPU)

### Data Storage

**Azure SQL Database:**
```
Cost = DTU tier × cost per tier or vCore × $0.15–0.30/vCore-hour

Example (Standard S1):
  20 DTUs × $30/month = $30/month (small/test)
  
Example (General Purpose, 2 vCore):
  2 × $0.30 × 730 = $438/month (production)
```

**Cosmos DB:**
```
Cost = RUs (request units) × $1.25 per million RUs (consumption pricing)

Example:
  400 RUs provisioned × 730 hours × 3,600 seconds/hour = 1.05 billion RUs/month
  1.05 billion × ($1.25 / 1 million) = ~$1,312/month

vs. Provisioned: $0.125 per 100 RUs/second = $3,600/month for 400 RUs
```

**Decision:**
- SQL for transactional, structured data (small-medium scale)
- Cosmos for global scale, high throughput, NoSQL
- Redis for caching (expensive if used for primary storage)

### Messaging

**Service Bus:**
```
Cost = $10–50/month depending on tier and throughput

Standard: $10/month + $0.50 per million messages
Example: 10 million messages/month = $10 + $5 = $15/month

Premium: $550/month + lower per-message cost (for guaranteed throughput)
```

**Event Hubs:**
```
Cost = throughput units × $0.085/hour

1 throughput unit = 1 MB/sec ingress
Example: 10 MB/sec × 10 units × 730 hours × $0.085 = $6,205/month
```

**Decision:** Service Bus for transactional messaging, Event Hubs for high-volume telemetry.

### API Management (APIM)

**Cost:**
```
Developer tier: ~$40/month (non-production)
Standard: ~$250/month + throughput
Premium: ~$1,500+/month

Throughput costs: $0.50–10 per million calls depending on tier
Example: 100 million calls/month on Standard = $250 + $50 = $300/month
```

**Decision:** APIM if you need gateway governance (rate limiting, auth, analytics). Otherwise, use Container Apps ingress (free).

### Observability

**Application Insights:**
```
Cost = data ingestion volume

First 5 GB/month: free
5–100 GB/month: $2.30 per GB
>100 GB/month: $1.15 per GB

Example: 50 GB/month = 5 GB free + 45 GB × $2.30 = $103.50/month
```

**Log Analytics:**
```
Per GB ingested: $2.30 (if Application Insights ingestion) or $2.76 (standalone)

Example: 30 GB/month = $69/month (included with Application Insights)
```

**Optimization:**
- Sample traces (10% of requests) instead of 100%
- Set log retention (default 30 days; reduce to 7 if budget-constrained)
- Alert aggressively to reduce MTTR (mean time to recovery)

### Total Cost Example — E-Commerce Microservices

**Architecture:**
- 5 Container Apps services (2 instances each)
- 1 Azure SQL for transactional data
- 1 Redis cache
- 1 Service Bus for messaging
- 1 Application Insights + Log Analytics
- 1 APIM (optional)

**Costs:**

| Component | Monthly Cost |
|---|---|
| Container Apps (5 svc × 2 inst × 0.5 vCPU) | $640 |
| Azure SQL (General Purpose, 2 vCore) | $438 |
| Redis Premium | $120 |
| Service Bus | $15 |
| Application Insights (50GB ingestion) | $104 |
| Log Analytics (included) | $0 |
| APIM (optional, Standard) | $300 |
| **Total (with APIM)** | **$1,617** |
| **Total (without APIM)** | **$1,317** |

**Annualized:** ~$15,800–19,400/year

**Cost per user (SaaS example):**
If 10,000 users, cost per user = $1,317 / 10,000 = $0.13/user/month

## Trade-offs: Cost vs. Complexity vs. Capability

### Trade-off 1 — Monolith vs. Microservices

| Aspect | Monolith | Microservices |
|---|---|---|
| Cost | ~$500–1000/month | ~$1500–5000/month |
| Operational complexity | Low (one deployment) | High (many services) |
| Team autonomy | Low (shared codebase) | High (independent services) |
| Deployment frequency | Slow (~weekly) | Fast (~daily or hourly) |
| Scalability | Vertical (bigger box) | Horizontal (more services) |
| When to use | <50 engineers, <1M users | >50 engineers, >1M users |

**Decision:** Monolith unless you have compelling reason to go distributed.

### Trade-off 2 — Kubernetes (AKS) vs. Container Apps

| Aspect | AKS | Container Apps |
|---|---|---|
| Cost | ~$100/node + container cost | Consumption-based, cheap for low load |
| Complexity | High (clusters, namespaces, operators) | Low (managed) |
| Control | Full (networking, scheduling) | Limited (managed) |
| When to use | Complex requirements, existing K8s investment | Greenfield, standard microservices |

**Decision:** Start with Container Apps; migrate to AKS only if constrained.

### Trade-off 3 — Serverless Functions vs. Always-On Services

| Aspect | Functions | Services |
|---|---|---|
| Cost | $0.20 per million executions + execution time | ~$640/month (minimum) |
| Latency | High (cold start 1–5 seconds) | Low (always warm) |
| When to use | Async jobs, low-frequency tasks | User-facing APIs, high-frequency |

**Decision:** Functions for background jobs (email, cleanup); services for APIs.

### Trade-off 4 — Caching Strategy

| Strategy | Cost | Hit Rate | Staleness |
|---|---|---|---|
| No cache | Low (no Redis) | 0% | Real-time |
| Redis cache (full dataset) | ~$120/month | ~80% | <5 min |
| Application-level cache | Medium (compute CPU) | ~50% | <1 min |
| CDN cache | Medium ($50–200/mo) | ~95% | <1 hour |

**Decision:** Cache if read/write ratio > 10:1. CDN cache for static content (CSS, images).

### Trade-off 5 — Strong Consistency vs. Eventual Consistency

| Approach | Consistency | Cost | Complexity |
|---|---|---|---|
| Single database (monolith) | Strong | Low | Low |
| Distributed 2-phase commit | Strong | Medium (coordinated writes) | High |
| Saga + compensation | Eventual | Medium (saga orchestration) | High |
| Caching + async sync | Eventual | Low | Medium |

**Decision:** Distributed 2-phase commit only if regulatory requirement (e.g., financial transactions). Otherwise, eventual consistency with saga.

## Architecture Decision Matrix

Given constraints, choose the architecture:

| Constraint | Decision |
|---|---|
| <$1000/month budget | Monolith on App Service or single Container Apps instance |
| <50 engineers | Modular monolith (one codebase, multiple modules) |
| <1M users | Monolith on AKS is wasteful (don't do it) |
| Need independent deployments | Microservices |
| Need to scale one feature independently | Microservices (that feature is a separate service) |
| Need 99.99% availability | Microservices across regions (multi-region failover) |
| Must process 1M transactions/sec | Microservices with Cosmos DB + Event Hubs |
| Regulated environment (financial, healthcare) | Microservices with strong audit, encryption, compliance |

## Cost Optimization Techniques

### 1. Right-size resources

**Before:**
```
All services: 2 vCPU, 2GB memory
Monthly: 5 × $150 = $750
```

**After (right-sized):**
```
API service: 1 vCPU (high load) = $150
Worker services: 0.5 vCPU (background) × 4 = $300
Monthly: $450 (40% savings)
```

### 2. Scale to zero

**Before:**
```
Notification service: always running = $120/month
```

**After:**
```
Notification service: runs only on demand (Azure Function) = $2/month
(assuming 10k notifications/month)
```

### 3. Reserve capacity

**Before (pay-as-you-go):**
```
Container Apps: $640/month
```

**After (1-year reservation):**
```
Container Apps: $640 × 0.75 = $480/month
Savings: $160/month ($1,920/year)
```

### 4. Database optimization

**Before:**
```
Azure SQL General Purpose 2 vCore = $438/month
Used for: transactional data + reporting queries
```

**After (CQRS):**
```
Azure SQL (write model, 1 vCore) = $219/month
Azure Cognitive Search (read model) = $50/month
Total: $269/month (39% savings)
```

### 5. Log retention

**Before:**
```
Log Analytics: 365 days retention = $200/month
```

**After:**
```
Log Analytics: 30 days retention + 60 days archive = $50/month
Archive to blob storage: $0.50/GB (~1GB/day = $15/month)
Total: $65/month (68% savings)
```

## Worked Example — Cost Optimization Journey

**Starting architecture (Year 1):**
```
5 Container Apps services (1 vCPU, 2GB each): $1,500
2 Azure SQL (General Purpose, 2 vCore): $876
Redis Premium: $120
Service Bus: $50
APIM: $300
Observability: $150
Total: $2,996/month
```

**Problem:** Cost per user at $0.30/month. Business wants $0.15/month.

**Optimization (Year 2):**
```
Right-size: Reduce vCPU to 0.5 (services are not CPU-bound)
  Impact: -$500/month

CQRS: Split write and read databases
  Impact: -$200/month (simpler write DB, Cognitive Search is cheap)

Remove APIM: Use Container Apps ingress instead
  Impact: -$300/month

Reserved instances: 1-year commitment
  Impact: -$400/month (20% discount)

Log retention: 30 days instead of 365
  Impact: -$100/month

New total: $1,496/month (50% reduction)
Cost per user: $0.15/month ✓
```

## Verification Questions

1. **Total cost:** What's the monthly Azure bill? Does it align with business expectations?

2. **Cost drivers:** Which components cost the most? Can you reduce those?

3. **Trade-offs:** Are you paying for capability you don't use? (e.g., APIM if you have simple routing)

4. **Optimization:** Can you scale to zero? Reserve capacity? Use cheaper alternatives?

5. **Scalability:** As load grows 10x, what happens to cost? 10x load should not cost 100x.

## What to read next

- For total cost of ownership calculator: Azure Pricing Calculator (pricing.azure.com)
- For optimization: Azure cost management tools
- For architectural patterns that reduce cost: all prior skills (right design = efficient design)
