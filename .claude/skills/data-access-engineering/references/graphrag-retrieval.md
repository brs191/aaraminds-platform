# GraphRAG Retrieval

This reference covers the CIF's product layer: answering a question by retrieving a relevant *subgraph* — not text chunks — and serializing it as grounded, cited context for an LLM. Retrieval quality and provenance are what make the answer trustworthy. It is the read-side counterpart to `graph-write-path.md` and the data-access realization of the RAG archetype in `ai-application-architecture`.

## Why a subgraph beats chunk RAG

Classic RAG embeds text chunks and returns the top-k by vector similarity. It misses *structure* — the caller of a function, the blast radius of a change, the dependency path between two services — because those facts live in edges, not in any single chunk. GraphRAG retrieves a connected subgraph: seed by relevance, expand by structure, hand the model the entities *and the relationships between them*. The graph supplies the structural context vectors cannot, and every node carries an id back to its source, which is the basis for citation.

## The retrieval pipeline — five stages

1. **Anchor — find seed nodes.** Map the question to entry nodes by (a) vector similarity over node embeddings, (b) full-text / keyword on names and signatures, or (c) exact structural lookup of a known symbol id. Hybrid is the default: vectors for "about what," structural lookup for "this exact symbol."
2. **Expand — bounded traversal to a subgraph.** From the seeds, walk a *bounded* neighbourhood (`graph-traversal-queries.md`) — callers/callees to depth N, dependencies, the blast-radius set. The bound is non-negotiable; an unbounded expand returns the whole graph.
3. **Rank and prune to a budget.** A subgraph is usually too big for the context window. Score nodes (seed distance, relevance, degree/centrality, recency), keep the top set within an explicit token budget. Pruning is where retrieval quality is won or lost.
4. **Serialize for the model.** Render the pruned subgraph as compact structured context — entities with their key properties and the edges between them — not raw driver rows, not whole node objects. Structured JSON or a terse textual form; keep stable ids in the payload.
5. **Cite — attach provenance.** Every node carries its source (file, line, commit) and its graph id, included in the serialized context, so the model can cite and a downstream check can verify each claim. **No provenance, no trust** — an answer that cannot point back to a source node is exactly the hallucination GraphRAG exists to prevent.

```cypher
// Retrieve: vector/full-text supplies $seedIds; expand a bounded neighbourhood;
// return entities WITH provenance, capped to a token budget.
UNWIND $seedIds AS sid
MATCH (s:Method {id: sid})
CALL apoc.path.expandConfig(s, {
  relationshipFilter: 'CALLS>|DEPENDS_ON>',
  minLevel: 1, maxLevel: $maxDepth, uniqueness: 'NODE_GLOBAL'
}) YIELD path
WITH DISTINCT last(nodes(path)) AS n
RETURN n.id AS id, n.name AS name, n.kind AS kind,
       n.sourceFile AS file, n.sourceLine AS line   // provenance travels with the fact
ORDER BY n.relevance DESC
LIMIT $budget
```

## Hybrid retrieval — vector for seeds, graph for context

The vector index finds *semantically* relevant entry points; the graph adds the *structurally* relevant neighbourhood. Use them in that order: vector or full-text to anchor cheaply, then graph expansion for the context that makes the answer correct. The vector index can live in the graph engine (Neo4j vector index), in Postgres (`pgvector`), or in Azure AI Search — that engine choice is `azure-data-tier-design`'s; this layer consumes whatever it selected, it does not pick.

## Token budget and determinism

The context window is the hard constraint. Budget it explicitly: cap node count, prefer high-score nodes, summarize or drop distant context, never serialize whole paths when the node set is the answer. For evaluation and reproducibility, make retrieval **deterministic** — same question + same graph version yields the same subgraph — and cache by `(question embedding, buildVersion)`. Determinism is what lets `ai-evaluation-harness` score retrieval at all; a retriever that returns a different subgraph each call cannot be regression-tested.

## Freshness — retrieve against a complete, known graph

Read only a graph marked complete (`graph-write-path.md`'s `buildComplete` / version stamp). Retrieving mid-build returns a half-written subgraph; citing a node from a stale build points the user at code that no longer exists. Pin the `buildVersion` for a retrieval session so every cited id resolves against one consistent snapshot.

## Failure modes

- **Return the whole subgraph** → token blowout and cost. Prune to a budget.
- **No provenance** → unverifiable answer; the model can hallucinate freely. Carry source + id on every node.
- **Vector-only retrieval** → misses the structural context (callers, blast radius) that is the entire point of using a graph.
- **Unbounded expand** → the "subgraph" is the graph. Always bound the walk.
- **Retrieve against a building or stale graph** → cites code that is half-written or deleted. Pin a complete `buildVersion`.

## Verification questions

1. Does retrieval return a bounded subgraph with provenance (source + id) on every node, not unattributed text?
2. Is anchoring hybrid — vector / full-text for seeds, structural lookup where the symbol is known?
3. Is the subgraph pruned to an explicit token budget before serialization?
4. Is retrieval deterministic and pinned to a complete `buildVersion`, so it is reproducible and never cites a half-built graph?
5. Is the serialized context structured entities-and-edges, not raw rows or whole nodes?

## What to read next

- `graph-traversal-queries.md` — the bounded expansion walk this layer runs
- `graph-write-path.md` — the `buildComplete` / version stamp retrieval pins to
- `ai-application-architecture` — the RAG archetype and serving topology this feeds
- `ai-evaluation-harness` — scoring retrieval groundedness and answer faithfulness
- `azure-data-tier-design`, `references/graph-databases.md` — the graph + vector engine choice
