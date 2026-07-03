# Tool-risk register — aara-copilot-cost-reviewer

| Tool | Purpose | Class | Data accessed | Scope | Failure mode | Guardrail | HITL? | Audit |
|---|---|---|---|---|---|---|---|---|
| Read/Glob/Grep | ingest billing report + metrics export | read | **sensitive** per-user Copilot spend, usage | scoped repo/workspace read | reads stale/partial data | source-or-[VERIFY]; re-verify rates | no | logged |
| Write/Edit | draft the cost review / recommendations | write (draft) | output folder | drafts only — never admin actions | over-reach to enacting a control | recommend-don't-enact; human-only gate | yes (any admin change) | logged |

Trifecta: sensitive data (yes) · untrusted content (low — internal dashboards) · external comms (low) →
not full trifecta. No `Bash`. **Confidentiality:** per-user spend is PII-adjacent — build only in the
approved workspace, no export, mark internal. Production: scoped GitHub OAuth; per-user cost needs
enterprise access.
