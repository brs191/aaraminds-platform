# Agent Identity Specification — aara-psql-expert

Human-readable rendering of agent-identity-spec.json (the schema-validated source of truth).

## Principal

Dedicated agent identity (never a shared user account), distinct from all user principals. IdP: Entra ID pattern [VERIFY per implementation].

## Credential Pattern

OAuth 2.0 with federated identity credentials; shared secrets forbidden; maximum credential lifetime 24h. Local development uses an isolated credential, never production secrets.

## Scopes

| Resource | Permission | Environment | Justification |
|---|---|---|---|
| get_schema_context | read | dev | Read provided schema DDL, migration files, and table/column/constraint definitions supplied for the engagement. Reads files only — never a live database. |
| search_sql_knowledge | read | dev | Search an approved PostgreSQL knowledge base (official documentation, patterns, and internal SQL standards) for relevant guidance. |
| create_sql_draft | write | dev | Produce a reviewed PL/pgSQL or SQL draft document (stored procedure, function, trigger, migration, or query) with cited rationale, for human review. Never executes SQL. |

## Conditional Access

Read-only file access within the engagement sandbox; no database connection and no outbound network egress are granted (advise-and-draft scope). Standard conditional access applies: device compliance required, no anonymous or unmanaged sessions, and access limited to the active engagement namespace. Credentials are short-lived federated tokens (<= 24h); no standing secret.

## Lifecycle & Owner

Provisioning, rotation, and retirement: see agent-identity-spec.json. Owner: Raja Shekar Bollam (acting engineering lead).
