# Tool-risk register — Leadership Status Agent

| Tool | Purpose | Class | Data accessed | Scope | Failure mode | Guardrail | HITL? | Audit |
|---|---|---|---|---|---|---|---|---|
| Read/Glob/Grep | read inputs/prior deck | read | internal financials/status | workspace read | stale data | verification report | no | logged |
| Write/Edit | emit deck + deliverables | write | output folder | selected folder | overwrite | save to dated file | no | logged |
| Bash (inherited) | (none required) | write/destructive | shell | UNSCOPED | arbitrary exec | NONE | NO | none |

Trifecta: private data yes · untrusted content low · external comms low → not full trifecta.
**Finding F-002 (Major):** Bash is inherited with no scope/guardrail/HITL and is not needed — remove it
or sandbox it. Until then this agent cannot pass the production-candidate gate.
