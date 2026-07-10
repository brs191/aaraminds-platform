# Common PostgreSQL antipatterns and their fixes

A checklist of the mistakes that show up most in real PL/pgSQL and SQL, each with
the fix. The agent should scan every draft against this list.

## Correctness

- **Assuming schema.** Inventing column names/types instead of verifying from
  DDL. → Retrieve schema; cite it; ask if missing.
- **`SELECT INTO` without `STRICT`.** Silently takes the first of many rows. →
  Add `STRICT`, or an explicit `ORDER BY ... LIMIT 1` with intent.
- **Read-then-write lost updates.** Compute `x-1` in app, write it back. →
  Atomic `SET stock = stock - 1 WHERE ...`, or `FOR UPDATE`.
- **Catching `unique_violation` in a loop** to emulate upsert. → `INSERT ... ON
  CONFLICT`.
- **`float` for money.** Rounding errors. → `numeric`.
- **`timestamp` instead of `timestamptz`.** Silent timezone loss. → `timestamptz`.
- **`NULL` mishandling.** `= NULL` never matches; `NOT IN (subquery with NULLs)`
  returns no rows. → `IS [NOT] NULL`, `IS DISTINCT FROM`, `NOT EXISTS`.

## Performance

- **Non-sargable predicates.** `WHERE lower(email)=$1` / `created_at::date=$1`. →
  Expression index, or rewrite as a range.
- **N+1 in PL/pgSQL.** A `FOR` loop running one query per row. → Single
  set-based statement or `RETURN QUERY`.
- **`SELECT *` in production code.** Fetches unused columns, breaks index-only
  scans, and breaks silently when columns change. → List needed columns.
- **`OFFSET` for deep pagination.** `OFFSET 100000` scans and discards. →
  Keyset/seek pagination (`WHERE (created_at, id) < ($1, $2) ORDER BY ... LIMIT n`).
- **Count-everything.** `SELECT count(*)` on a huge table for a UI badge. →
  Approximate via `pg_class.reltuples`, or maintain a counter.
- **Indexing everything.** Every index taxes writes. → Index measured query
  patterns only; drop unused (`pg_stat_user_indexes`).
- **Over-wide GIN / wrong opclass on JSONB.** → `jsonb_path_ops` for `@>`-only;
  promote hot fields to columns.

## Concurrency and operations

- **`ALTER TABLE` that rewrites a hot table** or takes `ACCESS EXCLUSIVE`
  unguarded. → `NOT VALID` + `VALIDATE`, `CREATE INDEX CONCURRENTLY`, batched
  backfills, `SET lock_timeout`.
- **`ADD COLUMN ... DEFAULT <volatile>`** forces a full rewrite. → Add nullable,
  backfill in batches, then set default.
- **Long transactions.** Hold locks and bloat, block vacuum. → Keep
  transactions short; don't hold locks across external calls.
- **Inconsistent lock order** across code paths → deadlocks. → Lock in a fixed
  order (e.g. ascending id).

## Security

- **String-concatenated dynamic SQL.** → `format()` `%I` + `USING` binds.
- **`SECURITY DEFINER` without pinned `search_path`.** Privilege escalation. →
  `SET search_path = pg_catalog, public` and minimal grants.
- **Runtime role owns objects / has `ALL` / `BYPASSRLS`.** → Least privilege;
  separate owner and runtime roles.
- **Secrets or PII in code, `RAISE` text, or comments.** → Never; redact.

## Design

- **JSONB as a schema-avoidance dump** for fields you filter and constrain. →
  Model them as columns with constraints.
- **Triggers for rules a constraint can enforce.** Hidden, bypassable if the
  trigger is dropped. → `CHECK`/`FK`/`UNIQUE`/`EXCLUDE`/temporal constraints.
- **Arrays instead of a join table** for many-to-many. → Junction table.
- **`enum` for a frequently-changing set.** Altering enums is disruptive. →
  Lookup table + FK.

## The agent's habit

For every draft, before returning it: run this list, state which items were
checked, and surface any it can't verify (e.g. "index benefit unconfirmed — needs
`EXPLAIN` on the target server"). Honesty about unverified claims is part of the
deliverable.
