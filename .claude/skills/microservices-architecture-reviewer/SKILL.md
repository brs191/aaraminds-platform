---
name: microservices-architecture-reviewer
description: Reviews an existing Azure-hosted microservices architecture end-to-end and produces a structured verdict report, applying a 9-dimension framework (boundaries, data, communication, API contracts, resilience, Azure mapping, observability, security, cost) with pass / soft-fail / hard-fail per dimension. Use when auditing a system before a roadmap or re-platforming decision, producing a review for a steering committee, a brownfield health check, or scoping technical debt. Do not use for designing a new system (use microservices-architecture-design), code-level PR review (use pr-review-azure-microservices), or MCP-server readiness (use mcp-go-production-review).
version: 1.0.1
last_updated: 2026-05-30
---

# Microservices Architecture Reviewer

## When to use

Trigger this skill when reviewing an architecture that already exists or has been proposed in detail: an annual or pre-investment health check, a brownfield audit before a re-platforming, a steering-committee review, validating a third-party or vendor design, or scoping technical debt prior to a major release or compliance audit. Common triggers: "review this architecture," "we have an existing microservices estate — is it healthy," "produce an architecture review report for the steering committee," "do a brownfield audit before we add the new domain," "validate this vendor design."

Do **not** use this skill for: designing a new system from scratch (`microservices-architecture-design`); code-level PR review on a diff (`pr-review-azure-microservices`); MCP-server-specific production readiness (`mcp-go-production-review`); cost-only review (`azure-microservices-cost-review`); SOC 2 / ISO 27001 evidence collection only (`soc2-iso27001-controls-mapping`); narrow pattern questions like "saga vs. CQRS" (use the relevant narrower skill).

## The critical decision rule — review the system, do not redesign it

Architecture review evaluates the system *as it stands*. The output is a verdict plus a punch list of concrete defects, each with the smallest viable fix. It is not an excuse to redesign from scratch — even when the existing system is flawed, a clean-slate rewrite is almost always the wrong recommendation. Find what is broken, name it, propose the least-disruptive remediation path, and leave the rest alone unless it materially blocks the system's goals.

Two reviewer failure modes to avoid: (1) **rewrite drift** — every finding ends with "rebuild it from scratch," which is unactionable; (2) **rubber-stamp drift** — the reviewer notes minor cosmetic issues but misses load-bearing flaws because surfacing them feels confrontational. The discipline is to call out fatal flaws directly, propose surgical fixes, and stay silent on cosmetic issues that do not move the needle.

## The 9-dimension review framework

| # | Dimension | What to check | Hard-fail trigger | Source skill |
|---|---|---|---|---|
| 1 | Domain & service boundaries | Bounded contexts align with business capabilities; one team owns each service; no shared schemas | Functional decomposition (UserService / ProductService), shared mutable DB across services | `microservices-architecture-design` |
| 2 | Data architecture & consistency | Cross-service transactions named with consistency model; outbox or saga where needed; engine choice fits workload | Distributed 2PC; cross-service joins; no idempotency on state-changing operations | `microservices-data-architecture`, `azure-data-tier-design` |
| 3 | Communication topology | Sync vs. async choice justified; broker fits ordering / volume / fan-out needs; DLQs configured | Synchronous chains of 4+ hops; no DLQ on critical async paths | `microservices-async-messaging` |
| 4 | API contracts | Versioning strategy explicit; error semantics consistent; gateway placement intentional | Breaking changes shipped without version bump; inconsistent error envelopes across services | `microservices-api-design` |
| 5 | Resilience | Every outbound call has timeout + retry + circuit breaker; bulkheads where shared resources; named rollout strategy | Unbounded retries; no timeout on outbound calls; one slow dependency takes down N callers | `microservices-resilience` |
| 6 | Azure service mapping | Compute / data / messaging choices fit the workload and team operability; no service drift (AWS-isms, Bicep) | Wrong tier (e.g., Cosmos for transactional OLTP without partition design); cloud drift | `azure-service-mapping` |
| 7 | Observability | OTel-instrumented; SLOs defined and alerted; traces cross every async boundary; runbook per service | No SLO; alerts fire on symptoms (CPU) not user impact; tracing gap at the broker | `azure-microservices-observability` |
| 8 | Security & compliance | Entra ID auth + managed identity service-to-service; Key Vault for secrets; SOC 2 / ISO 27001 controls mapped | Any plaintext secret in code / config / Terraform; missing authorization on state-changing endpoints | `azure-microservices-security`, `soc2-iso27001-controls-mapping` |
| 9 | Cost & operability | Sizing matches measured load; scale-to-zero where appropriate; reserved capacity for steady-state; idle resources flagged | Over-provisioned by ≥2× without justification; pay-as-you-go for steady 24×7 baseline | `azure-microservices-cost-review` |

For the full checklist with detection cues, severity rationale, and remediation patterns per dimension, see `references/review-framework.md`.

## Review-pass logic

1. **Establish scope and inputs.** What is being reviewed: the whole estate, one product, one service? What artifacts exist (ADRs, diagrams, OpenAPI specs, Terraform, dashboards, runbooks, bill)? If artifacts are missing, that itself is a finding — call it out before walking the dimensions.

2. **Walk dimensions 1–9 in order.** Each dimension gets a pass / soft-fail / hard-fail rating with a one-line note explaining the call. Hard-fails are load-bearing flaws that materially threaten the system's goals; soft-fails are tracked follow-ups; passes get a brief affirming note so the reader knows the dimension was examined.

3. **For each hard-fail, name the specific defect.** Not "improve resilience" but "no timeout on `OrderService → PaymentService` calls; observed in `internal/payments/client.go` line 47 — one slow Payment call exhausts Order's thread pool." Specificity is the discipline. See `references/review-framework.md` for cue phrasing.

4. **Match findings against the architecture anti-pattern catalog.** The known catalog covers distributed monolith, shared database, chatty service graph, missing async boundary, synchronous saga, untraced async, etc. Fast pattern matching against the system's diagrams catches the load-bearing smells quickly. See `references/anti-patterns.md`.

5. **Propose the smallest viable fix per hard-fail.** Each blocker gets a remediation path that is incremental and reversible. Strangler fig over rewrite. Add a missing timeout over replacing the HTTP client library. The fix has a named owner, a measurable success signal, and a target quarter.

6. **Produce the verdict.** "Healthy," "Healthy with named risks," "At risk (with named blockers)," or "Architecturally unsound (rebuild path required)." Final-tier verdicts are rare; use them only when remediation cannot recover the system within a year. The report follows `references/review-report-template.md`.

## Worked example — brownfield: reviewing a 7-service Azure estate showing peak-hour instability

Setup: 3-year-old Azure estate, 7 Spring Boot services on Container Apps, Postgres Flexible per service, Service Bus between Order and Fulfillment, Front Door at the edge. Recent peak-hour incident: a slow Payment vendor caused Order, Checkout, and Cart to time out simultaneously. Architecture review requested before the team commits to a new "Loyalty" service.

Review walk:

1. **Scope and inputs.** Estate-level review. Artifacts available: C4 diagrams, 4 ADRs (2 stale, 2 current), OpenAPI for 5/7 services, Terraform for infra, Grafana dashboards exist, no runbooks. **Finding (cross-cutting):** runbooks missing; 2/7 services lack OpenAPI. Track as soft-fails before walking dimensions.
2. **Dimension 1 — Domain & boundaries.** Service list maps to business capabilities (Order, Checkout, Cart, Payment, Fulfillment, Catalog, Identity). One team per service. No shared schemas. **Pass.**
3. **Dimension 2 — Data architecture.** Each service owns its Postgres. Order → Fulfillment uses outbox + Service Bus. Order → Payment is synchronous HTTP (the incident root). No saga; no compensation defined for failed Payment after Fulfillment dispatch. **Hard-fail.** Cross-service transaction unnamed; payment failure mid-flow leaves orphan fulfillment.
4. **Dimension 3 — Communication topology.** Order → Payment is sync; Order → Fulfillment is async via Service Bus with DLQ. The sync hop to Payment is the load-bearing brittleness. **Hard-fail.** Async via outbox is the right pattern for state-changing Order → Payment; sync is wrong for the vendor-dependent leg.
5. **Dimension 5 — Resilience.** Reviewed the incident: Payment client uses default HTTP timeout (no override) and no circuit breaker. When the vendor stalls, Payment stalls, then Order's thread pool exhausts, then Checkout and Cart (which both call Order synchronously) cascade. **Hard-fail.** Three named missing controls.
6. **Dimension 6 — Azure mapping.** Container Apps for all 7 services. Postgres Flexible. Service Bus Standard. Front Door. No drift; choices fit. **Pass.**
7. **Dimension 7 — Observability.** OTel everywhere. Per-service dashboards. SLO defined for Order and Checkout only (not Payment, the failing service). Alerts fired on the incident — too late (after user impact). No runbooks. **Hard-fail.** Missing SLO and runbook on Payment.
8. **Dimension 8 — Security & compliance.** Entra ID at Front Door; managed identity service-to-service; Key Vault for secrets. SOC 2 controls mapped (the team did the work). **Pass.**
9. **Dimension 9 — Cost & operability.** Container Apps with scale-to-zero on Cart (low traffic); reserved capacity on Order and Checkout. Sizing measured. **Pass.**

**Verdict:** *At risk*. Three hard-fails, all in the Order → Payment leg and its observability. Remediation: (a) introduce circuit breaker + 2s timeout + bounded retries on Payment client — 1 sprint; (b) move Order → Payment to outbox + Service Bus with compensation on failure — 2 sprints, strangler-fig; (c) add Payment SLO, alerts on saturation, and a runbook — 1 sprint, parallel. Loyalty service decision deferred until (a) lands; (b) can proceed in parallel.

## Anti-pattern — review-as-redesign

**Bad:** Reviewer is asked to audit a 7-service estate. The output is a 40-page document proposing a new event-sourced architecture with Kafka, CQRS everywhere, and a service mesh. None of the original system's specific flaws are named; the recommendations apply to any microservices system.

**Why it fails:** The team cannot act on a clean-slate redesign while running production. The actual flaws (a missing timeout, a missing compensation, a missing SLO) go unaddressed because they were not specifically identified. Six months later, the team has neither delivered the redesign nor fixed the original issues.

**Detection signal:** the review's recommendations are framework-shaped ("adopt event sourcing," "introduce a service mesh") rather than defect-shaped ("add a 2-second timeout to `PaymentClient` and a circuit breaker around the Payment vendor call"). If a stranger could write the same recommendation without seeing the system, the review didn't actually review the system.

**Fix:** Anchor every finding to a specific named defect in the actual system. Propose the smallest viable fix. Reserve framework-level recommendations for cases where the system genuinely cannot recover incrementally — and call that out explicitly with a rebuild verdict.

## Verification questions

1. Did the review walk all 9 dimensions explicitly, or did it skip some?
2. For each hard-fail, is there a specific named defect tied to a file, service, or operation — not a generic improvement statement?
3. Is the proposed fix per blocker incremental and reversible, with an owner and a target quarter?
4. Was the architecture anti-pattern catalog scanned against the system's diagrams and operational reality, not just the docs?
5. Is the verdict explicit (Healthy / Healthy with risks / At risk / Unsound), not implicit in narrative?
6. For brownfield reviews: does the report propose strangler-fig or incremental migration where structural change is recommended, rather than rewrite?
7. Were missing artifacts (runbooks, ADRs, OpenAPI specs) themselves flagged as findings, not silently ignored?

## What to read next

- `references/review-framework.md` — full 9-dimension checklist with detection cues, severity rationale, and remediation patterns
- `references/anti-patterns.md` — architecture-level smells (distributed monolith, shared database, chatty graph, sync saga, untraced async, etc.) with detection signals and fixes
- `references/review-report-template.md` — standardized report layout for the verdict and findings
- `microservices-architecture-design` skill — the design-time counterpart; this reviewer audits what that skill produces
- `pr-review-azure-microservices` skill — for code-level review on a diff, complementary at a different scope
- `mcp-go-production-review` skill — for MCP-server-specific readiness, complementary at a narrower surface
- `azure-microservices-cost-review` skill — for cost-only deep review feeding Dimension 9
- `soc2-iso27001-controls-mapping` skill — for compliance evidence feeding Dimension 8
