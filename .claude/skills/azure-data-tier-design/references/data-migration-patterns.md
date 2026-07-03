# Data Migration Patterns

## Purpose

Schema and engine migrations are the highest-risk data-tier work. This reference covers zero-downtime schema change (expand/contract), in-engine migrations (data backfill, dual-write pitfalls), and cross-engine migrations (Postgres → Cosmos, Cosmos partition-key re-shape, Mongo flavour swap). The default posture is **brownfield with no downtime window** — if you have a downtime window, you don't need most of this.

## The expand/contract pattern

Every backwards-incompatible schema change follows this sequence. There are no shortcuts.

```
1. Expand    — add the new schema element, nullable / additive
2. Dual-write— writers fill old AND new
3. Backfill  — fill new for historical data
4. Dual-read — readers read new, fall back to old (verification window)
5. Cut read  — readers read new only
6. Stop write— writers fill new only
7. Contract  — remove old schema element
```

Each step is independently deployable, independently revertible, and independently observable. Skipping steps is the source of most migration outages.

### Worked example — renaming `customer_email` to `email` on Postgres orders table

Current state: `orders.customer_email VARCHAR(255) NOT NULL`. Goal: rename to `email`.

You cannot `ALTER COLUMN ... RENAME` while the application uses the column name. The migration:

1. **Expand**: `ALTER TABLE orders ADD COLUMN email VARCHAR(255) NULL` (nullable initially).
2. **Dual-write**: app version N+1 writes both `customer_email` and `email` on every insert/update. Deploy this. Verify both are populated for new rows.
3. **Backfill**: `UPDATE orders SET email = customer_email WHERE email IS NULL` — run in batches with `LIMIT` if the table is large; throttle to avoid replication lag spikes.
4. **Dual-read**: app version N+2 reads `email`, falls back to `customer_email` if `email IS NULL`. Deploy. Monitor fallback-rate metric — should drop to zero after backfill.
5. **Cut read**: app version N+3 reads `email` only. Deploy after fallback metric is zero for 7 days.
6. **Stop write**: app version N+4 stops writing `customer_email`. Deploy.
7. **Contract**: `ALTER TABLE orders DROP COLUMN customer_email` after 30 days (delete-safe window for rollback).

Total elapsed: 4–8 weeks for a column on a busy table. This is the cost of zero downtime.

## Backfill discipline

Bulk `UPDATE` on a large table:

- Blocks vacuum, bloats the WAL, can stall replication.
- Take in batches: `UPDATE ... WHERE id BETWEEN $start AND $end LIMIT 1000`.
- Sleep between batches if replication lag rises.
- Run during low-traffic window even if the migration is "zero downtime" — backfill load is real.

Postgres-specific:
```sql
DO $$
DECLARE
  v_min bigint := 0;
  v_max bigint;
BEGIN
  SELECT MAX(id) INTO v_max FROM orders;
  WHILE v_min < v_max LOOP
    UPDATE orders SET email = customer_email
    WHERE id BETWEEN v_min AND v_min + 999 AND email IS NULL;
    v_min := v_min + 1000;
    PERFORM pg_sleep(0.05);  -- breathe
  END LOOP;
END $$;
```

For Cosmos: use a Change Feed processor or a one-shot reader script that iterates the container with continuation tokens; throttle RU consumption.

## Dual-write is rarely safe — prefer outbox + consumer

Dual-write means the application writes to two stores in one operation:

```python
# DANGER
db.orders.insert(order)
cosmos.orders.upsert(order)
```

This is not transactional. If the first succeeds and the second fails, the two stores diverge silently. Over time, drift accumulates; the "fix" is a reconciliation job that papers over the design defect.

**Safe alternatives:**

1. **Transactional outbox + async consumer.** Write the order and an outbox row in one DB transaction. A consumer reads the outbox and writes to the second store. The outbox is the audit trail. See `microservices-data-architecture` and `../../microservices-data-architecture/references/patterns/transactional-outbox.md`.
2. **Change feed → consumer** (Cosmos source). Cosmos emits a change feed; an Azure Function consumer fans out writes. The change feed is durable and replayable.
3. **Logical replication** (Postgres source). Postgres logical replication slots stream changes to a consumer that writes to the secondary store. Use `pglogical` or the native `pgoutput` plugin with a consumer like Debezium.

If you must dual-write for some short-term reason: write to the *secondary* first, then the primary. If the secondary fails, you know about it before the primary commits. (Most app code does this backwards.)

## Cross-engine migration — Postgres → Cosmos via outbox

The CQRS / read-model setup. Source of truth stays in Postgres; a denormalized projection in Cosmos powers reads.

```
                       outbox row
                          │
   Postgres ──────────────▼─────────────→ Azure Function consumer
   (write)                                          │
                                                    ▼
                                              Cosmos DB
                                              (read model)
```

Sequence:

1. Add outbox table to Postgres: `outbox(id, aggregate_id, event_type, payload jsonb, created_at, processed_at)`.
2. Wrap business writes in a transaction that updates both the business table and the outbox.
3. Deploy Azure Function with Postgres trigger (or polling consumer) that reads `outbox WHERE processed_at IS NULL`, writes to Cosmos, marks processed.
4. Backfill: read all existing business rows, write to Cosmos directly (one-shot, not via outbox).
5. Run dual-read for 2–4 weeks; verify parity with a sampling job.
6. Cut reads to Cosmos. Postgres remains source of truth.

Failure recovery: if the Function falls behind, alert on `MAX(now() - created_at) FROM outbox WHERE processed_at IS NULL > 60s`. The outbox is durable — reprocessing is safe (idempotent consumer; see `../../microservices-data-architecture/references/patterns/idempotent-consumer.md`).

## Cosmos partition-key re-shape

You discover the partition key is wrong (hot partition, query mismatch). The container can't be re-partitioned in place.

```
1. Create new container with the new partition key
2. Backfill via Change Feed:
     old_container.change_feed → consumer → new_container
3. Switch writers to new_container (dual-write window optional)
4. Verify new_container has full history
5. Switch readers to new_container
6. Stop writes to old_container
7. After retention window, delete old_container
```

Cost: storage doubles during the migration. RU cost on both containers during dual-write. Plan ~2× normal Cosmos spend for the migration window.

The Change Feed reads only inserts/updates by default. If you need to capture deletes too, enable full-fidelity change feed (preview/GA depending on region) before starting.

## Mongo flavour swap (RU-based Cosmos for Mongo → vCore)

Mongo-to-Mongo, both on Azure, same wire protocol. Sounds easy. The gotcha is RU-based has quirks (RU errors, partial aggregation support) and vCore is closer to real Mongo.

Sequence:

1. Stand up vCore cluster, configure VNet/peering and Entra auth.
2. Use `mongodump` from RU-based source, `mongorestore` to vCore. For large datasets, use **Azure Database Migration Service** or Mongo's `mongomirror` for ongoing sync.
3. Switch readers to vCore (read-only window — easy).
4. Switch writers to vCore (cut over with brief downtime, or use change-stream replay).
5. Decommission RU-based after a verification period.

The RU-based source has unusual error semantics under high RU pressure. If `mongomirror` slows down, raise RU throughput temporarily during migration.

## Postgres major-version upgrade

`pg_upgrade` does the heavy lifting; Azure exposes this via portal / CLI.

Sequence:

1. Take a backup snapshot.
2. Restore the snapshot to a new Flexible Server instance running the target major version (e.g., 14 → 16).
3. Run the application test suite against the new instance.
4. Switch application to point at the new instance during a maintenance window.
5. Decommission old instance after 30 days.

In-place upgrade is also offered by Azure (Major Version Upgrade) — supports 11→16 paths as of 2025. Test in staging; in-place has occasionally surprised users with extension incompatibility.

**Always upgrade staging first**, run the full app test suite against it for a week, then production. Don't `pg_upgrade` directly in prod on a Monday.

## Rollback plan — required at every step

Every migration step needs a written rollback:

| Step | Rollback |
|---|---|
| Expand (add column) | Drop column (safe if no writes yet) |
| Dual-write | Stop writing new field (deploy previous app version) |
| Backfill | Stop the backfill job; the new field is just incomplete |
| Dual-read | Read old field again (deploy previous app version) |
| Cut read | Re-enable old-field fallback (deploy previous app version) |
| Stop write | Resume dual-write (deploy previous app version) |
| Contract | **No rollback** — the column is gone; restore from PITR if needed |

Contract is the only irreversible step. Wait long enough that you trust it before contracting.

## Tools

| Tool | When |
|---|---|
| **Azure Database Migration Service (DMS)** | Postgres → Postgres, Postgres → Cosmos for Postgres, Mongo → Cosmos for Mongo at scale |
| **`pg_dump` / `pg_restore`** | Postgres exports for one-shot moves; not for large prod-to-prod |
| **`pglogical` / Debezium** | Continuous Postgres CDC for cross-engine streaming |
| **`mongodump` / `mongorestore`** | Mongo exports; small datasets |
| **`mongomirror`** | Live Mongo cluster-to-cluster sync; Atlas-recommended |
| **Cosmos Data Migration Tool / `azdata`** | Cosmos-to-Cosmos bulk copies, JSON file → Cosmos |
| **Change Feed processor library** | Custom Cosmos-to-anywhere stream |

## Anti-patterns

- **Renaming a column in one deploy.** Application reads/writes the old name during the deploy window. 100% outage.
- **Dual-write with no atomicity.** Stores diverge; reconciliation jobs forever.
- **Bulk update without batching.** Replication lag spikes; possible primary-replica failover.
- **Schema change without rollback plan.** First production failure is the time you discover you can't undo it.
- **Skipping dual-read verification.** You don't know if backfill is complete until you compare reads against both sources.
- **Contracting too early.** The old column / field is gone; rollback now requires PITR. Wait at least 30 days.
- **Doing schema migrations during business hours on Friday.** Even zero-downtime migrations have surprise modes.

## Verification questions

1. Is every migration step independently deployable and revertible (except the final contract)?
2. Has the backfill been batched and throttled, with replication-lag monitoring?
3. If dual-write is used: is there an outbox or change feed as the safe alternative considered first?
4. For cross-engine migrations: is there a verification job comparing reads from both sources?
5. Is the rollback plan written down for each step?
6. Has the migration been tested in staging end-to-end before touching prod?

## What to read next

- `postgres-on-azure.md` — Flexible Server PITR, replication lag dashboards
- `cosmos-db-design.md` — change feed for migration sources
- `mongodb-on-azure.md` — flavour swap specifics
- `../../microservices-data-architecture/references/patterns/transactional-outbox.md` — the safe alternative to dual-write
- `../../microservices-data-architecture/references/patterns/idempotent-consumer.md` — required for replayable consumers
- `azure-microservices-observability` skill — backfill and replication-lag dashboards
