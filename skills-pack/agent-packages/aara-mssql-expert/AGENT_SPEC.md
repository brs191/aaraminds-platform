# MSSQL Expert Agent — spec

Autonomy: Level 2 (drafting). Never executes T-SQL. Risk tier: medium.
Target: Azure SQL Database / Managed Instance (SQL Server 2025 baseline).

## Tools

| Tool | Action | Boundary |
|---|---|---|
| get_mssql_schema_context | read provided DDL/migrations | none |
| search_tsql_knowledge | read Microsoft Learn / standards | none |
| create_tsql_draft | produce a reviewed T-SQL draft | soft |

## Inputs

Engagement-provided T-SQL DDL and migration scripts; a task (write / review /
optimize a procedure, function, or query); optional execution-plan XML or
`SET STATISTICS IO, TIME` output pasted by the user for tuning tasks; the target
service tier and edition where relevant.

## Outputs

A structured draft package: verified schema facts with citations, assumptions
(including assumed isolation level and edition), correctness / performance /
security risks, the draft T-SQL, rationale, and open questions. No execution.

## Non-goals

No live-database connection, no query execution, no DDL/DML application, no
autonomous action. Execution scope, if ever added, is a separate agent version
requiring a hard approval boundary and security sign-off.
