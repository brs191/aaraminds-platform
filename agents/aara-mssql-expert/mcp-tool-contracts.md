# MCP Tool Contracts — aara-mssql-expert

## Contract Index

Truth lives in tool-contracts/*.contract.yaml, validated against schemas/mcp-tool-contract.schema.json. This page is an index only.

| Tool | Action type | Writes | Proposed boundary | Contract status |
|---|---|---|---|---|
| get_mssql_schema_context | mssql_schema_context_read | false | none (read-only) | exists |
| search_tsql_knowledge | tsql_knowledge_search | false | none (read-only) | exists |
| create_tsql_draft | tsql_draft_create | true | soft or hard (write action) | exists |
