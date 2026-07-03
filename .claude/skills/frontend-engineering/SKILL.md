---
name: frontend-engineering
description: Builds the React and Next.js frontend — component design and state, App Router architecture and rendering strategy, the Backend-for-Frontend (BFF) tier from ai-application-architecture's serving topology, streaming LLM tokens from the BFF to the browser, and TypeScript discipline at the API boundary. Use when building or reviewing a React UI, structuring a Next.js app, deciding server vs client components, building the BFF route layer, streaming model output, or hardening frontend TypeScript. Do not use for AI archetype/serving design (use ai-application-architecture), the Python orchestration tier (use python-service-engineering), or the Go/MCP tier (use mcp-go-server-building).
version: 1.0.1
last_updated: 2026-05-30
---

# Frontend Engineering

## When to use

Trigger this skill when building the user-facing tier — a React UI, a Next.js app, the BFF route layer behind it. In an AI product the frontend is often a *review surface*: the place a human inspects, accepts, or corrects what the system produced. The CIF's Trust Gate — where an engineer reviews a reconstructed HLD or BRD before it is trusted — is exactly this. Common triggers: "build the review UI," "structure this Next.js app," "should this be a server or client component," "stream the model's output to the page," "the frontend types are a mess."

This is the frontend companion to `ai-application-architecture`. That skill designs the serving topology — including the Next.js BFF; this skill builds the React and Next.js code that runs in and behind the browser. Use them together.

Do **not** use this skill for: designing the AI application, model, retrieval, or serving topology (`ai-application-architecture`); the Python orchestration tier the BFF calls (`python-service-engineering`); the Go gateway or MCP servers (`mcp-go-server-building`); the test suite (`test-engineering`).

## The critical decision rule — render every state, and never trust the browser

Two failures define bad frontend code, and one rule covers both. First: **every asynchronous surface has more than one state.** A fetch or a stream is, at minimum, loading, error, empty, and success — and a token stream adds a fifth, streaming. The default failure is to code only the success state, so the UI shows a blank or a crash the moment the network is slow, the result is empty, or the call fails. Enumerate the states first, render each deliberately. Second: **the browser is an untrusted client.** It never holds a secret, never calls the model or the knowledge graph directly. It talks only to the BFF, which owns authentication, secret material, and re-streaming. A model API key in client code is shipped to every visitor. Render every state; trust nothing in the browser.

## React component design

Components composed small, with a clear split between state that is *owned* and state that is *derived* — derived values are computed during render, never stored and synced. Server Components by default, Client Components (`"use client"`) only where interactivity or browser APIs require it. Reach for a state manager only when prop-drilling genuinely hurts; URL and server state are not React state. `references/react-component-design.md`.

## Next.js app architecture

The App Router as the structure: the route tree, layouts, server components fetching data on the server, the rendering strategy chosen per route (static, dynamic, streamed) rather than defaulted. Route Handlers are where the BFF lives. `references/nextjs-app-architecture.md`.

## The BFF tier

The Backend-for-Frontend is the server-side tier that exists for this frontend: it authenticates the user, holds the secrets, calls the Python orchestration and Go gateway, and shapes their responses for the UI. It is the trust boundary `ai-application-architecture`'s serving topology places between the browser and the system. `references/bff-tier.md`.

## Streaming model output

An LLM response arrives token by token over seconds; a UI that blocks on the whole response feels broken. The BFF consumes the model's stream and re-streams it to the browser; the browser renders tokens as they arrive. This is the core interaction of a review surface like the Trust Gate. `references/streaming-llm-responses.md`.

## TypeScript discipline

`strict` on, `any` treated as a defect. The API boundary — what the BFF returns — is a typed contract, validated at the edge with a runtime schema (Zod), not asserted. UI states modeled as a discriminated union so the compiler forces every state to be handled. `references/typescript-discipline.md`.

## Testing the frontend

Frontend tests — component tests with Testing Library, BFF route tests, end-to-end with Playwright — follow the same discipline as the rest of the stack and are owned by `test-engineering`: test observable behavior (what the user sees) at a stable seam, not component internals. Build the UI test-first against that skill.

## Accessibility and performance are not later

A review surface is a working tool; it must be keyboard-navigable, screen-reader-correct, and fast. Semantic HTML and correct ARIA are part of building the component, not a pass afterward. Performance has a budget: server-render what can be server-rendered, keep the client bundle small, lazy-load what is heavy, and measure Core Web Vitals rather than guessing. A frontend that is slow or unusable with a keyboard is unfinished, not done-but-rough.

## Worked example — brownfield: a review surface that calls the model from the browser

Setup: a prototype of the CIF Trust Gate review UI is a single-page React app. It calls the LLM API directly from the browser with the API key in client code; it renders the reconstructed document only after the full response returns; it shows a blank screen while waiting and a stack trace on failure.

Decision walk: (1) The key in client code is shipped to every visitor — move the model call behind a Next.js Route Handler (the BFF); the browser now calls the BFF, the BFF holds the key and authenticates the user. (2) The BFF consumes the model's token stream and re-streams it to the browser. (3) The review component renders all states — loading, streaming (tokens appearing live), the empty case, an error with a retry, and the final reviewable document. (4) Model the state as a discriminated union so the compiler refuses a missing branch. (5) Type the BFF response as a contract and validate it with Zod at the boundary. (6) Make the document view server-rendered where it can be, the interactive review controls a client component. (7) Keyboard and screen-reader pass on the review controls.

The wrong move is to "just hide the API key with an environment variable" in the client bundle — a client-side env var is still shipped to the browser; only a server tier actually keeps the secret.

## Anti-pattern — the happy-path client

**Bad:** components that render only the success state; async data with no loading, error, or empty branch; the model or an external API called directly from the browser with credentials in client code. **Why it fails:** the UI breaks visibly the first time the network is slow, the result is empty, or a call fails — and any secret in the client bundle is public. **Detection signal:** no loading or error UI; `fetch` to a model or third-party API in client code; an API key in a `NEXT_PUBLIC_` variable or client component; UI state as scattered booleans (`isLoading`, `isError`) instead of one union. **Fix:** enumerate states as a discriminated union and render each; put every privileged call behind the BFF — the decision rule above.

## Verification questions

1. Does every asynchronous surface render loading, error, and empty states — not just success — with streaming handled where the data streams?
2. Is every privileged call (model, graph, secrets) behind the BFF, with nothing sensitive in the client bundle?
3. Are Server Components the default, with `"use client"` used only where interactivity genuinely requires it?
4. Is the rendering strategy chosen per route, not defaulted?
5. Is `strict` TypeScript on, `any` absent, and the BFF response a typed contract validated at the edge with a runtime schema?
6. Is UI state modeled as a discriminated union so the compiler forces every state to be handled?
7. Is the surface keyboard-navigable and screen-reader-correct, with a measured performance budget?
8. Is owned state distinguished from derived state — derived values computed during render, never stored and synced?
9. Are frontend tests written against observable behavior (what the user sees) rather than component internals, per `test-engineering`?

## What to read next

Tier-2 references: `references/react-component-design.md` · `references/nextjs-app-architecture.md` · `references/bff-tier.md` · `references/streaming-llm-responses.md` · `references/typescript-discipline.md`.

Related skills: `ai-application-architecture` (designs the serving topology this skill's BFF sits in — read it first) · `python-service-engineering` (the orchestration tier the BFF calls) · `mcp-go-server-building` (the Go gateway behind the BFF) · `test-engineering` (the frontend test suite) · `azure-microservices-security` (auth and secret handling the BFF enforces).
