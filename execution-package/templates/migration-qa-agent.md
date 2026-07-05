# Template: Migration QA Agent

Status: reference template #3 — the differentiated enterprise story (chosen over FinOps per Phase 0 proposal; revisit at DG-001 if executive appeal matters more).

| Field | Value |
|---|---|
| Primary job | Source/target validation, exception analysis, reconciliation summaries during migrations |
| Default autonomy | Level 1–2 (Advisory/Drafting); **no autonomous data mutation — ever** |
| Risk tier | High (client-confidential and potentially PII data; migration decisions depend on its accuracy) |

## Tools
| Tool | Action type | Approval boundary |
|---|---|---|
| get_source_inventory | read | none |
| get_target_inventory | read | none |
| run_reconciliation_query | read (bounded, read-only credentials) | soft |
| draft_exception_report | draft-write | none |
| (never) fix_data_mismatch | — | blocked (listed in blocked_actions_ref) |

## Identity
Strictly read-only scopes on source and target systems, separate credentials per environment, `pii_allowed` decided per engagement in manifest memory config. This template exercises the identity spec hardest — it is the reason the schema requires per-scope justification.

## Data & evidence
Domains: source inventory (authoritative: source system), target inventory (authoritative: target system), reconciliation results (derived — must be marked derived with staleness note). Evidence rule: every mismatch claim cites the query and row evidence; `uncited_output_behavior: block`. Memory: engagement-scoped only (leakage gate is critical here).

## Evaluation focus
Golden suite: reconciliation accuracy against seeded known-mismatch datasets; false-negative rate on exceptions is the headline metric. Safety: prompt-injection via data content (field values containing instructions). Memory-leakage tests across engagements are a critical blocker.

## Why this template matters
It proves the platform on a high-stakes, evidence-heavy, read-only agent — the strongest demonstration that readiness verdicts, evidence contracts, and blocked-action enforcement carry real enterprise weight.
