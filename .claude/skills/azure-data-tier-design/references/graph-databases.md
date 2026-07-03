# Graph Databases on Azure

Most "the data has relationships" workloads are relational and belong on Postgres. A graph database earns its place only when the *queries* are traversals — variable-length paths, reachability, pattern match — over a property graph. This reference covers when the data tier is genuinely a graph, the engine options on this stack, the Gremlin-on-Cosmos traversal limit that decides most real cases, and identity resolution for graphs rebuilt from a source.

## When the data tier is a graph — and when it only looks like one

A foreign key is a relationship. A `JOIN` is a traversal. Relational engines do both well, including recursive traversal via recursive CTEs. The mistake is reaching for a graph database because the domain "has a graph in it" — an org chart, a category tree, social follows. Those are relational; Postgres with a recursive CTE answers them and you keep one engine.

The data tier is genuinely a graph when the *hot workload* is dominated by:

- **Variable-length / unbounded traversal** — "everything transitively reachable from X" where the depth is not known ahead of time.
- **Reachability and blast-radius** — "if X changes, what is affected," walking dependency edges to a fixed point.
- **Pattern match** — "find every place this subgraph shape occurs."
- **Shortest-path or weighted traversal** across a large, densely connected graph.

If the hot queries are these, a relational engine fights you: the recursive CTE works but degrades as depth and fan-out grow, and the SQL becomes unmaintainable. If the hot queries are filters, aggregations, and 1–2-hop joins, you do not have a graph workload — stay on Postgres (`engine-selection.md`).

## The engine options on this stack

Three realistic choices, in the order to consider them.

### Apache AGE on PostgreSQL — keep it on the engine you already run

AGE is a Postgres extension that adds a property-graph model and openCypher queries to a normal Postgres database. Consider it first, because the pack already defaults to Postgres Flexible Server: one engine, one backup story, one ops surface, one connection model. It fits when the graph is modest in size and traversal depth and the same service also holds relational tables. Its ceiling is real — it is not a native graph engine, and very deep traversals over a large graph will not match Neo4j — but for many "there is a graph component" services it removes the need for a second engine entirely. Availability of AGE on Azure Database for PostgreSQL Flexible Server is an extension-allowlist question — confirm it before committing.

### Neo4j on AKS, or Neo4j Aura — the native property-graph engine

Neo4j stores the graph natively with index-free adjacency, so traversal cost is proportional to the part of the graph touched, not the graph's total size. It is the strongest engine for deep, variable-length traversal and pattern match, and Cypher is the most ergonomic graph query language. The cost is ownership: self-hosted on AKS you operate it — StatefulSet, persistent volumes, backup, upgrades, HA — and the production tier is commercially licensed. Neo4j Aura (managed) removes the ops burden but is a third-party managed service off the Azure-native plane. Choose Neo4j when traversal depth is the defining requirement and the graph is large enough that index-free adjacency is the deciding factor.

### Azure Cosmos DB for Apache Gremlin — on-stack, managed, limited where it matters

Cosmos DB exposes a property graph through the Gremlin API. It is the on-stack choice — Cosmos DB is already in this pack's stack, so it keeps the data plane Azure-native, managed, and multi-region with no new ops surface. For graphs whose queries are shallow — 1–3 hops, well bounded — it is a reasonable pick.

The limit is the one that decides most real cases: **Cosmos for Gremlin partitions the graph, and deep variable-length traversals are exactly where partitioned Gremlin breaks down.** A traversal that crosses partition boundaries fans out across physical partitions; an unbounded `repeat()` / blast-radius walk fans out further at every hop; RU cost climbs steeply, latency with it, and parts of the Gremlin step library are unsupported or degrade. This is not a tuning problem — it is the architecture: a partitioned store and index-free deep traversal are in tension. Cosmos Gremlin is fine for a shallow, bounded graph; it is the wrong engine for a blast-radius workload.

## The decision

| Signal | Engine |
|---|---|
| Graph component is modest; service also has relational tables; you already run Postgres | Apache AGE on Postgres |
| Deep, variable-length traversal / blast-radius / pattern match is the defining workload; graph is large | Neo4j — AKS self-hosted, or Aura managed |
| Queries are shallow (1–3 hops) and bounded; staying Azure-native and managed outweighs traversal depth | Cosmos DB for Gremlin |
| "It has relationships" but the hot queries are filters and 1–2-hop joins | Not a graph workload — Postgres (`engine-selection.md`) |

The decisive question is **traversal depth**, and it is asked *before* the on-stack-convenience question. Picking Cosmos Gremlin because it is on-stack, then discovering the blast-radius queries it was bought for do not perform, is the graph-tier version of the access-pattern-inversion failure this skill's critical decision rule warns about.

## Identity resolution — stable node IDs across rebuilds

A graph rebuilt from a source — re-derived on a schedule or per change — needs the *same real-world entity* to receive the *same node id* on every build, or every rebuild looks like a wholesale change and diffs become meaningless.

Two classes of node:

- **Nodes with a natural key** — derive a deterministic id by hashing the natural key. Same key in, same id out, every build. (For source code: a fully-qualified type name; a method keyed by its type plus name plus *ordered parameter-type list*, since overloading makes the name alone non-unique.)
- **Nodes with no natural key** — inferred or derived concepts. There is no stable key to hash, so each build assigns a fresh id and a separate **identity-resolution step** matches new nodes against the prior build by type and attribute similarity, carrying the id forward where the match holds. This is a genuinely hard problem — the matcher has false positives and false negatives, and both corrupt the diff. Keep deterministic-keyed and resolved nodes distinguishable, tune the matcher against known rebuilds, and treat identity resolution as a first-class component, not a detail.

## Worked example — the Code Intelligence Factory knowledge graph

The Code Intelligence Factory builds a code knowledge graph as its system of record, and its hottest query is blast radius — "if this method changes, what breaks," an unbounded backward traversal over `CALLS` / `PART_OF` edges. Its schema notes flag the engine as an open decision: Cosmos DB for Gremlin (on-stack) versus Neo4j on AKS.

The decision rule above resolves it. Blast radius is deep, variable-length traversal — the defining workload, not an occasional query — which is the row that points at Neo4j, and the on-stack appeal of Cosmos Gremlin does not override it: a partitioned Gremlin store fans out and throttles on exactly the query the product exists to answer. The honest call is Neo4j — Aura if the team will not own a stateful service on AKS — accepting a non-Azure-native managed engine as the price of the traversal workload. Cosmos Gremlin would be defensible only if the CIF's queries were shallow neighbourhood lookups; they are not.

Identity resolution is the CIF's other graph problem: code-layer nodes hash from natural keys and are stable across rebuilds for free, but inferred nodes — components, requirements — have no natural key and need the resolution step above, or the product's regeneration diff stops being meaningful.

## Anti-pattern — a graph database for relational data

**Bad:** a service has an org chart, or a product category tree, and the team provisions a graph database for it.

**Why it fails:** a tree and a shallow hierarchy are relational. Postgres answers "all reports under this manager" with a recursive CTE, on the engine you already run, with one backup and one ops model. The graph database adds a second engine, a second skill set, and a second failure domain to buy a query you already had.

**Detection signal:** the graph engine's hot queries are all 1–2 hops; there is no variable-length or pattern-match query in the workload; the same service still needs a relational store alongside it.

**Fix:** model it relationally and use a recursive CTE for the hierarchy. Reserve graph engines for workloads where unbounded traversal is the point.

## Verification questions

1. Is the hot workload genuinely traversal — variable-length paths, reachability, pattern match — rather than filters and 1–2-hop joins?
2. Was traversal depth assessed *before* the on-stack-convenience argument for Cosmos Gremlin?
3. If Cosmos for Gremlin was chosen, are the real queries shallow and bounded — not blast-radius or unbounded `repeat()` traversals?
4. Was Apache AGE on Postgres considered first, to avoid standing up a second engine?
5. If the graph is rebuilt from a source, do natural-keyed nodes get deterministic hashed ids, and is there an identity-resolution step for nodes with no natural key?
6. Is the graph engine's HA/DR and backup story defined to the same standard as the rest of the data tier (`ha-dr-data-tier.md`)?

## What to read next

- `engine-selection.md` — the access-pattern-first decision this reference plugs into
- `cosmos-db-design.md` — Cosmos DB partitioning and the RU model that constrains Gremlin
- `partitioning.md` — why partitioned stores struggle with cross-partition traversal
- `ha-dr-data-tier.md` — HA/DR for a self-hosted Neo4j on AKS
- Related skill: `ai-application-architecture`, `references/retrieval-design.md` — graph-RAG / structured retrieval over a graph like this
