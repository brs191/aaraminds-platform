# Pattern — Tool Handler Middleware Chain

## Problem

A Go MCP server with N tools is N opportunities to forget input validation, audit logging, rate limiting, redaction, or any other cross-cutting safety concern. Per-handler safety code drifts: one tool gets validation, another gets audit logging, none gets all of it consistently. The first tool missing a check is the breach. The middleware-chain pattern centralizes cross-cutting concerns in composable wrappers and forces every tool handler through the same sequence.

## Use when

- Building a new Go MCP server (use the pattern from day 1)
- Retrofitting safety to an existing Go MCP server (introduce the chain in one shared package, then migrate handlers one at a time)
- Any time you find yourself copy-pasting the same `if err := validate(args); err != nil { ... }` across multiple handlers

## Avoid when

- The server has a single tool (overhead exceeds benefit; just code the safety inline)
- The MCP SDK in use offers built-in middleware semantics with all the layers you need (rare; verify first)

## Implementation steps

### Step 1 — define the handler type

The MCP SDK provides a tool handler signature. Wrap it as a type for composability:

```go
package guardrails

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
)

type Handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

type Middleware func(Handler) Handler
```

### Step 2 — the chain combinator

```go
func Chain(mws ...Middleware) Middleware {
    return func(h Handler) Handler {
        for i := len(mws) - 1; i >= 0; i-- {
            h = mws[i](h)
        }
        return h
    }
}
```

Middleware applies in registration order — the first listed wraps outermost, runs first on the request, last on the response.

### Step 3 — define each middleware as a `Middleware`

```go
func Validate(validators map[string]ValidatorFunc) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            if v, ok := validators[req.Params.Name]; ok {
                if err := v(req); err != nil {
                    return mcp.NewToolResultError("validation: " + err.Error()), nil
                }
            }
            return next(ctx, req)
        }
    }
}

func Audit(logger *slog.Logger) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            start := time.Now()
            res, err := next(ctx, req)
            outcome := "success"
            if err != nil {
                outcome = "error"
            } else if res != nil && res.IsError {
                outcome = "tool_error"
            }
            logger.LogAttrs(ctx, slog.LevelInfo, "tool_call",
                slog.String("tool", req.Params.Name),
                slog.String("outcome", outcome),
                slog.Int64("duration_ms", time.Since(start).Milliseconds()),
            )
            return res, err
        }
    }
}

// Additional middleware: RateLimit, Timeout, PromptInjection, RedactOutput, Trace, Authorize...
```

### Step 4 — assemble the chain once in main

Targets mcp-go v0.52.0. The package import for the server type is `github.com/mark3labs/mcp-go/server`; tool definitions come from `github.com/mark3labs/mcp-go/mcp`.

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"

    "yourorg/mcp-server/internal/guardrails"
    "yourorg/mcp-server/internal/telemetry"
)

func main() {
    ctx := context.Background()

    // initialize dependencies
    logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
    limiter := guardrails.NewToolLimiter()
    shield, _ := guardrails.NewShield(os.Getenv("CONTENT_SAFETY_ENDPOINT"))
    redactor := guardrails.NewRedactor()
    authz := guardrails.NewToolAuthz()
    riskyTools := map[string]bool{"generate_adr": true, "summarize_url": true}
    toolValidators := guardrails.DefaultValidators()

    shutdownTrace, _ := telemetry.InitTracer(ctx, "mcp-orders", "1.0.0")
    defer shutdownTrace(ctx)

    // Chain order is load-bearing (Trace outermost, RedactOutput innermost
    // after the handler). See "Step 6 — order matters" below.
    chain := guardrails.Chain(
        guardrails.Trace(),
        guardrails.Audit(logger, redactor),
        guardrails.Authorize(authz),
        guardrails.RateLimit(limiter),
        guardrails.Validate(toolValidators),
        guardrails.PromptInjection(shield, riskyTools),
        guardrails.Timeout(30 * time.Second),
        guardrails.RedactOutput(redactor),
    )

    // server.NewMCPServer returns *server.MCPServer
    s := server.NewMCPServer("orders", "1.0.0",
        server.WithToolCapabilities(true),
        server.WithRecovery(),
    )

    // mcp.NewTool uses the fluent builder. Every tool registers with its schema.
    adrTool := mcp.NewTool("generate_adr",
        mcp.WithDescription("Generate an ADR for an architectural decision"),
        mcp.WithString("title", mcp.Required(), mcp.Description("ADR title")),
        mcp.WithString("context", mcp.Required(), mcp.Description("Decision context")),
    )
    s.AddTool(adrTool, chain(generateADRHandler))

    detectTool := mcp.NewTool("detect_risks",
        mcp.WithDescription("Detect architecture risks in a system design"),
        mcp.WithString("system_json", mcp.Required(), mcp.Description("System description JSON")),
    )
    s.AddTool(detectTool, chain(detectRisksHandler))

    // ... all tools wrapped through chain; same pattern, per-tool schema

    // ServeStdio is a function, not a method on the server type.
    if err := server.ServeStdio(s); err != nil {
        logger.Error("server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

### Step 5 — enforce the no-bypass rule

The risk is that someone adds a new tool with `server.AddTool("name", handler)` directly, bypassing the chain. Two enforcement options:

**Code review checklist**: add a line to the PR review template: "every new `AddTool` call wraps with `chain(...)`."

**Linter**: a custom `golangci-lint` rule or simple `go vet`-style check that flags `s.AddTool(...)` calls where the second argument isn't a `chain(...)` call. Static, automated, doesn't depend on reviewer attention.

Either way, make the rule explicit. The first bypass is a hole; making it visible is the cheap fix.

### Step 6 — order matters

The ordering of middleware in the chain is load-bearing. Standard order for a Go MCP server:

```
Trace               ← outermost; span covers everything below
Audit               ← record outcome of everything below
Authorize           ← reject unauthorized before doing real work
RateLimit           ← reject over-budget before doing real work
Validate            ← reject invalid input before doing real work
PromptInjection     ← classify input before handler runs
Timeout             ← cap handler execution time
[handler]           ← the actual work
RedactOutput        ← scrub output before it returns
```

Rules of thumb:

- **Reject early**: validation, authorization, rate limit — run before anything expensive
- **Span outermost**: trace covers timing and outcome of everything
- **Audit outside redactor**: audit log writes through the redactor independently; never logs raw output
- **Timeout above handler**: enforces handler doesn't run forever
- **Redact innermost-after-handler**: scrubs whatever the handler produced before it leaves the chain

## Trade-offs

| Choice | Gain | Cost |
|---|---|---|
| One chain for all tools | Consistency; no missed layer | One slow middleware (e.g., Prompt Shields call) slows every tool |
| Per-tool chain variants | Selective middleware per tool | Maintenance burden; consistency drift |
| Middleware shared across services | Cross-service consistency | Coordination overhead |

Default: one chain, all tools, with middleware that's selective internally (Prompt Shields only on flagged risky tools, not on every call). That achieves both selectivity and consistency.

## Common failure modes

### A new tool registered without the chain
**Detection**: audit log shows tool calls without the structured fields the audit middleware adds; or production trace shows tool calls without span attributes.
**Fix**: lint rule + PR checklist. Every `s.AddTool(...)` wraps the handler with `chain(...)`.

### Middleware order accidentally changed
**Detection**: validation failures appear in audit log as `success` (because validate is now after audit). Or rate-limit rejections aren't traced (rate limit moved outside trace).
**Fix**: document the standard order in `../runtime-guardrails-go.md`; assert order in a unit test.

### Slow middleware on every call
**Detection**: p99 latency on cheap tools is high; spans show 200ms in the Prompt Shields call.
**Fix**: gate Prompt Shields call by tool risk tier (only call on `riskyTools` set). Local heuristic runs on every call as cheap pre-filter.

### Middleware that depends on previous middleware's context
**Detection**: `Authorize` middleware fails because the principal isn't in context; but principal is set by the HTTP-layer auth middleware that runs *before* the MCP chain.
**Fix**: define the contract clearly — what context values does each middleware require? Document; test with empty context.

### Per-tool config drift
**Detection**: each tool has its own subtly different validator instead of using the shared `toolValidators` map.
**Fix**: centralize validators in one map keyed by tool name; let the middleware look up; reject if missing for new tools (default-deny pattern).

## MCP tool opportunities

- **`generate_middleware_chain`** — given a list of tools with declared risk tiers, output the middleware chain initialization code for `main`, with the right middleware ordering and per-tool config.
- **`audit_handler_registrations`** — scan a Go MCP server's `main` package and flag any `s.AddTool(...)` call whose handler isn't wrapped by `chain(...)`.
- **`recommend_middleware_order`** — given a custom set of middleware, propose the right ordering with rationale.

## What to read next

- `../runtime-guardrails-go.md` — the middleware implementations this chain composes
- `structured-audit-log.md` — the audit middleware's schema
- `argument-sanitization.md` — sibling pattern for in-handler arg use
- `../prompt-injection-defense.md` — the prompt-injection middleware
- `../tool-authorization.md` — the authorize middleware
- `../observability-with-otel.md` — the trace middleware
