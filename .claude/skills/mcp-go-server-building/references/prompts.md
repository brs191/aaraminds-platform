# Skill — MCP-Go Prompts

## Purpose

Design MCP prompts to expose pre-shaped, parameterised prompts (or "slash commands") that agents and clients can discover and invoke. Prompts are the user-facing entry points to recurring tasks; they convert ad-hoc free-text into structured, repeatable interactions. This skill is about when prompts pay, how to shape them, and what regressions to watch for.

## What prompts are and aren't

A prompt in MCP is a server-defined template that the client discovers and the user (or agent) instantiates with arguments. Think slash commands in a chat UI — `/review-deployment service=payment env=prod` becomes a structured request.

Prompts are not:
- A pipeline of tool calls. (That's an orchestration; use a tool that runs the workflow.)
- A way to pass system instructions to the LLM. (Those live in client-side configuration.)
- A substitute for tools. Prompts don't act; they shape the LLM's input.

Prompts *are*:
- Discoverable templates that show up in the client's UI as commands.
- Parameterised: the user fills in slots, the server expands them.
- Reproducible: the same arguments produce the same prompt text every time.

## When prompts pay

- Recurring tasks where the prompt shape matters and ad-hoc phrasing drifts ("review this deployment for resilience gaps" said five different ways).
- Workflows that depend on consistent framing — security review, runbook generation, architecture critique.
- Templates with non-trivial structure where the user shouldn't have to remember every section.
- Multi-step elicitations where the prompt also instructs the LLM how to interact with available tools.

When prompts *don't* pay:
- One-off requests; the user can phrase them naturally.
- Free-form exploration; constraining the input hurts.
- When the variation across calls is too high to template usefully.

## Anatomy of a good prompt

```
Name:          review-deployment
Description:   Review a service's deployment posture for resilience and rollback gaps
Arguments:
  service:     name of the service (required)
  environment: prod | staging | dev (required, default: prod)
  date_range:  ISO date range for incident history (optional)
Template:
  You are reviewing the deployment posture of {{service}} in {{environment}}.
  Focus on resilience controls (retries, circuit breakers, bulkheads), rollback story
  (blue-green, canary, feature flags), and observability gaps.
  {{#if date_range}}
  Include incident patterns from {{date_range}}.
  {{/if}}
  Use the following tools as needed: detect_architecture_risks, generate_observability_plan.
  Return: a structured review with prioritised findings.
```

Key properties:
- **Name** is verb-led and stable across versions.
- **Description** tells the user when to pick this prompt.
- **Arguments** are typed (string, enum, optional). Defaults reduce friction.
- **Template** is deterministic given the arguments. No randomness.
- The template can *mention* tools but doesn't call them — that's the LLM's job after expansion.

## Go implementation

```go
// internal/prompts/deployment/prompt.go
package deployment

import (
    "context"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func Register(s *server.MCPServer) {
    s.AddPrompt(
        mcp.NewPrompt("review-deployment",
            mcp.WithPromptDescription("Review a service's deployment posture for resilience and rollback gaps."),
            mcp.WithArgument("service",
                mcp.ArgumentDescription("Name of the service to review"),
                mcp.RequiredArgument(),
            ),
            mcp.WithArgument("environment",
                mcp.ArgumentDescription("Target environment: prod | staging | dev"),
            ),
            mcp.WithArgument("date_range",
                mcp.ArgumentDescription("Optional ISO date range for incident lookback"),
            ),
        ),
        func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
            service := req.Params.Arguments["service"]
            env := req.Params.Arguments["environment"]
            if env == "" {
                env = "prod"
            }
            dateRange := req.Params.Arguments["date_range"]

            body := fmt.Sprintf(
                "You are reviewing the deployment posture of %s in %s.\n\n"+
                    "Focus on resilience controls, rollback story, and observability gaps.\n",
                service, env)
            if dateRange != "" {
                body += fmt.Sprintf("Include incident patterns from %s.\n", dateRange)
            }
            body += "\nUse detect_architecture_risks and generate_observability_plan as needed.\n"

            return &mcp.GetPromptResult{
                Description: "Deployment posture review",
                Messages: []mcp.PromptMessage{
                    {
                        Role:    mcp.RoleUser,
                        Content: mcp.TextContent{Type: "text", Text: body},
                    },
                },
            }, nil
        },
    )
}
```

The handler validates required arguments, applies defaults, and expands the template. Output is a list of messages; for most prompts it's a single user message with the expanded body.

## Worked example: choosing prompt vs. tool

Scenario: the team wants a "generate ADR" feature.

- If the agent should *produce* the ADR with a specific structure: a **prompt** templated with the system context, decision, and alternatives slots. The LLM does the drafting.
- If the server should produce a templated ADR by itself (no LLM drafting): a **tool** named `generate_adr_template` that returns the structured skeleton.
- If both — let the LLM call the tool to get the skeleton, then fill it in — keep them separate: a tool for the skeleton, a prompt that includes "use generate_adr_template to fetch the structure."

## Common failure modes

- **Prompt that calls tools internally.** Some implementations sneak tool calls into prompt-rendering logic. Detection: the prompt handler imports the tools package or has business logic. Fix: prompts produce text; tools execute. Keep the boundary.
- **Untyped arguments leading to silent failures.** Missing required argument returns an empty prompt instead of an error. Detection: a prompt expansion that contains `{{service}}` literal. Fix: validate required arguments, return a structured error.
- **Hidden state.** The prompt template loads "the latest config" from somewhere, so the same arguments produce different outputs across calls. Detection: non-reproducible expansions. Fix: prompts are pure functions of their arguments; data they need should be passed in or fetched via tools.
- **Prompt that's also documentation.** The description balloons into a 500-word manual. Detection: prompt descriptions longer than three sentences. Fix: the prompt is for discovery; documentation lives elsewhere.
- **Versioning forgotten.** The prompt's template evolves and breaks existing user scripts. Detection: complaints that "the slash command output changed". Fix: bump prompt name or argument when the contract changes; treat prompts as a versioned API.

## Verification questions

1. For each prompt, can a user invoke it correctly given only its name and arguments?
2. Are required arguments validated before expansion? What happens if they're missing?
3. Is the template deterministic given the arguments? (Same in, same out, every time?)
4. Does the prompt avoid calling tools during rendering?
5. Is the prompt versioned, or will template evolution silently break consumers?

## What to read next

- `tool-design.md` — when prompts won't do, design a tool
- `resources.md` — the other read-side primitive
- `project-structure.md` — where prompts live in the package layout
