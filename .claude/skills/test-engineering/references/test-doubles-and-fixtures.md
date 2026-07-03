# Test Doubles and Fixtures

This reference covers replacing a unit's collaborators in a test — the kinds of double, when each is right — and building the test data a test runs on.

## The kinds of double

"Mock" is used loosely for all of these; the distinctions matter.

- **Stub** — returns canned answers to calls. Use to supply input the unit needs (a stub repository that returns a fixed row).
- **Fake** — a real, working, simplified implementation. An in-memory repository, an in-memory queue. Behaves like the real thing; just not production-grade.
- **Mock** — a double pre-programmed with *expectations* about how it will be called, and which fails the test if the calls do not match.
- **Spy** — records how it was called so the test can assert afterward.

## Prefer a fake over a mock

A mock asserts *how* the unit used its collaborator — which method, which arguments, in which order. That couples the test to the implementation: refactor the unit without changing its behavior, and mock-based tests break anyway. This is the change-detector anti-pattern from the SKILL router. A fake asserts only the *outcome* — the unit did the right thing, observable in the fake's resulting state — so it survives refactors. Default to a fake. Reach for a mock only when the interaction *is* the behavior under test — e.g. verifying a retry actually retried, or that an audit event was emitted. Even then, assert the meaningful call, not the full call script.

## Do not mock what you do not own

Mocking a third-party SDK or a database driver bakes your *assumption* of how that dependency behaves into the test — and the assumption is often wrong. The test stays green while production breaks. Wrap the external dependency in a thin interface you own, fake *that* interface in unit tests, and cover the real dependency in an integration test (see `integration-tests-and-real-dependencies.md`). For an LLM, fake the model-client wrapper your code owns; never assert against the live model in a unit test.

## Fixtures and builders

Test data should be created so the test reads clearly. A **builder** (or object mother) constructs a valid default object and lets a test override only the fields it cares about:

```python
def a_method(**overrides):
    base = dict(name="process", visibility="public", params=[], pkg="com.acme")
    return MethodNode(**{**base, **overrides})

def test_private_method_excluded_from_api():
    m = a_method(visibility="private")
    assert not is_api_surface(m)
```

The test states *only* what matters to it — `visibility="private"` — and the reader sees the one relevant fact. Avoid giant shared fixture blobs that force a reader to hunt for which field drives the test.

## Determinism: freeze time and randomness

A test that depends on the wall clock, a random seed, a UUID, or map-iteration order is flaky by construction. Inject these as dependencies: pass a clock, pass a seeded RNG, pass an ID generator — production wires the real one, tests wire a frozen one. `freezegun` / a fixed clock in Python, an injected `Clock` interface in Go. A test must mean the same thing on every run, on every machine, or it is noise.
