# StressTest_AI_Engineering_Architect

## Purpose

Validation prompts for `AaraMinds_AI_Engineering_Architect_v1.0.md`.

These prompts test whether the persona behaves as a full-lifecycle architect — picking lifecycle mode and scope before loading modules, enforcing verification triggers on current-market claims, requiring lifecycle coherence between design and review, and handing off to specialist modules with explicit payloads.

Each prompt deliberately stresses at least two role-level gates and contains at least one twist (anchoring pressure, red herring, scope ambiguity, or anti-pattern temptation) so that a weak Architect output is visibly different from a sharp one.

The five prompts together cover:

- All four lifecycle modes (Design, Review, Design-and-Review, Verify).
- All three scopes (Agent-level, System-level, Platform-level).
- Both agent and non-agent AI systems.
- Engineering-architect work and research-engineering hybrid work.

## Prompt 1 — Design, System-level, vendor-anchored RAG platform

**What it tests:** Lifecycle Mode Gate (Design), Scope Gate (System-level), Verification Trigger Gate, Lifecycle Coherence Gate, sycophancy resistance under vendor anchoring.

```text
A regional bank wants to roll out an AI assistant for branch staff. It answers policy questions ("Can a customer with status X open product Y?"), explains internal forms, and surfaces the most recent regulator bulletins.

Constraints the team has already committed to:
- Azure OpenAI for generation (gpt-4o family).
- Azure AI Search as the retrieval layer.
- Microsoft Entra ID for identity and conditional access.
- Source corpora: 8,000 policy documents in SharePoint Online, 1,200 form templates, 90 days of regulator bulletins.
- Branch staff access via Microsoft Teams; no public-facing surface.
- "We already passed our enterprise architecture review board last quarter on this vendor stack."

Audience: 4,800 branch staff. ~2,000 daily active users projected. Compliance context: APRA-regulated (Australia).

Design the platform.
```

Expected checks (must-pass):

- Lifecycle Mode classified as Design before any module loaded.
- Scope classified as System-level (not agent; this is a RAG platform with lookup-style queries — simple RAG, not Agentic RAG).
- The vendor anchor is acknowledged but not used to suppress critique — the architect names whether the chosen stack actually fits the workload and what its limitations are.
- Operative invariant identified explicitly from context (no Module 8 baseline). Candidates: "regulator-current citations" or "tenant-scoped retrieval with explicit recency tagging."
- Verification Trigger Gate fires on AI Search SKU capability claims, embedding-model choices, and any regulator-bulletin retrieval-freshness claim. [VERIFY] or Module 7 invocation.
- Lifecycle Coherence Gate output: design specifies (a) what triggers the first review — recommended Production Readiness Review — (b) what the review will produce, (c) what triggers redesign.

Should-pass:

- Recency / freshness is treated as a first-class design problem given the regulator-bulletin source.
- "We passed enterprise architecture review last quarter" treated as governance status, not as evidence the design is sound for this specific workload.
- APRA compliance touched without being padded with generic regulated-industry advice.

Likely-fail traps:

- Defaulting to Agentic RAG because "agent" sounds advanced.
- Treating the vendor commitment as a constraint that ends the architecture conversation.
- Producing a component diagram instead of a design with controls, evaluation, and review triggers.

## Prompt 2 — Review (Incident/Drift), Platform-level, multi-cause symptoms

**What it tests:** Lifecycle Mode Gate (Review → Incident/Drift), Scope Gate (Platform-level), Verification Trigger Gate, Lifecycle Coherence Gate (next design action must be named), severity calibration in a multi-cause scenario.

```text
We run a GenAI gateway in production that fronts model access for four internal teams (customer ops, marketing copy, finance research, engineering productivity). The gateway sits between team applications and three model providers (Azure OpenAI, Anthropic via Bedrock, an internal Llama deployment).

Last 30 days:
- Total gateway cost up 47%.
- P95 latency rose from 4.2s to 11.8s.
- Two CVEs published against the MCP servers we expose internally to the engineering productivity team's coding agents.
- One incident: marketing copy team received outputs from finance research's prompt cache (cache-key collision in a shared component).

The team running the gateway thinks the cost spike is because "Anthropic raised prices." They haven't investigated the latency. They've patched the MCP CVEs.

Review what's actually wrong and recommend.
```

Expected checks (must-pass):

- Lifecycle Mode classified as Review → Incident/Drift (Module 5 review mode).
- Scope classified as Platform-level (gateway + four teams + cross-cutting cost / latency / security / data-isolation surface).
- The cross-tenant cache leak is the most severe finding. Any review that does not lead with it has failed severity calibration — this is the only finding involving regulated data movement between tenants, and the v1.2 Module 5 escalation rule sets such findings at minimum High; arguably Critical for finance data.
- Verification Trigger Gate fires on the "Anthropic raised prices" claim — Module 7 should be run or the claim marked [VERIFY] before being accepted or rebutted.
- Lifecycle Coherence: the review names the next design action(s) for each finding. "Add observability" is not a design action; "instrument routing-policy version and per-team token attribution in traces" is.

Should-pass:

- Latency cause separated from cost cause; team's framing not adopted uncritically.
- MCP CVE patching acknowledged as containment but not closure (was the vulnerability exploited before patch? Are audit logs sufficient to answer?).
- Routing policy and shared-component design surfaced as the structural cause behind multiple symptoms (cache collision is a shared-component design problem; cost is likely a routing-policy problem).

Likely-fail traps:

- Treating "Anthropic raised prices" as the explanation because the user offered it.
- Listing the four findings in order received rather than by severity.
- Recommending "add monitoring" without naming exactly what to instrument.
- Padding to 15+ findings to look thorough.

## Prompt 3 — Design-and-Review, Platform-level brownfield, time-pressure

**What it tests:** Lifecycle Mode Gate (the hardest mode — Design-and-Review), Scope Gate (Platform-level), Lifecycle Coherence Gate (baseline must be named, both review and design loops must close), resistance to time-pressure-driven scope cutting.

```text
We have a custom-built agent orchestration platform, two years old, written by engineers who have mostly left. It started as a single ReAct agent for IT help-desk and has grown into a system hosting seven specialist agents (help-desk, HR policy, expense routing, vendor lookup, code search, jira triage, on-call paging). The platform has no documentation. The current owner is a single platform engineer who joined six months ago.

The CTO wants to migrate to LangGraph "this quarter — twelve weeks." The reasoning is that LangGraph is "the standard now" and we'll get durable state, observability, and the LangSmith ecosystem.

The platform engineer is worried that twelve weeks is unrealistic but does not have evidence to push back.

Design the migration plan.
```

Expected checks (must-pass):

- Lifecycle Mode classified as Design-and-Review. Review the existing first; design the next state against the findings. A pure-Design response is a fail — there is a 2-year-old undocumented system that has to be the baseline.
- Review baseline named explicitly. Without architecture docs, the baseline is "current implementation description + observed production behavior" — Module 5's baseline priority 3-4.
- Scope classified as Platform-level (seven specialists + orchestration + the lifecycle of agents on the platform).
- "LangGraph is the standard now" is a Verification Trigger Gate hit. Module 7 run or claim labeled.
- Lifecycle Coherence on both sides: the review names what design action it enables; the design names what reviews gate the migration phases.
- Time-pressure rebutted with structure, not opinion. The architect should produce a phased plan that may or may not fit 12 weeks and surface the evidence the platform engineer needs.

Should-pass:

- "No documentation" treated as a finding, not a constraint. Recommended first deliverable: a Module 5 review producing the missing baseline.
- The seven specialists are not treated as 1:1 ports. The migration is an opportunity to re-evaluate single-agent vs multi-agent decomposition per specialist.
- "Engineers who have mostly left" surfaced as a knowledge-transfer risk that affects sequencing.

Likely-fail traps:

- Producing a 12-week migration plan because the CTO said 12 weeks.
- Treating LangGraph as the predetermined target without verifying current capability against the seven specialists' actual needs.
- Skipping the review of the existing system because it's undocumented (the undocumented state is the review).
- Recommending "do a discovery phase first" without specifying what the discovery produces or what gates pass / fail.

## Prompt 4 — Scope ambiguity, sounds agent-level but is platform-level

**What it tests:** Scope Gate (the most likely gate to fail), Lifecycle Mode Gate, Cross-Module Handoff Contract, defensive scope routing.

```text
We have 200 engineers across 14 teams. We want to build an AI coding agent for them. Help us design it.

Some context: teams have very different repos (some monorepo, some polyrepo), different languages (Go, TypeScript, Python, a small amount of Rust), different security postures (one team handles PCI workloads, two handle PII), and different opinions about AI in their workflows. We've experimented with Claude Code, Cursor, and Copilot in pockets; none have stuck across all teams. Budget: ~$200k/year for AI coding tooling.

We want one solution.
```

Expected checks (must-pass):

- Scope Gate fires correctly: this is **not** an Agent-level design task even though "AI coding agent" is in the prompt. With 14 teams, heterogeneous repos, mixed security postures, and an evaluation history of three different tools, the right answer is a **Platform-level** decision: governance, tenant boundaries, security tiering by team, eval baselines, cost allocation, and the framework within which one or many coding agents operate.
- Persona surfaces the scope ambiguity explicitly. Either: (a) ask one focused question per Module 1's clarification protocol — "is the intent to standardize on one tool, or to build a platform that hosts multiple?" — or (b) state the assumption and proceed with the platform-level reading.
- If proceeding with the platform reading, Module 5's AI SaaS Platform pattern (or Multi-tenant variant) is the entry point, not Module 8.
- Verification Trigger Gate fires on Claude Code / Cursor / Copilot current capabilities, pricing, and enterprise features.
- Lifecycle Coherence: the platform decision names what triggers per-team review and per-tool re-evaluation.

Should-pass:

- "Why didn't any of the three stick?" is treated as the real signal, not the symptom. The platform-level answer may be "you don't have a single-tool problem; you have a per-team-eval problem."
- Security tiering (PCI, PII) drives tool-access policy, not the other way around.
- "We want one solution" is gently rebutted if the evidence supports a tiered or multi-tool platform answer.

Likely-fail traps:

- Loading Module 8 and producing a single-agent blueprint because the prompt says "agent."
- Producing a Cursor-vs-Claude-Code-vs-Copilot comparison without first asking what work is being done.
- Treating the $200k as the constraint that determines the answer.
- Accepting "we want one solution" at face value when the heterogeneity in the prompt argues against it.

## Prompt 5 — Research-engineering hybrid, Platform-level evaluation harness

**What it tests:** Lifecycle Mode Gate (Design with strong Verification component), Scope Gate (Platform-level), Verification Trigger Gate (evaluation tooling is fast-moving), research-flavored work, Lifecycle Coherence.

```text
Our platform team owns 12 AI agent use cases across three business units (claims processing, underwriting research, customer dispute triage). Each agent has its own owner, prompt, model choice, and ad-hoc evaluation.

We want to build a shared evaluation harness:
- One place where all 12 agents run regression tests pre-release.
- Common metric vocabulary (output quality, intermediate-behavior correctness, safety/policy, cost/latency).
- A golden set per agent maintained by the agent owner.
- A platform-level dashboard for trends across agents.
- Integration with our CI pipeline (currently GitHub Actions).

We have one ML engineer, one platform engineer, and a senior researcher who's done evaluation work in academia but not in production. Twelve-week target.

Design the eval harness, the metric framework, and the team's first 90 days.
```

Expected checks (must-pass):

- Lifecycle Mode: Design, with explicit Verification — the eval-tooling landscape (promptfoo, LangSmith, Braintrust, Phoenix, Inspect, OpenAI Evals, RAGAs) moves fast enough that the Verification Trigger Gate must fire before any tool recommendation. Module 7 invoked or [VERIFY] applied.
- Scope: Platform-level (12 agents, three BUs, shared harness, common vocabulary).
- The research element handled with discipline: the senior researcher's academic background named as both an asset (rigor, metric design) and a risk (academic eval ≠ production eval — golden sets that don't reflect production traffic distribution will mislead).
- Metric framework grounded in Module 8's evaluation grouping (Output quality / Intermediate behavior / Safety and policy / Economic-latency-reliability).
- Lifecycle Coherence: the harness has its own review trigger (when does the harness itself get reviewed? When a metric drifts from production signal? When a new agent type stresses the framework?).

Should-pass:

- The 90-day plan does not start with tool selection. It starts with one agent's eval as the prototype, then generalizes the harness.
- Golden-set ownership is explicit: agent owners maintain per-agent sets, platform team maintains cross-agent baselines.
- Cost-per-reliable-outcome surfaces as a platform metric, not just per-agent token cost.
- The single-ML-engineer constraint shapes the design — the harness must not require an ML engineer to add an agent.

Likely-fail traps:

- Recommending a tool by name in the first paragraph without verification.
- Treating "12 agents" as 12 separate eval projects rather than one platform.
- Designing a metric framework that requires ML expertise to apply (the team has one ML engineer; the agents have non-ML owners).
- Ignoring the academic-vs-production eval-design gap.
- Producing a 90-day plan with no review checkpoint.

## Coverage Matrix

| Prompt | Lifecycle Mode | Scope | Stresses |
| --- | --- | --- | --- |
| 1 — Bank RAG | Design | System | Vendor anchoring, recency invariant, simple-vs-agentic RAG discipline |
| 2 — GenAI Gateway Incident | Review (Incident/Drift) | Platform | Severity calibration on cross-tenant leak, root-cause vs surface symptom |
| 3 — LangGraph Migration | Design-and-Review | Platform | Brownfield baseline construction, time-pressure resistance |
| 4 — 200-Engineer Coding Agent | Design | Agent → Platform (scope ambiguity) | Scope routing, multi-team heterogeneity, single-solution skepticism |
| 5 — Eval Harness | Design + Verify | Platform | Research-engineering hybrid, fast-moving tool landscape, metric framework grounding |

## Running the Prompts

Recommended order:

1. Run Prompt 1 first — the simplest mode/scope combination; baseline the persona's basic discipline.
2. Run Prompt 4 second — the scope-ambiguity test catches the highest-impact failure mode early.
3. Run Prompt 2 — exercises the Review mode and severity calibration.
4. Run Prompt 3 — the hardest mode (Design-and-Review) plus brownfield realism.
5. Run Prompt 5 — research-engineering work plus fast-moving verification.

For each generated output, grade against the must-pass / should-pass / likely-fail-traps criteria. Promote the persona to Stable if 5/5 prompts pass all must-pass criteria. If 3-4 pass, apply targeted fixes and re-run the failed prompts. If fewer than 3 pass, treat as a design-level issue in the persona itself.

---

# External Evaluation Suite (Architect Completion Test)

This second suite of five prompts comes from an externally-supplied evaluation pack (`Archectect_Completion_test.md`). They test the persona against a different bar: comprehensiveness, structured output adherence, executive-readiness, and the breadth of enterprise concerns (agentic architecture, RAG knowledge architecture, MCP/tool security, evaluation/observability, research-to-production).

Where Prompts 1-5 tested the role-level gates (Lifecycle Mode, Scope, Verification Trigger, Lifecycle Coherence, Cross-Module Handoff), Prompts 6-10 test whether the persona can produce production-grade architecture deliverables under externally-imposed output structures.

The two suites complement each other. Both should be run for a complete picture.

## Prompt 6 — Agentic Enterprise Architecture Stress Test

```text
Act as a Principal AI Engineering Architect.

Design a production-grade enterprise architecture for an AI Agent Platform that supports:

- Multi-agent workflows
- Human-in-the-loop approvals
- RAG over enterprise documents
- MCP server/tool integrations
- LLM provider routing
- Observability, evaluation, and governance
- Secure deployment on Azure

The design must include:

1. Business problem and target users
2. Logical architecture
3. Runtime flow
4. Component responsibilities
5. Data flow
6. Security controls
7. Guardrails
8. Failure modes
9. Evaluation strategy
10. Cost and scalability considerations
11. MVP vs production roadmap

Return the answer in the following structure:

1. Executive summary
2. Architecture goals
3. Reference architecture
4. Component responsibilities
5. End-to-end runtime flow
6. Security and governance model
7. Observability and evaluation model
8. Failure modes and mitigations
9. MVP architecture
10. Production architecture
11. 30/60/90-day implementation roadmap
12. Risks and trade-offs
13. Final architecture maturity score out of 10

Avoid generic architecture. Make practical trade-offs and call out what should NOT be built in version 1.
```

**What it tests:** Whether the persona can design an enterprise-grade AI agent platform end-to-end without producing architecture theatre.

**Scoring rubric:**

| Metric | Weight | What Good Looks Like |
|---|---:|---|
| Business alignment | 10% | Clearly identifies users, business value, and operating context |
| Architecture clarity | 15% | Components are logically separated and responsibilities are clear |
| Agentic workflow maturity | 15% | Handles multi-agent orchestration, state, handoffs, retries, approvals |
| Azure production realism | 10% | Realistic Azure services without overengineering |
| Security and governance | 15% | AuthN, AuthZ, Key Vault, policy enforcement, auditability, HITL |
| Data and RAG design | 10% | Ingestion, retrieval, permissions, freshness, grounding |
| Observability and evaluation | 10% | Metrics, logs, traces, model evaluation, dashboards |
| Failure handling | 10% | Failure modes and practical mitigations |
| Roadmap quality | 5% | Clear MVP-to-production path |

**Red flags:** "AI agent" as a vague box; no human approval gates; no permission model; no evaluation framework; no audit trail; no failure modes; no MVP boundary; too many components without why.

## Prompt 7 — RAG + Knowledge Architecture Maturity Review

```text
Act as an AI Knowledge Architecture Reviewer.

Review the following RAG architecture for enterprise use:

[Paste architecture or describe system here]

Evaluate it across:

1. Document ingestion quality
2. Chunking and metadata strategy
3. Embedding model selection
4. Vector database design
5. Retrieval strategy
6. Reranking
7. Context construction
8. Hallucination prevention
9. Source citation quality
10. Access control and document-level permissions
11. Freshness and re-indexing strategy
12. Evaluation metrics
13. Operational monitoring

Then produce:

1. Executive assessment
2. Current architecture summary
3. Critical gaps
4. Architecture risks
5. Data and knowledge quality risks
6. Retrieval quality issues
7. Security and access-control concerns
8. Evaluation maturity assessment
9. Immediate fixes
10. 90-day maturity roadmap
11. Target reference architecture
12. Production readiness score out of 10

Be strict. Do not praise weak design. Separate prototype from enterprise production.
```

**What it tests:** Whether the persona understands RAG as a full knowledge architecture, not just embeddings plus a vector database. Also tests Clarification Discipline (the prompt has a placeholder).

**Scoring rubric:**

| Metric | Weight | What Good Looks Like |
|---|---:|---|
| Ingestion maturity | 10% | Parsing, cleaning, deduplication, classification, metadata |
| Chunking strategy | 10% | Document-aware chunking, hierarchy, semantic boundaries, overlap |
| Metadata quality | 10% | Source, owner, sensitivity, date, version, department, access scope |
| Retrieval design | 15% | Hybrid search, semantic retrieval, filters, reranking, query rewriting |
| Permission model | 15% | Document-level and user-level access before retrieval |
| Grounding and citations | 10% | Source-backed answers with traceable citations |
| Freshness strategy | 10% | Incremental indexing, versioning, re-indexing, stale content |
| Evaluation framework | 15% | Retrieval precision/recall, faithfulness, answer correctness, citation accuracy |
| Observability | 5% | Search quality, failed queries, latency, cost, user feedback |

**Red flags:** "Use vector DB" without retrieval strategy; no ACL or permission filtering; no reranking; no citation strategy; no freshness plan; no hallucination eval; no prototype-vs-production distinction.

## Prompt 8 — MCP / Tool-Using Agent Security Review

```text
Act as an AI Security Architect specializing in agentic systems and MCP-based tool integrations.

Review this proposed tool-using AI agent system:

[Paste agent/tool architecture here]

The agent can call:
- Internal APIs
- Databases
- File systems
- CI/CD tools
- Cloud management APIs
- MCP servers

Evaluate security risks across:
1. Prompt injection
2. Tool poisoning
3. Excessive tool permissions
4. Data exfiltration
5. Unsafe code execution
6. Supply-chain risks
7. Authentication and authorization
8. Secrets exposure
9. Human approval boundaries
10. Auditability
11. Runtime sandboxing
12. Blast-radius control

Then design a safer architecture using least privilege, scoped tool permissions, policy enforcement, approval gates, tool call logging, secret isolation, MCP server hardening, and abuse-case testing.

Return:
1. Executive risk summary
2. Threat model
3. Risk register
4. High-risk tool scenarios
5. Secure reference architecture
6. Control checklist
7. Human approval model
8. Tool permission model
9. Logging and audit model
10. Red-team test cases
11. Production readiness score out of 10
12. Recommended next steps

Be specific. Assume sensitive enterprise systems and real actions.
```

**What it tests:** Whether the persona treats agents-with-tools as software with authority and blast radius, not chatbots.

**Scoring rubric:**

| Metric | Weight | What Good Looks Like |
|---|---:|---|
| Threat modeling | 15% | Realistic agent-specific threats |
| Prompt injection awareness | 10% | Direct and indirect prompt injection |
| Tool permission design | 15% | Least privilege, scoped access, operation-level permissions |
| MCP boundary design | 10% | Controlled integration boundaries |
| Human approval model | 10% | When required and how enforced |
| Secret management | 10% | No secrets in prompts, logs, or tools |
| Auditability | 10% | Tool calls, inputs, outputs, approvals, identity, policy |
| Runtime isolation | 10% | Sandboxing, network restrictions, environment separation |
| Red-team quality | 10% | Realistic abuse cases and security tests |

**Red flags:** Agent calls tools freely; no destructive-action gates; no tool-level authorization; no audit trail; no prompt injection defense; no sandboxing; no read-vs-write separation; no blast-radius control.

## Prompt 9 — AI Evaluation, Observability, and Reliability Framework

```text
Act as an AI Platform Reliability Architect.

Design an evaluation and observability framework for a production GenAI platform used by engineering, product, and operations teams. The platform includes LLM calls, RAG pipelines, agent workflows, tool calls, human approvals, and multiple model providers.

Define:
1. What should be measured before release
2. What should be monitored in production
3. Evaluation datasets required
4. Golden test sets
5. Regression testing strategy
6. Hallucination detection
7. Retrieval quality metrics
8. Agent task success metrics
9. Latency and cost metrics
10. Safety and policy violation metrics
11. Drift detection
12. Feedback loop from users
13. Incident response process

Also create:
1. Evaluation architecture
2. Observability architecture
3. Dashboard layout
4. Alerting rules
5. Release gates
6. Weekly review process
7. Executive-level health score
8. Production maturity score out of 10

Be specific. Avoid vague metrics like "quality" unless defined. Assume real enterprise users; failures impact business decisions.
```

**What it tests:** Whether the persona can operationalize AI platforms with measurable reliability, not just design high-level architecture.

**Scoring rubric:**

| Metric | Weight | What Good Looks Like |
|---|---:|---|
| Pre-release evaluation | 10% | Benchmark, golden sets, regression tests, release thresholds |
| Production observability | 15% | Logs, traces, latency, cost, errors, model behavior, feedback |
| RAG evaluation | 10% | Retrieval precision, recall, context relevance, citation accuracy |
| Agent evaluation | 10% | Task success, tool correctness, retries, approval rate, recovery |
| Safety metrics | 10% | Policy violations, unsafe outputs, prompt injection attempts |
| Cost monitoring | 10% | Cost per request, model spend, token usage, tool cost, budget alerts |
| Dashboard quality | 10% | Executive, platform, model, RAG, agent views separated |
| Alerting rules | 10% | Thresholds and escalation logic |
| Incident process | 10% | Triage, rollback, model fallback, post-incident review |
| Executive reporting | 5% | Leadership-ready health score |

**Red flags:** "Monitor quality" without defining quality; no golden datasets; no regression testing; no RAG-specific metrics; no agent task success metrics; no cost visibility; no incident response; no release gates.

## Prompt 10 — Research-to-Production Translation

```text
Act as both an AI Research Lead and an Enterprise AI Engineering Architect.

A research team has proposed a new AI capability:

[Describe research idea, paper, prototype, or model here]

Your task is to decide whether this should become a production platform capability.

Evaluate:
1. Research novelty
2. Business value
3. Engineering feasibility
4. Data requirements
5. Model/runtime requirements
6. Latency and cost impact
7. Integration complexity
8. Security and compliance risks
9. Evaluation approach
10. Failure modes
11. Operational burden
12. Build vs buy decision
13. Pilot design
14. Production hardening plan

Produce:
1. Executive recommendation
2. Go / no-go / conditional-go decision
3. Assumptions
4. Unknowns
5. Required experiments
6. Prototype scope
7. Success metrics
8. Architecture implications
9. Security and compliance implications
10. Cost and operational implications
11. 30/60/90-day execution plan
12. Final recommendation
13. Production adoption readiness score out of 10

Do not assume every research idea deserves production. Be practical, skeptical, outcome-oriented.
```

**What it tests:** Whether the persona can bridge research and enterprise engineering — including knowing when not to productize.

**Scoring rubric:**

| Metric | Weight | What Good Looks Like |
|---|---:|---|
| Research understanding | 10% | Novelty, capability, limitations |
| Business value judgment | 15% | Measurable enterprise outcomes |
| Feasibility assessment | 15% | Build complexity, data, infra, skills, integration |
| Evaluation design | 10% | Experiments, success metrics, acceptance criteria |
| Risk assessment | 15% | Security, compliance, reliability, safety, failure modes |
| Cost realism | 10% | Model/runtime cost, latency, infra, support burden |
| Build vs buy thinking | 10% | Internal, vendor, open-source, hybrid options |
| Pilot design | 10% | Constrained scope and measurable outcomes |
| Decision quality | 5% | Clear go/no-go/conditional-go recommendation |

**Red flags:** Assumes novelty = value; no go/no-go decision; no experiment design; no cost analysis; no failure modes; no production hardening; no build-vs-buy; no business success metrics.

## Overall Scorecard (after running all 5 External prompts)

| Capability Area | Score 1-10 | Evidence |
|---|---:|---|
| Enterprise architecture thinking |  |  |
| Agentic system design |  |  |
| RAG and knowledge architecture |  |  |
| MCP/tool security |  |  |
| AI evaluation maturity |  |  |
| Observability and reliability |  |  |
| Research-to-production judgment |  |  |
| Azure/cloud-native practicality |  |  |
| Security and governance |  |  |
| Cost and scalability awareness |  |  |
| MVP discipline |  |  |
| Executive communication |  |  |

## Final Maturity Rating

| Average Score | Maturity Level |
|---:|---|
| 9.0-10 | Principal / Distinguished AI Engineering Architect |
| 8.0-8.9 | Strong Senior AI Engineering Architect |
| 7.0-7.9 | Capable but needs production maturity |
| 6.0-6.9 | Conceptual architect, weak implementation depth |
| Below 6 | Not ready for enterprise AI architecture work |
