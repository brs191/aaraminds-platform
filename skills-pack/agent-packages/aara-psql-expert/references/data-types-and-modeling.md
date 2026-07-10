# Data types and modeling

Correct types are correctness, not cosmetics. The right type enforces invariants,
enables the right index, and prevents whole classes of bug. Verify actual column
types from the provided DDL before drafting — never assume.

## Numbers

- `numeric`/`decimal` for money and any exact value — never `float`/`double` for
  currency (binary floats can't represent 0.10 exactly). Specify precision/scale
  where it matters: `numeric(12,2)`.
- `integer` vs `bigint` — use `bigint` for surrogate keys that can grow; an
  `int` PK that overflows at 2.1B is a production incident.
- Beware implicit casts in predicates (`bigint_col = '123'`) — they can defeat
  indexes; match the literal's type.

## Text

- `text` is the default; `varchar(n)` only when a length limit is a real domain
  rule. `char(n)` is almost never right (blank-padded). There is no performance
  penalty for `text` over `varchar`.
- Case-insensitive matching: an expression index on `lower(col)` (portable) or
  the `citext` extension.

## Time

- `timestamptz` (timestamp with time zone) for all points in time — it stores
  UTC and converts on display. `timestamp` (without tz) is a frequent bug: it
  silently drops zone context.
- `date`, `time`, `interval` for their exact domains. Compare ranges, not
  `::date` casts, to stay sargable (see `query-optimization.md`).

## Identifiers / keys

- `bigint` identity (`GENERATED ALWAYS AS IDENTITY`) is the simplest surrogate
  key. Prefer it over the legacy `serial`.
- UUID keys: `[PG18]` prefer `uuidv7()` (timestamp-ordered) over random
  `uuidv4()`/`gen_random_uuid()` — random UUIDs fragment the PK B-tree and cause
  page splits; v7 keeps inserts local.

## JSONB

- Use `jsonb` (not `json`) — binary, deduplicated keys, indexable. Use it for
  genuinely schemaless/variable data, **not** as an escape from modeling columns
  you query and constrain.
- Query operators: `->`/`->>` (get), `@>` (contains), `?`/`?|`/`?&` (key
  exists), `jsonb_path_query` / `@@` (JSONPath).
- Index with GIN: `CREATE INDEX ON t USING gin (data jsonb_path_ops)` for
  `@>`-style containment (smaller/faster than default `jsonb_ops`, which also
  supports key-existence).
- Extract hot fields into real columns (or a `[PG18]` virtual generated column)
  when you filter/sort/join on them — a normal B-tree beats digging into JSONB.

## Arrays, ranges, enums

- Arrays: fine for small, bounded lists owned by one row; index with GIN for
  containment. Don't use arrays to dodge a proper join table for many-to-many.
- Ranges (`int4range`, `tstzrange`, …) + `EXCLUDE` constraints enforce
  no-overlap invariants (e.g. no double-booked room). `[PG18]` temporal
  `PRIMARY KEY`/`UNIQUE`/`FOREIGN KEY ... WITHOUT OVERLAPS` make this declarative.
- `enum` for a small, stable, ordered set of values. If values change often or
  carry attributes, use a lookup table + FK instead — altering an enum is more
  disruptive than inserting a row.

## Constraints do modeling work

Push invariants into the schema so no code path can violate them:
- `NOT NULL`, `CHECK`, `UNIQUE`, `FOREIGN KEY` (with the right `ON DELETE`
  action), `EXCLUDE` for overlap/spatial rules.
- Partial unique index for "unique among active rows".
- A constraint is validated, visible, and unbypassable — always prefer it over a
  trigger or application check when it can express the rule.

## Generated columns

- `[PG18]` virtual generated columns (computed on read) are the default and add
  no storage/write cost — good for derived display values.
- `STORED` generated columns persist and can be indexed — use when you filter or
  join on the derived value.

## Review checklist

1. Are money/exact values `numeric`, keys `bigint`/`uuidv7`, timestamps
   `timestamptz`?
2. Is JSONB used for genuinely variable data, with the right GIN opclass, and hot
   fields promoted to columns?
3. Are invariants enforced by constraints rather than triggers/app code?
4. Do types match the provided DDL exactly (verified, not assumed)?
