# Data types and modeling (T-SQL)

Correct types are correctness. Verify actual column types from the provided DDL
before drafting — never assume.

## Numbers

- `decimal(p,s)` / `numeric` for money and exact values. **Avoid `money`** — it
  has only 4 decimal places and rounding quirks in division/aggregation; use
  `decimal(19,4)` instead.
- Never `float`/`real` for currency (binary approximation).
- `int` overflows at ~2.1B; use `bigint` for surrogate keys that can grow.

## Text — Unicode matters

- `nvarchar` (Unicode) vs `varchar` (single-byte). Mixing them in predicates
  causes implicit conversion and index scans (`nvarchar` param vs `varchar`
  column). Standardize on `nvarchar` for user text unless you have a proven
  reason and matching types everywhere.
- `varchar(max)`/`nvarchar(max)` for >8000 bytes, but keep large blobs out of hot
  rows where possible.
- Collation affects comparison/sorting and case sensitivity; be explicit when it
  matters, and beware collation mismatches across databases/columns.

## Dates and time

- `datetime2` over legacy `datetime` — wider range, higher precision, and
  standards-aligned. Use `datetimeoffset` when you need the UTC offset preserved.
- `date`/`time` for their exact domains. Compare ranges, not `CONVERT(date, col)`
  wraps, to stay sargable.

## Keys and identity

- `IDENTITY` or a `SEQUENCE` for surrogate keys; `SEQUENCE` when you need the
  value before insert or shared across tables.
- GUID keys: prefer **`NEWSEQUENTIALID()`** (as a column default) over `NEWID()`
  — random GUIDs as a clustered key fragment the table and every nonclustered
  index (page splits). If you must use `NEWID()` GUIDs, don't cluster on them.

## JSON and vector (SQL Server 2025)

- **Native `JSON` type** `[SQL2025]` — stop using `NVARCHAR(MAX)` for queried
  JSON. Functions: `JSON_VALUE`, `JSON_QUERY`, `JSON_MODIFY`, `JSON_CONTAINS`,
  `ISJSON`, `OPENJSON`, `JSON_OBJECT_AGG`, `JSON_ARRAY_AGG`. Promote hot JSON
  fields to computed/persisted columns and index those for filtering.
- **`VECTOR` type + vector search** `[SQL2025]` for embeddings; native to T-SQL
  with in-DB model integration. `[VERIFY]` availability per Azure SQL tier.

## Constraints do modeling work

Push invariants into the schema: `NOT NULL`, `CHECK`, `UNIQUE`, `FOREIGN KEY`
(with the right `ON DELETE`), and unique **filtered** indexes for "unique among
active rows" (`WHERE IsDeleted = 0`). A constraint is validated and unbypassable
— prefer it over a trigger or app check when it can express the rule.

## Computed columns

`AS (expr)` computed columns (optionally `PERSISTED`) derive values; `PERSISTED`
+ an index makes them filterable. Use to extract and index a hot JSON field or a
normalized form (e.g. `PERSISTED` `LOWER(Email)` for case-insensitive lookups
with a matching collation).

## Review checklist

1. Money as `decimal(19,4)` (not `money`/`float`), keys `bigint`/`NEWSEQUENTIALID`,
   dates `datetime2`/`datetimeoffset`?
2. Unicode types consistent to avoid implicit conversions?
3. Queried JSON as the native `JSON` type with hot fields promoted/indexed?
4. Invariants enforced by constraints rather than triggers/app code?
5. Types match the provided DDL exactly (verified, not assumed)?
