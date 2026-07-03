# Architecture Review Framework — 9 Dimensions

This is the operational depth behind the 9-dimension table in `SKILL.md`. Each dimension lists what to inspect, the detection cues that distinguish pass / soft-fail / hard-fail, the typical remediation patterns, and the source skill that owns the design-time criteria.

Walk the dimensions in order. Earlier dimensions constrain later ones — a wrong service boundary (dimension 1) makes data architecture (dimension 2) and observability (dimension 7) findings downstream. If dimension 1 hard-fails, note it and continue walking; do not collapse later findings into the structural one.

## Severity definitions

- **Pass** — the dimension meets the bar. Brief affirming note in the report so the reader knows it was examined.
- **Soft-fail** — defect exists but does not threaten the system's near-term goals. Tracked follow-up with owner and target quarter. Does not block the verdict.
- **Hard-fail** — load-bearing flaw materially threatening reliability, security, compliance, cost, or the team's ability to operate the system. Blocks "Healthy" verdict. Each hard-fail requires a named defect, named owner, and a named smallest-viable-fix.

## Dimension 1 — Domain & service boundaries

**Inspect:** the service inventory, the bounded contexts they map to, the team-to-service ownership table, the data ownership claims per service, and any shared schemas or shared mutable storage across services.

**Pass cues:**
- Each service maps to a business capability (Sales, Fulfillment, Accounting), not a technical layer (UserService, ProductService).
- One team owns each service end-to-end; no service requires changes coordinated across teams for a normal feature.
- Each service owns its data; cross-service access is via API or event, never via direct DB.

**Soft-fail cues:**
- Two services in different bounded contexts share an authentication library but nothing else — fine, but document it.
- A service is co-owned by two teams during a transition window with a named end date — fine if dated.

**Hard-fail cues:**
- Functional decomposition (UserService, ProductService, OrderService) instead of capability decomposition. Detection: look at the service names and ask "what business question does this answer?" — if the answer is "it stores users," the decomposition is wrong.
- Shared mutable database across services. Detection: search Terraform / Helm for two services pointing at the same Postgres / Cosmos instance with write permissions.
- A service has no clear owning team, or its owning team also owns 4+ other services.

**Remediation patterns:**
- Wrong decomposition: identify the correct bounded contexts using DDD; plan strangler-fig extraction of the right service from the wrong one; explicitly accept the current shape until extraction lands.
- Shared DB: introduce a service API in front of the shared schema; migrate writes through the API first, reads later, drop the direct DB grant last.
- Ownership gap: assign a team or merge the orphan service into the most adjacent service.

**Source skill:** `microservices-architecture-design` — `references/domain-decomposition.md`, `references/service-boundaries.md`.

## Dimension 2 — Data architecture & consistency

**Inspect:** the cross-service transaction map, the consistency model claimed for each, the compensation paths, the idempotency story, the engine choice per service, partition design for Cosmos / Mongo, and the migration / DR plan.

**Pass cues:**
- Every cross-service operation that changes state in two services is named in an ADR with its consistency model (eventual, saga, outbox) and compensation path.
- State-changing endpoints accept and enforce an idempotency key.
- Engine choice fits the workload (transactional OLTP on Postgres or Cosmos with sound partition keys; analytical on Synapse/Fabric; cache on Redis).

**Soft-fail cues:**
- Outbox is in place but the relay process lacks a dead-letter behavior — track and add.
- One service uses Mongo where Postgres would be marginally better but the migration cost outweighs the gain — note the choice, accept it.

**Hard-fail cues:**
- Distributed two-phase commit across service-owned databases. Detection: search for XA, JTA distributed transaction managers, or "we just both commit, it usually works."
- Cross-service joins via direct DB grants instead of API or event.
- No idempotency on state-changing endpoints — replays produce duplicate state.
- Cosmos used for a transactional OLTP workload without a partition-key design (single hot partition, throughput throttling).

**Remediation patterns:**
- 2PC: replace with saga (orchestration or choreography) per `microservices-data-architecture/references/patterns/saga-*.md`.
- Cross-service joins: introduce CQRS read model per `microservices-data-architecture/references/patterns/cqrs.md`.
- Idempotency gap: add idempotency key column and uniqueness constraint at write time; client supplies the key.
- Cosmos hot partition: redesign partition key based on access pattern; back-migrate via change feed.

**Source skills:** `microservices-data-architecture`, `azure-data-tier-design`.

## Dimension 3 — Communication topology

**Inspect:** the service-to-service call graph, sync vs. async per edge, the broker selection per async edge (Service Bus / Event Grid / Event Hubs), ordering and delivery guarantees, dead-letter handling, and trace propagation across async boundaries.

**Pass cues:**
- Async chosen wherever the caller does not need a response synchronously and the operation is durable.
- Service Bus for ordered command flows; Event Grid for fan-out notifications; Event Hubs for high-volume ingest.
- DLQs configured on every queue / subscription; DLQ depth alerted.

**Soft-fail cues:**
- An async path uses Service Bus where Event Grid would be cheaper given the volume — track for cost optimization.
- Trace context propagates across most boundaries but one consumer pulls without restoring the parent span — fix.

**Hard-fail cues:**
- Synchronous chains of 4+ hops. Detection: trace a typical request through the call graph and count hops. Each hop multiplies latency and failure probability.
- No DLQ on a critical async path. Detection: poison messages will hang the consumer or get retried indefinitely.
- Sync-only for an operation that has a long-running dependency (e.g., a vendor with multi-second tail latency) — the caller's thread pool is one bad vendor away from exhaustion.

**Remediation patterns:**
- Long sync chain: identify the longest hops without strong consistency needs and move them to async via outbox + broker.
- Missing DLQ: enable on the Service Bus subscription / queue; add a metric and alert on DLQ depth.
- Vendor-dependent sync: introduce an internal async boundary in front of the vendor (worker pulls from Service Bus, calls the vendor, posts result event).

**Source skill:** `microservices-async-messaging`.

## Dimension 4 — API contracts

**Inspect:** the OpenAPI / gRPC contracts per service, the versioning strategy, error envelope consistency, pagination and idempotency conventions, the API gateway placement, and any BFF layers.

**Pass cues:**
- Every public service publishes an OpenAPI (REST) or `.proto` (gRPC) contract that the consumers depend on.
- Versioning is explicit (URI version or header) and the policy for breaking change is named (deprecate, parallel-run, sunset).
- Error envelopes are consistent across services (problem-details RFC 7807 or an equivalent).
- API Management or Front Door fronts external traffic; service-to-service is internal-only.

**Soft-fail cues:**
- One service publishes OpenAPI but it has drifted from implementation — schedule a regeneration.
- BFF exists for the mobile client but not for the web client where one would help — track.

**Hard-fail cues:**
- Breaking changes shipped without version bump — consumers break on deploy.
- Inconsistent error envelopes across services — clients have to handle N error shapes.
- External traffic hits services directly without gateway — no rate limiting, no centralized auth, no WAF.

**Remediation patterns:**
- Versioning gap: introduce a versioning convention (URI or header) and a deprecation policy; add a CI check that fails on schema changes without version bump.
- Error envelope drift: standardize on RFC 7807; provide a shared library; soft-deprecate the old shapes.
- Missing gateway: front the external endpoints with API Management; phase migration via DNS.

**Source skill:** `microservices-api-design`.

## Dimension 5 — Resilience

**Inspect:** for every outbound call (HTTP, gRPC, broker, DB), confirm the timeout, retry policy with jitter, circuit breaker, and bulkhead. For every rollout, confirm the strategy (blue-green, canary, strangler-fig) and the rollback path.

**Pass cues:**
- Every outbound HTTP / gRPC call has an explicit timeout (typically 1–5 s), retry with jitter (capped at 3 attempts), and a circuit breaker.
- Bulkheads (separate thread pools or connection pools) where shared resources back multiple flows.
- Rollouts use a named strategy; rollback is tested at least quarterly.

**Soft-fail cues:**
- Timeouts present but uniform across all calls (3 s everywhere) without per-dependency tuning — refine.
- Canary exists but the abort criteria are informal — codify.

**Hard-fail cues:**
- Unbounded retries — a transient failure becomes a thundering herd.
- No timeout on outbound calls — one slow dependency exhausts the caller's threads.
- No circuit breaker around a vendor or unreliable internal dependency — one bad day for the dependency takes down the caller.
- Big-bang deploy with no canary or feature flag for a load-bearing service.

**Remediation patterns:**
- Missing timeout: add per-client with measured tuning (P99 of healthy latency × 2, capped at SLO budget).
- Missing breaker: introduce Resilience4j (Java) or `sony/gobreaker` / `failsafe-go` (Go); start with conservative thresholds (50% errors over 20 calls).
- Big-bang rollout: introduce blue-green via Container Apps multiple revisions; feature-flag the routing.

**Source skill:** `microservices-resilience`.

## Dimension 6 — Azure service mapping

**Inspect:** the compute hosting choice per service (Container Apps / AKS / App Service), the data engine choice, the messaging broker, caching, service discovery, and any sidecars. Confirm there is no drift to non-Azure services (AWS-isms, Bicep, Pulumi, Datadog, etc.) unintroduced.

**Pass cues:**
- Container Apps for stateless services unless a specific AKS need is named (sidecars, custom CRDs, specific networking).
- Postgres Flexible / Cosmos / Mongo Atlas chosen against the access pattern, not by default preference.
- Service Bus / Event Grid / Event Hubs chosen by ordering / fan-out / volume per `azure-service-mapping`.
- No cloud drift in the IaC.

**Soft-fail cues:**
- App Service in use for a service that would migrate cleanly to Container Apps for cost — track.
- A managed Redis tier is one size larger than measurement supports — right-size.

**Hard-fail cues:**
- Cosmos chosen for transactional OLTP without partition-key design (overlaps with Dimension 2).
- Cloud drift: AWS SDK in the codebase, Bicep alongside Terraform, Pulumi, GitLab CI for new pipelines.
- Service mesh introduced without operational ownership — Istio with no on-call engineer who knows it.

**Remediation patterns:**
- Service drift: produce an ADR for the existing choice or a migration plan with a target quarter.
- Cloud drift: standardize on the pack stack (Azure-primary, Terraform AzureRM, GitHub Actions OIDC); remove the deviating tooling on the next refactor.

**Source skill:** `azure-service-mapping`.

## Dimension 7 — Observability

**Inspect:** OTel instrumentation coverage, SLO definitions, alert design, dashboard quality, trace propagation across async boundaries, log structure, runbook presence per service.

**Pass cues:**
- Every service emits OTel traces, metrics, and structured logs to the shared Grafana / Prometheus / Tempo stack.
- Each service has 1–3 SLOs that measure user-facing impact (request success, p95 latency, freshness for async).
- Alerts page on SLO burn rate, not raw resource metrics like CPU.
- Traces span async boundaries (broker propagates the trace context).
- Runbook exists per service with first-response steps for each named alert.

**Soft-fail cues:**
- Dashboards exist but the SLO panel is missing or hard to find — fix layout.
- One service has a runbook last updated 14 months ago — refresh.

**Hard-fail cues:**
- No SLO defined for a load-bearing service — the team cannot tell if it is meeting users' needs.
- Alerts fire on CPU or memory thresholds with no user-impact correlation — pages are noisy and miss real impact.
- Tracing gap at a broker boundary — every async hop is a debugging black hole.
- No runbook for a service that pages on-call.

**Remediation patterns:**
- Missing SLO: define from incident history (what user impact has occurred?); pick 2–3; alert on burn rate.
- Alert drift: re-derive alerts from SLOs; remove CPU/memory alerts unless they correlate with a known failure mode.
- Trace gap: enable OTel context propagation in the broker client; verify traces in a smoke test.
- Missing runbook: shortest viable template (what does this service do, what wakes you, what to check first).

**Source skill:** `azure-microservices-observability`.

## Dimension 8 — Security & compliance

**Inspect:** authentication entry, authorization model, service-to-service identity, secret management, network segmentation, audit logging, and the SOC 2 / ISO 27001 control mapping if compliance is in scope.

**Pass cues:**
- External auth via Entra ID; OAuth 2.1 with the correct flow per client.
- Service-to-service identity via Managed Identity or Workload Identity; no shared secrets between services.
- Secrets in Key Vault; accessed via Managed Identity; no plaintext secrets in code, config, or Terraform.
- TLS everywhere; mTLS or token-bound service-to-service per the zero-trust pattern.
- Audit log writes to an append-only sink (Log Analytics → Sentinel).
- SOC 2 / ISO 27001 controls mapped to Azure-native evidence where compliance is in scope.

**Soft-fail cues:**
- One service relies on a long-lived service principal that should be migrated to Managed Identity — track.
- Audit log exists but retention is shorter than the compliance bar — adjust.

**Hard-fail cues:**
- Any plaintext secret in code, config, or Terraform — detection via secret scanning.
- State-changing endpoint without authorization. Detection: each endpoint should map to a scope or role; missing means unauthenticated escalation.
- TLS off on an internal hop — defense-in-depth gap.
- Compliance scope claimed but controls not mapped — audit will fail.

**Remediation patterns:**
- Plaintext secret: rotate the secret, store in Key Vault, wire via Managed Identity, remove from source control history.
- Missing authz: define scopes or roles; enforce at the service layer; add a contract test that rejects unauthorized calls.
- TLS gap: enable end-to-end TLS via Container Apps ingress and service-to-service.
- Compliance gap: invoke `soc2-iso27001-controls-mapping` to produce the control-to-Azure-evidence map.

**Source skills:** `azure-microservices-security`, `soc2-iso27001-controls-mapping`.

## Dimension 9 — Cost & operability

**Inspect:** the monthly bill breakdown, the per-service sizing against measured load, scale-to-zero opportunities, reserved capacity for steady-state, idle resource detection, and the FinOps cadence.

**Pass cues:**
- Sizing is measured (P95 + 20% headroom), not arbitrary.
- Scale-to-zero on low-traffic services where cold start is acceptable.
- Reserved capacity on services with 24×7 baseline.
- Idle resources flagged and decommissioned on a quarterly cadence.
- Monthly bill is reviewed by a named FinOps owner.

**Soft-fail cues:**
- One service over-provisioned by ~30% — right-size.
- Reserved capacity not yet purchased for a service eligible for it — schedule.

**Hard-fail cues:**
- Over-provisioned by ≥2× without justification — measurable waste.
- Pay-as-you-go for a 24×7 steady-state workload that would be 30%+ cheaper on reserved capacity.
- Idle storage / queues / dead test environments still billing.

**Remediation patterns:**
- Right-size: re-baseline against the last 30 days of metrics; reduce limits in 25% increments with monitoring.
- Reserved capacity: 1-year reserved on services with proven steady-state.
- Idle resource cleanup: monthly script to detect and notify; quarterly review to decommission.

**Source skill:** `azure-microservices-cost-review`.

## Cross-dimension findings

Some findings touch multiple dimensions. Examples:

- **Distributed monolith** — every dimension shows symptoms (chatty topology in 3, shared schemas in 1 and 2, no per-service SLO in 7, cascading failures in 5). Report it as a structural finding and link to the dimension hard-fails it produces.
- **Compliance scope expansion without controls map** — produces findings in both 8 (no controls) and downstream in 2 (data retention not aligned), 7 (audit log retention), 9 (cost of compliance not budgeted).

For cross-dimension findings, surface them in the report as named structural issues with the per-dimension hard-fails listed underneath, not as separate dimension entries.

## When dimensions disagree

A design that scores well on most dimensions but hard-fails on one is *not* "healthy with risks" — it depends on which one. A hard-fail in security or data consistency outranks a hard-fail in cost. A hard-fail in observability outranks a hard-fail in API contracts (you cannot operate what you cannot see).

Order of priority when collapsing to a verdict:

1. Security & compliance (Dimension 8)
2. Data architecture & consistency (Dimension 2)
3. Resilience (Dimension 5)
4. Observability (Dimension 7)
5. Communication topology (Dimension 3)
6. Domain & boundaries (Dimension 1) — load-bearing but expensive to fix; long-term plan acceptable
7. API contracts (Dimension 4)
8. Azure service mapping (Dimension 6)
9. Cost & operability (Dimension 9)

A hard-fail in 1–4 pushes the verdict to "At risk" at minimum. A hard-fail in 8 with sensitive data in scope pushes to "Unsound" until remediated.
