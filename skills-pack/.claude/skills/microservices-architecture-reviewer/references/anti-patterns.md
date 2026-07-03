# Architecture-Level Anti-Patterns

These are the named structural smells the reviewer scans for during the 9-dimension walk. Each card states what the anti-pattern is, the detection signal, why it fails, and the smallest viable fix. The catalog is deliberately tight — these are the load-bearing smells, not every possible flaw.

Anti-patterns are different from per-dimension defects: an anti-pattern is a *shape* that produces hard-fails across multiple dimensions. Detect the shape once; the dimension-level findings fall out naturally.

## Distributed monolith

**Shape:** services are split on the deployment unit but coupled on every other axis — shared database, synchronous chain of calls per business operation, lockstep deploys.

**Detection signal:**
- A normal feature requires changes in 3+ services *and* coordinated deploys.
- The runtime call graph shows a synchronous chain matching the business operation 1:1 (Order → Inventory → Pricing → Payment → Fulfillment, all sync).
- One service's release is regularly held up because another service's tests fail.
- "We have microservices but we deploy them together."

**Why it fails:** worst of both worlds. The team pays the operational tax of microservices (network, observability, deploy pipeline per service) and the coupling tax of a monolith (lockstep changes, coordinated releases, cascading failures). Velocity is lower than either a clean monolith or a properly decomposed estate.

**Hard-fails produced:** Dimension 1 (boundaries are wrong), Dimension 3 (sync chain is brittle), Dimension 5 (resilience cascades).

**Smallest viable fix:** identify the worst sync chain; introduce an async boundary at the highest-latency / most-failure-prone hop via outbox + Service Bus; verify the deploy decoupling holds (one service can deploy without the others). Repeat per chain. Do not redesign the whole topology in one go.

## Shared database

**Shape:** two or more services read and write the same database schema directly, bypassing each other's APIs.

**Detection signal:**
- Two `azurerm_postgresql_flexible_server_database` resources in Terraform point at the same Postgres for different services.
- A service's repository layer has SQL referencing tables it does not own.
- A migration in one service requires testing two services.

**Why it fails:** the "service" boundary is fiction; the real coupling is at the schema. Any schema change is a coordinated migration. Data ownership is ambiguous. Independent scaling, independent deploys, independent ownership — all impossible.

**Hard-fails produced:** Dimension 1, Dimension 2.

**Smallest viable fix:** pick the dominant owner of each table by domain meaning. The owner exposes a service API for the data. Migrate the non-owners to call the API for reads first; then for writes. Drop the direct DB grants last. Strangler fig over rewrite.

## Chatty service graph

**Shape:** rendering a single user-facing operation requires 8+ internal service calls, often with N+1 patterns at service granularity (call Catalog once per cart line).

**Detection signal:**
- Trace view of a typical request shows 8+ spans across services for one operation.
- Frontend or BFF makes a loop of calls, one per item.
- P99 latency is dominated by serial network hops, not by individual service work.

**Why it fails:** latency is the sum of hops; failure probability is multiplied. The system becomes slower than the monolith equivalent and less reliable. Adding caching at every hop adds complexity without solving the structural problem.

**Hard-fails produced:** Dimension 3 (topology), Dimension 5 (resilience), Dimension 7 (observability cost balloons).

**Smallest viable fix:** introduce a CQRS read model for the rendering path — denormalize once at write time, read in one call. Or introduce a BFF that fans out in parallel and aggregates. The chatty graph is fine for *commands*; it fails for *queries* on hot paths.

## Synchronous saga

**Shape:** a long-running business process implemented as a synchronous chain of service calls, where each step blocks the previous one.

**Detection signal:**
- A request handler issues 3+ outbound HTTP calls before returning, each conditional on the previous.
- Timeouts at the top-level handler are 30s+ to accommodate the longest plausible chain.
- Compensation logic is implemented as nested try-catch in the orchestrator handler, with explicit rollback calls per failure point.

**Why it fails:** the orchestrator's thread is held for the duration of the saga; partial failures leave the system in inconsistent states that compensation does not reliably recover. The saga pattern was designed for *async* execution precisely to avoid these issues.

**Hard-fails produced:** Dimension 2 (consistency), Dimension 3 (topology), Dimension 5 (resilience).

**Smallest viable fix:** convert to an orchestration saga with explicit state stored in the orchestrator's database; each step is dispatched via Service Bus; compensation is a named handler per failed step. Use Durable Functions or a workflow engine if the team does not want to hand-roll the state machine.

## Untraced async

**Shape:** async messaging is in use, but the consumer does not restore the parent trace context, so traces stop at the broker.

**Detection signal:**
- Open a typical trace in Tempo / Application Insights — the trace ends at the producer's "publish" span; no consumer span attaches.
- Debugging an async-flow incident requires correlating timestamps and message IDs by hand.
- The team avoids using async because "we can't tell what's happening."

**Why it fails:** observability disappears across the most operationally fragile boundary. Incident mean-time-to-detect and mean-time-to-resolve are dominated by manual correlation. The team retreats to sync calls to keep traces intact, undoing the async benefits.

**Hard-fails produced:** Dimension 7 (observability), often Dimension 3 (because the team avoids async for the wrong reason).

**Smallest viable fix:** enable OTel context propagation in the broker client (Service Bus message application properties carry the trace context). Verify with a smoke test that a trace spans producer → broker → consumer. Standardize across all async paths.

## Missing async boundary at a vendor edge

**Shape:** the system calls an external vendor (payment processor, fraud screening, email send) synchronously, holding the caller's thread for the vendor's tail latency.

**Detection signal:**
- An incident review traces the cause to "vendor X was slow."
- The caller's thread pool exhausted during the incident; downstream callers cascaded.
- The vendor's SLA is materially worse than the internal services' SLAs.

**Why it fails:** the system's reliability is now bounded below by the worst vendor's reliability. One bad day at the vendor is one bad day for everyone calling that path synchronously.

**Hard-fails produced:** Dimension 3, Dimension 5.

**Smallest viable fix:** introduce an internal worker between the system and the vendor. The system posts to Service Bus; a worker consumes and calls the vendor with a generous timeout, retry, and circuit breaker; the worker posts the result as an event. The system's hot path is freed; vendor instability is bounded to the worker pool.

## Functional decomposition

**Shape:** services are named by technical concept (UserService, ProductService, OrderService, NotificationService) rather than by business capability (Identity, Catalog, Sales, Communications).

**Detection signal:**
- The service list reads like a layer cake (auth, data, business logic, notification).
- A normal business operation crosses 4+ services because the operation cuts across the technical layers.
- New domain features routinely require changes in 3+ services.

**Why it fails:** the boundaries are along axes the business does not change, so the boundaries are constantly fighting business change. Every feature is a coordination problem.

**Hard-fails produced:** Dimension 1 (root cause), and cascades to Dimensions 2, 3, 5.

**Smallest viable fix:** re-decompose on bounded contexts via DDD. Identify the correct contexts; identify the worst current mismatch (the service that touches the most other services on a feature); extract the right service from the wrong layering via strangler fig.

## Kitchen-sink service

**Shape:** one service has grown to do four or five unrelated things because it owned one of them originally and the others were "easier to add here."

**Detection signal:**
- A service's tool list / endpoint list has 30+ entries spanning unrelated business concerns.
- The owning team is the largest team in the org because the service does the work of multiple services.
- Deploys are slow and risk-laden because the blast radius is large.

**Why it fails:** the service becomes a bottleneck for changes, a giant deploy risk, and a deployment-coordination problem. Different concerns inside the service have different scaling, reliability, and compliance profiles, and they conflict.

**Hard-fails produced:** Dimension 1, Dimension 9 (operability).

**Smallest viable fix:** identify the lowest-coupling internal cleavage; extract that piece into its own service. Continue iteratively until the remaining service has a coherent purpose.

## Compliance afterthought

**Shape:** SOC 2 / ISO 27001 is "in scope" but no controls are mapped to the architecture, audit logging is incidental rather than designed, data retention is ad hoc per service.

**Detection signal:**
- The team mentions an audit window approaching but cannot point to a controls-to-evidence map.
- Audit log retention varies per service.
- Asked "where would an auditor look for evidence of access control?", the team improvises an answer.

**Why it fails:** the audit fails or the audit demands hasty remediation under deadline pressure, often producing wrong or fragile fixes. Compliance posture is unstable.

**Hard-fails produced:** Dimension 8 (root cause), Dimension 7 (audit logging gaps).

**Smallest viable fix:** invoke `soc2-iso27001-controls-mapping` to produce the controls-to-Azure-evidence map. Identify the gaps. Remediate the highest-risk gaps first (typically access logs, secret management, change management evidence). Codify retention policy across services.

## Vendor-lock by accident

**Shape:** the architecture has accumulated dependencies on a specific managed service or third-party platform in ways that were not deliberate, and migration off is now expensive.

**Detection signal:**
- A team member says "we can't change X because everything depends on it" and the dependency was not an intentional architectural choice.
- A vendor price increase or feature regression cannot be addressed.
- Cloud drift: an Azure-primary estate that has accumulated AWS-isms or vendor SDKs nobody planned for.

**Why it fails:** the system loses the option value of choosing differently later. The lock-in might be acceptable, but it should be a deliberate trade.

**Hard-fails produced:** Dimension 6 (Azure service mapping with drift).

**Smallest viable fix:** inventory the lock-in dependencies. For each, decide: (a) accept it deliberately and document the trade-off in an ADR; (b) plan a migration with a target quarter. Do not pretend the lock-in is not there.

## Big-bang rollout culture

**Shape:** new features and infrastructure changes deploy in one cutover; there is no canary, no feature flag, no blue-green strategy.

**Detection signal:**
- Incident history shows correlation between "deploy day" and "outage."
- Rollback procedure is "redeploy the previous container image" — slow and risky under fire.
- Feature flagging is ad hoc per service or not present.

**Why it fails:** every deploy is a high-stakes event. Bad changes hit 100% of users immediately. Rollback is slow. Teams become conservative about deploys, reducing the value of CI/CD.

**Hard-fails produced:** Dimension 5 (rollout strategy), Dimension 9 (operability tax).

**Smallest viable fix:** enable Container Apps multiple-revisions mode; introduce blue-green by routing traffic between revisions; add a feature-flag system (Azure App Configuration or a third-party); standardize a canary procedure with abort criteria.

## How to scan a system fast

Given diagrams + Terraform + a representative trace, the scan order that catches the most in the least time:

1. **Look at one trace.** Count hops, identify sync chains, identify async-trace gaps. (Catches: chatty graph, untraced async, sync saga, distributed monolith.)
2. **Read the Terraform for shared resources.** (Catches: shared database, vendor lock-in via specific PaaS choices.)
3. **Read the deploy pipeline.** (Catches: big-bang rollout, missing feature flag, missing canary.)
4. **Open the runbook (or note its absence).** (Catches: compliance afterthought, missing SLO, missing observability discipline.)
5. **Ask for the service-to-team ownership table.** (Catches: kitchen-sink service, functional decomposition, ownership gaps.)

Five inspections, 30 minutes total, will catch most of the load-bearing anti-patterns before opening any service code.
