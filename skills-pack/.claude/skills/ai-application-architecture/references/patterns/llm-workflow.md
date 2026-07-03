# Pattern: Deterministic LLM Workflow

## Problem

A task is a known sequence of steps — a DAG you can draw before any run — where the model is needed at *specific* nodes (extract, summarize, classify, draft) but the control flow itself is fixed. An agentic loop would hand the model control it does not need and cannot be tested; a single call cannot express the sequence. The workflow keeps orchestration deterministic and uses the model only where judgement is required.

## Use When

- The steps and their order are known ahead of time.
- The model fills specific nodes; the edges between nodes are code, not model decisions.
- You want each model-touched node independently testable and independently eval-gated.
- Reproducibility and predictable cost matter.

## Avoid When

- The next step genuinely depends on the model's judgement about *what to do* → `agentic-loop.md`.
- It is one model call → `single-shot.md`.
- The flow is so branchy that encoding it as a DAG is harder than letting the model drive — rare; be honest about whether that is actually true.

## Shape

A Python orchestration-tier pipeline: each node is a function — a deterministic transform, or a single-shot model call with its own prompt template and output schema. Edges are ordinary control flow. Use a workflow library (LangGraph for durable, resumable runs; plain Python or Pydantic AI for short ones — `orchestration-frameworks.md`). Each model node is evaluated individually; an end-to-end eval sits on top. Persist intermediate state so a failed run resumes rather than restarts.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Predictability | Cost, latency, and execution path are all knowable ahead of a run |
| Testability | Every model node is eval-gated in isolation; the whole is gated end to end |
| Rigidity | A genuinely new branch needs a code change, not a prompt tweak |
| Cost | Much cheaper and more debuggable than an agentic loop for the same multi-step task |

## Common Failure Modes

- **Node-failure cascade** — one node's bad output silently corrupts every downstream node. Detection: validate each node's output schema before passing it on. Prevention: fail fast at the node; do not propagate.
- **Hidden agent** — the workflow keeps gaining model-decided branches until it is an unmanaged loop. Detection: count model-decided edges; if non-trivial, reassess. Prevention: if control flow is becoming dynamic, move to `agentic-loop.md` deliberately.
- **No resumability** — a five-node run fails at node four and restarts from node one, repaying cost and latency. Detection: re-run cost equals first-run cost. Prevention: checkpoint state per node.
- **Per-node eval gap** — only the end-to-end output is scored, so a regression cannot be localized. Detection: a node regresses but only the final eval moves. Prevention: evaluate each model node and the whole.

## Decision Signals

Use a workflow when you can draw the DAG before the run. This is the right default for multi-step model-assisted tasks — reach for `agentic-loop.md` only when the control flow itself must be dynamic.

## Worked signal — Code Intelligence Factory

This is the shape of the CIF's BA and QA agents in v1. HLD generation is a workflow: walk the graph for module boundaries → for each component, retrieve its neighbourhood → draft that section (single-shot) → assemble → link every section to its `DERIVED_FROM` graph nodes. The steps are fixed; the model fills only the section-drafting nodes. It is a document *generator*, exactly as the brief specifies — not an orchestrated agent.

## References

- Pattern: `agentic-loop.md` — when control flow must be dynamic instead
- Pattern: `single-shot.md` — the per-node model call
- `../orchestration-frameworks.md` — workflow libraries
- `../evaluation.md` — per-node and end-to-end scoring
