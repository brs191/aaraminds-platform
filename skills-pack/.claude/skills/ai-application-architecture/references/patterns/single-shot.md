# Pattern: Single-Shot Completion

## Problem

A feature needs one model call to transform a bounded input into a structured output — extract fields from a document, classify a ticket, summarize a passage, draft a paragraph. There is no private corpus to ground against and no decision the model must make about *what to do next*. Reaching for RAG or an agent here adds retrieval infrastructure and a control loop the task never needed.

## Use When

- The input fits comfortably in the context window and is self-contained.
- The output schema is known ahead of time — a JSON object, a label, a bounded span of text.
- The task is one transform, not a sequence whose later steps depend on earlier results.
- Any factual content in the output comes from the input itself, not from knowledge the model must already hold.

## Avoid When

- The answer must be grounded in private or changing data the prompt cannot carry → `rag.md`.
- The model must choose and invoke tools to reach the goal → `agentic-loop.md`.
- The work is a fixed sequence of model-assisted steps → `llm-workflow.md`.
- Output correctness depends on facts the model would have to recall rather than read — either ground it (RAG) or do not use a model.

## Shape

One Python orchestration-tier function: validate input, render a prompt from a versioned template, call the model with a structured-output schema enforced (a Pydantic model / JSON schema), validate the response against that schema, return. No retrieval, no loop, no memory. The Go gateway fronts it; the Next.js BFF re-streams if the output is user-facing text. This is the cheapest and most testable archetype — keep it that way.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Cost | Lowest — a single model call |
| Latency | One round trip |
| Testability | Highest — deterministic at temperature 0 with a pinned model |
| Grounding | None — the model sees only the input |
| Failure surface | Smallest, but silent hallucination on a factual claim is still possible |

## Common Failure Modes

- **Schema violation** — the model returns prose or malformed JSON. Detection: schema validation fails. Prevention: enforce structured output at the API level; reject-and-retry once, then fail loudly.
- **Silent hallucination** — the model invents a plausible value for a field absent from the input. Detection: an eval set with inputs that have missing fields, checking the model abstains. Prevention: instruct explicit "null when absent"; score abstention in the eval.
- **Prompt drift** — the template is edited without re-running the eval. Detection: the prompt template is version-controlled and CI runs the eval on every change. Prevention: treat the prompt as code.
- **Long-tail context overflow** — most inputs fit, the 99th-percentile input does not. Detection: token-count the input before the call. Prevention: measure the real input distribution; truncate or chunk explicitly rather than letting the API truncate silently.

## Decision Signals

Use single-shot when one input maps to one output, there is no corpus, and the schema is known. Do not use it when you find yourself wanting "and then look up…" (that is RAG) or "and then decide whether to…" (that is an agent) — name the real archetype instead.

## Worked signal — Code Intelligence Factory

The CIF's deterministic extractors (AST → graph nodes) are not model calls at all. But a step like "summarize this method's purpose into a `doc` property" is single-shot: bounded input (the method body), known output (one short string), no retrieval. The resulting node carries `provenance = inferred`, not `deterministic` — and the eval set must include methods whose purpose is genuinely unclear, to verify the model says so rather than inventing intent.

## References

- Skill: `../../SKILL.md` — archetype selection and the LLM gate
- `../evaluation.md` — abstention and schema-conformance scoring
- `../safety.md` — output validation
