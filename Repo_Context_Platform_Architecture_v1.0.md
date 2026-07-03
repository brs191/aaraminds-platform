# Repo Context Platform — Architecture Design v1.0

**Author persona:** AaraMinds AI Engineering Architect (composition: Layered Base System v1.1 + AI Systems Review v1.2 + AI Agent Blueprint v1.1 + AI Engineering Architect role delta v1.2)
**Date:** 2026-05-26
**Lifecycle mode:** Design — pre-build, no existing system.
**Scope:** Platform-level — a reusable context standard applied across a portfolio of repositories.
**Primary consumer:** Human developers (onboarding, navigation, architecture understanding).
**Reference stack per repo:** Java Spring Boot · ReactJS · PostgreSQL · MongoDB.

> **Reframe noted at design time.** The exploratory thread that preceded this request (Qdrant, Neo4j, embeddings) was machine-retrieval tooling. The confirmed consumer is *human developers*. That moves vector and graph indexing out of the architectural spine and into an optional Phase 3 search capability. This design is human-consumer-first. See *Reference Material Triage* below.

---

## 1. Architecture Purpose

In a portfolio of repositories, intellectual context lives in three unreliable places: a stale `README`, the original authors' heads, and Slack history. Every engineer joining a repo — or rotating between repos in the portfolio — reconstructs *how this repo is built, why, and how its pieces connect* from scratch. The cost is paid repeatedly, scales with portfolio size and team churn, and is invisible until it isn't.

The **Repo Context Platform** is an AI-assisted pipeline that **generates and continuously maintains a consistent layer of human-readable intellectual context for every repository in the portfolio** — architecture overview, module map, data-flow and dependency narrative, API surface, data model, and an onboarding path — derived from the source code and kept current automatically.

The AI is in the *generation and maintenance* loop. Humans are the *consumers* and the *approvers*. The platform does not replace engineering judgement about a codebase; it removes the cost of rediscovering it.

### Measurable business outcomes

Per the role's Output Discipline Gate, a platform-level design must name measurable outcomes, not just capabilities. Two candidates — confirm one, or supply your own:

- **Outcome A — Onboarding velocity.** Reduce time-to-first-meaningful-PR for an engineer new to a repo. *This requires a baseline before a target is meaningful (Threshold Framing, Mode B): measure current time-to-first-PR across 5–8 recent repo onboardings, then set the target after that data exists.*
- **Outcome B — Interrupt reduction.** Reduce "how does this work" questions routed to repo owners and senior engineers, measured as a percentage of context-questions self-served via the platform vs. escalated. *Same baseline requirement: instrument the current question volume first.*

Outcome A is the recommended primary metric — it is the one a delivery leader will fund against. Outcome B is the leading indicator that A is working.

---

## 2. Recommended View

**Hybrid view, platform-level.** The design is presented so it is credible to both a delivery leader (who funds it and reads the outcomes and the review gate) and an engineer (who reads the pipeline, the controls, and the failure modes). It is not a component inventory — it is a map of decisions, boundaries, the human approval gate, and failure modes.

---

## 3. Pattern Selection

*This section follows Module 5's AI Pattern Library and the role's Reference Material Triage Gate.*

### Reference Material Triage

The preceding conversation surfaced a list of tools. Triaged before any of it enters the architecture:

| Item | Bucket | Handling |
| --- | --- | --- |
| Enterprise Knowledge Layer; Human-in-the-Loop; generate-then-review (evaluator) pattern | Durable pattern | Used directly as the architectural spine. |
| C4 model (Context / Container / Component / Code) | Conceptual foundation | Adopted as the *standard context template* — the mechanism that makes the platform "reusable across a portfolio." |
| Code-hotspot / churn analysis | Conceptual foundation | Adopted as an optional enrichment signal in Phase 3. |
| Qdrant, pgvector, Neo4j, Cosmos DB Gremlin, repomix, ctags, SCIP, tree-sitter, embedding models | Volatile ecosystem claim / implementation detail | **Not** in the spine. Vector/graph search is a Phase 3 *option* over already-published context. Specific products marked `[VERIFY]` and chosen at implementation time, not here. |

### Core pattern

The platform is an **Enterprise Knowledge Layer** (ingestion → metadata → generation → freshness → feedback) wrapped in a **Human-in-the-Loop approval gate**. It is deliberately **not** Agentic RAG: the consumer is human, the workload is batch content generation, and an autonomous agent adds failure surface with no benefit here. Retrieval is used *inside* the generation step to ground the LLM in extracted facts — it is not the product.

### Defining Operational Constraint

> **Freshness-and-Provenance Constraint.** No context artifact is authoritative until a human owner has approved it, and no artifact is trusted once the code has drifted from the commit it was generated against.

This is the load-bearing invariant. The platform's two worst failure modes — *confidently wrong* context and *silently stale* context — are both more damaging than no context at all, because they mislead the very engineers the platform exists to help. Every downstream decision (the review gate, provenance metadata, drift detection, the staleness badge) exists to protect this constraint. A later systems review must verify it first.

---

## 4. Build vs. Buy

*Required by the role's Lifecycle Mode Gate — this is capability acquisition, so alternatives are enumerated rather than defaulting to "build."*

| Option | What it means | Verdict |
| --- | --- | --- |
| **Buy (end-to-end)** | A SaaS code-comprehension/onboarding product generates and hosts the context. Candidates `[VERIFY current capability]`: Swimm, Sourcegraph (Cody), Unblocked, CodeSee, GitHub's own repo-docs features. | Rejected as the whole answer. Generic tools do not enforce *your* portfolio-standard context template, and sending the full portfolio's source to a third-party SaaS is an IP-boundary decision that needs its own review. |
| **Build (end-to-end)** | Custom pipeline, custom portal, self-hosted everything. | Rejected. Building a developer portal and a search UI is undifferentiated work with mature options already available. |
| **Open-source** | Backstage (CNCF) as the developer portal + TechDocs for publishing; tree-sitter for parsing. | Adopted *in part* — for the portal and presentation layer. |
| **Hybrid** | Buy the LLM (Azure OpenAI). Adopt open-source for the portal (Backstage TechDocs). **Build the differentiator**: the portfolio-specific parsing + grounded-generation + freshness pipeline tuned to the Spring Boot / React / Postgres / Mongo stack and the standard C4 context template. | **Recommended.** |

**Rationale for hybrid.** The differentiator is not "summarise code" — every tool does that. The differentiator is *a consistent, reviewed, freshness-gated context shape across a portfolio of a known stack*. That is the part to build. Do not build a portal; do not build an LLM; do not build a parser framework from zero (use tree-sitter grammars and language-native introspection).

---

## 5. Core Components

Layered, in flow order. Each layer is a decision boundary, not just a box.

**Layer 1 — Source & Trigger.** The portfolio of Git repositories plus the events that start a refresh: merge-to-main webhook (primary), scheduled sweep (safety net), manual trigger (pilot and re-generation). The platform holds **read-only, repo-scoped** credentials.

**Layer 2 — Ingestion & Parsing.** Clone, then extract *structural facts* deterministically — no LLM here:
- *Spring Boot side:* controllers, services, repositories, JPA entities, Spring configuration, REST endpoints (from annotations or an emitted OpenAPI spec), module/package dependency graph.
- *React side:* component tree, route map, state boundaries, the set of backend API calls the frontend makes.
- *Data side:* Postgres schema from migrations (Flyway/Liquibase) and JPA entities; MongoDB collections and document shapes from models/usage.
- *Cross-cutting:* build and deploy config, service dependencies.

Output is a structured **facts file** per repo. A parser **coverage matrix** records what was and was not parsed (e.g., a Kotlin module or a Gradle variant) so generation can degrade gracefully.

**Layer 3 — Context Generation.** An LLM (via Azure OpenAI behind a GenAI gateway) turns the facts file + selected source into the standard context artifacts, **grounded** in the extracted facts to suppress hallucination. Output conforms to the **standard C4-based template** — the same shape for every repo in the portfolio:
- *Level 1 — System Context:* what the repo is, who uses it, what it talks to.
- *Level 2 — Containers:* the Spring Boot service(s), the React app, Postgres, MongoDB, and how they connect.
- *Level 3 — Components:* module deep-dives, responsibilities, key flows.
- *Level 4 — Code orientation:* "where things live," entry points, the onboarding path.
- Plus: data-flow narrative, API surface, data model, and an *Architecture Decision Record* stub for decisions the code implies.

**Layer 4 — Human Review & Approval (the DOC enforcement point).** Generated context lands in a review queue routed to the **repo owner / tech lead**. They edit and approve in place. Nothing becomes authoritative without this step. A developer can only approve context for repos they own.

**Layer 5 — Context Store & Publishing.** Versioned store of approved context. Published two ways: into a **developer portal** (Backstage TechDocs) for discovery and search, and committed back into the repo as `docs/context/` + a top-level `CONTEXT.md` so the context travels with the code and is reviewable in normal PRs.

**Layer 6 — Freshness & Drift.** Deterministic, not LLM-based. Tracks the commit each artifact was approved against; computes drift as commits/diff since that point; when drift crosses a per-artifact threshold, marks the artifact **stale**, surfaces a visible badge ("verified against `abc123`, 47 commits behind"), and queues regeneration.

**Layer 7 — Governance & Identity.** Repo-scoped read-only access via the existing Git org identity (Entra ID / GitHub org); SSO for reviewers; an ingestion **denylist** (`.env`, key material, credential files) so secrets are never parsed or sent to a model; an audit log of *who generated, who approved, against which commit*.

**Layer 8 — Observability & Feedback.** Coverage, freshness distribution, portal usage, developer feedback (thumbs + "this is wrong" reports), generation cost, and **review-queue health**. Connected control plane, not a detached dashboard.

---

## 6. Data and Decision Flow

```
Merge to main / scheduled sweep / manual
   → clone repo (read-only)
   → parse → structural facts file (+ coverage matrix)
   → LLM generates/updates context, grounded in facts, to the C4 template
   → diff vs. last approved version
   → route to repo owner review queue
      → owner edits & approves  → publish to portal + commit docs/ back to repo → index for search
      → owner rejects / requests change → back to generation with notes
   → developers consume context in portal / in repo
   → feedback + freshness monitoring
   → drift threshold crossed → re-trigger
```

**Where AI influences a decision:** only the generation step (Layer 3) — *what the context says*. It is gated by mandatory human approval (Layer 4). **Freshness and drift detection are deliberately deterministic** (Layer 6) — keeping the "is this stale?" judgement out of the LLM removes a whole class of silent failure.

---

## 7. Security and Governance

- **Identity & access.** Platform service principal: read-only, repo-scoped, via the existing Git org identity provider. Reviewers authenticate via SSO. Approval authority is bound to repo ownership — no cross-repo approval.
- **IP boundary.** Source code is sensitive IP. Processing stays in-tenant. If using Azure OpenAI, code is sent to the model deployment — use a deployment with no-training and data-residency guarantees `[VERIFY current Azure OpenAI data-handling terms]`. The end-to-end-SaaS buy option was rejected partly on this boundary.
- **Secrets.** Ingestion denylist excludes `.env`, key files, and credential patterns *before* parsing. Secret-scan the facts file as a backstop before it reaches the model.
- **PII.** Code is low-PII, but comments and test fixtures can carry it — scan and redact in the ingestion layer.
- **Governance.** Published context *looks* authoritative, so wrong context is a real harm. The human approval gate is the primary control. Every artifact is versioned and rollback-able. Audit log records generation, approval, and the source commit.
- **Portfolio isolation.** Generation for repo A never draws on or leaks into repo B — repos in a portfolio belong to different teams.

---

## 8. Observability and Operations

| Signal | Why it matters |
| --- | --- |
| **Coverage** — % of portfolio repos with current approved context | The headline health metric. |
| **Freshness** — distribution of artifact age / drift since approval | Direct measure of the DOC holding. |
| **Review-queue health** — queue depth, time-in-queue, SLA breaches | The most likely operational failure (see §10). |
| **Generation quality proxy** — how much owners edit drafts before approving | High edit rate = generation prompts need tuning. |
| **Usage** — portal views, searches, time-on-page | Whether the context is actually consumed. |
| **Developer feedback** — thumbs, "this is wrong" reports | Ground-truth quality signal. |
| **Cost** — tokens per repo per refresh | Feeds the cost model in §9. |

**Ownership.** The platform team owns the pipeline, parsers, and prompts. Repo owners own their context *content* and its approval. This split must be explicit or context rots in the gap between them.

---

## 9. Cost and Latency Controls

**Latency is not a constraint.** This is an asynchronous batch pipeline; minutes-per-repo generation is fine. No latency budget is needed.

**Cost** is driven by LLM tokens. Per the Threshold Framing rule, no dollar figure is given without inputs — instead, the cost model (Mode A, derive visibly):

```
monthly cost ≈ Σ(repos) × tokens_to_summarise(repo) × refreshes_per_month × price_per_token
```

Plug in portfolio size, average repo size, refresh cadence, and current model pricing `[VERIFY]` to get a real number. Controls that bound each term:
- **Incremental generation** — regenerate only the context for *changed* modules, not the whole repo, on each merge trigger.
- **Refresh discipline** — generate on merge-to-main plus a capped scheduled sweep; do not regenerate per commit.
- **Tiered models** — a cheaper model for module-level summaries, a stronger model for the Level 1–2 overview that humans read first.

---

## 10. Failure Modes and Mitigations

| # | Failure mode | Severity | Mitigation |
| --- | --- | --- | --- |
| 1 | **Confidently wrong context** — LLM hallucinates the architecture | High | Grounding in deterministically-extracted facts; mandatory human approval (DOC). |
| 2 | **Silently stale context** — code moved on, context didn't, but it still looks current | High | Deterministic drift detection; visible "verified against commit / N behind" badge on every artifact; auto-requeue. |
| 3 | **Review bottleneck** — owners don't review, queue backs up, nothing publishes | High | The most likely real-world killer. Unreviewed drafts are still visible but clearly marked `DRAFT — UNREVIEWED`; review is lightweight approve-with-edits in the portal; review-queue SLA is monitored and escalates. |
| 4 | **Secret or IP leakage** — credentials or code reach a model or get published | Critical | Ingestion denylist + backstop secret scan; no-training in-tenant model deployment; SaaS buy option rejected on this boundary. |
| 5 | **Portfolio inconsistency** — each repo's context drifts to a different shape | Medium | The standard C4 template is enforced by the generator, not left to authors. |
| 6 | **Unparseable stack variant** — a repo uses Kotlin, Gradle, a different frontend | Medium | Parser coverage matrix; graceful degradation — partial facts, generation still runs with an explicit coverage caveat. |

---

## 11. Diagram Recommendation

A single **hybrid-view layered diagram** is warranted — it makes the human approval gate and the freshness loop visible, which prose alone buries. One diagram, not a set. The Mermaid source is delivered alongside this document (`Repo_Context_Platform_Architecture.mermaid`). It must show: the trigger sources, the deterministic parse step, the grounded generation step, the human review gate as an explicit decision node (approve / reject branches), publishing to both portal and repo, the consumption path, and the freshness loop feeding back to the trigger — with governance and observability drawn as connected planes, not detached boxes.

Visual polish (board-ready styling) is a downstream Module 2 pass; this diagram is engineering-grade.

---

## 12. Phased Rollout

- **Phase 1 — Pilot.** 3–5 representative repos. Manual trigger, generation → review → commit `docs/` back. Purpose: prove generation quality *and* measure the real review burden before committing to portfolio scale.
- **Phase 2 — Portfolio rollout.** CI-triggered refresh, Backstage TechDocs portal, deterministic freshness detection, search over published context.
- **Phase 3 — Optimise.** Incremental generation, tiered models, feedback-driven prompt tuning, code-hotspot enrichment. *Optional here:* a vector or graph index over published context if developers want semantic search across the portfolio — this is where the earlier Qdrant/Neo4j thread re-enters, as an enhancement, not the spine.

---

## 13. Lifecycle Coherence

*Required by the role's Lifecycle Coherence Gate — a design is a baseline, not a deliverable.*

- **First review trigger.** After the Phase 1 pilot, before Phase 2 portfolio rollout.
- **What the review produces.** A Module 5 **Production Readiness Review** — a findings report on the security boundaries, the approval gate, drift detection, secret handling, and rollback. Verdict gates the rollout.
- **What triggers a redesign.**
  - A confidently-wrong-context artifact reaches and misleads a real developer decision (DOC breach).
  - Review-queue SLA is chronically breached — the human-in-the-loop model does not scale and the gate needs rethinking.
  - A new major stack enters the portfolio (e.g., a Python or Go service) that the parser layer was not designed for.
  - Generation cost materially exceeds the §9 model.

---

## 14. Implementation Handoffs

This design stays at architecture level. The following are downstream implementation-specification work and are flagged, not produced here:

- Per-language parser specs — tree-sitter grammars, JavaParser / Spring & JPA introspection, React AST extraction, migration-file parsing.
- The grounded-generation prompt templates and the C4 artifact schema.
- Backstage TechDocs plugin configuration and the developer-portal information architecture.
- CI workflow definitions and the webhook/trigger wiring.
- The drift-threshold tuning per artifact type.

---

## 15. Acceptance Criteria for Systems Review

A later Module 5 review should verify, in priority order:

1. **DOC preserved** — no artifact is published without human approval; every artifact carries its source commit; stale artifacts are visibly flagged.
2. **Boundaries** — read-only repo-scoped access; ingestion denylist active; no-training in-tenant model deployment confirmed.
3. **Review gate** — approval authority bound to repo ownership; unreviewed drafts unambiguously marked.
4. **Determinism split** — drift detection is deterministic; the LLM influences only generated content.
5. **Observability** — coverage, freshness, review-queue health, cost, and feedback are all instrumented.
6. **Portfolio consistency** — every repo's context conforms to the standard C4 template.
7. **Failure-mode coverage** — each failure mode in §10 has a live detection or mitigation path.

---

*Module-delegation transparency: pattern selection and the layered structure follow Module 5 v1.2 (AI Systems Review); the Defining Operational Constraint, failure-mode table, and acceptance criteria follow Module 8 v1.1 (AI Agent Blueprint) discipline; the clarification, build-vs-buy, threshold-framing, and lifecycle-coherence gates are the AI Engineering Architect role delta v1.2.*
