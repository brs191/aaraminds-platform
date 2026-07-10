# PL/pgSQL language reference

PL/pgSQL is PostgreSQL's default procedural language. Use it when set-based SQL
can't express the logic — never as a first resort. A single well-written query
usually beats a loop. Reach for PL/pgSQL for control flow, per-row side effects
that can't be expressed as a set operation, exception handling, and dynamic SQL.

## Block structure

```sql
CREATE OR REPLACE FUNCTION f(p_id bigint)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
  v_count integer := 0;          -- initialized at block entry
BEGIN
  -- statements
  RETURN v_count;
EXCEPTION
  WHEN no_data_found THEN
    RAISE;                        -- re-raise unless you can truly recover
END;
$$;
```

`DECLARE` is optional. Variables are block-scoped; inner blocks can shadow outer
ones (avoid it — it hides bugs). Always dollar-quote the body (`$$` or `$func$`)
so embedded quotes survive.

## Variables and assignment

- Assign with `:=` (or `=`). `SELECT ... INTO var` assigns from a query;
  `STRICT` makes zero or many rows raise instead of silently taking the first:

```sql
SELECT price INTO STRICT v_price FROM products WHERE id = p_id;
-- raises NO_DATA_FOUND / TOO_MANY_ROWS instead of returning a wrong row
```

- `%TYPE` and `%ROWTYPE` bind a variable's type to a column or row so it tracks
  schema changes: `v_price products.price%TYPE;`, `r products%ROWTYPE;`.
- `CONSTANT` for immutables; `NOT NULL` to reject null assignment.

## Control flow

`IF / ELSIF / ELSE / END IF`, `CASE`, and loops: `LOOP`, `WHILE`, `FOR`.

```sql
FOR r IN SELECT id, qty FROM order_lines WHERE order_id = p_id LOOP
  -- r.id, r.qty
END LOOP;

FOR i IN 1..10 BY 2 LOOP ... END LOOP;         -- integer range
FOREACH x IN ARRAY p_arr LOOP ... END LOOP;    -- iterate an array
```

Prefer `RETURN QUERY` / a single statement over a `FOR` loop that runs one query
per row — the loop is often an accidental N+1.

## Returning data

- Scalar: `RETURNS integer` + `RETURN expr;`
- Set: `RETURNS SETOF t` or `RETURNS TABLE(col type, ...)`:

```sql
CREATE FUNCTION recent_orders(p_customer bigint)
RETURNS TABLE(order_id bigint, total numeric)
LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY
    SELECT o.id, o.total FROM orders o
    WHERE o.customer_id = p_customer
    ORDER BY o.created_at DESC;
END; $$;
```

`RETURN NEXT expr;` appends one row at a time (use when rows are computed
individually); `RETURN QUERY` streams a query's result. `RETURN QUERY` is faster
for query-shaped output.

## PERFORM

Run a query for side effects and discard the result: `PERFORM fn(x);`. Using
`SELECT fn(x);` inside PL/pgSQL is an error unless you capture the result.

## Exception handling

An `EXCEPTION` block starts a subtransaction (savepoint). It has a cost — do not
wrap tight loops in per-iteration exception blocks. Catch specific conditions,
not the world:

```sql
BEGIN
  INSERT INTO t(id) VALUES (p_id);
EXCEPTION
  WHEN unique_violation THEN
    -- handle the known case
  WHEN OTHERS THEN
    RAISE;    -- never swallow OTHERS silently
END;
```

`GET STACKED DIAGNOSTICS` inside a handler exposes `RETURNED_SQLSTATE`,
`MESSAGE_TEXT`, `PG_EXCEPTION_DETAIL`, `CONSTRAINT_NAME`, etc. Prefer
`INSERT ... ON CONFLICT` over catching `unique_violation` in a loop — it's
cheaper and race-free.

## RAISE

```sql
RAISE EXCEPTION 'order % not found', p_id USING ERRCODE = 'no_data_found';
RAISE WARNING 'slow path taken for %', p_id;
RAISE NOTICE 'debug: count=%', v_count;
```

`%` is the placeholder. Use `USING ERRCODE`, `DETAIL`, `HINT`, `CONSTRAINT` to
produce structured, catchable errors. Never build the message by concatenating
untrusted input into the format string.

## Dynamic SQL — the injection-critical path

Use dynamic SQL only when identifiers or structure aren't known until runtime.
**Always** quote with `format()` and bind values with `USING`:

```sql
EXECUTE format('SELECT count(*) FROM %I WHERE %I = $1', v_table, v_col)
  INTO v_n USING p_value;
```

- `%I` quotes an identifier (table/column) safely.
- `%L` quotes a literal — but prefer `$1` + `USING` for values so the planner
  can cache and there's no literal-escaping risk.
- Never do `EXECUTE 'SELECT ... ' || p_value`. That is the classic injection.

See `security.md` for the full injection-safe pattern and `SECURITY DEFINER`
hardening.

## GET DIAGNOSTICS

```sql
GET DIAGNOSTICS v_rows = ROW_COUNT;   -- rows affected by the last statement
```

## `GET DIAGNOSTICS` vs `FOUND`

`FOUND` is a boolean set by `SELECT INTO`, `PERFORM`, `UPDATE`, `INSERT`,
`DELETE`, and `FOR` loops. Use it for quick "did anything match" checks;
`ROW_COUNT` when you need the number.

## Cursors

Explicit cursors (`DECLARE ... CURSOR`, `OPEN`, `FETCH`, `CLOSE`) are for
streaming very large result sets or returning a `refcursor` to the caller. For
ordinary iteration, a `FOR ... IN SELECT` implicit cursor is simpler and just as
fast. Don't reach for explicit cursors unless you need `FETCH`-level control or a
cursor handoff.

## Common correctness traps

- `SELECT INTO` without `STRICT` silently takes the first row of many — usually a
  bug. Add `STRICT` or `ORDER BY ... LIMIT 1` with intent.
- Variable name equal to a column name makes the column ambiguous; prefix
  variables (`v_`) and parameters (`p_`).
- Per-row `EXCEPTION` blocks in a hot loop are slow (each is a savepoint).
- Building queries by string concatenation — always `format()` + `USING`.
