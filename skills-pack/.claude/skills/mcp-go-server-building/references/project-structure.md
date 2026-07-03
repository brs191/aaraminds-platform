# Skill — MCP-Go Project Structure

## Purpose

Lay out an MCP server project so the boundaries between MCP protocol, tool handling, business logic, backend connectors, security, and observability are explicit and stay enforced over time. Bad project structure is the slowest-moving but most expensive defect — by the time you notice, the cost of refactoring is high enough that teams live with the pain instead of fixing it.

## The principle

**Separate by layer, not by feature.** Tools, services, connectors, policy, audit, telemetry each get their own package. Adding a feature touches multiple packages because every feature crosses these layers; that is the design, not an inconvenience.

The alternative — packaging by feature — creates packages where MCP protocol handling, business logic, and backend calls live together. Six months in, every package looks the same: a thin smear of every layer. Cross-cutting concerns like audit and policy get reimplemented in each feature package, drift, and become inconsistent. The pack's anti-patterns document treats "business logic inside MCP handler" as a known failure mode for exactly this reason.

## Recommended layout

```text
mcp-server/
├── cmd/
│   └── server/
│       └── main.go              # Wire-up only: parse env, build dependency graph, start transport
├── internal/
│   ├── app/                     # App-level orchestration: builds and runs the server
│   ├── config/                  # Environment-bound config types and loading
│   ├── mcpserver/               # MCP server construction and tool registration ONLY
│   ├── tools/                   # Tool handlers, grouped by capability
│   │   ├── cost/                # cost.go, register.go, tests
│   │   ├── incidents/
│   │   └── health/              # Health check tool (low-risk, always present)
│   ├── services/                # Business logic, free of MCP types
│   │   ├── cost/
│   │   └── incidents/
│   ├── connectors/              # External system clients
│   │   ├── azure/
│   │   ├── servicenow/
│   │   └── mocks/               # In-package mocks for service tests
│   ├── auth/                    # Identity, token validation, tenant resolution
│   ├── policy/                  # Authorization, risk-tier checks, approval workflow
│   ├── audit/                   # Audit event types and emitters
│   ├── telemetry/               # Slog setup, metrics, traces, request-scoped context
│   ├── security/                # Redaction, secret scanning, input sanitization
│   └── errors/                  # Domain error types, error→MCP-result mapping
├── configs/                     # Sample config files, not secrets
├── contracts/                   # Source-of-truth tool/resource/prompt contracts (markdown)
├── deployments/
│   ├── docker/
│   ├── k8s/
│   └── containerapps/           # Azure Container Apps manifests
├── docs/
├── test/
│   └── integration/             # Integration tests separate from package-level unit tests
├── testdata/                    # Golden outputs, fixtures
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

## Package responsibilities — what goes where

The boundaries are load-bearing. If you find yourself wanting to put something "for convenience" in a package that doesn't own that responsibility, that's a signal you're about to introduce drift.

### `cmd/server/main.go`

Wire-up only. Parse environment, build dependency graph, start the transport, handle shutdown signals. No business logic. No tool registration. No HTTP handlers. This file should be small (50–150 lines) and stable.

### `internal/mcpserver/`

MCP server construction. Registers tools from `internal/tools/`. Wires policy, auth, audit, telemetry into the tool registration. **Does not contain business logic or backend calls.** Every line here is "register tool X, wire dependencies Y."

### `internal/tools/`

Tool handlers grouped by capability. Each subpackage contains:
- The handler functions (parse input, validate, authorize, call service, format result, audit)
- A `register.go` that wires the handlers to the MCP server
- Handler-level tests

Tools must not contain business logic. They are the MCP-aware shell around the service layer.

### `internal/services/`

Business logic that knows nothing about MCP. A service method's signature should be readable as a pure business operation: `GetCostSummary(ctx, CostSummaryRequest) (CostSummary, error)`. If MCP request/result types appear in service signatures, the layering is broken.

Services depend on connectors via interfaces, not concrete types. This is what makes services testable without spinning up real backends.

### `internal/connectors/`

External system clients. One subpackage per external system. Each connector exposes Go interfaces that services depend on, and concrete implementations that talk to the real system. Mocks for testing live in a `mocks/` subdirectory or alongside the connector with a `_mock.go` suffix.

Connectors are where you handle external API quirks, rate limits, retries, and backend-specific error translation. The service layer should never have to know that the Azure Cost Management API uses a different response shape than the AWS Cost Explorer API.

### `internal/auth/`

Identity-level concerns: token validation, principal resolution, tenant identification. Returns an authenticated identity that downstream policy checks can read. Does not make authorization decisions itself — that's policy's job.

### `internal/policy/`

Authorization decisions. Given an authenticated identity and a tool call, can this call proceed? Implements risk-tier checks, per-tool RBAC, tenant scope enforcement, environment scope (dev/staging/prod), and approval-workflow gating for high-risk tools.

### `internal/audit/`

Audit event types and emitters. Every tool call produces an audit event with input/output/decision/identity/timestamp. The emitter can write to stdout (for stdio transport), a structured log, an event bus, or a SIEM. Decoupled from where the events go.

### `internal/telemetry/`

Structured logging (slog), metrics (OpenTelemetry or Application Insights), traces (OpenTelemetry), and request-scoped context carrying trace IDs and request IDs. Tools and services depend on this for instrumentation; they should not construct logger/tracer instances themselves.

### `internal/security/`

Cross-cutting security utilities: output redaction (scan for and replace secret-like values), input sanitization, secret scanners. Used by tools when formatting responses and by audit when recording inputs.

### `internal/errors/`

Domain error types and the mapping from internal errors to MCP `CallToolResult` error shapes. Centralizing this prevents inconsistent error responses across tools.

## Package rules — enforce these

These rules should be enforceable via lint or import-graph checks. They are not style preferences; they are the layering contract.

1. **`internal/mcpserver/` may import `internal/tools/`. Not the reverse.** Tools don't know about the server they're registered on.
2. **`internal/tools/` may import `internal/services/`, `internal/policy/`, `internal/audit/`, `internal/telemetry/`, `internal/security/`. Not `internal/connectors/`.** Tools call services, not connectors directly.
3. **`internal/services/` may import `internal/connectors/` interfaces, `internal/telemetry/`. Not `internal/tools/`, not `internal/policy/`, not `internal/audit/`.** Services are pure business logic.
4. **`internal/connectors/` may import third-party client libraries. Not internal packages other than `internal/telemetry/`.** Connectors are self-contained.
5. **No package imports `cmd/`.** Cmd is the composition root, not a library.

A linter rule (`go-arch-lint`, `import-boss`, or a custom check in CI) enforces this. Without enforcement, the rules erode within months.

## Mono-repo vs multi-repo decision

**Default to a single repo per MCP server.** One server, one repo, one go.mod. This is true even when the server has many tools and many connectors.

**Move to multi-repo** when one of these specific conditions applies:

| Condition | Why multi-repo |
|---|---|
| Multiple servers share a connector that has independent lifecycle | Connector becomes its own module to version separately |
| Multiple servers share contract definitions | Shared contracts repo, semver-versioned |
| One server has independent deployment cadence from the rest of an MCP catalog | Releases can ship without coordinating with sibling servers |
| Different teams own different MCP servers and need code-ownership boundaries | One repo per team's servers |

**Do not move to multi-repo because:**
- The single repo "feels too big" — repo size is rarely the real problem
- "Microservices best practices say so" — that's about deployment topology, not code organization
- One team member prefers it — alignment cost exceeds the supposed benefit

Multi-repo has real ongoing cost: dependency management across modules, coordinated releases, harder cross-cutting changes. The cost is justified when the conditions above genuinely apply. It is not justified by aesthetics.

## What lives outside the server repo

Some things should always live outside the individual server repo, even when you're in mono-repo mode for the server itself:

- **Tool contracts as the source of truth** — these may live in the server repo's `contracts/` directory OR in a separate contracts repo if consumed by multiple servers. Either way, the contract is checked-in, versioned, and reviewed; it is not generated from code as an afterthought.
- **Catalog and registry data** — which servers exist, who owns them, what versions are deployed. Lives in a platform registry, not in any individual server repo.
- **Shared Go utilities** — only when there are genuinely shared patterns across multiple MCP servers. Most "shared utility" code is better duplicated than shared early. The MCP Server Expert pack explicitly endorses small per-repo duplication of policy/audit/redaction/CI scaffolding so each server can be copied independently. Premature sharing creates coupling.

## Common layering anti-patterns

**MCP types leaking into services.** Service signature reads `func (s *Service) Handle(req mcp.CallToolRequest)`. Now the service can't be called from anything except MCP. Refactor: extract a service-layer request type, translate in the tool handler.

**Connectors imported directly by tools.** Tool handler reaches past the service layer and calls `azureClient.GetCosts()`. Business logic now lives in the handler. Refactor: put the logic in the service, have the tool call the service.

**Policy checks scattered across handlers and services.** Some authorization happens in tool handlers, some deeper in services. Coverage is non-uniform. Refactor: centralize policy at the handler boundary, have services trust their callers (they're internal).

**`internal/utils/` package.** Becomes the dumping ground for anything that doesn't fit elsewhere. Six months later, half the codebase imports utils, which imports half the codebase. Refactor: kill the utils package; force each utility into the layer it actually belongs to.

**Tests live in a separate `tests/` tree rather than alongside packages.** Loses Go's `_test.go` convention. Refactor cost grows because tests don't move with the code. Use `package_test.go` files next to the code they test; reserve `test/integration/` for tests that genuinely cross package boundaries.

## What to read next

- For tool design that fits this structure: `tool-design.md`
- For security controls that live in `internal/auth/`, `internal/policy/`, `internal/security/`: `enterprise-security.md`
- For observability that lives in `internal/telemetry/`: `observability.md`
