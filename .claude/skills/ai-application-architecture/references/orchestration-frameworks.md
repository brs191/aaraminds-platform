# Orchestration Frameworks

Orchestration is the code that decides which model calls, retrievals, and tool invocations happen, in what order, with what state carried between them. This reference covers the build-vs-buy decision (hosted Foundry Agent Service vs a self-built orchestrator) and, when self-built, the framework choice. It assumes Stack 3 — orchestration is Python.

## Build vs buy: start on Foundry Agent Service

Default to **Microsoft Foundry Agent Service** — hosted agents on the OpenAI Responses API, with managed conversation memory and managed evaluations that monitor into Azure Monitor. It removes three bodies of code you would otherwise own and operate: the orchestration runtime, the memory and thread store, and the monitoring wiring. For most features that is the right trade — you are not in the business of operating an agent runtime, and the managed one is a smaller attack surface and a smaller on-call burden than a bespoke one.

Buying the managed runtime first also de-risks the build: you ship the feature, learn its real orchestration shape from production traffic, and only then decide whether a self-built runtime is justified. Self-building on day one to avoid a managed dependency you may never outgrow is premature — it spends weeks of runtime engineering before the feature has taught you what it needs.

## When to graduate to a self-built orchestrator

Move to a self-built Python orchestrator when you hit a concrete wall, not on principle:

- **Orchestration logic the hosted runtime cannot express** — control flow, branching, or step composition the managed loop does not support.
- **Portability** — you need to run off the Responses API, across providers, or on infrastructure the hosted service does not reach.
- **Per-step control** — you need to inspect, gate, cache, or modify individual steps in a way the managed loop does not expose.

Each of these is a real, nameable limitation. "We might need flexibility later" is not one — it is the premature-build argument in disguise. When you do graduate, the migration is contained if you kept model and tool access behind internal interfaces from the start (see below).

## The framework is an explicit decision

When the design calls for a self-built orchestrator, the framework is a decision to *state and justify in the design*, not a default to drift past because someone already imported one. Record the choice and the reason. The pack's options:

### Pydantic AI — the default self-built choice

**Pydantic AI** is the default. It is type-safe, structured-output-first, and has FastAPI-style ergonomics — which matches the pack's typed, disciplined house style and the typed-seam rule in `serving-topology.md`. Reach for it for single-shot nodes (`patterns/single-shot.md`), RAG generation (`patterns/rag.md`), short workflows, and agentic loops that do not need durable, resumable execution. It is the right choice for the majority of self-built orchestration in this stack.

### LangGraph — when you need durable, stateful graphs

Choose **LangGraph** when the orchestration genuinely needs explicit stateful-graph control: durable execution that survives a process restart, resumable long-running runs, branching and retry as first-class graph constructs, or human-in-the-loop pauses where a run suspends pending an external decision. A deterministic workflow that must checkpoint per node and resume after failure (`patterns/llm-workflow.md`), or an agentic loop that runs long enough to need durability, is LangGraph's case. Do not reach for it for a short stateless pipeline — its durable-graph machinery is overhead you are not using.

### LlamaIndex and Microsoft Agent Framework — the niche picks

**LlamaIndex** is justified when the workload is RAG-dominant and you want its retrieval and indexing abstractions as the spine. **Microsoft Agent Framework** is justified for Azure-native multi-agent orchestration where Python / Java / C# parity across the team matters. Both are deliberate, narrower choices — name the specific reason or use the default.

### Decision matrix

| Need | Choice |
|---|---|
| A managed runtime, memory, and monitoring with no orchestration code | Foundry Agent Service |
| Self-built; typed single-shot, RAG, short workflow, simple loop | Pydantic AI |
| Self-built; durable execution, resumable long runs, human-in-the-loop, first-class branching | LangGraph |
| Self-built; RAG-dominant workload built around retrieval abstractions | LlamaIndex |
| Self-built; Azure-native multi-agent with Python/Java/C# parity | Microsoft Agent Framework |

Go's role in this stack is the gateway and the MCP tool tier (`mcp-go-server-building`), not orchestration. Do not reach for a Go agent framework to keep orchestration in Go — that fights the stack and the decision recorded in `.claude/CLAUDE.md`.

## Generators before agents — earn the orchestration

The heaviest orchestration decision is whether to orchestrate at all. Multi-agent orchestration — agents calling agents, dynamic control flow — buys latency, cost, and new failure modes (non-termination, tool thrash, cost runaway; see `patterns/agentic-loop.md`). It must be *earned* by a requirement, not assumed because "agent" is the fashionable noun.

Build a capability as a document or output **generator over a fixed workflow** (`patterns/llm-workflow.md`) first. Promote it to a fully orchestrated agent only when interactive, iterative behaviour is genuinely required — when the user must steer the process mid-run, or the control flow truly cannot be drawn ahead of time. The Code Intelligence Factory makes this call explicitly: its BA and QA capabilities ship as generators over the knowledge graph, and the brief defers orchestrated-agent promotion until interactivity is a real requirement. That is the correct default sequencing for any capability in this pack.

## Migration: managed to self-built without a rewrite

The graduation from Foundry Agent Service to a self-built orchestrator is cheap or expensive depending on one decision made early: keep model access behind a single internal client (`model-and-inference-layer.md`) and tool access behind the typed MCP tool tier (`mcp-go-server-building`) from day one. With those seams in place, migration replaces the orchestration layer between them and leaves the model client, the tools, and the serving tiers untouched. Without them, orchestration logic leaks into tool calls and model calls, and the migration becomes a rewrite. The interfaces are the same discipline the pack applies everywhere — they are also what makes the build-vs-buy decision reversible.

## Verification questions

1. Was build-vs-buy decided explicitly — Foundry Agent Service vs self-built — with a stated reason?
2. If self-built, is the framework named in the design (Pydantic AI by default; LangGraph, LlamaIndex, or Microsoft Agent Framework with a specific reason)?
3. If LangGraph was chosen, is there a concrete need for durable or resumable execution — not just "flexibility later"?
4. Is the capability built as a generator over a fixed workflow before being promoted to an orchestrated agent?
5. Are model access and tool access behind internal interfaces, so build-vs-buy stays reversible?
6. Is orchestration in the Python tier — with Go confined to the gateway and tool tier?

## What to read next

- `patterns/agentic-loop.md` and `patterns/llm-workflow.md` — the archetypes orchestration runs
- `serving-topology.md` — where the orchestration tier sits and how it connects
- `model-and-inference-layer.md` — the model client interface that keeps migration cheap
- `mcp-go-server-building` — the Go tool tier orchestration calls
