# Template — tool-risk register

Tool risk is the #1 real-world agent risk (OWASP ASI02 Tool Misuse, ASI03 Privilege Abuse, LLM06
Excessive Agency). Every agent with tools gets a register. One row per tool. A `high` row with no
guardrail or no HITL is a **Blocker** finding.

| Tool | Purpose | Class (read / write / destructive) | Data accessed | Permission scope (least-privilege) | Failure mode | Guardrail (block/flag/confirm, at the side effect) | HITL required? | Audit event |
|---|---|---|---|---|---|---|---|---|
| `example_read` | look up X | read | non-sensitive | scoped read token | stale data | output validation | no | `tool_call` logged |
| `example_write` | mutate Y | write | customer record | one record, one action | wrong record updated | confirm + post-write validation | yes (irreversible) | `tool_call` + `approval` |

## Risk-tiering rule
- **read** → low (informational).
- **write** → high (state change) — needs a guardrail at the call and usually HITL if irreversible.
- **destructive / financial / external-comms** → high — **always** HITL, scoped token, audit event, and
  a kill-switchable path.

## Lethal-trifecta flag (fill once per agent)
- Holds private/sensitive data? ☐   - Exposed to untrusted content (tool/RAG/email)? ☐   - Can
  communicate externally? ☐
- **If all three are checked → remove or gate one leg** before release (dual-LLM / quarantine, or HITL
  on the external action). Record the decision here.

## Register summary
- Tools total: __ · high-risk: __ · with guardrail: __ · with HITL: __ · with audit event: __
- Any high-risk tool missing a guardrail or HITL? → **Blocker**; cannot pass the release gate.
