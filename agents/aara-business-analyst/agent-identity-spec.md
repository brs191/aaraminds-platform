# Agent Identity Specification — aara-business-analyst

Human-readable rendering of agent-identity-spec.json (the schema-validated source of truth).

## Principal

Dedicated agent identity (never a shared user account), distinct from all user principals. IdP: Entra ID pattern [VERIFY per implementation].

## Credential Pattern

OAuth 2.0 with federated identity credentials; shared secrets forbidden; maximum credential lifetime 24h. Local development uses an isolated credential, never production secrets.

## Scopes

| Resource | Permission | Environment | Justification |
|---|---|---|---|
| get_project_context | read | dev | Read engagement brief, stakeholders, and scope from the engagement repository. |
| search_knowledge_base | read | dev | Search prior engagement knowledge and standards for relevant patterns. |
| create_requirements_draft | write | dev | Create a structured requirements draft document for review. |

## Conditional Access

[TODO security reviewer: network restrictions and risk policies for this agent.]

## Lifecycle & Owner

Provisioning, rotation, and retirement: see agent-identity-spec.json. Owner: Raja Shekar Bollam (acting engineering lead).
