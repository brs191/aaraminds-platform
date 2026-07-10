# Functions vs procedures, volatility, and security context

## Function or procedure — pick with intent

**Functions** (`CREATE FUNCTION`) return a value and run inside the caller's
transaction. They cannot `COMMIT`/`ROLLBACK`. Use them for computation and
set-returning logic — the overwhelming majority of cases.

**Procedures** (`CREATE PROCEDURE`, invoked with `CALL`) return nothing (except
via `INOUT` params) and **can** manage transactions with `COMMIT` and `ROLLBACK`
between statements. Use a procedure only when you genuinely need transaction
control mid-body — e.g. a batch job that commits every N rows to bound lock
duration and WAL. If you don't need `COMMIT` inside, write a function.

```sql
CREATE PROCEDURE purge_batches()
LANGUAGE plpgsql AS $$
BEGIN
  LOOP
    DELETE FROM events WHERE id IN (SELECT id FROM events WHERE ts < now() - interval '90 days' LIMIT 10000);
    EXIT WHEN NOT FOUND;
    COMMIT;   -- bound transaction size; only legal in a procedure
  END LOOP;
END; $$;
```

## Argument modes

`IN` (default), `OUT`, `INOUT`, and `VARIADIC`. `OUT`/`INOUT` params define the
result shape for procedures and multi-value functions. Prefer named args at call
sites for readability: `f(p_id => 42)`.

## Volatility — get this right or the planner will mislead you

Every function has a volatility class. It controls caching and when the function
may be evaluated:

- `IMMUTABLE` — same inputs always yield the same output, no DB access (e.g.
  pure math, `lower(text)`). The planner can constant-fold and use the function
  in index expressions. Marking something IMMUTABLE that isn't (e.g. reads a
  table, or depends on `search_path`/timezone) causes **wrong results** and stale
  index entries.
- `STABLE` — consistent within a single statement; may read the DB but doesn't
  modify it (e.g. `now()`-based lookups, most read-only functions). Required for
  a function used in an index scan condition.
- `VOLATILE` (default) — may return different results every call and/or have side
  effects (e.g. `random()`, `nextval()`, anything that writes). The planner
  evaluates it per row and won't optimize it away.

Mark functions as tightly as is *truthfully* possible — but never over-promise.
When unsure, `STABLE` for read-only, `VOLATILE` for anything that writes.

## Parallel safety

`PARALLEL SAFE | RESTRICTED | UNSAFE`. Mark read-only, side-effect-free functions
`PARALLEL SAFE` so they don't disable parallel query plans. Anything that writes,
uses sequences, or touches session state is `UNSAFE` (the default is UNSAFE).

## SECURITY DEFINER vs INVOKER

- `SECURITY INVOKER` (default) — runs with the caller's privileges.
- `SECURITY DEFINER` — runs with the *owner's* privileges. Powerful and
  dangerous; it's the SQL equivalent of setuid. If you use it, you **must** pin
  `search_path` to prevent search-path injection:

```sql
CREATE FUNCTION admin_reset(p_id bigint)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public   -- pin it; do not trust the caller's path
AS $$ ... $$;
```

Without a pinned `search_path`, a caller can create a malicious `public.now()`
(or similar) that your definer function then runs with elevated rights. See
`security.md`.

## Cost and rows hints

`COST` and `ROWS` hint the planner about a function's expense and set-returning
cardinality. Only tune these when a plan is demonstrably wrong because of a bad
function estimate — they're a scalpel, not a default.

## Overloading and naming

PostgreSQL overloads by argument types. Keep signatures unambiguous; avoid
overloads that differ only by numeric type (`int` vs `bigint`) — implicit casts
make the resolution surprising. Schema-qualify calls in `SECURITY DEFINER`
bodies.

## Draft review checklist for a function/procedure

1. Is a function the right tool, or does this need a procedure's `COMMIT`?
2. Is the volatility class truthful and as tight as correct?
3. If `SECURITY DEFINER`: is `search_path` pinned and are privileges minimal?
4. Are all schema references verified against provided DDL?
5. Is dynamic SQL parameterized (`format()` + `USING`)?
6. Does it return the documented shape (`RETURNS TABLE`/`SETOF`) exactly?
