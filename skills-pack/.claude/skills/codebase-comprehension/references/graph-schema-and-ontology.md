# Graph Schema and Ontology

This reference defines the code knowledge-graph schema — the node and edge taxonomy, the property and identity scheme, and how provenance and versioning are modeled. It is the CIF's **M0 deliverable**: the graph is the system-of-record, and the write path (`data-access-engineering`, `references/graph-write-path.md`), the traversals, and GraphRAG retrieval are all defined against this schema. Design it before the extractor writes a node; a wrong schema makes every downstream layer wrong.

## Nodes — the label taxonomy, tagged by provenance

| Layer | Labels | Provenance |
|---|---|---|
| Code | `Repository`, `Module`, `Package`, `File`, `Type` (Class/Interface/Enum/Record), `Method`, `Field`, `Endpoint` | Deterministic — parsed |
| Design | `Component`, `Boundary`, `ExternalIntegration`, `DataStore` | Deterministic when annotation-derived; else inferred |
| Capability | `Capability`, `Requirement`, `Actor` | Inferred — weakest, needs non-code sources |

The lower layers are evidence; the upper layers are interpretation (`codebase-comprehension`'s deterministic-vs-inferred rule, made structural).

## Edges — directional relationship types

`CONTAINS` (structural nesting: Repository→Module→Package→File→Type→Method), `CALLS` (Method→Method), `IMPORTS`, `EXTENDS`, `IMPLEMENTS`, `INJECTS` (DI edge), `EXPOSES` (Type→Endpoint), `READS` / `WRITES` (→`DataStore`), `BELONGS_TO` (→`Component`), `REALIZES` (`Component`→`Capability`). Edges carry their own provenance: `CALLS` resolved from the AST is deterministic; `BELONGS_TO` from clustering is inferred.

## Properties — every node and edge

Every node carries: `id` (deterministic, below), `name`, `kind`, `sourceFile`, `sourceLine`, `buildVersion`, `provenance` (`deterministic` | `inferred`), `confidence` (`1.0` for parsed facts). Edges carry `type`, `provenance`, and edge-specific properties (`CALLS.callSites`). `sourceFile`/`sourceLine` are the provenance that makes GraphRAG citations resolvable.

## Identity — the deterministic ID

The `id` is a hash of the node's **natural key**, not a sequence number: the fully-qualified, overload-aware signature for a `Method` (`com.acme.Order#total(java.util.List)`), the FQN for a `Type`. Same artifact → same id on every build, which is what makes `MERGE` idempotent (`graph-write-path.md`) and the rebuild diff meaningful. Name-based identity breaks the moment a method is overloaded or renamed; the signature-hash does not.

```cypher
CREATE CONSTRAINT node_id IF NOT EXISTS FOR (n:Method) REQUIRE n.id IS UNIQUE;
// one per node label; the id is the MERGE key and the citation target
```

## Provenance and confidence are first-class

Tag every node and edge as `deterministic` or `inferred` and never blend them. A model that presents an inference with the authority of a parsed fact produces confident fiction — and the whole value of the graph is that its consumers, including a GraphRAG answer, can trust what is deterministic. Inference is a separate, visible layer on top, never mixed into the parsed layer.

## Versioning and evolution

Stamp every node and edge with `buildVersion`; a `BuildMeta {version, complete}` node marks a finished build (`graph-write-path.md`), and GraphRAG pins it (`graphrag-retrieval.md`). Evolve the schema **additively** — new labels, new optional properties — so existing ids stay stable and old queries keep working; a breaking change to the identity scheme forces a full rebuild and invalidates every stored citation.

## Multi-language

Keep the schema **language-agnostic at the core** — `Type`, `Method`, `CALLS`, the identity scheme — so a second language (the CIF's v2 React frontend, a Postgres schema) is a new extractor feeding the same model, not a second schema. Language-specific concepts attach as additional labels/properties, never by forking the core.

## Failure modes

- **No explicit schema** → ad-hoc labels drift between extractor passes; the graph means different things in different places.
- **Name-based identity** → overloads collapse, renames duplicate; `MERGE` and the diff break.
- **Inferred blended into deterministic** → consumers can't tell ground truth from a guess; citations cite hypotheses.
- **No `buildVersion`** → can't diff builds, can't pin a snapshot, can't retrieve reproducibly.

## Verification questions

1. Is there an explicit node/edge taxonomy, with every type tagged deterministic or inferred?
2. Is the `id` a hash of an overload-aware natural key (signature/FQN), stable across rebuilds?
3. Does every node carry `sourceFile`/`sourceLine` so a GraphRAG citation resolves to source?
4. Is provenance/confidence on every node and edge, with inferred never blended into deterministic?
5. Is every node/edge `buildVersion`-stamped, and does schema evolution stay additive?
6. Is the core schema language-agnostic so a second language is a new extractor, not a new schema?

## What to read next

- `call-and-dependency-graphs.md` — the structural edges this schema names
- `incremental-code-modeling.md` — the identity scheme and clean diffs
- `data-access-engineering`, `references/graph-write-path.md` — the write path that `MERGE`s this schema
- `data-access-engineering`, `references/graphrag-retrieval.md` — the retriever that serializes it with citations
- `azure-data-tier-design`, `references/graph-databases.md` — the engine that stores it
