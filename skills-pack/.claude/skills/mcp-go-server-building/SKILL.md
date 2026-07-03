---
name: mcp-go-server-building
description: Designs and builds Go MCP (Model Context Protocol) servers — server skeleton and tool design, project structure, transport choice, observability, security, and agent/client integration. Covers SDK choice (mark3labs/mcp-go vs official go-sdk), stdio vs streamable HTTP transports, the typed-input / service-package layering rule, and MCP resources and prompts. Use when starting a new Go MCP server, adding a tool, choosing transport, designing the tool catalog, or evaluating an integration. Do not use for pre-production review and CI gates (use mcp-go-production-review) or threat modeling (use mcp-go-threat-modeling).
version: 1.0.1
last_updated: 2026-05-30
---

# MCP Go Server Building

## When to use

Trigger this skill when the question is about *building* a Go MCP server: SDK selection, server skeleton, tool design, project structure, transport, MCP resources, MCP prompts, observability inside the server, server-side security primitives, code-generation templates, agent integration (server's view of agent discovery), client integration (when you write the client), and reference walkthroughs. Common triggers: "set up a new MCP server in Go," "add a tool to this server," "stdio or HTTP for this deployment," "tool description isn't getting picked up by Claude," "where should the contract file live."

Do **not** use this skill for: pre-production readiness and CI gates (`mcp-go-production-review`); threat modeling and security tests (`mcp-go-threat-modeling`); microservices-level concerns (the `microservices-*` skills); Azure-service selection beyond hosting the server (`azure-service-mapping`).

## The critical decision rule — stdout is the protocol wire under stdio

Under stdio transport (the default for local agent integration), **any byte written to stdout corrupts an MCP message frame**. This is the single most common new-server bug. The fix is enforced at server bootstrap:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
slog.SetDefault(logger)
```

Stderr only. No `fmt.Println`, no default-init loggers writing to stdout, no debug prints "just for now." If a tool appears to be silently failing in Claude Code, this is the first thing to check.

## The build-time decision tree

| Decision | Default | Reference |
|---|---|---|
| Which SDK | `github.com/mark3labs/mcp-go` for this pack's example consistency; otherwise the official `modelcontextprotocol/go-sdk` for greenfield | `references/server-basics.md`, `references/ecosystem-facts.md` |
| Which transport | stdio for local / agent use; streamable HTTP for remote / multi-client (SSE is deprecated) | `references/transport-selection.md` |
| Project layout | `cmd/server/main.go` + `internal/mcpserver/` + `internal/services/<name>/` + `internal/tools/<name>/` | `references/project-structure.md` |
| Tool surface | One verb-led action per tool, typed Go input struct, contract file per tool | `references/tool-design.md` |
| MCP resources (read-only context) | Use for read-only domain context the LLM needs; URI-shaped, opaque to caller | `references/resources.md` |
| MCP prompts (slash commands) | Use for parameterized templates the user invokes by name; deterministic | `references/prompts.md` |
| Code generation templates | `cmd/server/main.go` skeleton, service skeleton, tool registration skeleton, Dockerfile, CI | `references/code-generation.md` |
| Cross-cutting | Enterprise security at the transport boundary; observability per tool call | `references/enterprise-security.md`, `references/observability.md` |
| Agent integration (server side) | Tool description as agent's only docs; JSON schema balanced (not permissive, not hostile) | `references/agent-integration.md` |
| Client integration (when you write a client) | Hand-rolled JSON-RPC over stdio is ~250 lines stdlib Python; the demo in this pack is the reference | `references/client-integration.md` |
| End-to-end demo design | Authentic: real protocol exchange, content-distinct outputs across inputs; not hardcoded fixtures | `references/e2e-agent-demo.md` |
| Reference implementation walkthrough | Read `examples/microservices-system-design-mcp-server/` top-down | `references/reference-implementation.md` |
| Runnable domain repos | Each example is a self-contained Go module under `examples/`; don't import across them | `references/runnable-domain-repos.md` |

For the dated ecosystem facts (Go version, MCP SDK versions, MCP spec version, stable transport choices), see `references/ecosystem-facts.md`.

## Build-time logic

1. **Start from the reference.** The pack's `examples/microservices-system-design-mcp-server/` is the canonical worked example: server skeleton, 13 tools, table-driven tests, multi-stage Dockerfile, GitHub Actions CI. Copy its structure. See `references/reference-implementation.md`.

2. **Lock the layering rule.** Three packages per tool: `internal/services/<name>/service.go` (rule logic, no MCP imports), `internal/services/<name>/service_test.go` (table-driven tests for every named rule), `internal/tools/<name>/register.go` (MCP wiring — parse-validate-call-format, ~25 lines). If `register.go` grows past 50 lines, business logic has leaked out of the service. See `references/project-structure.md`.

3. **Tool design discipline.**
   - One action per tool. No `manage_*` or `do_*` kitchen-sink tools that dispatch on an `action` string. Each action becomes its own tool with its own typed input.
   - Verb-led, intent-clear names: `generate_service_boundary_canvas`, not `boundary_service`.
   - Typed Go input struct (never `map[string]any` at the handler). Validation lives in the service's `Validate` method, not in the handler.
   - Contract file at `contracts/architecture-tools/implemented/<name>.md` describing inputs, outputs, errors, examples — the human-readable spec.
   - See `references/tool-design.md`.

4. **Tool description writing (this matters more than people think).** The description is the agent's *only* documentation for when to call the tool. Include: what it does (verb-led), when to use it (trigger condition), what it does *not* do (boundaries), what it returns (output shape). See `references/agent-integration.md`.

5. **Transport choice.** stdio for local agent integration (Claude Code, CLI workflows). Streamable HTTP for remote / multi-client deployments. Never SSE for new work — deprecated in the 2025-11-25 spec. See `references/transport-selection.md`.

6. **Resources vs. tools.** Read-only context exposed via URI (e.g., `docs://architecture/order-platform`) is a resource. Anything that performs an action — even a read with significant side effects (audit, billing) — is a tool. See `references/resources.md`.

7. **Prompts.** MCP prompts are parameterized templates the user invokes by name (`/review-deployment service=order-service env=prod`). Deterministic given args. Don't reach for prompts when a tool will do; use prompts for repeatable interaction shapes. See `references/prompts.md`.

8. **Observability per tool call.** Emit a `tool_call_started`, `tool_call_completed` or `tool_call_failed` event per invocation with `tool`, `latency_ms`, derived domain fields, and (if error) error class. Stderr-routed JSON. See `references/observability.md`.

9. **Server-side security.** Auth at transport. Per-tool authorization beyond auth. Risk tier per tool (informational / read / write / destructive). Sanitize sensitive fields in audit emissions. See `references/enterprise-security.md`.

## Worked example — brownfield: adding `generate_runbook_template` to an existing server

Setup: existing Go MCP server `examples/microservices-system-design-mcp-server/` with 13 tools using the standard layering. Need to add a 14th tool that generates an incident runbook skeleton from a structured input (incident class, affected services, RTO/RPO).

Decision walk:

1. **Service package first.** Create `internal/services/runbook/service.go` with `Input`, `Output`, `Validate`, `Generate`. No MCP imports. Write the table-driven `service_test.go` first; cover positive cases (typical input → expected sections), negatives (missing required field, severity out of enum), edge cases (empty `affected_services`, optional RTO). Tests pass under `go test -race -count=1`. See `references/project-structure.md`.
2. **Contract document.** `contracts/architecture-tools/implemented/generate_runbook_template.md`. Document input fields with types and constraints; output structure; error semantics; one full example. See `references/tool-design.md`.
3. **Tool registration.** `internal/tools/runbook/register.go`. ~25 lines: `mcp.NewTool` with description (verb-led, intent-clear), `mcp.WithInputSchema` with `runbook.Input`, handler that does parse → validate → call service → format. No business logic. See `references/code-generation.md` for the skeleton.
4. **Description for the agent.** "Generate an incident runbook skeleton for a Go microservice. Takes incident class (cpu-saturation, memory-leak, disk-pressure, latency-spike), affected services, RTO/RPO targets. Returns structured runbook with severity-keyed escalation paths, common-cause hypotheses, and rollback steps. Use when designing or reviewing incident response for a service. Does not generate live runbook content from production telemetry — that's a separate planned tool." See `references/agent-integration.md`.
5. **Wire into the composition root.** Add the registration call in `internal/mcpserver/server.go`. Single import + single call.
6. **Verify the build.** `go vet ./...`, `go test -race -count=1 ./...`, build the binary, smoke-test with the demo runner (or a one-off Python script) — call `tools/list`, verify the new tool appears with the right schema, then call it with a fixture and check the output.
7. **Risk tier and audit.** Risk tier: informational (read-shaped; produces a markdown skeleton, no external side effects). Audit emission: standard tool-call event; no special handling. See `references/enterprise-security.md`.

## Anti-pattern — kitchen-sink tool with `action` dispatch

**Bad:** A single tool `manage_orders` with input `{"action": "create"|"cancel"|"list", "args": {...}}` that dispatches internally on `action`.

**Why it fails:**
- The agent can't find the tool from its description. "Manages orders" is too vague. Three separate tools with verb-led names — `create_order`, `cancel_order`, `list_orders` — let the agent recognize the trigger.
- The input type becomes `map[string]any` because args differ per action; validation is now stringly-typed and easy to bypass.
- Risk tiers differ across actions (`create_order` is write, `list_orders` is read) but the kitchen-sink tool has one tier for all.
- Audit log entries blur together: it's harder to find "every create call" because the tool name is uniform.

**Detection signal:** a tool whose input has a discriminator field (`action`, `op`, `cmd`) and whose service has a switch/case dispatching on it. Or: a tool description that uses the word "manage" or "operate."

**Fix:** Split into per-action tools. Each has its own typed input, own risk tier, own audit shape, own description. The agent can now reason about which one to call.

## Verification questions

1. Is the logger initialized to stderr at server startup? Search `slog.New` calls; verify `os.Stderr`.
2. For each tool: is the description verb-led with a clear when-to-use trigger, or does it read like a noun?
3. For each tool: is the input a typed Go struct, not `map[string]any`?
4. For each tool: is there a contract file at `contracts/architecture-tools/implemented/<name>.md`?
5. Does `register.go` stay under 50 lines per tool? If not, business logic has leaked into the handler.
6. Is the transport choice (stdio vs. streamable HTTP) explicit in the deployment, and does it match the use case?
7. For each tool that writes or has side effects: is the risk tier set and the audit emission configured?

## What to read next

- `references/server-basics.md` — minimal `main.go`, slog wiring, graceful shutdown
- `references/ecosystem-facts.md` — dated Go / SDK / MCP-spec versions; freshness rules
- `references/tool-design.md` — typed inputs, verb-led naming, kitchen-sink anti-pattern detail
- `references/project-structure.md` — `internal/services` vs. `internal/tools` layering rule
- `references/transport-selection.md` — stdio vs. streamable HTTP, when to choose each
- `references/resources.md` — MCP resources as read-only context; URI design
- `references/prompts.md` — MCP prompts as templates; deterministic interaction shapes
- `references/enterprise-security.md` — server-side auth, authz, risk tier, audit emission
- `references/observability.md` — per-tool-call event log, derived metrics, alert design
- `references/agent-integration.md` — server's view of agent discovery; description as docs
- `references/client-integration.md` — when you write the client; stdlib Python reference
- `references/code-generation.md` — generation templates for new servers and new tools
- Reference implementation context: `references/reference-implementation.md`, `references/e2e-agent-demo.md`, `references/runnable-domain-repos.md`
- `mcp-go-production-review` skill — pre-deploy review for the server you just built
- `mcp-go-threat-modeling` skill — security testing and STRIDE for the tool surface
