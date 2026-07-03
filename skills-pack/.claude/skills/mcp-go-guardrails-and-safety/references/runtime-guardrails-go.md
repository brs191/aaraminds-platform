# Runtime Guardrails in Go

## Purpose

This is the in-process safety layer for every Go MCP server. It runs before the tool handler executes, validates the input beyond what the MCP SDK does, enforces resource caps, audit-logs the call, and runs cleanup after. Every tool goes through the same chain — no bypasses, no exceptions. This reference covers the middleware shape, per-tool input validation, resource caps (timeout, output size, concurrency), rate limiting, and the helper functions middlewares depend on.

**SDK version**: code in this reference targets `github.com/mark3labs/mcp-go` v0.52.0 (the version pinned in `examples/microservices-system-design-mcp-server`). If you're on a different version, verify the API surface (`mcp.NewTool` fluent options, `req.RequireString`, `server.ServeStdio` as a function, etc.).

## The middleware chain shape

Every tool handler is wrapped by the same chain:

```
validate → authorize → rate-limit → audit-begin → [handler] → audit-end → redact-output
```

A handler is a function with the standard MCP signature:

```go
type Handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
```

Middleware wraps handlers. Composable; one-liner per tool:

```go
type Middleware func(Handler) Handler

func Chain(mws ...Middleware) Middleware {
    return func(h Handler) Handler {
        for i := len(mws) - 1; i >= 0; i-- {
            h = mws[i](h)
        }
        return h
    }
}
```

Register tools through the chain, not directly. The mcp-go fluent builder defines the tool; the chain wraps the handler:

```go
s := server.NewMCPServer("orders", "1.0.0",
    server.WithToolCapabilities(true),
    server.WithRecovery(),
)

guardrails := Chain(
    Validate(validators),
    Authorize(authz),
    RateLimit(limiter),
    Audit(logger, redactor),
    Timeout(30 * time.Second),
    RedactOutput(redactor),
)

adrTool := mcp.NewTool("generate_adr",
    mcp.WithDescription("Generate an ADR for an architectural decision"),
    mcp.WithString("title", mcp.Required(), mcp.Description("ADR title")),
    mcp.WithString("context", mcp.Required(), mcp.Description("Decision context")),
)
s.AddTool(adrTool, guardrails(generateADRHandler))

// Repeat for each tool — same pattern, different schema.
// ServeStdio is a function that takes the server, not a method.
if err := server.ServeStdio(s); err != nil {
    logger.Error("server failed", slog.String("error", err.Error()))
    os.Exit(1)
}
```

See `patterns/tool-handler-middleware-chain.md` for the pattern card with full code.

## Per-tool input validation

The MCP SDK validates against the declared JSON schema. That catches type errors and required-field omissions. It does **not** catch:

- Strings that pass schema validation but are way too long (DoS via memory)
- Format violations not expressible in JSON schema (regex constraints, charset)
- Cross-field invariants
- Values within type range but semantically invalid (negative quantity, future date that shouldn't be future)

Add a validator per tool:

```go
type ValidatorFunc func(req mcp.CallToolRequest) error

var toolValidators = map[string]ValidatorFunc{
    "generate_adr": func(req mcp.CallToolRequest) error {
        title := req.GetString("title", "")
        if len(title) > 200 {
            return fmt.Errorf("title exceeds 200 characters")
        }
        if !adrTitleRe.MatchString(title) {
            return fmt.Errorf("title must match [A-Za-z0-9 _-]+")
        }
        return nil
    },
}

var adrTitleRe = regexp.MustCompile(`^[A-Za-z0-9 _-]+$`)

func Validate(validators map[string]ValidatorFunc) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            if v, ok := validators[req.Params.Name]; ok {
                if err := v(req); err != nil {
                    return mcp.NewToolResultError(fmt.Sprintf("validation: %s", err)), nil
                }
            }
            return next(ctx, req)
        }
    }
}
```

**Reject early.** Validation runs first; bad input never reaches the handler, the audit log captures the rejection.

## Resource caps

### Timeout per tool call

Most tools should complete in < 5 seconds. Long-running tools (file generation, large queries) cap at 30 seconds. Anything longer should be async with a poll/notify pattern, not a single tool call.

```go
func Timeout(d time.Duration) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            ctx, cancel := context.WithTimeout(ctx, d)
            defer cancel()

            type result struct {
                res *mcp.CallToolResult
                err error
            }
            ch := make(chan result, 1)
            go func() {
                r, e := next(ctx, req)
                ch <- result{r, e}
            }()

            select {
            case r := <-ch:
                return r.res, r.err
            case <-ctx.Done():
                return mcp.NewToolResultError("tool timed out"), nil
            }
        }
    }
}
```

### Output size cap

A runaway tool that produces a 100MB response will bloat client context and observability storage. Cap output size:

```go
const maxOutputBytes = 256 * 1024  // 256KB per call

func CapOutput(max int) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            res, err := next(ctx, req)
            if err != nil || res == nil {
                return res, err
            }
            for i, c := range res.Content {
                if tc, ok := c.(mcp.TextContent); ok && len(tc.Text) > max {
                    res.Content[i] = mcp.TextContent{
                        Type: "text",
                        Text: tc.Text[:max] + "\n…[truncated]",
                    }
                }
            }
            return res, nil
        }
    }
}
```

Tune the limit per tool if needed — some tools legitimately return larger output.

### Concurrency cap

Limit concurrent calls per tool (and per-server). Prevents a single client from saturating the server:

```go
type Semaphore struct{ ch chan struct{} }

func NewSemaphore(n int) *Semaphore {
    return &Semaphore{ch: make(chan struct{}, n)}
}

func (s *Semaphore) Acquire(ctx context.Context) error {
    select {
    case s.ch <- struct{}{}:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (s *Semaphore) Release() { <-s.ch }

func Concurrency(sem *Semaphore) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            if err := sem.Acquire(ctx); err != nil {
                return mcp.NewToolResultError("server busy"), nil
            }
            defer sem.Release()
            return next(ctx, req)
        }
    }
}
```

For stdio MCP (one client per process), per-tool concurrency is more useful than per-server. For HTTP MCP, per-server matters too.

## Rate limiting

`golang.org/x/time/rate` is the standard library. Token bucket per tool:

```go
import "golang.org/x/time/rate"

type ToolLimiter struct {
    limits map[string]*rate.Limiter
    def    *rate.Limiter
    mu     sync.Mutex
}

func NewToolLimiter() *ToolLimiter {
    return &ToolLimiter{
        limits: map[string]*rate.Limiter{
            "generate_adr": rate.NewLimiter(rate.Every(time.Second), 5),  // 1/s burst 5
            "detect_risks": rate.NewLimiter(rate.Every(2*time.Second), 3),
        },
        def: rate.NewLimiter(rate.Every(time.Second), 10),
    }
}

func (l *ToolLimiter) For(tool string) *rate.Limiter {
    l.mu.Lock()
    defer l.mu.Unlock()
    if lim, ok := l.limits[tool]; ok {
        return lim
    }
    return l.def
}

func RateLimit(l *ToolLimiter) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            lim := l.For(req.Params.Name)
            if !lim.Allow() {
                return mcp.NewToolResultError("rate limit exceeded"), nil
            }
            return next(ctx, req)
        }
    }
}
```

Initial budget: generous (1–10 req/sec per tool depending on cost). Tighten after observing real usage in production. The bucket is per-process; if you run multiple replicas of an HTTP MCP server, the effective limit is N × per-process. For stdio, one process per client connection.

## Output sanitization (stdio-protocol safety)

The stdio MCP wire is line-delimited JSON. A handler that returns text containing raw control characters or embedded JSON-RPC frames can corrupt the protocol. Sanitize before return:

```go
func SanitizeStdio(s string) string {
    var b strings.Builder
    b.Grow(len(s))
    for _, r := range s {
        switch {
        case r == '\n' || r == '\t':
            b.WriteRune(r)
        case unicode.IsControl(r):
            // skip other control chars
        default:
            b.WriteRune(r)
        }
    }
    return b.String()
}
```

This is separate from PII/secret redaction (`secrets-and-pii-redaction.md`); this is protocol hygiene.

## Shared helpers used by middleware

These helpers are referenced by middlewares in this file and in `prompt-injection-defense.md`, `secrets-and-pii-redaction.md`, `observability-with-otel.md`, and `patterns/structured-audit-log.md`. Define them once in the shared `guardrails` package.

```go
package guardrails

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "log/slog"
    "strings"

    "github.com/mark3labs/mcp-go/mcp"
)

// ctxKey types prevent string-key collisions in context.Value.
type ctxKey int

const (
    requestIDKey ctxKey = iota
    principalKey
)

// extractStringArgs concatenates all string-valued tool arguments for content
// analysis (e.g., the prompt-injection classifier). Non-string args (numbers,
// bools, nested objects) are skipped — the classifier only cares about text.
func extractStringArgs(req mcp.CallToolRequest) string {
    var parts []string
    for _, v := range req.Params.Arguments {
        switch x := v.(type) {
        case string:
            parts = append(parts, x)
        case []any:
            for _, item := range x {
                if s, ok := item.(string); ok {
                    parts = append(parts, s)
                }
            }
        }
    }
    return strings.Join(parts, "\n")
}

// extractTextContent returns the concatenated text from a tool result's content
// items. Used by output classifiers and the output-cap middleware.
func extractTextContent(res *mcp.CallToolResult) string {
    if res == nil {
        return ""
    }
    var parts []string
    for _, c := range res.Content {
        if tc, ok := c.(mcp.TextContent); ok {
            parts = append(parts, tc.Text)
        }
    }
    return strings.Join(parts, "\n")
}

// requestIDFromContext returns the per-call request ID, generating one if not
// already set. The ID correlates audit log entries with OTel spans.
func requestIDFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
        return v
    }
    b := make([]byte, 8)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

// WithRequestID returns a derived context with the supplied request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey, id)
}

// principalFields returns slog attributes for the authenticated principal.
// For stdio transport (no auth), returns sentinel values so audit log fields
// are consistent across transports.
func principalFields(p *Principal) []slog.Attr {
    if p == nil {
        return []slog.Attr{
            slog.String("principal_source", "stdio_implicit"),
            slog.String("principal_oid", ""),
            slog.String("principal_upn", ""),
        }
    }
    return []slog.Attr{
        slog.String("principal_source", "entra_jwt"),
        slog.String("principal_oid", p.OID),
        slog.String("principal_upn", p.UPN),
    }
}

// classifyOutcome maps (res, err) to a structured outcome string used in the
// audit log and in OTel span attributes. Keep these values stable across the
// codebase — dashboards filter on them.
func classifyOutcome(res *mcp.CallToolResult, err error) string {
    if err != nil {
        return "error"
    }
    if res != nil && res.IsError {
        return "tool_error"
    }
    return "success"
}

// hashArgs returns a short hex hash of tool arguments. Used as a correlation key
// in audit log and OTel spans — *not* for forensics. The full (redacted) args
// live in the audit log; this just dedupes/correlates.
func hashArgs(args map[string]any) string {
    b, _ := json.Marshal(args)
    h := sha256.Sum256(b)
    return hex.EncodeToString(h[:8])
}
```

Note on `mcp.TextContent` and `res.IsError`: these are mcp-go v0.52.0 API surface. If the SDK changes, update the helpers (not every middleware that calls them). Centralizing in helpers is the reason for this section.

`Principal` is defined in `tool-authorization.md` for HTTP transport. For stdio transport, principal is nil and `principalFields` returns stdio sentinel values — keep middleware-free of nil checks.

## Worked example — brownfield: adding the chain to an existing 8-tool server

Setup: existing Go MCP server, 8 tools registered via `server.AddTool("name", handler)` directly. No middleware. Going to production in 4 weeks.

Steps:

1. **Define the middleware chain once**, in a shared `internal/guardrails` package. Validators map, rate limiter, audit logger, redactor, timeout duration — all initialized in one place.
2. **Wrap one tool first** as the canonical pattern: change `server.AddTool("generate_adr", h)` to `server.AddTool("generate_adr", chain(h))`. Verify behavior identical.
3. **Add tests** that hit the wrapped tool with bad input (oversize, malformed) and assert the chain rejects before the handler.
4. **Migrate the remaining 7 tools** one per day, adding per-tool validators as you go. Run race tests after each.
5. **Audit log first**, **rate limit last**. Audit log enables observability of the rest. Rate limit may surprise existing clients; turn on after a measurement window.
6. **No handler bypasses the chain.** Add a `go vet`-style lint rule or code review checklist item: every `AddTool` call must wrap with `guardrails(...)`. The first bypass is a security hole; make it visible.

Total elapsed: 1–2 weeks for migration + 1 week for tuning rate limits. Production-ready after.

## Anti-patterns

- **Handler-by-handler validation.** Validation logic duplicated across tools; one tool gets it wrong; that's the breach. Centralize in the chain.
- **No output cap.** A handler bug returns 100MB; client context explodes; observability storage bloats. Cap at 256KB by default.
- **No timeout.** A handler waits on a dead network call forever; the call ties up a goroutine. Always timeout.
- **Rate limit on the wrong axis.** Per-client makes sense for HTTP MCP with identified callers; per-tool makes sense for stdio. Match the transport.
- **Rejection without audit.** A blocked call should still log — that's the security signal. Audit middleware runs before the handler so rejects are captured.
- **Custom validators that depend on side effects** (DB lookups, network calls). Slow; turns validation into work. Keep validators pure.

## Verification questions

1. Does every `server.AddTool(...)` call route through the guardrails chain — no direct handler registrations?
2. Are per-tool validators registered for each tool, with length caps and format constraints beyond the JSON schema?
3. Is there a default timeout (e.g., 30s) applied to every tool call?
4. Is output capped at a sensible size (256KB by default)?
5. Is rate limiting active with per-tool budgets defined?
6. Are rejected calls audit-logged with the rejection reason?
7. Is the stdio sanitizer applied to tool outputs before they hit stdout?

## What to read next

- `patterns/tool-handler-middleware-chain.md` — pattern card with the full chain code
- `patterns/argument-sanitization.md` — sanitizing args at the use site
- `patterns/structured-audit-log.md` — the audit log schema this middleware writes
- `secrets-and-pii-redaction.md` — the output redactor middleware
- `prompt-injection-defense.md` — the classifier middleware
- `tool-authorization.md` — the authorize middleware (HTTP transport)
- `../mcp-go-server-building` — the build skill this attaches to
