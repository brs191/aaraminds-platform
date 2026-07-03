# Idle Resource Detection — Patterns, Queries, Remediation

## When to use this reference

When the bill is rising without a corresponding traffic increase, when you suspect resources are "renting space" without doing work, or when leadership asks for a quick-win cost-cut pass before the larger architectural optimizations land. Idle resources are the cheapest savings to capture — no architectural risk, no commitment lock-in, just delete or downsize what isn't earning its keep.

## The detection mindset

Idle is rarely a flag in the portal. It's a pattern across telemetry: zero or near-zero work signals for sustained time, while the resource keeps billing. The four signal categories:

1. **Connection-count signals** — databases, caches, queues with no clients connected.
2. **Throughput signals** — services, queues, ingress endpoints with no requests/messages.
3. **Utilization signals** — vCPU, RU, IOPS, memory sitting far below allocated.
4. **Access-pattern signals** — storage tiers holding data that hasn't been read in months.

Each needs a different KQL query and a different remediation. The patterns below are the eight that recover the most spend in practice.

## Pattern 1 — Zero-connection databases

**Symptom**: an Azure SQL DB, Postgres Flexible Server, or Cosmos DB account billing every hour with no connected clients for days or weeks. Typical cause: a service was retired, migrated, or never went to production, but the database survived.

**KQL detection (Postgres Flexible Server)**:

```kql
AzureMetrics
| where TimeGenerated > ago(14d)
| where ResourceProvider == "MICROSOFT.DBFORPOSTGRESQL"
| where MetricName == "active_connections"
| summarize maxConn = max(Average) by Resource, ResourceId
| where maxConn < 1
| project Resource, ResourceId, maxConn
```

**KQL detection (Azure SQL)**:

```kql
AzureMetrics
| where TimeGenerated > ago(14d)
| where ResourceProvider == "MICROSOFT.SQL"
| where MetricName == "connection_successful"
| summarize totalConn = sum(Total) by Resource, ResourceId
| where totalConn == 0
```

**Remediation playbook**:
1. Cross-check with the deployment inventory — is there any service that *should* connect to this DB?
2. If owner can be identified, send a one-week deletion notice with the evidence.
3. If unclaimed after 7 days, snapshot/export, then delete.
4. For Azure SQL: consider **Auto-pause** (Serverless tier) instead of deletion when the DB might be needed sporadically — pause-on-idle for ~1 hour billing 0 vCore-seconds.
5. For Postgres Flexible Server: there is no auto-pause; stop the server manually via `az postgres flexible-server stop` (still bills for storage, no compute).

**Typical recovery**: $30–500/month per stranded DB. Multiple stranded DBs are common in estates >2 years old.

## Pattern 2 — Scaled-up services running at <5% utilization

**Symptom**: a Container App or AKS workload provisioned at 2 vCPU / 4 GB, p95 vCPU utilization sits at 0.05–0.2 vCPU for weeks. Common cause: initial sizing was a guess, never revisited.

**KQL detection (Container Apps)**:

```kql
ContainerAppConsoleLogs_CL
| where TimeGenerated > ago(7d)
| join kind=inner (
    AzureMetrics
    | where ResourceProvider == "MICROSOFT.APP"
    | where MetricName == "UsageNanoCores"
    | summarize p95Cpu = percentile(Average, 95) by Resource
) on Resource
| where p95Cpu < 50000000   // 0.05 vCPU
| project Resource, p95Cpu
```

Or simpler via Azure Monitor Workbook on the `UsageNanoCores` and `WorkingSetBytes` metrics across all Container Apps revisions, sorted ascending.

**Remediation playbook**:
1. Pull 14 days of p95 vCPU and p95 memory per revision.
2. Set new requests to `p95 × 2` (gives 2× headroom; halve again if even that proves over-sized).
3. For Container Apps: update `resources.cpu` and `resources.memory` via Terraform, deploy a new revision. Cost shift is immediate at the next revision.
4. For AKS: update Deployment resource requests; if the node pool becomes over-sized after the shift, downsize the pool or reduce node count.
5. Re-measure after 1 week. If p95 climbs near the new ceiling, scale back up.

**Typical recovery**: 40–70% of the per-service compute cost. A fleet of 20 over-sized services often recovers $500–2,000/month in aggregate.

## Pattern 3 — Hot-tier blob storage holding cold data

**Symptom**: a Storage Account on Hot tier holding GBs or TBs of blobs that haven't been read in 90+ days. Hot tier is ~$0.018/GB/month; Cool is ~$0.01; Cold (newer tier) is ~$0.0036; Archive is ~$0.00099. The price gap on rarely-accessed data is large.

**KQL detection** (requires Storage diagnostic logging to Log Analytics):

```kql
StorageBlobLogs
| where TimeGenerated > ago(90d)
| where OperationName in ("GetBlob", "GetBlobProperties")
| summarize lastRead = max(TimeGenerated) by AccountName, ContainerName, Uri
| where lastRead < ago(60d)
| project AccountName, ContainerName, Uri, lastRead
```

For accounts without blob logging, use **Azure Storage Lifecycle Management** rules directly — they evaluate access tracking metadata even retroactively.

**Remediation playbook**:
1. Enable blob access tracking on the storage account (no extra cost; required for lifecycle rules to use last-access time).
2. Define lifecycle policy: after 30 days no access → Cool, after 90 days → Cold, after 365 days → Archive (or delete if the data has a defined retention).
3. Apply via Terraform `azurerm_storage_management_policy`.
4. Watch for **re-hydration cost** — Archive blobs cost $0.02/GB to read back; ensure data classified as Archive truly doesn't need fast access.
5. For data that should never be archived (e.g., compliance evidence requiring fast retrieval), exclude its container from the policy explicitly.

**Typical recovery**: 60–90% on the storage line item for backups, build artifacts, log dumps, and old analytics outputs.

## Pattern 4 — Cosmos DB containers at <30% RU consumption

**Symptom**: a Cosmos DB container provisioned at 10,000 RU/s (manual or autoscale max), actual p95 consumption sits at 2,500 RU/s. The 7,500 RU/s headroom is paid every hour.

**KQL detection**:

```kql
AzureMetrics
| where TimeGenerated > ago(14d)
| where ResourceProvider == "MICROSOFT.DOCUMENTDB"
| where MetricName == "TotalRequestUnits"
| extend rps = Total / 60.0  // per-second from per-minute aggregation
| summarize p95rps = percentile(rps, 95) by Resource
| join kind=inner (
    AzureMetrics
    | where MetricName == "ProvisionedThroughput"
    | summarize provisioned = max(Maximum) by Resource
) on Resource
| extend utilization = p95rps / provisioned
| where utilization < 0.3
| project Resource, p95rps, provisioned, utilization
```

**Remediation playbook**:
1. Compute `p95 × 1.5` as the new ceiling (gives 50% spike headroom).
2. If currently on Manual provisioning → switch to Autoscale; the autoscale floor is 10% of max, so utilization-based right-sizing is automatic at the floor.
3. If already on Autoscale → lower `max_throughput`. Cosmos charges for autoscale max × 10% as the minimum bill; lowering max directly lowers the floor.
4. For containers with <5K RU/s peak and <50 GB → evaluate **Serverless tier**. No baseline cost; pay only per request. The cutover is a container recreation, not a config change.
5. Verify after 7 days that throttled requests (`429` responses) stay near zero.

**Typical recovery**: 30–60% on Cosmos line items. The pattern compounds because most teams over-provision Cosmos at creation "for safety."

## Pattern 5 — Service Bus / Event Hubs with no message flow

**Symptom**: a Service Bus namespace on Premium tier ($550/month minimum) or an Event Hubs namespace with allocated Throughput Units, but message ingestion and egress are flat at 0 for weeks.

**KQL detection (Service Bus)**:

```kql
AzureMetrics
| where TimeGenerated > ago(14d)
| where ResourceProvider == "MICROSOFT.SERVICEBUS"
| where MetricName in ("IncomingMessages", "OutgoingMessages")
| summarize totalMessages = sum(Total) by Resource
| where totalMessages < 100   // <100 messages in 14 days = effectively idle
```

**Remediation playbook**:
1. Identify owning service and topic/queue purpose.
2. For Premium tier Service Bus with no traffic → downgrade to Standard (Standard is $10/month + per-message; Premium is $550/month minimum). Premium-only features (large messages, JetStream, dedicated capacity) need to be confirmed unneeded.
3. For Event Hubs with no traffic → reduce Throughput Units to 1, or delete the namespace if no consumer exists.
4. Watch for "namespace per environment" sprawl — dev/test/staging namespaces inheriting Premium tier from prod IaC templates.

**Typical recovery**: $500–1,500/month per stranded Premium namespace.

## Pattern 6 — Public IPs, NAT Gateways, App Gateways without traffic

**Symptom**: Public IP addresses, NAT Gateways ($45/month + per-GB processed), or Application Gateways ($125–250/month base) provisioned for services that were retired or migrated, still billing every hour.

**KQL detection (Public IPs with zero bytes)**:

```kql
AzureMetrics
| where TimeGenerated > ago(14d)
| where ResourceProvider == "MICROSOFT.NETWORK"
| where ResourceType == "PUBLICIPADDRESSES"
| where MetricName == "ByteCount"
| summarize totalBytes = sum(Total) by Resource
| where totalBytes < 10000   // <10 KB over 14 days = orphaned
```

**Remediation playbook**:
1. Check association: `az network public-ip show --query ipConfiguration`. Unassociated standard-SKU IPs bill ~$3.65/month each — small per-IP but a fleet of 50 orphans is $180/month for nothing.
2. For App Gateways: confirm backend pool is empty or all-unhealthy; if so, delete.
3. NAT Gateways: detach and delete if subnet no longer needs outbound NAT (or shares one with another subnet).
4. Track via Azure Resource Graph queries weekly to catch new orphans within days.

## Pattern 7 — Log Analytics ingestion outpacing value

**Symptom**: Log Analytics workspace ingesting 50–500 GB/day at $2.30–2.76/GB ($3,500–35,000/month) with most of the data never queried. Common cause: a service set logging level to DEBUG in prod, or a noisy library logging every request.

**KQL detection (top tables by ingestion volume)**:

```kql
Usage
| where TimeGenerated > ago(7d)
| where IsBillable == true
| summarize ingestedGB = sum(Quantity) / 1024 by DataType
| order by ingestedGB desc
| take 10
```

Then for the top offenders, sample the actual log content:

```kql
<TopDataType>
| where TimeGenerated > ago(1h)
| take 100
```

**Remediation playbook**:
1. For services logging at DEBUG in prod → drop to INFO. The change is immediate.
2. For chatty libraries (e.g., HTTP client logging every request/response body) → add structured filters or move to sampling.
3. Apply **Log Analytics Workspace transformation rules** (`workspaceTransform` KQL pipeline) to drop low-value rows at ingest — billed only for what survives the transform.
4. Use **Basic Logs** tier (~$0.65/GB ingested) for high-volume audit-only data that's rarely queried; trade-off: 8-day retention, limited KQL features.
5. For audit logs needing long retention → ship to a Storage account at $0.018/GB instead of paying Log Analytics retention fees ($0.10–0.12/GB-month after free 31 days).
6. Set **daily cap** on the workspace as a hard guard against runaway ingestion.

**Typical recovery**: 50–80% on observability spend. A common one — fixing one DEBUG-in-prod service recovers $1,000+/month overnight.

## Pattern 8 — Dev/test environments running 24/7

**Symptom**: non-prod resource groups containing Container Apps, AKS node pools, or App Service Plans that bill the same amount on Saturday night as Tuesday morning. Engineers don't work 168 h/week; the bill shouldn't either.

**Detection**: tag-based query across cost data. Filter to `Environment in ('dev', 'test', 'staging')` resource groups; show 24-hour cost distribution. If it's flat, the environment is running idle 14–16 h/day.

**Remediation playbook**:
1. **AKS dev clusters** — use **Cluster Stop/Start** via `az aks stop`; bills only for storage and load balancers while stopped. Wire to an Azure Automation runbook on schedule.
2. **App Service Plans** — `az appservice plan update --sku F1` is not the answer (functional changes); instead, schedule `az webapp stop` / `start` on the apps.
3. **Container Apps** — set `min_replicas = 0` on every non-prod app. Scale-to-zero is the right tool here.
4. **VMs (rare in microservices estates)** — Auto-shutdown via DevTest Labs or Azure Automation.
5. Target: dev environments run 50–60 h/week (10 h/day, 5 days), not 168. Savings: 60–70% on dev compute.

## The detection rhythm

Run a **monthly idle sweep** on the first Monday:
1. Execute KQL queries for patterns 1, 4, 5, 6, 7 against the production subscription.
2. Pull a tag-grouped cost report; flag any resource over $50/month without an owner tag.
3. Generate a "candidates for cleanup" table; route each row to the owning team with a 2-week deletion notice.
4. Track recovered $/month in a running tally — this is the FinOps function's portfolio.

Run a **quarterly utilization right-sizing** on the second Monday of each quarter:
1. Execute pattern 2 (compute) and pattern 4 (Cosmos) detection across the estate.
2. Propose resize PRs in the Terraform repo for the top 10 offenders.
3. Measure savings 30 days post-deploy.

## Anti-patterns

- **"Looks busy in the portal" as the proof of life.** Activity indicators in the Azure portal often show provisioning events or auto-management traffic, not actual application use. Trust the metrics, not the spinner.
- **Deleting without snapshot.** A "stranded" database might be the source for a quarterly report nobody ran in 90 days. Snapshot first, delete second. Storage for the snapshot is pennies vs. data loss.
- **Resizing in a single jump from 4 vCPU to 0.25 vCPU.** Halve, observe a week, halve again. Catastrophic resize causes outages, which gets the entire cost program shut down.
- **Auto-pause without warning the app team.** Azure SQL Serverless auto-pause causes a ~30 second cold-start on next connection. If the app doesn't retry, the user sees an error. Communicate before enabling.
- **Letting orphans accumulate because each one is "only $30."** Twenty orphans is $600/month. The aggregate is real; sweep regularly.

## What this is not

This reference is detection and remediation of *underutilized or stranded* resources. For deciding the right *platform* in the first place (which determines what counts as "right-sized"), see `compute-tier-cost-analysis.md`. For deciding whether to *commit* to identified steady-state usage via RIs or Savings Plans, see `reserved-instances-and-savings-plans.md`. For data-tier-specific sizing (which is one input to pattern 1 and pattern 4 here), see `data-tier-cost-optimization.md`. For the general cost framework and per-service formulas, see `cost-and-tradeoffs.md`.
