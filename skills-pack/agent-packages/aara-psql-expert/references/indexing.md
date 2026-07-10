# Indexing strategy

An index is a write-time cost paid for read-time speed. Every index slows
`INSERT`/`UPDATE`/`DELETE` and consumes space and cache. Add indexes to serve
real, measured query patterns — not speculatively.

## Index types and when each wins

- **B-tree** (default) — equality and range (`=`, `<`, `>`, `BETWEEN`,
  `ORDER BY`, `IN`). The right answer for the vast majority of columns.
- **Hash** — equality only; rarely worth choosing over B-tree since B-tree also
  serves equality and more. Consider only for very large equality-only keys.
- **GIN** — "contains" queries over composite values: `jsonb` (`@>`, `?`),
  arrays, full-text `tsvector`. Use `jsonb_path_ops` opclass for smaller, faster
  `@>`-only JSONB indexes.
- **GiST** — geometric data, ranges, nearest-neighbor (`<->`), exclusion
  constraints. The engine behind range overlap and `WITHOUT OVERLAPS`.
- **SP-GiST** — space-partitioned data: non-balanced structures, some text/IP
  patterns.
- **BRIN** — huge, naturally-ordered tables (append-only time series). Tiny index
  that stores per-block-range min/max; great when physical order correlates with
  the column (e.g. `created_at` on an append-only log).

## Composite (multicolumn) indexes

Column order is the whole game. An index on `(a, b)` serves `WHERE a = ?`,
`WHERE a = ? AND b = ?`, and `ORDER BY a, b` — but historically **not** `WHERE b
= ?` alone. `[PG18]` skip scan relaxes this: a leading column can be skipped in
more cases, so a single `(a, b)` index now covers some `b`-only queries. Still
design column order for your dominant access pattern; skip scan is a bonus, not
a substitute.

Rule of thumb for order: equality columns first, then the range/sort column
last. For `WHERE tenant_id = ? AND created_at > ? ORDER BY created_at`, index
`(tenant_id, created_at)`.

## Specialized indexes

- **Partial** — index only the rows you query: `CREATE INDEX ON orders (id)
  WHERE status = 'open';`. Smaller, faster, cheaper to maintain. Ideal for
  "hot subset" queries and enforcing conditional uniqueness.
- **Expression** — index a computed value to keep predicates sargable:
  `CREATE INDEX ON users (lower(email));` then query `WHERE lower(email) = $1`.
- **Covering (`INCLUDE`)** — add non-key payload columns so a query is
  index-only: `CREATE INDEX ON orders (customer_id) INCLUDE (total, status);`.
  Avoids heap fetches when those columns are all that's selected.
- **Unique / conditional unique** — `UNIQUE` enforces a constraint and serves
  lookups; a partial unique index enforces "unique among active rows":
  `CREATE UNIQUE INDEX ON users (email) WHERE deleted_at IS NULL;`.

## Build indexes without locking writes

`CREATE INDEX CONCURRENTLY` builds without an `ACCESS EXCLUSIVE` lock, so writes
continue. It's slower, can't run in a transaction block, and leaves an `INVALID`
index if it fails (drop and retry). Always use `CONCURRENTLY` on a live table.
See `migrations-and-ddl-safety.md`.

## Maintenance and health

- Bloat and unused indexes cost writes for nothing. Find unused indexes via
  `pg_stat_user_indexes` (`idx_scan = 0`). Drop with `DROP INDEX CONCURRENTLY`.
- Rebuild bloated indexes with `REINDEX INDEX CONCURRENTLY`.
- Prefer `uuidv7()` `[PG18]` over random UUIDs for PK indexes — time-ordered keys
  reduce B-tree fragmentation and page splits versus `uuidv4()`.

## Review checklist

1. Does an existing index already serve this query (check the DDL)?
2. Is column order right for the dominant predicate + sort?
3. Would a partial/expression/covering index be smaller and sufficient?
4. On a live table, is the build `CONCURRENTLY`?
5. What write cost does this add, and is the read benefit worth it?
