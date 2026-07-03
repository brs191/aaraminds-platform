# Relational Migrations

This reference covers changing a relational schema in production without downtime — the expand/contract pattern, migration tooling, backfills, and rollback. It implements the migration design from `azure-data-tier-design`.

## Schema change is expand/contract, never edit-in-place

A live service reads and writes the schema while you change it, and old and new application code run side by side during a deploy. So a schema change cannot be a single edit-in-place — renaming a column out from under running code breaks it. The pattern is **expand/contract**: *expand* the schema to support old and new code at once, deploy the new code, then *contract* away the old shape once nothing uses it. A column rename becomes: add the new column (expand), backfill it, deploy code that writes both and reads new, then drop the old column (contract) — several deploys, each individually safe.

## A migration tool with versioned, ordered migrations

Schema changes are versioned migration files applied in order by a migration tool — Flyway or Liquibase on the JVM, Alembic for Python, golang-migrate for Go. Each migration is immutable once applied, forward-only in production, and recorded in a schema-history table so the tool knows what has run. Never hand-edit the production schema outside the tool: the migration history is the source of truth for what shape the database is in, and an out-of-band change makes it lie.

## The expand/contract sequence

A non-trivial change is a sequence of migrations across deploys, not one:

1. **Expand** — add the new structure (a nullable column, a new table); old code is unaffected because it ignores it.
2. **Backfill** — populate the new structure for existing rows (below).
3. **Migrate code** — deploy code that writes both old and new and reads new.
4. **Contract** — once no code reads the old structure, drop it.

Each step is its own migration and its own deploy. Collapsing them is how a migration takes the service down.

## Backfills — separate from the schema change

Backfilling existing rows is a *data* operation, not a schema operation, and it does not belong inside the schema migration. A schema migration must be fast — it can hold locks; a backfill that updates millions of rows inside the migration holds those locks for minutes and stalls the service. Run the backfill as a separate, batched, resumable job — update in chunks, commit per chunk — after the expand migration and before the contract. Keep schema DDL and bulk data updates apart.

## Rollback and forward-fix

In production, prefer **forward-fix** over rollback: a migration that has already run and been written against is safer to correct with a new migration than to reverse. Design each migration so the *previous* application version still works against it — that is what expand/contract gives you, and it means a bad deploy is recovered by redeploying old code, not by reversing the schema. A genuinely reversible migration with a tested `down` is worth having for the pre-traffic window; once real data depends on the new shape, forward-fix.

## Migrations in CI and at deploy

Run migrations automatically as a deploy step, not by hand — a separate step or job before the new code goes live, so the schema is ready when the code arrives. In CI, apply every migration to a throwaway database and run the tests against it, so a broken migration fails the build rather than production. A migration that has only ever run on the author's machine is an untested production change.

## Verification questions

1. Are schema changes done as expand/contract — never an edit-in-place that breaks running code?
2. Are migrations versioned, ordered, immutable-once-applied files run by a migration tool, with no out-of-band schema edits?
3. Is a multi-step change sequenced across deploys (expand, backfill, migrate code, contract)?
4. Are backfills separate batched jobs, not bulk updates inside a schema migration?
5. Does each migration keep the previous app version working, with forward-fix preferred over rollback?
6. Do migrations run as an automated deploy step and apply against a throwaway DB in CI?

## What to read next

- `azure-data-tier-design`, `references/data-migration-patterns.md` — the migration design
- `data-access-layer.md` — the code that reads the migrated schema
- `query-discipline.md` — query changes that accompany schema changes
- `test-engineering` — testing migrations
