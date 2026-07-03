---
name: ai-application-architecture
description: Designs the architecture of LLM/AI application features on Azure — archetype selection (single-shot, RAG, agentic loop, workflow, conversational, batch), model and inference layer, retrieval design, orchestration-framework choice, evaluation, safety, and the Python/Go/Next.js serving topology (Stack 3). Use when designing or reviewing an AI/LLM feature, choosing Foundry agents vs a self-built orchestrator, picking an orchestration framework, designing a RAG pipeline, or planning how the AI tier is served and evaluated. Do not use for building MCP servers (mcp-go-server-building), threat-modeling one (mcp-go-threat-modeling), or data-store/vector-index internals (azure-data-tier-design).
version: 1.1.0
last_updated: 2026-05-30
---

# AI Application Architecture

## When to use

Trigger this skill when designing, reviewing, or productionizing an application feature whose behavior depends on a large language model or other generative model — a RAG feature, an assistant or agent, an extraction or classification step, a content-generation flow. Common triggers: "should this be RAG or fine-tuning," "hosted Foundry agent or our own orchestrator," "which orchestration framework," "how do we evaluate this before it ships," "how is the AI tier served and traced."

Do **not** use this skill for: building or extending MCP servers (`mcp-go-server-building`); threat-modeling an MCP server (`mcp-go-threat-modeling`); operational data-store or vector-index internals — engine choice, partition keys, query tuning (`azure-data-tier-design`, which this skill *calls*, not replaces); or producing a single-agent blueprint as an advisory deliverable (the AI Agent Blueprint Advisor persona in `instruction-os`).

This skill assumes **Stack 3**: Python owns AI orchestration and evaluation, Go owns the gateway and tool/infrastructure tier, Next.js owns the BFF. Backend Node beyond the Next.js BFF is off-stack — see `.claude/CLAUDE.md` anti-pattern 1, AI application tier exception.

## The critical decision rule — earn the LLM before you design around it

Two gates, in order. **First: does the feature need a generative model at all?** If a deterministic rule, a trained classifier, or a SQL query satisfies the requirement, use that — an LLM buys latency, token cost, and a non-deterministic failure surface you then have to evaluate and guard. **Second: if it does need a model, the architecture follows the *archetype*, not the framework.** Name the archetype first; the framework, the model, and the topology are all downstream of it. Teams that pick "an agent" or a named framework before naming the archetype build the wrong thing well.

## Application archetypes

| Archetype | Use when | Routes to |
|---|---|---|
| Single-shot completion / extraction / classification | One model call, no retrieval, no loop | `patterns/single-shot.md` |
| Retrieval-augmented generation (RAG) | Answer must be grounded in private or changing data | `patterns/rag.md` + `azure-data-tier-design` |
| Agentic tool-calling loop | Model must choose and invoke tools to reach a goal | `patterns/agentic-loop.md` |
| Deterministic multi-step workflow | Steps are known; the model fills specific nodes | `patterns/llm-workflow.md` |
| Conversational with memory | Multi-turn, state carried across turns | `patterns/conversational.md` |
| Batch / async LLM processing | High volume, latency-tolerant, offline | `patterns/batch-llm.md` |

Pick the simplest archetype that meets the requirement. RAG is not the default; an agentic loop is rarely the default.

## Build vs buy — start on Foundry Agent Service

Default to **Microsoft Foundry Agent Service** (GA — hosted agents on the OpenAI Responses API, managed memory, GA evaluations with continuous monitoring into Azure Monitor). It removes orchestration-runtime, memory, and monitoring code you would otherwise own and operate. **Graduate to a self-built Python orchestrator** when you outgrow it: orchestration logic the hosted runtime cannot express, the need to be portable off the Responses API, or per-step control the managed loop will not give. Do not self-build on day one to avoid a managed dependency you may never outgrow.

## Orchestration framework — choose Pydantic AI or LangGraph (do not skip this)

When the design calls for a self-built orchestrator, the framework is an **explicit decision, not a default to drift past**. State the choice and the reason in the design.

- **Default: Pydantic AI** — type-safe, FastAPI-style ergonomics, structured outputs; matches the pack's typed, disciplined house style.
- **Choose LangGraph** when you need explicit stateful graph control: durable execution, branching and retry as first-class, long-running or human-in-the-loop loops.
- LlamaIndex when the workload is RAG-dominant; Microsoft Agent Framework for Azure-native multi-agent orchestration with Python/Java/C# parity.

Go's role is the gateway and tool tier — the official MCP Go SDK — not orchestration. Do not reach for a Go agent framework to keep orchestration in Go; that fights the stack.

## Serving topology — three tiers, two seams

Three tiers, three languages. The **Next.js BFF** handles rendering, the Entra ID / MSAL auth session, and re-streaming model tokens to the browser; it never calls a model directly. The **Go gateway and tool tier** is the API Management edge, the MCP tool layer, and high-throughput non-AI services. The **Python orchestration and evaluation tier** holds archetype logic, retrieval, model calls, and evals.

The two seams are the part no public single-language reference architecture gives you, so design them explicitly: cross-tier contracts are gRPC or REST with **generated typed clients** (OpenAPI / protobuf codegen) — never untyped JSON across a boundary; token streams flow browser ← SSE ← Next.js ← Python, with the BFF re-streaming, not originating; one OpenTelemetry trace (GenAI semantic conventions) spans all three tiers. No tier owns logic that belongs to another.

## Evaluation is not optional

An AI feature without an evaluation suite is a prototype, not a product. Use two tools: a lightweight framework for CI gating — **DeepEval or Ragas** — wired into the pipeline beside the pack's existing PR-review gates; and **Foundry Evaluations + Continuous Monitoring** for production scoring into Azure Monitor. Define the golden dataset and the pass threshold before the feature is built, not after the first incident.

## Reference architecture on Azure

Azure Container Apps hosts all three tiers; Foundry / Azure OpenAI sits behind API Management; Azure AI Search holds vectors; Cosmos DB or Postgres holds conversation and application state; Key Vault and managed identity hold secrets; Entra ID is the identity plane. Model hosting stays standard (pay-as-you-go) until sustained volume justifies provisioned throughput (PTU).

## Worked example — brownfield: a RAG prototype trapped in a notebook

Setup: a working RAG prototype lives in a Jupyter notebook — LangChain, an ad-hoc local vector index, model keys in environment variables, no evaluation. Leadership wants it shipped.

Decision walk: (1) Name the archetype — it is RAG, not an agent; do not add a tool loop it does not need. (2) Re-platform retrieval onto Azure AI Search with hybrid + semantic ranking; the notebook's local index does not survive — see `azure-data-tier-design`. (3) Move orchestration into a Python service; choose **Pydantic AI** here (no durable graph is needed) over LangGraph, and record the choice and reason. (4) Put the service behind the Go gateway; the Next.js BFF re-streams answers to the browser. (5) Before shipping: build a golden Q&A set, wire Ragas faithfulness and context-recall checks into CI, set a pass threshold. (6) Secrets move to Key Vault via managed identity; the keys leave the environment.

The wrong move is to "deploy the notebook" — containerize it untouched. That ships the missing evals, the unmanaged index, and the leaked keys as production defects.

## Anti-pattern — notebook-to-production

**Bad:** a prototype is promoted by wrapping the notebook or script in a container and exposing it. **Why it fails:** the prototype never had evaluation, retrieval durability, secret management, streaming, or a typed seam — promotion ships all of those gaps at once. **Detection signal:** no golden dataset in the repo; the retrieval index is created by an inline script; model keys sit in environment variables; there is no eval stage in CI. **Fix:** treat productionization as a re-architecture against this skill, not a deployment step.

## Verification questions

1. Was the LLM gate applied — is a generative model actually required, or would a deterministic approach do?
2. Is the application archetype named explicitly, and is it the simplest one that meets the requirement?
3. Was build-vs-buy decided — Foundry Agent Service vs self-built — with a stated reason?
4. If self-built: is the orchestration framework chosen explicitly (Pydantic AI or LangGraph), with the reason recorded?
5. Are the two cross-tier seams typed (generated clients), and does one trace span all three tiers?
6. Does the Next.js BFF re-stream rather than originate model calls, and are all secrets in Key Vault?
7. Is there a golden dataset and a CI eval gate with a pass threshold defined before build?

## What to read next

Related skills: `mcp-go-server-building` (the Go tool layer this skill routes tools through) · `azure-data-tier-design` (vector store and state-engine choice) · `azure-microservices-security` (Entra, Key Vault, network) · `azure-microservices-observability` (the trace and telemetry backbone) · `azure-microservices-cost-review` (token and inference cost). Persona: the AI Agent Blueprint Advisor in `instruction-os` for single-agent advisory blueprints.

Tier-2 depth lives in `references/`: `model-and-inference-layer.md`, `retrieval-design.md`, `orchestration-frameworks.md`, `serving-topology.md`, `evaluation.md`, `safety.md` — plus the six archetype cards under `references/patterns/` (`single-shot`, `rag`, `agentic-loop`, `llm-workflow`, `conversational`, `batch-llm`) that the archetype table routes to.
