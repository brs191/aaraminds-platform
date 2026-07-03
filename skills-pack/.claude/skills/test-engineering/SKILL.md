---
name: test-engineering
description: Designs and writes the test suite across the stack — unit and table-driven Go tests, pytest, integration tests against real dependencies (Testcontainers), characterization tests that pin behavior before a change, test doubles and fixtures, and test-suite health (the pyramid, flakiness, CI gating). Use when writing or reviewing tests, deciding what to test and at which level, pinning legacy behavior before a refactor, choosing a mock vs a fake, or fixing a flaky suite. Do not use for the AI/LLM eval harness (use ai-evaluation-harness) or the service code itself (use python-service-engineering or mcp-go-server-building).
version: 1.0.1
last_updated: 2026-05-30
---

# Test Engineering

## When to use

Trigger this skill when writing, reviewing, or repairing tests — at any level, in any language in the stack. Every milestone gate in a roadmap is, in practice, a test gate: "M1 done" means a suite passes. Common triggers: "write tests for this," "what should this test cover," "this needs tests before I can refactor it," "should this be a mock or a fake," "the suite is slow / flaky," "is this test actually testing anything."

This skill owns *test strategy and test code*. The service skills (`python-service-engineering`, `mcp-go-server-building`) build the code; this skill decides what to test, at which level, and writes the tests. Build test-first against this skill.

Do **not** use this skill for: the AI/LLM evaluation harness — golden datasets, rubrics, LLM-as-judge scoring, eval CI gating (`ai-evaluation-harness`); the service or extractor code itself (`python-service-engineering`, `mcp-go-server-building`, `codebase-extraction-engineering`); the disposable-engine setup for data-access integration tests (`data-access-engineering` — this skill owns the *test*, that skill owns the *engine wiring*).

## The critical decision rule — test observable behavior at a stable seam, never the implementation

A test exists to let you change the code with confidence. It can only do that if it is coupled to something that *should not change when the code is refactored* — a public function's contract, an API response, a stored row — and not to something that *will* change — a private method, a call order, an internal data shape. So the rule: **assert observable behavior at a stable seam.** A test that reaches into internals breaks on every refactor, which trains the team to delete or rewrite tests to make a change land — at which point the suite no longer protects anything, it just taxes change. The second half of the rule: **a test you cannot trust is worse than no test.** A flaky test that fails 1-in-20 for no reason teaches everyone to re-run CI until it is green, and a real failure dies in that same noise. Test behavior, at a seam, deterministically — or do not write the test.

## Unit tests

The base of the pyramid: fast, isolated, deterministic tests of one unit of behavior. Table-driven tests in Go (`tests []struct{...}` with `t.Run`), `pytest` with parametrization for Python. Each test names a behavior, not a method; tests are independent and order-free; no real I/O. `references/unit-tests-and-table-driven-go.md`.

## Integration tests

The middle of the pyramid: code tested against its *real* dependency — a real Postgres, a real Neo4j, a real HTTP server — in a container, via Testcontainers, torn down per run. Integration tests catch what unit tests with mocked dependencies cannot: the query the engine rejects, the migration that locks, the contract mismatch. `references/integration-tests-and-real-dependencies.md`.

## Characterization tests

When code has no tests and must be changed, you first pin what it *currently* does — including its bugs — with characterization (golden-master / approval) tests, so a refactor that changes behavior is caught. This is the safety net that makes brownfield change safe. `references/characterization-tests.md`.

## Test doubles and fixtures

Choosing the right double — stub, fake, mock, spy — and knowing a fake (a working in-memory implementation) usually beats a mock (a scripted expectation). Fixture and builder design, deterministic test data, and freezing time and randomness so a test means the same thing twice. `references/test-doubles-and-fixtures.md`.

## Suite health and CI gating

The suite is itself a system that decays: it gets slow, it gets flaky, coverage becomes a number people game. The test pyramid as a budget, a speed budget per tier, zero tolerance for flakes, coverage read as a signal not a target, and the suite wired as a CI gate. `references/suite-health-and-ci-gating.md`.

## Where tests end and evals begin

Tests and evaluations are different instruments and the boundary is non-deterministic output. A **test** asserts a deterministic contract: given this input, the function returns *exactly* this — pass or fail, no judgement. An **eval** scores a non-deterministic surface — an LLM's generated answer — against a rubric, producing a graded result and a threshold. Code is tested; model output is evaluated. The CIF's extractor, graph write path, and APIs are test-engineering's domain; the quality of a generated HLD document is `ai-evaluation-harness`'s. Do not write a brittle string-equality "test" over an LLM response — that is an eval wearing a test's clothes, and it will flap. Route model-output quality to `ai-evaluation-harness`.

## Test code is production code

A test suite is read far more often than it is written, and it is read most urgently at the worst moment — when it has just failed in CI and someone needs to know why. So test code carries the same bar as production code with one inversion: production code is kept DRY, test code is kept DAMP — descriptive and a little repetitive — because a reader must understand a failing test top-to-bottom without chasing helpers across files. Each test follows arrange / act / assert with the three phases visible. A test's name states the behavior and the condition (`returns_empty_graph_for_empty_repo`, not `test3`). A failure message says what was expected and what was got, so a red CI line is diagnosable without a debugger. Shared setup that hides what a test depends on is a liability, not reuse.

## Worked example — brownfield: pinning an untested extractor before a refactor

Setup: the CIF's code extractor (`codebase-extraction-engineering`) works but has no tests, and it needs a structural refactor to support incremental rebuilds. Refactoring untested code is how a working system silently breaks.

Decision walk: (1) Do not refactor first. Pick a small, representative sample repository as a fixture. (2) Run the current extractor over it and capture the emitted graph as a golden file — a characterization test that pins *current* behavior, bugs included. (3) Add one or two more sample repos covering generated code and a multi-module build, each with its own golden. (4) Now refactor under the net — any behavior change trips a golden diff, which you then judge: intended (update the golden) or a regression (fix the code). (5) Once the refactor lands, add real *unit* tests for the new internal structure — the call-graph resolver, the identity scheme — at their stable seams. (6) Add an *integration* test that runs the extractor against a real Neo4j and asserts the graph wrote correctly. The characterization tests can now be retired or kept as a coarse smoke test.

The wrong move is to refactor first and "add tests after" — the tests then encode whatever the refactor produced, including any regression it introduced, and the bug is canonized.

## Anti-pattern — the change-detector test

**Bad:** tests that assert implementation detail — a mock verifying a private method was called in a specific order, a snapshot of an entire internal object, an assertion on a log line's exact text. **Why it fails:** every such test fails on a refactor that changed nothing observable, so the team learns to mass-update or delete tests to land a change — and a suite that is routinely rewritten to pass protects nothing. **Detection signal:** tests break on every refactor while behavior is unchanged; mocks asserting call order; giant snapshot files; a test's name is a method name, not a behavior. **Fix:** assert observable behavior at a stable seam; prefer a fake over a mock; snapshot a small, meaningful output, not an internal object — the decision rule above.

## Verification questions

1. Does each test assert observable behavior at a stable seam — not a private method, call order, or internal shape?
2. Is the suite shaped like a pyramid — many fast unit tests, fewer integration tests, few end-to-end — with a speed budget per tier?
3. Are integration tests run against the *real* dependency in a container, not a mock of it?
4. Before any untested code was changed, was its current behavior pinned with characterization tests?
5. Is every test deterministic — time, randomness, and ordering controlled — with zero tolerated flakes?
6. Is a fake preferred over a mock wherever a working in-memory implementation is feasible?
7. Is model-output quality routed to `ai-evaluation-harness` rather than asserted with brittle string equality in a test?
8. Is test code DAMP — readable top-to-bottom with arrange/act/assert visible — and does each failure message say expected versus actual?
9. Does each test name state a behavior and its condition, so a CI failure is diagnosable without opening the test body?

## What to read next

Tier-2 references: `references/unit-tests-and-table-driven-go.md` · `references/integration-tests-and-real-dependencies.md` · `references/characterization-tests.md` · `references/test-doubles-and-fixtures.md` · `references/suite-health-and-ci-gating.md`.

Related skills: `ai-evaluation-harness` (scores non-deterministic model output — the eval counterpart to this skill's tests) · `python-service-engineering` / `mcp-go-server-building` (the service code this skill tests) · `data-access-engineering` (owns the disposable-engine wiring this skill's integration tests run against) · `codebase-extraction-engineering` (the extractor pinned in the worked example).
