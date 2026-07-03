# Skill — MCP-Go Testing

## Purpose

Define a layered testing strategy for MCP-Go servers that catches bugs at the right altitude. Most MCP server defects are caught by a handful of well-shaped unit and contract tests — but only if they exist at the layers that matter. This skill names those layers, what each one is responsible for, and what regressions each one catches.

## The eight-layer test pyramid

```
       Deployment Smoke Tests       ← integration in target env
      Security & Threat Tests       ← OWASP-shaped attacks
     Transport / Protocol Tests     ← MCP wire conformance
    MCP Contract Tests              ← schemas, error codes, semantics
   Connector / Mock Tests           ← external systems mocked
  Service Layer Tests               ← rule logic, table-driven
 Tool Handler Tests                 ← parsing, validation, errors
Unit Tests                          ← functions, small types
```

The shape: many fast unit tests at the bottom, few expensive smoke tests at the top. If you have more contract tests than unit tests, the pyramid is upside down.

## Layer 1 — Unit tests

For pure functions: helpers, validators, formatters. Run in milliseconds; no I/O.

```go
func TestValidateServiceName(t *testing.T) {
    tests := []struct {
        name    string
        in      string
        wantErr bool
    }{
        {"valid", "order-service", false},
        {"empty", "", true},
        {"too long", strings.Repeat("a", 64), true},
        {"invalid char", "order_service!", true},
        {"leading dash", "-order", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateServiceName(tt.in)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateServiceName(%q) err=%v, wantErr=%v", tt.in, err, tt.wantErr)
            }
        })
    }
}
```

Table-driven. The case name is documentation of what's being checked.

## Layer 2 — Tool handler tests

The MCP handler's job is parse → validate → call service → format. Test each handler with:

- A valid input passes through to the service.
- A missing required argument returns `mcp.NewToolResultError(...)` with a specific message.
- Invalid JSON in `input_json` returns a parsing error, not a panic.
- Service-returned errors propagate to the tool result.

```go
func TestBoundaryHandler_MissingInput(t *testing.T) {
    s := boundary.NewService(nil)
    h := newTestHandler(s, slog.Default())

    res, err := h(context.Background(), mcp.CallToolRequest{
        Params: mcp.CallToolParams{Arguments: map[string]any{}},
    })
    if err != nil {
        t.Fatalf("expected nil error, got %v", err)
    }
    if !res.IsError {
        t.Fatal("expected IsError=true")
    }
}
```

Don't mock the MCP framework itself — test through `mcp.CallToolRequest`.

## Layer 3 — Service-layer tests

The largest, richest layer. The service is plain Go code; tests run in milliseconds and exercise the rule logic exhaustively.

```go
func TestGenerateCanvas(t *testing.T) {
    cases := []struct {
        name     string
        input    boundary.Input
        wantScore int
        wantRisks []string
    }{
        {
            name: "clean boundaries",
            input: boundary.Input{...},
            wantScore: 100,
            wantRisks: nil,
        },
        {
            name: "shared data ownership",
            input: boundary.Input{...},
            wantScore: 75,
            wantRisks: []string{"data_co_ownership"},
        },
        // ... 5–10 more cases
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

Cover every named rule (capability cohesion, data ownership, dependency hygiene, ownership clarity, size sanity). Each rule deserves at least one positive and one negative case.

## Layer 4 — Connector / mock tests

If the service calls external systems (Azure SDK, databases, HTTP APIs), test the connector with a mock:

- Define an interface for the dependency.
- Implement a mock in `internal/services/<name>/mock_test.go`.
- Test the service against the mock; test the real connector in a separate integration test layer.

The aim: service tests run without network. Integration tests are slower and may be skipped in tight loops.

## Layer 5 — MCP contract tests

Verify the server adheres to the MCP protocol:

- `initialize` request → response includes capabilities.
- `tools/list` returns the expected catalog.
- `tools/call` with a known tool and input produces a structured result.
- Error responses match the JSON-RPC error schema.

```go
func TestContract_ToolsList(t *testing.T) {
    s := mcpserver.NewServer(slog.Default())
    // Drive via test transport (in-process)
    result, err := s.ListTools(context.Background(), mcp.ListToolsRequest{})
    if err != nil { t.Fatal(err) }
    want := []string{
        "generate_service_boundary_canvas",
        "generate_api_contract",
        "detect_architecture_risks",
        // ...
    }
    if !sameTools(result.Tools, want) {
        t.Errorf("unexpected tool catalog: %v", result.Tools)
    }
}
```

Contract tests catch broken registrations and tool-catalog regressions.

## Layer 6 — Transport tests

Verify the chosen transport doesn't corrupt MCP frames or violate operational constraints:

- **Stdio:** any byte written to stdout outside of an MCP message is a bug. Test: spawn the server, send a request, assert stdout contains only valid framed responses.
- **Streamable HTTP:** the server respects HTTP semantics (status codes, content-type), JSON-RPC frames roundtrip, server-side connection close is clean.

These tests are slower; one or two per transport is enough.

## Layer 7 — Security tests

OWASP-shaped attacks specific to MCP servers:

- **Oversized input:** send 100 MB `input_json` — the server rejects at size limit, doesn't OOM.
- **Schema-bypass input:** send fields that look like a different tool's input — server rejects, doesn't dispatch.
- **Path traversal in identifiers:** `service_id: "../../../etc/passwd"` — server rejects.
- **Authorization bypass:** without auth context, every state-changing tool denies.
- **Prompt-injection content in tool inputs:** server processes as data, returns sanitised output; the *agent* layer is responsible for treating output as untrusted.

Document these as a security test suite that runs in CI.

## Layer 8 — Deployment smoke tests

Run after deploy:

- The container starts and is healthy within N seconds.
- `tools/list` returns the expected count.
- One known-good tool call returns a structured response.
- Logs are flowing to the configured sink.

Smoke tests are the last gate before traffic; they prove the deployment artifact works, not just compiles.

## CI checks (required)

Every PR runs:

```bash
go test ./...
go test -race ./...
go vet ./...
gofmt -l . | tee /dev/stderr | (! grep .)
```

The fourth command fails the build if any file isn't gofmt-clean; the `tee` makes the offenders visible.

Coverage isn't a goal in itself; meaningful tests are. A 95% coverage report on hollow tests is worthless. Target: every named rule in service layers has tests; every error path in handlers has tests.

## Common failure modes

- **Tests against the framework, not the code.** A test that mocks the MCP server and asserts the framework called the handler. Detection: test references `MCPServer` directly with mocks. Fix: test the handler function as a unit.
- **No table-driven coverage.** Each test is a separate function repeating boilerplate. Detection: dozens of tiny `Test*` functions with copy-pasted setup. Fix: refactor into table-driven cases.
- **Integration tests masquerading as unit tests.** A "unit" test that opens a database. Detection: tests fail without external services. Fix: mock the dependency or move the test to the integration layer.
- **Missing negative cases.** Every test asserts "happy path works"; nothing tests failure modes. Detection: PR introducing a validation rule has no corresponding negative test. Fix: every rule has positive and negative cases.
- **Tests that snapshot byte-exact output without a refresh story.** Server output changes; tests fail; team starts to update goldens reflexively without reviewing. Detection: PRs that touch goldens with no rationale. Fix: golden refresh is its own decision with a documented justification.

## Verification questions

1. Does every named rule in the service layer have at least one positive and one negative test?
2. Do tool handlers test missing/invalid input paths, not just success?
3. Is there a contract test that catches a tool being accidentally unregistered?
4. Does the security test suite exist? Is it run in CI?
5. Is there a smoke test that runs after deploy?
6. Are tests independent of the framework — i.e., a framework upgrade doesn't break tests for unrelated reasons?

## What to read next

- `../../mcp-go-server-building/references/tool-design.md` — what tools should accept and validate
- `../../mcp-go-server-building/references/project-structure.md` — co-located tests next to service code
- `cicd-quality-gates.md` — wiring tests into CI gates
- `../../mcp-go-threat-modeling/references/security-test-generation.md` — generating security test inputs
