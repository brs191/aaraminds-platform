# The Graph Write Path

This reference covers building and updating the graph — the code that writes nodes and edges. The graph is rebuilt repeatedly, so the write path's defining requirement is idempotence.

## Building a graph is upserts, not inserts

The extractor (`codebase-extraction-engineering`) re-runs — on a schedule, per commit, incrementally. So the write path is not "insert the nodes"; it is "make the graph match the current extraction," run after run. Every write is therefore an **upsert**: create the node if it is new, update it if it exists, and — critically — produce the same result whether it runs once or five times. A write path built on plain inserts double-writes on the second run and corrupts the graph.

## MERGE on the deterministic ID — idempotent by construction

Cypher's `MERGE` is the upsert: `MERGE (n:Method {id: $id})` finds the node with that id or creates it, then `SET` applies the current properties. Keyed on the **deterministic ID** from the extractor (`codebase-comprehension`'s identity scheme — the hash of the natural key), `MERGE` is idempotent by construction: the same method always has the same id, so re-running the build re-merges the same node rather than duplicating it. The deterministic ID is what makes the whole write path safe to re-run; without it, idempotence is impossible.

Use `ON CREATE` / `ON MATCH` to separate first-write fields (created timestamp) from every-write fields (current properties, build stamp):

```cypher
UNWIND $batch AS row
MERGE (n:Method {id: row.id})
  ON CREATE SET n.createdBuild = $buildVersion
  ON MATCH  SET n.updatedBuild = $buildVersion
SET n.name = row.name, n.signature = row.signature,
    n.sourceFile = row.file, n.sourceLine = row.line,
    n.buildVersion = $buildVersion
```

## Batched writes with UNWIND

A repository's graph is large. Write it in batches — group nodes and edges into transactions of a bounded size (typically hundreds to a few thousand operations) rather than one giant transaction or one transaction per node. The idiomatic Cypher is one `UNWIND` over a parameter list per transaction (as above): one round trip, one plan, many rows. One giant transaction holds locks too long and risks memory limits; one-per-node pays transaction overhead per element. Batched writes also let a failed build resume from the last committed batch. **Write all nodes before the edges that connect them**, so an edge's endpoints always exist.

Edges merge the same way — `MERGE` the relationship between two already-merged endpoints, carrying relationship properties:

```cypher
UNWIND $edges AS e
MATCH (a:Method {id: e.fromId})
MATCH (b:Method {id: e.toId})
MERGE (a)-[r:CALLS]->(b)
SET r.callSites = e.callSites, r.buildVersion = $buildVersion
```

## Incremental updates — merge, and delete the delta

When the extractor runs incrementally (`codebase-extraction-engineering`, `references/incremental-rebuild-and-identity.md`), the write path applies a *delta*: upsert the changed nodes and edges, and **delete** what the new extraction no longer contains. Deletion is the part teams forget — a method removed from the source must be removed from the graph, or the graph accumulates ghosts. Stable ids make the delete set a set difference; the cheap, robust form is "delete anything not touched by this build":

```cypher
// anything carrying an older buildVersion was not re-merged this run -> it is gone from source
MATCH (n:Method) WHERE n.buildVersion <> $buildVersion
DETACH DELETE n
```

`DETACH DELETE` removes the node and its relationships together, so no dangling edges remain.

## Build versioning and the completeness marker

Each batch is a transaction — it commits atomically or not at all. Stamp every node and edge with the `buildVersion` (above) so a partially-applied build is detectable, resumable, and — critically — so a reader can pin one consistent snapshot (`graphrag-retrieval.md` relies on this). Do not leave the graph in a state where some of a build's nodes exist with no record that the build is incomplete: flip a `buildComplete` marker in the final transaction.

```cypher
MERGE (b:BuildMeta {version: $buildVersion})
SET b.complete = true, b.finishedAt = datetime()
```

Readers query against `BuildMeta {complete: true}` and pin its version; a build in progress is invisible to them until the marker flips.

## Concurrency

One writer per graph during a build. Concurrent builds writing the same nodes race on `MERGE` and can deadlock on relationship locks. Serialize builds (a build lock / queue); if a new commit arrives mid-build, finish or cancel the current build before starting the next, rather than interleaving two `buildVersion`s into one graph.

## Verification questions

1. Is every write an upsert (`MERGE`), producing the same graph whether it runs once or many times?
2. Are nodes and edges merged on the extractor's deterministic IDs, with `ON CREATE` / `ON MATCH` separating first-write from every-write fields?
3. Are writes batched with `UNWIND` into bounded transactions, with nodes written before their edges?
4. Does an incremental update delete what the new extraction no longer contains (e.g. by stale `buildVersion`), using `DETACH DELETE`?
5. Is each node/edge stamped with `buildVersion` and a `buildComplete` marker flipped last, so partial builds are invisible and one snapshot is pinnable?
6. Is there one writer per build, with concurrent builds serialized rather than interleaved?

## What to read next

- `graph-traversal-queries.md` — querying the graph this builds
- `graphrag-retrieval.md` — the reader that pins `buildComplete` / `buildVersion`
- `codebase-extraction-engineering`, `references/incremental-rebuild-and-identity.md` — the delta this applies
- `codebase-comprehension`, `references/incremental-code-modeling.md` — the identity scheme
- `azure-data-tier-design`, `references/graph-databases.md` — the graph engine
