# Skill — MCP-Go Reference Implementation

## Purpose

Point readers at the canonical worked example shipped with the pack, explain what it demonstrates, and provide a guided walkthrough from `main.go` to a deployed tool call. This skill is the bridge between "I've read the skills" and "I can build one of these from scratch."

## The reference implementation

The pack ships a complete, runnable MCP server at:

```
examples/microservices-system-design-mcp-server/
```

It implements the patterns described in skills 01, 02, 06, 07, 08, 09, 10, 17 and demonstrates them end-to-end. Eight tools, the v9.0 contract shape, table-driven tests, multi-stage Dockerfile, GitHub Actions CI, and a working stdio MCP server.

## What it demonstrates

| Concept | Where to look |
|---|---|
| Server skeleton with stderr logging and signal-bounded shutdown | `cmd/server/main.go` |
| Composition root (NewServer wires services into the MCP server) | `internal/mcpserver/server.go` |
| Service-package layering (rule logic isolated from MCP) | `internal/services/boundary/service.go` |
| Tool registration (thin handler, parse-validate-call-format) | `internal/tools/boundary/register.go` |
| Table-driven tests for every named rule | `internal/services/boundary/service_test.go` |
| Multi-stage build with distroless runtime | `Dockerfile` |
| CI gates (gofmt, vet, race, build) | `.github/workflows/ci.yml` |
| Tool contracts as separate documents | `contracts/architecture-tools/implemented/<name>.md` |
| Testdata as input + golden output JSON | `testdata/<tool>-input-*.json`, `testdata/<tool>-output-*.json` |
| MCP-driven demo against three architectures | `../../demo/architecture-review-demo/` |

## Guided walkthrough

### 1. Start at `cmd/server/main.go`

A tiny entry point. Read it top-to-bottom: logger initialisation (`slog.JSONHandler` on `os.Stderr`), composition (`mcpserver.NewServer(logger)`), signal-bounded context, transport selection, blocking start, shutdown log. Forty lines. Notice what's *not* there: no business logic, no tool definitions, no health endpoint plumbing.

### 2. Read `internal/mcpserver/server.go`

The composition root. It creates the MCP server, instantiates each service (passing the logger), and calls each tool package's `Register(s, svc, logger)`. The pattern repeats: one `Register` call per tool. Adding a tool means adding one line here and one new tool/service package.

### 3. Read one service end-to-end

Pick `internal/services/boundary/service.go`. The package declares `Input` and `Output` structs, a `Service` type with a constructor, a `Validate(input) error`, and a `GenerateCanvas(input) (*Canvas, error)`. No MCP imports. No protocol-aware code. The package depends on `errors`, `fmt`, `sort`, `strings` — that's it.

Read the rule logic: capability cohesion, data ownership clarity, dependency hygiene, ownership clarity, size sanity. Each rule is implemented as a named helper that produces a structured assessment or risk. The overall scoring is a deterministic function of those assessments.

### 4. Read the corresponding test

`internal/services/boundary/service_test.go` is table-driven. Cases cover clean boundaries, data co-ownership, no owner team, chatty dependency, fan-out, scoring rationality, stable ordering. Each case is an input fixture and an assertion on the output's shape and score. The whole file is unit-testable in under a second.

### 5. Read the registration

`internal/tools/boundary/register.go` is twenty-five lines of MCP wiring. It declares the tool name and description (intent-led, action-verb), takes `input_json` as the argument, unmarshals into the service's `Input`, calls `GenerateCanvas`, formats the result as JSON, returns. On any error, returns `mcp.NewToolResultError(...)` and logs at the right level.

### 6. Run the tests

```
cd examples/microservices-system-design-mcp-server
go test ./...
```

All pass; runtime under 5 seconds. If they don't pass on your environment, see `../../../../VERIFICATION_CHECKLIST.md`.

### 7. Build and run the server

```
go build -o ./mcp-server ./cmd/server
MCP_TRANSPORT=stdio ./mcp-server
```

The server starts and waits for MCP messages on stdin. Press Ctrl+C to exit.

### 8. Drive the server from the demo

```
cd ../../demo/architecture-review-demo
MCP_SERVER_BIN=../../examples/microservices-system-design-mcp-server/mcp-server make demo
make validate
```

Three architectures pass through five tools each. Outputs match the captured goldens. End-to-end works.

### 9. Read the demo runner

`demo/architecture-review-demo/demo_runner.py` is the client side. About 250 lines of stdlib Python: spawn the server, complete the MCP handshake, call each tool with shaped input, write the response to a file. The Python is *unaware* of the tool semantics — it just shapes JSON and forwards.

### 10. Read a contract

`contracts/architecture-tools/implemented/generate_service_boundary_canvas.md` is the human-facing specification. JSON schema, semantics, examples, error semantics. The contract is the source of truth that clients and tests refer to.

## How to use the reference for your own work

- **Copy the package layout.** Mirror `internal/services/<name>/` and `internal/tools/<name>/` for every new tool. Don't deviate.
- **Mirror the test shape.** Table-driven; one positive and one negative case per rule.
- **Use the same `register.go` skeleton.** Twenty-five lines, parse-validate-call-format.
- **Copy the Dockerfile.** Multi-stage, distroless, non-root, stripped.
- **Copy the CI workflow.** Lint, vet, race, build.

When you find yourself diverging, ask: is the reference wrong, or is my deviation an anti-pattern in disguise?

## What the reference doesn't demonstrate (yet)

- HTTP transport with auth middleware. The reference is stdio-default; an HTTP variant would add auth, rate limiting, health endpoint, distributed tracing.
- Prompts and resources. The example focuses on tools; future iterations can add prompt and resource examples.
- Multi-tenancy. The reference assumes single tenant; a multi-tenant variant would add tenant isolation.

These gaps are intentional — the reference is meant to demonstrate the most-used patterns, not every possible variant.

## Common failure modes when adopting the reference

- **Copying without understanding.** Team copies the layout, then violates it in the next PR. Detection: business logic shows up in `register.go`; tests live in odd places. Fix: keep a senior reviewer on early PRs to enforce the patterns.
- **Modifying the reference instead of using as template.** Team edits the example directly to add their tool. Detection: example loses its didactic clarity. Fix: vendor the patterns into a new repo; treat the example as read-only reference.
- **Diverging on logging or error patterns.** Team's new tool logs differently from the reference. Detection: dashboards show inconsistent log shapes across tools. Fix: tooling consistency is more valuable than each team's preferences; enforce the pattern.

## Verification questions

1. Can you follow a tool call from `main.go` through `Register` into the service, and back, in under five minutes of reading?
2. Could you add a new tool to the reference by copying one and modifying it?
3. Do `go test ./...` and `go vet ./...` pass on your machine?
4. Does the demo runner work against your build of the server?
5. Could you explain to a new team member why each layer exists?

## What to read next

- `project-structure.md` — the layering rule that the reference embodies
- `code-generation.md` — the templates to start from
- `../../../../validation/governance/release-checklist.md` — the pre-ship runbook
