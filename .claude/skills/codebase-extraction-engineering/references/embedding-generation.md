# Embedding Generation

This reference covers producing the node embeddings that GraphRAG anchors on. Embeddings are an **extraction-time** concern: the retriever (`data-access-engineering`, `references/graphrag-retrieval.md`) seeds on vector similarity, and those vectors are written alongside nodes during the build, at one `buildVersion`, so the graph and its vectors never drift apart.

## Why the extractor owns embeddings

GraphRAG retrieval anchors by finding semantically-relevant seed nodes, then expands structurally. The anchor needs a vector per searchable node, and the only place that vector can be produced consistently — same content, same model, same build — is the emit stage of the extractor. Generating embeddings at query time is slow, non-reproducible, and lets the vectors fall out of sync with the graph.

## What to embed — meaning, not bytes

Embed the node's **semantic text**, not its raw source. For a `Method`: the signature + name + Javadoc/docstring (+ optionally a short body summary). For a `Type`: name + doc + member names. Skip purely structural nodes (`Package`, `Module`) unless they are queried. Raw source bytes embed noise — whitespace, imports, boilerplate — and dilute the signal the seed search depends on.

## Which model, and pin its version

Use **Azure OpenAI embeddings** (`text-embedding-3-small` for cost, `-large` for recall). The embedding **model + version is part of the build's identity**: a model change re-spaces the whole vector index, so record it and re-embed when it changes, exactly as a schema change forces a rebuild. Store the model id on `BuildMeta` so retrieval knows which space it is querying.

## Incremental and deterministic

Re-embed only what changed. Content-hash the embed input (the semantic text above); if the hash matches the prior build, reuse the stored vector — most nodes are unchanged commit to commit, and embeddings cost per call. Cache by `(embed-input hash, model version)`. This keeps the build cheap and makes it deterministic in the way that matters: the same node text + same model yields the same stored vector, so a GraphRAG eval (`ai-evaluation-harness`, `references/retrieval-and-rag-evaluation.md`) is reproducible.

## Where the vector goes

Write each node's vector to the index `azure-data-tier-design` selected — the Neo4j vector index (vector co-located with the node), Postgres `pgvector`, or Azure AI Search — in the **same build transaction path** as the node, stamped with the same `buildVersion`. Co-locating the write is what guarantees the graph and the vectors describe the same snapshot.

## Security — untrusted source to an external API

The extractor runs on untrusted repositories (`build-integration-and-generated-code.md`), and embed inputs are sent to the Azure OpenAI endpoint. Do not send obvious secret-bearing text; the embed input is constructed from signatures and docs, not raw config. The call goes out from the sandboxed extractor over an allowlisted egress to the Azure endpoint only.

## Failure modes

- **Embed at query time** → slow retrieval and vectors that drift from the graph.
- **Embed raw source bytes** → noise drowns the signal; seed search degrades.
- **No model-version pinning** → a model upgrade silently re-spaces the index; old and new vectors are incomparable.
- **Re-embed everything each build** → avoidable cost; hash and cache.
- **Vectors written at a different `buildVersion` than the nodes** → retrieval anchors to a stale space and cites the wrong snapshot.

## Verification questions

1. Are embeddings generated at extraction/emit time and written with the node, not computed at query time?
2. Is the embed input the node's semantic text (signature, name, doc), not raw source bytes?
3. Is the embedding model + version recorded on the build and re-embedded when it changes?
4. Is re-embedding incremental — content-hashed, cached — and deterministic for unchanged nodes?
5. Are vectors written at the same `buildVersion` as the nodes, into the data-tier's chosen index?

## What to read next

- `extractor-architecture.md` — the emit stage embeddings are produced in
- `incremental-rebuild-and-identity.md` — the change detection embedding reuse keys on
- `data-access-engineering`, `references/graphrag-retrieval.md` — the retriever that consumes these vectors
- `azure-data-tier-design`, `references/graph-databases.md` — the vector-index choice
