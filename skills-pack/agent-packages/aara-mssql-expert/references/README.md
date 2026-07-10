# T-SQL / SQL Server knowledge base

Deep reference content for the MSSQL expert agent. The core behavioral skill is
`../agent.md`; this directory is the knowledge it routes into. Content targets
**SQL Server 2025** (GA November 2025) and **Azure SQL Database / Managed
Instance**. Version- and edition-gated features are flagged inline, e.g.
`[SQL2025]`, `[Enterprise]`, `[AzureSQL]`.

This is a SQL Server / T-SQL knowledge base, not a generic SQL one. Where T-SQL
diverges from other dialects (especially PostgreSQL's PL/pgSQL), the difference
is called out — because generic advice is often wrong for SQL Server.

## Routing table

| When the task is about… | Read |
|---|---|
| Writing T-SQL: batches, variables, temp tables, MERGE, SET options | `tsql-language.md` |
| Procedures, functions (scalar vs inline TVF), EXECUTE AS | `procedures-and-functions.md` |
| Error handling: TRY/CATCH, THROW, transactions, XACT_ABORT | `error-handling-and-transactions.md` |
| Injection-safe dynamic SQL (sp_executesql, QUOTENAME), security | `dynamic-sql-and-security.md` |
| Isolation, locking, RCSI/SNAPSHOT, deadlocks, queues | `concurrency-and-isolation.md` |
| Clustered vs nonclustered indexes, INCLUDE, filtered, columnstore | `indexing.md` |
| Reading plans, STATISTICS IO/TIME, Query Store, parameter sniffing | `query-tuning-and-parameter-sniffing.md` |
| Azure SQL DB / MI specifics that differ from on-prem | `azure-sql-specifics.md` |
| Types, datetime2, nvarchar, JSON/vector, keys | `data-types-and-modeling.md` |
| Common mistakes and their fixes | `antipatterns.md` |

## Non-negotiable rules (from agent.md)

1. Verify schema from retrieved evidence; never assume object names, types, or
   index shapes.
2. Parameterize dynamic SQL with `sp_executesql` + typed `@params`; quote
   identifiers with `QUOTENAME()`. Never concatenate untrusted input.
3. State the assumed isolation level and edition in concurrency/performance
   advice (Azure SQL DB defaults to RCSI).
4. Cite the source (DDL `source_ref` or Microsoft Learn) for every non-trivial
   claim.
5. Advise and draft only — never execute T-SQL.

## SQL Server 2025 highlights the agent should know

- Native `JSON` data type (2GB) with `JSON_MODIFY`, `JSON_CONTAINS`,
  `JSON_OBJECT_AGG`, `JSON_ARRAY_AGG` — stop storing JSON in `NVARCHAR(MAX)`.
- Native `VECTOR` data type with built-in vector search and in-database model
  management (Azure OpenAI / Foundry / Ollama) defined in T-SQL.
- Native regex functions: `REGEXP_LIKE`, `REGEXP_REPLACE`, `REGEXP_SUBSTR`.
- Change Event Streaming, REST API support, and Fabric mirroring via CDC.
- Microsoft Entra authentication across editions (including Express).

## Key T-SQL vs PL/pgSQL differences (do not copy Postgres advice)

| Concern | PostgreSQL | SQL Server / T-SQL |
|---|---|---|
| Errors | `EXCEPTION WHEN` blocks | `TRY...CATCH`, `THROW`/`RAISERROR` |
| Dynamic SQL | `format()` `%I`/`%L`, `EXECUTE ... USING` | `sp_executesql` + `QUOTENAME()` + typed `@params` |
| Isolation default | MVCC (readers never block) | Lock-based READ COMMITTED on-prem; **RCSI by default on Azure SQL DB** |
| Row store | heap + indexes | **clustered index defines physical order** + nonclustered |
| Upsert | `INSERT ... ON CONFLICT` | `MERGE` (with caveats) or `INSERT ... WHERE NOT EXISTS` + locking |
| Sequential keys | `uuidv7()` | `NEWSEQUENTIALID()` vs `NEWID()` |
| Plan stability | generic-plan/custom-plan | **parameter sniffing** (its own problem + toolkit) |
