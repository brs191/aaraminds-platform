# Security & Governance Checklist — aara-psql-expert

OWASP Top 10 for Agentic Applications (2026) mapping. Titles follow the official ASI01–ASI10 taxonomy. Status values: addressed / mitigated / not-applicable-with-reason / TODO.

## ASI01 Agent Goal Hijack

Status: addressed. Control: objectives fixed in system prompt; external content (schema files, comments, retrieved docs) is treated as data, not instructions (prompt-injection rule, golden case GC-12); manifest pins skills and tools so goals cannot be redirected by tool substitution.

## ASI02 Tool Misuse & Exploitation

Status: addressed by platform. Control: manifest allowlist + pinned contract versions; off-manifest and blocked calls denied and audited (tool-denial gates).

## ASI03 Agent Identity & Privilege Abuse

Status: mitigated. Control: identity spec complete (agent-identity-spec.json) — per-agent Entra Agent ID, scoped short-lived federated credentials (<= 24h), no shared accounts, no live-database access (read tools operate on provided files only). Live Entra provisioning is a deployment-time step tracked in the lifecycle section.

## ASI04 Agentic Supply Chain Compromise

Status: addressed. Control: skills and tool contracts are versioned and pinned in the manifest (contract_version 1.0.0); no unpinned dependencies at runtime; platform CI runs govulncheck on its own Go dependencies on every change.

## ASI05 Unexpected Code Execution

Status: not-applicable-with-reason. This agent produces SQL/PL-pgSQL as text drafts only and executes nothing — no tool in the manifest runs SQL or code against any system. Adding an execution tool (e.g. read-only query or migration apply) would be a separate agent version requiring a hard approval boundary and security sign-off; it is out of scope for this advise-and-draft agent.

## ASI06 Memory & Context Poisoning

Status: addressed by platform. Control: engagement-scoped memory, citation-required writes, leakage tests fail closed (memory gates).

## ASI07 Insecure Inter-Agent Communication

Status: not-applicable-with-reason — single-agent design; A2A interop is out of MVP scope. Revisit if multi-agent patterns are introduced.

## ASI08 Cascading Agent Failures

Status: addressed. Control: single-agent design with no downstream agents to cascade into; each tool contract sets timeout and retry policy; denials fail safe (closed); per-agent and per-tool kill switch available to the platform admin.

## ASI09 Human-Agent Trust Exploitation

Status: mitigated. Control: output structure separates verified facts (each cited to retrieved DDL) from assumptions, risks, and open questions; the evidence contract blocks confident uncited schema claims (golden case GC-07 hallucination guard); the agent surfaces what it could not verify rather than asserting it.

## ASI10 Rogue Agents

Status: addressed by platform. Control: manifest status lifecycle, kill switch, audit chain verification; agents cannot self-modify manifests.

## RBAC Summary

Reviewer roles per readiness report approvals_required: business-owner (scope approval), enterprise-ai-architect (design approval), and security-reviewer (approval of the soft-gated write tool create_sql_draft). Read tools (get_schema_context, search_sql_knowledge) carry no approval boundary. Runtime role is least-privileged and does not own the tool contracts.

## Data Classification Summary

Highest classification handled: medium-tier agent; data sensitivity input: client-confidential. See data-and-evidence-contract.md.

## Audit Obligations

100% of governed actions audited (tool calls, denials, approvals, memory writes, releases). Audit chain is tamper-evident and replayable.

## Kill-Switch Path

Accountable admin: Raja Shekar Bollam (acting engineering lead / platform admin). Disables the agent via manifest status change to blocked; tool-level disable via contract removal from the manifest allowlist. The activation gate independently prevents any agent from reaching active status without a current pass verdict.
