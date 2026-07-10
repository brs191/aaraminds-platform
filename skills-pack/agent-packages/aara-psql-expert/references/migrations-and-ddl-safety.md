# Safe schema migrations and lock-aware DDL

The failure mode isn't wrong SQL — it's correct SQL that takes an
`ACCESS EXCLUSIVE` lock on a hot table and stalls the whole application while it
rewrites. Migration drafts must be **lock-aware** and, wherever possible,
**reversible**.

## Golden rules

1. Prefer additive, backward-compatible changes; deploy in phases (expand →
   migrate → contract).
2. Always set a `lock_timeout` (and often `statement_timeout`) so a blocked DDL
   fails fast instead of queuing behind — and blocking — every query:
   `SET lock_timeout = '3s';`
3. Never combine a long-running data change with a schema change in one
   transaction that holds a strong lock.

## What locks, what rewrites

**Cheap (metadata-only, brief lock):**
- `ADD COLUMN` with no volatile default — instant in modern Postgres (a
  non-volatile default is stored as metadata, no rewrite).
- `DROP COLUMN` — marks the column dropped; no rewrite.
- Adding a `CHECK`/`FOREIGN KEY` as `NOT VALID`, then validating separately.
- `ALTER COLUMN ... DROP NOT NULL`.

**Expensive (table rewrite, holds ACCESS EXCLUSIVE):**
- `ADD COLUMN ... DEFAULT <volatile>` (e.g. `now()`, `gen_random_uuid()`).
- Changing a column type in a way that requires reformatting (`int` → `text`,
  most `USING` casts).
- `ADD COLUMN ... GENERATED ALWAYS AS ... STORED` (must compute every row).
  `[PG18]` **virtual** generated columns compute on read and avoid the rewrite —
  prefer virtual unless you need the stored value indexed.
- `SET NOT NULL` historically scanned the whole table; it can be made cheap by
  first adding a validated `CHECK (col IS NOT NULL) NOT VALID` then `VALIDATE`,
  then `SET NOT NULL` (which can use the proven constraint).

## Safe patterns

**Add a NOT NULL column with a default (no long lock):**
```sql
ALTER TABLE t ADD COLUMN status text;               -- fast, nullable
-- backfill in batches (see below), then:
ALTER TABLE t ALTER COLUMN status SET DEFAULT 'new';
ALTER TABLE t ADD CONSTRAINT t_status_nn CHECK (status IS NOT NULL) NOT VALID;
ALTER TABLE t VALIDATE CONSTRAINT t_status_nn;      -- scans without ACCESS EXCLUSIVE
```

**Add a foreign key without a long lock:**
```sql
ALTER TABLE child ADD CONSTRAINT fk FOREIGN KEY (parent_id)
  REFERENCES parent(id) NOT VALID;                  -- brief lock, no full scan
ALTER TABLE child VALIDATE CONSTRAINT fk;           -- scans under a weaker lock
```

**Build an index on a live table:**
```sql
CREATE INDEX CONCURRENTLY idx ON t (col);           -- no write blocking
-- cannot run inside a transaction; leaves INVALID index on failure — drop & retry
```

**Batched backfill (bound lock/WAL, don't rewrite in one shot):**
```sql
-- in a procedure or app loop, commit between batches:
UPDATE t SET status = 'new'
WHERE id IN (SELECT id FROM t WHERE status IS NULL LIMIT 10000);
```

## Expand → migrate → contract (zero-downtime column change)

1. **Expand**: add the new column/table; write to both old and new (trigger or
   app dual-write).
2. **Migrate**: backfill the new column in batches; verify.
3. **Contract**: switch reads to new; drop the old column in a later deploy.

Never rename-and-pray in one step while the app expects the old name.

## Reversibility

Every migration draft should state its rollback. Additive changes are trivially
reversible (drop the addition). Destructive changes (drop column, type change)
need an explicit, tested down-path or an expand/contract sequence so the previous
app version still works during deploy.

## Review checklist

1. Does any statement rewrite the table or take `ACCESS EXCLUSIVE` on a hot one?
2. Is `lock_timeout` set so a blocked DDL fails fast?
3. Are constraints added `NOT VALID` then validated separately?
4. Are indexes built `CONCURRENTLY`?
5. Is the change backward-compatible for the currently-deployed app version?
6. Is the rollback explicit?
