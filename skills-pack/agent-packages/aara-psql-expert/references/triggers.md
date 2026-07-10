# Triggers

Triggers are powerful and easy to misuse. Default stance: **prefer a constraint,
a generated column, or explicit application logic over a trigger.** Reach for a
trigger when the behavior must be enforced at the data layer regardless of who
writes, and can't be expressed declaratively.

## Anatomy

```sql
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;         -- BEFORE row triggers must return NEW (or NULL to skip)
END; $$;

CREATE TRIGGER trg_set_updated_at
  BEFORE UPDATE ON orders
  FOR EACH ROW
  EXECUTE FUNCTION set_updated_at();
```

## Timing and level

- `BEFORE` — can modify `NEW` (row triggers) or veto the row by returning `NULL`.
  Use for normalization, defaulting, validation.
- `AFTER` — runs after the row change is applied; `NEW`/`OLD` are read-only. Use
  for auditing, cascading writes to other tables, enqueuing work.
- `INSTEAD OF` — only on views; makes a view updatable.
- `FOR EACH ROW` vs `FOR EACH STATEMENT` — statement-level fires once per
  statement (good for bulk audit summaries); row-level fires per affected row.

Return value rules: `BEFORE ROW` returns `NEW` (modified or not) or `NULL` to
skip; `AFTER` and statement-level triggers ignore the return value (return
`NULL`).

## NEW / OLD by event

| Event | NEW | OLD |
|---|---|---|
| INSERT | the new row | null |
| UPDATE | post-image | pre-image |
| DELETE | null | the deleted row |

## Transition tables — the right tool for set-based auditing

For `AFTER` statement-level triggers, `REFERENCING OLD TABLE`/`NEW TABLE` exposes
all affected rows as a set, so you audit a 100k-row update with one insert-select
instead of 100k per-row trigger calls:

```sql
CREATE TRIGGER trg_audit
  AFTER UPDATE ON accounts
  REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
  FOR EACH STATEMENT
  EXECUTE FUNCTION audit_accounts();
-- inside: INSERT INTO audit SELECT ... FROM new_rows JOIN old_rows USING (id);
```

This is dramatically faster than per-row auditing on bulk changes.

## Ordering and recursion

- Multiple triggers of the same timing fire in **alphabetical order by name**.
  Name them with intent (`trg_10_...`, `trg_20_...`) when order matters.
- Triggers can cascade (a trigger writes a table that has its own triggers).
  Guard against infinite recursion; `pg_trigger_depth()` can detect re-entry.
- `WHEN (condition)` filters row triggers cheaply without entering the function:
  `... FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)`.

## Common, defensible patterns

- `updated_at` maintenance (BEFORE UPDATE, set `NEW.updated_at`).
- Immutable audit log (AFTER, transition tables).
- Soft-delete enforcement / denormalized counter maintenance (AFTER, but weigh
  against a materialized view or scheduled job).
- Making a view writable (INSTEAD OF).

## When NOT to use a trigger

- Business rules better expressed as `CHECK`, `FOREIGN KEY`, `UNIQUE`, exclusion,
  or `[PG18]` temporal constraints — declarative constraints are validated,
  visible, and can't be bypassed by a missing trigger.
- Cross-row invariants that a constraint can enforce.
- Anything the application already does reliably and visibly — hidden trigger
  side effects surprise the next engineer and complicate debugging.
- Heavy work in a synchronous trigger that blocks the writing transaction; move
  it to a queue (see `concurrency-and-locking.md`, `SKIP LOCKED`).

## Review checklist

1. Could a constraint or generated column do this instead?
2. Correct timing/level, and correct return value for that timing?
3. For bulk changes, does it use transition tables instead of per-row work?
4. Is ordering deterministic if multiple triggers exist?
5. Any recursion risk? Any hidden performance cost on hot writes?
