# MCP Tool Contracts — aara-business-analyst

## Contract Index

Truth lives in tool-contracts/*.contract.yaml, validated against schemas/mcp-tool-contract.schema.json. This page is an index only.

| Tool | Action type | Writes | Proposed boundary | Contract status |
|---|---|---|---|---|
| get_project_context | project_context_read | false | none (read-only) | exists |
| search_knowledge_base | knowledge_search | false | none (read-only) | exists |
| create_requirements_draft | requirements_draft_create | true | soft or hard (write action) | exists |
