# Indexing (clustered vs nonclustered, INCLUDE, filtered, columnstore)

SQL Server's index model is fundamentally different from Postgres: a table is
either a **heap** or has a **clustered index that IS the table's physical
storage**. Get the clustered key right first; nonclustered indexes hang off it.

## Clustered index — the table's backbone

- The clustered index key determines the physical row order and is the row
  locator stored in every nonclustered index. Choose it to be **narrow,
  static, ever-increasing, and unique** (classically an `IDENTITY`/`bigint` or
  `NEWSEQUENTIALID()` key).
- A wide or random clustered key (e.g. a random `uniqueidentifier` via `NEWID()`)
  bloats every nonclustered index and causes page splits/fragmentation — the
  SQL Server version of the uuidv7-vs-uuidv4 problem. Prefer
  `NEWSEQUENTIALID()` or a sequence if you need a surrogate.
- One clustered index per table. A heap (no clustered index) is usually only
  right for staging/bulk-load tables.

## Nonclustered indexes

- Serve equality/range seeks and ordering not covered by the clustered key.
- Every nonclustered index includes the clustered key as its row locator, so a
  wide clustered key taxes them all.
- **Key column order matters**: equality columns first, then the range/sort
  column (`WHERE TenantId = @t AND CreatedAt > @d ORDER BY CreatedAt` →
  `(TenantId, CreatedAt)`).

## Covering with INCLUDE

Add non-key payload columns so a query is satisfied from the index without a key
lookup back to the clustered index:
```sql
CREATE NONCLUSTERED INDEX IX_Orders_Customer
  ON dbo.Orders (CustomerId) INCLUDE (Total, Status);
```
`INCLUDE` columns aren't part of the key (no ordering/size cost in the b-tree
levels) but eliminate lookups — the fix for a plan showing a **Key Lookup**.

## Filtered indexes

Index only the rows you query — smaller and cheaper:
```sql
CREATE NONCLUSTERED INDEX IX_Orders_Open
  ON dbo.Orders (CreatedAt) WHERE Status = 'Open';
```
Also enforce conditional uniqueness (`WHERE IsDeleted = 0`). Note filtered
indexes have parameterization caveats — the predicate must match.

## Columnstore

`[Enterprise/AzureSQL]` Clustered or nonclustered **columnstore** indexes give
order-of-magnitude gains for analytic/aggregation scans over large tables (batch-
mode execution, high compression). Use for reporting/DW-style tables, not for
OLTP singleton lookups. A nonclustered columnstore on an OLTP table enables
real-time operational analytics.

## Missing-index and health signals

- `sys.dm_db_missing_index_details` / the missing-index warnings in a plan
  suggest candidates — treat as hints, not gospel (they ignore write cost and
  overlap).
- `sys.dm_db_index_usage_stats` finds unused indexes (writes with no seeks/
  scans) to drop.
- Fragmentation via `sys.dm_db_index_physical_stats`; `REORGANIZE` (light) vs
  `REBUILD` (heavy; `ONLINE = ON` `[Enterprise/AzureSQL]` avoids blocking).
- Fill factor: lower it only for indexes with heavy mid-key inserts causing page
  splits; don't blanket-set it.

## Review checklist

1. Is the clustered key narrow, unique, static, ever-increasing?
2. Does the query show a Key Lookup that `INCLUDE` would remove?
3. Right key column order for the dominant predicate + sort?
4. Would a filtered index be smaller and sufficient?
5. Analytic scan on a big table → columnstore candidate?
6. On a live table, is the rebuild `ONLINE = ON`?
