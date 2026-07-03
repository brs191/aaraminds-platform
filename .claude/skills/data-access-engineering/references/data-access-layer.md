# The Data Access Layer

This reference covers structuring data access in code — keeping queries behind a boundary, returning typed results, scoping transactions, and using connection pools correctly.

## Queries belong behind a boundary, not in business logic

A query embedded in business logic couples that logic to the store's shape: the business code now knows table names, Cypher syntax, the schema. Change the schema and unrelated code breaks; test the logic and you need a database. Put queries behind a **boundary** — a data-access module, a repository — that exposes intent-named operations (`findBlastRadius(methodId)`, `upsertComponent(component)`) and hides the query. Business logic calls the operation; only the data-access layer knows the query. The boundary is what lets the schema and the business logic change independently.

## The repository-style boundary

A repository-style component groups the data-access operations for one part of the model — a `MethodRepository`, a `ComponentRepository`, a `DocumentRepository`. Each exposes a small set of intent-named methods, takes and returns the application's typed objects, and owns the queries for its slice. Keep repositories focused — one per aggregate or model area — and free of business rules: a repository fetches and stores, it does not decide. This is the separation `mcp-go-server-building`'s service/handler split and `python-service-engineering`'s module discipline apply, at the data boundary.

## Typed results, not raw records

A data-access operation returns the application's typed objects — a `Method`, a `Component`, a list of typed results — not raw driver rows, dictionaries, or untyped records. Map the raw result to the typed object *inside* the data-access layer, at the boundary. Returning raw records leaks the store's shape — column names, types, nullability — into the caller and reintroduces exactly the untyped-boundary problem `python-service-engineering` and `mcp-go-server-building` warn against. The boundary's job includes the mapping.

## Transaction scope is a decision

A transaction's scope — what is inside one atomic unit — is a design decision, not a default. Too wide (a transaction held across unrelated work, or across a slow external call) holds locks and stalls others; too narrow (each write its own transaction when they must all succeed or fail together) loses atomicity. Decide deliberately: a logical unit of work is one transaction. Open it, do the work, commit or roll back, and do not hold it across a network call to something that is not the database. Transaction scope belongs in the data-access layer, visible — not hidden inside an ORM's defaults.

## Connection pools — use them correctly

Connections are a bounded, expensive resource; the application talks to the store through a connection **pool** (`azure-data-tier-design` sizes it, including the PgBouncer worked example). Using it correctly in code: acquire a connection as late as possible, release it as soon as the work is done — never hold one across user think-time or a slow external call — and always release it on the error path, via the language's `with` / `defer` / try-with-resources so a thrown exception cannot leak a connection. A leaked connection is gone from the pool until it times out; enough leaks and the pool is empty and the service stalls — the symptom `azure-data-tier-design`'s connection worked example diagnoses, caused here, in the access code.

## Verification questions

1. Are queries behind a data-access boundary, with business logic calling intent-named operations rather than embedding queries?
2. Are repositories focused — one per model area — and free of business rules?
3. Do data-access operations return typed application objects, with raw-record mapping done inside the boundary?
4. Is transaction scope a deliberate decision — one logical unit of work per transaction, not held across external calls?
5. Are pooled connections acquired late, released promptly, and released on the error path so none leak?

## What to read next

- `query-discipline.md` — the queries this layer wraps
- `azure-data-tier-design`, `references/patterns/connection-pool-sizing.md` — sizing the pool
- `python-service-engineering` and `mcp-go-server-building` — the services that call this layer
- `test-engineering` — testing the data-access layer
