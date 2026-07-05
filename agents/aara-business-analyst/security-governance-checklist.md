# Security & Governance Checklist — aara-business-analyst

OWASP Top 10 for Agentic Applications (2026) mapping. Titles follow the official ASI01–ASI10 taxonomy. Status values: addressed / mitigated / not-applicable-with-reason / TODO.

## ASI01 Agent Goal Hijack

Status: TODO. Control: objectives fixed in system prompt; external content cannot alter goals (prompt-injection rule); manifest pins skills and tools.

## ASI02 Tool Misuse & Exploitation

Status: addressed by platform. Control: manifest allowlist + pinned contract versions; off-manifest and blocked calls denied and audited (tool-denial gates).

## ASI03 Agent Identity & Privilege Abuse

Status: TODO pending identity provisioning. Control: per-agent identity, scoped short-lived credentials, no shared accounts (agent-identity-spec.json).

## ASI04 Agentic Supply Chain Compromise

Status: TODO. Control: skills and contracts are versioned and pinned in the manifest; no unpinned dependencies at runtime; platform CI runs govulncheck on its own dependencies.

## ASI05 Unexpected Code Execution

Status: TODO review. Control: no arbitrary code-execution tools proposed in this intake; adding one requires a hard boundary and security sign-off.

## ASI06 Memory & Context Poisoning

Status: addressed by platform. Control: engagement-scoped memory, citation-required writes, leakage tests fail closed (memory gates).

## ASI07 Insecure Inter-Agent Communication

Status: not-applicable-with-reason — single-agent design; A2A interop is out of MVP scope. Revisit if multi-agent patterns are introduced.

## ASI08 Cascading Agent Failures

Status: TODO. Control: contract timeout and retry policies; fail-safe denial semantics; kill switch per agent/tool.

## ASI09 Human-Agent Trust Exploitation

Status: TODO. Control: output separates facts from assumptions; evidence rules prevent confident uncited claims; RAG-status honesty rules where applicable.

## ASI10 Rogue Agents

Status: addressed by platform. Control: manifest status lifecycle, kill switch, audit chain verification; agents cannot self-modify manifests.

## RBAC Summary

Reviewer roles per readiness report approvals_required. [TODO security reviewer: confirm role-to-scope mapping.]

## Data Classification Summary

Highest classification handled: medium-tier agent; data sensitivity input: client-confidential. See data-and-evidence-contract.md.

## Audit Obligations

100% of governed actions audited (tool calls, denials, approvals, memory writes, releases). Audit chain is tamper-evident and replayable.

## Kill-Switch Path

Platform admin disables the agent via manifest status change to blocked; tool-level disable via contract removal from allowlist. [TODO: name the accountable admin.]
