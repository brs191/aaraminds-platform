# Compute Tier Cost Analysis — Container Apps vs AKS vs App Service

## When to use this reference

When sizing the compute platform for a new service, or when reviewing why compute spend on an existing estate looks wrong for the workload profile. Use this when the question is "are we on the wrong runtime?" — not "what runtime should we use functionally" (that's `azure-service-mapping`). The three Azure compute platforms have radically different bill shapes for the same workload; picking wrong costs 2–5× more than necessary at any non-trivial scale.

## The three platforms in one paragraph

**Container Apps** charges per vCPU-second and GB-second consumed, scales to zero on idle, and adds a ~$0 platform fee. It is the cheapest platform for spiky and bursty workloads and the *most expensive* platform per vCPU-hour at constant high utilization. **AKS** charges for the underlying VMs in the node pool (whether your pods use them or not) plus a flat ~$73/month cluster management fee per cluster (free tier available, Standard tier required for prod SLA). It is the cheapest platform per vCPU-hour at sustained high utilization and the most expensive at low utilization. **App Service** charges per App Service Plan instance (a VM rented continuously), with the plan being the unit of cost regardless of how many apps run on it; it sits between the two on per-vCPU price but has the worst scale-to-zero story.

## Per-platform cost shape

### Container Apps (Consumption + Dedicated workload profiles)

Consumption profile pricing (the default — verify current numbers on `azure.microsoft.com/pricing/details/container-apps/`):

| Resource | Price | Notes |
|---|---|---|
| vCPU-second | ~$0.000024 (~$0.086/vCPU-hour) | First 180,000 vCPU-seconds/month free per subscription |
| Memory GB-second | ~$0.000003 (~$0.011/GB-hour) | First 360,000 GB-seconds/month free per subscription |
| Requests | ~$0.40 per million | First 2M/month free |

Dedicated workload profiles (D-series, E-series) charge per VM-hour like AKS — used when you need GPU, large memory, or sustained high CPU.

Concrete shape for a single service running 0.5 vCPU + 1 GB, 2 replicas, 24/7:
- vCPU: 2 × 0.5 × 730 h = 730 vCPU-hours × $0.086 = **$62.78**
- Memory: 2 × 1 × 730 h = 1,460 GB-hours × $0.011 = **$16.06**
- Total: **~$79/month per service** before egress and free-tier credits

Same service scaled to zero 16 h/day (background worker active 8 h/day):
- Active hours: 8 × 30 = 240 h
- vCPU: 2 × 0.5 × 240 = 240 vCPU-hours × $0.086 = **$20.64**
- Memory: 2 × 1 × 240 = 480 GB-hours × $0.011 = **$5.28**
- Total: **~$26/month** — 67% savings, the entire reason scale-to-zero exists

### AKS

Node pool VM cost (Standard_D4s_v5 example):
- ~$0.192/hour × 730 h = **~$140/month per node**
- Cluster management fee (Standard tier, prod SLA): ~$0.10/hour ≈ **$73/month per cluster**

A 3-node D4s_v5 pool: $140 × 3 + $73 = **~$493/month minimum**, regardless of pod count or pod resource requests. You pay for the VMs whether they run 1 pod or 50.

Per-vCPU sustained cost: $0.192 / 4 vCPU = **~$0.048/vCPU-hour** — about 56% of Container Apps consumption pricing. This is the inflection point: above ~50% sustained node utilization, AKS beats Container Apps; below that, Container Apps wins.

### App Service

Premium v3 P1v3 plan (2 vCPU, 8 GB):
- ~$0.20/hour × 730 = **~$146/month per instance**
- Plan hosts multiple apps for free — a 10-app plan still costs $146 if it fits

Per-vCPU sustained: ~$0.10/vCPU-hour. Roughly midway between Container Apps and AKS. App Service is rarely the right answer for a microservices fleet — its sweet spot is a small number of medium-sized stateful web apps, not 30 ephemeral services.

## Decision table — pick the platform from the workload profile

| Workload profile | Pick | Why |
|---|---|---|
| <10 services, spiky traffic, dev/test included | Container Apps Consumption | Scale-to-zero recovers idle hours; no cluster fee; free tier covers most dev |
| 10–30 services, mixed steady + spiky | Container Apps (Consumption + Dedicated profile for the steady ones) | One platform, two cost shapes; avoid the AKS operational tax until you actually need it |
| 30+ services, mostly steady utilization >50%, in-house K8s expertise | AKS | Per-vCPU price wins at scale; you absorb the node management cost |
| Need GPUs, DaemonSets, complex networking, service mesh (Istio/Linkerd), Windows containers | AKS | Container Apps cannot do these |
| Legacy ASP.NET / Java web apps, sticky sessions, easy Easy Auth integration | App Service | Lift-and-shift target; do not refactor to Container Apps just for this |
| Background jobs <1 hour, event-driven, low frequency | Azure Functions (Consumption plan), not any of the above | Per-execution pricing beats per-second compute |

## Scale-to-zero — what each platform actually delivers

Scale-to-zero is the single highest-leverage cost optimization for non-customer-facing services. The platforms differ:

| Platform | Scale-to-zero | Cold start | When it works |
|---|---|---|---|
| Container Apps (Consumption) | Yes, native; `min_replicas = 0` | ~2–10 seconds first request | HTTP-triggered, queue-triggered, or scheduled workers |
| Container Apps (Dedicated) | No — billed per VM | n/a | Always-on tier |
| AKS | Only with KEDA + cluster autoscaler down to zero nodes — operationally complex | 30–120 seconds (node provisioning) | Possible but rarely worth the complexity |
| App Service | "Always Ready" requires plan running; can stop a plan but loses warm state | 5–30 seconds | Not a real scale-to-zero story |
| Functions (Consumption) | Yes, native | ~1–5 seconds | Best for true event-driven; not a microservice runtime |

**Rule**: if a service can tolerate 2–10s cold-start latency, run it on Container Apps with `min_replicas = 0`. Background workers, internal admin tools, scheduled jobs, and any service queried <1× per minute should default to zero minimum.

For a customer-facing API with strict p95 latency, set `min_replicas = 1` (or higher) and pay the full hour. Cold-start risk on the user path is not a place to save $30/month.

## Typical bill drivers — what eats compute spend

In rough order of frequency, when reviewing a Container Apps + AKS bill:

1. **`min_replicas` set defensively too high.** Every service set to `min_replicas = 2 or 3 "for HA"` even when traffic doesn't need it. Most internal services can run at 1 replica with the platform's own auto-restart as the recovery mechanism. Check `min_replicas` across the fleet; anything > 1 needs justification.

2. **Over-provisioned vCPU.** Services requested 1.0 or 2.0 vCPU at creation, never resized. Pull 7-day p95 vCPU from Azure Monitor; if p95 is < 30% of requested, halve the request. Container Apps re-creates the revision, so the cost shift is immediate.

3. **AKS node pools sized for peak with no autoscaler.** Three D8s_v5 nodes running at 15% sustained utilization. Either enable the cluster autoscaler with a low min, switch to smaller VM SKUs, or migrate the underutilized workloads to Container Apps Consumption.

4. **Dev/test environments running 24/7.** Non-prod estates on App Service Premium plans or AKS nodes that nobody stops at 6pm. Apply autostart/autostop via Azure Automation, or shift dev to Container Apps Consumption where the bill is zero on idle.

5. **Container Apps Dedicated profile selected by default.** Someone read the docs, saw "for predictable workloads," and put everything on Dedicated. Move steady workloads to Dedicated only if Consumption spend exceeds ~$200/service/month — the breakeven on D4 dedicated.

6. **Egress.** Cross-region or cross-cloud egress at $0.05–0.09/GB adds up fast. Colocate dependent services. A 10 TB/month chatty pair of services across regions is $500–900 in egress alone.

## When each platform is *cheaper* — concrete examples

**Container Apps wins**: 8 microservices, average 0.25 vCPU sustained each, traffic concentrated in 10 business hours/day. Container Apps with scale-to-zero on 4 of them and min_replicas=1 on 4 of them: ~$180/month. AKS equivalent: minimum 2-node D4 pool for HA = ~$280/month + cluster fee = ~$353/month. Container Apps is ~50% cheaper.

**AKS wins**: 25 services, average 1.5 vCPU sustained, traffic flat 24/7, all services interdependent. AKS with a 5-node D8s_v5 autoscaling pool: ~$700/month + $73 = ~$773. Container Apps Consumption equivalent: 25 × 1.5 × 730 = 27,375 vCPU-hours × $0.086 = **~$2,354** plus memory. AKS is ~3× cheaper at this scale.

**App Service wins**: legacy Spring Boot monolith + 3 small admin apps that share auth and session state. One P1v3 plan hosting all 4: $146/month. Container Apps equivalent (4 separate apps, each min_replicas=1, 1 vCPU + 2 GB): ~$390/month. App Service is ~2.7× cheaper, *and* you keep Easy Auth + slot swap for free.

## Brownfield migration angles

When the existing platform is wrong for the workload, the move is rarely free:

- **AKS → Container Apps**: usually a win for fewer than ~15 services. Cost: rewrite Helm charts to Container Apps YAML/Terraform, lose DaemonSets and CRDs, re-do ingress (Container Apps uses Envoy under the hood, not nginx). Plan 2–4 weeks per service for a clean cut. Run both in parallel for one billing cycle; cut over when the new bill is observed.
- **App Service → Container Apps**: viable when you've outgrown the plan model (10+ apps cohabiting and noisy-neighbor issues appearing). Watch out for: WebJobs (not supported on Container Apps — migrate to a separate worker), Easy Auth (replace with App Gateway + Entra), slot-swap deploys (replace with Container Apps revisions).
- **Container Apps → AKS**: only when you hit a platform limit (need a sidecar mesh, GPU, custom CNI, Windows containers, or sustained >50 services at high utilization). Once on AKS, you own node patching, version upgrades, and CNI debugging — the operational cost is real and recurring.

Do not migrate "for cost reasons" without a 30-day side-by-side cost comparison on a representative subset of services. The headline per-vCPU price hides the operational tax.

## Anti-patterns

- **Picking AKS because "we're a serious shop"**. AKS is a serious operational commitment, not a signal of seriousness. If your fleet is 8 services and one engineer, you are paying both the cluster fee and the engineering opportunity cost for prestige.
- **Quoting Container Apps Consumption pricing to estimate a 24/7 always-on fleet bill.** At sustained load, Consumption looks expensive on paper. Either commit to the workload reality (use Dedicated profile or AKS), or accept the Consumption tax in exchange for elasticity.
- **`min_replicas = 1` "for warm starts" on every internal service**. Most internal services have no SLA that justifies it. The first request of the day taking 4 seconds is fine; pay the $20/month back.
- **Counting only compute when comparing platforms**. AKS adds node patching, version-upgrade testing, cluster security baselines, CNI quirks. Container Apps adds ingress-feature gaps and revision-management quirks. Both have a non-zero human cost that should appear in the comparison.

## What this is not

This reference is platform-cost comparison only. For *functional* selection between Container Apps, AKS, App Service, and Functions (when each is the right tool regardless of cost), see `azure-service-mapping`. For reserved-capacity commitments that change all of these per-vCPU prices, see `reserved-instances-and-savings-plans.md`. For detecting idle compute resources within a chosen platform, see `idle-resource-detection.md`. For per-service cost formulas without the cross-platform decision lens, see `cost-and-tradeoffs.md`.
