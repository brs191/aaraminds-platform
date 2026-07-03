# Skill — MCP-Go Resources

## Purpose

Design MCP resources to expose read-only context (documents, configuration, system state) to agents in a structured, addressable way. Resources are the read-side of the MCP protocol; tools are the write/action side. Conflating the two is the most common modeling mistake — this skill is about getting the split right and building resources that are useful, safe, and discoverable.

## Resource vs. tool: the load-bearing distinction

| Concern | Resource | Tool |
|---|---|---|
| Operation type | Read-only retrieval of context | Action, possibly with side effects |
| Caller intent | "Give me X so I can reason about it" | "Do X" or "Compute X from these inputs" |
| Addressing | URI-shaped (`docs://platform/overview`) | Named verb (`generate_runbook`) |
| Idempotency | Naturally idempotent | Often not (mutations) |
| Caching | Often cacheable | Rarely cacheable |
| Audit | Read access logged | Every call audited, with arguments |

If an LLM has to call your "resource" with three parameters and the response varies based on them — that's a tool. If it could load it once and re-use it across a session — that's a resource.

## When resources pay

- Stable reference documentation (architecture overviews, runbooks, glossaries) that many sessions will reuse.
- Configuration or topology (service catalog, namespace inventory) that's slow-changing.
- Reports or summaries generated periodically (cost reports, capacity dashboards) where the agent reads the latest.
- System state snapshots where retrieving them is cheap and the response is large enough to benefit from caching.

When *not* to use a resource:
- The data is generated per-request from the agent's input — use a tool.
- The data is sensitive and access should be audited per-call with arguments — a tool gives finer control.
- The data is too large to send to the agent in one shot — consider a search tool instead, or paginate the resource.

## URI design

A good resource URI is human-readable, namespaced, and stable.

```
✓ docs://architecture/platform-overview
✓ catalog://services/payment-api
✓ cost://reports/monthly/2026-04
✓ k8s://cluster/prod/namespaces
✓ runbook://incident/high-cpu

✗ resource://x12345                  # opaque
✗ /api/v1/resources/get?id=42        # leaks transport
✗ ../../etc/passwd                   # path traversal risk
```

Recommended schemes by content type:

| Scheme | What it's for |
|---|---|
| `docs://` | Human-authored documentation |
| `catalog://` | Inventory of named entities |
| `config://` | Configuration values (non-secret) |
| `topology://` | System topology snapshots |
| `runbook://` | Operational runbooks |
| `report://` | Generated reports |

Avoid `file://` even if the backing is filesystem — the scheme is the contract, not the storage.

## Resource template

Every resource you expose should have a one-page contract:

```markdown
## URI
resource://path

## Purpose
One sentence: what context does this expose?

## Data source
Where does the data come from? Live query, cached snapshot, static file?

## MIME type
application/json | text/markdown | text/plain

## Access control
Who can read? (anonymous, authenticated, role-gated)

## Refresh strategy
static | cached-N-minutes | live | scheduled

## Security notes
Sensitive fields to redact, classification level, regulatory regime
```

## Worked example: catalog resource

```go
// internal/resources/catalog/resource.go
package catalog

import (
    "context"
    "encoding/json"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func Register(s *server.MCPServer, svc *Service) {
    s.AddResource(
        mcp.NewResource(
            "catalog://services",
            "Service catalog",
            mcp.WithResourceDescription("Inventory of internal services with owner, criticality, and capability."),
            mcp.WithMIMEType("application/json"),
        ),
        func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
            services, err := svc.List(ctx)
            if err != nil {
                return nil, err
            }
            b, _ := json.MarshalIndent(services, "", "  ")
            return []mcp.ResourceContents{
                mcp.TextResourceContents{
                    URI:      "catalog://services",
                    MIMEType: "application/json",
                    Text:     string(b),
                },
            }, nil
        },
    )
}
```

The handler stays thin; `svc.List` is unit-testable; the response is structured JSON the agent can reason about.

## Dynamic URIs (resource templates)

When the resource is parameterised by an identifier, use the resource template feature:

```
URI template: catalog://services/{service_id}
Example:      catalog://services/payment-api
```

The agent can construct the URI from a known service ID. Resolve the template server-side; reject paths that don't match.

## Safety and security

- **No secrets.** Never expose API keys, connection strings, or tokens as resource content. If the resource description includes "secret" or "credential," reconsider the design.
- **No raw logs by default.** Raw log access is a tool with auditing, not a resource. Summaries and dashboards are fine.
- **PII redaction.** If the data has personally identifiable information, redact at the service layer before returning. Mask emails, account numbers, addresses unless the caller is explicitly authorised.
- **Path-traversal defence.** Validate every URI component against an allow-list. `../` in a resource path is a security review red flag.
- **Size bounds.** A resource that returns 100 MB of text pushes the agent's context window. Cap response size and provide a `cursor` or search tool for large datasets.

## Common failure modes

- **Resource-as-tool.** The resource takes complex query parameters and returns different content per call. Detection: the handler signature has multiple non-URI parameters. Fix: convert to a tool.
- **Stale cached resource.** The resource caches forever; data is stale. Detection: agent reasons over old state. Fix: declare a refresh strategy and honour it.
- **Information disclosure via URI.** The URI scheme leaks internal names or IDs (`secrets://prod-db-password-v3`). Detection: URIs read like internal documentation. Fix: opaque-but-stable identifiers; access control at retrieval, not via obscurity.
- **Unbounded response size.** A 200 MB resource returned in one read. Detection: agent timeouts; context-window overruns. Fix: paginate or summarise; provide a search tool.

## Verification questions

1. For each resource you expose, can you point at the contract (URI, purpose, MIME, access, refresh, security)?
2. Could any resource leak a secret, a token, raw PII, or unbounded logs?
3. Are there resources that *should* be tools because they take parameters and have side-effects?
4. Are URI schemes consistent across the server and stable across versions?
5. Is response size bounded for every resource? What's the largest possible payload?

## What to read next

- `prompts.md` — the other read-side primitive: pre-shaped prompts
- `tool-design.md` — when the data needs to be a tool, not a resource
- `enterprise-security.md` — access control and audit for resources
