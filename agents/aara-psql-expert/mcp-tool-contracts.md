# MCP Tool Contracts — aara-psql-expert

## Contract Index

Truth lives in tool-contracts/*.contract.yaml, validated against schemas/mcp-tool-contract.schema.json. This page is an index only.

| Tool | Action type | Writes | Proposed boundary | Contract status |
|---|---|---|---|---|
| get_schema_context | schema_context_read | false | none (read-only) | exists |
| search_sql_knowledge | sql_knowledge_search | false | none (read-only) | exists |
| create_sql_draft | sql_draft_create | true | soft or hard (write action) | exists |
