# Skill — MCP-Go Code Generation

## Purpose

Generate practical, production-shaped MCP-Go code. This skill is the canonical template library: server bootstrap, tool handler, service-layer skeleton, table-driven test, dockerfile, CI workflow. It also names what *not* to generate — tool shapes that look helpful but create operational, security, or maintenance liabilities.

## Generation principles

When generating MCP-Go code, apply these rules in order:

1. **Simple structure first.** A working minimal server beats a perfect-but-incomplete one.
2. **Clear input validation.** Every input field has a known shape and a known failure mode.
3. **Safe error messages.** Errors that surface to clients don't leak stack traces or internal paths.
4. **Context-aware handlers.** Every handler accepts `context.Context` and propagates it to downstream calls.
5. **Service-layer separation.** Business logic in `internal/services/<name>/`; MCP wiring in `internal/tools/<name>/register.go`.
6. **Production-grade defaults.** Logger to stderr, structured JSON, bounded shutdown, health endpoints.
7. **README instructions.** Build, run, configure, troubleshoot — without leaving the README.
8. **Test strategy stated.** Even if tests aren't generated, the test plan is.

## What to NEVER generate

These are the high-risk patterns. Generating any of them is malpractice:

- **Arbitrary command execution.** `os.exec` on user input. The blast radius is the whole host.
- **Production write actions without approval logic.** A `delete_resource` tool that fires immediately. Always two-step (plan/apply) with `dry_run: true` as the default.
- **Hardcoded secrets.** API keys in source, connection strings in env vars. Always Key Vault + Managed Identity.
- **Unbounded log or query tools.** A `query_db` tool that takes free SQL. Always bound the surface (named queries, parameterised) or refuse.
- **Raw admin-API wrappers.** A tool that's literally "call any Azure ARM endpoint". Every operation is a separate, named, audited tool with clear scope.
- **Unauthenticated remote servers.** HTTP MCP without auth middleware. Network-exposed MCP without auth is a security incident waiting.

If a request is shaped like one of these, propose the safer alternative first.

## Default server bootstrap

`internal/mcpserver/server.go`:

```go
package mcpserver

import (
    "log/slog"

    boundarysvc "github.com/example/mcp-server/internal/services/boundary"
    boundarytool "github.com/example/mcp-server/internal/tools/boundary"

    "github.com/mark3labs/mcp-go/server"
)

// NewServer wires dependencies and registers tools.
func NewServer(logger *slog.Logger) *server.MCPServer {
    s := server.NewMCPServer(
        "enterprise-mcp-server",
        "1.0.0",
        server.WithToolCapabilities(true),
        server.WithRecovery(),
    )

    boundarytool.Register(s, boundarysvc.NewService(logger), logger)
    // Register more tools here.

    return s
}
```

The function is a tiny composition root: build services, register tools, return server. No business logic, no transport choice.

## Default `main.go`

`cmd/server/main.go`:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/example/mcp-server/internal/mcpserver"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    // Stderr-bound JSON logging. Stdio transport reserves stdout for the wire.
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    s := mcpserver.NewServer(logger)

    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    transport := os.Getenv("MCP_TRANSPORT")
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    switch transport {
    case "streamablehttp", "http":
        logger.Info("starting MCP server", slog.String("transport", "streamable_http"), slog.String("port", port))
        if err := server.NewStreamableHTTPServer(s).Start(":" + port); err != nil {
            logger.Error("http server failed", slog.String("error", err.Error()))
            os.Exit(1)
        }
    default:
        logger.Info("starting MCP server", slog.String("transport", "stdio"))
        if err := server.ServeStdio(s); err != nil {
            logger.Error("stdio server failed", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }

    <-ctx.Done()
    logger.Info("shutdown signal received")
}
```

Stderr logging, signal-bounded shutdown, transport via env. SSE intentionally not included (deprecated).

## Default tool registration

`internal/tools/<name>/register.go`:

```go
package <name>

import (
    "context"
    "encoding/json"
    "log/slog"

    svc "github.com/example/mcp-server/internal/services/<name>"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func Register(s *server.MCPServer, service *svc.Service, logger *slog.Logger) {
    tool := mcp.NewTool("<tool_name>",
        mcp.WithDescription("..."),
        mcp.WithString("input_json", mcp.Required(), mcp.Description("JSON-encoded Input per contract")),
    )

    s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        logger.Info("tool call started", slog.String("tool", "<tool_name>"))

        inputJSON, err := req.RequireString("input_json")
        if err != nil {
            return mcp.NewToolResultError("input_json is required"), nil
        }

        var input svc.Input
        if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
            return mcp.NewToolResultError("input_json must be valid JSON: " + err.Error()), nil
        }

        result, err := service.Generate(input)
        if err != nil {
            logger.Info("tool call rejected", slog.String("tool", "<tool_name>"), slog.String("error", err.Error()))
            return mcp.NewToolResultError(err.Error()), nil
        }

        b, err := json.MarshalIndent(result, "", "  ")
        if err != nil {
            logger.Error("marshal failed", slog.String("tool", "<tool_name>"), slog.String("error", err.Error()))
            return mcp.NewToolResultError("internal error: format failed"), nil
        }

        logger.Info("tool call completed", slog.String("tool", "<tool_name>"))
        return mcp.NewToolResultText(string(b)), nil
    })
}
```

Twenty-five lines of generation-shape. No logic; the service does the work.

## Default service skeleton

`internal/services/<name>/service.go`:

```go
package <name>

import (
    "errors"
    "log/slog"
)

type Input struct {
    SystemName string `json:"system_name"`
    // ...
}

type Output struct {
    SystemName string `json:"system_name"`
    Score      int    `json:"score"`
    // ...
}

type Service struct {
    logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
    return &Service{logger: logger}
}

func (s *Service) Validate(input Input) error {
    if input.SystemName == "" {
        return errors.New("system_name is required")
    }
    return nil
}

func (s *Service) Generate(input Input) (*Output, error) {
    if err := s.Validate(input); err != nil {
        return nil, err
    }
    // Rule logic here.
    return &Output{SystemName: input.SystemName, Score: 0}, nil
}
```

Validate + Generate. Unit-testable. No MCP dependency.

## Default health tool

```go
func registerHealthTool(s *server.MCPServer) {
    tool := mcp.NewTool("health_check",
        mcp.WithDescription("Check whether the MCP server is healthy. Returns ok if the server can handle tool calls."),
    )

    s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        return mcp.NewToolResultText(`{"status":"ok"}`), nil
    })
}
```

A health *tool* (callable by an agent) is different from a health *endpoint* (HTTP `/healthz` for the orchestrator's probes). Both can exist; they serve different consumers.

## CI workflow (minimum)

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 'stable' }
      - run: go mod tidy
      - run: gofmt -l . | tee /dev/stderr | (! grep .)
      - run: go vet ./...
      - run: go test -race ./...
      - run: go build ./...
```

## Generation defaults summary

| File | Lines (typical) | Purpose |
|---|---|---|
| `cmd/server/main.go` | 30–50 | Transport, logging, shutdown |
| `internal/mcpserver/server.go` | 20–40 | Composition root |
| `internal/services/<name>/service.go` | 80–200 | Rule logic |
| `internal/services/<name>/service_test.go` | 100–300 | Table-driven tests |
| `internal/tools/<name>/register.go` | 25–40 | MCP wiring |
| `contracts/architecture-tools/implemented/<name>.md` | 50–100 | Contract |
| `Dockerfile` | 20–30 | Multi-stage, distroless |
| `.github/workflows/ci.yml` | 15–25 | Lint, vet, test, build |
| `README.md` | 80–150 | Build, run, configure |

## Common failure modes when generating

- **Business logic in `register.go`.** The handler grows past 50 lines. Fix: push into the service package as you generate.
- **`map[string]any` input.** Generated handler skips the typed-input step. Fix: always generate a struct in `service.go`.
- **stdout logging.** A generated example uses `fmt.Println` or `log.Print`. Fix: always `slog.New(slog.NewJSONHandler(os.Stderr, ...))`.
- **No context propagation.** Generated handler ignores `ctx`. Fix: thread `ctx` through every call.
- **Missing health endpoint or health tool.** Production deploys need both. Fix: generate both by default.
- **SSE in the transport switch.** Deprecated. Fix: omit SSE; mention deprecation in code comments if maintaining legacy.

## Verification questions

1. Does generated `main.go` log to stderr and handle SIGTERM?
2. Does the generated service package have validate + generate + tests?
3. Does the generated `register.go` stay under 50 lines?
4. Are there any hardcoded secrets, unbounded queries, or raw admin wrappers in the generated code?
5. Does the generated CI workflow run `gofmt -l`, `go vet`, `go test -race`?

## What to read next

- `server-basics.md` — the why behind these defaults
- `tool-design.md` — tool-shape guidance
- `project-structure.md` — package layering rules
- `reference-implementation.md` — a complete worked example
