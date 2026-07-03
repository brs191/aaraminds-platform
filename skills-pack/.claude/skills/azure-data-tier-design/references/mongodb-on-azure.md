# MongoDB on Azure — Cosmos vCore vs RU vs Atlas

## Purpose

If the workload actually needs MongoDB semantics, three choices exist on Azure: Cosmos DB for MongoDB **vCore**, Cosmos DB for MongoDB **RU-based**, and **MongoDB Atlas** on Azure. They look similar from the outside; they are very different operationally. This reference picks between them.

## When you do not need MongoDB at all

First: confirm you need Mongo semantics. Many "we use Mongo" projects use Mongo as a generic document store with `find()`, `insertOne()`, `updateOne()`, and maybe `aggregate()` with `$match` / `$group`. Cosmos NoSQL/SQL API supports that surface natively at lower cost.

You need real MongoDB if any of the following are load-bearing:

- Complex aggregation pipelines (`$lookup` joins, `$graphLookup`, `$bucket`, multi-stage transforms)
- Multi-document ACID transactions across collections
- Change streams with full Mongo semantics
- Specific Mongo driver features (cursor types, retryable writes with exact Mongo semantics)
- An existing Mongo app being lifted with minimal code change

If none apply, **pick Cosmos NoSQL/SQL API**, not Mongo-on-Cosmos. The bill is lower and the operational surface is smaller.

## The three options

| Option | What it is | When to pick |
|---|---|---|
| **Cosmos DB for MongoDB vCore** | Cluster-based Mongo running on Cosmos backend. Compute-priced like a real Mongo cluster. | Default for new Mongo work on Azure. |
| **Cosmos DB for MongoDB (RU-based)** | Older offering. Pay-per-RU. Wire-compatible with Mongo but with feature gaps. | Legacy; only continue if already deployed. |
| **MongoDB Atlas on Azure** | Real MongoDB, vendor is MongoDB Inc., billed separately. Peered into your Azure VNet. | You need Atlas-specific features (Atlas Search, Atlas Vector Search), the latest Mongo version, or multi-cloud. |

## Cosmos for MongoDB vCore

**Architecture**: cluster of compute nodes (M30, M40, M50, etc.) running a Mongo-compatible engine on the Cosmos storage backend. Priced by cluster size + storage, not by RU.

**Strengths:**
- Real cluster pricing model — predictable, no RU sticker shock
- Closer Mongo feature parity than RU-based (better aggregation surface, larger transactions, more aggregation operators)
- HA via replicas across availability zones
- Azure-native: Private Link, Entra ID auth, integrated billing
- Free 32-vCore tier for dev / small workloads

**Weaknesses:**
- Mongo version lags official MongoDB by 1–2 minor versions
- Some features still gap-mapped (verify your workload's specific operators against the current docs)
- Not multi-cloud — Azure only

**Pick vCore for**: most new Mongo work on Azure. The RU-based model is the wrong default.

## Cosmos for MongoDB (RU-based)

**Architecture**: same Cosmos engine as the NoSQL API, exposed via Mongo wire protocol. Pay per RU.

**Strengths:**
- Serverless and autoscale options
- Multi-region writes (with conflict resolution)
- Same RU model as Cosmos NoSQL

**Weaknesses:**
- RU cost on Mongo workloads is harder to predict than vCore cluster cost
- Larger feature gaps vs real Mongo
- Aggregation pipeline support is incomplete; some operators don't work
- Confusing for Mongo-native developers — Mongo errors that mean "RU exceeded" are not standard Mongo errors

**Pick RU-based only for**: continuing an existing deployment, or workloads that match the Cosmos RU model exactly (small documents, well-known partition key, simple `find` / `update`). For new work, vCore is the better default.

## MongoDB Atlas on Azure

**Architecture**: managed by MongoDB Inc., runs on dedicated Azure VMs in your chosen region. Peered into your VNet.

**Strengths:**
- Newest MongoDB version, fastest feature parity
- Atlas-specific features: Atlas Search (Lucene-based), Atlas Vector Search, Atlas Stream Processing, Charts
- Multi-cloud (deploy clusters across Azure + AWS + GCP)
- Real Mongo semantics — no compatibility gap

**Weaknesses:**
- Separate vendor relationship: billing, support, contract
- Higher cost than Cosmos vCore for equivalent compute
- Not Azure-native — integration via peering and Atlas's own IAM model
- Adds a vendor surface that's outside your normal Azure SOC 2 audit scope

**Pick Atlas when:**

- A specific Atlas feature is load-bearing (Atlas Search or Atlas Vector Search at scale)
- Multi-cloud is a real requirement
- The team has deep Atlas expertise and prefers consolidating Mongo-of-record on Atlas across clouds
- The latest Mongo version's features are needed and Cosmos vCore's lag is unacceptable

**Don't pick Atlas just because:**

- "It's real Mongo" — if you're not using features that gap, Cosmos vCore is real-enough Mongo
- "We've always used Atlas elsewhere" — that's not a justification on its own; cost it out
- "It scales better" — Cosmos vCore scales to large clusters; the limit is unlikely to bite

## Decision tree

```
Do you actually need Mongo semantics (aggregation pipeline, transactions across collections, change streams)?
├── No → Cosmos DB NoSQL/SQL API (see cosmos-db-design.md)
└── Yes → 
    Do you need Atlas Search, Atlas Vector Search, or multi-cloud?
    ├── Yes → MongoDB Atlas on Azure
    └── No →
        Is this an existing Cosmos for Mongo RU deployment?
        ├── Yes → stay on RU until a clear reason to migrate
        └── No → Cosmos DB for MongoDB vCore
```

## Networking and security

For all three options:

- Private Link / private endpoint, no public access
- Entra ID authentication where supported (vCore yes; RU-based limited; Atlas via federated IDP)
- Disable SCRAM password auth in prod where possible
- TLS 1.2 minimum, prefer 1.3
- Connection string in Key Vault, retrieved via Managed Identity at startup

Atlas-specific: configure VNet peering between Azure VNet and Atlas's project network. Document the peering in the network architecture diagram. Atlas IPs in the allow-list must be private (post-peering); never allow `0.0.0.0/0`.

## Backups

- **Cosmos vCore**: continuous backup with PITR; backups managed by Azure.
- **Cosmos RU**: same backup model as Cosmos NoSQL API.
- **Atlas**: Atlas-managed snapshots, with policy-driven retention and PITR (Atlas Cloud Backup).

Test restore quarterly for whichever you choose. Untested backups are not backups.

## Observability

- **vCore / RU**: Azure Monitor metrics, diagnostic logs to Log Analytics, integrate with Grafana.
- **Atlas**: Atlas's own monitoring; export metrics to Azure Monitor via integration if you want unified dashboards.

For SOC 2 evidence (`soc2-iso27001-controls-mapping`): vCore / RU logs land in Log Analytics natively; Atlas requires explicit export + retention configuration on the Atlas side.

## Cost shape

Order-of-magnitude (verify with current Azure pricing):

| Workload | Cosmos vCore | Cosmos RU | Atlas |
|---|---|---|---|
| Dev / small | Free tier (32 vCore-hr/mo) | Serverless ~$25/mo | M10 dedicated ~$57/mo |
| Mid (M40 / 4K RU) | ~$700/mo | ~$200–500/mo depending on traffic | ~$900/mo |
| Large (M80 / 40K RU) | ~$4K/mo | ~$2K–6K/mo | ~$5K/mo |

RU pricing wins on low-traffic or bursty workloads; vCore wins on steady workloads with predictable cluster size; Atlas is typically 20–30% more than equivalent Cosmos vCore.

See `azure-microservices-cost-review` for proper FinOps comparison.

## Anti-patterns

- **Picking Mongo because the team is comfortable with Mongo, when the access pattern is document-with-bounded-queries.** Cosmos NoSQL is cheaper and more Azure-native.
- **Picking Atlas because "it's real Mongo" without naming the gap feature.** Audit which Mongo features actually fail on Cosmos vCore before paying the Atlas premium.
- **RU-based Cosmos for Mongo new work.** It's the older path; vCore is the modern default. Migrating off RU later is painful.
- **No version-compatibility check.** Mongo version on Cosmos vCore lags. Verify your driver and feature set work on the available version.
- **Public network access on Atlas / Cosmos for "easier dev."** Use Private Link from day 1; the wrong default sticks around.

## Verification questions

1. Has the Mongo-specific feature requirement (aggregation pipeline, transactions, change streams, Atlas Search) been named, or is "Mongo" just a habit?
2. If choosing vCore: has the Cosmos for Mongo version compatibility been verified against the application's driver and queries?
3. If choosing Atlas: which Atlas-specific feature justifies the vendor split?
4. Is Private Link / peering configured, with public network access disabled?
5. Is backup PITR enabled and has restore been tested?
6. For prod: is auth via Entra ID / federated IDP, not stored passwords?

## What to read next

- `engine-selection.md` — Mongo vs Postgres vs Cosmos NoSQL choice upstream of this
- `cosmos-db-design.md` — if you ended up at Cosmos NoSQL after all
- `data-migration-patterns.md` — moving from one Mongo flavour to another
- `azure-microservices-security` skill — Private Link, Entra auth
- `azure-microservices-cost-review` skill — vCore vs RU vs Atlas cost comparison
