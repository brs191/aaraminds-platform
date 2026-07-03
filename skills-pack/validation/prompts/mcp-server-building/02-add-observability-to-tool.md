---
id: mcp-server-building/02-add-observability-to-tool
area: mcp-server-building
exercises:
  - .claude/skills/mcp-go-server-building/references/observability.md
  - .claude/skills/mcp-go-server-building/references/tool-design.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Add observability to an existing tool

## Context

Attach `08-mcp-go-observability.md` and `02-mcp-go-tool-design.md`. The responder is reviewing a working but observability-light MCP tool.

## Prompt

Here is an MCP tool handler that works but has almost no observability:

```go
func handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    inputJSON, _ := req.RequireString("input_json")
    var input Input
    json.Unmarshal([]byte(inputJSON), &input)
    out, _ := svc.Generate(input)
    b, _ := json.Marshal(out)
    return mcp.NewToolResultText(string(b)), nil
}
```

Rewrite it to apply the observability guidance for this pack. Explain each change you make.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Structured logging on entry and completion** — emits a `tool call started` and `tool call completed` log with the tool name as a structured field (not a printf-shaped message)
- [ ] **Errors are not swallowed** — every `_` for an error becomes an explicit `if err != nil` path that logs and returns `mcp.NewToolResultError` with a user-meaningful message
- [ ] **Stderr logging** — calls out that stdio-transport servers must log to stderr, never stdout (stdout is the MCP wire)
- [ ] **Correlation field** — includes at least one structured field tying the log line to the call (e.g., `system` from the input, `services_count`, or a generated correlation ID)
- [ ] **No PII in logs** — does not log the full `inputJSON` blob; logs derived fields only
- [ ] **Distinguishes validation from internal error** — uses log levels deliberately (Info/Warn for caller-induced; Error for server bugs)
- [ ] **Logger is injected, not global** — accepts `logger *slog.Logger` as a parameter (handler closes over it) rather than reaching for a package-global
- [ ] **Counts what matters** — proposes at least one metric or structured-log field useful for SLO/alerting (latency, error rate, or domain-specific count like `risks_identified`)

## Reference output

> The original handler hides every failure and emits no signal that anything happened. Rewrite:
>
> ```go
> func handler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
>     logger.Info("tool call started", slog.String("tool", toolName))
>
>     inputJSON, err := req.RequireString("input_json")
>     if err != nil {
>         logger.Warn("input_json missing", slog.String("tool", toolName))
>         return mcp.NewToolResultError("input_json is required"), nil
>     }
>
>     var input Input
>     if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
>         logger.Warn("input_json failed to parse",
>             slog.String("tool", toolName),
>             slog.String("error", err.Error()),
>         )
>         return mcp.NewToolResultError("input_json must be valid JSON: " + err.Error()), nil
>     }
>
>     out, err := svc.Generate(input)
>     if err != nil {
>         logger.Info("tool call rejected",
>             slog.String("tool", toolName),
>             slog.String("error", err.Error()),
>         )
>         return mcp.NewToolResultError(err.Error()), nil
>     }
>
>     b, err := json.Marshal(out)
>     if err != nil {
>         logger.Error("failed to marshal output",
>             slog.String("tool", toolName),
>             slog.String("error", err.Error()),
>         )
>         return mcp.NewToolResultError("internal error: format failed"), nil
>     }
>
>     logger.Info("tool call completed",
>         slog.String("tool", toolName),
>         slog.String("system", input.SystemName),
>         slog.Int("items_generated", out.Count()),
>     )
>     return mcp.NewToolResultText(string(b)), nil
> }
> ```
>
> Why each change:
> - **Logging on entry and exit** gives an operator the timeline. Without it, you can't tell whether a request even reached the handler.
> - **Explicit error paths** mean the caller gets a useful message instead of an empty string or a Go-shaped stack trace.
> - **Levels** — caller-induced problems (missing input, bad JSON, validation failure) are `Warn` or `Info`; server-side problems (marshal failure of our own struct) are `Error`. Pages should fire on Error, not Warn.
> - **Structured fields, not formatted strings.** `slog.String("tool", toolName)` is queryable; `"tool call for " + toolName` is grep-only.
> - **`system` and `items_generated`** are domain-meaningful counters useful for dashboards. The logger never sees the full `inputJSON`, which may contain PII or large payloads.
> - **Logger injection** — `logger` comes in via `Register(s, svc, logger)`. Don't reach for `slog.Default()` here.
> - **Stderr only** in stdio transport (configured at server startup, but worth stating in the design note).

## Notes

Catches the most common observability regression: a tool that swallows errors silently and is impossible to debug post-incident. The before/after diff is also a useful teaching artifact.
