# Integration Tests and Real Dependencies

This reference covers the middle tier of the pyramid: code tested against its *real* collaborators — a real database, a real message broker, a real HTTP server — rather than mocks of them.

## What integration tests catch that unit tests cannot

A unit test with a mocked database confirms the code called the driver as the author imagined. It cannot confirm the query is valid SQL or Cypher, that the migration applies, that the index is used, that the transaction isolation behaves, or that the real engine returns the shape the code expects. Those bugs live in the *interaction* between the code and the real dependency — exactly what a mock removes. An integration test puts the real dependency back. For data-access code this is not optional (see `data-access-engineering`).

## Use a real, disposable dependency — not a shared one, not a mock

The dependency must be **real** (the actual Postgres/Neo4j version production runs, not SQLite-standing-in-for-Postgres) and **disposable** (created for the test run, torn down after — never a shared long-lived test database that accumulates state and cross-couples tests). Testcontainers is the standard tool in both Go (`testcontainers-go`) and Python (`testcontainers`): it starts the dependency in a container, hands the test a connection string, and kills it after. The container is the unit of isolation.

```go
func TestGraphWritePath(t *testing.T) {
    ctx := context.Background()
    neo4jC, err := neo4j.Run(ctx, "neo4j:5.26")
    if err != nil { t.Fatal(err) }
    defer neo4jC.Terminate(ctx)
    uri, _ := neo4jC.BoltUrl(ctx)
    // ... connect, run migrations, exercise the write path, assert the graph
}
```

## Seed state explicitly, per test

Each integration test sets up the exact state it needs and does not depend on state left by another. Apply schema migrations against the fresh container, then insert the fixtures this test requires. Tests that share a database and rely on insertion order are flaky by construction. Where a full container per test is too slow, share one container but isolate per test with a transaction rolled back at the end, or a truncate between tests — isolation by data, not by ordering luck.

## Contract tests at service boundaries

Where two services meet, an integration test is a *contract* test: it pins the request and response shape the consumer depends on so the provider cannot break it silently. For an internal API, a test that exercises the real HTTP handler and asserts the JSON contract. For the CIF's Go gateway calling a Python service, a contract test on the wire format. Contract tests are the seam-level safety net that lets two services evolve independently.

## Keep the tier small and deliberate

Integration tests are slower than unit tests by one to two orders of magnitude — a container start is seconds. The pyramid shape means there are *few* of them, each earning its cost by covering a real interaction. Do not reach for an integration test when a unit test would do; do not skip one when the bug can only live in the real interaction. The speed budget for this tier belongs in `suite-health-and-ci-gating.md`.
