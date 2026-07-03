---
name: python-service-engineering
description: Builds production Python services — the orchestration, AI, and evaluation tier. Project structure, Pydantic types at every boundary, async and concurrency, the orchestration-framework code (Pydantic AI, LangGraph) ai-application-architecture selects, and runtime concerns (config, secrets, telemetry). Companion to ai-application-architecture. Use when implementing a Python service, structuring a project, adding type/Pydantic discipline, writing async orchestration, or wiring config and telemetry. Do not use for AI archetype/serving design (use ai-application-architecture), Go/MCP servers (use mcp-go-server-building), or the eval harness (use ai-evaluation-harness).
version: 1.0.1
last_updated: 2026-05-30
---

# Python Service Engineering

## When to use

Trigger this skill when building a Python service — production Python code, not a notebook or a script. In this pack's Stack 3, Python owns the AI orchestration and evaluation tier: the agents, the document generators, the eval harness. Common triggers: "structure this Python service," "should this be async," "how do we type the model boundary," "wire config and secrets," "the Python service is a pile of dicts."

This is the implementation companion to `ai-application-architecture`. That skill designs the AI tier — archetype, model, retrieval, serving topology; this skill builds the Python that runs it. Use them together.

Do **not** use this skill for: designing the AI application (`ai-application-architecture`); building Go services or MCP servers (`mcp-go-server-building`); the test suite — pytest discipline and the test pyramid (`test-engineering`); designing the evaluation harness (`ai-evaluation-harness` — this skill *builds* eval code, that skill designs the harness).

## The critical decision rule — type every boundary; the model is the only thing allowed to be unpredictable

Python's dynamism is a liberty you spend, not a default you keep. In an AI service the model's output is already a non-deterministic, only-loosely-shaped surface — and the code around it must be the opposite: typed, validated, statically checkable. So the rule: **type every boundary** — function signatures, configuration, the model's input and output, every cross-module call — and validate external input with Pydantic at the edge. A bare `dict` or an `Any` passed between modules is how a Python AI service rots into something only its author can safely change, because the one tool that catches the mistake — the type checker — has been switched off by default. The model is allowed to be the unpredictable part of the system. The code is not.

## Project structure

A service is a package, not a folder of scripts: a `src/` layout, single-responsibility modules, a thin composition root that wires dependencies, and one dependency manager with a committed lockfile (`uv` is the strong current default; `poetry` or `pip-tools` are fine — pick one and lock it). A notebook is a fine place to prototype and never a place to ship from — productionizing means re-housing the logic in this structure, not packaging the `.ipynb`. Depth in `references/project-structure-and-packaging.md`.

## Typed boundaries

Type hints on every signature; Pydantic models for configuration, external input, and the model's structured input and output; `mypy` or `pyright` running in CI as a gate, not a suggestion. Validate at the edge — parse untrusted input into a Pydantic model once, at the boundary — and trust the typed object within. `references/typing-and-pydantic.md`.

## Async

The AI tier is I/O-bound — model calls, retrieval, tool calls — which is the case `async`/`await` is for. But async is a discipline, not a default: a single blocking call on the event loop stalls the whole service, and a CPU-bound or genuinely simple service is better off synchronous. Decide deliberately. `references/async-and-concurrency.md`.

## Orchestration code

Implement the framework `ai-application-architecture` chose — **Pydantic AI** by default, **LangGraph** when durable, resumable graphs are needed. Keep the orchestration structure clean: typed agents or graph nodes, a single model-client wrapper that owns retries, tracing, and cost, and orchestration logic kept out of tool code. `references/orchestration-code.md`.

## Service-runtime concerns

Configuration through a typed Pydantic Settings object validated at startup; secrets from Azure Key Vault via managed identity, never environment variables or code; structured JSON logging; OpenTelemetry with the GenAI semantic conventions; graceful shutdown; the Container Apps runtime shape. `references/service-runtime-concerns.md`.

## Testing

Testing a Python service — pytest, async tests, mocking the model boundary — is real work, and its discipline (the test pyramid, what to test, characterization tests) is owned by `test-engineering`. Build the service test-first against that skill; this skill's job is the service code, not the test strategy.

## Errors — fail typed, fail loud, fail at the edge

A Python AI service has several failure surfaces: the model call (timeout, throttle, malformed output), retrieval (a miss, the store down), input validation, a tool error. Handle them with typed exceptions, not a bare `except` that swallows everything. Let input validation fail *at the edge* — a Pydantic parse error at the boundary with a clear message, not an `AttributeError` deep in the orchestration. Convert a model or retrieval failure into a typed error the caller can branch on, not a generic crash. And never swallow an exception into a `None` that surfaces, mislabelled, three calls later — in a dynamic language that is the single hardest bug to trace. Fail typed, fail loud, fail close to the cause.

## Worked example — brownfield: a RAG prototype script becomes a service

Setup: a working RAG prototype is a single Python script — top-to-bottom, model keys read from `os.environ`, data passed around as dicts, no package structure, no types.

Decision walk: (1) Create a `src/` package and a `pyproject.toml` with a locked dependency set; the script's logic moves into single-responsibility modules. (2) Define Pydantic models for the config, the request, and the model's structured output; replace the loose dicts. (3) Move orchestration into a Pydantic AI service behind a clean interface (the archetype decision is `ai-application-architecture`'s — here it is RAG, not an agent). (4) Make the model and retrieval calls `async`; add a bounded-concurrency limit for any fan-out. (5) Replace the `os.environ` reads with one Pydantic Settings object; move the model key to Key Vault via managed identity. (6) Add structured logging and an OpenTelemetry trace. (7) Containerize for Container Apps. The script is now a service.

The wrong move is to `pip freeze` the script's environment and wrap the script in a web handler — that ships the untyped boundaries, the leaked key, and the unstructured everything as production code.

## Anti-pattern — the dict-typed service

**Bad:** data moves between modules as bare `dict` or `Any`; functions take `**kwargs`; configuration is `os.environ` reads scattered through the code. **Why it fails:** every boundary is a runtime surprise the type checker cannot catch; an AI service that is already non-deterministic at the model becomes unpredictable at every call site too. **Detection signal:** no Pydantic models; `mypy`/`pyright` not run or not gating CI; `dict[str, Any]` in signatures; `os.environ` accessed outside one config module. **Fix:** Pydantic models at every boundary, a type checker gating CI, one typed settings object — the typed-boundary rule above.

## Verification questions

1. Is the service a `src/`-layout package with single-responsibility modules and a committed dependency lockfile — not a script or a notebook?
2. Does every boundary have a type — signatures, config, the model's input and output — with Pydantic models for external input?
3. Is `mypy` or `pyright` running in CI as a gate?
4. Was async vs sync decided deliberately, and is the event loop free of blocking calls?
5. Is orchestration built on the framework `ai-application-architecture` chose, with one model-client wrapper owning retries, tracing, and cost?
6. Is configuration one typed Settings object, and are secrets read from Key Vault via managed identity — not environment variables?
7. Does the service emit structured logs and an OpenTelemetry trace, and shut down gracefully?

## What to read next

Tier-2 references: `references/project-structure-and-packaging.md` · `references/typing-and-pydantic.md` · `references/async-and-concurrency.md` · `references/orchestration-code.md` · `references/service-runtime-concerns.md`.

Related skills: `ai-application-architecture` (designs the AI tier this skill builds — read it first) · `test-engineering` (the test suite for the service) · `mcp-go-server-building` (the Go tool tier the Python service calls) · `azure-microservices-observability` (the telemetry backbone) · `azure-microservices-security` (Key Vault, managed identity).
