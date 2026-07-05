# Template: Scrum Master Agent

Status: reference template #2 — lowest risk; proves the Level 1–2 lightweight path (R-010 mitigation: governance must stay light for advisory agents).

| Field | Value |
|---|---|
| Primary job | Ceremony preparation, blocker summary, sprint health, action-item tracking |
| Default autonomy | Level 1 (Advisory) / Level 2 (Drafting) |
| Risk tier | Low (internal data; no writes in MVP) |

## Tools
| Tool | Action type | Approval boundary |
|---|---|---|
| get_sprint_state | read | none |
| get_team_calendar | read | none |
| draft_standup_summary | draft-write | none |
| (Phase 2) create_action_item | external write | hard |

## Identity
Read-only scopes on the work-tracking system [VERIFY: Jira vs ServiceNow per DG-001 environment]; no write scopes in MVP.

## Data & evidence
Domains: sprint/work items (authoritative: work tracker, read-only), team calendar (read-only). Evidence rule: blocker and health claims cite ticket IDs. Uncited output behavior: `flag`.

## Evaluation focus
Golden suite: blocker detection accuracy, action-item extraction, RAG-status honesty (no watermelon-green: a sprint with slipping committed scope cannot summarize as green). Prompt-injection: ticket descriptions are untrusted content.

## Why this template matters
It validates that a Level 1–2 agent flows through the factory with minimal ceremony — few contracts, no identity write scopes, small eval suite — proving the rubric scales down, not just up.
