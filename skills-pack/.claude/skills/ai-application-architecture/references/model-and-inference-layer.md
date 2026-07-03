# Model and Inference Layer

The model and inference layer is the part of an AI feature that turns a prompt into tokens: which model answers, how it is hosted, how the call is made resilient, and what it costs. This reference covers model selection, routing, hosting mode (standard vs provisioned), the Batch API, prompt caching, structured outputs, latency, resilience, and cost. Vector stores and state engines are out of scope — see `azure-data-tier-design`.

## The model is a dependency, not a given

Treat the model as a replaceable dependency behind an interface, exactly as you would a database. Pin a specific deployment, name it in the design, and make swapping it a one-line config change — not a code change scattered across the orchestration tier. Two consequences follow: every model call goes through one internal client wrapper (so retries, tracing, cost accounting, and fallback live in one place), and the evaluation suite (`evaluation.md`) is what tells you whether a model swap is safe. Without an eval, "upgrade the model" is a blind change to the most behaviour-defining dependency in the system.

## Model selection — match the model to the node, not the app

There is no single model for an application. There is a model for each node. Pick per node against three axes: task difficulty, latency budget, and cost sensitivity.

| Model tier | Use for | Avoid for |
|---|---|---|
| Frontier (largest reasoning models) | Hard reasoning, ambiguous extraction, agentic planning, anything where a wrong answer is expensive | High-volume simple nodes — it is slow and costly overkill |
| Mid-tier general models | The default workhorse — most RAG generation, summarization, structured extraction | Tasks needing deep multi-step reasoning |
| Small / fast models | Classification, routing decisions, simple extraction, high-volume nodes | Anything requiring real reasoning or broad world knowledge |
| Embedding models | Vectorizing text for retrieval — pick one and keep index and query on the *same* model | — |

Do not default to the largest model "to be safe." A frontier model on a node a small model handles correctly is pure latency and cost with no quality gain. Start each node at the smallest tier that passes its eval and move up only on a measured failure.

## Model routing

In a workflow (`patterns/llm-workflow.md`) or agentic loop (`patterns/agentic-loop.md`), different nodes warrant different models. Route explicitly: a cheap, fast model for the easy nodes, a frontier model for the one or two hard ones. A classification or routing node deciding *which path to take* should itself run on a small model — it is a cheap decision and putting a frontier model on it is the most common quiet cost leak in an AI pipeline. Record the per-node model choice in the design; it is a decision, not an accident.

## Hosting: standard vs provisioned throughput (PTU)

Azure OpenAI / Foundry models are consumed two ways, and the choice is a real cost-and-latency decision.

- **Standard / global-standard (pay-as-you-go)** — billed per token, no reserved capacity. Latency varies with shared-pool load; throughput is capped by an assigned quota and a busy pool returns `429`. Right for development, low or spiky volume, and any workload still finding its shape.
- **Provisioned Throughput (PTU)** — reserved capacity billed at a flat rate. Latency is predictable, throughput is guaranteed, cost is fixed regardless of token volume. Right for sustained, high-volume production traffic.

**Default to standard. Move to PTU when sustained volume makes the flat rate cheaper than metered tokens and predictable latency becomes a product requirement** — not before. The crossover is a utilization calculation: PTU only wins when you keep the reserved capacity busy. A half-idle PTU reservation is more expensive than pay-as-you-go. Revisit the decision with real traffic data on the cadence in `azure-microservices-cost-review`; do not buy PTU speculatively.

## The Batch API

For high-volume, latency-tolerant work (`patterns/batch-llm.md`), the Azure OpenAI Batch API processes requests asynchronously at roughly half the per-token price of online calls, against a separate quota pool. Two benefits, not one: the cost saving, and isolation — a large batch run cannot exhaust the quota the interactive path depends on. Use it for indexing-time model calls, backfills, and evaluation runs. The cost of the saving is latency measured in minutes to hours; never put an interactive request behind it.

## Prompt caching

Model calls that share a stable prefix — a long system prompt, a fixed output schema, a tool catalog, retrieved context reused across turns — benefit from prompt caching, which discounts and speeds up the repeated prefix. Structure prompts to exploit it: put the *stable* content first (system instructions, schema, examples) and the *variable* content (the user query, the specific input) last. This is a free latency and cost win that prompt layout alone unlocks; throwing the variable content in the middle forfeits it.

## Structured outputs are mandatory, not optional

Every model call whose output is consumed by code — which is almost all of them outside a chat UI — must enforce a schema. Use the model API's structured-output / JSON-schema mode and validate the response against a Pydantic model on the way out. An unparsed model response crossing into application logic is an untyped boundary; the pack does not allow those (see `serving-topology.md`). On a schema-validation failure, reject and retry once, then fail loudly — never coerce or best-effort-parse a malformed response.

## Latency budget

Decide the latency budget before choosing the model, because it constrains the choice. Track time-to-first-token separately from total completion time: for streamed, user-facing output, time-to-first-token is what the user perceives as "fast," and a model that starts streaming quickly beats a faster-overall model that pauses before it begins. Stream wherever a human reads the output (`serving-topology.md`). Measure p95 and p99, not the mean — model latency has a long tail, and the mean hides it.

## Resilience: fallback, retry, quota

The model API is a remote dependency that fails, throttles, and runs out of capacity. The internal model client must handle:

- **Retry with backoff** on transient errors and `429` throttling — bounded, with jitter, never an unbounded retry loop.
- **A fallback chain** — if the primary deployment is unavailable or quota-exhausted, fail over to a secondary (another region, or a different model tier) deployment. The eval suite must confirm the fallback model is *acceptable* for the node, or the fallback silently degrades quality.
- **Quota and region capacity** — model capacity is regional and quota-limited. A single-region deployment is a single point of failure; for production, provision in more than one region and know which is primary.
- **Timeouts** — every model call has a wall-clock timeout. A hung call must fail, not hang the request.

This is the same resilience discipline the pack applies to any remote dependency — see `microservices-resilience` — applied to the model endpoint.

## Cost

Model inference is usually the largest variable cost in an AI feature. Make it observable: the model client emits token counts (prompt and completion) and an estimated cost per call into the OpenTelemetry trace, tagged by node and feature. Without per-node cost telemetry, optimization is guesswork. The common leaks, in order of how often they bite: a frontier model on a node a small model handles; no prompt caching on a large stable prefix; online calls for work that belongs on the Batch API; and unbounded agentic loops (`patterns/agentic-loop.md`). Full cost-lever treatment is in `azure-microservices-cost-review`.

## Verification questions

1. Is the model behind a single internal client wrapper that owns retries, tracing, cost accounting, and fallback?
2. Is the model chosen per node, starting at the smallest tier that passes the node's eval — not defaulted to the largest?
3. Is the hosting mode (standard vs PTU) a recorded decision backed by a utilization calculation, not a default?
4. Does every code-consumed model call enforce and validate a structured-output schema?
5. Is there a bounded retry policy, a fallback chain, and a timeout on every model call — and is the fallback model eval-confirmed as acceptable?
6. Are prompt token counts and per-call cost emitted into the trace, tagged by node?
7. Is high-volume, latency-tolerant model work on the Batch API rather than the online path?

## What to read next

- `patterns/batch-llm.md` — the archetype that uses the Batch API
- `retrieval-design.md` — the embedding model and its consistency requirement
- `serving-topology.md` — where the model client sits and how its trace joins the others
- `azure-microservices-cost-review` — token and inference cost levers
- `microservices-resilience` — retry, timeout, and fallback discipline
