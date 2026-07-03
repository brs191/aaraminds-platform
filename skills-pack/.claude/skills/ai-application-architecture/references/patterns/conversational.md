# Pattern: Conversational with Memory

## Problem

A feature is multi-turn — the user and the system exchange messages, and each turn depends on what came before. The state carried across turns — the conversation, plus any durable facts about the user or task — is now part of the architecture. Done naively, the context window grows until it overflows or cost balloons, and "memory" becomes an unbounded transcript.

## Use When

- The interaction is genuinely multi-turn — clarification, refinement, follow-up.
- State must carry across turns: the dialogue, and possibly durable user or task facts.
- The user expects the system to "remember" within, and possibly across, sessions.

## Avoid When

- The interaction is one request and one response → `single-shot.md` or `rag.md`. A stateless Q&A endpoint is not conversational; do not add memory it does not need.
- "Conversational" is a UI choice over a fundamentally stateless task — keep the backend stateless and let the BFF hold display history.

## Shape

The Python tier owns the turn loop. Short-term memory is the windowed message history; long-term memory is durable facts written to Cosmos DB or Postgres (`azure-data-tier-design`) — not the raw transcript. Foundry Agent Service provides managed conversation memory and is the default; a self-built version manages its own thread store. The Next.js BFF holds the auth session and re-streams tokens — it does not own conversation state. Decide the memory strategy explicitly: windowing, summarization of older turns, or retrieval over past turns.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Continuity | The system feels coherent across a session |
| State management | A thread store, a memory strategy, and an eviction policy are now yours to own |
| Context growth | Every turn adds tokens — cost and latency rise across a session |
| Evaluation | Multi-turn is harder to score than single-turn — evaluate dialogues, not isolated turns |

## Common Failure Modes

- **Context overflow** — the window fills mid-conversation; the API truncates silently and the model "forgets" the start. Detection: token-count the assembled context each turn. Prevention: explicit windowing plus summarization of evicted turns.
- **Memory poisoning** — a wrong or adversarial fact enters long-term memory and persists across turns and sessions. Detection: long-term writes are typed and reviewable; test that a bad turn does not corrupt durable memory. Prevention: do not auto-promote raw turns to durable facts.
- **Persona drift** — tone or behaviour degrades over a long session. Detection: eval on long synthetic dialogues. Prevention: re-anchor the system prompt each turn; do not rely on it surviving in-context.
- **Cross-session leakage** — one user's memory surfaces for another. Detection: memory keys carry the user / tenant id; test isolation explicitly. Prevention: scope every memory read and write by identity.

## Decision Signals

Use the conversational archetype only when state genuinely crosses turns. A chat UI over a stateless task does not need it — keep the backend stateless.

## Worked signal — Code Intelligence Factory

v1 of the CIF is not conversational — HLD/LLD generation is a workflow, and human review happens at the Trust Gate, not in a chat. A conversational "interrogate the codebase" feature is plausible later; if built, its long-term memory must be scoped per repo and per user, and must never let one reviewer's session mutate the shared knowledge graph — the graph is the system of record, conversation memory is not.

## References

- `azure-data-tier-design` — conversation and durable-memory state stores
- `../orchestration-frameworks.md` — managed vs self-built memory
- `../evaluation.md` — multi-turn dialogue evaluation
- `../safety.md` — memory poisoning and cross-session isolation
