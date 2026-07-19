module github.com/example/microservices-system-design-mcp-server

go 1.25.11

// MCP Go SDK pin. Re-verify against skills/mcp-go-server-building/references/ecosystem-facts.md
// before production use. As of July 19, 2026 the verified-current versions are:
//   - mark3labs/mcp-go: v0.56.0 (2026-07-08), supports MCP spec 2025-11-25
//   - modelcontextprotocol/go-sdk (official): v1.6.1, stable post-1.0;
//     v1.7.0 ships support for MCP spec 2026-07-28 (finalizes July 28, 2026)
// This pack standardizes on mark3labs/mcp-go for alignment with v7.3
// example code. Teams without that constraint should consider the official SDK.
// ACTION: when spec 2026-07-28 lands, re-check mark3labs spec coverage; if it
// lags, revisit migration to the official SDK within the 12-month deprecation window.
require github.com/mark3labs/mcp-go v0.56.0

require (
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/text v0.14.0 // indirect
)
