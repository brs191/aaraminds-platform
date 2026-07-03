# Graph Traversal Queries

This reference covers writing traversal queries against a graph store — the Cypher and Gremlin queries that walk the knowledge graph. It is the query-writing companion to `azure-data-tier-design`'s graph-databases reference, which chooses the engine.

## A traversal is the query shape a graph exists for

The query a graph store is built to answer is the *traversal* — follow edges from a starting node, possibly to an unknown depth. "Everything this method transitively calls," "every component affected if this changes," "the dependency path between two services" are traversals. A graph store answers them by walking adjacency; a relational store answers them with recursive joins that degrade as depth and fan-out grow. Writing traversal queries well starts with recognizing that the traversal *is* the workload and writing it as one, not as a join.

## Cypher — the readable traversal language

Cypher (Neo4j, and openCypher elsewhere) expresses a traversal as a visual pattern: `(start)-[:CALLS]->(callee)` reads like the graph it matches. A traversal query names the start pattern, the relationship pattern to follow, and what to return. Its strength is that the query text mirrors the shape of the walk, which makes a complex traversal reviewable. Write the start of the traversal as a precise, indexed node lookup — the traversal's cost depends on starting from one node, not scanning all of them.

## Variable-length paths — and bounding them

The construct that makes a traversal a traversal is the variable-length path — in Cypher, `-[:CALLS*]->` follows `CALLS` edges any number of hops. It is exactly what blast radius needs and exactly where an unbounded query can run away on a densely connected graph. Always give it a bound: `*1..10` caps the depth. The bound is a backstop, not the design — if the real answer needs unbounded depth, that is a fact about the workload and an input to the engine choice, but an unbounded `*` with no cap in production code is a query waiting to hang.

**Gotcha:** Cypher will not let you parameterize the bounds of a variable-length pattern — `*1..$depth` is a syntax error. Either inline a literal (`*1..6`) or, when the depth must be dynamic, use `apoc.path.expandConfig` with `maxLevel: $depth`. Do not build the query string to interpolate the depth — that violates `query-discipline.md`.

## Blast radius — the backward traversal

Blast radius is a *backward* traversal: from a changed node, follow caller edges in reverse to every node that reaches it. In Cypher that is the variable-length pattern with the arrow reversed, the start node parameterized and indexed, and a narrow projection:

```cypher
// "What is affected if $methodId changes?" — bounded backward walk, distinct ids only
MATCH (affected:Method)-[:CALLS*1..6]->(changed:Method {id: $methodId})
RETURN DISTINCT affected.id AS id, affected.name AS name, affected.sourceFile AS file
```

This is the CIF's highest-value query and the one whose performance decides the graph-engine choice (`azure-data-tier-design`, `references/graph-databases.md`): a native graph engine walks it in time proportional to the affected subgraph; a partitioned Gremlin-over-Cosmos store fans the same walk across partitions. For a dynamic depth, the same query via APOC:

```cypher
MATCH (changed:Method {id: $methodId})
CALL apoc.path.expandConfig(changed, {
  relationshipFilter: '<CALLS',          // incoming CALLS = callers
  minLevel: 1, maxLevel: $maxDepth, uniqueness: 'NODE_GLOBAL'
}) YIELD path
RETURN DISTINCT last(nodes(path)).id AS id
```

## Index the start node

A traversal's cost is dominated by where it starts. The start-node lookup must be an index hit, not a label scan. Back the node key with a uniqueness constraint (which also creates the index):

```cypher
CREATE CONSTRAINT method_id IF NOT EXISTS FOR (m:Method) REQUIRE m.id IS UNIQUE;
```

`PROFILE` the query (`query-discipline.md`) and confirm the plan opens with a `NodeUniqueIndexSeek`, not an `AllNodesScan`. Relationship-type and property indexes help filtered traversals; the start-node index is the one that always matters.

## Paginate a large affected set

A blast-radius set can be thousands of nodes. Return **ids first**, ordered by a stable key, and page them — then hydrate detail per page — rather than streaming thousands of full nodes in one result:

```cypher
MATCH (affected:Method)-[:CALLS*1..6]->(:Method {id: $methodId})
RETURN DISTINCT affected.id AS id
ORDER BY id SKIP $offset LIMIT $pageSize
```

For deep pagination, prefer keyset (`WHERE id > $lastId ORDER BY id LIMIT $pageSize`) over large `SKIP`, which still walks the skipped rows.

## Gremlin — the same intent, the partition-aware caveat

Gremlin (the Cosmos DB graph API) expresses a traversal as a step pipeline — `g.V(start).repeat(out('CALLS')).times(N)`. The intent is the same as Cypher's; the caveat is the one from the engine reference: on Cosmos the graph is partitioned, and a deep `repeat()` crossing partitions fans out and costs RUs steeply. Write Gremlin traversals to stay within a partition where the data model allows, keep `repeat()` depths bounded and modest, and treat a deep cross-partition traversal as a signal that the workload wanted a native graph engine.

## Parameterize the start, project narrowly

Two disciplines on every traversal query. **Parameterize** the start-node id and any filter values — bound parameters, never interpolated text (`query-discipline.md`). **Project narrowly** — return the specific properties the caller needs, not whole nodes and not the matched paths, unless the path itself is the answer. A traversal that returns full nodes for a 10,000-node affected set ships ten thousand times more data than "the affected ids."

## Verification questions

1. Is the workload genuinely a traversal, and is the query written as one rather than as a recursive relational join?
2. Does every variable-length path carry a depth bound as a backstop — and is a dynamic depth handled via APOC, not string interpolation?
3. Is the blast-radius query a bounded backward traversal returning the distinct affected set?
4. Does the start node resolve via an index seek (constraint-backed), confirmed in `PROFILE`?
5. Are large result sets returned as paged ids, not thousands of full nodes in one result?
6. For Gremlin on Cosmos, are traversals partition-aware and `repeat()` depths bounded and modest?
7. Is the start node parameterized and the projection narrow?

## What to read next

- `graphrag-retrieval.md` — the retrieval layer that runs bounded expansions and serializes the result for an LLM
- `azure-data-tier-design`, `references/graph-databases.md` — the graph engine the queries run on
- `graph-write-path.md` — building the graph the traversals walk
- `query-discipline.md` — parameterization and query plans
- `codebase-comprehension`, `references/call-and-dependency-graphs.md` — what blast radius means
