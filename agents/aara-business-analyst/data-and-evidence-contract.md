# Data & Evidence Contract — aara-business-analyst

Human-readable rendering of data-evidence-contract.json (the schema-validated source of truth).

## Domain Table

| Domain | Authoritative source | Classification | Record type |
|---|---|---|---|
| project-context | engagement repository | client-confidential | read-only [TODO confirm] |
| knowledge-base | AaraMinds knowledge base | internal | read-only [TODO confirm] |
| requirements-drafts | agent blueprint repository | client-confidential | read-only [TODO confirm] |

## Evidence Rules

Factual claims require citations (document id or query id). Uncited output is flagged. Memory writes require citations — enforced by the platform memory-citation gate.

## Staleness and Conflict Notes

Retrieved content, memory, and generated summaries are not authoritative unless backed by this mapping. [TODO architect: mark cached, derived, or conflicting data per domain.]
