# Schema Design — Cross-Engine

## Purpose

Schema design happens before any of the operational topics in this skill matter. Wrong schema = wrong queries = wrong indexes = wrong everything. This reference covers normalization vs denormalization, embed vs reference for document stores, key strategy, audit columns, soft deletes, multi-tenancy, schema evolution, and constraint placement. Coverage spans Postgres / Azure SQL / MySQL for relational and Cosmos / Mongo for document.

## The first question — what shape is this data?

Before normalizing or denormalizing or embedding or referencing, name the shape:

1. **Access pattern.** What are the top 5–10 queries? Already named in `engine-selection.md`.
2. **Cardinality of relationships.** 1:1, 1:N, N:M.
3. **Update frequency by entity.** Some entities update often (order status); some rarely (product catalog); some never (audit log).
4. **Read vs write ratio per entity.**
5. **Atomicity boundaries.** What must update together?

Schema follows from these. Without naming them, schema design is taste.

## Normalization in relational engines

Default to **3NF** (third normal form): no transitive dependencies, every non-key attribute depends on the whole key. Right starting point for OLTP.

When to denormalize:

- **Read-heavy queries that always join the same way.** If 95% of `orders` reads also need `customer_name`, materialize it on the order row. Trade-off: customer rename requires update across all order rows.
- **Reporting / analytical shapes** — prefer CQRS to a denormalized OLTP table (see `microservices-data-architecture` + `analytical-engines.md`).
- **Performance-critical hot paths** where joins are measurably the bottleneck (not just theoretically).

Don't pre-denormalize for "scale" without evidence. Normalized schemas with good indexes handle most OLTP load.

## Document model — embed vs reference

For Mongo and Cosmos, the equivalent choice is **embed vs reference**.

| Pattern | Use when |
|---|---|
| **Embed** (subdocument or array) | Bounded; data accessed together; subdocuments don't need independent queries |
| **Reference** (foreign-style key) | Unbounded; subdocuments queried independently; many writers update children |

### Embed example

```json
{
  "id": "ord-123",
  "customer": "cust-456",
  "lineItems": [
    {"sku": "ABC", "qty": 2, "price": 9.99},
    {"sku": "XYZ", "qty": 1, "price": 19.99}
  ]
}
```

Embed when:
- Line items are read every time the order is read
- Line items are bounded (orders < 100 items)
- Atomic update to order + items needed (single-document atomicity)

### Reference example

```
User:     { "id": "user-1", "email": "..." }
Activity: { "id": "act-1", "userId": "user-1", "event": "login", "at": "..." }
```

Reference when:
- Activities are unbounded
- Activities queried independently across users
- Loading the user shouldn't load all activities

### The unbounded-array trap

Embedded arrays grow without bound. Mongo's 16MB document limit and Cosmos's per-item RU cost both penalize this. Detection: a document type whose embedded array grows over time and you can name a customer with > 1000 entries.

Fix: extract to a separate collection. The migration is non-trivial; design correctly from the start.

## Key strategy — surrogate vs natural

| Choice | When |
|---|---|
| **`BIGSERIAL` / `BIGINT AUTO_INCREMENT`** | Default for relational; ordered insertions; 8 bytes |
| **UUIDv4 (random)** | Distributed-write where coordination-free generation matters; **bad for InnoDB clustered PK** |
| **UUIDv7 (time-ordered)** | UUIDv4 use case without the InnoDB clustering penalty |
| **Natural keys** (email, order_number) | Only if guaranteed unique, immutable, and you accept FK ripple on any rename |
| **Cosmos `id` + partition key** | Cosmos requires both; design them together |

**Do not use random UUIDs as InnoDB clustered PK.** Random insertion scatters across the clustered index; page splits and write amplification destroy performance; secondary indexes (which contain the PK) bloat. Either `BIGINT AUTO_INCREMENT` or UUIDv7.

Postgres: `BIGSERIAL` default; UUID (`gen_random_uuid()` from `pgcrypto`, or v7 via extension) when distributed-coordination-free generation matters.

## Audit columns

Standard set on every entity table:

```sql
created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
created_by VARCHAR(255) NOT NULL,
updated_by VARCHAR(255) NOT NULL,
version    INTEGER     NOT NULL DEFAULT 1
```

Why each:

- `created_at` / `updated_at` — observability, debugging, audit
- `created_by` / `updated_by` — "who changed this?" is asked in every incident; SOC 2 / ISO 27001 evidence
- `version` — optimistic locking column (see `transactions-and-isolation.md`)

Auto-update `updated_at` in Postgres:

```sql
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

Cosmos / Mongo: enforce in application layer (or via change feed for derived `updated_at`). DB triggers aren't available.

Add these columns **from day 1**. Backfilling `created_at` for historical rows is impossible — best you can do is "before <import date>", which is useless for forensics.

## Soft delete

Two patterns:

| Pattern | Trade-off |
|---|---|
| **`deleted_at TIMESTAMPTZ NULL`** on the row | Simple; every query needs `WHERE deleted_at IS NULL`; FK constraints awkward |
| **Move to `<table>_archive`** | Clean primary table; harder to undelete; archive accumulates |

Default: soft-delete column for tables where undelete is plausible (user accounts, orders). Hard delete for tables where it isn't (idempotency keys, session tokens).

**Hidden trap**: every query, FK, and report needs the `deleted_at IS NULL` filter. Easy to forget. Mitigations:

- **Partial index**: `CREATE INDEX ON orders (customer_id) WHERE deleted_at IS NULL` — indexes live rows only.
- **View**: `CREATE VIEW v_orders AS SELECT * FROM orders WHERE deleted_at IS NULL`. App queries the view.
- **Row-Level Security** (Postgres, Azure SQL): policy enforces the filter at engine level.

## Multi-tenancy schema patterns

| Pattern | When |
|---|---|
| **Shared schema, `tenant_id` column** | Default; many tenants; cheap; row isolation via app + RLS |
| **Schema per tenant** (Postgres schemas) | Few-to-medium tenants; need DB-level isolation; per-tenant migrations |
| **Database per tenant** | Strict compliance, very large per-tenant data, regulated isolation |

For shared-schema multi-tenancy:

- Every entity table has `tenant_id`
- Every query filters by `tenant_id` (RLS, app layer, or both — RLS strongly preferred)
- `tenant_id` is the **first column** of every composite index — drives partition pruning if you also partition by tenant
- Connection-time tenant context: `SET LOCAL my.tenant_id = '...'` per request; RLS policy reads it

For Cosmos multi-tenancy: tenant_id is often the partition key (see `patterns/partition-key-design.md` — beware hot-tenant skew).

## Schema evolution — versioning

Schema changes happen. Plan for them.

- **Additive changes** (new column, table, index) — expand/contract pattern in `data-migration-patterns.md`. Safe with zero downtime.
- **Backward-incompatible changes** (rename, type change, FK addition) — require dual-write / dual-read window. Always.
- **Schema version tracking** — every database has a `schema_migrations` table maintained by Flyway / Liquibase / `golang-migrate`. Without migration tooling, environments drift silently.

For document stores: include `schemaVersion` on every document. Readers handle older versions gracefully ("if no `currency` field, default to USD"). Writers always write the latest. Old versions age out or get backfilled.

## Constraints — where to enforce

| Constraint | Where |
|---|---|
| `NOT NULL` | DB. Cheap and fundamental. |
| Foreign key (within service) | DB. Catches referential bugs at write time. |
| Foreign key (cross-service) | **Not the DB.** Use eventual consistency; see `microservices-data-architecture` database-per-service. |
| `CHECK` (structural, e.g., `total >= 0`) | DB. Defense in depth. |
| Uniqueness | DB. App-only checks are racy under concurrent inserts. |
| Business invariants ("can't ship before payment") | App / service layer. DB doesn't know the rule. |

Rule of thumb: **structural invariants in the DB, business invariants in the app.** The DB is the last line of defense against malformed data; it isn't the place to encode the business.

## Worked example — brownfield: adding multi-tenancy to a single-tenant Postgres schema

Setup: existing Spring Boot order service on Container Apps, Postgres Flexible Server, single-tenant. Business sells to a second customer; you need multi-tenancy. Downtime is not acceptable.

Decision walk:

1. **Confirm the pattern.** Two tenants growing to ~20 within a year. Shared-schema with `tenant_id` is the right choice; DB-per-tenant is overkill for that scale.
2. **Expand.** `ALTER TABLE orders ADD COLUMN tenant_id UUID NULL`. Same for every other entity table. Nullable initially — old rows have no tenant assigned yet. See `data-migration-patterns.md` expand/contract.
3. **Backfill.** All existing rows belong to tenant 1. `UPDATE orders SET tenant_id = '<tenant-1-uuid>' WHERE tenant_id IS NULL` in batches with throttling.
4. **Tighten.** `ALTER TABLE orders ALTER COLUMN tenant_id SET NOT NULL` after backfill completes.
5. **Application writes the tenant_id.** Source from the authenticated request context — JWT claim, header, or similar.
6. **Filter every query — but at engine level.** This is the dangerous part. Manual `WHERE tenant_id` audit on every query is unforgettable to forget. Enable **Postgres Row-Level Security**:
   ```sql
   ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
   CREATE POLICY tenant_isolation ON orders
     USING (tenant_id = current_setting('my.tenant_id')::uuid);
   ```
   App sets `SET LOCAL my.tenant_id = '...'` at the start of every transaction. Cross-tenant data leak becomes impossible at engine level.
7. **Index reorder.** Every composite index leads with `tenant_id`: `CREATE INDEX CONCURRENTLY ON orders (tenant_id, customer_id, created_at)`. Drop old single-tenant indexes after the new ones are confirmed used.
8. **Hot-tenant monitoring.** Per-tenant query rate dashboard. If one tenant exceeds 50% of traffic and degrades others, plan extraction to a dedicated database for that tenant.
9. **Verification.** Integration tests with two tenant contexts; assert zero rows of tenant-A visible from tenant-B context. Log the tenant_id on every request.

Total elapsed: 2–4 weeks for a small service with disciplined rollout; longer for systems with many tables. Downtime: zero.

Wrong answer: "fork the database per tenant from the start." Works for 2 tenants; breaks at 20.

## Anti-patterns

- **`SELECT *` in app code.** Schema change ripples to every caller silently.
- **Random UUID PK on InnoDB.** Clustered-index destruction; index bloat. Use BIGINT or UUIDv7.
- **Unbounded embedded arrays.** Document size grows until it hits the 16MB Mongo limit or Cosmos RU per item makes reads/writes punitive.
- **No `updated_at` / `version` column.** Optimistic locking impossible; debugging "when did this change?" impossible.
- **Hard delete on tables that might need undelete.** Data gone, recovery via PITR only.
- **Multi-tenant without RLS or audited filtering.** First missing `WHERE tenant_id` is a cross-tenant data leak. SOC 2 incident.
- **FK across service boundaries.** Tight coupling; joint deployment dance forever.
- **Business logic in `CHECK` constraints.** Brittle across schema changes; hard to debug when violated. Enforce in app.
- **Schema version not tracked.** Migrations get applied out of order; environments diverge silently.
- **Audit columns added later.** Historical rows have no `created_at` source of truth.
- **PK in Cosmos = `/id` because it's unique.** Yes, but it makes every query cross-partition. See `patterns/partition-key-design.md`.

## Verification questions

1. Is the access pattern (top 5–10 queries) documented before the schema was finalized?
2. For relational: are FKs enforced within service, omitted across services?
3. For multi-tenant: is `tenant_id` the first column of every composite index, with RLS or audited filtering enforcing isolation?
4. Is the key strategy ordered (BIGINT auto-increment or UUIDv7), not random UUID on InnoDB?
5. Do all entity tables have `created_at`, `updated_at`, `created_by`, `updated_by`, `version`?
6. For Cosmos / Mongo: are embedded arrays bounded? Has the worst case been measured?
7. Is schema versioning tooling in place (Flyway / Liquibase / `golang-migrate`)?
8. Is soft-delete filtering applied via partial index, view, or RLS — not manually on every query?

## What to read next

- `engine-selection.md` — schema shape drives engine choice
- `data-migration-patterns.md` — applying schema changes without downtime
- `transactions-and-isolation.md` — optimistic locking with `version`
- `partitioning.md` — when the schema needs table-level partitioning
- `patterns/partition-key-design.md` — Cosmos partition key (different concept)
- `cosmos-db-design.md` — document modeling specifics for Cosmos
- `mongodb-on-azure.md` — document modeling for Mongo
- `../microservices-data-architecture` — cross-service consistency; database-per-service
- `../azure-microservices-security` — RLS, Entra auth, multi-tenant data isolation
- `../soc2-iso27001-controls-mapping` — audit columns as compliance evidence
