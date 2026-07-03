# StressTest_Module5_SystemsReview

## Purpose

Validation prompts for the re-scoped Module 5: `05_AI_Architecture_Diagram_System_v1.1.md`.

The module now acts as the AaraMinds AI Systems Review System.

These prompts test whether it behaves as a diagnostic reviewer rather than a diagram generator.

## Prompt 1 — Blueprint Conformance Review

```text
Review this deployed Business Analyst Agent against the Module 8 blueprint baseline.

Blueprint constraint: Traceability-by-Construction.

Current implementation:
- Ingests Teams transcripts, SharePoint docs, and Jira tickets.
- Drafts user stories and acceptance criteria.
- Creates Jira tickets after product owner approval.
- Stores project memory and glossary terms.
- Routes security-sensitive requirements to a security reviewer.

Known issue:
Two unsupported requirements reached sprint planning last week.
The team has traces for model calls and tool calls, but no trace that links each requirement back to source evidence.
```

Expected checks:

- Uses Blueprint Conformance Review.
- Leads with findings.
- Flags missing source-to-requirement trace as High or Critical depending on impact.
- Tests preservation of Traceability-by-Construction.
- Checks write-path control around Jira creation.
- Reviews evidence, approvals, reviewer routing, observability, and eval gates.
- Provides owner and re-review trigger for each major finding.
- Does not produce a decorative diagram as the primary output.

## Prompt 2 — Production Readiness Review

```text
Assess this Agentic RAG system before production launch.

The system answers customer dispute questions using policy documents, CRM notes, and order history.
It can recommend refunds, but humans approve payment actions.
Architecture includes document ingestion, vector search, reranker, LLM answer generation, CRM lookup tool, order lookup tool, and refund recommendation workflow.
```

Expected checks:

- Uses Production Readiness Review.
- Checks identity, RBAC, PII, source grounding, retrieval policy, CRM/order-history boundaries, human approval, audit, rollback, cost, latency, and evaluation.
- Flags any missing approval, tool boundary, or audit path.
- States whether the system is ready, conditionally ready, or blocked.
- Includes failure modes and remediation priority.

## Prompt 3 — Incident / Drift Review

```text
Our internal coding assistant was stable for two months, but last week tool-call cost doubled and latency rose from 8s to 24s P95.
No one changed the model manually.
The system uses a model router, repo search, code execution sandbox, and PR comment tool.
We have traces, but they do not include routing policy version or retrieved context size.
Review what is structurally wrong.
```

Expected checks:

- Uses Incident / Drift Review.
- Identifies missing routing-policy and context-size telemetry as observability gaps.
- Reviews cost, latency, model routing, repo retrieval, sandbox use, and PR write path.
- Separates evidence from assumptions.
- Recommends immediate containment and deeper fix.
- Names re-review triggers.

## Prompt 4 — Diagram Review Pressure Test

```text
Review this AI platform diagram.

It has users, app layer, AI agent layer, model router, vector DB, tools, and monitoring.
The diagram looks polished, but it does not show identity, permissions, policy checks, evaluation, approval, rollback, or failure paths.
```

Expected checks:

- Uses Diagram Review.
- Says the diagram is visually polished but architecturally incomplete.
- Flags missing identity, permissions, policy, evaluation, approval, rollback, and failure paths.
- Recommends diagram changes that expose decisions, boundaries, flows, controls, and failure modes.
- Does not treat visual polish as architecture quality.

## Prompt 5 — Hardened Production Readiness (Module 5 v1.2 calibration test)

This prompt is designed to stress Module 5 v1.2 in ways Prompt 1 did not — no Module 8 blueprint to lean on (forcing in-context DOC identification), mixed severity profile, surface polish, anchoring pressure, regulated data, multi-tenant complexity, and a red herring.

```text
Assess this Agentic RAG platform before production launch.

The system is "ClauseScan," an enterprise legal-AI platform for contract review.
It serves five enterprise tenants today (pre-launch private preview); GA is planned in four weeks.

What it does:
- A lawyer uploads a contract (PDF or DOCX) and selects a review playbook (M&A, vendor, employment, NDA).
- An agent retrieves matching clauses from the tenant's prior contracts, statutes, and the firm's playbook commentary.
- The agent produces a redlined clause-by-clause review with citations to the retrieved sources.
- The lawyer accepts, edits, or rejects each suggestion.
- Approved redlines export back to Word as tracked changes.

Architecture as described by the team:
- Entra ID for AuthN; RBAC scoped by tenant_id and matter_id.
- One Azure OpenAI deployment per tenant for the generation model; shared embedding model across tenants.
- One shared Azure AI Search index per document_class (contracts, statutes, playbook commentary), with tenant_id metadata filter applied at query time.
- Chunking: 1,200-token chunks with 200-token overlap, semantic boundary aware.
- Retrieval: hybrid (BM25 + vector) with reranker; top-k=12; metadata filter on tenant_id and document_class.
- Generation: model produces redline + citation list; a verifier pass checks that each cited chunk_id appears in the retrieved set.
- Audit logs in Log Analytics: source access, tool calls, generation events, accept/reject events. 90-day retention.
- Traces via OpenTelemetry to Application Insights: retrieval spans, model calls, verifier outcomes, tool calls.
- Evaluation: golden set of 200 historical contracts with known correct redlines per tenant; pass@1 redline-correctness scorer; runs nightly.
- Kill switch: feature flag disables the generation endpoint per tenant.
- Rollback: prompt and model versions are pinned per tenant; revert is a config push.

Team additions worth noting:
- "We already passed SOC 2 Type 1 last quarter and Type 2 audit is scheduled for Q3."
- "Our embedding model is shared across tenants because per-tenant embeddings would 5x our cost — we evaluated this trade-off and metadata filtering is sufficient."
- "We considered prompt-injection defenses in citation grounding but our verifier pass catches mismatched citations, so we deprioritized."
- "Provider concentration is a known risk — we're on Azure OpenAI only — but we have a fallback plan to Anthropic via Bedrock if needed."

What we want: production-readiness verdict, gating findings, and remediation priority before GA.
```

Expected checks (accuracy criteria for grading the generated output):

Must-pass:

- Production Readiness mode selected per v1.2 selector table.
- Operative DOC identified from context (no Module 8 baseline given) — likely shape: "citation-grounded redlines must trace to retrieved tenant-scoped evidence."
- Shared-embedding-across-tenants flagged at least High, with v1.2 regulated-data escalation rule cited.
- Prompt-injection-via-retrieved-chunks flagged at least High; team's "verifier catches it" claim rebutted concretely (verifier checks IDs, not content).
- SOC 2 Type 1 anchor acknowledged but does not soften any finding; sharp output may note Type 1 ≠ Type 2 scope.
- Provider concentration omitted, called Low, or framed as "acknowledged and scoped" — not flagged High.
- Findings within 7-12 budget, with theme grouping where multiple sub-issues share a root.
- Verdict is Conditionally ready or Blocked — not Ready with monitored risks, not all-fine.

Should-pass:

- 90-day audit log retention flagged as likely insufficient for legal data.
- Per-tenant model deployment called out as a positive control (sharp reviews note what is right, not only what is wrong).
- Tenant golden-set evaluation noted as adequate for preview but needs scaling plan for GA.
- Verifier pass crisply distinguished: catches citation-ID mismatch, not injected content.

Likely-fail traps for a weak reviewer:

- Listing provider concentration as High based on pattern-matching.
- Treating SOC 2 Type 1 as evidence of operating effectiveness.
- Accepting "metadata filtering is sufficient" because the team said they evaluated the trade-off.
- Padding to 20+ findings to look thorough.
- Recommending "add prompt injection defense" without grounding the finding in the specific retrieved-chunks-as-injection-vector path.
