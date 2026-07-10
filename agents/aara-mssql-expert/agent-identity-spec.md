# Agent Identity Specification — aara-mssql-expert

Human-readable rendering of agent-identity-spec.json (the schema-validated source of truth).

## Principal

Dedicated agent identity (never a shared user account), distinct from all user principals. IdP: Entra ID pattern [VERIFY per implementation].

## Credential Pattern

OAuth 2.0 with federated identity credentials; shared secrets forbidden; maximum credential lifetime 24h. Local development uses an isolated credential, never production secrets.

## Scopes

| Resource | Permission | Environment | Justification |
|---|---|---|---|
| get_mssql_schema_context | read | dev | Read provided T-SQL schema DDL, migration scripts, and table/column/index definitions supplied for the engagement. Reads files only — never a live database. |
| search_tsql_knowledge | read | dev | Search an approved T-SQL / SQL Server knowledge base (Microsoft Learn docs, patterns, internal standards) for relevant guidance. |
| create_tsql_draft | write | dev | Produce a reviewed T-SQL draft (stored procedure, function, migration, or query) with cited rationale, for human review. Never executes SQL. |

## Conditional Access

Read-only file access within the engagement sandbox; no database connection and no outbound network egress are granted (advise-and-draft scope). Standard conditional access applies: device compliance required, no anonymous or unmanaged sessions, and access limited to the active engagement namespace. Credentials are short-lived federated tokens (<= 24h); no standing secret.

## Lifecycle & Owner

Provisioning, rotation, and retirement: see agent-identity-spec.json. Owner: Raja Shekar Bollam (acting engineering lead).
