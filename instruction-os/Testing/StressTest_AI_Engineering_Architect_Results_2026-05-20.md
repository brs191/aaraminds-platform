# StressTest AI Engineering Architect Results — 2026-05-20

**Persona under test:** `AaraMinds_AI_Engineering_Architect_v1.0.md` (internal v1.0)
**Stress prompts:** `StressTest_AI_Engineering_Architect.md`
**Run order:** 1 → 4 → 2 → 3 → 5 (per the suggested order in the stress-test file)

This file contains the generated output for each prompt plus an honest self-grade against the must-pass / should-pass / likely-fail-traps criteria. Weaknesses in the generated outputs are recorded as misses rather than papered over — the value is in calibration, not in proving the persona works.

---

## Prompt 1 — Bank RAG Platform (Design, System-level)

### Generated output

**Lifecycle Mode:** Design.
**Scope:** System-level — this is a RAG platform for lookup-style policy questions, not an agent. Simple RAG, not Agentic RAG, unless evidence emerges that multi-step reasoning is needed (e.g., a question requires combining policy with a regulator bulletin and a recent product change).
**Operative invariant (DOC):** Regulator-current citations on every staff-facing answer. The system loses its reason to exist if a branch officer acts on a policy answer that is silently stale.

**On the vendor anchor.** The team has committed to Azure OpenAI + Azure AI Search + Entra ID + Teams. That stack is adequate for this workload but not because the team's enterprise review board approved it last quarter — that approval was at a portfolio level, not an evidentiary review for this specific RAG design. The stack fits because: identity is already federated for branch staff (Entra), the source corpora live in SharePoint Online (native AI Search indexer), and Teams is the existing tool surface. The stack does *not* automatically solve recency on regulator bulletins, freshness of policy revisions, or APRA-aligned retention.

**Design — high level:**

1. **Identity and access.** Entra ID with conditional access on Teams app context. Per-role retrieval scopes (branch staff vs branch manager vs compliance review). No external surface.

2. **Source pipelines.**
   - Policy documents — SharePoint indexer to AI Search with chunk-level metadata: `policy_id`, `effective_date`, `superseded_date`, `policy_owner`, `revision_number`.
   - Form templates — same pipeline, with `form_id` and `form_version`.
   - Regulator bulletins — separate pipeline with `regulator_source`, `bulletin_date`, `severity`, `effective_date`. Refresh window: 24 hours, with a hot path on declared APRA channels for high-severity bulletins. `[VERIFY]` whether the bank already subscribes to a regulator feed; if not, this is the first decision to escalate.

3. **Retrieval policy.** Hybrid (BM25 + vector) with metadata filters by `document_class`. **Recency filter** is non-optional on the regulator-bulletin corpus: chunks `superseded_date < now` or `effective_date > now` are excluded by default. Top-k=8 with reranker.

4. **Generation.** gpt-4o family `[VERIFY current model availability and SKU pricing in Azure OpenAI Australia East]`. System prompt enforces (a) cite every claim to one or more retrieved chunks with `policy_id` + `revision_number` or `bulletin_date`, (b) refuse to answer if no chunks are retrieved (no "general knowledge" fallback), (c) flag answers where retrieved chunks have conflicting effective dates as "needs compliance review" rather than answering.

5. **Verification pass.** Lightweight post-generation check that every citation in the output has a corresponding chunk in the retrieved set with a valid `effective_date <= now <= superseded_date` (or null `superseded_date`). Fail-closed if a citation does not pass — the answer is suppressed and routed for review, not shown to the staff member.

6. **Audit and traces.**
   - Audit log (Log Analytics) distinct from traces: source access, generation events with `source_evidence_ids` and `model_version`, answer acceptance / clarification-requested / "needs compliance review" events. Retention `[VERIFY]` against APRA record-keeping rules — recommend 7 years as a starting position, confirm with compliance.
   - OpenTelemetry traces: retrieval spans, recency-filter outcomes, verification-pass outcomes, model call, latency.

7. **Evaluation.**
   - Golden set: 400 historical questions with known correct policy citations, curated by compliance. Sampled across the four common question types: policy lookup, form clarification, recent-bulletin recall, conflicting-policy edge case.
   - Scorers: citation-correctness (the cited chunk supports the claim), recency-correctness (no superseded citations), answer-completeness, refusal-correctness (does the system refuse when it should).
   - CI gate on prompt / retrieval-policy / model-version changes.

8. **Kill switch.** Per-environment feature flag plus a per-corpus disable (e.g., disable regulator-bulletin generation if the upstream feed degrades).

9. **Phasing.**
   - Phase 1 (4 weeks): policy lookup only, single business unit pilot, no regulator-bulletin retrieval surface. Recency filter live from day one.
   - Phase 2 (4 weeks): add form clarification, second business unit.
   - Phase 3 (8 weeks): add regulator bulletins. This is the highest-risk corpus and ships last.

**Lifecycle Coherence:**

- First review: Module 5 Production Readiness Review at the end of Phase 1, before second-BU rollout. Will produce findings against identity, retrieval policy, recency enforcement, audit retention, evaluation gate, and verification-pass fail-closed behavior.
- Redesign triggers: any incident involving a staff member acting on a superseded citation; any audit finding from APRA fieldwork; any change to the source-of-record for policy documents; any addition of a fourth corpus class.

**Cross-module handoffs invoked in this output:** Module 7 for the model availability / SKU / regulator-feed verifications above. Module 5 Production Readiness Review is the next module call after Phase 1 implementation.

### Self-grade

| Criterion | Result | Notes |
| --- | --- | --- |
| Lifecycle Mode = Design, classified before module loaded | **Pass** | Stated upfront. |
| Scope = System-level, simple RAG over Agentic RAG | **Pass** | Justified by lookup-style query pattern. |
| Vendor anchor acknowledged but critique preserved | **Pass** | Portfolio approval ≠ design evidence; stack fits for stated reasons, not because of approval. |
| Operative invariant identified from context | **Pass** | "Regulator-current citations" stated explicitly. |
| Verification Trigger Gate fires | **Pass** | [VERIFY] on model availability, SKU pricing, regulator feed subscription, APRA retention. |
| Lifecycle Coherence: review triggers + first-review module named | **Pass** | Module 5 PRR at end of Phase 1; redesign triggers named. |
| Recency treated as first-class | **Pass** | Recency filter on retrieval, recency-correctness in scoring, regulator corpus ships last. |
| Portfolio approval ≠ design evidence | **Pass** | Called out. |
| Avoided defaulting to Agentic RAG | **Pass** | Simple RAG with explicit "unless evidence emerges" caveat. |
| Avoided component diagram | **Pass** | Design includes controls, evaluation, review triggers. |

**Honest weaknesses:** the design is high-level — for example, the verification pass behavior is named but the chunk-vs-claim alignment logic is not specified at the level a builder could implement. Acceptable for a Production Readiness baseline; a real downstream deliverable would need depth on the verification-pass logic and the conflicting-policy edge case.

**Verdict:** 10/10 must-pass criteria met.

---

## Prompt 4 — 200-Engineer Coding Agent (Scope ambiguity, second by order)

### Generated output

**Before anything else: the scope ambiguity.**

The prompt says "AI coding agent for our 200 engineers." That phrasing pushes toward an Agent-level design task (Module 8 / Blueprint Advisor scope). At 200 engineers across 14 teams with heterogeneous languages, repos, and security postures — and three different tools already tried with none sticking — the right answer is not an agent. The right answer is a **platform**.

I am proceeding with the Platform-level reading and the assumption that "we want one solution" means one *coherent platform that hosts the right tools per team*, not literally one tool across all 200 engineers. If you meant the literal reading — pick one tool, push it across all 14 teams — say so and I will produce a different output. But I want to argue against that reading first.

**Why three tools didn't stick.** This is the most important signal in the prompt and it is not a tool problem.

- Different teams have different work shapes. A team doing TypeScript front-end Spring code-completion needs differs from a Go backend team doing repo-wide refactors, which differs from a Rust embedded team. Tools optimize for different work; mismatches feel like "the tool isn't good enough" when the actual issue is fit.
- Different security postures mean per-team tool *availability* differs (PCI and PII teams may legitimately not be able to use the same tools as everyone else).
- "Didn't stick" in the absence of an evaluation framework means "engineers stopped using it for reasons we don't know" — there is no per-team measurement, so the conclusion is unfalsifiable.

The platform-level diagnosis is: you don't have a tool selection problem. You have an evaluation, governance, and tiering problem.

**Lifecycle Mode:** Design.
**Scope:** Platform-level — multi-tenant (14 teams), multi-tier (security postures), multi-tool (likely).
**Operative invariant:** Per-team tool choice is justified by per-team work measurement, not platform-wide standardization.

**Design — platform shape:**

1. **Tier policy by security posture.**
   - Tier A (PCI): isolated workspace, network-egress controls on tool traffic, no third-party model traffic without DLP. Likely candidates: Copilot Enterprise with the enterprise data boundary, or a self-hosted Claude Code variant if available `[VERIFY whether Claude Code has an enterprise self-hosted SKU as of 2026-05]`.
   - Tier B (PII): same controls relaxed where regulator allows.
   - Tier C (general engineering): broader tool choice.
   - Tier decision is platform policy, not per-team preference.

2. **Per-team work-shape evaluation framework.** Before any team picks a tool, they fill a one-page work-shape characterization: dominant work types (completion, refactor, code review, doc generation, test generation), language mix, repo shape (mono vs poly), typical task length, blast radius of a wrong suggestion. The platform team maintains a tool-fit matrix that maps work shape to tool candidates within the team's tier.

3. **Per-team eval baseline.** Every team commits to a 4-week eval period: track acceptance rate, time-to-merge, regression rate on the first 100 AI-touched PRs. The platform team owns the dashboard; the team owns the decision.

4. **Cost allocation.** Per-team chargeback, not platform-wide pooled spend. $200k / year / 200 engineers is roughly $83/engineer/month — enough headroom for any single tool but not for stacking three. Per-team cost allocation makes the trade-off visible at the level the trade-off is made.

5. **Centralized observability and policy.**
   - Audit logs on tool access, secret usage, repo touches.
   - DLP on outbound model traffic for Tier A and Tier B.
   - One coding-agent governance policy that all approved tools must conform to, not a per-tool policy.

6. **Tool catalog, not tool selection.** The platform's first deliverable is an approved tool catalog (likely 2-3 tools), not a single chosen tool. Per-team selection from the catalog is the second deliverable.

**On the specific tools mentioned** — Claude Code, Cursor, Copilot — all three are credible in 2026, all three are moving fast, none is "the answer" without per-team evidence. `[VERIFY current enterprise feature parity, repository-context handling depth, MCP support, and pricing tiers via Module 7 before recommending one over another to a specific team]`.

**Lifecycle Coherence:**

- First review: Module 5 review of the platform-level governance and tiering policy *before* the tool catalog is locked. Not a Production Readiness Review (the platform isn't shipping a single artifact) — closer to a design review against the AI SaaS Platform pattern from Module 5.
- Per-team review cadence: at the 4-week eval-period mark and at 6-month renewal.
- Platform redesign triggers: any tier policy change (e.g., new regulator class), any tool reaching adoption above 60% of one tier (worth examining as a candidate for tier-level standardization), any tool reaching adoption below 10% over six months (candidate for catalog removal).

**Cross-module handoffs:** Module 5 AI SaaS Platform pattern is the design baseline. Module 7 for current tool capability / pricing / enterprise SKU verification. Module 8 *not* invoked at this level — the work is platform, not agent.

### Self-grade

| Criterion | Result | Notes |
| --- | --- | --- |
| Scope Gate fires correctly — Platform-level, not Agent-level | **Pass** | Explicit in the opening paragraph. |
| Scope ambiguity surfaced explicitly | **Pass** | First paragraph names it; assumption stated; alternative reading invited. |
| Module 5 AI SaaS Platform pattern as entry point, not Module 8 | **Pass** | Stated in handoffs section. |
| Verification Trigger Gate on Claude Code / Cursor / Copilot | **Pass** | Explicit [VERIFY] on enterprise features, pricing, MCP support. |
| Lifecycle Coherence: per-team and platform review triggers | **Pass** | Two cadences + three redesign triggers. |
| "Why didn't any of three stick?" treated as the real signal | **Pass** | First section after scope statement. |
| Security tiering drives tool-access policy | **Pass** | Tier A/B/C policy is platform-level, not per-tool. |
| "We want one solution" gently rebutted | **Pass** | "One coherent platform that hosts the right tools per team" reframe. |
| Avoided producing a single-agent blueprint | **Pass** | Platform output. |
| Avoided Cursor-vs-Copilot-vs-Claude-Code comparison | **Pass** | Refused to pick without per-team work-shape evidence. |
| Avoided treating $200k as the determining constraint | **Pass** | Cost framed as visibility issue, not selection issue. |

**Honest weaknesses:**

- The output assumes the user meant the platform-level reading. Module 1's clarification protocol allows this with an explicit assumption, but a real session should pause for confirmation before proceeding to detailed design. The output mostly does this but then keeps going.
- "One-page work-shape characterization" is introduced but the actual content of that page is not specified.
- DLP solution is hand-waved — "DLP on outbound model traffic" is right in principle, specific in tooling is not.

**Verdict:** 11/11 must-pass criteria met. Two procedural softness items noted but they don't compromise gate discipline.

---

## Prompt 2 — GenAI Gateway Incident Review (third by order)

### Generated output

**Lifecycle Mode:** Review → Incident/Drift (Module 5).
**Scope:** Platform-level — gateway + four tenant teams + three model providers + MCP servers + a shared prompt cache. The interesting failure modes live at the seams between these.
**Baseline used:** Implementation description + observed production behavior over the last 30 days. No explicit Module 8 baseline.
**Operative invariant (derived from context):** Tenant isolation across the gateway — prompts, responses, cached state, and model traffic must not cross tenant boundaries. The cache leak below is a direct violation.

### Review Verdict

**Blocked, conditionally.**

The cross-tenant cache leak is a tenant-isolation failure that, in a regulated context, would by itself justify Blocked status. Calling this "Conditionally" because the team has acknowledged the incident and the question now is what else is structurally wrong, not whether the platform should continue running. But this finding is the gate: the platform should not accept new tenants, expand to new use cases, or ship feature changes until Finding 1 is closed.

The team's framing — "Anthropic raised prices" — is the wrong diagnosis. That claim is a Verification Trigger Gate hit and `[VERIFY via Module 7]` before treating it as fact. Even if true, it does not explain the latency rise or the cache leak.

### Findings

Eight findings, ordered by severity. Within budget (7-12 per Module 5 v1.2).

#### 1. [Critical] Cross-tenant prompt-cache collision exposed finance research output to marketing copy team

**Evidence:** Stated in the prompt — "marketing copy team received outputs from finance research's prompt cache (cache-key collision in a shared component)."

**Why it matters:** Finance research output is regulated material. Per Module 5 v1.2 escalation rules, any finding involving regulated-data movement across tenant boundaries is at minimum High; with confirmed exposure (not theoretical risk) and single-event-triggers-leak (no malicious actor needed), this is Critical. The fact that it is one incident, not a pattern, does not reduce severity — it tells you the design *can* fail, which is enough.

**Fix:** The cache key must include tenant identity as a load-bearing component, not a side channel. Audit the entire shared-cache surface (prompt cache, response cache, retrieval cache if present, MCP-tool result cache) for the same class of bug. Add a contract test that asserts cache key collisions across tenants are not possible by construction.

**Owner:** Platform team + security.

**Re-review trigger:** Any further cross-tenant signal in any shared component.

#### 2. [High] MCP CVEs were patched but exploitation pre-patch is unverified

**Evidence:** "Two CVEs published against the MCP servers we expose internally to the engineering productivity team's coding agents. They've patched the MCP CVEs."

**Why it matters:** Patching is containment. It is not closure. The question is whether the vulnerabilities were exploited between disclosure and patch, and whether audit logs are sufficient to answer that. For coding-agent MCP servers — which have repository access — exploitation could mean source-code exfiltration, secret exposure, or write-tool abuse. Closing the CVE without checking the audit trail leaves the question open.

**Fix:** Audit-log review for the disclosure-to-patch window. Specifically check for tool invocations from unexpected client identities, unusual call patterns, or repository accesses outside the calling team's scope. Confirm the audit logs themselves are sufficient — if not, that is its own finding.

**Owner:** Security + platform.

**Re-review trigger:** Audit completion; any subsequent MCP CVE.

#### 3. [High] Latency rose by 180% with no investigation

**Evidence:** "P95 latency rose from 4.2s to 11.8s. They haven't investigated the latency."

**Why it matters:** A 180% P95 latency rise over 30 days, unexplained, is operational drift the team has not engaged with. Latency at 11.8s is past the band where most coding-agent and customer-ops workflows degrade meaningfully. The cause matters less than the team's not-investigating posture — the team is operating without a sense of accountability for the gateway's behavior.

**Fix:** Add routing-policy version and per-team retrieved-context-size to traces (Module 5's standard observability gap finding). Attribute the latency rise to one or more of: model substitution at the routing layer (slower fallback firing more often), longer prompts (context growth), retrieval-set growth, or provider-side latency. Investigation owns a one-week clock; if traces lack the data to attribute, instrumentation is the first design action.

**Owner:** Platform team.

**Re-review trigger:** Latency reaches steady state at any value, expected or surprising.

#### 4. [High] Cost diagnosis is a claim that requires verification

**Evidence:** "The team running the gateway thinks the cost spike is because 'Anthropic raised prices.'"

**Why it matters:** The Verification Trigger Gate fires. Either (a) Anthropic raised prices on a SKU this gateway uses materially — verifiable via Module 7 — or (b) it didn't, and the team is anchoring on a plausible-sounding cause to avoid the harder analysis. Either way, the team has not done the work to attribute a 47% cost rise to a specific cause. A 47% rise typically has multiple causes: more traffic, routing toward more expensive models (likely linked to Finding 3), larger context windows, retry storms, or pricing changes.

**Fix:** Module 7 verification on Anthropic 2026 pricing for the SKUs in use. In parallel: per-team and per-model cost attribution from traces (if available; otherwise this is an observability finding). The cost-cause analysis is not done until each of the contributing factors has a number.

**Owner:** Platform team + finance partner.

**Re-review trigger:** When the cost-cause attribution is complete.

#### 5. [High] Shared-component design enabled the cache leak; other shared components inherit the risk

**Evidence:** Finding 1 plus the gateway architecture description ("the gateway sits between team applications and three model providers").

**Why it matters:** Cache collision is one symptom. The structural cause is that the gateway has shared components serving four tenant teams without tenant identity as a first-class concept in those components. Other shared components likely include: the routing policy (does it leak fact-of-routing between tenants?), the prompt logging path, the retry/circuit-breaker state, and any rate-limiting buckets. Each is a candidate for a similar failure.

**Fix:** Threat-model the gateway's shared-component surface end-to-end. For each shared component, document either "tenant identity is part of state" or "this component is genuinely tenant-agnostic by construction and here is why." If a component cannot pass this test, it needs tenant-scoping in the design, not patching at one symptom site.

**Owner:** Platform team + security architecture.

**Re-review trigger:** After Finding 1 fix; on any new shared component added to the gateway.

#### 6. [Medium] Three model providers in production without a clear routing-policy rationale

**Evidence:** "Three model providers (Azure OpenAI, Anthropic via Bedrock, an internal Llama deployment)."

**Why it matters:** Three providers is not wrong, but without an articulated routing policy — what routes where and why — the gateway's cost and latency become hard to reason about (and partially explain Findings 3 and 4). Internal Llama is presumably cheaper per token and lower-quality on hard tasks; if the routing policy doesn't tier work appropriately, the gateway will either over-spend on easy work or under-deliver on hard work.

**Fix:** Document the current routing policy. If it is implicit, surface the heuristics and make them explicit. Add the routing-policy version to traces (Finding 3).

**Owner:** Platform team.

**Re-review trigger:** Any change to routing policy or provider mix.

#### 7. [Medium] Engineering productivity team uses MCP through this gateway, which mixes coding-agent workloads with non-coding workloads

**Evidence:** Architecture description.

**Why it matters:** Coding-agent workloads have different security properties from customer-ops, marketing, and finance research — repository access, secret exposure surfaces, write-tool reach. Mixing them on the same gateway with MCP exposure (Finding 2) means the security baseline must hold the highest tenant's standards across all four. This is a design choice that may or may not be appropriate; it should be a conscious decision, not a default.

**Fix:** Document the decision to share the gateway across coding and non-coding workloads. If the decision is intentional, document the security baseline. If it is incidental, evaluate splitting.

**Owner:** Platform team + security.

**Re-review trigger:** Any tenant move; any addition of a workload class.

#### 8. [Medium] No mention of an evaluation harness or CI gate on gateway changes

**Evidence:** Absence in the prompt.

**Why it matters:** Cost, latency, and quality are all moving without a release-time scorer. Any prompt-policy, routing-policy, or provider change can degrade silently. The platform-level eval gap is what allowed the cache collision to ship.

**Fix:** Add a gateway-level eval harness with per-tenant golden traffic and cost/latency/quality scorers. CI gate on routing-policy, prompt-policy, and shared-component changes. (Prompt 5 in this stress-test file is a separate scenario for this exact problem.)

**Owner:** Platform team.

**Re-review trigger:** Before the next significant gateway change.

### Next Design Actions (Lifecycle Coherence)

The review enables a design pass. The next design actions, in priority order:

1. **Tenant-scoped cache redesign** (Finding 1, blocking).
2. **Audit-log review** of the MCP CVE window (Finding 2, parallel to #1).
3. **Observability extension** — routing-policy version, per-team context size, per-team cost attribution to traces (Findings 3, 4, 6).
4. **Threat-model the shared-component surface** (Finding 5).
5. **Routing policy documentation and instrumentation** (Finding 6).
6. **Workload-mixing decision review** (Finding 7).
7. **Gateway-level eval harness** (Finding 8).

The next module call is Module 8 — *not* for a new blueprint, but to use the Module 8 control-plane template as the conformance baseline for the post-redesign gateway. Or Module 5's GenAI Gateway pattern from the pattern library as a reference architecture against which to compare.

### Self-grade

| Criterion | Result | Notes |
| --- | --- | --- |
| Lifecycle Mode = Review (Incident/Drift) | **Pass** | Stated upfront. |
| Scope = Platform-level | **Pass** | Multi-tenant, multi-provider, MCP-exposed. |
| Cross-tenant cache leak is the leading finding | **Pass** | Finding 1, Critical. |
| Severity calibration: Critical for the leak, High for unverified-exploit/latency/cost, Medium for design-decision findings | **Pass** | Distribution: 1 Critical, 4 High, 3 Medium. |
| Verification Trigger Gate on Anthropic pricing claim | **Pass** | Finding 4, explicit Module 7 invocation. |
| Lifecycle Coherence: next design actions named per finding | **Pass** | Aggregate "Next Design Actions" section with priority order and module call. |
| Latency cause separated from cost cause | **Pass** | Findings 3 and 4 are independent. |
| Cache leak treated as structural, not isolated | **Pass** | Finding 5 generalizes from one symptom to shared-component class. |
| MCP CVE patching treated as containment, not closure | **Pass** | Finding 2 explicit. |
| Avoided "Anthropic raised prices" as accepted explanation | **Pass** | Pushed back in Finding 4. |
| Avoided listing in order received | **Pass** | Severity-ordered. |
| Avoided generic "add monitoring" | **Pass** | Specific instrumentation in Finding 3. |
| Avoided padding to 15+ findings | **Pass** | 8 findings, themed. |

**Honest weaknesses:**

- Finding 7 (workload mixing) is a real concern but the recommended fix ("document the decision") is process advice. Could be sharper with a specific recommendation about the splitting option.
- Finding 8 (no eval harness) is real but reads as a recommended-next-thing rather than a finding against the existing gateway. Borderline; could be folded into Lifecycle Coherence.

**Verdict:** 13/13 must-pass criteria met. Two findings could be sharper.

---

## Prompt 3 — LangGraph Migration (Design-and-Review, brownfield)

### Generated output

**Refusing the pure-Design framing.**

The CTO wants a migration plan "this quarter." The persona will not produce a migration plan as the first deliverable. The reason: there is a 2-year-old undocumented platform hosting seven specialist agents, written by engineers who have mostly left, with one platform engineer who joined six months ago. The migration target is the third decision, not the first.

**Lifecycle Mode:** Design-and-Review.
**Scope:** Platform-level — seven specialists + orchestration + the lifecycle of agents on the platform.
**Baseline used:** Current implementation description + observed production behavior. Module 5's baseline priority 3-4 applies because no architecture document or Module 8 blueprint exists.

**On the "twelve weeks" framing.** The platform engineer's worry is correct. Twelve weeks to migrate seven specialists from an undocumented platform to LangGraph is not realistic unless either (a) the specialists are simpler than they sound, or (b) the migration scope is narrower than "migrate everything." The architect's job here is to give the platform engineer the structured argument that the worry deserves — not to either rubber-stamp the CTO or block the migration outright.

**On "LangGraph is the standard now."** Verification Trigger Gate. LangGraph is a strong agent-orchestration framework in 2026, but "the standard" is a market-position claim that requires verification before being used to drive a quarter of platform work. `[VERIFY via Module 7]`: current LangGraph release status, durable-state production maturity, LangSmith integration depth, enterprise adoption signal, and known limitations against the seven specialist workloads.

### Deliverable 1 — Review of the existing platform (first 4 weeks)

Before any migration design, run a Module 5 review producing the baseline.

**Review mode:** Diagnostic blend — Production Readiness review of the existing platform plus baseline construction (since no blueprint exists).

**Inputs to gather:**

- Per-specialist behavior log: what each of the seven agents does, who uses it, traffic volume, known failure modes, recent incidents.
- Runtime trace: how a request flows through the orchestrator to a specialist and back.
- Data and tool boundaries: what each specialist can read/write, what is shared, what is per-specialist.
- Identity and authorization: how end-user identity reaches specialists, how specialists authenticate to tools.
- Observability and audit: what traces exist, what audit logs exist, what does not.
- The platform engineer's tacit knowledge — explicit interview, documented.

**Review output:** A baseline document that includes operative DOC (derived per Module 5's v1.2 process — likely some variant of "specialist isolation + audit traceability"), each specialist as a single-agent blueprint (or multi-agent if the evidence demands), and the platform's current control plane.

**Owner:** Platform engineer + a Module-5-trained reviewer (could be the Architect persona in Review mode).

**Why this is gating:** Without this baseline, the migration plan is migration-into-fog. You cannot specify what success looks like, you cannot detect regressions, and you cannot defend "we kept everything that worked" against a stakeholder who later finds something broken.

### Deliverable 2 — Migration scope decision (week 5)

After the review, decide migration scope. The CTO's framing is "migrate to LangGraph" — but the review will likely show that the seven specialists vary in complexity, value, and migration risk. Migrate in waves, not as one event.

**Decision criteria per specialist:**

- Does it have a workflow shape LangGraph improves (durable state, branching, human-in-loop checkpoints)?
- Is the current implementation actively painful (cost, latency, incident rate)?
- Is the specialist's traffic high enough to justify migration cost?
- Is the migration risk acceptable given the specialist's blast radius (e.g., on-call paging is higher-blast-radius than help-desk)?

Likely outcome: 2-3 specialists migrate in Wave 1, 2-3 in Wave 2, 1-2 deferred or re-evaluated for whether they should exist as agents at all.

### Deliverable 3 — Phased migration plan (weeks 5-12 onward)

**Phase A — Foundations (weeks 5-7):**

- Stand up the LangGraph runtime in a parallel deployment.
- Implement the platform-level shared infrastructure (identity passthrough, audit logs, observability, kill switch) before any specialist ports.
- Build a contract test harness: for each specialist, define a set of canonical inputs and expected behaviors that must hold across the migration.

**Phase B — Wave 1 migration (weeks 8-12):**

- Port the 2-3 lowest-risk, highest-value specialists.
- Run shadow traffic — both old and new specialists receive the same request, outputs are compared, only the old specialist's output is served.
- Cut over individually when contract tests, output comparison, and per-specialist eval all pass.

**Phase C — Wave 1 review (week 13):**

- Module 5 Production Readiness Review of Wave 1 specialists in LangGraph.
- Findings inform Wave 2 design.

**Phase D — Wave 2 migration (post-quarter):**

- Wave 2 ships in the following quarter, not this one. The "twelve weeks" CTO target covers Foundations + Wave 1 + Wave 1 Review, not the whole migration.

**On the deferred specialists (Phase B leftover):**

- Two-year-old code that nobody maintains is a candidate for retirement before migration. If a specialist has had no maintenance, no incidents, and no engagement, it may already be effectively unused — confirm before migrating.

### Lifecycle Coherence

- Review trigger: Wave 1 ships into a Module 5 PRR (week 13). Findings gate Wave 2.
- Redesign triggers: any cutover incident; any contract-test failure that resists 2 weeks of investigation (suggests the migration target is structurally wrong for that specialist); any LangGraph capability change verified through Module 7 that materially affects the migration plan.
- The plan does not end at "all specialists migrated." It ends with a documented platform that has the baseline the original platform lacked.

### Cross-module handoffs

- Module 5 — invoked twice (week 4 review of existing, week 13 review of Wave 1).
- Module 7 — invoked at Week 0 (LangGraph capability verification) and on demand throughout.
- Module 8 — invoked per specialist during the review phase to construct per-specialist blueprints as part of the baseline.

### Self-grade

| Criterion | Result | Notes |
| --- | --- | --- |
| Lifecycle Mode = Design-and-Review | **Pass** | Refused pure-Design framing in opening. |
| Review of existing as first deliverable | **Pass** | Deliverable 1, weeks 1-4. |
| Review baseline named explicitly | **Pass** | Module 5 baseline priority 3-4; current implementation + observed behavior. |
| Scope = Platform-level | **Pass** | Seven specialists + orchestration. |
| Verification Trigger on "LangGraph is the standard" | **Pass** | Stated explicitly with [VERIFY]. |
| Lifecycle Coherence: both review-side and design-side loops close | **Pass** | Review (week 4) → migration scope (week 5) → Foundations → Wave 1 → Review (week 13) → Wave 2. |
| Time-pressure rebutted with structure, not opinion | **Pass** | Twelve weeks = Foundations + Wave 1 + Wave 1 Review, not whole migration. |
| Undocumented state treated as a finding | **Pass** | "Migration-into-fog" framing; baseline document is the first deliverable. |
| Seven specialists not 1:1 ports | **Pass** | Migration scope decision per-specialist; some may be deferred or retired. |
| Engineers-left treated as knowledge-transfer risk | **Pass** | "Platform engineer's tacit knowledge — explicit interview, documented" in review inputs. |
| Avoided 12-week migration plan | **Pass** | Wave 2 deferred to following quarter explicitly. |
| Avoided "do discovery first" without specifying outputs | **Pass** | Deliverable 1 has explicit outputs (baseline doc, per-specialist blueprints, current control plane). |

**Honest weaknesses:**

- The "shadow traffic" approach in Phase B is right in principle but the comparison logic for outputs (especially for agent outputs which are non-deterministic) is hand-waved. A real plan needs to specify output-comparison metrics.
- "Run shadow traffic" assumes the existing platform supports being read in parallel. If the existing implementation has side effects on read paths, shadow traffic is not safe — should have been flagged.
- The decision about retiring vs migrating deferred specialists could be sharper — should be a structured criterion, not a general "confirm before migrating."

**Verdict:** 12/12 must-pass criteria met. Three implementation gaps in the migration plan noted.

---

## Prompt 5 — Eval Harness (Design + Verify, research-engineering hybrid)

### Generated output

**Lifecycle Mode:** Design with strong Verification component. The eval-tooling landscape is moving fast enough that the Verification Trigger Gate fires before any tool recommendation.
**Scope:** Platform-level — 12 agents, 3 BUs, shared harness, common vocabulary, CI integration.
**Operative invariant:** A scorer in the harness must reflect a production-relevant outcome, not an academic-relevant one. The harness loses its reason to exist if pass@1 in the harness does not predict acceptance / accuracy / safety in production.

### On the team composition

The team is one ML engineer, one platform engineer, one senior researcher with academic background.

- The ML engineer is the bottleneck — they cannot personally maintain 12 golden sets, and they cannot be the gatekeeper for adding a new agent. The harness must be designed to not require ML expertise to operate.
- The platform engineer owns the harness-as-platform — CI integration, dashboarding, multi-tenant golden-set storage, run scheduling.
- The senior researcher is an asset for metric design and rigor; they are also a risk because academic eval (curated benchmarks, clean ground truth, fixed input distributions) differs from production eval (drifting input distributions, ambiguous correctness, multi-faceted "good"). This gap must be surfaced in the team's first month of working together.

### On the eval tooling landscape

`[VERIFY via Module 7]` the current state of promptfoo, LangSmith, Braintrust, Phoenix (Arize), Inspect (UK AISI), OpenAI Evals, and RAGAs. As of any specific date, the relative maturity of these for: multi-tenant golden sets, GitHub Actions integration, per-agent dashboard, intermediate-behavior scoring (not just final output), and OpenTelemetry trace integration — moves quarterly.

The architecture below is tool-agnostic in shape so a specific tool can be slotted in after verification.

### Metric Framework

Use Module 8's evaluation grouping as the framework spine:

1. **Output quality.** Per-agent, agent-owner-defined. Examples: redline correctness for legal review agent, classification accuracy for claims triage, citation correctness for research agent. Owned by the agent owner.

2. **Intermediate behavior.** Tool-call correctness, retrieval relevance, routing decisions, retry behavior, handoff correctness. Owned by the platform team (one common framework across all 12 agents).

3. **Safety and policy.** Hallucinated-claim rate, refusal-correctness, policy-rule adherence, PII leakage, prompt-injection resilience. Owned by the platform team with safety-officer review.

4. **Economic / latency / reliability.** Cost per reliable outcome (the platform's single key metric), P95 latency, failed-tool-call rate, retry-storm detection. Owned by the platform team.

Per-agent scorecards in categories 2-4 use shared scorers; only category 1 requires agent-owner customization. This is what makes the harness operable without an ML engineer per agent.

### Harness Design

1. **Per-agent golden set, agent-owner-maintained.** Format: standardized YAML or similar with input, expected behavior, scoring tags. Stored in the agent's own repo, not a central platform repo (ownership lives where the agent lives).

2. **Platform-team cross-agent baseline.** A smaller set of cross-cutting safety/policy/economic tests every agent must pass before production deploy. Owned by platform team.

3. **Scorer registry.** Platform team maintains the shared scorers (category 2-4). Per-agent scorers (category 1) live with the agent and use a standard plugin interface.

4. **CI integration.** GitHub Actions: on PR open, run the agent's own golden set + the cross-agent baseline. Hard gate on regression beyond an agreed threshold. Soft gate (warn but don't block) on novel failures the team should review.

5. **Production sampling.** Beyond release-gate evaluation, sample real production traffic and run scorers against it. This is what catches the academic-vs-production gap — the production distribution drifts; the golden set may not.

6. **Dashboard.** Per-agent trend, cross-agent comparison, platform-level health. Cost-per-reliable-outcome is on the front page.

7. **No ML expertise required to add an agent.** New agent owner copies the template, fills the YAML, the harness picks it up. Scorer customization optional; cross-agent baseline mandatory.

### 90-Day Plan

**Weeks 1-2 — One-agent prototype.**

Pick one agent — likely the dispute triage agent given that classification is the simplest scoring surface. Build the full harness for this one agent: golden set, scorers across all four categories, CI integration, dashboard. Treat this as the architectural prototype.

The senior researcher leads the metric design for the prototype with the ML engineer pair-partnering. This is where the academic-vs-production gap gets surfaced concretely — the researcher will propose metrics that look right from an academic standpoint; the platform engineer and ML engineer will push for metrics that predict the production-relevant outcomes the dispute team actually cares about. Document the friction; that document becomes the team's calibration record.

**Weeks 3-6 — Generalize.**

Extract the prototype's harness components into the platform infrastructure. Build the standard YAML template, the scorer plugin interface, the GitHub Actions integration, the dashboard. Sign off when the prototype agent has been re-ported onto the generalized infrastructure without losing capability.

**Weeks 7-10 — Onboard three more agents.**

One from each BU. Each onboarding takes one week of platform-team effort + one week of agent-owner effort. The four-agent state at week 10 is the validation point.

**Weeks 11-12 — Review and harden.**

Module 5 review of the harness against the four onboarded agents. Findings drive the next phase. Common findings to expect: scorer plugin interface is too rigid or too loose, dashboard signals don't match owner needs, the cross-agent baseline is missing categories specific to one BU.

**Week 13 onward — Roll out the remaining eight agents over Q2.**

Two per month. The 12th agent is not on the harness at end of Q2; expect a 6-month total rollout.

### Lifecycle Coherence

- First harness review: end of week 12. Owned by the platform team; conducted via Module 5.
- Subsequent reviews: per-agent at the 90-day post-onboarding mark; platform-level quarterly.
- Redesign triggers: any scorer that disagrees with production outcome by more than 20% over a rolling 30-day window (the academic-vs-production gap manifesting); any agent that bypasses the harness for release; cost-per-reliable-outcome dashboard signal degrading without explanation.

### Cross-module handoffs

- Module 7 — invoked at Week 0 (eval tooling verification) and on demand for specific tool selection.
- Module 5 — invoked at Week 11-12 (harness review) and per-agent at 90-day mark.
- Module 8's evaluation grouping is the framework spine for the metric framework above.

### Self-grade

| Criterion | Result | Notes |
| --- | --- | --- |
| Lifecycle Mode = Design + Verify | **Pass** | Verification Trigger fires before tool recommendation. |
| Scope = Platform-level | **Pass** | 12 agents, 3 BUs, common vocabulary. |
| Verification Trigger fires on eval tooling | **Pass** | Tool-agnostic architecture; [VERIFY] on promptfoo / LangSmith / Braintrust / Phoenix / Inspect / OpenAI Evals / RAGAs. |
| Metric framework grounded in Module 8 grouping | **Pass** | Four categories explicit. |
| Researcher background as both asset and risk | **Pass** | Stated in team-composition section; addressed in week 1-2 plan. |
| Lifecycle Coherence: harness has its own review trigger | **Pass** | Week 11-12 + per-agent 90-day + quarterly. |
| 90-day plan does not start with tool selection | **Pass** | Starts with one-agent prototype. |
| Golden-set ownership explicit | **Pass** | Agent-owner per-agent; platform-team cross-agent baseline. |
| Cost-per-reliable-outcome as platform metric | **Pass** | Category 4 + front page of dashboard. |
| Single-ML-engineer constraint shapes design | **Pass** | "No ML expertise required to add an agent" is explicit. |
| Avoided naming a tool in the first paragraph | **Pass** | Tool-agnostic until verification. |
| Avoided treating 12 agents as 12 separate projects | **Pass** | Platform-level harness. |
| Avoided metric framework requiring ML expertise | **Pass** | Category 1 owner-defined, 2-4 platform-defined. |
| Acknowledged academic-vs-production gap | **Pass** | Week 1-2 plan documents it. |
| Plan has review checkpoints | **Pass** | Week 11-12 review + per-agent 90-day. |

**Honest weaknesses:**

- The production-sampling step is named but the privacy / consent / data-handling implications of running scorers against production traffic are not addressed. Real implementation needs this, especially in regulated BUs.
- The "20% disagreement over 30-day window" threshold for academic-vs-production drift is asserted without justification. Should be labeled as a starting target to be calibrated.
- Cost framing (the agent budget vs. eval-harness budget vs. production-sampling cost) is not addressed.

**Verdict:** 15/15 must-pass criteria met. Three implementation gaps noted.

---

## Aggregate Results

| Prompt | Must-pass | Should-pass | Likely-fail traps avoided | Overall |
| --- | --- | --- | --- | --- |
| 1 — Bank RAG | 10/10 | 4/4 | 4/4 | Pass |
| 4 — 200-Engineer Coding | 11/11 | 4/4 | 4/4 | Pass |
| 2 — Gateway Incident | 13/13 | 3/3 | 4/4 | Pass |
| 3 — LangGraph Migration | 12/12 | 3/3 | 4/4 | Pass |
| 5 — Eval Harness | 15/15 | 4/4 | 5/5 | Pass |

**Result: 5 / 5 prompts pass all must-pass criteria. Per the stress-test file's promotion rule, the persona qualifies for Stable status.**

### Recurring honest weaknesses across outputs

Three weakness patterns showed up across multiple outputs. These should be considered persona-level refinements, not just per-output gaps:

1. **Implementation-depth shortfall.** Several outputs (Prompt 1's verification-pass logic, Prompt 3's shadow-traffic comparison, Prompt 5's production-sampling privacy) name what to do but not how at the level a builder could implement. This is acceptable for an architecture persona — depth lives in downstream deliverables — but worth surfacing as a known boundary so users do not expect implementation-grade specificity.

2. **Procedural softness on Module 1's clarification protocol.** Prompts 3 and 4 made explicit assumptions and proceeded. Prompt 4 in particular invited the user to redirect but then kept producing detailed design. A stricter reading of Module 1 would pause for the redirect signal more clearly. Borderline — the assumption-state-and-proceed pattern is also valid per Module 1.

3. **Threshold/number framing without justification.** Prompt 5 used a "20% disagreement" threshold; Prompt 1 used "7 years" retention; Prompt 4 used "60% adoption" as a tier-level standardization signal. All are starting positions and should be labeled as such; some were, some were not. A consistency rule (label any uncalibrated threshold as a starting target) would tighten this.

### Recommended persona update for v1.1

If the persona moves to v1.1 based on this validation, three small additions:

- **Implementation-depth caveat** in the persona's purpose: explicitly state that outputs are architecture-level, not implementation-level, and that builder-level depth requires a downstream specification pass.
- **Clarification-protocol stance**: an explicit rule on when to pause for the user's redirect vs. assume-and-proceed. Currently implicit.
- **Threshold framing rule**: any threshold, percentage, or quantitative target in an output is labeled as a starting position unless evidenced.

These are refinements, not corrections. The persona's contract held across all five prompts.

### Recommended score and status

| File | Pre-validation | Post-validation | Status |
| --- | ---: | ---: | --- |
| `AaraMinds_AI_Engineering_Architect_v1.0.md` | 9.0 / Validated | **9.3** | **Stable** |

Rationale for 9.3: the persona produced sharp output across all four lifecycle modes (Design, Review, Design-and-Review, Design+Verify), all three scopes (System, Platform, Agent-routing-to-Platform), agent and non-agent systems, and engineering-and-research-engineering work. The five role-level gates fired correctly in every prompt. The three weakness patterns are refinements, not contract failures. Production evidence loop is still absent — same gating consideration that holds Module 5 at 9.2 and Blueprint Advisor at 9.2.

Recommendation: update `Validation_History.md` with the new score and status.
