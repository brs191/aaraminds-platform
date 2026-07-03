# Tool-risk register — aara-business-analyst

| Tool | Purpose | Class | Data accessed | Scope | Failure mode | Guardrail | HITL? | Audit |
|---|---|---|---|---|---|---|---|---|
| Read/Glob/Grep | ingest evidence | read | stakeholder docs (untrusted), tickets, policies | workspace/scoped repo read | reads tainted doc | treat as untrusted; trace-or-[VERIFY] | no | logged |
| Write/Edit | draft artifacts/comments/review-requests | write (draft-only) | draft store | drafts only — never systems of record | over-reach to authoritative update | human-approval gate; no SoR write | yes (any authoritative update) | logged |

Trifecta: private data (moderate) · **untrusted content YES** (ingested docs) · external comms (low) →
not full trifecta, but the untrusted-content leg is real. Mitigation: write is draft-only + human-gated;
no tool can change a system of record without approval. Document in the deployment threat model (F-002).
No `Bash` / shell. Production: scoped MCP adapters (doc/ticketing/transcript/requirements/review-routing)
with Entra ID + audit.
