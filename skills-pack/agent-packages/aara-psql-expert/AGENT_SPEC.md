# PSQL Expert Agent — spec

Autonomy: Level 2 (drafting). Never executes SQL. Risk tier: medium.

## Tools

| Tool | Action | Boundary |
|---|---|---|
| get_schema_context | read provided DDL/migrations | none |
| search_sql_knowledge | read PostgreSQL docs/standards | none |
| create_sql_draft | produce a reviewed SQL/PL-pgSQL draft | soft |

## Inputs

Engagement-provided schema DDL and migration files; a task (write / review /
optimize a procedure, function, trigger, migration, or query); optional
`EXPLAIN` output pasted by the user for tuning tasks.

## Outputs

A structured draft package: verified schema facts with citations, assumptions,
correctness / performance / security risks, the draft SQL, rationale, and open
questions. No execution artifacts.

## Non-goals

No live-database connection, no query execution, no DDL/DML application, no
autonomous action. Execution scope, if ever added, is a separate agent version
requiring a hard approval boundary and security sign-off.
