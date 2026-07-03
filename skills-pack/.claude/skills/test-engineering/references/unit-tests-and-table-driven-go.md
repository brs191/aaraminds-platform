# Unit Tests and Table-Driven Go

This reference covers the base of the pyramid — fast, isolated, deterministic tests of one unit of behavior — and the table-driven form that Go makes idiomatic and that `pytest` parametrization mirrors.

## What a unit test is for

A unit test pins one behavior of one unit of code so that unit can be changed with confidence. It is fast (milliseconds), isolated (no network, no disk, no clock, no shared state), and deterministic (same result every run). If a test touches a real database or an HTTP endpoint it is an integration test — a different tier with a different budget — not a unit test. The FIRST properties name the bar: Fast, Isolated, Repeatable, Self-validating (a clear pass/fail, no human reading output), Timely (written with the code, not "later").

## Test behavior, not methods

Name and structure tests around behaviors, not around the methods of the unit. `resolves_relative_import_to_declaring_module` is a behavior; `TestResolve_case3` is not. One behavior per test keeps a failure diagnosable — when it goes red you know what broke from the name alone. A unit may need many tests because it has many behaviors and edge cases; that is correct, not duplication.

## Table-driven tests in Go

Go has no parametrized-test framework because it does not need one — a slice of structs is the idiom. Each row is a named case with inputs and the expected output; the loop runs each as a subtest:

```go
func TestResolveImport(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        pkg      string
        want     string
        wantErr  bool
    }{
        {"absolute import", "com.acme.Order", "com.acme", "com.acme.Order", false},
        {"relative same package", "Order", "com.acme", "com.acme.Order", false},
        {"unresolvable symbol", "Mystery", "com.acme", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ResolveImport(tt.input, tt.pkg)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

`t.Run` with the case name means a failure reports *which* row failed. Adding a behavior is adding a row. Use `t.Parallel()` inside the subtest only when cases share no mutable state. Reach for `testify` assertions if the team wants them, but plain `if`/`t.Errorf` is idiomatic and dependency-free.

## The pytest equivalent

`pytest.mark.parametrize` is the same idea for Python: one test function, a list of cases, each id'd. Use `assert` directly — pytest rewrites it to show both sides on failure. Async units use `pytest-asyncio`; the model boundary is faked, not called (see `test-doubles-and-fixtures.md`).

```python
@pytest.mark.parametrize("text,pkg,want", [
    ("com.acme.Order", "com.acme", "com.acme.Order"),
    ("Order",          "com.acme", "com.acme.Order"),
])
def test_resolves_import(text, pkg, want):
    assert resolve_import(text, pkg) == want
```

## Edge cases and failure paths

The table makes it cheap to be thorough: empty input, the zero value, the boundary, the malformed input, the error path. A unit tested only on its happy path is barely tested — most production bugs live in the cases the author did not type out. Add the error rows.
