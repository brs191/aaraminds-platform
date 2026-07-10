# Data & Evidence Contract — aara-mssql-expert

Human-readable rendering of data-evidence-contract.json (the schema-validated source of truth).

## Domain Table

| Domain | Authoritative source | Classification | Record type |
|---|---|---|---|
| schema-definitions | engagement-provided T-SQL DDL and migration repository | client-confidential | read-only [TODO confirm] |
| tsql-knowledge | Microsoft Learn (SQL Server / Azure SQL) and internal T-SQL standards | public | read-only [TODO confirm] |
| tsql-drafts | agent blueprint repository | client-confidential | read-only [TODO confirm] |

## Evidence Rules

Factual claims require citations (document id or query id). Uncited output is flagged. Memory writes require citations — enforced by the platform memory-citation gate.

## Staleness and Conflict Notes

Retrieved content, memory, and generated summaries are not authoritative unless backed by this mapping. [TODO architect: mark cached, derived, or conflicting data per domain.]
