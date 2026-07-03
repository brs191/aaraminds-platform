# Pattern — Structured Audit Log

## Problem

Every tool call needs an audit trail: who called what, when, with what args (redacted), what came back (redacted), how long it took, and whether the call was blocked by any guardrail. Without it, post-incident forensics is impossible ("did this tool get called during the breach window? by whom?"), SOC 2 / ISO 27001 evidence is missing, and operational debugging is "grep through stderr." The audit log is the load-bearing observability artifact for compliance. Built on `log/slog` with structured fields, shipped to Log Analytics via stderr → forwarder, it's cheap and durable.

## Use when

- Every Go MCP server. There is no scenario where this should be skipped.
- Retrofitting compliance evidence to an existing server (this is the first thing to add)
- Building the SOC 2 audit trail for tool-call activity

## Avoid when

- Toy / throwaway scripts not destined for production
- Cases where you genuinely have no compliance, forensics, or operational debugging need (rare for anything called by an LLM)

## Implementation steps

### Step 1 — define the audit log schema

Lock the schema before writing code. Every entry has these fields:

| Field | Type | Notes |
|---|---|---|
| `ts` | timestamp (auto from slog) | RFC3339 with nanoseconds |
| `level` | string (auto) | INFO for success, WARN for rejection, ERROR for unexpected failure |
| `event` | string | `tool_call_start`, `tool_call_end`, `tool_call_rejected` |
| `tool` | string | Tool name |
| `principal_oid` | string | Entra OID for HTTP transport; `""` for stdio |
| `principal_upn` | string | User principal name; `""` for stdio |
| `principal_source` | string | `entra_jwt`, `env_var`, `stdio_implicit` |
| `args_redacted` | json | Tool args after secret/PII redaction |
| `args_hash` | string | Short hash of original args for cross-reference with traces |
| `outcome` | string | `success` / `error` / `tool_error` / `rejected_validation` / `rejected_injection` / `rejected_authz` / `rejected_ratelimit` / `timeout` |
| `output_bytes` | int | Output size (not content) |
| `output_redactions` | array | List of redaction patterns hit (e.g., `["aws_access_key"]`) |
| `duration_ms` | int | End-to-end including middleware |
| `error_message` | string | If outcome != success |
| `request_id` | string | Unique per call; correlates with trace span |
| `mcp_protocol_version` | string | Negotiated MCP version |
| `server_name` | string | This server's name |
| `server_version` | string | This server's version |

Add fields as you discover needs. Don't add fields ad hoc per handler — schema must be uniform.

### Step 2 — initialize the logger

```go
package guardrails

import (
    "context"
    "log/slog"
    "os"
)

func NewAuditLogger(serverName, serverVersion string) *slog.Logger {
    h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: false,  // not useful in audit; we have the event field
    })
    return slog.New(h).With(
        slog.String("server_name", serverName),
        slog.String("server_version", serverVersion),
    )
}
```

**Logger output goes to stderr, not stdout.** For stdio MCP, stdout is the protocol wire — anything written to stdout corrupts the JSON-RPC stream. This is the #1 rule for stdio MCP and applies here.

### Step 3 — the audit middleware

```go
func Audit(logger *slog.Logger, redactor *Redactor) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            start := time.Now()
            reqID := requestIDFromContext(ctx)  // or generate a new one
            principal := PrincipalFromContext(ctx)

            // Redact args for log
            argsJSON, _ := json.Marshal(req.Params.Arguments)
            argsRedacted, _ := redactor.Redact(string(argsJSON))
            argsHash := hashArgs(req.Params.Arguments)

            // Start event
            logger.LogAttrs(ctx, slog.LevelInfo, "tool_call_start",
                slog.String("event", "tool_call_start"),
                slog.String("request_id", reqID),
                slog.String("tool", req.Params.Name),
                principalFields(principal)...,
            )

            res, err := next(ctx, req)

            // Outcome
            outcome := classifyOutcome(res, err)
            duration := time.Since(start)

            outputBytes := 0
            var redactions []string
            if res != nil {
                for _, c := range res.Content {
                    if tc, ok := c.(mcp.TextContent); ok {
                        outputBytes += len(tc.Text)
                        if hits := redactor.Hits(tc.Text); len(hits) > 0 {
                            redactions = append(redactions, hits...)
                        }
                    }
                }
            }

            level := slog.LevelInfo
            if outcome != "success" {
                level = slog.LevelWarn
            }
            if err != nil {
                level = slog.LevelError
            }

            attrs := []slog.Attr{
                slog.String("event", "tool_call_end"),
                slog.String("request_id", reqID),
                slog.String("tool", req.Params.Name),
                slog.String("args_hash", argsHash),
                slog.Any("args_redacted", json.RawMessage(argsRedacted)),
                slog.String("outcome", outcome),
                slog.Int("output_bytes", outputBytes),
                slog.Any("output_redactions", redactions),
                slog.Int64("duration_ms", duration.Milliseconds()),
            }
            if err != nil {
                attrs = append(attrs, slog.String("error_message", err.Error()))
            }
            attrs = append(attrs, principalFields(principal)...)

            logger.LogAttrs(ctx, level, "tool_call_end", attrs...)
            return res, err
        }
    }
}
```

### Step 4 — propagate to Log Analytics

stderr lines are JSON; capture them with whatever your platform uses:

- **Azure Container Apps**: built-in log capture → Log Analytics (configure in Terraform)
- **AKS**: Fluent Bit / OTel collector log pipeline → Log Analytics
- **Local dev**: file via `2>>audit.log` or just stderr to console

```hcl
resource "azurerm_container_app" "mcp_server" {
  # ...
  template {
    container {
      # ...
    }
  }

  # Log Analytics workspace via the environment
  # (configured at the azurerm_container_app_environment level)
}

resource "azurerm_container_app_environment" "main" {
  # ...
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
}
```

Audit log queries in Log Analytics (Kusto):

```kusto
ContainerAppConsoleLogs_CL
| where ContainerAppName_s == "mcp-orders"
| extend e = parse_json(Log_s)
| where e.event == "tool_call_end"
| where e.outcome != "success"
| project TimeGenerated, tool=e.tool, outcome=e.outcome,
          principal=e.principal_upn, duration=e.duration_ms
| order by TimeGenerated desc
```

### Step 5 — retention and access

Audit logs for SOC 2 / ISO 27001 typically need 1-year retention. Configure Log Analytics retention accordingly. Access control: Log Analytics RBAC limits who can query — auditors get read; engineers get read; nobody gets delete.

See `../soc2-iso27001-controls-mapping` for the specific control evidence the audit log satisfies.

## Trade-offs

| Choice | Gain | Cost |
|---|---|---|
| JSON to stderr | Universal capture; works locally and in any platform | Larger log volume than text |
| Per-call entry | Forensic value | Storage cost scales with call volume |
| Args redacted in log | Compliance-safe | Lost forensic detail for what the args really were |
| Args hashed alongside | Cross-reference with full args elsewhere if needed | Hashes mean nothing without a separate store |

Default: JSON to stderr, per-call, redacted args + hash. Logs are cheap; compliance and forensics aren't.

## Common failure modes

### Audit log writes to stdout (corrupts stdio MCP)
**Detection**: MCP client reports JSON parse errors; logs interleaved with protocol frames.
**Fix**: `slog.NewJSONHandler(os.Stderr, ...)`. Never `os.Stdout` for stdio MCP.

### Raw args end up in the log
**Detection**: a deliberate secret-bearing test input shows the secret unredacted in the audit log.
**Fix**: redactor runs in the audit middleware before serializing args. Test with a fuzzer or fixture-based test.

### Audit middleware not in the chain
**Detection**: tool calls don't appear in Log Analytics; or appear without standard fields.
**Fix**: enforce the chain wrapping pattern; lint or PR-checklist.

### Schema drift across tools
**Detection**: dashboards in Log Analytics get inconsistent results (some tools have `principal_oid`, others don't).
**Fix**: schema in `NewAuditLogger`; don't let individual middleware or handlers add ad-hoc fields. Add a versioned schema field if needed for evolution.

### Log Analytics costs unexpectedly high
**Detection**: monthly LA bill higher than expected.
**Fix**: data caps + sampling on success entries (keep all error/rejection entries). Configure ingestion-time sampling at the LA workspace.

### Audit log lost during failover
**Detection**: tool calls during a node failover have no audit trail.
**Fix**: confirm Log Analytics ingestion is buffered (it is by default); test failover in staging and verify logs survive.

## MCP tool opportunities

- **`generate_audit_log_schema`** — given a server's tool list and transport, output the schema definition + Kusto query templates for common forensic questions.
- **`audit_log_review`** — run periodic Kusto queries (top tools, top errors, rejected calls by reason) and produce a report.
- **`detect_audit_gaps`** — scan a Go MCP server's middleware chain and flag missing or misordered audit instrumentation.

## What to read next

- `../runtime-guardrails-go.md` — the middleware chain this slots into
- `tool-handler-middleware-chain.md` — sibling pattern for chain composition
- `argument-sanitization.md` — sibling pattern; sanitization happens before audit logs anything
- `../secrets-and-pii-redaction.md` — the redactor this middleware uses
- `../observability-with-otel.md` — the other half of the observability story (operations vs compliance)
- `../../soc2-iso27001-controls-mapping` — control evidence requirements
- `../../azure-microservices-observability` — broader Log Analytics setup
