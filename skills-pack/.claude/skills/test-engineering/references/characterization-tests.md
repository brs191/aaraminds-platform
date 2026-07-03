# Characterization Tests

This reference covers the technique for changing code that has no tests: pin its *current* behavior first, then change it under the net. Also called golden-master or approval testing.

## Why you cannot just "add tests"

When code is undocumented and untested, you do not know what it is *supposed* to do — only what it *does*. Writing tests from your assumption of intent risks encoding a bug as the spec, or missing a behavior something downstream secretly relies on. A characterization test sidesteps the question of intent entirely: it captures what the code *currently does*, exactly, bugs included, and turns that into a regression net. The goal is not correctness — it is *change detection*.

## How to write one

Pick a representative input, run the code, capture the entire output, and assert future runs produce the same output. The captured output is the "golden" or "approved" file, committed to the repo. The test fails on any behavior change; you then judge each diff: intended (re-approve — update the golden) or a regression (fix the code).

```python
def test_extractor_characterization(golden):
    result = run_extractor("fixtures/sample-repo")
    golden.assert_matches("sample-repo.graph.json", serialize(result))
```

The mechanics: a deterministic serialization of the output (sorted keys, stable ordering, no timestamps or absolute paths), a golden file per fixture, and an "approve" mode that rewrites the golden when a change is intended. Go: capture to a `testdata/` file gated behind a `-update` flag. Python: `pytest --snapshot-update`, or a small helper.

## Choosing fixtures

The net is only as good as its inputs. Pick fixtures that exercise the breadth of behavior — for the CIF extractor: a plain repo, a multi-module build, one with generated code, one with the framework idioms that matter. Each fixture gets its own golden. A handful of well-chosen fixtures catches more than one large one, because a diff localizes to the fixture.

## Use it as scaffolding, then move on

Characterization tests are a means, not the destination. The sequence: pin behavior with characterization tests → refactor or change safely under them → once the new structure is stable, write *real* unit and integration tests against its proper seams → retire the characterization tests, or demote them to a coarse smoke test. They are verbose, coupled to whole outputs, and noisy on intended change — fine as a temporary safety net, wrong as the permanent suite. Do not let "we have golden tests" stand in for a real suite forever.

## The discipline trap

The risk is approving diffs without reading them. A characterization test only protects you if every golden change is examined and consciously judged. A workflow that re-approves all goldens to make CI green has deleted the net while keeping the file. Review golden diffs in code review like any other change.
