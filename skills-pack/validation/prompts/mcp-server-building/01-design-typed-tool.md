---
id: mcp-server-building/01-design-typed-tool
area: mcp-server-building
exercises:
  - .claude/skills/mcp-go-server-building/references/tool-design.md
  - .claude/skills/mcp-go-server-building/references/project-structure.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Design a typed-input MCP tool

## Context

The responder is an engineer adding a new tool to an existing Go MCP server built on `github.com/mark3labs/mcp-go`. Attach `02-mcp-go-tool-design.md` and `06-mcp-go-project-structure.md` as context. The responder should propose code structure, not write the full implementation.

## Prompt

I'm adding a new MCP tool called `generate_runbook_template` to a Go MCP server. It takes a structured input describing an incident class (severity, affected services, on-call rotation, recovery objectives) and returns a runbook skeleton (sections, suggested commands, escalation rules).

Design the tool the way you'd recommend: where the code lives, what types it uses, how the registration is wired, and what the input/output contract looks like. Do not write the whole implementation — just the structure and the key decisions. Justify each choice briefly.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Separates service from registration** — places business logic in `internal/services/runbook/` and MCP wiring in `internal/tools/runbook/` (or equivalent package split)
- [ ] **Typed input struct** — defines a Go struct (not just `map[string]any`) for the input and references the JSON-schema-shaped contract; calls out that input parsing uses `json.Unmarshal` into the struct
- [ ] **Single-purpose tool** — the tool does one thing well and is named with an action verb (`generate_`, not `runbook_helper`); no muxing of unrelated capabilities into one tool
- [ ] **Mentions risk tier** — names the operation as low-risk / informational (it's a generator, not a state-changer) and decides on logging / dry-run accordingly
- [ ] **Error result shape** — returns `mcp.NewToolResultError(...)` (or equivalent) on validation failure rather than throwing or returning nil; differentiates validation errors from internal errors
- [ ] **Tests-near-rules** — proposes table-driven tests in `internal/services/runbook/service_test.go` covering at least: empty input, missing required field, well-formed input
- [ ] **Contract file** — locates the JSON contract under `contracts/architecture-tools/implemented/generate_runbook_template.md` (or equivalent path), separate from code
- [ ] **Avoids known anti-patterns** — no kitchen-sink tools, no untyped `interface{}` arguments leaking through, no business logic in `register.go`, no logger as a package-global

## Reference output

> I'd structure it like the other tools in this server: one Go package for the rule-based logic and one for the MCP wiring.
>
> **Layout**
> ```
> internal/services/runbook/
>   service.go          // Input/Output types, Validate, Generate
>   service_test.go     // table-driven tests
> internal/tools/runbook/
>   register.go         // MCP tool definition and handler
> contracts/architecture-tools/implemented/
>   generate_runbook_template.md   // contract spec, separate from code
> ```
>
> **Why split this way**
> The `register.go` handler stays thin: parse JSON → validate → call service → format result. All decisions about what a runbook should contain live in `service.go`. This keeps the MCP-aware code small enough to skim and makes the rule logic unit-testable without spinning up an MCP server. It matches the pattern in `internal/services/boundary/` and avoids the anti-pattern of stuffing business logic into the tool registration.
>
> **Input shape**
> ```go
> type Input struct {
>     IncidentClass    string   `json:"incident_class"`
>     Severity         string   `json:"severity"`
>     AffectedServices []string `json:"affected_services"`
>     OnCallRotation   string   `json:"on_call_rotation,omitempty"`
>     RTOMinutes       int      `json:"rto_minutes,omitempty"`
>     RPOMinutes       int      `json:"rpo_minutes,omitempty"`
> }
> ```
> Typed, not `map[string]any`. The MCP wire still sends `input_json` as a string for SDK compatibility, but `register.go` unmarshals into this struct so the service never sees raw JSON. `Validate(input) error` runs before generation and returns user-facing messages; the handler turns those into `mcp.NewToolResultError`.
>
> **Output shape**
> ```go
> type Runbook struct {
>     Title       string
>     Sections    []Section
>     Escalation  []EscalationStep
>     SuggestedCommands []SuggestedCommand
>     Score       int    // confidence in completeness, 0-100
> }
> ```
> Structured output — not free-form markdown — so callers can render it however they like and so tests can compare deterministically.
>
> **Risk tier and logging**
> This is a generator: it reads input, produces a runbook, never writes external state. Low-risk / informational tier per the tool-design skill. Log: tool call start, completion with `incident_class` and `services_count` as structured fields. No dry-run mode needed; no audit emitter beyond standard logs.
>
> **Tests**
> Table-driven in `service_test.go`. Minimum cases: (1) empty input → validation error, (2) missing `severity` → specific error message, (3) typical input → generates expected section count and escalation steps, (4) all-optional-fields-empty → still produces a usable runbook.
>
> **What I'd avoid**
> Don't make this tool also "execute the runbook" — that's a state-changing tool and belongs in a separate file with its own risk tier. Don't pass a global logger; thread it through `Register(s, svc, logger)` as the other tools do. Don't accept `interface{}` arguments — the typed struct catches schema errors at unmarshal time, not deep in the rule logic.

## Notes

This prompt is the highest-leverage check that an LLM is actually applying the project-structure and tool-design guidance, not just regurgitating boilerplate. The rubric is structure-and-rationale based, not content-based, so it catches whether the response engages with the skill's recommendations versus generating generic MCP advice.
