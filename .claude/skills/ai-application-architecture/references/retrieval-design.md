# Retrieval Design

Retrieval design is everything between "the corpus" and "the prompt the model sees": how the corpus is chunked, embedded, indexed, queried, and ranked, and how retrieval quality is measured. It is the substance behind the RAG archetype (`patterns/rag.md`). The storage engine — vector index internals, sizing, partitioning — is `azure-data-tier-design`; this reference is about the *design of retrieval*, not the database.

## Retrieval quality is the ceiling on generation quality

A generation grounded on retrieved context can be no better than the context retrieved. If the answer-bearing passage is not in what was retrieved, the model has three options and all are bad: answer from parametric memory (ungrounded), hedge, or hallucinate. The model cannot tell you it had bad context — a retrieval miss is invisible at generation time. So retrieval is not a preprocessing detail; it is the component that sets the upper bound on the whole feature. Design it deliberately and measure it directly (see retrieval evaluation below).

## Chunking — split by structure, not by length

The unit of retrieval is the chunk. Fixed-length character chunking is the default that quietly caps quality: it cuts mid-sentence, mid-function, mid-table, splitting one idea across two chunks so neither retrieves well.

Chunk along the natural structure of the source. Prose: by section and paragraph. Code: by method, class, or file — never by line count. Carry a small overlap so a concept straddling a boundary survives in at least one chunk. Attach metadata to every chunk — source path, section heading, commit, type — because metadata is what makes hybrid filtering and citation possible. Right-size the chunk: too large dilutes the embedding and wastes prompt tokens, too small loses the surrounding context that makes a passage meaningful. Tune the size against the retrieval eval, not by intuition.

## Embeddings — one model, index and query on the same one

Chunks are embedded into vectors by an embedding model. The single hard rule: the index and the query must be embedded by the *same* model. Re-embedding the corpus with a new model means re-indexing all of it; a query embedded by a different model than the index returns meaningless nearest-neighbours. Pin the embedding model explicitly, treat a change to it as a full re-index, and budget for that. Choose the model on the same axes as any other (`model-and-inference-layer.md`); embedding dimensionality trades index size and query speed against retrieval fidelity.

## The index: hybrid search on Azure AI Search

The pack's retrieval index is **Azure AI Search**. Use **hybrid search** — vector similarity *and* keyword (BM25) — not vector search alone. Pure vector search misses exact-term matches: identifiers, error codes, names, acronyms — precisely the high-signal tokens a keyword index nails. Pure keyword search misses paraphrase and concept matches. Hybrid runs both and fuses the rankings, and it is the correct default.

On top of hybrid, enable the **semantic ranker**, which re-scores the fused top results with a deeper relevance model. The combination — hybrid retrieval plus semantic ranking — is the configuration to start from; deviate only with an eval that shows the deviation helps. Index schema, replica and partition sizing, and vector configuration are `azure-data-tier-design`.

## Query-side: rewriting and expansion

The raw user query is often not the best retrieval query. Three techniques, each earning its place against the eval:

- **Query rewriting** — an LLM rewrites a conversational or underspecified query into a retrieval-optimized one (resolving pronouns from conversation history, expanding abbreviations).
- **Multi-query** — generate several query variants, retrieve for each, union the results. Helps recall on queries that can be phrased many ways.
- **HyDE (hypothetical document embeddings)** — embed a hypothetical *answer* rather than the question, since an answer is closer in vector space to the real answer chunks.

These add a model call and latency before retrieval. Add them when the retrieval eval shows a recall gap they close — not preemptively.

## Reranking

Retrieval optimizes for recall — get the right chunk into the candidate set. Reranking optimizes for precision — get it to the *top* of that set. Retrieve a generous candidate set (a larger top-k), then rerank with a cross-encoder or the semantic ranker and pass only the best few to the prompt. This directly attacks context dilution: a smaller, higher-precision context beats a large diluted one on both quality and token cost.

## Retrieval evaluation — measure it or you are guessing

Retrieval is tuned against an eval or it is tuned by feel, and feel is wrong. Build a golden set of queries each annotated with the chunks that *should* be retrieved, and measure:

- **recall@k** — is the answer-bearing chunk in the top-k? This is the retrieval ceiling.
- **context-precision** — of what was retrieved, how much is actually relevant? Low precision signals dilution.
- **context-recall** — does the retrieved context cover everything the ideal answer needs?

Every chunking, embedding, hybrid-weight, query-rewrite, and rerank change is a change to these numbers; CI runs the retrieval eval and the gate fails on regression. This is the retrieval-specific slice of the broader eval discipline in `evaluation.md`.

## Freshness and incremental indexing

A RAG index is a cache of a corpus and goes stale the moment the corpus changes. Stamp every index build with a corpus version — a commit SHA, a dataset id — so staleness is detectable rather than silent. Prefer incremental indexing — re-embed and update only changed chunks, wired to the source of change (a commit hook, a change feed) — over periodic full rebuilds, which are expensive and leave a stale window. Full rebuilds remain the right move when the chunking strategy or the embedding model changes, because both invalidate every existing vector.

## Structured and graph retrieval — when the corpus is not prose

Vector RAG assumes the corpus is text whose relevance is semantic similarity. When the corpus has hard structure, similarity is the wrong primary retriever.

The Code Intelligence Factory is the clean example. Its corpus is a **code knowledge graph**, and its highest-value queries are structural, not semantic: "what is the blast radius of this method" is a backward graph traversal over `CALLS` / `PART_OF` edges; "what tests cover this component" is a `COVERS` traversal. No embedding captures "transitively calls" — that is a graph walk, and an approximate vector match would be actively wrong for a blast-radius query a QA scope depends on.

The pattern there is **graph-RAG / structured retrieval**: the graph is the *primary* retriever — traversals answer the structural questions exactly — and vector search is *secondary*, applied only to the free-text islands in the graph (ADR prose, doc-strings, commit messages, PR discussion). Retrieval for a question like "explain this component's role" then fuses a graph neighbourhood (deterministic, exact) with vector hits over the attached text (semantic, approximate). The general rule: retrieve structure with traversals and prose with vectors, and do not force one to do the other's job. Graph engine selection — and the known limits of deep variable-length traversals on different engines — is in `azure-data-tier-design`.

## Verification questions

1. Are chunks split along the source's natural structure (sections, methods) rather than by fixed length, with overlap and metadata?
2. Are the index and the query embedded by the same, explicitly pinned model?
3. Is retrieval hybrid (vector + keyword) with the semantic ranker enabled — or is a deviation backed by an eval?
4. Is there a golden retrieval set, and does CI gate on recall@k, context-precision, and context-recall?
5. Is every index build stamped with a corpus version, and is indexing incremental where the corpus changes often?
6. For a structured corpus, is structure retrieved by traversal and prose by vector search — rather than vectors forced onto structural questions?

## What to read next

- `patterns/rag.md` — the archetype this reference supports
- `azure-data-tier-design` — vector index internals and graph engine selection
- `model-and-inference-layer.md` — embedding model choice
- `evaluation.md` — the full evaluation discipline retrieval eval is part of
