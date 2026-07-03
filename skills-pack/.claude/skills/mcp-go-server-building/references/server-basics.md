# Skill — MCP-Go Server Basics

## Purpose

Create a working, production-shape MCP server in Go. Covers SDK selection (two viable options), minimal working server, transport selection, the conventions every production server should follow from day one, and the failure modes new developers fall into.

## Cross-reference

For verified ecosystem facts (current Go version, SDK versions, MCP spec), see `ecosystem-facts.md`. This skill states **how** to use the ecosystem; that file states what the current state of the ecosystem is.

## Decision before code — which Go SDK

Two production-viable Go MCP SDKs exist as of May 2026. The choice matters for every example, every project structure, every CI pipeline. Make it explicitly.

### `github.com/modelcontextprotocol/go-sdk` (official, recommended for new projects)

- Stable since v1.0.0, formal commitment to no breaking API changes
- Maintained by Anthropic in collaboration with Google
- API style is closer to Go conventions for generics and typed handlers

### `github.com/mark3labs/mcp-go` (community, choose for existing code or specific tooling)

- Active development, no formal v1.0 stability commitment
- Larger import count (~1,880 vs ~1,443) — most existing Go MCP code is on this SDK
- API style emphasizes builder functions for tool registration

### When to pick which

| Situation | Pick |
|---|---|
| Greenfield enterprise project, no prior Go MCP code | Official SDK |
| Migration cost from existing mcp-go code is high | mark3labs/mcp-go |
| Stability commitment matters more than ecosystem size | Official SDK |
| Existing examples, tooling, third-party integrations target mcp-go | mark3labs/mcp-go |
| Team has no preference and asks "which one" | Official SDK (the slightly safer default) |

The remainder of this skill shows both. Production code should pick one and commit; do not mix SDKs in a single server.

> **This pack's standardization decision.** The general guidance above recommends the official SDK for greenfield projects. This pack, however, **standardizes on `github.com/mark3labs/mcp-go`** so the example servers, contracts, and tool-registration patterns are internally consistent and internally consistent. Teams without an existing-code or ecosystem-alignment constraint should still default to the official SDK per the table above; teams adopting this pack's examples wholesale stay on mark3labs/mcp-go. Either way, do not mix SDKs in one server.

## Minimal working server — official SDK

```go
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GreetInput struct {
	Name string `json:"name" jsonschema:"the name of the person to greet"`
}

type GreetOutput struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to return"`
}

func SayHi(ctx context.Context, req *mcp.CallToolRequest, input GreetInput) (*mcp.CallToolResult, GreetOutput, error) {
	return nil, GreetOutput{Greeting: "Hi " + input.Name}, nil
}

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "greeter", Version: "v1.0.0"},
		nil,
	)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)

	// Run over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
```

What this gives you:
- Typed input and output structs with JSON Schema annotations (no manual schema construction)
- Compile-time type safety on tool handlers
- A working server that responds to `tools/list` and `tools/call` over stdio

## Minimal working server — mark3labs/mcp-go

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"greeter",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	tool := mcp.NewTool("greet",
		mcp.WithDescription("Say hi to someone"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the person to greet")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Hi %s", name)), nil
	})

	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}
```

What this gives you:
- Builder-style tool registration (`mcp.WithString`, `mcp.WithDescription`)
- Untyped handler signature; parse arguments from the request manually
- Same working server semantics, different API ergonomics

## Transport selection — choose by deployment shape

| Transport | When to use | When to avoid |
|---|---|---|
| **stdio** | Local development, desktop client integrations (Claude Desktop, VS Code with local servers), CLI tooling | Production remote deployment, multi-tenant scenarios |
| **Streamable HTTP** | Production remote deployment, Azure-hosted MCP servers, multi-client production | Local desktop integrations that expect subprocess transport |
| **SSE (Server-Sent Events)** | **Avoid for new projects** | New work — current MCP spec deprecates SSE in favor of streamable HTTP |

The current MCP spec (2025-11-25) deprecates SSE. On Azure specifically, SSE has documented 4-minute idle-timeout issues with Azure Load Balancer. Use streamable HTTP for any HTTP-based deployment.

## Production-shape additions from day one

A minimal server like the examples above is fine for local testing. **Do not deploy it to production.** A production-shape MCP server has additional structure even on day one:

1. **Configuration via environment, not hard-coded values.** Bind address, port, transport choice, log level, secret references — all from `os.Getenv` or a config library. Hard-coded ports survive about three days in production before they're a problem.

2. **Structured logging.** Use `log/slog` (Go 1.21+ standard library). Every log line is a structured event with severity, message, and key-value attributes. JSON output to stderr is the production default.

3. **Graceful shutdown.** Trap SIGTERM and SIGINT. Give in-flight requests a bounded grace period (5–30 seconds typically). Production orchestrators will SIGKILL if you don't shut down cleanly within the grace window.

4. **Health endpoint when running streamable HTTP.** Container Apps, AKS, and App Service all expect a `/health` or `/healthz` endpoint that returns 200 when the server is ready. The MCP `/mcp` endpoint is not a substitute — it's not a liveness probe.

5. **CORS configured for the actual clients.** If VS Code, browser, or any cross-origin client will talk to your server, configure CORS explicitly. Allow only the origins, methods, and headers your clients need.

6. **Timeouts on backend calls.** Every connector call should have a context timeout. Unbounded waits are the most common cause of "the server is hung" reports.

## Failure modes new developers hit

These are the things that go wrong in the first two weeks of every new MCP server project. Watch for them.

**Mixing SDKs.** A team starts with mark3labs/mcp-go, hits a feature gap, and copies an example using the official SDK. Now the project depends on both. Imports look fine until you discover the two SDKs have different types for similar concepts. Pick one. Commit.

**Tool names that change between versions.** A tool starts as `get_data`, gets renamed to `get_user_data`, then to `query_users`. Every consumer integration breaks each time. Tool names are part of your public contract. Treat renames as breaking changes.

**Schemas that drift from documentation.** The tool's `Description` says "from_date is YYYY-MM-DD" but the validation accepts arbitrary strings. Six months later, someone passes `last week` and the backend explodes. Validate every input field against its documented constraint. Treat the docs and the validation as one source.

**No error path differentiation.** A handler returns `mcp.NewToolResultError("something went wrong")` for everything. Clients can't tell input errors from auth errors from backend outages. Use distinct error messages and structured error data. Future-you will be the one debugging.

**Streamable HTTP without CORS.** The server runs locally, all tests pass. Deploy to Azure, point VS Code at it, get cryptic browser errors. CORS is required, not optional, when the client is a browser or browser-based tool.

**Stdio server with stdout logging.** Logs go to stdout, which is also the MCP wire protocol. The first log message corrupts the protocol stream. Always log to **stderr** for stdio transport, not stdout.

## Read-only by default

Default every new MCP server to **read-only** against its backend systems. Write access is per-tool, with explicit justification, security review, and audit. The reasoning: an LLM calling a read tool that returns wrong data is recoverable. An LLM calling a write tool that mutates the wrong record is not.

This is not a paranoia rule. It is the lesson every team that has deployed a non-read-only MCP server has learned the expensive way. See `../../mcp-go-production-review/references/anti-patterns.md` for the failure modes that motivate this.

## What to read next

- For tool design specifically: `tool-design.md`
- For project structure beyond a single file: `project-structure.md`
- For security controls: `enterprise-security.md`
- For observability: `observability.md`
- For the failure modes to avoid: `../../mcp-go-production-review/references/anti-patterns.md`
