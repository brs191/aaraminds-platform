# PostgreSQL / PL-pgSQL knowledge base

Deep reference content for the PSQL expert agent. The core behavioral skill is
`../agent.md`; this directory is the knowledge it routes into. Content targets
**PostgreSQL 18** (current stable as of 2026; latest point release 18.3).
Version-gated features are flagged inline, e.g. `[PG18]`, `[PG17]`.

Freshness note: PL/pgSQL and core SQL semantics are extremely stable across
major versions; performance internals and syntax sugar change more often.
Re-verify version-gated claims against the official docs for the target
server's major version before shipping a draft.

## Routing table

| When the task is about… | Read |
|---|---|
| Writing PL/pgSQL: control flow, exceptions, dynamic SQL, cursors | `plpgsql-language.md` |
| Functions vs procedures, volatility, transaction control, SECURITY DEFINER | `procedures-and-functions.md` |
| Triggers: patterns, ordering, transition tables, when to avoid | `triggers.md` |
| Reading `EXPLAIN`, fixing slow queries, planner behavior | `query-optimization.md` |
| Choosing and designing indexes | `indexing.md` |
| Isolation, locks, deadlocks, upserts, job queues | `concurrency-and-locking.md` |
| Safe schema changes and lock-aware DDL | `migrations-and-ddl-safety.md` |
| Injection-safe dynamic SQL, SECURITY DEFINER hardening, RLS, privileges | `security.md` |
| Types, JSONB, arrays, ranges, generated columns, UUIDs | `data-types-and-modeling.md` |
| Common mistakes and their fixes | `antipatterns.md` |

## Non-negotiable rules (repeated from agent.md)

1. Verify schema from retrieved evidence; never assume column names, types, or
   function signatures.
2. Parameterize all dynamic SQL (`format()` with `%I`/`%L`, or `EXECUTE ... USING`).
   Never concatenate untrusted input.
3. Cite the source (DDL `source_ref` or knowledge-base entry) for every
   non-trivial claim.
4. Advise and draft only — never execute SQL against a database.

## PostgreSQL 18 highlights the agent should know

- Asynchronous I/O subsystem — up to ~3× faster reads on sequential scans,
  bitmap heap scans, and vacuum. Affects tuning advice for large scans.
- `uuidv7()` — timestamp-ordered UUIDs; prefer over random `uuidv4()` for
  primary keys to reduce index bloat and improve locality.
- Virtual generated columns — computed on read, now the default for
  `GENERATED ... AS`. Stored generated columns still available with `STORED`.
- `OLD`/`NEW` in `RETURNING` for `INSERT`/`UPDATE`/`DELETE`/`MERGE` — return
  both pre- and post-image in one statement.
- Temporal constraints — `PRIMARY KEY`/`UNIQUE`/`FOREIGN KEY ... WITHOUT
  OVERLAPS` over range types.
- Skip scan for multicolumn B-tree indexes — a leading column can be skipped in
  more cases, changing some index-design tradeoffs.
