# Pattern: Retrieval-Augmented Generation (RAG)

## Problem

An answer must be grounded in data the model was not trained on and that changes over time — internal documents, a code corpus, a knowledge graph. Fine-tuning bakes the data into weights and goes stale on the next change; stuffing everything into the prompt does not scale and dilutes attention. RAG retrieves the relevant slice at query time and grounds generation on it.

## Use When

- The answer depends on a private or frequently-changing corpus.
- The corpus is too large to fit in context, or large enough that stuffing it wastes tokens and degrades quality.
- Answers must cite their sources — RAG makes evidence-linking natural.
- Freshness matters: re-indexing is cheaper and safer than re-training.

## Avoid When

- The whole corpus fits comfortably in context and is stable — just include it; RAG is overhead.
- The data is small and static — a prompt constant or fine-tune may be simpler.
- The task needs no external grounding at all → `single-shot.md`.
- RAG is being chosen as a reflex. It is not the default. Name why retrieval is required.

## Shape

Two phases. **Index** (offline/batch): chunk the corpus, embed, write vectors plus metadata to Azure AI Search. **Query** (online): embed the query, run hybrid search (vector + keyword) with the semantic ranker, optionally rewrite the query and rerank results, assemble a grounded prompt, generate with citations. The Python tier owns both phases. Vector-store internals — index schema, partitioning, sizing — belong to `azure-data-tier-design`; retrieval *design* — chunking, ranking, retrieval evaluation — is `retrieval-design.md`.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Grounding | Strong — answers cite evidence and stay current with the corpus |
| Retrieval dependency | The generation is only as good as what was retrieved; a retrieval miss is invisible to the model |
| Cost | Embedding + search + larger grounded prompts |
| Latency | An extra retrieval round trip before generation |
| Operability | The index must be kept fresh as the corpus changes |

## Common Failure Modes

- **Retrieval miss** — the answer-bearing chunk is not in the top-k; the model falls back to parametric memory or hedges. Detection: retrieval eval — recall@k / context-recall on a golden set. Prevention: tune chunking and hybrid weights against that eval, not by feel.
- **Context dilution** — too many marginally-relevant chunks; the useful one is buried. Detection: a context-precision metric. Prevention: rerank, lower k, raise the relevance floor.
- **Stale index** — the corpus changed, the index did not. Detection: stamp each index build with a corpus version or commit; alert on drift. Prevention: incremental re-indexing wired to the source of change.
- **Citation fabrication** — the model cites a source that does not support the claim. Detection: a faithfulness / groundedness eval. Prevention: score faithfulness in CI and fail the gate below threshold.

## Decision Signals

Use RAG when the answer must be grounded in a corpus you control that changes over time. Do not use it as a reflex — if the corpus fits in context, stuff it; if no grounding is needed, do not retrieve.

## Worked signal — Code Intelligence Factory

The CIF's retrieval is not document RAG — it retrieves over the *code knowledge graph*. "What is the blast radius of this method" is a graph traversal, not a vector search; "explain this component's role" combines a graph neighbourhood with doc-string and ADR text. Treat it as graph-RAG / structured retrieval: the graph is the primary retriever, vector search is secondary over free text (ADRs, comments, commit messages). See `retrieval-design.md` and the graph-database reference in `azure-data-tier-design`.

## References

- `../retrieval-design.md` — chunking, hybrid search, reranking, retrieval evaluation
- `azure-data-tier-design` — vector index and graph engine selection
- `../evaluation.md` — faithfulness, context-recall, context-precision
- Pattern: `single-shot.md`, `agentic-loop.md`
