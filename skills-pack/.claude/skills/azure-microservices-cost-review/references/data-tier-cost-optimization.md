# Data Tier Cost Optimization — Cosmos, Azure SQL, Postgres, Storage, Backups

## When to use this reference

When the data tier is dominating the bill (often 40–60% of a microservices estate's total Azure spend), when sizing a new database before deployment, or when an existing database's pricing model no longer matches its workload (e.g., a Cosmos container outgrew Serverless, or a Postgres instance is on a tier that doesn't match its sustained CPU). Use when the question is "what should this database cost?" — not "which database engine should I use?" (that's `azure-data-tier-design`).

## The data-tier cost hierarchy

In a typical Azure microservices estate, data spend stacks like this from largest to smallest:

1. **Cosmos DB throughput (RUs)** — when present, usually the single largest line item per container.
2. **Azure SQL or Postgres vCore + storage** — second tier, predictable.
3. **Backup storage + geo-redundancy** — silent, accumulates over years.
4. **Blob storage at the wrong tier** — invisible until lifecycle policy gets attention.
5. **Redis** — small in absolute terms but often oversized at Premium tier.
6. **Long retention on Log Analytics for "audit"** — see `idle-resource-detection.md` pattern 7.

Optimize in this order. The biggest lever is RU sizing; the smallest is Redis SKU choice. Don't spend a week shaving Redis if Cosmos is bleeding $3K/month.

## Cosmos DB — RU sizing as the dominant cost lever

Cosmos charges Request Units per second of provisioned (or per request, on Serverless). The cost model:

| Mode | Price | When it wins |
|---|---|---|
| Provisioned (Manual) | ~$0.008 per 100 RU/s per hour (~$5.84/100 RU/s/month) | Truly steady load, p95 = p50 |
| Autoscale | ~$0.012 per 100 RU/s per hour at max (bills max(10% of max, actual)) | Variable load with predictable peak |
| Serverless | ~$0.25 per 1M RUs consumed | Spiky, low-volume (<5K RU/s peak, <50 GB) |
| Reserved Capacity | 1-year ~25% off, 3-year ~65% off provisioned | Steady ≥4K RU/s for the full term |

### Sizing the RU/s correctly

Most Cosmos cost waste comes from over-provisioning. The sizing protocol:

1. **Measure actual RU per operation.** Cosmos returns `x-ms-request-charge` on every response. Log it from day 1 in dev and staging.
2. **Multiply by peak ops/sec.** Point reads on a 1 KB doc are ~1 RU. Inserts ~5 RU. Cross-partition queries can be 100s.
3. **Add 50% headroom.** Round up to the nearest 1,000.
4. **Pick the mode**:
   - If peak ≤ 5K RU/s and storage ≤ 50 GB → **Serverless**. Often 30–60% cheaper for spiky workloads.
   - If peak ≥ 5× the trough → **Autoscale**. Pay 10% of max during quiet hours.
   - If peak ≈ trough (constant load) → **Provisioned + Reserved Capacity**. Cheapest per RU.

### Worked sizing example

A container averages 1,200 RU/s with bursts to 4,500 RU/s. Storage 8 GB.

- Wrong default: Provisioned at 10,000 RU/s "for safety" → **$584/month**.
- Right answer A — Autoscale max 6,750 RU/s (4500 × 1.5): bills 675 RU/s minimum (~$59) + actual usage above that. Typical monthly bill ~$200–280.
- Right answer B — Serverless: 1,200 × 60 × 60 × 24 × 30 = 3.1 billion RUs/month × $0.25/1M = **$777/month**.

For this workload, Autoscale wins. Serverless wins only when the workload is genuinely sparse (long idle periods); a sustained 1,200 RU/s baseline pays the per-request tax all day.

### Cross-partition query cost

A query without the partition key in `WHERE` is a **fan-out scan** — Cosmos hits every physical partition. RU cost is roughly (per-partition RU × partition count). At scale this is the most expensive thing you can do in Cosmos. One offender adds 5,000–10,000 RU/s of "phantom" provisioning.

Detection: enable Cosmos diagnostic logging to Log Analytics, then:

```kql
AzureDiagnostics
| where ResourceProvider == "MICROSOFT.DOCUMENTDB"
| where Category == "DataPlaneRequests"
| where requestCharge_d > 100   // anything over 100 RU is a red flag
| project TimeGenerated, requestCharge_d, statusCode_s, activityId_g, queryHash_g
| order by requestCharge_d desc
```

Fix the query (add partition key filter), not the throughput.

## Azure SQL — DTU vs vCore

Two purchase models, often confused:

| Model | When | Cost shape |
|---|---|---|
| **DTU** (Basic, Standard, Premium tiers) | Small workloads, dev/test, simple sizing | Bundled compute + IO + storage; fixed price per tier |
| **vCore** (General Purpose, Business Critical, Hyperscale) | Production OLTP, predictable resource needs, RI eligibility | Separate compute + storage; supports HA, AZ redundancy, RIs |

**Default for prod**: vCore General Purpose. Reasons: RI-eligible (~33% off 1-year, ~55% off 3-year), separate storage scaling, AZ-redundancy option, transparent CPU/memory sizing.

DTU sizing chart for reference:

| Tier | DTUs | Approx vCore equivalent | Monthly |
|---|---|---|---|
| Basic | 5 | ~0.1 vCore | $5 |
| Standard S1 | 20 | ~0.5 vCore | $30 |
| Standard S3 | 100 | ~1 vCore | $150 |
| Premium P1 | 125 | ~1 vCore + fast storage | $465 |

vCore pricing (GP, single DB, ~$0.20/vCore-hour for Gen5):

| vCores | Monthly | With 1-yr RI | With 3-yr RI |
|---|---|---|---|
| 2 | ~$365 | ~$245 | ~$165 |
| 4 | ~$730 | ~$490 | ~$330 |
| 8 | ~$1,460 | ~$980 | ~$660 |

### Serverless tier — when it wins

Azure SQL Serverless (vCore GP only) auto-pauses on idle (>1 hour no connections, configurable) and bills 0 vCore-seconds while paused.

Wins for:
- Dev and test DBs (commonly idle nights/weekends → 60–70% savings).
- Internal admin tools queried <2× per day.
- New services in pre-launch, traffic uncertain.

Loses for:
- Customer-facing OLTP (auto-resume cold-start is ~30 seconds — the first request errors if the app doesn't retry).
- DBs that get external connection probes (health checks, monitoring) every few minutes — these keep the DB awake without doing useful work, and you pay full vCore.

### Hyperscale — when to consider

Hyperscale separates compute from storage; storage scales to 100 TB independently, with rapid backup/restore (point-in-time restore in seconds, not hours). Premium for the architecture, but the per-vCore compute price is similar to GP. Pick Hyperscale when storage exceeds ~4 TB or PITR speed is operationally critical (e.g., frequent test-environment seeding from prod). Otherwise stay on GP.

## Postgres Flexible Server — tier selection

Three tiers, mapped to workload profiles:

| Tier | Use case | Cost shape |
|---|---|---|
| **Burstable (B-series)** | Dev, low-traffic prod, sidecar DBs | Cheapest; CPU credits cap sustained CPU at ~10–20% |
| **General Purpose (D-series)** | Default for prod backend services | Predictable CPU; OLTP sweet spot |
| **Memory Optimized (E-series)** | Memory-heavy workloads (large `shared_buffers`, in-memory analytics) | ~30% more $/vCore than GP, 2× memory |

Pricing reference (West Europe, verify on `azure.microsoft.com/pricing/details/postgresql/flexible-server/`):

| SKU | vCores | Memory | Monthly (PAYG) | With 1-yr RI |
|-----|--------|--------|----------------|--------------|
| B1ms | 1 | 2 GB | ~$25 | n/a (Burstable has no RI) |
| B2s | 2 | 4 GB | ~$50 | n/a |
| D2s_v3 | 2 | 8 GB | ~$140 | ~$95 |
| D4s_v3 | 4 | 16 GB | ~$280 | ~$190 |
| D8s_v3 | 8 | 32 GB | ~$560 | ~$380 |
| E4s_v3 | 4 | 32 GB | ~$360 | ~$245 |

**Sizing rule**: start at D2s_v3 or D4s_v3 for prod OLTP. Monitor CPU and memory pressure for 2 weeks. Resize *down* if sustained <40%, resize *up* if sustained >70%. Don't pre-size to D8 "for headroom" — the bill is real and you can scale online with ~60 second downtime.

### Storage cost — the silent accumulator

Flexible Server storage is ~$0.115/GB/month (P10 SSD, GP tier). For a 500 GB DB: $57.50/month. IOPS scale with storage size; provisioned IOPS for high-throughput workloads cost extra. Two common cost leaks:

1. **Auto-grow with no ceiling**. Storage grows automatically as the DB fills, never shrinks. A DB that grew to 2 TB during a one-time migration stays at 2 TB and bills $230/month for storage forever. Manual remediation: dump, recreate at the right size, restore — non-trivial downtime, plan accordingly.
2. **Over-provisioned IOPS**. Provisioned IOPS on a workload that doesn't need them is pure waste. Default to standard IOPS (scales with storage); enable provisioned only with a measured IO bottleneck.

### Reserved Capacity for Postgres

Postgres Flexible Server supports 1- and 3-year Reserved Capacity on D-series and E-series tiers (not Burstable). Discount: ~30% (1-year), ~55% (3-year). Sizing rule mirrors compute RIs: commit to the p10 of vCore consumption over 90 days, not the average.

## Storage tier transitions — automate via lifecycle

Blob storage pricing (Hot vs Cool vs Cold vs Archive — West Europe-ish, verify current):

| Tier | Storage | Read | Write | Retrieval | Min retention |
|---|---|---|---|---|---|
| Hot | ~$0.018/GB | ~$0.004/10K | ~$0.05/10K | $0 | 0 |
| Cool | ~$0.010/GB | ~$0.01/10K | ~$0.10/10K | $0 | 30 days |
| Cold | ~$0.0036/GB | ~$0.05/10K | ~$0.18/10K | $0.01/GB | 90 days |
| Archive | ~$0.00099/GB | retrieval required | ~$0.18/10K | $0.02/GB + hours | 180 days |

The price gap on rarely-accessed data is large; the gotcha is **early-deletion fees**. Moving a blob to Cool and deleting it 5 days later costs the full 30 days. Plan retention tiers to match real lifecycle.

### Lifecycle policy template (Terraform)

```hcl
resource "azurerm_storage_management_policy" "main" {
  storage_account_id = azurerm_storage_account.main.id

  rule {
    name    = "tier-transitions"
    enabled = true
    filters {
      blob_types = ["blockBlob"]
    }
    actions {
      base_blob {
        tier_to_cool_after_days_since_last_access_time_greater_than    = 30
        tier_to_cold_after_days_since_last_access_time_greater_than    = 90
        tier_to_archive_after_days_since_last_access_time_greater_than = 365
        delete_after_days_since_modification_greater_than              = 2555  # 7 years
      }
    }
  }
}
```

Enable **last-access time tracking** on the storage account (no extra cost; required for `since_last_access_time` rules):

```hcl
resource "azurerm_storage_account" "main" {
  # ...
  blob_properties {
    last_access_time_enabled = true
  }
}
```

## Backup retention — the line item nobody reviews

Default backup retention varies:

| Service | Default | Max | Cost shape |
|---|---|---|---|
| Azure SQL (vCore GP) | 7 days PITR + 1 week LTR | 35 days PITR + 10 years LTR | Free up to DB size; LTR ~$0.05/GB/month |
| Postgres Flexible Server | 7 days | 35 days | Free up to DB size; geo-redundant ~$0.20/GB/month extra |
| Cosmos DB Continuous | 7 or 30 days | 30 days | ~$0.20/GB-month + restore cost |
| Cosmos DB Periodic | 2/day | configurable | Free up to 2× DB size |

**Common waste**:
- 35-day PITR retention on dev databases that don't need it. Drop to 7 days → save 80% of backup cost on the DB.
- Long-term retention (LTR) configured for compliance but never reviewed. A 10-year LTR on a 500 GB DB accumulates ~$2,000/year in backup storage.
- Geo-redundant backup enabled on non-prod. Halve immediately (use locally redundant for dev/test).

Audit annually. Most pre-prod databases need 7 days PITR, no LTR, locally redundant.

## Redis — right-sizing by tier

Redis Cache tiers:

| Tier | Use | Cost shape |
|---|---|---|
| Basic | Dev/test only — single node, no SLA | C0 (250 MB) ~$16, C1 (1 GB) ~$40 |
| Standard | Prod with budget constraint — primary/replica, 99.9% SLA | C1 (1 GB) ~$80, C2 (2.5 GB) ~$155 |
| Premium | Prod with VNet integration, persistence, clustering | P1 (6 GB) ~$415, P2 (13 GB) ~$830 |
| Enterprise | Multi-region active-active, RediSearch, RedisJSON | $$$$ — only for explicit Redis-Enterprise features |

**Common waste**: defaulting to Premium "for prod" without using Premium features. Premium is justified by VNet injection, persistence, or geo-replication. If none of these is in use, Standard is correct and ~5× cheaper.

For caching where data loss is tolerable (the entire point of a cache), **Standard tier is the right default for prod**. Premium only if a specific feature is needed.

## Decision table — picking the data tier mode

| Workload profile | Pick |
|---|---|
| Cosmos, peak <5K RU/s, spiky, <50 GB | Cosmos Serverless |
| Cosmos, steady 1K–4K RU/s baseline + 2–3× peaks | Autoscale at 1.5× peak |
| Cosmos, steady ≥5K RU/s, 12+ month horizon | Provisioned + 1-yr Reserved Capacity |
| SQL, dev/test, idle nights/weekends | Serverless vCore with 1-hour auto-pause |
| SQL, prod OLTP, ≥1 vCore sustained | vCore GP + 1-yr RI |
| SQL, storage > 4 TB or PITR-speed critical | Hyperscale |
| Postgres, low-traffic prod or sidecar | Burstable B1ms or B2s |
| Postgres, prod OLTP | Flexible D2s_v3 → D4s_v3 + 1-yr RI |
| Postgres, in-memory heavy (analytics, large `shared_buffers`) | Memory-Optimized E-series |
| Redis, ephemeral cache, no VNet needed | Standard tier |
| Redis, requires VNet, persistence, or geo-replication | Premium tier |
| Blob storage, mixed hot/cold access | Lifecycle policy: Hot → Cool (30d) → Cold (90d) → Archive (365d) |

## Brownfield — migrating between tiers

Tier migrations have different friction profiles:

- **Cosmos Provisioned → Autoscale**: zero-downtime config change. Do this first; almost always recovers cost.
- **Cosmos Provisioned → Serverless**: requires container recreation (data migration). Plan with the data-tier migration playbook.
- **Azure SQL vCore Provisioned → Serverless**: in-place tier change, ~60 second downtime. Reversible.
- **Azure SQL DTU → vCore**: in-place tier change, ~5 minute downtime. Do this to unlock RI eligibility.
- **Postgres Burstable → General Purpose** (or vice versa): in-place tier change with brief restart. Reversible.
- **Hot → Cool/Archive blob**: lifecycle policy handles it transparently; no app changes.

## Anti-patterns

- **Default-indexing every Cosmos container**. Cosmos auto-indexes every property; writes pay RU per indexed path. For high-write containers, customize the indexing policy to include only queried paths. 30–50% RU savings common.
- **Provisioning Cosmos at "10K RU/s for safety"**. The platform charges from minute one. Measure first; size to p95 × 1.5; use Autoscale or Serverless to absorb variability.
- **Geo-redundant backups on dev**. Dev environments don't need cross-region disaster recovery. Half the backup cost evaporates.
- **35-day PITR on every database**. Most workloads need 7 days. Set PITR per environment, not as a tenant-wide default.
- **Redis Premium with no Premium-tier features in use**. Premium is for VNet injection, persistence, geo-replication. If you don't use those, Standard is correct.
- **Hot-tier blob lifetime for build artifacts**. Artifacts older than 30 days are almost never read. Lifecycle to Cool or Cold; the read price is irrelevant for artifacts you re-fetch once a year.
- **Auto-grow storage with no ceiling on Postgres or SQL**. Storage grew during a one-time migration and now bills forever. Set a ceiling or reclaim manually.

## What this is not

This reference is the data-tier cost lens. For *which engine to use* (Postgres vs Cosmos vs Mongo etc.) and the operational design of each, see `azure-data-tier-design`. For compute-side cost decisions (the other ~30–40% of the bill), see `compute-tier-cost-analysis.md`. For RI/Savings Plan commitments — the cross-cutting commitment strategy that applies to compute and data tiers — see `reserved-instances-and-savings-plans.md`. For detection of zero-connection or under-utilized databases (the input to many resize decisions here), see `idle-resource-detection.md`. For the general cost framework, see `cost-and-tradeoffs.md`.
