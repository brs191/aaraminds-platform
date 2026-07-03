# Query Discipline

This reference covers writing individual queries well — parameterization, query plans, the N+1 problem, and projecting narrowly. It is the per-query craft beneath the data-access layer.

## Parameterize — every query, every time

Every value that varies — an id, a filter, anything from a user, a model, or another service — is a **bound parameter**, passed to the driver alongside the query, never concatenated or interpolated into the query text. This is the security rule — a string-built query is an injection waiting for a value with a quote in it — but it is also correctness and performance: a parameterized query is one statement the engine plans once and reuses, where a string-built query is textually unique every call and re-planned every time. There is no query for which string-building is the right choice. If a query's *structure* must vary — an optional filter, a dynamic sort column — build that structure from a fixed allowlist of known-safe fragments, never from the input value itself.

## Read the query plan

When a query is slow, the answer is in its execution plan, not in guesses. Use the engine's plan tool — `EXPLAIN` / `EXPLAIN ANALYZE` on Postgres, `PROFILE` on Neo4j — and read what the engine actually does: a sequential scan where an index was expected, a join order that materializes a huge intermediate, a graph traversal expanding far more nodes than the result. Fix the cause the plan shows — an index, a rewritten query, a different start point — rather than adding indexes hopefully. A plan read once tells you more than an hour of tuning by feel.

## The N+1 problem

The most common data-access performance bug: code fetches a list, then issues one more query per item — 1 query becomes 1 + N. It hides easily behind an ORM or a tidy-looking loop. Detect it by watching query counts under realistic load, or with a query log: a request that should run a handful of queries running hundreds is N+1. Fix it by fetching what the loop needs in one query — a join, an `IN` over the collected ids, a single traversal returning the whole neighbourhood — rather than a query per element.

## Project narrowly

Return the columns or properties the caller needs, not `SELECT *` and not whole nodes. Over-projection ships data across the wire and through serialization that the caller discards — invisible in a small test, expensive at scale, and especially costly for a graph traversal returning a large node set. A query's result shape is part of its design: name the fields.

## Statement reuse

A parameterized query lets the engine cache its plan; reuse the same statement text so that cache is hit. Prepared statements — explicitly, or via the driver's automatic statement caching — make this concrete for hot queries. This is a free win that string-built queries forfeit entirely, one more reason the parameterization rule is not negotiable.

## Verification questions

1. Is every query parameterized — values bound by the driver, never concatenated — and is any dynamic structure built from an allowlist, not from input?
2. When a query is slow, is its execution plan read (`EXPLAIN` / `PROFILE`) and the cause fixed — rather than indexes added by guess?
3. Is the code free of N+1 — collections fetched in one query, not one query per element?
4. Does each query project only the fields the caller needs?
5. Are hot queries reusing cached statement plans?

## What to read next

- `graph-traversal-queries.md` — parameterizing and shaping traversals
- `data-access-layer.md` — where queries are structured in code
- `azure-data-tier-design`, `references/query-execution-and-indexing.md` — the indexing design
- `test-engineering` — testing queries against a real engine
