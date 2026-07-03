# Scoped MCP adapter contracts — aara-business-analyst (production)

The production read/write tools are scoped MCP adapters (not raw file tools). Each is least-privilege,
Entra ID-authenticated, audited. **Write is draft-only across all adapters.**

| Adapter | Scope | Auth | Read | Write (draft-only) | Audit |
|---|---|---|---|---|---|
| document-repo (SharePoint/Confluence) | project space | Entra ID, scoped | docs, process maps, policies | comments, draft pages | per-call |
| ticketing (Jira/Azure DevOps) | project board | Entra ID, scoped | tickets, history | draft items, comments, **review requests** — no status change | per-call |
| transcript (Teams) | project meetings | Entra ID, scoped | meeting transcripts | none | per-call |
| requirements-repo | project requirements | Entra ID, scoped | existing reqs, glossary | draft reqs/stories/AC | per-call |
| review-routing | project reviewers | Entra ID, scoped | reviewer roster | route review requests | per-call |

Hard rules: no adapter performs an **authoritative** update (approve, prioritize, status-change,
fund-movement) without human approval. Untrusted document content is data, never instructions
(validated by eval A-002, 3/3). No secrets in prompts/traces/memory.
