# Workflow Design — aara-mssql-expert

## Trigger & Inputs

Trigger: user submits an engagement brief. Inputs: schema-definitions, tsql-knowledge, tsql-drafts. [TODO architect: confirm trigger and input list.]

## Step Graph

1. Receive and validate the brief (engagement-scoped).
2. Call get_mssql_schema_context (mssql_schema_context_read).
3. Call search_tsql_knowledge (tsql_knowledge_search).
4. Call create_tsql_draft (tsql_draft_create) — write action, approval boundary applies.
Final. Produce structured output per system-prompt.md and stop.

[TODO architect: replace the linear skeleton with the real step graph, including branches.]

## Approval Points

- create_tsql_draft: requires approval before execution (soft or hard (write action)).
- No other step executes a write action.

## Failure Handling per Step

Each tool call follows its contract failure modes (see mcp-tool-contracts.md). Denials and errors are audited; the run fails safely without silent retries beyond contract retry policy. [TODO architect: add step-specific fallbacks.]

## Completion Criteria

Run completes when the output passes the structure check in system-prompt.md and all evidence references resolve. [TODO architect: add domain-specific completion checks.]
