# Workflow Design — aara-business-analyst

## Trigger & Inputs

Trigger: user submits an engagement brief. Inputs: project-context, knowledge-base, requirements-drafts. [TODO architect: confirm trigger and input list.]

## Step Graph

1. Receive and validate the brief (engagement-scoped).
2. Call get_project_context (project_context_read).
3. Call search_knowledge_base (knowledge_search).
4. Call create_requirements_draft (requirements_draft_create) — write action, approval boundary applies.
Final. Produce structured output per system-prompt.md and stop.

[TODO architect: replace the linear skeleton with the real step graph, including branches.]

## Approval Points

- create_requirements_draft: requires approval before execution (soft or hard (write action)).
- No other step executes a write action.

## Failure Handling per Step

Each tool call follows its contract failure modes (see mcp-tool-contracts.md). Denials and errors are audited; the run fails safely without silent retries beyond contract retry policy. [TODO architect: add step-specific fallbacks.]

## Completion Criteria

Run completes when the output passes the structure check in system-prompt.md and all evidence references resolve. [TODO architect: add domain-specific completion checks.]
