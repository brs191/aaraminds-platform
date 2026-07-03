# Suite Health and CI Gating

This reference covers the test suite as a system that decays — and the discipline that keeps it a trustworthy gate rather than a tax.

## The pyramid as a budget

The test pyramid is a budget on where test effort goes: many fast unit tests at the base, fewer integration tests in the middle, a few end-to-end tests at the top. The shape is dictated by cost and stability — unit tests are milliseconds and stable; end-to-end tests are slow, flaky, and expensive to diagnose. An "ice-cream cone" — mostly end-to-end, few unit tests — runs for an hour, flakes constantly, and localizes nothing. When a behavior *can* be covered at a lower tier, cover it there. Reserve each higher tier for what genuinely needs it: integration tests for real-dependency interactions, end-to-end for a few critical user journeys.

## A speed budget per tier

Slow suites get run less, and a test that is not run protects nothing. Set and hold a budget: the unit suite finishes in well under a minute, the integration suite in a few minutes, end-to-end measured but kept short. When the unit suite drifts past its budget, something has leaked into it — real I/O, a `sleep`, a too-broad scope. Find it and fix it; do not raise the budget. Parallelize where tests are isolated (`go test` parallelizes packages; `pytest-xdist` for Python).

## Flakiness is a P1, not a nuisance

A flaky test — one that passes and fails on the same code — is more damaging than a missing test. It trains the team to re-run CI until green, and that habit means a *real* failure is also re-run away. Treat a flake as a priority bug: quarantine it from the gate immediately so it stops eroding trust, then fix the root cause — almost always a hidden dependency on time, ordering, shared state, or a real network call. Do not "fix" a flake with a `sleep` or a retry wrapper; that hides it. Zero tolerated flakes in the gating suite.

## Coverage is a signal, not a target

Line coverage tells you what code *ran* during tests — not what was *verified*. Code can be 100% covered by tests that assert nothing. Read coverage as a signal: a sharp drop in a PR, or a critical module sitting near zero, is worth investigating. Made a target — "the build fails under 80%" — it gets gamed with assertion-free tests that execute lines, and the number becomes a lie. Watch the trend and the gaps; do not gate on the percentage.

## The suite as a CI gate

Tests earn their keep by *blocking* a merge when they fail. Wire the suite into CI (GitHub Actions, per the stack) so a red suite blocks the PR. Order by speed: run the fast unit suite first for quick feedback, then integration, then end-to-end. A milestone gate ("M1 complete") is operationally the moment a defined suite passes in CI — the suite *is* the gate. Combine with the static checks (`mypy`/`pyright`, `go vet`, lint) and, for AI features, the eval gate from `ai-evaluation-harness` — tests and evals are separate gates on the same pipeline.

## Where this meets the eval discipline

CI runs two kinds of quality gate. **Tests** assert deterministic behavior of code — binary pass/fail. **Evals** score non-deterministic model output against a rubric with a threshold (`ai-evaluation-harness`). Both block the pipeline; they are not interchangeable. Keep them as distinct stages so a failure tells you which kind of thing broke: the code, or the model's output quality.
