---
name: data-access-engineering
description: Implements the data-access layer against the stores azure-data-tier-design selects: graph traversal queries (Cypher/Gremlin blast-radius walks), the idempotent graph write path, GraphRAG retrieval (bounded subgraph with provenance for LLM context), relational expand/contract migrations, query discipline, and a repository boundary keeping queries out of business logic. Use when writing traversals, building the graph write path, retrieving graph context for an LLM, authoring migrations, or hardening query code. Do not use for engine choice, schema, sizing, or HA/DR (use azure-data-tier-design) or the extraction pipeline that populates the graph (use codebase-extraction-engineering).
version: 1.1.0
last_updated: 2026-05-30
---

# Data Access Engineering

## When to use

Trigger this skill when writing the code that reads and writes the data tier — queries, migrations, the access layer. Common triggers: "write the blast-radius traversal," "build the graph write path," "retrieve graph context for an LLM," "author this schema migration," "this query is slow or unsafe," "structure the data access."

This is the implementation companion to `azure-data-tier-design`. That skill designs the data tier — engine choice, schema, partition keys, sizing, HA/DR; this skill writes the queries, migrations, and access code against it. Use them together.

Do **not** use this skill for: choosing the data engine or designing the schema, partitioning, sizing, or HA/DR (`azure-data-tier-design`); the graph engine decision (`azure-data-tier-design`, `references/graph-databases.md`); the code-extraction pipeline that populates the graph (`codebase-extraction-engineering`).

## The critical decision rule — the query is a program: shape it for the engine, never build it from a string

A query is not a request for data; it is a program the store executes, and two things decide whether it is a good one. First, its **shape must match the engine's execution model**: a deep blast-radius traversal is near-free on a native graph engine, a fan-out problem as a recursive CTE on Postgres, and a cross-partition explosion as Gremlin over Cosmos — the same intent, three costs, set by the engine `azure-data-tier-design` chose. Second, the query is **parameterized, never string-built**: an input value is a parameter the driver binds, never text concatenated into the query — that is the line between a query and an injection. Shape deliberately; parameterize always.

## Graph traversal queries

Write the traversal to the engine's model, and bound it. A blast-radius walk — "what reaches this node" — is a variable-length path on a native graph engine (`MATCH (n)<-[:CALLS*1..6]-(c)` in Cypher), not a recursive CTE bolted onto a relational table. Every variable-length traversal carries an explicit max-depth bound as a backstop; an unbounded `*` walk on a dense graph is how a query that passed in a test hangs in production. Project narrowly — return the nodes the caller needs, not whole subgraphs. On Gremlin over Cosmos, keep the traversal inside a partition where the access pattern allows it; a cross-partition deep traversal is the expensive path. `references/graph-traversal-queries.md`.

## The graph write path

Build the graph idempotently, or it cannot be built twice. Every node and edge is written with `MERGE` on a deterministic ID derived from the source artifact — never `CREATE` — so re-extracting the same code updates the graph in place instead of duplicating it. Batch writes into bounded transactions sized to the driver: not one transaction per node, not one for the whole repository. An incremental rebuild diffs against the prior build and writes only what changed; a full `DELETE`-then-rebuild is the move to avoid, because it breaks every node ID a consumer is holding. `references/graph-write-path.md`.

## GraphRAG retrieval

Retrieve a bounded *subgraph* with provenance, never dump the graph at the model. The CIF answers a question by anchoring to seed nodes (vector or full-text for "about what," structural lookup for a known symbol), expanding a *bounded* traversal to a relevant subgraph, pruning to a token budget, and serializing entities-and-edges with a source id on every node — so every claim is citable and verifiable. Provenance is the anti-hallucination mechanism: an answer that cannot point back to a source node is the failure GraphRAG exists to prevent. Retrieve only against a graph marked complete, and keep retrieval deterministic so it can be evaluated. `references/graphrag-retrieval.md`.

## Relational migrations

Change a relational schema in expand/contract steps, never in one destructive migration. Add the new column or table and deploy (expand); backfill existing rows as a separate, restartable job, not inside the migration; switch the application to the new shape; only then drop the old (contract). Each step is independently deployable, and the schema stays compatible with the running code on both sides of every deploy. Migrations are versioned, ordered, forward-only, and applied in CI against a real engine. Once a migration has run in production, prefer a forward-fix migration over a rollback. `references/relational-migrations.md`.

## Query discipline

Parameterize every query, and read its plan before it ships. Parameterization is the decision rule above, applied without exception. Beyond it: read the execution plan (`EXPLAIN ANALYZE` on Postgres, `PROFILE` on Neo4j) of any query on a hot path before it merges — a plan read in review costs minutes, the same plan read during an incident costs an outage. Watch for N+1 — a query issued inside a loop is one query becoming N — and fold it into a single set-based query or a batched fetch. A query's cost is a property of its shape and its indexes, not something to discover under production load. `references/query-discipline.md`.

## The data-access layer

Keep queries behind a boundary; business logic never sees a query string or a raw driver row. Data access lives in a repository-style layer that exposes typed operations — `callers_of(method_id) -> list[Method]` — and returns typed results, not cursors or loose dictionaries. Transaction scope is explicit and owned by the boundary, never leaked to callers as an ambient connection. The connection pool is sized deliberately and shared, not opened per request. This boundary is also where the per-language driver choice is contained — see the next section. `references/data-access-layer.md`.

## Driver and language specifics

The access principles are constant; the driver mechanics are not. In Python, use an async driver — `asyncpg` for Postgres, the Neo4j async driver for the graph — with an explicitly sized connection pool and rows mapped onto Pydantic models rather than passed around as tuples. In Go, use `database/sql` with the `pgx` driver, `context`-scoped transactions so a cancelled request releases its connection, and `sqlc`-generated typed queries so the query and its result type are checked at build time. Placeholder syntax differs — `$1` positional on Postgres, `$param` named in Cypher — but parameterization is not optional in either. The data-access boundary is where these per-language choices are contained, so business logic never touches a raw driver. `references/data-access-layer.md` · `references/query-discipline.md`.

## Test data access against a real engine

Data-access code cannot be meaningfully tested against a mocked driver. The bugs in a query live in how the *engine* executes it — a traversal that fans out, a migration that locks, an N+1 a mock cannot exhibit, a parameterized query the mock never actually runs. So test queries and migrations against the *real* engine — a disposable Postgres or Neo4j instance, in a container, in CI (`test-engineering`). A mock confirms the code called the driver; only a real engine confirms the query is correct, the migration applies, and the plan is sane. This is the one kind of code where an integration test is not optional.

## Worked example — brownfield: a blast-radius query that times out

Setup: the blast-radius query — "what is affected if this method changes" — is implemented as a recursive CTE over a relational `calls` table, and it times out on a large repository.

Decision walk: (1) Recognize the shape mismatch — an unbounded backward traversal is a graph workload, not a relational one (`azure-data-tier-design`, `references/graph-databases.md` makes this call). (2) The model belongs on a graph engine; write the traversal as a Cypher variable-length backward path, bounded by a max depth as a backstop. (3) Build the graph with idempotent `MERGE` on deterministic IDs so re-extraction merges cleanly. (4) Parameterize the start node — never interpolate it. (5) Wrap the query behind a data-access function returning a typed result, not raw records. (6) Read the query plan and index the node lookup the traversal starts from.

The wrong move is to add more indexes to the relational `calls` table — indexes do not fix a recursive CTE doing exponential fan-out.

## Anti-pattern — the string-built query

**Bad:** queries assembled by concatenating or f-string-interpolating input values into the query text. **Why it fails:** it is an injection vulnerability by construction, it defeats the engine's query-plan cache (every query is textually unique), and it breaks the moment a value contains a quote. **Detection signal:** query strings built with `+` or format-strings around input; no parameter placeholders; user or model input reaching a query as text. **Fix:** parameter placeholders bound by the driver, for every query, in every language — `query-discipline.md`.

## Verification questions

1. Is each query's shape matched to the chosen engine's execution model — not a traversal forced onto a relational store or vice versa?
2. Is every query parameterized — input bound by the driver, never concatenated into query text?
3. Is the graph write path idempotent — `MERGE` on deterministic IDs — so re-running a build does not duplicate?
4. Are schema changes done as expand/contract via a migration tool, with backfills as separate jobs?
5. Do variable-length graph traversals carry a depth bound?
6. Is data access behind a repository-style boundary returning typed results, with queries kept out of business logic?
7. Are query plans read and acted on *before* a query ships — not diagnosed after it is slow in production?
8. Are data-access integration tests run against a real disposable engine in CI, not against a mocked driver?
9. Are per-language driver choices contained behind the data-access boundary, so business logic never holds a raw connection or driver handle?
10. For GraphRAG retrieval: is the returned subgraph bounded, pruned to a token budget, and carrying provenance (source + id) on every node — retrieved against a complete, pinned `buildVersion`?

## What to read next

Tier-2 references: `references/graph-traversal-queries.md` · `references/graph-write-path.md` · `references/graphrag-retrieval.md` · `references/relational-migrations.md` · `references/query-discipline.md` · `references/data-access-layer.md`.

Related skills: `azure-data-tier-design` (designs the data tier this skill writes against — read it first) · `codebase-extraction-engineering` (the extractor that feeds the graph write path) · `codebase-comprehension` (the model and identity scheme) · `python-service-engineering` / `mcp-go-server-building` (the services that call this layer).
