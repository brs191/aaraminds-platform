---
name: mcp-go-guardrails-and-safety
description: Embeds layered runtime and CI-time guardrails into Go MCP servers — input validation, argument sanitization, output redaction, prompt-injection defense, audit logging, rate limiting, tool authorization — plus a CI eval suite (promptfoo) and OpenTelemetry observability (Langfuse default, Phoenix alternative). Use when designing safety into a new MCP server, retrofitting guardrails to an existing one, or wiring the CI safety gate. Do not use for design-time STRIDE threat modeling (use mcp-go-threat-modeling) or the pre-production readiness checklist (use mcp-go-production-review) — this skill is the implementation layer those skills depend on.
version: 1.1.0
last_updated: 2026-05-30
---

# MCP-Go Guardrails and Safety

## When to use

Trigger this skill when the question is about **implementing** safety into a Go MCP server: middleware chains, input validation beyond JSON schema, prompt-injection defense, secret/PII redaction, rate limiting, tool authorization, audit logging, the CI eval suite, and the OpenTelemetry wiring to an LLM-aware observability platform. Common triggers: "this MCP server is going to production, what's the safety layer," "retrofit guardrails to an existing server," "add prompt-injection defense to a tool that takes free-text input," "wire the server's traces to Langfuse."

Do **not** use for: design-time STRIDE threat modeling (`mcp-go-threat-modeling`); pre-production readiness checklist (`mcp-go-production-review`); the broader Azure microservices security surface — Entra, Private Link, Key Vault — that's the platform layer (`azure-microservices-security`).

## The critical decision rule — guardrails are layers, pick one tool per layer

Two failure modes dominate when teams add "safety" to LLM-adjacent systems:

1. **Conflating eval frameworks with runtime guardrails.** Ragas, DeepEval, Braintrust, Langfuse, Phoenix, promptfoo — these *measure quality and surface traces*. None of them *stop* a malicious input or *redact* a sensitive output at runtime. Treating them as guardrails leaves the actual runtime surface unprotected. Use eval tools for what they're for; use real guardrails (in-process validation + classifier + middleware) for runtime.
2. **Stacking five tools per layer.** Picking promptfoo *and* Braintrust *and* DeepEval as "the eval setup" produces three sets of YAML/code/dashboards that drift apart. Pick one per layer; commit; move on.

The pack default for a Go MCP server: native Go runtime guardrails + Azure AI Content Safety for prompt-injection classification + promptfoo for CI evals + OpenTelemetry → Langfuse for observability. Substitute only with a written reason.

## The four-layer model

| Layer | Where it runs | Tool (default) | What it does |
|---|---|---|---|
| **Runtime guardrails** | In-process Go middleware | Native Go, no external dep | Validates input, sanitizes args, redacts output, rate-limits, audit-logs, caps resources |
| **Prompt-injection defense** | In-process + hosted classifier | Local heuristics + Azure AI Content Safety **Prompt Shields** | Defense-in-depth *signal* — detects injection patterns in input/output. Not the primary control: structured output + client framing + least privilege are (see `mcp-go-threat-modeling`). |
| **CI-time eval suite** | CI pipeline | **promptfoo** (YAML, language-agnostic) | Regression suite: tool catalog, golden outputs, injection probes |
| **Observability** | Out-of-process via OTel | **OpenTelemetry → Langfuse** (or Phoenix) | Production traces: tool calls, latency, outcomes, error patterns |

Optional fifth layer — advanced evals (hallucination, faithfulness) — runs out-of-process as a Python harness using DeepEval, driving the MCP server as a subprocess. Skip unless you have a named requirement.

## The work — sequence for embedding guardrails

1. **Choose the middleware chain shape** before writing tool handlers. Every tool handler is wrapped by the same chain: validate → authorize → rate-limit → audit-begin → execute → audit-end → redact-output. See `references/patterns/tool-handler-middleware-chain.md`.
2. **Per-tool input validation.** JSON schema from the MCP SDK is necessary but not sufficient — add per-tool custom validators (length caps, character allowlists, format regex). See `references/runtime-guardrails-go.md`.
3. **Argument sanitization at the boundary.** Tool args that flow into shell, SQL, file paths, or URLs get domain-specific sanitization before use. See `references/patterns/argument-sanitization.md`.
4. **Output redaction before return** (and before logging). Regex for secret patterns; structured redaction for known PII fields. See `references/secrets-and-pii-redaction.md`.
5. **Prompt-injection defense** for tools whose input is LLM-controlled or whose output goes back into LLM context. Run local heuristics + Azure AI Content Safety Prompt Shields as *non-blocking* signals (flag, log, gate) — never hard-reject legitimate data on a heuristic. The primary defense is structured output + client framing + least privilege; this is defense-in-depth. See `references/prompt-injection-defense.md` and `mcp-go-threat-modeling` for the hierarchy.
6. **Audit log** every tool call with structured slog fields. Ship to Log Analytics for SOC 2 / ISO 27001 evidence. See `references/patterns/structured-audit-log.md`.
7. **Rate limiting and resource caps** per tool. Token bucket via `golang.org/x/time/rate`; context timeout per call. See `references/runtime-guardrails-go.md`.
8. **Tool authorization** if the transport is HTTP. Entra ID with managed identity; per-tool auth declaration. Stdio transport has implicit auth from the parent process. See `references/tool-authorization.md`.
9. **CI eval gate.** promptfoo YAML with tool catalog regression + injection probe suite. Wire into GitHub Actions; block merges on failure. See `references/eval-and-ci.md`.
10. **Observability** before go-live. OTel from the Go server → Langfuse. Span attributes for tool name, outcome, duration. Never log raw args (PII concern). See `references/observability-with-otel.md`.

## Worked example — brownfield: retrofitting guardrails to an existing stdio MCP server

Setup: existing Go stdio MCP server with 8 tools. JSON schema validation only. No middleware chain, no audit log, no rate limit, no prompt-injection defense, no CI eval. Logging is unstructured `log.Printf`. Going to production for an internal team in 4 weeks.

Decision walk:

1. **Don't refactor all 8 tools at once.** Introduce the middleware chain *between* the MCP SDK's tool dispatcher and each handler. Wrap one tool first as the pattern; migrate the rest one per day.
2. **Audit log first.** Cheapest, highest value. Switch to `log/slog` with structured fields. Ship the audit log to Log Analytics via stderr → forwarder. SOC 2 evidence starts accumulating immediately.
3. **Then output redaction.** Run a `pg_stat_statements`-style audit of what tool outputs *could* contain. Add regex redactor for AWS keys / GitHub tokens / Azure SAS / JWTs. Verify with a fuzz test.
4. **Then prompt-injection on the two riskiest tools.** Identify the tools whose input is most LLM-controlled (free-text args) and whose output goes back into the agent's context. Add Prompt Shields call. Local heuristic for cheap pre-filter — block obvious patterns without the API hit.
5. **Then rate limiting.** Per-tool token bucket. Initial limit generous (1 req/sec/tool); tighten after observing real usage.
6. **promptfoo CI gate.** YAML with the existing tool catalog and a small injection probe suite. Wire into the GitHub Actions workflow as a required check. Block merges on regression.
7. **OTel last.** Once the in-server work is done, add OTel SDK, define spans for each tool call, ship to Langfuse running in your subscription's Container Apps. Now you have traces.

Order matters: audit log → redaction → injection → rate limit → CI → observability. Each step adds value without depending on the next.

## Anti-pattern — eval frameworks as guardrails

**Bad**: team adopts Ragas/DeepEval/Braintrust and treats them as "the safety story." Tool args still flow unchecked into shell commands. Output still contains secrets. No rate limit. But there's a beautiful eval dashboard.

**Why it fails**: eval frameworks measure quality *after the fact*. They don't stop a single malicious input or prevent a single secret leak at runtime. A prompt-injection that exfiltrates data via a tool call passes every eval and shows up as a successful tool call in the dashboard.

**Detection**: ask "what happens to a tool call with a malicious arg?" If the answer is "the eval suite would catch it eventually," the runtime layer is missing.

**Fix**: implement runtime guardrails (middleware chain, redaction, classifier, rate limit) *before* the eval suite. Evals are a regression net, not a fence.

## Verification questions

1. Does every tool handler go through the same middleware chain (no handlers bypass it)?
2. Is per-tool input validation present beyond the MCP SDK's JSON schema check?
3. Is output redaction running before the response goes back to the client and before it lands in any log?
4. For tools accepting free-text input: is prompt-injection defense (local heuristic + Prompt Shields) wired up?
5. Is every tool call audit-logged with structured fields, shipped to Log Analytics?
6. Is rate limiting active per tool with a defined budget?
7. For HTTP transport: is auth Entra-based with managed identity, not bearer tokens in environment?
8. Is the promptfoo CI gate a required check on the main-branch protection?
9. Are OTel traces flowing to Langfuse (or Phoenix) with tool spans carrying name, outcome, duration — and **never** raw args?
10. Has the safety surface been tested with a hostile prompt set, not just functional happy-path?

## What to read next

References: `references/runtime-guardrails-go.md` · `references/prompt-injection-defense.md` · `references/secrets-and-pii-redaction.md` · `references/tool-authorization.md` · `references/eval-and-ci.md` · `references/observability-with-otel.md`

Pattern cards: `references/patterns/tool-handler-middleware-chain.md` · `references/patterns/structured-audit-log.md` · `references/patterns/argument-sanitization.md`

Related skills: `mcp-go-server-building` (the build skill this skeleton attaches to) · `mcp-go-threat-modeling` (design-time STRIDE that names what to defend) · `mcp-go-production-review` (pre-prod checklist this skill's outputs satisfy) · `azure-microservices-security` (Entra, Private Link, Key Vault, platform layer) · `azure-microservices-observability` (broader observability surface beyond MCP)
