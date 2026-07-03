# Orchestration Code

This reference covers implementing the orchestration layer in Python — the code that runs the archetype `ai-application-architecture` specified. It assumes the design decisions (archetype, build-vs-buy, framework) are made; this is how to build them.

## Implement the framework the design chose

`ai-application-architecture` decides build-vs-buy — Foundry Agent Service vs a self-built orchestrator — and, when self-built, the framework: Pydantic AI by default, LangGraph for durable graphs. This skill does not re-litigate that; it implements it. If the design says Pydantic AI, build a Pydantic AI orchestrator; if LangGraph, build the graph. Implementing a different framework than the design chose, or drifting into a hand-rolled loop, breaks the design's reasoning silently.

## Pydantic AI — typed agents

Pydantic AI's fit with this pack is its typing: an agent has a typed dependency object and a typed structured-output type, and the framework validates the model's output into that type. Implement each agent as a small, focused unit — one agent, one job — with its system prompt, its output model, and its dependencies declared. Tools the agent can call are typed functions registered with it. The result is orchestration where the model's output is a validated Pydantic object at every step, matching the typed-boundary rule.

## LangGraph — the graph, nodes, durable state

When the design calls for LangGraph, the implementation is an explicit graph: nodes are functions, edges are control flow, and the state threaded through is a typed model. LangGraph's reason to exist is durable execution — implement the checkpointing so a long or human-in-the-loop run survives a process restart and resumes rather than restarting. A LangGraph build that does not use checkpointing is paying LangGraph's complexity for nothing; if you see that, the design wanted Pydantic AI.

## The single model-client wrapper

Every model call in the service goes through one internal client wrapper — not scattered SDK calls. That wrapper owns retries with backoff, the fallback chain, timeouts, the OpenTelemetry span, and per-call cost accounting (`ai-application-architecture`, `references/model-and-inference-layer.md`). One wrapper means those concerns are implemented once and the orchestration code calls a clean, typed method. Scattered raw SDK calls means retry and tracing logic copied and drifting out of sync.

## Keep orchestration out of tool code

Tools — the functions an agent calls, or the MCP tools in the Go tier — do one thing and return a typed result. They do not contain orchestration logic, do not decide what runs next, do not call other agents. Orchestration decides; tools execute. Mixing the two produces tools that cannot be tested or reused and orchestration you cannot follow.

## Keep build-vs-buy reversible

`ai-application-architecture` notes that graduating from Foundry Agent Service to a self-built orchestrator stays cheap only if model access is behind the client wrapper and tool access is behind the typed tool tier. Implement those seams from the start: the orchestrator depends on a model-client interface and a tool interface, not on a specific SDK. Then swapping the managed runtime for a self-built one replaces the orchestration layer between stable seams, rather than rewriting the service.

## Verification questions

1. Does the implementation use the framework `ai-application-architecture` selected — not a different one or a hand-rolled loop?
2. Are agents or nodes small and single-purpose, with typed dependencies and typed structured output?
3. If LangGraph: is durable checkpointing actually implemented and used?
4. Does every model call go through one internal client wrapper owning retries, fallback, timeout, tracing, and cost?
5. Do tools execute one job and return a typed result, with no orchestration logic inside them?
6. Are model access and tool access behind interfaces, keeping build-vs-buy reversible?

## What to read next

- `ai-application-architecture`, `references/orchestration-frameworks.md` — the design decision this implements
- `typing-and-pydantic.md` — the typed agents and outputs
- `async-and-concurrency.md` — async orchestration
- `mcp-go-server-building` — the Go tool tier orchestration calls
