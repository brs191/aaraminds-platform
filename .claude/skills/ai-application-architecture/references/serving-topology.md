# Serving Topology

This reference covers how an AI feature is served: the three tiers, the language each is written in, the two seams between them, the single trace that spans them, and how the whole thing is deployed and secured on Azure. It is the topology behind the SKILL.md's "three tiers, two seams" — expanded into the design detail no single-language public reference architecture gives you.

## Three tiers, three languages, two seams

An AI feature in this stack is three tiers, each a different language, each owning one concern:

- **Next.js BFF** — rendering, the user's auth session, and re-streaming model tokens to the browser.
- **Go gateway and tool tier** — the API Management edge, the MCP tool layer, and high-throughput non-AI services.
- **Python orchestration and evaluation tier** — archetype logic, retrieval, model calls, and evals.

The tiers are deliberate: each language is doing what it is best at, and no tier owns logic that belongs to another. The risk is not the tiers — it is the **two seams** between them. Seams are where untyped JSON, double-buffered streams, and broken traces creep in, and no public single-language reference architecture designs them for you. Design both seams explicitly.

## The Python orchestration and evaluation tier

Python owns the AI logic: archetype implementation (the patterns in `patterns/`), retrieval (`retrieval-design.md`), the model client (`model-and-inference-layer.md`), and the evaluation suite (`evaluation.md`). It is where Pydantic AI or LangGraph runs (`orchestration-frameworks.md`). It does not face the browser and it does not face the public internet — it sits behind the Go gateway. This is the one tier where Python is on-stack for a production path; the `.claude/CLAUDE.md` anti-pattern 1 exception is scoped to exactly this tier and no wider.

## The Go gateway and tool tier

Go owns the edge and the tools. The gateway is the API Management-fronted boundary: authentication enforcement, rate limiting, request routing, and the public contract. The tool tier is the MCP tool layer (`mcp-go-server-building`) — the typed, governed tools an agentic loop or workflow invokes. High-throughput non-AI services that have no reason to be in Python also live here. Go faces the public internet; Python does not.

## The Next.js BFF tier

Next.js owns the browser-facing concern: server-side rendering, the Entra ID / MSAL auth session, and re-streaming model tokens to the client. The single hard rule for this tier: **the BFF never calls a model directly.** It calls the Go gateway, which routes to Python. A BFF that calls the model API itself has skipped the gateway's auth and rate limiting, has no access to the orchestration tier's retrieval and evals, and puts model credentials in the most exposed tier. Backend Node beyond this BFF is off-stack.

## Seam 1 — typed cross-tier contracts

Every call that crosses a tier boundary — BFF to Go, Go to Python — uses a **typed contract with a generated client**: gRPC with protobuf, or REST with an OpenAPI schema, and the client generated from that schema. Never hand-written, never untyped JSON across a boundary.

The reason is that the model layer is already a non-deterministic surface; the *plumbing* around it must be the opposite — boring and statically checked. An untyped JSON boundary turns a renamed field into a runtime failure deep in another tier, in another language, discovered in production. A generated typed client turns the same mistake into a compile error in CI. The model is allowed to be the unpredictable part; nothing else is.

## Seam 2 — the token stream

Model output is streamed token-by-token, and the stream crosses both seams: it originates in Python (the model call), passes through Go, and reaches the browser via the Next.js BFF. The rule: the stream flows **browser ← SSE ← Next.js ← Python**, and the BFF **re-streams** — it forwards tokens as they arrive. It does not buffer the full completion and then send it; buffering destroys the streaming UX (time-to-first-token, `model-and-inference-layer.md`) the architecture exists to deliver. Each tier forwards chunks; no tier collects the whole response before passing it on. The BFF re-streams; it never originates the stream.

## One trace across three tiers

A request touches three languages, three tiers, and at least one model call. It must be **one OpenTelemetry trace**, not three disconnected ones. Propagate trace context across both seams (it rides the typed contract), and instrument the model calls with the **OpenTelemetry GenAI semantic conventions** so model latency, token counts, and cost appear as spans in the same trace as the HTTP and tool spans. A broken trace at a seam means an AI feature you cannot debug — when a response is slow or wrong, you cannot tell which tier, which retrieval, or which model call is responsible. The telemetry backbone and span conventions are `azure-microservices-observability`.

## Deployment: Azure Container Apps

All three tiers deploy as containers on **Azure Container Apps** — independent apps, independently scaled, in one environment. Their scaling profiles differ and should be set independently: the Python tier scales on model-call concurrency and is the expensive tier to over-scale; the Go gateway scales on request rate; the BFF scales on user sessions. Batch model work (`patterns/batch-llm.md`) runs as Container Apps **jobs**, not on the always-on serving apps. Foundry / Azure OpenAI sits behind API Management; Azure AI Search holds vectors; Cosmos DB or Postgres holds conversation and application state.

## Identity and secrets

Identity is **Entra ID** end to end. The user authenticates to the BFF (MSAL); service-to-service calls across the seams use **managed identity**, not shared keys. All secrets — model API keys, search keys, connection strings — live in **Azure Key Vault** and are read via managed identity; no secret sits in an environment variable or a container image. This is the same identity and secret discipline as `azure-microservices-security`, applied to the AI tiers; the AI feature gets no exemption.

## Verification questions

1. Are there exactly three tiers — Next.js BFF, Go gateway/tool, Python orchestration — each owning only its concern?
2. Does the Next.js BFF re-stream model tokens rather than call a model directly or buffer the full completion?
3. Are both cross-tier seams typed contracts with generated clients — no untyped JSON across a boundary?
4. Does one OpenTelemetry trace span all three tiers, with model calls instrumented to the GenAI semantic conventions?
5. Are the three tiers independently scaled, and does batch model work run as Container Apps jobs?
6. Is identity Entra ID end to end, service-to-service on managed identity, and every secret in Key Vault?

## What to read next

- `orchestration-frameworks.md` — what runs inside the Python tier
- `model-and-inference-layer.md` — the model client the Python tier calls
- `mcp-go-server-building` — the Go tool tier
- `azure-microservices-observability` — the trace and telemetry backbone
- `azure-microservices-security` — the Entra, Key Vault, and network discipline
