# Go Anti-Patterns to Flag in PR Review

Named patterns to scan for when reviewing a Go 1.25+ PR. Fast to spot, high-leverage.

## 1. Swallowed errors

**Pattern:**
```go
result, _ := svc.DoWork(ctx)
```

**Detection cue:** any `_, _ := ...` or `_ := ...` where the discarded position is an `error`.

**Why it fails:** silent failure. The error is gone; the caller proceeds with a zero-value result.

**Fix:**
```go
result, err := svc.DoWork(ctx)
if err != nil {
    return fmt.Errorf("do work: %w", err)
}
```

`%w` wraps the error so callers can `errors.Is` / `errors.As` it. Naked error strings without `%w` break that chain.

## 2. `log` package or `logrus`

**Pattern:** `import "log"` or `import "github.com/sirupsen/logrus"`.

**Detection cue:** any import of `log` or `logrus` in new code.

**Why it fails:** `log` has no structured output. `logrus` is maintenance-mode and not the stdlib direction. Java services have `slf4j`+`logback`; Go services should use `log/slog` (stdlib since 1.21).

**Fix:** `log/slog` with JSON handler:
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
```

For MCP servers using stdio transport, **`os.Stderr` instead of `os.Stdout`** — see `mcp-go-server-building`.

## 3. Package-level mutable globals

**Pattern:**
```go
var cache = map[string]*Customer{}

func GetCustomer(id string) *Customer {
    return cache[id]
}
```

**Detection cue:** any `var foo = ...` at package level that's not a constant or a logger.

**Why it fails:** can't unit-test without resetting global state; concurrent access is a race; lifecycle is implicit (no `Close`, no clear init).

**Fix:** struct + constructor:
```go
type CustomerCache struct {
    data sync.Map
}

func NewCustomerCache() *CustomerCache { return &CustomerCache{} }
```

## 4. `init()` for dependency wiring

**Pattern:**
```go
var db *sql.DB

func init() {
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil { panic(err) }
}
```

**Detection cue:** `init()` function that opens connections, reads config, or sets up dependencies.

**Why it fails:** can't pass test config; can't handle errors gracefully (panic on startup); can't test the package without the dependency being available.

**Fix:** explicit wiring in `internal/app/Run` or `internal/app/New`. See `../../new-azure-service-bootstrap/references/go-scaffold.md`.

## 5. Missing context propagation

**Pattern:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
    result, err := svc.DoWork()       // No context
    ...
}
```

**Detection cue:** function calls in a handler chain where context is not passed.

**Why it fails:** request cancellation doesn't propagate. If the client disconnects or the request times out, downstream calls (DB, HTTP) keep running until they complete or their own timeout fires. Wastes resources; can produce stale writes after the user has given up.

**Fix:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
    result, err := svc.DoWork(r.Context())
}
```

Every function that does I/O takes `ctx context.Context` as the first parameter.

## 6. `fmt.Errorf` without `%w` for wrapped errors

**Pattern:**
```go
if err != nil {
    return fmt.Errorf("fetch: %v", err)
}
```

**Detection cue:** `%v` (or `%s`) in `fmt.Errorf` formatting an `err` variable.

**Why it fails:** breaks the error chain. `errors.Is(err, sql.ErrNoRows)` returns false even if the wrapped error was `sql.ErrNoRows`.

**Fix:** `%w`:
```go
return fmt.Errorf("fetch: %w", err)
```

`%w` only works on one error per `Errorf` call. For multiple errors, use `errors.Join`.

## 7. Naked goroutines

**Pattern:**
```go
go doWork(req)
```

**Detection cue:** `go` keyword launching a function in a handler or business logic, without explicit lifecycle / cancellation / error recovery.

**Why it fails:** the goroutine outlives the request; it has no way to communicate failure; if `doWork` panics, the whole process crashes; if `doWork` leaks (e.g., blocked on a channel), it leaks forever.

**Fix:** at minimum, recover from panic:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("goroutine panic", slog.Any("recover", r))
        }
    }()
    doWork(ctx)
}()
```

Better: use an `errgroup.Group` or a worker pool with a bounded queue, and pass a derived context that the parent can cancel.

## 8. `http.DefaultClient` for outbound calls

**Pattern:**
```go
resp, err := http.Get("https://downstream/api")
// or
resp, err := http.DefaultClient.Do(req)
```

**Detection cue:** `http.Get`, `http.Post`, `http.DefaultClient` anywhere in business code.

**Why it fails:** `http.DefaultClient` has *no timeout*. A slow or hung downstream blocks the goroutine until the OS times out the TCP connection (minutes). Also has no instrumentation, no connection pool tuning.

**Fix:** explicit client with timeout:
```go
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
    Timeout:   5 * time.Second,
}
```

`otelhttp.NewTransport` adds OpenTelemetry tracing. See `microservices-resilience` for retry / circuit-breaker patterns on top of the client.

## 9. `interface{}` / `any` parameters with type assertions

**Pattern:**
```go
func Process(input any) error {
    s, ok := input.(string)
    if !ok {
        return errors.New("expected string")
    }
    ...
}
```

**Detection cue:** `any` (or `interface{}`) parameter followed by `.(SomeType)` assertion.

**Why it fails:** loses compile-time type safety. The caller can pass anything; the function discovers at runtime. Generics (Go 1.18+) almost always express this better.

**Fix:** generic function or specific type:
```go
func Process[T constraint](input T) error { ... }
```

Or just `func Process(input string) error`.

## 10. Returning unwrapped third-party errors

**Pattern:**
```go
func GetCustomer(ctx context.Context, id string) (*Customer, error) {
    row := db.QueryRowContext(ctx, "SELECT ...", id)
    var c Customer
    if err := row.Scan(&c.ID, &c.Name); err != nil {
        return nil, err          // Bare error from sql driver
    }
    return &c, nil
}
```

**Detection cue:** `return ..., err` where `err` is from a third-party package (sql, http, redis client), without context.

**Why it fails:** the caller sees `pq: connection refused` or `sql: no rows in result set` with no application context. Debug logs become forensic exercises.

**Fix:** wrap with application context:
```go
if err := row.Scan(&c.ID, &c.Name); err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrCustomerNotFound
    }
    return nil, fmt.Errorf("get customer %s: %w", id, err)
}
```

Translate driver errors to domain errors where the boundary makes sense.

## 11. `panic` for non-programmer errors

**Pattern:**
```go
data, err := json.Marshal(payload)
if err != nil {
    panic(err)
}
```

**Detection cue:** `panic(err)` for an `err` from I/O, parsing, or external systems.

**Why it fails:** `panic` is for programmer errors that should crash the process (nil dereferences, impossible-state assertions). I/O errors are runtime conditions; the caller should decide.

**Fix:** return the error. If the call is in a goroutine without a return path, log + recover.

## 12. Unbounded `select` with no `<-ctx.Done()`

**Pattern:**
```go
for {
    select {
    case msg := <-ch:
        process(msg)
    }
}
```

**Detection cue:** `for { select { ... } }` without a `<-ctx.Done()` case.

**Why it fails:** the loop runs forever; shutdown signals can't cancel it; the goroutine leaks.

**Fix:**
```go
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case msg := <-ch:
        process(msg)
    }
}
```

## 13. `defer` in a loop

**Pattern:**
```go
for _, file := range files {
    f, err := os.Open(file)
    if err != nil { ... }
    defer f.Close()           // Defers stack up — file handles leak until function returns
    process(f)
}
```

**Detection cue:** `defer` inside a `for` loop.

**Why it fails:** `defer` runs at function return, not loop iteration. For long-running functions, file handles, mutex locks, or DB connections accumulate.

**Fix:** wrap the body in a closure:
```go
for _, file := range files {
    if err := func() error {
        f, err := os.Open(file)
        if err != nil { return err }
        defer f.Close()
        return process(f)
    }(); err != nil { return err }
}
```

## 14. Using `time.Now()` for monotonic timing

**Pattern:**
```go
start := time.Now()
work()
elapsed := time.Now().Sub(start)
```

This is actually *fine* in Go because `time.Now()` includes a monotonic reading and `Sub` uses it. The anti-pattern is:

```go
start := time.Now().UTC()         // .UTC() strips the monotonic component
work()
elapsed := time.Now().UTC().Sub(start)
```

**Detection cue:** `.UTC()` (or `.Local()`, `.Round()`, `.Truncate()`) chained on `time.Now()` for timing measurements.

**Why it fails:** these methods strip monotonic clock readings, leaving you with wall-clock time. If the system clock jumps backward (NTP correction), `elapsed` is negative or wildly wrong.

**Fix:** keep the monotonic clock — don't transform `time.Now()` until you actually need to format it.

## 15. `var x interface{} = ...` for "any value"

**Pattern:** `var x interface{} = someValue`

**Detection cue:** explicit `interface{}` (or `any`) typing of variables that have a concrete known type.

**Why it fails:** loses type information for no gain. The variable's downstream uses now need type assertions.

**Fix:** let Go infer the type, or declare the concrete type explicitly.

## 16. Mutex value receiver

**Pattern:**
```go
type Counter struct {
    mu sync.Mutex
    n  int
}

func (c Counter) Increment() {     // Value receiver
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}
```

**Detection cue:** struct with `sync.Mutex` (or `sync.RWMutex`) field has a method with a value receiver.

**Why it fails:** value receiver copies the mutex; each `Increment` locks a *different copy* of the mutex; the `c.n++` doesn't affect the original. Race + lost updates.

**Fix:** pointer receiver:
```go
func (c *Counter) Increment() { ... }
```

`go vet` catches this with the `copylocks` check. Verify CI runs `go vet`.

## 17. SQL string concatenation (injection)

**Pattern:** `db.QueryContext(ctx, "SELECT * FROM users WHERE name = '" + name + "'")`

**Detection cue:** any `+` in a SQL string built from user-controlled data.

**Why it fails:** SQL injection.

**Fix:** parameter binding: `db.QueryContext(ctx, "SELECT * FROM users WHERE name = $1", name)`. For pgx: `pool.Query(ctx, "...", arg1, arg2)`.

## 18. Forgetting `rows.Close()`

**Pattern:**
```go
rows, err := db.QueryContext(ctx, "...")
if err != nil { return err }
for rows.Next() {
    ...
}
return nil      // No rows.Close()
```

**Detection cue:** `db.QueryContext` or `pool.Query` followed by `for rows.Next()` without a corresponding `defer rows.Close()`.

**Why it fails:** connection leak. The connection is returned to the pool only on `Close()`.

**Fix:**
```go
rows, err := db.QueryContext(ctx, "...")
if err != nil { return err }
defer rows.Close()
```

## How to use this list in review

Scan a Go diff for these patterns by name. The fastest signals to grep for: `_, _ :=` (swallowed errors), `http.Get` / `http.DefaultClient`, `var ... = ...` at package scope, `panic(`, `for { select { case ... }` (look for missing ctx.Done), `defer` inside `for`. Most of these can be caught in 5 minutes for a typical PR.

`go vet` and `staticcheck` catch some of these automatically — verify they run in CI. Treat this list as the human-judgment layer on top of the tooling.
